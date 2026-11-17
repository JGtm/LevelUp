package replayartifacts

// raster_ancien_test.go — LE FICHIER TACTIQUE D'UNE VERSION ANCIENNE (JSON brut) : jamais lu, retiré à l'écriture
// du fichier courant.

import (
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
)

// poserFichierTactiqueAncienBrut depose, a la place du fichier tactique ancien d'un match, un fichier JSON brut
// (ou le dossier qui l'empeche d'etre retire quand `bloque`).
func poserFichierTactiqueAncienBrut(t *testing.T, pr *titlePkg.PathResolver, matchID string, bloque bool) string {
	t.Helper()
	ancien := pr.TacticalRasterLegacyPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(ancien), 0o755); err != nil {
		t.Fatal(err)
	}
	if bloque {
		// Un dossier non vide ne se retire pas par os.Remove.
		if err := os.MkdirAll(filepath.Join(ancien, "contenu"), 0o755); err != nil {
			t.Fatal(err)
		}
		return ancien
	}
	if err := os.WriteFile(ancien, []byte(`{"schema_version":7}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return ancien
}

// L'ecriture du fichier tactique retire le fichier d'une version ancienne qui lui est voisin, et le dit ; sans
// voisin, elle ne dit rien ; un voisin qui ne se retire pas se dit aussi, sans faire echouer l'ecriture.
func TestEcrireSidecarRaster_RetireLeFichierAncien(t *testing.T) {
	s, err := ProjeterRasterTactique(t.Context(), ecrireFichier(t, "artefact.json", artefactImmobile), true)
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	t.Run("voisin_retire", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		ancien := poserFichierTactiqueAncienBrut(t, pr, s.MatchID, false)
		journal := capturerJournal(t)
		if err := EcrireSidecarRaster(t.Context(), pr.TacticalRasterPath(titlePkg.DefaultSlug, s.MatchID), s); err != nil {
			t.Fatalf("ecriture: %v", err)
		}
		if _, err := os.Stat(ancien); !os.IsNotExist(err) {
			t.Errorf("le fichier ancien devait etre retire (err = %v)", err)
		}
		if !aDit(t, journal, "INFO", "version ancienne retire") {
			t.Errorf("le retrait ne s'est pas dit : %s", journal)
		}
	})
	t.Run("sans_voisin_silence", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		journal := capturerJournal(t)
		if err := EcrireSidecarRaster(t.Context(), pr.TacticalRasterPath(titlePkg.DefaultSlug, s.MatchID), s); err != nil {
			t.Fatalf("ecriture: %v", err)
		}
		if aDit(t, journal, "INFO", "version ancienne") || aDit(t, journal, "WARN", "version ancienne") {
			t.Errorf("aucun fichier ancien : rien a dire, vu %s", journal)
		}
	})
	t.Run("retrait_en_echec_se_dit", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		ancien := poserFichierTactiqueAncienBrut(t, pr, s.MatchID, true)
		journal := capturerJournal(t)
		chemin := pr.TacticalRasterPath(titlePkg.DefaultSlug, s.MatchID)
		if err := EcrireSidecarRaster(t.Context(), chemin, s); err != nil {
			t.Fatalf("l'echec du retrait ne doit pas faire echouer l'ecriture : %v", err)
		}
		if !replayartifact.Exists(chemin) {
			t.Error("le fichier courant n'a pas ete ecrit")
		}
		if !aDit(t, journal, "WARN", "version ancienne non retire") {
			t.Errorf("l'echec du retrait n'a pas ete journalise (%s) : %s", ancien, journal)
		}
	})
}
