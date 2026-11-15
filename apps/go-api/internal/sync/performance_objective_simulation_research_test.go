//go:build research && cgo

package sync

// performance_objective_simulation_research_test.go — SIMULATION de la note de performance à
// objectif (V7, branchée dans computeChainPerformanceScore) sur une COPIE À FROID des bases.
// Instrument de recherche, lecture seule : aucune base n'est écrite (les notes ne sont calculées
// qu'en mémoire).
//
// Il rejoue la boucle de batchComputePerformanceScores (mêmes chargements, mêmes fenêtres de 50
// matchs par chaîne, même seuil de 10 matchs) et, pour chaque match d'une chaîne à objectif,
// compare la note STOCKÉE de la copie (« Actuelle » : celle de l'ancienne formule, calculée avant
// la montée de révision) à la note que le code de production donne maintenant (V7). Le rapport
// (tableaux Markdown) est écrit par simWriteReport
// (performance_objective_simulation_report_research_test.go).
//
// Lancement (depuis apps/go-api) :
//
//	LEVELUP_PERFSIM_ROOT=<copie>/halo_infinite LEVELUP_PERFSIM_OUT=<dossier>/rapport.md \
//	  LEVELUP_PERFSIM_PROFILES=<db_profiles.json> \
//	  go test -tags=research ./internal/sync/ -run TestResearchPerformanceObjectiveSimulation -v -timeout 40m
//
// <copie>/halo_infinite contient `warehouse/` et `players/<gamertag>/stats.duckdb`.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/ctxkeys"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/skill"
)

// simOut : une note.
type simOut struct{ Score float64 }

// simRecord : un match à objectif d'un joueur suivi : note stockée (Old) et note de V7.
type simRecord struct {
	Player, MatchID, Start, Map, Pair, Chain string
	Kills, Deaths, Assists, KDA              float64
	Team                                     int
	Old                                      simOut
	Share                                    narrative.ObjectiveTeamShare
	By                                       map[string]simOut
}

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

// simMatchInfo : date, carte, mode des matchs, pour la lecture humaine du rapport.
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

	env := &simEnv{ctx: ctx, shared: shared, metaPath: metaPath, info: simMatchInfo(shared), xuids: xuids,
		objRows: map[string]map[string][]narrative.ObjectiveTeamRow{}}
	var records []simRecord
	for _, gt := range players {
		xuid := xuids[gt]
		if xuid == "" {
			continue
		}
		player := openRO(t, filepath.Join(root, "players", gt, "stats.duckdb"))
		recs := simPlayer(t, env, gt, xuid, player)
		t.Logf("%s : %d matchs à objectif simulés", gt, len(recs))
		records = append(records, recs...)
	}
	if err := os.WriteFile(outPath, []byte(simWriteReport(t, env, records)), 0o644); err != nil {
		t.Fatalf("écriture %s : %v", outPath, err)
	}
}

// simEnv : ce que la simulation partage entre joueurs.
type simEnv struct {
	ctx      context.Context
	shared   *sql.DB
	metaPath string
	info     map[string][3]string
	xuids    map[string]string
	// objRows : par joueur suivi, les lignes d'objectif des deux camps de ses matchs.
	objRows map[string]map[string][]narrative.ObjectiveTeamRow
}

// simPlayer rejoue la boucle de notes d'un joueur et rend ses matchs à objectif notés des deux façons.
func simPlayer(t *testing.T, env *simEnv, gt, xuid string, player *sql.DB) []simRecord {
	t.Helper()
	all, err := loadHistoryForPerf(env.ctx, env.shared, xuid)
	if err != nil {
		t.Fatalf("%s : historique : %v", gt, err)
	}
	if excluded, err := loadExcludedMatchIDs(env.ctx, player); err == nil && len(excluded) > 0 {
		kept := all[:0]
		for _, m := range all {
			if !excluded[m.MatchID] {
				kept = append(kept, m)
			}
		}
		all = kept
	}
	if err := applyPerfEnrichments(env.ctx, env.shared, xuid, all, loadMedalExploitMap(env.ctx, env.metaPath, env.shared, xuid)); err != nil {
		t.Fatalf("%s : %v", gt, err)
	}
	rows, err := duckdbpkg.LoadObjectiveTeamRowsForPlayer(env.ctx, env.shared, xuid)
	if err != nil {
		t.Fatalf("%s : lignes d'objectif : %v", gt, err)
	}
	env.objRows[gt] = rows
	stored := simStoredNotes(player)

	var recs []simRecord
	chainHistory := map[string][]historyRow{}
	for i := range all {
		m := all[i]
		history := chainHistory[m.Chain]
		chainHistory[m.Chain] = append(history, m)
		if len(history) < skill.MinMatchesPerChainForRelative || !skill.IsObjectiveChain(m.Chain) {
			continue
		}
		old, hasOld := stored[m.MatchID]
		v7 := computeChainPerformanceScore(m.Chain, &m, history[max(0, len(history)-perfWindowSize):])
		if v7 == nil || !hasOld {
			continue
		}
		inf := env.info[m.MatchID]
		rec := simRecord{Player: gt, MatchID: m.MatchID, Start: inf[0], Map: inf[1], Pair: inf[2], Chain: m.Chain,
			Kills: m.Kills, Deaths: m.Deaths, Assists: m.Assists, KDA: m.KDA, Team: narrative.NoObjectiveTeam,
			Old: simOut{old}, By: map[string]simOut{"V7": {*v7}}}
		rec.Share, _ = narrative.ComputeObjectiveTeamShare(rows[m.MatchID], xuid)
		for _, r := range rows[m.MatchID] {
			if r.XUID == xuid {
				rec.Team = r.TeamID
			}
		}
		recs = append(recs, rec)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Start < recs[j].Start })
	return recs
}
