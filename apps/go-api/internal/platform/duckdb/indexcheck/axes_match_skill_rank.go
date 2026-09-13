package indexcheck

// axes_match_skill_rank.go — LA CARTE DES AXES INDEXÉS de `match_skill_rank`.
//
// SOURCE UNIQUE, et c'est le point. Cette carte a deux consommateurs : la sonde
// data-health périodique (`internal/scheduler/data_health_msr_index.go`) et
// l'outil de réparation (`cmd/repair_msr_index`). Recopiée, elle diverge en
// silence dès qu'une migration ajoute ou retire un index : la sonde continuerait
// de dire « sain » sur un axe qu'elle ne regarde plus. Garde-rail :
// `internal/archlint/no_local_msr_axes_test.go` interdit tout autre fichier de
// nommer un `"idx_msr_*"` en littéral Go.
//
// AUTORITÉ DES INDEX : `internal/games/halo_infinite/migrations/steps_player_match_skill_rank.go`
// (et son miroir de création `internal/sync/schema.go`). Ajouter un index là-bas
// sans ajouter son axe ici laisse le nouvel index NON SURVEILLÉ.

// MatchSkillRankTable — la table couverte par la carte ci-dessous.
const MatchSkillRankTable = "match_skill_rank"

// MatchSkillRankAxes rend les trois axes indexés de `match_skill_rank`, un par
// index posé par la migration. La copie est défensive : un appelant ne doit pas
// pouvoir modifier la carte partagée.
func MatchSkillRankAxes() []Axis {
	axes := make([]Axis, len(matchSkillRankAxes))
	copy(axes, matchSkillRankAxes)
	return axes
}

var matchSkillRankAxes = []Axis{
	{
		Name:        "playlist_group (idx_msr_playlist)",
		KeyExprs:    []string{"playlist_group || ''"},
		LookupWhere: "playlist_group = ?",
		Indexes:     []string{"idx_msr_playlist"},
	},
	{
		Name:        "rating_type (idx_msr_rating_type)",
		KeyExprs:    []string{"rating_type || ''"},
		LookupWhere: "rating_type = ?",
		Indexes:     []string{"idx_msr_rating_type"},
	},
	{
		// L'index porte (match_id, rating_type, written_at). Le lookup sonde le
		// triplet entier : c'est le seul prédicat que cet index sert pleinement.
		Name:        "match_id+rating_type+written_at (idx_msr_match_lookup)",
		KeyExprs:    []string{"match_id || ''", "rating_type || ''", "CAST(written_at AS VARCHAR)"},
		LookupWhere: "match_id = ? AND rating_type = ? AND written_at = CAST(? AS TIMESTAMP)",
		Indexes:     []string{"idx_msr_match_lookup"},
	},
}
