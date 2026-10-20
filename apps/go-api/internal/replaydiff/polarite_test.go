package replaydiff

import "testing"

// polariteDeTest rend la polarite d'un chemin de couverture, et echoue hors couverture.
func polariteDeTest(t *testing.T, chemin string) Polarite {
	t.Helper()
	p, ok := PolariteDe(chemin)
	if !ok {
		t.Fatalf("%s : hors couverture", chemin)
	}
	return p
}

// Un compteur d'echec de la couverture qui BAISSE est un gain ; qui MONTE, une perte ; les
// mesures de succes gardent la lecture generique (plus = mieux). Mutation : retirer
// l'inversion de `sensSelonPolarite` -> les trois premiers cas rougissent.
func TestComparer_CompteursDEchecLusAlEnvers(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	kUnpub := cle("couverture", "coverage.objectives.unpublished")
	kNoSlot := cle("couverture", "coverage.shots.noSlot")
	kUnnamed := cle("couverture", "coverage.bridge.unnamedLives")
	kNamed := cle("couverture", "coverage.bridge.livesNamed")
	kCaptures := cle("objectifs", "captures")
	a := Empreinte{Mesures: map[string]Mesure{
		kUnpub: num(35), kNoSlot: num(35), kUnnamed: num(0), kNamed: num(40), kCaptures: num(3),
	}}
	b := Empreinte{Mesures: map[string]Mesure{
		kUnpub:    num(0),  // echec qui baisse : GAIN
		kNoSlot:   num(50), // echec qui monte : PERTE
		kUnnamed:  num(2),  // echec qui apparait : PERTE
		kNamed:    num(38), // succes qui baisse : PERTE
		kCaptures: num(4),  // hors couverture, richesse qui monte : GAIN (generique)
	}}
	rap := Comparer(a, b)
	attendu := map[string]string{
		kUnpub: SensGain, kNoSlot: SensPerte, kUnnamed: SensPerte, kNamed: SensPerte, kCaptures: SensGain,
	}
	vus := map[string]string{}
	for _, d := range rap.Differences {
		vus[cle(d.Axe, d.Metrique)] = d.Sens
	}
	for k, sens := range attendu {
		got, ok := vus[k]
		if !ok {
			t.Fatalf("%s : aucun ecart rapporte (vus : %v)", k, vus)
		}
		if got != sens {
			t.Errorf("%s : sens %q, attendu %q", k, got, sens)
		}
	}
	if bil := rap.Bilans["couverture"]; bil.Gains != 1 || bil.Pertes != 3 {
		t.Errorf("bilan couverture : gains=%d pertes=%d, attendu 1 gain / 3 pertes", bil.Gains, bil.Pertes)
	}
}

// Un DENOMINATEUR, une voie de nommage, une feuille de telemetrie et une feuille de couverture
// INCONNUE ne sont ni un gain ni une perte : un CHANGEMENT. Mutation : faire rendre `sens` a la
// branche `default` de `sensSelonPolarite` -> rouge.
func TestComparer_NeutresTelemetrieEtInconnuesSontDesChangements(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	for _, c := range []struct {
		chemin string
		a, b   float64
	}{
		{"coverage.bridge.namedByNextLife", 13, 8},           // voie
		{"coverage.shots.available", 0, 289},                 // denominateur qui monte
		{"coverage.deathsPaths.walk.population", 52, 51},     // denominateur qui baisse (J11)
		{"coverage.tracks.minPoints", 2, 3},                  // telemetrie
		{"coverage.vehicles.ridesFromGap", 4, 0},             // heritee, neutre
		{"coverage.inventeParLeTest.compteur", 3, 7},         // inconnue
		{"bombStats.coverage.inventeParLeTest.compte", 1, 0}, // inconnue du calque d'Assaut
	} {
		k := cle("couverture", c.chemin)
		rap := Comparer(Empreinte{Mesures: map[string]Mesure{k: num(c.a)}},
			Empreinte{Mesures: map[string]Mesure{k: num(c.b)}})
		if len(rap.Differences) != 1 || rap.Differences[0].Sens != SensChangement {
			t.Errorf("%s %v -> %v doit etre un changement, obtenu %+v", c.chemin, c.a, c.b, rap.Differences)
		}
	}
}

// Hors couverture, aucune polarite ne s'applique ; `deathOffsetRunnerUp` est un nombre de voix
// (neutre), pas un echec.
func TestPolariteDe_HorsCouvertureEtVoix(t *testing.T) {
	if _, ok := PolariteDe("unpublished"); ok {
		t.Fatal("hors du chemin coverage., aucune polarite")
	}
	if _, ok := PolariteDe("bombStats.players/n"); ok {
		t.Fatal("bombStats hors de sa couverture n'est pas une mesure de couverture")
	}
	if p := polariteDeTest(t, "coverage.bridge.deathOffsetRunnerUp"); p != PolariteNeutre {
		t.Fatalf("deathOffsetRunnerUp : %v, attendu neutre (un nombre de voix)", p)
	}
}

// TestFermeturesSontDesVoiesPasDesRichesses — LE LOT E2 LES A REDUITES A ZERO, ET C'EST UN GAIN.
//
// `closedByShot` / `closedByRespawn` disent par quelle preuve une FERMETURE a comble le pont :
// des voies (neutres). `closedContested` / `closedRefused` comptent ce que la fermeture a REFUSE :
// des echecs.
//
// MUTATION : deplacer `closedByShot` en succes dans la table -> ROUGE.
func TestFermeturesSontDesVoiesPasDesRichesses(t *testing.T) {
	for _, c := range []string{"coverage.bridge.closedByShot", "coverage.bridge.closedByRespawn"} {
		if p := polariteDeTest(t, c); p != PolariteNeutre {
			t.Errorf("%s : %v, attendu neutre (une voie)", c, p)
		}
	}
	for _, c := range []string{"coverage.bridge.closedContested", "coverage.bridge.closedRefused"} {
		if p := polariteDeTest(t, c); p != PolariteEchec {
			t.Errorf("%s : %v, attendu echec (un refus)", c, p)
		}
	}
}

// TestPolaritesDuRapportJ11 — LES FEUILLES QUE LE G-CORPUS J11 (§4) A TROUVEES MAL LUES.
//
// Chacune est un compteur d'echec que la liste fermee ne connaissait pas : sa baisse sortait en
// perte, sa hausse en gain invisible. Les memes noms de feuille designent ailleurs une richesse
// ou une ventilation (`weaponChanges.dropped`, `placements.dropped`) : la table classe par
// CHEMIN, pas par dernier segment.
//
// MUTATION : retirer `holesUnlocated` des echecs de `coverage.continuousFire.` -> ROUGE (et le
// ratchet de la table rougit aussi : la feuille n'a plus de polarite).
func TestPolaritesDuRapportJ11(t *testing.T) {
	echecs := []string{
		"coverage.continuousFire.holes", "coverage.continuousFire.holesUnlocated",
		"coverage.continuousFire.holesNotClosing", "coverage.stances.eventPacketsUnlocated",
		"coverage.groundWeapons.unknown", "coverage.placements.unknown",
		"coverage.placements.byCause.no_owner", "coverage.stances.dropped",
		"coverage.stances.forgottenBindings", "coverage.birthLoadouts.desync",
		"coverage.birthLoadouts.unconfirmed", "coverage.pickups.originUnknown",
		"coverage.padDating.uncovered", "coverage.vehicles.shotsAmbiguous",
		"coverage.tracks.horsEmprise", "coverage.vehicles.echantillonsHorsEmprise",
		"coverage.vehicles.turretRidesDropped", "coverage.pickups.spawnerByPointKind.unknown",
		"coverage.birthLoadouts.noDisplayable", "coverage.stances.desyncs",
		"coverage.stances.refusedNews", "coverage.bridge.slotCollisions",
		"coverage.placements.byFamilyOrigin.wall/unknown", "coverage.vehicles.unknownChassis.1a2b3c4d",
	}
	for _, c := range echecs {
		if p := polariteDeTest(t, c); p != PolariteEchec {
			t.Errorf("%s : %v, attendu echec", c, p)
		}
	}
	autres := map[string]Polarite{
		"coverage.weaponChanges.dropped":                   PolariteNeutre,
		"coverage.placements.dropped":                      PolariteNeutre,
		"coverage.groundWeapons.dropped":                   PolariteNeutre,
		"coverage.pickups.spawnerByPointKind.grenade":      PolariteNeutre,
		"coverage.placements.byFamilyOrigin.wall/deployed": PolariteNeutre,
		"coverage.placements.byCause.spawn_event":          PolariteSucces,
		"coverage.padDating.dated":                         PolariteSucces,
		"coverage.stances.byKind.slide":                    PolariteNeutre,
	}
	for c, attendu := range autres {
		if p := polariteDeTest(t, c); p != attendu {
			t.Errorf("%s : %v, attendu %v", c, p, attendu)
		}
	}
}
