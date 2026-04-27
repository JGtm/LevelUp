package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// helpers

func strPtr(s string) *string { return &s }

func mappingComposite(norm, children string) domain.CitationFullMapping {
	return domain.CitationFullMapping{
		NameNorm:          norm,
		MappingType:       "composite",
		CompositeChildren: strPtr(children),
	}
}

func mappingStat(norm string, tiers *string) domain.CitationFullMapping {
	m := domain.CitationFullMapping{
		NameNorm:    norm,
		MappingType: "stat",
		TierTargets: tiers,
	}
	return m
}

// ---------------------------------------------------------------------------
// computeCompositeCitations — via ComputeFullMatchCitations avec ctx vide
// ---------------------------------------------------------------------------

// buildMappings crée les mappings pour les tests composites.
func buildMappings(statMappings []domain.CitationFullMapping, compositeMappings []domain.CitationFullMapping) []domain.CitationFullMapping {
	out := make([]domain.CitationFullMapping, 0, len(statMappings)+len(compositeMappings))
	out = append(out, statMappings...)
	out = append(out, compositeMappings...)
	return out
}

// injectStats crée un CitationContext minimal avec des stats injectées.
func injectStats(stats map[string]float64) domain.CitationContext {
	return domain.CitationContext{Stats: stats, Medals: map[int64]int{}, Awards: map[string]int{}}
}

// TestComposite_AllChildrenFire vérifie qu'un composite vaut 2 si 2 enfants matchent.
func TestComposite_AllChildrenFire(t *testing.T) {
	// Enfants : wins_slayer (tier [1]), wins_ctf (tier [1]) — tous les deux >= 1 this match
	tiers1 := "1"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers1),
			mappingStat("wins_ctf", &tiers1),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer","wins_ctf"]`),
		},
	)

	// Inject stats so that wins_slayer et wins_ctf ont une valeur >= 1.
	// Le moteur lit Stats["wins_slayer"] via dispatchFull/stat.
	ctx := injectStats(map[string]float64{"wins_slayer": 1, "wins_ctf": 1})
	deltas := analysis.ComputeFullMatchCitations(ctx, mappings)

	found := false
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" {
			if d.Value != 2 {
				t.Errorf("combo_wins: attendu 2, obtenu %d", d.Value)
			}
			found = true
		}
	}
	if !found {
		t.Error("combo_wins absent des deltas")
	}
}

// TestComposite_OneChildMissing vérifie qu'un composite vaut 1 si un seul enfant matche.
func TestComposite_OneChildMissing(t *testing.T) {
	tiers := "1"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers),
			mappingStat("wins_ctf", &tiers),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer","wins_ctf"]`),
		},
	)
	ctx := injectStats(map[string]float64{"wins_slayer": 1, "wins_ctf": 0})
	deltas := analysis.ComputeFullMatchCitations(ctx, mappings)
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" && d.Value != 1 {
			t.Errorf("combo_wins: attendu 1, obtenu %d", d.Value)
		}
	}
}

// TestComposite_NoChildFires vérifie qu'un composite absent si aucun enfant ne matche.
func TestComposite_NoChildFires(t *testing.T) {
	tiers := "5"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer"]`),
		},
	)
	// wins_slayer = 2 < tiers(5) → not masterised
	ctx := injectStats(map[string]float64{"wins_slayer": 2})
	deltas := analysis.ComputeFullMatchCitations(ctx, mappings)
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" {
			t.Errorf("combo_wins ne doit pas apparaître si aucun enfant masterisé, obtenu %d", d.Value)
		}
	}
}

// TestComposite_NoTierTargets vérifie que sans tier_targets, masterisé dès val > 0.
func TestComposite_NoTierTargets(t *testing.T) {
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("kills_badge", nil), // pas de tier_targets
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_any", `["kills_badge"]`),
		},
	)
	ctx := injectStats(map[string]float64{"kills_badge": 3})
	deltas := analysis.ComputeFullMatchCitations(ctx, mappings)
	found := false
	for _, d := range deltas {
		if d.NameNorm == "combo_any" {
			found = true
			if d.Value != 1 {
				t.Errorf("combo_any: attendu 1, obtenu %d", d.Value)
			}
		}
	}
	if !found {
		t.Error("combo_any devrait apparaître (enfant sans tier_targets et val>0)")
	}
}

// ---------------------------------------------------------------------------
// scoreboard_extremes — MVP/LVP multi-colonnes
// ---------------------------------------------------------------------------

func mkScoreboardRow(xuid string, kills, deaths, assists int, kda *float64) domain.ScoreboardRaw {
	return domain.ScoreboardRaw{
		XUID:    xuid,
		Kills:   kills,
		Deaths:  deaths,
		Assists: assists,
		KDA:     kda,
	}
}

func f64(v float64) *float64 { return &v }

func TestMVPLVP_MultiColumn(t *testing.T) {
	// A : top kills, top assists → devrait être MVP (2+ best cells)
	// B : top deaths → seul worst → devrait être LVP (1 worst cell, pas assez)
	// C : middle everywhere
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A", Kills: 15, Deaths: 2, Assists: 8, KDA: f64(3.0)},
		{XUID: "B", Kills: 5, Deaths: 12, Assists: 1, KDA: f64(0.4)},
		{XUID: "C", Kills: 8, Deaths: 6, Assists: 4, KDA: f64(1.2)},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	if ext.MVPXUID != "A" {
		t.Errorf("attendu MVP=A (plus de kills+assists), obtenu %q", ext.MVPXUID)
	}
	// B a deaths élevé (worst cell pour deaths), KDA bas (worst cell pour kda) → LVP
	if ext.LVPXUID != "B" {
		t.Errorf("attendu LVP=B (deaths+KDA worst), obtenu %q", ext.LVPXUID)
	}
}

func TestMVPLVP_BotExcluded(t *testing.T) {
	// bid( préfixe = bot → exclu du calcul MVP/LVP
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "bid(12345)", Kills: 50, Deaths: 0, Assists: 30, KDA: f64(10.0)},
		{XUID: "human1", Kills: 10, Deaths: 5, Assists: 3, KDA: f64(1.5)},
		{XUID: "human2", Kills: 5, Deaths: 8, Assists: 1, KDA: f64(0.6)},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	if ext.MVPXUID == "bid(12345)" {
		t.Error("un bot ne doit pas être MVP")
	}
}

func TestMVPLVP_TooFewPlayers(t *testing.T) {
	// 1 seul joueur → pas de MVP/LVP
	ext := analysis.ComputeMVPLVP([]domain.ScoreboardRaw{
		{XUID: "solo", Kills: 10, Deaths: 5},
	})
	if ext.MVPXUID != "" || ext.LVPXUID != "" {
		t.Error("pas de MVP/LVP avec un seul joueur")
	}
}

func TestMVPLVP_InsufficientBestCells(t *testing.T) {
	// A n'a qu'une seule best cell (kills) → pas de MVP (besoin ≥ 2)
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A", Kills: 10, Deaths: 5, Assists: 3},
		{XUID: "B", Kills: 5, Deaths: 5, Assists: 5},
		{XUID: "C", Kills: 3, Deaths: 5, Assists: 8},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	// A a top kills (+1), C a top assists (+1), aucun n'a 2+ → MVP=""
	if ext.MVPXUID != "" {
		t.Errorf("attendu MVP vide (aucun joueur avec ≥2 best cells), obtenu %q", ext.MVPXUID)
	}
}
