// Package lab — tests unitaires des helpers PURS de diff de contrats OpenAPI
// (zéro DB, zéro filesystem). Les chemins DB (loadAssets/loadMedals/...) sont
// integration-only et restent non couverts en unit par construction.
package lab

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestCompareOpenAPIRoutes : couvre les trois sorties du diff —
//   - missing : route FastAPI absente côté Go ;
//   - extra : route Go absente côté FastAPI ;
//   - mismatch : même route (modulo normalisation des params de chemin) mais
//     méthodes divergentes. La normalisation {id}→{*} doit faire matcher
//     /users/{id} (FastAPI) avec /users/{userId} (Go).
func TestCompareOpenAPIRoutes(t *testing.T) {
	fastapi := map[string][]string{
		"/users/{id}": {"get", "post"},
		"/legacy":     {"get"}, // absente côté Go → missing
		"/shared":     {"get"},
	}
	goRoutes := map[string][]string{
		"/users/{userId}": {"get"},  // même route normalisée, méthodes ≠ → mismatch
		"/new":            {"post"}, // absente côté FastAPI → extra
		"/shared":         {"get"},  // identique → rien
	}

	missing, extra, mismatches := compareOpenAPIRoutes(fastapi, goRoutes)

	if len(missing) != 1 || missing[0].Path != "/legacy" {
		t.Errorf("missing = %v, want [/legacy]", missing)
	}
	if len(extra) != 1 || extra[0].Path != "/new" {
		t.Errorf("extra = %v, want [/new]", extra)
	}
	if len(mismatches) != 1 {
		t.Fatalf("mismatches = %d, want 1\n%#v", len(mismatches), mismatches)
	}
	mm := mismatches[0]
	if mm.FastAPIPath != "/users/{id}" || mm.GoPath != "/users/{userId}" {
		t.Errorf("mismatch paths = (%q,%q)", mm.FastAPIPath, mm.GoPath)
	}
	if !reflect.DeepEqual(mm.MissingMethods, []string{"post"}) {
		t.Errorf("MissingMethods = %v, want [post]", mm.MissingMethods)
	}
}

// TestCompareOpenAPIRoutes_Identical : deux specs identiques → aucune divergence.
func TestCompareOpenAPIRoutes_Identical(t *testing.T) {
	routes := map[string][]string{"/a": {"get"}, "/b": {"get", "post"}}
	missing, extra, mismatches := compareOpenAPIRoutes(routes, routes)
	if len(missing) != 0 || len(extra) != 0 || len(mismatches) != 0 {
		t.Errorf("specs identiques → diff non vide: missing=%v extra=%v mismatch=%v", missing, extra, mismatches)
	}
}

// TestLabContractStatus : OK ssi ni missing ni mismatch ; DIVERGENCES sinon
// (un extra seul ne suffit PAS à passer DIVERGENCES — c'est volontaire, le statut
// est piloté par missing+mismatch dans labContractStatus).
func TestLabContractStatus(t *testing.T) {
	if got := labContractStatus(nil, nil); got != "OK" {
		t.Errorf("status(vide) = %q, want OK", got)
	}
	withMissing := []domain.LabRouteMethods{{Path: "/x"}}
	if got := labContractStatus(withMissing, nil); got != "DIVERGENCES" {
		t.Errorf("status(missing) = %q, want DIVERGENCES", got)
	}
	withMismatch := []domain.LabMethodMismatch{{FastAPIPath: "/y"}}
	if got := labContractStatus(nil, withMismatch); got != "DIVERGENCES" {
		t.Errorf("status(mismatch) = %q, want DIVERGENCES", got)
	}
}

func TestSameMethods(t *testing.T) {
	cases := []struct {
		l, r []string
		want bool
	}{
		{[]string{"get", "post"}, []string{"get", "post"}, true},
		{[]string{"get"}, []string{"get", "post"}, false},
		{[]string{"get", "post"}, []string{"post", "get"}, false}, // ordre significatif (les listes sont pré-triées par l'appelant)
		{nil, nil, true},
	}
	for _, tc := range cases {
		if got := sameMethods(tc.l, tc.r); got != tc.want {
			t.Errorf("sameMethods(%v,%v) = %v, want %v", tc.l, tc.r, got, tc.want)
		}
	}
}

func TestDiffMethods(t *testing.T) {
	got := diffMethods([]string{"get", "post", "delete"}, []string{"get"})
	if !reflect.DeepEqual(got, []string{"post", "delete"}) {
		t.Errorf("diffMethods = %v, want [post delete]", got)
	}
	if got := diffMethods([]string{"get"}, []string{"get", "post"}); got != nil {
		t.Errorf("diffMethods(sous-ensemble) = %v, want nil", got)
	}
}

func TestLikeQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Foo", "%foo%"},
		{"  Bar  ", "%bar%"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		if got := likeQuery(tc.in); got != tc.want {
			t.Errorf("likeQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestIsMissingRelationError : ne reconnaît QUE le pattern DuckDB "catalog error
// ... does not exist" (table absente, tolérée en best-effort), pas les autres
// erreurs SQL ni nil.
func TestIsMissingRelationError(t *testing.T) {
	if !isMissingRelationError(errors.New("Catalog Error: Table x does not exist")) {
		t.Error("doit reconnaître le catalog error 'does not exist'")
	}
	if isMissingRelationError(errors.New("Constraint Error: duplicate key")) {
		t.Error("ne doit PAS matcher une autre erreur SQL")
	}
	if isMissingRelationError(fmt.Errorf("wrapped: %w", errors.New("catalog error does not exist"))) != true {
		t.Error("doit matcher à travers un wrap (Error() contient le pattern)")
	}
	if isMissingRelationError(nil) {
		t.Error("nil ne doit jamais matcher")
	}
}

func TestOrEmptyCSR(t *testing.T) {
	if got := orEmptyCSR(nil); got == nil || len(got) != 0 {
		t.Errorf("orEmptyCSR(nil) = %v, want slice vide non-nil", got)
	}
	in := []domain.CSRSeasonCalendar{{}}
	if got := orEmptyCSR(in); len(got) != 1 {
		t.Errorf("orEmptyCSR(non-vide) doit préserver le contenu, got len=%d", len(got))
	}
}
