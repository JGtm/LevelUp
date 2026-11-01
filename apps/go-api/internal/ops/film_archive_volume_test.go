package ops

import (
	"path/filepath"
	"testing"
)

func TestIsUnder(t *testing.T) {
	racine := filepath.Join(string(filepath.Separator), "donnees")
	for dir, attendu := range map[string]bool{
		filepath.Join(racine, "cache"): true,
		racine:                         true,
		filepath.Join(racine, "cache", "x", "..", "y"):              true,
		filepath.Join(string(filepath.Separator), "donnees-autres"): false,
		filepath.Join(string(filepath.Separator), "ailleurs"):       false,
	} {
		if got := IsUnder(dir, racine); got != attendu {
			t.Errorf("IsUnder(%q, %q) = %v, attendu %v", dir, racine, got, attendu)
		}
	}
}

// TestMeasureVolume_DossierPasEncoreCree : l'archive jamais écrite se mesure par son ancêtre
// existant, sans que le dossier soit créé.
func TestMeasureVolume_DossierPasEncoreCree(t *testing.T) {
	racine := t.TempDir()
	cible := filepath.Join(racine, "pas", "encore", "cree")
	vol, err := MeasureVolume(cible)
	if err != nil || vol.TotalBytes == 0 || vol.Path != racine {
		t.Fatalf("MeasureVolume = %+v, %v ; attendu le volume de %s", vol, err, racine)
	}
	if vol.Status == "" {
		t.Fatal("statut vide")
	}
}
