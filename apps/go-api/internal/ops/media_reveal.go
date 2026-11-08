// Package ops — media_reveal.go : ouverture de l'explorateur du système sur le fichier d'un média.
//
// Implémente port.MediaFileRevealer. Ce type ne décide rien : il ouvre ce qu'on lui désigne, ou le
// dossier existant le plus proche quand le fichier a disparu. Le lancement réel vit dans
// media_reveal_win.go (Windows) et media_reveal_other.go (ailleurs : erreur) ; le code portable
// ci-dessous les interprète, et les tests y injectent une doublure : aucun test n'ouvre un explorateur.
package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// OSFileRevealer ouvre l'explorateur du système.
type OSFileRevealer struct {
	// open lance l'explorateur sur target : en sélectionnant le fichier quand selectIt est vrai, en
	// ouvrant le dossier sinon.
	open func(ctx context.Context, target string, selectIt bool) error
}

// NewOSFileRevealer construit le revealer sur le lanceur de la plateforme.
func NewOSFileRevealer() *OSFileRevealer {
	return &OSFileRevealer{open: systemRevealOpen}
}

// Reveal ouvre l'explorateur sur absPath en le sélectionnant. Si absPath n'existe plus, ouvre le
// dossier existant le plus proche (sans sélection). Une erreur dit qu'aucun dossier n'existe ou que
// le lancement a échoué.
func (r *OSFileRevealer) Reveal(ctx context.Context, absPath string) error {
	clean := filepath.Clean(absPath)
	if absPath == "" || !filepath.IsAbs(clean) {
		return fmt.Errorf("media_reveal: chemin non absolu %q", absPath)
	}
	if _, err := os.Stat(clean); err == nil {
		return r.launch(ctx, clean, true)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("media_reveal: stat %s: %w", clean, err)
	}
	dir, ok := nearestExistingDir(clean)
	if !ok {
		return fmt.Errorf("media_reveal: aucun dossier existant au-dessus de %s", clean)
	}
	slog.InfoContext(ctx, "media_reveal: fichier absent, dossier le plus proche ouvert", "abs", clean, "dir", dir)
	return r.launch(ctx, dir, false)
}

func (r *OSFileRevealer) launch(ctx context.Context, target string, selectIt bool) error {
	if err := r.open(ctx, target, selectIt); err != nil {
		return fmt.Errorf("media_reveal: ouverture de %s: %w", target, err)
	}
	return nil
}

// nearestExistingDir remonte depuis p jusqu'au premier dossier existant. Faux quand la racine
// elle-même (lecteur) est absente.
func nearestExistingDir(p string) (string, bool) {
	for dir := filepath.Dir(p); ; dir = filepath.Dir(dir) {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir, true
		}
		if filepath.Dir(dir) == dir {
			return "", false
		}
	}
}
