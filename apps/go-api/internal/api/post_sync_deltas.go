// Package api — post_sync_deltas.go : émission des notifications delta après une sync.
//
// Stratégie : avant la sync on capture une PlayerSnapshot (rank season pass +
// count des awards complétés), après la sync on capture une nouvelle, et on
// émet les notifications correspondant aux deltas observés.
//
// MVP couvre `season_pass_level` et `objective_completed`. Les hooks plus
// complexes (challenge_completed via citations diff, personal_record via
// référentiel records, threshold_crossed sur KD/winrate, objective_assigned)
// sont laissés en TODO documenté ci-dessous — l'infrastructure est en place
// pour les ajouter incrémentalement.
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// buildPostSyncDeltaHook construit la closure consommée par sync_handler :
// elle prend une snapshot avant le run, retourne une fonction qui prend
// la snapshot d'après et émet les notifications correspondantes.
func buildPostSyncDeltaHook(reg *ServiceRegistry) handlers.PostSyncDeltaHook {
	return func(ctx context.Context, slug string) func(ctx context.Context) {
		pdb, err := reg.resolve(ctx, slug)
		if err != nil {
			slog.WarnContext(ctx, "post_sync: resolve before", "slug", slug, "err", err)
			return nil
		}
		before, err := SnapshotPlayerState(ctx, pdb)
		if err != nil {
			slog.WarnContext(ctx, "post_sync: snapshot before", "slug", slug, "err", err)
		}
		return func(ctx context.Context) {
			pdb2, err := reg.resolve(ctx, slug)
			if err != nil {
				slog.WarnContext(ctx, "post_sync: resolve after", "slug", slug, "err", err)
				return
			}
			after, err := SnapshotPlayerState(ctx, pdb2)
			if err != nil {
				slog.WarnContext(ctx, "post_sync: snapshot after", "slug", slug, "err", err)
				return
			}
			emitter, err := reg.NotificationsEmitter(ctx, slug)
			if err != nil {
				slog.WarnContext(ctx, "post_sync: emitter", "slug", slug, "err", err)
				return
			}
			EmitPostSyncDeltas(ctx, emitter, slug, before, after)
		}
	}
}

// PlayerSnapshot capture l'état pertinent d'un joueur pour la détection delta.
type PlayerSnapshot struct {
	CurrentRank        int
	CurrentRankName    string
	PersonalAwardCount int
	CitationsCount     int
}

// SnapshotPlayerState lit l'état courant nécessaire à la détection delta.
// Robuste aux tables vides ou colonnes manquantes (renvoie zero-values).
func SnapshotPlayerState(ctx context.Context, pdb *duckdb.PlayerDB) (*PlayerSnapshot, error) {
	if pdb == nil || pdb.Player == nil {
		return &PlayerSnapshot{}, nil
	}
	s := &PlayerSnapshot{}

	// Career rank : dernière entrée career_progression
	var rank sql.NullInt64
	var rankName sql.NullString
	err := pdb.ReadDB().QueryRow(ctx,
		`SELECT rank, rank_name FROM career_progression
		 ORDER BY recorded_at DESC LIMIT 1`,
	).Scan(&rank, &rankName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Table peut être vide ou absente — log mais continue
		slog.DebugContext(ctx, "snapshot: career_progression query", "err", err)
	}
	if rank.Valid {
		s.CurrentRank = int(rank.Int64)
	}
	if rankName.Valid {
		s.CurrentRankName = rankName.String
	}

	// Awards : count total
	var awardCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx,
		`SELECT COUNT(*) FROM personal_score_awards`,
	).Scan(&awardCount); err != nil {
		slog.DebugContext(ctx, "snapshot: psa count", "err", err)
	}
	if awardCount.Valid {
		s.PersonalAwardCount = int(awardCount.Int64)
	}

	// Citations : count total (pour challenges_completed futur)
	var citationsCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx,
		`SELECT COUNT(*) FROM match_citations`,
	).Scan(&citationsCount); err != nil {
		slog.DebugContext(ctx, "snapshot: citations count", "err", err)
	}
	if citationsCount.Valid {
		s.CitationsCount = int(citationsCount.Int64)
	}

	return s, nil
}

// EmitPostSyncDeltas compare 2 snapshots et émet les notifications applicables.
//
// Best-effort : toute erreur est loguée et n'interrompt pas le flux de sync.
func EmitPostSyncDeltas(
	ctx context.Context,
	emitter notifications.Emitter,
	slug string,
	before, after *PlayerSnapshot,
) {
	if emitter == nil || before == nil || after == nil {
		return
	}

	// season_pass_level : nouveau rang franchi
	if after.CurrentRank > before.CurrentRank && after.CurrentRank > 0 {
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategorySeasonPassLevel,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.season_pass_level.title",
			BodyKey:  "notif.season_pass_level.body",
			Params: map[string]any{
				"level":     after.CurrentRank,
				"rank_name": after.CurrentRankName,
			},
			TargetRoute: fmt.Sprintf("/players/%s/palmares/season-pass", slug),
			Source:      "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: season_pass_level", "err", err)
		}
	}

	// objective_completed (agrégé) : nouveaux awards
	if after.PersonalAwardCount > before.PersonalAwardCount {
		delta := after.PersonalAwardCount - before.PersonalAwardCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryObjectiveCompleted,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.objective_completed.title",
			BodyKey:  "notif.objective_completed.body",
			Params: map[string]any{
				"count": delta,
				"name":  fmt.Sprintf("%d nouvel(s) objectif(s)", delta),
			},
			TargetRoute: fmt.Sprintf("/players/%s/objectifs", slug),
			Source:      "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_completed", "err", err)
		}
	}

	// challenge_completed (via citations diff) — émis seulement si delta>0
	if after.CitationsCount > before.CitationsCount {
		delta := after.CitationsCount - before.CitationsCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryChallengeCompleted,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.challenge_completed.title",
			BodyKey:  "notif.challenge_completed.body",
			Params: map[string]any{
				"count": delta,
				"name":  fmt.Sprintf("%d nouvelle(s) citation(s)", delta),
			},
			TargetRoute: fmt.Sprintf("/players/%s/defis", slug),
			Source:      "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_completed", "err", err)
		}
	}

	// TODO(post-sync v2):
	// - personal_record : nécessite une table de records (premier KDA > seuil,
	//   première victoire flawless, série de N victoires). Snapshot pourrait
	//   inclure max_kda, max_streak, has_flawless.
	// - threshold_crossed : KD/WR aggregés franchissant un palier (0.05 / 5%).
	//   Snapshot inclurait current_kd et current_winrate.
	// - objective_assigned : ne peut être détecté qu'avec un signal "auto-assigned"
	//   du module Objectifs (pas dans personal_score_awards).
	// - challenge_added : pareil, signal côté Prestige.
}
