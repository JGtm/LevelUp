package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// TestRecordSnapshotCut_Metrics : recordSnapshotCut émet les bons compteurs/gauges
// expvar titrés selon l'issue (produit / no-op / échec). Titre unique par cas pour
// isoler les compteurs cumulatifs process-wide.
func TestRecordSnapshotCut_Metrics(t *testing.T) {
	// Produit : total + produced + gauges version/ready.
	tProd := "snaptest_prod"
	recordSnapshotCut(tProd, ops.SnapshotResult{Produced: true, Version: 7, ReadyMatchCount: 42}, nil, 5*time.Millisecond)
	if got := observability.LoadCounterT(tProd, "snapshot_cut_total"); got != 1 {
		t.Errorf("cut_total = %d, attendu 1", got)
	}
	if got := observability.LoadCounterT(tProd, "snapshot_cut_produced_total"); got != 1 {
		t.Errorf("cut_produced_total = %d, attendu 1", got)
	}
	if got := observability.LoadCounterT(tProd, "snapshot_version"); got != 7 {
		t.Errorf("snapshot_version (gauge) = %d, attendu 7", got)
	}
	if got := observability.LoadCounterT(tProd, "snapshot_ready_match_count"); got != 42 {
		t.Errorf("snapshot_ready_match_count (gauge) = %d, attendu 42", got)
	}

	// Échec (copy) : failures total + ventilation par raison.
	tFail := "snaptest_fail"
	recordSnapshotCut(tFail, ops.SnapshotResult{}, fmt.Errorf("%w: x", ops.ErrSnapshotCopy), time.Millisecond)
	if got := observability.LoadCounterT(tFail, "snapshot_cut_failures_total"); got != 1 {
		t.Errorf("cut_failures_total = %d, attendu 1", got)
	}
	if got := observability.LoadCounterT(tFail, "snapshot_cut_failures_total_"+snapCutFailCopy); got != 1 {
		t.Errorf("cut_failures_total_copy_failed = %d, attendu 1", got)
	}
	if got := observability.LoadCounterT(tFail, "snapshot_cut_produced_total"); got != 0 {
		t.Errorf("cut_produced_total sur échec = %d, attendu 0", got)
	}

	// No-op (unchanged) : noop total + ventilation.
	tNoop := "snaptest_noop"
	recordSnapshotCut(tNoop, ops.SnapshotResult{NoopReason: "unchanged"}, nil, time.Millisecond)
	if got := observability.LoadCounterT(tNoop, "snapshot_cut_noop_total"); got != 1 {
		t.Errorf("cut_noop_total = %d, attendu 1", got)
	}
	if got := observability.LoadCounterT(tNoop, "snapshot_cut_noop_total_unchanged"); got != 1 {
		t.Errorf("cut_noop_total_unchanged = %d, attendu 1", got)
	}
}

// TestClassifySnapshotCutErr : chaque sentinelle ops.ErrSnapshot* est réduite à la
// bonne raison de l'enum fermé ; une erreur inconnue tombe dans "other".
func TestClassifySnapshotCutErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"copy", fmt.Errorf("%w: COPY x", ops.ErrSnapshotCopy), snapCutFailCopy},
		{"manifest", fmt.Errorf("%w: flip", ops.ErrSnapshotManifest), snapCutFailManifest},
		{"read", fmt.Errorf("%w: gather", ops.ErrSnapshotRead), snapCutFailRead},
		{"other", errors.New("inconnu"), snapCutFailOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifySnapshotCutErr(c.err); got != c.want {
				t.Errorf("classifySnapshotCutErr(%v) = %q, attendu %q", c.err, got, c.want)
			}
		})
	}
}
