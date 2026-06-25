package canonical

import "testing"

// TestNormalizeMedalCategory_RawH5Enums vérifie que les enums bruts de l'API
// Halo 5 (Classification, PascalCase) sont mappés vers les clés canoniques.
func TestNormalizeMedalCategory_RawH5Enums(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"multikill", "MultiKill", MedalCategoryMultikill},
		{"killing spree", "KillingSpree", MedalCategorySpree},
		{"weapon proficiency", "WeaponProficiency", MedalCategoryProficiency},
		{"style", "Style", MedalCategoryStyle},
		{"capture the flag -> mode", "CaptureTheFlag", MedalCategoryMode},
		{"oddball -> mode", "Oddball", MedalCategoryMode},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeMedalCategory(tc.raw); got != tc.want {
				t.Errorf("NormalizeMedalCategory(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeMedalCategory_Idempotent vérifie que les valeurs déjà canoniques
// (Halo Infinite normalise en amont) ne sont pas re-mappées.
func TestNormalizeMedalCategory_Idempotent(t *testing.T) {
	t.Parallel()
	for _, key := range AllMedalCategories() {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeMedalCategory(key); got != key {
				t.Errorf("NormalizeMedalCategory(%q) = %q, want %q (idempotent)", key, got, key)
			}
		})
	}
}

// TestNormalizeMedalCategory_CaseInsensitive : la casse ne doit pas changer la clé.
func TestNormalizeMedalCategory_CaseInsensitive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
	}{
		{"MULTIKILL", MedalCategoryMultikill},
		{"multikill", MedalCategoryMultikill},
		{"MultiKill", MedalCategoryMultikill},
		{"  MULTIKILL  ", MedalCategoryMultikill}, // trim + lower
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeMedalCategory(tc.raw); got != tc.want {
				t.Errorf("NormalizeMedalCategory(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeMedalCategory_UnknownAndEmptyFallbackOther : valeur inconnue ou
// vide retombe sur "other" (clé existante côté frontend).
func TestNormalizeMedalCategory_UnknownAndEmptyFallbackOther(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"unknown enum", "ZzzUnknown"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeMedalCategory(tc.raw); got != MedalCategoryOther {
				t.Errorf("NormalizeMedalCategory(%q) = %q, want %q (fallback)", tc.raw, got, MedalCategoryOther)
			}
		})
	}
}
