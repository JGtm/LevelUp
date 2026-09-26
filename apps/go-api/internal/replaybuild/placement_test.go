// Package replaybuild — placement_test.go : le point de décision unique.
//
// Ce que ces tests protègent n'est pas une table de correspondance, c'est une
// RÈGLE DE CONCEPTION : le VPS web ne décode jamais un film. Elle doit échouer
// bruyamment (erreur explicite) plutôt que silencieusement (placement ignoré).
package replaybuild

import (
	"errors"
	"testing"
)

// TestDecidePlacement_TroisValeurs : chaque réglage mène à UN chemin, et un seul.
func TestDecidePlacement_TroisValeurs(t *testing.T) {
	dev := PlacementEnv{Production: false, WorkerConfigured: true}
	cases := map[string]struct {
		setting string
		env     PlacementEnv
		attend  Placement
		erreur  bool
	}{
		"local en dev":                 {"local", dev, PlacementLocal, false},
		"worker en dev":                {"worker", dev, PlacementWorker, false},
		"off":                          {"off", dev, PlacementOff, false},
		"défaut en dev = local":        {"", dev, PlacementLocal, false},
		"défaut en prod = worker":      {"", PlacementEnv{Production: true, WorkerConfigured: true}, PlacementWorker, false},
		"valeur inconnue":              {"n-importe-quoi", dev, PlacementOff, true},
		"worker sans ouvrier ouvert":   {"worker", PlacementEnv{WorkerConfigured: false}, PlacementOff, true},
		"défaut prod sans ouvrier":     {"", PlacementEnv{Production: true}, PlacementOff, true},
		"local en prod (REFUS)":        {"local", PlacementEnv{Production: true, WorkerConfigured: true}, PlacementOff, true},
		"local en prod sans ouvrier":   {"local", PlacementEnv{Production: true}, PlacementOff, true},
		"off en prod reste silencieux": {"off", PlacementEnv{Production: true}, PlacementOff, false},
	}
	for nom, c := range cases {
		got, err := DecidePlacement(c.setting, c.env)
		if got != c.attend {
			t.Errorf("%s : placement = %q, attendu %q", nom, got, c.attend)
		}
		if (err != nil) != c.erreur {
			t.Errorf("%s : erreur = %v, attendue = %v", nom, err, c.erreur)
		}
	}
}

// TestDecidePlacement_LocalEnProduction_MotifNomme : le refus est IDENTIFIABLE
// par l'appelant (l'API en fait un 400 avec le motif), pas une erreur anonyme.
func TestDecidePlacement_LocalEnProduction_MotifNomme(t *testing.T) {
	_, err := DecidePlacement("local", PlacementEnv{Production: true, WorkerConfigured: true})
	if !errors.Is(err, ErrLocalBuildInProduction) {
		t.Fatalf("erreur = %v, attendue ErrLocalBuildInProduction", err)
	}
}

// TestValidPlacementSetting : la validation du PATCH accepte le vide (« défaut de
// l'instance ») et rien d'inventé.
func TestValidPlacementSetting(t *testing.T) {
	for _, ok := range []string{"", "local", "worker", "off"} {
		if !ValidPlacementSetting(ok) {
			t.Errorf("%q devrait être accepté", ok)
		}
	}
	for _, ko := range []string{"Local", "distant", "vps", "true"} {
		if ValidPlacementSetting(ko) {
			t.Errorf("%q devrait être refusé", ko)
		}
	}
}
