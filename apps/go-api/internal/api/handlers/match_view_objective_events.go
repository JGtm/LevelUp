// Package handlers — match_view_objective_events.go : endpoint
// GET .../matches/{match_id}/objective-events.
//
// Sérialise la timeline objectif v3 (events mode-agnostiques décodés du film) en
// JSON camelCase. On NE sérialise PAS domain.ObjectiveEvent brut (struct sans
// tags json → clés PascalCase non consommables par le front) : un DTO dédié
// porte les tags et omet les champs non utiles à la timeline (objective_id,
// details).
package handlers

import (
	"context"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// objectiveEventDTO est la projection JSON camelCase d'un domain.ObjectiveEvent.
// Les pointeurs (timeMs, teamId, value) sont omitempty : un champ NULL en DB
// (team unreliable, value absente) est absent du JSON plutôt que rendu à 0.
type objectiveEventDTO struct {
	MatchID       string                    `json:"matchId"`
	Seq           int                       `json:"seq"`
	TimeMS        *int                      `json:"timeMs,omitempty"`
	ObjectiveType string                    `json:"objectiveType"`
	EventType     string                    `json:"eventType"`
	TeamID        *int                      `json:"teamId,omitempty"`
	Value         *int                      `json:"value,omitempty"`
	Source        string                    `json:"source"`
	Confidence    string                    `json:"confidence"`
	Players       []objectiveEventPlayerDTO `json:"players"`
}

// objectiveEventPlayerDTO est la projection JSON camelCase d'un joueur impliqué.
type objectiveEventPlayerDTO struct {
	XUID string `json:"xuid"`
	Role string `json:"role"`
}

// toObjectiveEventsDTO mappe []domain.ObjectiveEvent → []objectiveEventDTO.
// Retourne un slice non-nil vide pour sérialiser `[]` plutôt que `null`.
func toObjectiveEventsDTO(events []domain.ObjectiveEvent) []objectiveEventDTO {
	out := make([]objectiveEventDTO, 0, len(events))
	for _, ev := range events {
		players := make([]objectiveEventPlayerDTO, 0, len(ev.Players))
		for _, p := range ev.Players {
			players = append(players, objectiveEventPlayerDTO{XUID: p.XUID, Role: p.Role})
		}
		out = append(out, objectiveEventDTO{
			MatchID:       ev.MatchID,
			Seq:           ev.Seq,
			TimeMS:        ev.TimeMS,
			ObjectiveType: ev.ObjectiveType,
			EventType:     ev.EventType,
			TeamID:        ev.TeamID,
			Value:         ev.Value,
			Source:        ev.Source,
			Confidence:    ev.Confidence,
			Players:       players,
		})
	}
	return out
}

// objectiveEventsInput : {player_slug} parent + {match_id}.
type objectiveEventsInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
}

type objectiveEventsOutput struct{ Body []objectiveEventDTO }

// handleGetObjectiveEvents retourne la timeline objectif v3 d'un match.
// GET /api/v1/players/{player_slug}/matches/{match_id}/objective-events
//
// Capability gating : si le titre n'expose pas la timeline objectif (repo non
// câblé ou tables absentes), le service remonte games.ErrCapabilityNotSupported
// → 503 capability_not_supported (dégradation gracieuse, pas une 500).
func (h *MatchViewHandler) handleGetObjectiveEvents(
	ctx context.Context, in *objectiveEventsInput,
) (*objectiveEventsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	if in.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}

	events, err := svc.GetObjectiveEvents(ctx, in.MatchID)
	if err != nil {
		if capErr, ok := MapCapabilityError(ctx, err, "match.detail.objective_events"); ok {
			return nil, capErr
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "objective_events_error", err.Error())
	}
	return &objectiveEventsOutput{Body: toObjectiveEventsDTO(events)}, nil
}
