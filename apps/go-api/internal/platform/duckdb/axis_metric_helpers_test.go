// Package duckdb — axis_metric_helpers_test.go : test pur (no DB) du mapping
// axe radar campaign -> expression SQL (axisValueExpression). Le test frère
// TestMapMetricToColumn a suivi mapMetricToColumn dans le sous-package
// duckdb/prestige lors de l'extraction K3a.
package duckdb

import (
	"testing"

	"levelup/go-api/internal/campaign"
)

// TestAxisValueExpression : mapping axis radar → expression SQL.
// Seuls 5 axes radar sont mappés sur une colonne. Tout le reste retourne ("", false).
func TestAxisValueExpression(t *testing.T) {
	tests := []struct {
		axis    string
		kind    campaign.AxisKind
		wantSQL string
		wantOK  bool
	}{
		{"combat", campaign.AxisKindRadar, "CAST(mp.kills AS DOUBLE)", true},
		{"survival", campaign.AxisKindRadar,
			"GREATEST(CAST(mp.kills AS DOUBLE) - CAST(mp.deaths AS DOUBLE), 0)", true},
		{"support", campaign.AxisKindRadar, "CAST(mp.assists AS DOUBLE)", true},
		{"score", campaign.AxisKindRadar, "CAST(mp.personal_score AS DOUBLE)", true},
		{"impact", campaign.AxisKindRadar, "CAST(mp.max_killing_spree AS DOUBLE) * 10", true},
		// objective : mesuré par match depuis les lignes d'objectif (loadObjectiveSamples), pas par une colonne.
		{"objective", campaign.AxisKindRadar, "", false},
		// Axes inconnus radar → empty/false.
		{"unknown_axis", campaign.AxisKindRadar, "", false},
		{"", campaign.AxisKindRadar, "", false},
		// kind non-radar → empty/false (loadLUSRComponentSamples gère le composant).
		{"combat", campaign.AxisKindLUSRComponent, "", false},
	}
	for _, tc := range tests {
		gotSQL, gotOK := axisValueExpression(tc.axis, tc.kind)
		if gotSQL != tc.wantSQL || gotOK != tc.wantOK {
			t.Errorf("axisValueExpression(%q, %v) = (%q, %v), want (%q, %v)",
				tc.axis, tc.kind, gotSQL, gotOK, tc.wantSQL, tc.wantOK)
		}
	}
}
