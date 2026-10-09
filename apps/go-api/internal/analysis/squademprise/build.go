package squademprise

// build.go — L'ASSEMBLAGE : le périmètre match par match, puis la soirée (bilan, objets,
// production) comme somme des matchs, puis l'habitude.

import (
	"sort"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// soiree — les sommes d'un ensemble de matchs.
type soiree struct {
	obj *objets
	// bonusMatches / tiersMatches : les matchs où bonus / niveaux de socle se lisent.
	bonusMatches, tiersMatches int
	outcomes                   [2]sessionusage.OutcomeCounts
	effectMS                   [2]int64
	effectKills                [2]int
	// pwk : frags aux armes spéciales, tous les matchs qui les portent ; pwkOnTiers : ceux
	// dont les niveaux de socle sont mesurés (le périmètre de l'exposition).
	pwk, pwkOnTiers domain.SquadEmpriseCount
	pwkMatches      int
	// prisesRendement : les prises du rendement des armes spéciales, sur les mêmes matchs et les
	// mêmes familles que pwkOnTiers (matchTally.pwkPrises).
	prisesRendement domain.SquadEmpriseCount
	// veh : les véhicules, indépendants du film (vehicles.go).
	veh vehicleSum
}

func newSoiree() *soiree { return &soiree{obj: newObjets()} }

// add verse un match dans la soirée. Les objets d'un match ne sont versés que là où ils se
// lisent : tallyMatch n'en compte pas ailleurs.
func (s *soiree) add(t matchTally) {
	s.veh.add(t.veh)
	s.obj.merge(t.obj, estVehicule)
	if t.pwk != nil {
		s.pwkMatches++
		s.pwk.Us += t.pwk.Us
		s.pwk.Them += t.pwk.Them
		if t.tiersMeasured() {
			s.pwkOnTiers.Us += t.pwk.Us
			s.pwkOnTiers.Them += t.pwk.Them
		}
	}
	if !t.bonusMeasured() {
		return
	}
	s.bonusMatches++
	if t.tiersMeasured() {
		s.tiersMatches++
		p := t.obj.total(domain.EmpriseResourcePowerWeapon)
		if t.pwkPrises != nil {
			p = *t.pwkPrises
		}
		s.prisesRendement.Us += p.Us
		s.prisesRendement.Them += p.Them
	}
	s.obj.merge(t.obj, func(res string) bool { return !estVehicule(res) })
	for i := 0; i < 2; i++ {
		s.outcomes[i].Add(t.outcomes[i])
		s.effectMS[i] += t.effectMS[i]
		s.effectKills[i] += t.effectKills[i]
	}
}

// Build rend le bloc publié.
func Build(in Input) domain.SquadEmpriseBlock {
	ix := newIndex(&in)
	players := in.Players
	if players == nil {
		players = []domain.SessionUsageSquadPlayer{}
	}
	in.Players = players
	current := chronologique(in.Current)
	block := domain.SquadEmpriseBlock{
		MatchesTotal: len(current), FilmUnavailable: in.FilmUnavailable,
		SheetUnavailable: in.SheetUnavailable, Players: players,
		Matches: make([]domain.SquadEmpriseMatch, 0, len(current)),
	}
	s := newSoiree()
	for _, m := range current {
		t := tallyMatch(m.MatchID, ix)
		// Mesuré = filmé ET camp connu : un match filmé au camp inconnu n'apporte aucun compte.
		if t.bonusMeasured() {
			block.MatchesMeasured++
		}
		block.Matches = append(block.Matches, publierMatch(m, t, &in))
		s.add(t)
	}
	block.Resources = bilan(s)
	block.Objects = s.obj.publier("", players, &in)
	block.Production = production(s)
	if in.Film != nil && len(in.Timeline) > 0 {
		block.Habit = buildHabit(&in, ix, current)
	}
	return block
}

// bilan — la soirée camp contre camp, par ressource ; une ressource sans prise est absente.
func bilan(s *soiree) []domain.SquadEmpriseResource {
	out := []domain.SquadEmpriseResource{}
	if taken := s.obj.total(domain.EmpriseResourcePowerup); s.bonusMatches > 0 && taken.Us+taken.Them > 0 {
		out = append(out, domain.SquadEmpriseResource{
			Resource: domain.EmpriseResourcePowerup, Taken: taken, MatchesMeasured: s.bonusMatches,
			Outcomes: &domain.SquadEmpriseCampOutcomes{
				Us: outcomeCounts(s.outcomes[0]), Them: outcomeCounts(s.outcomes[1]),
			},
		})
	}
	if taken := s.obj.total(domain.EmpriseResourcePowerWeapon); s.tiersMatches > 0 && taken.Us+taken.Them > 0 {
		out = append(out, domain.SquadEmpriseResource{
			Resource: domain.EmpriseResourcePowerWeapon, Taken: taken, MatchesMeasured: s.tiersMatches,
		})
	}
	if taken := s.obj.total(domain.EmpriseResourceVehicle); s.veh.measured > 0 && taken.Us+taken.Them > 0 {
		out = append(out, domain.SquadEmpriseResource{
			Resource: domain.EmpriseResourceVehicle, Taken: taken, MatchesMeasured: s.veh.measured,
		})
	}
	return out
}

func estVehicule(resource string) bool { return resource == domain.EmpriseResourceVehicle }

func outcomeCounts(o sessionusage.OutcomeCounts) domain.SquadEmpriseOutcomeCounts {
	return domain.SquadEmpriseOutcomeCounts{Taken: o.Taken, Used: o.Used, Kept: o.Kept, Dropped: o.Dropped}
}

// chronologique — une copie triée par heure de début (identifiant en départage).
func chronologique(matches []Match) []Match {
	out := append([]Match(nil), matches...)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].StartTime.Equal(out[j].StartTime) {
			return out[i].StartTime.Before(out[j].StartTime)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}
