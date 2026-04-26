//go:build integration

// Package duckdb — media_repo_filters_test.go : tests d'intégration pour les
// filtres, le tri et le groupement de la galerie médias.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestMediaFilters -v
package duckdb

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
)

const (
	mediaTestPlayerXUID = "xuid_hero"
	mediaTestPlayerSlug = "HeroPlayer"
)

// newTestPlayerDBForMediaScenario crée une PlayerDB avec un dataset contrôlé
// de 3 matchs, 5 médias appartenant à 3 owners (Alice, Bob, HeroPlayer).
//
// Matchs :
//
//	m1 : 2025-01-10 14:00 UTC, Aquarius, Slayer
//	m2 : 2025-01-15 18:00 UTC, Bazaar, Capture the Flag
//	m3 : 2025-01-20 10:00 UTC, Catalyst, Slayer
//
// Médias (capture_end_utc, kind brut DB, owner, liked, match) :
//
//	med-A1  Alice       2025-01-10 14:30  video  liked   -> m1 (Aquarius/Slayer)
//	med-A2  Alice       2025-01-15 18:30  image  -       -> m2 (Bazaar/CTF)
//	med-B1  Bob         2025-01-20 10:30  video  liked   -> m3 (Catalyst/Slayer)
//	med-C1  HeroPlayer  2025-01-10 16:00  image  -       -> m1 (Aquarius/Slayer)
//	med-C2  HeroPlayer  2025-01-15 19:00  video  liked   -> orphelin (pas de match)
func newTestPlayerDBForMediaScenario(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedDBSchema(t, social)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()

	// Wipe les rows par défaut pour partir d'une feuille blanche
	for _, q := range []string{
		`DELETE FROM media_match_associations`,
		`DELETE FROM media_files`,
		`DELETE FROM shared.match_registry`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe defaults: %v\nSQL: %s", err, q)
		}
	}

	// 3 matchs
	matchInserts := []struct {
		id, ts, mapName, pairName string
	}{
		{"m1", "2025-01-10 14:00:00+00", "Aquarius", "Slayer"},
		{"m2", "2025-01-15 18:00:00+00", "Bazaar", "Capture the Flag"},
		{"m3", "2025-01-20 10:00:00+00", "Catalyst", "Slayer"},
	}
	for _, m := range matchInserts {
		if _, err := social.Exec(ctx,
			`INSERT INTO shared.match_registry
				(match_id, start_time, map_name, pair_name, playlist_name, is_ranked)
				VALUES (?, ?, ?, ?, ?, FALSE)`,
			m.id, m.ts, m.mapName, m.pairName, m.pairName,
		); err != nil {
			t.Fatalf("insert match %s: %v", m.id, err)
		}
	}

	// 5 médias
	mediaInserts := []struct {
		id, owner, path, name, kind, capture string
		liked                                bool
	}{
		{"med-A1", "Alice", "/A1.mp4", "A1.mp4", "video", "2025-01-10 14:30:00+00", true},
		{"med-A2", "Alice", "/A2.png", "A2.png", "image", "2025-01-15 18:30:00+00", false},
		{"med-B1", "Bob", "/B1.mp4", "B1.mp4", "video", "2025-01-20 10:30:00+00", true},
		{"med-C1", mediaTestPlayerSlug, "/C1.png", "C1.png", "image", "2025-01-10 16:00:00+00", false},
		{"med-C2", mediaTestPlayerSlug, "/C2.mp4", "C2.mp4", "video", "2025-01-15 19:00:00+00", true},
	}
	for _, m := range mediaInserts {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files
				(id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.id, m.owner, m.path, m.name, m.kind, m.capture, m.liked,
		); err != nil {
			t.Fatalf("insert media %s: %v", m.id, err)
		}
	}

	// Associations matchs (sauf med-C2 qui reste orphelin)
	assocInserts := []struct{ mediaID, matchID string }{
		{"med-A1", "m1"},
		{"med-A2", "m2"},
		{"med-B1", "m3"},
		{"med-C1", "m1"},
	}
	for _, a := range assocInserts {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_match_associations (media_file_id, match_id) VALUES (?, ?)`,
			a.mediaID, a.matchID,
		); err != nil {
			t.Fatalf("insert assoc %s->%s: %v", a.mediaID, a.matchID, err)
		}
	}

	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		SharedSocial: social,
		Metadata:     meta,
		XUID:         mediaTestPlayerXUID,
		Gamertag:     mediaTestPlayerSlug,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// pathsOf extrait les file_path d'une slice de MediaFileRow dans l'ordre reçu.
func pathsOf(rows []domain.MediaFileRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.FilePath
	}
	return out
}

func assertOrder(t *testing.T, got []domain.MediaFileRow, want []string, scenario string) {
	t.Helper()
	gotPaths := pathsOf(got)
	if len(gotPaths) != len(want) {
		t.Fatalf("%s: got %d items %v, want %d %v", scenario, len(gotPaths), gotPaths, len(want), want)
	}
	for i := range want {
		if gotPaths[i] != want[i] {
			t.Errorf("%s: position %d = %q, want %q (full got=%v)", scenario, i, gotPaths[i], want[i], gotPaths)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Sort tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_Sort_DateDesc_Default(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// date_desc : B1 (2025-01-20), C2 (2025-01-15 19), A2 (2025-01-15 18:30), C1 (2025-01-10 16), A1 (2025-01-10 14:30)
	assertOrder(t, rows, []string{"/B1.mp4", "/C2.mp4", "/A2.png", "/C1.png", "/A1.mp4"}, "default sort=date_desc")
}

func TestMediaFilters_Sort_DateAsc(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{Sort: "date_asc"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	assertOrder(t, rows, []string{"/A1.mp4", "/C1.png", "/A2.png", "/C2.mp4", "/B1.mp4"}, "sort=date_asc")
}

func TestMediaFilters_Sort_MapAsc(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{Sort: "map_asc"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// map_asc primary, date_desc secondary :
	//   Aquarius : C1 (16:00), A1 (14:30)
	//   Bazaar   : A2
	//   Catalyst : B1
	//   (no map) : C2 (orphelin → '' tri en tête car COALESCE(map,'') ASC)
	// Ordre attendu : C2 (vide), C1, A1, A2, B1
	assertOrder(t, rows, []string{"/C2.mp4", "/C1.png", "/A1.mp4", "/A2.png", "/B1.mp4"}, "sort=map_asc")
}

func TestMediaFilters_Sort_ModeAsc(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{Sort: "mode_asc"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// mode_asc primary, date_desc secondary :
	//   (vide)           : C2 (orphelin)
	//   Capture the Flag : A2
	//   Slayer           : B1, C1, A1
	assertOrder(t, rows, []string{"/C2.mp4", "/A2.png", "/B1.mp4", "/C1.png", "/A1.mp4"}, "sort=mode_asc")
}

// ─────────────────────────────────────────────────────────────────────────────
// GroupBy tests (le groupement = un préfixe de tri côté backend)
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_GroupBy_Owner(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{GroupBy: "owner"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// player_slug ASC, date_desc :
	//   Alice      : A2 (18:30), A1 (14:30)
	//   Bob        : B1
	//   HeroPlayer : C2 (19:00), C1 (16:00)
	assertOrder(t, rows, []string{"/A2.png", "/A1.mp4", "/B1.mp4", "/C2.mp4", "/C1.png"}, "groupBy=owner")
}

func TestMediaFilters_GroupBy_Map(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{GroupBy: "map"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// COALESCE(map,'~zzz') ASC, date_desc :
	//   Aquarius : C1, A1
	//   Bazaar   : A2
	//   Catalyst : B1
	//   '~zzz'   : C2 (orphelin)
	assertOrder(t, rows, []string{"/C1.png", "/A1.mp4", "/A2.png", "/B1.mp4", "/C2.mp4"}, "groupBy=map")
}

func TestMediaFilters_GroupBy_Mode(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{GroupBy: "mode"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// COALESCE(mode,'~zzz') ASC, date_desc :
	//   Capture the Flag : A2
	//   Slayer           : B1, C1, A1
	//   '~zzz'           : C2 (orphelin)
	assertOrder(t, rows, []string{"/A2.png", "/B1.mp4", "/C1.png", "/A1.mp4", "/C2.mp4"}, "groupBy=mode")
}

func TestMediaFilters_GroupBy_Empty_BehavesLikeDefault(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	withEmpty, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{GroupBy: ""}, 100, 0)
	if err != nil {
		t.Fatalf("empty groupBy: %v", err)
	}
	noFilter, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("no filter: %v", err)
	}
	if !sameOrder(withEmpty, noFilter) {
		t.Errorf("groupBy='' devrait égaler aucun groupBy : got=%v vs want=%v", pathsOf(withEmpty), pathsOf(noFilter))
	}
}

func sameOrder(a, b []domain.MediaFileRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].FilePath != b[i].FilePath {
			return false
		}
	}
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// Filter tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_KindFilter_ScreenshotMatchesImage(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{KindFilter: "screenshot"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// Le frontend envoie "screenshot" mais la DB stocke "image" (nouveau schéma).
	// Le filtre doit accepter les deux conventions.
	want := map[string]bool{"/A2.png": true, "/C1.png": true}
	got := map[string]bool{}
	for _, r := range rows {
		got[r.FilePath] = true
	}
	if len(rows) != 2 {
		t.Fatalf("kind=screenshot: got %d rows %v, want 2 (A2, C1)", len(rows), pathsOf(rows))
	}
	for k := range want {
		if !got[k] {
			t.Errorf("kind=screenshot: %s manquant (got %v)", k, pathsOf(rows))
		}
	}
}

func TestMediaFilters_KindFilter_ClipMatchesVideo(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{KindFilter: "clip"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// Le frontend envoie "clip" mais la DB stocke "video".
	if len(rows) != 3 {
		t.Fatalf("kind=clip: got %d rows %v, want 3 (A1, B1, C2)", len(rows), pathsOf(rows))
	}
}

func TestMediaFilters_LikedOnly(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{LikedOnly: true}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// liked=true : A1, B1, C2. Tri date_desc : B1 (20), C2 (15-19), A1 (10)
	assertOrder(t, rows, []string{"/B1.mp4", "/C2.mp4", "/A1.mp4"}, "likedOnly")
}

func TestMediaFilters_MapFilter(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{MapFilter: "Bazaar"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 || rows[0].FilePath != "/A2.png" {
		t.Fatalf("mapFilter=Bazaar: got %v, want [/A2.png]", pathsOf(rows))
	}
}

func TestMediaFilters_AuthorSlugs_SingleAuthor(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{AuthorSlugs: []string{"Alice"}}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	assertOrder(t, rows, []string{"/A2.png", "/A1.mp4"}, "authorSlugs=[Alice]")
}

func TestMediaFilters_AuthorSlugs_MultipleAuthors(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{AuthorSlugs: []string{"Alice", "Bob"}}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// Alice + Bob, date_desc : B1, A2, A1
	assertOrder(t, rows, []string{"/B1.mp4", "/A2.png", "/A1.mp4"}, "authorSlugs=[Alice,Bob]")
}

func TestMediaFilters_SectionFilter_Mine(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{SectionFilter: "mine"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// HeroPlayer = C1, C2. date_desc : C2, C1
	assertOrder(t, rows, []string{"/C2.mp4", "/C1.png"}, "section=mine")
}

func TestMediaFilters_SectionFilter_Teammate(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{SectionFilter: "teammate"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// !HeroPlayer : Alice + Bob. date_desc : B1, A2, A1
	assertOrder(t, rows, []string{"/B1.mp4", "/A2.png", "/A1.mp4"}, "section=teammate")
}

// ─────────────────────────────────────────────────────────────────────────────
// PlayerSlug exposé : critique pour le groupement "by owner" frontend
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_OwnerGamertag_PopulatedFromPlayerSlug(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	// Chaque row doit exposer son owner (player_slug == gamertag dans ce projet).
	expected := map[string]string{
		"/A1.mp4": "Alice",
		"/A2.png": "Alice",
		"/B1.mp4": "Bob",
		"/C1.png": mediaTestPlayerSlug,
		"/C2.mp4": mediaTestPlayerSlug,
	}
	for _, r := range rows {
		want := expected[r.FilePath]
		if r.PlayerSlug == nil {
			t.Errorf("%s: PlayerSlug nil, want %q", r.FilePath, want)
			continue
		}
		if *r.PlayerSlug != want {
			t.Errorf("%s: PlayerSlug=%q, want %q", r.FilePath, *r.PlayerSlug, want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Combos
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_Combo_GroupByMap_LikedOnly(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{GroupBy: "map", LikedOnly: true}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// liked=true (A1, B1, C2), groupé par map asc puis date_desc :
	//   Aquarius : A1
	//   Catalyst : B1
	//   '~zzz'   : C2 (orphelin)
	assertOrder(t, rows, []string{"/A1.mp4", "/B1.mp4", "/C2.mp4"}, "groupBy=map+likedOnly")
}

func TestMediaFilters_Combo_KindFilter_GroupByOwner(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{KindFilter: "clip", GroupBy: "owner"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// kind=clip (≡ video) + groupBy owner ASC + date_desc :
	//   Alice      : A1
	//   Bob        : B1
	//   HeroPlayer : C2
	assertOrder(t, rows, []string{"/A1.mp4", "/B1.mp4", "/C2.mp4"}, "kind=clip + groupBy=owner")
}

// ─────────────────────────────────────────────────────────────────────────────
// CountMediaFiles
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_Count_RespectsFilters(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	cases := []struct {
		name    string
		filters domain.MediaFilters
		want    int
	}{
		{"no filters", domain.MediaFilters{}, 5},
		{"liked only", domain.MediaFilters{LikedOnly: true}, 3},
		{"section mine", domain.MediaFilters{SectionFilter: "mine"}, 2},
		{"section teammate", domain.MediaFilters{SectionFilter: "teammate"}, 3},
		{"map=Bazaar", domain.MediaFilters{MapFilter: "Bazaar"}, 1},
		{"author Alice", domain.MediaFilters{AuthorSlugs: []string{"Alice"}}, 2},
		{"kind=screenshot (≡image)", domain.MediaFilters{KindFilter: "screenshot"}, 2},
		{"kind=clip (≡video)", domain.MediaFilters{KindFilter: "clip"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.CountMediaFiles(context.Background(), tc.filters)
			if err != nil {
				t.Fatalf("CountMediaFiles: %v", err)
			}
			if got != tc.want {
				t.Errorf("count = %d, want %d", got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SQL generated (sanity check sur les requêtes assemblées)
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaFilters_Sql_SelectsPlayerSlug(t *testing.T) {
	q, _ := buildQ37MediaQuery(domain.MediaFilters{}, 24, 0, mediaQueryConfig{playerSlug: mediaTestPlayerSlug})
	if !strings.Contains(q, "mf.player_slug AS player_slug") {
		t.Errorf("expected mf.player_slug in SELECT for shared social schema, got: %s", q)
	}
}

func TestMediaFilters_Sql_SelectsNullPlayerSlugInLegacy(t *testing.T) {
	q, _ := buildQ37MediaQuery(domain.MediaFilters{}, 24, 0, mediaQueryConfig{})
	if !strings.Contains(q, "NULL AS player_slug") {
		t.Errorf("expected NULL AS player_slug for legacy schema, got: %s", q)
	}
}
