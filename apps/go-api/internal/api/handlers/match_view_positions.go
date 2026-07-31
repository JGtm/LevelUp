// Package handlers — match_view_positions.go : endpoint
// GET .../matches/{match_id}/positions.
//
// Sérialise les positions joueurs keyframe v3 (décodées du film, match-level —
// §N de .ai/RESEARCH_THEATER_RE.md) en JSON camelCase. On NE sérialise PAS
// positions.PlayerPosition brut (struct sans tags json → clés PascalCase non
// consommables par le front) : un DTO dédié porte les tags.
//
// LIMITE (honnête, §N) : v1 = MATCH-LEVEL — pas d'attribution xuid (la
// delta-compression bloque l'index par joueur). team est best-effort
// (-1 = inconnu, sérialisé tel quel pour permettre une heatmap globale).
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/games"
)

// matchPositionDTO est la projection JSON camelCase d'une positions.PlayerPosition.
// team vaut -1 (TeamUnknown) quand aucun clustering spatial net ne permet de
// l'attribuer ; 0/1 sinon (best-effort, non garanti).
type matchPositionDTO struct {
	TimeMS int     `json:"timeMs"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Team   int     `json:"team"`
}

// toMatchPositionsDTO mappe []positions.PlayerPosition → []matchPositionDTO.
// Retourne un slice non-nil vide pour sérialiser `[]` plutôt que `null`.
func toMatchPositionsDTO(pos []positions.PlayerPosition) []matchPositionDTO {
	out := make([]matchPositionDTO, 0, len(pos))
	for _, p := range pos {
		out = append(out, matchPositionDTO{
			TimeMS: p.TimeMS,
			X:      p.X,
			Y:      p.Y,
			Z:      p.Z,
			Team:   p.Team,
		})
	}
	return out
}

// GetMatchPositions retourne les positions joueurs keyframe v3 d'un match.
// GET /api/v1/players/{player_slug}/matches/{match_id}/positions
//
// Capability gating : si le titre n'expose pas le film (repo non câblé ou table
// absente), le service remonte games.ErrCapabilityNotSupported → 503
// capability_not_supported (dégradation gracieuse, pas une 500).
func (h *MatchViewHandler) GetMatchPositions(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	pos, err := svc.GetMatchPositions(r.Context(), matchID)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			writeError(r.Context(), w, http.StatusServiceUnavailable,
				"capability_not_supported", "positions indisponibles pour ce titre")
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "positions_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toMatchPositionsDTO(pos))
}
