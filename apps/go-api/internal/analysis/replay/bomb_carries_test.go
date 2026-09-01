package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// bomb_carries_test.go — LE PORTEUR DE LA BOMBE, sur des transitions synthetiques (CI, sans
// film). On y prouve quatre choses : les transitions du canal des armes tenues deviennent des
// periodes, la MORT du porteur ferme SANS emission (le piege mesure du protocole B2), le gate
// de presence ecarte et rogne comme pour le crane, et la garde `CarryScanned` retient tout.

// bombChange fabrique une transition du canal des armes tenues, datee µs film.
func bombChange(tUS uint64, slot uint32, family, previous uint32) filmdec.HeldWeaponChange {
	return filmdec.HeldWeaponChange{TimestampUS: tUS, Slot: slot, Family: family, Previous: previous}
}

const bombTestOther = uint32(0x11111111)

// TestBombHeldEventsFilterAndClock : seules les transitions de la famille bombe sortent, datees
// sur l'horloge du MATCH (µs film / 1000 − deathOffsetMS).
func TestBombHeldEventsFilterAndClock(t *testing.T) {
	changes := []filmdec.HeldWeaponChange{
		bombChange(10_000_000, 3, bombHeldFamily, filmdec.NoWeaponVariant), // prise a 10 s film
		bombChange(12_000_000, 3, bombTestOther, bombHeldFamily),           // lacher (swap) a 12 s
		bombChange(15_000_000, 5, bombTestOther, filmdec.NoWeaponVariant),  // une arme : ignoree
	}
	evs := bombHeldEventsOf(changes, 2_000) // horlogeFilm = horlogeMatch + 2 s
	if len(evs) != 2 {
		t.Fatalf("evenements = %d, attendu 2 (prise + lacher) : %+v", len(evs), evs)
	}
	if !evs[0].Pickup || evs[0].TimeMS != 8_000 || evs[0].Slot != 3 {
		t.Errorf("prise = %+v, attendu {8000, slot 3, pickup}", evs[0])
	}
	if evs[1].Pickup || evs[1].TimeMS != 10_000 {
		t.Errorf("lacher = %+v, attendu {10000, drop}", evs[1])
	}
}

// bombTestCarry reconstruit une chronologie depuis des evenements ms match.
func bombTestCarry(evs []HeldObjectEvent, slotXUID map[uint32]uint64, deaths []Death) HeldObjectCarry {
	return BuildHeldObjectCarry(evs, slotXUID, deaths)
}

// TestBombCarriesDeathClosesWithoutEmission : une prise sans lacher dont le porteur MEURT se
// ferme a la mort (ByDeath), une prise sans rien reste ouverte (clampee a la fin de l'axe).
func TestBombCarriesDeathClosesWithoutEmission(t *testing.T) {
	slotXUID := map[uint32]uint64{3: 111, 7: 222}
	evs := []HeldObjectEvent{
		{TimeMS: 1_000, Slot: 3, Pickup: true}, // porteur 111 : prise...
		{TimeMS: 5_000, Slot: 7, Pickup: true}, // ...222 prend (111 est mort a 4 s, sans emission)
	}
	deaths := []Death{{XUID: 111, TimeMS: 4_000}}
	// step = 1000 µs/frame => 1 frame par ms.
	carries, cov := buildBombCarries(bombTestCarry(evs, slotXUID, deaths),
		matchClock{origin: 0, step: 1000, frames: 20_000}, nil)
	if cov == nil || !cov.BombFilm {
		t.Fatalf("couverture absente ou BombFilm faux : %+v", cov)
	}
	if len(carries) != 2 {
		t.Fatalf("portages = %d, attendu 2 : %+v", len(carries), carries)
	}
	// Periode 1 : fermee A LA MORT (4 000), pas a la prise suivante (5 000).
	if carries[0].XUID != "111" || carries[0].T0 != 1_000 || carries[0].T1 != 4_000 || !carries[0].Closed {
		t.Errorf("periode 1 = %+v, attendu {111, 1000, 4000, fermee par mort}", carries[0])
	}
	// Periode 2 : rien ne la ferme — ouverte, clampee a la fin de l'axe.
	if carries[1].XUID != "222" || carries[1].T0 != 5_000 || carries[1].T1 != 19_999 || carries[1].Closed {
		t.Errorf("periode 2 = %+v, attendu {222, 5000, 19999, ouverte}", carries[1])
	}
	if cov.ByDeath != 1 || cov.Closed != 1 || cov.Open != 1 {
		t.Errorf("couverture = %+v, attendu ByDeath 1, Closed 1, Open 1", cov)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestBombCarriesBridgeAndPresence : un slot non ponte part en NoBridge ; un porteur absent des
// pistes publiees part en CarrierAbsent ; un portage qui deborde de la vie est ROGNE.
func TestBombCarriesBridgeAndPresence(t *testing.T) {
	slotXUID := map[uint32]uint64{3: 111, 7: 222}
	evs := []HeldObjectEvent{
		{TimeMS: 1_000, Slot: 3, Pickup: true}, {TimeMS: 3_000, Slot: 3, Pickup: false},
		{TimeMS: 4_000, Slot: 9, Pickup: true}, {TimeMS: 5_000, Slot: 9, Pickup: false}, // non ponte
		{TimeMS: 6_000, Slot: 7, Pickup: true}, {TimeMS: 9_000, Slot: 7, Pickup: false},
	}
	presence := map[string][]presenceSpan{
		"111": {{f0: 0, f1: 2_000}},       // vie plus courte que le portage [1000, 3000] : rognage
		"222": {{f0: 12_000, f1: 15_000}}, // aucune vie ne couvre [6000, 9000] : fantome
	}
	carries, cov := buildBombCarries(bombTestCarry(evs, slotXUID, nil),
		matchClock{origin: 0, step: 1000, frames: 20_000}, presence)
	if len(carries) != 1 {
		t.Fatalf("portages = %d, attendu 1 (NoBridge et CarrierAbsent ecartes) : %+v", len(carries), carries)
	}
	if carries[0].XUID != "111" || carries[0].T1 != 2_000 {
		t.Errorf("portage = %+v, attendu {111, ..., rogne a 2000}", carries[0])
	}
	if cov.NoBridge != 1 || cov.CarrierAbsent != 1 || cov.Periods != 3 {
		t.Errorf("couverture = %+v, attendu NoBridge 1, CarrierAbsent 1, Periods 3", cov)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestBombCarriesGuards : la garde de mode retient tout ; sans pont, la couverture dit ce qui a
// ete vu et rien n'est publie.
func TestBombCarriesGuards(t *testing.T) {
	doc := ReplayDocument{MatchID: "test", Coverage: &Coverage{}}
	opt := Options{WeaponChanges: []filmdec.HeldWeaponChange{
		bombChange(1_000_000, 3, bombHeldFamily, filmdec.NoWeaponVariant),
	}}
	// Garde fermee : rien, pas meme une couverture.
	attachBombCarries(&doc, opt, OwnerReport{}, replayClock{origin: 0, step: 1000, frames: 100})
	if doc.BombCarries != nil || doc.Coverage.BombCarries != nil {
		t.Fatalf("garde fermee : calque %v, couverture %v — attendu rien",
			doc.BombCarries, doc.Coverage.BombCarries)
	}
	// Garde ouverte, pont vide : couverture avec les transitions vues, aucun portage.
	opt.Bomb.CarryScanned = true
	attachBombCarries(&doc, opt, OwnerReport{}, replayClock{origin: 0, step: 1000, frames: 100})
	cov := doc.Coverage.BombCarries
	if cov == nil || !cov.BombFilm || cov.Events != 1 || cov.Carries != 0 {
		t.Fatalf("pont vide : couverture %+v, attendu {BombFilm, Events 1, Carries 0}", cov)
	}
}
