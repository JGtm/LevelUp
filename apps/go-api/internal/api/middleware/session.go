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

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// sessionKey est la clé privée pour stocker la session dans context.Context.
type sessionKey struct{}

// WithSession renvoie un middleware qui injecte une SessionData dans le contexte.
// Si le cookie est absent ou invalide, une nouvelle session est créée.
func WithSession(store *session.Store, isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := loadOrCreate(r, store)
			_ = store.Touch(sess)
			setCookie(w, store, sess, isProduction)
			ctx := context.WithValue(r.Context(), sessionKey{}, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
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

// loadOrCreate charge la session depuis le cookie ou en crée une nouvelle.
func loadOrCreate(r *http.Request, store *session.Store) *domain.SessionData {
	c, err := r.Cookie(session.CookieName)
	if err == nil && c.Value != "" {
		if sessionID := store.UnsignCookie(c.Value); sessionID != "" {
			if sess := store.Load(sessionID); sess != nil {
				return sess
			}
		}
	}
	return store.New()
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
