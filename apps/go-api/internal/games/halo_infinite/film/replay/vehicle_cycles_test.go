package replay

// vehicle_cycles_test.go — LE CYCLE DE REAPPARITION DES VEHICULES, sur tampon synthetique.
//
// AUCUN FILM N EST OUVERT ICI, et c est le point : la regle se juge sur des vies FABRIQUEES, ou
// chaque ecart et chaque manque est voulu. Un test de corpus dirait « ca marche sur ce film » ;
// celui-ci dit CE QUE LA REGLE FAIT.

import "testing"

// vcStep est le pas d horloge des tampons : 1 000 000 us = 1 s par frame. Les durees se lisent
// donc directement en frames.
const vcStep = uint64(1_000_000)

// vcVie fabrique une vie de vehicule nee en (x, y) a la frame t0. `tEnd` negatif : le film
// n ecrit pas sa mort, et la vie porte alors une fin NON datee.
func vcVie(x, y float32, t0, tEnd int, famille string) VehicleTrack {
	tr := VehicleTrack{
		T0: t0, T1: t0 + 1, Family: famille,
		Spawn: &VehicleSpawn{X: x, Y: y},
		End:   VehicleEndUnknown,
	}
	if tEnd >= 0 {
		fin := tEnd
		tr.End, tr.TEnd = VehicleEndDestroyed, &fin
	}
	return tr
}

func TestBuildVehicleCyclesEtablitUnCycleSurDeuxEcarts(t *testing.T) {
	// Trois vies au MEME endroit : deux ecarts de 30 s, tous deux mesurables.
	tracks := []VehicleTrack{
		vcVie(10, 10, 0, 100, "warthog"),
		vcVie(10.5, 10.5, 130, 200, "warthog"),
		vcVie(9.8, 10.2, 230, -1, "warthog"),
	}
	var cov VehicleCoverage
	got := buildVehicleCycles(tracks, vcStep, &cov)
	if len(got) != 1 {
		t.Fatalf("cycles publies = %d, attendu 1 : %+v", len(got), got)
	}
	c := got[0]
	if c.Gaps != 2 {
		t.Errorf("ecarts = %d, attendu 2", c.Gaps)
	}
	if c.MedianS != 30 {
		t.Errorf("mediane = %v s, attendu 30", c.MedianS)
	}
	if c.Missing != 0 {
		t.Errorf("manques = %d, attendu 0 : les deux occasions ont rendu un ecart", c.Missing)
	}
	if c.Family != "warthog" {
		t.Errorf("famille = %q, attendu warthog", c.Family)
	}
	if cov.CycleLocations != 1 || cov.Cycles != 1 || cov.CycleGaps != 2 || cov.CycleMissing != 0 {
		t.Errorf("couverture = emplacements %d, cycles %d, ecarts %d, manques %d — attendu 1/1/2/0",
			cov.CycleLocations, cov.Cycles, cov.CycleGaps, cov.CycleMissing)
	}
}

// TestBuildVehicleCyclesUnSeulEcartNePublieRien : la regle d etablissement est celle de
// `PadCycle`, et elle est reprise TELLE QUELLE. Un ecart unique n a pas d ecart-type, donc rien
// ne dit qu il se repete — le publier rendrait un « 30 s » qui se lirait comme une mesure.
func TestBuildVehicleCyclesUnSeulEcartNePublieRien(t *testing.T) {
	tracks := []VehicleTrack{
		vcVie(10, 10, 0, 100, "ghost"),
		vcVie(10, 10, 130, -1, "ghost"),
	}
	var cov VehicleCoverage
	if got := buildVehicleCycles(tracks, vcStep, &cov); got != nil {
		t.Fatalf("cycles publies = %+v, attendu aucun (un seul ecart)", got)
	}
	if cov.CycleGaps != 1 {
		t.Errorf("ecarts MESURES = %d, attendu 1 : l ecart existe, c est le VERDICT qui manque",
			cov.CycleGaps)
	}
}

// TestBuildVehicleCyclesFinNonDateeEstUnManque : le cas qui a bloque tout le calque jusqu au lot
// 5.1.7-b. Une vie dont le film n ecrit PAS la mort ne rend aucun ecart — et l occasion se
// COMPTE, elle ne se devine pas.
func TestBuildVehicleCyclesFinNonDateeEstUnManque(t *testing.T) {
	tracks := []VehicleTrack{
		vcVie(10, 10, 0, -1, "warthog"),
		vcVie(10, 10, 130, -1, "warthog"),
		vcVie(10, 10, 230, -1, "warthog"),
	}
	var cov VehicleCoverage
	if got := buildVehicleCycles(tracks, vcStep, &cov); got != nil {
		t.Fatalf("cycles publies = %+v, attendu aucun", got)
	}
	if cov.CycleGaps != 0 || cov.CycleMissing != 2 {
		t.Errorf("ecarts %d / manques %d, attendu 0 / 2", cov.CycleGaps, cov.CycleMissing)
	}
}

// TestBuildVehicleCyclesFinNonDESTRUCTRICEEstUnManque : `film_end` n est PAS une mort. Le film
// est simplement arrete ; compter cet instant reviendrait a mesurer la fin du film.
func TestBuildVehicleCyclesFinNonDestructriceEstUnManque(t *testing.T) {
	a := vcVie(10, 10, 0, 100, "warthog")
	a.End = VehicleEndFilmEnd // la date reste, la CAUSE change
	tracks := []VehicleTrack{a, vcVie(10, 10, 130, 200, "warthog"), vcVie(10, 10, 230, -1, "warthog")}
	var cov VehicleCoverage
	if got := buildVehicleCycles(tracks, vcStep, &cov); got != nil {
		t.Fatalf("cycles publies = %+v, attendu aucun (un seul ecart mesurable)", got)
	}
	if cov.CycleGaps != 1 || cov.CycleMissing != 1 {
		t.Errorf("ecarts %d / manques %d, attendu 1 / 1", cov.CycleGaps, cov.CycleMissing)
	}
}

// TestBuildVehicleCyclesSepareDeuxEmplacements : 2 m agglomere, au-dela NON. Deux socles
// distants de 10 m sont deux emplacements, et melanger leurs ecarts fabriquerait un cycle qui
// n existe nulle part.
func TestBuildVehicleCyclesSepareDeuxEmplacements(t *testing.T) {
	tracks := []VehicleTrack{
		vcVie(0, 0, 0, 100, "warthog"),
		vcVie(0, 0, 130, 200, "warthog"),
		vcVie(0, 0, 230, -1, "warthog"),
		vcVie(50, 50, 10, 110, "ghost"),
		vcVie(50, 50, 200, -1, "ghost"),
	}
	var cov VehicleCoverage
	got := buildVehicleCycles(tracks, vcStep, &cov)
	if cov.CycleLocations != 2 {
		t.Fatalf("emplacements = %d, attendu 2", cov.CycleLocations)
	}
	if len(got) != 1 || got[0].Family != "warthog" {
		t.Fatalf("cycles = %+v, attendu le seul emplacement warthog", got)
	}
}

// TestBuildVehicleCyclesIgnoreLesViesSansNaissance : une vie sans position de naissance n a pas
// d emplacement. Lui en inventer un ferait naitre des ecarts entre des endroits differents.
func TestBuildVehicleCyclesIgnoreLesViesSansNaissance(t *testing.T) {
	sans := vcVie(10, 10, 130, 200, "warthog")
	sans.Spawn = nil
	tracks := []VehicleTrack{vcVie(10, 10, 0, 100, "warthog"), sans, vcVie(10, 10, 230, -1, "warthog")}
	var cov VehicleCoverage
	got := buildVehicleCycles(tracks, vcStep, &cov)
	if got != nil {
		t.Fatalf("cycles = %+v, attendu aucun : une seule occasion subsiste", got)
	}
	if cov.CycleGaps != 1 {
		t.Errorf("ecarts = %d, attendu 1 (0 -> 230, la vie sans naissance ne compte pas)", cov.CycleGaps)
	}
}

// TestBuildVehicleCyclesRefuseLesEcartsDisperses : le juge est `gwPadsCycleFromGaps`, avec son
// ecart-type sous 20 % de la mediane. Des ecarts de 5 s et 200 s ne sont pas un cycle.
func TestBuildVehicleCyclesRefuseLesEcartsDisperses(t *testing.T) {
	tracks := []VehicleTrack{
		vcVie(10, 10, 0, 100, "warthog"),
		vcVie(10, 10, 105, 200, "warthog"),
		vcVie(10, 10, 400, -1, "warthog"),
	}
	var cov VehicleCoverage
	if got := buildVehicleCycles(tracks, vcStep, &cov); got != nil {
		t.Fatalf("cycles = %+v, attendu aucun : 5 s contre 200 s", got)
	}
	if cov.CycleGaps != 2 {
		t.Errorf("ecarts = %d, attendu 2 : ils sont MESURES, c est le verdict qui les refuse",
			cov.CycleGaps)
	}
}

// TestBuildVehicleCyclesFamilleInconnueGardeSonCycle : un emplacement dont aucun chassis n est
// nomme publie son cycle SANS famille — il n emprunte pas le nom d un voisin. C est la meme
// regle que le chassis affiche en hexadecimal, et la meme lecon que le correctif du 2026-09-19
// sur `vehicleFamilyIsRideable`.
func TestBuildVehicleCyclesFamilleInconnueGardeSonCycle(t *testing.T) {
	tracks := []VehicleTrack{
		vcVie(10, 10, 0, 100, ""),
		vcVie(10, 10, 130, 200, ""),
		vcVie(10, 10, 230, -1, ""),
	}
	got := buildVehicleCycles(tracks, vcStep, nil)
	if len(got) != 1 {
		t.Fatalf("cycles = %+v, attendu 1", got)
	}
	if got[0].Family != "" {
		t.Errorf("famille = %q, attendue VIDE : la cle est omise, jamais devinee", got[0].Family)
	}
}
