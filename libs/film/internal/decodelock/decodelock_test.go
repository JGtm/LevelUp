package decodelock

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPrendreDecritLeDetenteurEtLeRetireALaFin : pendant le test la description nomme l'outil et
// le match ; un second preneur est refuse en la citant ; a la fin du sous-test tout est rendu.
func TestPrendreDecritLeDetenteurEtLeRetireALaFin(t *testing.T) {
	dir := t.TempDir()
	t.Run("tenu", func(t *testing.T) {
		Prendre(t, dir, "outil-a", "abc12345")
		b, err := os.ReadFile(filepath.Join(dir, soloHolderFileName))
		if err != nil {
			t.Fatal(err)
		}
		var h soloHolder
		if err := json.Unmarshal(b, &h); err != nil || h.Tool != "outil-a" || h.MatchID != "abc12345" {
			t.Fatalf("description %s (%v)", b, err)
		}
		_, err = tenter(dir, "outil-b", "x")
		if !errors.Is(err, errOccupe) || !strings.Contains(err.Error(), "outil-a") {
			t.Fatalf("second preneur : %v, attendu un refus qui nomme outil-a", err)
		}
	})
	if _, err := os.Stat(filepath.Join(dir, soloHolderFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("description encore presente apres le test : %v", err)
	}
	l, err := tenter(dir, "outil-c", "y")
	if err != nil {
		t.Fatalf("verrou non rendu a la fin du test : %v", err)
	}
	rendre(t, l)
}
