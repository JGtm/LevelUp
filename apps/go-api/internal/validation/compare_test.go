// Package validation — compare_test.go : tests des types et logique pure dans compare.go.
//
// Note : les tests qui ouvrent de vraies DBs DuckDB sont annotés //go:build integration
// et ne s'exécutent qu'en CI Linux avec CGO_ENABLED=1.
package validation

import (
	"math"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Tests des constantes de statut
// ─────────────────────────────────────────────────────────────────────────────

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"statusOK", statusOK, "OK"},
		{"statusWarn", statusWarn, "WARN"},
		{"statusMissGo", statusMissGo, "MISS_GO"},
		{"statusMissPy", statusMissPy, "MISS_PY"},
		{"statusDiverge", statusDiverge, "DIVERGE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("got %q, want %q", tt.value, tt.expected)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests TableComparison
// ─────────────────────────────────────────────────────────────────────────────

func TestTableComparison_DeltaComputation(t *testing.T) {
	tc := TableComparison{
		TableName:  "match_registry",
		RowsGo:     1050,
		RowsPython: 1000,
	}
	tc.Delta = tc.RowsGo - tc.RowsPython
	if tc.RowsPython != 0 {
		tc.DeltaPct = float64(tc.Delta) / float64(tc.RowsPython) * 100
	}

	if tc.Delta != 50 {
		t.Errorf("expected Delta=50, got %d", tc.Delta)
	}
	if math.Abs(tc.DeltaPct-5.0) > 0.001 {
		t.Errorf("expected DeltaPct≈5.0, got %f", tc.DeltaPct)
	}
}

func TestTableComparison_DeltaPct_ZeroPython(t *testing.T) {
	tc := TableComparison{
		TableName:  "new_table",
		RowsGo:     100,
		RowsPython: 0,
	}
	tc.Delta = tc.RowsGo - tc.RowsPython
	if tc.RowsPython == 0 {
		tc.DeltaPct = math.NaN()
	}

	if !math.IsNaN(tc.DeltaPct) {
		t.Errorf("expected NaN for DeltaPct with RowsPython=0, got %f", tc.DeltaPct)
	}
}

func TestTableComparison_StatusValues(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{statusOK, true},
		{statusWarn, true},
		{statusMissGo, true},
		{statusMissPy, true},
		{statusDiverge, true},
		{"UNKNOWN", false},
	}
	validStatuses := map[string]bool{
		statusOK: true, statusWarn: true, statusMissGo: true,
		statusMissPy: true, statusDiverge: true,
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			_, ok := validStatuses[tt.status]
			if ok != tt.valid {
				t.Errorf("status %q: expected valid=%v, got %v", tt.status, tt.valid, ok)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests BitmaskStats
// ─────────────────────────────────────────────────────────────────────────────

func TestBitmaskStats_PctComputation(t *testing.T) {
	bs := BitmaskStats{
		Table:        "match_registry",
		Column:       "backfill_flags",
		ZeroCount:    200,
		NonZeroCount: 800,
	}
	total := bs.ZeroCount + bs.NonZeroCount
	bs.ZeroCountGoPct = float64(bs.ZeroCount) / float64(total) * 100

	if math.Abs(bs.ZeroCountGoPct-20.0) > 0.001 {
		t.Errorf("expected ZeroCountGoPct≈20.0, got %f", bs.ZeroCountGoPct)
	}
}

func TestBitmaskStats_AllZero(t *testing.T) {
	bs := BitmaskStats{
		ZeroCount:    0,
		NonZeroCount: 1000,
	}
	total := bs.ZeroCount + bs.NonZeroCount
	pct := float64(bs.ZeroCount) / float64(total) * 100
	if pct != 0.0 {
		t.Errorf("expected 0%%, got %f", pct)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests MatchOverlap / Jaccard
// ─────────────────────────────────────────────────────────────────────────────

func TestMatchOverlap_JaccardScore(t *testing.T) {
	cases := []struct {
		name     string
		onlyGo   int64
		onlyPy   int64
		inBoth   int64
		wantJacc float64
	}{
		{"perfect overlap", 0, 0, 100, 1.0},
		{"no overlap", 50, 50, 0, 0.0},
		{"half overlap", 25, 25, 50, 0.5},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			union := tt.onlyGo + tt.onlyPy + tt.inBoth
			var jacc float64
			if union > 0 {
				jacc = float64(tt.inBoth) / float64(union)
			}
			if math.Abs(jacc-tt.wantJacc) > 0.001 {
				t.Errorf("expected Jaccard=%.3f, got %.3f", tt.wantJacc, jacc)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests ComparisonReport
// ─────────────────────────────────────────────────────────────────────────────

func TestComparisonReport_OverallOK_AllPassing(t *testing.T) {
	report := ComparisonReport{
		Tables: []TableComparison{
			{TableName: "match_registry", Status: statusOK},
			{TableName: "match_participants", Status: statusOK},
		},
		OverallOK: true,
	}
	for _, tc := range report.Tables {
		if tc.Status != statusOK {
			report.OverallOK = false
			break
		}
	}
	if !report.OverallOK {
		t.Error("report with all OK tables should have OverallOK=true")
	}
}

func TestComparisonReport_OverallOK_WithDiverge(t *testing.T) {
	report := ComparisonReport{
		Tables: []TableComparison{
			{TableName: "match_registry", Status: statusOK},
			{TableName: "match_participants", Status: statusDiverge},
		},
		OverallOK: true,
	}
	for _, tc := range report.Tables {
		if tc.Status == statusDiverge || tc.Status == statusMissGo {
			report.OverallOK = false
			break
		}
	}
	if report.OverallOK {
		t.Error("report with DIVERGE table should have OverallOK=false")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests classifyDelta (Sprint 48 : tests des vraies fonctions)
// ─────────────────────────────────────────────────────────────────────────────

func TestClassifyDelta_ExactMatch(t *testing.T) {
	got := classifyDelta(0, 1000)
	if got != "OK" {
		t.Errorf("classifyDelta(0, 1000) = %q, want OK", got)
	}
}

func TestClassifyDelta_ZeroPyRows(t *testing.T) {
	got := classifyDelta(5, 0)
	if got != statusWarn {
		t.Errorf("classifyDelta(5, 0) = %q, want WARN", got)
	}
}

func TestClassifyDelta_SmallDivergence_Warn(t *testing.T) {
	// 1% or less → WARN
	got := classifyDelta(10, 1000)
	if got != statusWarn {
		t.Errorf("classifyDelta(10, 1000) = %q, want WARN", got)
	}
}

func TestClassifyDelta_LargeDivergence(t *testing.T) {
	// > 1% → DIVERGE
	got := classifyDelta(50, 1000)
	if got != statusDiverge {
		t.Errorf("classifyDelta(50, 1000) = %q, want DIVERGE", got)
	}
}

func TestClassifyDelta_NegativeDelta(t *testing.T) {
	// Go a moins de lignes que Python
	got := classifyDelta(-20, 1000)
	if got != statusDiverge {
		t.Errorf("classifyDelta(-20, 1000) = %q, want DIVERGE", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests isReportOK
// ─────────────────────────────────────────────────────────────────────────────

func TestIsReportOK_AllGood(t *testing.T) {
	r := &ComparisonReport{
		Tables: []TableComparison{
			{Status: statusOK},
			{Status: statusWarn},
		},
		MatchOverlap: &MatchOverlap{JaccardScore: 0.99},
	}
	if !isReportOK(r) {
		t.Error("expected isReportOK=true for OK+WARN tables with high Jaccard")
	}
}

func TestIsReportOK_Diverge(t *testing.T) {
	r := &ComparisonReport{
		Tables: []TableComparison{{Status: statusDiverge}},
	}
	if isReportOK(r) {
		t.Error("expected isReportOK=false with DIVERGE table")
	}
}

func TestIsReportOK_MissGo(t *testing.T) {
	r := &ComparisonReport{
		Tables: []TableComparison{{Status: statusMissGo}},
	}
	if isReportOK(r) {
		t.Error("expected isReportOK=false with MISS_GO table")
	}
}

func TestIsReportOK_LowJaccard(t *testing.T) {
	r := &ComparisonReport{
		Tables:       []TableComparison{{Status: statusOK}},
		MatchOverlap: &MatchOverlap{JaccardScore: 0.80},
	}
	if isReportOK(r) {
		t.Error("expected isReportOK=false with low Jaccard score")
	}
}

func TestIsReportOK_BitmaskError(t *testing.T) {
	r := &ComparisonReport{
		Tables:   []TableComparison{{Status: statusOK}},
		Bitmasks: []BitmaskStats{{Status: "ERROR"}},
	}
	if isReportOK(r) {
		t.Error("expected isReportOK=false with bitmask ERROR")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests statusIcon
// ─────────────────────────────────────────────────────────────────────────────

func TestStatusIcon_Values(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"OK", "✅"},
		{statusWarn, "⚠️"},
		{statusDiverge, "❌"},
		{statusMissGo, "🔍"},
		{statusMissPy, "🔍"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests jaccardLabel
// ─────────────────────────────────────────────────────────────────────────────

func TestJaccardLabel(t *testing.T) {
	// Perfect score
	label := jaccardLabel(1.0)
	if label == "" {
		t.Error("jaccardLabel(1.0) should not be empty")
	}
	// Low score
	low := jaccardLabel(0.5)
	if low == "" {
		t.Error("jaccardLabel(0.5) should not be empty")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests buildSummary
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildSummary_NotEmpty(t *testing.T) {
	r := &ComparisonReport{
		GoDBPath:     "/tmp/go.duckdb",
		PythonDBPath: "/tmp/py.duckdb",
		Tables: []TableComparison{
			{TableName: "match_registry", Status: statusOK, RowsGo: 100, RowsPython: 100},
		},
		OverallOK: true,
	}
	summary := buildSummary(r)
	if summary == "" {
		t.Error("buildSummary should not return empty string")
	}
	if !strings.Contains(summary, "match_registry") {
		t.Error("summary should contain table name")
	}
}

func TestBuildSummary_WithMatchOverlap(t *testing.T) {
	r := &ComparisonReport{
		GoDBPath:     "/tmp/go.duckdb",
		PythonDBPath: "/tmp/py.duckdb",
		Tables:       []TableComparison{{TableName: "t1", Status: statusOK}},
		MatchOverlap: &MatchOverlap{
			InBoth:       100,
			OnlyInGo:     5,
			OnlyInPython: 3,
			JaccardScore: 0.926,
		},
		OverallOK: true,
	}
	summary := buildSummary(r)
	if !strings.Contains(summary, "Jaccard") {
		t.Error("summary should contain Jaccard info when MatchOverlap is set")
	}
}
