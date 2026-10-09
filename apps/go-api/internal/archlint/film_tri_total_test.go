package archlint

// film_tri_total_test.go — LE CLIQUET DES TRIS NON TOTAUX DU DECODEUR (lot J10.1, 2026-09-27,
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, DT-9 amende par la decision utilisateur du
// 2026-09-27).
//
// # POURQUOI
//
// `sort.Slice` n'est pas stable, et `sort.Sort` non plus : a cle egale, le rang de deux elements
// est celui que le tri leur donne — stable PAR ACCIDENT sous treize elements (tri par insertion),
// tire par pdqsort au-dela. Quand l'entree est batie en iterant une MAP, ce rang change d'une
// execution a l'autre ; quand elle vient du film, il change des qu'un amont bouge. Dans un
// decodeur dont la sortie est publiee, persistee et comparee a l'octet, un ordre qui ne tient qu'au
// tri est un defaut, pas un detail.
//
// DT-9 fixe la forme : `slices.SortStableFunc` / `slices.SortFunc` a comparateur TOTAL — une chaine
// `cmp.Or` qui finit sur une cle unique, ou, quand aucun champ n'est unique, un tri STABLE sur une
// entree deja dans l'ordre du film (la cle unique est alors ce rang, et le commentaire du tri le
// dit).
//
// # LA DECISION DU 2026-09-27 : CORRECTION CIBLEE + CLIQUET
//
// La mesure d'entree a trouve 231 appels `sort.Slice*` / `sort.Sort` de production dans le
// perimetre (144 `Slice`, 87 `SliceStable`, 149 fichiers ; l'audit en annoncait 43). Le lot J10.1 a
// converti les 42 qui DECIDAIENT D'UNE SORTIE sans cle prouvee unique (releve au rapport du lot, un
// test d'ex aequo par tri corrige). Les 189 restants sont listes NOMMEMENT ci-dessous.
// Le lot J12.1 (2026-09-30) les a tous convertis : la table est vide.
//
// # LA REGLE, EN DEUX LIGNES
//
//	appel hors table     interdit : un NOUVEL appel est rouge (TestTriTotalAucunNouvelAppel)
//	entree de la table   son compte ne peut que BAISSER : une entree dont le compte reel est plus
//	                     bas, ou qui n'existe plus, est rouge (TestTriTotalTableNeFaitQueBaisser)
//
// La cle est « fichier:fonction englobante » (methode : `(*T).Nom`), jamais un numero de ligne :
// une ligne bouge a chaque lot, une fonction non. Un appel hors de toute fonction (initialisation
// de variable de paquet) a pour fonction `(niveau paquet)`.
//
// # PERIMETRE
//
// La PRODUCTION de `film/**` (hors `film/research/` et hors fichiers `//go:build research`),
// `replaybuild`, `filmproc` et `sync/killcollector`. Les tests en sont exclus : un tri de test ne
// publie rien. `sort.Stable` est compte avec `sort.Sort` (meme forme, zero appel le 2026-09-27).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// racinesTriTotal : le perimetre du cliquet, relatif a apps/go-api.
var racinesTriTotal = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/filmproc",
	"internal/sync/killcollector",
}

// exclusTriTotal : les sous-arbres exclus (instruments de recherche).
var exclusTriTotal = []string{"internal/games/halo_infinite/film/research"}

// plancherFichiersTriTotal : CONTRE UN BALAYAGE MUET. 588 fichiers de production mesures le
// 2026-09-27 ; nettement moins veut dire que l'arborescence a bouge et que le cliquet ne garde plus
// rien.
const plancherFichiersTriTotal = 500

// fonctionsDeTriNonTotal : les fonctions du paquet `sort` que le cliquet compte.
var fonctionsDeTriNonTotal = map[string]bool{"Slice": true, "SliceStable": true, "Sort": true, "Stable": true}

// trisNonTotauxToleresAu20260927 — LA TABLE DATEE (2026-09-27, lot J10.1). Elle ne peut que
// DIMINUER. Conversion du reste : J12 (modernisation neutre, DT-10).
//
// 173 entrees, 189 appels : les 87 `sort.SliceStable` (deterministes sur une entree deterministe,
// mais sans comparateur total ecrit) et les `sort.Slice` dont la cle est prouvee unique, qui
// trient des valeurs, qui alimentent un calcul insensible a l'ordre ou qui vivent hors production
// (verdicts au rapport du lot J10.1).
//
// FUSION DE J10 APRES J8 (2026-09-27) : 172 entrees, 188 appels. J8 avait deplace le tri de
// `objectives.roundStartsOf` dans `roundStartsOfCompte` ; son comparateur, deja total (le numero
// de manche est unique), s ecrit en `cmp.Or` et l entree est retiree.
//
// LOT J12.1 (2026-09-30) : TABLE VIDE. Les 188 appels sont convertis en `slices.SortStableFunc`
// (les 87 stables, comparateur fidele, sans departage) ou `slices.SortFunc` / `slices.Sort` (les
// 101 `sort.Slice` : valeurs, cle prouvee unique, ou departage ajoute jusqu a l unicite). Le
// cliquet devient une interdiction pure : tout appel `sort.Slice*` / `sort.Sort` / `sort.Stable`
// de production dans le perimetre est rouge.
var trisNonTotauxToleresAu20260927 = map[string]int{}

func TestTriTotalAucunNouvelAppel(t *testing.T) {
	vus := balayerTrisNonTotaux(t)
	for _, cle := range clesDeTriTriees(vus) {
		n, tolere := vus[cle], trisNonTotauxToleresAu20260927[cle]
		if n <= tolere {
			continue
		}
		t.Errorf("%s : %d appel(s) sort.Slice/SliceStable/Sort/Stable, %d tolere(s) au 2026-09-27.\n"+
			"Un NOUVEL appel est interdit (DT-9) : slices.SortStableFunc / slices.SortFunc a "+
			"comparateur TOTAL — une chaine cmp.Or qui finit sur une cle unique. Ajouter une entree "+
			"a trisNonTotauxToleresAu20260927 N'EST PAS une reponse : la table ne fait que baisser.",
			cle, n, tolere)
	}
}

func TestTriTotalTableNeFaitQueBaisser(t *testing.T) {
	vus := balayerTrisNonTotaux(t)
	for _, cle := range clesDeTriTriees(trisNonTotauxToleresAu20260927) {
		tolere, n := trisNonTotauxToleresAu20260927[cle], vus[cle]
		switch {
		case n == 0:
			t.Errorf("%s est dans la table mais ne porte plus aucun appel (converti, renomme ou "+
				"deplace) : RETIRER l'entree — une entree perimee finit par autoriser n'importe quoi.", cle)
		case n < tolere:
			t.Errorf("%s : %d appel(s) pour %d tolere(s) — BAISSER l'entree a %d (cliquet).", cle, n, tolere, n)
		}
	}
}

// balayerTrisNonTotaux rend, par « fichier:fonction », le nombre d'appels comptes.
func balayerTrisNonTotaux(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	fichiers := 0
	balayerLaProduction(t, racinesTriTotal, func(rel, chemin string) error {
		compte, err := compterTrisDuFichier(chemin, rel, out)
		fichiers += compte
		return err
	})
	if fichiers < plancherFichiersTriTotal {
		t.Fatalf("balayage muet : %d fichiers de production vus dans %v, plancher %d.",
			fichiers, racinesTriTotal, plancherFichiersTriTotal)
	}
	return out
}

// repertoireExcluDuTriTotal : repertoires caches, `_x`, `testdata` et sous-arbres de recherche.
func repertoireExcluDuTriTotal(nom, rel string) bool {
	if strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_") || nom == "testdata" {
		return true
	}
	for _, ex := range exclusTriTotal {
		if rel == ex {
			return true
		}
	}
	return false
}

// compterTrisDuFichier ajoute a `out` les appels du fichier ; rend 1 si le fichier est compte
// (production hors tag research), 0 sinon.
func compterTrisDuFichier(chemin, rel string, out map[string]int) (int, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du perimetre
	if err != nil {
		return 0, err
	}
	if estSousTagResearch(blob) {
		return 0, nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), chemin, blob, 0)
	if err != nil {
		return 0, err
	}
	nomSort := nomDImportDeSort(f)
	switch nomSort {
	case "", "_":
		return 1, nil
	case ".":
		out[rel+":(import point de sort)"]++
		return 1, nil
	}
	for _, decl := range f.Decls {
		englobante := "(niveau paquet)"
		if fd, ok := decl.(*ast.FuncDecl); ok {
			englobante = nomDeFonction(fd)
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			if estAppelDeTriNonTotal(n, nomSort) {
				out[rel+":"+englobante]++
			}
			return true
		})
	}
	return 1, nil
}

// nomDImportDeSort rend le nom sous lequel le fichier importe `sort`, ou "" s'il ne l'importe pas.
func nomDImportDeSort(f *ast.File) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "sort" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "sort"
	}
	return ""
}

// estAppelDeTriNonTotal dit si le noeud est un appel `sort.Slice`, `sort.SliceStable`,
// `sort.Sort` ou `sort.Stable` (sous le nom d'import du fichier).
func estAppelDeTriNonTotal(n ast.Node, nomSort string) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == nomSort && fonctionsDeTriNonTotal[sel.Sel.Name]
}

func clesDeTriTriees(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
