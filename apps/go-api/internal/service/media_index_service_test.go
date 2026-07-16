// Tests pour MediaIndexer — F.3 du plan de tests.
//
// Couvre :
//   - Le contract d'interface : DirMediaIndexer implemente MediaIndexer
//   - NewDirMediaIndexer construit un indexer non-nil
//   - BuildMediaScanHook propage les 2 closures (capturesBaseDir + timezone)
//
// Note : les chemins fonctionnels (scan filesystem + DuckDB + ffprobe) sont
// testes par les tests d'integration existants (handlers/settings.go test
// + e2e ops.IndexMedia). Ici on assure le contract minimum.
package service

import (
	"context"
	"sync/atomic"
	"testing"
)

// TestDirMediaIndexer_ImplementsInterface verifie le contract MediaIndexer.
// Sentinelle : si quelqu'un supprime/renomme ResetAndReindex ou ScanAllMedia
// sur DirMediaIndexer, ce test ne compile plus.
func TestDirMediaIndexer_ImplementsInterface(t *testing.T) {
	var _ MediaIndexer = (*DirMediaIndexer)(nil)
	var _ MediaIndexer = NewDirMediaIndexer()
}

// TestNewDirMediaIndexer_NotNil : constructeur retourne instance non-nil.
func TestNewDirMediaIndexer_NotNil(t *testing.T) {
	idx := NewDirMediaIndexer()
	if idx == nil {
		t.Fatal("NewDirMediaIndexer() = nil")
	}
}

// TestMediaIndexer_InterfaceMethods : sentinelle anti-regression sur les
// signatures methodes de l'interface. Si quelqu'un change la signature,
// le compilateur fail ce test.
func TestMediaIndexer_InterfaceMethods(t *testing.T) {
	var idx MediaIndexer = NewDirMediaIndexer()
	_ = idx // suppress unused
	// Les methodes ResetAndReindex et ScanAllMedia sont publiques ici via
	// l'interface — assertion compile-time uniquement.
	t.Log("MediaIndexer interface contract: ResetAndReindex + ScanAllMedia")
}

// TestBuildMediaScanHook_PropagatesTimezone est l'anti-régression directe du
// bug observé 2026-05-25 : BuildMediaScanHook ignorait la timezone et appelait
// IndexMedia avec opts.Timezone="", ce qui désactive la regex filename → 0
// associations match pendant 4 jours. Le test vérifie que la closure timezoneFn
// EST appelée à chaque invocation du hook (lecture live des settings).
func TestBuildMediaScanHook_PropagatesTimezone(t *testing.T) {
	var capturesFnCalls atomic.Int32
	var timezoneFnCalls atomic.Int32

	capturesFn := func() string {
		capturesFnCalls.Add(1)
		return "" // dir absent → hook return early, on observe juste les calls
	}
	timezoneFn := func() string {
		timezoneFnCalls.Add(1)
		return "Europe/Paris"
	}

	hook := BuildMediaScanHook("/tmp/nonexistent-repo", "fake-gamertag", capturesFn, timezoneFn, nil)
	hook(context.Background())

	if capturesFnCalls.Load() != 1 {
		t.Errorf("capturesBaseDirFn appelée %d fois, want 1", capturesFnCalls.Load())
	}
	if timezoneFnCalls.Load() != 1 {
		t.Errorf("timezoneFn appelée %d fois, want 1 (sans ça, opts.Timezone vide → bug 2026-05-25)", timezoneFnCalls.Load())
	}
}

// TestBuildMediaScanHook_NilTimezoneFn_DoesNotPanic vérifie la défense contre
// caller négligent qui passerait nil pour timezoneFn (rétro-compat avec
// d'éventuels mocks de tests externes).
func TestBuildMediaScanHook_NilTimezoneFn_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("BuildMediaScanHook panic avec timezoneFn=nil : %v", r)
		}
	}()
	hook := BuildMediaScanHook("/tmp/nonexistent-repo", "fake-gamertag",
		func() string { return "" },
		nil, // timezoneFn nil
		nil, // deleteSourceFn nil
	)
	hook(context.Background())
}
