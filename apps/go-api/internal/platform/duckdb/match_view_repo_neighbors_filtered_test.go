//go:build integration

// Package duckdb — match_view_repo_neighbors_filtered_test.go : test
// d'intégration de Q25NeighborMatchesTemplate avec dataset hétérogène
// (Phase 2b du rework header MatchView).
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestMatchViewRepo_GetMatchNeighborsFiltered -v
//
// Couvre 8 matchs étalés sur 30 jours, mix PvP (Slayer/BTB/Fiesta) + PvE
// (Firefight), playlists distinctes, outcomes variés (win/loss/draw/dnf).
// Vérifie que prev/next pointent dans le bon sous-ensemble selon les filtres.
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite"
)

// testModeTaxonomy : la taxonomie Halo Infinite injectée dans le repo pour les
// tests neighbors (le wiring prod l'injecte via haloInfiniteModeTaxonomy ; F15-2).
func testModeTaxonomy() analysis.ModeTaxonomy {
	return analysis.ModeTaxonomy{
		InferCategory: halo_infinite.InferModeCategoryFromPairName,
		PrefixesFor:   halo_infinite.PairNamePrefixesForCategory,
		AllPrefixes:   halo_infinite.AllKnownPairNamePrefixes,
		Other:         halo_infinite.ModeCategoryOther,
	}
}

// matchSeed : description compacte d'un match pour le dataset de test.
type matchSeed struct {
	id       string
	startUTC string // "2026-04-15 14:00:00+00"
	mapName  string
	pairName string // "BTB:CTF" / "Arena:Slayer" / etc.
	playlist string
	outcome  int // 1=draw 2=win 3=loss 4=dnf
	isRanked bool
}

// neighborsTestDataset : 8 matchs bien diversifiés.
// Ordre chronologique DESC (récent en tête) — m8 le plus récent, m1 le plus ancien.
var neighborsTestDataset = []matchSeed{
	{"n8", "2026-04-30 18:00:00+00", "Aquarius", "Arena:Slayer", "Ranked Slayer", 2, true},       // win, ranked, Assassin
	{"n7", "2026-04-29 17:00:00+00", "Bazaar", "BTB:CTF", "Big Team Battle", 3, false},           // loss, BTB
	{"n6", "2026-04-25 21:00:00+00", "Aquarius", "Arena:Slayer", "Ranked Slayer", 3, true},       // loss, ranked
	{"n5", "2026-04-20 14:00:00+00", "Catalyst", "Fiesta:Slayer", "Fiesta", 2, false},            // win, Fiesta
	{"n4", "2026-04-15 12:00:00+00", "Aquarius", "Arena:Slayer", "Ranked Slayer", 1, true},       // draw, ranked
	{"n3", "2026-04-10 19:00:00+00", "Catalyst", "BTB:Strongholds", "Big Team Battle", 2, false}, // win, BTB
	{"n2", "2026-04-05 16:00:00+00", "PvE Map", "Firefight:Solo", "Firefight Solo", 4, false},    // dnf, PvE
	{"n1", "2026-04-01 10:00:00+00", "Aquarius", "Arena:Slayer", "Ranked Slayer", 2, true},       // win, ranked
}

// newTestPlayerDBForNeighborsScenario : PlayerDB avec dataset neighborsTestDataset.
func newTestPlayerDBForNeighborsScenario(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	meta := openMemDB(t)

	// Topologie post-ADR 0016 : shared porte le schéma `shared.*` (lu via
	// SharedReader), player n'a pas d'ATTACH. Le seedPlayerSchema simulait
	// l'ancien ATTACH — supprimé.
	seedPlayerSchema(t, player)
	seedSharedDBSchema(t, shared)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()

	// Wipe les rows par défaut du seed shared (m1 unique avec pTestXUID).
	for _, q := range []string{
		`DELETE FROM shared.match_participants`,
		`DELETE FROM shared.match_registry`,
	} {
		if _, err := shared.Exec(ctx, q); err != nil {
			t.Fatalf("wipe defaults: %v\nSQL: %s", err, q)
		}
	}

	// Insertion du dataset sur la conn shared (= où SharedReader lira).
	for _, m := range neighborsTestDataset {
		_, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry
				(match_id, start_time, start_time_utc, map_name, pair_name, playlist_name, is_ranked)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.id, m.startUTC, m.startUTC, m.mapName, m.pairName, m.playlist, m.isRanked,
		)
		if err != nil {
			t.Fatalf("insert match %s: %v", m.id, err)
		}
		_, err = shared.Exec(ctx,
			`INSERT INTO shared.match_participants
				(match_id, xuid, gamertag, outcome, team_id)
				VALUES (?, ?, ?, ?, 0)`,
			m.id, pTestXUID, pTestGamertag, m.outcome,
		)
		if err != nil {
			t.Fatalf("insert participant %s: %v", m.id, err)
		}
	}

	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		Metadata:     meta,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// timePtr : helper local pour les pointeurs de spec. (strPtr est défini dans
// home_repo_cache_challenges_roundtrip_test.go — même package, sans build tag —
// donc disponible aussi en build `integration` ; le redéclarer ici cassait le
// build de test sous `-tags=integration`.)
func timePtr(s string) *time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return &t
}

// assertNeighbors : helper d'assertion pour comparer prev/next/index/total.
func assertNeighbors(t *testing.T, got *domain.MatchNeighbors, wantPrev, wantNext *string, wantIdx, wantTotal int) {
	t.Helper()
	if got == nil {
		t.Fatalf("neighbors nil")
	}
	if !ptrStringEq(got.PreviousMatchID, wantPrev) {
		t.Errorf("PreviousMatchID = %v, want %v", deref(got.PreviousMatchID), deref(wantPrev))
	}
	if !ptrStringEq(got.NextMatchID, wantNext) {
		t.Errorf("NextMatchID = %v, want %v", deref(got.NextMatchID), deref(wantNext))
	}
	if got.CurrentIndex != wantIdx {
		t.Errorf("CurrentIndex = %d, want %d", got.CurrentIndex, wantIdx)
	}
	if got.TotalMatches != wantTotal {
		t.Errorf("TotalMatches = %d, want %d", got.TotalMatches, wantTotal)
	}
}

func ptrStringEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestMatchViewRepo_GetMatchNeighborsFiltered_NilSpec_GlobalChronology
// vérifie que sans spec on retombe sur Q25 global (8 matchs).
func TestMatchViewRepo_GetMatchNeighborsFiltered_NilSpec_GlobalChronology(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	// n4 est au milieu (idx 4 sur 8 trié DESC) : prev = n3, next = n5
	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n4", nil)
	if err != nil {
		t.Fatalf("GetMatchNeighborsFiltered: %v", err)
	}
	assertNeighbors(t, got, strPtr("n3"), strPtr("n5"), 4, 8)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_FilterPlaylist
// "Ranked Slayer" → 4 matchs : n8, n6, n4, n1.
func TestMatchViewRepo_GetMatchNeighborsFiltered_FilterPlaylist(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Slayer"}}

	// n6 est au milieu de la sous-liste DESC [n8, n6, n4, n1] → idx 1
	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n6", spec)
	if err != nil {
		t.Fatalf("filter playlist: %v", err)
	}
	assertNeighbors(t, got, strPtr("n4"), strPtr("n8"), 1, 4)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_MultiPlaylist (Phase 3)
// "Ranked Slayer" ∪ "Big Team Battle" → 6 matchs : n8, n7, n6, n4, n3, n1
// (n5=Fiesta et n2=Firefight Solo exclus). Valide la clause IN (?, ?).
func TestMatchViewRepo_GetMatchNeighborsFiltered_MultiPlaylist(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{
		PlaylistNames: []string{"Ranked Slayer", "Big Team Battle"},
	}

	// Sous-liste DESC [n8, n7, n6, n4, n3, n1] : n6 idx=2 → prev=n4, next=n7
	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n6", spec)
	if err != nil {
		t.Fatalf("filter multi-playlist: %v", err)
	}
	assertNeighbors(t, got, strPtr("n4"), strPtr("n7"), 2, 6)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_FilterModeCategory_BTB
// "BTB" → préfixe BTB:* → 2 matchs : n7 (BTB:CTF), n3 (BTB:Strongholds).
func TestMatchViewRepo_GetMatchNeighborsFiltered_FilterModeCategory_BTB(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{ModeCategories: []string{"BTB"}}

	// n7 en tête (DESC) → next=nil, prev=n3
	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n7", spec)
	if err != nil {
		t.Fatalf("filter BTB: %v", err)
	}
	assertNeighbors(t, got, strPtr("n3"), nil, 0, 2)

	// n3 en queue → prev=nil, next=n7
	got2, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n3", spec)
	if err != nil {
		t.Fatalf("filter BTB n3: %v", err)
	}
	assertNeighbors(t, got2, nil, strPtr("n7"), 1, 2)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_FilterDateRange
// 2026-04-10 → 2026-04-25 inclus → 4 matchs : n6, n5, n4, n3.
func TestMatchViewRepo_GetMatchNeighborsFiltered_FilterDateRange(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{
		DateFrom: timePtr("2026-04-10T00:00:00Z"),
		DateTo:   timePtr("2026-04-25T23:59:59Z"),
	}

	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n5", spec)
	if err != nil {
		t.Fatalf("filter dates: %v", err)
	}
	// Liste DESC dans [n6, n5, n4, n3] : n5 idx=1, prev=n4, next=n6
	assertNeighbors(t, got, strPtr("n4"), strPtr("n6"), 1, 4)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_FilterOutcome_Win
// outcome=win (code 2) → 4 matchs : n8, n5, n3, n1.
func TestMatchViewRepo_GetMatchNeighborsFiltered_FilterOutcome_Win(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{Outcome: strPtr("win")}

	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n5", spec)
	if err != nil {
		t.Fatalf("filter outcome win: %v", err)
	}
	// DESC [n8, n5, n3, n1] : n5 idx=1, prev=n3, next=n8
	assertNeighbors(t, got, strPtr("n3"), strPtr("n8"), 1, 4)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_Combined
// playlist=Ranked Slayer + outcome=win → 3 matchs : n8, n4(draw exclu), n1.
// Wait : n4 est draw (1), donc outcome=win exclut → reste n8, n1.
// (n6=loss exclu).
func TestMatchViewRepo_GetMatchNeighborsFiltered_Combined(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	spec := &domain.MatchFilterSpec{
		PlaylistNames: []string{"Ranked Slayer"},
		Outcome:       strPtr("win"),
	}

	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n8", spec)
	if err != nil {
		t.Fatalf("combined: %v", err)
	}
	// DESC [n8, n1] : n8 idx=0 (en tête) → next=nil, prev=n1
	assertNeighbors(t, got, strPtr("n1"), nil, 0, 2)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_MatchOutOfScope
// si le matchId courant n'est pas dans le scope filtré → MatchNeighbors zero.
func TestMatchViewRepo_GetMatchNeighborsFiltered_MatchOutOfScope(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	// n2 est PvE Firefight ; on filtre par BTB → n2 n'y est pas
	spec := &domain.MatchFilterSpec{ModeCategories: []string{"BTB"}}
	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n2", spec)
	if err != nil {
		t.Fatalf("out of scope: %v", err)
	}
	if got.TotalMatches != 0 {
		t.Errorf("hors scope : TotalMatches = %d, want 0 (zero value)", got.TotalMatches)
	}
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_EmptySpec_DelegatesToGlobal
// spec vide doit donner le même résultat que nil.
func TestMatchViewRepo_GetMatchNeighborsFiltered_EmptySpec_DelegatesToGlobal(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())

	got, err := repo.GetMatchNeighborsFiltered(context.Background(), pTestXUID, "n4", &domain.MatchFilterSpec{})
	if err != nil {
		t.Fatalf("empty spec: %v", err)
	}
	assertNeighbors(t, got, strPtr("n3"), strPtr("n5"), 4, 8)
}

// TestMatchViewRepo_GetMatchNeighborsFiltered_FilterWithPlayer
// (Phase 2c) : restreint aux matchs où un coéquipier (XUID 99) était présent.
// On insère le coéquipier dans n8, n6, n3 → sous-liste DESC [n8, n6, n3].
//
// Vérifie aussi le cas combiné with_player + outcome=loss (n6 seul match
// perdu où le coéquipier était présent).
func TestMatchViewRepo_GetMatchNeighborsFiltered_FilterWithPlayer(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t)
	ctx := context.Background()

	// Seed : insère un coéquipier (XUID "99") dans 3 matchs de pTestXUID.
	// Le JOIN Q25 ne sélectionne que les rows mp.xuid=pTestXUID, donc ces
	// participations supplémentaires n'affectent pas les autres tests qui
	// utilisent la même factory — elles ne sont visibles que via la clause
	// EXISTS (mp2) ajoutée par WithPlayerXuid.
	const teammateXUID = "99"
	const teammateGT = "CoolMate"
	for _, mid := range []string{"n8", "n6", "n3"} {
		_, err := pdb.Shared.Exec(ctx,
			`INSERT INTO shared.match_participants
				(match_id, xuid, gamertag, outcome, team_id)
				VALUES (?, ?, ?, 0, 0)`,
			mid, teammateXUID, teammateGT,
		)
		if err != nil {
			t.Fatalf("seed teammate %s: %v", mid, err)
		}
	}

	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy())
	xuid := teammateXUID
	spec := &domain.MatchFilterSpec{WithPlayerXuid: &xuid}

	// Sous-liste DESC [n8, n6, n3] :
	//   n6 idx=1 → prev=n3, next=n8
	got, err := repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "n6", spec)
	if err != nil {
		t.Fatalf("filter with_player n6: %v", err)
	}
	assertNeighbors(t, got, strPtr("n3"), strPtr("n8"), 1, 3)

	// n8 en tête → next=nil, prev=n6
	got2, err := repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "n8", spec)
	if err != nil {
		t.Fatalf("filter with_player n8: %v", err)
	}
	assertNeighbors(t, got2, strPtr("n6"), nil, 0, 3)

	// Combiné with_player + outcome=loss : n6 = loss avec coéquipier (n7=loss
	// est exclu car coéquipier absent). Reste 1 seul match dans le scope.
	specCombined := &domain.MatchFilterSpec{
		WithPlayerXuid: &xuid,
		Outcome:        strPtr("loss"),
	}
	got3, err := repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "n6", specCombined)
	if err != nil {
		t.Fatalf("combined with_player+outcome: %v", err)
	}
	assertNeighbors(t, got3, nil, nil, 0, 1)

	// Match hors scope (n4 — coéquipier absent) → MatchNeighbors zero.
	got4, err := repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "n4", spec)
	if err != nil {
		t.Fatalf("with_player out of scope: %v", err)
	}
	if got4.TotalMatches != 0 {
		t.Errorf("hors scope with_player : TotalMatches = %d, want 0", got4.TotalMatches)
	}
}
