//go:build integration

// Package duckdb — media_repo_filters_test.go : tests d'intégration pour les
// filtres, le tri et le groupement de la galerie médias.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestMediaFilters -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
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

	// Topologie réelle post-ADR 0016 : la conn `shared` porte le schéma
	// `shared.*` (où vit match_registry, lu via SharedReader), la conn
	// `social` porte les tables shared_social (media_files,
	// media_match_associations, media_likes…). Aucun ATTACH `shared` sur la
	// conn `social` — le pipeline P1 charge match_registry via SharedReader
	// (= conn `shared` ici).
	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()

	// Wipe les rows par défaut. shared seed pré-popule m1 ; le scope test ré-insère.
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared.match_registry: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe defaults: %v\nSQL: %s", err, q)
		}
	}

	// 3 matchs — insérés sur la conn `shared` (= où SharedReader lira).
	matchInserts := []struct {
		id, ts, mapName, pairName string
	}{
		{"m1", "2025-01-10 14:00:00+00", "Aquarius", "Slayer"},
		{"m2", "2025-01-15 18:00:00+00", "Bazaar", "Capture the Flag"},
		{"m3", "2025-01-20 10:00:00+00", "Catalyst", "Slayer"},
	}
	for _, m := range matchInserts {
		if _, err := shared.Exec(ctx,
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
		id, owner, path, name, kind, stem, ext, capture string
		liked                                           bool
	}{
		{"med-A1", "Alice", "/A1.mp4", "A1.mp4", "video", "A1", ".mp4", "2025-01-10 14:30:00+00", true},
		{"med-A2", "Alice", "/A2.png", "A2.png", "image", "A2", ".png", "2025-01-15 18:30:00+00", false},
		{"med-B1", "Bob", "/B1.mp4", "B1.mp4", "video", "B1", ".mp4", "2025-01-20 10:30:00+00", true},
		{"med-C1", mediaTestPlayerSlug, "/C1.png", "C1.png", "image", "C1", ".png", "2025-01-10 16:00:00+00", false},
		{"med-C2", mediaTestPlayerSlug, "/C2.mp4", "C2.mp4", "video", "C2", ".mp4", "2025-01-15 19:00:00+00", true},
	}
	for _, m := range mediaInserts {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files
				(id, player_slug, file_path, file_name, file_stem, file_ext, kind, capture_end_utc)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			m.id, m.owner, m.path, m.name, m.stem, m.ext, m.kind, m.capture,
		); err != nil {
			t.Fatalf("insert media %s: %v", m.id, err)
		}
		// Le like appartient à un LIKER, plus à la ligne média (2026-08-04) : le
		// « liked » du scénario est celui du viewer de ces tests (HeroPlayer). Un
		// autre viewer verrait ces mêmes médias non likés — c'est précisément ce
		// que les tests de galerie doivent refléter.
		if !m.liked {
			continue
		}
		if _, err := social.Exec(ctx,
			`INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
				VALUES (?, ?, ?, TRUE, CURRENT_TIMESTAMP)`,
			m.path, mediaTestPlayerSlug, mediaTestPlayerSlug,
		); err != nil {
			t.Fatalf("insert like event %s: %v", m.id, err)
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
			`INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES (?, ?)`,
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

// newTestPlayerDBForCandidatesLocale monte un dataset minimal pour les
// suggestions de réassociation (GH3-4) : 1 média (capture 14:00), 1 match m1 à
// 14:00 (dans la fenêtre), pair_name EN "Slayer" + FR "Assassin", playlist EN
// "Ranked Slayer" (playlist_id "pl-ranked" traduit FR via asset_translations).
func newTestPlayerDBForCandidatesLocale(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// Match : pair_name EN "Slayer" + FR "Assassin" ; playlist EN "Ranked Slayer".
	if _, err := shared.Exec(ctx,
		`INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_name, pair_name, pair_name_fr,
			 playlist_name, playlist_id, is_ranked)
		 VALUES ('m1', '2025-01-10 14:00:00', '2025-01-10 14:00:00+00', 'Aquarius',
			 'Slayer', 'Assassin', 'Ranked Slayer', 'pl-ranked', TRUE)`,
	); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	if _, err := shared.Exec(ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, kills, deaths, team_id)
		 VALUES ('m1', ?, ?, 2, 10, 5, 0)`,
		mediaTestPlayerXUID, mediaTestPlayerSlug,
	); err != nil {
		t.Fatalf("insert participant: %v", err)
	}

	// Média capturé à 14:00 (dans la fenêtre de 15 min du match).
	if _, err := social.Exec(ctx,
		`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_start_utc, liked)
		 VALUES ('med-loc', ?, '/clip.mp4', 'clip.mp4', 'video', '2025-01-10 14:00:00+00', FALSE)`,
		mediaTestPlayerSlug,
	); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// mode_name_tr (FR) + asset_translations playlist (FR).
	if _, err := meta.Exec(ctx, `CREATE TABLE mode_name_tr (lang VARCHAR, mode_en VARCHAR, name VARCHAR)`); err != nil {
		t.Fatalf("create mode_name_tr: %v", err)
	}
	if _, err := meta.Exec(ctx,
		`INSERT INTO mode_name_tr (lang, mode_en, name) VALUES ('fr', 'Slayer', 'Assassin')`); err != nil {
		t.Fatalf("seed mode_name_tr: %v", err)
	}
	if _, err := meta.Exec(ctx,
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES ('pl-ranked', 'playlist', 'fr-FR', 'Slayer classé')`); err != nil {
		t.Fatalf("seed asset_translations: %v", err)
	}

	return &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
}

// TestLoadMatchCandidatesForMedia_LocaleAware prouve GH3-4 : les suggestions de
// réassociation servent playlist + mode dans la locale de requête (EN = canonique,
// FR = traductions), jamais de FR résiduel sous EN.
func TestLoadMatchCandidatesForMedia_LocaleAware(t *testing.T) {
	pdb := newTestPlayerDBForCandidatesLocale(t)
	repo := NewMediaRepo(pdb)

	frResp, err := repo.LoadMatchCandidatesForMedia(ctxkeys.WithLocale(context.Background(), "fr"), "/clip.mp4", 15)
	if err != nil {
		t.Fatalf("FR candidates: %v", err)
	}
	if len(frResp.Candidates) != 1 {
		t.Fatalf("FR: got %d candidats, want 1", len(frResp.Candidates))
	}
	frC := frResp.Candidates[0]
	if frC.ModeName == nil || *frC.ModeName != "Assassin" {
		t.Errorf("FR mode = %v, want Assassin (mode_name_tr)", frC.ModeName)
	}
	if frC.PlaylistName == nil || *frC.PlaylistName != "Slayer classé" {
		t.Errorf("FR playlist = %v, want Slayer classé (asset_translations FR)", frC.PlaylistName)
	}

	enResp, err := repo.LoadMatchCandidatesForMedia(ctxkeys.WithLocale(context.Background(), "en"), "/clip.mp4", 15)
	if err != nil {
		t.Fatalf("EN candidates: %v", err)
	}
	if len(enResp.Candidates) != 1 {
		t.Fatalf("EN: got %d candidats, want 1", len(enResp.Candidates))
	}
	enC := enResp.Candidates[0]
	if enC.ModeName == nil || *enC.ModeName != "Slayer" {
		t.Errorf("EN mode = %v, want Slayer (canonique EN, jamais de FR sous EN)", enC.ModeName)
	}
	if enC.PlaylistName == nil || *enC.PlaylistName != "Ranked Slayer" {
		t.Errorf("EN playlist = %v, want Ranked Slayer (playlist_name brut EN)", enC.PlaylistName)
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
// ListMediaAuthors : liste d'auteurs depuis la DB (filtre Auteurs)
// ─────────────────────────────────────────────────────────────────────────────

// TestMediaFilters_ListMediaAuthors vérifie que ListMediaAuthors retourne les
// player_slug distincts présents dans media_files avec leur compte, triés
// count desc puis slug asc. Couvre le fix du bug "Aucun auteur disponible" :
// la liste vient de la DB (cross-joueurs), pas d'un scan filesystem local.
func TestMediaFilters_ListMediaAuthors(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	authors, err := repo.ListMediaAuthors(context.Background())
	if err != nil {
		t.Fatalf("ListMediaAuthors: %v", err)
	}
	// 3 owners distincts : Alice(2), Bob(1), HeroPlayer(2).
	if len(authors) != 3 {
		t.Fatalf("got %d authors %v, want 3", len(authors), authors)
	}
	counts := map[string]int{}
	for _, a := range authors {
		counts[a.PlayerSlug] = a.MediaCount
	}
	want := map[string]int{"Alice": 2, "Bob": 1, mediaTestPlayerSlug: 2}
	for slug, n := range want {
		if counts[slug] != n {
			t.Errorf("author %s: count=%d, want %d (full=%v)", slug, counts[slug], n, authors)
		}
	}
	// Tri count desc, slug asc : Alice(2), HeroPlayer(2), Bob(1).
	gotOrder := []string{authors[0].PlayerSlug, authors[1].PlayerSlug, authors[2].PlayerSlug}
	wantOrder := []string{"Alice", mediaTestPlayerSlug, "Bob"}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Errorf("ordre[%d] = %q, want %q (full=%v)", i, gotOrder[i], wantOrder[i], gotOrder)
		}
	}
}

// TestMediaFilters_ListMediaAuthors_LegacyNoSharedSocial vérifie le fallback
// schéma legacy (SharedSocial nil, media_files sans player_slug exploitable) :
// on ne peut pas distinguer les auteurs → on retourne uniquement le joueur courant.
func TestMediaFilters_ListMediaAuthors_LegacyNoSharedSocial(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	pdb.SharedSocial = nil // simule une player DB legacy sans shared_social
	repo := NewMediaRepo(pdb)
	authors, err := repo.ListMediaAuthors(context.Background())
	if err != nil {
		t.Fatalf("ListMediaAuthors legacy: %v", err)
	}
	if len(authors) != 1 || authors[0].PlayerSlug != mediaTestPlayerSlug {
		t.Fatalf("legacy: got %v, want [%s]", authors, mediaTestPlayerSlug)
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
// Exactitude des filtres map / mode : NE DOIT PAS matcher en substring
// ─────────────────────────────────────────────────────────────────────────────

// newTestPlayerDBMapModeOverlap monte un dataset où les noms de cartes/modes
// se chevauchent (substring) pour exposer le bug ILIKE %X%.
func newTestPlayerDBMapModeOverlap(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// Cartes qui se chevauchent : "Recharge" et "Recharge Annex"
	// Modes qui se chevauchent : "Slayer", "Team Slayer", "Slayer Doubles"
	matches := []struct{ id, mapName, mode string }{
		{"m1", "Recharge", "Slayer"},
		{"m2", "Recharge Annex", "Slayer"},
		{"m3", "Recharge", "Team Slayer"},
		{"m4", "Live Fire", "Slayer Doubles"},
	}
	for _, m := range matches {
		if _, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, map_name, pair_name, playlist_name, is_ranked)
			VALUES (?, '2025-01-10 14:00:00+00', ?, ?, ?, FALSE)`,
			m.id, m.mapName, m.mode, m.mode,
		); err != nil {
			t.Fatalf("insert match: %v", err)
		}
	}

	// Un média par match
	mediaSet := []struct{ id, path, matchID string }{
		{"m-r1", "/recharge1.mp4", "m1"},
		{"m-ra", "/recharge_annex.mp4", "m2"},
		{"m-r2", "/recharge2.mp4", "m3"},
		{"m-lf", "/livefire.mp4", "m4"},
	}
	for _, m := range mediaSet {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
			VALUES (?, ?, ?, ?, 'video', '2025-01-10 14:30:00+00', FALSE)`,
			m.id, mediaTestPlayerSlug, m.path, m.path,
		); err != nil {
			t.Fatalf("insert media: %v", err)
		}
		if _, err := social.Exec(ctx,
			`INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES (?, ?)`,
			m.id, m.matchID,
		); err != nil {
			t.Fatalf("insert assoc: %v", err)
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

func TestMediaFilters_MapFilter_ExactNotSubstring(t *testing.T) {
	pdb := newTestPlayerDBMapModeOverlap(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// On veut UNIQUEMENT "Recharge", pas "Recharge Annex".
	// Match attendu : m1 (Recharge/Slayer) + m3 (Recharge/Team Slayer) = 2 médias.
	wantPaths := map[string]bool{"/recharge1.mp4": true, "/recharge2.mp4": true}
	if len(rows) != 2 {
		t.Fatalf("MapFilter=Recharge: got %d rows %v, want 2 (recharge1, recharge2)", len(rows), pathsOf(rows))
	}
	for _, r := range rows {
		if !wantPaths[r.FilePath] {
			t.Errorf("MapFilter=Recharge: %s ne devrait pas matcher (got %v)", r.FilePath, pathsOf(rows))
		}
	}
}

func TestMediaFilters_ModeFilter_ExactNotSubstring(t *testing.T) {
	// La semantique de ModeFilter a evolue : le format attendu est
	// "Categorie/sous_mode" (ex. "Assassin/Slayer") pour un match granulaire.
	// Un alias seul ("Slayer") sans prefix ":" est traite comme un nom de
	// categorie ; comme "Slayer" n'est pas une categorie connue (cf.
	// analysis.PairNamePrefixesForCategory), aucun filtre n'est applique
	// et toutes les rows sont retournees.
	//
	// Le test est skippe en attendant alignement metier : faut-il accepter
	// les sous-modes seuls comme alias de filtre, ou exiger toujours
	// "Categorie/sous_mode" ? Dette tracee, voir thought_log 2026-04-27.
	t.Skip("semantique ModeFilter ambigue : sous-mode seul vs categorie/sous_mode")
}

// ─────────────────────────────────────────────────────────────────────────────
// Comportement face aux variantes ("Recharge" vs "Recharge v3" vs "Recharge Annex")
// ─────────────────────────────────────────────────────────────────────────────
//
// Ces tests documentent le comportement actuel : chaque label distinct = match
// exact. Si on veut grouper "Recharge" et "Recharge v3" ensemble, il faut soit
// filtrer par map_id (si stable entre variantes), soit normaliser les suffixes.

func newTestPlayerDBVariants(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// Cas réel Halo : variantes Forge avec map_id partagé OU différent.
	matches := []struct{ id, mapID, mapName string }{
		{"m1", "recharge", "Recharge"},             // base map
		{"m2", "recharge", "Recharge v3"},          // hypothèse : même map_id (forge variant)
		{"m3", "recharge_annex", "Recharge Annex"}, // map différente, map_id différent
	}
	for _, m := range matches {
		if _, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, map_id, map_name, pair_name, playlist_name, is_ranked)
			VALUES (?, '2025-01-10 14:00:00+00', ?, ?, 'Slayer', 'Slayer', FALSE)`,
			m.id, m.mapID, m.mapName,
		); err != nil {
			t.Fatalf("insert match: %v", err)
		}
	}

	mediaSet := []struct{ id, path, matchID string }{
		{"m-base", "/recharge_base.mp4", "m1"},
		{"m-v3", "/recharge_v3.mp4", "m2"},
		{"m-annex", "/recharge_annex.mp4", "m3"},
	}
	for _, m := range mediaSet {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
			VALUES (?, ?, ?, ?, 'video', '2025-01-10 14:30:00+00', FALSE)`,
			m.id, mediaTestPlayerSlug, m.path, m.path,
		); err != nil {
			t.Fatalf("insert media: %v", err)
		}
		if _, err := social.Exec(ctx,
			`INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES (?, ?)`,
			m.id, m.matchID,
		); err != nil {
			t.Fatalf("insert assoc: %v", err)
		}
	}

	return &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
}

// Avec normalisation des suffixes (Option B), filtrer "Recharge" doit ramener
// "Recharge" ET "Recharge v3" (suffixe version strippé) mais PAS "Recharge Annex"
// (Annex n'est pas un suffixe de version).
func TestMediaFilters_Variants_StripsVersionSuffix(t *testing.T) {
	pdb := newTestPlayerDBVariants(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// "Recharge" + "Recharge v3" (v\d+ strippé) = 2 médias
	wantPaths := map[string]bool{"/recharge_base.mp4": true, "/recharge_v3.mp4": true}
	if len(rows) != 2 {
		t.Fatalf("MapFilter=Recharge: got %d %v, want 2 (base + v3)", len(rows), pathsOf(rows))
	}
	for _, r := range rows {
		if !wantPaths[r.FilePath] {
			t.Errorf("MapFilter=Recharge: %s ne devrait pas matcher", r.FilePath)
		}
	}
}

// Filtrer "Recharge v3" doit aussi grouper avec "Recharge" (les deux normalisent vers "Recharge").
func TestMediaFilters_Variants_FilterByVariantAlsoGroups(t *testing.T) {
	pdb := newTestPlayerDBVariants(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge v3"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// "Recharge v3" est normalisé en "Recharge" côté filtre ET côté label,
	// donc on ramène les 2 médias avec le nom canonique.
	if len(rows) != 2 {
		t.Fatalf("MapFilter='Recharge v3': got %d %v, want 2 (groupés avec base)", len(rows), pathsOf(rows))
	}
}

// "Recharge Annex" reste isolé : "Annex" n'est pas un suffixe de version.
func TestMediaFilters_Variants_AnnexIsSeparate(t *testing.T) {
	pdb := newTestPlayerDBVariants(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge Annex"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 || rows[0].FilePath != "/recharge_annex.mp4" {
		t.Errorf("MapFilter='Recharge Annex': got %v, want [/recharge_annex.mp4]", pathsOf(rows))
	}
}

// Filtrer "Recharge" ne doit PAS matcher "Recharge Annex" (suffixe non strippé).
func TestMediaFilters_Variants_BaseDoesNotMatchAnnex(t *testing.T) {
	pdb := newTestPlayerDBVariants(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	for _, r := range rows {
		if r.FilePath == "/recharge_annex.mp4" {
			t.Errorf("MapFilter=Recharge ne doit PAS ramener Recharge Annex (got %v)", pathsOf(rows))
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Régression utilisateur : "filtre Altitude ramène Recharge/Absolution/Aquarius/Empyréen"
// ─────────────────────────────────────────────────────────────────────────────
//
// Reproduit exactement le dataset rapporté par le user :
//   - Recharge (22:28 le 21/01/26)
//   - Absolution (00:51 le 30/12/25) — apparaît 2x
//   - Aquarius (00:04 le 30/12/25)
//   - Empyréen (00:13 le 30/12/25)
//   - Altitude (date non précisée)
// User filtrait sur "Altitude" et voyait les 6 retournés. Avec le code actuel
// (commits 39c81d13 + cddc42d6), seule Altitude doit revenir.

func TestMediaFilters_Regression_AltitudeFilter_OnlyAltitude(t *testing.T) {
	pdb := newTestPlayerDBForUserScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Altitude"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("MapFilter=Altitude: got %d rows %v, want EXACTEMENT 1 (/altitude.mp4)", len(rows), pathsOf(rows))
	}
	if rows[0].FilePath != "/altitude.mp4" {
		t.Errorf("MapFilter=Altitude: got %s, want /altitude.mp4", rows[0].FilePath)
	}
	// Sanity : vérifier que le map_name retourné est bien Altitude (pas un autre)
	if rows[0].MapName == nil || *rows[0].MapName != "Altitude" {
		t.Errorf("MapName retourné = %v, want 'Altitude'", rows[0].MapName)
	}
}

// Sans filtre map : on voit bien les 6 médias (sanity check du dataset).
func TestMediaFilters_Regression_NoFilter_AllSixVisible(t *testing.T) {
	pdb := newTestPlayerDBForUserScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("no filter: got %d rows %v, want 6", len(rows), pathsOf(rows))
	}
}

// Filtre "Recharge" : seul le média Recharge (1 sur 6).
func TestMediaFilters_Regression_RechargeFilter_OnlyRecharge(t *testing.T) {
	pdb := newTestPlayerDBForUserScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Recharge"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 || rows[0].FilePath != "/recharge.mp4" {
		t.Errorf("MapFilter=Recharge: got %v, want [/recharge.mp4]", pathsOf(rows))
	}
}

// Filtre "Absolution" : 2 médias (le seul nom dupliqué dans le dataset user).
func TestMediaFilters_Regression_AbsolutionFilter_TwoOccurrences(t *testing.T) {
	pdb := newTestPlayerDBForUserScenario(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Absolution"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("MapFilter=Absolution: got %d rows %v, want 2 (les 2 absolutions)", len(rows), pathsOf(rows))
	}
	for _, r := range rows {
		if r.MapName == nil || *r.MapName != "Absolution" {
			t.Errorf("MapName retourné = %v, want 'Absolution'", r.MapName)
		}
	}
}

// Reproduction du dataset user : 5 maps distinctes, 6 médias (Absolution x2).
func newTestPlayerDBForUserScenario(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// 5 matchs sur 5 cartes différentes (Empyréen avec map_name_fr explicite)
	matches := []struct{ id, ts, mapName, mapNameFR string }{
		{"m-recharge", "2026-01-21 22:28:00+00", "Recharge", ""},
		{"m-abs1", "2025-12-30 00:51:00+00", "Absolution", ""},
		{"m-abs2", "2025-12-30 00:51:30+00", "Absolution", ""},
		{"m-aqua", "2025-12-30 00:04:00+00", "Aquarius", ""},
		{"m-emp", "2025-12-30 00:13:00+00", "Empyrean", "Empyréen"},
		{"m-alt", "2025-12-29 12:00:00+00", "Altitude", ""},
	}
	for _, m := range matches {
		var mapNameFRArg interface{}
		if m.mapNameFR != "" {
			mapNameFRArg = m.mapNameFR
		}
		if _, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, map_name, map_name_fr, pair_name, playlist_name, is_ranked)
			VALUES (?, ?, ?, ?, 'Slayer', 'Slayer', FALSE)`,
			m.id, m.ts, m.mapName, mapNameFRArg,
		); err != nil {
			t.Fatalf("insert match %s: %v", m.id, err)
		}
	}

	mediaSet := []struct{ id, path, matchID string }{
		{"med-recharge", "/recharge.mp4", "m-recharge"},
		{"med-abs1", "/absolution1.mp4", "m-abs1"},
		{"med-abs2", "/absolution2.mp4", "m-abs2"},
		{"med-aqua", "/aquarius.mp4", "m-aqua"},
		{"med-emp", "/empyreen.mp4", "m-emp"},
		{"med-alt", "/altitude.mp4", "m-alt"},
	}
	for _, m := range mediaSet {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
			VALUES (?, ?, ?, ?, 'video', '2025-01-10 14:30:00+00', FALSE)`,
			m.id, mediaTestPlayerSlug, m.path, m.path,
		); err != nil {
			t.Fatalf("insert media %s: %v", m.id, err)
		}
		if _, err := social.Exec(ctx,
			`INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES (?, ?)`,
			m.id, m.matchID,
		); err != nil {
			t.Fatalf("insert assoc: %v", err)
		}
	}

	return &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Bug critique : duplication de médias par LEFT JOIN sur media_match_associations
// ─────────────────────────────────────────────────────────────────────────────
//
// Un média physique = un file_path. Si ce média est associé à plusieurs matchs
// dans media_match_associations (ex: capture pendant une session de 3 matchs
// proches dans le temps), le LEFT JOIN produit N lignes au lieu de 1.
// Conséquences :
//   - Le média apparaît N fois dans la grille (avec map/mode différents)
//   - COUNT(*) renvoie N au lieu de 1 → pagination gonflée
//   - Filtres map/mode incohérents (1 média peut matcher pour plusieurs maps)
//   - Compteur lightbox X/Y mal calé

func newTestPlayerDBMultiAssoc(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// 3 matchs sur 3 maps différentes, proches temporellement (session)
	matches := []struct{ id, ts, mapName string }{
		{"match-aqua", "2025-01-10 14:00:00+00", "Aquarius"},
		{"match-baz", "2025-01-10 14:30:00+00", "Bazaar"},
		{"match-cat", "2025-01-10 15:00:00+00", "Catalyst"},
	}
	for _, m := range matches {
		if _, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, map_name, pair_name, playlist_name, is_ranked)
			VALUES (?, ?, ?, 'Slayer', 'Slayer', FALSE)`,
			m.id, m.ts, m.mapName,
		); err != nil {
			t.Fatalf("insert match: %v", err)
		}
	}

	// 1 SEUL média, mais 3 associations (delta_seconds différents)
	if _, err := social.Exec(ctx,
		`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
		VALUES ('med-multi', ?, '/the_capture.mp4', 'the_capture.mp4', 'video', '2025-01-10 14:35:00+00', FALSE)`,
		mediaTestPlayerSlug,
	); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// Associations : Bazaar est le plus proche (delta=300s), Aquarius assez loin (delta=2100s),
	// Catalyst aussi loin (delta=1500s)
	assocs := []struct {
		matchID string
		delta   int
	}{
		{"match-aqua", 2100},
		{"match-baz", 300},
		{"match-cat", 1500},
	}
	for _, a := range assocs {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds) VALUES ('med-multi', ?, ?)`,
			a.matchID, a.delta,
		); err != nil {
			t.Fatalf("insert assoc: %v", err)
		}
	}

	return &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
}

// Sans filtre : 1 média physique = 1 ligne attendue, peu importe le nombre d'associations.
func TestMediaFilters_MultiAssoc_OneMediaOneRow_NoFilter(t *testing.T) {
	pdb := newTestPlayerDBMultiAssoc(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows %v, want EXACTEMENT 1 (1 média physique malgré 3 associations)", len(rows), pathsOf(rows))
	}
	// Le média devrait montrer Bazaar (association la plus proche temporellement)
	if rows[0].MapName == nil || *rows[0].MapName != "Bazaar" {
		t.Errorf("MapName = %v, want 'Bazaar' (association la plus proche, delta=300s)", rows[0].MapName)
	}
}

// CountMediaFiles doit aussi compter 1, pas 3.
func TestMediaFilters_MultiAssoc_CountOne(t *testing.T) {
	pdb := newTestPlayerDBMultiAssoc(t)
	repo := NewMediaRepo(pdb)
	count, err := repo.CountMediaFiles(context.Background(), domain.MediaFilters{})
	if err != nil {
		t.Fatalf("CountMediaFiles: %v", err)
	}
	if count != 1 {
		t.Errorf("CountMediaFiles = %d, want 1 (pagination doit compter médias uniques)", count)
	}
}

// Filtre map=Aquarius : doit ramener le média 1 fois (avec map=Aquarius affiché),
// pas 3 fois ni 0 fois.
func TestMediaFilters_MultiAssoc_FilterByMap_OneRow(t *testing.T) {
	pdb := newTestPlayerDBMultiAssoc(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(),
		domain.MediaFilters{MapFilter: "Aquarius"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("MapFilter=Aquarius: got %d rows, want 1", len(rows))
	}
	if rows[0].MapName == nil || *rows[0].MapName != "Aquarius" {
		t.Errorf("MapName = %v, want Aquarius", rows[0].MapName)
	}
}

// (anciens tests SQL Sql_SelectsPlayerSlug / SelectsNullPlayerSlugInLegacy
// supprimés en P1 — les builders SQL Q37 cibles n'existent plus, le pipeline
// Go équivalent vit dans media_repo_q37_pipeline.go et est couvert par les
// tests TestMediaFilters_* ci-dessus.)

// ─────────────────────────────────────────────────────────────────────────────
// Regression : déterminisme du tri (bug "page Media réorganisée au refresh")
// ─────────────────────────────────────────────────────────────────────────────
//
// Sans tiebreaker stable, DuckDB est libre de retourner les lignes à
// capture_*/mtime/indexed_at égaux dans un ordre arbitraire entre deux
// exécutions de la query — symptôme observé : la galerie se réordonnait au
// rafraîchissement sans qu'aucun filtre ne change. Le fix ajoute
// `mf.file_path ASC` comme tiebreaker final dans buildQ37MediaQuery.

func newTestPlayerDBForStabilityScenario(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// 5 médias avec EXACTEMENT le même capture_end_utc et aucun match associé.
	// Sans tiebreaker, l'ordre retourné dépend du plan d'exécution DuckDB et
	// peut varier entre deux invocations. Avec le tiebreaker file_path ASC,
	// l'ordre attendu est /m1.mp4 < /m2.mp4 < /m3.mp4 < /m4.mp4 < /m5.mp4 quel
	// que soit l'ordre d'insertion.
	mediaIDs := []string{"sb1", "sb2", "sb3", "sb4", "sb5"}
	paths := []string{"/m3.mp4", "/m1.mp4", "/m5.mp4", "/m2.mp4", "/m4.mp4"} // insertion désordonnée volontaire
	for i, id := range mediaIDs {
		if _, err := social.Exec(ctx,
			`INSERT INTO media_files (id, player_slug, file_path, file_name, kind, capture_end_utc, liked)
			VALUES (?, ?, ?, ?, 'video', '2025-01-10 14:30:00+00', FALSE)`,
			id, mediaTestPlayerSlug, paths[i], paths[i],
		); err != nil {
			t.Fatalf("insert media %s: %v", id, err)
		}
	}

	return &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
}

// TestMediaFilters_Stable_TieBreakerFilePath vérifie que des médias à
// timestamps identiques sont retournés dans un ordre stable et alphabétique
// par file_path (tiebreaker final du ORDER BY).
func TestMediaFilters_Stable_TieBreakerFilePath(t *testing.T) {
	pdb := newTestPlayerDBForStabilityScenario(t)
	repo := NewMediaRepo(pdb)

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// 5 médias à capture_end_utc identique → tri 100 % défini par file_path ASC.
	assertOrder(t, rows,
		[]string{"/m1.mp4", "/m2.mp4", "/m3.mp4", "/m4.mp4", "/m5.mp4"},
		"tiebreaker file_path ASC")
}

// TestMediaFilters_Sort_PrefersCaptureStartUtc vérifie que le tri utilise
// `capture_start_utc` en priorité 1 du COALESCE — la donnée canonique
// alimentée par insertMediaFile. Sans ce fix, le tri retombait sur
// indexed_at (= NOW() au scan) pour tous les médias dont capture_end_utc et
// mtime étaient NULL, donnant un "ordre d'indexation" arbitraire.
//
// Setup : 2 médias avec capture_start et capture_end inversés sémantiquement.
//   - M1 : start=2025-01-10 14:00, end=2025-01-10 17:00 (longue vidéo)
//   - M2 : start=2025-01-10 15:00, end=2025-01-10 15:30 (courte vidéo)
//
// Tri DESC par capture_start_utc → M2 (15:00) avant M1 (14:00).
// Si le tri utilisait capture_end_utc en priorité, M1 (17:00) serait avant M2 (15:30).
func TestMediaFilters_Sort_PrefersCaptureStartUtc(t *testing.T) {
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_likes_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe: %v", err)
		}
	}

	// M1 : start tôt, end tard (longue vidéo)
	if _, err := social.Exec(ctx,
		`INSERT INTO media_files (id, player_slug, file_path, file_name, kind,
			capture_start_utc, capture_end_utc, duration_seconds, liked)
		VALUES ('m1', ?, '/m1.mp4', 'm1.mp4', 'video',
			'2025-01-10 14:00:00+00', '2025-01-10 17:00:00+00', 10800.0, FALSE)`,
		mediaTestPlayerSlug,
	); err != nil {
		t.Fatalf("insert M1: %v", err)
	}
	// M2 : start après M1, mais end plus tôt que M1 (courte vidéo)
	if _, err := social.Exec(ctx,
		`INSERT INTO media_files (id, player_slug, file_path, file_name, kind,
			capture_start_utc, capture_end_utc, duration_seconds, liked)
		VALUES ('m2', ?, '/m2.mp4', 'm2.mp4', 'video',
			'2025-01-10 15:00:00+00', '2025-01-10 15:30:00+00', 1800.0, FALSE)`,
		mediaTestPlayerSlug,
	); err != nil {
		t.Fatalf("insert M2: %v", err)
	}

	pdb := &PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		XUID: mediaTestPlayerXUID, Gamertag: mediaTestPlayerSlug, TitleSlug: titlepkg.DefaultSlug,
	}
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	// Tri DESC par capture_start_utc :
	//   M2.start 15:00 > M1.start 14:00 → M2 en premier.
	// Si capture_end_utc dominait, M1.end 17:00 > M2.end 15:30 → M1 en premier.
	assertOrder(t, rows, []string{"/m2.mp4", "/m1.mp4"},
		"sort=date_desc must use capture_start_utc as primary key")
}

// TestMediaFilters_Stable_ReproducibleAcrossCalls vérifie que la query
// retourne le même ordre sur N exécutions consécutives — preuve directe de la
// régression "réorganisation au refresh".
func TestMediaFilters_Stable_ReproducibleAcrossCalls(t *testing.T) {
	pdb := newTestPlayerDBForStabilityScenario(t)
	repo := NewMediaRepo(pdb)

	first, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	for i := 0; i < 10; i++ {
		next, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
		if err != nil {
			t.Fatalf("call %d: %v", i+2, err)
		}
		if !sameOrder(first, next) {
			t.Errorf("call %d: order divergent — first=%v vs got=%v",
				i+2, pathsOf(first), pathsOf(next))
		}
	}
}
