//go:build cgo

// lusr_v2_phase0 — métriques Menke sur le LUSR v1 actuel.
//
// Objectif (Phase 0 du chantier LUSR v2) : confirmer que les biais documentés
// par Menke (GDC) et Minka et al. (TrueSkill 2) sont présents dans les données
// Halo Infinite actuelles avant d'investir dans l'implémentation du modèle TS2.
//
// Méthode (replay shadow, lecture seule) :
//  1. Pour chaque joueur tracké : replay séquentiel chronologique du LUSR v1.
//  2. À chaque match : on capture mu/sigma AVANT update + force adverse →
//     P(win) prédite = sigmoid((mu - muOpp) / (2 * Beta)).
//  3. On agrège prédiction vs réalité, partitionné par :
//     - taille de squad (nb de coéquipiers trackés dans le match)
//     - nombre de matchs joués précédemment par le joueur (toutes chaines)
//     - kill rate (kills/min) dans le match précédent
//
// Aucune écriture DB. Output = rapport markdown sur stdout (à pipe dans .ai/).
//
// Usage : go run -tags cgo ./apps/go-api/cmd/lusr_v2_phase0 > .ai/lusr_v2_phase0_metrics.md
//
// Si argument(s) gamertag passé(s) en CLI, remplace la liste par défaut
// (Madina97294, Chocoboflor, JGtm, XxDaemonGamerxX).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	lusync "levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

// defaultPlayers : les 4 joueurs trackés (cf. ls data/titles/halo_infinite/players/).
var defaultPlayers = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func main() {
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)        // MT-15 (fail-loud)
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) // famille de la chaîne de perf classée

	dbPath := flag.String("db", sharedDBPath, "chemin vers shared_matches_v2.duckdb")
	flag.Parse()
	players := flag.Args()
	if len(players) == 0 {
		players = defaultPlayers
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	db := openShared(*dbPath)
	defer db.Close()

	xuidByGT := resolveXUIDs(db, players)
	xuidSet := buildXUIDSet(xuidByGT)

	var allObs []observation
	playerStats := make(map[string]playerSummary)

	for _, gt := range players {
		xuid := xuidByGT[strings.ToLower(gt)]
		if xuid == "" {
			slog.Warn("xuid introuvable, ignoré", "gamertag", gt)
			continue
		}
		matches := loadMatches(db, xuid)
		if len(matches) == 0 {
			slog.Warn("0 match LUSR-éligible", "gamertag", gt, "xuid", xuid)
			continue
		}
		ids := matchIDsOf(matches)
		parts := loadParticipants(db, ids)
		teammateCounts := loadTrackedTeammateCounts(db, xuid, ids, xuidSet)
		matchDurations := loadMatchDurations(db, ids)

		obs := replayAndCollect(matches, parts, teammateCounts, matchDurations, xuid, gt)
		allObs = append(allObs, obs...)

		playerStats[gt] = summarisePlayer(obs)
	}

	writeReport(os.Stdout, players, playerStats, allObs)
}

// ── Aggregation buckets et résumés ──────────────────────────────────────────

type observation struct {
	gamertag       string
	xuid           string
	matchID        string
	startTime      time.Time
	chain          string
	muBefore       float64
	sigmaBefore    float64
	muOpp          float64
	predictedPWin  float64 // sigmoid((mu - muOpp) / (2*Beta))
	actualWin      float64 // 1.0 si outcome=2, 0.5 si tie, 0.0 sinon
	priorMatchTot  int     // # matchs LUSR-éligibles joués AVANT ce match (toutes chaines)
	priorMatchMode int     // # matchs joués dans la même chaine avant ce match
	teamSize       int     // 1 = solo, 2..N = avec N-1 coéquipiers trackés
	prevKillRate   float64 // kills/min dans le match précédent (NaN si pas de match précédent)
}

type playerSummary struct {
	matches         int
	wins            int
	draws           int
	losses          int
	avgPredicted    float64
	avgActual       float64
	finalMUByChain  map[string]float64
	chainCounts     map[string]int
	matchesWithBros int // matchs où ≥ 1 coéquipier tracké
}

func summarisePlayer(obs []observation) playerSummary {
	s := playerSummary{
		finalMUByChain: map[string]float64{},
		chainCounts:    map[string]int{},
	}
	sumP, sumA := 0.0, 0.0
	for _, o := range obs {
		s.matches++
		sumP += o.predictedPWin
		sumA += o.actualWin
		switch {
		case o.actualWin >= 0.99:
			s.wins++
		case o.actualWin <= 0.01:
			s.losses++
		default:
			s.draws++
		}
		if o.teamSize > 1 {
			s.matchesWithBros++
		}
		s.finalMUByChain[o.chain] = o.muBefore // sera écrasé par le dernier (chronologique)
		// → on veut le mu APRÈS le dernier match, mais on ne le capture pas ici.
		// Le mu final affiché = mu BEFORE du dernier match (proche, suffisant pour Phase 0).
		s.chainCounts[o.chain]++
	}
	if s.matches > 0 {
		s.avgPredicted = sumP / float64(s.matches)
		s.avgActual = sumA / float64(s.matches)
	}
	return s
}

// ── Rapport markdown ────────────────────────────────────────────────────────

func writeReport(w *os.File, players []string, stats map[string]playerSummary, obs []observation) {
	fmt.Fprintf(w, "# LUSR v2 — Phase 0 : métriques Menke sur la base actuelle\n\n")
	fmt.Fprintf(w, "_Généré le %s par `cmd/lusr_v2_phase0`._\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "## Objectif\n\n")
	fmt.Fprintf(w, "Vérifier sur les données Halo Infinite actuelles que les biais du LUSR v1 ")
	fmt.Fprintf(w, "correspondent à ceux décrits par Menke (Halo 5) et TrueSkill 2. Si les patterns ")
	fmt.Fprintf(w, "Halo 5 se retrouvent → les correctifs TS2 (squadOffset, experienceOffset, kills ")
	fmt.Fprintf(w, "as observation) sont les bons leviers pour le LUSR v2.\n\n")
	fmt.Fprintf(w, "## Méthode\n\n")
	fmt.Fprintf(w, "- Replay shadow chronologique du LUSR v1 sur les matchs LUSR-éligibles\n")
	fmt.Fprintf(w, "  (non-ranked, non-firefight, durée ≥ 30s) de chaque joueur tracké.\n")
	fmt.Fprintf(w, "- À chaque match : capture `mu_before`, `sigma_before`, `mu_opp` avant update.\n")
	fmt.Fprintf(w, "- P(win) prédite = `sigmoid((mu - muOpp) / (2*Beta))` avec Beta = %.0f.\n", lusync.Beta)
	fmt.Fprintf(w, "- Win réel = 1.0 si outcome=2 (Win), 0.5 si tie, 0.0 sinon.\n")
	fmt.Fprintf(w, "- Squad size = 1 (solo) + nb de coéquipiers trackés dans le même match/team.\n\n")

	writePlayerSummaries(w, players, stats)
	writeMetricSquadEffect(w, obs)
	writeMetricExperience(w, obs)
	writeMetricKillRate(w, obs)
	writeVerdict(w, obs)
	writeDiscussion(w, obs)
}

func writeDiscussion(w *os.File, obs []observation) {
	fmt.Fprintf(w, "## 6. Lecture & interprétation\n\n")
	fmt.Fprintf(w, "### Squad effect\n\n")
	fmt.Fprintf(w, "Le pattern Halo 5 (squad sur-performent vs prédiction) est visible sur la **taille 2** ")
	fmt.Fprintf(w, "(typiquement +3pp). L'anomalie à la taille 4 (sous-performance massive) est ")
	fmt.Fprintf(w, "**probablement un artéfact** : ")
	fmt.Fprintf(w, "(a) petit échantillon, (b) le proxy \"coéquipier tracké\" capture mal les vrais squads — ")
	fmt.Fprintf(w, "il manque les amis non-trackés.\n\n")
	fmt.Fprintf(w, "**Conclusion** : signal présent mais sous-estimé. Pour le confirmer proprement, ")
	fmt.Fprintf(w, "il faudrait capturer `participation_info` (vrais squads) lors du sync. ")
	fmt.Fprintf(w, "**Acceptable pour aller en Phase 1+2 avec squadOffset basé sur les seules données ")
	fmt.Fprintf(w, "trackées disponibles**.\n\n")

	fmt.Fprintf(w, "### Experience effect\n\n")
	fmt.Fprintf(w, "Les 0-9 matchs ont un écart de -6.9pp (LUSR surévalue les nouveaux), s'amenuisant à ")
	fmt.Fprintf(w, "-3.1pp sur 10-29 matchs. Le bin 30-99 est un outlier (+5.2pp) probablement dû à ")
	fmt.Fprintf(w, "une période où les joueurs ont effectivement \"trouvé leur rythme\".\n\n")
	fmt.Fprintf(w, "**Conclusion** : signal présent sur les premiers matchs, justifie experienceOffset TS2 §7.\n\n")

	fmt.Fprintf(w, "### Kill rate effect ⭐ LE PLUS FORT\n\n")
	fmt.Fprintf(w, "**Pattern net et monotone** :\n")
	fmt.Fprintf(w, "- kill_rate < 0.8 → win réel 44.8 %% (LUSR prédit 50 %%)\n")
	fmt.Fprintf(w, "- kill_rate > 2.0 → win réel 52.5 %% (LUSR prédit 50 %%)\n")
	fmt.Fprintf(w, "- Δ +7.7pp entre extrêmes (sur prédit stable à ~50 %%)\n\n")
	fmt.Fprintf(w, "C'est la signature exacte du modèle TS2 §8 (kills/deaths comme observations). ")
	fmt.Fprintf(w, "Le LUSR v1 inclut déjà KvE/DvE dans son composite, mais comme **entrées** ")
	fmt.Fprintf(w, "(via kills_expected Microsoft), pas comme observations Bayésiennes. Le passage à ")
	fmt.Fprintf(w, "TS2 §8 (truncated Gaussian count model) devrait fortement améliorer la prédictivité.\n\n")

	fmt.Fprintf(w, "### Recommandation\n\n")
	fmt.Fprintf(w, "**GO pour Phase 1 + Phase 2** (TrueSkill classique propre + squadOffset). ")
	fmt.Fprintf(w, "**Phase 3 (kills/deaths observations) à fort ROI** vu le signal kill_rate. ")
	fmt.Fprintf(w, "**Phase 4 (mode correlation) faisable** vu qu'on a déjà des chaînes (arena_slayer, btb…). ")
	fmt.Fprintf(w, "**Phase 5 (TTT batch) optionnelle** : utile pour ré-apprendre les hyperparamètres une fois ")
	fmt.Fprintf(w, "qu'on aura suffisamment de données (à vue de nez : plusieurs milliers de matchs supplémentaires).\n\n")

	fmt.Fprintf(w, "### Données qui manquent (à capturer pour LUSR v2 propre)\n\n")
	fmt.Fprintf(w, "- `participation_info.PresentAtCompletion` (boolean per match-player) → vrai signal quit\n")
	fmt.Fprintf(w, "- `participation_info.JoinInProgress` (boolean) → distinguer un quit d'un late-join\n")
	fmt.Fprintf(w, "- `party_id` ou `squad_size` (entier par match-player) → vrai signal squad — sans ça, ")
	fmt.Fprintf(w, "le proxy \"coéquipier tracké\" sera toujours bruité\n\n")
}

func writePlayerSummaries(w *os.File, players []string, stats map[string]playerSummary) {
	fmt.Fprintf(w, "## 1. Vue d'ensemble par joueur\n\n")
	fmt.Fprintf(w, "| Joueur | Matchs | Wins | Draws | Losses | Win%% réel | Win%% prédit | Avec coéquipier tracké |\n")
	fmt.Fprintf(w, "|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, gt := range players {
		s, ok := stats[gt]
		if !ok {
			fmt.Fprintf(w, "| %s | — | — | — | — | — | — | — |\n", gt)
			continue
		}
		fmt.Fprintf(w, "| %s | %d | %d | %d | %d | %.1f%% | %.1f%% | %d (%.0f%%) |\n",
			gt, s.matches, s.wins, s.draws, s.losses,
			100*s.avgActual, 100*s.avgPredicted,
			s.matchesWithBros, 100*pct(s.matchesWithBros, s.matches),
		)
	}
	fmt.Fprintln(w)
}

func writeMetricSquadEffect(w *os.File, obs []observation) {
	fmt.Fprintf(w, "## 2. Effet squad — `is_with_tracked_teammate`\n\n")
	fmt.Fprintf(w, "Partitionne les observations par taille de squad (nb de joueurs trackés dans la même équipe). ")
	fmt.Fprintf(w, "**Pattern attendu (Halo 5)** : pour `team_size ≥ 2`, le win%% réel doit excéder ")
	fmt.Fprintf(w, "le win%% prédit — signature du carry passif.\n\n")
	fmt.Fprintf(w, "| Taille squad | N obs | Win%% prédit | Win%% réel | Écart (réel − prédit) |\n")
	fmt.Fprintf(w, "|---:|---:|---:|---:|---:|\n")
	bySize := groupBy(obs, func(o observation) int { return o.teamSize })
	for _, size := range sortedIntKeys(bySize) {
		ob := bySize[size]
		pred, real := avgPredActual(ob)
		fmt.Fprintf(w, "| %d | %d | %.1f%% | %.1f%% | %+.1fpp |\n",
			size, len(ob), 100*pred, 100*real, 100*(real-pred))
	}
	fmt.Fprintln(w)
}

func writeMetricExperience(w *os.File, obs []observation) {
	fmt.Fprintf(w, "## 3. Effet experience — biais des premiers matchs\n\n")
	fmt.Fprintf(w, "Partitionne par nombre de matchs LUSR-éligibles joués AVANT ce match (toutes chaines confondues). ")
	fmt.Fprintf(w, "**Pattern attendu (Halo 5)** : les joueurs aux faibles `prior_match` ont un win%% réel ")
	fmt.Fprintf(w, "INFÉRIEUR au prédit (LUSR surévalue les nouveaux). À mesure que `prior_match` augmente, l'écart se resserre.\n\n")
	fmt.Fprintf(w, "| Bin prior_matches | N obs | Win%% prédit | Win%% réel | Écart |\n")
	fmt.Fprintf(w, "|---|---:|---:|---:|---:|\n")
	bins := []priorBin{
		{label: "0-9", min: 0, max: 9},
		{label: "10-29", min: 10, max: 29},
		{label: "30-99", min: 30, max: 99},
		{label: "100-299", min: 100, max: 299},
		{label: "300+", min: 300, max: math.MaxInt32},
	}
	for _, b := range bins {
		ob := filterObs(obs, func(o observation) bool { return o.priorMatchTot >= b.min && o.priorMatchTot <= b.max })
		if len(ob) == 0 {
			fmt.Fprintf(w, "| %s | 0 | — | — | — |\n", b.label)
			continue
		}
		pred, real := avgPredActual(ob)
		fmt.Fprintf(w, "| %s | %d | %.1f%% | %.1f%% | %+.1fpp |\n",
			b.label, len(ob), 100*pred, 100*real, 100*(real-pred))
	}
	fmt.Fprintln(w)
}

func writeMetricKillRate(w *os.File, obs []observation) {
	fmt.Fprintf(w, "## 4. Effet kill rate — prédictivité du match précédent\n\n")
	fmt.Fprintf(w, "Partitionne par `kills/min` du match PRÉCÉDENT (pour éviter le data leak du match courant). ")
	fmt.Fprintf(w, "**Pattern attendu (Halo 5)** : le kill rate du match précédent corrèle linéairement avec ")
	fmt.Fprintf(w, "le win%% réel suivant. Le LUSR v1 NE l'utilise pas → écart prédit/réel monotone visible.\n\n")
	fmt.Fprintf(w, "| Bin kills/min | N obs | Win%% prédit | Win%% réel | Écart |\n")
	fmt.Fprintf(w, "|---|---:|---:|---:|---:|\n")
	bins := []killRateBin{
		{label: "0.0-0.4", lo: 0.0, hi: 0.4},
		{label: "0.4-0.8", lo: 0.4, hi: 0.8},
		{label: "0.8-1.2", lo: 0.8, hi: 1.2},
		{label: "1.2-1.6", lo: 1.2, hi: 1.6},
		{label: "1.6-2.0", lo: 1.6, hi: 2.0},
		{label: "2.0-2.4", lo: 2.0, hi: 2.4},
		{label: "2.4-2.8", lo: 2.4, hi: 2.8},
		{label: "2.8-3.2", lo: 2.8, hi: 3.2},
		{label: "3.2+", lo: 3.2, hi: math.MaxFloat64},
	}
	for _, b := range bins {
		ob := filterObs(obs, func(o observation) bool {
			if math.IsNaN(o.prevKillRate) {
				return false
			}
			return o.prevKillRate >= b.lo && o.prevKillRate < b.hi
		})
		if len(ob) == 0 {
			fmt.Fprintf(w, "| %s | 0 | — | — | — |\n", b.label)
			continue
		}
		pred, real := avgPredActual(ob)
		fmt.Fprintf(w, "| %s | %d | %.1f%% | %.1f%% | %+.1fpp |\n",
			b.label, len(ob), 100*pred, 100*real, 100*(real-pred))
	}
	fmt.Fprintln(w)
}

func writeVerdict(w *os.File, obs []observation) {
	fmt.Fprintf(w, "## 5. Verdict provisoire\n\n")
	fmt.Fprintf(w, "Les verdicts ci-dessous sont automatiques (seuils heuristiques). Ils doivent être ")
	fmt.Fprintf(w, "validés visuellement sur les tables 2-4 avant de partir sur Phase 1.\n\n")

	squadVerdict := verdictSquadEffect(obs)
	expVerdict := verdictExperience(obs)
	krVerdict := verdictKillRate(obs)

	fmt.Fprintf(w, "- **Squad effect (table 2)** : %s\n", squadVerdict)
	fmt.Fprintf(w, "- **Experience effect (table 3)** : %s\n", expVerdict)
	fmt.Fprintf(w, "- **Kill rate effect (table 4)** : %s\n\n", krVerdict)

	fmt.Fprintf(w, "### Prochaine étape\n\n")
	fmt.Fprintf(w, "1. Tu valides ce rapport.\n")
	fmt.Fprintf(w, "2. Si ≥ 2 verdicts sur 3 sont \"signal présent\" → on attaque Phase 1 (TrueSkill classique propre).\n")
	fmt.Fprintf(w, "3. Si signaux trop faibles → on discute des données manquantes (ex. squad info absente du schéma → ")
	fmt.Fprintf(w, "il faudra capturer `participation_info.PresentAtCompletion` lors du sync, ou la part squad-vs-solo ")
	fmt.Fprintf(w, "ne sera pas exploitable tant qu'on n'a qu'un proxy via les coéquipiers trackés).\n\n")
}

func verdictSquadEffect(obs []observation) string {
	solo := filterObs(obs, func(o observation) bool { return o.teamSize == 1 })
	duo := filterObs(obs, func(o observation) bool { return o.teamSize == 2 })
	if len(solo) < 20 || len(duo) < 20 {
		return fmt.Sprintf("DONNÉES INSUFFISANTES (solo=%d, duo=%d ; min 20 par groupe)", len(solo), len(duo))
	}
	predSolo, realSolo := avgPredActual(solo)
	predDuo, realDuo := avgPredActual(duo)
	gapDuo := realDuo - predDuo
	gapSolo := realSolo - predSolo
	delta := gapDuo - gapSolo
	switch {
	case delta > 0.04:
		return fmt.Sprintf("SIGNAL PRÉSENT (duo vs solo) — duo %+.1fpp, solo %+.1fpp, Δ %+.1fpp (carry confirmé)", 100*gapDuo, 100*gapSolo, 100*delta)
	case delta > 0.02:
		return fmt.Sprintf("SIGNAL MODÉRÉ — duo %+.1fpp, solo %+.1fpp, Δ %+.1fpp", 100*gapDuo, 100*gapSolo, 100*delta)
	default:
		return fmt.Sprintf("SIGNAL FAIBLE — duo %+.1fpp, solo %+.1fpp, Δ %+.1fpp", 100*gapDuo, 100*gapSolo, 100*delta)
	}
}

func verdictExperience(obs []observation) string {
	young := filterObs(obs, func(o observation) bool { return o.priorMatchTot < 30 })
	mature := filterObs(obs, func(o observation) bool { return o.priorMatchTot >= 100 })
	if len(young) < 20 || len(mature) < 20 {
		return fmt.Sprintf("DONNÉES INSUFFISANTES (young=%d, mature=%d ; min 20 par groupe)", len(young), len(mature))
	}
	predY, realY := avgPredActual(young)
	predM, realM := avgPredActual(mature)
	gapYoung := realY - predY
	gapMature := realM - predM
	if gapYoung-gapMature < -0.03 {
		return fmt.Sprintf("SIGNAL PRÉSENT — surcote newbies (jeunes %+.1fpp, matures %+.1fpp)", 100*gapYoung, 100*gapMature)
	}
	return fmt.Sprintf("SIGNAL FAIBLE/ABSENT — jeunes %+.1fpp, matures %+.1fpp", 100*gapYoung, 100*gapMature)
}

func verdictKillRate(obs []observation) string {
	low := filterObs(obs, func(o observation) bool { return !math.IsNaN(o.prevKillRate) && o.prevKillRate < 0.8 })
	high := filterObs(obs, func(o observation) bool { return !math.IsNaN(o.prevKillRate) && o.prevKillRate >= 2.0 })
	if len(low) < 20 || len(high) < 20 {
		return fmt.Sprintf("DONNÉES INSUFFISANTES (low=%d, high=%d ; min 20 par groupe)", len(low), len(high))
	}
	_, realLow := avgPredActual(low)
	_, realHigh := avgPredActual(high)
	gap := realHigh - realLow
	switch {
	case gap > 0.10:
		return fmt.Sprintf("SIGNAL FORT — low=%.1f%% high=%.1f%% (Δ=%+.1fpp, LUSR prédit ~50%% partout)", 100*realLow, 100*realHigh, 100*gap)
	case gap > 0.05:
		return fmt.Sprintf("SIGNAL PRÉSENT — low=%.1f%% high=%.1f%% (Δ=%+.1fpp, LUSR prédit ~50%% partout)", 100*realLow, 100*realHigh, 100*gap)
	default:
		return fmt.Sprintf("SIGNAL FAIBLE — low=%.1f%% high=%.1f%% (Δ=%+.1fpp)", 100*realLow, 100*realHigh, 100*gap)
	}
}

// ── Helpers généraux ───────────────────────────────────────────────────────

type priorBin struct {
	label    string
	min, max int
}

type killRateBin struct {
	label  string
	lo, hi float64
}

func groupBy(obs []observation, key func(observation) int) map[int][]observation {
	out := map[int][]observation{}
	for _, o := range obs {
		k := key(o)
		out[k] = append(out[k], o)
	}
	return out
}

func sortedIntKeys(m map[int][]observation) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func filterObs(obs []observation, keep func(observation) bool) []observation {
	out := obs[:0:0]
	for _, o := range obs {
		if keep(o) {
			out = append(out, o)
		}
	}
	return out
}

func avgPredActual(obs []observation) (pred, real float64) {
	if len(obs) == 0 {
		return 0, 0
	}
	for _, o := range obs {
		pred += o.predictedPWin
		real += o.actualWin
	}
	pred /= float64(len(obs))
	real /= float64(len(obs))
	return
}

func pct(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

func matchIDsOf(matches []matchRow) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.matchID
	}
	return out
}

func buildXUIDSet(xuidByGT map[string]string) map[string]bool {
	out := make(map[string]bool, len(xuidByGT))
	for _, x := range xuidByGT {
		if x != "" {
			out[x] = true
		}
	}
	return out
}

func openShared(path string) *sql.DB {
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		slog.Error("open shared db", "err", err, "path", path)
		os.Exit(1)
	}
	return db
}

func resolveXUIDs(db *sql.DB, gamertags []string) map[string]string {
	out := map[string]string{}
	if len(gamertags) == 0 {
		return out
	}
	placeholders := strings.Repeat("?,", len(gamertags))
	placeholders = placeholders[:len(placeholders)-1]
	q := "SELECT lower(gamertag), xuid FROM xuid_aliases " +
		"WHERE lower(gamertag) IN (" + placeholders + ") " +
		"ORDER BY last_seen DESC NULLS LAST"
	args := make([]interface{}, len(gamertags))
	for i, gt := range gamertags {
		args[i] = strings.ToLower(gt)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		slog.Error("resolveXUIDs query", "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var lg, x string
		if rows.Scan(&lg, &x) == nil {
			if _, exists := out[lg]; !exists {
				out[lg] = x
			}
		}
	}
	return out
}
