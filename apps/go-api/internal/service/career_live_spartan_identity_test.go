// Package service — career_live_spartan_identity_test.go : tests pour
// upsertSpartanIdentity (Phase 2 PLAN_SPARTAN_IDENTITY_REFACTOR §11).
//
// Couvre le contrat de persistence customisation dans la table dédiée :
//   - spartanRepo nil → no-op (compat tests + rollback safe)
//   - xuid vide → no-op
//   - custom nil → status_only api_empty (préserve URLs précédentes)
//   - custom avec tous champs vides → status_only api_empty
//   - custom rempli → Upsert data complet avec status=ok
package service

import (
	"context"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// mockSpartanIdentityStore capture les appels Upsert + simule Load via une
// row stockée. Implémente SpartanIdentityStore.
type mockSpartanIdentityStore struct {
	mu     sync.Mutex
	calls  []spartanIdentityCall
	err    error
	stored *duckdb.SpartanIdentityRow
}

type spartanIdentityCall struct {
	xuid   string
	data   *duckdb.SpartanIdentityRow
	status duckdb.SpartanIdentityStatus
}

func (m *mockSpartanIdentityStore) Load(_ context.Context, _ string) (*duckdb.SpartanIdentityRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stored, nil
}

func (m *mockSpartanIdentityStore) Upsert(
	_ context.Context,
	xuid string,
	data *duckdb.SpartanIdentityRow,
	status duckdb.SpartanIdentityStatus,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, spartanIdentityCall{xuid: xuid, data: data, status: status})
	return m.err
}

func (m *mockSpartanIdentityStore) Calls() []spartanIdentityCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]spartanIdentityCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func TestUpsertSpartanIdentity_NoRepoNoOp(t *testing.T) {
	// spartanRepo non câblé → no-op silencieux (rollback safe).
	svc := NewCareerLiveService(nil, nil, nil, nil)
	// Pas de panic attendu, pas d'erreur retournée non plus (fire-and-forget).
	svc.upsertSpartanIdentity(context.Background(), "xuid123",
		&syncpkg.SpartanCustomizationData{SpartanID: "OKLM"})
}

func TestUpsertSpartanIdentity_EmptyXUIDNoOp(t *testing.T) {
	writer := &mockSpartanIdentityStore{}
	svc := NewCareerLiveService(nil, nil, nil, nil).WithSpartanIdentityRepo(writer)
	svc.upsertSpartanIdentity(context.Background(), "",
		&syncpkg.SpartanCustomizationData{SpartanID: "OKLM"})
	if got := writer.Calls(); len(got) != 0 {
		t.Errorf("attendu 0 appel sur xuid vide, got %d", len(got))
	}
}

func TestUpsertSpartanIdentity_CustomNilStatusOnly(t *testing.T) {
	writer := &mockSpartanIdentityStore{}
	svc := NewCareerLiveService(nil, nil, nil, nil).WithSpartanIdentityRepo(writer)
	svc.upsertSpartanIdentity(context.Background(), "xuid123", nil)

	calls := writer.Calls()
	if len(calls) != 1 {
		t.Fatalf("attendu 1 appel, got %d", len(calls))
	}
	c := calls[0]
	if c.xuid != "xuid123" {
		t.Errorf("xuid = %q, want xuid123", c.xuid)
	}
	if c.data != nil {
		t.Errorf("data = %+v, want nil (status-only)", c.data)
	}
	if c.status != duckdb.SpartanIdentityStatusAPIEmpty {
		t.Errorf("status = %q, want api_empty", c.status)
	}
}

func TestUpsertSpartanIdentity_CustomAllEmptyStatusOnly(t *testing.T) {
	writer := &mockSpartanIdentityStore{}
	svc := NewCareerLiveService(nil, nil, nil, nil).WithSpartanIdentityRepo(writer)
	// Tous les champs vides — sémantiquement équivalent à nil.
	svc.upsertSpartanIdentity(context.Background(), "xuid123",
		&syncpkg.SpartanCustomizationData{})

	calls := writer.Calls()
	if len(calls) != 1 {
		t.Fatalf("attendu 1 appel, got %d", len(calls))
	}
	c := calls[0]
	if c.data != nil {
		t.Errorf("data should be nil for all-empty custom, got %+v", c.data)
	}
	if c.status != duckdb.SpartanIdentityStatusAPIEmpty {
		t.Errorf("status = %q, want api_empty", c.status)
	}
}

func TestUpsertSpartanIdentity_CustomFullStatusOK(t *testing.T) {
	writer := &mockSpartanIdentityStore{}
	svc := NewCareerLiveService(nil, nil, nil, nil).WithSpartanIdentityRepo(writer)
	custom := &syncpkg.SpartanCustomizationData{
		SpartanID:        "OKLM",
		BannerImageURL:   "https://halo.api/banner.png",
		EmblemImageURL:   "https://halo.api/emblem.png",
		BackdropImageURL: "https://halo.api/backdrop.png",
	}
	svc.upsertSpartanIdentity(context.Background(), "xuid123", custom)

	calls := writer.Calls()
	if len(calls) != 1 {
		t.Fatalf("attendu 1 appel, got %d", len(calls))
	}
	c := calls[0]
	if c.status != duckdb.SpartanIdentityStatusOK {
		t.Errorf("status = %q, want ok", c.status)
	}
	if c.data == nil {
		t.Fatal("data is nil, want filled row")
	}
	if c.data.SpartanID != "OKLM" {
		t.Errorf("SpartanID = %q, want OKLM", c.data.SpartanID)
	}
	if c.data.BannerImageURL != "https://halo.api/banner.png" {
		t.Errorf("BannerImageURL = %q", c.data.BannerImageURL)
	}
	if c.data.EmblemImageURL != "https://halo.api/emblem.png" {
		t.Errorf("EmblemImageURL = %q", c.data.EmblemImageURL)
	}
	if c.data.BackdropImageURL != "https://halo.api/backdrop.png" {
		t.Errorf("BackdropImageURL = %q", c.data.BackdropImageURL)
	}
	if c.data.XUID != "xuid123" {
		t.Errorf("data.XUID = %q, want xuid123", c.data.XUID)
	}
}

func TestUpsertSpartanIdentity_CustomPartialStatusOK(t *testing.T) {
	// Au moins 1 champ non-vide → status=ok (le contrat IsEmpty est "tous
	// les champs URL/SpartanID sont vides"). On veut tracer même un
	// SpartanID seul, car le repo s'occupe d'écrire ce qu'il a.
	writer := &mockSpartanIdentityStore{}
	svc := NewCareerLiveService(nil, nil, nil, nil).WithSpartanIdentityRepo(writer)
	custom := &syncpkg.SpartanCustomizationData{SpartanID: "OKLM"}
	svc.upsertSpartanIdentity(context.Background(), "xuid123", custom)

	calls := writer.Calls()
	if len(calls) != 1 {
		t.Fatalf("attendu 1 appel, got %d", len(calls))
	}
	if calls[0].status != duckdb.SpartanIdentityStatusOK {
		t.Errorf("status = %q, want ok (partial counts as ok)", calls[0].status)
	}
	if calls[0].data == nil || calls[0].data.SpartanID != "OKLM" {
		t.Errorf("data = %+v, want SpartanID=OKLM", calls[0].data)
	}
}

// TestOverlayFromSpartanIdentity_NoRepoNoOp : si spartanRepo non câblé →
// identity inchangée (compat tests + rollback safe).
func TestOverlayFromSpartanIdentity_NoRepoNoOp(t *testing.T) {
	builder := &mockIdentityBuilder{}
	svc := NewCareerLiveService(nil, builder, nil, nil) // pas de spartanRepo
	id := &domain.HomeSpartanIdentityRow{RankNumber: 10}
	got := svc.overlayFromSpartanIdentityTable(context.Background(), "xuid123", id)
	if got != id {
		t.Errorf("identity should be unchanged (same pointer)")
	}
	if builder.receivedOverlay != nil {
		t.Errorf("builder.ApplySpartanIdentityOverlay was called, want skip")
	}
}

// TestOverlayFromSpartanIdentity_RowAbsent : row absente dans la nouvelle
// table → identity inchangée, pas d'overlay. Cas typique pendant la
// migration (avant que Phase 4 backfill ait peuplé la table).
func TestOverlayFromSpartanIdentity_RowAbsent(t *testing.T) {
	writer := &mockSpartanIdentityStore{stored: nil} // Load retourne nil
	builder := &mockIdentityBuilder{}
	svc := NewCareerLiveService(nil, builder, nil, nil).WithSpartanIdentityRepo(writer)
	id := &domain.HomeSpartanIdentityRow{RankNumber: 10}
	got := svc.overlayFromSpartanIdentityTable(context.Background(), "xuid123", id)
	if got == nil {
		t.Fatal("identity should not be nil")
	}
	if builder.receivedOverlay != nil {
		t.Errorf("builder.ApplySpartanIdentityOverlay was called, want skip when row absent")
	}
}

// TestOverlayFromSpartanIdentity_RowPresent : row présente → overlay appliqué.
// La nouvelle table est PRIORITAIRE sur les valeurs legacy.
func TestOverlayFromSpartanIdentity_RowPresent(t *testing.T) {
	writer := &mockSpartanIdentityStore{
		stored: &duckdb.SpartanIdentityRow{
			XUID:           "xuid123",
			SpartanID:      "NEW-TAG",
			BannerImageURL: "new-banner.png",
		},
	}
	builder := &mockIdentityBuilder{}
	svc := NewCareerLiveService(nil, builder, nil, nil).WithSpartanIdentityRepo(writer)
	oldTag := "OLD-TAG"
	id := &domain.HomeSpartanIdentityRow{
		RankNumber: 10,
		SpartanID:  &oldTag,
	}
	svc.overlayFromSpartanIdentityTable(context.Background(), "xuid123", id)
	if builder.receivedOverlay == nil {
		t.Fatal("builder.ApplySpartanIdentityOverlay was not called")
	}
	if builder.receivedOverlay.SpartanID != "NEW-TAG" {
		t.Errorf("receivedOverlay.SpartanID = %q, want NEW-TAG", builder.receivedOverlay.SpartanID)
	}
	// Le mock applique l'overlay → identity.SpartanID doit être maintenant NEW-TAG.
	if id.SpartanID == nil || *id.SpartanID != "NEW-TAG" {
		t.Errorf("identity.SpartanID = %v, want NEW-TAG (overlay should have replaced)", id.SpartanID)
	}
}

// TestOverlayFromSpartanIdentity_EmptyXUIDNoOp : xuid vide → no-op (sécurité).
func TestOverlayFromSpartanIdentity_EmptyXUIDNoOp(t *testing.T) {
	writer := &mockSpartanIdentityStore{
		stored: &duckdb.SpartanIdentityRow{SpartanID: "X"},
	}
	builder := &mockIdentityBuilder{}
	svc := NewCareerLiveService(nil, builder, nil, nil).WithSpartanIdentityRepo(writer)
	id := &domain.HomeSpartanIdentityRow{RankNumber: 10}
	svc.overlayFromSpartanIdentityTable(context.Background(), "", id)
	if builder.receivedOverlay != nil {
		t.Errorf("overlay should not be called for empty xuid")
	}
}
