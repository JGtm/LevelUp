package analysis

import "testing"

func TestIsBot(t *testing.T) {
	cases := []struct {
		name string
		xuid string
		want bool
	}{
		{"empty", "", false},
		{"human xuid 17 digits", "12345678901234567", false},
		{"human xuid with leading zero", "00000000000000001", false},
		{"bot prefix bid(", "bid(0)", true},
		{"bot bid(42)", "bid(42)", true},
		{"bot bid(WallE-1)", "bid(WallE-1)", true},
		{"prefix close but not equal", "bot(123)", false},
		{"contains bid( in middle", "12345bid(99)", false},
		{"only prefix", "bid(", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsBot(tc.xuid)
			if got != tc.want {
				t.Errorf("IsBot(%q) = %v, want %v", tc.xuid, got, tc.want)
			}
		})
	}
}

func TestBotDisplayName(t *testing.T) {
	cases := []struct {
		xuid string
		want string
	}{
		{"bid(3.0)", "343 Bot 3"},
		{"bid(18.0)", "343 Bot 18"},
		{"bid(35.0)", "343 Bot 35"},
		{"bid(0.0)", "343 Bot 0"},
		{"bid(3.0", "343 Bot 3"},                   // paren manquante (API bug)
		{"bid(WallE-1)", "bid(WallE-1)"},           // pas de numéro → retourne le xuid brut
		{"12345678901234567", "12345678901234567"}, // humain → inchangé
	}
	for _, tc := range cases {
		t.Run(tc.xuid, func(t *testing.T) {
			got := BotDisplayName(tc.xuid)
			if got != tc.want {
				t.Errorf("BotDisplayName(%q) = %q, want %q", tc.xuid, got, tc.want)
			}
		})
	}
}
