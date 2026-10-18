package observability

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestMatchSteps_DesarmeNeCollecteRien(t *testing.T) {
	EnableMatchSteps(false)
	ReportMatchStep("halo_infinite", "m1", domain.MatchStepStatsWritten, "")
	if got := DrainMatchSteps(); len(got) != 0 {
		t.Fatalf("collecteur désarmé : %d instants", len(got))
	}
}

func TestMatchSteps_CaptureVidageEtRemiseEnTete(t *testing.T) {
	EnableMatchSteps(true)
	t.Cleanup(func() { EnableMatchSteps(false) })

	fin := time.Date(2026, 10, 17, 20, 0, 0, 0, time.FixedZone("Paris", 2*3600))
	ReportMatchStepAt("halo_infinite", "m1", domain.MatchStepEnd, fin, "")
	ReportMatchStep("halo_infinite", "  ", domain.MatchStepStatsWritten, "")       // match anonyme : écarté
	ReportMatchStepAt("halo_infinite", "m1", domain.MatchStepEnd, time.Time{}, "") // instant nul : écarté
	ReportMatchStep("halo_infinite", "m1", domain.MatchStepStatsWritten, "")

	got := DrainMatchSteps()
	if len(got) != 2 || got[0].Step != domain.MatchStepEnd || got[1].Step != domain.MatchStepStatsWritten {
		t.Fatalf("instants captés : %+v", got)
	}
	if got[0].At.Location() != time.UTC || !got[0].At.Equal(fin) {
		t.Fatalf("instant ramené en UTC attendu : %v", got[0].At)
	}
	if len(DrainMatchSteps()) != 0 {
		t.Fatal("le vidage doit vider le collecteur")
	}

	ReportMatchStep("halo_infinite", "m2", domain.MatchStepDetected, "")
	RequeueMatchSteps(got)
	apres := DrainMatchSteps()
	if len(apres) != 3 || apres[0].MatchID != "m1" || apres[2].MatchID != "m2" {
		t.Fatalf("remise en tête attendue : %+v", apres)
	}
}

func TestMatchSteps_BorneRespectee(t *testing.T) {
	EnableMatchSteps(true)
	t.Cleanup(func() { EnableMatchSteps(false) })
	for i := 0; i < MatchStepsBufferCap+3; i++ {
		ReportMatchStep("halo_infinite", "m", domain.MatchStepDetected, "")
	}
	if got := DrainMatchSteps(); len(got) != MatchStepsBufferCap {
		t.Fatalf("borne : %d instants, attendu %d", len(got), MatchStepsBufferCap)
	}
}
