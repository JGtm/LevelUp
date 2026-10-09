package grammar

// debut_par_fermeture_test.go — LE DEBUT D UNE LISTE NON LOCALISEE SE CHOISIT SUR LA FERMETURE
// DU PAQUET (`debut_de_liste.go`, [debutParFermetureRangee]) : le premier candidat d ou la lecture
// ferme le paquet sans contredire l ecrivain, a defaut le premier d ou elle le ferme au bit pres —
// et le rang retenu est rendu.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// deuxDebuts ecrit un paquet ou la lecture depuis le bit 0 lit le DELTA 124 puis le DELTA 123
// (ferme au bit pres, l ordre de l ecrivain contredit), et depuis le bit 21 le DELTA 123 seul
// (ferme, ecrivable). Les deux lectures finissent sur le meme terminateur.
func deuxDebuts() (pay []byte, faux, juste int) {
	var bw bitWriter
	bw.deltaMasque13(124)
	juste = bw.n
	bw.deltaMasque13(123)
	bw.finDeVueB()
	bw.bit(0) // vue C vide
	return bw.buf, 0, juste
}

// TestDebutParFermeturePrefereLaLectureEcrivable : MUTATION — revenir au premier candidat ferme
// au bit pres rend 0 au lieu de 21, ROUGE.
func TestDebutParFermeturePrefereLaLectureEcrivable(t *testing.T) {
	pay, faux, juste := deuxDebuts()
	w := mondeDeCarte()
	if l := lectureDEssai(pay, w, cadreDeTete, faux); !l.FermeeAuBit || l.Fermee || l.Invariant != InvariantOrdreVueB {
		t.Fatalf("lecture depuis %d : %+v, attendu ferme au bit, ordre contredit", faux, l)
	}
	if l := lectureDEssai(pay, w, cadreDeTete, juste); !l.Fermee {
		t.Fatalf("lecture depuis %d : %+v, attendu fermee", juste, l)
	}
	if got, rang := debutParFermetureRangee(pay, []int{faux, juste}, w, cadreDeTete); rang != lecture.DebutParFermeture || got != juste {
		t.Errorf("debut %d (rang %d), attendu %d au premier rang : la lecture qui ferme sans contredire l ecrivain passe devant",
			got, rang, juste)
	}
}

// TestDebutParFermetureGardeLaTeteFermeeAuBit : sans candidat ecrivable, le debut reste le premier
// candidat ferme au bit pres (ses records sont lus, le paquet n est pas ferme), rendu au second
// rang ; sans candidat ferme du tout, la liste n est pas localisee.
func TestDebutParFermetureGardeLaTeteFermeeAuBit(t *testing.T) {
	pay, faux, _ := deuxDebuts()
	w := mondeDeCarte()
	if got, rang := debutParFermetureRangee(pay, []int{faux}, w, cadreDeTete); rang != lecture.DebutParFermetureAuBit || got != faux {
		t.Errorf("debut %d (rang %d), attendu %d au second rang", got, rang, faux)
	}
	if got, rang := debutParFermetureRangee(pay, []int{faux + 1}, w, cadreDeTete); rang != lecture.DebutNonLocalise || got != -1 {
		t.Errorf("aucun candidat ferme : debut %d (rang %d), attendu -1 non localise", got, rang)
	}
	if _, lie := w.ArchetypeForSlot(124); !lie {
		t.Error("les essais ont modifie le monde : il doit etre restaure apres chacun")
	}
}

// TestLeRepliDuDebutFermeAuBitEstCompte : seule une liste prise au SECOND rang compte le repli
// `repli_debut_de_liste_ferme_au_bit` ; le premier rang et la liste non localisee ne le comptent
// pas. MUTATION — retirer l appel `compterDebutDeListeParRepliFermeAuBit` : ROUGE.
func TestLeRepliDuDebutFermeAuBitEstCompte(t *testing.T) {
	pay, faux, juste := deuxDebuts()
	w := mondeDeCarte()
	cfg := cadreDeTete
	cfg.Obs = NouvelleObservation()
	debutParFermetureRangee(pay, []int{faux, juste}, w, cfg)
	debutParFermetureRangee(pay, []int{faux + 1}, w, cfg)
	if got := cfg.Obs.DebutsDeListeParRepliFermeAuBit; got != 0 {
		t.Fatalf("premier rang et liste non localisee : %d repli(s) compte(s), attendu 0", got)
	}
	debutParFermetureRangee(pay, []int{faux}, w, cfg)
	if got := cfg.Obs.DebutsDeListeParRepliFermeAuBit; got != 1 {
		t.Errorf("second rang : %d repli(s) compte(s), attendu 1", got)
	}
}
