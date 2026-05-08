// Tests pour NormalizeModeLabel — extraction canonique du label de mode
// depuis un pair_name brut Halo Infinite.
package analysis

import "testing"

func TestNormalizeModeLabel(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		// Format technique standard — extrait le sous-mode (containers Assassin)
		{"arena slayer", "Arena:Slayer on Bazaar", "Slayer"},
		{"tactical slayer", "Tactical:Slayer on Recharge", "Slayer"},
		{"assault ctf", "Assault:CTF on Highpower", "CTF"},
		{"community oddball", "Community:Oddball on Live Fire", "Oddball"},
		{"btb ctf", "BTB:CTF on Highpower", "CTF"},
		{"btb heavies slayer", "BTB Heavies:Slayer on Fortitude", "Slayer"},
		{"ranked slayer", "Ranked:Slayer on Aquarius", "Slayer"},
		{"firefight king", "Firefight:King of the Hill on Argyle", "King of the Hill"},

		// Régression "Super Fiesta affiché Assassin" : préfixe playlist-identity
		// → on garde le préfixe au lieu d'extraire "Slayer" (qui se traduit FR en
		// "Assassin" via mode_name_tr et masque l'identité de la playlist).
		{"super fiesta slayer", "Super Fiesta:Slayer on Forbidden - Forge", "Super Fiesta"},
		{"super fiesta forest", "Super Fiesta:Slayer on Forest", "Super Fiesta"},
		{"husky raid ctf", "Husky Raid:CTF on Pharaoh", "Husky Raid"},
		{"super husky raid ctf", "Super Husky Raid:CTF on Pharaoh", "Super Husky Raid"},
		// Casse non standard depuis l'API : on normalise via la map case-insensitive.
		{"super fiesta lowercase", "super fiesta:Slayer on Forbidden", "Super Fiesta"},

		// Format FR avec séparateur espacé " : " → prend avant
		{"fr assassin classe", "Assassin : Classé", "Assassin"},

		// Sans séparateur : on retourne tel quel après strip map/forge
		{"husky raid sans sub", "Husky Raid", "Husky Raid"},
		{"sniper slayer", "Sniper Slayer", "Sniper Slayer"},

		// Strip " - Forge" et " - Ranked"
		{"strip forge", "Arena:Slayer on Bazaar - Forge", "Slayer"},
		{"strip ranked", "Arena:Slayer on Bazaar - Ranked", "Slayer"},

		// Cas vides / dégénérés
		{"empty", "", ""},
		{"only colon", ":", ":"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeModeLabel(tc.raw)
			if got != tc.want {
				t.Errorf("NormalizeModeLabel(%q) = %q ; want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeModeLabel_StripsKnownMap vérifie que le strip map prioritaire
// (mapLabels) tombe avant l'extraction du préfixe.
func TestNormalizeModeLabel_StripsKnownMap(t *testing.T) {
	got := NormalizeModeLabel("Arena:Slayer on Bazaar", "Bazaar")
	if got != "Slayer" {
		t.Errorf("NormalizeModeLabel(Arena:Slayer on Bazaar, Bazaar) = %q ; want Slayer", got)
	}

	// Avec préfixe playlist-identity : le strip map ne doit PAS empêcher la
	// préservation du préfixe.
	got = NormalizeModeLabel("Super Fiesta:Slayer on Forbidden", "Forbidden")
	if got != "Super Fiesta" {
		t.Errorf("NormalizeModeLabel(Super Fiesta:Slayer on Forbidden, Forbidden) = %q ; want Super Fiesta", got)
	}
}
