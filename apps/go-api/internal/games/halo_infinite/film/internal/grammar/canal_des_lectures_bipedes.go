package grammar

// canal_des_lectures_bipedes.go — LE CANAL DES LECTURES BIPEDES DE LA MARCHE DES TRAMES, ET
// L ANCRAGE QUI PASSE DERRIERE ELLE ([lecturesBipedes]).

import (
	"math/bits"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
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

// trameDuCanal est ce que le canal retient d une trame pour l ancrage qui passe derriere : la
// fermeture prouvee, et les slots des records que la marche y a lus.
type trameDuCanal struct {
	fermee bool
	slots  []uint32
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
}

// nouveauCanalDesLecturesBipedes prepare le canal des lectures bipedes du film `fc`.
func nouveauCanalDesLecturesBipedes(fc *FilmContext) *canalDesLecturesBipedes {
	c := &canalDesLecturesBipedes{fc: fc, trames: map[paquetDuFlux]trameDuCanal{}, mortA: map[types.LifeKey]uint64{}}
	if arch, err := fc.bipedArchetype(); err == nil {
		c.utiles = composantsDesLecteursBipedes(arch)
	}
	return c
}

// Interets : les composants des huit lecteurs, sur le bipede, dans les trames.
func (c *canalDesLecturesBipedes) Interets() []Interet {
	arch, err := c.fc.bipedArchetype()
	if err != nil {
		return nil
	}
	var out []Interet
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
	t := trameDuCanal{}
	if !lus || !c.m.attribuable() {
		// Une trame que la marche n a pas lue, ou qu elle n a pas marchee par classes de vue (sans
		// position de lecture) : l ancrage la lira entiere.
		c.lu.horsRecord += len(appels)
		c.trames[paquetDuFlux{p.Chunk, p.Index}] = t
		return
	}
	t.fermee = p.Fermeture.Verdict == lecture.VerdictFerme
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
		if _, deja := c.mortA[vie]; !deja && r.Trace.Dead != nil {
			c.mortA[vie] = p.TS
		}
		if _, mort := c.mortA[vie]; mort {
			c.lu.corpsMorts++
			continue
		}
		if rb.masque&c.utiles != 0 {
			c.lu.records = append(c.lu.records, rb)
		}
	}
	c.lu.horsRecord += len(appels) - k
	c.trames[paquetDuFlux{p.Chunk, p.Index}] = t
}

// Clore fait passer l ancrage derriere la marche, range les records dans l ordre du flux et les
// donne au contexte, avec le compte des records recuperes.
func (c *canalDesLecturesBipedes) Clore(BilanDeMarche) {
	rec := c.recuperer()
	c.lu.recuperes = len(rec)
	c.lu.records = append(c.lu.records, rec...)
	rangerDansLeFlux(c.lu.records, c.fc.ChunkNumbers())
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
// qu elle n a pas lus, dans une trame qu elle n a pas fermee.
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
		if t, vu := c.trames[paquetDuFlux{r.Chunk, r.Packet.Index}]; !rendParLAncrage(t, vu, r.Slot) {
			return
		}
		if mort, connue := c.mortA[r.Vie()]; connue && r.Packet.TimestampUS >= mort {
			c.lu.corpsMorts++
			return
		}
		c.lu.examines++
		rb := recordBipedeLu{Slot: r.Slot, Gen: r.Gen, Chunk: r.Chunk, Packet: r.Packet, I0: r.I0,
			masque: masqueDesIndex(r.Mask), arret: -1, Recupere: true}
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
		if rb.masque&c.utiles != 0 {
			out = append(out, rb)
		}
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

// rendParLAncrage dit si l ancrage rend un record du slot `slot` dans une trame que le canal a vue
// (`vu`) telle que `t` : une trame que la marche n a pas rendue, ou une trame qu elle n a pas fermee
// et ou elle n a lu aucun record de ce slot.
func rendParLAncrage(t trameDuCanal, vu bool, slot uint32) bool {
	return !vu || (!t.fermee && !slices.Contains(t.slots, slot))
}
