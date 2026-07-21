package scheduler

// data_health_lusr_test.go — tests unitaires de la décision auto-heal LUSR
// (kill-switch + seuil + hook), sans DB. Le scan de bout en bout est couvert par
// data_health_lusr_e2e_test.go.

import (
	"context"
	"os"
	"testing"
)

const autoHealEnvFlag = "LEVELUP_LUSR_AUTOHEAL_ENABLED"

func TestIsLUSRAutoHealEnabled(t *testing.T) {
	orig := os.Getenv(autoHealEnvFlag)
	t.Cleanup(func() { _ = os.Setenv(autoHealEnvFlag, orig) })

	cases := []struct {
		v    string
		want bool
	}{
		{"", false}, {"0", false}, {"false", false}, {"no", false}, {"random", false},
		{"1", true}, {"true", true}, {"TRUE", true}, {"yes", true}, {"  1  ", true},
	}
	for _, c := range cases {
		_ = os.Setenv(autoHealEnvFlag, c.v)
		if got := isLUSRAutoHealEnabled(); got != c.want {
			t.Errorf("isLUSRAutoHealEnabled(env=%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

// TestMaybeAutoHealLUSR_Decision couvre la matrice de déclenchement : le replay ne
// fire QUE si (hook câblé ET flag ON ET gamertag non vide ET impact ≥ seuil).
func TestMaybeAutoHealLUSR_Decision(t *testing.T) {
	orig := os.Getenv(autoHealEnvFlag)
	t.Cleanup(func() { _ = os.Setenv(autoHealEnvFlag, orig) })

	newSched := func(hook func(context.Context, string, string) error) *HealthScheduler {
		s := NewDataHealthScheduler(t.TempDir())
		if hook != nil {
			s.WithLUSRAutoHeal(hook)
		}
		return s
	}
	atThreshold := lusrHealCandidate{titleSlug: "halo_infinite", gamertag: "GT", interiorGaps: lusrAutoHealMinGaps}

	t.Run("flag OFF → pas de fire", func(t *testing.T) {
		_ = os.Setenv(autoHealEnvFlag, "0")
		calls := 0
		newSched(func(context.Context, string, string) error { calls++; return nil }).
			maybeAutoHealLUSR(context.Background(), atThreshold)
		if calls != 0 {
			t.Errorf("flag off : %d appel(s), want 0", calls)
		}
	})

	t.Run("flag ON + hook nil → ni panic ni fire", func(t *testing.T) {
		_ = os.Setenv(autoHealEnvFlag, "1")
		newSched(nil).maybeAutoHealLUSR(context.Background(), atThreshold) // ne doit pas paniquer
	})

	t.Run("flag ON + sous le seuil → pas de fire", func(t *testing.T) {
		_ = os.Setenv(autoHealEnvFlag, "1")
		calls := 0
		newSched(func(context.Context, string, string) error { calls++; return nil }).
			maybeAutoHealLUSR(context.Background(), lusrHealCandidate{
				titleSlug: "halo_infinite", gamertag: "GT", interiorGaps: lusrAutoHealMinGaps - 1,
			})
		if calls != 0 {
			t.Errorf("sous le seuil : %d appel(s), want 0", calls)
		}
	})

	t.Run("flag ON + gamertag vide → pas de fire", func(t *testing.T) {
		_ = os.Setenv(autoHealEnvFlag, "1")
		calls := 0
		newSched(func(context.Context, string, string) error { calls++; return nil }).
			maybeAutoHealLUSR(context.Background(), lusrHealCandidate{interiorGaps: 99})
		if calls != 0 {
			t.Errorf("gamertag vide : %d appel(s), want 0", calls)
		}
	})

	t.Run("flag ON + au seuil + gamertag → fire avec les bons args", func(t *testing.T) {
		_ = os.Setenv(autoHealEnvFlag, "1")
		var gotTitle, gotGT string
		calls := 0
		newSched(func(_ context.Context, title, gt string) error {
			gotTitle, gotGT = title, gt
			calls++
			return nil
		}).maybeAutoHealLUSR(context.Background(), atThreshold)
		if calls != 1 {
			t.Fatalf("%d appel(s), want 1", calls)
		}
		if gotTitle != "halo_infinite" || gotGT != "GT" {
			t.Errorf("args = (%q, %q), want (halo_infinite, GT)", gotTitle, gotGT)
		}
	})
}
