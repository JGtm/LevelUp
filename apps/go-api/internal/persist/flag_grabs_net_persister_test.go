//go:build integration

// Package persist — flag_grabs_net_persister_test.go : ce que le persister des prises de
// drapeau ECRIT, ce qu'il REFUSE, et les trois proprietes qui comptent ici — UNE PASSE
// REMPLACE LA PRECEDENTE PAR LA VUE `_latest` (jamais par un UPDATE), LA PASSE EST L UNITE
// (un joueur que la nouvelle passe ne nomme plus est RETRACTE, il ne survit pas avec
// l ANCIENNE fenetre), et LA FENETRE VOYAGE AVEC LA MESURE.
//
// Le schema vient des MIGRATIONS REELLES (migration.RunForDB), jamais d'une DDL recopiee : une
// DDL de test recopiee derive sans que rien ne le signale.

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

func openFlagGrabsNetTestDB(t *testing.T) *sql.DB {
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

// lireLatest rend (brut, net, fenetre) d'un joueur, LU PAR LA VUE (ADR 0026).
func lireLatest(t *testing.T, db *sql.DB, matchID, xuid string) (int, int, int) {
	t.Helper()
	var brut, net, w int
	err := db.QueryRow(`SELECT flag_grabs_raw, flag_grabs_net, juggle_window_ms
		FROM match_flag_grabs_net_latest WHERE match_id = ? AND xuid = ?`, matchID, xuid).
		Scan(&brut, &net, &w)
	if err != nil {
		t.Fatalf("lecture _latest %s/%s: %v", matchID, xuid, err)
	}
	return brut, net, w
}

// comptePar compte les lignes rendues par la VUE pour un match.
func compteLatest(t *testing.T, db *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_flag_grabs_net_latest WHERE match_id = ?`,
		matchID).Scan(&n); err != nil {
		t.Fatalf("compte _latest %s: %v", matchID, err)
	}
	return n
}

func TestFlagGrabsNet_EcritEtRelitParLaVue(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()

	pass := FlagGrabsNetBatch{MatchID: "m-ctf", WindowMS: 1500, Openings: 85, Players: []FlagGrabsNetRow{
		{XUID: "xuid(1)", Raw: 27, Net: 4},
		{XUID: "xuid(2)", Raw: 3, Net: 3},
		// UN ZERO MESURE EST UNE MESURE : ce joueur du roster n'a jamais touche le drapeau.
		{XUID: "xuid(3)", Raw: 0, Net: 0},
	}}
	if err := NewFlagGrabsNetPersister(db).PersistPass(ctx, pass); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	if brut, net, w := lireLatest(t, db, "m-ctf", "xuid(1)"); brut != 27 || net != 4 || w != 1500 {
		t.Errorf("xuid(1) = (%d, %d, %d), want (27, 4, 1500)", brut, net, w)
	}
	// Un joueur sans jonglage : les deux comptes sont egaux, et c'est une mesure, pas un defaut.
	if brut, net, _ := lireLatest(t, db, "m-ctf", "xuid(2)"); brut != 3 || net != 3 {
		t.Errorf("xuid(2) = (%d, %d), want (3, 3)", brut, net)
	}
	// Le joueur a zero EXISTE en base : sans sa ligne, il serait indistinguable d'un joueur
	// d'un match sans film.
	if brut, net, _ := lireLatest(t, db, "m-ctf", "xuid(3)"); brut != 0 || net != 0 {
		t.Errorf("xuid(3) = (%d, %d), want (0, 0)", brut, net)
	}
	// Les ouvertures de l'oracle voyagent avec la passe, sur chaque ligne.
	var openings int
	if err := db.QueryRow(`SELECT openings FROM match_flag_grabs_net_latest
		WHERE match_id = 'm-ctf' AND xuid = 'xuid(3)'`).Scan(&openings); err != nil {
		t.Fatalf("lecture openings: %v", err)
	}
	if openings != 85 {
		t.Errorf("openings = %d, want 85", openings)
	}
}

// TestFlagGrabsNet_UnePasseRetracteLesJoueursQuElleNeNommePlus — LA propriete que
// `decode_pass` existe pour tenir, et que l'arbitrage par cle ne tenait PAS : un joueur absent
// de la nouvelle passe disparait de la vue, au lieu d'y survivre avec l'ANCIENNE fenetre.
func TestFlagGrabsNet_UnePasseRetracteLesJoueursQuElleNeNommePlus(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()
	p := NewFlagGrabsNetPersister(db)

	if err := p.PersistPass(ctx, FlagGrabsNetBatch{MatchID: "m", WindowMS: 1000, Openings: 9,
		Players: []FlagGrabsNetRow{
			{XUID: "a", Raw: 5, Net: 4}, {XUID: "b", Raw: 3, Net: 3},
		}}); err != nil {
		t.Fatalf("passe 1: %v", err)
	}
	// Passe 2, AUTRE fenetre, et `b` n'est plus nomme.
	if err := p.PersistPass(ctx, FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500, Openings: 9,
		Players: []FlagGrabsNetRow{{XUID: "a", Raw: 5, Net: 2}}}); err != nil {
		t.Fatalf("passe 2: %v", err)
	}
	if n := compteLatest(t, db, "m"); n != 1 {
		t.Fatalf("_latest rend %d lignes, want 1 — la passe precedente fuit dans la vue", n)
	}
	// Et la fenetre du scope est UNIQUE : c'est tout l'enjeu.
	rows, err := db.Query(`SELECT DISTINCT juggle_window_ms FROM match_flag_grabs_net_latest WHERE match_id = 'm'`)
	if err != nil {
		t.Fatalf("fenetres: %v", err)
	}
	defer func() { _ = rows.Close() }()
	fenetres := 0
	for rows.Next() {
		fenetres++
	}
	if fenetres != 1 {
		t.Errorf("%d fenetres distinctes dans la vue, want 1", fenetres)
	}
}

// TestFlagGrabsNet_UnePasseRemplaceLaPrecedente — la propriete anti-ART : « remplacer » une
// passe consiste a en ECRIRE une nouvelle. La table garde les deux lignes, la vue ne rend que
// la derniere. Aucun UPDATE, aucun DELETE.
func TestFlagGrabsNet_UnePasseRemplaceLaPrecedente(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()
	p := NewFlagGrabsNetPersister(db)

	if err := p.PersistPass(ctx, FlagGrabsNetBatch{MatchID: "m", WindowMS: 1000, Openings: 12,
		Players: []FlagGrabsNetRow{{XUID: "x", Raw: 10, Net: 7}}}); err != nil {
		t.Fatalf("passe 1: %v", err)
	}
	if err := p.PersistPass(ctx, FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500, Openings: 12,
		Players: []FlagGrabsNetRow{{XUID: "x", Raw: 10, Net: 5}}}); err != nil {
		t.Fatalf("passe 2: %v", err)
	}

	var lignes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_flag_grabs_net WHERE match_id = 'm'`).Scan(&lignes); err != nil {
		t.Fatalf("compte brut: %v", err)
	}
	if lignes != 2 {
		t.Errorf("table brute = %d lignes, want 2 (append-only : rien n'est efface)", lignes)
	}
	// LA FENETRE SUIT LA MESURE : la vue rend la passe 2, donc 1500 et non 1000.
	if brut, net, w := lireLatest(t, db, "m", "x"); brut != 10 || net != 5 || w != 1500 {
		t.Errorf("_latest = (%d, %d, %d), want (10, 5, 1500)", brut, net, w)
	}
}

func TestFlagGrabsNet_Refus(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()
	p := NewFlagGrabsNetPersister(db)

	cas := []struct {
		nom     string
		batch   FlagGrabsNetBatch
		attendu string
	}{
		{"match sans identifiant", FlagGrabsNetBatch{WindowMS: 1500,
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: 1, Net: 1}}}, "MatchID vide"},
		{"passe sans fenetre", FlagGrabsNetBatch{MatchID: "m",
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: 1, Net: 1}}}, "juggle_window_ms"},
		{"joueur sans xuid", FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500,
			Players: []FlagGrabsNetRow{{Raw: 1, Net: 1}}}, "XUID vide"},
		{"doublon de xuid", FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500,
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: 1, Net: 1}, {XUID: "x", Raw: 2, Net: 2}}}, "doublon"},
		{"compte negatif", FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500,
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: -1, Net: 0}}}, "negatif"},
		{"nettes > brutes", FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500,
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: 2, Net: 3}}}, "nettes > brutes"},
		{"ouvertures negatives", FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500, Openings: -1,
			Players: []FlagGrabsNetRow{{XUID: "x", Raw: 1, Net: 1}}}, "openings"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := p.PersistPass(ctx, c.batch)
			if err == nil {
				t.Fatalf("%s : accepte, attendu un refus", c.nom)
			}
			if !strings.Contains(err.Error(), c.attendu) {
				t.Errorf("%s : erreur = %v, attendu un message contenant %q", c.nom, err, c.attendu)
			}
		})
	}
	// Aucun refus n'a laisse de ligne derriere lui : la validation passe AVANT la transaction.
	var lignes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&lignes); err != nil {
		t.Fatalf("compte brut: %v", err)
	}
	if lignes != 0 {
		t.Errorf("%d lignes ecrites malgre les refus, want 0", lignes)
	}
}

// TestFlagGrabsNet_PasseVideNEcritRien — un match sans joueur nomme ne pose aucune ligne : la
// vue continue de servir la passe precedente, et « zero prise » ne s'ecrit jamais.
func TestFlagGrabsNet_PasseVideNEcritRien(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()
	if err := NewFlagGrabsNetPersister(db).PersistPass(ctx,
		FlagGrabsNetBatch{MatchID: "m", WindowMS: 1500}); err != nil {
		t.Fatalf("passe vide: %v", err)
	}
	var lignes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&lignes); err != nil {
		t.Fatalf("compte: %v", err)
	}
	if lignes != 0 {
		t.Errorf("%d lignes ecrites pour une passe vide, want 0", lignes)
	}
}

// TestFlagGrabsNet_PersistViaBatchEstUnNoOpSansSousBatch — le cablage BatchBuilder ne doit
// jamais ecrire quand le match n'a pas de calque de drapeau.
func TestFlagGrabsNet_PersistViaBatchEstUnNoOpSansSousBatch(t *testing.T) {
	db := openFlagGrabsNetTestDB(t)
	ctx := context.Background()
	if err := NewFlagGrabsNetPersister(db).Persist(ctx, &MatchBatch{}); err != nil {
		t.Fatalf("Persist sans sous-batch: %v", err)
	}
	var lignes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&lignes); err != nil {
		t.Fatalf("compte: %v", err)
	}
	if lignes != 0 {
		t.Errorf("%d lignes ecrites sans sous-batch, want 0", lignes)
	}
}
