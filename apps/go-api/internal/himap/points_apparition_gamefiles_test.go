//go:build gamefiles

package himap

// points_apparition_gamefiles_test.go — LE GARDE-RAIL de la table figee.
//
// `mapvar.spawnPointTypes` est une table EN DUR : treize `type_id` recopies dans le code parce
// que la generation du catalogue doit tourner hors ligne, sur une machine sans le jeu installe.
// Une table en dur derive — c'est la lecon que le depot paie depuis le predicat bot passe de 8
// a 36 copies. Ce test est le lien qui l'empeche : il RE-DERIVE la table depuis les fichiers du
// jeu par la recette elle-meme, et echoue si les deux ne coincident plus.
//
// IL COUTE CHER, ET LE PRIX EST ECRIT ICI PLUTOT QUE DECOUVERT PAR UN PLANTAGE : le balayage
// extrait les 4 235 tags `food` du catalogue Forge et, pour chacun, les tags qu'il reference.
// MESURE du 2026-09-01 : le processus de test du paquet monte a 11 GiB de memoire resident et
// la suite depasse le delai `go test` par defaut (600 s). Lancer ce paquet exige donc
// `-timeout 45m` et une machine libre — surtout, NE PAS le lancer en parallele d'une cuisson
// d'artefact, dont le plafond memoire suppose la machine disponible.
//
// IL SAUTE SANS LE JEU INSTALLE, comme tous les `*_gamefiles_test.go` du paquet : il ne peut
// donc pas garder la CI. Il garde la machine qui REGENERE le catalogue, et c'est la seule qui
// puisse faire deriver la table.

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

// TestRecetteRedonneLaTableDesPoints re-derive la table et la compare a celle du code.
func TestRecetteRedonneLaTableDesPoints(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	idxForge := origIndexForge(t, racine)
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	idxRef, err := NewModuleIndex(append(append([]string{}, geo...),
		filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module"))...)
	if err != nil {
		t.Skipf("index de reference indisponible : %v", err)
	}
	mod, err := himodule.Open(filepath.Join(racine, "any", "globals", "forge",
		"forge_objects-rtx-new.module"))
	if err != nil {
		t.Skipf("catalogue Forge illisible : %v", err)
	}
	// Balayage de TOUS les tags `food` du catalogue, comme le fait la recette de recherche.
	var retenus []uint32
	for _, f := range mod.Files("food") {
		if ok, _ := EstPointDApparition(idxForge, idxRef, f.GlobalID); ok {
			retenus = append(retenus, f.GlobalID)
		}
	}
	sort.Slice(retenus, func(i, j int) bool { return retenus[i] < retenus[j] })

	// LA TABLE DU CODE = les points d'apparition + les trois socles d'arme. Les socles ne sont
	// PAS dans `spawnPointTypes` (ils sortent par `PadFamilyOf`), mais la recette les retient :
	// c'est cette somme qu'il faut comparer, sans quoi le garde-rail signalerait un faux ecart.
	attendus := map[uint32]bool{}
	for _, id := range mapvar.SpawnPointTypeIDs() {
		attendus[uint32(id)] = true
	}
	for _, id := range soclesDArmeConnus {
		attendus[id] = true
	}

	obtenus := map[uint32]bool{}
	for _, id := range retenus {
		obtenus[id] = true
	}
	var manquants, enTrop []uint32
	for id := range attendus {
		if !obtenus[id] {
			manquants = append(manquants, id)
		}
	}
	for id := range obtenus {
		if !attendus[id] {
			enTrop = append(enTrop, id)
		}
	}
	sort.Slice(manquants, func(i, j int) bool { return manquants[i] < manquants[j] })
	sort.Slice(enTrop, func(i, j int) bool { return enTrop[i] < enTrop[j] })

	t.Logf("recette : %d types retenus sur le catalogue Forge · table du code : %d points "+
		"d'apparition + %d socles d'arme = %d",
		len(retenus), mapvar.SpawnPointTypeCount(), len(soclesDArmeConnus), len(attendus))
	if len(manquants) > 0 {
		t.Errorf("la table du code declare %d type(s) que la recette NE RETIENT PLUS : %s — "+
			"soit la recette a change, soit la table a ete editee a la main",
			len(manquants), listeHex(manquants))
	}
	if len(enTrop) > 0 {
		t.Errorf("la recette retient %d type(s) ABSENT(S) de la table du code : %s — la carte "+
			"inconnue en porterait des points que le catalogue n'ecrirait pas",
			len(enTrop), listeHex(enTrop))
	}
}

// soclesDArmeConnus — les trois `type_id` de socle d'ARME, propriete de `mapvar.PadFamilyOf`.
// Recopies ici parce que `mapvar` ne les exporte pas et n'a aucune raison de le faire : ce test
// est le SEUL endroit du depot qui ait besoin de la somme des deux tables.
var soclesDArmeConnus = []uint32{0x5F379533, 0x6253CFC0, 0x5E86D110}

func listeHex(ids []uint32) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += " "
		}
		out += hex8(id)
	}
	return out
}
