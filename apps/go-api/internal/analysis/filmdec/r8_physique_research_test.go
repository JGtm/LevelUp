package filmdec

// r8_physique_research_test.go — MESURE 3 du lot R8 : L'ORACLE PHYSIQUE.
//
// LE PRINCIPE. Une pose `deployed` est une DATE. Si cette date est celle d'un USAGE, la
// CONSEQUENCE PHYSIQUE de l'usage doit se voir sur les trajectoires publiees, qui sont
// lues d'un canal totalement different (positions i0). L'oracle est donc INDEPENDANT du
// signal teste — il ne sert jamais de detecteur, seulement de juge.
//
//	PROPULSEUR  le PORTEUR est projete : bouffee de vitesse sur sa propre trajectoire.
//	REPULSEUR   un bipede VOISIN est pousse : bouffee de vitesse sur la sienne.
//
// SEUILS ET TEMOINS ECRITS AVANT LA MESURE (aucun n'a ete ajuste apres) :
//
//	v(f)       vitesse horizontale = distance(P(f), P(f+1)) / 100 ms, en m/s. Les frames
//	           non consecutives sont exclues (un trou n'est pas une teleportation).
//	pic        max de v sur [t0-1, t0+4] pour le porteur, [t0, t0+4] pour le voisin.
//	base       mediane de v sur TOUTE la piste concernee — sa vitesse ordinaire a elle.
//	voisin     autre slot, a <= 6,0 m en horizontal et <= 3,0 m d'altitude a l'instant t0.
//
// REGLE DE DECISION, POSEE D'AVANCE : l'hypothese « la pose date un usage » est ACCEPTEE
// pour une famille si la MEDIANE du pic de sa population depasse le P90 du pic du temoin
// aleatoire apparie, ET si le temoin « autres familles deployables » reste, lui, sous ce
// P90. Un seul des deux ne suffit pas : le premier sans le second mesurerait un artefact
// de la fenetre, pas un usage.
//
// TEMOINS : aleatoire apparie (10 instants tires dans les memes pistes, graine fixe),
// autres familles deployables (`wall`, `sensor`, ...), et poses `dropped` des memes
// familles (le porteur meurt : sa vitesse doit s'effondrer, pas bondir).
//
// TEMOIN POSITIF — SANS LUI LA MESURE NE VAUDRAIT RIEN. `grappleLines[]` publie des
// instants d'usage CERTAINS d'un equipement de MOBILITE (le grappin a son canal propre,
// lu ailleurs qu'ici). Si l'oracle ne voit pas la traction du grappin sur les memes
// trajectoires, il n'a aucune PUISSANCE et un negatif sur le propulseur ne dirait rien
// d'autre que « les pistes publiees sont trop lisses ». Ce temoin est donc la premiere
// chose a lire dans le tableau de sortie.
//
// DEUX FENETRES, parce qu'une datation peut etre decalee : `pic` etroit sur [-1, +4]
// (0,5 s) et `picL` large sur [-5, +10] (1,5 s). Un signal reel doit ressortir sur les
// deux ; un signal qui n'apparait que sur la fenetre large est du hasard elargi.

import (
	"math/rand"
	"sort"
	"testing"
)

const (
	// r8NeighbourRadiusM : rayon horizontal du voisinage du repulseur. 6,0 m — au-dessus de
	// la portee annoncee de l'effet, pour ne pas manquer une poussee par un rayon trop juste.
	r8NeighbourRadiusM = 6.0
	// r8NeighbourDZ : ecart d'altitude maximal d'un voisin (m).
	r8NeighbourDZ = 3.0
	// r8RandomPerTrack : instants aleatoires tires par piste pour le temoin apparie.
	r8RandomPerTrack = 10
	// r8Seed : graine du temoin aleatoire. Fixe — la mesure doit se rejouer a l'identique.
	r8Seed = 20260903
)

// r8Sample est une mesure d'oracle sur un instant : pic sur la fenetre etroite, pic sur
// la fenetre large, et vitesse ordinaire de la piste mesuree.
type r8Sample struct {
	peak, wide, base float64
}

// r8Pop est une population de mesures.
type r8Pop struct {
	self, near []r8Sample
	noNeigh    int
}

func (p *r8Pop) addSelf(s r8Sample) { p.self = append(p.self, s) }
func (p *r8Pop) addNear(s r8Sample) { p.near = append(p.near, s) }

// r8FilmSpeeds pre-calcule les vitesses et la base de chaque piste du film.
type r8FilmSpeeds struct {
	sp   []map[int]float64
	base []float64
}

func r8ComputeSpeeds(a *r8Artifact) *r8FilmSpeeds {
	f := &r8FilmSpeeds{sp: make([]map[int]float64, len(a.Tracks)), base: make([]float64, len(a.Tracks))}
	for i := range a.Tracks {
		f.sp[i] = r8Speeds(a.Tracks[i], a.FrameIntervalMs)
		v := make([]float64, 0, len(f.sp[i]))
		for _, x := range f.sp[i] {
			v = append(v, x)
		}
		f.base[i] = r8Median(v)
	}
	return f
}

// r8PointAt rend le point de la piste `ti` a la frame `f`, s'il existe.
func r8PointAt(tr r8Track, f int) (r8Point, bool) {
	i := sort.Search(len(tr.Points), func(k int) bool { return tr.Points[k].T >= f })
	if i < len(tr.Points) && tr.Points[i].T == f {
		return tr.Points[i], true
	}
	return r8Point{}, false
}

// r8SelfSample mesure l'oracle « le porteur bondit ».
func r8SelfSample(fs *r8FilmSpeeds, ti, t0 int) (r8Sample, bool) {
	peak, seen := r8PeakSpeed(fs.sp[ti], t0, -1, 4)
	if seen == 0 {
		return r8Sample{}, false
	}
	wide, _ := r8PeakSpeed(fs.sp[ti], t0, -5, 10)
	return r8Sample{peak: peak, wide: wide, base: fs.base[ti]}, true
}

// r8NearSample mesure l'oracle « un voisin est pousse » : le bipede d'un AUTRE slot le
// plus proche du point (x, y, z) a l'instant t0, dans le rayon annonce.
func r8NearSample(a *r8Artifact, fs *r8FilmSpeeds, p r8Placement, t0 int) (r8Sample, bool) {
	best, bestD := -1, r8NeighbourRadiusM
	for i := range a.Tracks {
		if a.Tracks[i].Slot == uint32(max(p.Owner, 0)) && p.Owner >= 0 {
			continue
		}
		pt, ok := r8PointAt(a.Tracks[i], t0)
		if !ok {
			continue
		}
		if dz := pt.Z - p.Z; dz > r8NeighbourDZ || dz < -r8NeighbourDZ {
			continue
		}
		if d := r8Dist2(p.X, p.Y, pt.X, pt.Y); d <= bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return r8Sample{}, false
	}
	peak, seen := r8PeakSpeed(fs.sp[best], t0, 0, 4)
	if seen == 0 {
		return r8Sample{}, false
	}
	wide, _ := r8PeakSpeed(fs.sp[best], t0, -5, 10)
	return r8Sample{peak: peak, wide: wide, base: fs.base[best]}, true
}

// r8Collect remplit une population depuis une pose.
func r8Collect(a *r8Artifact, fs *r8FilmSpeeds, p r8Placement, pop *r8Pop) {
	if p.Owner >= 0 {
		if ti := r8TrackAt(a.Tracks, uint32(p.Owner), p.T0); ti >= 0 { //nolint:gosec // Owner >= 0
			if s, ok := r8SelfSample(fs, ti, p.T0); ok {
				pop.addSelf(s)
			}
		}
	}
	if s, ok := r8NearSample(a, fs, p, p.T0); ok {
		pop.addNear(s)
	} else {
		pop.noNeigh++
	}
}

func TestR8OraclePhysique(t *testing.T) {
	corpus := r8LoadCorpus(t)
	pops := map[string]*r8Pop{}
	get := func(k string) *r8Pop {
		if pops[k] == nil {
			pops[k] = &r8Pop{}
		}
		return pops[k]
	}
	rng := rand.New(rand.NewSource(r8Seed)) //nolint:gosec // temoin reproductible, pas de crypto
	for _, a := range corpus {
		fs := r8ComputeSpeeds(a)
		hosts := map[int]bool{}
		for i := range a.Placements {
			p := a.Placements[i]
			k := r8PopKey(a, i, p)
			if k == "" {
				continue
			}
			r8Collect(a, fs, p, get(k))
			if p.Owner >= 0 {
				if ti := r8TrackAt(a.Tracks, uint32(p.Owner), p.T0); ti >= 0 { //nolint:gosec // Owner >= 0
					hosts[ti] = true
				}
			}
		}
		r8RandomWitness(a, fs, hosts, rng, get("TEMOIN aleatoire"))
		r8GrappleWitness(a, fs, get("POSITIF grappin"))
	}
	r8LogPhysique(t, pops)
}

// r8GrappleWitness mesure l'oracle sur les instants d'usage CERTAINS du grappin.
func r8GrappleWitness(a *r8Artifact, fs *r8FilmSpeeds, pop *r8Pop) {
	for _, g := range a.Grapples {
		ti := r8TrackAt(a.Tracks, g.Slot, g.T0)
		if ti < 0 {
			continue
		}
		if s, ok := r8SelfSample(fs, ti, g.T0); ok {
			pop.addSelf(s)
		}
	}
}

// r8PopKey nomme la population d'une pose, ou rend "" si la pose ne sert a rien ici.
func r8PopKey(a *r8Artifact, i int, p r8Placement) string {
	o := r8OriginOrUnknown(p.Origin)
	switch {
	case r8TargetFamilies[p.Family] && o == "deployed":
		if r8AtSocle(a.Placements, i) {
			return p.Family + " deployed (socle)"
		}
		return p.Family + " deployed"
	case r8TargetFamilies[p.Family] && o == "dropped":
		return "TEMOIN " + p.Family + " dropped"
	case r8DeployableFamilies[p.Family] && o == "deployed":
		return "TEMOIN autres deployables"
	}
	return ""
}

// r8RandomWitness tire `r8RandomPerTrack` instants dans chaque piste qui a heberge une
// pose cible — le temoin est ainsi APPARIE aux memes joueurs et aux memes films.
func r8RandomWitness(
	a *r8Artifact, fs *r8FilmSpeeds, hosts map[int]bool, rng *rand.Rand, pop *r8Pop,
) {
	idx := make([]int, 0, len(hosts))
	for ti := range hosts {
		idx = append(idx, ti)
	}
	sort.Ints(idx) // deterministe : l'ordre d'une map ne l'est pas
	for _, ti := range idx {
		pts := a.Tracks[ti].Points
		if len(pts) < 8 {
			continue
		}
		for n := 0; n < r8RandomPerTrack; n++ {
			pt := pts[rng.Intn(len(pts))]
			if s, ok := r8SelfSample(fs, ti, pt.T); ok {
				pop.addSelf(s)
			}
			fake := r8Placement{X: pt.X, Y: pt.Y, Z: pt.Z, Owner: int(a.Tracks[ti].Slot)}
			if s, ok := r8NearSample(a, fs, fake, pt.T); ok {
				pop.addNear(s)
			} else {
				pop.noNeigh++
			}
		}
	}
}

func r8LogPhysique(t *testing.T, pops map[string]*r8Pop) {
	t.Helper()
	keys := make([]string, 0, len(pops))
	for k := range pops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("oracle physique — pic de vitesse horizontale (m/s). PORTEUR : fenetres [-1,+4]" +
		" et [-5,+10]. VOISIN : [0,+4] et [-5,+10].")
	t.Logf("%-34s %5s %6s %6s %6s %6s | %5s %6s %6s %6s",
		"population", "nSelf", "medPic", "p90Pic", "medLrg", "medBas",
		"nVois", "medPic", "p90Pic", "medLrg")
	for _, k := range keys {
		p := pops[k]
		t.Logf("%-34s %5d %6.2f %6.2f %6.2f %6.2f | %5d %6.2f %6.2f %6.2f",
			k, len(p.self), r8Q(p.self, 0.5), r8Q(p.self, 0.9),
			r8QWide(p.self, 0.5), r8QBase(p.self, 0.5),
			len(p.near), r8Q(p.near, 0.5), r8Q(p.near, 0.9), r8QWide(p.near, 0.5))
	}
}

func r8Q(s []r8Sample, q float64) float64 {
	v := make([]float64, len(s))
	for i := range s {
		v[i] = s[i].peak
	}
	return r8Quantile(v, q)
}

func r8QWide(s []r8Sample, q float64) float64 {
	v := make([]float64, len(s))
	for i := range s {
		v[i] = s[i].wide
	}
	return r8Quantile(v, q)
}

func r8QBase(s []r8Sample, q float64) float64 {
	v := make([]float64, len(s))
	for i := range s {
		v[i] = s[i].base
	}
	return r8Quantile(v, q)
}
