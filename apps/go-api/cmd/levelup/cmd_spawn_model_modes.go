package main

// cmd_spawn_model_modes.go — LES FAMILLES DE MODES DE LA MESURE DE REAPPARITION, et le registre
// des matchs qui les porte.
//
// Une famille se decide sur le nom de la variante de jeu et la categorie du registre. Les
// mots-cles couvrent les noms anglais du jeu et leurs equivalents francais ; une variante qui ne
// ressemble a rien n'a pas de famille et n'est pas mesuree. Le BTB et les rotations Husky Raid sont ecartes
// d'emblee : leurs cartes et leurs effectifs ne sont pas ceux de l'Arene.

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
)

// Familles de modes mesurees.
const (
	familleAssassin = "assassin"
	familleFiesta   = "fiesta"
	familleDrapeau  = "drapeau"
	familleBases    = "bases"
	familleColline  = "colline"
	familleOddball  = "oddball"
)

// famillesMesurees : l'ordre des rapports.
var famillesMesurees = []string{
	familleAssassin, familleFiesta, familleDrapeau, familleBases, familleColline, familleOddball,
}

// motsCles : mots d'une variante (en minuscules) -> famille, dans l'ordre d'essai.
var motsCles = []struct {
	famille string
	mots    []string
}{
	{familleDrapeau, []string{"ctf", "flag", "drapeau"}},
	{familleBases, []string{"stronghold", "bases"}},
	{familleColline, []string{"king of the hill", "koth", "colline"}},
	{familleOddball, []string{"oddball", "balle"}},
	{familleAssassin, []string{"slayer", "assassin"}},
}

// familleDeMode rend la famille d'une variante, ou "" si elle n'est pas mesuree (BTB, PvE,
// rotation Husky Raid, variante inconnue). Fiesta passe avant les modes d'objectif : une variante Fiesta reste
// Fiesta, ses armes et ses reapparitions ne sont pas celles de l'Arene.
func familleDeMode(variante, categorie string) string {
	v, c := strings.ToLower(strings.TrimSpace(variante)), strings.ToLower(strings.TrimSpace(categorie))
	if strings.HasPrefix(v, "btb") || c == "btb" || strings.Contains(c, "husky") {
		return ""
	}
	if strings.Contains(v, "fiesta") || strings.Contains(c, "fiesta") {
		return familleFiesta
	}
	for _, k := range motsCles {
		for _, m := range k.mots {
			if strings.Contains(v, m) {
				return k.famille
			}
		}
	}
	return ""
}

// infoMatch est ce que le registre sait d'un match.
type infoMatch struct {
	matchID, carteID, carteNom, variante, categorie, famille string
}

// registreDesMatchs lit le registre des matchs JcJ, indexe par la forme COURTE du match_id (la
// cle des artefacts).
func registreDesMatchs(ctx context.Context, db *sql.DB) (map[string]infoMatch, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT match_id, COALESCE(map_id, ''), COALESCE(map_name, ''),
		       COALESCE(game_variant_name, ''), COALESCE(mode_category, '')
		FROM match_registry
		WHERE COALESCE(is_firefight, FALSE) = FALSE`)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]infoMatch{}
	for rows.Next() {
		var m infoMatch
		if err := rows.Scan(&m.matchID, &m.carteID, &m.carteNom, &m.variante, &m.categorie); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		m.famille = familleDeMode(m.variante, m.categorie)
		out[titlePkg.FilmShortMatchID(m.matchID)] = m
	}
	return out, rows.Err()
}

// variantesDuParc compte, par (famille, variante, categorie), les matchs du registre dont
// l'artefact existe : de quoi verifier que les familles disent ce qu'on croit.
func variantesDuParc(registre map[string]infoMatch, shorts []string) []string {
	type cle struct{ famille, variante, categorie string }
	n := map[cle]int{}
	for _, s := range shorts {
		if m, ok := registre[s]; ok {
			n[cle{m.famille, m.variante, m.categorie}]++
		}
	}
	lignes := make([]string, 0, len(n))
	for k, v := range n {
		f := k.famille
		if f == "" {
			f = "(hors mesure)"
		}
		lignes = append(lignes, fmt.Sprintf("%-14s | %-40s | %-14s | %5d", f, k.variante, k.categorie, v))
	}
	sort.Strings(lignes)
	return lignes
}
