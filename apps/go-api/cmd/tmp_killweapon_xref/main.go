// tmp_killweapon_xref — SONDE : ingere le CSV de capture CE filmdec_killweapon
// (hook FUN_1406730c4) et reconstitue le kill feed du film 000d5950 :
//
//	event+0x04 = player-index VICTIME, event+0x08 = player-index TUEUR
//	(encodes 0xE1500000 + idx*0x10002), weaponHandle = [event+0x538].
//
// Decode victime/tueur -> gamertag (map pi figee + DB), groupe par weaponHandle
// (= arme par instance), et croise avec shared.killer_victim_pairs.weapon_id pour
// rattacher l'arme connue eventuelle. Sert a valider tueur/victime et a preparer le
// mapping handle->arme (le pont handle->WST viendra du workflow RE en cours).
//
// Usage : tmp_killweapon_xref <csv>
package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

// playerIdxEncBase + idx*playerIdxStride = la valeur encodee d'un player-index dans
// les structs kill-feed/dead-state (resolue par FUN_140e958c4).
const (
	playerIdxEncBase uint32 = 0xE1500000 // = 3780116480 (idx0) ; le "0xE15C0000" historique etait faux
	playerIdxStride  uint32 = 0x10002    // = 65538
)

// map pi (0-7) -> xuid (film 000d5950), figee depuis xuid_aliases.
var piXUID = [8]string{
	"2535467794760703", // pi0 whiteknight2519
	"2535437947245250", // pi1 JAVIERLOLITO540
	"2533274823110022", // pi2 JGtm
	"2533274980284321", // pi3 LORD PEINX13
	"2533274815845110", // pi4 IKE ILYA
	"2535444178793711", // pi5 Akatsuki fire17
	"2533274882097883", // pi6 aldusbroncus
	"2533274826120416", // pi7 VitaminA1688
}

// decodeIdx : (val - base)/stride si entier 0..7, sinon -1.
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

type row struct {
	weaponDefId  int
	weaponHandle uint32
	attacker     uint32
	victimIdx    int
	killerIdx    int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_killweapon_xref <csv>")
		return
	}
	rows := parseCSV(os.Args[1])
	fmt.Printf("=== %d kills ===\n", len(rows))

	gt := loadGamertags()

	// 1) table par kill
	fmt.Println("\n=== kill feed reconstitue (tueur -> victime | handle | defId) ===")
	for i, r := range rows {
		fmt.Printf("  #%-2d %-16s -> %-16s | handle=%-10d defId=%d\n",
			i+1, name(gt, r.killerIdx), name(gt, r.victimIdx), r.weaponHandle, r.weaponDefId)
	}

	// 2) vocabulaire des handles (arme par instance) : quel(s) tueur(s) par handle
	fmt.Println("\n=== vocabulaire weaponHandle (arme = instance ; tueurs associes) ===")
	byHandle := map[uint32]map[int]int{}
	for _, r := range rows {
		if byHandle[r.weaponHandle] == nil {
			byHandle[r.weaponHandle] = map[int]int{}
		}
		byHandle[r.weaponHandle][r.killerIdx]++
	}
	var handles []uint32
	for h := range byHandle {
		handles = append(handles, h)
	}
	sort.Slice(handles, func(a, b int) bool { return handles[a] < handles[b] })
	for _, h := range handles {
		var ks []string
		for k, c := range byHandle[h] {
			ks = append(ks, fmt.Sprintf("%s x%d", name(gt, k), c))
		}
		sort.Strings(ks)
		fmt.Printf("  handle=%-10d : %s\n", h, strings.Join(ks, ", "))
	}

	// 3) croisement DB : killer_victim_pairs.weapon_id par (killer,victim)
	crossKVP(rows, gt)
}

func name(gt map[string]string, idx int) string {
	if idx < 0 || idx > 7 {
		return fmt.Sprintf("idx?%d", idx)
	}
	x := piXUID[idx]
	if g, ok := gt[x]; ok && g != "" {
		return g
	}
	return "xuid:" + x
}

func parseCSV(path string) []row {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var out []row
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
			if strings.HasPrefix(line, "weapon") {
				continue
			}
		}
		p := strings.Split(line, ",")
		if len(p) < 8 {
			continue
		}
		ai := func(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
		au := func(s string) uint32 { n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64); return uint32(n) }
		// header: weaponDefId,weaponHandle,attacker,p00,p04,p08,...
		r := row{
			weaponDefId:  ai(p[0]),
			weaponHandle: au(p[1]),
			attacker:     au(p[2]),
			victimIdx:    decodeIdx(au(p[4])), // p04
			killerIdx:    decodeIdx(au(p[5])), // p08
		}
		out = append(out, r)
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

func crossKVP(rows []row, gt map[string]string) {
	fmt.Println("\n=== croisement killer_victim_pairs.weapon_id (DB) ===")
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("(DB indispo:", err, ")")
		return
	}
	defer db.Close()
	var fullID string
	if err := db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&fullID); err != nil {
		fmt.Println("  match lookup:", err)
		return
	}
	matched := 0
	for i, r := range rows {
		if r.killerIdx < 0 || r.victimIdx < 0 {
			continue
		}
		kx, vx := piXUID[r.killerIdx], piXUID[r.victimIdx]
		var wid sql.NullString
		var cnt sql.NullInt64
		err := db.QueryRow(`SELECT weapon_id, count FROM killer_victim_pairs WHERE match_id=? AND killer_xuid=? AND victim_xuid=? LIMIT 1`,
			fullID, kx, vx).Scan(&wid, &cnt)
		if err == nil {
			matched++
			fmt.Printf("  #%-2d %-16s->%-16s handle=%-10d : KVP.weapon_id=%s (count=%d)\n",
				i+1, name(gt, r.killerIdx), name(gt, r.victimIdx), r.weaponHandle, wid.String, cnt.Int64)
		}
	}
	fmt.Printf("  >>> %d/%d kills apparies dans killer_victim_pairs\n", matched, len(rows))
}
