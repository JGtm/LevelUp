// Package analysis â€” home_canonical_test.go : tests paritÃ© entre les wrappers
// `*FromCanonical` et leurs versions legacy (P4.3b, ADR 0011).
//
// La conversion canonical â†’ HomeMatchRow est encapsulÃ©e dans le wrapper. Les
// tests vÃ©rifient que appeler le wrapper sur des canonical rows produit le
// mÃªme rÃ©sultat que appeler la version legacy directement sur les
// HomeMatchRow Ã©quivalents.
package analysis

import (
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// fixturePairForHome construit la mÃªme donnÃ©e match dans les 2 formats.
func fixturePairForHome(matchID string, kills, deaths, assists int, outcome int, isRanked, isWithFriends bool) (legacymatch.HomeMatchRow, canonical.PlayerMatchRow) {
	startTime := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	kda := float64(kills+assists) / float64(maxInt(deaths, 1))
	accuracy := 0.42
	timePlayed := 600
	perfScore := 75.5
	ratio := float64(kills) / float64(maxInt(deaths, 1))

	canonicalOutcome := canonical.OutcomeWin
	switch outcome {
	case domain.OutcomeLoss:
		canonicalOutcome = canonical.OutcomeLoss
	case domain.OutcomeDraw:
		canonicalOutcome = canonical.OutcomeTie
	case domain.OutcomeDNF:
		canonicalOutcome = canonical.OutcomeDNF
	}

	domainRow := legacymatch.HomeMatchRow{
		MatchID:          matchID,
		StartTime:        startTime,
		Outcome:          outcome,
		Kills:            kills,
		Deaths:           deaths,
		Assists:          assists,
		KDA:              &kda,
		Ratio:            &ratio,
		Accuracy:         &accuracy,
		TimePlayedSecs:   &timePlayed,
		PerformanceScore: &perfScore,
		IsRanked:         isRanked,
		IsFirefight:      false,
		IsWithFriends:    isWithFriends,
	}

	canonicalRow := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: startTime,
			IsRanked:     boolPtr(isRanked),
			IsPvE:        boolPtr(false),
			Outcome:      canonicalOutcome,
		},
		Self: canonical.MatchParticipant{
			Kills:      &kills,
			Deaths:     &deaths,
			Assists:    &assists,
			KDA:        &kda,
			Accuracy:   &accuracy,
			TimePlayed: &timePlayed,
			Outcome:    canonicalOutcome,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			IsWithFriends:    isWithFriends,
			PerformanceScore: &perfScore,
		},
	}
	return domainRow, canonicalRow
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestComputeKPIsFromCanonical_RendementAlignedAndDistinct reproduit le bug rapporté :
// deux joueurs au rendement clairement différent affichaient le MÊME pourcentage.
// Après fix (agrégat volume-pondéré + dégâts/frag-équivalent), leurs % diffèrent et
// chaque % = 225 / (dégâts par frag-équivalent affiché).
func TestComputeKPIsFromCanonical_RendementAlignedAndDistinct(t *testing.T) {
	t.Parallel()
	// Joueur A : moins efficace (plus de dégâts par frag-équivalent).
	playerA := []canonical.PlayerMatchRow{
		mkRowWithDamage(20, 6, 8, 8000, 4000),
		mkRowWithDamage(10, 4, 6, 4000, 3000),
	}
	// Joueur B : plus efficace (moins de dégâts par frag-équivalent).
	playerB := []canonical.PlayerMatchRow{
		mkRowWithDamage(25, 8, 7, 6000, 3500),
		mkRowWithDamage(12, 4, 5, 3000, 2500),
	}
	kpiA := ComputeKPIsFromCanonical(playerA, len(playerA), "fr", 225)
	kpiB := ComputeKPIsFromCanonical(playerB, len(playerB), "fr", 225)

	for name, k := range map[string]domain.HeroKPIs{"A": kpiA, "B": kpiB} {
		if k.AvgOffensiveConversion == nil || k.DmgPerKill == nil {
			t.Fatalf("joueur %s : AvgOC/DmgPerKill nil", name)
		}
		// Invariant d'alignement : % = 225 / dégâts par frag-équivalent (AvgOC est
		// arrondi 2 décimales à l'affichage → tolérance = un cran d'arrondi).
		if math.Abs(*k.AvgOffensiveConversion-225.0/(*k.DmgPerKill)) > 0.01 {
			t.Errorf("joueur %s : AvgOC (%v) != 225/DmgPerKill (%v)",
				name, *k.AvgOffensiveConversion, 225.0/(*k.DmgPerKill))
		}
	}
	// Le cœur du bug : les deux joueurs ne doivent PLUS avoir le même rendement,
	// et le plus efficace (B, dégâts/frag plus bas) doit avoir le % le plus haut.
	if *kpiA.AvgOffensiveConversion == *kpiB.AvgOffensiveConversion {
		t.Fatalf("régression : rendement identique (%v) pour deux joueurs distincts", *kpiA.AvgOffensiveConversion)
	}
	if *kpiB.DmgPerKill >= *kpiA.DmgPerKill {
		t.Fatalf("fixture invalide : B (%v) devrait avoir moins de dégâts/frag que A (%v)", *kpiB.DmgPerKill, *kpiA.DmgPerKill)
	}
	if *kpiB.AvgOffensiveConversion <= *kpiA.AvgOffensiveConversion {
		t.Errorf("le joueur plus efficace (B) devrait avoir le rendement le plus haut : A=%v B=%v",
			*kpiA.AvgOffensiveConversion, *kpiB.AvgOffensiveConversion)
	}
}

// TestComputeKPIsFromCanonical_OffensiveConversionDecoupledFromDamageTaken :
// régression H5. Avec DamageDealt présent mais DamageTaken == nil (Halo 5 n'a
// pas de damage_taken), AvgOffensiveConversion doit être calculé (> 0) tandis que
// AvgDefensiveResistance reste nil (pas de DR fabriquée).
func TestComputeKPIsFromCanonical_OffensiveConversionDecoupledFromDamageTaken(t *testing.T) {
	t.Parallel()
	kills, assists, deaths, dmgDealt := 10, 4, 6, 1500
	rows := []canonical.PlayerMatchRow{
		{
			Self: canonical.MatchParticipant{
				Outcome:     canonical.OutcomeWin,
				Kills:       &kills,
				Assists:     &assists,
				Deaths:      &deaths,
				DamageDealt: &dmgDealt,
				DamageTaken: nil, // Halo 5 : pas de damage_taken
			},
		},
	}
	got := ComputeKPIsFromCanonical(rows, len(rows), "fr", 225)
	if got.AvgOffensiveConversion == nil || *got.AvgOffensiveConversion <= 0 {
		t.Errorf("AvgOffensiveConversion: want non-nil > 0 (DamageDealt présent), got %v", got.AvgOffensiveConversion)
	}
	if got.AvgDefensiveResistance != nil {
		t.Errorf("AvgDefensiveResistance: want nil (pas de DamageTaken → pas de DR fabriquée), got %v", *got.AvgDefensiveResistance)
	}
}

// TestComputeKPIsFromCanonical_BothDamageFieldsSetYieldsBothKPIs : garde-fou Halo
// Infinite — DamageDealt ET DamageTaken présents → les deux KPIs sont calculés.
func TestComputeKPIsFromCanonical_BothDamageFieldsSetYieldsBothKPIs(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithDamage(10, 4, 6, 1500, 1800),
	}
	got := ComputeKPIsFromCanonical(rows, len(rows), "fr", 225)
	if got.AvgOffensiveConversion == nil || *got.AvgOffensiveConversion <= 0 {
		t.Errorf("AvgOffensiveConversion: want non-nil > 0, got %v", got.AvgOffensiveConversion)
	}
	if got.AvgDefensiveResistance == nil || *got.AvgDefensiveResistance <= 0 {
		t.Errorf("AvgDefensiveResistance: want non-nil > 0 (DamageTaken présent), got %v", got.AvgDefensiveResistance)
	}
}

// TestHomeMatchRowFromCanonical_RoundtripFields garantit que les champs clÃ©s
// (kills/deaths/outcome/ratio/KDA) survivent Ã  la conversion canonical â†’ HomeMatchRow.
func TestHomeMatchRowFromCanonical_RoundtripFields(t *testing.T) {
	_, canonicalRow := fixturePairForHome("m-1", 15, 6, 3, domain.OutcomeWin, true, true)
	out := HomeMatchRowFromCanonical(canonicalRow)

	if out.MatchID != "m-1" {
		t.Errorf("MatchID: got %q, want m-1", out.MatchID)
	}
	if out.Kills != 15 || out.Deaths != 6 || out.Assists != 3 {
		t.Errorf("K/D/A: got %d/%d/%d, want 15/6/3", out.Kills, out.Deaths, out.Assists)
	}
	if out.Outcome != domain.OutcomeWin {
		t.Errorf("Outcome: got %d, want %d", out.Outcome, domain.OutcomeWin)
	}
	if !out.IsRanked || out.IsFirefight || !out.IsWithFriends {
		t.Errorf("flags: ranked=%v firefight=%v friends=%v", out.IsRanked, out.IsFirefight, out.IsWithFriends)
	}
	if out.Ratio == nil || *out.Ratio != 15.0/6.0 {
		t.Errorf("Ratio: got %v, want 2.5", out.Ratio)
	}
}

// TestBuildSessionSummariesFromCanonical_NormalizesDominantMode garantit que
// le mode dominant est normalisé avant agrégation : "Arena:Slayer on Bazaar"
// et "Arena:Slayer on Live Fire" doivent fusionner sur "Slayer" plutôt que
// remonter le pair_name brut anglais. Régression historique : les sessions
// solo n'ont souvent pas de pair_name_fr → fallback EN brut affiché.
func TestBuildSessionSummariesFromCanonical_NormalizesDominantMode(t *testing.T) {
	sessionLabel := "session-1"
	startBase := time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC)
	mkRow := func(matchID, mapEN, pairNameEN string, offset time.Duration) canonical.PlayerMatchRow {
		t := startBase.Add(offset)
		return canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      matchID,
				StartedAtUTC: t,
				Outcome:      canonical.OutcomeWin,
				Map: &canonical.AssetReference{
					ID: mapEN, DefaultLabel: mapEN,
					Labels: map[string]string{"en": mapEN},
				},
				PairMode: &canonical.AssetReference{
					ID: pairNameEN, DefaultLabel: pairNameEN,
					Labels: map[string]string{"en": pairNameEN},
				},
			},
			Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
			Enrichment: canonical.PlayerMatchEnrichment{
				SessionLabel:  &sessionLabel,
				IsWithFriends: false,
			},
		}
	}

	rows := []canonical.PlayerMatchRow{
		mkRow("m1", "Bazaar", "Arena:Slayer on Bazaar", 0),
		mkRow("m2", "Live Fire", "Arena:Slayer on Live Fire", time.Minute),
		mkRow("m3", "Streets", "Arena:CTF on Streets", 2*time.Minute),
	}

	got := BuildSessionSummariesFromCanonical(rows, false, 5, "fr", 225)
	if len(got) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(got))
	}
	if got[0].DominantMode == nil {
		t.Fatalf("DominantMode = nil, want \"Slayer\"")
	}
	if *got[0].DominantMode != "Slayer" {
		t.Errorf("DominantMode = %q, want \"Slayer\" (les 2 maps Slayer doivent fusionner et être normalisées)", *got[0].DominantMode)
	}
}

// TestBuildSessionSummariesFromCanonical_PreservesPlaylistIdentity garantit
// que les playlists promues (Super Fiesta, Husky Raid) ne soient PAS extraites
// au sous-mode "Slayer"/"Assassin" mais préservent leur identité, même quand
// le pair_name brut contient suffixe map + " - Forge".
func TestBuildSessionSummariesFromCanonical_PreservesPlaylistIdentity(t *testing.T) {
	sessionLabel := "session-fiesta"
	startBase := time.Date(2026, 3, 31, 14, 20, 0, 0, time.UTC)
	mkRow := func(matchID, mapEN, pairNameEN string, offset time.Duration) canonical.PlayerMatchRow {
		t := startBase.Add(offset)
		return canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      matchID,
				StartedAtUTC: t,
				Outcome:      canonical.OutcomeWin,
				Map: &canonical.AssetReference{
					ID: mapEN, DefaultLabel: mapEN,
					Labels: map[string]string{"en": mapEN},
				},
				PairMode: &canonical.AssetReference{
					ID: pairNameEN, DefaultLabel: pairNameEN,
					Labels: map[string]string{"en": pairNameEN},
				},
			},
			Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
			Enrichment: canonical.PlayerMatchEnrichment{
				SessionLabel:  &sessionLabel,
				IsWithFriends: false,
			},
		}
	}

	rows := []canonical.PlayerMatchRow{
		mkRow("m1", "Behemoth", "Super Fiesta:Slayer on Behemoth - Forge", 0),
		mkRow("m2", "Streets", "Super Fiesta:CTF on Streets", 10*time.Minute),
	}

	got := BuildSessionSummariesFromCanonical(rows, false, 5, "fr", 225)
	if len(got) != 1 || got[0].DominantMode == nil {
		t.Fatalf("DominantMode missing: got=%+v", got)
	}
	if *got[0].DominantMode != "Super Fiesta" {
		t.Errorf("DominantMode = %q, want \"Super Fiesta\" (identité de playlist promue préservée, pas \"Slayer\")", *got[0].DominantMode)
	}
}

// TestInferHomeSkillHistoryFromCanonical_ParityWithLocal vÃ©rifie que la
// version canonical produit le mÃªme rÃ©sultat que la version locale legacy.
func TestInferHomeSkillHistoryFromCanonical_ParityWithLocal(t *testing.T) {
	dataset := []struct {
		isRanked, isPvE, isWithFriends bool
	}{
		{true, false, true},
		{false, false, false},
		{false, true, false}, // PvE doit Ãªtre exclu
		{true, false, false},
	}
	domainRows := make([]legacymatch.HomeMatchRow, 0, len(dataset))
	canonicalRows := make([]canonical.PlayerMatchRow, 0, len(dataset))
	for i, d := range dataset {
		dr := legacymatch.HomeMatchRow{
			MatchID:     "m-" + string(rune('0'+i)),
			IsRanked:    d.isRanked,
			IsFirefight: d.isPvE,
		}
		domainRows = append(domainRows, dr)
		canonicalRows = append(canonicalRows, canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:  "m-" + string(rune('0'+i)),
				IsRanked: boolPtr(d.isRanked),
				IsPvE:    boolPtr(d.isPvE),
			},
		})
	}

	gotR, gotU := InferHomeSkillHistoryFromCanonical(canonicalRows)
	// Replique la logique legacy de inferHomeSkillHistory.
	wantR, wantU := false, false
	for _, m := range domainRows {
		if m.IsFirefight {
			continue
		}
		if m.IsRanked {
			wantR = true
		} else {
			wantU = true
		}
	}

	if gotR != wantR || gotU != wantU {
		t.Errorf("InferHomeSkillHistoryFromCanonical = (%v,%v), want (%v,%v)", gotR, gotU, wantR, wantU)
	}
}
