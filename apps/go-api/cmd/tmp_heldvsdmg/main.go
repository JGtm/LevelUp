// tmp_heldvsdmg — THROWAWAY (A.0) : valide le modèle HELD-WEAPON (arme tenue du tueur <= T)
// contre les 519 RECORDS DE DÉGÂT (famille + temps, sans auteur) du film 000d5950.
//
// Question : pour chaque kill, la famille tenue par le tueur (dernier event fire/melee <= T) est-elle
// corroborée par un record de dégât de CETTE famille proche de T ?
//   - MÊLÉE (held=marteau/épée) : NON corroborable par construction (la mêlée ne produit pas de record
//     de dégât ; Gravity Hammer = 0/519). On les compte à part.
//   - ARME À FEU (held=fire) : corroborable. Taux = (held-fam présent dans record [T-1500,T+200]) / gun-kills.
//
// Indépendamment : proxy = famille du record de dégât le plus proche AVANT T ; accord held vs proxy.
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
const deserStartBit = 36
const variantSuffix = uint32(0x42c9679f)

var h32 = map[uint32]string{}
var grenades = map[uint32]string{0xB0171062: "Frag Grenade", 0xC0E34C44: "Plasma Grenade", 0x3B2567D4: "Shock Grenade", 0x9212E428: "Spike Grenade"}
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}
var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}

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
		if q>>3 >= len(p) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+size]})
		off += 16 + size
	}
	return out
}
func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

// ---- 519 records de dégât : famille + tms ----
type dmgRec struct {
	tms int
	fam string
}

func decodeRecordFam(payload []byte) (fam string, ok bool) {
	br := filmdec.NewBitReader(payload)
	br.Skip(deserStartBit)
	if !br.ReadBit() {
		br.ReadBits(2)
	}
	br.ReadBits(5)
	gid := uint32(br.ReadBits(32))
	low := uint32(br.ReadBits(32))
	if low != variantSuffix {
		return "", false
	}
	fam, ok = h32[gid]
	return
}

func damageRecords() []dmgRec {
	var out []dmgRec
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d) {
			if p.typ != 0 || len(p.payload) == 0 || p.payload[0] != 0xd2 {
				continue
			}
			fam, ok := decodeRecordFam(p.payload)
			if !ok {
				continue
			}
			out = append(out, dmgRec{tsToMs(p.ts), fam})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tms < out[j].tms })
	return out
}

// ---- held events (fire+melee, type-0 propre) ----
type actEvt struct {
	tms  int
	kind string
	wpn  string
	pidx int
}

func heldEvents() []actEvt {
	var evs []actEvt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d) {
			if p.ts < t0Us {
				continue
			}
			tms := tsToMs(p.ts)
			if tms < 0 || tms > maxMatchMs {
				continue
			}
			pl := p.payload
			total := len(pl) * 8
			// MELEE (type-0 seulement utile pour held)
			for bp := 0; bp+120 < total; bp++ {
				m := bitsAt(pl, bp, 11)
				if m != 0x534 && m != 0x535 {
					continue
				}
				anchor := bp + 3
				typ := uint8(bitsAt(pl, anchor+76, 8))
				var woff int
				switch typ {
				case 0x47:
					woff = anchor + 86
				case 0x42:
					woff = anchor + 88
				case 0x60:
					woff = anchor + 101
				default:
					continue
				}
				hi := uint32(bitsAt(pl, woff, 32))
				name, ok := h32[hi]
				if !ok && typ == 0x60 {
					hi = uint32(bitsAt(pl, anchor+103, 32))
					name, ok = h32[hi]
				}
				if !ok {
					continue
				}
				evs = append(evs, actEvt{tms, "melee", name, int(bitsAt(pl, anchor+23, 5))})
			}
			// FIRE (type-0 seulement = vrais events de tir, pas keyframe)
			if p.typ == 0 {
				for bp := 4; bp+96 < total; bp++ {
					hi := uint32(bitsAt(pl, bp, 32))
					name, ok := h32[hi]
					if !ok {
						continue
					}
					if uint32(bitsAt(pl, bp+32, 32)) != variantSuffix {
						continue
					}
					evs = append(evs, actEvt{tms, "fire", name, int(bitsAt(pl, bp-4, 5))})
				}
			}
		}
	}
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	return evs
}

// allActions = heldEvents (fire+melee) + grenade (type-0 propre).
func allActions() []actEvt {
	evs := heldEvents()
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d) {
			if p.ts < t0Us {
				continue
			}
			tms := tsToMs(p.ts)
			if tms < 0 || tms > maxMatchMs {
				continue
			}
			pl := p.payload
			total := len(pl) * 8
			for bp := 0; bp+110 < total; bp++ {
				if bitsAt(pl, bp, 24) != 0x4c0c00 {
					continue
				}
				if gname, ok := grenades[uint32(bitsAt(pl, bp+24, 32))]; ok {
					evs = append(evs, actEvt{tms, "grenade", gname, int(bitsAt(pl, bp+24+32+47, 5))})
				}
			}
		}
	}
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	return evs
}

// ---- kill feed ----
type kfRow struct {
	killerPi, victimPi int
	t                  int
}

func killFeed() []kfRow {
	raw, _ := os.ReadFile(cache + "/chunk_27.bin")
	events, _ := analysis.ParseHighlightEvents(raw, 0)
	type ev struct {
		xuid uint64
		t    int
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
			if usedD[i] || d.xuid == k.xuid {
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
		kp, ok := xuidToPi[k.xuid]
		if !ok {
			kp = -1
		}
		vp, ok2 := xuidToPi[deaths[best].xuid]
		if !ok2 {
			vp = -1
		}
		feed = append(feed, kfRow{kp, vp, k.t})
	}
	return feed
}

func main() {
	build()
	dmg := damageRecords()
	held := heldEvents()
	feed := killFeed()
	fmt.Printf("=== %d records de dégât (0xd2, famille connue) ; %d held-events (fire+melee) ; %d kills ===\n",
		len(dmg), len(held), len(feed))

	// held-weapon par kill : dernier fire/melee du tueur <= T
	const corrLo, corrHi = -1500, 200 // fenêtre record de dégât autour de T (avant surtout)
	var gunKills, gunCorrob, meleeKills, noHeld, agreeProxy, gunWithProxy int
	type row struct {
		t                 int
		killer, victim    string
		heldKind, heldFam string
		heldDt            int
		corrob            bool
		proxyFam          string
	}
	var rows []row
	for _, k := range feed {
		if k.killerPi < 0 {
			continue
		}
		// held
		bi, bt := -1, -1
		for i, a := range held {
			if a.pidx != k.killerPi || (a.kind != "fire" && a.kind != "melee") {
				continue
			}
			if a.tms <= k.t && a.tms > bt {
				bt, bi = a.tms, i
			}
		}
		// proxy : famille du record de dégât le plus proche AVANT T (toutes familles, sans auteur)
		proxyFam := ""
		bpd := 1 << 30
		for _, r := range dmg {
			if r.tms <= k.t && k.t-r.tms < bpd {
				bpd, proxyFam = k.t-r.tms, r.fam
			}
		}
		if proxyFam != "" {
			gunWithProxy++
		}
		r := row{t: k.t, killer: piName[k.killerPi], proxyFam: proxyFam}
		if k.victimPi >= 0 {
			r.victim = piName[k.victimPi]
		}
		if bi < 0 {
			noHeld++
			r.heldKind = "(aucune)"
			rows = append(rows, r)
			continue
		}
		a := held[bi]
		r.heldKind, r.heldFam, r.heldDt = a.kind, a.wpn, a.tms-k.t
		if a.kind == "melee" {
			meleeKills++
		} else {
			gunKills++
			// corroboration : held-fam présent dans un record de dégât [T+corrLo, T+corrHi]
			for _, rec := range dmg {
				if rec.tms >= k.t+corrLo && rec.tms <= k.t+corrHi && rec.fam == a.wpn {
					r.corrob = true
					break
				}
			}
			if r.corrob {
				gunCorrob++
			}
			if proxyFam == a.wpn {
				agreeProxy++
			}
		}
		rows = append(rows, r)
	}

	fmt.Printf("\n=== RÉSULTAT A.0 ===\n")
	fmt.Printf("  kills held=MÊLÉE (non corroborable par 519, attendu) : %d\n", meleeKills)
	fmt.Printf("  kills held=ARME À FEU : %d\n", gunKills)
	if gunKills > 0 {
		fmt.Printf("    -> held-fam corroborée par un record de dégât même famille [%dms,%dms] : %d/%d (%.0f%%)\n",
			corrLo, corrHi, gunCorrob, gunKills, 100*float64(gunCorrob)/float64(gunKills))
		fmt.Printf("    -> held-fam == proxy (record dégât le plus proche avant T) : %d/%d (%.0f%%)\n",
			agreeProxy, gunKills, 100*float64(agreeProxy)/float64(gunKills))
	}
	fmt.Printf("  kills sans arme tenue : %d\n", noHeld)

	// baseline proxy seul : combien de kills ont un record de dégât proche (sans held)
	var proxyAt300 int
	for _, k := range feed {
		for _, r := range dmg {
			if abs(r.tms-k.t) <= 300 {
				proxyAt300++
				break
			}
		}
	}
	fmt.Printf("  (baseline : kills avec un record de dégât ±300ms = %d/%d)\n", proxyAt300, len(feed))

	// dump détaillé des kills gun (held vs corrob vs proxy)
	fmt.Printf("\n=== Détail kills ARME À FEU (held vs corroboration vs proxy) ===\n")
	sort.Slice(rows, func(i, j int) bool { return rows[i].t < rows[j].t })
	for _, r := range rows {
		if r.heldKind != "fire" {
			continue
		}
		mark := "✗"
		if r.corrob {
			mark = "✓"
		}
		fmt.Printf("  t=%6.1fs %-16s->%-16s held=%-22s(dt=%+ds) corrob=%s proxy=%s\n",
			float64(r.t)/1000, r.killer, r.victim, r.heldFam, r.heldDt/1000, mark, r.proxyFam)
	}

	// ---- MODÈLE EVENT-À-T (le modèle PRÉCIS) : event auteur le plus proche de T (fire+melee+grenade) ----
	// Mesure la vraie densité de capture : combien de kills ont un event de l'auteur près de T,
	// et de ceux-là combien sont corroborés (gun) par un record de dégât même famille.
	grenadeEvents := allActions() // fire+melee+grenade type-0 propre
	fmt.Printf("\n=== MODÈLE EVENT-À-T (event auteur le plus proche, tous types) ===\n")
	for _, W := range []int{500, 1500, 3000} {
		var nKills, nGun, nGunCorrob int
		for _, k := range feed {
			if k.killerPi < 0 {
				continue
			}
			bi, bd := -1, 1<<30
			for i, a := range grenadeEvents {
				if a.pidx != k.killerPi {
					continue
				}
				if d := abs(a.tms - k.t); d < bd {
					bd, bi = d, i
				}
			}
			if bi < 0 || bd > W {
				continue
			}
			nKills++
			a := grenadeEvents[bi]
			if a.kind == "fire" {
				nGun++
				for _, rec := range dmg {
					if rec.tms >= k.t-1500 && rec.tms <= k.t+200 && rec.fam == a.wpn {
						nGunCorrob++
						break
					}
				}
			}
		}
		corr := 0.0
		if nGun > 0 {
			corr = 100 * float64(nGunCorrob) / float64(nGun)
		}
		fmt.Printf("  fenêtre ±%4dms : %d/%d kills ont un event auteur ; gun=%d dont corroborés=%d (%.0f%%)\n",
			W, nKills, len(feed), nGun, nGunCorrob, corr)
	}

	// narration
	fmt.Printf("\n=== Narration (held + record dégât dans ±1500ms) ===\n")
	for _, nr := range []struct {
		kp, vp int
		label  string
		times  []int
	}{
		{4, 2, "Marteau IKE->JGtm", []int{115500, 292500, 355700, 375100}},
		{2, 5, "BR75 JGtm->Akatsuki", []int{112900, 329800}},
	} {
		fmt.Printf("-- %s --\n", nr.label)
		for _, T := range nr.times {
			fams := map[string]int{}
			for _, r := range dmg {
				if r.tms >= T-1500 && r.tms <= T+1500 {
					fams[r.fam]++
				}
			}
			// held du tueur <= T
			hf, ht := "(aucune)", -1
			for _, a := range held {
				if a.pidx == nr.kp && (a.kind == "fire" || a.kind == "melee") && a.tms <= T && a.tms > ht {
					ht, hf = a.tms, a.wpn
				}
			}
			fmt.Printf("   T=%.1fs held=%s | records±1.5s=%v\n", float64(T)/1000, hf, fams)
		}
	}
}
