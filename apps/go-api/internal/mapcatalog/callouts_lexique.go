package mapcatalog

// callouts_lexique.go — LE LEXIQUE DES NOMS DE LIEU : string_id -> libellé joueur EN/FR.
//
// Une zone Forge ne porte que le StringId de son lieu : ce fichier plat
// (`reference/callouts_lexique.csv`, PathResolver.MapCalloutsLexiquePath) est la SEULE
// jointure vers le texte que le jeu affiche. Il est produit hors serveur par
// `mapcallouts-build --lexique` depuis les listes de chaînes du jeu installé et VERSIONNÉ ;
// sa LECTURE, elle, n'exige rien — c'est ce qui permet au serveur de nommer les zones d'une
// carte rattrapée sans le jeu.
//
// IL COUVRE LE CSV FIGÉ DES CARTES INTÉGRÉES au caractère près (garde-rail
// `cmd/mapcallouts-build/lexique_test.go`, TestLexiqueCouvreLeCSVFigeAuCaractereRes) : la
// fusion « CSV puis lexique » de la chaîne de fabrication et la lecture du lexique seul au
// runtime donnent donc les mêmes libellés.

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Libelle est le texte joueur d'un nom de lieu, dans les deux langues du dépôt.
type Libelle struct {
	EN, FR string
}

// Lexique indexe les libellés par string_id de lieu.
type Lexique map[uint32]Libelle

// ColonnesLexique : l'en-tête attendu du fichier (un fichier réordonné doit échouer). Partagé
// par le lecteur et par l'écrivain de la chaîne de fabrication.
var ColonnesLexique = []string{"string_id", "en", "fr"}

// bomUTF8 : la marque d'ordre des octets qu'un tableur ajoute en tête d'un CSV.
const bomUTF8 = string(rune(0xFEFF))

// ChargerLexique lit le lexique. Une entrée sans texte dans l'une des deux langues, ou un
// string_id porteur de deux libellés, fait échouer la lecture : un faux nom de zone est pire
// qu'un nom absent, et une moitié de couple nommerait la zone dans une seule langue.
func ChargerLexique(path string) (Lexique, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("lexique invalide (%s) : %w", path, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("lexique vide (%s)", path)
	}
	if err := verifierEnteteLexique(rows[0]); err != nil {
		return nil, err
	}
	out := Lexique{}
	for n, row := range rows[1:] {
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[0], "0x"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("lexique ligne %d : string_id %q : %w", n+2, row[0], err)
		}
		if row[1] == "" || row[2] == "" {
			return nil, fmt.Errorf("lexique ligne %d (%s) : libellé vide", n+2, row[0])
		}
		l := Libelle{EN: row[1], FR: row[2]}
		if vu, deja := out[uint32(sid)]; deja && vu != l {
			return nil, fmt.Errorf("lexique ligne %d : string_id %08x porte deux libellés", n+2, sid)
		}
		out[uint32(sid)] = l
	}
	return out, nil
}

// verifierEnteteLexique contrôle l'en-tête, BOM toléré.
func verifierEnteteLexique(head []string) error {
	if len(head) > 0 {
		head[0] = strings.TrimPrefix(head[0], bomUTF8)
	}
	if len(head) != len(ColonnesLexique) {
		return fmt.Errorf("lexique : %d colonnes, attendu %d", len(head), len(ColonnesLexique))
	}
	for i, c := range ColonnesLexique {
		if head[i] != c {
			return fmt.Errorf("lexique : colonne %d = %q, attendu %q", i, head[i], c)
		}
	}
	return nil
}
