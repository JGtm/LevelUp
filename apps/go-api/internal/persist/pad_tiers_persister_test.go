//go:build integration

// Package persist — pad_tiers_persister_test.go : ce que le persister des niveaux d'armes
// ECRIT, ce qu'il REFUSE, et les deux proprietes qui comptent ici — UNE PASSE REMPLACE LA
// PRECEDENTE PAR LA VUE `_latest` (jamais par un UPDATE), et LA PASSE EST L UNITE : un niveau
// que la nouvelle passe ne produit plus est RETRACTE, il ne survit pas a cote des nouveaux.
//
// CETTE SECONDE PROPRIETE EST LE COEUR DU SUJET. Le niveau d'un socle depend de la REFERENCE
// DES CARTES au moment de la projection : une carte ajoutee a la reference fait basculer des
// prises de « non classe » vers « terrain » ou « puissance ». Un arbitrage par cle laisserait
// les anciennes lignes survivre, et le total du niveau compterait deux fois la meme prise.
//
// Le schema vient des MIGRATIONS REELLES (migration.RunForDB), jamais d'une DDL recopiee.

package persist

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openPadTiersTestDB(t *testing.T) *sql.DB {
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

// lireNiveaux rend, LU PAR LA VUE (ADR 0026), la carte (xuid|tier|arme) -> prises.
func lireNiveaux(t *testing.T, db *sql.DB, matchID string) map[string]int {
	t.Helper()
	rows, err := db.Query(
		`SELECT xuid, tier, weapon_family, pickups FROM match_pad_pickups_by_tier_latest
		 WHERE match_id = ?`, matchID)
	if err != nil {
		t.Fatalf("lecture de la vue _latest: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var x, tier, arme string
		var n int
		if err := rows.Scan(&x, &tier, &arme, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[x+"|"+tier+"|"+arme] = n
	}
	return out
}

func passeTemoin() PadTiersBatch {
	return PadTiersBatch{
		MatchID: "m1", PadsConfirmed: 2, PadsTotal: 3, RandomStarts: false,
		Rows: []PadTierRow{
			{XUID: "a1", Tier: PadTierGround, WeaponFamily: "b619d84a", Pickups: 3},
			{XUID: "a1", Tier: PadTierPower, WeaponFamily: "9d6aaed2", Pickups: 1},
			{XUID: "b1", Tier: PadTierNoPickup},
		},
	}
}

func TestPadTiersPersister_EcritLaPasse(t *testing.T) {
	db := openPadTiersTestDB(t)
	p := NewPadTiersPersister(db)
	if err := p.PersistPass(context.Background(), passeTemoin()); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	got := lireNiveaux(t, db, "m1")
	if got["a1|"+PadTierGround+"|b619d84a"] != 3 {
		t.Errorf("prises de terrain d'Alpha = %d, attendu 3 (%v)", got["a1|"+PadTierGround+"|b619d84a"], got)
	}
	if got["a1|"+PadTierPower+"|9d6aaed2"] != 1 {
		t.Errorf("prises de puissance d'Alpha = %d, attendu 1 (%v)", got["a1|"+PadTierPower+"|9d6aaed2"], got)
	}
	// LE ZERO MESURE est EN BASE : sans lui, Bravo serait indistinguable d'un joueur d'un
	// match sans film.
	if _, ok := got["b1|"+PadTierNoPickup+"|"]; !ok {
		t.Errorf("la ligne %s de Bravo manque : %v", PadTierNoPickup, got)
	}
	// Les valeurs de MATCH sont ecrites sur chaque ligne.
	var confirmes, total int
	var aleatoire bool
	if err := db.QueryRow(
		`SELECT pads_confirmed, pads_total, random_starts FROM match_pad_pickups_by_tier_latest
		 WHERE match_id = 'm1' LIMIT 1`).Scan(&confirmes, &total, &aleatoire); err != nil {
		t.Fatalf("lecture des valeurs de match: %v", err)
	}
	if confirmes != 2 || total != 3 || aleatoire {
		t.Errorf("valeurs de match = %d/%d aleatoire=%v, attendu 2/3 false", confirmes, total, aleatoire)
	}
}

// TestPadTiersPersister_UnePasseRetracteLaPrecedente — LA PROPRIETE CENTRALE.
//
// Simule une reference de cartes COMPLETEE : la premiere passe ne classait rien, la seconde
// classe tout. Apres la seconde, il ne doit RIEN rester de la premiere.
func TestPadTiersPersister_UnePasseRetracteLaPrecedente(t *testing.T) {
	db := openPadTiersTestDB(t)
	p := NewPadTiersPersister(db)
	ctx := context.Background()

	avant := PadTiersBatch{
		MatchID: "m1", PadsConfirmed: 0, PadsTotal: 2,
		Rows: []PadTierRow{
			{XUID: "a1", Tier: PadTierUnclassified, WeaponFamily: "b619d84a", Pickups: 3},
			{XUID: "a1", Tier: PadTierUnclassified, WeaponFamily: "9d6aaed2", Pickups: 1},
		},
	}
	if err := p.PersistPass(ctx, avant); err != nil {
		t.Fatalf("premiere passe: %v", err)
	}
	if err := p.PersistPass(ctx, passeTemoin()); err != nil {
		t.Fatalf("seconde passe: %v", err)
	}

	got := lireNiveaux(t, db, "m1")
	for cle := range got {
		if strings.Contains(cle, PadTierUnclassified) {
			t.Errorf("une ligne de la passe PRECEDENTE survit : %s (%v)", cle, got)
		}
	}
	if len(got) != 3 {
		t.Errorf("%d lignes servies, attendu 3 (la passe entiere, et elle seule) : %v", len(got), got)
	}
	// La table BRUTE, elle, garde tout : rien n'est jamais supprime (append-only).
	var brutes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_pad_pickups_by_tier`).Scan(&brutes); err != nil {
		t.Fatalf("comptage brut: %v", err)
	}
	if brutes != 5 {
		t.Errorf("%d lignes en table brute, attendu 5 — rien ne doit etre supprime", brutes)
	}
}

func TestPadTiersPersister_Refuse(t *testing.T) {
	cas := []struct {
		nom     string
		batch   PadTiersBatch
		extrait string
	}{
		{"match sans identifiant", PadTiersBatch{Rows: passeTemoin().Rows}, "MatchID vide"},
		{"plus de socles confirmes que publies",
			PadTiersBatch{MatchID: "m", PadsConfirmed: 3, PadsTotal: 2}, "ne cree jamais de socle"},
		{"niveau hors vocabulaire", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "a", Tier: "quelque_chose", WeaponFamily: "x", Pickups: 1}}}, "hors vocabulaire"},
		{"xuid vide", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "", Tier: PadTierGround, WeaponFamily: "x", Pickups: 1}}}, "XUID vide"},
		{"doublon", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "a", Tier: PadTierGround, WeaponFamily: "x", Pickups: 1},
			{XUID: "a", Tier: PadTierGround, WeaponFamily: "x", Pickups: 2}}}, "doublon"},
		{"zero mesure qui porte une arme", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "a", Tier: PadTierNoPickup, WeaponFamily: "x"}}}, "ne porte aucune arme"},
		{"niveau sans arme", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "a", Tier: PadTierGround, Pickups: 1}}}, "sans famille d arme"},
		{"niveau a zero prise", PadTiersBatch{MatchID: "m", Rows: []PadTierRow{
			{XUID: "a", Tier: PadTierGround, WeaponFamily: "x"}}}, "a zero prise"},
	}
	db := openPadTiersTestDB(t)
	p := NewPadTiersPersister(db)
	for _, c := range cas {
		err := p.PersistPass(context.Background(), c.batch)
		if err == nil {
			t.Errorf("%s : accepte alors qu'il devait etre refuse", c.nom)
			continue
		}
		if !strings.Contains(err.Error(), c.extrait) {
			t.Errorf("%s : erreur %q, attendu un message contenant %q", c.nom, err, c.extrait)
		}
	}
	// Aucun refus ne laisse de ligne derriere lui : la validation passe AVANT la transaction.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_pad_pickups_by_tier`).Scan(&n); err != nil {
		t.Fatalf("comptage: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites malgre des passes refusees", n)
	}
}

// TestPadTiersPersister_PasseVideNEcritRien — une passe vide n'est pas une passe a zero : elle
// est ignoree, et la vue continue de servir la precedente.
func TestPadTiersPersister_PasseVideNEcritRien(t *testing.T) {
	db := openPadTiersTestDB(t)
	p := NewPadTiersPersister(db)
	ctx := context.Background()
	if err := p.PersistPass(ctx, passeTemoin()); err != nil {
		t.Fatalf("premiere passe: %v", err)
	}
	if err := p.PersistPass(ctx, PadTiersBatch{MatchID: "m1", PadsTotal: 3, PadsConfirmed: 2}); err != nil {
		t.Fatalf("passe vide: %v", err)
	}
	if got := lireNiveaux(t, db, "m1"); len(got) != 3 {
		t.Errorf("%d lignes servies apres une passe vide, attendu 3 (la precedente) : %v", len(got), got)
	}
}
