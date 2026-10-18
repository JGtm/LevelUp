//go:build cgo

// Package ops — match_steps_cgo_test.go : les instants mesurés s'écrivent et se relisent en
// délai du débrief, du match le plus récent au plus ancien. Driver DuckDB requis.
package ops

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestMatchSteps_EcritureEtRapport(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	fin := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)
	ev := func(match string, step domain.MatchStep, apres time.Duration, detail string) domain.MatchStepEvent {
		return domain.MatchStepEvent{TitleSlug: "halo_infinite", MatchID: match, Step: step, At: fin.Add(apres), Detail: detail}
	}
	err := st.RecordMatchSteps(ctx, []domain.MatchStepEvent{
		ev("ancien", domain.MatchStepEnd, -time.Hour, ""),
		ev("ancien", domain.MatchStepStatsWritten, -time.Hour+time.Minute, ""),
		ev("recent", domain.MatchStepEnd, 0, ""),
		ev("recent", domain.MatchStepDerivationsWritten, 3*time.Minute, ""),
		ev("recent", domain.MatchStepCitationsWritten, 4*time.Minute, "x1"),
		// Un vieux match qui repasse (rattrapage) sans fin mesurée, APRÈS les autres : hors rapport.
		ev("historique", domain.MatchStepArtifactStored, time.Hour, "worker"),
		ev("historique", domain.MatchStepDerivationsWritten, time.Hour+time.Second, ""),
	})
	if err != nil {
		t.Fatalf("RecordMatchSteps: %v", err)
	}
	if err := st.RecordMatchSteps(ctx, nil); err != nil {
		t.Fatalf("lot vide : %v", err)
	}

	delais, err := st.MatchDelays(ctx, 10)
	if err != nil {
		t.Fatalf("MatchDelays: %v", err)
	}
	if len(delais) != 2 || delais[0].MatchID != "recent" || delais[1].MatchID != "ancien" {
		t.Fatalf("ordre des matchs : %+v (un match sans fin mesurée n'y figure pas)", delais)
	}
	if delais[0].TotalSeconds == nil || *delais[0].TotalSeconds != 240 || !*delais[0].ObjectiveMet {
		t.Fatalf("délai du match récent : %+v", delais[0])
	}
	if delais[1].TotalSeconds != nil {
		t.Fatalf("match ancien incomplet, aucun total attendu : %+v", delais[1])
	}

	limite, err := st.MatchDelays(ctx, 1)
	if err != nil || len(limite) != 1 || limite[0].MatchID != "recent" {
		t.Fatalf("limite : %+v, %v", limite, err)
	}
}
