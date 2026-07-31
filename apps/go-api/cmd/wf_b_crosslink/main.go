package main

// wf_b_crosslink — Angle B : cross-link fire-events <-> keyframes.
//
// Objectif : résoudre record(keyframe) -> pi(fire-event) -> xuid via accord
// arme+temps, agrégé sur toutes les keyframes type-2. Valider sur le ground
// truth (R0=whiteknight, R1=Javier, R7=VitaminA) puis lire l'arme du record de
// JGtm à la keyframe proche de 355s (Frag Parfait BR75).

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

// ───────────────────────── roster ─────────────────────────

type rosterEntry struct {
	xuid  uint64
	label string
}

var roster = []rosterEntry{
	{2533274980284321, "p_980284321"},
	{2533274815845110, "p_815845110"},
	{2533274882097883, "p_882097883"},
	{2535437947245250, "p_947245250"},
	{2535467794760703, "p_794760703"},
	{2535444178793711, "p_793711"},
	{2533274826120416, "p_826120416"},
	{2533274823110022, "JGtm"},
}

// ───────────────────────── bit/inflate helpers ─────────────────────────

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

func extractPacket(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}

func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

// weaponHit — un littéral high-32 d'arme trouvé à une position bit.
type weaponHit struct {
	pos  int
	high uint32
	name string
}

// scanWeapons : positions bit de chaque littéral high-32 d'arme (famille).
func scanWeapons(d []byte) []weaponHit {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h := uint32(id >> 32)
		// fold variante -> tête de famille pour comparaison canonique.
		if canon, ok := analysis.WeaponFusionMap[n]; ok {
			n = canon
		}
		h2n[h] = n
	}
	var out []weaponHit
	tot := len(d) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		h := uint32(rb(d, bp, 32))
		if n, ok := h2n[h]; ok {
			out = append(out, weaponHit{bp, h, n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	return out
}

// ───────────────────────── record extraction ─────────────────────────

// record — un slot joueur dans une keyframe : ancre bit + armes tenues.
type record struct {
	anchor  int
	weapons []weaponHit
}

const (
	recordGapBits = 1500 // saut min entre records (anchors ~2600-2800 bits)
	pairGapBits   = 400  // 2 armes d'un même record ~195 bits l'une de l'autre
)

// clusterRecords regroupe les littéraux armes en records : un nouveau record
// démarre quand l'écart au précédent littéral dépasse recordGapBits.
func clusterRecords(hits []weaponHit) []record {
	var recs []record
	if len(hits) == 0 {
		return recs
	}
	cur := record{anchor: hits[0].pos, weapons: []weaponHit{hits[0]}}
	for i := 1; i < len(hits); i++ {
		if hits[i].pos-hits[i-1].pos > recordGapBits {
			recs = append(recs, cur)
			cur = record{anchor: hits[i].pos, weapons: []weaponHit{hits[i]}}
			continue
		}
		cur.weapons = append(cur.weapons, hits[i])
	}
	recs = append(recs, cur)
	return recs
}

// ───────────────────────── fire timeline ─────────────────────────

// fireShot — un tir résolu (pi, arme canonique, temps ms).
type fireShot struct {
	pi     int
	name   string // nom canonique (fold)
	timeMS float64
}

func canonName(weaponName string) string {
	if c, ok := analysis.WeaponFusionMap[weaponName]; ok {
		return c
	}
	return weaponName
}

// ───────────────────────── main ─────────────────────────

type chunkBlob struct {
	idx     int
	data    []byte
	startMS int
}

func main() {
	// 1. charger tous les chunks décompressés.
	var blobs []chunkBlob
	for i := 0; i <= 27; i++ {
		p := fmt.Sprintf("%s/chunk_%02d.bin", cache, i)
		d := inflate(p)
		if len(d) == 0 {
			continue
		}
		blobs = append(blobs, chunkBlob{idx: i, data: d, startMS: (i - 2) * 20000})
	}

	// 2. résoudre xuid -> pi.
	// IMPORTANT : chunk_00 et chunk_27 sont DÉGÉNÉRÉS (les 8 xuids y figurent UNE
	// fois, tous avec pi=0 — table de roster brute, pas le flux pi). ResolveBest
	// "premier chunk gagnant" verrouille alors pi=0 pour tout le monde. On résout
	// donc sur les chunks GAMEPLAY (01..26), où les 5 bits avant le xuid portent un
	// pi STABLE et distinct (vérifié wf_b_xuidcheck : mapping identique 26 chunks).
	rxuids := make([]uint64, len(roster))
	for i, r := range roster {
		rxuids[i] = r.xuid
	}
	gameplayDatas := make([][]byte, 0, len(blobs))
	for _, b := range blobs {
		if b.idx >= 1 && b.idx <= 26 {
			gameplayDatas = append(gameplayDatas, b.data)
		}
	}
	xuidToPI := weaponv3.ResolveBest(rxuids, gameplayDatas)
	piToLabel := map[int]string{}
	piToXuid := map[int]uint64{}
	for _, r := range roster {
		if pi, ok := xuidToPI[r.xuid]; ok {
			piToLabel[pi] = r.label
			piToXuid[pi] = r.xuid
		}
	}
	fmt.Println("=== xuid -> pi ===")
	for _, r := range roster {
		if pi, ok := xuidToPI[r.xuid]; ok {
			fmt.Printf("  %-14s xuid=%d pi=%d\n", r.label, r.xuid, pi)
		} else {
			fmt.Printf("  %-14s xuid=%d pi=NOT FOUND\n", r.label, r.xuid)
		}
	}

	// 3. timeline de tirs par pi (toutes keyframes type-2, layout 4-bit + relax3).
	var shots []fireShot
	for _, b := range blobs {
		// les fire-events vivent dans le payload type-2 (gameplay). On scanne le
		// chunk entier comme le backfill v2 (USEstimator gère les frames).
		est := weaponv3.USEstimator(b.data, b.startMS)
		fires := weaponv3.ScanFireEventsV3(b.data, est, weaponv3.FirePi4High, true)
		for _, f := range fires {
			shots = append(shots, fireShot{pi: f.PlayerIndex, name: canonName(f.WeaponName), timeMS: f.TimestampMS})
		}
	}
	sort.Slice(shots, func(i, j int) bool { return shots[i].timeMS < shots[j].timeMS })
	fmt.Printf("\n=== %d fire-shots scannés ===\n", len(shots))
	// distribution par pi
	piCount := map[int]int{}
	for _, s := range shots {
		piCount[s.pi]++
	}
	pis := []int{}
	for pi := range piCount {
		pis = append(pis, pi)
	}
	sort.Ints(pis)
	for _, pi := range pis {
		lbl := piToLabel[pi]
		fmt.Printf("  pi=%2d shots=%4d  (%s)\n", pi, piCount[pi], lbl)
	}

	// 4. extraire les records de chaque keyframe type-2.
	type kf struct {
		idx     int
		startMS int
		recs    []record
	}
	var kfs []kf
	for _, b := range blobs {
		payload := extractPacket(b.data, 2)
		if payload == nil {
			continue
		}
		hits := scanWeapons(payload)
		recs := clusterRecords(hits)
		kfs = append(kfs, kf{idx: b.idx, startMS: b.startMS, recs: recs})
	}
	fmt.Printf("\n=== keyframes type-2 : %d ===\n", len(kfs))
	for _, k := range kfs {
		fmt.Printf("  chunk_%02d t=%5.1fs  records=%d\n", k.idx, float64(k.startMS)/1000.0, len(k.recs))
	}

	// 5. matrice d'accord record(index dans chunk_02) x pi.
	// On ancre les records sur chunk_02 (8 records = ground truth verrouillé).
	// Pour les keyframes suivantes, on associe le record positionnel i au record
	// i de chunk_02 (les 8 premiers anchors restent stables ; les records >8 sont
	// des armes au sol -> ignorés pour le vote record-fixe).
	var anchorKF *kf
	for i := range kfs {
		if kfs[i].idx == 2 {
			anchorKF = &kfs[i]
			break
		}
	}
	if anchorKF == nil {
		fmt.Println("!! chunk_02 introuvable")
		return
	}
	nRec := len(anchorKF.recs)
	if nRec > 8 {
		nRec = 8
	}
	fmt.Printf("\n=== chunk_02 : %d records (ancrage) ===\n", len(anchorKF.recs))
	for i, r := range anchorKF.recs {
		names := []string{}
		for _, w := range r.weapons {
			names = append(names, w.name)
		}
		fmt.Printf("  R%d anchor=%d weapons=%v\n", i, r.anchor, names)
	}

	// fenêtre temporelle pour matcher arme keyframe<->tir.
	const matchWindowMS = 12000.0

	// agree[recIdx][pi] = nb de votes.
	agree := make(map[int]map[int]int)
	for i := 0; i < nRec; i++ {
		agree[i] = map[int]int{}
	}
	for _, k := range kfs {
		T := float64(k.startMS)
		for ri := 0; ri < nRec && ri < len(k.recs); ri++ {
			rec := k.recs[ri]
			// pour chaque arme du record, vote pour le pi qui a tiré cette arme
			// le plus proche de T (dans la fenêtre).
			for _, w := range rec.weapons {
				best := -1
				bestDelta := matchWindowMS + 1
				for _, s := range shots {
					if s.name != w.name {
						continue
					}
					d := s.timeMS - T
					if d < 0 {
						d = -d
					}
					if d <= matchWindowMS && d < bestDelta {
						best, bestDelta = s.pi, d
					}
				}
				if best >= 0 {
					agree[ri][best]++
				}
			}
		}
	}

	fmt.Println("\n=== matrice accord record x pi (votes) ===")
	hdr := "        "
	for _, pi := range pis {
		hdr += fmt.Sprintf("pi%-3d ", pi)
	}
	fmt.Println(hdr)
	for ri := 0; ri < nRec; ri++ {
		line := fmt.Sprintf("  R%d   ", ri)
		for _, pi := range pis {
			line += fmt.Sprintf("%4d  ", agree[ri][pi])
		}
		fmt.Println(line)
	}

	// 6. affectation greedy max-accord (record -> pi), 1:1.
	type cell struct {
		ri, pi, v int
	}
	var cells []cell
	for ri := 0; ri < nRec; ri++ {
		for _, pi := range pis {
			if agree[ri][pi] > 0 {
				cells = append(cells, cell{ri, pi, agree[ri][pi]})
			}
		}
	}
	sort.Slice(cells, func(i, j int) bool { return cells[i].v > cells[j].v })
	recAssigned := map[int]int{} // ri -> pi
	piUsed := map[int]bool{}
	for _, c := range cells {
		if _, ok := recAssigned[c.ri]; ok {
			continue
		}
		if piUsed[c.pi] {
			continue
		}
		recAssigned[c.ri] = c.pi
		piUsed[c.pi] = true
	}

	fmt.Println("\n=== affectation record -> pi -> xuid/label ===")
	gt := map[int]string{0: "whiteknight (R0)", 1: "Javier (R1)", 7: "VitaminA (R7)"}
	for ri := 0; ri < nRec; ri++ {
		pi, ok := recAssigned[ri]
		c02names := []string{}
		for _, w := range anchorKF.recs[ri].weapons {
			c02names = append(c02names, w.name)
		}
		if !ok {
			fmt.Printf("  R%d %-30v -> (non affecté)\n", ri, c02names)
			continue
		}
		lbl := piToLabel[pi]
		gtCheck := ""
		if exp, has := gt[ri]; has {
			gtCheck = " | GT=" + exp
		}
		fmt.Printf("  R%d %-30v -> pi=%d xuid=%d %s votes=%d%s\n",
			ri, c02names, pi, piToXuid[pi], lbl, agree[ri][pi], gtCheck)
	}

	// 7. JGtm : son pi, son record, son arme près de 355s.
	jgtmPI, hasJ := xuidToPI[2533274823110022]
	fmt.Printf("\n=== JGtm (xuid 2533274823110022) ===\n")
	if !hasJ {
		fmt.Println("  pi non résolu -> impossible")
		return
	}
	fmt.Printf("  JGtm pi=%d\n", jgtmPI)
	jgtmRec := -1
	for ri, pi := range recAssigned {
		if pi == jgtmPI {
			jgtmRec = ri
		}
	}
	fmt.Printf("  record affecté à JGtm : R%d (votes=%d)\n", jgtmRec, func() int {
		if jgtmRec >= 0 {
			return agree[jgtmRec][jgtmPI]
		}
		return 0
	}())

	// keyframe la plus proche de 355s.
	target := 355000.0
	var nearKF *kf
	bestD := 1e18
	for i := range kfs {
		d := float64(kfs[i].startMS) - target
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, nearKF = d, &kfs[i]
		}
	}
	fmt.Printf("\n=== keyframe proche de 355s : chunk_%02d t=%.1fs ===\n", nearKF.idx, float64(nearKF.startMS)/1000.0)
	if jgtmRec >= 0 && jgtmRec < len(nearKF.recs) {
		names := []string{}
		for _, w := range nearKF.recs[jgtmRec].weapons {
			names = append(names, w.name)
		}
		fmt.Printf("  R%d (JGtm) tient : %v\n", jgtmRec, names)
	} else {
		fmt.Printf("  record R%d absent de cette keyframe (records=%d)\n", jgtmRec, len(nearKF.recs))
	}

	// Aussi : tous les records de la keyframe proche de 355s + recherche BR75.
	fmt.Printf("\n=== tous records de chunk_%02d (355s) ===\n", nearKF.idx)
	for ri, rec := range nearKF.recs {
		names := []string{}
		hasBR := false
		for _, w := range rec.weapons {
			names = append(names, w.name)
			if w.name == "BR75" {
				hasBR = true
			}
		}
		owner := ""
		for r, pi := range recAssigned {
			if r == ri {
				owner = fmt.Sprintf(" owner=pi%d %s", pi, piToLabel[pi])
			}
		}
		mark := ""
		if hasBR {
			mark = "  <-- BR75"
		}
		fmt.Printf("  R%d %v%s%s\n", ri, names, owner, mark)
	}

	// chunk_19 (340s) — contient des BR75 selon le brief.
	for _, k := range kfs {
		if k.idx == 19 {
			fmt.Printf("\n=== chunk_19 (340s) records ===\n")
			for ri, rec := range k.recs {
				names := []string{}
				for _, w := range rec.weapons {
					names = append(names, w.name)
				}
				fmt.Printf("  R%d %v\n", ri, names)
			}
		}
	}
}
