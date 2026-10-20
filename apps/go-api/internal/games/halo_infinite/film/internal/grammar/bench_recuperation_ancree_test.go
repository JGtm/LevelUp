package grammar

// bench_recuperation_ancree_test.go — LE BANC DES NEUF LECTEURS ANCRES (lot 2.4 du plan de l etape 2
// de la representation intermediaire).
//
// Il mesure ce que la cuisson paie pour les records bipedes ancres : les positions, puis les huit
// balayages de canal qui ancrent avec les memes parametres (changements d arme, deltas d inventaire,
// rangs, changements d equipement, camouflage, grappin, impulsions et charges de capacite). Matiere :
// la mini-bobine contigue de `killsource` (`bobineFamilles`), chargee une fois ; un `FilmContext`
// NEUF par tour, parce que le contexte memorise son ancrage et qu un tour qui relirait celui du tour
// precedent ne mesurerait plus rien.
//
// MESURE A/B, comme `bench_balayage_bits_test.go` (binaires alternes, au moins dix fois chacun,
// depuis le repertoire du paquet) :
//
//	avant.test.exe -test.run '^$' -test.bench '^BenchmarkRecuperationAncree$' -test.count 1 -test.benchtime 20x

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// BenchmarkRecuperationAncree joue, par tour, les neuf lecteurs ancres d une cuisson sur la
// mini-bobine.
func BenchmarkRecuperationAncree(b *testing.B) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		b.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	wr := profile.QuantRangeCEBiped() // bornes MESUREES du film 000d5950, comme le golden des familles
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	opt.CaptureDirs = true
	for b.Loop() {
		lireLesNeufLecteursAncres(b, NewFilmContext(film), opt)
	}
}

// lireLesNeufLecteursAncres fait UN tour. Une erreur est fatale : un lecteur qui refuse la bobine
// ne mesurerait plus son ancrage, et le banc rendrait un temps flatteur.
func lireLesNeufLecteursAncres(b *testing.B, fc *FilmContext, opt ScanFilmOptions) {
	b.Helper()
	echec := func(nom string, err error) {
		if err != nil {
			b.Fatalf("%s : %v", nom, err)
		}
	}
	_, err := ScanBipedPositions(fc, opt)
	echec("ScanBipedPositions", err)
	_, _, err = ScanHeldWeaponChanges(fc, func(uint32, uint64) (SpawnState, bool) { return SpawnState{}, false })
	echec("ScanHeldWeaponChanges", err)
	_, _, err = ScanInventoryDeltas(fc)
	echec("ScanInventoryDeltas", err)
	_, _, err = ScanAbilityRanks(fc)
	echec("ScanAbilityRanks", err)
	_, _, err = ScanEquipmentChanges(fc, func(uint32) (uint64, bool) { return 0, false })
	echec("ScanEquipmentChanges", err)
	_, _, err = ScanCamoStates(fc)
	echec("ScanCamoStates", err)
	_, _, err = ScanGrappleReads(fc)
	echec("ScanGrappleReads", err)
	_, _, err = ScanAbilityImpulses(fc)
	echec("ScanAbilityImpulses", err)
	_, _, err = ScanAbilityCharges(fc)
	echec("ScanAbilityCharges", err)
}
