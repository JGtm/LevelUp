//go:build cgo

// diag_lusr_volatility — étude empirique de la volatilité LUSR v2 (lecture seule).
//
// Objectif : chiffrer, sur les joueurs trackés, l'ampleur réelle des "grosses
// chutes de palier en un match" et ce que chaque protection candidate (a..e)
// apporterait concrètement en termes de niveau affiché + de volatilité.
//
// Source ground-truth : player_skill_state_v2 (append-only, shared DB) donne la
// trajectoire (μ, σ) par match SANS rejouer le modèle TrueSkill. Le palier
// affiché = InferTier(μ) (cf. skill_v2_canonical.go). On reconstruit donc la
// séquence de paliers réelle, puis on rejoue les options sur cette séquence.
//
// Aucune écriture. Usage :
//
//	go run -tags cgo ./apps/go-api/cmd/diag_lusr_volatility [-db <shared.duckdb>]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	skillv2 "levelup/go-api/internal/analysis/skill_v2"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

type player struct {
	gamertag string
	xuid     string
}

var players = []player{
	{"Madina97294", "2533274858283686"},
	{"Chocoboflor", "2535469190789936"},
	{"JGtm", "2533274823110022"},
}

// snap = état (μ,σ) après un match + contexte du match.
type snap struct {
	mu, sigma float64
	exp       int
	matchID   string
	at        time.Time
	quit      bool
	outcome   int
}

func main() {
	dbPath := flag.String("db", sharedDBPath, "chemin vers shared_matches_v2.duckdb")
	flag.Parse()

	db, err := sql.Open("duckdb", *dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "ping (DB verrouillée par air.exe ? copier puis -db copie):", err)
		os.Exit(1)
	}

	ctx := context.Background()
	bnd := skillv2.DefaultTierBoundaries()

	fmt.Println("# LUSR v2 — étude de volatilité & options de protection")
	fmt.Printf("\nGrille tiers (μ→palier), largeur d'un sous-palier en μ :\n")
	printGridWidths(bnd)

	for _, p := range players {
		printPlayerCoverage(ctx, db, p)
		group := dominantGroup(ctx, db, p.xuid)
		if group == "" {
			fmt.Printf("\n## %s — aucun état v2\n", p.gamertag)
			continue
		}
		traj := loadTrajectory(ctx, db, p.xuid, group)
		fmt.Printf("\n## %s — groupe « %s » (%d matchs)\n", p.gamertag, group, len(traj))
		if len(traj) < 3 {
			fmt.Println("trop peu de matchs, skip.")
			continue
		}
		report(traj, bnd)
	}
}

// ── chargement ──────────────────────────────────────────────────────────────

// printPlayerCoverage diagnostique la couverture des données v2 (étendue
// temporelle + total matchs) pour distinguer un régime réel d'un artefact de
// rebuild récent. Compare aussi au total de matchs LUSR-éligibles bruts.
func printPlayerCoverage(ctx context.Context, db *sql.DB, p player) {
	var nStates, nGroups int
	var minAt, maxAt sql.NullTime
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT last_match_id), COUNT(DISTINCT playlist_group),
		       MIN(last_match_at), MAX(last_match_at)
		FROM player_skill_state_v2
		WHERE xuid = ? AND last_match_id IS NOT NULL`, p.xuid).Scan(&nStates, &nGroups, &minAt, &maxAt)

	var nEligible int
	var minM, maxM sql.NullTime
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       MIN(`+analysis.SQLStartTimeCanonical("mr")+`),
		       MAX(`+analysis.SQLStartTimeCanonical("mr")+`)
		FROM match_registry mr JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND COALESCE(mr.is_ranked,FALSE)=FALSE
		  AND COALESCE(mr.is_firefight,FALSE)=FALSE AND mr.start_time IS NOT NULL`, p.xuid).Scan(&nEligible, &minM, &maxM)

	fmt.Printf("\n### %s — couverture v2\n", p.gamertag)
	fmt.Printf("- états v2 : %d matchs / %d groupes | période %s → %s\n",
		nStates, nGroups, fmtT(minAt), fmtT(maxAt))
	fmt.Printf("- matchs LUSR-éligibles bruts (toutes dates) : %d | période %s → %s\n",
		nEligible, fmtT(minM), fmtT(maxM))
}

func fmtT(t sql.NullTime) string {
	if !t.Valid {
		return "—"
	}
	return t.Time.Format("2006-01-02")
}

func dominantGroup(ctx context.Context, db *sql.DB, xuid string) string {
	rows, err := db.QueryContext(ctx, `
		SELECT playlist_group, COUNT(DISTINCT last_match_id) c
		FROM player_skill_state_v2
		WHERE xuid = ? AND last_match_id IS NOT NULL
		GROUP BY 1 ORDER BY c DESC LIMIT 1`, xuid)
	if err != nil {
		return ""
	}
	defer rows.Close()
	var g string
	var c int
	if rows.Next() {
		_ = rows.Scan(&g, &c)
	}
	return g
}

func loadTrajectory(ctx context.Context, db *sql.DB, xuid, group string) []snap {
	rows, err := db.QueryContext(ctx, `
		WITH dedup AS (
			SELECT mu, sigma, experience, last_match_id, last_match_at,
			       ROW_NUMBER() OVER (PARTITION BY last_match_id ORDER BY written_at DESC) rn
			FROM player_skill_state_v2
			WHERE xuid = ? AND playlist_group = ? AND last_match_id IS NOT NULL
		)
		SELECT d.mu, d.sigma, d.experience, d.last_match_id, d.last_match_at,
		       COALESCE(mp.left_in_progress, FALSE), COALESCE(mp.outcome, 0)
		FROM dedup d
		LEFT JOIN match_participants mp
		       ON mp.match_id = d.last_match_id AND mp.xuid = ?
		WHERE d.rn = 1
		ORDER BY d.last_match_at ASC`, xuid, group, xuid)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loadTrajectory:", err)
		return nil
	}
	defer rows.Close()
	var out []snap
	for rows.Next() {
		var s snap
		var at sql.NullTime
		if err := rows.Scan(&s.mu, &s.sigma, &s.exp, &s.matchID, &at, &s.quit, &s.outcome); err != nil {
			continue
		}
		if at.Valid {
			s.at = at.Time
		}
		out = append(out, s)
	}
	return out
}

// ── ordinal palier (0..30) pour mesurer les sauts ───────────────────────────

// tierBase = nb de sous-paliers cumulés avant le tier idx (Onyx compté 1).
func tierBases(bnd []skillv2.TierBoundary) []int {
	bases := make([]int, len(bnd))
	acc := 0
	for i, b := range bnd {
		bases[i] = acc
		n := b.SubTiers
		if n < 1 {
			n = 1
		}
		acc += n
	}
	return bases
}

func tierIdx(mu float64, bnd []skillv2.TierBoundary) int {
	idx := 0
	for i, b := range bnd {
		if mu >= b.MinMu {
			idx = i
		}
	}
	return idx
}

// ordinal global du palier pour un μ donné (monotone croissant en μ).
func ordinal(mu float64, bnd []skillv2.TierBoundary, bases []int) int {
	idx := tierIdx(mu, bnd)
	_, sub := skillv2.InferTier(mu, bnd)
	if sub <= 0 {
		sub = 1
	}
	return bases[idx] + (sub - 1)
}

// ── rapport par joueur ──────────────────────────────────────────────────────

func report(traj []snap, bnd []skillv2.TierBoundary) {
	bases := tierBases(bnd)

	// Trajectoire de référence (telle qu'affichée aujourd'hui = InferTier(μ)).
	ord := make([]int, len(traj))
	for i, s := range traj {
		ord[i] = ordinal(s.mu, bnd, bases)
	}
	last := traj[len(traj)-1]
	fmt.Printf("- Palier final actuel : **%s** (μ=%.3f, σ=%.3f)\n",
		skillv2.FormatTierLabel(last.mu, bnd), last.mu, last.sigma)

	// 1) Δμ par match + volatilité brute.
	var dmu []float64
	for i := 1; i < len(traj); i++ {
		dmu = append(dmu, traj[i].mu-traj[i-1].mu)
	}
	absd := absSorted(dmu)
	fmt.Printf("- |Δμ| par match : médiane=%.3f  p80=%.3f  p95=%.3f  max=%.3f\n",
		pct(absd, 50), pct(absd, 80), pct(absd, 95), pct(absd, 100))

	subW := subTierWidthAt(last.mu, bnd)
	fmt.Printf("- Largeur sous-palier au niveau du joueur ≈ %.3f μ → un match p80 traverse ≈ %.1f sous-paliers\n",
		subW, pct(absd, 80)/nz(subW))

	// 2) "Grosses chutes" sur la séquence affichée actuelle.
	dropsSub, dropsTier, maxDrop := bigDrops(ord, bnd, bases, traj)
	fmt.Printf("- **Chutes actuelles** : %d matchs avec −≥2 sous-paliers, %d matchs avec −≥1 TIER complet, pire chute = %d sous-paliers en 1 match\n",
		dropsSub, dropsTier, -maxDrop)

	// 3) Option (a) — affichage conservatif μ−k·σ.
	fmt.Println("- **(a) Affichage μ−k·σ** (au lieu de μ brut) :")
	for _, k := range []float64{1, 2, 3} {
		oc := make([]int, len(traj))
		for i, s := range traj {
			oc[i] = ordinal(s.mu-k*s.sigma, bnd, bases)
		}
		ds, dt, _ := bigDrops(oc, bnd, bases, traj)
		fmt.Printf("    k=%.0f → palier final %s | chutes −≥2sp:%d  −≥1tier:%d\n",
			k, skillv2.FormatTierLabel(last.mu-k*last.sigma, bnd), ds, dt)
	}

	// 4) Option (b) — hystérésis : montée immédiate, descente ≤1 sous-palier/match.
	disp := make([]int, len(traj))
	disp[0] = ord[0]
	for i := 1; i < len(traj); i++ {
		switch {
		case ord[i] > disp[i-1]:
			disp[i] = ord[i] // promotion immédiate
		case ord[i] < disp[i-1]:
			disp[i] = disp[i-1] - 1 // descente bridée
		default:
			disp[i] = disp[i-1]
		}
	}
	lag := disp[len(disp)-1] - ord[len(ord)-1]
	dsB, dtB, _ := bigDrops(disp, bnd, bases, traj)
	fmt.Printf("- **(b) Hystérésis descente ≤1 sp/match** : chutes −≥2sp:%d  −≥1tier:%d | lag final = %d sous-paliers vs μ brut\n",
		dsB, dtB, lag)

	// 5) Option (c) — grille plus grossière (3 sous-paliers au lieu de 6).
	coarse := halveSubTiers(bnd)
	cb := tierBases(coarse)
	oc := make([]int, len(traj))
	for i, s := range traj {
		oc[i] = ordinal(s.mu, coarse, cb)
	}
	dsC, dtC, _ := bigDrops(oc, coarse, cb, traj)
	recSub := int(math.Floor(subTierFullWidthAt(last.mu, bnd) / nz(pct(absd, 80))))
	fmt.Printf("- **(c) Grille grossière (3 sp/tier)** : chutes −≥2sp:%d  −≥1tier:%d | pour que p80(|Δμ|) tienne dans 1 sp il faudrait ≈ %d sous-paliers max sur ce tier (vs 6)\n",
		dsC, dtC, maxInt(1, recSub))

	// 6) Option (d) — quit penalty.
	var nQuit int
	var sumQuit, sumNon float64
	var nNon int
	for i := 1; i < len(traj); i++ {
		d := traj[i].mu - traj[i-1].mu
		if traj[i].quit {
			nQuit++
			sumQuit += d
		} else {
			nNon++
			sumNon += d
		}
	}
	fmt.Printf("- **(d) Quit penalty** : %d matchs quit (%.0f%%) | Δμ moyen quit=%.3f vs non-quit=%.3f | pénalité brute −1.0/−2.5 μ = %.1f/%.1f sous-paliers d'un coup\n",
		nQuit, 100*float64(nQuit)/float64(len(traj)), avg(sumQuit, nQuit), avg(sumNon, nNon),
		1.0/nz(subW), 2.5/nz(subW))

	// 7) Option (e) — période de placement.
	fmt.Printf("- **(e) Placement** : σ passe sous 1.5 après %s match(s), sous 1.0 après %s ; amplitude μ sur les 10 premiers = %.2f\n",
		firstBelow(traj, 1.5), firstBelow(traj, 1.0), muRangeFirst(traj, 10))
}

// ── métriques ───────────────────────────────────────────────────────────────

// bigDrops compte, sur une séquence d'ordinaux, les matchs perdant ≥2 sous-paliers,
// ceux perdant ≥1 tier complet, et la pire chute (valeur négative).
func bigDrops(ord []int, bnd []skillv2.TierBoundary, bases []int, traj []snap) (dropSub, dropTier, maxDrop int) {
	for i := 1; i < len(ord); i++ {
		d := ord[i] - ord[i-1]
		if d <= -2 {
			dropSub++
		}
		if d < maxDrop {
			maxDrop = d
		}
		// chute d'un tier complet : l'ordinal passe sous la base du tier précédent.
		if tierOf(ord[i], bases) < tierOf(ord[i-1], bases) {
			dropTier++
		}
	}
	return
}

func tierOf(ord int, bases []int) int {
	idx := 0
	for i, b := range bases {
		if ord >= b {
			idx = i
		}
	}
	return idx
}

func subTierWidthAt(mu float64, bnd []skillv2.TierBoundary) float64 {
	idx := tierIdx(mu, bnd)
	n := bnd[idx].SubTiers
	if n < 1 {
		n = 1
	}
	return subTierFullWidthAt(mu, bnd) / float64(n)
}

func subTierFullWidthAt(mu float64, bnd []skillv2.TierBoundary) float64 {
	idx := tierIdx(mu, bnd)
	if idx+1 < len(bnd) {
		return bnd[idx+1].MinMu - bnd[idx].MinMu
	}
	return 1.0 // Onyx ouvert
}

func halveSubTiers(bnd []skillv2.TierBoundary) []skillv2.TierBoundary {
	out := make([]skillv2.TierBoundary, len(bnd))
	copy(out, bnd)
	for i := range out {
		if out[i].SubTiers > 1 {
			out[i].SubTiers = 3
		}
	}
	return out
}

func firstBelow(traj []snap, thr float64) string {
	for i, s := range traj {
		if s.sigma < thr {
			return fmt.Sprintf("%d", i+1)
		}
	}
	return "jamais"
}

func muRangeFirst(traj []snap, n int) float64 {
	if len(traj) < 2 {
		return 0
	}
	if n > len(traj) {
		n = len(traj)
	}
	lo, hi := traj[0].mu, traj[0].mu
	for i := 0; i < n; i++ {
		lo = math.Min(lo, traj[i].mu)
		hi = math.Max(hi, traj[i].mu)
	}
	return hi - lo
}

func printGridWidths(bnd []skillv2.TierBoundary) {
	for i, b := range bnd {
		w := 1.0
		if i+1 < len(bnd) {
			w = bnd[i+1].MinMu - b.MinMu
		}
		n := b.SubTiers
		if n < 1 {
			n = 1
		}
		fmt.Printf("  %-9s μ≥%-5.1f  largeur tier=%-4.1f  sous-palier=%.3f μ\n", b.NameFR, b.MinMu, w, w/float64(n))
	}
}

// ── utilitaires numériques ──────────────────────────────────────────────────

func absSorted(xs []float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = math.Abs(x)
	}
	sort.Float64s(out)
	return out
}

func pct(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := int(p / 100 * float64(len(sorted)-1))
	return sorted[idx]
}

func avg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func nz(v float64) float64 {
	if v == 0 {
		return 1e-9
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
