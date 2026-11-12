package importstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// sharedMemoryDB ouvre une base partagée en mémoire au schéma de production (colonnes ajoutées par les
// migrations de titre comprises, que persist.SharedPersister écrit).
func sharedMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sync.EnsureSharedSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq`,
		`CREATE TABLE IF NOT EXISTS highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
			time_ms INTEGER, xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR)`,
		`ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS match_intensity DOUBLE`,
		`ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS backfill_completed BIGINT DEFAULT 0`,
		`ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS backfill_bits INTEGER`,
		`ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS assassination_kills SMALLINT DEFAULT 0`,
		`ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS ground_pound_kills SMALLINT DEFAULT 0`,
		`ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS shoulder_bash_kills SMALLINT DEFAULT 0`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s : %v", q, err)
		}
	}
	return db
}

func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%s : %v", q, err)
	}
	return n
}

func TestSharedAcquirer_EcritMatchEvenementsEtAlias(t *testing.T) {
	db := sharedMemoryDB(t)
	acq := NewSharedAcquirer(sharedprovider.FromInMemoryDB(db, ""), "")
	ctx := context.Background()

	store, release, err := acq.AcquireShared(ctx)
	if err != nil {
		t.Fatalf("AcquireShared: %v", err)
	}
	defer release()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	write := port.OpenSpartanMatchWrite{
		Registry:     domain.MatchRegistryRow{MatchID: "11111111-1111-1111-1111-111111111111", StartTime: start, ModeCategory: "Other"},
		Participants: []domain.MatchParticipantRow{{MatchID: "11111111-1111-1111-1111-111111111111", XUID: "2603225103667623"}},
		Medals:       []domain.MedalRow{{MatchID: "11111111-1111-1111-1111-111111111111", XUID: "2603225103667623", MedalNameID: 7, Count: 2}},
		Aliases:      []port.OpenSpartanAlias{{XUID: "2603225103667623", Gamertag: "Owner", LastSeen: start}},
	}
	if err := store.PersistMatch(ctx, write); err != nil {
		t.Fatalf("PersistMatch: %v", err)
	}
	// Idempotent : un match déjà présent n'est pas réécrit.
	if err := store.PersistMatch(ctx, write); err != nil {
		t.Fatalf("PersistMatch (second) : %v", err)
	}
	if n := count(t, db, `SELECT count(*) FROM match_registry`); n != 1 {
		t.Errorf("match_registry : %d ligne(s), attendu 1", n)
	}
	if n := count(t, db, `SELECT count(*) FROM match_participants`); n != 1 {
		t.Errorf("match_participants : %d ligne(s), attendu 1", n)
	}
	if n := count(t, db, `SELECT count(*) FROM medals_earned`); n != 1 {
		t.Errorf("medals_earned : %d ligne(s), attendu 1", n)
	}
	if n := count(t, db, `SELECT count(*) FROM xuid_aliases WHERE xuid = ?`, "2603225103667623"); n != 1 {
		t.Errorf("xuid_aliases : %d ligne(s), attendu 1", n)
	}

	got, err := store.InsertHighlightEvents(ctx, "11111111-1111-1111-1111-111111111111",
		[]highlightevent.HighlightEvent{{EventType: "kill", XUID: 2603225103667623, TimeMS: 1200}})
	if err != nil || got != 1 {
		t.Fatalf("InsertHighlightEvents : n=%d err=%v", got, err)
	}
	if err := store.UpsertOwnerAlias(ctx, "2603225103667623", "OwnerRenamed"); err != nil {
		t.Fatalf("UpsertOwnerAlias: %v", err)
	}
	var gt string
	if err := db.QueryRow(`SELECT gamertag FROM xuid_aliases WHERE xuid = ?`, "2603225103667623").Scan(&gt); err != nil || gt != "OwnerRenamed" {
		t.Errorf("alias de l'importeur = %q (err %v), attendu OwnerRenamed", gt, err)
	}
}

// failingProvider refuse la prise du writer et compte les essais.
type failingProvider struct {
	sharedprovider.Provider
	tries atomic.Int32
}

var errWriterRefused = errors.New("writer refusé par le test")

func (p *failingProvider) AcquireWriter(context.Context) (*sharedprovider.WriterHandle, error) {
	p.tries.Add(1)
	return nil, errWriterRefused
}

func TestSharedAcquirer_RendLErreurDeLaPriseDuWriter(t *testing.T) {
	p := &failingProvider{}
	store, release, err := NewSharedAcquirer(p, "").AcquireShared(context.Background())
	if !errors.Is(err, errWriterRefused) || store != nil || release != nil {
		t.Fatalf("attendu l'échec du writer sans magasin : store=%v err=%v", store, err)
	}
	if p.tries.Load() != 1 {
		t.Errorf("essais = %d, attendu 1", p.tries.Load())
	}
}

func openPostImportSession(t *testing.T, provider sharedprovider.Provider) port.OpenSpartanPostImportSession {
	t.Helper()
	dir := t.TempDir()
	session, err := NewPostImportStore(provider).Open(context.Background(), port.OpenSpartanPostImportPaths{
		Player:   filepath.Join(dir, "player", "stats.duckdb"),
		Shared:   filepath.Join(dir, "shared.duckdb"),
		Metadata: filepath.Join(dir, "metadata.duckdb"),
		PVE:      filepath.Join(dir, "pve.duckdb"),
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestPostImportSession_PrimeEnrichmentEstIdempotent(t *testing.T) {
	session := openPostImportSession(t, nil)
	ctx := context.Background()
	for pass := 1; pass <= 2; pass++ {
		failures, err := session.PrimeEnrichment(ctx, []string{"prime-1", "prime-2"})
		if err != nil || len(failures) != 0 {
			t.Fatalf("passage %d : failures=%v err=%v", pass, failures, err)
		}
	}
	impl := session.(*postImportSession)
	if n := count(t, impl.player, `SELECT count(*) FROM player_match_enrichment_latest WHERE match_id IN ('prime-1', 'prime-2')`); n != 2 {
		t.Errorf("%d ligne(s) dans la vue, attendu 2", n)
	}
	if n := count(t, impl.player, `SELECT count(*) FROM player_match_enrichment WHERE stage = 'live'`); n != 2 {
		t.Errorf("%d ligne(s) 'live' écrites, attendu 2 (pas de doublon au second passage)", n)
	}
}

// Une étape qui ne peut pas prendre la base partagée le dit par son nom : le service range l'échec sous ce
// nom.
func TestPostImportSession_EchecDePriseNommeLEtape(t *testing.T) {
	p := &failingProvider{}
	session := openPostImportSession(t, p)
	ctx := context.Background()

	calls := map[string]func() error{
		"csr_acquire":         func() error { _, err := session.ProjectCSR(ctx, "x"); return err },
		"lusr_acquire":        func() error { _, err := session.RecomputeLUSR(ctx, "x"); return err },
		"perf_scores_acquire": func() error { _, err := session.ComputePerfScores(ctx, "x", false); return err },
	}
	for want, call := range calls {
		var se *port.OpenSpartanStageError
		err := call()
		if !errors.As(err, &se) || se.Stage != want {
			t.Errorf("%s : err = %v, attendu une OpenSpartanStageError d'étape %q", want, err, want)
			continue
		}
		if !errors.Is(err, errWriterRefused) || err.Error() != se.Err.Error() {
			t.Errorf("%s : la cause du fournisseur doit rester lisible et le texte inchangé, obtenu %q", want, err)
		}
	}
}

func TestSiteShared_RendLeHandleDuFournisseur(t *testing.T) {
	db := sharedMemoryDB(t)
	got, release, err := SiteShared(sharedprovider.FromInMemoryDB(db, ""), "")(context.Background())
	if err != nil {
		t.Fatalf("SiteShared: %v", err)
	}
	defer release()
	if got != db {
		t.Error("le handle rendu doit être celui du fournisseur")
	}
	if _, _, err := SiteShared(&failingProvider{}, "")(context.Background()); !errors.Is(err, errWriterRefused) {
		t.Errorf("l'échec du writer doit remonter : %v", err)
	}
}

func TestSitePlayer_OuvreLaBaseDuJoueurSousSonBail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player", "stats.duckdb")
	db, release, err := SitePlayer(path)(context.Background())
	if err != nil {
		t.Fatalf("SitePlayer: %v", err)
	}
	defer release()
	if n := count(t, db, `SELECT count(*) FROM player_match_enrichment_latest`); n != 0 {
		t.Errorf("base neuve : %d ligne(s)", n)
	}
}
