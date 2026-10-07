package grammar

// canal_des_lectures_bipedes.go — LE CANAL DES LECTURES BIPEDES DE LA MARCHE DES TRAMES, ET
// L ANCRAGE QUI PASSE DERRIERE ELLE ([lecturesBipedes]).

import (
	"math"
	"math/bits"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// capteurDeLectures recoit les publications des onze crochets des lecteurs bipedes et les garde,
// chacune avec la position de lecture que `position` rend (la marche), ou -1 (l ancrage).
type capteurDeLectures struct {
	position func() int
	appels   []appelEnAttente
}

// appelEnAttente est une publication recue, pas encore attribuee a un composant.
type appelEnAttente struct {
	bit     int
	rejouer func(*Observation)
}

// noter garde une publication.
func (c *capteurDeLectures) noter(rejouer func(*Observation)) {
	bit := -1
	if c.position != nil {
		bit = c.position()
	}
	c.appels = append(c.appels, appelEnAttente{bit: bit, rejouer: rejouer})
}

// brancher pose sur `obs` les onze crochets des lecteurs bipedes ; chaque publication est gardee
// avec ses valeurs, une tranche copiee.
func (c *capteurDeLectures) brancher(obs *Observation) {
	c.brancherLesCapacites(obs)
	c.brancherLInventaire(obs)
}

// brancherLesCapacites : charges, impulsions, rangs, camouflage et grappin.
func (c *capteurDeLectures) brancherLesCapacites(obs *Observation) {
	obs.AbilityEnergyHook = func(m uint32, ch [AbilityEnergyCharges]int) {
		c.noter(func(o *Observation) {
			if o.AbilityEnergyHook != nil {
				o.AbilityEnergyHook(m, ch)
			}
		})
	}
	obs.SpartanAbilityHook = func(tag, sub, ref uint64, a bool) {
		c.noter(func(o *Observation) {
			if o.SpartanAbilityHook != nil {
				o.SpartanAbilityHook(tag, sub, ref, a)
			}
		})
	}
	obs.AbilityNonPredictedHook = func(st AbilityNonPredictedState) {
		c.noter(func(o *Observation) {
			if o.AbilityNonPredictedHook != nil {
				o.AbilityNonPredictedHook(st)
			}
		})
	}
	obs.AbilitySetHook = func(n uint64, rang, w int) {
		c.noter(func(o *Observation) {
			if o.AbilitySetHook != nil {
				o.AbilitySetHook(n, rang, w)
			}
		})
	}
	obs.CamoStateHook = func(st CamoState) {
		c.noter(func(o *Observation) {
			if o.CamoStateHook != nil {
				o.CamoStateHook(st)
			}
		})
	}
}

// brancherLInventaire : grenades, armes portees, munitions, cartouches et equipement d unite.
func (c *capteurDeLectures) brancherLInventaire(obs *Observation) {
	obs.GrenadeSetHook = func(m uint32, sel int) {
		c.noter(func(o *Observation) {
			if o.GrenadeSetHook != nil {
				o.GrenadeSetHook(m, sel)
			}
		})
	}
	obs.GrenadeCountsHook = func(n uint64, v []uint64) {
		v = slices.Clone(v)
		c.noter(func(o *Observation) {
			if o.GrenadeCountsHook != nil {
				o.GrenadeCountsHook(n, v)
			}
		})
	}
	obs.HeldWeaponHook = func(h, l uint32) {
		c.noter(func(o *Observation) {
			if o.HeldWeaponHook != nil {
				o.HeldWeaponHook(h, l)
			}
		})
	}
	obs.WeaponAmmoHook = func(a bool, m uint32, b bool, f uint32) {
		c.noter(func(o *Observation) {
			if o.WeaponAmmoHook != nil {
				o.WeaponAmmoHook(a, m, b, f)
			}
		})
	}
	obs.WeaponRoundsHook = func(n uint32) {
		c.noter(func(o *Observation) {
			if o.WeaponRoundsHook != nil {
				o.WeaponRoundsHook(n)
			}
		})
	}
	obs.UnitEquipmentHook = func(u UnitEquipmentRead) {
		c.noter(func(o *Observation) {
			if o.UnitEquipmentHook != nil {
				o.UnitEquipmentHook(u)
			}
		})
	}
}

// trameDuCanal est ce que le canal retient d une trame pour l ancrage qui passe derriere : d ou sa
// fermeture prouve sa liste, et les slots des records que la marche y a lus.
type trameDuCanal struct {
	// prouveeDes est le premier bit que la fermeture de la trame prouve. Une trame fermee dont le
	// debut de vue B a ete LU (la tete du paquet, ou la fin de sa vue A lue) prouve tout le paquet
	// (0) : la vue A lue jusqu a son terminateur precede la vue B. Une trame fermee dont le debut de
	// vue B a ete LOCALISE ([debutLocalise]) ne prouve que
	// la liste lue depuis ce debut : ce qui le precede, la marche ne l a pas lu. Une trame qui n est
	// pas fermee ne prouve rien ([rienDeProuve]).
	prouveeDes uint32
	slots      []uint32
}

// rienDeProuve est le [trameDuCanal.prouveeDes] d une trame que sa fermeture ne prouve pas.
const rienDeProuve = math.MaxUint32

// preuveDeLaTrame rend le premier bit que la fermeture de `p` prouve ([trameDuCanal.prouveeDes]).
func preuveDeLaTrame(p *lecture.Paquet) uint32 {
	switch {
	case p.Fermeture.Verdict != lecture.VerdictFerme:
		return rienDeProuve
	case debutLocalise(p.Debut):
		return p.VueB.Debut
	}
	return 0
}

// debutLocalise dit si le debut de la vue B a ete trouve par une localisation, et non lu : la
// signature du slot 123, la chaine des records NEW de tete, ou la fermeture elle-meme. La fermeture
// prouve la liste lue depuis un tel debut, pas qu aucun record ne le precede : le premier candidat
// d ou la marche ferme le paquet peut etre un record du milieu de la liste.
func debutLocalise(d lecture.DebutDeVueB) bool {
	switch d {
	case lecture.DebutParSignature, lecture.DebutParChaine, lecture.DebutParFermeture, lecture.DebutParFermetureAuBit:
		return true
	}
	return false
}

// paquetDuFlux designe un paquet delta : son chunk et son rang.
type paquetDuFlux struct{ chunk, index int }

// canalDesLecturesBipedes est le canal des trames qui recueille les lectures bipedes de la marche
// ([CanalDesTrames]), puis, a la cloture, fait passer l ancrage derriere elle et range le tout
// dans le contexte ([FilmContext.lecturesBipedes]).
type canalDesLecturesBipedes struct {
	fc  *FilmContext
	m   *MarcheDistribuee
	cap capteurDeLectures
	// utiles : les composants des huit lecteurs dans l archetype bipede ([composantsDesLecteursBipedes]).
	utiles uint64
	lu     lecturesBipedes
	trames map[paquetDuFlux]trameDuCanal
	// mortA : l instant du dead-state de chaque vie que la marche a lu, jusqu au record NEW qui la
	// recree (generation reutilisee).
	mortA map[types.LifeKey]uint64
	// lay : le decoupage d i0 du film, qui juge qu un i0 est absolu dans la region jouee ; layOK faux
	// quand il est illisible (aucune position ne se retient).
	lay   profile.I0Layout
	layOK bool
	// positionsMarche, positionsRecuperees : les records dont la position se lit, de la marche puis de
	// l ancrage derriere elle ([positionsBipedes]).
	positionsMarche, positionsRecuperees positionsBipedes
	// bande : la bande bipede du contexte ([FilmContext.BipedSlots]), la population des positions.
	bande SlotBand
}

// nouveauCanalDesLecturesBipedes prepare le canal des lectures bipedes du film `fc`. Les
// generations vivantes du film se relevent ICI, avant la marche : leur releve marche les
// images-cles du film, ce qui ne se fait pas au milieu de la marche des trames, ou chaque trame
// date ensuite la sienne ([FilmContext.GenerationsVivantesA]).
func nouveauCanalDesLecturesBipedes(fc *FilmContext) *canalDesLecturesBipedes {
	c := &canalDesLecturesBipedes{fc: fc, trames: map[paquetDuFlux]trameDuCanal{}, mortA: map[types.LifeKey]uint64{}}
	if arch, err := fc.bipedArchetype(); err == nil {
		c.utiles = composantsDesLecteursBipedes(arch)
	}
	if lay, err := fc.I0Layout(); err == nil {
		c.lay, c.layOK = lay, true
		c.bande = fc.BipedSlots()
	}
	fc.GenerationsVivantesA(0)
	return c
}

// Interets : les composants des huit lecteurs, sur le bipede, dans les trames ; et la position (i0),
// que le canal designe au lecteur de position ([positionsDuContexte]).
func (c *canalDesLecturesBipedes) Interets() []Interet {
	arch, err := c.fc.bipedArchetype()
	if err != nil {
		return nil
	}
	var out []Interet
	if c.layOK {
		out = append(out, Interet{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: arch.component(indexDeLaPosition)})
	}
	for m := c.utiles; m != 0; m &= m - 1 {
		id := bits.TrailingZeros64(m)
		out = append(out, Interet{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: arch.component(id)})
	}
	return out
}

// Brancher pose les onze crochets ; chaque publication est datee de la position de lecture de la
// marche.
func (c *canalDesLecturesBipedes) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.m = m
	c.cap.position = m.positionDeLecture
	c.cap.brancher(obs)
}

// Trame attribue les publications de la trame aux composants de ses records bipedes delta, et
// retient sa fermeture et les slots que la marche y a lus.
func (c *canalDesLecturesBipedes) Trame(p *lecture.Paquet) {
	appels := c.cap.appels
	c.cap.appels = c.cap.appels[:0]
	recs, lus := c.m.recordsDeLaTrame()
	if !lus || !c.m.attribuable() {
		// Une trame que la marche n a pas lue, ou qu elle n a pas marchee par classes de vue (sans
		// position de lecture) : l ancrage la lira entiere.
		c.lu.horsRecord += len(appels)
		c.trames[paquetDuFlux{p.Chunk, p.Index}] = trameDuCanal{prouveeDes: rienDeProuve}
		return
	}
	c.recueillir(p, recs, appels, c.fc.GenerationsVivantesA(p.TS))
}

// recueillir retient, des records `recs` que la marche a lus dans la trame `p`, les records bipedes
// delta d un corps vivant a l instant du filtre `gens`, chacun avec les publications d `appels`
// qu il porte ; et retient la trame pour l ancrage qui passe derriere.
func (c *canalDesLecturesBipedes) recueillir(p *lecture.Paquet, recs []FrameRecord, appels []appelEnAttente,
	gens *GenerationsVivantes) {
	t := trameDuCanal{prouveeDes: preuveDeLaTrame(p)}
	k := 0
	for i := range recs {
		r := &recs[i]
		t.slots = append(t.slots, r.Slot)
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		vie := types.LifeKey{Slot: r.Slot, Gen: r.ID >> 30}
		if r.Type == recNew {
			delete(c.mortA, vie)
			continue
		}
		if r.Type != recDelta {
			continue
		}
		c.lu.examines++
		rb := recordDeLaMarche(p, r)
		k = attribuerLesAppels(&rb, r, appels, k, &c.lu.horsRecord)
		if !gens.Accepte(vie) {
			// La garde des generations vivantes datees, celle de l ancrage (lot R2-bis) : un corps
			// n est pas lu avant son record de creation, ni sous une generation que le film ne
			// connait pas.
			c.lu.generationsRefusees++
			continue
		}
		if _, deja := c.mortA[vie]; !deja && r.Trace.Dead != nil && r.Trace.Dead.Mort {
			c.mortA[vie] = p.TS
		}
		if _, mort := c.mortA[vie]; mort {
			c.lu.corpsMorts++
			continue
		}
		c.noterLaPosition(p, r, &rb)
		if rb.masque&c.utiles != 0 {
			c.lu.records = append(c.lu.records, rb)
		}
	}
	c.lu.horsRecord += len(appels) - k
	c.trames[paquetDuFlux{p.Chunk, p.Index}] = t
}

// Clore fait passer l ancrage derriere la marche, range les records dans l ordre du flux et les
// donne au contexte, avec le compte des records recuperes : tous ceux que l ancrage rend, pour un des
// huit lecteurs ou pour leur seule position.
func (c *canalDesLecturesBipedes) Clore(BilanDeMarche) {
	rec := c.recuperer()
	c.lu.recuperes = len(c.positionsRecuperees.records)
	c.lu.records = append(c.lu.records, rec...)
	rangerDansLeFlux(c.lu.records, c.fc.ChunkNumbers())
	c.lu.positions = fondrePositions(&c.positionsMarche, &c.positionsRecuperees, c.fc.ChunkNumbers())
	c.lu.trames = c.trames
	c.fc.recup.lectures = &c.lu
	c.fc.NoterReplis(ComptesDesReplis{AncragesBipedesApresLaMarche: c.lu.recuperes})
}

// recordDeLaMarche rend le record bipede delta `r` de la trame `p`, sans ses publications.
func recordDeLaMarche(p *lecture.Paquet, r *FrameRecord) recordBipedeLu {
	rb := recordBipedeLu{Slot: r.Slot, Gen: r.ID >> 30, Chunk: p.Chunk, I0: r.HeaderBit, masque: r.Trace.Mask,
		Packet: FilmPacket{Index: p.Index, Type: p.Type, TimestampUS: p.TS}, arret: -1}
	for _, comp := range r.Trace.Comps {
		switch {
		case !comp.Ported:
			rb.arret = comp.Index
			return rb
		case comp.Index == 0:
			rb.I0 = comp.StartBit
		case comp.Index < 64:
			rb.atteints |= 1 << uint(comp.Index)
		}
	}
	return rb
}

// attribuerLesAppels donne a `rb` les publications de `appels`, a partir du rang `k`, que l etendue
// d un de ses composants lus porte : un composant commence a son bit de depart et finit au depart
// du suivant, ou a la fin du record (pour le dernier, et pour celui sur lequel la lecture s est
// arretee) ; une publication tombe APRES le premier bit qu il a lu. Les publications qui precedent
// le record ne sont a personne. Rend le rang suivant.
func attribuerLesAppels(rb *recordBipedeLu, r *FrameRecord, appels []appelEnAttente, k int, horsRecord *int) int {
	comps := r.Trace.Comps
	for j := range comps {
		debut, fin := comps[j].StartBit, r.FinBit
		if comps[j].Ported && j+1 < len(comps) {
			fin = comps[j+1].StartBit
		}
		for ; k < len(appels) && appels[k].bit <= debut; k++ {
			*horsRecord++
		}
		for ; k < len(appels) && appels[k].bit <= fin; k++ {
			rb.appels = append(rb.appels, appelDeComposant{composant: comps[j].Index, rejouer: appels[k].rejouer})
		}
		if !comps[j].Ported {
			break
		}
	}
	return k
}

// recuperer fait passer l ancrage d en-tete bipede derriere la marche : il ne rend que les records
// d un slot qu elle n a pas lu dans le paquet, hors de ce que la fermeture de la trame prouve
// ([rendParLAncrage]).
func (c *canalDesLecturesBipedes) recuperer() []recordBipedeLu {
	lay, err := c.fc.I0Layout()
	if err != nil {
		return nil
	}
	arch, err := c.fc.bipedArchetype()
	if err != nil {
		return nil
	}
	obs := NouvelleObservation()
	var capt capteurDeLectures
	capt.brancher(obs)
	g := grammaireRecord{lay: lay, arch: arch, prof: c.fc.ProfilDeBalayage(), obs: obs}
	var out []recordBipedeLu
	c.fc.parcourirLesAncresBipedes(func(r deltaBipedRecord) {
		if t, vu := c.trames[paquetDuFlux{r.Chunk, r.Packet.Index}]; !rendParLAncrage(t, vu, r.Slot, r.I0) {
			return
		}
		if mort, connue := c.mortA[r.Vie()]; connue && r.Packet.TimestampUS >= mort {
			c.lu.corpsMorts++
			return
		}
		c.lu.examines++
		masque := masqueDesIndex(r.Mask)
		c.positionsRecuperees.noter(r.Chunk, r.Packet, positionLue{i0: uint32(r.I0), slot: r.Slot, gen: uint8(r.Gen), //nolint:gosec // position dans un payload, generation sur 2 bits
			recupere: true, masque: masque})
		if masque&c.utiles == 0 {
			return // sa position seule : aucun des huit lecteurs n y lit rien
		}
		rb := recordBipedeLu{Slot: r.Slot, Gen: r.Gen, Chunk: r.Chunk, Packet: r.Packet, I0: r.I0,
			masque: masque, arret: -1, Recupere: true}
		capt.appels = capt.appels[:0]
		suivant := 1 // rang, dans le masque, du composant que la marche lit ensuite (i0 est le 0)
		walkRecordComponents(r.Payload, r.I0, r.Total, r.Mask, g, func(id int) bool {
			for _, a := range capt.appels {
				rb.appels = append(rb.appels, appelDeComposant{composant: id, rejouer: a.rejouer})
			}
			capt.appels = capt.appels[:0]
			rb.atteints |= 1 << uint(id)
			suivant++
			return true
		})
		if len(capt.appels) > 0 && suivant < len(r.Mask) {
			// La marche s est arretee sur le composant suivant APRES qu il a publie.
			rb.arret = r.Mask[suivant]
			for _, a := range capt.appels {
				rb.appels = append(rb.appels, appelDeComposant{composant: rb.arret, rejouer: a.rejouer})
			}
		}
		out = append(out, rb)
	})
	return out
}

// positionDeLecture rend la position du lecteur de la marche dans la trame en cours ; -1 hors d une
// marche par classes de vue.
func (m *MarcheDistribuee) positionDeLecture() int {
	if m.marche == nil || m.marche.lecteur == nil {
		return -1
	}
	return m.marche.lecteur.BitPos()
}

// attribuable dit si la trame en cours a ete marchee par classes de vue : ses publications portent
// alors la position de lecture.
func (m *MarcheDistribuee) attribuable() bool {
	return m.marche != nil && m.marche.trame.parRangs
}

// rendParLAncrage dit si l ancrage rend un record du slot `slot`, dont le composant i0 commence au
// bit `i0`, dans une trame que le canal a vue (`vu`) telle que `t` : une trame que la marche n a pas
// rendue ; sinon un slot dont la marche n a lu aucun record dans la trame, hors de ce que sa
// fermeture prouve.
func rendParLAncrage(t trameDuCanal, vu bool, slot uint32, i0 int) bool {
	return !vu || (!slices.Contains(t.slots, slot) && int64(i0) < int64(t.prouveeDes))
}

// noterLaPosition retient la position du record `rb` que la marche a lu (`r`) quand son slot est de
// la bande bipede du contexte, que son i0 a ete traverse et qu il est absolu dans la region jouee
// ([i0AbsoluDeLaRegion]). LA BANDE RESTE LA POPULATION DES POSITIONS, celle de l ancrage : la marche
// lit aussi des corps qu aucune image-cle ne porte — les corps poses en fin de match pour la scene des
// vainqueurs, un joueur ne au dernier chunk d une bobine —, que les traces ne publiaient pas.
func (c *canalDesLecturesBipedes) noterLaPosition(p *lecture.Paquet, r *FrameRecord, rb *recordBipedeLu) {
	if !c.layOK || len(r.Trace.Comps) == 0 || !c.bande.Has(rb.Slot) {
		return
	}
	i0 := r.Trace.Comps[0]
	if i0.Index != indexDeLaPosition || !i0.Ported || !i0AbsoluDeLaRegion(p.Payload, i0.StartBit, c.lay) {
		return
	}
	c.positionsMarche.noter(rb.Chunk, rb.Packet, positionLue{i0: uint32(i0.StartBit), slot: rb.Slot, //nolint:gosec // position dans un payload
		gen: uint8(rb.Gen), masque: rb.masque}) //nolint:gosec // generation sur 2 bits
}
