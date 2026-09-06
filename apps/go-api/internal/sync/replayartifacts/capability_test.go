package replayartifacts

// capability_test.go — LA PORTE DE PRODUCTION, EPROUVEE SUR LES VRAIS TOML DU DEPOT.
//
// Le gate se lit dans config/titles/{slug}/mappings/capabilities.toml : halo_infinite declare
// `film.replay_artifact`, halo_5 ne le declare pas. Un test qui fabriquerait ses propres TOML
// prouverait la mecanique du gate, pas la CONFIGURATION LIVREE — meme doctrine que
// usage_integration_test.go et postsync_test.go du collecteur de kills.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/replaybuild"
)

// racineDepot remonte du package au depot (apps/go-api/internal/sync/replayartifacts -> racine)
// pour que les gates par capability lisent les capabilities.toml LIVRES.
//
// Vit dans un fichier SANS tag de build : la porte de production s'exerce dans les tests
// ordinaires, la persistance du resume d'usage dans les tests d'integration, et les deux ont
// besoin du meme chemin.
func racineDepot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := filepath.Join(wd, "..", "..", "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "titles")); err != nil {
		t.Fatalf("racine du depot introuvable depuis %s: %v", wd, err)
	}
	return root
}

// TestTitreProduitDesArtefacts_SurLesTOMLLivres : halo_infinite oui, halo_5 non.
func TestTitreProduitDesArtefacts_SurLesTOMLLivres(t *testing.T) {
	root := racineDepot(t)
	cas := []struct {
		slug   string
		attend bool
	}{
		{"halo_infinite", true},
		{"halo_5", false},
		// LE TITRE SYNTHETIQUE : il declare `film.replay_artifact` = supported et ses cinq
		// derives `not_exposed`. La porte de PRODUCTION doit donc s'ouvrir pour lui — c'est
		// le chemin nominal d'un titre qui produit l'artefact sans qu'aucun derive soit
		// cable. Sans ce cas, la fixture ne prouvait rien (revue C-R1, constat C3, M3/M4).
		{"synthetic_title_b", true},
	}
	for _, c := range cas {
		got := titreProduitDesArtefacts(context.Background(), Deps{RepoRoot: root, TitleSlug: c.slug})
		if got != c.attend {
			t.Errorf("titre %s : porte = %v, attendu %v (cf. config/titles/%s/mappings/capabilities.toml)",
				c.slug, got, c.attend, c.slug)
		}
	}
}

// TestTitreProduitDesArtefacts_CapabilitiesIllisibles : une racine sans config/titles est un
// INCIDENT (le titre est mal configure), pas une configuration de titre — la porte se ferme
// et le WARN le dit.
func TestTitreProduitDesArtefacts_CapabilitiesIllisibles(t *testing.T) {
	buf := capturerJournal(t)
	if titreProduitDesArtefacts(context.Background(), Deps{RepoRoot: t.TempDir(), TitleSlug: "halo_infinite"}) {
		t.Fatal("capabilities illisibles : la porte s'est ouverte")
	}
	if !aDit(t, buf, "WARN", "capabilities illisibles") {
		t.Errorf("capabilities illisibles : aucun WARN.\nJournal :\n%s", buf.String())
	}
}

// TestRun_TitreSansCapability_NeSelectionneRien — LE CONSTAT D3 FERME.
//
// L'etape mettait en file (`enqueueAll`) et rattrapait le catalogue de cartes AVANT sa seule
// sonde de titre, laquelle n'etait qu'une degradation par absence de donnee. Un titre sans
// decodeur de film entrait donc dans la chaine, telechargeait des films et remplissait la
// file. La porte etant desormais en tete de Run, la BASE N'EST MEME PAS LUE : c'est la
// preuve la plus economique que rien n'a commence.
func TestRun_TitreSansCapability_NeSelectionneRien(t *testing.T) {
	// Le niveau du refus depend du memo par titre (cf. capability.go) : sans cette remise a
	// zero, l'ordre d'execution des tests du paquet deciderait si la ligne est INFO ou DEBUG.
	oublierRefusDits()
	buf := capturerJournal(t)
	lu := false
	enfile := false
	Run(context.Background(), Deps{
		RepoRoot:  racineDepot(t),
		TitleSlug: "halo_5",
		Placement: replaybuild.PlacementWorker,
		Enqueue: func(context.Context, string, string) error {
			enfile = true
			return nil
		},
		WithRead: func(context.Context, string, func(*sql.DB)) { lu = true },
	}, []string{"m1"})

	if lu {
		t.Error("halo_5 ne declare pas film.replay_artifact : la base a ete lue pour selectionner du travail")
	}
	if enfile {
		t.Error("halo_5 : un match a ete mis en file")
	}
	if !aDit(t, buf, "INFO", "titre sans la capability") {
		t.Errorf("le refus est muet.\nJournal :\n%s", buf.String())
	}
}

// TestTitreProduitDesArtefacts_RefusDitUneFoisEnINFOPuisEnDEBUG — observation C7 de la revue
// adversariale (2026-09-06) fermee.
//
// Le refus se disait en INFO a CHAQUE cycle : sur halo_5 — titre ACTIF qui n'aura jamais de
// decodeur de film — c'etait du bruit permanent sur un etat statique. Il se dit desormais une
// fois par titre et par process, puis en DEBUG. Le test fige les DEUX niveaux : sans le
// second volet, un retour au « toujours INFO » passerait inapercu.
func TestTitreProduitDesArtefacts_RefusDitUneFoisEnINFOPuisEnDEBUG(t *testing.T) {
	oublierRefusDits()
	root := racineDepot(t)
	d := Deps{RepoRoot: root, TitleSlug: "halo_5"}

	premier := capturerJournal(t)
	if titreProduitDesArtefacts(context.Background(), d) {
		t.Fatal("halo_5 : la porte s'est ouverte")
	}
	if !aDit(t, premier, "INFO", "titre sans la capability") {
		t.Errorf("premier refus : attendu une ligne INFO.\nJournal :\n%s", premier.String())
	}

	suivant := capturerJournal(t)
	if titreProduitDesArtefacts(context.Background(), d) {
		t.Fatal("halo_5 : la porte s'est ouverte au second appel")
	}
	if aDit(t, suivant, "INFO", "titre sans la capability") {
		t.Errorf("second refus du MEME titre : encore INFO — le memo par titre ne tient pas, "+
			"le journal se remplit d'une configuration qui ne changera jamais.\nJournal :\n%s",
			suivant.String())
	}
	if !aDit(t, suivant, "DEBUG", "titre sans la capability") {
		t.Errorf("second refus : la ligne DEBUG manque — le refus doit rester tracable, "+
			"seulement moins fort.\nJournal :\n%s", suivant.String())
	}

	// UN AUTRE titre parle a son tour : le memo est PAR TITRE, pas global.
	autre := capturerJournal(t)
	if titreProduitDesArtefacts(context.Background(), Deps{RepoRoot: root, TitleSlug: "titre_sans_config"}) {
		t.Fatal("titre inconnu : la porte s'est ouverte")
	}
	if !aDit(t, autre, "WARN", "capabilities illisibles") {
		t.Errorf("titre inconnu : attendu un WARN d'incident.\nJournal :\n%s", autre.String())
	}
}
