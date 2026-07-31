package main

// wf_b_jgtm355 — la voie FIABLE : la timeline de tirs est pi-résolue (pi=2=JGtm,
// vérifié). On regarde ce que JGtm tire autour de 355s, et si un BR75 apparaît
// pour lui (le Frag Parfait est un kill NON capté par les fires, donc on cherche
// le tir BR75 le plus tardif de JGtm + le contexte temporel).

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func canonName(n string) string {
	if c, ok := analysis.WeaponFusionMap[n]; ok {
		return c
	}
	return n
}

type shot struct {
	pi     int
	name   string
	timeMS float64
}

func main() {
	const jgtmPI = 2 // résolu et vérifié (xuid 2533274823110022 -> pi=2)

	var shots []shot
	for i := 0; i <= 27; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		if len(d) == 0 {
			continue
		}
		_ = binary.LittleEndian
		est := weaponv3.USEstimator(d, (i-2)*20000)
		fires := weaponv3.ScanFireEventsV3(d, est, weaponv3.FirePi4High, true)
		for _, f := range fires {
			shots = append(shots, shot{f.PlayerIndex, canonName(f.WeaponName), f.TimestampMS})
		}
	}
	sort.Slice(shots, func(i, j int) bool { return shots[i].timeMS < shots[j].timeMS })

	// 1. derniers tirs de JGtm (pi=2), toutes armes.
	fmt.Println("=== derniers 30 tirs de JGtm (pi=2) ===")
	var jg []shot
	for _, s := range shots {
		if s.pi == jgtmPI {
			jg = append(jg, s)
		}
	}
	start := 0
	if len(jg) > 30 {
		start = len(jg) - 30
	}
	for _, s := range jg[start:] {
		fmt.Printf("  t=%7.1fs  %s\n", s.timeMS/1000.0, s.name)
	}

	// 2. dernier tir BR75 de JGtm + dernier tir tout court.
	lastBR := -1.0
	lastAny := -1.0
	brWeaponsByJ := map[string]int{}
	for _, s := range jg {
		brWeaponsByJ[s.name]++
		if s.timeMS > lastAny {
			lastAny = s.timeMS
		}
		if s.name == "BR75" && s.timeMS > lastBR {
			lastBR = s.timeMS
		}
	}
	fmt.Printf("\n=== JGtm : dernier tir %v à %.1fs ; dernier BR75 à %.1fs ===\n", "", lastAny/1000.0, lastBR/1000.0)
	fmt.Printf("  armes tirées par JGtm (count) : %v\n", brWeaponsByJ)

	// 3. fenêtre 340-370s : qui tire quoi (pour situer 355s).
	fmt.Println("\n=== tous les tirs 340s..370s (autour du Frag Parfait 355s) ===")
	for _, s := range shots {
		if s.timeMS >= 340000 && s.timeMS <= 370000 {
			tag := ""
			if s.pi == jgtmPI {
				tag = "  <== JGtm"
			}
			fmt.Printf("  t=%7.1fs  pi=%d  %s%s\n", s.timeMS/1000.0, s.pi, s.name, tag)
		}
	}

	// 4. JGtm tirs BR75 sur toute la partie (timestamps).
	fmt.Println("\n=== tous les tirs BR75 de JGtm (pi=2) ===")
	any := false
	for _, s := range jg {
		if s.name == "BR75" {
			fmt.Printf("  t=%7.1fs\n", s.timeMS/1000.0)
			any = true
		}
	}
	if !any {
		fmt.Println("  AUCUN tir BR75 capté pour JGtm")
	}
}
