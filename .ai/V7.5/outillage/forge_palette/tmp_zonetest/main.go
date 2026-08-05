// tmp_zonetest — jetable. LE TEMOIN de Q2 : un joueur qui capture une zone doit etre DEDANS.
//
// Confronte l'artefact de rejeu (positions joueur datees) aux zones lues dans le .mvar,
// a un instant donne par le RELEVE TERRAIN. Teste les deux lectures des dimensions :
//   - « demi-extents » : s5/s6 sont des distances au centre ;
//   - « tailles pleines » : s5/s6 sont des largeurs, donc demi = s5/2, s6/2.
//
// Le cylindre (famille 2) prend s5 comme RAYON dans les deux lectures — un rayon est deja
// une demi-mesure. Un temoin negatif (les memes zones translatees) borne le hasard.
//
//	tmp_zonetest <artefact.json> <carte.mvar> <t_secondes> [t2 t3 ...]
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

const fixedPointUnit = 65536.0

type point struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type track struct {
	Slot   int     `json:"slot"`
	Team   int     `json:"team"`
	XUID   string  `json:"xuid"`
	Points []point `json:"points"`
}

type artifact struct {
	MatchID    string  `json:"matchId"`
	FrameCount int     `json:"frameCount"`
	IntervalMS int     `json:"intervalMs"`
	Tracks     []track `json:"tracks"`
}

type zone struct {
	role     string
	kind     int32
	a, b     float64 // s5, s6 BRUTS convertis en metres, sans interpretation
	top, bot float64
	pos, fwd mapvar.Vec3
	team     int
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: tmp_zonetest <artefact.json> <carte.mvar> <t_secondes...>")
		os.Exit(2)
	}
	art := loadArtifact(os.Args[1])
	zones := loadZones(os.Args[2])
	fmt.Printf("artefact %s : %d traces, %d images, pas %d ms\n",
		art.MatchID, len(art.Tracks), art.FrameCount, art.IntervalMS)
	fmt.Printf("zones lues : %d\n", len(zones))
	for i, z := range zones {
		fmt.Printf("  zone %d  role=%-18s equipe=%-3d centre=(%.2f, %.2f, %.2f)  s5=%.2f s6=%.2f haut=%.2f bas=%.2f  famille=%d\n",
			i, z.role, z.team, z.pos.X, z.pos.Y, z.pos.Z, z.a, z.b, z.top, z.bot, z.kind)
	}
	for _, arg := range os.Args[3:] {
		sec, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			continue
		}
		testAt(art, zones, sec)
	}
}

// testAt rend, pour chaque joueur present a l'instant demande, la zone qui le contient
// sous chacune des deux lectures.
func testAt(art artifact, zones []zone, sec float64) {
	frame := int(math.Round(sec * 1000 / float64(art.IntervalMS)))
	fmt.Printf("\n=== t = %.0f s (image %d) ===\n", sec, frame)
	fmt.Printf("%-6s %-4s %-22s %-26s %-26s\n", "slot", "eq", "position", "lecture DEMI-EXTENTS", "lecture TAILLES PLEINES")
	nHalf, nFull, present := 0, 0, 0
	for _, tr := range art.Tracks {
		p, ok := posAt(tr, frame)
		if !ok {
			continue
		}
		present++
		h := which(zones, p, true)
		f := which(zones, p, false)
		if h >= 0 {
			nHalf++
		}
		if f >= 0 {
			nFull++
		}
		if h < 0 && f < 0 {
			continue
		}
		fmt.Printf("%-6d %-4d (%7.2f,%7.2f,%6.2f) %-26s %-26s\n",
			tr.Slot, tr.Team, p.X, p.Y, p.Z, label(zones, h), label(zones, f))
	}
	fmt.Printf("joueurs presents a cet instant : %d\n", present)
	fmt.Printf("  dans une zone, lecture DEMI-EXTENTS   : %d\n", nHalf)
	fmt.Printf("  dans une zone, lecture TAILLES PLEINES : %d\n", nFull)
	// Temoin negatif : les memes formes deplacees de 12 m en x et en y.
	moved := make([]zone, len(zones))
	copy(moved, zones)
	for i := range moved {
		moved[i].pos.X += 12
		moved[i].pos.Y += 12
	}
	cH, cF := 0, 0
	for _, tr := range art.Tracks {
		p, ok := posAt(tr, frame)
		if !ok {
			continue
		}
		if which(moved, p, true) >= 0 {
			cH++
		}
		if which(moved, p, false) >= 0 {
			cF++
		}
	}
	fmt.Printf("  TEMOIN NEGATIF (zones deplacees de 12 m) : demi=%d  plein=%d\n", cH, cF)
}

func label(zones []zone, i int) string {
	if i < 0 {
		return "-"
	}
	return fmt.Sprintf("zone %d (%s, eq %d)", i, zones[i].role, zones[i].team)
}

// which rend l'indice de la premiere zone contenant le point, ou -1.
// Le test est 3D : la hauteur compte (haut au-dessus du centre, bas au-dessous).
func which(zones []zone, p point, half bool) int {
	for i, z := range zones {
		if z.contains(p, half) {
			return i
		}
	}
	return -1
}

func (z zone) contains(p point, half bool) bool {
	dz := p.Z - z.pos.Z
	if dz > z.top || dz < -z.bot {
		return false
	}
	ha, hb := z.a, z.b
	if !half {
		ha, hb = z.a/2, z.b/2
	}
	dx, dy := p.X-z.pos.X, p.Y-z.pos.Y
	fx, fy := z.fwd.X, z.fwd.Y
	n := math.Hypot(fx, fy)
	if n < 1e-6 {
		fx, fy, n = 1, 0, 1
	}
	fx, fy = fx/n, fy/n
	u := dx*fx + dy*fy
	w := -dx*fy + dy*fx
	if z.kind == 2 {
		// Cylindre : le rayon est deja une demi-mesure, les deux lectures coincident.
		return math.Hypot(u, w) <= z.a
	}
	return math.Abs(u) <= ha && math.Abs(w) <= hb
}

func posAt(tr track, frame int) (point, bool) {
	i := sort.Search(len(tr.Points), func(i int) bool { return tr.Points[i].T >= frame })
	if i >= len(tr.Points) {
		return point{}, false
	}
	if tr.Points[i].T > frame && i > 0 {
		i--
	}
	if abs(tr.Points[i].T-frame) > 20 { // plus de 2 s d'ecart : le joueur n'est pas suivi ici
		return point{}, false
	}
	return tr.Points[i], true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func loadArtifact(path string) artifact {
	b, err := os.ReadFile(path)
	must(err)
	var a artifact
	must(json.Unmarshal(b, &a))
	if a.IntervalMS == 0 {
		a.IntervalMS = 100
	}
	return a
}

func loadZones(path string) []zone {
	buf, err := os.ReadFile(path)
	must(err)
	root, err := mapvar.DecodeRoot(buf)
	must(err)
	v, err := mapvar.Parse(buf)
	must(err)
	objs, _ := root.Field(3)
	var out []zone
	for _, ob := range v.Objectives() {
		kind, s, ok := shapeOf(objs.Items[ob.ObjectIdx])
		if !ok {
			continue
		}
		o := v.Objects[ob.ObjectIdx]
		out = append(out, zone{
			role: string(ob.Role), kind: kind,
			a:   float64(s[5]) / fixedPointUnit,
			b:   float64(s[6]) / fixedPointUnit,
			top: float64(s[7]) / fixedPointUnit,
			bot: float64(s[8]) / fixedPointUnit,
			pos: o.Pos, fwd: o.Forward, team: o.TeamIndex,
		})
	}
	return out
}

func shapeOf(raw mapvar.Value) (int32, [9]int64, bool) {
	var slots [9]int64
	bag, ok := raw.Field(8)
	if !ok {
		return 0, slots, false
	}
	lst, ok := bag.Field(0)
	if !ok || len(lst.Items) == 0 {
		return 0, slots, false
	}
	inner, ok := lst.Items[0].Field(0)
	if !ok || len(inner.Items) == 0 {
		return 0, slots, false
	}
	sh := inner.Items[0]
	kind, ok := sh.Field(0)
	if !ok {
		return 0, slots, false
	}
	for i := uint16(1); i <= 8; i++ {
		if f, ok := sh.Field(i); ok {
			if v, ok2 := f.Field(0); ok2 {
				slots[i] = v.Int
			}
		}
	}
	return int32(kind.Int), slots, true
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
