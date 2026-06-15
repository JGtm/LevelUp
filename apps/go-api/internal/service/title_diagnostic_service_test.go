package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

type stubInspector struct {
	rows map[string]int64
}

func (s stubInspector) CountRows(_ context.Context, _ string, table string) (int64, bool, error) {
	n, ok := s.rows[table]
	return n, ok, nil
}

func cfgByName(rep *domain.TitleDiagnostic, name string) (domain.ConfigFileStatus, bool) {
	for _, c := range rep.ConfigFiles {
		if c.Name == name {
			return c, true
		}
	}
	return domain.ConfigFileStatus{}, false
}

func dbByName(rep *domain.TitleDiagnostic, name string) (domain.DatabaseStatus, bool) {
	for _, d := range rep.Databases {
		if d.Name == name {
			return d, true
		}
	}
	return domain.DatabaseStatus{}, false
}

func TestTitleDiagnosticService_Diagnose(t *testing.T) {
	root := t.TempDir()
	slug := "test_title"

	// Config : fields.toml présent, capabilities.toml absent.
	mappingsDir := filepath.Join(root, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(mappingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mappingsDir, "fields.toml"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Base metadata présente (fichier factice → os.Stat OK) ; le stub inspecteur
	// fournit les comptes sans ouvrir le fichier.
	metaPath := filepath.Join(root, "data", "titles", slug, "warehouse", "metadata.duckdb")
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewTitleDiagnosticService(root, stubInspector{rows: map[string]int64{
		"season_calendars": 5,
		"career_ranks":     272,
		// asset_translations absent → exists=false
	}})

	rep, err := svc.Diagnose(context.Background(), slug)
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if rep.TitleSlug != slug {
		t.Errorf("slug = %q, want %q", rep.TitleSlug, slug)
	}

	// Config : fields présent+requis, capabilities absent+requis.
	if f, ok := cfgByName(rep, "fields.toml"); !ok || !f.Present || !f.Required {
		t.Errorf("fields.toml = %+v (want present+required)", f)
	}
	if c, ok := cfgByName(rep, "capabilities.toml"); !ok || c.Present || !c.Required {
		t.Errorf("capabilities.toml = %+v (want absent+required)", c)
	}

	// metadata présente avec comptes ; shared absente.
	meta, ok := dbByName(rep, "metadata.duckdb")
	if !ok || !meta.Exists {
		t.Fatalf("metadata.duckdb = %+v (want exists)", meta)
	}
	var seasonRows int64 = -1
	for _, tbl := range meta.Tables {
		if tbl.Name == "season_calendars" {
			seasonRows = tbl.Rows
			if !tbl.Exists {
				t.Errorf("season_calendars devrait exister")
			}
		}
		if tbl.Name == "asset_translations" && tbl.Exists {
			t.Errorf("asset_translations ne devrait pas exister (stub)")
		}
	}
	if seasonRows != 5 {
		t.Errorf("season_calendars rows = %d, want 5", seasonRows)
	}
	if shared, ok := dbByName(rep, "shared_matches_v2.duckdb"); !ok || shared.Exists {
		t.Errorf("shared_matches_v2.duckdb = %+v (want absent)", shared)
	}
}
