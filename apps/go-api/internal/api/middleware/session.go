// Package middleware — session.go : middleware d'injection de session dans le context HTTP.
//
// Le middleware lit le cookie "levelup_session", valide la signature HMAC,
// charge (ou crée) la SessionData, et l'injecte dans context.Context.
// La session est sauvegardée et le cookie rafraîchi en fin de requête.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// sessionKey est la clé privée pour stocker la session dans context.Context.
type sessionKey struct{}

// WithSession renvoie un middleware qui injecte une SessionData dans le contexte.
// Si le cookie est absent ou invalide, une nouvelle session est créée.
//
// policy décide du flag Secure du cookie PAR REQUÊTE (cf. SecureCookiePolicy) :
// un même binaire sert ainsi en HTTP local (cookie non-Secure → round-trip OK) et
// en HTTPS prod (cookie Secure) sans reconfiguration.
func WithSession(store *session.Store, policy SecureCookiePolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, loaded := loadOrCreate(r, store)
			// Le cookie est posé via un wrapper, au moment de l'écriture de la réponse,
			// et UNIQUEMENT si la session mérite d'être conservée (déjà existante, ou
			// devenue significative pendant la requête : login, OAuth state, préférence…).
			// CRUCIAL : une requête anonyme jetable (bot, sonde, asset, ou simple
			// navigation non connectée) ne pose donc AUCUN cookie → elle ne peut plus
			// ÉCRASER le cookie d'auth du navigateur (bug : utilisateur déconnecté car
			// son cookie pointait soudain sur une session anonyme).
			sw := &sessionResponseWriter{
				ResponseWriter: w,
				store:          store,
				sess:           sess,
				loaded:         loaded,
				secure:         policy.Secure(r),
			}
			ctx := context.WithValue(r.Context(), sessionKey{}, sess)
			if sess.HaloTokens != nil {
				xuid := ""
				if sess.LinkedHaloIdentity != nil {
					xuid = sess.LinkedHaloIdentity.XUID
				}
				ctx = ctxkeys.WithHaloAuth(ctx, sess.HaloTokens, xuid)
			}
			next.ServeHTTP(sw, r.WithContext(ctx))
			sw.commitCookie() // au cas où le handler n'a rien écrit
			// Persistance disque : seulement pour une session déjà persistée (TTL
			// glissant) ou devenue significative. Une session anonyme vierge n'est
			// jamais écrite → plus de spam dans data/sessions/.
			if loaded || sess.IsMeaningful() {
				if err := store.Touch(sess); err != nil {
					slog.ErrorContext(r.Context(), "session touch failed", "err", err)
				}
			}
		})
	}
}

// GetSession extrait la SessionData depuis le context. Retourne nil si absente.
func GetSession(ctx context.Context) *domain.SessionData {
	v := ctx.Value(sessionKey{})
	if v == nil {
		return nil
	}
	return v.(*domain.SessionData)
}

// SessionPlayerSlug retourne le slug du joueur courant de la session (le
// VIEWER), ou "" si aucune session ou aucun joueur courant.
//
// Point unique de résolution de l'identité du viewer : les likes par viewer
// (lecture ET écriture), la suppression de média et le scoping des surfaces
// sociales doivent tous répondre à la même question « qui regarde ? ». Avant sa
// centralisation, chaque call-site refaisait `sess != nil && sess.CurrentPlayerSlug
// != nil` — trois copies divergeaient déjà (CLAUDE.md règle n°6).
//
// ATTENTION : le viewer n'est PAS le propriétaire de la page. Sur
// /players/{owner}/media, `player_slug` désigne la galerie consultée tandis que
// le viewer est l'utilisateur connecté — ils diffèrent dès qu'on regarde les
// médias d'un coéquipier.
func SessionPlayerSlug(ctx context.Context) string {
	sess := GetSession(ctx)
	if sess == nil || sess.CurrentPlayerSlug == nil {
		return ""
	}
	return *sess.CurrentPlayerSlug
}

// InjectSession returns a context with the given SessionData attached under
// the same private key WithSession uses. Intended for tests of handlers that
// want to bypass the cookie-based middleware and assert behaviour against a
// pre-built session.
func InjectSession(ctx context.Context, sess *domain.SessionData) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}

// loadOrCreate charge la session depuis le cookie ou en crée une nouvelle.
// Le second retour vaut true si la session a été chargée depuis le disque (elle
// existe déjà), false si c'est une session neuve créée pour cette requête.
func loadOrCreate(r *http.Request, store *session.Store) (*domain.SessionData, bool) {
	c, err := r.Cookie(session.CookieName)
	if err == nil && c.Value != "" {
		if sessionID := store.UnsignCookie(c.Value); sessionID != "" {
			if sess := store.Load(r.Context(), sessionID); sess != nil {
				return sess, true
			}
		}
	}
	return store.New(), false
}

// sessionResponseWriter diffère la pose du cookie de session jusqu'à la première
// écriture de la réponse, et ne le pose QUE si la session mérite d'être conservée
// (déjà existante sur disque, ou devenue significative pendant la requête). Une
// requête anonyme jetable ne pose ainsi aucun cookie et ne peut pas écraser le
// cookie d'auth du navigateur.
type sessionResponseWriter struct {
	http.ResponseWriter
	store       *session.Store
	sess        *domain.SessionData
	loaded      bool
	secure      bool
	wroteHeader bool
}

// commitCookie pose le cookie (une seule fois, avant l'écriture du statut) si la
// session est significative ou déjà persistée. Idempotent.
func (w *sessionResponseWriter) commitCookie() {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if w.loaded || w.sess.IsMeaningful() {
		setCookie(w.ResponseWriter, w.store, w.sess, w.secure)
	}
}

func (w *sessionResponseWriter) WriteHeader(status int) {
	w.commitCookie()
	w.ResponseWriter.WriteHeader(status)
}

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	w.commitCookie()
	return w.ResponseWriter.Write(b)
}

// Flush propage le Flush sous-jacent (compression, streaming) si supporté.
func (w *sessionResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		w.commitCookie()
		f.Flush()
	}
}

// setCookie pose le cookie de session signé sur la réponse.
func setCookie(w http.ResponseWriter, store *session.Store, sess *domain.SessionData, secure bool) {
	signed := store.SignCookie(sess.SessionID)
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   int(session.DefaultTTL / time.Second),
	})
}
