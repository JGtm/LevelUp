package analysis

// team_score_label_guard_test.go — LE GARDE-RAIL QUI EMPÊCHE LA 6e COPIE.
//
// Une factorisation sans garde-rail re-diverge : c'est la leçon écrite du dépôt (le prédicat
// bot est passé de 8 à 36 copies APRÈS avoir été centralisé, cf. règle 6 du CLAUDE.md). Le
// libellé de score d'équipe venait d'en faire la démonstration à petite échelle — cinq
// fabrications, deux formats, deux sources — et rien n'aurait empêché la sixième.
//
// CE QUE CE TEST SCANNE, ET POURQUOI PAS TOUT LE DÉPÔT. Le périmètre est
// `internal/analysis` + `internal/service` hors fichiers de test : les couches qui
// fabriquent des libellés de score. Un scan global mordrait sur les outils de diagnostic
// (`cmd/diag_*` impriment des `%d-%d` de mise au point) et sur les messages d'erreur de
// tests, sans rien protéger de plus — la sixième copie naîtrait dans un service, pas dans un
// CLI jetable.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// formatsInterdits : les deux gabarits par lesquels un score « X - Y » se fabrique. Toute
// occurrence hors du fichier canonique est une copie qui va diverger.
var formatsInterdits = []string{`%d-%d`, `%d - %d`}

// fichierCanonique porte l'unique Sprintf autorisé.
const fichierCanonique = "team_score_display.go"

// racinesScannees : les couches qui produisent des libellés destinés à l'UI.
var racinesScannees = []string{".", "../service"}

var reCommentaireLigne = regexp.MustCompile(`//[^\n]*`)
var reCommentaireBloc = regexp.MustCompile(`(?s)/\*.*?\*/`)

// sansCommentaires retire les commentaires : ce fichier-ci, comme les cinq appelants
// migrés, NOMME les formats interdits pour expliquer pourquoi on ne les emploie plus.
func sansCommentaires(src string) string {
	return reCommentaireLigne.ReplaceAllString(reCommentaireBloc.ReplaceAllString(src, ""), "")
}

func TestPasDeSixiemeCopieDuLibelleDeScore(t *testing.T) {
	scannes := 0
	for _, racine := range racinesScannees {
		err := filepath.Walk(racine, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			if strings.HasSuffix(path, "_test.go") || filepath.Base(path) == fichierCanonique {
				return nil
			}
			body, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				return err
			}
			scannes++
			code := sansCommentaires(string(body))
			for _, format := range formatsInterdits {
				if strings.Contains(code, format) {
					t.Errorf("%s fabrique un libellé de score au format %q.\n"+
						"Il n'y a qu'une source pour ce libellé : analysis.TeamScoreLabel "+
						"(%s). Cinq copies avaient déjà divergé sur deux formats — la règle "+
						"« manches plutôt que points » ne tient que si elle est posée à un "+
						"seul endroit.", path, format, fichierCanonique)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", racine, err)
		}
	}
	// Un scan qui ne lit rien passerait toujours : on exige d'avoir vu les deux couches.
	if scannes < 50 {
		t.Fatalf("%d fichiers scannés — le garde-rail ne voit plus les couches analysis/service", scannes)
	}
}

// TestGardeRailLibelleMord prouve que le scan détecte une vraie copie et laisse passer un
// commentaire qui se contente de nommer le format. Un garde-rail qui ne mord jamais est un
// garde-rail fantôme.
func TestGardeRailLibelleMord(t *testing.T) {
	copie := "label := fmt.Sprintf(\"%d - %d\", a, b)"
	if !strings.Contains(sansCommentaires(copie), "%d - %d") {
		t.Error("le scan ne détecte pas une copie réelle")
	}
	commentaire := "// on n'écrit plus %d-%d ni %d - %d ici\nconst x = 1"
	code := sansCommentaires(commentaire)
	for _, format := range formatsInterdits {
		if strings.Contains(code, format) {
			t.Errorf("le scan mord sur un COMMENTAIRE contenant %q — faux positif", format)
		}
	}
}
