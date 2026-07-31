// tmp_traj — TRAJECTOIRES par joueur (déplacements) en image PNG, couleur par joueur
// (palette Okabe-Ito, daltonien-safe). Décodage dense par-slot du composant i0
// (object-position-dynamic-precision) via world_dump CE, par-slot biped 512-519.
//
// NB : les positions per-slot fiables exigent le world_dump CE → film 000d5950
// (Cliffhanger) ou 7344d24f (Vagabond). Le frame-decoder ne résout proprement que
// ~2-3 slots (mur §3 desync), les autres sont sparse.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_traj [filmDir] [out.png]
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

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

// bipedSlots = TOUTES les entités biped (ti=35) du world_dump (~99), pas seulement
// 512-519. Ce sont des ENTITÉS (vies/respawns + cadavres-ragdolls cumulés sur le
// match), pas 99 joueurs : ~8 joueurs actifs/tick. Rempli par loadBindings.
var bipedSlots = map[uint32]bool{}

// Okabe-Ito (apps/web/.../palettes/okabe-ito.ts) — qualitatif daltonien-safe.
var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
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

func listFrames(d []byte) []struct {
	ts  uint64
	pay []byte
} {
	var out []struct {
		ts  uint64
		pay []byte
	}
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, struct {
				ts  uint64
				pay []byte
			}{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

var cachedBindings [][2]uint32 // (slot, typeIndex) parsés une fois

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			cachedBindings = append(cachedBindings, [2]uint32{slot, ti})
			if ti == 35 { // biped : joueur, bot ou remplaçant (les slots réels sont 512-618)
				bipedSlots[slot&0x3fffffff] = true
			}
		}
	}
}

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range cachedBindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

type sample struct {
	t    int
	kind filmdec.PosKind
	vec  [3]float32
}

type track struct {
	slot uint32
	pts  []sample
}

// segmentTrajectory reconstruit la trajectoire selon le modèle RÉEL du composant i0 :
// position ABSOLUE au spawn/respawn, puis progression en DELTAS. On ancre à chaque
// absolu (vérité terrain qui corrige la dérive) et on accumule les deltas suivants.
//   - Un DELTA de pas world > maxStep = mis-décodé (largeur runtime imparfaite) → REJETÉ
//     (ne pas accumuler une dérive garbage : c'est la « ligne droite qui traverse la map »).
//   - Un ABSOLU sépare les segments UNIQUEMENT sur un vrai trou temporel (respawn/gap :
//     dt > gapThr) ; sinon il RECALE la position accumulée (correction de dérive) sans couper.
//
// Ne jamais relier deux segments (ce serait le zigzag faussement lu « bruit »).
// Retourne une liste de segments (polylignes), chacun un mouvement continu.
func segmentTrajectory(p []sample, gapThr float64, maxStep float64) [][]sample {
	var segs [][]sample
	var cur []sample
	var pos [3]float32
	have := false
	lastT := 0
	flush := func() {
		if len(cur) >= 2 {
			segs = append(segs, cur)
		}
		cur = nil
	}
	for _, s := range p {
		dt := float64(s.t-lastT) / 1000.0
		switch s.kind {
		case filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback:
			// L'absolu ANCRE un NOUVEAU segment (spawn). On n'y recale PAS en cours de
			// segment : le décode absolu est plus bruité que les deltas (le chemin propre
			// vient de l'accumulation de deltas, pas des re-sends absolus). Nouveau segment
			// sur trou temporel OU premier absolu.
			if !have || dt > gapThr {
				flush()
				pos = s.vec
				have = true
				cur = append(cur, sample{s.t, s.kind, pos})
			}
			lastT = s.t
		case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
			if !have {
				continue // pas encore d'ancre absolue
			}
			step := math.Sqrt(float64(s.vec[0]*s.vec[0] + s.vec[1]*s.vec[1] + s.vec[2]*s.vec[2]))
			if step > maxStep {
				continue // delta mis-décodé (dérive garbage) -> ignoré
			}
			pos[0] += s.vec[0]
			pos[1] += s.vec[1]
			pos[2] += s.vec[2]
			cur = append(cur, sample{s.t, s.kind, pos})
			lastT = s.t
		}
	}
	flush()
	return segs
}

// buildTrajectory choisit la méthode SELON les données du slot :
//   - assez de deltas (joueur enregistreur, ex slot519) → accumulation PURE des deltas
//     depuis le spawn (les absolus sont bruités/mal-attribués → ignorés).
//   - sinon assez d'absolus (re-sends pleins, ex slot515) → absolus dans le temps + continuité.
func buildTrajectory(p []sample) []sample {
	nD, nA := 0, 0
	for _, s := range p {
		switch s.kind {
		case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
			nD++
		case filmdec.PosKindAbsolute:
			nA++
		}
	}
	// Préférer les ABSOLUS (repère monde commun → overlay multi-joueurs cohérent)
	// quand il y en a assez ; sinon retomber sur l'accumulation de deltas (repère
	// relatif au spawn, non comparable entre joueurs).
	minAbs, minDelta := 30, 30
	if v := os.Getenv("MINABS"); v != "" {
		fmt.Sscanf(v, "%d", &minAbs)
	}
	if v := os.Getenv("MINDELTA"); v != "" {
		fmt.Sscanf(v, "%d", &minDelta)
	}
	if os.Getenv("ABS") == "1" && nA >= minAbs {
		return absTrajectory(p)
	}
	if nD >= minDelta {
		return pureDeltas(p)
	}
	if nA >= minAbs {
		return absTrajectory(p)
	}
	return nil
}

// pureDeltas : spawn (1er absolu) + accumulation des deltas seuls.
func pureDeltas(p []sample) []sample {
	start := -1
	for i, s := range p {
		if s.kind == filmdec.PosKindAbsolute {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	cur := p[start].vec
	out := []sample{{p[start].t, p[start].kind, cur}}
	for i := start + 1; i < len(p); i++ {
		if k := p[i].kind; k == filmdec.PosKindDelta8 || k == filmdec.PosKindDeltaAxis {
			cur[0] += p[i].vec[0]
			cur[1] += p[i].vec[1]
			cur[2] += p[i].vec[2]
			out = append(out, sample{p[i].t, p[i].kind, cur})
		}
	}
	return out
}

// absTrajectory : positions absolues (re-sends) triées + filtre de continuité (rejette
// les sauts = mis-attributions). Pas trop serré car les absolus sont espacés dans le temps.
func absTrajectory(p []sample) []sample {
	var a []sample
	for _, s := range p {
		if s.kind == filmdec.PosKindAbsolute {
			a = append(a, s)
		}
	}
	if len(a) == 0 {
		return nil
	}
	const thr = 400.0
	out := []sample{a[0]}
	for i := 1; i < len(a); i++ {
		if dist(out[len(out)-1].vec, a[i].vec) <= thr {
			out = append(out, a[i])
		}
	}
	return out
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func main() {
	dir, out := defFilm, "trajectoires.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	filmdec.SetRecordStateParam(2)
	loadBindings(dir)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { frameSamples = append(frameSamples, s) })

	// Rejet des faux-positifs du resync : une position qui impliquerait une VITESSE
	// impossible depuis la dernière position connue du slot (seuil scalé par le temps
	// écoulé) est refusée. maxSpeed en unités-map/s (Cliffhanger ~1153 u/axe). Ancre
	// mise à jour à chaque frame par les positions retenues (normal + resync accepté).
	lastPos := map[uint32][3]float32{}
	lastTime := map[uint32]int{}
	curMs := 0
	maxSpeed := 300.0
	if v := os.Getenv("MAXSPEED"); v != "" {
		fmt.Sscanf(v, "%f", &maxSpeed)
	}
	accept := func(slot uint32, pos [3]float32, hasPos bool) bool {
		if !hasPos {
			return true
		}
		if lt, ok := lastTime[slot]; ok {
			dt := float64(curMs-lt) / 1000.0
			if dt < 0.001 {
				dt = 0.001
			}
			if float64(dist(lastPos[slot], pos))/dt > maxSpeed {
				return false // téléport -> faux-positif
			}
		}
		return true
	}

	if os.Getenv("INFERRESYNC") != "" {
		// resync VALIDÉ : sur desync irrésoluble, scan-forward vers un biped cible dont
		// la CONTINUATION est confirmée par le marcheur de chaînes (bien moins de faux
		// positifs que le resync brut).
		filmdec.SetInferResyncTargets(bipedSlots)
	}
	if os.Getenv("INFER") == "2" {
		filmdec.SetInferChain(true) // inférence RÉCURSIVE des chaînes de transitoires
	}
	if os.Getenv("INFERREPAIR") != "" {
		filmdec.SetInferRepair(true) // harness : inférence de largeur des composants non-portés
	}
	nFrames, nDesync := 0, 0
	desyncCauses := map[string]int{}
	bySlot := map[uint32][]sample{}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(dir, reg)
			frameSamples = frameSamples[:0]
			curMs = int((fr.ts - t0Us) / 1000)
			var recs []filmdec.FrameRecord
			if os.Getenv("INFER") != "" {
				// décode-correctement : infère l'archétype des transitoires non-liés (zéro faux-positif)
				recs, _ = filmdec.DecodeFrameInfer(fr.pay, w, calCfg)
				nFrames++
				if n := len(recs); n > 0 && recs[n-1].DesyncAt != -1 {
					nDesync++
					r := recs[n-1]
					switch {
					case r.Type == 1:
						desyncCauses[fmt.Sprintf("NEW ti=%d @i%d", r.TypeIndex, r.DesyncAt)]++
					case r.Type == 3:
						if _, bound := w.ArchetypeForSlot(r.Slot); !bound {
							desyncCauses["delta non-lié: inférence échouée"]++
						} else {
							desyncCauses[fmt.Sprintf("delta ti=%d @i%d", r.TypeIndex, r.DesyncAt)]++
						}
					default:
						desyncCauses[fmt.Sprintf("type=%d", r.Type)]++
					}
				}
			} else if os.Getenv("RESYNC") != "" {
				// resync ciblé biped : récupère les bipèdes tardifs après un desync
				recs = filmdec.DecodeFrameResync(fr.pay, w, calCfg, bipedSlots, accept)
			} else {
				br := filmdec.NewBitReader(fr.pay)
				recs, _ = filmdec.DecodeFrameRecords(br, w, calCfg)
			}
			tms := curMs
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				// i0 (position) peut être N'IMPORTE OÙ dans les composants — le chercher
				// par nom (bug précédent : on ne captait que Comps[0] → on ratait les autres bipèdes).
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[c.StartBit]; ok {
						bySlot[r.Slot] = append(bySlot[r.Slot], sample{tms, s.Kind, s.Vec})
						if s.Kind == filmdec.PosKindAbsolute || s.Kind == filmdec.PosKindAbsFallback {
							lastPos[r.Slot], lastTime[r.Slot] = s.Vec, tms // ancre continuité
						}
					}
					break
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	if os.Getenv("INFER") != "" {
		imm, deep, amb, none, bud := filmdec.ChainStats()
		fmt.Printf("frames=%d desync=%d (%.1f%%) | chaînes: immédiates=%d profondes=%d ambiguës=%d aucune=%d budget=%d | réparations=%d resyncs=%d\n",
			nFrames, nDesync, 100*float64(nDesync)/float64(max(nFrames, 1)), imm, deep, amb, none, bud, filmdec.ChainRepairedCount(), filmdec.InferResyncCount())
		for name, widths := range filmdec.CompWidthObservations() {
			fmt.Printf("  largeurs %s: %v\n", name, widths)
		}
		type kv struct {
			k string
			n int
		}
		var cs []kv
		for k, n := range desyncCauses {
			cs = append(cs, kv{k, n})
		}
		sort.Slice(cs, func(i, j int) bool { return cs[i].n > cs[j].n })
		for i, c := range cs {
			if i >= 15 {
				break
			}
			fmt.Printf("  desync-cause %5d× %s\n", c.n, c.k)
		}
	}

	// Mode diagnostic : dump des positions absolues brutes par slot (pour juger si
	// elles forment un chemin cohérent ou du bruit dispersé).
	if os.Getenv("DUMPABS") != "" {
		for s := uint32(512); s <= 519; s++ {
			var a []sample
			for _, x := range bySlot[s] {
				if x.kind == filmdec.PosKindAbsolute {
					a = append(a, x)
				}
			}
			sort.Slice(a, func(i, j int) bool { return a[i].t < a[j].t })
			fmt.Printf("--- slot%d : %d absolus ---\n", s, len(a))
			for _, x := range a {
				fmt.Printf("  t=%6dms  (%.1f, %.1f, %.1f)\n", x.t, x.vec[0], x.vec[1], x.vec[2])
			}
		}
	}

	// garde les slots avec assez de points (vraie trajectoire), triés par temps.
	var tracks []track
	skip519 := os.Getenv("SKIP519") != ""
	deltaAccum := os.Getenv("DELTAACCUM") != "" // modèle spawn-absolu + accumulation deltas + segmentation respawn
	gapThr := 3.0                               // s : trou temporel séparant deux segments (respawn)
	if v := os.Getenv("GAPTHR"); v != "" {
		fmt.Sscanf(v, "%f", &gapThr)
	}
	// pas world max d'un delta plausible (au-delà = mis-décodé). Défaut LARGE : les
	// deltas légitimes du POV sont grands (échantillons espacés) ; ne filtrer que les
	// dérives évidentes. Baisser via MAXSTEP pour tester le rejet.
	maxStep := 100000.0
	if v := os.Getenv("MAXSTEP"); v != "" {
		fmt.Sscanf(v, "%f", &maxStep)
	}
	minTrack := 6
	if v := os.Getenv("MINTRACK"); v != "" {
		fmt.Sscanf(v, "%d", &minTrack)
	}
	// itère TOUS les slots biped (joueurs + bots + remplaçants), triés.
	var allBipeds []uint32
	for s := range bipedSlots {
		allBipeds = append(allBipeds, s)
	}
	sort.Slice(allBipeds, func(i, j int) bool { return allBipeds[i] < allBipeds[j] })
	for _, s := range allBipeds {
		if s == 519 && skip519 {
			continue
		}
		p := bySlot[s]
		sort.Slice(p, func(i, j int) bool { return p[i].t < p[j].t })
		nAbs, nD8, nDax := 0, 0, 0
		for _, s := range p {
			switch s.kind {
			case filmdec.PosKindAbsolute:
				nAbs++
			case filmdec.PosKindDelta8:
				nD8++
			case filmdec.PosKindDeltaAxis:
				nDax++
			}
		}
		if deltaAccum {
			// Modèle position réel : absolu au spawn, deltas ensuite, segmenter au respawn.
			segs := segmentTrajectory(p, gapThr, maxStep)
			tot := 0
			for _, seg := range segs {
				tot += len(seg)
				if len(seg) >= minTrack {
					tracks = append(tracks, track{s, seg})
				}
			}
			fmt.Printf("  slot%d : raw=%d (abs=%d d8=%d dax=%d) → %d segments, %d pts\n", s, len(p), nAbs, nD8, nDax, len(segs), tot)
			continue
		}
		// Les positions film sont RELATIVES (deltas) à la précédente, depuis le SPAWN
		// (insight user). On part du 1er absolu (= spawn) puis on ACCUMULE les deltas.
		clean := buildTrajectory(p)
		fmt.Printf("  slot%d : raw=%d (abs=%d d8=%d dax=%d) → accum=%d\n", s, len(p), nAbs, nD8, nDax, len(clean))
		if len(clean) >= minTrack {
			tracks = append(tracks, track{s, clean})
		}
	}
	fmt.Printf("film=%s : %d slots avec trajectoire (≥15 pts)\n", dir, len(tracks))
	for _, t := range tracks {
		fmt.Printf("  slot%d : %d points\n", t.slot, len(t.pts))
	}
	if len(tracks) == 0 {
		fmt.Println("aucune trajectoire exploitable.")
		return
	}

	// bornes communes (plan = 2 axes de plus grand span sur l'ensemble).
	var mn, mx [3]float32
	first := true
	for _, t := range tracks {
		for _, s := range t.pts {
			if first {
				mn, mx, first = s.vec, s.vec, false
			}
			for a := 0; a < 3; a++ {
				if s.vec[a] < mn[a] {
					mn[a] = s.vec[a]
				}
				if s.vec[a] > mx[a] {
					mx[a] = s.vec[a]
				}
			}
		}
	}
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a

	renderTraj(out, tracks, ax, ay, mn, mx)
	fmt.Printf("→ %s\n", out)
}

func renderTraj(path string, tracks []track, ax, ay int, mn, mx [3]float32) {
	const S, pad = 820, 40
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	// grille discrète de repère.
	for g := 0; g <= 10; g++ {
		c := pad + int(float64(g)/10*span)
		for k := pad; k <= S-pad; k++ {
			img.Set(c, k, color.RGBA{28, 31, 40, 255})
			img.Set(k, c, color.RGBA{28, 31, 40, 255})
		}
	}
	sx, sy := float64(mx[ax]-mn[ax]), float64(mx[ay]-mn[ay])
	px := func(v [3]float32) (int, int) {
		u := (float64(v[ax]) - float64(mn[ax])) / sx
		w := (float64(v[ay]) - float64(mn[ay])) / sy
		return pad + int(u*span), S - pad - int(w*span)
	}
	thickLine := func(x0, y0, x1, y1 int, col color.RGBA) {
		dx, dy := x1-x0, y1-y0
		n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
		for i := 0; i <= n; i++ {
			x := x0 + dx*i/n
			y := y0 + dy*i/n
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					if x+ox >= 0 && x+ox < S && y+oy >= 0 && y+oy < S {
						img.Set(x+ox, y+oy, col)
					}
				}
			}
		}
	}
	disc := func(cx, cy, r int, col color.RGBA) {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r && cx+dx >= 0 && cx+dx < S && cy+dy >= 0 && cy+dy < S {
					img.Set(cx+dx, cy+dy, col)
				}
			}
		}
	}
	for i, t := range tracks {
		// couleur par SLOT : les segments d'un même joueur (respawns) partagent la
		// couleur. Modulo palette (99 bipeds > 8 couleurs).
		ci := i
		if t.slot >= 512 {
			ci = int(t.slot - 512)
		}
		col := okabeIto[ci%len(okabeIto)]
		var lx, ly int
		for j, s := range t.pts {
			x, y := px(s.vec)
			if j > 0 {
				thickLine(lx, ly, x, y, col)
			}
			lx, ly = x, y
		}
		// marqueurs début (cercle) / fin (disque plein).
		x0, y0 := px(t.pts[0].vec)
		xn, yn := px(t.pts[len(t.pts)-1].vec)
		disc(x0, y0, 4, color.RGBA{255, 255, 255, 255})
		disc(x0, y0, 2, col)
		disc(xn, yn, 5, col)
	}
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}
