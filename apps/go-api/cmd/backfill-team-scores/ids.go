package main

// ids.go — CHARGEMENT DE LA LISTE DE TRAVAIL.
//
// LE FICHIER N'EST QU'UNE LISTE. Le TSV livré par la phase 1 porte aussi les valeurs
// mesurées le 2026-08-24 (`db_t0`, `api_t0`, …) : elles ne sont JAMAIS lues ici, et c'est
// délibéré. Une valeur d'API vieille de plusieurs semaines n'a pas à devenir une écriture ;
// seule la colonne `match_id` sort de ce fichier, et le score est re-téléchargé à
// l'exécution. Si l'API a encore bougé entre-temps, l'outil le verra.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// matchIDColumn est l'en-tête de la SEULE colonne exploitée.
const matchIDColumn = "match_id"

// LoadMatchIDs lit les match_id d'un fichier TSV ou d'une liste nue.
//
// Deux formes acceptées, parce que les deux existent dans le dépôt :
//   - un TSV avec ligne d'en-tête contenant `match_id` : la colonne est retrouvée par son
//     NOM, jamais par sa position (le TSV de la phase 1 en a 13, l'ordre peut changer) ;
//   - un fichier d'une colonne, sans en-tête : chaque ligne non vide est un match_id.
//
// Les lignes vides et les commentaires `#` sont ignorés ; les doublons sont écartés en
// conservant l'ordre du fichier (rejouer deux fois le même match ne casse rien, mais
// gonfle le décompte et le quota d'API pour rien).
func LoadMatchIDs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la liste %s : %w", path, err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	col := -1
	headerSeen := false
	var out []string
	seen := map[string]struct{}{}

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if !headerSeen {
			headerSeen = true
			if idx := indexOfColumn(fields, matchIDColumn); idx >= 0 {
				col = idx
				continue // la ligne d'en-tête ne porte pas de donnée
			}
			col = 0 // pas d'en-tête : liste nue, la donnée commence ici
		}
		if col >= len(fields) {
			return nil, fmt.Errorf("liste %s : ligne à %d colonnes, colonne %q attendue en position %d",
				path, len(fields), matchIDColumn, col)
		}
		id := strings.TrimSpace(fields[col])
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("lecture de la liste %s : %w", path, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("liste %s : aucun match_id exploitable", path)
	}
	return out, nil
}

// indexOfColumn retourne la position d'une colonne par son nom, -1 si absente.
func indexOfColumn(header []string, name string) int {
	for i := range header {
		if strings.EqualFold(strings.TrimSpace(header[i]), name) {
			return i
		}
	}
	return -1
}
