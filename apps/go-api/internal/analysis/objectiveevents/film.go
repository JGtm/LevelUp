// Package objectiveevents — décodage des timelines d'events objectif (CTF
// captures, Strongholds/KOTH zones, Oddball crâne) depuis les chunks film Halo,
// vers des []domain.ObjectiveEvent (mode-agnostique, cf.
// .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).
//
// Algos PURS : zéro accès DB, zéro Streamlit. L'IO (lecture des chunks/manifest,
// résolution xuid->team via match_participants) est fournie par l'appelant via
// FilmSource + Roster. Le CLI diagnostic v3 (shadow) câble la source disque + le
// roster DB ; les tests câblent une source disque sur data/cache.
//
// film.go porte les PRIMITIVES de décodage adaptées des décodeurs jetables
// validés (tmp_film_explore/{filmx,ctfcap,ctfsig,t2score,firemap}) :
//   - load+zlib d'un chunk ; marche des paquets 16 octets ;
//   - extraction du payload TYPE_2 ;
//   - extraction des events footer th=10 (interactions objectif horodatées) ;
//   - détection des bursts de capture CTF (échelle 6-tiers) avec match_ms.
//
// Faits de décode VALIDÉS (RESEARCH_THEATER_RE.md §M, §M-ter) :
//   - footer = chunk de plus haut index, chunk_type 3 ; ses events th=10 = les
//     interactions objectif (t=horloge BE @octets 48-51, slot b36, team b37/b55,
//     xuid). Le champ team du film est NON FIABLE sur certains matchs -> l'équipe
//     canonique vient du roster (xuid->team_id de match_participants).
//   - capture CTF = une FRAME re-transmettant la table objectif complète
//     (tiers==6) ; team de la capture = l'event th=10 de t MAX dans le cluster
//     coïncident. objective_id (zone/colline) non récupérable -> NULL.
package objectiveevents

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"sort"
)

// Packet types du conteneur film (en-tête 16 octets : [type u16le][b2][b3][size
// u32le][us u64le]). Seuls ceux consommés ici sont nommés.
const (
	packetFrame = 0  // FRAME : snapshot/delta d'état horodaté (us)
	packetType2 = 2  // TYPE_2 : snapshot game-state par chunk (score byte-aligné)
	packetEnd   = 7  // CHUNK_END
	packetHdr   = 16 // taille de l'en-tête de paquet
)

// Bornes plausibles d'un xuid Halo (filtre anti-bruit du scan footer).
const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

// readBitsBE lit n bits big-endian (MSB-first) à partir de bitPos. Adapté de
// tmp_film_explore (identique dans filmx/ctfcap/t2score).
func readBitsBE(data []byte, bitPos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		bi := (bitPos + i) / 8
		if bi >= len(data) {
			return r
		}
		off := 7 - ((bitPos + i) % 8)
		bit := uint64((data[bi] >> uint(off)) & 1)
		r = (r << 1) | bit
	}
	return r
}

// readByteAtBit lit un octet à un offset BIT arbitraire (gère le non-alignement).
func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[bi]
	}
	return data[bi]<<off | data[bi+1]>>(8-off)
}

// readU64LEAtBit lit un uint64 little-endian à un offset bit arbitraire.
func readU64LEAtBit(data []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(data, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

// decompressChunk renvoie le contenu décompressé d'un chunk film. Les chunks
// sont compressés en zlib (magic 0x78) ; un chunk non compressé est renvoyé tel
// quel (robustesse). Adapté du load() de tmp_film_explore.
func decompressChunk(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		r, err := zlib.NewReader(bytes.NewReader(raw))
		if err == nil {
			defer r.Close()
			if inf, err := io.ReadAll(r); err == nil {
				return inf
			}
		}
	}
	return raw
}

// framePacket = un paquet FRAME décodé (us absolu + tranche de payload).
type framePacket struct {
	us   uint64
	size int
	off  int // offset du payload dans data (après l'en-tête 16 octets)
}

// walkFrames marche les paquets 16 octets et renvoie tous les FRAME (type 0) avec
// leur us + offset de payload. Adapté de walkFrames (ctfcap/ctfsig).
func walkFrames(data []byte) []framePacket {
	var out []framePacket
	off := 0
	for off+packetHdr <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		us := binary.LittleEndian.Uint64(data[off+8:])
		if size < 0 || size > len(data) {
			break
		}
		if typ == packetFrame && off+packetHdr+size <= len(data) {
			out = append(out, framePacket{us: us, size: size, off: off + packetHdr})
		}
		off += packetHdr + size
		if typ == packetEnd {
			break
		}
	}
	return out
}

// extractType2 renvoie le payload du premier paquet TYPE_2 (snapshot game-state),
// ou nil si absent. Adapté de extractType2 (ctfcap/t2score).
func extractType2(data []byte) []byte {
	off := 0
	for off+packetHdr <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size < 0 || size > len(data) {
			break
		}
		if typ == packetType2 && off+packetHdr+size <= len(data) {
			return data[off+packetHdr : off+packetHdr+size]
		}
		if typ == packetEnd {
			break
		}
		off += packetHdr + size
	}
	return nil
}

// th10Event = un event highlight de type_hint==10 (interaction objectif) décodé
// depuis le footer. t = horloge match (ms) ; slot = b36 (0-3, stable par xuid) ;
// teamRaw = b55 (NON fiable sur certains matchs -> à confirmer via roster) ; xuid
// = acteur. Structure validée RESEARCH_THEATER_RE.md §M.
type th10Event struct {
	t       int
	slot    int
	teamRaw int
	xuid    uint64
}

// scanTh10Events extrait tous les events th=10 d'un chunk décompressé (typiquement
// le footer chunk_type 3). Adapté de evDump/thTally (filmx) : on localise chaque
// XUID (préfixe 0x2d/0x25, suffixe 0xc0, valeur LE plausible), on cherche le
// end-marker [00 00 2e e0] de son bloc, on recule de 60 octets, on lit th@b47,
// t=BE@b48-51, slot@b36, team@b55. Filtré sur th==10 et dédupliqué par bloc.
func scanTh10Events(data []byte) []th10Event {
	total := len(data) * 8
	var out []th10Event
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := readByteAtBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := readU64LEAtBit(data, xstart)
		if x <= minXUID || x >= maxXUID {
			continue
		}
		seen[xstart] = true
		if ev, ok := decodeTh10Block(data, xstart, total); ok {
			out = append(out, th10Event{ev.t, ev.slot, ev.teamRaw, x})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// decodeTh10Block, à partir du début d'XUID, cherche le end-marker du bloc
// d'event puis décode le bloc de 60 octets le précédant. Renvoie ok=false si le
// bloc n'est pas un th=10. (Le xuid est rempli par l'appelant.)
func decodeTh10Block(data []byte, xstart, total int) (th10Event, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
	for b := xstart; b <= win-32; b++ {
		if readByteAtBit(data, b) == 0 && readByteAtBit(data, b+8) == 0 &&
			readByteAtBit(data, b+16) == 0x2e && readByteAtBit(data, b+24) == 0xe0 {
			ebs := b - 60*8
			if ebs < xstart {
				return th10Event{}, false
			}
			if int(readByteAtBit(data, ebs+47*8)) != 10 {
				return th10Event{}, false
			}
			t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
				int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
			return th10Event{
				t:       t,
				slot:    int(readByteAtBit(data, ebs+36*8)),
				teamRaw: int(readByteAtBit(data, ebs+55*8)),
			}, true
		}
	}
	return th10Event{}, false
}

// ladderTiers = têtes des 6 records de l'échelle de score-contribution CTF (la
// valeur gauche double ; préfixes constants inter-match). Une FRAME re-transmettant
// les 6 tiers = un burst de capture. Adapté de ctfcap.
var ladderTiers = [][]byte{
	{0xa4, 0x00, 0x00, 0x00},
	{0x03, 0x48, 0x00, 0x00, 0x01},
	{0x06, 0x90, 0x00, 0x00, 0x02},
	{0x0d, 0x20, 0x00, 0x00, 0x05},
	{0x1a, 0x40, 0x00, 0x00, 0x0a},
	{0x34, 0x80, 0x00, 0x00, 0x15},
}

// captureMinTiers = nombre de tiers distincts requis pour qualifier un burst de
// capture CTF. tiers==6 = rock-solid (0 manque / 0 faux positif sur 4 matchs de
// ground-truth) ; tiers<6 sur-compte (RESEARCH_THEATER_RE.md §M-ter).
const captureMinTiers = 6

// countDistinctTiers renvoie le nombre de tiers distincts présents dans payload.
func countDistinctTiers(pay []byte) int {
	distinct := 0
	for _, t := range ladderTiers {
		if bytes.Contains(pay, t) {
			distinct++
		}
	}
	return distinct
}

// captureBurst = un burst de capture CTF détecté, horodaté en ms match.
type captureBurst struct {
	matchMS int
}

// scanCaptureBursts détecte les bursts de capture CTF sur un chunk gameplay
// (type 2) décompressé. startMS = start_ms du chunk (manifest) ; on convertit
// l'us de chaque FRAME en ms match via le premier FRAME comme ancre (la première
// frame du chunk = état complet, ignorée). Adapté de ctfcap detect.
func scanCaptureBursts(data []byte, startMS int) []captureBurst {
	frames := walkFrames(data)
	if len(frames) == 0 {
		return nil
	}
	first := frames[0].us
	var out []captureBurst
	for i, f := range frames {
		if i == 0 {
			continue // frame d'état complet de début de chunk
		}
		if f.off+f.size > len(data) {
			continue
		}
		if countDistinctTiers(data[f.off:f.off+f.size]) >= captureMinTiers {
			matchMS := startMS + int(int64(f.us-first)/1000)
			out = append(out, captureBurst{matchMS: matchMS})
		}
	}
	return out
}
