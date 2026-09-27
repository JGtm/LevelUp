//go:build cgo

package main

// cmd_compact_passes_rewrite_test.go — `--rewrite-file` sous verrou tenu (C.10) : le verrou DuckDB
// de la base est tenu du début à la fin (un tiers est refusé à chaque étape), jamais de fenêtre
// sans base au chemin, remplacement propre à chaque système, WAL étranger refusé d'entrée,
// sauvegarde sur un autre volume, aucun reste sur échec.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// baseComplete : le chemin porte une base DuckDB lisible dont la table témoin sert `attendu`
// lignes (lecture seule : à appeler quand la commande ne tient plus la base).
func baseComplete(path string, attendu int64) error {
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

// aucunReste : ni fichier neuf, ni sauvegarde d'avant réécriture, dans les dossiers donnés.
func aucunReste(t *testing.T, path string, dossiers ...string) {
	t.Helper()
	for _, d := range append([]string{filepath.Dir(path)}, dossiers...) {
		for _, motif := range []string{"*" + suffixeFichierReecriture + "*", "*avant-reecriture*"} {
			if s := fichiers(t, d, motif); len(s) != 0 {
				t.Errorf("fichier laissé : %v", s)
			}
		}
	}
}

// baseCompacteeDeTest : une base au schéma réel, déjà compactée (3 lignes servies et brutes).
func baseCompacteeDeTest(t *testing.T) string {
	t.Helper()
	path := baseDeTest(t)
	if err := compacterTitre(context.Background(), "halo_infinite", path, compactPassesOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, s := range fichiers(t, filepath.Dir(path), "*avant-compaction*") {
		_ = os.Remove(s) // sauvegarde de la compaction préparatoire : hors du sujet
	}
	return path
}

// envEssayerLaBase : chemin de la base que le processus auxiliaire TENTE d'ouvrir en écriture.
const envEssayerLaBase = "LEVELUP_TEST_ESSAYER_BASE"

// TestAideEssayerLaBase n'est pas un test : c'est le PROCESSUS AUXILIAIRE qui tente d'ouvrir la
// base en écriture (comme un serveur qui démarre) et écrit « ouvert » ou « refuse ».
func TestAideEssayerLaBase(t *testing.T) {
	path := os.Getenv(envEssayerLaBase)
	if path == "" {
		t.Skip("processus auxiliaire de TestReecriture_TiersRefuseDuDebutALaFin")
	}
	db, err := sql.Open("duckdb", path)
	if err == nil {
		_, err = db.Exec(`SELECT count(*) FROM match_bomb_stats`)
		_ = db.Close()
	}
	if err != nil {
		fmt.Println("refuse")
		return
	}
	fmt.Println("ouvert")
}

// tiersPeutOuvrir lance le processus auxiliaire et dit s'il a pu ouvrir la base.
func tiersPeutOuvrir(t *testing.T, path string) bool {
	t.Helper()
	aux := exec.Command(os.Args[0], "-test.run=^TestAideEssayerLaBase$")
	aux.Env = append(os.Environ(), envEssayerLaBase+"="+path)
	sortie, err := aux.Output()
	if err != nil {
		t.Fatalf("processus auxiliaire : %v (%s)", err, sortie)
	}
	switch {
	case strings.Contains(string(sortie), "ouvert"):
		return true
	case strings.Contains(string(sortie), "refuse"):
		return false
	}
	t.Fatalf("processus auxiliaire : sortie inattendue %q", sortie)
	return false
}

// TestReecriture_TiersRefuseDuDebutALaFin : à chaque étape avant le remplacement — y compris
// juste avant lui — un autre processus ne peut pas ouvrir la base : le verrou est TENU.
func TestReecriture_TiersRefuseDuDebutALaFin(t *testing.T) {
	path := baseCompacteeDeTest(t)
	sousVerrou := map[string]bool{etapeApresCopie: true, etapeApresInventaire: true,
		etapeApresSauvegarde: true, etapeAvantRemplacement: true}
	var vues []string
	avecEtape(t, func(etape string) error {
		if sousVerrou[etape] {
			vues = append(vues, etape)
			if tiersPeutOuvrir(t, path) {
				t.Errorf("étape %s : un autre processus a pu ouvrir la base", etape)
			}
		}
		return nil
	})
	if err := reecrireFichier(context.Background(), path, ""); err != nil {
		t.Fatalf("réécriture : %v", err)
	}
	if len(vues) != len(sousVerrou) {
		t.Fatalf("étapes observées %v, attendu les %d étapes sous verrou", vues, len(sousVerrou))
	}
	if err := baseComplete(path, 3); err != nil {
		t.Fatal(err)
	}
}

// TestReecriture_JamaisSansBaseAuChemin : à CHAQUE étape un fichier est au chemin de la base ; un
// arrêt à chaque étape laisse une base complète et aucun reste (sauf après le remplacement, où la
// base neuve est en place et la sauvegarde gardée) ; aucun WAL de l'ancienne base ne reste.
func TestReecriture_JamaisSansBaseAuChemin(t *testing.T) {
	var etapes []string
	t.Run("sans-arret", func(t *testing.T) {
		path := baseCompacteeDeTest(t)
		avecEtape(t, func(etape string) error {
			etapes = append(etapes, etape)
			if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
				t.Errorf("étape %s : aucun fichier au chemin de la base (%v)", etape, err)
			}
			return nil
		})
		if err := reecrireFichier(context.Background(), path, ""); err != nil {
			t.Fatal(err)
		}
		if err := baseComplete(path, 3); err != nil {
			t.Fatal(err)
		}
		if fi, err := os.Stat(path + ".wal"); err == nil {
			t.Errorf("un WAL (%d octets) est resté à côté de la base réécrite", fi.Size())
		}
	})
	for _, arret := range etapes {
		if arret == etapeApresEchange {
			continue
		}
		t.Run("arret="+arret, func(t *testing.T) {
			path := baseCompacteeDeTest(t)
			avecEtape(t, func(etape string) error {
				if etape == arret {
					return errArret
				}
				return nil
			})
			if err := reecrireFichier(context.Background(), path, ""); !errors.Is(err, errArret) {
				t.Fatalf("attendu l'arrêt simulé, got %v", err)
			}
			if err := baseComplete(path, 3); err != nil {
				t.Fatalf("après l'arrêt à %q : %v", arret, err)
			}
			if arret != etapeApresRemplacement {
				aucunReste(t, path)
			}
		})
	}
}

var errArret = errors.New("arrêt simulé")

// TestReecriture_TiersTientLaBaseAuRemplacement_Windows : un tiers ouvre la base entre notre
// fermeture et notre rename (point d'observation `apres-fermeture`) — Windows refuse de remplacer
// un fichier tenu : refus, base intacte, aucun reste.
func TestReecriture_TiersTientLaBaseAuRemplacement_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("variante Windows (fermeture puis rename) ; la variante POSIX renomme sous verrou")
	}
	path := baseCompacteeDeTest(t)
	var liberer func()
	avecEtape(t, func(etape string) error {
		if etape == etapeApresFermeture {
			liberer = tenirLaBase(t, path)
		}
		return nil
	})
	err := reecrireFichier(context.Background(), path, "")
	if liberer != nil {
		liberer()
	}
	if err == nil || !strings.Contains(err.Error(), "n'a pas pu être remplacée") {
		t.Fatalf("attendu un refus au remplacement, got %v", err)
	}
	if err := baseComplete(path, 3); err != nil {
		t.Fatal(err)
	}
	aucunReste(t, path)
}

// TestReecriture_RemplacementSousVerrou_POSIX : hors Windows, le rename se fait PENDANT que la
// connexion tient l'ancienne base, et un tiers est refusé jusqu'au remplacement. Sous Windows ce
// test ne peut que sauter (un fichier tenu ne s'y remplace pas) : il n'est prouvé que par la CI.
func TestReecriture_RemplacementSousVerrou_POSIX(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows refuse de remplacer un fichier tenu (« Accès refusé », mesuré) : " +
			"variante POSIX prouvée par la CI Linux seulement")
	}
	path := baseCompacteeDeTest(t)
	neuf := path + suffixeFichierReecriture
	var sousVerrou bool
	avecEtape(t, func(etape string) error {
		switch etape {
		case etapeAvantRemplacement:
			if tiersPeutOuvrir(t, path) {
				t.Error("un tiers a pu ouvrir la base avant le remplacement")
			}
		case etapeApresRemplacement:
			_, errNeuf := os.Stat(neuf)
			sousVerrou = os.IsNotExist(errNeuf)
		}
		return nil
	})
	if err := reecrireFichier(context.Background(), path, ""); err != nil {
		t.Fatal(err)
	}
	if !sousVerrou {
		t.Fatal("le rename n'a pas eu lieu pendant que la connexion était ouverte")
	}
	if err := baseComplete(path, 3); err != nil {
		t.Fatal(err)
	}
}

// TestReecriture_WALNonVideAuLancement : un processus tué avant son CHECKPOINT a laissé un WAL
// non vide — refus d'entrée ; le WAL et la base sont intacts, et ses transactions ne sont pas
// perdues (la ligne est servie dès que la base est rouverte normalement).
func TestReecriture_WALNonVideAuLancement(t *testing.T) {
	path := baseCompacteeDeTest(t)
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`PRAGMA disable_checkpoint_on_shutdown`,
		`INSERT INTO match_bomb_stats (match_id, xuid) VALUES ('m9', 'x9')`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path + ".wal")
	if err != nil || fi.Size() == 0 {
		t.Fatalf("fixture : WAL non vide attendu (%v)", err)
	}
	err = reecrireFichier(context.Background(), path, "")
	if err == nil || !strings.Contains(err.Error(), "wal non vide") {
		t.Fatalf("attendu un refus d'entrée sur le WAL, got %v", err)
	}
	if apres, _ := os.Stat(path + ".wal"); apres == nil || apres.Size() != fi.Size() {
		t.Fatal("le WAL étranger a été touché")
	}
	aucunReste(t, path)
	if err := baseComplete(path, 4); err != nil {
		t.Fatalf("transaction du WAL perdue : %v", err)
	}
}

// TestReecriture_SauvegardeSurUnAutreVolume : `--backup-dir` sur un autre volume. Aucun second
// volume n'est garanti sur un poste de test : le rename est SIMULÉ par `renommer`, qui échoue
// comme `rename(2)` (EXDEV) ou `MoveFileEx` sans MOVEFILE_COPY_ALLOWED dès que la source et la
// cible ne sont pas dans le même dossier. La sauvegarde (COPY FROM DATABASE) n'est jamais un
// rename ; le seul rename reste dans le dossier de la base.
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

// TestCopierFichier_EchecEnCoursDeCopieNeLaisseRien : une copie de sauvegarde qui échoue APRÈS la
// création de la cible (ici : la « base » est un dossier, sa lecture échoue) ne laisse aucun
// fichier tronqué sous un nom de sauvegarde.
func TestCopierFichier_EchecEnCoursDeCopieNeLaisseRien(t *testing.T) {
	dir := t.TempDir()
	faux := filepath.Join(dir, "base.duckdb")
	if err := os.Mkdir(faux, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if _, err := copierFichier(context.Background(), faux, dest, "avant-compaction"); err == nil {
		t.Fatal("copie d'un dossier acceptée")
	}
	if s := fichiers(t, dest, "*"); len(s) != 0 {
		t.Fatalf("copie partielle laissée : %v", s)
	}
}
