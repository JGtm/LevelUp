package analysis

import (
	"strings"
	"testing"
)

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

// ─── BotSQLCase ─────────────────────────────────────────────────────────

func TestBotSQLCase_ContainsCaseHeader(t *testing.T) {
	t.Parallel()
	sql := BotSQLCase("xa.xuid")
	if !strings.HasPrefix(sql, "CASE xa.xuid") {
		t.Errorf("BotSQLCase doit commencer par 'CASE xa.xuid', got: %q", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "END") {
		t.Errorf("BotSQLCase doit se terminer par 'END', got: %q", sql)
	}
}

func TestBotSQLCase_IncludesAllBots(t *testing.T) {
	t.Parallel()
	sql := BotSQLCase("xuid")
	// Vérifie qu'un échantillon de bots connus est bien mappé.
	checks := map[string]string{
		"bid(0.0)":  "343 Ritzy",
		"bid(35.0)": "343 BF Scrub",
		"bid(59.0)": "343 Hollis",
	}
	for k, v := range checks {
		// Forme attendue : WHEN 'bid(X)' THEN 'Name'
		want := "WHEN '" + k + "' THEN '" + v + "'"
		if !strings.Contains(sql, want) {
			t.Errorf("BotSQLCase missing %q", want)
		}
	}
}

func TestBotSQLCase_ContainsElseFallback(t *testing.T) {
	t.Parallel()
	sql := BotSQLCase("col_x")
	// Le fallback ELSE doit utiliser l'expression passée en argument.
	if !strings.Contains(sql, "ELSE col_x") {
		t.Errorf("BotSQLCase doit contenir 'ELSE col_x' (fallback), got: %q", sql)
	}
}

func TestBotSQLCase_DeterministicOrdering(t *testing.T) {
	t.Parallel()
	// Le code utilise sort.Strings → la sortie doit être déterministe.
	sql1 := BotSQLCase("xuid")
	sql2 := BotSQLCase("xuid")
	if sql1 != sql2 {
		t.Errorf("BotSQLCase non-deterministic: 2 appels successifs diffèrent")
	}
}

func TestBotSQLCase_CustomExpression(t *testing.T) {
	t.Parallel()
	// Doit utiliser l'expression custom dans le CASE et le ELSE.
	sql := BotSQLCase("COALESCE(t.xuid, '')")
	if !strings.Contains(sql, "CASE COALESCE(t.xuid, '')") {
		t.Errorf("BotSQLCase ne prend pas en compte l'expression custom dans CASE")
	}
	if !strings.Contains(sql, "ELSE COALESCE(t.xuid, '')") {
		t.Errorf("BotSQLCase ne prend pas en compte l'expression custom dans ELSE")
	}
}

func TestBotDisplayName(t *testing.T) {
	cases := []struct {
		xuid string
		want string
	}{
		{"bid(3.0)", "343 Ellis"},
		{"bid(18.0)", "343 Cream Corn"},
		{"bid(35.0)", "343 BF Scrub"},
		{"bid(0.0)", "343 Ritzy"},
		{"bid(3.0", "343 Ellis"},                   // paren manquante (API bug)
		{"bid(WallE-1)", "bid(WallE-1)"},           // hors map → retourne le xuid brut
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
