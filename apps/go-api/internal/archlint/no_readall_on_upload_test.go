// Package archlint — no_readall_on_upload_test.go : garde-rail « un envoi se copie en flux »
// (plan sécurité du 2026-10-15, lot L5, C3-5).
//
// POURQUOI. io.ReadAll sur un fichier de formulaire multipart porte le fichier ENTIER en
// mémoire : quatre envois de 500 Mo en parallèle suffisaient à saturer le serveur. Le
// contenu d'un envoi se copie par io.Copy vers un fichier (domain.UploadedFile.Open +
// service.storeUploadedFile).
//
// INTERDIT dans le code du module (hors _test.go) : un fichier qui manipule un formulaire
// multipart (import mime/multipart, ou FormFile / MultipartForm / MultipartReader) et appelle
// io.ReadAll, ioutil.ReadAll ou bytes.Buffer.ReadFrom. Aucune exception : lire un champ de
// formulaire de texte passe par FormValue.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// multipartMarkers — identifiants qui désignent la manipulation d'un formulaire multipart.
var multipartMarkers = map[string]bool{"FormFile": true, "MultipartForm": true, "MultipartReader": true}

// readAllInMultipartFile rend les sites ReadAll d'un fichier qui manipule du multipart.
func readAllInMultipartFile(src goSource) []string {
	usesMultipart := false
	for _, imp := range src.File.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err == nil && path == "mime/multipart" {
			usesMultipart = true
		}
	}
	var readAlls []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if multipartMarkers[x.Sel.Name] {
				usesMultipart = true
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && sel.Sel.Name == "ReadAll" && (pkg.Name == "io" || pkg.Name == "ioutil") {
				readAlls = append(readAlls, src.Rel+":"+strconv.Itoa(src.Fset.Position(x.Pos()).Line))
			}
			if sel.Sel.Name == "ReadFrom" {
				readAlls = append(readAlls, src.Rel+":"+strconv.Itoa(src.Fset.Position(x.Pos()).Line))
			}
		}
		return true
	})
	if !usesMultipart {
		return nil
	}
	return readAlls
}

func TestNoReadAllOnMultipartUploads(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, "internal", "cmd") {
		violations = append(violations, readAllInMultipartFile(src)...)
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("io.ReadAll / ReadFrom dans un fichier qui manipule un formulaire multipart (fichier entier en "+
			"mémoire) : copier en flux (io.Copy vers un fichier) :\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestReadAllGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "internal/api/handlers/x.go", `package x
import ("io"; "net/http")
func f(r *http.Request) { fh := r.MultipartForm.File["files"][0]; f, _ := fh.Open(); data, _ := io.ReadAll(f); _ = data }`)
	if got := readAllInMultipartFile(bad); len(got) != 1 {
		t.Fatalf("ReadAll sur un fichier multipart non signalé : %v", got)
	}
	viaImport := parseGoSnippet(t, "internal/api/handlers/y.go", `package y
import ("io"; "mime/multipart")
func f(f multipart.File) { _, _ = io.ReadAll(f) }`)
	if got := readAllInMultipartFile(viaImport); len(got) != 1 {
		t.Fatalf("ReadAll avec import multipart non signalé : %v", got)
	}
	clean := parseGoSnippet(t, "internal/api/handlers/z.go", `package z
import ("io"; "os")
func f(r io.Reader) { b, _ := io.ReadAll(r); _ = b; _ = os.Stdout }`)
	if got := readAllInMultipartFile(clean); len(got) != 0 {
		t.Fatalf("faux positif hors multipart : %v", got)
	}
}
