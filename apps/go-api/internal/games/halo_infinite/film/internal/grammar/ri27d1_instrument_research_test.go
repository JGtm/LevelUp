//go:build research

package grammar

// ri27d1_instrument_research_test.go — 2.7.d1 (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// §1.2) : les complements de l instrument 2.7.d0 (`ri27d0_images_cles_research_test.go`) — classes
// d admission, temoin de hasard, marque de portage, familles en plus, dump par record.
//
//	A   film  adm  classe  compte            classes de comparaison par classe d admission du record
//	                                         (A ferme, B non ferme mais n(i22) = 4, C le reste)
//	T   film  cle  compte                    temoin de hasard : la meme marche, en-tete a 109 bits
//	M   film  iNN  nom  decalage  compte     marque de portage, records FERMES seulement
//	MX  film  preuve  iNN  compte            marque de portage, records non fermes (reference)
//	Y   film  adm  famille  iNN  nom  decalage  compte
//	                                         familles que la fenetre trouve EN PLUS de la grammaire
//	                                         dans un record admis (A ou B), situees dans les composants ;
//	                                         variante « mot_un_bit_plus_tot » : la famille lue un bit avant
//	R   film  ts  slot  debut  preuve  desync  bits  adm  n22  temoin  grammaire  fenetre  comps  T1  T2  ids
//	                                         dump par record (films de RI27D1_RECORDS)
//	SF/AF/TF/MF/YF  fNN  ...                 les memes, agreges par version de format

import (
	"fmt"
	"slices"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ri27d1Ext porte les complements de l instrument pour un film.
type ri27d1Ext struct {
	reg      *Registry
	adm      string         // classe d admission du record en cours
	parAdm   map[string]int // adm \t classe
	temoin   map[string]int
	marquesF map[string]int // iNN \t nom \t decalage (fermes)
	marquesX map[string]int // preuve \t iNN (non fermes)
	enPlus   map[string]int // famille \t iNN \t nom \t decalage (fermes)
	dump     bool
	lignes   []string
	n22      int // n(i22) du record en cours, -1 non lu
	tem      string
	mb       int        // lignes MB ecrites
	d        *ri27d1D10 // mesures de D1.0 (ri27d1_d10_research_test.go)
}

func nouvelleExt(reg *Registry, dump bool) *ri27d1Ext {
	return &ri27d1Ext{reg: reg, parAdm: map[string]int{}, temoin: map[string]int{}, marquesF: map[string]int{},
		marquesX: map[string]int{}, enPlus: map[string]int{}, dump: dump, d: &ri27d1D10{}}
}

// admettre pose la classe d admission du record.
func (x *ri27d1Ext) admettre(c *ri27d0Canal, r *lecture.Record, g *ri27d0Gram) {
	i22 := archIndexOf(c.arch, invDeltaGrenadeCountsName)
	x.n22 = -1
	if g.atteint[i22] {
		x.n22 = int(g.compte) //nolint:gosec // R(3)
	}
	switch {
	case r.Preuve == lecture.PreuveFerme:
		x.adm = "A"
	case g.atteint[i22] && g.compte == 4 && len(g.gren) == 4:
		x.adm = "B"
	default:
		x.adm = "C"
	}
	x.parAdm[x.adm+"\trecords"]++
	x.poserLesEtiquettes(c, r, g)
}

// temoinDeHasard rejoue la marche du record avec un en-tete de 109 bits (temoin nomme de
// `keyframeFullStateTemoin`) et dit si elle ferme sur le record suivant, et ce qu elle lit en i22.
func (x *ri27d1Ext) temoinDeHasard(c *ri27d0Canal, p *lecture.Paquet, i int, ctx ContexteDeLecture) {
	r := &p.Records[i]
	if !x.d.vuOrig {
		x.d.origine, x.d.vuOrig = p.TS, true
	}
	x.tem = "sans_frontiere"
	if i+1 >= len(p.Records) {
		x.temoin["sans_frontiere"]++
		return
	}
	ctx.Obs = nil
	tr := walkKeyframeFullState(p.Payload, int(r.Debut), x.reg, ctx, keyframeFullStateTemoin{EnTeteBits: 109})
	ferme := tr.DesyncAt < 0 && tr.EndBit == int(p.Records[i+1].Debut)
	vrai := r.Preuve == lecture.PreuveFerme
	x.temoin["bornes"]++
	x.temoin[fmt.Sprintf("temoin_ferme_%v|vrai_ferme_%v", ferme, vrai)]++
	x.tem = fmt.Sprintf("t%v", ferme)
	if tr.DesyncAt < 0 {
		x.temoin["temoin_au_bout"]++
	}
	i22 := archIndexOf(c.arch, invDeltaGrenadeCountsName)
	n := -1
	for _, cr := range tr.Comps {
		if cr.Index != i22 {
			continue
		}
		br := relecteurDEtatComplet(p.Payload, ctx,
			&Observation{GrenadeCountsHook: func(v uint64, _ []uint64) { n = int(v) }}) //nolint:gosec // R(3)
		br.SetBitPos(cr.StartBit)
		consumeByNameCapturing(br, cr.Name, uint32(keyframeBipedTI), c.arch.Level(i22)) //nolint:gosec // archetype constant
	}
	x.temoin[fmt.Sprintf("temoin_n_i22_%d", n)]++
	x.temoin[fmt.Sprintf("temoin_n_i22_%d|vrai_n_i22_%d", n, x.n22)]++
	x.temoinTDecale(c, p, r, tr, ferme, n, ctx)
}

// fenetresDe rend les fenetres de 32 bits connues de `vues` dont le PREMIER bit tombe dans
// [debut, fin) — l attribution exacte de `familiesByRecordRecs`.
func fenetresDe(pay []byte, debut, fin int, vues map[uint32]bool) [][2]int {
	total := len(pay) * 8
	var out [][2]int
	var w uint32
	for b := debut; b < total && b-31 < fin; b++ {
		w = w<<1 | uint32(source.BitAt(pay, b))
		if b-debut < 31 || !vues[w] {
			continue
		}
		out = append(out, [2]int{b - 31, int(w)})
	}
	return out
}

// situer rend le composant (index, nom) qui contient le bit `b`, et le decalage dans ce composant.
func (c *ri27d0Canal) situerBit(comps []lecture.Composant, b int) (string, string, int) {
	for _, co := range comps {
		if co.Etat == lecture.EtatInfranchissable || co.Etat == lecture.EtatArrete {
			return fmt.Sprintf("au_dela_de_i%d", co.Index), c.arch.component(int(co.Index)), b - int(co.Debut)
		}
		if b >= int(co.Debut) && b < int(co.Debut+co.Bits) {
			return fmt.Sprintf("i%02d", co.Index), c.arch.component(int(co.Index)), b - int(co.Debut)
		}
	}
	if len(comps) > 0 && b < int(comps[0].Debut) {
		return "avant_le_premier_composant", "", b - int(comps[0].Debut)
	}
	if len(comps) > 0 {
		der := comps[len(comps)-1]
		return "apres_la_traversee", "", b - int(der.Debut+der.Bits)
	}
	return "sans_composant", "", 0
}

// situerMarquesEtEnPlus range les marques de portage (fermes : composant, nom, decalage) et les
// familles que la fenetre trouve en plus de la grammaire dans un record ferme.
func (x *ri27d1Ext) situerMarquesEtEnPlus(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram,
	fen []uint32, fin int) {
	comps := p.Comps[r.Comps[0]:r.Comps[1]]
	marques := fenetresDe(p.Payload, int(r.Debut), fin, carrierMarkViews)
	x.teteDuSuivantDuMort(c, p, r, len(marques) > 0)
	x.marqueDePortage(c, p, r, len(marques) > 0)
	for _, f := range marques {
		o, nom, d := c.situerBit(comps, f[0])
		if r.Preuve == lecture.PreuveFerme {
			x.marquesF[fmt.Sprintf("%s\t%s\t%d", o, nom, d)]++
			x.bitsDuComposant(c, p, r, comps, f[0])
		} else {
			x.marquesX[fmt.Sprintf("preuve_%d\t%s", r.Preuve, o)]++
		}
	}
	if x.adm != "A" && x.adm != "B" {
		return
	}
	gn := c.nomsDe(ri27d1FamillesGrammaire(c, g))
	var enPlus []string
	for _, n := range c.nomsDe(fen) {
		if !slices.Contains(gn, n) {
			enPlus = append(enPlus, n)
		}
	}
	if len(enPlus) == 0 {
		return
	}
	for _, n := range enPlus {
		x.parAdm[x.adm+"\tenplus_record:"+n]++
	}
	for _, f := range fenetresDe(p.Payload, int(r.Debut), fin, c.known) {
		n := c.noms[uint32(f[1])] //nolint:gosec // mot de 32 bits
		if !slices.Contains(enPlus, n) {
			continue
		}
		o, nom, d := c.situerBit(comps, f[0])
		x.enPlus[fmt.Sprintf("%s\t%s\t%s\t%s\t%d", x.adm, n, o, nom, d)]++
		// LE MOT LU UN BIT PLUS TOT : si c est une famille que la grammaire lit, la fenetre voit
		// l identifiant de la grammaire decale d un bit (faux positif de la fenetre).
		prec := "aucune"
		if f[0] >= 1 {
			if pn, ok := c.noms[uint32(source.BitsBourres(p.Payload, f[0]-1, 32))]; ok { //nolint:gosec // mot de 32 bits
				prec = pn
			}
		}
		x.enPlus[fmt.Sprintf("%s\t%s\tmot_un_bit_plus_tot=%s\t%s\t%d", x.adm, n, prec, o, d)]++
	}
}

// ri27d1FamillesGrammaire rend les familles connues que la grammaire lit aux emplacements i43 a i46.
func ri27d1FamillesGrammaire(c *ri27d0Canal, g *ri27d0Gram) []uint32 {
	var gf []uint32
	for k := range 4 {
		if !g.armeLue[k] {
			continue
		}
		for _, f := range []uint32{g.idHigh[k], g.idLow[k]} {
			if c.known[f] {
				gf = append(gf, f)
			}
		}
	}
	return gf
}

// armesParEmplacement compare, emplacement par emplacement, la famille que la grammaire lit a
// celles que la fenetre trouve dans l emprise du record.
func (x *ri27d1Ext) armesParEmplacement(c *ri27d0Canal, g *ri27d0Gram, fen []uint32) {
	fn := c.nomsDe(fen)
	for k := range 4 {
		if !g.armeLue[k] {
			continue
		}
		gn := c.nomsDe(ri27d1FamillesGrammaire(c, &ri27d0Gram{armeLue: [4]bool{k == 0, k == 1, k == 2, k == 3},
			idHigh: g.idHigh, idLow: g.idLow}))
		switch {
		case len(gn) == 0 && len(fn) == 0:
			x.parAdm[x.adm+"\tarmes_empl:aucune_des_deux"]++
		case len(gn) == 0:
			x.parAdm[x.adm+"\tarmes_empl:grammaire_vide"]++
		case len(fn) == 0:
			x.parAdm[x.adm+"\tarmes_empl:fenetre_vide"]++
		case inclus(gn, fn):
			x.parAdm[x.adm+"\tarmes_empl:egales"]++
		default:
			x.parAdm[x.adm+"\tarmes_empl:differentes"]++
		}
	}
}

// dumpRecord ecrit la ligne R du record.
func (x *ri27d1Ext) dumpRecord(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram, fen []uint32) {
	if !x.dump {
		return
	}
	var cs []string
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		cs = append(cs, fmt.Sprintf("%d:%d:%d", co.Index, co.Bits, co.Etat))
	}
	x.lignes = append(x.lignes, fmt.Sprintf("R\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%d\t%s\t%s\t%s\t%v\t%s\tT1=%v\tT2=%v\tids=%08x/%08x/%08x/%08x", c.court, p.TS,
		r.Vie.Slot, r.Debut, r.Preuve, r.Desync, r.Bits, x.adm, x.n22, x.tem,
		strings.Join(c.nomsDe(ri27d1FamillesGrammaire(c, g)), "+"), strings.Join(c.nomsDe(fen), "+"), g.gren,
		strings.Join(cs, ","), x.d.t1, x.d.t2, g.idHigh[0], g.idHigh[1], g.idHigh[2], g.idHigh[3]))
}

// ri27d1Lignes rend les lignes du film et ajoute ses comptes aux agregats par format.
func (x *ri27d1Ext) ri27d1Lignes(court string, fm string, classes map[string]int, agr map[string]map[string]int) []string {
	var out []string
	out = append(out, ri27d0Trier(court, "A", x.parAdm)...)
	out = append(out, ri27d0Trier(court, "T", x.temoin)...)
	out = append(out, ri27d0Trier(court, "M", x.marquesF)...)
	out = append(out, ri27d0Trier(court, "MX", x.marquesX)...)
	out = append(out, ri27d0Trier(court, "Y", x.enPlus)...)
	out = append(out, x.lignes...)
	for genre, m := range map[string]map[string]int{"SF": classes, "AF": x.parAdm, "TF": x.temoin, "MF": x.marquesF,
		"YF": x.enPlus} {
		for k, v := range m {
			if agr[genre] == nil {
				agr[genre] = map[string]int{}
			}
			agr[genre][fm+"\t"+k] += v
			agr[genre]["tous\t"+k] += v
		}
	}
	return out
}

// bitsDuComposant ecrit (au plus 12 par film) la ligne MB : les bits du composant qui porte une
// marque de portage d un record ferme, pour situer le champ a la main contre son lecteur.
func (x *ri27d1Ext) bitsDuComposant(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, comps []lecture.Composant, at int) {
	if x.mb >= 12 {
		return
	}
	for ci, co := range comps {
		if at < int(co.Debut) || at >= int(co.Debut+co.Bits) {
			continue
		}
		x.mb++
		suivant := "aucun"
		if ci+1 < len(comps) {
			nx := comps[ci+1]
			var sn strings.Builder
			for b := int(nx.Debut); b < int(nx.Debut+nx.Bits) && b < int(nx.Debut)+24; b++ {
				sn.WriteByte(byte('0' + source.BitAt(p.Payload, b))) //nolint:gosec // un bit
			}
			suivant = fmt.Sprintf("i%02d %s %d %s", nx.Index, c.arch.component(int(nx.Index)), nx.Bits, sn.String())
		}
		var sb strings.Builder
		for b := int(co.Debut); b < int(co.Debut+co.Bits); b++ {
			sb.WriteByte(byte('0' + source.BitAt(p.Payload, b))) //nolint:gosec // un bit
		}
		x.lignes = append(x.lignes, fmt.Sprintf("MB\t%s\t%d\t%d\ti%02d\t%s\t%d\t%d\t%s", c.court, p.TS, r.Vie.Slot, co.Index,
			c.arch.component(int(co.Index)), at-int(co.Debut), co.Bits, sb.String()+"\t"+suivant))
	}
}

// teteDuSuivantDuMort compte, sur les records FERMES, la tete (6 bits) du composant qui suit
// l etat de mort (i11), selon que le record porte la marque de portage ou non.
func (x *ri27d1Ext) teteDuSuivantDuMort(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, marque bool) {
	if r.Preuve != lecture.PreuveFerme {
		return
	}
	comps := p.Comps[r.Comps[0]:r.Comps[1]]
	for ci, co := range comps {
		if c.arch.component(int(co.Index)) != deadStateComponentName || ci+1 >= len(comps) {
			continue
		}
		bitsDe := func(a, n int) string {
			var sb strings.Builder
			for b := a; b < a+n; b++ {
				sb.WriteByte(byte('0' + source.BitAt(p.Payload, b))) //nolint:gosec // un bit
			}
			return sb.String()
		}
		mort := bitsDe(int(co.Debut), int(co.Bits))
		if mort != "000000000011000000000000000010000000000000" {
			mort = fmt.Sprintf("autre_%d_bits", co.Bits)
		}
		suite := ""
		for k := ci + 1; k < len(comps) && k <= ci+3; k++ {
			n := min(int(comps[k].Bits), 8)
			suite += fmt.Sprintf("_i%02d(%s,%d)=%s", comps[k].Index, c.arch.component(int(comps[k].Index)), comps[k].Bits, bitsDe(int(comps[k].Debut), n))
		}
		x.temoin[fmt.Sprintf("mort_suivant_marque_%v_i11=%s%s", marque, mort, suite)]++
	}
}
