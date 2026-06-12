package prestige

import (
	"reflect"
	"testing"
)

func TestAggregateSquadAxes_MeanPerAxis(t *testing.T) {
	got := AggregateSquadAxes([]map[string]float64{
		{"combat": 0.8, "survival": 0.4, "support": 0.2},
		{"combat": 0.6, "survival": 0.6, "objective": 0.9}, // pas de support, ajoute objective
	})
	// combat: (0.8+0.6)/2=0.7 ; survival: 0.5 ; support: 0.2 (1 seul) ; objective: 0.9 (1 seul)
	want := map[string]float64{"combat": 0.7, "survival": 0.5, "support": 0.2, "objective": 0.9}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("axe %s = %v, want %v", k, got[k], w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("axes = %v, want %d clés", got, len(want))
	}
}

func TestSquadFocusAxis_WeakestAxis(t *testing.T) {
	// support (0.2) est le plus faible → axe à renforcer.
	axes := map[string]float64{"combat": 0.7, "survival": 0.5, "support": 0.2, "objective": 0.9}
	if got := SquadFocusAxis(axes); got != "support" {
		t.Errorf("focus = %q, want support", got)
	}
}

func TestSquadFocusAxis_EmptyReturnsEmpty(t *testing.T) {
	if got := SquadFocusAxis(map[string]float64{}); got != "" {
		t.Errorf("focus = %q, want \"\"", got)
	}
	// Une map sans axe canonique (que des clés inconnues) → "".
	if got := SquadFocusAxis(map[string]float64{"inconnu": 0.1}); got != "" {
		t.Errorf("focus = %q, want \"\" (axe non canonique ignoré)", got)
	}
}

func TestMetricToRadarAxis(t *testing.T) {
	cases := map[string]string{
		"kills_total":          "combat",
		"kda":                  "combat",
		"deaths_total":         "survival",
		"assists":              "support",
		"personal_score":       "score",
		"objective_carry_time": "objective",
		"damage_dealt":         "impact",
		"métrique_inconnue":    "",
	}
	for metric, want := range cases {
		if got := MetricToRadarAxis(metric); got != want {
			t.Errorf("MetricToRadarAxis(%q) = %q, want %q", metric, got, want)
		}
	}
}

func TestBiasTemplatesByAxis_MatchedFirstStable(t *testing.T) {
	templates := []Template{
		{ID: "t_kills", Metric: "kills_total"},     // combat
		{ID: "t_assist", Metric: "assists"},        // support  <- focus
		{ID: "t_deaths", Metric: "deaths_total"},   // survival
		{ID: "t_assist2", Metric: "assists_total"}, // support  <- focus
	}
	got := BiasTemplatesByAxis(templates, "support")
	gotIDs := make([]string, len(got))
	for i, tpl := range got {
		gotIDs[i] = tpl.ID
	}
	// Les 2 support en tête (ordre stable), puis le reste (ordre stable).
	want := []string{"t_assist", "t_assist2", "t_kills", "t_deaths"}
	if !reflect.DeepEqual(gotIDs, want) {
		t.Errorf("ordre = %v, want %v", gotIDs, want)
	}
}

func TestBiasTemplatesByAxis_EmptyFocusUnchanged(t *testing.T) {
	templates := []Template{{ID: "a", Metric: "kills_total"}, {ID: "b", Metric: "assists"}}
	got := BiasTemplatesByAxis(templates, "")
	if !reflect.DeepEqual(got, templates) {
		t.Errorf("focus vide doit laisser inchangé, got %v", got)
	}
}
