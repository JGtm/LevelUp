package filmreplis

import (
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
)

func TestLaPartDeLHoteCompleteExactementLeRegistreDeLaBrique(t *testing.T) {
	for _, pb := range Verifier() {
		t.Errorf("registre des replis, part de l hote : %s", pb)
	}
}

// TestSitesEtAppelantsSontRelatifsALaRacineDuDepot : chaque chemin rendu nomme la racine de son
// registre, de sorte qu'un garde-rail l'ouvre depuis la racine du depot.
func TestSitesEtAppelantsSontRelatifsALaRacineDuDepot(t *testing.T) {
	vus := 0
	for _, r := range decfilm.Table() {
		for _, s := range Sites(r.Nom, r.Sites) {
			vus++
			if !strings.HasPrefix(s.Fichier, racineDeLaBrique) && !strings.HasPrefix(s.Fichier, racineDeLHote) {
				t.Errorf("%s : site %q hors des deux racines", r.Nom, s.Fichier)
			}
		}
		if r.HorsProduction != nil {
			for _, a := range Appelants(r.Nom, r.HorsProduction.Appelants) {
				if !strings.HasPrefix(a, racineDeLaBrique) && !strings.HasPrefix(a, racineDeLHote) {
					t.Errorf("%s : appelant %q hors des deux racines", r.Nom, a)
				}
			}
		}
	}
	if vus == 0 {
		t.Fatal("aucun site rendu : le registre est-il vide ?")
	}
}
