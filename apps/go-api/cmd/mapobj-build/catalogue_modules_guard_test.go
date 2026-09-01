package main

// GARDE-RAIL DU CHAMP `module` DU CATALOGUE D'OBJECTIFS.
//
// POURQUOI IL EXISTE. Le 2026-08-25, le re-tirage réseau des 73 cartes (`d50f3b728`) a écrasé
// `module` par le nom de fichier servi par le réseau — `map.mvar` pour la plupart. Le catalogue
// est passé de **71 modules distincts à 12**, et 58 entrées se sont retrouvées à `map`.
// Conséquence mesurée le 2026-08-26 : `mapfond-build` ne savait plus cuire que 11 des 19 fonds
// natifs publiés. Cliffhanger, Aquarius, Prism, Streets, Recharge, Chasm, Launch Site et
// Behemoth étaient devenus incuisables — six d'entre elles étaient précisément les cartes que
// l'utilisateur venait de demander à retravailler.
//
// CE QUE LE GARDE-RAIL DU LOT FAUTIF MESURAIT : le nombre de collines (« 133 -> 273, 0 perdue »).
// Il était vert. Un garde-rail ne protège que le champ qu'il compte.
//
// CE TEST COMPTE CELUI-LÀ, sur l'asset PUBLIÉ — pas sur une fixture, parce que c'est l'asset
// publié qui est lu en production.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testutil"
)

// cheminCatalogueObjectifs : l asset publie, relatif a la RACINE du depot. La racine se demande
// a testutil.RepoRoot() et jamais par une echelle de « .. » — une echelle se casse en silence des
// que le test change de dossier, et le garde-rail archlint l interdit.
const cheminCatalogueObjectifs = "data/titles/halo_infinite/reference/map_objectives.json"

// modulesDistinctsMin / entreesParModuleMax : le cliquet.
//
// Au 2026-08-26, catalogue réparé : 73 entrées, 71 modules distincts, et un seul module porté
// par plus d'une entrée — `map`, sur Vagabond et une variante de Highpower, deux cartes dont le
// réseau sert réellement un fichier `map.mvar` (mesuré le 2026-08-08, cf. `saveVariantFile`).
//
// Ces bornes ne se relèvent PAS pour faire passer une régénération : si une régénération les
// fait rougir, c'est elle qui a perdu de l'information, pas le test qui est trop strict.
const (
	modulesDistinctsMin  = 70
	entreesParModuleMax  = 2
	moduleGeneriqueConnu = "map"
)

func TestCatalogueObjectifsModulesDistincts(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(cheminCatalogueObjectifs)))
	if err != nil {
		t.Skipf("catalogue d'objectifs absent (%v) — test de données, pas de code", err)
	}
	var cat struct {
		Maps map[string]struct {
			MvarFile string `json:"mvar_file"`
			Module   string `json:"module"`
		} `json:"maps"`
	}
	if err := json.Unmarshal(blob, &cat); err != nil {
		t.Fatalf("catalogue illisible : %v", err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide")
	}

	parModule := map[string][]string{}
	for id, e := range cat.Maps {
		if e.Module == "" {
			t.Errorf("carte %s (%s) : module VIDE — elle ne se resoudra vers aucun dossier installe",
				id, e.MvarFile)
			continue
		}
		parModule[e.Module] = append(parModule[e.Module], id)
	}

	if n := len(parModule); n < modulesDistinctsMin {
		t.Errorf("modules distincts = %d, attendu >= %d sur %d entrees.\n"+
			"C'est la signature de la regression du 2026-08-25 : un tirage reseau a ecrase\n"+
			"`module` par le nom de fichier servi (`map.mvar`). Ne pas baisser le seuil —\n"+
			"regenerer le catalogue en conservant les modules deja connus (gardeModuleConnu).",
			n, modulesDistinctsMin, len(cat.Maps))
	}
	for mod, ids := range parModule {
		if len(ids) <= entreesParModuleMax {
			continue
		}
		t.Errorf("module %q porte par %d entrees (max %d) : %v.\n"+
			"Un module partage par plus de deux cartes n'est pas un module, c'est un nom de\n"+
			"fichier generique — la resolution vers le dossier installe est perdue.",
			mod, len(ids), entreesParModuleMax, ids)
	}
	// Le cas connu est nommé pour qu'il ne se dilue pas dans le seuil : si `map` cessait un jour
	// d'être porté par deux cartes, c'est que la donnée a bougé et le commentaire ci-dessus doit
	// être relu, pas contourné.
	if n := len(parModule[moduleGeneriqueConnu]); n > entreesParModuleMax {
		t.Errorf("module generique %q porte par %d entrees", moduleGeneriqueConnu, n)
	}
}
