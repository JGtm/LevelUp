//go:build cgo

// Package sync — citations_terminal_state_test.go : état terminal des citations
// (matchs annulés dont les events n'arrivent jamais).
//
// Ces tests couvrent la DÉCISION isolée (matchAge + isCitationsTerminalNoEvents,
// citations.go) sur un match_registry minimal. Le comportement de bout en bout à travers
// BackfillMatchCitations (jeton posé, sortie du pool, non-régression du chemin
// events présents) est couvert par citations_terminal_state_pipeline_test.go,
// qui a besoin de la fixture complète (build tag integration).
package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openTerminalStateDB ouvre une shared DB minimale portant les DEUX colonnes de
// temps du fragment canonique (start_time_utc prioritaire, start_time en repli)
// plus events_loaded. start_time est nullable ici — contrairement à la fixture de
// pipeline — pour pouvoir exercer le cas « âge indéterminable ».
func openTerminalStateDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (
		match_id        VARCHAR PRIMARY KEY,
		start_time      TIMESTAMP,
		start_time_utc  TIMESTAMPTZ,
		events_loaded   BOOLEAN DEFAULT FALSE
	)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	return db
}

// insertRegistryTZ insère un match dont le début est porté par start_time_utc
// (colonne canonique prioritaire), à `ago` dans le passé.
func insertRegistryTZ(t *testing.T, db *sql.DB, matchID string, ago time.Duration) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?)`,
		matchID, time.Now().Add(-ago),
	); err != nil {
		t.Fatalf("insert %s: %v", matchID, err)
	}
}

// TestMatchAge_CanonicalSources vérifie que l'âge se lit sur les deux bras du
// fragment canonique : start_time_utc quand il est présent, start_time (naïf,
// interprété UTC) en repli.
func TestMatchAge_CanonicalSources(t *testing.T) {
	ctx := context.Background()
	db := openTerminalStateDB(t)

	insertRegistryTZ(t, db, "m-tz", 30*24*time.Hour)
	// Repli : start_time_utc NULL, seul le start_time naïf (horloge UTC) porte la date.
	if _, err := db.Exec(
		`INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)`,
		"m-naive", time.Now().UTC().Add(-30*24*time.Hour),
	); err != nil {
		t.Fatalf("insert m-naive: %v", err)
	}

	for _, matchID := range []string{"m-tz", "m-naive"} {
		age, err := matchAge(ctx, db, matchID)
		if err != nil {
			t.Fatalf("matchAge(%s): %v", matchID, err)
		}
		// Bande large (29-31 j) : tolère l'offset du fuseau machine sur le bras naïf,
		// mais reste très loin du seuil de 7 j — le test garde son mordant.
		if age < 29*24*time.Hour || age > 31*24*time.Hour {
			t.Errorf("matchAge(%s) = %v, attendu ~30 jours", matchID, age)
		}
	}
}

// TestMatchAge_Indeterminable vérifie que chaque cas d'âge illisible remonte une
// ERREUR (jamais un âge par défaut, qui jetonnerait un match à tort).
func TestMatchAge_Indeterminable(t *testing.T) {
	ctx := context.Background()
	db := openTerminalStateDB(t)

	// Match présent mais sans aucun horodatage exploitable.
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-no-time')`); err != nil {
		t.Fatalf("insert m-no-time: %v", err)
	}
	// Registre sans les colonnes de temps : la requête d'âge échoue au binder.
	noCols, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	noCols.SetMaxOpenConns(1)
	t.Cleanup(func() { noCols.Close() })
	if _, err := noCols.Exec(
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, events_loaded BOOLEAN)`); err != nil {
		t.Fatalf("create match_registry sans colonnes de temps: %v", err)
	}

	cases := []struct {
		name    string
		db      *sql.DB
		matchID string
	}{
		{"horodatages NULL", db, "m-no-time"},
		{"match absent du registre", db, "m-unknown"},
		{"colonnes de temps absentes", noCols, "m-no-time"},
		{"sharedDB nil", nil, "m-no-time"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := matchAge(ctx, c.db, c.matchID); err == nil {
				t.Error("matchAge doit retourner une erreur quand l'âge est indéterminable")
			}
		})
	}
}

// TestIsCitationsTerminalNoEvents vérifie l'arbitrage de part et d'autre du seuil
// et le tempérament d'échec sûr. Les deux premiers cas encadrent le seuil : une
// inversion de la comparaison d'âge fait rougir le test.
func TestIsCitationsTerminalNoEvents(t *testing.T) {
	ctx := context.Background()
	db := openTerminalStateDB(t)

	insertRegistryTZ(t, db, "m-vieux", citationsTerminalNoEventsAge+24*time.Hour)
	insertRegistryTZ(t, db, "m-recent", citationsTerminalNoEventsAge-24*time.Hour)
	insertRegistryTZ(t, db, "m-frais", time.Hour)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-no-time')`); err != nil {
		t.Fatalf("insert m-no-time: %v", err)
	}

	cases := []struct {
		name    string
		db      *sql.DB
		matchID string
		want    bool
	}{
		{"au-dela du seuil - etat terminal", db, "m-vieux", true},
		{"sous le seuil - reste candidat", db, "m-recent", false},
		{"match du jour - reste candidat", db, "m-frais", false},
		{"age illisible - reste candidat", db, "m-no-time", false},
		{"match inconnu - reste candidat", db, "m-unknown", false},
		{"sharedDB nil - reste candidat", nil, "m-vieux", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isCitationsTerminalNoEvents(ctx, c.db, c.matchID); got != c.want {
				t.Errorf("isCitationsTerminalNoEvents(%s) = %v, want %v", c.matchID, got, c.want)
			}
		})
	}
}
