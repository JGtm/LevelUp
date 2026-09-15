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

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
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
	kf := filmdec.WorldObjectKeyframes{
		Band:    map[uint32]bool{},
		TimesUS: []uint64{2_000_000, 22_000_000, 42_000_000},
		SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{},
	}
	scan := VehicleScan{Scanned: true}
	for _, v := range tourellesDeBfecd02b {
		key := filmdec.EquipmentLifeKey{Slot: v.slot, Gen: 1}
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
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_500_000, 1, 2, vehChassisUnknown)},
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
