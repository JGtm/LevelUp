package analysis

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Ratchet du pont transitoire ouvert a l'item 2.5.h (voir `highlight_event_pont_film.go`).
//
// Le pont n'existe que pour les fichiers de `internal/games/halo_infinite/film/` qui
// citent encore `analysis.HighlightEvent` ou `analysis.EventType*` : ni la frontiere de
// fichiers du lot (2.5.b requalifie `film/` en parallele) ni l'immobilite de `GrammarRev`
// (`film/facts/killsource` est une racine hachee) ne permettaient de les re-pointer.
//
// Ce test est ce qui empeche le pont de devenir un garde-rail eternel : le jour ou le lot
// 2.5.e re-pointe le dernier de ces fichiers derriere la facade exportee, le compte tombe
// a zero et CE TEST ECHOUE en demandant la suppression du pont. Meme mecanique que les
// allowlists datees du ratchet des couches, qui rougissent sur une entree devenue sans
// objet.
var citationPontTempsFort = regexp.MustCompile(`analysis\.(HighlightEvent|EventTypeKill|EventTypeDeath|EventTypeMedal|EventTypeMode)\b`)

func TestPontTempsFortVersFilmNEstPasPerime(t *testing.T) {
	racine := racineFilmDepuisAnalysis(t)
	fichiers, total := citationsDuPontDansFilm(t, racine)

	if total == 0 {
		t.Fatalf("plus aucun fichier de %s ne cite analysis.HighlightEvent ni analysis.EventType* : "+
			"le pont transitoire de l'item 2.5.h est PERIME — supprimer "+
			"internal/analysis/highlight_event_pont_film.go et ce test", racine)
	}
	t.Logf("pont encore requis : %d citations dans %d fichiers de film/ — %s",
		total, len(fichiers), strings.Join(fichiers, ", "))
}

// racineFilmDepuisAnalysis resout `internal/games/halo_infinite/film` depuis l'emplacement
// de ce fichier, sans dependre du repertoire de travail des tests.
func racineFilmDepuisAnalysis(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	internalDir := filepath.Dir(filepath.Dir(ici)) // .../internal/analysis -> .../internal
	racine := filepath.Join(internalDir, "games", "halo_infinite", "film")
	if _, err := os.Stat(racine); err != nil {
		t.Fatalf("racine film introuvable (%s) : %v", racine, err)
	}
	return racine
}

// citationsDuPontDansFilm compte les citations HORS COMMENTAIRE des cinq symboles du
// pont, sources de test comprises. Les commentaires sont exclus : une reference
// documentaire ne fait pas compiler un fichier, donc ne justifie pas le pont.
func citationsDuPontDansFilm(t *testing.T, racine string) ([]string, int) {
	t.Helper()
	vus := map[string]int{}
	total := 0

	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		blob, errLire := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis la racine du module
		if errLire != nil {
			return errLire
		}
		for _, ligne := range strings.Split(string(blob), "\n") {
			if strings.HasPrefix(strings.TrimSpace(ligne), "//") {
				continue
			}
			n := len(citationPontTempsFort.FindAllString(ligne, -1))
			if n == 0 {
				continue
			}
			rel, _ := filepath.Rel(racine, chemin)
			vus[filepath.ToSlash(rel)] += n
			total += n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}

	fichiers := make([]string, 0, len(vus))
	for f := range vus {
		fichiers = append(fichiers, f)
	}
	sort.Strings(fichiers)
	return fichiers, total
}
