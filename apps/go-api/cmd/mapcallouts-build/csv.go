package main

// csv.go — les LIBELLÉS officiels des zones, depuis la copie versionnée de
// callouts_i18n.csv (data/titles/{slug}/reference/).
//
// LE CSV EST LA SOURCE FIGÉE DES LIBELLÉS : il résout les 816 string_id du corpus vers
// le libellé joueur EN et FR (extraction uslg faite UNE fois par la recherche — on ne
// re-extrait pas uslg, règle du plan parité lot 3). Une ligne par named location,
// indexée (carte, volumeIndex) ; le string_id de la ligne se VÉRIFIE contre celui du
// tag — une divergence est un CSV périmé, jamais une zone à publier quand même.

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

type libelle struct {
	en, fr   string
	stringID uint32
}

// libelles indexe les lignes du CSV par (carte, volumeIndex).
type libelles map[string]map[int]libelle

// Colonnes attendues du CSV (en-tête vérifié : un CSV réordonné doit échouer).
var colonnesAttendues = []string{"carte", "volumeIndex", "string_id", "nom_conception",
	"en", "fr", "source", "tronque", "nom_brut", "a_forme", "x", "y", "z"}

// chargeLibelles lit le CSV versionné (séparateur « ; », BOM UTF-8 toléré).
func chargeLibelles(path string) (libelles, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("CSV des libellés illisible (%s) : %w", path, err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV des libellés invalide (%s) : %w", path, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV des libellés vide (%s)", path)
	}
	head := rows[0]
	if len(head) > 0 {
		head[0] = strings.TrimPrefix(head[0], "\ufeff")
	}
	if len(head) != len(colonnesAttendues) {
		return nil, fmt.Errorf("CSV : %d colonnes, attendu %d", len(head), len(colonnesAttendues))
	}
	for i, c := range colonnesAttendues {
		if head[i] != c {
			return nil, fmt.Errorf("CSV : colonne %d = %q, attendu %q", i, head[i], c)
		}
	}
	out := libelles{}
	for n, row := range rows[1:] {
		carte := row[0]
		vi, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, fmt.Errorf("CSV ligne %d : volumeIndex %q : %w", n+2, row[1], err)
		}
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[2], "0x"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("CSV ligne %d : string_id %q : %w", n+2, row[2], err)
		}
		if row[4] == "" || row[5] == "" {
			return nil, fmt.Errorf("CSV ligne %d (%s vi=%d) : libellé vide — le contrat est 816/816 résolus", n+2, carte, vi)
		}
		if out[carte] == nil {
			out[carte] = map[int]libelle{}
		}
		out[carte][vi] = libelle{en: row[4], fr: row[5], stringID: uint32(sid)}
	}
	return out, nil
}

// resolve rend le libellé d'une zone et vérifie la cohérence CSV <-> tag.
func (l libelles) resolve(module string, c himap.Callout) (libelle, error) {
	lbl, ok := l[module][c.VolumeIndex]
	if !ok {
		return libelle{}, fmt.Errorf("%s vi=%d (%s) : aucune ligne CSV — le contrat est 816/816", module, c.VolumeIndex, c.Name)
	}
	if lbl.stringID != c.StringID {
		return libelle{}, fmt.Errorf("%s vi=%d : string_id CSV %08x != tag %08x (CSV périmé ?)",
			module, c.VolumeIndex, lbl.stringID, c.StringID)
	}
	return lbl, nil
}
