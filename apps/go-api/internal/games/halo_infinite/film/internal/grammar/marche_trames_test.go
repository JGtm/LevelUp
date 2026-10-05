package grammar

// marche_trames_test.go — LA MARCHE DES TRAMES RANGE CE QU ELLE LIT (ADR 0037 IR-1, IR-4 a IR-6) :
// etendues des vues, des records, des composants et des tours de vue C, sortie typee de la vue B,
// verdict de fermeture, provenance des liaisons et table d entites, sur des paquets synthetiques
// ecrits bit a bit (`frame_closure_test.go`), dont chaque position est connue. Sur les bobines du
// depot : `marche_trames_bobines_test.go`.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// rangerUn marche un payload depuis la tete du paquet sur le monde `w`, comme la marche des trames
// ([marcheurDesTrames.marcherLePaquet]), et le range dans un paquet neuf.
func rangerUn(w *World, pay []byte) *lecture.Paquet {
	cfg := cadreDeCarte()
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, pay, w, cfg, departDeTrame{bit: DefaultPacketPreambleBits}, &l)
	p := &lecture.Paquet{Payload: pay, Debut: lecture.DebutEnTete}
	rangerLaTrame(p, &l, true, nil)
	return p
}

// rangerUnAvec marche un payload depuis la tete comme [marcheurDesTrames.marcherLePaquet] : la vue A
// lue et rangee d abord, sous la grammaire `g` que le film declare ([rangerLaTete]), puis passee a la
// marche par rangs.
func rangerUnAvec(w *World, pay []byte, g grammaireDeLaVueA) *lecture.Paquet {
	cfg := cadreDeCarte()
	p := &lecture.Paquet{Payload: pay, Debut: lecture.DebutEnTete}
	a := rangerLaTete(p, cfg.Profil, g)
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, pay, w, cfg, departDeTrame{bit: DefaultPacketPreambleBits, vueA: &a}, &l)
	rangerLaTrame(p, &l, true, nil)
	return p
}

// TestLaMarcheRangeUnPaquetFerme : bit de configuration, vue A vide, un DELTA du slot 123 (21 bits),
// le terminateur de la vue B, une vue C a une entree. Chaque etendue tombe sur le bit ecrit, le
// paquet est ferme et son record aussi. MUTATION — la fin d un record prise a la fin de son en-tete
// (`rec.FinBit = finEntete` dans [decodeInferLoop]) : son etendue tombe a 16 bits, ROUGE.
func TestLaMarcheRangeUnPaquetFerme(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.finDeVueB()
	debutC := bw.n
	bw.vueCUneEntree()
	fin := bw.n
	p := rangerUn(mondeDeCarte(), bw.buf)

	if a := p.VueA; a.Debut != 1 || a.Bits != 1 || a.Etat != lecture.VueTerminee || len(a.Genres) != 0 {
		t.Errorf("vue A %+v, attendu [1, 2) terminee, sans message", a)
	}
	if b := p.VueB; b != (lecture.VueB{Debut: 2, Bits: 24, Sortie: lecture.SortieTerminateur}) {
		t.Errorf("vue B %+v, attendu [2, 26) sortie par son terminateur", b)
	}
	want := lecture.Record{Genre: lecture.GenreDelta, Vue: lecture.RangVueB, Liaison: lecture.LiaisonImageCle,
		Preuve: lecture.PreuveFerme, TI: 4, Desync: lecture.SansDesynchronisation,
		Vie: types.LifeKey{Slot: 123, Gen: 1}, Debut: 2, Bits: 21}
	if len(p.Records) != 1 || p.Records[0] != want {
		t.Fatalf("records %+v, attendu %+v", p.Records, want)
	}
	c := p.VueC
	if c.Debut != uint32(debutC) || c.Bits != uint32(fin-debutC) || c.Etat != lecture.VueTerminee || len(c.Entrees) != 1 {
		t.Fatalf("vue C %+v, attendu [%d, %d) terminee, une entree", c, debutC, fin)
	}
	if e := c.Entrees[0]; e != (lecture.EntreeVueC{Debut: uint32(debutC), Bits: uint32(fin - 1 - debutC),
		Kind: kindVueCControle, Index: 2}) {
		t.Errorf("entree %+v, attendu [%d, %d) kind 0 index 2", e, debutC, fin-1)
	}
	if f := p.Fermeture; f != (lecture.Fermeture{Verdict: lecture.VerdictFerme, AuBit: true,
		Consommes: uint32(fin), Longueur: uint32(len(bw.buf) * 8)}) {
		t.Errorf("fermeture %+v, attendu fermee au bit pres, %d bits consommes", f, fin)
	}
}

// TestLaMarcheRangeLaSortieParRejet : un en-tete DELTA d un slot que le monde ne lie pas arrete la
// vue B ; il n est pas un record, la vue B finit sur lui et rend son eid complet, la vue C est lue
// de la. Le paquet se ferme au bit pres mais sa fermeture est REFUSEE : la sortie par rejet
// contredit l ecrivain. Aucune queue opaque, aucun record prouve.
func TestLaMarcheRangeLaSortieParRejet(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.enteteDelta13(500)
	bw.bit(0) // la vue C : son terminateur seul
	p := rangerUn(mondeDeCarte(), bw.buf)

	if b := p.VueB; b != (lecture.VueB{Debut: 2, Bits: 21 + 16, Sortie: lecture.SortieRejetHorsDatum,
		EIDRejete: 1<<30 | 500}) {
		t.Errorf("vue B %+v, attendu [2, 39) sortie par rejet hors datum de 0x%x", b, uint32(1<<30|500))
	}
	if len(p.Records) != 1 || p.Records[0].Preuve != lecture.PreuveNonProuve {
		t.Errorf("records %+v, attendu le seul DELTA 123, non prouve", p.Records)
	}
	if c := p.VueC; c.Debut != 39 || c.Bits != 1 || c.Etat != lecture.VueTerminee || len(c.Entrees) != 0 {
		t.Errorf("vue C %+v, attendu [39, 40) terminee et vide", c)
	}
	if f := p.Fermeture; f != (lecture.Fermeture{Verdict: lecture.VerdictRefuse, AuBit: true,
		Regle: uint8(InvariantSortieParRejet), Consommes: 40, Longueur: 40}) {
		t.Errorf("fermeture %+v, attendu refusee par la regle de la sortie par rejet", f)
	}
}

// TestLaMarcheRangeLaQueueDUnComposantNonPorte : un DELTA dont le masque annonce un composant sans
// lecteur. La vue B s arrete sur ce record, infranchissable ; son composant est rendu a sa position,
// sans largeur, et la queue opaque part de lui, le record et le composant designes. La vue C n est
// pas lue.
func TestLaMarcheRangeLaQueueDUnComposantNonPorte(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(124)
	bw.deltaMasque13(123, 0)
	debutComposant := bw.n
	bw.bits(0x2a5, 10) // la charge que personne ne sait lire
	p := rangerUn(mondeDeCarte(composantSansLecteurJ40), bw.buf)

	if p.VueB.Sortie != lecture.SortieRecordInfranchissable || len(p.Records) != 2 {
		t.Fatalf("vue B %+v, %d record(s) : attendu deux records, sortie sur un record infranchissable",
			p.VueB, len(p.Records))
	}
	r := p.Records[1]
	if r.Desync != 0 || r.Debut != 23 || r.Bits != uint32(debutComposant-23) || r.Masque != 1 || r.Comps != [2]uint32{0, 1} {
		t.Errorf("record %+v, attendu desynchronise au composant 0, [23, %d), un composant", r, debutComposant)
	}
	if c := p.Comps[0]; c != (lecture.Composant{Index: 0, Etat: lecture.EtatInfranchissable,
		Debut: uint32(debutComposant)}) {
		t.Errorf("composant %+v, attendu infranchissable au bit %d, sans largeur", c, debutComposant)
	}
	want := lecture.QueueOpaque{Debut: uint32(debutComposant), Cause: lecture.CauseComposantNonPorte, Record: 1, Composant: 0}
	if f := p.Fermeture; f.Verdict != lecture.VerdictQueueOpaque || f.Queue != want {
		t.Errorf("fermeture %+v, attendu la queue opaque %+v", f, want)
	}
	if p.VueC.Etat != lecture.VueNonLue || p.Records[0].Preuve != lecture.PreuveNonProuve {
		t.Errorf("vue C %+v, preuve %d : attendu une vue C non lue et un record non prouve", p.VueC, p.Records[0].Preuve)
	}
}

// TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide : une vue A qui porte un message arrete la
// marche partie de la tete apres sa tete (le genre du premier message) ; la vue B n est pas
// atteinte et le paquet est une queue opaque depuis la fin de ce genre.
//
// LE GENRE 5 EST PORTE DEPUIS LE LOT VA (`projectile_detonate`, `FUN_1408096f8`, position de niveau
// 0xf) : sous un film dont la table des genres est la table native, la vue A se lit jusqu a son
// terminateur et se range terminee, avec son etendue entiere ; sous un film sans table, elle
// s arrete apres son genre. Dans les deux cas la marche s arrete au meme bit : la vue B d un paquet a
// evenements part d un debut localise ([marcheurDesTrames.marcherLePaquet]), et la fin de la vue A
// n en decide pas a l etape V1 du lot.
func TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide(t *testing.T) {
	var bw bitWriter
	bw.bit(1)     // le bit de configuration
	bw.bit(1)     // la vue A porte un message
	bw.bits(5, 7) // son genre : projectile_detonate
	bw.bits(0, 3) // trois gardes de reference fermees
	bw.bits(0, 6) // FUN_140809454
	bw.bit(1)     // variante presente : rien ne suit
	bw.bit(0)     // FUN_14080d69c absent
	bw.bit(1)     // position : porte posee, table DEFAUT au niveau 0xf
	w := profile.LargeursAxeParDefautDuBuild(0xf)
	bw.bits(0, int(w[0]+w[1]+w[2]))
	bw.bits(0, 0x13+5+1+9) // direction, FUN_1406d84b4 (5), R(1), R(9)
	bw.bit(0)              // R(1) : FUN_140809530 absent
	bw.bit(0)              // seconde direction a 8 bits
	bw.bits(0, 8+2)        // FUN_14076dc04, FUN_1424cd2fc
	bw.bit(0)              // le terminateur de la vue A
	finVueA := bw.n
	bw.bits(0, 16)
	for _, c := range []struct {
		nom  string
		g    grammaireDeLaVueA
		vueA lecture.VueA
	}{
		{"film sans table", grammaireDeLaVueA{}, lecture.VueA{Debut: 1, Bits: 8, Etat: lecture.VueArretee, Genres: []uint8{5}}},
		{"film recent", grammaireRecente(), lecture.VueA{Debut: 1, Bits: uint32(finVueA - 1), Etat: lecture.VueTerminee,
			Genres: []uint8{5}}},
	} {
		p := rangerUnAvec(mondeDeCarte(), bw.buf, c.g)
		if a := p.VueA; a.Debut != c.vueA.Debut || a.Bits != c.vueA.Bits || a.Etat != c.vueA.Etat ||
			!slices.Equal(a.Genres, c.vueA.Genres) {
			t.Errorf("%s : vue A %+v, attendu %+v", c.nom, a, c.vueA)
		}
		if p.VueB != (lecture.VueB{}) || len(p.Records) != 0 {
			t.Errorf("%s : vue B %+v, records %+v : attendu non atteinte", c.nom, p.VueB, p.Records)
		}
		want := lecture.QueueOpaque{Debut: 9, Cause: lecture.CauseMessageVueANonPorte,
			Record: lecture.SansRecord, Composant: lecture.SansComposant}
		if f := p.Fermeture; f.Verdict != lecture.VerdictQueueOpaque || f.Queue != want || f.Consommes != 9 {
			t.Errorf("%s : fermeture %+v, attendu la queue opaque %+v", c.nom, f, want)
		}
	}
}

// TestLaMarcheNeRelitPasLaVueAQuElleRecoit : la marche des trames lit la vue A UNE fois
// ([rangerLaTete]) et la passe a la marche par rangs par son depart ; la marche ne la relit pas.
// Une vue A recue VIDE fait entrer la marche dans la vue B a sa fin, meme sur un payload dont le
// premier message n est pas le terminateur, et la marche ne la re-range pas. MUTATION — la marche
// qui relit la vue A malgre son depart : ROUGE.
func TestLaMarcheNeRelitPasLaVueAQuElleRecoit(t *testing.T) {
	var bw bitWriter
	bw.bit(1)     // le bit de configuration
	bw.bit(1)     // une continuation a 1 : la vue A du payload n est pas vide
	bw.bits(0, 7) // genre 0
	bw.bits(0, 16)
	cfg := cadreDeCarte()
	recue := FluxVueA{Debut: 1, Vide: true, Porte: true, Fin: 2}
	br := LecteurSur(bw.buf)
	br.poserCadre(cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, bw.buf, mondeDeCarte(), cfg, departDeTrame{bit: DefaultPacketPreambleBits, vueA: &recue}, &l)
	if !l.vueARecue || l.debutVueB != 2 || l.rangs == 0 {
		t.Fatalf("lecture %+v : attendu la vue A recue, la vue B a partir du bit 2", l)
	}
	p := &lecture.Paquet{Payload: bw.buf}
	rangerLaTrame(p, &l, true, nil)
	if p.VueA.Bits != 0 || p.VueA.Etat != lecture.VueNonLue {
		t.Errorf("vue A %+v re-rangee par la marche : elle l est par rangerLaTete seule", p.VueA)
	}
}

// TestLaFermetureNeConfondJamaisSesTroisVerdicts : chaque arret de la marche rend son verdict et,
// pour une queue opaque, sa cause et sa position — fermeture refusee et queue opaque ne se
// confondent pas (ADR 0037 IR-6).
func TestLaFermetureNeConfondJamaisSesTroisVerdicts(t *testing.T) {
	vueC := func(arret ArretVueC) lectureDeTrame {
		return lectureDeTrame{vueCAtteinte: true, fluxC: FluxVueC{Arret: arret}, curseur: 77}
	}
	vueB := func(s lecture.SortieVueB) lectureDeTrame { return lectureDeTrame{sortieVueB: s, curseur: 50} }
	cas := []struct {
		nom     string
		l       lectureDeTrame
		records []lecture.Record
		verdict lecture.Verdict
		cause   lecture.CauseDeQueue
		record  int32
	}{
		{"vue A sans terminateur", lectureDeTrame{enTete: true}, nil, lecture.VerdictQueueOpaque,
			lecture.CauseFinDePayloadVueA, lecture.SansRecord},
		{"plafond de la vue B", vueB(lecture.SortiePlafond), nil, lecture.VerdictQueueOpaque,
			lecture.CausePlafondVueB, lecture.SansRecord},
		{"fin du payload dans la vue B", vueB(lecture.SortieFinDePayload), nil, lecture.VerdictQueueOpaque,
			lecture.CauseFinDePayloadVueB, lecture.SansRecord},
		{"slot non lie", vueB(lecture.SortieRecordInfranchissable),
			[]lecture.Record{{TI: lecture.TINonResolu, Desync: 0}}, lecture.VerdictQueueOpaque,
			lecture.CauseSlotNonLie, 0},
		{"archetype hors registre", vueB(lecture.SortieRecordInfranchissable),
			[]lecture.Record{{TI: 7, Desync: 0}}, lecture.VerdictQueueOpaque,
			lecture.CauseArchetypeHorsRegistre, 0},
		{"debordement de la vue C", vueC(ArretVueCDebordement), nil, lecture.VerdictQueueOpaque,
			lecture.CauseDebordementVueC, lecture.SansRecord},
		{"kind non porte", vueC(ArretVueCKindNonPorte), nil, lecture.VerdictQueueOpaque,
			lecture.CauseKindVueCNonPorte, lecture.SansRecord},
		{"bloc 0xbc", vueC(ArretVueCBlocBC), nil, lecture.VerdictQueueOpaque,
			lecture.CauseBlocBCVueC, lecture.SansRecord},
		{"plafond de la vue C", vueC(ArretVueCPlafond), nil, lecture.VerdictQueueOpaque,
			lecture.CausePlafondVueC, lecture.SansRecord},
		{"ferme", lectureDeTrame{vueCAtteinte: true, fluxC: FluxVueC{Porte: true},
			verdict: LectureVueC{Fermee: true, FermeeAuBit: true}}, nil, lecture.VerdictFerme,
			lecture.CauseAucune, 0},
		{"refuse", lectureDeTrame{vueCAtteinte: true, fluxC: FluxVueC{Porte: true},
			verdict: LectureVueC{FermeeAuBit: true, Invariant: InvariantOrdreVueB}}, nil, lecture.VerdictRefuse,
			lecture.CauseAucune, 0},
	}
	for _, c := range cas {
		p := &lecture.Paquet{Records: c.records}
		rangerLaFermeture(p, &c.l)
		f := p.Fermeture
		if f.Verdict != c.verdict || f.Queue.Cause != c.cause {
			t.Errorf("%s : verdict %d cause %d, attendu %d et %d", c.nom, f.Verdict, f.Queue.Cause, c.verdict, c.cause)
			continue
		}
		if c.verdict != lecture.VerdictQueueOpaque {
			if f.Queue != (lecture.QueueOpaque{}) {
				t.Errorf("%s : queue %+v hors d une queue opaque", c.nom, f.Queue)
			}
			continue
		}
		if f.Queue.Debut != uint32(max(c.l.curseur, 0)) || f.Queue.Record != c.record {
			t.Errorf("%s : queue %+v, attendu au curseur %d, record %d", c.nom, f.Queue, c.l.curseur, c.record)
		}
	}
}

// TestUneListeNonLocaliseeEstUneQueueOpaque : un paquet a liste d evenements dont le debut n est
// pas trouve n a aucune vue lue ; il est une queue opaque depuis son premier bit.
func TestUneListeNonLocaliseeEstUneQueueOpaque(t *testing.T) {
	p := &lecture.Paquet{Payload: make([]byte, 5), Debut: lecture.DebutNonLocalise}
	rangerUneListeNonLocalisee(p)
	want := lecture.Fermeture{Verdict: lecture.VerdictQueueOpaque, Longueur: 40,
		Queue: lecture.QueueOpaque{Cause: lecture.CauseListeNonLocalisee, Record: lecture.SansRecord,
			Composant: lecture.SansComposant}}
	if p.Fermeture != want || len(p.Records) != 0 {
		t.Errorf("fermeture %+v, attendu %+v", p.Fermeture, want)
	}
}

// TestLesLiaisonsDesRecordsDisentLeurProvenance : un NEW lu pose sa liaison (« lue »), un DELTA
// lu sous une liaison d anticipation le dit, un DEL n en porte aucune. La table d entites de la
// marche rend, apres la trame, l entite que le NEW a liee et celle que l anticipation a liee.
func TestLesLiaisonsDesRecordsDisentLeurProvenance(t *testing.T) {
	w := mondeDeCarte()
	tab := NouvelleTableAnticipee()
	tab.entrees[cleAnticipee{slot: 77, tete: 1}] = []declarationAnticipee{{chunk: 5, ti: 3}}
	w.PoserTableAnticipee(tab)
	w.PoserChunkCourant(2)
	var bw bitWriter
	bw.teteDePaquet()
	bw.neuf13(300, 2)
	bw.deltaMasque13(77)
	bw.bit(0) // DEL : type `0` puis `10`
	bw.bits(recDel, 2)
	bw.bits(124, 13)
	bw.bits(1, 2)
	bw.bits(0, 32)
	bw.finDeVueB()
	bw.bit(0) // vue C vide
	p := rangerUn(w, bw.buf)

	attendus := []struct {
		genre   lecture.Genre
		liaison lecture.Liaison
		ti      int16
	}{
		{lecture.GenreNeuf, lecture.LiaisonLueNeuf, 2},
		{lecture.GenreDelta, lecture.LiaisonAnticipation, 3},
		{lecture.GenreSuppression, lecture.LiaisonAucune, lecture.TINonResolu},
	}
	if len(p.Records) != len(attendus) {
		t.Fatalf("records %+v, attendu un NEW, un DELTA et un DEL", p.Records)
	}
	for i, a := range attendus {
		if r := p.Records[i]; r.Genre != a.genre || r.Liaison != a.liaison || r.TI != a.ti {
			t.Errorf("record %d : %+v, attendu genre %d liaison %d ti %d", i, r, a.genre, a.liaison, a.ti)
		}
	}
	e := entitesDuMonde{w: w}
	if n, ok := e.Entite(300); !ok || n.Liaison != lecture.LiaisonLueNeuf || n.TI != 2 || n.EID != 1<<30|300 || !n.GenerationConnue {
		t.Errorf("entite 300 : %+v (%t), attendu liee par le NEW lu, eid complet", n, ok)
	}
	if n, ok := e.Entite(77); !ok || n.Liaison != lecture.LiaisonAnticipation || n.TI != 3 || n.GenerationConnue {
		t.Errorf("entite 77 : %+v (%t), attendu liee par anticipation, generation inconnue", n, ok)
	}
	if _, ok := e.Entite(124); ok {
		t.Error("entite 124 encore liee : le DEL l a retiree")
	}
}

// TestLaTableDEntitesDitLaProvenanceDeChaqueLiaison : chaque porte de liaison du monde pose sa
// provenance, et la table d entites la rend avec l eid, l archetype, le rang de vue du FILM et la
// generation connue ou non (ADR 0037 IR-5). Un slot non lie n est pas une entite.
func TestLaTableDEntitesDitLaProvenanceDeChaqueLiaison(t *testing.T) {
	w := NewWorld(&Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}}})
	w.BindImageCle(1, 10, 4)
	w.BindImageCle(2, 11, 4) // un second rang d image-cle : une vue que rien ne situe
	w.BindDatum(12, 3)
	w.BindFull(2<<30|13, 2)
	w.BindSoft(1<<30|14, 1)
	w.BindWildcard(15, 0)
	vueB := int8(lecture.RangVueB)
	attendues := map[uint32]lecture.Entite{
		10: {EID: 10, TI: 4, Vue: vueB, Liaison: lecture.LiaisonImageCle},
		11: {EID: 11, TI: 4, Vue: lecture.VueInconnue, Liaison: lecture.LiaisonImageCle},
		12: {EID: 12, TI: 3, Vue: lecture.VueInconnue, Liaison: lecture.LiaisonDatum},
		13: {EID: 2<<30 | 13, TI: 2, Vue: vueB, Liaison: lecture.LiaisonLueNeuf, GenerationConnue: true},
		14: {EID: 1<<30 | 14, TI: 1, Vue: vueB, Liaison: lecture.LiaisonInference, GenerationConnue: true},
		15: {EID: 15, TI: 0, Vue: vueB, Liaison: lecture.LiaisonJoker},
	}
	e := entitesDuMonde{w: w}
	for slot, want := range attendues {
		if got, ok := e.Entite(slot); !ok || got != want {
			t.Errorf("slot %d : %+v (%t), attendu %+v", slot, got, ok, want)
		}
	}
	if e.Liees() != len(attendues) {
		t.Errorf("%d entite(s) liee(s), attendu %d", e.Liees(), len(attendues))
	}
	if _, ok := e.Entite(99); ok {
		t.Error("slot 99 jamais lie et rendu comme entite")
	}
}
