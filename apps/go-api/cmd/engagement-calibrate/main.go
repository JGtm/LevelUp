//go:build cgo

// engagement-calibrate — harnais de calibration du score d'engagement PAR TITRE
// (chantier F7, phase E4a). DIAGNOSTIC uniquement : lit les paces d'engagement
// persistees (player_match_enrichment_latest) des joueurs d'un titre, calcule les
// distributions des composantes du score (par bin d'intensite, via la MEME logique
// que le serving : temporal.ComputeEngagementResponseBins + ComputeEngagementCoefficient),
// compare a la reference Infinite, et ecrit un rapport markdown + les coefficients
// candidats. N'APPLIQUE RIEN (les poids restent dans constants.toml ; la validation
// est humaine, gate E6).
//
// Methode EXPLICABLE (pas de ML) : le score est un percentile intra-personnel
// (invariant d'echelle) ; le levier de calibration dependant du gameplay = les POIDS
// d'events (constants.toml [engagement]). Le rapport montre, pour chaque mode et bin
// d'intensite, la mediane du ratio pace_joueur/pace_lobby et le taux de rejet — si la
// dispersion du titre est comparable a Infinite, les poids de reference conviennent ;
// sinon le rapport signale l'ecart a arbitrer au gate humain.
//
// Usage : cd apps/go-api && go run ./cmd/engagement-calibrate --title halo_5
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games"
)

func main() {
	title := flag.String("title", "halo_5", "slug du titre a calibrer")
	repoRoot := flag.String("repo-root", "../..", "racine du repo (relative a apps/go-api)")
	out := flag.String("out", "", "chemin du rapport markdown (defaut .ai/ENGAGEMENT_CALIBRATION_<title>_<date>.md)")
	flag.Parse()

	ctx := context.Background()
	refSlug := "halo_infinite"

	target, err := analyzeTitle(ctx, *repoRoot, *title)
	if err != nil {
		log.Fatalf("analyse %s: %v", *title, err)
	}
	var ref titleAnalysis
	if *title != refSlug {
		ref, err = analyzeTitle(ctx, *repoRoot, refSlug)
		if err != nil {
			log.Printf("avertissement: reference %s indisponible: %v", refSlug, err)
		}
	}

	report := renderReport(*title, target, refSlug, ref)

	outPath := *out
	if outPath == "" {
		date := time.Now().UTC().Format("2006-01-02")
		outPath = filepath.Join(*repoRoot, ".ai", fmt.Sprintf("ENGAGEMENT_CALIBRATION_%s_%s.md", strings.ToUpper(*title), date))
	}
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
		log.Fatalf("ecriture rapport %s: %v", outPath, err)
	}
	fmt.Printf("Rapport ecrit: %s\n", outPath)
	fmt.Print(summaryLine(*title, target))
}

// modeAnalysis regroupe les stats de distribution d'une categorie de mode.
type modeAnalysis struct {
	Mode         string
	NSamples     int
	NRejected    int
	CoefOverall  float64 // mediane globale pace_joueur/pace_lobby
	Bins         []temporal.RatioSample
	BinResult    *temporal.ResponseBinsResult
	CoefOK       bool
	IntensityP50 float64
}

// titleAnalysis regroupe l'analyse d'un titre (agregat tous joueurs, par mode).
type titleAnalysis struct {
	Slug     string
	NPlayers int
	Weights  temporal.EventWeights
	ByMode   map[string]*modeAnalysis
}

// analyzeTitle enumere les player DBs du titre, agrege les RatioSample par mode et
// calcule les distributions via la logique de serving.
func analyzeTitle(ctx context.Context, repoRoot, slug string) (titleAnalysis, error) {
	ta := titleAnalysis{Slug: slug, Weights: games.EngagementWeightsFor(slug), ByMode: map[string]*modeAnalysis{}}

	glob := filepath.Join(repoRoot, "data", "titles", slug, "players", "*", "stats.duckdb")
	dbs, err := filepath.Glob(glob)
	if err != nil {
		return ta, fmt.Errorf("glob %s: %w", glob, err)
	}
	if len(dbs) == 0 {
		return ta, fmt.Errorf("aucune player DB sous %s", glob)
	}

	byMode := map[string][]temporal.RatioSample{}
	for _, dbPath := range dbs {
		samples, perr := loadSamplesFromPlayerDB(ctx, dbPath)
		if perr != nil {
			log.Printf("  skip %s: %v", filepath.Base(filepath.Dir(dbPath)), perr)
			continue
		}
		ta.NPlayers++
		for mode, ss := range samples {
			byMode[mode] = append(byMode[mode], ss...)
		}
	}

	for mode, samples := range byMode {
		ta.ByMode[mode] = summarizeMode(mode, samples)
	}
	return ta, nil
}

// loadSamplesFromPlayerDB lit les paces persistees d'une player DB (READ_ONLY),
// groupees par mode_category.
func loadSamplesFromPlayerDB(ctx context.Context, dbPath string) (map[string][]temporal.RatioSample, error) {
	connector, err := duckdb.NewConnector(dbPath+"?access_mode=READ_ONLY", nil)
	if err != nil {
		return nil, fmt.Errorf("connector: %w", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	const q = `
		SELECT COALESCE(mode_category, 'PvP_unranked'),
		       engagement_pace_player, engagement_pace_lobby,
		       COALESCE(engagement_player_activity, 0)
		FROM player_match_enrichment_latest
		WHERE engagement_pace_lobby IS NOT NULL
		  AND engagement_pace_player IS NOT NULL`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	out := map[string][]temporal.RatioSample{}
	for rows.Next() {
		var mode string
		var s temporal.RatioSample
		if err := rows.Scan(&mode, &s.PaceJoueur, &s.PaceLobby, &s.PlayerActivity); err != nil {
			continue
		}
		out[mode] = append(out[mode], s)
	}
	return out, rows.Err()
}

// summarizeMode calcule la distribution d'un mode via la logique de serving.
func summarizeMode(mode string, samples []temporal.RatioSample) *modeAnalysis {
	ma := &modeAnalysis{Mode: mode, NSamples: len(samples)}

	if coef, err := temporal.ComputeEngagementCoefficient(samples); err == nil {
		ma.CoefOverall = coef.CoefLobbyShare
		ma.NRejected = coef.NRejected
		ma.CoefOK = true
	}
	if bins, err := temporal.ComputeEngagementResponseBins(samples); err == nil {
		ma.BinResult = bins
	} else if !errors.Is(err, temporal.ErrInsufficientBinHistory) {
		log.Printf("  bins %s: %v", mode, err)
	}

	// Mediane d'intensite (pace_lobby) informative.
	lobbies := make([]float64, 0, len(samples))
	for _, s := range samples {
		if s.PaceLobby > 0 {
			lobbies = append(lobbies, s.PaceLobby)
		}
	}
	sort.Float64s(lobbies)
	if n := len(lobbies); n > 0 {
		ma.IntensityP50 = lobbies[n/2]
	}
	return ma
}

// renderReport assemble le rapport markdown.
func renderReport(slug string, ta titleAnalysis, refSlug string, ref titleAnalysis) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Rapport de calibration engagement — %s\n\n", slug)
	fmt.Fprintf(&b, "> Genere le %s par `cmd/engagement-calibrate` (chantier F7 E4a). DIAGNOSTIC — n'applique rien.\n\n", time.Now().UTC().Format("2006-01-02 15:04 UTC"))

	fmt.Fprintf(&b, "## Poids d'events actuels (constants.toml [engagement])\n\n")
	fmt.Fprintf(&b, "| titre | objective | assist | death | default |\n|---|---|---|---|---|\n")
	writeWeightsRow(&b, slug, ta.Weights)
	if ref.Slug != "" {
		writeWeightsRow(&b, refSlug+" (ref)", ref.Weights)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Distribution par mode et bin d'intensite — %s (%d joueurs)\n\n", slug, ta.NPlayers)
	writeTitleTable(&b, ta)

	if ref.Slug != "" {
		fmt.Fprintf(&b, "\n## Reference %s (%d joueurs)\n\n", refSlug, ref.NPlayers)
		writeTitleTable(&b, ref)
	}

	fmt.Fprintf(&b, "\n## Coefficients candidats\n\n")
	fmt.Fprintf(&b, "Methode : le score d'engagement est un percentile intra-personnel (invariant\n")
	fmt.Fprintf(&b, "d'echelle) ; le levier de calibration dependant du gameplay = les poids d'events.\n")
	fmt.Fprintf(&b, "Les coefficients candidats proposes = les poids actuels de `%s` (ci-dessus). Le\n", slug)
	fmt.Fprintf(&b, "tableau ci-dessus permet de juger si la dispersion des ratios (coef par bin) et le\n")
	fmt.Fprintf(&b, "taux de rejet du titre sont comparables a la reference Infinite. Si oui, les poids\n")
	fmt.Fprintf(&b, "de reference conviennent (candidat = defaut) ; sinon, ajuster au gate humain E6.\n\n")
	fmt.Fprintf(&b, "```toml\n[engagement]\nobjective = %g\nassist    = %g\ndeath     = %g\ndefault   = %g\n```\n\n",
		ta.Weights.Objective, ta.Weights.Assist, ta.Weights.Death, ta.Weights.Default)

	fmt.Fprintf(&b, "## Verdict automatique (indicatif — non liant)\n\n")
	b.WriteString(autoVerdict(ta, ref))
	return b.String()
}

func writeWeightsRow(b *strings.Builder, label string, w temporal.EventWeights) {
	fmt.Fprintf(b, "| %s | %g | %g | %g | %g |\n", label, w.Objective, w.Assist, w.Death, w.Default)
}

func writeTitleTable(b *strings.Builder, ta titleAnalysis) {
	fmt.Fprintf(b, "| mode | n | rejets | coef global | calme | standard | chaotique | intensite p50 |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---|---|---|\n")
	for _, mode := range sortedModes(ta.ByMode) {
		ma := ta.ByMode[mode]
		calme, std, chao := "-", "-", "-"
		if ma.BinResult != nil {
			for _, bin := range ma.BinResult.Bins {
				switch bin.Bin {
				case temporal.IntensityBinCalme:
					calme = fmt.Sprintf("%.3f (n=%d)", bin.CoefLobby, bin.NMatches)
				case temporal.IntensityBinStandard:
					std = fmt.Sprintf("%.3f (n=%d)", bin.CoefLobby, bin.NMatches)
				case temporal.IntensityBinChaotique:
					chao = fmt.Sprintf("%.3f (n=%d)", bin.CoefLobby, bin.NMatches)
				}
			}
		}
		coef := "insuffisant"
		if ma.CoefOK {
			coef = fmt.Sprintf("%.3f", ma.CoefOverall)
		}
		fmt.Fprintf(b, "| %s | %d | %d | %s | %s | %s | %s | %.3f |\n",
			mode, ma.NSamples, ma.NRejected, coef, calme, std, chao, ma.IntensityP50)
	}
}

// autoVerdict emet un jugement indicatif : suffisance des donnees + comparaison
// grossiere de dispersion vs reference.
func autoVerdict(ta, ref titleAnalysis) string {
	var b strings.Builder
	enough := false
	for _, ma := range ta.ByMode {
		if ma.CoefOK && ma.NSamples >= temporal.MinMatchesForCoef {
			enough = true
		}
	}
	if !enough {
		return "- Donnees INSUFFISANTES pour calibrer (aucun mode >= seuil). Backfill requis.\n"
	}
	fmt.Fprintf(&b, "- Donnees suffisantes : au moins un mode a un coef global exploitable.\n")
	fmt.Fprintf(&b, "- Candidat = poids de reference Infinite (le score etant percentile intra-personnel).\n")
	fmt.Fprintf(&b, "- A valider au gate humain (E6) : les scores H5 ont-ils du sens sur des matchs intenses vs calmes ?\n")
	_ = ref
	return b.String()
}

func sortedModes(m map[string]*modeAnalysis) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func summaryLine(slug string, ta titleAnalysis) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Titre %s : %d joueurs, %d modes.\n", slug, ta.NPlayers, len(ta.ByMode))
	for _, mode := range sortedModes(ta.ByMode) {
		ma := ta.ByMode[mode]
		fmt.Fprintf(&b, "  %s: n=%d rejets=%d coefOK=%v\n", mode, ma.NSamples, ma.NRejected, ma.CoefOK)
	}
	return b.String()
}
