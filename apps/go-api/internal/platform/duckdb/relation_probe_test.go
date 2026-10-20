//go:build integration

// relation_probe_test.go - E6.2a-F5 : une sonde de catalogue EN ERREUR (ici : connexion fermee)
// est une panne qui REMONTE, jamais « relation absente » (`false, nil`) ni une section vide.
// Mutation : `return false, err` -> `return false, nil` dans relationExists rend ces tests rouges.

package duckdb

import (
	"context"
	"testing"
)

func TestRelationExists_ErreurDeSonde_Remonte(t *testing.T) {
	db := openMemDBForProbe(t)
	present, err := relationExists(context.Background(), db, "match_bomb_stats_latest")
	if err != nil || present {
		t.Fatalf("relation absente d'une base saine : present=%v err=%v, attendu false, nil", present, err)
	}
	if _, err := db.Exec(`CREATE TABLE t_presente (id INTEGER)`); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	if present, err := relationExists(context.Background(), db, "t_presente"); err != nil || !present {
		t.Fatalf("relation presente : present=%v err=%v, attendu true, nil", present, err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("fermeture : %v", err)
	}
	present, err = relationExists(context.Background(), db, "t_presente")
	if err == nil {
		t.Fatalf("sonde sur connexion fermee : erreur attendue, obtenu present=%v err=nil", present)
	}
	if present {
		t.Errorf("une sonde en erreur ne dit jamais « presente » : present=true")
	}
}

// Les appelants de la sonde remontent l'erreur au lieu de rendre une section vide.
func TestRelationExists_AppelantsRemontentLErreurDeSonde(t *testing.T) {
	ctx := context.Background()
	// pdbSondeCassee : lecteur partage dont la connexion est fermee (toute sonde echoue).
	pdbSondeCassee := func(t *testing.T) *PlayerDB {
		pdb := newB4PlayerDB(t, "halo_infinite")
		if err := pdb.Shared.SQLDb().Close(); err != nil {
			t.Fatalf("fermeture du lecteur partage : %v", err)
		}
		return pdb
	}

	t.Run("GetMatchSharedCSRs", func(t *testing.T) {
		m, err := NewMatchViewRepo(pdbSondeCassee(t), b4XUIDA).GetMatchSharedCSRs(ctx, "arena1")
		if err == nil {
			t.Fatalf("erreur de sonde attendue, obtenu une section vide : %v", m)
		}
	})
	t.Run("GetMatchBulkWeaponKills", func(t *testing.T) {
		rows, err := NewMatchViewRepo(pdbSondeCassee(t), b4XUIDA).GetMatchBulkWeaponKills(ctx, "arena1")
		if err == nil {
			t.Fatalf("erreur de sonde attendue, obtenu une section vide : %v", rows)
		}
	})
	t.Run("loadMatchBombStats", func(t *testing.T) {
		out, err := loadMatchBombStats(ctx, pdbSondeCassee(t).Shared.SQLDb(), "arena1")
		if err == nil {
			t.Fatalf("erreur de sonde attendue, obtenu une section vide : %v", out)
		}
	})
	t.Run("lookupMedalCitationLabels", func(t *testing.T) {
		pdb := newB4PlayerDB(t, "halo_infinite")
		if err := pdb.Player.SQLDb().Close(); err != nil {
			t.Fatalf("fermeture : %v", err)
		}
		out, err := lookupMedalCitationLabels(ctx, pdb.Player, []int64{1})
		if err == nil {
			t.Fatalf("erreur de sonde attendue, obtenu %v", out)
		}
	})
}
