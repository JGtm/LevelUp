package replay

// positions_porte_vehicules_departage_test.go — LE DEPARTAGE DU REPLI F-2 (lot M1 des retours du
// rejeu, reprise apres revue adverse du 2026-09-24) : quand deux sejours de replication d une vie
// se contredisent au travers d un silence, lequel tombe — et quand F-2 REFUSE de trancher.
//
// Les trois regles qu on y epingle : la NAISSANCE de la vie est la voisine gauche du premier
// sejour ; un sejour plus long que `vehicleSejourAberrantMax` n est jamais ecarte ; sans preuve
// pour departager, les deux sejours restent publies et le refus se compte
// (`coverage.vehicles.silencesNonTranches`).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// departageVie assemble UNE vie de vehicule recensee de 2 s a la fin du film, nee (si `nee`) a
// 1,5 s en (5, 5), avec les echantillons donnes.
func departageVie(t *testing.T, nee bool, pos []grammar.BipedPosition) (VehicleTrack, VehicleCoverage) {
	t.Helper()
	key := types.EquipmentLifeKey{Slot: 770, Gen: 1}
	times := []uint64{2_000_000, 22_000_000, 42_000_000, 62_000_000, 82_000_000, 102_000_000}
	scan := VehicleScan{Scanned: true, Keyframes: vehKeyframes(times, key, times), Positions: pos}
	if nee {
		scan.Creations = []types.EquipmentCreation{vehCreation(key, 1_500_000, 5, 5, vehChassisKnown)}
	}
	got, cov, _ := buildVehicleTracks(scan, nil, IdentityRegistry{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1", len(got))
	}
	return got[0], cov
}

// aberrantEn rend `n` echantillons aberrants (la valeur de bits recurrente du parc, 250 m du point
// d apparition), un toutes les 100 ms a partir de `deUS`.
func aberrantEn(deUS uint64, n int) []grammar.BipedPosition {
	out := make([]grammar.BipedPosition, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, vehPos(770, deUS+uint64(i)*100_000, -201.16, 88.24))
	}
	return out
}

// verifierDepartage controle ce que la vie publie a l abscisse aberrante et au point vrai, et les
// deux comptes du repli.
func verifierDepartage(t *testing.T, tr VehicleTrack, cov VehicleCoverage, aberrants, vrais, ecartes, nonTranches int) {
	t.Helper()
	xs := porteXs(tr)
	if xs[-201.16] != aberrants || xs[5] != vrais {
		t.Errorf("echantillons publies %v : attendu %d aberrant(s) et %d vrai(s)", xs, aberrants, vrais)
	}
	if cov.EchantillonsAuTraversDUnSilence != ecartes || cov.SilencesNonTranches != nonTranches {
		t.Errorf("echantillonsAuTraversDUnSilence = %d, silencesNonTranches = %d ; attendu %d et %d",
			cov.EchantillonsAuTraversDUnSilence, cov.SilencesNonTranches, ecartes, nonTranches)
	}
}

// L ABERRANT EN TETE, PUIS UN SEUL ECHANTILLON VRAI (vehicule gare, replique une fois par silence) :
// a taille egale, c est la NAISSANCE qui departage — elle contredit l aberrant. Avant la reprise,
// « a egalite encore, le plus tardif » ecartait le VRAI et publiait l aberrant.
func TestDepartageAberrantEnTeteSuiviDUnVrai(t *testing.T) {
	pos := append(aberrantEn(3_000_000, 1), vehPos(770, 30_000_000, 5, 5))
	tr, cov := departageVie(t, true, pos)
	verifierDepartage(t, tr, cov, 0, 1, 1, 0)
}

// L ABERRANT EN TETE, PUIS DEUX ECHANTILLONS VRAIS.
func TestDepartageAberrantEnTeteSuiviDeDeuxVrais(t *testing.T) {
	pos := append(aberrantEn(3_000_000, 1), vehPos(770, 30_000_000, 5, 5), vehPos(770, 30_100_000, 5, 5))
	tr, cov := departageVie(t, true, pos)
	verifierDepartage(t, tr, cov, 0, 2, 1, 0)
}

// LE VRAI EN TETE (soutenu par la naissance), PUIS UN ABERRANT SANS VOISIN A DROITE : l aberrant tombe.
func TestDepartageVraiEnTeteSuiviDUnAberrant(t *testing.T) {
	pos := append([]grammar.BipedPosition{vehPos(770, 3_000_000, 5, 5)}, aberrantEn(30_000_000, 1)...)
	tr, cov := departageVie(t, true, pos)
	verifierDepartage(t, tr, cov, 0, 1, 1, 0)
}

// LA SUPPORT PRIME SUR LA TAILLE : trois echantillons aberrants identiques en tete (la Wraith de
// 0a44c6cc), puis UN vrai. « Le plus court tombe » ecartait le vrai ; la naissance contredit
// l aberrant, qui tombe.
func TestDepartageLaNaissancePrimeSurLaTaille(t *testing.T) {
	pos := append(aberrantEn(3_000_000, 3), vehPos(770, 30_000_000, 5, 5))
	tr, cov := departageVie(t, true, pos)
	verifierDepartage(t, tr, cov, 0, 1, 3, 0)
}

// SANS NAISSANCE LUE, A TAILLE EGALE, RIEN NE DEPARTAGE : F-2 refuse de trancher, les deux
// echantillons restent publies, et le refus se compte.
func TestDepartageSansPreuveRefuseDeTrancher(t *testing.T) {
	pos := append(aberrantEn(3_000_000, 1), vehPos(770, 30_000_000, 5, 5))
	tr, cov := departageVie(t, false, pos)
	verifierDepartage(t, tr, cov, 1, 1, 0, 1)
}

// LA BORNE : une vraie trajectoire reprise apres un silence de plus de `lifeGapUS` a plus de 2 m
// n est jamais ecartee en bloc — un faux en-tete fait 1 ou 2 echantillons au parc. 31 echantillons
// en (10, 10), 6 s de silence, puis 21 en (60, 10) : rien ne tombe, le desaccord se compte.
func TestDepartageSejourLongJamaisEcarte(t *testing.T) {
	pos := porteVehSejour(3_000_000, 6_000_000, 10, 10)
	pos = append(pos, porteVehSejour(12_000_000, 14_000_000, 60, 10)...)
	tr, cov := departageVie(t, true, pos)
	if xs := porteXs(tr); xs[10] != 31 || xs[60] != 21 {
		t.Errorf("echantillons publies %v : attendu les 31 et les 21", xs)
	}
	if cov.EchantillonsAuTraversDUnSilence != 0 || cov.SilencesNonTranches != 1 {
		t.Errorf("echantillonsAuTraversDUnSilence = %d, silencesNonTranches = %d ; attendu 0 et 1",
			cov.EchantillonsAuTraversDUnSilence, cov.SilencesNonTranches)
	}
}

// LA BORNE VAUT DES DEUX COTES : le premier sejour (31 echantillons a 195 m de la naissance, a
// 130 m/s) est celui que sa voisine contredit, mais il est trop long pour etre un faux en-tete —
// rien ne tombe, le desaccord se compte.
func TestDepartagePremierSejourLongJamaisEcarte(t *testing.T) {
	pos := porteVehSejour(3_000_000, 6_000_000, 200, 5)
	pos = append(pos, vehPos(770, 30_000_000, 5, 5))
	tr, cov := departageVie(t, true, pos)
	if xs := porteXs(tr); xs[200] != 31 || xs[5] != 1 {
		t.Errorf("echantillons publies %v : attendu les 31 et le dernier", xs)
	}
	if cov.EchantillonsAuTraversDUnSilence != 0 || cov.SilencesNonTranches != 1 {
		t.Errorf("echantillonsAuTraversDUnSilence = %d, silencesNonTranches = %d ; attendu 0 et 1",
			cov.EchantillonsAuTraversDUnSilence, cov.SilencesNonTranches)
	}
}

// UN RECORD DE CREATION POSTERIEUR A LA FIN DE LA VIE n est pas sa naissance : c est celle d un
// objet ulterieur du meme (slot, gen) — la generation ne fait que 2 bits. La vie finit a 42 s (la
// premiere image-cle qui ne la recense plus) ; le record de 60 s ne lui donne ni point
// d apparition ni chassis.
func TestNaissancePosterieureALaFinDeLaVieIgnoree(t *testing.T) {
	key := types.EquipmentLifeKey{Slot: 770, Gen: 1}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes([]uint64{2_000_000, 22_000_000, 42_000_000}, key, []uint64{2_000_000, 22_000_000}),
		Creations: []types.EquipmentCreation{vehCreation(key, 60_000_000, 40, 40, vehChassisKnown)},
		Positions: []grammar.BipedPosition{vehPos(770, 3_000_000, 5, 5), vehPos(770, 3_100_000, 5, 5)},
	}
	got, _, _ := buildVehicleTracks(scan, nil, IdentityRegistry{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1", len(got))
	}
	if got[0].Spawn != nil || got[0].Chassis != "" {
		t.Errorf("naissance %+v, chassis %q : un record posterieur a la fin de la vie n est pas sa naissance",
			got[0].Spawn, got[0].Chassis)
	}
}
