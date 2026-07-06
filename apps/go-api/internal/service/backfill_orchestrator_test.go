package service

import (
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	go_sync "levelup/go-api/internal/sync"
)

// warnUnimplemented a été extrait du handler backfill vers l'orchestrateur (K1f).
// Ces tests (ex-handlers/backfill_test.go) valident qu'il liste les types non
// implémentés (medals/skill/aliases) et PAS ceux désormais implémentés (events).

// TestWarnUnimplemented_EventsNotListed (Phase 2 PLAN_HIGHLIGHT_EVENTS_BACKFILL) :
// scope.Events=true ne doit PLUS produire une warning listant 'events'.
func TestWarnUnimplemented_EventsNotListed(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "player-test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Events: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if got == nil {
		t.Fatalf("Get(%s) returned nil", job.JobID)
	}
	for _, w := range got.Warnings {
		if strings.Contains(w, "events") {
			t.Errorf("warning ne doit PAS lister 'events' (Phase 2 livrée) : %q", w)
		}
	}
}

// TestWarnUnimplemented_StillListsOthers : les types non implémentés (medals, skill)
// doivent rester listés.
func TestWarnUnimplemented_StillListsOthers(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "player-test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Medals: true, Skill: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if got == nil {
		t.Fatalf("Get(%s) returned nil", job.JobID)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("attendu une warning pour medals+skill non-implémentés")
	}
	hasMedals := false
	hasSkill := false
	for _, w := range got.Warnings {
		if strings.Contains(w, "medals") {
			hasMedals = true
		}
		if strings.Contains(w, "skill") {
			hasSkill = true
		}
	}
	if !hasMedals {
		t.Error("warning devrait mentionner medals")
	}
	if !hasSkill {
		t.Error("warning devrait mentionner skill")
	}
}

func TestWarnUnimplemented_NoTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{}} // no flags

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d", len(got.Warnings))
	}
}

func TestWarnUnimplemented_WithTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Medals: true, Events: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(got.Warnings))
	}
}
