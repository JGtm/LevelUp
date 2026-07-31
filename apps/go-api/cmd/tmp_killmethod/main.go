// tmp_killmethod — M1 : CATEGORIE DE MORT (melee/grenade/distance/skill) via chunk_27.
//
// Le record de degat-arme (519 paquets type-0 0xd2) ne couvre PAS melee/grenade/terrain
// (Gravity Hammer = 0/519). Cette sonde teste si le KILL FEED chunk_27 distingue deja la
// METHODE, en s'appuyant sur les events MEDAL (type-100) horodates apparies aux KILLs.
//
// Pipeline :
//  1. analysis.ParseHighlightEvents(chunk_27) -> kills/deaths/medals (xuid,t,MedalType=b59).
//  2. Resoudre b59 (enum local de medaille highlight) -> medal_name_id en croisant, par
//     joueur, le NOMBRE d'events type-100 d'un b59 donne avec medals_earned (DB).
//  3. medal_name_id -> categorie de methode (melee/grenade/distance/...) via parquet
//     medal_definitions (name_en + heuristique de categorie).
//  4. Pour chaque kill : medaille-methode simultanee du tueur -> classer le kill.
//  5. Focus : kill marteau IKE ILYA -> JGtm (melee, 0 record d'arme) -> classable ?
package main

import (
	"database/sql"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	matchID    = "000d5950-83d9-423f-ab55-d068a7237b9f"
	chunk27    = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_27.bin`
	sharedDB   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	defPq      = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/backups/staging/halo_infinite/metadata/medal_definitions_20260607_184630.parquet`
	medalWinMS = 200 // fenetre d'appariement kill <-> medal du meme joueur (ms)
)

// medalDef : nom + categorie de methode deduite.
type medalDef struct {
	id       int64
	nameEn   string
	nameFr   string
	mtype    string // skill/multikill/style/spree/proficiency
	category string // melee/grenade/distance/headshot/multi/spree/other
}

// methodCategory classe une medaille par sa SEMANTIQUE de methode de kill (catalogue
// Halo Infinite stable). On ne se fie pas a medal_type (skill couvre tout) mais au NOM.
func methodCategory(nameEn, mtype string) string {
	switch nameEn {
	// --- melee / contact direct ---
	case "Back Smack", "Bulltrue", "Nightmare", "Bodyguard", "Hold This",
		"Ballista", "Pummel", "Combat Evolved", "Grand Slam", "Sword":
		return "melee"
	// --- grenade ---
	case "Stick", "Nade Shot", "Grenadier", "Demon", "Boom Block", "Cluster Luck",
		"Pineapple Express", "Hail Mary", "Bank Shot":
		return "grenade"
	// --- distance / sniper / precision arme ---
	case "Snipe", "No Scope", "Reload This", "Sneak King", "Quigley", "Mounted & Loaded",
		"Tag & Bag", "Gunslinger", "Sharpshooter":
		return "distance"
	// --- explosif lourd / vehicule / terrain ---
	case "Splatter", "Whiplash", "Mind the Gap", "Bomber", "Breacher":
		return "explosif/vehicule"
	}
	// medailles d'enchainement / multi : pas une methode unique de ce kill precis
	switch mtype {
	case "multikill":
		return "multi"
	case "spree":
		return "spree"
	}
	return "other(" + mtype + ")"
}

func loadDefs(idset map[int64]bool) (map[int64]medalDef, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	out := map[int64]medalDef{}
	for id := range idset {
		var en, fr, mt sql.NullString
		q := "SELECT COALESCE(name_en,''),COALESCE(name_fr,''),COALESCE(medal_type,'') FROM read_parquet('" + defPq + "') WHERE medal_name_id=?"
		if err := db.QueryRow(q, id).Scan(&en, &fr, &mt); err != nil {
			out[id] = medalDef{id: id, nameEn: fmt.Sprintf("?%d", id), category: "unknown"}
			continue
		}
		out[id] = medalDef{id: id, nameEn: en.String, nameFr: fr.String, mtype: mt.String,
			category: methodCategory(en.String, mt.String)}
	}
	return out, nil
}

// loadEarned : medals_earned du match -> par xuid : map[medal_name_id]count.
func loadEarned() (map[uint64]map[int64]int, map[int64]bool, error) {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT xuid, medal_name_id, count FROM medals_earned WHERE match_id=?", matchID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byX := map[uint64]map[int64]int{}
	idset := map[int64]bool{}
	for rows.Next() {
		var xs string
		var mid int64
		var c int
		rows.Scan(&xs, &mid, &c)
		var xu uint64
		fmt.Sscan(xs, &xu)
		if byX[xu] == nil {
			byX[xu] = map[int64]int{}
		}
		byX[xu][mid] += c
		idset[mid] = true
	}
	return byX, idset, nil
}

func gtFor(x uint64) string {
	m := map[uint64]string{
		2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
		2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
		2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
		2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
	}
	if g, ok := m[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

func main() {
	data, err := os.ReadFile(chunk27)
	if err != nil {
		fmt.Println("read chunk_27:", err)
		return
	}
	events, err := analysis.ParseHighlightEvents(data, 0)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	type ev struct {
		xuid uint64
		t    int
		b59  int
	}
	var kills, deaths, medals []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS, e.MedalType})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS, e.MedalType})
		case analysis.EventTypeMedal:
			medals = append(medals, ev{e.XUID, e.TimeMS, e.MedalType})
		}
	}
	fmt.Printf("chunk_27 : kills=%d deaths=%d medals(type-100..)=%d\n\n", len(kills), len(deaths), len(medals))

	earned, idset, err := loadEarned()
	if err != nil {
		fmt.Println("loadEarned:", err)
		return
	}
	defs, _ := loadDefs(idset)

	// ETAPE 2 : decoder b59 -> medal_name_id.
	// Hypothese : b59 est un index/enum stable de la medaille highlight. Pour chaque
	// joueur, le NB d'events type-100 d'un b59 donne == count de la medaille correspondante
	// dans medals_earned. On vote : pour chaque b59, on cherche le medal_name_id dont les
	// occurrences (par joueur) collent le mieux aux comptes d'events de ce b59.
	type b59x struct {
		b59  int
		xuid uint64
	}
	b59ByPlayer := map[b59x]int{}
	for _, m := range medals {
		b59ByPlayer[b59x{m.b59, m.xuid}]++
	}
	// b59 -> nb total events
	b59total := map[int]int{}
	for _, m := range medals {
		b59total[m.b59]++
	}
	// Pour chaque b59, candidat medal_name_id = celui present chez les memes joueurs avec
	// des comptes compatibles. Score = nb de joueurs ou (events b59) <= (count medaille).
	const synthAvenger = int64(9000000001) // medaille synthetique projet, presente partout -> exclure
	b59ToMedal := map[int]int64{}
	b59Score := map[int]int{}
	for b59 := range b59total {
		bestID := int64(0)
		bestScore := -1
		for id := range idset {
			if id == synthAvenger {
				continue
			}
			score := 0
			ok := true
			for bx, cnt := range b59ByPlayer {
				if bx.b59 != b59 {
					continue
				}
				ec := earned[bx.xuid][id]
				// match de comptage EXACT par joueur : le nb d'events b59 doit egaler
				// le count de la medaille (un meme b59 = une meme medaille unique).
				if ec == cnt {
					score++
				} else {
					ok = false
				}
			}
			if ok && score > bestScore {
				bestScore = score
				bestID = id
			}
		}
		b59ToMedal[b59] = bestID
		b59Score[b59] = bestScore
	}

	fmt.Println("=== ETAPE 2 : decodage b59 -> medal_name_id (vote count-match par joueur) ===")
	var b59keys []int
	for k := range b59total {
		b59keys = append(b59keys, k)
	}
	sort.Ints(b59keys)
	for _, b59 := range b59keys {
		id := b59ToMedal[b59]
		d := defs[id]
		fmt.Printf("  b59=%-3d (n=%2d events)  -> medal_name_id=%-12d %-18s [%s]  (score=%d)\n",
			b59, b59total[b59], id, d.nameEn, d.category, b59Score[b59])
	}

	// ETAPE 4 : classer chaque kill via medaille-methode simultanee.
	fmt.Println("\n=== ETAPE 4 : KILLs avec medaille-methode simultanee (fenetre +-200ms) ===")
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	classCount := map[string]int{}
	classified := 0
	for _, k := range kills {
		var hit *ev
		for i := range medals {
			if medals[i].xuid == k.xuid && abs(medals[i].t-k.t) <= medalWinMS {
				hit = &medals[i]
				break
			}
		}
		if hit == nil {
			classCount["(pas de medaille @t)"]++
			continue
		}
		id := b59ToMedal[hit.b59]
		d := defs[id]
		cat := d.category
		classCount[cat]++
		if cat != "" && cat != "unknown" && cat != "multi" && cat != "spree" &&
			cat[:5] != "other" {
			classified++
			fmt.Printf("  t=%6.1fs %-16s  b59=%-3d -> %-18s [%s]\n",
				float64(k.t)/1000, gtFor(k.xuid), hit.b59, d.nameEn, cat)
		}
	}
	fmt.Printf("\n  -> %d/%d kills classes par une medaille de METHODE (melee/grenade/distance/...)\n", classified, len(kills))
	fmt.Println("  distribution categories (tous kills) :")
	var cats []string
	for c := range classCount {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, c := range cats {
		fmt.Printf("    %-22s : %d\n", c, classCount[c])
	}

	// ETAPE 3/FOCUS : le kill marteau IKE ILYA -> JGtm.
	fmt.Println("\n=== FOCUS : kill marteau IKE ILYA -> JGtm (melee, 0 record d'arme) ===")
	const ike = uint64(2533274815845110)
	const jgtm = uint64(2533274823110022)
	// On cherche les KILLs d'IKE et les DEATHs de JGtm proches dans le temps.
	fmt.Println("  KILLs d'IKE ILYA :")
	for _, k := range kills {
		if k.xuid != ike {
			continue
		}
		// medaille @t ?
		mtxt := "(aucune medaille @t)"
		for i := range medals {
			if medals[i].xuid == ike && abs(medals[i].t-k.t) <= medalWinMS {
				id := b59ToMedal[medals[i].b59]
				mtxt = fmt.Sprintf("medaille b59=%d -> %s [%s]", medals[i].b59, defs[id].nameEn, defs[id].category)
				break
			}
		}
		// JGtm meurt-il ~en meme temps ?
		victim := ""
		for _, d := range deaths {
			if d.xuid == jgtm && abs(d.t-k.t) <= 400 {
				victim = "<-> DEATH JGtm"
				break
			}
		}
		fmt.Printf("    t=%6.1fs  IKE kill  %-14s  %s\n", float64(k.t)/1000, victim, mtxt)
	}
	fmt.Println("\n  medailles brutes du match liees au melee (Back Smack / Bulltrue / ...) :")
	for x, mm := range earned {
		for id, c := range mm {
			if defs[id].category == "melee" {
				fmt.Printf("    %-16s %-16s x%d (id=%d)\n", gtFor(x), defs[id].nameEn, c, id)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
