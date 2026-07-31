// tmp_defstate_measure — THROWAWAY. NE COMMIT JAMAIS. Lecture seule.
//
// Mesure empirique du coût réel (en bits) du port bit-exact de FUN_140F44C38
// (consumeBipedDefaultState) sur le record Hydra (start=194126 -> default-state
// commence à bit194132 = start+6). Attendu si vtable[0x60] = TOUTE la default-state :
// arrêt à bit194512 (= 380 bits). Sinon : on imprime l'écart = le résidu manquant.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const (
	hydraStart = 194126 // R(6) typeIndex @194126 (=35)
	defStart   = 194132 // hydraStart + 6 : début de la default-state
	gateBit    = 194512 // R(1) gate @194512 (= defStart + 380)
)

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func main() {
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("keyframe type-2 : %d octets\n\n", len(payload))

	// (1) Mesure du port nu (mediaFrame absent, tail=0).
	filmdec.SetBipedMediaFramePresent(false)
	filmdec.SetBipedDefaultStateTailBits(0)
	br := filmdec.NewBitReader(payload)
	br.Skip(defStart)
	filmdec.ConsumeBipedDefaultStateProbe(br) // wrapper exporté
	stop := br.BitPos()
	consumed := stop - defStart
	fmt.Printf("--- (1) consumeBipedDefaultState nu (mediaFrame=false, tail=0) ---\n")
	fmt.Printf("    début=%d  arrêt=%d  bits consommés=%d  (attendu 380, gate @%d)\n",
		defStart, stop, consumed, gateBit)
	fmt.Printf("    écart vs gate Hydra = %d bits  => %s\n", gateBit-stop, verdict(stop == gateBit))

	// (2) Variante mediaFrame présent.
	filmdec.SetBipedMediaFramePresent(true)
	br2 := filmdec.NewBitReader(payload)
	br2.Skip(defStart)
	filmdec.ConsumeBipedDefaultStateProbe(br2)
	stop2 := br2.BitPos()
	filmdec.SetBipedMediaFramePresent(false)
	fmt.Printf("\n--- (2) avec media-frame quat (iVar15 != -1) ---\n")
	fmt.Printf("    arrêt=%d  bits consommés=%d  écart vs gate=%d\n", stop2, stop2-defStart, gateBit-stop2)

	// (3) Diagnostic du résidu : si le port s'arrête AVANT 194512, combien manque-t-il,
	//     et quelle valeur de tailBits ferait atteindre exactement le gate.
	fmt.Printf("\n--- (3) résolution de l'énigme ---\n")
	if stop == gateBit {
		fmt.Printf("    vtable[0x60] consomme EXACTEMENT 380 bits : il EST toute la default-state.\n")
	} else {
		residue := gateBit - stop
		fmt.Printf("    vtable[0x60] consomme %d bits ; il MANQUE %d bits pour atteindre le gate @%d.\n",
			consumed, residue, gateBit)
		fmt.Printf("    => le résidu est lu AILLEURS (tail config-gated de FUN_141f86704, ou\n")
		fmt.Printf("       un sous-lecteur sous-compté). tailBits=%d ferait atteindre le gate.\n", residue)
	}
}

func verdict(ok bool) string {
	if ok {
		return "EXACT (vtable[0x60] = toute la default-state)"
	}
	return "INEXACT (résidu lu ailleurs)"
}
