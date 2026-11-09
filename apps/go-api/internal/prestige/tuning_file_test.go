package prestige

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/testutil"
)

// Le fichier livré config/prestige/tuning.toml est valide, et ses noms de niveau (un texte par
// langue) sont ceux du repli DefaultTuning : LoadTuning rend DefaultTuning sur un fichier invalide,
// donc sans ce test une erreur du fichier passerait inaperçue.
func TestTuningFichierLivre_ValideEtAligneSurLeRepli(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "config", "prestige", "tuning.toml"))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	var got Tuning
	if err := toml.Unmarshal(raw, &got); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("validation : %v", err)
	}
	if want := DefaultTuning().Levels; !reflect.DeepEqual(got.Levels, want) {
		t.Errorf("levels du fichier = %v, repli = %v", got.Levels, want)
	}
}
