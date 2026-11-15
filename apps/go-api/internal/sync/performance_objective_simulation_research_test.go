//go:build research && cgo

package sync

// performance_objective_simulation_research_test.go — SIMULATION de la note de performance à
// objectif (variante D-7 corrigée, performance_objective_blend.go) sur une COPIE À FROID des
// bases. Instrument de recherche, lecture seule : aucune base n'est écrite (les notes ne sont
// calculées qu'en mémoire), aucune révision de notation ne bouge.
//
// Il rejoue la boucle de batchComputePerformanceScores (mêmes chargements, mêmes fenêtres de 50
// matchs par chaîne, même seuil de 10 matchs) et, pour chaque match d'une chaîne à objectif,
// compare la note actuelle (recalculée par le code) aux variantes de simVariants, toutes
// SYMÉTRIQUES, sur la base retenue V2 ((1 - alpha) C + alpha O, indice brut, mi-rang, 229 points
// de vie) : V5 (V2 + le soutien, mesuré à part du temps de port), V6 (V5 + O_obj moyenne du rang
// de l'indice et du rang de la part). Le rapport (tableaux Markdown) est écrit par
// simWriteReport (performance_objective_simulation_report_research_test.go).
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

// simHP : points de vie effective du Rendement et de la Résistance, pour toutes les variantes
// (la base retenue : 229 au lieu de 225).
const simHP = 229

// simVariant : un réglage de la variante de note.
type simVariant struct {
	Name   string
	Params objectiveBlendParams
}

func simVariants() []simVariant {
	v2 := defaultObjectiveBlendParams()
	v5 := v2
	v5.UseSupport = true
	v6 := v5
	v6.ShareRank = true
	return []simVariant{{"V2", v2}, {"V5", v5}, {"V6", v6}}
}

// simOut : la note d'un match sous un réglage, avec son détail.
type simOut struct {
	Score, Combat, Alpha, Pct, Part float64
	Has, FamilyScope                bool
}

// simRecord : un match à objectif d'un joueur suivi, sous tous les réglages.
type simRecord struct {
	Player, MatchID, Start, Map, Pair, Chain string
	Kills, Deaths, Assists                   float64
	Team                                     int
	Stored                                   float64
	HasStored                                bool
	Old                                      simOut
	Share                                    narrative.ObjectiveTeamShare // poids de la table, sans relèvement
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

// simClone copie l'historique : chaque réglage pose ses propres faits d'objectif.
func simClone(all []historyRow) []historyRow {
	out := make([]historyRow, len(all))
	copy(out, all)
	return out
}

// simFacts pose sur chaque match les faits des variantes ; rend, par match, la part calculée.
func simFacts(matches []historyRow, rows map[string][]narrative.ObjectiveTeamRow, xuid string) map[string]narrative.ObjectiveTeamShare {
	shares := map[string]narrative.ObjectiveTeamShare{}
	for i := range matches {
		matches[i].ObjectiveBlend = nil
		share, ok := narrative.ComputeObjectiveTeamShare(rows[matches[i].MatchID], xuid)
		if !ok {
			continue
		}
		shares[matches[i].MatchID] = share
		matches[i].ObjectiveBlend = objectiveBlendFactsOf(share, matches[i].TimePlayedSeconds)
	}
	return shares
}

// simYield recalcule la Conversion offensive et la Résistance défensive avec hp points de vie
// effective (mêmes formules que skill.ComputeCombatYield, constante paramétrée).
func simYield(matches []historyRow, hp float64) {
	for i := range matches {
		m := &matches[i]
		m.OffensiveConversion, m.DefensiveResistance = 0, 0
		if m.DamageDealt > 0 {
			m.OffensiveConversion = hp * (m.Kills + m.Assists/3.0) / m.DamageDealt
		}
		if m.DamageTaken > 0 && m.Deaths > 0 {
			m.DefensiveResistance = m.DamageTaken / (hp * m.Deaths)
		}
	}
}

// simChain rejoue la boucle de chaînes du batch et rend, par match d'une chaîne à objectif, la
// sortie de `score`.
func simChain(all []historyRow, score func(m *historyRow, window []historyRow, weights map[string]float64) (simOut, bool)) map[string]simOut {
	out := map[string]simOut{}
	chainHistory := map[string][]historyRow{}
	for i := range all {
		m := all[i]
		history := chainHistory[m.Chain]
		chainHistory[m.Chain] = append(history, m)
		if len(history) < skill.MinMatchesPerChainForRelative {
			continue
		}
		weights := skill.WeightsForChain(m.Chain)
		if _, isObj := profileWeight(weights, MetricKeyObjectiveParticipation); !isObj {
			continue
		}
		window := history[max(0, len(history)-perfWindowSize):]
		if o, ok := score(&m, window, weights); ok {
			out[m.MatchID] = o
		}
	}
	return out
}

func simOld(m *historyRow, window []historyRow, weights map[string]float64) (simOut, bool) {
	s := computeRelativePerformanceScore(m, window, weights)
	if s == nil {
		return simOut{}, false
	}
	return simOut{Score: *s}, true
}

func simBlend(p objectiveBlendParams) func(*historyRow, []historyRow, map[string]float64) (simOut, bool) {
	return func(m *historyRow, window []historyRow, weights map[string]float64) (simOut, bool) {
		r, ok := computeObjectiveBlend(m, window, weights, p)
		if !ok {
			return simOut{}, false
		}
		return simOut{Score: r.Score, Combat: r.Combat, Alpha: r.Alpha, Pct: r.Percentile, Part: r.Part,
			Has: r.HasObjective, FamilyScope: r.FamilyScope}, true
	}
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
	info := simMatchInfo(shared)

	env := &simEnv{ctx: ctx, shared: shared, metaPath: metaPath, info: info, xuids: xuids,
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
	report := simWriteReport(t, env, records)
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
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

// simPlayer rejoue la boucle de notes d'un joueur sous tous les réglages.
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
	applyPerfEnrichments(env.ctx, player, xuid, all, loadMedalExploitMap(env.ctx, env.metaPath, env.shared, xuid))
	rows, err := duckdbpkg.LoadObjectiveTeamRowsForPlayer(env.ctx, env.shared, xuid)
	if err != nil {
		t.Fatalf("%s : lignes d'objectif : %v", gt, err)
	}
	env.objRows[gt] = rows

	old := simChain(all, simOld)

	variants := map[string]map[string]simOut{}
	rowsHP := simClone(all)
	simYield(rowsHP, simHP)
	baseShares := simFacts(rowsHP, rows, xuid)
	for _, v := range simVariants() {
		variants[v.Name] = simChain(rowsHP, simBlend(v.Params))
	}

	stored := simStoredNotes(player)
	var recs []simRecord
	for id, o := range old {
		inf := env.info[id]
		rec := simRecord{Player: gt, MatchID: id, Start: inf[0], Map: inf[1], Pair: inf[2], Old: o,
			Share: baseShares[id], By: map[string]simOut{}, Team: narrative.NoObjectiveTeam}
		for name, m := range variants {
			rec.By[name] = m[id]
		}
		if v, ok := stored[id]; ok {
			rec.Stored, rec.HasStored = v, true
		}
		for _, h := range all {
			if h.MatchID == id {
				rec.Chain, rec.Kills, rec.Deaths, rec.Assists = h.Chain, h.Kills, h.Deaths, h.Assists
				break
			}
		}
		for _, r := range rows[id] {
			if r.XUID == xuid {
				rec.Team = r.TeamID
			}
		}
		recs = append(recs, rec)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Start < recs[j].Start })
	return recs
}
