package squademprise

// maps.go — LA GRILLE PAR CARTE (Séries temporelles › Usages, « Contrôle des ressources, carte par
// carte ») : une colonne par carte jouée, la plus jouée d'abord, puis une colonne « Autres cartes »
// qui somme le reste au-delà de domain.EmpriseGridMaxMaps.
//
// UNE COLONNE EST UNE SOIRÉE SUR LES MATCHS DE LA CARTE : `tallyMatch` puis `soiree.add`, les MÊMES
// règles de lisibilité que le bilan et la grille match par match (un objet ne se verse que là où
// il se lit : film et camp connu, niveaux de socle pour les armes, couverture pour les véhicules).

import (
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// carteJouee — les matchs du périmètre sur une carte.
type carteJouee struct {
	key, label string
	matches    []Match
}

// BuildMaps rend les colonnes de la grille par carte, depuis les matchs du périmètre (Current).
func BuildMaps(in Input) []domain.EmpriseMapColumn {
	ix := newIndex(&in)
	if in.Players == nil {
		in.Players = []domain.SessionUsageSquadPlayer{}
	}
	cartes := cartesJouees(chronologique(in.Current))
	out := make([]domain.EmpriseMapColumn, 0, min(len(cartes), domain.EmpriseGridMaxMaps+1))
	if len(cartes) <= domain.EmpriseGridMaxMaps+1 {
		for _, c := range cartes {
			out = append(out, colonneDeCarte(c.key, c.label, c.matches, ix, &in))
		}
		return out
	}
	for _, c := range cartes[:domain.EmpriseGridMaxMaps] {
		out = append(out, colonneDeCarte(c.key, c.label, c.matches, ix, &in))
	}
	var reste []Match
	for _, c := range cartes[domain.EmpriseGridMaxMaps:] {
		reste = append(reste, c.matches...)
	}
	autres := colonneDeCarte("", "", reste, ix, &in)
	autres.OtherMaps = len(cartes) - domain.EmpriseGridMaxMaps
	return append(out, autres)
}

// cartesJouees groupe les matchs par carte (clé, sinon libellé) et les range : plus de matchs
// d'abord, puis libellé, puis clé — un ordre total.
func cartesJouees(matches []Match) []carteJouee {
	index := map[string]int{}
	var out []carteJouee
	for _, m := range matches {
		id := m.MapKey
		if id == "" {
			id = "\x00" + m.MapLabel
		}
		i, ok := index[id]
		if !ok {
			i = len(out)
			index[id] = i
			out = append(out, carteJouee{key: m.MapKey, label: m.MapLabel})
		}
		out[i].matches = append(out[i].matches, m)
	}
	sort.SliceStable(out, func(a, b int) bool {
		if len(out[a].matches) != len(out[b].matches) {
			return len(out[a].matches) > len(out[b].matches)
		}
		if out[a].label != out[b].label {
			return out[a].label < out[b].label
		}
		return out[a].key < out[b].key
	})
	return out
}

// colonneDeCarte somme des matchs en une colonne.
func colonneDeCarte(key, label string, matches []Match, ix *index, in *Input) domain.EmpriseMapColumn {
	col := domain.EmpriseMapColumn{
		MapKey: key, MapLabel: label, Matches: len(matches),
		Resources: []domain.SquadEmpriseMatchResource{},
	}
	s := newSoiree()
	for _, m := range matches {
		t := tallyMatch(m.MatchID, ix)
		if t.hasFilm {
			col.MatchesFilmed++
		}
		s.add(t)
		switch m.Outcome {
		case canonical.OutcomeWin:
			col.Wins++
		case canonical.OutcomeLoss:
			col.Losses++
		default:
			col.Others++
		}
	}
	col.MatchesMeasured, col.MatchesTiers, col.VehiclesMeasured = s.bonusMatches, s.tiersMatches, s.veh.measured
	for _, res := range resourceOrder {
		if !s.obj.has(res) {
			continue
		}
		col.Resources = append(col.Resources, domain.SquadEmpriseMatchResource{
			Resource: res, Taken: s.obj.total(res), Objects: s.obj.publier(res, in.Players, in),
		})
	}
	if s.pwkMatches > 0 {
		pwk := s.pwk
		col.PowerWeaponKills = &pwk
	}
	return col
}
