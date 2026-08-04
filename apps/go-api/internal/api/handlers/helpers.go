// Package handlers contient les handlers HTTP des endpoints P0.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
)

// Clés JSON partagées entre handlers (params actor logs, response error envelopes).
// Externalisées pour éviter les doublons goconst dans les helpers et middlewares.
const (
	jsonKeyStatus    = "status"
	jsonKeyCount     = "count"
	jsonKeyCode      = "code"
	jsonKeyMessage   = "message"
	jsonKeyRetryable = "retryable"

	// Valeurs courantes
	jsonBoolTrueStr = "true"

	// Codes d'erreur partagés (MSAL device-code flow).
	errCodeMSALAcquire = "msal_acquire_error"

	// Modes auth.
	authModeXbox = "xbox"

	// headerRetryAfter : nom de l'en-tête HTTP 503 « réessayer dans N s », émis par
	// tous les handlers qui rencontrent une base occupée (dblease.ErrDBLocked).
	// Posée au passage du ratchet goconst (10e occurrence du littéral), puis
	// GÉNÉRALISÉE à l'ensemble du paquet le 2026-08-03 : plus aucun littéral
	// "Retry-After" ne subsiste dans le code de production de handlers, et
	// TestNoRetryAfterLiteralInHandlers (no_retry_after_literal_test.go) interdit
	// sa réapparition.
	// Les tests gardent le littéral : ils lisent l'en-tête par son NOM HTTP réel,
	// ce qui est précisément ce qu'on veut vérifier côté client.
	headerRetryAfter = "Retry-After"
)

// errDBBusy construit la réponse canonique « base occupée » : 503 `db_busy` +
// en-tête `Retry-After: 5`, via huma.ErrorWithHeaders.
//
// Source unique du triplet (statut, code, message) : l'enveloppe était recopiée
// à l'identique dans 7 handlers write (notifications, media_likes, media_delete,
// match_favorite, match_exclusion, engagement, prestige_squads) — un client qui
// distingue les 503 sur le `code` dépend de leur stricte égalité. À appeler dès
// qu'une erreur wrappe dblease.ErrDBLocked.
//
// Garde-rail (CLAUDE.md règle 6) : TestNoDBBusyLiteralInHandlers
// (no_db_busy_envelope_test.go) interdit la réapparition du littéral "db_busy".
// N'englobe PAS le code distinct "home_page_db_busy" (home.go), qui porte une
// sémantique de dégradation de page, pas un refus d'écriture.
func errDBBusy() error {
	return huma.ErrorWithHeaders(
		humacore.NewError(http.StatusServiceUnavailable, "db_busy",
			"database is currently busy, please retry"),
		http.Header{headerRetryAfter: []string{"5"}},
	)
}

// writeJSON sérialise v en JSON et l'écrit dans w avec le code HTTP donné.
// Utilise json.Marshal (buffer complet) avant d'écrire les headers : si la
// sérialisation échoue (ex. float64 NaN/Inf), renvoie 500 avec un corps JSON
// valide au lieu d'un 200 avec corps vide (ancien comportement silencieux).
// Passe humacore.SanitizeFloatsForJSON en amont pour neutraliser les NaN/Inf
// (logique partagée avec le format Huma byte-identique) ; chaque neutralisation
// est loggée avec son path de struct pour identifier la source.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	sanitized, nanPaths := humacore.SanitizeFloatsForJSON(v)
	if len(nanPaths) > 0 {
		slog.Warn("writeJSON: NaN/Inf neutralized", "paths", nanPaths, "count", len(nanPaths))
	}
	body, err := json.Marshal(sanitized)
	if err != nil {
		slog.Error("writeJSON: marshal failed", "err", err)
		// Réponse d'erreur inline — pas via writeError pour éviter la récursion.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"encode_error","message":"erreur de sérialisation","retryable":true}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n"))
}

// writeJSONCached sérialise v, pose un ETag SHA-256 et retourne 304 si le client est à jour.
// À utiliser sur les endpoints GET dont les données changent peu entre deux syncs.
//
// tous les callers passent 200/OK mais la signature reste configurable.
//
//nolint:unparam // status paramétré pour cohérence avec writeJSON ; aujourd'hui
func writeJSONCached(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	sanitized, nanPaths := humacore.SanitizeFloatsForJSON(v)
	if len(nanPaths) > 0 {
		slog.WarnContext(r.Context(), "writeJSONCached: NaN/Inf neutralized", "paths", nanPaths, "count", len(nanPaths))
	}
	body, err := json.Marshal(sanitized)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "encode_error", "erreur de sérialisation")
		return
	}
	sum := sha256.Sum256(body)
	etag := fmt.Sprintf(`"%x"`, sum[:8])
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError écrit une réponse d'erreur JSON standardisée et **logge l'erreur
// côté serveur** selon le statut HTTP :
//   - 5xx → slog.ErrorContext (erreur serveur, anormal — toujours visible)
//   - 401/403/422 → slog.WarnContext (auth/validation refusée — tracé pour audit)
//   - autres 4xx → slog.DebugContext (404 / bad request — volume, log à la demande)
//
// L'objectif est qu'aucune erreur renvoyée au client n'échappe à un trace serveur.
// Le ctx fait remonter event_id / request_id auto-injectés par ContextHandler.
func writeError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	// Le détail (souvent err.Error()) est toujours logué côté serveur, corrélable
	// via event_id/request_id. Mais sur une 5xx on ne l'expose JAMAIS au client :
	// message générique stable, le `code` (non sensible) porte la sémantique
	// (revue 2026-06-02 — fuite d'info interne sur les réponses 500).
	logErrorResponse(ctx, status, code, message, nil)
	clientMessage := message
	if status >= 500 {
		clientMessage = "internal error"
	}
	writeJSON(w, status, map[string]interface{}{
		"code":      code,
		"message":   clientMessage,
		"retryable": status >= 500,
	})
}

// httpError est un wrapper de net/http.Error qui logge l'erreur côté serveur
// (mêmes règles de niveau que writeError) avant de répondre en text/plain.
//
// À utiliser pour les endpoints qui servent du contenu non-JSON (assets, fichiers
// statiques, redirects) — sinon préférer writeError qui renvoie un body JSON
// standardisé.
func httpError(ctx context.Context, w http.ResponseWriter, message string, status int) {
	logErrorResponse(ctx, status, "", message, nil)
	clientMessage := message
	if status >= 500 {
		// Idem writeError : pas de détail interne au client sur une 5xx.
		clientMessage = "internal error"
	}
	http.Error(w, clientMessage, status)
}

// logErrorResponse centralise la décision de niveau de log selon le statut HTTP.
// Séparé de writeError pour permettre la réutilisation (writeServerError, et
// futurs helpers d'erreur).
func logErrorResponse(ctx context.Context, status int, code, message string, err error) {
	attrs := []any{jsonKeyStatus, status, jsonKeyCode, code, jsonKeyMessage, message}
	if err != nil {
		attrs = append(attrs, "err", err)
	}
	switch {
	case status >= 500:
		slog.ErrorContext(ctx, "http: erreur réponse", attrs...)
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusUnprocessableEntity:
		slog.WarnContext(ctx, "http: refus", attrs...)
	case status >= 400:
		slog.DebugContext(ctx, "http: client error", attrs...)
	}
}
