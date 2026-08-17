package himap

// SONDE (2026-08-17, plan .ai/V7.5/replay2d/PLAN_IDENTITE_TI37.md phase 0.3) — NOMMER les
// valeurs du champ `mpp-word32` lu dans le record de CREATION des objets d'equipement (ti=37).
//
// CE QUE LA MESURE AMONT A DONNE (filmdec/equipment_creation_test.go, 2026-08-17) : le champ
// de 32 bits INCONDITIONNEL du bloc `object-multiplayer-properties` prend HUIT valeurs
// distinctes sur 274 records de creation de `000d5950` et SIX sur 229 de `00162144`, dont
// TROIS communes aux deux films. Une enumeration stable, pas du bruit.
//
// LA QUESTION POSEE ICI : ces valeurs sont-elles des GlobalID de tag ? Si oui, leur groupe
// (`eqip`, `bloc`, `scen`, `mach`, `weap`...) dit CE QU'EST chaque objet, et l'identite de
// ti=37 est resolue par le record de creation — ce que le plan cherchait.
//
// LECTURE SEULE, saute si le jeu n'est pas installe. UN SEUL module en memoire a la fois :
// `himodule.Open` lit le fichier ENTIER, et `pc/globals/globals-rtx-new.module` fait 7,4 Go.
// On indexe donc la variante `any/` (634 Mo au plus), module par module.

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// ti37Observes : les valeurs de `mpp-word32` relevees dans les POSES confirmees des records de
// creation, avec le nombre de films du corpus et l'effectif total. Recopiees de la mesure amont
// — c'est la PIECE.
//
// MISE A JOUR DU 2026-08-18 (plan PLAN_POSES_EQUIPEMENT_PUBLICATION, phase 1) : le corpus passe
// de 2 films a 11 (le douzieme, `0014603f`, ne porte aucune pose mesurable et se refuse), et la
// cohorte n'est plus le balayage BRUT mais les poses CONFIRMEES par l'oracle de position
// (filmdec.ScanFilmEquipmentPlacements). Les effectifs changent donc de definition autant que
// de valeur : les anciens comptaient des records dont une bonne part etait du bruit d'ancre.
func ti37Observes() map[uint32]string {
	return map[uint32]string{
		0xbcabbe43: "9 films · 933 poses",
		0x0f5716ff: "8 films · 306",
		0xcaaadcb0: "9 films · 293",
		0xaada07f3: "6 films · 177",
		0x273fe0eb: "5 films · 105",
		0x8c77ffe7: "3 films · 83",
		0x7ca85adc: "4 films · 77",
		0xeef5d48d: "3 films · 70",
		0x72199cba: "3 films · 60  (CAPTEUR, rang 22 famille B)",
		0x4396db42: "5 films · 51",
		0x32d97758: "2 films · 48",
		0x528fce46: "3 films · 45  (rang 19 famille B — 2e identifiant du mur)",
		0x72b63d69: "2 films · 43  (rang 1 famille A)",
		0x8e2dc574: "3 films · 42  (MUR, rang 19 famille B)",
		0x2974c233: "3 films · 41  (rang 2 famille A)",
		0x430dda48: "2 films · 33",
		0x686b40c9: "3 films · 25  (rang 2 famille A)",
		0x4744d742: "1 film · 4",
		0xb781197a: "1 film · 1",
		0x730dc70f: "1 film · 1",
		0xe7be9f5c: "1 film · 1",
	}
}

// ti37Modules : les modules indexes, du plus probable au plus lourd. La variante `any/` porte
// les tags de definition ; sa plus grosse archive fait 634 Mo, contre 7,4 Go cote `pc/`.
var ti37Modules = []string{
	"any/globals/multiplayer-rtx-new.module",
	"any/globals/multiplayer_r1-rtx-new.module",
	"any/globals/multiplayer_r2-rtx-new.module",
	"any/globals/multiplayer_r3-rtx-new.module",
	"any/globals/levels-rtx-new.module",
	"any/globals/forge-rtx-new.module",
	"any/globals/forge/forge_objects-rtx-new.module",
	"any/globals/common-rtx-new.module",
	"any/globals/globals-rtx-new.module",
}

// TestSondeTI37MppWord32 resout les valeurs observees contre l'index des tags du jeu.
func TestSondeTI37MppWord32(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	cibles := ti37Observes()
	trouve := map[uint32]string{}
	indexes := 0
	for _, rel := range ti37Modules {
		chemin := filepath.Join(root, filepath.FromSlash(rel))
		m, err := himodule.Open(chemin)
		if err != nil {
			t.Logf("module %s illisible : %v", rel, err)
			continue
		}
		n := 0
		for _, f := range m.Files("") {
			n++
			if _, veut := cibles[f.GlobalID]; veut {
				if _, deja := trouve[f.GlobalID]; !deja {
					trouve[f.GlobalID] = fmt.Sprintf("%s (%s)", f.Group, rel)
				}
			}
		}
		indexes += n
		t.Logf("  %-52s %7d tags · cumul resolus %d/%d", rel, n, len(trouve), len(cibles))
		m = nil // un seul module en memoire a la fois
	}
	t.Logf("== %d tags parcourus · %d valeurs sur %d resolues ==", indexes, len(trouve), len(cibles))
	keys := make([]uint32, 0, len(cibles))
	for k := range cibles {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		g, ok := trouve[k]
		if !ok {
			g = "NON RESOLU"
		}
		t.Logf("  %#010x  %-28s  %s", k, g, cibles[k])
	}
}

// TestSondeTI37PaletteSofd lit les palettes de capacites (`sofd`) et rend, pour chacune, la
// suite ORDONNEE des `eqip` qu'elle reference. C'est le pont rang -> equipement : le rang lu
// dans le film par `i48` indexe cette suite, et la RECETTE_LOADOUT §13 nomme les rangs de la
// famille A. Une valeur observee dans un record de creation qui retombe au rang attendu est le
// NOM de l'objet, obtenu sans releve terrain.
//
// LA METHODE DE LECTURE EST GROSSIERE ET ASSUMEE : on balaye les octets du tag a la recherche
// de mots de 32 bits qui sont des GlobalID de tags `eqip`, dans l'ordre d'apparition. Le format
// du `sofd` n'est pas decode ; ce qui fait la selectivite, c'est l'appartenance a l'ensemble des
// `eqip` du jeu — quelques centaines de valeurs dans un espace de 2^32.
func TestSondeTI37PaletteSofd(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	m, err := himodule.Open(filepath.Join(root, filepath.FromSlash("any/globals/globals-rtx-new.module")))
	if err != nil {
		t.Skipf("globals illisible : %v", err)
	}
	eqip := map[uint32]bool{}
	var sofd []himodule.File
	groupes := map[string]int{}
	for _, f := range m.Files("") {
		groupes[f.Group]++
		switch f.Group {
		case "eqip":
			eqip[f.GlobalID] = true
		case "sofd":
			sofd = append(sofd, f)
		}
	}
	t.Logf("== INDEX : %d tags `eqip` · %d tags `sofd` (denominateur anti-hasard : 11 valeurs"+
		" observees sur 11 tombent dans les %d `eqip`) ==", len(eqip), len(sofd), len(eqip))
	obs := ti37Observes()
	sort.Slice(sofd, func(i, j int) bool { return sofd[i].GlobalID < sofd[j].GlobalID })
	for _, f := range sofd {
		raw, err := m.Extract(f)
		if err != nil {
			t.Logf("  sofd %#010x illisible : %v", f.GlobalID, err)
			continue
		}
		ordre := ti37ScanEqip(raw, eqip)
		t.Logf("  sofd %#010x — %d octets · %d `eqip` references", f.GlobalID, len(raw), len(ordre))
		for rang, id := range ordre {
			marque := ""
			if c, ok := obs[id]; ok {
				marque = "  <== OBSERVE  " + c
			}
			t.Logf("      rang %2d  %#010x%s", rang, id, marque)
		}
	}
}

// ti37ScanEqip rend les GlobalID de tags `eqip` rencontres dans raw, dans l'ordre, sans doublon.
func ti37ScanEqip(raw []byte, eqip map[uint32]bool) []uint32 {
	var out []uint32
	vu := map[uint32]bool{}
	for i := 0; i+4 <= len(raw); i++ {
		v := binary.LittleEndian.Uint32(raw[i:])
		if eqip[v] && !vu[v] {
			vu[v] = true
			out = append(out, v)
		}
	}
	return out
}
