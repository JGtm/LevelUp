// probe_pi_reconcile — FRONT A : réconciliation des 3 player_index sur l'oracle
// 000d5950 (Slayer, Frag Parfait Sidekick de JGtm xuid 2533274823110022).
//
// Teste l'hypothèse de DÉSALIGNEMENT entre :
//   - acurtis pi    : ResolveXuidToPI (xuid bit-searché, 5 bits avant) — weaponv3.
//   - fire-pi b5>>4 : le tireur du fire-event type-2 (4 bits, ou variantes 5 bits).
//   - slot36 type-3 : b36 de l'acteur du highlight-event (footer chunk-27).
//
// Sortie : pi de chaque joueur (acurtis), fire-pi des tirs Sidekick proches des
// kills connus de JGtm, et le slot36 footer par xuid. Aucune écriture DB.
package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	cacheDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`
	sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	matchPfx = "000d5950"

	jgtmXUID = uint64(2533274823110022) // JGtm — tueur du Frag Parfait

	// Sidekick : high32 0xf408190f. Fire weapon = high32<<32 | suffixe 0x42c9679f.
	sidekickHigh32   = uint32(0xf408190f)
	commonSuffix     = uint32(0x42c9679f)
	sidekickFireID   = uint64(sidekickHigh32)<<32 | uint64(commonSuffix)
	nearKillWindowMS = 1500 // fenêtre |fire - kill| pour « proche du Frag Parfait »
)

// Kills connus du Frag Parfait (ms), fournis par l'oracle.
var fragKillsMS = []int{104995, 112869, 149471, 192884, 230628, 258590}

type chunkRec struct {
	index     int
	chunkType int
	startMS   int
	data      []byte // décompressé
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "probe: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	db, release, err := duckdb.OpenReadForQuery(sharedDB)
	if err != nil {
		return fmt.Errorf("open RO: %w", err)
	}
	defer release()

	full, err := resolveFull(ctx, db, matchPfx)
	if err != nil {
		return err
	}
	fmt.Printf("match_id complet : %s\n\n", full)

	roster, err := loadRoster(ctx, db, full)
	if err != nil {
		return err
	}

	chunks, err := loadChunks()
	if err != nil {
		return err
	}

	// ---------- 1) acurtis pi par joueur ----------
	type2 := type2Datas(chunks)
	xuids := make([]uint64, 0, len(roster))
	for x := range roster {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	acurtis := weaponv3.ResolveBest(xuids, type2)

	fmt.Println("=== 1) acurtis pi (ResolveXuidToPI / ResolveBest, type-2) ===")
	fmt.Printf("%-22s %-16s %-6s\n", "xuid", "gamertag", "pi")
	for _, x := range xuids {
		pi := "ABSENT"
		if v, ok := acurtis[x]; ok {
			pi = strconv.Itoa(v)
		}
		mark := ""
		if x == jgtmXUID {
			mark = "  <-- JGtm (tueur Frag Parfait)"
		}
		fmt.Printf("%-22d %-16s %-6s%s\n", x, roster[x], pi, mark)
	}
	jgtmAcurtis, jgtmHas := acurtis[jgtmXUID]
	fmt.Println()

	// ---------- 2) fire-pi des tirs Sidekick proches des kills ----------
	fmt.Println("=== 2) fire-events Sidekick (high32 0xf408190f) proches des Frag-kills ===")
	for _, layout := range allLayouts() {
		scanSidekickFires(chunks, layout, roster, jgtmAcurtis, jgtmHas)
	}

	// Vue d'ensemble : distribution des fire-pi de TOUS les Sidekick du match.
	fmt.Println("=== 2-bis) distribution fire-pi de TOUS les fire Sidekick du match (par layout) ===")
	for _, layout := range allLayouts() {
		distSidekick(chunks, layout)
	}

	// ---------- 3) slot36 footer (type-3) par xuid ----------
	fmt.Println("=== 3) slot36 footer type-3 (b36 highlight-events) par xuid ===")
	footerB36(chunks, roster)

	// ---------- 5) mapping acurtis-pi <-> fire-pi via kills par joueur ----------
	fmt.Println("=== 5) MAPPING acurtis-pi <-> fire-pi (corrélation kills->fires, layout 4high) ===")
	kvp, err := loadKVP(ctx, db, full)
	if err != nil {
		return err
	}
	buildMapping(chunks, kvp, roster, acurtis)

	// ---------- 7) qui tue à chaque Frag-time + tous les Sidekick fires à pi=2 ----------
	fmt.Println("=== 7) killer à chaque Frag-kill-time (killer_victim_pairs) ===")
	for _, km := range fragKillsMS {
		printKillerAt(kvp, roster, km)
	}
	fmt.Println()

	fmt.Println("=== 7-bis) TOUS les fire Sidekick (4high) avec leur fire-pi + temps ===")
	for _, f := range allFires(chunks, weaponv3.FirePi4High) {
		if isSidekick(f.WeaponBytes) {
			fmt.Printf("  t=%-7d fire-pi=%-2d slot=%d b5=0x%02x\n", int(f.TimestampMS), f.PlayerIndex, f.Slot, f.B5)
		}
	}
	fmt.Println()

	// ---------- 8) tous les fires À pi=2 proches des 6 Frag-kills ----------
	fmt.Println("=== 8) fires (TOUTES armes, 4high) à fire-pi=2 dans ±1500ms des 6 Frag-kills ===")
	allF := allFires(chunks, weaponv3.FirePi4High)
	for _, km := range fragKillsMS {
		fmt.Printf("  -- kill %d --\n", km)
		n := 0
		for _, f := range allF {
			if f.PlayerIndex != 2 {
				continue
			}
			ts := int(f.TimestampMS)
			d := ts - km
			if d < -1500 || d > 1500 {
				continue
			}
			id := binary.BigEndian.Uint64(f.WeaponBytes[:])
			name := analysis.WeaponIDToName[id]
			if name == "" {
				name = fmt.Sprintf("INCONNU(high=0x%08x)", uint32(id>>32))
			}
			fmt.Printf("     t=%-7d Δ=%+5d b5=0x%02x weapon=%s\n", ts, d, f.B5, name)
			n++
		}
		if n == 0 {
			fmt.Println("     (AUCUN fire pi=2 dans la fenêtre)")
		}
	}
	fmt.Println()

	// ---------- 9) recall relâché : Sidekick fires à pi=2 récupérables ? ----------
	fmt.Println("=== 9) scan RELAX3 (marqueur 3-bit) : y a-t-il des Sidekick à fire-pi=2 ? ===")
	relaxSidekickPi2(chunks)

	fmt.Println("=== 6) SYNTHÈSE JGtm ===")
	if jgtmHas {
		fmt.Printf("acurtis pi JGtm = %d ; footer slot36 JGtm = 1 ; fires Sidekick proches kills = fire-pi 6 (4high)\n", jgtmAcurtis)
	} else {
		fmt.Println("acurtis pi JGtm = ABSENT (xuid non bit-trouvé)")
	}
	return nil
}

// killRow — un kill (killer xuid + time_ms) de killer_victim_pairs.
type killRow struct {
	killer uint64
	timeMS int
}

func loadKVP(ctx context.Context, db *sql.DB, matchID string) ([]killRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT killer_xuid, time_ms FROM killer_victim_pairs
		WHERE match_id = ? AND time_ms IS NOT NULL
		ORDER BY time_ms`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadKVP: %w", err)
	}
	defer rows.Close()
	var out []killRow
	for rows.Next() {
		var ks string
		var t int
		if err := rows.Scan(&ks, &t); err != nil {
			continue
		}
		if n, ok := parseXUID(ks); ok {
			out = append(out, killRow{killer: n, timeMS: t})
		}
	}
	return out, rows.Err()
}

// buildMapping : pour chaque joueur, prend ses kills (killer_victim_pairs), trouve
// le fire-event (TOUS armes, layout 4high) le plus proche AVANT le kill (<=1500ms),
// et tally le fire-pi de ce tir. Le fire-pi dominant = le « vrai » fire-pi du joueur.
// On confronte au acurtis pi du joueur => mapping acurtis<->fire.
func buildMapping(chunks []chunkRec, kvp []killRow, roster map[uint64]string, acurtis map[uint64]int) {
	fires := allFires(chunks, weaponv3.FirePi4High)
	sort.Slice(fires, func(i, j int) bool { return fires[i].TimestampMS < fires[j].TimestampMS })

	// killer xuid -> fire-pi -> count
	byKiller := map[uint64]map[int]int{}
	for _, k := range kvp {
		var best *analysis.FireEvent
		for i := range fires {
			ts := int(fires[i].TimestampMS)
			if ts > k.timeMS {
				break
			}
			if k.timeMS-ts > nearKillWindowMS {
				continue
			}
			if best == nil || fires[i].TimestampMS > best.TimestampMS {
				best = &fires[i]
			}
		}
		if best == nil {
			continue
		}
		if byKiller[k.killer] == nil {
			byKiller[k.killer] = map[int]int{}
		}
		byKiller[k.killer][best.PlayerIndex]++
	}

	xuids := make([]uint64, 0, len(roster))
	for x := range roster {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return acurtis[xuids[i]] < acurtis[xuids[j]] })
	fmt.Printf("%-16s %-8s %-8s %-s\n", "gamertag", "acurtis", "fire-pi*", "fire-pi-dist(kills proches)")
	for _, x := range xuids {
		ap := "?"
		if v, ok := acurtis[x]; ok {
			ap = strconv.Itoa(v)
		}
		dom, dist := dominant(byKiller[x])
		fmt.Printf("%-16s %-8s %-8s %-s\n", roster[x], ap, dom, dist)
	}
	fmt.Println("(* fire-pi dominant des tirs précédant les kills du joueur ; si != acurtis => désalignement)")
	fmt.Println()
}

// relaxSidekickPi2 scanne en recall relâché (3-bit) et compte les Sidekick par
// fire-pi, pour voir si un pi=2 Sidekick émerge (sinon : vraie absence du tir).
func relaxSidekickPi2(chunks []chunkRec) {
	tally := map[int]int{}
	var pi2times []int
	for _, c := range chunks {
		if c.chunkType != 2 {
			continue
		}
		est := weaponv3.USEstimator(c.data, c.startMS)
		for _, f := range weaponv3.ScanFireEventsV3(c.data, est, weaponv3.FirePi4High, true) {
			if isSidekick(f.WeaponBytes) {
				tally[f.PlayerIndex]++
				if f.PlayerIndex == 2 {
					pi2times = append(pi2times, int(f.TimestampMS))
				}
			}
		}
	}
	fmt.Printf("  RELAX3 Sidekick total par fire-pi: %s\n", tallyStr(tally))
	if len(pi2times) == 0 {
		fmt.Println("  -> RELAX3 ne récupère AUCUN Sidekick à fire-pi=2 : le tir Sidekick de JGtm est VRAIMENT absent du flux fire-event scanné.")
	} else {
		fmt.Printf("  -> RELAX3 récupère %d Sidekick à fire-pi=2 aux temps %v\n", len(pi2times), pi2times)
	}
	fmt.Println()
}

// printKillerAt imprime le(s) killer(s) trouvés à un time_ms exact (tolérance 0).
func printKillerAt(kvp []killRow, roster map[uint64]string, tms int) {
	found := false
	for _, k := range kvp {
		if k.timeMS == tms {
			fmt.Printf("  t=%-7d killer=%s (xuid %d)\n", tms, roster[k.killer], k.killer)
			found = true
		}
	}
	if !found {
		// proche
		best, bestD := killRow{}, 1<<30
		for _, k := range kvp {
			d := k.timeMS - tms
			if d < 0 {
				d = -d
			}
			if d < bestD {
				best, bestD = k, d
			}
		}
		fmt.Printf("  t=%-7d (exact absent) proche Δ=%d killer=%s (xuid %d)\n", tms, bestD, roster[best.killer], best.killer)
	}
}

func dominant(m map[int]int) (string, string) {
	if len(m) == 0 {
		return "-", "(aucun fire proche)"
	}
	bestPI, bestN := -1, -1
	for pi, n := range m {
		if n > bestN {
			bestPI, bestN = pi, n
		}
	}
	return strconv.Itoa(bestPI), tallyStr(m)
}

// ---- layouts ----

type layoutSpec struct {
	name   string
	layout weaponv3.FirePiLayout
}

func allLayouts() []layoutSpec {
	return []layoutSpec{
		{"4high (v2 b5>>4)", weaponv3.FirePi4High},
		{"5span (b5-1..b5+3)", weaponv3.FirePi5SpanBefore},
		{"5highb5 (b5>>3)", weaponv3.FirePi5HighInB5},
		{"5lowb5 (b5&0x1f)", weaponv3.FirePi5LowInB5},
	}
}

// scanSidekickFires scanne les fires d'un layout, garde les Sidekick proches d'un
// Frag-kill, et imprime leur fire-pi + écart au kill.
func scanSidekickFires(chunks []chunkRec, ls layoutSpec, roster map[uint64]string, jgtmPI int, jgtmHas bool) {
	fires := allFires(chunks, ls.layout)
	fmt.Printf("--- layout %s ---\n", ls.name)
	any := false
	piTally := map[int]int{}
	for _, f := range fires {
		if !isSidekick(f.WeaponBytes) {
			continue
		}
		ms := int(f.TimestampMS)
		k, d := nearestKill(ms)
		if d > nearKillWindowMS {
			continue
		}
		any = true
		piTally[f.PlayerIndex]++
		fmt.Printf("  fire t=%-7d (kill=%-7d Δ=%+5d)  fire-pi=%-3d slot=%d b5=0x%02x fireCounter=%d\n",
			ms, k, ms-k, f.PlayerIndex, f.Slot, f.B5, f.FireCounter)
	}
	if !any {
		fmt.Println("  (aucun fire Sidekick dans une fenêtre kill)")
	} else {
		fmt.Printf("  -> tally fire-pi (proches kills): %s\n", tallyStr(piTally))
		if jgtmHas {
			if _, hit := piTally[jgtmPI]; hit {
				fmt.Printf("  -> MATCH: des tirs Sidekick sont à fire-pi=%d (= acurtis pi JGtm)\n", jgtmPI)
			} else {
				fmt.Printf("  -> MISMATCH: AUCUN tir Sidekick à fire-pi=%d (acurtis pi JGtm) ; tirs ailleurs\n", jgtmPI)
			}
		}
	}
	fmt.Println()
}

// distSidekick imprime la distribution fire-pi de TOUS les Sidekick du match.
func distSidekick(chunks []chunkRec, ls layoutSpec) {
	fires := allFires(chunks, ls.layout)
	tally := map[int]int{}
	total := 0
	for _, f := range fires {
		if isSidekick(f.WeaponBytes) {
			tally[f.PlayerIndex]++
			total++
		}
	}
	fmt.Printf("  %-22s total=%-4d pi-dist=%s\n", ls.name, total, tallyStr(tally))
}

// allFires concatène les fires de tous les chunks type-2 d'un layout (avec USEstimator).
func allFires(chunks []chunkRec, layout weaponv3.FirePiLayout) []analysis.FireEvent {
	var out []analysis.FireEvent
	for _, c := range chunks {
		if c.chunkType != 2 {
			continue
		}
		est := weaponv3.USEstimator(c.data, c.startMS)
		if layout == weaponv3.FirePi4High {
			out = append(out, analysis.ScanFireEventsB5(c.data, est)...)
			continue
		}
		out = append(out, weaponv3.ScanFireEventsV3(c.data, est, layout, false)...)
	}
	return out
}

func isSidekick(wb [8]byte) bool {
	id := binary.BigEndian.Uint64(wb[:])
	high := uint32(id >> 32)
	low := uint32(id & 0xffffffff)
	return high == sidekickHigh32 && low == commonSuffix
}

func nearestKill(ms int) (kill, delta int) {
	best, bestD := 0, 1<<30
	for _, k := range fragKillsMS {
		d := ms - k
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = k, d
		}
	}
	return best, bestD
}

func tallyStr(m map[int]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := "{"
	for i, k := range keys {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("pi%d:%d", k, m[k])
	}
	return s + "}"
}

// ---- footer type-3 b36 ----

// footerB36 scanne le footer chunk-type-3 : pour chaque bloc highlight-event
// (end-marker 00 00 2e e0, recul 60 octets), lit th@b47, b36@slot, xuid associé,
// et tally (xuid -> {th, b36}). Donne le slot36 stable par joueur.
func footerB36(chunks []chunkRec, roster map[uint64]string) {
	var footers [][]byte
	for _, c := range chunks {
		if c.chunkType == 3 {
			footers = append(footers, c.data)
		}
	}
	if len(footers) == 0 {
		// fallback : le footer peut être le dernier chunk même mal typé.
		fmt.Println("  (aucun chunk type-3 ; scan du dernier chunk en fallback)")
		if len(chunks) > 0 {
			footers = append(footers, chunks[len(chunks)-1].data)
		}
	}
	// xuid -> b36 -> count, et xuid -> th -> count
	b36By := map[uint64]map[int]int{}
	for _, data := range footers {
		for _, ev := range scanHighlightBlocks(data) {
			if b36By[ev.xuid] == nil {
				b36By[ev.xuid] = map[int]int{}
			}
			b36By[ev.xuid][ev.b36]++
		}
	}
	xuids := make([]uint64, 0, len(roster))
	for x := range roster {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	fmt.Printf("%-22s %-16s %-s\n", "xuid", "gamertag", "b36(slot36)-dist")
	for _, x := range xuids {
		dist := "(aucun event)"
		if m := b36By[x]; m != nil {
			dist = tallyStr(m)
		}
		mark := ""
		if x == jgtmXUID {
			mark = "  <-- JGtm"
		}
		fmt.Printf("%-22d %-16s %-s%s\n", x, roster[x], dist, mark)
	}
	fmt.Println()
}

type hlEvent struct {
	xuid uint64
	th   int
	b36  int
	t    int
}

// scanHighlightBlocks localise chaque XUID (préfixe 0x2d/0x25, suffixe 0xc0), trouve
// l'end-marker [00 00 2e e0], recule 60 octets et lit th@b47 + b36@b36 (TOUS th, pas
// seulement th==10 : on veut aussi les kill/death events).
func scanHighlightBlocks(data []byte) []hlEvent {
	total := len(data) * 8
	var out []hlEvent
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := readByteAtBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := readU64LEAtBit(data, xstart)
		if x <= uint64(2e15) || x >= uint64(3e15) {
			continue
		}
		seen[xstart] = true
		if ev, ok := decodeBlock(data, xstart, total); ok {
			ev.xuid = x
			out = append(out, ev)
		}
	}
	return out
}

func decodeBlock(data []byte, xstart, total int) (hlEvent, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
	for b := xstart; b <= win-32; b++ {
		if readByteAtBit(data, b) == 0 && readByteAtBit(data, b+8) == 0 &&
			readByteAtBit(data, b+16) == 0x2e && readByteAtBit(data, b+24) == 0xe0 {
			ebs := b - 60*8
			if ebs < xstart {
				return hlEvent{}, false
			}
			th := int(readByteAtBit(data, ebs+47*8))
			t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
				int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
			b36 := int(readByteAtBit(data, ebs+36*8))
			return hlEvent{th: th, b36: b36, t: t}, true
		}
	}
	return hlEvent{}, false
}

// ---- bit helpers (port objectiveevents) ----

func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[bi]
	}
	return data[bi]<<off | data[bi+1]>>(8-off)
}

func readU64LEAtBit(data []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(data, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

// ---- IO ----

func type2Datas(chunks []chunkRec) [][]byte {
	var out [][]byte
	for _, c := range chunks {
		if c.chunkType == 2 {
			out = append(out, c.data)
		}
	}
	return out
}

func loadChunks() ([]chunkRec, error) {
	mfPath := filepath.Join(cacheDir, "film_manifests", matchPfx+".json")
	raw, err := os.ReadFile(mfPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var mf struct {
		Chunks []struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, err
	}
	var out []chunkRec
	for _, c := range mf.Chunks {
		p := filepath.Join(cacheDir, "film_chunks", matchPfx, fmt.Sprintf("chunk_%02d.bin", c.Index))
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			continue
		}
		out = append(out, chunkRec{
			index:     c.Index,
			chunkType: c.ChunkType,
			startMS:   c.StartMS,
			data:      decompress(b),
		})
	}
	return out, nil
}

func decompress(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if r, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
			defer r.Close()
			if inf, err := io.ReadAll(r); err == nil {
				return inf
			}
		}
	}
	return raw
}

func loadRoster(ctx context.Context, db *sql.DB, matchID string) (map[uint64]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT p.xuid, COALESCE(a.gamertag, '')
		FROM match_participants p
		LEFT JOIN xuid_aliases a ON a.xuid = p.xuid
		WHERE p.match_id = ? AND p.team_id IS NOT NULL`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadRoster: %w", err)
	}
	defer rows.Close()
	out := map[uint64]string{}
	for rows.Next() {
		var xs, gt string
		if err := rows.Scan(&xs, &gt); err != nil {
			continue
		}
		if n, ok := parseXUID(xs); ok {
			out[n] = gt
		}
	}
	return out, rows.Err()
}

func parseXUID(s string) (uint64, bool) {
	if len(s) > 6 && s[:5] == "xuid(" && s[len(s)-1] == ')' {
		s = s[5 : len(s)-1]
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func resolveFull(ctx context.Context, db *sql.DB, pfx string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY match_id LIMIT 1`,
		pfx+"%").Scan(&id)
	if err != nil {
		return "", fmt.Errorf("resolveFull(%s): %w", pfx, err)
	}
	return id, nil
}
