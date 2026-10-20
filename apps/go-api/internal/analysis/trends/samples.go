package trends

import (
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain/equipmentusage"
)

// ObjectiveSample porte, pour un match à objectif, ce que le joueur et son camp
// ont fait par rôle. Les valeurs sont des actions (Prendre, Défendre) ou des
// secondes (Tenir). TeamSize est l'effectif du camp présent à la fin.
type ObjectiveSample struct {
	Take, Defend, HoldSeconds             float64
	TeamTake, TeamDefend, TeamHoldSeconds float64
	TeamSize                              int
}

// EquipmentSample porte l'usage de l'équipement d'un match mesuré : Used est le
// nombre d'objets utilisés, Total la somme utilisés + gardés + lâchés.
type EquipmentSample struct {
	Used, Total int
}

// ObjectiveInput regroupe les lectures dont ObjectiveSamples tire ses échantillons.
type ObjectiveInput struct {
	PlayerXUID string
	// Rows : lignes de rôle des deux camps.
	Rows []sessionusage.ObjectiveRow
	// FlagGrabs : prises nettes de drapeau, ajoutées à Prendre.
	FlagGrabs []sessionusage.FlagGrabsNetRow
	// Teams : camps et effectifs des matchs lus.
	Teams sessionusage.TeamContext
}

// ObjectiveSamples construit l'échantillon d'objectif de chaque match (clé =
// identifiant de match). Un match n'a d'échantillon que si le joueur y porte au
// moins une ligne de rôle et si son camp est connu. Les valeurs du joueur sont la
// somme de ses lignes ; celles de l'équipe, la somme des lignes des joueurs de
// son camp, lui compris. Les prises nettes de drapeau s'ajoutent à Prendre.
func ObjectiveSamples(in ObjectiveInput) map[string]ObjectiveSample {
	out := map[string]ObjectiveSample{}
	for i := range in.Rows {
		r := &in.Rows[i]
		if r.XUID != in.PlayerXUID {
			continue
		}
		if _, ok := in.Teams.PlayerTeam[r.MatchID]; !ok {
			continue
		}
		if _, seen := out[r.MatchID]; !seen {
			out[r.MatchID] = ObjectiveSample{TeamSize: in.Teams.TeamSize[r.MatchID]}
		}
	}
	for i := range in.Rows {
		r := &in.Rows[i]
		addRoles(out, in, r.MatchID, r.XUID, r.Take, r.Defend, r.HoldSeconds)
	}
	for i := range in.FlagGrabs {
		g := &in.FlagGrabs[i]
		addRoles(out, in, g.MatchID, g.XUID, float64(g.Net), 0, 0)
	}
	return out
}

// addRoles cumule les rôles d'un participant sur l'échantillon de son match : sur
// le joueur s'il est le joueur, sur l'équipe s'il est du camp du joueur.
func addRoles(out map[string]ObjectiveSample, in ObjectiveInput, matchID, xuid string, take, defend, hold float64) {
	s, ok := out[matchID]
	if !ok {
		return
	}
	if xuid == in.PlayerXUID {
		s.Take += take
		s.Defend += defend
		s.HoldSeconds += hold
	}
	if team, known := in.Teams.TeamOf[matchID][xuid]; known && team == in.Teams.PlayerTeam[matchID] {
		s.TeamTake += take
		s.TeamDefend += defend
		s.TeamHoldSeconds += hold
	}
	out[matchID] = s
}

// EquipmentSamples construit l'échantillon d'équipement de chaque match où le
// joueur porte une ligne d'usage ; un match sans ligne n'est pas mesuré.
func EquipmentSamples(playerXUID string, players []sessionusage.PlayerRow) map[string]EquipmentSample {
	out := map[string]EquipmentSample{}
	families := equipmentusage.EquipmentOutcomeFamilies()
	for i := range players {
		p := &players[i]
		if p.XUID != playerXUID {
			continue
		}
		c := sessionusage.PlayerOutcomeCounts(p, families)
		s := out[p.MatchID]
		s.Used += c.Used
		s.Total += c.Used + c.Kept + c.Dropped
		out[p.MatchID] = s
	}
	return out
}

// Attach pose les échantillons sur les matchs, par identifiant.
func Attach(matches []Match, objectives map[string]ObjectiveSample, equipment map[string]EquipmentSample) {
	for i := range matches {
		id := matches[i].ID
		if o, ok := objectives[id]; ok {
			o := o
			matches[i].Objective = &o
		}
		if e, ok := equipment[id]; ok {
			e := e
			matches[i].Equipment = &e
		}
	}
}

// MatchIDsSince retourne, dans l'ordre de ms, les identifiants des matchs qui
// ne commencent pas avant since. Les matchs sans identifiant sont ignorés.
func MatchIDsSince(matches []Match, since time.Time) []string {
	out := make([]string, 0, len(matches))
	for i := range matches {
		if matches[i].ID != "" && !matches[i].Start.Before(since) {
			out = append(out, matches[i].ID)
		}
	}
	return out
}
