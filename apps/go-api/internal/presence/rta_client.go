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
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// rtaEndpoint est le WebSocket RTA Xbox Live.
	// L'URL réelle inclut un nonce query param obtenu via rtaNonceEndpoint.
	rtaEndpoint = "wss://rta.xboxlive.com/connect"

	// rtaNonceEndpoint retourne un nonce one-shot requis pour upgrader la
	// connexion WebSocket en mode "push complet". Sans nonce, Xbox accepte
	// la connexion mais limite les subscribes : les events ne sont pas
	// poussés (snapshot one-shot uniquement). Cf. LucienHH/xbox-rta.
	rtaNonceEndpoint = "https://rta.xboxlive.com/nonce"

	// rtaPresenceTopicFmt est le template du topic de présence par XUID + titleID.
	//
	// Diagnostic 2026-05-25 (3 endpoints testés post-fix nonce) :
	//   - `/titles/<TID>` (utilisé ici) : snapshot OK au subscribe, 0 push
	//     après (22 min d'observation, extinction Halo non détectée). Best
	//     effort = on a au moins le snapshot initial qui détecte si l'user
	//     est en jeu au moment de la connexion.
	//   - `/richpresence` : status=3 persistant même avec nonce. Le RelyingParty
	//     `http://xboxlive.com` (notre flow SPNKr) n'a pas les claims pour ce
	//     topic. LucienHH/xbox-rta passe par `Titles.XboxAppIOS` (impersonate
	//     l'app Xbox iOS, borderline ToS).
	//   - `/devices/current/titles/current` : non validé.
	//
	// Le push temps réel cross-titres n'étant pas accessible via notre auth,
	// la détection des transitions Active↔Inactive est faite côté REST poll
	// (cf. internal/presence/rest_client.go + internal/watcher/rest_poller.go).
	rtaPresenceTopicFmt = "https://userpresence.xboxlive.com/users/xuid(%s)/titles/%s"

	// writeTimeout pour les messages WebSocket sortants.
	writeTimeout = 10 * time.Second

	// pongWait est le délai max sans pong du serveur avant déconnexion.
	pongWait = 60 * time.Second

	// pingPeriod : intervalle entre 2 pings WebSocket sortants.
	// Doit être < pongWait (typiquement pongWait * 0.9 / 2). Sans ping
	// périodique, le serveur peut couper la connexion silencieusement et
	// le ReadDeadline n'est jamais reset car aucun pong n'arrive.
	pingPeriod = 25 * time.Second
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

	// status3GraceStarted garantit qu'un seul timer de grâce tourne par connexion
	// (évite N goroutines pour N status=3 reçus en burst).
	status3GraceStarted atomic.Bool

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

// fetchNonce récupère un nonce one-shot Xbox via GET /nonce. Sans nonce,
// la connexion WebSocket fonctionne en mode dégradé (les subscribes sont
// acceptés mais les events ne sont pas poussés). Le nonce est consommé
// par le query param ?nonce={nonce} de l'URL WebSocket à la connexion.
//
// Référence : github.com/LucienHH/xbox-rta (lib node prod-ready).
func (c *RTAClient) fetchNonce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rtaNonceEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("rta fetchNonce req: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rta fetchNonce do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rta fetchNonce status %d", resp.StatusCode)
	}
	var body struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("rta fetchNonce decode: %w", err)
	}
	if body.Nonce == "" {
		return "", fmt.Errorf("rta fetchNonce: nonce vide")
	}
	return body.Nonce, nil
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
	c.status3GraceStarted.Store(false)

	// Étape 1 : récupérer un nonce one-shot. Critique — sans ça les pushes
	// d'events ne marchent pas (cf. fetchNonce doc).
	nonce, err := c.fetchNonce(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "rta: fetchNonce échoué", "err", err)
		return fmt.Errorf("rta connect: %w", err)
	}
	// Nonce est base64 (44 chars = 32 bytes) → contient + / = qui DOIVENT être
	// URL-encodés sinon le handshake échoue en 400 (Bad Request) car Xbox
	// rejette les caractères non URL-safe dans le query param.
	connectURL := rtaEndpoint + "?nonce=" + url.QueryEscape(nonce)
	slog.InfoContext(ctx, "rta: connexion WebSocket", "endpoint", rtaEndpoint, "nonce_len", len(nonce))

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		// Sous-protocole obligatoire pour Xbox Live RTA. Sans cet en-tête
		// Sec-WebSocket-Protocol, le serveur accepte la connexion mais refuse
		// systématiquement tous les subscribes avec status=3.
		Subprotocols: []string{"rta.xboxlive.com.V2"},
	}
	// Pas de header Authorization sur le WS upgrade : LucienHH/xbox-rta ne le
	// fait pas non plus. L'auth est portée par le nonce one-shot dans l'URL —
	// envoyer Authorization en plus du nonce déclenche un 400 Bad Request
	// silencieux (Content-Length: 0) observé en prod 2026-05-25 21:46.
	conn, resp, err := dialer.DialContext(ctx, connectURL, nil)
	if err != nil {
		if resp != nil {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			slog.ErrorContext(ctx, "rta: échec connexion WebSocket",
				"err", err,
				"status", resp.StatusCode,
				"resp_body", string(bodyBytes),
				"resp_headers", fmt.Sprintf("%v", resp.Header),
			)
			_ = resp.Body.Close()
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
	// Read deadline initial — sans ça, ReadMessage bloque indéfiniment
	// si Xbox ne push rien (et donc aucun pong n'arrive pour reset).
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn = conn
	c.connected.Store(true)
	slog.InfoContext(ctx, "rta: connecté")

	// Goroutine de keepalive : envoie un ping toutes les pingPeriod.
	// Détecte les connexions silencieusement mortes (Xbox cut la TCP
	// sans envoyer de close frame) — le pongWait reset le ReadDeadline,
	// si pas de pong → ReadMessage retourne err → reconnect logic prend
	// le relais.
	go c.pingLoop(ctx, conn)
	return nil
}

// pingLoop envoie un ping WebSocket périodique pour garder la connexion
// vivante et détecter les déconnexions silencieuses. S'arrête quand le
// ctx est annulé OU quand la connexion change (Close puis Connect crée
// un nouveau conn — la goroutine de l'ancienne conn écrit sur l'ancien
// handle qui est fermé → erreur ignorée, return).
func (c *RTAClient) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.connMu.Lock()
			cur := c.conn
			c.connMu.Unlock()
			if cur != conn {
				// Une nouvelle connexion a remplacé celle-ci → on s'arrête.
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.DebugContext(ctx, "rta: ping échoué — connexion probablement morte", "err", err)
				return
			}
		}
	}
}

// Subscribe s'abonne au topic de présence d'un titre pour un joueur.
// titleID est le Title ID Xbox Live du jeu (ex: "2043073184" pour Halo Infinite).
func (c *RTAClient) Subscribe(ctx context.Context, xuid, titleID string, handler EventHandler) error {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return fmt.Errorf("rta: pas connecté")
	}

	topic := fmt.Sprintf(rtaPresenceTopicFmt, xuid, titleID)
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

// evaluateStatus3AfterGrace attend 2s puis décide si le status=3 reçu indique
// une vraie auth expirée (aucun subscribe n'a réussi) ou une simple privacy
// denial individuelle (au moins un subscribe a réussi). Garantit qu'une seule
// goroutine de grace tourne par connexion via status3GraceStarted CAS.
func (c *RTAClient) evaluateStatus3AfterGrace(ctx context.Context) {
	if !c.status3GraceStarted.CompareAndSwap(false, true) {
		return // déjà une grace en cours pour cette connexion
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(2 * time.Second):
	}
	c.subsMu.RLock()
	successCount := len(c.subs)
	c.subsMu.RUnlock()
	if successCount > 0 {
		slog.WarnContext(ctx, "rta: status=3 ignoré après grace (au moins 1 sub OK — privacy individuelle)",
			"successful_subs", successCount,
		)
		return
	}
	c.authExpired.Store(true)
	c.closeOnce.Do(func() {
		slog.WarnContext(ctx, "rta: aucun sub réussi après grace — auth expirée, fermeture connexion")
		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()
		if conn != nil {
			go conn.Close()
		}
	})
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
	if err := json.Unmarshal(msg[1], &seq); err != nil {
		slog.WarnContext(ctx, "rta: subscribe response seq invalide", "err", err)
	}
	if err := json.Unmarshal(msg[2], &status); err != nil {
		slog.WarnContext(ctx, "rta: subscribe response status invalide", "err", err)
	}
	// Xbox renvoie subID comme int pour certains endpoints (/titles/<TID>)
	// et comme string pour d'autres (/richpresence). On essaie int d'abord,
	// puis fallback string convertie via FNV hash pour avoir un int stable
	// utilisable comme clé dans la map c.subs.
	if err := json.Unmarshal(msg[3], &subID); err != nil {
		var subIDStr string
		if errStr := json.Unmarshal(msg[3], &subIDStr); errStr == nil {
			subID = hashSubIDString(subIDStr)
			slog.DebugContext(ctx, "rta: subID string convertie en int via hash",
				"sub_id_str", subIDStr, "sub_id_int", subID)
		} else {
			slog.WarnContext(ctx, "rta: subscribe response sub_id invalide",
				"err", err, "raw", string(msg[3]))
		}
	}

	c.pendingMu.Lock()
	ps, ok := c.pending[seq]
	delete(c.pending, seq)
	c.pendingMu.Unlock()

	if !ok {
		slog.WarnContext(ctx, "rta: subscribe response pour seq inconnue", "seq", seq)
		return
	}

	// Status acceptés comme une souscription valide.
	//
	//   0  → succès standard documenté.
	//   2  → également un succès. Microsoft ne publie pas la sémantique exacte
	//        des codes RTA, mais on observe empiriquement que Xbox Live renvoie
	//        status=2 sur les subscribes à userpresence.xboxlive.com/users/xuid(X)/titles/Y
	//        pour signaler une souscription enregistrée côté serveur (probablement
	//        "AlreadySubscribed" / "ResubscribeOK"). Avant ce changement, status=2
	//        était traité comme un refus, la sub n'était pas stockée dans c.subs,
	//        et tous les events de présence qui arrivaient ensuite étaient jetés
	//        silencieusement avec un log DEBUG "event pour sub_id inconnu".
	//        Cf. rta_subscribe_status_test.go pour la régression.
	if status != 0 && status != 2 {
		slog.WarnContext(ctx, "rta: subscribe refusé",
			"xuid", ps.xuid,
			"status", status,
			"sub_id", subID,
		)
		if status == 3 {
			// Status=3 sur userpresence.xboxlive.com a deux causes possibles :
			//  1. Token XSTS expiré (refus global de tous les subs)
			//  2. Privacy refusée pour ce XUID précis (le compte ne peut pas voir
			//     la rich presence de cet utilisateur — pas friend, privacy strict, etc.)
			//
			// On diffère la décision de 2s : si au moins un autre subscribe réussit
			// pendant ce délai, c'est forcément (1) faux → c'est de la privacy
			// individuelle, on garde la connexion ouverte. Si AUCUN succès après 2s,
			// c'est bien une auth expirée, on signale et on ferme pour reconnect.
			go c.evaluateStatus3AfterGrace(ctx)
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
	if err := json.Unmarshal(msg[1], &subID); err != nil {
		slog.WarnContext(ctx, "rta: event sub_id invalide", "err", err)
		return
	}

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

	// Si le parser n'a extrait aucun PresenceDetail (event "vide"), Xbox envoie
	// une notif sans presenceDetails actionnables (keep-alive, user pas dans le
	// titre, format inattendu sur /titles/<TID>) — log INFO court pour rester
	// scannable, puis DEBUG avec raw pour diagnostic ciblé (activable via
	// LEVELUP_LOGS_FILE_LEVEL=debug).
	if event.PresenceDetail == nil {
		slog.InfoContext(ctx, "rta: event de présence (payload sans titre actif)",
			"xuid", event.XUID,
			"state", event.PresenceState,
			"title", titleFromEvent(event),
		)
		slog.DebugContext(ctx, "rta: payload brut (no PresenceDetail)",
			"xuid", event.XUID,
			"raw", string(raw),
		)
	} else {
		slog.InfoContext(ctx, "rta: event de présence",
			"xuid", event.XUID,
			"state", event.PresenceState,
			"title", titleFromEvent(event),
		)
	}

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

// hashSubIDString convertit un sub_id string (renvoyé par Xbox pour
// certains endpoints comme /richpresence) en int stable utilisable
// comme clé dans c.subs. Utilise FNV-1a 32-bit qui a une distribution
// uniforme et est rapide. Collisions extrêmement improbables vu le
// petit nombre de subscriptions actives.
func hashSubIDString(s string) int {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return int(h)
}
