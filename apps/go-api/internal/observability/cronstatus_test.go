// Package observability — cronstatus_test.go : registre central des crons (A6).
package observability

import (
	"errors"
	"testing"
	"time"
)

func TestReportCronRun_RegistryLifecycle(t *testing.T) {
	ResetCronStatus()
	t.Cleanup(ResetCronStatus)
	start := time.Now()

	// Succès initial.
	ReportCronRun("test_cron", start, nil, 120)
	snap := CronStatusSnapshot()
	if len(snap) != 1 {
		t.Fatalf("attendu 1 record, got %d", len(snap))
	}
	rec := snap[0]
	if rec.Runs != 1 || rec.ConsecutiveFailures != 0 || rec.LastSuccessAt.IsZero() || rec.LastError != "" {
		t.Fatalf("après succès : %+v", rec)
	}

	// Trois échecs consécutifs.
	for i := 0; i < 3; i++ {
		ReportCronRun("test_cron", start.Add(time.Duration(i+1)*time.Minute), errors.New("boom"), 50)
	}
	rec = CronStatusSnapshot()[0]
	if rec.ConsecutiveFailures != 3 || rec.LastError != "boom" || rec.Runs != 4 {
		t.Fatalf("après 3 échecs : %+v", rec)
	}
	if !rec.LastSuccessAt.Equal(start.Truncate(0)) && rec.LastSuccessAt.IsZero() {
		t.Fatalf("last_success perdu : %+v", rec)
	}

	// Un succès remet les échecs consécutifs à zéro.
	ReportCronRun("test_cron", start.Add(10*time.Minute), nil, 80)
	rec = CronStatusSnapshot()[0]
	if rec.ConsecutiveFailures != 0 || rec.LastError != "" {
		t.Fatalf("après récupération : %+v", rec)
	}
}

func TestReportCronRun_SinkRelay(t *testing.T) {
	ResetCronStatus()
	t.Cleanup(ResetCronStatus)

	type call struct {
		name string
		ok   bool
		err  string
	}
	var calls []call
	SetCronRunSink(func(name string, _ time.Time, ok bool, errStr string, _ int64) {
		calls = append(calls, call{name, ok, errStr})
	})

	ReportCronRun("relayed", time.Now(), nil, 1)
	ReportCronRun("relayed", time.Now(), errors.New("ka"), 2)
	if len(calls) != 2 || !calls[0].ok || calls[1].ok || calls[1].err != "ka" {
		t.Fatalf("sink calls = %+v", calls)
	}
}

func TestHeartbeat_ReadBack(t *testing.T) {
	Heartbeat("test_feature")
	if unix := HeartbeatUnix("test_feature"); unix <= 0 {
		t.Fatalf("heartbeat non posé : %d", unix)
	}
	if unix := HeartbeatUnix("test_feature_never"); unix != 0 {
		t.Fatalf("feature jamais vue doit être 0 : %d", unix)
	}
}
