package grammar

// debut_de_liste_haute_frequence_test.go — la CHAINE des records NEW de tete derriere la signature
// haute frequence ([localiserLaListe], troisieme etage), sur des paquets reels de
// `minibobine_000d5950` dont le premier delta du slot 123 est recopie sur un autre objet de
// l archetype `high-frequency` ([recopieSurUnAutreSlot]).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// TestLaChaineDeTeteSuitLaSignatureHauteFrequence : sous la table anticipee du film (celle de la
// marche des trames), des paquets reels ouvrent leur liste par des records NEW dont la chaine finit
// au bit pres sur la signature du slot 123 : la cuisson rend leur premier NEW
// ([lecture.DebutParChaine]). Le premier delta recopie sur un autre slot de l archetype (meme
// ecrivain, meme generation), le slot 123 ne porte plus de signature ; quand aucun candidat de tete
// ne ferme le paquet, la cuisson atteint la signature haute frequence, et la MEME chaine finit sur
// elle : la cuisson rend le meme NEW de tete, par la chaine. MUTATION : `return debut,
// lecture.DebutParSignature` juste apres l etage haute frequence de [localiserLaListe] (la chaine
// n est plus essayee derriere lui) : ROUGE.
func TestLaChaineDeTeteSuitLaSignatureHauteFrequence(t *testing.T) {
	memeGeneration := func(rec FrameRecord, x uint32) uint32 { return rec.ID&^0x3fffffff | x }
	vecteurs := 0
	parcourirLaBobine(t, bobineMarcheDir(), true, func(p paquetDeLocalisation, cfg FrameConfig) {
		s := marchLocateStrict(p.pay, p.w, cfg)
		if s < 0 || !generationDuMonde(p, cfg, s) {
			return
		}
		tete, comment := localiserLaListe(p.pay, p.w, cfg)
		if comment != lecture.DebutParChaine {
			return
		}
		pay, x, restaurer, ok := recopieSurUnAutreSlot(p, cfg, s, memeGeneration)
		if !ok {
			return
		}
		defer restaurer()
		if hf, _ := LocaliserBoucleDeRecords(pay, p.w, cfg, SignatureHauteFrequence); hf != s {
			return // une signature haute frequence anterieure (D-LS-2) : le vecteur ne dit rien
		}
		candidats := candidatsDeTete(pay, len(pay)*8, p.w)
		if _, rang := debutParFermetureRangee(pay, candidats, p.w, cfg); rang != lecture.DebutNonLocalise {
			return // la fermeture par NEW de tete precede l etage haute frequence
		}
		vecteurs++
		if d, c := localiserLaListe(pay, p.w, cfg); d != tete || c != lecture.DebutParChaine {
			t.Errorf("slot %d : cuisson (%d, %v), attendu (%d, chaine) — la signature haute frequence est a %d",
				x, d, c, tete, s)
		}
	})
	if vecteurs == 0 {
		t.Fatal("aucun paquet a chaine de tete derriere une signature haute frequence : le test ne garde rien")
	}
	t.Logf("%d vecteur(s)", vecteurs)
}
