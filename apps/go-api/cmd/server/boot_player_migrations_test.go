package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// creerBaseJoueur pose un fichier de base joueur vide au chemin résolu (seule son existence compte ici).
func creerBaseJoueur(t *testing.T, pr *title.PathResolver, slug, gamertag string) string {
	t.Helper()
	p := pr.PlayerDBPath(slug, gamertag)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Chaque profil migre la base de SON titre, sous le jeu de migrations de son titre : avant le
// correctif, la boucle du boot ne visait que le titre par défaut et les bases du second titre
// ne recevaient jamais de migration.
func TestMigratePlayerDBs_ChaqueProfilSousSonTitre(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	defaut, second := title.DefaultSlug, "second_titre_test"
	attendus := []string{
		creerBaseJoueur(t, pr, defaut, "Alice") + "|" + defaut,
		creerBaseJoueur(t, pr, second, "Alice") + "|" + second,
		creerBaseJoueur(t, pr, second, "Bob") + "|" + second,
	}
	players := []domain.PlayerSummary{
		{Gamertag: "Alice", TitleSlug: defaut},
		{Gamertag: "Alice", TitleSlug: second},
		{Gamertag: "Bob", TitleSlug: second},
		{Gamertag: "SansBase", TitleSlug: second},           // compte token-only : ignoré
		{Gamertag: "Demo", TitleSlug: second, IsDemo: true}, // démo : ignoré
		{Gamertag: "", TitleSlug: second},                   // sans gamertag : ignoré
	}

	var vus []string
	n := migratePlayerDBs(context.Background(), players, pr, defaut, func(path, slug string) error {
		vus = append(vus, path+"|"+slug)
		return nil
	})

	sort.Strings(vus)
	sort.Strings(attendus)
	if n != 3 || len(vus) != 3 {
		t.Fatalf("attendu 3 bases migrées, obtenu n=%d : %v", n, vus)
	}
	for i := range vus {
		if vus[i] != attendus[i] {
			t.Errorf("migration %d : %q, attendu %q", i, vus[i], attendus[i])
		}
	}
}

// Un échec n'interrompt pas la boucle ; un profil sans titre retombe sur le titre de repli.
func TestMigratePlayerDBs_EchecNonBloquantEtRepli(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	creerBaseJoueur(t, pr, title.DefaultSlug, "Casse")
	creerBaseJoueur(t, pr, title.DefaultSlug, "SansTitre")
	players := []domain.PlayerSummary{
		{Gamertag: "Casse", TitleSlug: title.DefaultSlug},
		{Gamertag: "SansTitre"},
	}
	var slugs []string
	n := migratePlayerDBs(context.Background(), players, pr, title.DefaultSlug, func(path, slug string) error {
		slugs = append(slugs, slug)
		if filepath.Base(filepath.Dir(path)) == "Casse" {
			return errors.New("base verrouillée")
		}
		return nil
	})
	if n != 1 || len(slugs) != 2 || slugs[1] != title.DefaultSlug {
		t.Fatalf("attendu 2 tentatives dont 1 réussie, repli sur le titre par défaut : n=%d, %v", n, slugs)
	}
}
