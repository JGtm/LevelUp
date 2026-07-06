// Package service - teammates_squad_charts.go : index des builders pour les
// charts teammates. Le code est decoupe en fichiers thematiques pour respecter
// la limite des 500 lignes par fichier (CLAUDE.md). Les builders vivent dans :
//
//   - teammates_squad_charts_sessions_maps.go         : buildSquadSessionTimeline (.04) +
//     buildSquadMapHeatmap (.03)
//   - teammates_squad_charts_impact_events.go         : buildSquadImpactMatrix (.07) +
//     buildSquadFirstEvents + helpers
//   - teammates_squad_charts_weapons_perf.go          : buildSquadWeaponKills +
//     buildSquadPerformanceSeries
//   - teammates_squad_charts_synergy.go               : buildSquadSynergyRadar (6 axes) +
//     SynergyOffensiveConversion +
//     SynergyDefensiveResistance +
//     synergyMainFallbackAxes +
//     loadSynergyMateAxes
//   - teammates_squad_charts_intensity_perminute.go   : buildSquadIntensityProfile +
//     buildSquadPerMinuteStats
//   - teammates_squad_charts_medal_digest.go          : buildMedalDigest +
//     loadImpactEventsByMatch + helpers
//
// Ce fichier ne contient plus de code - il sert de point d'entree documentaire
// pour reperer rapidement quel builder vit ou.
package teammates
