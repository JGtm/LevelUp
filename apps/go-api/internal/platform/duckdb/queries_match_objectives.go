// Package duckdb — queries_match_objectives.go : requête des stats d'objectifs de
// la vue Match, ISOLÉE du scoreboard (G1, 2026-07-26).
//
// Pourquoi un fichier séparé : cette requête est la SEULE lecture de la vue
// match_objective_stats_latest côté Match View, et c'est une lecture DÉGRADABLE.
// La sortir de queries_match.go (déjà au-delà du seuil de 500 lignes) rend la
// frontière explicite — le scoreboard (Q12) ne dépend plus d'elle.
package duckdb

// Q12bObjectiveStats : stats objectifs par joueur d'un match, lues SÉPARÉMENT de
// Q12. Paramètre : ?1 = match_id. Exécutée sur SharedReader (ADR 0016) — pas de
// préfixe `shared.`.
//
// Requête DÉGRADABLE : la vue match_objective_stats_latest peut manquer sur une DB
// non (encore) migrée, ou sur un schéma shared reconstruit (lecture snapshot). Le
// caller (loadMatchObjectiveStats, match_view_repo_scoreboard.go) logge un WARN
// explicite et sert un scoreboard SANS objectifs plutôt que de faire échouer toute
// la vue Match — c'est la régression prod du 25/07 (LEFT JOIN inconditionnel dans
// Q12 → Catalog Error → « le match n'a pas pu être chargé »).
//
// Colonnes dans l'ordre des champs de domain.ObjectiveRaw : CTF / Zones
// (Strongholds+KOTH) / Oddball / Stockpile / Extraction / VIP. NULL hors mode à
// objectif ou titre non supporté — le service ne construit
// MatchScoreboardRow.Objective que si un bloc est non-NULL (data-driven par mode).
const Q12bObjectiveStats = `
SELECT
    o.xuid,
    o.flag_captures, o.flag_capture_assists, o.flag_grabs, o.flag_secures,
    o.flag_steals, o.flag_returns, o.flag_carriers_killed, o.flag_returners_killed,
    o.kills_as_flag_carrier, o.kills_as_flag_returner, o.time_as_flag_carrier_seconds,
    o.zone_captures, o.zone_secures, o.zone_offensive_kills, o.zone_defensive_kills,
    o.zone_scoring_ticks, o.time_in_zones_seconds,
    o.kills_as_skull_carrier, o.skull_carriers_killed, o.skull_grabs,
    o.skull_scoring_ticks, o.time_as_skull_carrier_seconds,
    o.longest_time_as_skull_carrier_seconds,
    o.kills_as_power_seed_carrier, o.power_seed_carriers_killed, o.power_seeds_deposited,
    o.power_seeds_stolen, o.time_as_power_seed_carrier_seconds,
    o.time_as_power_seed_driver_seconds,
    o.extraction_conversions_completed, o.extraction_conversions_denied,
    o.extraction_initiations_completed, o.extraction_initiations_denied,
    o.successful_extractions,
    o.kills_as_vip, o.vip_kills, o.vip_assists, o.times_selected_as_vip,
    o.max_killing_spree_as_vip, o.time_as_vip_seconds, o.longest_time_as_vip_seconds
FROM match_objective_stats_latest o
WHERE o.match_id = ?`
