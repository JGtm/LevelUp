package worldrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestMain câble le provider de steps title-owned (halo_infinite), comme le boot : le schéma partagé en
// dépend.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}

const testSeason = "csrseason13-2"

func newRepo(t *testing.T) (*Repo, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1) // Get et AcquireWriter partagent le handle
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migration shared: %v", err)
	}
	return New(sharedprovider.FromInMemoryDB(db, dbPath)), db
}

// entries fabrique un lot de n entrées. Un SEUL instant pour tout le lot : la vue world_csr_leaderboard_latest ne
// garde que les lignes du dernier fetched_at du couple, un instant par ligne en ferait des lots d'une ligne.
func entries(n, withXUID int, season, playlist string) []domain.LeaderboardEntry {
	fetchedAt := time.Now().UTC()
	out := make([]domain.LeaderboardEntry, 0, n)
	for i := 0; i < n; i++ {
		e := domain.LeaderboardEntry{
			Season: season, Playlist: playlist, Rank: i + 1,
			Gamertag: fmt.Sprintf("P%d", i+1), CSRValue: 2000 - i, FetchedAt: fetchedAt,
		}
		if i < withXUID {
			e.XUID = fmt.Sprintf("253500000000%04d", i+1)
		}
		out = append(out, e)
	}
	return out
}

func TestForProvider_NilSansFournisseur(t *testing.T) {
	if ForProvider(nil) != nil {
		t.Error("sans fournisseur, le magasin doit être nil (le cron est alors un no-op)")
	}
}

func TestRepo_EcritPuisLitLeClassement(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()

	if _, ok, err := r.LatestSeason(ctx); err != nil || ok {
		t.Fatalf("base vide : LatestSeason ok=%v err=%v", ok, err)
	}
	if _, ok, err := r.SnapshotAge(ctx, testSeason); err != nil || ok {
		t.Fatalf("base vide : SnapshotAge ok=%v err=%v", ok, err)
	}

	w, release, err := r.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	all := append(entries(3, 2, "csrseason6-1", "pl-a"), entries(3, 3, testSeason, "pl-a")...)
	n, err := w.InsertCSRSnapshot(ctx, "halo_infinite", all)
	if err != nil || n != 6 {
		t.Fatalf("InsertCSRSnapshot = %d err=%v, attendu 6", n, err)
	}
	if n, err := w.RefreshSeasonCatalog(ctx, "halo_infinite", []domain.WorldSeasonRef{{SeasonID: testSeason}}); err != nil || n != 1 {
		t.Errorf("RefreshSeasonCatalog = %d err=%v, attendu 1", n, err)
	}
	release()

	season, ok, err := r.LatestSeason(ctx)
	if err != nil || !ok || season != testSeason {
		t.Errorf("LatestSeason = %q ok=%v err=%v, attendu %s (rang numérique, pas lexicographique)", season, ok, err, testSeason)
	}
	age, ok, err := r.SnapshotAge(ctx, testSeason)
	if err != nil || !ok || age < 0 || age > time.Hour {
		t.Errorf("SnapshotAge = %v ok=%v err=%v, attendu un âge récent", age, ok, err)
	}
	players, err := r.SeasonPlayers(ctx, testSeason)
	if err != nil || len(players) != 3 {
		t.Errorf("SeasonPlayers = %v err=%v, attendu 3 joueurs", players, err)
	}

	w, release, err = r.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter (stats): %v", err)
	}
	defer release()
	stats := []domain.WorldPlayerSeasonStats{{TitleSlug: "halo_infinite", Gamertag: "P1", SeasonID: testSeason, PlaylistID: "pl-a", MatchCount: 4}}
	if n, err := w.InsertPlayerSeasonStats(ctx, stats); err != nil || n != 1 {
		t.Errorf("InsertPlayerSeasonStats = %d err=%v, attendu 1", n, err)
	}
}

func TestRepo_AssessBatch(t *testing.T) {
	r, db := newRepo(t)
	ctx := context.Background()

	// Aucun lot servi : première capture, rien à protéger.
	v, err := r.AssessBatch(ctx, "halo_infinite", testSeason, "pl-a", entries(10, 10, testSeason, "pl-a"), 200)
	if err != nil || v.HasServed || v.Degraded != "" {
		t.Fatalf("sans lot servi : %+v err=%v", v, err)
	}

	w, release, err := r.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if _, err := w.InsertCSRSnapshot(ctx, "halo_infinite", entries(100, 100, testSeason, "pl-a")); err != nil {
		t.Fatalf("seed lot servi: %v", err)
	}
	release()
	var served int
	if err := db.QueryRow(`SELECT COUNT(*) FROM world_csr_leaderboard_latest WHERE playlist_id = 'pl-a'`).Scan(&served); err != nil || served != 100 {
		t.Fatalf("lot servi = %d err=%v", served, err)
	}

	// Un lot effondré (10 lignes contre 100 servies, profondeur 200) est refusé, les chiffres en font foi.
	v, err = r.AssessBatch(ctx, "halo_infinite", testSeason, "pl-a", entries(10, 10, testSeason, "pl-a"), 200)
	if err != nil || !v.HasServed || v.Degraded == "" || v.Served.Rows != 100 || v.Candidate.Rows != 10 || v.Served.WithXUID != 100 {
		t.Errorf("lot effondré : %+v err=%v", v, err)
	}
	// Un lot complet passe.
	v, err = r.AssessBatch(ctx, "halo_infinite", testSeason, "pl-a", entries(100, 100, testSeason, "pl-a"), 200)
	if err != nil || !v.HasServed || v.Degraded != "" {
		t.Errorf("lot complet : %+v err=%v", v, err)
	}
}

type failingProvider struct{ sharedprovider.Provider }

var errBoom = errors.New("fournisseur indisponible")

func (failingProvider) Get(context.Context) (*sql.DB, func(), error) { return nil, nil, errBoom }
func (failingProvider) AcquireWriter(context.Context) (*sharedprovider.WriterHandle, error) {
	return nil, sharedprovider.ErrDrainTimeout
}

// Les erreurs du fournisseur remontent telles quelles : le cron retente sur ErrDrainTimeout (errors.Is) et
// traite les lectures en échec comme dégradées.
func TestRepo_LesErreursDuFournisseurRemontent(t *testing.T) {
	r := New(failingProvider{})
	ctx := context.Background()
	if _, err := r.SeasonPlayers(ctx, testSeason); !errors.Is(err, errBoom) {
		t.Errorf("SeasonPlayers : %v", err)
	}
	if _, _, err := r.SnapshotAge(ctx, testSeason); !errors.Is(err, errBoom) {
		t.Errorf("SnapshotAge : %v", err)
	}
	if _, _, err := r.LatestSeason(ctx); !errors.Is(err, errBoom) {
		t.Errorf("LatestSeason : %v", err)
	}
	if _, err := r.AssessBatch(ctx, "halo_infinite", testSeason, "pl-a", nil, 0); !errors.Is(err, errBoom) {
		t.Errorf("AssessBatch : %v", err)
	}
	if _, _, err := r.AcquireWriter(ctx); !errors.Is(err, sharedprovider.ErrDrainTimeout) {
		t.Errorf("AcquireWriter : %v", err)
	}
}
