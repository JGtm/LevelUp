package duckdb

// player_snapshot_store.go : la capture de l'état d'UN joueur (rang, awards, défis, niveaux de classement,
// KD, médailles...) avant et après une synchronisation, derrière `port.PlayerSnapshotReader`. Lectures seules,
// best-effort : une table vide, absente (base ancienne) ou illisible laisse les champs à zéro et se journalise
// au niveau Debug. Les tables append-only se lisent par leurs vues `_latest`.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

var _ port.PlayerSnapshotReader = (*PlayerSnapshotStore)(nil)

// PlayerSnapshotStore lit l'état d'un joueur dans sa base et dans la base partagée de son titre.
type PlayerSnapshotStore struct {
	pdb *PlayerDB
}

// NewPlayerSnapshotStore lie le dépôt au joueur de `pdb` ; pdb.Player doit être renseigné pour
// FillPlayerCounters, pdb.XUID et pdb.TitleSlug pour FillSharedAggregates.
func NewPlayerSnapshotStore(pdb *PlayerDB) *PlayerSnapshotStore {
	return &PlayerSnapshotStore{pdb: pdb}
}

// FillPlayerCounters remplit ce que la base du joueur sait : rang carrière, awards, citations, défis, niveaux
// de classement (CSR / LUSR), battle pass. Complexité linéaire en nombre de compteurs suivis, pas un défaut de
// conception.
//
//nolint:funlen // une lecture par compteur suivi ; chacune est NULL-aware.
func (st *PlayerSnapshotStore) FillPlayerCounters(ctx context.Context, s *port.PlayerSnapshot) {
	pdb := st.pdb
	// scanSnapshotRow exécute une lecture mono-ligne best-effort sur la player DB via QueryRowRecovered
	// (tolère un Reopen() concurrent — sweep recovery player-DB 2026-07-16). Les tables peuvent être vides ou
	// absentes (DB legacy) : une erreur non-ErrNoRows est loggée en Debug et les dest restent en zero-value.
	// Centralise les 6 lectures mono-ligne du snapshot (CLAUDE.md règle 6 : ≤ 2 copies d'un même pattern).
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

	st.fillSkillTiers(ctx, s)

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
}

// fillSkillTiers remplit SkillTierByPlaylist : dernière entrée par playlist_group dans match_skill_rank. La
// map est (playlist_group → "rating_type|tier|sub_tier") — toute transition de cette valeur déclenche une
// émission skill_tier.
func (st *PlayerSnapshotStore) fillSkillTiers(ctx context.Context, s *port.PlayerSnapshot) {
	rows, err := st.pdb.ReadDB().QueryRecovered(ctx, `
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
}

// FillSharedAggregates remplit ce que la base partagée sait du joueur : KD agrégé et taux de victoire, meilleur
// KDA, dernier match, médailles obtenues (SharedReader — ADR 0016). Sans xuid, rien n'est lu.
func (st *PlayerSnapshotStore) FillSharedAggregates(ctx context.Context, s *port.PlayerSnapshot) {
	pdb := st.pdb
	if pdb.XUID == "" {
		return
	}
	sharedDB, release, sharedErr := pdb.SharedReadDB().Get(ctx)
	if sharedErr != nil {
		slog.DebugContext(ctx, "snapshot: shared reader unavailable", "err", sharedErr)
		return
	}
	// Campagne exclue des trois lectures (D-5, backlog B4 ; no-op pour un titre sans variante de Campagne) :
	// les deltas notifiés suivent les agrégats affichés.
	campagneExclue := analysis.SQLExcludeCampaignByMatchID(pdb.TitleSlug, "match_id")
	var kd, winrate sql.NullFloat64
	err := sharedDB.QueryRowContext(ctx, `
				SELECT
					CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)        AS kd_ratio,
					AVG(CASE WHEN outcome = ? THEN 1.0 ELSE 0.0 END)            AS winrate
				FROM match_participants
				WHERE xuid = ?`+campagneExclue, domain.OutcomeWin, pdb.XUID).Scan(&kd, &winrate)
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
				WHERE xuid = ?`+campagneExclue+`
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

	// LastMatchStartTime : watermark de la détection « rival croisé » (lot relations-E). Fragment timezone
	// canonique partagé (règle CLAUDE.md n°8) via StartTimeCanonicalSQL — jamais start_time brut.
	var lastStart sql.NullTime
	err = sharedDB.QueryRowContext(ctx, `
				SELECT MAX(`+StartTimeCanonicalSQL("r")+`)
				FROM match_participants p
				JOIN match_registry r ON r.match_id = p.match_id
				WHERE p.xuid = ?`+analysis.SQLExcludeCampaignVariants(pdb.TitleSlug, "r"), pdb.XUID).Scan(&lastStart)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.DebugContext(ctx, "snapshot: last_match_start", "err", err)
	}
	if lastStart.Valid {
		s.LastMatchStartTime = lastStart.Time.UTC()
	}

	// EarnedMedalIDs : set des médailles déjà obtenues (V72-20). nil = INVALIDE (cf. loadEarnedMedalIDs) →
	// détection « médaille inédite » en seed silencieux.
	s.EarnedMedalIDs = loadEarnedMedalIDs(ctx, sharedDB, pdb.XUID, pdb.TitleSlug)
	release()
}

// loadEarnedMedalIDs retourne l'ensemble des medal_name_id déjà décrochés par le
// joueur : lecture BRUTE de medals_earned (table per-match INSERT-only, agrégat
// SUM(count) > 0 ; pas de vue _latest à consommer — parité Q36a), hors matchs de Campagne du
// titre (une médaille obtenue seulement en Campagne n'est pas « déjà obtenue »).
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
func loadEarnedMedalIDs(ctx context.Context, sharedDB *sql.DB, xuid, titleSlug string) map[int64]struct{} {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT medal_name_id
		FROM medals_earned
		WHERE xuid = ?`+analysis.SQLExcludeCampaignByMatchID(titleSlug, "match_id")+`
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
