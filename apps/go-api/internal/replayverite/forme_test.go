package replayverite

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// formeDuDocument : l'empreinte de forme du document publie, figee par le producteur
// (`film/replay/document_shape_test.go`). Lue comme TEXTE : le banc n'importe pas le producteur.
const formeDuDocument = "../games/halo_infinite/film/replay/testdata/document_shape.golden"

// TestForme_ChaqueCleLueExisteChezLeProducteur : une cle renommee cote producteur ferait lire une
// valeur zero au banc, qui jugerait alors un document vide « sans faux ». Ce test la rend bruyante.
func TestForme_ChaqueCleLueExisteChezLeProducteur(t *testing.T) {
	blob, err := os.ReadFile(formeDuDocument)
	if err != nil {
		t.Fatalf("empreinte de forme illisible : %v", err)
	}
	forme := string(blob)
	vues := map[reflect.Type]bool{}
	var parcourir func(reflect.Type)
	parcourir = func(ty reflect.Type) {
		for ty.Kind() == reflect.Pointer || ty.Kind() == reflect.Slice || ty.Kind() == reflect.Map {
			ty = ty.Elem()
		}
		if ty.Kind() != reflect.Struct || vues[ty] {
			return
		}
		vues[ty] = true
		for i := 0; i < ty.NumField(); i++ {
			f := ty.Field(i)
			nom, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if nom == "" {
				t.Errorf("%s.%s : champ sans etiquette json", ty.Name(), f.Name)
				continue
			}
			if !strings.Contains(forme, `json:"`+nom+`"`) && !strings.Contains(forme, `json:"`+nom+`,`) {
				t.Errorf("%s.%s : la cle %q n'existe plus dans %s", ty.Name(), f.Name, nom, formeDuDocument)
			}
			parcourir(f.Type)
		}
	}
	parcourir(reflect.TypeOf(Document{}))
	if len(vues) < 20 {
		t.Fatalf("seulement %d structures parcourues : le parcours est casse", len(vues))
	}
}
