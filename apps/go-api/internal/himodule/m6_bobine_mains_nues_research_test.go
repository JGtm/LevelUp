//go:build research

// m6_bobine_mains_nues_research_test.go — SONDE M6.3 / M6.4 (campagne « retours rejeu »,
// 2026-09-24) : les pièces qui fondent trois entrées du catalogue d'armes, relues dans les modules
// installés (lecture seule, aucun film ouvert, rien d'écrit). Même méthode et mêmes lecteurs que
// la sonde CA9 (`ca9_unarmed_research_test.go`).
//
// CE QUE L'INSTRUMENT MESURE :
//
//  1. LE NOM que le script Lua global (`hsc*` `A35C6CE9`) donne à chaque tag, relu dans son pool
//     de constantes sous la forme « clé puis TAG(<GlobalID>) » (encodage mesuré par CA9) :
//     `unarmed` = `00007CA9`, `fusion_coil` = `1D63A8CD` (table `WeaponTags`),
//     `forge_fusion_coil_mp` = `E9E7FF79` (table `MiscWeaponTags`), `hotrod` = `2AC9C2FF`,
//     `proto_heatwave` = `230447B1`, `ranked_heatwave` = `5AC6CFB2`.
//  2. LA MOITIÉ BASSE de l'identifiant d'arme du film (`<tag><variante>`), relue dans le corps du
//     tag : la liste des variantes porte le GlobalID du tag suivi de sa première variante
//     (`42C9679F` pour les armes de l'arsenal, témoin : le fusil d'assaut `48C19D2D`).
//  3. LA NATURE DES DEUX BOBINES : chacune déclare le tag de dégât que `damagetag/data/labels.tsv`
//     range en explosion `sb_008_exp_single_small_kineticunsc` — la bobine à fusion UNSC.
//
// Commande (depuis `apps/go-api`, CGO actif ; racine `deploy` par l'environnement, jamais écrite
// dans le dépôt) :
//
//	M6_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy go test -tags=research -count=1 -v \
//	  -run '^TestM6CatalogueBobineMainsNues$' ./internal/himodule/
package himodule_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m6NomsAttendus : la clé Lua qui précède chaque TAG dans le pool du script global.
var m6NomsAttendus = map[uint32]string{
	0x00007CA9: "unarmed",
	0x1D63A8CD: "fusion_coil",
	0xE9E7FF79: "forge_fusion_coil_mp",
	0x2AC9C2FF: "hotrod",
	0x230447B1: "proto_heatwave",
	0x5AC6CFB2: "ranked_heatwave",
	0x71AB0A2C: "rocket_launcher", // témoin (clé déjà relue par CA9)
}

// m6CleDe rend la constante chaîne qui précède IMMÉDIATEMENT l'entier TAG(gid) dans le pool
// (écart mesuré de 9 à 64 octets : la chaîne, puis éventuellement `TAG` ou des constantes nil).
func m6CleDe(lua []byte, gid uint32) string {
	motif := make([]byte, 9)
	motif[0] = 0x02
	binary.BigEndian.PutUint64(motif[1:], uint64(gid))
	idx := bytes.Index(lua, motif)
	if idx < 0 {
		return ""
	}
	for off := idx - 11; off >= 0 && off >= idx-64; off-- {
		if s, fin, ok := ca9Chaine(lua, off); ok && s != "TAG" && fin <= idx && idx-fin <= 2 {
			return s
		}
	}
	return ""
}

// m6Variante rend le mot de 32 bits qui SUIT la première occurrence du GlobalID du tag dans son
// propre corps, hors table des dépendances — la première variante de la liste des variantes.
func m6Variante(tag []byte, gid uint32) (uint32, bool) {
	motif := make([]byte, 4)
	binary.LittleEndian.PutUint32(motif, gid)
	debut := 0x50 + 24*int(binary.LittleEndian.Uint32(tag[0x18:]))
	for off := debut; off+8 <= len(tag); {
		i := bytes.Index(tag[off:], motif)
		if i < 0 {
			return 0, false
		}
		at := off + i
		if at%4 == 0 {
			return binary.LittleEndian.Uint32(tag[at+4:]), true
		}
		off = at + 1
	}
	return 0, false
}

func TestM6CatalogueBobineMainsNues(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("M6_DEPLOY"))
	if racine == "" {
		t.Skip("M6_DEPLOY non posé (racine deploy de l'installation)")
	}
	mc, ic := ca9Index(t, filepath.Join(racine, "any", "globals", "common-rtx-new.module"))
	_, lua := ca9Extraire(t, mc, ic, ca9ScriptLua)
	for gid, attendu := range m6NomsAttendus {
		cle := m6CleDe(lua, gid)
		t.Logf("Lua : %q = TAG(%#08x)", cle, gid)
		if cle != attendu {
			t.Errorf("TAG(%#08x) : clé Lua %q, attendu %q", gid, cle, attendu)
		}
	}
	mg, ig := ca9Index(t, filepath.Join(racine, "any", "globals", "globals-rtx-new.module"))
	for _, gid := range []uint32{0x00007CA9, 0x1D63A8CD, 0xE9E7FF79, 0x48C19D2D} {
		m, idx := mc, ic
		if _, ok := idx[gid]; !ok {
			m, idx = mg, ig
		}
		f, ok := idx[gid]
		if !ok {
			t.Errorf("%#08x absent des modules common et globals", gid)
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			t.Fatalf("extraire %#08x : %v", gid, err)
		}
		v, ok := m6Variante(data, gid)
		t.Logf("tag %#08x : groupe %s, première variante %#08x (lue : %v) -> identifiant de film %#08x%08x",
			gid, f.Group, v, ok, gid, v)
		if f.Group != "weap" || !ok {
			t.Errorf("%#08x : groupe %q, variante lue %v", gid, f.Group, ok)
		}
		if jpt, attendu := m6DegatCinetique[gid]; attendu {
			declare := false
			for _, d := range ca9Deps(data) {
				declare = declare || (d.groupe == "jpt!" && d.gid == jpt)
			}
			t.Logf("  %#08x déclare le dégât %#08x (explosion kineticunsc de labels.tsv) : %v", gid, jpt, declare)
			if !declare {
				t.Errorf("%#08x ne déclare pas le dégât cinétique UNSC %#08x", gid, jpt)
			}
		}
	}
}

// m6DegatCinetique : le tag de dégât que chaque bobine DÉCLARE, et que `damagetag/data/labels.tsv`
// range « OBJET EXPLOSIF PORTÉ, sb_008_exp_single_small_kineticunsc » — la bobine à fusion UNSC
// (`hinf_coil_kinetic`, vignette 44 « UNSC fusion coil »).
var m6DegatCinetique = map[uint32]uint32{
	0xE9E7FF79: 0x0D203522,
	0x1D63A8CD: 0xFEAC2551,
}
