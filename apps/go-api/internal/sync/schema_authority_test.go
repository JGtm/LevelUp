//go:build integration

package sync_test

// schema_authority_test.go — GARDE-RAIL DE L'AUTORITÉ DE SCHÉMA UNIQUE des player DB
// (décision 2026-08-05).
//
// L'INVARIANT : sur une player DB construite par la SEULE chaîne de migrations,
// sync.EnsurePlayerSchema doit être un NO-OP INTÉGRAL — aucun objet créé (table, vue,
// index, séquence, colonne) et aucun WARN `schema_drift_healed` émis.
//
// POURQUOI C'EST LE TEST QUI MANQUAIT. Deux autorités de schéma concurrentes ont
// cohabité des mois (chaîne de migrations + EnsurePlayerSchema). Le soin SILENCIEUX
// d'Ensure masquait la dérive : elle n'était visible que sur une DB fraîche/reconstruite
// (bug prod career_progression du 2026-08-05, colonne last_fetch_status effacée par un
// step postérieur). Tant qu'aucun test ne comparait les DEUX chemins, toute évolution de
// schéma posée d'un seul côté repartait en dérive.
//
// L'ORACLE EST INDÉPENDANT : la photographie de schéma est ré-implémentée ICI (requêtes
// de catalogue écrites dans ce fichier), et non importée de l'implémentation testée —
// sinon un bug de la photo rendrait le garde-rail complice.
//
// MORSURE PROUVÉE : TestPlayerSchemaAuthority_BiteProof retire de la DB fraîche des
// objets posés par la chaîne (l'effet observable exact d'un step supprimé) et exige que
// le test rougisse EN DÉSIGNANT l'objet manquant.

import (
	"context"
	"database/sql"
	"log/slog"
	"sort"
	"strings"
	stdsync "sync"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── Oracle indépendant : photographie du schéma ───────────────────────────────

// authoritySchemaQueries : une entrée par famille d'objets, chaque requête renvoyant
// une CLÉ LISIBLE (utilisée telle quelle dans les messages d'échec).
var authoritySchemaQueries = []string{
	`SELECT 'table ' || table_name FROM duckdb_tables() WHERE schema_name = 'main'`,
	`SELECT 'view ' || view_name FROM duckdb_views() WHERE schema_name = 'main' AND NOT internal`,
	`SELECT 'index ' || index_name || ' ON ' || table_name FROM duckdb_indexes() WHERE schema_name = 'main'`,
	`SELECT 'sequence ' || sequence_name FROM duckdb_sequences() WHERE schema_name = 'main'`,
	`SELECT 'column ' || table_name || '.' || column_name FROM duckdb_columns() WHERE schema_name = 'main'`,
}

// snapshotSchemaKeys photographie tous les objets de schéma de la DB.
func snapshotSchemaKeys(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, q := range authoritySchemaQueries {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatalf("introspection (%s): %v", q, err)
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				t.Fatalf("scan introspection: %v", err)
			}
			out[key] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatalf("rows introspection: %v", err)
		}
		rows.Close()
	}
	if len(out) == 0 {
		t.Fatal("photographie de schéma vide — introspection cassée, le garde-rail se mentirait")
	}
	return out
}

// missingKeys retourne, triées, les clés de `after` absentes de `before`.
func missingKeys(before, after map[string]bool) []string {
	var out []string
	for k := range after {
		if !before[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// ── Capture des WARN schema_drift_healed ─────────────────────────────────────

type driftCapture struct {
	mu    stdsync.Mutex
	lines []string
}

func (c *driftCapture) add(line string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, line)
}

func (c *driftCapture) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.lines...)
}

// driftHandler : handler slog minimal qui ne retient que les enregistrements
// `schema_drift_healed`, aplatis en une ligne lisible.
type driftHandler struct{ cap *driftCapture }

func (h driftHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h driftHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Message != "schema_drift_healed" {
		return nil
	}
	var parts []string
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, a.Key+"="+a.Value.String())
		return true
	})
	sort.Strings(parts)
	h.cap.add(r.Message + " " + strings.Join(parts, " "))
	return nil
}

func (h driftHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h driftHandler) WithGroup(string) slog.Handler      { return h }

// captureDriftWarnings redirige le logger par défaut vers le capteur le temps du test.
func captureDriftWarnings(t *testing.T) *driftCapture {
	t.Helper()
	cap := &driftCapture{}
	previous := slog.Default()
	slog.SetDefault(slog.New(driftHandler{cap: cap}))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return cap
}

// ── DB fraîche par la SEULE chaîne de migrations ──────────────────────────────

func freshMigratedPlayerDB(t *testing.T) *sql.DB {
	t.Helper()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	return db
}

// ── L'INVARIANT ──────────────────────────────────────────────────────────────

// TestPlayerSchemaAuthority_EnsureIsNoOpOnFreshMigrations — LE test d'autorité unique.
// Rouge dès qu'un objet de schéma naît d'un seul des deux chemins.
func TestPlayerSchemaAuthority_EnsureIsNoOpOnFreshMigrations(t *testing.T) {
	db := freshMigratedPlayerDB(t)
	capture := captureDriftWarnings(t)

	before := snapshotSchemaKeys(t, db)
	if err := sync.EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema sur DB fraîchement migrée: %v", err)
	}
	after := snapshotSchemaKeys(t, db)

	if added := missingKeys(before, after); len(added) > 0 {
		t.Errorf("AUTORITÉ DE SCHÉMA VIOLÉE : sync.EnsurePlayerSchema a créé %d objet(s) absent(s) de "+
			"la chaîne de migrations — ces objets doivent recevoir leur step de migration :\n  - %s",
			len(added), strings.Join(added, "\n  - "))
	}
	if removed := missingKeys(after, before); len(removed) > 0 {
		t.Errorf("sync.EnsurePlayerSchema a SUPPRIMÉ %d objet(s) d'une DB fraîchement migrée :\n  - %s",
			len(removed), strings.Join(removed, "\n  - "))
	}
	if warns := capture.snapshot(); len(warns) > 0 {
		t.Errorf("AUTORITÉ DE SCHÉMA VIOLÉE : %d WARN schema_drift_healed émis sur une DB fraîchement "+
			"migrée (attendu : zéro) :\n  - %s", len(warns), strings.Join(warns, "\n  - "))
	}
}

// TestPlayerSchemaAuthority_NoCareerXuidIndex — décision A du 2026-08-05 :
// idx_career_xuid est supprimé PARTOUT (xuid quasi constant dans une player DB →
// sélectivité nulle ; surface ART #23046 pure perte). Ni la chaîne, ni le soin ne
// doivent le laisser en place.
func TestPlayerSchemaAuthority_NoCareerXuidIndex(t *testing.T) {
	db := freshMigratedPlayerDB(t)
	if err := sync.EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if snapshotSchemaKeys(t, db)["index idx_career_xuid ON career_progression"] {
		t.Error("idx_career_xuid présent après migrations + soin — l'index ART doit être supprimé partout " +
			"(step drop_career_xuid_art_index_v1 + retrait de playerSchemaSQL)")
	}
}

// ── MORSURE ──────────────────────────────────────────────────────────────────

// TestPlayerSchemaAuthority_BiteProof — preuve que l'invariant MORD. On reproduit dans la
// DB fraîche l'effet observable exact d'un step de migration retiré (l'objet qu'il crée
// n'est plus là), et on exige que la comparaison le DÉSIGNE nommément et qu'un WARN
// `schema_drift_healed` soit émis. Sans ce test, un invariant trivialement vert ne
// prouverait rien.
func TestPlayerSchemaAuthority_BiteProof(t *testing.T) {
	cases := []struct {
		name       string
		mutations  []string // effet d'un step retiré
		wantObject string   // fragment attendu dans la clé de schéma manquante
		wantWarn   string   // fragment attendu dans le WARN schema_drift_healed
	}{
		{
			// = create_personal_score_awards_player_v1 retiré.
			name:       "step_create_personal_score_awards_retire",
			mutations:  []string{`DROP VIEW IF EXISTS personal_score_awards_latest`, `DROP TABLE personal_score_awards`},
			wantObject: "table personal_score_awards",
			wantWarn:   "object=personal_score_awards",
		},
		{
			// = create_player_csr_snapshots_player_v1 retiré.
			name:       "step_create_player_csr_snapshots_retire",
			mutations:  []string{`DROP VIEW IF EXISTS player_csr_snapshots_latest`, `DROP TABLE player_csr_snapshots`},
			wantObject: "table player_csr_snapshots",
			wantWarn:   "object=player_csr_snapshots",
		},
		{
			// = colonne effacée par un step postérieur (dérive prod du 2026-08-05).
			name:       "colonne_career_last_fetch_status_effacee",
			mutations:  []string{`ALTER TABLE career_progression DROP COLUMN last_fetch_status`},
			wantObject: "column career_progression.last_fetch_status",
			wantWarn:   "object=last_fetch_status",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := freshMigratedPlayerDB(t)
			for _, stmt := range tc.mutations {
				if _, err := db.Exec(stmt); err != nil {
					t.Fatalf("mutation sonde (%s): %v", stmt, err)
				}
			}
			capture := captureDriftWarnings(t)

			before := snapshotSchemaKeys(t, db)
			if err := sync.EnsurePlayerSchema(context.Background(), db); err != nil {
				t.Fatalf("EnsurePlayerSchema: %v", err)
			}
			after := snapshotSchemaKeys(t, db)

			added := missingKeys(before, after)
			if !containsFragment(added, tc.wantObject) {
				t.Errorf("morsure absente : l'objet %q n'apparaît pas dans les objets recréés par le soin %v",
					tc.wantObject, added)
			}
			warns := capture.snapshot()
			if !containsFragment(warns, tc.wantWarn) {
				t.Errorf("morsure absente : aucun WARN schema_drift_healed ne porte %q — WARN captés : %v",
					tc.wantWarn, warns)
			}
		})
	}
}

func containsFragment(lines []string, fragment string) bool {
	for _, l := range lines {
		if strings.Contains(l, fragment) {
			return true
		}
	}
	return false
}
