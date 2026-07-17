//go:build cgo

package prestigetuning

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTelemetryDB crée les tables minimales (challenge + prestige_telemetry) et
// insère un jeu de fixtures reproduisant la structure réelle : les défis créés/
// complétés ont une ligne challenge (jointure), les rejets n'en ont pas.
func setupTelemetryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE challenge (
			id VARCHAR PRIMARY KEY, metric VARCHAR, window_type VARCHAR, window_value VARCHAR, source VARCHAR
		);
		CREATE TABLE prestige_telemetry (
			id VARCHAR PRIMARY KEY, challenge_id VARCHAR, event_type VARCHAR, source VARCHAR
		);`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	// 3 défis coach accuracy/last_n_matches:10 : 3 créés, 1 complété.
	challenges := []struct{ id, metric, wt, wv, src string }{
		{"c1", "accuracy", "last_n_matches", "10", "coach"},
		{"c2", "accuracy", "last_n_matches", "10", "coach"},
		{"c3", "accuracy", "last_n_matches", "10", "coach"},
	}
	for _, c := range challenges {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO challenge VALUES (?,?,?,?,?)`, c.id, c.metric, c.wt, c.wv, c.src); err != nil {
			t.Fatalf("insert challenge: %v", err)
		}
	}
	events := []struct{ id, cid, et, src string }{
		{"t1", "c1", "created", "coach"},
		{"t2", "c2", "created", "coach"},
		{"t3", "c3", "created", "coach"},
		{"t4", "c1", "completed", "coach"},
		{"t5", "c2", "abandoned", "coach"},
		// rejet SANS ligne challenge (auto-reject too_easy non persisté).
		{"t6", "cX", "rejected:too_easy", "coach"},
		{"t7", "cY", "rejected:too_easy", "coach"},
	}
	for _, e := range events {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO prestige_telemetry VALUES (?,?,?,?)`, e.id, e.cid, e.et, e.src); err != nil {
			t.Fatalf("insert telemetry: %v", err)
		}
	}
	return db
}

func TestCollectFromDB_JoinAndAcceptance(t *testing.T) {
	db := setupTelemetryDB(t)
	ctx := context.Background()

	counts, accept, err := CollectFromDB(ctx, db)
	if err != nil {
		t.Fatalf("CollectFromDB: %v", err)
	}

	// Jointure : 1 ligne (coach, accuracy, last_n_matches, 10) → 3 created, 1 completed,
	// 1 abandoned. Les 2 rejets n'ont PAS de challenge → exclus de la jointure.
	if len(counts) != 1 {
		t.Fatalf("counts len = %d, want 1", len(counts))
	}
	c := counts[0]
	if c.Created != 3 || c.Completed != 1 || c.Abandoned != 1 {
		t.Errorf("counts = created %d/completed %d/abandoned %d, want 3/1/1", c.Created, c.Completed, c.Abandoned)
	}
	if c.Metric != "accuracy" || c.WindowSpec() != "last_n_matches:10" {
		t.Errorf("metric/window = %s / %s", c.Metric, c.WindowSpec())
	}

	// Acceptance (sans jointure) : coach = 3 created, 2 rejected → 3/5 = 0.6.
	if len(accept) != 1 {
		t.Fatalf("accept len = %d, want 1", len(accept))
	}
	a := accept[0]
	if a.Created != 3 || a.Rejected != 2 {
		t.Errorf("acceptance = %d/%d, want 3/2", a.Created, a.Rejected)
	}
	if a.AcceptanceRate < 0.59 || a.AcceptanceRate > 0.61 {
		t.Errorf("acceptance rate = %.3f, want ~0.60", a.AcceptanceRate)
	}
}

// DB legacy sans colonne source : les événements sont agrégés sous "unknown"
// (le joueur n'est pas perdu), pas d'erreur.
func TestCollectFromDB_LegacyNoSourceColumn(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE challenge (id VARCHAR PRIMARY KEY, metric VARCHAR, window_type VARCHAR, window_value VARCHAR);
		CREATE TABLE prestige_telemetry (id VARCHAR PRIMARY KEY, challenge_id VARCHAR, event_type VARCHAR);
		INSERT INTO challenge VALUES ('c1','kda','session',NULL);
		INSERT INTO prestige_telemetry VALUES ('t1','c1','created'),('t2','c1','completed');`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	counts, accept, err := CollectFromDB(ctx, db)
	if err != nil {
		t.Fatalf("CollectFromDB legacy: %v", err)
	}
	if len(counts) != 1 || counts[0].Source != "unknown" {
		t.Fatalf("counts = %+v, want 1 ligne source=unknown", counts)
	}
	if counts[0].WindowSpec() != "session" {
		t.Errorf("window = %q, want session", counts[0].WindowSpec())
	}
	if len(accept) != 1 || accept[0].Source != "unknown" || accept[0].Created != 1 {
		t.Errorf("accept = %+v, want unknown/created 1", accept)
	}
}

// DB sans table prestige_telemetry : résultat vide, aucune erreur (best-effort).
func TestCollectFromDB_NoTelemetryTable(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	counts, accept, err := CollectFromDB(context.Background(), db)
	if err != nil {
		t.Fatalf("want nil error on legacy DB, got %v", err)
	}
	if len(counts) != 0 || len(accept) != 0 {
		t.Errorf("want empty result, got counts=%d accept=%d", len(counts), len(accept))
	}
}

// bout-en-bout : collecte → analyse produit une recommandation d'ajustement
// (complétion 33% < 30%? non — ajustons le seuil pour déclencher).
func TestCollectAndAnalyze_EndToEnd(t *testing.T) {
	db := setupTelemetryDB(t)
	ctx := context.Background()
	counts, accept, err := CollectFromDB(ctx, db)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	grammar := NewGrammarView(map[string][]string{"accuracy": {"last_n_matches:10"}})
	// Seuil échantillon abaissé à 3 pour statuer sur ce petit fixture ; complétion
	// 1/3 = 33% < 40% → recommandation.
	thr := Thresholds{MinCompletionRate: 0.40, MinSample: 3, Source: "coach"}
	rep := Analyze(counts, accept, grammar, thr, fixedNow)

	m := findMetric(t, rep.Metrics, "accuracy")
	if m.Status != StatusRecommendAdjust {
		t.Fatalf("status = %q, want recommend_adjust", m.Status)
	}
	if rep.SourceAcceptance[0].Source != "coach" {
		t.Errorf("source acceptance manquante")
	}
}
