// Package duckdb — squad_repo_assist_pairs_test.go : Q32d contre une VRAIE base DuckDB
// en mémoire, au schéma de production (cf. match_view_repo_assist_pairs_test.go pour la
// raison : la lecture passe par la vue `_latest`, pas par la table).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// querySquadAssistPairs joue Q32d + son scan, comme le fait le lecteur du repo.
func querySquadAssistPairs(
	t *testing.T, db *sql.DB, matchIDs, squadXUIDs []string,
) (pairsByKey map[string][2]int, measured int) {
	t.Helper()
	query := fmt.Sprintf(Q32dSquadAssistPairsTemplate,
		placeholderList(len(matchIDs)),
		placeholderList(len(matchIDs)),
		placeholderList(len(squadXUIDs)),
		placeholderList(len(squadXUIDs)),
	)
	args := make([]interface{}, 0, 2*len(matchIDs)+2*len(squadXUIDs))
	args = appendStrArgs(args, matchIDs)
	args = appendStrArgs(args, matchIDs)
	args = appendStrArgs(args, squadXUIDs)
	args = appendStrArgs(args, squadXUIDs)

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("Q32d: %v", err)
	}
	defer rows.Close()
	raw, measured, err := scanSquadAssistPairs(rows)
	if err != nil {
		t.Fatalf("scanSquadAssistPairs: %v", err)
	}
	pairsByKey = make(map[string][2]int, len(raw))
	for _, p := range raw {
		pairsByKey[p.AssistXUID+">"+p.KillerXUID] = [2]int{p.AssistCount, p.StolenCount}
	}
	return pairsByKey, measured
}

// TestQ32dSquadAssistPairs_InternesSeules : LE FILTRE QUI DÉFINIT LA QUESTION POSÉE.
// La page Synergies parle de l'escouade : une assistance rendue à un joueur hors
// escouade — ou reçue d'un joueur hors escouade — n'y répond pas et gonflerait le
// dénominateur de la colonne « part ».
func TestQ32dSquadAssistPairs_InternesSeules(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		// interne × 2, dont une volée
		{"m1", true, 1000, "v1", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(30), intPtr(69)},
		{"m1", true, 2000, "v2", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(80), intPtr(19)},
		// assistant DANS l'escouade, tueur DEHORS -> écartée
		{"m1", true, 3000, "v3", strPtr("X9"), true, strPtr("Un"), strPtr("S1"), intPtr(30), intPtr(69)},
		// assistant DEHORS, tueur dans l'escouade -> écartée
		{"m2", true, 4000, "v4", strPtr("S1"), true, strPtr("Etr"), strPtr("X9"), intPtr(30), intPtr(69)},
		// interne sur un autre match de la sélection
		{"m2", true, 5000, "v5", strPtr("S1"), true, strPtr("Deux"), strPtr("S2"), intPtr(10), intPtr(89)},
		// match HORS sélection -> jamais lu
		{"m3", true, 6000, "v6", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(10), intPtr(89)},
	})
	pairs, measured := querySquadAssistPairs(t, db, []string{"m1", "m2"}, []string{"S1", "S2"})

	if measured != 2 {
		t.Errorf("matchs mesurés = %d, attendu 2", measured)
	}
	if len(pairs) != 2 {
		t.Fatalf("paires = %+v, attendu 2 (S1>S2 et S2>S1)", pairs)
	}
	if got := pairs["S1>S2"]; got != [2]int{2, 1} {
		t.Errorf("S1>S2 = %v, attendu [2 1]", got)
	}
	if got := pairs["S2>S1"]; got != [2]int{1, 1} {
		t.Errorf("S2>S1 = %v, attendu [1 1]", got)
	}
}

// TestQ32dSquadAssistPairs_CouvertureSansPaire : la sélection est mesurée mais l'escouade
// ne s'est pas entraidée. La COUVERTURE doit sortir quand même — sans elle, « aucune
// assistance » et « rien mesuré » rendraient tous deux zéro ligne.
func TestQ32dSquadAssistPairs_CouvertureSansPaire(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("S1"), true, nil, nil, intPtr(100), nil},
		{"m2", true, 2000, "v2", strPtr("S2"), true, strPtr("Etr"), strPtr("X9"), intPtr(30), intPtr(69)},
	})
	pairs, measured := querySquadAssistPairs(t, db, []string{"m1", "m2"}, []string{"S1", "S2"})
	if len(pairs) != 0 {
		t.Fatalf("paires = %+v, attendu aucune", pairs)
	}
	if measured != 2 {
		t.Errorf("matchs mesurés = %d, attendu 2", measured)
	}
}

// TestQ32dSquadAssistPairs_CouverturePartielle : le cœur du bandeau « mesuré sur N des M ».
// Un match sans film et un match non publiable ligne à ligne ne comptent PAS comme mesurés.
func TestQ32dSquadAssistPairs_CouverturePartielle(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(30), intPtr(69)},
		// m2 : film présent mais assistance non mesurée
		{"m2", true, 2000, "v2", strPtr("S2"), false, nil, nil, nil, nil},
		// m3 : mesuré mais NON publiable ligne à ligne (BTB)
		{"m3", false, 3000, "v3", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(30), intPtr(69)},
		// m4 : aucune ligne du tout (absent de la table)
	})
	pairs, measured := querySquadAssistPairs(t, db,
		[]string{"m1", "m2", "m3", "m4"}, []string{"S1", "S2"})
	if measured != 1 {
		t.Errorf("matchs mesurés = %d, attendu 1 (seul m1)", measured)
	}
	if got := pairs["S1>S2"]; got != [2]int{1, 1} {
		t.Errorf("S1>S2 = %v, attendu [1 1] (m3 non publiable écarté)", got)
	}
}

// TestQ32dSquadAssistPairs_SelectionVide : aucune ligne dans la sélection → couverture à
// zéro et pas une paire. Le builder n'émet alors aucun bloc.
func TestQ32dSquadAssistPairs_SelectionVide(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"autre", true, 1000, "v1", strPtr("S2"), true, strPtr("Un"), strPtr("S1"), intPtr(30), intPtr(69)},
	})
	pairs, measured := querySquadAssistPairs(t, db, []string{"m1"}, []string{"S1", "S2"})
	if len(pairs) != 0 || measured != 0 {
		t.Fatalf("paires = %+v, mesurés = %d ; attendu aucune paire et 0", pairs, measured)
	}
}

// TestPlaceholderList : le gabarit reçoit exactement n placeholders, sans virgule
// pendante — une virgule de trop rend un SQL invalide, et le lecteur dégrade alors en
// silence (couverture à zéro) au lieu de signaler.
func TestPlaceholderList(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{
		{1, "?"},
		{2, "?,?"},
		{5, "?,?,?,?,?"},
	} {
		if got := placeholderList(c.n); got != c.want {
			t.Errorf("placeholderList(%d) = %q, attendu %q", c.n, got, c.want)
		}
		if strings.HasSuffix(placeholderList(c.n), ",") {
			t.Errorf("placeholderList(%d) finit par une virgule", c.n)
		}
	}
}
