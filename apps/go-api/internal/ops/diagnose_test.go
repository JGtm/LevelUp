// Package ops — diagnose_test.go : tests pour le diagnostic DB.
//
//go:build integration

package ops

import (
	"testing"
)

func TestFormatDiagnoseReport_Empty(t *testing.T) {
	report := DiagnoseReport{
		DBPath: "test.duckdb",
	}
	result := FormatDiagnoseReport(report)
	if result == "" {
		t.Error("expected non-empty formatted report")
	}
}

func TestFormatDiagnoseReport_WithTables(t *testing.T) {
	report := DiagnoseReport{
		DBPath: "test.duckdb",
		Tables: []TableSchema{
			{Name: "match_registry", RowCount: 100, Columns: []ColumnInfo{{Name: "match_id", DataType: "VARCHAR"}}},
		},
	}
	result := FormatDiagnoseReport(report)
	if result == "" {
		t.Error("expected non-empty formatted report")
	}
}
