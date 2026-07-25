package openapigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"gopkg.in/yaml.v3"
)

// APIV1Prefix — préfixe de montage chi des routes métier. Le document Huma porte
// les chemins ABSOLUS (H1) ; le contrat publié reste RELATIF au server `/api/v1`
// (convention historique du yaml et de generated.ts) → le préfixe est retiré des
// clés de `paths` à la sérialisation.
const APIV1Prefix = "/api/v1"

// GeneratedHeader — bandeau du fichier généré (fait partie du golden).
const GeneratedHeader = `# FICHIER GÉNÉRÉ — NE PAS ÉDITER À LA MAIN.
#
# Source : document OpenAPI PARTAGÉ de Huma (routes + schémas dérivés des types Go)
# fusionné avec le fragment manuel versionné api/openapi_manual_fragment.yaml
# (tout ce qui n'est PAS dérivable : routes chi-brut, corps RawBody, sémantique
# d'opération, descriptions de schéma racine, schémas sans type Go).
#
# Régénérer :  make openapi-gen
# Verrouillé par : TestOpenAPIYAMLIsUpToDate (internal/api) — golden byte-à-byte.
`

// Generate assemble le routeur de démo, fusionne le fragment manuel et retourne
// le YAML du contrat. UN seul chemin de code pour `make openapi-gen` et pour le
// golden `TestOpenAPIYAMLIsUpToDate`.
//
// `root` est un répertoire de travail ISOLÉ (t.TempDir / os.MkdirTemp) ;
// `fragmentPath` le fragment manuel versionné.
func Generate(ctx context.Context, root, fragmentPath string) ([]byte, error) {
	fragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		return nil, fmt.Errorf("openapigen: lecture fragment %s: %w", fragmentPath, err)
	}
	_, doc, err := BuildDemoRouter(ctx, Options{
		Root:           root,
		GroupStorePath: filepath.Join(root, "groups.json"),
		// Surface MAXIMALE : mêmes flags que les garde-rails de fidélité H1/H2.
		MultiTitleAPI: true,
		Prestige:      true,
	})
	if err != nil {
		return nil, err
	}
	return Render(doc, fragment)
}

// Render sérialise le document Huma en YAML DÉTERMINISTE après trois passes :
//  1. les chemins absolus perdent le préfixe /api/v1 (contrat relatif au server) ;
//  2. les requestBody `application/octet-stream` auto-générés pour les handlers
//     RawBody sont retirés (wrapper par-route SANS valeur : Huma ne voit qu'un
//     []byte opaque — le vrai schéma JSON vient du fragment) ;
//  3. le fragment manuel est fusionné en profondeur, avec PRÉCÉDENCE MANUELLE.
func Render(doc *huma.OpenAPI, fragment []byte) ([]byte, error) {
	tree, err := docToTree(doc)
	if err != nil {
		return nil, err
	}
	if err := stripAPIV1Prefix(tree); err != nil {
		return nil, err
	}
	dropOpaqueBodyStubs(tree)

	if len(bytes.TrimSpace(fragment)) > 0 {
		var frag map[string]any
		if err := yaml.Unmarshal(fragment, &frag); err != nil {
			return nil, fmt.Errorf("openapigen: parse fragment: %w", err)
		}
		merged, ok := mergeNode(tree, frag, "").(map[string]any)
		if !ok {
			return nil, fmt.Errorf("openapigen: fusion du fragment: racine non-objet")
		}
		tree = merged
	}
	return emitYAML(tree)
}

// ---------------------------------------------------------------------------
// Document Huma → arbre générique
// ---------------------------------------------------------------------------

// docToTree passe par le JSON de Huma (source de vérité de la sérialisation
// OpenAPI) puis normalise les nombres : json.Number → int64 quand la valeur est
// entière (sinon yaml.v3 émettrait `1e+06` pour un float64).
func docToTree(doc *huma.OpenAPI) (map[string]any, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("openapigen: marshal document: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree map[string]any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("openapigen: décodage document: %w", err)
	}
	out, ok := normalizeNumbers(tree).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("openapigen: document non-objet")
	}
	return out, nil
}

func normalizeNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = normalizeNumbers(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = normalizeNumbers(t[i])
		}
		return t
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		f, err := t.Float64()
		if err != nil {
			return t.String()
		}
		return f
	default:
		return v
	}
}

// ---------------------------------------------------------------------------
// Passe 1 — chemins relatifs au server /api/v1
// ---------------------------------------------------------------------------

func stripAPIV1Prefix(tree map[string]any) error {
	paths, ok := tree["paths"].(map[string]any)
	if !ok {
		return fmt.Errorf("openapigen: document sans section paths")
	}
	out := make(map[string]any, len(paths))
	keys := sortedKeys(paths)
	for _, p := range keys {
		rel := p
		if strings.HasPrefix(p, APIV1Prefix+"/") {
			rel = strings.TrimPrefix(p, APIV1Prefix)
		}
		if _, dup := out[rel]; dup {
			return fmt.Errorf("openapigen: collision de chemin après retrait de %s : %s", APIV1Prefix, rel)
		}
		out[rel] = paths[p]
	}
	tree["paths"] = out
	return nil
}

// ---------------------------------------------------------------------------
// Passe 2 — retrait des wrappers de corps opaques auto-générés
// ---------------------------------------------------------------------------

// dropOpaqueBodyStubs neutralise les deux wrappers par-route que Huma fabrique
// autour d'un `[]byte` — ils décrivent le TRANSPORT, pas le contrat :
//
//   - requestBody `application/octet-stream` d'un champ `RawBody []byte` (le
//     handler décode du JSON à la main pour préserver le 400 {invalid_body}) →
//     SUPPRIMÉ ; le fragment restitue le schéma JSON des routes documentées ;
//   - schéma de réponse `{type: string, contentEncoding: base64}` d'un
//     `Body []byte` (passe-plat JSON pour un ETag byte-exact) → réduit au schéma
//     VIDE `{}` (forme non dérivable), que le fragment complète.
//
// Les schémas NOMMÉS et référencés ne sont jamais touchés.
func dropOpaqueBodyStubs(tree map[string]any) {
	paths, _ := tree["paths"].(map[string]any)
	for _, pathItem := range paths {
		item, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}
		for _, opAny := range item {
			op, ok := opAny.(map[string]any)
			if !ok {
				continue
			}
			if isRawBodyStub(op["requestBody"]) {
				delete(op, "requestBody")
			}
			neutralizeBinaryResponses(op)
		}
	}
}

// neutralizeBinaryResponses remplace les schémas de réponse binaires opaques par
// un schéma vide (voir dropOpaqueBodyStubs).
func neutralizeBinaryResponses(op map[string]any) {
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return
	}
	for _, respAny := range responses {
		resp, ok := respAny.(map[string]any)
		if !ok {
			continue
		}
		content, ok := resp["content"].(map[string]any)
		if !ok {
			continue
		}
		for _, mediaAny := range content {
			media, ok := mediaAny.(map[string]any)
			if !ok {
				continue
			}
			if schema, ok := media["schema"].(map[string]any); ok && isOpaqueBinarySchema(schema) {
				media["schema"] = map[string]any{}
			}
		}
	}
}

// isOpaqueBinarySchema : schéma d'un `[]byte` Go (string base64 / binaire) SANS
// autre information — aucune valeur de contrat.
func isOpaqueBinarySchema(schema map[string]any) bool {
	if schema["type"] != "string" {
		return false
	}
	if schema["contentEncoding"] != "base64" && schema["format"] != "binary" {
		return false
	}
	for k := range schema {
		switch k {
		case "type", "contentEncoding", "format", "contentMediaType":
		default:
			return false
		}
	}
	return true
}

func isRawBodyStub(v any) bool {
	rb, ok := v.(map[string]any)
	if !ok {
		return false
	}
	content, ok := rb["content"].(map[string]any)
	if !ok || len(content) != 1 {
		return false
	}
	media, ok := content["application/octet-stream"].(map[string]any)
	if !ok {
		return false
	}
	schema, ok := media["schema"].(map[string]any)
	if !ok {
		return false
	}
	return schema["type"] == "string" && schema["format"] == "binary"
}

// ---------------------------------------------------------------------------
// Passe 3 — fusion du fragment manuel (PRÉCÉDENCE MANUELLE)
// ---------------------------------------------------------------------------

// mergeNode fusionne `overlay` (fragment manuel) dans `base` (document Huma),
// façon JSON Merge Patch. Règles, dans l'ordre :
//   - deux objets → fusion RÉCURSIVE clé à clé (le fragment n'écrase que ce
//     qu'il déclare : il enrichit sans effacer le dérivé) ; une valeur `null`
//     SUPPRIME la clé dérivée (seul moyen de retirer un nœud faux — ex. le 200
//     que Huma documente quand le handler pose 201/202 via son champ Status) ;
//   - deux listes `parameters` d'opération → fusion PAR (name, in) : le fragment
//     enrichit le paramètre dérivé (description/enum/default) et peut en ajouter,
//     sans jamais réordonner ni supprimer les paramètres dérivés ;
//   - tout le reste (scalaire, liste, types incompatibles) → le fragment GAGNE.
func mergeNode(base, overlay any, path string) any {
	bMap, bOK := base.(map[string]any)
	oMap, oOK := overlay.(map[string]any)
	if bOK && oOK {
		for _, k := range sortedKeys(oMap) {
			if oMap[k] == nil {
				delete(bMap, k)
				continue
			}
			bMap[k] = mergeNode(bMap[k], oMap[k], path+"/"+k)
		}
		return bMap
	}
	bList, bOK := base.([]any)
	oList, oOK := overlay.([]any)
	if bOK && oOK && strings.HasPrefix(path, "/paths/") && strings.HasSuffix(path, "/parameters") {
		return mergeParameters(bList, oList, path)
	}
	return overlay
}

// mergeParameters fusionne deux listes de paramètres par clé (name, in).
func mergeParameters(base, overlay []any, path string) []any {
	index := map[string]int{}
	for i, p := range base {
		if k := parameterKey(p); k != "" {
			index[k] = i
		}
	}
	for _, p := range overlay {
		k := parameterKey(p)
		if i, ok := index[k]; ok && k != "" {
			base[i] = mergeNode(base[i], p, path+"/"+k)
			continue
		}
		base = append(base, p)
	}
	return base
}

func parameterKey(p any) string {
	m, ok := p.(map[string]any)
	if !ok {
		return ""
	}
	name, _ := m["name"].(string)
	in, _ := m["in"].(string)
	if name == "" || in == "" {
		return ""
	}
	return in + ":" + name
}

// ---------------------------------------------------------------------------
// Sérialisation YAML déterministe
// ---------------------------------------------------------------------------

// topLevelOrder — ordre de tête stable (lisibilité). Toute clé supplémentaire est
// émise ensuite par ordre alphabétique (aucune perte silencieuse).
var topLevelOrder = []string{"openapi", "info", "servers", "tags", "paths", "components"}

func emitYAML(tree map[string]any) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}
	emitted := map[string]bool{}
	appendKey := func(k string) error {
		v, ok := tree[k]
		if !ok {
			return nil
		}
		emitted[k] = true
		value := &yaml.Node{}
		if err := value.Encode(v); err != nil {
			return fmt.Errorf("openapigen: encodage %q: %w", k, err)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k}, value)
		return nil
	}
	for _, k := range topLevelOrder {
		if err := appendKey(k); err != nil {
			return nil, err
		}
	}
	for _, k := range sortedKeys(tree) {
		if emitted[k] {
			continue
		}
		if err := appendKey(k); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	buf.WriteString(GeneratedHeader)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("openapigen: encodage YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("openapigen: fermeture encodeur YAML: %w", err)
	}
	return buf.Bytes(), nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
