//go:build research

package grammar

// r_nais_pied_research_test.go — CHANTIER « NAIS » : R-L1 (b), LES PAQUETS FERMES APRES UN REJET
// (D-2, D-56), et R-L1 (a), CE QUE LA VUE B AVAIT LU AVANT L EN-TETE REJETE DES PAQUETS (iii').
//
// (b) Une ligne par paquet ferme dont la vue B s est arretee sur un en-tete rejete : l en-tete
// (slot, tete, classe de naissance, plausibilite), ce que la vue B avait lu (NEW / DELTA / DEL et
// le genre du dernier), le reste du payload derriere l en-tete, la vue C, le verdict du juge des
// invariants de l ecrivain ([cmContredit]) et les bits de la queue du paquet, de 66 bits avant
// l en-tete rejete (un DEL fait 3 + 15 + 32 bits) jusqu a la fin, `|` aux bornes de l en-tete.

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// rnLargeurEnTete : prefixe DELTA (1) + identifiant (IDLowBits) + tete (2).
func rnLargeurEnTete(cfg FrameConfig) int { return 1 + cfg.IDLowBits + largeurTagDeGeneration }

// rnBits rend les bits [a, b) du payload, `|` avant chaque position de `marques`.
func rnBits(pay []byte, a, b int, marques ...int) string {
	var s strings.Builder
	for i := max(a, 0); i < b && i < len(pay)*8; i++ {
		for _, m := range marques {
			if i == m {
				s.WriteByte('|')
			}
		}
		if source.BitsTolerants(pay, i, 1) != 0 {
			s.WriteByte('1')
		} else {
			s.WriteByte('0')
		}
	}
	return s.String()
}

// rnGenre nomme un genre de record (vide : aucun).
func rnGenre(t int) string {
	switch t {
	case recNew:
		return "NEW"
	case recDelta:
		return "DELTA"
	case recDel:
		return "DEL"
	case -1:
		return "aucun"
	}
	return fmt.Sprintf("T%d", t)
}

// rnClasseReste classe le reste du payload derriere l en-tete rejete.
func rnClasseReste(n int) string {
	switch {
	case n <= 8:
		return "reste <= 8"
	case n <= 16:
		return "reste 9-16"
	case n <= 64:
		return "reste 17-64"
	}
	return "reste > 64"
}

// rnLu resume ce que la vue B a lu avant le rejet.
func rnLu(q rnQ) string {
	switch {
	case q.neufs+q.deltas+q.dels == 0:
		return "rien lu"
	case q.deltas == 0 && q.dels == 0:
		return "NEW seulement"
	case q.deltas > 0 && q.dels == 0:
		return "NEW* DELTA+"
	}
	return "DEL lu"
}

// ligneDePied : la ligne (b) d un paquet ferme apres un rejet.
func (a *rnP1) ligneDePied(p *cmPaquet, q rnQ) string {
	cfg := a.f.cfg
	h := rnLargeurEnTete(cfg)
	debutEnTete := p.d.FinVueB - h
	reste := p.d.Bits - p.d.FinVueB
	contre := strings.Join(cmContredit(a.chk, p), "+")
	if contre == "" {
		contre = "sain"
	}
	valDel := "-"
	if q.dernier == recDel && debutEnTete-32 >= 0 {
		valDel = fmt.Sprintf("%08x", source.BitsTolerants(p.pay, debutEnTete-32, 32))
	}
	a.t.un("b_pied", cmJoindre(contre, rnLu(q), "dernier "+rnGenre(q.dernier), rnClasseReste(reste),
		fmt.Sprintf("eid nul : %v", q.eid == 0), q.classe))
	return rnTab(a.f.id, a.f.build, a.chunk, p.d.Index, p.d.Bits, p.d.FinVueB, reste,
		q.eid&0x3fffffff, q.eid>>30, q.classe, a.b.plausibilite(q.classe, q.eid), q.neufs, q.deltas, q.dels,
		rnGenre(q.dernier), valDel, len(p.d.VueC.Entrees), p.d.VueC.Vide, contre,
		rnBits(p.pay, debutEnTete-66, p.d.Bits, debutEnTete, p.d.FinVueB))
}

// rnTetePied : l en-tete du TSV (b).
const rnTetePied = "film\tbuild\tchunk\tpaquet\tbits\tfin_vue_b\treste_apres_entete\tslot_rejete\ttete\t" +
	"classe\tplausibilite\tneufs_lus\tdeltas_lus\tdels_lus\tdernier\tvaleur_del\tentrees_vue_c\t" +
	"vue_c_vide\tjuge\tqueue_bits"

// tablesDesRejets : R-L1 (b) temoin — pour TOUT paquet a rejet hors datum de la reference, le
// reste derriere l en-tete et la fermeture : si la fermeture apres rejet n arrive que quand
// l en-tete finit dans la queue de zeros du paquet, elle est un effet de position.
func (a *rnP1) tablesDesRejets() {
	for _, q := range a.qInfo {
		a.t.un("b_rejets_reste_x_ferme", cmJoindre(rnClasseReste(q.bits-q.finVue), fmt.Sprintf("ferme : %v", q.ferme)))
	}
}

// aDescripteurs : R-L1 (a) — pour chaque liaison (iii') apres rejet (meilleure occurrence de
// l eid en region (iii') rejet, a <= 3 paquets), ce que la vue B du paquet Q avait lu avant son
// en-tete rejete X, la classe de X, et si le paquet Q se FERME, sain, lu depuis l occurrence.
func (a *rnP1) aDescripteurs() {
	for i := range a.cibles {
		r := &a.cibles[i]
		if !r.lie || a.occs[r.meilleure].region != rnRegionRejet {
			continue
		}
		o := a.occs[r.meilleure]
		q := a.qInfo[[2]int{o.chunk, o.pkIndex}]
		k := cmCompte{n: 1, paquets: r.nPaq, horsCadre: r.horsCadre, fermes: r.fermes}
		ferme := "Q ne se ferme pas depuis l occurrence"
		if o.sain {
			ferme = "Q se ferme, sain, depuis l occurrence"
		} else if o.ferme {
			ferme = "Q se ferme, contredit, depuis l occurrence"
		}
		x := "X " + q.classe
		a.t.add("a_liaisons", cmJoindre(rnLu(q), x, ferme), k)
		a.t.add("a_liaisons_lu", rnLu(q), k)
		a.t.add("a_liaisons_classe_x", x, k)
		a.t.add("a_liaisons_ferme", ferme, k)
		a.t.add("a_liaisons_q_ferme_en_ref", fmt.Sprintf("Q ferme en reference : %v", q.ferme), k)
		a.t.add("a_liaisons_decalage", cmJoindre(x, rnClasseDecalage(o.pos-q.finVue)), k)
		pont := "pont non resolu"
		if r.pont >= 0 {
			pont = fmt.Sprintf("R(6) = archetype du pont : %v", int(o.ti) == r.pont)
		}
		a.t.add("a_liaisons_pont", cmJoindre(x, pont), k)
	}
	for _, o := range a.occs {
		if o.region != rnRegionRejet || o.dist > 3 {
			continue
		}
		qui := "rejete"
		if o.temoin {
			qui = "temoin"
		}
		etat := "non ferme"
		if o.sain {
			etat = "ferme sain"
		} else if o.ferme {
			etat = "ferme contredit"
		}
		a.t.un("a_occurrences_iii_rejet", cmJoindre(qui, etat))
		if q, ok := a.qInfo[[2]int{o.chunk, o.pkIndex}]; ok {
			a.t.un("a_occurrences_iii_rejet_decalage", cmJoindre(qui, rnClasseDecalage(o.pos-q.finVue)))
		}
	}
}

// rnClasseDecalage classe la distance, en bits, entre la fin de l en-tete rejete du paquet Q et
// l occurrence trouvee derriere lui.
func rnClasseDecalage(n int) string {
	switch {
	case n < 0:
		return "avant la fin de l en-tete"
	case n < 16:
		return "decalage 0-15"
	case n < 64:
		return "decalage 16-63"
	case n < 256:
		return "decalage 64-255"
	}
	return "decalage >= 256"
}
