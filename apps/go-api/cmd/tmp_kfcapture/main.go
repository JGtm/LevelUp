// tmp_kfcapture — croise la CAPTURE LIVE des frontières de records (kf_capture.txt :
// "type id bitpos dataptr" par ligne, hook CE non-bloquant sur FUN_1406cbaa0) avec le
// BUFFER keyframe dumpé (kf_slot0_live.bin). Pour chaque record NEW du keyframe (prefix à
// bit-pos croissant), lit R(6) ti au bit-pos et mesure la largeur (bp[i+1]-bp[i]).
//
// But : distribution des archétypes + repérage des bipeds (ti=35, joueurs) + leur largeur
// (= le default-state qui porte la position). C'est la vérité terrain pour dérouler le keyframe.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

type rec struct {
	slot, ti, bp, width int
}

func main() {
	bufPath := os.Args[1]
	capPath := os.Args[2]
	buf, err := os.ReadFile(bufPath)
	if err != nil {
		panic(err)
	}
	f, err := os.Open(capPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var recs []rec
	prevBP := -1
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 || fields[0] != "1" { // NEW seulement
			continue
		}
		id, _ := strconv.ParseUint(fields[1], 16, 64)
		bp, _ := strconv.Atoi(fields[2])
		// keyframe = préfixe à bit-pos STRICTEMENT croissant (ensuite = spawns playback,
		// buffer réécrit, bit-pos qui reset -> on s'arrête).
		if bp <= prevBP {
			break
		}
		prevBP = bp
		br := filmdec.NewBitReader(buf)
		br.SetBitPos(bp)
		ti := int(br.ReadBits(6))
		recs = append(recs, rec{slot: int(id) & 0x3fffffff, ti: ti, bp: bp})
	}
	// largeurs
	for i := 0; i+1 < len(recs); i++ {
		recs[i].width = recs[i+1].bp - recs[i].bp
	}
	fmt.Printf("=== KEYFRAME : %d records NEW à bit-pos croissant (jusqu'à bit %d) ===\n",
		len(recs), func() int {
			if len(recs) > 0 {
				return recs[len(recs)-1].bp
			}
			return 0
		}())

	// distribution des ti
	tiCount := map[int]int{}
	tiWidth := map[int][]int{}
	for _, r := range recs {
		tiCount[r.ti]++
		if r.width > 0 {
			tiWidth[r.ti] = append(tiWidth[r.ti], r.width)
		}
	}
	var tis []int
	for ti := range tiCount {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	fmt.Println("--- distribution des archétypes (ti : count : largeurs de body observées) ---")
	for _, ti := range tis {
		ws := tiWidth[ti]
		wmin, wmax := 0, 0
		if len(ws) > 0 {
			wmin, wmax = ws[0], ws[0]
			for _, w := range ws {
				if w < wmin {
					wmin = w
				}
				if w > wmax {
					wmax = w
				}
			}
		}
		flag := ""
		if ti == 35 {
			flag = "  <== BIPED (joueur)"
		}
		fmt.Printf("  ti=%-3d count=%-4d width=[%d..%d]%s\n", ti, tiCount[ti], wmin, wmax, flag)
	}

	// détail des bipeds (ti=35)
	fmt.Println("--- records BIPED (ti=35) : slot, bit-pos, largeur ---")
	nb := 0
	for _, r := range recs {
		if r.ti == 35 {
			fmt.Printf("  slot=%-6d bp=%-6d width=%d\n", r.slot, r.bp, r.width)
			nb++
		}
	}
	fmt.Printf("=> %d bipeds dans le keyframe\n", nb)

	// premiers records (aperçu)
	fmt.Println("--- 24 premiers records ---")
	for i := 0; i < len(recs) && i < 24; i++ {
		fmt.Printf("  #%-3d slot=%-6d ti=%-3d bp=%-6d w=%d\n", i, recs[i].slot, recs[i].ti, recs[i].bp, recs[i].width)
	}
}
