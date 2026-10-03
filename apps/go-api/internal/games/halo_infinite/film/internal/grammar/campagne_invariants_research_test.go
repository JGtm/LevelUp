//go:build research

package grammar

// campagne_invariants_research_test.go — MESURES CIBLEES (etape 4) : LES INVARIANTS DE L ECRIVAIN,
// paquet par paquet, sur ce que la marche a deja lu.
//
//	ordre de la vue B (T1-2, C4)     NEW*, DELTA*, DEL*, slots strictement croissants par groupe ;
//	masques (T2-4)                   aucun bit >= nombre de composants, epars croissant, dense > 7 ;
//	vue C (T5-1, T5-2, T5-4)         kind 3, index croissants, <= 32 entrees, en-tete cdc04, code 63 ;
//	bourrage (T8-C2, T8-C3, T8-C4)   terminateur et corps dans la queue minimale ou au-dela ;
//	tetes (T1-6)                     tete d un DELTA contre la generation vivante du bloc ;
//	vehicules (T6-C1, T6-C3)         i30 / i33 / i34 annonces, par chassis.

import (
	"fmt"
	"math/bits"
	"strings"
)

// invariants range un paquet dans les tables d invariants.
func (c *cmCollecteur) invariants(p *cmPaquet) {
	if p.d.DebutVueB < 0 {
		return
	}
	c.contre = c.contre[:0]
	c.ordre(p)
	c.masques(p)
	c.vueC(p)
	if p.d.Fermee {
		cle := "aucun invariant contredit"
		if len(c.contre) > 0 {
			cle = "contredit : " + strings.Join(c.contre, " + ")
		}
		c.t.add("fermes_contredits", cmJoindre(cle, fmt.Sprintf("%d record(s)", min(len(p.recs), 3))), compteDuPaquet(p))
	}
	c.bourrage(p)
	c.tetes(p)
	if p.chaine != nil {
		c.t.add("chaines_de_tete", cmJoindre(violationDOrdre(p.chaine, true), classeDePaquetCM(p)), compteDuPaquet(p))
	}
}

// violationDOrdre rend la premiere violation de l ordre de l ecrivain dans une suite d en-tetes,
// « ordre respecte » sinon. `tete` : chaine de tete (aucun DEL admis).
func violationDOrdre(h []cmEnTete, tete bool) string {
	phase := 0 // 0 NEW, 1 DELTA, 2 DEL
	var dernier [3]int64
	dernier[0], dernier[1], dernier[2] = -1, -1, -1
	for _, x := range h {
		var ph int
		switch x.typ {
		case recNew:
			ph = 0
		case recDelta:
			ph = 1
		case recDel:
			ph = 2
			if tete {
				return "DEL dans la chaine de tete"
			}
		default:
			continue
		}
		if ph < phase {
			return fmt.Sprintf("%s apres %s", nomDeTypeDeRecord(x.typ), nomDePhase(phase))
		}
		if ph > phase {
			phase = ph
		}
		if int64(x.slot) <= dernier[ph] {
			return "slot non croissant (" + nomDeTypeDeRecord(x.typ) + ")"
		}
		dernier[ph] = int64(x.slot)
	}
	return "ordre respecte"
}

func nomDePhase(ph int) string { return [3]string{"NEW", "DELTA", "DEL"}[ph] }

// ordre : T1-2 / C4 sur les records de la vue B.
func (c *cmCollecteur) ordre(p *cmPaquet) {
	h := make([]cmEnTete, 0, len(p.recs))
	for _, r := range p.recs {
		h = append(h, cmEnTete{typ: r.Type, slot: r.Slot})
	}
	v := violationDOrdre(h, false)
	if v != "ordre respecte" {
		c.contre = append(c.contre, "ordre")
	}
	if v != "ordre respecte" && !p.d.Fermee && len(h) > 0 {
		if violationDOrdre(h[:len(h)-1], false) == "ordre respecte" {
			v += " · sur le dernier record"
		} else {
			v += " · avant le dernier record"
		}
	}
	c.t.add("ordre_vue_b", cmJoindre(classeDePaquetCM(p), p.d.Sortie.String(), v), compteDuPaquet(p))
}

// classeDeTaille classe un archetype par son nombre de composants.
func classeDeTaille(n int) string {
	switch {
	case n <= 10:
		return "n<=10"
	case n <= 47:
		return "n 11-47"
	case n <= 63:
		return "n 48-63"
	}
	return "n=64+"
}

// cmMasque est un masque relu.
type cmMasque struct {
	dense bool
	val   uint64
	idx   []int
}

func cmLireMasque(br *Lecteur) cmMasque {
	if br.ReadBit() {
		return cmMasque{dense: true, val: br.ReadBits(64)}
	}
	m := cmMasque{}
	n := int(br.ReadBits(3))
	for i := 0; i < n; i++ {
		x := int(br.ReadBits(6))
		m.idx = append(m.idx, x)
		m.val |= uint64(1) << uint(x&63)
	}
	return m
}

// violationDeMasque rend la premiere violation de l ecrivain, « sain » sinon.
func violationDeMasque(m cmMasque, n int) string {
	switch {
	case n < 64 && m.val>>uint(n) != 0:
		return "bit hors archetype"
	case m.dense && bits.OnesCount64(m.val) <= 7:
		return "dense <= 7 bits"
	}
	for i := 1; i < len(m.idx); i++ {
		if m.idx[i] <= m.idx[i-1] {
			return "epars non croissant"
		}
	}
	return "sain"
}

// cmPrefixeNeuf est le prefixe de [TraverseEntity] (R(6), etat par defaut, porte) : il laisse le
// lecteur sur le masque.
func cmPrefixeNeuf(br *Lecteur, defaultStateBits int) uint32 {
	ti := uint32(br.ReadBits(6))
	if ti >= objectArchetypeCount {
		return ti
	}
	switch fn, ok := defaultStateDeserByTI[ti]; {
	case ti == bipedDefaultStateTypeIndex:
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
	case ti == ProjectileTypeIndex && br.p.Grammaire.DeserEtatParArchetype:
		consumeDefaultStateTI41(br, true)
	case ti == archetypeProprieteGeree && br.p.Grammaire.DeserEtatParArchetype:
		consumeDefaultStateTI13RecordNeuf(br)
	case ok && br.p.Grammaire.DeserEtatParArchetype:
		fn(br)
	default:
		br.Skip(defaultStateBits)
	}
	br.ReadBit()
	return ti
}

// masques : T2-4 — relit le masque de chaque record de la vue B, dans l ordre, depuis le debut de
// la vue. S arrete sur un DELTA dont l archetype a ete INFERE (sa fin n est pas connue).
func (c *cmCollecteur) masques(p *cmPaquet) {
	cfg := c.f.cfg
	obs := NouvelleObservation()
	var mpp uint32
	obs.MppHook = func(f MPPField, v uint64, _ bool) {
		if f == MPPWord32 {
			mpp = uint32(v) //nolint:gosec // mot de 32 bits
		}
	}
	cfgT := cfg
	cfgT.Obs = obs
	br := LecteurSur(p.pay)
	br.poserCadre(cfgT)
	defer obs.neutraliserCaptures()()
	pos := p.d.DebutVueB
	premiere := -1
boucle:
	for k, r := range p.recs {
		br.SetBitPos(pos)
		if cfg.HasExtraFields {
			br.Skip(32)
		}
		typ := readRecordType(br)
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		if typ != r.Type || id != r.ID {
			c.t.un("masques_relecture", "en-tete discordant : arret")
			break boucle
		}
		var m cmMasque
		ti := r.TypeIndex
		switch r.Type {
		case recDel:
			br.Skip(32)
			pos = br.BitPos()
			continue
		case recNew:
			mpp = 0
			ti = cmPrefixeNeuf(br, cfg.NewDefaultStateBits)
			if ti >= objectArchetypeCount {
				c.t.un("masques_relecture", "NEW ti >= 50 : arret")
				break boucle
			}
			m = cmLireMasque(br)
			if ti == VehicleTypeIndex && mpp != 0 {
				c.chassisLu[r.Slot] = mpp
			}
		case recDelta:
			if r.Trace.EndBit == 0 && len(r.Trace.Comps) == 0 {
				c.t.un("masques_relecture", "DELTA infere : arret")
				break boucle
			}
			if br.ReadBit() {
				br.Skip(7)
			}
			m = cmLireMasque(br)
		}
		if m.val != r.Trace.Mask {
			c.t.un("masques_relecture", "masque discordant : arret")
			break boucle
		}
		c.t.un("masques_relecture", "relu")
		arch, ok := c.f.reg.Archetype(int(ti))
		if !ok {
			break boucle
		}
		v := violationDeMasque(m, len(arch.Components))
		if v != "sain" && premiere < 0 {
			premiere = k
		}
		c.t.add("masques", cmJoindre(c.populationDuRecord(p, k), nomDeTypeDeRecord(r.Type),
			classeDeTaille(len(arch.Components)), formeDeMasque(m), v), cmCompte{n: 1})
		if v == "bit hors archetype" && p.d.Fermee {
			n := len(arch.Components)
			c.t.un("masques_hors_archetype_fermes", fmt.Sprintf("%s ti=%d n=%d bits au-dela=%#x", nomDeTypeDeRecord(r.Type), ti, n, m.val>>uint(n)))
			c.t.un("masques_hors_archetype_fermes_contexte", cmJoindre(nomDeTypeDeRecord(r.Type), p.d.Sortie.String(), fmt.Sprintf("record %d sur %d", k+1, len(p.recs)), formeDeMasque(m)))
		}
		c.vehicule(p, r, ti, arch, m)
		if r.DesyncAt >= 0 {
			break boucle
		}
		pos = r.Trace.EndBit
	}
	if premiere >= 0 {
		c.contre = append(c.contre, "masque")
	}
	if !p.d.Fermee && p.d.Cause == CauseHorsCadre {
		v := "aucune violation relue"
		switch {
		case premiere >= 0 && premiere == len(p.recs)-1:
			v = "violation au dernier record"
		case premiere >= 0:
			v = "violation avant le dernier record"
		}
		c.t.add("masques_hors_cadre", cmJoindre(p.d.Sortie.String(), v), compteDuPaquet(p))
	}
}

// populationDuRecord : la population d un record pour les taux de T2-4.
func (c *cmCollecteur) populationDuRecord(p *cmPaquet, k int) string {
	switch {
	case p.d.Fermee:
		return "paquet ferme"
	case p.d.Cause == CauseHorsCadre && k == len(p.recs)-1:
		return "hors cadre, dernier record, sortie " + p.d.Sortie.String()
	case p.d.Cause == CauseHorsCadre:
		return "hors cadre, record anterieur"
	}
	return "autre non ferme"
}

// vehicule : T6-C1 / T6-C3 — les annonces de i30, i33, i34 d un record ti=40, par chassis.
func (c *cmCollecteur) vehicule(p *cmPaquet, r FrameRecord, ti uint32, arch Archetype, m cmMasque) {
	if ti != VehicleTypeIndex {
		return
	}
	complet := "masque partiel"
	if n := len(arch.Components); n < 64 && m.val == (uint64(1)<<uint(n))-1 {
		complet = "masque complet"
	}
	for i, nom := range arch.Components {
		var court string
		switch CleComposant(int(ti), nom).Nom {
		case "vehicle-auto-turret-triggers":
			court = "i30 auto-turret-triggers"
		case "vehicle-type-state":
			court = "i33 type-state"
		case "vehicle-type-physics":
			court = "i34 type-physics"
		default:
			continue
		}
		if m.val&(uint64(1)<<uint(i&63)) == 0 {
			continue
		}
		c.t.add("t6_annonces", cmJoindre(court, nomDeTypeDeRecord(r.Type), c.chassisDe(p, r.Slot), complet),
			compteDuPaquet(p))
	}
}

// chassisDe rend le chassis (MPPWord32) de la vie courante d un slot : la DERNIERE creation lue
// (balayeur de creations) avant le paquet, sinon le dernier NEW ti=40 relu par la marche.
func (c *cmCollecteur) chassisDe(p *cmPaquet, slot uint32) string {
	var best uint32
	ok := false
	for _, x := range c.creations[slot] {
		if x.Chunk < p.d.Chunk || (x.Chunk == p.d.Chunk && x.PacketIndex <= p.d.Index) {
			best, ok = uint32(x.MPPVal[MPPWord32]), true //nolint:gosec // mot de 32 bits
		}
	}
	if !ok {
		if v, lu := c.chassisLu[slot]; lu {
			best, ok = v, true
		}
	}
	if !ok {
		return "chassis inconnu"
	}
	return fmt.Sprintf("%08x", best)
}

// vueC : T5-1 / T5-2 / T5-4.
func (c *cmCollecteur) vueC(p *cmPaquet) {
	if strings.HasPrefix(p.d.Cause, "vue C : ") && p.d.Cause != CauseHorsCadre {
		c.t.add("vuec_arrets_x_sortie", cmJoindre(p.d.Cause, p.d.Sortie.String()), compteDuPaquet(p))
	}
	if !p.d.VueCAtteinte {
		return
	}
	f := p.d.VueC
	pop := classeDePaquetCM(p)
	if f.Vide {
		c.t.add("vuec_verdict", cmJoindre(pop, "vide : sans entree jugeable"), compteDuPaquet(p))
		return
	}
	crit := map[string]bool{}
	for _, k := range f.Kinds {
		if k == 3 {
			crit["kind 3"] = true
		}
		if k == 1 || k == 2 {
			crit["kind 1 ou 2"] = true
		}
	}
	if len(f.Entrees) > 32 {
		crit["plus de 32 entrees"] = true
	}
	for i, e := range f.Entrees {
		if i > 0 && e.Index <= f.Entrees[i-1].Index {
			crit["index non croissant"] = true
		}
		if e.Champs.Cdc04 != ChampDeControleAbsent {
			crit["en-tete cdc04 pose"] = true
		}
		if e.Bloc && (e.Champs.Analogique[0] == 63 || e.Champs.Analogique[1] == 63) {
			crit["code analogique 63"] = true
		}
		ab := "bloc 0x68"
		if !e.Bloc {
			ab = "a=b=0"
		}
		c.t.un("vuec_entrees", cmJoindre(pop, ab))
	}
	verdict := "non contredit"
	for _, nom := range []string{"kind 1 ou 2", "kind 3", "index non croissant", "plus de 32 entrees",
		"en-tete cdc04 pose", "code analogique 63"} {
		if crit[nom] {
			c.t.add("vuec_criteres", cmJoindre(pop, nom), compteDuPaquet(p))
			if verdict == "non contredit" {
				verdict = "impossible : " + nom
			}
		}
	}
	if verdict != "non contredit" {
		c.contre = append(c.contre, "vue C")
	}
	c.t.add("vuec_verdict", cmJoindre(pop, verdict), compteDuPaquet(p))
}

// bourrage : T8-C2 / T8-C4.
func (c *cmCollecteur) bourrage(p *cmPaquet) {
	bitsP := p.d.Bits
	k := compteDuPaquet(p)
	if p.d.Sortie == SortieVueBTerminateur {
		cl := "terminateur dans le payload, place pour la vue C"
		switch {
		case p.d.FinVueB > bitsP:
			cl = "terminateur lu au-dela du payload"
		case p.d.FinVueB > bitsP-1:
			cl = "terminateur sans place pour la vue C"
		}
		c.t.add("t8_terminateurs", cl, k)
	}
	maxPos, ou := p.d.FinVueB, "terminateur ou en-tete de la vue B"
	for _, r := range p.recs {
		if r.Trace.EndBit > maxPos {
			maxPos, ou = r.Trace.EndBit, "corps de record de la vue B"
		}
		if r.Trace.EndBit == 0 {
			continue
		}
		switch {
		case r.Trace.EndBit > bitsP:
			c.t.add("t8_records", cmJoindre(nomDeTypeDeRecord(r.Type), "deborde du payload"), cmCompte{n: 1, fermes: k.fermes})
		case r.Trace.EndBit > bitsP-c.queueMin:
			c.t.add("t8_records", cmJoindre(nomDeTypeDeRecord(r.Type), "finit dans la queue minimale"), cmCompte{n: 1, fermes: k.fermes})
		}
	}
	if p.d.VueCAtteinte && p.d.Curseur > maxPos {
		maxPos, ou = p.d.Curseur, "vue C"
	}
	if maxPos > bitsP {
		c.t.add("t8_depassement", ou, k)
	}
}

// tetes : T1-6 — tete d un DELTA contre la generation que le bloc du chunk donne a son slot vivant.
func (c *cmCollecteur) tetes(p *cmPaquet) {
	k := compteDuPaquet(p)
	for _, r := range p.recs {
		if r.Type != recDelta {
			continue
		}
		cur, ok := c.b.entree(c.chunk, r.Slot)
		if !ok || !cur.Vivante() {
			continue
		}
		tete := uint8(r.ID >> 30) //nolint:gosec // deux bits
		cl := "tete = generation du bloc"
		if tete != cur.Gen {
			cl = "tete != generation : inexpliquee"
			if nx, okN := c.b.entree(c.b.suivant[c.chunk], r.Slot); okN && cmAlloueSous(nx, tete) {
				cl = "tete != generation : reallocation au bloc suivant"
			}
		}
		c.t.add("t1_6_tetes", cmJoindre(classeDePaquetCM(p), cl), cmCompte{n: 1, fermes: k.fermes})
	}
}

// formeDeMasque : epars, dense, ou dense « tout a un » (le masque complet).
func formeDeMasque(m cmMasque) string {
	switch {
	case !m.dense:
		return "epars"
	case m.val == ^uint64(0):
		return "dense tout a un"
	}
	return "dense"
}
