// Package service — match_history_match_id_search_test.go : la recherche par match ID de la
// page Explorer et sa TOLÉRANCE AUX BLANCS.
//
// Ce qui se joue ici n'est pas un confort de saisie. Un match ID se colle, il ne se tape pas :
// il vient d'un log, d'une URL, d'un message où la ligne a été repliée. Le collage ramène des
// blancs, et une comparaison littérale rendait alors ZÉRO ligne — un « ce match n'existe pas »
// pour un identifiant parfaitement valide. Le filtre retire donc tous les blancs de la requête
// avant de comparer, jamais ceux de la donnée (un GUID stocké n'en porte pas).
package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// matchIDSearchRows : deux matchs aux identifiants disjoints, de quoi distinguer un filtre qui
// mord d'un filtre qui laisse tout passer.
func matchIDSearchRows() []domain.MatchHistoryRawRow {
	return []domain.MatchHistoryRawRow{
		{MatchID: "0c0f4e70-5b3e-4a0e-9d1c-2f8b7a6d5e4c"},
		{MatchID: "9ab12345-1111-2222-3333-444455556666"},
	}
}

// idsOf : les identifiants rendus, dans l'ordre, pour comparer sans dépendre du reste de la row.
func idsOf(rows []domain.MatchHistoryRawRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.MatchID)
	}
	return out
}

func TestFilterByExplorerMatchIDSearchToleratesSpaces(t *testing.T) {
	rows := matchIDSearchRows()
	const target = "0c0f4e70-5b3e-4a0e-9d1c-2f8b7a6d5e4c"

	cases := []struct {
		name  string
		query string
	}{
		{"exact", target},
		{"blancs en tête et en queue", "  " + target + "\t"},
		{"saut de ligne d'un collage multi-ligne", "\n" + target + "\n"},
		{"blanc INTERNE — la source a replié la ligne", "0c0f4e70-5b3e-4a0e- 9d1c-2f8b7a6d5e4c"},
		{"espace insécable, celle des collages navigateur", " " + target},
		{"casse indifférente, comme avant", "  0C0F4E70-5B3E-4A0E-9D1C-2F8B7A6D5E4C  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := idsOf(filterByExplorerMatchIDSearch(rows, tc.query))
			if len(got) != 1 || got[0] != target {
				t.Fatalf("requête %q : attendu [%s], obtenu %v", tc.query, target, got)
			}
		})
	}
}

// Une requête qui ne contient QUE des blancs n'exprime aucun critère : elle vaut la requête
// vide, donc pas de filtre. Le contraire — ne rendre aucune ligne — ferait passer une barre
// d'espaces involontaire pour un « aucun résultat ».
func TestFilterByExplorerMatchIDSearchBlankIsNoFilter(t *testing.T) {
	rows := matchIDSearchRows()
	for _, q := range []string{"", "   ", "\t\n", " "} {
		if got := filterByExplorerMatchIDSearch(rows, q); len(got) != len(rows) {
			t.Fatalf("requête blanche %q : attendu les %d lignes, obtenu %d", q, len(rows), len(got))
		}
	}
}

// Le nettoyage des blancs n'élargit PAS la recherche : un identifiant absent reste absent.
func TestFilterByExplorerMatchIDSearchStillFiltersOut(t *testing.T) {
	if got := filterByExplorerMatchIDSearch(matchIDSearchRows(), "  deadbeef  "); len(got) != 0 {
		t.Fatalf("attendu aucune ligne, obtenu %v", idsOf(got))
	}
}
