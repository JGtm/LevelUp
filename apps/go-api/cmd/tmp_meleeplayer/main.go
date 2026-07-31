// tmp_meleeplayer — THROWAWAY : trouve l'offset du player index dans les events MELEE
// (paquets 0xD3 avec id64 arme, méthode collectMelee de tmp_acurtis).
//
// Le "bit20" d'acurtis est FAUX (constant ~6 sur 000d5950 ET 0215fe6b). On SWEEP l'offset
// du player index en absolu (depuis bit0 du paquet) ET en relatif (depuis l'id64 arme), et on
// classe chaque offset par 3 critères empiriques :
//
//	(1) borné 0-7 ; (2) VARIE (non dégénéré) ; (3) pour une arme de POUVOIR UNIQUE
//	    (Diminisher / Rushdown / Mutilator / Bloodblade), la valeur est piecewise-constante
//	    dans le temps (une possession = 1 joueur) et matche le tueur du kill-feed chunk_27.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_meleeplayer [film]
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

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const t0Us = uint64(4537898226)

// roster 000d5950 : xuid -> player index (repris de tmp_roster, validé).
var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

// id64 des armes de pouvoir UNIQUES (utile pour le test de possession).
const (
	idDiminisher = 0x841ac5e5a730e49f
	idRushdown   = 0x841ac5e5d8d07ca1
	idMutilator  = 0xd791556542c9679f
	idBloodblade = 0x4ff3937e1ec48c7a
)

var uniqueWeapons = map[uint64]string{
	idDiminisher: "Diminisher", idRushdown: "Rushdown", idMutilator: "Mutilator", idBloodblade: "Bloodblade",
}

// kills narrés 000d5950 (t sec, id64 arme, killer pi attendu) — VÉRITÉ TERRAIN forte.
type narr struct {
	t      float64
	id64   uint64
	killer int
	label  string
}

var narrated = []narr{
	{147.6, idDiminisher, 3, "LORD PEINX13->Akatsuki Diminisher"},
	{323.9, idRushdown, 5, "Akatsuki->LORD PEINX13 Rushdown"},
	{344.6, idRushdown, 6, "aldusbroncus->LORD PEINX13 Rushdown"},
}

var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

func buildCatalog() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
		id64name[id] = n
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

type pkt struct {
	ts     uint64
	marker byte
	pl     []byte
}

func loadPackets(film string) []pkt {
	cache := root + "/" + film
	var out []pkt
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ == 0 && len(pl) > 0 {
				out = append(out, pkt{ts, pl[0], pl})
			}
		}
	}
	return out
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// mev : un event mêlée (paquet 0xD3 + id64 arme à bit q).
type mev struct {
	tms  int
	id64 uint64
	name string
	fam  string
	q    int // position bit de l'id64
	pl   []byte
}

func collectMelee(pkts []pkt) []mev {
	var out []mev
	for _, p := range pkts {
		if p.marker != 0xD3 {
			continue
		}
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 64
		for bp := 0; bp <= nb; bp++ {
			hi := uint32(bitsAt(p.pl, bp, 32))
			fam, ok := h32name[hi]
			if !ok {
				continue
			}
			id := (uint64(hi) << 32) | uint64(bitsAt(p.pl, bp+32, 32))
			nm, ok := id64name[id]
			if !ok {
				continue
			}
			out = append(out, mev{tms: tms, id64: id, name: nm, fam: fam, q: bp, pl: p.pl})
			bp += 63
		}
	}
	return out
}

// kfKill : kill apparié du kill feed chunk_27.
type kfKill struct {
	killer, victim uint64
	tms            int
}

func loadKills(film string) []kfKill {
	data, _ := os.ReadFile(root + "/" + film + "/chunk_27.bin")
	events, _ := analysis.ParseHighlightEvents(data, 0)
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
	var out []kfKill
	used := make([]bool, len(deaths))
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.xuid == k.xuid {
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
			out = append(out, kfKill{k.xuid, deaths[best].xuid, k.t})
		}
	}
	return out
}

// offCand : un offset candidat (absolu ou relatif à l'id64).
type offCand struct {
	rel bool // true = relatif à q ; false = absolu (bit0)
	off int
	w   int
}

// read renvoie la valeur du champ pour un event donné, ou -1 si hors paquet.
func (c offCand) read(e mev) int {
	pos := c.off
	if c.rel {
		pos = e.q + c.off
	}
	if pos < 0 || pos+c.w > len(e.pl)*8 {
		return -1
	}
	return int(bitsAt(e.pl, pos, c.w))
}

type killPi struct {
	killer, victim, tms int
}

type score struct {
	c            offCand
	bounded      int // #events avec valeur 0-7
	total        int
	distinct     int
	narrHits     int    // #kills narrés matchés (field==killer)
	narrDetail   string // détail par kill narré
	possTrans    int    // transitions intra-possession (armes uniques) — plus bas = mieux
	possEvents   int    // #events d'armes uniques considérés
	killMatch    int    // #kills feed (mêlée) où field==killer
	killMatchTot int
	degenerate   bool
}

func main() {
	film := "000d5950"
	if len(os.Args) >= 2 {
		film = os.Args[1]
	}
	buildCatalog()
	pkts := loadPackets(film)
	evs := collectMelee(pkts)
	kills := loadKills(film)

	// mode probe : film probe <abs|rel> <off> <w> — dump timelines armes uniques à cet offset.
	if len(os.Args) >= 6 && os.Args[2] == "probe" {
		rel := os.Args[3] == "rel"
		var off, w int
		fmt.Sscanf(os.Args[4], "%d", &off)
		fmt.Sscanf(os.Args[5], "%d", &w)
		c := offCand{rel, off, w}
		uniq := map[uint64][]mev{}
		for _, e := range evs {
			if _, ok := uniqueWeapons[e.id64]; ok {
				uniq[e.id64] = append(uniq[e.id64], e)
			}
		}
		for id := range uniq {
			sort.Slice(uniq[id], func(a, b int) bool { return uniq[id][a].tms < uniq[id][b].tms })
		}
		lbl := fmt.Sprintf("abs+%d/W%d", off, w)
		if rel {
			lbl = fmt.Sprintf("id64%+d/W%d", off, w)
		}
		fmt.Printf("=== PROBE %s %s ===\n", film, lbl)
		for id, list := range uniq {
			// segments (transitions) sur q canonique (q<=130) uniquement
			var canon []mev
			for _, e := range list {
				if e.q <= 130 {
					canon = append(canon, e)
				}
			}
			fmt.Printf("\n-- %s : %d events (%d q-canoniques) --\n", uniqueWeapons[id], len(list), len(canon))
			for _, e := range list {
				v := c.read(e)
				tag := ""
				if e.q > 130 {
					tag = "  [q aberrant]"
				}
				fmt.Printf("   %7.1fs field=%d q=%d%s\n", float64(e.tms)/1000, v, e.q, tag)
			}
		}
		return
	}
	fmt.Printf("=== tmp_meleeplayer %s : %d paquets type-0, %d events mêlée, %d kills feed ===\n", film, len(pkts), len(evs), len(kills))

	// distribution q + armes
	qHist := map[int]int{}
	wHist := map[string]int{}
	for _, e := range evs {
		qHist[e.q]++
		wHist[e.name]++
	}
	fmt.Printf("q (position id64) : %s\n", topInts(qHist, 12))
	fmt.Printf("armes : %v\n", wHist)

	// SWEEP restreint aux events à position CANONIQUE (q<=130). Les q aberrants (154..1365) sont
	// des paquets où l'id64 apparaît hors d'un swing (world-state), sans layout stable.
	var cev []mev
	for _, e := range evs {
		if e.q <= 130 {
			cev = append(cev, e)
		}
	}
	fmt.Printf("(sweep sur %d events q-canoniques <=130 ; %d au total)\n", len(cev), len(evs))

	// armes de pouvoir uniques présentes + leurs timelines (canoniques)
	uniqPresent := map[uint64][]mev{}
	for _, e := range cev {
		if _, ok := uniqueWeapons[e.id64]; ok {
			uniqPresent[e.id64] = append(uniqPresent[e.id64], e)
		}
	}
	for id := range uniqPresent {
		sort.Slice(uniqPresent[id], func(a, b int) bool { return uniqPresent[id][a].tms < uniqPresent[id][b].tms })
	}

	// build kill list mapped to pi (film 000d5950 seulement a roster)
	var kpis []killPi
	for _, k := range kills {
		kp, ok1 := xuidToPi[k.killer]
		vp, ok2 := xuidToPi[k.victim]
		if !ok1 || !ok2 {
			continue
		}
		kpis = append(kpis, killPi{kp, vp, k.tms})
	}

	// génère les offsets candidats : absolu 4..120, relatif -112..160 ; W ∈ {5,4,3}.
	// EXCLUSION CRITIQUE : un offset relatif dont la fenêtre chevauche [0,64) lit les BITS de
	// l'id64 lui-même (artefact — id64+41 lisait bits 41-43 de l'id, coïncidence avec 3/5).
	var cands []offCand
	for _, w := range []int{5, 4, 3} {
		for o := 4; o <= 120; o++ {
			cands = append(cands, offCand{false, o, w})
		}
		for o := -112; o <= 160; o++ {
			if o < 64 && o+w > 0 {
				continue // chevauche l'id64 -> lit les bits de l'arme, rejeté
			}
			cands = append(cands, offCand{true, o, w})
		}
	}

	// pré-calcule pour armes uniques : liste triée par temps (déjà fait).
	var scores []score
	for _, c := range cands {
		s := score{c: c}
		dist := map[int]int{}
		for _, e := range cev {
			v := c.read(e)
			if v < 0 {
				continue
			}
			s.total++
			dist[v]++
			if v <= 7 {
				s.bounded++
			}
		}
		s.distinct = len(dist)
		// bounded strict + non-dégénéré
		if s.total == 0 || s.bounded < s.total {
			continue // rejette : au moins un event hors 0-7
		}
		// dégénéré = une seule valeur domine >90%
		maxv := 0
		for _, n := range dist {
			if n > maxv {
				maxv = n
			}
		}
		if s.distinct < 2 || maxv*100 >= s.total*90 {
			s.degenerate = true
		}

		// (3a) transitions intra-possession pour armes uniques
		for _, list := range uniqPresent {
			prev := -999
			for _, e := range list {
				v := c.read(e)
				if v < 0 {
					continue
				}
				s.possEvents++
				if prev != -999 && v != prev {
					s.possTrans++
				}
				prev = v
			}
		}

		// (3b) kills narrés : event le plus proche du même id64
		nd := ""
		for _, n := range narrated {
			var best *mev
			bd := 100000
			for i := range cev {
				if cev[i].id64 != n.id64 {
					continue
				}
				dt := cev[i].tms - int(n.t*1000)
				if dt < 0 {
					dt = -dt
				}
				if dt < bd {
					bd, best = dt, &cev[i]
				}
			}
			if best != nil {
				v := c.read(*best)
				ok := v == n.killer
				if ok {
					s.narrHits++
				}
				nd += fmt.Sprintf("[%s Δ%dms v=%d exp=%d %v] ", n.label, bd, v, n.killer, ok)
			}
		}
		s.narrDetail = nd

		// (4) correlation kill-feed : pour chaque kill (pi connus), event mêlée le plus proche ±600ms -> field==killer ?
		if len(kpis) > 0 {
			for _, kp := range kpis {
				var best *mev
				bd := 600
				for i := range evs {
					dt := evs[i].tms - kp.tms
					if dt < 0 {
						dt = -dt
					}
					if dt <= bd {
						bd, best = dt, &evs[i]
					}
				}
				if best != nil {
					s.killMatchTot++
					if c.read(*best) == kp.killer {
						s.killMatch++
					}
				}
			}
		}
		scores = append(scores, s)
	}

	// classe : narrHits desc, puis non-dégénéré, puis possTrans asc, puis killMatch desc
	sort.Slice(scores, func(a, b int) bool {
		if scores[a].narrHits != scores[b].narrHits {
			return scores[a].narrHits > scores[b].narrHits
		}
		if scores[a].degenerate != scores[b].degenerate {
			return !scores[a].degenerate
		}
		if scores[a].possTrans != scores[b].possTrans {
			return scores[a].possTrans < scores[b].possTrans
		}
		return scores[a].killMatch > scores[b].killMatch
	})

	fmt.Printf("\n=== TOP 25 offsets (narrHits desc, non-dégén, possTrans asc, killMatch desc) ===\n")
	fmt.Printf("%-16s %-8s %-6s %-9s %-10s %-14s\n", "offset", "distinct", "narr", "possTrans", "killMatch", "flags")
	for i, s := range scores {
		if i >= 25 {
			break
		}
		lbl := fmt.Sprintf("abs+%d/W%d", s.c.off, s.c.w)
		if s.c.rel {
			lbl = fmt.Sprintf("id64%+d/W%d", s.c.off, s.c.w)
		}
		fl := ""
		if s.degenerate {
			fl = "DEGEN"
		}
		fmt.Printf("%-16s %-8d %d/3   %2d/%-6d %2d/%-7d %-14s %s\n",
			lbl, s.distinct, s.narrHits, s.possTrans, s.possEvents, s.killMatch, s.killMatchTot, fl, s.narrDetail)
	}

	// détail du meilleur non-dégénéré avec narrHits max
	var best *score
	for i := range scores {
		if !scores[i].degenerate {
			best = &scores[i]
			break
		}
	}
	if best == nil && len(scores) > 0 {
		best = &scores[0]
	}
	if best != nil {
		dumpBest(*best, evs, uniqPresent)
	}

	// === DIAGNOSTIC GROUND-TRUTH ===
	// 1) kill feed pi-mappé complet
	fmt.Printf("\n=== KILL FEED (pi-mappé) ===\n")
	for _, kp := range kpis {
		fmt.Printf("   %6.1fs  %-16s -> %-16s\n", float64(kp.tms)/1000, piName[kp.killer], piName[kp.victim])
	}

	// 2) HANDOFF SCAN : pour chaque arme UNIQUE, chercher les offsets où le champ VARIE dans le
	//    temps (>=2 segments) tout en restant 0-7 — candidat "live holder". On liste, pour Rushdown
	//    et Diminisher, les offsets non-constants avec leur séquence temporelle de valeurs.
	for _, id := range []uint64{idRushdown, idDiminisher} {
		list := uniqPresent[id]
		if len(list) < 3 {
			continue
		}
		fmt.Printf("\n=== HANDOFF SCAN %s (%d events) : offsets NON-constants bornés 0-7 ===\n", uniqueWeapons[id], len(list))
		type seq struct {
			c    offCand
			vals []int
			segs int
		}
		var seqs []seq
		var allC []offCand
		for _, w := range []int{3, 4, 5} {
			for o := -60; o <= 60; o++ {
				allC = append(allC, offCand{true, o, w})
			}
			for o := 8; o <= 120; o++ {
				allC = append(allC, offCand{false, o, w})
			}
		}
		for _, c := range allC {
			var vals []int
			bounded := true
			for _, e := range list {
				v := c.read(e)
				if v < 0 || v > 7 {
					bounded = false
					break
				}
				vals = append(vals, v)
			}
			if !bounded {
				continue
			}
			segs := 1
			for i := 1; i < len(vals); i++ {
				if vals[i] != vals[i-1] {
					segs++
				}
			}
			distinct := map[int]bool{}
			for _, v := range vals {
				distinct[v] = true
			}
			// on veut : varie (>=2 segments) mais pas du bruit (segs <= 4) et <=3 valeurs distinctes
			if segs >= 2 && segs <= 4 && len(distinct) >= 2 && len(distinct) <= 3 {
				seqs = append(seqs, seq{c, vals, segs})
			}
		}
		sort.Slice(seqs, func(a, b int) bool { return seqs[a].segs < seqs[b].segs })
		if len(seqs) == 0 {
			fmt.Printf("   AUCUN offset non-constant borné 0-7 (le champ est weapon-static à toutes les positions testées)\n")
		}
		for i, s := range seqs {
			if i >= 15 {
				fmt.Printf("   ... (%d offsets non-constants au total)\n", len(seqs))
				break
			}
			lbl := fmt.Sprintf("abs+%d/W%d", s.c.off, s.c.w)
			if s.c.rel {
				lbl = fmt.Sprintf("id64%+d/W%d", s.c.off, s.c.w)
			}
			fmt.Printf("   %-14s segs=%d vals=%v\n", lbl, s.segs, s.vals)
		}
	}

	// === CRUX : peut-on distinguer le holder Diminisher(pi3) du holder Rushdown(pi5) ? ===
	// Diminisher & Rushdown sont au MÊME q=90 (structure identique, ne diffèrent que par l'id64).
	// On cherche TOUS les offsets (y compris dans l'id) où les deux marteaux sont uniformes dans le
	// temps ET portent des valeurs DIFFÉRENTES. Si TOUS ces offsets tombent dans l'id64 [0,64),
	// alors aucun champ acteur réel ne distingue les deux joueurs -> le player n'est PAS dans ces paquets.
	cruxDistinguish(uniqPresent)

	// === KILLRANK : vérité terrain = KILLS mêlée du feed ===
	// Un event mêlée (position canonique q<=150) proche (±win) d'un kill : l'acteur du swing
	// létal devrait être le TUEUR. On classe chaque offset par #(field==killer) et #(field==victim).
	runKillRank(evs, kpis)
}

// uniformVal renvoie la valeur unique du champ sur la liste (ou -1 si non uniforme / hors borne).
func uniformVal(c offCand, list []mev) int {
	v := -1
	for _, e := range list {
		x := c.read(e)
		if x < 0 || x > 7 {
			return -1
		}
		if v == -1 {
			v = x
		} else if x != v {
			return -2 // non uniforme
		}
	}
	return v
}

func cruxDistinguish(uniq map[uint64][]mev) {
	dim := uniq[idDiminisher]
	rush := uniq[idRushdown]
	if len(dim) == 0 || len(rush) == 0 {
		fmt.Printf("\n=== CRUX : Diminisher/Rushdown absents, test non applicable ===\n")
		return
	}
	fmt.Printf("\n=== CRUX : offsets où Diminisher & Rushdown sont uniformes ET DISTINCTS ===\n")
	nOut, nIn := 0, 0
	for _, w := range []int{5, 4, 3} {
		for o := -96; o <= 160; o++ {
			c := offCand{true, o, w}
			dv := uniformVal(c, dim)
			rv := uniformVal(c, rush)
			if dv < 0 || rv < 0 || dv == rv {
				continue
			}
			inID := o < 64 && o+w > 0
			tag := "HORS-id (champ réel candidat)"
			if inID {
				tag = "DANS id64 (artefact bits d'arme)"
				nIn++
			} else {
				nOut++
			}
			if !inID || nIn <= 6 {
				fmt.Printf("   id64%+d/W%d : Diminisher=%d Rushdown=%d  [%s]\n", o, w, dv, rv, tag)
			}
		}
	}
	fmt.Printf("   >>> offsets distinctifs DANS l'id64=%d, HORS id64=%d\n", nIn, nOut)
	if nOut == 0 {
		fmt.Printf("   >>> CONCLUSION : AUCUN champ hors-id ne distingue pi3 de pi5. Le holder n'est\n")
		fmt.Printf("       PAS encodé dans ces paquets 0xD3 (structure identique sauf l'id de l'arme).\n")
	}
}

func runKillRank(evs []mev, kpis []killPi) {
	// canonical q : on jette les positions aberrantes (q>150) qui cassent les offsets absolus.
	var cev []mev
	for _, e := range evs {
		if e.q <= 150 {
			cev = append(cev, e)
		}
	}
	const win = 250
	// ground truth : pour chaque event canonique, le kill le plus proche ±win.
	type gt struct {
		e              mev
		killer, victim int
		dt             int
	}
	var gts []gt
	for _, e := range cev {
		best := -1
		bd := win + 1
		for i, kp := range kpis {
			dt := e.tms - kp.tms
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			gts = append(gts, gt{e, kpis[best].killer, kpis[best].victim, bd})
		}
	}
	fmt.Printf("\n=== KILLRANK : %d events canoniques, %d appariés à un kill ±%dms ===\n", len(cev), len(gts), win)

	var cands []offCand
	for _, w := range []int{5, 4, 3} {
		for o := 4; o <= 120; o++ {
			cands = append(cands, offCand{false, o, w})
		}
		for o := -112; o <= 48; o++ {
			cands = append(cands, offCand{true, o, w})
		}
	}
	type kr struct {
		c              offCand
		killer, victim int
		bounded, total int
	}
	var krs []kr
	for _, c := range cands {
		k := kr{c: c}
		bad := false
		for _, e := range cev {
			v := c.read(e)
			if v < 0 || v > 7 {
				bad = true
				break
			}
		}
		if bad {
			continue // exige borné 0-7 sur TOUS les events canoniques
		}
		for _, g := range gts {
			v := c.read(g.e)
			k.total++
			if v == g.killer {
				k.killer++
			}
			if v == g.victim {
				k.victim++
			}
		}
		krs = append(krs, k)
	}
	sort.Slice(krs, func(a, b int) bool { return krs[a].killer > krs[b].killer })
	fmt.Printf("-- TOP par field==KILLER (borné 0-7 partout) --\n")
	for i, k := range krs {
		if i >= 15 {
			break
		}
		lbl := fmt.Sprintf("abs+%d/W%d", k.c.off, k.c.w)
		if k.c.rel {
			lbl = fmt.Sprintf("id64%+d/W%d", k.c.off, k.c.w)
		}
		fmt.Printf("   %-14s killer=%2d/%d  victim=%2d/%d\n", lbl, k.killer, k.total, k.victim, k.total)
	}
	sort.Slice(krs, func(a, b int) bool { return krs[a].victim > krs[b].victim })
	fmt.Printf("-- TOP par field==VICTIM --\n")
	for i, k := range krs {
		if i >= 8 {
			break
		}
		lbl := fmt.Sprintf("abs+%d/W%d", k.c.off, k.c.w)
		if k.c.rel {
			lbl = fmt.Sprintf("id64%+d/W%d", k.c.off, k.c.w)
		}
		fmt.Printf("   %-14s victim=%2d/%d  killer=%2d/%d\n", lbl, k.victim, k.total, k.killer, k.total)
	}
	// liste des events appariés (pour inspection manuelle)
	fmt.Printf("-- events mêlée appariés à un kill (t, arme, Δ, killer->victim) --\n")
	sort.Slice(gts, func(a, b int) bool { return gts[a].e.tms < gts[b].e.tms })
	for _, g := range gts {
		fmt.Printf("   %6.1fs %-18s Δ%4dms  %-16s -> %-16s\n", float64(g.e.tms)/1000, g.e.name, g.dt, piName[g.killer], piName[g.victim])
	}
}

func dumpBest(s score, evs []mev, uniq map[uint64][]mev) {
	lbl := fmt.Sprintf("abs+%d/W%d", s.c.off, s.c.w)
	if s.c.rel {
		lbl = fmt.Sprintf("id64%+d/W%d", s.c.off, s.c.w)
	}
	fmt.Printf("\n=== MEILLEUR CANDIDAT %s ===\n", lbl)
	fmt.Printf("narrDetail : %s\n", s.narrDetail)
	fmt.Printf("distribution complète : ")
	dist := map[int]int{}
	for _, e := range evs {
		v := s.c.read(e)
		dist[v]++
	}
	fmt.Printf("%s\n", fmtMap(dist))

	// timelines des armes uniques
	ids := []uint64{idDiminisher, idRushdown, idMutilator, idBloodblade}
	for _, id := range ids {
		list := uniq[id]
		if len(list) == 0 {
			continue
		}
		fmt.Printf("\n-- %s (%d events) timeline (t : field) --\n", uniqueWeapons[id], len(list))
		for _, e := range list {
			v := s.c.read(e)
			nm := ""
			if v >= 0 && v <= 7 {
				nm = piName[v]
			}
			fmt.Printf("   %6.1fs  field=%d (%-16s) q=%d\n", float64(e.tms)/1000, v, nm, e.q)
		}
	}
}

func fmtMap(m map[int]int) string {
	ks := []int{}
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

func topInts(m map[int]int, lim int) string {
	type kv struct{ k, v int }
	var xs []kv
	for k, v := range m {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(a, b int) bool { return xs[a].v > xs[b].v })
	s := ""
	for i, x := range xs {
		if i >= lim {
			break
		}
		s += fmt.Sprintf("%d:%d ", x.k, x.v)
	}
	return s
}
