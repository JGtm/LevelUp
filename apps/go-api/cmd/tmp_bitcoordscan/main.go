// tmp_bitcoordscan — scan de coordonnées au niveau BIT (les films sont bit-packés/
// shiftés). À CHAQUE offset de bit, lit 3 float32 (32 bits MSB-first) et teste si le
// triplet ressemble à une coordonnée monde. Cherche des triplets VARIÉS (trajectoire),
// pas des constantes. Teste plusieurs interprétations de bits (MSB-first, octets inversés).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bitcoordscan [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
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

// bits32At lit 32 bits MSB-first à l'offset binaire b (bit 0 = bit 7 de l'octet 0).
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

func f32(u uint32) float32  { return math.Float32frombits(u) }
func bswap(u uint32) uint32 { return u>>24 | (u>>8)&0xff00 | (u<<8)&0xff0000 | u<<24 }

// coordLike : float32 normaux, ≥2 axes de magnitude de vraie coord [8,3000], tous <5000.
func coordLike(x, y, z float32) bool {
	big := 0
	for _, v := range []float32{x, y, z} {
		a := math.Abs(float64(v))
		if math.IsNaN(a) || math.IsInf(a, 0) || a > 5000 {
			return false
		}
		if a != 0 && a < 1e-10 {
			return false // dénormal
		}
		if a >= 8 {
			big++
		}
	}
	return big >= 2
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	type variant struct {
		name  string
		count int
		vals  [][3]float32
		xh    map[int]int // histogramme de X arrondi (détecte variété vs constantes)
	}
	V := []*variant{
		{name: "MSB-first", xh: map[int]int{}},
		{name: "octets-inversés", xh: map[int]int{}},
	}
	// échelle MONDE Cliffhanger (bornes CE) + au moins un axe franchement loin de 0.
	inMap := func(x, y, z float32) bool {
		if !(x >= -1100 && x <= 300 && y >= -500 && y <= 1200 && z >= -200 && z <= 600) {
			return false
		}
		return math.Abs(float64(x)) > 50 || math.Abs(float64(y)) > 50
	}
	mapCount := 0
	var mapVals [][3]float32
	var mnX, mxX, mnY, mxY, mnZ, mxZ float32 = 1e9, -1e9, 1e9, -1e9, 1e9, -1e9
	totalBits := 0
	for idx := 2; idx <= 26; idx++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))
		nb := len(d) * 8
		totalBits += nb
		for b := 0; b+96 <= nb; b++ {
			u0, u1, u2 := bits32At(d, b), bits32At(d, b+32), bits32At(d, b+64)
			x, y, z := f32(u0), f32(u1), f32(u2)
			if coordLike(x, y, z) {
				V[0].count++
				V[0].xh[int(x)]++
				if len(V[0].vals) < 20 {
					V[0].vals = append(V[0].vals, [3]float32{x, y, z})
				}
				if inMap(x, y, z) {
					mapCount++
					if x < mnX {
						mnX = x
					}
					if x > mxX {
						mxX = x
					}
					if y < mnY {
						mnY = y
					}
					if y > mxY {
						mxY = y
					}
					if z < mnZ {
						mnZ = z
					}
					if z > mxZ {
						mxZ = z
					}
					if len(mapVals) < 16 {
						mapVals = append(mapVals, [3]float32{x, y, z})
					}
				}
			}
			x2, y2, z2 := f32(bswap(u0)), f32(bswap(u1)), f32(bswap(u2))
			if coordLike(x2, y2, z2) {
				V[1].count++
				V[1].xh[int(x2)]++
				if len(V[1].vals) < 20 {
					V[1].vals = append(V[1].vals, [3]float32{x2, y2, z2})
				}
			}
		}
	}
	fmt.Printf(">>> ÉCHELLE MONDE (MSB, boîte Cliffhanger, ≥1 axe>50) : %d triplets | X[%.0f,%.0f] Y[%.0f,%.0f] Z[%.0f,%.0f]\n", mapCount, mnX, mxX, mnY, mxY, mnZ, mxZ)
	fmt.Printf("    exemples monde : %v\n", mapVals[:min(10, len(mapVals))])
	fmt.Printf("bits scannés=%d\n", totalBits)
	for _, v := range V {
		// diversité : combien de valeurs X distinctes (arrondies) ? bcp = trajectoire, peu = constantes
		fmt.Printf("\n=== %s : %d triplets coord-like, %d valeurs X distinctes ===\n", v.name, v.count, len(v.xh))
		// top X répétés
		type kv struct{ x, n int }
		var arr []kv
		for x, n := range v.xh {
			arr = append(arr, kv{x, n})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
		top := ""
		for i := 0; i < 5 && i < len(arr); i++ {
			top += fmt.Sprintf("X=%d×%d ", arr[i].x, arr[i].n)
		}
		fmt.Printf("  X les plus répétés : %s\n", top)
		fmt.Printf("  exemples : %v\n", v.vals[:min(8, len(v.vals))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
