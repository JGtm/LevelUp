//go:build cgo

package ops

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func enqueueAt(t *testing.T, st *MonitoringStore, matchID string, priority int) (domain.BuildQueueJob, bool) {
	t.Helper()
	job, created, err := st.EnqueueBuildJob(context.Background(), EnqueueBuildJobRequest{
		JobType: string(domain.JobTypeReplayBuild), TitleSlug: "halo_infinite", MatchID: matchID,
		Priority: priority,
		Payload:  &domain.BuildQueuePayload{MatchID: matchID, ShortID: matchID[:8], TitleSlug: "halo_infinite"},
	})
	if err != nil {
		t.Fatalf("EnqueueBuildJob %s: %v", matchID, err)
	}
	return job, created
}

func claimMatch(t *testing.T, st *MonitoringStore) string {
	t.Helper()
	job, err := st.ClaimBuildJob(context.Background(), "ouvrier-1", "poste", "test/1")
	if err != nil || job == nil {
		t.Fatalf("ClaimBuildJob: job=%v err=%v", job, err)
	}
	return job.MatchID
}

// TestBuildQueue_PrioriteAvantAnciennete : le match inséré passe devant le récent, qui passe
// devant le rattrapage, quel que soit l'ordre de mise en file.
func TestBuildQueue_PrioriteAvantAnciennete(t *testing.T) {
	st := newTestStore(t)
	enqueueAt(t, st, "aaaaaaaa-rattrapage", domain.BuildPriorityRattrapage)
	enqueueAt(t, st, "bbbbbbbb-recent", domain.BuildPriorityRecent)
	enqueueAt(t, st, "cccccccc-insere", domain.BuildPriorityInsere)

	for _, attendu := range []string{"cccccccc-insere", "bbbbbbbb-recent", "aaaaaaaa-rattrapage"} {
		if got := claimMatch(t, st); got != attendu {
			t.Fatalf("pris %s, attendu %s", got, attendu)
		}
	}
}

// TestBuildQueue_RemiseEnFileReleveLaPriorite : un match d'abord enfilé par le rattrapage puis
// redemandé frais passe devant ; une demande plus basse ne l'abaisse pas ; un job en cours n'est
// pas retouché.
func TestBuildQueue_RemiseEnFileReleveLaPriorite(t *testing.T) {
	st := newTestStore(t)
	enqueueAt(t, st, "aaaaaaaa-recent-plus-ancien", domain.BuildPriorityRecent)
	enqueueAt(t, st, "dddddddd-redemande", domain.BuildPriorityRattrapage)

	job, created := enqueueAt(t, st, "dddddddd-redemande", domain.BuildPriorityInsere)
	if created || job.Priority != domain.BuildPriorityInsere {
		t.Fatalf("relèvement : créé=%v priorité=%d", created, job.Priority)
	}
	job, _ = enqueueAt(t, st, "dddddddd-redemande", domain.BuildPriorityRattrapage)
	if job.Priority != domain.BuildPriorityInsere {
		t.Fatalf("abaissement refusé attendu, priorité=%d", job.Priority)
	}
	if got := claimMatch(t, st); got != "dddddddd-redemande" {
		t.Fatalf("pris %s, attendu le match relevé", got)
	}

	// Le job pris est en cours : une nouvelle demande plus haute ne le touche pas.
	job, created = enqueueAt(t, st, "dddddddd-redemande", domain.BuildPriorityInsere+1)
	if created || job.Status != domain.JobStatusRunning || job.Priority != domain.BuildPriorityInsere {
		t.Fatalf("job en cours retouché : %+v créé=%v", job, created)
	}
}
