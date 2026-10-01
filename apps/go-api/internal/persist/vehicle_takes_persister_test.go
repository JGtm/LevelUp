//go:build integration

package persist

// vehicle_takes_persister_test.go — ce que le persister de la ressource vehicules ECRIT sur la
// base migree REELLE : la passe, la ligne match, la vue `_latest`, le retrait d'une passe par la
// suivante, le refus avant toute ecriture, le chemin BatchBuilder.

import (
	"context"
	"database/sql"
	"testing"
)

func comptesVehicules(t *testing.T, db *sql.DB, matchID string) (vue, brut int) {
	t.Helper()
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_vehicle_takes_latest WHERE match_id = ?`, matchID).Scan(&vue); err != nil {
		t.Fatalf("comptage de la vue: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_vehicle_takes WHERE match_id = ?`, matchID).Scan(&brut); err != nil {
		t.Fatalf("comptage brut: %v", err)
	}
	return vue, brut
}

func TestVehicleTakesPersister_EcritLaLigneMatchEtLesPrises(t *testing.T) {
	db := openPadTiersTestDB(t)
	if err := NewVehicleTakesPersister(db).PersistPass(context.Background(), passeVehiculesValide()); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	if vue, _ := comptesVehicules(t, db, "m1"); vue != 3 {
		t.Fatalf("%d lignes servies, attendu 3 (1 match + 2 prises)", vue)
	}
	var kind string
	var camp, frags, unmatched, noXUID int
	if err := db.QueryRow(`SELECT row_kind, camp, frags_total, frags_unmatched, episodes_unnamed
		FROM match_vehicle_takes_latest WHERE match_id = 'm1' AND row_kind = 'match'`).
		Scan(&kind, &camp, &frags, &unmatched, &noXUID); err != nil {
		t.Fatalf("ligne match: %v", err)
	}
	if camp != -1 || frags != 3 || unmatched != 1 || noXUID != 1 {
		t.Errorf("ligne match : camp=%d frags_total=%d non_apparies=%d sans_xuid=%d", camp, frags, unmatched, noXUID)
	}
}

func TestVehicleTakesPersister_UnePasseRetracteLaPrecedente(t *testing.T) {
	db := openPadTiersTestDB(t)
	p := NewVehicleTakesPersister(db)
	ctx := context.Background()
	if err := p.PersistPass(ctx, passeVehiculesValide()); err != nil {
		t.Fatalf("premiere passe: %v", err)
	}
	// La passe suivante n'a plus aucune prise : la ligne match RETIENT la passe, les prises partent.
	suivante := VehicleTakesBatch{MatchID: "m1", Measured: true, FragsRead: true, DocSchema: 71}
	if err := p.PersistPass(ctx, suivante); err != nil {
		t.Fatalf("seconde passe: %v", err)
	}
	vue, brut := comptesVehicules(t, db, "m1")
	if vue != 1 || brut != 4 {
		t.Errorf("vue=%d (attendu 1, la ligne match seule) brut=%d (attendu 4, append-only)", vue, brut)
	}
}

func TestVehicleTakesPersister_RefusNEcritRien(t *testing.T) {
	db := openPadTiersTestDB(t)
	in := passeVehiculesValide()
	in.Rows[1] = in.Rows[0] // doublon
	if err := NewVehicleTakesPersister(db).PersistPass(context.Background(), in); err == nil {
		t.Fatal("un doublon doit etre refuse")
	}
	if _, brut := comptesVehicules(t, db, "m1"); brut != 0 {
		t.Errorf("%d ligne(s) ecrite(s) par une passe refusee", brut)
	}
}

func TestVehicleTakesPersister_CheminBatchBuilder(t *testing.T) {
	db := openPadTiersTestDB(t)
	passe := passeVehiculesValide()
	batch := NewBatchBuilder("halo_infinite", "Joueur", "1", "test").SetVehicleTakes(&passe).Build()
	if err := NewVehicleTakesPersister(db).Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if vue, _ := comptesVehicules(t, db, "m1"); vue != 3 {
		t.Errorf("%d lignes servies par le chemin du batch, attendu 3", vue)
	}
	vide := NewBatchBuilder("halo_infinite", "Joueur", "1", "test").Build()
	if err := NewVehicleTakesPersister(db).Persist(context.Background(), vide); err != nil {
		t.Fatalf("batch sans vehicules: %v", err)
	}
	if _, brut := comptesVehicules(t, db, "m1"); brut != 3 {
		t.Errorf("un batch sans vehicules a ecrit une passe : brut=%d", brut)
	}
}
