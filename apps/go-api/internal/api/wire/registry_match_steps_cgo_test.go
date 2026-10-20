//go:build cgo

package wire

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// TestBuildQueueReport_DelaiDuDebrief : un instant capté n'importe où dans le processus arrive
// dans le rapport de la file sans attendre la boucle de flush, et le collecteur s'arme et se
// désarme avec le store.
func TestBuildQueueReport_DelaiDuDebrief(t *testing.T) {
	ctx := context.Background()
	st, err := ops.NewMonitoringStore(ctx, filepath.Join(t.TempDir(), "monitoring.duckdb"))
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	t.Cleanup(func() { observability.EnableMatchSteps(false) })
	_ = observability.DrainMatchSteps()

	reg := (&ServiceRegistry{cfg: &config.AppConfig{RepoRoot: t.TempDir()}}).WithMonitoringStore(st)
	fin := time.Now().Add(-4 * time.Minute)
	observability.ReportMatchStepAt("halo_infinite", "m-debrief", domain.MatchStepEnd, fin, "")
	observability.ReportMatchStepAt("halo_infinite", "m-debrief", domain.MatchStepDerivationsWritten, fin.Add(2*time.Minute), "")
	observability.ReportMatchStepAt("halo_infinite", "m-debrief", domain.MatchStepCitationsWritten, fin.Add(3*time.Minute), "x1")

	resp, err := reg.BuildQueueReport(ctx, 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if len(resp.MatchDelays) != 1 || resp.MatchDelays[0].MatchID != "m-debrief" {
		t.Fatalf("délai du débrief absent du rapport : %+v", resp.MatchDelays)
	}
	if d := resp.MatchDelays[0]; d.TotalSeconds == nil || *d.TotalSeconds != 180 || !*d.ObjectiveMet {
		t.Fatalf("délai : %+v", d)
	}

	reg.closeMonitoringStore()
	observability.ReportMatchStep("halo_infinite", "m-apres", domain.MatchStepDetected, "")
	if got := observability.DrainMatchSteps(); len(got) != 0 {
		t.Fatalf("collecteur armé après la fermeture du store : %+v", got)
	}
}
