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
//
// LES OBJETS QUE L EQUIPEMENT ENGENDRE. Les entrees presentes de la pile i28 sont des handles : les
// vies des objets que l equipement porte suit (la faille du translocateur, le champ de reparation,
// des entites ti=41 nees dans le paquet ou la pile les nomme). Le canal retient chaque vie nommee,
// meme apres la fin de l objet porte, et rend sa SUPPRESSION (`tracked-end`), rattachee a l objet
// qui la suivait : c est la fin, ecrite par le film, de l objet engendre.
//
// L OBJET D EFFET (plan des durees d effets, etape B). Certains objets engendres ne sont que le vehicule de
// l effet : la grenade de l ecran occultant est detruite a l impact, et c est une AUTRE vie (ti=38) que
// la pile i28 ne nomme pas, nee au meme instant, qui porte la bulle (7,507 s, mesure sur 5 films). Son
// record de creation ne designe rien (i0, i2, i14, i15 seulement) : le canal rend donc, a la suppression
// de CHAQUE objet suivi, les vies ti=38 non nommees nees 100 a 700 ms avant elle (`effect-birth`, une lecture
// par candidate), puis la suppression de chaque candidate (`effect-end`). Le sens (quelle famille a un
// effet, laquelle des candidates est la bonne) se tranche a la publication.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// equipementPorteLu est ce que le canal a lu d un film, range dans le contexte a la cloture.
type equipementPorteLu struct {
	lectures []types.CarriedEquipmentRead
	st       types.CarriedEquipmentStats
	// quanta : les quanta i0 lus a la creation des objets engendres, par vie, et les largeurs de la
	// marche ; [ScanEquipementPorte] les dequantifie aux bornes de la carte.
	quanta map[types.LifeKey][3]uint64
	// quantaEffets : les quanta i0 des candidates a l effet, par vie ET par instant de creation — la cle (slot,
	// generation) revient dans le match, et deux candidates de meme cle se partageraient les quanta de la derniere.
	quantaEffets map[naissanceCle][3]uint64
	lg           profile.PrecisionDescriptor
}

// vieDObjetPorte est un objet porte suivi : sa generation et son porteur.
type vieDObjetPorte struct {
	gen, porteur uint32
}

// objetSuivi est une vie designee par la pile i28 d un objet porte : sa generation, et l objet porte
// (vie, porteur) qui la suit.
type objetSuivi struct {
	gen     uint32
	objet   types.LifeKey
	porteur uint32
}

// canalDeLEquipementPorte lit, dans les trames que la marche rend, les records des objets ti=37
// crees attaches.
type canalDeLEquipementPorte struct {
	fc   *FilmContext
	reg  *Registry
	m    *MarcheDistribuee
	vies map[uint32]vieDObjetPorte
	// suivis : les vies que la pile i28 d un objet porte a designees, par slot : l objet qui les suit.
	suivis map[uint32]objetSuivi
	// naissances : la derniere creation (record NEW) lue par slot, avec sa position i0 — la naissance
	// d un objet que la pile i28 nommera.
	naissances map[uint32]naissanceLue
	// nommes : les instants ou la pile i28 d un objet porte a designe chaque vie. Une candidate a l effet n est
	// jamais nommee ; la cle (slot, generation) revient dans le match (2 bits de generation), d ou l instant.
	nommes map[types.LifeKey][]uint64
	// effets : les candidates dont la suppression sera rendue, par slot. nes38 : la derniere creation ti=38 lue par
	// slot, DANS LES RECORDS DE LECTURE de chaque trame (cf. [canalDeLEquipementPorte.lireLesEffets]).
	effets map[uint32]objetSuivi
	nes38  map[uint32]naissanceLue
	lu     equipementPorteLu
}

// Les genres de lecture de l objet d effet. Declares ICI et non dans `types` : le paquet `types` est dans le
// perimetre de la couche `source` (fermeture des imports), et un genre de lecture que seule la grammaire
// ecrit ne doit pas faire monter la porte aux octets.
const (
	// CarriedEffectBirth : une CANDIDATE a l objet d effet d un objet suivi, lue a la suppression de ce dernier.
	// Object et Carrier sont ceux de l objet porte qui le suivait, Tracked[0] la vie suivie qui vient d etre
	// supprimee, Spawned[0] la creation (instant, archetype, position i0) d une vie ti=38 QUE NULLE PILE i28 NE
	// NOMME, nee entre 100 et 700 ms avant cette suppression. Le film ne designe pas la vie suivie dans le
	// record de creation de la candidate (i0, i2, i14, i15 seulement) : c est une ASSOCIATION par l instant, que
	// la publication resserre par la famille et la position de repos.
	CarriedEffectBirth types.CarriedReadKind = "effect-birth"
	// CarriedEffectEnd : la suppression d une candidate lue en [CarriedEffectBirth] ; Object et Carrier sont ceux de
	// l objet porte qui avait suivi la vie supprimee, Tracked[0] la vie de l effet.
	CarriedEffectEnd types.CarriedReadKind = "effect-end"
)

// Les bornes de la fenetre, avant la suppression d un objet suivi, ou nait la vie d effet. Mesure du
// plan des durees d effets (etape A, 5 films, 38 ecrans) : la bulle nait 150 a 600 ms avant la
// suppression de la grenade (mediane 250) ; 100 ms et 700 ms laissent une marge d un paquet de chaque cote.
const (
	// effetNommageUS : une vie nee a moins de 2 s d un nommage de sa cle est celle que la pile designe ; au-dela, la
	// cle a ete recyclee. Meme borne que la mesure de l etape A.
	effetNommageUS     = 2_000_000
	archetypeDeLEffet  = 38
	effetAvantFinMinUS = 100_000
	effetAvantFinMaxUS = 700_000
)

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
	return &canalDeLEquipementPorte{fc: fc, reg: reg, vies: map[uint32]vieDObjetPorte{},
		suivis: map[uint32]objetSuivi{}, naissances: map[uint32]naissanceLue{},
		nommes: map[types.LifeKey][]uint64{}, effets: map[uint32]objetSuivi{}, nes38: map[uint32]naissanceLue{},
		lu: equipementPorteLu{quanta: map[types.LifeKey][3]uint64{}, quantaEffets: map[naissanceCle][3]uint64{}}}
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

// Brancher retient la marche, et les largeurs d objet du monde qu elle lit : celles des quanta i0.
func (c *canalDeLEquipementPorte) Brancher(_ *Observation, m *MarcheDistribuee) {
	c.m = m
	c.lu.lg = c.fc.LargeursObjetDuMonde()
}

// Clore range ce que le canal a lu dans le contexte.
func (c *canalDeLEquipementPorte) Clore(BilanDeMarche) { c.fc.recup.equipementPorte = &c.lu }

// Trame lit les records d objets portes de la trame `p`.
func (c *canalDeLEquipementPorte) Trame(p *lecture.Paquet) {
	c.lireLesEffets(p)
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		return
	}
	// LES NAISSANCES D ABORD : un objet engendre peut naitre dans le paquet meme ou la pile le nomme.
	for i := range recs {
		if recs[i].Type == recNew && recs[i].DesyncAt == -1 {
			c.naissance(p, &recs[i])
		}
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

// suppression clot le suivi d un objet porte, ou date la fin d une vie que sa pile i28 a designee.
func (c *canalDeLEquipementPorte) suppression(ts uint64, r *FrameRecord) {
	if s, ok := c.suivis[r.Slot]; ok && s.gen == r.ID>>30 {
		c.lu.st.TrackedEnds++
		delete(c.suivis, r.Slot)
		fin := types.LifeKey{Slot: r.Slot, Gen: s.gen}
		c.lu.lectures = append(c.lu.lectures, types.CarriedEquipmentRead{TimestampUS: ts, Object: s.objet,
			Carrier: s.porteur, Kind: types.CarriedTrackedEnd, Tracked: []types.LifeKey{fin}})
		c.candidatesALEffet(ts, s, fin)
		return
	}
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
			l.HasHandles, l.Handles, l.Tracked = true, uint32(e.Valeur), e.Suivis //nolint:gosec // R(4)
			c.lu.st.HandleReads++
			c.lu.st.TrackedEntries += len(e.Suivis)
			c.suivre(&l, v)
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
// sinon une marche faite pour elles seules. La position a la creation des objets engendres
// (`Spawned`) est dequantifiee aux bornes `wr` de la carte ; sans bornes, elle reste absente.
func ScanEquipementPorte(fc *FilmContext, wr *profile.Vec3Range) ([]types.CarriedEquipmentRead,
	types.CarriedEquipmentStats, error) {
	if fc.recup.equipementPorte == nil {
		if err := Distribuer(fc, nouveauCanalDeLEquipementPorte(fc)); err != nil {
			return nil, types.CarriedEquipmentStats{}, err
		}
	}
	lu := fc.recup.equipementPorte
	if wr == nil {
		return lu.lectures, lu.st, nil
	}
	ls := make([]types.CarriedEquipmentRead, len(lu.lectures))
	st := lu.st
	for i, l := range lu.lectures {
		ls[i] = l
		if len(l.Spawned) == 0 {
			continue
		}
		ls[i].Spawned = make([]types.SpawnedObject, len(l.Spawned))
		for k, n := range l.Spawned {
			q, ok := lu.quanta[n.Life]
			if l.Kind == CarriedEffectBirth {
				q, ok = lu.quantaEffets[naissanceCle{n.Life, n.TimestampUS}]
			}
			if ok {
				v := dequantObjetDuMonde(q, wr, lu.lg)
				n.HasPosition, n.X, n.Y, n.Z = true, v[0], v[1], v[2]
				// SpawnedPositioned est un compte de VIES NOMMEES (son denominateur est SpawnedBirths) : une
				// candidate a l effet n est pas nommee, elle ne s y ajoute pas.
				if l.Kind != CarriedEffectBirth {
					st.SpawnedPositioned++
				}
			}
			ls[i].Spawned[k] = n
		}
	}
	return ls, st, nil
}

// naissance retient la creation d une vie et les quanta de sa position i0, lus par le lecteur des
// pistes du monde ([lireQuantaObjetDuMonde] : chemin dominant, region de la carte, quanta satures
// rejetes) au bit ou la marche a trouve le composant.
func (c *canalDeLEquipementPorte) naissance(p *lecture.Paquet, r *FrameRecord) {
	n := naissanceLue{obj: types.SpawnedObject{Life: types.LifeKey{Slot: r.Slot, Gen: r.ID >> 30},
		TimestampUS: p.TS, TypeIndex: r.TypeIndex}}
	if comps := r.Trace.Comps; len(comps) > 0 && comps[0].Name == compObjectPosition && comps[0].Ported {
		n.q, n.aQ = lireQuantaObjetDuMonde(p.Payload, comps[0].StartBit, c.lu.lg)
	}
	c.naissances[r.Slot] = n
}

// naissanceCle identifie une creation : sa vie et l instant du paquet.
type naissanceCle struct {
	vie types.LifeKey
	ts  uint64
}

// naissanceLue est la derniere creation lue d un slot, et les quanta de sa position i0.
type naissanceLue struct {
	obj types.SpawnedObject
	q   [3]uint64
	aQ  bool
}

// suivre retient les vies que la pile i28 d une lecture nomme, et joint a la lecture la creation de
// celles qu elle nomme pour la premiere fois, quand la marche l a lue.
func (c *canalDeLEquipementPorte) suivre(l *types.CarriedEquipmentRead, v vieDObjetPorte) {
	for _, s := range l.Tracked {
		if deja, ok := c.suivis[s.Slot]; ok && deja.gen == s.Gen {
			continue
		}
		c.suivis[s.Slot] = objetSuivi{gen: s.Gen, objet: l.Object, porteur: v.porteur}
		c.nommes[s] = append(c.nommes[s], l.TimestampUS)
		n, ok := c.naissances[s.Slot]
		if !ok || n.obj.Life != s {
			continue
		}
		l.Spawned = append(l.Spawned, n.obj)
		c.lu.st.SpawnedBirths++
		if n.aQ {
			c.lu.quanta[s] = n.q
		}
	}
}

// candidatesALEffet rend, a la suppression de la vie suivie `fin` (instant `ts`, suivie par `s`), une lecture
// `effect-birth` par vie ti=38 NON NOMMEE nee dans la fenetre [ts - 700 ms, ts - 100 ms], dans l ordre de
// leur naissance (puis de leur slot), et retient chacune pour rendre sa suppression.
func (c *canalDeLEquipementPorte) candidatesALEffet(ts uint64, s objetSuivi, fin types.LifeKey) {
	var cands []naissanceLue
	for _, n := range c.nes38 {
		if c.estNommee(n.obj) {
			continue
		}
		if n.obj.TimestampUS+effetAvantFinMinUS > ts || ts > n.obj.TimestampUS+effetAvantFinMaxUS {
			continue
		}
		cands = append(cands, n)
	}
	slices.SortFunc(cands, func(a, b naissanceLue) int {
		return cmp.Or(cmp.Compare(a.obj.TimestampUS, b.obj.TimestampUS), cmp.Compare(a.obj.Life.Slot, b.obj.Life.Slot))
	})
	for _, n := range cands {
		c.lu.lectures = append(c.lu.lectures, types.CarriedEquipmentRead{TimestampUS: ts, Object: s.objet,
			Carrier: s.porteur, Kind: CarriedEffectBirth, Tracked: []types.LifeKey{fin},
			Spawned: []types.SpawnedObject{n.obj}})
		if n.aQ {
			c.lu.quantaEffets[naissanceCle{n.obj.Life, n.obj.TimestampUS}] = n.q
		}
		// UNE CANDIDATE RETENUE PAR DEUX OBJETS SUIVIS GARDE SA PREMIERE ATTRIBUTION : sa suppression est rendue une
		// fois, sous le premier objet porte qui l a retenue. La publication joint les fins d effet par la vie de la
		// candidate, sans regarder l objet porte (`finsDEffetLues`) : l attribution n y decide rien.
		if prev, ok := c.effets[n.obj.Life.Slot]; !ok || prev.gen != n.obj.Life.Gen {
			c.effets[n.obj.Life.Slot] = objetSuivi{gen: n.obj.Life.Gen, objet: s.objet, porteur: s.porteur}
		}
	}
}

// estNommee dit si la pile i28 d un objet porte a nomme CETTE vie : meme cle, nommage a moins de [effetNommageUS] de sa
// creation. Une cle nommee il y a longtemps est celle d une vie anterieure.
func (c *canalDeLEquipementPorte) estNommee(n types.SpawnedObject) bool {
	for _, at := range c.nommes[n.Life] {
		if at+effetNommageUS >= n.TimestampUS && n.TimestampUS+effetNommageUS >= at {
			return true
		}
	}
	return false
}

// lireLesEffets lit, dans les RECORDS DE LECTURE de la trame (ceux que l etape A du plan des durees d effets a
// mesures : 38 ecrans sur 39), les creations ti=38 et les suppressions des candidates retenues, sans passer par les
// records de la marche de la cuisson ([MarcheDistribuee]) : une trame que la marche ne localise pas (`lus` faux) ne
// doit pas faire taire la bulle. Sur les cinq films du plan les deux sources rendent les memes effets.
func (c *canalDeLEquipementPorte) lireLesEffets(p *lecture.Paquet) {
	for _, r := range p.Records {
		switch r.Genre {
		case lecture.GenreNeuf:
			if r.TI != archetypeDeLEffet {
				continue
			}
			n := naissanceLue{obj: types.SpawnedObject{Life: types.LifeKey{Slot: r.Vie.Slot, Gen: r.Vie.Gen},
				TimestampUS: p.TS, TypeIndex: archetypeDeLEffet}}
			for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
				if k.Index == 0 && k.Bits > 0 {
					n.q, n.aQ = lireQuantaObjetDuMonde(p.Payload, int(k.Debut), c.lu.lg)
					break
				}
			}
			c.nes38[r.Vie.Slot] = n
		case lecture.GenreSuppression:
			if e, ok := c.effets[r.Vie.Slot]; ok && e.gen == r.Vie.Gen {
				delete(c.effets, r.Vie.Slot)
				c.lu.lectures = append(c.lu.lectures, types.CarriedEquipmentRead{TimestampUS: p.TS, Object: e.objet,
					Carrier: e.porteur, Kind: CarriedEffectEnd, Tracked: []types.LifeKey{{Slot: r.Vie.Slot, Gen: e.gen}}})
			}
		}
	}
}
