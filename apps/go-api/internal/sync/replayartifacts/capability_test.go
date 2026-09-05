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

// racineIsoleeAvecManifestes rend un TempDir qui porte une COPIE des manifestes du titre
// (config/titles/{slug}/mappings/), pour les tests qui ecrivent dans leur racine — la
// cuisson y depose des artefacts, on ne peut donc pas leur passer la racine du depot.
//
// Depuis que Run passe la porte `film.replay_artifact` (capability.go), une racine vide
// n'est plus neutre : elle ferme la porte et l'etape ne fait rien. Ce helper rend a ces
// tests le titre qu'ils croyaient avoir.
func racineIsoleeAvecManifestes(t *testing.T, slug string) string {
	t.Helper()
	racine := t.TempDir()
	src := filepath.Join(racineDepot(t), "config", "titles", slug, "mappings")
	dst := filepath.Join(racine, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dst, err)
	}
	entrees, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", src, err)
	}
	copies := 0
	for _, e := range entrees {
		if e.IsDir() {
			continue
		}
		octets, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), octets, 0o644); err != nil {
			t.Fatalf("ecriture de %s: %v", e.Name(), err)
		}
		copies++
	}
	if copies == 0 {
		t.Fatalf("aucun manifeste copie depuis %s — le titre serait muet", src)
	}
	return racine
}
