package ops

// film_archive_volume.go — l'espace du volume qui porte l'archive des films (lot L14).
//
// L'archive peut vivre sur un autre disque que les données (réglage `film_archive_dir`) : la
// surveillance disque existante (wire.RunDiskWatchLoop) la mesure alors avec les MÊMES seuils
// (80 / 90 %, [EvaluateDiskStatus]) et la page des réglages affiche la même mesure. Une seule
// fonction, donc un seul verdict.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/platform/diskfree"
)

// VolumeUsage : l'espace d'un volume et son statut au sens de [EvaluateDiskStatus].
type VolumeUsage struct {
	Path       string // dossier mesuré (l'ancêtre existant le plus proche du dossier demandé)
	FreeBytes  uint64
	TotalBytes uint64
	Status     string
}

// MeasureVolume mesure le volume qui porte `dir`. Un dossier pas encore créé (archive jamais
// écrite) se mesure par son plus proche ancêtre existant : le volume est le même, et créer le
// dossier pour le mesurer serait un effet de bord.
func MeasureVolume(dir string) (VolumeUsage, error) {
	probe := nearestExisting(dir)
	free, total, err := diskfree.Free(probe)
	if err != nil {
		return VolumeUsage{Path: probe}, fmt.Errorf("mesure du volume de %s : %w", probe, err)
	}
	return VolumeUsage{Path: probe, FreeBytes: free, TotalBytes: total, Status: EvaluateDiskStatus(free, total)}, nil
}

// nearestExisting remonte de `dir` jusqu'au premier dossier qui existe.
func nearestExisting(dir string) string {
	cur := filepath.Clean(dir)
	for {
		if _, err := os.Stat(cur); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return cur
		}
		cur = parent
	}
}

// IsUnder dit si `dir` est `root` ou se trouve dessous (comparaison lexicale après nettoyage,
// insensible à la casse sous Windows). Sert à décider si l'archive partage le volume des données :
// sous la racine de données, c'est le même volume et la surveillance déjà en place suffit.
func IsUnder(dir, root string) bool {
	rel, err := filepath.Rel(strings.ToLower(filepath.Clean(root)), strings.ToLower(filepath.Clean(dir)))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
