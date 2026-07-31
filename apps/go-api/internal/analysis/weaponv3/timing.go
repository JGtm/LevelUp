package weaponv3

// timing.go — estimateur de timestamp µs-précis depuis l'en-tête de paquet
// 16 octets (réf .ai/RESEARCH_THEATER_RE.md §L, port de cmd/tmp_p2valid).
//
// Le chunk film est une suite de paquets : [Type u16 LE @0][b2][b3]
// [Size u32 LE @4][µs u64 LE @8], suivis d'un payload de `Size` octets à @16.
// Les paquets de Type==0 sont des FRAMES : ils couvrent l'intervalle de bytes
// [off+16, off+16+size] et portent un horodatage µs. Type==7 = CHUNK_END (stop).
//
// L'estimateur ancre le µs du PREMIER FRAME du chunk sur le StartMS du manifeste,
// puis pour une position de byte donnée, retrouve (binary search) le FRAME qui
// la contient et convertit son écart µs en ms. Cela remplace le bucketing
// grossier v2 par un timestamp fin (levier P2 vers 90% de confiance).

import "encoding/binary"

const (
	packetHeaderSize = 16 // [Type u16][b2][b3][Size u32][µs u64]
	frameType        = 0  // Type==0 => FRAME (intervalle horodaté)
	chunkEndType     = 7  // Type==7 => CHUNK_END (arrêt du parcours)
)

// frame — intervalle de bytes [start, end) du payload d'un FRAME et son µs.
type frame struct {
	start int
	end   int
	us    uint64
}

// frameIndex parcourt les paquets 16 octets et collecte les FRAMES (Type==0).
// Stoppe sur Type==7 ou si un paquet déborde du buffer (header ou payload).
func frameIndex(d []byte) []frame {
	var frames []frame
	off := 0
	for off+packetHeaderSize <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		us := binary.LittleEndian.Uint64(d[off+8:])
		if size < 0 || off+packetHeaderSize+size > len(d) {
			break
		}
		if typ == frameType {
			frames = append(frames, frame{
				start: off + packetHeaderSize,
				end:   off + packetHeaderSize + size,
				us:    us,
			})
		}
		off += packetHeaderSize + size
		if typ == chunkEndType {
			break
		}
	}
	return frames
}

// USEstimator construit un estimateur estimateTS(bytePos)->ms pour un chunk.
// Le µs du premier FRAME est ancré sur startMS ; un bytePos est converti via le
// FRAME qui le contient (binary search). Si le chunk ne contient aucun FRAME,
// l'estimateur renvoie toujours float64(startMS).
func USEstimator(chunk []byte, startMS int) func(bytePos int) float64 {
	frames := frameIndex(chunk)
	if len(frames) == 0 {
		return func(int) float64 { return float64(startMS) }
	}
	firstUs := frames[0].us
	return func(bytePos int) float64 {
		// Recherche du dernier FRAME dont start <= bytePos (FRAME contenant).
		lo, hi, idx := 0, len(frames)-1, -1
		for lo <= hi {
			mid := (lo + hi) / 2
			if frames[mid].start <= bytePos {
				idx = mid
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
		if idx < 0 {
			idx = 0
		}
		return float64(startMS) + float64(int64(frames[idx].us-firstUs))/1000.0
	}
}
