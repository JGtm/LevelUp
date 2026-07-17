package prestigetuning

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderText_ContainsSections(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 60, Completed: 6},
	}
	rep := Analyze(counts, []SourceAcceptance{{Source: "coach", Created: 60, Rejected: 5, AcceptanceRate: 0.92}},
		testGrammar(), DefaultThresholds(), fixedNow)
	rep.TitleSlug = "halo_infinite"
	rep.PlayerScope = "all"
	rep.PlayersScanned = 4

	txt := RenderText(rep)
	for _, want := range []string{"Analyse de tuning", "Acceptation par origine", "accuracy", "AJUSTER", "Synthèse"} {
		if !strings.Contains(txt, want) {
			t.Errorf("texte ne contient pas %q\n---\n%s", want, txt)
		}
	}
}

func TestRenderJSON_Valid(t *testing.T) {
	rep := Analyze(nil, nil, testGrammar(), DefaultThresholds(), fixedNow)
	out, err := RenderJSON(rep)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var back Report
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if len(back.Metrics) != len(rep.Metrics) {
		t.Errorf("roundtrip metrics = %d, want %d", len(back.Metrics), len(rep.Metrics))
	}
}
