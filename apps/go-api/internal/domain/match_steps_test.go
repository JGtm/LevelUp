package domain

import (
	"reflect"
	"strings"
	"testing"
)

// TestMatchSteps_EnumOpenAPI : la liste des étapes publiée au contrat OpenAPI (tag `enum` de
// MatchDelayStep.Step) est exactement [MatchSteps], dans le même ordre — la seconde copie de la
// liste ne peut pas diverger de la première.
func TestMatchSteps_EnumOpenAPI(t *testing.T) {
	f, ok := reflect.TypeOf(MatchDelayStep{}).FieldByName("Step")
	if !ok {
		t.Fatal("champ Step absent")
	}
	attendu := make([]string, len(MatchSteps))
	for i, s := range MatchSteps {
		attendu[i] = string(s)
	}
	if got := f.Tag.Get("enum"); got != strings.Join(attendu, ",") {
		t.Fatalf("enum OpenAPI = %q, attendu %q", got, strings.Join(attendu, ","))
	}
}
