package analysis

import "testing"

func TestResolvePairNameFR(t *testing.T) {
	modes := map[string]string{
		"CTF":              "Capture du drapeau",
		"Strongholds":      "Bases",
		"Slayer":           "Assassin",
		"Team Slayer":      "Assassin en équipe",
		"Neutral Flag CTF": "Drapeau neutre",
	}

	tests := []struct {
		name      string
		rawPair   string
		currentFR string
		assetName string
		want      string
	}{
		{
			name:      "priorite 1 - mode_name_tr depuis pair_name brut",
			rawPair:   "Arena:CTF on Aquarius",
			currentFR: "Arena:CTF on Aquarius", // placeholder COALESCE
			assetName: "",
			want:      "Capture du drapeau",
		},
		{
			name:      "priorite 1 - BTB:Strongholds → Bases",
			rawPair:   "BTB:Strongholds on Highpower",
			currentFR: "BTB:Strongholds on Highpower",
			assetName: "",
			want:      "Bases",
		},
		{
			name:      "priorite 2 - rawPair vide, asset normalisable → re-lookup",
			rawPair:   "",
			currentFR: "",
			assetName: "Arena:CTF on Shiro",
			want:      "Capture du drapeau",
		},
		{
			name:      "priorite 2 - rawPair UUID, asset normalisable",
			rawPair:   "bd1457cc-4fd8-4da1-be87-381c142017e8",
			currentFR: "bd1457cc-4fd8-4da1-be87-381c142017e8",
			assetName: "Arena:Team Slayer on Bazaar - Forge",
			want:      "Assassin en équipe",
		},
		{
			name:      "priorite 3 - asset present mais pas dans mode_name_tr, currentFR vide",
			rawPair:   "",
			currentFR: "",
			assetName: "Castle Wars",
			want:      "Castle Wars",
		},
		{
			name:      "priorite 3 - asset present, currentFR == EN (placeholder)",
			rawPair:   "Custom:Unknown",
			currentFR: "Custom:Unknown",
			assetName: "Mode Custom Inconnu",
			want:      "Mode Custom Inconnu",
		},
		{
			name:      "preservation - currentFR vrai FR, pas d'override raw",
			rawPair:   "Custom:Unknown",
			currentFR: "Mode Personnalisé",
			assetName: "Custom:Unknown", // EN raw, ne doit pas écraser le FR existant
			want:      "Mode Personnalisé",
		},
		{
			name:      "tout vide → string vide",
			rawPair:   "",
			currentFR: "",
			assetName: "",
			want:      "",
		},
		{
			name:      "currentFR seul (pas de modeNamesFR ni asset)",
			rawPair:   "Some:Mode",
			currentFR: "Mode Connu",
			assetName: "",
			want:      "Mode Connu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePairNameFR(tt.rawPair, tt.currentFR, tt.assetName, modes)
			if got != tt.want {
				t.Errorf("ResolvePairNameFR(%q, %q, %q) = %q, want %q",
					tt.rawPair, tt.currentFR, tt.assetName, got, tt.want)
			}
		})
	}
}

func TestResolvePairNameFR_PreservesIdentityPrefixesViaNormalize(t *testing.T) {
	// Super Fiesta:Slayer n'a pas d'entrée mode_name_tr (intentionnel — c'est
	// un playlist identity prefix). Le helper doit donc préserver le raw EN
	// (currentFR), et la normalisation finale (qui extrait "Super Fiesta")
	// est faite par les consumers downstream via NormalizeModeLabel.
	modes := map[string]string{
		"Slayer": "Assassin",
	}
	raw := "Super Fiesta:Slayer on Streets - Forge"
	got := ResolvePairNameFR(raw, raw, "", modes)
	if got != raw {
		t.Errorf("got %q, want %q (raw EN preserved when no FR available)", got, raw)
	}
	// La normalisation downstream produit bien "Super Fiesta" (identity prefix).
	if normalized := NormalizeModeLabel(got); normalized != "Super Fiesta" {
		t.Errorf("downstream NormalizeModeLabel(%q) = %q, want %q", got, normalized, "Super Fiesta")
	}
}

func TestNeedsFRTranslationOverride(t *testing.T) {
	tests := []struct {
		name   string
		fr, en string
		want   bool
	}{
		{"FR vide → override", "", "Arena:CTF", true},
		{"FR == EN → override (COALESCE placeholder)", "Arena:CTF", "Arena:CTF", true},
		{"FR == EN ignore case", "ARENA:CTF", "arena:ctf", true},
		{"FR vrai FR → preserve", "Capture du drapeau", "Arena:CTF", false},
		{"FR rempli, EN vide → preserve", "Mode FR", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsFRTranslationOverride(tt.fr, tt.en)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
