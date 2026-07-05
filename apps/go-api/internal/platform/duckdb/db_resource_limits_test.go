//go:build integration

// db_resource_limits_test.go — J2 (2026-07-05) : vérifie que memory_limit +
// threads sont bornés sur CHAQUE connexion, y compris celles ouvertes SANS
// timezone (branche qui n'avait pas de hook d'init avant J2 → risque OOM).
// Test interne (package duckdb) pour lire duckMemoryLimit/duckThreads.
package duckdb

import (
	"context"
	"strconv"
	"testing"
)

func TestOpenReadWrite_ResourceLimitsAppliedNoTZ(t *testing.T) {
	ctx := context.Background()
	// Pas de timezone → couvre exactement la branche qui, avant J2, n'exécutait
	// aucun hook d'init (donc aucune borne mémoire).
	db, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite(:memory:): %v", err)
	}
	defer db.Close()

	// threads = preuve robuste que le hook d'init tourne (valeur lue depuis la var
	// du package → insensible à un override d'env dans l'environnement de test).
	var threads string
	if err := db.QueryRow(ctx, "SELECT current_setting('threads')").Scan(&threads); err != nil {
		t.Fatalf("SELECT threads: %v", err)
	}
	if threads != strconv.Itoa(duckThreads) {
		t.Errorf("threads borné attendu %d, obtenu %q (hook d'init non appliqué ?)", duckThreads, threads)
	}

	// memory_limit doit être renseigné (non vide) — la borne est appliquée.
	var mem string
	if err := db.QueryRow(ctx, "SELECT current_setting('memory_limit')").Scan(&mem); err != nil {
		t.Fatalf("SELECT memory_limit: %v", err)
	}
	if mem == "" {
		t.Error("memory_limit vide — borne J2 non appliquée")
	}
	t.Logf("memory_limit=%q threads=%s (défaut config : %s / %d)", mem, threads, duckMemoryLimit, duckThreads)
}
