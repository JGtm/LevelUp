//go:build cgo

// public_routes_cost_class_test.go — classe de coût des routes publiques (plan sécurité du
// 2026-10-15, lot L5, C3-7).
//
// POURQUOI. bare_routes_ratchet_test.go prouve qu'une route nue est justifiée, pas ce
// qu'un anonyme peut lui faire faire : /auth/device-flow/start lançait un quart d'heure
// d'appels vers Microsoft, /assets/* des appels signés vers les API Halo. Chaque entrée de
// publicRoutesAllowlist porte donc une classe (cheap / outbound / disk) ; outbound et disk
// NOMMENT le test qui prouve qu'un anonyme n'en tire ni appel sortant ni écriture non
// bornée, et ce test doit exister dans le module.
package api_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var testFuncRe = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)

// moduleTestNames rend l'ensemble des noms de fonctions Test... du module.
func moduleTestNames(t *testing.T) map[string]bool {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // apps/go-api
	names := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range testFuncRe.FindAllSubmatch(src, -1) {
			names[string(m[1])] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours des tests : %v", err)
	}
	if len(names) < 1000 {
		t.Fatalf("%d tests recensés : le parcours ne couvre pas le module", len(names))
	}
	return names
}

// publicRouteProblems rend les écarts de l'allowlist à la règle : classe connue, preuve
// nommée et existante pour outbound/disk, aucune preuve pour cheap, raison non vide.
func publicRouteProblems(routes map[string]publicRoute, tests map[string]bool) []string {
	var out []string
	for key, r := range routes {
		switch r.class {
		case costCheap:
			if r.proof != "" {
				out = append(out, key+" : une route cheap ne nomme pas de preuve ("+r.proof+")")
			}
		case costOutbound, costDisk:
			if r.proof == "" {
				out = append(out, key+" : route "+string(r.class)+" sans test de preuve nommé")
			} else if !tests[r.proof] {
				out = append(out, key+" : le test de preuve "+r.proof+" n'existe pas")
			}
		default:
			out = append(out, key+" : classe de coût inconnue ou absente ("+string(r.class)+")")
		}
		if strings.TrimSpace(r.reason) == "" {
			out = append(out, key+" : justification vide")
		}
	}
	sort.Strings(out)
	return out
}

func TestPublicRoutesCarryACostClass(t *testing.T) {
	if problems := publicRouteProblems(publicRoutesAllowlist, moduleTestNames(t)); len(problems) > 0 {
		t.Errorf("allowlist des routes publiques : %d écart(s) à la règle de classe de coût :\n  %s",
			len(problems), strings.Join(problems, "\n  "))
	}
	classes := map[costClass]int{}
	for _, r := range publicRoutesAllowlist {
		classes[r.class]++
	}
	if classes[costOutbound] == 0 || classes[costDisk] == 0 {
		t.Errorf("classes recensées %v : l'allowlist doit encore classer des routes outbound et disk", classes)
	}
}

// TestPublicRouteGuardDetectsDefects : le contrôle sait échouer sur chaque forme visée.
func TestPublicRouteGuardDetectsDefects(t *testing.T) {
	tests := map[string]bool{"TestExiste": true}
	bad := map[string]publicRoute{
		"GET /sans-classe":          {reason: "x"},
		"GET /outbound-sans-preuve": outbound("x", ""),
		"GET /disk-preuve-fantome":  disk("x", "TestNexistePas"),
		"GET /cheap-avec-preuve":    {class: costCheap, reason: "x", proof: "TestExiste"},
		"GET /sans-raison":          cheap(""),
	}
	if got := publicRouteProblems(bad, tests); len(got) != len(bad) {
		t.Fatalf("%d écart(s) relevé(s) pour %d défauts : %v", len(got), len(bad), got)
	}
	good := map[string]publicRoute{
		"GET /a": cheap("lecture"),
		"GET /b": outbound("appel", "TestExiste"),
		"GET /c": disk("écriture", "TestExiste"),
	}
	if got := publicRouteProblems(good, tests); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}
}
