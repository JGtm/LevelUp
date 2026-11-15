package leadshare

import "testing"

func ptr(v int) *int { return &v }

func ptr64(v int64) *int64 { return &v }

// equipesDeDeuxCamps : A (camp 0) mène aux frames 10 et 30, égalité aux frames 20 et 40.
func equipesDeDeuxCamps() []Team {
	return []Team{
		{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 2}}},
		{ID: ptr(1), Totals: []Tick{{20, 1}, {40, 2}}},
	}
}

func leadDe(r Result, camp int) int64 {
	for _, c := range r.Camps {
		if c.Camp == camp {
			return c.LeadMS
		}
	}
	return -1
}

func sommeDesTemps(r Result) int64 {
	total := r.TieMS
	for _, c := range r.Camps {
		total += c.LeadMS
	}
	return total
}

func TestMeasure_PartEnTeteSurLaFenetreJouee(t *testing.T) {
	r, reason := Measure(Input{
		Teams: equipesDeDeuxCamps(), CampCount: 2, FrameIntervalMS: 100,
		Window: Window{StartMS: 0, EndMS: 5000},
	})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.MeasuredMS != 5000 || r.TieMS != 3000 || leadDe(r, 0) != 2000 || leadDe(r, 1) != 0 {
		t.Errorf("mesure = %+v, attendu durée 5000, égalité 3000, camp 0 en tête 2000, camp 1 0", r)
	}
	if r.LeadChanges != 0 || r.SilentCamps != 0 {
		t.Errorf("retournements %d, camps muets %d, attendu 0 et 0", r.LeadChanges, r.SilentCamps)
	}
	if sommeDesTemps(r) != r.MeasuredMS {
		t.Errorf("la somme des temps (%d) n'est pas la durée mesurée (%d)", sommeDesTemps(r), r.MeasuredMS)
	}
}

func TestMeasure_LaFenetreBorneLesTemps(t *testing.T) {
	// Fenêtre [1500, 3500] : le palier de la frame 10 est déjà tenu au début, celui de la frame
	// 40 (4000 ms) tombe après la fin.
	r, reason := Measure(Input{
		Teams: equipesDeDeuxCamps(), CampCount: 2, FrameIntervalMS: 100,
		Window: Window{StartMS: 1500, EndMS: 3500},
	})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.MeasuredMS != 2000 || leadDe(r, 0) != 1000 || r.TieMS != 1000 {
		t.Errorf("mesure = %+v, attendu durée 2000, camp 0 en tête 1000, égalité 1000", r)
	}
	if sommeDesTemps(r) != r.MeasuredMS {
		t.Errorf("la somme des temps (%d) n'est pas la durée mesurée (%d)", sommeDesTemps(r), r.MeasuredMS)
	}
}

// Le pas du document n'est pas 100 ms partout : à 33 ms (30 images/s) la frame 10 vaut 330 ms, et
// ignorer le pas décalerait chaque palier. Les attendus ci-dessous ne valent que pour 33 ms.
func TestMeasure_AuPasDe33ms(t *testing.T) {
	// Paliers aux frames 10, 20, 30, 40 = 330, 660, 990, 1320 ms. Fenêtre [0, 1650].
	// Égalité [0,330], camp 0 en tête [330,660], égalité [660,990], camp 0 [990,1320], égalité [1320,1650].
	r, reason := Measure(Input{
		Teams: equipesDeDeuxCamps(), CampCount: 2, FrameIntervalMS: 33,
		Window: Window{StartMS: 0, EndMS: 1650},
	})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.MeasuredMS != 1650 || leadDe(r, 0) != 660 || leadDe(r, 1) != 0 || r.TieMS != 990 {
		t.Errorf("mesure = %+v, attendu durée 1650, camp 0 en tête 660, égalité 990 (au pas de 100 ms on lirait 2000 et 3000)", r)
	}
	if sommeDesTemps(r) != r.MeasuredMS {
		t.Errorf("la somme des temps (%d) n'est pas la durée mesurée (%d)", sommeDesTemps(r), r.MeasuredMS)
	}
}

func TestMeasure_AuPasDe33msLaFenetreBorneLesTemps(t *testing.T) {
	// Fenêtre [500, 1500] : le camp 0 tient déjà la tête depuis 330 ms, jusqu'à 660 (160 ms) ;
	// égalité [660,990] ; camp 0 [990,1320] ; égalité [1320,1500].
	r, reason := Measure(Input{
		Teams: equipesDeDeuxCamps(), CampCount: 2, FrameIntervalMS: 33,
		Window: Window{StartMS: 500, EndMS: 1500},
	})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.MeasuredMS != 1000 || leadDe(r, 0) != 490 || r.TieMS != 510 {
		t.Errorf("mesure = %+v, attendu durée 1000, camp 0 en tête 490, égalité 510", r)
	}
}

func TestMeasure_LesRetournementsSontBornesALaFenetreJouee(t *testing.T) {
	// 0 mène à 1000 ms, 1 passe devant à 2000, 0 à 3000, 1 à 4000 : trois retournements au match.
	teams := []Team{
		{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 3}}},
		{ID: ptr(1), Totals: []Tick{{20, 2}, {40, 4}}},
	}
	tout, _ := Measure(Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{0, 5000}})
	if tout.LeadChanges != 3 {
		t.Fatalf("fenêtre entière : %d retournements, attendu 3", tout.LeadChanges)
	}
	// Fenêtre [2500, 3500] : seul celui de 3000 ms est joué ; celui de 2000 précède, celui de 4000 suit.
	r, reason := Measure(Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{2500, 3500}})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.LeadChanges != 1 {
		t.Errorf("retournements = %d, attendu 1 (ceux de la fenêtre jouée seulement)", r.LeadChanges)
	}
	// Le meneur d'avant la fenêtre sert de repère : l'équipe 1, en tête depuis 2000 ms, est dépassée à 3000.
	if leadDe(r, 1) != 500 || leadDe(r, 0) != 500 {
		t.Errorf("mesure = %+v, attendu 500 ms en tête de chaque côté", r)
	}
}

func TestMeasure_UnRetournementAuPremierInstantDeLaFenetreEstJoue(t *testing.T) {
	teams := []Team{
		{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 3}}},
		{ID: ptr(1), Totals: []Tick{{20, 2}}},
	}
	// 0 mène à 1000, 1 à 2000, 0 à 3000. Fenêtre [2000, 3000[ : le retournement de 2000 est joué (borne
	// de début incluse), celui de 3000 non (borne de fin exclue, comme l'intégration des temps).
	r, _ := Measure(Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{2000, 3000}})
	if r.LeadChanges != 1 {
		t.Errorf("retournements = %d, attendu 1", r.LeadChanges)
	}
}

func TestMeasure_NommeLesCampsMuetsQuandLAppelantLesDit(t *testing.T) {
	teams := []Team{{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 2}}}}
	base := Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{0, 5000}}

	nomme := base
	nomme.SilentCampIDs = []int{1}
	if r, _ := Measure(nomme); len(r.SilentIDs) != 1 || r.SilentIDs[0] != 1 || r.SilentCamps != 1 {
		t.Errorf("camps muets = %v (%d), attendu [1]", r.SilentIDs, r.SilentCamps)
	}

	cas := map[string][]int{
		"sans identifiants":              nil,
		"trop d'identifiants":            {1, 2},
		"un identifiant qui a une série": {0},
		"un identifiant en double":       {1, 1},
	}
	for nom, ids := range cas {
		t.Run(nom, func(t *testing.T) {
			in := base
			in.SilentCampIDs = ids
			r, reason := Measure(in)
			if reason != ReasonNone {
				t.Fatalf("refus inattendu : %s", reason)
			}
			if r.SilentIDs != nil {
				t.Errorf("camps muets nommés = %v : une liste qui ne coïncide pas avec les camps muets ne se nomme pas", r.SilentIDs)
			}
			if r.SilentCamps != 1 {
				t.Errorf("le compte des camps muets doit rester 1, lu %d", r.SilentCamps)
			}
		})
	}
}

func TestMeasure_AvantLePremierPalierLeScoreEstNulEtLeMatchALegalite(t *testing.T) {
	r, _ := Measure(Input{
		Teams: equipesDeDeuxCamps(), CampCount: 2, FrameIntervalMS: 100,
		Window: Window{StartMS: 0, EndMS: 1000},
	})
	if r.TieMS != 1000 || leadDe(r, 0) != 0 {
		t.Errorf("mesure = %+v, attendu 1000 ms à égalité avant le premier point", r)
	}
}

func TestMeasure_CompteLesRetournements(t *testing.T) {
	teams := []Team{
		{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 3}}},
		{ID: ptr(1), Totals: []Tick{{20, 2}, {40, 4}}},
	}
	r, reason := Measure(Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{0, 5000}})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	// 0 mène [1000,2000] et [3000,4000] ; 1 mène [2000,3000] et [4000,5000] ; égalité [0,1000].
	if r.LeadChanges != 3 || leadDe(r, 0) != 2000 || leadDe(r, 1) != 2000 || r.TieMS != 1000 {
		t.Errorf("mesure = %+v, attendu 3 retournements, 2000/2000 en tête, 1000 à égalité", r)
	}
}

func TestMeasure_UnCampMuetNeMenePasEtSeCompte(t *testing.T) {
	teams := []Team{{ID: ptr(0), Totals: []Tick{{10, 1}, {30, 2}}}}
	r, reason := Measure(Input{Teams: teams, CampCount: 2, FrameIntervalMS: 100, Window: Window{0, 5000}})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if r.SilentCamps != 1 || leadDe(r, 0) != 4000 || r.TieMS != 1000 {
		t.Errorf("mesure = %+v, attendu 1 camp muet, camp 0 en tête 4000, égalité 1000", r)
	}
}

func TestMeasure_PlusDeDeuxCampsLeMeneurEstLeSeulEnTete(t *testing.T) {
	teams := []Team{
		{ID: ptr(0), Totals: []Tick{{10, 2}}},
		{ID: ptr(1), Totals: []Tick{{10, 2}}},
		{ID: ptr(2), Totals: []Tick{{10, 1}, {20, 3}}},
	}
	r, _ := Measure(Input{Teams: teams, CampCount: 3, FrameIntervalMS: 100, Window: Window{0, 3000}})
	// Égalité en tête [0,2000] (zéro à zéro puis 2-2-1), camp 2 seul en tête [2000,3000].
	if r.TieMS != 2000 || leadDe(r, 2) != 1000 || leadDe(r, 0) != 0 || len(r.Camps) != 3 {
		t.Errorf("mesure = %+v", r)
	}
}

func TestMeasure_Refus(t *testing.T) {
	cas := []struct {
		nom  string
		in   Input
		want Reason
	}{
		{"série sans camp", Input{
			Teams:           []Team{{ID: nil, Totals: []Tick{{10, 1}}}, {ID: ptr(1), Totals: []Tick{{20, 1}}}},
			CampCount:       2,
			FrameIntervalMS: 100, Window: Window{0, 1000},
		}, ReasonTeamUnresolved},
		{"deux séries du même camp", Input{
			Teams:           []Team{{ID: ptr(1), Totals: []Tick{{10, 1}}}, {ID: ptr(1), Totals: []Tick{{20, 1}}}},
			CampCount:       2,
			FrameIntervalMS: 100, Window: Window{0, 1000},
		}, ReasonTeamUnresolved},
		{"une série et un nombre de camps inconnu", Input{
			Teams:           []Team{{ID: ptr(0), Totals: []Tick{{10, 1}}}},
			FrameIntervalMS: 100, Window: Window{0, 1000},
		}, ReasonSingleCamp},
		{"aucun palier publié", Input{
			Teams:           []Team{{ID: ptr(0)}, {ID: ptr(1)}},
			CampCount:       2,
			FrameIntervalMS: 100, Window: Window{0, 1000},
		}, ReasonScoreNotCovered},
		{"pas de pas de grille", Input{
			Teams:     equipesDeDeuxCamps(),
			CampCount: 2, Window: Window{0, 1000},
		}, ReasonClockStepMissing},
		{"fenêtre vide", Input{
			Teams:           equipesDeDeuxCamps(),
			CampCount:       2,
			FrameIntervalMS: 100, Window: Window{1000, 1000},
		}, ReasonWindowIncoherent},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, got := Measure(c.in); got != c.want {
				t.Errorf("raison = %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestPlayedWindow(t *testing.T) {
	base := Clock{
		OriginMS: ptr64(5000), T0FilmMS: ptr64(20000), DurationSeconds: ptr64(600),
		FrameIntervalMS: 100, FrameCount: 10000,
	}
	w, reason := PlayedWindow(base)
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	// début = T0 film - origine ; fin = durée du match - origine.
	if w.StartMS != 15000 || w.EndMS != 595000 {
		t.Errorf("fenêtre = %+v, attendu [15000, 595000]", w)
	}

	court := base
	court.FrameCount = 1000 // le film s'arrête à 99 900 ms de son axe
	if w, _ := PlayedWindow(court); w.EndMS != 99900 {
		t.Errorf("fin = %d, attendu la dernière image du film (99900)", w.EndMS)
	}

	tard := base
	tard.T0FilmMS = ptr64(3000) // le premier paquet arrive APRÈS le coup d'envoi
	if w, _ := PlayedWindow(tard); w.StartMS != 0 {
		t.Errorf("début = %d, attendu 0 (rien avant l'image zéro)", w.StartMS)
	}
}

// Au pas de 33 ms la dernière image du film vaut (FrameCount-1) x 33 : ignorer le pas (100 ms) la
// placerait à 999 900 ms et la fin de la fenêtre tomberait sur la durée du match (595 000).
func TestPlayedWindow_AuPasDe33ms(t *testing.T) {
	w, reason := PlayedWindow(Clock{
		OriginMS: ptr64(5000), T0FilmMS: ptr64(20000), DurationSeconds: ptr64(600),
		FrameIntervalMS: 33, FrameCount: 10000,
	})
	if reason != ReasonNone {
		t.Fatalf("refus inattendu : %s", reason)
	}
	if w.StartMS != 15000 || w.EndMS != 329967 {
		t.Errorf("fenêtre = %+v, attendu [15000, 329967] (dernière image = 9999 x 33)", w)
	}
}

func TestPlayedWindow_Refus(t *testing.T) {
	cas := []struct {
		nom    string
		mutate func(*Clock)
		want   Reason
	}{
		{"origine absente", func(c *Clock) { c.OriginMS = nil }, ReasonClockOriginMissing},
		{"pas de grille absent", func(c *Clock) { c.FrameIntervalMS = 0 }, ReasonClockStepMissing},
		{"coup d'envoi non mesuré", func(c *Clock) { c.T0FilmMS = nil }, ReasonKickoffUnmeasured},
		{"durée du match inconnue", func(c *Clock) { c.DurationSeconds = nil }, ReasonDurationUnknown},
		{"moins de deux images", func(c *Clock) { c.FrameCount = 1 }, ReasonWindowIncoherent},
		{"fin avant le début", func(c *Clock) { c.DurationSeconds = ptr64(10) }, ReasonWindowIncoherent},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			clock := Clock{
				OriginMS: ptr64(5000), T0FilmMS: ptr64(20000), DurationSeconds: ptr64(600),
				FrameIntervalMS: 100, FrameCount: 10000,
			}
			c.mutate(&clock)
			if _, got := PlayedWindow(clock); got != c.want {
				t.Errorf("raison = %q, attendu %q", got, c.want)
			}
		})
	}
}
