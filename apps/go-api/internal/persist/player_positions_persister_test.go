//go:build integration

// Package persist — player_positions_persister_test.go : ce que [PlayerPositionsPersister]
// ECRIT, et surtout QU IL N EFFACE RIEN.
//
// La table remplace un DELETE-then-INSERT par match (decision utilisateur 1, plan v2). Le test
// central est donc celui de la REPRISE : une seconde projection doit superseder la premiere PAR
// LA VUE, en la laissant intacte EN BASE. Le schema est celui des migrations REELLES (RunForDB
// sur TargetShared), pas un DDL recopie — meme doctrine que kill_position_persister_test.go.

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openPlayerPositionsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

func positionsDeTest() []PlayerPositionRow {
	return []PlayerPositionRow{
		{TimeMS: 0, X: 1, Y: 2, Z: 3, Team: 0},
		{TimeMS: 20000, X: 4, Y: 5, Z: 6, Team: 1},
		{TimeMS: 40000, X: 7, Y: 8, Z: 9, Team: -1},
	}
}

// TestPlayerPositionsPersistPass_EcritEtRelitParLaVue — le chemin nominal.
func TestPlayerPositionsPersistPass_EcritEtRelitParLaVue(t *testing.T) {
	db := openPlayerPositionsTestDB(t)
	p := NewPlayerPositionsPersister(db)
	ctx := context.Background()

	if err := p.PersistPass(ctx, PlayerPositionsBatch{MatchID: "m-1", Rows: positionsDeTest()}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_player_positions_latest WHERE match_id = 'm-1'`).Scan(&n); err != nil {
		t.Fatalf("count vue: %v", err)
	}
	if n != 3 {
		t.Fatalf("%d ligne(s) servie(s) par la vue, attendu 3", n)
	}
	// TOUTES LES LIGNES D UNE PASSE PARTAGENT SON IDENTIFIANT ET SON HORODATAGE : c est ce qui
	// rend la vue capable de retenir une generation ENTIERE.
	var passes, horodatages int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT positions_pass), COUNT(DISTINCT written_at)
		FROM match_player_positions WHERE match_id = 'm-1'`).Scan(&passes, &horodatages); err != nil {
		t.Fatalf("distinct: %v", err)
	}
	if passes != 1 || horodatages != 1 {
		t.Errorf("passes distinctes = %d, horodatages distincts = %d — attendu 1 et 1 "+
			"(sans quoi la vue eclaterait une projection en autant de generations)", passes, horodatages)
	}
	// Le team -1 (« non attribuee ») survit tel quel : c est une valeur PLEINE.
	var team sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT team FROM match_player_positions_latest WHERE match_id = 'm-1' AND time_ms = 40000`).
		Scan(&team); err != nil {
		t.Fatalf("select team: %v", err)
	}
	if !team.Valid || team.Int64 != -1 {
		t.Errorf("team = %v, attendu -1", team)
	}
}

// TestPlayerPositionsPersistPass_RepriseSupersedeSansEffacer — LE TEST QUI REMPLACE LE
// DELETE-then-INSERT. Une seconde projection prend la main a la lecture, la premiere reste en
// base : zero DELETE, donc zero declencheur ART.
func TestPlayerPositionsPersistPass_RepriseSupersedeSansEffacer(t *testing.T) {
	db := openPlayerPositionsTestDB(t)
	p := NewPlayerPositionsPersister(db)
	ctx := context.Background()

	if err := p.PersistPass(ctx, PlayerPositionsBatch{MatchID: "m-1", Rows: positionsDeTest()}); err != nil {
		t.Fatalf("passe 1: %v", err)
	}
	// La seconde passe est PLUS COURTE : avec un DELETE+INSERT le compte tomberait a 1 ; avec
	// une vue par passe, il tombe a 1 A LA LECTURE et reste a 4 en base.
	if err := p.PersistPass(ctx, PlayerPositionsBatch{
		MatchID: "m-1",
		Rows:    []PlayerPositionRow{{TimeMS: 5000, X: 9, Y: 9, Z: 9, Team: 0}},
	}); err != nil {
		t.Fatalf("passe 2: %v", err)
	}

	var vue, brut int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_player_positions_latest WHERE match_id = 'm-1'`).Scan(&vue); err != nil {
		t.Fatalf("count vue: %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_player_positions WHERE match_id = 'm-1'`).Scan(&brut); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if vue != 1 {
		t.Errorf("vue = %d ligne(s), attendu 1 (la DERNIERE passe)", vue)
	}
	if brut != 4 {
		t.Errorf("table brute = %d ligne(s), attendu 4 — une reprise ne doit RIEN effacer", brut)
	}
	var t0 int
	if err := db.QueryRowContext(ctx,
		`SELECT time_ms FROM match_player_positions_latest WHERE match_id = 'm-1'`).Scan(&t0); err != nil {
		t.Fatalf("select time_ms: %v", err)
	}
	if t0 != 5000 {
		t.Errorf("time_ms servi = %d, attendu 5000 (la passe la plus recente)", t0)
	}
}

// TestPlayerPositionsPersistPass_Refus — les deux refus du contrat.
func TestPlayerPositionsPersistPass_Refus(t *testing.T) {
	db := openPlayerPositionsTestDB(t)
	p := NewPlayerPositionsPersister(db)
	ctx := context.Background()

	if err := p.PersistPass(ctx, PlayerPositionsBatch{Rows: positionsDeTest()}); err == nil {
		t.Error("un matchID vide doit etre REFUSE")
	}
	// UNE PASSE VIDE EST IGNOREE, PAS UNE ERREUR — mais elle n ecrit rien : ecrire zero ligne
	// serait indistinguable d un match sans positions decodables.
	if err := p.PersistPass(ctx, PlayerPositionsBatch{MatchID: "m-vide"}); err != nil {
		t.Errorf("une passe vide ne doit pas echouer, got %v", err)
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_player_positions WHERE match_id = 'm-vide'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("une passe vide a ecrit %d ligne(s), attendu 0", n)
	}
}
