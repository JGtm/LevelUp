// Command openapi-diff émet un RÉSUMÉ sémantique stable et diffable d'un document
// OpenAPI (paths + méthodes + schémas + propriétés/types/enums/defaults/examples/
// required/descriptions). But : figer une BASELINE « avant » du chantier V72-01
// (openapi.yaml généré par Huma) pour détecter toute perte sémantique à chaque
// étape (gate H0/H3/H8 du plan .ai/V7/PLAN_V72_HUMA_OPENAPI.md).
//
// Ce n'est PAS un binaire externe en CI : c'est un outil Go du repo, exécuté à la
// demande, dont la SORTIE (texte trié) est comparée par `diff`. Deux modes :
//   - défaut : émet le résumé sur stdout (ou -out fichier) ;
//   - -diff old.txt : compare la baseline `old.txt` au résumé courant et sort en
//     code 1 s'ils diffèrent (affiche les lignes ajoutées/retirées).
//
// Chargement via kin-openapi (openapi3). La validation stricte n'est PAS lancée
// (le document déclare 3.1.0 avec un corps de style 3.0 — nullable:, etc. — et on
// veut la STRUCTURE, pas un verdict de conformité).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	in := flag.String("in", "api/openapi.yaml", "chemin du document OpenAPI à résumer")
	out := flag.String("out", "", "fichier de sortie (défaut: stdout)")
	diff := flag.String("diff", "", "baseline à comparer; sortie code 1 si divergence")
	flag.Parse()

	summary, err := summarizeFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-diff: %v\n", err)
		os.Exit(2)
	}

	if *diff != "" {
		os.Exit(runDiff(*diff, summary))
	}

	if *out != "" {
		if err := os.WriteFile(*out, []byte(summary), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "openapi-diff: écriture %s: %v\n", *out, err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "openapi-diff: baseline écrite dans %s (%d octets)\n", *out, len(summary))
		return
	}
	fmt.Print(summary)
}

func summarizeFile(path string) (string, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return "", fmt.Errorf("chargement %s: %w", path, err)
	}
	return summarize(doc), nil
}

func summarize(doc *openapi3.T) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# OpenAPI semantic baseline\n")
	fmt.Fprintf(&b, "openapi: %s\n", doc.OpenAPI)
	if doc.Info != nil {
		fmt.Fprintf(&b, "info.title: %s\n", doc.Info.Title)
		fmt.Fprintf(&b, "info.version: %s\n", doc.Info.Version)
	}
	writePaths(&b, doc)
	writeSchemas(&b, doc)
	return b.String()
}

// ---------------------------------------------------------------------------
// Paths
// ---------------------------------------------------------------------------

func writePaths(b *strings.Builder, doc *openapi3.T) {
	fmt.Fprintf(b, "\n## PATHS\n")
	if doc.Paths == nil {
		fmt.Fprintf(b, "(aucun)\n")
		return
	}
	pathKeys := make([]string, 0, len(doc.Paths.Map()))
	for k := range doc.Paths.Map() {
		pathKeys = append(pathKeys, k)
	}
	sort.Strings(pathKeys)

	nOps := 0
	for _, p := range pathKeys {
		item := doc.Paths.Map()[p]
		methods := make([]string, 0)
		ops := item.Operations()
		for m := range ops {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		for _, m := range methods {
			nOps++
			writeOperation(b, m, p, ops[m])
		}
	}
	fmt.Fprintf(b, "# paths=%d operations=%d\n", len(pathKeys), nOps)
}

func writeOperation(b *strings.Builder, method, path string, op *openapi3.Operation) {
	fmt.Fprintf(b, "\n%s %s\n", method, path)
	if op.OperationID != "" {
		fmt.Fprintf(b, "  operationId: %s\n", op.OperationID)
	}
	if d := singleLine(op.Summary); d != "" {
		fmt.Fprintf(b, "  summary: %s\n", d)
	}
	if len(op.Tags) > 0 {
		tags := append([]string(nil), op.Tags...)
		sort.Strings(tags)
		fmt.Fprintf(b, "  tags: %s\n", strings.Join(tags, ","))
	}
	writeParams(b, op.Parameters)
	writeRequestBody(b, op.RequestBody)
	writeResponses(b, op.Responses)
}

func writeParams(b *strings.Builder, params openapi3.Parameters) {
	if len(params) == 0 {
		return
	}
	lines := make([]string, 0, len(params))
	for _, pr := range params {
		if pr == nil || pr.Value == nil {
			continue
		}
		p := pr.Value
		line := fmt.Sprintf("    param %s in=%s required=%t type=%s",
			p.Name, p.In, p.Required, schemaRefType(p.Schema))
		if extra := inlineScalarExtras(p.Schema); extra != "" {
			line += " " + extra
		}
		if d := singleLine(p.Description); d != "" {
			line += " desc=" + d
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(b, l)
	}
}

// inlineScalarExtras rend enum/default/example d'un schéma INLINE (ex. schéma de
// paramètre) — le $ref nommé est décrit ailleurs (section SCHEMAS).
func inlineScalarExtras(ref *openapi3.SchemaRef) string {
	if ref == nil || ref.Ref != "" || ref.Value == nil {
		return ""
	}
	s := ref.Value
	var parts []string
	if len(s.Enum) > 0 {
		parts = append(parts, "enum="+jsonValues(s.Enum))
	}
	if s.Default != nil {
		parts = append(parts, "default="+jsonValue(s.Default))
	}
	if s.Example != nil {
		parts = append(parts, "example="+jsonValue(s.Example))
	}
	return strings.Join(parts, " ")
}

func writeRequestBody(b *strings.Builder, rb *openapi3.RequestBodyRef) {
	if rb == nil || rb.Value == nil {
		return
	}
	fmt.Fprintf(b, "    requestBody required=%t\n", rb.Value.Required)
	writeContent(b, "      req", rb.Value.Content)
}

func writeResponses(b *strings.Builder, responses *openapi3.Responses) {
	if responses == nil {
		return
	}
	codes := make([]string, 0)
	for code := range responses.Map() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		rr := responses.Map()[code]
		if rr == nil || rr.Value == nil {
			fmt.Fprintf(b, "    response %s\n", code)
			continue
		}
		desc := ""
		if rr.Value.Description != nil {
			desc = singleLine(*rr.Value.Description)
		}
		if desc != "" {
			fmt.Fprintf(b, "    response %s desc=%s\n", code, desc)
		} else {
			fmt.Fprintf(b, "    response %s\n", code)
		}
		writeContent(b, "      resp", rr.Value.Content)
	}
}

// writeContent décrit le contenu (par content-type trié) : ref/type du schéma,
// exemple au niveau media-type, et expansion récursive des schémas INLINE (les
// $ref nommés sont décrits dans la section SCHEMAS).
func writeContent(b *strings.Builder, prefix string, content openapi3.Content) {
	if len(content) == 0 {
		return
	}
	cts := make([]string, 0, len(content))
	for ct := range content {
		cts = append(cts, ct)
	}
	sort.Strings(cts)
	for _, ct := range cts {
		mt := content[ct]
		ref := "-"
		if mt != nil {
			ref = schemaRefType(mt.Schema)
		}
		line := fmt.Sprintf("%s %s -> %s", prefix, ct, ref)
		if mt != nil && mt.Example != nil {
			line += " example=" + jsonValue(mt.Example)
		}
		fmt.Fprintln(b, line)
		if mt != nil && mt.Schema != nil && mt.Schema.Ref == "" {
			walkSchema(b, prefix+"  ", "", mt.Schema)
		}
	}
}

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

func writeSchemas(b *strings.Builder, doc *openapi3.T) {
	fmt.Fprintf(b, "\n## SCHEMAS\n")
	if doc.Components == nil || doc.Components.Schemas == nil {
		fmt.Fprintf(b, "(aucun)\n")
		return
	}
	names := make([]string, 0, len(doc.Components.Schemas))
	for n := range doc.Components.Schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		writeSchema(b, n, doc.Components.Schemas[n])
	}
	fmt.Fprintf(b, "# schemas=%d\n", len(names))
}

func writeSchema(b *strings.Builder, name string, ref *openapi3.SchemaRef) {
	fmt.Fprintf(b, "\nschema %s\n", name)
	walkSchema(b, "  ", "", ref)
}

// walkSchema décrit récursivement un schéma inline en s'arrêtant à chaque $ref
// (émis comme "$ref:Nom", jamais expansé → pas de récursion infinie sur les
// cycles, qui ne passent que par des refs nommés). `path` est le chemin pointé
// depuis la racine du schéma (ex. "meta.items[]"). Capture à CHAQUE profondeur :
// type/format/enum/default/example/description/required.
func walkSchema(b *strings.Builder, indent, path string, ref *openapi3.SchemaRef) {
	if ref == nil {
		return
	}
	label := path
	if label == "" {
		label = "(root)"
	}
	if ref.Ref != "" {
		fmt.Fprintf(b, "%snode %s: $ref:%s\n", indent, label, refName(ref.Ref))
		return
	}
	s := ref.Value
	if s == nil {
		return
	}
	fmt.Fprintf(b, "%snode %s: %s\n", indent, label, scalarSummary(s))
	if len(s.Required) > 0 {
		req := append([]string(nil), s.Required...)
		sort.Strings(req)
		fmt.Fprintf(b, "%s  required: %s\n", indent, strings.Join(req, ","))
	}
	walkChildren(b, indent, path, s)
}

func walkChildren(b *strings.Builder, indent, path string, s *openapi3.Schema) {
	names := make([]string, 0, len(s.Properties))
	for n := range s.Properties {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		walkSchema(b, indent, joinPath(path, n), s.Properties[n])
	}
	if s.Items != nil {
		walkSchema(b, indent, path+"[]", s.Items)
	}
	if s.AdditionalProperties.Schema != nil {
		walkSchema(b, indent, path+"{}", s.AdditionalProperties.Schema)
	}
	walkComposition(b, indent, path, "allOf", s.AllOf)
	walkComposition(b, indent, path, "anyOf", s.AnyOf)
	walkComposition(b, indent, path, "oneOf", s.OneOf)
}

func walkComposition(b *strings.Builder, indent, path, kind string, refs openapi3.SchemaRefs) {
	for i, r := range refs {
		walkSchema(b, indent, fmt.Sprintf("%s|%s[%d]", path, kind, i), r)
	}
}

// scalarSummary rend les attributs scalaires d'un schéma inline sur une ligne.
func scalarSummary(s *openapi3.Schema) string {
	parts := []string{"type=" + typesStr(s.Type)}
	if s.Format != "" {
		parts = append(parts, "format="+s.Format)
	}
	if len(s.Enum) > 0 {
		parts = append(parts, "enum="+jsonValues(s.Enum))
	}
	if s.Default != nil {
		parts = append(parts, "default="+jsonValue(s.Default))
	}
	if s.Example != nil {
		parts = append(parts, "example="+jsonValue(s.Example))
	}
	if d := singleLine(s.Description); d != "" {
		parts = append(parts, "desc="+d)
	}
	return strings.Join(parts, " ")
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// ---------------------------------------------------------------------------
// Helpers stables
// ---------------------------------------------------------------------------

func schemaRefType(ref *openapi3.SchemaRef) string {
	if ref == nil {
		return "-"
	}
	if ref.Ref != "" {
		return "$ref:" + refName(ref.Ref)
	}
	if ref.Value == nil {
		return "?"
	}
	if ref.Value.Items != nil {
		it := ref.Value.Items
		if it.Ref != "" {
			return "array<$ref:" + refName(it.Ref) + ">"
		}
		if it.Value != nil {
			return "array<" + typesStr(it.Value.Type) + ">"
		}
	}
	return typesStr(ref.Value.Type)
}

func typesStr(t *openapi3.Types) string {
	if t == nil {
		return "-"
	}
	s := append([]string(nil), t.Slice()...)
	sort.Strings(s)
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, "|")
}

func refName(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

func singleLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func jsonValue(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func jsonValues(vs []any) string {
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		parts = append(parts, jsonValue(v))
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, ",") + "]"
}

// ---------------------------------------------------------------------------
// Diff
// ---------------------------------------------------------------------------

func runDiff(baselinePath, current string) int {
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-diff: lecture baseline %s: %v\n", baselinePath, err)
		return 2
	}
	if string(data) == current {
		fmt.Fprintf(os.Stderr, "openapi-diff: identique à la baseline %s\n", baselinePath)
		return 0
	}
	added, removed := lineDiff(string(data), current)
	fmt.Fprintf(os.Stderr, "openapi-diff: DIVERGENCE vs %s (+%d / -%d lignes)\n", baselinePath, len(added), len(removed))
	for _, l := range removed {
		fmt.Println("- " + l)
	}
	for _, l := range added {
		fmt.Println("+ " + l)
	}
	return 1
}

// lineDiff renvoie (ajoutées dans cur, retirées vs base) par comptage multiset.
func lineDiff(base, cur string) (added, removed []string) {
	baseCount := countLines(base)
	curCount := countLines(cur)
	for l, c := range curCount {
		if c > baseCount[l] {
			for i := 0; i < c-baseCount[l]; i++ {
				added = append(added, l)
			}
		}
	}
	for l, c := range baseCount {
		if c > curCount[l] {
			for i := 0; i < c-curCount[l]; i++ {
				removed = append(removed, l)
			}
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func countLines(s string) map[string]int {
	m := map[string]int{}
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		m[sc.Text()]++
	}
	return m
}
