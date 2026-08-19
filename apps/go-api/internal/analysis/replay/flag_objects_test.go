package replay

// flag_objects_test.go — LES DEUX CORRECTIONS QUE L'OBJET DRAPEAU APPORTE AUX PORTAGES, sans film.
//
// Meme forme que `flag_carries_test.go`, et pour la meme raison : la verite terrain vit dans
// l'instrument sous garde (`drapeau_objet_controle_test.go`), qui lit de vrais films et publie
// ses denominateurs au plan ; ici on fige les REGLES et on les fait tomber si elles disparaissent.
//
// LES DEUX QUI COMPTENT :
//
//	le LACHER VOLONTAIRE SE DATE — un portage que rien ne fermait se ferme a l'instant ou
//	  l'objet reapparait AUX PIEDS de son porteur ;
//	le LACHER CHANGE DE PLACE — `dropped` passe de la derniere position du PORTEUR au dernier
//	  point de la piste LIBRE, la ou l'objet repose.
//
// CHACUNE EST TESTEE AVEC SES TEMOINS NEGATIFS, et il en faut DEUX : une vie libre nee LOIN du
// porteur, et une vie libre nee A UN SOCLE. La seconde n'est pas theorique — c'est la seule
// population que le controle 3 valide qui compte ici, et un drapeau qui rentre a sa base pendant
// qu'un porteur meurt juste a cote suffirait a confondre les deux. Une regle qui ne refuse rien
// ne prouve rien.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// flagTestLife fabrique une vie libre a l'echelle du contexte de test (100 ms/frame, origine 0) :
// `frame` est la frame de sa creation, les points suivent toutes les 5 frames.
func flagTestLife(frame int, pts ...[2]float32) flagFreeLife {
	l := flagFreeLife{ID: 0x2a392328, Key: filmdec.EquipmentLifeKey{Slot: 7}}
	for i, p := range pts {
		at := uint64(frame+i*5) * 100_000
		l.Pts = append(l.Pts, flagFreeSample{TUS: at, X: p[0], Y: p[1]})
	}
	l.T0US, l.T1US = l.Pts[0].TUS, l.Pts[len(l.Pts)-1].TUS
	return l
}

// flagTestSpawns : les deux socles d'equipe des tests.
func flagTestSpawns() []FlagSpawn {
	return []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}}
}

// flagTestOpenScan monte un portage que RIEN ne ferme : une prise, aucune capture, aucune mort.
func flagTestOpenScan(free []flagFreeLife) FlagCarryScan {
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: map[int]string{12: "1"},
		Spawns:   flagTestSpawns(),
		Free:     free,
	}
}

// TestUneVieLibreDateLeLacherVolontaire — LA mesure du lot : un portage `carried_open` devient
// `carried`, ferme a l'instant ou l'objet reapparait aux pieds de son porteur.
func TestUneVieLibreDateLeLacherVolontaire(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	ctx := flagTestCtx(tracks, nil, 100)

	// TEMOIN — sans vie libre, le portage court jusqu'a la fin de l'axe et se publie OUVERT.
	got, cov := buildFlagCarries(flagTestOpenScan(nil), ctx)
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarriedOpen})
	if cov.Open != 1 || cov.Closed != 0 || cov.ClosedByObject != 0 {
		t.Fatalf("temoin : couverture %+v, attendu 1 ouvert / 0 ferme / 0 ferme par l'objet", *cov)
	}

	// LA REGLE — l'objet reapparait a la frame 30, a 0,7 m du porteur : c'est le lacher.
	free := []flagFreeLife{flagTestLife(30, [2]float32{50.5, 50.5}, [2]float32{51, 51})}
	got, cov = buildFlagCarries(flagTestOpenScan(free), ctx)
	f = flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped})
	if f.Spans[1].T1 != 30 {
		t.Errorf("portage ferme a la frame %d, attendu 30 (la naissance de la vie libre)",
			f.Spans[1].T1)
	}
	if cov.ClosedByObject != 1 || cov.Closed != 1 || cov.Open != 0 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 1 ferme par l'objet, 0 ouvert, invariant tenu", *cov)
	}
}

// TestUneVieLibreLoinDuPorteurNeFermeRien — PREMIER TEMOIN NEGATIF : le meme instant, mais a
// l'autre bout de la carte, ne ferme rien.
//
// SANS CE REFUS, LA REGLE NE MESURERAIT QUE LE TEMPS. En CTF il y a DEUX drapeaux, et l'autre vit
// sa vie pendant qu'on porte le sien.
func TestUneVieLibreLoinDuPorteurNeFermeRien(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	free := []flagFreeLife{flagTestLife(30, [2]float32{70, 70})}
	got, cov := buildFlagCarries(flagTestOpenScan(free), flagTestCtx(tracks, nil, 100))
	assertFlagStates(t, flagOfTeam(t, got, 0), []string{FlagStateHome, FlagStateCarriedOpen})
	if cov.ClosedByObject != 0 || cov.Open != 1 {
		t.Errorf("couverture %+v : une vie libre a 28 m du porteur ne ferme aucun portage", *cov)
	}
}

// TestUneVieLibreNeeAUnSocleNeFermeRien — SECOND TEMOIN NEGATIF, et le plus important : une
// naissance AU SOCLE est ecartee MEME quand le porteur est juste a cote.
//
// C'EST LA CONDITION DE L'ARBITRAGE DU 2026-08-18. Les corrections ne se livrent que parce
// qu'elles ne touchent que la sous-population validee par le controle 3 — les vies nees AUX PIEDS
// D'UN PORTEUR. Un drapeau qui rentre a sa base pendant qu'un porteur agonise devant ce socle
// n'est pas un lacher, et sans ce refus la distance seule les confondrait.
func TestUneVieLibreNeeAUnSocleNeFermeRien(t *testing.T) {
	// Le porteur est A 0,5 m du socle de l'equipe 1 : seule la naissance au socle les separe.
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 100.5, 100)}
	free := []flagFreeLife{flagTestLife(30, [2]float32{100, 100})}
	scan := flagTestOpenScan(free)
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.ClosedByObject != 0 || cov.Open != 1 {
		t.Errorf("couverture %+v : une vie libre nee AU SOCLE ne ferme aucun portage, meme a "+
			"0,5 m du porteur", *cov)
	}
	assertFlagStates(t, flagOfTeam(t, got, 1), []string{FlagStateHome, FlagStateCarriedOpen})
}

// TestLeLacherPrendLaPositionDeLaPisteLibre — `dropped` passe de la derniere position du PORTEUR
// au dernier point de la piste LIBRE. Un drapeau tombe rebondit ; le porteur meurt rarement la ou
// l'objet se pose.
func TestLeLacherPrendLaPositionDeLaPisteLibre(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 95, 95)}
	deaths := []Death{{XUID: 1, TimeMS: 4000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: map[int]string{12: "1"},
		Spawns:   flagTestSpawns(),
	}
	ctx := flagTestCtx(tracks, deaths, 100)

	// TEMOIN — sans piste libre, le lacher reste a la position du porteur.
	got, cov := buildFlagCarries(scan, ctx)
	f := flagOfTeam(t, got, 1)
	if f.Spans[2].X != 95 || f.Spans[2].Y != 95 || cov.DropsRepositioned != 0 {
		t.Fatalf("temoin : lacher en (%v,%v), %d repositionnes — attendu (95,95) et 0",
			f.Spans[2].X, f.Spans[2].Y, cov.DropsRepositioned)
	}

	// LA REGLE — l'objet nait a la mort (frame 40) puis roule jusqu'a (97, 93).
	scan.Free = []flagFreeLife{flagTestLife(40, [2]float32{95.2, 95.2}, [2]float32{97, 93})}
	got, cov = buildFlagCarries(scan, ctx)
	f = flagOfTeam(t, got, 1)
	if f.Spans[2].State != FlagStateDropped {
		t.Fatalf("etat %q, attendu `dropped`", f.Spans[2].State)
	}
	if f.Spans[2].X != 97 || f.Spans[2].Y != 93 {
		t.Errorf("lacher en (%v,%v), attendu le dernier point de la piste libre (97,93)",
			f.Spans[2].X, f.Spans[2].Y)
	}
	if cov.DropsRepositioned != 1 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 1 lacher repositionne et l'invariant tenu", *cov)
	}
}

// TestFlagFreeLivesApparieCreationEtPiste — une vie libre est un record de CREATION plus la piste
// delta de la MEME vie, et rien d'autre : une creation d'ARME au sol n'en fait pas partie.
func TestFlagFreeLivesApparieCreationEtPiste(t *testing.T) {
	const drapeau = uint32(0x2a392328)
	arme := gwTestFamily(t, 0)
	scan := WorldObjectScan{
		Scanned: true,
		Creations: []filmdec.EquipmentCreation{
			gwTestCreation(7, 0, 1_000_000, drapeau, 10, 10),
			gwTestCreation(8, 0, 2_000_000, arme, 20, 20),
		},
		Tracks: []filmdec.ProjectileTrack{{
			Slot: 7, Gen: 0, Pts: []filmdec.ProjectileSample{
				{TimestampUS: 1_000_000, X: 10, Y: 10},
				{TimestampUS: 1_500_000, X: 12, Y: 11},
			},
		}},
	}
	lives := flagFreeLives(scan, map[uint32]Label{drapeau: {En: "Flag", Fr: "Drapeau"}})
	if len(lives) != 1 {
		t.Fatalf("%d vies libres, attendu 1 (l'arme au sol n'en est pas une)", len(lives))
	}
	l := lives[0]
	if l.T0US != 1_000_000 || l.T1US != 1_500_000 || len(l.Pts) != 2 {
		t.Fatalf("vie [%d,%d] avec %d points, attendu [1000000,1500000] et 2 points",
			l.T0US, l.T1US, len(l.Pts))
	}
	if x, y := l.First(); x != 10 || y != 10 {
		t.Errorf("creation en (%v,%v), attendu (10,10)", x, y)
	}
	if x, y := l.Last(); x != 12 || y != 11 {
		t.Errorf("repos en (%v,%v), attendu le dernier point replique (12,11)", x, y)
	}
	if n := len(flagFreeLives(scan, nil)); n != 0 {
		t.Errorf("%d vies libres sans table d'identite, attendu 0 — un titre qui ne declare "+
			"aucun drapeau n'en publie aucun", n)
	}
}
