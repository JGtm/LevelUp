//go:build integration

// Package persist — bomb_stats_persister_test.go : ce que le persister d'Assaut ECRIT, ce
// qu'il REFUSE, et la propriete qui compte le plus ici — ABSENT N'EST PAS ZERO.
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

func openBombStatsTestDB(t *testing.T) *sql.DB {
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

func fPtr(v float64) *float64 { return &v }

// joueurMesureTout : une ligne dont les cinq champs sont mesures.
func joueurMesureTout(xuid string) BombPlayerStatsRow {
	return BombPlayerStatsRow{
		XUID: xuid, Detonations: iPtr(2), Arms: iPtr(1), Grabs: iPtr(3),
		TimeAsCarrierSeconds: fPtr(41.5), CarriersKilled: iPtr(4),
	}
}

func faitArme(timeMS int, xuid string) BombEventRow {
	return BombEventRow{EventType: BombEventArmed, TimeMS: timeMS, XUID: xuid,
		Source: "ring", Confidence: "exact"}
}

func faitExplose(timeMS int, xuid string) BombEventRow {
	return BombEventRow{EventType: BombEventDetonated, TimeMS: timeMS, XUID: xuid,
		Source: "statborg", Confidence: "approx"}
}

// TestAbsentNEstPasZero — LA propriete centrale. Un champ non mesure (pointeur nil) s'ecrit
// NULL ; un champ mesure a zero s'ecrit 0. Les confondre transformerait « on n'a pas regarde »
// en « rien ne s'est passe », et un agregat sommerait ces faux zeros sans rien signaler.
func TestAbsentNEstPasZero(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()

	// Portage lu (grabs a 0 = un vrai zero mesure), detonations JAMAIS lues (nil).
	pass := BombStatsBatch{MatchID: "m-assaut", Players: []BombPlayerStatsRow{{
		XUID: "xuid(1)", Grabs: iPtr(0), TimeAsCarrierSeconds: fPtr(0),
	}}}
	if err := NewBombStatsPersister(db).PersistPass(ctx, pass); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var deto, grabs sql.NullInt64
	var secondes sql.NullFloat64
	err := db.QueryRow(`SELECT bomb_detonations, bomb_grabs, time_as_bomb_carrier_seconds
		FROM match_bomb_stats_latest WHERE match_id = 'm-assaut' AND xuid = 'xuid(1)'`).
		Scan(&deto, &grabs, &secondes)
	if err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	if deto.Valid {
		t.Errorf("bomb_detonations non mesure devrait etre NULL, vaut %d", deto.Int64)
	}
	if !grabs.Valid || grabs.Int64 != 0 {
		t.Errorf("bomb_grabs mesure a zero devrait valoir 0, got valid=%v v=%d", grabs.Valid, grabs.Int64)
	}
	if !secondes.Valid || secondes.Float64 != 0 {
		t.Errorf("time_as_bomb_carrier_seconds mesure a zero devrait valoir 0, got valid=%v v=%v",
			secondes.Valid, secondes.Float64)
	}
}

// TestVueRetientLaDernierePasse — append-only : la 2e passe n'ECRASE rien, elle s'ajoute, et la
// vue _latest ne rend que la derniere ligne par (match_id, xuid).
func TestVueRetientLaDernierePasse(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()
	p := NewBombStatsPersister(db)

	if err := p.PersistPass(ctx, BombStatsBatch{MatchID: "m1",
		Players: []BombPlayerStatsRow{{XUID: "xuid(1)", Detonations: iPtr(1)}}}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	if err := p.PersistPass(ctx, BombStatsBatch{MatchID: "m1",
		Players: []BombPlayerStatsRow{{XUID: "xuid(1)", Detonations: iPtr(3)}}}); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_bomb_stats WHERE match_id = 'm1'`).
		Scan(&brut); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if brut != 2 {
		t.Errorf("append-only : la table brute devrait garder les 2 lignes, en a %d", brut)
	}
	var n, deto int
	if err := db.QueryRow(`SELECT COUNT(*), MIN(bomb_detonations) FROM match_bomb_stats_latest
		WHERE match_id = 'm1'`).Scan(&n, &deto); err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	if n != 1 || deto != 3 {
		t.Errorf("_latest devrait rendre 1 ligne a 3 detonations, got n=%d deto=%d", n, deto)
	}
}

// TestFaitDateSansActeurResteUnFait — un armement que la jointure ne nomme pas est PUBLIE, sans
// aucune ligne d'acteur. Jamais un acteur devine, jamais un armement escamote.
func TestFaitDateSansActeurResteUnFait(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()

	pass := BombStatsBatch{MatchID: "m2",
		Players: []BombPlayerStatsRow{joueurMesureTout("xuid(7)")},
		Events: []BombEventRow{
			faitArme(299176, ""),           // slot non ponte : aucun acteur
			faitExplose(304106, "xuid(7)"), // detonateur nomme par le statborg
		}}
	if err := NewBombStatsPersister(db).PersistPass(ctx, pass); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var faits int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_objective_events
		WHERE match_id = 'm2' AND objective_type = 'bomb'`).Scan(&faits); err != nil {
		t.Fatalf("count faits: %v", err)
	}
	if faits != 2 {
		t.Errorf("les 2 faits datent doivent etre publies, %d ecrits", faits)
	}
	var acteurs int
	var role, xuid string
	if err := db.QueryRow(`SELECT COUNT(*), MIN(role), MIN(xuid)
		FROM match_objective_event_players WHERE match_id = 'm2'`).
		Scan(&acteurs, &role, &xuid); err != nil {
		t.Fatalf("count acteurs: %v", err)
	}
	if acteurs != 1 || role != "scorer" || xuid != "xuid(7)" {
		t.Errorf("un seul acteur attendu (scorer/xuid(7)), got n=%d role=%q xuid=%q",
			acteurs, role, xuid)
	}
	// Le type d'evenement doit rester dans le vocabulaire de la table.
	var types string
	if err := db.QueryRow(`SELECT string_agg(event_type, ',' ORDER BY seq)
		FROM match_objective_events WHERE match_id = 'm2'`).Scan(&types); err != nil {
		t.Fatalf("types: %v", err)
	}
	if types != "bomb_armed,bomb_detonated" {
		t.Errorf("event_type inattendus : %q", types)
	}
}

// TestSeqSuitLeMaximumDuMatch — la PK (match_id, seq) est PARTAGEE avec les autres producteurs
// de cette table. Repartir de 0 entrerait en collision avec eux ; ce test fige l'allocation a la
// suite du maximum deja present.
func TestSeqSuitLeMaximumDuMatch(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()

	for seq := 0; seq < 3; seq++ {
		if _, err := db.Exec(`INSERT INTO match_objective_events
			(match_id, seq, time_ms, objective_type, event_type, source, confidence)
			VALUES ('m3', ?, 1000, 'flag', 'capture', 'burst', 'exact')`, seq); err != nil {
			t.Fatalf("seed producteur tiers seq=%d: %v", seq, err)
		}
	}

	pass := BombStatsBatch{MatchID: "m3", Events: []BombEventRow{faitExplose(5000, "xuid(1)")}}
	if err := NewBombStatsPersister(db).PersistPass(ctx, pass); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var seq int
	if err := db.QueryRow(`SELECT seq FROM match_objective_events
		WHERE match_id = 'm3' AND objective_type = 'bomb'`).Scan(&seq); err != nil {
		t.Fatalf("lecture seq: %v", err)
	}
	if seq != 3 {
		t.Errorf("le fait de bombe devrait prendre seq=3 (a la suite du max 2), got %d", seq)
	}
}

// TestFaitsDejaPresentsNonReecrits — la table des faits n'a pas de semantique de generation : une
// 2e passe laisse les faits en place ET ecrit quand meme ses statistiques (elles, savent
// remplacer une generation).
func TestFaitsDejaPresentsNonReecrits(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()
	p := NewBombStatsPersister(db)

	passeA := BombStatsBatch{MatchID: "m4",
		Players: []BombPlayerStatsRow{{XUID: "xuid(1)", Detonations: iPtr(1)}},
		Events:  []BombEventRow{faitExplose(1000, "xuid(1)")}}
	if err := p.PersistPass(ctx, passeA); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	passeB := BombStatsBatch{MatchID: "m4",
		Players: []BombPlayerStatsRow{{XUID: "xuid(1)", Detonations: iPtr(2)}},
		Events:  []BombEventRow{faitExplose(1000, "xuid(1)"), faitArme(900, "xuid(1)")}}
	if err := p.PersistPass(ctx, passeB); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var faits int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_objective_events WHERE match_id = 'm4'`).
		Scan(&faits); err != nil {
		t.Fatalf("count faits: %v", err)
	}
	if faits != 1 {
		t.Errorf("les faits de la passe A devaient rester seuls en place, %d en base", faits)
	}
	var deto int
	if err := db.QueryRow(`SELECT bomb_detonations FROM match_bomb_stats_latest
		WHERE match_id = 'm4'`).Scan(&deto); err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	if deto != 2 {
		t.Errorf("les statistiques, elles, devaient etre rafraichies a 2, got %d", deto)
	}
}

// TestRefusEtAucuneLigneLaisseeDerriere — la validation passe AVANT la transaction : un refus
// n'ecrit rien, ni statistique ni fait.
func TestRefusEtAucuneLigneLaisseeDerriere(t *testing.T) {
	cas := []struct {
		nom    string
		pass   BombStatsBatch
		motif  string
		refuse bool
	}{
		{"match_id vide", BombStatsBatch{Players: []BombPlayerStatsRow{joueurMesureTout("xuid(1)")}},
			"MatchID vide", true},
		{"xuid vide", BombStatsBatch{MatchID: "mx", Players: []BombPlayerStatsRow{
			{XUID: "", Detonations: iPtr(1)}}}, "XUID vide", true},
		{"doublon de xuid", BombStatsBatch{MatchID: "mx", Players: []BombPlayerStatsRow{
			joueurMesureTout("xuid(1)"), joueurMesureTout("xuid(1)")}}, "doublon de xuid", true},
		{"compte negatif", BombStatsBatch{MatchID: "mx", Players: []BombPlayerStatsRow{
			{XUID: "xuid(1)", Grabs: iPtr(-1)}}}, "negatif", true},
		{"duree negative", BombStatsBatch{MatchID: "mx", Players: []BombPlayerStatsRow{
			{XUID: "xuid(1)", TimeAsCarrierSeconds: fPtr(-0.5)}}}, "negatif", true},
		{"ligne integralement NULL", BombStatsBatch{MatchID: "mx", Players: []BombPlayerStatsRow{
			{XUID: "xuid(1)"}}}, "aucune mesure", true},
		{"event_type inconnu", BombStatsBatch{MatchID: "mx", Events: []BombEventRow{
			{EventType: "bomb_defused", TimeMS: 10, Source: "s", Confidence: "c"}}}, "inconnu", true},
		{"instant negatif", BombStatsBatch{MatchID: "mx", Events: []BombEventRow{
			{EventType: BombEventArmed, TimeMS: -1, Source: "s", Confidence: "c"}}}, "negatif", true},
		{"source vide", BombStatsBatch{MatchID: "mx", Events: []BombEventRow{
			{EventType: BombEventArmed, TimeMS: 10, Confidence: "c"}}}, "obligatoires", true},
		{"confidence vide", BombStatsBatch{MatchID: "mx", Events: []BombEventRow{
			{EventType: BombEventArmed, TimeMS: 10, Source: "s"}}}, "obligatoires", true},
		{"doublon de fait", BombStatsBatch{MatchID: "mx", Events: []BombEventRow{
			faitArme(10, "xuid(1)"), faitArme(10, "xuid(1)")}}, "doublon", true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			db := openBombStatsTestDB(t)
			err := NewBombStatsPersister(db).PersistPass(context.Background(), c.pass)
			if err == nil {
				t.Fatalf("attendu un refus, got nil")
			}
			if !strings.Contains(err.Error(), c.motif) {
				t.Errorf("motif attendu %q, got %v", c.motif, err)
			}
			var stats, faits int
			if err := db.QueryRow(`SELECT
				(SELECT COUNT(*) FROM match_bomb_stats),
				(SELECT COUNT(*) FROM match_objective_events)`).Scan(&stats, &faits); err != nil {
				t.Fatalf("comptage: %v", err)
			}
			if stats != 0 || faits != 0 {
				t.Errorf("un refus ne doit rien laisser derriere lui, got stats=%d faits=%d",
					stats, faits)
			}
		})
	}
}

// TestPasseVideBombeNEcritRien — ecrire zero ligne serait indistinguable d'un match sans Assaut ; la
// vue continue de servir la passe precedente.
func TestPasseVideBombeNEcritRien(t *testing.T) {
	db := openBombStatsTestDB(t)
	if err := NewBombStatsPersister(db).PersistPass(context.Background(),
		BombStatsBatch{MatchID: "m5"}); err != nil {
		t.Fatalf("passe vide devrait etre un no-op, got %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_bomb_stats`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("aucune ligne attendue, %d ecrites", n)
	}
}

// TestCheminBatchBuilder — la charge posee par SetBombStats() arrive bien en base, et un batch
// sans charge est un no-op. Un setter dont la charge serait silencieusement jetee serait une
// perte de donnees MUETTE.
func TestCheminBatchBuilder(t *testing.T) {
	db := openBombStatsTestDB(t)
	ctx := context.Background()

	vide := NewBatchBuilder("halo_infinite", "GT", "xuid(1)", "test").Build()
	if err := NewBombStatsPersister(db).Persist(ctx, vide); err != nil {
		t.Fatalf("batch sans BombStats devrait etre un no-op: %v", err)
	}

	charge := NewBatchBuilder("halo_infinite", "GT", "xuid(1)", "test").
		SetBombStats(&BombStatsBatch{MatchID: "m6",
			Players: []BombPlayerStatsRow{joueurMesureTout("xuid(1)")},
			Events:  []BombEventRow{faitExplose(2000, "xuid(1)")}}).
		Build()
	if err := NewBombStatsPersister(db).Persist(ctx, charge); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	var stats, faits int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_bomb_stats_latest WHERE match_id = 'm6'),
		(SELECT COUNT(*) FROM match_objective_events WHERE match_id = 'm6')`).
		Scan(&stats, &faits); err != nil {
		t.Fatalf("comptage: %v", err)
	}
	if stats != 1 || faits != 1 {
		t.Errorf("charge du builder non persistee : stats=%d faits=%d", stats, faits)
	}
}
