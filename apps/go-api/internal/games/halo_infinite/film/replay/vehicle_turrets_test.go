package replay

// vehicle_turrets_test.go — LES PIECES MONTEES POSEES SUR LEUR PORTEUR (schema 69, lot M4a des
// retours du rejeu 2026-09-23), testees SANS film : `vehicle_turrets.go` est pur. Les gabarits
// reproduisent la disposition MESUREE au parc (la piece prend le slot juste avant son chassis),
// jamais un match particulier.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// vtLAAG / vtRoquettes / vtFalconGL / vtFalconLMG : les mots d identite des pieces, ecrits comme
// le document les publie (`formatChassisID`).
const (
	vtLAAG      = "dd7f9102"
	vtRoquettes = "bcfb852f"
	vtFalconGL  = "1a043c29"
	vtFalconLMG = "f4c45d71"
)

// vtPiece fabrique une piece montee sans echantillon, nee a (500, 500) — loin de son porteur.
func vtPiece(slot uint32, chassis string, rides ...VehicleRide) VehicleTrack {
	return VehicleTrack{
		Slot: slot, Gen: 1, Chassis: chassis, T0: 0, T1: 90, T1Max: 100,
		Spawn: &VehicleSpawn{X: 500, Y: 500}, Rides: rides,
	}
}

// vtChassis fabrique un chassis qui roule de (0, 0) a (90, 0).
func vtChassis(slot uint32, family string, rides ...VehicleRide) VehicleTrack {
	return VehicleTrack{
		Slot: slot, Gen: 2, Family: family, T0: 0, T1: 90, T1Max: 100,
		Samples: []VehicleSample{{T: 0, X: 0, Y: 0}, {T: 90, X: 90, Y: 0}}, Rides: rides,
	}
}

func vtRide(slot uint32, t0, t1 int) VehicleRide {
	seat := 0
	return VehicleRide{T0: t0, T1: t1, Slot: slot, Seat: &seat, Src: VehicleRideSrcFilm}
}

// TestTourellePoseeSurLeChassisVoisin — la LAAG (slot 100) est posee sur le Warthog du slot 101 :
// elle est marquee piece, nomme son porteur, et son artilleur passe a bord du Warthog SANS siege
// (le siege 0 lu etait celui de la tourelle : le garder ferait de l artilleur le conducteur).
func TestTourellePoseeSurLeChassisVoisin(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtLAAG, vtRide(20, 10, 40)),
		vtChassis(101, familleWarthog, vtRide(21, 5, 80)),
	}
	fb := fallback.NouveauCompteur()
	tally := poseTurretsOnCarriers(tracks, fb)
	piece, porteur := tracks[0], tracks[1]
	if piece.Part != VehiclePartTurret {
		t.Fatalf("piece.Part = %q, attendu %q", piece.Part, VehiclePartTurret)
	}
	if piece.Carrier == nil || *piece.Carrier != (VehicleLifeRef{Slot: 101, Gen: 2}) {
		t.Fatalf("porteur = %+v, attendu {101 2}", piece.Carrier)
	}
	if len(piece.Rides) != 0 || len(porteur.Rides) != 2 {
		t.Fatalf("episodes piece/porteur = %d/%d, attendu 0/2", len(piece.Rides), len(porteur.Rides))
	}
	artilleur := porteur.Rides[1]
	if artilleur.Slot != 20 || artilleur.Seat != nil ||
		artilleur.Turret == nil || *artilleur.Turret != (VehicleLifeRef{Slot: 100, Gen: 1}) {
		t.Errorf("artilleur reporte = %+v, attendu slot 20, sans siege, tourelle {100 1}", artilleur)
	}
	if tally != (turretTally{turrets: 1, onCarrier: 1, rides: 1}) {
		t.Errorf("bilan = %+v", tally)
	}
	if fb.Rapport()[0].Nom != fallback.NomTourellePorteurVoisinDeSlot {
		t.Errorf("repli non compte : %+v", fb.Rapport())
	}
}

// TestTourelleNommeLaVarianteDuPorteur — le lance-roquettes fait le Rockethog. La FAMILLE du
// porteur ne change pas (son moteur, son explosion) : seule la variante (le sprite) est nommee.
func TestTourelleNommeLaVarianteDuPorteur(t *testing.T) {
	tracks := []VehicleTrack{vtPiece(100, vtRoquettes), vtChassis(101, familleWarthog)}
	poseTurretsOnCarriers(tracks, nil)
	if tracks[1].Variant != familleRockethog || tracks[1].Family != familleWarthog {
		t.Errorf("porteur = famille %q variante %q, attendu warthog / rockethog",
			tracks[1].Family, tracks[1].Variant)
	}
}

// TestTourelleVoisinDUneAutreFamilleRefuse — un voisin de slot d une AUTRE famille n est pas le
// porteur : la piece reste publiee, marquee, SANS porteur, et le repli ne se compte pas.
func TestTourelleVoisinDUneAutreFamilleRefuse(t *testing.T) {
	tracks := []VehicleTrack{vtPiece(100, vtLAAG, vtRide(20, 10, 40)), vtChassis(101, familleMongoose)}
	fb := fallback.NouveauCompteur()
	tally := poseTurretsOnCarriers(tracks, fb)
	if tracks[0].Carrier != nil || tracks[0].Part != VehiclePartTurret || len(tracks[0].Rides) != 1 {
		t.Errorf("piece = %+v, attendu marquee, sans porteur, episode garde", tracks[0])
	}
	if tally.onCarrier != 0 || len(fb.Rapport()) != 0 {
		t.Errorf("bilan %+v, replis %+v : rien ne devait etre decide", tally, fb.Rapport())
	}
}

// TestTourelleHorsDeLaFenetreDuVoisinRefusee — meme famille, mais les deux vies ne coexistent pas.
func TestTourelleHorsDeLaFenetreDuVoisinRefusee(t *testing.T) {
	porteur := vtChassis(101, familleWarthog)
	porteur.T0, porteur.T1, porteur.T1Max = 200, 300, 310
	tracks := []VehicleTrack{vtPiece(100, vtLAAG), porteur}
	poseTurretsOnCarriers(tracks, nil)
	if tracks[0].Carrier != nil {
		t.Errorf("porteur = %+v, attendu aucun (fenetres disjointes)", tracks[0].Carrier)
	}
}

// TestTourelleDuFalconPorteurASlotPlusDeux — la chaine mesuree `[1a043c29][f4c45d71][Falcon]` :
// la premiere piece trouve son porteur en +2, par-dessus sa jumelle. Le Falcon est NON PILOTABLE
// (decision du 2026-09-02) : ses artilleurs restent sur la piece, comptes, jamais affirmes a bord.
func TestTourelleDuFalconPorteurASlotPlusDeux(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtFalconGL, vtRide(20, 10, 40)),
		vtPiece(101, vtFalconLMG),
		vtChassis(102, familleFalcon),
	}
	tally := poseTurretsOnCarriers(tracks, nil)
	for i := 0; i < 2; i++ {
		if tracks[i].Carrier == nil || tracks[i].Carrier.Slot != 102 {
			t.Errorf("piece %d : porteur %+v, attendu 102", i, tracks[i].Carrier)
		}
	}
	if len(tracks[0].Rides) != 1 || len(tracks[2].Rides) != 0 || tally.dropped != 1 {
		t.Errorf("episodes piece/Falcon = %d/%d (bilan %+v) : un Falcon ne porte aucun occupant",
			len(tracks[0].Rides), len(tracks[2].Rides), tally)
	}
}

// TestTourelleOccupantDejaABordNonDouble — le conducteur que le trou de position designe AUSSI
// pour la piece nee au meme point n est pas reporte une seconde fois sur son vehicule.
func TestTourelleOccupantDejaABordNonDouble(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtLAAG, vtRide(21, 10, 30)),
		vtChassis(101, familleWarthog, vtRide(21, 5, 80)),
	}
	tally := poseTurretsOnCarriers(tracks, nil)
	if len(tracks[1].Rides) != 1 || len(tracks[0].Rides) != 1 || tally.dropped != 1 {
		t.Errorf("episodes porteur/piece = %d/%d, attendu 1/1", len(tracks[1].Rides), len(tracks[0].Rides))
	}
}

// TestPieceMonteeHorsDesChassisInconnus — une piece montee n est pas un chassis INCONNU : elle ne
// compte pas dans `familyUnknown` et ne declenche pas le marqueur neutre.
func TestPieceMonteeHorsDesChassisInconnus(t *testing.T) {
	tracks := []VehicleTrack{vtPiece(100, vtLAAG), vtChassis(101, familleWarthog)}
	tracks[1].Chassis = "fe32c0f4"
	poseTurretsOnCarriers(tracks, nil)
	cov := VehicleCoverage{UnknownChassis: map[string]int{}}
	fb := fallback.NouveauCompteur()
	tallyVehicleCoverage(tracks, &cov, fb)
	if cov.FamilyUnknown != 0 || len(cov.UnknownChassis) != 0 || len(fb.Rapport()) != 0 {
		t.Errorf("inconnus = %d %v, replis %+v : la piece est nommee", cov.FamilyUnknown,
			cov.UnknownChassis, fb.Rapport())
	}
}

// TestArtilleurReporteNEstPasUneAmbiguite — le conducteur et l artilleur reporte d une tourelle se
// chevauchent sur le porteur : ce n est pas une ambiguite (compteur `ambiguous` inchange).
func TestArtilleurReporteNEstPasUneAmbiguite(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtLAAG, vtRide(20, 10, 40)),
		vtChassis(101, familleWarthog, vtRide(21, 5, 80)),
	}
	poseTurretsOnCarriers(tracks, nil)
	cov := VehicleCoverage{UnknownChassis: map[string]int{}}
	tallyVehicleCoverage(tracks, &cov, nil)
	if cov.Ambiguous != 0 || cov.Rides != 2 {
		t.Errorf("ambigus = %d, episodes = %d : attendu 0 et 2", cov.Ambiguous, cov.Rides)
	}
}
