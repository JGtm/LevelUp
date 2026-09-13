package mappings

// loader_regulation_juggle_test.go — LA FENÊTRE DE JONGLAGE DU DRAPEAU, et son absence.
//
// Cette grandeur n'a pas de valeur par défaut : un titre qui ne la déclare pas ne publie
// AUCUNE prise nette. Ces tests verrouillent les deux moitiés de cette phrase — la valeur
// livrée pour Halo Infinite, et le silence pour un titre qui se tait.

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func regulationRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
}

// TestFlagJuggleWindow_HaloInfiniteLivree : la valeur MESURÉE du 2026-09-13 est bien celle
// du fichier livré. Le chiffre est dans le test parce que le perdre coûterait de refaire la
// mesure (404 délais sur 13 films).
func TestFlagJuggleWindow_HaloInfiniteLivree(t *testing.T) {
	t.Parallel()
	root := regulationRepoRoot(t)
	hi, err := LoadRegulationFromFile(filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_infinite regulation.toml: %v", err)
	}
	w, ok := hi.FlagJuggleWindow()
	if !ok {
		t.Fatal("halo_infinite : fenêtre de jonglage absente — les prises nettes ne seraient pas publiées")
	}
	if w != 1500*time.Millisecond {
		t.Errorf("fenêtre = %v, want 1.5s (coupure mesurée 2026-09-13)", w)
	}
}

// TestFlagJuggleWindow_TitreSansCleNePubliePas : Halo 5 ne déclare pas la règle, et c'est
// le comportement attendu — pas une lacune à combler par un défaut.
func TestFlagJuggleWindow_TitreSansCleNePubliePas(t *testing.T) {
	t.Parallel()
	root := regulationRepoRoot(t)
	h5, err := LoadRegulationFromFile(filepath.Join(root, "config", "titles", "halo_5", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_5 regulation.toml: %v", err)
	}
	if w, ok := h5.FlagJuggleWindow(); ok {
		t.Errorf("halo_5 : fenêtre = %v, ok=true — un titre sans la clé ne doit rien publier", w)
	}
}

// TestFlagJuggleWindow_NilSafe : un RegulationSet absent se lit comme un titre qui se tait.
func TestFlagJuggleWindow_NilSafe(t *testing.T) {
	t.Parallel()
	var s *RegulationSet
	if w, ok := s.FlagJuggleWindow(); ok || w != 0 {
		t.Errorf("nil = (%v, %v), want (0, false)", w, ok)
	}
}

// TestFlagJuggleWindow_NegativeRefusee : une valeur négative est une erreur de
// configuration, pas un silence. Pour ne pas publier la grandeur, on RETIRE la ligne.
func TestFlagJuggleWindow_NegativeRefusee(t *testing.T) {
	t.Parallel()
	raw := []byte("[meta]\ntitle_slug = \"t\"\nschema_version = 1\n\n[flag_grabs_net]\nflag_juggle_window_s = -1.0\n")
	if _, err := LoadRegulationFromBytes("test.toml", raw); err == nil {
		t.Fatal("fenêtre négative acceptée — attendu un refus au chargement")
	}
}

// TestFlagJuggleWindow_ZeroSeLitCommeAbsente : 0 n'est pas un interrupteur, c'est un
// silence. Le chargement passe, la lecture ne publie rien.
func TestFlagJuggleWindow_ZeroSeLitCommeAbsente(t *testing.T) {
	t.Parallel()
	raw := []byte("[meta]\ntitle_slug = \"t\"\nschema_version = 1\n\n[flag_grabs_net]\nflag_juggle_window_s = 0.0\n")
	s, err := LoadRegulationFromBytes("test.toml", raw)
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	if _, ok := s.FlagJuggleWindow(); ok {
		t.Error("fenêtre 0 publiée — attendu le même silence qu'une clé absente")
	}
}
