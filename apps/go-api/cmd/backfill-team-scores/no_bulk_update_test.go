package main

// no_bulk_update_test.go — GARDE-RAIL LOCAL ANTI-ART, posé le 2026-08-24.
//
// POURQUOI IL FAUT CELUI-CI EN PLUS. Le ratchet du dépôt,
// `internal/sync/no_art_patterns_test.go`, EXCLUT explicitement tout chemin `/cmd/` de ses
// trois scans (lignes 220, 296, 452 : « CLI/scripts one-shot »). Un
// `UPDATE match_registry … FROM (VALUES …)` écrit ICI passerait donc tous les gates du
// dépôt sans être vu — c'est le constat P2-1 de la revue adversariale du 2026-08-24.
//
// Ce test rétablit la protection là où elle manque, sur le seul périmètre de ce paquet.
// La forme interdite n'est pas une coquetterie : un UPDATE bulk multi-lignes touche N
// entrées d'index en un statement, ce qui déclenche le bug DuckDB ART #23046
// (« Failed to delete all rows from index ») — celui qui a corrompu des bases en prod
// (ADR 0019 / 0026 / 0030).

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Les commentaires sont retirés avant le scan : ce fichier-ci comme `registry.go` ou
// `main.go` NOMMENT les formes interdites pour expliquer pourquoi on ne les emploie pas.
// Un commentaire explicatif n'est pas une écriture — même convention que le ratchet du
// dépôt (`no_art_patterns_test.go`, `reBlockComment`/`reLineComment`).
var (
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLineComment  = regexp.MustCompile(`//[^\n]*`)
)

func stripComments(src string) string {
	return reLineComment.ReplaceAllString(reBlockComment.ReplaceAllString(src, ""), "")
}

// formesInterdites : littéraux SQL qui ne doivent JAMAIS apparaître dans ce paquet.
//
//   - `FROM (VALUES` : l'UPDATE bulk multi-lignes, déclencheur ART direct.
//   - `ON CONFLICT`  : l'UPSERT concurrent, l'autre déclencheur documenté. Ce backfill
//     n'en a aucun besoin (il UPDATE une ligne existante) ; son apparition
//     signalerait une dérive vers un chemin d'écriture non prévu.
var formesInterdites = []string{
	"FROM (VALUES",
	"ON CONFLICT",
}

func TestPasDeFormeArtDansCePaquet(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	scanned := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		// Ce fichier-ci porte les littéraux dans `formesInterdites` par construction :
		// les scanner reviendrait à échouer sur sa propre définition.
		if e.Name() == "no_bulk_update_test.go" {
			continue
		}
		body, err := os.ReadFile(filepath.Clean(e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		scanned++
		upper := strings.ToUpper(stripComments(string(body)))
		for _, forme := range formesInterdites {
			if strings.Contains(upper, forme) {
				t.Errorf("%s contient la forme SQL interdite %q.\n"+
					"Sur une table indexée elle déclenche le bug ART DuckDB #23046. "+
					"Utiliser N UPDATE row-by-row `WHERE match_id = ?` (cf. registry.go WriteScores). "+
					"Le ratchet du dépôt ne couvre PAS cmd/ — ce test est la seule protection ici.",
					e.Name(), forme)
			}
		}
	}
	// Un scan qui ne lit rien passerait toujours : on exige d'avoir vu les fichiers du paquet.
	if scanned < 4 {
		t.Fatalf("%d fichiers .go scannés — le garde-rail ne voit plus le paquet (fichiers renommés ?)", scanned)
	}
}

// TestGardeRailMord valide que le scan DÉTECTE réellement une violation et laisse passer
// un commentaire qui se contente de la nommer. Un garde-rail qui ne mord jamais est un
// garde-rail fantôme — c'est exactement le reproche fait au ratchet du dépôt sur `cmd/`.
func TestGardeRailMord(t *testing.T) {
	violation := "q := `UPDATE match_registry SET team_0_score = v.t0 FROM (VALUES (1, 2)) AS v(t0, t1)`"
	if !strings.Contains(strings.ToUpper(stripComments(violation)), "FROM (VALUES") {
		t.Error("le scan ne détecte pas un UPDATE bulk réel")
	}
	commentaire := "// on n'utilise jamais FROM (VALUES ni ON CONFLICT ici\nconst x = 1"
	up := strings.ToUpper(stripComments(commentaire))
	for _, forme := range formesInterdites {
		if strings.Contains(up, forme) {
			t.Errorf("le scan mord sur un COMMENTAIRE contenant %q — faux positif", forme)
		}
	}
}

// TestFormeDEcritureEstRowByRow fige la forme attendue de l'UPDATE : un seul
// `WHERE match_id = ?`, trois placeholders, et rien qui ressemble à du set-based.
func TestFormeDEcritureEstRowByRow(t *testing.T) {
	if !strings.Contains(updateScoresSQL, "WHERE match_id = ?") {
		t.Errorf("updateScoresSQL doit cibler UNE ligne par `WHERE match_id = ?` : %q", updateScoresSQL)
	}
	if n := strings.Count(updateScoresSQL, "?"); n != 3 {
		t.Errorf("updateScoresSQL a %d placeholders, 3 attendus (team_0, team_1, match_id) : %q", n, updateScoresSQL)
	}
}
