package anonymisegamertags

// anonymiser_test.go — LA PORTE : inventaire, ou reecriture sous GAMERTAGS_ECRIRE=1.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// racineDuDepot : ce paquet vit sous apps/go-api/cmd/anonymise-gamertags.
var racineDuDepot = filepath.Join("..", "..", "..", "..")

// bilanFichier : ce qu un fichier contenait.
type bilanFichier struct {
	chemin string
	forme  forme
	compte map[string]int
}

func (b bilanFichier) total() int {
	n := 0
	for _, v := range b.compte {
		n += v
	}
	return n
}

// TestAnonymiserFixtures : la porte. Sans table, elle ne fait rien (CI comprise).
func TestAnonymiserFixtures(t *testing.T) {
	cheminTable := os.Getenv("GAMERTAGS_CORRESPONDANCE")
	if cheminTable == "" {
		t.Skip("GAMERTAGS_CORRESPONDANCE non defini : porte d anonymisation des fixtures fermee")
	}
	f, err := os.Open(cheminTable) //nolint:gosec // chemin fourni par l operateur
	if err != nil {
		t.Fatalf("table de correspondance : %v", err)
	}
	ps, err := lireCorrespondance(f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("table de correspondance : %v", err)
	}
	ecrire := os.Getenv("GAMERTAGS_ECRIRE") == "1"
	fichiers, err := fichiersDesChemins(os.Getenv("GAMERTAGS_CHEMINS"))
	if err != nil {
		t.Fatal(err)
	}
	var bilans []bilanFichier
	for _, chemin := range fichiers {
		b, err := traiterFichier(chemin, ps, ecrire)
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		if b.total() > 0 {
			bilans = append(bilans, b)
		}
	}
	t.Fatalf("%s", rendreBilan(bilans, len(fichiers), len(ps), ecrire))
}

// fichiersDesChemins developpe les chemins (`;`), relatifs a la racine du depot ou absolus.
func fichiersDesChemins(liste string) ([]string, error) {
	var out []string
	for _, c := range strings.Split(liste, ";") {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !filepath.IsAbs(c) {
			c = filepath.Join(racineDuDepot, c)
		}
		err := filepath.WalkDir(c, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				out = append(out, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("parcours de %s : %w", c, err)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("GAMERTAGS_CHEMINS ne designe aucun fichier")
	}
	sort.Strings(out)
	return out, nil
}

// traiterFichier inventorie un fichier, et le reecrit si demande et s il change.
func traiterFichier(chemin string, ps []paire, ecrire bool) (bilanFichier, error) {
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours demande
	if err != nil {
		return bilanFichier{}, err
	}
	sortie, compte, frm, err := anonymiserOctets(brut, ps)
	if err != nil {
		return bilanFichier{}, err
	}
	b := bilanFichier{chemin: chemin, forme: frm, compte: compte}
	if ecrire && b.total() > 0 {
		info, err := os.Stat(chemin)
		if err != nil {
			return bilanFichier{}, err
		}
		if err := os.WriteFile(chemin, sortie, info.Mode().Perm()); err != nil {
			return bilanFichier{}, err
		}
	}
	return b, nil
}

// rendreBilan : une ligne par fichier touche, puis le total.
func rendreBilan(bilans []bilanFichier, nFichiers, nPaires int, ecrire bool) string {
	var sb strings.Builder
	verbe := "INVENTAIRE (rien ecrit ; GAMERTAGS_ECRIRE=1 pour reecrire)"
	if ecrire {
		verbe = "REECRITURE"
	}
	noms := map[string]bool{}
	total := 0
	for _, b := range bilans {
		fmt.Fprintf(&sb, "\n  %-5s %6d occurrence(s), %3d nom(s)  %s", b.forme, b.total(), len(b.compte),
			filepath.ToSlash(b.chemin))
		for n := range b.compte {
			noms[n] = true
		}
		total += b.total()
	}
	return fmt.Sprintf("%s : %d fichier(s) parcourus, %d contiennent des originaux ; %d occurrence(s) ; "+
		"%d original(aux) distinct(s) sur %d dans la table%s", verbe, nFichiers, len(bilans), total,
		len(noms), nPaires, sb.String())
}
