// Package presence — rta_client.go : client WebSocket Xbox Live RTA (Real-Time Activity).
//
// Protocole RTA :
//   - Endpoint : wss://rta.xboxlive.com/connect
//   - Auth : header Authorization: XBL3.0 x=<userhash>;<xsts_token>
//   - Subscribe : [1, 1, "https://userpresence.xboxlive.com/users/xuid(<XUID>)/richpresence"]
//   - Event push : [5, <sub_id>, <payload>]
//   - Heartbeat : côté serveur via ping frames (gorilla/websocket gère automatiquement)
//
// Le client maintient UNE connexion WebSocket partagée pour tous les joueurs trackés.
// Les événements de présence sont émis via un callback par joueur.
package presence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// rtaEndpoint est le WebSocket RTA Xbox Live.
	rtaEndpoint = "wss://rta.xboxlive.com/connect"

	// rtaPresenceTopicFmt est le template du topic de présence par XUID.
	rtaPresenceTopicFmt = "https://userpresence.xboxlive.com/users/xuid(%s)/richpresence"

	// writeTimeout pour les messages WebSocket sortants.
	writeTimeout = 10 * time.Second

	// pongWait est le délai max sans pong du serveur avant déconnexion.
	pongWait = 60 * time.Second
)

// RTA message types (protocole Xbox)
const (
	rtaSubscribe   = 1 // [1, <seq>, "<topic>"]
	rtaUnsubscribe = 2 // [2, <seq>, <sub_id>]
	rtaEvent       = 5 // [5, <sub_id>, <payload>]
)

// PresenceEvent représente un événement de changement de présence.
type PresenceEvent struct {
	XUID           string
	PresenceState  string          // "Online", "Offline", "Away"
	PresenceDetail *PresenceDetail // nil si offline ou pas de jeu
}

// PresenceDetail contient les informations du titre en cours.
type PresenceDetail struct {
	TitleID   string
	TitleName string
	IsGame    bool
	IsPrimary bool
	Device    string
	State     string // "Active", "Inactive"
}

// EventHandler est le callback appelé quand un event de présence est reçu.
type EventHandler func(event PresenceEvent)

// subscription enregistre un abonnement RTA actif.
type subscription struct {
	XUID    string
	SubID   int
	Handler EventHandler
}

// RTAClient gère la connexion WebSocket RTA et les abonnements de présence.
type RTAClient struct {
	authHeader string
	conn       *websocket.Conn
	connMu     sync.Mutex

	nextSeq atomic.Int64
	// subsBySeq : map[seqID] → XUID (pending subscribe, avant confirmation)
	pendingMu sync.Mutex
	pending   map[int64]pendingSub

	// subs : map[subID] → subscription (confirmées)
	subsMu sync.RWMutex
	subs   map[int]*subscription

	// xuidToSub : map[XUID] → subID (lookup rapide)
	xuidToSub map[string]int

	// connected indique si le WebSocket est actif
	connected atomic.Bool

	// authExpired est positionné à true quand un subscribe retourne status=3.
	// Permet à RunWithReconnect de déclencher un refresh XSTS avant de reconnecter.
	authExpired atomic.Bool

	// closeOnce évite de fermer la connexion plusieurs fois pour la même cause.
	// Réinitialisé à chaque Connect().
	closeOnce *sync.Once
}

type pendingSub struct {
	xuid    string
	handler EventHandler
}

// NewRTAClient crée un client RTA non connecté.
func NewRTAClient(authHeader string) *RTAClient {
	c := &RTAClient{
		authHeader: authHeader,
		pending:    make(map[int64]pendingSub),
		subs:       make(map[int]*subscription),
		xuidToSub:  make(map[string]int),
		closeOnce:  &sync.Once{},
	}
	c.nextSeq.Store(1)
	return c
}

// Connect établit la connexion WebSocket RTA.
func (c *RTAClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
	// Réinitialiser le closeOnce + authExpired pour la nouvelle connexion.
	c.closeOnce = &sync.Once{}
	c.authExpired.Store(false)

	slog.InfoContext(ctx, "rta: connexion WebSocket", "endpoint", rtaEndpoint)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}
	header := http.Header{}
	header.Set("Authorization", c.authHeader)

	conn, resp, err := dialer.DialContext(ctx, rtaEndpoint, header)
	if err != nil {
		if resp != nil {
			slog.ErrorContext(ctx, "rta: échec connexion WebSocket",
				"err", err,
				"status", resp.StatusCode,
			)
		} else {
			slog.ErrorContext(ctx, "rta: échec connexion WebSocket", "err", err)
		}
		return fmt.Errorf("rta connect: %w", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	c.conn = conn
	c.connected.Store(true)
	slog.InfoContext(ctx, "rta: connecté")
	return nil
}

// Subscribe s'abonne au topic de présence d'un joueur.
func (c *RTAClient) Subscribe(ctx context.Context, xuid string, handler EventHandler) error {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return fmt.Errorf("rta: pas connecté")
	}

	topic := fmt.Sprintf(rtaPresenceTopicFmt, xuid)
	seq := c.nextSeq.Add(1) - 1
	msg := []any{rtaSubscribe, seq, topic}

	c.pendingMu.Lock()
	c.pending[seq] = pendingSub{xuid: xuid, handler: handler}
	c.pendingMu.Unlock()

	data, _ := json.Marshal(msg)
	if err := c.writeMessage(data); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, seq)
		c.pendingMu.Unlock()
		return fmt.Errorf("rta subscribe xuid=%s: %w", xuid, err)
	}

	slog.InfoContext(ctx, "rta: subscribe envoyé",
		"xuid", xuid,
		"seq", seq,
		"topic", topic,
	)
	return nil
}

// ReadLoop lit les messages RTA en boucle. Bloquant — à lancer dans une goroutine.
// Retourne quand la connexion est fermée ou ctx annulé.
func (c *RTAClient) ReadLoop(ctx context.Context) error {
	slog.InfoContext(ctx, "rta: read loop démarré")
	defer func() {
		c.connected.Store(false)
		slog.InfoContext(ctx, "rta: read loop terminé")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()
		if conn == nil {
			return fmt.Errorf("rta: connexion fermée")
		}

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.InfoContext(ctx, "rta: close frame reçu", "err", err)
			} else {
				slog.WarnContext(ctx, "rta: erreur lecture", "err", err)
			}
			return fmt.Errorf("rta read: %w", err)
		}

		c.handleMessage(ctx, raw)
	}
}

// Close ferme la connexion WebSocket proprement.
func (c *RTAClient) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	c.connected.Store(false)
	if c.conn == nil {
		return nil
	}
	err := c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	c.conn.Close()
	c.conn = nil
	slog.Info("rta: connexion fermée")
	return err
}

// IsConnected retourne true si le WebSocket est actif.
func (c *RTAClient) IsConnected() bool {
	return c.connected.Load()
}

// UpdateAuth met à jour le header d'auth (après refresh XSTS).
func (c *RTAClient) UpdateAuth(authHeader string) {
	c.authHeader = authHeader
}

// IsAuthExpired retourne true si un subscribe a été refusé avec status=3
// (token XSTS expiré). Utilisé par RunWithReconnect pour déclencher un refresh.
func (c *RTAClient) IsAuthExpired() bool {
	return c.authExpired.Load()
}

// ResetAuthExpired remet le flag à false. Appelé par RunWithReconnect après
// avoir traité le refresh, juste avant la reconnexion.
func (c *RTAClient) ResetAuthExpired() {
	c.authExpired.Store(false)
}

// Subscriptions retourne la liste des XUIDs actuellement abonnés.
func (c *RTAClient) Subscriptions() []string {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()
	xuids := make([]string, 0, len(c.subs))
	for _, sub := range c.subs {
		xuids = append(xuids, sub.XUID)
	}
	return xuids
}

// handleMessage dispatche un message RTA brut.
func (c *RTAClient) handleMessage(ctx context.Context, raw []byte) {
	var msg []json.RawMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		slog.WarnContext(ctx, "rta: message non-JSON", "raw", string(raw))
		return
	}
	if len(msg) < 2 {
		slog.WarnContext(ctx, "rta: message trop court", "len", len(msg))
		return
	}

	var msgType int
	if err := json.Unmarshal(msg[0], &msgType); err != nil {
		slog.WarnContext(ctx, "rta: type de message invalide", "raw", string(msg[0]))
		return
	}

	switch msgType {
	case rtaSubscribe:
		c.handleSubscribeResponse(ctx, msg)
	case rtaEvent:
		c.handleEvent(ctx, msg)
	default:
		slog.DebugContext(ctx, "rta: message type ignoré", "type", msgType)
	}
}

// handleSubscribeResponse traite la confirmation d'un subscribe.
// Format : [1, <seq>, <status>, <sub_id>]  ou  [1, <seq>, <status>, <sub_id>, <initial_data>]
func (c *RTAClient) handleSubscribeResponse(ctx context.Context, msg []json.RawMessage) {
	if len(msg) < 4 {
		slog.WarnContext(ctx, "rta: subscribe response trop courte", "len", len(msg))
		return
	}
	var seq int64
	var status int
	var subID int
	json.Unmarshal(msg[1], &seq)
	json.Unmarshal(msg[2], &status)
	json.Unmarshal(msg[3], &subID)

	c.pendingMu.Lock()
	ps, ok := c.pending[seq]
	delete(c.pending, seq)
	c.pendingMu.Unlock()

	if !ok {
		slog.WarnContext(ctx, "rta: subscribe response pour seq inconnue", "seq", seq)
		return
	}

	if status != 0 {
		slog.WarnContext(ctx, "rta: subscribe refusé",
			"xuid", ps.xuid,
			"status", status,
			"sub_id", subID,
		)
		if status == 3 {
			// Status=3 = accès refusé, probablement token XSTS expiré.
			// Signaler authExpired pour que RunWithReconnect rafraîchisse le token
			// AVANT de reconnecter (évite la boucle infinie reconnect→status=3).
			c.authExpired.Store(true)
			c.closeOnce.Do(func() {
				slog.WarnContext(ctx, "rta: status=3 — authExpired signalé, fermeture connexion")
				c.connMu.Lock()
				conn := c.conn
				c.connMu.Unlock()
				if conn != nil {
					go conn.Close()
				}
			})
		}
		return
	}

	c.subsMu.Lock()
	c.subs[subID] = &subscription{
		XUID:    ps.xuid,
		SubID:   subID,
		Handler: ps.handler,
	}
	c.xuidToSub[ps.xuid] = subID
	c.subsMu.Unlock()

	slog.InfoContext(ctx, "rta: subscribe confirmé",
		"xuid", ps.xuid,
		"sub_id", subID,
	)

	// Si la réponse inclut des données initiales, les traiter comme un event
	if len(msg) >= 5 {
		c.dispatchPayload(ctx, subID, msg[4])
	}
}

// handleEvent traite un event push RTA.
// Format : [5, <sub_id>, <payload>]
func (c *RTAClient) handleEvent(ctx context.Context, msg []json.RawMessage) {
	if len(msg) < 3 {
		slog.WarnContext(ctx, "rta: event trop court", "len", len(msg))
		return
	}
	var subID int
	json.Unmarshal(msg[1], &subID)

	c.dispatchPayload(ctx, subID, msg[2])
}

// dispatchPayload parse le payload de présence et appelle le handler.
func (c *RTAClient) dispatchPayload(ctx context.Context, subID int, raw json.RawMessage) {
	c.subsMu.RLock()
	sub, ok := c.subs[subID]
	c.subsMu.RUnlock()
	if !ok {
		slog.DebugContext(ctx, "rta: event pour sub_id inconnu", "sub_id", subID)
		return
	}

	event, err := ParsePresencePayload(raw, sub.XUID)
	if err != nil {
		slog.WarnContext(ctx, "rta: erreur parsing présence",
			"xuid", sub.XUID,
			"err", err,
			"raw", string(raw),
		)
		return
	}

	slog.InfoContext(ctx, "rta: event de présence",
		"xuid", event.XUID,
		"state", event.PresenceState,
		"title", titleFromEvent(event),
	)

	sub.Handler(event)
}

func titleFromEvent(e PresenceEvent) string {
	if e.PresenceDetail != nil {
		return e.PresenceDetail.TitleName
	}
	return "none"
}

// writeMessage envoie un message avec timeout.
func (c *RTAClient) writeMessage(data []byte) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("rta: pas connecté")
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
