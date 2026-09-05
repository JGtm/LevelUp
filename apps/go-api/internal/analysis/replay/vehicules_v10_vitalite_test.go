package replay

// vehicules_v10_vitalite_test.go — INSTRUMENT DE MESURE (lot V10, signal « EPAVE » : vitalite
// effondree en fin de vie). LECTURE SEULE, garde par V4_ROOT / V4_FILMS (le SOCLE du lot V4 est
// reutilise tel quel : c est celui qui reproduit le contexte de `BuildFromFilm`).
// Le RAPPORT vit dans `vehicules_v10_rapport_test.go` ; ce fichier ne fait que MESURER.
//
// LE CADRAGE, ET POURQUOI IL A CHANGE DEUX FOIS. Le lot V9 a rendu la vitalite `i4` de `ti=40`
// LISIBLE et a refute la destruction datee par « i4 atteint zero » : 3 vies sur 315. CE REFUS NE
// PROUVAIT RIEN — il testait un modele faux (« le dernier etat replique vaut zero »). Le modele
// retenu ici est celui de l EPAVE : un vehicule detruit n est pas forcement retire du monde a
// l instant de sa mort ; si sa carcasse persiste, le film la REPLIQUE, et la fin de vie porte
// alors un PALIER BAS, pas un point isole.
//
// LES QUATRE GRANDEURS MESUREES PAR VIE, toutes ecrites avant la mesure :
//
//	1. la DERNIERE valeur d `i4` de la fenetre de vie, et la PENTE sur v10PenteS secondes ;
//	2. le PALIER BAS TERMINAL : le nombre de lectures terminales CONSECUTIVES sous un seuil
//	   (v10SeuilBas et v10SeuilMoyen), et sa DUREE. Un vehicule qui explose et dont l epave
//	   persiste donne un palier de plusieurs secondes ; un vehicule intact qui cesse d etre
//	   recense n en donne aucun ;
//	3. la CINEMATIQUE : une carcasse ne roule plus. La vitesse vient de la velocite `i1` du MEME
//	   record (validee V1a.3, ecart median 1,7-2,1 deg au deplacement). Trois profils se
//	   distinguent — epave (vitalite basse + vitesse nulle), abandon intact (vitalite pleine +
//	   vitesse nulle), disparition en conduite (vitesse non nulle a la fin) ;
//	4. l OCCUPATION, qui donne les deux populations de controle :
//	     (A) CANDIDATES — la vie finit alors qu un episode est OUVERT ou vient de se fermer
//	         (dernier episode termine a moins de v10OccupeS de la fin du recensement) ;
//	     (B) TEMOIN INTACT — vies ABANDONNEES DE LONGUE DATE (aucun episode, ou dernier episode
//	         termine depuis plus de v10AbandonS). Le lot V3 a mesure qu un vehicule replique
//	         encore 13 a 36 s apres avoir ete quitte : ces vies-la finissent par MISE AU REPOS.
//
// L INSTANT PUBLIABLE, s il l est un jour, est le DEBUT DU PALIER (`chuteUS`), pas la fin de vie :
// c est la chute qui date la destruction, et elle est plus precise que la borne du recensement.
//
// TEMOIN PAR DECALAGE TEMPOREL : la meme grandeur lue v10TemoinS AVANT la fin de vie.
//
//	CGO_ENABLED=0 V4_ROOT=<depot>/data/cache \
//	  V4_FILMS="0d76e8f1:Behemoth,fccc61cd:Launch Site" \
//	  go test ./internal/analysis/replay/ -run '^TestV10VitaliteTerminale$' -v -timeout 180m

import (
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// SEUILS DE CLASSEMENT, ecrits AVANT toute mesure.
const (
	// v10OccupeS : un episode termine a moins de ce delai de la fin du recensement rend la vie
	// CANDIDATE. 5 s — l ordre de grandeur d une explosion suivie du retrait, et bien en deca des
	// 13 a 36 s de replication post-abandon mesurees au lot V3.
	v10OccupeS = 5.0
	// v10AbandonS : au-dela de ce delai depuis le dernier episode, la vie est ABANDONNEE (temoin
	// intact). 15 s — la borne basse des medianes de replication post-abandon du lot V3.
	v10AbandonS = 15.0
	// v10TemoinS : le decalage du temoin temporel.
	v10TemoinS = 30.0
	// v10PenteS : la fenetre sur laquelle la pente et la vitesse terminales sont calculees.
	v10PenteS = 5.0
	// v10SeuilBas / v10SeuilMoyen : les deux seuils de « vitalite effondree » demandes.
	v10SeuilBas   = 0.10
	v10SeuilMoyen = 0.25
	// v10ArretMPS : sous cette vitesse le vehicule est A L ARRET. 0,5 m/s — un ordre de grandeur
	// sous `vehicleMinSpeedMPS` (5 m/s), le seuil sous lequel la production refuse deja de tirer
	// un cap de la velocite.
	v10ArretMPS = 0.5
)

// v10Vie porte le verdict d UNE vie de vehicule.
type v10Vie struct {
	film       string
	slot, gen  uint32
	famille    string
	nI4        int
	lastFrac   float64 // derniere fraction de vie lue dans la fenetre
	lastUS     uint64  // instant de cette derniere lecture
	loUS, hiUS uint64  // bornes de la fenetre de vie
	minFrac    float64
	temoin     float64 // fraction lue v10TemoinS avant la fin de vie
	temoinOK   bool
	pente      float64 // (fraction/s) sur v10PenteS ; negatif = usure
	penteOK    bool
	// PALIER BAS TERMINAL, par seuil : nombre de lectures terminales consecutives sous le seuil,
	// duree en secondes, et instant de la PREMIERE d entre elles (= la CHUTE, l instant publiable).
	palierN  [2]int
	palierS  [2]float64
	chuteUS  [2]uint64
	vFin     float64 // vitesse mediane sur les v10PenteS dernieres secondes (m/s)
	vFinOK   bool
	vMax     float64 // vitesse maximale de la vie : a-t-elle roule ?
	vMaxOK   bool
	rides    int
	finApres float64 // (fin du recensement - fin du dernier episode) en s ; v10Inf si aucun episode
	classe   string  // "A_candidate" | "B_abandon" | "C_intermediaire"
}

// v10Seuils : les deux seuils indexes comme `palierN` / `palierS` / `chuteUS`.
var v10Seuils = [2]float64{v10SeuilBas, v10SeuilMoyen}

func TestV10VitaliteTerminale(t *testing.T) {
	root := v4Root(t)
	var all []v10Vie
	for _, f := range v4Corpus(t) {
		all = append(all, v10VitaliteUnFilm(t, root, f)...)
	}
	if len(all) == 0 {
		t.Skip("aucune vie mesurable")
	}
	v10Rapport(t, all)
}

func v10VitaliteUnFilm(t *testing.T, root string, f v0Film) []v10Vie {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	ctx, ok := v4Decode(t, root, f)
	if !ok {
		return nil
	}
	rides, _ := buildVehicleRides(vehicleRideInputs{
		vehBySlot: ctx.vehBySlot, bipeds: ctx.bip, events: ctx.scan.Events, own: ctx.own,
		lives:    ctx.lives,
		drawable: vehicleDrawableLives(ctx.lives, ctx.spawns, ctx.vehBySlot), clock: ctx.clock,
	})
	var out []v10Vie
	nA, nB, nC, sansI4 := 0, 0, 0, 0
	for _, l := range ctx.lives {
		v, ok := v10VieDe(f.ID, l, ctx, rides[l.key])
		if !ok {
			sansI4++
			continue
		}
		switch v.classe {
		case "A_candidate":
			nA++
		case "B_abandon":
			nB++
		default:
			nC++
		}
		out = append(out, v)
	}
	t.Logf("V10 %s — %d vies recensees · %d sans i4 · A(candidates) %d · B(abandon) %d · C(intermediaire) %d",
		f.ID, len(ctx.lives), sansI4, nA, nB, nC)
	return out
}

// v10Ech est une lecture de la vie : instant, fraction de vitalite, et vitesse quand `i1` la porte.
type v10Ech struct {
	us     uint64
	frac   float64
	v      float64
	vValid bool
}

// v10VieDe remplit le verdict d une vie. `false` = aucune lecture d `i4` dans sa fenetre.
func v10VieDe(film string, l vehicleLife, ctx v4Ctx, rd []VehicleRide) (v10Vie, bool) {
	v := v10Vie{film: film, slot: l.key.Slot, gen: l.key.Gen, minFrac: 1, rides: len(rd),
		loUS: l.loUS, hiUS: l.hiUS, vFin: -1, vMax: -1}
	if sp, has := ctx.spawns[l.key]; has {
		v.famille = vehicleFamilyOf(uint32(sp.MPPVal[filmdec.MPPWord32]))
	}
	series, vAll := v10SeriesDe(ctx.vehBySlot[l.key.Slot], l)
	if len(series) == 0 {
		return v, false
	}
	v.nI4 = len(series)
	last := series[len(series)-1]
	v.lastFrac, v.lastUS = last.frac, last.us
	for _, e := range series {
		if e.frac < v.minFrac {
			v.minFrac = e.frac
		}
	}
	v10RemplitTemoinEtPente(&v, series, last)
	v10RemplitPaliers(&v, series, last)
	v10RemplitVitesse(&v, series, vAll, last)
	v10Classe(&v, l, ctx, rd)
	return v, true
}

// v10SeriesDe extrait les lectures d `i4` de la fenetre de vie (triees), et A PART la serie des
// VITESSES de la meme fenetre — la velocite `i1` est portee par bien plus de records que la
// vitalite, et la restreindre aux records porteurs d `i4` perdrait la cinematique de fin.
func v10SeriesDe(pts []filmdec.BipedPosition, l vehicleLife) (series []v10Ech, vAll []v10Ech) {
	for _, p := range pts {
		if p.TimestampUS < l.loUS || p.TimestampUS > l.hiUS {
			continue
		}
		sp, spOK := 0.0, false
		if vec, ok := p.VelocityVector(); ok {
			sp, spOK = v10Norme(vec), true
			vAll = append(vAll, v10Ech{us: p.TimestampUS, v: sp, vValid: true})
		}
		if h, ok := p.HealthAt(); ok {
			series = append(series, v10Ech{us: p.TimestampUS, frac: float64(h), v: sp, vValid: spOK})
		}
	}
	sort.Slice(series, func(i, j int) bool { return series[i].us < series[j].us })
	sort.Slice(vAll, func(i, j int) bool { return vAll[i].us < vAll[j].us })
	return series, vAll
}

func v10Norme(v [3]float32) float64 {
	x, y, z := float64(v[0]), float64(v[1]), float64(v[2])
	return math.Sqrt(x*x + y*y + z*z)
}

func v10RemplitTemoinEtPente(v *v10Vie, series []v10Ech, last v10Ech) {
	cut := v10SubUS(last.us, uint64(v10TemoinS*1e6))
	for _, e := range series {
		if e.us <= cut {
			v.temoin, v.temoinOK = e.frac, true
		}
	}
	pcut := v10SubUS(last.us, uint64(v10PenteS*1e6))
	for _, e := range series {
		if e.us <= pcut {
			v.pente = (last.frac - e.frac) / (float64(last.us-e.us) / 1e6)
			v.penteOK = true
		}
	}
}

// v10RemplitPaliers remonte la serie DEPUIS LA FIN tant que la lecture reste sous le seuil : le
// palier est TERMINAL par construction (une chute suivie d une remontee n en est pas un).
func v10RemplitPaliers(v *v10Vie, series []v10Ech, last v10Ech) {
	for s, seuil := range v10Seuils {
		i := len(series) - 1
		for i >= 0 && series[i].frac <= seuil {
			i--
		}
		n := len(series) - 1 - i
		if n == 0 {
			continue
		}
		v.palierN[s] = n
		v.chuteUS[s] = series[i+1].us
		v.palierS[s] = float64(last.us-series[i+1].us) / 1e6
	}
}

// v10RemplitVitesse rend la vitesse MEDIANE des v10PenteS dernieres secondes et le MAXIMUM de la
// vie. Le maximum repond a « ce vehicule a-t-il roule » : une vie a l arret de bout en bout est un
// vehicule jamais pris, pas une epave.
func v10RemplitVitesse(v *v10Vie, series, vAll []v10Ech, last v10Ech) {
	_ = series
	cut := v10SubUS(last.us, uint64(v10PenteS*1e6))
	var fin []float64
	for _, e := range vAll {
		if !e.vValid {
			continue
		}
		if e.v > v.vMax || !v.vMaxOK {
			v.vMax, v.vMaxOK = e.v, true
		}
		if e.us >= cut && e.us <= last.us {
			fin = append(fin, e.v)
		}
	}
	if len(fin) > 0 {
		sort.Float64s(fin)
		v.vFin, v.vFinOK = fin[len(fin)/2], true
	}
}

// v10Classe pose la population de controle a partir de l occupation.
func v10Classe(v *v10Vie, l vehicleLife, ctx v4Ctx, rd []VehicleRide) {
	v.finApres = v10Inf
	if len(rd) > 0 {
		endUS := ctx.clock.origin + uint64(rd[len(rd)-1].T1)*ctx.clock.step
		v.finApres = (float64(l.lastUS) - float64(endUS)) / 1e6
	}
	switch {
	case v.finApres <= v10OccupeS:
		v.classe = "A_candidate"
	case v.finApres > v10AbandonS:
		v.classe = "B_abandon"
	default:
		v.classe = "C_intermediaire"
	}
}

// v10Inf marque « aucun episode d occupation » : la vie n a jamais ete conduite (dans ce que la
// primitive sait rattacher), donc elle est ABANDONNEE par construction.
const v10Inf = 1e9

func v10SubUS(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}
