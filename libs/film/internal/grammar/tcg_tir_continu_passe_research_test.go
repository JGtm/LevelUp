//go:build research

package grammar

// tcg_tir_continu_passe_research_test.go — LA PASSE UNIQUE de la sonde P1 et les tableaux de S1
// (voir l en-tete de `tcg_tir_continu_research_test.go`). Un seul decodage du film : les images-
// cles lient le monde comme la marche de production du calque des etats (`t525Lier`), chaque
// paquet delta est lu deux fois AU MEME BIT DE DEPART — la liste d evenements par le marcheur R7,
// la vue B par `t519Marcher` —, et la seconde lecture sert d oracle a la premiere.

import (
	"fmt"
	"sort"
	"testing"
)

// tcgRec est UN record de la vue B retenu pour S2.
type tcgRec struct {
	trame, rang   int
	ts            uint64
	slot, gen, ti uint32
	typ           int
	masque        uint64
	desync        int
	fermee        bool
	parent        uint32 // slot du parent quand `i10` est lu attache
	aParent       bool
}

// tcgImageCle est UNE image-cle : les entites qu elle declare.
type tcgImageCle struct {
	trame int
	ents  []KeyframeRec
}

// tcgPasse porte tout ce que la passe rend.
type tcgPasse struct {
	cad                                        tcgCadre
	evs                                        []tcgEv
	ctls                                       []tcgCtl
	vueCLue, paquetsParTrame                   map[int]int // par trame : paquets dont la vue C est lue / paquets
	recs, news                                 []tcgRec
	ics                                        []tcgImageCle
	paquets, avecListe, valides, nonLocalises  int
	trames, fermees                            int
	stops                                      map[r7Stop]int
	opaques, courtes113, tetes, tetesValidees  map[int]int
	prodTetes, desaccords36, evsTotal, evsHors map[bool]int
}

// tcgMarcher decode le film UNE fois.
func tcgMarcher(tc t516Temoin, cad tcgCadre, ctx r7Ctx) *tcgPasse {
	p := &tcgPasse{cad: cad, stops: map[r7Stop]int{}, opaques: map[int]int{},
		courtes113: map[int]int{}, tetes: map[int]int{}, tetesValidees: map[int]int{},
		prodTetes: map[bool]int{}, desaccords36: map[bool]int{}, evsTotal: map[bool]int{},
		evsHors: map[bool]int{}, vueCLue: map[int]int{}, paquetsParTrame: map[int]int{}}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			switch {
			case pk.Type == PacketTypeKeyframe:
				p.ics = append(p.ics, tcgImageCle{cad.trame(pk.TimestampUS),
					WalkKeyframeWorld(pk.Payload(data))})
			case pk.Type == PacketTypeDelta && pk.Size >= 1:
				p.paquet(c, pk, pk.Payload(data), w, tc.cfg, ctx)
			}
		}
	}
	return p
}

// paquet lit UN paquet delta : la liste (S1), puis la vue B (S2).
func (p *tcgPasse) paquet(c int, pk FilmPacket, pay []byte, w *World, cfg FrameConfig, ctx r7Ctx) {
	p.paquets++
	tr := p.cad.trame(pk.TimestampUS)
	debut := DefaultPacketPreambleBits
	if typ, present := PacketHeadEventType(pay); present {
		p.avecListe++
		p.tetes[typ]++
		if typ == 36 && len(pay)*8 < ancienneGardeTeteTir {
			p.courtes113[len(pay)*8]++
		}
		p.prodTetes[pay[0] == 0xD2 && len(pay)*8 >= ancienneGardeTeteTir]++
		debut = marchLocate(pay, w, cfg)
		p.liste(c, pk, pay, tr, debut, ctx)
		if debut < 0 {
			p.nonLocalises++
			return
		}
	}
	mar := t519Marcher(pay, w, cfg, debut)
	p.trames++
	p.paquetsParTrame[tr]++
	reste := len(pay)*8 - mar.m.FinVueC
	fermee := reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, mar.m.FinVueC)
	if fermee {
		p.fermees++
	}
	if mar.m.HitEndB {
		p.vueCLue[tr]++
		p.ctls = append(p.ctls, tcgLireVueC(pay, mar.m.FinVueB, tr, pk.TimestampUS, fermee)...)
	}
	for i, r := range mar.recs {
		p.record(r, i, tr, pk.TimestampUS, fermee)
	}
}

// liste marche la liste d evenements, la valide contre le debut de la vue B, et garde les
// evenements utiles.
func (p *tcgPasse) liste(c int, pk FilmPacket, pay []byte, tr, debut int, ctx r7Ctx) {
	evs, stop, fin := tcgMarcherListe(pay, ctx)
	p.stops[stop]++
	if stop == r7StopOpaque && len(evs) > 0 {
		p.opaques[evs[len(evs)-1].typ]++
	}
	valide := debut >= 0 && stop == r7StopFin && fin == debut
	if valide {
		p.valides++
		p.tetesValidees[evs[0].typ]++
	}
	hors := p.cad.episodeDe(tr, 0) < 0
	for i := range evs {
		e := &evs[i]
		e.trame, e.chunk, e.paquet, e.ts = tr, c, pk.Index, pk.TimestampUS
		e.bitsPaquet, e.valide = len(pay)*8, valide
		p.evsTotal[valide]++
		if hors {
			p.evsHors[valide]++
		}
		if !tcgGarder(*e, p.cad) {
			continue
		}
		tcgDetailler(pay, e, ctx)
		if e.tir != nil {
			p.desaccords36[e.tir.desaccord]++
		}
		p.evs = append(p.evs, *e)
	}
}

// record retient un record de la vue B : vehicules et pilotes suivis, objets attaches a un
// vehicule suivi, et toute naissance (S2 y cherche l objet arme ne avec le vehicule).
func (p *tcgPasse) record(r FrameRecord, rang, tr int, ts uint64, fermee bool) {
	x := tcgRec{trame: tr, rang: rang, ts: ts, slot: r.Slot, gen: r.ID >> 30, ti: r.TypeIndex,
		typ: r.Type, masque: r.Trace.Mask, desync: r.DesyncAt, fermee: fermee}
	for _, cr := range r.Trace.Comps {
		if st, ok := cr.ParentOf(); ok && st.Attached {
			x.aParent = true
			x.parent = (st.Quant16 & parentHandleValueMask) + parentHandleBase
		}
	}
	if r.Type == recNew {
		p.news = append(p.news, x)
	}
	if p.cad.suivi(r.Slot) || (x.aParent && p.cad.vehiculeSuivi(x.parent)) {
		p.recs = append(p.recs, x)
	}
}

// tcgS1Recensement publie le cadrage : combien de listes, combien validees par l oracle.
func tcgS1Recensement(t *testing.T, p *tcgPasse) {
	t.Helper()
	t.Logf("== S1.0 RECENSEMENT : %d paquets delta · %d a liste · %d listes VALIDEES (fin de marche "+
		"== debut de vue B, %.1f %%) · %d non localisees · trames lues %d dont fermees %d",
		p.paquets, p.avecListe, p.valides, 100*float64(p.valides)/float64(max(1, p.avecListe)),
		p.nonLocalises, p.trames, p.fermees)
	t.Logf("   arrets de marche : fin %d · opaque %d · buffer %d · type>=123 %d · sans domaine %d",
		p.stops[r7StopFin], p.stops[r7StopOpaque], p.stops[r7StopBuffer], p.stops[r7StopTypeInconnu],
		p.stops[r7StopSansDomaine])
	t.Logf("   types OPAQUES (arret) : %s", tcgCompteTri(p.opaques, tcgNomType, 15))
	t.Logf("   tetes de liste : %s", tcgCompteTri(p.tetes, tcgNomType, 20))
	t.Logf("   evenements traverses : %d en liste validee, %d en liste non validee (hors episodes : "+
		"%d / %d)", p.evsTotal[true], p.evsTotal[false], p.evsHors[true], p.evsHors[false])
	t.Logf("   tetes que la PRODUCTION lit (octet 0 = 0xD2 et >= 113 bits) : %d ; tetes 36 de moins "+
		"de 113 bits (ecartees) : %s", p.prodTetes[true],
		tcgCompteTri(p.courtes113, func(k int) string { return fmt.Sprintf("%db", k) }, 0))
	t.Logf("   type 36 : fin du lecteur champ a champ == fin du marcheur : %d · DESACCORD %d",
		p.desaccords36[false], p.desaccords36[true])
}

// tcgS1Tirs publie les `action_weapon_fire` : position, classe d arme, et ce que la production en
// lit.
func tcgS1Tirs(t *testing.T, p *tcgPasse) {
	t.Helper()
	type cle struct {
		tete, valide bool
		classe       string
	}
	parCle := map[cle]int{}
	armes := map[string]map[uint64]int{}
	for _, e := range p.evs {
		if e.tir == nil {
			continue
		}
		k := cle{e.pos == 1, e.valide, e.tir.classe()}
		parCle[k]++
		if armes[k.classe] == nil {
			armes[k.classe] = map[uint64]int{}
		}
		armes[k.classe][e.tir.arme()]++
	}
	t.Logf("== S1.1 TYPE 36 action_weapon_fire, par (position, liste validee, classe d arme) :")
	cles := make([]cle, 0, len(parCle))
	for k := range parCle {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return fmt.Sprint(cles[i]) < fmt.Sprint(cles[j]) })
	for _, k := range cles {
		t.Logf("   tete=%-5v validee=%-5v classe=%-8s : %d", k.tete, k.valide, k.classe, parCle[k])
	}
	for _, c := range []string{"vehicule", "autre", "nul"} {
		t.Logf("   armes de classe %s : %s", c, tcgArmes(armes[c], 12))
	}
}

// tcgArmes formate une distribution d identifiants d arme.
func tcgArmes(m map[uint64]int, top int) string {
	cles := make([]uint64, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return m[cles[i]] > m[cles[j]] })
	out := ""
	for i, k := range cles {
		if i >= top {
			return out + fmt.Sprintf(" (+%d)", len(cles)-top)
		}
		out += fmt.Sprintf(" %08X:%08X=%d", k>>32, k&0xFFFFFFFF, m[k])
	}
	return out
}

// tcgLigne rend un evenement sur une ligne.
func tcgLigne(e tcgEv) string {
	s := fmt.Sprintf("t=%d c%d/p%d pos=%d %s refs=[%v %v %v] valide=%v bits=%d", e.trame, e.chunk,
		e.paquet, e.pos, tcgNomType(e.typ), e.refs[0], e.refs[1], e.refs[2], e.valide, e.bitsPaquet)
	switch {
	case e.tir != nil:
		x := e.tir
		s += fmt.Sprintf(" | court=%v bloc=%v att=%d idx5=%d p2=%d arme=%08X:%08X(%s) cibles=%v "+
			"desaccord=%v", x.court, x.bloc, x.attaquant, x.idx5, x.p2, x.haut, x.bas, x.classe(),
			x.cibles, x.desaccord)
	case e.degat != nil:
		s += fmt.Sprintf(" | source=%08X(%v) mag=%.2f victime=%d(%v)", e.degat.sourceID,
			e.degat.hasSource, e.degat.dmgClear, e.degat.victimIdx, e.degat.hasVictim)
	case e.aArme:
		s += fmt.Sprintf(" | arme=%08X:%08X", e.arme>>32, e.arme&0xFFFFFFFF)
	case e.aVariant:
		s += fmt.Sprintf(" | tag=%08X(%v) variante=%08X", e.tag, e.aTag, e.variante)
	}
	return s
}

// tcgS1Suivis publie, dans les episodes, les evenements qui touchent le pilote ou son vehicule :
// tout type 36 (tete ou non) dont l index 5 bits est celui du pilote ou dont la ref0 designe un
// slot suivi, et tout evenement d un autre type dont une reference designe un slot suivi.
func tcgS1Suivis(t *testing.T, p *tcgPasse) {
	t.Helper()
	parType, parTypeHors := map[int]int{}, map[int]int{}
	var lignes []string
	n36Episode, n36NonTete := 0, 0
	for _, e := range p.evs {
		dedans := p.cad.episodeDe(e.trame, 10) >= 0
		if e.tir != nil && dedans {
			n36Episode++
			if e.pos > 1 {
				n36NonTete++
			}
		}
		du := e.refSuivie(p.cad) || (e.tir != nil && e.tir.idx5 == p.cad.index)
		if !du {
			continue
		}
		if !dedans {
			parTypeHors[e.typ]++
			continue
		}
		parType[e.typ]++
		if len(lignes) < 120 {
			lignes = append(lignes, tcgLigne(e))
		}
	}
	t.Logf("== S1.2 DANS LES EPISODES (+/-1 s) : %d type 36 au total, dont %d HORS TETE", n36Episode,
		n36NonTete)
	t.Logf("   evenements qui designent le pilote / le vehicule (ref suivie, ou type 36 d index %d) : "+
		"%s", p.cad.index, tcgCompteTri(parType, tcgNomType, 0))
	t.Logf("   les memes HORS episodes : %s", tcgCompteTri(parTypeHors, tcgNomType, 0))
	for _, l := range lignes {
		t.Logf("     %s", l)
	}
	t.Logf("   type 36 HORS TETE dans les episodes (tous tireurs) :")
	k := 0
	for _, e := range p.evs {
		if e.tir != nil && e.pos > 1 && p.cad.episodeDe(e.trame, 10) >= 0 && k < 60 {
			t.Logf("     %s", tcgLigne(e))
			k++
		}
	}
}

// tcgFenetre compte, dans [f-20, f], les evenements d une categorie.
func tcgFenetre(p *tcgPasse, f int, cat func(tcgEv) bool) int {
	n := 0
	for _, e := range p.evs {
		if e.trame >= f-tcgAvantFrag && e.trame <= f && cat(e) {
			n++
		}
	}
	return n
}

// tcgS1Frags est le GATE de S1 : par categorie d evenement, la presence dans les 2 s avant chaque
// frag, contre le temoin decale de +/-60 s, et le compte hors des episodes.
func tcgS1Frags(t *testing.T, p *tcgPasse) {
	t.Helper()
	c := p.cad
	suivi := func(e tcgEv) bool { return e.refSuivie(c) }
	cats := []struct {
		nom string
		f   func(tcgEv) bool
	}{
		{"36 tout tireur", func(e tcgEv) bool { return e.tir != nil }},
		{"36 hors tete", func(e tcgEv) bool { return e.tir != nil && e.pos > 1 }},
		{"36 du pilote (idx5)", func(e tcgEv) bool { return e.tir != nil && e.tir.idx5 == c.index }},
		{"36 ref suivie", func(e tcgEv) bool { return e.tir != nil && suivi(e) }},
		{"37 surchauffe", func(e tcgEv) bool { return e.typ == 37 }},
		{"10 weapon_effect", func(e tcgEv) bool { return e.typ == 10 }},
		{"0 degat, ref suivie", func(e tcgEv) bool { return e.degat != nil && suivi(e) }},
		{"tout type, ref suivie", suivi},
	}
	t.Logf("== S1.3 GATE : presence dans [f-2 s, f] pour les %d frags, temoin -60 s / +60 s, et "+
		"total hors episodes (+/-1 s)", len(c.frags))
	for _, k := range cats {
		var par, moins, plus []int
		for _, f := range c.frags {
			par = append(par, tcgFenetre(p, f, k.f))
			moins = append(moins, tcgFenetre(p, f-tcgDecalage, k.f))
			plus = append(plus, tcgFenetre(p, f+tcgDecalage, k.f))
		}
		hors := 0
		for _, e := range p.evs {
			if k.f(e) && c.episodeDe(e.trame, 10) < 0 {
				hors++
			}
		}
		t.Logf("   %-22s frags=%v (couverts %d/%d) · temoin-60s=%v · temoin+60s=%v · hors episodes=%d",
			k.nom, par, tcgNonNuls(par), len(par), moins, plus, hors)
	}
	for _, f := range c.frags {
		t.Logf("   -- frag t=%d : evenements 36 / 37 / 0 suivis / refs suivies dans [f-20, f+2]", f)
		for _, e := range p.evs {
			garde := e.tir != nil || e.typ == 37 || e.refSuivie(c)
			if garde && e.trame >= f-tcgAvantFrag && e.trame <= f+2 {
				t.Logf("        %s", tcgLigne(e))
			}
		}
	}
}

// tcgNonNuls compte les valeurs non nulles.
func tcgNonNuls(v []int) int {
	n := 0
	for _, x := range v {
		if x > 0 {
			n++
		}
	}
	return n
}

// ancienneGardeTeteTir est l ancienne garde de longueur de la tete (le cinquieme « drapeau », bit
// 112) : l instrument `tcg_tir_continu_passe_research_test.go` mesure les tetes qu elle ecartait.
const ancienneGardeTeteTir = fireFlagsBit + 5
