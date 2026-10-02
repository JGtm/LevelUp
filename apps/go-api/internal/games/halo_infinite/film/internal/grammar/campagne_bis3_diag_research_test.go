//go:build research

package grammar

// campagne_bis3_diag_research_test.go — MESURES BIS 3 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01) :
// LES POPULATIONS MESUREES SANS LOT NI RECHERCHE (points 27 et 28 de CRITIQUE_COMPLETUDE_1, D-24).
//
// Le diagnostic recoit la marche de reference (meme marche que la carte v2, [cmMarcher]) et, par
// chunk, reprend les PREMIERS rejets hors datum de chaque eid comme la sonde M1
// (`campagne_naissances_research_test.go`) : meme classe de naissance ([cmBlocs.naissance]), meme
// plausibilite ([cmBlocs.plausibilite]), meme recherche des en-tetes NEW ([cmCollecteur.chercher])
// et meme temoin. Il ventile ensuite chaque population par les grandeurs qui departagent ses causes
// possibles, et pose les liaisons des oracles par population (la borne de gain de chacune).
//
// Aucun fichier de production n est touche ; aucune sortie des sondes anterieures ne change.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// Les populations de la critique (point 27) ; P7, la region (iii), a sa propre sonde.
const (
	b3P1 = "P1 image-cle incomplete"
	b3P2 = "P2 realloue sous une autre generation"
	b3P3 = "P3 aucune allocation"
	b3P4 = "P4 slot au-dela du plafond"
	b3P5 = "P5 introuvable"
	b3P6 = "P6 trouve a plus de 3 paquets"
)

// b3Diag est l ecouteur de diagnostic.
type b3Diag struct {
	f  *cmFilm
	b  *cmBlocs
	x  *cmCollecteur // porte chercher / meilleur / region
	t  cmTables
	m1 bool // false : classes et plausibilite seulement (balayage des largeurs)

	chunk, chunkPrec int
	paquets, prec    []*cmPaquet
	liesAuDebut      map[uint32]bool
	finPrec, finCour map[uint32]bool
	dernierDelta     int
	kfPays           [][]byte
	kfEcartes        map[uint32]bool
	pred             map[uint32]int // predictions de l allocateur du chunk

	// pop : population -> (chunk, eid) ; imageCle : oracle P1 (liaison au debut du chunk, archetype
	// du pont du bloc DU CHUNK) ; etendu : filtre -> oracle des naissances trouvees a plus de 3 paquets.
	pop      map[string]map[[2]uint32]bool
	imageCle map[[2]int][]cmLiaison
	etendu   map[string]map[[2]int][]cmLiaison
}

func b3NouveauDiag(f *cmFilm, b *cmBlocs, m1 bool) *b3Diag {
	return &b3Diag{f: f, b: b, m1: m1, t: cmTables{}, chunk: -1, chunkPrec: -1,
		x:        &cmCollecteur{f: f, b: b, t: cmTables{}},
		pop:      map[string]map[[2]uint32]bool{},
		imageCle: map[[2]int][]cmLiaison{}, etendu: map[string]map[[2]int][]cmLiaison{}}
}

func (d *b3Diag) debutDeChunk(c int, data []byte, pks []FilmPacket, w *World) {
	d.finir()
	d.prec, d.paquets = d.paquets, nil
	d.chunkPrec, d.chunk = d.chunk, c
	d.finPrec, d.finCour = d.finCour, nil
	d.liesAuDebut = make(map[uint32]bool, len(w.slots))
	for s := range w.slots {
		d.liesAuDebut[s] = true
	}
	d.dernierDelta, d.kfPays, d.kfEcartes = -1, nil, map[uint32]bool{}
	marche := d.f.fc.MarcheDImageCle()
	for _, pk := range pks {
		switch {
		case pk.Type == PacketTypeDelta && pk.Size >= 1:
			d.dernierDelta = pk.Index
		case pk.Type == PacketTypeKeyframe:
			pay := pk.Payload(data)
			d.kfPays = append(d.kfPays, pay)
			for _, r := range marche.Marcher(pay).Ecartes {
				d.kfEcartes[uint32(r.Slot)] = true //nolint:gosec // slot borne par le walker
			}
		}
	}
}

func (d *b3Diag) paquet(_ int, p *cmPaquet, w *World) {
	d.paquets = append(d.paquets, p)
	if p.pk.Index == d.dernierDelta {
		d.finCour = make(map[uint32]bool, len(w.slots))
		for s := range w.slots {
			d.finCour[s] = true
		}
	}
}

func (d *b3Diag) finDeFilm() { d.finir() }

// b3Rejet : le premier rejet d un eid dans le chunk.
type b3Rejet struct {
	r             *cmRejet
	classe, plaus string
}

// rejets reprend les premiers rejets du chunk (regle de [cmCollecteur.naissances]).
func (d *b3Diag) rejets() ([]uint32, map[uint32]*b3Rejet) {
	neufs := map[uint32]bool{}
	out := map[uint32]*b3Rejet{}
	var ordre []uint32
	for j, p := range d.paquets {
		for _, s := range p.d.NeufsLus {
			neufs[s] = true
		}
		if p.d.Sortie != SortieVueBRejetHorsDatum {
			continue
		}
		e := p.d.EIDRejete
		x := out[e]
		if x == nil {
			cl := d.b.naissance(d.chunk, e, neufs[e&0x3fffffff])
			x = &b3Rejet{r: &cmRejet{eid: e, premier: j, classe: cl, ordre: cmOrdreRejet(p)}, classe: cl,
				plaus: d.b.plausibilite(cl, e)}
			out[e] = x
			ordre = append(ordre, e)
		}
		k := compteDuPaquet(p)
		x.r.compte.paquets += k.paquets
		x.r.compte.horsCadre += k.horsCadre
		x.r.compte.fermes += k.fermes
		x.r.compte.enJeu += k.enJeu
	}
	return ordre, out
}

// finir analyse le chunk tamponne.
func (d *b3Diag) finir() {
	if len(d.paquets) == 0 {
		return
	}
	d.controleImageCle()
	ordre, rej := d.rejets()
	if len(ordre) == 0 {
		return
	}
	if d.m1 {
		pris := map[uint32]bool{}
		cibles := map[uint32]uint32{}
		rs := map[uint32]*cmRejet{}
		for _, e := range ordre {
			r := rej[e].r
			rs[e], cibles[e] = r, e
			if tm, ok := d.b.cmTemoin(d.chunk, e, pris); ok {
				r.temoin, r.aTemoin = tm, true
				cibles[tm] = e
			}
		}
		d.x.chunk, d.x.paquets = d.chunk, d.paquets
		d.x.chercher(rs, cibles)
	}
	d.pred = d.b.predictions(d.chunk)
	for _, e := range ordre {
		d.ranger(rej[e])
	}
}

// etat nomme l etat d une entree de bloc.
func b3Etat(e DatumEntry, ok bool) string {
	switch {
	case !ok:
		return "absent"
	case e.Vivante():
		return "vivant"
	case e.Gen != 0 || e.Drapeaux != 0:
		return "trace"
	}
	return "vide"
}

// ranger classe un eid rejete dans ses populations et ventile chacune.
func (d *b3Diag) ranger(x *b3Rejet) {
	r := x.r
	k := r.compte
	k.n = 1
	slot, g := r.eid&0x3fffffff, uint8(r.eid>>30) //nolint:gosec // deux bits
	cle := [2]uint32{uint32(d.chunk), r.eid}      //nolint:gosec // numero de chunk
	_, decl := d.b.declares[d.chunk][slot]
	var pops []string
	if x.classe == "vivant au bloc du chunk" && !d.liesAuDebut[slot] && !decl {
		pops = append(pops, b3P1)
	}
	switch x.classe {
	case "realloue sous une autre generation":
		pops = append(pops, b3P2)
	case "aucune allocation":
		pops = append(pops, b3P3)
	}
	if x.plaus == "slot au-dela du plafond du film" {
		pops = append(pops, b3P4)
	}
	var best *cmTrouve
	reg, ok, dist := "sans recherche", false, -1
	if d.m1 {
		best, reg, ok = d.x.meilleur(r.trouves, r.eid)
		if best == nil {
			pops = append(pops, b3P5)
		} else {
			dist = r.premier - best.j
			if ok && dist > 3 {
				pops = append(pops, b3P6)
			}
		}
	}
	d.t.add("classe_x_plausibilite", cmJoindre(x.classe, x.plaus), k)
	if len(pops) == 0 {
		d.t.add("population", "aucune des six", k)
	}
	for _, p := range pops {
		if d.pop[p] == nil {
			d.pop[p] = map[[2]uint32]bool{}
		}
		d.pop[p][cle] = true
		d.t.add("population", p, k)
		d.t.add("population_x_classe", cmJoindre(p, x.classe, x.plaus), k)
		if d.m1 {
			d.t.add("population_x_m1", cmJoindre(p, b3M1(best, reg, ok, dist)), k)
		}
	}
	d.t.add("dernier_record_x_classe", cmJoindre(x.classe, d.dernierRecord(d.paquets[r.premier])), k)
	d.t.add("dernier_composant_x_classe", cmJoindre(x.classe, d.dernierComposant(d.paquets[r.premier])), k)
	cur, curOK := d.b.entree(d.chunk, slot)
	n, aSuiv := d.b.suivant[d.chunk]
	nx, nxOK := d.b.entree(n, slot)
	if !aSuiv {
		nxOK = false
	}
	for _, p := range pops {
		switch p {
		case b3P1:
			d.p1(r, slot, g, cur, nx, nxOK, k)
		case b3P2:
			d.t.add("p2_detail", cmJoindre(b3RelGen("g", g, cur.Gen), b3RelGen("bloc suivant", nx.Gen, g),
				"bloc "+b3Etat(cur, curOK), "suivant "+b3Etat(nx, nxOK)), k)
		case b3P3:
			tete := "tete autre"
			if g == (cur.Gen+1)&3 {
				tete = "tete predite (gen+1)"
			}
			d.t.add("p3_detail", cmJoindre(fmt.Sprintf("g=%d", g), tete, "bloc "+b3Etat(cur, curOK),
				fmt.Sprintf("slot > plafond : %v", slot > d.b.plafond), r.ordre, classeDeCascade(r.compte.paquets)), k)
			d.t.add("p3_ferme", cmJoindre(fmt.Sprintf("paquet du rejet ferme : %v", d.paquets[r.premier].d.Fermee),
				fmt.Sprintf("paquets du rejet %s", classeDeCascade(r.compte.paquets))), k)
		case b3P4:
			d.t.add("p4_detail", cmJoindre(fmt.Sprintf("plafond %s", b3Tranche(int(d.b.plafond))),
				fmt.Sprintf("slot %s", b3Tranche(int(slot))), x.classe, classeDeCascade(r.compte.paquets)), k)
		case b3P5:
			d.p5(r, slot, k)
		}
	}
	if d.m1 {
		d.p6(r, k)
	}
}

// b3M1 nomme ce que M1 a trouve.
func b3M1(best *cmTrouve, reg string, ok bool, dist int) string {
	switch {
	case best == nil:
		return "M1 introuvable"
	case !ok:
		return "M1 trouve hors region non lue"
	case dist <= 3:
		return "M1 region non lue a <= 3 paquets : " + reg
	}
	return "M1 region non lue a > 3 paquets : " + reg
}

// b3RelGen : la relation entre deux generations.
func b3RelGen(nom string, a, b uint8) string {
	switch a {
	case b:
		return nom + " = meme gen"
	case (b + 1) & 3:
		return nom + " = gen+1"
	case (b + 2) & 3:
		return nom + " = gen+2"
	}
	return nom + " = gen+3"
}

// b3Tranche range une valeur de slot.
func b3Tranche(s int) string {
	switch {
	case s < 1024:
		return "< 1024"
	case s < 2048:
		return "1024-2047"
	case s < 4096:
		return "2048-4095"
	}
	return ">= 4096"
}

// dernierRecord : le dernier record lu avant l en-tete rejete.
func (d *b3Diag) dernierRecord(p *cmPaquet) string {
	if len(p.recs) == 0 {
		return "aucun record lu avant le rejet"
	}
	r := p.recs[len(p.recs)-1]
	return fmt.Sprintf("%s ti=%d", b3Genre(r.Type), r.TypeIndex)
}

// p1 ventile l image-cle incomplete.
func (d *b3Diag) p1(r *cmRejet, slot uint32, g uint8, cur, nx DatumEntry, nxOK bool, k cmCompte) {
	kf := "en-tete exact absent de l image-cle"
	autre := false
	for _, pay := range d.kfPays {
		switch b3EnTeteDImageCle(pay, r.eid) {
		case 2:
			kf = "en-tete exact present"
		case 1:
			autre = true
		}
	}
	if kf != "en-tete exact present" && autre {
		kf = "meme slot sous une autre generation present"
	}
	if len(d.kfPays) == 0 {
		kf = "chunk sans image-cle"
	}
	prec := "pas de bloc precedent"
	if p, ok := d.b.entree(d.chunkPrec, slot); ok && d.b.entrees[d.chunkPrec] != nil {
		switch {
		case cmAlloueSous(p, g) && p.Vivante():
			prec = "vivant au bloc precedent (meme gen)"
		case cmAlloueSous(p, g):
			prec = "trace au bloc precedent (meme gen)"
		default:
			prec = "absent du bloc precedent (ne pendant le chunk precedent)"
		}
	}
	vue := fmt.Sprintf("masque par vue : bloc %#x", cur.MasqueVue)
	if nxOK {
		vue += fmt.Sprintf(", suivant %#x", nx.MasqueVue)
	}
	ti, pont := d.b.pontDuBloc(cur)
	d.t.add("p1_detail", cmJoindre(fmt.Sprintf("g=%d", g), kf, fmt.Sprintf("ecarte par la marche : %v", d.kfEcartes[slot]),
		fmt.Sprintf("lie a la fin du chunk precedent : %v", d.finPrec[slot]), prec), k)
	d.t.add("p1_vue", cmJoindre(vue, pont), k)
	d.t.add("p1_pont", cmJoindre(pont, fmt.Sprintf("ti=%d", ti)), k)
	if pont == "resolu" {
		cle := [2]int{d.chunk, -1}
		d.imageCle[cle] = append(d.imageCle[cle], cmLiaison{eid: r.eid, ti: ti})
	}
}

// b3EnTeteDImageCle : 2 si l en-tete exact `[eid][field 0][ti < 50]` est a une position du
// payload, 1 si le meme slot y est sous une autre generation, 0 sinon.
func b3EnTeteDImageCle(pay []byte, eid uint32) int {
	out := 0
	total := len(pay) * 8
	slot := eid & 0x3fffffff
	for q := 0; q+64 <= total; q++ {
		id := uint32(source.BitsBourres(pay, q, 32)) //nolint:gosec // 32 bits
		if id&0x3fffffff != slot {
			continue
		}
		mot := source.BitsBourres(pay, q+32, 32)
		if mot>>6 != 0 || mot&63 >= objectArchetypeCount {
			continue
		}
		if id == eid {
			return 2
		}
		out = 1
	}
	return out
}

// pontDuBloc : l archetype que le pont masque -> archetype donne a une entree.
func (b *cmBlocs) pontDuBloc(e DatumEntry) (uint32, string) {
	if e.Composants == ([4]uint64{}) {
		return 0, "masque vide"
	}
	tis := b.dico[e.Composants]
	switch len(tis) {
	case 0:
		return 0, "masque inconnu"
	case 1:
		for ti := range tis {
			return ti, "resolu"
		}
	}
	return 0, "ambigu"
}

// controleImageCle : les entrees VIVANTES du bloc du chunk, par declaration de l image-cle et
// par bit de la vue du film (le temoin de P1).
func (d *b3Diag) controleImageCle() {
	for s, e := range d.b.entrees[d.chunk] {
		if !e.Vivante() {
			continue
		}
		_, decl := d.b.declares[d.chunk][uint32(s)] //nolint:gosec // slot
		d.t.un("p1_controle", cmJoindre(fmt.Sprintf("declare par l image-cle : %v", decl),
			fmt.Sprintf("lie au debut : %v", d.liesAuDebut[uint32(s)]), //nolint:gosec // slot
			fmt.Sprintf("masque par vue %#x", e.MasqueVue), fmt.Sprintf("g=%d", e.Gen)))
	}
}

// p5 cherche plus loin les eid introuvables.
func (d *b3Diag) p5(r *cmRejet, slot uint32, k cmCompte) {
	p := d.paquets[r.premier]
	meme := "absent du paquet du rejet"
	if b3ChercherNeuf(p.pay, r.eid, false, 0, p.d.FinVueB) {
		meme = "present dans le paquet du rejet, avant la fin de vue B"
	}
	prec := "absent du chunk precedent"
	for _, q := range d.prec {
		if b3ChercherNeuf(q.pay, r.eid, false, 0, len(q.pay)*8) {
			prec = "present dans le chunk precedent"
			break
		}
	}
	autreGen := "aucun NEW du slot sous une autre generation"
	for _, q := range d.paquets[:r.premier] {
		if b3ChercherNeuf(q.pay, r.eid, true, 0, len(q.pay)*8) {
			autreGen = "NEW du slot sous une autre generation avant le rejet"
			break
		}
	}
	kf := "image-cle : en-tete absent"
	for _, pay := range d.kfPays {
		if b3EnTeteDImageCle(pay, r.eid) == 2 {
			kf = "image-cle : en-tete exact present"
		}
	}
	_ = slot
	d.t.add("p5_detail", cmJoindre(r.classe, meme, prec, autreGen, kf), k)
}

// b3ChercherNeuf cherche un en-tete NEW `0 01 slot tete R(6)<50` de l eid entre deux bits ;
// `slotSeul` : le meme slot sous une AUTRE generation.
func b3ChercherNeuf(pay []byte, eid uint32, slotSeul bool, de, a int) bool {
	fin := min(a, len(pay)*8-woNewHeaderBits)
	for pos := max(de, 0); pos <= fin; pos++ {
		if source.BitsTolerants(pay, pos, 1) != 0 || source.BitsTolerants(pay, pos+1, 2) != 1 {
			continue
		}
		h := LireHandle(pay, pos+woNewTypeBits)
		e := h.Gen<<30 | h.Slot
		if slotSeul {
			if h.Slot != eid&0x3fffffff || e == eid {
				continue
			}
		} else if e != eid {
			continue
		}
		ti := source.BitsTolerants(pay, pos+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)
		if ti < objectArchetypeCount {
			return true
		}
	}
	return false
}

// dernierComposant : l archetype et le dernier composant lu du dernier record avant l en-tete
// rejete (le composant dont la largeur fausse aurait decale le curseur).
func (d *b3Diag) dernierComposant(p *cmPaquet) string {
	if len(p.recs) == 0 {
		return "aucun record lu avant le rejet"
	}
	r := p.recs[len(p.recs)-1]
	nom := "aucun composant"
	if n := len(r.Trace.Comps); n > 0 {
		nom = nomComposantBloquant(d.f.reg, int(r.TypeIndex), r.Trace.Comps[n-1].Index)
	}
	return fmt.Sprintf("%s ti=%d dernier composant %s", b3Genre(r.Type), r.TypeIndex, nom)
}
