package ops

// seed_demo_replay_vehicles_test.go — la famille de rejeu « véhicules » de la démo : la lecture
// de la règle en base partagée (loadDemoReplayCandidates, avec la vraie table append-only et sa
// vue `_latest`), le choix (pickDemoReplays) et l'émission additive (EmitDemoReplayPicks).

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const vehiclesTestXUID = "2533274000000001"

// vehiclesTestID : un match_id au format UUID dont les 8 premiers caractères (la forme courte
// du film) sont uniques par n.
func vehiclesTestID(n int) string { return fmt.Sprintf("%08x-0000-4000-8000-000000000000", n) }

// vehiclesFixture construit la base partagée source et le cache de films ; chaque méthode
// ajoute un match, avec son film au cache.
type vehiclesFixture struct {
	t    *testing.T
	root string
	db   *sql.DB
	path string
}

func newVehiclesFixture(t *testing.T) *vehiclesFixture {
	t.Helper()
	root := t.TempDir()
	f := &vehiclesFixture{t: t, root: root, path: filepath.Join(root, "shared.duckdb")}
	db, err := sql.Open("duckdb", f.path)
	if err != nil {
		t.Fatal(err)
	}
	f.db = db
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, game_variant_name VARCHAR, map_name VARCHAR,
			start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, is_firefight BOOLEAN)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER,
			present_at_beginning BOOLEAN)`,
	} {
		f.exec(q)
	}
	if err := migration.ExecScript(db, migration.MatchVehicleTakesTableSQL("match_vehicle_takes")+
		migration.MatchVehicleTakesLatestViewSQL("match_vehicle_takes")); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *vehiclesFixture) exec(q string) {
	f.t.Helper()
	if _, err := f.db.Exec(q); err != nil {
		f.t.Fatalf("fixture : %v\n%s", err, q)
	}
}

// match ajoute un match du joueur source à la date (jour d'octobre 2026) donnée, avec son film.
// slots décrit les participants : « équipe:présence » (présence T, F ou N pour NULL), le
// premier est le joueur source.
func (f *vehiclesFixture) match(n int, variant string, day int, slots ...string) string {
	f.t.Helper()
	id := vehiclesTestID(n)
	ts := fmt.Sprintf("2026-10-%02d 10:00:00", day)
	f.exec(fmt.Sprintf(`INSERT INTO match_registry VALUES ('%s', '%s', 'Carte%d', TIMESTAMP '%s', TIMESTAMP '%s', FALSE)`,
		id, variant, n, ts, ts))
	for i, s := range slots {
		team, present, _ := strings.Cut(s, ":")
		pres := map[string]string{"T": "TRUE", "F": "FALSE", "N": "NULL"}[present]
		xuid := fmt.Sprintf("%d%02d", 2533274000000000+int64(n)*100, i)
		if i == 0 {
			xuid = vehiclesTestXUID
		}
		f.exec(fmt.Sprintf(`INSERT INTO match_participants VALUES ('%s', '%s', %s, %s)`, id, xuid, team, pres))
	}
	manifests := filepath.Join(titlePkg.NewPathResolver(f.root).CacheRootDir(), "film_manifests")
	if err := os.MkdirAll(manifests, 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifests, id[:8]+".json"), []byte("{}"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return id
}

// pass écrit une passe de la ressource véhicules : une ligne `match` et une ligne `take` par
// famille (« famille=montées »). Les passes se classent par `written_at`.
func (f *vehiclesFixture) pass(id, decodePass, writtenAt string, takes ...string) {
	f.t.Helper()
	row := func(kind, family string, n int) {
		f.exec(fmt.Sprintf(`INSERT INTO match_vehicle_takes (match_id, decode_pass, written_at, row_kind, camp, xuid,
			family, takes, aboard_ms, episodes, proximity_episodes, frags, measured, unmeasured_reason, doc_schema,
			episodes_read, episodes_unnamed, episodes_no_camp, frags_read, frags_reason, frags_total, frags_unmatched)
			VALUES ('%s', '%s', TIMESTAMP '%s', '%s', 0, '%s', '%s', %d, 0, 0, 0, 0, TRUE, '', 100, 0, 0, 0, TRUE, '', 0, 0)`,
			id, decodePass, writtenAt, kind, vehiclesTestXUID, family, n))
	}
	row("match", "", 0)
	for _, tk := range takes {
		fam, n, _ := strings.Cut(tk, "=")
		var count int
		if _, err := fmt.Sscanf(n, "%d", &count); err != nil {
			f.t.Fatal(err)
		}
		row("take", fam, count)
	}
}

func (f *vehiclesFixture) opts() SeedDemoOptions {
	f.t.Helper()
	f.exec(`CHECKPOINT`)
	if err := f.db.Close(); err != nil {
		f.t.Fatal(err)
	}
	for _, name := range []string{"stats.duckdb", "metadata.duckdb"} {
		if err := os.WriteFile(filepath.Join(f.root, name), nil, 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	return SeedDemoOptions{
		RepoRoot: f.root, TitleSlug: titlePkg.DefaultSlug, SourceSharedDB: f.path, SourceXUID: vehiclesTestXUID,
		SourcePlayerDB: filepath.Join(f.root, "stats.duckdb"), SourceMetaDB: filepath.Join(f.root, "metadata.duckdb"),
		OutDir: filepath.Join(f.root, "out"),
	}
}

var (
	quatreContreQuatre = []string{"0:T", "0:T", "0:T", "0:T", "1:T", "1:T", "1:T", "1:T"}
)

func TestLoadDemoReplayCandidates_LitLaRegleVehicules(t *testing.T) {
	f := newVehiclesFixture(t)
	// 4v4 à deux familles ; une passe PLUS ANCIENNE en portait trois (la vue rend la dernière).
	a := f.match(1, "CTF:Arena", 1, quatreContreQuatre...)
	f.pass(a, "p-old", "2026-10-01 00:00:00", "scorpion=1", "warthog=1", "banshee=1")
	f.pass(a, "p-new", "2026-10-02 00:00:00", "warthog=3", "banshee=2")
	// 4v4 à une seule famille, plus la famille inconnue qui ne compte pas.
	b := f.match(2, "Slayer:Arena", 2, quatreContreQuatre...)
	f.pass(b, "p", "2026-10-02 00:00:00", "warthog=9", "unknown=4")
	// 3v3 et 5v5 au coup d'envoi : deux familles, mais pas un 4v4.
	c := f.match(3, "Slayer:Arena", 3, "0:T", "0:T", "0:T", "1:T", "1:T", "1:T")
	f.pass(c, "p", "2026-10-02 00:00:00", "warthog=3", "banshee=2")
	d := f.match(4, "Slayer:Arena", 4, "0:T", "0:T", "0:T", "0:T", "0:T", "1:T", "1:T", "1:T", "1:T", "1:T")
	f.pass(d, "p", "2026-10-02 00:00:00", "warthog=3", "banshee=2")
	// 4v4 au départ avec deux remplaçants tardifs : 10 lignes, 8 joueurs au coup d'envoi.
	e := f.match(5, "Slayer:Arena", 5, append([]string{"0:F", "1:F"}, quatreContreQuatre...)...)
	f.pass(e, "p", "2026-10-02 00:00:00", "warthog=1", "banshee=1")
	// drapeau de présence jamais renseigné : repli sur tous les participants.
	g := f.match(6, "Slayer:Arena", 6, "0:N", "0:N", "0:N", "0:N", "1:N", "1:N", "1:N", "1:N")
	f.pass(g, "p", "2026-10-02 00:00:00", "warthog=1", "banshee=1")
	// 8 joueurs en trois camps (4, 2, 2) : pas un 4 contre 4.
	h := f.match(7, "Slayer:Arena", 7, "0:T", "0:T", "0:T", "0:T", "1:T", "1:T", "2:T", "2:T")
	f.pass(h, "p", "2026-10-02 00:00:00", "warthog=1", "banshee=1")
	// 4v4 sans aucune prise : ni types ni montées.
	i := f.match(8, "Slayer:Arena", 8, quatreContreQuatre...)

	cands, err := loadDemoReplayCandidates(context.Background(), f.opts())
	if err != nil {
		t.Fatalf("loadDemoReplayCandidates : %v", err)
	}
	type lecture struct {
		fourVsFour   bool
		types, takes int64
	}
	want := map[string]lecture{
		a: {true, 2, 5}, b: {true, 1, 9}, c: {false, 2, 5}, d: {false, 2, 5},
		e: {true, 2, 2}, g: {true, 2, 2}, h: {false, 2, 2}, i: {false, 0, 0}, // i : aucune prise, 4 contre 4 non établi
	}
	if len(cands) != len(want) {
		t.Fatalf("%d candidats, attendu %d : %+v", len(cands), len(want), cands)
	}
	for _, c := range cands {
		w, ok := want[c.matchID]
		if !ok {
			t.Errorf("candidat inattendu %s", c.matchID)
			continue
		}
		if got := (lecture{c.fourVsFour, c.vehicleTypes, c.vehicleTakes}); got != w {
			t.Errorf("%s : lu %+v, attendu %+v", c.matchID, got, w)
		}
	}
	picks, _ := pickDemoReplays(cands)
	var veh []DemoReplayPick
	for _, p := range picks {
		if p.Mode == demoModeVehicles {
			veh = append(veh, p)
		}
	}
	// a (5 montées) devance e et g (2) ; b (une famille), c, d, h (pas 4v4) et i (aucune prise) sont écartés.
	if len(veh) != 1 || veh[0].MatchID != a {
		t.Fatalf("famille véhicules = %+v, attendu le seul match %s", veh, a)
	}
}

func TestPickDemoReplays_FamilleVehicules(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	veh := func(id string, takes int64, start time.Time) demoReplayCandidate {
		return demoReplayCandidate{matchID: id, variant: "Slayer:Arena", mapName: "Carte", start: start,
			fourVsFour: true, vehicleTypes: 2, vehicleTakes: takes}
	}
	pickOf := func(cands ...demoReplayCandidate) (DemoReplayPick, bool) {
		picks, _ := pickDemoReplays(cands)
		for _, p := range picks {
			if p.Mode == demoModeVehicles {
				return p, true
			}
		}
		return DemoReplayPick{}, false
	}

	t.Run("candidat retenu, aussi meilleur de son mode", func(t *testing.T) {
		picks, _ := pickDemoReplays([]demoReplayCandidate{veh("m1", 4, t0)})
		if len(picks) != 2 || picks[0].Mode != "slayer" || picks[1].Mode != demoModeVehicles ||
			picks[0].MatchID != "m1" || picks[1].MatchID != "m1" || picks[1].Map != "Carte" {
			t.Fatalf("les deux entrées coexistent, véhicules en dernier : %+v", picks)
		}
	})
	t.Run("un seul type de véhicule est écarté", func(t *testing.T) {
		one := veh("m1", 9, t0)
		one.vehicleTypes = 1
		if p, ok := pickOf(one); ok {
			t.Errorf("retenu à tort : %+v", p)
		}
	})
	t.Run("ni 6 ni 10 joueurs", func(t *testing.T) {
		not := veh("m1", 9, t0)
		not.fourVsFour = false
		if p, ok := pickOf(not); ok {
			t.Errorf("retenu à tort : %+v", p)
		}
	})
	t.Run("variante sans famille de mode : retenue quand même", func(t *testing.T) {
		c := veh("m1", 4, t0)
		c.variant = "CASTLE WARS"
		picks, unclassified := pickDemoReplays([]demoReplayCandidate{c})
		if len(picks) != 1 || picks[0].Mode != demoModeVehicles || len(unclassified) != 1 {
			t.Errorf("picks=%+v unclassified=%v", picks, unclassified)
		}
	})
	t.Run("départage : montées, puis date, puis match_id", func(t *testing.T) {
		cases := []struct {
			name string
			cs   []demoReplayCandidate
			want string
		}{
			{"plus de montées l'emporte sur la date", []demoReplayCandidate{veh("a", 3, t0.Add(48*time.Hour)), veh("b", 5, t0)}, "b"},
			{"à montées égales, le plus récent", []demoReplayCandidate{veh("a", 5, t0), veh("b", 5, t0.Add(time.Hour))}, "b"},
			{"à tout égal, le plus petit match_id", []demoReplayCandidate{veh("b", 5, t0), veh("a", 5, t0)}, "a"},
		}
		for _, tc := range cases {
			if p, ok := pickOf(tc.cs...); !ok || p.MatchID != tc.want {
				t.Errorf("%s : %+v, attendu %s", tc.name, p, tc.want)
			}
		}
	})
	t.Run("l'artefact à jour n'avantage pas la famille véhicules", func(t *testing.T) {
		a, b := veh("a", 3, t0), veh("b", 5, t0)
		a.upToDate = true
		if p, _ := pickOf(a, b); p.MatchID != "b" {
			t.Errorf("choix %+v, attendu b", p)
		}
	})
}

func TestMergeDemoReplayPicks_Additive(t *testing.T) {
	existing := []DemoReplayPick{
		{MatchID: "old-ctf", Mode: "ctf", Map: "A"},
		{MatchID: "legacy", Mode: "capture_the_moon", Map: "Z"},
		{MatchID: "old-fiesta", Mode: "fiesta"},
	}
	fresh := []DemoReplayPick{
		{MatchID: "new-fiesta", Mode: "fiesta"}, // plus récent, mais la famille est déjà figée
		{MatchID: "new-koth", Mode: "koth"},     // famille vide : remplie
		{MatchID: "new-ctf", Mode: "ctf"},
		{MatchID: "new-veh", Mode: demoModeVehicles},
	}
	got := mergeDemoReplayPicks(existing, fresh)
	want := []DemoReplayPick{
		{MatchID: "old-fiesta", Mode: "fiesta"},
		{MatchID: "new-koth", Mode: "koth"},
		{MatchID: "old-ctf", Mode: "ctf", Map: "A"},
		{MatchID: "new-veh", Mode: demoModeVehicles},
		{MatchID: "legacy", Mode: "capture_the_moon", Map: "Z"}, // famille inconnue : conservée telle quelle
	}
	if len(got.picks) != len(want) {
		t.Fatalf("fusion %+v, attendu %+v", got.picks, want)
	}
	for i := range want {
		if got.picks[i] != want[i] {
			t.Errorf("entrée %d = %+v, attendu %+v", i, got.picks[i], want[i])
		}
	}
	if strings.Join(got.kept, ",") != "fiesta,ctf" || strings.Join(got.filled, ",") != "koth,vehicles" ||
		strings.Join(got.unknown, ",") != "capture_the_moon" {
		t.Errorf("bilan kept=%v filled=%v unknown=%v", got.kept, got.filled, got.unknown)
	}
	// Une famille sans candidat neuf garde néanmoins son entrée ; un manifeste sans rejeu se remplit.
	if g := mergeDemoReplayPicks(existing[:1], nil); len(g.picks) != 1 || g.picks[0].MatchID != "old-ctf" {
		t.Errorf("entrée existante sans candidat neuf perdue : %+v", g.picks)
	}
	if g := mergeDemoReplayPicks(nil, fresh); len(g.picks) != len(fresh) {
		t.Errorf("manifeste vide non rempli : %+v", g.picks)
	}
}

func TestEmitDemoReplayPicks_AdditifSurLaBase(t *testing.T) {
	f := newVehiclesFixture(t)
	// Fiesta : un match ancien figé au manifeste, un plus récent en base.
	oldFiesta := f.match(1, "BTB:Fiesta CTF", 1, quatreContreQuatre...)
	f.match(2, "BTB:Fiesta CTF", 9, quatreContreQuatre...)
	// KOTH : famille absente du manifeste, remplie.
	koth := f.match(3, "KOTH:Arena", 3, quatreContreQuatre...)
	// Véhicules : famille absente, remplie par le 4v4 à deux types.
	veh := f.match(4, "Slayer:Arena", 4, quatreContreQuatre...)
	f.pass(veh, "p", "2026-10-04 00:00:00", "warthog=2", "banshee=1")
	opts := f.opts()

	mp := filepath.Join(f.root, "config", "demo", "JGtm", "halo_infinite.json")
	if err := writeDemoManifest(mp, &DemoManifest{
		Version: demoManifestVersion, TitleSlug: titlePkg.DefaultSlug,
		Corpus: DemoManifestCorpus{
			SoloMatchIDs: []string{"solo-1"},
			ReplayMatches: []DemoReplayPick{
				{MatchID: oldFiesta, Mode: "fiesta", Map: "Carte1"},
				{MatchID: "legacy-id", Mode: "capture_the_moon"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := EmitDemoReplayPicks(context.Background(), opts, mp)
	if err != nil {
		t.Fatalf("EmitDemoReplayPicks : %v", err)
	}
	byMode := map[string]string{}
	for _, p := range got {
		byMode[p.Mode] = p.MatchID
	}
	if byMode["fiesta"] != oldFiesta {
		t.Errorf("fiesta = %s, le match figé (%s) doit survivre au candidat plus récent", byMode["fiesta"], oldFiesta)
	}
	if byMode["koth"] != koth {
		t.Errorf("koth = %s, attendu %s (famille vide remplie)", byMode["koth"], koth)
	}
	if byMode[demoModeVehicles] != veh {
		t.Errorf("vehicles = %s, attendu %s", byMode[demoModeVehicles], veh)
	}
	if byMode["capture_the_moon"] != "legacy-id" {
		t.Errorf("l'entrée de famille inconnue doit être conservée : %+v", got)
	}
	// Ce qui est écrit au manifeste est ce qui est rendu, le reste du corpus est intact, et
	// une seconde émission ne change plus rien (idempotence).
	for pass := 1; pass <= 2; pass++ {
		m, found, err := LoadDemoManifest(mp)
		if err != nil || !found {
			t.Fatalf("relecture : found=%v err=%v", found, err)
		}
		if len(m.Corpus.ReplayMatches) != len(got) || len(m.Corpus.SoloMatchIDs) != 1 {
			t.Fatalf("passe %d : manifeste %+v", pass, m.Corpus)
		}
		for i, p := range m.Corpus.ReplayMatches {
			if p != got[i] {
				t.Errorf("passe %d, entrée %d = %+v, attendu %+v", pass, i, p, got[i])
			}
		}
		if pass == 1 {
			if again, err := EmitDemoReplayPicks(context.Background(), opts, mp); err != nil || len(again) != len(got) {
				t.Fatalf("seconde émission : %+v err=%v", again, err)
			}
		}
	}
}
