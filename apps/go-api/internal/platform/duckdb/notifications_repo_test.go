// Package duckdb_test — notifications_repo_test.go : tests intégration NotificationsRepo.
//
// Utilise un fichier DuckDB temporaire (pattern privacy_state_repo_test.go).
package duckdb_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// newNotifTestDB crée shared_social.duckdb temporaire et applique la suite
// complète des migrations TargetSharedSocial (création + drop_idx_pn_xuid_unread).
// Le test reflète ainsi exactement le schéma de prod, ce qui garantit que tout
// changement de migration est attrapé sans dérive de DDL inline.
func newNotifTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")

	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("newNotifTestDB OpenReadWrite: %v", err)
	}
	defer rw.Close()

	if err := migration.RunForDB(rw.SQLDb(), migration.TargetSharedSocial); err != nil {
		t.Fatalf("newNotifTestDB RunForDB(TargetSharedSocial): %v", err)
	}
	return dbPath
}

// openNotifPlayerDB ouvre le shared_social.duckdb de test et construit un
// PlayerDB minimal où SharedSocial pointe sur cette DB. Le repo lit/écrit
// désormais sur SharedSocial et non plus Player, donc Player peut rester nil.
func openNotifPlayerDB(t *testing.T, dbPath string) *duckdb.PlayerDB {
	t.Helper()
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("openNotifPlayerDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &duckdb.PlayerDB{
		SharedSocial: db,
		XUID:         "xuid-notif-test",
		Gamertag:     "NotifTestPlayer",
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

// ─── player_records (table d'records pour personal_record) ───────────────

// TestPlayerRecords_UpsertAndLoad : valide qu'on peut écrire/lire un record via
// le shared_social.duckdb avec scoping xuid. Réutilise newNotifTestDB qui crée
// la table avec PK (xuid, metric).
func TestPlayerRecords_UpsertAndLoad(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	ctx := context.Background()

	// Insert via raw SQL pour bypasser le repo (focus sur le schéma shared_social)
	rwDB, err := duckdb.OpenReadWrite(pdb.SharedSocial.Path())
	if err != nil {
		t.Fatalf("rwDB: %v", err)
	}
	if _, err := rwDB.Exec(ctx, `
		INSERT INTO player_records (xuid, metric, value, achieved_match_id)
		VALUES (?, 'best_kda', 4.5, 'm1')`, pdb.XUID); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	rwDB.Close()

	// Read via SharedSocial pour vérifier round-trip + scoping xuid
	var v float64
	var matchID string
	err = pdb.SharedSocial.QueryRow(ctx,
		`SELECT value, achieved_match_id FROM player_records WHERE xuid = ? AND metric = 'best_kda'`,
		pdb.XUID,
	).Scan(&v, &matchID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if v != 4.5 {
		t.Errorf("expected 4.5, got %v", v)
	}
	if matchID != "m1" {
		t.Errorf("expected match m1, got %q", matchID)
	}

	// Sécurité : un autre xuid ne doit PAS voir ce record
	var otherV sql.NullFloat64
	_ = pdb.SharedSocial.QueryRow(ctx,
		`SELECT value FROM player_records WHERE xuid = 'other-xuid' AND metric = 'best_kda'`,
	).Scan(&otherV)
	if otherV.Valid {
		t.Errorf("xuid scoping leak: other-xuid sees value %v", otherV.Float64)
	}
}
