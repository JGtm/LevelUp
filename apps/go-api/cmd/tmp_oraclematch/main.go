// tmp_oraclematch — TEST DECISIF : les valeurs de position de l'oracle (x,y,z reconstruites
// par le jeu) apparaissent-elles comme des float32 BRUTS litteraux dans les octets du film ?
// On scanne chaque chunk bit-par-bit, on lit un triplet float32 (BE via bits32At, et LE par
// byteswap), et on cherche s'il EGALE une valeur oracle (tol 0.01 sur les 3 axes SIMULTANEMENT).
// Un match 3-axes par hasard a une proba ~1e-10 -> tout match est une preuve.
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const filmDir = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const oracleCSV = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`

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

func bits32At(d []byte, b int) uint32 {
	bi := b >> 3
	off := uint(b & 7)
	var v uint64
	for i := 0; i < 5; i++ {
		var by uint64
		if bi+i < len(d) {
			by = uint64(d[bi+i])
		}
		v = v<<8 | by
	}
	return uint32((v >> (8 - off)) & 0xffffffff)
}

func f32(u uint32) float32 { return math.Float32frombits(u) }

type triplet struct{ x, y, z float32 }

func loadOracle() []triplet {
	f, _ := os.Open(oracleCSV)
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	seen := map[[3]uint32]bool{}
	var out []triplet
	for sc.Scan() {
		ln := sc.Text()
		if strings.HasPrefix(ln, "#") || strings.HasPrefix(ln, "eid") {
			continue
		}
		p := strings.Split(ln, ",")
		if len(p) < 6 {
			continue
		}
		x, e1 := strconv.ParseFloat(p[3], 32)
		y, e2 := strconv.ParseFloat(p[4], 32)
		z, e3 := strconv.ParseFloat(p[5], 32)
		if e1 != nil || e2 != nil || e3 != nil {
			continue
		}
		fx, fy, fz := float32(x), float32(y), float32(z)
		// filtre anti-piege : rejette les triplets degeneres (0,0,0 et axes quasi-nuls)
		// qui matcheraient n'importe quel float32 denormal. On exige une magnitude reelle
		// sur les 3 axes -> un candidat near-zero ne peut jamais matcher.
		if math.Abs(float64(fx)) < 1.0 || math.Abs(float64(fy)) < 1.0 || math.Abs(float64(fz)) < 1.0 {
			continue
		}
		k := [3]uint32{math.Float32bits(fx), math.Float32bits(fy), math.Float32bits(fz)}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, triplet{fx, fy, fz})
	}
	return out
}

const tol = 0.01
const bucketRes = 0.02 // taille bucket pour l'index sur x

func bucketKey(x float32) int64 { return int64(math.Floor(float64(x) / bucketRes)) }

func finiteInBand(v float32) bool {
	a := math.Abs(float64(v))
	if math.IsNaN(a) || math.IsInf(a, 0) {
		return false
	}
	return a <= 200 // large : ne pas rater un repere translate
}

type match struct {
	chunk      int
	bit        int
	byteAli    bool
	endian     string
	ox, oy, oz float32
	fx, fy, fz float32
}

func main() {
	oracle := loadOracle()
	fmt.Printf("oracle : %d triplets distincts\n", len(oracle))

	// index x -> indices oracle (buckets)
	idx := map[int64][]int{}
	for i, t := range oracle {
		k := bucketKey(t.x)
		idx[k] = append(idx[k], i)
	}

	// lookup : renvoie l'indice oracle dont (x,y,z) matche within tol, ou -1
	lookup := func(x, y, z float32) int {
		if !finiteInBand(x) || !finiteInBand(y) || !finiteInBand(z) {
			return -1
		}
		bk := bucketKey(x)
		for k := bk - 1; k <= bk+1; k++ {
			for _, i := range idx[k] {
				t := oracle[i]
				if math.Abs(float64(x-t.x)) <= tol &&
					math.Abs(float64(y-t.y)) <= tol &&
					math.Abs(float64(z-t.z)) <= tol {
					return i
				}
			}
		}
		return -1
	}

	var matches []match
	const maxKeep = 5000
	for ci := 0; ci <= 27; ci++ {
		p := fmt.Sprintf("%s/chunk_%02d.bin", filmDir, ci)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		d := inflate(p)
		nb := len(d)*8 - 96
		cnt := 0
		for b := 0; b <= nb; b++ {
			ux := bits32At(d, b)
			// BE
			xb := f32(ux)
			if finiteInBand(xb) {
				if oy := lookup(xb, f32(bits32At(d, b+32)), f32(bits32At(d, b+64))); oy >= 0 {
					t := oracle[oy]
					matches = append(matches, match{ci, b, b%8 == 0, "BE", t.x, t.y, t.z, xb, f32(bits32At(d, b+32)), f32(bits32At(d, b+64))})
					cnt++
				}
			}
			// LE (byteswap chaque mot)
			xl := f32(bswap(ux))
			if finiteInBand(xl) {
				yl := f32(bswap(bits32At(d, b+32)))
				zl := f32(bswap(bits32At(d, b+64)))
				if oy := lookup(xl, yl, zl); oy >= 0 {
					t := oracle[oy]
					matches = append(matches, match{ci, b, b%8 == 0, "LE", t.x, t.y, t.z, xl, yl, zl})
					cnt++
				}
			}
			if len(matches) > maxKeep {
				break
			}
		}
		fmt.Printf("chunk_%02d : %d octets inflates, %d matches\n", ci, len(d), cnt)
		if len(matches) > maxKeep {
			fmt.Println("... arret : trop de matches (>5000), probable band trop large")
			break
		}
	}

	fmt.Printf("\n=== TOTAL MATCHES 3-AXES : %d ===\n", len(matches))
	// stats endian / alignement
	be, le, ba := 0, 0, 0
	for _, m := range matches {
		if m.endian == "BE" {
			be++
		} else {
			le++
		}
		if m.byteAli {
			ba++
		}
	}
	fmt.Printf("BE=%d LE=%d | byte-aligned=%d / %d\n", be, le, ba, len(matches))

	// echantillons
	fmt.Println("\n--- echantillons (max 40) ---")
	for i, m := range matches {
		if i >= 40 {
			break
		}
		fmt.Printf("chunk%02d bit%d ali=%v %s : oracle(%.4f,%.4f,%.4f) film(%.4f,%.4f,%.4f)\n",
			m.chunk, m.bit, m.byteAli, m.endian, m.ox, m.oy, m.oz, m.fx, m.fy, m.fz)
	}

	// ecriture fichier
	writeReport(matches, len(oracle))
}

func bswap(u uint32) uint32 {
	return (u&0xff)<<24 | (u&0xff00)<<8 | (u&0xff0000)>>8 | (u&0xff000000)>>24
}

func writeReport(matches []match, norac int) {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "oracle triplets distincts=%d\n", norac)
	fmt.Fprintf(sb, "TOTAL MATCHES 3-AXES tol=0.01 : %d\n\n", len(matches))
	// distribution offset-bit modulo pour voir alignement
	byteAliByEnd := map[string]int{}
	for _, m := range matches {
		if m.byteAli {
			byteAliByEnd[m.endian]++
		}
	}
	fmt.Fprintf(sb, "byte-aligned BE=%d LE=%d\n\n", byteAliByEnd["BE"], byteAliByEnd["LE"])
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].chunk != matches[j].chunk {
			return matches[i].chunk < matches[j].chunk
		}
		return matches[i].bit < matches[j].bit
	})
	for _, m := range matches {
		fmt.Fprintf(sb, "chunk%02d bit%d byteAli=%v %s oracle(%.4f,%.4f,%.4f) film(%.4f,%.4f,%.4f)\n",
			m.chunk, m.bit, m.byteAli, m.endian, m.ox, m.oy, m.oz, m.fx, m.fy, m.fz)
	}
	os.WriteFile(scratch+"/float32_match.txt", []byte(sb.String()), 0644)
	fmt.Println("\n-> ecrit", scratch+"/float32_match.txt")
}

const scratch = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`

var _ = binary.LittleEndian
