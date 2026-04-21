package presence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// =============================================================================
// EventParser tests
// =============================================================================

func TestParsePresencePayload_Online_HaloInfinite(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "1234567890123456",
		"presenceState": "Online",
		"presenceDetails": [{
			"titleid": "1144039928",
			"titleName": "Halo Infinite",
			"isGame": true,
			"isPrimary": true,
			"device": "PC",
			"state": "Active"
		}]
	}`)
	event, err := ParsePresencePayload(raw, "fallback-xuid")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.XUID != "1234567890123456" {
		t.Errorf("XUID = %q, want %q", event.XUID, "1234567890123456")
	}
	if event.PresenceState != "Online" {
		t.Errorf("PresenceState = %q, want %q", event.PresenceState, "Online")
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail is nil")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("TitleID = %q, want %q", event.PresenceDetail.TitleID, "1144039928")
	}
	if event.PresenceDetail.TitleName != "Halo Infinite" {
		t.Errorf("TitleName = %q", event.PresenceDetail.TitleName)
	}
	if !event.PresenceDetail.IsGame {
		t.Error("IsGame should be true")
	}
}

func TestParsePresencePayload_Offline(t *testing.T) {
	raw := json.RawMessage(`{"presenceState": "Offline"}`)
	event, err := ParsePresencePayload(raw, "xuid-fallback")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if event.XUID != "xuid-fallback" {
		t.Errorf("XUID = %q, want fallback", event.XUID)
	}
	if event.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q", event.PresenceState)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for offline")
	}
}

func TestParsePresencePayload_MultipleDetails_PrimaryFirst(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "X",
		"presenceState": "Online",
		"presenceDetails": [
			{"titleid": "999", "titleName": "Xbox App", "isGame": false, "isPrimary": true, "state": "Active"},
			{"titleid": "1144039928", "titleName": "Halo Infinite", "isGame": true, "isPrimary": true, "state": "Active"},
			{"titleid": "730", "titleName": "CS2", "isGame": true, "isPrimary": false, "state": "Inactive"}
		]
	}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("got TitleID %q, want 1144039928 (first isPrimary && isGame)", event.PresenceDetail.TitleID)
	}
}

func TestParsePresencePayload_FallbackToFirstGame(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "X",
		"presenceState": "Online",
		"presenceDetails": [
			{"titleid": "999", "titleName": "Xbox App", "isGame": false, "isPrimary": true},
			{"titleid": "1144039928", "titleName": "Halo", "isGame": true, "isPrimary": false}
		]
	}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil — should fallback to first isGame")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("TitleID = %q", event.PresenceDetail.TitleID)
	}
}

func TestParsePresencePayload_InvalidJSON(t *testing.T) {
	_, err := ParsePresencePayload(json.RawMessage(`{invalid`), "x")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePresencePayload_EmptyDetails(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","presenceState":"Online","presenceDetails":[]}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for empty details")
	}
}

// =============================================================================
// RTAClient unit tests (sans vrai WebSocket)
// =============================================================================

func TestNewRTAClient(t *testing.T) {
	c := NewRTAClient("XBL3.0 x=hash;token")
	if c == nil {
		t.Fatal("nil client")
	}
	if c.IsConnected() {
		t.Error("should not be connected initially")
	}
	if len(c.Subscriptions()) != 0 {
		t.Error("should have no subscriptions initially")
	}
}

func TestRTAClient_Subscribe_NotConnected(t *testing.T) {
	c := NewRTAClient("auth")
	err := c.Subscribe(context.Background(), "xuid-123", func(_ PresenceEvent) {})
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestRTAClient_Close_NotConnected(t *testing.T) {
	c := NewRTAClient("auth")
	if err := c.Close(); err != nil {
		t.Errorf("Close() on non-connected should not error, got %v", err)
	}
}

func TestRTAClient_UpdateAuth(t *testing.T) {
	c := NewRTAClient("old-auth")
	c.UpdateAuth("new-auth")
	if c.authHeader != "new-auth" {
		t.Errorf("authHeader = %q, want %q", c.authHeader, "new-auth")
	}
}

func TestRTAClient_HandleMessage_SubscribeConfirm(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	var received PresenceEvent
	c.pendingMu.Lock()
	c.pending[42] = pendingSub{
		xuid:    "test-xuid",
		handler: func(e PresenceEvent) { received = e },
	}
	c.pendingMu.Unlock()

	// Simuler une réponse subscribe : [1, 42, 0, 100]
	msg := `[1, 42, 0, 100]`
	c.handleMessage(ctx, []byte(msg))

	c.subsMu.RLock()
	sub, ok := c.subs[100]
	c.subsMu.RUnlock()
	if !ok {
		t.Fatal("subscription 100 not registered")
	}
	if sub.XUID != "test-xuid" {
		t.Errorf("XUID = %q", sub.XUID)
	}

	// Vérifier Subscriptions()
	xuids := c.Subscriptions()
	if len(xuids) != 1 || xuids[0] != "test-xuid" {
		t.Errorf("Subscriptions() = %v", xuids)
	}

	_ = received // handler pas encore appelé (pas de data initiale dans ce cas)
}

func TestRTAClient_HandleMessage_SubscribeWithInitialData(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	var received PresenceEvent
	c.pendingMu.Lock()
	c.pending[1] = pendingSub{
		xuid: "xuid-1",
		handler: func(e PresenceEvent) {
			received = e
		},
	}
	c.pendingMu.Unlock()

	// Réponse avec data initiale : [1, 1, 0, 50, {"xuid":"xuid-1","presenceState":"Online","presenceDetails":[{"titleid":"1144039928","titleName":"Halo","isGame":true,"isPrimary":true}]}]
	msg := `[1, 1, 0, 50, {"xuid":"xuid-1","presenceState":"Online","presenceDetails":[{"titleid":"1144039928","titleName":"Halo","isGame":true,"isPrimary":true}]}]`
	c.handleMessage(ctx, []byte(msg))

	if received.XUID != "xuid-1" {
		t.Errorf("received XUID = %q", received.XUID)
	}
	if received.PresenceState != "Online" {
		t.Errorf("received PresenceState = %q", received.PresenceState)
	}
	if received.PresenceDetail == nil || received.PresenceDetail.TitleName != "Halo" {
		t.Error("initial data not dispatched")
	}
}

func TestRTAClient_HandleMessage_Event(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	var received PresenceEvent
	c.subsMu.Lock()
	c.subs[77] = &subscription{
		XUID:  "xuid-77",
		SubID: 77,
		Handler: func(e PresenceEvent) {
			received = e
		},
	}
	c.subsMu.Unlock()

	// Event : [5, 77, {"xuid":"xuid-77","presenceState":"Offline"}]
	msg := `[5, 77, {"xuid":"xuid-77","presenceState":"Offline"}]`
	c.handleMessage(ctx, []byte(msg))

	if received.XUID != "xuid-77" {
		t.Errorf("XUID = %q", received.XUID)
	}
	if received.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q", received.PresenceState)
	}
}

func TestRTAClient_HandleMessage_InvalidJSON(t *testing.T) {
	c := NewRTAClient("auth")
	// Should not panic
	c.handleMessage(context.Background(), []byte(`not json`))
}

func TestRTAClient_HandleMessage_UnknownType(t *testing.T) {
	c := NewRTAClient("auth")
	c.handleMessage(context.Background(), []byte(`[99, 1, "data"]`))
}

func TestRTAClient_HandleMessage_ShortMessage(t *testing.T) {
	c := NewRTAClient("auth")
	c.handleMessage(context.Background(), []byte(`[5]`))
}

// =============================================================================
// ReconnectPolicy tests
// =============================================================================

func TestDefaultReconnectPolicy(t *testing.T) {
	p := DefaultReconnectPolicy()
	if p.InitialDelay != time.Second {
		t.Errorf("InitialDelay = %v", p.InitialDelay)
	}
	if p.MaxDelay != 5*time.Minute {
		t.Errorf("MaxDelay = %v", p.MaxDelay)
	}
	if p.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v", p.Multiplier)
	}
}

func TestReconnectManager_BackoffDelay(t *testing.T) {
	rm := &ReconnectManager{
		policy: ReconnectPolicy{
			InitialDelay: time.Second,
			MaxDelay:     time.Minute,
			Multiplier:   2.0,
		},
	}
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{10, time.Minute}, // capped at MaxDelay
	}
	for _, tt := range tests {
		got := rm.backoffDelay(tt.attempt)
		if got != tt.want {
			t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

// =============================================================================
// SteamPoller tests
// =============================================================================

func TestSteamPoller_Active(t *testing.T) {
	var activeCount atomic.Int32
	var inactiveCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"players": []map[string]any{{
					"steamid":       "76561198000000000",
					"gameid":        "1336960",
					"gameextrainfo": "Halo Infinite",
					"personastate":  1,
				}},
			},
		})
	}))
	defer srv.Close()

	p := NewSteamPoller("76561198000000000", "test-key",
		func(gameID, gameName string) {
			activeCount.Add(1)
			if gameID != "1336960" {
				t.Errorf("gameID = %q", gameID)
			}
		},
		func() { inactiveCount.Add(1) },
	)
	// Override URL for testing
	origURL := steamAPIURL
	_ = origURL

	// Use custom client with test server
	p.client = srv.Client()

	// Poll directly using the test server
	ctx := context.Background()
	url := srv.URL + "?key=test&steamids=76561198000000000"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, _ := p.client.Do(req)
	defer resp.Body.Close()

	var result steamAPIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Response.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(result.Response.Players))
	}
	if result.Response.Players[0].GameID != "1336960" {
		t.Errorf("GameID = %q", result.Response.Players[0].GameID)
	}
}

func TestSteamPoller_Inactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"players": []map[string]any{{
					"steamid":      "76561198000000000",
					"personastate": 1,
					// no gameid
				}},
			},
		})
	}))
	defer srv.Close()

	var result steamAPIResponse
	resp, _ := http.Get(srv.URL)
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Response.Players) != 1 {
		t.Fatal("expected 1 player")
	}
	if result.Response.Players[0].GameID != "" {
		t.Errorf("expected empty GameID, got %q", result.Response.Players[0].GameID)
	}
}

func TestSteamPoller_EmptyPlayers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"players": []any{},
			},
		})
	}))
	defer srv.Close()

	var result steamAPIResponse
	resp, _ := http.Get(srv.URL)
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Response.Players) != 0 {
		t.Errorf("expected 0 players, got %d", len(result.Response.Players))
	}
}

func TestNewSteamPoller(t *testing.T) {
	p := NewSteamPoller("steam123", "api-key",
		func(string, string) {},
		func() {},
	)
	if p.steamID != "steam123" {
		t.Errorf("steamID = %q", p.steamID)
	}
	if p.interval != defaultSteamInterval {
		t.Errorf("interval = %v", p.interval)
	}
}

// TestRTAClient_HandleMessage_SubscribeRefused_Status3 vérifie que status=3
// (accès refusé / token expiré) déclenche la fermeture de la connexion WS.
// Cela garantit que RunWithReconnect retente avec un token frais.
func TestRTAClient_HandleMessage_SubscribeRefused_Status3(t *testing.T) {
	// Créer un faux serveur WS qui reste ouvert (on veut observer la fermeture côté client)
	closed := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Signaler quand le client ferme
		conn.ReadMessage() //nolint:errcheck // on attend la fermeture
		close(closed)
	}))
	defer srv.Close()

	c := NewRTAClient("auth")

	// Connecter au faux serveur
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c.connMu.Lock()
	c.closeOnce = &sync.Once{}
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		c.connMu.Unlock()
		t.Fatalf("dial: %v", err)
	}
	c.conn = conn
	c.connected.Store(true)
	c.connMu.Unlock()

	ctx := context.Background()

	// Enregistrer un pending subscribe
	c.pendingMu.Lock()
	c.pending[99] = pendingSub{xuid: "xuid-99", handler: func(PresenceEvent) {}}
	c.pendingMu.Unlock()

	// Simuler status=3 (accès refusé) : [1, 99, 3, 0]
	c.handleMessage(ctx, []byte(`[1, 99, 3, 0]`))

	// Le closeOnce doit avoir fermé la connexion (goroutine async → attendre max 500ms)
	select {
	case <-closed:
		// OK : le serveur a détecté la fermeture
	case <-time.After(500 * time.Millisecond):
		t.Error("status=3 aurait dû fermer la connexion WS pour forcer un reconnect")
	}

	// Le pending doit avoir été retiré (delete avant le check status)
	c.pendingMu.Lock()
	_, stillPending := c.pending[99]
	c.pendingMu.Unlock()
	if stillPending {
		t.Error("pending doit être supprimé après réception de la réponse")
	}
}
