// Package media — binpath.go : résolveur unique des binaires ffmpeg et ffprobe.
//
// Ordre de recherche : 1) le dossier désigné par LEVELUP_FFMPEG_DIR (si posé et s'il contient
// le binaire) ; 2) le dossier de l'exécutable courant ; 3) PATH. Résolu à chaque appel (quelques stat,
// un lancement de ffmpeg est rare) : un changement d'environnement est vu au lancement suivant.
//
// Seul fichier du dépôt (internal/ et cmd/) qui porte les noms « ffmpeg » / « ffprobe » : garde-rail
// archlint/media_binpath_single_resolver_test.go.
package media

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FFmpegBinary et FFprobeBinary : noms des binaires, sans suffixe de plateforme.
const (
	FFmpegBinary  = "ffmpeg"
	FFprobeBinary = "ffprobe"
)

// EnvFFmpegDir désigne le dossier qui contient ffmpeg et ffprobe (premier emplacement cherché).
const EnvFFmpegDir = "LEVELUP_FFMPEG_DIR"

// ErrBinaryNotFound : le binaire n'est dans aucun des trois emplacements.
var ErrBinaryNotFound = errors.New("binaire média introuvable")

// ResolveFFmpeg rend le chemin de ffmpeg selon l'ordre de recherche du paquet.
func ResolveFFmpeg() (string, error) { return locateBinary(FFmpegBinary, executableDir, exec.LookPath) }

// ResolveFFprobe rend le chemin de ffprobe selon l'ordre de recherche du paquet.
func ResolveFFprobe() (string, error) {
	return locateBinary(FFprobeBinary, executableDir, exec.LookPath)
}

// locateBinary applique l'ordre de recherche. Les sources de l'exécutable et du PATH sont
// injectées pour que le test fixe l'ordre ; la variable d'environnement est lue ici.
func locateBinary(name string, exeDir func() (string, error), lookPath func(string) (string, error)) (string, error) {
	file := name + binarySuffix()
	if dir := os.Getenv(EnvFFmpegDir); dir != "" {
		if p := filepath.Join(dir, file); isRegularFile(p) {
			return p, nil
		}
	}
	if dir, err := exeDir(); err != nil {
		slog.WarnContext(context.Background(), "media: dossier de l'exécutable indisponible, recherche ailleurs",
			"binaire", name, "err", err)
	} else if p := filepath.Join(dir, file); isRegularFile(p) {
		return p, nil
	}
	if p, err := lookPath(name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s : %w (%s, dossier de l'exécutable, PATH)", name, ErrBinaryNotFound, EnvFFmpegDir)
}

// failedCommand rend une commande qui ne se lance pas : Start rend l'erreur de résolution.
func failedCommand(err error) *exec.Cmd { return &exec.Cmd{Err: err} }

func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func binarySuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func isRegularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}
