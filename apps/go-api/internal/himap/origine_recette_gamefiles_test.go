package himap

// origine_recette_gamefiles_test.go — LA RECETTE : trouver les points d'apparition d'objet
// ramassable sur une carte JAMAIS VUE, sans aucune liste curée à la main.
//
// ## CE QUI A ÉTÉ ÉTABLI, ET DANS QUEL ORDRE
//
// 1. Le `type_id` d'un objet de `.mvar` est le `GlobalID` d'un tag du catalogue FORGE
//    (`any/globals/forge/forge_objects-rtx-new.module`). Mesuré : sur le chemin de géométrie
//    (module de carte + globaux), AUCUN des huit types sondés ne résout — pas même les trois
//    socles prouvés. Sur le module Forge, les huit résolvent.
// 2. LE GROUPE NE SÉPARE PAS : les huit rendent `food`, socles PROUVÉS comme pavés de décor.
//    Une recette « groupe == food » ramasserait toute la carte.
// 3. CE QUI SÉPARE EST UN CRAN PLUS BAS — ce que le tag `food` RÉFÉRENCE :
//
//	  0x5F379533 power    (socle prouvé)  -> foki:4
//	  0x6253CFC0 rack     (socle prouvé)  -> foki:4
//	  0x5E86D110 powerup  (socle prouvé)  -> foki:4
//	  0xADEEE6D8 candidat                 -> foki:4
//	  0xE42158DF candidat                 -> foki:4
//	  0xA495FE83 décor (95-100 par carte) -> AUCUN foki
//	  0x8ACF288B décor (63-67 par carte)  -> AUCUN foki
//	  0xCBB239F7 décor (18-22 par carte)  -> AUCUN foki
//
// ## LA RECETTE
//
//	Un objet d'une variante de carte est un POINT D'APPARITION D'OBJET RAMASSABLE si et
//	seulement si son `type_id` résout, dans le module Forge, vers un tag `food` qui référence
//	au moins un tag du groupe `foki`.
//
// Elle est AUTONOME par construction : elle ne connaît aucune carte, aucune position, aucune
// liste d'identifiants. Elle lit le fichier de carte et interroge le catalogue du jeu.
//
// ## CE QU'ELLE NE PRÉTEND PAS
//
// Elle décrit la CARTE, pas le match : un point posé n'est pas un point allumé (le mode
// décide — Cliffhanger porte ses socles en CTF et zéro en Super Fiesta). Et elle ne dit pas
// encore CE QUI apparaît : distinguer arme, équipement et grenade demande de lire le `foki`
// lui-même, ce que ce lot ne fait pas.
//
// LECTURE SEULE sur les fichiers du jeu installé.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

func origIndexForge(t *testing.T, racine string) *ModuleIndex {
	t.Helper()
	chemins := []string{filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module")}
	if p := filepath.Join(racine, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existeFichier(p) {
		chemins = append(chemins, p)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	return idx
}

// EstPointDApparition est LA RECETTE, en une fonction.
//
// `idxForge` porte les tags `food` (catalogue Forge) ; `idxRef` résout les références inline
// et doit couvrir les groupes d'objet ramassable. Rend aussi le nombre de `foki` référencés —
// publié parce qu'un compte constant (4 sur tous les points mesurés) est un témoin de forme :
// s'il dérivait, la recette mériterait d'être re-mesurée.

// TestOrigineRecetteSepareSurLesTypesConnus — VALIDATION (a) : la recette retrouve-t-elle les
// cinq points connus sans ramasser le décor ?
//
// SEUIL ÉCRIT AVANT : 5 points sur 5 reconnus, 0 décor sur 3 retenu. Toute autre issue invalide
// la recette.
func TestOrigineRecetteSepareSurLesTypesConnus(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	idxForge := origIndexForge(t, racine)
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	chemins := append([]string{}, geo...)
	idxRef, err := NewModuleIndex(append(chemins, filepath.Join(racine, "any", "globals", "forge",
		"forge_objects-rtx-new.module"))...)
	if err != nil {
		t.Skipf("index de référence indisponible : %v", err)
	}
	points, decor := 0, 0
	for _, s := range origTypesASonder {
		ok, n := EstPointDApparition(idxForge, idxRef, s.id)
		verdict := "décor"
		if ok {
			verdict = "POINT D'APPARITION"
		}
		t.Logf("0x%08X  %-20s foki=%d  %s", s.id, verdict, n, s.libelle)
		attenduPoint := s.id == 0x5F379533 || s.id == 0x6253CFC0 || s.id == 0x5E86D110 ||
			s.id == 0xADEEE6D8 || s.id == 0xE42158DF
		switch {
		case attenduPoint && ok:
			points++
		case !attenduPoint && !ok:
			decor++
		default:
			t.Errorf("0x%08X : la recette se trompe (attendu point=%v, obtenu %v)", s.id, attenduPoint, ok)
		}
	}
	t.Logf("VERDICT : %d/5 points reconnus · %d/3 décors écartés", points, decor)
}

// TestOrigineRecetteBalayeLeCatalogueForge — combien de types du catalogue Forge sont des
// points d'apparition ? Un ordre de grandeur crédible est attendu (des dizaines, pas des
// milliers) : le jeu a un nombre fini de socles.
//
// C'est la mesure qui dit si la recette est SÉLECTIVE. Une recette qui retiendrait 10 % du
// catalogue ne trierait rien.
func TestOrigineRecetteBalayeLeCatalogueForge(t *testing.T) {
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
		t.Skipf("index de référence indisponible : %v", err)
	}
	// ÉNUMÉRATION DIRECTE du module, sans ajouter de méthode à `ModuleIndex` : c'est un lot de
	// recherche, il ne touche pas la production.
	mod, err := himodule.Open(filepath.Join(racine, "any", "globals", "forge",
		"forge_objects-rtx-new.module"))
	if err != nil {
		t.Skipf("module Forge illisible : %v", err)
	}
	var ids []uint32
	vus := map[uint32]bool{}
	for _, f := range mod.Files("food") {
		if !vus[f.GlobalID] {
			vus[f.GlobalID] = true
			ids = append(ids, f.GlobalID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	points, parFoki := 0, map[int]int{}
	var exemples []uint32
	for _, id := range ids {
		ok, n := EstPointDApparition(idxForge, idxRef, id)
		if !ok {
			continue
		}
		points++
		parFoki[n]++
		if len(exemples) < 12 {
			exemples = append(exemples, id)
		}
	}
	t.Logf("== SÉLECTIVITÉ DE LA RECETTE SUR LE CATALOGUE FORGE ==")
	t.Logf("tags `food` du catalogue : %d · retenus comme POINTS D'APPARITION : %d (%.2f %%)",
		len(ids), points, 100*float64(points)/float64(max(len(ids), 1)))
	keys := make([]int, 0, len(parFoki))
	for k := range parFoki {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		t.Logf("   %d référence(s) `foki` : %d type(s)", k, parFoki[k])
	}
	s := ""
	for i, id := range exemples {
		if i > 0 {
			s += " "
		}
		s += hex8(id)
	}
	t.Logf("   exemples : %s", s)

	// LA SORTIE DE LA RECETTE EST UN LIVRABLE : la liste des types retenus est écrite pour que
	// l'instrument d'origine (paquet `replay`) la consomme telle quelle. C'est ce qui ferme la
	// chaîne — recette -> liste -> classification — sans qu'aucune main ne recopie un
	// identifiant.
	if out := os.Getenv("ORIGINE_TYPES_OUT"); out != "" {
		var retenus []uint32
		for _, id := range ids {
			if ok, _ := EstPointDApparition(idxForge, idxRef, id); ok {
				retenus = append(retenus, id)
			}
		}
		blob, err := json.MarshalIndent(retenus, "", " ")
		if err != nil {
			t.Fatalf("sérialisation des types : %v", err)
		}
		if err := os.WriteFile(out, blob, 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", out, err)
		}
		t.Logf("   types retenus écrits : %s (%d)", out, len(retenus))
	}
}

func hex8(v uint32) string {
	const d = "0123456789ABCDEF"
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = d[v&0xF]
		v >>= 4
	}
	return "0x" + string(b)
}
