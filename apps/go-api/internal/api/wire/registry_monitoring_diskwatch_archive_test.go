package wire

// registry_monitoring_diskwatch_archive_test.go — l'archive des films est surveillée par la boucle
// disque existante quand elle vit hors du volume des données (lot L14), et seulement alors.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

func TestFilmArchiveWatch_AbsenteSousLaRacineDesDonnees(t *testing.T) {
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: t.TempDir()}}
	if w := reg.filmArchiveWatch(); w != nil {
		t.Fatalf("archive par défaut (sous les données) : surveillance dédiée %+v, attendu nil (même volume)", w)
	}
}

// TestFilmArchiveWatch_MemeVolumeHorsDeLaRacineDesDonnees : une archive hors de la racine des
// donnees mais sur le meme disque n'ajoute AUCUNE surveillance (journaux et notifications doubles).
func TestFilmArchiveWatch_MemeVolumeHorsDeLaRacineDesDonnees(t *testing.T) {
	racine, archive := t.TempDir(), t.TempDir() // deux dossiers sans lien, meme disque
	w := filmArchiveWatchFor(titlePkg.NewPathResolverRoots(titlePkg.Roots{Resources: racine, FilmArchive: archive}))
	if w != nil {
		t.Fatalf("archive sur le meme volume que les donnees : surveillance dediee %+v, attendu nil", w)
	}
}

// TestFilmArchiveWatch_AutreVolume : la surveillance dediee existe quand le volume differe. Un
// second disque n'existe pas partout : le cas se joue avec un deuxieme volume si l'environnement
// en offre un (variable LEVELUP_TEST_OTHER_VOLUME), sinon il est saute et couvert par les tests
// de diskfree.SameVolume.
func TestFilmArchiveWatch_AutreVolume(t *testing.T) {
	autre := os.Getenv("LEVELUP_TEST_OTHER_VOLUME")
	if autre == "" {
		t.Skip("aucun second volume declare (LEVELUP_TEST_OTHER_VOLUME)")
	}
	archive := filepath.Join(autre, "films-test")
	w := filmArchiveWatchFor(titlePkg.NewPathResolverRoots(titlePkg.Roots{Resources: t.TempDir(), FilmArchive: archive}))
	if w == nil {
		t.Fatal("archive sur un autre volume : surveillance attendue")
	}
	disk := w.measure()
	if disk.Status == domain.FreshnessStatusUnknown || disk.TotalBytes == 0 {
		t.Fatalf("mesure = %+v", disk)
	}
	if !strings.Contains(w.msgCritical, "archive des films") || !strings.Contains(w.msgWarn, "archive des films") {
		t.Fatalf("messages d'alerte non distincts de ceux des donnees : %q / %q", w.msgCritical, w.msgWarn)
	}
}

func TestFilmArchiveWatchStatePath(t *testing.T) {
	if got := filmArchiveWatchStatePath(filepath.Join("etat", "disk_watch.json")); got != filepath.Join("etat", "disk_watch_film_archive.json") {
		t.Fatalf("etat anti-spam de l'archive = %q", got)
	}
}
