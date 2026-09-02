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

// libellesParStringID indexe les MÊMES lignes par string_id.
//
// POURQUOI CE SECOND INDEX. Une carte Forge n'a ni module ni indice de volume : son
// map.mvar ne porte que le StringId du lieu. C'est donc la SEULE clé de jointure possible
// vers le libellé joueur — et elle est légitime, le vocabulaire est le même (mesure du
// 2026-08-27 : 439 des 463 chaînes du CSV figurent au tag global `locs`, celui que les
// variantes référencent). Le CSV est cohérent sur cette clé, et `chargeLibelles` REFUSE
// un CSV où deux lignes se contrediraient.
type libellesParStringID map[uint32]libelle

// Colonnes attendues du CSV (en-tête vérifié : un CSV réordonné doit échouer).
var colonnesAttendues = []string{"carte", "volumeIndex", "string_id", "nom_conception",
	"en", "fr", "source", "tronque", "nom_brut", "a_forme", "x", "y", "z"}

// chargeLibelles lit le CSV versionné (séparateur « ; », BOM UTF-8 toléré) et rend les
// DEUX index : par (carte, volumeIndex) pour les cartes intégrées, par string_id pour les
// cartes Forge.
func chargeLibelles(path string) (libelles, libellesParStringID, error) {
	rows, err := litCSVLibelles(path)
	if err != nil {
		return nil, nil, err
	}
	out := libelles{}
	parSID := libellesParStringID{}
	for n, row := range rows[1:] {
		carte := row[0]
		vi, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, nil, fmt.Errorf("CSV ligne %d : volumeIndex %q : %w", n+2, row[1], err)
		}
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[2], "0x"), 16, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("CSV ligne %d : string_id %q : %w", n+2, row[2], err)
		}
		if row[4] == "" || row[5] == "" {
			return nil, nil, fmt.Errorf("CSV ligne %d (%s vi=%d) : libellé vide — le contrat est 816/816 résolus", n+2, carte, vi)
		}
		l := libelle{en: row[4], fr: row[5], stringID: uint32(sid)}
		if out[carte] == nil {
			out[carte] = map[int]libelle{}
		}
		out[carte][vi] = l
		// Un string_id qui porterait DEUX libellés différents rendrait la jointure Forge
		// indéterminée : on refuse plutôt que de trancher au hasard de l'ordre des lignes.
		// Mesuré sain sur le CSV versionné : 463 string_id, un seul couple (en, fr) chacun.
		if vu, deja := parSID[l.stringID]; deja && (vu.en != l.en || vu.fr != l.fr) {
			return nil, nil, fmt.Errorf("CSV ligne %d : string_id %08x porte deux libellés (%q/%q puis %q/%q)",
				n+2, l.stringID, vu.en, vu.fr, l.en, l.fr)
		}
		parSID[l.stringID] = l
	}
	return out, parSID, nil
}

// litCSVLibelles lit le fichier et VÉRIFIE son en-tête (un CSV réordonné doit échouer).
func litCSVLibelles(path string) ([][]string, error) {
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
	return rows, nil
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
