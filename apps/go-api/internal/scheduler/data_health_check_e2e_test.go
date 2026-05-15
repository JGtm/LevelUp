// Package scheduler_test — data_health_check_e2e_test.go : tests E2E du
// HealthScheduler avec un vrai NotificationsRepo DuckDB derrière.
//
// Couvre le call-site réel de la notif `data_health_warning` :
//   - HealthScheduler.RunOnce → audit shared_matches_v2 → Emit
//   - notifications.Service.Emit → IsCategoryEnabled → Insert
//   - NotificationsRepo (DuckDB shared_social) avec idx_pn_xuid_unread déjà droppé
//
// Reproduction du scénario log de prod 2026-05-14 : émission d'une notif
// data_health_warning, suivie d'opérations utilisateur (List, MarkAllRead)
// pour valider que la connexion shared_social reste saine.

package scheduler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// healthE2ESetup prépare l'arborescence repoRoot/data/titles/halo_infinite/
// avec shared_matches_v2.duckdb (TargetShared) + shared_social.duckdb
// (TargetSharedSocial). Retourne un (repoRoot, sharedSocialPath).
func healthE2ESetup(t *testing.T) (repoRoot, sharedSocialPath string) {
	t.Helper()
	repoRoot = t.TempDir()
	titleDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite")
	warehouseDir := filepath.Join(titleDir, "warehouse")
	if err := os.MkdirAll(warehouseDir, 0o755); err != nil {
		t.Fatalf("mkdir warehouse: %v", err)
	}

	sharedPath := filepath.Join(warehouseDir, "shared_matches_v2.duckdb")
	shared, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared_matches_v2: %v", err)
	}
	if err := migration.RunForDB(shared.SQLDb(), migration.TargetShared); err != nil {
		shared.Close()
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared.Close()

	sharedSocialPath = filepath.Join(titleDir, "shared_social.duckdb")
	social, err := duckdb.OpenReadWrite(sharedSocialPath)
	if err != nil {
		t.Fatalf("open shared_social: %v", err)
	}
	if err := migration.RunForDB(social.SQLDb(), migration.TargetSharedSocial); err != nil {
		social.Close()
		t.Fatalf("RunForDB(SharedSocial): %v", err)
	}
	social.Close()

	return repoRoot, sharedSocialPath
}

// seedRawUUIDMapName injecte une anomalie UUID brut dans match_registry.
// Force HealthScheduler.runCycle à détecter au moins un warning et donc
// à déclencher emitWarningNotification.
func seedRawUUIDMapName(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared rw: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(context.Background(), `
		INSERT INTO match_registry (match_id, map_name, pair_name)
		VALUES ('match-uuid-anomaly', '12345678-1234-1234-1234-1234567890ab', 'normal_pair')
	`)
	if err != nil {
		t.Fatalf("seed raw UUID map_name: %v", err)
	}
}

// buildNotifService construit un notifications.Service réel adossé à un
// NotificationsRepo qui écrit dans le shared_social.duckdb fourni.
func buildNotifService(t *testing.T, sharedSocialPath, xuid string) (*notifications.Service, *duckdb.PlayerDB) {
	t.Helper()
	social, err := duckdb.OpenReadWrite(sharedSocialPath)
	if err != nil {
		t.Fatalf("open shared_social: %v", err)
	}
	t.Cleanup(func() { social.Close() })

	pdb := &duckdb.PlayerDB{
		SharedSocial: social,
		XUID:         xuid,
		Gamertag:     "HealthE2EPlayer",
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)
	return svc, pdb
}

// ─── 1. RunOnce sans anomalie → pas d'émission ───────────────────────────────

func TestHealthScheduler_E2E_NoWarnings_DoesNotEmit(t *testing.T) {
	repoRoot, sharedSocialPath := healthE2ESetup(t)
	svc, pdb := buildNotifService(t, sharedSocialPath, "xuid-no-warn")

	sched := scheduler.NewDataHealthScheduler(repoRoot, svc)
	res := sched.RunOnce(context.Background())

	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.WarningsTotal != 0 {
		t.Errorf("WarningsTotal: attendu 0 sur DB vierge, obtenu %d", res.WarningsTotal)
	}

	// Aucune notif émise (warnings_total == 0).
	repo := duckdb.NewNotificationsRepo(pdb)
	list, err := repo.List(context.Background(), notifications.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("attendu 0 notif, obtenu %d", len(list.Items))
	}
}

// ─── 2. Cycle complet : anomalie → notif émise → DB saine après MarkAllRead ──

func TestHealthScheduler_E2E_WithAnomaly_EmitsAndStaysHealthy(t *testing.T) {
	repoRoot, sharedSocialPath := healthE2ESetup(t)
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite",
		"warehouse", "shared_matches_v2.duckdb")
	seedRawUUIDMapName(t, sharedPath)

	svc, pdb := buildNotifService(t, sharedSocialPath, "xuid-anomaly")
	sched := scheduler.NewDataHealthScheduler(repoRoot, svc)

	// Cycle 1 : doit détecter l'anomalie et émettre.
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.UUIDsRawCount < 1 {
		t.Errorf("UUIDsRawCount: attendu >= 1, obtenu %d", res.UUIDsRawCount)
	}
	if res.WarningsTotal < 1 {
		t.Errorf("WarningsTotal: attendu >= 1, obtenu %d", res.WarningsTotal)
	}

	repo := duckdb.NewNotificationsRepo(pdb)
	list, err := repo.List(context.Background(),
		notifications.ListFilter{Category: notifications.CategoryDataHealthWarning, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("attendu 1 notif data_health_warning, obtenu %d", len(list.Items))
	}
	if list.Items[0].Severity != notifications.SeverityWarn {
		t.Errorf("severity attendu warn, obtenu %s", list.Items[0].Severity)
	}
	if list.Items[0].TitleKey != "notif.data_health_warning.title" {
		t.Errorf("title_key inattendu: %s", list.Items[0].TitleKey)
	}

	// Cycle 2 : l'anomalie est toujours là → re-émission. Pas de corruption.
	res2 := sched.RunOnce(context.Background())
	if res2.WarningsTotal < 1 {
		t.Errorf("cycle 2 WarningsTotal: attendu >= 1, obtenu %d", res2.WarningsTotal)
	}

	list2, err := repo.List(context.Background(),
		notifications.ListFilter{Category: notifications.CategoryDataHealthWarning, Limit: 10})
	if err != nil {
		t.Fatalf("List cycle 2: %v (DB invalidée par le 2e cycle ?)", err)
	}
	if len(list2.Items) != 2 {
		t.Errorf("attendu 2 notifs après 2 cycles, obtenu %d", len(list2.Items))
	}

	// Interaction UI typique : MarkAllRead — c'est la séquence exacte qui
	// déclenchait le bug en prod. Doit rester saine post-fix.
	n, err := svc.MarkAllRead(context.Background(), notifications.CategoryDataHealthWarning)
	if err != nil {
		t.Fatalf("MarkAllRead: %v (régression du bug ART/NULL ?)", err)
	}
	if n.Updated != 2 {
		t.Errorf("MarkAllRead.Updated: attendu 2, obtenu %d", n.Updated)
	}

	// 3e cycle après MarkAllRead — la DB doit toujours répondre.
	res3 := sched.RunOnce(context.Background())
	if res3 == nil {
		t.Fatal("RunOnce cycle 3 a retourné nil")
	}

	count, err := repo.UnreadCount(context.Background())
	if err != nil {
		t.Fatalf("UnreadCount: %v (régression ?)", err)
	}
	// Les 2 premières notifs sont lues, la 3e est non-lue.
	if count.Count != 1 {
		t.Errorf("UnreadCount: attendu 1 (notif cycle 3 non-lue), obtenu %d", count.Count)
	}
}

// ─── 3. Emitter nil : le scheduler doit fonctionner en mode dégradé ─────────

func TestHealthScheduler_E2E_NilEmitter_NoCrash(t *testing.T) {
	repoRoot, _ := healthE2ESetup(t)
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite",
		"warehouse", "shared_matches_v2.duckdb")
	seedRawUUIDMapName(t, sharedPath)

	// notif = nil — le scheduler doit calculer + logger sans émettre.
	sched := scheduler.NewDataHealthScheduler(repoRoot, nil)
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil avec emitter nil")
	}
	if res.UUIDsRawCount < 1 {
		t.Errorf("anomalie devrait être détectée même sans emitter, obtenu %d", res.UUIDsRawCount)
	}
}

// ─── 4. Catégorie désactivée : Emit doit dropper silencieusement ────────────

func TestHealthScheduler_E2E_CategoryDisabled_DropsSilently(t *testing.T) {
	repoRoot, sharedSocialPath := healthE2ESetup(t)
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite",
		"warehouse", "shared_matches_v2.duckdb")
	seedRawUUIDMapName(t, sharedPath)

	svc, pdb := buildNotifService(t, sharedSocialPath, "xuid-disabled")

	// Désactiver data_health_warning pour ce xuid.
	repo := duckdb.NewNotificationsRepo(pdb)
	if err := repo.UpsertPreferences(context.Background(), []notifications.Preference{{
		Category: notifications.CategoryDataHealthWarning,
		Enabled:  false,
		Delivery: notifications.DeliveryOff,
	}}); err != nil {
		t.Fatalf("UpsertPreferences: %v", err)
	}

	sched := scheduler.NewDataHealthScheduler(repoRoot, svc)
	res := sched.RunOnce(context.Background())
	if res.WarningsTotal < 1 {
		t.Errorf("WarningsTotal: attendu >= 1, obtenu %d", res.WarningsTotal)
	}

	// Mais aucune notif insérée (Service.Emit drop silencieux).
	list, err := repo.List(context.Background(), notifications.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("attendu 0 notif (pref OFF), obtenu %d", len(list.Items))
	}
}
