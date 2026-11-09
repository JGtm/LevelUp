// Package ops — media_trash.go : envoi de l'original d'un média à la Corbeille du système.
//
// Implémente port.MediaFileTrasher. La règle qui gouverne tout le fichier : un échec ne devient
// JAMAIS une suppression définitive. Ce type ne supprime rien lui-même ; quand il ne peut pas envoyer
// un fichier à la Corbeille il le dit (erreur) ou dit qu'il n'y a pas de Corbeille (false, nil), et
// c'est l'appelant qui décide de la suite en connaissance de cause.
//
// Les deux appels système (type du lecteur, opération de coquille) sont des champs : le code
// portable ci-dessous les interprète, les tests y injectent des doublures et n'envoient JAMAIS un
// fichier réel à la Corbeille. Leurs implémentations réelles vivent dans media_trash_win.go
// (Windows) et media_trash_other.go (ailleurs : aucune Corbeille, false, nil).
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

// driveTypeFixed est la valeur de GetDriveType pour un disque local fixe (DRIVE_FIXED). Seul un
// lecteur fixe a une Corbeille : sur un support amovible ou réseau, l'opération de coquille avec
// l'option « annuler possible » supprimerait définitivement.
const driveTypeFixed = 3

// OSMediaFileTrasher envoie des originaux à la Corbeille du système.
type OSMediaFileTrasher struct {
	// capturesBase est relu à CHAQUE appel (réglage modifiable à chaud), comme pour
	// OSMediaFileRemover.
	capturesBase func() string
	// driveType rend le type du lecteur dont root est la racine (valeurs de GetDriveType).
	driveType func(root string) uint32
	// shellDelete envoie abs à la Corbeille. code est le retour de l'opération (0 = réussite),
	// aborted dit que le système a annulé tout ou partie de l'opération ; err est un échec de
	// l'appel lui-même.
	shellDelete func(abs string) (code int32, aborted bool, err error)
}

// NewOSMediaFileTrasher construit le trasher sur les appels système de la plateforme.
func NewOSMediaFileTrasher(capturesBase func() string) *OSMediaFileTrasher {
	return &OSMediaFileTrasher{
		capturesBase: capturesBase,
		driveType:    systemDriveType,
		shellDelete:  systemShellDelete,
	}
}

// TrashMediaFile envoie le fichier désigné par storedPath à la Corbeille. Rend true quand il y est
// parti et a bien quitté son dossier ; false, nil quand il n'y a rien à envoyer (chemin non
// résolvable, fichier absent, lecteur sans Corbeille) ; une erreur quand l'envoi a échoué, auquel cas
// le fichier est resté en place.
func (t *OSMediaFileTrasher) TrashMediaFile(ctx context.Context, storedPath string) (bool, error) {
	abs, ok := t.resolve(ctx, storedPath)
	if !ok {
		return false, nil
	}
	fi, err := os.Lstat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		slog.DebugContext(ctx, "media_trash: fichier déjà absent", "abs", abs)
		return false, nil
	case err != nil:
		return false, fmt.Errorf("media_trash: stat %s: %w", abs, err)
	case fi.IsDir():
		return false, fmt.Errorf("media_trash: %s est un dossier, refusé", abs)
	}
	root := filepath.VolumeName(abs) + `\`
	if kind := t.driveType(root); kind != driveTypeFixed {
		slog.InfoContext(ctx, "media_trash: lecteur sans Corbeille", "abs", abs, "drive_type", kind)
		return false, nil
	}
	return t.sendToTrash(ctx, abs)
}

// resolve rend le chemin absolu de storedPath, ou false (avec un log) quand il n'est pas
// résolvable : rien n'est envoyé à l'aveugle.
func (t *OSMediaFileTrasher) resolve(ctx context.Context, storedPath string) (string, bool) {
	if strings.TrimSpace(storedPath) == "" {
		return "", false
	}
	base := ""
	if t.capturesBase != nil {
		base = t.capturesBase()
	}
	abs := MediaPathStore{CapturesBase: base}.ToAbs(storedPath)
	if abs == "" || !filepath.IsAbs(abs) {
		slog.WarnContext(ctx, "media_trash: chemin non résolvable, fichier laissé en place",
			"stored", storedPath, "captures_base", base)
		return "", false
	}
	return filepath.Clean(abs), true
}

// sendToTrash lance l'opération de coquille et en interprète le résultat. Une réussite annoncée
// alors que le fichier est toujours là est une erreur : « envoyé à la Corbeille » ne se dit que
// d'un fichier qui a quitté son dossier.
func (t *OSMediaFileTrasher) sendToTrash(ctx context.Context, abs string) (bool, error) {
	code, aborted, err := t.shellDelete(abs)
	switch {
	case err != nil:
		return false, fmt.Errorf("media_trash: envoi de %s à la Corbeille: %w", abs, err)
	case aborted:
		return false, fmt.Errorf("media_trash: envoi de %s à la Corbeille annulé par le système", abs)
	case code != 0:
		return false, fmt.Errorf("media_trash: envoi de %s à la Corbeille refusé (code 0x%x)", abs, code)
	}
	if _, serr := os.Lstat(abs); serr == nil {
		return false, fmt.Errorf("media_trash: %s toujours présent après l'envoi à la Corbeille", abs)
	}
	slog.InfoContext(ctx, "media_trash: original envoyé à la Corbeille", "abs", abs)
	return true, nil
}
