//go:build integration

// Package duckdb — match_view_repo_annuaire_test.go : LA PARITÉ DES NOMS du lot A (plan perf
// « lectures par périmètre », 2026-09-26, ADR 0036 I1).
//
// La vue match (Q12, Q21, Q23), les événements de match (ResolveGamertags) et Relations (Q28,
// heatmap Q29) ne joignent plus v_gamertag_lookup : leurs noms viennent de l'annuaire en PORTÉE
// BASE (squad_repo_annuaire.go, DA.3). La référence est l'ANCIENNE expression des lecteurs sur la
// VRAIE vue canonique (nomSelonLaVue), plus le nom de chaque niveau écrit en clair
// (attendusAnnuaire, attendusPorteeBase). Les écarts acceptés (DA.4) sont épinglés un par un :
// (i) TestMatchView_Annuaire_Q21EcartNomme_XuidInconnu, (iii)
// TestAnnuairePorteeBase_EcartNomme_NomDeKillFeedVariable ; (ii), le bot hors de toutes les sources,
// par TestSquadRepo_Annuaire_BotHorsDeToutesLesSources (même cascade).
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/observability/timing"
)

// attendusPorteeBase : les deux xuids que SEULE la portée base nomme — leur nom est HORS des
// matchs de la lecture (ma1), comme les 11 couples (match, joueur) de la copie de production que
// la portée de la lecture perdait (journal L7, (b)).
var attendusPorteeBase = map[string]string{
	"x_ailleurs":   "NomAilleurs",   // participant sans nom sur ma1, nommé comme participant de mb1
	"x_kfailleurs": "NomKFAilleurs", // ni alias ni participant nommé : seul le kill-feed de mb2 le nomme
}

// seedPorteeBase : seedAnnuaire, plus x_ailleurs et x_kfailleurs, joueurs de ma1 (équipe du
// joueur) que rien ne nomme sur ma1 ; mq10, un troisième match commun (Relations ne garde que
// les récurrents) ; et une date sur les trois matchs (la heatmap écarte les heures NULL).
func seedPorteeBase(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	seedAnnuaire(t, pdb)
	seedSecondMatch(t, pdb)
	ctx := context.Background()
	participant := `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, 0)`
	for _, p := range [][3]string{
		{"ma1", "x_ailleurs", ""}, {"mq10", "x_ailleurs", ""}, {"mb1", "x_ailleurs", "NomAilleurs"},
		{"ma1", "x_kfailleurs", ""}, {"mq10", "x_kfailleurs", ""},
	} {
		execOnSharedDBs(t, pdb, ctx, participant, p[0], p[1], p[2])
	}
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_kill_events_latest
		(match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms)
		VALUES ('mb2', 'x_kfailleurs', 'NomKFAilleurs', 'x_autre', 'NomAutre', 1000)`)
	execOnSharedDBs(t, pdb, ctx, `UPDATE shared.match_registry SET start_time_utc = TIMESTAMPTZ '2026-09-01 20:00:00+00'
		WHERE match_id IN ('ma1', 'ma2', 'mq10')`)
	// Q12 écarte les lignes toutes nulles (kills/deaths/assists/score à 0) ; Q21 lit raw_json,
	// absent du schéma de test partagé.
	execOnSharedDBs(t, pdb, ctx, `UPDATE shared.match_participants SET kills = 1 WHERE match_id = 'ma1'`)
	execOnSharedDBs(t, pdb, ctx, `ALTER TABLE shared.highlight_events ADD COLUMN IF NOT EXISTS raw_json VARCHAR`)
	simulerJournalBrut(t, pdb)
}

// simulerJournalBrut pose la table brute du kill-feed que lit la localisation du repli en portée
// base (DA.10) : le harnais commun n'a qu'une version par ligne, la brute y est donc la `_latest`.
// brancherJournalVersionne la remplace par un vrai journal à passes.
func simulerJournalBrut(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(),
		`CREATE VIEW match_kill_events AS SELECT * FROM shared.match_kill_events_latest`); err != nil {
		t.Fatalf("table brute simulée : %v", err)
	}
}

// verifierNomPorteeBase : la parité avec la jointure, et le nom en clair quand il est connu.
func verifierNomPorteeBase(t *testing.T, pdb *PlayerDB, lecture, xuid, obtenu string) {
	t.Helper()
	verifierNom(t, pdb, lecture, xuid, obtenu)
	if attendu, ok := attendusPorteeBase[xuid]; ok && obtenu != attendu {
		t.Errorf("%s : xuid %s nommé %q, attendu %q (nom hors des matchs de la lecture)", lecture, xuid, obtenu, attendu)
	}
}

// TestMatchView_Annuaire_Q12MemeNomsQueLaVue : le tableau de score de ma1 — un niveau de la
// cascade par xuid, dont deux nommés HORS du match (participant d'un autre match, kill-feed d'un
// autre match) : la portée base les nomme comme la vue.
func TestMatchView_Annuaire_Q12MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	got, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchScoreboard(context.Background(), "ma1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard : %v", err)
	}
	// ma1 : le joueur, x_alias, x_alias_vide, x_part, x_triple, x_kfl, bid(1.0), bid(99.0),
	// x_ennemi, x_ailleurs, x_kfailleurs.
	if len(got) != 11 {
		t.Fatalf("%d lignes, attendu 11 : %+v", len(got), got)
	}
	vus := map[string]bool{}
	for _, s := range got {
		verifierNomPorteeBase(t, pdb, "Q12", s.XUID, s.Gamertag)
		vus[s.XUID] = true
	}
	for x := range attendusPorteeBase {
		if !vus[x] {
			t.Errorf("Q12 : %s absent du tableau de score", x)
		}
	}
}

// TestMatchView_Annuaire_Q23MemeNomsQueLaVue : les rencontres de ma1 (hors joueur et bots).
func TestMatchView_Annuaire_Q23MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	got, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchEncounters(context.Background(), "ma1", pTestXUID)
	if err != nil {
		t.Fatalf("GetMatchEncounters : %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("%d rencontres, attendu 8 : %+v", len(got), got)
	}
	for _, e := range got {
		verifierNomPorteeBase(t, pdb, "Q23", e.XUID, e.Gamertag)
	}
}

// TestMatchView_Annuaire_Q21EcartNomme_XuidInconnu : les events de ma1 portent le nom de la
// vue ; x_orphelin, qu'AUCUNE source ne connaît, reçoit le libellé masqué là où la jointure
// rendait NULL (l'écran affichait alors le xuid brut) — l'écart (i) de DA.4. Un event sans xuid
// garde un nom NULL.
func TestMatchView_Annuaire_Q21EcartNomme_XuidInconnu(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	execOnSharedDBs(t, pdb, context.Background(), `INSERT INTO shared.highlight_events (match_id, xuid, event_type, time_ms)
		VALUES ('ma1', NULL, 'kill', 3000), ('ma1', 'x_kfailleurs', 'kill', 4000)`)
	got, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchEvents(context.Background(), "ma1")
	if err != nil {
		t.Fatalf("GetMatchEvents : %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("%d events, attendu 8", len(got))
	}
	for _, e := range got {
		if e.XUID == nil {
			if e.Gamertag != nil {
				t.Errorf("event sans xuid nommé %q, attendu NULL", *e.Gamertag)
			}
			continue
		}
		if e.Gamertag == nil {
			t.Errorf("event de %s sans nom", *e.XUID)
			continue
		}
		if *e.XUID == "x_orphelin" {
			if *e.Gamertag != "Joueur elin" {
				t.Errorf("x_orphelin nommé %q, attendu « Joueur elin »", *e.Gamertag)
			}
			if vue := nomSelonLaVue(t, pdb, "x_orphelin"); vue != "Joueur elin" {
				t.Errorf("l'écart (i) suppose x_orphelin absent de la vue (COALESCE masqué), la vue rend %q", vue)
			}
			continue
		}
		verifierNomPorteeBase(t, pdb, "Q21", *e.XUID, *e.Gamertag)
	}
}

// TestGamertagRepo_ResolveGamertags_PorteeBase : la carte des xuids d'un match (DA.5) — les xuids
// NOMMÉS par la cascade, avec le nom de la vue (dont les deux nommés hors du match) ; x_rien,
// connu d'une source mais sans nom, et x_orphelin, inconnu, en sont ABSENTS (le front masque,
// même libellé qu'analysis.MaskedXuidLabel).
func TestGamertagRepo_ResolveGamertags_PorteeBase(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	xuids := []string{"x_alias", "x_part", "x_kfl", "x_ennemi", "x_kvp", "x_ailleurs", "x_kfailleurs",
		"bid(1.0)", "x_rien", "x_orphelin"}
	got, err := NewGamertagRepo(pdb.SharedReadDB()).ResolveGamertags(context.Background(), "ma1", xuids)
	if err != nil {
		t.Fatalf("ResolveGamertags : %v", err)
	}
	for _, x := range xuids[:8] {
		nom, ok := got[x]
		if !ok {
			t.Errorf("%s absent de la carte", x)
			continue
		}
		verifierNomPorteeBase(t, pdb, "ResolveGamertags", x, nom)
	}
	for _, x := range []string{"x_rien", "x_orphelin"} {
		if nom, ok := got[x]; ok {
			t.Errorf("%s nommé %q : un xuid que la cascade ne nomme pas doit être ABSENT", x, nom)
		}
	}
}

// TestRelations_Annuaire_MemeNomsQueLaVue : Relations (Q28, scopé ou non) et sa heatmap (Q29)
// nomment chaque joueur récurrent comme la vue, dont les deux nommés hors de l'historique lu.
func TestRelations_Annuaire_MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	ctx := context.Background()
	repo := NewCareerRepo(pdb)
	for _, scope := range [][]string{nil, {"ma1", "ma2", "mq10"}} {
		rel, err := repo.GetRelations(ctx, scope)
		if err != nil {
			t.Fatalf("GetRelations(%v) : %v", scope, err)
		}
		// Récurrents non-bots : x_alias, x_alias_vide, x_part, x_triple, x_kfl, x_ennemi, x_kvp,
		// x_rien, x_ailleurs, x_kfailleurs.
		if len(rel) != 10 {
			t.Fatalf("GetRelations(%v) : %d lignes, attendu 10 : %+v", scope, len(rel), rel)
		}
		for _, r := range rel {
			verifierNomPorteeBase(t, pdb, "Q28", r.XUID, r.Gamertag)
		}
		heat, err := repo.GetRelationsHeatmap(ctx, scope, 12)
		if err != nil {
			t.Fatalf("GetRelationsHeatmap(%v) : %v", scope, err)
		}
		if len(heat) != 10 {
			t.Fatalf("GetRelationsHeatmap(%v) : %d lignes, attendu 10 : %+v", scope, len(heat), heat)
		}
		for _, h := range heat {
			verifierNomPorteeBase(t, pdb, "Q29", h.XUID, h.Gamertag)
		}
	}
}

// TestMatchView_Annuaire_SectionsDeDuree : DA.7 — chaque lecture du lot et son annuaire ont leur
// section (des feuilles, cf. observability/timing).
func TestMatchView_Annuaire_SectionsDeDuree(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	ctx, chrono := timing.WithTimings(context.Background())
	mv := NewMatchViewRepo(pdb, pTestXUID)
	if _, err := mv.GetMatchScoreboard(ctx, "ma1"); err != nil {
		t.Fatalf("GetMatchScoreboard : %v", err)
	}
	if _, err := mv.GetMatchEvents(ctx, "ma1"); err != nil {
		t.Fatalf("GetMatchEvents : %v", err)
	}
	if _, err := mv.GetMatchEncounters(ctx, "ma1", pTestXUID); err != nil {
		t.Fatalf("GetMatchEncounters : %v", err)
	}
	if _, err := mv.GetMatchEncounterStats(ctx, "ma1", pTestXUID); err != nil {
		t.Fatalf("GetMatchEncounterStats : %v", err)
	}
	if _, err := NewGamertagRepo(pdb.SharedReadDB()).ResolveGamertags(ctx, "ma1", []string{"x_alias"}); err != nil {
		t.Fatalf("ResolveGamertags : %v", err)
	}
	repo := NewCareerRepo(pdb)
	if _, err := repo.GetRelations(ctx, nil); err != nil {
		t.Fatalf("GetRelations : %v", err)
	}
	if _, err := repo.GetRelationsHeatmap(ctx, nil, 12); err != nil {
		t.Fatalf("GetRelationsHeatmap : %v", err)
	}
	appels := map[string]int{}
	for _, s := range chrono.Snapshot() {
		appels[s.Name] = s.Calls
	}
	for _, nom := range []string{"match_scoreboard", "match_scoreboard_annuaire", "match_events",
		"match_events_annuaire", "match_encounters", "match_encounters_annuaire", "match_encounter_stats",
		"resolve_gamertags_annuaire", "relations", "relations_annuaire", "relations_heatmap",
		"relations_heatmap_annuaire"} {
		if appels[nom] != 1 {
			t.Errorf("section %q : %d appel(s), attendu 1 (sections : %v)", nom, appels[nom], appels)
		}
	}
}

// brancherJournalVersionne remplace, sur la connexion que lisent les repos, le journal simulé
// (une version par ligne) par un journal VERSIONNÉ : une table brute où un match a plusieurs
// passes de décodage, et une vue `_latest` qui n'en garde que la dernière — même règle que la
// migration (`decode_pass` de la ligne la plus récente par match, `written_at` puis `id`).
func brancherJournalVersionne(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	const cols = "match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms"
	for _, q := range []string{
		`CREATE TABLE shared.mke_versions (id INTEGER, match_id VARCHAR, decode_pass INTEGER,
			written_at TIMESTAMP, feed_killer_xuid VARCHAR, feed_killer_gamertag VARCHAR,
			victim_xuid VARCHAR, victim_gamertag VARCHAR, time_ms INTEGER)`,
		`CREATE OR REPLACE VIEW match_kill_events AS
			SELECT ` + cols + ` FROM shared.match_kill_events_latest
			UNION ALL SELECT ` + cols + ` FROM shared.mke_versions`,
		`CREATE OR REPLACE VIEW match_kill_events_latest AS
			SELECT ` + cols + ` FROM shared.match_kill_events_latest
			UNION ALL SELECT ` + cols + ` FROM (
				SELECT * FROM shared.mke_versions
				QUALIFY decode_pass = FIRST_VALUE(decode_pass) OVER (
					PARTITION BY match_id ORDER BY written_at DESC, id DESC))`,
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("journal versionné : %v\nSQL: %s", err, q)
		}
	}
}

// TestAnnuairePorteeBase_RepliLitLaDerniereVersion : DA.10 — le repli « toute la base » LOCALISE
// ses matchs dans la table brute mais LIT les noms dans `_latest`. Sur mw1 (un autre match que
// celui qu'on ouvre), une ANCIENNE passe nommait x_version « ZzAncienNom » et x_disparu
// « ZzDisparu » ; la dernière passe nomme x_version « AaNouveauNom » et ne porte plus x_disparu.
// Le nom rendu est celui de `_latest` (ou aucun) ; lire la brute rendrait le MAX des versions
// (« ZzAncienNom », « ZzDisparu »).
func TestAnnuairePorteeBase_RepliLitLaDerniereVersion(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	brancherJournalVersionne(t, pdb)
	ctx := context.Background()
	for _, x := range []string{pTestXUID, "x_version", "x_disparu"} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id, kills)
			VALUES ('mv1', ?, '', 2, 0, 1)`, x)
	}
	if _, err := pdb.Player.Exec(ctx, `INSERT INTO shared.mke_versions VALUES
		(1, 'mw1', 1, TIMESTAMP '2026-09-01 10:00:00', 'x_version', 'ZzAncienNom', 'x_autre', 'NomAutre', 1000),
		(2, 'mw1', 1, TIMESTAMP '2026-09-01 10:00:00', 'x_disparu', 'ZzDisparu', 'x_autre', 'NomAutre', 2000),
		(3, 'mw1', 2, TIMESTAMP '2026-09-02 10:00:00', 'x_version', 'AaNouveauNom', 'x_autre', 'NomAutre', 1000)`); err != nil {
		t.Fatalf("versions : %v", err)
	}
	attendus := map[string]string{"x_version": "AaNouveauNom", "x_disparu": "Joueur paru"}
	rows, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchScoreboard(ctx, "mv1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard : %v", err)
	}
	vus := 0
	for _, s := range rows {
		if attendu, ok := attendus[s.XUID]; ok {
			vus++
			if s.Gamertag != attendu {
				t.Errorf("Q12 : %s nommé %q, attendu %q (le nom de `_latest`)", s.XUID, s.Gamertag, attendu)
			}
			verifierNom(t, pdb, "Q12", s.XUID, s.Gamertag)
		}
	}
	if vus != 2 {
		t.Fatalf("%d xuids versionnés dans le tableau de score, attendu 2 : %+v", vus, rows)
	}
	carte, err := NewGamertagRepo(pdb.SharedReadDB()).ResolveGamertags(ctx, "mv1", []string{"x_version", "x_disparu"})
	if err != nil {
		t.Fatalf("ResolveGamertags : %v", err)
	}
	if carte["x_version"] != "AaNouveauNom" {
		t.Errorf("ResolveGamertags : x_version nommé %q, attendu « AaNouveauNom »", carte["x_version"])
	}
	if nom, ok := carte["x_disparu"]; ok {
		t.Errorf("ResolveGamertags : x_disparu nommé %q par une version périmée, attendu absent", nom)
	}
}

// TestAnnuairePorteeBase_RepliNommeLesVictimes : les deux branches VICTIME de la localisation du
// repli (revue adversariale du lot A). Sur mv2, x_vicmke et x_vickvp ne sont nommés par rien
// (ni alias, ni participant, ni kill-feed de mv2) ; hors de la lecture, x_vicmke n'est nommé QUE
// comme victime du journal canonique (mb3) et x_vickvp QUE comme victime de la table historique
// (mb4). La vue match et ResolveGamertags doivent leur rendre ces noms — la localisation doit
// donc trouver mb3 et mb4 par leur colonne victime.
func TestAnnuairePorteeBase_RepliNommeLesVictimes(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	ctx := context.Background()
	for _, x := range []string{pTestXUID, "x_vicmke", "x_vickvp"} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id, kills)
			VALUES ('mv2', ?, '', 2, 0, 1)`, x)
	}
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_kill_events_latest
		(match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms)
		VALUES ('mb3', 'x_tueur', 'NomTueur', 'x_vicmke', 'NomVicMKE', 1000)`)
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.killer_victim_pairs
		(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
		VALUES ('mb4', 'x_tueur', 'NomTueur', 'x_vickvp', 'NomVicKVP')`)
	attendus := map[string]string{"x_vicmke": "NomVicMKE", "x_vickvp": "NomVicKVP"}

	rows, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchScoreboard(ctx, "mv2")
	if err != nil {
		t.Fatalf("GetMatchScoreboard : %v", err)
	}
	vus := 0
	for _, s := range rows {
		if attendu, ok := attendus[s.XUID]; ok {
			vus++
			if s.Gamertag != attendu {
				t.Errorf("Q12 : %s nommé %q, attendu %q (victime hors de la lecture)", s.XUID, s.Gamertag, attendu)
			}
			verifierNom(t, pdb, "Q12", s.XUID, s.Gamertag)
		}
	}
	if vus != 2 {
		t.Fatalf("%d victimes dans le tableau de score, attendu 2 : %+v", vus, rows)
	}
	carte, err := NewGamertagRepo(pdb.SharedReadDB()).ResolveGamertags(ctx, "mv2", []string{"x_vicmke", "x_vickvp"})
	if err != nil {
		t.Fatalf("ResolveGamertags : %v", err)
	}
	for x, attendu := range attendus {
		if carte[x] != attendu {
			t.Errorf("ResolveGamertags : %s nommé %q, attendu %q (victime hors de la lecture)", x, carte[x], attendu)
		}
	}
}

// TestAnnuairePorteeBase_EcartNomme_NomDeKillFeedVariable : l'écart (iii) de DA.4, admis et
// épinglé ici (ADR 0036 I1 : un écart se nomme, se compte et s'épingle). x_varie n'a ni alias ni
// nom de participant ; le kill-feed `_latest` du match lu (mv3) le nomme « NomA », celui d'un autre
// match (mb5) « NomB ». L'annuaire lit d'abord le kill-feed des matchs de la lecture : Q12 et
// ResolveGamertags sur mv3 rendent « NomA », le MAX des matchs LUS. La vue prend le MAX de toute la
// base et rendrait « NomB ». Sur la copie de production, zéro cas parmi les lignes comparées
// (journal P2 du plan).
func TestAnnuairePorteeBase_EcartNomme_NomDeKillFeedVariable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPorteeBase(t, pdb)
	ctx := context.Background()
	for _, x := range []string{pTestXUID, "x_varie"} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id, kills)
			VALUES ('mv3', ?, '', 2, 0, 1)`, x)
	}
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_kill_events_latest
		(match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms)
		VALUES ('mv3', 'x_varie', 'NomA', ?, ?, 1000),
		       ('mb5', 'x_varie', 'NomB', 'x_autre', 'NomAutre', 1000)`, pTestXUID, pTestGamertag)

	if vue := nomSelonLaVue(t, pdb, "x_varie"); vue != "NomB" {
		t.Fatalf("la vue rend %q ; l'écart (iii) suppose le MAX de toute la base, « NomB »", vue)
	}
	rows, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchScoreboard(ctx, "mv3")
	if err != nil {
		t.Fatalf("GetMatchScoreboard : %v", err)
	}
	vu := false
	for _, s := range rows {
		if s.XUID == "x_varie" {
			vu = true
			if s.Gamertag != "NomA" {
				t.Errorf("Q12 : x_varie nommé %q, attendu « NomA » (MAX des matchs lus, écart (iii))", s.Gamertag)
			}
		}
	}
	if !vu {
		t.Fatalf("x_varie absent du tableau de score : %+v", rows)
	}
	carte, err := NewGamertagRepo(pdb.SharedReadDB()).ResolveGamertags(ctx, "mv3", []string{"x_varie"})
	if err != nil {
		t.Fatalf("ResolveGamertags : %v", err)
	}
	if carte["x_varie"] != "NomA" {
		t.Errorf("ResolveGamertags : x_varie nommé %q, attendu « NomA » (écart (iii))", carte["x_varie"])
	}
}
