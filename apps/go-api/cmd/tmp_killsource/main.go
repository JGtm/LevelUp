// tmp_killsource — SYNTHESE : TABLE SOURCE-PAR-KILL sur les 93 kills de chunk_27 (000d5950).
//
// Combine les acquis A1 + M1 + le decodage record-de-degat (tmp_dmgscan) :
//
//	(a) MELEE : si le kill porte une medaille-melee @t (chunk_27 type-100) -> categorie melee
//	    (path M1). Aucun record d'arme (Gravity Hammer = 0/519) -> seule voie pour le marteau.
//	(b) ARME : sinon, on apparie le kill au record de degat (type-0 0xd2) le plus pertinent
//	    AVANT/AUTOUR de la mort. Comme le record ne porte NI attaquant NI victime (A1 : +0x0c
//	    = arme, pas joueur ; +0x10 victime gate=0 dans 519/519), l'attribution se fait par
//	    PROXIMITE TEMPORELLE : dernier tick (densest famille) dans la fenetre [t-W, t+marge].
//	    On rapporte l'AMBIGUITE (nb de familles distinctes dans la fenetre) = mesure de fiabilite.
//
// Sortie : par kill, (tueur, victime, t, methode-attribution, source/categorie, ambiguite).
// Calibre la fenetre W et resume la couverture par voie.
//
// Usage : tmp_killsource [W_ms]   (defaut 1500)
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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	cache      = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	chunk27    = cache + `/chunk_27.bin`
	matchID    = "000d5950-83d9-423f-ab55-d068a7237b9f"
	sharedDB   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	defPq      = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/backups/staging/halo_infinite/metadata/medal_definitions_20260607_184630.parquet`
	t0Us       = uint64(4537898226)
	deserStart = 36
	variantSfx = uint32(0x42c9679f)
	medalWinMS = 200
)

var xuidGT = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
		id64name[id] = n
	}
}

func nameOf(x uint64) string {
	if g, ok := xuidGT[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
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
	typ     uint16
	ts      uint64
	payload []byte
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+size]})
		off += 16 + size
	}
	return out
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

// dmgRec : un tick de degat decode (famille + cause + temps). Pas d'attaquant/victime.
type dmgRec struct {
	tms   int
	fam   string
	cause uint64
}

func collectDmg() []dmgRec {
	var recs []dmgRec
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d) {
			if p.typ != 0 || len(p.payload) == 0 || p.payload[0] != 0xd2 {
				continue
			}
			br := filmdec.NewBitReader(p.payload)
			br.Skip(deserStart)
			var cause uint64
			if br.ReadBit() {
				cause = 0
			} else {
				cause = 1 + br.ReadBits(2)
			}
			br.ReadBits(5) // R5 prefixe
			fam32 := uint32(br.ReadBits(32))
			low := uint32(br.ReadBits(32))
			fam, known := h32name[fam32]
			if !known || low != variantSfx {
				continue
			}
			recs = append(recs, dmgRec{tsToMs(p.ts), fam, cause})
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tms < recs[j].tms })
	return recs
}

type kfRow struct {
	killer, victim uint64
	t              int
}

type medalEv struct {
	xuid uint64
	t    int
	b59  int
}

func killFeed() (feed []kfRow, medals []medalEv) {
	events, _ := analysis.ParseHighlightEvents(inflate(chunk27), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeMedal:
			medals = append(medals, medalEv{e.XUID, e.TimeMS, e.MedalType})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.x == k.x {
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
			feed = append(feed, kfRow{k.x, deaths[best].x, k.t})
		}
	}
	return
}

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

func loadDefs(idset map[int64]bool) map[int64]string {
	db, _ := sql.Open("duckdb", ":memory:")
	defer db.Close()
	out := map[int64]string{}
	for id := range idset {
		var en, mt sql.NullString
		q := "SELECT COALESCE(name_en,''),COALESCE(medal_type,'') FROM read_parquet('" + defPq + "') WHERE medal_name_id=?"
		if db.QueryRow(q, id).Scan(&en, &mt) != nil {
			continue
		}
		out[id] = methodCategory(en.String, mt.String)
	}
	return out
}

func methodCategory(nameEn, mtype string) string {
	switch nameEn {
	case "Back Smack", "Bulltrue", "Nightmare", "Bodyguard", "Hold This",
		"Ballista", "Pummel", "Combat Evolved", "Grand Slam", "Sword":
		return "melee"
	case "Stick", "Nade Shot", "Grenadier", "Demon", "Boom Block", "Cluster Luck",
		"Pineapple Express", "Hail Mary", "Bank Shot":
		return "grenade"
	case "Snipe", "No Scope", "Reload This", "Sneak King", "Quigley", "Mounted & Loaded",
		"Tag & Bag", "Gunslinger", "Sharpshooter":
		return "distance"
	}
	return "other"
}

// meleeByKiller : pour chaque (xuid tueur, t kill), la medaille-melee @t existe-t-elle ?
func meleeMedalForKiller(medals []medalEv, b59melee map[int]bool, killer uint64, t int) bool {
	for _, m := range medals {
		if m.xuid == killer && abs(m.t-t) <= medalWinMS && b59melee[m.b59] {
			return true
		}
	}
	return false
}

type kill struct {
	killer, victim uint64
	t              int
	method         string // "melee-medaille" | "arme-record" | "non-resolu"
	source         string // categorie melee OU famille d'arme
	ambig          int    // nb familles distinctes dans la fenetre (arme)
	nrec           int    // nb de records dans la fenetre
	dt             int    // ecart temporel record retenu
}

func main() {
	build()
	W := 1500
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &W)
	}
	feed, medals := killFeed()
	recs := collectDmg()
	fmt.Printf("=== %d kills (chunk_27) ; %d records de degat-arme (type-0 0xd2) ; fenetre W=%dms ===\n\n",
		len(feed), len(recs), W)

	// melee : decoder les b59-melee une fois.
	b59melee := decodeB59Melee(medals)

	var table []kill

	// stats de couverture
	var nMelee, nArmeStrict, nArmeAmbig, nNon int

	for _, k := range feed {
		row := kill{killer: k.killer, victim: k.victim, t: k.t, method: "non-resolu", source: "-", ambig: 0, nrec: 0, dt: -1}

		// (a) MELEE via medaille @t du tueur.
		if meleeMedalForKiller(medals, b59melee, k.killer, k.t) {
			row.method = "melee-medaille(M1)"
			row.source = "MELEE (categorie)"
			nMelee++
			table = append(table, row)
			continue
		}

		// (b) ARME : records dans [t-W, t+marge]. marge = 300ms (latence kill-feed vs tick letal).
		const marge = 300
		fams := map[string]int{}
		bestIdx, bestDt := -1, 1<<30
		for i := range recs {
			if recs[i].tms < k.t-W || recs[i].tms > k.t+marge {
				continue
			}
			fams[recs[i].fam]++
			row.nrec++
			// dernier tick avant la mort (ou le plus proche) : on prend le plus proche du kill.
			d := recs[i].tms - k.t
			if d < 0 {
				d = -d
			}
			if d < bestDt {
				bestDt, bestIdx = d, i
			}
		}
		row.ambig = len(fams)
		if bestIdx >= 0 {
			row.dt = recs[bestIdx].tms - k.t
			row.source = recs[bestIdx].fam
			row.method = "arme-record(proximite)"
			if len(fams) == 1 {
				nArmeStrict++
			} else {
				nArmeAmbig++
			}
		} else {
			nNon++
		}
		table = append(table, row)
	}

	// ── AFFICHAGE TABLE ──
	fmt.Println("=== TABLE SOURCE-PAR-KILL (93) ===")
	fmt.Printf("  %-7s %-16s %-16s %-22s %-26s %s\n", "t(s)", "tueur", "victime", "methode", "source", "ambig/nrec/dt")
	for _, r := range table {
		fmt.Printf("  %6.1f  %-16s %-16s %-22s %-26s %d/%d/%+dms\n",
			float64(r.t)/1000, nameOf(r.killer), nameOf(r.victim), r.method, r.source, r.ambig, r.nrec, r.dt)
	}

	// ── COUVERTURE ──
	fmt.Println("\n=== COUVERTURE par voie ===")
	fmt.Printf("  melee (categorie, via medaille M1)        : %d\n", nMelee)
	fmt.Printf("  arme record NON-AMBIGU (1 famille window) : %d   <- attribution fiable famille\n", nArmeStrict)
	fmt.Printf("  arme record AMBIGU (>1 famille window)    : %d   <- famille la + proche, NON fiable\n", nArmeAmbig)
	fmt.Printf("  non-resolu (aucun record, pas de melee)   : %d\n", nNon)
	fmt.Printf("  TOTAL                                     : %d/%d\n", nMelee+nArmeStrict+nArmeAmbig+nNon, len(feed))

	// ── VALIDATION NARRATION ──
	fmt.Println("\n=== VALIDATION NARRATION ===")
	checkNarr(table, "IKE ILYA", "JGtm", "MELEE (marteau)")
	checkNarr(table, "JGtm", "Akatsuki fire17", "BR75")
}

func decodeB59Melee(medals []medalEv) map[int]bool {
	byX, idset, err := loadEarned()
	if err != nil {
		return map[int]bool{}
	}
	defs := loadDefs(idset)
	b59ByPlayer := map[[2]int64]int{}
	b59total := map[int]int{}
	for _, m := range medals {
		b59ByPlayer[[2]int64{int64(m.b59), int64(m.xuid)}]++
		b59total[m.b59]++
	}
	out := map[int]bool{}
	for b59 := range b59total {
		bestID, bestScore := int64(0), -1
		for id := range idset {
			score, ok := 0, true
			for k, cnt := range b59ByPlayer {
				if int(k[0]) != b59 {
					continue
				}
				if byX[uint64(k[1])][id] == cnt {
					score++
				} else {
					ok = false
				}
			}
			if ok && score > bestScore {
				bestScore, bestID = score, id
			}
		}
		if defs[bestID] == "melee" {
			out[b59] = true
		}
	}
	return out
}

func checkNarr(table []kill, killer, victim, want string) {
	found := false
	for _, r := range table {
		if nameOf(r.killer) == killer && nameOf(r.victim) == victim {
			found = true
			fmt.Printf("  %s -> %s [attendu %s] : methode=%s source=%s (ambig=%d nrec=%d dt=%+dms) t=%.1fs\n",
				killer, victim, want, r.method, r.source, r.ambig, r.nrec, r.dt, float64(r.t)/1000)
		}
	}
	if !found {
		fmt.Printf("  %s -> %s [attendu %s] : AUCUN kill apparie dans la table\n", killer, victim, want)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
