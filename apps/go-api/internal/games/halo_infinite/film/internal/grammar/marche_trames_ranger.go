package grammar

// marche_trames_ranger.go — CE QUE LA MARCHE DES TRAMES RANGE DANS LA STRUCTURE DE LECTURE
// (ADR 0037 IR-1, IR-4 a IR-6) : des fonctions de ce que la marche par rangs a DEJA lu
// ([lectureDeTrame]), sans relire un bit.
//
// Les etendues sont celles de la marche : un record va de son en-tete ([FrameRecord.HeaderBit])
// a la position qui suit son dernier bit lu ([FrameRecord.FinBit]) ; un composant, de son
// premier bit au debut du suivant (au bout du corps pour le dernier) ; un tour de vue C, de son
// bit de continuation au tour suivant (au terminateur pour le dernier, a la position d arret pour
// un tour qui arrete la marche).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// decalageDeGeneration : la generation d un handle est dans les deux bits de tete de l eid
// ([readRecordID]).
const decalageDeGeneration = 30

// rangerUneListeNonLocalisee range un paquet a liste d evenements dont le debut n a pas ete
// trouve : aucune vue lue, une queue opaque depuis le premier bit.
func rangerUneListeNonLocalisee(p *lecture.Paquet) {
	p.Fermeture = lecture.Fermeture{Verdict: lecture.VerdictQueueOpaque,
		Longueur: uint32(len(p.Payload) * 8), //nolint:gosec // un payload tient sur 32 bits
		Queue:    queueOpaque(0, lecture.CauseListeNonLocalisee)}
}

// rangerLaTrame range ce que la marche a lu d une trame : ses records, et — marchee par classes
// de vue — ses vues et son verdict de fermeture. `interets` dit quelles occurrences les canaux de
// la marche interpretent.
func rangerLaTrame(p *lecture.Paquet, l *lectureDeTrame, parRangs bool, interets interetsResolus) {
	p.Fermeture.Longueur = uint32(len(p.Payload) * 8) //nolint:gosec // un payload tient sur 32 bits
	p.Fermeture.Consommes = uint32(max(l.curseur, 0)) //nolint:gosec // borne ci-contre
	rangerLesRecords(p, l.recs, interets)
	if !parRangs {
		for i := range p.Records {
			p.Records[i].Preuve = lecture.PreuveNonProuve // aucune vue C lue, aucun verdict
		}
		return
	}
	rangerLesVues(p, l)
	rangerLaFermeture(p, l)
}

// rangerLesRecords range les records de la vue B et leurs composants dans l arene.
func rangerLesRecords(p *lecture.Paquet, recs []FrameRecord, interets interetsResolus) {
	for i := range recs {
		r := &recs[i]
		ti := archetypeDuRecord(r)
		premier := uint32(len(p.Comps)) //nolint:gosec // l arene d un paquet tient sur 32 bits
		for k := range r.Trace.Comps {
			p.Comps = append(p.Comps, composantLu(&r.Trace, k, interets.contient(int(ti), r.Trace.Comps[k].Index)))
		}
		p.Records = append(p.Records, lecture.Record{
			Genre: genreDuRecord(r.Type), Vue: lecture.RangVueB, Liaison: r.Liaison,
			TI: ti, Desync: int16(r.DesyncAt), //nolint:gosec // index de composant < 64
			Vie:   types.LifeKey{Slot: r.Slot, Gen: r.ID >> decalageDeGeneration},
			Debut: uint32(r.HeaderBit), Bits: uint32(r.FinBit - r.HeaderBit), //nolint:gosec // positions d un payload
			Masque: r.Trace.Mask,
			Comps:  [2]uint32{premier, uint32(len(p.Comps))}, //nolint:gosec // idem
		})
	}
}

// genreDuRecord rend le genre d un record de la vue B.
func genreDuRecord(typ int) lecture.Genre {
	switch typ {
	case recNew:
		return lecture.GenreNeuf
	case recDel:
		return lecture.GenreSuppression
	case recDelta:
		return lecture.GenreDelta
	}
	return lecture.GenreNonRenseigne
}

// archetypeDuRecord rend l archetype d un record, [lecture.TINonResolu] pour un DEL (il n en
// porte pas) et pour un delta dont l archetype n a pas ete resolu ([cleArchetype]).
func archetypeDuRecord(r *FrameRecord) int16 {
	if r.Type == recDel || cleArchetype(*r) == ArchetypeNonResolu {
		return lecture.TINonResolu
	}
	return int16(r.TypeIndex) //nolint:gosec // un archetype tient sur 6 bits
}

// composantLu rend l occurrence `k` d une trace : un composant non porte est infranchissable et
// n a pas de largeur ; un composant porte est traverse — interprete quand un canal de la marche
// l interprete (`interesse`, ADR 0037 IR-4), delimite sinon.
func composantLu(t *EntityTrace, k int, interesse bool) lecture.Composant {
	cr := &t.Comps[k]
	c := lecture.Composant{Index: uint8(cr.Index), Debut: uint32(cr.StartBit)} //nolint:gosec // index < 64, position d un payload
	if !cr.Ported {
		c.Etat = lecture.EtatInfranchissable
		return c
	}
	fin := t.EndBit
	if k+1 < len(t.Comps) {
		fin = t.Comps[k+1].StartBit
	}
	c.Etat, c.Prov, c.Bits = lecture.EtatDelimite, cr.Prov, uint32(fin-cr.StartBit) //nolint:gosec // fin >= debut
	if interesse {
		c.Etat = lecture.EtatInterprete
	}
	return c
}

// rangerLesVues range l etendue et l etat des trois vues. Une marche partie de la tete re-range la
// vue A qu elle a lue (la tete rangee avant la marche n en est que le debut) ; depuis un debut
// localise, la vue A reste la tete ([rangerLaTete]).
func rangerLesVues(p *lecture.Paquet, l *lectureDeTrame) {
	if l.enTete {
		p.VueA.Debut, p.VueA.Bits = uint32(l.debutVueA), uint32(l.finVueA-l.debutVueA) //nolint:gosec // positions
		p.VueA.Etat = etatDeVue(l.vueA.Porte)
		p.VueA.Genres = p.VueA.Genres[:0]
		for _, g := range l.vueA.Genres {
			p.VueA.Genres = append(p.VueA.Genres, uint8(g)) //nolint:gosec // genre R(7)
		}
	}
	if l.debutVueB >= 0 {
		p.VueB = lecture.VueB{Debut: uint32(l.debutVueB), Bits: uint32(l.finVueB - l.debutVueB), //nolint:gosec // positions
			Sortie: l.sortieVueB, EIDRejete: l.eidRejete}
	}
	if l.vueCAtteinte {
		p.VueC.Debut, p.VueC.Bits = uint32(l.debutVueC), uint32(l.curseur-l.debutVueC) //nolint:gosec // positions
		p.VueC.Etat = etatDeVue(l.fluxC.Porte)
		p.VueC.Entrees = toursDeLaVueC(p.VueC.Entrees, &l.fluxC, l.curseur)
	}
}

// etatDeVue rend l etat d une vue lue : terminee, ou arretee avant son terminateur.
func etatDeVue(terminee bool) lecture.EtatDeVue {
	if terminee {
		return lecture.VueTerminee
	}
	return lecture.VueArretee
}

// toursDeLaVueC ajoute a `out` les tours de la vue C, dans l ordre du flux. Une entree `kind 0`
// lue jusqu au bout porte l index de controle de son participant ; celle qui arrete la marche
// n en porte pas.
func toursDeLaVueC(out []lecture.EntreeVueC, c *FluxVueC, curseur int) []lecture.EntreeVueC {
	fin := curseur
	if c.Porte {
		fin = curseur - 1 // le terminateur `R(1) = 0`
	}
	entree := 0
	for k, debut := range c.Tours {
		finDuTour := fin
		if k+1 < len(c.Tours) {
			finDuTour = c.Tours[k+1]
		}
		e := lecture.EntreeVueC{Debut: uint32(debut), Bits: uint32(finDuTour - debut), //nolint:gosec // positions
			Kind: uint8(c.Kinds[k]), Index: lecture.IndexDeControleAbsent} //nolint:gosec // R(2)
		if c.Kinds[k] == kindVueCControle && entree < len(c.Entrees) {
			e.Index = int8(c.Entrees[entree].Index) //nolint:gosec // R(5)
			entree++
		}
		out = append(out, e)
	}
	return out
}

// rangerLaFermeture range le verdict de fermeture de la trame et la preuve de ses records.
func rangerLaFermeture(p *lecture.Paquet, l *lectureDeTrame) {
	f := &p.Fermeture
	f.AuBit, f.Regle = l.verdict.FermeeAuBit, uint8(l.verdict.Invariant)
	switch {
	case l.enTete && !l.vueA.Porte:
		f.Verdict, f.Queue = lecture.VerdictQueueOpaque, queueOpaque(l.curseur, causeDeLaVueA(&l.vueA))
	case !l.vueCAtteinte:
		f.Verdict, f.Queue = lecture.VerdictQueueOpaque, queueDeLaVueB(p, l)
	case !l.fluxC.Porte:
		f.Verdict, f.Queue = lecture.VerdictQueueOpaque, queueOpaque(l.curseur, causeDeLaVueC(l.fluxC.Arret))
	case l.verdict.Fermee:
		f.Verdict = lecture.VerdictFerme
	default:
		f.Verdict = lecture.VerdictRefuse
	}
	for i := range p.Records {
		p.Records[i].Preuve = lecture.PreuveNonProuve
		if f.Verdict == lecture.VerdictFerme && p.Records[i].Desync == lecture.SansDesynchronisation {
			p.Records[i].Preuve = lecture.PreuveFerme
		}
	}
}

// queueOpaque rend une queue opaque qui ne designe aucun record.
func queueOpaque(debut int, cause lecture.CauseDeQueue) lecture.QueueOpaque {
	return lecture.QueueOpaque{Debut: uint32(max(debut, 0)), Cause: cause, //nolint:gosec // borne ci-contre
		Record: lecture.SansRecord, Composant: lecture.SansComposant}
}

// causeDeLaVueA rend ce qui a arrete la vue A : un corps de message non porte, ou la fin du
// payload avant son terminateur.
func causeDeLaVueA(a *FluxVueA) lecture.CauseDeQueue {
	if len(a.Genres) > 0 {
		return lecture.CauseMessageVueANonPorte
	}
	return lecture.CauseFinDePayloadVueA
}

// queueDeLaVueB rend la queue opaque d une vue B qui n a pas clos sa liste : le record qui l a
// arretee et son composant infranchissable, la fin du payload, ou le plafond de la boucle.
func queueDeLaVueB(p *lecture.Paquet, l *lectureDeTrame) lecture.QueueOpaque {
	if l.sortieVueB == lecture.SortiePlafond {
		return queueOpaque(l.curseur, lecture.CausePlafondVueB)
	}
	n := len(p.Records) - 1
	if l.sortieVueB != lecture.SortieRecordInfranchissable || n < 0 {
		return queueOpaque(l.curseur, lecture.CauseFinDePayloadVueB)
	}
	r := &p.Records[n]
	q := queueOpaque(l.curseur, lecture.CauseArchetypeHorsRegistre)
	q.Record = int32(n) //nolint:gosec // rang d un record du paquet
	switch {
	case r.Comps[1] > r.Comps[0] && p.Comps[r.Comps[1]-1].Etat == lecture.EtatInfranchissable:
		c := p.Comps[r.Comps[1]-1]
		q.Debut, q.Cause, q.Composant = c.Debut, lecture.CauseComposantNonPorte, int16(c.Index)
	case r.TI == lecture.TINonResolu:
		q.Cause = lecture.CauseSlotNonLie
	}
	return q
}

// causeDeLaVueC rend la cause typee d un arret de la vue C ([ArretVueC]) ; le plafond de tours
// est le seul arret qui reste.
func causeDeLaVueC(a ArretVueC) lecture.CauseDeQueue {
	switch a {
	case ArretVueCDebordement:
		return lecture.CauseDebordementVueC
	case ArretVueCKindNonPorte:
		return lecture.CauseKindVueCNonPorte
	case ArretVueCBlocBC:
		return lecture.CauseBlocBCVueC
	}
	return lecture.CausePlafondVueC
}
