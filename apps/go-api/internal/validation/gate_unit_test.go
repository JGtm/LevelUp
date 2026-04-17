package validation

import (
	"strings"
	"testing"
	"time"
)

func TestGateReport_Format_AllPassed(t *testing.T) {
	r := &GateReport{
		GeneratedAt: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		Items: []GateItem{
			{ID: "G1", Label: "Binary exists", Passed: true},
			{ID: "G2", Label: "DB accessible", Passed: true},
		},
		AllPassed: true,
	}
	out := r.Format()
	if !strings.Contains(out, "GATE PHASE 4 VALIDÉE") {
		t.Error("expected success message")
	}
	if !strings.Contains(out, "2/2") {
		t.Error("expected 2/2 criteria")
	}
}

func TestGateReport_Format_WithFailures(t *testing.T) {
	r := &GateReport{
		GeneratedAt: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		Items: []GateItem{
			{ID: "G1", Label: "Binary exists", Passed: true},
			{ID: "G2", Label: "DB accessible", Passed: false, Message: "file not found"},
		},
		AllPassed: false,
	}
	out := r.Format()
	if !strings.Contains(out, "NON VALIDÉE") {
		t.Error("expected failure message")
	}
	if !strings.Contains(out, "1 échec") {
		t.Error("expected 1 failure count")
	}
	if !strings.Contains(out, "file not found") {
		t.Error("expected error message detail")
	}
}

func TestGateReport_Format_Empty(t *testing.T) {
	r := &GateReport{
		GeneratedAt: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		Items:       nil,
		AllPassed:   true,
	}
	out := r.Format()
	if out == "" {
		t.Error("expected non-empty output")
	}
}
