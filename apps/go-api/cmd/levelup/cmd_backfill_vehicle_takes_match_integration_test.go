//go:build integration

package main

// cmd_backfill_vehicle_takes_match_integration_test.go — `backfill-vehicle-takes --match` (plan
// Emprise vehicules, L7.5, RV8). Le 2026-10-01 une LISTE passee a `--match` etait prise telle
// quelle pour un identifiant : la chaine entiere est devenue le `match_id` d une passe, dans une
// table append-only. Ces tests jouent la commande ENTIERE (`runBackfillVehicleTakes`) sur une
// racine temporaire : un shared migre par les vraies migrations, un registre, des artefacts.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// vtRacineAvecRegistre monte une racine (manifeste de capabilities copie du depot, shared migre,
// registre `ids`, un artefact par id) et rend la config de la commande.
func vtRacineAvecRegistre(t *testing.T, ids ...string) *config.AppConfig {
	t.Helper()
	racine := t.TempDir()
	src := filepath.Join("..", "..", "..", "..", "config", "titles", titlePkg.DefaultSlug)
	dst := filepath.Join(racine, "config", "titles", titlePkg.DefaultSlug)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(dst, "mappings"), os.DirFS(filepath.Join(src, "mappings"))); err != nil {
		t.Fatalf("copie des mappings: %v", err)
	}
	chemin := titlePkg.NewPathResolver(racine).SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", chemin)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrerSchemaPartage(db, titlePkg.DefaultSlug); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	for _, id := range ids {
		if _, err := db.Exec(`INSERT INTO match_registry (match_id, map_name) VALUES (?, 'streets')`, id); err != nil {
			t.Fatalf("registre %s: %v", id, err)
		}
		vtArtefact(t, racine, id, 61)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return &config.AppConfig{RepoRoot: racine}
}

// vtIdentifiantsEcrits : les `match_id` de la table BRUTE (le but est de voir TOUT ce qui a ete
// ecrit, y compris un identifiant etranger au registre) ; la base est rouverte apres la commande.
func vtIdentifiantsEcrits(t *testing.T, cfg *config.AppConfig) []string {
	t.Helper()
	h, err := duckdb.OpenReadWrite(titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(titlePkg.DefaultSlug))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	rows, err := h.SQLDb().QueryContext(context.Background(),
		`SELECT DISTINCT match_id FROM match_vehicle_takes ORDER BY match_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

func TestBackfillVehicleTakes_Match(t *testing.T) {
	ids := registreDeTest // aaaaaaaa-..., bbbbbbbb-..., deux abcd1234-... (ambigus)
	cas := []struct {
		nom    string
		match  string
		ecrits []string
		erreur string
	}{
		{"liste de deux prefixes", "aaaaaaaa,bbbbbbbb", []string{ids[0], ids[1]}, ""},
		{"prefixe univoque", "bbbbbbbb", []string{ids[1]}, ""},
		{"identifiant complet", ids[3], []string{ids[3]}, ""},
		{"prefixe inconnu refuse, rien d ecrit", "aaaaaaaa,deadbeef", nil, "aucun match du registre"},
		{"prefixe ambigu refuse, rien d ecrit", "bbbbbbbb,abcd1234", nil, "AMBIGU"},
		{"trop court refuse, rien d ecrit", "aaaaaaa", nil, "au moins 8"},
		{"liste vide de virgules refusee", " , ", nil, "aucun identifiant"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cfg := vtRacineAvecRegistre(t, ids...)
			err := runBackfillVehicleTakes(cfg, []string{"--match", c.match})
			if c.erreur == "" && err != nil {
				t.Fatalf("invocation valide refusee : %v", err)
			}
			if c.erreur != "" && (err == nil || !strings.Contains(err.Error(), c.erreur)) {
				t.Fatalf("erreur = %v, attendu un message contenant %q", err, c.erreur)
			}
			got := vtIdentifiantsEcrits(t, cfg)
			if strings.Join(got, "|") != strings.Join(c.ecrits, "|") {
				t.Errorf("match_id ecrits = %v, attendu %v (la table append-only ne doit jamais recevoir "+
					"la valeur brute de --match)", got, c.ecrits)
			}
		})
	}
}
