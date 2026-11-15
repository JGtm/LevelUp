package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Rien de lancé, rien en vol : ni job, ni « en vol », ni délai.
func TestPlayerSyncStatus_Repos(t *testing.T) {
	f := newSyncJobsFixture(t)
	st, err := f.svc.PlayerSyncStatus(context.Background(), "jgtm")
	if err != nil {
		t.Fatal(err)
	}
	if st.PlayerSlug != "jgtm" || st.Syncing || st.Job != nil || st.CooldownRetryAfterSec != 0 {
		t.Errorf("état de repos attendu, obtenu %+v", st)
	}
}

// Un job actif du joueur est rendu, et le joueur est « en vol » ; le job d'un AUTRE joueur ne l'est pas.
func TestPlayerSyncStatus_JobActifDuJoueurSeulement(t *testing.T) {
	f := newSyncJobsFixture(t)
	other := f.jobs.Create(domain.JobTypeInitialSync, "h5only")
	mine := f.jobs.Create(domain.JobTypeInitialSync, "jgtm")
	f.jobs.SetStatus(mine.JobID, domain.JobStatusRunning, nil)

	st, err := f.svc.PlayerSyncStatus(context.Background(), "jgtm")
	if err != nil {
		t.Fatal(err)
	}
	if !st.Syncing || st.Job == nil || st.Job.JobID != mine.JobID || st.Job.PlayerSlug != "jgtm" {
		t.Fatalf("job actif de jgtm attendu, obtenu %+v", st)
	}
	if st.Job.JobID == other.JobID {
		t.Fatal("le job d'un autre joueur ne doit jamais être servi")
	}
}

// Une synchronisation d'une autre source (porte de déduplication) met le joueur « en vol » sans job.
func TestPlayerSyncStatus_EnVolParUneAutreSource(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.gate.refuseAll = true
	st, err := f.svc.PlayerSyncStatus(context.Background(), "jgtm")
	if err != nil {
		t.Fatal(err)
	}
	if !st.Syncing || st.Job != nil {
		t.Errorf("en vol sans job attendu, obtenu %+v", st)
	}
}

// Un job échoué reste lisible (avec sa cause) quand plus rien ne tourne.
func TestPlayerSyncStatus_DernierJobEchoue(t *testing.T) {
	f := newSyncJobsFixture(t)
	job := f.jobs.Create(domain.JobTypeInitialSync, "jgtm")
	f.jobs.Update(job.JobID, func(j *domain.AsyncJobStatus) {
		j.Status = domain.JobStatusFailed
		j.Error = domain.NewJobError("sync_halo_api_error", true)
	})
	st, err := f.svc.PlayerSyncStatus(context.Background(), "jgtm")
	if err != nil {
		t.Fatal(err)
	}
	if st.Syncing || st.Job == nil || st.Job.Status != domain.JobStatusFailed || st.Job.Error == nil || st.Job.Error.Code != "sync_halo_api_error" {
		t.Errorf("job échoué attendu, obtenu %+v", st)
	}
}

// Après un sync manuel, le délai restant est servi (sans être consommé par la lecture) ; un autre joueur n'en a pas.
func TestPlayerSyncStatus_DelaiRestantDuSyncManuel(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.svc.WithManualSyncCooldown(time.Hour)
	if _, err := f.svc.StartDeltaSync(context.Background(), sessionCaller(), "jgtm"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // deux lectures : la lecture ne consomme rien
		st, err := f.svc.PlayerSyncStatus(context.Background(), "jgtm")
		if err != nil {
			t.Fatal(err)
		}
		if st.CooldownRetryAfterSec <= 0 || st.CooldownRetryAfterSec > 3601 {
			t.Fatalf("délai attendu dans ]0, 3601] s (arrondi au plafond), obtenu %d", st.CooldownRetryAfterSec)
		}
	}
	other, err := f.svc.PlayerSyncStatus(context.Background(), "h5only")
	if err != nil {
		t.Fatal(err)
	}
	if other.CooldownRetryAfterSec != 0 {
		t.Errorf("l'autre joueur n'a aucun délai, obtenu %d", other.CooldownRetryAfterSec)
	}
}

func TestPlayerSyncStatus_JoueurInconnu(t *testing.T) {
	f := newSyncJobsFixture(t)
	_, err := f.svc.PlayerSyncStatus(context.Background(), "inconnu")
	requireSyncJobError(t, err, port.SyncErrPlayerNotFound)
}
