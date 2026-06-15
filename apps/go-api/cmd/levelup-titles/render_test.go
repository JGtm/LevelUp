//go:build cgo

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func fixedReport() *domain.TitleDiagnostic {
	return &domain.TitleDiagnostic{
		TitleSlug: "synthetic_title_b",
		ConfigFiles: []domain.ConfigFileStatus{
			{Name: "fields.toml", Present: true, Required: true},
			{Name: "capabilities.toml", Present: false, Required: true},
		},
		Databases: []domain.DatabaseStatus{
			{Name: "shared_matches_v2.duckdb", Exists: true, Tables: []domain.TableStatus{
				{Name: "match_registry", Exists: true, Rows: 42},
				{Name: "match_participants", Exists: false},
			}},
			{Name: "shared_pve.duckdb", Exists: false},
		},
	}
}

func TestRenderDiagnosticText_Golden(t *testing.T) {
	got := renderDiagnosticText(fixedReport())
	want := strings.Join([]string{
		"Diagnostic du titre : synthetic_title_b",
		"",
		"Mappings TOML :",
		"  fields.toml : present (requis)",
		"  capabilities.toml : manquant (requis)",
		"",
		"Bases de donnees :",
		"  shared_matches_v2.duckdb : presente",
		"    match_registry : 42 lignes",
		"    match_participants : absente",
		"  shared_pve.duckdb : absente",
		"",
	}, "\n")
	if got != want {
		t.Errorf("golden text mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderDiagnosticJSON_Golden(t *testing.T) {
	b, err := json.Marshal(fixedReport())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"title_slug":"synthetic_title_b","config_files":[{"name":"fields.toml","present":true,"required":true},{"name":"capabilities.toml","present":false,"required":true}],"databases":[{"name":"shared_matches_v2.duckdb","exists":true,"tables":[{"name":"match_registry","exists":true,"rows":42},{"name":"match_participants","exists":false,"rows":0}]},{"name":"shared_pve.duckdb","exists":false}]}`
	if string(b) != want {
		t.Errorf("golden json mismatch:\ngot:  %s\nwant: %s", string(b), want)
	}
}
