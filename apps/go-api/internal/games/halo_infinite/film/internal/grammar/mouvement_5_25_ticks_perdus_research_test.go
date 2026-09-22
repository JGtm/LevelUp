//go:build research

package grammar

// mouvement_5_25_ticks_perdus_research_test.go — COMBIEN DE TICKS DE JOUEUR LE CALQUE DES ETATS
// PERD-IL ENCORE APRES LA TABLE ANTICIPEE ? (lot 5.25, mesure seule.)
//
// Le lot 5.23 a ferme 74,7 % des rejets et laisse 5 893 (25,3 %). L utilisateur a fait remarquer
// que 25 % des REJETS ne veut pas dire 25 % des DONNEES : cet instrument rend le chiffre qui
// compte — les ticks de bipede de JOUEUR attendus, lus et perdus — et rien d autre. Il ne
// propose aucune cause et aucun lot : la cause est deja nommee (D1 du 5.19, une entite nee et
// morte entre deux images-cles), et le pilote decide sur le chiffre.
//
// UNE PASSE, UN DECODAGE PAR PAQUET. La marche est celle de la PRODUCTION du calque
// (`ScanMovementStates`) : cadre de balayage du film, images-cles liees, table de datums liee,
// TABLE ANTICIPEE posee (5.23.3), trois rangs de vue. Elle emprunte `t519Marcher` (lot 5.19) —
// le seul marcheur du depot qui rende a la fois les records de la vue B et les curseurs de
// rang — et `a523IdRejete` (lot 5.23) pour l eid de l en-tete rejete. Aucune marche n est
// recopiee. Le crochet `Observation.EtatMouvementHook` fournit, dans le MEME decodage, les
// etats lus et la vitesse tenue : c est la regle du calque (le hook EST la grammaire).
//
// CE QU IL PUBLIE, EN CINQ TABLEAUX :
//
//	(a) TRAMES            total, fermees a reste NUL, abandonnees, en debordement, non
//	                      localisees ; bits lus / non lus, au total et par trame abandonnee
//	(b) POSITION DU REJET records lus avant le rejet contre l attendu mesure sur les trames
//	                      FERMEES voisines (+-5), et la part de la trame lue avant l abandon
//	(c) TICKS DE JOUEUR   par vie et par joueur : ticks attendus / lus / perdus (LE CHIFFRE)
//	(d) ETATS PERDUS      la borne haute des transitions que le calque peut manquer
//	(e) ENTITES REJETEES  duree de vie, nombre de rejets, evenement de tete du paquet
//
// Les tableaux (c), (d) et (e) vivent dans `mouvement_5_25_tableaux_research_test.go`
// (deplacement pur, seuil de 500 lignes par fichier).
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestTicks525$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// Les quatre classes d une trame delta. Elles reproduisent EXACTEMENT le partage du gate 5.16
// (`g516Paquet`) : `fermees a reste NUL` + `reste hors bourrage` + `debordements` = les paquets
// LOCALISES du film. La quatrieme, `non localisee`, est hors de ce total et comptee a part.
const (
	t525Fermee uint8 = iota
	t525Abandonnee
	t525Debordement
	t525NonLocalisee
)

// t525SeuilVitesse est le seuil, en m/s, au-dessus duquel la vitesse TENUE d un bipede le declare
// EN MOUVEMENT. La marche d un Spartan vaut ~4,5 m/s et le sprint ~6,4 ; un corps immobile lit
// une norme nulle. Le seuil est bas DELIBEREMENT : il doit inclure tout ce qui bouge, puisque
// c est l attendu qu il borne. La distribution des normes est publiee pour qu il soit relu.
const t525SeuilVitesse = 0.5

// t525VoisinsFermes est le rayon, en trames, de la fenetre sur laquelle l attendu de records
// d une trame abandonnee est mesure — sur les trames FERMEES de cette fenetre, et sur elles
// seules (une trame abandonnee n a pas d attendu observable).
const t525VoisinsFermes = 5

// t525FenetreEtat est le rayon, en trames, de la fenetre de voisinage d un changement d etat
// (tableau (d)) ; t525ZoneMin la longueur minimale d une zone abandonnee contigue pour qu un
// intervalle ENTIER puisse y tenir.
const (
	t525FenetreEtat = 5
	t525ZoneMin     = 5
)

// t525Trame est UNE trame delta du film, mesuree.
type t525Trame struct {
	chunk, paquet int
	ts            uint64
	bits, curseur int
	classe        uint8
	records       int
	rejetID       uint32
	rejet         bool
	// tete est le type de l evenement de tete du paquet, -1 quand le paquet n en porte pas.
	tete int
	// bipedes sont les slots `ti=35` DISTINCTS lus dans cette trame ; vitesses la norme de la
	// vitesse TENUE de chacun a cet instant (`i1` ne voyage que sur changement).
	bipedes  []uint32
	vitesses []float64
}

// t525Etat est UNE lecture d etat de mouvement, datee a la trame.
type t525Etat struct {
	trame int
	slot  uint32
	genre string
	actif bool
}

// t525Passe porte tout ce qu une passe rend.
type t525Passe struct {
	tr    []t525Trame
	etats []t525Etat
	// vit est la vitesse TENUE par slot pendant la marche (etat courant, pas un historique).
	vit map[uint32]float64
	// cur est l index de la trame que le crochet alimente.
	cur int
	// parTI[0] : records par archetype sur les trames FERMEES ; parTI[1] : sur les abandonnees.
	parTI [2]map[uint32]int
	// normes : l histogramme des normes de vitesse tenue au moment d une lecture de bipede.
	normes map[int]int
	tsVus  map[uint64]bool
	obs    *Observation
	tab    *TableAnticipee
}

// TestTicks525 joue la passe sur le film courant et publie les cinq tableaux.
func TestTicks525(t *testing.T) {
	tc := t516Cadre(t)
	p := t525Marcher(tc)
	t525TableauA(t, p)
	t525TableauB(t, p)
	v := t525Vies(t, tc, p)
	t525TableauC(t, p, v)
	t525TableauD(t, p, v)
	t525TableauE(t, p)
}

// t525Marcher decode le film UNE fois, sous la marche de production du calque des etats.
func t525Marcher(tc t516Temoin) *t525Passe {
	p := &t525Passe{vit: map[uint32]float64{}, normes: map[int]int{}, tsVus: map[uint64]bool{}}
	p.parTI[0], p.parTI[1] = map[uint32]int{}, map[uint32]int{}
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	p.obs = obs
	w := NewWorld(tc.reg)
	// LA PRODUCTION, TELLE QUELLE : `ScanMovementStates` pose la table anticipee du film et
	// annonce son chunk courant avant chaque liaison (lot 5.23.3).
	p.tab = ConstruireTableAnticipee(tc.fc)
	w.PoserTableAnticipee(p.tab)
	obs.EtatMouvementHook = func(comp EtatMouvementComposant, slot uint32, v []uint64) {
		p.recevoir(w, comp, slot, v)
	}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			p.paquet(c, pk, data, w, cfg)
		}
	}
	return p
}

// t525Lier pose sur le monde ce que le chunk declare : les images-cles puis la table de datums.
// C est, a la ligne pres, `movementStateScanner.lierLeMonde` — la marche de production.
func t525Lier(w *World, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
			w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
	}
	LierTableDeDatums(w, data, pks)
}

// paquet mesure UNE trame delta.
func (p *t525Passe) paquet(c int, pk FilmPacket, data []byte, w *World, cfg FrameConfig) {
	pay := pk.Payload(data)
	tr := t525Trame{chunk: c, paquet: pk.Index, ts: pk.TimestampUS, bits: len(pay) * 8, tete: -1}
	p.tsVus[pk.TimestampUS] = true
	debut := DefaultPacketPreambleBits
	if typ, present := PacketHeadEventType(pay); present {
		tr.tete = typ
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			tr.classe = t525NonLocalisee
			p.tr = append(p.tr, tr)
			return
		}
	}
	p.cur = len(p.tr) // la trame que le crochet alimente est celle qu on s apprete a ajouter
	mar := t519Marcher(pay, w, cfg, debut)
	tr.curseur, tr.records = mar.m.FinVueC, len(mar.recs)
	switch reste := tr.bits - tr.curseur; {
	case reste < 0:
		tr.classe = t525Debordement
	case reste <= m5116GateOctet && c514ResteNul(pay, tr.curseur):
		tr.classe = t525Fermee
	default:
		tr.classe = t525Abandonnee
	}
	p.cumuler(&tr, mar.recs)
	if id, ok := a523IdRejete(pay, cfg, mar); ok {
		tr.rejetID, tr.rejet = id, true
	}
	p.tr = append(p.tr, tr)
}

// cumuler ventile les records d une trame : par archetype, et les slots de bipede DISTINCTS.
func (p *t525Passe) cumuler(tr *t525Trame, recs []FrameRecord) {
	idx := -1
	switch tr.classe {
	case t525Fermee:
		idx = 0
	case t525Abandonnee:
		idx = 1
	}
	for _, r := range recs {
		if idx >= 0 {
			p.parTI[idx][r.TypeIndex]++
		}
		if r.TypeIndex != BipedTypeIndex || t525Contient(tr.bipedes, r.Slot) {
			continue
		}
		v := p.vit[r.Slot]
		tr.bipedes = append(tr.bipedes, r.Slot)
		tr.vitesses = append(tr.vitesses, v)
		p.normes[int(v*2)]++ // pas de 0,5 m/s
	}
}

// t525Contient dit si un slot est deja dans la liste (les trois rangs de vue peuvent republier
// le meme corps ; un tick est un tick, pas deux).
func t525Contient(s []uint32, v uint32) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// recevoir capte UNE publication du deserialiseur : un etat lu, ou la vitesse qui sert d oracle
// de mouvement. Le filtre de slot est celui de la production (`movementStateScanner.recevoir`) :
// seuls les slots LIES AU BIPEDE comptent.
func (p *t525Passe) recevoir(w *World, comp EtatMouvementComposant, slot uint32, v []uint64) {
	if ti, ok := w.ArchetypeForSlot(slot); !ok || ti != BipedTypeIndex {
		return
	}
	var genre string
	var actif bool
	switch comp {
	case EtatVitesse:
		if len(v) < 4 || v[0] != 0 || v[1] != 0 {
			return // pleine precision non dequantifiee, ou vitesse ABSENTE de cet instant
		}
		vec := DecodeVelocity(v[2], v[3])
		p.vit[slot] = math.Sqrt(float64(vec[0])*float64(vec[0]) +
			float64(vec[1])*float64(vec[1]) + float64(vec[2])*float64(vec[2]))
		return
	case EtatAccroupi:
		genre, actif = "accroupi", len(v) >= 1 && v[0] != 0
	case EtatGlissade:
		genre, actif = "glissade", len(v) >= 1 && v[0] != 0
	case EtatMobilite:
		genre, actif = "escalade", len(v) >= 1 && v[0] != 0
	case EtatCapaciteActive:
		genre, actif = "sprint", len(v) >= 1 && v[0] == sprintAbilitySlotRaw
	default:
		return // posture, controle d unite : hors du perimetre du calque
	}
	if len(v) == 0 {
		return
	}
	p.etats = append(p.etats, t525Etat{trame: p.cur, slot: slot, genre: genre, actif: actif})
}

// ---------------------------------------------------------------------------------------------
// (a) LES TRAMES, ET CE QU ELLES LAISSENT SUR LA TABLE
// ---------------------------------------------------------------------------------------------

// t525TableauA publie le partage des trames et les bits non lus.
func t525TableauA(t *testing.T, p *t525Passe) {
	t.Helper()
	var n [4]int
	var bitsTot, bitsLus, bitsAband, bitsAbandLus int
	for _, tr := range p.tr {
		n[tr.classe]++
		if tr.classe == t525NonLocalisee {
			bitsTot += tr.bits
			continue
		}
		bitsTot += tr.bits
		bitsLus += tr.curseur
		if tr.classe == t525Abandonnee {
			bitsAband += tr.bits
			bitsAbandLus += tr.curseur
		}
	}
	loc := n[t525Fermee] + n[t525Abandonnee] + n[t525Debordement]
	t.Logf("(a) TRAMES : %d deltas · %d LOCALISEES · %d non localisees · %d horodatages distincts",
		len(p.tr), loc, n[t525NonLocalisee], len(p.tsVus))
	t.Logf("    fermees a reste NUL %d (%.1f %%) · ABANDONNEES %d (%.1f %%) · debordements %d",
		n[t525Fermee], m533bPart(n[t525Fermee], loc), n[t525Abandonnee],
		m533bPart(n[t525Abandonnee], loc), n[t525Debordement])
	t.Logf("    CONTROLE (gate 5.23) : rejets hors datum %d · de vue %d · liaisons par "+
		"anticipation %d", p.obs.RejetsHorsDatum, p.obs.RejetsDeVue,
		t525Somme(p.obs.LiaisonsParAnticipation))
	t.Logf("    BITS : %d au total · %d lus (%.1f %%) · %d NON LUS (%.1f %%)",
		bitsTot, bitsLus, m533bPart(bitsLus, bitsTot), bitsTot-bitsLus,
		m533bPart(bitsTot-bitsLus, bitsTot))
	t.Logf("    dont trames ABANDONNEES : %d bits, %d lus (%.1f %%), %d NON LUS",
		bitsAband, bitsAbandLus, m533bPart(bitsAbandLus, bitsAband), bitsAband-bitsAbandLus)
	t525Deciles(t, "    part de la trame LUE avant l abandon", p.tr, t525Abandonnee)
	t525HistNormes(t, p)
}

// t525Somme additionne un histogramme par archetype.
func t525Somme(m map[uint32]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// t525Deciles publie la distribution, par dixiemes, de la part de trame lue.
func t525Deciles(t *testing.T, titre string, trs []t525Trame, classe uint8) {
	t.Helper()
	var d [11]int
	n := 0
	for _, tr := range trs {
		if tr.classe != classe || tr.bits == 0 {
			continue
		}
		k := tr.curseur * 10 / tr.bits
		if k < 0 {
			k = 0
		}
		if k > 10 {
			k = 10
		}
		d[k]++
		n++
	}
	t.Logf("%s (%d trames) :", titre, n)
	for i, c := range d {
		if c == 0 {
			continue
		}
		t.Logf("      %3d-%3d %% : %6d (%.1f %%)", i*10, i*10+10, c, m533bPart(c, n))
	}
}

// t525HistNormes publie la distribution des normes de vitesse tenue au moment d une lecture —
// c est elle qui rend `t525SeuilVitesse` relisible.
func t525HistNormes(t *testing.T, p *t525Passe) {
	t.Helper()
	cles := make([]int, 0, len(p.normes))
	tot := 0
	for k, v := range p.normes {
		cles = append(cles, k)
		tot += v
	}
	sort.Ints(cles)
	var sb string
	for _, k := range cles {
		if k > 20 {
			continue
		}
		sb += fmt.Sprintf(" [%.1f-%.1f m/s]:%d", float64(k)/2, float64(k)/2+0.5, p.normes[k])
	}
	t.Logf("    VITESSE TENUE a la lecture (%d lectures de bipede) :%s", tot, sb)
}

// ---------------------------------------------------------------------------------------------
// (b) OU LE REJET TOMBE DANS LA TRAME
// ---------------------------------------------------------------------------------------------

// t525TableauB compare les records lus AVANT le rejet a l attendu mesure sur les trames FERMEES
// voisines (+-5). Une trame abandonnee n a pas d attendu observable ; ses voisines fermees en
// ont un, et c est le seul etalon qui ne soit pas un postulat.
func t525TableauB(t *testing.T, p *t525Passe) {
	t.Helper()
	var d [11]int
	n, sansVoisin := 0, 0
	sommeLus := 0
	sommeAttendu := 0.0
	for i, tr := range p.tr {
		if tr.classe != t525Abandonnee || !tr.rejet {
			continue
		}
		att, ok := t525AttenduVoisin(p.tr, i)
		if !ok {
			sansVoisin++
			continue
		}
		n++
		sommeLus += tr.records
		sommeAttendu += att
		k := int(float64(tr.records) / att * 10)
		if k < 0 {
			k = 0
		}
		if k > 10 {
			k = 10
		}
		d[k]++
	}
	t.Logf("(b) TRAMES ABANDONNEES SUR UN REJET : %d avec voisine fermee · %d sans",
		n, sansVoisin)
	t.Logf("    records lus AVANT le rejet : %d au total, %.2f par trame · ATTENDU (voisines "+
		"fermees +-%d) : %.0f, %.2f par trame · manque %.1f %%",
		sommeLus, float64(sommeLus)/math.Max(float64(n), 1), t525VoisinsFermes,
		sommeAttendu, sommeAttendu/math.Max(float64(n), 1),
		100-100*float64(sommeLus)/math.Max(sommeAttendu, 1))
	for i, c := range d {
		if c == 0 {
			continue
		}
		t.Logf("      part de l attendu lue %3d-%3d %% : %6d (%.1f %%)",
			i*10, i*10+10, c, m533bPart(c, n))
	}
	t525ParArchetype(t, p)
}

// t525AttenduVoisin rend le nombre MOYEN de records des trames FERMEES de la fenetre.
func t525AttenduVoisin(trs []t525Trame, i int) (float64, bool) {
	somme, n := 0, 0
	for j := i - t525VoisinsFermes; j <= i+t525VoisinsFermes; j++ {
		if j < 0 || j >= len(trs) || j == i || trs[j].classe != t525Fermee {
			continue
		}
		somme += trs[j].records
		n++
	}
	if n == 0 {
		return 0, false
	}
	return float64(somme) / float64(n), true
}

// t525ParArchetype publie ce que chaque population de trames rend, par archetype et par trame.
func t525ParArchetype(t *testing.T, p *t525Passe) {
	t.Helper()
	var nF, nA int
	for _, tr := range p.tr {
		switch tr.classe {
		case t525Fermee:
			nF++
		case t525Abandonnee:
			nA++
		}
	}
	cles := map[uint32]bool{}
	for k := range p.parTI[0] {
		cles[k] = true
	}
	for k := range p.parTI[1] {
		cles[k] = true
	}
	ordre := make([]int, 0, len(cles))
	for k := range cles {
		ordre = append(ordre, int(k))
	}
	sort.Slice(ordre, func(a, b int) bool {
		//nolint:gosec // index d archetype, borne par le registre
		return p.parTI[1][uint32(ordre[a])] > p.parTI[1][uint32(ordre[b])]
	})
	t.Logf("    RECORDS PAR ARCHETYPE, par trame : fermees (%d) contre abandonnees (%d)", nF, nA)
	for i, k := range ordre {
		if i >= 10 {
			break
		}
		f, a := p.parTI[0][uint32(k)], p.parTI[1][uint32(k)] //nolint:gosec // index d archetype
		t.Logf("      ti=%-3d fermee %7d (%.2f/trame) · abandonnee %7d (%.2f/trame)",
			k, f, float64(f)/math.Max(float64(nF), 1), a, float64(a)/math.Max(float64(nA), 1))
	}
}

// t525Ordre rend les cles d un histogramme de seaux dans l ordre des seaux (cf. `t525Seau`,
// `mouvement_5_25_tableaux_research_test.go`).
func t525Ordre(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
