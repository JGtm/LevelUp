// tmp_f32traj — DENSITE + LISSAGE du float32 LE. Pour chaque frame gameplay (chunks 3..26),
// scanne TOUS les bits pour un triplet float32 LE (byteswap) real3D dans la boite oracle serree.
// Dedup par frame, puis suit des trajectoires par plus-proche-voisin temporel. Mesure la
// densite (nb float32 in-box / frame ~ nb joueurs ?) et le lissage (pas 2D) vs oracle
// (0.043/0.042/2.83). Repond : le float32 LE est-il un flux DENSE lisse, ou des ancres sparse ?
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"
)

const film = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const scratch = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255}, {0x88, 0x88, 0x88, 255},
	{0xAA, 0x44, 0x99, 255}, {0x44, 0xAA, 0x99, 255}, {0x99, 0x99, 0x44, 255},
}

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

type frame struct {
	ts  uint64
	pay []byte
}

func listFrames(d []byte) []frame {
	var out []frame
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, frame{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
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
func bswap(u uint32) uint32 {
	return (u&0xff)<<24 | (u&0xff00)<<8 | (u&0xff0000)>>8 | (u&0xff000000)>>24
}
func rdLE(d []byte, b int) float32 { return math.Float32frombits(bswap(bits32At(d, b))) }

// real3D serre : dans la boite oracle x[-6,36] y[-25,27] z[-4,7], 3 axes non-nuls (>0.5).
func real3D(x, y, z float32) bool {
	if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
		return false
	}
	if !(x >= -6 && x <= 36 && y >= -25 && y <= 27 && z >= -4 && z <= 7) {
		return false
	}
	for _, v := range []float32{x, y, z} {
		if math.Abs(float64(v)) < 0.5 {
			return false
		}
	}
	return true
}

type pt struct {
	t       int
	x, y, z float32
}

func dist2D(a, b pt) float64 {
	dx, dy := float64(a.x-b.x), float64(a.y-b.y)
	return math.Sqrt(dx*dx + dy*dy)
}

func main() {
	// 1) collecte tous les float32 LE real3D par frame (dedup spatial 0.05).
	type fpts struct {
		t   int
		pts []pt
	}
	var frames []fpts
	totalHits := 0
	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", film, idx))) {
			tms := int((fr.ts - t0Us) / 1000)
			nb := len(fr.pay)*8 - 96
			seen := map[[3]int32]bool{}
			var ps []pt
			for b := 0; b <= nb; b++ {
				x := rdLE(fr.pay, b)
				if x < -6 || x > 36 || math.Abs(float64(x)) < 0.5 {
					continue
				}
				y, z := rdLE(fr.pay, b+32), rdLE(fr.pay, b+64)
				if !real3D(x, y, z) {
					continue
				}
				k := [3]int32{int32(x * 20), int32(y * 20), int32(z * 20)}
				if seen[k] {
					continue
				}
				seen[k] = true
				ps = append(ps, pt{tms, x, y, z})
				totalHits++
			}
			frames = append(frames, fpts{tms, ps})
		}
	}
	// densite
	sort.Slice(frames, func(i, j int) bool { return frames[i].t < frames[j].t })
	var counts []int
	for _, f := range frames {
		counts = append(counts, len(f.pts))
	}
	sort.Ints(counts)
	med := 0
	if len(counts) > 0 {
		med = counts[len(counts)/2]
	}
	fmt.Printf("frames gameplay=%d ; float32 LE real3D total=%d ; par frame median=%d min=%d max=%d\n",
		len(frames), totalHits, med, counts[0], counts[len(counts)-1])

	// 2) tracking plus-proche-voisin : chaque track = suite de points a <2u/frame.
	type track struct{ pts []pt }
	var tracks []*track
	maxStep := 3.0
	for _, f := range frames {
		used := make([]bool, len(f.pts))
		// etendre les tracks existants
		for _, tr := range tracks {
			last := tr.pts[len(tr.pts)-1]
			if f.t-last.t > 2000 { // trou temporel : ne pas etendre
				continue
			}
			best, bi := maxStep, -1
			for i, p := range f.pts {
				if used[i] {
					continue
				}
				if d := dist2D(last, p); d < best {
					best, bi = d, i
				}
			}
			if bi >= 0 {
				tr.pts = append(tr.pts, f.pts[bi])
				used[bi] = true
			}
		}
		// nouveaux tracks pour les points non-appuyes
		for i, p := range f.pts {
			if !used[i] {
				tracks = append(tracks, &track{pts: []pt{p}})
			}
		}
	}
	// garder les tracks longs
	var long []*track
	for _, tr := range tracks {
		if len(tr.pts) >= 20 {
			long = append(long, tr)
		}
	}
	sort.Slice(long, func(i, j int) bool { return len(long[i].pts) > len(long[j].pts) })
	fmt.Printf("tracks total=%d ; tracks>=20pts=%d\n", len(tracks), len(long))

	// 3) lissage sur les tracks longs
	var steps []float64
	tele := 0
	for _, tr := range long {
		for j := 1; j < len(tr.pts); j++ {
			d := dist2D(tr.pts[j-1], tr.pts[j])
			steps = append(steps, d)
			if d > 5 {
				tele++
			}
		}
	}
	if len(steps) > 0 {
		sort.Float64s(steps)
		sum := 0.0
		for _, s := range steps {
			sum += s
		}
		fmt.Printf("LISSAGE tracks>=20 : mean=%.3f p50=%.3f max=%.3f teleports(>5)=%d/%d (oracle 0.043/0.042/2.83)\n",
			sum/float64(len(steps)), steps[len(steps)/2], steps[len(steps)-1], tele, len(steps))
	}
	for i := 0; i < len(long) && i < 12; i++ {
		tr := long[i]
		fmt.Printf("  track%d : %dpts t[%d..%d] ex(%.2f,%.2f,%.2f)\n", i, len(tr.pts),
			tr.pts[0].t, tr.pts[len(tr.pts)-1].t, tr.pts[0].x, tr.pts[0].y, tr.pts[0].z)
	}

	// 4) PNG
	if len(long) == 0 {
		fmt.Println("aucun track long -> pas de PNG")
		return
	}
	var mnx, mxx, mny, mxy float32 = 1e9, -1e9, 1e9, -1e9
	for _, tr := range long {
		for _, p := range tr.pts {
			if p.x < mnx {
				mnx = p.x
			}
			if p.x > mxx {
				mxx = p.x
			}
			if p.y < mny {
				mny = p.y
			}
			if p.y > mxy {
				mxy = p.y
			}
		}
	}
	const S, pad = 820, 40
	span := float64(S - 2*pad)
	sx, sy := float64(mxx-mnx), float64(mxy-mny)
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	px := func(p pt) (int, int) {
		return pad + int((float64(p.x)-float64(mnx))/sx*span), S - pad - int((float64(p.y)-float64(mny))/sy*span)
	}
	for i, tr := range long {
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, p := range tr.pts {
			x, y := px(p)
			if j > 0 {
				dx, dy := x-lx, y-ly
				n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
				for k := 0; k <= n; k++ {
					xx, yy := lx+dx*k/n, ly+dy*k/n
					if xx >= 0 && xx < S && yy >= 0 && yy < S {
						img.Set(xx, yy, col)
					}
				}
			}
			lx, ly = x, y
		}
	}
	f, _ := os.Create(scratch + "/trajectoires_f32le.png")
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("-> %s/trajectoires_f32le.png (%d tracks)\n", scratch, len(long))
}
