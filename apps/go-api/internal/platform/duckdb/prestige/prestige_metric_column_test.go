// Package prestige — prestige_metric_column_test.go : test pur (no DB) du mapping
// FieldKey canonique -> colonne match_participants (mapMetricToColumn). Extrait de
// duckdb/axis_metric_helpers_test.go lors de l'extraction du sous-package prestige
// (K3a) : mapMetricToColumn a suivi prestige_baseline_provider.go ici, alors que
// axisValueExpression (radar campaign) reste dans duckdb-root.
package prestige

import "testing"

// TestMapMetricToColumn : mapping FieldKey/lowercase -> colonne match_participants.
// Couvre l'ensemble des cas listés dans le switch + cas inconnu -> "".
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
