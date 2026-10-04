package grammar

// bench_objets_du_monde_test.go — LE BANC DES BALAYAGES D OBJETS DU MONDE DE LA CUISSON (lot 2.5 du
// plan de l etape 2 de la representation intermediaire).
//
// Il joue, dans l ordre de la cuisson, ce que les poses, les socles, les vehicules et les
// projectiles demandent : les pistes de l equipement, des armes au sol et des projectiles (celles
// de l equipement deux fois), et les creations de l equipement, des armes au sol et des vehicules
// (celles de l equipement deux fois). Matiere : la mini-bobine contigue de `killsource`
// (`bobineFamilles`), chargee une fois ; un `FilmContext` NEUF par tour, parce que le contexte
// memorise ce qu il releve.
//
// MESURE A/B, comme `bench_balayage_bits_test.go` (binaires alternes, au moins dix fois chacun,
// depuis le repertoire du paquet) :
//
//	avant.test.exe -test.run '^$' -test.bench '^BenchmarkObjetsDuMonde$' -test.count 1 -test.benchtime 20x

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// BenchmarkObjetsDuMonde joue, par tour, les balayages d objets du monde d une cuisson sur la
// mini-bobine.
func BenchmarkObjetsDuMonde(b *testing.B) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		b.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	wr := profile.QuantRangeCEBiped() // bornes MESUREES du film 000d5950, comme le golden des familles
	for b.Loop() {
		balayerLesObjetsDuMonde(b, contexteDeBobine(film), &wr)
	}
}

// balayerLesObjetsDuMonde fait UN tour. Une erreur est fatale : un balayage qui refuse la bobine ne
// mesurerait plus rien, et le banc rendrait un temps flatteur.
func balayerLesObjetsDuMonde(b *testing.B, fc *FilmContext, wr *profile.Vec3Range) {
	b.Helper()
	echec := func(nom string, err error) {
		if err != nil {
			b.Fatalf("%s : %v", nom, err)
		}
	}
	equipement := worldObjectSlotBand(fc, EquipmentTypeIndex)
	armes := worldObjectSlotBand(fc, GroundWeaponTypeIndex)
	vehicules := worldObjectSlotBand(fc, VehicleTypeIndex)
	_, err := ScanWorldObjectsForBand(fc, wr, equipement)
	echec("pistes de l equipement (poses)", err)
	_, _, err = ScanEquipmentCreationsForBand(fc, wr, equipement)
	echec("creations de l equipement (poses)", err)
	_, _, err = ScanGroundWeaponCreationsForBand(fc, wr, armes)
	echec("creations des armes au sol", err)
	_, err = ScanWorldObjectsForBand(fc, wr, armes)
	echec("pistes des armes au sol", err)
	_, _, err = ScanEquipmentCreationsForBand(fc, wr, equipement)
	echec("creations de l equipement (socles)", err)
	_, err = ScanWorldObjectsForBand(fc, wr, equipement)
	echec("pistes de l equipement (socles)", err)
	if len(vehicules) > 0 {
		_, _, err = ScanVehicleCreationsForBand(fc, wr, vehicules)
		echec("creations des vehicules", err)
	}
	_, err = ScanWorldObjects(fc, wr, ProjectileTypeIndex)
	echec("pistes des projectiles", err)
}
