// Package persist — demo_seed_columns.go : contrat de colonnes des tables
// critiques, source UNIQUE partagée avec le seeder démo synthétique (internal/ops).
//
// Contexte (F7, revue 2026-07-17) : le seeder démo recopiait à la main les listes
// de colonnes de ces tables. La recette append-only ADR 0026 ne touche que persist
// → le corpus démo rotait silencieusement quand persist gagnait une colonne. Ces
// constantes exportées deviennent la référence : leur fidélité à l'INSERT persist
// est verrouillée par demo_seed_columns_test.go (auto-parité), et le seeder est
// épinglé dessus par ops/seed_demo_column_parity_test.go (test qui CASSE si les
// colonnes du seeder divergent de persist, hors allowlist documentée).
//
// MAJ OBLIGATOIRE avec la recette ADR 0026 : ajouter une colonne à l'INSERT persist
// d'une de ces tables impose de mettre à jour la constante correspondante ICI (sinon
// l'auto-parité échoue) PUIS de statuer côté seeder (sinon la parité seeder échoue).
//
// Note match_skill_rank : DEUX persisters écrivent cette table (persistSkillRank,
// chemin CSR, colonnes ci-dessous ; lusr_append_only_persister ajoute
// expected_win_prob + start_time). Cette constante reflète le chemin CSR primaire.
package persist

// MatchRegistryColumns — colonnes de l'INSERT persistMatchRegistry (shared_persister.go).
var MatchRegistryColumns = []string{
	//nolint:goconst // listes de colonnes SQL déclaratives — une const par nom de colonne nuirait à la lisibilité du contrat
	"match_id", "start_time", "end_time", "start_time_utc", "end_time_utc",
	"playlist_id", "playlist_name", "playlist_version_id",
	"map_id", "map_name", "map_version_id",
	"pair_id", "pair_name", "pair_version_id",
	"game_variant_id", "game_variant_name", "game_variant_version_id",
	"mode_category", "is_ranked", "is_firefight", "season_id",
	"duration_seconds", "playable_duration_seconds",
	"real_start_time", "team_0_score", "team_1_score",
	"team_0_ps_score", "team_1_ps_score",
	"team_0_rounds_won", "team_1_rounds_won", "rounds_total",
	"match_intensity", "backfill_completed", "events_loaded",
	"first_sync_by", "first_sync_at", "last_updated_at",
	"player_count",
	"created_at", "updated_at",
}

// MatchParticipantsColumns — colonnes de l'INSERT persistParticipants (shared_persister.go).
var MatchParticipantsColumns = []string{
	"match_id", "xuid", "gamertag",
	"team_id", "outcome", "rank", "score",
	"kills", "deaths", "assists",
	"shots_fired", "shots_hit",
	"damage_dealt", "damage_taken",
	"kda", "accuracy", "personal_score",
	"time_played_seconds", "avg_life_seconds",
	"kills_expected", "deaths_expected", "kills_stddev", "deaths_stddev",
	"team_mmr", "enemy_mmr",
	"headshot_kills",
	"max_killing_spree", "grenade_kills", "melee_kills", "power_weapon_kills",
	"assassination_kills", "ground_pound_kills", "shoulder_bash_kills",
	"present_at_beginning", "present_at_completion", "joined_in_progress", "left_in_progress",
	"first_joined_time", "last_leave_time",
	"backfill_bits",
	"created_at",
}

// MatchSkillRankColumns — colonnes de l'INSERT persistSkillRank (player_persister.go,
// chemin CSR). Append-only (ADR 0026) : written_at posé par DEFAULT côté DB.
var MatchSkillRankColumns = []string{
	"match_id", "rating_type", "rating_value", "rating_deviation",
	"tier", "tier_fr", "sub_tier", "tier_label", "rating_delta", "playlist_group",
}

// MatchCSRColumns — colonnes de l'INSERT persistMatchCSRs (shared_persister.go).
// Append-only (ADR 0026) : written_at posé par DEFAULT côté DB.
var MatchCSRColumns = []string{
	"match_id", "xuid", "rating_type",
	"rating_value", "tier", "sub_tier", "tier_label",
	"rating_delta", "measurement_matches_remaining", "season_id",
}
