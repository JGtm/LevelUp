package killcollector

// capture_portee_test.go — LA PORTEE DU RADAR VOYAGE PAR LA CAPTURE (plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2b.1).
//
// Les trois lieux de naissance du collecteur appliquent la capture (garde-rail
// `archlint/no_collecteur_sans_capture_test.go`) ; ces tests pincent les deux maillons qui leur
// donnent la portee : `CaptureDepuisCatalogue` la charge (configuration REELLE du depot), et
// `AvecCapture` la pose sur le collecteur.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// TestCaptureDepuisCatalogue_PorteeDuDepot — la capture construite sur la configuration du depot
// rend 18 m en Arene et 24 m en BTB, nettoie la cle, et ne devine rien pour une variante absente.
func TestCaptureDepuisCatalogue_PorteeDuDepot(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	deps, err := CaptureDepuisCatalogue(racine, "halo_infinite",
		fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}})
	if err != nil {
		t.Fatalf("capture du depot : %v", err)
	}
	if deps.Portee == nil {
		t.Fatal("capture du depot sans portee du radar")
	}
	for _, c := range []struct {
		variante string
		metres   float64
		connue   bool
	}{
		{"Team Slayer:Arena", 18, true},
		{"CTF:Arena", 18, true},
		{" Team Slayer:Arena ", 18, true},
		{"BTB:Slayer", 24, true},
		{"BTB:CTF", 24, true},
		{"Super Fiesta:Fiesta", 0, false},
	} {
		if m, ok := deps.Portee(c.variante); m != c.metres || ok != c.connue {
			t.Errorf("%q : (%v, %v), attendu (%v, %v)", c.variante, m, ok, c.metres, c.connue)
		}
	}
}

// TestAvecCapture_PoseLaPorteeDuRadar — l'application de la capture pose la portee sur le
// collecteur : sans elle, toute ligne du placement s'ecrirait sans portee.
func TestAvecCapture_PoseLaPorteeDuRadar(t *testing.T) {
	deps := DepsCapture{
		MapNames: fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}},
		Bounds:   testMapQuantCatalog(),
		Portee:   func(v string) (float64, bool) { return 18, v == "Team Slayer:Arena" },
	}
	c := (&KillSourceCollector{}).AvecCapture(deps)
	r := c.placement.porteeDe("Team Slayer:Arena")
	if r == nil || *r != 18 {
		t.Fatalf("portee apres AvecCapture = %v, attendu 18 m", r)
	}
	if c.placement.porteeDe("Super Fiesta:Fiesta") != nil {
		t.Fatal("variante absente : une portee a ete devinee")
	}
}

// TestPorteeDuTitre_BestEffort — fichier absent ou illisible : aucune portee (journalise), jamais
// une erreur qui couperait les positions.
func TestPorteeDuTitre_BestEffort(t *testing.T) {
	if p := porteeDuTitre(t.TempDir(), "halo_infinite"); p != nil {
		t.Fatal("regulation.toml absent : attendu aucune portee")
	}
	racine := t.TempDir()
	chemin := mappings.RegulationPath(racine, "halo_infinite")
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatalf("dossier : %v", err)
	}
	if err := os.WriteFile(chemin, []byte("[meta]\ntitle_slug = \"halo_infinite\"\n"), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	if p := porteeDuTitre(racine, "halo_infinite"); p != nil {
		t.Fatal("regulation.toml invalide (schema_version absent) : attendu aucune portee")
	}
}
