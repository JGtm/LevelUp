//go:build ignore

// tmp_killfeed_pair : teste l'hypothèse "KILL@t <-> DEATH@t" pour reconstituer
// le couple tueur->victime à partir des events type-3 (chacun ne porte qu'UN
// joueur). Croise contre killer_victim_pairs (oracle, avec time_ms).
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
	"unicode/utf16"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	minXUID         = uint64(2e15)
	maxXUID         = uint64(3e15)
	eventWindowBits = 20000
	eventDataBytes  = 60
	matchID         = "000d5950-83d9-423f-ab55-d068a7237b9f"
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

type ev struct {
	xuid     uint64
	gt       string
	typeHint int
	timeMS   int
	team     int
	duo      int
}

type pair struct {
	killer, victim uint64
	t              int
}

func main() {
	// 1) Oracle depuis DB
	oracle := loadOracle()
	fmt.Printf("oracle pairs=%d\n", len(oracle))

	// 2) Events depuis chunk_27
	path := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_27.bin`
	data, _ := os.ReadFile(path)
	payload := data
	if r, e := zlib.NewReader(bytes.NewReader(data)); e == nil {
		if inf, e2 := io.ReadAll(r); e2 == nil {
			payload = inf
		}
	}
	evs := scan(payload)

	var kills, deaths []ev
	for _, e := range evs {
		switch e.typeHint {
		case 50:
			kills = append(kills, e)
		case 20:
			deaths = append(deaths, e)
		}
	}
	fmt.Printf("kills=%d deaths=%d\n\n", len(kills), len(deaths))

	// 3) Pour chaque oracle pair (killer,victim,t), vérifier :
	//    - existe un KILL event de killer à |dt|<=tol
	//    - existe un DEATH event de victim à |dt|<=tol
	const tol = 3 // ms : oracle time_ms vs film time_ms (on a vu des dt=0..2)
	matchedK, matchedD, matchedBoth := 0, 0, 0
	teamMismatch := 0
	for _, p := range oracle {
		k := findEv(kills, p.killer, p.t, tol)
		d := findEv(deaths, p.victim, p.t, tol)
		if k != nil {
			matchedK++
		}
		if d != nil {
			matchedD++
		}
		if k != nil && d != nil {
			matchedBoth++
			// validation : killer.team != victim.team (no team-kill)
			if k.team == d.team {
				teamMismatch++
			}
		}
	}
	fmt.Println("=== Hypothèse KILL@t<->DEATH@t vs oracle killer_victim_pairs ===")
	fmt.Printf("  oracle pairs            : %d\n", len(oracle))
	fmt.Printf("  KILL ev trouvé (killer) : %d/%d\n", matchedK, len(oracle))
	fmt.Printf("  DEATH ev trouvé (victim): %d/%d\n", matchedD, len(oracle))
	fmt.Printf("  les DEUX trouvés        : %d/%d\n", matchedBoth, len(oracle))
	fmt.Printf("  team(killer)==team(victim) [doit être 0] : %d\n", teamMismatch)

	// 4) Reconstruction "aveugle" : pour chaque KILL@t, chercher LE DEATH@t (tol)
	//    de l'équipe adverse ; vérifier (killer,victim) ∈ oracle.
	fmt.Println("\n=== Reconstruction aveugle KILL->DEATH (sans oracle), validée a posteriori ===")
	oracleSet := map[[2]uint64]bool{}
	for _, p := range oracle {
		oracleSet[[2]uint64{p.killer, p.victim}] = true
	}
	good, bad, ambiguous, nodeath := 0, 0, 0, 0
	for _, ke := range kills {
		cands := findAll(deaths, ke.timeMS, tol)
		// retenir ceux d'équipe adverse
		var adv []ev
		for _, d := range cands {
			if d.team != ke.team && d.xuid != ke.xuid {
				adv = append(adv, d)
			}
		}
		switch len(adv) {
		case 0:
			nodeath++
		case 1:
			if oracleSet[[2]uint64{ke.xuid, adv[0].xuid}] {
				good++
			} else {
				bad++
			}
		default:
			ambiguous++
		}
	}
	fmt.Printf("  KILL total              : %d\n", len(kills))
	fmt.Printf("  1 seul DEATH adverse @t : good(∈oracle)=%d  bad(∉oracle)=%d\n", good, bad)
	fmt.Printf("  ambigus (>1 death @t)   : %d\n", ambiguous)
	fmt.Printf("  aucun death adverse @t  : %d\n", nodeath)

	// 5) Distribution des |dt| entre KILL et son DEATH apparié (diagnostic tol)
	fmt.Println("\n=== distribution dt(KILL,DEATH apparié sur oracle) ===")
	dtHist := map[int]int{}
	for _, p := range oracle {
		k := findEv(kills, p.killer, p.t, 50)
		d := findEv(deaths, p.victim, p.t, 50)
		if k != nil && d != nil {
			dtHist[abs(k.timeMS-d.timeMS)]++
		}
	}
	dks := []int{}
	for k := range dtHist {
		dks = append(dks, k)
	}
	sort.Ints(dks)
	for _, k := range dks {
		fmt.Printf("  dt=%dms : %d\n", k, dtHist[k])
	}

	// 6) Les 112-93 = 19 KILL "en trop" : doublons (multi-kill simultané ?) ou kills
	//    sans entrée oracle. Lister les KILL dont le couple n'est dans aucune oracle pair.
	fmt.Println("\n=== KILL events sans DEATH adverse simultané (suicide/bot/fin ?) ===")
	for _, ke := range kills {
		cands := findAll(deaths, ke.timeMS, 3)
		hasAdv := false
		for _, d := range cands {
			if d.team != ke.team && d.xuid != ke.xuid {
				hasAdv = true
			}
		}
		if !hasAdv {
			fmt.Printf("  KILL t=%-7d gt=%-16s team=%d duo=%d  deaths@t=%v\n", ke.timeMS, ke.gt, ke.team, ke.duo, deathsSummary(cands))
		}
	}

	// 7) doublons KILL (même xuid, même t) = multi-kill comptés 1× dans oracle agrégé ?
	fmt.Println("\n=== KILL events groupés par (xuid,t) : détecter doublons ===")
	cnt := map[[2]uint64]int{} // (xuid, t) -> n  (t encodé uint64)
	for _, ke := range kills {
		cnt[[2]uint64{ke.xuid, uint64(ke.timeMS)}]++
	}
	dup := 0
	for k, c := range cnt {
		if c > 1 {
			dup++
			fmt.Printf("  xuid=%d t=%d  x%d\n", k[0], k[1], c)
		}
	}
	fmt.Printf("  total KILL=%d ; couples (xuid,t) distincts=%d ; doublons=%d\n", len(kills), len(cnt), dup)

	// 8) Couverture : KILL film (déduit killer,victim via @t adverse) vs oracle.
	//    Reconstituer les couples depuis le film et comparer ensemble à l'oracle.
	fmt.Println("\n=== Couverture film->oracle (couples reconstitués) ===")
	filmPairs := map[[3]uint64]bool{} // killer,victim,t
	for _, ke := range kills {
		for _, d := range deaths {
			if abs(d.timeMS-ke.timeMS) <= 3 && d.team != ke.team && d.xuid != ke.xuid {
				filmPairs[[3]uint64{ke.xuid, d.xuid, uint64(ke.timeMS)}] = true
			}
		}
	}
	oracleTriple := map[[3]uint64]bool{}
	for _, p := range oracle {
		oracleTriple[[3]uint64{p.killer, p.victim, uint64(p.t)}] = true
	}
	// film couples retrouvés dans oracle (tolérance t +-3 sur le 3e champ)
	inBoth := 0
	for _, p := range oracle {
		for fk := range filmPairs {
			if fk[0] == p.killer && fk[1] == p.victim && abs(int(fk[2])-p.t) <= 3 {
				inBoth++
				break
			}
		}
	}
	fmt.Printf("  oracle pairs=%d ; couples film distincts=%d ; oracle retrouvés dans film=%d\n",
		len(oracle), len(filmPairs), inBoth)
}

func deathsSummary(ds []ev) string {
	s := ""
	for _, d := range ds {
		s += fmt.Sprintf("[%s team%d] ", d.gt, d.team)
	}
	if s == "" {
		return "(aucun)"
	}
	return s
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func findEv(list []ev, xuid uint64, t, tol int) *ev {
	var best *ev
	bestDt := tol + 1
	for i := range list {
		if list[i].xuid != xuid {
			continue
		}
		dt := abs(list[i].timeMS - t)
		if dt <= tol && dt < bestDt {
			best = &list[i]
			bestDt = dt
		}
	}
	return best
}

func findAll(list []ev, t, tol int) []ev {
	var out []ev
	for _, e := range list {
		if abs(e.timeMS-t) <= tol {
			out = append(out, e)
		}
	}
	return out
}

func loadOracle() []pair {
	dbPath := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT killer_xuid, victim_xuid, COALESCE(time_ms,-1) FROM killer_victim_pairs WHERE match_id = ? ORDER BY time_ms`, matchID)
	if err != nil {
		fmt.Println("oracle query:", err)
		os.Exit(1)
	}
	defer rows.Close()
	var out []pair
	for rows.Next() {
		var ks, vs string
		var t int
		rows.Scan(&ks, &vs, &t)
		var k, v uint64
		fmt.Sscan(ks, &k)
		fmt.Sscan(vs, &v)
		out = append(out, pair{k, v, t})
	}
	return out
}

// ---- décodeur chunk_27 (identique au parser de prod) ----

func scan(data []byte) []ev {
	totalBits := len(data) * 8
	var out []ev
	seen := map[int]bool{}
	for ms := 8; ms <= totalBits-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pfx := readByteAtBit(data, xe)
		if pfx != 0x2d && pfx != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		xuid := readU64LE(data, xs)
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}
		e, ok := parseAt(data, xs, xuid)
		if !ok {
			continue
		}
		out = append(out, e)
		seen[xs] = true
	}
	return out
}

func parseAt(data []byte, xs int, xuid uint64) (ev, bool) {
	total := len(data) * 8
	wend := xs + eventWindowBits
	if wend > total {
		wend = total
	}
	from := xs
	for {
		end := findMarker(data, from, wend, endMarker)
		if end < 0 {
			return ev{}, false
		}
		st := end - eventDataBytes*8
		if st < xs {
			from = end + 1
			continue
		}
		eb := readBytes(data, st, eventDataBytes)
		if eb == nil {
			from = end + 1
			continue
		}
		th := int(eb[47])
		valid := th == 50 || th == 20 || th == 10 || th == 51 || th == 52 ||
			th == 100 || th == 101 || th == 150 || th == 200 || th == 205 ||
			th == 210 || th == 220 || th == 225 || th == 230 || th == 235 ||
			th == 240 || th == 245 || th == 250
		if !valid {
			from = end + 1
			continue
		}
		return ev{
			xuid:     xuid,
			gt:       utf16le(eb[0:32]),
			typeHint: th,
			timeMS:   int(binary.BigEndian.Uint32(eb[48:52])),
			duo:      int(eb[36]),
			team:     int(eb[37]),
		}, true
	}
}

func utf16le(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u))
}

func readByteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return d[bi]
	}
	return d[bi]<<off | d[bi+1]>>(8-off)
}
func readBytes(d []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(d)*8 {
		return nil
	}
	o := make([]byte, n)
	for i := 0; i < n; i++ {
		o[i] = readByteAtBit(d, bit+i*8)
	}
	return o
}
func readU64LE(d []byte, bit int) uint64 {
	b := readBytes(d, bit, 8)
	if b == nil {
		return 0
	}
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(b[i]) << (uint(i) * 8)
	}
	return x
}
func findMarker(d []byte, s, e int, pat []byte) int {
	if s < 0 {
		s = 0
	}
	tb := len(d) * 8
	if e > tb {
		e = tb
	}
	pb := len(pat) * 8
	for bit := s; bit <= e-pb; bit++ {
		m := true
		for i := 0; i < len(pat); i++ {
			if readByteAtBit(d, bit+i*8) != pat[i] {
				m = false
				break
			}
		}
		if m {
			return bit
		}
	}
	return -1
}
