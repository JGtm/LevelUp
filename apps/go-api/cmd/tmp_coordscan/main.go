// tmp_coordscan — TEST hypothèse OpenSpartan (scan d'octets) : le film contient-il des
// float32 ALIGNÉS-OCTETS qui ressemblent à des coordonnées monde (3 consécutifs, finis,
// dans une plage plausible) ? Si oui en grand nombre et regroupés, on peut extraire les
// positions SANS décoder le bit-stream ECS. Scanne tous les chunks décompressés.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_coordscan [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math"
	"os"
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

func f32le(b []byte) float32 {
	return math.Float32frombits(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}
func f32be(b []byte) float32 {
	return math.Float32frombits(uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24)
}

// normalCoord : float32 NORMAL (pas dénormal/NaN/inf), magnitude de VRAIE coordonnée
// (entre 1 et 3000 en abs), OU exactement 0. Exclut le bruit dénormal/1e-41.
func normalCoord(v float32) bool {
	a := math.Abs(float64(v))
	if math.IsNaN(a) || math.IsInf(a, 0) {
		return false
	}
	return a == 0 || (a >= 1.0 && a <= 3000.0)
}

// plausible : les 3 axes normaux ET au moins 2 axes avec |v|>=20 (une vraie position
// s'étale sur la map ; exclut les triplets de petits entiers 2/3 = artefacts d'octets).
func plausible(x, y, z float32) bool {
	if !normalCoord(x) || !normalCoord(y) || !normalCoord(z) {
		return false
	}
	big := 0
	for _, v := range []float32{x, y, z} {
		if math.Abs(float64(v)) >= 20 {
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
	// Plage réelle Cliffhanger (index 0) pour un filtre plus STRICT (comptage séparé).
	inBox := func(x, y, z float32) bool {
		return x >= -1100 && x <= 300 && y >= -500 && y <= 1200 && z >= -200 && z <= 600
	}
	totalBytes := 0
	var leLoose, leBox, beLoose, beBox int
	var exLE, exBE [][3]float32
	for idx := 2; idx <= 26; idx++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))
		totalBytes += len(d)
		for o := 0; o+12 <= len(d); o++ {
			xl, yl, zl := f32le(d[o:]), f32le(d[o+4:]), f32le(d[o+8:])
			if plausible(xl, yl, zl) {
				leLoose++
				if inBox(xl, yl, zl) {
					leBox++
					if len(exLE) < 12 {
						exLE = append(exLE, [3]float32{xl, yl, zl})
					}
				}
			}
			xb, yb, zb := f32be(d[o:]), f32be(d[o+4:]), f32be(d[o+8:])
			if plausible(xb, yb, zb) {
				beLoose++
				if inBox(xb, yb, zb) {
					beBox++
					if len(exBE) < 12 {
						exBE = append(exBE, [3]float32{xb, yb, zb})
					}
				}
			}
		}
	}
	fmt.Printf("octets scannés=%d\n", totalBytes)
	fmt.Printf("LE : plausibles=%d dans-boîte-Cliffhanger=%d\n", leLoose, leBox)
	fmt.Printf("BE : plausibles=%d dans-boîte-Cliffhanger=%d\n", beLoose, beBox)
	fmt.Printf("exemples LE dans-boîte : %v\n", exLE)
	fmt.Printf("exemples BE dans-boîte : %v\n", exBE)
}
