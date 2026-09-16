package replay

// vehicle_end_test.go — LA FIN DE VIE D UN VEHICULE : ce que le film ecrit, ce que l assemblage
// en publie, et ce qu il refuse d affirmer (lot 1.9.10).
//
// LE TEST DE MUTATION DU LOT est `TestFinDeVieIgnorerLeDeadStateRendUneFinFausse` : il joue la
// mutation « un dead-state ecrit est ignore » et EXIGE que la fin publiee devienne fausse. Un
// test qui resterait vert sous cette mutation ne prouverait rien de la lecture.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// horlogeDeTest : une grille de 100 ms sur `frames` frames, origine a zero.
func horlogeDeTest(frames int) replayClock {
	return replayClock{origin: 0, step: 100_000, frames: frames}
}

// vieDeTest : une vie bornee a la main, deja fenetree (c est ce que fait `assignVehicleWindows`).
func vieDeTest(slot, gen uint32, firstUS, lastUS, goneByUS, loUS, hiUS uint64) vehicleLife {
	return vehicleLife{
		key: grammar.EquipmentLifeKey{Slot: slot, Gen: gen}, firstUS: firstUS, lastUS: lastUS,
		goneByUS: goneByUS, loUS: loUS, hiUS: hiUS, census: 2,
	}
}

// TestAssignVehicleDeathsAttribueParLaFenetre : deux vies du MEME `(slot, gen)` — le pool de
// slots reboucle et la generation ne fait que 2 bits — se departagent par leur fenetre, et la
// mort la PLUS PRECOCE de la fenetre gagne (le film re-replique le dead-state sur plusieurs
// ticks).
func TestAssignVehicleDeathsAttribueParLaFenetre(t *testing.T) {
	lives := []vehicleLife{
		vieDeTest(777, 1, 10_000_000, 40_000_000, 60_000_000, 0, 60_000_000),
		vieDeTest(777, 1, 100_000_000, 140_000_000, 0, 60_000_000, ^uint64(0)),
	}
	deaths := []grammar.ObjectDeath{
		{TimestampUS: 55_000_000, Slot: 777, Gen: 1},
		{TimestampUS: 56_000_000, Slot: 777, Gen: 1},
		{TimestampUS: 150_000_000, Slot: 777, Gen: 1, TailDesync: true},
		{TimestampUS: 33_000_000, Slot: 999, Gen: 0},
	}
	tally := assignVehicleDeaths(lives, deaths)
	if lives[0].deathUS != 55_000_000 {
		t.Errorf("premiere vie : deathUS=%d, attendu 55000000 (la mort la plus precoce de la"+
			" fenetre)", lives[0].deathUS)
	}
	if lives[1].deathUS != 150_000_000 || !lives[1].deathTailDesync {
		t.Errorf("seconde vie : deathUS=%d tailDesync=%v, attendu 150000000 / true",
			lives[1].deathUS, lives[1].deathTailDesync)
	}
	if tally.read != 4 || tally.matched != 3 || tally.unmatched != 1 || tally.tailDesync != 1 {
		t.Errorf("bilan = %+v, attendu read=4 matched=3 unmatched=1 tailDesync=1", tally)
	}
}

// TestAssignVehicleDeathsSansVie : une mort que personne ne reprend est COMPTEE, jamais jetee —
// c est le signal qu une vie manque au recensement.
func TestAssignVehicleDeathsSansVie(t *testing.T) {
	tally := assignVehicleDeaths(nil, []grammar.ObjectDeath{{TimestampUS: 1, Slot: 5}})
	if tally.read != 1 || tally.unmatched != 1 || tally.matched != 0 {
		t.Errorf("bilan = %+v, attendu read=1 unmatched=1 matched=0", tally)
	}
}

// TestVehicleEndOfLesTroisCauses : les trois valeurs publiees, et l ORDRE qui les decide —
// on LIT d abord (la mort ecrite), on constate ensuite (la vie court jusqu au bout du film), et
// `unknown` est l aveu compte qu aucun des deux ne s applique.
func TestVehicleEndOfLesTroisCauses(t *testing.T) {
	clock := horlogeDeTest(3000)
	for _, cas := range []struct {
		nom       string
		vie       vehicleLife
		veutCause string
		veutTEnd  int // -1 = absent
	}{
		{
			nom:       "le film ecrit la mort",
			vie:       vehicleLife{deathUS: 28_200_000, goneByUS: 28_740_000},
			veutCause: VehicleEndDestroyed, veutTEnd: 282,
		},
		{
			nom:       "la derniere image-cle la recense encore",
			vie:       vehicleLife{goneByUS: 0},
			veutCause: VehicleEndFilmEnd, veutTEnd: -1,
		},
		{
			nom:       "elle disparait du recensement sans mort ecrite",
			vie:       vehicleLife{goneByUS: 28_740_000},
			veutCause: VehicleEndUnknown, veutTEnd: -1,
		},
		{
			nom:       "la mort ecrite prime sur la fin de film",
			vie:       vehicleLife{deathUS: 10_000_000, goneByUS: 0},
			veutCause: VehicleEndDestroyed, veutTEnd: 100,
		},
	} {
		cause, tEnd := vehicleEndOf(cas.vie, clock)
		if cause != cas.veutCause {
			t.Errorf("%s : cause %q, attendue %q", cas.nom, cause, cas.veutCause)
		}
		switch {
		case cas.veutTEnd < 0 && tEnd != nil:
			t.Errorf("%s : tEnd=%d publie alors que la fin n est pas datee", cas.nom, *tEnd)
		case cas.veutTEnd >= 0 && tEnd == nil:
			t.Errorf("%s : tEnd absent alors que la fin est datee", cas.nom)
		case cas.veutTEnd >= 0 && *tEnd != cas.veutTEnd:
			t.Errorf("%s : tEnd=%d, attendu %d", cas.nom, *tEnd, cas.veutTEnd)
		}
	}
}

// TestTallyVehicleEndsCompteLaContradiction : un echantillon POSTERIEUR a une fin datee est une
// contradiction publiee, et la trajectoire n est PAS coupee — couper masquerait le desaccord qui
// dirait que l attribution est fausse.
func TestTallyVehicleEndsCompteLaContradiction(t *testing.T) {
	f282 := 282
	tracks := []VehicleTrack{
		{End: VehicleEndDestroyed, TEnd: &f282, Samples: []VehicleSample{{T: 280}, {T: 283}, {T: 290}}},
		{End: VehicleEndFilmEnd, Samples: []VehicleSample{{T: 10}}},
		{End: VehicleEndUnknown},
		{End: VehicleEndUnknown},
	}
	var cov VehicleCoverage
	tallyVehicleEnds(tracks, &cov)
	if cov.EndDestroyed != 1 || cov.EndFilmEnd != 1 || cov.EndUnknown != 2 {
		t.Errorf("ventilation = destroyed %d / film_end %d / unknown %d, attendu 1 / 1 / 2",
			cov.EndDestroyed, cov.EndFilmEnd, cov.EndUnknown)
	}
	if cov.SamplesAfterEnd != 2 {
		t.Errorf("SamplesAfterEnd=%d, attendu 2", cov.SamplesAfterEnd)
	}
	if n := len(tracks[0].Samples); n != 3 {
		t.Errorf("la trajectoire a ete coupee a la fin datee (%d points restants) : la"+
			" contradiction doit se COMPTER, pas se masquer", n)
	}
}

// TestAssignVehicleWindowsSansBorneHaute : LE CORRECTIF DU LOT. Une vie que le recensement ne
// ferme jamais (`goneByUS == 0`) finit AVEC le film : sa fenetre n a pas de borne haute. L
// ancienne borne « dernier recensement + 20 s » coupait le nuage de positions d un vehicule
// vivant — c est elle qui effacait les vehicules avant la fin du rejeu.
func TestAssignVehicleWindowsSansBorneHaute(t *testing.T) {
	lives := []vehicleLife{
		{key: grammar.EquipmentLifeKey{Slot: 777, Gen: 1}, firstUS: 0, lastUS: 200_000_000, goneByUS: 0},
	}
	assignVehicleWindows(lives)
	if lives[0].hiUS != ^uint64(0) {
		t.Errorf("hiUS=%d : une vie que le recensement ne ferme jamais ne doit pas etre bornee"+
			" (l ancienne valeur etait lastUS + 20 s = %d)", lives[0].hiUS,
			200_000_000+vehicleCensusTolUS)
	}
}

// TestAssignVehicleWindowsFrontiereEntreDeuxVies : la vie SUIVANTE du meme slot reste le seul
// decoupage legitime d une fenetre sans borne haute.
func TestAssignVehicleWindowsFrontiereEntreDeuxVies(t *testing.T) {
	lives := []vehicleLife{
		{key: grammar.EquipmentLifeKey{Slot: 777, Gen: 1}, firstUS: 0, lastUS: 40_000_000, goneByUS: 0},
		{key: grammar.EquipmentLifeKey{Slot: 777, Gen: 2}, firstUS: 100_000_000, lastUS: 140_000_000},
	}
	assignVehicleWindows(lives)
	if lives[0].hiUS != 100_000_000 {
		t.Errorf("hiUS de la premiere vie = %d, attendu 100000000 (la naissance de la suivante)",
			lives[0].hiUS)
	}
	if lives[1].loUS < lives[0].hiUS {
		t.Errorf("les deux fenetres se chevauchent : loUS=%d < hiUS=%d", lives[1].loUS, lives[0].hiUS)
	}
}

// TestVehicleLivesPoseLaMortEcrite : le CABLAGE, pas seulement la fonction. `vehicleLives` est
// le seul point d'ou l'assemblage appelle la lecture ; ce test echoue si cet appel disparait.
func TestVehicleLivesPoseLaMortEcrite(t *testing.T) {
	kf := grammar.WorldObjectKeyframes{
		TimesUS: []uint64{0, 20_000_000, 40_000_000, 60_000_000},
		SeenUS: map[grammar.EquipmentLifeKey][]uint64{
			{Slot: 777, Gen: 1}: {0, 20_000_000, 40_000_000},
			{Slot: 778, Gen: 1}: {0, 20_000_000, 40_000_000, 60_000_000},
		},
	}
	deaths := []grammar.ObjectDeath{{TimestampUS: 45_000_000, Slot: 777, Gen: 1}}
	lives, tally := vehicleLives(kf, deaths)
	if tally.matched != 1 {
		t.Fatalf("bilan = %+v : la lecture n est pas cablee dans `vehicleLives`", tally)
	}
	var vue777 bool
	for _, l := range lives {
		if l.key.Slot != 777 {
			continue
		}
		vue777 = true
		if l.deathUS != 45_000_000 {
			t.Errorf("la vie 777 porte deathUS=%d, attendu 45000000", l.deathUS)
		}
	}
	if !vue777 {
		t.Fatalf("la vie 777 est absente des vies assemblees")
	}
}

// TestFinDeVieIgnorerLeDeadStateRendUneFinFausse — LA MUTATION DU LOT.
//
// Elle rejoue la situation du `ghost` slot 777 de `bfecd02b` : une vie que le film declare morte
// a 282,1 s alors que le recensement ne la ferme qu a 287,4 s. La mutation « ignorer le
// dead-state ecrit » est simulee en n attribuant AUCUNE mort — c est exactement ce que faisait
// le code avant ce lot — et le test EXIGE que la fin publiee devienne fausse : ni datee, ni
// nommee `destroyed`.
//
// Restauration PAR NOM : l appel `assignVehicleDeaths`, dans `vehicleLives`.
func TestFinDeVieIgnorerLeDeadStateRendUneFinFausse(t *testing.T) {
	clock := horlogeDeTest(5033)
	const mortUS = 282_100_000
	vie := vieDeTest(777, 1, 0, 268_000_000, 287_400_000, 0, 287_400_000)
	deaths := []grammar.ObjectDeath{{TimestampUS: mortUS, Slot: 777, Gen: 1}}

	lu := []vehicleLife{vie}
	assignVehicleDeaths(lu, deaths)
	cause, tEnd := vehicleEndOf(lu[0], clock)
	if cause != VehicleEndDestroyed || tEnd == nil || *tEnd != 2821 {
		t.Fatalf("lecture : cause=%q tEnd=%v, attendu %q / 2821", cause, tEnd, VehicleEndDestroyed)
	}

	// MUTATION : le dead-state ecrit est ignore (aucune mort attribuee).
	mute := []vehicleLife{vie}
	assignVehicleDeaths(mute, nil)
	causeMutee, tEndMute := vehicleEndOf(mute[0], clock)
	if causeMutee == VehicleEndDestroyed || tEndMute != nil {
		t.Fatalf("MUTATION SANS EFFET : la fin reste %q / %v alors que le dead-state est ignore"+
			" — le test ne prouve rien de la lecture", causeMutee, tEndMute)
	}
	if causeMutee != VehicleEndUnknown {
		t.Errorf("sous mutation, la cause vaut %q ; c est %q qu elle valait avant le lot",
			causeMutee, VehicleEndUnknown)
	}
}
