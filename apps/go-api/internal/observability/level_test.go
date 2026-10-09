package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLevelUnlessCanceled(t *testing.T) {
	t.Parallel()
	vivante := context.Background()
	annulee, cancel := context.WithCancel(context.Background())
	cancel()
	cas := []struct {
		nom  string
		ctx  context.Context
		err  error
		want slog.Level
	}{
		{"panne, requête vivante", vivante, errors.New("database is locked"), slog.LevelError},
		{"requête terminée", annulee, errors.New("database is locked"), slog.LevelDebug},
		{"annulation enveloppée, requête vivante", vivante, fmt.Errorf("lecture : %w", context.Canceled), slog.LevelDebug},
		{"échéance interne, requête vivante", vivante, fmt.Errorf("lecture : %w", context.DeadlineExceeded), slog.LevelError},
	}
	for _, c := range cas {
		if got := LevelUnlessCanceled(c.ctx, c.err, slog.LevelError); got != c.want {
			t.Errorf("%s : %v, want %v", c.nom, got, c.want)
		}
	}
}

// predicatEnLigne : la copie en ligne du prédicat de LevelUnlessCanceled pour choisir un
// niveau de journal.
var predicatEnLigne = regexp.MustCompile(`ctx\.Err\(\) != nil \|\| errors\.Is\(err, context\.Canceled\) \{`)

// TestLevelUnlessCanceled_SourceUnique : garde-rail CLAUDE.md n°6 — le prédicat « fin de la
// requête » d'un niveau de journal ne se recopie pas en ligne sous internal/ (hors ce
// fichier) : passer par LevelUnlessCanceled. Une factorisation sans garde-rail re-diverge.
func TestLevelUnlessCanceled_SourceUnique(t *testing.T) {
	t.Parallel()
	var copies []string
	err := filepath.WalkDir("..", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			filepath.ToSlash(path) == "../observability/level.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if predicatEnLigne.Match(data) {
			copies = append(copies, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de internal/ : %v", err)
	}
	if len(copies) > 0 {
		t.Errorf("prédicat de niveau « fin de requête » recopié en ligne : %v — utiliser observability.LevelUnlessCanceled", copies)
	}
}
