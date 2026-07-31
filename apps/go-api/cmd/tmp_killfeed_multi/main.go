//go:build ignore

// tmp_killfeed_multi : valide la généralisation du décodage (team @block[37],
// duo @block[36], KILL@t<->DEATH@t adverse) sur plusieurs matchs.
package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf16"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	minXUID         = uint64(2e15)
	maxXUID         = uint64(3e15)
	eventWindowBits = 20000
	eventDataBytes  = 60
	chunkRoot       = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
	sharedDB        = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

type ev struct {
	xuid     uint64
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
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	prefixes := []string{"000d5950", "00502e52", "00ba2e1c", "01e1f945", "0215fe6b"}
	if len(os.Args) > 1 {
		prefixes = os.Args[1:]
	}

	for _, pfx := range prefixes {
		validateMatch(db, pfx)
		fmt.Println()
	}
}

func validateMatch(db *sql.DB, pfx string) {
	// résoudre UUID complet
	var mid string
	err := db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE ? LIMIT 1`, pfx+"%").Scan(&mid)
	if err != nil {
		fmt.Printf("[%s] match_id introuvable: %v\n", pfx, err)
		return
	}
	// oracle pairs
	oracle := loadOracle(db, mid)
	// team réel (match_participants)
	realTeam := loadTeams(db, mid)

	path := filepath.Join(chunkRoot, pfx, "chunk_27.bin")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[%s] read chunk: %v\n", pfx, err)
		return
	}
	payload := data
	if r, e := zlib.NewReader(bytes.NewReader(data)); e == nil {
		if inf, e2 := io.ReadAll(r); e2 == nil {
			payload = inf
		}
	}
	evs := scan(payload)
	var kills, deaths []ev
	hist := map[int]int{}
	for _, e := range evs {
		hist[e.typeHint]++
		if e.typeHint == 50 {
			kills = append(kills, e)
		}
		if e.typeHint == 20 {
			deaths = append(deaths, e)
		}
	}

	// team film vs team DB
	filmTeam := map[uint64]int{}
	for _, e := range evs {
		filmTeam[e.xuid] = e.team
	}
	teamOK, teamTot := 0, 0
	// la valeur film peut être un relabel ; on vérifie la PARTITION (même classe)
	// via cohérence : 2 joueurs DB-même-équipe ont film-même-valeur.
	partitionOK := checkPartition(filmTeam, realTeam)

	// KILL<->DEATH vs oracle
	const tol = 3
	both := 0
	for _, p := range oracle {
		k := find(kills, p.killer, p.t, tol)
		d := find(deaths, p.victim, p.t, tol)
		if k != nil && d != nil {
			both++
		}
	}
	// reconstruction aveugle
	oset := map[[2]uint64]bool{}
	for _, p := range oracle {
		oset[[2]uint64{p.killer, p.victim}] = true
	}
	good, bad, amb, nod := 0, 0, 0, 0
	for _, ke := range kills {
		var adv []ev
		for _, d := range deaths {
			if absI(d.timeMS-ke.timeMS) <= tol && d.team != ke.team && d.xuid != ke.xuid {
				adv = append(adv, d)
			}
		}
		switch len(adv) {
		case 0:
			nod++
		case 1:
			if oset[[2]uint64{ke.xuid, adv[0].xuid}] {
				good++
			} else {
				bad++
			}
		default:
			amb++
		}
	}
	_ = teamOK
	_ = teamTot

	fmt.Printf("[%s] %s\n", pfx, mid)
	fmt.Printf("  type_hint hist: %v\n", hist)
	fmt.Printf("  oracle=%d kills=%d deaths=%d | KILL&DEATH@oracle=%d/%d\n", len(oracle), len(kills), len(deaths), both, len(oracle))
	fmt.Printf("  team-partition film==DB: %v\n", partitionOK)
	fmt.Printf("  reconstruction aveugle: good=%d bad=%d ambig=%d no-death=%d (sur %d kills)\n", good, bad, amb, nod, len(kills))
}

// checkPartition : la classification film induit-elle la MÊME partition que la DB ?
// (les labels peuvent différer, on compare l'équivalence par paires)
func checkPartition(film, real map[uint64]int) bool {
	xs := []uint64{}
	for x := range real {
		if _, ok := film[x]; ok {
			xs = append(xs, x)
		}
	}
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ {
			a, b := xs[i], xs[j]
			sameReal := real[a] == real[b]
			sameFilm := film[a] == film[b]
			if sameReal != sameFilm {
				return false
			}
		}
	}
	return len(xs) > 0
}

func find(list []ev, xuid uint64, t, tol int) *ev {
	var best *ev
	bd := tol + 1
	for i := range list {
		if list[i].xuid != xuid {
			continue
		}
		dt := absI(list[i].timeMS - t)
		if dt <= tol && dt < bd {
			best = &list[i]
			bd = dt
		}
	}
	return best
}
func absI(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func loadOracle(db *sql.DB, mid string) []pair {
	rows, err := db.Query(`SELECT killer_xuid, victim_xuid, COALESCE(time_ms,-1) FROM killer_victim_pairs WHERE match_id=? ORDER BY time_ms`, mid)
	if err != nil {
		return nil
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
func loadTeams(db *sql.DB, mid string) map[uint64]int {
	rows, err := db.Query(`SELECT xuid, team_id FROM match_participants WHERE match_id=?`, mid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[uint64]int{}
	for rows.Next() {
		var x string
		var t int
		rows.Scan(&x, &t)
		var xu uint64
		fmt.Sscan(x, &xu)
		out[xu] = t
	}
	return out
}

// ---- décodeur ----
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
		valid := th == 50 || th == 20 || th == 10 || th == 100 || th == 101 ||
			th == 51 || th == 52 || th == 150 || th == 200 || th == 205 ||
			th == 210 || th == 220 || th == 225 || th == 230 || th == 235 ||
			th == 240 || th == 245 || th == 250
		if !valid {
			from = end + 1
			continue
		}
		return ev{
			xuid:     xuid,
			typeHint: th,
			timeMS:   int(binary.BigEndian.Uint32(eb[48:52])),
			team:     int(eb[37]),
			duo:      int(eb[36]),
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

var _ = utf16le
