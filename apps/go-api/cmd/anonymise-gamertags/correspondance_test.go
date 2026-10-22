package anonymisegamertags

// correspondance_test.go — LA TABLE original -> pseudonyme, et ses refus.

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
)

// paire : un gamertag reel et son pseudonyme, de meme longueur.
type paire struct {
	origine    string
	pseudonyme string
}

// lireCorrespondance lit la table et la valide. L ordre rendu place les originaux les plus LONGS
// d abord : un original qui en contiendrait un autre est remplace avant lui.
func lireCorrespondance(r io.Reader) ([]paire, error) {
	var out []paire
	sc := bufio.NewScanner(r)
	ligne := 0
	for sc.Scan() {
		ligne++
		l := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}
		champs := strings.Split(l, "\t")
		if len(champs) != 2 {
			return nil, fmt.Errorf("ligne %d : attendu original<TAB>pseudonyme, lu %q", ligne, l)
		}
		out = append(out, paire{origine: champs[0], pseudonyme: champs[1]})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("lecture de la table : %w", err)
	}
	if err := validerCorrespondance(out); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].origine) > len(out[j].origine) })
	return out, nil
}

// validerCorrespondance refuse ce qui decalerait un flux ou rendrait la substitution ambigue.
func validerCorrespondance(ps []paire) error {
	if len(ps) == 0 {
		return fmt.Errorf("table vide")
	}
	origines := map[string]bool{}
	pseudos := map[string]bool{}
	for _, p := range ps {
		if len(p.origine) < 3 {
			return fmt.Errorf("%q : un original de moins de 3 caracteres se confondrait avec du bruit", p.origine)
		}
		if !imprimableASCII(p.origine) || !imprimableASCII(p.pseudonyme) {
			return fmt.Errorf("%q -> %q : ASCII imprimable seulement (sinon UTF-8 et UTF-16 n ont plus "+
				"la meme longueur)", p.origine, p.pseudonyme)
		}
		if len(p.origine) != len(p.pseudonyme) ||
			len(utf16.Encode([]rune(p.origine))) != len(utf16.Encode([]rune(p.pseudonyme))) {
			return fmt.Errorf("%q -> %q : longueurs differentes", p.origine, p.pseudonyme)
		}
		if origines[p.origine] {
			return fmt.Errorf("%q : original en double", p.origine)
		}
		if pseudos[p.pseudonyme] {
			return fmt.Errorf("%q : pseudonyme en double", p.pseudonyme)
		}
		origines[p.origine], pseudos[p.pseudonyme] = true, true
	}
	for _, p := range ps {
		if origines[p.pseudonyme] {
			return fmt.Errorf("%q : pseudonyme qui est aussi un original", p.pseudonyme)
		}
	}
	return nil
}

func imprimableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return s != ""
}
