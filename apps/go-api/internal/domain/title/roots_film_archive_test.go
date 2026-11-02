package title

// roots_film_archive_test.go — l'archive des films a sa propre racine (lot L14, ADR 0045 D-12) :
// elle se déplace seule, et rien de ce qui se refait du film ne la suit.

import (
	"path/filepath"
	"testing"
)

func TestFilmArchiveDir_ParDefautLaRacineDuCache(t *testing.T) {
	pr := NewPathResolverRoots(Roots{Resources: "/res", Data: "/donnees"})
	if got, want := pr.FilmArchiveDir(), filepath.Join("/donnees", "cache"); got != want {
		t.Fatalf("FilmArchiveDir par défaut = %q, attendu %q (le chemin d'avant le réglage)", got, want)
	}
	if pr.FilmArchiveDir() != pr.CacheRootDir() {
		t.Fatal("sans réglage, l'archive doit rester la racine du cache applicatif")
	}
	if pr.Roots().DefaultFilmArchiveDir() != pr.FilmArchiveDir() {
		t.Fatal("DefaultFilmArchiveDir doit désigner l'emplacement sans réglage")
	}
}

// TestFilmArchiveDir_HorsRacineAucunCheminDeFaitsNeLaSuit : une archive posée ailleurs ne déplace
// ni les faits décodés, ni les artefacts de rejeu, ni le reste du cache.
func TestFilmArchiveDir_HorsRacineAucunCheminDeFaitsNeLaSuit(t *testing.T) {
	archive := filepath.Join("/disque-externe", "films")
	pr := NewPathResolverRoots(Roots{Resources: "/res", Data: "/donnees", FilmArchive: archive})
	if pr.FilmArchiveDir() != archive {
		t.Fatalf("FilmArchiveDir = %q, attendu %q", pr.FilmArchiveDir(), archive)
	}
	cache := filepath.Join("/donnees", "cache")
	if pr.CacheRootDir() != cache {
		t.Errorf("CacheRootDir = %q, ne doit pas suivre l'archive (attendu %q)", pr.CacheRootDir(), cache)
	}
	for nom, chemin := range map[string]string{
		"faits d'un film":      pr.FilmFactsPath("halo_infinite", "000d5950"),
		"dossier des faits":    pr.FilmFactsDir("halo_infinite"),
		"artefact de rejeu":    pr.ReplayArtifactPath("halo_infinite", "000d5950"),
		"dossier des artefact": pr.ReplayArtifactsDir("halo_infinite"),
		"cache des jobs":       pr.JobsCachePath(),
	} {
		if rel, err := filepath.Rel(cache, chemin); err != nil || rel == ".." || len(rel) >= 2 && rel[:2] == ".." {
			t.Errorf("%s = %q : sorti de %q avec l'archive", nom, chemin, cache)
		}
	}
}

// TestFilmArchiveDir_PontDesRacinesDuProcessus : un outil qui ne reçoit qu'une racine nue
// (NewPathResolver) trouve l'archive réglée, comme le serveur.
func TestFilmArchiveDir_PontDesRacinesDuProcessus(t *testing.T) {
	t.Cleanup(ResetProcessRoots)
	archive := filepath.Join("/disque-externe", "films")
	InstallProcessRoots(Roots{Resources: "/res", Data: "/donnees", FilmArchive: archive})
	if got := NewPathResolver("/res").FilmArchiveDir(); got != archive {
		t.Fatalf("NewPathResolver(racine du processus).FilmArchiveDir = %q, attendu %q", got, archive)
	}
	ResetProcessRoots()
	if got, want := NewPathResolver("/res").FilmArchiveDir(), filepath.Join("/res", "data", "cache"); got != want {
		t.Fatalf("sans pont : %q, attendu %q", got, want)
	}
}
