package powerpos

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// kill fabrique un echantillon d'elimination lisible dans les tests.
func kill(matchID string, kx, ky, kz, vx, vy, vz float64) KillSample {
	return KillSample{
		MatchID: matchID,
		KillerX: kx, KillerY: ky, KillerZ: kz,
		VictimX: vx, VictimY: vy, VictimZ: vz,
	}
}

// celluleEn retrouve la cellule publiee qui contient (x, y).
func celluleEn(t *testing.T, cellules []Cellule, g tactical.Grille, x, y float64) Cellule {
	t.Helper()
	adr, ok := g.Cellule(x, y)
	if !ok {
		t.Fatalf("position (%v, %v) non adressable", x, y)
	}
	for _, c := range cellules {
		if c.Col == adr.Col && c.Lig == adr.Lig {
			return c
		}
	}
	t.Fatalf("aucune cellule alimentee en (%v, %v)", x, y)
	return Cellule{}
}

// TestAjouteKillRepartitDepuisEtDedans : un kill nourrit DEUX cellules, et pas la meme
// grandeur dans chacune.
func TestAjouteKillRepartitDepuisEtDedans(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	a.AjouteKill(kill("m1", 10, 10, 5, 20, 10, 2))

	cellules := a.Cellules()
	if len(cellules) != 2 {
		t.Fatalf("cellules alimentees = %d, attendu 2", len(cellules))
	}
	tueur := celluleEn(t, cellules, g, 10, 10)
	if tueur.KillsDepuis != 1 || tueur.MortsDedans != 0 {
		t.Errorf("cellule du tueur : kills=%d morts=%d, attendu 1/0", tueur.KillsDepuis, tueur.MortsDedans)
	}
	if math.Abs(tueur.PorteeMedianeM-math.Sqrt(100+9)) > 1e-9 {
		t.Errorf("portee mediane = %v, attendu %v", tueur.PorteeMedianeM, math.Sqrt(109))
	}
	if math.Abs(tueur.DeniveleMedianM-3) > 1e-9 {
		t.Errorf("denivele median = %v, attendu 3", tueur.DeniveleMedianM)
	}
	victime := celluleEn(t, cellules, g, 20, 10)
	if victime.KillsDepuis != 0 || victime.MortsDedans != 1 {
		t.Errorf("cellule de la victime : kills=%d morts=%d, attendu 0/1", victime.KillsDepuis, victime.MortsDedans)
	}
	// La portee est une propriete du POSTE DE TIR : la cellule de la victime n'en porte pas.
	if victime.PorteeMedianeM != 0 {
		t.Errorf("portee de la cellule de la victime = %v, attendu 0", victime.PorteeMedianeM)
	}
}

// TestAjouteKillEcarteLesExtremitesNonFinies : un kill ampute est ecarte EN ENTIER et
// compte — jamais moitie pris moitie jete (cf. doc.go).
func TestAjouteKillEcarteLesExtremitesNonFinies(t *testing.T) {
	a := NouvelAccumulateur(tactical.GrilleParDefaut())
	a.AjouteKill(kill("m1", math.NaN(), 0, 0, 5, 5, 0))
	a.AjouteKill(kill("m1", 0, 0, 0, math.Inf(1), 5, 0))
	a.AjouteKill(kill("m1", 0, 0, math.NaN(), 5, 5, 0))

	if a.KillsIgnores() != 3 {
		t.Errorf("kills ignores = %d, attendu 3", a.KillsIgnores())
	}
	if a.NbCellules() != 0 {
		t.Errorf("cellules alimentees = %d, attendu 0", a.NbCellules())
	}
	if a.NbMatchs() != 0 {
		t.Errorf("matchs = %d, attendu 0 (aucun echantillon retenu)", a.NbMatchs())
	}
}

// TestMatchsDistinctsParSource : le plancher se compte en matchs, PAR SOURCE, et le meme
// match ne compte qu'une fois quels que soient ses passages.
func TestMatchsDistinctsParSource(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	for i := 0; i < 5; i++ {
		a.AjouteKill(kill("m1", 3, 3, 0, 9, 9, 0))
	}
	a.AjouteKill(kill("m2", 3.2, 3.1, 0, 9, 9, 0))
	a.AjoutePresence(PresenceSample{MatchID: "m3", X: 3.4, Y: 3.4, DurMS: 100, Gagnant: true})

	c := celluleEn(t, a.Cellules(), g, 3, 3)
	if c.KillsDepuis != 6 {
		t.Errorf("kills depuis = %d, attendu 6", c.KillsDepuis)
	}
	if c.MatchsKills != 2 {
		t.Errorf("matchs de kills = %d, attendu 2 (cinq passages d'un meme match = un match)", c.MatchsKills)
	}
	if c.MatchsPresence != 1 {
		t.Errorf("matchs de presence = %d, attendu 1", c.MatchsPresence)
	}
	if c.MatchsDistincts != 3 {
		t.Errorf("matchs distincts = %d, attendu 3", c.MatchsDistincts)
	}
	if a.NbMatchs() != 3 {
		t.Errorf("matchs du corpus = %d, attendu 3", a.NbMatchs())
	}
}

// TestAjoutePresenceSepareLesIssues : l'occupation se range du cote de l'issue du MATCH.
func TestAjoutePresenceSepareLesIssues(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	// Les trois positions tombent dans la MEME cellule : [-4,0 ; -3,5) x [7,0 ; 7,5).
	a.AjoutePresence(PresenceSample{MatchID: "m1", Team: 0, Gagnant: true, X: -4, Y: 7, DurMS: 250})
	a.AjoutePresence(PresenceSample{MatchID: "m1", Team: 1, Gagnant: false, X: -3.9, Y: 7.2, DurMS: 100})
	a.AjoutePresence(PresenceSample{MatchID: "m2", Team: 1, Gagnant: false, X: -3.8, Y: 7.4, DurMS: 50})

	c := celluleEn(t, a.Cellules(), g, -4, 7)
	if c.OccupationGagnantsMS != 250 {
		t.Errorf("occupation gagnants = %v, attendu 250", c.OccupationGagnantsMS)
	}
	if c.OccupationPerdantsMS != 150 {
		t.Errorf("occupation perdants = %v, attendu 150", c.OccupationPerdantsMS)
	}
	if c.TotalOccupationMS() != 400 {
		t.Errorf("occupation totale = %v, attendu 400", c.TotalOccupationMS())
	}
}

// TestAjoutePresenceEcarteDureeEtPositionInvalides : une duree nulle n'occupe rien.
func TestAjoutePresenceEcarteDureeEtPositionInvalides(t *testing.T) {
	a := NouvelAccumulateur(tactical.GrilleParDefaut())
	a.AjoutePresence(PresenceSample{MatchID: "m1", X: 1, Y: 1, DurMS: 0})
	a.AjoutePresence(PresenceSample{MatchID: "m1", X: 1, Y: 1, DurMS: -5})
	a.AjoutePresence(PresenceSample{MatchID: "m1", X: math.NaN(), Y: 1, DurMS: 10})
	a.AjoutePresence(PresenceSample{MatchID: "m1", X: 1, Y: 1, DurMS: math.Inf(1)})

	if a.PresencesIgnorees() != 4 {
		t.Errorf("presences ignorees = %d, attendu 4", a.PresencesIgnorees())
	}
	if a.NbCellules() != 0 {
		t.Errorf("cellules alimentees = %d, attendu 0", a.NbCellules())
	}
}

// TestCellulesTrieesEtAncreesSurLOrigine : l'ordre de sortie est stable, et l'adresse d'une
// cellule ne depend pas des bornes de la lecture (cf. tactical/doc.go, ANCRAGE).
func TestCellulesTrieesEtAncreesSurLOrigine(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	a.AjouteKill(kill("m1", 12, -3, 0, -7, 8, 0))
	a.AjouteKill(kill("m1", -7.4, 8.2, 0, 12.1, -3.1, 0))

	cellules := a.Cellules()
	for i := 1; i < len(cellules); i++ {
		avant, apres := cellules[i-1], cellules[i]
		if avant.Col > apres.Col || (avant.Col == apres.Col && avant.Lig > apres.Lig) {
			t.Fatalf("cellules non triees en %d : %+v puis %+v", i, avant, apres)
		}
	}
	c := celluleEn(t, cellules, g, 12, -3)
	if c.Col != 24 || c.Lig != -6 {
		t.Errorf("adresse de (12, -3) = (%d, %d), attendu (24, -6) au pas de 0,5 m", c.Col, c.Lig)
	}
	if math.Abs(c.CentreX-12.25) > 1e-9 || math.Abs(c.CentreY-(-2.75)) > 1e-9 {
		t.Errorf("centre = (%v, %v), attendu (12.25, -2.75)", c.CentreX, c.CentreY)
	}
}
