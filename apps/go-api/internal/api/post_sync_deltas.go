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
	"sort"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Constantes partagées par les EmitInput émis depuis ce module.
const (
	// postSyncSource est la valeur du champ Source pour les notifications
	// émises après une sync (utilisée par notifications.Emitter pour la
	// dédup et l'analytics).
	postSyncSource = "post_sync"
	// paramKeyCount est la clé "count" du Params{} pour les notifications
	// delta agrégées (objectives, citations, friend_sync, etc.).
	paramKeyCount = "count"
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
		before, err := SnapshotPlayerState(ctx, pdb, newCitationsServiceForPDB(pdb))
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
			after, err := SnapshotPlayerState(ctx, pdb2, newCitationsServiceForPDB(pdb2))
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

			// Couche progression V2 (Ascension) — pipeline streaks/records/
			// milestones/coach. Non bloquant : toute erreur reste en slog.Warn.
			progDeps := BuildPlayerProgressionDeps(pdb2, emitter)
			if _, err := EvaluateProgressionAfterSync(ctx, pdb2, defaultProgressionTitleSlug(), progDeps, time.Now().UTC()); err != nil {
				slog.WarnContext(ctx, "post_sync: progression evaluate", "slug", slug, "err", err)
			}
		}
	}
}

// defaultProgressionTitleSlug retourne le slug de titre utilisé par le
// pipeline progression. Aligné sur l'unique titre supporté (halo_infinite).
// Quand le projet supportera plusieurs titres, lire depuis le contexte
// (ctxkeys.TitleSlug) sera la voie à privilégier.
func defaultProgressionTitleSlug() string {
	return "halo_infinite"
}

// PlayerSnapshot capture l'état pertinent d'un joueur pour la détection delta.
type PlayerSnapshot struct {
	// Rang carrière Halo lifetime (career_progression).
	CurrentRank     int
	CurrentRankName string

	// Awards / objectifs personnels.
	PersonalAwardCount int

	// Total brut de citations (legacy — utilisé pour la rétro-compat des tests
	// existants, plus émis comme challenge_completed depuis 2026-05-16).
	CitationsCount int

	// Défis daily/weekly (challenge_snapshots).
	ChallengePathsCount     int // nb challenge_path distinct connus
	ChallengeCompletedCount int // nb challenge_path dont le dernier status = 'Completed'

	// CSR / LUSR : dernière entrée par playlist_group dans match_skill_rank.
	// clé = playlist_group, valeur = "rating_type|tier|sub_tier".
	SkillTierByPlaylist map[string]string

	// Battle pass : tracks ayant atteint leur rang max (has_reached_max_rank=TRUE
	// dans le dernier snapshot par track).
	BattlepassCompletedTracks int

	// Citations / commendations agrégées via CitationsService.
	CitationTotalEarnedTiers int // somme des EarnedTiers sur toutes les citations
	CitationMasteryCount     int // nb de citations avec MasteryPct >= 100

	// Métriques agrégées.
	KDRatio        float64 // KD agrégé sur tous les matchs ingérés
	Winrate        float64 // 0..1 — fraction de matchs gagnés (outcome=2)
	BestKDA        float64 // record matériel (kills+assists)/max(deaths,1) sur 1 match
	BestKDAMatchID string  // match associé au record
}

// SnapshotPlayerState lit l'état courant nécessaire à la détection delta.
// Robuste aux tables vides ou colonnes manquantes (renvoie zero-values).
//
// `citationsSvc` peut être nil — dans ce cas les compteurs citation_tier /
// citation_mastery restent à 0 (pas d'émission).
//
// performance, etc.) avec branches NULL-aware. Complexité linéaire en nombre de KPIs
// trackés, pas un défaut de design.
//
//nolint:funlen,gocyclo // Snapshot scanne 12+ tables (career, citations, achievements,
func SnapshotPlayerState(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	citationsSvc port.CitationsService,
) (*PlayerSnapshot, error) {
	if pdb == nil || pdb.Player == nil {
		return &PlayerSnapshot{}, nil
	}
	s := &PlayerSnapshot{SkillTierByPlaylist: map[string]string{}}

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

	// Challenge completed : nb challenge_path dont le DERNIER snapshot a
	// status='Completed' (vrai détecteur de défi termin, vs. compteur citations
	// utilisé auparavant). Insensible à la casse pour matcher 'Completed'/'COMPLETED'.
	var completedCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT challenge_path, LAST(status ORDER BY snapshot_at) AS last_status
			FROM challenge_snapshots
			GROUP BY challenge_path
		) WHERE UPPER(last_status) = 'COMPLETED'
	`).Scan(&completedCount); err != nil {
		slog.DebugContext(ctx, "snapshot: challenge completed", "err", err)
	}
	if completedCount.Valid {
		s.ChallengeCompletedCount = int(completedCount.Int64)
	}

	// Skill tier (CSR / LUSR) : dernière entrée par playlist_group dans
	// match_skill_rank. La map est (playlist_group → "rating_type|tier|sub_tier")
	// — toute transition de cette valeur déclenche un emit skill_tier.
	rows, err := pdb.ReadDB().Query(ctx, `
		SELECT playlist_group, rating_type, tier, sub_tier
		FROM (
			SELECT
				playlist_group,
				rating_type,
				tier,
				sub_tier,
				ROW_NUMBER() OVER (PARTITION BY playlist_group ORDER BY start_time DESC) AS rn
			FROM match_skill_rank
			WHERE playlist_group IS NOT NULL AND tier IS NOT NULL
		) WHERE rn = 1
	`)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.DebugContext(ctx, "snapshot: skill_tier query", "err", err)
	}
	if rows != nil {
		for rows.Next() {
			var playlist, ratingType, tier sql.NullString
			var subTier sql.NullInt64
			if err := rows.Scan(&playlist, &ratingType, &tier, &subTier); err != nil {
				continue
			}
			if !playlist.Valid || !tier.Valid {
				continue
			}
			key := playlist.String
			val := fmt.Sprintf("%s|%s|%d", ratingType.String, tier.String, subTier.Int64)
			s.SkillTierByPlaylist[key] = val
		}
		_ = rows.Close()
	}

	// Battle pass : nb de tracks dont le DERNIER snapshot a has_reached_max_rank=TRUE.
	var bpCompleted sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT reward_track_path,
			       LAST(has_reached_max_rank ORDER BY snapshot_at) AS last_max
			FROM battlepass_snapshots
			GROUP BY reward_track_path
		) WHERE last_max = TRUE
	`).Scan(&bpCompleted); err != nil {
		slog.DebugContext(ctx, "snapshot: battlepass completed", "err", err)
	}
	if bpCompleted.Valid {
		s.BattlepassCompletedTracks = int(bpCompleted.Int64)
	}

	// Citations / commendations agrégées via le service (réutilise la chaîne
	// Q34 + Q35 + MergeCitationTotals → garantit que la sémantique tiers/mastery
	// est identique à celle de la page Citations). citationsSvc peut être nil.
	if citationsSvc != nil {
		page, err := citationsSvc.GetCitationsPage(ctx)
		if err != nil {
			slog.DebugContext(ctx, "snapshot: citations page", "err", err)
		} else if page != nil {
			for _, item := range page.Citations {
				s.CitationTotalEarnedTiers += item.EarnedTiers
				if item.MasteryPct >= 100.0 {
					s.CitationMasteryCount++
				}
			}
		}
	}

	// KD agrégé + winrate via match_participants (SharedReader — ADR 0016).
	if pdb.XUID != "" {
		sharedDB, release, sharedErr := pdb.SharedReadDB().Get(ctx)
		if sharedErr != nil {
			slog.DebugContext(ctx, "snapshot: shared reader unavailable", "err", sharedErr)
		} else {
			var kd, winrate sql.NullFloat64
			err := sharedDB.QueryRowContext(ctx, `
				SELECT
					CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)        AS kd_ratio,
					AVG(CASE WHEN outcome = 2 THEN 1.0 ELSE 0.0 END)            AS winrate
				FROM match_participants
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
			err = sharedDB.QueryRowContext(ctx, `
				SELECT
					CAST(kills + assists AS DOUBLE) / GREATEST(deaths, 1) AS kda,
					match_id
				FROM match_participants
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
			release()
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
//
// et émet 1 notification par delta significatif. Complexité reflète le nombre
// de KPIs surveillés, pas un défaut de conception.
//
//nolint:funlen,gocyclo // Émetteur multi-événements : compare ~12 snapshots delta
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

	// career_rank : nouveau rang Halo lifetime franchi (career_progression).
	// Remplace l'ancien câblage CategorySeasonPassLevel qui pointait à tort sur
	// career_progression — déprécié depuis 2026-05-16.
	if after.CurrentRank > before.CurrentRank && after.CurrentRank > 0 {
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryCareerRank,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.career_rank.title",
			BodyKey:  "notif.career_rank.body",
			Params: map[string]any{
				"rank":      after.CurrentRank,
				"rank_name": after.CurrentRankName,
				"previous":  before.CurrentRank,
			},
			TargetRoute: fmt.Sprintf("/players/%s/career", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: career_rank", "err", err)
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
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/objectifs", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: objective_completed", "err", err)
		}
	}

	// challenge_completed (vrais défis daily/weekly) : nb challenge_path dont le
	// dernier status est 'Completed'. Recâblé depuis match_citations vers
	// challenge_snapshots le 2026-05-16 — la sémantique citations est désormais
	// traitée par citation_tier / citation_mastery.
	if after.ChallengeCompletedCount > before.ChallengeCompletedCount {
		delta := after.ChallengeCompletedCount - before.ChallengeCompletedCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:     notifications.CategoryChallengeCompleted,
			Severity:     notifications.SeveritySuccess,
			TitleKey:     "notif.challenge_completed.title",
			BodyKey:      "notif.challenge_completed.body",
			Params:       map[string]any{paramKeyCount: delta},
			TargetRoute:  fmt.Sprintf("/players/%s/objectifs", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: challenge_completed", "err", err)
		}
	}

	// skill_tier (CSR / LUSR unifié) : une notif par playlist_group dont le
	// tier|sub_tier|rating_type a changé entre les 2 snapshots. Les apparitions
	// inédites (playlist absente avant, présente après) sont aussi émises.
	for _, playlist := range sortedPlaylistKeys(after.SkillTierByPlaylist) {
		newVal := after.SkillTierByPlaylist[playlist]
		oldVal := before.SkillTierByPlaylist[playlist]
		if newVal == oldVal {
			continue
		}
		ratingType, tier, subTier := splitSkillTier(newVal)
		oldRT, oldTier, oldSub := splitSkillTier(oldVal)
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategorySkillTier,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.skill_tier.title",
			BodyKey:  "notif.skill_tier.body",
			Params: map[string]any{
				"playlist_group":    playlist,
				"rating_type":       ratingType,
				"tier":              tier,
				"sub_tier":          subTier,
				"previous_type":     oldRT,
				"previous_tier":     oldTier,
				"previous_sub_tier": oldSub,
			},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: skill_tier", "playlist", playlist, "err", err)
		}
	}

	// battlepass_completed : un track de plus a atteint son rang max.
	if after.BattlepassCompletedTracks > before.BattlepassCompletedTracks {
		delta := after.BattlepassCompletedTracks - before.BattlepassCompletedTracks
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryBattlepassCompleted,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.battlepass_completed.title",
			BodyKey:     "notif.battlepass_completed.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/career/season-pass", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: battlepass_completed", "err", err)
		}
	}

	// citation_tier : au moins un nouveau palier franchi sur une commendation.
	// Agrégé : count = somme des paliers franchis depuis la sync précédente.
	if after.CitationTotalEarnedTiers > before.CitationTotalEarnedTiers {
		delta := after.CitationTotalEarnedTiers - before.CitationTotalEarnedTiers
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryCitationTier,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.citation_tier.title",
			BodyKey:     "notif.citation_tier.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/citations", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: citation_tier", "err", err)
		}
	}

	// citation_mastery : une (ou plusieurs) commendation(s) viennent d'atteindre
	// 100 % de masterisation.
	if after.CitationMasteryCount > before.CitationMasteryCount {
		delta := after.CitationMasteryCount - before.CitationMasteryCount
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryCitationMastery,
			Severity:    notifications.SeveritySuccess,
			TitleKey:    "notif.citation_mastery.title",
			BodyKey:     "notif.citation_mastery.body",
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/citations", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: citation_mastery", "err", err)
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
			Params:       map[string]any{paramKeyCount: delta},
			TargetRoute:  fmt.Sprintf("/players/%s/objectifs", slug),
			TargetSearch: map[string]any{"tab": "challenges"},
			Source:       postSyncSource,
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
			Params:      map[string]any{paramKeyCount: delta},
			TargetRoute: fmt.Sprintf("/players/%s/objectifs", slug),
			Source:      postSyncSource,
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
			Source:      postSyncSource,
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
			Source:      postSyncSource,
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
				Source:      postSyncSource,
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

// newCitationsServiceForPDB construit un CitationsService scopé sur le joueur.
// Retourne nil si pdb est invalide — SnapshotPlayerState saute alors la lecture
// citations.
func newCitationsServiceForPDB(pdb *duckdb.PlayerDB) port.CitationsService {
	if pdb == nil || pdb.Player == nil {
		return nil
	}
	return service.NewCitationsService(duckdb.NewCitationsRepo(pdb))
}

// sortedPlaylistKeys retourne les clés de m triées — garantit un ordre d'émission
// stable (utile pour les tests + journaux deterministes).
func sortedPlaylistKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// splitSkillTier décompose une signature "rating_type|tier|sub_tier" en
// composants. Renvoie ("", "", 0) sur entrée vide ou mal formée — ce qui
// laisse les params previous_* à zéro pour une 1re apparition de playlist.
func splitSkillTier(sig string) (ratingType, tier string, subTier int) {
	if sig == "" {
		return "", "", 0
	}
	var sub int
	parts := [3]string{}
	idx := 0
	start := 0
	for i := 0; i < len(sig) && idx < 3; i++ {
		if sig[i] == '|' {
			parts[idx] = sig[start:i]
			idx++
			start = i + 1
		}
	}
	if idx < 3 {
		parts[idx] = sig[start:]
	}
	_, _ = fmt.Sscanf(parts[2], "%d", &sub)
	return parts[0], parts[1], sub
}
