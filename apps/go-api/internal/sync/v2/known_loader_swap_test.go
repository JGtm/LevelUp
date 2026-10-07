// Package v2 — known_loader_swap_test.go : l'ensemble connu face aux bascules RO↔RW de la base
// partagée (ADR 0016). SharedProvider RÉEL sur fichier DuckDB : une bascule en cours au début de
// la découverte est attendue, une bascule demandée pendant la lecture attend sa fin. Avant
// l'emprunt, la connexion lue en cache était nil ou fermée sous la lecture : ErrSharedUnreadable
// et TOUT le cycle arrêté.
package v2

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
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

// sansCache : la connexion en cache telle qu'une bascule la laisse (aucune) — ce que le
// chargeur lisait avant l'emprunt.
func sansCache() *sql.DB { return nil }

// TestCycle_BasculeEnCoursAuDebutDeLaDecouverte_CycleAboutit : un écrivain tient la base
// partagée (bascule RO→RW faite) quand le cycle démarre ; il la rend 300 ms plus tard. La
// découverte attend le retour en RO, lit l'ensemble connu, et le cycle aboutit : seul le
// match nouveau est récupéré.
func TestCycle_BasculeEnCoursAuDebutDeLaDecouverte_CycleAboutit(t *testing.T) {
	players := mkPlayers("alice")
	prov := newSwapProvider(t, map[string][]string{players[0].XUID: {"m_old"}})
	playerDB, _ := setupTestPlayerDB(t, []string{"m_old"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}

	writer, err := prov.AcquireWriter(context.Background())
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if prov.State() != sharedprovider.StateRW {
		t.Fatalf("pré-condition : état %v, attendu RW (bascule en cours)", prov.State())
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		writer.Release()
	}()

	loader := NewKnownLoader(opener, SharedBorrower(prov.Get, sansCache))
	listProvider := &mockProvider{allMatches: map[string][]string{"alice": {"m_new", "m_old"}}}
	sharedFetcher := &mockFetcher{perMatchData: map[string]map[string]any{"m_new": {"k": 1}}}
	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, &mockEnrichmentFetcher{},
		&mockCyclePersister{}, &mockPostSyncRunner{}, CycleConfig{})

	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("cycle en échec pendant une bascule : %v", err)
	}
	if got := sharedFetcher.totalCalls.Load(); got != 1 {
		t.Errorf("récupérations = %d, attendu 1 (m_new ; m_old connu)", got)
	}
	if st := res.PerPlayer["alice"].Status; st == "failed" {
		t.Errorf("alice en échec : %+v", res.PerPlayer["alice"])
	}
}

// TestKnownLoader_BasculeDemandeePendantLaLecture_AttendSaFin : un écrivain demande la base
// pendant que l'ensemble connu est lu. La bascule attend que l'emprunt soit rendu ; la lecture
// aboutit. Le release de l'emprunt n'est rendu au chargeur qu'une fois la bascule RO→RW faite :
// une lecture menée APRÈS ce release lirait une connexion fermée.
func TestKnownLoader_BasculeDemandeePendantLaLecture_AttendSaFin(t *testing.T) {
	const xuid = "999"
	prov := newSwapProvider(t, map[string][]string{xuid: {"m1", "m2"}})
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(context.Context, string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}

	writers := make(chan *sharedprovider.WriterHandle, 1)
	borrow := func(ctx context.Context) (*sql.DB, func(), error) {
		db, release, err := prov.Get(ctx)
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

	known, err := NewKnownLoader(opener, borrow).LoadKnown(context.Background(),
		PlayerProfile{Gamertag: "alice", XUID: xuid})
	if err != nil {
		t.Fatalf("LoadKnown pendant une bascule demandée : %v", err)
	}
	if len(known) != 2 || !known["m1"] || !known["m2"] {
		t.Errorf("connus = %v, attendu {m1, m2}", known)
	}
}
