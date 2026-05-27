//go:build integration

// Package sync — refresh_shared_views_ro_repro_test.go : reproduction
// minimale du bug "Cannot execute statement of type CREATE on database
// shared_matches_v2 which is attached in read-only mode!" observé dans
// logs/sync.log 23:57:51 sur refreshSharedViews.
//
// HYPOTHÈSE — auto-attach DuckDB :
//
//	Quand 2 connexions sur le même fichier sont ouvertes dans le même
//	process (RO via SharedReader + RW via Provider.AcquireWriter), DuckDB
//	auto-attache l'instance déjà existante sur la nouvelle conn. La RW
//	auto-attache l'instance RO sous le nom dérivé du filename
//	(`shared_matches_v2`). À ce moment, CREATE VIEW sans qualification
//	"main." est résolu vers la database `shared_matches_v2` (attachée RO)
//	au lieu de `main` (la RW directe) → erreur.
//
// Si ce test reproduit l'erreur, le fix est de qualifier toutes les
// queries DDL avec `main.` (ou de forcer `USE main;` au début de la
// transaction).
package sync

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestReproduceCreateViewROAttachViaExplicit_ATTACH : la 2ème hypothèse —
// le bug "attached in read-only mode" provient d'un ATTACH explicite
// READ_ONLY sur la même conn, suivi d'un CREATE VIEW non-qualifié.
//
// Scénario reproduit : un autre fichier ouvert en RW comme "main" + ATTACH
// du shared en READ_ONLY → CREATE VIEW non-qualifié sur xuid_aliases
// (table de shared) tombe dans le schéma de la base attached RO → erreur.
func TestReproduceCreateViewROAttachViaExplicit_ATTACH(t *testing.T) {
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared_matches_v2.duckdb")
	mainPath := filepath.Join(tmp, "main_writer.duckdb")

	// Setup : crée shared avec ses tables.
	setup, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatalf("setup open: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMP)`,
		`CREATE TABLE match_participants (xuid VARCHAR, gamertag VARCHAR)`,
	} {
		if _, err := setup.Exec(ddl); err != nil {
			t.Fatalf("setup ddl: %v", err)
		}
	}
	_ = setup.Close()

	// Ouvre une 2ème DB RW (main_writer), puis ATTACH shared READ_ONLY.
	rwDB, err := sql.Open("duckdb", mainPath)
	if err != nil {
		t.Fatalf("rw open: %v", err)
	}
	defer rwDB.Close()
	if err := rwDB.PingContext(context.Background()); err != nil {
		t.Fatalf("rw ping: %v", err)
	}
	if _, err := rwDB.Exec(`ATTACH '` + sharedPath + `' AS shared_matches_v2 (READ_ONLY)`); err != nil {
		t.Fatalf("ATTACH: %v", err)
	}

	// Tente CREATE VIEW sans qualification, en référençant des tables de shared.
	query := `CREATE OR REPLACE VIEW v_gamertag_lookup AS
		SELECT xa.xuid, xa.gamertag FROM shared_matches_v2.xuid_aliases xa`
	_, err = rwDB.Exec(query)

	if err != nil {
		if strings.Contains(err.Error(), "attached in read-only mode") {
			t.Logf("Bug REPRODUIT (scénario ATTACH READ_ONLY) : %v", err)
			// Tester le fix qualifié main.
			qualifiedQuery := strings.Replace(query, "VIEW v_gamertag_lookup", "VIEW main.v_gamertag_lookup", 1)
			if _, err2 := rwDB.Exec(qualifiedQuery); err2 != nil {
				t.Logf("Fix `main.v_xxx` AUSSI échoue : %v", err2)
			} else {
				t.Logf("Fix `main.v_xxx` PASSE — qualifier explicitement résout le bug")
			}
			return
		}
		t.Logf("CREATE VIEW failed avec autre erreur : %v", err)
		return
	}

	// Pas d'erreur — le bug n'est pas dans ce scénario non plus.
	t.Logf("CREATE VIEW OK sur main avec shared attached RO — donc la query EST bien routée vers main, pas vers shared")

	// Voyons si CREATE VIEW SANS shared qualifier dans la query (juste table name)
	// est routé vers shared (puisque la table existe seulement dans shared).
	query2 := `CREATE OR REPLACE VIEW v_test2 AS SELECT xuid, gamertag FROM xuid_aliases`
	_, err = rwDB.Exec(query2)
	if err != nil {
		t.Logf("CREATE VIEW v_test2 (table ref sans schéma) : %v", err)
		if strings.Contains(err.Error(), "attached in read-only mode") {
			t.Logf("→ BUG REPRODUIT : DuckDB résout `xuid_aliases` vers shared_matches_v2 (la seule où elle existe) ET essaie de créer la VIEW dans le MÊME schéma")
			qualified := `CREATE OR REPLACE VIEW main.v_test3 AS SELECT xuid, gamertag FROM xuid_aliases`
			if _, err2 := rwDB.Exec(qualified); err2 != nil {
				t.Logf("Fix qualified : %v", err2)
			} else {
				t.Logf("✓ Fix `CREATE VIEW main.v_xxx` PASSE")
			}
		}
	} else {
		t.Logf("CREATE VIEW v_test2 OK (table ref sans schéma → routée vers main par défaut)")
	}
}
