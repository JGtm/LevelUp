package replay

// seuils_ms_cadence_test.go — LES SEUILS TEMPORELS SONT DES DUREES, PAS DES NOMBRES D'IMAGES.
//
// Chaque test rejoue la MEME situation reelle (les memes ecarts en millisecondes) a 100 ms et a
// 33 ms par image : la decision doit etre identique. Un seuil ecrit en images changerait de sens
// avec la cadence (un ecart de 150 ms serait refuse a 33 ms par un seuil « 2 images »).

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// cadencesDeTest : le pas historique et une cadence triple.
var cadencesDeTest = []int{100, 33}

func TestFramesOfMS(t *testing.T) {
	for _, c := range []struct {
		nom          string
		ms, interval int
		attendu      int
	}{
		{"200 ms a 100 ms", 200, 100, 2},
		{"100 ms a 100 ms", 100, 100, 1},
		{"300 ms a 100 ms", 300, 100, 3},
		{"200 ms a 33 ms (arrondi vers le bas)", 200, 33, 6},
		{"100 ms a 33 ms", 100, 33, 3},
		{"100 ms a 40 ms (jamais plus large que la duree)", 100, 40, 2},
		{"duree nulle", 0, 100, 0},
		{"duree negative", -50, 100, 0},
		{"pas absent = pas de construction", 200, 0, 2},
		{"pas negatif = pas de construction", 200, -1, 2},
		{"pas plus grand que la duree", 100, 250, 0},
	} {
		if got := framesOfMS(c.ms, c.interval); got != c.attendu {
			t.Errorf("%s : framesOfMS(%d, %d) = %d, attendu %d", c.nom, c.ms, c.interval, got, c.attendu)
		}
	}
}

// TestPasHistoriqueDistinctDuDefautDeConstruction : deux constantes, deux roles. Le pas historique
// est celui des documents publies sans `frameIntervalMs` ; le defaut est celui de la construction.
func TestPasHistoriqueDistinctDuDefautDeConstruction(t *testing.T) {
	if HistoricalFrameIntervalMS != 100 {
		t.Errorf("HistoricalFrameIntervalMS = %d, attendu 100 (documents publies sans frameIntervalMs)",
			HistoricalFrameIntervalMS)
	}
	if DefaultFrameIntervalMS != 100 {
		t.Errorf("DefaultFrameIntervalMS = %d, attendu 100", DefaultFrameIntervalMS)
	}
}

// TestToleranceDePositionDeZoneEnMsIndependanteDeLaCadence : un echantillon de position a 150 ms de
// l'action est retenu, un echantillon a 250 ms est refuse, a toute cadence.
func TestToleranceDePositionDeZoneEnMsIndependanteDeLaCadence(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	for _, interval := range cadencesDeTest {
		for _, c := range []struct {
			nom       string
			echantMS  int
			attribuee bool
		}{
			{"150 ms tolere", 150, true},
			{"250 ms refuse", 50, false},
		} {
			const actionMS = 300
			act := ObjectiveAction{T: actionMS / interval, XUID: "2533", Stat: "zone_secures", TimeMS: actionMS}
			tracks := []Track{track("2533", pointAt(c.echantMS/interval, 1, 0, 0))}
			res, cov := AttributeZones([]ObjectiveAction{act}, tracks, zones, nil,
				AttributeOptions{FrameIntervalMS: interval})
			checkInvariant(t, cov)
			if res[0].Attributed != c.attribuee {
				t.Errorf("pas %d ms, %s : attribuee = %v, attendu %v", interval, c.nom, res[0].Attributed, c.attribuee)
			}
		}
	}
}

// TestToleranceDePositionDeZoneParDefautVaut200ms : au pas de construction, le defaut vaut les deux
// frames historiques, et `DefaultMaxGapFrames` (cmd de mesure) en reste l'expression a ce pas.
func TestToleranceDePositionDeZoneParDefautVaut200ms(t *testing.T) {
	if DefaultMaxGapMS != 200 {
		t.Errorf("DefaultMaxGapMS = %d, attendu 200", DefaultMaxGapMS)
	}
	if DefaultMaxGapFrames != 2 {
		t.Errorf("DefaultMaxGapFrames = %d, attendu 2 (200 ms au pas de %d ms)", DefaultMaxGapFrames, DefaultFrameIntervalMS)
	}
	if got := (AttributeOptions{}).maxGapFrames(); got != 2 {
		t.Errorf("tolerance par defaut = %d frames, attendu 2", got)
	}
	if got := (AttributeOptions{FrameIntervalMS: 33}).maxGapFrames(); got != 6 {
		t.Errorf("tolerance par defaut a 33 ms = %d frames, attendu 6", got)
	}
	if got := (AttributeOptions{MaxGapFrames: 4, FrameIntervalMS: 33}).maxGapFrames(); got != 4 {
		t.Errorf("surcharge explicite en frames = %d, attendu 4", got)
	}
}

// TestFenetreTirProjectileEnMsIndependanteDeLaCadence : un tir a 100 ms de la naissance du projectile
// le lie, un tir a 200 ms non, a toute cadence.
func TestFenetreTirProjectileEnMsIndependanteDeLaCadence(t *testing.T) {
	for _, interval := range cadencesDeTest {
		const naissanceMS = 1000
		for _, c := range []struct {
			nom  string
			tirM int
			lie  bool
		}{
			{"tir 100 ms avant", naissanceMS - 100, true},
			{"tir 100 ms apres", naissanceMS + 100, true},
			{"tir 200 ms avant", naissanceMS - 200, false},
			{"tir 200 ms apres", naissanceMS + 200, false},
		} {
			projs := []Projectile{projectileNeA(naissanceMS/interval, 0, 0)}
			shots := []Shot{tirDe(c.tirM/interval, 1, "A", 0, 0)}
			linked, _ := linkShotsToProjectiles(projs, shots, nil, interval)
			if (linked == 1) != c.lie {
				t.Errorf("pas %d ms, %s : lie = %v, attendu %v", interval, c.nom, linked == 1, c.lie)
			}
		}
	}
}

// TestNaissanceCommuneDeTourelleEnMsIndependanteDeLaCadence : un porteur ne 100 ms apres la piece est
// ne avec elle, un porteur ne 200 ms apres non, a toute cadence.
func TestNaissanceCommuneDeTourelleEnMsIndependanteDeLaCadence(t *testing.T) {
	for _, interval := range cadencesDeTest {
		for _, c := range []struct {
			nom      string
			ecartMS  int
			ensemble bool
		}{
			{"ne 100 ms apres", 100, true},
			{"ne 200 ms apres", 200, false},
		} {
			porteur := vtChassis(101, familleWarthog)
			porteur.T0 = c.ecartMS / interval
			porteur.Spawn = nil
			tracks := []VehicleTrack{vtPiece(100, vtLAAG), porteur}
			anchors := vehicleBoardingAnchors{clock: replayClock{step: uint64(interval) * 1000}}
			tally := poseTurretsOnCarriers(tracks, anchors, fallback.NouveauCompteur())
			if got := tracks[0].Carrier != nil; got != c.ensemble {
				t.Errorf("pas %d ms, %s : porteur pose = %v, attendu %v", interval, c.nom, got, c.ensemble)
			}
			if !c.ensemble && tally.birthMismatch != 1 {
				t.Errorf("pas %d ms, %s : refus compte = %d, attendu 1", interval, c.nom, tally.birthMismatch)
			}
		}
	}
}

// TestToleranceDeNaissanceDesObjetsEngendresEnMsIndependanteDeLaCadence : une trajectoire nee 300 ms
// apres la pile qui nomme l'objet le situe, une nee 400 ms apres non, a toute cadence.
func TestToleranceDeNaissanceDesObjetsEngendresEnMsIndependanteDeLaCadence(t *testing.T) {
	if toleranceDeNaissanceMS != 300 {
		t.Errorf("toleranceDeNaissanceMS = %d, attendu 300", toleranceDeNaissanceMS)
	}
	for _, interval := range cadencesDeTest {
		const t0MS = 30_000
		for _, c := range []struct {
			nom     string
			ecartMS int
			situe   bool
		}{
			{"300 ms apres", 300, true},
			{"300 ms avant", -300, true},
			{"400 ms apres", 400, false},
			{"400 ms avant", -400, false},
		} {
			cands := []reposEngendre{cand((t0MS+c.ecartMS)/interval, 7)}
			if _, ok := reposNeAPeuPres(cands, t0MS/interval, interval); ok != c.situe {
				t.Errorf("pas %d ms, %s : situe = %v, attendu %v", interval, c.nom, ok, c.situe)
			}
		}
	}
}

// TestPasImpossibleDeProjectileEnVitesseIndependantDeLaCadence : un projectile a 60 m/s se lit, un
// projectile a 150 m/s est coupe, a toute cadence.
func TestPasImpossibleDeProjectileEnVitesseIndependantDeLaCadence(t *testing.T) {
	if projectileMaxStepM(100_000) != 10 {
		t.Errorf("a 100 ms le pas maximal vaut %v m, attendu 10", projectileMaxStepM(100_000))
	}
	for _, interval := range cadencesDeTest {
		pasUS := uint64(interval) * 1000
		for _, c := range []struct {
			nom     string
			vitesse float64 // m/s
			coupe   bool
		}{
			{"60 m/s", 60, false},
			{"150 m/s", 150, true},
		} {
			pasM := float32(c.vitesse * float64(interval) / 1000)
			pts := make([]types.ProjectileSample, 6)
			for i := range pts {
				pts[i] = types.ProjectileSample{TimestampUS: 1_000_000 + uint64(i)*pasUS, X: float32(i) * pasM}
			}
			pts[5].AtRest = true
			tracks := []types.ProjectileTrack{{Slot: 4096, Gen: 1, Pts: pts}}
			out, _, tronquees := buildProjectiles(tracks, 1_000_000, pasUS, nil, nil)
			if (tronquees == 1) != c.coupe {
				t.Errorf("pas %d ms, %s : coupe = %v, attendu %v", interval, c.nom, tronquees == 1, c.coupe)
			}
			if !c.coupe && (len(out) != 1 || len(out[0].P) != 6) {
				t.Errorf("pas %d ms, %s : vol incomplet : %+v", interval, c.nom, out)
			}
		}
	}
}
