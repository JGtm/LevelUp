package archlint

// no_raw_film_bytes_outside_source_helpers_test.go — LE BALAYAGE du ratchet « une seule porte
// aux octets ». Les tables, les regles et les messages vivent dans
// `no_raw_film_bytes_outside_source_test.go` ; ce fichier ne porte que la MECANIQUE :
// parcours des racines, carte des types declares par paquet, reconnaissance des cinq motifs.
//
// Tout passe par `go/parser` : un ident est un ident, jamais du texte. C est la raison d etre
// de ce fichier — les chemins, les noms de fonctions interdites et les extraits de grammaire
// abondent dans les COMMENTAIRES et les chroniques du decodeur, et un grep les compterait tous.
// `facts/fallback/registre_killsource.go` en est la preuve vivante : il cite
// `"func evBody(r *curseurEv"` dans une CHAINE, et ce balayage ne le compte pas.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// siteBrut : une lecture d octets bruts mesuree, la ou elle est.
type siteBrut struct {
	fichier string // chemin relatif a apps/go-api, en slash
	motif   string // un des motifs* declares dans le fichier principal
	detail  string // ce qui a ete vu (identifiant, expression)
	ligne   int
}

// cleLectureBrute : la cle (fichier, motif) que l allowlist indexe.
func cleLectureBrute(fichier, motif string) string { return fichier + " | " + motif }

// balayerLecturesBrutes parcourt les racines surveillees et rend les sites mesures, plus le
// nombre de fichiers `.go` de production reellement parcourus (pour le plancher).
func balayerLecturesBrutes(t *testing.T) ([]siteBrut, int) {
	t.Helper()
	racineAPI := apiRootDepuisIci(t)
	paquets := paquetsSurveillesOctets(t, racineAPI)
	var sites []siteBrut
	var fichiers int
	for _, dir := range clesTrieesOctets(paquets) {
		fset := token.NewFileSet()
		asts := map[string]*ast.File{}
		for _, chemin := range paquets[dir] {
			f, err := parser.ParseFile(fset, chemin, nil, 0)
			if err != nil {
				t.Fatalf("parse de %s : %v", chemin, err)
			}
			rel, _ := filepath.Rel(racineAPI, chemin)
			asts[filepath.ToSlash(rel)] = f
			fichiers++
		}
		types := typesDeclaresDuPaquet(asts)
		for _, rel := range clesTrieesOctets(asts) {
			sites = append(sites, motifsDuFichier(rel, asts[rel], types, fset)...)
		}
	}
	trierSitesBruts(sites)
	return sites, fichiers
}

// trierSitesBruts ordonne les sites par fichier puis par ligne (messages stables).
func trierSitesBruts(sites []siteBrut) {
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].fichier != sites[j].fichier {
			return sites[i].fichier < sites[j].fichier
		}
		return sites[i].ligne < sites[j].ligne
	})
}

// paquetsSurveillesOctets rend, par repertoire de paquet, les fichiers `.go` de PRODUCTION des
// racines surveillees, exclusions deduites.
func paquetsSurveillesOctets(t *testing.T, racineAPI string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, racine := range racinesOctetsBruts {
		abs := filepath.Join(racineAPI, filepath.FromSlash(racine))
		err := filepath.WalkDir(abs, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, rerr := filepath.Rel(racineAPI, chemin)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if exclusionDOctets(rel) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			out[filepath.Dir(chemin)] = append(out[filepath.Dir(chemin)], chemin)
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", racine, err)
		}
	}
	return out
}

// exclusionDOctets dit si un repertoire est declare hors de la regle.
func exclusionDOctets(rel string) bool {
	for _, ex := range exclusionsOctetsBruts {
		if rel == ex.chemin {
			return true
		}
	}
	return false
}

// typesDeclaresDuPaquet rend, par nom d identifiant en minuscules, l ENSEMBLE des types
// declares pour ce nom dans le paquet (champs de structure, parametres, resultats, variables
// typees). Un meme nom peut porter deux types dans un paquet : la carte les garde tous, et
// `estTrancheDOctetsPartout` ne conclut que si TOUS sont des tranches d octets.
func typesDeclaresDuPaquet(asts map[string]*ast.File) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	ajouter := func(noms []*ast.Ident, typ ast.Expr) {
		rendu := renduDuType(typ)
		for _, n := range noms {
			if n == nil || n.Name == "" || n.Name == "_" {
				continue
			}
			cle := strings.ToLower(n.Name)
			if out[cle] == nil {
				out[cle] = map[string]bool{}
			}
			out[cle][rendu] = true
		}
	}
	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Field:
				ajouter(v.Names, v.Type)
			case *ast.ValueSpec:
				if v.Type != nil {
					ajouter(v.Names, v.Type)
				}
			}
			return true
		})
	}
	return out
}

// renduDuType rend une forme textuelle STABLE des types qui nous interessent (`[]byte`,
// `[][]byte`) ; tout le reste rend une chaine quelconque non vide, qui suffit a dire « ce n est
// pas une tranche d octets ».
func renduDuType(typ ast.Expr) string {
	switch v := typ.(type) {
	case *ast.ArrayType:
		if v.Len != nil {
			return "tableau"
		}
		return "[]" + renduDuType(v.Elt)
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + renduDuType(v.X)
	case *ast.SelectorExpr:
		return renduDuType(v.X) + "." + v.Sel.Name
	case *ast.Ellipsis:
		return "[]" + renduDuType(v.Elt)
	default:
		return "autre"
	}
}

// estTrancheDOctetsPartout : le nom ne designe QUE des `[]byte` / `[][]byte` dans ce paquet.
// Inconnu ou melange -> faux : un ratchet qui invente des violations se fait desarmer.
func estTrancheDOctetsPartout(types map[string]map[string]bool, nom string) bool {
	vus := types[strings.ToLower(nom)]
	if len(vus) == 0 {
		return false
	}
	for typ := range vus {
		if typ != "[]byte" && typ != "[][]byte" {
			return false
		}
	}
	return true
}

// motifsDuFichier rend les sites d un fichier : les imports de decompression, puis les quatre
// motifs syntaxiques.
func motifsDuFichier(rel string, f *ast.File, types map[string]map[string]bool,
	fset *token.FileSet) []siteBrut {
	var sites []siteBrut
	for _, imp := range f.Imports {
		chemin := strings.Trim(imp.Path.Value, "\"")
		if paquetsDeDecompression[chemin] {
			sites = append(sites, siteBrut{rel, motifInflate, chemin,
				fset.Position(imp.Pos()).Line})
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		motif, detail, ok := motifDuNoeud(n, types)
		if ok {
			sites = append(sites, siteBrut{rel, motif, detail, fset.Position(n.Pos()).Line})
		}
		return true
	})
	return sites
}

// motifDuNoeud reconnait, sur un noeud, l un des quatre motifs syntaxiques.
//
// Le motif « lecteur de bits » vise la CONSTRUCTION et la DECLARATION, jamais la simple
// mention d un type : `func decodeX(br *BitReader)` fait circuler un lecteur deja construit
// sur des octets lus ailleurs, il n ouvre aucune porte. Ce qui ouvre la porte, c est
// `NewBitReader(pay)`, `&evReader{pl: pl}`, `bitAt(d, p)` — et la declaration de ces
// fonctions et de ces types.
func motifDuNoeud(n ast.Node, types map[string]map[string]bool) (string, string, bool) {
	switch v := n.(type) {
	case *ast.CallExpr:
		if nom := nomDeLaFonctionAppelee(v.Fun); constructeursDeLecteurDeBits[nom] {
			return motifLecteur, nom + "(...)", true
		}
	case *ast.FuncDecl:
		if v.Name != nil && constructeursDeLecteurDeBits[v.Name.Name] {
			return motifLecteur, "func " + v.Name.Name, true
		}
	case *ast.TypeSpec:
		if v.Name != nil && typesDeLecteurDeBits[v.Name.Name] {
			return motifLecteur, "type " + v.Name.Name, true
		}
	case *ast.CompositeLit:
		if nom := nomDeLaFonctionAppelee(v.Type); typesDeLecteurDeBits[nom] {
			return motifLecteur, nom + "{...}", true
		}
	case *ast.StructType:
		if estTypeDeLecteurDeBits(v) {
			return motifTypeLecteur, "struct{[]byte + position en bits}", true
		}
	case *ast.SelectorExpr:
		if x, ok := v.X.(*ast.Ident); ok && x.Name == "binary" && ordresDOctets[v.Sel.Name] {
			return motifBinaire, "binary." + v.Sel.Name, true
		}
	case *ast.IndexExpr:
		if nom, ok := nomDeChunkIndexe(v.X, types); ok {
			return motifChunk, nom + "[...]", true
		}
	case *ast.SliceExpr:
		if nom, ok := nomDeChunkIndexe(v.X, types); ok {
			return motifChunk, nom + "[...]", true
		}
	}
	return "", "", false
}

// nomDeLaFonctionAppelee rend le nom terminal d une expression d appel ou de type
// (`f`, `pkg.F`, `*pkg.T`), chaine vide sinon.
func nomDeLaFonctionAppelee(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.StarExpr:
		return nomDeLaFonctionAppelee(v.X)
	default:
		return ""
	}
}

// nomDeChunkIndexe : l expression indexee est-elle un chunk de film, par son nom ET par son
// type declare dans le paquet ?
func nomDeChunkIndexe(x ast.Expr, types map[string]map[string]bool) (string, bool) {
	nom := nomDeLaFonctionAppelee(x)
	if nom == "" || !nomsDeChunk[strings.ToLower(nom)] || !estTrancheDOctetsPartout(types, nom) {
		return "", false
	}
	return nom, true
}

// estTypeDeLecteurDeBits : une structure qui porte a la fois des octets et une position
// exprimee EN BITS est un lecteur de bits, quel que soit le nom qu on lui donne.
func estTypeDeLecteurDeBits(s *ast.StructType) bool {
	var octets, position bool
	for _, champ := range s.Fields.List {
		typ := renduDuType(champ.Type)
		for _, n := range champ.Names {
			if typ == "[]byte" {
				octets = true
			}
			if entiersDePosition[typ] && nomsDePositionEnBits[strings.ToLower(n.Name)] {
				position = true
			}
		}
	}
	return octets && position
}

// repertoireExisteSousAPI dit si un chemin relatif a `apps/go-api` designe un repertoire.
func repertoireExisteSousAPI(t *testing.T, rel string) bool {
	t.Helper()
	info, err := os.Stat(filepath.Join(apiRootDepuisIci(t), filepath.FromSlash(rel)))
	return err == nil && info.IsDir()
}

// clesTrieesOctets rend les cles d une carte, triees.
func clesTrieesOctets[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
