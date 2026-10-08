// Package v2 — known_loader_swap_test.go : l'ensemble connu face aux bascules RO↔RW de la base
// partagée (ADR 0016). SharedProvider RÉEL sur fichier DuckDB, connexion en cache réelle
// (duckdbpkg.LookupCachedDB, comme le câblage) :
//
//   - un écrivain qui tient la base longtemps n'arrête pas la découverte : elle lit sa connexion RW ;
//   - une bascule en cours (vidange, ou aucune connexion le temps d'une réouverture) est attendue ;
//   - une base réellement illisible (provider fermé, aucune connexion au-delà de la borne) arrête
//     la découverte ;
//   - une bascule demandée pendant la lecture en RO attend la fin de la lecture ;
//   - une connexion fermée sous la lecture est relue une fois sur un nouvel emprunt.
package v2

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync/knownset"
)

// newSwapProvider : base partagée sur disque (registre + participants), servie par un provider
// B-swap réel. participants : xuid → match_id, chaque match inscrit au registre.
func newSwapProvider(t *testing.T, participants map[string][]string) sharedprovider.Provider {
	t.Helper()
	duckdbpkg.CloseAll()
	t.Cleanup(duckdbpkg.CloseAll)
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	stmts := []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`,
	}
	for _, s := range stmts {
		if _, err := db.SQLDb().Exec(s); err != nil {
			t.Fatalf("schéma shared: %v", err)
		}
	}
	for xuid, ids := range participants {
		for _, id := range ids {
			if _, err := db.SQLDb().Exec(`INSERT INTO match_registry VALUES (?)`, id); err != nil {
				t.Fatalf("registre %s: %v", id, err)
			}
			if _, err := db.SQLDb().Exec(`INSERT INTO match_participants VALUES (?, ?)`, id, xuid); err != nil {
				t.Fatalf("participant %s: %v", id, err)
			}
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close shared: %v", err)
	}
	prov, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	return prov
}

// cacheOf : la connexion en cache du fichier partagé, telle que le câblage la lit.
func cacheOf(prov sharedprovider.Provider) func() *sql.DB {
	return func() *sql.DB {
		if c, ok := duckdbpkg.LookupCachedDB(prov.Path()); ok {
			return c.SQLDb()
		}
		return nil
	}
}

// holdWriter : un écrivain prend la base (bascule RO→RW faite) ; rendu au plus tard au nettoyage.
func holdWriter(t *testing.T, prov sharedprovider.Provider) func() {
	t.Helper()
	w, err := prov.AcquireWriter(context.Background())
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if prov.State() != sharedprovider.StateRW {
		t.Fatalf("pré-condition : état %v, attendu RW", prov.State())
	}
	var once atomic.Bool
	release := func() {
		if once.CompareAndSwap(false, true) {
			w.Release()
		}
	}
	t.Cleanup(release)
	return release
}

// waitProviderState attend que le provider atteigne l'état voulu (bascule observée).
func waitProviderState(t *testing.T, prov sharedprovider.Provider, want sharedprovider.State) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for prov.State() != want {
		if time.Now().After(deadline) {
			t.Fatalf("provider : état %v, attendu %v", prov.State(), want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// runCycleAlice : un cycle V2 complet pour alice (m_old connu, m_new nouveau) ; rend l'erreur du
// cycle et le nombre de récupérations.
func runCycleAlice(ctx context.Context, t *testing.T, borrow SharedDBAcquirer) (CycleResult, int32, error) {
	t.Helper()
	playerDB, _ := setupTestPlayerDB(t, []string{"m_old"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	listProvider := &mockProvider{allMatches: map[string][]string{"alice": {"m_new", "m_old"}}}
	sharedFetcher := &mockFetcher{perMatchData: map[string]map[string]any{"m_new": {"k": 1}}}
	orch := NewCycleOrchestrator(NewKnownLoader(opener, borrow), listProvider, sharedFetcher,
		&mockEnrichmentFetcher{}, &mockCyclePersister{}, &mockPostSyncRunner{}, CycleConfig{})
	res, err := orch.Run(ctx, mkPlayers("alice"))
	return res, sharedFetcher.totalCalls.Load(), err
}

// assertCycleAbouti : cycle sans erreur, seul m_new récupéré (m_old lu connu dans la base partagée).
func assertCycleAbouti(t *testing.T, res CycleResult, fetched int32, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("cycle en échec : %v", err)
	}
	if fetched != 1 {
		t.Errorf("récupérations = %d, attendu 1 (m_new ; m_old connu)", fetched)
	}
	if st := res.PerPlayer["alice"].Status; st == "failed" {
		t.Errorf("alice en échec : %+v", res.PerPlayer["alice"])
	}
}

// TestCycle_EcrivainLongTientLaBase_DecouverteAboutit : un écrivain tient la base partagée
// pendant TOUT le cycle (sync V1 du suivi en direct, attente du film…). La découverte lit sa
// connexion RW en cache et le cycle aboutit tout de suite — attendre la fin de l'écriture (30 s
// au provider) arrêterait tout le cycle jusqu'au tick suivant.
func TestCycle_EcrivainLongTientLaBase_DecouverteAboutit(t *testing.T) {
	prov := newSwapProvider(t, map[string][]string{mkPlayers("alice")[0].XUID: {"m_old"}})
	holdWriter(t, prov)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, fetched, err := runCycleAlice(ctx, t, SharedBorrower(prov, cacheOf(prov)))
	assertCycleAbouti(t, res, fetched, err)
	if prov.State() != sharedprovider.StateRW {
		t.Errorf("état %v après le cycle, attendu RW (l'écrivain tient toujours la base)", prov.State())
	}
}

// TestCycle_BasculeSansConnexion_DecouverteAttendEtAboutit : bascule en cours, aucune connexion
// lisible en cache (fenêtre entre la fermeture d'une connexion et l'ouverture de la suivante) ;
// l'écrivain rend la base 300 ms plus tard. La découverte attend le retour en RO puis aboutit.
func TestCycle_BasculeSansConnexion_DecouverteAttendEtAboutit(t *testing.T) {
	prov := newSwapProvider(t, map[string][]string{mkPlayers("alice")[0].XUID: {"m_old"}})
	release := holdWriter(t, prov)
	go func() {
		time.Sleep(300 * time.Millisecond)
		release()
	}()

	aucuneConnexion := func() *sql.DB { return nil }
	res, fetched, err := runCycleAlice(context.Background(), t, SharedBorrower(prov, aucuneConnexion))
	assertCycleAbouti(t, res, fetched, err)
}

// TestSharedBorrower_VidangeEnCours_AttendLaConnexionRW : un lecteur suivi retient la vidange
// (bascule RO→RW demandée). L'emprunt attend : il ne sert PAS la connexion RO encore en cache,
// que la fin de la vidange va fermer ; il sert la connexion RW une fois la bascule faite.
func TestSharedBorrower_VidangeEnCours_AttendLaConnexionRW(t *testing.T) {
	prov := newSwapProvider(t, map[string][]string{"999": {"m1"}})
	roDB, releaseReader, err := prov.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	writers := make(chan *sharedprovider.WriterHandle, 1)
	go func() {
		w, werr := prov.AcquireWriter(context.Background())
		if werr != nil {
			t.Errorf("AcquireWriter: %v", werr)
			close(writers)
			return
		}
		writers <- w
	}()
	waitProviderState(t, prov, sharedprovider.StateDraining)

	type emprunt struct {
		db      *sql.DB
		release func()
		err     error
	}
	got := make(chan emprunt, 1)
	go func() {
		db, rel, err := SharedBorrower(prov, cacheOf(prov))(context.Background())
		got <- emprunt{db, rel, err}
	}()
	select {
	case e := <-got:
		t.Fatalf("emprunt servi pendant la vidange (db RO = %v, err = %v), attendu une attente", e.db == roDB, e.err)
	case <-time.After(150 * time.Millisecond):
	}
	releaseReader()

	e := <-got
	if e.err != nil {
		t.Fatalf("emprunt après la vidange : %v", e.err)
	}
	defer e.release()
	if e.db == roDB {
		t.Fatal("emprunt = connexion RO fermée par la bascule, attendu la connexion RW")
	}
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("lecture sur l'emprunt : n = %d, err = %v", n, err)
	}
	if w, ok := <-writers; ok {
		w.Release()
	}
}

// TestKnownLoader_ProviderFerme_ArreteLaDecouverte : base réellement illisible (provider fermé) —
// erreur typée, sans attendre.
func TestKnownLoader_ProviderFerme_ArreteLaDecouverte(t *testing.T) {
	prov := newSwapProvider(t, map[string][]string{"999": {"m1"}})
	if err := prov.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}

	debut := time.Now()
	_, err := NewKnownLoader(opener, SharedBorrower(prov, cacheOf(prov))).LoadKnown(context.Background(),
		PlayerProfile{Gamertag: "alice", XUID: "999"})
	if !errors.Is(err, knownset.ErrSharedUnreadable) || !errors.Is(err, sharedprovider.ErrProviderClosed) {
		t.Fatalf("err = %v, attendu knownset.ErrSharedUnreadable (provider fermé)", err)
	}
	if d := time.Since(debut); d > 2*time.Second {
		t.Errorf("arrêt après %v, attendu immédiat", d)
	}
}

// TestKnownLoader_AucuneConnexionAuDelaDeLaBorne_ArreteLaDecouverte : bascule qui ne se termine
// pas et aucune connexion lisible — l'attente est bornée, la découverte s'arrête (erreur typée).
func TestKnownLoader_AucuneConnexionAuDelaDeLaBorne_ArreteLaDecouverte(t *testing.T) {
	prov := newSwapProvider(t, map[string][]string{"999": {"m1"}})
	holdWriter(t, prov)
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	const borne = 200 * time.Millisecond
	inner := sharedBorrower(prov, func() *sql.DB { return nil }, borne, 20*time.Millisecond)
	var emprunts atomic.Int32
	borrow := func(ctx context.Context) (*sql.DB, func(), error) {
		emprunts.Add(1)
		return inner(ctx)
	}

	debut := time.Now()
	_, err := NewKnownLoader(opener, borrow).LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"})
	if !errors.Is(err, knownset.ErrSharedUnreadable) || !errors.Is(err, errSharedSwapWait) {
		t.Fatalf("err = %v, attendu knownset.ErrSharedUnreadable (borne d'attente dépassée)", err)
	}
	if d := time.Since(debut); d < borne || d > borne+2*time.Second {
		t.Errorf("arrêt après %v, attendu juste après la borne %v", d, borne)
	}
	if n := emprunts.Load(); n != 1 {
		t.Errorf("emprunts = %d, attendu 1 (un emprunt impossible n'est pas refait : l'attente est déjà bornée)", n)
	}
}

// TestKnownLoader_BasculeDemandeePendantLaLecture_AttendSaFin : en RO, l'emprunt est SUIVI par le
// provider — un écrivain qui demande la base pendant la lecture de l'ensemble connu attend que
// l'emprunt soit rendu ; la lecture aboutit. La bascule RO→RW n'est faite qu'après le release :
// une lecture menée sur la connexion RO sans emprunt suivi serait fermée sous elle.
func TestKnownLoader_BasculeDemandeePendantLaLecture_AttendSaFin(t *testing.T) {
	const xuid = "999"
	prov := newSwapProvider(t, map[string][]string{xuid: {"m1", "m2"}})
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}

	writers := make(chan *sharedprovider.WriterHandle, 1)
	inner := SharedBorrower(prov, cacheOf(prov))
	borrow := func(ctx context.Context) (*sql.DB, func(), error) {
		db, release, err := inner(ctx)
		if err != nil {
			return nil, nil, err
		}
		go func() {
			w, werr := prov.AcquireWriter(context.Background())
			if werr != nil {
				t.Errorf("AcquireWriter: %v", werr)
				close(writers)
				return
			}
			writers <- w
		}()
		waitProviderState(t, prov, sharedprovider.StateDraining)
		return db, func() {
			release()
			if w, ok := <-writers; ok {
				w.Release()
			}
		}, nil
	}

	known, err := knownOf(NewKnownLoader(opener, borrow).LoadKnown(context.Background(),
		PlayerProfile{Gamertag: "alice", XUID: xuid}))
	if err != nil {
		t.Fatalf("LoadKnown pendant une bascule demandée : %v", err)
	}
	if len(known) != 2 || !known["m1"] || !known["m2"] {
		t.Errorf("connus = %v, attendu {m1, m2}", known)
	}
}

// TestKnownLoader_ConnexionFermeeSousLaLecture_RelueUneFois : la connexion servie est fermée
// (l'écrivain a rendu la base pendant la lecture) — la lecture est refaite une fois sur un
// nouvel emprunt et aboutit.
func TestKnownLoader_ConnexionFermeeSousLaLecture_RelueUneFois(t *testing.T) {
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	sharedDB := setupTestSharedDB(t, map[string][]string{"999": {"m1", "m2"}})
	fermee := setupTestSharedDB(t, map[string][]string{"999": {"m1"}})
	if err := fermee.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	var emprunts atomic.Int32
	borrow := func(context.Context) (*sql.DB, func(), error) {
		if emprunts.Add(1) == 1 {
			return fermee, func() {}, nil
		}
		return sharedDB, func() {}, nil
	}

	known, err := knownOf(NewKnownLoader(opener, borrow).LoadKnown(context.Background(),
		PlayerProfile{Gamertag: "alice", XUID: "999"}))
	if err != nil {
		t.Fatalf("LoadKnown après une connexion fermée sous la lecture : %v", err)
	}
	if len(known) != 2 || !known["m1"] || !known["m2"] {
		t.Errorf("connus = %v, attendu {m1, m2}", known)
	}
	if n := emprunts.Load(); n != 2 {
		t.Errorf("emprunts = %d, attendu 2", n)
	}
}

// TestKnownLoader_BaseSansTables_ArretApresUneRelecture : base partagée lisible mais sans les
// tables de la règle — réellement illisible : une relecture, puis arrêt (erreur typée).
func TestKnownLoader_BaseSansTables_ArretApresUneRelecture(t *testing.T) {
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	vide := setupTestSharedDB(t, nil)
	if _, err := vide.Exec(`DROP TABLE match_participants`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	var emprunts atomic.Int32
	borrow := func(context.Context) (*sql.DB, func(), error) {
		emprunts.Add(1)
		return vide, func() {}, nil
	}

	_, err := NewKnownLoader(opener, borrow).LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"})
	if !errors.Is(err, knownset.ErrSharedUnreadable) {
		t.Fatalf("err = %v, attendu knownset.ErrSharedUnreadable", err)
	}
	if n := emprunts.Load(); n != 2 {
		t.Errorf("emprunts = %d, attendu 2 (une relecture, pas plus)", n)
	}
}
