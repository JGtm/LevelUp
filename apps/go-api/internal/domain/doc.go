// Package domain — types métier canoniques et DTO partagés (matchs, sessions,
// career, citations, skill_v2, engagement, admin, ...).
//
// Structs purs sans I/O ni logique métier lourde, consommés par les couches
// service / handlers / analysis. C'est le vocabulaire partagé de l'API Go ; les
// types inter-titres canoniques vivent sous internal/games/canonical/.
package domain
