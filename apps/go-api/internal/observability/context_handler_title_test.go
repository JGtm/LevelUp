package observability

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// TestContextHandler_TitleAttr — MT-05 (PMT-10) PR-1 : l'attribut "title" est
// ajouté SEULEMENT quand le titre est réellement dans le ctx. Un log background
// (sans titre) reste inchangé — pas de title fantôme.
func TestContextHandler_TitleAttr(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	// 1. ctx sans titre → aucun attribut "title" (byte-identique au comportement legacy).
	logger.InfoContext(context.Background(), "background")
	if strings.Contains(buf.String(), `"title"`) {
		t.Errorf("ctx sans titre : ne doit PAS contenir \"title\", got %s", buf.String())
	}

	// 2. ctx avec titre → attribut "title" présent et exact.
	buf.Reset()
	ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_title_b")
	logger.InfoContext(ctx, "request")
	if !strings.Contains(buf.String(), `"title":"synthetic_title_b"`) {
		t.Errorf("ctx avec titre : doit contenir \"title\":\"synthetic_title_b\", got %s", buf.String())
	}
}
