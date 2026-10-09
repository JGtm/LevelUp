//go:build psarepro

// psa_index_repro_msr_fixture_test.go — fixture de la mesure D-4
// (psa_index_repro_msr_planprobe_test.go, voir ce fichier pour la methode) : DDL,
// remplissage realiste de match_skill_rank et formes de lecture mesurees.
package migration

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

// msrProbeDDL : instantane de la table et de la vue _latest de sync/schema.go.
const msrProbeDDL = `
CREATE SEQUENCE IF NOT EXISTS msr_seq START 1;
CREATE TABLE IF NOT EXISTS match_skill_rank (
    id                BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
    match_id          VARCHAR NOT NULL,
    rating_type       VARCHAR NOT NULL,
    rating_value      FLOAT,
    rating_deviation  FLOAT,
    tier              VARCHAR,
    tier_fr           VARCHAR,
    sub_tier          SMALLINT DEFAULT 0,
    tier_label        VARCHAR,
    rating_delta      FLOAT,
    playlist_group    VARCHAR,
    expected_win_prob FLOAT,
    start_time        TIMESTAMP,
    written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE OR REPLACE VIEW match_skill_rank_latest AS
    SELECT * FROM match_skill_rank
    QUALIFY ROW_NUMBER() OVER (
        PARTITION BY match_id
        ORDER BY
            CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
            start_time DESC NULLS LAST,
            written_at DESC,
            id DESC
    ) = 1;`

// msrProbeIndexes : les trois index tels que les posaient les autorites avant B3.
var msrProbeIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
	`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
	`CREATE INDEX IF NOT EXISTS idx_msr_playlist    ON match_skill_rank(playlist_group)`,
}

// Chaines LUSR de Halo Infinite (skillchain.Chains) et leur poids (sur 20).
var msrProbeChains = []struct {
	name   string
	weight int
}{{"arena_slayer", 9}, {"arena_objectif", 5}, {"btb", 4}, {"chaos", 2}}

type msrProbeData struct {
	matchIDs []string // ordre chronologique
	starts   []time.Time
}

func msrProbeChainOf(i int) string {
	k := (i * 7) % 20
	for _, c := range msrProbeChains {
		if k < c.weight {
			return c.name
		}
		k -= c.weight
	}
	return msrProbeChains[0].name
}

// msrProbeFill ecrit exactement msrProbeRows lignes, par lots de 25 matchs (un
// cycle de sync) en transaction, CHECKPOINT toutes les 2 000 lignes environ.
// Match i : classe (i%4==0) → 1 CSR, + 1 version reecrite si i%8==0 ; social → LUSR +
// LUSR_V2 (chaine ponderee, expected_win_prob sur LUSR), + 1 version LUSR reecrite
// si i%3!=0 (backfill, rebuild de chaine).
func msrProbeFill(t *testing.T, db *sql.DB) msrProbeData {
	t.Helper()
	ctx := context.Background()
	base := time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC)
	const ins = `INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, rating_deviation, tier, tier_fr, sub_tier,
		 tier_label, rating_delta, playlist_group, expected_win_prob, start_time, written_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	var d msrProbeData
	rows, sinceCkpt := 0, 0
	var tx *sql.Tx
	put := func(mid, rt, grp string, v float64, ewp any, st time.Time, wr time.Time) {
		if rows >= msrProbeRows {
			return
		}
		if _, err := tx.ExecContext(ctx, ins, mid, rt, v, 80.0, "Gold", "Or", 3, "Gold 3",
			1.5, grp, ewp, st, wr); err != nil {
			_ = tx.Rollback()
			t.Fatalf("insert: %v", err)
		}
		rows++
		sinceCkpt++
	}
	for i := 0; rows < msrProbeRows; i++ {
		if i%25 == 0 {
			if tx != nil {
				if err := tx.Commit(); err != nil {
					t.Fatalf("commit: %v", err)
				}
				if sinceCkpt >= 2000 {
					reproExec(t, db, `CHECKPOINT`)
					sinceCkpt = 0
				}
			}
			var err error
			if tx, err = db.BeginTx(ctx, nil); err != nil {
				t.Fatalf("begin: %v", err)
			}
		}
		mid := reproMatchID(i)
		st := base.Add(time.Duration(i) * 5 * time.Hour)
		wr := st.Add(20 * time.Minute)
		d.matchIDs = append(d.matchIDs, mid)
		d.starts = append(d.starts, st)
		v := 1200 + float64((i*37)%900)
		if i%4 == 0 {
			put(mid, "CSR", "ranked_arena", v, nil, st, wr)
			if i%8 == 0 {
				put(mid, "CSR", "ranked_arena", v, nil, st, wr.Add(30*24*time.Hour))
			}
			continue
		}
		grp := msrProbeChainOf(i)
		if i%230 == 1 {
			grp = "h5_arena" // chaine etrangere rare (incident du 2026-09-13)
		}
		put(mid, "LUSR", grp, v, 0.4+float64(i%20)/100, st, wr)
		put(mid, "LUSR_V2", grp, v+3, nil, st, wr)
		if i%3 != 0 {
			put(mid, "LUSR", grp, v+1, 0.45, st, wr.Add(60*24*time.Hour))
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit final: %v", err)
	}
	reproExec(t, db, `CHECKPOINT`)
	return d
}

type msrProbeForm struct {
	id, label, sql string
	args           []any
	core           bool // dans les sept formes du critere D-4
}

func msrProbeIn(ids []string) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","), args
}

// msrProbeForms : les sept formes de lecture BRUTE (plan §5 B3, pieces), puis les
// formes supplementaires. Parametres choisis pour la pire selectivite realiste.
func msrProbeForms(d msrProbeData) []msrProbeForm {
	n := len(d.matchIDs)
	cit, citArgs := msrProbeIn(d.matchIDs[n-200:])
	sq, sqArgs := msrProbeIn(d.matchIDs[n-1000:])
	cur, curStart := d.matchIDs[n-1], d.starts[n-1]
	return []msrProbeForm{
		{"F1", "Q24LUSRHistory (platform/duckdb/queries_career_encounters.go)", `
SELECT msr.match_id, msr.rating_value, msr.rating_deviation, msr.playlist_group
FROM match_skill_rank msr ORDER BY msr.match_id ASC`, nil, true},
		{"F2", "Q26gPlaylistPhaseAMSRTpl, IN(200) (queries_home_citations.go)", `
WITH exploitable AS (
	SELECT match_id, rating_type, rating_value,
		NULLIF(TRIM(tier), '') AS tier, NULLIF(TRIM(tier_fr), '') AS tier_fr,
		COALESCE(sub_tier, 0) AS sub_tier, NULLIF(TRIM(tier_label), '') AS tier_label,
		ROW_NUMBER() OVER (PARTITION BY match_id ORDER BY CASE UPPER(COALESCE(rating_type, ''))
			WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 WHEN 'LUSR_V2' THEN 2 ELSE 3
		END, written_at DESC, id DESC) AS rn
	FROM match_skill_rank
	WHERE match_id IN (` + cit + `) AND rating_value IS NOT NULL
	  AND NOT (UPPER(COALESCE(rating_type, '')) = 'CSR' AND rating_value = 0))
SELECT match_id, rating_value, tier, tier_fr, sub_tier, tier_label FROM exploitable WHERE rn = 1`,
			citArgs, true},
		{"F3", "QSquadExpectedWinProbTpl, IN(1000) (queries_squad.go)", `
SELECT match_id, MAX(expected_win_prob) AS expected_win_prob FROM match_skill_rank
WHERE match_id IN (` + sq + `) AND expected_win_prob IS NOT NULL GROUP BY match_id`, sqArgs, true},
		{"F4", "loadExistingCSRMatchIDs (sync/csr_writes.go)",
			`SELECT DISTINCT match_id FROM match_skill_rank WHERE rating_type = 'CSR'`, nil, true},
		{"F5a", "LoadExistingRatingIDs('LUSR') (sync/skill/skill_rating_loaders.go)",
			`SELECT match_id FROM match_skill_rank WHERE rating_type = ?`, []any{"LUSR"}, true},
		{"F5b", "loadExistingLUSRStates (sync/skill/skill_rating_loaders.go)", `
SELECT msr.playlist_group, msr.rating_value, msr.rating_deviation FROM match_skill_rank msr
JOIN (SELECT playlist_group, MAX(start_time) AS max_st FROM match_skill_rank
      WHERE rating_type = 'LUSR' GROUP BY playlist_group) last
  ON msr.playlist_group = last.playlist_group AND msr.start_time = last.max_st
WHERE msr.rating_type = 'LUSR'`, nil, true},
		{"F6a", "loadPreviousLUSRRating('arena_slayer') (sync/skill/skill_v2_canonical.go)", `
SELECT rating_value FROM match_skill_rank
WHERE rating_type = 'LUSR' AND playlist_group = ? AND rating_value IS NOT NULL
  AND match_id != ? AND (start_time IS NULL OR start_time < ?)
ORDER BY start_time DESC NULLS LAST, written_at DESC, id DESC LIMIT 1`,
			[]any{"arena_slayer", cur, curStart}, true},
		{"F6b", "loadPreviousDisplayedOrdinal('chaos') (sync/skill/skill_v2_canonical.go)", `
SELECT tier, sub_tier FROM match_skill_rank
WHERE rating_type = 'LUSR' AND playlist_group = ? AND tier IS NOT NULL
  AND match_id != ? AND (start_time IS NULL OR start_time < ?)
ORDER BY start_time DESC NULLS LAST, written_at DESC, id DESC LIMIT 1`,
			[]any{"chaos", cur, curStart}, true},
		{"F7", "RunDualRowSentinel (sync/skill/skill_v2_metrics.go)", `
WITH per_match AS (SELECT match_id, BOOL_OR(rating_type = 'LUSR') AS has_lusr,
  BOOL_OR(rating_type = 'LUSR_V2') AS has_lusr_v2 FROM match_skill_rank
  WHERE rating_type IN ('LUSR', 'LUSR_V2') GROUP BY match_id)
SELECT match_id, has_lusr, has_lusr_v2 FROM per_match`, nil, true},
		{"S1", "invariants I8 (sync/invariants/invariants.go)",
			`SELECT DISTINCT match_id FROM match_skill_rank`, nil, false},
		{"S2", "invariants I9 lusr_v2_orphan (invariants.go)", `
SELECT match_id FROM match_skill_rank GROUP BY match_id
HAVING SUM(CASE WHEN rating_type = 'LUSR_V2' THEN 1 ELSE 0 END) > 0
   AND SUM(CASE WHEN rating_type = 'LUSR' THEN 1 ELSE 0 END) = 0`, nil, false},
		{"S3", "lusr_chain_foreign_title (invariants_lusr_chain.go)", `
SELECT playlist_group, COUNT(*) AS n FROM match_skill_rank
WHERE rating_type IN (?, ?) AND playlist_group IS NOT NULL AND playlist_group <> ''
  AND playlist_group NOT IN (?, ?, ?, ?)
GROUP BY playlist_group ORDER BY n DESC, playlist_group ASC`,
			[]any{"LUSR", "LUSR_V2", "arena_slayer", "arena_objectif", "btb", "chaos"}, false},
		{"S4", "vue match_skill_rank_latest, IN(200)",
			`SELECT match_id, rating_value FROM match_skill_rank_latest WHERE match_id IN (` + cit + `)`,
			citArgs, false},
		// Temoins de CALIBRAGE du detecteur. C0 : lookup par PRIMARY KEY (ART present
		// dans les deux bases) → Index Scan attendu des deux cotes, sinon le detecteur
		// ne voit rien. C1/C2 : la forme exacte de l'incident du 2026-09-13
		// (cmd/purge_foreign_lusr_chain, COUNT d'une chaine etrangere peu peuplee).
		{"C0", "temoin : PK id = 5",
			`SELECT * FROM match_skill_rank WHERE id = 5`, nil, false},
		{"C1", "temoin : COUNT(*) WHERE playlist_group = litteral 'h5_arena'",
			`SELECT COUNT(*) FROM match_skill_rank WHERE playlist_group = 'h5_arena'`, nil, false},
		{"C2", "temoin : COUNT(*) WHERE playlist_group = ? ('h5_arena')",
			`SELECT COUNT(*) FROM match_skill_rank WHERE playlist_group = ?`, []any{"h5_arena"}, false},
	}
}
