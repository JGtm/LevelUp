// tmp_killevt — décode les kill-events + dégâts 0xd2 capturés live (killevents_sample.txt,
// même buffer/flux = même horloge) et CORRÈLE kill↔dégât pour l'arme par kill.
//
//   - kill-event (FUN_14104bd08, grammaire §174) : tueur/victime/assistant = R1+optR5 ×3
//     (présence = R(1)==0 → R(5)), puis R32.
//   - dégât 0xd2 (FUN_14080c1f8, §169) : en-tête 36 bits → R5 attaquant → famille (~bit 41)
//   - suffixe 0x42c9679f.
//   - corrélation same-clock : pour chaque kill, le dégât le plus proche (ordre de capture =
//     chronologique) dont l'attaquant = le tueur → l'arme (famille).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_killevt <killevents_sample.txt>
package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

type event struct {
	tag                    string
	off                    int
	data                   []byte
	killer, victim, assist int
	s1                     uint64
	attacker               int
	family                 uint64
	hasSuffix              bool
}

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var events []event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fl := strings.Fields(sc.Text())
		if len(fl) < 4 {
			continue
		}
		off, _ := strconv.Atoi(fl[2])
		data, e := hex.DecodeString(fl[3])
		if e != nil {
			continue
		}
		ev := event{tag: fl[0], off: off, data: data, killer: -1, victim: -1, assist: -1, attacker: -1}
		br := filmdec.NewBitReader(data)
		br.Skip(off)
		if ev.tag == "KILL" {
			readOpt := func() int {
				if !br.ReadBit() {
					return int(br.ReadBits(5))
				}
				return -1
			}
			ev.killer, ev.victim, ev.assist = readOpt(), readOpt(), readOpt()
			ev.s1 = br.ReadBits(32)
		} else { // DMG
			br.Skip(36)
			ev.attacker = int(br.ReadBits(5))
			ev.family = br.ReadBits(8)
			ev.hasSuffix = strings.Contains(fl[3], "42c9679f")
		}
		events = append(events, ev)
	}

	// corrélation : pour chaque kill, le dégât le plus proche (index) dont l'attaquant matche
	// le tueur. On teste 3 conventions de numérotation (==, R5>>1 = slot, tueur*2).
	match := func(atk, killer int) bool {
		return atk == killer || atk>>1 == killer || atk == killer*2
	}
	nOK := 0
	nKill := 0
	for i, e := range events {
		if e.tag != "KILL" {
			continue
		}
		nKill++
		best, bestDist := -1, 1<<30
		for j, d := range events {
			if d.tag != "DMG" || !match(d.attacker, e.killer) {
				continue
			}
			dist := i - j
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist, best = dist, j
			}
		}
		if best >= 0 {
			nOK++
			fmt.Printf("KILL tueur=%d victime=%d s1=%08X -> DMG#%d attaquant=%d famille=%02X suffixe=%v (dist=%d)\n",
				e.killer, e.victim, e.s1, best, events[best].attacker, events[best].family, events[best].hasSuffix, bestDist)
		} else {
			fmt.Printf("KILL tueur=%d victime=%d s1=%08X -> AUCUN degat attaquant=%d\n", e.killer, e.victim, e.s1, e.killer)
		}
	}
	// distribution des attaquants de dégât (pour voir la numérotation)
	atkCount := map[int]int{}
	for _, e := range events {
		if e.tag == "DMG" {
			atkCount[e.attacker]++
		}
	}
	fmt.Printf("\n=== %d kills, %d avec dégât attaquant matché ===\n", nKill, nOK)
	fmt.Print("distribution attaquants DMG (R5): ")
	for a := 0; a < 32; a++ {
		if atkCount[a] > 0 {
			fmt.Printf("%d=%d ", a, atkCount[a])
		}
	}
	fmt.Println()
}
