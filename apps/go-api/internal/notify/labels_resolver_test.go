package notify

import "testing"

type fakeLabels struct{ name string }

func (f fakeLabels) Outcome(_, _ string) string { return "X" }
func (f fakeLabels) TitleName() string          { return f.name }

func TestLabelsForSlug_FailsafeAndRouting(t *testing.T) {
	t.Cleanup(func() { SetDefaultLabelsResolver(nil) })

	// 1. Resolver non câblé → HaloLabels (byte-identique Halo).
	SetDefaultLabelsResolver(nil)
	if got := LabelsForSlug("whatever"); got.TitleName() != HaloLabels().TitleName() {
		t.Errorf("sans resolver, TitleName = %q ; want HaloLabels", got.TitleName())
	}

	// 2. Resolver câblé → route le titre.
	SetDefaultLabelsResolver(func(slug string) NotifyLabels {
		if slug == "synthetic_title_b" {
			return fakeLabels{name: "Synthetic Title B"}
		}
		return nil
	})
	if got := LabelsForSlug("synthetic_title_b"); got.TitleName() != "Synthetic Title B" {
		t.Errorf("routing TitleName = %q ; want Synthetic Title B", got.TitleName())
	}

	// 3. Resolver renvoie nil pour un titre inconnu → HaloLabels (failsafe).
	if got := LabelsForSlug("unknown"); got.TitleName() != HaloLabels().TitleName() {
		t.Errorf("nil resolver result → HaloLabels attendu, got %q", got.TitleName())
	}
}
