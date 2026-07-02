// Package api — post_sync_deltas_snapshot.go : capture de l'état joueur
// (PlayerSnapshot) avant/après sync. Extrait de post_sync_deltas.go (refactor god-file).
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

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
		`SELECT COUNT(*) FROM personal_score_awards_latest`,
	).Scan(&awardCount); err != nil {
		slog.DebugContext(ctx, "snapshot: psa count", "err", err)
	}
	if awardCount.Valid {
		s.PersonalAwardCount = int(awardCount.Int64)
	}

	// Citations : count total (pour challenges_completed)
	var citationsCount sql.NullInt64
	if err := pdb.ReadDB().QueryRow(ctx,
		`SELECT COUNT(*) FROM match_citations_latest`,
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
				ROW_NUMBER() OVER (PARTITION BY playlist_group ORDER BY start_time DESC, match_id DESC) AS rn
			FROM match_skill_rank_latest
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
