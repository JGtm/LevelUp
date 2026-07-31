// tmp_kw2_xref — SONDE : ingere le CSV de filmdec_killweapon2 (capture CE auto-nommee)
// et sort le kill feed NOMME : tueur -> victime | arme(famille).
//
// CSV (header "variantName,weaponHandle,e00,e04,e08,...") : variantName = high-32 du
// variant_name d'arme (= cle high-32 de analysis.WeaponIDToName, la FAMILLE) ; e04 =
// player-index VICTIME, e08 = player-index TUEUR (0xE1500000 + idx*0x10002).
//
// Mappe variantName -> nom d'arme (famille) via analysis.WeaponIDToName (high-32),
// decode tueur/victime -> gamertag (map pi + DB). Valide le principe sur n'importe quel film.
//
// Usage : tmp_kw2_xref <csv>
package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

const (
	playerIdxEncBase uint32 = 0xE1500000
	playerIdxStride  uint32 = 0x10002
)

var piXUID = [8]string{
	"2535467794760703", "2535437947245250", "2533274823110022", "2533274980284321",
	"2533274815845110", "2535444178793711", "2533274882097883", "2533274826120416",
}

func decodeIdx(v uint32) int {
	if v < playerIdxEncBase {
		return -1
	}
	d := v - playerIdxEncBase
	if d%playerIdxStride != 0 {
		return -1
	}
	idx := d / playerIdxStride
	if idx > 7 {
		return -1
	}
	return int(idx)
}

// weaponFamily : nom (famille) d'une arme dont high-32 == v. "" si inconnu.
func weaponFamily(v uint32) string {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n
		}
	}
	return ""
}

type kill struct {
	variant   uint32
	handle    uint32
	victimIdx int
	killerIdx int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_kw2_xref <csv>")
		return
	}
	kills := parseCSV(os.Args[1])
	fmt.Printf("=== %d kills ===\n", len(kills))
	gt := loadGamertags()

	fmt.Println("\n=== KILL FEED reconstitue (tueur -> victime | arme) ===")
	namedOK := 0
	for i, k := range kills {
		wf := weaponFamily(k.variant)
		armed := wf
		if armed == "" {
			armed = fmt.Sprintf("variant=0x%08x (hors catalogue)", k.variant)
		} else {
			namedOK++
		}
		fmt.Printf("  #%-2d %-16s -> %-16s | %s\n", i+1, name(gt, k.killerIdx), name(gt, k.victimIdx), armed)
	}
	fmt.Printf("\n>>> %d/%d kills avec arme nommee via catalogue\n", namedOK, len(kills))

	// vocabulaire des variantName (familles distinctes)
	fmt.Println("\n=== variantName distincts (famille) ===")
	vocab := map[uint32]int{}
	for _, k := range kills {
		vocab[k.variant]++
	}
	var vs []uint32
	for v := range vocab {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(a, b int) bool { return vocab[vs[a]] > vocab[vs[b]] })
	for _, v := range vs {
		wf := weaponFamily(v)
		if wf == "" {
			wf = "(inconnu)"
		}
		fmt.Printf("  0x%08x x%-2d -> %s\n", v, vocab[v], wf)
	}
}

func name(gt map[string]string, idx int) string {
	if idx < 0 || idx > 7 {
		return fmt.Sprintf("idx?%d", idx)
	}
	if g, ok := gt[piXUID[idx]]; ok && g != "" {
		return g
	}
	return "xuid:" + piXUID[idx]
}

func parseCSV(path string) []kill {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var out []kill
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	header := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if header {
			header = false
			if strings.HasPrefix(line, "variant") {
				continue
			}
		}
		p := strings.Split(line, ",")
		if len(p) < 5 {
			continue
		}
		au := func(s string) uint32 { n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64); return uint32(n) }
		// header: variantName,weaponHandle,e00,e04,e08,...
		out = append(out, kill{
			variant:   au(p[0]),
			handle:    au(p[1]),
			victimIdx: decodeIdx(au(p[3])), // e04
			killerIdx: decodeIdx(au(p[4])), // e08
		})
	}
	return out
}

func loadGamertags() map[string]string {
	gt := map[string]string{}
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("(DB indispo:", err, ")")
		return gt
	}
	defer db.Close()
	for _, x := range piXUID {
		var g sql.NullString
		db.QueryRow(`SELECT gamertag FROM xuid_aliases WHERE xuid=? LIMIT 1`, x).Scan(&g)
		gt[x] = g.String
	}
	return gt
}
