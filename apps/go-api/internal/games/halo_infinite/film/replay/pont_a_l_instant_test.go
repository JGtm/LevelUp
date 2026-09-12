package replay

// pont_a_l_instant_test.go — LES CINQ LECTEURS QUI INTERROGENT LE SIEGE A UN INSTANT.
//
// # CE QUE CE FICHIER EPINGLE (lot 6.1, 2026-09-10)
//
// Cinq calques demandaient « qui occupe ce siege ? » au pont APLATI (`PontParSlot`), qui garde le
// PREMIER occupant d'un siege recycle entre deux joueurs. Les cinq CONNAISSENT pourtant l'instant
// de leur lecture — le ramassage est date, l'episode d'equipement est borne, la prise de bombe
// est datee, la lecture d'inventaire porte sa frame, le coup fatal porte son horodatage. Ils
// passent desormais par le registre A L'INSTANT (`XUIDNumAt`).
//
// LA MESURE QUI A COMMANDE CE LOT (rapport `.ai/V7.5/RAPPORT_PONT_APLATI_2026-09-10.md`) : sur
// 74 films, UN SEUL siege ambigu (`084a804d`/603) — mais il suffit a publier deux ramassages sous
// le nom d'un joueur que le film place ailleurs. Le parc mesure le VOLUME ; ce fichier epingle la
// REGLE, sur un siege synthetique que le parc ne fournit qu'une fois.
//
// # LE SIEGE DE REFERENCE
//
// Un siege 900 que DEUX joueurs se partagent (111 puis 222), et un siege 901 tenu par un
// troisieme (333) — le temoin qui doit rester insensible. Le pont aplati donne 900 -> 111 pour
// tout le film ; le registre a l'instant donne 111 dans la premiere vie, 222 dans la seconde, et
// RIEN dans le trou entre les deux (un siege ambigu hors vie nommee ne se nomme pas — c'est
// l'abstention de `xuidAt`, et elle est aussi importante que la correction).
//
// # MUTATION (a rejouer pour prouver que ces tests mordent)
//
// Remplacer `reg.XUIDNumAt(...)` par une lecture aplatie du siege dans l'un des cinq sites
// migres : le test correspondant rougit avec l'identite du PREMIER occupant (111) la ou l'on
// attend le second (222), ou avec un nom la ou l'on attend le silence.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// Les instants du siege 900, en microsecondes de l'horloge du film. Le pas de grille des tests
// vaut 100 000 us (100 ms), donc frame = us / 100 000.
const (
	siegeVie1FinUS   = 1_000_000 // 111 tient le siege jusque-la
	siegeTrouUS      = 1_500_000 // entre les deux vies : personne n'est etabli
	siegeVie2DebutUS = 2_000_000
	siegeLectureUS   = 2_500_000 // l'instant de toutes les lectures : 222 tient le siege
	siegeVie2FinUS   = 3_000_000
)

// registreSiegeRecycle monte le registre du siege partage. `regDeTest` pose les tables telles
// quelles : c'est exactement ce que `ownersFromLives` produit d'un siege a deux occupants nommes
// — le pont aplati garde le premier, et le siege est marque AMBIGU.
func registreSiegeRecycle() IdentityRegistry {
	return regDeTest(
		[]lifeSpan{
			{slot: 900, from: 0, to: siegeVie1FinUS, xuid: 111},
			{slot: 900, from: siegeVie2DebutUS, to: siegeVie2FinUS, xuid: 222},
			{slot: 901, from: 0, to: siegeVie2FinUS, xuid: 333},
		},
		map[uint32]uint64{900: 111, 901: 333},
		map[uint32]bool{900: true},
	)
}

// horlogeDesTests : origine a zero, pas de 100 ms — la correspondance frame <-> microseconde est
// alors une simple division, et chaque instant du fichier se lit sans table de conversion.
func horlogeDesTests() replayClock {
	return replayClock{origin: 0, step: 100_000, frames: 40}
}

// TestRegistreNommeLOccupantDuSiegeALInstant : la propriete dont les cinq autres tests decoulent.
func TestRegistreNommeLOccupantDuSiegeALInstant(t *testing.T) {
	reg := registreSiegeRecycle()
	if x := reg.XUIDNumAt(900, siegeLectureUS); x != 222 {
		t.Errorf("siege 900 a %d us : occupant %d, attendu 222 (le pont aplati sert 111)", siegeLectureUS, x)
	}
	if x := reg.XUIDNumAt(900, 500_000); x != 111 {
		t.Errorf("siege 900 a 500 000 us : occupant %d, attendu 111", x)
	}
	if x := reg.XUIDNumAt(900, siegeTrouUS); x != 0 {
		t.Errorf("siege 900 dans le trou : occupant %d, attendu AUCUN (siege ambigu hors vie nommee)", x)
	}
	if x := reg.XUIDNumAt(901, siegeLectureUS); x != 333 {
		t.Errorf("siege temoin 901 : occupant %d, attendu 333", x)
	}
}

// TestRamassageSuitLOccupantDuSiege : `buildPickups` nomme le ramasseur par l'occupant du siege A
// L'INSTANT du ramassage, jamais par le premier occupant du film.
func TestRamassageSuitLOccupantDuSiege(t *testing.T) {
	reg := registreSiegeRecycle()
	pickups := []filmdec.BipedPickup{
		{TimestampUS: siegeLectureUS, Slot: 900, CatalogID: 0x1234, Class: 0},
		{TimestampUS: siegeTrouUS, Slot: 900, CatalogID: 0x1234, Class: 0},
		{TimestampUS: siegeLectureUS, Slot: 901, CatalogID: 0x1234, Class: 0},
	}
	got, cov := buildPickups(pickups, horlogeDesTests(), pickupInputs{occupant: reg.XUIDNumAt})
	if len(got) != 3 {
		t.Fatalf("3 ramassages publies attendus, obtenu %d", len(got))
	}
	if got[0].XUID != "222" {
		t.Errorf("ramassage dans la 2e vie du siege : xuid %q, attendu \"222\"", got[0].XUID)
	}
	if got[1].XUID != "" {
		t.Errorf("ramassage dans le trou d'un siege ambigu : xuid %q, attendu le SILENCE", got[1].XUID)
	}
	if got[2].XUID != "333" {
		t.Errorf("ramassage du siege temoin : xuid %q, attendu \"333\"", got[2].XUID)
	}
	if cov.Named != 2 {
		t.Errorf("ramassages nommes = %d, attendu 2 (celui du trou ne se nomme pas)", cov.Named)
	}
}

// TestFragSousEquipementSuitLOccupantDuSiege : un frag sous camouflage credite l'episode du corps
// qui tenait le siege a cet instant.
func TestFragSousEquipementSuitLOccupantDuSiege(t *testing.T) {
	reg := registreSiegeRecycle()
	eps := []EquipmentEpisode{
		{Slot: 900, Fam: EquipFamilyCamo, T0: 20, T1: 30},
		{Slot: 901, Fam: EquipFamilyCamo, T0: 20, T1: 30},
	}
	kills := []EquipmentKillRef{
		{XUID: 222, TimeMS: 2_500},
		{XUID: 333, TimeMS: 2_500, AssistXUID: 222, AssistKnown: true},
	}
	attachEpisodeKills(eps, kills, occupantParFrame(reg, horlogeDesTests()), 0, 100)
	if eps[0].K != 1 || eps[0].A != 1 {
		t.Errorf("episode du 2e occupant : k=%d a=%d, attendu k=1 a=1 (le pont aplati creditait 111)",
			eps[0].K, eps[0].A)
	}
	if eps[1].K != 1 || eps[1].A != 0 {
		t.Errorf("episode du siege temoin : k=%d a=%d, attendu k=1 a=0", eps[1].K, eps[1].A)
	}
}

// TestPortageDeBombeSuitLOccupantDuSiege : la periode de portage revient au corps qui tenait le
// siege a la PRISE.
func TestPortageDeBombeSuitLOccupantDuSiege(t *testing.T) {
	reg := registreSiegeRecycle()
	events := []HeldObjectEvent{
		{TimeMS: 2_500, Slot: 900, Pickup: true},
		{TimeMS: 2_800, Slot: 900, Pickup: false},
	}
	carry := BuildHeldObjectCarry(events, occupantParMatchMS(reg), nil)
	if len(carry.Periods) != 1 {
		t.Fatalf("une periode attendue, obtenu %d", len(carry.Periods))
	}
	if carry.Periods[0].XUID != 222 {
		t.Errorf("porteur = %d, attendu 222 (le pont aplati servait 111)", carry.Periods[0].XUID)
	}
	if carry.CarryMSByXUID[222] != 300 {
		t.Errorf("duree portee par 222 = %d ms, attendu 300", carry.CarryMSByXUID[222])
	}
}

// TestInventaireMortSuitLOccupantDuSiege : la requalification `unknown` -> `dead` consulte les
// morts du joueur qui tenait le siege a l'instant de la lecture.
func TestInventaireMortSuitLOccupantDuSiege(t *testing.T) {
	reg := registreSiegeRecycle()
	inv := []Inventory{
		{T: 25, Slot: 900, Empty: InventoryEmptyUnknown},
		{T: 15, Slot: 900, Empty: InventoryEmptyUnknown},
	}
	deaths := []Death{{XUID: 222, TimeMS: 2_000}, {XUID: 111, TimeMS: 1_400}}
	n := markInventoryDeadReadings(inv, deaths, reg, horlogeDesTests())
	if n != 1 {
		t.Fatalf("lectures requalifiees = %d, attendu 1", n)
	}
	if inv[0].Empty != InventoryEmptyDead {
		t.Errorf("lecture a la frame 25 : %q, attendu %q (222 vient de mourir)",
			inv[0].Empty, InventoryEmptyDead)
	}
	if inv[1].Empty != InventoryEmptyUnknown {
		t.Errorf("lecture dans le trou d'un siege ambigu : %q, attendu %q — la mort de 111 ne la "+
			"qualifie pas, rien n'etablit qu'il tenait encore le siege",
			inv[1].Empty, InventoryEmptyUnknown)
	}
}

// TestPositionDeKillSuitLOccupantDuSiege : la position du tueur se cherche sur le corps qu'il
// occupe A L'INSTANT du coup fatal — le second occupant d'un siege recycle n'est plus invisible.
func TestPositionDeKillSuitLOccupantDuSiege(t *testing.T) {
	reg := registreSiegeRecycle()
	pos := []filmdec.BipedPosition{
		posAt(900, siegeLectureUS, 7, 8, 0),
		posAt(901, siegeLectureUS, 1, 2, 0),
	}
	kills := []KillRef{{KillerXUID: 222, VictimXUID: 333, TimeMS: 2_500}}
	got, rep := BuildKillPositions(pos, reg, kills, 0)
	if len(got) != 1 {
		t.Fatalf("une mort placee attendue, obtenu %d (%+v)", len(got), rep)
	}
	if got[0].Killer == nil || got[0].Killer.X != 7 {
		t.Errorf("position du tueur = %+v, attendu le corps du siege 900 (x=7) ; le pont aplati "+
			"ne donnait AUCUN siege a 222", got[0].Killer)
	}
	if got[0].Victim == nil || got[0].Victim.X != 1 {
		t.Errorf("position de la victime = %+v, attendu le siege temoin 901 (x=1)", got[0].Victim)
	}
	if rep.Both != 1 || rep.NoBridge != 0 {
		t.Errorf("compte rendu = %+v, attendu Both=1 NoBridge=0", rep)
	}
}

// TestPositionDeKillSeTaitHorsDesVies : le PREMIER occupant ne recupere pas le corps du second.
// C'est l'autre moitie de la correction — celle qui retire une position fausse plutot que d'en
// rendre une vraie.
func TestPositionDeKillSeTaitHorsDesVies(t *testing.T) {
	reg := registreSiegeRecycle()
	pos := []filmdec.BipedPosition{
		posAt(900, siegeLectureUS, 7, 8, 0),
		posAt(901, siegeLectureUS, 1, 2, 0),
	}
	// 111 n'a AUCUN corps a cet instant : sa seule vie sur 900 est finie depuis 1,5 s.
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 333, TimeMS: 2_500}}
	got, rep := BuildKillPositions(pos, reg, kills, 0)
	if len(got) != 1 || got[0].Killer != nil {
		t.Fatalf("le tueur ne devait PAS etre place sur le corps d'un autre : %+v", got)
	}
	if rep.VictimOnly != 1 || rep.NoBridge != 1 {
		t.Errorf("compte rendu = %+v, attendu VictimOnly=1 NoBridge=1", rep)
	}
}
