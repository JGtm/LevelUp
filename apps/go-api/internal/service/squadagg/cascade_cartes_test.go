package squadagg

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func rowCarte(matchID, mapID, en, fr string) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{
		MatchID: matchID,
		Map:     &canonical.AssetReference{Kind: "map", ID: mapID, DefaultLabel: en, Labels: map[string]string{"fr": fr}},
	}}
}

func idsDesLignes(rows []canonical.PlayerMatchRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Summary.MatchID)
	}
	return out
}

// Choisir la carte de base dans le filtre retient aussi ses variantes (et les autres identifiants de la
// base) ; une carte hors groupe reste filtrée par son propre libellé.
func TestFilterRowsByCascade_CarteDeBaseRetientSesVariantes(t *testing.T) {
	groups := domain.NewGroupesDeCartes(map[string]string{
		"th-heavies": "th", "th-bis": "th", "lf-ranked": "lf",
	}).AvecLibelles(map[string]map[string]string{
		"th": {"fr": "Thunderhead", "en": "Thunderhead"},
		"lf": {"fr": "Live Fire", "en": "Live Fire"},
	})
	rows := []canonical.PlayerMatchRow{
		rowCarte("a", "th", "Thunderhead", "Thunderhead"),
		rowCarte("b", "th-heavies", "Thunderhead Heavies", "Du lourd sur Thunderhead"),
		rowCarte("c", "th-bis", "Thunderhead", "Thunderhead"),
		rowCarte("d", "lf-ranked", "Live Fire - Ranked", "Live Fire - Classée"),
		rowCarte("e", "lf", "Live Fire", "Live Fire"),
		rowCarte("f", "pit", "The Pit", "La Fosse"),
	}
	cas := []struct {
		maps []string
		want []string
	}{
		{[]string{"Thunderhead"}, []string{"a", "b", "c"}},
		{[]string{"Live Fire"}, []string{"d", "e"}},
		{[]string{"La Fosse"}, []string{"f"}},
	}
	for _, c := range cas {
		got := idsDesLignes(FilterRowsByCascade(rows, domain.CascadeFilter{Maps: c.maps}, groups))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("filtre %v = %v, attendu %v", c.maps, got, c.want)
		}
	}
	// Sans groupe (valeur zéro) : le comportement d'avant, chaque carte par son libellé.
	if got := idsDesLignes(FilterRowsByCascade(rows, domain.CascadeFilter{Maps: []string{"Thunderhead"}}, domain.GroupesDeCartes{})); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Errorf("sans groupe : %v, attendu [a c]", got)
	}
}
