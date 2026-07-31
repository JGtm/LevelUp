// Package sync — comeback.go : backfill du dominance_flag par match.
//
// Port de src/data/comeback_backfill.py.
//
// Le routage dépend du mode (objectiveevents.ObjectiveTypeOf sur game_variant_name) :
//
//   - Mode "flag" (CTF) : chemin OBJECTIF. La courbe de score vient des captures
//     (match_objective_events), pas des kills ni de la médaille Steaktacular.
//   - Mode "zone"/"hill"/"skull" (Strongholds/KOTH/Oddball) : retourne 0 tant que
//     le décodeur de score-over-time n'est pas câblé (cf. computeMatchDominanceFlag).
//   - Mode "" (non-objectif, ex. Slayer) : chemin HISTORIQUE inchangé :
//     1. DOMINATION   : médaille Steaktacular (ID 1169390319) gagnée par mon équipe.
//     2. HUMILIATION  : médaille Steaktacular gagnée par l'équipe adverse.
//     3-5. REMONTADA / DÉBÂCLE / CONTRE-REMONTADA : score curve depuis kill events,
//     uniquement si game_variant_name like '%slayer%'.
//
// Les valeurs 3-5 utilisent l'algo ComputeDominanceFlag (analysis/comeback.go)
// avec la sensibilité "standard", quelle que soit la source de courbe.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// steaktacularMedalIDForTitle résout l'ID de la médaille "killing spree"
// (DOMINATION/HUMILIATION via médaille) d'un titre. Title-aware (C6) : seul
// Infinite a un ID câblé (MedalSteaktacularID) ; les autres titres retombent sur
// la marge de score finale (analysis.ComputeScoreMarginDominance), déjà
// title-agnostic. ""/halo_infinite → (id, true) ; tout autre titre → (0, false)
// pour skipper la requête médaille (idempotent : l'ID HINF ne matche de toute
// façon aucune médaille d'un autre catalogue → même résultat, sans requête morte).
func steaktacularMedalIDForTitle(slug string) (int64, bool) {
	if slug == "" || slug == titlePkg.DefaultSlug {
		return analysis.MedalSteaktacularID, true
	}
	return 0, false
}

// BackfillDominanceFlags calcule et persiste le dominance_flag pour une liste de matchs.
//
// sharedDB : connexion à shared_matches_v2.duckdb (medals, events, participants, registry).
// playerDB : connexion à stats.duckdb du joueur (écriture player_match_enrichment).
// xuid     : identifiant Xbox du joueur.
// matchIDs : liste des match_id à traiter.
func BackfillDominanceFlags(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	xuid string,
	matchIDs []string,
) error {
	// Phase 3 du refactor ART (suppression LEVELUP_POSTSYNC_INSERT_ONLY) :
	// le chemin legacy row-by-row (writeDominanceFlag via ON CONFLICT DO UPDATE)
	// est supprimé car il déclenchait le bug ART DuckDB. Le batch path est
	// désormais le seul utilisé.
	return backfillDominanceFlagsBatch(ctx, sharedDB, playerDB, xuid, matchIDs)
}

// computeMatchDominanceFlag calcule le flag pour un match, en routant selon le
// mode (objectif vs Slayer). team_id/outcome du joueur sont communs aux chemins.
func computeMatchDominanceFlag(ctx context.Context, db *sql.DB, xuid, matchID string) (int, error) {
	myTeamID, outcome, err := loadMyTeamAndOutcome(ctx, db, matchID, xuid)
	if err != nil {
		return 0, fmt.Errorf("team/outcome: %w", err)
	}

	gameVariant, err := loadGameVariant(ctx, db, matchID)
	if err != nil {
		return 0, nil // non critique
	}

	switch objectiveevents.ObjectiveTypeOf(gameVariant) {
	case objectiveevents.ObjectiveTypeFlag:
		// CTF : courbe de score depuis les captures objectif (pas de Steaktacular).
		return computeCTFDominanceFlag(ctx, db, matchID, myTeamID, outcome)
	case objectiveevents.ObjectiveTypeZone,
		objectiveevents.ObjectiveTypeHill,
		objectiveevents.ObjectiveTypeSkull:
		// TODO(objectif) : score continu non décodé (Value nil sur th=10) — la
		// dominance objectif reste limitée au CTF tant que le décodeur de
		// score-over-time (byte842 varint Strongholds, etc.) n'est pas câblé.
		// On NE fabrique PAS de fausse courbe depuis des counts d'events.
		return analysis.DominanceFlagNone, nil
	default:
		// Mode non-objectif (Slayer, etc.) : chemin historique inchangé.
		return computeSlayerDominanceFlag(ctx, db, matchID, gameVariant, myTeamID, outcome)
	}
}

// computeSlayerDominanceFlag applique le chemin historique (médaille Steaktacular
// puis courbe de score depuis les kill events), réservé aux modes non-objectif.
func computeSlayerDominanceFlag(
	ctx context.Context, db *sql.DB, matchID, gameVariant string, myTeamID, outcome int,
) (int, error) {
	// 1. Vérifier médaille Steaktacular (DOMINATION / HUMILIATION).
	steakByTeam, err := loadSteaktacularByTeam(ctx, db, matchID)
	if err != nil {
		return 0, fmt.Errorf("steaktacular: %w", err)
	}
	if steakByTeam[myTeamID] > 0 {
		return analysis.DominanceFlagDomination, nil
	}
	for teamID := range steakByTeam {
		if teamID != myTeamID && steakByTeam[teamID] > 0 {
			return analysis.DominanceFlagHumiliation, nil
		}
	}

	// 2. Courbe de kills (timeline). Absente pour les titres dont les events ne
	// portent pas de kill-feed exploitable (Halo 5 : highlight_events = médailles
	// seules → 0 kill-event).
	events, err := loadKillEventsWithTeam(ctx, db, matchID)
	if err != nil {
		return 0, nil // non critique
	}
	if len(events) == 0 {
		// Pas de timeline → fallback marge de score FINALE (DOMINATION/HUMILIATION
		// title-agnostic, cf. analysis.ComputeScoreMarginDominance + .ai/STEAKTACULAR.md).
		// Gate HINF (byte-identique) : la prémisse historique « Infinite a TOUJOURS des
		// kill-events » est FAUSSE pour ~244 matchs BTB sans kill-event. Pour ne pas
		// relabelliser rétroactivement des badges dominance HINF existants (règle projet
		// HINF byte-identique), le fallback marge de score reste réservé aux titres sans
		// kill-feed exploitable (Halo 5). HINF garde dominance_flag=0 ici, comme avant.
		if ctxkeys.TitleSlug(ctx) == titlePkg.DefaultSlug {
			return 0, nil
		}
		// Limité aux 2-équipes (0/1) : FFA/multi-équipes → pas de dominance.
		if myTeamID != 0 && myTeamID != 1 {
			return 0, nil
		}
		t0, t1, ok := loadTeamScoresOrKillSums(ctx, db, matchID)
		if !ok {
			return 0, nil
		}
		myScore, enemyScore := t0, t1
		if myTeamID == 1 {
			myScore, enemyScore = t1, t0
		}
		return analysis.ComputeScoreMarginDominance(myScore, enemyScore, outcome, analysis.StandardLeadPct()), nil
	}

	// 3. Comeback (remontada + domination via courbe) — uniquement les modes Slayer.
	// `gameVariant` est déjà chargé par l'appelant, qui s'en sert pour router entre
	// chemin objectif et chemin historique : pas de second aller-retour en base.
	if !strings.Contains(strings.ToLower(gameVariant), "slayer") {
		return 0, nil
	}

	snapshots := analysis.BuildScoreSnapshots(events)
	return analysis.ComputeDominanceFlag(
		snapshots, myTeamID, outcome,
		"standard", false, false, matchID,
	), nil
}

// computeCTFDominanceFlag construit la courbe de score d'un match CTF depuis les
// captures (match_objective_events) et délègue à ComputeDominanceFlag. Retourne 0
// si aucune capture exploitable n'est disponible (courbe vide).
func computeCTFDominanceFlag(
	ctx context.Context, db *sql.DB, matchID string, myTeamID, outcome int,
) (int, error) {
	events, err := loadCaptureEvents(ctx, db, matchID)
	if err != nil {
		return 0, nil // non critique : pas de courbe -> pas de flag
	}
	snapshots := analysis.BuildObjectiveScoreSnapshots(events)
	if len(snapshots) == 0 {
		return analysis.DominanceFlagNone, nil
	}
	return analysis.ComputeDominanceFlag(
		snapshots, myTeamID, outcome,
		"standard", false, false, matchID,
	), nil
}

// loadTeamScoresOrKillSums retourne (team0, team1) en privilégiant les scores
// d'équipe du registry ; à défaut (Halo 5 ne peuple pas team_*_score) la somme des
// kills par équipe depuis match_participants (kills = score pour le Slayer/Arena).
func loadTeamScoresOrKillSums(ctx context.Context, db *sql.DB, matchID string) (int, int, bool) {
	var t0, t1 sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = ? LIMIT 1`,
		matchID).Scan(&t0, &t1)
	if err == nil && t0.Valid && t1.Valid && (t0.Int64 != 0 || t1.Int64 != 0) {
		return int(t0.Int64), int(t1.Int64), true
	}
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(team_id, 0) AS team, COALESCE(SUM(kills), 0) AS k
FROM match_participants WHERE match_id = ? GROUP BY team_id`, matchID)
	if err != nil {
		return 0, 0, false
	}
	defer rows.Close()
	var k0, k1 int
	found := false
	for rows.Next() {
		var team, k int
		if err := rows.Scan(&team, &k); err != nil {
			return 0, 0, false
		}
		found = true
		switch team {
		case 0:
			k0 = k
		case 1:
			k1 = k
		}
	}
	if !found {
		return 0, 0, false
	}
	return k0, k1, true
}

// loadMyTeamAndOutcome charge team_id et outcome du joueur pour un match.
func loadMyTeamAndOutcome(ctx context.Context, db *sql.DB, matchID, xuid string) (int, int, error) {
	row := db.QueryRowContext(ctx, `
SELECT COALESCE(team_id, 0) AS team_id, COALESCE(outcome, 0) AS outcome
FROM match_participants
WHERE match_id = ? AND xuid = ?
LIMIT 1`, matchID, xuid)

	var teamID, outcome int
	if err := row.Scan(&teamID, &outcome); err != nil {
		return 0, 0, err
	}
	return teamID, outcome, nil
}

// loadSteaktacularByTeam retourne team_id → nombre de Steaktacular gagnées dans le match.
// Title-aware (C6) : un titre sans médaille killing-spree câblée (≠ Infinite) skippe
// la requête et retombe sur la marge de score (computeMatchDominanceFlag étape 1bis).
func loadSteaktacularByTeam(ctx context.Context, db *sql.DB, matchID string) (map[int]int, error) {
	medalID, ok := steaktacularMedalIDForTitle(ctxkeys.TitleSlug(ctx))
	if !ok {
		return map[int]int{}, nil
	}
	rows, err := db.QueryContext(ctx, `
SELECT mp.team_id, SUM(me.count) AS total
FROM medals_earned me
JOIN match_participants mp
    ON mp.match_id = me.match_id AND mp.xuid = me.xuid
WHERE me.match_id = ? AND me.medal_name_id = ?
GROUP BY mp.team_id`, matchID, medalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var teamID, total int
		if err := rows.Scan(&teamID, &total); err != nil {
			return nil, err
		}
		result[teamID] = total
	}
	return result, rows.Err()
}

// loadGameVariant charge le nom du game variant depuis match_registry.
func loadGameVariant(ctx context.Context, db *sql.DB, matchID string) (string, error) {
	row := db.QueryRowContext(ctx, `
SELECT COALESCE(game_variant_name, '') FROM match_registry WHERE match_id = ? LIMIT 1`, matchID)
	var name string
	return name, row.Scan(&name)
}

// loadKillEventsWithTeam charge les kill events avec le team_id de l'acteur.
func loadKillEventsWithTeam(ctx context.Context, db *sql.DB, matchID string) ([]analysis.KillEvent, error) {
	rows, err := db.QueryContext(ctx, `
SELECT he.time_ms, COALESCE(mp.team_id, 0) AS team_id
FROM highlight_events he
JOIN match_participants mp
    ON mp.match_id = he.match_id AND mp.xuid = he.xuid
WHERE he.match_id = ?
  AND he.event_type = 'kill'
  AND he.xuid IS NOT NULL
ORDER BY he.time_ms ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []analysis.KillEvent
	for rows.Next() {
		var e analysis.KillEvent
		if err := rows.Scan(&e.TimeMS, &e.TeamID); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// loadCaptureEvents charge les events de capture CTF (match_objective_events) avec
// un team_id et un time_ms résolus, mappés vers analysis.ObjectiveScoreEvent.
// Value est COALESCE à 1 (1 point par capture de drapeau). Ordre par seq pour
// suivre l'ordre de décodage du film.
func loadCaptureEvents(ctx context.Context, db *sql.DB, matchID string) ([]analysis.ObjectiveScoreEvent, error) {
	rows, err := db.QueryContext(ctx, `
SELECT time_ms, team_id, COALESCE(value, 1) AS value
FROM match_objective_events
WHERE match_id = ?
  AND event_type = 'capture'
  AND team_id IS NOT NULL
  AND time_ms IS NOT NULL
ORDER BY seq ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []analysis.ObjectiveScoreEvent
	for rows.Next() {
		var timeMS int64
		var teamID, value int
		if err := rows.Scan(&timeMS, &teamID, &value); err != nil {
			return nil, err
		}
		events = append(events, analysis.ObjectiveScoreEvent{
			TimeMS: timeMS,
			TeamID: teamID,
			Value:  value,
		})
	}
	return events, rows.Err()
}
