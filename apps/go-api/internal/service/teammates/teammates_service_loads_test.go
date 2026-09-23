package teammates

// teammates_service_loads_test.go — lot perf L2 : les lectures partagées d'une requête
// (teammates_service_loads.go). Chaque test fait tourner GetPage sur une page COMPLÈTE (les
// blocs consommateurs rendent tous une section) et compte les lectures faites.

import (
	"context"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// countingSquadRepo compte les lectures Q32 faites par une requête.
type countingSquadRepo struct {
	*mockSquadRepo
	mu           sync.Mutex
	impactCalls  int
	impactMatchs [][]string
}

func (c *countingSquadRepo) LoadImpactEvents(ctx context.Context, ids []string) ([]domain.ImpactEventRow, error) {
	c.mu.Lock()
	c.impactCalls++
	c.impactMatchs = append(c.impactMatchs, append([]string(nil), ids...))
	c.mu.Unlock()
	return c.mockSquadRepo.LoadImpactEvents(ctx, ids)
}

// pageCompleteFixture : le main (x_main) et un coéquipier suivi (Ally, x_ally) sur trois
// matchs communs — assez pour que la matrice d'impact, le profil d'intensité (≥ 3 matchs),
// les séries de performance (spree calculée depuis les events) et le premier frag rendent
// chacun une section.
func pageCompleteFixture() (*countingSquadRepo, *fakeSquadLoader) {
	t0 := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	ids := []string{"m1", "m2", "m3"}
	var squadRows []domain.SquadMatchRow
	var synth []legacymatch.SynthesisMatchRow
	var impacts []domain.ImpactEventRow
	var allies []domain.AllyParticipant
	mainRows := make([]canonical.PlayerMatchRow, 0, len(ids))
	allyRows := make([]canonical.PlayerMatchRow, 0, len(ids))
	for i, id := range ids {
		start := t0.Add(time.Duration(i) * time.Hour)
		squadRows = append(squadRows, domain.SquadMatchRow{
			MatchID: id, StartTime: start, MapUI: "Aquarius", MapID: "map-aq",
			Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 3,
			TimePlayedSecs: 600, IsWithFriends: true, SessionLabel: strPtr("S1"),
		})
		synth = append(synth, legacymatch.SynthesisMatchRow{
			MatchID: id, StartTime: start, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5,
			IsWithFriends: true, SessionLabel: strPtr("S1"),
		})
		impacts = append(impacts,
			domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "kill", TimeMS: 10_000},
			domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "death", TimeMS: 20_000},
			domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "kill", TimeMS: 300_000},
			domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "death", TimeMS: 400_000},
		)
		allies = append(allies,
			domain.AllyParticipant{MatchID: id, XUID: "x_main", Gamertag: "Main", Kills: 10, Deaths: 5, Assists: 3, Outcome: domain.OutcomeWin},
			domain.AllyParticipant{MatchID: id, XUID: "x_ally", Gamertag: "Ally", Kills: 4, Deaths: 7, Assists: 1, Outcome: domain.OutcomeWin},
		)
		mainRows = append(mainRows, rowWithStatsXUID("x_main", id, start, canonical.OutcomeWin, 10, 5, 3, 600, 45, 60))
		allyRows = append(allyRows, rowWithStatsXUID("x_ally", id, start, canonical.OutcomeWin, 4, 7, 1, 600, 40, 50))
	}
	repo := &countingSquadRepo{mockSquadRepo: &mockSquadRepo{
		topRows:    []domain.TopTeammateRow{{XUID: "x_ally", Gamertag: "Ally", GamesTogether: 3}},
		squadRows:  squadRows,
		synthRows:  synth,
		impactRows: impacts,
		allyRows:   allies,
	}}
	loader := &fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{"Main": mainRows, "Ally": allyRows}}
	return repo, loader
}

func servicePageComplete(repo *countingSquadRepo, loader *fakeSquadLoader) *TeammatesService {
	return NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, nil), "halo_infinite", "Main").
		WithSquadLoader(loader)
}

// TestGetPage_LitLesEvenementsDImpactUneSeuleFois (L2.2 / D2.2) : matrice d'impact, profil
// d'intensité, séries de performance et premier frag consomment UNE lecture Q32.
func TestGetPage_LitLesEvenementsDImpactUneSeuleFois(t *testing.T) {
	repo, loader := pageCompleteFixture()
	resp, err := servicePageComplete(repo, loader).GetPage(context.Background(), "x_main",
		domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally"}})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	// Les quatre consommateurs ont bien tourné — sinon « une lecture » ne prouverait rien.
	if resp.ImpactMatrix == nil || resp.IntensityProfile == nil || len(resp.FirstBlood) == 0 {
		t.Fatalf("sections attendues : matrice %v, intensité %v, premier frag %d",
			resp.ImpactMatrix != nil, resp.IntensityProfile != nil, len(resp.FirstBlood))
	}
	spree := false
	for _, pt := range resp.PerformanceSeries["Main"] {
		spree = spree || pt.MaxKillingSpree != nil
	}
	if !spree {
		t.Fatal("séries de performance : la spree calculée depuis les events est absente")
	}
	if repo.impactCalls != 1 {
		t.Fatalf("LoadImpactEvents appelé %d fois (matchs %v), attendu 1", repo.impactCalls, repo.impactMatchs)
	}
}

// TestGetPage_DeuxRequetesDeuxLectures : la mémoire est celle d'UNE requête — la suivante
// relit la base.
func TestGetPage_DeuxRequetesDeuxLectures(t *testing.T) {
	repo, loader := pageCompleteFixture()
	svc := servicePageComplete(repo, loader)
	for i := 0; i < 2; i++ {
		if _, err := svc.GetPage(context.Background(), "x_main",
			domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally"}}); err != nil {
			t.Fatalf("GetPage %d : %v", i, err)
		}
	}
	if repo.impactCalls != 2 {
		t.Fatalf("LoadImpactEvents appelé %d fois sur deux requêtes, attendu 2", repo.impactCalls)
	}
}

// TestCleDEnsemble : l'ensemble identifie la lecture, pas l'ordre de la liste.
func TestCleDEnsemble(t *testing.T) {
	if cleDEnsemble([]string{"m2", "m1", "m1"}) != cleDEnsemble([]string{"m1", "m2"}) {
		t.Error("deux listes du même ensemble doivent partager la lecture")
	}
	if cleDEnsemble([]string{"m1"}) == cleDEnsemble([]string{"m1", "m2"}) {
		t.Error("deux ensembles différents ne partagent pas la lecture")
	}
}
