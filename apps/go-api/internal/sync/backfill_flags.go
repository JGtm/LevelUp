// Package sync — backfill_flags.go : bitmasks pour le tracking granulaire des données backfillées.
//
// Portage numériquement identique de src/data/sync/constants.py (ParticipantBits,
// MatchBits, PveBits) et src/data/sync/migrations.py (BACKFILL_FLAGS).
//
// Trois niveaux de bitmask :
//   - ParticipantBits : stocké dans match_participants.backfill_bits
//     Granularité par joueur × match. Indique quelles colonnes sont remplies.
//   - MatchBits : stocké dans match_registry.backfill_completed (bits ≥ 16)
//     Granularité par match uniquement. Données globales au match.
//   - PveBits : stocké dans pve_match_stats.pve_bits (shared_pve.duckdb)
//     Granularité par joueur × match Firefight.
//
// Les anciens BACKFILL_FLAGS (bits 0-15 de match_registry.backfill_completed)
// sont conservés dans BackfillFlags pour rétrocompatibilité.
// MatchBits utilise les bits ≥ 16 pour éviter toute collision.
package sync

import "levelup/go-api/internal/sync/matchflags"

// ─────────────────────────────────────────────────────────────────────────────
// ParticipantBits — match_participants.backfill_bits (données par joueur × match)
// ─────────────────────────────────────────────────────────────────────────────

const (
	// Stats skill / MMR (Halo Infinite : pas d'assists, l'API ne les fournit pas).
	PBitTeamMMR   = 1 << 0 // 1       — team_mmr
	PBitEnemyMMR  = 1 << 1 // 2       — enemy_mmr
	PBitKillsExp  = 1 << 2 // 4       — kills_expected, kills_stddev
	PBitDeathsExp = 1 << 3 // 8       — deaths_expected, deaths_stddev
	// bit 4 (PBitAssistsExp) retiré — l'API Halo Infinite ne renvoie pas d'Assists
	// dans StatPerformances. Si un futur titre les expose, reservé pour ré-utilisation.

	// Stats de combat (get_match_stats)
	PBitAccuracy = 1 << 5 // 32      — accuracy
	PBitShots    = 1 << 6 // 64      — shots_fired, shots_hit
	PBitDamage   = 1 << 7 // 128     — damage_dealt, damage_taken
	PBitAvgLife  = 1 << 8 // 256     — avg_life_seconds
	PBitMedals   = 1 << 9 // 512     — médailles dans medals_earned pour ce xuid

	// Kills détaillés
	PBitGrenadeKills  = 1 << 10 // 1024  — grenade_kills
	PBitMeleeKills    = 1 << 11 // 2048  — melee_kills
	PBitPowerWeapon   = 1 << 12 // 4096  — power_weapon_kills
	PBitPersonalScore = 1 << 13 // 8192  — personal_score
	PBitHeadshotKills = 1 << 14 // 16384 — headshot_kills
	PBitMaxSpree      = 1 << 15 // 32768 — max_killing_spree

	// Stats calculées
	PBitKDA        = 1 << 16 // 65536  — kda (calculé)
	PBitTimePlayed = 1 << 17 // 131072 — time_played_seconds

	// Killer/victim
	PBitKillerVictim = 1 << 18 // 262144 — ce joueur est présent dans killer_victim_pairs

	// ── Groupes (combinaisons logiques) ──
	PBitMMR         = PBitTeamMMR | PBitEnemyMMR
	PBitExpected    = PBitKillsExp | PBitDeathsExp
	PBitSkill       = PBitMMR | PBitExpected
	PBitCombat      = PBitAccuracy | PBitShots | PBitDamage
	PBitKillsDetail = PBitGrenadeKills | PBitMeleeKills | PBitPowerWeapon | PBitHeadshotKills
	PBitCoreStats   = PBitAccuracy | PBitShots | PBitDamage | PBitAvgLife |
		PBitGrenadeKills | PBitMeleeKills | PBitPowerWeapon | PBitPersonalScore |
		PBitHeadshotKills | PBitMaxSpree | PBitKDA | PBitTimePlayed
	PBitAllStats = PBitSkill | PBitCoreStats | PBitMedals | PBitKillerVictim
)

// ─────────────────────────────────────────────────────────────────────────────
// MatchBits — match_registry.backfill_completed (bits ≥ 16)
// ─────────────────────────────────────────────────────────────────────────────

// MBit* sont désormais DÉFINIS dans le package feuille matchflags (K3c — rupture du cycle
// sync↔snapshot) et RÉ-EXPORTÉS ici : tous les usages existants (sync + domain/ops/scheduler/
// cmd) restent inchangés ; sync/snapshot importe matchflags directement (feuille pure).
const (
	MBitEvents            = matchflags.MBitEvents
	MBitKillerVictim      = matchflags.MBitKillerVictim
	MBitPVEStats          = matchflags.MBitPVEStats
	MBitWeaponKills       = matchflags.MBitWeaponKills
	MBitWeaponKillsNoFilm = matchflags.MBitWeaponKillsNoFilm
)

// ─────────────────────────────────────────────────────────────────────────────
// PveBits — pve_match_stats.pve_bits (shared_pve.duckdb)
// ─────────────────────────────────────────────────────────────────────────────

const (
	PveBitTotalKills = 1 << 0  // 1    — total_enemy_kills
	PveBitBossKills  = 1 << 1  // 2    — boss_kills
	PveBitGrunt      = 1 << 2  // 4    — grunt_kills
	PveBitElite      = 1 << 3  // 8    — elite_kills
	PveBitJackal     = 1 << 4  // 16   — jackal_kills
	PveBitBrute      = 1 << 5  // 32   — brute_kills
	PveBitHunter     = 1 << 6  // 64   — hunter_kills
	PveBitSkimmer    = 1 << 7  // 128  — skimmer_kills
	PveBitCrawler    = 1 << 8  // 256  — crawler_kills  (Forerunner)
	PveBitSoldier    = 1 << 9  // 512  — soldier_kills  (Forerunner)
	PveBitKnight     = 1 << 10 // 1024 — knight_kills   (Forerunner)
	PveBitWarden     = 1 << 11 // 2048 — warden_kills   (Forerunner)
	PveBitSentinel   = 1 << 12 // 4096 — sentinel_kills (Forerunner, rare)
	PveBitMarine     = 1 << 13 // 8192 — marine_kills   (alliés, rare)

	PveBitAllEnemies = PveBitGrunt | PveBitElite | PveBitJackal | PveBitBrute |
		PveBitHunter | PveBitSkimmer | PveBitCrawler | PveBitSoldier |
		PveBitKnight | PveBitWarden | PveBitSentinel | PveBitMarine
	PveBitFullPVE = PveBitTotalKills | PveBitBossKills | PveBitAllEnemies
)

// ─────────────────────────────────────────────────────────────────────────────
// BackfillFlags — legacy bitmask (bits 0-15 de match_registry.backfill_completed)
//
// Portage exact de BACKFILL_FLAGS (src/data/sync/migrations.py).
// Conservé pour rétrocompatibilité. Les bits ≥ 16 utilisent MatchBits.
// ─────────────────────────────────────────────────────────────────────────────

// Noms canoniques des types de backfill, partagés entre BackfillFlags
// (this file) et requestedTypeMap (scope.go). Centralisés pour éviter
// les duplications littérales (lint goconst).
const (
	BackfillTypeMedals         = "medals"
	BackfillTypeEvents         = "events"
	BackfillTypeSkill          = "skill"
	BackfillTypePersonalScores = "personal_scores"
	BackfillTypeShots          = "shots"
	BackfillTypeEnemyMMR       = "enemy_mmr"
	BackfillTypeAliases        = "aliases"
)

// BackfillFlags mappe les noms de flags legacy vers leur valeur de bit.
var BackfillFlags = map[string]int{
	BackfillTypeMedals:         1 << 0, // 1
	BackfillTypeEvents:         1 << 1, // 2
	BackfillTypeSkill:          1 << 2, // 4
	BackfillTypePersonalScores: 1 << 3, // 8
	// performance_scores supprimé : granularité joueur×match (bit 4 non utilisé)
	MetricKeyAccuracy:       1 << 5,  // 32
	BackfillTypeShots:       1 << 6,  // 64
	BackfillTypeEnemyMMR:    1 << 7,  // 128
	"assets":                1 << 8,  // 256
	"participants":          1 << 9,  // 512
	"participants_scores":   1 << 10, // 1024
	"participants_kda":      1 << 11, // 2048
	"participants_shots":    1 << 12, // 4096
	"participants_damage":   1 << 13, // 8192
	BackfillTypeAliases:     1 << 14, // 16384
	"participants_avg_life": 1 << 15, // 32768
	// weapon_kills (bit 18 = 262144) — OBSOLÈTE, voir MBitWeaponKills (1<<21)
	"weapon_kills": 1 << 18, // 262144
}

// ComputeBackfillMask calcule le masque de bits pour les types demandés.
//
//	ComputeBackfillMask("medals", "events") → 3
func ComputeBackfillMask(types ...string) int {
	mask := 0
	for _, t := range types {
		if bit, ok := BackfillFlags[t]; ok {
			mask |= bit
		}
	}
	return mask
}

// ComputeParticipantBitsFromData calcule le bitmask ParticipantBits depuis un
// map de données participant. Chaque colonne clé non-nil active le bit correspondant.
//
// Portage exact de compute_participant_bits_from_data() (constants.py).
func ComputeParticipantBitsFromData(data map[string]interface{}) int {
	bits := 0
	check := func(col string, bit int) {
		if v, ok := data[col]; ok && v != nil {
			bits |= bit
		}
	}
	check("team_mmr", PBitTeamMMR)
	check("enemy_mmr", PBitEnemyMMR)
	check("kills_expected", PBitKillsExp)
	check("deaths_expected", PBitDeathsExp)
	check("accuracy", PBitAccuracy)
	check("shots_fired", PBitShots)
	check("damage_dealt", PBitDamage)
	check("avg_life_seconds", PBitAvgLife)
	check("grenade_kills", PBitGrenadeKills)
	check("melee_kills", PBitMeleeKills)
	check("power_weapon_kills", PBitPowerWeapon)
	check("personal_score", PBitPersonalScore)
	check("headshot_kills", PBitHeadshotKills)
	check("max_killing_spree", PBitMaxSpree)
	check("kda", PBitKDA)
	check("time_played_seconds", PBitTimePlayed)
	return bits
}
