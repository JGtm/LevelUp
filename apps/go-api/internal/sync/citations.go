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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/citationsfilm"
	"levelup/go-api/internal/sync/debrief"
)

// BackfillMatchCitations calcule et persiste les citations pour une liste de matchs.
//
// metadataDB : connexion à metadata.duckdb (lecture citation_mappings, weapon_labels).
// sharedDB   : connexion à shared_matches_v2.duckdb (medals, stats, events, weapon_kills), tenue par l'appelant tout le lot (le post-sync passe par backfillCitationsLot).
// playerDB   : connexion à stats.duckdb du joueur (awards en lecture, match_citations en écriture).
// pve        : lecteur auto-réparant sur shared_pve.duckdb (stats Firefight, type
//
//	de citation pve_stat). RecoveringReader et NON *sql.DB : la boucle ci-dessous
//	dure le temps du batch et le handle peut être fermé sous ses pieds par un
//	post-sync concurrent (cf. OpenPveReadForCitations). Nil accepté (titre sans
//	Firefight / DB absente) → dégradation gracieuse : citations pve_stat à 0.
//
// xuid       : identifiant Xbox du joueur.
// matchIDs   : liste des match_id à traiter (triés en interne par start_time ASC).
func BackfillMatchCitations(
	ctx context.Context,
	metadataDB, sharedDB, playerDB *sql.DB,
	pve *duckdbpkg.RecoveringReader,
	xuid string,
	matchIDs []string,
	// killSource : le traducteur « source de degat -> cle du registre » du titre, quand il
	// en a un. VARIADIQUE et OPTIONNEL a dessein : la signature portait deja sept
	// parametres et huit sites d appel, dont cinq de test qui n ont aucune source de degat
	// a fournir. Absent ou nil = voie historique, byte-identique.
	killSource ...CitationWeaponSource,
) error {
	return backfillCitationsLot(ctx, lotCitations{
		metadataDB: metadataDB, playerDB: playerDB, shared: lireSharedTenu(sharedDB),
		pve: pve, xuid: xuid, matchIDs: matchIDs, killSource: killSource,
	})
}

// buildCitationContext charge toutes les données du match pour le moteur.
func buildCitationContext(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	pve *duckdbpkg.RecoveringReader,
	weaponNames map[uint64]string,
	source citationWeaponSource,
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

	// Titre a decodeur de film : les frags par arme viennent de la SOURCE DU DEGAT, celle
	// qui voit l epee et le marteau (bascule du 2026-09-01). Sinon, voie historique.
	var weaponKills map[string]int
	if source.actif() {
		weaponKills, err = loadWeaponKillsFromSource(ctx, sharedDB, source, matchID, xuid)
	} else {
		weaponKills, err = loadWeaponKills(ctx, sharedDB, weaponNames, matchID, xuid)
	}
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: frags par arme", "match_id", matchID, "err", err)
	}
	for k, v := range weaponKills {
		stats["weapon_kills:"+k] = float64(v)
	}

	// BUG A (I7, 2026-07-24) : les stats PvE Firefight n'étaient JAMAIS chargées →
	// les citations de type pve_stat (grunt_kills, elite_kills, boss_kills,
	// total_enemy_kills, ...) restaient toujours à 0. On les injecte ici dans le
	// même map Stats que le moteur consulte (dispatchFull, cas pve_stat).
	// Dégradation gracieuse : pve nil / match non-Firefight / DB périmée →
	// pveStats vide, WARN best-effort, aucune erreur fatale.
	pveStats, err := loadPveStats(ctx, pve, matchID, xuid)
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
	// Résumé d'usage du film (armes prises sur socle, frags sans bouclier) : même patron.
	for k, v := range citationsfilm.LoadFilmStats(ctx, sharedDB, matchID, xuid) {
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
			&m.NameNorm, &m.MappingType,
			&m.MedalID, &m.MedalIDs, &m.StatName, &m.AwardName,
			&m.CustomFunction, &m.CompositeChildren, &m.TierTargets,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// loadWeaponNames charge les noms d'armes de weapon_label_names depuis metadata.duckdb.
// Retourne weapon_id → nom dans la langue de repli (canonique EN) : c'est la clé des stats
// `weapon_kills:<nom>` des citations, pas un libellé d'affichage, donc jamais une langue de requête.
func loadWeaponNames(ctx context.Context, db *sql.DB) (map[uint64]string, error) {
	const q = `SELECT weapon_id, name FROM weapon_label_names WHERE lang = ?`
	rows, err := db.QueryContext(ctx, q, string(i18n.Fallback))
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
//   - pve nil (titre sans Firefight / DB absente) → (nil, nil) ;
//   - match non-Firefight (aucune ligne PvE) → (nil, nil), cas normal ;
//   - vue absente / autre erreur SQL → (nil, err) : le caller logge WARN et
//     poursuit (les citations pve_stat restent à 0, pas d'échec du pipeline).
//
// Lecture via RecoveringReader.Do et NON via un `*sql.DB` capturé au début du
// batch : cause racine des WARN « pve_stats: sql: database is closed » d'août
// 2026, détaillée dans OpenPveReadForCitations (citations_backfill.go).
func loadPveStats(ctx context.Context, pve *duckdbpkg.RecoveringReader, matchID, xuid string) (map[string]float64, error) {
	if pve == nil {
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
	err := pve.Do(ctx, func(db *sql.DB) error {
		return db.QueryRowContext(ctx, q, matchID, xuid).Scan(
			&grunt, &elite, &jackal, &hunter, &brute,
			&skimmer, &sentinel, &marine, &boss, &total,
		)
	})
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
// terminal (2026-08-25), « rester candidat » cesse quand le pipeline events rend
// son verdict : cf. isEventsEmptyDefinitive ci-dessous.
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

// ─── État terminal : le VERDICT du pipeline events, pas une devinette d'âge ──
//
// Contexte (constaté en prod le 2026-08-25) : la règle Phase 4 ci-dessus laisse
// candidat tout match à 0 citation dont les events ne sont pas chargés, au motif
// qu'ils finiront par arriver. Pour un match ANNULÉ par les serveurs, ils
// n'arrivent jamais — le film est une coquille (chunk récupéré et parsé sans
// erreur, mais 0 event exploitable) — et le match était re-sélectionné, recalculé
// puis re-rejeté à chaque cycle de sync, indéfiniment, avec ses WARN collatéraux :
// une boucle perpétuelle de plus par nouveau match annulé.
//
// La question « les events arriveront-ils encore ? » a DÉJÀ un propriétaire : le
// pipeline events. Y répondre ici par un seuil d'âge local produirait deux
// politiques concurrentes — et la nôtre serait la mauvaise, puisque
// defaultFilmRetryWindow vaut 30 JOURS (engine_highlight_events.go, révisé après
// les incidents RC-A/RC-E où 48 h avaient marqué définitivement absents des films
// encore disponibles), fenêtre qu'un opérateur peut même RALLONGER via
// LEVELUP_FILM_RETRY_WINDOW_HOURS. Tout seuil local plus court jetonnerait des
// matchs que le pipeline events retente encore — régression Phase 4 rouverte, en
// masse pendant une panne, et silencieuse (un jeton est indistinguable après coup).
//
// On lit donc le verdict au lieu de le deviner : match_registry.events_empty,
// posé par persist.MarkEventsEmptyDefinitive UNIQUEMENT sous isNoFilmDefinitive
// (donc au-delà de la fenêtre de retry, par construction). Les trois états du
// pipeline events sont alors couverts sans recouvrement :
//   - film définitivement absent (404 hors fenêtre) → MarkNoFilmDefinitive pose
//     events_loaded=TRUE → chemin sentinel EXISTANT, inchangé ;
//   - film-coquille (chunk lu, 0 event) → events_empty=TRUE → cette branche ;
//   - film encore attendu → aucun marqueur → le match reste candidat, retenté.
//
// Latence assumée : un match annulé boucle au pire le temps de la fenêtre de
// retry (~30 j, BORNÉ, contre l'infini d'avant), puis est jetonné au cycle
// citations suivant le verdict. Les deux pipelines ne se contredisent jamais.
//
// Ce bloc vit dans citations.go, et non dans un fichier dédié, parce que le
// ratchet K3c (archlint/sync_root_freeze_test.go, ADR 0027) gèle le nombre de
// fichiers à la racine du god-package sync/ : le neuf ne peut y ajouter aucun
// fichier. La longueur de citations.go est de la dette gelée antérieure.

// readEventsEmpty lit match_registry.events_empty pour un match. Retourne une
// ERREUR — jamais un défaut silencieux — quand le verdict est ILLISIBLE :
// sharedDB nil, colonne absente (DB non migrée, titre au schéma minimal), match
// hors registre. NULL n'est pas une erreur : c'est l'absence de verdict (le
// pipeline events n'a pas conclu), qui se lit false.
func readEventsEmpty(ctx context.Context, sharedDB *sql.DB, matchID string) (bool, error) {
	if sharedDB == nil {
		return false, errors.New("sharedDB nil")
	}
	var empty sql.NullBool
	if err := sharedDB.QueryRowContext(ctx,
		`SELECT events_empty FROM match_registry WHERE match_id = ?`, matchID).Scan(&empty); err != nil {
		return false, err
	}
	return empty.Valid && empty.Bool, nil
}

// isEventsEmptyDefinitive décide si un match à 0 citation dont les events ne sont
// PAS chargés doit malgré tout recevoir le jeton "_processed".
//
// Appelée UNIQUEMENT dans la branche rare (0 delta ET events absents), par
// court-circuit du && côté BackfillMatchCitations : le chemin chaud ne paie
// aucune requête supplémentaire.
//
// Tempérament symétrique de isEventsLoaded, mais d'échec sûr INVERSE :
// isEventsLoaded répond true quand elle ne sait pas (poser le jeton = legacy) ;
// ici, ne pas savoir doit laisser le match candidat — un cycle de plus coûte
// infiniment moins qu'un match jetonné à tort, dont les citations resteraient
// vides jusqu'au prochain recompute force=true. Tout verdict illisible est donc
// logué en WARN et retourne false. Effet de bord voulu : sur un titre dont le
// schéma ne porte pas events_empty, la branche est inerte et le comportement
// Phase 4 reste exactement celui d'avant.
func isEventsEmptyDefinitive(ctx context.Context, sharedDB *sql.DB, matchID string) bool {
	empty, err := readEventsEmpty(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "citations: verdict events_empty illisible — match laissé candidat (0 delta, events absents)",
			"match_id", matchID, "err", err)
		return false
	}
	return empty
}

// writeCitations écrit les deltas dans match_citations (idempotent).
// Quand deltas est vide (match sans citation active), une row sentinel
// "_processed" est insérée : cela sort le match du pool de
// selectMatchesForCitations et évite de le retraiter à chaque sync.
// NB Phase 4 : ce sentinel n'est posé (cas 0 delta) que si les events sont
// chargés (isEventsLoaded) OU si le pipeline events a rendu son verdict terminal
// sur le film (isEventsEmptyDefinitive) — décision prise par le caller
// BackfillMatchCitations.
func writeCitations(ctx context.Context, db *sql.DB, matchID string, deltas []domain.CitationMatchDelta) error {
	// Append-only #23645 (Phase 2) : plus de PK composite ni ON CONFLICT ni DELETE
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

// ── Lot de citations : lecture de la base partagée par segments ──────────────────────────────
//
// Le calcul des citations d'un lot lit la base partagée PAR SEGMENTS : un pour le tri
// chronologique, puis un par match. Entre deux matchs la base est rendue : un lecteur en vol empêche
// toute prise d'écriture (le provider attend son relâchement, au plus son délai de vidange, puis
// abandonne — ADR 0016), et un lot se compte en centaines de matchs après une recuisson des films.

// lotCitations : les entrées du calcul des citations d'un lot de matchs d'un joueur.
type lotCitations struct {
	metadataDB, playerDB *sql.DB
	shared               lireShared
	pve                  *duckdbpkg.RecoveringReader
	xuid                 string
	matchIDs             []string
	killSource           []CitationWeaponSource
}

// reglesCitations : ce que le calcul lit dans la metadata, une fois par lot.
type reglesCitations struct {
	mappings    []domain.CitationFullMapping
	weaponNames map[uint64]string
	source      citationWeaponSource
}

// chargerReglesCitations lit les règles du lot. nil sans erreur : aucun mapping, rien à calculer.
func chargerReglesCitations(ctx context.Context, metadataDB *sql.DB, killSource []CitationWeaponSource) (*reglesCitations, error) {
	mappings, err := loadFullCitationMappings(ctx, metadataDB)
	if err != nil {
		return nil, fmt.Errorf("BackfillMatchCitations: mappings: %w", err)
	}
	if len(mappings) == 0 {
		slog.InfoContext(ctx, "BackfillMatchCitations: aucun mapping — skip")
		return nil, nil
	}
	weaponNames, err := loadWeaponNames(ctx, metadataDB)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: weapon_names non chargés", "err", err)
		weaponNames = map[uint64]string{}
	}
	// Source de degat : armee seulement si le titre fournit un traducteur ET que la
	// metadata porte les noms canoniques par cle. Sans l un des deux, la voie historique
	// reste servie — degradation silencieuse ET reversible, jamais une mesure vide.
	r := &reglesCitations{mappings: mappings, weaponNames: weaponNames}
	if len(killSource) > 0 && killSource[0].Classifier != nil {
		names, nerr := loadWeaponKeyNames(ctx, metadataDB, killSource[0].TitleSlug)
		if nerr != nil {
			slog.WarnContext(ctx, "BackfillMatchCitations: noms par cle de registre non charges",
				"title", killSource[0].TitleSlug, "err", nerr)
		} else {
			r.source = citationWeaponSource{classifier: killSource[0].Classifier, keyNames: names}
		}
	}
	return r, nil
}

// trierLot rend les matchs du lot en ordre chronologique (cumulPre correct d'un match au
// suivant), dans un segment de lecture à lui. Tri impossible : ordre reçu, journalisé.
func (l lotCitations) trierLot(ctx context.Context) []string {
	var sorted []string
	err := l.shared(func(sharedDB *sql.DB) error {
		var serr error
		sorted, serr = sortMatchIDsChrono(ctx, sharedDB, l.matchIDs)
		return serr
	})
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: sort chrono failed, ordre non garanti", "err", err)
		return l.matchIDs
	}
	return sorted
}

// backfillCitationsLot calcule et persiste les citations du lot (contrat : BackfillMatchCitations).
// Un segment de lecture qui ne s'ouvre pas interrompt le lot : les matchs restants restent
// candidats et reviennent au cycle suivant, ceux déjà écrits le sont pour de bon.
func backfillCitationsLot(ctx context.Context, l lotCitations) error {
	if len(l.matchIDs) == 0 {
		return nil
	}
	regles, err := chargerReglesCitations(ctx, l.metadataDB, l.killSource)
	if err != nil || regles == nil {
		return err
	}
	sorted := l.trierLot(ctx)
	// Baseline : somme cumulée des citations pour les matchs hors du batch courant.
	// Pour un sync incrémental (nouveaux matchs), ces matchs sont antérieurs → correct.
	// Pour un recompute total (all matchIDs), la baseline est {} (tout est exclu).
	cumulPre, err := loadCumulExcluding(ctx, l.playerDB, l.matchIDs)
	if err != nil {
		return fmt.Errorf("BackfillMatchCitations: cumulPre baseline: %w", err)
	}
	slog.InfoContext(ctx, "citations: traitement batch",
		"xuid", l.xuid, "match_count", len(sorted), "baseline_citations", len(cumulPre))

	written, skipped := 0, 0
	for _, matchID := range sorted {
		var m matchCitations
		if err := l.shared(func(sharedDB *sql.DB) error {
			m = l.calculerMatch(ctx, sharedDB, regles, cumulPre, matchID)
			return nil
		}); err != nil {
			slog.WarnContext(ctx, "citations: base partagée indisponible — lot interrompu, matchs restants repris au cycle suivant",
				"xuid", l.xuid, "written", written, "remaining", len(sorted)-written-skipped, "err", err)
			return fmt.Errorf("BackfillMatchCitations: lecture shared: %w", err)
		}
		if !m.aEcrire {
			skipped++
			continue
		}
		if err := l.ecrireMatch(ctx, matchID, m, cumulPre); err != nil {
			slog.WarnContext(ctx, "BackfillMatchCitations: write", "match_id", matchID, "err", err)
			skipped++
			continue
		}
		written++
	}
	slog.InfoContext(ctx, "citations: batch terminé",
		"xuid", l.xuid, "written", written, "skipped", skipped)
	return nil
}

// matchCitations : le résultat de la lecture d'un match. aEcrire=false : le match reste
// candidat (contexte illisible, ou 0 delta en attendant son film).
type matchCitations struct {
	deltas           []domain.CitationMatchDelta
	terminalNoEvents bool
	aEcrire          bool
}

// calculerMatch lit le match dans la base partagée (segment de l'appelant) et calcule ses deltas.
func (l lotCitations) calculerMatch(
	ctx context.Context, sharedDB *sql.DB, r *reglesCitations, cumulPre map[string]int, matchID string,
) matchCitations {
	citCtx, err := buildCitationContext(ctx, sharedDB, l.playerDB, l.pve, r.weaponNames, r.source, l.xuid, matchID)
	if err != nil {
		slog.WarnContext(ctx, "BackfillMatchCitations: context", "match_id", matchID, "err", err)
		return matchCitations{}
	}
	deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{
		Ctx:      citCtx,
		CumulPre: cumulPre,
	}, r.mappings)

	// Append-only #23645 : pas de DELETE avant réécriture ; writeCitations alloue une génération
	// qui supersède la précédente via match_citations_latest (recompute soustractif préservé).
	//
	// Le sentinel "_processed" n'est PAS posé si 0 delta vient d'events pas encore chargés (film
	// retardé) : le match resterait sinon hors de selectMatchesForCitations(force=false) à vie,
	// citations vides en permanence. ÉTAT TERMINAL : un match annulé par les serveurs (film réduit
	// à une coquille, 0 event exploitable) n'aura jamais ses events ; le VERDICT du pipeline events
	// (match_registry.events_empty, isEventsEmptyDefinitive) lève alors la condition et le jeton
	// est posé. Le jeton n'est pas une impasse : force=true sélectionne tous les matchs sans
	// consulter match_citations, un match jetonné reste rattrapable par un recompute.
	m := matchCitations{deltas: deltas, aEcrire: true}
	if len(deltas) == 0 && !isEventsLoaded(ctx, sharedDB, matchID) {
		m.terminalNoEvents = isEventsEmptyDefinitive(ctx, sharedDB, matchID)
		if !m.terminalNoEvents {
			slog.DebugContext(ctx, "citations: 0 delta + events non chargés → skip sentinel (match reste candidat)",
				"match_id", matchID)
			return matchCitations{}
		}
	}
	return m
}

// ecrireMatch persiste les citations du match (base joueur, hors segment de lecture partagée) et
// avance cumulPre pour le match suivant du lot.
func (l lotCitations) ecrireMatch(ctx context.Context, matchID string, m matchCitations, cumulPre map[string]int) error {
	if err := writeCitations(ctx, l.playerDB, matchID, m.deltas); err != nil {
		return err
	}
	// Logué APRÈS l'écriture, jamais avant : un log posé dans la décision affirmerait un jeton
	// que writeCitations peut avoir échoué à poser (player DB read-only, disque plein).
	if m.terminalNoEvents {
		slog.InfoContext(ctx, "citations: état terminal — jeton _processed posé",
			"match_id", matchID,
			"raison", "events_empty — film récupéré, aucun event exploitable")
	}
	debrief.ReportCitations(ctxkeys.TitleSlug(ctx), l.xuid, matchID)
	for _, d := range m.deltas {
		cumulPre[d.NameNorm] += d.Value
	}
	return nil
}

// runPostSyncCitations branche les citations dans le pipeline post-sync : base joueur tenue par
// runPostSyncPipeline, base partagée lue par segments courts (sélection, tri, un par match).
// Best-effort : retourne (0, nil) si metadata.duckdb absent ou citation_mappings vide.
func (e *SyncEngine) runPostSyncCitations(ctx context.Context, playerDB *sql.DB, shared lireShared) (int, error) {
	// Emprunte la connexion metadata déjà ouverte dans le cache process (RW par
	// le pool serveur). OpenReadOnly échoue si une connexion RW existe déjà sur
	// le même fichier ("Can't open a connection with a different configuration").
	// LookupCachedDB retourne RW en priorité, RO sinon — sans incrémenter le
	// refCount, donc pas de Close() ici.
	var metaDB *sql.DB
	if e.metaDB != nil {
		metaDB = e.metaDB
	} else if cached, ok := duckdbpkg.LookupCachedDB(e.metadataDBPath); ok {
		metaDB = cached.SQLDb()
	} else {
		slog.WarnContext(ctx, "citations post-sync: metadata DB non disponible — skip", "player", e.gamertag)
		return 0, nil
	}

	matchIDs, err := selectMatchesForCitations(ctx, playerDB, false)
	if err != nil {
		return 0, fmt.Errorf("select: %w", err)
	}
	// Les matchs dont le film a été dérivé APRÈS leurs citations : recalcul complet.
	var filmPlusRecent []string
	var errFilm error
	if err := shared(func(sharedDB *sql.DB) error {
		filmPlusRecent, errFilm = citationsfilm.SelectMatchesFilmPlusRecent(ctx, playerDB, sharedDB)
		return nil
	}); err != nil {
		return 0, fmt.Errorf("lecture shared: %w", err)
	}
	if errFilm != nil {
		slog.WarnContext(ctx, "citations post-sync: dérivations du film illisibles, citations du film non recalculées",
			"player", e.gamertag, "err", errFilm)
	}
	if len(filmPlusRecent) > 0 {
		slog.InfoContext(ctx, "citations post-sync: film dérivé après les citations, recalcul",
			"player", e.gamertag, "count", len(filmPlusRecent))
	}
	matchIDs = citationsfilm.UnionMatchIDs(matchIDs, filmPlusRecent)
	if len(matchIDs) == 0 {
		slog.DebugContext(ctx, "citations post-sync: aucun nouveau match", "player", e.gamertag)
		return 0, nil
	}
	slog.InfoContext(ctx, "citations post-sync: nouveaux matchs détectés",
		"player", e.gamertag, "count", len(matchIDs))

	// shared_pve en lecture RO (BUG A / I7) : le reader réutilise le handle
	// process s'il est déjà tenu. Dégradation gracieuse si absent.
	pve := OpenPveReadForCitations(ctx, e.pveDBPath)
	defer pve.Close()

	if err := backfillCitationsLot(ctx, lotCitations{
		metadataDB: metaDB, playerDB: playerDB, shared: shared,
		pve: pve, xuid: e.xuid, matchIDs: matchIDs, killSource: e.citationWeaponSourceDuMoteur(ctx),
	}); err != nil {
		return 0, fmt.Errorf("backfill: %w", err)
	}
	return len(matchIDs), nil
}
