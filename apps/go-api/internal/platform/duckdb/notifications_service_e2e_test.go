// Package duckdb_test — notifications_service_e2e_test.go : tests E2E du
// notifications.Service avec un vrai NotificationsRepo DuckDB.
//
// Les tests dans internal/notifications/service_test.go utilisent un fakeRepo
// en mémoire qui ne reproduit pas les contraintes DuckDB (PK, indexes ART,
// NULL handling). Ici on couvre la frontière Service↔Repo end-to-end, en
// particulier les chemins qui passaient par l'ancien idx_pn_xuid_unread :
//   - Emit + Insert avec params/target JSON, actor, severity custom
//   - Emit avec préférence OFF → drop silencieux
//   - Service.MarkRead/MarkAllRead/Delete via le vrai repo
//   - UpdatePreferences + Emit qui respecte la pref fraîche

package duckdb_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// newServiceE2E construit un notifications.Service réel sur un repo DuckDB
// fraichement migré (incluant drop_idx_pn_xuid_unread).
func newServiceE2E(t *testing.T) (*notifications.Service, *duckdb.NotificationsRepo) {
	t.Helper()
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)
	return svc, repo
}

// ─── 1. Emit avec catégorie activée par défaut (pas de pref en base) ────────

func TestServiceE2E_Emit_DefaultEnabled_PersistsViaRepo(t *testing.T) {
	svc, repo := newServiceE2E(t)
	ctx := context.Background()

	err := svc.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryDataHealthWarning,
		Severity: notifications.SeverityWarn,
		TitleKey: "notif.data_health_warning.title",
		BodyKey:  "notif.data_health_warning.body",
		Params: map[string]any{
			"warnings_total": 3,
			"uuids_raw":      2,
			"hint":           "rerun cmd/repair_data_consistency",
		},
		TargetRoute: "/admin/data-health",
		Source:      "data_health_scheduler",
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	list, err := repo.List(ctx, notifications.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("attendu 1 item, obtenu %d", len(list.Items))
	}

	got := list.Items[0]
	if got.Category != notifications.CategoryDataHealthWarning {
		t.Errorf("category: %s", got.Category)
	}
	if got.Severity != notifications.SeverityWarn {
		t.Errorf("severity: %s", got.Severity)
	}
	if got.Source != "data_health_scheduler" {
		t.Errorf("source: %s", got.Source)
	}
	if got.TargetRoute != "/admin/data-health" {
		t.Errorf("target_route: %s", got.TargetRoute)
	}
	if got.ReadAt != nil {
		t.Error("notif nouvellement émise doit avoir read_at NULL")
	}

	// Params JSON doit être conservé tel quel.
	var decoded map[string]any
	if err := json.Unmarshal(got.Params, &decoded); err != nil {
		t.Fatalf("params not valid JSON: %v (%s)", err, string(got.Params))
	}
	// JSON numbers décodent en float64.
	if v, _ := decoded["warnings_total"].(float64); v != 3 {
		t.Errorf("params.warnings_total: attendu 3, obtenu %v", decoded["warnings_total"])
	}
	if v, _ := decoded["hint"].(string); v != "rerun cmd/repair_data_consistency" {
		t.Errorf("params.hint: %s", decoded["hint"])
	}
}

// ─── 2. Emit avec pref OFF (drop silencieux côté Service) ───────────────────

func TestServiceE2E_Emit_PrefOff_DropsSilently(t *testing.T) {
	svc, repo := newServiceE2E(t)
	ctx := context.Background()

	// Désactive data_health_warning avant tout Emit.
	if _, err := svc.UpdatePreferences(ctx, []notifications.Preference{{
		Category: notifications.CategoryDataHealthWarning,
		Enabled:  false,
		Delivery: notifications.DeliveryOff,
	}}); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	err := svc.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryDataHealthWarning,
		TitleKey: "notif.data_health_warning.title",
		Source:   "test",
	})
	if err != nil {
		t.Errorf("Emit avec pref OFF doit être silencieux, obtenu err=%v", err)
	}

	// Aucune ligne insérée.
	list, _ := repo.List(ctx, notifications.ListFilter{Limit: 10})
	if len(list.Items) != 0 {
		t.Errorf("attendu 0 notif, obtenu %d (drop silencieux pas respecté)", len(list.Items))
	}

	// Et l'erreur sentinelle ErrCategoryDisabled ne doit jamais remonter au caller.
	if errors.Is(err, notifications.ErrCategoryDisabled) {
		t.Error("ErrCategoryDisabled ne doit PAS être propagé au caller (drop silencieux côté Service)")
	}
}

// ─── 3. UpdatePreferences + Emit : la pref fraîche prend effet ───────────────

func TestServiceE2E_UpdatePref_ThenEmit_RespectsFreshPref(t *testing.T) {
	svc, repo := newServiceE2E(t)
	ctx := context.Background()

	// 1ère phase : pref ON par défaut → Emit insère.
	_ = svc.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		TitleKey: "notif.match_synced.title",
		Source:   "test",
	})

	// 2e phase : on désactive → Emit suivant doit dropper.
	if _, err := svc.UpdatePreferences(ctx, []notifications.Preference{{
		Category: notifications.CategoryMatchSynced,
		Enabled:  false,
		Delivery: notifications.DeliveryOff,
	}}); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	_ = svc.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		TitleKey: "notif.match_synced.title",
		Source:   "test",
	})

	// 3e phase : on réactive → Emit suivant doit insérer.
	if _, err := svc.UpdatePreferences(ctx, []notifications.Preference{{
		Category: notifications.CategoryMatchSynced,
		Enabled:  true,
		Delivery: notifications.DeliveryToast,
	}}); err != nil {
		t.Fatalf("UpdatePreferences re-enable: %v", err)
	}
	_ = svc.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		TitleKey: "notif.match_synced.title",
		Source:   "test",
	})

	list, _ := repo.List(ctx, notifications.ListFilter{Limit: 10})
	if len(list.Items) != 2 {
		t.Errorf("attendu 2 notifs (Emit phase 1 + 3), obtenu %d", len(list.Items))
	}
}

// ─── 4. Service.MarkRead/MarkAllRead/Delete via vrai repo ───────────────────

func TestServiceE2E_MarkRead_MarkAllRead_Delete_FullFlow(t *testing.T) {
	svc, _ := newServiceE2E(t)
	ctx := context.Background()

	// Émet 5 notifs data_health_warning via Service (donc avec ID auto-généré).
	for i := 0; i < 5; i++ {
		if err := svc.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryDataHealthWarning,
			Severity: notifications.SeverityWarn,
			TitleKey: "notif.data_health_warning.title",
			Source:   "data_health_scheduler",
		}); err != nil {
			t.Fatalf("Emit #%d: %v", i, err)
		}
	}

	list, err := svc.List(ctx, notifications.ListFilter{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 5 {
		t.Fatalf("attendu 5 notifs émises, obtenu %d", len(list.Items))
	}

	// MarkRead bulk (chemin UPDATE read_at IS NULL → l'ancien index ART).
	ids := make([]int64, 0, 5)
	for _, it := range list.Items {
		ids = append(ids, it.ID)
	}
	mr, err := svc.MarkRead(ctx, ids[:3])
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if mr.Updated != 3 {
		t.Errorf("MarkRead.Updated: %d", mr.Updated)
	}

	// MarkAllRead sur ce qui reste.
	mar, err := svc.MarkAllRead(ctx, notifications.CategoryDataHealthWarning)
	if err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if mar.Updated != 2 {
		t.Errorf("MarkAllRead.Updated: %d", mar.Updated)
	}

	count, _ := svc.UnreadCount(ctx)
	if count.Count != 0 {
		t.Errorf("UnreadCount après MarkAllRead: attendu 0, obtenu %d", count.Count)
	}

	// Delete une notif.
	if err := svc.Delete(ctx, ids[0]); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Delete ID inconnu → ErrNotFound propagé.
	err = svc.Delete(ctx, 99999999)
	if !errors.Is(err, notifications.ErrNotFound) {
		t.Errorf("Delete(unknown): attendu ErrNotFound, obtenu %v", err)
	}

	// La connexion reste valide après tout ce cycle.
	final, err := svc.List(ctx, notifications.ListFilter{Limit: 50})
	if err != nil {
		t.Fatalf("List final: %v", err)
	}
	if len(final.Items) != 4 {
		t.Errorf("attendu 4 notifs restantes, obtenu %d", len(final.Items))
	}
}

// ─── 5. Emit avec Actor + TargetSearch JSON ──────────────────────────────────

func TestServiceE2E_Emit_WithActorAndTargetSearch(t *testing.T) {
	svc, repo := newServiceE2E(t)
	ctx := context.Background()

	err := svc.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryMediaAdded,
		TitleKey:    "notif.media_added.title",
		Source:      "media_handler",
		TargetRoute: "/media",
		TargetSearch: map[string]any{
			"period": "30d",
			"map":    "Aquarius",
		},
		Actor: &notifications.Actor{
			XUID: "actor-xuid-001",
			Name: "FriendGT",
		},
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	list, _ := repo.List(ctx, notifications.ListFilter{Limit: 10})
	if len(list.Items) != 1 {
		t.Fatalf("attendu 1 item, obtenu %d", len(list.Items))
	}
	got := list.Items[0]
	if got.Actor == nil || got.Actor.XUID != "actor-xuid-001" || got.Actor.Name != "FriendGT" {
		t.Errorf("actor mal persisté: %+v", got.Actor)
	}
	if len(got.TargetSearch) == 0 {
		t.Error("target_search vide après round-trip")
	} else {
		var decoded map[string]any
		if err := json.Unmarshal(got.TargetSearch, &decoded); err != nil {
			t.Fatalf("target_search not JSON: %v", err)
		}
		if decoded["map"] != "Aquarius" {
			t.Errorf("target_search.map: %v", decoded["map"])
		}
	}
}

// ─── 6. EmitCoalesced (C5/DP5) : latest fusionne, history conserve les events ─

func TestNotificationsE2E_EmitCoalesced_MergesLatestKeepsHistory(t *testing.T) {
	dbPath := newNotifTestDB(t)
	pdb := openNotifPlayerDB(t, dbPath)
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)
	ctx := context.Background()

	mediaIn := func() notifications.EmitInput {
		return notifications.EmitInput{
			Category: notifications.CategoryMediaAdded,
			TitleKey: "notif.media_added.title",
			Source:   "media_handler",
			Actor:    &notifications.Actor{Name: "JGtm"},
			Params:   map[string]any{"actor_name": "JGtm", "count": 1},
		}
	}
	if err := svc.EmitCoalesced(ctx, mediaIn(), time.Hour); err != nil {
		t.Fatalf("EmitCoalesced 1: %v", err)
	}
	if err := svc.EmitCoalesced(ctx, mediaIn(), time.Hour); err != nil {
		t.Fatalf("EmitCoalesced 2: %v", err)
	}

	// player_notifications_latest : 1 ligne, non lue, count=2.
	list, err := repo.List(ctx, notifications.ListFilter{Category: notifications.CategoryMediaAdded, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("latest : attendu 1 ligne coalescée, obtenu %d", len(list.Items))
	}
	if list.Items[0].ReadAt != nil {
		t.Error("la notif coalescée doit rester non lue")
	}
	var p map[string]any
	if err := json.Unmarshal(list.Items[0].Params, &p); err != nil {
		t.Fatalf("params JSON: %v", err)
	}
	if v, _ := p["count"].(float64); v != 2 {
		t.Errorf("count attendu 2, obtenu %v", p["count"])
	}

	// player_notifications_history : 2 events (append-only).
	var histCount int
	row := pdb.SharedSocial.QueryRow(ctx,
		`SELECT COUNT(*) FROM player_notifications_history WHERE xuid = ? AND category = ?`,
		"xuid-notif-test", "media_added")
	if err := row.Scan(&histCount); err != nil {
		t.Fatalf("history count: %v", err)
	}
	if histCount != 2 {
		t.Errorf("history : attendu 2 events, obtenu %d", histCount)
	}
}
