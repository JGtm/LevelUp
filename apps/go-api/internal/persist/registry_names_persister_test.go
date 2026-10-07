//go:build integration

// Package persist — registry_names_persister_test.go : la réinscription des noms d'assets du
// registre sur le schéma shared réel (migrations), DuckDB en mémoire.
//
//	CGO_ENABLED=1 go test -tags integration ./internal/persist/ -run RegistryNames
package persist

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openRegistryNamesTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	return db
}

type nomsDuRegistre struct {
	playlist, carte, paire, variante, categorie sql.NullString
}

func lireNoms(t *testing.T, db *sql.DB, matchID string) nomsDuRegistre {
	t.Helper()
	var n nomsDuRegistre
	if err := db.QueryRow(`SELECT playlist_name, map_name, pair_name, game_variant_name, mode_category
		FROM match_registry WHERE match_id = ?`, matchID).
		Scan(&n.playlist, &n.carte, &n.paire, &n.variante, &n.categorie); err != nil {
		t.Fatalf("lecture %s: %v", matchID, err)
	}
	return n
}

// TestRegistryNamesPersister_GardeEtIdempotence : NULL et nom = identifiant sont réécrits, un
// vrai nom ne l'est jamais, mode_category n'est pas touchée, et une seconde passe n'écrit rien.
func TestRegistryNamesPersister_GardeEtIdempotence(t *testing.T) {
	db := openRegistryNamesTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, playlist_id, playlist_name, map_id, map_name, pair_id, pair_name,
		 game_variant_id, game_variant_name, mode_category)
		VALUES ('m1', 'pl-1', 'Ranked Arena', 'map-1', NULL, 'pair-1', 'pair-1', 'gv-1', 'gv-1', 'other')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p := NewRegistryNamesPersister(db)
	ecrits, err := p.WriteMatchNames(ctx, "m1", []RegistryNameWrite{
		{Kind: RegistryNamePlaylist, Name: "NE PAS ECRIRE"}, // vrai nom en place : garde
		{Kind: RegistryNameMap, Name: "Recharge"},           // NULL
		{Kind: RegistryNamePair, Name: "Slayer on Recharge"},
		{Kind: RegistryNameGameVariant, Name: "Slayer"},
	})
	if err != nil {
		t.Fatalf("WriteMatchNames: %v", err)
	}
	if want := []string{RegistryNameMap, RegistryNamePair, RegistryNameGameVariant}; !reflect.DeepEqual(ecrits, want) {
		t.Errorf("genres écrits = %v, want %v", ecrits, want)
	}
	got := lireNoms(t, db, "m1")
	if got.playlist.String != "Ranked Arena" || got.carte.String != "Recharge" ||
		got.paire.String != "Slayer on Recharge" || got.variante.String != "Slayer" {
		t.Errorf("noms = %+v", got)
	}
	if got.categorie.String != "other" {
		t.Errorf("mode_category = %q, want inchangée (other)", got.categorie.String)
	}

	// Seconde passe : tout est convergé, rien ne s'écrit, même avec d'autres noms.
	ecrits, err = p.WriteMatchNames(ctx, "m1", []RegistryNameWrite{{Kind: RegistryNameMap, Name: "Autre"}})
	if err != nil || len(ecrits) != 0 {
		t.Fatalf("seconde passe : écrits = %v, err = %v, want aucun", ecrits, err)
	}
	if lireNoms(t, db, "m1").carte.String != "Recharge" {
		t.Error("seconde passe : un nom juste a été réécrit")
	}
}

// TestRegistryNamesPersister_RefusAvantEcriture : genre inconnu, nom vide ou match vide sont
// refusés sans rien écrire — même pas les écritures valides du même appel.
func TestRegistryNamesPersister_RefusAvantEcriture(t *testing.T) {
	db := openRegistryNamesTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, map_id, map_name) VALUES ('m2', 'map-2', NULL)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	p := NewRegistryNamesPersister(db)
	cas := []struct {
		nom    string
		match  string
		writes []RegistryNameWrite
	}{
		{"genre inconnu", "m2", []RegistryNameWrite{{Kind: RegistryNameMap, Name: "Ok"}, {Kind: "mode_category", Name: "x"}}},
		{"nom vide", "m2", []RegistryNameWrite{{Kind: RegistryNameMap, Name: "Ok"}, {Kind: RegistryNamePair, Name: "  "}}},
		{"match vide", " ", []RegistryNameWrite{{Kind: RegistryNameMap, Name: "Ok"}}},
	}
	for _, c := range cas {
		if _, err := p.WriteMatchNames(ctx, c.match, c.writes); err == nil {
			t.Errorf("%s : err = nil, want refus", c.nom)
		}
	}
	if got := lireNoms(t, db, "m2"); got.carte.Valid {
		t.Errorf("map_name = %q après refus, want NULL (aucune écriture)", got.carte.String)
	}
}
