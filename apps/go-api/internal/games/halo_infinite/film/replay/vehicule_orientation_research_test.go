package replay

// vehicule_orientation_research_test.go — LOT 5.2b.2 : `i2` EST-IL L AVANT DU CHASSIS ?
//
// # LA QUESTION, ET POURQUOI ELLE EST ROUVERTE
//
// Le cap publie d un vehicule (`vehicles[].samples[].h`) est l atan2 de sa VELOCITE
// (`vehicleHeadingOf`). Marche arriere, derapage, vol et arret sont donc faux, et l utilisateur
// a tranche (2026-09-19) : le chassis s oriente la ou l ARME pointe, c est-a-dire l avant du
// vehicule.
//
// `document_vehicles.go` porte une REFUTATION de `i2 object-forward-and-up-dynamic-precision`
// comme cap — medianes d ecart de 40 a 137 deg sur quatre films. Elle est PERIMEE sur deux
// points, et les deux sont ecrits au plan : `param_4` de `i2` etait FAUX sur les vieux builds
// avant le lot 5.1.7-a (il est desormais LU au registre du film), et surtout son ORACLE etait la
// VISEE DE L OCCUPANT (`i21` du bipede a bord), qui n est pas l avant du chassis — sur un
// Warthog le tireur regarde ou il veut.
//
// # L ORACLE DE CETTE MESURE-CI
//
// La DIRECTION DE DEPLACEMENT, sur les seuls echantillons ou le vehicule AVANCE nettement. Un
// vehicule qui avance a 10 m/s a son nez dans la direction de sa vitesse : c est le seul
// regime ou l avant du chassis est connu sans rien supposer.
//
// TROIS POPULATIONS SONT RENDUES, et la distinction est le point :
//
//	TOUTE VITESSE >= seuil    la population honnete. Elle contient la marche arriere, donc une
//	                          queue a 180 deg qui est un VRAI comportement, pas une erreur.
//	AVANCE (scalaire > 0)     la population ou l avant est connu. C est sur elle que se lit la
//	                          justesse de `i2` comme avant du chassis.
//	TEMOIN PAR PERMUTATION    le `forward` d un AUTRE chassis, au meme instant. Il doit rendre
//	                          une mediane d environ 90 deg, sinon la mesure ne mesure rien.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte, un film a la fois.
//
//	ATT_FILM=<depot>/data/cache V0_FILMS=4f77afc1:flood gulch \
//	  go test ./internal/games/halo_infinite/film/replay/ -run TestOrientationChassis -v -count=1

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// TestOrientationChassisContreDeplacement — LA MESURE.
func TestOrientationChassisContreDeplacement(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		ctx, ok := v4Decode(t, root, f)
		if !ok {
			continue
		}
		mesurerOrientationChassis(t, f.ID, ctx)
	}
}

// echantillonOrientation : un instant ou le film donne A LA FOIS l avant du chassis (`i2`) et sa
// velocite (`i1`).
type echantillonOrientation struct {
	slot    uint32
	tUS     uint64
	vitesse float64 // m/s, dans le plan du sol
	capVel  float64 // degres, atan2 de la velocite
	capFwd  float64 // degres, atan2 de l avant projete au sol
}

func mesurerOrientationChassis(t *testing.T, id string, ctx v4Ctx) {
	t.Helper()
	ech, avecFwd, total := echantillonsDOrientation(ctx)
	t.Logf("FILM %s — %d echantillon(s) de vehicule, %d portent `i2` (%.1f %%), %d portent `i1` ET `i2`",
		id, total, avecFwd, 100*float64(avecFwd)/float64(denomNonNul(total)), len(ech))
	if len(ech) == 0 {
		t.Logf("   rien a mesurer : le film ne porte pas les deux lectures au meme instant")
		return
	}
	rapide := filtrer(ech, func(e echantillonOrientation) bool { return e.vitesse >= vehicleMinSpeedMPS })
	avance := filtrer(rapide, func(e echantillonOrientation) bool {
		return math.Cos(radians(ecartAngulaire(e.capFwd, e.capVel))) > 0
	})
	journaliserPopulation(t, "TOUTE VITESSE >= seuil", rapide)
	journaliserPopulation(t, "AVANCE (scalaire > 0)", avance)
	journaliserTemoin(t, rapide)
	// DEUX FOIS ET TROIS FOIS LE SEUIL : si `i2` etait l avant du chassis, l ecart DIMINUERAIT
	// quand le vehicule va vite — un vehicule lance ne derape pas.
	for _, mult := range []float64{2, 3} {
		vite := filtrer(ech, func(e echantillonOrientation) bool {
			return e.vitesse >= mult*vehicleMinSpeedMPS
		})
		journaliserPopulation(t, fmt.Sprintf("VITESSE >= %.0fx seuil", mult), vite)
	}
	journaliserVerticalite(t, ctx)
	mesurerVecteursBruts(t, ctx)
}

// journaliserVerticalite : LA QUESTION QUI TRANCHE CE QUE LE CHAMP PORTE. Le composant s appelle
// `object-forward-and-UP` : si la direction de 19 bits capturee etait le vecteur HAUT et non
// l avant, elle serait quasi VERTICALE sur un vehicule terrestre, et sa projection au sol ne
// serait qu un bruit faiblement correle a la pente.
func journaliserVerticalite(t *testing.T, ctx v4Ctx) {
	t.Helper()
	var zs, plans []float64
	for _, p := range ctx.scan.Positions {
		fwd, ok := p.AimVector()
		if !ok {
			continue
		}
		zs = append(zs, math.Abs(float64(fwd[2])))
		plans = append(plans, math.Hypot(float64(fwd[0]), float64(fwd[1])))
	}
	if len(zs) == 0 {
		return
	}
	sort.Float64s(zs)
	sort.Float64s(plans)
	t.Logf("   %-24s |z| mediane %.3f · p90 %.3f · part |z| > 0.9 : %.1f %% · norme au sol mediane %.3f",
		"NATURE DU VECTEUR", quantile(zs, 0.50), quantile(zs, 0.90),
		100*part(zs, func(v float64) bool { return v > 0.9 }), quantile(plans, 0.50))
}

// echantillonsDOrientation rend les instants portant les deux lectures, plus les denominateurs.
func echantillonsDOrientation(ctx v4Ctx) (ech []echantillonOrientation, avecFwd, total int) {
	for _, p := range ctx.scan.Positions {
		total++
		fwd, okF := p.AimVector()
		if okF {
			avecFwd++
		}
		v, okV := p.VelocityVector()
		if !okF || !okV {
			continue
		}
		vitesse := math.Hypot(float64(v[0]), float64(v[1]))
		if vitesse == 0 || (fwd[0] == 0 && fwd[1] == 0) {
			continue // un avant strictement vertical n a pas de cap au sol
		}
		ech = append(ech, echantillonOrientation{
			slot: p.Slot, tUS: p.TimestampUS, vitesse: vitesse,
			capVel: degres(math.Atan2(float64(v[1]), float64(v[0]))),
			capFwd: degres(math.Atan2(float64(fwd[1]), float64(fwd[0]))),
		})
	}
	return ech, avecFwd, total
}

// journaliserPopulation rend la distribution de l ecart angulaire d une population.
func journaliserPopulation(t *testing.T, nom string, ech []echantillonOrientation) {
	t.Helper()
	if len(ech) == 0 {
		t.Logf("   %-24s aucun echantillon", nom)
		return
	}
	ecarts := make([]float64, 0, len(ech))
	for _, e := range ech {
		ecarts = append(ecarts, ecartAngulaire(e.capFwd, e.capVel))
	}
	sort.Float64s(ecarts)
	t.Logf("   %-24s n=%-6d mediane %5.1f deg · p75 %5.1f · p90 %5.1f · < 15 deg %5.1f %% · > 135 deg %5.1f %%",
		nom, len(ecarts), quantile(ecarts, 0.50), quantile(ecarts, 0.75), quantile(ecarts, 0.90),
		100*part(ecarts, func(d float64) bool { return d < 15 }),
		100*part(ecarts, func(d float64) bool { return d > 135 }))
}

// journaliserTemoin : LE NEGATIF. L avant d un AUTRE chassis, au meme rang dans la population.
// Une mesure dont le temoin ne rend pas ~90 deg ne mesure pas ce qu elle croit.
func journaliserTemoin(t *testing.T, ech []echantillonOrientation) {
	t.Helper()
	if len(ech) < 2 {
		return
	}
	var ecarts []float64
	for i, e := range ech {
		autre := ech[(i+len(ech)/2)%len(ech)]
		if autre.slot == e.slot {
			continue // un temoin doit permuter DEUX CHASSIS, pas un chassis avec lui-meme
		}
		ecarts = append(ecarts, ecartAngulaire(autre.capFwd, e.capVel))
	}
	if len(ecarts) == 0 {
		t.Logf("   %-24s aucun couple de chassis DIFFERENTS : temoin sans valeur", "TEMOIN (permutation)")
		return
	}
	sort.Float64s(ecarts)
	t.Logf("   %-24s n=%-6d mediane %5.1f deg · p90 %5.1f · < 15 deg %5.1f %%",
		"TEMOIN (permutation)", len(ecarts), quantile(ecarts, 0.50), quantile(ecarts, 0.90),
		100*part(ecarts, func(d float64) bool { return d < 15 }))
}

// ecartAngulaire rend l ecart NON SIGNE entre deux caps en degres, dans [0,180].
func ecartAngulaire(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func degres(rad float64) float64  { return rad * 180 / math.Pi }
func radians(deg float64) float64 { return deg * math.Pi / 180 }

// quantile sur une tranche DEJA triee.
func quantile(tri []float64, q float64) float64 {
	if len(tri) == 0 {
		return 0
	}
	i := int(q * float64(len(tri)-1))
	return tri[i]
}

// part rend la fraction d une tranche qui verifie un predicat.
func part(vals []float64, ok func(float64) bool) float64 {
	if len(vals) == 0 {
		return 0
	}
	n := 0
	for _, v := range vals {
		if ok(v) {
			n++
		}
	}
	return float64(n) / float64(len(vals))
}

func filtrer(ech []echantillonOrientation, ok func(echantillonOrientation) bool) []echantillonOrientation {
	out := make([]echantillonOrientation, 0, len(ech))
	for _, e := range ech {
		if ok(e) {
			out = append(out, e)
		}
	}
	return out
}

// denomNonNul evite une division par zero dans un pourcentage de couverture.
func denomNonNul(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}

// mesurerVecteursBruts : LE CHEMIN « KEEP » (mode 2) D i2, ses DEUX vec3 float32 — la seule
// lecture EXACTE de ce composant. Elle repond a la question que la direction de 19 bits laisse
// ouverte : l avant du chassis est-il DANS ce composant, sur un autre de ses chemins ?
//
// Le premier vecteur est confronte au deplacement ; le second aussi. Celui qui est l AVANT doit
// coller a la velocite quand le vehicule avance ; celui qui est le HAUT doit etre vertical.
func mesurerVecteursBruts(t *testing.T, ctx v4Ctx) {
	t.Helper()
	var n int
	parMode := map[uint8]int{}
	var z1, z2 []float64
	var e1, e2 []float64
	for _, p := range ctx.scan.Positions {
		parMode[p.FwdMode]++
		if !p.HasFwdVecs {
			continue
		}
		n++
		z1 = append(z1, math.Abs(float64(p.FwdVec1[2])))
		z2 = append(z2, math.Abs(float64(p.FwdVec2[2])))
		v, ok := p.VelocityVector()
		if !ok || math.Hypot(float64(v[0]), float64(v[1])) < vehicleMinSpeedMPS {
			continue
		}
		capVel := degres(math.Atan2(float64(v[1]), float64(v[0])))
		e1 = append(e1, ecartAngulaire(degres(math.Atan2(float64(p.FwdVec1[1]), float64(p.FwdVec1[0]))), capVel))
		e2 = append(e2, ecartAngulaire(degres(math.Atan2(float64(p.FwdVec2[1]), float64(p.FwdVec2[0]))), capVel))
	}
	t.Logf("   %-24s ventilation des chemins d i2 : mode0 %d · mode1 %d · mode2 %d ; %d vec3 LUS",
		"MODE 2 (keep)", parMode[0], parMode[1], parMode[2], n)
	if n == 0 {
		return
	}
	journaliserSerie(t, "     vec1 |z|", z1)
	journaliserSerie(t, "     vec2 |z|", z2)
	journaliserSerie(t, "     vec1 vs deplacement", e1)
	journaliserSerie(t, "     vec2 vs deplacement", e2)
}

// journaliserSerie rend mediane / p90 / part sous 15 d une serie de mesures.
func journaliserSerie(t *testing.T, nom string, vals []float64) {
	t.Helper()
	if len(vals) == 0 {
		t.Logf("   %-26s aucune valeur", nom)
		return
	}
	sort.Float64s(vals)
	t.Logf("   %-26s n=%-6d mediane %6.3f · p90 %6.3f · < 15 : %5.1f %%",
		nom, len(vals), quantile(vals, 0.50), quantile(vals, 0.90),
		100*part(vals, func(v float64) bool { return v < 15 }))
}

// TestOrientationChassisChemin30Bits — LE CHEMIN DOMINANT d i2 sur `ti=40` : la direction de
// 30 bits du mode « config ». Meme mesure, meme oracle, meme temoin que la direction de 19 bits.
func TestOrientationChassisChemin30Bits(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		ctx, ok := v4Decode(t, root, f)
		if !ok {
			continue
		}
		mesurerChemin30Bits(t, f.ID, ctx)
	}
}

func mesurerChemin30Bits(t *testing.T, id string, ctx v4Ctx) {
	t.Helper()
	var ech []echantillonOrientation
	var zs []float64
	porteurs := 0
	for _, p := range ctx.scan.Positions {
		fwd, okF := p.ForwardVector30()
		if !okF {
			continue
		}
		porteurs++
		zs = append(zs, math.Abs(float64(fwd[2])))
		v, okV := p.VelocityVector()
		if !okV || (fwd[0] == 0 && fwd[1] == 0) {
			continue
		}
		vitesse := math.Hypot(float64(v[0]), float64(v[1]))
		if vitesse == 0 {
			continue
		}
		ech = append(ech, echantillonOrientation{
			slot: p.Slot, tUS: p.TimestampUS, vitesse: vitesse,
			capVel: degres(math.Atan2(float64(v[1]), float64(v[0]))),
			capFwd: degres(math.Atan2(float64(fwd[1]), float64(fwd[0]))),
		})
	}
	t.Logf("FILM %s — CHEMIN 30 BITS : %d record(s) le portent, %d avec velocite", id, porteurs, len(ech))
	if len(ech) == 0 {
		return
	}
	sort.Float64s(zs)
	t.Logf("   %-24s |z| mediane %.3f · p90 %.3f · part |z| > 0.9 : %.1f %%",
		"NATURE DU VECTEUR", quantile(zs, 0.50), quantile(zs, 0.90),
		100*part(zs, func(v float64) bool { return v > 0.9 }))
	rapide := filtrer(ech, func(e echantillonOrientation) bool { return e.vitesse >= vehicleMinSpeedMPS })
	avance := filtrer(rapide, func(e echantillonOrientation) bool {
		return math.Cos(radians(ecartAngulaire(e.capFwd, e.capVel))) > 0
	})
	journaliserPopulation(t, "TOUTE VITESSE >= seuil", rapide)
	journaliserPopulation(t, "AVANCE (scalaire > 0)", avance)
	journaliserTemoin(t, rapide)
}
