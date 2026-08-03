//go:build cgo

// openapi_schema_semantics_test.go — garde-rail H3 (chantier V72-01).
//
// H3 porte la sémantique de champ du contrat manuel (api/openapi.yaml) dans des
// tags struct Go (doc / enum / default / example) pour que le document généré par
// Huma (H6) soit au moins aussi riche. Deux garde-rails complémentaires :
//
//   - TestPortedDTOSchemaSemantics (Partie A, INDÉPENDANT DU ROUTAGE) : régénère
//     le schéma de CHAQUE type instrumenté via un registre Huma neuf et vérifie
//     que les tags produisent bien la sémantique attendue. Couvre aussi les types
//     dont la route n'est pas montée par le routeur de démo.
//   - TestSharedDocSchemaSemanticsVsYAML (Partie B, ce que demande le plan) :
//     compare la sémantique du document PARTAGÉ en mémoire au yaml manuel, champ
//     par champ, pour chaque schéma COMMUN. Toute perte hors allowlist (=
//     inventaire fermé des sémantiques NON portables en tags) échoue. Les
//     compteurs figent l'inventaire : un nouvel oubli OU une entrée d'allowlist
//     devenue caduque font échouer le test.
//
// Inventaire fermé des sémantiques NON portables (destination = fragment manuel
// H6, sauf mention) — cf. .ai/V7/PLAN_V72_HUMA_OPENAPI.md, section H3 :
//   - descriptions de SCHÉMA RACINE (Huma n'a pas de tag type-level ; seul
//     SchemaProvider/SchemaTransformer, hors périmètre tag) : cf. rootDescAllowlist.
//   - schémas SANS type Go huma-généré (chi-brut / décodage manuel RawBody /
//     champ Go scalaire) : FreshnessInfo (Go *string), SortSpec,
//     MatchHistoryExportRequest, PlotlyFigurePayload (legacy Python),
//     BackupStatusResponse/BackupRunResult/IntegrityCheckResult (backup chi-brut).
//     (ApiErrorSchema : RÉSOLU en H4 — fusionné dans le type Go huma-généré
//     ApiError, désormais riche : code/message/retryable/details/field_errors.)
//   - schémas d'ENTRÉE décodés à la main (var req domain.X) donc jamais générés
//     par Huma : CompareRequest, MatchHistoryQueryRequest, PaginationRequest,
//     MediaPageRequest, SessionContextRequest, ExplorerMatchesQueryRequest.
//   - dérives de NOM yaml↔Go (ExplorerMatchRow↔ExplorerMatchesRow,
//     GamertagSuggestion↔GamertagSearchResult, CascadeInput↔CascadeFilter,
//     CareerTopMatch sans équivalent DTO) : la sémantique portable a été posée
//     sur le type Go réel ; le nom yaml diverge (H6/H7).
//   - defaults triviaux (valeur zéro : bool false / int 0) NON posés (no-op).
//   - sémantique de PARAMÈTRE (enum/default de query/path) et exemples de RÉPONSE
//     (5 exemples d'erreur, désormais sur l'ApiError unifié H4, portés au niveau
//     components.responses) : hors schéma.champ → restent au fragment manuel H6.
//     L'exemple de CHAMP code=player_not_found est, lui, porté en tag Huma (H4).
//   - 16 request bodies RawBody : → fragment manuel.

package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Partie A — vérification directe des tags (indépendante du routage)
// ---------------------------------------------------------------------------

// portedField = une sémantique attendue sur un champ d'un type instrumenté.
// kind: "desc" (présence non vide) | "enum" | "default" (égalité de valeur JSON).
type portedField struct {
	path string
	kind string
	val  string // JSON normalisé attendu (ignoré pour kind=desc)
}

func portedDTOExpectations() []struct {
	name  string
	typ   reflect.Type
	items []portedField
} {
	return []struct {
		name  string
		typ   reflect.Type
		items []portedField
	}{
		{"BootstrapResponse", reflect.TypeOf(domain.BootstrapResponse{}), []portedField{
			{"auth_state", "enum", `["missing","partial","ready"]`},
			{"setup_state", "enum", `["no_halo_link","halo_linked_no_profile","profile_ready_no_sync","ready"]`},
			{"auth_mode", "enum", `["none","password","xbox"]`},
			{"registration_mode", "enum", `["invite","open","closed"]`},
			{"locale", "default", `"fr"`},
			{"hints_visible_default", "default", `true`},
			{"available_titles", "desc", ""},
			{"current_title_slug", "desc", ""},
			{"privacy", "desc", ""},
		}},
		{"TitleSummary", reflect.TypeOf(domain.TitleSummary{}), []portedField{
			{"status", "enum", `["active","coming_soon","archived"]`},
			{"provides_damage_taken", "desc", ""},
			{"provides_team_mmr", "desc", ""},
			{"provides_max_killing_spree", "desc", ""},
		}},
		{"DeviceFlowStartResponse", reflect.TypeOf(domain.DeviceFlowStartResponse{}), []portedField{
			{"user_code", "desc", ""},
			{"expires_in", "desc", ""},
			{"poll_interval_seconds", "default", `5`},
		}},
		{"DeviceFlowStatusResponse", reflect.TypeOf(domain.DeviceFlowStatusResponse{}), []portedField{
			{"status", "enum", `["pending","authorized","provisioned","failed","expired"]`},
		}},
		{"EngagementScoreResult", reflect.TypeOf(domain.EngagementScoreResult{}), []portedField{
			{"calibration", "desc", ""},
			{"expected_basis", "desc", ""},
			{"intensity_bin", "desc", ""},
			{"signal_basis", "desc", ""},
		}},
		{"PlayerMediaAudioConfig", reflect.TypeOf(domain.PlayerMediaAudioConfig{}), []portedField{
			{"mode", "enum", `["auto","manual"]`},
			{"mode", "desc", ""},
			{"track_roles", "desc", ""},
			{"track_roles[]", "enum", `["game","voice","other"]`},
			{"updated_at", "desc", ""},
		}},
		{"SettingsResponse", reflect.TypeOf(domain.SettingsResponse{}), []portedField{
			{"lang", "enum", `["fr","en"]`},
			{"discord_lang", "enum", `["fr","en"]`},
			{"media_delete_source_after_transcode", "desc", ""},
			{"show_progression", "desc", ""},
		}},
		{"AdminResourcesResponse", reflect.TypeOf(domain.AdminResourcesResponse{}), []portedField{
			{"db_inventory_status", "enum", `["ok","unavailable"]`},
			{"db_inventory_status", "desc", ""},
		}},
		{"ExplorerMatchesRow", reflect.TypeOf(domain.ExplorerMatchesRow{}), []portedField{
			{"experience_type_label", "default", `"Non classé"`},
			{"had_bot_teammate", "desc", ""},
		}},
		{"GamertagSearchResult", reflect.TypeOf(domain.GamertagSearchResult{}), []portedField{
			{"score", "default", `1`},
		}},
	}
}

func TestPortedDTOSchemaSemantics(t *testing.T) {
	var checked int
	for _, tc := range portedDTOExpectations() {
		reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
		s := reg.Schema(tc.typ, false, tc.name)
		if s == nil {
			t.Fatalf("%s : schéma nil", tc.name)
		}
		for _, it := range tc.items {
			checked++
			fs := fieldSchema(s, it.path)
			if fs == nil {
				t.Errorf("%s.%s : champ absent du schéma généré", tc.name, it.path)
				continue
			}
			switch it.kind {
			case "desc":
				if strings.TrimSpace(fs.Description) == "" {
					t.Errorf("%s.%s : description vide (tag doc: manquant ?)", tc.name, it.path)
				}
			case "enum":
				if got := jsonNorm(fs.Enum); got != it.val {
					t.Errorf("%s.%s : enum=%s, attendu=%s", tc.name, it.path, got, it.val)
				}
			case "default":
				if got := jsonNorm(fs.Default); got != it.val {
					t.Errorf("%s.%s : default=%s, attendu=%s", tc.name, it.path, got, it.val)
				}
			}
		}
	}
	t.Logf("H3 Partie A : %d sémantiques de champ vérifiées sur %d types",
		checked, len(portedDTOExpectations()))
}

// fieldSchema navigue un chemin pointé (avec suffixe "[]" pour les items de
// tableau) dans un schéma Huma déjà généré.
func fieldSchema(s *huma.Schema, path string) *huma.Schema {
	cur := s
	for _, seg := range strings.Split(path, ".") {
		item := false
		if strings.HasSuffix(seg, "[]") {
			item = true
			seg = strings.TrimSuffix(seg, "[]")
		}
		if cur == nil || cur.Properties == nil {
			return nil
		}
		cur = cur.Properties[seg]
		if item {
			if cur == nil {
				return nil
			}
			cur = cur.Items
		}
	}
	return cur
}

// ---------------------------------------------------------------------------
// Partie B — document partagé en mémoire vs yaml manuel (schéma.champ commun)
// ---------------------------------------------------------------------------

// H6 : l'allowlist statique des sémantiques non portables a été REMPLACÉE par le
// fragment manuel (api/openapi_manual_fragment.yaml). Invariant vérifié ici :
// toute sémantique du contrat que le document Huma ne porte PAS doit être
// DÉCLARÉE dans le fragment — et réciproquement, une déclaration de fragment
// devenue inutile (le tag Go la produit désormais) est signalée CADUQUE.

type semNode struct {
	desc    string
	enum    string // JSON normalisé ("" si absent)
	def     string
	example string
}

func TestSharedDocSchemaSemanticsVsYAML(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	yamlSem := loadYAMLSchemaSem(t)
	if len(yamlSem) < 100 {
		t.Fatalf("yaml : %d schémas chargés (attendu >= 100)", len(yamlSem))
	}
	fragSem := loadFragmentSchemaSem(t)
	if len(fragSem) == 0 {
		t.Fatal("fragment manuel : aucune sémantique de schéma chargée — lecture cassée ?")
	}

	_, doc := buildTestRouterWithDoc(t)
	if doc == nil || doc.Components == nil {
		t.Fatal("document partagé nil")
	}
	docMap := doc.Components.Schemas.Map()

	var common, checked, matched int
	losses := map[string]bool{}
	for name, docSchema := range docMap {
		ysem, ok := yamlSem[name]
		if !ok {
			continue
		}
		common++
		dsem := map[string]semNode{}
		walkHumaSem(docSchema, "", dsem, doc)
		for path, y := range ysem {
			for _, kind := range []string{"desc", "enum", "default", "example"} {
				want := pick(y, kind)
				if want == "" && kind != "desc" {
					continue
				}
				if kind == "desc" && y.desc == "" {
					continue
				}
				checked++
				d := dsem[path]
				got := pick(d, kind)
				ok := false
				if kind == "desc" {
					ok = strings.TrimSpace(d.desc) != ""
				} else {
					ok = got != "" && got == want
				}
				if ok {
					matched++
					continue
				}
				key := name + "|" + path + "|" + kind
				losses[key] = true
				if !fragSem[key] {
					t.Errorf("PERTE hors fragment : %s (yaml %s=%q, doc=%q) — porter la sémantique en tag Go "+
						"ou la déclarer dans api/openapi_manual_fragment.yaml", key, kind, want, got)
				}
			}
		}
	}

	// Anti-caducité : toute sémantique déclarée par le fragment SUR UN SCHÉMA
	// DÉRIVÉ doit correspondre à une perte réelle (sinon le tag Go la produit
	// déjà → la retirer du fragment). Les schémas purement manuels (aucun type Go,
	// absents du document) ne sont pas concernés.
	for key := range fragSem {
		name := key[:strings.Index(key, "|")]
		if _, derived := docMap[name]; !derived {
			continue
		}
		if !losses[key] {
			t.Errorf("déclaration de fragment CADUQUE (le document Huma porte déjà la sémantique) : %s — la retirer", key)
		}
	}

	if common < 8 {
		t.Fatalf("schémas communs doc∩yaml : %d (attendu >= 8) — mapping cassé ?", common)
	}
	t.Logf("H3 Partie B : %d schémas communs, %d sémantiques vérifiées, %d en parité, %d pertes (fragment=%d)",
		common, checked, matched, len(losses), len(fragSem))
}

// loadFragmentSchemaSem indexe les sémantiques (desc/enum/default/example) que le
// FRAGMENT MANUEL déclare sous components.schemas, avec la même convention de
// clé que le walker du document : "Schema|chemin|kind".
func loadFragmentSchemaSem(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi_manual_fragment.yaml"))
	if err != nil {
		t.Fatalf("lecture du fragment manuel : %v", err)
	}
	var frag struct {
		Components struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &frag); err != nil {
		t.Fatalf("parse du fragment manuel : %v", err)
	}
	out := map[string]bool{}
	for name, schema := range frag.Components.Schemas {
		walkFragmentSem(schema, name, "", out)
	}
	return out
}

// walkFragmentSem parcourt un schéma du fragment (arbre YAML générique).
func walkFragmentSem(node any, name, path string, out map[string]bool) {
	m, ok := node.(map[string]any)
	if !ok {
		return
	}
	label := path
	if label == "" {
		label = "(root)"
	}
	for key, kind := range map[string]string{"description": "desc", "enum": "enum", "default": "default", "example": "example"} {
		if _, has := m[key]; has {
			out[name+"|"+label+"|"+kind] = true
		}
	}
	if props, ok := m["properties"].(map[string]any); ok {
		for prop, sub := range props {
			walkFragmentSem(sub, name, join(path, prop), out)
		}
	}
	if items, ok := m["items"]; ok {
		walkFragmentSem(items, name, path+"[]", out)
	}
	if add, ok := m["additionalProperties"]; ok {
		walkFragmentSem(add, name, path+"{}", out)
	}
}

func pick(n semNode, kind string) string {
	switch kind {
	case "desc":
		return n.desc
	case "enum":
		return n.enum
	case "default":
		return n.def
	case "example":
		return n.example
	}
	return ""
}

// ---------------------------------------------------------------------------
// Walkers — même convention de chemin des deux côtés
// ---------------------------------------------------------------------------

func loadYAMLSchemaSem(t *testing.T) map[string]map[string]semNode {
	t.Helper()
	path := filepath.Join("..", "..", "api", "openapi.yaml")
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("chargement %s: %v", path, err)
	}
	out := map[string]map[string]semNode{}
	for name, ref := range doc.Components.Schemas {
		m := map[string]semNode{}
		walkYAMLSem(ref, "", m)
		out[name] = m
	}
	return out
}

func walkYAMLSem(ref *openapi3.SchemaRef, path string, out map[string]semNode) {
	if ref == nil || ref.Ref != "" || ref.Value == nil {
		return
	}
	s := ref.Value
	label := path
	if label == "" {
		label = "(root)"
	}
	n := out[label]
	if d := strings.TrimSpace(s.Description); d != "" {
		n.desc = d
	}
	if len(s.Enum) > 0 {
		n.enum = jsonNorm(s.Enum)
	}
	if s.Default != nil {
		n.def = jsonNorm(s.Default)
	}
	if s.Example != nil {
		n.example = jsonNorm(s.Example)
	}
	out[label] = n
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		walkYAMLSem(s.Properties[k], join(path, k), out)
	}
	if s.Items != nil {
		walkYAMLSem(s.Items, path+"[]", out)
	}
	if s.AdditionalProperties.Schema != nil {
		walkYAMLSem(s.AdditionalProperties.Schema, path+"{}", out)
	}
}

func walkHumaSem(s *huma.Schema, path string, out map[string]semNode, doc *huma.OpenAPI) {
	if s == nil {
		return
	}
	label := path
	if label == "" {
		label = "(root)"
	}
	n := out[label]
	if d := strings.TrimSpace(s.Description); d != "" {
		n.desc = d
	}
	if len(s.Enum) > 0 {
		n.enum = jsonNorm(s.Enum)
	}
	if s.Default != nil {
		n.def = jsonNorm(s.Default)
	}
	if len(s.Examples) > 0 {
		// Huma 3.1 : examples[] ; le yaml manuel 3.0 : example unique. On compare
		// le premier exemple (aligne les deux formes).
		n.example = jsonNorm(s.Examples[0])
	}
	out[label] = n
	if s.Ref != "" {
		// $ref : Huma peut poser une sémantique SIBLING (description/nullable en
		// 3.1) sur le nœud tout en pointant un schéma nommé — on l'a captée
		// ci-dessus ; on ne déréférence pas (pas de propriétés inline).
		return
	}
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		walkHumaSem(s.Properties[k], join(path, k), out, doc)
	}
	if s.Items != nil {
		walkHumaSem(s.Items, path+"[]", out, doc)
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// jsonNorm rend une valeur (ou une liste d'enum) sous forme JSON stable pour
// comparaison. Les enums conservent l'ordre du contrat.
func jsonNorm(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
