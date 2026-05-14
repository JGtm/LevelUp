package presence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Ces tests verifient quels status codes de subscribe response sont acceptes
// comme une souscription valide. Microsoft ne documente pas publiquement les
// status codes du protocole RTA WebSocket ; l'utilisateur LevelUp a observe
// empiriquement que Xbox Live renvoyait status=2 pour signaler une souscription
// reussie sur userpresence.xboxlive.com/users/xuid(X)/titles/Y. Le code initial
// traitait tout status != 0 comme un refus, perdant silencieusement les events
// de presence qui arrivaient ensuite (sub_id non enregistre dans c.subs).

// TestRTAClient_HandleMessage_Subscribe_Status2_RegistersSubscription verifie
// qu'une subscribe response avec status=2 enregistre bien la souscription dans
// c.subs[subID] et c.xuidToSub[xuid], de sorte que les events suivants soient
// dispatches au handler.
func TestRTAClient_HandleMessage_Subscribe_Status2_RegistersSubscription(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	c.pendingMu.Lock()
	c.pending[99] = pendingSub{
		xuid:    "xuid-status2",
		handler: func(PresenceEvent) {},
	}
	c.pendingMu.Unlock()

	// Simuler subscribe response status=2 (succes selon test empirique
	// utilisateur — voir doc du fichier).
	c.handleMessage(ctx, []byte(`[1, 99, 2, 42]`))

	c.subsMu.RLock()
	sub, ok := c.subs[42]
	subIDByXUID, xuidOk := c.xuidToSub["xuid-status2"]
	c.subsMu.RUnlock()

	if !ok {
		t.Fatal("subscription 42 doit etre enregistree dans c.subs apres status=2")
	}
	if sub.XUID != "xuid-status2" {
		t.Errorf("sub.XUID = %q, want %q", sub.XUID, "xuid-status2")
	}
	if !xuidOk || subIDByXUID != 42 {
		t.Errorf("c.xuidToSub[xuid-status2] = %d (ok=%v), want 42", subIDByXUID, xuidOk)
	}

	if c.IsAuthExpired() {
		t.Error("authExpired ne doit pas etre set pour status=2 (ce n'est pas un refus)")
	}
}

// TestRTAClient_HandleMessage_Subscribe_Status2_EventDispatched verifie qu'un
// event push pour un subID enregistre via status=2 est bien dispatche au
// handler avec le bon payload. C'est le test de bout en bout du pipeline
// status=2 -> sub registree -> event delivre.
func TestRTAClient_HandleMessage_Subscribe_Status2_EventDispatched(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	var received PresenceEvent
	var receivedMu sync.Mutex

	c.pendingMu.Lock()
	c.pending[1] = pendingSub{
		xuid: "xuid-evt",
		handler: func(e PresenceEvent) {
			receivedMu.Lock()
			received = e
			receivedMu.Unlock()
		},
	}
	c.pendingMu.Unlock()

	// Subscribe response status=2 sans initial data : juste enregistrer la sub.
	c.handleMessage(ctx, []byte(`[1, 1, 2, 77]`))

	// Event push pour sub_id=77 avec payload presence Halo Infinite.
	c.handleMessage(ctx, []byte(`[5, 77, {"xuid":"xuid-evt","presenceState":"Online","presenceDetails":[{"titleid":"2043073184","titleName":"Halo Infinite","isGame":true,"isPrimary":true,"state":"Active"}]}]`))

	receivedMu.Lock()
	defer receivedMu.Unlock()
	if received.XUID != "xuid-evt" {
		t.Errorf("received.XUID = %q, want %q (event jete au lieu d'etre delivre)", received.XUID, "xuid-evt")
	}
	if received.PresenceState != "Online" {
		t.Errorf("received.PresenceState = %q", received.PresenceState)
	}
	if received.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil — event presence non parse correctement")
	}
	if received.PresenceDetail.TitleID != "2043073184" {
		t.Errorf("TitleID = %q, want 2043073184", received.PresenceDetail.TitleID)
	}
}

// TestRTAClient_HandleMessage_Subscribe_Status2_WithInitialData verifie qu'un
// payload initial joint a la subscribe response avec status=2 est aussi
// dispatche au handler (cas ou Xbox envoie l'etat courant directement dans la
// reponse subscribe).
func TestRTAClient_HandleMessage_Subscribe_Status2_WithInitialData(t *testing.T) {
	c := NewRTAClient("auth")
	ctx := context.Background()

	var received PresenceEvent
	var receivedMu sync.Mutex

	c.pendingMu.Lock()
	c.pending[5] = pendingSub{
		xuid: "xuid-init",
		handler: func(e PresenceEvent) {
			receivedMu.Lock()
			received = e
			receivedMu.Unlock()
		},
	}
	c.pendingMu.Unlock()

	// [1, seq=5, status=2, sub_id=88, initial_payload]
	c.handleMessage(ctx, []byte(`[1, 5, 2, 88, {"xuid":"xuid-init","presenceState":"Online","presenceDetails":[{"titleid":"2043073184","titleName":"Halo Infinite","isGame":true,"isPrimary":true}]}]`))

	receivedMu.Lock()
	defer receivedMu.Unlock()
	if received.XUID != "xuid-init" {
		t.Errorf("initial data non dispatchee pour status=2 — received.XUID = %q", received.XUID)
	}
	if received.PresenceDetail == nil || received.PresenceDetail.TitleID != "2043073184" {
		t.Error("PresenceDetail.TitleID Halo Infinite non recu sur initial data status=2")
	}
}

// TestRTAClient_HandleMessage_Subscribe_OtherStatuses_StillRefused verifie que
// le patch status=2 n'ouvre pas la porte a tous les autres status — les codes
// autres que {0, 2} doivent toujours etre traites comme un refus et NE PAS
// enregistrer la sub.
func TestRTAClient_HandleMessage_Subscribe_OtherStatuses_StillRefused(t *testing.T) {
	for _, status := range []int{1, 4, 5, 10, 42} {
		t.Run("status_"+itoa(status), func(t *testing.T) {
			c := NewRTAClient("auth")
			ctx := context.Background()

			c.pendingMu.Lock()
			c.pending[1] = pendingSub{
				xuid:    "xuid-refused",
				handler: func(PresenceEvent) {},
			}
			c.pendingMu.Unlock()

			msg := []byte(`[1, 1, ` + itoa(status) + `, 99]`)
			c.handleMessage(ctx, msg)

			c.subsMu.RLock()
			_, ok := c.subs[99]
			c.subsMu.RUnlock()
			if ok {
				t.Errorf("status=%d : sub ne doit pas etre enregistree (c'est un refus)", status)
			}
		})
	}
}

// TestRTAClient_EndToEnd_Status2_ViaMockWebSocket verifie le pipeline complet
// via un vrai serveur WebSocket de mock. Le mock accepte la connexion (avec
// subprotocol rta.xboxlive.com.V2), repond aux subscribes avec status=2 et
// pousse un event de presence Halo Infinite. Le client doit traiter ce flux
// comme une souscription valide et appeler le handler avec l'event.
//
// Ce test sert de proof-of-concept pour le pipeline RTA complet (Connect +
// Subscribe + ReadLoop + dispatch) sans dependance Xbox Live reelle.
func TestRTAClient_EndToEnd_Status2_ViaMockWebSocket(t *testing.T) {
	received := make(chan PresenceEvent, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifier le subprotocol obligatoire envoye par le client.
		if got := r.Header.Get("Sec-WebSocket-Protocol"); !strings.Contains(got, "rta.xboxlive.com.V2") {
			t.Errorf("subprotocol manquant : Sec-WebSocket-Protocol = %q", got)
		}
		upgrader := websocket.Upgrader{
			CheckOrigin:  func(*http.Request) bool { return true },
			Subprotocols: []string{"rta.xboxlive.com.V2"},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Lire le premier subscribe envoye par le client.
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg []json.RawMessage
		if jerr := json.Unmarshal(raw, &msg); jerr != nil || len(msg) < 3 {
			return
		}
		var msgType int
		_ = json.Unmarshal(msg[0], &msgType)
		var seq int64
		_ = json.Unmarshal(msg[1], &seq)
		if msgType != rtaSubscribe {
			return
		}

		// Repondre avec status=2 + subID=200 (succes empirique observe par
		// l'utilisateur).
		resp, _ := json.Marshal([]any{rtaSubscribe, seq, 2, 200})
		_ = conn.WriteMessage(websocket.TextMessage, resp)

		// Pousser un event de presence Halo Infinite sur la sub.
		evtPayload := json.RawMessage(`{"xuid":"xuid-e2e","presenceState":"Online","presenceDetails":[{"titleid":"2043073184","titleName":"Halo Infinite","isGame":true,"isPrimary":true,"state":"Active"}]}`)
		evt, _ := json.Marshal([]any{rtaEvent, 200, evtPayload})
		_ = conn.WriteMessage(websocket.TextMessage, evt)

		// Garder la connexion ouverte un instant pour laisser le client lire.
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewRTAClient("XBL3.0 x=hash;token")
	c.connMu.Lock()
	c.closeOnce = &sync.Once{}
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	dialer := websocket.Dialer{
		Subprotocols: []string{"rta.xboxlive.com.V2"},
	}
	header := http.Header{"Authorization": {"XBL3.0 x=hash;token"}}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		c.connMu.Unlock()
		t.Fatalf("dial mock: %v", err)
	}
	c.conn = conn
	c.connected.Store(true)
	c.connMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.ReadLoop(ctx) //nolint:errcheck // loop returns on ctx cancel

	if err := c.Subscribe(ctx, "xuid-e2e", "2043073184", func(e PresenceEvent) {
		received <- e
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case e := <-received:
		if e.XUID != "xuid-e2e" {
			t.Errorf("received.XUID = %q, want xuid-e2e", e.XUID)
		}
		if e.PresenceDetail == nil || e.PresenceDetail.TitleID != "2043073184" {
			t.Errorf("event presence non recu correctement : %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("aucun event de presence recu via mock WebSocket avec status=2 — la subscription a ete jetee silencieusement (bug)")
	}
}

// itoa convertit un int en string sans importer strconv au niveau du fichier.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
