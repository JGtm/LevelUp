package duckdb

// post_sync_progression_store.go : les lectures de l'orchestrateur de progression qui suit une
// synchronisation, derrière `port.PostSyncProgressionReader`. Toutes sont en lecture seule. Depuis ADR 0016
// (retrait final d'attachShared), les lectures croisées sont scindées en deux phases : la partie partagée
// (match_participants, match_registry) lue par le fournisseur de lecture partagée (AcquireSharedRead,
// résilient à la fenêtre de swap), la partie joueur (player_match_enrichment_latest) lue sur la base du
// joueur ; la jointure se fait côté appelant. La Campagne est exclue de toutes les lectures (D-5).
//
// Les budgets de temps (progressionLoadBudget) sont posés par l'appelant : ce fichier prend le contexte tel
// quel.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
	"levelup/go-api/internal/port"
)

var _ port.PostSyncProgressionReader = (*PostSyncProgressionStore)(nil)

// PostSyncProgressionStore lit les matchs et les agrégats d'UN joueur pour le pipeline de progression.
type PostSyncProgressionStore struct {
	pdb *PlayerDB
}

// NewPostSyncProgressionStore lie le dépôt au joueur de `pdb` ; pdb.Player, pdb.XUID et pdb.TitleSlug doivent
// être renseignés.
func NewPostSyncProgressionStore(pdb *PlayerDB) *PostSyncProgressionStore {
	return &PostSyncProgressionStore{pdb: pdb}
}

// accuracyPercentScale : match_participants.accuracy porte la précision native de l'API en pourcentage
// (0..100, sync/transforms.go) ; le port et les métriques de records la lisent en ratio 0..1 (ADR 0006,
// bornes de progression/records). La conversion se fait à cette lecture, la seule qui alimente les records.
const accuracyPercentScale = "100.0"

// MatchRows lit les participants et le registre depuis la base partagée : les matchs du joueur depuis `since`,
// du plus ancien au plus récent. Un match sans instant de début est sauté.
func (st *PostSyncProgressionStore) MatchRows(ctx context.Context, since time.Time) ([]port.ProgressionMatchRow, error) {
	pdb := st.pdb
	sharedDB, release, err := AcquireSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return nil, fmt.Errorf("loadProgressionMatches: %w", err)
	}
	defer release()

	// start_time via le fragment timezone canonique (règle CLAUDE.md n°8) : SELECT,
	// filtre et tri passent tous par COALESCE(start_time_utc, start_time AT TIME
	// ZONE 'UTC') — un start_time brut mal-classe/exclut les imports OpenSpartan à
	// start_time NULL/TZ-décalé mais start_time_utc renseigné (alignement G1).
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			mp.match_id,
			`+sqlfrag.SQLStartTimeCanonical("mr")+` AS start_time,
			COALESCE(mp.kda, 0) AS kda,
			COALESCE(mp.kills, 0) AS kills,
			COALESCE(mp.deaths, 0) AS deaths,
			COALESCE(mp.assists, 0) AS assists,
			COALESCE(mp.accuracy, 0) / `+accuracyPercentScale+` AS accuracy,
			COALESCE(mp.personal_score, 0) AS personal_score,
			COALESCE(mp.time_played_seconds, 0) AS time_played_seconds,
			COALESCE(mp.damage_dealt, 0) AS damage_dealt,
			COALESCE(mp.damage_taken, 0) AS damage_taken
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND `+sqlfrag.SQLStartTimeCanonical("mr")+` >= ?`+
		sqlfrag.SQLExcludeCampaignVariants(pdb.TitleSlug, "mr")+`
		ORDER BY start_time ASC
	`, pdb.XUID, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("query progression matches: %w", err)
	}
	defer rows.Close()

	var loaded []port.ProgressionMatchRow
	for rows.Next() {
		var (
			r         port.ProgressionMatchRow
			startTime sql.NullTime
		)
		if err := rows.Scan(&r.MatchID, &startTime, &r.KDA, &r.Kills, &r.Deaths, &r.Assists, &r.Accuracy, &r.PersonalScore, &r.TimeSeconds, &r.DamageDealt, &r.DamageTaken); err != nil {
			return nil, fmt.Errorf("scan progression match: %w", err)
		}
		if !startTime.Valid {
			continue
		}
		r.StartTime = startTime.Time
		loaded = append(loaded, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return loaded, nil
}

// PerformanceScores lit performance_score depuis player_match_enrichment_latest (vue `_latest` de la table
// append-only), dans la base du joueur.
func (st *PostSyncProgressionStore) PerformanceScores(ctx context.Context, matchIDs []string) (map[string]float64, error) {
	perfByMatch := make(map[string]float64, len(matchIDs))
	if len(matchIDs) == 0 {
		return perfByMatch, nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	pmeRows, err := st.pdb.Player.QueryRecovered(ctx, `
		SELECT match_id, COALESCE(performance_score, 0)
		FROM player_match_enrichment_latest
		WHERE match_id IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query performance_score: %w", err)
	}
	defer pmeRows.Close()
	for pmeRows.Next() {
		var mid string
		var perf float64
		if err := pmeRows.Scan(&mid, &perf); err != nil {
			return nil, fmt.Errorf("scan performance_score: %w", err)
		}
		perfByMatch[mid] = perf
	}
	return perfByMatch, nil
}

// Aggregates agrège les compteurs cumulatifs nécessaires aux jalons : matches_played, wins, kills, headshots,
// assists, le nombre de jours distincts avec au moins un match de précision >= seuil, et les matchs qui
// franchissent les seuils de combat. Convention « victoire » : le seam d'issues title-aware (K1a), jamais un
// `outcome = 2` en dur côté appelant.
func (st *PostSyncProgressionStore) Aggregates(ctx context.Context, p port.ProgressionAggregateParams) (port.ProgressionAggregates, error) {
	pdb := st.pdb
	var out port.ProgressionAggregates
	sharedDB, release, err := AcquireSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return out, fmt.Errorf("loadPlayerStats: %w", err)
	}
	defer release()

	// « wins » via le seam d'issues title-aware (K1a) plutôt qu'un `outcome = 2`
	// codé en dur : byte-identique pour halo_infinite, correct pour tout titre au
	// raw_code différent. Slug explicite (pdb.TitleSlug) car le ctx post-sync
	// détaché ne porte pas forcément le titleSlug.
	winExpr := OutcomeSQLEqSlug(pdb.TitleSlug, "outcome", canonical.OutcomeWin, "outcome = 2")
	// Campagne exclue des trois agrégats (D-5, backlog B4 ; no-op pour un titre sans
	// variante de Campagne) : sous-requête par match_id là où le registre n'est pas joint.
	campagneExclue := sqlfrag.SQLExcludeCampaignByMatchID(pdb.TitleSlug, "match_id")
	campagneExclueMR := sqlfrag.SQLExcludeCampaignVariants(pdb.TitleSlug, "mr")
	statsQuery := `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN ` + winExpr + ` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(kills), 0),
			COALESCE(SUM(headshot_kills), 0),
			COALESCE(SUM(assists), 0)
		FROM match_participants
		WHERE xuid = ?` + campagneExclue + `
	`
	if err := sharedDB.QueryRowContext(ctx, statsQuery, pdb.XUID).Scan(&out.MatchesPlayed, &out.Wins, &out.Kills, &out.Headshots, &out.Assists); err != nil {
		return out, fmt.Errorf("aggregate stats: %w", err)
	}

	// accuracy_threshold_days : COUNT(DISTINCT jour UTC) où au moins 1 match a
	// accuracy >= seuil. Le bucket jour passe par le fragment timezone canonique
	// (règle CLAUDE.md n°8) — cohérent avec le backfill A6 (ops.ComputeMilestoneCrossings) ;
	// un CAST(start_time AS DATE) brut décalerait les jours aux frontières de fuseau.
	if err := sharedDB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT CAST(`+sqlfrag.SQLStartTimeCanonical("mr")+` AS DATE))
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mp.accuracy >= ?`+campagneExclueMR+`
	`, pdb.XUID, p.AccuracyThreshold).Scan(&out.AccuracyDays); err != nil {
		return out, fmt.Errorf("aggregate accuracy days: %w", err)
	}

	// effectiveHpToKill = baseline PV-pour-tuer du titre (225 Infinite ; 115 Halo 5).
	// Float de confiance issu de la config → injecté via Sprintf (%g), pas une
	// entrée utilisateur. Les seuils de milestone restent en bind params (?).
	combatMetricsQuery := fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN damage_dealt > 0
				AND %[1]g * (COALESCE(kills,0) + COALESCE(assists,0)/3.0) / damage_dealt >= ? THEN 1 END),
			COUNT(CASE WHEN damage_taken > 0 AND COALESCE(deaths,0) > 0
				AND damage_taken / (%[1]g * deaths) >= ? THEN 1 END),
			COUNT(CASE WHEN damage_dealt > 0 AND damage_taken > 0 AND COALESCE(deaths,0) > 0
				AND %[1]g * (COALESCE(kills,0) + COALESCE(assists,0)/3.0) / damage_dealt >= ?
				AND damage_taken / (%[1]g * deaths) >= ? THEN 1 END)
		FROM match_participants
		WHERE xuid = ?%[2]s
	`, p.HPToKill, campagneExclue)
	if err := sharedDB.QueryRowContext(ctx, combatMetricsQuery,
		p.OCThreshold, p.DRThreshold, p.OCThreshold, p.DRThreshold,
		pdb.XUID).Scan(&out.PrecisionMatches, &out.EnduranceMatches, &out.ExcellenceMatches); err != nil {
		return out, fmt.Errorf("aggregate combat metrics: %w", err)
	}
	return out, nil
}

// LastMatchTimes lit les deux matchs les plus récents du joueur, du plus récent au plus ancien.
//
// start_time via le fragment timezone canonique (règle CLAUDE.md n°8), SELECT + filtre NOT NULL + tri : un
// match à start_time NULL mais start_time_utc renseigné compte comme activité valide (alignement G1). Sur
// une itération interrompue, les instants déjà lus sont rendus avec l'erreur.
func (st *PostSyncProgressionStore) LastMatchTimes(ctx context.Context) ([]time.Time, error) {
	pdb := st.pdb
	sharedDB, release, err := AcquireSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return nil, fmt.Errorf("loadComebackContext: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT `+sqlfrag.SQLStartTimeCanonical("mr")+` AS start_time
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND `+sqlfrag.SQLStartTimeCanonical("mr")+` IS NOT NULL`+
		sqlfrag.SQLExcludeCampaignVariants(pdb.TitleSlug, "mr")+`
		ORDER BY start_time DESC
		LIMIT 2
	`, pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("query last matches: %w", err)
	}
	defer rows.Close()
	var times []time.Time
	for rows.Next() {
		var t sql.NullTime
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		if t.Valid {
			times = append(times, t.Time)
		}
	}
	return times, rows.Err()
}
