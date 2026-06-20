package duckdb

// patterns_repo_test.go — caractérisation de la logique pure déplacée depuis
// le handler (refactor Axe 1). Verrouille le comportement de merge + deltas :
// un refactor de découplage ne doit RIEN changer à ces sorties.

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/patterns"
)

func ptrF(v float64) *float64 { return &v }

// TestMergePatternRows_FieldsAndRatingRouting vérifie KDA, HSRate, le mapping
// des enrichissements et le routage rating_type → DeltaLUSR / CSRValue.
func TestMergePatternRows_FieldsAndRatingRouting(t *testing.T) {
	shared := []patternSharedRow{
		{
			MatchID: "m1", Mode: "Slayer", MapID: "map1", Outcome: 2,
			DurationSec: 600, Kills: 10, Deaths: 5, Assists: 4,
			Accuracy: 0.55, DamageDlt: 2000, DamageTkn: 1500, HeadshotKills: 3,
			IsRanked: true,
		},
		{
			MatchID: "m2", Mode: "Oddball", MapID: "map2", Outcome: 3,
			DurationSec: 500, Kills: 0, Deaths: 0, Assists: 0,
			DamageDlt: 0, DamageTkn: 0, HeadshotKills: 0,
		},
	}
	enrichMap := map[string]patternEnrichmentRow{
		"m1": {PerfScore: ptrF(0.8), SessionID: "s1", IsWithFriends: true, EngageScore: ptrF(0.6), ResidualBrut: ptrF(0.1)},
	}
	skillMap := map[string]patternSkillRankRow{
		"m1": {RatingValue: ptrF(1500), RatingType: "LUSR"},
		"m2": {RatingValue: ptrF(1200), RatingType: "CSR"},
	}

	out := mergePatternRows(shared, enrichMap, skillMap, 225)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}

	m1 := out[0]
	// KDA = (10 + 4/2) / 5 = 2.4
	if m1.KDA != 2.4 {
		t.Errorf("m1.KDA = %v, want 2.4", m1.KDA)
	}
	// HSRate = 3/10 = 0.3
	if m1.HSRate != 0.3 {
		t.Errorf("m1.HSRate = %v, want 0.3", m1.HSRate)
	}
	if m1.OC <= 0 {
		t.Errorf("m1.OC = %v, want > 0 (damage_dealt > 0)", m1.OC)
	}
	// Enrichissements mappés
	if m1.SessionID != "s1" || !m1.IsWithFriends || m1.PerfScore == nil || *m1.PerfScore != 0.8 {
		t.Errorf("m1 enrichments mal mappés: %+v", m1)
	}
	// rating_type LUSR → DeltaLUSR (valeur brute à ce stade), pas CSRValue
	if m1.DeltaLUSR == nil || *m1.DeltaLUSR != 1500 {
		t.Errorf("m1.DeltaLUSR = %v, want 1500", m1.DeltaLUSR)
	}
	if m1.CSRValue != nil {
		t.Errorf("m1.CSRValue = %v, want nil (rating_type=LUSR)", m1.CSRValue)
	}

	m2 := out[1]
	// Deaths=0 → denom forcé à 1 → KDA = 0
	if m2.KDA != 0 {
		t.Errorf("m2.KDA = %v, want 0", m2.KDA)
	}
	// Kills=0 → HSRate reste 0 (pas de division par zéro)
	if m2.HSRate != 0 {
		t.Errorf("m2.HSRate = %v, want 0", m2.HSRate)
	}
	// rating_type CSR → CSRValue, pas DeltaLUSR
	if m2.CSRValue == nil || *m2.CSRValue != 1200 {
		t.Errorf("m2.CSRValue = %v, want 1200", m2.CSRValue)
	}
	if m2.DeltaLUSR != nil {
		t.Errorf("m2.DeltaLUSR = %v, want nil (rating_type=CSR)", m2.DeltaLUSR)
	}
	// Pas d'enrichissement pour m2 → champs neutres
	if m2.SessionID != "" || m2.PerfScore != nil {
		t.Errorf("m2 ne devrait pas avoir d'enrichissements: %+v", m2)
	}
}

// TestComputePatternSkillDeltas_TwoMatches verrouille : delta = récent - ancien,
// premier match (le plus ancien) à nil, ordre DESC restauré en sortie.
func TestComputePatternSkillDeltas_TwoMatches(t *testing.T) {
	older := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)

	// Entrée DESC (plus récent en premier), comme le SQL ORDER BY played_at DESC.
	rows := []patterns.MatchRow{
		{MatchID: "newer", PlayedAt: newer, DeltaLUSR: ptrF(1550), CSRValue: ptrF(1250)},
		{MatchID: "older", PlayedAt: older, DeltaLUSR: ptrF(1500), CSRValue: ptrF(1200)},
	}
	computePatternSkillDeltas(rows)

	// Sortie DESC : index 0 = newer, index 1 = older.
	if rows[0].MatchID != "newer" || rows[1].MatchID != "older" {
		t.Fatalf("ordre DESC non restauré: [%s, %s]", rows[0].MatchID, rows[1].MatchID)
	}
	// newer : delta LUSR = 1550 - 1500 = 50 ; delta CSR = 1250 - 1200 = 50
	if rows[0].DeltaLUSR == nil || *rows[0].DeltaLUSR != 50 {
		t.Errorf("newer.DeltaLUSR = %v, want 50", rows[0].DeltaLUSR)
	}
	if rows[0].DeltaCSR == nil || *rows[0].DeltaCSR != 50 {
		t.Errorf("newer.DeltaCSR = %v, want 50", rows[0].DeltaCSR)
	}
	// older (le plus ancien) : pas de delta
	if rows[1].DeltaLUSR != nil {
		t.Errorf("older.DeltaLUSR = %v, want nil", rows[1].DeltaLUSR)
	}
	if rows[1].DeltaCSR != nil {
		t.Errorf("older.DeltaCSR = %v, want nil", rows[1].DeltaCSR)
	}
}

// TestComputePatternSkillDeltas_SingleRow : moins de 2 rows → no-op total
// (retour anticipé sur len < 2). La valeur brute posée par le merge reste
// telle quelle — elle n'est PAS remise à nil. Comportement historique verrouillé.
func TestComputePatternSkillDeltas_SingleRow(t *testing.T) {
	rows := []patterns.MatchRow{
		{MatchID: "solo", PlayedAt: time.Now(), DeltaLUSR: ptrF(1500)},
	}
	computePatternSkillDeltas(rows)
	if rows[0].DeltaLUSR == nil || *rows[0].DeltaLUSR != 1500 {
		t.Errorf("single row DeltaLUSR = %v, want 1500 inchangé (no-op)", rows[0].DeltaLUSR)
	}
}
