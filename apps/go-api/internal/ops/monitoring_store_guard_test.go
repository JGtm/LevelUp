// Package ops — monitoring_store_guard_test.go : garde-rail append-only.
//
// La base monitoring est append-only (ADR 0026) : le store n'a JAMAIS le droit
// d'émettre un UPDATE ou un DELETE sur les tables d'événements (detection_events,
// detection_status_events, cron_runs, data_health_runs). Toute mutation d'état
// passe par un nouvel INSERT + la vue _latest. Ce test scanne la source du store
// pour interdire la ré-introduction du pattern (sans justification datée).
package ops

import (
	"os"
	"regexp"
	"testing"
)

func TestMonitoringStore_NoUpdateOrDeleteOnAppendOnlyTables(t *testing.T) {
	src, err := os.ReadFile("monitoring_store.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)UPDATE\s+detection_events`),
		regexp.MustCompile(`(?i)UPDATE\s+detection_status_events`),
		regexp.MustCompile(`(?i)UPDATE\s+cron_runs`),
		regexp.MustCompile(`(?i)UPDATE\s+data_health_runs`),
		regexp.MustCompile(`(?i)DELETE\s+FROM\s+detection_events`),
		regexp.MustCompile(`(?i)DELETE\s+FROM\s+detection_status_events`),
		regexp.MustCompile(`(?i)DELETE\s+FROM\s+cron_runs`),
		regexp.MustCompile(`(?i)DELETE\s+FROM\s+data_health_runs`),
	}
	for _, re := range forbidden {
		if re.Match(src) {
			t.Errorf("monitoring_store.go viole l'append-only : pattern interdit %q trouvé", re.String())
		}
	}
}
