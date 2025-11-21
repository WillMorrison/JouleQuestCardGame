// This implements a REST API that allows clients to play the game. It is intended for use by a
// single client who plays all players at once and does not make extraneous requests for game state
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type game struct {
	game              *engine.GameState
	possibleActions   chan []engine.PlayerAction // possible player actions from GameState
	nextAction        chan engine.PlayerAction   // player action to send to GameState
	gotInitialRequest bool                       // Boolean set to true once the first request has been received
	logBuf            bytes.Buffer               // The json log for the game gets written here
}

// newGame creates a GameState and calls Run() in a goroutine
func newGame(players int, gameParams params.Params) (*game, error) {
	g := &game{
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
	Reason            string `json:",omitempty"`
	Round             int
	EmissionsCounter  int
	Players           []engine.PlayerState
	LastRoundSnapshot engine.Snapshot
	PossibleActions   []engine.PlayerAction
}

// Returns the client-observable game state. Blocks on receive from possibleActions
func (g *game) getState() stateResponse {
	actions := <-g.possibleActions
	status := g.game.Status
	var reason string
	if status == engine.GameStatusLoss {
		reason = g.game.Reason.String()
	}
	return stateResponse{
		Status:            g.game.Status.String(),
		Reason:            reason,
		Round:             g.game.Round,
		EmissionsCounter:  g.game.CarbonEmissions,
		Players:           g.game.Players,
		LastRoundSnapshot: g.game.LastSnapshot,
		PossibleActions:   actions,
	}
}

// stateHandler accepts POST requests with the selected player action and returns the observable game state
func (g *game) stateHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if !g.gotInitialRequest {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte("Must call init before POSTing actions\n"))
			return
		}

		// Write the PlayerAction encoded in the request to the state machine
		if g.game.Status == engine.GameStatusOngoing {
			var pa engine.PlayerAction
			d := json.NewDecoder(req.Body)
			err := d.Decode(&pa)
			if err != nil {
				resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
				resp.WriteHeader(http.StatusBadRequest)
				resp.Write([]byte(err.Error()))
				return
			}
			g.nextAction <- pa
		}

		// Respond with game state
		s := g.getState()
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(resp).Encode(s); err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(err.Error()))
		}
	}
}

// initHandler accepts a GET request returns the observable game state
func (g *game) initHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, _ *http.Request) {
		if g.gotInitialRequest {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte("Cannot call init more than once\n"))
			return
		}

		// Respond with game state
		s := g.getState()
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(resp).Encode(s); err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(err.Error()))
		}
		g.gotInitialRequest = true
	}
}

// logHandler responds to all requests with the contents of the log
func (g *game) logHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/jsonl; charset=utf-8")
		resp.Write(g.logBuf.Bytes())
	}
}

func (g *game) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /init", g.initHandler())
	mux.HandleFunc("POST /action", g.stateHandler())
	mux.HandleFunc("GET /log", g.logHandler())
	return mux
}

func main() {
	var listenAddr string
	var socketPath string
	var numPlayers int
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:", "TCP address to listen on")
	flag.StringVar(&socketPath, "socket", "", "Path to a unix socket to listen on")
	flag.IntVar(&numPlayers, "players", 4, "Number of players between 3 and 7")
	flag.Parse()

	g, err := newGame(numPlayers, params.Default)
	if err != nil {
		log.Fatal(err)
	}

	var listener net.Listener
	if socketPath == "" {
		listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Listening on %s", listener.Addr().String())
	} else {
		// Clean up old socket file if it exists. Ignore errors (e.g. if the file didn't exist)
		os.Remove(socketPath)
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Listening on unix:%s", listener.Addr().String())

		// Clean up on ctrl-C
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		go func(c chan os.Signal) {
			<-c
			fmt.Println("Caught signal: shutting down and removing socket")
			os.Remove(socketPath)
			os.Exit(0)
		}(sigc)
	}
	defer listener.Close()

	err = http.Serve(listener, g.Mux())
	if err != nil {
		log.Fatal(err)
	}
}
