package hub

import (
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

const maxPlayers = 7
const minPlayers = 2
const noID int64 = -1

// encodeID encodes an integer ID as a URL-safe base64-encoded string.
func encodeID(id int64) string {
	var encodedID = make([]byte, 8)
	binary.BigEndian.PutUint64(encodedID, uint64(id))
	return base64.RawURLEncoding.EncodeToString(encodedID)
}

type ClientState int

//go:generate go tool stringer -type=ClientState -trimprefix=ClientState
const (
	ClientStateInactive     ClientState = iota // disconnected
	ClientStateUnassociated                    // connected but not associated with a game
	ClientStateAssociated                      // connected and associated with a game
	ClientStateReady                           // associated and ready to play
	ClientStatePlaying                         // associated and playing
)

type GameState int

//go:generate go tool stringer -type=GameState -trimprefix=GameState
const (
	GameStateBlocked    GameState = iota // cannnot progress due to missing required players
	GameStateJoinable                    // joinable by players in the lobby
	GameStateFull                        // full and cannot be joined
	GameStateInProgress                  // in progress and cannot be joined
)

type Event int

const (
	EventGameStateChange Event = iota
	EventLobbyStateChange
	EventClientStateChange
)

type Client struct {
	ID                int64
	Name              string
	State             ClientState
	GameID            int64     // the ID of the game the client is associated with, if any.
	index             int       // the index of the client in the game, if any.
	disconnectedSince time.Time // the time the client disconnected, if any.
}

type Game struct {
	ID    int64
	State GameState
	game  *engine.ProceduralGameState // the game state, if the game is in progress.
}

type Notification struct {
	Event Event
	ForID int64 // the ID of the game the notification is for, if applicable.
}

func (n Notification) String() string {
	switch n.Event {
	case EventGameStateChange:
		return "game: " + encodeID(n.ForID)
	case EventLobbyStateChange:
		return "lobby"
	case EventClientStateChange:
		return "client: " + encodeID(n.ForID)
	}
	return "unknown"
}

// Hub holds the state of all clients and games. Methods are safe to call concurrently.
type Hub struct {
	mu sync.RWMutex

	games       map[int64]*Game
	clients     map[int64]*Client
	notifySinks map[int64]chan<- Notification // channels to notify when events occur
}

func NewHub() *Hub {
	return &Hub{
		games:       make(map[int64]*Game),
		clients:     make(map[int64]*Client),
		notifySinks: make(map[int64]chan<- Notification),
	}
}

// broadcast sends the notification to all channels. The lock should be held by the caller.
func (h *Hub) broadcast(n Notification) {
	for _, sink := range h.notifySinks {
		select {
		case sink <- n:
		default:
		}
	}
}

// getClientInState returns the client with the given ID and state, if it exists. The lock should be held by the caller.
func (h *Hub) getClientInState(id int64, states ...ClientState) (*Client, bool) {
	client, ok := h.clients[id]
	if !ok || !slices.Contains(states, client.State) {
		return nil, false
	}
	return client, true
}

// getGameInState returns the game with the given ID and state, if it exists. The lock should be held by the caller.
func (h *Hub) getGameInState(id int64, states ...GameState) (*Game, bool) {
	game, ok := h.games[id]
	if !ok || !slices.Contains(states, game.State) {
		return nil, false
	}
	return game, true
}

// clientsInGame returns the IDs of all clients in the game. The lock should be held by the caller.
func (h *Hub) clientsInGame(gameID int64) []int64 {
	var clients []int64
	for _, client := range h.clients {
		switch client.State {
		case ClientStateAssociated, ClientStateReady, ClientStatePlaying:
			if client.GameID == gameID {
				clients = append(clients, client.ID)
			}
		}
	}
	return clients
}

// startGameIfAllClientsReady starts the game if all clients are ready. Returns true if the game was started, false otherwise. The lock should be held by the caller.
func (h *Hub) startGameIfAllClientsReady(game *Game) bool {
	if game.State != GameStateJoinable && game.State != GameStateFull {
		return false
	}
	clientsInGame := make([]*Client, 0)
	for _, client := range h.clients {
		if client.GameID != game.ID {
			continue
		}
		if client.State != ClientStateReady {
			return false
		}
		clientsInGame = append(clientsInGame, client)
	}
	if len(clientsInGame) < minPlayers {
		return false
	}

	// Create a new game
	pg, err := engine.NewProceduralGame(len(clientsInGame), params.Default, eventlog.NullLogger{})
	if err != nil {
		return false
	}
	pg.SetRNGSeed(rand.Uint64())

	// Update the status for the game and clients
	game.State = GameStateInProgress
	game.game = pg
	for i, client := range clientsInGame {
		client.State = ClientStatePlaying
		client.GameID = game.ID
		client.index = i
	}
	return true
}

// maybeBlockGameOnExitingClient checks if the game should be blocked from progressing by the client exiting.
// Returns true if the game was blocked, false otherwise. The lock should be held by the caller.
func (h *Hub) maybeBlockGameOnExitingClient(client Client, game *Game) (blocked bool) {
	if client.State != ClientStatePlaying || client.GameID != game.ID {
		return false
	}
	if game.State == GameStateInProgress && game.game.Game().Status == core.GameStatusOngoing && game.game.Game().Players[client.index].Status == core.PlayerStatusActive {
		// Game is blocked from progressing by this player leaving
		game.State = GameStateBlocked
		blocked = true
	}
	if len(h.clientsInGame(game.ID)) == 1 {
		// last player left, delete the game
		delete(h.games, game.ID)
		return false
	}
	return
}

// maybeUnblockGameOnReturningClient checks if the game was unblocked from progressing by the client returning.
// Returns true if the game was unblocked, false otherwise. The lock should be held by the caller.
func (h *Hub) maybeUnblockGameOnReturningClient(client Client, game *Game) bool {
	if game.State != GameStateBlocked {
		return false
	}
	if game.game.Game().Status != core.GameStatusOngoing || game.game.Game().Players[client.index].Status != core.PlayerStatusActive {
		return false
	}
	// Get the set of clients connected in this game
	var clientsInGame []*Client
	for _, client := range h.clients {
		if client.GameID == game.ID && client.State == ClientStatePlaying {
			clientsInGame = append(clientsInGame, client)
		}
	}
	// Check that all active players are connected
	for i := range game.game.Game().Players {
		if game.game.Game().Players[i].Status == core.PlayerStatusActive {
			if !slices.ContainsFunc(clientsInGame, func(c *Client) bool { return c.index == int(i) }) {
				// Player is active but client is not in the game.
				return false
			}
		}
	}
	// All active players are connected, unblock the game
	game.State = GameStateInProgress
	return true
}

// DoGarbageCollection removes games with no clients and clients which have been disconnected for 30 minutes
func (h *Hub) DoGarbageCollection() {
	h.mu.Lock()
	defer h.mu.Unlock()
	gamesWithClients := make(map[int64]struct{})
	for clientID, client := range h.clients {
		switch client.State {
		case ClientStateAssociated, ClientStateReady, ClientStatePlaying:
			gamesWithClients[client.GameID] = struct{}{}
		case ClientStateInactive:
			if time.Since(client.disconnectedSince) > 30*time.Minute {
				delete(h.clients, clientID)
			}
		}
	}

	for gameID, _ := range h.games {
		if _, ok := gamesWithClients[gameID]; !ok {
			delete(h.games, gameID)
		}
	}
}

// GetNotifyChan returns a new channel to notify when events occur. The ID is used to identify the channel for later removal.
func (h *Hub) GetNotifyChan() (int64, <-chan Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := rand.Int63()
	sink := make(chan Notification)
	h.notifySinks[id] = sink
	return id, sink
}

// RemoveNotifyChan removes the channel identified by the given ID, if it exists.
func (h *Hub) RemoveNotifyChan(id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if nc, ok := h.notifySinks[id]; ok {
		close(nc)
	}
	delete(h.notifySinks, id)
}

// NewClient creates a new client with the given name and returns its ID. The client is added to the lobby.
// Intended to be called for clients that do not provide a session ID.
func (h *Hub) NewClient(name string) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	client := &Client{
		ID:     rand.Int63(),
		Name:   name,
		State:  ClientStateUnassociated,
		GameID: -1,
		index:  -1,
	}
	h.clients[client.ID] = client
	h.broadcast(Notification{Event: EventClientStateChange, ForID: client.ID})
	h.broadcast(Notification{Event: EventLobbyStateChange})
	return client.ID
}

// AssociateClientToNewGame associates a client with a new game in the lobby. Returns true if the client was successfully associated, false otherwise.
func (h *Hub) AssociateClientToNewGame(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	client, ok := h.getClientInState(clientID, ClientStateUnassociated)
	if !ok {
		return false
	}

	gameID := rand.Int63()
	game := &Game{
		ID:    gameID,
		State: GameStateJoinable,
	}
	h.games[gameID] = game

	client.State = ClientStateAssociated
	client.GameID = gameID
	client.index = -1

	h.broadcast(Notification{Event: EventLobbyStateChange})
	return true
}

// AssociateClientToGame associates a client with an existing game. Returns true if the client was successfully associated, false otherwise.
func (h *Hub) AssociateClientToGame(clientID int64, gameID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	client, ok := h.getClientInState(clientID, ClientStateUnassociated)
	if !ok {
		return false
	}
	game, ok := h.getGameInState(gameID, GameStateJoinable)
	if !ok {
		return false
	}

	client.GameID = gameID
	client.index = -1
	client.State = ClientStateAssociated

	if len(h.clientsInGame(gameID)) == maxPlayers {
		// Stop more players from joining this game
		game.State = GameStateFull
	}

	h.broadcast(Notification{Event: EventLobbyStateChange})
	return true
}

// UnassociateClientFromGame unassociates a client from a game. Returns true if the client was successfully unassociated, false otherwise.
func (h *Hub) UnassociateClient(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.getClientInState(clientID, ClientStateAssociated, ClientStateReady)
	if !ok {
		return false
	}
	game, ok := h.getGameInState(client.GameID, GameStateJoinable, GameStateFull)
	if !ok {
		return false
	}

	// Update
	client.State = ClientStateUnassociated
	client.GameID = -1
	client.index = -1

	if numLeft := len(h.clientsInGame(game.ID)); numLeft == 0 {
		// last clients left, delete the game
		delete(h.games, game.ID)
	} else {
		if game.State == GameStateFull && numLeft < maxPlayers {
			// still clients waiting, make the game joinable again
			game.State = GameStateJoinable
		}
		if h.startGameIfAllClientsReady(game) {
			// All remaining clients were ready, start the game
			h.broadcast(Notification{Event: EventGameStateChange, ForID: game.ID})
		}
	}

	h.broadcast(Notification{Event: EventLobbyStateChange})
	return true
}

// ReadyClient makes a client ready to play. Returns true if the client was successfully made ready, false otherwise.
func (h *Hub) ReadyClient(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.getClientInState(clientID, ClientStateAssociated)
	if !ok {
		return false
	}
	game, ok := h.getGameInState(client.GameID, GameStateJoinable, GameStateFull)
	if !ok {
		return false
	}

	// Update
	client.State = ClientStateReady
	if h.startGameIfAllClientsReady(game) {
		h.broadcast(Notification{Event: EventGameStateChange, ForID: game.ID})
	}
	h.broadcast(Notification{Event: EventLobbyStateChange})
	return true
}

// UnreadyClient makes a client unready to play. Returns true if the client was successfully made unready, false otherwise.
func (h *Hub) UnreadyClient(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.getClientInState(clientID, ClientStateReady)
	if !ok {
		return false
	}

	// Update
	client.State = ClientStateAssociated
	h.broadcast(Notification{Event: EventLobbyStateChange})
	return true
}

// ExitOngoingGame removes a client from an ongoing game and returns them to the lobby. Returns true if the client was successfully exited, false otherwise.
func (h *Hub) ExitOngoingGame(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.getClientInState(clientID, ClientStatePlaying)
	if !ok {
		return false
	}
	game, ok := h.getGameInState(client.GameID, GameStateInProgress, GameStateBlocked)
	if !ok {
		return false
	}

	// Update
	h.maybeBlockGameOnExitingClient(*client, game)
	h.broadcast(Notification{Event: EventGameStateChange, ForID: game.ID})

	client.State = ClientStateUnassociated
	client.GameID = -1
	client.index = -1
	h.broadcast(Notification{Event: EventLobbyStateChange})

	return true
}

// ClientDisconnected should be called whenever a client disconnects
func (h *Hub) ClientDisconnected(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.clients[clientID]
	if !ok {
		return false
	}
	switch client.State {
	case ClientStateUnassociated, ClientStateAssociated, ClientStateReady:
		// Update
		client.GameID = -1
		client.index = -1
		h.broadcast(Notification{Event: EventLobbyStateChange})
	case ClientStatePlaying:
		game, ok := h.getGameInState(client.GameID, GameStateInProgress, GameStateBlocked)
		if !ok {
			return false
		}
		// Update
		h.maybeBlockGameOnExitingClient(*client, game)
		h.broadcast(Notification{Event: EventGameStateChange, ForID: game.ID})
		// Leave client in the game in case they reconnect
	}

	client.State = ClientStateInactive
	client.disconnectedSince = time.Now()
	h.broadcast(Notification{Event: EventClientStateChange, ForID: clientID})

	return true
}

// ClientReconnected should be called when a client reconnects and provides its ID.
// Returns whether the client ID is still valid. If false, call NewClient()
func (h *Hub) ClientReconnected(clientID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.clients[clientID]
	if !ok {
		return false
	}

	// Update
	h.broadcast(Notification{Event: EventClientStateChange, ForID: client.ID})
	game, ok := h.getGameInState(client.GameID, GameStateInProgress, GameStateBlocked)
	if !ok {
		client.State = ClientStateUnassociated
		client.GameID = noID
		client.index = -1
		h.broadcast(Notification{Event: EventLobbyStateChange})
		return true
	}
	client.State = ClientStatePlaying
	h.maybeUnblockGameOnReturningClient(*client, game)
	h.broadcast(Notification{Event: EventGameStateChange, ForID: client.GameID})
	return true
}

// LookupGameStateForClient returns the game state that the client is currently playing, or nil if no such game exists.
func (h *Hub) LookupGameStateForClient(clientID int64) *engine.ProceduralGameState {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Lookup and Validate
	client, ok := h.getClientInState(clientID, ClientStatePlaying)
	if !ok {
		return nil
	}
	game, ok := h.getGameInState(client.GameID, GameStateInProgress, GameStateBlocked)
	if !ok {
		return nil
	}

	return game.game
}

// LookupClient returns the client with the given ID, and whether that client was found.
func (h *Hub) LookupClient(clientID int64) (Client, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client, ok := h.clients[clientID]
	if !ok {
		return Client{}, false
	}
	return *client, true
}

// LookupGame returns the game with the given ID, and whether that game was found.
func (h *Hub) LookupGame(gameID int64) (Game, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	game, ok := h.games[gameID]
	if !ok {
		return Game{}, false
	}
	return *game, true

}

// LookupLobbyGames returns the set of IDs of games which are currently in the lobby
func (h *Hub) LookupLobbyGames() []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var gameIDs []int64
	for gameID, game := range h.games {
		if game.State == GameStateJoinable || game.State == GameStateFull {
			gameIDs = append(gameIDs, gameID)
		}
	}
	return gameIDs
}
