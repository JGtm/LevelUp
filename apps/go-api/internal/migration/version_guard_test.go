//go:build cgo

package migration

// version_guard_test.go — la garde de version lit l'état de schéma SANS écrire, compte les
// étapes en attente et refuse une base plus récente que le binaire.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// registerGuardSteps enregistre des étapes sur une target de test isolée du reste du registre
// (une target par test).
func registerGuardSteps(t *testing.T, target TargetDB, names ...string) {
	t.Helper()
	for _, n := range names {
		if IsRegistered(n) {
			continue // registre global au processus : une seule fois par nom
		}
		Register(Migration{
			Name:        n,
			TargetDB:    target,
			Description: "étape de test de la garde de version",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + n + " (id INTEGER)")
				return err
			},
		})
	}
}

func openGuardRW(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open rw: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// openGuardRO ouvre la base en lecture seule : une écriture y échouerait.
func openGuardRO(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open ro: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestInspectSchema_BaseSansLedger_EtapesEnAttenteSansEcriture(t *testing.T) {
	target := TargetDB("guard_test_fresh")
	registerGuardSteps(t, target, "guard_fresh_a", "guard_fresh_b")

	path := filepath.Join(t.TempDir(), "fresh.duckdb")
	rw := openGuardRW(t, path)
	if _, err := rw.Exec("CREATE TABLE other (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := InspectSchema(context.Background(), openGuardRO(t, path), DefaultSlug, target)
	if err != nil {
		t.Fatalf("InspectSchema sur connexion lecture seule: %v", err)
	}
	if st.Version != 0 || st.Pending != 2 || st.BinaryVersion != len(canonicalOrder) {
		t.Errorf("état = %+v, attendu version 0, 2 en attente, binaire %d", st, len(canonicalOrder))
	}
	if err := st.CheckNotNewer(); err != nil {
		t.Errorf("base sans ledger jugée plus récente: %v", err)
	}
}

func TestInspectSchema_ApresMontee_RienEnAttenteEtNouvelleEtapeCompte(t *testing.T) {
	target := TargetDB("guard_test_ladder")
	registerGuardSteps(t, target, "guard_ladder_a")

	path := filepath.Join(t.TempDir(), "ladder.duckdb")
	rw := openGuardRW(t, path)
	if err := RunForDB(rw, target); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := InspectSchema(context.Background(), openGuardRO(t, path), DefaultSlug, target)
	if err != nil {
		t.Fatal(err)
	}
	if st.Version != len(canonicalOrder) || st.Pending != 0 {
		t.Errorf("après montée : %+v, attendu version %d et 0 en attente", st, len(canonicalOrder))
	}

	registerGuardSteps(t, target, fmt.Sprintf("guard_ladder_b_%d", time.Now().UnixNano())) // neuve à chaque rejeu (-count)
	st, err = InspectSchema(context.Background(), openGuardRO(t, path), DefaultSlug, target)
	if err != nil {
		t.Fatal(err)
	}
	if st.Pending != 1 {
		t.Errorf("une étape ajoutée au binaire : %d en attente, attendu 1", st.Pending)
	}
}

func TestSchemaState_CheckNotNewer(t *testing.T) {
	if err := (SchemaState{Version: 10, BinaryVersion: 10}).CheckNotNewer(); err != nil {
		t.Errorf("versions égales refusées: %v", err)
	}
	if err := (SchemaState{Version: 9, BinaryVersion: 10}).CheckNotNewer(); err != nil {
		t.Errorf("base plus ancienne refusée: %v", err)
	}
	err := (SchemaState{Version: 11, BinaryVersion: 10}).CheckNotNewer()
	if !errors.Is(err, ErrSchemaNewerThanBinary) {
		t.Errorf("base N+1 : err = %v, attendu ErrSchemaNewerThanBinary", err)
	}
}

func TestInspectSchema_BaseN1_Detectee(t *testing.T) {
	target := TargetDB("guard_test_newer")
	registerGuardSteps(t, target, "guard_newer_a")

	path := filepath.Join(t.TempDir(), "newer.duckdb")
	rw := openGuardRW(t, path)
	if err := RunForDB(rw, target); err != nil {
		t.Fatal(err)
	}
	if _, err := rw.Exec("UPDATE title_schema_version SET version = ? WHERE target = ?",
		len(canonicalOrder)+1, string(target)); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := InspectSchema(context.Background(), openGuardRO(t, path), DefaultSlug, target)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(st.CheckNotNewer(), ErrSchemaNewerThanBinary) {
		t.Errorf("base N+1 non détectée : %+v", st)
	}
}
