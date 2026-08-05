// frontb_coverage — FRONT B : remesure de la couverture fire-event sur les kills
// oracle de 000d5950 avec le BON indexage (acurtis pi vs fire b5>>4).
//
// Pour CHAQUE kill oracle (tueur JGtm), on cherche le DERNIER fire-event du tueur
// AVANT le kill dans plusieurs fenêtres (500/1000/1500/3000 ms), sous deux
// hypothèses d'indexage :
//
//	(a) fire pi == acurtis pi du tueur ;
//	(b) fire pi ∈ ensemble empirique des pi qui portent réellement les Sidekick.
//
// On teste aussi les 4 layouts de lecture du fire-pi (FirePi4High v2, 5SpanBefore,
// 5HighInB5, 5LowInB5) × relax3 {off,on}, pour voir si le pi du fire-event s'aligne
// sur l'acurtis pi sous un autre layout (auquel cas il n'y a PAS de désalignement,
// juste une mauvaise lecture du champ).
//
// LECTURE SEULE. Aucune écriture DB. Usage :
//
//	CGO_ENABLED=1 go run ./cmd/frontb_coverage
package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
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
)

const (
	cacheDir   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`
	sharedDB   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	matchShort = "000d5950"
	jgtmXUID   = "2533274823110022"
	sidekickHi = uint32(0xf408190f) // high-32 du Mk51 Sidekick (Frag Parfait)
)

// oracleKills — les 6 kills connus de JGtm (ms), source killer_victim_pairs / énoncé.
var oracleKills = []int{104995, 112869, 149471, 192884, 230628, 258590}

// windows — fenêtres AVANT le kill testées.
var windows = []int{500, 1000, 1500, 3000}

type chunk struct {
	index     int
	data      []byte
	startMS   int
	durMS     int
	chunkType int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "frontb_coverage:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	full, err := resolveFull(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("=== FRONT B — couverture fire-event sur kills oracle (%s) ===\n", matchShort)
	fmt.Printf("match=%s tueur=JGtm xuid=%s\n\n", full, jgtmXUID)

	chunks, err := loadChunks()
	if err != nil {
		return err
	}
	roster, err := loadRoster(ctx, full)
	if err != nil {
		return err
	}
	fmt.Printf("roster numérique: %d joueurs\n", len(roster))

	// 1) Résolution acurtis pi (bit-level) — premier chunk gagnant.
	datas := gameplayDatas(chunks)
	xuidToPI := weaponv3.ResolveBest(roster, datas)
	jgtmN, _ := strconv.ParseUint(jgtmXUID, 10, 64)
	acurtisPI, hasAcurtis := xuidToPI[jgtmN]
	fmt.Printf("acurtis pi (bit-level) JGtm = %d (résolu=%v)\n", acurtisPI, hasAcurtis)
	fmt.Print("acurtis pi de tout le roster: ")
	printRosterPI(roster, xuidToPI)
	fmt.Println()

	// 2) Pour chaque (layout, relax3) : scanner les fires, isoler les Sidekick,
	//    montrer leur distribution de pi, puis mesurer la couverture sur 6 kills
	//    sous hypothèse (a) acurtis-pi et (b) pi-empirique-des-Sidekick.
	layouts := []struct {
		name string
		l    weaponv3.FirePiLayout
	}{
		{"4high(v2)", weaponv3.FirePi4High},
		{"5span", weaponv3.FirePi5SpanBefore},
		{"5highb5", weaponv3.FirePi5HighInB5},
		{"5lowb5", weaponv3.FirePi5LowInB5},
	}
	for _, relax := range []bool{false, true} {
		for _, lay := range layouts {
			analyzeLayout(chunks, lay.name, lay.l, relax, acurtisPI)
		}
	}

	// 3) DIAGNOSTIC APPROFONDI (layout v2 4high, relax3 ON = recall max) :
	//    pour chaque kill, lister TOUS les fires (toute arme) du pi=acurtis(2)
	//    dans ±3000ms, puis TOUS les fires (toute arme, tout pi) dans ±1500ms.
	//    Détermine si pi=2 tire jamais, et quels pi tirent autour des kills.
	fmt.Println("════════ DIAGNOSTIC APPROFONDI (4high, relax3=on) ════════")
	deepFires := scanAllFires(chunks, weaponv3.FirePi4High, true)
	deepDiagnostic(deepFires, acurtisPI)

	// 4) COUVERTURE RÉELLE : pour chaque kill, le DERNIER fire (TOUTE arme) de
	//    pi=acurtis(2) avant le kill, par fenêtre. C'est la vraie mesure "le tueur
	//    a-t-il un fire récupérable sous son pi acurtis ?".
	fmt.Println("\n════════ COUVERTURE RÉELLE (pi=acurtis, toute arme, dernier fire avant kill) ════════")
	realCoverage(deepFires, acurtisPI)

	// 5) Ce que weaponv3 attribue réellement par kill (pipeline complet).
	fmt.Println("\n════════ ATTRIBUTION weaponv3 (pipeline complet) ════════")
	v3Attributions(full, roster, chunks)

	// 6) Vérification croisée DB : killer_victim_pairs + highlight_events.
	fmt.Println("\n════════ VÉRIFICATION CROISÉE DB ════════")
	dumpKillerVictim(ctx, full)
	return nil
}

// realCoverage : dernier fire (toute arme) de pi=acurtis avant chaque kill, fenêtre.
func realCoverage(fires []analysis.FireEvent, acurtisPI int) {
	cnt := map[int]int{}
	for _, k := range oracleKills {
		fmt.Printf("KILL @%dms : ", k)
		for _, w := range windows {
			var best *analysis.FireEvent
			for i := range fires {
				f := &fires[i]
				if f.PlayerIndex != acurtisPI {
					continue
				}
				ts := int(f.TimestampMS)
				if ts > k || ts < k-w {
					continue
				}
				if best == nil || f.TimestampMS > best.TimestampMS {
					best = f
				}
			}
			if best != nil {
				cnt[w]++
				fmt.Printf("[%dms→%s Δ%d] ", w, shortName(best.WeaponName), k-int(best.TimestampMS))
			} else {
				fmt.Printf("[%dms→∅] ", w)
			}
		}
		fmt.Println()
	}
	fmt.Print("COUVERTURE (pi=acurtis): ")
	for _, w := range windows {
		fmt.Printf("%dms=%d/6(%.0f%%) ", w, cnt[w], 100*float64(cnt[w])/6)
	}
	fmt.Println()
}

// v3Attributions exécute le pipeline weaponv3 complet et imprime l'arme attribuée
// à chaque kill de JGtm (pour comparer à la couverture mesurée).
func v3Attributions(full string, roster []uint64, chunks []chunk) {
	var ci []weaponv3.ChunkInput
	for _, c := range chunks {
		ci = append(ci, weaponv3.ChunkInput{
			Index: c.index, Data: c.data, StartMS: c.startMS,
			DurationMS: c.durMS, ChunkType: c.chunkType,
		})
	}
	var kills []analysis.Kill
	for _, k := range oracleKills {
		kills = append(kills, analysis.Kill{MatchID: full, XUID: jgtmXUID, TimeMS: k})
	}
	in := weaponv3.V3Input{MatchID: full, Kills: kills, RosterXuids: roster, Chunks: ci}
	attrs := weaponv3.BuildV3Attributions(in)
	for _, a := range attrs {
		if a.XUID != jgtmXUID {
			continue
		}
		name := "(none)"
		if a.HighWeaponID != nil {
			name = weaponv3.WeaponName(*a.HighWeaponID)
			if name == "" {
				name = fmt.Sprintf("high=%08x", *a.HighWeaponID)
			}
		}
		fmt.Printf("KILL @%dms → %s [path=%s conf=%s signal=%s]\n",
			a.TimeMS, name, a.AttributionPath, a.Confidence, a.SourceSignal)
	}
}

// deepDiagnostic : par kill, montre les fires (toute arme) de pi=acurtis dans
// ±3000ms et l'activité globale par pi dans ±1500ms.
func deepDiagnostic(fires []analysis.FireEvent, acurtisPI int) {
	// distribution globale des pi sur TOUS les fires
	gd := map[int]int{}
	for _, f := range fires {
		gd[f.PlayerIndex]++
	}
	fmt.Printf("distribution pi de TOUS les fires (%d): %s\n\n", len(fires), fmtDist(gd))

	for _, k := range oracleKills {
		fmt.Printf("KILL @%dms\n", k)
		// fires du pi=acurtis(2) dans ±3000ms
		var mine []analysis.FireEvent
		for _, f := range fires {
			if f.PlayerIndex != acurtisPI {
				continue
			}
			if abs(int(f.TimestampMS)-k) <= 3000 {
				mine = append(mine, f)
			}
		}
		fmt.Printf("  pi=%d (acurtis JGtm) fires dans ±3000ms: %d", acurtisPI, len(mine))
		if len(mine) > 0 {
			sort.Slice(mine, func(i, j int) bool { return mine[i].TimestampMS < mine[j].TimestampMS })
			var d []string
			for _, f := range mine {
				d = append(d, fmt.Sprintf("%.0f:%s", f.TimestampMS, shortName(f.WeaponName)))
			}
			fmt.Printf(" [%s]", joinComma(d))
		}
		fmt.Println()
		// activité par pi dans la fenêtre serrée AVANT le kill (kill-1500 .. kill)
		win := map[int][]string{}
		for _, f := range fires {
			ts := int(f.TimestampMS)
			if ts > k || ts < k-1500 {
				continue
			}
			win[f.PlayerIndex] = append(win[f.PlayerIndex], shortName(f.WeaponName))
		}
		fmt.Printf("  fires AVANT le kill [k-1500..k] par pi: ")
		for _, p := range keys(countMap(win)) {
			fmt.Printf("pi%d{%s} ", p, joinComma(win[p]))
		}
		fmt.Println()
	}
}

func countMap(m map[int][]string) map[int]int {
	out := map[int]int{}
	for k, v := range m {
		out[k] = len(v)
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// analyzeLayout scanne les fires sous (layout, relax3), affiche la distribution de
// pi des Sidekick et mesure la couverture des 6 kills sous les 2 hypothèses.
func analyzeLayout(chunks []chunk, name string, layout weaponv3.FirePiLayout, relax3 bool, acurtisPI int) {
	fires := scanAllFires(chunks, layout, relax3)
	sk := sidekickFires(fires)
	fmt.Printf("──────── layout=%s relax3=%v ────────\n", name, relax3)
	fmt.Printf("fires total=%d, Sidekick(high=f408190f)=%d\n", len(fires), len(sk))
	piDist := piDistribution(sk)
	fmt.Printf("distribution pi des Sidekick: %s\n", fmtDist(piDist))

	// Ensemble empirique des pi qui portent réellement des Sidekick (hypothèse b).
	empPIs := keys(piDist)

	// Hypothèse (a) : fire pi == acurtis pi du tueur.
	fmt.Printf("  Hyp.(a) fire_pi == acurtis_pi(JGtm)=%d :\n", acurtisPI)
	covA := coverage(sk, []int{acurtisPI})
	printCoverage(covA)

	// Hypothèse (b) : le tueur est CELUI qui porte les Sidekick → on autorise tout
	// pi présent dans la distribution Sidekick (réconciliation : si un seul kill =
	// Frag Parfait Sidekick, alors le ou les pi Sidekick contiennent le tueur).
	fmt.Printf("  Hyp.(b) fire_pi ∈ pi-empiriques-Sidekick %v :\n", empPIs)
	covB := coverage(sk, empPIs)
	printCoverage(covB)
	fmt.Println()
}

// coverageResult — par kill : la fenêtre minimale qui trouve un fire + l'arme.
type coverageResult struct {
	killMS   int
	foundIn  map[int]bool // window -> trouvé ?
	lastFire map[int]*fireHit
}

type fireHit struct {
	tsMS float64
	pi   int
	high uint32
	name string
}

// coverage mesure, pour chaque kill oracle, le dernier Sidekick (pi ∈ allowedPIs)
// AVANT le kill, par fenêtre.
func coverage(sk []skFire, allowedPIs []int) []coverageResult {
	allowed := map[int]bool{}
	for _, p := range allowedPIs {
		allowed[p] = true
	}
	out := make([]coverageResult, 0, len(oracleKills))
	for _, k := range oracleKills {
		cr := coverageResult{killMS: k, foundIn: map[int]bool{}, lastFire: map[int]*fireHit{}}
		for _, w := range windows {
			var best *skFire
			for i := range sk {
				f := &sk[i]
				if !allowed[f.pi] {
					continue
				}
				ts := int(f.tsMS)
				if ts > k || ts < k-w {
					continue
				}
				if best == nil || f.tsMS > best.tsMS {
					best = f
				}
			}
			if best != nil {
				cr.foundIn[w] = true
				cr.lastFire[w] = &fireHit{tsMS: best.tsMS, pi: best.pi, high: best.high, name: best.name}
			}
		}
		out = append(out, cr)
	}
	return out
}

func printCoverage(res []coverageResult) {
	// Compte de couverture par fenêtre.
	cnt := map[int]int{}
	for _, r := range res {
		for _, w := range windows {
			if r.foundIn[w] {
				cnt[w]++
			}
		}
	}
	for _, w := range windows {
		fmt.Printf("    fenêtre %4dms : %d/6 kills couverts", w, cnt[w])
		// détail des fires trouvés
		var det []string
		for _, r := range res {
			if fh := r.lastFire[w]; fh != nil {
				det = append(det, fmt.Sprintf("k%d→pi%d@%.0f(%s)", r.killMS, fh.pi, fh.tsMS, shortName(fh.name)))
			}
		}
		if len(det) > 0 {
			fmt.Printf("  [%s]", joinComma(det))
		}
		fmt.Println()
	}
}

// ── helpers fires ───────────────────────────────────────────────────────────

type skFire struct {
	tsMS float64
	pi   int
	high uint32
	name string
}

func scanAllFires(chunks []chunk, layout weaponv3.FirePiLayout, relax3 bool) []analysis.FireEvent {
	useV2 := layout == weaponv3.FirePi4High && !relax3
	var fires []analysis.FireEvent
	for _, c := range chunks {
		if c.chunkType != 2 {
			continue
		}
		est := weaponv3.USEstimator(c.data, c.startMS)
		if useV2 {
			fires = append(fires, analysis.ScanFireEventsB5(c.data, est)...)
			continue
		}
		fires = append(fires, weaponv3.ScanFireEventsV3(c.data, est, layout, relax3)...)
	}
	return fires
}

// sidekickFires garde les fire-events dont le high-32 == Sidekick (0xf408190f).
func sidekickFires(fires []analysis.FireEvent) []skFire {
	var out []skFire
	for _, f := range fires {
		high := uint32(0)
		for i := 0; i < 4; i++ {
			high = (high << 8) | uint32(f.WeaponBytes[i])
		}
		if high != sidekickHi {
			continue
		}
		out = append(out, skFire{tsMS: f.TimestampMS, pi: f.PlayerIndex, high: high, name: f.WeaponName})
	}
	return out
}

func piDistribution(sk []skFire) map[int]int {
	d := map[int]int{}
	for _, f := range sk {
		d[f.pi]++
	}
	return d
}

func keys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func fmtDist(d map[int]int) string {
	ks := keys(d)
	parts := make([]string, 0, len(ks))
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("pi%d×%d", k, d[k]))
	}
	if len(parts) == 0 {
		return "(aucun)"
	}
	return joinComma(parts)
}

// ── chargement chunks / roster / match-id ───────────────────────────────────

func loadChunks() ([]chunk, error) {
	mfPath := filepath.Join(cacheDir, "film_manifests", matchShort+".json")
	raw, err := os.ReadFile(mfPath)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	var mf struct {
		Chunks []struct {
			Index      int `json:"index"`
			ChunkType  int `json:"chunk_type"`
			StartMS    int `json:"start_ms"`
			DurationMS int `json:"duration_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, err
	}
	var out []chunk
	for _, c := range mf.Chunks {
		p := filepath.Join(cacheDir, "film_chunks", matchShort, fmt.Sprintf("chunk_%02d.bin", c.Index))
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			continue
		}
		out = append(out, chunk{
			index: c.Index, data: decompress(b),
			startMS: c.StartMS, durMS: c.DurationMS, chunkType: c.ChunkType,
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

func gameplayDatas(chunks []chunk) [][]byte {
	var out [][]byte
	for _, c := range chunks {
		if c.chunkType == 2 {
			out = append(out, c.data)
		}
	}
	return out
}

func openRO() (*sql.DB, error) {
	return sql.Open("duckdb", sharedDB+"?access_mode=read_only")
}

func resolveFull(ctx context.Context) (string, error) {
	db, err := openRO()
	if err != nil {
		return "", err
	}
	defer db.Close()
	var full string
	err = db.QueryRowContext(ctx,
		`SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY match_id LIMIT 1`,
		matchShort+"%").Scan(&full)
	return full, err
}

func loadRoster(ctx context.Context, full string) ([]uint64, error) {
	db, err := openRO()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx,
		`SELECT xuid FROM match_participants WHERE match_id = ? AND team_id IS NOT NULL`, full)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uint64
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			continue
		}
		s := x
		if len(s) > 6 && s[:5] == "xuid(" && s[len(s)-1] == ')' {
			s = s[5 : len(s)-1]
		}
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out, rows.Err()
}

func printRosterPI(roster []uint64, m map[uint64]int) {
	type kv struct {
		x  uint64
		pi int
		ok bool
	}
	var rows []kv
	for _, x := range roster {
		pi, ok := m[x]
		rows = append(rows, kv{x, pi, ok})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ok != rows[j].ok {
			return rows[i].ok
		}
		return rows[i].pi < rows[j].pi
	})
	for _, r := range rows {
		if r.ok {
			fmt.Printf("[pi%d:…%d] ", r.pi, r.x%1000000)
		} else {
			fmt.Printf("[?:…%d] ", r.x%1000000)
		}
	}
}

// ── petites utils string (évite strings import lourd côté lisibilité) ────────

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

func shortName(n string) string {
	if len(n) > 14 {
		return n[:14]
	}
	return n
}
