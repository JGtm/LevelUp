package replay

// flag_return_gauge_test.go — L APPARIEMENT DE LA JAUGE DE RETOUR, sur tampon synthetique.
//
// AUCUN FILM N EST OUVERT ICI. Les tampons reproduisent la SEPARATION MESUREE du lot 5.1.6 —
// une jauge a 100 % dans les lachers de son drapeau, un slot voisin a 15,8 % — et verifient que
// la regle tranche le bon cote, qu elle refuse le mauvais, et qu elle publie un ESCALIER.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// grammarManagedRead : une lecture de `ti=13` dite en FRAMES, que `frgInput` traduit en
// microsecondes. Ecrire des frames rend les tampons lisibles ; la conversion reste celle de
// production (`frameOf`), donc le test ne recopie aucune arithmetique d horloge.
type grammarManagedRead struct {
	slot  uint32
	frame int
	value uint64
}

// frgClock : origine a zero, une frame par seconde, cent frames.
func frgClock() matchClock { return matchClock{origin: 0, step: frgStep, frames: 100} }

// frgInput monte les lectures en entree de calque, BALAYAGE DECLARE.
func frgInput(rs []grammarManagedRead) FlagInput {
	in := FlagInput{GaugeScanned: true}
	for _, r := range rs {
		in.Gauge = append(in.Gauge, grammar.ManagedPropertyRead{
			Slot:        r.slot,
			TimestampUS: uint64(r.frame) * frgStep,
			Field:       grammar.ManagedPropertyScalar,
			Tag:         grammar.ManagedPropertyTagQuant,
			Value:       r.value,
			HasValue:    true,
		})
	}
	return in
}

// frgStep : 1 000 000 us par frame, soit une frame par seconde. `zoneGaugeGapFrames` rend alors 1.
const frgStep = uint64(1_000_000)

// frgQuantum rend le quantum du tag 3 pour une fraction de [0, 1] sur l echelle du JEU.
func frgQuantum(v float64) uint64 {
	return uint64(zoneGaugeQuantZero + int(v*zoneGaugeQuantUnit))
}

// frgLecture fabrique une lecture scalaire de `ti=13` au tag 3.
func frgLecture(slot uint32, frame int, v float64) grammarManagedRead {
	return grammarManagedRead{slot: slot, frame: frame, value: frgQuantum(v)}
}

// frgCarry fabrique un drapeau : un `home`, un `dropped` sur [t0, t1], puis un `home`.
func frgCarry(team, t0, t1 int) FlagCarry {
	return FlagCarry{Team: team, Spans: []FlagSpan{
		{State: FlagStateHome, T0: 0, T1: t0 - 1},
		{State: FlagStateDropped, T0: t0, T1: t1},
		{State: FlagStateHome, T0: t1 + 1, T1: t1 + 50},
	}}
}

func TestAttachFlagReturnGaugesApparieLaJaugeEtPublieUnEscalier(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40)}
	// Dix lectures, TOUTES dans le lacher : la jauge de ce drapeau.
	var reads []grammarManagedRead
	for i := 0; i < 10; i++ {
		reads = append(reads, frgLecture(1490, 20+2*i, float64(i)/10))
	}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	pts := carries[0].Spans[1].ReturnProgress
	if len(pts) == 0 {
		t.Fatalf("aucune serie publiee — couverture : %+v", cov)
	}
	if !cov.GaugeScanned || cov.GaugePaired != 1 || cov.GaugeSpans != 1 {
		t.Fatalf("couverture = balaye %v, apparies %d, lachers %d — attendu true/1/1",
			cov.GaugeScanned, cov.GaugePaired, cov.GaugeSpans)
	}
	if pts[0].T < 20 || pts[len(pts)-1].T > 40 {
		t.Errorf("la serie deborde du lacher [20, 40] : %d..%d", pts[0].T, pts[len(pts)-1].T)
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].T <= pts[i-1].T {
			t.Fatalf("T n est pas STRICTEMENT croissant : %+v", pts)
		}
	}
	if pts[0].V != 0 {
		t.Errorf("premier point = %v, attendu 0 (le depart de la rampe est toujours publie)", pts[0].V)
	}
	// LES AUTRES ETATS NE PORTENT RIEN : la jauge ne se remplit que pour un drapeau AU SOL.
	if carries[0].Spans[0].ReturnProgress != nil || carries[0].Spans[2].ReturnProgress != nil {
		t.Error("un etat `home` porte une jauge : seul `dropped` peut en porter")
	}
}

// TestAttachFlagReturnGaugesRefuseLeSlotVoisin reproduit le contraste MESURE : le slot 1495 de
// `bcb6d393` ne tombe dans les lachers qu a 15,8 %. Il ne doit pas etre pris pour une jauge.
func TestAttachFlagReturnGaugesRefuseLeSlotVoisin(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40)}
	var reads []grammarManagedRead
	for i := 0; i < 19; i++ { // 3 dedans sur 19 = 15,8 %
		f := 60 + i
		if i < 3 {
			f = 22 + i
		}
		reads = append(reads, frgLecture(1495, f, 0.5))
	}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	if carries[0].Spans[1].ReturnProgress != nil {
		t.Fatal("le slot voisin a ete pris pour la jauge du drapeau")
	}
	if cov.GaugeSlots != 1 || cov.GaugePaired != 0 {
		t.Errorf("couverture = slots %d, apparies %d — attendu 1/0 : le canal parle, rien ne correle",
			cov.GaugeSlots, cov.GaugePaired)
	}
}

// TestAttachFlagReturnGaugesRefuseUnePoigneeDEchantillons : deux lectures tombees dans un lacher
// font 100 % sans rien prouver. Le plancher les ecarte.
func TestAttachFlagReturnGaugesRefuseUnePoigneeDEchantillons(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40)}
	reads := []grammarManagedRead{frgLecture(1490, 25, 0.1), frgLecture(1490, 30, 0.6)}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	if carries[0].Spans[1].ReturnProgress != nil {
		t.Fatal("deux echantillons ont suffi a etablir une jauge")
	}
}

// TestAttachFlagReturnGaugesUnSlotParDrapeau : deux drapeaux, deux slots, et chacun va au sien.
// Le jeu attache la propriete a l OBJET, et la mesure le confirme sur les deux films.
func TestAttachFlagReturnGaugesUnSlotParDrapeau(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40), frgCarry(1, 60, 80)}
	var reads []grammarManagedRead
	for i := 0; i < 10; i++ {
		reads = append(reads, frgLecture(1614, 20+2*i, float64(i)/10))
		reads = append(reads, frgLecture(1619, 60+2*i, float64(i)/10))
	}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	if cov.GaugePaired != 2 {
		t.Fatalf("drapeaux apparies = %d, attendu 2", cov.GaugePaired)
	}
	if carries[0].Spans[1].ReturnProgress == nil || carries[1].Spans[1].ReturnProgress == nil {
		t.Fatal("un des deux drapeaux n a pas recu sa serie")
	}
}

// TestAttachFlagReturnGaugesNeLitRienSansBalayage : `GaugeScanned` faux est le cas nominal hors
// CTF. La couverture le DIT, et aucune serie n est publiee.
func TestAttachFlagReturnGaugesNeLitRienSansBalayage(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40)}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, FlagInput{}, frgClock(), &cov)
	if cov.GaugeScanned || cov.GaugeSlots != 0 || carries[0].Spans[1].ReturnProgress != nil {
		t.Errorf("un balayage absent a publie quelque chose : %+v", cov)
	}
}
