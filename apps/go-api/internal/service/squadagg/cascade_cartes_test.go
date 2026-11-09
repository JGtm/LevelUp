package squadagg

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
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
	}).AvecLibelles(map[string]i18n.Text{
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

// Les stats « avec cette escouade » de la base et de sa variante sont UNE entrée, performance pondérée.
func TestFoldMapStats_SommeEtPondere(t *testing.T) {
	groups := domain.NewGroupesDeCartes(map[string]string{"var": "base"})
	p1, p2 := 50.0, 80.0
	got := FoldMapStats(map[string]domain.MapSquadStats{
		"base":  {Wins: 3, Total: 4, PerfAvg: &p1, PerfCount: 3},
		"var":   {Wins: 1, Total: 2, PerfAvg: &p2, PerfCount: 1},
		"seule": {Wins: 1, Total: 1},
	}, groups)
	g := got["base"]
	if g.Wins != 4 || g.Total != 6 || g.PerfAvg == nil || *g.PerfAvg != 57.5 {
		t.Errorf("groupe = %+v (perf %v), attendu 4/6 et perf (50*3+80*1)/4 = 57,5", g, g.PerfAvg)
	}
	if _, ok := got["var"]; ok {
		t.Error("la variante ne doit plus avoir d'entrée propre")
	}
	if got["seule"].Total != 1 {
		t.Errorf("carte seule = %+v", got["seule"])
	}
	if same := FoldMapStats(map[string]domain.MapSquadStats{"a": {Total: 1}}, domain.GroupesDeCartes{}); same["a"].Total != 1 {
		t.Error("sans groupe, la table est rendue telle quelle")
	}
}
