//go:build psarepro

// psa_index_repro_player_planprobe_test.go — MESURE VERSIONNEE du retrait des quatre index
// secondaires des bases joueur encore poses apres le retrait de ceux de match_skill_rank :
// idx_lch_component et idx_lch_match (lusr_component_history), idx_pme_match_lookup
// (player_match_enrichment), idx_pcs_lookup (player_csr_snapshots). Meme recette et meme
// critere que psa_index_repro_msr_planprobe_test.go (critere D-4 amende du plan backlog du
// 2026-09-26) : pour chaque forme de lecture, mediane SANS index <= mediane AVEC index +
// msrProbeMargin (temps client), plan lu par EXPLAIN ANALYZE (seul revelateur en DuckDB 1.5.5),
// resultats identiques avec et sans index.
//
// Difference avec la mesure MSR : la base est une COPIE d'une vraie base joueur, pas une
// fixture synthetique (lignes, etapes et versions reelles). Le chemin est donne par
// LEVELUP_C2_PLAYER_DB_COPY ; un chemin sous data/titles/ est refuse (la copie est copiee a
// nouveau deux fois dans un dossier temporaire, la source n'est jamais ouverte).
//
//	LEVELUP_C2_PLAYER_DB_COPY=<copie>.duckdb go test -tags=psarepro -count=1 ./internal/migration/ -run PlayerSecondaryIndex -v
//
// Les textes SQL sont des INSTANTANES des lecteurs de production (symbole cite pour chacun) :
// le paquet migration ne peut importer ni sync ni platform/duckdb (cycle).
package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// playerProbeIndexes : les quatre index tels que les posent les autorites (DDL relu).
var playerProbeIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name)`,
	`CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id)`,
	`CREATE INDEX IF NOT EXISTS idx_pme_match_lookup ON player_match_enrichment(match_id, written_at)`,
	`CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at)`,
}

var playerProbeIndexNames = []string{"idx_lch_component", "idx_lch_match", "idx_pme_match_lookup", "idx_pcs_lookup"}

type playerProbeData struct {
	pmeIDs, lchIDs []string // match_id distincts, du plus recent au plus ancien (id)
	component      string
	playlists      []string
	season         string
}

func playerProbeStrings(t *testing.T, db *sql.DB, q string) []string {
	t.Helper()
	r, err := db.Query(q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var s sql.NullString
		if err := r.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s.String)
	}
	return out
}

func playerProbeLoad(t *testing.T, db *sql.DB) playerProbeData {
	t.Helper()
	d := playerProbeData{
		pmeIDs: playerProbeStrings(t, db, `SELECT match_id FROM player_match_enrichment
			GROUP BY match_id ORDER BY MAX(id) DESC`),
		lchIDs: playerProbeStrings(t, db, `SELECT match_id FROM lusr_component_history
			GROUP BY match_id ORDER BY MAX(id) DESC`),
		playlists: playerProbeStrings(t, db, `SELECT DISTINCT playlist_id FROM player_csr_snapshots ORDER BY 1`),
	}
	if c := playerProbeStrings(t, db, `SELECT component_name FROM lusr_component_history
		GROUP BY 1 ORDER BY COUNT(*) DESC, 1 LIMIT 1`); len(c) == 1 {
		d.component = c[0]
	}
	if s := playerProbeStrings(t, db, `SELECT season_id FROM player_csr_snapshots
		GROUP BY 1 ORDER BY COUNT(*) DESC, 1 LIMIT 1`); len(s) == 1 {
		d.season = s[0]
	}
	if len(d.pmeIDs) < 1000 || len(d.playlists) == 0 {
		t.Fatalf("copie trop petite pour la mesure : %d matchs PME, %d playlists CSR",
			len(d.pmeIDs), len(d.playlists))
	}
	return d
}

// playerProbeForms : formes de lecture de production des trois tables, puis temoins.
func playerProbeForms(d playerProbeData) []msrProbeForm {
	pme200, pme200Args := msrProbeIn(d.pmeIDs[:200])
	pme1000, pme1000Args := msrProbeIn(d.pmeIDs[:1000])
	pl, plArgs := msrProbeIn(d.playlists)
	forms := []msrProbeForm{
		{"E1", "ancre d'idempotence du persisteur (persist/player_persister.go Persist)",
			`SELECT EXISTS(SELECT 1 FROM player_match_enrichment WHERE match_id = ? AND stage = 'live')`,
			[]any{d.pmeIDs[0]}, true},
		{"E2", "matchs deja engages (sync/engagement.go)",
			`SELECT match_id FROM player_match_enrichment WHERE stage = 'engagement'`, nil, true},
		{"E3", "vue _latest, match_id = ? (queries_match.go, engagement_score_repo.go)",
			`SELECT * FROM player_match_enrichment_latest WHERE match_id = ?`, []any{d.pmeIDs[0]}, true},
		{"E4", "vue _latest, IN(200) (queries_home_citations.go, patterns_repo.go)",
			`SELECT match_id, performance_score, session_label FROM player_match_enrichment_latest
			 WHERE match_id IN (` + pme200 + `)`, pme200Args, true},
		{"E5", "vue _latest, IN(1000) (squad_repo_mapstats.go)",
			`SELECT match_id, performance_score FROM player_match_enrichment_latest
			 WHERE match_id IN (` + pme1000 + `) AND performance_score IS NOT NULL`, pme1000Args, true},
		{"E6", "vue _latest, filtre hors cle (sync/aggregates.go)",
			`SELECT match_id, performance_score FROM player_match_enrichment_latest
			 WHERE performance_score IS NOT NULL`, nil, true},
	}
	// Une base sans historique de composantes LUSR (joueur sans chaine calculee) ne porte pas
	// les formes L ; au-dela de 200 matchs, la liste IN est bornee comme en production.
	if n := min(len(d.lchIDs), 200); n > 0 {
		lchIn, lchArgs := msrProbeIn(d.lchIDs[:n])
		forms = append(forms, msrProbeForm{"L1", "lusr_component_history_latest, composante + IN(200) (campaign_repo.go)",
			`SELECT match_id, value FROM lusr_component_history_latest
			 WHERE component_name = ? AND match_id IN (` + lchIn + `)`,
			append([]any{d.component}, lchArgs...), true},
			msrProbeForm{"L2", "lusr_component_history_latest, agregat par composante (progression/profile/queries.go)", `
			WITH ranked AS (
				SELECT component_name, value,
					ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at DESC) AS rk_desc,
					ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at ASC)  AS rk_asc,
					COUNT(*)    OVER (PARTITION BY component_name)                            AS n
				FROM lusr_component_history_latest)
			SELECT component_name, AVG(value), QUANTILE_CONT(value, 0.8),
				AVG(CASE WHEN n >= 20 AND rk_desc <= 10 THEN value END),
				AVG(CASE WHEN n >= 20 AND rk_asc  <= 10 THEN value END), MAX(n)
			FROM ranked GROUP BY component_name`, nil, true})
	}
	return append(forms, []msrProbeForm{
		{"S1", "player_csr_snapshots_latest, saison (queries_career_encounters.go Q24)",
			`SELECT playlist_id, current_value, alltime_value FROM player_csr_snapshots_latest
			 WHERE (? = '' OR season_id = ?) ORDER BY alltime_value DESC, current_value DESC`,
			[]any{d.season, d.season}, true},
		{"S2", "player_csr_snapshots_latest, playlist IN (queries_home_citations.go)", `
			WITH ranked AS (
				SELECT playlist_id, current_measurement_remaining,
					ROW_NUMBER() OVER (PARTITION BY playlist_id ORDER BY fetched_at DESC, season_id DESC) AS rn
				FROM player_csr_snapshots_latest WHERE playlist_id IN (` + pl + `))
			SELECT playlist_id, current_measurement_remaining FROM ranked WHERE rn = 1`, plArgs, true},
		{"S3", "player_csr_snapshots_latest, pic all-time (Q26csrAlltimePeak)",
			`SELECT alltime_value, alltime_tier, alltime_sub_tier FROM player_csr_snapshots_latest
			 WHERE (alltime_tier IS NOT NULL AND TRIM(alltime_tier) != '')
			    OR (alltime_value IS NOT NULL AND alltime_value > 0)`, nil, true},
		{"C0", "temoin : PK id = 5 (player_match_enrichment)",
			`SELECT * FROM player_match_enrichment WHERE id = 5`, nil, false},
	}...)
}

func playerProbeIndexCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name IN
		('idx_lch_component', 'idx_lch_match', 'idx_pme_match_lookup', 'idx_pcs_lookup')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestPlayerSecondaryIndexRemovalPlanProbe : mesure du lot C2 (plan des recommandations du
// 2026-10-09).
func TestPlayerSecondaryIndexRemovalPlanProbe(t *testing.T) {
	src := os.Getenv("LEVELUP_C2_PLAYER_DB_COPY")
	if src == "" {
		// Mesure sur donnees reelles uniquement : sans copie designee, rien a mesurer.
		t.Skip("LEVELUP_C2_PLAYER_DB_COPY non pose")
	}
	if strings.Contains(filepath.ToSlash(src), "/data/titles/") {
		t.Fatalf("refus : %s est sous data/titles/ — donner une COPIE", src)
	}
	dir := t.TempDir()
	withPath := filepath.Join(dir, "with_index.duckdb")
	withoutPath := filepath.Join(dir, "without_index.duckdb")
	msrProbeCopy(t, src, withPath)
	db := reproOpen(t, withPath)
	for _, s := range playerProbeIndexes {
		reproExec(t, db, s)
	}
	reproExec(t, db, `CHECKPOINT`)
	db.Close()
	msrProbeCopy(t, withPath, withoutPath)
	db = reproOpen(t, withoutPath)
	for _, n := range playerProbeIndexNames {
		reproExec(t, db, `DROP INDEX IF EXISTS `+n)
	}
	reproExec(t, db, `CHECKPOINT`)
	db.Close()

	with, without := reproOpen(t, withPath), reproOpen(t, withoutPath)
	defer with.Close()
	defer without.Close()
	if got := playerProbeIndexCount(t, with); got != 4 {
		t.Fatalf("base avec index : %d index, attendu 4", got)
	}
	if got := playerProbeIndexCount(t, without); got != 0 {
		t.Fatalf("base sans index : %d index, attendu 0", got)
	}
	data := playerProbeLoad(t, with)

	t.Logf("%-4s | %-10s %9s %9s %9s | %-10s %9s %9s %9s | %6s | forme", "id",
		"plan+idx", "med", "p90", "moteur", "plan-idx", "med", "p90", "moteur", "lignes")
	for _, f := range playerProbeForms(data) {
		a := msrProbeMeasure(t, with, f)
		b := msrProbeMeasure(t, without, f)
		t.Logf("%-4s | %-10s %9s %9s %9s | %-10s %9s %9s %9s | %6d | %s", f.id,
			a.plan, a.median.Round(time.Microsecond), a.p90.Round(time.Microsecond), a.engine.Round(time.Microsecond),
			b.plan, b.median.Round(time.Microsecond), b.p90.Round(time.Microsecond), b.engine.Round(time.Microsecond),
			b.n, f.label)
		// L2 departage des egalites de computed_at (une passe ecrit ses huit composantes au
		// meme instant) par ROW_NUMBER : son resultat varie d'une execution a l'autre sur la
		// MEME base. Une forme non deterministe n'est comparee que par son nombre de lignes.
		if _, again := msrProbeDigest(t, with, f); again != a.digest {
			t.Logf("[%s] resultat non deterministe sur la meme base : comparaison au nombre de lignes", f.id)
			if a.n != b.n {
				t.Errorf("[%s] %d lignes avec index, %d sans", f.id, a.n, b.n)
			}
		} else if a.n != b.n || a.digest != b.digest {
			t.Errorf("[%s] resultats differents avec (%d lignes) et sans index (%d lignes)", f.id, a.n, b.n)
		}
		if f.id == "C0" {
			if !strings.HasPrefix(a.plan, "IDX") || !strings.HasPrefix(b.plan, "IDX") {
				t.Fatalf("CALIBRAGE : lookup PK sans Index Scan (%s / %s) — le detecteur ne voit pas l'ART", a.plan, b.plan)
			}
		} else if !strings.HasPrefix(b.plan, "SEQ") {
			t.Errorf("[%s] plan sans index = %s, attendu sequentiel", f.id, b.plan)
		}
		if f.core && b.median > a.median+msrProbeMargin {
			t.Errorf("CRITERE NON TENU : [%s] %s — mediane sans index %s > avec index %s + %s",
				f.id, f.label, b.median, a.median, msrProbeMargin)
		}
	}
}
