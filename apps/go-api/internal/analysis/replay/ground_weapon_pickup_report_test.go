package replay

// ground_weapon_pickup_report_test.go — LES QUATRE PUBLICATIONS de la phase 2, separees de la
// lecture du film (`ground_weapon_pickup_research_test.go`) parce qu'elles ne decodent rien :
// elles ne font que compter, comparer et ecrire.
//
// CHAQUE MESURE PORTE SON TEMOIN, et c'est la seule raison pour laquelle elle est lisible :
//   - 2.1 : un critere de proximite qui se declenche partout ne date rien. Deux temoins le
//     disent — la distance du joueur le plus proche a un instant tire au sort PENDANT la
//     presence de l'objet, et la part de FENETRES de meme largeur, placees au hasard pendant
//     cette presence, qui contiennent elles aussi un passage a moins de 1,5 m ;
//   - 2.2 : l'oracle des loadouts se compare a un joueur tire au sort parmi les AUTRES a la
//     meme image-cle. Sans lui, un accord eleve pourrait n'etre que la popularite de l'arme.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// gwPickup21Tally compte le bornage, la datation et les deux temoins sur un sous-ensemble.
type gwPickup21Tally struct {
	N, Dated, Unknown, Never, NoLaterKF, NeverSeen int
	Dists, Widths, Witness                         []float64
	WitClose                                       int
	// WinDenom / WinHit / WinDated : le temoin de fenetre et la MESURE REELLE sur le MEME
	// denominateur. Comparer un temoin a une mesure prise sur une autre population serait
	// exactement le genre de comparaison flatteuse que ce chantier refuse.
	WinDenom, WinHit, WinDated int
}

// gwPickupReport21 publie le bornage, la datation, et les deux temoins — globalement puis par
// sous-ensemble, parce qu'une arme lachee a une mort et une arme de socle ne disparaissent pas
// pour la meme raison.
func gwPickupReport21(t *testing.T, f *gwPickupFilm, objs []gwPickupObject) {
	t.Helper()
	sets := []string{gwPickupSetAll, gwPickupSetAtRest, gwPickupSetDropped, gwPickupSetSwap}
	tally := map[string]*gwPickup21Tally{}
	for _, s := range sets {
		tally[s] = &gwPickup21Tally{}
	}
	for _, o := range objs {
		gwPickup21Add(f, o, tally[gwPickupSetAll], tally[gwPickupSet(o)])
	}
	for _, s := range sets {
		gwPickup21Log(t, s, tally[s])
	}
}

// gwPickup21Add compte UNE apparition dans les deux tallies (global et sous-ensemble).
func gwPickup21Add(f *gwPickupFilm, o gwPickupObject, all, set *gwPickup21Tally) {
	dInst, okInst := 0.0, false
	winHit, okWin := false, false
	if o.Status != gwPickupStatusNever {
		dInst, okInst = gwPickupWitnessInstant(f, o)
		winHit, okWin = gwPickupWitnessWindow(f, o)
	}
	for _, a := range []*gwPickup21Tally{all, set} {
		a.N++
		switch o.Status {
		case gwPickupStatusNever:
			a.Never++
			continue
		case gwPickupStatusDated:
			a.Dated++
			a.Dists = append(a.Dists, o.Picker.DistM)
		default:
			a.Unknown++
		}
		a.Widths = append(a.Widths, o.Bounds.WidthS())
		if o.Bounds.NoLaterKF {
			a.NoLaterKF++
		}
		if o.Bounds.SeenKF == 0 {
			a.NeverSeen++
		}
		if okInst {
			a.Witness = append(a.Witness, dInst)
			if dInst < originDropMaxDist {
				a.WitClose++
			}
		}
		if okWin {
			a.WinDenom++
			if winHit {
				a.WinHit++
			}
			if o.Status == gwPickupStatusDated {
				a.WinDated++
			}
		}
	}
}

// gwPickup21Log publie un sous-ensemble.
func gwPickup21Log(t *testing.T, set string, a *gwPickup21Tally) {
	t.Helper()
	t.Logf("2.1 BORNAGE [%s] — apparitions %d · DATEES %s · unknown %s · jamais ramassees %s"+
		" · sans image-cle posterieure %d · jamais recensees a une image-cle %d",
		set, a.N, gwPadsPart(a.Dated, a.N), gwPadsPart(a.Unknown, a.N),
		gwPadsPart(a.Never, a.N), a.NoLaterKF, a.NeverSeen)
	t.Logf("2.1 INTERVALLES [%s] (s) — %s || DISTANCES du ramasseur (m) — %s",
		set, gwPickupSpread(a.Widths), gwPickupSpread(a.Dists))
	t.Logf("2.1 TEMOIN d'instant [%s] (joueur le plus proche a un instant tire au sort PENDANT"+
		" la presence) — %s · dont sous %.1f m %s",
		set, gwPickupSpread(a.Witness), originDropMaxDist, gwPadsPart(a.WitClose, len(a.Witness)))
	t.Logf("2.1 TEMOIN de fenetre [%s] — MESURE (le vrai intervalle date) %s CONTRE TEMOIN"+
		" (fenetre de meme largeur, placee au hasard pendant la presence) %s · MEME denominateur",
		set, gwPadsPart(a.WinDated, a.WinDenom), gwPadsPart(a.WinHit, a.WinDenom))
}

// gwPickupWitnessInstant tire un instant au sort PENDANT la presence de l'objet — entre sa
// creation et la date retenue du ramassage — et rend la distance du joueur le plus proche.
func gwPickupWitnessInstant(f *gwPickupFilm, o gwPickupObject) (float64, bool) {
	fin := o.gwPickupDateUS()
	if fin <= o.Appar.TUS {
		return 0, false
	}
	at := o.Appar.TUS + uint64(f.rng.Int63n(int64(fin-o.Appar.TUS)))
	return gwPickupNearestAt(o.Pos, at, f.bySlot)
}

// gwPickupWitnessWindow place une fenetre de la MEME largeur que l'intervalle de bornage, au
// hasard pendant la presence PROUVEE de l'objet (de sa creation a la derniere image-cle qui le
// recense), et dit si elle contient elle aussi un passage sous 1,5 m.
//
// C'EST LE TEMOIN QUI DECIDE SI LA DATATION VEUT DIRE QUELQUE CHOSE. Si une fenetre quelconque
// de meme largeur contient presque toujours un passage, alors le passage trouve dans le VRAI
// intervalle n'est pas le ramassage : c'est le premier passant.
func gwPickupWitnessWindow(f *gwPickupFilm, o gwPickupObject) (bool, bool) {
	w := o.Bounds.HighUS - o.Bounds.LowUS
	if w == 0 || o.Bounds.LowUS <= o.Appar.TUS || o.Bounds.LowUS-o.Appar.TUS <= w {
		return false, false
	}
	start := o.Appar.TUS + uint64(f.rng.Int63n(int64(o.Bounds.LowUS-o.Appar.TUS-w)))
	return gwPickupNearestPass(o.Pos, start, start+w, f.positions, originDropMaxDist).Found, true
}

// gwPickupOracleTally compte l'oracle des loadouts sur un sous-ensemble de ramassages.
type gwPickupOracleTally struct {
	Denom, Agree, Already, DenomNew, AgreeNew, Absent, SansKF, WitDenom, WitAgree int
}

// gwPickupSetAll et les trois sous-ensembles de l'item 1.1. LE DECOUPAGE N'EST PAS UN CONFORT :
// une arme `dropped` (lachee a une mort) et une arme de socle ne se ramassent pas de la meme
// facon, et publier un taux global sans ce detail masquerait lequel des deux l'oracle refuse.
const (
	gwPickupSetAll     = "tous"
	gwPickupSetAtRest  = "at_rest" // socles : `spawned` sans vie delta (jeu de l'item 1.2)
	gwPickupSetDropped = "dropped" // lachee a une mort
	gwPickupSetSwap    = "echange" // `spawned` AVEC vie delta (decouverte 7)
)

// gwPickupSet range un ramassage dans son sous-ensemble.
func gwPickupSet(o gwPickupObject) string {
	switch {
	case o.Appar.Class == gwClassDropped:
		return gwPickupSetDropped
	case !o.Appar.HasDelta:
		return gwPickupSetAtRest
	default:
		return gwPickupSetSwap
	}
}

// gwPickupReport22 publie l'ORACLE DES LOADOUTS — le controle independant du gate 2.
func gwPickupReport22(t *testing.T, f *gwPickupFilm, objs []gwPickupObject) {
	t.Helper()
	tally := map[string]*gwPickupOracleTally{}
	for _, s := range []string{gwPickupSetAll, gwPickupSetAtRest, gwPickupSetDropped, gwPickupSetSwap} {
		tally[s] = &gwPickupOracleTally{}
	}
	for _, o := range objs {
		if o.Status != gwPickupStatusDated {
			continue
		}
		gwPickupTally(f, o, tally[gwPickupSetAll], tally[gwPickupSet(o)])
	}
	for _, s := range []string{gwPickupSetAll, gwPickupSetAtRest, gwPickupSetDropped, gwPickupSetSwap} {
		a := tally[s]
		t.Logf("2.2 ORACLE [%s] — denominateur (ramassage date + image-cle suivante) %d (sans"+
			" image-cle suivante %d) · ACCORD %s · slot du ramasseur ABSENT de l'image-cle %d"+
			" · portait DEJA la famille avant %s · ACCORD sur les cas NOUVEAUX %s · TEMOIN"+
			" (autre joueur, meme image-cle) %s",
			s, a.Denom, a.SansKF, gwPadsPart(a.Agree, a.Denom), a.Absent,
			gwPadsPart(a.Already, a.Denom), gwPadsPart(a.AgreeNew, a.DenomNew),
			gwPadsPart(a.WitAgree, a.WitDenom))
	}
	gwPickupReport22Coverage(t, f)
}

// gwPickupTally compte UN ramassage dans les deux tallies (global et sous-ensemble).
func gwPickupTally(f *gwPickupFilm, o gwPickupObject, all, set *gwPickupOracleTally) {
	kfAfter, ok := gwPickupNextAfter(f.kfTimes, o.Picker.TUS)
	if !ok {
		all.SansKF, set.SansKF = all.SansKF+1, set.SansKF+1
		return
	}
	fams, present := f.loadouts[kfAfter][o.Picker.Slot]
	porte := gwPickupHasFamily(fams, o.Appar.Family)
	avant := gwPickupCarriedBefore(f, o)
	autre, hasAutre := gwPickupOtherSlot(f, kfAfter, o.Picker.Slot)
	temoin := hasAutre && gwPickupHasFamily(f.loadouts[kfAfter][autre], o.Appar.Family)
	for _, a := range []*gwPickupOracleTally{all, set} {
		a.Denom++
		if !present {
			a.Absent++
		}
		if porte {
			a.Agree++
		}
		if avant {
			a.Already++
		} else {
			a.DenomNew++
			if porte {
				a.AgreeNew++
			}
		}
		if hasAutre {
			a.WitDenom++
			if temoin {
				a.WitAgree++
			}
		}
	}
}

// gwPickupReport22Coverage mesure la COUVERTURE de l'oracle, et surtout que les deux chaines
// parlent DU MEME ESPACE DE SLOTS : le slot d'un `BipedPosition` (qui designe le ramasseur) et
// celui d'un `KeyframeLoadout` (qui porte ses armes). Sans ce controle, un taux d'accord bas ne
// se distinguerait pas d'un decalage d'espace de slots — et le gate serait juge sur un artefact.
func gwPickupReport22Coverage(t *testing.T, f *gwPickupFilm) {
	t.Helper()
	repliques, lus, communs := 0, 0, 0
	for _, kf := range f.kfTimes {
		at := f.loadouts[kf]
		lus += len(at)
		for s, pts := range f.bySlot {
			if !gwPickupAliveAt(pts, kf) {
				continue
			}
			repliques++
			if _, ok := at[s]; ok {
				communs++
			}
		}
	}
	t.Logf("2.2 COUVERTURE de l'oracle — %d images-cles · slots de bipede repliques a +-100 ms"+
		" %d · loadouts lisibles %d · slots PORTANT LES DEUX %d (%s des repliques)",
		len(f.kfTimes), repliques, lus, communs, gwPadsPart(communs, repliques))
}

// gwPickupAliveAt dit si le slot a un echantillon de position a `gwPickupWitnessTolUS` pres de
// l'instant donne — le seul sens mesurable de « vivant et replique » ici.
func gwPickupAliveAt(pts []filmdec.BipedPosition, atUS uint64) bool {
	i := sort.Search(len(pts), func(i int) bool { return pts[i].TimestampUS >= atUS })
	for _, j := range []int{i - 1, i} {
		if j >= 0 && j < len(pts) && equipTimeGap(pts[j].TimestampUS, atUS) <= gwPickupWitnessTolUS {
			return true
		}
	}
	return false
}

// gwPickupCarriedBefore dit si le ramasseur portait deja la famille a la derniere image-cle
// PRECEDANT le ramassage. Un accord obtenu sur une arme deja portee ne prouve rien : c'est la
// distinction que le plan demande d'ecrire.
func gwPickupCarriedBefore(f *gwPickupFilm, o gwPickupObject) bool {
	i := sort.Search(len(f.kfTimes), func(i int) bool { return f.kfTimes[i] >= o.Picker.TUS })
	if i == 0 {
		return false
	}
	return gwPickupHasFamily(f.loadouts[f.kfTimes[i-1]][o.Picker.Slot], o.Appar.Family)
}

// gwPickupOtherSlot tire au sort un AUTRE slot de bipede present a l'image-cle — le temoin de
// l'oracle. L'ordre des candidats est rendu TOTAL avant le tirage : une map Go s'itere au
// hasard, et un temoin dont l'ordre varie n'est pas reproductible.
func gwPickupOtherSlot(f *gwPickupFilm, kfUS uint64, exclude uint32) (uint32, bool) {
	at := f.loadouts[kfUS]
	cand := make([]uint32, 0, len(at))
	for s := range at {
		if s != exclude {
			cand = append(cand, s)
		}
	}
	if len(cand) == 0 {
		return 0, false
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i] < cand[j] })
	return cand[f.rng.Intn(len(cand))], true
}

// gwPickupReport23 publie les lignes `PICKUP` — la PROPOSITION de format pour la phase 3, pas
// un contrat : le document de rejeu n'est pas touche par ce lot.
func gwPickupReport23(t *testing.T, f *gwPickupFilm, objs []gwPickupObject) {
	t.Helper()
	for _, o := range objs {
		slot := int64(-1)
		if o.Picker.Found {
			slot = int64(o.Picker.Slot)
		}
		t.Logf("PICKUP\t%s\t%s\t%d\t%.3f\t%.3f\t%.3f\t%s\t%d\t%d\t%d\t%s\t%t",
			f.film, f.mapName, o.gwPickupDateUS(), o.Pos[0], o.Pos[1], o.Pos[2],
			o.Appar.Family, slot, o.Bounds.LowUS, o.Bounds.HighUS, o.Status, o.Moved)
	}
}

// gwPickupReport24 re-mesure le cycle DEPUIS LE RAMASSAGE, socle par socle, sur le jeu
// `at_rest` de l'item 1.2 — et le compare au cycle d'APPARITION de l'item 1.3.
func gwPickupReport24(t *testing.T, f *gwPickupFilm, objs []gwPickupObject) {
	t.Helper()
	sel, src := make([]gwPadApparition, 0, len(objs)), make([]int, 0, len(objs))
	for i, o := range objs {
		if o.Appar.Class == gwClassSpawned && !o.Appar.HasDelta {
			sel, src = append(sel, o.Appar), append(src, i)
		}
	}
	pads, assign := gwPadsClusterAssign(sel, gwPadRadiusM)
	members := map[int][]int{}
	for j := range sel {
		members[assign[j]] = append(members[assign[j]], src[j])
	}
	socles, etablisAppar, etablisRamas, gagnes, sansRamassage := 0, 0, 0, 0, 0
	pool, byFamily := []float64{}, map[string][]float64{}
	for p := range pads {
		if len(pads[p].TS) < gwPadMinHits {
			continue
		}
		socles++
		gaps, manques := gwPickupPadGaps(objs, members[p])
		sansRamassage += manques
		pool = append(pool, gaps...)
		byFamily[pads[p].Family] = append(byFamily[pads[p].Family], gaps...)
		appar, ramas := gwPadsCycle(pads[p].TS), gwPadsCycleFromGaps(gaps)
		if appar.Established {
			etablisAppar++
		}
		if ramas.Established {
			etablisRamas++
			if !appar.Established {
				gagnes++
			}
		}
		t.Logf("PADCYCLE\t%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%d\t%.1f\t%.1f\t%t\t%d\t%.1f\t%.1f\t%.1f\t%t",
			f.film, f.mapName, pads[p].Family, pads[p].X, pads[p].Y, pads[p].Z,
			len(pads[p].TS), appar.MedianS, appar.SDS, appar.Established,
			len(gaps), ramas.MedianS, ramas.P10S, ramas.P90S, ramas.Established)
		gwPickupPadState(t, f, objs, members[p], pads[p])
	}
	t.Logf("2.4 CYCLE DEPUIS LE RAMASSAGE — socles `at_rest` %d · cycle d'APPARITION etabli %s"+
		" · cycle depuis le RAMASSAGE etabli %s · socles GAGNES %d · reapparitions dont le"+
		" ramassage precedent n'est pas date %d",
		socles, gwPadsPart(etablisAppar, socles), gwPadsPart(etablisRamas, socles), gagnes,
		sansRamassage)
	gwPickupPoolCycle(t, f, pool, byFamily)
}

// gwPickupPoolCycle publie les ecarts ramassage -> reapparition MIS EN COMMUN, tous socles
// confondus puis par famille.
//
// POURQUOI CE REGROUPEMENT EST LEGITIME, ET CE QU'IL NE REMPLACE PAS. Le verdict « etabli » de
// la decision 3 se prononce PAR SOCLE, et un socle vu deux fois ne donne qu'un seul ecart : il
// ne peut pas etre etabli, quelle que soit la valeur. Le regroupement ne rebaisse pas ce seuil —
// il ne le remplace pas non plus. Il repond a une AUTRE question, celle que l'arbitrage pose :
// l'horloge repart-elle du ramassage ? Si oui, les ecarts de socles DIFFERENTS se ressemblent.
func gwPickupPoolCycle(t *testing.T, f *gwPickupFilm, pool []float64, byFamily map[string][]float64) {
	t.Helper()
	c := gwPadsCycleFromGaps(pool)
	t.Logf("2.4 ECARTS MIS EN COMMUN (tous socles) — %s · p10 %.1f · ecart-type %.1f ·"+
		" stable au sens de la decision 3 : %t", gwPickupSpread(pool), c.P10S, c.SDS,
		c.Established)
	fams := make([]string, 0, len(byFamily))
	for k := range byFamily {
		fams = append(fams, k)
	}
	sort.Strings(fams)
	for _, k := range fams {
		g := gwPadsCycleFromGaps(byFamily[k])
		t.Logf("PADFAMCYCLE\t%s\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%.1f\t%t",
			f.film, f.mapName, k, len(byFamily[k]), g.MedianS, g.P10S, g.P90S, g.SDS,
			g.Established)
	}
}

// gwPickupPadState publie l'etat d'un socle DANS LE TEMPS : une ligne par occupation, de
// l'apparition de l'arme a l'instant ou le socle redevient vide (-1 = jamais).
func gwPickupPadState(
	t *testing.T, f *gwPickupFilm, objs []gwPickupObject, members []int, pad gwPadCluster,
) {
	t.Helper()
	ms := append([]int(nil), members...)
	sort.Slice(ms, func(i, j int) bool { return objs[ms[i]].Appar.TUS < objs[ms[j]].Appar.TUS })
	for _, i := range ms {
		o := objs[i]
		vide := int64(-1)
		if o.Status == gwPickupStatusDated {
			vide = int64(o.Picker.TUS)
		}
		t.Logf("PADSTATE\t%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%d\t%d\t%s",
			f.film, f.mapName, pad.Family, pad.X, pad.Y, pad.Z, o.Appar.TUS, vide, o.Status)
	}
}

// gwPickupSpread resume une distribution : n, mediane, p90 — les denominateurs toujours
// publies, jamais une mediane seule.
func gwPickupSpread(v []float64) string {
	if len(v) == 0 {
		return "n=0 (aucune mesure)"
	}
	return fmt.Sprintf("n=%d · mediane %.2f · p90 %.2f · max %.2f",
		len(v), gwPadsQuantile(v, 0.5), gwPadsQuantile(v, 0.90), gwPickupMax(v))
}

func gwPickupMax(v []float64) float64 {
	m := v[0]
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}
