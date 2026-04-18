package sync

import "testing"

// ── NeedsLocalOnly ───────────────────────────────────────────────────────────

func TestNeedsLocalOnly_AllFalse(t *testing.T) {
	s := &SyncScope{}
	if s.NeedsLocalOnly() {
		t.Fatal("expected false with no local flags")
	}
}

func TestNeedsLocalOnly_WithSessions(t *testing.T) {
	s := &SyncScope{Sessions: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with Sessions")
	}
}

func TestNeedsLocalOnly_WithLUSR(t *testing.T) {
	s := &SyncScope{LUSR: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with LUSR")
	}
}

func TestNeedsLocalOnly_WithKillerVictim(t *testing.T) {
	s := &SyncScope{KillerVictim: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with KillerVictim")
	}
}

func TestNeedsLocalOnly_WithCitations(t *testing.T) {
	s := &SyncScope{Citations: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with Citations")
	}
}

func TestNeedsLocalOnly_WithSkillRank(t *testing.T) {
	s := &SyncScope{SkillRank: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with SkillRank")
	}
}

func TestNeedsLocalOnly_WithEndTime(t *testing.T) {
	s := &SyncScope{EndTime: true}
	if !s.NeedsLocalOnly() {
		t.Fatal("expected true with EndTime")
	}
}

func TestNeedsLocalOnly_APIOnly(t *testing.T) {
	s := &SyncScope{Medals: true, Events: true}
	if s.NeedsLocalOnly() {
		t.Fatal("expected false with API-only flags")
	}
}
