//go:build ignore

// diag_expected_kd — passe d'inspection + validation pour le modèle expected K/D
// local Halo 5 (D2-v2). Prédicteur sans fuite = expected_win_prob (pré-match,
// stocké par match dans le player DB via LUSR v2). Cible = kills / deaths réels
// (shared match_participants). On confirme d'abord le prérequis (win_prob
// peuplé + tailles d'échantillon), puis on fit kills~win_prob / deaths~win_prob
// par mode avec split train/test et on compare à la baseline « moyenne ».
//
// Lancer (depuis apps/go-api) :
//
//	go run ./cmd/diag_expected_kd/main.go <base_titre> [gamertag...]
//
// ex: go run ./cmd/diag_expected_kd/main.go ../../data/titles/halo_5
package main

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

type sample struct {
	player   string
	mode     string
	playlist string
	wp       float64
	rating   float64
	kills    float64
	deaths   float64
	assists  float64
	dmgDealt float64
	durSec   float64
}

func main() {
	base := "../../data/titles/halo_5"
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	players := []string{"Chocoboflor", "JGtm", "Madina97294", "XxDaemonGamerxX"}
	if len(os.Args) > 2 {
		players = os.Args[2:]
	}
	shared := base + "/warehouse/shared_matches_v2.duckdb"

	db, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", shared)); err != nil {
		fmt.Println("attach shared:", err)
		os.Exit(1)
	}

	var all []sample
	for _, gt := range players {
		pdb := base + "/players/" + gt + "/stats.duckdb"
		if _, statErr := os.Stat(pdb); statErr != nil {
			fmt.Printf("[%s] SKIP — pas de DB (%s)\n", gt, pdb)
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS pl (READ_ONLY)", pdb)); err != nil {
			fmt.Printf("[%s] attach: %v\n", gt, err)
			continue
		}

		var xuid string
		if err := db.QueryRow("SELECT xuid FROM shared.xuid_aliases WHERE lower(gamertag)=lower(?) LIMIT 1", gt).Scan(&xuid); err != nil {
			fmt.Printf("[%s] xuid introuvable: %v\n", gt, err)
			db.Exec("DETACH pl")
			continue
		}

		// Couverture win_prob dans le player DB.
		var msrTotal, wpNonNull int
		_ = db.QueryRow("SELECT COUNT(*) FROM pl.match_skill_rank_latest").Scan(&msrTotal)
		if err := db.QueryRow("SELECT COUNT(*) FROM pl.match_skill_rank_latest WHERE expected_win_prob IS NOT NULL").Scan(&wpNonNull); err != nil {
			fmt.Printf("[%s] expected_win_prob ABSENT du schéma: %v\n", gt, err)
			db.Exec("DETACH pl")
			continue
		}

		rows, err := db.Query(`
			SELECT COALESCE(r.game_variant_name,'__unknown__'),
			       COALESCE(msr.playlist_group,'__none__'),
			       CAST(msr.expected_win_prob AS DOUBLE),
			       CAST(COALESCE(msr.rating_value,0) AS DOUBLE),
			       CAST(mp.kills AS DOUBLE),
			       CAST(mp.deaths AS DOUBLE),
			       CAST(COALESCE(mp.assists,0) AS DOUBLE),
			       CAST(COALESCE(mp.damage_dealt,0) AS DOUBLE),
			       CAST(COALESCE(r.duration_seconds,0) AS DOUBLE)
			FROM pl.match_skill_rank_latest msr
			JOIN shared.match_participants mp ON mp.match_id = msr.match_id AND mp.xuid = ?
			JOIN shared.match_registry r ON r.match_id = msr.match_id
			WHERE msr.expected_win_prob IS NOT NULL
			  AND mp.kills IS NOT NULL AND mp.deaths IS NOT NULL
		`, xuid)
		if err != nil {
			fmt.Printf("[%s] load: %v\n", gt, err)
			db.Exec("DETACH pl")
			continue
		}
		n := 0
		for rows.Next() {
			var s sample
			if err := rows.Scan(&s.mode, &s.playlist, &s.wp, &s.rating, &s.kills, &s.deaths, &s.assists, &s.dmgDealt, &s.durSec); err == nil {
				s.player = gt
				all = append(all, s)
				n++
			}
		}
		rows.Close()
		fmt.Printf("[%s] xuid=%s  msr_latest=%d  win_prob_non_null=%d  samples_joints=%d\n", gt, xuid, msrTotal, wpNonNull, n)
		db.Exec("DETACH pl")
	}

	fmt.Printf("\n=== TOTAL échantillons : %d ===\n", len(all))
	if len(all) == 0 {
		fmt.Println("Aucun échantillon (win_prob absent ou join vide) → prérequis NON satisfait.")
		return
	}

	wpSel := func(s sample) float64 { return s.wp }
	ratSel := func(s sample) float64 { return s.rating }
	kSel := func(s sample) float64 { return s.kills }
	dSel := func(s sample) float64 { return s.deaths }
	aSel := func(s sample) float64 { return s.assists }
	ddSel := func(s sample) float64 { return s.dmgDealt }

	// Distributions (variance = condition nécessaire pour qu'un prédicteur serve).
	fmt.Println("\n=== Distributions ===")
	printDist("win_prob", all, wpSel)
	printDist("rating  ", all, ratSel)
	printDist("kills   ", all, kSel)
	printDist("deaths  ", all, dSel)

	// Mystère du mode : quelles colonnes sont peuplées ?
	fmt.Println("\n=== Colonnes de mode/playlist (top valeurs) ===")
	printGroups("game_variant_name", all, func(s sample) string { return s.mode })
	printGroups("playlist_group", all, func(s sample) string { return s.playlist })

	// Corrélations de Pearson — DÉTERMINISTE (aucun split, aucun hasard) : mesure
	// directe du lien linéaire. C'est l'anti-coïncidence.
	fmt.Println("\n=== Corrélations de Pearson (pooled, tous joueurs) ===")
	fmt.Printf("  corr(win_prob, kills)  = %+.3f\n", pearson(all, wpSel, kSel))
	fmt.Printf("  corr(rating,   kills)  = %+.3f   <- inter-joueurs (les bons fraggent plus)\n", pearson(all, ratSel, kSel))
	fmt.Printf("  corr(win_prob, deaths) = %+.3f\n", pearson(all, wpSel, dSel))
	fmt.Printf("  corr(rating,   deaths) = %+.3f\n", pearson(all, ratSel, dSel))

	// L'ASYMÉTRIE : prédicteurs INTRA-match (stats du même match) vs PRÉ-match
	// (skill). C'est pour ça qu'assists se modélise (depuis frags/dégâts) mais pas
	// kills depuis le skill. Test INTRA-joueur (corr moyenne sur les 4 joueurs).
	fmt.Println("\n=== ASYMÉTRIE intra-match vs pré-match (corr INTRA-joueur, moyenne 4 joueurs) ===")
	fmt.Printf("  PRÉ-match  corr(rating,   kills)   = %+.3f   (faible → kills pas prédictible du skill)\n", avgPerPlayerCorr(all, ratSel, kSel))
	fmt.Printf("  INTRA-match corr(dmg_dealt, kills)  = %+.3f   (FORT → kills suit les dégâts du match)\n", avgPerPlayerCorr(all, ddSel, kSel))
	fmt.Printf("  INTRA-match corr(kills,     assists)= %+.3f   (assists suit les frags du match)\n", avgPerPlayerCorr(all, kSel, aSel))
	fmt.Printf("  INTRA-match corr(dmg_dealt, assists)= %+.3f\n", avgPerPlayerCorr(all, ddSel, aSel))
	fmt.Printf("  INTRA-match corr(dmg_dealt, deaths) = %+.3f   (les morts : faible partout)\n", avgPerPlayerCorr(all, ddSel, dSel))
	fmt.Println("  → assists/kills se prédisent depuis les DÉGÂTS du match (stat en aval), pas le skill.")
	fmt.Println("    Mais « kills ~ dégâts » = c'est le RENDEMENT (qu'on a déjà), pas un « attendu » pré-match.")

	// MATCH LENGTH — le facteur que TrueSkill2 inclut (counts × durée) et que
	// j'avais oublié. Un match plus long → plus de kills attendus.
	durSel := func(s sample) float64 { return s.durSec }
	withDur := make([]sample, 0, len(all))
	for _, s := range all {
		if s.durSec > 60 {
			withDur = append(withDur, s)
		}
	}
	fmt.Printf("\n=== DURÉE du match — facteur TrueSkill2 (n avec durée=%d/%d) ===\n", len(withDur), len(all))
	if len(withDur) > 50 {
		printDist("duration_s", withDur, durSel)
		fmt.Printf("  corr(durée, kills)  pooled=%+.3f  intra-joueur=%+.3f\n", pearson(withDur, durSel, kSel), avgPerPlayerCorr(withDur, durSel, kSel))
		fmt.Printf("  corr(durée, deaths) pooled=%+.3f  intra-joueur=%+.3f\n", pearson(withDur, durSel, dSel), avgPerPlayerCorr(withDur, durSel, dSel))
		// Modèle TrueSkill2-like : count ~ rating + durée (2 prédicteurs, split random).
		if mr, mb, ok := fit2Validate(withDur, ratSel, durSel, kSel); ok {
			fmt.Printf("  MULTIVAR kills ~ rating + durée  : RMSE=%.2f vs baseline=%.2f → %s\n", mr, mb, verdict(mr, mb))
		}
		if dr, db, ok := fit2Validate(withDur, ratSel, durSel, dSel); ok {
			fmt.Printf("  MULTIVAR deaths ~ rating + durée : RMSE=%.2f vs baseline=%.2f → %s\n", dr, db, verdict(dr, db))
		}
	} else {
		fmt.Println("  duration_seconds non peuplé pour H5 (peu/pas de durée) → facteur indisponible.")
	}

	// Fits pooled, 2 prédicteurs × 2 cibles.
	fmt.Println("\n=== Fits hors-échantillon (RMSE modèle vs baseline=moyenne) ===")
	report("kills ~ win_prob", all, wpSel, kSel)
	report("kills ~ rating  ", all, ratSel, kSel)
	report("deaths ~ win_prob", all, wpSel, dSel)
	report("deaths ~ rating  ", all, ratSel, dSel)

	// Per playlist_group (au cas où le pooled masque un signal par mode).
	byPl := map[string][]sample{}
	for _, s := range all {
		byPl[s.playlist] = append(byPl[s.playlist], s)
	}
	pls := make([]string, 0, len(byPl))
	for p := range byPl {
		pls = append(pls, p)
	}
	sort.Slice(pls, func(i, j int) bool { return len(byPl[pls[i]]) > len(byPl[pls[j]]) })
	fmt.Println("\n=== Par playlist_group (kills~win_prob / kills~rating) ===")
	for _, p := range pls {
		ss := byPl[p]
		if len(ss) < 40 {
			continue
		}
		wpR, wpB, ok1 := fitValidate(ss, wpSel, kSel)
		raR, raB, ok2 := fitValidate(ss, ratSel, kSel)
		c1, c2 := "n<seuil", "n<seuil"
		if ok1 {
			c1 = fmt.Sprintf("RMSE=%.2f base=%.2f %s", wpR, wpB, verdict(wpR, wpB))
		}
		if ok2 {
			c2 = fmt.Sprintf("RMSE=%.2f base=%.2f %s", raR, raB, verdict(raR, raB))
		}
		fmt.Printf("%-22s n=%4d | win_prob: %-26s | rating: %-26s\n", trunc(p, 22), len(ss), c1, c2)
	}

	// DÉCISIF : kills~rating INTRA-joueur. Si ça bat la baseline par joueur, le
	// modèle rating apporte vs la moyenne perso (suit la trajectoire de skill).
	// Sinon, le +22% pooled était inter-joueurs → la moyenne perso suffit.
	byPlayer := map[string][]sample{}
	for _, s := range all {
		byPlayer[s.player] = append(byPlayer[s.player], s)
	}
	fmt.Println("\n=== DÉCISIF — kills~rating INTRA-joueur (vs moyenne perso) ===")
	for _, gt := range []string{"Chocoboflor", "JGtm", "Madina97294", "XxDaemonGamerxX"} {
		ss := byPlayer[gt]
		if len(ss) < 40 {
			fmt.Printf("  %-16s n=%d (trop peu)\n", gt, len(ss))
			continue
		}
		r, b, ok := fitValidate(ss, ratSel, kSel)
		rr, rb, _ := fitValidateRandom(ss, ratSel, kSel)
		if !ok {
			continue
		}
		corr := pearson(ss, ratSel, kSel)
		fmt.Printf("  %-16s n=%4d | corr(rating,kills)=%+.3f | RMSE chrono=%.2f/%.2f rand=%.2f/%.2f → %s\n",
			gt, len(ss), corr, r, b, rr, rb, verdict(r, b))
	}

	fmt.Println("\nVerdict : corr(rating,kills) INTRA-joueur proche de 0 = rating n'explique pas la")
	fmt.Println("variance des frags d'un même joueur (split chrono ET random donnent pareil). Le")
	fmt.Println("+22% pooled vient de la corr inter-joueurs (rating sépare les joueurs, pas leurs matchs).")
}

// pearson : corrélation linéaire entre x(s) et y(s). Déterministe.
func pearson(ss []sample, x, y func(sample) float64) float64 {
	n := float64(len(ss))
	if n < 2 {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for _, s := range ss {
		xv, yv := x(s), y(s)
		sx += xv
		sy += yv
		sxx += xv * xv
		syy += yv * yv
		sxy += xv * yv
	}
	num := n*sxy - sx*sy
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den < 1e-12 {
		return 0
	}
	return num / den
}

// avgPerPlayerCorr : corrélation x↔y calculée PAR joueur puis moyennée (isole
// l'effet intra-joueur, sans le biais inter-joueurs du pooled).
func avgPerPlayerCorr(ss []sample, x, y func(sample) float64) float64 {
	byP := map[string][]sample{}
	for _, s := range ss {
		byP[s.player] = append(byP[s.player], s)
	}
	var sum float64
	var n int
	for _, grp := range byP {
		if len(grp) < 40 {
			continue
		}
		sum += pearson(grp, x, y)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// fitValidateRandom : comme fitValidate mais split ALÉATOIRE (déterministe via
// seed fixe) — contrôle que le résultat n'est pas un artefact du split chrono.
func fitValidateRandom(ss []sample, x, y func(sample) float64) (float64, float64, bool) {
	cp := make([]sample, len(ss))
	copy(cp, ss)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	return fitValidate(cp, x, y)
}

// fit2Validate : OLS y = a + b*x1 + c*x2, split aléatoire 70/30, RMSE test vs
// baseline=moyenne. Le modèle TrueSkill2-like (count ~ skill + durée).
func fit2Validate(ss []sample, x1, x2, y func(sample) float64) (float64, float64, bool) {
	if len(ss) < 30 {
		return 0, 0, false
	}
	cp := make([]sample, len(ss))
	copy(cp, ss)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	cut := len(cp) * 7 / 10
	train, test := cp[:cut], cp[cut:]
	if len(test) < 5 {
		return 0, 0, false
	}
	var n, sx1, sx2, sy, s11, s22, s12, s1y, s2y float64
	n = float64(len(train))
	for _, s := range train {
		a1, a2, yy := x1(s), x2(s), y(s)
		sx1 += a1
		sx2 += a2
		sy += yy
		s11 += a1 * a1
		s22 += a2 * a2
		s12 += a1 * a2
		s1y += a1 * yy
		s2y += a2 * yy
	}
	a, b, c, ok := solve3(n, sx1, sx2, sx1, s11, s12, sx2, s12, s22, sy, s1y, s2y)
	if !ok {
		return 0, 0, false
	}
	mean := sy / n
	var seM, seB float64
	for _, s := range test {
		yy := y(s)
		pred := a + b*x1(s) + c*x2(s)
		seM += (yy - pred) * (yy - pred)
		seB += (yy - mean) * (yy - mean)
	}
	nt := float64(len(test))
	return math.Sqrt(seM / nt), math.Sqrt(seB / nt), true
}

// solve3 : système 3×3 par règle de Cramer. Retourne (a,b,c,ok).
func solve3(a11, a12, a13, a21, a22, a23, a31, a32, a33, b1, b2, b3 float64) (float64, float64, float64, bool) {
	det := a11*(a22*a33-a23*a32) - a12*(a21*a33-a23*a31) + a13*(a21*a32-a22*a31)
	if math.Abs(det) < 1e-9 {
		return 0, 0, 0, false
	}
	d1 := b1*(a22*a33-a23*a32) - a12*(b2*a33-a23*b3) + a13*(b2*a32-a22*b3)
	d2 := a11*(b2*a33-a23*b3) - b1*(a21*a33-a23*a31) + a13*(a21*b3-b2*a31)
	d3 := a11*(a22*b3-b2*a32) - a12*(a21*b3-b2*a31) + b1*(a21*a32-a22*a31)
	return d1 / det, d2 / det, d3 / det, true
}

func report(name string, ss []sample, x, y func(sample) float64) {
	r, b, ok := fitValidate(ss, x, y)
	if !ok {
		fmt.Printf("%-18s : n<seuil\n", name)
		return
	}
	fmt.Printf("%-18s : RMSE=%.3f vs baseline=%.3f → %s\n", name, r, b, verdict(r, b))
}

func printDist(name string, ss []sample, sel func(sample) float64) {
	if len(ss) == 0 {
		return
	}
	mn, mx, sum := math.Inf(1), math.Inf(-1), 0.0
	for _, s := range ss {
		v := sel(s)
		mn = math.Min(mn, v)
		mx = math.Max(mx, v)
		sum += v
	}
	mean := sum / float64(len(ss))
	varSum := 0.0
	for _, s := range ss {
		d := sel(s) - mean
		varSum += d * d
	}
	sd := math.Sqrt(varSum / float64(len(ss)))
	fmt.Printf("  %s  min=%.2f  mean=%.2f  max=%.2f  sd=%.2f\n", name, mn, mean, mx, sd)
}

func printGroups(name string, ss []sample, sel func(sample) string) {
	cnt := map[string]int{}
	for _, s := range ss {
		cnt[sel(s)]++
	}
	keys := make([]string, 0, len(cnt))
	for k := range cnt {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return cnt[keys[i]] > cnt[keys[j]] })
	fmt.Printf("  %s (%d distinctes) :", name, len(keys))
	for i, k := range keys {
		if i >= 6 {
			fmt.Printf(" …")
			break
		}
		fmt.Printf(" %s=%d", trunc(k, 18), cnt[k])
	}
	fmt.Println()
}

// fitValidate : split 70/30 ordonné, fit y~1+wp sur train (OLS simple), RMSE sur
// test. Baseline = prédire la moyenne(train) sur test. Retourne (rmseModel,
// rmseBaseline, ok). ok=false si trop peu d'échantillons.
func fitValidate(ss []sample, x, y func(sample) float64) (float64, float64, bool) {
	const minN = 20
	if len(ss) < minN {
		return 0, 0, false
	}
	cut := len(ss) * 7 / 10
	train, test := ss[:cut], ss[cut:]
	if len(test) < 3 {
		return 0, 0, false
	}
	// OLS simple y = a + b*x sur train.
	var sx, sy, sxx, sxy float64
	n := float64(len(train))
	for _, s := range train {
		xv := x(s)
		yy := y(s)
		sx += xv
		sy += yy
		sxx += xv * xv
		sxy += xv * yy
	}
	mean := sy / n
	denom := n*sxx - sx*sx
	a, b := mean, 0.0
	if math.Abs(denom) > 1e-9 {
		b = (n*sxy - sx*sy) / denom
		a = (sy - b*sx) / n
	}
	var seModel, seBase float64
	for _, s := range test {
		yy := y(s)
		pred := a + b*x(s)
		seModel += (yy - pred) * (yy - pred)
		seBase += (yy - mean) * (yy - mean)
	}
	nt := float64(len(test))
	return math.Sqrt(seModel / nt), math.Sqrt(seBase / nt), true
}

func verdict(model, base float64) string {
	if base <= 1e-9 {
		return "?"
	}
	gain := (base - model) / base * 100
	switch {
	case gain >= 5:
		return fmt.Sprintf("(+%.0f%% signal)", gain)
	case gain <= -5:
		return fmt.Sprintf("(PIRE %.0f%%)", gain)
	default:
		return "(~ moyenne)"
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
