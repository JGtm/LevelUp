//go:build integration

// replay_map_repo_test.go — la carte d'un match, telle que le fond de carte du rejeu 2D
// a besoin de la connaitre.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestReplayMapRepo

package duckdb

import (
	"context"
	"errors"
	"testing"
)

// seedReplayMapFixtures crée le registre partagé minimal et le référentiel de traductions.
func seedReplayMapFixtures(t *testing.T) (*DB, *DB) {
	t.Helper()
	ctx := context.Background()
	shared := openMemDB(t)
	meta := openMemDB(t)

	if _, err := shared.Exec(ctx, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, map_name VARCHAR, map_id VARCHAR)`); err != nil {
		t.Fatalf("ddl registry: %v", err)
	}
	if _, err := meta.Exec(ctx, `CREATE TABLE asset_translations (
		asset_type VARCHAR, asset_id VARCHAR, lang VARCHAR, name VARCHAR)`); err != nil {
		t.Fatalf("ddl translations: %v", err)
	}
	return shared, meta
}

func insertRegistry(t *testing.T, shared *DB, matchID string, mapName, mapID any) {
	t.Helper()
	if _, err := shared.Exec(context.Background(),
		`INSERT INTO match_registry (match_id, map_name, map_id) VALUES (?, ?, ?)`,
		matchID, mapName, mapID,
	); err != nil {
		t.Fatalf("insert registry %s: %v", matchID, err)
	}
}

func insertTranslation(t *testing.T, meta *DB, assetID, lang, name string) {
	t.Helper()
	if _, err := meta.Exec(context.Background(),
		`INSERT INTO asset_translations (asset_type, asset_id, lang, name) VALUES ('map', ?, ?, ?)`,
		assetID, lang, name,
	); err != nil {
		t.Fatalf("insert translation %s/%s: %v", assetID, lang, err)
	}
}

// TestReplayMapRepo_NomDuRegistre — le cas courant : le registre porte le nom canonique.
func TestReplayMapRepo_NomDuRegistre(t *testing.T) {
	shared, meta := seedReplayMapFixtures(t)
	insertRegistry(t, shared, "m1", "Cliffhanger", nil)

	got, err := NewReplayMapRepo(LegacySharedReader(shared), meta).
		MapNamesForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapNamesForMatch: %v", err)
	}
	if len(got) != 1 || got[0] != "Cliffhanger" {
		t.Errorf("candidats = %v, attendu [Cliffhanger]", got)
	}
}

// TestReplayMapRepo_UUIDBrut — LE CAS QUI JUSTIFIE LA CASCADE.
//
// Quand la synchro n'a pas résolu le libellé, `map_name` porte l'UUID de l'asset. Le
// témoin départage : sans la traduction, le seul candidat est un UUID, qui ne désigne
// aucun module ; avec elle, le nom canonique arrive EN PREMIER.
func TestReplayMapRepo_UUIDBrut(t *testing.T) {
	shared, meta := seedReplayMapFixtures(t)
	const uuid = "5324364b-39a8-4f93-96a6-b80a1f18ce8a"
	insertRegistry(t, shared, "m1", uuid, uuid)
	insertTranslation(t, meta, uuid, "en-US", "Cliffhanger")

	got, err := NewReplayMapRepo(LegacySharedReader(shared), meta).
		MapNamesForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapNamesForMatch: %v", err)
	}
	if len(got) == 0 || got[0] != "Cliffhanger" {
		t.Fatalf("candidats = %v, attendu Cliffhanger en tête", got)
	}

	// Contre-épreuve : sans référentiel de traductions, il ne reste que l'UUID.
	sansMeta, err := NewReplayMapRepo(LegacySharedReader(shared), nil).
		MapNamesForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapNamesForMatch sans metadata: %v", err)
	}
	if len(sansMeta) != 1 || sansMeta[0] != uuid {
		t.Errorf("sans metadata : candidats = %v, attendu [%s]", sansMeta, uuid)
	}
}

// TestReplayMapRepo_DeuxCandidatsSansDoublon — les deux sources concordent : un seul
// candidat sort, pas deux fois le même (la casse ne fait pas un second nom).
func TestReplayMapRepo_DeuxCandidatsSansDoublon(t *testing.T) {
	shared, meta := seedReplayMapFixtures(t)
	insertRegistry(t, shared, "m1", "catalyst", "asset-catalyst")
	insertTranslation(t, meta, "asset-catalyst", "en-US", "Catalyst")

	got, err := NewReplayMapRepo(LegacySharedReader(shared), meta).
		MapNamesForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapNamesForMatch: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("candidats = %v, attendu un seul (dédup insensible à la casse)", got)
	}
}

// TestReplayMapRepo_Absences — match inconnu, et match sans carte nommée : même
// sentinelle, jamais une erreur technique.
func TestReplayMapRepo_Absences(t *testing.T) {
	shared, meta := seedReplayMapFixtures(t)
	insertRegistry(t, shared, "vide", nil, nil)
	repo := NewReplayMapRepo(LegacySharedReader(shared), meta)

	for _, matchID := range []string{"inconnu", "vide"} {
		if _, err := repo.MapNamesForMatch(context.Background(), matchID); !errors.Is(err, ErrMatchMapUnknown) {
			t.Errorf("%s: err = %v, attendu ErrMatchMapUnknown", matchID, err)
		}
	}
}
