// cmd/diag_perfsim — simulation OFFLINE et LECTURE SEULE de la note de
// performance (lot 0 du plan .ai/PLAN_PERF_NOTE_OBJECTIFS.md).
//
// Rejoue batchComputePerformanceScores (internal/sync/performance.go) sur les
// données réelles des joueurs suivis, sous deux régimes :
//   - ACTUEL  : chaînes actuelles (ranked unique) + skill.RelativeWeights partout ;
//   - NOUVEAU : ranked scindé en ranked_slayer / ranked_objectif (D-A) + profil de
//     poids objectif avec la métrique ospm (D-C) sur arena_objectif/ranked_objectif.
//
// Outil JETABLE : aucune écriture (DB ouvertes en access_mode=read_only), aucune
// dépendance du code produit sur lui. fmt autorisé (convention cmd/diag_*).
//
// Usage :
//
//	go run ./cmd/diag_perfsim [-data <racine data/>] [-out <rapport.md>]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	titleSlug  = "halo_infinite"
	windowSize = 50

	defaultDataRoot = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data`
	defaultOut      = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-perfnote\.ai\RAPPORT_SIM_PERF_NOTE_2026-08.md`
)

// ospmVariants — poids ospm testés (D-C : 0.12 de départ, ±0.04 de sensibilité).
var ospmVariants = []float64{0.08, 0.12, 0.16}

// ospmReference — variante de référence pour la purge et les tables de décision.
const ospmReference = 0.12

type player struct {
	Gamertag string
	XUID     string
}

var players = []player{
	{Gamertag: "JGtm", XUID: "2533274823110022"},
	{Gamertag: "Chocoboflor", XUID: "2535469190789936"},
	{Gamertag: "Madina97294", XUID: "2533274858283686"},
	{Gamertag: "XxDaemonGamerxX", XUID: "2533274833178266"},
}

// matchRow porte les colonnes de loadHistoryForPerf (performance_helpers.go:153)
// enrichies des champs nécessaires à la simulation (outcome, exclusion, awards).
type matchRow struct {
	MatchID           string
	StartTime         time.Time
	Kills             float64
	Deaths            float64
	Assists           float64
	KDA               float64
	Accuracy          float64
	TimePlayedSeconds float64
	PersonalScore     float64
	DamageDealt       float64
	DamageTaken       float64
	Rank              float64
	TeamMMR           float64
	EnemyMMR          float64
	KillsExpected     float64
	DeathsExpected    float64
	PairName          string
	IsRanked          bool
	IsFirefight       bool
	Outcome           int

	// Dérivés (calculés après chargement, comme loadHistoryForPerf).
	OffensiveConversion float64
	DefensiveResistance float64

	// Hors SQL production.
	Excluded       bool
	PSACovered     bool
	ObjectiveScore float64
}

// playerResult agrège tout ce que le rapport doit rendre pour un joueur.
type playerResult struct {
	Player      player
	Universe    []matchRow
	Scorable    []matchRow
	DNFCount    int
	ExcludedCnt int
	Actual      *regimeResult
	ActualNoDmg *regimeResult
	NewByW      map[float64]*regimeResult
	Stored      []storedRow
	Purge       purgeReport
	Concord     concordance
	PSA         psaStats
}

func main() {
	dataRoot := flag.String("data", defaultDataRoot, "racine du dossier data/ (lecture seule)")
	out := flag.String("out", defaultOut, "chemin du rapport Markdown")
	flag.Parse()

	ctx := context.Background()
	sharedPath := filepath.Join(*dataRoot, "titles", titleSlug, "warehouse", "shared_matches_v2.duckdb")
	sharedDB, err := openRO(sharedPath)
	if err != nil {
		fatal("ouverture shared: %v", err)
	}
	defer sharedDB.Close()

	results := make([]*playerResult, 0, len(players))
	for _, p := range players {
		pr, runErr := runPlayer(ctx, sharedDB, *dataRoot, p)
		if runErr != nil {
			fatal("joueur %s: %v", p.Gamertag, runErr)
		}
		results = append(results, pr)
		printPlayerSummary(pr)
	}

	if err := writeReport(*out, results); err != nil {
		fatal("écriture rapport: %v", err)
	}
	fmt.Printf("rapport écrit : %s\n", *out)
}

// runPlayer charge, annote et simule les deux régimes pour un joueur.
func runPlayer(ctx context.Context, sharedDB *sql.DB, dataRoot string, p player) (*playerResult, error) {
	playerPath := filepath.Join(dataRoot, "titles", titleSlug, "players", p.Gamertag, "stats.duckdb")
	pdb, err := openRO(playerPath)
	if err != nil {
		return nil, err
	}
	defer pdb.Close()

	universe, err := loadUniverse(ctx, sharedDB, p.XUID)
	if err != nil {
		return nil, err
	}
	excluded, err := loadExcludedMatchIDs(ctx, pdb)
	if err != nil {
		return nil, err
	}
	psa, stats, err := loadObjectiveByMatch(ctx, pdb, p.XUID)
	if err != nil {
		return nil, err
	}
	stored, err := loadStoredScores(ctx, pdb)
	if err != nil {
		return nil, err
	}

	res := &playerResult{Player: p, Stored: stored, PSA: stats, NewByW: map[float64]*regimeResult{}}
	annotate(universe, excluded, psa)
	res.Universe = universe
	res.Scorable, res.DNFCount, res.ExcludedCnt = splitScorable(universe)

	res.Actual = runRegime("ACTUEL", res.Scorable, chainCurrent, weightsCurrentFor)
	res.ActualNoDmg = runRegime("ACTUEL-sans-dpm_damage", res.Scorable, chainCurrent, weightsMinusDamageFor)
	for _, w := range ospmVariants {
		res.NewByW[w] = runRegime(fmt.Sprintf("NOUVEAU ospm=%.2f", w), res.Scorable, chainSplit, weightsFor(w))
	}

	res.Purge = buildPurge(res)
	res.Concord = buildConcordance(res)
	return res, nil
}

// annotate reporte exclusions et awards objectif sur l'univers chargé.
func annotate(universe []matchRow, excluded map[string]bool, psa map[string]psaMatch) {
	for i := range universe {
		m := &universe[i]
		m.Excluded = excluded[m.MatchID]
		if info, ok := psa[m.MatchID]; ok {
			m.PSACovered = info.Covered
			m.ObjectiveScore = info.ObjectiveScore
		}
	}
}

// splitScorable applique les deux filtres du batch production : outcome != 4
// (WHERE SQL de loadHistoryForPerf) puis is_excluded (performance.go:271-284).
func splitScorable(universe []matchRow) (scorable []matchRow, dnf, excl int) {
	scorable = make([]matchRow, 0, len(universe))
	for _, m := range universe {
		switch {
		case m.Outcome == 4:
			dnf++
		case m.Excluded:
			excl++
		default:
			scorable = append(scorable, m)
		}
	}
	return scorable, dnf, excl
}

func printPlayerSummary(pr *playerResult) {
	ref := pr.NewByW[ospmReference]
	fmt.Printf("[%-16s] univers=%4d dnf=%3d exclus=%2d scorables=%4d | notes ACTUEL=%4d NOUVEAU=%4d | psa_couverts=%4d | purge=%d\n",
		pr.Player.Gamertag, len(pr.Universe), pr.DNFCount, pr.ExcludedCnt, len(pr.Scorable),
		countScored(pr.Actual), countScored(ref), pr.PSA.MatchesCovered, pr.Purge.Total())
}

func countScored(r *regimeResult) int {
	n := 0
	for _, st := range r.Chains {
		n += st.NScored
	}
	return n
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "diag_perfsim: "+format+"\n", args...)
	os.Exit(1)
}
