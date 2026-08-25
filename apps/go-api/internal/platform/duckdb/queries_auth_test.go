//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// TestReadOAuthRefreshToken couvre le DERNIER lecteur legacy encore câblé
// (migration boot ADR 0023 Phase 5). Deux formes de sync_meta existent en prod :
// avec PRIMARY KEY sur `key` (schéma courant) et sans (player DB legacy créée
// par l'ancien sync Python, cf. Chocoboflor) — la lecture doit marcher sur les
// deux, et rendre "" quand la clé est absente (joueur déjà migré).
func TestReadOAuthRefreshToken(t *testing.T) {
	cases := []struct {
		name string
		ddl  string
	}{
		{"avec_pk", "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)"},
		{"legacy_sans_pk", "CREATE TABLE sync_meta (key VARCHAR, value VARCHAR, updated_at TIMESTAMP)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openMemDB(t)
			ctx := context.Background()
			if _, err := db.Exec(ctx, tc.ddl); err != nil {
				t.Fatalf("create sync_meta: %v", err)
			}

			// Clé absente → "" sans erreur (le joueur n'a rien à migrer).
			if got, err := ReadOAuthRefreshToken(ctx, db); err != nil || got != "" {
				t.Fatalf("clé absente = (%q, %v), want (\"\", nil)", got, err)
			}

			if _, err := db.Exec(ctx,
				"INSERT INTO sync_meta(key, value) VALUES ('oauth_refresh_token', 'rt_legacy')"); err != nil {
				t.Fatalf("insert: %v", err)
			}
			got, err := ReadOAuthRefreshToken(ctx, db)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != "rt_legacy" {
				t.Fatalf("read = %q, want rt_legacy", got)
			}
		})
	}
}
