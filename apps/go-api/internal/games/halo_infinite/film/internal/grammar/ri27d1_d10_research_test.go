//go:build research

package grammar

// ri27d1_d10_research_test.go — LES MESURES DE D1.0 (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// items D1.0.4 a D1.0.7), complements de l instrument 2.7.d0/2.7.d1 (`ri27d0_images_cles_research_test.go`,
// `ri27d1_instrument_research_test.go`). Aucun fichier de production n est touche.
//
//   - Temoin T apres i43 (D1.0.6, decision U-1) : T1 = au moins un emplacement d arme non vide ET
//     chaque famille d emplacement non vide connue du catalogue, sur un record traverse au-dela du
//     dernier emplacement ; T2 = masque d i47 egal a la bitmap des compteurs d i22. Chaque classe de
//     comparaison est comptee aussi sous les etiquettes `<adm>T1`, `<adm>T2`, `<adm>T12` (A et B).
//     Le meme temoin est joue sur la marche decalee d un bit (en-tete a 109 bits), seulement sur
//     les records ou elle n a pas rejoint la vraie marche avant i43 (lignes T `tw_*`).
//   - Jeu d armes i42 (D1.0.5) : les deux `FUN_1406d00ec` relus a l etendue de l occurrence
//     (param[1] emplacement desire en main principale, param[2] en seconde main), compares au
//     `DrawnSlot` de la fenetre (dernier R(2) present) : classes `degaine:*`.
//   - Marque de portage (D1.0.4, decision U-2) : (a) drapeaux 0x4 et 0x8 du R(5) de tete d i13 ;
//     (b) configuration de la fenetre lue par la grammaire (i11 sous sa forme par defaut de 42 bits,
//     bit d i12 a 1, R(5) d i13 = 01111) ; lignes MK (comptes) et MP (porteurs de (a) que la
//     fenetre ne voit pas : film, horodatage, slot, classe, forme d i11, R(5) d i13).
//   - Reserves (D1.0.7) : lignes RS, au plus deux records par film dont la reserve0 differe — debut
//     d i30 lu par la marche contre debut du bloc de munitions retenu par la fenetre, candidats.

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27d1D10 porte l etat des mesures D1.0 d un film.
type ri27d1D10 struct {
	t1, t2  bool
	labels  []string // classes d admission du record en cours (adm, puis adm+T*)
	origine uint64   // horodatage de la premiere image-cle du film
	vuOrig  bool
	rs      int // lignes RS ecrites
	// roles, dernier : les roles du canal de production et son dernier emplacement (confrontation D1.1).
	roles   map[int]roleDOccurrence
	dernier int
}

// ri27d1MortParDefaut est la forme par defaut de 42 bits de l etat de mort (i11), celle que la
// marque de portage de la fenetre recouvre (mesure 1, plan §2 G-6).
const ri27d1MortParDefaut = "000000000011000000000000000010000000000000"

// ri27d1VitalitesMarque est le R(5) de tete d i13 des records marques (01111 : drapeaux 0x1, 0x2,
// 0x4 et 0x8 de `FUN_1407eef08`).
const ri27d1VitalitesMarque = 0b01111

// temoinT calcule T1 et T2 sur une lecture de la grammaire. `desync` est l index ou la marche
// s est arretee (-1 au bout) : T1 exige qu elle ait depasse le dernier emplacement d arme.
func (c *ri27d0Canal) temoinT(g *ri27d0Gram, desync int) (t1, t2 bool) {
	maxEmpl := -1
	for id := range weaponEmplacements(c.arch) {
		maxEmpl = max(maxEmpl, id)
	}
	if maxEmpl >= 0 && (desync < 0 || desync > maxEmpl) {
		nonVides, connues := 0, true
		for k := range 4 {
			if !g.armeLue[k] || g.idHigh[k] == noVariant {
				continue
			}
			nonVides++
			connues = connues && c.known[g.idHigh[k]]
		}
		t1 = nonVides > 0 && connues
	}
	if g.gsLu && g.compte == 4 && len(g.gren) == 4 {
		var bm uint32
		for r, v := range g.gren {
			if v > 0 {
				bm |= 1 << uint(r)
			}
		}
		t2 = g.gsMask == bm
	}
	return t1, t2
}

// poserLesEtiquettes pose les classes d admission du record en cours (apres [ri27d1Ext.admettre]).
func (x *ri27d1Ext) poserLesEtiquettes(c *ri27d0Canal, r *lecture.Record, g *ri27d0Gram) {
	d := x.d
	d.t1, d.t2 = c.temoinT(g, int(r.Desync))
	d.labels = append(d.labels[:0], x.adm)
	if x.adm == "A" || x.adm == "B" {
		if d.t1 {
			d.labels = append(d.labels, x.adm+"T1")
		}
		if d.t2 {
			d.labels = append(d.labels, x.adm+"T2")
		}
		if d.t1 && d.t2 {
			d.labels = append(d.labels, x.adm+"T12")
		}
	}
	for _, l := range d.labels[1:] {
		x.parAdm[l+"\trecords"]++
	}
	if !d.t1 {
		x.parAdm[x.adm+"\tt1_refuse:"+c.raisonT1(g, int(r.Desync))]++
	}
}

// raisonT1 dit pourquoi T1 refuse un record : marche arretee avant le dernier emplacement, aucun
// emplacement non vide, ou famille hors catalogue (avec la famille lue).
func (c *ri27d0Canal) raisonT1(g *ri27d0Gram, desync int) string {
	maxEmpl := -1
	for id := range weaponEmplacements(c.arch) {
		maxEmpl = max(maxEmpl, id)
	}
	if maxEmpl < 0 || (desync >= 0 && desync <= maxEmpl) {
		return "marche_arretee_avant_i46"
	}
	for k := range 4 {
		if g.armeLue[k] && g.idHigh[k] != noVariant && !c.known[g.idHigh[k]] {
			return fmt.Sprintf("famille_hors_catalogue_emplacement_%d", k)
		}
	}
	return "aucun_emplacement_non_vide"
}

// compterAdm compte une classe de comparaison sous chaque etiquette d admission du record.
func (x *ri27d1Ext) compterAdm(classe string) {
	for _, l := range x.d.labels {
		x.parAdm[l+"\t"+classe]++
	}
}

// relireTrace relit, a l etendue de leurs occurrences, les composants de role d une marche
// rejouee (le temoin decale) — la meme relecture que [ri27d0Canal.lire], sur une trace.
func (c *ri27d0Canal) relireTrace(p *lecture.Paquet, tr EntityTrace, ctx ContexteDeLecture) ri27d0Gram {
	var comps []lecture.Composant
	for k, cr := range tr.Comps {
		if !cr.Ported {
			break
		}
		fin := tr.EndBit
		if k+1 < len(tr.Comps) {
			fin = tr.Comps[k+1].StartBit
		}
		comps = append(comps, lecture.Composant{Index: uint8(cr.Index), Etat: lecture.EtatInterprete, //nolint:gosec // index < 64
			Debut: uint32(cr.StartBit), Bits: uint32(fin - cr.StartBit)}) //nolint:gosec // bits du payload
	}
	q := &lecture.Paquet{Payload: p.Payload, Comps: comps}
	return c.lire(q, &lecture.Record{Comps: [2]uint32{0, uint32(len(comps))}}, ctx) //nolint:gosec // borne
}

// temoinTDecale joue T sur la marche decalee d un bit, quand elle n a pas rejoint la vraie marche
// avant i43 (une marche qui la rejoint relit les memes bits : elle n est plus un temoin).
func (x *ri27d1Ext) temoinTDecale(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, tr EntityTrace,
	ferme bool, n22 int, ctx ContexteDeLecture) {
	i43 := -1
	for id, k := range weaponEmplacements(c.arch) {
		if k == 0 {
			i43 = id
		}
	}
	vrais := map[int]uint32{}
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		vrais[int(co.Index)] = co.Debut
	}
	for _, cr := range tr.Comps {
		if cr.Index > i43 {
			break
		}
		if d, ok := vrais[cr.Index]; ok && int(d) == cr.StartBit {
			x.temoin["tw_rejoint"]++
			return
		}
	}
	x.temoin["tw_independant"]++
	g := c.relireTrace(p, tr, ctx)
	t1, t2 := c.temoinT(&g, tr.DesyncAt)
	base := ferme || n22 == 4
	for _, v := range []struct {
		nom string
		ok  bool
	}{{"base", base}, {"T1", base && t1}, {"T2", base && t2}, {"T12", base && t1 && t2},
		{"T1_seul", t1}, {"T2_seul", t2}} {
		if v.ok {
			x.temoin["tw_admis_"+v.nom]++
		}
	}
}

// degaine compare le jeu d armes i42 relu (param[1], param[2]) au `DrawnSlot` de la fenetre.
func (x *ri27d1Ext) degaine(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, fenetre int) {
	if p1, p2, ok := ri27d1JeuDArmes(c, p, r); ok {
		switch {
		case p1 == -1:
			x.compterAdm(fmt.Sprintf("degaine:p1_moins_un_fenetre_%v", fenetre >= 0))
		case p2 != -1:
			x.compterAdm(fmt.Sprintf("degaine:ambidextrie_fenetre_egale_p2_%v", fenetre == p2))
		case fenetre < 0:
			x.compterAdm("degaine:fenetre_vide")
		case p1 == fenetre:
			x.compterAdm("degaine:egal")
		default:
			x.compterAdm("degaine:different")
		}
		return
	}
	x.compterAdm("degaine:non_atteint")
}

// ri27d1JeuDArmes relit param[1] et param[2] de l occurrence i42 du record, a la main (lecture de
// test, hors du lecteur de production) ; faux quand i42 n est pas traverse.
func ri27d1JeuDArmes(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record) (p1, p2 int, ok bool) {
	i42 := archIndexOf(c.arch, "biped-desired-weapon-set")
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if int(co.Index) != i42 || co.Etat != lecture.EtatInterprete {
			continue
		}
		br := LecteurSur(p.Payload)
		br.SetBitPos(int(co.Debut))
		br.ReadBits(3)      // FUN_1406d0f20, param[0]
		lit := func() int { // FUN_1406d00ec : R(1) ; si 0, R(2) ; sinon -1
			if br.ReadBit() {
				return -1
			}
			return int(br.ReadBits(2)) //nolint:gosec // deux bits
		}
		p1 := lit()
		return p1, lit(), true
	}
	return -1, -1, false
}

// ri27d1Bits rend les bits [a, a+n) du payload, en texte.
func ri27d1Bits(pay []byte, a, n int) string {
	br := LecteurSur(pay)
	br.SetBitPos(a)
	out := make([]byte, 0, n)
	for range n {
		out = append(out, '0'+byte(br.ReadBits(1))) //nolint:gosec // un bit
	}
	return string(out)
}

// marqueDePortage compte les deux predicats de la marque (U-2) contre la fenetre et ecrit les
// porteurs de (a) que la fenetre ne voit pas.
func (x *ri27d1Ext) marqueDePortage(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, fenetre bool) {
	mort, i12, vit := ri27d1Configuration(c, p, r)
	predA := vit >= 0 && vit&0xC == 0xC
	predB := mort == ri27d1MortParDefaut && i12 == "1" && vit == ri27d1VitalitesMarque
	ferme := r.Preuve == lecture.PreuveFerme
	pop := []string{"tous", x.adm}
	if ferme {
		pop = append(pop, "fermes")
	}
	pop = append(pop, x.d.labels[1:]...)
	for _, k := range pop {
		x.temoin[fmt.Sprintf("mk_%s|a_%v|b_%v|fenetre_%v", k, predA, predB, fenetre)]++
	}
	if fenetre || (predA && mort != ri27d1MortParDefaut) {
		forme := "defaut"
		if mort != ri27d1MortParDefaut {
			forme = fmt.Sprintf("autre_%d_bits", len(mort))
		}
		x.temoin[fmt.Sprintf("mk_detail|fenetre_%v|i11_%s|i12_%s|i13_%05b", fenetre, forme, i12, max(vit, 0))]++
	}
	if predA && !fenetre {
		forme := "defaut"
		if mort != ri27d1MortParDefaut {
			forme = mort
		}
		x.lignes = append(x.lignes, fmt.Sprintf("MP\t%s\t%d\t%d\t%d\t%s\tferme_%v\t%s\t%s\t%05b", c.court, p.TS,
			p.TS-x.d.origine, r.Vie.Slot, x.adm, ferme, forme, i12, vit))
	}
}

// reservesEchantillon ecrit, pour au plus deux records par film dont la reserve0 differe, le debut
// d i30 de la marche, le debut du bloc de munitions de la fenetre et ses candidats.
func (x *ri27d1Ext) reservesEchantillon(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram,
	fenRes int) {
	if x.d.rs >= 2 || (x.adm != "A" && x.adm != "B") || !g.resLu[0] || g.res[0] < 0 || fenRes < 0 ||
		g.res[0] == fenRes {
		return
	}
	fin := len(p.Payload) * 8
	for i := range p.Records {
		if &p.Records[i] == r && i+1 < len(p.Records) {
			fin = int(p.Records[i+1].Debut)
		}
	}
	am0 := -1
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if c.roles[int(co.Index)] == "am0" {
			am0 = int(co.Debut)
		}
	}
	first, ok := invFirstFamily(p.Payload, int(r.Debut), fin, c.known)
	if !ok {
		return
	}
	lo := max(first-1-invAmmoSearchSpan, int(r.Debut))
	sols := invSolveAmmoBlock(p.Payload, first-1, lo)
	x.d.rs++
	x.lignes = append(x.lignes, fmt.Sprintf("RS\t%s\t%d\t%d\t%s\tgrammaire=%d\tfenetre=%d\ti30_marche=%d\tblocs_fenetre=%v\tecart=%d\tcandidat_marche=%v",
		c.court, p.TS, r.Vie.Slot, x.adm, g.res[0], fenRes, am0, sols, ri27d1Ecart(sols, am0), slices.Contains(sols, am0)))
}

// ri27d1Reserve0 rend la reserve du premier emplacement lue par la fenetre, -1 sans lecture.
func ri27d1Reserve0(inv types.KeyframeInventory) int {
	if !inv.AmmoRead || inv.Ammo[0].Res == nil {
		return -1
	}
	return int(*inv.Ammo[0].Res)
}

// ri27d1Ecart rend l ecart entre le debut retenu par la fenetre (le plus petit) et celui de la marche.
func ri27d1Ecart(sols []int, am0 int) int {
	if len(sols) == 0 || am0 < 0 {
		return 0
	}
	return sols[0] - am0
}

// confronterAuCanal rejoue, sur le record, la lecture et la regle d admission du canal de
// production (`keyframe_etat_complet_*.go`, D1.1) et compte, sous `canal_*`, son accord avec la
// relecture de l instrument : meme verdict d admission (A ou B, T1 et T2), et, sur un record admis,
// memes armes, compteurs de grenades, chargeurs, reserves, emplacement desire, grenade selectionnee,
// rang de capacite et marque de portage.
func (x *ri27d1Ext) confronterAuCanal(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram,
	ctx ContexteDeLecture) {
	if x.d.roles == nil {
		x.d.roles = rolesDuBipede(c.arch)
		x.d.dernier = -1
		for id := range weaponEmplacements(c.arch) {
			x.d.dernier = max(x.d.dernier, id)
		}
	}
	l := lireLEtatComplet(p, r, c.arch, x.d.roles, ctx)
	x.temoin[fmt.Sprintf("canal_debordements_%d", l.debordements)]++
	admis := admettre(r, &l, x.d.dernier) == refusAucun
	attendu := (x.adm == "A" || x.adm == "B") && x.d.t1 && x.d.t2
	if admis != attendu {
		x.temoin[fmt.Sprintf("canal_admission_differente|canal_%v|instrument_%v", admis, attendu)]++
		return
	}
	if !admis {
		x.temoin["canal_non_admis"]++
		return
	}
	x.temoin["canal_admis"]++
	for _, d := range x.ecartsAuCanal(c, p, r, g, &l) {
		x.temoin["canal_valeur_differente:"+d]++
	}
}

// ecartsAuCanal rend les champs publies par le canal qui different de la relecture de l instrument.
func (x *ri27d1Ext) ecartsAuCanal(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram,
	l *lectureDEtatComplet) []string {
	var out []string
	ecart := func(nom string, egal bool) {
		if !egal {
			out = append(out, nom)
		}
	}
	var fams []uint32
	for k := range 4 {
		if g.armeLue[k] && g.idHigh[k] != noVariant {
			fams = append(fams, g.idHigh[k])
		}
	}
	ecart("armes", slices.Equal(fams, l.armesDe(r.Vie.Slot).Families))
	inv, _ := l.inventaireDe(r.Vie.Slot)
	for k := range 4 {
		ecart(fmt.Sprintf("grenades%d", k), int(inv.Grenades[k]) == int(g.gren[k])) //nolint:gosec // R(8)
		mag, res := -1, -1
		if inv.Ammo[k].Mag != nil {
			mag = int(*inv.Ammo[k].Mag)
		}
		if inv.Ammo[k].Res != nil {
			res = int(*inv.Ammo[k].Res)
		}
		ecart(fmt.Sprintf("chargeur%d", k), mag == g.mag[k])
		ecart(fmt.Sprintf("reserve%d", k), res == g.res[k])
	}
	sel := -1
	if g.gsSel != GrenadeSetNoSelection {
		sel = g.gsSel - 1
	}
	ecart("grenade_selectionnee", inv.SelectedGrenadeRank == sel)
	rang := -1
	if g.rang >= rangDeCapaciteMin && g.rang <= rangDeCapaciteMax {
		rang = g.rang
	}
	ecart("capacite", inv.AbilityRank == rang)
	p1, _, _ := ri27d1JeuDArmes(c, p, r)
	ecart("degaine", inv.DrawnSlot == p1)
	mort, i12, vit := ri27d1Configuration(c, p, r)
	ecart("marque", l.porteLaMarque() == (mort == ri27d1MortParDefaut && i12 == "1" && vit == ri27d1VitalitesMarque))
	return out
}

// ri27d1Configuration relit, dans leurs bits, l etat de mort i11, le bit d i12 et le R(5) de tete
// d i13 du record (-1 sans i13).
func ri27d1Configuration(c *ri27d0Canal, p *lecture.Paquet, r *lecture.Record) (mort, i12 string, vit int) {
	i11 := archIndexOf(c.arch, deadStateComponentName)
	i13 := archIndexOf(c.arch, compObjectMaximumVitalities)
	vit = -1
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if co.Etat != lecture.EtatInterprete && co.Etat != lecture.EtatDelimite { // traverse, interprete ou non
			continue
		}
		switch int(co.Index) {
		case i11:
			mort = ri27d1Bits(p.Payload, int(co.Debut), int(co.Bits))
		case i11 + 1:
			i12 = ri27d1Bits(p.Payload, int(co.Debut), int(co.Bits))
		case i13:
			if co.Bits >= 5 {
				br := LecteurSur(p.Payload)
				br.SetBitPos(int(co.Debut))
				vit = int(br.ReadBits(5)) //nolint:gosec // cinq bits
			}
		}
	}
	return mort, i12, vit
}

// TestRI27d1Replis — LE GATE DE D1.2 : sur chaque film, en contexte de cuisson, les trois replis des
// fenetres derriere la lecture comptent exactement les records bipedes non admis, et le contexte
// du film porte les memes comptes. Sortie `replis.tsv` (RI27C_OUT) : film, bipedes, admis, non
// admis, replis armes / inventaire / marque (contexte), recuperes armes / inventaire / marque.
func TestRI27d1Replis(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	known := weaponv3.FamillesConnues()
	lignes := []string{"film\tbipedes\tadmis\tnon_admis\trepli_armes\trepli_inventaire\trepli_marque\trec_armes\trec_inventaire\trec_marque"}
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		e, err := ScanEtatsDesImagesCles(fc, known, 0)
		if err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		a, r := e.Admission, fc.ComptesDesReplis()
		var ra, ri, rm int
		for _, k := range e.Recuperes {
			ra, ri, rm = ra+unSi(k.Armes), ri+unSi(k.Inventaire), rm+unSi(k.Marque)
		}
		nonAdmis := a.Bipedes - a.Admis
		if r.FenetresArmesImageCle != nonAdmis || r.FenetresInventaireImageCle != nonAdmis || r.FenetresMarqueDePortage != nonAdmis {
			t.Errorf("%s : replis %+v, attendu %d chacun", court, r, nonAdmis)
		}
		lignes = append(lignes, fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", court, a.Bipedes, a.Admis, nonAdmis,
			r.FenetresArmesImageCle, r.FenetresInventaireImageCle, r.FenetresMarqueDePortage, ra, ri, rm))
	}
	ri27cEcrire(t, filepath.Join(sortie, "replis.tsv"), lignes)
}
