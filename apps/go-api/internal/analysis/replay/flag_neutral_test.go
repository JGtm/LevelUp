package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_neutral_test.go — LA VARIANTE « DRAPEAU NEUTRE » SE RECONNAIT, sans film.
//
// La regle qu'ils figent : le mode n'est pas dans le film, c'est l'OBJET qui tranche — le socle
// ou il RENAIT. Le defaut est la variante ordinaire, et il faut un signal franc pour en sortir.
// Verite terrain sur films reels : `ctf_retour_zone_research_test.go`, corpus neutre.

// flagNeutralScan fabrique un balayage avec les TROIS socles et des naissances a volonte.
func flagNeutralScan(auNeutre, auxEquipes int) FlagCarryScan {
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Spawns: []FlagSpawn{
			{Team: 0, X: 0, Y: 0},
			{Team: 1, X: 100, Y: 100},
			{Team: TeamNeutral, X: 50, Y: 50},
		},
	}
	// LES NAISSANCES DE REFERENCE TOMBENT TOT (pas de 0,1 s), avant tout portage : elles servent a
	// COMPTER, pas a declencher une rentree. Une naissance posee sur l'instant d'un lacher en
	// declencherait une, et le test mesurerait alors autre chose que ce qu'il annonce.
	for i := 0; i < auNeutre; i++ {
		scan.Free = append(scan.Free, flagFreeLifeAt(uint64(i+1)*100_000, 50, 50))
	}
	for i := 0; i < auxEquipes; i++ {
		scan.Free = append(scan.Free, flagFreeLifeAt(uint64(i+50)*1_000_000, 0, 0))
	}
	return scan
}

// flagFreeLifeAt fabrique une vie libre reduite a sa naissance.
func flagFreeLifeAt(tus uint64, x, y float32) flagFreeLife {
	return flagFreeLife{T0US: tus, T1US: tus, Pts: []flagFreeSample{{TUS: tus, X: x, Y: y}}}
}

// TestFlagVarianteOrdinaireEcarteLeSocleNeutre — le defaut : deux drapeaux d'equipe, centre ecarte.
func TestFlagVarianteOrdinaireEcarteLeSocleNeutre(t *testing.T) {
	choix := flagChooseSpawns(flagNeutralScan(0, 6))
	if choix.Neutral {
		t.Fatalf("variante neutre reconnue a tort : %+v", choix)
	}
	if len(choix.Spawns) != 2 {
		t.Fatalf("%d socles retenus, attendu 2 (les deux socles d'equipe)", len(choix.Spawns))
	}
	for _, s := range choix.Spawns {
		if s.Team == TeamNeutral {
			t.Fatal("le socle neutre est retenu sur une partie ordinaire")
		}
	}
}

// TestFlagVarianteNeutreNeGardeQueLeCentre — l'objet renait au centre : un seul drapeau.
func TestFlagVarianteNeutreNeGardeQueLeCentre(t *testing.T) {
	choix := flagChooseSpawns(flagNeutralScan(6, 0))
	if !choix.Neutral {
		t.Fatalf("variante neutre NON reconnue : %+v", choix)
	}
	if len(choix.Spawns) != 1 || choix.Spawns[0].Team != TeamNeutral {
		t.Fatalf("socles retenus %+v, attendu le seul socle neutre", choix.Spawns)
	}
	if choix.NeutralBirths != 6 || choix.TeamBirths != 0 {
		t.Fatalf("comptes %+v : 6 naissances au centre, 0 aux socles d'equipe", choix)
	}
}

// TestFlagVarianteNeutreExigeUnSignalFRANC — sous le seuil, ou a egalite, le defaut l'emporte.
//
// C'EST LA MOITIE QUI COMPTE : une naissance egaree au centre ne doit PAS faire disparaitre les
// deux drapeaux d'une partie ordinaire. On se trompe du cote qui ne casse rien.
func TestFlagVarianteNeutreExigeUnSignalFranc(t *testing.T) {
	for _, c := range []struct{ neutre, equipes int }{
		{2, 0}, // sous le seuil de naissances
		{4, 4}, // pas strictement majoritaire
		{0, 0}, // film muet : aucune naissance lue
	} {
		choix := flagChooseSpawns(flagNeutralScan(c.neutre, c.equipes))
		if choix.Neutral {
			t.Fatalf("neutre=%d equipes=%d : variante neutre reconnue a tort (%+v)",
				c.neutre, c.equipes, choix)
		}
	}
}

// TestFlagVarianteNeutrePublieUnSeulDrapeau — le calque complet, de bout en bout : un vol, une
// mort, et le drapeau NEUTRE qui rentre chez lui.
func TestFlagVarianteNeutrePublieUnSeulDrapeau(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	deaths := []Death{{XUID: 1, TimeMS: 4000}}
	scan := flagNeutralScan(4, 0)
	scan.Events = []objectiveevents.NamedEvent{
		{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
	}
	scan.Identity = objectiveevents.FlatRoundIdentity(map[int]string{12: "1"})
	// Une naissance TARDIVE au centre, pendant que le drapeau git : c'est sa rentree.
	scan.Free = append(scan.Free, flagFreeLifeAt(7_000_000, 50, 50))
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if !cov.NeutralFlag || cov.Spawns != 1 {
		t.Fatalf("couverture %+v : variante neutre et UN socle attendus", *cov)
	}
	if len(got) != 1 {
		t.Fatalf("%d drapeaux publies, attendu 1", len(got))
	}
	if got[0].Team != TeamNeutral {
		t.Fatalf("equipe du drapeau %d, attendu %d (neutre)", got[0].Team, TeamNeutral)
	}
	assertFlagStates(t, got[0],
		[]string{FlagStateHome, FlagStateCarried, FlagStateDropped, FlagStateHome})
}

// TestFlagNaissancesLaDISTANCEDecideDuDENOMINATEUR — LA RESERVE DE LA REVUE 6.R, INSTRUITE.
//
// CE QUE LE RELECTEUR A NOTE. Le lot 6.13 a resserre [flagBirthsNear] du rayon du LACHER (1,5 m)
// a [flagHomeExactDist] (0,10 m). Les tests ci-dessus posent leurs naissances EXACTEMENT sur les
// socles : ils passent avec l'ancien seuil comme avec le nouveau, et ne disent donc rien du
// resserrement. Ce test-ci fait varier la SEULE distance.
//
// LA MESURE SUR FILM (revue 6.R, parc de 12 films CTF cuits hors ligne) : `4ecdf3e7`
// (High Ground, la seule partie a drapeau neutre du parc) reste classe NEUTRE apres 6.13, avec
// CINQ naissances a 0,10 m au plus du socle neutre et ZERO aux socles d'equipe — deux de marge
// au-dessus de [flagNeutralMinBirths], et un denominateur adverse vide.
//
// LES DEUX CAS : une naissance a 0,05 m est un drapeau QUI RENTRE (le catalogue est juste a
// 0,006 m pres sur le parc) ; une naissance a 0,50 m est un LACHER a portee du support, et elle
// ne dit rien de la variante — c'est exactement ce que le resserrement retire du denominateur.
func TestFlagNaissancesLaDISTANCEDecideDuDenominateur(t *testing.T) {
	for _, c := range []struct {
		nom    string
		ecart  float32
		neutre bool
	}{
		{"a 0,05 m du socle : le drapeau rentre", 0.05, true},
		{"a 0,50 m du socle : un lacher a portee du support", 0.50, false},
	} {
		scan := flagNeutralScan(0, 0)
		for i := 0; i < 6; i++ {
			scan.Free = append(scan.Free,
				flagFreeLifeAt(uint64(i+1)*100_000, 50+c.ecart, 50))
		}
		choix := flagChooseSpawns(scan)
		if choix.Neutral != c.neutre {
			t.Errorf("%s : neutre=%v (attendu %v), comptes neutre=%d equipes=%d",
				c.nom, choix.Neutral, c.neutre, choix.NeutralBirths, choix.TeamBirths)
		}
	}
}
