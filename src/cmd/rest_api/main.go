// This implements a REST API that allows clients to play the game. It is intended for use by a
// single client who plays all players at once and does not make extraneous requests for game state
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

// writeError formats the error as JSON and writes it to the response with the given error code.
func writeError(resp http.ResponseWriter, statusCode int, err error) {
	resp.Header().Set("Content-Type", "application/jsonl; charset=utf-8")
	resp.WriteHeader(statusCode)
	json.NewEncoder(resp).Encode(map[string]string{"error": err.Error()})
}

type game struct {
	game            *engine.GameState
	possibleActions chan []engine.PlayerAction // possible player actions from GameState
	nextAction      chan engine.PlayerAction   // player action to send to GameState
	logBuf          bytes.Buffer               // The json log for the game gets written here
	id              string                     // A unique ID for this game
}

// newGame creates a GameState and calls Run() in a goroutine
func newGame(id string, players int, gameParams params.Params) (*game, error) {
	g := &game{
		id:              id,
		possibleActions: make(chan []engine.PlayerAction),
		nextAction:      make(chan engine.PlayerAction),
	}
	gs, err := engine.NewGame(
		players,
		gameParams,
		eventlog.NewJsonLogger(&g.logBuf),
		func(pas []engine.PlayerAction) engine.PlayerAction {
			g.possibleActions <- pas
			return <-g.nextAction
		},
		func() {
			close(g.possibleActions)
		},
	)
	if err != nil {
		return nil, err
	}
	g.game = gs
	go g.game.Run()
	return g, nil
}

type stateResponse struct {
	Status            string
	Reason            string
	Round             int
	EmissionsCounter  int
	Players           []engine.PlayerState
	LastRoundSnapshot engine.Snapshot
	TakeoverPool      assets.AssetMix
}

type gameResponse struct {
	ID              string
	Game            stateResponse
	PossibleActions []engine.PlayerAction
}

// Returns the client-observable game state. Blocks on receive from possibleActions
func (g *game) getState() gameResponse {
	actions := <-g.possibleActions
	return gameResponse{
		ID: g.id,
		Game: stateResponse{
			Status:            g.game.Status.String(),
			Reason:            g.game.Reason.String(),
			Round:             g.game.Round,
			EmissionsCounter:  g.game.CarbonEmissions,
			Players:           g.game.Players,
			LastRoundSnapshot: g.game.LastSnapshot,
			TakeoverPool:      g.game.TakeoverPool,
		},
		PossibleActions: actions,
	}
}

// writeStateResponse writes the observable game state to the response. It blocks on receive from possibleActions
func (g *game) writeStateResponse(resp http.ResponseWriter) {
	s := g.getState()
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(resp).Encode(s); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(err.Error()))
	}
}

// handleAction handles requests with the selected player action and returns the observable game state
func (g *game) handleAction(resp http.ResponseWriter, req *http.Request) {
	// Write the PlayerAction encoded in the request to the state machine
	if g.game.Status == core.GameStatusOngoing {
		var pa engine.PlayerAction
		d := json.NewDecoder(req.Body)
		err := d.Decode(&pa)
		if err != nil {
			writeError(resp, http.StatusBadRequest, err)
			return
		}
		g.nextAction <- pa
	}

	// Write the resulting state to the response
	g.writeStateResponse(resp)
}

// writeLogToRequest writes the contents of the log to the response
func (g *game) writeLogToRequest(resp http.ResponseWriter) {
	resp.Header().Set("Content-Type", "application/jsonl; charset=utf-8")
	resp.Write(g.logBuf.Bytes())
}

// A server manages multiple games
type server struct {
	mu    sync.RWMutex
	games map[string]*game // Currently running games
	rng   rand.Source      // RNG used to create game IDs
}

func newServer() *server {
	return &server{
		games: make(map[string]*game),
		rng:   rand.NewSource(846254781), // Fixed RNG seed for game IDs
	}
}

// newGame handles createion of new game with the given number of players, and returns the state
func (s *server) newGame() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		var numPlayers int
		_, err := fmt.Sscanf(req.FormValue("numPlayers"), "%d", &numPlayers)
		if err != nil {
			writeError(resp, http.StatusBadRequest, fmt.Errorf("cannot read numPlayers: %w", err))
			return
		}
		var encodedID = make([]byte, 8)
		binary.BigEndian.PutUint64(encodedID, uint64(s.rng.Int63()))
		sid := base64.RawURLEncoding.EncodeToString(encodedID)
		// Starts the game running in a goroutine
		game, err := newGame(sid, numPlayers, params.Default)
		if err != nil {
			writeError(resp, http.StatusInternalServerError, fmt.Errorf("cannot create new game: %w", err))
			return
		}
		{
			s.mu.Lock()
			defer s.mu.Unlock()
			s.games[sid] = game
		}
		game.writeStateResponse(resp)
	}
}

// actionHandler posts the latest action to the game with the given ID, and returns its new state
func (s *server) actionHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sid := req.PathValue("id")
		if sid == "" {
			writeError(resp, http.StatusInternalServerError, fmt.Errorf(`cannot look up "id" in pattern %s`, req.Pattern))
			return
		}

		s.mu.RLock()
		defer s.mu.RUnlock()
		game, ok := s.games[sid]
		if !ok {
			writeError(resp, http.StatusNotFound, fmt.Errorf("no game with id %q", sid))
			return
		}
		game.handleAction(resp, req)
	}
}

// logHandler returns the log for the game with the given ID
func (s *server) logHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sid := req.PathValue("id")
		if sid == "" {
			writeError(resp, http.StatusInternalServerError, fmt.Errorf(`cannot look up "id" in pattern %s`, req.Pattern))
			return
		}

		s.mu.RLock()
		defer s.mu.RUnlock()
		game, ok := s.games[sid]
		if !ok {
			writeError(resp, http.StatusNotFound, fmt.Errorf("no game with id %q", sid))
			return
		}
		game.writeLogToRequest(resp)
	}
}

// deleteHandler deletes a game. It is a no-op if the game doesn't exist
func (s *server) deleteHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sid := req.PathValue("id")
		if sid == "" {
			writeError(resp, http.StatusInternalServerError, fmt.Errorf(`cannot look up "id" in pattern %s`, req.Pattern))
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.games, sid)
	}
}

// rootHandler returns the set of game IDs
func (s *server) rootHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		keys := make([]string, 0, len(s.games))
		for k := range s.games {
			keys = append(keys, k)
		}

		resp.Header().Set("Content-Type", "application/json")
		json.NewEncoder(resp).Encode(map[string][]string{"ids": keys})
	}
}

func (s *server) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /new", s.newGame())
	mux.HandleFunc("POST /g/{id}/action", s.actionHandler())
	mux.HandleFunc("GET /g/{id}/log", s.logHandler())
	mux.HandleFunc("DELETE /g/{id}", s.deleteHandler())
	mux.HandleFunc("GET /{$}", s.rootHandler())
	return mux
}

const (
	defaultNetAddr = "localhost:0"
)

// getListener returns a listener to use for the server, a cleanup function, or an error.
//
// It is the caller's responsibility to call listener.Close() and cleanup() if the function does not return an error.
// If multiple arguments are set, the first one in [socketPath, netAddr] is used.
//
// Args:
//   - netAddr: An IP:[port] address for using TCP. The port may be left empty to automatically choose one.
//   - socketPath: A path to create a unix socket. Any existing file at the path will be removed, and the socket file will be removed by cleanup().
func getListener(netAddr string, socketPath string) (listener net.Listener, cleanup func(), err error) {
	switch {
	case socketPath != "":
		os.Remove(socketPath)
		cleanup = func() {
			log.Printf("Removing socket: %s", socketPath)
			os.Remove(socketPath)
		}
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		log.Printf("Listening on unix socket %s", socketPath)
		return
	case netAddr != "":
		cleanup = func() {}
		listener, err = net.Listen("tcp", netAddr)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("Listening on %s", listener.Addr().String())
		return
	default:
		return nil, nil, fmt.Errorf("one of netAddr or socketPath must be set")
	}
}

func main() {
	var err error
	var netAddr string
	var socketPath string
	flag.StringVar(&netAddr, "addr", defaultNetAddr, "Address in host:port format. If the port is 0 or empty, an unused port will be selected.")
	flag.StringVar(&socketPath, "socket", "", "Path to create a unix socket at. Server will listen for connections on the UNIX socket.")
	flag.Parse()

	// Create listener for the server to accept connections on
	listener, cleanup, err := getListener(netAddr, socketPath)
	if err != nil {
		log.Fatalf("Could not set up listener: %s", err)
	}
	defer listener.Close()
	defer cleanup()

	// Handle connections on the listener, forward unexpected errors to errChan
	httpServer := http.Server{Handler: newServer().Mux()}
	errChan := make(chan error, 1)
	go func() {
		err := httpServer.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Need to trap Interrupt (Ctrl+C) and Terminate signals, otherwise deferred cleanup code doesn't run
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-sigChan:
		log.Printf("Received OS signal: %v. Starting graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	case err := <-errChan:
		log.Printf("Serve() exited with unexpected error: %v", err)
	}
}
