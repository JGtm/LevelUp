// Package halo_5 réserve l'espace de nommage de l'adapter Halo 5: Guardians,
// le 2e titre EXPERIMENTAL (cf. .ai/HANDOFF_HALO5_EXPERIMENTAL.md).
//
// Phase 0 (état actuel) : Halo 5 n'existe que comme config découverte au boot
// (config/titles/halo_5/, status coming_soon). Aucun adapter n'est encore câblé —
// le titre n'est pas servi. skeleton_test.go vérifie l'intégrité de la config
// (manifest coming_soon, matrice de capabilities, mappings chargeables).
//
// Phase 1 : implémentera ici games.TitleDataAdapter (client interne cryptum +
// mapping JSON Halo 5 -> canonical) + TitleSemanticAdapter, une fois la sonde
// live confirmée (343 sert encore Halo 5 en 2026).
package halo_5
