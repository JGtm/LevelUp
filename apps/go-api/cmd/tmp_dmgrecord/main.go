// tmp_dmgrecord — MISSION E2 : LOCALISER le record de dégât dans le flux film 000d5950.
//
// L1/L2 ont établi : le record de dégât (deser FUN_14080c1f8, struct RAM 0x328 o) porte
// variant_name@+0x14 (R32 BE = high32 FAMILLE d'arme), slot/cause@+0x08, 'weap'@+0x18 (id
// RÉSOLU, pas un FourCC littéral). Il n'est PAS dans le flux d'entités type-0 (prouvé négatif).
// L2 conclut : "racine de message" hors type-0, framing inconnu.
//
// Ce probe ABANDONNE le RE statique (bloqué) et fait de l'EXPLORATION EMPIRIQUE du conteneur :
//   - PHASE A : cartographie fine des packet-types (count exact, appariement, b2/b3, ts).
//     Découverte clé du run packettypes : 30418 type-10 (10 o) APPARIÉS 1:1 aux 30418 type-0.
//   - PHASE B : dump du contenu des type-10 (80 bits) et corrélation temporelle aux 93 kills.
//   - PHASE C : pour chaque kill (ts connu via chunk_27), repérer le(s) packet(s) dont le ts
//     encadre le kill, et chercher le variant_name (high32 famille) R32-BE aligné bit OU octet
//     dans CE packet précis (pas tout le flux). Si le record de dégât est une racine de message
//     dans un petit packet par-frame, le high32 famille y apparaîtra à un offset ~stable.
//
// Usage : tmp_dmgrecord [phase]   (phase = A|B|C|D|all, défaut all)
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
const weapFourCC = uint32(0x77656170) // 'weap'

var h32set = map[uint32]bool{}
var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h := uint32(id >> 32)
		h32set[h] = true
		h32name[h] = n
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
	ts       uint64
	payload  []byte
	chunkIdx int
	seq      int // index dans le chunk
}

func listPackets(d []byte, chunkIdx int) []pkt {
	var out []pkt
	off := 0
	seq := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		b2, b3 := d[off+2], d[off+3]
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, b2, b3, ts, d[off+16 : off+16+size], chunkIdx, seq})
		off += 16 + size
		seq++
	}
	return out
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

func tsToMs(ts uint64) int { return int((ts - t0Us) / 1000) }

// ── kill feed de référence (chunk_27) ──
type kfRow struct {
	killer, victim uint64
	t              int
}

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

func killFeed() []kfRow {
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
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

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }

// allPackets retourne tous les packets de tous les chunks, dans l'ordre.
func allPackets() []pkt {
	var all []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		if len(d) == 0 {
			continue
		}
		all = append(all, listPackets(d, n)...)
	}
	return all
}

func main() {
	phase := "all"
	if len(os.Args) >= 2 {
		phase = os.Args[1]
	}
	build()
	fmt.Printf("=== catalogue : %d id64, %d high32 familles ===\n", len(id64name), len(h32set))
	all := allPackets()
	feed := killFeed()
	fmt.Printf("=== %d packets total, %d kills appariés ===\n\n", len(all), len(feed))

	if phase == "all" || phase == "A" {
		phaseA(all)
	}
	if phase == "all" || phase == "B" {
		phaseB(all, feed)
	}
	if phase == "all" || phase == "C" {
		phaseC(all, feed)
	}
	if phase == "all" || phase == "D" {
		phaseD(feed)
	}
	if phase == "all" || phase == "E" {
		phaseE(all, feed)
	}
}

// ── PHASE E : test de concentration STRICTE. Pour chaque kill, on prend la frame
// type-0 dont le ts est le plus proche (±100ms) et on liste TOUTES les familles
// high32 trouvées dedans (signature, sans walk). Question : la frame du kill
// contient-elle UNE famille dominante = l'arme du tueur ? Et les 2 kills narrés
// matchent-ils (marteau IKE->JGtm = high32 0x841ac5e5 ; BR75 JGtm->Akatsuki) ?
func phaseE(all []pkt, feed []kfRow) {
	const hammerHigh32 = uint32(0x841ac5e5)
	var br75High32 uint32
	for id, n := range analysis.WeaponIDToName {
		if bytes.Contains([]byte(n), []byte("BR75")) {
			br75High32 = uint32(id >> 32)
		}
	}
	fmt.Println("=== PHASE E : familles high32 dans la frame type-0 du kill (±100ms) ===")
	fmt.Printf("  (hammer high32=0x%08x  BR75 high32=0x%08x)\n", hammerHigh32, br75High32)

	// indexe les frames type-0 par ts
	var t0 []pkt
	for _, p := range all {
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	sort.Slice(t0, func(i, j int) bool { return t0[i].ts < t0[j].ts })

	// stat globale : combien de kills ont une famille UNIQUE dans leur frame ±100ms ?
	var nUnique, nAny int
	for _, k := range feed {
		fams := famsInWindow(t0, k.t, 100)
		if len(fams) >= 1 {
			nAny++
		}
		if len(fams) == 1 {
			nUnique++
		}
	}
	fmt.Printf("  kills avec ≥1 famille dans frame ±100ms : %d/%d ; famille UNIQUE : %d/%d\n",
		nAny, len(feed), nUnique, len(feed))

	// focus : les 2 kills narrés
	fmt.Println("  -- focus kills narrés (familles dans frame ±100ms, ±300ms, ±1000ms) --")
	narr := []struct {
		killer, victim string
		want           string
	}{
		{"IKE ILYA", "JGtm", "Hammer"},
		{"JGtm", "Akatsuki fire17", "BR75"},
	}
	for _, nr := range narr {
		for _, k := range feed {
			if nameOf(k.killer) != nr.killer || nameOf(k.victim) != nr.victim {
				continue
			}
			fmt.Printf("    %dms %s->%s [attendu %s]\n", k.t, nr.killer, nr.victim, nr.want)
			for _, win := range []int{100, 300, 1000} {
				fams := famsInWindow(t0, k.t, win)
				fmt.Printf("        ±%dms : %v\n", win, fams)
			}
		}
	}
}

// famsInWindow : familles high32 distinctes trouvées (signature) dans toutes les
// frames type-0 dont le ts est dans [t-win, t+win] ms.
func famsInWindow(t0 []pkt, t, win int) map[string]int {
	out := map[string]int{}
	for _, p := range t0 {
		pms := tsToMs(p.ts)
		if pms < t-win || pms > t+win {
			continue
		}
		d := p.payload
		mx := len(d)*8 - 32
		for bp := 0; bp <= mx; bp++ {
			h := uint32(bitsAt(d, bp, 32))
			if h32set[h] {
				out[h32name[h]]++
			}
		}
	}
	return out
}

// ── PHASE D : décode chaque frame type-0, mesure le recEnd, et localise les high32
// famille par rapport à la fin de la boucle d'entités. Si le record de dégât est
// entrelacé dans le type-0 (après le recEnd), les high32 famille du kill tomberont
// dans le TAIL (bits après recEnd). Si dans la zone entités, ils tomberont avant.
func phaseD(feed []kfRow) {
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		fmt.Println("registry parse error:", err)
		return
	}
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

	// stats globales sur tous les type-0 : clean vs desync, bits totaux, bits de tail.
	var nFrames, nClean int
	var totalTailBits int64
	// pour les frames clean : combien de high32 famille AVANT vs APRÈS le recEnd ?
	var h32BeforeEnd, h32AfterEnd int

	type frameInfo struct {
		ts        uint64
		payload   []byte
		clean     bool
		endBit    int // bit du recEnd (fin boucle entités)
		totalBits int
	}
	var frames []frameInfo

	for n := 2; n <= 26; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d, n) {
			if p.typ != 0 {
				continue
			}
			nFrames++
			// world frais par frame (les deltas dépendent du world ; sans persistance
			// inter-frame le delta desync vite, mais on mesure le recEnd des frames simples).
			w := filmdec.NewWorld(reg)
			loadWorldDump(w)
			br := filmdec.NewBitReader(p.payload)
			_, derr := filmdec.DecodeFrameRecords(br, w, cfg)
			clean := derr == nil
			endBit := br.BitPos()
			total := len(p.payload) * 8
			if clean {
				nClean++
				tail := total - endBit
				totalTailBits += int64(tail)
				// high32 famille avant/après recEnd
				for bp := 0; bp+32 <= total; bp++ {
					h := uint32(bitsAt(p.payload, bp, 32))
					if !h32set[h] {
						continue
					}
					if bp < endBit {
						h32BeforeEnd++
					} else {
						h32AfterEnd++
					}
				}
			}
			frames = append(frames, frameInfo{p.ts, p.payload, clean, endBit, total})
		}
	}
	fmt.Println("=== PHASE D : décode type-0, recEnd, position des high32 famille ===")
	fmt.Printf("  frames type-0 : %d total, %d clean (%.0f%%)\n", nFrames, nClean, pct(nClean, nFrames))
	if nClean > 0 {
		fmt.Printf("  tail moyen (bits après recEnd, frames clean) : %.1f bits\n", float64(totalTailBits)/float64(nClean))
	}
	fmt.Printf("  high32 famille dans frames clean : %d AVANT recEnd, %d APRÈS recEnd (tail)\n", h32BeforeEnd, h32AfterEnd)

	// Pour chaque kill : la frame la plus proche, son statut, et les high32 dans le tail.
	sort.Slice(frames, func(i, j int) bool { return frames[i].ts < frames[j].ts })
	fmt.Println("\n  -- pour 12 premiers kills : frame la plus proche, clean?, high32 dans tail --")
	for ki, k := range feed {
		if ki >= 12 {
			break
		}
		best := -1
		bd := 1 << 30
		for i := range frames {
			dt := tsToMs(frames[i].ts) - k.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd = dt
				best = i
			}
		}
		if best < 0 {
			continue
		}
		f := frames[best]
		tailFams := []string{}
		if f.clean {
			for bp := f.endBit; bp+32 <= f.totalBits; bp++ {
				h := uint32(bitsAt(f.payload, bp, 32))
				if h32set[h] {
					tailFams = append(tailFams, fmt.Sprintf("%s@bit%d(+%d)", h32name[h], bp, bp-f.endBit))
				}
			}
		}
		fmt.Printf("    kill %dms %s->%s | frameΔ%dms clean=%v endBit=%d/%d tailHigh32=%v\n",
			k.t, nameOf(k.killer), nameOf(k.victim), bd, f.clean, f.endBit, f.totalBits, tailFams)
	}
}

func loadWorldDump(w *filmdec.World) {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
}

// ── PHASE A : cartographie + appariement type-10 <-> type-0 ──
func phaseA(all []pkt) {
	fmt.Println("=== PHASE A : appariement type-10 / type-0 + b2/b3 distincts ===")
	// b2/b3 sur les type-10 : un event-type encodé ?
	b2b3 := map[uint16]int{}
	var n10, n0 int
	for _, p := range all {
		if p.typ == 10 {
			n10++
			b2b3[uint16(p.b2)|uint16(p.b3)<<8]++
		}
		if p.typ == 0 {
			n0++
		}
	}
	fmt.Printf("  type-10=%d  type-0=%d\n", n10, n0)
	fmt.Printf("  type-10 tailles distinctes : ")
	szset := map[int]int{}
	for _, p := range all {
		if p.typ == 10 {
			szset[len(p.payload)]++
		}
	}
	for s, c := range szset {
		fmt.Printf("%do×%d ", s, c)
	}
	fmt.Println()
	fmt.Printf("  type-10 b2|b3 distincts (%d valeurs) top : ", len(b2b3))
	type kv struct {
		k uint16
		c int
	}
	var arr []kv
	for k, c := range b2b3 {
		arr = append(arr, kv{k, c})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].c > arr[b].c })
	for i := 0; i < len(arr) && i < 12; i++ {
		fmt.Printf("%04x:%d ", arr[i].k, arr[i].c)
	}
	fmt.Println()

	// les type-10 NON-zéro b2/b3 sont-ils rares (= events) ? dump des 30 premiers non-nuls.
	fmt.Println("  -- type-10 avec b2/b3 != 0 (candidats events), 30 premiers --")
	cnt := 0
	for _, p := range all {
		if p.typ != 10 || (p.b2 == 0 && p.b3 == 0) {
			continue
		}
		fmt.Printf("    chunk%02d seq%-4d ts=%dms b2=%02x b3=%02x payload=% x\n",
			p.chunkIdx, p.seq, tsToMs(p.ts), p.b2, p.b3, p.payload)
		cnt++
		if cnt >= 30 {
			break
		}
	}
	fmt.Println()
}

// ── PHASE B : dump type-10 + corrélation aux kills ──
func phaseB(all []pkt, feed []kfRow) {
	fmt.Println("=== PHASE B : type-10 (80 bits) — distinct payloads + corrélation kills ===")
	// les payloads de type-10 sont-ils tous identiques (sync) ou variables (events) ?
	distinct := map[string]int{}
	for _, p := range all {
		if p.typ == 10 {
			distinct[string(p.payload)]++
		}
	}
	fmt.Printf("  %d payloads type-10 distincts sur %d\n", len(distinct), countType(all, 10))
	// montrer les payloads les plus fréquents
	type kv struct {
		k string
		c int
	}
	var arr []kv
	for k, c := range distinct {
		arr = append(arr, kv{k, c})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].c > arr[b].c })
	for i := 0; i < len(arr) && i < 8; i++ {
		fmt.Printf("    % x : ×%d\n", []byte(arr[i].k), arr[i].c)
	}

	// corrélation : pour chaque kill, le type-10 dont le ts est le plus proche — son payload diffère-t-il ?
	fmt.Println("  -- type-10 le plus proche de chaque kill (10 premiers kills) --")
	var t10 []pkt
	for _, p := range all {
		if p.typ == 10 {
			t10 = append(t10, p)
		}
	}
	sort.Slice(t10, func(i, j int) bool { return t10[i].ts < t10[j].ts })
	for i, k := range feed {
		if i >= 10 {
			break
		}
		best := -1
		bd := 1 << 30
		for j := range t10 {
			dt := tsToMs(t10[j].ts) - k.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd = dt
				best = j
			}
		}
		if best >= 0 {
			fmt.Printf("    kill t=%dms %s->%s | t10 Δ%dms b2b3=%02x%02x payload=% x\n",
				k.t, nameOf(k.killer), nameOf(k.victim), bd, t10[best].b2, t10[best].b3, t10[best].payload)
		}
	}
	fmt.Println()
}

// ── PHASE C : recherche du variant_name famille dans les packets encadrant chaque kill ──
func phaseC(all []pkt, feed []kfRow) {
	fmt.Println("=== PHASE C : high32 famille (variant_name) dans les packets autour de chaque kill ===")
	// Pour chaque kill, on prend TOUS les packets dont ts ∈ [t-1500, t+500]ms et on scanne
	// le high32 famille (bit-aligné BE) + 'weap' FourCC dans CHACUN. On regarde la concentration.
	hitByType := map[uint16]int{} // dans quel packet-type le high32 famille apparaît le plus
	var killsWithHit int
	for _, k := range feed {
		found := false
		for _, p := range all {
			pms := tsToMs(p.ts)
			if pms < k.t-1500 || pms > k.t+500 {
				continue
			}
			// scan high32 famille bit-aligné (coûteux mais petits packets)
			d := p.payload
			if len(d) > 4000 {
				continue // skip gros packets (keyframe/registre)
			}
			mx := len(d)*8 - 32
			for bp := 0; bp <= mx; bp++ {
				h := uint32(bitsAt(d, bp, 32))
				if h32set[h] {
					hitByType[p.typ]++
					found = true
					break
				}
			}
		}
		if found {
			killsWithHit++
		}
	}
	fmt.Printf("  kills avec ≥1 high32 famille dans un petit packet (<4Ko) en [-1.5s,+0.5s] : %d/%d\n", killsWithHit, len(feed))
	fmt.Printf("  répartition par packet-type : ")
	for t, c := range hitByType {
		fmt.Printf("type%d:%d ", t, c)
	}
	fmt.Println()

	// 'weap' FourCC dans les petits packets autour des kills (toutes orientations)
	weapHits := 0
	for _, p := range all {
		if len(p.payload) > 4000 || p.typ == 0 {
			continue
		}
		for i := 0; i+4 <= len(p.payload); i++ {
			le := binary.LittleEndian.Uint32(p.payload[i:])
			be := binary.BigEndian.Uint32(p.payload[i:])
			if le == weapFourCC || be == weapFourCC {
				weapHits++
			}
		}
	}
	fmt.Printf("  'weap' FourCC dans petits packets non-type0 (byte-aligné) : %d\n", weapHits)
	fmt.Println()
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func countType(all []pkt, t uint16) int {
	n := 0
	for _, p := range all {
		if p.typ == t {
			n++
		}
	}
	return n
}
