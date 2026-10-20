package replay

// killpos_a_l_instant_test.go — LA LECTURE DU LIEU A L'INSTANT QUAND LE FILM SE TAIT (D-16) : tenue,
// siege, et le meme lieu dans `kill_positions` et dans le contexte de mort.
//
//	R-TENUE          victime immobile, silencieuse 800 ms avant sa mort : tenue au dernier
//	                 echantillon ; sans lecture du bord, nil (la condition « hors embarquement » ne
//	                 se verifie pas) ;
//	R-MORT           la tenue ne traverse pas une mort du journal ;
//	R-SIEGE          une montee a bord court sur le silence : le lieu est celui du vehicule ;
//	R-SANS-VEHICULE  le vehicule n'a pas d'echantillon a la tolerance : nil, compte ;
//	R-DEBARQUE       montee puis descente pendant le silence : nil, compte ;
//	R-RETOUR         une montee suivie d'un retour du bipede au flux ne met pas a bord ;
//	R-UN-LIEU        le lieu du contexte de mort est celui de `kill_positions`, a 1e-6 pres, pour
//	                 l'echantillon, la tenue et le siege — sous une origine du film distincte du calage
//	                 du registre.

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// filmDuSilence : la victime 111 (slot 1) immobile en (5, 0), echantillonnee de 0,2 s a `finVictimeMS`,
// tuee a 10 s par 222 (slot 2, en (20, 0) jusqu'a 20 s) ; 333 (slot 3), coequipier de 111, en (5, 4).
func filmDuSilence(finVictimeMS int64) ([]grammar.BipedPosition, IdentityRegistry) {
	var pos []grammar.BipedPosition
	pos = append(pos, pisteMonde(1, 200, finVictimeMS, 5, 0)...)
	pos = append(pos, pisteMonde(2, 200, 20_000, 20, 0)...)
	pos = append(pos, pisteMonde(3, 200, 20_000, 5, 4)...)
	reg := BuildIdentityRegistry(context.Background(), IdentityInput{
		Positions: pos,
		BipedCreations: []grammar.BipedCreation{creationDe(1, 100_000, 0), creationDe(2, 100_000, 1),
			creationDe(3, 100_000, 2)},
		PlayerIndices: indexDe(111, 222, 333),
	})
	return pos, reg
}

// placerLeFragDuSilence place le frag de 222 sur 111 a 10 s.
func placerLeFragDuSilence(t *testing.T, pos []grammar.BipedPosition, reg IdentityRegistry, bord LectureDuBord,
	journal []MortDuJournal) (KillPosition, KillPosReport) {
	t.Helper()
	out, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
		Kills: []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}}, Morts: journal, Bord: bord})
	if len(out) != 1 {
		t.Fatalf("frag non ecrit : %+v", rep)
	}
	return out[0], rep
}

func journalDuSilence() []MortDuJournal { return []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}} }

func TestTenueDuDernierEchantillonDansLeSilence(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	kp, rep := placerLeFragDuSilence(t, pos, reg, LectureDuBord{Lue: true}, journalDuSilence())
	if kp.Victim == nil || kp.Victim.X != 5 || kp.Victim.Y != 0 || rep.Held != 1 {
		t.Fatalf("victime %+v, tenus %d : attendu (5, 0) tenu", kp.Victim, rep.Held)
	}
	if kp.Killer == nil || kp.Killer.X != 20 {
		t.Fatalf("tueur %+v : attendu son echantillon a l'instant", kp.Killer)
	}
	kp, rep = placerLeFragDuSilence(t, pos, reg, LectureDuBord{}, journalDuSilence())
	if kp.Victim != nil || rep.Held != 0 {
		t.Fatalf("sans lecture du bord : victime %+v, tenus %d, attendu nil", kp.Victim, rep.Held)
	}
}

func TestTenueNeTraversePasUneMort(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	journal := append(journalDuSilence(), MortDuJournal{VictimeXUID: 111, TempsMS: 9_250})
	kp, rep := placerLeFragDuSilence(t, pos, reg, LectureDuBord{Lue: true}, journal)
	if kp.Victim != nil || rep.Held != 0 {
		t.Fatalf("victime %+v, tenus %d : une mort a 9,25 s clot la vie, attendu nil", kp.Victim, rep.Held)
	}
}

// bordDuVehicule900 : 111 monte a 9,2 s dans le vehicule 900, echantillonne en (50, 50) de 9 s a 12 s.
func bordDuVehicule900(occ ...types.VehicleOccupancy) LectureDuBord {
	var veh []grammar.BipedPosition
	for _, p := range pisteMonde(900, 9_000, 12_000, 50, 50) {
		veh = append(veh, p)
	}
	return LectureDuBord{Lue: true, Occupation: occ, Vehicules: veh}
}

func monteeA(tMS int64) types.VehicleOccupancy {
	return types.VehicleOccupancy{Slot: 1, TimestampUS: uint64(tMS) * 1000, Attached: true, ParentSlot: 900} //nolint:gosec // test
}

func TestSiegeDuVehiculeOccupe(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	kp, rep := placerLeFragDuSilence(t, pos, reg, bordDuVehicule900(monteeA(9_200)), journalDuSilence())
	if kp.Victim == nil || kp.Victim.X != 50 || kp.Victim.Y != 50 || rep.Seated != 1 || rep.Held != 0 {
		t.Fatalf("victime %+v (sieges %d, tenus %d) : attendu (50, 50) au siege", kp.Victim, rep.Seated, rep.Held)
	}
}

func TestABordDUnVehiculeSansEchantillon(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	bord := bordDuVehicule900(monteeA(9_200))
	bord.Vehicules = pisteMonde(900, 0, 5_000, 50, 50)
	kp, rep := placerLeFragDuSilence(t, pos, reg, bord, journalDuSilence())
	if kp.Victim != nil || rep.AboardNoVehicle != 1 {
		t.Fatalf("victime %+v, a bord sans vehicule %d : attendu nil compte", kp.Victim, rep.AboardNoVehicle)
	}
}

func TestDebarquePendantLeSilence(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	descente := types.VehicleOccupancy{Slot: 1, TimestampUS: 9_600_000}
	kp, rep := placerLeFragDuSilence(t, pos, reg, bordDuVehicule900(monteeA(9_200), descente), journalDuSilence())
	if kp.Victim != nil || rep.Disembarked != 1 {
		t.Fatalf("victime %+v, debarques %d : attendu nil compte", kp.Victim, rep.Disembarked)
	}
}

func TestMonteeSuivieDUnRetourAuFluxNeMetPasABord(t *testing.T) {
	pos, reg := filmDuSilence(9_200)
	kp, rep := placerLeFragDuSilence(t, pos, reg, bordDuVehicule900(monteeA(7_000)), journalDuSilence())
	if kp.Victim == nil || kp.Victim.X != 5 || rep.Held != 1 || rep.Seated != 0 {
		t.Fatalf("victime %+v (tenus %d, sieges %d) : le bipede a replique apres la montee, attendu la tenue",
			kp.Victim, rep.Held, rep.Seated)
	}
}

// TestLeLieuDuContexteEstCeluiDeKillPositions — R-UN-LIEU. L'origine du film (80 ms avant le calage du
// registre) et une victime en marche separent l'ancienne lecture du contexte de celle de
// `kill_positions` ; les trois voies (echantillon, tenue, siege) doivent rendre le meme point.
func TestLeLieuDuContexteEstCeluiDeKillPositions(t *testing.T) {
	cas := []struct {
		nom      string
		finMS    int64
		enMarche bool
		bord     LectureDuBord
		voie     func(KillPosReport) int
	}{
		{"echantillon", 10_000, true, LectureDuBord{Lue: true}, func(r KillPosReport) int { return 1 - r.Held - r.Seated }},
		{"tenue", 9_200, false, LectureDuBord{Lue: true}, func(r KillPosReport) int { return r.Held }},
		{"siege", 9_200, false, bordDuVehicule900(monteeA(9_200)), func(r KillPosReport) int { return r.Seated }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			pos, reg := filmDuSilence(c.finMS)
			if c.enMarche {
				pos = append(pisteEnMarche(1, 200, c.finMS), pos[len(pisteMonde(1, 200, c.finMS, 5, 0)):]...)
				reg = BuildIdentityRegistry(context.Background(), IdentityInput{Positions: pos,
					BipedCreations: []grammar.BipedCreation{creationDe(1, 100_000, 0), creationDe(2, 100_000, 1),
						creationDe(3, 100_000, 2)}, PlayerIndices: indexDe(111, 222, 333)})
			}
			const origine = -80_000
			out, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg,
				Kills: []KillRef{{KillerXUID: 222, VictimXUID: 111, TimeMS: 10_000}},
				Morts: journalDuSilence(), OrigineUS: origine, Bord: c.bord})
			if len(out) != 1 || out[0].Victim == nil || c.voie(rep) != 1 {
				t.Fatalf("kill_positions : %+v (%+v)", out, rep)
			}
			ctx := contexteDe(t, ContextesDesMorts(EntreeContexteMorts{Positions: pos, Registre: reg,
				Journal: journalDuSilence(), Equipes: map[uint64]int{111: 0, 333: 0, 222: 1},
				OrigineDuFilmUS: origine, Bord: c.bord}), 111, 10_000)
			ref := out[0].Victim
			for nom, d := range map[string]float64{"X": ctx.Lieu.X - ref.X, "Y": ctx.Lieu.Y - ref.Y, "Z": ctx.Lieu.Z - ref.Z} {
				if math.Abs(d) > 1e-6 {
					t.Fatalf("lieu %+v != kill_positions %+v (ecart %s = %g)", ctx.Lieu, *ref, nom, d)
				}
			}
		})
	}
}

// pisteEnMarche echantillonne un slot toutes les 100 ms en le faisant avancer d'un metre par
// echantillon (x = t / 100) : deux instants a 100 ms d'ecart donnent deux lieux distincts.
func pisteEnMarche(slot uint32, deMS, aMS int64) []grammar.BipedPosition {
	out := []grammar.BipedPosition{}
	for t := deMS; t <= aMS; t += 100 {
		out = append(out, posMonde(slot, t, float32(t/100), 0))
	}
	return out
}
