// apierror_schema_test.go — garde-rail du SchemaTransformer d'apiError
// (reliquat V72-01 « sémantiques non exprimables en tag struct »).
package humacore

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// apiErrorSchema régénère le schéma d'apiError via un registre Huma NEUF —
// indépendant du routage et du document partagé.
func apiErrorSchema(t *testing.T) *huma.Schema {
	t.Helper()
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := reg.Schema(reflect.TypeOf(apiError{}), false, "ApiError")
	if s == nil {
		t.Fatal("schéma ApiError nil")
	}
	return s
}

// TestApiErrorSchema_DetailsOneOf — `details` est `any` côté Go (Huma en dérive un
// schéma vide) : le SchemaTransformer doit restaurer `oneOf: [object, array]`.
func TestApiErrorSchema_DetailsOneOf(t *testing.T) {
	details, ok := apiErrorSchema(t).Properties["details"]
	if !ok || details == nil {
		t.Fatal("propriété details absente du schéma ApiError")
	}
	if len(details.OneOf) != 2 {
		t.Fatalf("details.oneOf = %d branche(s), attendu 2", len(details.OneOf))
	}
	obj, arr := details.OneOf[0], details.OneOf[1]
	if obj.Type != huma.TypeObject {
		t.Errorf("1re branche type=%q, attendu %q", obj.Type, huma.TypeObject)
	}
	// Sans additionalProperties explicite, un `{type: object}` nu se traduit côté
	// client généré par « objet sans aucune clé autorisée » — l'inverse du contrat.
	if obj.AdditionalProperties != true {
		t.Errorf("1re branche additionalProperties=%v, attendu true", obj.AdditionalProperties)
	}
	if arr.Type != huma.TypeArray {
		t.Errorf("2e branche type=%q, attendu %q", arr.Type, huma.TypeArray)
	}
}

// TestApiErrorSchema_TransformerKeepsDerivedShape — le transformer ENRICHIT le
// schéma dérivé, il ne le remplace pas : propriétés et `required` restent ceux des
// tags struct.
func TestApiErrorSchema_TransformerKeepsDerivedShape(t *testing.T) {
	s := apiErrorSchema(t)
	for _, name := range []string{"code", "message", "retryable", "details", "field_errors"} {
		if _, ok := s.Properties[name]; !ok {
			t.Errorf("propriété %q perdue par le transformer", name)
		}
	}
	if got, want := s.Required, []string{"code", "message", "retryable"}; !reflect.DeepEqual(got, want) {
		t.Errorf("required = %v, attendu %v", got, want)
	}
	if s.Properties["code"].Description == "" {
		t.Error("description de `code` perdue (tag doc:)")
	}
}

// TestNewError_RuntimeBodyUnchanged — le transformer n'agit QUE sur le contrat :
// le corps JSON runtime reste {code, message, retryable} (details/field_errors
// omitempty, jamais peuplés par NewError).
func TestNewError_RuntimeBodyUnchanged(t *testing.T) {
	body, err := json.Marshal(NewError(http.StatusNotFound, "player_not_found", "Joueur introuvable."))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("corps = %v (%d clés), attendu exactement code/message/retryable", got, len(got))
	}
	for _, k := range []string{"code", "message", "retryable"} {
		if _, ok := got[k]; !ok {
			t.Errorf("clé %q absente du corps runtime", k)
		}
	}
}
