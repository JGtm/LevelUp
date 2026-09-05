package replay

// vehicules_v6_etats_test.go — INSTRUMENT DE MESURE (lot V6) : CE QUE LA MACHINE D ETATS
// D OCCUPATION VOIT, ET OU ELLE PERD. LECTURE SEULE, garde V4_ROOT / V4_FILMS (le meme harnais
// que le lot V4 : un seul decodeur de contexte pour tout le chantier vehicule).
//
// IL NE FAIT QUE COMPTER. Aucun seuil n y est choisi ; il rend, film par film :
//
//	les EVENEMENTS bruts (embarquements, sorties, occupant en bande) ;
//	les EPISODES que la machine construit AVANT rattachement, par forme (deux bords
//	  d evenement, sortie seule, SILENCE TERMINAL) ;
//	combien de ces episodes trouvent un VEHICULE, et par QUELLE ancre (debut ou fin) ;
//	les TROUS, et combien survivent a la regle anti-doublon.
//
// LE TEMOIN : les memes ancres decalees de 60 s (le temoin du lot V4, meme valeur), qui dit
// combien d episodes un rattachement au hasard produirait.

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// TestV6EtatsOccupation : le bilan chiffre de la machine d etats, film par film.
func TestV6EtatsOccupation(t *testing.T) {
	root := v4Root(t)
	for _, f := range v4Corpus(t) {
		v6EtatsUnFilm(t, root, f)
	}
}

func v6EtatsUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	ctx, ok := v4Decode(t, root, f)
	if !ok {
		return
	}
	in := vehicleRideInputs{
		vehBySlot: ctx.vehBySlot, bipeds: ctx.bip, events: ctx.scan.Events,
		own: ctx.own, lives: ctx.lives, clock: ctx.clock,
	}
	boards, exits := vehicleEventsByOccupant(ctx.scan.Events)
	nb, ne := 0, 0
	for _, v := range boards {
		nb += len(v)
	}
	for _, v := range exits {
		ne += len(v)
	}
	t.Logf("== V6 ETATS %s (%s) ==", f.ID, f.Carte)
	t.Logf("  evenements bruts : %d · en bande : embarquements %d · sorties %d",
		len(ctx.scan.Events), nb, ne)
	bySlot := vehiclePositionsBySlot(ctx.bip)
	eps := vehicleEventEpisodes(boards, exits, bySlot)
	v6EtatsEpisodes(t, eps, bySlot, in)
	v6EtatsTrous(t, eps, bySlot, in)
}

// v6EtatsEpisodes detaille la construction et le rattachement des episodes d evenement.
func v6EtatsEpisodes(
	t *testing.T, eps []vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition,
	in vehicleRideInputs,
) {
	t.Helper()
	var deux, sortieSeule, terminal int
	var okDebut, okFin, perdus, okTemoin int
	for _, ep := range eps {
		switch {
		case ep.openEnd:
			terminal++
		case ep.borders >= 2:
			deux++
		default:
			sortieSeule++
		}
		pts := bySlot[ep.slot]
		a0, h0 := vehicleAnchorAt(pts, ep.startUS, false)
		if _, ok := vehicleLifeForAnchor(a0, h0, ep.startUS, in); ok {
			okDebut++
		} else if !ep.openEnd {
			a1, h1 := vehicleAnchorAt(pts, ep.endUS, true)
			if _, ok := vehicleLifeForAnchor(a1, h1, ep.endUS, in); ok {
				okFin++
			} else {
				perdus++
			}
		} else {
			perdus++
		}
		if h0 {
			b := a0
			b.TimestampUS += v4TemoinUS
			if _, ok := vehicleLifeForAnchor(b, true, ep.startUS+v4TemoinUS, in); ok {
				okTemoin++
			}
		}
	}
	t.Logf("  episodes construits : %d — deux bords %d · sortie seule %d · SILENCE TERMINAL %d",
		len(eps), deux, sortieSeule, terminal)
	t.Logf("  %s", v6EtatsDistances(eps, bySlot, in))
	t.Logf("  %s", v6EtatsRayons(eps, bySlot, in))
	t.Logf("  rattaches : par l ancre de DEBUT %d · par l ancre de FIN %d · PERDUS %d "+
		"(TEMOIN +60 s : %d)", okDebut, okFin, perdus, okTemoin)
}

// v6EtatsDistances rend la distance au vehicule le plus proche AUX DEUX ANCRES, episode par
// episode, avec son temoin decale. C est la MESURE qui doit preceder tout choix de rayon.
func v6EtatsDistances(
	eps []vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition, in vehicleRideInputs,
) string {
	s := "distances (m) au vehicule le plus proche — debut / fin / temoin+60s :"
	for _, ep := range eps {
		pts := bySlot[ep.slot]
		a0, h0 := vehicleAnchorAt(pts, ep.startUS, false)
		a1, h1 := vehicleAnchorAt(pts, ep.endUS, true)
		kind := "sortie"
		if ep.openEnd {
			kind = "TERMINAL"
		}
		b := a0
		b.TimestampUS += v4TemoinUS
		s += "\n     slot " + itoa32(ep.slot) + " " + kind +
			" · debut " + v6Dist(a0, h0, in) +
			" · fin " + v6Dist(a1, h1 && !ep.openEnd, in) +
			" · temoin " + v6Dist(b, h0, in)
	}
	return s
}

// v6RayonsM : les rayons compares pour l ANCRE D EVENEMENT. 1,5 m est la production du trou.
var v6RayonsM = []float64{1.5, 2, 3, 5, 8, 12}

// v6EtatsRayons compte, par rayon candidat, les episodes rattaches et l AMBIGUITE (un SECOND
// vehicule sous le meme rayon), avec le temoin decale de 60 s. C est la table qui justifie — ou
// refuse — d ouvrir le rayon de l ancre d evenement.
func v6EtatsRayons(
	eps []vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition, in vehicleRideInputs,
) string {
	s := "rattachement par rayon (ancre d evenement) :"
	for _, r := range v6RayonsM {
		ok, amb, temoin := 0, 0, 0
		for _, ep := range eps {
			pts := bySlot[ep.slot]
			a0, h0 := vehicleAnchorAt(pts, ep.startUS, false)
			a1, h1 := vehicleAnchorAt(pts, ep.endUS, true)
			n := v6CountWithin(a0, h0, in, r)
			if n == 0 && !ep.openEnd {
				n = v6CountWithin(a1, h1, in, r)
			}
			if n > 0 {
				ok++
			}
			if n > 1 {
				amb++
			}
			b := a0
			b.TimestampUS += v4TemoinUS
			if v6CountWithin(b, h0, in, r) > 0 {
				temoin++
			}
		}
		s += fmt.Sprintf("\n     R=%.1f m : rattaches %d/%d · ambigus %d · TEMOIN+60s %d",
			r, ok, len(eps), amb, temoin)
	}
	return s
}

// v6CountWithin compte les vehicules FRAIS sous le rayon a l instant de l echantillon.
func v6CountWithin(e filmdec.BipedPosition, has bool, in vehicleRideInputs, r float64) int {
	if !has {
		return 0
	}
	n := 0
	for slot := range in.vehBySlot {
		p, gap, ok := vehicleSampleNear(in.vehBySlot[slot], e.TimestampUS)
		if !ok || gap > vehicleNearestSampleUS {
			continue
		}
		if planDist(e.X, e.Y, p.X, p.Y) <= r {
			n++
		}
	}
	return n
}

// v6Dist rend la distance en plan au vehicule le plus proche a l instant de l echantillon, ou
// « - » quand il n y a pas d ancre / pas de vehicule frais.
func v6Dist(e filmdec.BipedPosition, has bool, in vehicleRideInputs) string {
	if !has {
		return "-"
	}
	best, found := 0.0, false
	for slot := range in.vehBySlot {
		p, gap, ok := vehicleSampleNear(in.vehBySlot[slot], e.TimestampUS)
		if !ok || gap > vehicleNearestSampleUS {
			continue
		}
		if d := planDist(e.X, e.Y, p.X, p.Y); !found || d < best {
			best, found = d, true
		}
	}
	if !found {
		return "aucun<1s"
	}
	return fmtM(best)
}

func fmtM(v float64) string { return fmt.Sprintf("%.1f", v) }

func itoa32(v uint32) string { return fmt.Sprintf("%d", v) }

// v6EtatsTrous compte les trous et ceux que la regle anti-doublon laisse passer.
func v6EtatsTrous(
	t *testing.T, eps []vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition,
	in vehicleRideInputs,
) {
	t.Helper()
	var kept []vehicleEpisode
	for _, ep := range eps {
		if _, _, resolved, ok := vehicleRideFromEpisode(ep, bySlot, in); ok {
			kept = append(kept, resolved)
		}
	}
	gaps := vehicleGaps(in.bipeds)
	couverts, rattaches := 0, 0
	for _, g := range gaps {
		if vehicleEpisodeCovers(kept, g) {
			couverts++
			continue
		}
		vs, ok := vehicleNearestTo(g.last, in.vehBySlot)
		if !ok {
			continue
		}
		if _, ok := vehicleLifeAt(in.lives, vs, g.startUS); ok {
			rattaches++
		}
	}
	t.Logf("  trous >= 3 s : %d — deja couverts par un episode publie %d · repli rattache %d",
		len(gaps), couverts, rattaches)
	t.Logf("  TOTAL PUBLIE : %d episodes d evenement + %d de repli = %d",
		len(kept), rattaches, len(kept)+rattaches)
}
