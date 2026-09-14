package filmdec

// registry_entree_jeu_test.go — LE REGISTRE COMMENCE A L'OCTET 8 (lot 1.2).
//
// # LA QUESTION
//
// `registry.go` a longtemps lu le registre comme une suite de slots de 260 octets a partir de
// l'octet 0, layout `[u32 kind][u32 flags][nom @ +8]`. Le jeu lit des ENTREES de 0x104 octets a
// partir de l'octet 8, layout `[nom @ +0x00][u32 niveau @ +0x100]` (`FUN_142e2c690`, cf.
// keyframe_fullstate_loop.go). Les NOMS tombent au meme octet dans les deux lectures ; les
// NIVEAUX, non : le `flags` lu en `slot+4` est le niveau du composant PRECEDENT.
//
// # CE QUE CE FICHIER MESURE, PUIS VERROUILLE
//
//   - `TestRegistreNiveauxVoisinsCensus` (item 1.2.1) : la mesure, faite sur les OCTETS BRUTS et
//     non sur l'API — elle se lit donc des DEUX cotes du correctif, sans rien changer d'elle.
//     Elle publie, par bobine et par archetype, combien de composants voient leur niveau bouger,
//     et croise ce compte avec les lecteurs qui consomment reellement un niveau.
//   - `TestRegistreEntreeDuJeu` (item 1.2.5) : la preuve d'equivalence — le niveau du composant
//     `i` rendu par le parse est le u32 que l'ancienne lecture rendait pour `i+1` — et le fait
//     structurel qui l'accompagne : apres la suite nommee, l'entree de terminaison est
//     ENTIEREMENT nulle (l'ancienne regle devait en exempter quatre octets, qui n'etaient rien
//     d'autre que le niveau du dernier composant nomme).

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"
)

// niveauxBobines : les bobines par build qui portent leur `chunk_00` (celles du lot 0.A.2).
// `minifilm_000d5950` n'en a pas, donc aucun registre a y mesurer.
func niveauxBobines() []string { return closureMiniFilms() }

// lireRegistreBobine rend les octets INFLATES du `chunk_00` d'une bobine de `replay/testdata`.
func lireRegistreBobine(t *testing.T, court string) []byte {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	data, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 de %s illisible : %v", court, err)
	}
	return data
}

// niveauAncienneLecture rend le `flags` que le layout `[u32 kind][u32 flags][nom @ +8]` lisait
// pour le slot `i` du bloc `b` : le u32 LE a `b*archetypeBlockSize + i*registrySlotSize + 4`.
// Zero hors du tampon.
func niveauAncienneLecture(d []byte, b, i int) uint32 {
	off := b*archetypeBlockSize + i*registrySlotSize + 4
	if off < 0 || off+4 > len(d) {
		return 0
	}
	return binary.LittleEndian.Uint32(d[off:])
}

// archetypesMesures : les archetypes cites par le lot 1.2 (ti=9, 11, 12, 35, 40, 42, 43).
func archetypesMesures() []int { return []int{9, 11, 12, 35, 40, 42, 43} }

// lecteursQuiConsommentLeNiveau : les composants dont le deser de `consumeByName` utilise
// l'argument `level`, releves le 2026-09-14 (`grep -n level traverse.go`). TOUT le reste du
// dispatch ignore le niveau : un niveau qui bouge n'y change pas un bit, et c'est ce qui borne
// l'effet du correctif.
func lecteursQuiConsommentLeNiveau() map[string]string {
	return map[string]string{
		"crew-order-component":        "traverse.go quantAxisWidth(level)",
		"tacmap-poiiconoffset":        "traverse.go quantAxisWidth(level)",
		"tacmap-poiicon":              "traverse.go quantAxisWidth(level)",
		"flock-destination-component": "traverse.go quantAxisWidth(level)",
		compPlayerDesiredRespawnLoc:   "traverse.go consumePlayerDesiredRespawnLocation(br, level)",
		"flock-position-component":    "traverse.go consumeFlockPosition(br, uint(level))",
		"asset-transform-component":   "traverse.go quantAxisWidth(level) x5",
	}
}

// TestRegistreNiveauxVoisinsCensus (1.2.1) publie la distribution des niveaux voisins.
func TestRegistreNiveauxVoisinsCensus(t *testing.T) {
	for _, court := range niveauxBobines() {
		d := lireRegistreBobine(t, court)
		reg, err := ParseRegistryChunk(d)
		if err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		t.Logf("======== %s — %d octets, %d blocs d'archetype ========",
			court, len(d), len(reg.Archetypes))
		censusGlobal(t, d, reg)
		for _, ti := range archetypesMesures() {
			censusArchetype(t, d, reg, ti)
		}
	}
}

// censusGlobal compte, sur TOUT le registre, les composants dont le niveau change, et detaille
// ceux dont un deser consomme reellement le niveau.
func censusGlobal(t *testing.T, d []byte, reg *Registry) {
	t.Helper()
	consommateurs := lecteursQuiConsommentLeNiveau()
	total, change, sensibles, sensiblesChange := 0, 0, 0, 0
	var detail []string
	for b, a := range reg.Archetypes {
		for i, nom := range a.Components {
			total++
			avant, apres := niveauAncienneLecture(d, b, i), niveauAncienneLecture(d, b, i+1)
			if avant != apres {
				change++
			}
			if _, sensible := consommateurs[nom]; !sensible {
				continue
			}
			sensibles++
			marque := "="
			if avant != apres {
				sensiblesChange++
				marque = "CHANGE"
			}
			detail = append(detail, fmt.Sprintf("ti=%-2d i%-2d %-48s ancien L=%d -> JEU L=%d  %s",
				b, i, nom, avant, apres, marque))
		}
	}
	t.Logf("  GLOBAL : %d composants · %d niveaux changent (%.1f %%) · %d instances consomment "+
		"le niveau, dont %d changent", total, change, 100*float64(change)/float64(total),
		sensibles, sensiblesChange)
	for _, l := range detail {
		t.Logf("    CONSOMMATEUR %s", l)
	}
}

// censusArchetype publie le detail d'UN archetype.
func censusArchetype(t *testing.T, d []byte, reg *Registry, ti int) {
	t.Helper()
	a, ok := reg.Archetype(ti)
	if !ok {
		t.Logf("  ti=%-2d ABSENT du registre", ti)
		return
	}
	var diffs []string
	for i, nom := range a.Components {
		avant, apres := niveauAncienneLecture(d, ti, i), niveauAncienneLecture(d, ti, i+1)
		if avant != apres {
			diffs = append(diffs, fmt.Sprintf("i%-2d %-58s ancien L=%d -> JEU L=%d", i, nom, avant, apres))
		}
	}
	t.Logf("  ti=%-2d %3d composants · %d niveaux changent", ti, len(a.Components), len(diffs))
	for _, l := range diffs {
		t.Logf("      %s", l)
	}
}
