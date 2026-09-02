package replay

// build_vehicles_test.go — les REGLES du calque des vehicules, sur fixtures.
//
// POURQUOI CES TESTS EXISTENT. Le golden d assemblage ne couvre pas ce calque (sa fixture lui est
// anterieure) et ses regles decident de ce que l utilisateur VOIT : un sprite oriente ou non, un
// vehicule qui disparait trop tot, un conducteur attribue au mauvais vehicule. Chaque test
// correspond a une decision ECRITE dans l en-tete du fichier de production, et aucune ne fabrique
// un chiffre : les seuils employes ici sont ceux des constantes de production.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehClock : origine 1 s, pas 100 ms, 1 200 frames (2 minutes) — les frames attendues se lisent
// de tete (t = 1 s -> frame 0, t = 11 s -> frame 100).
func vehClock() replayClock {
	return replayClock{origin: 1_000_000, step: 100_000, frames: 1200}
}

// vehChassisKnown / vehChassisUnknown : un mot d identite present dans la table statique, et un
// qui n y est pas. Le premier est celui du Ghost observe en film (cf. vehicle_families.go).
const (
	vehChassisKnown   = uint32(0x5b80c406)
	vehChassisUnknown = uint32(0xdeadbeef)
)

// vehKeyframes fabrique un recensement : `timesUS` sont TOUTES les images-cles du film, `seen`
// celles qui recensent la vie (slot, gen).
func vehKeyframes(timesUS []uint64, key filmdec.EquipmentLifeKey, seen []uint64) filmdec.WorldObjectKeyframes {
	return filmdec.WorldObjectKeyframes{
		Band:    map[uint32]bool{key.Slot: true},
		TimesUS: timesUS,
		SeenUS:  map[filmdec.EquipmentLifeKey][]uint64{key: seen},
	}
}

// vehCreation fabrique un record de creation : naissance datee, position, mot d identite.
func vehCreation(key filmdec.EquipmentLifeKey, tUS uint64, x, y float32, chassis uint32) filmdec.EquipmentCreation {
	c := filmdec.EquipmentCreation{Slot: key.Slot, Gen: key.Gen, TimestampUS: tUS, X: x, Y: y}
	c.MPPPresent[filmdec.MPPWord32] = true
	c.MPPVal[filmdec.MPPWord32] = uint64(chassis)
	return c
}

// vehPos fabrique un echantillon de position monde IMMOBILE (aucune velocite lue).
func vehPos(slot uint32, tUS uint64, x, y float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{Slot: slot, TimestampUS: tUS, X: x, Y: y, HasWorld: true}
}

// vehMoving fabrique un echantillon PORTEUR d une velocite : direction `dir` (cubemap 19 bits) et
// magnitude quantifiee. C est ce que `CaptureDirs` remplit sur un record `ti=40`.
func vehMoving(t *testing.T, slot uint32, tUS uint64, x, y float32, dir [3]float32, mps float64) filmdec.BipedPosition {
	t.Helper()
	raw, ok := filmdec.EncodeAimVector(dir, vehTestDirBits)
	if !ok {
		t.Fatalf("direction %v non encodable en cubemap %d bits", dir, vehTestDirBits)
	}
	p := vehPos(slot, tUS, x, y)
	p.HasVel = true
	p.VelRaw = raw
	p.VelScale = vehVelScale(mps)
	return p
}

// vehTestDirBits / vehTestScaleBits : les largeurs que `BipedPosition.VelocityVector` emploie
// pour relire ce que la fixture ecrit (cubemap 19 bits, magnitude log/exp 10 bits).
const (
	vehTestDirBits   = uint(19)
	vehTestScaleBits = uint(10)
)

// vehVelScale rend le plus petit quantum de magnitude dont le DECODEUR DE PRODUCTION rend au
// moins `mps`. La quantification est log/exp : l inverser a la main dans une fixture serait une
// seconde ecriture de la formule, et elle divergerait.
func vehVelScale(mps float64) uint32 {
	n := uint32(1) << vehTestScaleBits
	for s := uint32(0); s < n; s++ {
		if float64(filmdec.DecodeVelocityMagnitude(uint64(s), vehTestScaleBits)) >= mps {
			return s
		}
	}
	return n - 1
}

// --- Vie SANS occupant : elle sort quand meme, avec sa naissance et sa trajectoire ------------

func TestVehiculeVieSansOccupant(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000, 42_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_500_000, -100.7, 53.7, vehChassisKnown)},
		Positions: []filmdec.BipedPosition{
			vehPos(700, 3_000_000, -100.7, 53.7),
			vehPos(700, 4_000_000, -100.7, 53.7),
		},
	}
	got, cov := buildVehicleTracks(scan, nil, OwnerReport{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1 : une vie recensee avec naissance DOIT sortir, "+
			"meme si personne ne l a conduite", len(got))
	}
	tr := got[0]
	if len(tr.Rides) != 0 {
		t.Errorf("episodes = %d, attendu 0 : aucun trou de bipede n a ete fourni", len(tr.Rides))
	}
	if tr.Family != "ghost" || tr.Chassis != "5b80c406" {
		t.Errorf("chassis = %q famille = %q, attendu 5b80c406 / ghost", tr.Chassis, tr.Family)
	}
	if tr.End != VehicleEndUnknown {
		t.Errorf("fin = %q, attendu %q : la destruction datee est REFUTEE, aucune autre cause ne "+
			"se publie", tr.End, VehicleEndUnknown)
	}
	// Naissance a 1,5 s (frame 5) ; derniere preuve de presence = dernier recensement a 22 s
	// (frame 210) ; premiere preuve d absence = image-cle a 42 s (frame 410).
	if tr.T0 != 5 || tr.T1 != 210 || tr.T1Max != 410 {
		t.Errorf("bornes = [%d, %d, %d], attendu [5, 210, 410]", tr.T0, tr.T1, tr.T1Max)
	}
	if tr.Spawn == nil || tr.Spawn.X != -100.7 {
		t.Fatalf("naissance = %+v : la position du record de creation DOIT etre publiee", tr.Spawn)
	}
	if tr.Spawn.H != nil {
		t.Error("cap de naissance publie : la feuille 4 du default-state ti=40 n est PAS etablie")
	}
	if cov.Published != 1 || cov.WithSpawn != 1 || cov.FamilyResolved != 1 {
		t.Errorf("couverture = %+v : la vie publiee doit se compter avec sa naissance et sa famille", cov)
	}
}

// --- Vie AVEC episodes board -> exit : bornes datees a la milliseconde ------------------------

func TestVehiculeEpisodeBoardExit(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	const bipedSlot = uint32(42)
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_500_000, 0, 0, vehChassisKnown)},
		Positions: []filmdec.BipedPosition{
			vehPos(700, 5_000_000, 0, 0),
			vehPos(700, 12_000_000, 0, 0),
			vehPos(700, 20_000_000, 0, 0),
		},
		Events: []filmdec.VehicleEvent{
			{Kind: filmdec.EventBipedBoardVehicle, TimestampUS: 5_200_000, OccupantPresent: true,
				OccupantInBand: true, OccupantSlot: bipedSlot, Seat: 0, SeatValid: true},
			{Kind: filmdec.EventUnitExitVehicle, TimestampUS: 16_800_000, OccupantPresent: true,
				OccupantInBand: true, OccupantSlot: bipedSlot, Seat: 0, SeatValid: true},
		},
	}
	// Le bipede est SUR le vehicule a 5 s, puis son flux s interrompt 12 s (le trou), puis il
	// reapparait a 17 s : la signature exacte d un occupant attache.
	bipeds := []filmdec.BipedPosition{
		vehPos(bipedSlot, 4_500_000, 0.4, 0),
		vehPos(bipedSlot, 5_000_000, 0.4, 0),
		vehPos(bipedSlot, 17_000_000, 3, 3),
	}
	own := OwnerReport{SlotXUID: map[uint32]uint64{bipedSlot: 2533274800000001}}
	got, cov := buildVehicleTracks(scan, bipeds, own, vehClock())
	if len(got) != 1 || len(got[0].Rides) != 1 {
		t.Fatalf("vies = %d, episodes = %v : un trou de position pres du vehicule DOIT rendre "+
			"un episode", len(got), got)
	}
	r := got[0].Rides[0]
	if r.Src != VehicleRideSrcEvent {
		t.Errorf("provenance = %q, attendu %q : les deux bornes tombent sur un evenement",
			r.Src, VehicleRideSrcEvent)
	}
	// Embarquement a 5,2 s -> frame 42 ; sortie a 16,8 s -> frame 158.
	if r.T0 != 42 || r.T1 != 158 {
		t.Errorf("bornes = [%d, %d], attendu [42, 158] : l evenement date a la milliseconde et "+
			"PRIME sur le bord du trou (5,0 s / 17,0 s)", r.T0, r.T1)
	}
	if r.XUID != "2533274800000001" || r.Slot != bipedSlot {
		t.Errorf("occupant = %q slot=%d : le pont slot -> xuid DOIT nommer l episode", r.XUID, r.Slot)
	}
	if r.Seat == nil || *r.Seat != 0 {
		t.Errorf("siege = %v, attendu 0 (conducteur) : le siege 0 est la valeur la plus frequente "+
			"et un pointeur existe pour qu elle ne soit pas effacee par omitempty", r.Seat)
	}
	if cov.Rides != 1 || cov.RidesNamed != 1 || cov.RidesFromEvent != 1 || cov.VehiclesRidden != 1 {
		t.Errorf("couverture = %+v : l episode doit se compter, nomme et borne par evenement", cov)
	}
}

// TestVehiculeEpisodeSansEvenement : sans liste d evenements, le TROU DE POSITION seul borne
// l episode — c est le repli mesure (86,3 % des trous portent leur sortie, mais 13,7 % non).
func TestVehiculeEpisodeSansEvenement(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	const bipedSlot = uint32(42)
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_500_000, 0, 0, vehChassisKnown)},
		Positions: []filmdec.BipedPosition{vehPos(700, 5_000_000, 0, 0), vehPos(700, 20_000_000, 0, 0)},
	}
	bipeds := []filmdec.BipedPosition{
		vehPos(bipedSlot, 5_000_000, 0.4, 0),
		vehPos(bipedSlot, 17_000_000, 3, 3),
	}
	got, _ := buildVehicleTracks(scan, bipeds, OwnerReport{}, vehClock())
	if len(got) != 1 || len(got[0].Rides) != 1 {
		t.Fatalf("un episode etait attendu, obtenu %v", got)
	}
	r := got[0].Rides[0]
	if r.Src != VehicleRideSrcGap {
		t.Errorf("provenance = %q, attendu %q", r.Src, VehicleRideSrcGap)
	}
	if r.T0 != 40 || r.T1 != 160 {
		t.Errorf("bornes = [%d, %d], attendu [40, 160] (bords du trou : 5,0 s et 17,0 s)", r.T0, r.T1)
	}
	if r.Seat != nil {
		t.Error("siege publie sans evenement : il n existe QUE dans la charge d un evenement")
	}
	if r.XUID != "" {
		t.Errorf("xuid = %q sans pont : un episode anonyme reste publie, mais il reste anonyme", r.XUID)
	}
}

// TestVehiculeTrouLoinDuVehicule : un trou dont le dernier point est HORS du rayon d embarquement
// n est pas un episode. Sans ce refus, tout joueur qui se deconnecte ou entre dans un ascenseur
// deviendrait le conducteur du vehicule le plus proche de la carte.
func TestVehiculeTrouLoinDuVehicule(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_500_000, 0, 0, vehChassisKnown)},
		Positions: []filmdec.BipedPosition{vehPos(700, 5_000_000, 0, 0), vehPos(700, 20_000_000, 0, 0)},
	}
	bipeds := []filmdec.BipedPosition{
		vehPos(42, 5_000_000, 12, 0), // 12 m du vehicule : bien au-dela des 1,5 m
		vehPos(42, 17_000_000, 12, 0),
	}
	got, cov := buildVehicleTracks(scan, bipeds, OwnerReport{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies = %d, attendu 1", len(got))
	}
	if len(got[0].Rides) != 0 || cov.Rides != 0 {
		t.Errorf("episodes = %d : un trou a 12 m du vehicule n est pas un embarquement", cov.Rides)
	}
}

// --- Famille inconnue : la vie sort, sans sprite, et le compteur le DIT ------------------------

func TestVehiculeFamilleInconnue(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 701, Gen: 0}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000}, key, []uint64{2_000_000}),
		Creations: []filmdec.EquipmentCreation{vehCreation(key, 1_800_000, 5, 5, vehChassisUnknown)},
		Positions: []filmdec.BipedPosition{vehPos(701, 3_000_000, 5, 5)},
	}
	got, cov := buildVehicleTracks(scan, nil, OwnerReport{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies = %d, attendu 1 : un chassis inconnu ne supprime pas le vehicule", len(got))
	}
	if got[0].Family != "" {
		t.Errorf("famille = %q, attendue vide : emprunter celle d un voisin dessinerait un "+
			"Warthog en Banshee", got[0].Family)
	}
	if got[0].Chassis != "deadbeef" {
		t.Errorf("chassis = %q, attendu deadbeef : le mot brut reste a cote de la famille", got[0].Chassis)
	}
	if cov.FamilyUnknown != 1 || cov.UnknownChassis["deadbeef"] != 1 {
		t.Errorf("couverture = %+v : le chassis non resolu DOIT se compter, sans quoi un film "+
			"entier sortirait sans sprite en silence", cov)
	}
	if cov.FamilyResolved != 0 {
		t.Errorf("famillesResolues = %d, attendu 0", cov.FamilyResolved)
	}
}

// --- Cap : la velocite i1 oriente, l arret reporte le dernier cap -----------------------------

func TestVehiculeCapParVelocite(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	// Direction +Y a 10 m/s : cap attendu 90 deg. Puis un echantillon a l ARRET : il doit garder
	// le cap precedent, pas en perdre l orientation.
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Positions: []filmdec.BipedPosition{
			vehPos(700, 3_000_000, 0, 0),
			vehMoving(t, 700, 4_000_000, 0, 5, [3]float32{0, 1, 0}, 10),
			vehMoving(t, 700, 5_000_000, 0, 10, [3]float32{0, 1, 0}, 0.2), // quasi a l arret
		},
	}
	got, cov := buildVehicleTracks(scan, nil, OwnerReport{}, vehClock())
	if len(got) != 1 || len(got[0].Samples) != 3 {
		t.Fatalf("echantillons = %v, attendu 3", got)
	}
	s := got[0].Samples
	if s[0].H != 0 {
		t.Errorf("cap[0] = %v : aucun cap n est connu avant le premier echantillon mobile", s[0].H)
	}
	if s[1].H < 89 || s[1].H > 91 {
		t.Errorf("cap[1] = %v, attendu ~90 deg (velocite +Y)", s[1].H)
	}
	if s[2].H != s[1].H {
		t.Errorf("cap[2] = %v, attendu %v : sous %g m/s la direction est du bruit, le dernier cap "+
			"connu est REPORTE", s[2].H, s[1].H, vehicleMinSpeedMPS)
	}
	if cov.WithHeading != 2 {
		t.Errorf("avecCap = %d, attendu 2", cov.WithHeading)
	}
}

// --- Deux vies d un meme slot : le recensement les separe, la fenetre les decoupe --------------

func TestVehiculeDeuxViesDunMemeSlot(t *testing.T) {
	k0 := filmdec.EquipmentLifeKey{Slot: 700, Gen: 0}
	k1 := filmdec.EquipmentLifeKey{Slot: 700, Gen: 1}
	kf := filmdec.WorldObjectKeyframes{
		Band:    map[uint32]bool{700: true},
		TimesUS: []uint64{2_000_000, 22_000_000, 42_000_000, 62_000_000},
		SeenUS: map[filmdec.EquipmentLifeKey][]uint64{
			k0: {2_000_000, 22_000_000},
			k1: {42_000_000, 62_000_000},
		},
	}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: kf,
		Creations: []filmdec.EquipmentCreation{
			vehCreation(k0, 1_500_000, 0, 0, vehChassisKnown),
			vehCreation(k1, 41_000_000, 9, 9, vehChassisKnown),
		},
		Positions: []filmdec.BipedPosition{
			vehPos(700, 3_000_000, 0, 0),
			vehPos(700, 50_000_000, 9, 9),
		},
	}
	got, cov := buildVehicleTracks(scan, nil, OwnerReport{}, vehClock())
	if len(got) != 2 {
		t.Fatalf("vies = %d, attendu 2 : le recensement separe deux generations d un meme slot", len(got))
	}
	if len(got[0].Samples) != 1 || len(got[1].Samples) != 1 {
		t.Errorf("echantillons = %d / %d, attendu 1 / 1 : la fenetre de chaque vie decoupe le "+
			"nuage, que `BipedPosition` ne sait pas separer (aucune generation)",
			len(got[0].Samples), len(got[1].Samples))
	}
	if got[0].Samples[0].X != 0 || got[1].Samples[0].X != 9 {
		t.Errorf("positions croisees entre les deux vies : %v / %v", got[0].Samples[0], got[1].Samples[0])
	}
	if cov.Lives != 2 || cov.Published != 2 {
		t.Errorf("couverture = %+v, attendu 2 vies recensees et publiees", cov)
	}
}

// --- Silences : « pas balaye » n est pas « aucun vehicule » ------------------------------------

func TestVehiculeNonBalayeNePublieRien(t *testing.T) {
	got, cov := buildVehicleTracks(VehicleScan{}, nil, OwnerReport{}, vehClock())
	if got != nil {
		t.Errorf("vies = %v, attendu nil : un film non balaye ne publie aucun vehicule", got)
	}
	if cov.Scanned {
		t.Error("couverture.Scanned vrai : c est LA distinction entre « pas lu » et « aucun vehicule »")
	}
}

// TestVehiculeVieSansPositionNiNaissance : une vie recensee dont rien ne donne la position n a
// rien a dessiner. Elle est ECARTEE et COMPTEE — jamais inventee a l origine du repere.
func TestVehiculeVieSansPositionNiNaissance(t *testing.T) {
	key := filmdec.EquipmentLifeKey{Slot: 702, Gen: 2}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000}, key, []uint64{2_000_000}),
	}
	got, cov := buildVehicleTracks(scan, nil, OwnerReport{}, vehClock())
	if len(got) != 0 {
		t.Fatalf("vies publiees = %v, attendu aucune", got)
	}
	if cov.Lives != 1 || cov.NoPosition != 1 {
		t.Errorf("couverture = %+v : la vie recensee et son refus doivent tous deux se compter", cov)
	}
}
