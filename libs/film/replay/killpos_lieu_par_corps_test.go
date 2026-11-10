package replay

// killpos_lieu_par_corps_test.go — LE LIEU D'UN JOUEUR QUAND LE SIEGE CHANGE D'OCCUPANT OU QUE LE
// VEHICULE NE REPLIQUE PAS :
//
//	R-AUTRE-CORPS   un siege recycle ne place pas un joueur par l'echantillon du corps suivant ;
//	R-TOLERANCE     une descente a la tolerance de l'instant laisse le joueur a bord ;
//	R-PIECE         un occupant de piece montee sans echantillon est au chassis qui la porte ;
//	R-PIECE-NAISS   un voisin de slot qui n'est pas ne avec la piece n'est pas son porteur ;
//	R-FIXE          un poste de tir fixe reste ou le film le cree ;
//	R-AUTRE-FAMILLE un vehicule sans echantillon d'une autre famille reste sans lieu.
//
// Et le meme silence a l'entame (`BuildKillOpenings`) : tenue, siege, garde de meme vie.

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func TestLieu_AutreCorpsDuSiegeNePlacePasLeJoueur(t *testing.T) {
	// Le siege 1 porte 111 (generation 1, [1 s..4 s]), puis 222 (generation 2, creee a 10,10 s, premier
	// echantillon a 10,11 s). 111 meurt a 10 s dans le siege 2 ; a cet instant le corps de generation 1
	// tient encore le siege 1, mais l'echantillon a la tolerance est celui du corps de generation 2.
	pos := append(pisteMonde(1, 1_000, 4_000, 1, 1), pisteMonde(1, 10_110, 15_000, 7, 7)...)
	pos = append(pos, pisteMonde(2, 8_000, 9_900, 5, 0)...)
	pos = append(pos, pisteMonde(3, 100, 20_000, 20, 0)...)
	reg := BuildIdentityRegistry(context.Background(), IdentityInput{Positions: pos,
		BipedCreations: []grammar.BipedCreation{creationGen(1, 500_000, 0, 1), creationGen(1, 10_100_000, 1, 2),
			creationDe(2, 8_000_000, 0), creationDe(3, 100_000, 2)},
		PlayerIndices: indexDe(111, 222, 333)})
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
		Kills: []KillRef{{KillerXUID: 333, VictimXUID: 111, TimeMS: 10_000}},
		Morts: []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}, Bord: LectureDuBord{Lue: true}})
	if len(got) != 1 || got[0].Victim == nil || got[0].Victim.X != 5 || got[0].Victim.Y != 0 {
		t.Fatalf("victime attendue au siege 2 (5, 0) : %+v (%+v)", got, rep)
	}
	if rep.Both != 1 {
		t.Errorf("rapport %+v, attendu les deux cotes places", rep)
	}
}

func TestLieu_DescenteALaToleranceResteABord(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	descente := types.VehicleOccupancy{Slot: 1, TimestampUS: 9_950_000}
	kp, rep := placerLeFragDuSilence(t, pos, reg, bordDuVehicule900(monteeA(9_200), descente), journalDuSilence())
	if kp.Victim == nil || kp.Victim.X != 50 || kp.Victim.Y != 50 || rep.Seated != 1 || rep.Disembarked != 0 {
		t.Fatalf("victime %+v (sieges %d, debarques %d) : une descente a la tolerance laisse a bord",
			kp.Victim, rep.Seated, rep.Disembarked)
	}
}

// creationVehicule : un record de creation `ti=40` portant son mot d'identite de chassis.
func creationVehicule(slot uint32, tUS uint64, x, y float32, chassis uint32) types.EquipmentCreation {
	c := types.EquipmentCreation{Slot: slot, Gen: 1, TimestampUS: tUS, X: x, Y: y}
	c.MPPPresent[grammar.MPPWord32] = true
	c.MPPVal[grammar.MPPWord32] = uint64(chassis)
	return c
}

// bordDeLaPiece905 : 111 monte a 9,2 s sur la piece 905 ; aucun echantillon de piece.
func bordDeLaPiece905(creations ...types.EquipmentCreation) LectureDuBord {
	monte := types.VehicleOccupancy{Slot: 1, TimestampUS: 9_200_000, Attached: true, ParentSlot: 905}
	return LectureDuBord{Lue: true, Occupation: []types.VehicleOccupancy{monte},
		Vehicules: pisteMonde(906, 9_000, 12_000, 50, 50), Creations: creations}
}

const (
	chassisLAAG     uint32 = 0xdd7f9102 // piece montee, porteur Warthog
	chassisWarthog  uint32 = 0x00002705 // familleWarthog
	chassisFixe     uint32 = 0xb857fb95 // familleTourelleFixe
	chassisMongoose uint32 = 0xaf31ab1a // familleMongoose, sans piece
)

func TestLieu_PieceMonteeSansEchantillonEstAuChassis(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	bord := bordDeLaPiece905(creationVehicule(905, 1_000_000, 10, 10, chassisLAAG),
		creationVehicule(906, 1_050_000, 10, 10, chassisWarthog))
	fb := fallback.NouveauCompteur()
	out, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
		Kills: []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}, Morts: journalDuSilence(), Fallbacks: fb, Bord: bord})
	if len(out) != 1 || out[0].Victim == nil || out[0].Victim.X != 50 || out[0].Victim.Y != 50 || rep.Seated != 1 {
		t.Fatalf("victime %+v (sieges %d, a bord sans vehicule %d) : attendu le chassis porteur (50, 50)",
			out, rep.Seated, rep.AboardNoVehicle)
	}
	if n := fb.Compte(fallback.NomTourellePorteurVoisinDeSlot); n != 1 {
		t.Errorf("repli du voisin de slot compte %d fois, attendu 1", n)
	}
}

func TestLieu_PieceMonteeVoisinPasNeAvecElleNEstPasPorteur(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	// Le voisin de slot est un Warthog ne ailleurs : 200 m plus loin.
	bord := bordDeLaPiece905(creationVehicule(905, 1_000_000, 10, 10, chassisLAAG),
		creationVehicule(906, 1_050_000, 210, 10, chassisWarthog))
	kp, rep := placerLeFragDuSilence(t, pos, reg, bord, journalDuSilence())
	if kp.Victim != nil || rep.AboardNoVehicle != 1 {
		t.Fatalf("victime %+v, a bord sans vehicule %d : le voisin n'est pas le porteur", kp.Victim, rep.AboardNoVehicle)
	}
}

func TestLieu_PosteFixeSansEchantillonEstOuLeFilmLeCree(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	bord := bordDeLaPiece905(creationVehicule(905, 1_000_000, -30, 7, chassisFixe))
	bord.Vehicules = nil
	kp, rep := placerLeFragDuSilence(t, pos, reg, bord, journalDuSilence())
	if kp.Victim == nil || kp.Victim.X != -30 || kp.Victim.Y != 7 || rep.Seated != 1 {
		t.Fatalf("victime %+v (sieges %d) : attendu le poste fixe (-30, 7)", kp.Victim, rep.Seated)
	}
}

func TestLieu_VehiculeDAutreFamilleSansEchantillonResteSansLieu(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	bord := bordDeLaPiece905(creationVehicule(905, 1_000_000, 10, 10, chassisMongoose))
	bord.Vehicules = nil
	kp, rep := placerLeFragDuSilence(t, pos, reg, bord, journalDuSilence())
	if kp.Victim != nil || rep.AboardNoVehicle != 1 {
		t.Fatalf("victime %+v, a bord sans vehicule %d : attendu nil compte", kp.Victim, rep.AboardNoVehicle)
	}
}

// filmDeLEntame : 222 tue 111 a 10 s ; 111 (slot 1) est echantillonne jusqu'a 8 s, se tait, reapparait
// au flux a 10 s (meme sejour de moins de 5 s) ; 222 (slot 2) echantillonne jusqu'a 20 s.
func filmDeLEntame() ([]grammar.BipedPosition, IdentityRegistry) {
	pos := pisteMonde(1, 200, 8_000, 5, 0)
	pos = append(pos, posMonde(1, 10_000, 5, 0))
	pos = append(pos, pisteMonde(2, 200, 20_000, 20, 0)...)
	reg := BuildIdentityRegistry(context.Background(), IdentityInput{Positions: pos,
		BipedCreations: []grammar.BipedCreation{creationDe(1, 100_000, 0), creationDe(2, 100_000, 1)},
		PlayerIndices:  indexDe(111, 222)})
	return pos, reg
}

func TestEntame_LeSilenceSeLitCommeAuCoupFatal(t *testing.T) {
	kills := []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}
	pos, reg := filmDeLEntame()
	sans, rep := BuildKillOpenings(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts})
	if len(sans) != 1 || sans[0].Victim != nil {
		t.Fatalf("sans bord : %+v (%+v), attendu la victime nil (silence a l'entame)", sans, rep)
	}
	got, rep := BuildKillOpenings(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts,
		Bord: LectureDuBord{Lue: true}})
	if len(got) != 1 || got[0].Victim == nil || got[0].Victim.X != 5 || got[0].TimeMS != 10_000 {
		t.Fatalf("tenue : %+v (%+v), attendu la victime en (5, 0) au kill", got, rep)
	}
	if rep.OpeningOutOfLife != 0 {
		t.Errorf("rapport %+v, attendu aucun cote hors vie", rep)
	}
}

func TestEntame_LeSiegeSeGardeParLeJournalPasParLeSiege(t *testing.T) {
	kills := []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}
	pos, reg := filmDuSilence(8_000)
	bord := bordDuVehicule900(monteeA(8_000))
	bord.Vehicules = pisteMonde(900, 7_000, 12_000, 50, 50)
	got, rep := BuildKillOpenings(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts, Bord: bord})
	if len(got) != 1 || got[0].Victim == nil || got[0].Victim.X != 50 || got[0].Victim.Y != 50 {
		t.Fatalf("siege a l'entame : %+v (%+v), attendu (50, 50)", got, rep)
	}
	if rep.OpeningOutOfLife != 0 {
		t.Errorf("rapport %+v, attendu aucun cote hors vie", rep)
	}
	// Une mort du joueur entre l'entame et le kill : ce n'est plus sa vie.
	autre := append([]MortDuJournal{{VictimeXUID: 111, TempsMS: 9_000}}, morts...)
	got, rep = BuildKillOpenings(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: autre, Bord: bord})
	if len(got) != 1 || got[0].Victim != nil || rep.OpeningOutOfLife != 1 {
		t.Fatalf("mort entre l'entame et le kill : %+v (%+v), attendu la victime ecartee et comptee", got, rep)
	}
}

func TestEntame_InstantAvantLOrigineNeLitPasLeSilence(t *testing.T) {
	pos, reg := filmDeLEntame()
	kills := []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 1_000}}
	got, rep := BuildKillOpenings(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills,
		Bord: LectureDuBord{Lue: true}})
	if len(got) != 0 || rep.Dropped != 1 {
		t.Fatalf("kill %+v (%+v) : une entame avant l'origine du film ne place personne", got, rep)
	}
}

// TestLieu_LaTenueNePrendPasLeDernierEchantillonDUnAutreCorps : le siege 1 porte 111 (generation 1,
// [1 s..3 s]), un corps d index 5 sans xuid (generation 2, [9 s..14 s]), puis 111 de nouveau (generation 3,
// creee a 20 s, silencieuse jusqu a 27 s). 111 meurt a 25 s sans mort anterieure : le dernier echantillon du
// siege est celui de l'autre corps, il ne la tient pas la.
func TestLieu_LaTenueNePrendPasLeDernierEchantillonDUnAutreCorps(t *testing.T) {
	pos := append(pisteMonde(1, 1_000, 3_000, 1, 1), pisteMonde(1, 9_000, 14_000, 40, 40)...)
	pos = append(pos, pisteMonde(1, 27_000, 30_000, 2, 2)...)
	pos = append(pos, pisteMonde(3, 100, 40_000, 20, 0)...)
	reg := BuildIdentityRegistry(context.Background(), IdentityInput{Positions: pos,
		BipedCreations: []grammar.BipedCreation{creationGen(1, 500_000, 0, 1), creationGen(1, 8_500_000, 5, 2),
			creationGen(1, 20_000_000, 0, 3), creationDe(3, 100_000, 2)},
		PlayerIndices: indexDe(111, 222, 333)})
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
		Kills: []KillRef{{KillerXUID: 333, VictimXUID: 111, TimeMS: 25_000}},
		Morts: []MortDuJournal{{VictimeXUID: 111, TempsMS: 25_000}}, Bord: LectureDuBord{Lue: true}})
	if len(got) != 1 || got[0].Victim != nil || rep.Held != 0 {
		t.Fatalf("victime %+v (tenus %d) : le dernier echantillon est celui d'un autre corps, attendu nil", got, rep.Held)
	}
}

// TestLieu_UnSiegeEnCollisionGardeSonEchantillon : le siege 100 porte deux records de la MEME generation et
// d'index differents, 111 puis 222 : collision, aucun corps etabli. Un echantillon juste apres la fin de la
// vie de 222 (le corps replique encore apres sa mort) reste le sien : le silence du registre hors de la vie
// ne prouve pas un autre occupant.
func TestLieu_UnSiegeEnCollisionGardeSonEchantillon(t *testing.T) {
	pos := append(pisteMonde(100, 1_000, 3_000, 1, 1), pisteMonde(100, 9_000, 12_000, 7, 7)...)
	reg := BuildIdentityRegistry(context.Background(), IdentityInput{Positions: pos,
		BipedCreations: []grammar.BipedCreation{creationGen(100, 500_000, 0, 1), creationGen(100, 8_000_000, 1, 1)},
		PlayerIndices:  indexDe(111, 222)})
	if !reg.collisionDeSiege(100) {
		t.Fatal("fixture : le siege 100 doit etre une collision")
	}
	l := lectureALInstant{registre: reg}
	if !l.memeJoueur(100, 12_060_000, 222) {
		t.Fatal("echantillon a la fin de vie + 60 ms sur un siege en collision : attendu garde")
	}
}
