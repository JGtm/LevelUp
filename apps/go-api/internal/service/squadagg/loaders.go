package squadagg

import "levelup/go-api/internal/games/canonical"

// BuildSquadOrder construit l'ordre stable des gamertags du squad : main puis
// coequipiers dans l'ordre d'arrivee.
func BuildSquadOrder(mainGT string, teammates []string) []string {
	out := make([]string, 0, 1+len(teammates))
	out = append(out, mainGT)
	out = append(out, teammates...)
	return out
}

// ExtractSquadXUIDs derive le mapping gamertag -> xuid en regardant la
// premiere PlayerMatchRow disponible pour chaque joueur. Si un joueur n'a
// aucun match (capability absente), il est omis.
func ExtractSquadXUIDs(squadOrder []string, perPlayer map[string][]canonical.PlayerMatchRow) map[string]string {
	out := make(map[string]string, len(squadOrder))
	for _, gt := range squadOrder {
		rows := perPlayer[gt]
		if len(rows) == 0 {
			continue
		}
		if xuid := rows[0].Self.Identity.XUID; xuid != "" {
			out[gt] = xuid
		}
	}
	return out
}
