package testutil

import (
	"fmt"
	"testing"
)

type dtoOK struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type dtoNilSlice struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"` // sans omitempty : nil → null JSON → CRASH front
}

type dtoNilSliceOmitempty struct {
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"` // toléré : sera omis du JSON
}

type dtoNested struct {
	Inner dtoOK   `json:"inner"`
	Items []dtoOK `json:"items"`
}

func TestRequireNoNilSlicesWithoutOmitempty_OK_EmptyInitialised(t *testing.T) {
	v := dtoOK{Name: "x", Tags: []string{}}
	mt := &mockT{}
	RequireNoNilSlicesWithoutOmitempty(mt, v)
	if mt.failed {
		t.Errorf("expected no failure, got errors: %v", mt.errors)
	}
}

func TestRequireNoNilSlicesWithoutOmitempty_Fail_NilWithoutOmitempty(t *testing.T) {
	v := dtoNilSlice{Name: "x"} // Tags reste nil
	mt := &mockT{}
	RequireNoNilSlicesWithoutOmitempty(mt, v)
	if !mt.failed {
		t.Errorf("expected failure on nil slice without omitempty")
	}
}

func TestRequireNoNilSlicesWithoutOmitempty_OK_NilWithOmitempty(t *testing.T) {
	v := dtoNilSliceOmitempty{Name: "x"} // Tags nil, mais omitempty → OK
	mt := &mockT{}
	RequireNoNilSlicesWithoutOmitempty(mt, v)
	if mt.failed {
		t.Errorf("expected no failure (omitempty tolerated), got: %v", mt.errors)
	}
}

func TestRequireNoNilSlicesWithoutOmitempty_RecursesIntoStructsAndSlices(t *testing.T) {
	v := dtoNested{
		Inner: dtoOK{Tags: []string{}},
		Items: []dtoOK{{Tags: nil}}, // Tags nil dans un élément du slice
	}
	mt := &mockT{}
	RequireNoNilSlicesWithoutOmitempty(mt, v)
	if !mt.failed {
		t.Errorf("expected failure: nested struct in slice has nil Tags")
	}
}

// mockT capture les appels Errorf sans faire échouer le test parent.
type mockT struct {
	failed bool
	errors []string
}

func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}
func (m *mockT) Helper() {}
