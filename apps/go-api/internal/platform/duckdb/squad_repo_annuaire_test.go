//go:build integration

// Package duckdb — squad_repo_annuaire_test.go : LA PARITE DES NOMS du lot perf L2.
//
// Q29 (LoadTopTeammates), Q32 (LoadImpactEvents) et Q32b (LoadMainTeamParticipants) ne
// joignent plus v_gamertag_lookup : leurs noms viennent de l'annuaire de la lecture
// (squad_repo_annuaire.go). La reference de ces tests est l'ANCIENNE expression des lecteurs,
// `COALESCE(vg.gamertag, 'Joueur ' || RIGHT(xuid, 4))` sur la VRAIE vue canonique
// (analysis.GamertagLookupViewSQL — le schema de test n'en porte qu'une version reduite aux
// alias), evaluee sur la meme base : chaque ligne rendue doit porter le nom que la jointure
// lui aurait donne.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
)

// seedAnnuaire pose un niveau de la cascade par xuid (tous dans l'equipe 0 du joueur, sauf
// x_ennemi), sur deux matchs « avec amis » ma1 / ma2.
func seedAnnuaire(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, analysis.GamertagLookupViewSQL()); err != nil {
		t.Fatalf("vue canonique : %v", err)
	}
	participant := `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, ?)`
	for _, p := range []struct {
		match, xuid, gamertag string
		team                  int
	}{
		{"ma1", pTestXUID, pTestGamertag, 0},
		{"ma1", "x_alias", "", 0},                      // niveau 2 : alias
		{"ma1", "x_alias_vide", "NomPartAliasVide", 0}, // alias vide -> niveau 3
		{"ma1", "x_part", "NomPart", 0},                // niveau 3 : participant
		{"ma1", "x_triple", "NomPartTriple", 0},        // alias ET participant ET kill-feed : l'alias gagne
		{"ma1", "x_kfl", "", 0},                        // niveau 4 : journal canonique (_latest)
		{"ma1", "bid(1.0)", "", 0},                     // niveau 1 : bot connu
		{"ma1", "bid(99.0)", "", 0},                    // niveau 1 : bot inconnu -> xuid tel quel
		{"ma1", "x_ennemi", "", 1},                     // equipe adverse, nomme par le kill-feed (victime)
		{"ma2", pTestXUID, pTestGamertag, 0},
		{"ma2", "x_kvp", "", 0},  // niveau 4 : table historique seule
		{"ma2", "x_rien", "", 0}, // niveau 5 : libelle masque
		{"ma2", "x_part", "NomPart", 0},
	} {
		execOnSharedDBs(t, pdb, ctx, participant, p.match, p.xuid, p.gamertag, p.team)
	}
	for _, id := range []string{"ma1", "ma2"} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry (match_id) VALUES (?)`, id)
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment
			(match_id, performance_score, session_id, session_label, dominance_flag, is_with_friends, is_excluded)
			VALUES (?, 50.0, 2, 'Session 2', 0, TRUE, FALSE)`, id); err != nil {
			t.Fatalf("enrichment %s : %v", id, err)
		}
	}
	for _, a := range [][2]string{{"x_alias", "NomAlias"}, {"x_alias_vide", ""}, {"x_triple", "NomAliasTriple"}} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, a[0], a[1])
	}
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_kill_events_latest
		(match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms)
		VALUES ('ma1', 'x_kfl', 'NomKFL', 'x_ennemi', 'NomEnnemi', 1000),
		       ('ma1', 'x_triple', 'NomKFTriple', 'x_ennemi', 'NomEnnemi', 2000)`)
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.killer_victim_pairs
		(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
		VALUES ('ma2', 'x_kvp', 'NomKVP', 'x_rien', '')`)
	// highlight_events : le lobby de Q32, dont un xuid qu'AUCUNE source ne connait.
	for _, h := range [][2]string{
		{"ma1", pTestXUID}, {"ma1", "x_alias"}, {"ma1", "x_kfl"}, {"ma1", "x_ennemi"},
		{"ma1", "x_orphelin"}, {"ma1", "bid(99.0)"}, {"ma2", "x_kvp"}, {"ma2", "x_rien"},
	} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.highlight_events (match_id, xuid, event_type, time_ms)
			VALUES (?, ?, 'kill', 1000)`, h[0], h[1])
	}
}

// nomSelonLaVue rejoue l'ANCIENNE expression des lecteurs sur la vraie vue canonique.
func nomSelonLaVue(t *testing.T, pdb *PlayerDB, xuid string) string {
	t.Helper()
	var nom string
	if err := pdb.Player.QueryRow(context.Background(), `
		SELECT COALESCE(vg.gamertag, ('Joueur ' || RIGHT(t.x, 4)))
		FROM (SELECT ?::VARCHAR AS x) t
		LEFT JOIN v_gamertag_lookup vg ON vg.xuid = t.x`, xuid).Scan(&nom); err != nil {
		t.Fatalf("nom selon la vue (%s) : %v", xuid, err)
	}
	return nom
}

// attendusAnnuaire : le nom de chaque niveau, ecrit en clair — si la vue ET l'annuaire se
// dereglaient ensemble, la parite seule ne le verrait pas.
var attendusAnnuaire = map[string]string{
	"x_alias":      "NomAlias",
	"x_alias_vide": "NomPartAliasVide",
	"x_part":       "NomPart",
	"x_triple":     "NomAliasTriple",
	"x_kfl":        "NomKFL",
	"x_kvp":        "NomKVP",
	"x_ennemi":     "NomEnnemi",
	"x_rien":       "Joueur rien",
	"x_orphelin":   "Joueur elin",
	"bid(1.0)":     "343 Meowlnir",
	"bid(99.0)":    "bid(99.0)",
}

func verifierNom(t *testing.T, pdb *PlayerDB, lecture, xuid, obtenu string) {
	t.Helper()
	if vue := nomSelonLaVue(t, pdb, xuid); obtenu != vue {
		t.Errorf("%s : xuid %s nomme %q, la jointure sur la vue rendait %q", lecture, xuid, obtenu, vue)
	}
	if attendu, ok := attendusAnnuaire[xuid]; ok && obtenu != attendu {
		t.Errorf("%s : xuid %s nomme %q, attendu %q", lecture, xuid, obtenu, attendu)
	}
}

func TestSquadRepo_Annuaire_Q32bMemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	got, err := NewSquadRepo(pdb).LoadMainTeamParticipants(context.Background(), pTestXUID, []string{"ma1", "ma2"})
	if err != nil {
		t.Fatalf("LoadMainTeamParticipants : %v", err)
	}
	if len(got) != 12 {
		t.Fatalf("%d allies, attendu 12 (8 sur ma1, 4 sur ma2 ; x_ennemi exclu)", len(got))
	}
	for _, a := range got {
		verifierNom(t, pdb, "Q32b", a.XUID, a.Gamertag)
	}
}

func TestSquadRepo_Annuaire_Q32MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	got, err := NewSquadRepo(pdb).LoadImpactEvents(context.Background(), []string{"ma1", "ma2"})
	if err != nil {
		t.Fatalf("LoadImpactEvents : %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("%d events, attendu 8", len(got))
	}
	for _, e := range got {
		verifierNom(t, pdb, "Q32", e.XUID, e.Gamertag)
	}
}

func TestSquadRepo_Annuaire_Q29MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	got, err := NewSquadRepo(pdb).LoadTopTeammates(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadTopTeammates : %v", err)
	}
	// Coequipiers non-bots de l'equipe du joueur sur ma1 + ma2 : x_alias, x_alias_vide,
	// x_part, x_triple, x_kfl, x_kvp, x_rien (les bots sont ecartes par Q29).
	if len(got) != 7 {
		t.Fatalf("%d coequipiers, attendu 7 : %+v", len(got), got)
	}
	for _, r := range got {
		verifierNom(t, pdb, "Q29", r.XUID, r.Gamertag)
	}
}

// TestSquadRepo_Annuaire_BotHorsDeToutesLesSources : le SEUL ecart assume avec la jointure.
// Un bot connu que ni les participants ni le kill-feed ne portent n'etait pas dans la vue :
// la jointure rendait « Joueur 0.0) ». L'annuaire applique la regle des bots (BotSQLCase)
// sans condition et rend son nom officiel. Aucun bot dans highlight_events sur la base de
// production (mesure du 2026-09-23) : l'ecart est theorique, et il retire un pseudo-xuid.
func TestSquadRepo_Annuaire_BotHorsDeToutesLesSources(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	execOnSharedDBs(t, pdb, context.Background(), `INSERT INTO shared.highlight_events (match_id, xuid, event_type, time_ms)
		VALUES ('ma2', 'bid(0.0)', 'kill', 2000)`)
	got, err := NewSquadRepo(pdb).LoadImpactEvents(context.Background(), []string{"ma2"})
	if err != nil {
		t.Fatalf("LoadImpactEvents : %v", err)
	}
	for _, e := range got {
		if e.XUID != "bid(0.0)" {
			continue
		}
		if e.Gamertag != "343 Ritzy" {
			t.Errorf("bot hors sources nomme %q, attendu 343 Ritzy", e.Gamertag)
		}
		if vue := nomSelonLaVue(t, pdb, e.XUID); vue != "Joueur 0.0)" {
			t.Errorf("la jointure rendait %q ; l'ecart documente suppose « Joueur 0.0) »", vue)
		}
		return
	}
	t.Fatal("event du bot absent")
}
