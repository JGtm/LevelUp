// tmp_offlinepng — rend un PNG XY des trajectoires offline (CSV slot,chunk,pkt,ts,x,y,z).
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_offlinepng <trajectories.csv> <out.png>
package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"
)

const (
	wpx = 1200
	hpx = 1000
)

type pt struct{ x, y float64 }

func hsv(h float64) color.RGBA {
	i := math.Floor(h * 6)
	f := h*6 - i
	q := 1 - f
	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = 1, f, 0
	case 1:
		r, g, b = q, 1, 0
	case 2:
		r, g, b = 0, 1, f
	case 3:
		r, g, b = 0, q, 1
	case 4:
		r, g, b = f, 0, 1
	case 5:
		r, g, b = 1, 0, q
	}
	return color.RGBA{uint8(r * 235), uint8(g * 235), uint8(b * 235), 255}
}

func line(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := int(math.Abs(float64(x1-x0))), -int(math.Abs(float64(y1-y0)))
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for n := 0; n < 8000; n++ {
		if x0 >= 0 && x0 < wpx && y0 >= 0 && y0 < hpx {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_offlinepng <csv> <png>")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		fmt.Println(err)
		return
	}
	traj := map[int][]pt{}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, rec := range rows[1:] {
		s, _ := strconv.Atoi(rec[0])
		x, _ := strconv.ParseFloat(rec[4], 64)
		y, _ := strconv.ParseFloat(rec[5], 64)
		traj[s] = append(traj[s], pt{x, y})
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	// marge
	spanX, spanY := maxX-minX, maxY-minY
	span := math.Max(spanX, spanY) * 1.05
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	sc := math.Min(float64(wpx), float64(hpx)) / span
	px := func(p pt) (int, int) {
		return int(float64(wpx)/2 + (p.x-cx)*sc), int(float64(hpx)/2 - (p.y-cy)*sc)
	}
	img := image.NewRGBA(image.Rect(0, 0, wpx, hpx))
	for i := range img.Pix {
		img.Pix[i] = 18
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	var slots []int
	for s := range traj {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for i, s := range slots {
		c := hsv(float64(i) / float64(len(slots)))
		pts := traj[s]
		for k := 1; k < len(pts); k++ {
			// n'affiche pas les sauts > 2u (respawn/teleport)
			if math.Hypot(pts[k].x-pts[k-1].x, pts[k].y-pts[k-1].y) > 2 {
				continue
			}
			x0, y0 := px(pts[k-1])
			x1, y1 := px(pts[k])
			line(img, x0, y0, x1, y1, c)
		}
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()
	_ = png.Encode(out, img)
	fmt.Printf("PNG %dx%d : %d slots, boite X[%.2f,%.2f] Y[%.2f,%.2f] -> %s\n",
		wpx, hpx, len(slots), minX, maxX, minY, maxY, os.Args[2])
}
