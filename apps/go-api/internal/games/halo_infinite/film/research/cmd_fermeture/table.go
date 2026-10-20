//go:build research

package main

// table.go — CE QUE L OUTIL LIT DE `ecs_table.tsv` : l usage produit (qui fait d un record un
// record UTILE) et, pour le classement des bloquants, le statut de chaque composant.
//
// `grammar` ne lit aucun fichier (lot J4.0) : c est ICI que la table se lit, et la carte recoit
// l ensemble des composants utiles en entree. La regle « utile = colonne `product_use` renseignee
// et differente de `aucun` » a une seconde copie, dans le ratchet de la carte
// (`frame_closure_ratchet_test.go`) : deux copies, pas trois (CLAUDE.md regle 6).

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// Les colonnes lues, par position (en-tete verifie a la lecture).
const (
	colTI         = 0
	colComposant  = 3
	colStatut     = 5
	colProductUse = 11
	colonnesMin   = colProductUse + 1
)

// ligneDeTable est ce que la table dit d un composant.
type ligneDeTable struct {
	statut, usage string
}

// tableECS est la table relue : les composants utiles, et chaque ligne par cle de composant.
type tableECS struct {
	chemin string
	utiles grammar.UsagesProduit
	parCle map[grammar.ComposantUtile]ligneDeTable
}

// lireTable relit `ecs_table.tsv` et en verifie l en-tete.
func lireTable(chemin string) (tableECS, error) {
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin donne par l operateur, lecture seule
	if err != nil {
		return tableECS{}, fmt.Errorf("table ECS illisible : %w", err)
	}
	lignes := strings.Split(strings.TrimRight(string(brut), "\n"), "\n")
	if len(lignes) < 2 {
		return tableECS{}, fmt.Errorf("table ECS vide : %s", chemin)
	}
	tete := strings.Split(lignes[0], "\t")
	if len(tete) < colonnesMin || tete[colTI] != "ti" || tete[colComposant] != "component" ||
		tete[colStatut] != "status" || tete[colProductUse] != "product_use" {
		return tableECS{}, fmt.Errorf("en-tete inattendu dans %s : %q", chemin, lignes[0])
	}
	t := tableECS{chemin: chemin, utiles: grammar.UsagesProduit{},
		parCle: map[grammar.ComposantUtile]ligneDeTable{}}
	for n, l := range lignes[1:] {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(c) < colonnesMin {
			return tableECS{}, fmt.Errorf("%s ligne %d : %d colonnes", chemin, n+2, len(c))
		}
		ti, err := strconv.Atoi(c[colTI])
		if err != nil {
			return tableECS{}, fmt.Errorf("%s ligne %d : ti %q", chemin, n+2, c[colTI])
		}
		if ti < 0 {
			continue // alias d orthographe : aucun archetype
		}
		cle := grammar.CleComposant(ti, c[colComposant])
		t.parCle[cle] = ligneDeTable{statut: c[colStatut], usage: c[colProductUse]}
		if usageDeclare(c[colProductUse]) {
			t.utiles[cle] = true
		}
	}
	if len(t.utiles) == 0 {
		return tableECS{}, fmt.Errorf("aucun composant a usage produit dans %s", chemin)
	}
	return t, nil
}

// usageDeclare : la colonne `product_use` dit que le produit lit le composant.
func usageDeclare(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && !strings.HasPrefix(v, "aucun")
}

// decrire rend le statut et l usage produit d un composant, « hors table » s il n y est pas (un
// build dont l archetype ne porte pas le meme rang que le registre de reference de la table).
func (t tableECS) decrire(ti int, composant string) (statut, usage string) {
	l, ok := t.parCle[grammar.CleComposant(ti, composant)]
	if !ok {
		return "hors table", "hors table"
	}
	return l.statut, l.usage
}
