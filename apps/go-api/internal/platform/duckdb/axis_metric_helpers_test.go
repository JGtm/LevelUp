// Package duckdb — axis_metric_helpers_test.go : tests purs (no DB) pour
// les helpers de mapping métrique/axe vers colonnes SQL.
package duckdb

import (
	"testing"

	"levelup/go-api/internal/campaign"
)

// TestMapMetricToColumn : mapping FieldKey/lowercase → colonne match_participants.
// Couvre l'ensemble des cas listés dans le switch + cas inconnu → "".
func TestMapMetricToColumn(t *testing.T) {
	tests := []struct {
		metric string
		want   string
	}{
		// FieldKey canonical
		{"FieldKDA", "kda"},
		{"FieldKDR", "kd"},
		{"FieldAccuracy", "accuracy"},
		{"FieldKills", "kills"},
		{"FieldDeaths", "deaths"},
		{"FieldAssists", "assists"},
		{"FieldHeadshotKills", "headshot_kills"},
		{"FieldMeleeKills", "melee_kills"},
		{"FieldGrenadeKills", "grenade_kills"},
		{"FieldPowerWeaponKills", "power_weapon_kills"},
		{"FieldDamageDealt", "damage_dealt"},
		{"FieldPersonalScore", "personal_score"},
		{"FieldMaxKillingSpree", "max_killing_spree"},
		// Alias lowercase équivalents
		{"kda", "kda"},
		{"kd", "kd"},
		{"accuracy", "accuracy"},
		{"kills", "kills"},
		{"deaths", "deaths"},
		// Whitespace toléré
		{"  kills  ", "kills"},
		// Inconnu → empty
		{"unknown_metric", ""},
		{"", ""},
		{"FieldEngagementScore", ""},
	}
	for _, tc := range tests {
		if got := mapMetricToColumn(tc.metric); got != tc.want {
			t.Errorf("mapMetricToColumn(%q) = %q, want %q", tc.metric, got, tc.want)
		}
	}
}

// TestAxisValueExpression : mapping axis radar → expression SQL.
// Seuls les 6 axes radar V1 sont mappés. Tout le reste retourne ("", false).
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
		// objective V1 : placeholder constant (V2 commit-2 prévu pour le mapping awards).
		{"objective", campaign.AxisKindRadar, "0.0", true},
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
