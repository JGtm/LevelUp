package main

// mapmode.go — « tmp_forgedraw map <mvar> <sortie.png> [type_id ...] »
//
// Dessine TOUS les objets places d'une variante de carte vue de dessus, et met en
// evidence les type_id demandes. Sert a faire reconnaitre un emplacement a l'oeil :
// « ces trois marqueurs sont-ils le surbouclier, ou les socles d'armes ? » — une
// question que seule une personne qui connait la carte peut trancher.

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func cmdMap(mvarPath, outPath string, wanted []string) {
	buf, err := os.ReadFile(mvarPath)
	must(err)
	root, err := mapvar.DecodeRoot(buf)
	must(err)
	v, err := mapvar.Parse(buf)
	must(err)
	objs, _ := root.Field(3)

	hi := map[int32]bool{}
	for _, w := range wanted {
		if n, err := strconv.ParseInt(w, 10, 64); err == nil {
			hi[int32(n)] = true
		}
	}
	roleByIdx := map[int]string{}
	for _, ob := range v.Objectives() {
		roleByIdx[ob.ObjectIdx] = string(ob.Role)
	}

	b := rect{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, o := range v.Objects {
		b.minX, b.minY = math.Min(b.minX, o.Pos.X), math.Min(b.minY, o.Pos.Y)
		b.maxX, b.maxY = math.Max(b.maxX, o.Pos.X), math.Max(b.maxY, o.Pos.Y)
	}
	b.minX, b.minY, b.maxX, b.maxY = b.minX-4, b.minY-4, b.maxX+4, b.maxY+4
	c := newCanvas(1400, 1400, b)

	// Tous les objets : un point gris, taille proportionnelle a l'emprise du sac
	// de forme quand il y en a une (sinon un point fixe).
	for i, o := range v.Objects {
		x, y := c.px(o.Pos.X, o.Pos.Y)
		col := color.RGBA{150, 158, 170, 255}
		r := 1
		if roleByIdx[i] != "" {
			col, r = color.RGBA{90, 170, 255, 255}, 3
		}
		if hi[o.TypeID] {
			continue
		}
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				c.set(x+dx, y+dy, col)
			}
		}
	}
	// Les zones d'objectif, en empreinte orientee (lecture « tailles pleines »).
	for _, ob := range v.Objectives() {
		kind, slots, ok := shapeOf(objs.Items[ob.ObjectIdx])
		if !ok {
			continue
		}
		o := v.Objects[ob.ObjectIdx]
		z := zone{kind: kind,
			a: float64(slots[5]) / fixedPointUnit / 2, b: float64(slots[6]) / fixedPointUnit / 2,
			pos: o.Pos, fwd: o.Forward}
		if kind == 2 {
			z.a = float64(slots[5]) / fixedPointUnit
			c.circle(z.pos.X, z.pos.Y, z.a, color.RGBA{90, 170, 255, 255})
			continue
		}
		c.poly(z.footprintOBB(), color.RGBA{90, 170, 255, 255}, false)
	}
	// Les types demandes : croix rouge de 9 px, bien visible.
	n := 0
	for _, o := range v.Objects {
		if !hi[o.TypeID] {
			continue
		}
		n++
		x, y := c.px(o.Pos.X, o.Pos.Y)
		for d := -9; d <= 9; d++ {
			c.set(x+d, y, color.RGBA{255, 70, 60, 255})
			c.set(x, y+d, color.RGBA{255, 70, 60, 255})
			c.set(x+d, y+1, color.RGBA{255, 70, 60, 255})
			c.set(x+1, y+d, color.RGBA{255, 70, 60, 255})
		}
		fmt.Printf("  marque : type=%d pos=(%.2f, %.2f, %.2f)\n", o.TypeID, o.Pos.X, o.Pos.Y, o.Pos.Z)
	}
	fmt.Printf("%s : %d objets, %d marques, %d objectifs\n",
		mvarPath, len(v.Objects), n, len(v.Objectives()))
	writePNG(outPath, c)
}
