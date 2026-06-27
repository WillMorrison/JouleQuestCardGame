package hub

import (
	"testing"
	"time"
)

func Test_Hub_NewClient(t *testing.T) {
	h := NewHub()

	clientID := h.NewClient("Foo")

	client, ok := h.LookupClient(clientID)
	if !ok {
		t.Fatalf("Couldn't lookup new client %d", clientID)
	}
	want := Client{ID: clientID, Name: "Foo", State: ClientStateUnassociated, GameID: noID, index: -1, disconnectedSince: time.Time{}}
	if client != want {
		t.Errorf("LookupClient(%d)=%+v, want=%+v", clientID, client, want)
	}
}

func Test_Hub_ClientGameLifecycle(t *testing.T) {
	h := NewHub()

	// Foo and Bar connect
	clientID1 := h.NewClient("Foo")
	t.Logf("Client %d is Foo", clientID1)
	clientID2 := h.NewClient("Bar")
	t.Logf("Client %d is Bar", clientID2)

	// Foo creates a new game in the lobby
	if !h.AssociateClientToNewGame(clientID1) {
		t.Fatalf("Couldn't create a new game")
	}
	clientFoo, _ := h.LookupClient(clientID1)
	if clientFoo.State != ClientStateAssociated {
		t.Errorf("Client %d state is %s, want %s", clientFoo.ID, clientFoo.State, ClientStateAssociated)
	}
	clientBar, _ := h.LookupClient(clientID2)
	if clientBar.State != ClientStateUnassociated {
		t.Errorf("Client %d state is %s, want %s", clientBar.ID, clientBar.State, ClientStateUnassociated)
	}
	game, ok := h.LookupGame(clientFoo.GameID)
	if !ok {
		t.Fatalf("Couldn't lookup game %d", clientFoo.GameID)
	}
	if game.State != GameStateJoinable {
		t.Errorf("Game %d state is %s, want %s", game.ID, game.State, GameStateJoinable)
	}
	lobbyGames := h.LookupLobbyGames()
	if len(lobbyGames) != 1 || lobbyGames[0] != game.ID {
		t.Errorf("LookupLobbyGames=%v, want []int64{%d}", lobbyGames, game.ID)
	}

	// Bar joins the game Foo created
	if !h.AssociateClientToGame(clientBar.ID, game.ID) {
		t.Fatalf("Couldn't associate client %d to game %d", clientBar.ID, game.ID)
	}
	clientBar, _ = h.LookupClient(clientBar.ID)
	if clientBar.State != ClientStateAssociated {
		t.Errorf("Client %d state is %s, want %s", clientBar.ID, clientBar.State, ClientStateAssociated)
	}

	// Bar marks as ready, game doesn't start
	if !h.ReadyClient(clientBar.ID) {
		t.Errorf("Couldn't mark client %d as ready", clientBar.ID)
	}
	clientBar, _ = h.LookupClient(clientBar.ID)
	if clientBar.State != ClientStateReady {
		t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientBar.State, ClientStateReady)
	}

	// Foo marks as ready, game begins
	if !h.ReadyClient(clientFoo.ID) {
		t.Errorf("Couldn't mark client %d as ready", clientFoo.ID)
	}
	game, _ = h.LookupGame(game.ID)
	if game.State != GameStateInProgress {
		t.Errorf("Game state was %s, want %s", game.State, GameStateInProgress)
	}
	clientFoo, _ = h.LookupClient(clientFoo.ID)
	if clientFoo.State != ClientStatePlaying {
		t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStatePlaying)
	}
	clientBar, _ = h.LookupClient(clientBar.ID)
	if clientBar.State != ClientStatePlaying {
		t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientFoo.State, ClientStatePlaying)
	}

	// Foo disconnects, game is blocked
	if !h.ClientDisconnected(clientFoo.ID) {
		t.Errorf("Client %d could not disconnect", clientFoo.ID)
	}
	clientFoo, _ = h.LookupClient(clientFoo.ID)
	if clientFoo.State != ClientStateInactive {
		t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateInactive)
	}
	if clientFoo.GameID != game.ID {
		t.Errorf("Client %d GameID was cleared on disconnection", clientFoo.ID)
	}
	game, _ = h.LookupGame(game.ID)
	if game.State != GameStateBlocked {
		t.Errorf("Game state was %s, want %s", game.State, GameStateBlocked)
	}

	// Foo reconnects, game is restored to in progress
	if !h.ClientReconnected(clientFoo.ID) {
		t.Fatalf("Client %d was unable to reconnect", clientFoo.ID)
	}
	clientFoo, _ = h.LookupClient(clientFoo.ID)
	if clientFoo.State != ClientStatePlaying {
		t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStatePlaying)
	}
	game, _ = h.LookupGame(game.ID)
	if game.State != GameStateInProgress {
		t.Errorf("Game state was %s, want %s", game.State, GameStateInProgress)
	}

	// Bar exits to lobby, game is blocked
	if !h.ExitOngoingGame(clientBar.ID) {
		t.Errorf("client %d couldn't exit the game", clientBar.ID)
	}
	clientBar, _ = h.LookupClient(clientBar.ID)
	if clientBar.State != ClientStateUnassociated {
		t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientFoo.State, ClientStateUnassociated)
	}
	if clientBar.GameID != noID {
		t.Errorf("Client %d has game ID %d, want %d", clientBar.ID, clientBar.GameID, noID)
	}
	game, _ = h.LookupGame(game.ID)
	if game.State != GameStateBlocked {
		t.Errorf("Game state was %s, want %s", game.State, GameStateBlocked)
	}

	// Foo disconnects, game is deleted due to no players remaining
	if !h.ClientDisconnected(clientFoo.ID) {
		t.Errorf("Client %d could not disconnect", clientFoo.ID)
	}
	if _, exists := h.LookupGame(game.ID); exists {
		t.Errorf("Game %d was not deleted due to having no players", game.ID)
	}

	// Foo reconnects, keeps ID but is now unassociated
	if !h.ClientReconnected(clientFoo.ID) {
		t.Fatalf("Client %d was unable to reconnect", clientFoo.ID)
	}
	clientFoo, _ = h.LookupClient(clientFoo.ID)
	if clientFoo.State != ClientStateUnassociated {
		t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateUnassociated)
	}
	if clientFoo.GameID != noID {
		t.Errorf("Client %d has game ID %d, want %d", clientFoo.ID, clientFoo.GameID, noID)
	}
}
