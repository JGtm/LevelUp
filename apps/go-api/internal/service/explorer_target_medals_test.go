package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type stubMedalDefs struct {
	defs map[int64]port.MedalDefinitionRow
	err  error
}

func (s stubMedalDefs) LookupByIDs(_ context.Context, _ []int64, _ string) (map[int64]port.MedalDefinitionRow, error) {
	return s.defs, s.err
}

func TestBuildTargetTopMedals_SortAndEnrich(t *testing.T) {
	repo := stubMedalDefs{defs: map[int64]port.MedalDefinitionRow{
		10: {Label: "Double Kill", Description: "2 kills", Difficulty: "Normal", MedalType: "multikill"},
		20: {Label: "Killing Spree", Description: "5 kills", Difficulty: "Heroic", MedalType: "spree"},
	}}
	medals := []domain.RemoteMedalCount{
		{NameID: 10, Count: 3},
		{NameID: 20, Count: 50},
	}

	got := buildTargetTopMedals(context.Background(), repo, medals, "halo_infinite", "fr")
	if len(got) != 2 {
		t.Fatalf("attendu 2 médailles, got %d", len(got))
	}
	// Tri par count desc : Killing Spree (50) avant Double Kill (3).
	if got[0].MedalID != 20 || got[0].TotalCount != 50 {
		t.Errorf("attendu medal 20 (count 50) en tête, got %+v", got[0])
	}
	if got[0].Label != "Killing Spree" || got[0].Difficulty != "Heroic" {
		t.Errorf("enrichissement label/difficulty incorrect: %+v", got[0])
	}
	if got[0].ImageURL != "/static/medals/halo_infinite/20.png" {
		t.Errorf("image URL attendue /static/medals/halo_infinite/20.png, got %q", got[0].ImageURL)
	}
}

func TestBuildTargetTopMedals_CapAt20(t *testing.T) {
	medals := make([]domain.RemoteMedalCount, 0, 30)
	for i := 1; i <= 30; i++ {
		medals = append(medals, domain.RemoteMedalCount{NameID: int64(i), Count: i})
	}
	got := buildTargetTopMedals(context.Background(), stubMedalDefs{}, medals, "halo_infinite", "fr")
	if len(got) != explorerTopMedalsCap {
		t.Fatalf("attendu cap %d, got %d", explorerTopMedalsCap, len(got))
	}
	// Le plus gros count (30) doit être en tête.
	if got[0].TotalCount != 30 {
		t.Errorf("attendu top count 30, got %d", got[0].TotalCount)
	}
}

func TestBuildTargetTopMedals_LookupError(t *testing.T) {
	// LookupByIDs échoue → dégradation gracieuse : médailles rendues avec
	// image + compteur, mais label/description vides.
	repo := stubMedalDefs{err: errors.New("metadata db down")}
	medals := []domain.RemoteMedalCount{{NameID: 42, Count: 9}}

	got := buildTargetTopMedals(context.Background(), repo, medals, "halo_infinite", "fr")
	if len(got) != 1 {
		t.Fatalf("attendu 1 médaille malgré l'erreur lookup, got %d", len(got))
	}
	m := got[0]
	if m.MedalID != 42 || m.TotalCount != 9 {
		t.Errorf("compteur/id attendus (42, 9), got (%d, %d)", m.MedalID, m.TotalCount)
	}
	if m.ImageURL != "/static/medals/halo_infinite/42.png" {
		t.Errorf("image URL attendue malgré l'erreur, got %q", m.ImageURL)
	}
	if m.Label != "" || m.Description != "" {
		t.Errorf("label/description attendus vides en cas d'erreur lookup, got label=%q desc=%q", m.Label, m.Description)
	}
}

func TestBuildTargetTopMedals_NilSafe(t *testing.T) {
	if got := buildTargetTopMedals(context.Background(), nil, []domain.RemoteMedalCount{{NameID: 1, Count: 1}}, "t", "fr"); got != nil {
		t.Errorf("repo nil → nil attendu, got %v", got)
	}
	if got := buildTargetTopMedals(context.Background(), stubMedalDefs{}, nil, "t", "fr"); got != nil {
		t.Errorf("médailles vides → nil attendu, got %v", got)
	}
}

// TestBuildTargetProfile_TopMedalsLocal : le bloc « Top médailles » du profil de
// combat suit le toggle — TopMedalsLocal est agrégé sur les matchs LOCAUX et
// reste vide quand le profil de combat local l'est. Sans auth, les sous-blocs
// live (identité/carrière/CSR) sont skippés : on isole bien le chemin local.
func TestBuildTargetProfile_TopMedalsLocal(t *testing.T) {
	t.Parallel()
	defs := stubMedalDefs{defs: map[int64]port.MedalDefinitionRow{
		7: {Label: "Frag parfait", MedalType: "skill"},
	}}

	t.Run("profil de combat local vide → top médailles local vide", func(t *testing.T) {
		t.Parallel()
		repo := &mockExplorerRepo{
			// Aucun match local, mais le repo médailles renverrait des données :
			// on vérifie qu'il n'est même pas interrogé.
			topMedals: []domain.RemoteMedalCount{{NameID: 7, Count: 9}},
		}
		svc := NewExplorerService(repo, "self").
			WithTargetProfileProviders(ExplorerTargetProfileDeps{MedalDefs: defs, TitleSlug: "halo_infinite"})

		profile := svc.buildTargetProfile(context.Background(), "target-xuid", "Target", nil)
		if profile == nil {
			t.Fatal("profil attendu non-nil")
		}
		if len(profile.CombatProfileLocal) != 0 {
			t.Fatalf("pré-condition : profil de combat local attendu vide, got %d", len(profile.CombatProfileLocal))
		}
		if len(profile.TopMedalsLocal) != 0 {
			t.Errorf("top médailles local attendu vide, got %+v", profile.TopMedalsLocal)
		}
	})

	t.Run("profil de combat local peuplé → top médailles local enrichi", func(t *testing.T) {
		t.Parallel()
		repo := &mockExplorerRepo{
			recentMatches: []domain.ExplorerTargetRecentMatch{{MatchID: "m1", Outcome: 2, Kills: 10}},
			topMedals:     []domain.RemoteMedalCount{{NameID: 7, Count: 9}},
		}
		svc := NewExplorerService(repo, "self").
			WithTargetProfileProviders(ExplorerTargetProfileDeps{MedalDefs: defs, TitleSlug: "halo_infinite"})

		profile := svc.buildTargetProfile(context.Background(), "target-xuid", "Target", nil)
		if profile == nil {
			t.Fatal("profil attendu non-nil")
		}
		if len(profile.TopMedalsLocal) != 1 {
			t.Fatalf("attendu 1 médaille locale, got %d : %+v", len(profile.TopMedalsLocal), profile.TopMedalsLocal)
		}
		got := profile.TopMedalsLocal[0]
		if got.MedalID != 7 || got.TotalCount != 9 {
			t.Errorf("médaille locale attendue (id 7, count 9), got (%d, %d)", got.MedalID, got.TotalCount)
		}
		// Même enrichissement que le lifetime : libellé + image via buildTargetTopMedals.
		if got.Label != "Frag parfait" {
			t.Errorf("libellé attendu enrichi comme le lifetime, got %q", got.Label)
		}
		if got.ImageURL != "/static/medals/halo_infinite/7.png" {
			t.Errorf("image attendue /static/medals/halo_infinite/7.png, got %q", got.ImageURL)
		}
		// TopMedals (lifetime) reste vide sans auth : les deux listes sont bien distinctes.
		if len(profile.TopMedals) != 0 {
			t.Errorf("top médailles lifetime attendu vide sans auth, got %+v", profile.TopMedals)
		}
	})

	t.Run("erreur repo → dégradation en vide, jamais fatale", func(t *testing.T) {
		t.Parallel()
		repo := &mockExplorerRepo{
			recentMatches: []domain.ExplorerTargetRecentMatch{{MatchID: "m1"}},
			topMedalsErr:  errors.New("shared reader down"),
		}
		svc := NewExplorerService(repo, "self").
			WithTargetProfileProviders(ExplorerTargetProfileDeps{MedalDefs: defs, TitleSlug: "halo_infinite"})

		profile := svc.buildTargetProfile(context.Background(), "target-xuid", "Target", nil)
		if profile == nil {
			t.Fatal("profil attendu non-nil malgré l'erreur médailles")
		}
		if len(profile.TopMedalsLocal) != 0 {
			t.Errorf("top médailles local attendu vide sur erreur, got %+v", profile.TopMedalsLocal)
		}
		if len(profile.CombatProfileLocal) != 1 {
			t.Errorf("le profil de combat local doit survivre à l'échec médailles, got %d", len(profile.CombatProfileLocal))
		}
	})
}
