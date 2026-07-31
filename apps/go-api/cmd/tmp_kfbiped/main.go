// tmp_kfbiped — décode un record de spawn BIPED (joueur) capturé live (biped_buf2.bin,
// slot 0x23B ti=35 @bit 107) et EXTRAIT sa position via le hook emitPos du default-state
// biped. Preuve que le keyframe/spawn porte la position des joueurs.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfbiped <biped_buf.bin> <bitpos> [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func main() {
	buf, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	bitpos, _ := strconv.Atoi(os.Args[2])
	film := defFilm
	if len(os.Args) > 3 {
		film = os.Args[3]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(film + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	filmdec.SetRecordStateParam(2)
	filmdec.SetFilmComponentCorruptionCheck(true) // mode film
	filmdec.SetBipedDefaultStateDecodeMovement(true)

	// SWEEP de la largeur d'axe i0 (TraversalPrecision) : la largeur qui décode le plus
	// profond (desync le plus tardif) = la bonne pour cette map/idx.
	fmt.Println("--- sweep AxisW i0 (desync le plus tardif = bonne largeur) ---")
	best, bestDesync, bestComps := 0, -1, 0
	for aw := uint(6); aw <= 20; aw++ {
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{aw, aw, aw}}
		br := filmdec.NewBitReader(buf)
		br.SetBitPos(bitpos)
		t := filmdec.TraverseEntity(br, reg, 0)
		fmt.Printf("  AxisW=%-2d ti=%d desync=%d comps=%d fin=%d\n", aw, t.TypeIndex, t.DesyncAt, len(t.Comps), br.BitPos())
		if t.DesyncAt == -1 || t.DesyncAt > bestDesync {
			best, bestDesync, bestComps = int(aw), t.DesyncAt, len(t.Comps)
		}
	}
	fmt.Printf("=> meilleure AxisW=%d (desync=%d comps=%d)\n\n", best, bestDesync, bestComps)

	// décode final avec la meilleure AxisW + capture position (émet pour tout idx)
	filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{uint(best), uint(best), uint(best)}}
	var samples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { samples = append(samples, s) })
	br := filmdec.NewBitReader(buf)
	br.SetBitPos(bitpos)
	t := filmdec.TraverseEntity(br, reg, 0)
	fmt.Printf("record final @bit%d : ti=%d desync=%d comps=%d\n", bitpos, t.TypeIndex, t.DesyncAt, len(t.Comps))
	for i, s := range samples {
		fmt.Printf("POSITION #%d kind=%s vec=[%.2f %.2f %.2f] bit=%d\n", i, s.Kind, s.Vec[0], s.Vec[1], s.Vec[2], s.BitPos)
	}
	for i, c := range t.Comps {
		if i >= 6 {
			break
		}
		fmt.Printf("  comp[%d] i%d %s ported=%v @bit%d\n", i, c.Index, c.Name, c.Ported, c.StartBit)
	}
}
