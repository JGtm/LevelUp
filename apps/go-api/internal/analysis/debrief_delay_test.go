package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

var debriefFin = time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)

func etape(step domain.MatchStep, apres time.Duration, detail string) domain.MatchStepEvent {
	return domain.MatchStepEvent{TitleSlug: "halo_infinite", MatchID: "m1", Step: step, At: debriefFin.Add(apres), Detail: detail}
}

func etapeDuRapport(t *testing.T, d domain.MatchDelay, step domain.MatchStep) domain.MatchDelayStep {
	t.Helper()
	for _, s := range d.Steps {
		if s.Step == step {
			return s
		}
	}
	t.Fatalf("étape %s absente du rapport : %+v", step, d.Steps)
	return domain.MatchDelayStep{}
}

func TestDebriefDelay_MatchCompletOuvrier(t *testing.T) {
	d := DebriefDelay([]domain.MatchStepEvent{
		etape(domain.MatchStepEnd, 0, ""),
		etape(domain.MatchStepDetected, 40*time.Second, ""),
		etape(domain.MatchStepStatsWritten, 55*time.Second, ""),
		etape(domain.MatchStepKillSourceWritten, 90*time.Second, ""),
		etape(domain.MatchStepCitationsWritten, 100*time.Second, "x1"), // avant le film : ignorée
		etape(domain.MatchStepReplayEnqueued, 95*time.Second, ""),
		etape(domain.MatchStepReplayClaimed, 97*time.Second, ""),
		etape(domain.MatchStepArtifactStored, 160*time.Second, "worker"),
		etape(domain.MatchStepDerivationsWritten, 165*time.Second, ""),
		etape(domain.MatchStepCitationsWritten, 170*time.Second, "x1"),
		etape(domain.MatchStepCitationsWritten, 200*time.Second, "x2"),
	})
	if d.MatchID != "m1" || d.EndedAt != "2026-10-17T20:00:00Z" {
		t.Fatalf("identité ou fin : %+v", d)
	}
	if d.TotalSeconds == nil || *d.TotalSeconds != 200 {
		t.Fatalf("total = %v, attendu 200", d.TotalSeconds)
	}
	if d.ObjectiveMet == nil || !*d.ObjectiveMet {
		t.Fatalf("objectif tenu attendu : %v", d.ObjectiveMet)
	}
	if d.CitationsPlayers != 2 {
		t.Fatalf("joueurs cités après le film = %d, attendu 2", d.CitationsPlayers)
	}
	if c := etapeDuRapport(t, d, domain.MatchStepCitationsWritten); c.Detail != "x2" || *c.OffsetSeconds != 200 {
		t.Fatalf("citation retenue : %+v", c)
	}
	// L'ordre d'affichage est celui de domain.MatchSteps, pas celui d'arrivée.
	if d.Steps[0].Step != domain.MatchStepEnd || d.Steps[len(d.Steps)-1].Step != domain.MatchStepCitationsWritten {
		t.Fatalf("ordre des étapes : %+v", d.Steps)
	}
}

func TestDebriefDelay_PremiereOccurrenceRetenue(t *testing.T) {
	d := DebriefDelay([]domain.MatchStepEvent{
		etape(domain.MatchStepEnd, 0, ""),
		etape(domain.MatchStepStatsWritten, 90*time.Second, ""),
		etape(domain.MatchStepStatsWritten, 60*time.Second, ""),
	})
	if s := etapeDuRapport(t, d, domain.MatchStepStatsWritten); *s.OffsetSeconds != 60 {
		t.Fatalf("première occurrence attendue : %+v", s)
	}
	if d.TotalSeconds != nil || d.ObjectiveMet != nil {
		t.Fatalf("pas de total sans dérivations : %+v", d)
	}
}

func TestDebriefDelay_ObjectifManque(t *testing.T) {
	d := DebriefDelay([]domain.MatchStepEvent{
		etape(domain.MatchStepEnd, 0, ""),
		etape(domain.MatchStepReplayBuildStarted, 30*time.Second, ""),
		etape(domain.MatchStepDerivationsWritten, 6*time.Minute, ""),
		etape(domain.MatchStepCitationsWritten, 6*time.Minute+5*time.Second, "x1"),
	})
	if d.TotalSeconds == nil || *d.TotalSeconds != 365 || d.ObjectiveMet == nil || *d.ObjectiveMet {
		t.Fatalf("objectif manqué attendu : total=%v tenu=%v", d.TotalSeconds, d.ObjectiveMet)
	}
}

func TestDebriefDelay_CitationsSeulementAvantLeFilm(t *testing.T) {
	d := DebriefDelay([]domain.MatchStepEvent{
		etape(domain.MatchStepEnd, 0, ""),
		etape(domain.MatchStepCitationsWritten, 50*time.Second, "x1"),
		etape(domain.MatchStepDerivationsWritten, 3*time.Minute, ""),
	})
	if d.CitationsPlayers != 0 || d.TotalSeconds != nil {
		t.Fatalf("citations antérieures au film : rien de complet attendu, %+v", d)
	}
	if c := etapeDuRapport(t, d, domain.MatchStepCitationsWritten); *c.OffsetSeconds != 50 {
		t.Fatalf("la dernière citation connue reste affichée : %+v", c)
	}
}

func TestDebriefDelay_SansFinDeMatch(t *testing.T) {
	d := DebriefDelay([]domain.MatchStepEvent{
		etape(domain.MatchStepDerivationsWritten, time.Minute, ""),
		etape(domain.MatchStepCitationsWritten, 2*time.Minute, "x1"),
	})
	if d.EndedAt != "" || d.TotalSeconds != nil {
		t.Fatalf("sans fin de match, aucun délai : %+v", d)
	}
	if s := etapeDuRapport(t, d, domain.MatchStepDerivationsWritten); s.OffsetSeconds != nil {
		t.Fatalf("aucun écart sans référence : %+v", s)
	}
}

func TestDebriefDelay_Vide(t *testing.T) {
	d := DebriefDelay(nil)
	if d.Steps == nil || len(d.Steps) != 0 || d.MatchID != "" {
		t.Fatalf("rapport vide attendu : %+v", d)
	}
}
