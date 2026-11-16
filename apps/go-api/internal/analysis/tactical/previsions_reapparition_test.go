package tactical

import (
	"math"
	"reflect"
	"testing"
)

// poidsPrevision : un seul poids, la distance a l'ennemi le plus proche (positif : le jeu choisit
// un point loin des ennemis). Les autres variables n'influencent pas ces cas.
var poidsPrevision = [NbVariables]float64{VarEnnemiProche: 2}

// zoneRect est une zone rectangulaire du sol, de x0 a x1, sur y de -10 a 10, tranche de -5 a 5.
func zoneRect(idx int, x0, x1 float64) ZoneNommee {
	return ZoneNommee{
		VolumeIndex: idx, X: (x0 + x1) / 2,
		Polygone: [][2]float64{{x0, -10}, {x1, -10}, {x1, 10}, {x0, 10}},
		ZBas:     -5, ZHaut: 5,
	}
}

// zonesPrevision : la zone 1 couvre x de 0 a 30, la zone 2 x de 100 a 130.
func zonesPrevision() []ZoneNommee { return []ZoneNommee{zoneRect(1, 0, 30), zoneRect(2, 100, 130)} }

// repetitionsParPoint : chaque point d'un match est reapparu autant de fois, pour atteindre le plancher
// de reapparitions (ReapparitionsMinCartePrevision) avec peu de matchs et de points.
const repetitionsParPoint = 5

// evenementsCarte rend repetitionsParPoint reapparitions par match et par point.
func evenementsCarte(matchs []string, points ...Point3) []EvenementReapparition {
	var out []EvenementReapparition
	for _, m := range matchs {
		for _, p := range points {
			for i := 0; i < repetitionsParPoint; i++ {
				out = append(out, EvenementReapparition{MatchID: m, CarteID: "carte", Reel: p})
			}
		}
	}
	return out
}

var (
	pointZone1  = Point3{X: 10}
	pointZone2  = Point3{X: 110}
	troisAutres = []string{"x1", "x2", "x3"}
)

// matchPrevu : le joueur « a » (equipe 0) meurt a la frame 100 en (50, 0, 0) et reapparait a la frame
// 340 en zone 1 ; l'ennemi « e » (equipe 1) est en zone 2 jusqu'a la frame 200, en zone 1 ensuite.
// Pas de 33 ms : une seconde = 30 frames, donc des echantillons aux frames 100, 130, ..., 310.
func matchPrevu() EntreeOccupation {
	mort := Piste{XUID: "a", Equipe: 0, Points: []PointPiste{{T: 0, X: 50}, {T: 100, X: 50}}}
	vie := Piste{XUID: "a", Equipe: 0, Points: []PointPiste{{T: 340, X: pointZone1.X}, {T: 500, X: 12}}}
	ennemi := Piste{XUID: "e", Equipe: 1, Points: []PointPiste{{T: 0, X: 110}, {T: 200, X: 10}, {T: 500, X: 10}}}
	return EntreeOccupation{
		MatchID: "prevu", IntervalleFrameMs: 33, Pistes: []Piste{mort, vie, ennemi},
		DebutsDeCompte: []DebutCompte{{XUID: "a", Frame: 100}},
	}
}

func entreePrevision(evts []EvenementReapparition) EntreePrevision {
	return EntreePrevision{Match: matchPrevu(), MatchID: "prevu", Evenements: evts, Poids: poidsPrevision, Zones: zonesPrevision()}
}

func probaDeZone(e EchantillonPrevision, zone int) float64 {
	for _, z := range e.Zones {
		if z.Zone == zone {
			return z.P
		}
	}
	return 0
}

func TestPrevision_UneMortUneSecondeParEchantillon(t *testing.T) {
	r := PrevoirLesReapparitions(entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2)))
	if r.Statut != PrevisionPrete || len(r.Morts) != 1 {
		t.Fatalf("statut = %q, morts = %d, attendu une mort prevue", r.Statut, len(r.Morts))
	}
	m := r.Morts[0]
	if m.XUID != "a" || m.Mort.Frame != 100 || m.FrameReapparition != 340 {
		t.Errorf("mort = %+v, attendu le joueur a, mort a 100, reapparition a 340", m)
	}
	if m.Reel != pointZone1 || m.ZoneReelle != 1 {
		t.Errorf("reel = %+v zone %d, attendu le point de la zone 1", m.Reel, m.ZoneReelle)
	}
	var frames []int
	for _, e := range m.Echantillons {
		frames = append(frames, e.Frame)
	}
	if want := []int{100, 130, 160, 190, 220, 250, 280, 310}; !reflect.DeepEqual(frames, want) {
		t.Errorf("frames des echantillons = %v, attendu %v (une par seconde, la reapparition exclue)", frames, want)
	}
	if r.MatchsCarte != 4 || r.Candidats != 2 {
		t.Errorf("matchs = %d, candidats = %d, attendu 4 et 2", r.MatchsCarte, r.Candidats)
	}
}

func TestPrevision_LesProbabilitesSuiventLesPositionsAChaqueInstant(t *testing.T) {
	m := PrevoirLesReapparitions(entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2))).Morts[0]
	avant, apres := m.Echantillons[0], m.Echantillons[len(m.Echantillons)-1]
	// L'ennemi est en zone 2 a la mort : la zone 1, loin de lui, est la plus probable. Il passe en
	// zone 1 a la frame 200 : a la frame 310, c'est la zone 2 qui l'est.
	if probaDeZone(avant, 1) <= probaDeZone(avant, 2) {
		t.Errorf("frame %d : zone 1 = %v, zone 2 = %v, attendu la zone 1 devant", avant.Frame, probaDeZone(avant, 1), probaDeZone(avant, 2))
	}
	if probaDeZone(apres, 2) <= probaDeZone(apres, 1) {
		t.Errorf("frame %d : zone 1 = %v, zone 2 = %v, attendu la zone 2 devant", apres.Frame, probaDeZone(apres, 1), probaDeZone(apres, 2))
	}
	if avant.Zones[0].Zone != 1 || apres.Zones[0].Zone != 2 {
		t.Errorf("les zones se rendent de la plus probable a la moins probable : %+v puis %+v", avant.Zones, apres.Zones)
	}
}

func TestPrevision_LaSommeDesProbabilitesFaitUn(t *testing.T) {
	// Un troisieme point, hors de toute zone, retient une part : HorsZone la porte.
	horsZone := Point3{X: 500}
	m := PrevoirLesReapparitions(entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2, horsZone))).Morts[0]
	for _, e := range m.Echantillons {
		somme := e.HorsZone
		for _, z := range e.Zones {
			somme += z.P
		}
		if math.Abs(somme-1) > 0.003 {
			t.Errorf("frame %d : zones + hors zone = %v, attendu 1 (aux arrondis du millieme pres)", e.Frame, somme)
		}
		if e.HorsZone <= 0 {
			t.Errorf("frame %d : le point hors zone doit porter une part", e.Frame)
		}
	}
}

func TestPrevision_UneZoneRecoitLaSommeDeSesPoints(t *testing.T) {
	// Deux points dans la zone 1, un dans la zone 2, tous a egale distance de l'ennemi : la zone 1,
	// qui a deux points, recoit le double de la zone 2.
	e := entreePrevision(evenementsCarte(troisAutres, Point3{X: 5}, Point3{X: 15}, Point3{X: 105}))
	e.Match.Pistes[2].Points = []PointPiste{{T: 0, X: 1000}, {T: 500, X: 1000}}
	e.Poids = [NbVariables]float64{}
	m := PrevoirLesReapparitions(e).Morts[0]
	got := m.Echantillons[0]
	if p1, p2 := probaDeZone(got, 1), probaDeZone(got, 2); math.Abs(p1-2*p2) > 0.002 {
		t.Errorf("zone 1 = %v, zone 2 = %v, attendu deux fois plus pour la zone 1 (somme des points)", p1, p2)
	}
}

// TestPrevision_NeVoitPasLeMatchPrevu : les reapparitions du match prevu n'entrent ni dans les
// candidats ni dans la frequence. La prevision est la meme avec ou sans elles.
func TestPrevision_NeVoitPasLeMatchPrevu(t *testing.T) {
	autres := evenementsCarte([]string{"x1", "x2"}, pointZone1, pointZone2)
	fuite := evenementsCarte([]string{"prevu"}, pointZone1, pointZone2, Point3{X: 20})
	sans := PrevoirLesReapparitions(entreePrevision(autres))
	avec := PrevoirLesReapparitions(entreePrevision(append(append([]EvenementReapparition{}, autres...), fuite...)))
	if !reflect.DeepEqual(sans, avec) {
		t.Fatalf("les evenements du match prevu changent la prevision :\nsans = %+v\navec = %+v", sans, avec)
	}
	if avec.Candidats != 2 {
		t.Errorf("candidats = %d, attendu 2 : le point (20, 0, 0) n'est vu que dans le match prevu", avec.Candidats)
	}
	// Un point que le match prevu est seul a completer (un seul autre match l'a vu) n'est pas candidat.
	seulAvecUn := append(append([]EvenementReapparition{}, autres...), evenementsCarte([]string{"x1"}, Point3{X: 25})...)
	seulAvecUn = append(seulAvecUn, evenementsCarte([]string{"prevu"}, Point3{X: 25})...)
	if r := PrevoirLesReapparitions(entreePrevision(seulAvecUn)); r.Candidats != 2 {
		t.Errorf("candidats = %d, attendu 2 : un point vu dans un autre match et dans le match prevu n'est pas candidat", r.Candidats)
	}
}

func TestPrevision_CartePauvre(t *testing.T) {
	// Un seul autre match : deux matchs sur la carte, sous le seuil, et aucun point vu deux fois.
	r := PrevoirLesReapparitions(entreePrevision(evenementsCarte([]string{"x1"}, pointZone1, pointZone2)))
	if r.Statut != PrevisionCartePauvre || len(r.Morts) != 0 {
		t.Errorf("statut = %q, morts = %d, attendu carte pauvre sans prevision", r.Statut, len(r.Morts))
	}
	// Trois matchs et vingt reapparitions, mais aucun point vu dans deux autres matchs : pas de candidat.
	e := append(evenementsCarte([]string{"x1"}, pointZone1, Point3{X: 5}),
		evenementsCarte([]string{"x2"}, pointZone2, Point3{X: 105})...)
	if r := PrevoirLesReapparitions(entreePrevision(e)); r.Statut != PrevisionCartePauvre || r.Candidats != 0 {
		t.Errorf("statut = %q, candidats = %d, attendu carte pauvre sans candidat", r.Statut, r.Candidats)
	}
}

// Les seuils sont ceux de la lecture Tactique (3 matchs, 20 reapparitions) : une reapparition de moins et la
// carte est pauvre, bien que les matchs et les candidats suffisent.
func TestPrevision_PlancherDeReapparitions(t *testing.T) {
	if MatchsMinCartePrevision != 3 || ReapparitionsMinCartePrevision != 20 {
		t.Fatalf("seuils = %d matchs, %d reapparitions, attendu 3 et 20 comme la lecture Tactique",
			MatchsMinCartePrevision, ReapparitionsMinCartePrevision)
	}
	evts := evenementsCarte(troisAutres[:2], pointZone1, pointZone2) // 2 matchs x 2 points x 5 = 20
	if r := PrevoirLesReapparitions(entreePrevision(evts)); r.Statut != PrevisionPrete {
		t.Errorf("statut = %q avec 20 reapparitions, attendu %q", r.Statut, PrevisionPrete)
	}
	if r := PrevoirLesReapparitions(entreePrevision(evts[1:])); r.Statut != PrevisionCartePauvre {
		t.Errorf("statut = %q avec 19 reapparitions, attendu %q", r.Statut, PrevisionCartePauvre)
	}
}

func TestPrevision_SansZone(t *testing.T) {
	e := entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2))
	e.Zones = nil
	if r := PrevoirLesReapparitions(e); r.Statut != PrevisionSansZone {
		t.Errorf("statut = %q, attendu %q", r.Statut, PrevisionSansZone)
	}
}

func TestPrevision_AucuneReapparition(t *testing.T) {
	e := entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2))
	// Une seule vie, sans mort suivie d'une reapparition.
	e.Match = EntreeOccupation{
		MatchID: "prevu", IntervalleFrameMs: 33,
		Pistes: []Piste{{XUID: "a", Equipe: 0, Points: []PointPiste{{T: 0, X: 50}, {T: 400, X: 50}}}, matchPrevu().Pistes[2]},
	}
	if r := PrevoirLesReapparitions(e); r.Statut != PrevisionAucuneReapparition || r.MortsEcartees != 0 {
		t.Errorf("statut = %q, ecartees = %d, attendu aucune reapparition", r.Statut, r.MortsEcartees)
	}
}

func TestPrevision_MortSansAdversaireEcartee(t *testing.T) {
	e := entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2))
	e.Match.Pistes = e.Match.Pistes[:2] // plus d'ennemi : rien a prevoir
	r := PrevoirLesReapparitions(e)
	if r.Statut != PrevisionAucuneReapparition || r.MortsEcartees != 1 {
		t.Errorf("statut = %q, ecartees = %d, attendu la mort comptee comme ecartee", r.Statut, r.MortsEcartees)
	}
}

func TestPrevision_Deterministe(t *testing.T) {
	e := entreePrevision(evenementsCarte(troisAutres, pointZone1, pointZone2, Point3{X: 500}))
	if a, b := PrevoirLesReapparitions(e), PrevoirLesReapparitions(e); !reflect.DeepEqual(a, b) {
		t.Errorf("deux executions rendent deux previsions differentes")
	}
}

func TestIndiceDeZone(t *testing.T) {
	zs := zonesPrevision()
	if got := IndiceDeZone(zs, Point3{X: 10}); got != 1 {
		t.Errorf("IndiceDeZone(10) = %d, attendu 1", got)
	}
	if got := IndiceDeZone(zs, Point3{X: 500}); got != -1 {
		t.Errorf("IndiceDeZone(500) = %d, attendu -1 (hors de toute zone)", got)
	}
	if got := IndiceDeZone(nil, Point3{X: 10}); got != -1 {
		t.Errorf("IndiceDeZone sans zone = %d, attendu -1", got)
	}
}
