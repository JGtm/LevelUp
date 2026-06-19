//go:build integration

// consolidate_aliases_test.go — valide le contrat de dédup du merge
// global→shared.xuid_aliases (sous-commande levelup consolidate-aliases) :
// AUCUN doublon, les xuids déjà présents côté shared sont PRÉSERVÉS (pas
// écrasés), les gamertags vides sont ignorés.

package duckdb

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestConsolidateAliases_DedupByXuid(t *testing.T) {
	db, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	ddl := []string{
		// shared.xuid_aliases : xuid en PK (comme la vraie table).
		`CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY, gamertag VARCHAR, last_seen TIMESTAMP,
			source VARCHAR, updated_at TIMESTAMP
		)`,
		// table simulant glb.xuid_aliases (3 colonnes).
		`CREATE TABLE global_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMP)`,
		// shared a déjà A et B.
		`INSERT INTO xuid_aliases VALUES ('A','Alice',NULL,'sync',now()), ('B','Bob',NULL,'sync',now())`,
		// global : B (overlap, gamertag différent), C (nouveau), D (gamertag vide).
		`INSERT INTO global_aliases VALUES ('B','Bobby',NULL), ('C','Carol',NULL), ('D','',NULL)`,
	}
	for _, s := range ddl {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}

	// Le MÊME INSERT que la sous-commande (glb -> table locale ici).
	if _, err := db.Exec(ctx, `
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		SELECT g.xuid, g.gamertag, g.last_seen, 'global_xbox', now()
		FROM global_aliases g
		WHERE g.gamertag IS NOT NULL AND g.gamertag != ''
		ON CONFLICT (xuid) DO NOTHING`); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// 1. Aucun doublon : COUNT == COUNT(DISTINCT xuid).
	var total, distinct int
	if err := db.QueryRow(ctx, `SELECT COUNT(*), COUNT(DISTINCT xuid) FROM xuid_aliases`).Scan(&total, &distinct); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != distinct {
		t.Errorf("DOUBLON détecté : total=%d distinct=%d", total, distinct)
	}
	// A, B (conservé), C (ajouté) = 3 ; D ignoré (gamertag vide).
	if total != 3 {
		t.Errorf("total = %d, want 3 (A, B, C ; D ignoré car gamertag vide)", total)
	}

	// 2. B PRÉSERVÉ côté shared (pas écrasé par 'Bobby' du global).
	var bGt, bSource string
	if err := db.QueryRow(ctx, `SELECT gamertag, source FROM xuid_aliases WHERE xuid='B'`).Scan(&bGt, &bSource); err != nil {
		t.Fatalf("read B: %v", err)
	}
	if bGt != "Bob" || bSource != "sync" {
		t.Errorf("B écrasé : gamertag=%q source=%q, want Bob/sync (shared prioritaire)", bGt, bSource)
	}

	// 3. C ajouté avec source global_xbox.
	var cGt, cSource string
	if err := db.QueryRow(ctx, `SELECT gamertag, source FROM xuid_aliases WHERE xuid='C'`).Scan(&cGt, &cSource); err != nil {
		t.Fatalf("read C: %v", err)
	}
	if cGt != "Carol" || cSource != "global_xbox" {
		t.Errorf("C = %q/%q, want Carol/global_xbox", cGt, cSource)
	}

	// 4. Idempotent : re-run ne change rien.
	if _, err := db.Exec(ctx, `
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		SELECT g.xuid, g.gamertag, g.last_seen, 'global_xbox', now()
		FROM global_aliases g WHERE g.gamertag IS NOT NULL AND g.gamertag != ''
		ON CONFLICT (xuid) DO NOTHING`); err != nil {
		t.Fatalf("merge idempotent: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM xuid_aliases`).Scan(&total); err != nil {
		t.Fatalf("count 2: %v", err)
	}
	if total != 3 {
		t.Errorf("idempotent: total = %d, want 3", total)
	}
}
