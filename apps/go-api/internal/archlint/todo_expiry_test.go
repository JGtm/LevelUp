// Package archlint — todo_expiry_test.go : garde-fou de la dette datée.
//
// Convention (DETTE reco 7) : toute dette technique assumée sous forme de TODO
// dont on connaît l'échéance porte le marqueur `TODO(expiry:YYYY-MM-DD)` suivi
// d'une justification. Ce test scanne les sources Go de apps/go-api et ÉCHOUE dès
// qu'un tel marqueur a dépassé sa date (ou porte une date malformée) — la dette
// datée devient alors une action à traiter, pas un commentaire oublié.
//
// Les TODO/FIXME nus (sans expiry) restent tolérés : la convention est opt-in, le
// lint ne force pas la conversion des 500+ TODO existants. Il rend seulement la
// dette DATÉE tenable dans le temps.
//
// `now` est injectable via LEVELUP_TODO_EXPIRY_NOW=YYYY-MM-DD (déterminisme CI /
// test du lint) ; à défaut, l'heure murale UTC. Sans build tag — CI normale.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// todoExpiryRE capture la date d'un marqueur TODO(expiry:YYYY-MM-DD).
var todoExpiryRE = regexp.MustCompile(`TODO\(expiry:(\d{4})-(\d{2})-(\d{2})\)`)

// scannerBasename : ce fichier contient le motif littéral et s'auto-exclut.
const scannerBasename = "todo_expiry_test.go"

// expiryLayout est le format de date des marqueurs et de LEVELUP_TODO_EXPIRY_NOW.
const expiryLayout = "2006-01-02"

func TestNoExpiredTODO(t *testing.T) {
	now := todoExpiryNow(t)

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// .../internal/archlint/todo_expiry_test.go → remonter à la racine go-api.
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	err := filepath.WalkDir(goAPIRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || filepath.Base(path) == scannerBasename {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(goAPIRoot, path)
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			m := todoExpiryRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			exp, parseErr := time.Parse(expiryLayout, m[1]+"-"+m[2]+"-"+m[3])
			loc := rel + ":" + strconv.Itoa(i+1)
			if parseErr != nil {
				violations = append(violations, loc+"  date malformée: "+m[0])
				continue
			}
			if exp.Before(now) {
				violations = append(violations, loc+"  échu le "+m[1]+"-"+m[2]+"-"+m[3]+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk go-api: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("%d TODO(expiry:...) échu(s)/malformé(s) — traiter la dette ou repousser l'échéance "+
			"avec justification :\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}

// todoExpiryNow résout l'instant de référence : LEVELUP_TODO_EXPIRY_NOW si défini
// (format YYYY-MM-DD), sinon l'heure murale UTC du jour.
func todoExpiryNow(t *testing.T) time.Time {
	if v := strings.TrimSpace(os.Getenv("LEVELUP_TODO_EXPIRY_NOW")); v != "" {
		parsed, err := time.Parse(expiryLayout, v)
		if err != nil {
			t.Fatalf("LEVELUP_TODO_EXPIRY_NOW invalide (%q, attendu YYYY-MM-DD): %v", v, err)
		}
		return parsed
	}
	n := time.Now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}
