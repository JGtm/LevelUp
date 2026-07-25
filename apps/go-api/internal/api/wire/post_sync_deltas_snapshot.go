// Package api — post_sync_deltas_snapshot.go : capture de l'état joueur
// (PlayerSnapshot) avant/après sync. Extrait de post_sync_deltas.go (refactor god-file).
package wire

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

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

	// LastMatchStartTime : start canonique (UTC) du match le plus récent du joueur
	// dans le shared. Watermark de la détection « rival croisé » (lot relations-E) :
	// after > before ⇒ nouveau match ⇒ détection ; sinon SKIP (coût nul sur les
	// syncs à vide). Zéro si aucun match (aucune détection possible).
	LastMatchStartTime time.Time

	// EarnedMedalIDs : ensemble des medal_name_id à SUM(count) > 0 pour le joueur
	// dans le shared du titre (medals_earned, lecture brute — parité Q36a, table
	// per-match INSERT-only sans vue _latest). Base de la détection « médaille
	// inédite » (V72-20) : after \ before = médailles décrochées pour la première
	// fois. nil/vide si medals_earned est absente (titre sans la capability) ou
	// illisible → la garde cold-start de emitMedalFirstEarned sème alors sans notifier.
	EarnedMedalIDs map[int64]struct{}
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

	// scanSnapshotRow exécute une lecture mono-ligne best-effort sur la player DB
	// via QueryRowRecovered (tolère un Reopen() concurrent — sweep recovery
	// player-DB 2026-07-16, PLAN_PLAYER_DB_RECOVERY_SWEEP). Les tables peuvent être
	// vides ou absentes (DB legacy) : une erreur non-ErrNoRows est loggée en Debug et
	// les dest restent en zero-value. Centralise les 6 lectures mono-ligne du snapshot
	// (CLAUDE.md règle 6 : ≤ 2 copies d'un même pattern).
	scanSnapshotRow := func(logMsg, query string, dest ...any) {
		rows, err := pdb.ReadDB().QueryRowRecovered(ctx, query)
		if err == nil {
			err = rows.Scan(dest...)
			rows.Close()
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, logMsg, "err", err)
		}
	}

	// Career rank : dernière entrée career_progression (table peut être vide/absente).
	var rank sql.NullInt64
	var rankName sql.NullString
	scanSnapshotRow("snapshot: career_progression query",
		`SELECT rank, rank_name FROM career_progression
		 ORDER BY recorded_at DESC LIMIT 1`,
		&rank, &rankName)
	if rank.Valid {
		s.CurrentRank = int(rank.Int64)
	}
	if rankName.Valid {
		s.CurrentRankName = rankName.String
	}

	// Awards : count total
	var awardCount sql.NullInt64
	scanSnapshotRow("snapshot: psa count",
		`SELECT COUNT(*) FROM personal_score_awards_latest`, &awardCount)
	if awardCount.Valid {
		s.PersonalAwardCount = int(awardCount.Int64)
	}

	// Citations : count total (pour challenges_completed)
	var citationsCount sql.NullInt64
	scanSnapshotRow("snapshot: citations count",
		`SELECT COUNT(*) FROM match_citations_latest`, &citationsCount)
	if citationsCount.Valid {
		s.CitationsCount = int(citationsCount.Int64)
	}

	// Challenge paths distincts (pour challenge_added)
	var pathsCount sql.NullInt64
	scanSnapshotRow("snapshot: challenge paths",
		`SELECT COUNT(DISTINCT challenge_path) FROM challenge_snapshots`, &pathsCount)
	if pathsCount.Valid {
		s.ChallengePathsCount = int(pathsCount.Int64)
	}

	// Challenge completed : nb challenge_path dont le DERNIER snapshot a
	// status='Completed' (vrai détecteur de défi termin, vs. compteur citations
	// utilisé auparavant). Insensible à la casse pour matcher 'Completed'/'COMPLETED'.
	var completedCount sql.NullInt64
	scanSnapshotRow("snapshot: challenge completed", `
		SELECT COUNT(*) FROM (
			SELECT challenge_path, LAST(status ORDER BY snapshot_at) AS last_status
			FROM challenge_snapshots
			GROUP BY challenge_path
		) WHERE UPPER(last_status) = 'COMPLETED'
	`, &completedCount)
	if completedCount.Valid {
		s.ChallengeCompletedCount = int(completedCount.Int64)
	}

	// Skill tier (CSR / LUSR) : dernière entrée par playlist_group dans
	// match_skill_rank. La map est (playlist_group → "rating_type|tier|sub_tier")
	// — toute transition de cette valeur déclenche un emit skill_tier.
	rows, err := pdb.ReadDB().QueryRecovered(ctx, `
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
	scanSnapshotRow("snapshot: battlepass completed", `
		SELECT COUNT(*) FROM (
			SELECT reward_track_path,
			       LAST(has_reached_max_rank ORDER BY snapshot_at) AS last_max
			FROM battlepass_snapshots
			GROUP BY reward_track_path
		) WHERE last_max = TRUE
	`, &bpCompleted)
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

			// Best KDA matériel (single match).
			//
			// DETTE K1a (ADR 0006) : ce calcul `(kills+assists)/GREATEST(deaths,1)`
			// est un QUOTIENT — interdit par ADR 0006 (per-match KDA = valeur API
			// native `mp.kda`, jamais un quotient recalculé). Le fix correct =
			// `SELECT kda ... ORDER BY kda DESC`. MAIS il ne peut PAS être appliqué
			// seul : les records best_kda déjà persistés sont sur l'échelle quotient ;
			// la KDA native étant systématiquement plus petite, les anciens records
			// resteraient BLOQUÉS (jamais battus → jamais auto-guéris). Requiert donc
			// formule + RE-BACKFILL coordonné des records (décision data/produit,
			// op prod gated). Scopé — cf. plan K1a + DETTE_ASSUMEE.
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

			// LastMatchStartTime : watermark de la détection « rival croisé »
			// (lot relations-E). Fragment timezone canonique partagé (règle
			// CLAUDE.md n°8) via StartTimeCanonicalSQL — jamais start_time brut.
			var lastStart sql.NullTime
			err = sharedDB.QueryRowContext(ctx, `
				SELECT MAX(`+duckdb.StartTimeCanonicalSQL("r")+`)
				FROM match_participants p
				JOIN match_registry r ON r.match_id = p.match_id
				WHERE p.xuid = ?`, pdb.XUID).Scan(&lastStart)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				slog.DebugContext(ctx, "snapshot: last_match_start", "err", err)
			}
			if lastStart.Valid {
				s.LastMatchStartTime = lastStart.Time.UTC()
			}

			// EarnedMedalIDs : set des médailles déjà obtenues (V72-20). nil =
			// INVALIDE (cf. loadEarnedMedalIDs) → détection « médaille inédite »
			// en seed silencieux.
			s.EarnedMedalIDs = loadEarnedMedalIDs(ctx, sharedDB, pdb.XUID)
			release()
		}
	}

	return s, nil
}

// loadEarnedMedalIDs retourne l'ensemble des medal_name_id déjà décrochés par le
// joueur : lecture BRUTE de medals_earned (table per-match INSERT-only, agrégat
// SUM(count) > 0 ; pas de vue _latest à consommer — parité Q36a).
//
// CONTRAT D'INVALIDITÉ (contre-revue V7.2, 2026-07-25) : retourne nil dès que le
// set ne peut pas être garanti COMPLET — table absente (titre sans capability
// médailles), requête en erreur, échec de scan, ou rows.Err() non nul après une
// itération partielle. Un set TRONQUÉ est PIRE qu'un set absent : côté snapshot
// « before » il fait passer des médailles déjà connues pour inédites, donc de
// FAUSSES notifications « médaille inédite ». nil ⇒ la garde cold-start de
// emitMedalFirstEarned sème silencieusement et le diff se tait pour ce cycle
// (même pattern que snapshotLooksCold). Un set VIDE non-nil (0 ligne, requête
// saine) reste un état valide et légitime : joueur sans médaille.
func loadEarnedMedalIDs(ctx context.Context, sharedDB *sql.DB, xuid string) map[int64]struct{} {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT medal_name_id
		FROM medals_earned
		WHERE xuid = ?
		GROUP BY medal_name_id
		HAVING SUM(count) > 0`, xuid)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, "snapshot: medals_earned indisponible — set médailles invalidé", "err", err)
		}
		return nil
	}
	defer rows.Close()

	out := map[int64]struct{}{}
	for rows.Next() {
		var medalID sql.NullInt64
		if err := rows.Scan(&medalID); err != nil {
			slog.WarnContext(ctx, "snapshot: medals_earned scan — set médailles invalidé (diff supprimé)", "err", err)
			return nil
		}
		if medalID.Valid {
			out[medalID.Int64] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "snapshot: medals_earned itération interrompue — set médailles invalidé (diff supprimé)",
			"err", err, "partiel", len(out))
		return nil
	}
	return out
}

// thresholdCrossed retourne true si une métrique est passée au-dessus d'un palier
// (granularité step) entre deux snapshots, vers le haut uniquement.
//
// Exemple : before=0.99, after=1.04, step=0.05 → crosses 1.00 → true.
//
//	before=1.04, after=0.99, step=0.05 → descente → false.
