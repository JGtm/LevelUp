package replay

// vehicle_turrets_test.go — LES PIECES MONTEES POSEES SUR LEUR PORTEUR (schema 69, lot M4a des
// retours du rejeu 2026-09-23), testees SANS film : `vehicle_turrets.go` est pur. Les gabarits
// reproduisent la disposition MESUREE au parc (la piece prend le slot juste avant son chassis),
// jamais un match particulier.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
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
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
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
	poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
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
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
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
	poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	if tracks[0].Carrier != nil {
		t.Errorf("porteur = %+v, attendu aucun (fenetres disjointes)", tracks[0].Carrier)
	}
}

// TestTourelleDuFalconPorteurASlotPlusDeux — la chaine mesuree `[1a043c29][f4c45d71][Falcon]` :
// la premiere piece trouve son porteur en +2, par-dessus sa jumelle. Le Falcon est PILOTABLE
// depuis la decision du 2026-09-24 (« le Falcon ca depend ») : son artilleur passe a bord, SANS
// siege et avec la reference de sa piece — il se dessine sur le Falcon, comme celui d un Warthog.
func TestTourelleDuFalconPorteurASlotPlusDeux(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtFalconGL, vtRide(20, 10, 40)),
		vtPiece(101, vtFalconLMG),
		vtChassis(102, familleFalcon),
	}
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	for i := 0; i < 2; i++ {
		if tracks[i].Carrier == nil || tracks[i].Carrier.Slot != 102 {
			t.Errorf("piece %d : porteur %+v, attendu 102", i, tracks[i].Carrier)
		}
	}
	if len(tracks[0].Rides) != 0 || len(tracks[2].Rides) != 1 || tally.rides != 1 || tally.kept.total() != 0 {
		t.Fatalf("episodes piece/Falcon = %d/%d (bilan %+v) : l artilleur du Falcon passe a bord",
			len(tracks[0].Rides), len(tracks[2].Rides), tally)
	}
	r := tracks[2].Rides[0]
	if r.Slot != 20 || r.Seat != nil || r.Turret == nil || *r.Turret != (VehicleLifeRef{Slot: 100, Gen: 1}) {
		t.Errorf("episode reporte = %+v (turret %+v), attendu l artilleur 20 sans siege, piece {100 1}", r, r.Turret)
	}
}

// vtClock : l horloge des gabarits — origine 0, une frame = 100 ms, 200 frames.
func vtClock() replayClock { return replayClock{origin: 0, step: 100_000, frames: 200} }

// vtAnchors : le nuage des bipedes des gabarits, un point par (slot, frame, x, y).
func vtAnchors(pts ...[4]float32) vehicleBoardingAnchors {
	a := vehicleBoardingAnchors{bySlot: map[uint32][]grammar.BipedPosition{}, clock: vtClock()}
	for _, p := range pts {
		s := uint32(p[0])
		a.bySlot[s] = append(a.bySlot[s], grammar.BipedPosition{
			Slot: s, TimestampUS: uint64(p[1]) * 100_000, X: p[2], Y: p[3], HasWorld: true})
	}
	return a
}

// vtRepli : un episode de REPLI (`proximity`), sans siege — la seule source que la garde juge.
func vtRepli(slot uint32, t0, t1 int) VehicleRide {
	return VehicleRide{T0: t0, T1: t1, Slot: slot, Src: VehicleRideSrcProximity}
}

// TestTourelleMonteeLoinDuPorteurEcartee — REVUE ADVERSE DU LOT M7b (RR-M7b-01). Un episode de
// REPLI sur une piece n est reporte sur son porteur que si l occupant a ete vu PRES de lui a la
// montee ; sinon il est ECARTE (ni sur le porteur, ni sur la piece) et le repli nomme est compte. Le gabarit reproduit la forme mesuree au parc (fin d une vie loin du vehicule, point
// d une vie anterieure du slot, occupant pas encore ne), jamais un match : le Warthog roule de
// (0, 0) a (90, 0), le bipede est vu a la frame du debut de l episode.
func TestTourelleMonteeLoinDuPorteurEcartee(t *testing.T) {
	for _, c := range []struct {
		nom     string
		ride    VehicleRide
		anchors vehicleBoardingAnchors
		reporte bool
	}{
		{"repli, dernier point a 1 m du porteur", vtRepli(20, 10, 40), vtAnchors([4]float32{20, 10, 10, 1}), true},
		{"repli, dernier point a 2 s pile", vtRepli(20, 30, 40), vtAnchors([4]float32{20, 10, 10, 1}), true},
		{"repli, dernier point a 35 m (fin d une vie ailleurs)", vtRepli(20, 10, 40), vtAnchors([4]float32{20, 10, 10, 35}), false},
		{"repli, dernier point PERIME (vie anterieure du slot)", vtRepli(20, 40, 60), vtAnchors([4]float32{20, 10, 10, 0}), false},
		{"repli, aucun point (occupant pas encore ne)", vtRepli(20, 10, 40), vtAnchors([4]float32{20, 50, 50, 0}), false},
		{"lu dans le film, loin : jamais juge", vtRide(20, 10, 40), vtAnchors([4]float32{20, 10, 10, 35}), true},
	} {
		t.Run(c.nom, func(t *testing.T) {
			tracks := []VehicleTrack{vtPiece(100, vtLAAG, c.ride), vtChassis(101, familleWarthog)}
			fb := fallback.NouveauCompteur()
			tally := poseTurretsOnCarriers(tracks, c.anchors, fb)
			porteur, piece := len(tracks[1].Rides), len(tracks[0].Rides)
			refus, attendu := 0, 1
			if !c.reporte {
				refus, attendu = 1, 0
			}
			if porteur != attendu || piece != 0 || tally.rides != attendu {
				t.Fatalf("piece/porteur = %d/%d (bilan %+v), attendu 0/%d : ecarte, jamais garde",
					piece, porteur, tally, attendu)
			}
			if tally.kept.total() != 0 || fbCount(fb, fallback.NomTourelleMonteeLoinDuPorteur) != refus {
				t.Errorf("bilan %+v, replis %+v : attendu %d declenchement, aucun episode garde", tally.kept, fb.Rapport(), refus)
			}
		})
	}
}

// fbCount rend le nombre de declenchements d un repli dans un compteur.
func fbCount(fb *fallback.Compteur, nom fallback.Nom) int {
	for _, e := range fb.Rapport() {
		if e.Nom == nom {
			return e.Declenchements
		}
	}
	return 0
}

// TestTourelleOccupantDejaABordNonDouble — le conducteur que le trou de position designe AUSSI
// pour la piece nee au meme point n est pas reporte une seconde fois sur son vehicule.
func TestTourelleOccupantDejaABordNonDouble(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtLAAG, vtRide(21, 10, 30)),
		vtChassis(101, familleWarthog, vtRide(21, 5, 80)),
	}
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	if len(tracks[1].Rides) != 1 || len(tracks[0].Rides) != 1 || tally.kept.total() != 1 {
		t.Errorf("episodes porteur/piece = %d/%d, attendu 1/1", len(tracks[1].Rides), len(tracks[0].Rides))
	}
}

// TestPieceMonteeHorsDesChassisInconnus — une piece montee n est pas un chassis INCONNU : elle ne
// compte pas dans `familyUnknown` et ne declenche pas le marqueur neutre.
func TestPieceMonteeHorsDesChassisInconnus(t *testing.T) {
	tracks := []VehicleTrack{vtPiece(100, vtLAAG), vtChassis(101, familleWarthog)}
	tracks[1].Chassis = "fe32c0f4"
	poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
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
	poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	cov := VehicleCoverage{UnknownChassis: map[string]int{}}
	tallyVehicleCoverage(tracks, &cov, nil)
	if cov.Ambiguous != 0 || cov.Rides != 2 {
		t.Errorf("ambigus = %d, episodes = %d : attendu 0 et 2", cov.Ambiguous, cov.Rides)
	}
}

// TestTourelleChangementDeSiegeBornesJointivesReporte — revue adverse du lot M4a (F2). Le meme
// joueur CONDUIT le Warthog jusqu a la frame 40 puis passe a la LAAG a la frame 40 : ses deux
// episodes se TOUCHENT a une frame, ils ne se recouvrent pas. L episode d artilleur est reporte
// sur le porteur (sinon l artilleur restait sur la piece cachee, sans pion ni cone).
func TestTourelleChangementDeSiegeBornesJointivesReporte(t *testing.T) {
	tracks := []VehicleTrack{
		vtPiece(100, vtLAAG, vtRide(21, 40, 70)),
		vtChassis(101, familleWarthog, vtRide(21, 5, 40)),
	}
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	if len(tracks[0].Rides) != 0 || len(tracks[1].Rides) != 2 || tally.rides != 1 {
		t.Fatalf("episodes piece/porteur = %d/%d (bilan %+v), attendu 0/2 : un changement de siege "+
			"n est pas un doublon", len(tracks[0].Rides), len(tracks[1].Rides), tally)
	}
	if tally.kept.total() != 0 {
		t.Errorf("episodes gardes = %+v, attendu aucun", tally.kept)
	}
}

// TestTourelleRefusVentilesParRaison — revue adverse du lot M4a (F4) : les refus qui GARDENT un
// episode sur sa piece sont comptes chacun sous leur raison par `poseTurretsOnCarriers`, et
// `turretRidesDropped` est leur somme. Depuis la reprise du lot M7b (2026-09-24) : le porteur non
// pilotable n a plus de branche (`turretRidesNotRideable` = 0), et la montee a bord non vue pres
// du porteur ECARTE l episode — elle ne compte pas parmi les episodes gardes.
func TestTourelleRefusVentilesParRaison(t *testing.T) {
	porteurCourt := vtChassis(101, familleWarthog, vtRide(21, 5, 30))
	porteurCourt.T1, porteurCourt.T1Max = 50, 50
	tracks := []VehicleTrack{
		// hors fenetre (60-80 apres la fin du porteur a 50), deja a bord (10-20 dans 5-30) et
		// monte loin du porteur (repli 35-45, occupant vu a 40 m a la frame 35).
		vtPiece(100, vtLAAG, vtRide(20, 60, 80), vtRide(21, 10, 20), vtRepli(22, 35, 45)),
		porteurCourt,
	}
	tally := poseTurretsOnCarriers(tracks, vtAnchors([4]float32{22, 35, 35, 40}), nil)
	if tally.kept != (turretRidesKept{outOfWindow: 1, alreadyAboard: 1}) || len(tracks[0].Rides) != 2 {
		t.Fatalf("refus = %+v, episodes gardes = %d : attendu un par raison, l episode monte loin ecarte",
			tally.kept, len(tracks[0].Rides))
	}
	var cov VehicleCoverage
	tally.applyTo(&cov)
	if cov.TurretRidesDropped != 2 || cov.TurretRidesNotRideable != 0 ||
		cov.TurretRidesOutOfWindow != 1 || cov.TurretRidesAlreadyAboard != 1 {
		t.Errorf("couverture = %+v", cov)
	}
}

// TestTourelleVoisinPasNeAvecLaPieceRefuse — revue adverse du lot M4a (F5). Le voisin de slot est
// de la bonne famille et present au meme moment, mais il n est pas NE avec la piece (autre point,
// ou autre instant) : ce n est pas son porteur. La piece reste sans porteur, le refus est COMPTE,
// et le repli du voisinage ne se declenche pas.
func TestTourelleVoisinPasNeAvecLaPieceRefuse(t *testing.T) {
	for _, c := range []struct {
		nom      string
		t0       int
		spawn    *VehicleSpawn
		refusDit bool
	}{
		{"meme instant, autre point", 0, &VehicleSpawn{X: 0, Y: 0}, true},
		{"meme point, autre instant", 30, &VehicleSpawn{X: 500, Y: 500}, true},
		{"ne ensemble, naissance du porteur non lue", 1, nil, false},
		{"ne ensemble", 0, &VehicleSpawn{X: 500.4, Y: 500}, false},
	} {
		t.Run(c.nom, func(t *testing.T) {
			porteur := vtChassis(101, familleWarthog)
			porteur.T0, porteur.Spawn = c.t0, c.spawn
			tracks := []VehicleTrack{vtPiece(100, vtLAAG), porteur}
			fb := fallback.NouveauCompteur()
			tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
			if c.refusDit {
				if tracks[0].Carrier != nil || tally.birthMismatch != 1 || len(fb.Rapport()) != 0 {
					t.Errorf("porteur %+v, refus %d, replis %+v : attendu aucun porteur, 1 refus compte",
						tracks[0].Carrier, tally.birthMismatch, fb.Rapport())
				}
				return
			}
			if tracks[0].Carrier == nil || tally.birthMismatch != 0 {
				t.Errorf("porteur %+v, refus %d : attendu pose, aucun refus", tracks[0].Carrier,
					tally.birthMismatch)
			}
		})
	}
}
