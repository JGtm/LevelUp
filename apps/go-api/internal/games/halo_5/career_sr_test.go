package halo_5

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
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

// TestBuildSpartanRankCatalog : le catalog title-aware Halo 5 résout « SR N » pour
// tous les niveaux (label EN + FR), porte le seuil XP de chaque rang, et marque
// SR152 comme max (pas de rang suivant). C'est le mécanisme qui remplace le fallback
// HINF « Rang N » sur la Home, sans aucune écriture DB.
func TestBuildSpartanRankCatalog(t *testing.T) {
	cat := BuildSpartanRankCatalog()

	if cat.Len() != h5MaxSpartanRank {
		t.Fatalf("Len = %d, want %d", cat.Len(), h5MaxSpartanRank)
	}

	// Cas cible Home : SR 147 → label « SR 147 » (FR et EN), is_max dérivé false.
	if label, ok := cat.FullLabel(147, "fr"); !ok || label != "SR 147" {
		t.Errorf("FullLabel(147,'fr') = %q,%v, want 'SR 147',true", label, ok)
	}
	if label, ok := cat.FullLabel(147, "en"); !ok || label != "SR 147" {
		t.Errorf("FullLabel(147,'en') = %q,%v, want 'SR 147',true", label, ok)
	}
	if _, ok := cat.Next(147); !ok {
		t.Error("Next(147) doit exister (SR147 n'est pas max)")
	}

	// XPRequired(111) = delta de la table (XP pour compléter le rang 111).
	wantXP := h5SRStartXP[111] - h5SRStartXP[110]
	if e, ok := cat.Get(111); !ok || e.XPRequired != wantXP {
		t.Errorf("Get(111).XPRequired = %d (ok=%v), want %d", e.XPRequired, ok, wantXP)
	}

	// SR152 = max : présent, mais sans rang suivant et sans XPRequired (pas de palier
	// à compléter au sommet) — buildHomeCareerRank en dérive is_max=true.
	if _, ok := cat.Get(h5MaxSpartanRank); !ok {
		t.Errorf("Get(%d) absent", h5MaxSpartanRank)
	}
	if _, ok := cat.Next(h5MaxSpartanRank); ok {
		t.Errorf("Next(%d) doit être absent (rang max)", h5MaxSpartanRank)
	}
	if e, _ := cat.Get(h5MaxSpartanRank); e.XPRequired != 0 {
		t.Errorf("Get(%d).XPRequired = %d, want 0 (max, pas de rang suivant)", h5MaxSpartanRank, e.XPRequired)
	}
}

// fakeCareerLocal implémente CareerLocalSource (substrat DuckDB synchronisé) sans DB.
type fakeCareerLocal struct {
	data    *domain.H5CareerLocal
	err     error
	history []domain.XPHistoryPoint
	histErr error
}

func (f fakeCareerLocal) GetLatestCareer(context.Context) (*domain.H5CareerLocal, error) {
	return f.data, f.err
}

func (f fakeCareerLocal) GetXPHistory(context.Context) ([]domain.XPHistoryPoint, error) {
	return f.history, f.histErr
}

// TestLoadCareerSnapshot_LiveFails_FallbackLocalSR : quand le live échoue (token du
// joueur mort — RT révoqué), LoadCareerSnapshot sert le rang SR + l'XP PERSISTÉS depuis
// la source locale. Résilience clé : la carrière d'un joueur SUIVI ne disparaît pas
// parce que SON refresh_token est mort (le rang/XP ne dépend pas du token du joueur).
func TestLoadCareerSnapshot_LiveFails_FallbackLocalSR(t *testing.T) {
	live := &fakeSource{srErr: errors.New("token mort (RT révoqué)")} // voie live KO
	local := fakeCareerLocal{data: &domain.H5CareerLocal{
		HasCSR: true, CSRTier: "Diamond", CSRSubTier: 5,
		SpartanRank: 111, TotalXP: 3908120,
	}}
	a := NewDataAdapter(srcFactory(live), nil).WithCareerSource(local)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot nil — le fallback local aurait dû servir le rang persisté")
	}
	if snap.RankNumber != 111 {
		t.Errorf("RankNumber (SR) = %d, want 111 (servi depuis le local après échec live)", snap.RankNumber)
	}
	if snap.XPTotal == nil || *snap.XPTotal != 3908120 {
		t.Errorf("XPTotal = %v, want 3908120 (TotalXP local)", snap.XPTotal)
	}
	if snap.RankTier == nil || *snap.RankTier == "" {
		t.Error("RankTier (CSR) devrait être posé depuis le local")
	}
}

// TestLoadCareerSnapshot_LiveOK_PrefersLive : quand le live répond, on le sert (frais)
// SANS toucher au local — pas de régression de fraîcheur pour un joueur au token sain.
func TestLoadCareerSnapshot_LiveOK_PrefersLive(t *testing.T) {
	live := &fakeSource{
		sr:      mustServiceRecord(t),
		matches: &H5MatchesResponse{Results: nil}, // pas de match → SR live non enrichi
	}
	// Local renvoie un SR ARBITRAIRE différent : il NE doit PAS être servi si le live marche.
	local := fakeCareerLocal{data: &domain.H5CareerLocal{SpartanRank: 1, TotalXP: 1}}
	a := NewDataAdapter(srcFactory(live), nil).WithCareerSource(local)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap.RankNumber == 1 {
		t.Error("le local (SR1) a été servi alors que le live répondait — live-first violé")
	}
}

// TestLoadCareerSnapshot_AttachesXPHistory : signalement #8 (Historique XP). Avec
// opts.IncludeHistory=true, snap.History est peuplé depuis careerLocal.GetXPHistory
// (parité Infinite) ; sans l'option, l'historique n'est PAS chargé.
func TestLoadCareerSnapshot_AttachesXPHistory(t *testing.T) {
	live := &fakeSource{srErr: errors.New("token mort")} // force le fallback local
	hist := []domain.XPHistoryPoint{
		{Rank: 100, CurrentXP: 5000, XPTotal: 1000000},
		{Rank: 101, CurrentXP: 6000, XPTotal: 1006000},
	}
	local := fakeCareerLocal{
		data:    &domain.H5CareerLocal{SpartanRank: 101, TotalXP: 1006000},
		history: hist,
	}
	a := NewDataAdapter(srcFactory(live), nil).WithCareerSource(local)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{IncludeHistory: true})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if len(snap.History) != 2 {
		t.Fatalf("History len = %d, want 2 (peuplé depuis GetXPHistory)", len(snap.History))
	}
	if snap.History[1].RankNumber != 101 || snap.History[1].XPTotal == nil || *snap.History[1].XPTotal != 1006000 {
		t.Errorf("History[1] mal projeté: %+v", snap.History[1])
	}

	snap2, _ := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if len(snap2.History) != 0 {
		t.Errorf("History sans IncludeHistory = %d, want 0 (pas de chargement)", len(snap2.History))
	}
}

// TestLoadCareerSnapshot_XPHistoryError_Graceful : une erreur de lecture de l'historique
// XP ne casse PAS le snapshot (dégradation propre — le graphe se masque, la carrière reste).
func TestLoadCareerSnapshot_XPHistoryError_Graceful(t *testing.T) {
	live := &fakeSource{srErr: errors.New("token mort")}
	local := fakeCareerLocal{
		data:    &domain.H5CareerLocal{SpartanRank: 50, TotalXP: 1},
		histErr: errors.New("lecture career_progression échouée"),
	}
	a := NewDataAdapter(srcFactory(live), nil).WithCareerSource(local)

	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{IncludeHistory: true})
	if err != nil {
		t.Fatalf("histErr ne doit PAS casser le snapshot: %v", err)
	}
	if len(snap.History) != 0 {
		t.Errorf("History sur histErr = %d, want 0 (dégradation propre)", len(snap.History))
	}
}
