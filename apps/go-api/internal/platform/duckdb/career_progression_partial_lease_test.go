//go:build integration

// Package duckdb — career_progression_partial_lease_test.go : l'écriture d'une ligne
// d'apparence passe par le verrou d'écrivain de la player DB, et le lecteur des
// emblèmes des blocs d'escouade, de Sessions et de Séries temporelles sert la ligne
// la plus récente dès qu'elle est écrite.
package duckdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

const (
	leaseTestOldEmblem = "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Inventory/Emblems/olympus_angrykitty_emblem.png"
	leaseTestNewEmblem = "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Inventory/Emblems/nouvel_embleme.png"
)

// newLeaseTestPlayerDB rend une player DB migrée dont le chemin, clé du verrou
// d'écrivain, est propre au test : un autre test du paquet qui prend le verrou
// d'une base « :memory: » ne peut pas interférer.
func newLeaseTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	pdb := newFreshMigratedPlayerDB(t)
	pdb.Player = newTestDB(pdb.Player.SQLDb(), "lease-test:"+t.Name())
	return pdb
}

func countCareerRows(t *testing.T, pdb *PlayerDB) int {
	t.Helper()
	var n int
	if err := pdb.Player.QueryRow(context.Background(), `SELECT COUNT(*) FROM career_progression`).Scan(&n); err != nil {
		t.Fatalf("comptage career_progression : %v", err)
	}
	return n
}

// TestInsertPartial_SousLeVerrouDEcrivain : tant qu'un autre écrivain tient la
// player DB, l'écriture attend ; si le contexte expire avant, elle échoue avec
// dblease.ErrDBLocked sans rien écrire. Le verrou rendu, elle passe.
func TestInsertPartial_SousLeVerrouDEcrivain(t *testing.T) {
	pdb := newLeaseTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	partial := &CareerProgressionPartial{
		EmblemImageURL:  partialPtr(leaseTestNewEmblem),
		LastFetchStatus: partialPtr("ok"),
	}

	held, err := pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		t.Fatalf("prise du verrou par l'écrivain concurrent : %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err = repo.InsertCareerProgressionPartial(ctx, pTestXUID, partial)
	cancel()
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Fatalf("écriture pendant que le verrou est tenu : err=%v, attendu dblease.ErrDBLocked", err)
	}
	if n := countCareerRows(t, pdb); n != 0 {
		t.Fatalf("%d ligne(s) écrite(s) sans le verrou", n)
	}

	held.Release()
	inserted, err := repo.InsertCareerProgressionPartial(context.Background(), pTestXUID, partial)
	if err != nil || !inserted {
		t.Fatalf("écriture après libération du verrou : inserted=%v err=%v", inserted, err)
	}
}

// TestLoadEmblemURLs_SertLaLigneLaPlusRecente : la base porte l'ancien emblème
// (ligne « ok ») puis des tentatives sans emblème (« api_empty ») ; une nouvelle
// lecture réussie est écrite par le chemin habituel. Le lecteur des emblèmes rend
// le nouveau, sans cache intermédiaire à invalider.
func TestLoadEmblemURLs_SertLaLigneLaPlusRecente(t *testing.T) {
	pdb := newLeaseTestPlayerDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seed := []struct {
		at     time.Time
		emblem any
		status string
	}{
		{now.Add(-84 * 24 * time.Hour), leaseTestOldEmblem, "ok"},
		{now.Add(-31 * 24 * time.Hour), nil, "api_empty"},
	}
	for _, s := range seed {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO career_progression (xuid, recorded_at, emblem_image_url, last_fetch_status) VALUES (?, ?, ?, ?)`,
			pTestXUID, s.at, s.emblem, s.status); err != nil {
			t.Fatalf("seed career_progression : %v", err)
		}
	}
	adapter := NewSquadV2LoaderAdapter(func(context.Context, string, string) (*PlayerDB, error) {
		return pdb, nil
	})
	before := adapter.LoadEmblemURLs(ctx, "halo_infinite", []string{pTestGamertag})[pTestGamertag]
	if want := *buildHomeIdentityAssetURL("emblem", "halo_infinite", leaseTestOldEmblem); before != want {
		t.Fatalf("avant la nouvelle lecture : %q, attendu l'ancien emblème %q", before, want)
	}

	inserted, err := NewCareerLiveRepo(pdb).InsertCareerProgressionPartial(ctx, pTestXUID, &CareerProgressionPartial{
		EmblemImageURL:  partialPtr(leaseTestNewEmblem),
		LastFetchStatus: partialPtr("ok"),
	})
	if err != nil || !inserted {
		t.Fatalf("écriture de la nouvelle lecture : inserted=%v err=%v", inserted, err)
	}

	after := adapter.LoadEmblemURLs(ctx, "halo_infinite", []string{pTestGamertag})[pTestGamertag]
	if want := *buildHomeIdentityAssetURL("emblem", "halo_infinite", leaseTestNewEmblem); after != want {
		t.Fatalf("après la nouvelle lecture : %q, attendu le nouvel emblème %q", after, want)
	}
}
