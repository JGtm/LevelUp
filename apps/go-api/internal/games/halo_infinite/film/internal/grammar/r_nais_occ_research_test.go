//go:build research

package grammar

// r_nais_occ_research_test.go — CHANTIER « NAIS » : LA PASSE 1 (marche de reference).
//
// Elle refait, chunk par chunk, ce que la sonde M1 fait (`campagne_naissances_research_test.go`,
// `campagne_m1_research_test.go`) — memes premiers rejets, memes temoins ([cmBlocs.cmTemoin], meme
// ordre), meme recherche d en-tetes NEW, memes regions ([cmCollecteur.region]), meme meilleure
// occurrence ([cmCollecteur.meilleur]) — et GARDE chaque occurrence (eid rejete ET temoin), pour
// que la passe 2 la juge sur le monde d avant son paquet : « le paquet se ferme-t-il, sain, si
// on le lit depuis cette occurrence ? » (le filtre d OCCURRENCE de R-P6 et de R-L1 (a)).
//
// Elle range aussi, par paquet a rejet, ce que la vue B a lu avant l en-tete rejete (R-L1 (a)),
// le detail de chaque paquet ferme apres un rejet (R-L1 (b), `r_nais_pied_research_test.go`), et le
// score de l allocateur par chunk (R-L1 (d)).

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// rnRegionRejet est la region (iii') « apres un rejet » de M1.
const rnRegionRejet = "(iii') apres l arret de la vue B : rejet hors datum"

// rnOcc : une occurrence d en-tete NEW d un eid cherche, dans un paquet anterieur a son premier rejet.
type rnOcc struct {
	chunk, j, pkIndex, pos, dist int
	eid                          uint32
	temoin                       bool
	ti                           uint32
	region                       string
	ok, propre, suivant, alloc   bool
	evaluee, ferme, sain         bool
}

// rnCible : un eid rejete (premier rejet du chunk) et son temoin.
type rnCible struct {
	eid, temoin             uint32
	aTemoin                 bool
	chunk, premier          int
	classe                  string
	occs, occsT             []int
	meilleure, meilleureT   int
	lie                     bool
	pont                    int
	horsCadre, fermes, nPaq int
}

// rnQ : ce que la vue B d un paquet a rejet a lu avant l en-tete rejete.
type rnQ struct {
	eid                               uint32
	classe                            string
	neufs, deltas, dels, finVue, bits int
	dernier                           int
	ferme                             bool
}

// rnP1 ecoute la marche de reference.
type rnP1 struct {
	f         *cmFilm
	b         *cmBlocs
	x, chk    *cmCollecteur
	chunk     int
	paquets   []*cmPaquet
	neufs     map[uint32]bool
	pred      map[uint32]int
	occs      []rnOcc
	cibles    []rnCible
	parPaquet map[[2]int][]int
	qInfo     map[[2]int]rnQ
	pied      []string
	t         cmTables
	score     map[int][4]int
}

func rnNouveauP1(f *cmFilm, b *cmBlocs) *rnP1 {
	return &rnP1{f: f, b: b, x: &cmCollecteur{f: f, b: b, t: cmTables{}}, chk: nouveauCollecteur(f, b, nil),
		chunk: -1, parPaquet: map[[2]int][]int{}, qInfo: map[[2]int]rnQ{}, t: cmTables{}, score: map[int][4]int{}}
}

func (a *rnP1) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) {
	a.finir()
	a.chunk, a.paquets, a.neufs = c, nil, map[uint32]bool{}
	a.pred = a.b.predictions(c)
	a.chk.chunk = c
}

func (a *rnP1) finDeFilm() { a.finir() }

func (a *rnP1) paquet(c int, p *cmPaquet, _ *World) {
	a.paquets = append(a.paquets, p)
	for _, s := range p.d.NeufsLus {
		a.neufs[s] = true
	}
	a.compterScore(c, p)
	if !p.d.Sortie.EstUnRejet() {
		return
	}
	q := rnQ{eid: p.d.EIDRejete, finVue: p.d.FinVueB, bits: p.d.Bits, ferme: p.d.Fermee, dernier: -1}
	q.classe = a.b.naissance(c, q.eid, a.neufs[q.eid&0x3fffffff])
	for _, r := range p.recs {
		switch r.Type {
		case recNew:
			q.neufs++
		case recDelta:
			q.deltas++
		case recDel:
			q.dels++
		}
		q.dernier = r.Type
	}
	a.qInfo[[2]int{c, p.d.Index}] = q
	if p.d.Fermee {
		a.pied = append(a.pied, a.ligneDePied(p, q))
	}
}

// compterScore : R-L1 (d) — les NEW lus proprement par la marche de reference, et ceux que
// l allocateur a cinq pools predit (rang < 64, tete predite) : [0, 1] sur tous les paquets, [2, 3]
// sur les seuls paquets SANS liste d evenements (le localisateur n y joue pas).
func (a *rnP1) compterScore(c int, p *cmPaquet) {
	refuses := map[uint32]bool{}
	for _, rf := range p.refuses {
		refuses[rf.slot] = true
	}
	s := a.score[c]
	for _, r := range p.recs {
		if r.Type != recNew || r.DesyncAt >= 0 || refuses[r.Slot] {
			continue
		}
		predit := false
		if rang, ok := a.pred[r.Slot]; ok && rang < cmProfondeurPrediction {
			e, _ := a.b.entree(c, r.Slot)
			predit = (e.Gen+1)&3 == uint8(r.ID>>30) //nolint:gosec // deux bits
		}
		s[0]++
		if predit {
			s[1]++
		}
		if p.strict == -2 {
			s[2]++
			if predit {
				s[3]++
			}
		}
	}
	a.score[c] = s
}

// finir analyse le chunk tamponne : premiers rejets, temoins, occurrences.
func (a *rnP1) finir() {
	if len(a.paquets) == 0 {
		return
	}
	a.x.paquets = a.paquets
	neufs := map[uint32]bool{}
	idx := map[uint32]int{}
	var ordre []uint32
	for j, p := range a.paquets {
		for _, s := range p.d.NeufsLus {
			neufs[s] = true
		}
		if p.d.Sortie != SortieVueBRejetHorsDatum {
			continue
		}
		e := p.d.EIDRejete
		k, vu := idx[e]
		if !vu {
			k = len(a.cibles)
			idx[e] = k
			ordre = append(ordre, e)
			a.cibles = append(a.cibles, rnCible{eid: e, chunk: a.chunk, premier: j, meilleure: -1, meilleureT: -1,
				pont: -1, classe: a.b.naissance(a.chunk, e, neufs[e&0x3fffffff])})
		}
		r := &a.cibles[k]
		r.nPaq++
		if p.d.Fermee {
			r.fermes++
		} else if p.d.Cause == CauseHorsCadre {
			r.horsCadre++
		}
	}
	if len(ordre) == 0 {
		return
	}
	pris := map[uint32]bool{}
	cibles := map[uint32]uint32{}
	for _, e := range ordre {
		r := &a.cibles[idx[e]]
		cibles[e] = e
		if tm, ok := a.b.cmTemoin(a.chunk, e, pris); ok {
			r.temoin, r.aTemoin = tm, true
			cibles[tm] = e
		}
	}
	a.chercher(idx, cibles)
	for _, e := range ordre {
		a.conclure(&a.cibles[idx[e]])
	}
}

// chercher est [cmCollecteur.chercher] : memes positions, meme plafond, memes champs, gardes.
func (a *rnP1) chercher(idx map[uint32]int, cibles map[uint32]uint32) {
	cfgT := a.f.cfg
	cfgT.Obs = nil
	for j, q := range a.paquets {
		pay := q.pay
		fin := len(pay)*8 - woNewHeaderBits
		for pos := 0; pos <= fin; pos++ {
			if source.BitsTolerants(pay, pos, 1) != 0 || source.BitsTolerants(pay, pos+1, 2) != 1 {
				continue
			}
			h := LireHandle(pay, pos+woNewTypeBits)
			eid := h.Gen<<30 | h.Slot
			e, ok := cibles[eid]
			if !ok {
				continue
			}
			r := &a.cibles[idx[e]]
			if j >= r.premier {
				continue
			}
			ti := uint32(source.BitsTolerants(pay, pos+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)) //nolint:gosec // 6 bits
			if ti >= objectArchetypeCount {
				continue
			}
			temoin := eid != e
			if (!temoin && len(r.occs) >= cmPlafondTrouves) || (temoin && len(r.occsT) >= cmPlafondTrouves) {
				continue
			}
			o := rnOcc{chunk: a.chunk, j: j, pkIndex: q.pk.Index, pos: pos, dist: r.premier - j, eid: eid,
				temoin: temoin, ti: ti, propre: cmTraverseePropre(pay, pos, a.f.reg, cfgT)}
			o.region, o.ok = a.x.region(q, pos, eid)
			if o.ok && o.propre {
				o.suivant = a.suivantPlausible(pay, pos, h.Slot)
			}
			o.alloc = a.allocPredit(eid)
			a.occs = append(a.occs, o)
			if temoin {
				r.occsT = append(r.occsT, len(a.occs)-1)
			} else {
				r.occs = append(r.occs, len(a.occs)-1)
			}
		}
	}
}

// allocPredit : (slot, tete) est une allocation predite (rang < 64, tete (gen+1)&3) — le filtre
// « alloc » de `campagne_bis3_p6_research_test.go`, propriete de l EID.
func (a *rnP1) allocPredit(eid uint32) bool {
	slot, g := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	if rg, ok := a.pred[slot]; ok && rg < cmProfondeurPrediction {
		e, _ := a.b.entree(a.chunk, slot)
		return (e.Gen+1)&3 == g
	}
	return false
}

// suivantPlausible est [b3Diag.suivantPlausible] sur le chunk courant.
func (a *rnP1) suivantPlausible(pay []byte, pos int, slot uint32) bool {
	d := &b3Diag{f: a.f, b: a.b, chunk: a.chunk}
	return d.suivantPlausible(pay, pos, slot)
}

// conclure : meilleure occurrence (regle de M1), liaison de l oracle-NEW, occurrences a juger.
func (a *rnP1) conclure(r *rnCible) {
	meilleur := func(ids []int, eid uint32) int {
		ts := make([]cmTrouve, len(ids))
		for i, k := range ids {
			o := a.occs[k]
			ts[i] = cmTrouve{j: o.j, pos: o.pos, ti: o.ti, propre: o.propre}
		}
		best, _, _ := a.x.meilleur(ts, eid)
		if best == nil {
			return -1
		}
		for i := range ts {
			if &ts[i] == best {
				return ids[i]
			}
		}
		return -1
	}
	r.meilleure, r.meilleureT = meilleur(r.occs, r.eid), meilleur(r.occsT, r.temoin)
	if k := r.meilleure; k >= 0 && a.occs[k].ok && a.occs[k].dist <= 3 {
		r.lie = true
	}
	if r.classe == "naissance non lue" {
		if ti, etat := a.b.archetypeDuBloc(a.chunk, r.eid&0x3fffffff); etat == "resolu" {
			r.pont = int(ti)
		}
	}
	for _, ids := range [][]int{r.occs, r.occsT} {
		for _, k := range ids {
			if o := a.occs[k]; o.ok {
				cle := [2]int{o.chunk, o.j}
				a.parPaquet[cle] = append(a.parPaquet[cle], k)
			}
		}
	}
}
