// Package sync — citations.go : backfill des citations calculées par match.
//
// Flow :
//  1. Charger les règles complètes depuis metadata.citation_mappings (Q40)
//  2. Pour chaque matchID :
//     a. medals depuis shared.medals_earned
//     b. stats depuis shared.match_participants (+ "weapon_kills:<name>" via weapon_labels)
//     c. awards depuis player.personal_score_awards
//     d. events depuis shared.highlight_events
//  3. ComputeFullMatchCitations (analysis) → deltas
//  4. Upsert dans player.match_citations (idempotent : DO NOTHING si déjà présent)
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// BackfillMatchCitations calcule et persiste les citations pour une liste de matchs.
//
// metadataDB : connexion à metadata.duckdb (lecture citation_mappings, weapon_labels).
// sharedDB   : connexion à shared_matches_v2.duckdb (medals, stats, events, weapon_kills).
// playerDB   : connexion à stats.duckdb du joueur (awards en lecture, match_citations en écriture).
// pveDB      : connexion RO à shared_pve.duckdb (stats Firefight, type de citation pve_stat).
//
//	Nil accepté (titre sans Firefight / DB absente) → dégradation gracieuse :
//	les citations pve_stat restent à 0, aucune erreur (cf. loadPveStats).
//
// xuid       : identifiant Xbox du joueur.
// matchIDs   : liste des match_id à traiter (triés en interne par start_time ASC).
func BackfillMatchCitations(
	ctx context.Context,
	metadataDB, sharedDB, playerDB, pveDB *sql.DB,
	xuid string,
	matchIDs []string,
) error {
	if len(matchIDs) == 0 {
		return nil
	}

	mappings, err := loadFullCitationMappings(ctx, metadataDB)
	if err != nil {
		return fmt.Errorf("BackfillMatchCitations: mappings: %w", err)
	}
	if len(mappings) == 0 {
		slog.InfoContext(ctx, "BackfillMatchCitations: aucun mapping — skip")
		return nil
	}

	weaponNames, err := loadWeaponNames(ctx, metadataDB)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: weapon_names non chargés", "err", err)
		weaponNames = map[uint64]string{}
	}

	// Tri chrono pour que le cumulPre soit correct entre les matchs du batch.
	sorted, err := sortMatchIDsChrono(ctx, sharedDB, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: sort chrono failed, ordre non garanti", "err", err)
		sorted = matchIDs
	}

	// Baseline : somme cumulée des citations pour les matchs hors du batch courant.
	// Pour un sync incrémental (nouveaux matchs), ces matchs sont antérieurs → correct.
	// Pour un recompute total (all matchIDs), la baseline est {} (tout est exclu).
	cumulPre, err := loadCumulExcluding(ctx, playerDB, matchIDs)
	if err != nil {
		return fmt.Errorf("BackfillMatchCitations: cumulPre baseline: %w", err)
	}

	slog.InfoContext(ctx, "citations: traitement batch",
		"xuid", xuid, "match_count", len(sorted), "baseline_citations", len(cumulPre))

	written, skipped := 0, 0
	for _, matchID := range sorted {
		citCtx, err := buildCitationContext(ctx, sharedDB, playerDB, pveDB, weaponNames, xuid, matchID)
		if err != nil {
			slog.WarnContext(ctx, "BackfillMatchCitations: context", "match_id", matchID, "err", err)
			skipped++
			continue
		}

		deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{
			Ctx:      citCtx,
			CumulPre: cumulPre,
		}, mappings)

		// Append-only #23046 (Phase 2) : plus de DELETE avant réécriture. La
		// réécriture (writeCitations) alloue une nouvelle génération qui supersède
		// la précédente via la vue match_citations_latest (recompute soustractif
		// préservé : la nouvelle génération porte EXACTEMENT le nouvel ensemble).

		// Phase 4 (fix recompute citations) : NE PAS poser le sentinel "_processed"
		// si 0 delta provient d'events pas encore chargés (film retardé). Sinon le
		// match entre dans match_citations et sort définitivement de
		// selectMatchesForCitations(force=false) (LEFT JOIN IS NULL) → jamais
		// recalculé quand les events arrivent → citations vides en permanence
		// (cause racine onglet Détails). On laisse le match candidat : le events
		// heal (post-sync, avant citations) charge les events, et le prochain cycle
		// citations recalcule. events_loaded distingue "vraiment 0 citation"
		// (events présents) de "0 par manque d'events" (film pas encore là).
		//
		// ÉTAT TERMINAL (2026-08-25) : le pari « les events finiront par arriver »
		// est FAUX pour un match annulé par les serveurs (film réduit à une coquille,
		// 0 event extractible). Un tel match restait candidat à vie et était retraité
		// à chaque cycle de sync, sans fin ni effet — charge qui croît d'une boucle
		// par match annulé. Passé citationsTerminalNoEventsAge, l'absence d'events
		// devient un état terminal (isCitationsTerminalNoEvents, plus bas dans ce
		// fichier) : la 3e condition tombe et le jeton EST posé par writeCitations
		// ci-dessous. Le jeton n'est PAS une impasse : le chemin
		// force=true (recompute) sélectionne tous les matchs sans consulter
		// match_citations — le LEFT JOIN IS NULL ne vit que dans la branche
		// force=false de selectMatchesForCitations — donc un match jetonné dont les
		// events arriveraient plus tard reste rattrapable par un recompute.
		if len(deltas) == 0 && !isEventsLoaded(ctx, sharedDB, matchID) &&
			!isCitationsTerminalNoEvents(ctx, sharedDB, matchID) {
			slog.DebugContext(ctx, "citations: 0 delta + events non chargés → skip sentinel (match reste candidat)",
				"match_id", matchID)
			skipped++
			continue
		}

		if err := writeCitations(ctx, playerDB, matchID, deltas); err != nil {
			slog.WarnContext(ctx, "BackfillMatchCitations: write", "match_id", matchID, "err", err)
			skipped++
			continue
		}

		written++
		// Mise à jour incrémentale du cumulPre pour le match suivant dans le batch.
		for _, d := range deltas {
			cumulPre[d.NameNorm] += d.Value
		}
	}
	slog.InfoContext(ctx, "citations: batch terminé",
		"xuid", xuid, "written", written, "skipped", skipped)
	return nil
}

// buildCitationContext charge toutes les données du match pour le moteur.
func buildCitationContext(
	ctx context.Context,
	sharedDB, playerDB, pveDB *sql.DB,
	weaponNames map[uint64]string,
	xuid, matchID string,
) (domain.CitationContext, error) {
	medals, err := loadMedalsForMatch(ctx, sharedDB, matchID, xuid)
	if err != nil {
		return domain.CitationContext{}, fmt.Errorf("medals: %w", err)
	}

	stats, outcome, playlist, gameVariant, isFirefight, err :=
		loadMatchStats(ctx, sharedDB, matchID, xuid)
	if err != nil {
		return domain.CitationContext{}, fmt.Errorf("stats: %w", err)
	}

	weaponKills, err := loadWeaponKills(ctx, sharedDB, weaponNames, matchID, xuid)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: weapon_kills", "match_id", matchID, "err", err)
	}
	for k, v := range weaponKills {
		stats["weapon_kills:"+k] = float64(v)
	}

	// BUG A (I7, 2026-07-24) : les stats PvE Firefight n'étaient JAMAIS chargées →
	// les citations de type pve_stat (grunt_kills, elite_kills, boss_kills,
	// total_enemy_kills, ...) restaient toujours à 0. On les injecte ici dans le
	// même map Stats que le moteur consulte (dispatchFull, cas pve_stat).
	// Dégradation gracieuse : pveDB nil / match non-Firefight / DB périmée →
	// pveStats vide, WARN best-effort, aucune erreur fatale.
	pveStats, err := loadPveStats(ctx, pveDB, matchID, xuid)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: pve_stats", "match_id", matchID, "err", err)
	}
	for k, v := range pveStats {
		stats[k] = v
	}

	// Plomberie citations objective_stat (v7.2) : stats objectifs (CTF/Zones/Oddball)
	// depuis shared.match_objective_stats_latest (MÊME sharedDB que match_participants).
	// Les citations objective_stat (charge/zone_captures, got_you/flag_returns,
	// stakeholder/zone_secures, flag_carrier_hunter/flag_carriers_killed) les consultent
	// via Stats (dispatchFull, cas objective_stat). Best-effort intégral : match
	// non-objectif (Slayer) / vue absente → aucune stat, aucune erreur (cf. loadObjectiveStats).
	for k, v := range loadObjectiveStats(ctx, sharedDB, matchID, xuid) {
		stats[k] = v
	}

	awards, err := loadAwards(ctx, playerDB, matchID, xuid)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: awards", "match_id", matchID, "err", err)
		awards = map[string]int{}
	}

	events, err := loadHighlightEvents(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: events", "match_id", matchID, "err", err)
		events = nil
	}

	return domain.CitationContext{
		Stats:       stats,
		Medals:      medals,
		Awards:      awards,
		Events:      events,
		PlayerXUID:  xuid,
		Playlist:    strings.ToLower(playlist),
		GameVariant: strings.ToLower(gameVariant),
		Outcome:     outcome,
		IsFirefight: isFirefight,
	}, nil
}

// loadFullCitationMappings charge tous les champs de citation_mappings (Q40).
func loadFullCitationMappings(ctx context.Context, db *sql.DB) ([]domain.CitationFullMapping, error) {
	const q = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(mapping_type, 'medal') AS mapping_type,
    medal_id,
    medal_ids,
    stat_name,
    award_name,
    custom_function,
    composite_children,
    tier_targets
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY citation_name_norm`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.CitationFullMapping
	for rows.Next() {
		var m domain.CitationFullMapping
		if err := rows.Scan(
			&m.NameNorm, &m.NameDisplay, &m.MappingType,
			&m.MedalID, &m.MedalIDs, &m.StatName, &m.AwardName,
			&m.CustomFunction, &m.CompositeChildren, &m.TierTargets,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// loadWeaponNames charge la table weapon_labels depuis metadata.duckdb.
// Retourne weapon_id → name_en (canonique EN).
func loadWeaponNames(ctx context.Context, db *sql.DB) (map[uint64]string, error) {
	const q = `SELECT weapon_id, name_en FROM weapon_labels`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64]string)
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}
	return result, rows.Err()
}

// loadMedalsForMatch charge les médailles d'un joueur pour un match depuis shared.
func loadMedalsForMatch(ctx context.Context, db *sql.DB, matchID, xuid string) (map[int64]int, error) {
	const q = `
SELECT medal_name_id, count
FROM medals_earned
WHERE match_id = ? AND xuid = ?`

	rows, err := db.QueryContext(ctx, q, matchID, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	medals := make(map[int64]int)
	for rows.Next() {
		var id int64
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		medals[id] += count
	}
	return medals, rows.Err()
}

// loadMatchStats charge les stats numériques du joueur pour le match depuis shared.match_participants.
// Retourne (stats map, outcome, playlist, game_variant, is_firefight, error).
func loadMatchStats(
	ctx context.Context,
	db *sql.DB,
	matchID, xuid string,
) (map[string]float64, int, string, string, bool, error) {
	const q = `
SELECT
    COALESCE(mp.kills, 0)            AS kills,
    COALESCE(mp.deaths, 0)           AS deaths,
    COALESCE(mp.assists, 0)          AS assists,
    COALESCE(mp.kda, 0.0)            AS kda,
    COALESCE(mp.score, 0)            AS score,
    COALESCE(mp.damage_dealt, 0.0)   AS damage_dealt,
    COALESCE(mp.damage_taken, 0.0)   AS damage_taken,
    COALESCE(mp.accuracy, 0.0)       AS accuracy,
    COALESCE(mp.headshot_kills, 0)   AS headshot_kills,
    COALESCE(mp.melee_kills, 0)      AS melee_kills,
    COALESCE(mp.power_weapon_kills, 0) AS power_weapon_kills,
    COALESCE(mp.grenade_kills, 0)    AS grenade_kills,
    COALESCE(mp.max_killing_spree, 0)  AS max_killing_spree,
    COALESCE(mp.avg_life_seconds, 0.0) AS avg_life_seconds,
    COALESCE(mp.outcome, 0)          AS outcome,
    COALESCE(r.playlist_name, '')    AS playlist_name,
    COALESCE(r.game_variant_name, '') AS game_variant_name,
    COALESCE(r.is_firefight, FALSE)  AS is_firefight
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
WHERE mp.match_id = ? AND mp.xuid = ?
LIMIT 1`

	row := db.QueryRowContext(ctx, q, matchID, xuid)

	var (
		kills, deaths, assists, score, headshotKills, meleeKills int
		powerWeaponKills, grenadeKills, maxKillingSpree          int
		kda, damagDealt, damageTaken, accuracy, avgLife          float64
		outcome                                                  int
		playlist, gameVariant                                    string
		isFirefight                                              bool
	)

	if err := row.Scan(
		&kills, &deaths, &assists, &kda, &score,
		&damagDealt, &damageTaken, &accuracy, &headshotKills,
		&meleeKills, &powerWeaponKills, &grenadeKills, &maxKillingSpree, &avgLife,
		&outcome, &playlist, &gameVariant, &isFirefight,
	); err != nil {
		return nil, 0, "", "", false, err
	}

	// BUG B (I7, 2026-07-24) : grenade_kills était omis du SELECT alors que la
	// colonne existe et est peuplée → citation « Regarde maman, sans goupille »
	// (look_ma_no_pin, stat_name grenade_kills) restait toujours à 0. Ajouté ici.
	stats := map[string]float64{
		"kills":              float64(kills),
		"deaths":             float64(deaths),
		"assists":            float64(assists),
		"kda":                kda,
		"score":              float64(score),
		"damage_dealt":       damagDealt,
		"damage_taken":       damageTaken,
		MetricKeyAccuracy:    accuracy,
		"headshot_kills":     float64(headshotKills),
		"melee_kills":        float64(meleeKills),
		"power_weapon_kills": float64(powerWeaponKills),
		"grenade_kills":      float64(grenadeKills),
		"max_killing_spree":  float64(maxKillingSpree),
		"avg_life_seconds":   avgLife,
	}
	return stats, outcome, playlist, gameVariant, isFirefight, nil
}

// loadPveStats charge les stats PvE Firefight du joueur pour le match depuis
// shared_pve.pve_match_stats_latest (vue append-only, ADR 0026 — jamais la table
// brute). Les clés retournées correspondent aux stat_name des citations de type
// pve_stat : les kills par type d'ennemi portent le nom de leur colonne
// (grunt_kills, elite_kills, ...) ; total_enemy_kills est l'alias de la colonne
// total_kills (nom réel côté shared_pve).
//
// Dégradation gracieuse (BUG A, I7) :
//   - pveDB nil (titre sans Firefight / DB absente) → (nil, nil) ;
//   - match non-Firefight (aucune ligne PvE) → (nil, nil), cas normal ;
//   - vue absente / autre erreur SQL → (nil, err) : le caller logge WARN et
//     poursuit (les citations pve_stat restent à 0, pas d'échec du pipeline).
func loadPveStats(ctx context.Context, pveDB *sql.DB, matchID, xuid string) (map[string]float64, error) {
	if pveDB == nil {
		return nil, nil
	}
	const q = `
SELECT
    COALESCE(grunt_kills, 0),
    COALESCE(elite_kills, 0),
    COALESCE(jackal_kills, 0),
    COALESCE(hunter_kills, 0),
    COALESCE(brute_kills, 0),
    COALESCE(skimmer_kills, 0),
    COALESCE(sentinel_kills, 0),
    COALESCE(marine_kills, 0),
    COALESCE(boss_kills, 0),
    COALESCE(total_enemy_kills, 0)
FROM pve_match_stats_latest
WHERE match_id = ? AND xuid = ?
LIMIT 1`

	var grunt, elite, jackal, hunter, brute, skimmer, sentinel, marine, boss, total int
	err := pveDB.QueryRowContext(ctx, q, matchID, xuid).Scan(
		&grunt, &elite, &jackal, &hunter, &brute,
		&skimmer, &sentinel, &marine, &boss, &total,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // match non-Firefight : aucune ligne PvE, comportement normal
	}
	if err != nil {
		return nil, err
	}
	return map[string]float64{
		"grunt_kills":       float64(grunt),
		"elite_kills":       float64(elite),
		"jackal_kills":      float64(jackal),
		"hunter_kills":      float64(hunter),
		"brute_kills":       float64(brute),
		"skimmer_kills":     float64(skimmer),
		"sentinel_kills":    float64(sentinel),
		"marine_kills":      float64(marine),
		"boss_kills":        float64(boss),
		"total_enemy_kills": float64(total),
	}, nil
}

// objectiveStatColumns — colonnes de match_objective_stats (schéma shared-core CTF/
// Zones/Oddball, cf. steps_shared_objective_stats). Chaque colonne devient une clé du
// map Stats (nom identique) → stat_name des citations objective_stat. Slice unique
// (source de vérité pour le SELECT ET les clés du map — pas de désync possible).
var objectiveStatColumns = []string{
	// CTF
	"flag_captures", "flag_capture_assists", "flag_grabs", "flag_secures",
	"flag_steals", "flag_returns", "flag_carriers_killed", "flag_returners_killed",
	"kills_as_flag_carrier", "kills_as_flag_returner", "time_as_flag_carrier_seconds",
	// Zones (Strongholds + KOTH)
	"zone_captures", "zone_secures", "zone_offensive_kills", "zone_defensive_kills",
	"zone_scoring_ticks", "time_in_zones_seconds",
	// Oddball
	"kills_as_skull_carrier", "skull_carriers_killed", "skull_grabs",
	"skull_scoring_ticks", "time_as_skull_carrier_seconds", "longest_time_as_skull_carrier_seconds",
}

// loadObjectiveStats charge les stats objectifs du joueur pour le match depuis
// shared.match_objective_stats_latest (vue append-only, ADR 0026 — jamais la table
// brute). Les clés portent le nom de leur colonne (zone_captures, flag_returns,
// flag_carriers_killed, ...) → stat_name des citations objective_stat. La table vit
// sur le MÊME sharedDB que match_participants (shared-core) — pas de handle dédié.
// Toutes les colonnes sont castées DOUBLE côté SQL → scan homogène en float64 (les
// *_seconds restent des secondes, tronquées à l'entier par le moteur).
//
// Best-effort intégral (plomberie citations objective_stat, v7.2) : retourne toujours
// une map (nil = aucune stat), jamais d'erreur. Cas dégradés → (nil) :
//   - sharedDB nil (titre sans handle) ;
//   - match sans ligne objectif (mode Slayer / non-objectif : ErrNoRows) ;
//   - vue absente (DB non migrée) / autre erreur SQL → Debug best-effort.
//     Les citations objective_stat restent alors à 0, jamais d'échec du pipeline.
func loadObjectiveStats(ctx context.Context, sharedDB *sql.DB, matchID, xuid string) map[string]float64 {
	if sharedDB == nil {
		return nil
	}
	sel := make([]string, len(objectiveStatColumns))
	for i, c := range objectiveStatColumns {
		sel[i] = "COALESCE(CAST(" + c + " AS DOUBLE), 0)"
	}
	q := "SELECT " + strings.Join(sel, ", ") +
		" FROM match_objective_stats_latest WHERE match_id = ? AND xuid = ? LIMIT 1"

	vals := make([]float64, len(objectiveStatColumns))
	dst := make([]any, len(objectiveStatColumns))
	for i := range vals {
		dst[i] = &vals[i]
	}
	err := sharedDB.QueryRowContext(ctx, q, matchID, xuid).Scan(dst...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // match non-objectif : aucune ligne, comportement normal
	}
	if err != nil {
		// vue absente sur une DB non migrée, ou autre erreur SQL : best-effort.
		slog.DebugContext(ctx, "citations: objective_stats indisponible (best-effort)",
			"match_id", matchID, "err", err)
		return nil
	}
	out := make(map[string]float64, len(objectiveStatColumns))
	for i, c := range objectiveStatColumns {
		out[c] = vals[i]
	}
	return out
}

// loadWeaponKills charge les kills par arme (effective_weapon_id → canonical name_en) depuis shared.
func loadWeaponKills(
	ctx context.Context,
	db *sql.DB,
	weaponNames map[uint64]string,
	matchID, xuid string,
) (map[string]int, error) {
	const q = `
SELECT CAST(effective_weapon_id AS UBIGINT) AS wid, COUNT(*) AS kills
FROM v_weapon_kills
WHERE match_id = ? AND xuid = ?
  AND effective_weapon_id NOT IN (0, 1, 2)
GROUP BY effective_weapon_id`

	rows, err := db.QueryContext(ctx, q, matchID, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var wid uint64
		var kills int
		if err := rows.Scan(&wid, &kills); err != nil {
			return nil, err
		}
		name, ok := weaponNames[wid]
		if !ok {
			continue
		}
		result[name] += kills
	}
	return result, rows.Err()
}

// loadAwards charge les awards du joueur pour un match depuis player.personal_score_awards.
func loadAwards(ctx context.Context, db *sql.DB, matchID, xuid string) (map[string]int, error) {
	const q = `
SELECT award_name, COALESCE(SUM(award_count), 0) AS total
FROM personal_score_awards_latest
WHERE match_id = ? AND xuid = ?
GROUP BY award_name`

	rows, err := db.QueryContext(ctx, q, matchID, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	awards := make(map[string]int)
	for rows.Next() {
		var name string
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		awards[name] = total
	}
	return awards, rows.Err()
}

// loadHighlightEvents charge les events du match depuis shared.highlight_events.
func loadHighlightEvents(ctx context.Context, db *sql.DB, matchID string) ([]domain.CitationEventRow, error) {
	const q = `
SELECT event_type, COALESCE(time_ms, 0) AS time_ms, COALESCE(xuid, '') AS xuid
FROM highlight_events
WHERE match_id = ?
ORDER BY time_ms ASC`

	rows, err := db.QueryContext(ctx, q, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.CitationEventRow
	for rows.Next() {
		var e domain.CitationEventRow
		if err := rows.Scan(&e.EventType, &e.TimeMS, &e.XUID); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// isEventsLoaded retourne true si highlight_events sont définitivement traités
// pour ce match (match_registry.events_loaded). Best-effort : toute erreur de
// lecture (table absente en tests minimaux, match inconnu) → true pour préserver
// le comportement legacy (poser le sentinel). Sert à décider, en Phase 4, si un
// match à 0 citation peut recevoir le sentinel "_processed" (events présents) ou
// doit rester candidat (events pas encore chargés — film retardé). Depuis l'état
// terminal (2026-08-25), « rester candidat » est en plus borné dans le temps :
// cf. isCitationsTerminalNoEvents ci-dessous.
func isEventsLoaded(ctx context.Context, sharedDB *sql.DB, matchID string) bool {
	if sharedDB == nil {
		return true
	}
	var loaded sql.NullBool
	err := sharedDB.QueryRowContext(ctx,
		`SELECT events_loaded FROM match_registry WHERE match_id = ?`, matchID).Scan(&loaded)
	if err != nil {
		return true // best-effort : ne pas bloquer le pipeline
	}
	return loaded.Valid && loaded.Bool
}

// ─── État terminal : les events n'arriveront jamais ──────────────────────────
//
// Contexte (constaté en prod le 2026-08-25) : la règle Phase 4 ci-dessus laisse
// candidat tout match à 0 citation dont les events ne sont pas chargés, au motif
// qu'ils finiront par arriver. Pour un match ANNULÉ par les serveurs, ils
// n'arrivent jamais — le film est une coquille (chunk non vide mais 0 event
// extractible) — et le match était re-sélectionné, recalculé puis re-rejeté à
// chaque cycle de sync, indéfiniment, avec ses WARN collatéraux : une boucle
// perpétuelle de plus par nouveau match annulé.
//
// Ce bloc vit dans citations.go, et non dans un fichier dédié, parce que le
// ratchet K3c (archlint/sync_root_freeze_test.go, ADR 0027) gèle le nombre de
// fichiers à la racine du god-package sync/ : le neuf ne peut y ajouter aucun
// fichier. La longueur de citations.go est de la dette gelée antérieure.

// citationsTerminalNoEventsAge — âge au-delà duquel un match à 0 citation dont
// les events ne sont pas chargés est déclaré en état terminal.
//
// 7 jours : un film Theater est publié dans les heures qui suivent la fin du
// match, ou jamais. Passé une semaine il ne reste que deux causes possibles —
// match annulé par les serveurs (film-coquille) ou film indisponible — et
// l'attente est perpétuelle dans les deux cas. Le seuil est volontairement large
// devant le délai réel d'arrivée (heures) : une panne d'API ou un arrêt du
// watcher de plusieurs jours ne doit pas jetonner un match encore récupérable.
const citationsTerminalNoEventsAge = 7 * 24 * time.Hour

// matchAge retourne l'âge du match (maintenant − début du match) lu depuis
// shared.match_registry.
//
// Le début du match passe par le fragment timezone CANONIQUE (règle CLAUDE.md
// n°8, analysis.SQLStartTimeCanonical) — jamais start_time brut : les imports
// OpenSpartan portent un start_time naïf décalé, et un âge calculé dessus
// jetonnerait (ou refuserait de jetonner) au mauvais moment.
//
// Retourne une ERREUR — jamais un âge par défaut — dès que l'âge est
// indéterminable : sharedDB nil, match absent du registre, colonnes illisibles,
// horodatages NULL. Le caller doit alors choisir l'échec sûr.
func matchAge(ctx context.Context, sharedDB *sql.DB, matchID string) (time.Duration, error) {
	if sharedDB == nil {
		return 0, errors.New("matchAge: sharedDB nil")
	}
	q := `SELECT ` + analysis.SQLStartTimeCanonical("") + ` FROM match_registry WHERE match_id = ?`
	var start sql.NullTime
	if err := sharedDB.QueryRowContext(ctx, q, matchID).Scan(&start); err != nil {
		return 0, fmt.Errorf("matchAge %s: %w", matchID, err)
	}
	if !start.Valid {
		return 0, fmt.Errorf("matchAge %s: début de match canonique NULL", matchID)
	}
	return time.Since(start.Time), nil
}

// isCitationsTerminalNoEvents décide si un match à 0 citation dont les events ne
// sont PAS chargés doit malgré tout recevoir le jeton "_processed".
//
// Appelée UNIQUEMENT dans la branche rare (0 delta ET events absents), par
// court-circuit du && côté BackfillMatchCitations : le chemin chaud ne paie
// aucune requête supplémentaire.
//
// Tempérament symétrique de isEventsLoaded, mais d'échec sûr INVERSE :
// isEventsLoaded répond true quand elle ne sait pas (poser le jeton = legacy) ;
// ici, ne pas savoir doit laisser le match candidat — un cycle de plus coûte
// infiniment moins qu'un match jetonné à tort, dont les citations resteraient
// vides jusqu'au prochain recompute force=true. Toute lecture d'âge en échec est
// donc loguée en WARN et retourne false.
func isCitationsTerminalNoEvents(ctx context.Context, sharedDB *sql.DB, matchID string) bool {
	age, err := matchAge(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "citations: âge du match illisible — match laissé candidat (0 delta, events absents)",
			"match_id", matchID, "err", err)
		return false
	}
	if age < citationsTerminalNoEventsAge {
		return false
	}
	slog.InfoContext(ctx, "citations: état terminal — jeton _processed posé",
		"match_id", matchID,
		"age_days", int(age.Hours()/24),
		"seuil_jours", int(citationsTerminalNoEventsAge.Hours()/24),
		"raison", "events jamais arrivés — état terminal")
	return true
}

// writeCitations écrit les deltas dans match_citations (idempotent).
// Quand deltas est vide (match sans citation active), une row sentinel
// "_processed" est insérée : cela sort le match du pool de
// selectMatchesForCitations et évite de le retraiter à chaque sync.
// NB Phase 4 : ce sentinel n'est posé (cas 0 delta) que si les events sont
// chargés (isEventsLoaded) OU si le match est assez vieux pour que leur absence
// soit un état terminal (isCitationsTerminalNoEvents) — décision prise par le
// caller BackfillMatchCitations.
func writeCitations(ctx context.Context, db *sql.DB, matchID string, deltas []domain.CitationMatchDelta) error {
	// Append-only #23046 (Phase 2) : plus de PK composite ni ON CONFLICT ni DELETE
	// préalable. Chaque réécriture d'un match alloue UNE génération
	// (match_citations_generation_seq), partagée par toutes ses rows ; la vue
	// match_citations_latest ne lit que la génération MAX par match_id. La
	// sentinelle '_processed' (0 citation active) fait partie de la génération.
	var gen int64
	if err := db.QueryRowContext(ctx, `SELECT nextval('match_citations_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("writeCitations gen %s: %w", matchID, err)
	}
	const q = `INSERT INTO match_citations (match_id, citation_name_norm, value, generation_id) VALUES (?, ?, ?, ?)`
	if len(deltas) == 0 {
		_, err := db.ExecContext(ctx, q, matchID, "_processed", 0, gen)
		return err
	}
	for _, d := range deltas {
		if _, err := db.ExecContext(ctx, q, matchID, d.NameNorm, d.Value, gen); err != nil {
			return fmt.Errorf("writeCitations %s/%s: %w", matchID, d.NameNorm, err)
		}
	}
	return nil
}

// sortMatchIDsChrono trie les match_ids par start_time ASC depuis shared.match_registry.
// Les match_ids absents du registre sont placés à la fin.
func sortMatchIDsChrono(ctx context.Context, db *sql.DB, matchIDs []string) ([]string, error) {
	if len(matchIDs) <= 1 {
		return matchIDs, nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT match_id FROM match_registry WHERE match_id IN (` +
		strings.Join(placeholders, ",") + `) ORDER BY start_time ASC`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sorted := make([]string, 0, len(matchIDs))
	seen := make(map[string]bool, len(matchIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sorted = append(sorted, id)
		seen[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range matchIDs {
		if !seen[id] {
			sorted = append(sorted, id)
		}
	}
	return sorted, nil
}

// loadCumulExcluding charge la somme cumulée des citations en excluant les matchIDs
// à recomputer. Sert de baseline pour le calcul incrémental du cumulPre.
func loadCumulExcluding(ctx context.Context, db *sql.DB, matchIDs []string) (map[string]int, error) {
	var (
		q    string
		args []any
	)
	if len(matchIDs) == 0 {
		q = `SELECT citation_name_norm, COALESCE(SUM(value), 0) FROM match_citations_latest GROUP BY citation_name_norm`
	} else {
		placeholders := make([]string, len(matchIDs))
		args = make([]any, len(matchIDs))
		for i, id := range matchIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		q = `SELECT citation_name_norm, COALESCE(SUM(value), 0) FROM match_citations_latest WHERE match_id NOT IN (` +
			strings.Join(placeholders, ",") + `) GROUP BY citation_name_norm`
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var name string
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		result[name] = total
	}
	return result, rows.Err()
}
