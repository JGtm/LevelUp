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

// ti37Observes : les valeurs de `mpp-word32` relevees dans les records de creation, avec le
// film et le compte qui les ont produites. Recopiees de la mesure amont — c'est la PIECE.
func ti37Observes() map[uint32]string {
	return map[uint32]string{
		0xbcabbe43: "000d5950:49 · 00162144:128",
		0xcaaadcb0: "000d5950:53 · 00162144:63",
		0x0f5716ff: "000d5950:60 · 00162144:19",
		0xaada07f3: "000d5950:31",
		0x8c77ffe7: "000d5950:30",
		0xeef5d48d: "000d5950:25",
		0x528fce46: "000d5950:15",
		0x72199cba: "000d5950:11",
		0x686b40c9: "00162144:10",
		0x273fe0eb: "00162144:8",
		0x4396db42: "00162144:1",
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
