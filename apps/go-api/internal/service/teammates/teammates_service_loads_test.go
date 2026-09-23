package teammates

// teammates_service_loads_test.go — lot perf L2 : les lectures partagées d'une requête
// (teammates_service_loads.go). Chaque test fait tourner GetPage sur une page COMPLÈTE (les
// blocs consommateurs rendent tous une section) et compte les lectures faites.

import (
	"context"
	"strings"
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

// TestGetPage_UnLoadForParMembre (L2.3 / D2.3) : heatmap, stats par minute, radar, séries de
// performance et bandeau relisent l'historique des membres ; chaque membre n'est lu qu'UNE
// fois par requête (avant le lot : 9 lectures pour un coéquipier, dont 2 par l'intensité).
func TestGetPage_UnLoadForParMembre(t *testing.T) {
	repo, loader := pageCompleteFixture()
	resp, err := servicePageComplete(repo, loader).GetPage(context.Background(), "x_main",
		domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally"}})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if resp.MapHeatmap == nil || len(resp.PerMinuteStats) == 0 || len(resp.SynergyRadar) == 0 ||
		len(resp.PerformanceSeries) == 0 || resp.Header == nil || len(resp.Header.PlayerCards) == 0 {
		t.Fatal("les consommateurs de LoadFor doivent tous rendre leur section")
	}
	parMembre := map[string]int{}
	for _, gt := range loader.calls {
		parMembre[gt]++
	}
	if parMembre["Main"] != 1 || parMembre["Ally"] != 1 || len(loader.calls) != 2 {
		t.Fatalf("LoadFor par membre = %v (appels %v), attendu une lecture chacun", parMembre, loader.calls)
	}
}

// TestGetPage_SansPopulationEscouade_BandeauSeulLitLesCoequipiers : sans match commun, seul le
// bandeau relit les coéquipiers — le préchargement ne lit pas le joueur principal pour rien.
func TestGetPage_SansPopulationEscouade_BandeauSeulLitLesCoequipiers(t *testing.T) {
	repo, loader := pageCompleteFixture()
	repo.squadRows = nil
	if _, err := servicePageComplete(repo, loader).GetPage(context.Background(), "x_main",
		domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally"}}); err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if len(loader.calls) != 1 || loader.calls[0] != "Ally" {
		t.Fatalf("LoadFor = %v, attendu [Ally] (bandeau seul)", loader.calls)
	}
	if repo.impactCalls != 0 {
		t.Fatalf("LoadImpactEvents appelé %d fois sans population escouade, attendu 0", repo.impactCalls)
	}
}

// TestBuildSquadIntensityProfile_XUIDsDeLaPage (D2.3) : la ligne d'un joueur lit son xuid dans
// la page (mainXUID, teammates[].XUID), jamais par LoadFor — un coéquipier sans historique
// chargeable (non suivi, chargeur absent) a quand même sa ligne, comme au premier frag.
func TestBuildSquadIntensityProfile_XUIDsDeLaPage(t *testing.T) {
	repo, rows := tlFixture()
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	ally := tlAllyXUID
	got := svc.buildSquadIntensityProfile(context.Background(), rows, "main", tlMainXUID,
		[]string{"Ally"}, []domain.TeammateRow{{Gamertag: "Ally", XUID: &ally}}, nil)
	for _, gt := range []string{"main", "Ally"} {
		if r := tlRowFor(t, got, gt, "m1"); r.Phases[1] != 1 {
			t.Errorf("ligne %s : attendu le frag du bucket 1 (xuid de la page), phases %v", gt, r.Phases)
		}
	}
}

// tacticalRepoEspion capture aussi les requêtes du contexte des morts (nuage d'isolement).
type tacticalRepoEspion struct {
	*mockTacticalRepo
	vuesMorts []domain.TacticalQuery
}

func (e *tacticalRepoEspion) MortsAvecContexte(ctx context.Context, q domain.TacticalQuery) (domain.TacticalMortsContexte, error) {
	e.vuesMorts = append(e.vuesMorts, q)
	return e.mockTacticalRepo.MortsAvecContexte(ctx, q)
}

// TestBuildSquadEchange_JournalRestreintALaComposition (L2.4 / D2.4) : le journal des morts et
// le contexte des morts sont lus UNE fois chacun, avec pour liste blanche l'historique de la
// COMPOSITION (l'habituel), jamais tout l'historique du joueur.
func TestBuildSquadEchange_JournalRestreintALaComposition(t *testing.T) {
	ids := []string{"m1", "m2"}
	espion := &tacticalRepoEspion{mockTacticalRepo: &mockTacticalRepo{
		lecture: domain.TacticalKillEvents{
			Univers: domain.TacticalUnivers{Matchs: isolementMatches("Arena", ids...), Equipes: equipesDeuxContreDeux(ids...)},
			Events: []domain.KillEvent{
				{MatchID: "m2", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 10_000},
				{MatchID: "m2", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 12_000},
			},
		},
		morts: domain.TacticalMortsContexte{Morts: []domain.MortContexte{
			mortContexte("m2", "x_main", 10_000, metres(9), 1, 0),
		}},
	}}
	svc := &TeammatesService{
		titleSlug: "halo_infinite", gamertag: "main",
		tacticalRepo: espion, caps: capsFiables(), radarRange: isolementRadar(),
	}
	// Le filtre de la page retient m2 ; l'historique de la composition compte m1 et m2.
	got := svc.buildSquadEchange(context.Background(),
		isolementRows("m2"), isolementRows("m1", "m2"), "main", "x_main", echangeMates("Ami"))
	if got == nil || got.NuageIsolement == nil {
		t.Fatalf("section echange et nuage attendus, obtenu %+v", got)
	}
	if len(espion.vues) != 1 || len(espion.vuesMorts) != 1 {
		t.Fatalf("lectures : journal %d, contexte %d — attendu une de chaque", len(espion.vues), len(espion.vuesMorts))
	}
	for _, q := range []domain.TacticalQuery{espion.vues[0], espion.vuesMorts[0]} {
		if !q.Matchs.Restreint() || strings.Join(q.Matchs.IDs(), ",") != "m1,m2" {
			t.Errorf("liste blanche = %v (restreinte %v), attendu l'habituel [m1 m2]", q.Matchs.IDs(), q.Matchs.Restreint())
		}
	}
}
