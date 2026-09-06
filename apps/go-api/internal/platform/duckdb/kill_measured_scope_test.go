//go:build integration

// Package duckdb — kill_measured_scope_test.go : LA SOUS-REQUÊTE `fragSolo` EST BORNÉE.
//
// Résidu 4.0a du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md. La garde d'unicité du frag se juge
// sur un GROUPE, donc dans une branche de jointure — et tant que cette branche ne portait pas
// son propre scope, DuckDB ne pouvait POUSSER aucun filtre jusqu'au balayage de
// `match_kill_events` : la vue entière était lue à chaque lecture de Synthèse (mesuré ×15,6,
// 192 ms contre 12 ms sur 140 000 événements).
//
// POURQUOI UN TEST DE PLAN ET PAS UN TEST DE RÉSULTAT. Le scope de la sous-requête ne change
// AUCUN résultat, et c'est précisément ce qui le rend licite : les deux morts d'un double frag
// partagent le même `match_id` (première colonne de la clé), donc aucune ne sort du scope sans
// l'autre — la garde juge le même groupe qu'avant. Le retirer laisserait toute la suite verte.
// Seul le plan d'exécution le dit.
//
// FORME DU PLAN, VÉRIFIÉE SUR PIÈCES LE 2026-09-06 : DuckDB matérialise la vue
// `match_kill_events_latest` en UNE `CTE` que les deux branches relisent (`CTE_SCAN`). Il n'y a
// donc qu'UN balayage de `match_kill_events`, et la question est de savoir s'il porte le
// filtre — il ne peut le porter que si les DEUX branches le demandent.
package duckdb

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// noeudPlan — un opérateur du plan JSON de DuckDB. `extra_info` est hétérogène (Projections
// est tantôt un tableau, tantôt une chaîne) : on ne décode que ce qu'on lit.
type noeudPlan struct {
	Name      string                     `json:"name"`
	Children  []noeudPlan                `json:"children"`
	ExtraInfo map[string]json.RawMessage `json:"extra_info"`
}

// texteExtra rend une entrée d'extra_info quand c'est une chaîne, "" sinon.
func (n noeudPlan) texteExtra(cle string) string {
	raw, ok := n.ExtraInfo[cle]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// planDe rend le plan d'exécution JSON d'une requête. Le format JSON est choisi contre le
// format par défaut, dont les boîtes ASCII COUPENT le texte des filtres à 27 caractères : un
// détecteur posé dessus raterait silencieusement les filtres longs, c'est-à-dire justement
// ceux d'un vrai scope multi-matchs.
func planDe(t *testing.T, pdb *PlayerDB, query string, args []any) noeudPlan {
	t.Helper()
	rows, err := pdb.Shared.Query(context.Background(), "EXPLAIN (FORMAT JSON) "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN: %v", err)
	}
	defer rows.Close()
	var brut string
	for rows.Next() {
		var cle, valeur string
		if err := rows.Scan(&cle, &valeur); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		brut = valeur
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	var racines []noeudPlan
	if err := json.Unmarshal([]byte(brut), &racines); err != nil {
		t.Fatalf("plan illisible: %v\n%s", err, brut)
	}
	if len(racines) != 1 {
		t.Fatalf("plan à %d racine(s), attendu 1", len(racines))
	}
	return racines[0]
}

// balayagesDeTable rend les SEQ_SCAN du plan portant sur une table donnée (nom qualifié
// `catalogue.schema.table` : on compare le dernier segment).
func balayagesDeTable(n noeudPlan, table string, out []noeudPlan) []noeudPlan {
	if n.Name == "SEQ_SCAN" {
		qualifie := n.texteExtra("Table")
		if segments := strings.Split(qualifie, "."); segments[len(segments)-1] == table {
			out = append(out, n)
		}
	}
	for _, enfant := range n.Children {
		out = balayagesDeTable(enfant, table, out)
	}
	return out
}

// verifieScopePousse : tout balayage du kill-feed porte un filtre sur match_id.
func verifieScopePousse(t *testing.T, plan noeudPlan) {
	t.Helper()
	scans := balayagesDeTable(plan, "match_kill_events", nil)
	if len(scans) == 0 {
		t.Fatal("aucun balayage de match_kill_events dans le plan — la requête a changé de forme")
	}
	for i, s := range scans {
		filtres := s.texteExtra("Filters")
		if !strings.Contains(filtres, "match_id") {
			t.Errorf("balayage %d/%d de match_kill_events : Filters = %q, attendu un filtre sur "+
				"match_id — la branche fragSolo balaie la vue entière (résidu 4.0a)",
				i+1, len(scans), filtres)
		}
	}
}

// TestFragSolo_ScopeBorneLeBalayageDuKillFeed — le lecteur de la Synthèse (forme `IN (...)`).
//
// MUTATION PROUVÉE ROUGE le 2026-09-06 : `fragScope` ramené à `TRUE` dans
// buildWeaponRangeQuery -> « Filters = "" , attendu un filtre sur match_id », alors que les
// douze tests de résultat WeaponRange restent verts.
func TestFragSolo_ScopeBorneLeBalayageDuKillFeed(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	q, args := buildWeaponRangeQuery(positionsAtKill, weaponRangeKillerColumn, wrFilters())
	verifieScopePousse(t, planDe(t, pdb, q, args))
}

// TestKillDistance_ScopeBorneLeBalayageDuKillFeed — le POC, forme `= ?`. Il partage l'helper
// depuis le lot 3 : son plan gagne le même filtre, et ses neuf tests de résultat d'origine
// font foi sur le fait que ses chiffres, eux, n'ont pas bougé.
func TestKillDistance_ScopeBorneLeBalayageDuKillFeed(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	q := measuredKillsQuery(positionsAtKill, killDistanceFragScope, killDistanceWhere)
	verifieScopePousse(t, planDe(t, pdb, q, []any{kscMatchID, kscMatchID}))
}
