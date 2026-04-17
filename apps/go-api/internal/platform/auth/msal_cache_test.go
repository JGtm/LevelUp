// Package auth — msal_cache_test.go : tests pour InMemoryCacheAccessor.
package auth_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"

	auth "levelup/go-api/internal/platform/auth"
)

// ─────────────────────────────────────────────────────────────────────────────
// InMemoryCacheAccessor
// ─────────────────────────────────────────────────────────────────────────────

func TestInMemoryCacheAccessor_Serialize_Empty(t *testing.T) {
	acc := &auth.InMemoryCacheAccessor{}
	s, err := acc.Serialize()
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if s != "" {
		t.Errorf("expected empty string for empty cache, got %q", s)
	}
}

// stubMarshaler implémente cache.Marshaler pour les tests.
type stubMarshaler struct {
	data []byte
}

func (m *stubMarshaler) Marshal() ([]byte, error) {
	return m.data, nil
}

// stubUnmarshaler implémente cache.Unmarshaler pour les tests.
type stubUnmarshaler struct {
	received []byte
}

func (u *stubUnmarshaler) Unmarshal(data []byte) error {
	u.received = data
	return nil
}

func TestInMemoryCacheAccessor_Export_Then_Serialize(t *testing.T) {
	acc := &auth.InMemoryCacheAccessor{}
	payload := map[string]string{"key": "value"}
	data, _ := json.Marshal(payload)

	ctx := context.Background()
	m := &stubMarshaler{data: data}
	if err := acc.Export(ctx, m, cache.ExportHints{}); err != nil {
		t.Fatalf("Export inattendu: %v", err)
	}

	s, err := acc.Serialize()
	if err != nil {
		t.Fatalf("Serialize inattendu: %v", err)
	}
	if s == "" {
		t.Error("expected non-empty string after Export")
	}
}

func TestInMemoryCacheAccessor_Replace_Empty(t *testing.T) {
	acc := &auth.InMemoryCacheAccessor{}
	ctx := context.Background()
	u := &stubUnmarshaler{}
	// Replace sur cache vide ne doit pas appeler Unmarshal
	if err := acc.Replace(ctx, u, cache.ReplaceHints{}); err != nil {
		t.Fatalf("Replace inattendu: %v", err)
	}
	if len(u.received) != 0 {
		t.Error("Unmarshal ne doit pas être appelé si cache vide")
	}
}

func TestInMemoryCacheAccessor_Export_Then_Replace(t *testing.T) {
	acc := &auth.InMemoryCacheAccessor{}
	payload := map[string]string{"token": "abc"}
	data, _ := json.Marshal(payload)
	ctx := context.Background()

	// Export
	if err := acc.Export(ctx, &stubMarshaler{data: data}, cache.ExportHints{}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Replace doit propager les données à l'Unmarshaler
	u := &stubUnmarshaler{}
	if err := acc.Replace(ctx, u, cache.ReplaceHints{}); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if string(u.received) != string(data) {
		t.Errorf("Replace: expected %q, got %q", data, u.received)
	}
}
