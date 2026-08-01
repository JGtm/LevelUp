//go:build integration

// Package persist — weapon_shots_persister_test.go : la PORTE de publication et les refus.
//
// La porte est la propriete centrale de ce persister : elle vit ICI et nulle part ailleurs,
// l appelant fournit la REFERENCE et jamais le VERDICT. Ces tests verrouillent les quatre
// verdicts et les cinq refus.

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

func openWeaponShotsTestDB(t *testing.T) *sql.DB {
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

func iPtr(v int) *int { return &v }

func passeTirs(players ...WeaponShotsPlayer) WeaponShotsBatch {
	return WeaponShotsBatch{MatchID: "match-1", DecoderRev: "weaponshots-v1", Players: players}
}

func joueurValide() WeaponShotsPlayer {
	return WeaponShotsPlayer{
		PlayerIndex: 3,
		XUID:        "xuid(1)",
		ShotsFired:  iPtr(100),
		Weapons:     []WeaponShotCount{{WeaponID: 1000, Shots: 100}},
	}
}

// TestPorteDePublicationQuatreVerdicts — la porte est la seule copie de la regle des +-10 %.
// Chaque verdict s ecrit en toutes lettres, y compris le succes : une portee qui ne s ecrit
// que sur les echecs laisse croire que le succes n en a pas.
func TestPorteDePublicationQuatreVerdicts(t *testing.T) {
	cas := []struct {
		nom       string
		decode    int
		reference *int
		publiable bool
		motif     string
	}{
		{"dans la tolerance (ratio 1.0)", 100, iPtr(100), true, GateReasonWithinTolerance},
		{"borne basse exacte (0.90)", 90, iPtr(100), true, GateReasonWithinTolerance},
		{"borne haute exacte (1.10)", 110, iPtr(100), true, GateReasonWithinTolerance},
		{"sous la tolerance", 89, iPtr(100), false, GateReasonOutOfTolerance},
		{"au-dessus de la tolerance", 111, iPtr(100), false, GateReasonOutOfTolerance},
		{"reference absente", 50, nil, false, GateReasonNoReference},
		{"reference nulle, decodage nul", 0, iPtr(0), true, GateReasonWithinTolerance},
		{"reference nulle, decodage non nul", 42, iPtr(0), false, GateReasonReferenceZero},
	}
	for _, c := range cas {
		pub, motif := EvaluateShotsGate(c.decode, c.reference)
		if pub != c.publiable || motif != c.motif {
			t.Errorf("%s : (%v, %q), attendu (%v, %q)", c.nom, pub, motif, c.publiable, c.motif)
		}
	}
}

// TestVerdictEcritSurChaqueLigneDuJoueur — la porte est PAR JOUEUR parce que la reference l est
// (`shots_fired` est un total de joueur) : toutes les armes d un joueur partagent son verdict.
func TestVerdictEcritSurChaqueLigneDuJoueur(t *testing.T) {
	db := openWeaponShotsTestDB(t)

	bon := joueurValide()
	bon.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 60}, {WeaponID: 2000, Shots: 40}}

	refuse := joueurValide()
	refuse.PlayerIndex = 4
	refuse.XUID = "xuid(2)"
	refuse.ShotsFired = iPtr(500)
	refuse.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 10}}

	if err := NewWeaponShotsPersister(db).PersistPass(context.Background(),
		passeTirs(bon, refuse)); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	rows, err := db.Query(`SELECT player_index, publishable, gate_reason
		FROM match_weapon_shots_latest ORDER BY player_index, weapon_id`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type ligne struct {
		pub    bool
		motif  string
		compte int
	}
	vu := map[int]*ligne{}
	for rows.Next() {
		var idx int
		var pub bool
		var motif string
		if err := rows.Scan(&idx, &pub, &motif); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if vu[idx] == nil {
			vu[idx] = &ligne{pub: pub, motif: motif}
		}
		vu[idx].compte++
		if vu[idx].pub != pub || vu[idx].motif != motif {
			t.Errorf("joueur %d : verdict incoherent entre ses armes", idx)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if l := vu[3]; l == nil || !l.pub || l.compte != 2 || l.motif != GateReasonWithinTolerance {
		t.Errorf("joueur 3 : %+v, attendu 2 lignes publiables", l)
	}
	if l := vu[4]; l == nil || l.pub || l.motif != GateReasonOutOfTolerance {
		t.Errorf("joueur 4 : %+v, attendu refuse hors tolerance", l)
	}
}

// TestVueTirsRetientLaDernierePasseParMatch — meme unite de generation que les morts.
func TestVueTirsRetientLaDernierePasseParMatch(t *testing.T) {
	db := openWeaponShotsTestDB(t)
	ctx := context.Background()
	p := NewWeaponShotsPersister(db)

	a := joueurValide()
	a.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 50}, {WeaponID: 2000, Shots: 50}}
	if err := p.PersistPass(ctx, passeTirs(a)); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	b := joueurValide()
	b.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 100}}
	if err := p.PersistPass(ctx, passeTirs(b)); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var vue, table int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_weapon_shots_latest),
		(SELECT COUNT(*) FROM match_weapon_shots)`).Scan(&vue, &table); err != nil {
		t.Fatalf("select: %v", err)
	}
	if vue != 1 {
		t.Errorf("vue = %d lignes, attendu 1 — l arme 2000 de la passe A ne doit PAS survivre "+
			"a une passe qui ne la decode plus", vue)
	}
	if table != 3 {
		t.Errorf("table = %d, attendu 3 (append-only)", table)
	}
}

// TestRefusDuPersisterTirs — un test par propriete que le schema affirme.
func TestRefusDuPersisterTirs(t *testing.T) {
	cas := []struct {
		nom      string
		mutation func(*WeaponShotsPlayer)
		batch    func(*WeaponShotsBatch)
		extrait  string
	}{
		{"indice hors des 5 bits", func(p *WeaponShotsPlayer) { p.PlayerIndex = 32 }, nil, "5 bits"},
		{"indice negatif", func(p *WeaponShotsPlayer) { p.PlayerIndex = -1 }, nil, "5 bits"},
		{"reference negative", func(p *WeaponShotsPlayer) { p.ShotsFired = iPtr(-1) }, nil, "negatif"},
		{"arme sentinelle (grenade=0)", func(p *WeaponShotsPlayer) {
			p.Weapons = []WeaponShotCount{{WeaponID: 0, Shots: 5}}
		}, nil, "sentinelle"},
		{"arme sentinelle (vehicule=2)", func(p *WeaponShotsPlayer) {
			p.Weapons = []WeaponShotCount{{WeaponID: 2, Shots: 5}}
		}, nil, "sentinelle"},
		{"tirs a zero", func(p *WeaponShotsPlayer) {
			p.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 0}}
		}, nil, "absence de ligne"},
		{"doublon (indice, arme)", func(p *WeaponShotsPlayer) {
			p.Weapons = []WeaponShotCount{{WeaponID: 1000, Shots: 5}, {WeaponID: 1000, Shots: 7}}
		}, nil, "doublon"},
		{"DecoderRev vide", nil, func(b *WeaponShotsBatch) { b.DecoderRev = "" }, "DecoderRev"},
		{"MatchID vide", nil, func(b *WeaponShotsBatch) { b.MatchID = "" }, "MatchID"},
	}

	db := openWeaponShotsTestDB(t)
	ctx := context.Background()
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			pl := joueurValide()
			if c.mutation != nil {
				c.mutation(&pl)
			}
			b := passeTirs(pl)
			if c.batch != nil {
				c.batch(&b)
			}
			err := NewWeaponShotsPersister(db).PersistPass(ctx, b)
			if err == nil {
				t.Fatalf("refus attendu, la passe a ete acceptee")
			}
			if !strings.Contains(err.Error(), c.extrait) {
				t.Errorf("message = %q, attendu contenant %q", err.Error(), c.extrait)
			}
		})
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_weapon_shots`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d ligne(s) laissee(s) par des passes refusees — la validation doit passer "+
			"AVANT la transaction", n)
	}
}

// TestArmeBitDePoidsFortEcriteEtRelue — Fuel Rod SPNKr (0x9d6aaed242c9679f) : un `int64()`
// le rendrait NEGATIF et le CAST AS UBIGINT echouerait. Le passage en chaine decimale est la
// parade, et ce test la verrouille.
func TestArmeBitDePoidsFortEcriteEtRelue(t *testing.T) {
	db := openWeaponShotsTestDB(t)
	const fuelRod uint64 = 0x9d6aaed242c9679f

	pl := joueurValide()
	pl.Weapons = []WeaponShotCount{{WeaponID: fuelRod, Shots: 100}}
	if err := NewWeaponShotsPersister(db).PersistPass(context.Background(), passeTirs(pl)); err != nil {
		t.Fatalf("arme a bit de poids fort refusee: %v", err)
	}

	var relu string
	if err := db.QueryRow(`SELECT CAST(weapon_id AS VARCHAR) FROM match_weapon_shots_latest`).Scan(&relu); err != nil {
		t.Fatalf("select: %v", err)
	}
	if relu != "11343070829572876191" {
		t.Errorf("weapon_id relu = %s, attendu 11343070829572876191", relu)
	}
}

// TestPersistTirsViaBatchBuilder — chemin builder + no-op sur batch sans sous-batch.
func TestPersistTirsViaBatchBuilder(t *testing.T) {
	db := openWeaponShotsTestDB(t)
	ctx := context.Background()

	vide := NewBatchBuilder("halo_infinite", "Joueur", "xuid(1)", "test").Build()
	if err := NewWeaponShotsPersister(db).Persist(ctx, vide); err != nil {
		t.Fatalf("batch sans WeaponShots doit etre un no-op: %v", err)
	}

	pass := passeTirs(joueurValide())
	batch := NewBatchBuilder("halo_infinite", "Joueur", "xuid(1)", "test").SetWeaponShots(&pass).Build()
	if err := NewWeaponShotsPersister(db).Persist(ctx, batch); err != nil {
		t.Fatalf("Persist via builder: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_weapon_shots_latest`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 1 {
		t.Errorf("%d ligne(s), attendu 1", n)
	}
}
