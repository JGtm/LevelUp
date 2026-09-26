//go:build integration

// Package duckdb — match_view_scoreboard_bomb_test.go : LES STATISTIQUES D'ASSAUT SUR LE
// TABLEAU DES SCORES.
//
// CE QU'IL FERME, et c'est la même classe de panne que G1 (la vue objectifs absente rendait
// TOUTE la vue Match illisible) : la lecture d'Assaut est une SECONDE requête sur une SECONDE
// vue, et elle doit être dégradable INDÉPENDAMMENT. Trois contrats :
//
//	(1) capability absente        aucune requête payée, aucune colonne exposée ;
//	(2) vue absente               scoreboard SERVI, colonnes d'Assaut vides, WARN explicite ;
//	(3) chemin nominal            les cinq colonnes rejoignent la ligne du joueur par xuid,
//	                              et une colonne NULL reste NULL (« absent n'est pas zéro »).
package duckdb

import (
	"context"
	"testing"
)

// insererStatsBombe pose une passe de statistiques d'Assaut pour `m1`. `bomb_carriers_killed`
// est laissé à NULL : c'est l'état réel du chantier, et le test l'exige tel quel.
func insererStatsBombe(t *testing.T, pdb *PlayerDB, matchID, xuid string) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(), `
		INSERT INTO shared.match_bomb_stats
			(id, match_id, xuid, bomb_detonations, bomb_arms, bomb_grabs,
			 time_as_bomb_carrier_seconds, bomb_carriers_killed, written_at)
		VALUES (1, ?, ?, 2, 1, 3, 41.5, NULL, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		matchID, xuid,
	); err != nil {
		t.Fatalf("INSERT match_bomb_stats: %v", err)
	}
}

// TestGetMatchScoreboard_BombStatsSansCapability_AucuneColonne : sans `film.bomb_stats`, la
// requête n'est même pas payée — les colonnes restent absentes ALORS QUE LA BASE LES PORTE.
// C'est le contrat du gate : un titre sans décodeur de film n'expose rien.
func TestGetMatchScoreboard_BombStatsSansCapability_AucuneColonne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	insererStatsBombe(t, pdb, "m1", pTestXUID)

	repo := NewMatchViewRepo(pdb, pTestXUID) // WithBombStats non appelé = faux
	rows, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 joueur, obtenu %d", len(rows))
	}
	if rows[0].Obj.HasBomb() {
		t.Errorf("colonnes d'Assaut exposées sans la capability : %+v", rows[0].Obj)
	}
}

// TestGetMatchScoreboard_BombStatsVueAbsente_ServiEtAvertit : la vue manque (DB non migrée) →
// le scoreboard reste SERVI, seules les colonnes d'Assaut manquent, et le WARN précède la
// dégradation (CLAUDE.md règle n°3).
func TestGetMatchScoreboard_BombStatsVueAbsente_ServiEtAvertit(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `DROP VIEW IF EXISTS match_bomb_stats_latest`); err != nil {
		t.Fatalf("suppression de la vue d'Assaut (fixture DB non migrée): %v", err)
	}

	buf := captureSlog(t)
	repo := NewMatchViewRepo(pdb, pTestXUID).WithBombStats(true)
	rows, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard doit RESTER servi sans la vue d'Assaut, erreur reçue: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("scoreboard attendu 1 joueur, obtenu %d — la dégradation a vidé le scoreboard", len(rows))
	}
	if rows[0].Obj.HasBomb() {
		t.Errorf("colonnes d'Assaut attendues VIDES sans la vue, obtenu %+v", rows[0].Obj)
	}
	if !hasWarnContaining(t, buf, "Assaut") {
		t.Errorf("aucun WARN sur la dégradation des stats d'Assaut ; logs=%s", buf.String())
	}
}

// TestGetMatchScoreboard_BombStatsPresentes_JointesParXUID : le chemin nominal. Les quatre
// mesures rejoignent la ligne du joueur, et la cinquième — NULL en base — reste ABSENTE.
func TestGetMatchScoreboard_BombStatsPresentes_JointesParXUID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	insererStatsBombe(t, pdb, "m1", pTestXUID)

	repo := NewMatchViewRepo(pdb, pTestXUID).WithBombStats(true)
	rows, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 joueur, obtenu %d", len(rows))
	}
	o := rows[0].Obj
	if !o.HasBomb() {
		t.Fatalf("bloc d'Assaut absent alors que la base le porte : %+v", o)
	}
	if o.BombDetonations == nil || *o.BombDetonations != 2 ||
		o.BombArms == nil || *o.BombArms != 1 ||
		o.BombGrabs == nil || *o.BombGrabs != 3 ||
		o.TimeAsBombCarrierSeconds == nil || *o.TimeAsBombCarrierSeconds != 41.5 {
		t.Errorf("mesures d'Assaut incorrectes : %+v", o)
	}
	// UNE COLONNE NULL RESTE NULL : la lire à zéro ferait croire à une mesure.
	if o.BombCarriersKilled != nil {
		t.Errorf("bomb_carriers_killed = %d, attendu nil (NULL en base)", *o.BombCarriersKilled)
	}
}
