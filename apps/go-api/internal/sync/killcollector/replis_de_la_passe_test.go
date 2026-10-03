package killcollector

// replis_de_la_passe_test.go — LE COMPTEUR DE LA PASSE DU COLLECTEUR (lot J8.7, 2026-09-27, decision 3).
//
// Maillons tenus : le compteur ne au debut de la passe atteint les sites (par le contexte, et par les
// identites du match qui le recopient), et sa publication ecrit un compteur expvar PAR NOM.
//
// MUTATION JOUEE (2026-09-27) : retirer
// `ids.replis.DeclencheN(decfilm.NomCoequipiersPartisConstanteNulle, len(out))` de `toDeathContextRows`
// fait rougir `TestLesFaitsDIsolementComptentLaConstanteNulle` (« comptee 0 fois pour 1 ligne »).

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// TestLeCompteurDeLaPasseVoyageParLeContexte : hors passe, aucun compteur (nil, qui ne compte rien).
func TestLeCompteurDeLaPasseVoyageParLeContexte(t *testing.T) {
	if replisDeLaPasse(context.Background()) != nil {
		t.Fatal("hors passe : un compteur apparait")
	}
	fb := decfilm.NouveauCompteur()
	if replisDeLaPasse(avecReplisDeLaPasse(context.Background(), fb)) != fb {
		t.Fatal("le compteur pose sur le contexte ne se relit pas")
	}
}

// TestUnNomInconnuCompteSonXuidVide : un nom qu aucune table ne resout rend un xuid vide, compte.
func TestUnNomInconnuCompteSonXuidVide(t *testing.T) {
	fb := decfilm.NouveauCompteur()
	ids := MatchIdentities{ParNom: map[string]string{"Connu": "1"}, replis: fb}
	if x, _ := ids.Resoudre("Connu"); x != "1" {
		t.Fatalf("nom connu : xuid %q", x)
	}
	if x, _ := ids.Resoudre("Inconnu"); x != "" {
		t.Fatalf("nom inconnu : xuid %q", x)
	}
	if got := fb.Compte(decfilm.NomXuidVidePourNomInconnu); got != 1 {
		t.Errorf("xuid vide compte %d fois, attendu 1", got)
	}
}

// TestLesFaitsDIsolementComptentLaConstanteNulle : chaque ligne de contexte ecrite porte la colonne
// `teammates_left` a la constante 0 — un declenchement par ligne.
func TestLesFaitsDIsolementComptentLaConstanteNulle(t *testing.T) {
	fb := decfilm.NouveauCompteur()
	mat := materiauDIsolement{registre: registreDeTest(positionsDUneVie(), 0), positions: positionsDUneVie()}
	ids := MatchIdentities{Equipes: map[string]int{"111": 0}, replis: fb}
	rows, _ := toDeathContextRows(mat, ids, []persist.KillEventInsert{{TimeMS: 1000, VictimXUID: "111"}})
	if len(rows) == 0 {
		t.Fatal("aucune ligne de contexte : le temoin ne mesure rien")
	}
	if got := fb.Compte(decfilm.NomCoequipiersPartisConstanteNulle); got != len(rows) {
		t.Errorf("constante nulle comptee %d fois pour %d ligne(s)", got, len(rows))
	}
}

// TestLaPassePublieUnCompteurParNom : la publication ecrit `killsource_<nom>` par repli declenche.
func TestLaPassePublieUnCompteurParNom(t *testing.T) {
	nom := prefixeReplisDeLaPasse + string(decfilm.NomIndiceEnCollisionJete)
	avant := observability.LoadCounter(nom)
	fb := decfilm.NouveauCompteur()
	fb.DeclencheN(decfilm.NomIndiceEnCollisionJete, 2)
	publierReplisDeLaPasse(context.Background(), "m", fb)
	if got := observability.LoadCounter(nom) - avant; got != 2 {
		t.Errorf("%s : +%d, attendu +2", nom, got)
	}
}

// TestLeRapportDuContexteDuPontEstVerseALaPasse — revue finale, decouverte 2 (2026-10-02) : le
// rapport du contexte de film que la passe ouvre (`lireLePontDuCollecteur`) — ici le fil des morts lu
// au dernier numero, faute de manifeste — n etait verse nulle part au collecteur. Il l est a la
// sortie de la passe, UNE fois, meme quand la lecture du pont echoue (mini-bobine sans manifeste ni carte).
// Mutations vues rouges : ne plus inscrire le contexte ; ne plus le verser ; le verser deux fois.
func TestLeRapportDuContexteDuPontEstVerseALaPasse(t *testing.T) {
	passe := decfilm.NouveauCompteur()
	ctx := avecReplisDeLaPasse(context.Background(), passe)
	// La mini-bobine versionnee du rejeu : trois chunks, SANS manifeste — le fil des morts se lit au
	// dernier numero.
	film, err := decfilm.LoadDir(filepath.Join("..", "..", "games", "halo_infinite", "film", "replay",
		"testdata", "minifilm_000d5950"), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _ = lireLePontDuCollecteur(ctx, film, decfilm.MapQuantEntry{}, MatchIdentities{}, "m")
	const publie = prefixeReplisDeLaPasse + "repli_temps_forts_dernier_numero"
	avant := observability.LoadCounter(publie)
	cloreLaPasse(context.Background(), ctx, "m")
	verserLesContextesDeLaPasse(ctx) // la sortie ne verse qu une fois, meme rejouee
	if got := decfilm.Texte(passe.Rapport()); got != "repli_temps_forts_dernier_numero=1" {
		t.Fatalf("compteur de la passe = %q, attendu le seul repli du fil des morts, une fois", got)
	}
	if got := observability.LoadCounter(publie) - avant; got != 1 {
		t.Fatalf("%s a bouge de %d, attendu 1 : le rapport du contexte n est pas publie", publie, got)
	}
}
