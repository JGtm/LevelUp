// tmp_mapcloud — THROWAWAY : le nuage de positions keyframe d'un film dessine-t-il
// la carte en 2D ? Décode toutes les positions full-state (package positions,
// match-level, sans attribution) puis projette sur les 3 plans (XY/XZ/YZ) en grille
// ASCII de densité. L'axe "up" (faible variance) se repère ; le plan horizontal
// montre l'empreinte jouable.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_mapcloud <dir_chunks>
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/positions"
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

// firstTimestampUs lit le premier timestamp de paquet (header 16o, ts u64 @+8)
// pour estimer le startMS du chunk.
func chunkStartMS(d []byte) int {
	if len(d) >= 16 {
		ts := binary.LittleEndian.Uint64(d[8:])
		return int(ts / 1000)
	}
	return 0
}

func chunkType(d []byte) int {
	if len(d) >= 2 {
		return int(binary.LittleEndian.Uint16(d))
	}
	return -1
}

func main() {
	dir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, _ := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	sort.Strings(files)

	var chunks []positions.ChunkInput
	for _, f := range files {
		d := inflate(f)
		if d == nil {
			continue
		}
		// On force ChunkType=2 : type2Payload re-scanne et renvoie nil si pas de
		// paquet TYPE_2 dans ce chunk (pas de manifeste ici pour le vrai label).
		_ = chunkType
		chunks = append(chunks, positions.ChunkInput{
			Data: d, StartMS: chunkStartMS(d), ChunkType: 2,
		})
	}
	pts := positions.DecodeKeyframePositions(chunks)
	fmt.Printf("film=%s chunks=%d positions=%d\n", filepath.Base(dir), len(chunks), len(pts))
	if len(pts) == 0 {
		return
	}

	// Bornes + variance par axe pour identifier l'axe "up".
	var mn, mx [3]float32
	mn = [3]float32{1e9, 1e9, 1e9}
	mx = [3]float32{-1e9, -1e9, -1e9}
	var sum, sum2 [3]float64
	get := func(p positions.PlayerPosition, a int) float32 {
		return [3]float32{p.X, p.Y, p.Z}[a]
	}
	for _, p := range pts {
		for a := 0; a < 3; a++ {
			v := get(p, a)
			if v < mn[a] {
				mn[a] = v
			}
			if v > mx[a] {
				mx[a] = v
			}
			sum[a] += float64(v)
			sum2[a] += float64(v) * float64(v)
		}
	}
	n := float64(len(pts))
	axn := []string{"X", "Y", "Z"}
	fmt.Println("axe   min      max      span     stddev")
	for a := 0; a < 3; a++ {
		mean := sum[a] / n
		variance := sum2[a]/n - mean*mean
		fmt.Printf(" %s  %8.2f %8.2f %8.2f %8.2f\n",
			axn[a], mn[a], mx[a], mx[a]-mn[a], math.Sqrt(math.Max(0, variance)))
	}

	// Plan horizontal = les 2 axes de plus grand span (l'axe up a le plus petit).
	spans := []struct {
		a    int
		span float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(spans, func(i, j int) bool { return spans[i].span > spans[j].span })
	ax, ay := spans[0].a, spans[1].a
	fmt.Printf("\nPlan horizontal: %s (horiz) vs %s (vert), up=%s\n", axn[ax], axn[ay], axn[spans[2].a])
	renderGrid(pts, get, ax, ay, mn, mx)
}

// renderGrid trace une grille ASCII de densité (60x30) sur le plan (ax, ay).
func renderGrid(pts []positions.PlayerPosition, get func(positions.PlayerPosition, int) float32, ax, ay int, mn, mx [3]float32) {
	const W, H = 70, 32
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX := mx[ax] - mn[ax]
	spanY := mx[ay] - mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	for _, p := range pts {
		gx := int(float64(get(p, ax)-mn[ax]) / float64(spanX) * (W - 1))
		gy := int(float64(get(p, ay)-mn[ay]) / float64(spanY) * (H - 1))
		gy = H - 1 - gy // y vers le haut
		if gx >= 0 && gx < W && gy >= 0 && gy < H {
			grid[gy][gx]++
		}
	}
	ramp := []byte(" .:-=+*#%@")
	maxc := 0
	for _, row := range grid {
		for _, c := range row {
			if c > maxc {
				maxc = c
			}
		}
	}
	fmt.Println("+" + string(bytes.Repeat([]byte("-"), W)) + "+")
	for _, row := range grid {
		line := make([]byte, W)
		for x, c := range row {
			if c == 0 {
				line[x] = ' '
				continue
			}
			idx := 1 + int(float64(c)/float64(maxc)*float64(len(ramp)-2))
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			line[x] = ramp[idx]
		}
		fmt.Println("|" + string(line) + "|")
	}
	fmt.Println("+" + string(bytes.Repeat([]byte("-"), W)) + "+")
	fmt.Printf("densite max/cellule=%d, total points=%d\n", maxc, len(pts))
}
