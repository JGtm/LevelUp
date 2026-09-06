// Package duckdb — tactical_repo_cartes_test.go : la GRILLE D'ENTREE de l'onglet
// Tactique (MapsPlayed) et ce qui la conditionne — resolution du nom FR par
// `metadata.asset_translations`, filtre de l'Explorateur, masquage Campagne — plus
// les DEGRADATIONS du lecteur (aucune donnee, entrees vides, tables absentes).
//
// Scinde de tactical_repo_test.go le 2026-09-06 : celui-ci avait franchi les 500
// lignes sous les ajouts de la revue (meme discipline que la scission de
// merge_test.go en phase 1). La fixture partagee reste chez le voisin, meme paquet.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
)

// TestTacticalRepo_MapsPlayed : cartes jouees, comptes et decomposition V/D, dans
// l'ordre matchs decroissants. m4 (joue par un tiers) n'y figure pas.
func TestTacticalRepo_MapsPlayed(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("cartes = %d, want 2 : %+v", len(rows), rows)
	}
	if rows[0].MapID != tacCarteA || rows[0].Matchs != 2 {
		t.Fatalf("premiere carte = %+v, want %s a 2 matchs (tri matchs decroissants)", rows[0], tacCarteA)
	}
	if rows[0].Victoires != 1 || rows[0].Defaites != 1 {
		t.Errorf("carte A = %d V / %d D, want 1/1", rows[0].Victoires, rows[0].Defaites)
	}
	if rows[0].MapName != tacCarteA+"_en" {
		t.Errorf("libelle EN = %q, want %q", rows[0].MapName, tacCarteA+"_en")
	}
	// LE nom FR vient de metadata.asset_translations, jamais de match_registry
	// (colonne systematiquement NULLE en prod) — R3.
	if rows[0].MapNameFR != "Les Rues" {
		t.Errorf("libelle FR = %q, want %q (resolu par asset_translations)", rows[0].MapNameFR, "Les Rues")
	}
	if rows[1].MapID != tacCarteB || rows[1].Matchs != 1 || rows[1].Victoires != 1 {
		t.Errorf("seconde carte = %+v, want %s a 1 match / 1 V", rows[1], tacCarteB)
	}
	// Carte sans traduction : nom FR VIDE, sans erreur. L'appelant retombera sur l'EN.
	if rows[1].MapNameFR != "" {
		t.Errorf("carte sans traduction : MapNameFR = %q, want vide", rows[1].MapNameFR)
	}
}

// TestTacticalRepo_SansMetadata_NomFRVide : une metadata absente (ou une lecture en
// echec) laisse le nom FR vide sans faire echouer la grille — best-effort assume.
func TestTacticalRepo_SansMetadata_NomFRVide(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	pdb.Metadata = nil

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MapsPlayed sans metadata: %v", err)
	}
	if len(rows) != 2 || rows[0].MapNameFR != "" {
		t.Errorf("sans metadata : %d cartes, FR = %q — want 2 cartes servies, FR vide",
			len(rows), rows[0].MapNameFR)
	}
}

// TestTacticalRepo_Campagne_Masquee : les matchs de Campagne d'un joueur Halo 5
// (~287 lignes historiques en prod) n'entrent NI dans la grille des cartes NI dans
// l'univers des rasters — l'Explorateur les masque deja, l'onglet Tactique ne peut
// pas dire autre chose du meme historique.
//
// Le masquage est title-aware et pilote par la DONNEE du titre (ses game_variant_id
// de Campagne, `analysis.CampaignExcludedVariantIDs`), jamais par une comparaison de
// slug : c'est pour cela que la fixture ouvre un PlayerDB au titre `halo_5`.
func TestTacticalRepo_Campagne_Masquee(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	pdb.TitleSlug = "halo_5"
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	campagne := analysis.CampaignExcludedVariantIDs("halo_5")
	if len(campagne) == 0 {
		t.Fatal("aucun game_variant_id de Campagne pour halo_5 : la source unique a bouge")
	}

	// Un match d'arene, avec une mort mesuree.
	tacMatch(t, pdb, "m1", tacCarteA, base)
	tacParticipant(t, pdb, "m1", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "m1", tacXUIDAdv, 1, domain.OutcomeLoss)
	tacKill(t, pdb, "m1", tacXUIDMoi, tacXUIDAdv, 1000, true)
	tacPos(t, pdb, "m1", tacXUIDMoi, 1000, 2.0, 2.0, 4.0, 4.0)

	// Un match de CAMPAGNE, sur la MEME carte, avec lui aussi une mort mesuree.
	tacMatchVariant(t, pdb, "mc", tacCarteA, base.Add(time.Hour), campagne[0])
	tacParticipant(t, pdb, "mc", tacXUIDMoi, 0, domain.OutcomeWin)
	tacKill(t, pdb, "mc", tacXUIDMoi, tacXUIDAdv, 1000, true)
	tacPos(t, pdb, "mc", tacXUIDMoi, 1000, 40.0, 40.0, 42.0, 42.0)

	// Un match de Campagne sur une carte QUE LA CAMPAGNE SEULE utilise : la carte
	// entiere doit disparaitre de la grille.
	tacMatchVariant(t, pdb, "mc2", "map_campagne", base.Add(2*time.Hour), campagne[0])
	tacParticipant(t, pdb, "mc2", tacXUIDMoi, 0, domain.OutcomeWin)

	repo := NewTacticalRepo(pdb)

	rows, err := repo.MapsPlayed(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 1 || rows[0].MapID != tacCarteA || rows[0].Matchs != 1 {
		t.Fatalf("cartes = %+v, want la seule carte A a 1 match (les deux matchs de Campagne masques)", rows)
	}

	pos, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m1"}; !egales(matchIDs(pos.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (le match de Campagne est hors univers)", matchIDs(pos.Univers.Matchs), want)
	}
	for _, p := range pos.Points {
		if p.MatchID == "mc" {
			t.Errorf("position d'un match de Campagne servie : %+v", p)
		}
	}

	// Contre-epreuve title-aware : le MEME corpus, lu comme Halo Infinite (titre sans
	// aucun match Campagne au registre), ne masque rien — la clause est un no-op, pas
	// un filtre en dur.
	pdb.TitleSlug = titlepkg.DefaultSlug
	rows, err = NewTacticalRepo(pdb).MapsPlayed(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MapsPlayed (infinite): %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("cartes (infinite) = %+v, want 2 : le masquage doit etre no-op pour un titre sans Campagne", rows)
	}
}

// TestTacticalRepo_AucuneDonnee_ZeroLigneZeroErreur : un joueur sans aucun match
// est l'etat NOMINAL d'un compte neuf, pas une panne.
func TestTacticalRepo_AucuneDonnee_ZeroLigneZeroErreur(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	repo := NewTacticalRepo(pdb)

	rows, err := repo.MapsPlayed(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil || len(rows) != 0 {
		t.Errorf("MapsPlayed = %+v / %v, want 0 ligne, nil", rows, err)
	}
	pos, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil || len(pos.Univers.Matchs) != 0 || len(pos.Points) != 0 {
		t.Errorf("KillPositions = %+v / %v, want vide, nil", pos, err)
	}
}

// TestTacticalRepo_EntreesVides_Refus : jamais de scan complet — un XUID vide est
// un refus sur les TROIS lectures, et une carte vide en est un sur la lecture
// SPATIALE, pas un balayage de shared.kill_positions.
//
// La carte vide n'est PLUS un refus sur KillEvents depuis le 2026-09-06 : la page
// Escouade lit le journal des morts d'une composition, qui n'a pas de carte. La
// borne qui reste est le joueur — cf. TestTacticalRepo_KillEvents_SansCarte.
func TestTacticalRepo_EntreesVides_Refus(t *testing.T) {
	repo := NewTacticalRepo(newTacticalTestPlayerDB(t))
	if _, err := repo.MapsPlayed(context.Background(), domain.TacticalQuery{}); err == nil {
		t.Error("MapsPlayed sans xuid : attendu un refus")
	}
	if _, err := repo.KillPositions(context.Background(), domain.TacticalQuery{MapID: tacCarteA}); err == nil {
		t.Error("KillPositions sans xuid : attendu un refus")
	}
	if _, err := repo.KillPositions(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi}); err == nil {
		t.Error("KillPositions sans carte : attendu un refus")
	}
	if _, err := repo.KillEvents(context.Background(), domain.TacticalQuery{}); err == nil {
		t.Error("KillEvents sans xuid : attendu un refus")
	}
}

// TestTacticalRepo_TablesAbsentes_Capability : un shared sans les tables du film
// (titre sans decodeur) rend ErrCapabilityNotSupported — 503 propre en bout de
// chaine, jamais un 500.
func TestTacticalRepo_TablesAbsentes_Capability(t *testing.T) {
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	// Le strict minimum pour que l'univers se lise : pas de kill_positions, pas de
	// match_kill_events — exactement la situation d'un titre sans decodeur de film.
	for _, ddl := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, map_id VARCHAR, map_name VARCHAR,
			map_name_fr VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			playlist_name VARCHAR, pair_name VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			team_id INTEGER, outcome INTEGER)`,
		`INSERT INTO match_registry VALUES ('m1', 'map_streets', 'a', 'a', NULL, NULL, NULL, NULL)`,
		`INSERT INTO match_participants VALUES ('m1', '` + tacXUIDMoi + `', 'moi', 0, 2)`,
	} {
		if _, err := sharedSQL.Exec(ddl); err != nil {
			t.Fatalf("ddl minimale: %v", err)
		}
	}
	shared := newTestDB(sharedSQL, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: tacXUIDMoi, TitleSlug: titlepkg.DefaultSlug}

	repo := NewTacticalRepo(pdb)
	if _, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("KillPositions sur schema sans film: err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.KillEvents(context.Background(), tacQuery(tacCarteA)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("KillEvents sur schema sans film: err = %v, want ErrCapabilityNotSupported", err)
	}
}

// TestTacticalRepo_KillEvents_SansCarte : la carte est OPTIONNELLE pour le journal
// des morts (ajout 2026-09-06, phase 3 du plan tactique). La page Escouade mesure
// l'echange d'une COMPOSITION, qui n'a pas de carte : elle demande tout
// l'historique du joueur et resserre son perimetre en Go.
//
// L'univers doit alors porter les matchs des DEUX cartes (m1, m2, m3) — et
// TOUJOURS PAS le match tiers m4, ou le joueur n'est pas participant : la borne
// qui saute est la carte, jamais le joueur.
func TestTacticalRepo_KillEvents_SansCarte(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(), tacQuery(""))
	if err != nil {
		t.Fatalf("KillEvents sans carte: %v", err)
	}
	if want := []string{"m1", "m2", "m3"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (les deux cartes, jamais le match tiers)",
			matchIDs(got.Univers.Matchs), want)
	}
	if len(got.Events) != 3 {
		t.Fatalf("evenements = %d, want 3 (2 sur m1, 1 sur m3) : %+v", len(got.Events), got.Events)
	}
	for _, e := range got.Events {
		if e.MatchID == "m4" {
			t.Errorf("evenement d'un match ou le joueur n'a pas joue : %+v", e)
		}
	}
}

// TestTacticalRepo_KillPositions_ExigeUneCarte : la lecture SPATIALE, elle, refuse
// toujours une carte vide. Sans carte elle balaierait `kill_positions` sur tout
// l'historique du joueur pour une grille de 0,5 m qui n'a de sens que carte par
// carte.
func TestTacticalRepo_KillPositions_ExigeUneCarte(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	if _, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery("")); err == nil {
		t.Fatal("KillPositions sans carte doit etre un REFUS, pas un balayage")
	}
}

// TestTacticalRepo_UniversDrapeauMesure : l'univers dit, MATCH PAR MATCH, si son
// journal des morts est LISIBLE (correction G2, revue du 2026-09-06).
//
// Dans la fixture, m1 porte deux kill-events publiables et m2 aucun : m2 est bien
// RETENU par le filtre (le joueur y a joue) et n'est PAS mesure. C'est ce drapeau
// qui empeche le service de compter un match jamais decode au denominateur
// « par match » — sans quoi l'intensite d'une carte varierait avec la couverture de
// film au lieu du jeu.
func TestTacticalRepo_UniversDrapeauMesure(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	mesure := map[string]bool{}
	for _, m := range got.Univers.Matchs {
		mesure[m.MatchID] = m.Mesure
	}
	if want := map[string]bool{"m1": true, "m2": false}; len(mesure) != 2 ||
		mesure["m1"] != want["m1"] || mesure["m2"] != want["m2"] {
		t.Fatalf("drapeaux = %v, want %v", mesure, want)
	}
}

// TestTacticalRepo_MesureExigePublishable : une passe NON PUBLIABLE ne rend pas un
// match mesure.
//
// C'est la meme exigence que les deux lectures (`e.publishable` des deux cotes) :
// sans elle, un match dont toutes les lignes sont ecartees compterait comme mesure
// sur l'onglet Tactique et pas sur la page Escouade, qui lit le meme journal filtre
// de la meme facon — deux denominateurs pour la meme notion.
func TestTacticalRepo_MesureExigePublishable(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatch(t, pdb, "np", tacCarteA, base)
	tacParticipant(t, pdb, "np", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "np", tacXUIDAdv, 1, domain.OutcomeLoss)
	tacKill(t, pdb, "np", tacXUIDMoi, tacXUIDAdv, 1000, false) // passe non publiable

	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if len(got.Univers.Matchs) != 1 {
		t.Fatalf("univers = %v, want [np]", matchIDs(got.Univers.Matchs))
	}
	if got.Univers.Matchs[0].Mesure {
		t.Error("un match dont la seule passe est non publiable ne doit pas compter comme mesure")
	}
}
