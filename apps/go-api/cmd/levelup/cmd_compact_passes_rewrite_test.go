//go:build cgo

package main

// cmd_compact_passes_rewrite_test.go — les trois règles de l'échange de `--rewrite-file` (C.8,
// revue adversariale L1 du 2026-09-26) : sauvegarde sur un autre volume, jamais de fenêtre sans
// base au chemin, source modifiée ou tenue entre la copie et l'échange.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

// avecEtape pose le point d'observation de la réécriture le temps du test.
func avecEtape(t *testing.T, f func(etape string) error) {
	t.Helper()
	ancien := etapeReecriture
	etapeReecriture = f
	t.Cleanup(func() { etapeReecriture = ancien })
}

// baseComplete : le chemin porte une base DuckDB lisible dont la table témoin a `attendu` lignes.
// Ouverture en LECTURE SEULE : elle ne modifie pas le fichier (la réécriture le surveille).
func baseComplete(path string, attendu int64) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("aucun fichier au chemin de la base : %w", err)
	}
	h, err := duckdb.OpenReadOnly(path)
	if err != nil {
		return fmt.Errorf("base illisible : %w", err)
	}
	defer h.Close() //nolint:errcheck
	var n int64
	if err := h.SQLDb().QueryRow(`SELECT COUNT(*) FROM match_bomb_stats_latest`).Scan(&n); err != nil {
		return fmt.Errorf("base incomplète : %w", err)
	}
	if n != attendu {
		return fmt.Errorf("base : %d lignes servies, attendu %d", n, attendu)
	}
	return nil
}

func aucunReste(t *testing.T, path string) {
	t.Helper()
	for _, motif := range []string{filepath.Base(path) + ".reecriture*", "*avant-reecriture*"} {
		if s := fichiers(t, filepath.Dir(path), motif); len(s) != 0 {
			t.Errorf("fichier laissé : %v", s)
		}
	}
}

// TestReecriture_SauvegardeSurUnAutreVolume : `--backup-dir` sur un autre volume. Aucun second
// volume n'est garanti sur un poste de test : le rename est SIMULÉ par `renommer`, qui échoue
// comme `rename(2)` (EXDEV) ou `MoveFileEx` sans MOVEFILE_COPY_ALLOWED dès que la source et la
// cible ne sont pas dans le même dossier. La réécriture doit réussir et ne rien laisser.
func TestReecriture_SauvegardeSurUnAutreVolume(t *testing.T) {
	path := baseDeTest(t)
	autre := t.TempDir()
	ancien := renommer
	renommer = func(src, dst string) error {
		if filepath.Dir(src) != filepath.Dir(dst) {
			return fmt.Errorf("rename %s -> %s : volume différent (simulé)", src, dst)
		}
		return os.Rename(src, dst)
	}
	defer func() { renommer = ancien }()

	err := compacterTitre(context.Background(), "halo_infinite", path,
		compactPassesOptions{rewriteFile: true, backupDir: autre})
	if err != nil {
		t.Fatalf("réécriture avec --backup-dir sur un autre volume : %v", err)
	}
	if err := baseComplete(path, 3); err != nil {
		t.Fatal(err)
	}
	aucunReste(t, path)
	s := fichiers(t, autre, "*avant-reecriture*")
	if len(s) != 1 {
		t.Fatalf("sauvegarde d'avant réécriture dans %s : %v, attendu une", autre, s)
	}
	if err := baseComplete(s[0], 3); err != nil {
		t.Fatalf("la sauvegarde n'est pas la base compactée : %v", err)
	}
}

var errArret = errors.New("arrêt simulé")

// TestReecriture_JamaisSansBaseAuChemin : à CHAQUE étape, le chemin porte une base complète ; et
// un arrêt à chaque étape laisse la base complète, sans fichier temporaire.
func TestReecriture_JamaisSansBaseAuChemin(t *testing.T) {
	etapes := []string{etapeApresCopie, etapeApresInventaire, etapeApresSauvegarde, etapeApresVerification}
	for _, arret := range append(etapes, "") {
		t.Run("arret="+arret, func(t *testing.T) {
			path := baseDeTest(t)
			if err := compacterTitre(context.Background(), "halo_infinite", path, compactPassesOptions{}); err != nil {
				t.Fatal(err)
			}
			var vues []string
			avecEtape(t, func(etape string) error {
				vues = append(vues, etape)
				if err := baseComplete(path, 3); err != nil {
					t.Errorf("étape %s : %v", etape, err)
				}
				if etape == arret {
					return errArret
				}
				return nil
			})
			err := reecrireFichier(context.Background(), path, "")
			if arret != "" && !errors.Is(err, errArret) {
				t.Fatalf("attendu l'arrêt simulé, got %v", err)
			}
			if arret == "" && err != nil {
				t.Fatalf("réécriture : %v (étapes %v)", err, vues)
			}
			if err := baseComplete(path, 3); err != nil {
				t.Fatalf("après l'arrêt à %q : %v", arret, err)
			}
			if arret != "" {
				aucunReste(t, path)
			}
		})
	}
}

// TestReecriture_SourceModifieeApresLaCopie : une écriture dans la base entre la copie et
// l'échange (un serveur qui démarre) fait REFUSER l'échange ; l'écriture est conservée.
func TestReecriture_SourceModifieeApresLaCopie(t *testing.T) {
	path := baseDeTest(t)
	avecEtape(t, func(etape string) error {
		if etape != etapeApresSauvegarde {
			return nil
		}
		h, err := duckdb.OpenReadWrite(path)
		if err != nil {
			return err
		}
		defer h.Close() //nolint:errcheck
		_, err = h.SQLDb().Exec(`INSERT INTO match_bomb_stats (match_id, xuid) VALUES ('m9', 'x9')`)
		return err
	})
	err := compacterTitre(context.Background(), "halo_infinite", path, compactPassesOptions{rewriteFile: true})
	if err == nil || !strings.Contains(err.Error(), "a changé depuis la copie") {
		t.Fatalf("attendu un refus (source modifiée), got %v", err)
	}
	if err := baseComplete(path, 4); err != nil {
		t.Fatalf("l'écriture faite entre la copie et l'échange est perdue : %v", err)
	}
	aucunReste(t, path)
}

// TestReecriture_SourceTenueAvantLEchange : un autre processus tient la base au moment de
// l'échange — refus, base intacte, aucun fichier temporaire.
func TestReecriture_SourceTenueAvantLEchange(t *testing.T) {
	path := baseDeTest(t)
	var liberer func()
	avecEtape(t, func(etape string) error {
		if etape == etapeApresSauvegarde {
			liberer = tenirLaBase(t, path)
		}
		return nil
	})
	err := compacterTitre(context.Background(), "halo_infinite", path, compactPassesOptions{rewriteFile: true})
	if liberer != nil {
		liberer()
	}
	if err == nil || !strings.Contains(err.Error(), "tenue par un autre processus") {
		t.Fatalf("attendu un refus (source tenue), got %v", err)
	}
	if err := baseComplete(path, 3); err != nil {
		t.Fatal(err)
	}
	aucunReste(t, path)
}
