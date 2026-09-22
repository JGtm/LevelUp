package grammar

// etats_mouvement_porte_guard_test.go — LE GARDE-RAIL DE LA PORTE DE SPECULATION (lot 5.7.4).
//
// # CE QU IL EMPECHE DE REVENIR
//
// La porte des etats de mouvement, posee au lot 5.3.4, ne s etait pas inscrite dans
// `neutraliserCaptures` — la neutralisation que TOUS les chemins speculatifs de la marche
// declarent depuis le lot 2.2. Elle publiait donc les essais d alignement, dans un rapport
// MESURE de 14 a 152 pour un : sur `bfecd02b`, 7 463 lectures retenues par le balayage de
// production contre 400 apres correction ; sur `4f77afc1`, 112 592 contre 1 430.
//
// Une porte de publication qui oublie de s inscrire ici ne casse AUCUN test : elle publie plus,
// et « plus » ressemble a « mieux ». C est exactement le genre de regression qu un garde-rail
// doit attraper, et il en faut DEUX — un sur la composition (la neutralisation eteint-elle
// vraiment la porte ?) et un sur les APPELANTS (le localisateur la declare-t-il ?).

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// porteLocalisateurs : les trois fonctions du LOCALISATEUR de paquet. Ce sont les seuls chemins
// speculatifs du depot qui ne passent pas par une inference, donc les seuls qui doivent appeler
// la neutralisation eux-memes.
var porteLocalisateurs = []string{"marchLocateStrict", "marchLocateFallback", "marchLocate"}

// TestPorteEtatsMouvementNeutraliseeParLesTroisNeutralisations : les trois portes de
// neutralisation eteignent la publication des etats de mouvement, et la restaurent.
func TestPorteEtatsMouvementNeutraliseeParLesTroisNeutralisations(t *testing.T) {
	cas := []struct {
		nom        string
		neutralise func(*Observation) func()
	}{
		{"neutraliserCaptures", (*Observation).neutraliserCaptures},
		{"neutraliserCapturePosition", (*Observation).neutraliserCapturePosition},
		{"neutraliserEtatsDeMouvement", (*Observation).neutraliserEtatsDeMouvement},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var tire int
			obs := NouvelleObservation()
			obs.EtatMouvementHook = func(EtatMouvementComposant, uint32, []uint64) { tire++ }
			restaure := c.neutralise(obs)
			if obs.EtatMouvementHook != nil {
				t.Fatalf("%s : la porte des etats de mouvement est encore branchee — une "+
					"lecture speculative serait publiee comme une lecture retenue", c.nom)
			}
			restaure()
			if obs.EtatMouvementHook == nil {
				t.Fatalf("%s : la porte n a pas ete RESTAUREE — les records retenus qui "+
					"suivent ne publieraient plus rien", c.nom)
			}
			obs.EtatMouvementHook(EtatAccroupi, 0, nil)
			if tire != 1 {
				t.Fatalf("%s : la porte restauree n est pas celle d avant (%d tirs)", c.nom,
					tire)
			}
		})
	}
	// Le cas nil ne doit pas paniquer : la marche tourne sans observateur en production.
	var nul *Observation
	nul.neutraliserCaptures()()
	nul.neutraliserCapturePosition()()
	nul.neutraliserEtatsDeMouvement()()
}

// TestLocalisateursDeclarentLaPorte : les trois fonctions du localisateur appellent la
// neutralisation. C est un ratchet sur la SOURCE parce que le defaut qu il garde est une
// OMISSION — et une omission ne se voit pas a l execution : elle publie juste davantage.
func TestLocalisateursDeclarentLaPorte(t *testing.T) {
	const fichier = "object_deaths_march.go"
	data, err := os.ReadFile(fichier) //nolint:gosec // un fichier du paquet lui-meme
	if err != nil {
		t.Fatalf("lecture de %s : %v", fichier, err)
	}
	src := string(data)
	for _, nom := range porteLocalisateurs {
		re := regexp.MustCompile(`(?s)\nfunc ` + regexp.QuoteMeta(nom) + `\([^)]*\) int \{(.*?)\n\}`)
		m := re.FindStringSubmatch(src)
		if m == nil {
			t.Fatalf("%s : fonction %s introuvable — le ratchet ne garde plus rien (elle a "+
				"ete renommee ou deplacee : mettre a jour `porteLocalisateurs`)", fichier, nom)
		}
		if !strings.Contains(m[1], "neutraliserEtatsDeMouvement()()") {
			t.Errorf("%s : %s NE DECLARE PAS sa speculation.\n"+
				"Il essaie des offsets et traverse l entite pour de vrai a chacun : sans "+
				"`defer cfg.Obs.neutraliserEtatsDeMouvement()()`, chaque essai PUBLIE une "+
				"lecture d etat a une position de bit qui sera jetee. Mesure du lot 5.7.4 : "+
				"ce seul chemin pesait 2 359 a 6 358 lectures fantomes par film.", fichier, nom)
		}
	}
}
