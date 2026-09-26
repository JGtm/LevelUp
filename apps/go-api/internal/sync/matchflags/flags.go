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
	MBitKillerVictim = 1 << 19 // 524288  — killer_victim_pairs chargés (global)
	MBitPVEStats     = 1 << 20 // 1048576 — stats PvE tentées pour ce match
	MBitWeaponKills  = 1 << 21 // 2097152 — détail par arme chargé pour ce match
	// MBitFilmAbsent : le film Theater de ce match est DÉFINITIVEMENT indisponible
	// (404/410 côté 343, ou manifeste à 0 chunk). MARQUEUR TERMINAL : il dit qu'aucune
	// passe de film ne réussira jamais sur ce match, et c'est ce qui empêche les
	// rattrapages (étapes 1.57 et 1.58) de le redemander à chaque cycle — 581 des 999
	// candidats de 1.57 sont dans ce cas (mesure du 2026-08-29).
	//
	// RENOMMÉ le 2026-09-01 (ancien nom : MBitWeaponKillsNoFilm). La VALEUR est
	// inchangée — elle est persistée dans match_registry.backfill_completed, on ne
	// renumérote jamais un bit posé en base. Seul le nom bougeait : il parlait de
	// `weapon_kills`, table supprimée côté Halo Infinite le même jour, alors que le bit
	// n'a jamais rien dit de cette table. Son poseur a déménagé de l'étape 1.55
	// (supprimée) vers l'étape 1.57 (internal/sync/killcollector/registry_flags.go).
	MBitFilmAbsent     = 1 << 22 // 4194304 — film 404/expiré, 0 chunk dispo
	MBitObjectiveStats = 1 << 23 // 8388608 — stats objectifs TENTÉES pour ce match
	// (posé même sans ligne produite : un match sans mode à objectif — Slayer — est
	// marqué "traité" pour ne pas être re-fetché par le backfill objective_stats).
)
