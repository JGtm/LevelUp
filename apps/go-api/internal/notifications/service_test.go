package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRepo est une impl en mémoire de Repository pour les tests du Service.
type fakeRepo struct {
	enabledByCat map[Category]bool
	inserted     []*Notification
	prefs        []Preference
	insertErr    error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		enabledByCat: map[Category]bool{},
	}
}

func (r *fakeRepo) Insert(_ context.Context, n *Notification) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	if enabled, ok := r.enabledByCat[n.Category]; ok && !enabled {
		return ErrCategoryDisabled
	}
	r.inserted = append(r.inserted, n)
	return nil
}

func (r *fakeRepo) List(_ context.Context, _ ListFilter) (ListResult, error) {
	return ListResult{}, nil
}
func (r *fakeRepo) UnreadCount(_ context.Context) (UnreadCount, error) {
	return UnreadCount{}, nil
}
func (r *fakeRepo) MarkRead(_ context.Context, ids []int64) (int, error) { return len(ids), nil }
func (r *fakeRepo) MarkUnread(_ context.Context, _ int64) error          { return nil }
func (r *fakeRepo) MarkAllRead(_ context.Context, _ Category) (int, error) {
	return 0, nil
}
func (r *fakeRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (r *fakeRepo) CapAndSweep(_ context.Context, _ int) error        { return nil }
func (r *fakeRepo) GetPreferences(_ context.Context) ([]Preference, error) {
	return r.prefs, nil
}
func (r *fakeRepo) UpsertPreferences(_ context.Context, p []Preference) error {
	r.prefs = p
	return nil
}
func (r *fakeRepo) IsCategoryEnabled(_ context.Context, c Category) (bool, error) {
	if e, ok := r.enabledByCat[c]; ok {
		return e, nil
	}
	return true, nil // default-on
}

func TestServiceEmit_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMatchSynced,
		TitleKey: "notif.match_synced.title",
		Source:   "sync_engine",
		Params:   map[string]any{"count": 3},
	})
	if err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 inserted, got %d", len(repo.inserted))
	}
	got := repo.inserted[0]
	if got.Severity != SeverityInfo {
		t.Errorf("expected default severity=info, got %q", got.Severity)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID from generator")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt populated")
	}
}

func TestServiceEmit_DropsWhenCategoryDisabled(t *testing.T) {
	repo := newFakeRepo()
	repo.enabledByCat[CategoryMediaAdded] = false
	svc := NewService(repo)

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMediaAdded,
		TitleKey: "notif.media_added.title",
		Source:   "media_handler",
	})
	if err != nil {
		t.Fatalf("expected silent drop, got error: %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Errorf("expected 0 inserted (category disabled), got %d", len(repo.inserted))
	}
}

func TestServiceEmit_TranslatesErrCategoryDisabledFromInsert(t *testing.T) {
	repo := newFakeRepo()
	repo.insertErr = ErrCategoryDisabled
	svc := NewService(repo)

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMatchSynced,
		TitleKey: "k",
		Source:   "s",
	})
	if err != nil {
		t.Errorf("expected nil (silent drop), got: %v", err)
	}
}

func TestServiceEmit_PropagatesGenericInsertError(t *testing.T) {
	repo := newFakeRepo()
	boom := errors.New("disk full")
	repo.insertErr = boom
	svc := NewService(repo)

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMatchSynced,
		TitleKey: "k",
		Source:   "s",
	})
	if err == nil || !errors.Is(err, boom) {
		t.Errorf("expected wrapping disk full error, got: %v", err)
	}
}

func TestServiceEmit_ValidationErrors(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	cases := []struct {
		name string
		in   EmitInput
	}{
		{"missing category", EmitInput{TitleKey: "k", Source: "s"}},
		{"missing title_key", EmitInput{Category: CategoryAppRelease, Source: "s"}},
		{"missing source", EmitInput{Category: CategoryAppRelease, TitleKey: "k"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Emit(context.Background(), tc.in); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestServiceEmit_RejectsOversizedParams(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	huge := make([]byte, 5*1024)
	for i := range huge {
		huge[i] = 'x'
	}
	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryAppRelease,
		TitleKey: "k",
		Source:   "s",
		Params:   map[string]any{"blob": string(huge)},
	})
	if err == nil {
		t.Fatal("expected error for oversized params, got nil")
	}
}

func TestServiceMarkRead_NoIDs_NoOp(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	res, err := svc.MarkRead(context.Background(), nil)
	if err != nil || res.Updated != 0 {
		t.Errorf("expected no-op, got updated=%d err=%v", res.Updated, err)
	}
}

func TestIDGenerator_Monotonic(t *testing.T) {
	g := NewIDGenerator()
	prev := g.Next()
	for i := 0; i < 1000; i++ {
		next := g.Next()
		if next <= prev {
			t.Fatalf("non-monotonic: prev=%d next=%d", prev, next)
		}
		prev = next
	}
}

func TestIDGenerator_FixedClock_RollsOverWithinMs(t *testing.T) {
	g := &IDGenerator{clock: func() int64 { return 1_700_000_000_000 }}
	first := g.Next()
	second := g.Next()
	if second-first != 1 {
		t.Errorf("expected seq increment by 1 in same ms; got delta=%d", second-first)
	}
}

func TestIDGenerator_NewMs_ResetsSeq(t *testing.T) {
	t1 := int64(1_700_000_000_000)
	t2 := t1 + 1
	calls := 0
	g := &IDGenerator{clock: func() int64 {
		calls++
		if calls == 1 {
			return t1
		}
		return t2
	}}
	first := g.Next()
	second := g.Next()
	expectedFirst := t1 << seqBits
	expectedSecond := t2 << seqBits
	if first != expectedFirst {
		t.Errorf("first: expected %d got %d", expectedFirst, first)
	}
	if second != expectedSecond {
		t.Errorf("second: expected %d got %d (seq should reset on new ms)", expectedSecond, second)
	}
	_ = time.Now() // ensure import used
}

func TestNoopEmitter(t *testing.T) {
	if err := (NoopEmitter{}).Emit(context.Background(), EmitInput{}); err != nil {
		t.Errorf("NoopEmitter should never error, got: %v", err)
	}
}
