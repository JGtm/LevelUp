//go:build gamefiles

package himap

// origine_typeid_tags_gamefiles_test.go — RÉSOUDRE LA SÉMANTIQUE DES `type_id` DE VARIANTE.
//
// L'ENJEU, et il décide de la forme du livrable. Le chantier « origine socle vs sol » a trouvé
// deux `type_id` d'objet de carte qui tombent PILE (0,00-0,01 m) sur les positions où naissent
// les objets `ti=37` : `0xADEEE6D8` et `0xE42158DF`. Mais une LISTE de deux identifiants n'est
// pas une recette : sur une carte jamais vue, rien ne dit qu'ils suffisent, ni qu'ils sont les
// bons. Ce qu'il faut, c'est une FONCTION — « donne-moi les objets dont le type EST un point
// d'apparition d'objet ramassable » — et pour l'écrire il faut savoir ce que ces types SONT.
//
// LA CHAÎNE EXISTE DÉJÀ, et elle est documentée dans `cuisson_forge.go` : le `type_id` d'un
// objet de `.mvar` EST le `GlobalID` d'un tag du jeu. `ModuleIndex.Lookup` rend son GROUPE
// (fourCC : `eqip`, `weap`, `bloc`, `scen`…) et le module qui le porte. Le groupe est une
// sémantique : c'est le type de ressource au sens du moteur.
//
// CE QUE CETTE SONDE MESURE, sans rien supposer : le groupe de tag des deux `type_id` trouvés,
// de la liste blanche actuelle (`power`, `rack`, `powerup` — pour étalonner ce qu'un « socle »
// donne comme groupe), et des types de DÉCOR les plus fréquents (pour voir ce qu'un pavé
// donne). Si les groupes séparent, la recette s'écrit ; s'ils ne séparent pas, on le dit.
//
// LECTURE SEULE sur les fichiers du jeu installé. Sautée sans installation (DeployRoot).

import (
	"path/filepath"
	"sort"
	"testing"
)

// origTypeSonde est un type d'objet à résoudre, avec ce qu'on sait déjà de lui.
type origTypeSonde struct {
	id      uint32
	libelle string
}

// origTypesASonder : les deux candidats, les trois de la liste blanche, et le décor.
//
// LES TROIS DE LA LISTE BLANCHE SONT L'ÉTALON : ce sont des socles PROUVÉS (32/32 appariés,
// médiane 0,01 m). Le groupe qu'ils rendent est celui qu'un point d'apparition porte. Sans eux
// la mesure n'aurait pas de référence.
var origTypesASonder = []origTypeSonde{
	{0x5F379533, "liste blanche — power (socle d'arme lourde)"},
	{0x6253CFC0, "liste blanche — rack (râtelier d'arme)"},
	{0x5E86D110, "liste blanche — powerup (socle de bonus)"},
	{0xADEEE6D8, "CANDIDAT — 4 objets sur Catalyst, 5 sur Cliffhanger"},
	{0xE42158DF, "CANDIDAT — 4 objets sur Catalyst, 4 sur Cliffhanger"},
	{0xA495FE83, "DÉCOR écarté — 95 à 100 objets par carte"},
	{0x8ACF288B, "DÉCOR — 63 à 67 objets par carte"},
	{0xCBB239F7, "DÉCOR — 18 à 22 objets par carte"},
}

// TestOrigineTypeIDVersTag résout chaque type en groupe de tag et publie la table.
func TestOrigineTypeIDVersTag(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	// LE BON MODULE N'EST PAS CELUI DE LA CARTE, et le premier essai l'a montré : sur le
	// chemin de géométrie (module de carte + globaux, 56 766 entrées), AUCUN des huit types
	// ne résout — pas même les trois socles PROUVÉS. Le `type_id` d'un objet de variante est
	// le GlobalID d'un tag du catalogue FORGE, `any/globals/forge/forge_objects-rtx-new.module`
	// (cf. `indexForge`, cuisson_forge.go). C'est là qu'il faut chercher.
	chemins := []string{filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module")}
	if p := filepath.Join(racine, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existeFichier(p) {
		chemins = append(chemins, p)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	t.Logf("== RÉSOLUTION `type_id` -> TAG · %d entrées indexées ==", idx.Taille())
	t.Logf("%-12s %-8s %-28s %s", "type_id", "groupe", "module", "ce qu'on savait")
	groupes := map[string][]string{}
	for _, s := range origTypesASonder {
		groupe, module, ok := idx.Lookup(s.id)
		if !ok {
			t.Logf("0x%08X   %-8s %-28s %s", s.id, "—", "NON RÉSOLU", s.libelle)
			continue
		}
		if module == "" {
			module = "(module de la carte)"
		}
		t.Logf("0x%08X   %-8s %-28s %s", s.id, groupe, shortModule(module), s.libelle)
		groupes[groupe] = append(groupes[groupe], s.libelle)
	}
	t.Log("--- regroupement par GROUPE de tag ---")
	keys := make([]string, 0, len(groupes))
	for g := range groupes {
		keys = append(keys, g)
	}
	sort.Strings(keys)
	for _, g := range keys {
		t.Logf("  %-8s : %d type(s)", g, len(groupes[g]))
		for _, l := range groupes[g] {
			t.Logf("             %s", l)
		}
	}
	t.Log("LECTURE : si les socles PROUVÉS et les CANDIDATS partagent un groupe que le DÉCOR " +
		"n'a pas, la recette s'écrit « tout objet dont le type résout vers ce groupe ». Sinon, " +
		"le groupe ne suffit pas et il faut descendre d'un cran (contenu du tag).")
}

// shortModule raccourcit un chemin de module pour la lisibilité du tableau.
func shortModule(p string) string {
	if len(p) <= 26 {
		return p
	}
	return "…" + p[len(p)-25:]
}

// TestOrigineTagFoodReference — LE NIVEAU EN DESSOUS, parce que le groupe ne sépare pas.
//
// Les huit types résolvent TOUS en `food` : socles prouvés ET décor. Le groupe ne peut donc pas
// fonder la recette. Ce qui doit séparer, c'est ce que le tag food RÉFÉRENCE : un point
// d'apparition pointe l'objet RAMASSABLE qu'il fait naître (`eqip`, `weap`, …) ; un pavé de
// décor ne pointe que de la géométrie (`rtgo`, `bloc`, `scen`, `mach`).
//
// La sonde extrait chaque tag food et publie les GROUPES de ses références inline, résolues
// contre l'index COMPLET (Forge + globaux — c'est là que vivent `eqip` et `weap`).
func TestOrigineTagFoodReference(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	forge := []string{filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module")}
	if p := filepath.Join(racine, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existeFichier(p) {
		forge = append(forge, p)
	}
	idxForge, err := NewModuleIndex(forge...)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	// L'index de RÉSOLUTION des références : Forge + le chemin de géométrie complet.
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	idxTout, err := NewModuleIndex(append(append([]string{}, forge...), geo...)...)
	if err != nil {
		t.Skipf("index complet indisponible : %v", err)
	}
	t.Logf("== CE QUE LE TAG `food` RÉFÉRENCE · index Forge %d entrées, complet %d ==",
		idxForge.Taille(), idxTout.Taille())
	for _, s := range origTypesASonder {
		tag, err := idxForge.Extract(s.id)
		if err != nil {
			t.Logf("0x%08X : tag inextractible (%v) — %s", s.id, err, s.libelle)
			continue
		}
		compte := map[string]int{}
		RefsInline(tag, func(h uint32) bool {
			if g, _, ok := idxTout.Lookup(h); ok {
				compte[g]++
			}
			return false // on ne retient rien : on COMPTE les groupes rencontrés
		})
		keys := make([]string, 0, len(compte))
		for g := range compte {
			keys = append(keys, g)
		}
		sort.Slice(keys, func(i, j int) bool { return compte[keys[i]] > compte[keys[j]] })
		s2 := ""
		for i, g := range keys {
			if i >= 8 {
				break
			}
			if i > 0 {
				s2 += " "
			}
			s2 += g + ":" + itoaSimple(compte[g])
		}
		t.Logf("0x%08X (%d o) -> %s", s.id, len(tag), s2)
		t.Logf("            %s", s.libelle)
	}
	t.Log("LECTURE : si les socles prouvés et les candidats référencent un groupe d'objet " +
		"RAMASSABLE que le décor n'a pas, la recette tient et elle est autonome.")
}

func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
