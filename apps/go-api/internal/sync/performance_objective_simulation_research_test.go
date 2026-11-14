//go:build research && cgo

package sync

// performance_objective_simulation_research_test.go — SIMULATION de la note de performance à
// objectif (variante D-7, performance_objective_blend.go) sur une COPIE À FROID des bases.
//
// Instrument de recherche, lecture seule : aucune base n'est écrite (les notes ne sont
// calculées qu'en mémoire), aucune révision de notation ne bouge. Il rejoue la boucle de
// batchComputePerformanceScores (mêmes chargements, mêmes fenêtres de 50 matchs par chaîne,
// même seuil de 10 matchs) et, pour chaque match d'une chaîne à objectif, écrit une ligne TSV :
// l'ancienne note recalculée par le code actuel, la note stockée (contrôle), et la nouvelle
// note pour trois valeurs de AlphaMax (0,35 ; 0,50 ; 0,65).
//
// Lancement (depuis apps/go-api) :
//
//	LEVELUP_PERFSIM_ROOT=<copie>/halo_infinite LEVELUP_PERFSIM_OUT=<dossier>/sim.tsv \
//	  go test -tags=research ./internal/sync/ -run TestResearchPerformanceObjectiveSimulation -v -timeout 40m
//
// <copie>/halo_infinite contient `warehouse/` et `players/<gamertag>/stats.duckdb`. Les xuid
// viennent de LEVELUP_PERFSIM_PROFILES (db_profiles.json, section halo_infinite), à défaut de
// xuid_aliases de la base partagée.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/ctxkeys"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/skill"
)

// simAlphaMaxes : les trois valeurs de AlphaMax de la simulation (la décision D-7 est 0,50).
var simAlphaMaxes = []float64{0.35, 0.50, 0.65}

const simHeader = "player\tmatch_id\tstart\tmap\tpair\tchain\tfamily\tkills\tdeaths\tassists\tminutes\t" +
	"team\tshare\tpoints\tteam_points\tindex\tscope\tcombat\told\tstored\tnew35\tnew50\tnew65\talpha50\tpct50\tn_ref"

func openRO(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("ouverture %s : %v", path, err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func simXUIDs(t *testing.T, shared *sql.DB, players []string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if p := os.Getenv("LEVELUP_PERFSIM_PROFILES"); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("profils : %v", err)
		}
		var doc struct {
			Profiles map[string]map[string]struct {
				XUID string `json:"xuid"`
			} `json:"profiles"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("profils : %v", err)
		}
		for gt, pr := range doc.Profiles["halo_infinite"] {
			if pr.XUID != "" {
				out[gt] = pr.XUID
			}
		}
	}
	for _, gt := range players {
		if out[gt] != "" {
			continue
		}
		var x string
		if err := shared.QueryRow(`SELECT xuid FROM xuid_aliases WHERE gamertag = ? ORDER BY last_seen DESC LIMIT 1`, gt).Scan(&x); err == nil {
			out[gt] = x
		}
	}
	return out
}

// simMatchInfo : date, carte, mode et score d'un match, pour la lecture humaine des lignes.
func simMatchInfo(shared *sql.DB) map[string][3]string {
	out := map[string][3]string{}
	rows, err := shared.Query(`SELECT match_id, CAST(start_time AS VARCHAR), COALESCE(map_name, ''), COALESCE(pair_name, '') FROM match_registry`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, st, mp, pn string
		if rows.Scan(&id, &st, &mp, &pn) == nil {
			out[id] = [3]string{st, mp, pn}
		}
	}
	return out
}

func simStoredNotes(player *sql.DB) map[string]float64 {
	out := map[string]float64{}
	rows, err := player.Query(`SELECT match_id, performance_score FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var s float64
		if rows.Scan(&id, &s) == nil {
			out[id] = s
		}
	}
	return out
}

// simEnrich pose sur chaque match de l'historique les faits de la variante.
func simEnrich(matches []historyRow, rows map[string][]narrative.ObjectiveTeamRow, xuid string) map[string]narrative.ObjectiveTeamShare {
	shares := map[string]narrative.ObjectiveTeamShare{}
	for i := range matches {
		share, ok := narrative.ComputeObjectiveTeamShare(rows[matches[i].MatchID], xuid)
		if !ok {
			continue
		}
		shares[matches[i].MatchID] = share
		matches[i].ObjectiveBlend = &objectiveBlendFacts{Family: share.Family, Index: share.Index, Share: share.Share}
	}
	return shares
}

func f1(v float64) string { return fmt.Sprintf("%.1f", v) }

func TestResearchPerformanceObjectiveSimulation(t *testing.T) {
	root := os.Getenv("LEVELUP_PERFSIM_ROOT")
	outPath := os.Getenv("LEVELUP_PERFSIM_OUT")
	if root == "" || outPath == "" {
		t.Skip("LEVELUP_PERFSIM_ROOT et LEVELUP_PERFSIM_OUT requis")
	}
	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	shared := openRO(t, filepath.Join(root, "warehouse", "shared_matches_v2.duckdb"))
	metaPath := filepath.Join(root, "warehouse", "metadata.duckdb")

	entries, err := os.ReadDir(filepath.Join(root, "players"))
	if err != nil {
		t.Fatalf("players : %v", err)
	}
	var players []string
	for _, e := range entries {
		if e.IsDir() {
			players = append(players, e.Name())
		}
	}
	sort.Strings(players)
	xuids := simXUIDs(t, shared, players)
	info := simMatchInfo(shared)

	var out strings.Builder
	out.WriteString(simHeader + "\n")
	for _, gt := range players {
		xuid := xuids[gt]
		if xuid == "" {
			t.Logf("%s : xuid introuvable, joueur ignoré", gt)
			continue
		}
		player := openRO(t, filepath.Join(root, "players", gt, "stats.duckdb"))
		n := simPlayer(t, ctx, &out, gt, xuid, player, shared, metaPath, info)
		t.Logf("%s : %d matchs à objectif simulés", gt, n)
	}
	if err := os.WriteFile(outPath, []byte(out.String()), 0o644); err != nil {
		t.Fatalf("écriture %s : %v", outPath, err)
	}
}

// simPlayer rejoue la boucle de batchComputePerformanceScores pour un joueur et écrit les
// lignes des matchs des chaînes à objectif. Rend leur nombre.
func simPlayer(t *testing.T, ctx context.Context, out *strings.Builder, gt, xuid string,
	player, shared *sql.DB, metaPath string, info map[string][3]string,
) int {
	all, err := loadHistoryForPerf(ctx, shared, xuid)
	if err != nil {
		t.Fatalf("%s : historique : %v", gt, err)
	}
	if excluded, err := loadExcludedMatchIDs(ctx, player); err == nil && len(excluded) > 0 {
		kept := all[:0]
		for _, m := range all {
			if !excluded[m.MatchID] {
				kept = append(kept, m)
			}
		}
		all = kept
	}
	applyPerfEnrichments(ctx, player, xuid, all, loadMedalExploitMap(ctx, metaPath, shared, xuid))
	rows, err := duckdbpkg.LoadObjectiveTeamRowsForPlayer(ctx, shared, xuid)
	if err != nil {
		t.Fatalf("%s : lignes d'objectif : %v", gt, err)
	}
	shares := simEnrich(all, rows, xuid)
	stored := simStoredNotes(player)

	count := 0
	chainHistory := map[string][]historyRow{}
	for i := range all {
		m := all[i]
		history := chainHistory[m.Chain]
		chainHistory[m.Chain] = append(history, m)
		if len(history) < skill.MinMatchesPerChainForRelative {
			continue
		}
		window := history[max(0, len(history)-perfWindowSize):]
		weights := skill.WeightsForChain(m.Chain)
		old := computeRelativePerformanceScore(&m, window, weights)
		if old == nil {
			continue
		}
		if _, isObj := profileWeight(weights, MetricKeyObjectiveParticipation); !isObj {
			continue
		}
		count++
		simWriteRow(out, gt, &m, window, weights, *old, stored, shares[m.MatchID], info[m.MatchID], xuid, rows[m.MatchID])
	}
	return count
}

func simWriteRow(out *strings.Builder, gt string, m *historyRow, window []historyRow, weights map[string]float64,
	old float64, stored map[string]float64, share narrative.ObjectiveTeamShare, inf [3]string, xuid string, mrows []narrative.ObjectiveTeamRow,
) {
	team := ""
	for i := range mrows {
		if mrows[i].XUID == xuid {
			team = fmt.Sprint(mrows[i].TeamID)
		}
	}
	nw := make([]string, len(simAlphaMaxes))
	var mid objectiveBlendResult
	nref := 0
	for i, am := range simAlphaMaxes {
		p := defaultObjectiveBlendParams()
		p.AlphaMax = am
		res, ok := computeObjectiveBlend(m, window, weights, p)
		if !ok {
			nw[i] = ""
			continue
		}
		nw[i] = f1(res.Score)
		if am == 0.50 {
			mid = res
		}
	}
	if m.ObjectiveBlend != nil {
		series, _, ok := objectiveReferenceSeries(m.ObjectiveBlend, window, MinMatchesForRelative)
		if ok {
			nref = len(series)
		}
	}
	scope, st := "none", ""
	if mid.HasObjective {
		scope = "chain"
		if mid.FamilyScope {
			scope = "family"
		}
	}
	if v, ok := stored[m.MatchID]; ok {
		st = f1(v)
	}
	fam := string(share.Family)
	fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%.0f\t%.0f\t%.0f\t%.2f\t%s\t%.4f\t%.2f\t%.2f\t%.4f\t%s\t%.2f\t%s\t%s\t%s\t%s\t%s\t%.4f\t%.1f\t%d\n",
		gt, m.MatchID, inf[0], inf[1], inf[2], m.Chain, fam, m.Kills, m.Deaths, m.Assists, m.TimePlayedSeconds/60,
		team, share.Share, share.Points, share.TeamPoints, share.Index, scope, mid.Combat, f1(old), st, nw[0], nw[1], nw[2],
		mid.Alpha, mid.Percentile, nref)
}
