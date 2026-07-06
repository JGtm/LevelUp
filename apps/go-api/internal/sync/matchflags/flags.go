// Package matchflags — flags.go : bitmasks MatchBits (match_registry.backfill_completed,
// bits ≥ 16), package FEUILLE sans dépendance (K3c).
//
// Extrait de sync/backfill_flags.go (2026-07-06) pour rompre le cycle sync↔snapshot :
// le cluster snapshot (readiness) lit ces MBit* pour décider si un match est prêt, sans
// devoir importer tout le package sync. sync/backfill_flags.go les RÉ-EXPORTE (alias) pour
// que ses ~centaines d'usages existants restent inchangés — seuls le package sync et
// sync/snapshot dépendent d'ici, jamais l'inverse (feuille pure : zéro import).
//
// Valeurs numériquement identiques à l'origine (portage src/data/sync/constants.py MatchBits).
package matchflags

const (
	MBitEvents = 1 << 16 // 65536   — highlight_events chargés
	// bits 17 (assets) et 18 (aliases) RETIRÉS le 2026-05-08 (PLAN_BITMASKS_AUDIT_FIX).
	MBitKillerVictim      = 1 << 19 // 524288  — killer_victim_pairs chargés (global)
	MBitPVEStats          = 1 << 20 // 1048576 — stats PvE tentées pour ce match
	MBitWeaponKills       = 1 << 21 // 2097152 — weapon_kills chargés
	MBitWeaponKillsNoFilm = 1 << 22 // 4194304 — film 404/expiré, 0 chunk dispo
)
