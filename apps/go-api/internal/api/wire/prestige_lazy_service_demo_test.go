package wire

import (
	"context"
	"testing"
)

// TestGetUserPrestige_DemoMode : en DemoMode, GetUserPrestige sert la fixture
// (rang Prestige) sans toucher au bundle/player DB.
func TestGetUserPrestige_DemoMode(t *testing.T) {
	// bundle nil : en DemoMode on ne doit jamais le déréférencer (return anticipé).
	l := NewLazyPrestigeService(nil, nil, true)

	up, err := l.GetUserPrestige(context.Background(), "0000000000000000", "halo_infinite")
	if err != nil {
		t.Fatalf("demo prestige: erreur inattendue %v", err)
	}
	if up.UserID != "0000000000000000" || up.TitleSlug != "halo_infinite" {
		t.Errorf("demo prestige: user/title = %q/%q", up.UserID, up.TitleSlug)
	}
	if up.TotalPP <= 0 || up.Level == nil || up.Level.Name == "" {
		t.Errorf("demo prestige: fixture incomplète (pp=%d level=%v)", up.TotalPP, up.Level)
	}
}
