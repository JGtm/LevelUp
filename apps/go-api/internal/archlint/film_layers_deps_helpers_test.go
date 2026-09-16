package archlint

// film_layers_deps_helpers_test.go — LA LECTURE DU GRAPHE D IMPORTS pour
// `film_layers_deps_test.go` (item 2.5.0). Ce fichier ne porte AUCUNE regle ni AUCUNE table : les
// regles, la table des couches et les allowlists datees vivent dans le fichier voisin, qui est le
// seul que le lot 2.5 aura a modifier. Ici, seulement comment on va chercher les imports.
//
// Le graphe se lit par `go/parser` (ImportsOnly) sur les fichiers de PRODUCTION, pas par
// `go list -json` : `go list` charge le module entier et coute des dizaines de secondes, la ou le
// parcours de ~380 fichiers en ImportsOnly tient en une fraction de seconde. Un ratchet qu on
// hesite a lancer ne garde rien.

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// motifsDeViolationDeCouche rend les motifs pour lesquels l arete `de -> vers` sort des regles,
// ou nil. Les deux regles sont independantes : une arete peut violer les deux.
func motifsDeViolationDeCouche(de, vers string) []string {
	var motifs []string
	if souscheminDeFilm(vers, racineAnalysisFilm) {
		motifs = append(motifs, "R2 (le lieu) : une couche du decodeur depend de "+
			racineAnalysisFilm+", qui est title-agnostic (ADR 0012, ADR 0025)")
	}
	cDe, okDe := couchesDuDecodeur[de]
	cVers, okVers := couchesDuDecodeur[vers]
	if okDe && okVers && cDe.rang >= 0 && cVers.rang >= 0 && cVers.rang > cDe.rang {
		motifs = append(motifs, "R1 (le sens) : import vers le HAUT, "+cDe.nom+" -> "+cVers.nom)
	}
	return motifs
}

// cleArete : la forme canonique d une arete dans les messages et les index.
func cleArete(de, vers string) string { return de + " -> " + vers }

// souscheminDeFilm dit si `chemin` est `racine` ou vit dessous.
func souscheminDeFilm(chemin, racine string) bool {
	return chemin == racine || strings.HasPrefix(chemin, racine+"/")
}

// balayerPaquetsDuDecodeur rend, par paquet surveille (chemin relatif a `apps/go-api`, en
// slash), la liste triee de ses imports DU DEPOT eux aussi en chemin relatif, plus le nombre
// total de fichiers de production parcourus.
//
// Paquets surveilles = tous ceux qui portent du `.go` de production sous `film/`, plus toutes
// les cles de `couchesDuDecodeur` (ce qui couvre les couches encore posees hors du decodeur).
func balayerPaquetsDuDecodeur(t *testing.T) (map[string][]string, int) {
	t.Helper()
	racineAPI := apiRootDepuisIci(t)
	surveilles := map[string]bool{}
	for _, rel := range repertoiresDeProductionSousFilm(t, racineAPI) {
		surveilles[rel] = true
	}
	for rel := range couchesDuDecodeur {
		surveilles[rel] = true
	}
	graphe := make(map[string][]string, len(surveilles))
	total := 0
	for _, rel := range clesTrieesFilm(surveilles) {
		imports, n := importsDeProductionDuPaquet(t, racineAPI, rel)
		graphe[rel] = imports
		total += n
	}
	return graphe, total
}

// repertoiresDeProductionSousFilm rend les repertoires portant du `.go` de production sous
// `film/`, en chemin relatif a `apps/go-api`.
func repertoiresDeProductionSousFilm(t *testing.T, racineAPI string) []string {
	t.Helper()
	base := filepath.Join(racineAPI, filepath.FromSlash(racineDecodeurFilm))
	vus := map[string]bool{}
	err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != base && (strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		rel, rerr := filepath.Rel(racineAPI, filepath.Dir(chemin))
		if rerr != nil {
			return rerr
		}
		vus[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v — si le decodeur a DEMENAGE, deplacer `racineDecodeurFilm` "+
			"avec lui", base, err)
	}
	if len(vus) == 0 {
		t.Fatalf("aucun paquet de production sous %s : ce ratchet n aurait plus d objet",
			racineDecodeurFilm)
	}
	return clesTrieesFilm(vus)
}

// importsDeProductionDuPaquet parse les imports (go/parser, ImportsOnly) des fichiers non-test
// d un paquet et rend ceux du depot, en chemin relatif, tries et dedoublonnes. Un `grep` se
// ferait tromper par les chemins de paquet cites en commentaire, qui abondent ici.
func importsDeProductionDuPaquet(t *testing.T, racineAPI, rel string) ([]string, int) {
	t.Helper()
	dir := filepath.Join(racineAPI, filepath.FromSlash(rel))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s il a DEMENAGE, corriger `couchesDuDecodeur` "+
			"dans le commit qui le deplace", rel, err)
	}
	fset := token.NewFileSet()
	vus := map[string]bool{}
	fichiers := 0
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		fichiers++
		f, perr := parser.ParseFile(fset, filepath.Join(dir, nom), nil, parser.ImportsOnly)
		if perr != nil {
			t.Fatalf("analyse de %s/%s : %v", rel, nom, perr)
		}
		for _, imp := range f.Imports {
			chemin := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(chemin, prefixeModuleFilm) {
				continue
			}
			vus[strings.TrimPrefix(chemin, prefixeModuleFilm)] = true
		}
	}
	return clesTrieesFilm(vus), fichiers
}

// indexDesAretesTolerees indexe l allowlist des aretes par "de -> vers".
func indexDesAretesTolerees() map[string]areteToleree {
	out := make(map[string]areteToleree, len(aretesTolerees))
	for _, a := range aretesTolerees {
		out[cleArete(a.de, a.vers)] = a
	}
	return out
}

// clesTrieesFilm rend les cles d une map a cles chaines, triees.
func clesTrieesFilm[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
