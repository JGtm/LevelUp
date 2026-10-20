//go:build research

package grammar

// campagne_naissances_research_test.go — MESURES CIBLEES (etape 4) : LES REJETS ET LES NAISSANCES.
//
// Pour chaque PREMIER rejet hors datum d un eid E dans un chunk (paquet P), la sonde M1 (T1-1, C1,
// T1-5) cherche, a TOUTES les positions des paquets delta du chunk ANTERIEURS a P, un en-tete NEW
// `0 01 slot(13) tete(2)` de cet eid suivi d un `R(6)` < 50, et range la position trouvee dans la
// region que la marche n a pas lue. Un TEMOIN (meme tete, slot jamais alloue ni declare) subit la
// meme recherche : son taux de « trouve » est le taux de faux positifs de la sonde.
//
// La meme passe ventile les rejets par l ordre de l ecrivain (R1/R2, C4), par la plausibilite de
// l eid (T1-1 correction 3), par la prediction de l allocateur (T1-3), par le NEW desynchronise
// anterieur (T7-5), et juge les NEW refuses (T1-4, C5).

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// cmCollecteur recoit la marche de reference d un film et remplit ses tables.
type cmCollecteur struct {
	f           *cmFilm
	b           *cmBlocs
	t           cmTables
	chunk       int
	paquets     []*cmPaquet
	liesAuDebut map[uint32]bool
	oracle      map[[2]int][]cmLiaison
	oracleBloc  map[[2]int][]cmLiaison
	creations   map[uint32][]types.EquipmentCreation
	chassisLu   map[uint32]uint32
	queueMin    int
	statut      map[[2]int]bool
	contre      []string
	// parRegion et rangs : les liaisons d oracle par region, et chaque eid rejete range (mesures
	// bis 1, `campagne_bis1_research_test.go`).
	parRegion map[string]map[[2]int][]cmLiaison
	rangs     []cmRangDeRejet
}

func nouveauCollecteur(f *cmFilm, b *cmBlocs, cre []types.EquipmentCreation) *cmCollecteur {
	c := &cmCollecteur{f: f, b: b, t: cmTables{}, chunk: -1, oracle: map[[2]int][]cmLiaison{},
		oracleBloc: map[[2]int][]cmLiaison{}, statut: map[[2]int]bool{},
		creations: map[uint32][]types.EquipmentCreation{}, chassisLu: map[uint32]uint32{}, queueMin: 4}
	if f.cfg.HasExtraFields {
		c.queueMin += 32
	}
	for _, x := range cre {
		c.creations[x.Slot] = append(c.creations[x.Slot], x)
	}
	return c
}

func (c *cmCollecteur) debutDeChunk(n int, _ []byte, _ []FilmPacket, w *World) {
	c.finirChunk()
	c.chunk, c.paquets = n, nil
	c.liesAuDebut = make(map[uint32]bool, len(w.slots))
	for s := range w.slots {
		c.liesAuDebut[s] = true
	}
}

func (c *cmCollecteur) paquet(_ int, p *cmPaquet, _ *World) {
	c.paquets = append(c.paquets, p)
	c.statut[[2]int{p.d.Chunk, p.d.Index}] = p.d.Fermee
	c.invariants(p)
}

func (c *cmCollecteur) finDeFilm() { c.finirChunk() }

// finirChunk analyse le chunk tamponne.
func (c *cmCollecteur) finirChunk() {
	if len(c.paquets) == 0 {
		return
	}
	c.naissances()
	c.refus()
	c.neufsBordes()
	c.allocateurDesNeufs()
}

// classeDePaquetCM : ferme / hors cadre / autre.
func classeDePaquetCM(p *cmPaquet) string {
	switch {
	case p.d.Fermee:
		return "ferme"
	case p.d.Cause == CauseHorsCadre:
		return "hors cadre"
	}
	return "autre non ferme"
}

// compteDuPaquet : le compte d un paquet.
func compteDuPaquet(p *cmPaquet) cmCompte {
	k := cmCompte{n: 1, paquets: 1, enJeu: p.d.UtilesEnJeu}
	switch classeDePaquetCM(p) {
	case "ferme":
		k.fermes = 1
	case "hors cadre":
		k.horsCadre = 1
	}
	return k
}

// cmOrdreRejet classe un rejet par l ordre de l ecrivain (C4) : R1 si aucun DEL n a ete lu et si
// le slot rejete suit le dernier DELTA lu ; R2 sinon.
func cmOrdreRejet(p *cmPaquet) string {
	slot := p.d.EIDRejete & 0x3fffffff
	del, dernier, deltas := false, uint32(0), 0
	for _, r := range p.recs {
		switch r.Type {
		case recDel:
			del = true
		case recDelta:
			deltas++
			dernier = r.Slot
		}
	}
	switch {
	case del:
		return "R2 (DEL lu avant)"
	case deltas > 0 && slot <= dernier:
		return "R2 (slot <= dernier DELTA)"
	case deltas == 0:
		return "R1 (aucun DELTA lu)"
	}
	return "R1 (slot > dernier DELTA)"
}

// naissances est la sonde M1 et ses ventilations, sur le chunk tamponne.
func (c *cmCollecteur) naissances() {
	neufs := map[uint32]bool{}
	desync := map[uint32]string{}
	rejets := map[uint32]*cmRejet{}
	var ordre []uint32
	for j, p := range c.paquets {
		for _, s := range p.d.NeufsLus {
			neufs[s] = true
		}
		if p.d.Sortie == SortieVueBRejetHorsDatum {
			e := p.d.EIDRejete
			r := rejets[e]
			if r == nil {
				r = &cmRejet{eid: e, premier: j, classe: c.b.naissance(c.chunk, e, neufs[e&0x3fffffff]),
					ordre: cmOrdreRejet(p), desync: desync[e]}
				rejets[e] = r
				ordre = append(ordre, e)
			}
			k := compteDuPaquet(p)
			r.compte.paquets += k.paquets
			r.compte.horsCadre += k.horsCadre
			r.compte.fermes += k.fermes
			r.compte.enJeu += k.enJeu
			c.t.add("rejets_ordre", cmJoindre(classeDePaquetCM(p), cmOrdreRejet(p)), k)
		}
		for _, r := range p.recs {
			if r.Type == recNew && r.DesyncAt >= 0 {
				desync[r.ID] = fmt.Sprintf("ti=%d %s", r.TypeIndex, c.composantDeDesync(r))
			}
		}
	}
	if len(rejets) == 0 {
		return
	}
	pris := map[uint32]bool{}
	cibles := map[uint32]uint32{} // eid cherche -> eid rejete
	for _, e := range ordre {
		r := rejets[e]
		cibles[e] = e
		if tm, ok := c.b.cmTemoin(c.chunk, e, pris); ok {
			r.temoin, r.aTemoin = tm, true
			cibles[tm] = e
		}
	}
	c.chercher(rejets, cibles)
	pred := c.b.predictions(c.chunk)
	for _, e := range ordre {
		c.rangerRejet(rejets[e], pred)
	}
}

// composantDeDesync nomme le composant ou un record a desynchronise.
func (c *cmCollecteur) composantDeDesync(r FrameRecord) string {
	if n := len(r.Trace.Comps); n > 0 {
		return nomComposantBloquant(c.f.reg, int(r.TypeIndex), r.Trace.Comps[n-1].Index)
	}
	return "avant les composants"
}

func classeDeCascade(n int) string {
	switch {
	case n <= 1:
		return "1 paquet"
	case n <= 3:
		return "2-3 paquets"
	case n <= 10:
		return "4-10 paquets"
	case n <= 50:
		return "11-50 paquets"
	}
	return "plus de 50 paquets"
}

// slotFautif : le slot du premier record fautif d un paquet (rejete, ou desynchronise), -1 sinon.
func slotFautif(p *cmPaquet) int64 {
	if p.d.Sortie.EstUnRejet() {
		return int64(p.d.EIDRejete & 0x3fffffff)
	}
	if n := len(p.recs); n > 0 && p.recs[n-1].DesyncAt >= 0 && p.d.Sortie == SortieVueBOuverte {
		return int64(p.recs[n-1].Slot)
	}
	return -1
}

// fautesSuivantes compte les paquets du chunk APRES j dont le premier record fautif porte `slot`.
func (c *cmCollecteur) fautesSuivantes(j int, slot uint32) int {
	n := 0
	for _, q := range c.paquets[j+1:] {
		if slotFautif(q) == int64(slot) {
			n++
		}
	}
	return n
}

// refus juge les NEW refuses (T1-4, C5).
func (c *cmCollecteur) refus() {
	pred := c.b.predictions(c.chunk)
	decl := c.b.declares[c.b.suivant[c.chunk]]
	for j, p := range c.paquets {
		for _, rf := range p.refuses {
			verdict := "indecis"
			if ti, ok := decl[rf.slot]; ok && ti == rf.vivant {
				verdict = "lecture fausse"
			} else if ok && ti == rf.neuf {
				verdict = "creation perdue"
			}
			ordreNeuf, eid := "NEW introuvable dans les records", uint32(0)
			vu := false
			for _, r := range p.recs {
				if r.Type == recNew && r.Slot == rf.slot && r.DesyncAt < 0 {
					eid = r.ID
					ordreNeuf = "groupe NEW respecte"
					if vu {
						ordreNeuf = "NEW apres DELTA/DEL (ordre viole)"
					}
					break
				}
				if r.Type == recDelta || r.Type == recDel {
					vu = true
				}
			}
			bloc := "ni naissance ni occupant au bloc"
			cur, _ := c.b.entree(c.chunk, rf.slot)
			nx, okN := c.b.entree(c.b.suivant[c.chunk], rf.slot)
			gen := uint8(eid >> 30) //nolint:gosec // deux bits
			switch {
			case okN && cmAlloueSous(nx, gen) && !cmAlloueSous(cur, gen):
				bloc = "naissance de cet eid au bloc suivant"
			case cur.Vivante():
				bloc = "occupant vivant au bloc du chunk"
			}
			pool := "pool 0 ou 4"
			if p := cmPoolDe(rf.slot); p >= 1 && p <= 3 {
				pool = "pool 1-3"
			}
			k := cmCompte{n: 1, paquets: c.fautesSuivantes(j, rf.slot)}
			c.t.add("refus", cmJoindre(verdict, ordreNeuf, bloc), k)
			c.t.add("refus_pool", cmJoindre(pool, verdict), k)
			c.t.add("refus_prediction", cmJoindre(verdict, c.b.classeDeRang(c.chunk, pred, eid)), k)
		}
	}
}

// neufsBordes : T8-C4 (d) — NEW lies dont le corps finit dans la queue minimale ou deborde, et les
// paquets suivants du chunk dont le premier record fautif porte leur slot.
func (c *cmCollecteur) neufsBordes() {
	for j, p := range c.paquets {
		refuses := map[uint32]bool{}
		for _, rf := range p.refuses {
			refuses[rf.slot] = true
		}
		for _, r := range p.recs {
			if r.Type != recNew || r.DesyncAt >= 0 || refuses[r.Slot] {
				continue
			}
			bits := p.d.Bits
			var cl string
			switch {
			case r.Trace.EndBit > bits:
				cl = "NEW lie, corps deborde du payload"
			case r.Trace.EndBit > bits-c.queueMin:
				cl = "NEW lie, corps finit dans la queue minimale"
			default:
				continue
			}
			c.t.add("t8_neufs_bordes", cl, cmCompte{n: 1, paquets: c.fautesSuivantes(j, r.Slot)})
		}
	}
}

// allocateurDesNeufs : T1-3 M2 — le rang predit des NEW lus proprement (et du premier par pool).
func (c *cmCollecteur) allocateurDesNeufs() {
	pred := c.b.predictions(c.chunk)
	if pred == nil {
		return
	}
	premier := map[int]bool{}
	for _, p := range c.paquets {
		refuses := map[uint32]bool{}
		for _, rf := range p.refuses {
			refuses[rf.slot] = true
		}
		for _, r := range p.recs {
			if r.Type != recNew || r.DesyncAt >= 0 || refuses[r.Slot] {
				continue
			}
			cl := c.b.classeDeRang(c.chunk, pred, r.ID)
			c.t.un("allocateur_neufs", cl)
			if pool := cmPoolDe(r.Slot); !premier[pool] {
				premier[pool] = true
				c.t.un("allocateur_premier_neuf_du_pool", cl)
			}
		}
	}
}
