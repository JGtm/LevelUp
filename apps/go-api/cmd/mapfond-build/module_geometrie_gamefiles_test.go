//go:build gamefiles

package main

// cmd/mapfond-build — module_geometrie_gamefiles_test.go : LE SEUL TEST DES REGLAGES QUI OUVRE
// LE JEU, ISOLE SOUS SON TAG.
//
// POURQUOI CE FICHIER EXISTE (registre d'audit du 2026-09-05, constat I1). Ce test vivait dans
// `reglages_test.go`, hors du tag `gamefiles` : sur un poste ou Halo Infinite est installe, il
// balayait l'installation a chaque `go test ./cmd/...`, et en CI il prenait son `t.Skip` — donc
// personne ne voyait le cout. Les quatre autres tests du fichier ne lisent QUE l'asset publie
// (`data/titles/halo_infinite/reference/map_fond_reglages.json`) : ils restent, eux, dans le
// build par defaut et continuent de tourner en CI. La regle est celle de CLAUDE.md, section
// « Commandes utiles » : un test qui ouvre le jeu se nomme `*_gamefiles_test.go` ET porte le
// tag. Garde-rail : `internal/archlint`, TestCorpusGamefilesEstTague.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himap"
)

// TestModuleGeometrieExisteDansLInstallation : un `moduleGeometrie` mal orthographie ne se
// verrait qu a la cuisson, sous la forme d un « module introuvable » au milieu de 160 cartes.
// Le chemin est une DONNEE, il se verifie comme une donnee.
//
// Le test verifie AUSSI que la cle de l entree est bien un dossier installe : la cle de
// publication d une carte native est le nom du dossier du module (`filepath.Base` de son
// repertoire), et une cle qui ne correspond a rien serait un reglage silencieusement ignore.
func TestModuleGeometrieExisteDansLInstallation(t *testing.T) {
	racine, err := himap.DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	reg, err := chargeReglages(filepath.FromSlash(cheminReglagesPublies))
	if err != nil {
		t.Fatal(err)
	}
	vus := 0
	for cle, c := range reg.Cartes {
		if c.ModuleGeometrie == "" {
			continue
		}
		vus++
		chemin := filepath.Join(racine, filepath.FromSlash(c.ModuleGeometrie))
		if _, err := os.Stat(chemin); err != nil {
			t.Errorf("reglage %q : moduleGeometrie %q introuvable (%v)", cle, c.ModuleGeometrie, err)
		}
		propre, ok := himap.ChercheModuleInstalle(cle)
		if !ok {
			t.Errorf("reglage %q : aucun module installe ne porte cette cle — le reglage serait ignore", cle)
			continue
		}
		if filepath.Base(filepath.Dir(propre)) != cle {
			t.Errorf("reglage %q : la cle de publication serait %q", cle, filepath.Base(filepath.Dir(propre)))
		}
	}
	t.Logf("%d reglage(s) declarent un moduleGeometrie", vus)
}
