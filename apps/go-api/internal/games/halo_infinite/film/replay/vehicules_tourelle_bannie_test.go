package replay

// vehicules_tourelle_bannie_test.go — LE TEMOIN `bfecd02b` : NEUF TOURELLES AUTOMATIQUES
// BANNIES, NOMMEES, A LEUR POSITION (lot 1.9.9).
//
// D OU VIENNENT LES NEUF VIES. De l ARTEFACT `bfecd02b` deja cuit (schema 54), lu une fois le
// 2026-09-16 par l instrument `TestInventaireChassisDesArtefacts` puis FIGE ici. Aucun film n est
// decode par ce test : les positions de naissance, la fenetre et le nombre d echantillons sont
// ceux que la production a publies, et le test rejoue l assemblage de production
// (`buildVehicleTracks`) dessus.
//
// POURQUOI UNE FIXTURE FIGEE PLUTOT QUE L ARTEFACT LU. Le parc (`data/cache/replays`) ne vit que
// dans le checkout principal, il n est pas versionne, et il se recuit par jalon : un test qui en
// depend saute partout ailleurs (CI comprise) et ne tient donc rien. Les neuf lignes ci-dessous
// sont la DONNEE, pas une invention — chacune est reproductible par
// `jq '.vehicles[] | select(.chassis=="038df01a")' bfecd02b.json`.
//
// CE QUE LE TEST TIENT, ET C EST LA MUTATION DEMANDEE PAR LE LOT : retirer l entree
// `0x038df01a` de `vehicleFamilyByChassis` fait rougir `TestTemoinBfecd02bNeufTourellesNommees`
// (famille vide, repli `repli_chassis_vehicule_marqueur_neutre` declenche neuf fois). Mutation
// jouee et restauree par NOM le 2026-09-16.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// tourelleBannieChassis : le mot d identite lu dans le default-state des records `ti=40` de
// `bfecd02b`. Ecrit ici en clair (et pas via la constante de famille) pour que le test echoue si
// la TABLE change de cle, pas seulement si elle change de valeur.
const tourelleBannieChassis = uint32(0x038df01a)

// tourelleBannieVie — une des neuf vies du temoin, telle que l artefact la publie.
type tourelleBannieVie struct {
	slot       uint32
	x, y, z    float32
	echantillo int // nombre d echantillons de trajectoire publies
}

// tourellesDeBfecd02b : les NEUF vies de chassis `038df01a` du temoin, dans l ordre des slots.
//
// TOUTES ont `t0 = 0`, `t1 = t1max = 5032` (le film en compte 5033 images de 100 ms) : leur
// fenetre est LE MATCH ENTIER. HUIT n ont AUCUN echantillon de trajectoire, la neuvieme en a
// deux — c est la signature d un objet fixe, pas d un vehicule.
var tourellesDeBfecd02b = []tourelleBannieVie{
	{slot: 768, x: -8.98, y: -90.14, z: 56.61},
	{slot: 769, x: -3.88, y: -118.99, z: 55.57},
	{slot: 770, x: -11.35, y: -105.06, z: 56.21},
	{slot: 771, x: 9.13, y: -124.34, z: 56.53},
	{slot: 772, x: 24.13, y: -125.18, z: 56.76},
	{slot: 773, x: 38.84, y: -121.88, z: 57.55, echantillo: 2},
	{slot: 774, x: 18.61, y: -74.68, z: 58.83},
	{slot: 775, x: 34.46, y: -79.5, z: 58.52},
	{slot: 776, x: 51.04, y: -93.96, z: 58.76},
}

// scanDesTourelles rejoue l entree du constructeur de vehicules pour les neuf vies du temoin.
func scanDesTourelles() VehicleScan {
	const naissanceUS = 1_200_000
	kf := grammar.WorldObjectKeyframes{
		Band:    map[uint32]bool{},
		TimesUS: []uint64{2_000_000, 22_000_000, 42_000_000},
		SeenUS:  map[grammar.EquipmentLifeKey][]uint64{},
	}
	scan := VehicleScan{Scanned: true}
	for _, v := range tourellesDeBfecd02b {
		key := grammar.EquipmentLifeKey{Slot: v.slot, Gen: 1}
		kf.Band[v.slot] = true
		kf.SeenUS[key] = []uint64{2_000_000, 22_000_000, 42_000_000}
		c := vehCreation(key, naissanceUS, v.x, v.y, tourelleBannieChassis)
		c.Z = v.z
		scan.Creations = append(scan.Creations, c)
		for i := 0; i < v.echantillo; i++ {
			scan.Positions = append(scan.Positions,
				vehPos(v.slot, uint64(3_000_000+i*1_000_000), v.x, v.y))
		}
	}
	scan.Keyframes = kf
	return scan
}

// TestTemoinBfecd02bNeufTourellesNommees — LE TEST D ACCEPTATION DU LOT.
//
// Neuf vies publiees, toutes de famille `tourelle_auto_bannie`, chacune a SA position de
// naissance, AUCUNE avec un occupant, et AUCUN declenchement du repli du marqueur neutre.
func TestTemoinBfecd02bNeufTourellesNommees(t *testing.T) {
	fb := fallback.NouveauCompteur()
	clock := vehClock()
	clock.fb = fb
	got, cov, _ := buildVehicleTracks(scanDesTourelles(), nil, IdentityRegistry{}, clock)
	if len(got) != len(tourellesDeBfecd02b) {
		t.Fatalf("vies publiees = %d, attendu %d (les neuf tourelles du temoin bfecd02b)",
			len(got), len(tourellesDeBfecd02b))
	}
	parSlot := map[uint32]VehicleTrack{}
	for _, tr := range got {
		parSlot[tr.Slot] = tr
	}
	for _, v := range tourellesDeBfecd02b {
		tr, ok := parSlot[v.slot]
		if !ok {
			t.Errorf("slot %d absent des vies publiees", v.slot)
			continue
		}
		verifieTourelle(t, v, tr)
	}
	if cov.FamilyResolved != len(tourellesDeBfecd02b) || cov.FamilyUnknown != 0 {
		t.Errorf("couverture : famillesResolues=%d famillesInconnues=%d, attendu %d et 0",
			cov.FamilyResolved, cov.FamilyUnknown, len(tourellesDeBfecd02b))
	}
	if n := fb.Compte(fallback.NomChassisVehiculeMarqueurNeutre); n != 0 {
		t.Errorf("repli %q declenche %d fois, attendu 0 : le chassis est NOMME en table",
			fallback.NomChassisVehiculeMarqueurNeutre, n)
	}
}

// verifieTourelle : une vie nommee, a sa position, sans occupant.
func verifieTourelle(t *testing.T, v tourelleBannieVie, tr VehicleTrack) {
	t.Helper()
	if tr.Family != familleTourelleAutoBannie {
		t.Errorf("slot %d : famille = %q, attendu %q — sans elle le client dessine le marqueur"+
			" neutre, ce que la decision utilisateur du 2026-09-14 refuse",
			v.slot, tr.Family, familleTourelleAutoBannie)
	}
	if tr.Chassis != "038df01a" {
		t.Errorf("slot %d : chassis publie = %q, attendu \"038df01a\"", v.slot, tr.Chassis)
	}
	if tr.Spawn == nil {
		t.Fatalf("slot %d : aucune naissance publiee — rien a dessiner a sa position", v.slot)
	}
	if tr.Spawn.X != v.x || tr.Spawn.Y != v.y {
		t.Errorf("slot %d : position publiee (%.2f, %.2f), attendu (%.2f, %.2f)",
			v.slot, tr.Spawn.X, tr.Spawn.Y, v.x, v.y)
	}
	if len(tr.Rides) != 0 {
		t.Errorf("slot %d : %d episode(s) d occupation, attendu 0 — un element de carte ne porte"+
			" JAMAIS d occupant (cf. vehicleFamillesNonPilotables)", v.slot, len(tr.Rides))
	}
}

// TestTourelleBannieNEstPasPilotable : la famille refuse tout episode d occupation.
//
// C EST LA GARDE QUI PROTEGE UN JOUEUR REEL. Un episode accroche a un objet de decor efface le
// pion du joueur que le pont croit embarque — le defaut exact constate le 2026-09-02 sur les
// props `falcon` de `0d76e8f1`.
func TestTourelleBannieNEstPasPilotable(t *testing.T) {
	if vehicleFamilyIsRideable(familleTourelleAutoBannie) {
		t.Errorf("%q est declaree pilotable : un element de carte immobile ne porte pas"+
			" d occupant", familleTourelleAutoBannie)
	}
	// CONTROLE : une famille de vehicule reelle, elle, le reste — sans quoi ce test passerait
	// aussi si `vehicleFamilyIsRideable` rendait faux pour tout.
	if !vehicleFamilyIsRideable(familleWarthog) {
		t.Errorf("%q n est plus pilotable : la garde de ce lot a deborde sur les vehicules",
			familleWarthog)
	}
}

// TestChassisInconnuCompteLeRepli : tout AUTRE chassis absent de la table reste un repli COMPTE.
//
// La mesure du lot (76 artefacts du parc) en a releve d autres que la tourelle — dont un
// MOBILE, `0xae845375`, 18 vies sur quatre films. Ils ne sont PAS nommes par ce lot (regle 7 :
// zero fix opportuniste) : ils doivent donc continuer de se compter, sous leur nom de repli.
func TestChassisInconnuCompteLeRepli(t *testing.T) {
	key := grammar.EquipmentLifeKey{Slot: 700, Gen: 1}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000}),
		Creations: []grammar.EquipmentCreation{vehCreation(key, 1_500_000, 1, 2, vehChassisUnknown)},
	}
	fb := fallback.NouveauCompteur()
	clock := vehClock()
	clock.fb = fb
	_, cov, _ := buildVehicleTracks(scan, nil, IdentityRegistry{}, clock)
	if cov.FamilyUnknown != 1 {
		t.Fatalf("famillesInconnues = %d, attendu 1", cov.FamilyUnknown)
	}
	if n := fb.Compte(fallback.NomChassisVehiculeMarqueurNeutre); n != 1 {
		t.Errorf("repli %q declenche %d fois, attendu 1 : un chassis hors table doit se COMPTER,"+
			" pas se taire", fallback.NomChassisVehiculeMarqueurNeutre, n)
	}
}

// --- LES SECONDS `vehi` DU MANIFESTE (lot 1.9.9, elargissement du 2026-09-16) -----------------

// TestSecondsChassisDuManifesteSontEnTable : les sept identifiants que le manifeste de la chaine
// de destruction rattache a une famille DEJA en table y sont, et sous la bonne famille.
//
// POURQUOI CE TEST EXISTE, ET CE QU IL A COUTE DE NE PAS L AVOIR. Un meme vehicule est declare
// dans PLUSIEURS modules du jeu, sous un GlobalID par module (cf. l en-tete de
// `vehicle_families.go`). La table avait ete peuplee depuis `pc/globals` ; les films ecrivent
// l identifiant du module charge. Resultat mesure sur 76 artefacts : le Wraith et le Scorpion,
// qui n avaient QUE leur identifiant de base, ne se resolvaient JAMAIS — 21 vies publiees sans
// sprite ET sans occupant. Ce test fige la correspondance, piece par piece.
func TestSecondsChassisDuManifesteSontEnTable(t *testing.T) {
	cas := []struct {
		chassis uint32
		famille string
		piece   string
	}{
		{0xae845375, familleWraith, `manifeste « Wraith » : vehi "00002706 (+ ae845375)", hlmt 5b5c960d`},
		{0xf6f54e56, familleScorpion, `manifeste « Scorpion » : "f6f54e56 (any/globals) = chassis 0000d3db (pc/globals), meme hlmt"`},
		{0x9af9e693, familleGhost, `manifeste « Ghost » : vehi "0000d3dc (+ 5b80c406, 9af9e693)"`},
		{0x0001530a, familleBanshee, `manifeste « Banshee » : vehi "000026ed (+ 0001530a, c6e79dcc)"`},
		{0x5159c8ef, familleWarthog, `manifeste « Warthog » : "(+ 5159c8ef, 75312e51, 7617ff6e dans any/globals/common)"`},
		{0x75312e51, familleWarthog, `idem`},
		{0x7617ff6e, familleWarthog, `idem`},
	}
	for _, c := range cas {
		if got := vehicleFamilyOf(c.chassis); got != c.famille {
			t.Errorf("vehicleFamilyOf(%#08x) = %q, attendu %q — %s",
				c.chassis, got, c.famille, c.piece)
		}
	}
}

// TestChassisWraithPublieSesOccupants — L ACCEPTATION DE L ELARGISSEMENT.
//
// CE QU ELLE PROUVE. Une vie de chassis `0xae845375` publie desormais ses EPISODES D OCCUPATION,
// avec le xuid de leur occupant. Avant l entree en table, `vehicleTrackOf` les JETAIT tous :
// `vehicleFamilyIsRideable("")` est faux, donc `tr.Rides = nil` sur toute famille vide. Ce n est
// PAS une subtilite theorique — c est ce que le parc montre, film par film : sur `4f77afc1`,
// les familles resolues portent 36/36, 15/15, 9/9 et 1/1 occupants nommes, et les 11 vies de
// `ae845375` en portent ZERO malgre 376 a 2 352 echantillons de trajectoire chacune.
func TestChassisWraithPublieSesOccupants(t *testing.T) {
	const wraithDuFilm = uint32(0xae845375)
	key := grammar.EquipmentLifeKey{Slot: 700, Gen: 1}
	const bipedSlot = uint32(42)
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []grammar.EquipmentCreation{vehCreation(key, 1_500_000, 0, 0, wraithDuFilm)},
		Positions: []grammar.BipedPosition{
			vehPos(700, 5_000_000, 0, 0),
			vehPos(700, 12_000_000, 0, 0),
			vehPos(700, 20_000_000, 0, 0),
		},
		Events: []grammar.VehicleEvent{
			{Kind: grammar.EventBipedBoardVehicle, TimestampUS: 5_200_000, OccupantPresent: true,
				OccupantInBand: true, OccupantSlot: bipedSlot, Seat: 0, SeatValid: true},
			{Kind: grammar.EventUnitExitVehicle, TimestampUS: 16_800_000, OccupantPresent: true,
				OccupantInBand: true, OccupantSlot: bipedSlot, Seat: 0, SeatValid: true},
		},
	}
	bipeds := []grammar.BipedPosition{
		vehPos(bipedSlot, 4_500_000, 0.4, 0),
		vehPos(bipedSlot, 5_000_000, 0.4, 0),
		vehPos(bipedSlot, 17_000_000, 3, 3),
	}
	own := regDe(OwnerReport{SlotXUID: map[uint32]uint64{bipedSlot: 2533274800000001}})
	got, cov, _ := buildVehicleTracks(scan, bipeds, own, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1", len(got))
	}
	tr := got[0]
	if tr.Family != familleWraith {
		t.Fatalf("famille = %q, attendu %q : sans elle, tout ce qui suit est jete",
			tr.Family, familleWraith)
	}
	if len(tr.Rides) != 1 {
		t.Fatalf("episodes d occupation = %d, attendu 1 — une famille vide les jetait TOUS "+
			"(vehicleFamilyIsRideable)", len(tr.Rides))
	}
	if tr.Rides[0].XUID == "" {
		t.Errorf("occupant ANONYME : le pont slot -> xuid doit le nommer, comme sur toute autre " +
			"famille pilotable")
	}
	if cov.RidesNamed != 1 || cov.FamilyResolved != 1 || cov.FamilyUnknown != 0 {
		t.Errorf("couverture = {famillesResolues:%d famillesInconnues:%d occupantsNommes:%d}, "+
			"attendu 1 / 0 / 1", cov.FamilyResolved, cov.FamilyUnknown, cov.RidesNamed)
	}
}
