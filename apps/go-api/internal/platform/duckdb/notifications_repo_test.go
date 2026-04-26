// Package duckdb_test — notifications_repo_test.go : tests intégration NotificationsRepo.
//
// Utilise un fichier DuckDB temporaire (pattern privacy_state_repo_test.go).
package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// newNotifTestDB crée stats.duckdb temporaire avec les 2 tables notifications.
func newNotifTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.duckdb")

	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("newNotifTestDB OpenReadWrite: %v", err)
	}
	defer rw.Close()

	ddl := `
		CREATE TABLE player_notifications (
			id            BIGINT PRIMARY KEY,
			category      VARCHAR NOT NULL,
			severity      VARCHAR NOT NULL DEFAULT 'info',
			title_key     VARCHAR NOT NULL,
			body_key      VARCHAR,
			params        VARCHAR,
			target_route  VARCHAR,
			target_search VARCHAR,
			actor_xuid    VARCHAR,
			actor_name    VARCHAR,
			source        VARCHAR NOT NULL,
			created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			read_at       TIMESTAMP
		);
		CREATE INDEX idx_pn_read_at      ON player_notifications(read_at);
		CREATE INDEX idx_pn_created_desc ON player_notifications(created_at DESC);
		CREATE INDEX idx_pn_category     ON player_notifications(category);
		CREATE TABLE notification_preferences (
			category   VARCHAR PRIMARY KEY,
			enabled    BOOLEAN NOT NULL DEFAULT TRUE,
			delivery   VARCHAR NOT NULL DEFAULT 'both',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	for _, stmt := range splitStmts(ddl) {
		if _, err := rw.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("newNotifTestDB DDL %q: %v", stmt, err)
		}
	}
	return dbPath
}

func splitStmts(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == ';' {
			if t := trimSpace(cur); t != "" {
				out = append(out, t)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if t := trimSpace(cur); t != "" {
		out = append(out, t)
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func openNotifPlayerDB(t *testing.T, dbPath string) *duckdb.PlayerDB {
	t.Helper()
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("openNotifPlayerDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &duckdb.PlayerDB{
		Player:   db,
		XUID:     "xuid-notif-test",
		Gamertag: "NotifTestPlayer",
	}
}

func newTestNotif(id int64, cat notifications.Category, src string) *notifications.Notification {
	return &notifications.Notification{
		ID:        id,
		Category:  cat,
		Severity:  notifications.SeverityInfo,
		TitleKey:  "notif." + string(cat) + ".title",
		Source:    src,
		CreatedAt: time.Now().UTC(),
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestNotificationsRepo_InsertList(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	if err := repo.Insert(ctx, newTestNotif(1, notifications.CategoryMatchSynced, "test")); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := repo.Insert(ctx, newTestNotif(2, notifications.CategoryMediaAdded, "test")); err != nil {
		t.Fatalf("Insert 2: %v", err)
	}

	res, err := repo.List(ctx, notifications.ListFilter{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(res.Items))
	}
}

func TestNotificationsRepo_FilterUnreadOnly(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		_ = repo.Insert(ctx, newTestNotif(i, notifications.CategoryMatchSynced, "t"))
	}

	if _, err := repo.MarkRead(ctx, []int64{1}); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	res, err := repo.List(ctx, notifications.ListFilter{UnreadOnly: true, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("expected 2 unread items, got %d", len(res.Items))
	}
}

func TestNotificationsRepo_FilterByCategory(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	_ = repo.Insert(ctx, newTestNotif(1, notifications.CategoryMatchSynced, "t"))
	_ = repo.Insert(ctx, newTestNotif(2, notifications.CategoryMediaAdded, "t"))
	_ = repo.Insert(ctx, newTestNotif(3, notifications.CategoryMediaAdded, "t"))

	res, err := repo.List(ctx, notifications.ListFilter{
		Category: notifications.CategoryMediaAdded,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("expected 2 media items, got %d", len(res.Items))
	}
}

func TestNotificationsRepo_UnreadCount(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	_ = repo.Insert(ctx, newTestNotif(1, notifications.CategoryMatchSynced, "t"))
	_ = repo.Insert(ctx, newTestNotif(2, notifications.CategoryMediaAdded, "t"))
	_ = repo.Insert(ctx, newTestNotif(3, notifications.CategoryMediaAdded, "t"))
	_, _ = repo.MarkRead(ctx, []int64{1})

	count, err := repo.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count.Count != 2 {
		t.Errorf("expected count=2, got %d", count.Count)
	}
	if count.ByCategory[string(notifications.CategoryMediaAdded)] != 2 {
		t.Errorf("expected 2 media unread, got %d", count.ByCategory[string(notifications.CategoryMediaAdded)])
	}
}

func TestNotificationsRepo_MarkUnread_NotFound(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	err := repo.MarkUnread(ctx, 999)
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNotificationsRepo_MarkAllRead(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	_ = repo.Insert(ctx, newTestNotif(1, notifications.CategoryMatchSynced, "t"))
	_ = repo.Insert(ctx, newTestNotif(2, notifications.CategoryMediaAdded, "t"))
	_ = repo.Insert(ctx, newTestNotif(3, notifications.CategoryMediaAdded, "t"))

	n, err := repo.MarkAllRead(ctx, notifications.CategoryMediaAdded)
	if err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 marked, got %d", n)
	}

	count, _ := repo.UnreadCount(ctx)
	if count.Count != 1 {
		t.Errorf("expected 1 remaining unread (match_synced), got %d", count.Count)
	}
}

func TestNotificationsRepo_Delete(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	_ = repo.Insert(ctx, newTestNotif(42, notifications.CategoryMatchSynced, "t"))
	if err := repo.Delete(ctx, 42); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	res, _ := repo.List(ctx, notifications.ListFilter{Limit: 50})
	if len(res.Items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(res.Items))
	}

	if err := repo.Delete(ctx, 999); err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown id, got %v", err)
	}
}

func TestNotificationsRepo_Preferences_DefaultEnabled(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	// Catégorie sans entrée explicite → considérée enabled (default-on)
	en, err := repo.IsCategoryEnabled(ctx, notifications.CategoryMatchSynced)
	if err != nil {
		t.Fatalf("IsCategoryEnabled: %v", err)
	}
	if !en {
		t.Error("expected default-enabled when no row")
	}
}

func TestNotificationsRepo_Preferences_RoundTrip(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	prefs := []notifications.Preference{
		{Category: notifications.CategoryMatchSynced, Enabled: false, Delivery: notifications.DeliveryOff},
		{Category: notifications.CategoryMediaAdded, Enabled: true, Delivery: notifications.DeliveryToast},
	}
	if err := repo.UpsertPreferences(ctx, prefs); err != nil {
		t.Fatalf("UpsertPreferences: %v", err)
	}

	got, err := repo.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 prefs, got %d", len(got))
	}

	// Verify Insert respecte la pref OFF
	err = repo.Insert(ctx, newTestNotif(1, notifications.CategoryMatchSynced, "t"))
	if err != notifications.ErrCategoryDisabled {
		t.Errorf("expected ErrCategoryDisabled when pref disabled, got %v", err)
	}
}

func TestNotificationsRepo_CursorPagination(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	// 5 items avec IDs croissants, but spaced created_at to force ORDER BY semantics
	base := time.Now().UTC()
	for i := int64(1); i <= 5; i++ {
		n := newTestNotif(i, notifications.CategoryMatchSynced, "t")
		n.CreatedAt = base.Add(time.Duration(i) * time.Second)
		if err := repo.Insert(ctx, n); err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
	}

	// Page 1 : 2 items les plus récents (id=5, 4)
	page1, err := repo.List(ctx, notifications.ListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1 expected 2, got %d", len(page1.Items))
	}
	if page1.NextCursor == nil {
		t.Fatal("expected next_cursor on page1")
	}
	if page1.Items[0].ID != 5 {
		t.Errorf("page1[0] expected id=5, got %d", page1.Items[0].ID)
	}

	// Page 2 : avant le cursor
	page2, err := repo.List(ctx, notifications.ListFilter{Limit: 2, BeforeID: *page1.NextCursor})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Errorf("page2 expected 2, got %d", len(page2.Items))
	}
	if page2.Items[0].ID >= *page1.NextCursor {
		t.Errorf("page2[0].id %d should be < cursor %d", page2.Items[0].ID, *page1.NextCursor)
	}
}

func TestNotificationsRepo_CapAndSweep(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	for i := int64(1); i <= 10; i++ {
		_ = repo.Insert(ctx, newTestNotif(i, notifications.CategoryMatchSynced, "t"))
	}

	if err := repo.CapAndSweep(ctx, 3); err != nil {
		t.Fatalf("CapAndSweep: %v", err)
	}

	res, _ := repo.List(ctx, notifications.ListFilter{Limit: 50})
	if len(res.Items) != 3 {
		t.Errorf("expected 3 items after cap=3, got %d", len(res.Items))
	}
}
