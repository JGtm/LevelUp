// media_index_base_test.go — couvre effectiveMediaBase (résolution de la base
// média avec fallback {root}/data/media). Package interne car la fonction est
// non exportée.
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestEffectiveMediaBase(t *testing.T) {
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

	t.Run("configuré vide → \"\" (comportement interne inchangé)", func(t *testing.T) {
		if got := effectiveMediaBase(ctx, pr, ""); got != "" {
			t.Errorf("got %q, want \"\"", got)
		}
	})

	t.Run("configuré valide → renvoyé tel quel", func(t *testing.T) {
		if got := effectiveMediaBase(ctx, pr, validBase); got != validBase {
			t.Errorf("got %q, want %q", got, validBase)
		}
	})

	t.Run("configuré invalide → fallback data/media", func(t *testing.T) {
		invalid := filepath.Join(root, "Z-inexistant", "Captures") // simule chemin Windows sur Linux
		if got := effectiveMediaBase(ctx, pr, invalid); got != mediaDir {
			t.Errorf("got %q, want %q (fallback)", got, mediaDir)
		}
	})

	t.Run("configuré invalide ET pas de data/media → \"\"", func(t *testing.T) {
		root2 := t.TempDir() // pas de data/media créé
		pr2 := titlePkg.NewPathResolver(root2)
		if got := effectiveMediaBase(ctx, pr2, filepath.Join(root2, "nope")); got != "" {
			t.Errorf("got %q, want \"\"", got)
		}
	})
}
