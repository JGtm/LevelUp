package anonymisegamertags

// registre_test.go — LE REGISTRE DES IDENTIFIANTS FICTIFS ET SON GARDE-RAIL.
//
// `libs/film/testdata/IDENTITES.tsv` liste les xuids fictifs qui tiennent lieu d identifiants de
// joueurs dans les fixtures du decodeur. Le garde-rail tourne a chaque `go test` : un xuid de la
// plage des xuids qui figure dans un fichier TEXTE de la brique sans etre au registre est un
// identifiant que personne n a rendu fictif — le plus souvent un xuid reel recopie d un film ou de
// la base dans un test.
//
// Il ne voit pas les octets des films : pour eux, c est la lecture du decodeur (table des joueurs,
// fil des morts) qui l etablit, a la regeneration d une fixture (cf. doc.go).

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/identitefictive"
)

// cheminRegistre : le registre, depuis la racine du depot.
var cheminRegistre = filepath.Join("libs", "film", "testdata", "IDENTITES.tsv")

// extensionsTexte : les fichiers que le garde-rail lit (les films, binaires, n en font pas partie).
var extensionsTexte = map[string]bool{
	".go": true, ".json": true, ".golden": true, ".txt": true, ".tsv": true, ".md": true,
}

var suiteDeChiffres = regexp.MustCompile(`[0-9]+`)

// lireRegistre rend les xuids du registre, et refuse un registre incoherent : une ligne mal
// formee, un xuid hors plage ou en double, un registre vide.
func lireRegistre(chemin string) (map[uint64]bool, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin fixe, issu de la racine du depot
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[uint64]bool{}
	sc := bufio.NewScanner(f)
	for ligne := 1; sc.Scan(); ligne++ {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		champs := strings.Split(l, "\t")
		if len(champs) != 2 {
			return nil, fmt.Errorf("ligne %d : attendu decimal<TAB>hexadecimal, lu %q", ligne, l)
		}
		v, err := strconv.ParseUint(champs[0], 10, 64)
		if err != nil || champs[1] != "0x"+hexa(v, false) {
			return nil, fmt.Errorf("ligne %d : %q et %q ne decrivent pas le meme xuid", ligne, champs[0], champs[1])
		}
		if !identitefictive.DansLaPlage(v) {
			return nil, fmt.Errorf("ligne %d : %d hors de la plage des xuids", ligne, v)
		}
		if out[v] {
			return nil, fmt.Errorf("ligne %d : %d en double", ligne, v)
		}
		out[v] = true
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("registre vide")
	}
	return out, nil
}

// xuidsHorsRegistre parcourt les fichiers texte sous `racine` et rend, par fichier, les xuids de la
// plage que le registre ne porte pas.
func xuidsHorsRegistre(racine string, registre map[uint64]bool) (map[string][]string, error) {
	hors := map[string][]string{}
	err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !extensionsTexte[filepath.Ext(p)] {
			return err
		}
		brut, err := os.ReadFile(p) //nolint:gosec // chemin issu du parcours
		if err != nil {
			return err
		}
		for _, m := range suiteDeChiffres.FindAll(brut, -1) {
			if len(m) != 16 {
				continue
			}
			v, err := strconv.ParseUint(string(m), 10, 64)
			if err == nil && identitefictive.DansLaPlage(v) && !registre[v] {
				hors[filepath.ToSlash(p)] = append(hors[filepath.ToSlash(p)], string(m))
			}
		}
		return nil
	})
	return hors, err
}

// TestLesXuidsDesFixturesSontFictifs : aucun xuid de la plage, hors registre, dans les fichiers
// texte de `libs/film`.
func TestLesXuidsDesFixturesSontFictifs(t *testing.T) {
	registre, err := lireRegistre(filepath.Join(racineDuDepot, cheminRegistre))
	if err != nil {
		t.Fatalf("registre des identifiants fictifs : %v", err)
	}
	hors, err := xuidsHorsRegistre(filepath.Join(racineDuDepot, "libs", "film"), registre)
	if err != nil {
		t.Fatal(err)
	}
	if len(hors) == 0 {
		return
	}
	fichiers := make([]string, 0, len(hors))
	for f := range hors {
		fichiers = append(fichiers, f)
	}
	sort.Strings(fichiers)
	var sb strings.Builder
	for _, f := range fichiers {
		fmt.Fprintf(&sb, "\n  %s : %s", f, strings.Join(hors[f], ", "))
	}
	t.Fatalf("%d fichier(s) portent un xuid hors de %s : un identifiant reel, ou un fictif a ajouter au "+
		"registre (utiliser de preference un xuid qui y figure deja)%s", len(hors), filepath.ToSlash(cheminRegistre), sb.String())
}

// TestLeGardeRailVoitUnXuidHorsRegistre : sur un dossier synthetique, un xuid absent du registre
// est signale, un xuid du registre et un nombre qui n est pas dans la plage ne le sont pas, et un
// nombre colle a d autres chiffres n est pas un xuid.
func TestLeGardeRailVoitUnXuidHorsRegistre(t *testing.T) {
	dossier := t.TempDir()
	const dedans, dehors = "2550913680076637", "2533274790395904"
	contenu := "fictif " + dedans + "\nreel " + dehors + "\nautre 1234567890123456\ncolle 1" + dehors + "9\n"
	if err := os.WriteFile(filepath.Join(dossier, "x.go"), []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dossier, "x.bin"), []byte(dehors), 0o600); err != nil {
		t.Fatal(err)
	}
	v, _ := strconv.ParseUint(dedans, 10, 64)
	hors, err := xuidsHorsRegistre(dossier, map[uint64]bool{v: true})
	if err != nil || len(hors) != 1 {
		t.Fatalf("hors registre : %v (%v)", hors, err)
	}
	for _, l := range hors {
		if len(l) != 1 || l[0] != dehors {
			t.Fatalf("attendu le seul %s, lu %v", dehors, l)
		}
	}
}
