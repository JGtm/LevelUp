package replay

// killpos_bord_lu_test.go — ce que la lecture du BORD apporte a la lecture du lieu, et que les tests
// sans bord ne voient pas (revue adversariale E10.1, constats C7, C9, C10) :
//
//	C7   le bord d'une passe se lit UNE fois : `kill_positions` et le contexte de mort le partagent,
//	     et le repli `position_hors_emprise_ecartee` se compte une seule fois ;
//	C9   un tueur POSTHUME est pose par la tenue ou par le siege, pas seulement par un echantillon a la
//	     tolerance de l'instant de sa mort ;
//	C10  deux sieges d'un meme joueur a la tolerance, d'horodatages DIFFERENTS, refusent la lecture
//	     meme quand le bord est lu (sinon la tenue en choisirait un).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestLeBordPartageNeCompteLeRepliHorsEmpriseQuUneFois : un match dont des echantillons de vehicule
// sont hors de l'emprise jouee. Les deux lecteurs d'une passe (placement des morts, contexte de mort)
// recoivent la MEME lecture partagee : le repli se compte une fois, et le lieu rendu est inchange.
func TestLeBordPartageNeCompteLeRepliHorsEmpriseQuUneFois(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	vehicules := pisteMonde(900, 9_000, 12_000, 50, 50)
	for _, ms := range []int64{9_100, 9_200, 9_300} {
		vehicules = append(vehicules, posMonde(901, ms, 1e6, 1e6))
	}
	attendus := 0
	{
		gardes, ech, nais := ecarterVehiculesHorsEmprise(VehicleScan{Scanned: true, Positions: vehicules},
			empriseDesAxes(axesDesPositions(pos)), nil)
		attendus = ech + nais
		if attendus == 0 || len(gardes.Positions) >= len(vehicules) {
			t.Fatalf("fixture : aucun echantillon hors emprise (%d ecartes)", attendus)
		}
	}
	bord := LectureDuBord{Lue: true, Occupation: []types.VehicleOccupancy{monteeA(9_200)}, Vehicules: vehicules}.Partagee()
	fb := fallback.NouveauCompteur()
	out, _ := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
		Kills: []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}, Morts: journalDuSilence(),
		Fallbacks: fb, Bord: bord})
	contextes := ContextesDesMorts(EntreeContexteMorts{Positions: pos, Registre: reg,
		Journal: journalDuSilence(), Equipes: map[uint64]int{111: 0, 333: 0, 222: 1},
		Fallbacks: fb, Bord: bord})
	if got := fb.Compte(fallback.NomPositionHorsEmpriseEcartee); got != attendus {
		t.Errorf("repli compte %d fois, attendu %d (une seule lecture du bord par passe)", got, attendus)
	}
	if len(out) != 1 || out[0].Victim == nil || out[0].Victim.X != 50 {
		t.Fatalf("la victime devait etre posee au siege (50, 50) : %+v", out)
	}
	if len(contextes) != 1 || contextes[0].Lieu.X != 50 {
		t.Errorf("le contexte devait lire le meme lieu que kill_positions : %+v", contextes)
	}
}

// posthumeAVantage : le tueur 111 (siege 1) meurt a 9 s ; son dernier echantillon est a 8,7 s, hors
// de la tolerance de sa mort. La victime 222 (siege 2) meurt a 10 s sous sa grenade.
func posthumeAVantage() ([]grammar.BipedPosition, []KillRef, []MortDuJournal, IdentityRegistry) {
	pos := []grammar.BipedPosition{
		posAt(1, 8_500_000, 7, 8, 0), posAt(1, 8_700_000, 7, 8, 0),
		posAt(2, 10_000_000, 4, 5, 0),
	}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 9_000}, {VictimeXUID: 222, TempsMS: 10_000}}
	return pos, kills, morts, regPlat(map[uint32]uint64{1: 111, 2: 222})
}

// TestKillPositions_TueurPosthumePoseParLaTenue : sans bord, la mort du tueur n'a aucun echantillon a
// la tolerance, il reste nil ; avec le bord lu, la tenue le pose a son dernier echantillon.
func TestKillPositions_TueurPosthumePoseParLaTenue(t *testing.T) {
	pos, kills, morts, reg := posthumeAVantage()
	sans, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts})
	if len(sans) != 1 || sans[0].Killer != nil || rep.PosthumousUnplaced != 1 {
		t.Fatalf("sans bord le tueur posthume reste nil : %+v (%+v)", sans, rep)
	}
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts,
		Bord: LectureDuBord{Lue: true}})
	if len(got) != 1 || got[0].Killer == nil || got[0].Killer.X != 7 || got[0].Killer.Y != 8 {
		t.Fatalf("tueur posthume attendu en (7, 8) par la tenue : %+v", got)
	}
	if rep.Posthumous != 1 || rep.PosthumousUnplaced != 0 {
		t.Errorf("rapport %+v, attendu 1 posthume pose", rep)
	}
}

// TestKillPositions_TueurPosthumePoseParLeSiege : le tueur est monte a bord du vehicule 900 apres son
// dernier echantillon ; sa mort a 9 s est lue au siege du vehicule.
func TestKillPositions_TueurPosthumePoseParLeSiege(t *testing.T) {
	pos, kills, morts, reg := posthumeAVantage()
	bord := LectureDuBord{Lue: true,
		Occupation: []types.VehicleOccupancy{{Slot: 1, TimestampUS: 8_750_000, Attached: true, ParentSlot: 900}},
		Vehicules:  pisteMonde(900, 8_900, 9_100, 50, 50)}
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts, Bord: bord})
	if len(got) != 1 || got[0].Killer == nil || got[0].Killer.X != 50 || got[0].Killer.Y != 50 {
		t.Fatalf("tueur posthume attendu en (50, 50) au siege : %+v", got)
	}
	if rep.Posthumous != 1 {
		t.Errorf("rapport %+v, attendu 1 posthume pose", rep)
	}
}

// TestLectureDuLieu_DeuxSiegesALaToleranceRefusentMemeAvecLeBord : le joueur 111 a deux sieges dont
// les echantillons tombent a la tolerance de l'instant (9,95 s et 10,05 s) : deux corps pour un joueur,
// la lecture se tait. Sans la garde, la tenue choisirait le dernier echantillon d'avant l'instant.
func TestLectureDuLieu_DeuxSiegesALaToleranceRefusentMemeAvecLeBord(t *testing.T) {
	pos := []grammar.BipedPosition{
		posAt(1, 9_950_000, 5, 0, 0), posAt(3, 10_050_000, 6, 0, 0),
		posAt(2, 10_000_000, 20, 0, 0),
	}
	reg := regPlat(map[uint32]uint64{1: 111, 3: 111, 2: 222})
	kills := []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills,
		Bord: LectureDuBord{Lue: true}})
	if len(got) != 1 {
		t.Fatalf("le frag devait s'ecrire (tueur place) : %+v", got)
	}
	if got[0].Victim != nil || rep.Held != 0 {
		t.Errorf("victime %+v (tenus %d) : deux sieges a la tolerance, attendu nil", got[0].Victim, rep.Held)
	}
	if got[0].Killer == nil {
		t.Error("le tueur, a un seul siege, devait etre place")
	}
}
