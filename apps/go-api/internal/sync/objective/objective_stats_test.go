package objective

// objective_stats_test.go — tests golden purs de ExtractObjectiveStats sur les fixtures
// P0 (payloads réels anonymisés, PLAN_V72_OBJECTIVE_STATS ; Stockpile + Extraction :
// V721-02). Vérifie : mapping exact des champs natifs par bloc, conversion des durées
// ISO-8601 en secondes DOUBLE (fractions préservées), exclusivité des blocs par mode
// (colonnes des autres modes = nil), et qu'un joueur sans bloc objectif (Slayer) ne
// produit aucune row.

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/persist"
)

func loadObjectiveFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "objective_stats", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return m
}

func rowByXUID(rows []persist.ObjectiveStatsInsert, xuid string) *persist.ObjectiveStatsInsert {
	for i := range rows {
		if rows[i].XUID == xuid {
			return &rows[i]
		}
	}
	return nil
}

func wantInt(t *testing.T, got *int, want int, field string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %d", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %d, want %d", field, *got, want)
	}
}

func wantFloat(t *testing.T, got *float64, want float64, field string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %g", field, want)
		return
	}
	if math.Abs(*got-want) > 1e-9 {
		t.Errorf("%s = %g, want %g", field, *got, want)
	}
}

func wantNilInt(t *testing.T, got *int, field string) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %d, want nil (bloc d'un autre mode)", field, *got)
	}
}

func wantNilFloat(t *testing.T, got *float64, field string) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %g, want nil (bloc d'un autre mode)", field, *got)
	}
}

func TestExtractObjectiveStats_CTF(t *testing.T) {
	rows := ExtractObjectiveStats("m_ctf", loadObjectiveFixture(t, "ctf_match.json"))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row xuid 2500000000000001 absente")
	}
	if r.MatchID != "m_ctf" {
		t.Errorf("MatchID = %q, want m_ctf", r.MatchID)
	}
	wantInt(t, r.FlagCaptures, 1, "FlagCaptures")
	wantInt(t, r.FlagCaptureAssists, 0, "FlagCaptureAssists")
	wantInt(t, r.FlagGrabs, 9, "FlagGrabs")
	wantInt(t, r.FlagSecures, 7, "FlagSecures")
	wantInt(t, r.FlagSteals, 3, "FlagSteals")
	wantInt(t, r.FlagReturns, 2, "FlagReturns")
	wantInt(t, r.FlagCarriersKilled, 1, "FlagCarriersKilled")
	wantInt(t, r.FlagReturnersKilled, 0, "FlagReturnersKilled")
	wantInt(t, r.KillsAsFlagCarrier, 0, "KillsAsFlagCarrier")
	wantInt(t, r.KillsAsFlagReturner, 0, "KillsAsFlagReturner")
	wantFloat(t, r.TimeAsFlagCarrierSeconds, 140.0, "TimeAsFlagCarrierSeconds") // PT2M20S
	// Exclusivité : aucune colonne Zones/Oddball/Stockpile/Extraction.
	wantNilInt(t, r.ZoneCaptures, "ZoneCaptures")
	wantNilInt(t, r.SkullGrabs, "SkullGrabs")
	wantNilFloat(t, r.TimeInZonesSeconds, "TimeInZonesSeconds")
	wantNilInt(t, r.PowerSeedsDeposited, "PowerSeedsDeposited")
	wantNilInt(t, r.SuccessfulExtractions, "SuccessfulExtractions")
	wantNilFloat(t, r.TimeAsPowerSeedCarrierSeconds, "TimeAsPowerSeedCarrierSeconds")

	r2 := rowByXUID(rows, "2500000000000002")
	if r2 == nil {
		t.Fatal("row xuid 2500000000000002 absente")
	}
	wantInt(t, r2.FlagReturns, 3, "FlagReturns(p2)")
	wantInt(t, r2.FlagCaptures, 0, "FlagCaptures(p2)")
	wantFloat(t, r2.TimeAsFlagCarrierSeconds, 18.8, "TimeAsFlagCarrierSeconds(p2)") // PT18.8S
}

func TestExtractObjectiveStats_Strongholds(t *testing.T) {
	rows := ExtractObjectiveStats("m_sh", loadObjectiveFixture(t, "strongholds_match.json"))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row absente")
	}
	wantInt(t, r.ZoneCaptures, 9, "ZoneCaptures")
	wantInt(t, r.ZoneSecures, 1, "ZoneSecures")
	wantInt(t, r.ZoneOffensiveKills, 3, "ZoneOffensiveKills")
	wantInt(t, r.ZoneDefensiveKills, 1, "ZoneDefensiveKills")
	wantInt(t, r.ZoneScoringTicks, 0, "ZoneScoringTicks")
	wantFloat(t, r.TimeInZonesSeconds, 112.7, "TimeInZonesSeconds") // PT1M52.7S
	// Exclusivité : ni CTF ni Oddball.
	wantNilInt(t, r.FlagCaptures, "FlagCaptures")
	wantNilInt(t, r.SkullGrabs, "SkullGrabs")
}

func TestExtractObjectiveStats_KOTH(t *testing.T) {
	// KOTH partage ZonesStats avec Strongholds (confirmé P0) ; StrongholdScoringTicks>0
	// est le discriminant de la colline.
	rows := ExtractObjectiveStats("m_koth", loadObjectiveFixture(t, "koth_match.json"))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row absente")
	}
	wantInt(t, r.ZoneCaptures, 7, "ZoneCaptures")
	wantInt(t, r.ZoneScoringTicks, 89, "ZoneScoringTicks")
	wantFloat(t, r.TimeInZonesSeconds, 135.7, "TimeInZonesSeconds") // PT2M15.7S
}

func TestExtractObjectiveStats_Oddball(t *testing.T) {
	rows := ExtractObjectiveStats("m_odd", loadObjectiveFixture(t, "oddball_match.json"))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row absente")
	}
	wantInt(t, r.KillsAsSkullCarrier, 0, "KillsAsSkullCarrier")
	wantInt(t, r.SkullCarriersKilled, 3, "SkullCarriersKilled")
	wantInt(t, r.SkullGrabs, 3, "SkullGrabs")
	wantInt(t, r.SkullScoringTicks, 47, "SkullScoringTicks")
	wantFloat(t, r.TimeAsSkullCarrierSeconds, 49.1, "TimeAsSkullCarrierSeconds")               // PT49.1S
	wantFloat(t, r.LongestTimeAsSkullCarrierSeconds, 11.7, "LongestTimeAsSkullCarrierSeconds") // PT11.7S
	// Exclusivité : ni CTF ni Zones.
	wantNilInt(t, r.FlagCaptures, "FlagCaptures")
	wantNilInt(t, r.ZoneCaptures, "ZoneCaptures")
}

// TestExtractObjectiveStats_Stockpile — bloc StockpileStats (V721-02, capture P0 sur
// 3 matchs BTB:Stockpile / 79 blocs). Les 2 durées passent par objectiveDurationSeconds
// (fractions PRÉSERVÉES) : « PT1M9.6S » = 69.6 s, pas 69 — parsePTDuration tronquerait.
func TestExtractObjectiveStats_Stockpile(t *testing.T) {
	rows := ExtractObjectiveStats("m_stock", loadObjectiveFixture(t, "stockpile_match.json"))
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row xuid 2500000000000001 absente")
	}
	if r.MatchID != "m_stock" {
		t.Errorf("MatchID = %q, want m_stock", r.MatchID)
	}
	wantInt(t, r.KillsAsPowerSeedCarrier, 2, "KillsAsPowerSeedCarrier")
	wantInt(t, r.PowerSeedCarriersKilled, 1, "PowerSeedCarriersKilled")
	wantInt(t, r.PowerSeedsDeposited, 6, "PowerSeedsDeposited")
	wantInt(t, r.PowerSeedsStolen, 0, "PowerSeedsStolen")
	wantFloat(t, r.TimeAsPowerSeedCarrierSeconds, 59.1, "TimeAsPowerSeedCarrierSeconds") // PT59.1S
	wantFloat(t, r.TimeAsPowerSeedDriverSeconds, 0, "TimeAsPowerSeedDriverSeconds")      // PT0S
	// Exclusivité : aucune colonne CTF/Zones/Oddball/Extraction.
	wantNilInt(t, r.FlagCaptures, "FlagCaptures")
	wantNilInt(t, r.ZoneCaptures, "ZoneCaptures")
	wantNilInt(t, r.SkullGrabs, "SkullGrabs")
	wantNilInt(t, r.SuccessfulExtractions, "SuccessfulExtractions")
	wantNilFloat(t, r.TimeAsFlagCarrierSeconds, "TimeAsFlagCarrierSeconds")

	// 2e joueur : minutes + fraction sur les DEUX durées (69.6 / 64.2).
	r2 := rowByXUID(rows, "2500000000000002")
	if r2 == nil {
		t.Fatal("row xuid 2500000000000002 absente")
	}
	wantInt(t, r2.PowerSeedsDeposited, 2, "PowerSeedsDeposited(p2)")
	wantFloat(t, r2.TimeAsPowerSeedCarrierSeconds, 69.6, "TimeAsPowerSeedCarrierSeconds(p2)") // PT1M9.6S
	wantFloat(t, r2.TimeAsPowerSeedDriverSeconds, 64.2, "TimeAsPowerSeedDriverSeconds(p2)")   // PT1M4.2S

	// 3e joueur : seul porteur de PowerSeedsStolen > 0 (distingue « absent » de « zéro »).
	r3 := rowByXUID(rows, "2500000000000003")
	if r3 == nil {
		t.Fatal("row xuid 2500000000000003 absente")
	}
	wantInt(t, r3.PowerSeedsStolen, 2, "PowerSeedsStolen(p3)")
	wantInt(t, r3.PowerSeedsDeposited, 0, "PowerSeedsDeposited(p3)")
	wantFloat(t, r3.TimeAsPowerSeedCarrierSeconds, 3.8, "TimeAsPowerSeedCarrierSeconds(p3)") // PT3.8S
}

// TestExtractObjectiveStats_Extraction — bloc ExtractionStats (V721-02, capture P0 sur
// 2 matchs BTB:Extraction / 52 blocs). 5 compteurs entiers, AUCUNE durée : toutes les
// colonnes DOUBLE de la table doivent rester nil.
func TestExtractObjectiveStats_Extraction(t *testing.T) {
	rows := ExtractObjectiveStats("m_extr", loadObjectiveFixture(t, "extraction_match.json"))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row xuid 2500000000000001 absente")
	}
	if r.MatchID != "m_extr" {
		t.Errorf("MatchID = %q, want m_extr", r.MatchID)
	}
	wantInt(t, r.ExtractionConversionsCompleted, 1, "ExtractionConversionsCompleted")
	wantInt(t, r.ExtractionConversionsDenied, 1, "ExtractionConversionsDenied")
	wantInt(t, r.ExtractionInitiationsCompleted, 1, "ExtractionInitiationsCompleted")
	wantInt(t, r.ExtractionInitiationsDenied, 0, "ExtractionInitiationsDenied")
	wantInt(t, r.SuccessfulExtractions, 2, "SuccessfulExtractions")
	// Exclusivité : aucune colonne CTF/Zones/Oddball/Stockpile, aucune durée.
	wantNilInt(t, r.FlagCaptures, "FlagCaptures")
	wantNilInt(t, r.ZoneCaptures, "ZoneCaptures")
	wantNilInt(t, r.SkullGrabs, "SkullGrabs")
	wantNilInt(t, r.PowerSeedsDeposited, "PowerSeedsDeposited")
	wantNilFloat(t, r.TimeAsPowerSeedCarrierSeconds, "TimeAsPowerSeedCarrierSeconds")
	wantNilFloat(t, r.TimeAsFlagCarrierSeconds, "TimeAsFlagCarrierSeconds")
	wantNilFloat(t, r.TimeInZonesSeconds, "TimeInZonesSeconds")

	// 2e joueur : seul porteur d'ExtractionInitiationsDenied > 0.
	r2 := rowByXUID(rows, "2500000000000002")
	if r2 == nil {
		t.Fatal("row xuid 2500000000000002 absente")
	}
	wantInt(t, r2.ExtractionInitiationsDenied, 1, "ExtractionInitiationsDenied(p2)")
	wantInt(t, r2.SuccessfulExtractions, 0, "SuccessfulExtractions(p2)")
}

// TestExtractObjectiveStats_Vip — bloc VipStats (V721-02, capture P0 sur le match
// Arena:VIP 00761d27-… : 8 blocs joueur). Le bloc existe AUSSI sous Teams[].Stats
// (agrégat équipe) : seul le niveau joueur est extrait, la fixture ne porte donc pas
// de section Teams. VipStats était absent de StatsBundle avant ce chantier.
func TestExtractObjectiveStats_Vip(t *testing.T) {
	rows := ExtractObjectiveStats("m_vip", loadObjectiveFixture(t, "vip_match.json"))
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row xuid 2500000000000001 absente")
	}
	if r.MatchID != "m_vip" {
		t.Errorf("MatchID = %q, want m_vip", r.MatchID)
	}
	wantInt(t, r.KillsAsVip, 2, "KillsAsVip")
	wantInt(t, r.VipKills, 1, "VipKills")
	wantInt(t, r.VipAssists, 0, "VipAssists")
	wantInt(t, r.TimesSelectedAsVip, 4, "TimesSelectedAsVip")
	wantInt(t, r.MaxKillingSpreeAsVip, 1, "MaxKillingSpreeAsVip")
	// Durées ISO : PT1M49.5S = 109.5 s (fraction préservée), PT48S = 48 s.
	wantFloat(t, r.TimeAsVipSeconds, 109.5, "TimeAsVipSeconds")
	wantFloat(t, r.LongestTimeAsVipSeconds, 48, "LongestTimeAsVipSeconds")
	// Exclusivité : aucune colonne CTF/Zones/Oddball/Stockpile/Extraction.
	wantNilInt(t, r.FlagCaptures, "FlagCaptures")
	wantNilInt(t, r.ZoneCaptures, "ZoneCaptures")
	wantNilInt(t, r.SkullGrabs, "SkullGrabs")
	wantNilInt(t, r.PowerSeedsDeposited, "PowerSeedsDeposited")
	wantNilInt(t, r.SuccessfulExtractions, "SuccessfulExtractions")
	wantNilFloat(t, r.TimeAsSkullCarrierSeconds, "TimeAsSkullCarrierSeconds")

	// 2e joueur : minutes ENTIÈRES sans fraction (PT3M25S = 205 s), possession la plus longue.
	r2 := rowByXUID(rows, "2500000000000002")
	if r2 == nil {
		t.Fatal("row xuid 2500000000000002 absente")
	}
	wantInt(t, r2.MaxKillingSpreeAsVip, 5, "MaxKillingSpreeAsVip(p2)")
	wantInt(t, r2.VipKills, 0, "VipKills(p2)")
	wantFloat(t, r2.TimeAsVipSeconds, 205, "TimeAsVipSeconds(p2)")               // PT3M25S
	wantFloat(t, r2.LongestTimeAsVipSeconds, 205, "LongestTimeAsVipSeconds(p2)") // PT3M25S

	// 3e joueur : chasseur de VIP (6 VIP abattus, 0 frag EN TANT QUE VIP) — un compteur
	// à zéro doit rester non-nil (distinct d'un champ absent).
	r3 := rowByXUID(rows, "2500000000000003")
	if r3 == nil {
		t.Fatal("row xuid 2500000000000003 absente")
	}
	wantInt(t, r3.VipKills, 6, "VipKills(p3)")
	wantInt(t, r3.KillsAsVip, 0, "KillsAsVip(p3)")
	wantInt(t, r3.MaxKillingSpreeAsVip, 0, "MaxKillingSpreeAsVip(p3)")
	wantFloat(t, r3.TimeAsVipSeconds, 31.9, "TimeAsVipSeconds(p3)") // PT31.9S
}

func TestExtractObjectiveStats_SlayerProducesNoRow(t *testing.T) {
	// Un joueur dont le Stats bundle n'a que CoreStats (Slayer) ne produit aucune row.
	payload := map[string]any{
		"Players": []any{
			map[string]any{
				"PlayerId":        "xuid(2500000000000009)",
				"PlayerType":      float64(1),
				"PlayerTeamStats": []any{map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{"Kills": float64(10)}}}},
			},
		},
	}
	rows := ExtractObjectiveStats("m_slayer", payload)
	if len(rows) != 0 {
		t.Fatalf("Slayer : got %d rows, want 0", len(rows))
	}
}

// TestExtractObjectiveStats_MultiTeamStatsAndBots couvre deux comportements
// jusqu'ici non assertés (contre-revue V7.2) :
//
//  1. PlayerTeamStats MULTI-ENTRÉES — findStatsBundle retient le PREMIER bloc
//     Stats trouvé (et non l'index 0 quand celui-ci n'en porte pas). Sans
//     assertion, un refactor pourrait basculer sur la dernière entrée / fusionner
//     les blocs sans qu'aucun test ne bronche.
//  2. Clé BOT — un bot doit être keyé EXACTEMENT comme dans match_participants
//     (« bid(3.0) »), y compris quand l'API renvoie la forme tronquée « bid(4.0 ».
//     L'ancien cleanXUID produisait « bid(3.0 » → lignes définitivement orphelines
//     dans match_objective_stats.
func TestExtractObjectiveStats_MultiTeamStatsAndBots(t *testing.T) {
	rows := ExtractObjectiveStats("m_mt", loadObjectiveFixture(t, "multi_teamstats_match.json"))
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4 (2 humains + 2 bots)", len(rows))
	}

	// 1a — 2 entrées porteuses : la PREMIÈRE gagne (4/6, pas 99/99).
	r := rowByXUID(rows, "2500000000000001")
	if r == nil {
		t.Fatal("row xuid 2500000000000001 absente")
	}
	wantInt(t, r.FlagCaptures, 4, "FlagCaptures (1re entrée PlayerTeamStats)")
	wantInt(t, r.FlagGrabs, 6, "FlagGrabs (1re entrée PlayerTeamStats)")
	wantFloat(t, r.TimeAsFlagCarrierSeconds, 70.0, "TimeAsFlagCarrierSeconds (1re entrée)") // PT1M10S

	// 1b — 1re entrée SANS bloc Stats : on retient le premier bloc TROUVÉ.
	r2 := rowByXUID(rows, "2500000000000002")
	if r2 == nil {
		t.Fatal("row xuid 2500000000000002 absente (1re entrée sans Stats : le 1er bloc trouvé doit être retenu)")
	}
	wantInt(t, r2.FlagCaptures, 7, "FlagCaptures (1er bloc Stats trouvé)")

	// 2 — bots keyés comme dans match_participants (parenthèse fermante incluse).
	bot := rowByXUID(rows, "bid(3.0)")
	if bot == nil {
		t.Fatalf("row bot « bid(3.0) » absente — clés présentes : %v", rowXUIDs(rows))
	}
	wantInt(t, bot.FlagCaptures, 1, "FlagCaptures (bot)")
	if rowByXUID(rows, "bid(3.0") != nil {
		t.Error("clé bot tronquée « bid(3.0 » présente : régression de cleanXUID (ligne orpheline garantie)")
	}
	if rowByXUID(rows, "bid(4.0)") == nil {
		t.Errorf("forme API tronquée « bid(4.0 » non normalisée en « bid(4.0) » — clés : %v", rowXUIDs(rows))
	}
}

// rowXUIDs : clés produites, pour des messages d'échec exploitables.
func rowXUIDs(rows []persist.ObjectiveStatsInsert) []string {
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].XUID)
	}
	return out
}

// TestCleanXUID_ParityWithSyncExtractXUID verrouille la parité de cleanXUID avec
// sync.extractXUID (transforms_helpers.go), qui produit les clés de
// match_participants. Copie locale assumée (cycle d'import) : sans ce test, les
// deux implémentations peuvent diverger silencieusement — c'est exactement ce qui
// s'est produit pour les bots.
func TestCleanXUID_ParityWithSyncExtractXUID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"xuid(2500000000000001)", "2500000000000001"},
		{"bid(3.0)", "bid(3.0)"}, // bot : forme intégrale conservée
		{"bid(3.0", "bid(3.0)"},  // bot : parenthèse fermante restaurée
		{"bid(12.0)", "bid(12.0)"},
		{"", ""},
		{"2500000000000001", ""}, // forme nue non reconnue → sauté (pas de ligne orpheline)
		{"garbage", ""},
	}
	for _, c := range cases {
		if got := cleanXUID(c.in); got != c.want {
			t.Errorf("cleanXUID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestObjectiveDurationSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want *float64
	}{
		{"PT0S", ptrF(0)},
		{"PT18.8S", ptrF(18.8)},
		{"PT2M20S", ptrF(140)},
		{"PT1M52.7S", ptrF(112.7)},
		{"PT1H2M3.5S", ptrF(3723.5)},
		{"", nil},
		{"garbage", nil},
	}
	for _, c := range cases {
		got := objectiveDurationSeconds(c.in)
		switch {
		case c.want == nil && got != nil:
			t.Errorf("%q → %g, want nil", c.in, *got)
		case c.want != nil && got == nil:
			t.Errorf("%q → nil, want %g", c.in, *c.want)
		case c.want != nil && got != nil && math.Abs(*got-*c.want) > 1e-9:
			t.Errorf("%q → %g, want %g", c.in, *got, *c.want)
		}
	}
}

func ptrF(v float64) *float64 { return &v }
