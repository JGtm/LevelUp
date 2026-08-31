package replay

// visee_etiquettes_delta_test.go — LOT G : LA CORRELATION SUR LES RECORDS DELTA.
//
// CE FICHIER NE CONNAIT QUE DEUX CHOSES : la marche ancree du lot F (importee telle quelle) et
// la grille d'etiquettes du lot G. Tout ce qui est seuil, marge ou controle est declare dans
// l'en-tete de `visee_etiquettes_research_test.go` ; tout ce qui est grammaire de composant est
// dans `visee_composant_research_test.go`. Ici on ne fait que joindre les deux.
//
// LA SEULE DIFFERENCE AVEC LE LOT F, ET ELLE EST ENTIERE : les classes ne viennent plus d'une
// onde carree globale valable pour UN joueur sur 60 s, mais d'une grille PAR SLOT couvrant tout
// le film. Consequence directe sur le code : le masque de classes ne peut plus se construire a
// partir des seuls horodatages (`ondeCol.marque`), il lui faut le SLOT de chaque echantillon —
// d'ou `vgColonne`, qui est un `vfColonne` augmente du vecteur des slots, et `vgMarque`, qui est
// `marque` avec une etiquette par (slot, instant) au lieu d'une par instant.
//
// LE SCORE, LUI, N'EST PAS RETOUCHE : `ondeMasque`, `ondeCol.evalue`, `vfMeilleur`,
// `vfClassement` et `ondeScorePos` sont ceux des lots C et F. C'est la condition pour que le
// verdict de ce lot se compare a celui du lot F.
//
// LE PONT SLOT -> JOUEUR N'EST PLUS NECESSAIRE, et c'est un gain de methode a lui seul. Le lot F
// avait du rattacher a la main des fragments de vie anonymes pour savoir quels records etaient
// ceux de Nilton410 — un pont a trois volets, publie fragment par fragment, et qui laissait un
// slot non rattache. Ici les etiquettes sont indexees par SLOT, exactement comme les records :
// la question « qui est ce joueur » ne se pose plus.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// vgColonne est un composant transpose en colonnes de bits, avec le slot de chaque echantillon.
type vgColonne struct {
	idx     int
	nom     string
	col     *ondeCol
	slots   []uint32
	n       int
	offsets int
	largMin int
	largMax int
}

// vgCollecte deroule TOUT le film par la marche ANCREE du lot F et rend les records bipedes des
// slots etiquetes, decoupes en composants.
//
// Le film entier, pas une fenetre : le controle par translation compare le meme materiau a des
// etiquettes deplacees, il lui faut donc tout le materiau.
func vgCollecte(dir string, s vfSource, cibles map[uint32]bool) ([]vfRecord, vfStat) {
	st := vfNewStat()
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	opt.QuantaOnly = true
	var ancres []vfAncre
	filmdec.SetRecordMaskHook(func(idx []int, _ []byte, afterI0 int) {
		ancres = append(ancres, vfAncre{idx: append([]int(nil), idx...), afterI0: afterI0})
	})
	defer filmdec.SetRecordMaskHook(nil)

	var out []vfRecord
	for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			st.paquets++
			pay := p.Payload(data)
			ancres = ancres[:0]
			recs := filmdec.ScanBipedRecords(pay, cibles, s.lay, opt)
			out = append(out, vgVersePaquet(&st, recs, ancres, pay, p, s)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out, st
}

// vgVersePaquet marche les records ancres d'UN paquet.
//
// L'APPARIEMENT HOOK <-> RECORD EST EXACT : les deux listes viennent du meme appel non filtre a
// `ScanBipedRecords`, et un desaccord de longueur ecarte le paquet plutot que de decaler en
// silence — c'est la garde que le lot F a payee une fois pour toutes.
//
// PAS DE FILTRE PAR VIE ICI, contrairement au lot F, et c'est voulu : le lot F devait rattacher
// un record a une PERSONNE (le releve nomme un gamertag) ; ce lot n'a besoin que du SLOT, que le
// record et l'etiquette partagent. Un slot qui migre a la reapparition emporte ses etiquettes
// avec lui, puisqu'elles sont reconstruites du meme flux d'evenements.
func vgVersePaquet(st *vfStat, recs []filmdec.BipedPosition, ancres []vfAncre, pay []byte,
	p filmdec.FilmPacket, s vfSource,
) []vfRecord {
	st.bipeds += len(recs)
	if len(recs) != len(ancres) {
		st.desappaires++
		return nil
	}
	tMS := int64(p.TimestampUS / 1000)
	var out []vfRecord
	for i, r := range recs {
		st.records++
		comps := vfMarche(pay, ancres[i], s, st)
		vfCompteCouverture(st, ancres[i], comps)
		if len(comps) == 0 {
			continue
		}
		out = append(out, vfRecord{tMS: tMS, slot: r.Slot, comps: comps})
	}
	return out
}

// vgBatColonnes transpose les records en colonnes par composant, en gardant le slot de chaque
// echantillon.
//
// LES OFFSETS S'ARRETENT A LA LARGEUR MINIMALE OBSERVEE, comme au lot F : au-dela du prefixe
// commun, l'offset o ne designerait plus le meme champ d'un record a l'autre.
func vgBatColonnes(recs []vfRecord, echMin int) []vgColonne {
	parComp := map[int][]vfComp{}
	temps := map[int][]int64{}
	slots := map[int][]uint32{}
	for _, r := range recs {
		for _, c := range r.comps {
			parComp[c.idx] = append(parComp[c.idx], c)
			temps[c.idx] = append(temps[c.idx], r.tMS)
			slots[c.idx] = append(slots[c.idx], r.slot)
		}
	}
	idxs := make([]int, 0, len(parComp))
	for id := range parComp {
		idxs = append(idxs, id)
	}
	sort.Ints(idxs)
	var out []vgColonne
	for _, id := range idxs {
		if vc, ok := vgUneColonne(id, parComp[id], temps[id], slots[id], echMin); ok {
			out = append(out, vc)
		}
	}
	return out
}

// vgUneColonne construit la colonne d'UN composant, ou dit qu'il n'est pas recevable (S6).
func vgUneColonne(id int, cs []vfComp, temps []int64, slots []uint32, echMin int) (vgColonne, bool) {
	if len(cs) < echMin {
		return vgColonne{}, false
	}
	vc := vgColonne{idx: id, nom: cs[0].nom, slots: slots, n: len(cs),
		largMin: cs[0].larg, largMax: cs[0].larg}
	for _, c := range cs {
		if c.larg < vc.largMin {
			vc.largMin = c.larg
		}
		if c.larg > vc.largMax {
			vc.largMax = c.larg
		}
	}
	vc.offsets = vc.largMin
	if vc.offsets > vfOffsetMax {
		vc.offsets = vfOffsetMax
	}
	if vc.offsets <= 0 {
		return vgColonne{}, false
	}
	vc.col = vfTranspose(temps, cs, vc.offsets)
	return vc, true
}

// vgMarque construit les masques de classe d'une colonne pour des etiquettes translatees de
// delta. C'est `ondeCol.marque` avec une etiquette PAR (slot, instant) au lieu d'une par
// instant, et sans fenetre d'analyse : la grille couvre deja tout le film.
func vgMarque(c *vgColonne, g *vgGrille, delta int64) ondeMasque {
	m := ondeMasque{un: make([]uint64, c.col.mots), zero: make([]uint64, c.col.mots),
		w0: c.col.mots, w1: 0}
	for i, t := range c.col.temps {
		mot, bit := i/64, uint(i%64)
		switch g.classe(c.slots[i], t, delta) {
		case 1:
			m.un[mot] |= 1 << bit
			m.n1++
		case 0:
			m.zero[mot] |= 1 << bit
			m.n0++
		default:
			m.nGarde++
			continue
		}
		if mot < m.w0 {
			m.w0 = mot
		}
		if mot > m.w1 {
			m.w1 = mot
		}
	}
	return m
}

// vgBalaye evalue tous les composants d'un domaine a un decalage donne et rend la pointe
// globale — le meilleur couple (composant, offset relatif), toutes polarites confondues.
func vgBalaye(cols []vgColonne, g *vgGrille, delta int64, echMin int) vfPointe {
	out := vfPointe{comp: -1}
	for i := range cols {
		m := vgMarque(&cols[i], g, delta)
		if m.n1 < echMin || m.n0 < echMin {
			continue
		}
		out.retenus++
		s := vfMeilleur(cols[i].col, m)
		if s.pos >= 0 && s.score > out.score.score {
			out.comp, out.nom, out.score = cols[i].idx, cols[i].nom, s
			out.n1, out.n0 = m.n1, m.n0
		}
	}
	return out
}

// vgDomaine est un sous-ensemble DECLARE de composants (cf. l'en-tete du fichier de recherche).
type vgDomaine struct {
	nom     string
	retient func(vgColonne) bool
}

// vgDomainesDe rend le domaine COMPLET puis UN DOMAINE PAR COMPOSANT — meme decoupage qu'au lot
// F, pour la meme raison : la puissance se paie au nombre d'hypotheses, et un domaine large peut
// etre aveugle la ou un domaine etroit conclut.
func vgDomainesDe(cols []vgColonne) []vgDomaine {
	out := []vgDomaine{
		{"COMPLET (tous les composants atteints)", func(vgColonne) bool { return true }},
	}
	for _, c := range cols {
		id, nom := c.idx, c.nom
		etiquette := vgEtiquetteDomaine(id, nom)
		out = append(out, vgDomaine{etiquette, func(x vgColonne) bool { return x.idx == id }})
	}
	return out
}

// vgEtiquetteDomaine nomme un domaine mono-composant, en signalant celui que la phase 7 designe.
func vgEtiquetteDomaine(id int, nom string) string {
	e := fmt.Sprintf("i%d %s", id, nom)
	if nom == vfNomCommandTick {
		e += " [DESIGNE par la phase 7 : octet 6 de la commande joueur]"
	}
	return e
}

// vgFiltre rend le sous-ensemble de colonnes d'un domaine.
func vgFiltre(cols []vgColonne, d vgDomaine) []vgColonne {
	var out []vgColonne
	for _, c := range cols {
		if d.retient(c) {
			out = append(out, c)
		}
	}
	return out
}
