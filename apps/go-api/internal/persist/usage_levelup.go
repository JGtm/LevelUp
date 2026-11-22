package persist

// usage_levelup.go — les trois mesures que LevelUp calcule sur le film et range à côté du résumé
// d'usage du décodeur : bris de bouclier, recharges complètes, vengeances. Elles voyagent à côté de
// `replay.UsagePlayerSummary` et non dedans : le type du décodeur ne change pas, et la révision de
// leur calcul est celle de LevelUp (colonne `match_usage_films.levelup_rev`).

import "database/sql"

// UsageLevelUpPlayer : les mesures d'UN joueur sur UN match. Un pointeur nil se range NULL
// (« non mesuré »), distinct d'un pointeur sur zéro.
type UsageLevelUpPlayer struct {
	ShieldBreaks    *int
	ShieldRecharges *int
	RevengeKills    *int
}

// UsageLevelUp : les mesures d'un match, par xuid, et la révision du calcul qui les a produites.
// La valeur zéro (Rev vide, aucun joueur) range une révision NULL et des colonnes NULL : une passe
// que le rattrapage re-dérive toujours.
type UsageLevelUp struct {
	Rev     string
	Players map[string]UsageLevelUpPlayer
}

// revValue range une révision vide en NULL.
func (u UsageLevelUp) revValue() sql.NullString {
	return sql.NullString{String: u.Rev, Valid: u.Rev != ""}
}
