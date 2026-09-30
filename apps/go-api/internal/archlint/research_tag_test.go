package archlint

// research_tag_test.go — LE GARDE-RAIL DU TAG `research`, SUR TOUT LE MODULE (J12.7, DU-5 a,
// 2026-09-30).
//
// CE QUE LE TAG PAIE. Les instruments de recherche (rebalayage des bobines, sondes de grammaire,
// outils de rétro-ingénierie) ne sont pas des tests de non-régression : ils tournent à la
// demande (`go test -tags research ...`), jamais dans le build par défaut. Sans tag, 383
// fichiers `*_research_test.go` dont 220 non tagués allongeaient chaque `go test` et
// alimentaient la baseline de présence. La CI les COMPILE (`go vet -tags=research ./...`) : un
// tag sans vet est un fichier qui pourrit en silence.
//
// DEUX SENS, comme `gamefiles_tag_test.go` :
//   - tout `*_research_test.go` porte `//go:build research` en ligne 1 (ou une contrainte
//     combinée `//go:build research && X`) ;
//   - tout fichier qui porte le tag est bien un instrument : un `_test.go` (les `*_research_test.go`
//     et leurs COMPAGNONS — mesures et helpers qui ne compilent qu en leur compagnie, tagués par
//     fermeture au J12.7 ; une garde réelle qui partage un helper est extraite, pas taguée), ou un
//     fichier sous un dossier d instruments (`film/research/`, `tools/film_re/`). Sans ce sens,
//     du code de production pourrait se cacher derrière le tag et sortir du build par défaut.
//
// Mutation qui doit le faire rougir : retirer la première ligne d'un `*_research_test.go`
// (sens 1), ou poser `//go:build research` sur un fichier de production (sens 2).

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tagResearch : la ligne exacte attendue en tête de chaque instrument.
const tagResearch = "//go:build research"

// instrumentsResearchPlancher : le corpus mesuré le 2026-09-30 (383 `*_research_test.go`).
// Un balayage qui rend moins ne garde plus rien.
const instrumentsResearchPlancher = 383

// dossiersInstruments : sous ces préfixes (chemins relatifs à `apps/go-api`, en slash), tout
// fichier tagué est un instrument par construction.
var dossiersInstruments = []string{
	"internal/games/halo_infinite/film/research/",
	"tools/film_re/",
}

// contrainteResearch rend vrai si la première ligne est `//go:build` et cite `research`
// comme terme d'une conjonction (`research` seul ou `research && X`).
func contrainteResearch(texte string) bool {
	ligne, _, _ := strings.Cut(texte, "\n")
	ligne = strings.TrimSpace(ligne)
	if ligne == tagResearch {
		return true
	}
	if !strings.HasPrefix(ligne, "//go:build ") || strings.Contains(ligne, "||") {
		return false
	}
	for _, terme := range strings.Split(strings.TrimPrefix(ligne, "//go:build "), "&&") {
		if strings.TrimSpace(terme) == "research" {
			return true
		}
	}
	return false
}

// balayerGo appelle `visiter` pour CHAQUE `.go` du module (mêmes filtres de dossiers que
// `balayerTests`).
func balayerGo(t *testing.T, visiter func(rel, texte string)) {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici)))
	err := filepath.WalkDir(goAPIRoot, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != goAPIRoot && (dossiersInvisiblesAuGo[nom] ||
				strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		buf, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
		if rerr != nil {
			return rerr
		}
		rel, _ := filepath.Rel(goAPIRoot, chemin)
		visiter(filepath.ToSlash(rel), string(buf))
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module (%s) : %v", goAPIRoot, err)
	}
}

// TestInstrumentsResearchSontTagues — chaque `*_research_test.go` du MODULE porte le tag.
func TestInstrumentsResearchSontTagues(t *testing.T) {
	vus := 0
	balayerGo(t, func(rel, texte string) {
		if !strings.HasSuffix(rel, "_research_test.go") {
			return
		}
		vus++
		if !contrainteResearch(texte) {
			t.Errorf("%s ne commence pas par %q (ni une conjonction `research && X`) — sans ce tag "+
				"l'instrument tourne dans le build par défaut", rel, tagResearch)
		}
	})
	if vus < instrumentsResearchPlancher {
		t.Errorf("%d fichier(s) *_research_test.go balayé(s), plancher %d (mesure du 2026-09-30) "+
			"— le garde-rail ne garde plus le corpus entier", vus, instrumentsResearchPlancher)
	}
}

// TestFichierTagueResearchEstUnInstrument — l'autre sens : le tag ne cache pas de production.
func TestFichierTagueResearchEstUnInstrument(t *testing.T) {
	balayerGo(t, func(rel, texte string) {
		if !contrainteResearch(texte) || strings.HasSuffix(rel, "_test.go") {
			return
		}
		for _, dossier := range dossiersInstruments {
			if strings.HasPrefix(rel, dossier) {
				return
			}
		}
		t.Errorf("%s porte %q sans être un instrument (un _test.go, ou un dossier %v"+
			") — du code de production sortirait du build par défaut",
			rel, tagResearch, dossiersInstruments)
	})
}
