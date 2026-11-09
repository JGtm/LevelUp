package parquetbackup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/diskfree"
)

// Erreurs de validation du dossier de sauvegarde ; leur identité fixe le code d'API que le serveur rend.
var (
	ErrDestNotAbsolute = errors.New("parquetbackup: le dossier de sauvegarde doit être un chemin absolu")
	ErrDestMissing     = errors.New("parquetbackup: le dossier de sauvegarde n'existe pas ou n'est pas un dossier")
	ErrDestNotWritable = errors.New("parquetbackup: le dossier de sauvegarde n'est pas accessible en écriture")
	ErrDestInsideData  = errors.New("parquetbackup: le dossier de sauvegarde est sous la racine de données")
)

// DestCheck : le verdict de ValidateDest quand le dossier est acceptable.
type DestCheck struct {
	// SameVolume : le dossier est sur le même volume que la racine de données. Accepté, mais une panne de
	// ce disque emporterait données et sauvegardes : l'interface l'affiche.
	SameVolume bool
}

// ValidateDest juge le dossier de sauvegarde choisi : chemin absolu, dossier existant, accessible en
// écriture (un fichier d'essai créé puis retiré), et PAS sous la racine de données (une sauvegarde dans
// ce qu'elle protège ne protège de rien). Le même volume que les données est accepté et signalé.
func ValidateDest(dest string, paths *title.PathResolver) (DestCheck, error) {
	if !filepath.IsAbs(dest) {
		return DestCheck{}, ErrDestNotAbsolute
	}
	if st, err := os.Stat(dest); err != nil || !st.IsDir() {
		return DestCheck{}, ErrDestMissing
	}
	if isUnder(paths.DataRoot(), dest) {
		return DestCheck{}, ErrDestInsideData
	}
	probe, err := os.CreateTemp(dest, ".levelup-write-test-*")
	if err != nil {
		return DestCheck{}, fmt.Errorf("%w : %w", ErrDestNotWritable, err)
	}
	name := probe.Name()
	_ = probe.Close()
	if err := os.Remove(name); err != nil {
		return DestCheck{}, fmt.Errorf("%w : %w", ErrDestNotWritable, err)
	}
	same, _ := diskfree.SameVolume(paths.DataRoot(), dest) // une mesure impossible = pas d'avertissement
	return DestCheck{SameVolume: same}, nil
}

// isUnder dit si dir est root ou un de ses descendants (liens symboliques résolus quand ils existent).
func isUnder(root, dir string) bool {
	root, dir = resolved(root), resolved(dir)
	if runtime.GOOS == "windows" {
		root, dir = strings.ToLower(root), strings.ToLower(dir)
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func resolved(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}
