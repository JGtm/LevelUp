package filmdec

// registre_hachage_research_test.go — LOT D1, VOLET « ET SI LES NOMS ETAIENT HACHES ? »
//
// LE VOLET « EN CLAIR » A DEJA REPONDU NON (TestD1NomsEnClair, avec temoin positif : les noms de
// composant, eux, se trouvent). Reste l'autre moitie de la question : le film pourrait declarer
// ses evenements par une EMPREINTE plutot que par un nom.
//
// LA RECETTE DU REGISTRE NE HACHE RIEN — c'est mesure, pas suppose. Le champ `kind` (u32 a
// slot+0), seul candidat d'empreinte du format de slot, vaut 0 sur 1 066 des 1 067 slots nommes
// (TestChunk00Kind). Le registre de composants est donc en CLAIR, sans empreinte accompagnante :
// « meme recette » signifie « en clair », et le volet hache n'a pas de precedent a imiter.
//
// ON LE TESTE QUAND MEME, ET AVEC UN TAUX DE FAUX POSITIFS MESURE, PAS CALCULE. Pour chacune de
// onze fonctions de hachage 32 bits usuelles plus deux 64 bits, on cherche l'empreinte de chaque
// nom dans chunk_00, aux deux boutismes, a tout offset (pas seulement aligne). Trois populations
// passent au meme tamis :
//   CIBLES  — les noms d'evenement du catalogue de l'exe ;
//   TEMOIN+ — les noms de COMPOSANT, dont on sait qu'ils sont dans ce chunk (en clair) : si le
//             format hachait aussi les composants, ce groupe s'allumerait ;
//   TEMOIN- — des leurres de meme forme et de meme longueur, obtenus en permutant deux lettres
//             des noms cibles. Leur taux de touche EST le taux de faux positifs du tamis sur ce
//             flux precis. Aucun seuil theorique n'est invoque.
//
// Garde CHUNK00_FILMS. Aucun code de production touche.

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fonctionHachage est une fonction de hachage candidate, nommee.
type fonctionHachage struct {
	nom string
	f   func(string) uint32
}

// hachages32 rassemble les fonctions 32 bits usuelles des formats de jeu.
func hachages32() []fonctionHachage {
	crcC := crc32.MakeTable(crc32.Castagnoli)
	return []fonctionHachage{
		{"fnv1a32", func(s string) uint32 { h := fnv.New32a(); _, _ = h.Write([]byte(s)); return h.Sum32() }},
		{"fnv1-32", func(s string) uint32 { h := fnv.New32(); _, _ = h.Write([]byte(s)); return h.Sum32() }},
		{"crc32ieee", func(s string) uint32 { return crc32.ChecksumIEEE([]byte(s)) }},
		{"crc32c", func(s string) uint32 { return crc32.Checksum([]byte(s), crcC) }},
		{"djb2", hashDJB2},
		{"djb2xor", hashDJB2Xor},
		{"sdbm", hashSDBM},
		{"elf", hashELF},
		{"jenkins", hashJenkins},
		{"murmur3", hashMurmur3},
		{"adler-like", hashSomme},
	}
}

func hashDJB2(s string) uint32 {
	h := uint32(5381)
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return h
}

func hashDJB2Xor(s string) uint32 {
	h := uint32(5381)
	for i := 0; i < len(s); i++ {
		h = h*33 ^ uint32(s[i])
	}
	return h
}

func hashSDBM(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = uint32(s[i]) + h<<6 + h<<16 - h
	}
	return h
}

func hashELF(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = h<<4 + uint32(s[i])
		if g := h & 0xf0000000; g != 0 {
			h ^= g >> 24
			h &^= g
		}
	}
	return h
}

func hashJenkins(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h += uint32(s[i])
		h += h << 10
		h ^= h >> 6
	}
	h += h << 3
	h ^= h >> 11
	h += h << 15
	return h
}

func hashMurmur3(s string) uint32 {
	const c1, c2 = 0xcc9e2d51, 0x1b873593
	h := uint32(0)
	n := len(s) / 4 * 4
	for i := 0; i < n; i += 4 {
		k := binary.LittleEndian.Uint32([]byte(s[i : i+4]))
		k *= c1
		k = k<<15 | k>>17
		k *= c2
		h ^= k
		h = h<<13 | h>>19
		h = h*5 + 0xe6546b64
	}
	var k uint32
	for i := len(s) - 1; i >= n; i-- {
		k = k<<8 | uint32(s[i])
	}
	if k != 0 {
		k *= c1
		k = k<<15 | k>>17
		k *= c2
		h ^= k
	}
	h ^= uint32(len(s))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func hashSomme(s string) uint32 {
	a, b := uint32(1), uint32(0)
	for i := 0; i < len(s); i++ {
		a = (a + uint32(s[i])) % 65521
		b = (b + a) % 65521
	}
	return b<<16 | a
}

// indexU32 construit l'ensemble de tous les u32 presents dans data, aux deux boutismes et a tout
// offset (pas seulement aligne sur 4). C'est le tamis le plus permissif possible : s'il ne trouve
// rien, aucun tamis plus fin ne trouvera.
func indexU32(data []byte) map[uint32]bool {
	m := make(map[uint32]bool, len(data))
	for i := 0; i+4 <= len(data); i++ {
		m[binary.LittleEndian.Uint32(data[i:])] = true
		m[binary.BigEndian.Uint32(data[i:])] = true
	}
	return m
}

// leurres fabrique un leurre par nom en permutant deux lettres interieures : meme longueur, meme
// alphabet, meme forme — mais un nom qui n'existe pas dans le moteur.
func leurres(noms []string) []string {
	out := make([]string, 0, len(noms))
	for _, n := range noms {
		if len(n) < 6 {
			out = append(out, n)
			continue
		}
		b := []byte(n)
		i, j := 2, len(b)-3
		b[i], b[j] = b[j], b[i]
		out = append(out, string(b))
	}
	return out
}

// TestD1NomsHaches cherche l'empreinte des noms d'evenement dans chunk_00, avec temoins.
func TestD1NomsHaches(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		idx := indexU32(data)
		composants := nomsDeComposants(data)
		t.Logf("=== %s === %d u32 distincts dans chunk_00 (les deux boutismes, tout offset) ;"+
			" %d noms de composant, %d noms d'evenement, %d leurres",
			filepath.Base(dir), len(idx), len(composants), len(eventNomsCibles),
			len(eventNomsCibles))
		t.Logf("  fonction | evenements touches | composants touches | LEURRES (faux positifs)")
		for _, h := range hachages32() {
			ev, evNoms := compter(idx, h.f, eventNomsCibles)
			co, _ := compter(idx, h.f, composants)
			le, _ := compter(idx, h.f, leurres(eventNomsCibles))
			detail := ""
			if len(evNoms) > 0 {
				detail = " -> " + strings.Join(evNoms, ",")
			}
			t.Logf("  %-10s | %2d / %2d%s | %3d / %3d | %2d / %2d",
				h.nom, ev, len(eventNomsCibles), detail, co, len(composants),
				le, len(eventNomsCibles))
		}
		t.Log("  LECTURE : le taux de touche des LEURRES est le taux de faux positifs du tamis" +
			" sur ce flux. Un hachage ne conclut que si les evenements le depassent nettement" +
			" ET que les composants s'allument aussi (le format hacherait alors ses noms).")
	}
}

// nomsDeComposants rend les noms de composant distincts du registre, tries.
func nomsDeComposants(data []byte) []string {
	vus := map[string]bool{}
	for _, s := range parsedSpans(data) {
		vus[slotName(data, s.off)] = true
	}
	out := make([]string, 0, len(vus))
	for n := range vus {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// compter rend le nombre de noms dont l'empreinte est presente, et la liste (bornee) des noms
// touches.
func compter(idx map[uint32]bool, f func(string) uint32, noms []string) (int, []string) {
	n := 0
	var touches []string
	for _, nom := range noms {
		if idx[f(nom)] {
			n++
			if len(touches) < 5 {
				touches = append(touches, fmt.Sprintf("%s=0x%08x", nom, f(nom)))
			}
		}
	}
	return n, touches
}
