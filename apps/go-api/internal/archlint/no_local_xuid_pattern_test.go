// Package archlint — no_local_xuid_pattern_test.go : ratchet « une seule definition du
// motif XUID » (revue ronde 2 de la phase 4 bis, 2026-09-06).
//
// POURQUOI. Le motif d'un XUID — un entier decimal borne — gardait DEUX frontieres sous
// DEUX definitions, dans deux paquets : le parametre `with_player` de l'Explorateur
// (`handlers.parseNeighborsFilterSpec`) et la composition de l'onglet Tactique. Rien
// n'empechait la prochaine main d'en changer une seule : deux verdicts pour « est-ce un
// XUID ? », dont l'un borne un cout de requete (un `EXISTS` correle par coequipier).
//
// La source unique est `domain.XUIDValide` (domain/tactical.go). Ce test interdit toute
// autre compilation du motif dans `internal/`, y compris ecrite autrement
// (`[0-9]{1,32}` au lieu de la classe abregee) — un ratchet qui ne verrouille qu'une
// orthographe se contourne sans le vouloir.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// motifBorneDecimale reconnait la FIN d'un motif « entier decimal borne, ancre » :
// la classe (abregee ou explicite) suivie de `{1,<n>}$`.
//
// Ecrit avec des classes a un caractere (`[{]`, `[}]`, `[$]`) plutot qu'avec des
// echappements : ce fichier decrit des motifs, et empiler les antislashs pour parler
// d'antislashs rend la regle illisible — et fragile a la premiere recopie.
var motifBorneDecimale = regexp.MustCompile(`(d|9])[{]1,[0-9]+[}][$]`)

func compileUnMotifXUID(line string) bool {
	return strings.Contains(line, "regexp.MustCompile") && motifBorneDecimale.MatchString(line)
}

func TestNoLocalXUIDPattern(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	source := filepath.Join(internalRoot, "domain", "tactical.go")

	// SELF-CHECK, ET IL EST LE PLUS IMPORTANT : le predicat doit reconnaitre la SOURCE
	// UNIQUE elle-meme. Sans lui, un motif de detection legerement faux rendrait ce
	// ratchet vert pour toujours — en ne detectant plus rien du tout, y compris ce
	// qu'il est cense interdire (defaut mesure ici meme, ronde 2).
	src, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("lecture de la source unique: %v", err)
	}
	trouve := false
	for _, line := range strings.Split(string(src), "\n") {
		if compileUnMotifXUID(line) {
			trouve = true
			break
		}
	}
	if !trouve {
		t.Fatal("le predicat ne reconnait PAS la definition de domain/tactical.go — " +
			"le ratchet ne verifie rien (a-t-elle change de forme ?)")
	}

	var violations []string
	domainRoot := filepath.Join(internalRoot, "domain")
	err = filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// La SOURCE UNIQUE est exemptée — c'est elle que les autres doivent appeler.
		if strings.HasPrefix(path, domainRoot+string(filepath.Separator)) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			if compileUnMotifXUID(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("motif XUID recompilé hors de internal/domain — appeler domain.XUIDValide "+
			"(source unique, cf. domain/tactical.go) :\n  %s", strings.Join(violations, "\n  "))
	}
}
