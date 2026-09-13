package duckdb

// queries_career_lusr_latest_test.go — garde C.3 / C.3 bis du lot finitions LUSR
// (2026-09-13).
//
// Q8LUSRHistoryPlayer (graphe « Évolution LUSR / CSR » de la page Carrière) lisait la
// table BRUTE match_skill_rank. Sur une table append-only, le brut sert AUSSI les
// lignes des passes précédentes : une chaîne fausse écrite une fois y restait tracée
// À VIE, même après un replay correct — c'est ce qui maintenait la série « Arène »
// fantôme des 4 joueurs Infinite après la réparation d'août
// (.ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §6.2).
//
// DEUX contrats, indissociables — c'est tout l'objet de la vue par TYPE :
//  1. une ligne périmée d'un match rejoué n'est plus servie ;
//  2. un match CLASSÉ portant une ligne CSR ET une ligne LUSR les rend TOUTES LES DEUX.
//     Le graphe trace deux séries (LUSR pleine, CSR pointillée) et calcule ses deltas
//     par (rating_type, playlist_group) : servir match_skill_rank_latest, qui arbitre
//     CSR > LUSR par match_id, ferait disparaître le point LUSR de ces matchs.
//
// Fixture sur FICHIER (t.TempDir), pas :memory: — on exerce la vue et les index posés
// par les MIGRATIONS RÉELLES, jamais une DDL recopiée.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// newPlayerFixtureDB ouvre une player DB de test sur fichier, schéma par migrations.
func newPlayerFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open player fixture: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	return db
}

func TestQ8LUSRHistoryPlayer_ServesOnlyLatestRowPerMatch(t *testing.T) {
	db := newPlayerFixtureDB(t)
	ctx := context.Background()

	start := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	// Même match, même rating_type : la ligne PÉRIMÉE porte la chaîne étrangère
	// h5_arena (passe fautive du 2026-06-26), la ligne RÉPARÉE porte arena_slayer et
	// un written_at postérieur. Un replay append-only n'efface jamais la première.
	for _, r := range []struct {
		group     string
		value     float64
		writtenAt time.Time
	}{
		{"h5_arena", 1100, time.Date(2026, 6, 26, 11, 23, 0, 0, time.UTC)},
		{"arena_slayer", 1350, time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
			VALUES ('m_replayed', 'LUSR', ?, ?, ?, ?)`,
			r.value, r.group, start, r.writtenAt); err != nil {
			t.Fatalf("insert %s: %v", r.group, err)
		}
	}
	// Un second match, une seule ligne : il doit rester servi (pas d'effet de bord).
	if _, err := db.ExecContext(ctx, `INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES ('m_simple', 'LUSR', 1200, 'btb', ?, ?)`,
		start.Add(time.Hour), time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert m_simple: %v", err)
	}

	got := serveQ8(t, ctx, db)
	if len(got) != 2 {
		t.Fatalf("lignes servies = %d (%v), want 2", len(got), got)
	}
	if got[q8Key{"m_replayed", "LUSR"}] != "arena_slayer" {
		t.Errorf("m_replayed/LUSR → playlist_group = %q, want arena_slayer "+
			"(la ligne h5_arena du 2026-06-26 est périmée et ne doit plus être servie)",
			got[q8Key{"m_replayed", "LUSR"}])
	}
	if got[q8Key{"m_simple", "LUSR"}] != "btb" {
		t.Errorf("m_simple/LUSR → playlist_group = %q, want btb", got[q8Key{"m_simple", "LUSR"}])
	}
}

// TestQ8LUSRHistoryPlayer_KeepsBothRatingTypesOnRankedMatch — C.3 bis : un match CLASSÉ
// peut porter une ligne CSR ET une ligne LUSR. Le graphe trace DEUX séries et calcule
// ses deltas par (rating_type, playlist_group) : les deux lignes doivent être servies.
// C'est ce que match_skill_rank_latest (partition par match_id, priorité CSR > LUSR)
// ferait perdre — d'où la vue dédiée match_skill_rank_latest_by_type.
func TestQ8LUSRHistoryPlayer_KeepsBothRatingTypesOnRankedMatch(t *testing.T) {
	db := newPlayerFixtureDB(t)
	ctx := context.Background()

	start := time.Date(2026, 7, 4, 21, 0, 0, 0, time.UTC)
	written := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	rows := []struct {
		ratingType string
		value      float64
		group      string
	}{
		{"CSR", 1523, "Ranked Arena"},
		{"LUSR", 1284, "arena_slayer"},
		{"LUSR_V2", 1284, "arena_slayer"}, // audit interne : jamais projeté au graphe
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
			VALUES ('m_ranked', ?, ?, ?, ?, ?)`,
			r.ratingType, r.value, r.group, start, written); err != nil {
			t.Fatalf("insert %s: %v", r.ratingType, err)
		}
	}

	got := serveQ8(t, ctx, db)
	if len(got) != 2 {
		t.Fatalf("lignes servies = %d (%v), want 2 (la CSR et la LUSR ; LUSR_V2 exclue)", len(got), got)
	}
	if got[q8Key{"m_ranked", "CSR"}] != "Ranked Arena" {
		t.Errorf("la ligne CSR du match classé n'est pas servie ; got = %v", got)
	}
	if got[q8Key{"m_ranked", "LUSR"}] != "arena_slayer" {
		t.Errorf("la ligne LUSR du match classé n'est pas servie — le point LUSR "+
			"disparaîtrait de la série pleine du graphe ; got = %v", got)
	}
	if _, projected := got[q8Key{"m_ranked", "LUSR_V2"}]; projected {
		t.Errorf("LUSR_V2 est une étiquette d'audit : elle ne doit jamais être projetée ; got = %v", got)
	}
}

// q8Key identifie une ligne servie par Q8 : le graphe fait une série par couple.
type q8Key struct{ matchID, ratingType string }

// serveQ8 exécute Q8LUSRHistoryPlayer et retourne playlist_group par (match, type),
// en échouant si un couple est servi deux fois (une ligne périmée aurait passé).
func serveQ8(t *testing.T, ctx context.Context, db *sql.DB) map[q8Key]string {
	t.Helper()
	rows, err := db.QueryContext(ctx, Q8LUSRHistoryPlayer)
	if err != nil {
		t.Fatalf("Q8LUSRHistoryPlayer: %v", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[q8Key]string{}
	for rows.Next() {
		var matchID, ratingType string
		var ratingValue float64
		var tierLabel, playlistGroup, tier sql.NullString
		var subTier sql.NullInt16
		if err := rows.Scan(&matchID, &ratingType, &ratingValue, &tierLabel,
			&playlistGroup, &tier, &subTier); err != nil {
			t.Fatalf("scan: %v", err)
		}
		k := q8Key{matchID, ratingType}
		if _, dup := out[k]; dup {
			t.Fatalf("couple (%s, %s) servi DEUX fois : le graphe reçoit une ligne périmée",
				matchID, ratingType)
		}
		out[k] = playlistGroup.String
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return out
}
