package parquetbackup

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// setNamePattern : un nom de dossier de jeu. La rotation et la lecture de l'état ne regardent que ceux-là.
var setNamePattern = regexp.MustCompile(`^` + SetPrefix + `\d{8}-\d{4}(-\d+)?$`)

// SetDirs rend les jeux COMPLETS (dossier `levelup-*` portant son manifeste) de dest, du plus récent au
// plus ancien. Un dossier d'un autre nom, un fichier ou un jeu sans manifeste n'y figure jamais.
func SetDirs(dest string) ([]string, error) {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || !setNamePattern.MatchString(e.Name()) {
			continue
		}
		if st, err := os.Stat(filepath.Join(dest, e.Name(), ManifestFile)); err != nil || st.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join(dest, n)
	}
	return out, nil
}

// rotate garde les keep jeux les plus récents de dest et supprime les autres, SAUF le dernier jeu COMPLET (non
// partiel) : si les keep plus récents sont tous partiels, le plus récent jeu complet est gardé en plus. Une suite
// de sauvegardes partielles (une base toujours occupée, une table trop longue) ne doit jamais évincer le dernier
// jeu qu'on sait restaurer en entier. N'efface que des dossiers dont le nom est celui d'un jeu
// (setNamePattern), direct enfants de dest : jamais autre chose. Rend le nombre de jeux supprimés. Un échec de suppression est journalisé et ne fait pas échouer la
// sauvegarde déjà écrite.
func rotate(ctx context.Context, dest string, keep int) int {
	if keep < 1 {
		return 0
	}
	sets, err := SetDirs(dest)
	if err != nil || len(sets) <= keep {
		return 0
	}
	spared := lastFullSetToSpare(sets, keep)
	removed := 0
	for _, dir := range sets[keep:] {
		if dir == spared {
			continue
		}
		if filepath.Dir(dir) != filepath.Clean(dest) || !setNamePattern.MatchString(filepath.Base(dir)) {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			slog.WarnContext(ctx, "backup parquet: rotation, jeu non supprimé", "set", dir, "err", err)
			continue
		}
		removed++
	}
	return removed
}

// removePartials supprime les dossiers `.partial-levelup-*` qu'un jeu interrompu a laissés.
func removePartials(ctx context.Context, dest string) {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), partialPrefix+SetPrefix) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dest, e.Name())); err != nil {
			slog.WarnContext(ctx, "backup parquet: jeu interrompu non supprimé", "dir", e.Name(), "err", err)
		}
	}
}

// lastFullSetToSpare rend le jeu complet le plus récent parmi ceux que la rotation supprimerait (sets[keep:]) quand
// aucun des keep jeux gardés n'est complet ; "" sinon (un jeu complet est déjà gardé, ou il n'en existe aucun).
// Un manifeste illisible compte comme partiel.
func lastFullSetToSpare(sets []string, keep int) string {
	for _, dir := range sets[:keep] {
		if isFullSet(dir) {
			return ""
		}
	}
	for _, dir := range sets[keep:] {
		if isFullSet(dir) {
			return dir
		}
	}
	return ""
}

// isFullSet : le jeu a un manifeste lisible qui ne le déclare pas partiel.
func isFullSet(dir string) bool {
	m, err := LoadManifest(dir)
	return err == nil && !m.Partial
}
