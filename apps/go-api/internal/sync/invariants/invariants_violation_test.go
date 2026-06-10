//go:build integration

// Package invariants — tests POSITIFS : chaque check est exercé avec des
// données EN VIOLATION (le gate d'intégration ne vérifie que l'absence de
// violations sur données saines — un check à faux-négatif y resterait vert à
// vie). Schémas minimaux in-memory, indépendants du moteur de sync.
package invariants

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openMemDB(t *testing.T, ddl string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return db
}

const sharedDDL = `
CREATE TABLE match_registry (match_id VARCHAR, pair_name VARCHAR, mode_category VARCHAR);
CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR);
CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT);
CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR);
`

const playerDDL = `
CREATE TABLE player_match_enrichment (match_id VARCHAR, session_id VARCHAR, performance_score FLOAT, psa_checked_at TIMESTAMP);
CREATE TABLE match_skill_rank (match_id VARCHAR, rating_type VARCHAR);
CREATE TABLE match_citations (match_id VARCHAR, citation_name_norm VARCHAR);
CREATE TABLE personal_score_awards (match_id VARCHAR, xuid VARCHAR);
`

func violationKeys(rep Report) map[string]Violation {
	out := make(map[string]Violation, len(rep.Violations))
	for _, v := range rep.Violations {
		out[v.Key] = v
	}
	return out
}

// TestCheckPlayer_AllPlayerChecksFireOnViolations : un dataset construit pour
// violer CHAQUE invariant par-joueur — tous doivent remonter.
func TestCheckPlayer_AllPlayerChecksFireOnViolations(t *testing.T) {
	const xuid = "1111"
	shared := openMemDB(t, sharedDDL)
	player := openMemDB(t, playerDDL)

	// m1 : en shared pour le joueur, AUCUNE row enrichment → enrichment_missing.
	// m2 : enrichi mais session NULL + perf NULL + médailles sans citations +
	//      pas de PSA → session_missing, performance_score_missing,
	//      citations_missing, psa_missing. PvP sans skill rank → skill_rank_missing.
	// m3 : row LUSR_V2 sans row LUSR → lusr_v2_orphan.
	mustExec(t, shared, `INSERT INTO match_registry VALUES
		('m1','Slayer on Aquarius','arena_slayer'),
		('m2','CTF on Catalyst','arena_objectif'),
		('m3','Slayer on Live Fire','arena_slayer')`)
	mustExec(t, shared, `INSERT INTO match_participants VALUES ('m1',?),('m2',?),('m3',?)`, xuid, xuid, xuid)
	mustExec(t, shared, `INSERT INTO medals_earned VALUES ('m2',?,100)`, xuid)
	mustExec(t, player, `INSERT INTO player_match_enrichment (match_id) VALUES ('m2'),('m3')`)
	mustExec(t, player, `INSERT INTO match_skill_rank VALUES ('m3','LUSR_V2')`)

	rep, err := CheckPlayer(context.Background(), player, shared, xuid)
	if err != nil {
		t.Fatalf("CheckPlayer: %v", err)
	}
	got := violationKeys(rep)
	for _, want := range []string{
		"enrichment_missing", "lusr_v2_orphan", "session_missing",
		"performance_score_missing", "skill_rank_missing", "citations_missing", "psa_missing",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("invariant %q non déclenché alors que les données le violent (got: %v)", want, keysOf(got))
		}
	}
	if v := got["enrichment_missing"]; v.Severity != SeverityFail || v.Count != 1 {
		t.Errorf("enrichment_missing = %+v, want FAIL count=1", v)
	}
	if v := got["lusr_v2_orphan"]; v.Severity != SeverityFail {
		t.Errorf("lusr_v2_orphan severity = %s, want fail", v.Severity)
	}
}

// TestCheckPlayer_FirefightExcludedFromSkillRank : un match PvE sans skill
// rank ne compte PAS (tolérance firefight).
func TestCheckPlayer_FirefightExcludedFromSkillRank(t *testing.T) {
	const xuid = "1111"
	shared := openMemDB(t, sharedDDL)
	player := openMemDB(t, playerDDL)
	mustExec(t, shared, `INSERT INTO match_registry VALUES ('ff1','Firefight on Oasis','firefight')`)
	mustExec(t, shared, `INSERT INTO match_participants VALUES ('ff1',?)`, xuid)
	mustExec(t, player, `INSERT INTO player_match_enrichment (match_id, session_id, performance_score, psa_checked_at)
		VALUES ('ff1','s1',50.0, now())`)
	mustExec(t, player, `INSERT INTO personal_score_awards VALUES ('ff1',?)`, xuid)
	mustExec(t, player, `INSERT INTO match_citations VALUES ('ff1','x')`)

	rep, err := CheckPlayer(context.Background(), player, shared, xuid)
	if err != nil {
		t.Fatalf("CheckPlayer: %v", err)
	}
	if v, ok := violationKeys(rep)["skill_rank_missing"]; ok {
		t.Errorf("skill_rank_missing déclenché sur un match firefight (tolérance PvE cassée) : %+v", v)
	}
}

// TestCheckShared_AllSharedChecksFireOnViolations : dataset violant chaque
// invariant global — orphelins registry/participants/medals, pair UUID, alias.
func TestCheckShared_AllSharedChecksFireOnViolations(t *testing.T) {
	shared := openMemDB(t, sharedDDL)
	player := openMemDB(t, playerDDL) // pas d'ATTACH global → fallback shared_legacy pour l'alias

	mustExec(t, shared, `INSERT INTO match_registry VALUES
		('orphan-reg','Slayer on X','arena_slayer'),
		('uuid-pair','a3b2c1d0-1234-4abc-8def-0123456789ab','arena_slayer')`)
	mustExec(t, shared, `INSERT INTO match_participants VALUES
		('uuid-pair','2222'),
		('orphan-part','3333')`)
	mustExec(t, shared, `INSERT INTO medals_earned VALUES ('orphan-medal','2222',100)`)
	// Aucun alias inséré → 2222 et 3333 manquants (bots exclus par ailleurs).
	mustExec(t, shared, `INSERT INTO match_participants VALUES ('uuid-pair','bid(2.0)')`)

	rep, err := CheckShared(context.Background(), player, shared)
	if err != nil {
		t.Fatalf("CheckShared: %v", err)
	}
	got := violationKeys(rep)
	for _, want := range []string{
		"participants_without_registry", "registry_without_participants",
		"medals_without_registry", "pair_name_uuid", "xuid_alias_missing",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("invariant global %q non déclenché (got: %v)", want, keysOf(got))
		}
	}
	if v := got["xuid_alias_missing"]; v.Count != 2 {
		t.Errorf("xuid_alias_missing count = %d, want 2 (les bots bid( doivent être exclus)", v.Count)
	}
	if v := got["pair_name_uuid"]; v.Severity != SeverityWarn || v.Count != 1 {
		t.Errorf("pair_name_uuid = %+v, want WARN count=1", v)
	}
}

// TestCheckShared_CleanDataset : aucune violation sur données cohérentes.
func TestCheckShared_CleanDataset(t *testing.T) {
	shared := openMemDB(t, sharedDDL)
	player := openMemDB(t, playerDDL)
	mustExec(t, shared, `INSERT INTO match_registry VALUES ('m1','Slayer on Aquarius','arena_slayer')`)
	mustExec(t, shared, `INSERT INTO match_participants VALUES ('m1','2222')`)
	mustExec(t, shared, `INSERT INTO medals_earned VALUES ('m1','2222',100)`)
	mustExec(t, shared, `INSERT INTO xuid_aliases VALUES ('2222','PlayerTwo')`)

	rep, err := CheckShared(context.Background(), player, shared)
	if err != nil {
		t.Fatalf("CheckShared: %v", err)
	}
	if len(rep.Violations) != 0 {
		t.Errorf("dataset sain : %d violation(s) inattendue(s) : %v", len(rep.Violations), rep.Violations)
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func keysOf(m map[string]Violation) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
