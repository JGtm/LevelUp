package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/ctxkeys"
)

// replay_demo_gate.go — LA GARDE DES ROUTES DE REJEU EN DÉMO.
//
// Hors démo, les routes de rejeu ne portent aucun filtre de transport : elles sont servies
// comme les autres routes joueur, sous les contrôles d'accès du groupe (propriété du joueur,
// participation au match). Seule la démo publique en pose un, parce que n'y est servi que ce
// qui a été figé.

// DemoReplayAllowlist dit, en DÉMO, si un match a un rejeu figé servi (index des rejeux de la
// démo, cf. service.DemoReplayAllowlist).
type DemoReplayAllowlist interface {
	Allows(ctx context.Context, titleSlug, matchID string) bool
}

// ReplayGate rend la garde des routes de rejeu.
//
//   - Hors démo (`demo` nil) : aucun filtre, la requête passe telle quelle.
//   - En DÉMO (décision D-2 du plan des recommandations du 2026-10-09) : le rejeu est servi
//     à tout visiteur, mais SEULEMENT pour les matchs figés de la démo ; tout autre match
//     reçoit un 404 `replay_not_available`.
//
// Le match se lit dans le chemin (`{match_id}`) et, à défaut, dans le CORPS JSON d'un POST
// (`match_id`, route de la séquence d'une mort) — voir [replayMatchID].
func ReplayGate(demo DemoReplayAllowlist) func(http.Handler) http.Handler {
	if demo == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			matchID := replayMatchID(r)
			if matchID == "" || !demo.Allows(r.Context(), ctxkeys.TitleSlug(r.Context()), matchID) {
				writeError(r.Context(), w, http.StatusNotFound, "replay_not_available",
					"the 2D replay is served in the demo for its frozen matches only")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// replayGateMaxBody borne le corps que la garde accepte de lire pour y trouver le match : un
// corps de séquence (match_id, victim_xuid, time_ms) tient en quelques dizaines d'octets.
const replayGateMaxBody = 4 << 10

// replayMatchID rend le match visé par la requête : le paramètre de chemin `match_id`, sinon
// le champ `match_id` du corps JSON d'un POST. Le corps est RESTITUÉ au handler, octet pour
// octet (la garde et le handler lisent la même valeur). "" si le match est introuvable : corps
// absent, illisible, au-delà de [replayGateMaxBody], ou `match_id` qui n'est pas du texte.
func replayMatchID(r *http.Request) string {
	if id := chi.URLParam(r, "match_id"); id != "" {
		return id
	}
	if r.Method != http.MethodPost || r.Body == nil || r.Body == http.NoBody {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, replayGateMaxBody+1))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if len(raw) > replayGateMaxBody {
		return ""
	}
	var body struct {
		MatchID string `json:"match_id"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	return body.MatchID
}
