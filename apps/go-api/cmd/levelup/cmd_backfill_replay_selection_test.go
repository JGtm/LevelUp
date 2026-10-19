package main

// cmd_backfill_replay_selection_test.go — la selection de `backfill-replay` : `--since` (borne
// incluse, sans date ecarte et compte), `--match` (complet, court, refus), `--demo` (manifeste
// de test), leurs combinaisons et leurs refus.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
)

const (
	idSelBorne    = "aaaaaaaa-0000-4000-8000-000000000001" // exactement 2025-09-01 00:00:00 UTC
	idSelVeille   = "bbbbbbbb-0000-4000-8000-000000000002" // 2025-08-31 23:59:59 UTC : juste avant
	idSelSansDate = "cccccccc-0000-4000-8000-000000000003" // ni start_time_utc ni start_time
	idSelNaif     = "dddddddd-0000-4000-8000-000000000004" // seul start_time (naif, lu comme UTC) : 2025-09-02
	idSelRecent   = "eeeeeeee-0000-4000-8000-000000000005" // 2026-03-15
)

// registreSelectionDeTest pose un registre a cinq matchs aux dates choisies et un film (manifeste
// a un chunk) par match, sous un depot temporaire.
func registreSelectionDeTest(t *testing.T) (*titlePkg.PathResolver, string) {
	t.Helper()
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	shared := pr.SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR, map_id VARCHAR, is_firefight BOOLEAN,
			start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`,
		`INSERT INTO match_registry VALUES
			('` + idSelBorne + `', 'Recharge', 'm1', FALSE, NULL, TIMESTAMPTZ '2025-09-01 00:00:00+00'),
			('` + idSelVeille + `', 'Recharge', 'm1', FALSE, NULL, TIMESTAMPTZ '2025-08-31 23:59:59+00'),
			('` + idSelSansDate + `', 'Recharge', 'm1', FALSE, NULL, NULL),
			('` + idSelNaif + `', 'Recharge', 'm1', FALSE, TIMESTAMP '2025-09-02 12:00:00', NULL),
			('` + idSelRecent + `', 'Recharge', 'm1', FALSE, NULL, TIMESTAMPTZ '2026-03-15 18:30:00+00')`,
		`CHECKPOINT`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cache := pr.CacheRootDir()
	if err := os.MkdirAll(filepath.Join(cache, "film_manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{idSelBorne, idSelVeille, idSelSansDate, idSelNaif, idSelRecent} {
		short := titlePkg.FilmShortMatchID(id)
		if err := os.WriteFile(filepath.Join(cache, "film_manifests", short+".json"), []byte(`{"chunks":[{}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pr, cache
}

func memesIDs(t *testing.T, got []replayCandidat, want ...string) {
	t.Helper()
	sort.Strings(want)
	g := idsDe(got)
	sort.Strings(g)
	if strings.Join(g, ",") != strings.Join(want, ",") {
		t.Fatalf("candidats %v, attendu %v", g, want)
	}
}

func TestParserDepuisReplay(t *testing.T) {
	if d, err := parserDepuisReplay(""); err != nil || !d.IsZero() {
		t.Fatalf("vide : %v, %v", d, err)
	}
	d, err := parserDepuisReplay(" 2025-09-01 ")
	if err != nil || !d.Equal(time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("valide : %v, %v", d, err)
	}
	for _, mauvais := range []string{"2025-9-1x", "01/09/2025", "2025-13-01", "hier"} {
		if _, err := parserDepuisReplay(mauvais); err == nil {
			t.Errorf("%q : attendu un refus", mauvais)
		}
	}
}

func TestParserOptionsReplay_Selection(t *testing.T) {
	o, err := parserOptionsReplay([]string{"--since", "2025-09-01", "--demo", "--dry-run", "--limit", "3"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.demo || !o.dryRun || o.limit != 3 || !o.depuis.Equal(time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("options mal lues : %+v", o)
	}
	o, err = parserOptionsReplay([]string{"--match", "aaaaaaaa,bbbbbbbb"})
	if err != nil || o.match != "aaaaaaaa,bbbbbbbb" || o.selectionActive() != true {
		t.Fatalf("--match : %+v, %v", o, err)
	}
	if o, _ := parserOptionsReplay(nil); o.selectionActive() {
		t.Error("sans option, la selection doit etre inactive")
	}
}

func TestValiderSelectionReplay(t *testing.T) {
	depuis := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	cas := []struct {
		nom string
		o   replayBackfillOptions
		ok  bool
	}{
		{"aucune option", replayBackfillOptions{}, true},
		{"since seul", replayBackfillOptions{depuis: depuis}, true},
		{"demo seul", replayBackfillOptions{demo: true}, true},
		{"since + demo (union)", replayBackfillOptions{depuis: depuis, demo: true}, true},
		{"match seul", replayBackfillOptions{match: "aaaaaaaa"}, true},
		{"match + demo", replayBackfillOptions{match: "aaaaaaaa", demo: true}, false},
		{"match + since", replayBackfillOptions{match: "aaaaaaaa", depuis: depuis}, false},
		{"match de virgules", replayBackfillOptions{match: " , ,"}, false},
		{"since + one", replayBackfillOptions{depuis: depuis, one: "x"}, false},
		{"demo + purge", replayBackfillOptions{demo: true, purgeNonGeres: true}, false},
		{"since + dry-run + limit + force", replayBackfillOptions{depuis: depuis, dryRun: true, limit: 2, force: true}, true},
		{"since + repair-impoverished", replayBackfillOptions{depuis: depuis, repairImpoverished: true}, true},
	}
	for _, c := range cas {
		if err := validerSelectionReplay(c.o); (err == nil) != c.ok {
			t.Errorf("%s : err = %v, ok attendu = %v", c.nom, err, c.ok)
		}
	}
}

// TestJugerSelection — la borne de date est INCLUSE, l'absence de date ecarte, et une liste
// (--demo) retient quelle que soit la date (union avec --since).
func TestJugerSelection(t *testing.T) {
	depuis := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	sel := selectionReplay{
		depuis: depuis,
		dates: map[string]time.Time{
			"borne":  depuis,
			"veille": depuis.Add(-time.Second),
			"futur":  depuis.AddDate(1, 0, 0),
			"Liste":  depuis.AddDate(-5, 0, 0),
		},
		voulus: map[string]bool{"liste": true, "listesansdate": true},
	}
	for id, want := range map[string]issueSelection{
		"borne":         selectionRetenue,
		"veille":        selectionHorsPerimetre,
		"futur":         selectionRetenue,
		"sans":          selectionSansDate,
		"Liste":         selectionRetenue, // dans la liste (casse ignoree), malgre une date ancienne
		"listesansdate": selectionRetenue, // dans la liste, sans date
	} {
		if got := sel.juger(id); got != want {
			t.Errorf("juger(%q) = %d, attendu %d", id, got, want)
		}
	}
	// Liste seule (pas de borne) : tout le reste est hors perimetre, sans date compris.
	seule := selectionReplay{voulus: map[string]bool{"a": true}}
	if seule.juger("a") != selectionRetenue || seule.juger("b") != selectionHorsPerimetre {
		t.Error("liste seule : a retenu, b hors perimetre")
	}
}

func TestLireDatesCanoniques(t *testing.T) {
	pr, _ := registreSelectionDeTest(t)
	db, err := sql.Open("duckdb", pr.SharedDBPath(titlePkg.DefaultSlug)+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	dates, err := lireDatesCanoniques(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if got := dates[idSelBorne]; !got.Equal(time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("borne : %v", got)
	}
	if got := dates[idSelVeille]; !got.Equal(time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)) {
		t.Errorf("veille : %v", got)
	}
	if got := dates[idSelNaif]; !got.Equal(time.Date(2025, 9, 2, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("start_time naif lu comme UTC : %v", got)
	}
	if _, ok := dates[idSelSansDate]; ok {
		t.Error("un match sans date canonique doit etre absent")
	}
}

func TestReplaysACuire_Since(t *testing.T) {
	pr, cache := registreSelectionDeTest(t)
	o := replayBackfillOptions{
		titleSlug: titlePkg.DefaultSlug,
		depuis:    time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	var report replayBackfillReport
	cands, err := replaysACuire(context.Background(), pr, cache, o, &report)
	if err != nil {
		t.Fatal(err)
	}
	// Borne incluse, start_time naif retenu ; la veille ecartee ; le match sans date ecarte ET compte.
	memesIDs(t, cands, idSelBorne, idSelNaif, idSelRecent)
	if report.horsPerimetre != 1 || report.sansDate != 1 {
		t.Errorf("bilan : hors perimetre %d (attendu 1), sans date %d (attendu 1)", report.horsPerimetre, report.sansDate)
	}
	if len(report.demandesSansFilm) != 0 {
		t.Errorf("aucune liste demandee : %v", report.demandesSansFilm)
	}
}

func TestReplaysACuire_Match(t *testing.T) {
	pr, cache := registreSelectionDeTest(t)
	court := idSelBorne[:8]
	o := replayBackfillOptions{titleSlug: titlePkg.DefaultSlug, match: court + "," + idSelRecent}
	var report replayBackfillReport
	cands, err := replaysACuire(context.Background(), pr, cache, o, &report)
	if err != nil {
		t.Fatal(err)
	}
	memesIDs(t, cands, idSelBorne, idSelRecent) // un prefixe de 8 et un identifiant complet
	if report.horsPerimetre != 3 || report.sansDate != 0 {
		t.Errorf("bilan : hors perimetre %d (attendu 3), sans date %d (attendu 0)", report.horsPerimetre, report.sansDate)
	}

	// Prefixe inconnu, trop court : refus, rien n'est selectionne.
	for _, mauvais := range []string{"99999999", "aaaa"} {
		o.match = mauvais
		if _, err := replaysACuire(context.Background(), pr, cache, o, &replayBackfillReport{}); err == nil {
			t.Errorf("--match %q : attendu un refus", mauvais)
		}
	}

	// Un match du registre dont le film n'est pas en cache est NOMME, pas tu.
	if err := os.Remove(filepath.Join(cache, "film_manifests", titlePkg.FilmShortMatchID(idSelRecent)+".json")); err != nil {
		t.Fatal(err)
	}
	o.match = court + "," + idSelRecent
	report = replayBackfillReport{}
	cands, err = replaysACuire(context.Background(), pr, cache, o, &report)
	if err != nil {
		t.Fatal(err)
	}
	memesIDs(t, cands, idSelBorne)
	if len(report.demandesSansFilm) != 1 || report.demandesSansFilm[0] != idSelRecent {
		t.Errorf("demandes sans film : %v, attendu [%s]", report.demandesSansFilm, idSelRecent)
	}
}

func TestReplaysACuire_DemoLitLeManifeste(t *testing.T) {
	pr, cache := registreSelectionDeTest(t)
	dir := filepath.Join(pr.RepoRoot(), "config", "demo", "JGtm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// La veille (avant --since) et un identifiant sans film : le premier est retenu par la demo
	// quelle que soit sa date, le second est nomme.
	manifeste := `{"version":"1","title_slug":"` + titlePkg.DefaultSlug + `","corpus":{"solo_match_ids":["x"],
		"replay_matches":[{"match_id":"` + idSelVeille + `","mode":"slayer"},{"match_id":"ffffffff-0000-4000-8000-0000000000ff","mode":"ctf"}]}}`
	if err := os.WriteFile(filepath.Join(dir, titlePkg.DefaultSlug+".json"), []byte(manifeste), 0o644); err != nil {
		t.Fatal(err)
	}
	// Un manifeste d'un autre titre ne contribue pas.
	autre := `{"version":"1","title_slug":"halo_5","corpus":{"solo_match_ids":["x"],"replay_matches":[{"match_id":"` + idSelRecent + `","mode":"slayer"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "halo_5.json"), []byte(autre), 0o644); err != nil {
		t.Fatal(err)
	}

	// --demo seul.
	o := replayBackfillOptions{titleSlug: titlePkg.DefaultSlug, demo: true}
	var report replayBackfillReport
	cands, err := replaysACuire(context.Background(), pr, cache, o, &report)
	if err != nil {
		t.Fatal(err)
	}
	memesIDs(t, cands, idSelVeille)
	if len(report.demandesSansFilm) != 1 || report.demandesSansFilm[0] != "ffffffff-0000-4000-8000-0000000000ff" {
		t.Errorf("demandes sans film : %v", report.demandesSansFilm)
	}

	// --since + --demo : union (les films depuis la date, plus ceux de la demo).
	o.depuis = time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	report = replayBackfillReport{}
	cands, err = replaysACuire(context.Background(), pr, cache, o, &report)
	if err != nil {
		t.Fatal(err)
	}
	memesIDs(t, cands, idSelBorne, idSelNaif, idSelRecent, idSelVeille)
	if report.sansDate != 1 || report.horsPerimetre != 0 {
		t.Errorf("bilan union : sans date %d (attendu 1), hors perimetre %d (attendu 0)", report.sansDate, report.horsPerimetre)
	}

	// Titre sans manifeste : refus net (jamais une liste vide qui laisserait tout passer).
	o = replayBackfillOptions{titleSlug: titlePkg.DefaultSlug, demo: true}
	if err := os.Remove(filepath.Join(dir, titlePkg.DefaultSlug+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := replaysACuire(context.Background(), pr, cache, o, &replayBackfillReport{}); err == nil {
		t.Error("--demo sans manifeste du titre : attendu un refus")
	}
}
