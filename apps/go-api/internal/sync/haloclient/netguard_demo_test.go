// netguard_demo_test.go — le mode démo ne doit émettre AUCUN appel à l'API Halo.
//
// POURQUOI CE TEST. C'est le chemin qui a provoqué l'incident : une démo lancée
// sur un poste porteur de vrais tokens appelait `careerranks` pour les xuid
// FACTICES de la fixture (0000000000000000/1/2) — HTTP 400, 4 tentatives,
// ~12 s par appel — jusqu'à affamer le rendu des pages du harnais visuel.
// La preuve par le journal d'un run ne suffit pas : sur une machine sans token
// utilisable, le chemin est déjà coupé en amont (« reason=no_auth_tokens ») et
// le test d'intégration passerait même si le garde avait disparu. On vérifie
// donc ICI, sur la couche qui émet.
package haloclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/netguard"
)

func TestDemoMode_NoOutboundHaloCall(t *testing.T) {
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	// Tokens NON vides : on veut prouver que le garde coupe même quand l'auth
	// est parfaitement valide — c'est exactement la situation de l'incident.
	c := NewHaloAPIClient("spartan-token-factice", "clearance-factice", 10)

	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{"GetCareerProgress", func(ctx context.Context) error {
			_, err := c.GetCareerProgress(ctx, "0000000000000000")
			return err
		}},
		{"GetSpartanCustomization", func(ctx context.Context) error {
			_, err := c.GetSpartanCustomization(ctx, "0000000000000000")
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			err := tc.call(context.Background())
			elapsed := time.Since(start)

			if !errors.Is(err, netguard.ErrOffline) {
				t.Fatalf("attendu netguard.ErrOffline, obtenu %v", err)
			}
			// Un retour immédiat prouve qu'aucune requête (ni retry) n'est partie :
			// le chemin réseau coûtait ~12 s avec ses 4 tentatives.
			if elapsed > 2*time.Second {
				t.Errorf("retour en %s — trop lent pour un court-circuit, une requête est probablement partie", elapsed)
			}
		})
	}
}

// TestNetguardDisabled_LeavesCallPathIntact : le garde ne doit rien changer hors
// mode démo. Sans cette vérification, un `SetOffline` mal remis à zéro passerait
// inaperçu et couperait la PRODUCTION.
func TestNetguardDisabled_LeavesCallPathIntact(t *testing.T) {
	netguard.SetOffline(false)
	if netguard.Offline() {
		t.Fatal("netguard actif alors qu'il a été désarmé")
	}
	if err := netguard.Check(context.Background(), "test"); err != nil {
		t.Fatalf("hors mode démo, Check doit retourner nil — obtenu %v", err)
	}
}
