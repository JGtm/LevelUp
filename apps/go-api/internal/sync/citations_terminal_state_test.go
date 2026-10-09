//go:build cgo

// Package sync — citations_terminal_state_test.go : état terminal des citations
// (matchs annulés dont le film ne livrera jamais d'event).
//
// Ces tests couvrent la DÉCISION isolée (readEventsEmpty, isEventsEmptyDefinitive
// — citations.go) sur un match_registry minimal, y compris le WARN de l'échec sûr.
// Le comportement de bout en bout à travers BackfillMatchCitations (jeton posé,
// sortie du pool, rattrapage par force=true, non-régression du chemin « events
// présents ») est couvert par citations_terminal_state_pipeline_test.go.
//
// OÙ CES TESTS TOURNENT : le job CI `unit` lance CGO_ENABLED=0 sur
// domain/analysis/contracttest uniquement — ce fichier n'y est donc PAS exécuté.
// Il tourne dans le job couverture/intégration (CGO_ENABLED=1, -tags=integration
// ./...), dans check_test_baseline.sh, et en local via `go test ./...`.
package sync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openTerminalStateDB ouvre une shared DB minimale : match_registry avec le
// verdict du pipeline events (events_empty) et events_loaded.
func openTerminalStateDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (
		match_id       VARCHAR PRIMARY KEY,
		events_loaded  BOOLEAN DEFAULT FALSE,
		events_empty   BOOLEAN
	)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	return db
}

// openRegistryWithoutEventsEmpty ouvre un registre au schéma NON migré (colonne
// events_empty absente) : le cas d'une DB de titre qui n'a pas la colonne.
func openRegistryWithoutEventsEmpty(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, events_loaded BOOLEAN)`); err != nil {
		t.Fatalf("create match_registry sans events_empty: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, events_loaded) VALUES ('m-verdict', FALSE)`); err != nil {
		t.Fatalf("insert m-verdict: %v", err)
	}
	return db
}

// seedVerdicts insère les trois états possibles du verdict pipeline events.
func seedVerdicts(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, row := range []struct {
		id    string
		empty any
	}{
		{"m-empty-true", true},   // verdict rendu : film-coquille, aucun event exploitable
		{"m-empty-false", false}, // verdict rendu : le film a livré des events
		{"m-empty-null", nil},    // pas de verdict : le pipeline events retente encore
	} {
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, events_loaded, events_empty) VALUES (?, FALSE, ?)`,
			row.id, row.empty); err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}
}

// TestReadEventsEmpty_VerdictVsIllisible sépare les deux natures de réponse :
// un verdict (true/false/absent = NULL) n'est jamais une erreur ; un verdict
// ILLISIBLE (colonne absente, match hors registre, DB nil) en est toujours une.
func TestReadEventsEmpty_VerdictVsIllisible(t *testing.T) {
	ctx := context.Background()
	db := openTerminalStateDB(t)
	seedVerdicts(t, db)
	noCol := openRegistryWithoutEventsEmpty(t)

	verdicts := []struct {
		matchID string
		want    bool
	}{
		{"m-empty-true", true},
		{"m-empty-false", false},
		{"m-empty-null", false}, // NULL = absence de verdict, pas une erreur
	}
	for _, c := range verdicts {
		got, err := readEventsEmpty(ctx, db, c.matchID)
		if err != nil {
			t.Errorf("readEventsEmpty(%s) erreur inattendue : %v", c.matchID, err)
			continue
		}
		if got != c.want {
			t.Errorf("readEventsEmpty(%s) = %v, want %v", c.matchID, got, c.want)
		}
	}

	illisibles := []struct {
		name    string
		db      *sql.DB
		matchID string
	}{
		{"colonne events_empty absente", noCol, "m-verdict"},
		{"match hors registre", db, "m-unknown"},
		{"sharedDB nil", nil, "m-empty-true"},
	}
	for _, c := range illisibles {
		t.Run(c.name, func(t *testing.T) {
			if _, err := readEventsEmpty(ctx, c.db, c.matchID); err == nil {
				t.Error("readEventsEmpty doit retourner une erreur quand le verdict est illisible")
			}
		})
	}
}

// TestIsEventsEmptyDefinitive vérifie l'arbitrage. Le premier cas est le SEUL qui
// autorise le jeton : retirer la condition ou inverser le booléen fait rougir.
func TestIsEventsEmptyDefinitive(t *testing.T) {
	ctx := context.Background()
	db := openTerminalStateDB(t)
	seedVerdicts(t, db)
	noCol := openRegistryWithoutEventsEmpty(t)

	cases := []struct {
		name    string
		db      *sql.DB
		matchID string
		want    bool
	}{
		{"verdict rendu - film sans event exploitable", db, "m-empty-true", true},
		{"verdict rendu - le film a livre des events", db, "m-empty-false", false},
		{"pas de verdict - le pipeline retente", db, "m-empty-null", false},
		{"colonne absente - reste candidat", noCol, "m-verdict", false},
		{"match hors registre - reste candidat", db, "m-unknown", false},
		{"sharedDB nil - reste candidat", nil, "m-empty-true", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isEventsEmptyDefinitive(ctx, c.db, c.matchID); got != c.want {
				t.Errorf("isEventsEmptyDefinitive(%s) = %v, want %v", c.matchID, got, c.want)
			}
		})
	}
}

// captureSlog redirige le logger par défaut vers un buffer JSON pour la durée du
// test (motif de engine_postsync_csr_warn_test.go).
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// hasWarn indique si le buffer contient un log de niveau WARN dont le message
// contient needle.
func hasWarn(t *testing.T, buf *bytes.Buffer, needle string) bool {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("ligne non-JSON : %s", line)
			continue
		}
		level, _ := entry["level"].(string)
		msg, _ := entry["msg"].(string)
		if level == "WARN" && strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// TestIsEventsEmptyDefinitive_WarnOnIllisible : un verdict illisible ne doit pas
// dégrader EN SILENCE. Sans ce test, supprimer le WARN laissait tout vert — or
// c'est ce WARN qui rend visible une DB non migrée ou un registre incohérent.
// Le second volet prouve que le WARN est SPÉCIFIQUE à l'erreur : l'absence de
// verdict (NULL), qui est un état normal, ne doit rien logger en WARN.
func TestIsEventsEmptyDefinitive_WarnOnIllisible(t *testing.T) {
	ctx := context.Background()
	const needle = "events_empty illisible"

	t.Run("verdict illisible - WARN emis", func(t *testing.T) {
		buf := captureSlog(t)
		noCol := openRegistryWithoutEventsEmpty(t)
		if isEventsEmptyDefinitive(ctx, noCol, "m-verdict") {
			t.Fatal("colonne absente doit laisser le match candidat")
		}
		if !hasWarn(t, buf, needle) {
			t.Errorf("aucun WARN contenant %q — la dégradation est silencieuse ; logs :\n%s", needle, buf.String())
		}
	})

	t.Run("pas de verdict - aucun WARN", func(t *testing.T) {
		buf := captureSlog(t)
		db := openTerminalStateDB(t)
		seedVerdicts(t, db)
		if isEventsEmptyDefinitive(ctx, db, "m-empty-null") {
			t.Fatal("absence de verdict doit laisser le match candidat")
		}
		if hasWarn(t, buf, needle) {
			t.Errorf("WARN émis sur un état NORMAL (pipeline events pas encore conclu) ; logs :\n%s", buf.String())
		}
	})
}
