package scheduler_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/cronstore"
	"levelup/go-api/internal/scheduler"
)

// L'avertissement d'une table au-delà du seuil nomme la compaction qui est attendue : celle du
// prochain démarrage quand l'instance compacte elle-même, celle de la nuit sinon.
func TestJaugePasses_MessageSelonQuiCompacte(t *testing.T) {
	for _, tc := range []struct {
		nom      string
		auDemarr bool
		attendu  string
		interdit string
	}{
		{"hote_de_nuit", false, "compaction de nuit attendue", "prochain démarrage"},
		{"au_demarrage", true, "compaction attendue au prochain démarrage", "de nuit"},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			repoRoot := t.TempDir()
			sharedPath := seedTitleSharedDB(t, repoRoot, "halo_infinite")
			seedBombStatsVersions(t, sharedPath)
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
			t.Cleanup(func() { slog.SetDefault(prev) })

			var lectures atomic.Int32
			sched := scheduler.NewDataHealthScheduler(repoRoot, cronstore.New(lecteurCompte(repoRoot, &lectures))).
				WithClock(func() time.Time { return horlogeJauge }).
				WithCompactionAtStartup(tc.auDemarr)
			if res := sched.RunOnce(context.Background()); res.PassesOverThreshold != 1 {
				t.Fatalf("au-dessus du seuil = %d, attendu 1", res.PassesOverThreshold)
			}
			if !strings.Contains(buf.String(), tc.attendu) {
				t.Errorf("journal sans %q :\n%s", tc.attendu, buf.String())
			}
			if strings.Contains(buf.String(), tc.interdit) {
				t.Errorf("journal avec %q :\n%s", tc.interdit, buf.String())
			}
		})
	}
}
