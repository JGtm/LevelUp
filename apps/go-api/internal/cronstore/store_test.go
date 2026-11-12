package cronstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// migratedDB crée une base migrée (cible donnée) à un chemin temporaire, y exécute seed, la referme et rend
// son chemin.
func migratedDB(t *testing.T, name string, target migration.TargetDB, seed ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	if err := migration.RunForDB(db.SQLDb(), target); err != nil {
		_ = db.Close()
		t.Fatalf("migrate %s: %v", name, err)
	}
	for _, q := range seed {
		if _, err := db.SQLDb().Exec(q); err != nil {
			_ = db.Close()
			t.Fatalf("%s : %v", q, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close %s: %v", name, err)
	}
	return path
}

func TestCountProbe_Compte(t *testing.T) {
	path := migratedDB(t, "shared.duckdb", migration.TargetShared,
		// 2 UUID bruts (1 carte, 1 paire), 1 nom normal.
		`INSERT INTO match_registry (match_id, map_name, pair_name) VALUES
			('m1', '12345678-1234-1234-1234-1234567890ab', 'Quick Play'),
			('m2', 'Recharge', '87654321-4321-4321-4321-ba0987654321'),
			('m3', 'Streets', 'Slayer')`,
		// m1 pose le bit des événements sans aucun événement derrière.
		`UPDATE match_registry SET backfill_completed = 65536 WHERE match_id = 'm1'`,
		// 1 humain sans alias, 1 humain avec alias.
		`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES ('m1', '2603225103667623', 'A'), ('m1', '2699999999999999', 'B')`,
		`INSERT INTO xuid_aliases (xuid, gamertag, last_seen) VALUES ('2699999999999999', 'B', now())`,
	)
	h, err := New(nil).OpenShared(path)
	if err != nil {
		t.Fatalf("OpenShared: %v", err)
	}
	defer h.Close()
	ctx := context.Background()

	want := map[port.HealthProbe]int{
		port.ProbeUUIDsMapName:    1,
		port.ProbeUUIDsPairName:   1,
		port.ProbeLyingBitsEvents: 1,
		port.ProbeOrphanXUIDs:     1,
	}
	for probe, n := range want {
		got, applicable, err := h.CountProbe(ctx, probe)
		if err != nil || !applicable || got != n {
			t.Errorf("%s = %d applicable=%v err=%v, attendu %d", probe, got, applicable, err, n)
		}
	}
	if _, _, err := h.CountProbe(ctx, port.HealthProbe("inconnue")); err == nil {
		t.Error("une sonde inconnue doit échouer")
	}
}

// La sonde des armes ne s'applique que si la base porte une table de preuve ; sinon elle est sautée sans
// erreur.
func TestCountProbe_ArmesSelonLaTableDePreuve(t *testing.T) {
	path := migratedDB(t, "shared.duckdb", migration.TargetShared,
		`INSERT INTO match_registry (match_id, backfill_completed) VALUES ('m1', 2097152)`)
	h, err := New(nil).OpenShared(path)
	if err != nil {
		t.Fatalf("OpenShared: %v", err)
	}
	defer h.Close()
	n, applicable, err := h.CountProbe(context.Background(), port.ProbeLyingBitsWeapons)
	if err != nil {
		t.Fatalf("CountProbe: %v", err)
	}
	if applicable && n != 1 {
		t.Errorf("sonde applicable : %d, attendu 1 (le bit posé sans arme derrière)", n)
	}
	if !applicable && n != 0 {
		t.Errorf("sonde non applicable : n = %d, attendu 0", n)
	}
}

func TestOpenShared_BaseAbsenteEchoue(t *testing.T) {
	if _, err := New(nil).OpenShared(filepath.Join(t.TempDir(), "absente.duckdb")); err == nil {
		t.Fatal("une base absente ne s'ouvre pas en lecture seule")
	}
}

func TestPendingKillSummaries_RemonteLEchecDeLaSonde(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vide.duckdb")
	raw, err := sql.Open("duckdb", path) // aucune table du journal : la sonde échoue
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE t (x INTEGER)`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	h, err := New(nil).OpenShared(path)
	if err != nil {
		t.Fatalf("OpenShared: %v", err)
	}
	defer h.Close()
	if _, err := h.PendingKillSummaries(context.Background()); err == nil {
		t.Error("l'échec de la sonde doit être rendu, pas avalé")
	}
}

func TestCountPlayerProbe_Bannieres(t *testing.T) {
	path := migratedDB(t, "stats.duckdb", migration.TargetPlayer,
		`INSERT INTO career_progression (xuid, banner_image_url) VALUES ('x1', 'https://h/Waypoint/file/images/b.png'), ('x2', 'https://h/ok.png')`)
	s := New(nil)
	n, err := s.CountPlayerProbe(context.Background(), port.ProbeGarbageBannerURLs, path)
	if err != nil || n != 1 {
		t.Errorf("bannières = %d err=%v, attendu 1", n, err)
	}
	_, err = s.CountPlayerProbe(context.Background(), port.ProbeGarbageBannerURLs, filepath.Join(t.TempDir(), "absente.duckdb"))
	if !errors.Is(err, port.ErrDataHealthUnavailable) {
		t.Errorf("base joueur absente : err = %v, attendu ErrDataHealthUnavailable", err)
	}
	if _, err := s.CountPlayerProbe(context.Background(), port.ProbeOrphanXUIDs, path); err == nil {
		t.Error("une sonde de la base partagée ne s'exécute pas sur une base joueur")
	}
}

func TestScanLUSRGaps_BaseJoueurInouvrable(t *testing.T) {
	path := migratedDB(t, "shared.duckdb", migration.TargetShared)
	h, err := New(nil).OpenShared(path)
	if err != nil {
		t.Fatalf("OpenShared: %v", err)
	}
	defer h.Close()
	_, err = h.ScanLUSRGaps(context.Background(), filepath.Join(t.TempDir(), "absente.duckdb"), "2603225103667623")
	if !errors.Is(err, port.ErrDataHealthUnavailable) {
		t.Errorf("err = %v, attendu ErrDataHealthUnavailable", err)
	}
}

func TestMeasurePasses_LecteurInjecteEtTableAbsente(t *testing.T) {
	boom := errors.New("swap en cours")
	_, present, err := New(func(context.Context, string) (*sql.DB, func(), error) { return nil, nil, boom }).
		MeasurePasses(context.Background(), "halo_infinite", "ignoré", migration.CompactableTables()[0])
	if !errors.Is(err, boom) || present {
		t.Errorf("lecteur en échec : present=%v err=%v", present, err)
	}

	raw, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "sans_table.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	released := false
	reader := func(context.Context, string) (*sql.DB, func(), error) {
		return raw, func() { released = true }, nil
	}
	_, present, err = New(reader).MeasurePasses(context.Background(), "halo_infinite", "ignoré", migration.CompactableTables()[0])
	if err != nil || present || !released {
		t.Errorf("table absente : present=%v err=%v released=%v, attendu absente, sans erreur, lecteur rendu", present, err, released)
	}
}

func TestMatchDatesByShortID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`,
		`INSERT INTO match_registry (match_id, start_time_utc) VALUES
			('aaaa0001-1111-4abc-9def-000000000001', '2026-03-04 05:06:07+00'),
			('bbbb0002-1111-4abc-9def-000000000002', NULL)`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("%s : %v", q, err)
		}
	}
	_ = raw.Close()

	got, err := New(nil).MatchDatesByShortID(context.Background(), path)
	if err != nil {
		t.Fatalf("MatchDatesByShortID: %v", err)
	}
	at, ok := got["aaaa0001"]
	if !ok || !at.UTC().Equal(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)) {
		t.Errorf("dates = %v, attendu aaaa0001 -> 2026-03-04 05:06:07 UTC", got)
	}
	if _, ok := got["bbbb0002"]; ok {
		t.Error("un match sans date ne doit pas figurer")
	}
	if _, err := New(nil).MatchDatesByShortID(context.Background(), filepath.Join(t.TempDir(), "absente.duckdb")); err == nil {
		t.Error("une base absente doit échouer")
	}
}
