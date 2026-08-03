// Package ops — media_remove.go : retrait disque des fichiers d'un média
// supprimé (v7.3 lot 2, item 3.1).
//
// Implémente port.MediaFileRemover. Vit dans ops parce que c'est déjà ici que
// résident MediaPathStore (résolution stocké↔absolu) et les autres accès disque
// média (indexation, miniatures, transcodage) — la règle des ≤ 2 copies interdit
// d'ouvrir une 3e zone de manipulation de chemins média.
package ops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// OSMediaFileRemover retire des fichiers média du disque local.
type OSMediaFileRemover struct {
	// capturesBase est résolu À CHAQUE APPEL : media_captures_base_dir est un
	// réglage modifiable à chaud (settings), une valeur capturée à la
	// construction du service pointerait vers l'ancien dossier après changement.
	// Même motif que capturesBaseDirFn dans BuildMediaScanHook.
	capturesBase func() string
}

// NewOSMediaFileRemover construit le remover. capturesBase peut être nil ou
// retourner "" : MediaPathStore bascule alors en pass-through et seuls les
// chemins déjà absolus (legacy) sont traitables.
func NewOSMediaFileRemover(capturesBase func() string) *OSMediaFileRemover {
	return &OSMediaFileRemover{capturesBase: capturesBase}
}

// RemoveMediaFiles retire les chemins stockés passés. Un fichier déjà absent
// n'est pas une erreur et n'est pas compté (idempotence : une suppression
// rejouée après un échec partiel doit converger).
//
// Le retrait continue après un échec unitaire pour maximiser ce qui est
// réellement effacé, et les erreurs sont agrégées : le service ne doit jamais
// annoncer « supprimé » si un fichier a résisté.
func (r *OSMediaFileRemover) RemoveMediaFiles(ctx context.Context, storedPaths []string) (int, error) {
	base := ""
	if r.capturesBase != nil {
		base = r.capturesBase()
	}
	store := MediaPathStore{CapturesBase: base}

	removed := 0
	var errs []error
	for _, stored := range storedPaths {
		if strings.TrimSpace(stored) == "" {
			continue
		}
		abs := store.ToAbs(stored)
		if abs == "" || !filepath.IsAbs(abs) {
			// Chemin non résolvable (base absente et stocké relatif) : ne rien
			// supprimer à l'aveugle, mais ne pas taire le cas.
			slog.WarnContext(ctx, "media_remove: chemin non résolvable, fichier laissé en place",
				"stored", stored, "captures_base", base)
			continue
		}
		n, err := removeMediaTarget(ctx, abs)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", stored, err))
			slog.ErrorContext(ctx, "media_remove: suppression disque échouée",
				"err", err, "stored", stored, "abs", abs)
			continue
		}
		removed += n
	}
	if len(errs) > 0 {
		return removed, errors.Join(errs...)
	}
	return removed, nil
}

// removeMediaTarget retire un fichier — ou, pour une playlist HLS, le RÉPERTOIRE
// de segments qui la contient.
//
// Sans le cas HLS, supprimer la seule master.m3u8 laisserait sur le disque tous
// les .m4s/.ts du clip : plusieurs centaines de Mo par vidéo, invisibles et
// jamais réclamés. Le répertoire n'est retiré que s'il est bien un dossier de
// transcodage (segment `hls` dans le chemin), garde-fou contre un file_path
// aberrant qui ferait supprimer un dossier de captures entier.
func removeMediaTarget(ctx context.Context, abs string) (int, error) {
	if isHLSPlaylist(abs) {
		dir := filepath.Dir(abs)
		if hasPathSegment(dir, "hls") {
			if err := os.RemoveAll(dir); err != nil {
				return 0, fmt.Errorf("remove hls dir: %w", err)
			}
			slog.InfoContext(ctx, "media_remove: répertoire HLS retiré", "dir", dir)
			return 1, nil
		}
		slog.WarnContext(ctx, "media_remove: playlist hors répertoire hls/, seul le fichier est retiré",
			"abs", abs)
	}
	if err := os.Remove(abs); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.DebugContext(ctx, "media_remove: fichier déjà absent", "abs", abs)
			return 0, nil
		}
		return 0, err
	}
	return 1, nil
}

// isHLSPlaylist reconnaît une playlist HLS par son extension.
func isHLSPlaylist(p string) bool {
	return strings.EqualFold(filepath.Ext(p), ".m3u8")
}

// hasPathSegment indique si le chemin contient le segment donné (comparaison
// insensible à la casse, séparateurs normalisés).
func hasPathSegment(p, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		if strings.EqualFold(part, segment) {
			return true
		}
	}
	return false
}
