package wire

// registry_monitoring_diskwatch_archive_test.go — l'archive des films est surveillée par la boucle
// disque existante quand elle vit hors du volume des données (lot L14), et seulement alors.

import (
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

func TestFilmArchiveWatch_HorsDuVolumeDesDonnees(t *testing.T) {
	racine := t.TempDir()
	archive := t.TempDir() // un autre dossier, hors de <racine>/data
	if filepath.Dir(archive) == racine {
		t.Skip("dossiers temporaires imbriqués")
	}
	w := filmArchiveWatchFor(titlePkg.NewPathResolverRoots(titlePkg.Roots{Resources: racine, FilmArchive: archive}))
	if w == nil {
		t.Fatal("archive hors des données : surveillance attendue")
	}
	disk := w.measure()
	if disk.Path != archive || disk.Status == domain.FreshnessStatusUnknown || disk.TotalBytes == 0 {
		t.Fatalf("mesure = %+v, attendu le volume de %s", disk, archive)
	}
	if !strings.Contains(w.msgCritical, "archive des films") || !strings.Contains(w.msgWarn, "archive des films") {
		t.Fatalf("messages d'alerte non distincts de ceux des données : %q / %q", w.msgCritical, w.msgWarn)
	}
	if got := filmArchiveWatchStatePath(filepath.Join("etat", "disk_watch.json")); got != filepath.Join("etat", "disk_watch_film_archive.json") {
		t.Fatalf("état anti-spam de l'archive = %q", got)
	}
}
