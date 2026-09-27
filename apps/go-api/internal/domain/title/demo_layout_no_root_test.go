package title

// demo_layout_no_root_test.go — DemoLayout n'expose pas sa racine (revue adversariale du
// lot B5, code mort relevé au triage ; lot B-C6 du backlog 2026-09-26).
//
// Root() n'avait aucun appelant (règle n°7 : supprimé). Il n'a pas à revenir : une racine
// exposée invite à reconstruire un chemin de la démo à la main (filepath.Join(l.Root(), …)),
// ce que la disposition existe précisément pour empêcher (garde-rail
// archlint/no_demo_layout_translation_test.go). Chaque chemin a sa méthode nommée.

import (
	"reflect"
	"testing"
)

func TestDemoLayout_NExposePasSaRacine(t *testing.T) {
	if _, ok := reflect.TypeOf(DemoLayout{}).MethodByName("Root"); ok {
		t.Error("DemoLayout.Root() existe : la disposition ne doit pas exposer sa racine — " +
			"ajouter une méthode nommée pour le chemin voulu")
	}
}
