// Package archlint — camps_canonical_single_source_test.go : garde-rail de « mon camp / l'autre
// camp » d'une ligne canonique (CLAUDE.md règle 6, plan Tactique v2 lot L13 F6).
//
// La lecture tient en deux gestes — retrouver les équipes 0 et 1 dans `Summary.Teams`, échanger
// quand `Self.TeamID` vaut 1 — et sa SOURCE UNIQUE est `analysis.CampsDuMatch`
// (`internal/analysis/camps_canonical.go`), lue par le score de l'accueil et des Sessions
// (`ScoreLabelCanonical`) et par le détail d'une zone de l'onglet Tactique (`scoreDuMatch`). Deux
// copies existaient ; une troisième aurait pu prendre le camp de l'adversaire pour le sien.
//
// Les empreintes d'une copie (lignes de commentaire ignorées) :
//  1. l'adressage d'une équipe par son indice dans la ligne (`Summary.Teams[i].TeamID`) ;
//  2. l'échange sur le camp du joueur (`Self.TeamID != nil && *r.Self.TeamID == 1`).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// campsHelper : la source unique.
const campsHelper = "internal/analysis/camps_canonical.go"

var (
	equipeParIndice = regexp.MustCompile(`Summary\.Teams\[\w+\]\.TeamID`)
	echangeDuCamp   = regexp.MustCompile(`Self\.TeamID\s*!=\s*nil\s*&&\s*\*\s*[\w.]*Self\.TeamID\s*==`)
)

// empreintesDeCamps rend les empreintes d'une lecture locale des camps d'une ligne canonique.
func empreintesDeCamps(source string) []string {
	source = sansCommentaires(source)
	var out []string
	if equipeParIndice.MatchString(source) {
		out = append(out, "équipe adressée par son indice dans la ligne canonique")
	}
	if echangeDuCamp.MatchString(source) {
		out = append(out, "échange sur le camp du joueur écrit à la main")
	}
	return out
}

func TestCampsCanonical_ReconnaitLesCopies(t *testing.T) {
	for _, c := range []string{
		// service.scoreDuMatch avant L13
		"\tfor i := range r.Summary.Teams {\n\t\tswitch r.Summary.Teams[i].TeamID {",
		// analysis.ScoreLabelCanonical avant L13
		"\tif r.Self.TeamID != nil && *r.Self.TeamID == 1 {\n\t\tmine, theirs = t1, t0\n\t}",
	} {
		if len(empreintesDeCamps(c)) == 0 {
			t.Errorf("le garde-rail ne reconnaît pas une copie :\n%s", c)
		}
	}
	for _, sain := range []string{
		"\tif r.Self.TeamID != nil {\n\t\tout.TeamID = *r.Self.TeamID\n\t}",
		"\t\tTeamID:             r.Self.TeamID,",
		"\tmien, autre, ok := analysis.CampsDuMatch(r)",
	} {
		if e := empreintesDeCamps(sain); len(e) != 0 {
			t.Errorf("faux positif sur %q : %v", sain, e)
		}
	}
}

func TestCampsCanonical_SourceUnique(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == campsHelper {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, e := range empreintesDeCamps(string(data)) {
				violations = append(violations, rel+" : "+e)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("lecture locale des camps d'une ligne canonique interdite — appeler analysis.CampsDuMatch (%s) :\n  %s",
			campsHelper, strings.Join(violations, "\n  "))
	}
}
