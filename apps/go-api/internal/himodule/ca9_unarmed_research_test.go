//go:build research

// ca9_unarmed_research_test.go — SONDE CA9 (campagne « retours rejeu », 2026-09-23) :
// l'identité de l'objet `00007CA9`, lu au 3e emplacement `weapon-state-type-info` (i46) de chaque
// bipède au coup d'envoi et porté par des ramassages natifs de classe ARME à t=0.
//
// CE QUE L'INSTRUMENT MESURE, sur les modules installés (lecture seule, aucun fichier écrit) :
//
//  1. `00007CA9` est un GlobalID de tag du module `globals` : groupe, taille, dépendances. Le
//     témoin est le fusil d'assaut, dont la famille du film (`48C19D2D`) est aussi un GlobalID
//     `weap` — même espace d'identifiants, forme basse ou haute.
//  2. Ses modèles : les `mode` que le `weap` et son `hlmt` déclarent (GlobalID, taille). Une
//     première lecture « à l'œil » (références inline du `hlmt`) avait conclu à un objet sans
//     modèle de rendu : FAUX, la table des dépendances en déclare un. Rendu tel quel, sans
//     verdict d'apparence.
//  3. Le script Lua global (`hsc*` `A35C6CE9`, module `common`) déclare la table `WeaponTags`
//     dont chaque clé est suivie, dans le pool de constantes, de l'entier `TAG(<GlobalID>)` :
//     on relit chaque couple (clé, entier) et on vérifie qu'il pointe une dépendance `weap` du
//     script. La clé qui porte `0x7CA9` est l'identité cherchée.
//
// Encodage du pool de constantes (MESURÉ sur les octets, 2026-09-23) : chaîne = octet de type
// 0x04 + taille sur 8 octets gros-boutiste (longueur + 1) + caractères + NUL ; entier = octet de
// type 0x02 + 8 octets gros-boutiste. Exemple : `unarmed\0` 02 00 00 00 00 00 00 7c a9.
//
// En-tête d'un tag : nombre de dépendances en +0x18 ; table des dépendances en +0x50, entrées de
// 24 octets [groupe u32][nom u32][AssetID u64][GlobalID u32][parent i32].
//
// Commande (depuis `apps/go-api`, CGO actif) :
//
//	CA9_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy go test -tags=research -count=1 -v \
//	  -run '^TestCA9Unarmed$' ./internal/himodule/
package himodule_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

const (
	ca9ID        uint32 = 0x00007CA9
	ca9Temoin    uint32 = 0x48C19D2D // famille du fusil d'assaut lue dans le film
	ca9ScriptLua uint32 = 0xA35C6CE9 // hsc* global (module common) qui porte `WeaponTags`
)

type ca9Dep struct {
	groupe string
	gid    uint32
}

// ca9Deps lit la table des dépendances déclarée en tête d'un tag.
func ca9Deps(tag []byte) []ca9Dep {
	if len(tag) < 0x50 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(tag[0x18:]))
	var out []ca9Dep
	for i := 0; i < n; i++ {
		o := 0x50 + 24*i
		if o+24 > len(tag) {
			break
		}
		g := tag[o : o+4]
		groupe := string([]byte{g[3], g[2], g[1], g[0]})
		out = append(out, ca9Dep{groupe: groupe, gid: binary.LittleEndian.Uint32(tag[o+16:])})
	}
	return out
}

// ca9Index ouvre un module et indexe ses entrées par GlobalID.
func ca9Index(t *testing.T, chemin string) (*himodule.Module, map[uint32]himodule.File) {
	t.Helper()
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatalf("ouvrir %s : %v", filepath.Base(chemin), err)
	}
	t.Cleanup(func() { _ = m.Close() })
	idx := map[uint32]himodule.File{}
	for _, f := range m.Files("") {
		if _, deja := idx[f.GlobalID]; !deja {
			idx[f.GlobalID] = f
		}
	}
	return m, idx
}

func ca9Extraire(t *testing.T, m *himodule.Module, idx map[uint32]himodule.File, id uint32) (himodule.File, []byte) {
	t.Helper()
	f, ok := idx[id]
	if !ok {
		t.Fatalf("GlobalID %#08x absent du module", id)
	}
	data, err := m.Extract(f)
	if err != nil {
		t.Fatalf("extraire %#08x : %v", id, err)
	}
	return f, data
}

// ca9Modeles rend le hlmt d'un objet et les `mode` déclarés par l'objet ou par son hlmt.
func ca9Modeles(t *testing.T, m *himodule.Module, idx map[uint32]himodule.File, objet []byte) (uint32, []uint32) {
	t.Helper()
	var hlmt uint32
	var modes []uint32
	for _, d := range ca9Deps(objet) {
		switch d.groupe {
		case "hlmt":
			if hlmt == 0 {
				hlmt = d.gid
			}
		case "mode":
			modes = append(modes, d.gid)
		}
	}
	if hlmt == 0 {
		t.Fatalf("aucun hlmt en dépendance")
	}
	_, data := ca9Extraire(t, m, idx, hlmt)
	for _, d := range ca9Deps(data) {
		if d.groupe == "mode" {
			modes = append(modes, d.gid)
		}
	}
	return hlmt, modes
}

// ca9WeaponTags relit la table `WeaponTags` du script : clés et entiers TAG(...), dans l'ordre.
func ca9WeaponTags(t *testing.T, lua []byte) ([]string, []uint64) {
	t.Helper()
	ancre := append([]byte{0x04, 0, 0, 0, 0, 0, 0, 0, 0x0b}, []byte("WeaponTags\x00")...)
	debut := bytes.Index(lua, ancre)
	if debut < 0 {
		t.Fatalf("constante `WeaponTags` introuvable")
	}
	var cles []string
	var vals []uint64
	// Lecture STRICTEMENT séquentielle du pool (les constantes y sont contiguës) : une chaîne
	// `TAG` (le nom de la fonction, interné à sa première occurrence) est ignorée ; une autre
	// chaîne devient la clé en attente ; un entier ferme la paire. Fin de table : deux chaînes
	// sans entier entre elles (ici `MPWeaponVariantTags`), ou un octet qui n'est pas une constante.
	// Les constantes nil (octet 0x00 seul) sont sautées.
	attente := ""
	for off := debut + len(ancre); off+9 <= len(lua); {
		if s, fin, ok := ca9Chaine(lua, off); ok {
			off = fin
			if s == "TAG" {
				continue
			}
			if attente != "" {
				break
			}
			attente = s
			continue
		}
		if lua[off] == 0x00 { // constante nil (1 octet) : deux avant le premier entier, mesuré
			off++
			continue
		}
		if lua[off] != 0x02 || attente == "" {
			break
		}
		cles, vals = append(cles, attente), append(vals, binary.BigEndian.Uint64(lua[off+1:]))
		attente, off = "", off+9
	}
	return cles, vals
}

// ca9Chaine relit une constante chaîne seule (identifiant ASCII).
func ca9Chaine(b []byte, off int) (string, int, bool) {
	if off+9 > len(b) || b[off] != 0x04 {
		return "", 0, false
	}
	taille := binary.BigEndian.Uint64(b[off+1:])
	if taille < 2 || taille > 64 || off+9+int(taille) > len(b) {
		return "", 0, false
	}
	s := b[off+9 : off+9+int(taille)]
	if s[len(s)-1] != 0 {
		return "", 0, false
	}
	for _, c := range s[:len(s)-1] {
		if c < 0x21 || c > 0x7e {
			return "", 0, false
		}
	}
	return string(s[:len(s)-1]), off + 9 + int(taille), true
}

func TestCA9Unarmed(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("CA9_DEPLOY"))
	if racine == "" {
		t.Skip("CA9_DEPLOY non posé (racine deploy de l'installation)")
	}
	mg, ig := ca9Index(t, filepath.Join(racine, "any", "globals", "globals-rtx-new.module"))

	// 1. Identité du tag et témoin.
	for _, id := range []uint32{ca9ID, ca9Temoin} {
		f, data := ca9Extraire(t, mg, ig, id)
		hlmt, modes := ca9Modeles(t, mg, ig, data)
		grp := map[string]int{}
		for _, d := range ca9Deps(data) {
			grp[d.groupe]++
		}
		t.Logf("tag %#08x : groupe=%s taille=%d deps=%v hlmt=%#08x", id, f.Group, len(data), grp, hlmt)
		for _, md := range modes {
			if fm, ok := ig[md]; ok {
				t.Logf("    mode %#08x : %d octets décompressés", md, fm.UncompSize)
			} else {
				t.Logf("    mode %#08x : hors du module globals", md)
			}
		}
		if f.Group != "weap" {
			t.Errorf("%#08x : groupe %q, attendu weap", id, f.Group)
		}
	}

	// 3. Le script Lua global et sa table WeaponTags.
	mc, ic := ca9Index(t, filepath.Join(racine, "any", "globals", "common-rtx-new.module"))
	_, lua := ca9Extraire(t, mc, ic, ca9ScriptLua)
	weapDeps := map[uint32]bool{}
	for _, d := range ca9Deps(lua) {
		if d.groupe == "weap" {
			weapDeps[d.gid] = true
		}
	}
	cles, vals := ca9WeaponTags(t, lua)
	resolues, cleCA9 := 0, ""
	for i, c := range cles {
		dep := vals[i] <= 0xffffffff && weapDeps[uint32(vals[i])]
		if dep {
			resolues++
		}
		if vals[i] == uint64(ca9ID) {
			cleCA9 = c
		}
		t.Logf("WeaponTags.%-26s = TAG(%#08x) dépendance weap=%v", c, vals[i], dep)
	}
	t.Logf("WeaponTags : %d clés, %d pointent une dépendance weap du script (%d weap déclarées)",
		len(cles), resolues, len(weapDeps))
	t.Logf("VERDICT : WeaponTags.%s = TAG(%#08x)", cleCA9, ca9ID)
	if len(cles) < 30 || resolues != len(cles) {
		t.Errorf("lecture de WeaponTags non fermée : %d/%d", resolues, len(cles))
	}
	if cleCA9 != "unarmed" {
		t.Errorf("clé portant %#08x : %q", ca9ID, cleCA9)
	}
}
