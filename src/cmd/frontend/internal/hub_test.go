package hub

import (
	"cmp"
	"slices"
	"testing"
	"time"
)

// MustLookupClient returns the client with the given ID or calls t.Fatal()
func MustLookupClient(t testing.TB, h *Hub, clientID int64) Client {
	t.Helper()
	client, ok := h.LookupClient(clientID)
	if !ok {
		t.Fatalf("Couldn't lookup client %d", clientID)
	}
	return client
}

// MustLookupGame returns the game with the given ID or calls t.Fatal()
func MustLookupGame(t testing.TB, h *Hub, gameID int64) Game {
	t.Helper()
	game, ok := h.LookupGame(gameID)
	if !ok {
		t.Fatalf("Couldn't lookup game %d", gameID)
	}
	return game
}

func cmpNotifications(a, b Notification) int {
	if n := cmp.Compare(a.Event, b.Event); n != 0 {
		return n
	}
	return cmp.Compare(a.ForID, b.ForID)
}

type notificationCollector struct {
	done      chan struct{}
	ch        <-chan Notification
	got       []Notification
	wantCount int
}

// Starts a goroutine that collects notifications from the given Hub. Callers should call CheckExpectations before the test ends
func ExpectNotifications(t testing.TB, h *Hub, wantCount int) *notificationCollector {
	t.Helper()
	nid, nCh := h.GetNotifyChan()
	nc := notificationCollector{
		wantCount: wantCount,
		ch:        nCh,
		done:      make(chan struct{}),
	}
	going := make(chan struct{})
	go func() {
		defer close(nc.done)
		defer h.RemoveNotifyChan(nid)
		for {
			select {
			case <-t.Context().Done():
				return
			case n := <-nc.ch:
				nc.got = append(nc.got, n)
				if len(nc.got) == nc.wantCount {
					return
				}
			case <-going:
			}
		}
	}()
	going <- struct{}{} // ensure goroutine is running before continuing, to avoid losing notifications
	return &nc
}

func (nc *notificationCollector) CheckExpectations(t testing.TB, want []Notification) {
	t.Helper()
	if len(want) != nc.wantCount {
		t.Errorf("Mismatch between len(want)=%d and initial count %d", len(want), nc.wantCount)
		return
	}
	select {
	case <-nc.done:
		// Wait for goroutine to process expected number of notifications
	case <-t.Context().Done():
		t.Fatalf("Did not receive all Notifications before test ended: got %v, want %d", nc.got, nc.wantCount)
	}

	// Order independent check
	slices.SortFunc(nc.got, cmpNotifications)
	slices.SortFunc(want, cmpNotifications)
	if !slices.Equal(nc.got, want) {
		t.Errorf("Got %+v, want %+v", nc.got, want)
	}
	if n, ok := <-nc.ch; ok {
		t.Errorf("Notification(s) left in channel after removal: %s", n)
	}
}

func Test_Hub_ClientGameLifecycle(t *testing.T) {
	h := NewHub()
	var clientFoo, clientBar Client
	var game Game

	if !t.Run("Connect two clients", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 4)
		clientID1 := h.NewClient("Foo")
		t.Logf("Client %d is Foo", clientID1)
		clientID2 := h.NewClient("Bar")
		t.Logf("Client %d is Bar", clientID2)

		wantFoo := Client{ID: clientID1, Name: "Foo", State: ClientStateUnassociated, GameID: noID, index: -1, disconnectedSince: time.Time{}}
		clientFoo = MustLookupClient(t, h, clientID1)
		if clientFoo != wantFoo {
			t.Errorf("LookupClient(%d)=%+v, want=%+v", clientID1, clientFoo, wantFoo)
		}
		wantBar := Client{ID: clientID2, Name: "Bar", State: ClientStateUnassociated, GameID: noID, index: -1, disconnectedSince: time.Time{}}
		clientBar = MustLookupClient(t, h, clientID2)
		if clientBar != wantBar {
			t.Errorf("LookupClient(%d)=%+v, want=%+v", clientID2, clientBar, wantBar)
		}

		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
			{Event: EventLobbyStateChange, ForID: noID},
			{Event: EventClientStateChange, ForID: clientID1},
			{Event: EventClientStateChange, ForID: clientID2},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo creates a new game in lobby", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		gameID, ok := h.AssociateClientToNewGame(clientFoo.ID)
		if !ok {
			t.Fatalf("Couldn't create a new game")
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateAssociated {
			t.Errorf("Client %d state is %s, want %s", clientFoo.ID, clientFoo.State, ClientStateAssociated)
		}
		if clientFoo.GameID != gameID {
			t.Errorf("Client %d was associated with game %d, want game %d", clientFoo.ID, clientFoo.GameID, gameID)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStateUnassociated {
			t.Errorf("Client %d state is %s, want %s", clientBar.ID, clientBar.State, ClientStateUnassociated)
		}
		game = MustLookupGame(t, h, gameID)
		if game.State != GameStateJoinable {
			t.Errorf("Game %d state is %s, want %s", game.ID, game.State, GameStateJoinable)
		}
		lobbyGames := h.LookupLobbyGames()
		if !slices.Equal(lobbyGames, []int64{game.ID}) {
			t.Errorf("LookupLobbyGames=%v, want %v", lobbyGames, []int64{game.ID})
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo readies but game doesn't start due to not enough players", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		if !h.ReadyClient(clientFoo.ID) {
			t.Errorf("Couldn't mark client %d as ready", clientFoo.ID)
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateJoinable {
			t.Errorf("Game state was %s, want %s", game.State, GameStateJoinable)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateReady {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateReady)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo unreadies", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		if !h.UnreadyClient(clientFoo.ID) {
			t.Errorf("Couldn't mark client %d as unready", clientFoo.ID)
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateJoinable {
			t.Errorf("Game state was %s, want %s", game.State, GameStateJoinable)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateAssociated {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateAssociated)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo unassociates and game is deleted", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		if !h.UnassociateClient(clientFoo.ID) {
			t.Errorf("Couldn't unassociate client %d", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateUnassociated {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateUnassociated)
		}
		if clientFoo.GameID != noID {
			t.Errorf("Client %d has game ID %d, want %d", clientFoo.ID, clientFoo.GameID, noID)
		}
		if _, exists := h.LookupGame(game.ID); exists {
			t.Errorf("Game %d was not deleted due to having no players", game.ID)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo creates another new game in lobby", func(t *testing.T) {
		gameID, ok := h.AssociateClientToNewGame(clientFoo.ID)
		if !ok {
			t.Fatalf("Couldn't create a new game")
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		game = MustLookupGame(t, h, gameID)
		if game.State != GameStateJoinable {
			t.Errorf("Game %d state is %s, want %s", game.ID, game.State, GameStateJoinable)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("Bar joins the game Foo created", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		if !h.AssociateClientToGame(clientBar.ID, game.ID) {
			t.Fatalf("Couldn't associate client %d to game %d", clientBar.ID, game.ID)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStateAssociated {
			t.Errorf("Client %d state is %s, want %s", clientBar.ID, clientBar.State, ClientStateAssociated)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Bar marks as ready but game doesn't start because Foo isn't ready", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 1)
		if !h.ReadyClient(clientBar.ID) {
			t.Errorf("Couldn't mark client %d as ready", clientBar.ID)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStateReady {
			t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientBar.State, ClientStateReady)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo marks as ready and game begins", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ReadyClient(clientFoo.ID) {
			t.Errorf("Couldn't mark client %d as ready", clientFoo.ID)
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateInProgress {
			t.Errorf("Game state was %s, want %s", game.State, GameStateInProgress)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStatePlaying {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStatePlaying)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStatePlaying {
			t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientFoo.State, ClientStatePlaying)
		}
		lobbyGames := h.LookupLobbyGames()
		if !slices.Equal(lobbyGames, []int64{}) {
			t.Errorf("LookupLobbyGames=%v, want %v", lobbyGames, []int64{})
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
			{Event: EventGameStateChange, ForID: game.ID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo disconnects and game is blocked", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ClientDisconnected(clientFoo.ID) {
			t.Errorf("Client %d could not disconnect", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateInactive {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateInactive)
		}
		if clientFoo.GameID != game.ID {
			t.Errorf("Client %d GameID was cleared on disconnection", clientFoo.ID)
		}
		if clientFoo.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time set", clientFoo.ID)
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateBlocked {
			t.Errorf("Game state was %s, want %s", game.State, GameStateBlocked)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientFoo.ID},
			{Event: EventGameStateChange, ForID: game.ID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo reconnects and game is restored to in progress", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ClientReconnected(clientFoo.ID) {
			t.Fatalf("Client %d was unable to reconnect", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStatePlaying {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStatePlaying)
		}
		if !clientFoo.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time unset, was %s", clientFoo.ID, clientFoo.disconnectedSince.Format(time.RFC3339))
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateInProgress {
			t.Errorf("Game state was %s, want %s", game.State, GameStateInProgress)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientFoo.ID},
			{Event: EventGameStateChange, ForID: game.ID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Bar exits to lobby and game is blocked", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ExitOngoingGame(clientBar.ID) {
			t.Errorf("client %d couldn't exit the game", clientBar.ID)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStateUnassociated {
			t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientFoo.State, ClientStateUnassociated)
		}
		if clientBar.GameID != noID {
			t.Errorf("Client %d has game ID %d, want %d", clientBar.ID, clientBar.GameID, noID)
		}
		game = MustLookupGame(t, h, game.ID)
		if game.State != GameStateBlocked {
			t.Errorf("Game state was %s, want %s", game.State, GameStateBlocked)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventLobbyStateChange, ForID: noID},
			{Event: EventGameStateChange, ForID: game.ID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo disconnects and game is deleted", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ClientDisconnected(clientFoo.ID) {
			t.Errorf("Client %d could not disconnect", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateInactive {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateInactive)
		}
		if clientFoo.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time set", clientFoo.ID)
		}
		if _, exists := h.LookupGame(game.ID); exists {
			t.Errorf("Game %d was not deleted due to having no players", game.ID)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientFoo.ID},
			{Event: EventGameStateChange, ForID: game.ID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo reconnects with same ID and is put into the lobby", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ClientReconnected(clientFoo.ID) {
			t.Fatalf("Client %d was unable to reconnect", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateUnassociated {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateUnassociated)
		}
		if clientFoo.GameID != noID {
			t.Errorf("Client %d has game ID %d, want %d", clientFoo.ID, clientFoo.GameID, noID)
		}
		if !clientFoo.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time unset, was %s", clientFoo.ID, clientFoo.disconnectedSince.Format(time.RFC3339))
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientFoo.ID},
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Bar disconnects from the lobby while unassociated", func(t *testing.T) {
		nc := ExpectNotifications(t, h, 2)
		if !h.ClientDisconnected(clientBar.ID) {
			t.Fatalf("Client %d was unable to reconnect", clientFoo.ID)
		}
		clientBar = MustLookupClient(t, h, clientBar.ID)
		if clientBar.State != ClientStateInactive {
			t.Errorf("Client %d state was %s, want %s", clientBar.ID, clientBar.State, ClientStateInactive)
		}
		if clientBar.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time set", clientBar.ID)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientBar.ID},
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}

	if !t.Run("Foo creates a new game in lobby then disconnects", func(t *testing.T) {
		gameID, ok := h.AssociateClientToNewGame(clientFoo.ID)
		if !ok {
			t.Fatalf("Couldn't create a new game")
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateAssociated {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateAssociated)
		}
		game = MustLookupGame(t, h, gameID)
		if game.State != GameStateJoinable {
			t.Errorf("Game %d state is %s, want %s", game.ID, game.State, GameStateJoinable)
		}

		nc := ExpectNotifications(t, h, 2)
		if !h.ClientDisconnected(clientFoo.ID) {
			t.Errorf("Client %d could not disconnect", clientFoo.ID)
		}
		clientFoo = MustLookupClient(t, h, clientFoo.ID)
		if clientFoo.State != ClientStateInactive {
			t.Errorf("Client %d state was %s, want %s", clientFoo.ID, clientFoo.State, ClientStateInactive)
		}
		if clientFoo.GameID != noID {
			t.Errorf("Client %d has game ID %d, want %d", clientFoo.ID, clientFoo.GameID, noID)
		}
		if clientFoo.disconnectedSince.IsZero() {
			t.Errorf("Client %d should have disconnection time set", clientFoo.ID)
		}
		if _, exists := h.LookupGame(game.ID); exists {
			t.Errorf("Game %d was not deleted due to having no players", game.ID)
		}
		nc.CheckExpectations(t, []Notification{
			{Event: EventClientStateChange, ForID: clientFoo.ID},
			{Event: EventLobbyStateChange, ForID: noID},
		})
	}) {
		t.FailNow()
	}
}
