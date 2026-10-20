package replay

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
)

// TestNiveauxDesConstatsEgalentSlog : la conversion `slog.Level(d.Niveau)` de
// [JournaliserDiagnostics] n est juste que si les quatre niveaux de `constat` valent ceux de
// `log/slog` (cf. l en-tete du paquet `constat`).
func TestNiveauxDesConstatsEgalentSlog(t *testing.T) {
	paires := []struct {
		constat constat.Niveau
		slog    slog.Level
	}{
		{constat.NiveauDebug, slog.LevelDebug},
		{constat.NiveauInfo, slog.LevelInfo},
		{constat.NiveauWarn, slog.LevelWarn},
		{constat.NiveauError, slog.LevelError},
	}
	for _, p := range paires {
		if slog.Level(p.constat) != p.slog {
			t.Errorf("constat %d != slog %v", p.constat, p.slog)
		}
	}
}

// TestJournaliserDiagnosticsEcritUneLigneParConstat : une ligne par diagnostic, a son niveau, avec
// son code stable.
func TestJournaliserDiagnosticsEcritUneLigneParConstat(t *testing.T) {
	var buf bytes.Buffer
	precedent := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(precedent) })

	JournaliserDiagnostics(context.Background(), []constat.Diagnostic{
		{Code: "test.a", Niveau: constat.NiveauWarn, Message: "premier", Attrs: []any{"k", 1}},
		{Code: "test.b", Niveau: constat.NiveauDebug, Message: "second"},
	})
	lignes := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lignes) != 2 {
		t.Fatalf("%d ligne(s), attendu 2 :\n%s", len(lignes), buf.String())
	}
	for i, attendu := range []string{"level=WARN msg=premier k=1 diagnostic=test.a", "level=DEBUG msg=second diagnostic=test.b"} {
		if !strings.Contains(lignes[i], attendu) {
			t.Errorf("ligne %d = %q, attendu %q", i, lignes[i], attendu)
		}
	}
}
