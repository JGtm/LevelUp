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

// SameVolume dit si `a` et `b` sont sur le MEME VOLUME (disque, partition, point de montage), pas
// seulement sous la meme racine : une archive en C:/Films et des donnees sous le profil utilisateur
// partagent un disque et n'ont qu'une surveillance. Un chemin pas encore cree se compare par son
// plus proche ancetre existant. Une erreur de mesure est rendue : l'appelant prefere alors
// surveiller deux fois que ne pas surveiller.
func SameVolume(a, b string) (bool, error) {
	return diskfree.SameVolume(nearestExisting(a), nearestExisting(b))
}
