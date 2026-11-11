// Package middleware — desktop_gate.go : l'entrée de la coquille desktop (ADR 0045 D-3).
//
// Le serveur du mode desktop écoute sur la boucle locale, mais tout processus du poste (et toute
// page web ouverte dans un navigateur, par rebond DNS) peut lui parler. Trois contrôles ferment
// cette porte :
//
//   - Host strict : l'en-tête Host doit être exactement l'autorité liée (127.0.0.1:<port>), sans
//     exception (sondes comprises). Un autre nom, un autre port ou une adresse IPv6 sont refusés
//     avec 421 et journalisés : c'est ce qui coupe le rebond DNS.
//   - Entrée par secret : GET /desktop/enter?secret=<aléa> compare le secret du lanceur en temps
//     constant et pose un cookie lié au port ; la coquille le charge une fois, au démarrage. Le secret
//     est À USAGE UNIQUE par lancement : l'URL d'entrée entre dans l'historique de la WebView, un secret
//     rejouable y survivrait vivant ; après une entrée réussie il est refusé (403), le cookie porte la
//     session.
//   - Cookie exigé : hors /desktop/enter et /health (la sonde du lanceur, sans donnée), toute
//     requête sans le cookie valide répond 401.
//   - Une seule exception d'entrée, sous la capacité de connexion par le navigateur (AllowBrowserCallback) :
//     le retour de Microsoft arrive dans le navigateur par défaut, sans cookie, sur http://localhost:<port>.
//     GET /auth/xbox/callback, et lui seul, passe sans cookie ET avec un Host « localhost » ou
//     « 127.0.0.1 » du port lié ; il est protégé par le state à usage unique de la tentative. La porte
//     de démarrage (StrictHost), qui reste devant le routeur, applique la même exemption.
//
// Le cookie n'est pas une session : sa valeur est une échéance signée (HMAC-SHA256 avec le secret de
// session du fichier, lié au port), vérifiée sans état, valable 30 jours et renouvelée à chaque
// entrée. Son nom porte le port (levelup_session_<port>) : les cookies ne sont pas cloisonnés par
// port, et deux serveurs de boucle locale (cette application et un serveur de développement) ne
// doivent ni s'envoyer ni s'écraser le leur.
package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// DesktopUnauthorizedCode : code d'erreur du 401 d'une requête sans cookie valide ; l'interface
	// le reconnaît (apps/web/src/app/serverStartup.ts) pour dire de rouvrir l'application.
	DesktopUnauthorizedCode = "desktop_unauthorized"
	// DesktopEnterPath : la route d'entrée de la coquille.
	DesktopEnterPath = "/desktop/enter"
	// DesktopShutdownPath : la route d'arrêt de secours (POST), exemptée du cookie : elle a ses propres
	// portes (LoopbackOnly et secret d'en-tête en temps constant) et la coquille ne peut pas fabriquer
	// le cookie, signé et HttpOnly. Le Host strict s'y applique comme ailleurs.
	DesktopShutdownPath = "/api/v1/internal/shutdown"
	// desktopHealthPath : la sonde du lanceur, ouverte sans cookie (aucune donnée).
	desktopHealthPath = "/health"
	// DesktopBrowserCallbackPath : le retour de Microsoft de la connexion par le navigateur, déclaré
	// côté Azure (http://localhost/auth/xbox/callback ; tout port est admis sur localhost).
	DesktopBrowserCallbackPath = "/auth/xbox/callback"
	// DesktopCookieTTL : durée de validité du cookie, posée à l'entrée de chaque lancement.
	DesktopCookieTTL = 30 * 24 * time.Hour
	// desktopCookiePrefix : début du nom du cookie, suivi du port lié.
	desktopCookiePrefix = "levelup_session_"
	// desktopCookieDomain : séparation des usages de la clé HMAC (jamais réutilisée par un autre cookie).
	desktopCookieDomain = "levelup-desktop-entry/v1"
	// desktopCookieSameSite : Lax, jamais Strict. La navigation vers /desktop/enter est lancée depuis la
	// page embarquée de la coquille (origine levelup-shell://, un autre site pour Chromium) : la chaîne
	// 302 vers « / » est une navigation de premier niveau d'initiateur inter-sites, sur laquelle Chromium
	// n'envoie PAS un cookie Strict (« / » arriverait sans cookie : 401). Lax est envoyé sur les
	// navigations de premier niveau et jamais sur les sous-requêtes inter-sites ; les mutations restent
	// protégées par le CSRF d'origine exacte, le Host strict et la boucle locale.
	desktopCookieSameSite = http.SameSiteLaxMode
	// desktopEnterSecretParam : paramètre de requête qui porte le secret du lanceur.
	desktopEnterSecretParam = "secret"
)

// DesktopGate porte la configuration de l'entrée de la coquille : l'autorité Host attendue, le
// port (nom et signature du cookie), la clé de signature et le secret du lanceur.
type DesktopGate struct {
	authority    string
	port         int
	key          []byte
	secretDigest [sha256.Size]byte
	configured   bool
	// entered : le secret a été présenté avec succès une fois ; toute présentation suivante est refusée.
	entered atomic.Bool
	now     func() time.Time
	// callbackAuthority : l'autorité « localhost:<port> » du retour de la connexion par le navigateur ;
	// vide tant que AllowBrowserCallback n'a pas ouvert cette route.
	callbackAuthority string
}

// NewDesktopGate construit l'entrée pour l'autorité authority (« 127.0.0.1:<port> ») et le port
// port. cookieKey signe le cookie (le secret de session) ; launcherSecret est le secret que la
// coquille présente à /desktop/enter (LEVELUP_DESKTOP_SECRET). Un secret de lanceur vide laisse
// l'entrée fermée : /desktop/enter répond 404 et aucun cookie ne s'obtient.
func NewDesktopGate(authority string, port int, cookieKey, launcherSecret string) *DesktopGate {
	return &DesktopGate{
		authority:    authority,
		port:         port,
		key:          []byte(cookieKey),
		secretDigest: sha256.Sum256([]byte(launcherSecret)),
		configured:   launcherSecret != "" && cookieKey != "",
		now:          time.Now,
	}
}

// AllowBrowserCallback ouvre GET /auth/xbox/callback sans cookie, avec le Host « localhost:<port> » (ou
// l'autorité numérique) du port lié, et lui seul : tout autre chemin, toute autre méthode, tout autre
// Host gardent les contrôles d'entrée. À appeler sous la capacité LoginViaBrowser seulement.
func (g *DesktopGate) AllowBrowserCallback() *DesktopGate {
	g.callbackAuthority = "localhost:" + strconv.Itoa(g.port)
	return g
}

// isBrowserCallback dit si la requête est le retour de la connexion par le navigateur : route ouverte,
// méthode GET, chemin exact (jamais ambigu), Host de la boucle locale du port lié.
func (g *DesktopGate) isBrowserCallback(r *http.Request) bool {
	return g.callbackAuthority != "" && r.Method == http.MethodGet && !AmbiguousPath(r) &&
		r.URL.Path == DesktopBrowserCallbackPath && (r.Host == g.callbackAuthority || r.Host == g.authority)
}

// DesktopCookieName rend le nom du cookie d'un serveur lié au port port.
func DesktopCookieName(port int) string {
	return desktopCookiePrefix + strconv.Itoa(port)
}

// sign rend la signature de l'échéance exp (secondes Unix) pour ce port.
func (g *DesktopGate) sign(exp int64) string {
	mac := hmac.New(sha256.New, g.key)
	mac.Write([]byte(desktopCookieDomain + "|" + strconv.Itoa(g.port) + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// cookieValue rend la valeur d'un cookie qui expire à exp : « <échéance>.<signature> ».
func (g *DesktopGate) cookieValue(exp time.Time) string {
	unix := exp.Unix()
	return strconv.FormatInt(unix, 10) + "." + g.sign(unix)
}

// validCookie dit si la requête porte un cookie de la coquille signé par ce serveur et non expiré.
func (g *DesktopGate) validCookie(r *http.Request) bool {
	c, err := r.Cookie(DesktopCookieName(g.port))
	if err != nil {
		return false
	}
	expStr, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return false
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || g.now().Unix() >= exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(g.sign(exp)))
}

// Middleware applique Host strict puis le cookie exigé. À monter en tête de chaîne, juste après
// PreserveRemoteAddr : rien d'autre (CORS, session, routeur) ne voit une requête refusée ici.
func (g *DesktopGate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.isBrowserCallback(r) {
			next.ServeHTTP(w, r)
			return
		}
		if refuseForeignHost(w, r, g.authority) {
			return
		}
		// Un chemin ambigu (cf. AmbiguousPath) n'est jamais ouvert : il passe par le cookie.
		if !AmbiguousPath(r) && (r.URL.Path == DesktopEnterPath || r.URL.Path == desktopHealthPath ||
			(r.Method == http.MethodPost && r.URL.Path == DesktopShutdownPath)) {
			next.ServeHTTP(w, r)
			return
		}
		if !g.validCookie(r) {
			slog.InfoContext(r.Context(), "middleware.desktop_gate: requête sans cookie de la coquille",
				"method", r.Method, "path", r.URL.Path)
			writeGateError(w, http.StatusUnauthorized, DesktopUnauthorizedCode, "entrée de la coquille requise")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// refuseForeignHost répond 421 (journalisé) et rend vrai quand l'en-tête Host n'est pas exactement
// l'autorité attendue.
func refuseForeignHost(w http.ResponseWriter, r *http.Request, authority string) bool {
	if r.Host == authority {
		return false
	}
	slog.WarnContext(r.Context(), "middleware.desktop_gate: Host refusé",
		"host", r.Host, "attendu", authority, "path", r.URL.Path)
	writeGateError(w, http.StatusMisdirectedRequest, "desktop_host_refused", "Host non autorisé")
	return true
}

// StrictHost n'accepte que l'autorité exacte du garde dans l'en-tête Host (421 sinon). Elle enveloppe
// la porte de démarrage du mode desktop, qui répond avant que le routeur (et son garde) existe : sans
// elle, la page de démarrage serait servie à tout nom d'hôte (rebond DNS). Aucun cookie n'y est exigé :
// la coquille charge cette page avant de l'avoir.
//
// La porte reste devant le routeur pour toute la vie du serveur : ce contrôle doit donc ouvrir exactement
// ce que Middleware ouvre, à savoir le retour de la connexion par le navigateur (isBrowserCallback),
// qui arrive avec un Host « localhost:<port> ». Sans cette exemption, le retour est refusé ici avant
// d'atteindre le garde du routeur, et la connexion par le navigateur n'aboutit jamais.
func (g *DesktopGate) StrictHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.isBrowserCallback(r) && refuseForeignHost(w, r, g.authority) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// EnterHandler sert GET /desktop/enter?secret=… : 404 si l'entrée n'est pas configurée, 403 si le
// secret manque, est faux (comparé en temps constant) ou a déjà servi (usage unique par lancement :
// l'URL d'entrée reste dans l'historique de la WebView), sinon pose le cookie (échéance de 30 jours)
// et redirige vers « / ». Le secret n'est jamais journalisé.
func (g *DesktopGate) EnterHandler(w http.ResponseWriter, r *http.Request) {
	if !g.configured {
		http.NotFound(w, r)
		return
	}
	got := sha256.Sum256([]byte(r.URL.Query().Get(desktopEnterSecretParam)))
	if subtle.ConstantTimeCompare(got[:], g.secretDigest[:]) != 1 {
		slog.WarnContext(r.Context(), "middleware.desktop_gate: secret d'entrée refusé")
		writeGateError(w, http.StatusForbidden, "forbidden", "secret d'entrée refusé")
		return
	}
	// Un seul gagnant, même sous deux requêtes simultanées.
	if !g.entered.CompareAndSwap(false, true) {
		slog.WarnContext(r.Context(), "middleware.desktop_gate: secret d'entrée déjà consommé")
		writeGateError(w, http.StatusForbidden, "forbidden", "secret d'entrée déjà consommé")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     DesktopCookieName(g.port),
		Value:    g.cookieValue(g.now().Add(DesktopCookieTTL)),
		Path:     "/",
		HttpOnly: true,
		SameSite: desktopCookieSameSite,
		MaxAge:   int(DesktopCookieTTL / time.Second),
	})
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, "/", http.StatusFound)
}

// writeGateError écrit un refus du garde au format d'erreur de l'API ({code, message, retryable}) :
// rien n'y changera en rejouant la requête.
func writeGateError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{"code": code, "message": message, "retryable": false}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Debug("middleware.desktop_gate: écriture du refus interrompue", "err", err)
	}
}
