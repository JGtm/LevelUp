package replay

// totalcontrol_instant_d3ter_test.go — D3-ter, VERROU 2 : « EXACTEMENT 3 » MESURE EN GLISSANT.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D3-ter, VERROU 2 »). Ce qui suit l'applique, il ne le decide pas.
//
// # CE QUE CETTE MESURE CORRIGE DE D3-BIS
//
// D3-bis comptait les zones designees PAR MANCHE. Sur ce corpus il n'y a qu'UNE manche par film :
// l'unite du protocole n'existait pas, l'union etait prise sur tout le match, et un designateur
// PARFAIT qui ferait tourner son trio N fois aurait rendu 3N valeurs. La reformulation supprime
// l'unite fautive : les ROTATIONS sont les points de changement de la serie elle-meme, et le
// cardinal se lit A CHAQUE INSTANT entre deux rotations.
//
// # LE MEME INSTRUMENT SERT AUX DEUX CORPUS, ET C'EST LE POINT
//
// `-koth` releve la meme grandeur sur les films KOTH — ceux dont la voie designateur est elue
// 4 sur 4 et SERVIE EN PRODUCTION. C'est ce releve, et lui seul, qui fixe le plancher `N` de la
// precondition. Mesurer TC avec une definition et KOTH avec une autre ferait passer une
// difference de convention pour une difference de donnee.
//
// REGIME : garde `ZONE_FILM` (repertoire de chunks), UN FILM PAR PROCESSUS, lecture seule,
// AUCUNE base.
//
//	$env:ZONE_FILM="<cache>/film_chunks/66aa5f0b"
//	go test ./internal/analysis/replay/ -run TotalControlInstant -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// d3iMargeRotationMS : la fenetre EXCLUE de part et d'autre de chaque point de changement.
	// Valeur INCHANGEE depuis D3-bis : a la bascule, le jeu retire des zones et en pose
	// d'autres, et compter cette fenetre ferait mecaniquement un cardinal double.
	d3iMargeRotationMS = 2000
	// d3iZonesAttendues : le cardinal exige. Total Control active TROIS zones.
	d3iZonesAttendues = 3
	// d3iPartMinimale : part du temps exploitable a cardinal exactement 3. Seuil du superviseur,
	// recopie sans modification.
	d3iPartMinimale = 0.80
	// d3iCouvertureMin : part du match que la serie doit couvrir pour etre exploitable.
	d3iCouvertureMin = 0.50
)

// d3iSerie est ce que le sous-canal tag 5 chaine rend sur un film — la grandeur COMMUNE aux deux
// corpus, et l'unique definition de « emission » du protocole.
type d3iSerie struct {
	// parSlot : par slot porteur, les couples (instant match ms, valeur), tries.
	parSlot map[uint32][]d3iPoint
	// emissions : le total, tous slots confondus. C'est le nombre que la regle de `N` compare.
	emissions int
	// premierMS / dernierMS bornent la serie ; matchMS est la duree du match.
	premierMS, dernierMS, matchMS int64
}

// couverture rend la part du match que la serie couvre.
func (s d3iSerie) couverture() float64 {
	if s.matchMS <= 0 || s.emissions == 0 {
		return 0
	}
	return float64(s.dernierMS-s.premierMS) / float64(s.matchMS)
}

// d3iPoint est UNE emission datee sur l'horloge du match.
type d3iPoint struct {
	tMS int64
	v   uint64
}

// TestTotalControlInstantCardinal — LA MESURE DU VERROU 2. Un film par processus.
func TestTotalControlInstantCardinal(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	g := filmproc.Arm("d3ter-instant", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — mesure interrompue, ce film ne compte "+
			"NI POUR NI CONTRE", short, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", short, float64(g.Peak())/(1<<30))
	}()

	ser, ok := d3iSerieDe(t, dir, short)
	if !ok {
		return
	}
	t.Logf("%s : %d emission(s) de tag 5 chainees sur %d slot(s) ; serie [%d ; %d] ms sur un "+
		"match de %d ms — couverture %.1f %%", short, ser.emissions, len(ser.parSlot),
		ser.premierMS, ser.dernierMS, ser.matchMS, 100*ser.couverture())

	// LE RELEVE KOTH S'ARRETE ICI : il ne mesure pas le cardinal, il fixe le plancher.
	if ser.couverture() < d3iCouvertureMin {
		t.Logf("COUVERTURE %s : %.1f %% < %.0f %% — la serie ne couvre pas assez le match.",
			short, 100*ser.couverture(), 100*d3iCouvertureMin)
	}

	d3iMesureCardinal(t, short, ser)
}

// d3iSerieDe balaye `ti=13` et rend la serie du sous-canal tag 5 CHAINE, datee sur l'horloge du
// match.
func d3iSerieDe(t *testing.T, dir, short string) (d3iSerie, bool) {
	t.Helper()
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", short, err)
	}
	recs := objectiveevents.StatRecords(p2aSource(t, dir))
	debut, fin, ok := d3iBornesMatch(recs)
	if !ok {
		t.Logf("NON EXPLOITABLE %s : aucun enregistrement d'entite — le match n'a pas de duree. "+
			"NI POUR NI CONTRE.", short)
		return d3iSerie{}, false
	}
	sc, err := filmdec.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Fatalf("%s : proprietes ti=13 illisibles : %v", short, err)
	}
	ser := d3iSerie{parSlot: map[uint32][]d3iPoint{}, matchMS: fin - debut}
	for _, r := range tcDesignateurs(sc.Reads) {
		for _, e := range r {
			if e.Value == 0 { // la valeur zero n'est pas une designation
				continue
			}
			if e.TimestampUS < clockUS {
				continue
			}
			tMS := int64(e.TimestampUS-clockUS) / 1000
			ser.parSlot[e.Slot] = append(ser.parSlot[e.Slot], d3iPoint{tMS: tMS, v: e.Value})
		}
	}
	for s := range ser.parSlot {
		pts := ser.parSlot[s]
		sort.Slice(pts, func(i, j int) bool { return pts[i].tMS < pts[j].tMS })
		ser.parSlot[s] = pts
		ser.emissions += len(pts)
	}
	if ser.emissions == 0 {
		t.Logf("NON EXPLOITABLE %s : AUCUNE emission de tag 5 chainee. NI POUR NI CONTRE.", short)
		return d3iSerie{}, false
	}
	ser.premierMS, ser.dernierMS = d3iBornesSerie(ser.parSlot)
	return ser, true
}

// d3iBornesMatch rend les bornes du match sur l'horloge des enregistrements d'entite.
func d3iBornesMatch(recs []objectiveevents.StatRecord) (int64, int64, bool) {
	if len(recs) == 0 {
		return 0, 0, false
	}
	lo, hi := int64(recs[0].TimeMS), int64(recs[0].TimeMS)
	for _, r := range recs {
		if int64(r.TimeMS) < lo {
			lo = int64(r.TimeMS)
		}
		if int64(r.TimeMS) > hi {
			hi = int64(r.TimeMS)
		}
	}
	return lo, hi, hi > lo
}

// d3iBornesSerie rend la premiere et la derniere emission, tous slots confondus.
func d3iBornesSerie(parSlot map[uint32][]d3iPoint) (int64, int64) {
	premier, dernier := int64(1<<62), int64(-1)
	for _, pts := range parSlot {
		if len(pts) == 0 {
			continue
		}
		if pts[0].tMS < premier {
			premier = pts[0].tMS
		}
		if pts[len(pts)-1].tMS > dernier {
			dernier = pts[len(pts)-1].tMS
		}
	}
	return premier, dernier
}

// d3iMesureCardinal decoupe le temps aux POINTS DE CHANGEMENT, exclut leurs fenetres, et somme la
// duree passee a cardinal exactement 3.
func d3iMesureCardinal(t *testing.T, short string, ser d3iSerie) {
	t.Helper()
	changes := d3iPointsDeChangement(ser)
	finMS := ser.matchMS
	if ser.dernierMS > finMS {
		finMS = ser.dernierMS
	}
	segments := d3iSegments(changes, ser.premierMS, finMS)
	var exploitable, aTrois int64
	histo := map[int]int64{}
	for _, sg := range segments {
		d := sg.finMS - sg.debutMS
		if d <= 0 {
			continue
		}
		card := len(d3iEnsembleA(ser, sg.debutMS))
		exploitable += d
		histo[card] += d
		if card == d3iZonesAttendues {
			aTrois += d
		}
	}
	t.Logf("%s : %d point(s) de changement, %d segment(s) hors fenetre de rotation (+/- %d ms) ; "+
		"temps exploitable %d ms sur un match de %d ms",
		short, len(changes), len(segments), d3iMargeRotationMS, exploitable, ser.matchMS)
	for _, c := range d3iCardinauxTries(histo) {
		t.Logf("  cardinal %d : %d ms (%s du temps exploitable)",
			c, histo[c], d3iPart(histo[c], exploitable))
	}
	if exploitable <= 0 {
		t.Logf("NON EXPLOITABLE %s : aucun segment hors fenetre de rotation. NI POUR NI CONTRE.",
			short)
		return
	}
	part := float64(aTrois) / float64(exploitable)
	verdict := "SOUS LE SEUIL"
	if part >= d3iPartMinimale {
		verdict = "TENU"
	}
	t.Logf("SIGNAL %s : cardinal %d pendant %s du temps exploitable (seuil %.0f %%) — %s",
		short, d3iZonesAttendues, d3iPart(aTrois, exploitable), 100*d3iPartMinimale, verdict)
}

// d3iSegment est un intervalle sur lequel l'ensemble designe est CONSTANT et hors rotation.
type d3iSegment struct{ debutMS, finMS int64 }

// d3iPointsDeChangement rend les instants ou l'ensemble designe change : un slot y prend une
// valeur differente de la sienne. La PREMIERE emission d'un slot en est un — l'ensemble passe de
// « sans ce slot » a « avec ».
func d3iPointsDeChangement(ser d3iSerie) []int64 {
	vu := map[int64]bool{}
	for _, pts := range ser.parSlot {
		for i, p := range pts {
			if i == 0 || p.v != pts[i-1].v {
				vu[p.tMS] = true
			}
		}
	}
	out := make([]int64, 0, len(vu))
	for t := range vu {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// d3iSegments rend les intervalles de [debut ; fin] qui restent une fois retirees les fenetres
// de +/- marge autour de chaque point de changement.
func d3iSegments(changes []int64, debutMS, finMS int64) []d3iSegment {
	var out []d3iSegment
	cur := debutMS
	for _, c := range changes {
		lo, hi := c-d3iMargeRotationMS, c+d3iMargeRotationMS
		if hi <= cur {
			continue // fenetre deja derriere le curseur
		}
		if lo > cur {
			out = append(out, d3iSegment{cur, min64(lo, finMS)})
		}
		cur = hi
		if cur >= finMS {
			return out
		}
	}
	if cur < finMS {
		out = append(out, d3iSegment{cur, finMS})
	}
	return out
}

// d3iEnsembleA rend les valeurs EN VIGUEUR a l'instant t : pour chaque slot, sa derniere emission
// a `<= t`. Un slot qui n'a pas encore emis ne contribue pas.
func d3iEnsembleA(ser d3iSerie, tMS int64) map[uint64]bool {
	out := map[uint64]bool{}
	for _, pts := range ser.parSlot {
		v, ok := uint64(0), false
		for _, p := range pts {
			if p.tMS > tMS {
				break
			}
			v, ok = p.v, true
		}
		if ok {
			out[v] = true
		}
	}
	return out
}

// d3iCardinauxTries rend les cardinaux observes, tries — sans quoi le parcours de map rendrait
// une sortie differente a chaque execution.
func d3iCardinauxTries(histo map[int]int64) []int {
	out := make([]int, 0, len(histo))
	for c := range histo {
		out = append(out, c)
	}
	sort.Ints(out)
	return out
}

// d3iPart rend un taux lisible. UN TAUX SANS DENOMINATEUR N'EST PAS ZERO.
func d3iPart(n, d int64) string {
	if d == 0 {
		return "pas de denominateur"
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}

// min64 rend le plus petit des deux.
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
