package weaponv3

// pi_resolver.go — résolution xuid → player_index (pi) au niveau BIT.
//
// Port de tmp_film_explore/piverify/main.go (méthode acurtis, vérifiée ground
// truth). Principe : dans un chunk film décompressé, le xuid d'un joueur est
// encodé en 8 octets LITTLE-ENDIAN, mais le flux est un bitstream non
// byte-aligné. On recherche donc, AU NIVEAU BIT, le motif 64-bit obtenu en
// relisant ces 8 octets LE comme un entier big-endian (target). Les 5 bits
// IMMÉDIATEMENT AVANT le motif trouvé portent le player_index (0-31).
//
// Cette résolution corrige l'hypothèse v2 (player_index = ordre DB) signalée
// fausse dans .ai/RESEARCH_THEATER_RE.md / PLAN §pi-fix.

import (
	"encoding/binary"
	"strings"
)

// PIBits : largeur du champ player_index (0-31). EXPORTEE parce que la ventilation des tirs
// (`sync/killcollector`) cherche le meme motif avec le meme champ devant lui : une seconde copie
// du 5 divergerait le jour ou l un des deux lecteurs changerait.
const PIBits = 5

// bitReader fournit un accès bit-level MSB-first à un buffer décompressé.
type bitReader struct {
	data  []byte
	total int // nombre total de bits
}

func newBitReader(data []byte) bitReader {
	return bitReader{data: data, total: len(data) * 8}
}

// bit retourne le bit à la position p (MSB-first dans chaque octet), 0 hors borne.
func (b bitReader) bit(p int) int {
	if p < 0 || p >= b.total {
		return 0
	}
	return int((b.data[p>>3] >> uint(7-(p&7))) & 1)
}

// readBits lit n bits à partir de bp, MSB-first.
//
// Lecture par MOT de 64 bits (cf. [wordBitsAt]) des que le depart est positif et la largeur
// tient sur 64 bits ; sinon la boucle d'origine, seule a porter le bourrage a ZERO d'une
// position NEGATIVE (`ResolveXuidToPI` relit les 5 bits qui PRECEDENT le motif : sur un motif
// trouve au tout debut du chunk, elle recule sous zero).
func (b bitReader) readBits(bp, n int) uint64 {
	if bp >= 0 && n >= 0 && n <= 64 {
		return wordBitsAt(b.data, bp, uint(n))
	}
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | uint64(b.bit(bp+i))
	}
	return v
}

// xuidTargetPattern encode un xuid en 8 octets LE puis les relit en big-endian :
// c'est le motif 64-bit à chercher au niveau bit dans le chunk.
func xuidTargetPattern(xuid uint64) uint64 {
	le := make([]byte, 8)
	binary.LittleEndian.PutUint64(le, xuid)
	return binary.BigEndian.Uint64(le)
}

// ResolveXuidToPI cherche, pour chaque xuid du roster, son motif 64-bit dans le
// chunk (déjà décompressé) au niveau bit, et lit les 5 bits précédents → pi.
// Les xuids non trouvés sont absents de la map retournée.
func ResolveXuidToPI(rosterXuids []uint64, chunk []byte) map[uint64]int {
	out := make(map[uint64]int)
	if len(chunk) == 0 {
		return out
	}
	br := newBitReader(chunk)
	for _, x := range rosterXuids {
		if bp, ok := findPattern64(chunk, xuidTargetPattern(x)); ok {
			out[x] = int(br.readBits(bp-PIBits, PIBits))
		}
	}
	return out
}

// ResolveXuidToPIStrings est la variante "xuids décimaux en string".
// Les bots (xuid préfixé "bid" ou non numérique) sont ignorés. Les pi résolus
// sont re-clés sur la string d'origine.
func ResolveXuidToPIStrings(rosterXuids []string, chunk []byte) map[string]int {
	numeric := make([]uint64, 0, len(rosterXuids))
	backRef := make(map[uint64]string, len(rosterXuids))
	for _, s := range rosterXuids {
		v, ok := parseDecimalXuid(s)
		if !ok {
			continue
		}
		numeric = append(numeric, v)
		backRef[v] = s
	}
	resolved := ResolveXuidToPI(numeric, chunk)
	out := make(map[string]int, len(resolved))
	for v, pi := range resolved {
		out[backRef[v]] = pi
	}
	return out
}

// parseDecimalXuid parse un xuid décimal ; rejette les bots (préfixe "bid") et
// toute chaîne non purement numérique.
func parseDecimalXuid(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "bid") {
		return 0, false
	}
	var v uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + uint64(c-'0')
	}
	return v, true
}

// ResolveBest essaie chaque chunk décompressé et fusionne les résultats : le
// PREMIER chunk qui trouve un xuid gagne (les chunks suivants ne l'écrasent
// pas). Renvoie la map couvrant le plus de xuids possible.
func ResolveBest(rosterXuids []uint64, chunks [][]byte) map[uint64]int {
	merged := make(map[uint64]int)
	for _, chunk := range chunks {
		part := ResolveXuidToPI(rosterXuids, chunk)
		for x, pi := range part {
			if _, seen := merged[x]; !seen {
				merged[x] = pi
			}
		}
	}
	return merged
}
