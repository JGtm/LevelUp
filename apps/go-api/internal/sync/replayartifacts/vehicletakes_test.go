package replayartifacts

// vehicletakes_test.go — la projection pure de la ressource vehicules et son cablage (plan Emprise
// vehicules, lot L7.2). L'ecriture en base vit dans vehicletakes_integration_test.go.

import (
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/persist"
)

func TestProjeterPrisesDeVehicules_UnePasseValideAuPersister(t *testing.T) {
	doc := vhDocumentPur("pur1")
	lect := VehicleFragsLecture{Read: true, Frags: []replay.VehicleFragRef{
		{XUID: "a1", TimeMS: 5000 + 1500}, {XUID: "a1", TimeMS: 5000 + 9000}}}
	b := ProjeterPrisesDeVehicules("pur1", doc, lect)
	if err := persist.ValidateVehicleTakesBatch(b); err != nil {
		t.Fatalf("passe refusee par le persister : %v", err)
	}
	if !b.Measured || b.FragsTotal != 2 || b.FragsUnmatched != 1 || b.EpisodesNoXUID != 1 {
		t.Errorf("passe : %+v", b)
	}
}

func TestProjeterPrisesDeVehicules_NonMesureEtFragsNonLus(t *testing.T) {
	doc := vhDocumentPur("pur2")
	doc.SchemaVersion = 61
	b := ProjeterPrisesDeVehicules("pur2", doc, VehicleFragsLecture{Read: true})
	if err := persist.ValidateVehicleTakesBatch(b); err != nil {
		t.Fatalf("passe non mesuree refusee : %v", err)
	}
	if b.Measured || len(b.Rows) != 0 || b.FragsRead || b.FragsReason != VehicleFragsNotMeasured {
		t.Errorf("passe non mesuree attendue : %+v", b)
	}
	// Mesure, mais frags non lus : la raison de la lecture voyage jusqu'a la passe.
	b2 := ProjeterPrisesDeVehicules("pur3", vhDocumentPur("pur3"),
		VehicleFragsLecture{Reason: VehicleFragsNoClassifier})
	if err := persist.ValidateVehicleTakesBatch(b2); err != nil {
		t.Fatalf("passe refusee : %v", err)
	}
	if !b2.Measured || b2.FragsRead || b2.FragsReason != VehicleFragsNoClassifier || len(b2.Rows) == 0 {
		t.Errorf("prises attendues, frags non lus (no_classifier) : %+v", b2)
	}
}

func TestProjeterPrisesDeVehicules_SansOrigineLesFragsSontNonLus(t *testing.T) {
	doc := vhDocumentPur("pur4")
	doc.OriginMs = nil
	b := ProjeterPrisesDeVehicules("pur4", doc, VehicleFragsLecture{Read: true,
		Frags: []replay.VehicleFragRef{{XUID: "a1", TimeMS: 6500}}})
	if err := persist.ValidateVehicleTakesBatch(b); err != nil {
		t.Fatalf("passe refusee : %v", err)
	}
	if b.FragsRead || b.FragsReason != "origin_missing" || b.FragsTotal != 0 {
		t.Errorf("sans origine, aucun appariement : %+v", b)
	}
}

// vhDocumentPur : le document de test sans dependance a la base (duplique a dessein en miniature :
// le fichier d'integration porte la sienne sous tag).
func vhDocumentPur(matchID string) *replay.ReplayDocument {
	o := int64(5000)
	t0 := 0
	return &replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID, FrameIntervalMS: 100, OriginMs: &o,
		Coverage: &replay.Coverage{Vehicles: &replay.VehicleCoverage{Scanned: true}},
		Roster:   []replay.RosterEntry{{XUID: "a1", Team: &t0}},
		Vehicles: []replay.VehicleTrack{
			{Slot: 1, Gen: 1, Family: "warthog", Rides: []replay.VehicleRide{
				{XUID: "a1", T0: 10, T1: 20, Src: "film"}}},
			{Slot: 2, Gen: 1, Family: "mongoose", Rides: []replay.VehicleRide{
				{XUID: "", T0: 50, T1: 60, Src: "film"}}},
		},
	}
}

// TestPorteCapabiliteVehicules_EstCABLEE — le cablage existe : la porte est declaree ET franchie
// avant toute lecture, la capability est au vocabulaire canonique, et Deriver prepare (lectures)
// AVANT le premier segment d'ecriture puis ecrit.
func TestPorteCapabiliteVehicules_EstCABLEE(t *testing.T) {
	src, err := os.ReadFile("vehicletakes.go")
	if err != nil {
		t.Fatalf("lecture de vehicletakes.go : %v", err)
	}
	for _, attendu := range []string{
		"porteCapability(ctx, d, games.CapFilmVehicleUsage,",
		"armee, incident := capabiliteVehiculesArmee(ctx, d)", "if !armee {",
	} {
		if !strings.Contains(string(src), attendu) {
			t.Errorf("la porte de capability des vehicules n'est plus cablee : %q absent", attendu)
		}
	}
	if !games.IsKnownCapabilityKey(games.CapFilmVehicleUsage) {
		t.Error("film.vehicle_usage absente de AllCapabilityKeys()")
	}
	der, err := os.ReadFile("derivations.go")
	if err != nil {
		t.Fatalf("lecture de derivations.go : %v", err)
	}
	txt := string(der)
	iPrep := strings.Index(txt, "vehicules := preparerPrisesDeVehicules(ctx, d, b, lus)")
	iAcq := strings.Index(txt, "d.AcquireWriter = w.acquerir")
	iEcr := strings.Index(txt, "ecrirePrisesDeVehicules(ctx, d, b, vehicules)")
	if iPrep < 0 || iEcr < 0 {
		t.Fatal("Deriver ne prepare plus ou n'ecrit plus la ressource vehicules")
	}
	if iAcq < 0 || iPrep > iAcq {
		t.Error("les frags de mort doivent se lire AVANT l'acquisition du segment d'ecriture (panne L4.1)")
	}
}
