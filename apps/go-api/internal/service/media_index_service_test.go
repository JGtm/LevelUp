// Tests pour MediaIndexer — F.3 du plan de tests.
//
// Couvre :
//   - Le contract d'interface : DirMediaIndexer implemente MediaIndexer
//   - NewDirMediaIndexer construit un indexer non-nil
//
// Note : les chemins fonctionnels (scan filesystem + DuckDB + ffprobe) sont
// testes par les tests d'integration existants (handlers/settings.go test
// + e2e). Ici on assure le contract minimum.
package service

import "testing"

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
