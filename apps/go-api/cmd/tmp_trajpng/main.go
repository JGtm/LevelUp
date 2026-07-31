// tmp_trajpng — PNG des trajectoires i0 gameplay reconstruites OFFLINE avec la VRAIE range CE
// (QuantRangeCEBiped), projection X-Y, superposées au nuage oracle et à la boîte oracle. PNG pur
// image/png (CGO_ENABLED=0). Sortie : scratchpad/trajectoires_realrange.png. THROWAWAY.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_trajpng [chunkLo chunkHi]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	cache   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	oracleP = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
	scratch = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	idLow   = 11
	Wpx     = 1000
	Hpx     = 700
)

var bipedSlots = []uint32{512, 513, 514, 515, 516, 517, 518, 519}
var slotColors = []color.RGBA{
	{230, 60, 60, 255}, {60, 160, 230, 255}, {80, 200, 90, 255}, {230, 170, 40, 255},
	{180, 90, 220, 255}, {40, 200, 200, 255}, {230, 110, 180, 255}, {150, 150, 60, 255},
}

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
func framePayload(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func listType0(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

type binding struct {
	full uint32
	ti   uint32
}

func offlineBindings(pay []byte) []binding {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
	}
	return out
}

func loadOracleXY() [][2]float64 {
	f, err := os.Open(oracleP)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out [][2]float64
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "eid") {
			continue
		}
		fs := strings.Split(t, ",")
		if len(fs) < 6 {
			continue
		}
		x, e1 := strconv.ParseFloat(strings.TrimSpace(fs[3]), 64)
		y, e2 := strconv.ParseFloat(strings.TrimSpace(fs[4]), 64)
		if e1 == nil && e2 == nil {
			out = append(out, [2]float64{x, y})
		}
	}
	return out
}

func offlineTraj(reg *filmdec.Registry, offBs []binding, lo, hi int) map[int][][2]float64 {
	w := filmdec.NewWorld(reg)
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	for _, b := range offBs {
		w.BindFull(b.full, b.ti)
	}
	traj := map[int][][2]float64{}
	filmdec.SetPositionAccumulator(w)
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if bipedSet[s.Slot] {
			traj[int(s.Slot)] = append(traj[int(s.Slot)], [2]float64{float64(s.Vec[0]), float64(s.Vec[1])})
		}
	})
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	for c := lo; c <= hi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listType0(data) {
			filmdec.ScanFrameTargets(pay, w, cfg, bipedSet, filmdec.HarvestNextBound)
		}
	}
	filmdec.SetPositionAccumulator(nil)
	filmdec.SetPositionCaptureHook(nil)
	return traj
}

// vue : bornes fixes = boîte oracle élargie pour montrer les débordements
var vx0, vx1, vy0, vy1 = -60.0, 80.0, -70.0, 70.0

func px(x, y float64) (int, int) {
	ix := int((x - vx0) / (vx1 - vx0) * float64(Wpx))
	iy := int((1 - (y-vy0)/(vy1-vy0)) * float64(Hpx))
	return ix, iy
}
func setpx(img *image.RGBA, ix, iy int, c color.RGBA) {
	if ix >= 0 && ix < Wpx && iy >= 0 && iy < Hpx {
		img.SetRGBA(ix, iy, c)
	}
}
func disk(img *image.RGBA, ix, iy, r int, c color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				setpx(img, ix+dx, iy+dy, c)
			}
		}
	}
}
func rect(img *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA) {
	ix0, iy0 := px(x0, y0)
	ix1, iy1 := px(x1, y1)
	for ix := ix0; ix <= ix1; ix++ {
		setpx(img, ix, iy0, c)
		setpx(img, ix, iy1, c)
	}
	for iy := iy1; iy <= iy0; iy++ {
		setpx(img, ix0, iy, c)
		setpx(img, ix1, iy, c)
	}
}

func main() {
	lo, hi := 3, 30
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay02 := framePayload(inflate(cache+"/chunk_02.bin"), 2)
	offBs := offlineBindings(pay02)
	traj := offlineTraj(reg, offBs, lo, hi)

	img := image.NewRGBA(image.Rect(0, 0, Wpx, Hpx))
	bg := color.RGBA{22, 24, 30, 255}
	for i := range img.Pix {
		_ = i
	}
	for y := 0; y < Hpx; y++ {
		for x := 0; x < Wpx; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	// nuage oracle (gris)
	for _, p := range loadOracleXY() {
		ix, iy := px(p[0], p[1])
		setpx(img, ix, iy, color.RGBA{90, 95, 105, 255})
	}
	// boîte oracle (blanc)
	rect(img, -6.33, -25.14, 35.70, 27.50, color.RGBA{240, 240, 240, 255})
	// trajectoires offline par slot
	for si, s := range bipedSlots {
		c := slotColors[si%len(slotColors)]
		pts := traj[int(s)]
		for _, p := range pts {
			ix, iy := px(p[0], p[1])
			disk(img, ix, iy, 2, c)
		}
	}
	_ = os.MkdirAll(scratch, 0o755)
	out := scratch + "/trajectoires_realrange.png"
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	tot := 0
	for _, s := range bipedSlots {
		tot += len(traj[int(s)])
	}
	fmt.Printf("PNG écrit : %s\n", out)
	fmt.Printf("points offline tracés=%d (boîte oracle blanche x[-6.33,35.70] y[-25.14,27.50] ; nuage oracle gris)\n", tot)
	fmt.Printf("range CE utilisée : %v\n", filmdec.WorldPositionRange)
}
