//go:build integration

// campaign_exclusion_lusr_test.go - E6.2a-F1 : la chronologie d'une composante LUSR
// (CampaignSampleProvider.loadLUSRMatchIDsInWindow) masque la Campagne comme le fait
// LoadAxisSamples. Un match de Campagne dont la composante est presente cote player DB
// ne doit pas entrer dans les echantillons ; Halo Infinite (aucune variante masquee)
// reste neutre. Fixture : newB4PlayerDB (arena1 01/09, camp1 02/09).

package duckdb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
)

func TestCampaignExclusion_LoadAxisSamplesLUSRComponent(t *testing.T) {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()
	for _, tc := range b4Titres {
		pdb := newB4PlayerDB(t, tc.slug)
		if _, err := pdb.Player.Exec(ctx, `CREATE TABLE lusr_component_history_latest (
			match_id VARCHAR, component_name VARCHAR, value DOUBLE)`); err != nil {
			t.Fatalf("%s : fixture lusr : %v", tc.slug, err)
		}
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO lusr_component_history_latest VALUES
			('arena1', 'kills_vs_expected', 0.4), ('camp1', 'kills_vs_expected', 0.9)`); err != nil {
			t.Fatalf("%s : fixture lusr : %v", tc.slug, err)
		}
		got, err := NewCampaignSampleProvider(pdb).LoadAxisSamples(ctx, b4XUIDA, tc.slug,
			"kills_vs_expected", campaign.AxisKindLUSRComponent, "all", since, until)
		if err != nil {
			t.Fatalf("%s : LoadAxisSamples LUSR : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, []float64{0.4}, []float64{0.4, 0.9}); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s : echantillons LUSR %v, attendu %v (Campagne masquee : %v)", tc.slug, got, want, tc.masquee)
		}
	}
}
