package ops

import (
	"path/filepath"
	"testing"
)

// TestSameVolume : un dossier et un chemin pas encore cree sous un autre dossier du meme disque
// sont sur le meme volume, qu'ils partagent ou non une racine.
func TestSameVolume(t *testing.T) {
	donnees, archive := t.TempDir(), t.TempDir() // deux racines sans lien, meme disque
	same, err := SameVolume(filepath.Join(archive, "pas", "encore"), donnees)
	if err != nil || !same {
		t.Fatalf("SameVolume = %v, %v ; attendu vrai (meme disque, racines distinctes)", same, err)
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
