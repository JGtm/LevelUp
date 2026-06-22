package halo_5

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// TestApplySpartanRank_MidRank : un SR intermédiaire (JGtm SR111, 3 908 120 XP)
// pose rang/label/XP courant/XP avant suivant/rang suivant cohérents avec la table.
func TestApplySpartanRank_MidRank(t *testing.T) {
	snap := &canonical.CareerSnapshot{}
	applySpartanRank(snap, 111, 3908120)

	if snap.RankNumber != 111 {
		t.Errorf("RankNumber = %d, want 111", snap.RankNumber)
	}
	if snap.CurrentRank == nil || snap.CurrentRank.DefaultLabel != "SR 111" {
		t.Errorf("CurrentRank = %+v, want label 'SR 111'", snap.CurrentRank)
	}
	if snap.IsMaxRank {
		t.Error("SR111 ne doit pas être max")
	}
	if snap.NextRank == nil || snap.NextRank.DefaultLabel != "SR 112" {
		t.Errorf("NextRank = %+v, want label 'SR 112'", snap.NextRank)
	}
	if snap.XPTotal == nil || *snap.XPTotal != 3908120 {
		t.Errorf("XPTotal = %v, want 3908120", snap.XPTotal)
	}
	// XP dans le rang + XP pour compléter le rang : dérivés de la table (auto-cohérent).
	wantCur := 3908120 - h5SRStartXP[110]
	if snap.CurrentXP == nil || *snap.CurrentXP != wantCur {
		t.Errorf("CurrentXP = %v, want %d", snap.CurrentXP, wantCur)
	}
	wantNeed := h5SRStartXP[111] - h5SRStartXP[110]
	if snap.XPForNextRank == nil || *snap.XPForNextRank != wantNeed {
		t.Errorf("XPForNextRank = %v, want %d", snap.XPForNextRank, wantNeed)
	}
}

// TestApplySpartanRank_Max : SR152 = MAX (IsMaxRank, aucun rang suivant ni « XP
// avant le suivant ») — surtout PAS un seuil « 0 ».
func TestApplySpartanRank_Max(t *testing.T) {
	snap := &canonical.CareerSnapshot{}
	applySpartanRank(snap, 152, 51000000)

	if !snap.IsMaxRank {
		t.Error("SR152 doit être IsMaxRank")
	}
	if snap.NextRank != nil {
		t.Errorf("aucun rang après SR152, got %+v", snap.NextRank)
	}
	if snap.XPForNextRank != nil {
		t.Errorf("aucun « XP avant suivant » au max, got %v", *snap.XPForNextRank)
	}
	if snap.CurrentRank == nil || snap.CurrentRank.DefaultLabel != "SR 152" {
		t.Errorf("CurrentRank = %+v, want 'SR 152'", snap.CurrentRank)
	}
	if snap.RankNumber != 152 {
		t.Errorf("RankNumber = %d, want 152", snap.RankNumber)
	}
}

// TestApplySpartanRank_OutOfBounds : un SR hors [1..152] laisse le snapshot intact.
func TestApplySpartanRank_OutOfBounds(t *testing.T) {
	for _, sr := range []int{0, -1, 153, 9999} {
		snap := &canonical.CareerSnapshot{}
		applySpartanRank(snap, sr, 100)
		if snap.RankNumber != 0 || snap.CurrentRank != nil {
			t.Errorf("SR=%d devrait être ignoré, got RankNumber=%d CurrentRank=%+v", sr, snap.RankNumber, snap.CurrentRank)
		}
	}
}

// TestLoadCareerSnapshot_AlwaysRankMax152 (AXE C1) : MÊME quand l'enrichissement SR
// live échoue (aucun match récent → enrichSpartanRank no-op), un CareerSnapshot h5
// avec stats arena DOIT toujours porter RankMax=152 (+ XPMax). C'est le filet
// déterministe qui évite que le front retombe sur le fallback HINF « X/272 ».
func TestLoadCareerSnapshot_AlwaysRankMax152(t *testing.T) {
	src := &fakeSource{
		sr:      mustServiceRecord(t),
		matches: &H5MatchesResponse{Results: nil}, // pas de match → SR non enrichi
	}
	a := NewDataAdapter(srcFactory(src), nil)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot nil")
	}
	if snap.RankMax == nil || *snap.RankMax != h5MaxSpartanRank {
		t.Errorf("RankMax = %v, want %d (filet déterministe même sans SR enrichi)", snap.RankMax, h5MaxSpartanRank)
	}
	if snap.XPMax == nil || *snap.XPMax != h5SRStartXP[h5MaxSpartanRank-1] {
		t.Errorf("XPMax = %v, want %d (XP cumulé au SR152)", snap.XPMax, h5SRStartXP[h5MaxSpartanRank-1])
	}
	// SR réel inconnu (pas d'enrichissement) → RankNumber 0, pas de CurrentRank SR
	// inventé. Le CSR (palier Diamant du service record) reste intact.
	if snap.RankNumber != 0 {
		t.Errorf("RankNumber = %d, want 0 (SR non enrichi, pas de valeur inventée)", snap.RankNumber)
	}
	if snap.RankTier == nil || *snap.RankTier != "Diamant" {
		t.Errorf("RankTier = %v, want Diamant (CSR du service record préservé)", snap.RankTier)
	}
}

// TestApplyDefaultSpartanRankBounds_NoOverwrite : les bornes par défaut n'écrasent
// JAMAIS un SR déjà enrichi (idempotent). applySpartanRank pose RankMax/XPMax via
// la table ; un second passage des défauts laisse ces valeurs intactes.
func TestApplyDefaultSpartanRankBounds_NoOverwrite(t *testing.T) {
	snap := &canonical.CareerSnapshot{}
	applySpartanRank(snap, 111, 3908120) // pose RankMax=152, XPMax=table
	wantRankMax := *snap.RankMax
	wantXPMax := *snap.XPMax
	applyDefaultSpartanRankBounds(snap) // doit être no-op
	if *snap.RankMax != wantRankMax || *snap.XPMax != wantXPMax {
		t.Errorf("défauts ont écrasé les bornes enrichies : RankMax=%d XPMax=%d", *snap.RankMax, *snap.XPMax)
	}
	if snap.RankNumber != 111 {
		t.Errorf("RankNumber = %d, want 111 (inchangé)", snap.RankNumber)
	}
}

// TestLoadCareerSnapshot_EnrichesSpartanRank : chemin complet — le service record
// pose le CSR, et la carnage du dernier match (XpInfo) ajoute le rang SR.
func TestLoadCareerSnapshot_EnrichesSpartanRank(t *testing.T) {
	src := &fakeSource{
		sr: mustServiceRecord(t),
		matches: &H5MatchesResponse{Results: []H5MatchResult{
			{Id: H5MatchID{MatchId: "m1", GameMode: 1}},
		}},
		carnage: &H5CarnageResponse{PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "OtherGuy"}, XpInfo: &H5XpInfo{SpartanRank: 50, TotalXP: 900000}},
			{Player: H5PlayerRef{Gamertag: "JGtm"}, XpInfo: &H5XpInfo{SpartanRank: 111, TotalXP: 3908120}},
		}},
	}
	a := NewDataAdapter(srcFactory(src), nil)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot nil")
	}
	if snap.RankNumber != 111 {
		t.Errorf("SR non enrichi: RankNumber = %d, want 111 (le SR du joueur JGtm, pas d'un autre roster)", snap.RankNumber)
	}
	if snap.CurrentRank == nil || snap.CurrentRank.DefaultLabel != "SR 111" {
		t.Errorf("CurrentRank = %+v, want 'SR 111'", snap.CurrentRank)
	}
}
