package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

const syncJobsInternalCause = "duckdb: SELECT * FROM secret_table failed"

// L'échec d'une sync (job et notification) se résume à un code et un message fixes.
func TestSyncFailure_CodeAndFixedMessage(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantCode  string
		retryable bool
	}{
		{"contention de bail", fmt.Errorf("lease: %w (%s)", dblease.ErrDBLocked, syncJobsInternalCause), "db_busy", true},
		{"échec réel", errors.New(syncJobsInternalCause), "sync_failed", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := syncFailure(tc.err)
			if got.Code != tc.wantCode || got.Retryable != tc.retryable {
				t.Errorf("syncFailure = %+v, attendu code %q retryable %v", got, tc.wantCode, tc.retryable)
			}
			if got.Message != domain.ErrorMessageForCode(tc.wantCode) {
				t.Errorf("message = %q, attendu le message fixe du code", got.Message)
			}
			if strings.Contains(got.Message, "duckdb") || strings.Contains(got.Message, "SELECT") {
				t.Errorf("le message laisse fuir la cause : %q", got.Message)
			}
		})
	}
}

// match_synced n'est émise que si des matchs ont été insérés ; sync_error est coalescée sur la catégorie
// et ne porte que le code et le job, jamais la cause.
func TestSyncJobs_Notifications(t *testing.T) {
	f := newSyncJobsFixture(t)
	em := &fakeEmitter{}
	f.svc.WithNotificationsEmitterFactory(func(context.Context, string) (notifications.Emitter, error) { return em, nil })

	f.svc.emitMatchSynced(context.Background(), "jgtm", 0)
	if len(em.emittedInputs()) != 0 {
		t.Error("aucun match inséré : aucune notification match_synced")
	}
	f.svc.emitMatchSynced(context.Background(), "jgtm", 4)
	got := em.emittedInputs()
	if len(got) != 1 || got[0].Category != notifications.CategoryMatchSynced || got[0].Params[notifParamCount] != 4 {
		t.Errorf("match_synced émise : %+v", got)
	}

	f.svc.emitSyncError(context.Background(), "jgtm", "job-1", syncFailure(errors.New(syncJobsInternalCause)))
	co := em.coalescedInputs()
	if len(co) != 1 || co[0].Category != notifications.CategorySyncError ||
		co[0].Params[notifParamCode] != "sync_failed" || co[0].Params["job_id"] != "job-1" {
		t.Errorf("sync_error émise : %+v", co)
	}
}

// Sans fabrique ou avec une fabrique en échec, l'émission est sans effet et sans panique.
func TestSyncJobs_NotificationsBestEffort(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.svc.emitMatchSynced(context.Background(), "jgtm", 1)
	f.svc.emitSyncError(context.Background(), "jgtm", "job-1", syncFailure(errors.New("x")))

	f.svc.WithNotificationsEmitterFactory(func(context.Context, string) (notifications.Emitter, error) {
		return nil, errors.New("fabrique en échec")
	})
	f.svc.emitMatchSynced(context.Background(), "jgtm", 1)
	f.svc.emitSyncError(context.Background(), "jgtm", "job-1", syncFailure(errors.New("x")))
}
