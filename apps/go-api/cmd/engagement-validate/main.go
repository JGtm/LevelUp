// Command engagement-validate — outil CLI Phase 0 du plan engagement.
//
// Reference : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §1 (Phase 0)
//
// Valide les hypotheses critiques avant deploiement production :
//
//	H1 : corr(engagement_score, performance_score) < 0.5  (axes decoreles)
//	H3 : stabilite du coef_team_share dans le temps
//	H5 : combien de joueurs ont < 30 matchs sur leur categorie principale
//
// Pre-requis : la migration Phase 2 doit etre appliquee ET les
// engagement_scores deja calcules pour le joueur cible (via sync ou backfill).
//
// Usage :
//
//	go run ./cmd/engagement-validate --player MonGT
//	go run ./cmd/engagement-validate --player MonGT --mode-category PvP_ranked
//	go run ./cmd/engagement-validate --player MonGT --json   (machine-readable)
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

type validationReport struct {
	Player          string  `json:"player"`
	ModeCategory    string  `json:"mode_category"`
	NMatches        int     `json:"n_matches"`
	H1Correlation   float64 `json:"h1_correlation_engagement_vs_perf"`
	H1Pass          bool    `json:"h1_pass_below_0_5"`
	H3CoefStability float64 `json:"h3_coef_stability_ratio"`
	H3Pass          bool    `json:"h3_pass_ratio_below_1_3"`
	H5InsufHistory  bool    `json:"h5_insufficient_history"`
	NHistoryMatches int     `json:"n_history_matches"`
}

func main() {
	var (
		playerGT     = flag.String("player", "", "Gamertag du joueur a valider (requis)")
		modeCategory = flag.String("mode-category", "PvP_ranked", "Categorie de mode a valider")
		repoRoot     = flag.String("repo-root", "../..", "Racine du repo (pour resolution data/)")
		asJSON       = flag.Bool("json", false, "Sortie JSON machine-readable")
	)
	flag.Parse()

	if *playerGT == "" {
		fmt.Fprintln(os.Stderr, "Erreur : --player requis")
		flag.Usage()
		os.Exit(2)
	}

	playerDBPath := filepath.Join(*repoRoot, "data", "titles", "halo_infinite", "players", *playerGT, "stats.duckdb")
	if _, err := os.Stat(playerDBPath); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur : DB joueur introuvable %s\n", playerDBPath)
		os.Exit(1)
	}

	db, err := sql.Open("duckdb", playerDBPath+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur ouverture DB : %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	report := validationReport{
		Player:       *playerGT,
		ModeCategory: *modeCategory,
	}

	if !engagementColumnsExist(db) {
		fmt.Fprintln(os.Stderr, "Erreur : la migration engagement Phase 2 n'a pas ete appliquee sur cette DB.")
		os.Exit(1)
	}

	// H1 : correlation engagement vs perf
	pairs, err := loadScorePairs(db, *modeCategory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur load scores : %v\n", err)
		os.Exit(1)
	}
	report.NMatches = len(pairs)

	if len(pairs) >= 30 {
		report.H1Correlation = pearsonCorrelation(pairs)
		report.H1Pass = math.Abs(report.H1Correlation) < 0.5
	} else {
		report.H5InsufHistory = true
	}

	// H3 : coef stability (compare 2 fenetres glissantes)
	stability, err := coefStability(db, *modeCategory)
	if err == nil {
		report.H3CoefStability = stability
		report.H3Pass = stability < 1.3 && stability > 0
	}

	histN, _ := countMatchesWithEngagement(db, *modeCategory)
	report.NHistoryMatches = histN

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(report)
		return
	}

	printHumanReport(report)
}

// loadScorePairs charge les couples (engagement_score, performance_score)
// pour la categorie demandee, avec les deux scores non-NULL.
func loadScorePairs(db *sql.DB, modeCategory string) ([][2]float64, error) {
	rows, err := db.Query(`
		SELECT engagement_score, performance_score
		FROM player_match_enrichment
		WHERE mode_category = ?
		  AND engagement_score IS NOT NULL
		  AND performance_score IS NOT NULL
	`, modeCategory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([][2]float64, 0)
	for rows.Next() {
		var eng, perf float64
		if err := rows.Scan(&eng, &perf); err != nil {
			continue
		}
		pairs = append(pairs, [2]float64{eng, perf})
	}
	return pairs, rows.Err()
}

// pearsonCorrelation calcule le coefficient de correlation de Pearson entre
// les deux series. Retourne 0 si moins de 2 points (pas de variance possible).
func pearsonCorrelation(pairs [][2]float64) float64 {
	n := len(pairs)
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for _, p := range pairs {
		sumX += p[0]
		sumY += p[1]
		sumXY += p[0] * p[1]
		sumX2 += p[0] * p[0]
		sumY2 += p[1] * p[1]
	}
	nf := float64(n)
	num := nf*sumXY - sumX*sumY
	denX := nf*sumX2 - sumX*sumX
	denY := nf*sumY2 - sumY*sumY
	if denX <= 0 || denY <= 0 {
		return 0
	}
	return num / math.Sqrt(denX*denY)
}

// coefStability compare la mediane des 100 derniers matchs vs les 100 prededents.
// Retourne le ratio max/min (cible < 1.3).
func coefStability(db *sql.DB, modeCategory string) (float64, error) {
	rows, err := db.Query(`
		SELECT engagement_score_brut
		FROM player_match_enrichment
		WHERE mode_category = ?
		  AND engagement_score_brut IS NOT NULL
		ORDER BY match_id DESC
		LIMIT 200
	`, modeCategory)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	values := make([]float64, 0, 200)
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err == nil {
			values = append(values, v)
		}
	}
	if len(values) < 100 {
		return 0, fmt.Errorf("pas assez d'historique : %d (besoin >= 100)", len(values))
	}

	medRecent := median(values[:100])
	medOlder := median(values[100:])
	maxV := math.Max(math.Abs(medRecent), math.Abs(medOlder))
	minV := math.Min(math.Abs(medRecent), math.Abs(medOlder))
	if minV < 0.001 {
		return 0, fmt.Errorf("mediane trop proche de zero pour ratio stable")
	}
	return maxV / minV, nil
}

// median calcule la mediane d'une slice (modifie l'ordre — copie en amont si besoin).
func median(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	cp := make([]float64, len(vs))
	copy(cp, vs)
	// tri par insertion (suffisant pour <= 100 elements)
	for i := 1; i < len(cp); i++ {
		v := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > v {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = v
	}
	if len(cp)%2 == 0 {
		return (cp[len(cp)/2-1] + cp[len(cp)/2]) / 2
	}
	return cp[len(cp)/2]
}

// countMatchesWithEngagement compte les matchs avec engagement_score non null.
func countMatchesWithEngagement(db *sql.DB, modeCategory string) (int, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE mode_category = ? AND engagement_score IS NOT NULL
	`, modeCategory).Scan(&n)
	return n, err
}

// engagementColumnsExist verifie que la migration Phase 2 a ete appliquee.
func engagementColumnsExist(db *sql.DB) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_score'
	`).Scan(&count)
	return err == nil && count > 0
}

// printHumanReport affiche le rapport en format lisible.
func printHumanReport(r validationReport) {
	fmt.Println("===== Engagement validation =====")
	fmt.Printf("Player          : %s\n", r.Player)
	fmt.Printf("Mode category   : %s\n", r.ModeCategory)
	fmt.Printf("Matches sample  : %d\n", r.NMatches)
	fmt.Printf("History total   : %d\n", r.NHistoryMatches)
	fmt.Println()

	fmt.Println("--- H1 : corr(engagement, performance) doit etre < 0.5 ---")
	if r.H5InsufHistory {
		fmt.Println("  SKIP : moins de 30 matchs en commun (engagement + perf scores).")
	} else {
		status := "FAIL"
		if r.H1Pass {
			status = "PASS"
		}
		fmt.Printf("  Correlation : %.3f -> %s\n", r.H1Correlation, status)
		if !r.H1Pass {
			fmt.Println("  ATTENTION : engagement et performance pourraient etre redondants.")
			fmt.Println("  Action recommandee : revoir le modele de pace_attendu (cf §13 du plan, baselines conditionnelles).")
		}
	}

	fmt.Println()
	fmt.Println("--- H3 : stabilite du coef_team_share (ratio < 1.3 souhaite) ---")
	if r.H3CoefStability == 0 {
		fmt.Println("  SKIP : pas assez d'historique pour comparer 2 fenetres glissantes.")
	} else {
		status := "FAIL"
		if r.H3Pass {
			status = "PASS"
		}
		fmt.Printf("  Ratio (mediane recente / mediane ancienne) : %.3f -> %s\n", r.H3CoefStability, status)
	}

	fmt.Println()
	if r.H1Pass && r.H3Pass {
		fmt.Println("VERDICT : OK pour deploiement. Les hypotheses critiques sont satisfaites.")
	} else if !r.H5InsufHistory {
		fmt.Println("VERDICT : a revoir avant deploiement production.")
	} else {
		fmt.Println("VERDICT : insuffisant pour valider — sync plus de matchs et relancer.")
	}
}
