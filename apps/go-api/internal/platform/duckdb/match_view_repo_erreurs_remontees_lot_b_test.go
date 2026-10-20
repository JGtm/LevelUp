//go:build integration

// Package duckdb — match_view_repo_erreurs_remontees_lot_b_test.go : E4.8b. Les lectures du lot B de
// la vue match (rang, CSR partagés, médias, stats attendues, historique, médailles et armes de tous
// les joueurs, stats du joueur, enrichissement, modèle d'assists, voisins, libellés de médailles) ne
// transforment plus une défaillance en « pas de donnée » : elle REMONTE. L'absence légitime (joueur
// sans ligne, relation que ce lecteur ne porte pas) se CONSTATE, et le test la fige aussi.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func lotBExec(t *testing.T, db *DB, sqlText string) {
	t.Helper()
	if _, err := db.Exec(context.Background(), sqlText); err != nil {
		t.Fatalf("fixture : %v\nSQL: %s", err, sqlText)
	}
}

func TestLotB_StatsDuJoueur_AbsenceVsPanne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	// Joueur sans ligne : absence constatée (structure vide, pas d'erreur).
	if s, err := repo.GetPlayerMatchStats(ctx, "xuid_absent", "m1"); err != nil || s == nil {
		t.Fatalf("absence : attendu (structure vide, nil), obtenu (%v, %v)", s, err)
	}
	// Match connu : les stats sont lues (la colonne backfill_bits fait partie du seed).
	if s, err := repo.GetPlayerMatchStats(ctx, pTestXUID, "m1"); err != nil || s == nil {
		t.Fatalf("lecture nominale : %v", err)
	}
	lotBExec(t, pdb.Player, `DROP VIEW match_participants`)
	if _, err := repo.GetPlayerMatchStats(ctx, pTestXUID, "m1"); err == nil {
		t.Fatal("relation absente : une erreur est attendue, pas une structure vide")
	}
}

func TestLotB_Enrichissement_AbsenceVsPanne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	if e, err := repo.GetMatchEnrichment(ctx, "m-inconnu"); err != nil || e == nil {
		t.Fatalf("absence : attendu (structure vide, nil), obtenu (%v, %v)", e, err)
	}
	lotBExec(t, pdb.Player, `DROP VIEW player_match_enrichment_latest`)
	if _, err := repo.GetMatchEnrichment(ctx, "m1"); err == nil {
		t.Fatal("vue absente : une erreur est attendue, pas une structure vide")
	}
}

func TestLotB_StatsAttendues_AbsenceVsPanne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	if s, err := repo.GetMatchExpectedStats(ctx, "m1", "xuid_absent"); err != nil || s != nil {
		t.Fatalf("absence : attendu (nil, nil), obtenu (%v, %v)", s, err)
	}
	lotBExec(t, pdb.Player, `DROP VIEW match_participants`)
	if _, err := repo.GetMatchExpectedStats(ctx, "m1", pTestXUID); err == nil {
		t.Fatal("relation absente : une erreur est attendue, pas nil")
	}
}

func TestLotB_HistoriqueDesMoyennes_Panne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	if _, err := repo.GetHistoryForAvg(ctx, pTestXUID); err != nil {
		t.Fatalf("lecture nominale GetHistoryForAvg : %v", err)
	}
	if _, err := repo.GetHistoryForAvgBulk(ctx, []string{pTestXUID}); err != nil {
		t.Fatalf("lecture nominale GetHistoryForAvgBulk : %v", err)
	}
	lotBExec(t, pdb.Player, `DROP VIEW match_participants`)
	if rows, err := repo.GetHistoryForAvg(ctx, pTestXUID); err == nil {
		t.Fatalf("GetHistoryForAvg : une erreur est attendue, obtenu %d lignes", len(rows))
	}
	if m, err := repo.GetHistoryForAvgBulk(ctx, []string{pTestXUID}); err == nil {
		t.Fatalf("GetHistoryForAvgBulk : une erreur est attendue, obtenu %v", m)
	}
}

func TestLotB_RangCompetitif_AbsenceRegistreEtPanne(t *testing.T) {
	ctx := context.Background()

	t.Run("match non classe = absence", func(t *testing.T) {
		repo := NewMatchViewRepo(newTestPlayerDB(t), pTestXUID)
		if r, err := repo.GetMatchSkillRank(ctx, "m-inconnu"); err != nil || r != nil {
			t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", r, err)
		}
	})
	t.Run("match absent du registre = repli sur le type brut", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DELETE FROM shared.match_registry WHERE match_id = 'm1'`)
		r, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchSkillRank(ctx, "m1")
		if err != nil || r == nil || r.RatingType != "CSR" {
			t.Fatalf("attendu (CSR via le repli, nil), obtenu (%+v, %v)", r, err)
		}
	})
	t.Run("vue de rang absente = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW match_skill_rank_latest`)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchSkillRank(ctx, "m1"); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
	t.Run("registre illisible = panne, pas un repli", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW match_registry`)
		r, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchSkillRank(ctx, "m1")
		if err == nil {
			t.Fatalf("une erreur est attendue, obtenu %+v", r)
		}
	})
}

func TestLotB_CSRPartages_AbsenceConstateeVsPanne(t *testing.T) {
	ctx := context.Background()

	t.Run("relation vide = absence", func(t *testing.T) {
		m, err := NewMatchViewRepo(newTestPlayerDB(t), pTestXUID).GetMatchSharedCSRs(ctx, "m1")
		if err != nil || m != nil {
			t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", m, err)
		}
	})
	t.Run("vue absente de ce lecteur = absence constatee par sonde", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW match_csrs_latest`)
		lotBExec(t, pdb.Player, `DROP VIEW match_csrs`)
		lotBExec(t, pdb.Player, `DROP TABLE shared.match_csrs`)
		m, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchSharedCSRs(ctx, "m1")
		if err != nil || m != nil {
			t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", m, err)
		}
	})
	t.Run("vue presente mais requete en echec = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `CREATE OR REPLACE VIEW match_csrs_latest AS SELECT 'm1' AS match_id`)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchSharedCSRs(ctx, "m1"); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
}

func TestLotB_MedaillesDeTousLesJoueurs_PanneEtLibellesDegrades(t *testing.T) {
	ctx := context.Background()
	graine := func(pdb *PlayerDB) {
		lotBExec(t, pdb.Player, `INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (1, 100, 'x1', 'm-lot-b', 2)`)
	}

	t.Run("relation absente = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW medals_earned`)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchBulkMedals(ctx, "m1"); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
	t.Run("referentiel de libelles en panne = medailles servies ET panne dite", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		graine(pdb)
		lotBExec(t, pdb.Metadata, `DROP TABLE medal_definitions`)
		repo := NewMatchViewRepo(pdb, pTestXUID)
		rows, err := repo.GetMatchBulkMedals(ctx, "m-lot-b")
		if !domain.IsSectionDegraded(err) {
			t.Fatalf("*domain.SectionDegradedError attendue, obtenu %v", err)
		}
		if len(rows) != 1 || rows[0].Label != "100" {
			t.Fatalf("les medailles doivent rester servies, libelle replie sur l'identifiant : %+v", rows)
		}
		mine, err := repo.GetMatchMedals(ctx, "x1", "m-lot-b")
		if !domain.IsSectionDegraded(err) || len(mine) != 1 {
			t.Fatalf("GetMatchMedals : donnees servies + SectionDegradedError attendues, obtenu (%v, %v)", mine, err)
		}
	})
	t.Run("referentiel sain = aucune erreur", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		graine(pdb)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchBulkMedals(ctx, "m-lot-b"); err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
	})
}

func TestLotB_LibellesDeMedailles_PanneRemonte(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)
	lotBExec(t, pdb.Metadata, `DROP TABLE medal_definitions`)
	if _, err := repo.lookupMedalMeta(ctx, []int64{100}); err == nil {
		t.Error("lookupMedalMeta : une erreur est attendue")
	}
	if m, err := repo.LookupMedalMetaByName(ctx, []string{"Killjoy"}); err == nil {
		t.Errorf("LookupMedalMetaByName : une erreur est attendue, obtenu %v", m)
	}
}

// citation_mappings est un référentiel PROPRE À UN TITRE : sa table absente est une absence constatée
// (Halo 5), une requête qui échoue sur une table présente est une panne.
func TestLotB_FallbackDeCitations_AbsenceConstateeVsPanne(t *testing.T) {
	ctx := context.Background()
	t.Run("table absente", func(t *testing.T) {
		db := openMemDB(t)
		m, err := lookupMedalCitationLabels(ctx, db, []int64{1})
		if err != nil || len(m) != 0 {
			t.Fatalf("attendu (vide, nil), obtenu (%v, %v)", m, err)
		}
	})
	t.Run("table presente mais illisible", func(t *testing.T) {
		db := openMemDB(t)
		lotBExec(t, db, `CREATE TABLE citation_mappings (x INTEGER)`)
		if _, err := lookupMedalCitationLabels(ctx, db, []int64{1}); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
}

func TestLotB_ArmesDeTousLesJoueurs_AbsenceConstateeEtPanne(t *testing.T) {
	ctx := context.Background()

	t.Run("relation absente de ce lecteur = absence constatee par sonde", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW v_weapon_kills`)
		lotBExec(t, pdb.Player, `DROP VIEW shared.v_weapon_kills`)
		rows, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchBulkWeaponKills(ctx, "m1")
		if err != nil || rows != nil {
			t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", rows, err)
		}
	})
	t.Run("relation presente mais requete en echec = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `CREATE OR REPLACE VIEW v_weapon_kills AS SELECT 'm1' AS match_id`)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchBulkWeaponKills(ctx, "m1"); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
	t.Run("source de degat : ligne illisible = panne, pas une ligne sautee", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW match_kill_events_latest`)
		lotBExec(t, pdb.Player, `CREATE VIEW match_kill_events_latest AS
			SELECT 'm1' AS match_id, 'x1' AS feed_killer_xuid, 'pas un entier' AS source_tag`)
		repo := NewMatchViewRepo(pdb, pTestXUID).WithKillSourceClassifier(fakeKillSourceClassifier{})
		if rows, err := repo.GetMatchBulkWeaponKills(ctx, "m1"); err == nil {
			t.Fatalf("une erreur est attendue, obtenu %v", rows)
		}
	})
}

func TestLotB_ModeleDAssists_AbsenceVsPanne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	// Table absente d'un player DB : panne (la migration la pose partout).
	if _, err := repo.GetPlayerAssistsModel(ctx, "Slayer"); err == nil {
		t.Fatal("table absente : une erreur est attendue")
	}
	lotBExec(t, pdb.Player, `CREATE TABLE player_assists_model (
		game_variant_name VARCHAR PRIMARY KEY, coef_intercept DOUBLE, coef_kills DOUBLE, coef_deaths DOUBLE,
		coef_damage_dealt DOUBLE, coef_damage_taken DOUBLE, coef_mmr_delta DOUBLE, r2 DOUBLE, n_samples INTEGER)`)
	// Mode sans modele : absence constatee.
	if m, err := repo.GetPlayerAssistsModel(ctx, "Slayer"); err != nil || m != nil {
		t.Fatalf("mode sans modele : attendu (nil, nil), obtenu (%v, %v)", m, err)
	}
	lotBExec(t, pdb.Player, `INSERT INTO player_assists_model VALUES ('Slayer', 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 40)`)
	if m, err := repo.GetPlayerAssistsModel(ctx, "Slayer"); err != nil || m == nil || m.N != 40 {
		t.Fatalf("modele present : attendu le modele, obtenu (%v, %v)", m, err)
	}
}

func TestLotB_VoisinsDeNavigation_AbsenceVsPanne(t *testing.T) {
	ctx := context.Background()

	t.Run("match hors scope = voisins vides", func(t *testing.T) {
		n, err := NewMatchViewRepo(newTestPlayerDB(t), pTestXUID).GetMatchNeighbors(ctx, pTestXUID, "m-inconnu")
		if err != nil || n == nil || n.TotalMatches != 0 {
			t.Fatalf("attendu (voisins vides, nil), obtenu (%+v, %v)", n, err)
		}
	})
	t.Run("relation absente = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.Player, `DROP VIEW match_registry`)
		repo := NewMatchViewRepo(pdb, pTestXUID)
		if n, err := repo.GetMatchNeighbors(ctx, pTestXUID, "m1"); err == nil {
			t.Fatalf("GetMatchNeighbors : une erreur est attendue, obtenu %+v", n)
		}
		spec := &domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Arena"}}
		if n, err := repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "m1", spec); err == nil {
			t.Fatalf("GetMatchNeighborsFiltered : une erreur est attendue, obtenu %+v", n)
		}
	})
}

func TestLotB_Medias_AbsenceVsPanne(t *testing.T) {
	ctx := context.Background()

	t.Run("base sociale non configuree = absence", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		pdb.SharedSocial = nil
		m, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchMedia(ctx, "m1")
		if err != nil || m != nil {
			t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", m, err)
		}
	})
	t.Run("vue absente = panne", func(t *testing.T) {
		pdb := newTestPlayerDB(t)
		lotBExec(t, pdb.SharedSocial, `DROP VIEW media_likes_latest`)
		if _, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchMedia(ctx, "m1"); err == nil {
			t.Fatal("une erreur est attendue")
		}
	})
}
