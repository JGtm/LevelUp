package replay

// map_background_index_catalogue_test.go — GARDE-RAIL DU CATALOGUE PUBLIÉ.
//
// CE QU'IL EMPÊCHE. L'index des fonds résout une carte par son NOM quand son map_id ne
// correspond plus (dérive d'identifiant d'asset). Cette voie n'est sûre que tant qu'un nom
// désigne UNE carte. Le jour où deux fonds publiés revendiquent la même identité, l'index les
// écarte tous les deux — les deux cartes perdent leur fond, en silence. Ce test rend
// l'événement bruyant AU MOMENT OÙ LE FOND EST COMMITÉ, pas six mois plus tard à l'écran.
//
// CE QU'IL NE FAIT PAS. Il ne juge pas la qualité d'un fond ni sa couverture : il ne regarde
// que les identités. Le catalogue de fonds est versionné (data/titles/{slug}/reference/
// map_backgrounds/), donc ce test tourne partout où le dépôt est là — y compris en CI.

import (
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// titreDuCatalogueDeFonds : le seul titre qui publie des fonds de carte à ce jour. Une
// constante nommée plutôt qu'un littéral répandu — et surtout PAS un `slug ==` de logique
// métier : c'est un chemin de donnée de test.
const titreDuCatalogueDeFonds = "halo_infinite"

// indexDuCatalogueLivre charge l'index des fonds réellement publiés dans le dépôt.
func indexDuCatalogueLivre(t *testing.T) *MapBackgroundIndex {
	t.Helper()
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine du dépôt introuvable — garde-rail de catalogue non applicable : %v", err)
	}
	dir := title.NewPathResolver(root).MapBackgroundDir(titreDuCatalogueDeFonds)
	idx, err := BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Skipf("catalogue de fonds absent (%s) : %v", dir, err)
	}
	return idx
}

// TestCatalogueDeFondsSansIdentiteAmbigue : aucun nom de carte ne doit désigner deux fonds.
//
// Mesuré le 2026-08-27 sur les 84 fonds publiés : 184 identités distinctes, ZÉRO ambiguïté.
// Si ce test rougit, NE PAS relâcher la règle : deux cartes distinctes portent le même nom
// public, et c'est la publication (clé map_id explicite pour l'une des deux) qu'il faut
// corriger — pas l'index.
func TestCatalogueDeFondsSansIdentiteAmbigue(t *testing.T) {
	idx := indexDuCatalogueLivre(t)
	if idx.Cles() == 0 {
		t.Fatal("aucun fond lu dans le catalogue publié — le garde-rail ne garde rien")
	}
	amb := idx.Ambigues()
	if len(amb) == 0 {
		return
	}
	noms := make([]string, 0, len(amb))
	for identite := range amb {
		noms = append(noms, identite)
	}
	sort.Strings(noms)
	var sb strings.Builder
	for _, identite := range noms {
		sb.WriteString("\n  " + identite + " -> " + strings.Join(amb[identite], ", "))
	}
	t.Errorf("identités de carte AMBIGUËS dans le catalogue publié (%d) — ces cartes n'auront "+
		"AUCUN fond au rejeu, l'index refusant de choisir :%s", len(amb), sb.String())
}

// TestCatalogueDeFondsIndexeChaqueCle : tout fond publié doit être atteignable par sa propre
// clé. Un fond que l'index ne sait pas nommer est un fond que personne n'affichera.
func TestCatalogueDeFondsIndexeChaqueCle(t *testing.T) {
	idx := indexDuCatalogueLivre(t)
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine du dépôt introuvable : %v", err)
	}
	dir := title.NewPathResolver(root).MapBackgroundDir(titreDuCatalogueDeFonds)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("catalogue de fonds illisible (%s) : %v", dir, err)
	}
	var muets []string
	for _, e := range entrees {
		cle, ok := cleDeSidecar(e)
		if !ok {
			continue
		}
		if got, trouve := idx.Lookup(cle); !trouve || got != cle {
			muets = append(muets, cle)
		}
	}
	if len(muets) > 0 {
		t.Errorf("fonds publiés que l'index ne résout pas par leur propre clé (%d) :\n  %s",
			len(muets), strings.Join(muets, "\n  "))
	}
}
