//go:build research && campagne_overlay

package grammar

// r_comp_p3_research_test.go — CHANTIER « comp », R-P3 (plan de la campagne §6.1, D-56) : les
// coupables residuels des decalages de curseur. La population P3 (MESURES_BIS_3 §3.2) est celle
// des en-tetes rejetes « aucune allocation » ; le DERNIER composant lu avant l en-tete rejete
// designe le record dont la lecture a decale le curseur. Quatre coupables restaient sans cause :
//
//	G1  DELTA ti=20 dernier composant i1 spawn-filter-weight-component   694 paquets hors cadre
//	G2  DEL ti=0 dernier composant aucun composant                        658
//	G3  DELTA ti=14 dernier composant i1 crew-marked-objects-component    241
//	G4  DELTA ti=41 dernier composant i2 object-forward-and-up-component  161
//
// Deux mesures :
//  1. DIAGNOSTIC (marche de reference) : pour chaque paquet hors cadre sorti par un rejet, le
//     dernier record lu (genre, archetype, masque, composants et leurs largeurs lues), le record
//     d avant, et pour un DEL la classe d allocation de son eid ;
//  2. A/B en surcouche (crochet [bis2Intercepteur]) des lectures corrigees, et les quatre comptes
//     du gate (table `dernier_composant_x_classe` de [b3Diag], la meme que MESURES_BIS_3).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -run '^TestRCompP3' ...

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// rp3Gates : les quatre cles du gate (classe « aucune allocation »).
var rp3Gates = []struct{ nom, cle string }{
	{"G1 ti=20 i1", "aucune allocation · DELTA ti=20 dernier composant i1 spawn-filter-weight-component"},
	{"G2 DEL ti=0", "aucune allocation · DEL ti=0 dernier composant aucun composant"},
	{"G3 ti=14 i1", "aucune allocation · DELTA ti=14 dernier composant i1 crew-marked-objects-component"},
	{"G4 ti=41 i2", "aucune allocation · DELTA ti=41 dernier composant i2 object-forward-and-up-component"},
}

// rp3Comps decrit les composants lus d un record : `iN nom (largeur)`.
func rp3Comps(r FrameRecord) string {
	var parts []string
	for k, c := range r.Trace.Comps {
		fin := r.Trace.EndBit
		if k+1 < len(r.Trace.Comps) {
			fin = r.Trace.Comps[k+1].StartBit
		}
		parts = append(parts, fmt.Sprintf("i%d(%d)", c.Index, fin-c.StartBit))
	}
	if len(parts) == 0 {
		return "aucun composant"
	}
	return strings.Join(parts, "+")
}

// rp3Diag : le diagnostic par paquet hors cadre sorti par un rejet « aucune allocation ». Il
// construit aussi l ORACLE DU PONT : pour un record dont le masque est impossible sous l archetype
// que le monde lui donne, l archetype que le bloc de type 1 du chunk designe (pont masque de
// composants -> archetype, [cmBlocs.pontDuBloc]) est pose sur l eid juste avant le paquet.
type rp3Diag struct {
	f     *cmFilm
	b     *cmBlocs
	t     cmTables
	chunk int
	prec  int // index du paquet delta precedent dans le chunk (-1 : debut du chunk)
	v     rp3Variante
	// desavoues : les liaisons retirees par les variantes de desaveu.
	desavoues int
	oracle    map[[2]int][]cmLiaison
	// imp : les eid dont le masque est impossible, par paquet precedent (oracles « archetype force »).
	imp map[[2]int][]uint32
}

func (d *rp3Diag) debutDeChunk(c int, _ []byte, _ []FilmPacket, w *World) {
	d.chunk, d.prec = c, -1
	if d.v.desaveuKF {
		for s, ti := range d.b.declares[c] {
			if e, ok := d.b.entree(c, s); ok && !e.Vivante() {
				if x, lie := w.slots[s]; lie && x.TypeIndex == ti && x.GenAny && !x.Soft {
					w.Unbind(s)
					d.desavoues++
				}
			}
		}
	}
}
func (d *rp3Diag) finDeFilm() {}

// rp3MasqueImpossible : le masque porte un bit au-dela des composants de l archetype.
func rp3MasqueImpossible(reg *Registry, r FrameRecord) bool {
	a, ok := reg.Archetype(int(r.TypeIndex))
	return ok && r.Trace.Mask>>uint(min(len(a.Components), 63)) != 0
}

// pont classe un record contre le bloc de type 1 du chunk, et pose l oracle quand le masque est
// impossible et que le pont designe un autre archetype qui le rend possible.
func (d *rp3Diag) pont(p *cmPaquet, r FrameRecord) string {
	imp := "masque possible"
	if rp3MasqueImpossible(d.f.reg, r) {
		d.t.add("liaison_du_masque_impossible", d.liaison(r), cmCompte{n: 1, paquets: 1})
		imp = "masque IMPOSSIBLE sous l archetype lie"
		d.imp[[2]int{d.chunk, d.prec}] = append(d.imp[[2]int{d.chunk, d.prec}], r.ID)
	}
	e, ok := d.b.entree(d.chunk, r.Slot)
	if !ok {
		return cmJoindre(imp, "slot hors du bloc")
	}
	ti, st := d.b.pontDuBloc(e)
	if st != "resolu" {
		return cmJoindre(imp, "pont : "+st)
	}
	verdict := fmt.Sprintf("pont ti=%d", ti)
	if ti == r.TypeIndex {
		verdict += " (= archetype lie)"
	} else {
		verdict += fmt.Sprintf(" (archetype lie ti=%d)", r.TypeIndex)
		a, okA := d.f.reg.Archetype(int(ti))
		if okA && r.Trace.Mask>>uint(min(len(a.Components), 63)) == 0 {
			cle := [2]int{d.chunk, d.prec}
			d.oracle[cle] = append(d.oracle[cle], cmLiaison{eid: r.ID, ti: ti})
			verdict += " ; masque possible sous le pont"
		}
	}
	return cmJoindre(imp, verdict)
}

func (d *rp3Diag) paquet(_ int, p *cmPaquet, w *World) {
	rp3Monde = w
	if d.v.newImpossible {
		for _, r := range p.recs {
			if x, lie := w.slots[r.Slot]; r.Type == recNew && rp3MasqueImpossible(d.f.reg, r) && lie && x.FullID == r.ID {
				w.Unbind(r.Slot)
				d.desavoues++
			}
		}
	}
	defer func() { d.prec = p.pk.Index }()
	if p.d.Fermee || p.d.Cause != CauseHorsCadre || p.d.Sortie != SortieVueBRejetHorsDatum || len(p.recs) == 0 {
		return
	}
	r := p.recs[len(p.recs)-1]
	if d.b.naissance(d.chunk, p.d.EIDRejete, false) != "aucune allocation" {
		return
	}
	k := cmCompte{n: 1, paquets: 1, horsCadre: 1, enJeu: p.d.UtilesEnJeu}
	switch {
	case r.Type == recDelta && r.TypeIndex == 20:
		tag := "i0 absent"
		for _, c := range r.Trace.Comps {
			if c.Index == 0 {
				tag = fmt.Sprintf("i0 cas %d", source.BitsBourres(p.pay, c.StartBit, 2))
			}
		}
		d.t.add("g1_ti20", cmJoindre(fmt.Sprintf("masque %#x", r.Trace.Mask), tag, rp3Comps(r)), k)
		d.t.add("g1_g3_slot", d.slotDuFilm(r), k)
		d.t.add("g1_ti20_pont", d.pont(p, r), k)
	case r.Type == recDel:
		_, lie := w.ArchetypeForSlot(r.Slot)
		d.t.add("g2_del", cmJoindre("eid du DEL : "+d.b.naissance(d.chunk, r.ID, false),
			fmt.Sprintf("lie au monde apres le paquet : %v", lie)), k)
		d.t.add("g2_del_nombre_de_records", fmt.Sprintf("%d records lus", len(p.recs)), k)
		n := len(p.recs) - 2
		for n >= 0 && p.recs[n].Type == recDel {
			n--
		}
		if n < 0 {
			d.t.add("g2_del_premier_non_del_avant", "aucun (que des DEL)", k)
			break
		}
		q := p.recs[n]
		cle := fmt.Sprintf("%s ti=%d %s", b3Genre(q.Type), q.TypeIndex, rp3Comps(q))
		if q.Type == recDelta {
			cle = cmJoindre(cle, d.pont(p, q))
		}
		d.t.add("g2_del_premier_non_del_avant", cmJoindre(fmt.Sprintf("%d DEL", len(p.recs)-1-n), cle), k)
	case r.Type == recDelta && r.TypeIndex == 14:
		d.t.add("g3_ti14", cmJoindre(fmt.Sprintf("masque %#x", r.Trace.Mask), rp3Comps(r)), k)
		d.t.add("g3_ti14_pont", d.pont(p, r), k)
		d.t.add("g1_g3_slot", d.slotDuFilm(r), k)
	case r.Type == recDelta && r.TypeIndex == 41 && len(r.Trace.Comps) > 0 && r.Trace.Comps[len(r.Trace.Comps)-1].Index == 2:
		d.t.add("g4_ti41", cmJoindre(fmt.Sprintf("masque %#x", r.Trace.Mask), rp3Comps(r),
			fmt.Sprintf("i0 tete %04b", source.BitsBourres(p.pay, r.Trace.Comps[0].StartBit, 4))), k)
		d.t.add("g4_ti41_pont", d.pont(p, r), k)
	default:
		return
	}
	d.t.add("genre", fmt.Sprintf("%s ti=%d", b3Genre(r.Type), r.TypeIndex), k)
}

// rp3Variante : une lecture corrigee mesuree par la surcouche.
type rp3Variante struct {
	nom    string
	oracle bool   // pose l oracle du pont construit par la marche de reference
	force  uint32 // > 0 : lie les eid a masque impossible a cet archetype (oracle « archetype force »)
	// desaveuKF : a l ouverture du chunk, retire la liaison d image-cle d un slot que le bloc de type
	// 1 du MEME chunk dit non vivant (l ecrivain ne declare que des entites allouees).
	desaveuKF bool
	// newImpossible : apres chaque paquet, retire la liaison posee par un NEW a masque impossible.
	newImpossible bool
	crochet       func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool)
}

// rp3LireTypeDeFiltreJeu porte `spawn-filter-type-component` (ti=20 i0, FUN_142ed708c ->
// FUN_142ecf744) tel que le jeu le lit (Ghidra, 2026-10-02) :
//
//	R(2) cas ; 0 : rien ;
//	1 : niveau < 2 -> FUN_1407f2058 = R(1) [0 -> R(5)] ; sinon FUN_142b67e34 = R(9) ;
//	2 : FUN_142b6ee08 = FUN_142b67f08 (R(1) [0 -> R(13)]) puis R(6) ;
//	3 : FUN_142b6eeec = R(32) (FUN_14080dec4) ; FUN_14076e494(0x10) ; R(3) ; R(4) = n ; n x R(32) ;
//	    FUN_1407f1ff4 -> FUN_1407f2058 = R(1) [0 -> R(5)].
//
// Le Go lit le cas 2 par `readQuantStat(1)` (sonde R(1), 13 ou 9 bits, puis DEUX bits de queue).
func rp3LireTypeDeFiltreJeu(br *Lecteur, level uint32) {
	switch br.ReadBits(2) {
	case 0:
	case 1:
		if level < 2 {
			if br.ReadBits(1) == 0 {
				br.ReadBits(5)
			}
		} else {
			br.ReadBits(9)
		}
	case 2:
		if br.ReadBits(1) == 0 {
			br.ReadBits(13)
		}
		br.ReadBits(6)
	default:
		br.ReadBits(32)
		lireE494(br, niveauPosition)
		br.ReadBits(3)
		n := int(br.ReadBits(4))
		for i := 0; i < n; i++ {
			br.ReadBits(32)
		}
		if br.ReadBits(1) == 0 {
			br.ReadBits(5)
		}
	}
}

// rp3Variantes : la reference et les lectures corrigees (une par coupable instruit).
func rp3Variantes() []rp3Variante {
	return []rp3Variante{
		{nom: "reference"},
		{nom: "oracle du pont (masque impossible)", oracle: true},
		{nom: "oracle archetype force ti=35", force: 35},
		{nom: "oracle archetype force ti=40", force: 40},
		{nom: "desaveu image-cle hors bloc", desaveuKF: true},
		{nom: "desaveu NEW a masque impossible", newImpossible: true},
		{nom: "desaveu des deux", desaveuKF: true, newImpossible: true},
		{nom: "ti20 i0 cas 2 du jeu", crochet: func(br *Lecteur, name string, _, level uint32) (bool, bool) {
			if name != "spawn-filter-type-component" {
				return false, false
			}
			rp3LireTypeDeFiltreJeu(br, level)
			return true, true
		}},
	}
}

// TestRCompP3 : diagnostic et A/B ; ecrit `r_comp_p3.tsv` (gate par film et variante) et
// `r_comp_p3_tables.tsv` (diagnostic).
func TestRCompP3(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	tete := "film\tbuild\tvariante\tpaquets\tfermes\tfermes_sains\tutiles_fermes\tutiles_fermes_sains\tutiles_lus\t" +
		"hors_cadre\tgagnes\tperdus\tgagnes_contredits\tperdus_sains"
	for _, g := range rp3Gates {
		tete += "\t" + g.nom
	}
	tete += "\tliaisons_oracle\tliaisons_sur_slot_occupe\tliaisons_desavouees"
	lignes := []string{tete}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"}
	defer func() { bis2Intercepteur = nil }()
	for _, id := range films {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		b := cmLireBlocs(f)
		var jref *cmJuge
		var ref map[[2]int]bool
		var oracleRef map[[2]int][]cmLiaison
		var impRef map[[2]int][]uint32
		for _, v := range rp3Variantes() {
			bis2Intercepteur = v.crochet
			d := b3NouveauDiag(f, b, false)
			diag := &rp3Diag{f: f, b: b, v: v, t: cmTables{}, oracle: map[[2]int][]cmLiaison{}, imp: map[[2]int][]uint32{}}
			var va cmVariante
			if v.oracle {
				va.oracle = oracleRef
			}
			if v.force > 0 {
				va.oracle = map[[2]int][]cmLiaison{}
				for cle, eids := range impRef {
					for _, e := range eids {
						va.oracle[cle] = append(va.oracle[cle], cmLiaison{eid: e, ti: v.force})
					}
				}
			}
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, _, occupes := cmMarcher(f, va, cmMux{d, diag, j, b3Statut{st}})
			bis2Intercepteur = nil
			if jref == nil {
				jref, ref, oracleRef, impRef = j, st.ref, diag.oracle, diag.imp
			}
			ps, us, perdusSains := rl3Sains(j)
			l := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom, r.Paquets,
				r.PaquetsFermes, ps, r.Utiles.RecordsFermes, us, r.Utiles.Records, r.Bloquants[CauseHorsCadre].Paquets,
				j.gagnes, j.perdus, j.gagnesContredits, perdusSains)
			for _, g := range rp3Gates {
				n := 0
				if x := d.t["dernier_composant_x_classe"][g.cle]; x != nil {
					n = x.horsCadre
				}
				l += fmt.Sprintf("\t%d", n)
			}
			nl := 0
			for _, x := range va.oracle {
				nl += len(x)
			}
			l += fmt.Sprintf("\t%d\t%d\t%d", nl, occupes, diag.desavoues)
			lignes = append(lignes, l)
			for nom, m := range diag.t {
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom, nom, cle,
						x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
				}
			}
		}
		fmt.Fprintf(os.Stderr, "%s %s : fait\n", id, f.build)
	}
	b2Ecrire(t, sortie, "r_comp_p3.tsv", lignes)
	b2Ecrire(t, sortie, "r_comp_p3_tables.tsv", tabs)
}

// rp3Monde : le monde vu par le diagnostic (pose a chaque paquet ; la liaison d un slot n est pas
// exportee, ce fichier est dans le paquet).
var rp3Monde *World

// liaison decrit la liaison du slot d un record a masque impossible : sa SOURCE (forme du
// [slotState] : NEW = eid complet ; image-cle = generation inconnue et vue connue ; table de
// datums ou anticipation = douce, generation et vue inconnues ; inference = douce, eid complet),
// l accord de generation, la declaration de la premiere image-cle du chunk et l etat du bloc.
func (d *rp3Diag) liaison(r FrameRecord) string {
	src := "slot non lie"
	if rp3Monde != nil {
		if s, ok := rp3Monde.slots[r.Slot]; ok {
			switch {
			case !s.Soft && !s.GenAny:
				src = "NEW (eid complet)"
				if s.FullID != r.ID {
					src += ", generation differente"
				}
			case !s.Soft && s.GenAny && s.Vue != vueInconnue:
				src = "image-cle ou joker"
			case s.Soft && s.GenAny:
				src = "table de datums ou anticipation"
			case s.Soft:
				src = "inference de chaine"
			default:
				src = "autre"
			}
		}
	}
	kf := "slot non declare par la 1re image-cle du chunk"
	if ti, ok := d.b.declares[d.chunk][r.Slot]; ok {
		kf = fmt.Sprintf("1re image-cle du chunk : ti=%d", ti)
	}
	e, _ := d.b.entree(d.chunk, r.Slot)
	bloc := fmt.Sprintf("bloc : gen %d (eid gen %d), vivante %v", e.Gen, r.ID>>30, e.Vivante())
	return cmJoindre(src, kf, bloc)
}

// slotDuFilm : le slot du record, et les archetypes que les images-cles DU FILM declarent pour lui.
func (d *rp3Diag) slotDuFilm(r FrameRecord) string {
	var tis []int
	if d.b.table != nil {
		for ti := range d.b.table.archetypesDuSlot[r.Slot] {
			tis = append(tis, int(ti))
		}
	}
	sort.Ints(tis)
	return fmt.Sprintf("ti=%d slot %d gen %d · archetypes declares au film %v", r.TypeIndex, r.Slot, r.ID>>30, tis)
}
