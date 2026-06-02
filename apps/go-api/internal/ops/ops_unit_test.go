package ops

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDiagnoseReport(t *testing.T) {
	r := DiagnoseReport{
		DBPath: "/test/db.duckdb",
		Tables: []TableSchema{
			{Name: "match_registry", Columns: []ColumnInfo{{Name: "id"}, {Name: "date"}}, RowCount: 100},
			{Name: "medals_earned", Columns: []ColumnInfo{{Name: "id"}}, RowCount: -1},
		},
		Views:   []string{"v_match_full", "v_gamertag_lookup"},
		Indexes: []IndexInfo{{Name: "idx_match_id", TableName: "match_registry"}},
	}
	out := FormatDiagnoseReport(r)
	if !strings.Contains(out, "match_registry") {
		t.Error("expected table name")
	}
	if !strings.Contains(out, "Tables (2)") {
		t.Error("expected table count")
	}
	if !strings.Contains(out, "v_match_full") {
		t.Error("expected view name")
	}
	if !strings.Contains(out, "Index (1)") {
		t.Error("expected index count")
	}
}

func TestHealthReport_Summary_OK(t *testing.T) {
	r := HealthReport{
		OK: true,
		Checks: []HealthCheck{
			{Name: "DB check", OK: true, Message: "ok"},
			{Name: "File check", OK: true, Message: "present"},
		},
		Duration: 150 * time.Millisecond,
	}
	out := r.Summary()
	if !strings.Contains(out, "TOUT OK") {
		t.Error("expected TOUT OK")
	}
	if !strings.Contains(out, "[OK]") {
		t.Error("expected success marker [OK]")
	}
}

func TestHealthReport_Summary_WithErrors(t *testing.T) {
	r := HealthReport{
		OK: false,
		Checks: []HealthCheck{
			{Name: "DB check", OK: true, Message: "ok"},
			{Name: "Missing file", OK: false, Message: "not found"},
		},
		Duration: 50 * time.Millisecond,
	}
	out := r.Summary()
	if !strings.Contains(out, "ERREURS DÉTECTÉES") {
		t.Error("expected error message")
	}
	if !strings.Contains(out, "[KO]") {
		t.Error("expected error marker [KO]")
	}
}

func TestCheckDirExists_Missing(t *testing.T) {
	c := checkDirExists("test", "/nonexistent/path")
	if c.OK {
		t.Error("expected failure for missing dir")
	}
}

func TestCheckDirExists_Present(t *testing.T) {
	c := checkDirExists("test", t.TempDir())
	if !c.OK {
		t.Errorf("expected success for existing dir: %s", c.Message)
	}
}

func TestCheckFileExists_Missing(t *testing.T) {
	c := checkFileExists("test", "/nonexistent/file.txt")
	if c.OK {
		t.Error("expected failure for missing file")
	}
}
