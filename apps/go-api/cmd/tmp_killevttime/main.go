// tmp_killevttime — THROWAWAY, OEIL NEUF (hypothèse user : records distribués par tranche de temps).
// Pour chaque kill connu (killer/victime/time, chunk_27), chercher dans les FRAMES de ce moment
// un record kill-event (grammaire FUN_14104bd08 : victim R(1)+R(5), killer R(1)+R(5), ...), et
// VALIDER par cohérence : sur les 93 kills, l'index participant du killer doit être constant par
// killer_xuid (idem victime). Le bruit se lave avec 93 ancres de vérité-terrain.
//
// Usage : tmp_killevttime [maxIdx] [winMs]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

func nm(x uint64) string {
	if g, ok := xuidName[x]; ok {
		return g
	}
	return fmt.Sprintf("%d", x)
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

func bit(d []byte, bp int) int {
	if bp>>3 >= len(d) || bp < 0 {
		return -1
	}
	return int((d[bp>>3] >> uint(7-(bp&7))) & 1)
}
func bitsN(d []byte, bp, n int) (int, int) {
	v := 0
	for i := 0; i < n; i++ {
		b := bit(d, bp)
		if b < 0 {
			return -1, -1
		}
		v = (v << 1) | b
		bp++
	}
	return v, bp
}
func readOpt5(d []byte, bp int) (int, int, bool) {
	b := bit(d, bp)
	if b < 0 {
		return -1, -1, false
	}
	bp++
	if b == 0 {
		v, nbp := bitsN(d, bp, 5)
		return v, nbp, true
	}
	return -1, bp, false
}

// decodeRecord : record kill-event à bp. Retourne (ok, newBp, victim, killer, assist).
func decodeRecord(d []byte, bp, maxIdx int) (bool, int, int, int, int) {
	v, bp, pv := readOpt5(d, bp)
	if !pv || v < 0 || v >= maxIdx {
		return false, 0, 0, 0, 0
	}
	k, bp, pk := readOpt5(d, bp)
	if !pk || k < 0 || k >= maxIdx || k == v {
		return false, 0, 0, 0, 0
	}
	_, bp = bitsN(d, bp, 32)
	if bp < 0 {
		return false, 0, 0, 0, 0
	}
	b := bit(d, bp)
	if b < 0 {
		return false, 0, 0, 0, 0
	}
	bp++
	a, bp, _ := readOpt5(d, bp)
	if bp < 0 || a >= maxIdx {
		return false, 0, 0, 0, 0
	}
	_, bp = bitsN(d, bp, 32)
	if bp < 0 {
		return false, 0, 0, 0, 0
	}
	return true, bp, v, k, a
}

type frame struct {
	tms  int
	data []byte
}

func main() {
	maxIdx := 8
	winMs := 250
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxIdx)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &winMs)
	}

	// 1) frames type-0 de tous les chunks gameplay (02-26), horodatées
	var frames []frame
	for c := 2; c <= 26; c++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz < 0 || off+16+sz > len(d) {
				break
			}
			if typ == 0 && ts >= t0Us {
				frames = append(frames, frame{int((ts - t0Us) / 1000), d[off+16 : off+16+sz]})
			}
			off += 16 + sz
		}
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].tms < frames[j].tms })
	fmt.Printf("=== %d frames type-0 (chunks 02-26), span %.0f..%.0f s ===\n",
		len(frames), float64(frames[0].tms)/1000, float64(frames[len(frames)-1].tms)/1000)

	// 2) kills chunk_27
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		if e.EventType == analysis.EventTypeKill {
			kills = append(kills, ev{e.XUID, e.TimeMS})
		} else if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type kf struct {
		killer, victim uint64
		t              int
	}
	var feed []kf
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.x == k.x {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{k.x, deaths[best].x, k.t})
		}
	}
	fmt.Printf("=== %d kills appariés ; fenêtre ±%dms, maxIdx=%d ===\n\n", len(feed), winMs, maxIdx)

	// 3) pour chaque kill, scanner les frames de la fenêtre ; agréger les votes idx->xuid.
	// On teste les 2 orientations (record = victim,killer ou killer,victim) en votant les deux.
	voteKiller := map[uint64]map[int]int{} // killer_xuid -> idx -> votes (idx en position killer du record)
	voteVictim := map[uint64]map[int]int{}
	candPerKill := 0
	for _, f := range feed {
		seen := map[[3]int]bool{}
		for fi := range frames {
			fr := &frames[fi]
			if fr.tms < f.t-winMs || fr.tms > f.t+winMs {
				continue
			}
			total := len(fr.data) * 8
			for bp := 0; bp+78 <= total; bp++ {
				ok, _, v, k, a := decodeRecord(fr.data, bp, maxIdx)
				if !ok {
					continue
				}
				key := [3]int{v, k, a}
				if seen[key] {
					continue
				}
				seen[key] = true
				candPerKill++
				// record = (victim=v, killer=k)
				if voteKiller[f.killer] == nil {
					voteKiller[f.killer] = map[int]int{}
				}
				if voteVictim[f.victim] == nil {
					voteVictim[f.victim] = map[int]int{}
				}
				voteKiller[f.killer][k]++
				voteVictim[f.victim][v]++
			}
		}
	}
	fmt.Printf("candidats distincts cumulés sur tous les kills : %d (moy %.1f/kill)\n\n", candPerKill, float64(candPerKill)/float64(len(feed)))

	// 4) pour chaque joueur, l'index dominant (position killer) + pureté
	fmt.Println("=== idx dominant en position KILLER par killer_xuid (record=victim,killer) ===")
	dumpVotes(voteKiller)
	fmt.Println("\n=== idx dominant en position VICTIME par victim_xuid ===")
	dumpVotes(voteVictim)

	fmt.Println("\n>>> Si chaque joueur a un idx dominant NET et DISTINCT (injectif), et que killer-map==victim-map,")
	fmt.Println(">>> alors le record kill-event est localisé et l'index participant identifié. Sinon = bruit (hypothèse à revoir).")
}

func dumpVotes(m map[uint64]map[int]int) {
	var xs []uint64
	for x := range m {
		xs = append(xs, x)
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	for _, x := range xs {
		var pairs []struct{ idx, c int }
		tot, top := 0, 0
		for idx, c := range m[x] {
			pairs = append(pairs, struct{ idx, c int }{idx, c})
			tot += c
			if c > top {
				top = c
			}
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].c > pairs[j].c })
		s := ""
		for i, p := range pairs {
			if i >= 5 {
				break
			}
			s += fmt.Sprintf(" idx%d:%d", p.idx, p.c)
		}
		pur := 0.0
		if tot > 0 {
			pur = 100 * float64(top) / float64(tot)
		}
		fmt.Printf("  %-16s (n=%-4d, pur=%2.0f%%) =>%s\n", nm(x), tot, pur, s)
	}
}

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
