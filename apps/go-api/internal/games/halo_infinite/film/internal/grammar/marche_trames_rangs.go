package grammar

// marche_trames_rangs.go — LA MARCHE D UNE TRAME DELTA PAR RANGS : le frame-processeur
// `FUN_142987460` porte sur un paquet (lot 5.14), une seule implantation.
//
// [decodeFrameParRangs] en rend les records, les rangs lus et le curseur ; la marche des trames
// ([FilmContext.Trames]) en range tout dans la structure de lecture (ADR 0037) ; la carte de
// fermeture detaillee en publie le detail. Les trois lisent la MEME marche : aucun d eux ne
// recopie le pilotage des rangs.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// lectureDeTrame est ce que la marche d UNE trame delta par rangs a lu.
type lectureDeTrame struct {
	// enTete : la marche est partie de la tete du paquet — bit de configuration, puis vue A.
	enTete bool
	// vueA : la vue A, quand enTete ; vueARecue : elle a ete lue avant la marche et passee par son
	// depart ([departDeTrame]) — la marche ne l a ni relue ni ne la re-range. `[debutVueA, finVueA)`
	// est ce que la marche en a traverse.
	vueA               FluxVueA
	vueARecue          bool
	debutVueA, finVueA int
	// debutVueB / finVueB : les bornes de la vue B, -1 quand elle n est pas atteinte ; sortieVueB
	// et eidRejete : comment elle s est arretee ([Lecteur.sortirDeLaVueB]).
	debutVueB, finVueB int
	sortieVueB         lecture.SortieVueB
	eidRejete          uint32
	// recs : les records de la vue B, dans l ordre du flux.
	recs []FrameRecord
	// vueCAtteinte, debutVueC et fluxC : la vue C, lue quand la vue B a clos sa liste.
	vueCAtteinte bool
	debutVueC    int
	fluxC        FluxVueC
	// verdict : le verdict de la vue C, tel qu il est publie au crochet
	// ([Observation.VueControleHook]) — vide quand la vue C n est pas atteinte.
	verdict LectureVueC
	// rangs : les vues lues jusqu a leur terminateur, comptees depuis le point de depart.
	rangs int
	// curseur : la position d arret de la marche ; deborde : la lecture a depasse la fin du
	// payload ([source.Bits.Deborde]).
	curseur int
	deborde bool
}

// departDeTrame est le point de depart de la marche d une trame : le bit (`skipLeadBits`) et, quand
// l appelant l a deja lue, la vue A du paquet ([rangerLaTete]) — nil : la marche la lit elle-meme
// si elle part de la tete.
type departDeTrame struct {
	bit  int
	vueA *FluxVueA
}

// lireTrameParRangs marche UNE trame delta sous la grammaire de chaque classe de vue
// (`GrammaireBalayage.ClassesDeVue`) — rang 0 la vue A, rang 1 la vue B, rang 2 la vue C — et
// range ce qu elle a lu dans `l`. Le verdict de la vue C est publie au crochet, une fois par
// paquet, atteinte ou non.
//
// LA TETE DU PAQUET : quand la marche part de la tete (`d.bit == cfg.PacketPreambleBits`), elle
// saute le seul bit de configuration et prend la vue A — celle de son depart, sinon la lecture
// unique ([lireLaVueA]) sans grammaire de film : sa tete seule decide de la marche. Elle ne
// traverse qu une vue A VIDE : une vue A qui porte un message arrete la marche apres sa tete, la ou
// elle s arretait avant que la vue A soit lue jusqu au bout ; la vue B d un paquet a evenements part
// de la fin de sa vue A lue ou d un debut LOCALISE ([debutDeLaVueBDeCuisson]). Depuis l un ou l autre,
// la vue A est derriere le point de depart et la marche commence au rang 1 (cf.
// [decodeFrameParRangs]).
func lireTrameParRangs(br *Lecteur, buf []byte, w *World, cfg FrameConfig, d departDeTrame,
	l *lectureDeTrame) {
	frameLen := len(buf) * 8
	*l = lectureDeTrame{debutVueB: -1, finVueB: -1}
	if d.bit == cfg.PacketPreambleBits && cfg.PacketPreambleBits >= 1 {
		l.enTete = true
		br.Skip(cfg.PacketPreambleBits - 1) // le bit de configuration du frame-processeur
		l.debutVueA = br.BitPos()
		if d.vueA != nil && d.vueA.Debut == l.debutVueA {
			l.vueA, l.vueARecue = *d.vueA, true
		} else {
			l.vueA = lireLaVueA(buf, l.debutVueA, cfg.Profil, grammaireDeLaVueA{}) // la tete seule decide
		}
		if !l.vueA.Vide {
			br.SetBitPos(l.vueA.finDeTete())
			l.finVueA = br.BitPos()
			l.publier(br) // vue C non atteinte : un TROU, que l appelant compte
			return
		}
		br.SetBitPos(l.vueA.Fin)
		l.finVueA = br.BitPos()
		l.rangs++
	} else {
		br.Skip(d.bit)
	}
	// RANG 1 — le gestionnaire d entites. L index de vue du MONDE HORS LIGNE pour cette classe
	// est `vueDeLImageCle` : le film la nomme rang 1, le monde la range en 0 (cf.
	// [World.vueDeLEspaceDeNoms]).
	w.PoserVueCourante(int(vueDeLImageCle))
	l.debutVueB = br.BitPos()
	var hitEnd bool
	l.recs, _, hitEnd = decodeInferLoop(br, buf, w, cfg)
	l.finVueB, l.sortieVueB, l.eidRejete = br.BitPos(), br.sortieVueB, br.eidRejete
	if !hitEnd {
		l.publier(br)
		return
	}
	l.rangs++
	// RANG 2 — la vue de controle. Elle ne rend AUCUN record d entite : ses records vont dans
	// le tableau que `FUN_142987460` applique par `vtable[0x48]`, et ce ne sont pas des deltas
	// d entite. La marche hors ligne n en publie donc aucun — c est ce qui supprime les records
	// DEL fantomes du pied de trame. Ses ENTREES DE CONTROLE (le tir continu, lot M4b) sont
	// publiees au hook, avec le verdict de fermeture ([verdictDeVueC]) : une vue qui ne ferme pas
	// le paquet ne rend rien.
	l.vueCAtteinte, l.debutVueC = true, br.BitPos()
	l.fluxC = consumeVueC(br, frameLen)
	if l.fluxC.Porte {
		l.rangs++
	}
	l.verdict = verdictDeVueC(buf, br.BitPos(), l.fluxC, l.recs, estUnRejet(l.sortieVueB))
	l.publier(br)
}

// publier rend le verdict de la vue C au crochet et clot la lecture : le curseur, le debordement.
func (l *lectureDeTrame) publier(br *Lecteur) {
	br.publierVueC(l.verdict)
	l.curseur, l.deborde = br.BitPos(), br.Deborde()
}
