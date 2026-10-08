package grammar

// objets_du_monde_lus.go — LES PISTES ET LES CREATIONS DES OBJETS DU MONDE : LA MARCHE DES TRAMES
// D ABORD, LES PASSES DE RECUPERATION DERRIERE ELLE (plan de l etape 2 de la representation
// intermediaire, 2.7.d3 ; ADR 0037 IR-6).
//
// # LA MARCHE DESIGNE LES RECORDS, LES LECTEURS DE TOUJOURS LES LISENT
//
// Le canal des objets du monde ([canalDesObjetsDuMonde]) retient, de chaque trame que la marche lit,
// les records DELTA des archetypes a pistes ([archetypesAPistes]) dont le composant i0 a ete
// traverse, et les records NEW des archetypes de creation ([archetypesDeCreation]) — chacun avec
// l archetype que la table d entites de la marche lui donne. Une piste se lit ensuite au bit d i0 du
// record ([decodeWorldObjectPos]), une creation a son en-tete ([equipCreationWalk.creationA]) : aux
// records que la passe trouvait aussi, la valeur est la meme par construction.
//
// # LES PASSES, DERRIERE
//
// La passe des pistes ([echantillonsDesBandes]) et celle des creations ([releverLesCreations]) ne
// rendent plus qu un record d un slot que la marche n a pas lu dans le paquet, hors de ce que la
// fermeture de la trame prouve ; dans une trame qu elle ne prouve pas, d un slot que la marche n a
// pas lu sous l archetype demande — la regle des huit lecteurs bipedes ([rendParLAncrage]). Ce
// qu elles rendent se compte (`repli_pistes_du_monde_apres_la_marche`,
// `repli_creations_du_monde_apres_la_marche`, ordre « apres la lecture »).
//
// # CE QUE LA REGLE CORRIGE
//
// Une bande de slots recouvre parfois celle d un autre archetype : la passe donnait alors un meme
// record a toutes les bandes qui portent son slot, et un equipement passait aussi pour une arme au
// sol ; dans une trame que la fermeture prouve, la marche le rend a son seul archetype. Un en-tete
// fortuit a l interieur d un autre record d une trame que la fermeture prouve n est plus une piste
// ni une creation (decouvertes 37 et 39 du plan, mesure 3 de 2.7.d0).
//
// # SANS REGISTRE, OU SANS ARCHETYPE, LA PASSE SEULE
//
// La marche exige le registre du film : un film sans `chunk_00` (bobines de test) garde les passes
// seules. Une demande de pistes sans archetype ([ArchetypeDeBandeInconnu] : les enveloppes
// d instrument, qui balaient des bandes arbitraires) aussi.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ArchetypeDeBandeInconnu est l archetype d une demande de pistes qui n en nomme aucun : la passe
// seule la sert.
const ArchetypeDeBandeInconnu = -1

// objetsDuMondeLus : les records d objets du monde que la marche des trames a lus, groupes par paquet
// dans l ordre du flux — les DELTA a i0 des archetypes a pistes, les NEW des archetypes de creation.
type objetsDuMondeLus struct {
	pistes, creations recordsDuMonde
}

// recordsDuMonde : des records d objets du monde groupes par paquet, dans l ordre du flux.
type recordsDuMonde struct {
	paquets []paquetAncre
	records []recordDuMonde
}

// recordDuMonde est un record d objet du monde que la marche a lu : le bit de son i0 (piste) ou de
// son en-tete (creation), son slot, sa generation, son archetype, et pour une piste le composant de
// repos (`projectile-at-rest-state`) a son masque.
type recordDuMonde struct {
	bit, slot uint32
	gen, ti   uint8
	repos     bool
}

// noter ajoute un record du paquet `pk` du chunk `chunk`.
func (rm *recordsDuMonde) noter(chunk int, pk FilmPacket, r recordDuMonde) {
	if n := len(rm.paquets); n == 0 || rm.paquets[n-1].chunk != chunk || rm.paquets[n-1].paquet.Index != pk.Index {
		rm.paquets = append(rm.paquets, paquetAncre{chunk: chunk, paquet: pk, premier: len(rm.records)})
	}
	rm.records = append(rm.records, r)
}

// recordsDu rend les records du k-ieme paquet.
func (rm *recordsDuMonde) recordsDu(k int) []recordDuMonde {
	fin := len(rm.records)
	if k+1 < len(rm.paquets) {
		fin = rm.paquets[k+1].premier
	}
	return rm.records[rm.paquets[k].premier:fin]
}

// parcourir rend a `visit` chaque record de l archetype `ti` et d un slot de `band`, avec le payload
// et l en-tete de son paquet, dans l ordre du flux.
func (rm *recordsDuMonde) parcourir(fc *FilmContext, ti int, band map[uint32]bool,
	visit func(pay []byte, chunk int, pk FilmPacket, r recordDuMonde)) {
	chunk := -1
	var data []byte
	var pks []FilmPacket
	for k, p := range rm.paquets {
		var pay []byte
		for _, r := range rm.recordsDu(k) {
			if int(r.ti) != ti || !band[r.slot] {
				continue
			}
			if pay == nil {
				if p.chunk != chunk {
					chunk = p.chunk
					data, pks, _ = fc.ChunkAt(chunk)
				}
				if pay = payloadDeRang(data, pks, p.paquet.Index); pay == nil {
					break
				}
			}
			visit(pay, p.chunk, p.paquet, r)
		}
	}
}

// canalDesObjetsDuMonde est le canal des trames qui retient les records d objets du monde que la
// marche lit ([objetsDuMondeLus]) et les range dans le contexte a la cloture.
type canalDesObjetsDuMonde struct {
	fc  *FilmContext
	m   *MarcheDistribuee
	lu  objetsDuMondeLus
	reg *Registry
	// aPistes, deCreation : les archetypes retenus, indexes par archetype (sous 64).
	aPistes, deCreation [64]bool
}

// nouveauCanalDesObjetsDuMonde prepare le canal des objets du monde du film `fc`.
func nouveauCanalDesObjetsDuMonde(fc *FilmContext) *canalDesObjetsDuMonde {
	reg, err := fc.Registry()
	if err != nil {
		reg = nil
	}
	c := &canalDesObjetsDuMonde{fc: fc, reg: reg}
	for _, ti := range archetypesAPistes() {
		c.aPistes[ti] = true
	}
	for _, ti := range archetypesDeCreation() {
		c.deCreation[ti] = true
	}
	return c
}

// Interets : le composant i0 des archetypes a pistes, dans les trames.
func (c *canalDesObjetsDuMonde) Interets() []Interet {
	if c.reg == nil {
		return nil
	}
	var out []Interet
	for _, ti := range archetypesAPistes() {
		if arch, ok := c.reg.Archetype(ti); ok && len(arch.Components) > 0 {
			out = append(out, Interet{Phase: PhaseTrames, TI: ti, Composant: arch.component(0)})
		}
	}
	return out
}

func (c *canalDesObjetsDuMonde) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Trame retient les records d objets du monde de la trame `p`. Une trame que la marche n a pas lue,
// ou pas marchee par classes de vue, ne designe rien : le canal des lectures bipedes n en retient
// aucun slot, et la passe la lit entiere, comme l ancrage ([canalDesLecturesBipedes.Trame]).
func (c *canalDesObjetsDuMonde) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	if !lus || !c.m.attribuable() {
		return
	}
	pk := FilmPacket{Index: p.Index, Type: p.Type, TimestampUS: p.TS}
	for i := range recs {
		r := &recs[i]
		ti := int(r.TypeIndex)
		lu := recordDuMonde{slot: r.Slot, gen: uint8(r.ID >> 30), ti: uint8(ti)} //nolint:gosec // generation sur 2 bits, archetype sur 6
		switch {
		case ti >= len(c.aPistes):
		case r.Type == recDelta && c.aPistes[ti]:
			if len(r.Trace.Comps) == 0 || r.Trace.Comps[0].Index != 0 || !r.Trace.Comps[0].Ported {
				continue
			}
			lu.bit = uint32(r.Trace.Comps[0].StartBit) //nolint:gosec // position dans un payload
			lu.repos = r.Trace.Mask>>projectileRestComponent&1 == 1
			c.lu.pistes.noter(p.Chunk, pk, lu)
		case r.Type == recNew && c.deCreation[ti]:
			lu.bit = uint32(r.HeaderBit) //nolint:gosec // position dans un payload
			c.lu.creations.noter(p.Chunk, pk, lu)
		}
	}
}

// Clore range les records dans le contexte.
func (c *canalDesObjetsDuMonde) Clore(BilanDeMarche) { c.fc.recup.objets = &c.lu }

// objetsDuMonde rend les records d objets du monde que la marche a lus et les trames qu elle a
// rendues ([lecturesBipedes.trames]), faits une fois par contexte : par la distribution de la
// cuisson, ou par une distribution faite ici pour eux seuls. Sans registre, une erreur : la passe
// seule sert alors la demande.
func (c *FilmContext) objetsDuMonde() (*objetsDuMondeLus, map[paquetDuFlux]trameDuCanal, error) {
	if c.recup.objets == nil || c.recup.lectures == nil {
		var canaux []Canal
		if c.recup.lectures == nil {
			canaux = append(canaux, nouveauCanalDesLecturesBipedes(c))
		}
		if c.recup.objets == nil {
			canaux = append(canaux, nouveauCanalDesObjetsDuMonde(c))
		}
		if err := Distribuer(c, canaux...); err != nil {
			return nil, nil, err
		}
	}
	return c.recup.objets, c.recup.lectures.trames, nil
}

// pistesDerriereLaMarche rend les pistes de l archetype `ti` sur la bande `band` : les echantillons
// des records que la marche a lus, puis ceux de la passe (`brut`) qu elle rend derriere elle ; et le
// nombre de ces derniers. Sans marche (pas de registre, ou archetype inconnu), la passe seule.
func (c *FilmContext) pistesDerriereLaMarche(wr profile.Vec3Range, lg profile.PrecisionDescriptor, ti int,
	band map[uint32]bool, brut []projSample) ([]types.ProjectileTrack, int) {
	vies := map[vieDePiste][]types.ProjectileSample{}
	if ti == ArchetypeDeBandeInconnu {
		return pistesDeLaPasse(vies, brut), 0
	}
	objets, trames, err := c.objetsDuMonde()
	if err != nil {
		return pistesDeLaPasse(vies, brut), 0
	}
	objets.pistes.parcourir(c, ti, band, func(pay []byte, chunk int, pk FilmPacket, r recordDuMonde) {
		v, ok := decodeWorldObjectPos(pay, int(r.bit), &wr, lg)
		if !ok {
			return
		}
		k := vieDePiste{r.slot, uint32(r.gen)}
		vies[k] = append(vies[k], types.ProjectileSample{TimestampUS: pk.TimestampUS, Chunk: chunk,
			X: v[0], Y: v[1], Z: v[2], AtRest: r.repos})
	})
	recuperes := 0
	for _, s := range brut {
		t, vu := trames[paquetDuFlux{s.Chunk, s.paquet}]
		if !rendParLAncrage(t, vu, s.slot, ti, s.bit) {
			continue
		}
		recuperes++
		k := vieDePiste{s.slot, s.gen}
		vies[k] = append(vies[k], s.ProjectileSample)
	}
	return pistesDesVies(vies), recuperes
}

// creationsDerriereLaMarche rend les creations de la marche de creation `w` : celles des records NEW
// que la marche des trames a lus, de son archetype et de sa bande, lues a leur en-tete
// ([equipCreationWalk.creationA]) ; puis celles de la passe (`brut`) qu elle rend derriere elle ; leurs
// comptes, et le nombre des creations rendues par la passe. Sans marche, la passe seule (`stBrut`).
func (c *FilmContext) creationsDerriereLaMarche(w equipCreationWalk, brut []types.EquipmentCreation,
	stBrut types.EquipmentCreationStats) ([]types.EquipmentCreation, types.EquipmentCreationStats, int) {
	objets, trames, err := c.objetsDuMonde()
	if err != nil {
		return brut, stBrut, 0
	}
	st := types.EquipmentCreationStats{Slots: stBrut.Slots}
	var out []types.EquipmentCreation
	objets.creations.parcourir(c, int(w.archetype()), w.band, func(pay []byte, chunk int, pk FilmPacket, r recordDuMonde) {
		lieu := lieuDuPaquet{chunk: chunk, pk: pk}
		if cre, ok := w.creationA(pay, int(r.bit), types.LifeKey{Slot: r.slot, Gen: uint32(r.gen)}, lieu, &st); ok {
			out = append(out, cre)
		}
	})
	recuperes := 0
	for _, x := range brut {
		t, vu := trames[paquetDuFlux{x.Chunk, x.PacketIndex}]
		if !rendParLAncrage(t, vu, x.Slot, int(w.archetype()), x.BitPos) {
			continue
		}
		recuperes++
		st.Anchors++
		compterLaCreation(&st, x)
		out = append(out, x)
	}
	rang := make(map[int]int, len(c.ChunkNumbers()))
	for i, n := range c.ChunkNumbers() {
		rang[n] = i
	}
	slices.SortStableFunc(out, func(a, b types.EquipmentCreation) int {
		return cmp.Or(cmp.Compare(rang[a.Chunk], rang[b.Chunk]), cmp.Compare(a.PacketIndex, b.PacketIndex),
			cmp.Compare(a.BitPos, b.BitPos))
	})
	return out, st, recuperes
}

// pistesDeLaPasse rend les pistes des seuls echantillons de la passe, ajoutes a `vies`.
func pistesDeLaPasse(vies map[vieDePiste][]types.ProjectileSample, brut []projSample) []types.ProjectileTrack {
	for _, s := range brut {
		k := vieDePiste{s.slot, s.gen}
		vies[k] = append(vies[k], s.ProjectileSample)
	}
	return pistesDesVies(vies)
}
