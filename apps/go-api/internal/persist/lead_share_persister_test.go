//go:build integration

package persist

// lead_share_persister_test.go — ce que le persister de la Mainmise ÉCRIT sur la base migrée
// RÉELLE : la passe, la ligne match, la vue `_latest`, le retrait d'une passe par la suivante, le
// refus avant toute écriture, le chemin BatchBuilder.

import (
	"context"
	"database/sql"
	"testing"
)

func comptesMainmise(t *testing.T, db *sql.DB, matchID string) (vue, brut int) {
	t.Helper()
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share_latest WHERE match_id = ?`, matchID).Scan(&vue); err != nil {
		t.Fatalf("comptage de la vue: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share WHERE match_id = ?`, matchID).Scan(&brut); err != nil {
		t.Fatalf("comptage brut: %v", err)
	}
	return vue, brut
}

func TestLeadSharePersister_EcritLaLigneMatchEtLesCamps(t *testing.T) {
	db := openPadTiersTestDB(t)
	if err := NewLeadSharePersister(db).PersistPass(context.Background(), passeMainmiseValide()); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	if vue, _ := comptesMainmise(t, db, "m1"); vue != 3 {
		t.Fatalf("%d lignes servies, attendu 3 (1 match + 2 camps)", vue)
	}
	var camp int
	var lead, measured, tie int64
	var changes int
	if err := db.QueryRow(`SELECT camp, lead_ms, measured_ms, tie_ms, lead_changes
		FROM match_lead_share_latest WHERE match_id = 'm1' AND row_kind = 'camp' AND camp = 0`).
		Scan(&camp, &lead, &measured, &tie, &changes); err != nil {
		t.Fatalf("ligne de camp: %v", err)
	}
	if lead != 1500 || measured != 5000 || tie != 3000 || changes != 2 {
		t.Errorf("ligne de camp : lead=%d mesuré=%d égalité=%d retournements=%d", lead, measured, tie, changes)
	}
}

func TestLeadSharePersister_UnRefusRetracteLesCampsDeLaPassePrecedente(t *testing.T) {
	db := openPadTiersTestDB(t)
	p := NewLeadSharePersister(db)
	ctx := context.Background()
	if err := p.PersistPass(ctx, passeMainmiseValide()); err != nil {
		t.Fatalf("première passe: %v", err)
	}
	refus := LeadShareBatch{MatchID: "m1", Reason: "score_not_covered", DocSchema: 71}
	if err := p.PersistPass(ctx, refus); err != nil {
		t.Fatalf("seconde passe: %v", err)
	}
	vue, brut := comptesMainmise(t, db, "m1")
	if vue != 1 || brut != 4 {
		t.Errorf("vue=%d (attendu 1, la ligne match seule) brut=%d (attendu 4, append-only)", vue, brut)
	}
	var measured bool
	var reason string
	if err := db.QueryRow(`SELECT measured, unmeasured_reason FROM match_lead_share_latest WHERE match_id = 'm1'`).
		Scan(&measured, &reason); err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if measured || reason != "score_not_covered" {
		t.Errorf("la vue sert measured=%v raison=%q, attendu non mesuré, score_not_covered", measured, reason)
	}
}

func TestLeadSharePersister_RefusNEcritRien(t *testing.T) {
	db := openPadTiersTestDB(t)
	in := passeMainmiseValide()
	in.Rows[1].Camp = 0 // doublon
	if err := NewLeadSharePersister(db).PersistPass(context.Background(), in); err == nil {
		t.Fatal("un doublon doit être refusé")
	}
	if _, brut := comptesMainmise(t, db, "m1"); brut != 0 {
		t.Errorf("%d ligne(s) écrite(s) par une passe refusée", brut)
	}
}

func TestLeadSharePersister_CheminBatchBuilder(t *testing.T) {
	db := openPadTiersTestDB(t)
	passe := passeMainmiseValide()
	batch := NewBatchBuilder("halo_infinite", "Joueur", "1", "test").SetLeadShare(&passe).Build()
	if err := NewLeadSharePersister(db).Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if vue, _ := comptesMainmise(t, db, "m1"); vue != 3 {
		t.Errorf("%d lignes servies, attendu 3", vue)
	}
	// Un batch sans Mainmise est un no-op.
	vide := NewBatchBuilder("halo_infinite", "Joueur", "1", "test").Build()
	if err := NewLeadSharePersister(db).Persist(context.Background(), vide); err != nil {
		t.Errorf("batch sans mainmise : %v", err)
	}
}
