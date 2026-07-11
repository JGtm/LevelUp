// Package canonical — schéma canonique services consommé par les services produit.
//
// Ce package définit le contrat stable que les TitleDataAdapter remplissent
// après lecture du stockage propre à chaque titre. Aucun service produit ne
// connaît le SQL, les noms de colonnes ou les unités natives d'un titre :
// il consomme uniquement ce canonique.
//
// Frontière avec internal/domain/ : domain/ héberge les types non-titre-aware
// (sessions, performance score, citations) qui restent calculés au-dessus du
// canonique services. canonical/ ne dépend de rien hors stdlib.
package canonical

// FieldKey est un identifiant stable, machine-friendly, agnostique du titre,
// désignant un champ produit (ex: kills, accuracy, duration_seconds).
//
// Les valeurs sont en snake_case et figées : un test garde-fou
// (fields_test.go) refuse tout renommage sans bump de schema_version dans les
// TOML mappings correspondants.
type FieldKey string

// --- Combat (group = "combat") ---

const (
	FieldKills            FieldKey = "kills"
	FieldDeaths           FieldKey = "deaths"
	FieldAssists          FieldKey = "assists"
	FieldHeadshotKills    FieldKey = "headshot_kills"
	FieldMeleeKills       FieldKey = "melee_kills"
	FieldGrenadeKills     FieldKey = "grenade_kills"
	FieldPowerWeaponKills FieldKey = "power_weapon_kills"
	FieldDamageDealt      FieldKey = "damage_dealt"
	FieldDamageTaken      FieldKey = "damage_taken"
	FieldShotsFired       FieldKey = "shots_fired"
	FieldShotsHit         FieldKey = "shots_hit"
	FieldAccuracy         FieldKey = "accuracy"
	FieldKDR              FieldKey = "kdr"
	FieldKDA              FieldKey = "kda"
	FieldMaxKillingSpree  FieldKey = "max_killing_spree"
)

// --- Match (group = "match") ---

const (
	FieldMatchID          FieldKey = "match_id"
	FieldStartedAtUTC     FieldKey = "started_at_utc"
	FieldDurationSeconds  FieldKey = "duration_seconds"
	FieldIsRanked         FieldKey = "is_ranked"
	FieldIsPvE            FieldKey = "is_pve"
	FieldOutcome          FieldKey = "outcome"
	FieldPersonalScore    FieldKey = "personal_score"
	FieldTeamScore        FieldKey = "team_score"
	FieldRankInMatch      FieldKey = "rank_in_match"
	FieldRankedMatchCount FieldKey = "ranked_match_count"
)

// --- Career (group = "career") ---

const (
	FieldCurrentRankID      FieldKey = "current_rank_id"
	FieldCurrentXP          FieldKey = "current_xp"
	FieldXPForNextRank      FieldKey = "xp_for_next_rank"
	FieldTotalMatchesPlayed FieldKey = "total_matches_played"
	FieldTotalKillsCareer   FieldKey = "total_kills_career"
	FieldWinRate            FieldKey = "win_rate"
)

// --- Skill (group = "skill") ---

const (
	FieldTeamMMR        FieldKey = "team_mmr"
	FieldEnemyMMR       FieldKey = "enemy_mmr"
	FieldKillsExpected  FieldKey = "kills_expected"
	FieldDeathsExpected FieldKey = "deaths_expected"
	FieldCSRValue       FieldKey = "csr_value"
	FieldLUSRValue      FieldKey = "lusr_value"
)

// --- PvE Firefight (group = "pve") ---

const (
	FieldWavesCompleted FieldKey = "waves_completed"
	FieldBossesKilled   FieldKey = "bosses_killed"
	FieldGruntKills     FieldKey = "grunt_kills"
	FieldEliteKills     FieldKey = "elite_kills"
	FieldJackalKills    FieldKey = "jackal_kills"
	FieldBruteKills     FieldKey = "brute_kills"
	FieldHunterKills    FieldKey = "hunter_kills"
)

// --- Derived KPIs (group = "derived") — métriques synthétisées sur une période ---

const (
	FieldMatchCount           FieldKey = "match_count"
	FieldHeadshotsPerMatch    FieldKey = "headshots_per_match"
	FieldDeathsPerMin         FieldKey = "deaths_per_min"
	FieldAssistsPerMin        FieldKey = "assists_per_min"
	FieldAvgMaxKillingSpree   FieldKey = "avg_max_killing_spree"
	FieldAvgDamageDealt       FieldKey = "avg_damage_dealt"
	FieldAvgDamageTaken       FieldKey = "avg_damage_taken"
	FieldAvgLifeSeconds       FieldKey = "avg_life_seconds"
	FieldKDRatio              FieldKey = "kd_ratio"
	FieldKillsPerMin          FieldKey = "kills_per_min"
	FieldPerformanceScore     FieldKey = "performance_score"
	FieldPerfectKillsPerMatch FieldKey = "perfect_kills_per_match"
	FieldTimePlayedSeconds    FieldKey = "time_played_seconds"
	FieldOffensiveConversion  FieldKey = "offensive_conversion"
	FieldDefensiveResistance  FieldKey = "defensive_resistance"

	// FieldEngagementScore est le score d'engagement intra-match (percentile
	// 0..100 du residu joueur-attendu sur l'historique du joueur, modele
	// lobby-anchored v2). Metrique de 1er ordre title-agnostic : chaque titre
	// qui expose l'engagement (capability engagement.score) declare ce FieldKey
	// dans son fields.toml. Unite = score sans dimension, bornes [0, 100].
	// cf. internal/analysis/temporal/engagement_score.go + ADR chantier F7.
	FieldEngagementScore FieldKey = "engagement_score"
)

// AllFieldKeys retourne la liste exhaustive des FieldKey supportés par le
// canonique. Utilisé par les tests garde-fous et par le loader TOML pour
// valider que les sections [fields.X] référencent un FieldKey connu.
//
// Toute modification de cette liste implique :
//  1. PR sur PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md (annexe §17)
//  2. bump de schema_version dans les TOML mappings concernés
//  3. mise à jour du fichier golden testdata/fields.golden.txt
func AllFieldKeys() []FieldKey {
	return []FieldKey{
		// Combat
		FieldKills, FieldDeaths, FieldAssists, FieldHeadshotKills,
		FieldMeleeKills, FieldGrenadeKills, FieldPowerWeaponKills,
		FieldDamageDealt, FieldDamageTaken,
		FieldShotsFired, FieldShotsHit, FieldAccuracy,
		FieldKDR, FieldKDA, FieldMaxKillingSpree,
		// Match
		FieldMatchID, FieldStartedAtUTC, FieldDurationSeconds,
		FieldIsRanked, FieldIsPvE, FieldOutcome,
		FieldPersonalScore, FieldTeamScore, FieldRankInMatch,
		FieldRankedMatchCount,
		// Career
		FieldCurrentRankID, FieldCurrentXP, FieldXPForNextRank,
		FieldTotalMatchesPlayed, FieldTotalKillsCareer, FieldWinRate,
		// Skill
		FieldTeamMMR, FieldEnemyMMR,
		FieldKillsExpected, FieldDeathsExpected,
		FieldCSRValue, FieldLUSRValue,
		// PvE
		FieldWavesCompleted, FieldBossesKilled,
		FieldGruntKills, FieldEliteKills, FieldJackalKills,
		FieldBruteKills, FieldHunterKills,
		// Derived KPIs
		FieldMatchCount, FieldHeadshotsPerMatch, FieldDeathsPerMin,
		FieldAssistsPerMin, FieldAvgMaxKillingSpree,
		FieldAvgDamageDealt, FieldAvgDamageTaken, FieldAvgLifeSeconds,
		FieldKDRatio, FieldKillsPerMin, FieldPerformanceScore,
		FieldPerfectKillsPerMatch, FieldTimePlayedSeconds,
		FieldOffensiveConversion, FieldDefensiveResistance,
		FieldEngagementScore,
	}
}

// IsKnownFieldKey vérifie qu'un FieldKey appartient au canonique central.
func IsKnownFieldKey(k FieldKey) bool {
	for _, known := range AllFieldKeys() {
		if known == k {
			return true
		}
	}
	return false
}

// String permet d'utiliser FieldKey comme paramètre string-like.
func (f FieldKey) String() string { return string(f) }
