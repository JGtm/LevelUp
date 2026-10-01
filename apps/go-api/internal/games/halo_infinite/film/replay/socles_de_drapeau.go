package replay

// socles_de_drapeau.go — LES SOCLES DE DRAPEAU D'UNE CARTE, projetes pour le calque du drapeau.
//
// DEPLACE depuis `replaybuild/flagspawns.go` le 2026-09-29 (lot V2 du plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`), sans changement de regle : le collecteur de kills lit
// desormais les porteurs du drapeau au sync (`PortagesAuSync`) et a besoin des MEMES socles que
// la cuisson. Une projection recopiee au sync aurait diverge au premier ajustement de la regle
// du socle neutre ; elle est ecrite ICI, une fois, et les deux appelants la lisent. Le
// CHARGEMENT du catalogue, lui, reste chez chaque appelant (il sait ou est le depot).

import "levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"

// SoclesDeDrapeau rend TOUS les socles de drapeau de la carte, en coordonnees monde — les deux
// socles d'equipe ET le socle neutre du centre. Le calque retient ceux qui correspondent a la
// variante qu'il reconnait (`flag_neutral.go`) : cette projection ne decide pas du mode.
func (e MapObjectivesEntry) SoclesDeDrapeau() []FlagSpawn {
	points := e.PointsOfRole(mapvar.RoleFlagSpawn)
	out := make([]FlagSpawn, 0, len(points))
	for _, p := range points {
		out = append(out, FlagSpawn{
			Team: equipeDuSocle(p), Neutral: p.Neutral,
			X: float32(p.Center.X), Y: float32(p.Center.Y),
		})
	}
	return out
}

// equipeDuSocle rend l'equipe proprietaire d'un socle de drapeau : celle du fichier de carte,
// SAUF si le socle porte le label de la variante « drapeau neutre » — auquel cas il est neutre,
// quoi que dise son `team_index`.
//
// LE LABEL PRIME SUR LE `team_index`, ET CE N'EST PAS UNE PRECAUTION THEORIQUE. Le socle
// central d'Illusion (`9e821f5e`, object_index 201, au point (0, 0)) porte
// `ctf_neutral_include` ET `team_index = 0` : lu par son team_index, il devenait un TROISIEME
// drapeau d'equipe 0 fige au milieu de la carte, et il creait la plus grande zone aveugle du
// parc. Corrige le 2026-09-13 (rapport 6.11, decouverte D1). Le recensement du catalogue est
// dans le godoc de [mapvar.Objective.IsCTFNeutral] : sur 63 socles neutres, le label est juste
// 63 fois, le team_index 62.
//
// CE QUE CETTE FONCTION NE DIT PAS (2026-09-13, decouverte D-B2). Elle rend une EQUIPE, pas une
// variante : `TeamNeutral` y signifie tantot « socle neutre », tantot « equipe inconnue » — huit
// socles du catalogue portent `team_index = -1` sans etre neutres. La neutralite voyage donc
// dans son propre champ, [FlagSpawn.Neutral], pose depuis le meme label, et c'est LUI que le tri
// du calque lit.
func equipeDuSocle(p PointObjective) int {
	if p.Neutral {
		return TeamNeutral
	}
	return p.TeamIndex
}
