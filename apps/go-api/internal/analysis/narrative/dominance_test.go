package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestResolveDominanceBadge_KnownFlags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		flag       canonical.DominanceFlag
		labelKey   string
		colorToken string
	}{
		{canonical.DominanceDomination, "narrative.dominance.domination", "narrative.dominance.win.strong"},
		{canonical.DominanceHumiliation, "narrative.dominance.humiliation", "narrative.dominance.loss.strong"},
		{canonical.DominanceRemontada, "narrative.dominance.remontada", "narrative.dominance.win.comeback"},
		{canonical.DominanceDebandade, "narrative.dominance.debandade", "narrative.dominance.loss.collapse"},
		{canonical.DominanceContreRem, "narrative.dominance.contre_remontada", "narrative.dominance.win.counter"},
	}
	for _, c := range cases {
		c := c
		t.Run(string(c.labelKey), func(t *testing.T) {
			t.Parallel()
			got := ResolveDominanceBadge(c.flag)
			if got == nil {
				t.Fatalf("expected badge for flag %v, got nil", c.flag)
			}
			if got.Flag != c.flag {
				t.Errorf("Flag mismatch: want %v got %v", c.flag, got.Flag)
			}
			if got.LabelKey != c.labelKey {
				t.Errorf("LabelKey: want %q got %q", c.labelKey, got.LabelKey)
			}
			if got.ColorToken != c.colorToken {
				t.Errorf("ColorToken: want %q got %q", c.colorToken, got.ColorToken)
			}
		})
	}
}

func TestResolveDominanceBadge_None(t *testing.T) {
	t.Parallel()
	if got := ResolveDominanceBadge(canonical.DominanceNone); got != nil {
		t.Errorf("DominanceNone should return nil, got %+v", got)
	}
}

func TestResolveDominanceBadge_Unknown(t *testing.T) {
	t.Parallel()
	if got := ResolveDominanceBadge(canonical.DominanceFlag(99)); got != nil {
		t.Errorf("unknown flag should return nil, got %+v", got)
	}
	if got := ResolveDominanceBadge(canonical.DominanceFlag(-1)); got != nil {
		t.Errorf("negative flag should return nil, got %+v", got)
	}
}
