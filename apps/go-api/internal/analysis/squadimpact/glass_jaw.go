package squadimpact

// glass_jaw.go — LE RÔLE « OS DE VERRE » : l'allié dont le bouclier a été brisé le plus souvent
// dans le match. Contrairement aux autres rôles, il ne se lit pas dans le fil des événements mais
// dans les mesures de bouclier tirées du film (bris et recharges complètes par joueur).

import "levelup/go-api/internal/analysis"

// ShieldCount — les bris de bouclier et les recharges complètes d'un joueur sur un match.
type ShieldCount struct {
	Breaks    int
	Recharges int
}

// GlassJaw rend le xuid de l'allié qui a le plus de bris de bouclier ; faux quand le match n'est
// pas mesuré (shields vide) ou qu'aucun allié n'a de bris. À égalité de bris, celui qui a le moins
// de recharges l'emporte, puis le xuid croissant. L'équipe alliée est celle de allies, COMPLÈTE :
// le rôle se décide à l'échelle de l'équipe, comme les autres, puis RolesOfMatch ne garde que les
// membres de l'escouade. Un allié sans xuid (bot) ou sans mesure ne concourt pas.
func GlassJaw(allies []analysis.ParticipantSnap, shields map[string]ShieldCount) (string, bool) {
	best, bestCount, found := "", ShieldCount{}, false
	for _, a := range allies {
		c, measured := shields[a.XUID]
		if a.XUID == "" || !measured || c.Breaks <= 0 {
			continue
		}
		if !found || moreBrittle(a.XUID, c, best, bestCount) {
			best, bestCount, found = a.XUID, c, true
		}
	}
	return best, found
}

// moreBrittle dit si (xuid, c) passe devant (best, bestCount) : plus de bris, puis moins de
// recharges, puis xuid croissant.
func moreBrittle(xuid string, c ShieldCount, best string, bestCount ShieldCount) bool {
	switch {
	case c.Breaks != bestCount.Breaks:
		return c.Breaks > bestCount.Breaks
	case c.Recharges != bestCount.Recharges:
		return c.Recharges < bestCount.Recharges
	default:
		return xuid < best
	}
}
