// tmp_chunk2627 — THROWAWAY, OEIL NEUF : que contiennent chunk_26 et chunk_27 au-delà du connu,
// et que contient highlight_events.raw_json ? On ne présume rien.
package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

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

// parse packets [type u16][b2 u8][b3 u8][size u32][ts u64], histogram + premiers.
func packetReport(name string, d []byte) {
	fmt.Printf("\n=== %s : %d octets inflatés ===\n", name, len(d))
	typeCount := map[uint16]int{}
	typeBytes := map[uint16]int{}
	off, npkt := 0, 0
	var firsts []string
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		typeCount[typ]++
		typeBytes[typ] += sz
		if npkt < 12 {
			firsts = append(firsts, fmt.Sprintf("    pkt%d off=%d type=%d size=%d ts=%d", npkt, off, typ, sz, ts))
		}
		off += 16 + sz
		npkt++
	}
	fmt.Printf("  %d paquets parsés (couverture %d/%d octets)\n", npkt, off, len(d))
	if npkt == 0 || off < len(d)/2 {
		fmt.Printf("  >>> framing [type][size][ts] NE colle PAS (couverture faible) — format différent (bit-packé ?)\n")
	}
	var types []uint16
	for t := range typeCount {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	for _, t := range types {
		fmt.Printf("    type=%-4d count=%-5d bytes=%d\n", t, typeCount[t], typeBytes[t])
	}
	for _, s := range firsts {
		fmt.Println(s)
	}
	// hex des 64 premiers octets
	n := 64
	if len(d) < n {
		n = len(d)
	}
	fmt.Printf("  hex[0:%d]: %x\n", n, d[:n])
}

func main() {
	packetReport("chunk_26", inflate(cache+"/chunk_26.bin"))
	packetReport("chunk_27", inflate(cache+"/chunk_27.bin"))

	// raw_json des highlight events
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("\ndb err:", err)
		return
	}
	defer db.Close()
	var full string
	db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&full)
	fmt.Println("\n=== highlight_events.raw_json (échantillons : 3 kills, 2 medals) ===")
	for _, et := range []string{"kill", "medal"} {
		lim := 3
		if et == "medal" {
			lim = 2
		}
		rows, err := db.Query(`SELECT event_type, time_ms, xuid, type_hint, raw_json FROM highlight_events WHERE match_id=? AND event_type=? LIMIT ?`, full, et, lim)
		if err != nil {
			fmt.Println("  err:", err)
			continue
		}
		for rows.Next() {
			var t, x, rj sql.NullString
			var tm, th sql.NullInt64
			rows.Scan(&t, &tm, &x, &th, &rj)
			fmt.Printf("  [%s t=%d xuid=%s th=%d] raw_json=%s\n", t.String, tm.Int64, x.String, th.Int64, rj.String)
		}
		rows.Close()
	}
}
