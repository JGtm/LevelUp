package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// debut_non_prouve.go — UNE LECTURE QUE RIEN NE PROUVE NE MODIFIE PAS LE MONDE.
//
// Le second rang de [debutParFermetureRangee] (repli `repli_debut_de_liste_ferme_au_bit`) prend
// pour debut d une liste un candidat d ou le paquet ferme au bit pres en contredisant une regle du
// jeu : sa lecture n est pas prouvee. La marche qui en part LIT ses records et publie ce qu ils
// publient sans toucher au monde (traversees, etats, positions, tir continu, comptes), mais :
//
//	un record NEW n est pas LIE        (ni [World.BindFull] ni [World.BindSoft]) ; sa liaison de
//	                                   lecture est [lecture.LiaisonAucune] ;
//	un record DEL ne DELIE pas         ([World.Unbind]).
//
// Un NEW non prouve lie ecraserait l archetype d un slot pour la suite du chunk ; un DEL non prouve
// retirerait une entite vivante. Le refus d un NEW qui contredit une entite vivante est compte au
// second rang aussi ([Observation.NeufsContreUnVivant]) : [corpsDeRecordNeuf] juge la contradiction
// AVANT la marche non prouvee.
//
// CETTE MARCHE MODIFIE ENCORE LE MONDE PAR UN CHEMIN : la liaison par anticipation. Un en-tete
// DELTA d un slot non lie y passe par [rejetDeVue], qui lie le slot ([World.LierParRepliDAnticipation],
// liaison `Soft` a l archetype qu une image-cle ULTERIEURE declare) et lit son corps. Cette liaison
// n est pas un fait de la lecture non prouvee, mais elle reste posee apres elle : seuls le NEW et
// le DEL sont geles ici.
//
// LE MECANISME : le monde porte l annonce de la marche suivante. [debutParFermetureRangee] la pose
// apres ses essais ; la boucle de records de la vue B la prend a son entree
// ([Lecteur.entrerDansLaVueB]) et la retire dans tous les cas : elle ne vaut que pour la marche qui
// part de ce debut, dans ce payload.

// debutDeMarche designe le point de depart d une marche : le payload (par son premier octet) et le
// bit.
type debutDeMarche struct {
	pay *byte
	bit int
}

// marquerDebutNonProuve annonce que la prochaine marche de `pay` partira du bit `bit`, un debut
// que rien ne prouve. `pay` n est jamais vide : un debut y a ete trouve.
func (w *World) marquerDebutNonProuve(pay []byte, bit int) {
	w.debutNonProuve = &debutDeMarche{pay: &pay[0], bit: bit}
}

// prendreDebutNonProuve dit si la marche qui commence dans `buf` au bit `bit` est celle que
// [World.marquerDebutNonProuve] a annoncee, et retire l annonce.
func (w *World) prendreDebutNonProuve(buf []byte, bit int) bool {
	d := w.debutNonProuve
	w.debutNonProuve = nil
	return d != nil && len(buf) > 0 && d.pay == &buf[0] && d.bit == bit
}

// entrerDansLaVueB prepare le lecteur a une boucle de records de la vue B : sortie non atteinte,
// aucun en-tete rejete, et la marche reste non prouvee des que son debut l est.
//
// Le drapeau n est jamais remis a faux : une marche non prouvee le reste jusqu au bout de son
// lecteur, vues suivantes comprises (marche sans classes de vue, [DecodeFrameViewsCurseur]). Sous
// le profil de production (`ClassesDeVue`), chaque trame a son lecteur ([LecteurSur] dans la
// marche des trames) et une seule boucle de vue B : une remise a faux a l entree y serait
// equivalente.
func (b *Lecteur) entrerDansLaVueB(w *World, buf []byte) {
	b.sortieVueB, b.eidRejete = lecture.SortieNonAtteinte, 0
	if w.prendreDebutNonProuve(buf, b.BitPos()) {
		b.marcheNonProuvee = true
	}
}

// delier applique un record DEL au monde, sauf dans une marche non prouvee.
func (b *Lecteur) delier(w *World, slot uint32) {
	if !b.marcheNonProuvee {
		w.Unbind(slot)
	}
}
