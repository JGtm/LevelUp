package presence

import (
	"context"
	"encoding/json"
	"fmt"
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

// Format /titles/<TID> + nonce observé en prod 2026-05-25 :
// state + devices[].titles[]. JGtm joue à Halo Infinite ; on doit
// extraire le titre actif depuis devices[0].titles[0].
func TestParsePresencePayload_DevicesFormat_Active(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"2533274823110022","state":"Online","devices":[{"type":"WindowsOneCore","titles":[{"id":"2043073184","name":"Halo Infinite","placement":"Full","state":"Active","lastModified":"2026-05-25T19:53:30.5573644"}]}]}`)
	event, err := ParsePresencePayload(raw, "fallback")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.XUID != "2533274823110022" {
		t.Errorf("XUID = %q, want 2533274823110022", event.XUID)
	}
	if event.PresenceState != "Online" {
		t.Errorf("PresenceState = %q, want Online", event.PresenceState)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil (parser n'a pas reconnu le format devices[])")
	}
	if event.PresenceDetail.TitleID != "2043073184" {
		t.Errorf("TitleID = %q, want 2043073184", event.PresenceDetail.TitleID)
	}
	if event.PresenceDetail.TitleName != "Halo Infinite" {
		t.Errorf("TitleName = %q, want Halo Infinite", event.PresenceDetail.TitleName)
	}
	if event.PresenceDetail.State != "Active" {
		t.Errorf("State = %q, want Active", event.PresenceDetail.State)
	}
	if !event.PresenceDetail.IsGame {
		t.Error("IsGame should be true (topic /titles/<TID> implique game)")
	}
	if !event.PresenceDetail.IsPrimary {
		t.Error("IsPrimary should be true (placement Full)")
	}
	if event.PresenceDetail.Device != "WindowsOneCore" {
		t.Errorf("Device = %q, want WindowsOneCore", event.PresenceDetail.Device)
	}
}

// Snapshot Offline avec lastSeen : on parse OK mais PresenceDetail reste nil
// (pas de devices[].titles[] actifs).
func TestParsePresencePayload_DevicesFormat_OfflineLastSeen(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"2533274833178266","state":"Offline","lastSeen":{"deviceType":"Win32","titleId":"2043073184","titleName":"Halo Infinite","timestamp":"2026-04-13T21:10:46.8592228"}}`)
	event, err := ParsePresencePayload(raw, "fallback")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q, want Offline", event.PresenceState)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for lastSeen-only payload")
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
	err := c.Subscribe(context.Background(), "xuid-123", "1144039928", func(_ PresenceEvent) {})
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
	resp, err := p.client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	var result steamAPIResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Response.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(result.Response.Players))
	}
	if result.Response.Players[0].GameID != "1336960" {
		t.Errorf("GameID = %q", result.Response.Players[0].GameID)
	}
}

func TestSteamPoller_Inactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
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
	resp, err := http.Get(srv.URL) //nolint:noctx
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Response.Players) != 1 {
		t.Fatal("expected 1 player")
	}
	if result.Response.Players[0].GameID != "" {
		t.Errorf("expected empty GameID, got %q", result.Response.Players[0].GameID)
	}
}

func TestSteamPoller_EmptyPlayers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"players": []any{},
			},
		})
	}))
	defer srv.Close()

	var result steamAPIResponse
	resp, err := http.Get(srv.URL) //nolint:noctx
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(&result)

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
// (accès refusé / token expiré) déclenche la fermeture de la connexion WS
// APRÈS un grace period de 2s si AUCUN subscribe n'a réussi entretemps.
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

	// Le pending doit avoir été retiré (delete avant le check status)
	c.pendingMu.Lock()
	_, stillPending := c.pending[99]
	c.pendingMu.Unlock()
	if stillPending {
		t.Error("pending doit être supprimé après réception de la réponse")
	}

	// La fermeture est différée de 2s (grace period). On attend jusqu'à 3s.
	select {
	case <-closed:
		// OK : le serveur a détecté la fermeture après le grace period
	case <-time.After(3 * time.Second):
		t.Error("status=3 sans aucun succès aurait dû fermer la connexion WS après le grace period")
	}
}

// TestRTAClient_HandleMessage_Status3_IgnoredWhenSubSucceeded vérifie que status=3
// est ignoré (pas de fermeture) si au moins un autre subscribe a réussi pendant
// le grace period — ce qui indique que c'est une privacy denial individuelle,
// pas une auth expirée globale.
func TestRTAClient_HandleMessage_Status3_IgnoredWhenSubSucceeded(t *testing.T) {
	closed := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.ReadMessage() //nolint:errcheck
		close(closed)
	}))
	defer srv.Close()

	c := NewRTAClient("auth")
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

	// Simuler un subscribe RÉUSSI (status=0) sur sub_id=42
	c.pendingMu.Lock()
	c.pending[10] = pendingSub{xuid: "xuid-ok", handler: func(PresenceEvent) {}}
	c.pendingMu.Unlock()
	c.handleMessage(ctx, []byte(`[1, 10, 0, 42]`))

	// Puis un status=3 — le grace period doit voir au moins 1 sub OK et NE PAS fermer
	c.pendingMu.Lock()
	c.pending[99] = pendingSub{xuid: "xuid-bad", handler: func(PresenceEvent) {}}
	c.pendingMu.Unlock()
	c.handleMessage(ctx, []byte(`[1, 99, 3, 0]`))

	// Attendre 2.5s : la fermeture NE doit PAS arriver (au moins 1 sub OK)
	select {
	case <-closed:
		t.Error("status=3 ne doit PAS fermer la connexion quand au moins 1 sub a réussi")
	case <-time.After(2500 * time.Millisecond):
		// OK : aucune fermeture, comportement attendu
	}

	if c.IsAuthExpired() {
		t.Error("authExpired ne doit pas être set quand au moins 1 sub a réussi")
	}
}

// =============================================================================
// RunWithReconnect — scénarios auth expired
// =============================================================================

// newReconnectManagerForTest crée un ReconnectManager avec un waitFn instantané
// (pas de vrais sleeps) et connecte les champs nécessaires pour les tests.
func newReconnectManagerForTest(
	client *RTAClient,
	connectFunc func(context.Context) error,
	onAuthExpired func(context.Context) error,
) *ReconnectManager {
	return &ReconnectManager{
		client:        client,
		policy:        DefaultReconnectPolicy(),
		connectFunc:   connectFunc,
		OnAuthExpired: onAuthExpired,
		// waitFn instantané pour que les tests ne bloquent pas.
		waitFn: func(ctx context.Context, _ time.Duration) bool {
			select {
			case <-ctx.Done():
				return false
			default:
				return true
			}
		},
	}
}

// TestReconnectManager_RunWithReconnect_AuthExpired_CallsOnAuthExpiredBeforeConnect
// vérifie que OnAuthExpired est appelé avant connectFunc quand authExpired=true.
func TestReconnectManager_RunWithReconnect_AuthExpired_CallsOnAuthExpiredBeforeConnect(t *testing.T) {
	client := NewRTAClient("header")
	client.authExpired.Store(true)

	var order []string
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())

	onAuthExpired := func(_ context.Context) error {
		mu.Lock()
		order = append(order, "onAuthExpired")
		mu.Unlock()
		return nil
	}
	connectFunc := func(_ context.Context) error {
		mu.Lock()
		order = append(order, "connectFunc")
		mu.Unlock()
		cancel() // arrêter la boucle après le premier connect
		return nil
	}

	rm := newReconnectManagerForTest(client, connectFunc, onAuthExpired)
	// connectFunc annule le ctx, donc ReadLoop retournera immédiatement car le contexte
	// est déjà annulé quand la boucle principale le check après le connect réussi.
	rm.RunWithReconnect(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(order) < 2 {
		t.Fatalf("attendu au moins 2 appels, got %v", order)
	}
	if order[0] != "onAuthExpired" {
		t.Errorf("premier appel attendu 'onAuthExpired', got %q", order[0])
	}
	if order[1] != "connectFunc" {
		t.Errorf("deuxième appel attendu 'connectFunc', got %q", order[1])
	}
}

// TestReconnectManager_RunWithReconnect_AuthExpired_ResetAfterSuccess
// vérifie que IsAuthExpired() retourne false après un refresh réussi.
func TestReconnectManager_RunWithReconnect_AuthExpired_ResetAfterSuccess(t *testing.T) {
	client := NewRTAClient("header")
	client.authExpired.Store(true)

	ctx, cancel := context.WithCancel(context.Background())

	onAuthExpired := func(_ context.Context) error { return nil }
	connectFunc := func(_ context.Context) error {
		cancel()
		return nil
	}

	rm := newReconnectManagerForTest(client, connectFunc, onAuthExpired)
	rm.RunWithReconnect(ctx)

	if client.IsAuthExpired() {
		t.Error("authExpired devrait être false après un refresh réussi")
	}
}

// TestReconnectManager_RunWithReconnect_AuthExpired_CallbackError
// vérifie qu'un callback en erreur n'appelle pas connectFunc et applique le délai.
func TestReconnectManager_RunWithReconnect_AuthExpired_CallbackError(t *testing.T) {
	client := NewRTAClient("header")
	client.authExpired.Store(true)

	connectCalled := false
	waitCalled := false

	ctx, cancel := context.WithCancel(context.Background())

	onAuthExpired := func(_ context.Context) error {
		return fmt.Errorf("refresh échoué")
	}
	connectFunc := func(_ context.Context) error {
		connectCalled = true
		return nil
	}

	callCount := 0
	rm := &ReconnectManager{
		client:        client,
		policy:        DefaultReconnectPolicy(),
		connectFunc:   connectFunc,
		OnAuthExpired: onAuthExpired,
		waitFn: func(ctx context.Context, _ time.Duration) bool {
			waitCalled = true
			callCount++
			if callCount >= 1 {
				cancel() // arrêter après le premier wait
			}
			return false // annulation immédiate
		},
	}
	rm.RunWithReconnect(ctx)

	if connectCalled {
		t.Error("connectFunc ne doit pas être appelé si OnAuthExpired échoue")
	}
	if !waitCalled {
		t.Error("waitFn doit être appelé après un échec de OnAuthExpired")
	}
}

// TestReconnectManager_RunWithReconnect_AuthExpired_NoCallback
// vérifie le comportement quand OnAuthExpired est nil.
func TestReconnectManager_RunWithReconnect_AuthExpired_NoCallback(t *testing.T) {
	client := NewRTAClient("header")
	client.authExpired.Store(true)

	connectCalled := false
	waitCalled := false
	ctx, cancel := context.WithCancel(context.Background())

	rm := &ReconnectManager{
		client:      client,
		policy:      DefaultReconnectPolicy(),
		connectFunc: func(_ context.Context) error { connectCalled = true; return nil },
		// OnAuthExpired intentionnellement nil
		waitFn: func(ctx context.Context, _ time.Duration) bool {
			waitCalled = true
			cancel()
			return false
		},
	}
	rm.RunWithReconnect(ctx)

	if connectCalled {
		t.Error("connectFunc ne doit pas être appelé si OnAuthExpired est nil")
	}
	if !waitCalled {
		t.Error("waitFn doit être appelé quand OnAuthExpired est nil")
	}
}
