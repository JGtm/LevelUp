// tmp_bufmatch — THROWAWAY : localise le buffer de message capturé live (debugger)
// dans les chunks du film 000d5950. On cherche la signature d'octets dans l'INFLATE
// ET le brut de chaque chunk, puis on identifie le packet-type porteur (par parse des
// headers 16o) + l'offset dans le payload.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// findAll returns all offsets of needle in hay.
func findAll(hay, needle []byte) []int {
	var out []int
	off := 0
	for {
		i := bytes.Index(hay[off:], needle)
		if i < 0 {
			break
		}
		out = append(out, off+i)
		off += i + 1
	}
	return out
}

// packetAt : parse les paquets [Type u16][b2][b3][Size u32][ts u64] et renvoie le
// packet-type + offset-relatif-payload pour une position absolue dans le buffer.
func packetAt(d []byte, pos int) (typ uint16, relOff int, found bool) {
	off := 0
	for off+16 <= len(d) {
		t := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return t, pos - (off + 16), true
		}
		off += 16 + sz
	}
	return 0, 0, false
}

func main() {
	// Signatures du buffer capturé (offsets dans le message de 214 o).
	full, _ := hex.DecodeString("d260440004384bd29ed42c9679f76036028265840c0023483e495a66340f6840c0")
	sig10, _ := hex.DecodeString("04384bd29ed42c9679f7") // offset 4..13
	sig6, _ := hex.DecodeString("9ed42c9679f7")          // offset 8..13 (variant suffix bit-packé)
	sigStart, _ := hex.DecodeString("d260440004384bd2")  // offset 0..7

	sigs := []struct {
		name string
		b    []byte
	}{{"full32", full}, {"sig10", sig10}, {"sig6", sig6}, {"start8", sigStart}}

	fmt.Printf("Recherche du buffer capturé (214o @0x237D0340000) dans les chunks 000d5950\n\n")
	anyHit := false
	for n := 0; n <= 29; n++ {
		path := fmt.Sprintf("%s/chunk_%02d.bin", cache, n)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		inf := inflate(raw)
		for _, kind := range []struct {
			label string
			data  []byte
		}{{"inflate", inf}, {"raw", raw}} {
			for _, s := range sigs {
				hits := findAll(kind.data, s.b)
				for _, h := range hits {
					anyHit = true
					t, rel, ok := packetAt(kind.data, h)
					pk := "(hors packet/raw)"
					if ok {
						pk = fmt.Sprintf("packet-type=%d payload_off=%d", t, rel)
					}
					fmt.Printf("chunk_%02d [%s] sig=%-7s @abs=%d (0x%x) %s\n",
						n, kind.label, s.name, h, h, pk)
				}
			}
		}
	}
	if !anyHit {
		fmt.Printf("AUCUN match — le buffer n'est pas un slice verbatim des chunks (le lecteur transforme/extrait).\n")
		fmt.Printf("=> il faut décompiler FUN_14080AADE pour le framing.\n")
	}
}
