package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Un retrait de copie temporaire qui échoue est journalisé (la copie reste sur le disque) ;
// un fichier déjà absent ne l'est pas.
func TestRemoveTempUpload_LogsFailure(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	// Un dossier non vide ne se retire pas par os.Remove.
	dir := t.TempDir()
	stuck := filepath.Join(dir, "openspartan_x.db")
	if err := os.MkdirAll(filepath.Join(stuck, "contenu"), 0o755); err != nil {
		t.Fatal(err)
	}
	removeTempUpload(context.Background(), stuck)
	if !strings.Contains(buf.String(), "retrait de la copie temporaire échoué") || !strings.Contains(buf.String(), stuck) {
		t.Fatalf("échec du retrait non journalisé : %q", buf.String())
	}

	buf.Reset()
	removeTempUpload(context.Background(), filepath.Join(dir, "absent.db"))
	if buf.Len() != 0 {
		t.Fatalf("fichier absent journalisé à tort : %q", buf.String())
	}
}
