package revision

// embarques.go — LES FICHIERS QU UNE SOURCE EMBARQUE ENTRENT DANS L EMPREINTE (lot J3.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat SRC-1).
//
// `film/damagetag` embarque sa table d identifiants et ses libelles (`//go:embed data/...`) : ce
// sont des DONNEES que le binaire porte et qui decident la sortie de `killsource`. Hacher la
// directive sans le fichier laisserait une ligne de la table changer sans que rien ne bouge —
// exactement le trou que SRC-1 nomme (« `F/damagetag` (donnees + seuil `Strong`) »).
//
// # LES MOTIFS, COMME LE COMPILATEUR LES LIT (sous-ensemble)
//
// Chaque motif est relatif au dossier de la source, en slash, eventuellement entre guillemets et
// prefixe de `all:`. Un motif est un glob (`path.Match`) ; un dossier trouve s embarque en entier,
// sans les noms qui commencent par `.` ou `_` sauf sous `all:`. Un motif qui ne trouve RIEN est
// une ERREUR — le compilateur la ferait aussi, et hacher du vide rendrait un gate vert.
//
// Le contenu est pris OCTET POUR OCTET, fins de ligne normalisees en LF : un checkout Windows qui
// convertit les `.tsv` en CRLF ne doit pas rendre une autre empreinte que la CI.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// prefixeEmbarque distingue un fichier embarque d une source dans les noms haches : aucun chemin
// de source ne commence ainsi.
const prefixeEmbarque = "embed:"

// collecte : les fichiers retenus d UNE racine, et les fichiers embarques deja vus (un fichier
// designe par deux directives n entre qu une fois).
type collecte struct {
	racine  string
	prefixe string
	deja    map[string]bool
	lus     []sourceLue
}

// embarquer resout les motifs d une source et ajoute les fichiers trouves.
func (c *collecte) embarquer(source string, motifs []string) error {
	dossier := filepath.Dir(source)
	for _, brut := range motifs {
		motif, tout, err := motifEmbarque(brut)
		if err != nil {
			return fmt.Errorf("%s : motif //go:embed %q : %w", source, brut, err)
		}
		trouves, err := filepath.Glob(filepath.Join(dossier, filepath.FromSlash(motif)))
		if err != nil || len(trouves) == 0 {
			return fmt.Errorf("%s : motif //go:embed %q sans fichier (%v)", source, brut, err)
		}
		for _, t := range trouves {
			if err := c.ajouterEmbarques(t, tout); err != nil {
				return err
			}
		}
	}
	return nil
}

// motifEmbarque rend le motif sans guillemets et dit s il porte `all:`.
func motifEmbarque(brut string) (string, bool, error) {
	motif := brut
	if strings.HasPrefix(motif, `"`) || strings.HasPrefix(motif, "`") {
		m, err := strconv.Unquote(motif)
		if err != nil {
			return "", false, err
		}
		motif = m
	}
	reste, tout := strings.CutPrefix(motif, "all:")
	return reste, tout, nil
}

// ajouterEmbarques ajoute un fichier trouve, ou tout un dossier trouve.
func (c *collecte) ajouterEmbarques(trouve string, tout bool) error {
	return filepath.WalkDir(trouve, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		cache := chemin != trouve && !tout && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_"))
		switch {
		case d.IsDir() && cache:
			return filepath.SkipDir
		case d.IsDir() || cache:
			return nil
		}
		rel, err := filepath.Rel(c.racine, chemin)
		if err != nil {
			return err
		}
		nom := prefixeEmbarque + nomHache(c.prefixe, filepath.ToSlash(rel))
		if c.deja[nom] {
			return nil
		}
		c.deja[nom] = true
		blob, err := os.ReadFile(chemin) //nolint:gosec // fichier designe par une directive d une source du depot
		if err != nil {
			return err
		}
		c.lus = append(c.lus, sourceLue{rel: nom, texte: strings.ReplaceAll(string(blob), "\r\n", "\n")})
		return nil
	})
}
