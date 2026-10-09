package replayverite

import "testing"

// TestSaut_AuDelaDuPlafondSaufTranslocationEtPorte : un point a 20 m en une image (200 m/s) est un
// saut ; le meme saut a une translocation publiee, ou vers une porte de carte mesuree, est exempte.
func TestSaut_AuDelaDuPlafondSaufTranslocationEtPorte(t *testing.T) {
	d := documentJuste()
	d.Tracks[0].Points = []Point{{T: 10, X: 0, Y: 0}, {T: 11, X: 20, Y: 0}, {T: 12, X: 20.5, Y: 0}}
	b := Noter(d, faitsJustes(), nil)
	exigerViolations(t, b, ViolSaut, 1)

	d.Translocations = []Translocation{{T: 12}}
	b = Noter(d, faitsJustes(), nil)
	exigerViolations(t, b, ViolSaut, 0)
	if b.Violations[ViolSaut].Exemptees != 1 {
		t.Errorf("exemptees = %d, veut 1", b.Violations[ViolSaut].Exemptees)
	}

	d.Translocations = nil
	porte := portesDeCarteMesurees[len(portesDeCarteMesurees)-1]
	d.Tracks[0].Points[1] = Point{T: 11, X: porte.centre[0] + 1, Y: porte.centre[1], Z: ptr(porte.centre[2])}
	d.Tracks[0].Points[2] = Point{T: 12, X: porte.centre[0] + 1.5, Y: porte.centre[1], Z: ptr(porte.centre[2])}
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolSaut, 1) // carte inconnue : pas d'exemption
	faits := faitsJustes()
	faits.MapID = porte.mapID
	exigerViolations(t, Noter(d, faits, nil), ViolSaut, 0)
}

// TestHorsEmprise_ObjetsEtVehiculesAuDelaDeLaMarge : un objet pose a plus de la marge de
// l'enveloppe est hors emprise ; a l'interieur de la marge, non ; un tir, jamais.
func TestHorsEmprise_ObjetsEtVehiculesAuDelaDeLaMarge(t *testing.T) {
	d := documentJuste()
	d.GroundWeapons = []Objet{{X: 100 + margeEmpriseObjetsM - 1, Y: 50}, {X: 100 + margeEmpriseObjetsM + 1, Y: 50}}
	d.EquipmentPlacements = []Objet{{X: 50, Y: -margeEmpriseObjetsM - 5}}
	d.WeaponPads = []Objet{{X: 50, Y: 50}}
	d.Vehicles = []Vehicule{{Slot: 700, Samples: []Echantillon{{T: 5, X: -500, Y: 0}, {T: 6, X: 5, Y: 5}}}}
	d.Shots = append(d.Shots, Action{T: 60, Slot: ptr(512)})
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolHorsEmprise, 3)
}

// TestHorsVie_ActionSurUnSlotSansVie : une action hors de toute vie de son slot, au-dela de la
// tolerance, est un faux ; dans la tolerance, non.
func TestHorsVie_ActionSurUnSlotSansVie(t *testing.T) {
	d := documentJuste()
	d.Shots = append(d.Shots, Action{T: 100 + toleranceVieImages, Slot: ptr(512)}, Action{T: 130, Slot: ptr(512)})
	d.Grenades = []Action{{T: 5, Slot: ptr(999)}}
	d.Pickups = []Action{{T: 120, Slot: ptr(513)}}
	d.WeaponChanges = []Action{{T: 301 + toleranceVieImages, Slot: ptr(514)}}
	d.Abilities = []Action{{T: 225, Slot: ptr(513)}}
	d.EquipmentChanges = []Action{{T: 10}}                            // sans slot : non jugeable, ignore
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolHorsVie, 4) // tir @130, grenade slot 999, changement @306, capacite @225
}

// TestHorsVie_UnTrajetDuSlotFaitPartieDeSaVie : la piste du slot 512 s'arrete a l'image 100, il
// monte a bord sur [110, 140] ; son tir @130 est dans sa vie, son tir @200 non. Le trajet d'un
// AUTRE slot ne couvre pas le tir du 512.
func TestHorsVie_UnTrajetDuSlotFaitPartieDeSaVie(t *testing.T) {
	d := documentJuste()
	d.Vehicles = []Vehicule{
		{Slot: 700, Rides: []Trajet{{T0: 110, T1: 140, Slot: 512, XUID: "111"}}},
		{Slot: 701, Rides: []Trajet{{T0: 190, T1: 220, Slot: 515, XUID: "222"}}},
	}
	d.Shots = append(d.Shots, Action{T: 130, Slot: ptr(512)}, Action{T: 200, Slot: ptr(512)})
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolHorsVie, 1) // tir @200 seul
}

// TestDeuxCorps_VieQuiRecouvreUneAutre : deux vies d'un meme xuid qui se recouvrent.
func TestDeuxCorps_VieQuiRecouvreUneAutre(t *testing.T) {
	d := documentJuste()
	d.Tracks = append(d.Tracks, piste(520, "111", 90, 120))
	b := Noter(d, faitsJustes(), nil)
	exigerViolations(t, b, ViolDeuxCorps, 1) // [90,120] recouvre [0,100], pas [150,300]
}

// TestAbsent_PresenceEtVieDuPorteur : un ramassage hors presence, une action d'objectif hors
// presence, un portage par un joueur mort sont des faux ; une action d'objectif d'un joueur mort
// mais present ne l'est pas.
func TestAbsent_PresenceEtVieDuPorteur(t *testing.T) {
	d := documentJuste()
	d.Roster[1].Presence = []Presence{{From: 0, To: 250, ToMax: ptr(260)}}
	d.Pickups = []Action{{T: 255, XUID: "222"}, {T: 270, XUID: "222"}}
	d.Objectives = []ActionObjectif{{T: 125, XUID: "111", Stat: "kills"}, {T: 280, XUID: "222", Stat: "kills"}}
	d.SkullCarries = []Portage{{XUID: "111", T0: 110, T1: 160}}
	d.FlagCarries = []Drapeau{{Spans: []EtatDrapeau{{State: etatDrapeauPorte, T0: 20, T1: 90, XUID: ptr("111")}, {State: "dropped", T0: 91, T1: 95}}}}
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolAbsent, 3)
}

// TestTrajet_PassagerLoinDuVehicule : un passager a plus de rayonTrajetM de son vehicule a la meme
// image est un faux ; a moins, non ; sans point a ±1 image, non jugeable.
func TestTrajet_PassagerLoinDuVehicule(t *testing.T) {
	d := documentJuste()
	d.Tracks[0].Points = []Point{{T: 10, X: 0, Y: 0}, {T: 20, X: 0, Y: 0}, {T: 30, X: 0, Y: 0}}
	d.Vehicles = []Vehicule{{Slot: 700,
		Samples: []Echantillon{{T: 10, X: 1, Y: 0}, {T: 20, X: rayonTrajetM + 1, Y: 0}, {T: 25, X: 50, Y: 0}, {T: 40, X: 50, Y: 0}},
		Rides:   []Trajet{{T0: 5, T1: 35, Slot: 512}}}}
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolTrajetLoin, 1)
}

// TestIdentite_CompteursPublies : V-7 somme les desaccords publies.
func TestIdentite_CompteursPublies(t *testing.T) {
	d := documentJuste()
	d.Coverage.Bridge = &CouvPont{Discordant: 2, SlotCollisions: 1}
	d.Coverage.Seats = &CouvSieges{Chevauchements: 1}
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolIdentite, 4)
}

// TestMortSansVie_IncrementLoinDeToutesLesFinsDeVie : une mort du statborg loin de toute fin de vie
// du joueur est un faux ; un joueur sans vie nommee n'est pas juge.
func TestMortSansVie_IncrementLoinDeToutesLesFinsDeVie(t *testing.T) {
	d := documentJuste()
	d.ScoreTimeline.Players[0].Deaths = serie(Pas{0, 0}, Pas{100 + toleranceMortImages, 1}, Pas{220, 2})
	d.ScoreTimeline.Players = append(d.ScoreTimeline.Players, ScoreJoueur{XUID: "333", Deaths: serie(Pas{50, 1})})
	exigerViolations(t, Noter(d, faitsJustes(), nil), ViolMortSansVie, 1)
}

// TestPreuves_FermetureContradictionsVerdicts : P-1, P-2 et le rang des verdicts P-4.
func TestPreuves_FermetureContradictionsVerdicts(t *testing.T) {
	d := documentJuste()
	d.Coverage.Keyframes = &CouvImagesCles{Refutations: 2, ContradictoryProofs: 1}
	d.Coverage.Verdict = map[string]string{"a": "nominal", "b": "partiel : moins des deux tiers", "c": "non publiable : x"}
	p := Noter(d, faitsJustes(), nil).Preuves
	if p[PreuveFermeture].Valeur != 80 || !p[PreuveFermeture].PlusEstMieux {
		t.Errorf("P-1 = %+v", p[PreuveFermeture])
	}
	if p[PreuveContradic].Valeur != 3 || p[PreuveContradic].PlusEstMieux {
		t.Errorf("P-2 = %+v", p[PreuveContradic])
	}
	for cle, veut := range map[string]int{"a": rangNominal, "b": rangPartiel, "c": rangNonPubliable} {
		if got := p[PreuveVerdicts+" "+cle].Valeur; got != veut {
			t.Errorf("P-4 %s = %d, veut %d", cle, got, veut)
		}
	}
}
