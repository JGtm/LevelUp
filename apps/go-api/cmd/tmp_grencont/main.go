// tmp_grencont — THROWAWAY : test de CONTINUITE TEMPORELLE des composants lus par le walk
// SEQUENTIEL, avec CONTROLE POSITIF et ALEA DE REFERENCE.
//
// Principe : entre deux trames consecutives (~125 ms) d'un MEME slot joueur, un compte de
// grenades / un index de capacite / une charge ne change quasi jamais (au plus quelques
// dizaines de fois sur tout le film). Le taux de STABILITE echantillon-a-echantillon est
// donc le discriminant :
//   - une valeur REELLE  -> stabilite proche de 100 %
//   - du BRUIT sur k bits -> stabilite proche de 1/2^k (aleatoire uniforme)
//
// CONTROLE POSITIF : i4 object-body-vitality et i5 object-shield-vitality sont deja
// decodes et valides ; ils sont mesures par le MEME chemin, avec la MEME metrique. Si
// eux aussi sortent au niveau de l'alea, c'est le walk qui est en cause, pas le composant.
//
// ALEA DE REFERENCE : la meme lecture faite a StartBit DECALE de +37 bits (decalage
// arbitraire, non aligne) donne la stabilite d'un champ que l'on SAIT faux.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_grencont [chunkLo chunkHi]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	idLow = 11
	t0Us  = uint64(4537898226)
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// probe decrit un champ a suivre : composant, offset dans le composant, largeur.
type probe struct {
	label string
	comp  int
	off   int
	width int
}

type series struct {
	stable, total int
	distinct      map[uint64]int
}

func main() {
	lo, hi := 1, 27
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	probes := []probe{
		// --- CONTROLE POSITIF : composants deja valides ---
		{"CTRL i04 vie (8 premiers bits)", 4, 0, 8},
		{"CTRL i05 bouclier (8 premiers bits)", 5, 0, 8},
		// --- CIBLES ---
		{"i22 count R(3)", 22, 0, 3},
		{"i22 valeur#0 R(8)", 22, 3, 8},
		{"i47 R(6)", 47, 0, 6},
		{"i47 R(3)", 47, 6, 3},
		{"i48 slot R(3)", 48, 0, 3},
		{"i48 porte R(1)", 48, 3, 1},
		{"i48 index R(6)", 48, 4, 6},
		{"i56 masque R(3)", 56, 0, 3},
		{"i56 charge#0 R(7)", 56, 3, 7},
		// --- ALEA DE REFERENCE : meme composant, decale de 37 bits ---
		{"ALEA i22+37 R(8)", 22, 37, 8},
		{"ALEA i56+37 R(7)", 56, 37, 7},
		{"ALEA i04+37 R(8)", 4, 37, 8},
	}

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	kfChunk := inflate(cache + "/chunk_02.bin")
	var kf []byte
	for _, p := range filmdec.WalkPackets(kfChunk) {
		if p.Type == filmdec.PacketTypeKeyframe {
			kf = p.Payload(kfChunk)
			break
		}
	}
	binds := filmdec.WalkKeyframeWorld(kf)
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	// last[probe][slot] = derniere valeur vue
	last := make([]map[uint32]uint64, len(probes))
	res := make([]*series, len(probes))
	for i := range probes {
		last[i] = map[uint32]uint64{}
		res[i] = &series{distinct: map[uint64]int{}}
	}

	for c := lo; c <= hi; c++ {
		chunk := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if chunk == nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if r.TypeIndex != filmdec.BipedTypeIndex || r.Slot < 512 || r.Slot > 519 {
					continue
				}
				for _, cr := range r.Trace.Comps {
					if !cr.Ported {
						continue
					}
					for pi := range probes {
						if probes[pi].comp != cr.Index {
							continue
						}
						v := bitsAt(pay, cr.StartBit+probes[pi].off, probes[pi].width)
						res[pi].distinct[v]++
						if prev, ok := last[pi][r.Slot]; ok {
							res[pi].total++
							if prev == v {
								res[pi].stable++
							}
						}
						last[pi][r.Slot] = v
					}
				}
			}
		}
	}

	fmt.Printf("=== CONTINUITE TEMPORELLE par slot joueur (chunks %d..%d, film 000d5950) ===\n", lo, hi)
	fmt.Printf("%-38s %8s %9s %8s %10s %10s\n", "champ", "n paires", "stables", "%stab", "%alea 1/2^w", "distincts")
	for i, p := range probes {
		r := res[i]
		if r.total == 0 {
			fmt.Printf("%-38s %8s\n", p.label, "aucune")
			continue
		}
		alea := 100.0 / float64(uint64(1)<<uint(p.width))
		fmt.Printf("%-38s %8d %9d %7.2f%% %9.2f%% %10d\n",
			p.label, r.total, r.stable, 100*float64(r.stable)/float64(r.total), alea, len(r.distinct))
	}

	// Detail : pour i22 count, la valeur la plus frequente et sa part.
	fmt.Println("\n-- part de la valeur modale (un champ reel a une modale forte) --")
	for i, p := range probes {
		r := res[i]
		var tot, best int
		var bv uint64
		for v, n := range r.distinct {
			tot += n
			if n > best {
				best, bv = n, v
			}
		}
		if tot == 0 {
			continue
		}
		fmt.Printf("   %-38s modale=%d part=%.2f%% (n=%d)\n", p.label, bv, 100*float64(best)/float64(tot), tot)
	}

	_ = sort.Ints
}
