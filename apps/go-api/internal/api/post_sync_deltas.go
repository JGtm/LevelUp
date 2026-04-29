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
		xuid := pdb.XUID // capturé pour l'invalidation — indépendant du resolve post-sync
		before, err := SnapshotPlayerState(ctx, pdb)
		if err != nil {
			slog.WarnContext(ctx, "post_sync: snapshot before", "slug", slug, "err", err)
		}
		return func(ctx context.Context) {
			// Invalider le cache home en premier — le sync a réussi, les données DB sont à jour.
			reg.homeMatchesCache.Invalidate(xuid)
			slog.InfoContext(ctx, "post_sync: home cache invalidé", "slug", slug, "xuid", xuid)

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
			EmitPostSyncDeltas(ctx, emitter, slug, before, after, pdb2)
		}
	}
}

// PlayerSnapshot capture l'état pertinent d'un joueur pour la détection delta.
type PlayerSnapshot struct {
	CurrentRank         int
	CurrentRankName     string
	PersonalAwardCount  int
	CitationsCount      int
	ChallengePathsCount int     // nb challenge_path distinct connus (challenge_snapshots)
	KDRatio             float64 // KD agrégé sur tous les matchs ingérés
	Winrate             float64 // 0..1 — fraction de matchs gagnés (outcome=2)
	BestKDA             float64 // record matériel (kills+assists)/max(deaths,1) sur 1 match
	BestKDAMatchID      string  // match associé au record
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

	// Citations : count total (pour challenges_completed)
	var citationsCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx,
		`SELECT COUNT(*) FROM match_citations`,
	).Scan(&citationsCount); err != nil {
		slog.DebugContext(ctx, "snapshot: citations count", "err", err)
	}
	if citationsCount.Valid {
		s.CitationsCount = int(citationsCount.Int64)
	}

	// Challenge paths distincts (pour challenge_added)
	var pathsCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx,
		`SELECT COUNT(DISTINCT challenge_path) FROM challenge_snapshots`,
	).Scan(&pathsCount); err != nil {
		slog.DebugContext(ctx, "snapshot: challenge paths", "err", err)
	}
	if pathsCount.Valid {
		s.ChallengePathsCount = int(pathsCount.Int64)
	}

	// KD agrégé + winrate via shared.match_participants (nécessite ATTACH actif)
	if pdb.XUID != "" {
		var kd, winrate sql.NullFloat64
		err := pdb.ReadDB().QueryRow(ctx, `
			SELECT
				CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)        AS kd_ratio,
				AVG(CASE WHEN outcome = 2 THEN 1.0 ELSE 0.0 END)            AS winrate
			FROM shared.match_participants
			WHERE xuid = ?`, pdb.XUID).Scan(&kd, &winrate)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, "snapshot: kd/winrate", "err", err)
		}
		if kd.Valid {
			s.KDRatio = kd.Float64
		}
		if winrate.Valid {
			s.Winrate = winrate.Float64
		}

		// Best KDA matériel (single match)
		var bestKDA sql.NullFloat64
		var matchID sql.NullString
		err = pdb.ReadDB().QueryRow(ctx, `
			SELECT
				CAST(kills + assists AS DOUBLE) / GREATEST(deaths, 1) AS kda,
				match_id
			FROM shared.match_participants
			WHERE xuid = ?
			ORDER BY kda DESC
			LIMIT 1`, pdb.XUID).Scan(&bestKDA, &matchID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, "snapshot: best_kda", "err", err)
		}
		if bestKDA.Valid {
			s.BestKDA = bestKDA.Float64
		}
		if matchID.Valid {
			s.BestKDAMatchID = matchID.String
		}
	}

	return s, nil
}

// thresholdCrossed retourne true si une métrique est passée au-dessus d'un palier
// (granularité step) entre deux snapshots, vers le haut uniquement.
//
// Exemple : before=0.99, after=1.04, step=0.05 → crosses 1.00 → true.
//
//	before=1.04, after=0.99, step=0.05 → descente → false.
func thresholdCrossed(before, after, step float64) (crossed bool, level float64) {
	if step <= 0 || after <= before {
		return false, 0
	}
	beforeBucket := int(before / step)
	afterBucket := int(after / step)
	if afterBucket > beforeBucket {
		return true, float64(afterBucket) * step
	}
	return false, 0
}

// EmitPostSyncDeltas compare 2 snapshots et émet les notifications applicables.
//
// Best-effort : toute erreur est loguée et n'interrompt pas le flux de sync.
// `pdb` est passé pour persister les nouveaux records ; peut être nil
// (dans ce cas personal_record est skippé).
func EmitPostSyncDeltas(
	ctx context.Context,
	emitter notifications.Emitter,
	slug string,
	before, after *PlayerSnapshot,
	pdb *duckdb.PlayerDB,
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

	// objective_completed (agrégé) : nouveaux awards. Le frontend résout
	// le libellé via templates i18n ; on ne passe que les paramètres machine.
	if after.PersonalAwardCount > before.PersonalAwardCount {
		delta := after.PersonalAwardCount - before.PersonalAwardCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryObjectiveCompleted,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.objective_completed.title",
			BodyKey:     "notif.objective_completed.body",
			Params:      map[string]any{"count": delta},
			TargetRoute: fmt.Sprintf("/players/%s/objectifs", slug),
			Source:      "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_completed", "err", err)
		}
	}

	// challenge_completed (via citations diff) — émis seulement si delta>0.
	// Le frontend résout le libellé via templates i18n.
	if after.CitationsCount > before.CitationsCount {
		delta := after.CitationsCount - before.CitationsCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:     notifications.CategoryChallengeCompleted,
			Severity:     notifications.SeveritySuccess,
			TitleKey:     "notif.challenge_completed.title",
			BodyKey:      "notif.challenge_completed.body",
			Params:       map[string]any{"count": delta},
			TargetRoute:  fmt.Sprintf("/players/%s/objectifs", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_completed", "err", err)
		}
	}

	// challenge_added : nouveaux challenge_path apparus dans challenge_snapshots.
	if after.ChallengePathsCount > before.ChallengePathsCount {
		delta := after.ChallengePathsCount - before.ChallengePathsCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:     notifications.CategoryChallengeAdded,
			Severity:     notifications.SeverityInfo,
			TitleKey:     "notif.challenge_added.title",
			BodyKey:      "notif.challenge_added.body",
			Params:       map[string]any{"count": delta},
			TargetRoute:  fmt.Sprintf("/players/%s/objectifs", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_added", "err", err)
		}
	}

	// objective_assigned : la table personal_score_awards ne distingue pas
	// "assigned" vs "completed". Le delta seul détecte une nouvelle entrée
	// (qui peut être déjà complétée). On émet quand même : l'utilisateur peut
	// distinguer dans l'UI via les params (target = score atteint vs assigné).
	// Doublon possible avec objective_completed sur le même match — accepté pour
	// le MVP, à raffiner avec un signal explicite côté Prestige.
	if after.PersonalAwardCount > before.PersonalAwardCount {
		delta := after.PersonalAwardCount - before.PersonalAwardCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryObjectiveAssigned,
			Severity:    notifications.SeverityInfo,
			TitleKey:    "notif.objective_assigned.title",
			BodyKey:     "notif.objective_assigned.body",
			Params:      map[string]any{"count": delta},
			TargetRoute: fmt.Sprintf("/players/%s/objectifs", slug),
			Source:      "post_sync",
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_assigned", "err", err)
		}
	}

	// threshold_crossed — KD ratio (palier 0.05). On envoie metric_key (clé i18n)
	// + value formaté ; le frontend résout metric_label via i18n.metricLabel.
	if crossed, level := thresholdCrossed(before.KDRatio, after.KDRatio, 0.05); crossed {
		_ = emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryThresholdCrossed,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.threshold_crossed.title",
			BodyKey:     "notif.threshold_crossed.body",
			Params:      map[string]any{"metric_key": "kd_ratio", "value": fmt.Sprintf("%.2f", level)},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      "post_sync",
		})
	}

	// threshold_crossed — Winrate (palier 0.05 = 5%)
	if crossed, level := thresholdCrossed(before.Winrate, after.Winrate, 0.05); crossed {
		_ = emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryThresholdCrossed,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.threshold_crossed.title",
			BodyKey:     "notif.threshold_crossed.body",
			Params:      map[string]any{"metric_key": "winrate", "value": fmt.Sprintf("%.0f%%", level*100)},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      "post_sync",
		})
	}

	// personal_record : best_kda matériel battu
	if pdb != nil && after.BestKDA > 0 {
		oldRec, err := loadPlayerRecord(ctx, pdb, "best_kda")
		if err != nil {
			slog.DebugContext(ctx, "post_sync: load best_kda record", "err", err)
		}
		if oldRec.Loaded && after.BestKDA > oldRec.Value+0.01 {
			// Record battu → emit + persist. metric_key résolu côté frontend.
			_ = emitter.Emit(ctx, notifications.EmitInput{
				Category: notifications.CategoryPersonalRecord,
				Severity: notifications.SeveritySuccess,
				TitleKey: "notif.personal_record.title",
				BodyKey:  "notif.personal_record.body",
				Params: map[string]any{
					"metric_key": "kda",
					"value":      fmt.Sprintf("%.2f", after.BestKDA),
					"previous":   fmt.Sprintf("%.2f", oldRec.Value),
				},
				TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
				Source:      "post_sync",
			})
		}
		// Toujours persister la nouvelle valeur (init au premier passage,
		// update si battue)
		if !oldRec.Loaded || after.BestKDA > oldRec.Value+0.01 {
			if err := upsertPlayerRecord(ctx, pdb, "best_kda", after.BestKDA, after.BestKDAMatchID); err != nil {
				slog.WarnContext(ctx, "post_sync: persist best_kda", "err", err)
			}
		}
	}
}

// playerRecord est l'état stocké d'un record dans player_records.
type playerRecord struct {
	Loaded        bool
	Value         float64
	AchievedMatch string
}

func loadPlayerRecord(ctx context.Context, pdb *duckdb.PlayerDB, metric string) (playerRecord, error) {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return playerRecord{}, nil
	}
	var v sql.NullFloat64
	var matchID sql.NullString
	err := pdb.SharedSocial.QueryRow(ctx,
		`SELECT value, achieved_match_id FROM player_records WHERE xuid = ? AND metric = ?`,
		pdb.XUID, metric,
	).Scan(&v, &matchID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return playerRecord{}, nil
	case err != nil:
		return playerRecord{}, err
	}
	return playerRecord{
		Loaded:        v.Valid,
		Value:         v.Float64,
		AchievedMatch: matchID.String,
	}, nil
}

func upsertPlayerRecord(ctx context.Context, pdb *duckdb.PlayerDB, metric string, value float64, matchID string) error {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return fmt.Errorf("upsertPlayerRecord: shared_social or xuid missing")
	}
	rwDB, err := duckdb.OpenReadWrite(pdb.SharedSocial.Path())
	if err != nil {
		return fmt.Errorf("open rw: %w", err)
	}
	defer rwDB.Close()
	_, err = rwDB.Exec(ctx, `
		INSERT INTO player_records (xuid, metric, value, achieved_at, achieved_match_id, updated_at)
		VALUES (?, ?, ?, NOW(), ?, NOW())
		ON CONFLICT (xuid, metric) DO UPDATE SET
			value             = EXCLUDED.value,
			achieved_at       = EXCLUDED.achieved_at,
			achieved_match_id = EXCLUDED.achieved_match_id,
			updated_at        = NOW()
	`, pdb.XUID, metric, value, nullableMatchID(matchID))
	return err
}

func nullableMatchID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
