package prestigetuning

import "testing"

func TestMergeCounts_SumsAcrossPlayers(t *testing.T) {
	in := []MetricWindowCount{
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10", Created: 10, Completed: 3},
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10", Created: 5, Completed: 2},
		{Source: "coach", Metric: "kda", WindowType: "session", WindowValue: "", Created: 4, Completed: 1},
	}
	out := MergeCounts(in)
	if len(out) != 2 {
		t.Fatalf("merged len = %d, want 2", len(out))
	}
	var acc MetricWindowCount
	for _, c := range out {
		if c.Metric == "accuracy" {
			acc = c
		}
	}
	if acc.Created != 15 || acc.Completed != 5 {
		t.Errorf("accuracy merged = created %d/completed %d, want 15/5", acc.Created, acc.Completed)
	}
}

func TestMergeAcceptance_RecomputesRate(t *testing.T) {
	in := []SourceAcceptance{
		{Source: "coach", Created: 30, Rejected: 10},
		{Source: "coach", Created: 20, Rejected: 0},
		{Source: "user", Created: 5, Rejected: 5},
	}
	out := MergeAcceptance(in)
	var coach SourceAcceptance
	for _, a := range out {
		if a.Source == "coach" {
			coach = a
		}
	}
	if coach.Created != 50 || coach.Rejected != 10 {
		t.Fatalf("coach merged = %d/%d, want 50/10", coach.Created, coach.Rejected)
	}
	if got := coach.AcceptanceRate; got < 0.83 || got > 0.834 {
		t.Errorf("acceptance = %.4f, want ~0.833", got)
	}
}
