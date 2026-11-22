package historyrepo

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/historystate"
)

// newPlayerDB crée sur disque une base joueur avec sa table sync_meta (une seule connexion), la
// referme et rend son chemin : la lecture rouvre le fichier par OpenReadForQuery.
func newPlayerDB(t *testing.T, markComplete bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
		t.Fatalf("create sync_meta: %v", err)
	}
	if markComplete && !historystate.MarkComplete(context.Background(), db, "JGtm") {
		t.Fatal("la fin d'historique n'a pas été rangée")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestHistoryComplete(t *testing.T) {
	cases := []struct {
		name string
		path func(t *testing.T) string
		want bool
	}{
		{"base joueur absente : aucune passe n'a parcouru l'historique", func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "absente.duckdb")
		}, false},
		{"base sans clé de fin : rattrapage en cours", func(t *testing.T) string { return newPlayerDB(t, false) }, false},
		{"clé de fin rangée : complet", func(t *testing.T) string { return newPlayerDB(t, true) }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := c.path(t)
			var gotTitle, gotGamertag string
			r := New(func(titleSlug, gamertag string) string {
				gotTitle, gotGamertag = titleSlug, gamertag
				return path
			}, nil)
			got, err := r.HistoryComplete(context.Background(), "halo_infinite", "JGtm")
			if err != nil {
				t.Fatalf("HistoryComplete: %v", err)
			}
			if got != c.want {
				t.Errorf("HistoryComplete = %v, attendu %v", got, c.want)
			}
			if gotTitle != "halo_infinite" || gotGamertag != "JGtm" {
				t.Errorf("chemin demandé pour (%q, %q)", gotTitle, gotGamertag)
			}
		})
	}
}

// memShared sert une base partagée en mémoire comme fournisseur de lecture ; release ne ferme rien (la
// base vit le temps du test).
type memShared struct{ db *sql.DB }

func (m memShared) Get(context.Context) (*sql.DB, func(), error) { return m.db, func() {}, nil }

type failingShared struct{}

func (failingShared) Get(context.Context) (*sql.DB, func(), error) {
	return nil, nil, errors.New("base partagée indisponible")
}

func newSharedDB(t *testing.T, seed ...string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	// Les deux tables lues, avec leurs colonnes réelles (types de l'étape shared_core du titre).
	for _, ddl := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, game_variant_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, PRIMARY KEY (match_id, xuid))`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("%s : %v", ddl, err)
		}
	}
	for _, q := range seed {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s : %v", q, err)
		}
	}
	return db
}

const campaignVariant = "00000003-0000-0010-8000-00aa00389b71" // Campagne Halo 5 (analysis.CampaignExcludedVariantIDs)

// Nombre de matchs du joueur et début du plus ancien : les matchs d'un autre joueur ne comptent pas,
// la Campagne d'un titre qui la masque non plus, et le début suit l'expression canonique de fuseau
// (start_time_utc, sinon start_time lu en UTC).
func TestReadHistoryProgress_ComptageEtPlusAncien(t *testing.T) {
	const me, other = "2533274800000001", "2533274800000002"
	db := newSharedDB(t,
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, game_variant_id) VALUES
			('m1', '2024-03-10 12:00:00', '2024-03-10 11:00:00+00', 'slayer'),
			('m2', '2022-05-01 08:30:00', NULL, 'slayer'),
			('m3', '2020-01-01 00:00:00', '2020-01-01 00:00:00+00', '`+campaignVariant+`'),
			('m4', '2019-01-01 00:00:00', '2019-01-01 00:00:00+00', 'slayer')`,
		`INSERT INTO match_participants (match_id, xuid) VALUES
			('m1', '`+me+`'), ('m2', '`+me+`'), ('m3', '`+me+`'), ('m4', '`+other+`')`,
	)
	path := newPlayerDB(t, true)
	r := New(func(string, string) string { return path }, func(string) duckdb.SharedReader { return memShared{db} })

	t.Run("titre sans mode masqué : tout compte", func(t *testing.T) {
		got, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", me)
		if err != nil {
			t.Fatalf("ReadHistoryProgress: %v", err)
		}
		want := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		if got.MatchCount != 3 || !got.OldestAt.Equal(want) || !got.Complete {
			t.Errorf("got %+v, attendu 3 matchs, plus ancien %v, complet", got, want)
		}
	})
	t.Run("titre dont la Campagne est masquée : elle ne compte pas", func(t *testing.T) {
		got, err := r.ReadHistoryProgress(context.Background(), "halo_5", "JGtm", me)
		if err != nil {
			t.Fatalf("ReadHistoryProgress: %v", err)
		}
		want := time.Date(2022, 5, 1, 8, 30, 0, 0, time.UTC) // start_time naïf lu en UTC
		if got.MatchCount != 2 || !got.OldestAt.Equal(want) {
			t.Errorf("got %+v, attendu 2 matchs, plus ancien %v", got, want)
		}
	})
	t.Run("joueur sans match : zéro et date absente", func(t *testing.T) {
		got, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", "2533274800000009")
		if err != nil {
			t.Fatalf("ReadHistoryProgress: %v", err)
		}
		if got.MatchCount != 0 || !got.OldestAt.IsZero() {
			t.Errorf("got %+v, attendu 0 match sans date", got)
		}
	})
}

// L'état rendu suit la clé de fin de la base joueur, indépendamment des matchs rangés.
func TestReadHistoryProgress_Rattrapage(t *testing.T) {
	db := newSharedDB(t)
	path := newPlayerDB(t, false)
	r := New(func(string, string) string { return path }, func(string) duckdb.SharedReader { return memShared{db} })
	got, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", "2533274800000001")
	if err != nil {
		t.Fatalf("ReadHistoryProgress: %v", err)
	}
	if got.Complete {
		t.Error("sans clé de fin, l'historique ne doit pas être réputé complet")
	}
}

func TestReadHistoryProgress_Erreurs(t *testing.T) {
	path := newPlayerDB(t, true)
	pathFn := func(string, string) string { return path }

	t.Run("base partagée indisponible : erreur, clé de fin tout de même rendue", func(t *testing.T) {
		r := New(pathFn, func(string) duckdb.SharedReader { return failingShared{} })
		got, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", "2533274800000001")
		if err == nil {
			t.Fatal("attendu une erreur")
		}
		if !got.Complete {
			t.Error("la clé de fin lue avant l'échec doit rester rendue")
		}
	})
	t.Run("aucun fournisseur pour le titre : erreur", func(t *testing.T) {
		r := New(pathFn, func(string) duckdb.SharedReader { return nil })
		if _, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", "2533274800000001"); err == nil {
			t.Fatal("attendu une erreur")
		}
	})
	t.Run("sans lecteur partagé ni xuid : la clé de fin seule", func(t *testing.T) {
		r := New(pathFn, nil)
		got, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", "2533274800000001")
		if err != nil || !got.Complete || got.MatchCount != 0 {
			t.Errorf("got %+v, err %v, attendu complet sans compte", got, err)
		}
	})
}
