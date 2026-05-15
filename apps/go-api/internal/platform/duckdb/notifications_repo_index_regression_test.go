// Package duckdb_test — notifications_repo_index_regression_test.go
//
// Tests de non-régression pour le bug DuckDB ART/NULL sur idx_pn_xuid_unread.
//
// Historique (2026-05-14) : la base shared_social s'invalidait en cours
// d'exécution avec « Failed to delete all rows from index. Only deleted 0
// out of 1 rows. » dès qu'un UPDATE ou DELETE touchait une notif avec
// read_at = NULL. L'index ART secondaire (xuid, read_at) ne parvenait pas
// à retirer ses entrées NULL, ce qui marquait la connexion fatale et
// faisait casser TOUT le repo (List, UnreadCount…) jusqu'au restart.
//
// Le fix supprime l'index via la migration `drop_idx_pn_xuid_unread`.
// Les tests ci-dessous couvrent les chemins UPDATE/DELETE sur read_at NULL
// — précisément la classe d'opérations qui déclenchait le bug.

package duckdb_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

// dataHealthNotif construit une notif data_health_warning identique à celle
// émise par scheduler.HealthScheduler.emitWarningNotification (cf. log
// 2026-05-14 17:47:09 où le bug s'est manifesté).
func dataHealthNotif(id int64) *notifications.Notification {
	return &notifications.Notification{
		ID:          id,
		Category:    notifications.CategoryDataHealthWarning,
		Severity:    notifications.SeverityWarn,
		TitleKey:    "notif.data_health_warning.title",
		BodyKey:     "notif.data_health_warning.body",
		Source:      "data_health_scheduler",
		TargetRoute: "/admin/data-health",
	}
}

func mustInsert(t *testing.T, repo *duckdb.NotificationsRepo, id int64, cat notifications.Category) {
	t.Helper()
	n := newTestNotif(id, cat, "regression")
	if err := repo.Insert(context.Background(), n); err != nil {
		t.Fatalf("Insert id=%d: %v", id, err)
	}
}

// ─── 1. MarkRead bulk sur lignes read_at=NULL ──────────────────────────────

// TestRegression_MarkRead_OnUnreadRows_NoIndexCorruption garantit qu'on peut
// faire un MarkRead bulk (UPDATE read_at = now() WHERE xuid = ? AND
// read_at IS NULL AND id IN (...)) sur 100 lignes non-lues sans corrompre
// l'index ni invalider la connexion.
func TestRegression_MarkRead_OnUnreadRows_NoIndexCorruption(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	const N = 100
	ids := make([]int64, 0, N)
	for i := int64(1); i <= N; i++ {
		mustInsert(t, repo, i, notifications.CategoryDataHealthWarning)
		ids = append(ids, i)
	}

	n, err := repo.MarkRead(ctx, ids)
	if err != nil {
		t.Fatalf("MarkRead(N=%d): %v", N, err)
	}
	if n != N {
		t.Errorf("MarkRead: attendu %d lignes marquées, obtenu %d", N, n)
	}

	// La connexion doit rester valide : List doit fonctionner immédiatement.
	res, err := repo.List(ctx, notifications.ListFilter{Limit: N + 10})
	if err != nil {
		t.Fatalf("List après MarkRead: %v (signe d'une connexion invalidée)", err)
	}
	if len(res.Items) != N {
		t.Errorf("List: attendu %d items, obtenu %d", N, len(res.Items))
	}

	count, err := repo.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count.Count != 0 {
		t.Errorf("UnreadCount: attendu 0 (tout marqué lu), obtenu %d", count.Count)
	}
}

// ─── 2. MarkAllRead sur lignes read_at=NULL ────────────────────────────────

// TestRegression_MarkAllRead_OnDataHealthWarning reproduit le scénario exact
// du log de prod (2026-05-14) : 1 notif data_health_warning non-lue, puis
// MarkAllRead (clic UI "marquer tout comme lu" côté front).
func TestRegression_MarkAllRead_OnDataHealthWarning(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	if err := repo.Insert(ctx, dataHealthNotif(7285886276755456)); err != nil {
		t.Fatalf("Insert data_health_warning: %v", err)
	}

	n, err := repo.MarkAllRead(ctx, notifications.CategoryDataHealthWarning)
	if err != nil {
		t.Fatalf("MarkAllRead(data_health_warning): %v "+
			"(régression : c'est exactement le scénario qui invalidait la DB)", err)
	}
	if n != 1 {
		t.Errorf("MarkAllRead: attendu 1 ligne, obtenu %d", n)
	}

	// La connexion doit rester valide après le bulk UPDATE qui touchait
	// l'ancien index ART (xuid, read_at NULL).
	count, err := repo.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("UnreadCount après MarkAllRead: %v (connexion invalidée ?)", err)
	}
	if count.Count != 0 {
		t.Errorf("UnreadCount: attendu 0, obtenu %d", count.Count)
	}
}

// TestRegression_MarkAllRead_NoCategory_BulkUnread fait un MarkAllRead sans
// filtre catégorie sur 50 lignes non-lues (mélange de catégories) — c'est
// le chemin le plus large d'UPDATE bulk sur read_at IS NULL.
func TestRegression_MarkAllRead_NoCategory_BulkUnread(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	categories := []notifications.Category{
		notifications.CategoryDataHealthWarning,
		notifications.CategoryMatchSynced,
		notifications.CategoryMediaAdded,
		notifications.CategorySyncError,
	}
	for i := int64(1); i <= 50; i++ {
		mustInsert(t, repo, i, categories[i%int64(len(categories))])
	}

	n, err := repo.MarkAllRead(ctx, "")
	if err != nil {
		t.Fatalf("MarkAllRead(empty category): %v", err)
	}
	if n != 50 {
		t.Errorf("MarkAllRead: attendu 50, obtenu %d", n)
	}

	count, _ := repo.UnreadCount(ctx)
	if count.Count != 0 {
		t.Errorf("UnreadCount: attendu 0, obtenu %d", count.Count)
	}
}

// ─── 3. Delete sur ligne read_at=NULL ──────────────────────────────────────

// TestRegression_Delete_OnUnreadRow vérifie qu'un DELETE d'une notif
// non-lue (read_at NULL) ne déclenche pas le bug ART et conserve la
// connexion valide pour les opérations suivantes.
func TestRegression_Delete_OnUnreadRow(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	mustInsert(t, repo, 1, notifications.CategoryDataHealthWarning)
	mustInsert(t, repo, 2, notifications.CategoryMatchSynced)

	if err := repo.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete id=1 (read_at NULL): %v", err)
	}

	// La connexion doit rester valide.
	res, err := repo.List(ctx, notifications.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List après Delete: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != 2 {
		t.Errorf("List: attendu [id=2], obtenu %+v", res.Items)
	}
}

// ─── 4. CapAndSweep sur lignes read_at=NULL ────────────────────────────────

// TestRegression_CapAndSweep_OnUnreadRows : CapAndSweep supprime en bulk
// les notifs les plus anciennes (DELETE FROM ... WHERE id NOT IN (top N)).
// Toutes les notifs ont read_at NULL — exactement le chemin qui faisait
// crasher l'ART (xuid, read_at).
func TestRegression_CapAndSweep_OnUnreadRows(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	const total = 20
	for i := int64(1); i <= total; i++ {
		mustInsert(t, repo, i, notifications.CategoryDataHealthWarning)
	}

	// Note : CapAndSweep est best-effort (n'expose pas l'erreur), donc on
	// vérifie l'effet via le compte final.
	if err := repo.CapAndSweep(ctx, 5); err != nil {
		t.Fatalf("CapAndSweep: %v", err)
	}

	res, err := repo.List(ctx, notifications.ListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("List après CapAndSweep: %v "+
			"(régression : DELETE bulk sur read_at NULL doit rester valide)", err)
	}
	if len(res.Items) != 5 {
		t.Errorf("CapAndSweep(5): attendu 5 items restants, obtenu %d", len(res.Items))
	}
}

// ─── 5. MarkUnread (UPDATE read_at NON-NULL → NULL) ────────────────────────

// TestRegression_MarkUnread_AfterMarkRead vérifie le chemin inverse :
// UPDATE read_at = NULL (réinsertion d'une entrée NULL dans l'ART). Ce
// chemin est aussi sensible aux index sur colonnes nullables.
func TestRegression_MarkUnread_AfterMarkRead(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	mustInsert(t, repo, 42, notifications.CategoryDataHealthWarning)

	if _, err := repo.MarkRead(ctx, []int64{42}); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := repo.MarkUnread(ctx, 42); err != nil {
		t.Fatalf("MarkUnread (read_at → NULL): %v", err)
	}

	count, err := repo.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count.Count != 1 {
		t.Errorf("UnreadCount après MarkUnread: attendu 1, obtenu %d", count.Count)
	}
}

// ─── 6. Concurrence INSERT + UPDATE + DELETE ───────────────────────────────

// TestRegression_ConcurrentInsertMarkDelete vérifie que des opérations
// d'écriture concurrentes (qui partagent la même *sql.DB via le cache
// process-level) ne corrompent pas l'index. Plusieurs goroutines Insert,
// MarkRead, Delete en parallèle sur des plages d'IDs disjointes.
func TestRegression_ConcurrentInsertMarkDelete(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	const workers = 8
	const perWorker = 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*3)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			base := int64(workerID*1000 + 1)

			// Insert N notifs non-lues.
			ids := make([]int64, 0, perWorker)
			for i := int64(0); i < perWorker; i++ {
				id := base + i
				if err := repo.Insert(ctx, newTestNotif(id, notifications.CategoryDataHealthWarning, "concur")); err != nil {
					errs <- fmt.Errorf("worker %d Insert(%d): %w", workerID, id, err)
					return
				}
				ids = append(ids, id)
			}

			// MarkRead 1ère moitié (UPDATE read_at = now() sur ART NULL).
			if _, err := repo.MarkRead(ctx, ids[:perWorker/2]); err != nil {
				errs <- fmt.Errorf("worker %d MarkRead: %w", workerID, err)
				return
			}

			// Delete les notifs marquées lues (DELETE ligne avec read_at non-NULL).
			for _, id := range ids[:perWorker/2] {
				if err := repo.Delete(ctx, id); err != nil {
					errs <- fmt.Errorf("worker %d Delete(%d): %w", workerID, id, err)
					return
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrence: %v", err)
	}

	// Vérification finale : on attend (workers * perWorker/2) notifs restantes,
	// toutes non-lues, et la connexion doit être valide.
	expected := workers * (perWorker - perWorker/2)
	count, err := repo.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("UnreadCount final: %v", err)
	}
	if count.Count != expected {
		t.Errorf("UnreadCount final: attendu %d, obtenu %d", expected, count.Count)
	}
}

// ─── 7. Pipeline complet HealthScheduler ───────────────────────────────────

// TestRegression_DataHealthWarning_FullCycle : cycle complet équivalent au
// scénario de prod (Insert via emitWarningNotification + interaction UI).
//   - Émission d'une notif data_health_warning (3 cycles consécutifs)
//   - Lecture de la liste (List, UnreadCount, IsCategoryEnabled)
//   - Clic "marquer comme lu" sur l'une (MarkRead)
//   - Clic "marquer toutes lues" (MarkAllRead)
//   - Suppression (Delete)
//   - Audit final
//
// Toutes les opérations doivent réussir sans invalider la DB.
func TestRegression_DataHealthWarning_FullCycle(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	ctx := context.Background()

	// Cycle 1 : 3 émissions data_health_warning successives (le scheduler
	// ré-émet à chaque cycle 24h si warnings_total > 0).
	for i := int64(1); i <= 3; i++ {
		if err := repo.Insert(ctx, dataHealthNotif(i)); err != nil {
			t.Fatalf("Insert cycle %d: %v", i, err)
		}
	}

	// Vérification côté UI : List et UnreadCount.
	res, err := repo.List(ctx, notifications.ListFilter{
		Category: notifications.CategoryDataHealthWarning,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("List(data_health_warning): %v", err)
	}
	if len(res.Items) != 3 {
		t.Errorf("List: attendu 3 items, obtenu %d", len(res.Items))
	}

	count, _ := repo.UnreadCount(ctx)
	if count.ByCategory[string(notifications.CategoryDataHealthWarning)] != 3 {
		t.Errorf("UnreadCount.ByCategory: attendu 3, obtenu %d",
			count.ByCategory[string(notifications.CategoryDataHealthWarning)])
	}

	// Clic UI 1 : marquer la notif id=1 comme lue.
	if _, err := repo.MarkRead(ctx, []int64{1}); err != nil {
		t.Fatalf("MarkRead(1): %v", err)
	}

	// Clic UI 2 : "marquer toutes lues" filtré data_health_warning.
	n, err := repo.MarkAllRead(ctx, notifications.CategoryDataHealthWarning)
	if err != nil {
		t.Fatalf("MarkAllRead(data_health_warning): %v", err)
	}
	if n != 2 { // 2 et 3 restantes non-lues
		t.Errorf("MarkAllRead: attendu 2 marquées, obtenu %d", n)
	}

	// Clic UI 3 : suppression d'une notif lue.
	if err := repo.Delete(ctx, 2); err != nil {
		t.Fatalf("Delete(2): %v", err)
	}

	// Audit final : connexion saine, état cohérent.
	res, err = repo.List(ctx, notifications.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List final: %v (connexion invalidée ?)", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("List final: attendu 2 items (id=1,3), obtenu %d", len(res.Items))
	}
	count, _ = repo.UnreadCount(ctx)
	if count.Count != 0 {
		t.Errorf("UnreadCount final: attendu 0, obtenu %d", count.Count)
	}
}
