// Package himap — moduleindex.go : resoudre un GlobalID de tag a travers PLUSIEURS modules.
//
// POURQUOI, ET C'EST MESURE (2026-08-08, ridgeline, 10 357 instances). Les references de
// geometrie d'une carte ne pointent PAS toutes dans le module de cette carte : 324 des 548
// references distinctes y sont, mais 70 vivent dans `globals/common`, 102 dans
// `globals/multiplayer` et 29 dans `globals/multiplayer_r3`. Le module de la carte ne
// couvre que **26 % des instances**. Une chaine « une carte = un module » rendrait donc une
// carte aux trois quarts vide en se croyant complete.
//
// Le handoff ne signalait ce piege que pour la COLLISION ; il vaut aussi pour le rendu.
package himap

import (
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/himodule"
)

// ModuleIndex resout un GlobalID de tag vers l'entree qui le porte, sur un ensemble de
// modules ouverts.
type ModuleIndex struct {
	modules []*himodule.Module
	chemins []string
	parID   map[uint32]refEntree
}

type refEntree struct {
	module  int
	fichier himodule.File
}

// NewModuleIndex ouvre les modules donnes et indexe leurs entrees par GlobalID.
//
// Un module illisible est une ERREUR, jamais un silence : un index partiel rendrait une
// carte trouee sans que rien ne le dise.
func NewModuleIndex(chemins ...string) (*ModuleIndex, error) {
	idx := &ModuleIndex{parID: map[uint32]refEntree{}}
	for _, p := range chemins {
		m, err := himodule.Open(p)
		if err != nil {
			return nil, fmt.Errorf("himap: ouvrir %s: %w", p, err)
		}
		i := len(idx.modules)
		idx.modules = append(idx.modules, m)
		idx.chemins = append(idx.chemins, p)
		for _, f := range m.Files("") {
			// Premier arrive, premier servi : le module de la carte est passe en tete, il
			// doit primer sur un homonyme global.
			if _, deja := idx.parID[f.GlobalID]; !deja {
				idx.parID[f.GlobalID] = refEntree{module: i, fichier: f}
			}
		}
	}
	return idx, nil
}

// GeometrySearchPath rend l'ordre de recherche d'une carte : son module d'abord, puis les
// modules globaux. C'est cet ordre qui donne la priorite au contenu de la carte.
func GeometrySearchPath(deployRoot, moduleCarte string) ([]string, error) {
	chemins := []string{moduleCarte}
	globs, err := filepath.Glob(filepath.Join(deployRoot, "pc", "globals", "*.module"))
	if err != nil {
		return nil, err
	}
	for _, g := range globs {
		if _, err := os.Stat(g); err == nil {
			chemins = append(chemins, g)
		}
	}
	return chemins, nil
}

// Lookup dit si un GlobalID est connu, et de quel module il vient.
func (idx *ModuleIndex) Lookup(id uint32) (groupe, module string, ok bool) {
	e, ok := idx.parID[id]
	if !ok {
		return "", "", false
	}
	return e.fichier.Group, idx.chemins[e.module], true
}

// Extract rend les octets decompresses du tag d'un GlobalID.
func (idx *ModuleIndex) Extract(id uint32) ([]byte, error) {
	e, ok := idx.parID[id]
	if !ok {
		return nil, fmt.Errorf("himap: GlobalID %#x absent des %d modules indexes", id, len(idx.modules))
	}
	return idx.modules[e.module].Extract(e.fichier)
}

// Taille rend le nombre d'entrees indexees (diagnostic).
func (idx *ModuleIndex) Taille() int { return len(idx.parID) }
