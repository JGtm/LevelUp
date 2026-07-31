// tmp_fireprobe2 — THROWAWAY : chercher (a) un player_index couvrant LES 8 joueurs autour des armes,
// (b) le VRAI event de tir (arme 32b famille + shot/player, marqueur propre), distinct des records
// d'entité-arme (spawn/pickup/refill = weapon-id 64b). Vérité-terrain = 519 records de dégât (famille+temps).
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
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const maxMatchMs = 600000
const variantSuffix = uint32(0x42c9679f)
const deserStartBit = 36

var h32 = map[uint32]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
	tms     int
}

func type0() []pkt {
	var out []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			if typ == 0 && ts >= t0Us {
				tms := int((ts - t0Us) / 1000)
				if tms >= 0 && tms <= maxMatchMs {
					out = append(out, pkt{typ, ts, d[off+16 : off+16+size], tms})
				}
			}
			off += 16 + size
		}
	}
	return out
}

// 519 records de dégât : famille + tms (vérité-terrain de tir d'arme à feu).
type dmg struct {
	tms int
	fam string
}

func damageRecs(pkts []pkt) []dmg {
	var out []dmg
	for _, p := range pkts {
		if len(p.payload) == 0 || p.payload[0] != 0xd2 {
			continue
		}
		br := filmdec.NewBitReader(p.payload)
		br.Skip(deserStartBit)
		if !br.ReadBit() {
			br.ReadBits(2)
		}
		br.ReadBits(5)
		gid := uint32(br.ReadBits(32))
		low := uint32(br.ReadBits(32))
		if low != variantSuffix {
			continue
		}
		if fam, ok := h32[gid]; ok {
			out = append(out, dmg{p.tms, fam})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tms < out[j].tms })
	return out
}

type occ struct {
	pl  []byte
	bp  int
	fam string
	low uint32
	tms int
}

func main() {
	build()
	pkts := type0()
	dmgs := damageRecs(pkts)

	// Toutes les occurrences weapon-id (high32∈cat) en type-0.
	var occs []occ
	for _, p := range pkts {
		pl := p.payload
		total := len(pl) * 8
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(pl, bp, 32))
			fam, ok := h32[hi]
			if !ok {
				continue
			}
			occs = append(occs, occ{pl, bp, fam, uint32(bitsAt(pl, bp+32, 32)), p.tms})
		}
	}
	fmt.Printf("=== %d occurrences weapon-id (type-0) ; %d records de dégât ===\n", len(occs), len(dmgs))

	// ---- (A) RECHERCHE EXHAUSTIVE d'un player_index couvrant LES 8 joueurs ----
	// Pour chaque (offset o ∈ [-64,+160], largeur w ∈ {3,4,5}), distribution des valeurs sur
	// toutes les occurrences ; on cherche un champ dont {0..7} TOUS présents (incl. 0 et 1) et
	// max<=7 (ou couverture 0-7 dominante). Score = #valeurs 0-7 présentes - pénalité valeurs >7.
	fmt.Println("\n=== (A) champs candidats player_index w=5, 0-7 tous présents, quasi-uniformes ===")
	searchField(occs, len(occs))

	// ---- (B) Le vrai event de tir = high32 SANS suffixe 0x42c9679f ? ----
	// Sépare les occurrences : suffixe (entité, id64 complet) vs NON-suffixe (candidat fire/shot).
	// Pour les NON-suffixe, le "low32" = données de tir (shot counter, etc.). Examine sa structure.
	fmt.Println("\n=== (B) occurrences SANS suffixe 0x42c9679f (candidat fire/shot) ===")
	var nosuf []occ
	for _, oc := range occs {
		if oc.low != variantSuffix {
			nosuf = append(nosuf, oc)
		}
	}
	fmt.Printf("  %d occurrences non-suffixe\n", len(nosuf))
	// distribution du 1er octet du low32 (= juste après high32) pour voir un pattern shot/marqueur
	hiByte := map[byte]int{}
	for _, oc := range nosuf {
		hiByte[byte(oc.low>>24)]++
	}
	type bc struct {
		b byte
		c int
	}
	var bcs []bc
	for b, c := range hiByte {
		bcs = append(bcs, bc{b, c})
	}
	sort.Slice(bcs, func(i, j int) bool { return bcs[i].c > bcs[j].c })
	fmt.Printf("  octet haut du low32 (top 10) : ")
	for i := 0; i < len(bcs) && i < 10; i++ {
		fmt.Printf("0x%02x:%d ", bcs[i].b, bcs[i].c)
	}
	fmt.Println()

	// ---- (C) VÉRITÉ-TERRAIN : occurrences à ±150ms d'un record de dégât MÊME FAMILLE ----
	// Ce sont des occurrences "fire-adjacentes" prouvées (l'arme a fait des dégâts à ce moment).
	// On examine LEUR structure (suffixe ? player field couvrant 0-7 ?) pour isoler le vrai fire event.
	fmt.Println("\n=== (C) occurrences confirmées fire-adjacentes (±150ms d'un record dégât même famille) ===")
	var conf []occ
	for _, oc := range occs {
		for _, dr := range dmgs {
			if dr.fam == oc.fam && abs(dr.tms-oc.tms) <= 150 {
				conf = append(conf, oc)
				break
			}
		}
	}
	sufConf, nosufConf := 0, 0
	for _, oc := range conf {
		if oc.low == variantSuffix {
			sufConf++
		} else {
			nosufConf++
		}
	}
	fmt.Printf("  %d occurrences fire-adjacentes (suffixe=%d, non-suffixe=%d)\n", len(conf), sufConf, nosufConf)
	// sur ces confirmées, re-cherche un player field couvrant 0-7
	fmt.Println("  -- champs w=5 0-7 quasi-uniformes sur les occurrences CONFIRMÉES --")
	searchField(conf, len(conf))

	// (D) VALIDATION pidx@bp-4 vs bp-9 contre le kill feed
	fmt.Println("\n=== (D) validation pidx (bp-4 vs bp-9) contre kill feed ===")
	validatePidx(occs, dmgs)
}

var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}

type kfRow struct {
	killerPi int
	t        int
}

func killFeed() []kfRow {
	raw, _ := os.ReadFile(cache + "/chunk_27.bin")
	events, _ := analysis.ParseHighlightEvents(raw, 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	var feed []kfRow
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.x == k.x {
				continue
			}
			if dt := abs(k.t - d.t); dt < bd {
				bd, best = dt, i
			}
		}
		if best < 0 {
			continue
		}
		usedD[best] = true
		kp, ok := xuidToPi[k.x]
		if !ok {
			kp = -1
		}
		feed = append(feed, kfRow{kp, k.t})
	}
	return feed
}

func validatePidx(occs []occ, dmgs []dmg) {
	feed := killFeed()
	for _, off := range []int{-4, -9} {
		for _, W := range []int{500, 1500} {
			var nEvt, nCorrob int
			for _, k := range feed {
				if k.killerPi < 0 {
					continue
				}
				bi, bd := -1, 1<<30
				for i := range occs {
					if int(bitsAt(occs[i].pl, occs[i].bp+off, 5)) != k.killerPi {
						continue
					}
					if d := abs(occs[i].tms - k.t); d < bd {
						bd, bi = d, i
					}
				}
				if bi < 0 || bd > W {
					continue
				}
				nEvt++
				for _, dr := range dmgs {
					if dr.fam == occs[bi].fam && dr.tms >= k.t-1500 && dr.tms <= k.t+200 {
						nCorrob++
						break
					}
				}
			}
			corr := 0.0
			if nEvt > 0 {
				corr = 100 * float64(nCorrob) / float64(nEvt)
			}
			fmt.Printf("  pidx@bp%+d ±%4dms : %d/%d kills ont un fire-event tueur ; corroborés=%d (%.0f%%)\n",
				off, W, nEvt, len(feed), nCorrob, corr)
		}
	}
}

// searchField : cherche un champ w=5 dont les valeurs 0-7 sont TOUTES présentes, quasi-uniformes
// (min/max >= 0.15), et >7 faible (<8%). Imprime la distribution des meilleurs candidats.
func searchField(set []occ, total int) {
	type cand struct {
		o     int
		dist  [8]int
		over  int
		ratio float64
	}
	var best []cand
	for o := -80; o <= 200; o++ {
		if o >= 0 && o < 32 {
			continue
		}
		cnt := map[int]int{}
		for _, oc := range set {
			cnt[int(bitsAt(oc.pl, oc.bp+o, 5))]++
		}
		var c cand
		c.o = o
		ok := true
		mn, mx := 1<<30, 0
		for v := 0; v <= 7; v++ {
			c.dist[v] = cnt[v]
			if cnt[v] == 0 {
				ok = false
			}
			if cnt[v] < mn {
				mn = cnt[v]
			}
			if cnt[v] > mx {
				mx = cnt[v]
			}
		}
		for v := 8; v < 32; v++ {
			c.over += cnt[v]
		}
		if !ok || mx == 0 {
			continue
		}
		c.ratio = float64(mn) / float64(mx)
		if c.over < total*8/100 && c.ratio >= 0.15 {
			best = append(best, c)
		}
	}
	sort.Slice(best, func(i, j int) bool { return best[i].ratio > best[j].ratio })
	if len(best) == 0 {
		fmt.Println("    AUCUN champ w=5 quasi-uniforme 0-7 (le player_index n'est pas un champ fixe sur l'ancre arme).")
		return
	}
	for i, c := range best {
		if i >= 6 {
			fmt.Printf("    ... (%d candidats)\n", len(best))
			break
		}
		fmt.Printf("    o=%+d ratio=%.2f >7=%d dist=%v\n", c.o, c.ratio, c.over, c.dist)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
