package grammar

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// vue_a_fil_test.go — le fil des evenements de la vue A : un `PlayerGameEventSmall` (genre 82) dont
// le sac texte nomme un couple se range avec son type et ses destinataires ; les autres se lisent
// sans se ranger. Vecteurs ecrits d apres `FUN_14080add8` (`FUN_14080ae70`, `FUN_14080b034`,
// `FUN_14080ae28`).

// ecrireEvenementCourt ecrit un `PlayerGameEventSmall` : en-tete, `R(32)` de tete, `R(8)`, un sac
// principal vide, le sac texte (`participants` emplacements de sous-type 1, aucun si nil) et le
// masque des destinataires.
func (w *bitWriter) ecrireEvenementCourt(typ uint64, participants []uint64, masque uint64) {
	w.ecrireEnTeteDeMessage(GenreEvenementJoueurCourt)
	w.bits(typ, 32)
	w.bits(5, 8)
	w.bits(0, int(largeurCompteSacCourt))
	if participants == nil {
		w.bit(0) // porte « texte » fermee
	} else {
		w.bit(1)
		w.bits(0x1234, 32) // nom
		w.bits(uint64(len(participants)), 3)
		for _, p := range participants {
			w.bits(sacTexteParticipant, 3)
			w.bit(0) // FUN_1407f2058 : porte a 0, indice present
			w.bits(p, 5)
		}
	}
	w.bits(masque, ParticipantsDuFil)
}

// TestLeFilDesEvenementsSeRangeAvecSesDestinataires : le message a couple se range (son debut, son
// type, son couple dans l ordre ecrit, son masque) ; celui sans sac texte se lit sans se ranger ; le
// premier bit du masque designe le participant 0.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer le rangement du genre 82 de [lireLaVueA] ; lire le
// masque a l envers dans [DestineA] ; intervertir tueur et victime dans
// [coupleDuSacTexte].
func TestLeFilDesEvenementsSeRangeAvecSesDestinataires(t *testing.T) {
	const masque = 1<<(ParticipantsDuFil-1-1) | 1<<(ParticipantsDuFil-1-3)
	var w bitWriter
	w.bit(1)
	w.ecrireEvenementCourt(70, []uint64{4, 11}, masque)
	second := w.n
	w.ecrireEvenementCourt(38, nil, 1<<(ParticipantsDuFil-1))
	w.bit(0) // terminateur
	fin := w.n
	w.bits(0, 16)
	a := lireSous(w.buf, grammaireRecente())
	attendu := lecture.EvenementDeFil{Debut: 1, Type: 70, Destinataires: masque, Tueur: 4, Victime: 11}
	if !a.Porte || a.Fin != fin || len(a.Genres) != 2 || len(a.Fil) != 1 || a.Fil[0] != attendu {
		t.Fatalf("vue A %+v (second message au bit %d), attendu portee jusqu au bit %d avec le fil %+v",
			a, second, fin, attendu)
	}
	for i, d := range map[int]bool{0: false, 1: true, 2: false, 3: true, 31: false} {
		if DestineA(a.Fil[0], i) != d {
			t.Errorf("Destine(%d) = %v, attendu %v", i, !d, d)
		}
	}
	var p lecture.Paquet
	p.Payload = w.buf
	rangerLaVueA(&p, &a)
	if len(p.VueA.Fil) != 1 || p.VueA.Fil[0] != attendu {
		t.Errorf("fil range %+v, attendu %+v", p.VueA.Fil, attendu)
	}
}

// TestLeCanalDesKillsDitSiLaTrameEstLueEntiere : le canal rend chaque message du fil d une trame avec
// l etat de sa vue A — `Complet` seulement quand la lecture a atteint son terminateur.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : deriver `Complet` d autre chose que `VueTerminee` dans
// [canalDesKills.Tete] (constant, ou « liste annoncee »).
func TestLeCanalDesKillsDitSiLaTrameEstLueEntiere(t *testing.T) {
	e := lecture.EvenementDeFil{Type: 70, Destinataires: 1 << (ParticipantsDuFil - 1), Tueur: 1, Victime: 2}
	for etat, complet := range map[lecture.EtatDeVue]bool{lecture.VueTerminee: true, lecture.VueArretee: false} {
		c := nouveauCanalDesKills(nil)
		var p lecture.Paquet
		p.Debut = lecture.DebutParVueA
		p.VueA.Etat, p.VueA.Genres, p.VueA.Fil = etat, []uint8{GenreEvenementJoueurCourt}, []lecture.EvenementDeFil{e}
		c.Tete(&p)
		if len(c.fil) != 1 || c.fil[0].Complet != complet || c.fil[0].Evenement != e || !c.fil[0].Destine(0) {
			t.Errorf("etat %d : fil %+v, attendu un message complet=%v", etat, c.fil, complet)
		}
	}
}
