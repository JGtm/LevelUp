// replay_map_repo_carte_test.go — les identites d'une carte designee par son SEUL map_id.
//
// SANS TAG DE BUILD, contrairement a replay_map_repo_test.go, et pour la meme raison que
// tactical_repo_test.go : le gate de la phase 4 joue `go test ./internal/platform/duckdb/...`
// sans `-tags=integration`, et un test derriere un tag que le gate ne pose pas ne garde rien.
//
// VRAIES MIGRATIONS, pas de DDL recopiee : `match_registry` est monte par
// `migration.RunForDB` — une table de test recopiee derive de la vraie sans que rien ne le
// dise, et le test reste vert sur un schema qui n'existe plus.

package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/migration"
)

// newCarteRepo : shared `:memory:` migre + metadata portant `asset_translations`.
func newCarteRepo(t *testing.T) (*ReplayMapRepo, *DB, *DB) {
	t.Helper()
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	_ = migration.All()
	if err := migration.RunForDB(sharedSQL, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sharedSQL, ":memory:")

	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	if _, err := meta.Exec(context.Background(), `CREATE TABLE asset_translations (
		asset_type VARCHAR, asset_id VARCHAR, lang VARCHAR, name VARCHAR)`); err != nil {
		t.Fatalf("ddl asset_translations: %v", err)
	}
	return NewReplayMapRepo(LegacySharedReader(shared), meta), shared, meta
}

func semerCarte(t *testing.T, shared *DB, matchID, mapName, mapID string) {
	t.Helper()
	if _, err := shared.Exec(context.Background(),
		`INSERT INTO match_registry (match_id, map_name, map_id) VALUES (?, ?, ?)`,
		matchID, mapName, mapID,
	); err != nil {
		t.Fatalf("insert registry %s: %v", matchID, err)
	}
}

// TestMapKeysForMap_NomDuRegistre : le map_id rend son nom candidat, pris sur N'IMPORTE
// QUEL match du registre qui le porte — la surface qui appelle ne connait pas de match.
func TestMapKeysForMap_NomDuRegistre(t *testing.T) {
	repo, shared, _ := newCarteRepo(t)
	semerCarte(t, shared, "m1", "Streets", "asset-streets")
	semerCarte(t, shared, "m2", "Streets", "asset-streets")

	got, err := repo.MapKeysForMap(context.Background(), "asset-streets")
	if err != nil {
		t.Fatalf("MapKeysForMap: %v", err)
	}
	if got.MapID != "asset-streets" {
		t.Errorf("MapID = %q", got.MapID)
	}
	if len(got.Names) != 1 || got.Names[0] != "Streets" {
		t.Errorf("candidats = %v, attendu [Streets]", got.Names)
	}
	// PairName qualifie un MATCH (le mode joue), jamais une carte : il doit rester vide.
	if got.PairName != "" {
		t.Errorf("PairName = %q, attendu vide", got.PairName)
	}
}

// TestMapKeysForMap_NomDAsset : le catalogue d'assets passe DEVANT le libelle du registre
// (meme cascade que par match) — c'est lui qui rend un nom canonique quand le registre
// porte un uuid brut.
func TestMapKeysForMap_NomDAsset(t *testing.T) {
	repo, shared, meta := newCarteRepo(t)
	semerCarte(t, shared, "m1", "0000-uuid-brut", "asset-recharge")
	if _, err := meta.Exec(context.Background(),
		`INSERT INTO asset_translations (asset_type, asset_id, lang, name)
		 VALUES ('map', 'asset-recharge', 'en', 'Recharge')`); err != nil {
		t.Fatalf("insert translation: %v", err)
	}

	got, err := repo.MapKeysForMap(context.Background(), "asset-recharge")
	if err != nil {
		t.Fatalf("MapKeysForMap: %v", err)
	}
	if len(got.Names) < 1 || got.Names[0] != "Recharge" {
		t.Errorf("candidats = %v, attendu Recharge en tete", got.Names)
	}
}

// TestMapKeysForMap_JamaisJouee : un map_id absent du registre reste exploitable — sa
// seule presence suffit a essayer la cle map_id du fond. C'est le cas d'une carte que CE
// joueur n'a pas jouee mais qui existe dans le titre.
func TestMapKeysForMap_JamaisJouee(t *testing.T) {
	repo, _, _ := newCarteRepo(t)
	got, err := repo.MapKeysForMap(context.Background(), "asset-inconnu")
	if err != nil {
		t.Fatalf("MapKeysForMap: %v", err)
	}
	if got.MapID != "asset-inconnu" || len(got.Names) != 0 {
		t.Errorf("got = %+v, attendu le seul map_id sans nom", got)
	}
}

// TestMapKeysForMap_Vide : sans map_id il n'y a rien a resoudre — erreur typee, jamais une
// requete a blanc.
func TestMapKeysForMap_Vide(t *testing.T) {
	repo, _, _ := newCarteRepo(t)
	if _, err := repo.MapKeysForMap(context.Background(), "   "); !errors.Is(err, ErrMatchMapUnknown) {
		t.Fatalf("err = %v, attendu ErrMatchMapUnknown", err)
	}
}
