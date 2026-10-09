//go:build integration

package killcollector

// resolveur_de_carte_integration_test.go — LE POST-SYNC TRADUIT UN `map_name` QUI EST UN UUID BRUT
// (revue du correctif J7, 2026-09-27).
//
// Quand la synchro n a pas resolu le libelle de la carte, `match_registry.map_name` porte l UUID de
// l asset. Le resolveur du post-sync etait construit SANS metadonnees : son seul candidat etait cet
// UUID, absent du catalogue de bornes, et le match etait ecarte pour toujours — alors que la carte
// est connue d `asset_translations` et que le backfill hors ligne la decode. Avant la carte
// obligatoire, ce match etait decode (aux largeurs de Cliffhanger) : c etait une regression.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne plus emprunter le handle metadata (metadata nil) ;
// emprunter un autre chemin que celui fourni.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

const uuidDeCarte = "1c1d9a0b-0000-4000-8000-00000000bazr"

// lecteurFixe : un lecteur shared qui rend toujours la meme base, sans la fermer.
type lecteurFixe struct{ db *sql.DB }

func (l lecteurFixe) Get(context.Context) (*sql.DB, func(), error) { return l.db, func() {}, nil }

// basesDuResolveur : un registre dont la carte est un UUID brut, et une base de metadonnees
// OUVERTE PAR LE « PROCESSUS » (cache de `platform/duckdb`, comme le serveur la tient) qui le
// traduit. Rend le chemin de cette base.
func basesDuResolveur(t *testing.T) (lecteurFixe, string) {
	t.Helper()
	ctx := context.Background()
	shared, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("shared : %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	if _, err := shared.ExecContext(ctx, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, map_name VARCHAR, map_id VARCHAR, pair_name VARCHAR)`); err != nil {
		t.Fatalf("ddl registre : %v", err)
	}
	if _, err := shared.ExecContext(ctx, `INSERT INTO match_registry VALUES ('m1', ?, ?, NULL)`,
		uuidDeCarte, uuidDeCarte); err != nil {
		t.Fatalf("registre : %v", err)
	}

	chemin := filepath.Join(t.TempDir(), "metadata.duckdb")
	meta, err := duckdb.OpenReadWriteShared(chemin)
	if err != nil {
		t.Fatalf("metadata : %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	if _, err := meta.Exec(ctx, `CREATE TABLE asset_translations (
		asset_type VARCHAR, asset_id VARCHAR, lang VARCHAR, name VARCHAR)`); err != nil {
		t.Fatalf("ddl traductions : %v", err)
	}
	if _, err := meta.Exec(ctx, `INSERT INTO asset_translations VALUES ('map', ?, 'en-US', 'Bazaar')`,
		uuidDeCarte); err != nil {
		t.Fatalf("traduction : %v", err)
	}
	return lecteurFixe{db: shared}, chemin
}

func TestResolveurDeCartePostSyncTraduitUnUUIDBrut(t *testing.T) {
	shared, chemin := basesDuResolveur(t)
	keys, err := ResolveurDeCartePostSync(shared, chemin).MapKeysForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapKeysForMatch : %v", err)
	}
	if len(keys.Names) == 0 || keys.Names[0] != "Bazaar" {
		t.Fatalf("candidats = %v, attendu [Bazaar %s] — le post-sync ne traduit pas l UUID brut du "+
			"registre : le match est ecarte pour toujours alors que sa carte est connue", keys.Names, uuidDeCarte)
	}
	if _, err := catalogueDuDepot(t).Lookup(keys.Names[0]); err != nil {
		t.Errorf("la carte traduite %q n est pas au catalogue de bornes : %v", keys.Names[0], err)
	}
}

// TestResolveurDeCartePostSyncSansMetadonneesTenues : aucune base de metadonnees tenue par le
// processus — le resolveur N OUVRE RIEN (modele mono-processus, ADR 0013/0016) et retombe sur le
// libelle brut.
func TestResolveurDeCartePostSyncSansMetadonneesTenues(t *testing.T) {
	shared, _ := basesDuResolveur(t)
	absent := filepath.Join(t.TempDir(), "jamais-ouverte.duckdb")
	keys, err := ResolveurDeCartePostSync(shared, absent).MapKeysForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapKeysForMatch : %v", err)
	}
	if len(keys.Names) != 1 || keys.Names[0] != uuidDeCarte {
		t.Errorf("candidats = %v, attendu le seul libelle brut", keys.Names)
	}
	if _, tenue := duckdb.LookupCachedDB(absent); tenue {
		t.Error("le resolveur a OUVERT une base de metadonnees : interdit (un seul writer par base)")
	}
}
