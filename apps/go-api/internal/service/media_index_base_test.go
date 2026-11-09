// media_index_base_test.go — couvre resolveMediaBase (résolution de la base
// média avec fallback {root}/data/media, et politique de marquage des médias
// introuvables qui en découle). Package interne car la fonction est non exportée.
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
)

func TestResolveMediaBase(t *testing.T) {
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	ctx := context.Background()

	// Fallback canonique {root}/data/media présent.
	mediaDir := filepath.Join(root, "data", "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Base configurée valide (existe sur disque).
	validBase := filepath.Join(root, "captures")
	if err := os.MkdirAll(validBase, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("configuré vide → \"\" (comportement interne inchangé), racine fiable", func(t *testing.T) {
		got, policy := resolveMediaBase(ctx, pr, "")
		if got != "" || policy != ops.MissingMark {
			t.Errorf("got (%q, %v), want (\"\", MissingMark)", got, policy)
		}
	})

	t.Run("configuré valide → renvoyé tel quel, racine fiable", func(t *testing.T) {
		got, policy := resolveMediaBase(ctx, pr, validBase)
		if got != validBase || policy != ops.MissingMark {
			t.Errorf("got (%q, %v), want (%q, MissingMark)", got, policy, validBase)
		}
	})

	t.Run("configuré invalide → fallback data/media, racine indisponible", func(t *testing.T) {
		invalid := filepath.Join(root, "Z-inexistant", "Captures") // simule chemin Windows sur Linux
		got, policy := resolveMediaBase(ctx, pr, invalid)
		if got != mediaDir || policy != ops.MissingRootUnavailable {
			t.Errorf("got (%q, %v), want (%q, MissingRootUnavailable)", got, policy, mediaDir)
		}
	})

	t.Run("configuré invalide ET pas de data/media → \"\", racine indisponible", func(t *testing.T) {
		root2 := t.TempDir() // pas de data/media créé
		pr2 := titlePkg.NewPathResolver(root2)
		got, policy := resolveMediaBase(ctx, pr2, filepath.Join(root2, "nope"))
		if got != "" || policy != ops.MissingRootUnavailable {
			t.Errorf("got (%q, %v), want (\"\", MissingRootUnavailable)", got, policy)
		}
	})
}
