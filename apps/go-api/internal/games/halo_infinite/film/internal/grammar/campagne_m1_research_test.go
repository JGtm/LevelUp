//go:build research

package grammar

// campagne_m1_research_test.go — MESURES CIBLEES (etape 4) : LA SONDE M1 (T1-1, C1, T1-5) et les
// deux ORACLES de naissance.
//
// Toutes les occurrences d un en-tete NEW de l eid cherche sont relevees dans les paquets du chunk
// anterieurs au premier rejet ; chacune est rangee dans sa REGION, et la plus plausible est
// retenue (region non lue d abord, traversee propre ensuite, la plus proche du rejet enfin). Le
// TEMOIN subit la meme regle : l ECART entre l eid rejete et son temoin, region par region, est ce
// que la sonde mesure au-dela du hasard. Deux validateurs independants du hasard :
//
//	distance  : le nombre de paquets entre la naissance trouvee et le premier rejet (une naissance
//	            vraie precede son premier delta de peu, un faux positif est uniforme) ;
//	archetype : le `R(6)` trouve contre l archetype que le PONT du bloc de type 1 suivant donne
//	            au slot (masque de composants -> archetype, NOTE 5.21 §3.2).
//
// Oracle « NEW »  : l eid est lie, avec le `R(6)` trouve, APRES le paquet de la naissance trouvee.
// Oracle « bloc » : toute « naissance non lue » dont le pont donne l archetype est liee AVANT son
//                   premier paquet rejete — la borne du correctif de T1 sans aucune recherche.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// cmRejet : un eid rejete dans un chunk.
type cmRejet struct {
	eid                   uint32
	premier               int
	compte                cmCompte
	classe, ordre, desync string
	temoin                uint32
	aTemoin               bool
	trouves, trouvesT     []cmTrouve
}

// cmTrouve : un en-tete NEW trouve par la sonde.
type cmTrouve struct {
	j, pos int
	ti     uint32
	propre bool
}

// cmPlafondTrouves borne les occurrences relevees par eid.
const cmPlafondTrouves = 256

// chercher balaie les paquets du chunk a la recherche des en-tetes NEW des eid cibles.
func (c *cmCollecteur) chercher(rejets map[uint32]*cmRejet, cibles map[uint32]uint32) {
	cfgT := c.f.cfg
	cfgT.Obs = nil
	for j, q := range c.paquets {
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
			r := rejets[e]
			if j >= r.premier {
				continue
			}
			ti := uint32(source.BitsTolerants(pay, pos+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)) //nolint:gosec // 6 bits
			if ti >= objectArchetypeCount {
				continue
			}
			tr := cmTrouve{j: j, pos: pos, ti: ti, propre: cmTraverseePropre(pay, pos, c.f.reg, cfgT)}
			if eid == e && len(r.trouves) < cmPlafondTrouves {
				r.trouves = append(r.trouves, tr)
			} else if eid != e && len(r.trouvesT) < cmPlafondTrouves {
				r.trouvesT = append(r.trouvesT, tr)
			}
		}
	}
}

// cmTraverseePropre : le corps NEW a `pos` se traverse sans desynchronisation dans le paquet.
func cmTraverseePropre(pay []byte, pos int, reg *Registry, cfg FrameConfig) bool {
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.SetBitPos(pos + woNewTypeBits + woNewSlotBits + woNewGenBits)
	tr := TraverseEntity(br, reg, cfg.NewDefaultStateBits)
	return tr.DesyncAt == -1 && tr.EndBit <= len(pay)*8
}

// region range une position trouvee dans un paquet anterieur ; le booleen dit qu elle est dans
// une region que la marche N A PAS LUE (une naissance y est possible).
func (c *cmCollecteur) region(q *cmPaquet, pos int, eid uint32) (string, bool) {
	if q.d.ListeNonLocalisee {
		return "(ii) paquet a evenements non localise", true
	}
	if q.strict != -2 && pos < q.debut {
		return "(i) tete d un paquet a evenements, avant le debut", true
	}
	for _, r := range q.recs {
		if r.Type != recNew || r.ID != eid {
			continue
		}
		if r.DesyncAt >= 0 {
			return "(iii) NEW lu desynchronise", true
		}
		for _, rf := range q.refuses {
			if rf.slot == r.Slot {
				return "(iv) NEW lu refuse", true
			}
		}
		return "NEW lu et lie", false
	}
	switch {
	case q.d.DebutVueB < 0:
		return "vue B non atteinte (vue A non portee)", true
	case pos < q.d.DebutVueB:
		return "dans la vue A (faux positif probable)", false
	case pos >= q.d.FinVueB:
		ok := q.d.Sortie != SortieVueBTerminateur
		return "(iii') apres l arret de la vue B : " + q.d.Sortie.String(), ok
	}
	return "dans un record lu (faux positif probable)", false
}

// meilleur retient l occurrence la plus plausible : region non lue, traversee propre, la plus
// proche du rejet.
func (c *cmCollecteur) meilleur(ts []cmTrouve, eid uint32) (*cmTrouve, string, bool) {
	var best *cmTrouve
	bestReg, bestOK := "", false
	score := func(t cmTrouve, ok bool) int {
		s := t.j
		if t.propre {
			s += 1 << 20
		}
		if ok {
			s += 1 << 21
		}
		return s
	}
	for i := range ts {
		reg, ok := c.region(c.paquets[ts[i].j], ts[i].pos, eid)
		if best == nil || score(ts[i], ok) > score(*best, bestOK) {
			best, bestReg, bestOK = &ts[i], reg, ok
		}
	}
	return best, bestReg, bestOK
}

// classeDeDistance classe le nombre de paquets entre une naissance trouvee et le premier rejet.
func classeDeDistance(d int) string {
	switch {
	case d <= 1:
		return "paquet precedent"
	case d <= 3:
		return "2-3 paquets avant"
	case d <= 10:
		return "4-10 paquets avant"
	}
	return "plus de 10 paquets avant"
}

// cleDOracle : la liaison se pose APRES le paquet `j` du chunk (-1 : au debut du chunk).
func (c *cmCollecteur) cleDOracle(j int) [2]int {
	if j < 0 {
		return [2]int{c.chunk, -1}
	}
	return [2]int{c.chunk, c.paquets[j].pk.Index}
}

// rangerRejet range un eid rejete dans les tables et pose ses liaisons d oracle.
func (c *cmCollecteur) rangerRejet(r *cmRejet, pred map[uint32]int) {
	k := r.compte
	k.n = 1
	tiBloc, etatBloc := uint32(0), "pas une naissance non lue"
	if r.classe == "naissance non lue" {
		tiBloc, etatBloc = c.b.archetypeDuBloc(c.chunk, r.eid&0x3fffffff)
		c.t.add("oracle_bloc", etatBloc, k)
		if etatBloc == "resolu" {
			cle := c.cleDOracle(r.premier - 1)
			c.oracleBloc[cle] = append(c.oracleBloc[cle], cmLiaison{eid: r.eid, ti: tiBloc})
		}
	}
	best, reg, ok := c.meilleur(r.trouves, r.eid)
	if best == nil {
		reg = "(v) introuvable"
	}
	suffixe := ""
	if best != nil {
		suffixe = " · traversee non propre"
		if best.propre {
			suffixe = " · traversee propre"
		}
		dist := classeDeDistance(r.premier - best.j)
		c.t.add("m1_region_x_distance", cmJoindre("rejete", reg, dist), k)
		if ok && r.premier-best.j <= 3 {
			c.t.add("m1_ti_naissance", fmt.Sprintf("ti=%d", best.ti), k)
		}
		c.t.add("m1_distance", cmJoindre("rejete", dist, fmt.Sprintf("region non lue : %v", ok)), k)
		if etatBloc == "resolu" {
			acc := "R(6) = archetype du pont"
			if best.ti != tiBloc {
				acc = "R(6) != archetype du pont"
			}
			c.t.add("m1_archetype", cmJoindre(acc, fmt.Sprintf("region non lue : %v", ok), dist), k)
		}
		if ok && r.premier-best.j <= 3 {
			cle := c.cleDOracle(best.j)
			c.oracle[cle] = append(c.oracle[cle], cmLiaison{eid: r.eid, ti: best.ti})
			c.lierParRegion(reg, cle, cmLiaison{eid: r.eid, ti: best.ti})
		}
	}
	c.t.add("m1_regions", reg+suffixe, k)
	c.t.add("m1_naissance_x_region", cmJoindre(r.classe, reg), k)
	if r.aTemoin {
		tb, treg, tok := c.meilleur(r.trouvesT, r.temoin)
		if tb == nil {
			treg = "(v) introuvable"
		} else {
			c.t.un("m1_distance", cmJoindre("temoin", classeDeDistance(r.premier-tb.j),
				fmt.Sprintf("region non lue : %v", tok)))
			c.t.un("m1_region_x_distance", cmJoindre("temoin", treg, classeDeDistance(r.premier-tb.j)))
		}
		c.t.un("m1_temoin_x_rejete", cmJoindre("rejete", reg))
		c.t.un("m1_temoin_x_rejete", cmJoindre("temoin", treg))
	}
	plaus := c.b.plausibilite(r.classe, r.eid)
	c.t.add("plausibilite", plaus, k)
	c.t.add("plausibilite_x_ordre", cmJoindre(plaus, r.ordre), k)
	nl := "autre classe"
	if r.classe == "naissance non lue" {
		nl = "naissance non lue"
	}
	c.t.add("allocateur_rejets", cmJoindre(nl, c.b.classeDeRang(c.chunk, pred, r.eid)), k)
	if r.desync != "" {
		c.t.add("t7_5_neuf_desynchronise_anterieur", r.desync, k)
	}
	if r.classe == "vivant au bloc du chunk" {
		lie := "non lie au debut du chunk"
		if c.liesAuDebut[r.eid&0x3fffffff] {
			lie = "lie au debut du chunk"
		}
		_, decl := c.b.declares[c.chunk][r.eid&0x3fffffff]
		c.t.add("vivant_au_bloc", cmJoindre(lie, fmt.Sprintf("image-cle declare le slot : %v", decl)), k)
	}
	c.t.add("cascade", classeDeCascade(r.compte.paquets), k)
	c.t.add("naissance", r.classe, k)
	c.noterRang(r, best, reg, ok)
}
