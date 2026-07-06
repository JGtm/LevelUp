// Package api — test_helpers_io.go : helpers IO partagés entre tests.
//
// `writeFileImpl` est extrait pour éviter la dépendance directe à `os` dans les
// fichiers de tests qui visent à rester minimaux.
//
//go:build !prod

package wire

import "os"

func writeFileImpl(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
