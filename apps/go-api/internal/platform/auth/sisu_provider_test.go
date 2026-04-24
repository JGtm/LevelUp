// Package auth — sisu_provider_test.go : tests unitaires pour sisu_provider.go.
package auth

import (
	"context"
	"testing"
)

// TestSISUProvider_ExchangeWithoutInit vérifie que Exchange panics si InitDeviceFlow
// n'a pas été appelé au préalable (protection contre un bug d'utilisation).
func TestSISUProvider_ExchangeWithoutInit(t *testing.T) {
	p := NewSISUProvider()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Exchange sans InitDeviceFlow préalable devrait paniquer")
		}
		msg, ok := r.(string)
		if !ok || msg == "" {
			t.Fatalf("valeur de panic inattendue : %v", r)
		}
	}()

	// Doit paniquer : aucun InitDeviceFlow n'a été appelé.
	_, _ = p.Exchange(context.Background(), "fake-access-token") //nolint:errcheck
}
