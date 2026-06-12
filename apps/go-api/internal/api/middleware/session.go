// Package middleware — session.go : middleware d'injection de session dans le context HTTP.
//
// Le middleware lit le cookie "levelup_session", valide la signature HMAC,
// charge (ou crée) la SessionData, et l'injecte dans context.Context.
// La session est sauvegardée et le cookie rafraîchi en fin de requête.
package middleware

import (
	"context"
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
			// Le cookie est posé AVANT le handler (header) : l'ID de session ne change
			// pas même si le handler enrichit la session (login, OAuth state…).
			setCookie(w, store, sess, policy.Secure(r))
			ctx := context.WithValue(r.Context(), sessionKey{}, sess)
			if sess.HaloTokens != nil {
				xuid := ""
				if sess.LinkedHaloIdentity != nil {
					xuid = sess.LinkedHaloIdentity.XUID
				}
				ctx = ctxkeys.WithHaloAuth(ctx, sess.HaloTokens, xuid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
			// Persistance APRÈS le handler : on n'écrit sur disque que pour une session
			// déjà persistée (TTL glissant) ou devenue significative pendant la requête
			// (login, OAuth, préférence…). Une session anonyme vierge n'est jamais
			// persistée → plus de spam d'un fichier par requête sans cookie
			// (bots/sondes/assets) dans data/sessions/.
			if loaded || sess.IsMeaningful() {
				_ = store.Touch(sess)
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
			if sess := store.Load(sessionID); sess != nil {
				return sess, true
			}
		}
	}
	return store.New(), false
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
