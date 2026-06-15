package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

type stubInspector struct {
	rows map[string]int64
}

func (s stubInspector) CountRows(_ context.Context, _ string, table string) (int64, bool, error) {
	n, ok := s.rows[table]
	return n, ok, nil
}

type stubCaps struct {
	set *mappings.CapabilityMappingSet
}

func (s stubCaps) GetCapabilities(string) (*mappings.CapabilityMappingSet, bool) {
	if s.set == nil {
		return nil, false
	}
	return s.set, true
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

// runDriftDiag prépare un titre temp (config + bases factices) et lance le
// diagnostic AVEC capabilities déclarées → calcul du drift.
func runDriftDiag(t *testing.T, declared map[string]string, rows map[string]int64, createDBs []string) *domain.TitleDiagnostic {
	t.Helper()
	root := t.TempDir()
	slug := "test_title"
	mdir := filepath.Join(root, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"fields.toml", "capabilities.toml"} {
		if err := os.WriteFile(filepath.Join(mdir, f), []byte("x = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wh := filepath.Join(root, "data", "titles", slug, "warehouse")
	if err := os.MkdirAll(wh, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, db := range createDBs {
		if err := os.WriteFile(filepath.Join(wh, db), []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	set := mappings.NewCapabilityMappingSet(slug, 1, declared)
	svc := NewTitleDiagnosticService(root, stubInspector{rows: rows}).WithCapabilities(stubCaps{set: set})
	rep, err := svc.Diagnose(context.Background(), slug)
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	return rep
}

func countDrift(rep *domain.TitleDiagnostic, kind string) int {
	n := 0
	for _, d := range rep.Drifts {
		if d.Kind == kind {
			n++
		}
	}
	return n
}

// TestTitleDiagnosticService_Drift — 4 scénarios déclaré-vs-réalité (PMT-14) :
// no drift / data drift / feature drift / cascade. Réutilise ComputeFeatureMatrix.
func TestTitleDiagnosticService_Drift(t *testing.T) {
	matchHistory := map[string]string{"match.history": "supported"}

	// 1. No drift : match.history supported + match_registry peuplée.
	rep := runDriftDiag(t, matchHistory,
		map[string]int64{"match_registry": 42}, []string{"shared_matches_v2.duckdb"})
	if len(rep.Drifts) != 0 {
		t.Errorf("no-drift : attendu 0 drift, got %+v", rep.Drifts)
	}

	// 2. Data drift : match.history supported mais match_registry vide.
	rep = runDriftDiag(t, matchHistory,
		map[string]int64{"match_registry": 0}, []string{"shared_matches_v2.duckdb"})
	if countDrift(rep, "data") != 1 || countDrift(rep, "feature") != 0 {
		t.Errorf("data-drift : attendu 1 data / 0 feature, got %+v", rep.Drifts)
	}

	// 3. Feature drift : match.detail.core supported sans l'enrichissement scoreboard.
	rep = runDriftDiag(t, map[string]string{"match.detail.core": "supported"},
		map[string]int64{}, nil)
	if countDrift(rep, "feature") != 1 || countDrift(rep, "data") != 0 {
		t.Errorf("feature-drift : attendu 1 feature / 0 data, got %+v", rep.Drifts)
	}

	// 4. Cascade : data drift + feature drift cumulés.
	rep = runDriftDiag(t,
		map[string]string{"match.history": "supported", "match.detail.core": "supported"},
		map[string]int64{"match_registry": 0}, []string{"shared_matches_v2.duckdb"})
	if countDrift(rep, "data") != 1 || countDrift(rep, "feature") != 1 {
		t.Errorf("cascade : attendu 1 data + 1 feature, got %+v", rep.Drifts)
	}
}
