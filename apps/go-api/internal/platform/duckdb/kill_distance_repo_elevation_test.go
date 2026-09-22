//go:build integration

// Package duckdb — kill_distance_repo_elevation_test.go : tests de LoadMatchElevation.
//
// Même harnais que kill_distance_repo_test.go (vraies migrations shared, vrai registre
// d'armes). CE QUE CES TESTS VERROUILLENT, et qui n'existait pas avant le lot Y :
//
//  1. la ligne rendue porte LES DEUX IDENTITÉS (tueur ET victime) — sans la victime, aucune
//     lecture ne peut dire « mes morts » ;
//  2. le dénivelé rendu est le dénivelé PHYSIQUE `killer_z - victim_z`, non signé d'un point
//     de vue (l'inversion est une règle d'analyse, pas de lecture) ;
//  3. une source hors registre est écartée, comme dans LoadMatch.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func loadElevation(t *testing.T, pdb *PlayerDB, classifier port.KillSourceClassifier) []domain.MatchElevationKillRaw {
	t.Helper()
	repo := NewKillDistanceRepo(pdb, classifier)
	rows, err := repo.LoadMatchElevation(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("LoadMatchElevation: %v", err)
	}
	return rows
}

// Le tueur est 4 m au-dessus de sa victime, 3 m de côté : distance 5, dénivelé +4.
func TestKillElevation_IdentitesEtDeniveleBrut(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKillEvent(t, pdb, killEventFixture{
		pass: kscDecodeV1, publishable: true,
		killerXUID: kdXUIDA, victimXUID: kdXUIDB, tag: kscTagRifle, timeMS: 1000,
	})
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 4.0, 3.0, 0.0, 0.0)

	rows := loadElevation(t, pdb, fakeKillSourceClassifier{})
	if len(rows) != 1 {
		t.Fatalf("lignes = %d, attendu 1 : %+v", len(rows), rows)
	}
	r := rows[0]
	if r.KillerXUID != kdXUIDA || r.VictimXUID != kdXUIDB {
		t.Errorf("identités = %q -> %q, attendu %q -> %q", r.KillerXUID, r.VictimXUID, kdXUIDA, kdXUIDB)
	}
	if r.KillerGamertag != "Tueur" || r.VictimGamertag != "Victime" {
		t.Errorf("gamertags = %q / %q, attendu Tueur / Victime", r.KillerGamertag, r.VictimGamertag)
	}
	if !almostEqual(r.DistanceM, 5.0) {
		t.Errorf("distance = %v, attendu 5.0", r.DistanceM)
	}
	if !almostEqual(r.DeltaZ, 4.0) {
		t.Errorf("dénivelé = %v, attendu +4.0 BRUT (killer_z - victim_z, jamais inversé ici)", r.DeltaZ)
	}
	if r.TimeMS != 1000 {
		t.Errorf("instant = %d, attendu 1000 (horloge du match)", r.TimeMS)
	}
}

// Une victime BOT (xuid NULL) rend une ligne avec un xuid de victime VIDE — pas une erreur,
// pas une ligne écartée : le frag a eu lieu, et le nuage le montre.
func TestKillElevation_VictimeBotGardeSaLigne(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0)

	rows := loadElevation(t, pdb, fakeKillSourceClassifier{})
	if len(rows) != 1 || rows[0].VictimXUID != "" {
		t.Fatalf("attendu une ligne à victime non résolue, obtenu %+v", rows)
	}
}

// Sans classificateur (titre sans décodeur) : nil, nil — jamais une erreur.
func TestKillElevation_SansClassificateur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillDistanceRepo(pdb, nil)
	rows, err := repo.LoadMatchElevation(context.Background(), kscMatchID)
	if err != nil || rows != nil {
		t.Fatalf("attendu (nil, nil), obtenu (%+v, %v)", rows, err)
	}
	if _, err := repo.LoadMatchElevation(context.Background(), ""); err == nil {
		t.Error("matchID vide doit être une erreur (jamais un scan complet)")
	}
}
