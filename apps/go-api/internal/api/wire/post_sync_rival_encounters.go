// Package api — post_sync_rival_encounters.go : émission de la notification
// « rival croisé » (lot relations-E) après une sync. Détection par watermark
// (PlayerSnapshot.LastMatchStartTime) : si la sync n'a ramené aucun nouveau match
// la détection est skippée (coût nul sur les syncs à vide, la majorité).
//
// Best-effort STRICT : toute erreur est loguée et n'interrompt JAMAIS le flux de
// sync — une notif ratée ne doit pas faire échouer une synchronisation.
package wire

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// rivalEncounterWinOutcome : libellé canonical d'un duel gagné (contrat DTO
// domain.RelationDuelEntry.Outcome). Sévérité SeveritySuccess sur ce cas, sinon
// SeverityInfo. Source côté service : duelLabelWin (relations_moments_service.go).
const rivalEncounterWinOutcome = "win"

// rivalEncounterDetector : sous-ensemble de *service.RelationsService consommé
// par l'émission post-sync. Interface locale pour rendre emitRivalEncounters
// testable avec un détecteur factice (E4).
type rivalEncounterDetector interface {
	DetectRivalEncounters(ctx context.Context, since time.Time) ([]domain.RivalEncounter, error)
}

// newRivalDetectorForPDB construit un détecteur scopé sur la player DB. Bare
// RelationsService (pas de filtres ni cross-game) : DetectRivalEncounters lit en
// scope nil et n'a besoin ni de segmentation ni du badge cross-jeu. Retourne nil
// si pdb est invalide — l'émission est alors sautée.
func newRivalDetectorForPDB(pdb *duckdb.PlayerDB) rivalEncounterDetector {
	if pdb == nil || pdb.Player == nil {
		return nil
	}
	return service.NewRelationsService(duckdb.NewCareerRepo(pdb))
}

// emitRivalEncounters détecte et émet les notifications « rival croisé » du cycle.
//
// Watermark : si after.LastMatchStartTime <= before.LastMatchStartTime, aucun
// nouveau match n'a été ingéré → détection SKIPPÉE. Sinon on détecte les duels
// postérieurs au watermark before (idempotent) et on émet une notification par
// duel (plafonné côté service à rivalNotifMaxPerSync).
func emitRivalEncounters(
	ctx context.Context,
	emitter notifications.Emitter,
	detector rivalEncounterDetector,
	slug string,
	before, after *PlayerSnapshot,
) {
	if emitter == nil || detector == nil || before == nil || after == nil {
		return
	}
	if !after.LastMatchStartTime.After(before.LastMatchStartTime) {
		slog.DebugContext(ctx, "post_sync: rival encounters skippés (aucun nouveau match)", "slug", slug)
		return
	}

	encounters, err := detector.DetectRivalEncounters(ctx, before.LastMatchStartTime)
	if err != nil {
		slog.WarnContext(ctx, "post_sync: détection rival croisé", "slug", slug, "err", err)
		return
	}

	for _, e := range encounters {
		severity := notifications.SeverityInfo
		if e.Outcome == rivalEncounterWinOutcome {
			severity = notifications.SeveritySuccess
		}
		emitErr := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryRivalEncounter,
			Severity: severity,
			TitleKey: "notif.rival_encounter.title",
			BodyKey:  "notif.rival_encounter.body",
			Params: map[string]any{
				"gamertag": e.Gamertag,
				"outcome":  e.Outcome,
				"kills":    e.KillsOnRival,
				"deaths":   e.DeathsByRival,
				"match_id": e.MatchID,
			},
			TargetRoute: fmt.Sprintf("/players/%s/matches/%s", slug, e.MatchID),
			Source:      postSyncSource,
		})
		if emitErr != nil {
			slog.WarnContext(ctx, "post_sync: rival_encounter emit",
				"slug", slug, "rival", e.Gamertag, "match_id", e.MatchID, "err", emitErr)
			continue
		}
		slog.InfoContext(ctx, "post_sync: rival_encounter émis",
			"slug", slug, "rival", e.Gamertag, "match_id", e.MatchID, "outcome", e.Outcome)
	}
	slog.DebugContext(ctx, "post_sync: rival encounters détectés", "slug", slug, "duels_new", len(encounters))
}
