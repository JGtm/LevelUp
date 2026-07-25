package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

func tm(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestBuildMatchesPerSeason_Bucketing(t *testing.T) {
	endS1 := tm("2022-05-03T17:00:00Z")
	endS2 := tm("2022-11-08T17:00:00Z")
	seasons := []SeasonCatalogEntry{
		{ID: "season1", Label: "Heroes of Reach", Start: tm("2021-12-08T17:00:00Z"), End: &endS1, Extra: map[string]string{"short_label": "S1"}},
		{ID: "season2", Label: "Lone Wolves", Start: tm("2022-05-03T17:00:00Z"), End: &endS2, Extra: map[string]string{"short_label": "S2"}},
		{ID: "season3", Label: "Open", Start: tm("2022-11-08T17:00:00Z"), End: nil}, // saison ouverte, pas de short_label
	}
	starts := []time.Time{
		tm("2022-01-10T10:00:00Z"), // S1
		tm("2022-02-10T10:00:00Z"), // S1
		tm("2022-06-10T10:00:00Z"), // S2
		tm("2023-01-10T10:00:00Z"), // S3 (ouverte)
		tm("2020-01-01T10:00:00Z"), // avant toute saison → ignoré
	}

	got := buildMatchesPerSeason(starts, seasons)
	if len(got) != 3 {
		t.Fatalf("attendu 3 saisons avec matchs, got %d (%+v)", len(got), got)
	}
	// Ordre préservé (DisplayOrder via l'ordre du slice).
	if got[0].SeasonID != "season1" || got[0].Matches != 2 || got[0].SeasonName != "S1" {
		t.Errorf("S1 attendu (2 matchs, label S1), got %+v", got[0])
	}
	if got[1].Matches != 1 || got[1].SeasonName != "S2" {
		t.Errorf("S2 attendu 1 match, got %+v", got[1])
	}
	// season3 sans short_label → fallback Label.
	if got[2].SeasonID != "season3" || got[2].Matches != 1 || got[2].SeasonName != "Open" {
		t.Errorf("S3 attendu 1 match label fallback 'Open', got %+v", got[2])
	}
}

func TestBuildMatchesPerSeason_Empty(t *testing.T) {
	seasons := []SeasonCatalogEntry{{ID: "s1", Start: tm("2021-01-01T00:00:00Z")}}
	if got := buildMatchesPerSeason(nil, seasons); got != nil {
		t.Errorf("starts vide → nil, got %v", got)
	}
	if got := buildMatchesPerSeason([]time.Time{tm("2021-06-01T00:00:00Z")}, nil); got != nil {
		t.Errorf("seasons vide → nil, got %v", got)
	}
}

func TestExtractSeasonNumber(t *testing.T) {
	cases := map[string]int{
		"Seasons/Season7.json":          7,
		"Seasons/Season10.json":         10,
		"Seasons/Season6-2.json":        6, // premier entier
		"Csr/Seasons/CsrSeason9-1.json": 9,
		"S13":                           13,
		"season1":                       1,
		"Season-Winter-Break-22.json":   22,
		"":                              -1,
		"nope":                          -1,
	}
	for in, want := range cases {
		if got := extractSeasonNumber(in); got != want {
			t.Errorf("extractSeasonNumber(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestGroupMatchmadeSeasonsByNumber(t *testing.T) {
	in := []string{
		"Seasons/Season6.json",
		"Seasons/Season6-2.json", // opération intra-S6 → même numéro
		"Seasons/Season7.json",
		"Csr/Seasons/CsrSeason6-1.json", // chemin CSR → ignoré
		"garbage",                       // ignoré
	}
	got := groupMatchmadeSeasonsByNumber(in)
	if len(got[6]) != 2 {
		t.Errorf("S6 attendu 2 chemins (Season6 + Season6-2), got %v", got[6])
	}
	if len(got[7]) != 1 {
		t.Errorf("S7 attendu 1 chemin, got %v", got[7])
	}
	if _, ok := got[9]; ok {
		t.Errorf("un chemin CSR ne doit pas créer d'entrée matchmade (n°9): %v", got)
	}
}

// mockSeasonSR : compte de matchs canné par (chemin CMS, filtre ranked).
// `fail` simule un chemin CMS en erreur (404 saison inconnue / API en panne) ;
// `perPath` trace le nombre d'appels par chemin (garde-rail anti-double-appel).
type mockSeasonSR struct {
	ranked   map[string]int
	unranked map[string]int
	fail     map[string]error
	calls    int32

	mu      sync.Mutex
	perPath map[string]int
}

func (m *mockSeasonSR) FetchSeasonServiceRecord(_ context.Context, _, seasonID string, isRanked *bool) (int, error) {
	atomic.AddInt32(&m.calls, 1)
	m.mu.Lock()
	if m.perPath == nil {
		m.perPath = make(map[string]int)
	}
	m.perPath[seasonID]++
	m.mu.Unlock()
	if err := m.fail[seasonID]; err != nil {
		return 0, err
	}
	if isRanked == nil {
		return m.ranked[seasonID] + m.unranked[seasonID], nil
	}
	if *isRanked {
		return m.ranked[seasonID], nil
	}
	return m.unranked[seasonID], nil
}

// pathCalls retourne le nombre d'appels effectués sur un chemin CMS donné.
func (m *mockSeasonSR) pathCalls(seasonID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.perPath[seasonID]
}

func breakdownTestSeasons() []SeasonCatalogEntry {
	mk := func(id, short, csr string) SeasonCatalogEntry {
		return SeasonCatalogEntry{ID: id, Label: id, Extra: map[string]string{"short_label": short, "csr_season_id": csr}}
	}
	return []SeasonCatalogEntry{
		mk("season6", "S6", "CsrSeason6-1"),
		mk("season7", "S7", "CsrSeason7-1"),
		mk("season8", "S8", "CsrSeason8-1"), // non joué
	}
}

func TestComputeSeasonBreakdown_Live(t *testing.T) {
	sr := &mockSeasonSR{
		ranked:   map[string]int{"Seasons/Season6.json": 10, "Seasons/Season6-2.json": 5, "Seasons/Season7.json": 20},
		unranked: map[string]int{"Seasons/Season6.json": 3, "Seasons/Season6-2.json": 2, "Seasons/Season7.json": 8},
	}
	csrCalls := int32(0)
	csr := SeasonCSRPeakFunc(func(_ context.Context, _, _ string, engaged []string) (*SeasonCSRPeak, error) {
		if len(engaged) == 0 {
			return nil, nil // contrat : pas d'appel utile sans playlist engagée
		}
		atomic.AddInt32(&csrCalls, 1)
		badge := "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"
		return &SeasonCSRPeak{Tier: "Onyx", SubTier: 0, BadgeURL: &badge}, nil
	})

	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, SeasonCSR: csr, TitleSlug: "halo_infinite",
		})

	played := []string{"Seasons/Season6.json", "Seasons/Season6-2.json", "Seasons/Season7.json", "Csr/Seasons/CsrSeason6-1.json"}
	engaged := []string{"ranked-arena-asset"}
	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true, played, engaged)
	if status != domain.ExplorerLiveOK {
		t.Errorf("status attendu ok (breakdown live), got %q", status)
	}

	if len(out) != 3 {
		t.Fatalf("attendu 3 saisons (catalogue complet), got %d", len(out))
	}
	// S6 : total sommé sur les deux chemins (opération Season6-2 incluse) :
	// (10+3) + (5+2) = 20.
	if out[0].Matches != 20 {
		t.Errorf("S6 attendu total=20, got %+v", out[0])
	}
	if out[0].CSRTier != "Onyx" || out[0].CSRBadgeImageURL == nil {
		t.Errorf("S6 badge CSR attendu, got tier=%q badge=%v", out[0].CSRTier, out[0].CSRBadgeImageURL)
	}
	// S7 : un seul chemin → 20+8 = 28.
	if out[1].Matches != 28 {
		t.Errorf("S7 attendu total=28, got %+v", out[1])
	}
	// S8 : non jouée → barre vide, aucun appel CSR.
	if out[2].Matches != 0 || out[2].CSRTier != "" {
		t.Errorf("S8 attendu vide, got %+v", out[2])
	}
	if csrCalls != 2 {
		t.Errorf("CSR attendu 2 appels (S6+S7 jouées), got %d", csrCalls)
	}
}

func TestComputeSeasonBreakdown_FallbackNoAuth(t *testing.T) {
	sr := &mockSeasonSR{}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	// hasAuth=false → fallback bucketing local : aucun appel live par saison.
	_, status := svc.computeSeasonBreakdown(context.Background(), "target-xuid", "Target", false,
		[]string{"Seasons/Season7.json"}, []string{"ranked-arena-asset"})
	if atomic.LoadInt32(&sr.calls) != 0 {
		t.Errorf("aucun appel SeasonSR attendu sans auth, got %d", sr.calls)
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status attendu no_auth, got %q", status)
	}
}

// TestComputeSeasonBreakdown_NoEngagedPlaylists_SkipsCSR : un joueur sans playlist
// ranked engagée (engagedPlaylistIDs vide → ex. joueur social) ne déclenche AUCUN
// appel CSR (optim), mais les totaux de matchs par saison restent calculés.
func TestComputeSeasonBreakdown_NoEngagedPlaylists_SkipsCSR(t *testing.T) {
	sr := &mockSeasonSR{
		ranked:   map[string]int{"Seasons/Season7.json": 0},
		unranked: map[string]int{"Seasons/Season7.json": 12},
	}
	csrCalls := int32(0)
	csr := SeasonCSRPeakFunc(func(_ context.Context, _, _ string, _ []string) (*SeasonCSRPeak, error) {
		atomic.AddInt32(&csrCalls, 1)
		return nil, nil
	})
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, SeasonCSR: csr, TitleSlug: "halo_infinite",
		})

	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true,
		[]string{"Seasons/Season7.json"}, nil /* aucune playlist engagée */)
	if status != domain.ExplorerLiveOK {
		t.Errorf("status attendu ok (breakdown live), got %q", status)
	}

	if out[1].Matches != 12 {
		t.Errorf("S7 total attendu 12, got %+v", out[1])
	}
	if csrCalls != 0 {
		t.Errorf("aucun appel CSR attendu sans playlist engagée, got %d", csrCalls)
	}
}

// ---------------------------------------------------------------------------
// V721-06 — union catalogue + live des chemins CMS
// ---------------------------------------------------------------------------

func TestDeterministicSeasonPath(t *testing.T) {
	mk := func(id string, extra map[string]string) SeasonCatalogEntry {
		return SeasonCatalogEntry{ID: id, Label: id, Extra: extra}
	}
	cases := []struct {
		name  string
		entry SeasonCatalogEntry
		want  string
	}{
		{"id numéroté", mk("season7", nil), "Seasons/Season7.json"},
		{"id numéroté deux chiffres", mk("season13", nil), "Seasons/Season13.json"},
		{"id hors gabarit sans extra", mk("season_winter_22", map[string]string{"short_label": "Winter 22"}), ""},
		{
			"extra matchmade_path",
			mk("season_winter_22", map[string]string{"matchmade_path": "Seasons/Season-Winter-Break-22.json"}),
			"Seasons/Season-Winter-Break-22.json",
		},
		// L'override TOML prime sur le gabarit (permet de corriger un chemin
		// sans toucher au code si Waypoint renomme une saison numérotée).
		{"extra prioritaire", mk("season4", map[string]string{"matchmade_path": "Seasons/Custom4.json"}), "Seasons/Custom4.json"},
		{"id non saison", mk("winter", nil), ""},
	}
	for _, c := range cases {
		if got := deterministicSeasonPath(&c.entry); got != c.want {
			t.Errorf("%s : deterministicSeasonPath(%q) = %q, want %q", c.name, c.entry.ID, got, c.want)
		}
	}
}

// TestSeasonCMSPaths_UnionDedup : l'union met le chemin déterministe en tête,
// ajoute les opérations intra-saison remontées par le live, et ne duplique JAMAIS
// un chemin présent dans les deux sources (sinon double comptage du total).
func TestSeasonCMSPaths_UnionDedup(t *testing.T) {
	live := map[int][]string{
		6: {"Seasons/Season6.json", "Seasons/Season6-2.json"},
		9: {"Seasons/Season9.json"},
	}
	s6 := SeasonCatalogEntry{ID: "season6", Label: "season6", Extra: map[string]string{"short_label": "S6"}}
	got := seasonCMSPaths(&s6, live)
	want := []string{"Seasons/Season6.json", "Seasons/Season6-2.json"}
	if len(got) != len(want) {
		t.Fatalf("S6 : %d chemins (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("S6 chemin %d = %q, want %q", i, got[i], want[i])
		}
	}

	// Saison au catalogue absente du live : le chemin déterministe suffit.
	s7 := SeasonCatalogEntry{ID: "season7", Label: "season7"}
	if got := seasonCMSPaths(&s7, live); len(got) != 1 || got[0] != "Seasons/Season7.json" {
		t.Errorf("S7 attendu [Seasons/Season7.json], got %v", got)
	}

	// Saison hors gabarit sans extra ni chemin live : rien à interroger.
	orphan := SeasonCatalogEntry{ID: "operation_alpha", Label: "Alpha"}
	if got := seasonCMSPaths(&orphan, live); got != nil {
		t.Errorf("saison sans chemin déductible ni live → nil, got %v", got)
	}
}

// TestComputeSeasonBreakdown_CatalogSeasonMissingFromLive — CŒUR DU CORRECTIF
// (V721-06) : Subqueries.SeasonIds ne liste PAS tout l'historique du joueur
// (5 saisons sur 14 observées le 2026-07-22). Les saisons du catalogue absentes
// de cette liste doivent quand même être interrogées via leur chemin déterministe.
// Sans le correctif, S7 et S8 restent à 0 (barres vides) : ce test échoue.
func TestComputeSeasonBreakdown_CatalogSeasonMissingFromLive(t *testing.T) {
	sr := &mockSeasonSR{
		unranked: map[string]int{
			"Seasons/Season6.json": 13,
			"Seasons/Season7.json": 28,
			"Seasons/Season8.json": 42,
		},
	}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	// Le live ne remonte que S6 — S7 et S8 sont pourtant jouées.
	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true,
		[]string{"Seasons/Season6.json"}, nil)
	if status != domain.ExplorerLiveOK {
		t.Errorf("status attendu ok, got %q", status)
	}
	if len(out) != 3 {
		t.Fatalf("attendu 3 saisons (catalogue complet), got %d", len(out))
	}
	if out[0].Matches != 13 {
		t.Errorf("S6 (remontée par le live) attendu 13, got %+v", out[0])
	}
	if out[1].Matches != 28 {
		t.Errorf("S7 absente du live : attendu 28 via le chemin déterministe, got %+v", out[1])
	}
	if out[2].Matches != 42 {
		t.Errorf("S8 absente du live : attendu 42 via le chemin déterministe, got %+v", out[2])
	}
	for i := range out {
		if out[i].Unresolved {
			t.Errorf("saison %q ne doit pas être indéterminée (l'API a répondu) : %+v", out[i].SeasonID, out[i])
		}
	}
	if got := int(atomic.LoadInt32(&sr.calls)); got != 3 {
		t.Errorf("3 appels SR attendus (1 par saison du catalogue), got %d", got)
	}
}

// TestComputeSeasonBreakdown_NoDoubleCountOnOverlap : quand le live remonte
// EXACTEMENT le chemin déterministe, le chemin n'est interrogé qu'une fois et le
// total n'est pas doublé (une union naïve donnerait 56 et 2 appels).
func TestComputeSeasonBreakdown_NoDoubleCountOnOverlap(t *testing.T) {
	sr := &mockSeasonSR{unranked: map[string]int{"Seasons/Season7.json": 28}}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	out, _ := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true,
		[]string{"Seasons/Season7.json"}, nil)
	if out[1].Matches != 28 {
		t.Errorf("S7 attendu 28 (chemin compté une seule fois), got %+v", out[1])
	}
	if n := sr.pathCalls("Seasons/Season7.json"); n != 1 {
		t.Errorf("Seasons/Season7.json attendu 1 appel (dédup union), got %d", n)
	}
}

// TestComputeSeasonBreakdown_FailedSeasonIsUnresolved : les trois états sont
// distincts — jouée (>0), NON jouée (l'API répond 0), indéterminée (appel en
// échec). Avant V721-06 les deux derniers étaient confondus en "barre vide".
func TestComputeSeasonBreakdown_FailedSeasonIsUnresolved(t *testing.T) {
	sr := &mockSeasonSR{
		unranked: map[string]int{"Seasons/Season6.json": 13}, // S7 → 0 (non jouée)
		fail:     map[string]error{"Seasons/Season8.json": errors.New("doGet: HTTP 500")},
	}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true, nil, nil)
	if out[0].Matches != 13 || out[0].Unresolved {
		t.Errorf("S6 attendu jouée (13 matchs, résolue), got %+v", out[0])
	}
	if out[1].Matches != 0 || out[1].Unresolved {
		t.Errorf("S7 attendu NON jouée (0 match, résolue — l'API a répondu), got %+v", out[1])
	}
	if !out[2].Unresolved || out[2].Matches != 0 {
		t.Errorf("S8 attendu indéterminée (appel en échec), got %+v", out[2])
	}
	if status != domain.ExplorerLiveLocalPartial {
		t.Errorf("status attendu local_partial (live partiel), got %q", status)
	}
}

// TestComputeSeasonBreakdown_AllSeasonsFailed : aucune saison résolue → statut
// failed (et non ok), toutes les lignes indéterminées.
func TestComputeSeasonBreakdown_AllSeasonsFailed(t *testing.T) {
	boom := errors.New("doGet: HTTP 503")
	sr := &mockSeasonSR{fail: map[string]error{
		"Seasons/Season6.json": boom,
		"Seasons/Season7.json": boom,
		"Seasons/Season8.json": boom,
	}}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true, nil, nil)
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status attendu failed, got %q", status)
	}
	for i := range out {
		if !out[i].Unresolved {
			t.Errorf("saison %q attendue indéterminée, got %+v", out[i].SeasonID, out[i])
		}
	}
}

// TestComputeSeasonBreakdown_NoCMSPathFallsBackLocal : un titre dont les saisons
// n'ont ni ID numéroté, ni matchmade_path, ni chemin live → aucun appel réseau,
// repli sur le bucketing local (dégradation propre, title-agnostic).
func TestComputeSeasonBreakdown_NoCMSPathFallsBackLocal(t *testing.T) {
	sr := &mockSeasonSR{}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons:  []SeasonCatalogEntry{{ID: "operation_alpha", Label: "Alpha"}},
			SeasonSR: sr, TitleSlug: "halo_5",
		})

	_, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true, nil, nil)
	if status != domain.ExplorerLiveLocalPartial {
		t.Errorf("status attendu local_partial (repli local), got %q", status)
	}
	if got := atomic.LoadInt32(&sr.calls); got != 0 {
		t.Errorf("aucun appel SR attendu sans chemin CMS, got %d", got)
	}
}

// TestDeterministicSeasonPath_RealCatalog cadenasse l'objectif « 14/14 » : CHAQUE
// saison du catalogue Halo Infinite doit avoir un chemin CMS interrogeable sans
// dépendre du live (ID numéroté, ou extra.matchmade_path pour l'opération
// hivernale). Un ajout de saison hors gabarit sans matchmade_path fait échouer
// ce test — c'est le garde-rail de la couverture complète.
func TestDeterministicSeasonPath_RealCatalog(t *testing.T) {
	// apps/go-api/internal/service → racine du repo = 4 niveaux.
	tomlPath := filepath.Join("..", "..", "..", "..",
		"config", "titles", "halo_infinite", "mappings", "assets.toml")
	set, err := mappings.LoadAssetsFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadAssetsFromFile(%s): %v", tomlPath, err)
	}
	catalog := projectTOMLSeasons(set)
	if len(catalog) < 14 {
		t.Fatalf("catalogue saisons = %d entrées, want >= 14 (S1-S13 + Winter Update)", len(catalog))
	}
	for i := range catalog {
		if p := deterministicSeasonPath(&catalog[i]); p == "" {
			t.Errorf("saison %q : aucun chemin CMS déductible — ajouter extra.matchmade_path dans assets.toml",
				catalog[i].ID)
		}
	}
}
