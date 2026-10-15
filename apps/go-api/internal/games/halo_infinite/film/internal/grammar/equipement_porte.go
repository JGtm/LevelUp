package grammar

// equipement_porte.go — L EQUIPEMENT PORTE, LU PAR LA MARCHE DES TRAMES (lot « equipement porte » du
// plan d ajustements du 2026-10-14).
//
// # CE QUE LE FILM ECRIT
//
// Quand un joueur prend un equipement, le jeu cree une entite ti=37 (record NEW) dans le MEME paquet
// que l annonce i48 du porteur ; son `object-parent-state` (i10) ouvre la porte « attache » et nomme
// le parent (categorie 1 de `FUN_1406d3140`, base 0x200 : [parentHandleBase]). Pendant le port,
// l objet n emet un record DELTA que lorsque son etat change ; quatre composants le disent, rendus
// par la couche de capture ([EquipementLu]) : l energie (i24), l activation (i21, porte a 1), le
// signal de controle (i22) et la pile des poignees suivies (i28). La suppression (DEL) clot le port.
//
// # CE QUE CE CANAL REND, ET CE QU IL NE DIT PAS
//
// Les lectures BRUTES de ces records, dans l ordre du flux, pour les seuls objets crees attaches :
// aucune interpretation (pose, consommation) — elle appartient au rejeu, qui joint la famille par
// l annonce i48 du meme paquet. Une trame dont la marche n a pas localise la liste ne rend rien : ce
// qu elle portait n est pas lu.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// equipementPorteLu est ce que le canal a lu d un film, range dans le contexte a la cloture.
type equipementPorteLu struct {
	lectures []types.CarriedEquipmentRead
	st       types.CarriedEquipmentStats
}

// vieDObjetPorte est un objet porte suivi : sa generation et son porteur.
type vieDObjetPorte struct {
	gen, porteur uint32
}

// canalDeLEquipementPorte lit, dans les trames que la marche rend, les records des objets ti=37
// crees attaches.
type canalDeLEquipementPorte struct {
	fc   *FilmContext
	reg  *Registry
	m    *MarcheDistribuee
	vies map[uint32]vieDObjetPorte
	lu   equipementPorteLu
}

// composantsDeLEquipementPorte rend les composants de ti=37 que le canal interprete.
func composantsDeLEquipementPorte() []string {
	return []string{compObjectParentState, compEquipmentActivated,
		compEquipmentControlSignal, compEquipmentEnergy, compEquipmentTrackedStack}
}

// nouveauCanalDeLEquipementPorte prepare le canal de l equipement porte du film `fc`.
func nouveauCanalDeLEquipementPorte(fc *FilmContext) *canalDeLEquipementPorte {
	reg, err := fc.Registry()
	if err != nil {
		reg = nil
	}
	return &canalDeLEquipementPorte{fc: fc, reg: reg, vies: map[uint32]vieDObjetPorte{}}
}

// Interets : les cinq composants de ti=37 que le registre du film declare, dans les trames.
func (c *canalDeLEquipementPorte) Interets() []Interet {
	if c.reg == nil {
		return nil
	}
	arch, ok := c.reg.Archetype(EquipmentTypeIndex)
	if !ok {
		return nil
	}
	var out []Interet
	for _, n := range composantsDeLEquipementPorte() {
		if len(arch.indicesOf(n)) > 0 {
			out = append(out, Interet{Phase: PhaseTrames, TI: EquipmentTypeIndex, Composant: n})
		}
	}
	return out
}

func (c *canalDeLEquipementPorte) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Clore range ce que le canal a lu dans le contexte.
func (c *canalDeLEquipementPorte) Clore(BilanDeMarche) { c.fc.recup.equipementPorte = &c.lu }

// Trame lit les records d objets portes de la trame `p`.
func (c *canalDeLEquipementPorte) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		return
	}
	for i := range recs {
		r := &recs[i]
		switch r.Type {
		case recNew:
			if r.TypeIndex == EquipmentTypeIndex && r.DesyncAt == -1 {
				c.creation(p.TS, r)
			}
		case recDelta:
			if r.TypeIndex == EquipmentTypeIndex {
				c.delta(p.TS, r)
			}
		case recDel:
			c.suppression(p.TS, r)
		}
	}
}

// creation suit l objet d un record NEW ti=37 dont i10 ouvre la porte « attache ».
func (c *canalDeLEquipementPorte) creation(ts uint64, r *FrameRecord) {
	c.lu.st.Created++
	parent, ok := parentDuRecord(r)
	if !ok || !parent.Attached {
		delete(c.vies, r.Slot)
		return
	}
	c.lu.st.Attached++
	v := vieDObjetPorte{gen: r.ID >> 30, porteur: (parent.Quant16 & parentHandleValueMask) + parentHandleBase}
	c.vies[r.Slot] = v
	l := c.lecture(ts, r, v, types.CarriedNew)
	c.lu.lectures = append(c.lu.lectures, l)
}

// delta rend la lecture d un record DELTA d un objet suivi qui porte au moins une des quatre valeurs ;
// un i10 qui ferme la porte « attache » arrete le suivi.
func (c *canalDeLEquipementPorte) delta(ts uint64, r *FrameRecord) {
	v, ok := c.vies[r.Slot]
	if !ok || v.gen != r.ID>>30 {
		return
	}
	c.lu.st.Deltas++
	if parent, ok := parentDuRecord(r); ok && !parent.Attached {
		c.lu.st.Detached++
		delete(c.vies, r.Slot)
		return
	}
	l := c.lecture(ts, r, v, types.CarriedDelta)
	if l.HasEnergy || l.HasActivation || l.HasSignal || l.HasHandles {
		c.lu.lectures = append(c.lu.lectures, l)
	}
}

// suppression clot le suivi d un objet porte.
func (c *canalDeLEquipementPorte) suppression(ts uint64, r *FrameRecord) {
	v, ok := c.vies[r.Slot]
	if !ok || v.gen != r.ID>>30 {
		return
	}
	c.lu.st.Deleted++
	delete(c.vies, r.Slot)
	c.lu.lectures = append(c.lu.lectures, types.CarriedEquipmentRead{TimestampUS: ts,
		Object: types.LifeKey{Slot: r.Slot, Gen: v.gen}, Carrier: v.porteur, Kind: types.CarriedDel})
}

// lecture rend la lecture d un record d objet porte : les valeurs que la capture a rendues.
func (c *canalDeLEquipementPorte) lecture(ts uint64, r *FrameRecord, v vieDObjetPorte,
	k types.CarriedReadKind) types.CarriedEquipmentRead {
	l := types.CarriedEquipmentRead{TimestampUS: ts, Object: types.LifeKey{Slot: r.Slot, Gen: v.gen},
		Carrier: v.porteur, Kind: k}
	for _, comp := range r.Trace.Comps {
		e, ok := comp.EquipementOf()
		if !ok || !comp.Ported {
			continue
		}
		switch e.Composant {
		case compEquipmentEnergy:
			l.HasEnergy, l.Energy = true, uint32(e.Valeur) //nolint:gosec // R(14)
			c.lu.st.EnergyReads++
		case compEquipmentActivated:
			if e.Porte {
				l.HasActivation = true
				c.lu.st.Activations++
			}
		case compEquipmentControlSignal:
			l.HasSignal, l.Signal = true, uint32(e.Valeur) //nolint:gosec // R(4)
			c.lu.st.Signals++
		case compEquipmentTrackedStack:
			l.HasHandles, l.Handles = true, uint32(e.Valeur) //nolint:gosec // R(4)
			c.lu.st.HandleReads++
		}
	}
	return l
}

// parentDuRecord rend la lecture i10 capturee d un record, s il en porte une.
func parentDuRecord(r *FrameRecord) (ObjectParentState, bool) {
	for _, comp := range r.Trace.Comps {
		if st, ok := comp.ParentOf(); ok && comp.Ported {
			return st, true
		}
	}
	return ObjectParentState{}, false
}

// ScanEquipementPorte rend les lectures de l equipement porte d un film DEJA CHARGE, dans l ordre du
// flux : celles de la marche de la cuisson quand elle les a deja recueillies ([ScanMarcheDesTramesAvec]),
// sinon une marche faite pour elles seules.
func ScanEquipementPorte(fc *FilmContext) ([]types.CarriedEquipmentRead, types.CarriedEquipmentStats, error) {
	if fc.recup.equipementPorte == nil {
		if err := Distribuer(fc, nouveauCanalDeLEquipementPorte(fc)); err != nil {
			return nil, types.CarriedEquipmentStats{}, err
		}
	}
	lu := fc.recup.equipementPorte
	return lu.lectures, lu.st, nil
}
