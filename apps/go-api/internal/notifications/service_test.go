package notifications

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// fakeRepo est une impl en mémoire de Repository pour les tests du Service.
type fakeRepo struct {
	enabledByCat map[Category]bool
	inserted     []*Notification
	prefs        []Preference
	insertErr    error

	// Capture des derniers appels pour assertions
	lastListLimit int
	lastUnreadID  int64
	lastAllCat    Category
	lastDeleteID  int64
	unread        UnreadCount
	sweepCalls    int
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

func (r *fakeRepo) List(_ context.Context, f ListFilter) (ListResult, error) {
	r.lastListLimit = f.Limit
	// Vue "latest par ID" (append-only) filtrée par catégorie, plus récent d'abord.
	latest := map[int64]*Notification{}
	for _, n := range r.inserted {
		if f.Category != "" && n.Category != f.Category {
			continue
		}
		latest[n.ID] = n // le dernier inséré pour cet ID gagne
	}
	items := make([]Notification, 0, len(latest))
	for _, n := range latest {
		items = append(items, *n)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return ListResult{Items: items}, nil
}
func (r *fakeRepo) UnreadCount(_ context.Context) (UnreadCount, error) {
	return r.unread, nil
}
func (r *fakeRepo) MarkRead(_ context.Context, ids []int64) (int, error) { return len(ids), nil }
func (r *fakeRepo) MarkUnread(_ context.Context, id int64) error {
	r.lastUnreadID = id
	return nil
}
func (r *fakeRepo) MarkAllRead(_ context.Context, c Category) (int, error) {
	r.lastAllCat = c
	return 0, nil
}
func (r *fakeRepo) Delete(_ context.Context, id int64) error {
	r.lastDeleteID = id
	return nil
}
func (r *fakeRepo) CapAndSweep(_ context.Context, _ int) error { return nil }
func (r *fakeRepo) SweepStaleInfoRead(_ context.Context, _ time.Time) error {
	r.sweepCalls++
	return nil
}
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

// ─── Tests des méthodes Service via fakeRepo ─────────────────────────────

func TestServiceList_DefaultsLimit(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	// Limit <=0 doit être normalisée vers DefaultListLimit
	_, err := svc.List(context.Background(), ListFilter{Limit: 0})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if repo.lastListLimit != DefaultListLimit {
		t.Errorf("expected limit=%d, got %d", DefaultListLimit, repo.lastListLimit)
	}
}

func TestServiceList_LimitTooLarge_Capped(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	_, _ = svc.List(context.Background(), ListFilter{Limit: 999})
	if repo.lastListLimit != DefaultListLimit {
		t.Errorf("expected cap to default %d, got %d", DefaultListLimit, repo.lastListLimit)
	}
}

func TestServiceList_RespectsValidLimit(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	_, _ = svc.List(context.Background(), ListFilter{Limit: 25})
	if repo.lastListLimit != 25 {
		t.Errorf("expected limit=25, got %d", repo.lastListLimit)
	}
}

func TestServiceUnreadCount(t *testing.T) {
	repo := newFakeRepo()
	repo.unread = UnreadCount{Count: 7, ByCategory: map[string]int{"match_synced": 7}}
	svc := NewService(repo)
	got, err := svc.UnreadCount(context.Background())
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if got.Count != 7 {
		t.Errorf("expected count=7, got %d", got.Count)
	}
}

func TestServiceMarkUnread_Delegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if err := svc.MarkUnread(context.Background(), 42); err != nil {
		t.Fatalf("MarkUnread: %v", err)
	}
	if repo.lastUnreadID != 42 {
		t.Errorf("expected MarkUnread(42), got %d", repo.lastUnreadID)
	}
}

func TestServiceMarkAllRead_PassesCategory(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	res, err := svc.MarkAllRead(context.Background(), CategoryMatchSynced)
	if err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if repo.lastAllCat != CategoryMatchSynced {
		t.Errorf("expected category=match_synced, got %q", repo.lastAllCat)
	}
	_ = res
}

func TestServiceDelete_Delegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if err := svc.Delete(context.Background(), 99); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.lastDeleteID != 99 {
		t.Errorf("expected Delete(99), got %d", repo.lastDeleteID)
	}
}

func TestServiceUpdatePreferences_RoundTrip(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	prefs := []Preference{{Category: CategoryMatchSynced, Enabled: false, Delivery: DeliveryOff}}
	got, err := svc.UpdatePreferences(context.Background(), prefs)
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	if len(got) != 1 || got[0].Enabled {
		t.Errorf("expected disabled pref, got %+v", got)
	}
}

// ─────────── WithWriterAcquirer (commit 4 db-concurrency) ───────────

// busyWriterAcquirer renvoie un acquéreur qui simule un lease saturé : il tient
// déjà le mutex sur le path donné, donc toute tentative supplémentaire timeout.
// Retourne aussi une fonction cleanup à appeler en defer dans le test.
func busyWriterAcquirer(t *testing.T) (func() (*dblease.LeasedWriter, error), func()) {
	t.Helper()
	path := "test://" + t.Name() + "/" + time.Now().Format("150405.000000000")

	// On tient le writer pendant toute la durée du test pour saturer le lease.
	heldWriter, err := dblease.AcquireWriter(nil, path, dblease.KindSharedSocial, time.Second)
	if err != nil {
		t.Fatalf("setup busyWriterAcquirer: %v", err)
	}
	cleanup := func() { heldWriter.Release() }

	acquirer := func() (*dblease.LeasedWriter, error) {
		// Timeout court pour ne pas faire traîner les tests.
		return dblease.AcquireWriter(nil, path, dblease.KindSharedSocial, 30*time.Millisecond)
	}
	return acquirer, cleanup
}

func TestService_Emit_LeaseBusy_BestEffort(t *testing.T) {
	acquirer, cleanup := busyWriterAcquirer(t)
	defer cleanup()

	repo := newFakeRepo()
	repo.enabledByCat[CategoryMatchSynced] = true
	svc := NewService(repo, WithWriterAcquirer(acquirer))

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMatchSynced,
		TitleKey: "k",
		BodyKey:  "b",
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("Emit should be best-effort on lease busy, got err: %v", err)
	}
	// Aucune insertion ne doit avoir eu lieu (le lease n'a pas pu être acquis).
	if len(repo.inserted) != 0 {
		t.Errorf("Emit should not insert when lease busy, got %d insertions", len(repo.inserted))
	}
}

func TestService_MarkRead_LeaseBusy_PropagatesErrDBLocked(t *testing.T) {
	acquirer, cleanup := busyWriterAcquirer(t)
	defer cleanup()

	repo := newFakeRepo()
	svc := NewService(repo, WithWriterAcquirer(acquirer))

	_, err := svc.MarkRead(context.Background(), []int64{1, 2, 3})
	if err == nil {
		t.Fatal("MarkRead should propagate ErrDBLocked when lease busy")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
}

func TestService_Delete_LeaseBusy_PropagatesErrDBLocked(t *testing.T) {
	acquirer, cleanup := busyWriterAcquirer(t)
	defer cleanup()

	repo := newFakeRepo()
	svc := NewService(repo, WithWriterAcquirer(acquirer))

	err := svc.Delete(context.Background(), 42)
	if err == nil {
		t.Fatal("Delete should propagate ErrDBLocked when lease busy")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
}

func TestService_UpdatePreferences_LeaseBusy_PropagatesErrDBLocked(t *testing.T) {
	acquirer, cleanup := busyWriterAcquirer(t)
	defer cleanup()

	repo := newFakeRepo()
	svc := NewService(repo, WithWriterAcquirer(acquirer))

	_, err := svc.UpdatePreferences(context.Background(),
		[]Preference{{Category: CategoryMatchSynced, Enabled: true, Delivery: DeliveryToast}})
	if err == nil {
		t.Fatal("UpdatePreferences should propagate ErrDBLocked when lease busy")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
}

// TestService_Emit_NoAcquirer_BehavesLikeBefore vérifie qu'un service sans
// WriterAcquirer (cas legacy / tests existants) garde son comportement
// strictement identique. Garde-fou de non-régression.
func TestService_Emit_NoAcquirer_BehavesLikeBefore(t *testing.T) {
	repo := newFakeRepo()
	repo.enabledByCat[CategoryMatchSynced] = true
	svc := NewService(repo) // pas d'option

	err := svc.Emit(context.Background(), EmitInput{
		Category: CategoryMatchSynced,
		TitleKey: "k",
		BodyKey:  "b",
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Errorf("expected 1 insertion, got %d", len(repo.inserted))
	}
}

// ─── C4 : EmitCoalesced ──────────────────────────────────────────────────────

func mediaInput(actor string) EmitInput {
	return EmitInput{
		Category: CategoryMediaAdded,
		TitleKey: "notif.media_added.title",
		Source:   "media_handler",
		Actor:    &Actor{Name: actor},
		Params:   map[string]any{"actor_name": actor, "count": 1},
	}
}

// latestFor retourne la vue latest d'une catégorie (via le Service.List).
func latestFor(t *testing.T, svc *Service, cat Category) []Notification {
	t.Helper()
	lr, err := svc.List(context.Background(), ListFilter{Category: cat, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return lr.Items
}

func TestEmitCoalesced_SameActor_MergesCountAndID(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	if err := svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour); err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	if err := svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour); err != nil {
		t.Fatalf("emit 2: %v", err)
	}
	items := latestFor(t, svc, CategoryMediaAdded)
	if len(items) != 1 {
		t.Fatalf("attendu 1 notif coalescée, got %d", len(items))
	}
	if got := coalescedCountOf(&items[0]); got != 2 {
		t.Errorf("count sommé attendu 2, got %d", got)
	}
}

func TestEmitCoalesced_DifferentActors_TwoNotifs(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	_ = svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour)
	_ = svc.EmitCoalesced(ctx, mediaInput("Madina"), time.Hour)
	if items := latestFor(t, svc, CategoryMediaAdded); len(items) != 2 {
		t.Errorf("acteurs différents → 2 notifs, got %d", len(items))
	}
}

func TestEmitCoalesced_OutsideWindow_TwoNotifs(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	_ = svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour)
	// Vieillir la candidate au-delà de la fenêtre.
	repo.inserted[0].CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	_ = svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour)
	if items := latestFor(t, svc, CategoryMediaAdded); len(items) != 2 {
		t.Errorf("candidate hors fenêtre → 2 notifs, got %d", len(items))
	}
}

func TestEmitCoalesced_ReadCandidate_NeverResurrected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	_ = svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour)
	// Marquer la candidate comme lue.
	now := time.Now().UTC()
	repo.inserted[0].ReadAt = &now
	_ = svc.EmitCoalesced(ctx, mediaInput("JGtm"), time.Hour)
	if items := latestFor(t, svc, CategoryMediaAdded); len(items) != 2 {
		t.Errorf("candidate lue → nouvelle notif (jamais ressuscitée), got %d", len(items))
	}
}

// C7 : sync_error (sans acteur) coalesce sur la catégorie seule, count incrémenté.
func TestEmitCoalesced_SyncError_NoActor_CategoryOnly(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	syncErr := func(msg string) EmitInput {
		return EmitInput{
			Category: CategorySyncError,
			TitleKey: "notif.sync_error.title",
			Source:   "sync_handler",
			Params:   map[string]any{"message": msg, "job_id": "j1"},
		}
	}
	_ = svc.EmitCoalesced(ctx, syncErr("boom 1"), 6*time.Hour)
	_ = svc.EmitCoalesced(ctx, syncErr("boom 2"), 6*time.Hour)
	_ = svc.EmitCoalesced(ctx, syncErr("boom 3"), 6*time.Hour)
	items := latestFor(t, svc, CategorySyncError)
	if len(items) != 1 {
		t.Fatalf("3 échecs → 1 notif coalescée, got %d", len(items))
	}
	if got := coalescedCountOf(&items[0]); got != 3 {
		t.Errorf("count attendu 3, got %d", got)
	}
	if msg, _ := jsonStringField(items[0].Params, "message"); msg != "boom 3" {
		t.Errorf("dernier message attendu 'boom 3', got %q", msg)
	}
}
