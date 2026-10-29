//go:build research

package grammar

// campagne_bis3_p6_research_test.go — MESURES BIS 3 : LES NAISSANCES TROUVEES A PLUS DE 3 PAQUETS
// (P6) ET LES FILTRES QUI LES DEPARTAGERAIENT DU HASARD.
//
// Au-dela de 3 paquets, la sonde M1 ne discrimine plus : le temoin (meme tete, slot jamais alloue)
// y trouve un en-tete NEW aussi souvent que l eid rejete. Une naissance lointaine est pourtant
// plausible (une entite statique n a pas de DELTA). Quatre filtres INDEPENDANTS du hasard de
// l en-tete, appliques a l eid rejete ET a son temoin (meme regle) :
//
//	propre   : le corps NEW se traverse sans desynchronisation dans le paquet ;
//	pont     : le `R(6)` lu = l archetype que le pont du bloc suivant donne au slot (cible commune a
//	           l eid et a son temoin : celle de l eid) ;
//	alloc    : (slot, tete) = une allocation predite par l allocateur (rang < 64, tete (gen+1)&3) ;
//	suivant  : l en-tete qui suit le corps est plausible — NEW de slot plus grand et `R(6)` < 50,
//	           ou DELTA / DEL d un slot alloue sous sa tete au bloc du chunk ou au suivant.
//
// Le taux du temoin est le taux de faux positifs du filtre ; un filtre dont le temoin tombe pres
// de zero rend les naissances lointaines liables, et l oracle « etendu » en mesure la borne.

import "fmt"

// b3Filtres : les combinaisons mesurees ; l oracle etendu est pose pour chacune.
var b3Filtres = []string{"region", "propre", "propre+suivant", "propre+alloc", "propre+pont",
	"propre+alloc+suivant", "propre+pont+suivant"}

// b3Genre nomme un genre de record.
func b3Genre(t int) string {
	switch t {
	case recNew:
		return "NEW"
	case recDel:
		return "DEL"
	case recDelta:
		return "DELTA"
	}
	return fmt.Sprintf("T%d", t)
}

// b3Occ : ce que les filtres disent d une occurrence.
type b3Occ struct {
	j, ti                          int
	region, propre, suivant, alloc bool
}

// b3Satisfait : l occurrence passe la combinaison `f` (cible : l archetype attendu, -1 sans).
func b3Satisfait(o b3Occ, f string, cible int) bool {
	if !o.region {
		return false
	}
	switch f {
	case "region":
		return true
	case "propre":
		return o.propre
	case "propre+suivant":
		return o.propre && o.suivant
	case "propre+alloc":
		return o.propre && o.alloc
	case "propre+pont":
		return o.propre && cible >= 0 && o.ti == cible
	case "propre+alloc+suivant":
		return o.propre && o.alloc && o.suivant
	case "propre+pont+suivant":
		return o.propre && o.suivant && cible >= 0 && o.ti == cible
	}
	return false
}

// p6 evalue les filtres sur les occurrences a plus de 3 paquets d un eid NON lie par l oracle-NEW
// (sa meilleure occurrence n est pas en region non lue a <= 3 paquets), et sur celles de son temoin.
func (d *b3Diag) p6(r *cmRejet, k cmCompte) {
	if best, _, ok := d.x.meilleur(r.trouves, r.eid); best != nil && ok && r.premier-best.j <= 3 {
		return
	}
	cible := -1
	if r.classe == "naissance non lue" {
		if ti, etat := d.b.archetypeDuBloc(d.chunk, r.eid&0x3fffffff); etat == "resolu" {
			cible = int(ti)
		}
	}
	pred := d.pred
	occs := d.occurrences(r.trouves, r.eid, r.premier, pred)
	var occsT []b3Occ
	if r.aTemoin {
		occsT = d.occurrences(r.trouvesT, r.temoin, r.premier, pred)
	}
	base := fmt.Sprintf("cible du pont : %v", cible >= 0)
	d.t.add("p6_base", cmJoindre("rejete", base), k)
	if r.aTemoin {
		d.t.un("p6_base", cmJoindre("temoin", base))
	}
	for _, f := range b3Filtres {
		for _, cl := range []string{"4-10 paquets avant", "plus de 10 paquets avant"} {
			if j := b3Premiere(occs, f, cible, r.premier, cl); j >= 0 {
				d.t.add("p6_filtres", cmJoindre("rejete", f, cl, base), k)
			}
			if r.aTemoin && b3Premiere(occsT, f, cible, r.premier, cl) >= 0 {
				d.t.un("p6_filtres", cmJoindre("temoin", f, cl, base))
			}
		}
		if i := b3Plus(occs, f, cible, r.premier); i >= 0 {
			o := occs[i]
			cle := d.x.cleDOracle(o.j)
			if d.etendu[f] == nil {
				d.etendu[f] = map[[2]int][]cmLiaison{}
			}
			d.etendu[f][cle] = append(d.etendu[f][cle], cmLiaison{eid: r.eid, ti: uint32(o.ti)}) //nolint:gosec // R(6)
			d.t.add("p6_lie", cmJoindre(f, r.classe), k)
		}
	}
}

// b3Premiere : l index (paquet) de la plus proche occurrence qui passe le filtre dans la classe de
// distance `cl`, -1 sinon.
func b3Premiere(occs []b3Occ, f string, cible, premier int, cl string) int {
	best := -1
	for _, o := range occs {
		if premier-o.j <= 3 || classeDeDistance(premier-o.j) != cl || !b3Satisfait(o, f, cible) {
			continue
		}
		if o.j > best {
			best = o.j
		}
	}
	return best
}

// b3Plus : l occurrence la plus proche du rejet (a plus de 3 paquets) qui passe le filtre.
func b3Plus(occs []b3Occ, f string, cible, premier int) int {
	best := -1
	for i, o := range occs {
		if premier-o.j <= 3 || !b3Satisfait(o, f, cible) {
			continue
		}
		if best < 0 || o.j > occs[best].j {
			best = i
		}
	}
	return best
}

// occurrences evalue les filtres sur chaque occurrence relevee.
func (d *b3Diag) occurrences(ts []cmTrouve, eid uint32, premier int, pred map[uint32]int) []b3Occ {
	out := make([]b3Occ, 0, len(ts))
	slot, g := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	alloc := false
	if rg, ok := pred[slot]; ok && rg < cmProfondeurPrediction {
		e, _ := d.b.entree(d.chunk, slot)
		alloc = (e.Gen+1)&3 == g
	}
	for _, t := range ts {
		if t.j >= premier {
			continue
		}
		q := d.paquets[t.j]
		_, ok := d.x.region(q, t.pos, eid)
		o := b3Occ{j: t.j, ti: int(t.ti), region: ok, propre: t.propre, alloc: alloc}
		if ok && t.propre {
			o.suivant = d.suivantPlausible(q.pay, t.pos, slot)
		}
		out = append(out, o)
	}
	return out
}

// suivantPlausible : l en-tete qui suit le corps NEW trouve a `pos` est plausible.
func (d *b3Diag) suivantPlausible(pay []byte, pos int, slot uint32) bool {
	cfg := d.f.cfg
	cfg.Obs = nil
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.SetBitPos(pos + woNewTypeBits + woNewSlotBits + woNewGenBits)
	tr := TraverseEntity(br, d.f.reg, cfg.NewDefaultStateBits)
	if tr.DesyncAt != -1 || tr.EndBit+woNewHeaderBits > len(pay)*8 {
		return false
	}
	br2 := LecteurSur(pay)
	br2.SetBitPos(tr.EndBit)
	typ := readRecordType(br2)
	if typ != recNew && typ != recDelta && typ != recDel {
		return false
	}
	id := readRecordID(br2, cfg.IDLowBits, cfg.IDBase)
	s2, g2 := id&0x3fffffff, uint8(id>>30) //nolint:gosec // deux bits
	if typ == recNew {
		return s2 > slot && br2.ReadBits(woNewTIBits) < objectArchetypeCount
	}
	cur, _ := d.b.entree(d.chunk, s2)
	if cur.Vivante() && cur.Gen == g2 {
		return true
	}
	n, ok := d.b.suivant[d.chunk]
	if !ok {
		return false
	}
	nx, ok := d.b.entree(n, s2)
	return ok && cmAlloueSous(nx, g2)
}
