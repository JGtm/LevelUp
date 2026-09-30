package replay

// vehicle_takes_test.go — UN TEST PAR REGLE de `ProjectVehicleTakes` (lot L7.1 du plan
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`). Chaque test est rouge sous une mutation nommee au
// compte rendu du lot ; aucun ne lit un film.

import "testing"

// Joueurs synthetiques : camp 0 (100, 101), camp 1 (200, 201), sans camp (300 : -1), bot sans xuid.
const (
	tkA1 = "100"
	tkA2 = "101"
	tkB1 = "200"
	tkB2 = "201"
	tkNC = "300"
)

func tkInt(v int) *int { return &v }

// tkDoc : un document au schema courant, calque balaye, pas d image de 100 ms.
func tkDoc(vehicles ...VehicleTrack) *ReplayDocument {
	return &ReplayDocument{
		SchemaVersion:   SchemaVersion,
		FrameIntervalMS: 100,
		Coverage:        &Coverage{Vehicles: &VehicleCoverage{Scanned: true}},
		Roster: []RosterEntry{
			{XUID: tkA1, Team: tkInt(0)}, {XUID: tkA2, Team: tkInt(0)},
			{XUID: tkB1, Team: tkInt(1)}, {XUID: tkB2, Team: tkInt(1)},
			{XUID: tkNC, Team: tkInt(-1)}, {XUID: "", Team: tkInt(1)}, // un bot : pas de xuid
		},
		Vehicles: vehicles,
	}
}

func tkLife(slot uint32, family string, rides ...VehicleRide) VehicleTrack {
	return VehicleTrack{Slot: slot, Gen: 1, Family: family, Rides: rides}
}

func tkRide(xuid string, t0, t1 int) VehicleRide {
	return VehicleRide{XUID: xuid, T0: t0, T1: t1, Src: VehicleRideSrcFilm}
}

func tkRow(rep VehicleTakesReport, xuid, family string) (VehicleUsageRow, bool) {
	for _, r := range rep.Rows {
		if r.XUID == xuid && r.Family == family {
			return r, true
		}
	}
	return VehicleUsageRow{}, false
}

func tkTakes(rep VehicleTakesReport, camp int) int {
	n := 0
	for _, r := range rep.Rows {
		if r.Camp == camp {
			n += r.Takes
		}
	}
	return n
}

func TestVehicleTakes_ChangementDeCamp(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20), tkRide(tkB1, 30, 40))))
	if !rep.Measured {
		t.Fatalf("calcul non mesure : %s", rep.Reason)
	}
	if tkTakes(rep, 0) != 1 || tkTakes(rep, 1) != 1 {
		t.Fatalf("prises camp0=%d camp1=%d, attendu 1 et 1 : %+v", tkTakes(rep, 0), tkTakes(rep, 1), rep.Rows)
	}
	if b, _ := tkRow(rep, tkB1, "warthog"); b.Takes != 1 {
		t.Fatalf("le joueur adverse doit etre credite de la prise : %+v", b)
	}
}

func TestVehicleTakes_RetourAuMemeCampApresUnPassageAdverse(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "warthog",
		tkRide(tkA1, 10, 20), tkRide(tkB1, 30, 40), tkRide(tkA2, 50, 60))))
	if got := tkTakes(rep, 0); got != 2 {
		t.Fatalf("prises du camp 0 = %d, attendu 2 (retour apres un passage adverse) : %+v", got, rep.Rows)
	}
	if a2, _ := tkRow(rep, tkA2, "warthog"); a2.Takes != 1 {
		t.Fatalf("le joueur qui reprend la vie doit etre credite : %+v", a2)
	}
}

func TestVehicleTakes_MemeCampSuccessifNeFaitQuUnePrise(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "ghost",
		tkRide(tkA1, 10, 20), tkRide(tkA2, 30, 40), tkRide(tkA1, 50, 60))))
	if got := tkTakes(rep, 0); got != 1 {
		t.Fatalf("prises du camp 0 = %d, attendu 1 (aucun adversaire monte) : %+v", got, rep.Rows)
	}
	if a2, _ := tkRow(rep, tkA2, "ghost"); a2.Takes != 0 {
		t.Fatalf("un coequipier qui monte apres coup ne prend rien : %+v", a2)
	}
}

func TestVehicleTakes_SiegesSimultanesLeConducteurEstCredite(t *testing.T) {
	passager := VehicleRide{XUID: tkA2, T0: 10, T1: 20, Src: VehicleRideSrcFilm, Seat: tkInt(1)}
	conducteur := VehicleRide{XUID: tkA1, T0: 10, T1: 20, Src: VehicleRideSrcFilm, Seat: tkInt(0)}
	inconnu := VehicleRide{XUID: tkB2, T0: 10, T1: 20, Src: VehicleRideSrcFilm} // sans siege : apres
	// Le passager puis le siege inconnu sont LISTES AVANT le conducteur : l ordre du document ne decide pas.
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "warthog", passager, inconnu, conducteur)))
	if c, _ := tkRow(rep, tkA1, "warthog"); c.Takes != 1 {
		t.Fatalf("le conducteur (siege 0) doit etre credite : %+v", rep.Rows)
	}
	if p, _ := tkRow(rep, tkA2, "warthog"); p.Takes != 0 {
		t.Fatalf("le passager ne prend pas la vie a la meme image : %+v", p)
	}
	// L occupant de siege inconnu, adverse, fait ensuite SA prise (il change le camp de la vie).
	if b, _ := tkRow(rep, tkB2, "warthog"); b.Takes != 1 {
		t.Fatalf("l adversaire de siege inconnu passe apres le conducteur : %+v", rep.Rows)
	}
}

func TestVehicleTakes_LeDecorPublieEstEcarteEtUneVieSansEpisodeNeFaitRien(t *testing.T) {
	doc := tkDoc(
		tkLife(1, "mongoose", tkRide(tkA1, 10, 20)),
		tkLife(2, "ghost", tkRide(tkB1, 10, 20)),
		tkLife(3, "warthog"), // aucun episode : pas de prise, pas de ligne
	)
	doc.VehicleScenery = &VehicleScenery{Hidden: []VehicleSceneryLife{{Slot: 1, Gen: 1, Reason: "off_play_area"}}}
	rep := ProjectVehicleTakes(doc)
	if _, ok := tkRow(rep, tkA1, "mongoose"); ok {
		t.Fatalf("la vie de decor publiee ne doit donner aucune ligne : %+v", rep.Rows)
	}
	if _, ok := tkRow(rep, tkB1, "ghost"); !ok {
		t.Fatalf("la vie jouee doit rester : %+v", rep.Rows)
	}
	if rep.Coverage.HiddenLives != 1 || rep.Coverage.Lives != 3 || rep.Coverage.LivesWithRides != 1 {
		t.Fatalf("couverture = %+v, attendu 3 vies, 1 decor, 1 avec episode", rep.Coverage)
	}
}

func TestVehicleTakes_LaPieceMonteeAppartientAuPorteur(t *testing.T) {
	porteur := tkLife(10, "warthog", tkRide(tkA1, 10, 30))
	piece := tkLife(9, "", tkRide(tkB1, 40, 60)) // tourelle : famille propre vide, episode reste sur elle
	piece.Part, piece.Carrier = VehiclePartTurret, &VehicleLifeRef{Slot: 10, Gen: 1}
	rep := ProjectVehicleTakes(tkDoc(piece, porteur)) // la piece est listee AVANT son porteur
	b, ok := tkRow(rep, tkB1, "warthog")
	if !ok || b.Takes != 1 || b.AboardMS != 2000 {
		t.Fatalf("l episode de la piece doit etre compte sur la famille du porteur : %+v", rep.Rows)
	}
	if _, ok := tkRow(rep, tkB1, VehicleFamilyUnknown); ok {
		t.Fatalf("aucune ligne ne doit sortir sous la famille propre de la piece : %+v", rep.Rows)
	}
	if rep.Coverage.PartRides != 1 {
		t.Fatalf("PartRides = %d, attendu 1", rep.Coverage.PartRides)
	}
}

func TestVehicleTakes_UnOccupantDejaABordDuPorteurNeDoublePasLeTemps(t *testing.T) {
	porteur := tkLife(10, "warthog", tkRide(tkA1, 10, 30))
	piece := tkLife(9, "", tkRide(tkA1, 20, 40)) // meme occupant, intervalle qui chevauche
	piece.Part, piece.Carrier = VehiclePartTurret, &VehicleLifeRef{Slot: 10, Gen: 1}
	rep := ProjectVehicleTakes(tkDoc(porteur, piece))
	a, _ := tkRow(rep, tkA1, "warthog")
	if a.AboardMS != 2000 || a.Episodes != 1 {
		t.Fatalf("temps = %d ms sur %d episode(s), attendu 2000 ms et 1 episode : %+v", a.AboardMS, a.Episodes, a)
	}
	if rep.Coverage.PartRidesDuplicate != 1 {
		t.Fatalf("PartRidesDuplicate = %d, attendu 1", rep.Coverage.PartRidesDuplicate)
	}
}

func TestVehicleTakes_UnePieceSansPorteurResteSaPropreVie(t *testing.T) {
	piece := tkLife(9, "", tkRide(tkA1, 10, 20))
	piece.Part = VehiclePartTurret
	piece.Carrier = &VehicleLifeRef{Slot: 99, Gen: 1} // porteur absent du document
	rep := ProjectVehicleTakes(tkDoc(piece))
	a, ok := tkRow(rep, tkA1, VehicleFamilyUnknown)
	if !ok || a.Takes != 1 || rep.Coverage.OrphanParts != 1 || rep.Coverage.PartRides != 0 {
		t.Fatalf("tourelle sans porteur : ligne %+v, couverture %+v", rep.Rows, rep.Coverage)
	}
}

func TestVehicleTakes_LaFamilleInconnueEstNommee(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "", tkRide(tkA1, 10, 20))))
	if r, ok := tkRow(rep, tkA1, VehicleFamilyUnknown); !ok || r.Takes != 1 {
		t.Fatalf("famille inconnue non nommee : %+v", rep.Rows)
	}
	if _, ok := tkRow(rep, tkA1, ""); ok {
		t.Fatalf("une famille vide ne doit jamais sortir : %+v", rep.Rows)
	}
}

func TestVehicleTakes_LesEpisodesDeRepliComptentEtSontMesures(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "falcon",
		VehicleRide{XUID: tkA1, T0: 0, T1: 50, Src: vehicleTakesSrcProximity},
		tkRide(tkB1, 60, 70))))
	a, _ := tkRow(rep, tkA1, "falcon")
	if a.Takes != 1 || a.AboardMS != 5000 || a.ProximityEpisodes != 1 {
		t.Fatalf("l episode proximity doit compter (prise, temps) et etre marque : %+v", a)
	}
	c := rep.Coverage
	if c.ProximityEpisodes != 1 || c.ProximityMS != 5000 || c.Episodes != 2 {
		t.Fatalf("couverture = %+v, attendu 1 episode de repli sur 2, 5000 ms", c)
	}
}

func TestVehicleTakes_TempsABordEstEpisodesFoisIntervalle(t *testing.T) {
	doc := tkDoc(tkLife(1, "ghost", tkRide(tkA1, 10, 20), tkRide(tkA1, 100, 130)))
	doc.FrameIntervalMS = 50
	rep := ProjectVehicleTakes(doc)
	if a, _ := tkRow(rep, tkA1, "ghost"); a.AboardMS != (10+30)*50 || a.Episodes != 2 {
		t.Fatalf("temps = %d ms, attendu %d : %+v", a.AboardMS, (10+30)*50, a)
	}
}

func TestVehicleTakes_UnOccupantSansXUIDOuSansCampNeFaitNiPriseNiTemps(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "warthog",
		tkRide("", 0, 10), tkRide(tkNC, 20, 30), tkRide("999", 40, 50), tkRide(tkA1, 60, 70))))
	if len(rep.Rows) != 1 || rep.Rows[0].XUID != tkA1 {
		t.Fatalf("seul l occupant au camp connu doit sortir : %+v", rep.Rows)
	}
	c := rep.Coverage
	if c.EpisodesNoXUID != 1 || c.EpisodesNoCamp != 2 {
		t.Fatalf("couverture = %+v, attendu 1 sans xuid et 2 sans camp (mode sans equipe, joueur hors roster)", c)
	}
}

func TestVehicleTakes_SansOccupationLueLeMatchEstNonMesure(t *testing.T) {
	ancien := tkDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	ancien.SchemaVersion = vehicleTakesMinSchema - 1
	if rep := ProjectVehicleTakes(ancien); rep.Measured || rep.Reason != VehicleTakesUnmeasuredSchema || len(rep.Rows) != 0 {
		t.Fatalf("un artefact de schema < 67 doit etre « non mesure », sans ligne : %+v", rep)
	}
	sansBalayage := tkDoc()
	sansBalayage.Coverage.Vehicles.Scanned = false
	if rep := ProjectVehicleTakes(sansBalayage); rep.Measured || rep.Reason != VehicleTakesUnmeasuredNotScanned {
		t.Fatalf("un calque non balaye doit etre « non mesure » : %+v", rep)
	}
	sansPas := tkDoc()
	sansPas.FrameIntervalMS = 0
	if rep := ProjectVehicleTakes(sansPas); rep.Measured || rep.Reason != VehicleTakesUnmeasuredNoInterval {
		t.Fatalf("sans intervalle d image le temps n est pas mesurable : %+v", rep)
	}
	if rep := ProjectVehicleTakes(nil); rep.Measured {
		t.Fatalf("un document absent n est pas mesure : %+v", rep)
	}
}

func TestVehicleTakes_UnFilmSansVehiculeEstMesureAZero(t *testing.T) {
	rep := ProjectVehicleTakes(tkDoc())
	if !rep.Measured || len(rep.Rows) != 0 {
		t.Fatalf("un film balaye sans vehicule est un ZERO mesure, pas « non mesure » : %+v", rep)
	}
}

func TestVehicleTakes_SortieTrieeEtStable(t *testing.T) {
	doc := tkDoc(
		tkLife(2, "warthog", tkRide(tkB1, 10, 20)),
		tkLife(1, "ghost", tkRide(tkA2, 10, 20)),
		tkLife(3, "ghost", tkRide(tkA1, 10, 20)),
	)
	rep := ProjectVehicleTakes(doc)
	want := []string{tkA1, tkA2, tkB1}
	if len(rep.Rows) != len(want) {
		t.Fatalf("lignes = %+v, attendu %d", rep.Rows, len(want))
	}
	for i, r := range rep.Rows {
		if r.XUID != want[i] {
			t.Fatalf("ordre des lignes = %+v, attendu les xuid %v", rep.Rows, want)
		}
	}
}

func TestVehicleTakes_ConstantesAlignees(t *testing.T) {
	if vehicleTakesSrcProximity != VehicleRideSrcProximity {
		t.Fatalf("la provenance de repli (%q) a derive de la constante publiee (%q)",
			vehicleTakesSrcProximity, VehicleRideSrcProximity)
	}
	if SchemaVersion < vehicleTakesMinSchema {
		t.Fatalf("le schema courant (%d) est sous le plancher du calcul (%d)", SchemaVersion, vehicleTakesMinSchema)
	}
}

func TestVehicleTakes_LOrdreEstCeluiDeLaMonteeEtPasCeluiDuDocument(t *testing.T) {
	// Listes a l envers : B1 (image 50), A2 (30), A1 (10). Dans l ordre de montee, A1 prend la vie
	// pour le camp 0, A2 (meme camp) ne prend rien, B1 la reprend pour le camp 1.
	rep := ProjectVehicleTakes(tkDoc(tkLife(1, "ghost",
		tkRide(tkB1, 50, 60), tkRide(tkA2, 30, 40), tkRide(tkA1, 10, 20))))
	a1, _ := tkRow(rep, tkA1, "ghost")
	a2, _ := tkRow(rep, tkA2, "ghost")
	b1, _ := tkRow(rep, tkB1, "ghost")
	if a1.Takes != 1 || a2.Takes != 0 || b1.Takes != 1 {
		t.Fatalf("prises A1=%d A2=%d B1=%d, attendu 1, 0, 1 : %+v", a1.Takes, a2.Takes, b1.Takes, rep.Rows)
	}
}
