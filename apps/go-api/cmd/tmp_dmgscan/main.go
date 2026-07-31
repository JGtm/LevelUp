// tmp_dmgscan — S1 : SCANNER TOUS LES RECORDS DE DEGAT du film 000d5950.
//
// Acquis (capture live + D1) : un record de degat est un paquet type-0 dont le
// PAYLOAD commence par l'entete de message (8 o consommes en amont par
// FUN_14080AADE), puis le deser FUN_14080c1f8 lit, a partir du bit 36 du payload :
//
//	+0x08 slot/cause  consumeId2 = R(1); si 0 R(2)
//	+0x0c global-id   R(5) + R(32) BE  -> le R(32) = high-32 FAMILLE d'arme
//	+0x10 handle      R(1) gate ; si set R(32)
//	+0x14 variant     R(32) BE (sous-variant)
//
// Le high-32 famille (global-id) est immediatement suivi du low-32 suffixe
// 0x42c9679f -> id64 complet contigu, cle de analysis.WeaponIDToName.
//
// PHASE 1 : trouver le DISCRIMINANT qui separe les records de degat des frames d'etat.
//
//	On parse TOUS les type-0, et pour chacun on teste plusieurs signatures candidates
//	(entete fixe, taille, suffixe variant a offset stable). On choisit le discriminant
//	qui isole une population coherente (familles d'arme decodables a startBit=36).
//
// PHASE 2 : pour chaque record de degat : decode famille (global-id @bit44) +
//
//	slot/cause (+0x08) + ts du header paquet.
//
// PHASE 3 : compte, distribution des familles, timeline.
//
// Usage : tmp_dmgscan
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

// startBit logique de demarrage du deser dans le payload (prouve par D1).
const deserStartBit = 36

// suffixe variant universel des armes (low-32 de tout id64 d'arme).
const variantSuffix = uint32(0x42c9679f)

var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

func build() {
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
	typ      uint16
	b2, b3   byte
	size     int
	ts       uint64
	payload  []byte
	chunkIdx int
}

func listPackets(d []byte, chunkIdx int) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		b2, b3 := d[off+2], d[off+3]
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, b2, b3, size, ts, d[off+16 : off+16+size], chunkIdx})
		off += 16 + size
	}
	return out
}

func allType0() []pkt {
	var all []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		if len(d) == 0 {
			continue
		}
		for _, p := range listPackets(d, n) {
			if p.typ == 0 {
				all = append(all, p)
			}
		}
	}
	return all
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

// decodeRecord rejoue le deser depuis deserStartBit sur le payload et renvoie
// slot/cause (bits consommes), la famille high-32 (global-id) et l'id64 complet.
func decodeRecord(payload []byte) (slotBits int, gid uint32, fam string, id64 uint64, famKnown bool) {
	br := filmdec.NewBitReader(payload)
	br.Skip(deserStartBit)
	// +0x08 slot/cause : R(1); si 0 R(2)
	start := br.BitPos()
	if !br.ReadBit() {
		br.ReadBits(2)
	}
	slotBits = br.BitPos() - start
	// +0x0c global-id : R(5) + R(32) BE
	br.ReadBits(5)
	gid = uint32(br.ReadBits(32))
	// low-32 contigu (suffixe variant)
	low := uint32(br.ReadBits(32))
	id64 = (uint64(gid) << 32) | uint64(low)
	fam, famKnown = h32name[gid]
	return
}

// readSlotCause lit la valeur slot/cause (consumeId2) : R(1); si 0 R(2).
// Renvoie une valeur lisible : 0 si bit==1 (forme courte), sinon 1..4 = 1+R(2).
func readSlotCause(payload []byte) uint64 {
	br := filmdec.NewBitReader(payload)
	br.Skip(deserStartBit)
	if br.ReadBit() {
		return 0 // forme courte (1 bit) — non observée sur les records de dégât-arme
	}
	return 1 + br.ReadBits(2) // 1..4 selon les 2 bits suivants
}

func main() {
	build()
	all := allType0()
	fmt.Printf("=== %d paquets type-0 ; catalogue : %d familles, %d id64 ===\n\n", len(all), len(h32name), len(id64name))

	// ── PHASE 1 : caractériser le DISCRIMINANT. ──
	// On teste 3 candidats sur tous les type-0 :
	//   (A) suffixe variant 0x42c9679f present quelque part (byte ou bit aligne) ;
	//   (B) decodage a startBit=36 donne une FAMILLE connue (global-id) suivie du
	//       suffixe 0x42c9679f contigu (= record de degat ARME, le plus strict) ;
	//   (C) entete fixe : 1er octet du payload.
	fmt.Println("=== PHASE 1 : recherche du discriminant ===")

	// (C) distribution du 1er octet du payload + de la taille.
	firstByte := map[byte]int{}
	sizeHist := map[int]int{}
	for _, p := range all {
		if len(p.payload) > 0 {
			firstByte[p.payload[0]]++
		}
		sizeHist[len(p.payload)]++
	}
	// top first-bytes
	type bc struct {
		b byte
		c int
	}
	var fbs []bc
	for b, c := range firstByte {
		fbs = append(fbs, bc{b, c})
	}
	sort.Slice(fbs, func(i, j int) bool { return fbs[i].c > fbs[j].c })
	fmt.Printf("  1er octet payload (top 8) : ")
	for i := 0; i < len(fbs) && i < 8; i++ {
		fmt.Printf("0x%02x:%d ", fbs[i].b, fbs[i].c)
	}
	fmt.Println()

	// (B) décodage strict : record de dégât ARME = famille connue @startBit36 +
	// suffixe variant contigu. C'est le discriminant retenu.
	type rec struct {
		ts       uint64
		tms      int
		fam      string
		id64     uint64
		slotBits int
		slotVal  uint64
		chunk    int
		size     int
	}
	var armRecs []rec      // global-id décode une famille ARME catalogue
	var headerHits int     // payload commence par l'entête type d2 60 44 00-like
	var suffixAnywhere int // suffixe variant présent (bit-aligné) n'importe où

	for _, p := range all {
		d := p.payload
		// suffixe présent ?
		hasSuffix := false
		mx := len(d)*8 - 32
		for bp := 0; bp <= mx; bp++ {
			if uint32(bitsAt(d, bp, 32)) == variantSuffix {
				hasSuffix = true
				break
			}
		}
		if hasSuffix {
			suffixAnywhere++
		}
		// décode au startBit canonique
		slotBits, gid, fam, id64, known := decodeRecord(d)
		// record de dégât ARME : global-id = famille connue ET suffixe contigu attendu.
		low := uint32(id64)
		if known && low == variantSuffix {
			armRecs = append(armRecs, rec{
				ts: p.ts, tms: tsToMs(p.ts), fam: fam, id64: id64,
				slotBits: slotBits, slotVal: readSlotCause(d), chunk: p.chunkIdx, size: len(d),
			})
			_ = gid
		}
		// entête fixe : payload[0]==0xd2 (signature du message capturé) ?
		if len(d) >= 4 && d[0] == 0xd2 {
			headerHits++
		}
	}

	fmt.Printf("  type-0 avec suffixe variant 0x%08x present (bit-aligne) : %d\n", variantSuffix, suffixAnywhere)
	fmt.Printf("  type-0 dont payload[0]==0xd2 (entete capturee) : %d\n", headerHits)
	fmt.Printf("  >>> DISCRIMINANT retenu = decode @startBit36 -> global-id famille connue + suffixe 0x%08x contigu\n", variantSuffix)
	fmt.Printf("  >>> RECORDS DE DEGAT ARME trouves : %d\n", len(armRecs))

	// Complétude : combien de records décodés ont payload[0]==0xd2 vs autre ?
	// (verifie que l'entete 0xd2 et le decodage strict designent la meme population).
	d2AndArm, armNotD2 := 0, 0
	for _, p := range all {
		_, _, _, id64, known := decodeRecord(p.payload)
		isArm := known && uint32(id64) == variantSuffix
		isD2 := len(p.payload) > 0 && p.payload[0] == 0xd2
		if isArm && isD2 {
			d2AndArm++
		}
		if isArm && !isD2 {
			armNotD2++
		}
	}
	fmt.Printf("  >>> recoupement : armRec ET 0xd2 = %d ; armRec mais PAS 0xd2 = %d (=> entete 0xd2 = discriminant equivalent)\n\n", d2AndArm, armNotD2)

	// ── PHASE 3 : distribution des familles + timeline. ──
	sort.Slice(armRecs, func(i, j int) bool { return armRecs[i].tms < armRecs[j].tms })

	famCount := map[string]int{}
	slotCount := map[uint64]int{}
	for _, r := range armRecs {
		famCount[r.fam]++
		slotCount[r.slotVal]++
	}

	fmt.Println("=== PHASE 3a : distribution des FAMILLES d'arme (records de degat) ===")
	type fc struct {
		n string
		c int
	}
	var fcs []fc
	for n, c := range famCount {
		fcs = append(fcs, fc{n, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	for _, f := range fcs {
		fmt.Printf("  %-28s x%d\n", f.n, f.c)
	}

	// couverture temporelle des records.
	if len(armRecs) > 0 {
		fmt.Printf("\n  couverture temporelle records : t=%.1fs .. %.1fs (%d records)\n",
			float64(armRecs[0].tms)/1000, float64(armRecs[len(armRecs)-1].tms)/1000, len(armRecs))
	}
	// Gravity Hammer (mêlée) présent ? scan id64 marteau dans TOUS les type-0 (pas que records 0xd2).
	const hammerID64 = uint64(0x841ac5e542c9679f)
	hammerInRec, hammerInAll := 0, 0
	for _, p := range all {
		// dans un record 0xd2 décodé ?
		_, gid, _, id64, _ := decodeRecord(p.payload)
		if id64 == hammerID64 {
			hammerInRec++
		}
		_ = gid
		// n'importe où bit-aligné dans le payload ?
		mx := len(p.payload)*8 - 64
		for bp := 0; bp <= mx; bp++ {
			if (uint64(bitsAt(p.payload, bp, 32))<<32)|uint64(bitsAt(p.payload, bp+32, 32)) == hammerID64 {
				hammerInAll++
				break
			}
		}
	}
	fmt.Printf("  Gravity Hammer (id64 0x%016x) : %d dans record décodé strict, %d type-0 le contiennent (n'importe où)\n",
		hammerID64, hammerInRec, hammerInAll)

	fmt.Println("\n=== PHASE 3b : distribution slot/cause (+0x08) ===")
	var scs []struct {
		v uint64
		c int
	}
	for v, c := range slotCount {
		scs = append(scs, struct {
			v uint64
			c int
		}{v, c})
	}
	sort.Slice(scs, func(i, j int) bool { return scs[i].c > scs[j].c })
	for _, s := range scs {
		fmt.Printf("  cause=%d  x%d\n", s.v, s.c)
	}

	fmt.Println("\n=== PHASE 3c : timeline (40 premiers records : ts, famille, cause, slotBits) ===")
	for i, r := range armRecs {
		if i >= 40 {
			fmt.Printf("  ... (%d records au total)\n", len(armRecs))
			break
		}
		fmt.Printf("  [%3d] t=%7.1fs  %-26s cause=%d slotBits=%d (chunk%02d size=%d)\n",
			i, float64(r.tms)/1000, r.fam, r.slotVal, r.slotBits, r.chunk, r.size)
	}

	// ── PHASE 4 : sanity — la 1ere occurrence (chunk_02 @le record capturé) ──
	fmt.Println("\n=== PHASE 4 : sanity sur le record CAPTURE (1er Disruptor attendu) ===")
	for _, r := range armRecs {
		if r.fam == "Disruptor" {
			fmt.Printf("  premier Disruptor : t=%.1fs cause=%d chunk%02d size=%d id64=0x%016x\n",
				float64(r.tms)/1000, r.slotVal, r.chunk, r.size, r.id64)
			break
		}
	}

	// ── PHASE 5 : corrélation kill feed (chunk_27). Combien de records de dégât
	// tombent dans une fenêtre ±N ms d'un kill, et quelle famille pour les kills narrés ?
	fmt.Println("\n=== PHASE 5 : corrélation kill feed (chunk_27) ===")
	feed := killFeed()
	fmt.Printf("  %d kills appariés ; %d records de dégât ARME\n", len(feed), len(armRecs))
	// pour chaque kill, le record de dégât dont le ts est le plus proche (±300ms)
	var matched int
	for _, k := range feed {
		best, bd := -1, 1<<30
		for i := range armRecs {
			dt := armRecs[i].tms - k.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 && bd <= 300 {
			matched++
		}
	}
	fmt.Printf("  kills avec un record de dégât ARME à ±300ms : %d/%d\n", matched, len(feed))

	// kills narrés à valider
	fmt.Println("  -- kills narrés : familles des records de dégât dans ±1000ms --")
	for _, nr := range []struct{ killer, victim, want string }{
		{"IKE ILYA", "JGtm", "Hammer/marteau"},
		{"JGtm", "Akatsuki fire17", "BR75"},
	} {
		for _, k := range feed {
			if nameOf(k.killer) != nr.killer || nameOf(k.victim) != nr.victim {
				continue
			}
			fams := map[string]int{}
			for _, r := range armRecs {
				if r.tms >= k.t-1000 && r.tms <= k.t+1000 {
					fams[r.fam]++
				}
			}
			fmt.Printf("    t=%dms %s->%s [attendu %s] : %v\n", k.t, nr.killer, nr.victim, nr.want, fams)
		}
	}
}

// ── kill feed (chunk_27) : KILL@t apparié à DEATH@t adjacent ──
var xuidGamertag = map[uint64]string{
	2535467794760703: "whiteknight2519",
	2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm",
	2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA",
	2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus",
	2533274826120416: "VitaminA1688",
}

func nameOf(x uint64) string {
	if g, ok := xuidGamertag[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

type kfRow struct {
	killer, victim uint64
	t              int
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
	var feed []kfRow
	usedD := make([]bool, len(deaths))
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.xuid == k.xuid {
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
			usedD[best] = true
			feed = append(feed, kfRow{k.xuid, deaths[best].xuid, k.t})
		}
	}
	return feed
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}
