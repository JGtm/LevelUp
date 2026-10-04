package grammar

// creations_du_monde.go — LES CREATIONS DES OBJETS DU MONDE, MARCHEES EN UNE PASSE POUR TOUS LEURS
// ARCHETYPES (ADR 0037 IR-6 ; lot 2.5 du plan de l etape 2).
//
// # UNE PASSE, UN CURSEUR PAR ARCHETYPE
//
// Les poses, les socles et les vehicules marchaient chacun les payloads delta du film, bit a bit,
// pour reconnaitre les en-tetes de creation de leur archetype — l equipement deux fois. Un en-tete
// NEW porte son archetype : a une position donnee, une seule marche peut le reconnaitre. La passe
// lit le debut de l en-tete une fois par position ([archetypeDeLEnTeteNEW]) et le donne a la marche
// de cet archetype, qui le reconnait en entier ([matchWorldObjectNewHeader]) et lit le record
// ([equipCreationWalk.creationA]). Chaque marche garde SON curseur : un record accepte n avance que
// le sien, comme la marche d un archetype seul. Les creations sont donc celles de ces marches.
//
// # CE QUI EST MARCHE, ET QUAND C EST REUTILISE
//
// A la premiere demande, la marche demandee et celles des autres archetypes de creation de la
// cuisson ([archetypesDeCreation]), sur leur bande, aux memes bornes et sous le profil de balayage
// du moment (largeurs MPP comprises). Une demande suivante ne reutilise une marche que si
// l archetype, la bande, les bornes et le profil sont les memes ([cleDeCreation]) ; sinon elle est
// marchee a nouveau. La reutilisation rend une COPIE.

import (
	"fmt"
	"reflect"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// archetypesDeCreation rend les archetypes dont la cuisson marche les creations : l equipement
// (poses et socles), les armes au sol (socles) et les vehicules.
func archetypesDeCreation() []uint32 {
	return []uint32{EquipmentTypeIndex, GroundWeaponTypeIndex, VehicleTypeIndex}
}

// cleDeCreation est ce dont depend une marche de creation : son archetype, sa bande, ses bornes et
// le profil de balayage sous lequel elle lit.
type cleDeCreation struct {
	ti    uint32
	slots []uint32
	wr    profile.Vec3Range
	prof  ProfilDeBalayage
}

// cleDe rend la cle d une marche.
func cleDe(w equipCreationWalk) cleDeCreation {
	return cleDeCreation{ti: w.archetype(), slots: slotsDeLaBande(w.band), wr: *w.wr, prof: w.prof}
}

// egale dit si deux cles designent la meme marche.
func (k cleDeCreation) egale(o cleDeCreation) bool {
	return k.ti == o.ti && k.wr == o.wr && slices.Equal(k.slots, o.slots) && reflect.DeepEqual(k.prof, o.prof)
}

// creationsRelevees : ce qu une marche de creation a rendu, sous sa cle.
type creationsRelevees struct {
	cle       cleDeCreation
	creations []types.EquipmentCreation
	stats     types.EquipmentCreationStats
}

// creationsDe rend ce qui a ete marche sous la cle `k`.
func (m *memoDesRecuperations) creationsDe(k cleDeCreation) (creationsRelevees, bool) {
	for _, r := range m.creations {
		if r.cle.egale(k) {
			return r, true
		}
	}
	return creationsRelevees{}, false
}

// marcheDeCreation construit la marche de creation de l archetype `ti` sur la bande `band`, sous le
// profil de balayage du contexte. Les erreurs sont celles de l archetype (absent du registre) et,
// pour le vehicule, du decoupage d i0.
func (c *FilmContext) marcheDeCreation(ti uint32, wr *profile.Vec3Range, band map[uint32]bool) (
	equipCreationWalk, error) {
	cur := &equipCreationRead{}
	w := equipCreationWalk{obs: installCreationHooks(cur), prof: c.ProfilDeBalayage(), wr: wr, band: band, cur: cur}
	switch ti {
	case GroundWeaponTypeIndex:
		arch, err := c.groundWeaponArchetype()
		if err != nil {
			return w, err
		}
		w.comps, w.ti, w.deser, w.ammoArch = len(arch.Components), GroundWeaponTypeIndex, consumeDefaultStateTI42, &arch
	case VehicleTypeIndex:
		lay, err := c.I0Layout()
		if err != nil {
			return w, fmt.Errorf("decoupage i0 illisible : %w", err)
		}
		arch, err := c.vehicleArchetype()
		if err != nil {
			return w, err
		}
		w.comps, w.ti, w.deser, w.posBits = len(arch.Components), VehicleTypeIndex, consumeDefaultStateTI40, lay.TotalBits()
		w.posDecode = func(pay []byte, at int) ([3]float32, bool) { return decodeBipedI0Pos(pay, at, lay, wr) }
	default:
		arch, err := c.EquipmentArchetype()
		if err != nil {
			return w, err
		}
		w.comps = len(arch.Components)
	}
	return w, nil
}

// creationsRelevees rend les creations et les comptes de la marche `w` : une copie de ce qui a ete
// marche sous sa cle, ou, au premier appel, une passe qui la marche avec celles des autres
// archetypes de creation de la cuisson.
func (c *FilmContext) creationsRelevees(w equipCreationWalk) ([]types.EquipmentCreation,
	types.EquipmentCreationStats) {
	if r, ok := c.recup.creationsDe(cleDe(w)); ok {
		return copierLesCreations(r.creations), r.stats
	}
	marches := []equipCreationWalk{w}
	for _, ti := range archetypesDeCreation() {
		if ti == w.archetype() {
			continue
		}
		band := worldObjectSlotBand(c, int(ti))
		if len(band) == 0 {
			continue
		}
		autre, err := c.marcheDeCreation(ti, w.wr, band)
		if err != nil {
			continue // un archetype que le film ne porte pas : sa propre demande rendra l erreur
		}
		if _, deja := c.recup.creationsDe(cleDe(autre)); !deja {
			marches = append(marches, autre)
		}
	}
	creations, stats := releverLesCreations(c, marches)
	for i, m := range marches {
		c.recup.creations = append(c.recup.creations, creationsRelevees{cle: cleDe(m), creations: creations[i],
			stats: stats[i]})
	}
	return copierLesCreations(creations[0]), stats[0]
}

// copierLesCreations rend une copie profonde de creations ; nil reste nil.
func copierLesCreations(cr []types.EquipmentCreation) []types.EquipmentCreation {
	if cr == nil {
		return nil
	}
	out := make([]types.EquipmentCreation, len(cr))
	for i, x := range cr {
		x.Mask = slices.Clone(x.Mask)
		out[i] = x
	}
	return out
}

// releverLesCreations marche, en une passe sur les payloads delta du film, les creations de
// chacune des `marches`.
func releverLesCreations(c *FilmContext, marches []equipCreationWalk) ([][]types.EquipmentCreation,
	[]types.EquipmentCreationStats) {
	ps := nouvellePasseDesCreations(marches)
	for _, ch := range c.ChunkNumbers() {
		data, pks, ok := c.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type == PacketTypeDelta {
				ps.payload(pk.Payload(data), lieuDuPaquet{chunk: ch, pk: pk})
			}
		}
	}
	return ps.outs, ps.sts
}

// lieuDuPaquet situe un payload delta dans le film : son chunk et l en-tete de son paquet.
type lieuDuPaquet struct {
	chunk int
	pk    FilmPacket
}

// passeDesCreations porte une passe des creations : ses marches, le curseur de chacune dans le
// payload en cours, et ce que chacune a rendu.
type passeDesCreations struct {
	marches  []equipCreationWalk
	curseurs []int
	outs     [][]types.EquipmentCreation
	sts      []types.EquipmentCreationStats
}

// nouvellePasseDesCreations prepare une passe des creations pour `marches`.
func nouvellePasseDesCreations(marches []equipCreationWalk) *passeDesCreations {
	ps := &passeDesCreations{marches: marches, curseurs: make([]int, len(marches)),
		outs: make([][]types.EquipmentCreation, len(marches)), sts: make([]types.EquipmentCreationStats, len(marches))}
	for i := range marches {
		ps.sts[i].Slots = len(marches[i].band)
	}
	return ps
}

// payload marche UN payload delta pour toutes les marches de la passe, chacune avec son curseur,
// remis a zero au debut du payload.
func (ps *passeDesCreations) payload(pay []byte, lieu lieuDuPaquet) {
	clear(ps.curseurs)
	total := len(pay) * 8
	for p := 0; p <= total-woNewHeaderBits; p++ {
		ti, ok := archetypeDeLEnTeteNEW(pay, p)
		if !ok {
			continue
		}
		for k, w := range ps.marches {
			if ps.curseurs[k] > p || w.archetype() != ti {
				continue
			}
			slot, gen, ok := matchWorldObjectNewHeader(pay, p, w.band, ti)
			if !ok {
				continue
			}
			cre, ok := w.creationA(pay, p, types.LifeKey{Slot: slot, Gen: gen}, lieu, &ps.sts[k])
			if !ok {
				continue
			}
			ps.outs[k] = append(ps.outs[k], cre)
			ps.curseurs[k] = cre.AfterBit // un record accepte n est pas re-balaye par sa marche
		}
	}
}
