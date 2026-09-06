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
//
// … SAUF SUR LA FORME `= ?`, ET LA NUANCE EST MESURÉE (2026-09-06, constat F1 de la revue
// adversariale du lot 4). Quand la portée externe est une ÉGALITÉ (`e.match_id = ?`, le POC
// KillDistanceRepo), DuckDB la propage à travers les clés de jointure : le balayage porte le
// filtre même si la sous-requête ne demande rien, et le plan ne distingue plus les deux
// versions. Sur la forme `IN (...)` (le lecteur de la Synthèse), il ne le fait pas : la même
// mutation y rend DEUX balayages, le second nu. D'où deux niveaux d'assertion dans ce fichier :
// `verifieFragSoloPorteLeScope` juge le TEXTE composé par la production — rouge dans les deux
// cas — et `verifieScopePousse` juge le PLAN, qui reste le seul témoin de ce qui est vraiment
// balayé.
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

// filtreDeFragSolo extrait la clause `WHERE` de la sous-requête `fragSolo` — pas la
// sous-requête entière : sa liste de projection cite `s.match_id`, donc une recherche sur le
// texte complet serait toujours satisfaite, y compris sur un `WHERE TRUE`.
func filtreDeFragSolo(t *testing.T, query string) string {
	t.Helper()
	debut := strings.Index(query, "JOIN (")
	fin := strings.Index(query, ") fragSolo")
	if debut < 0 || fin < 0 || fin <= debut {
		t.Fatalf("sous-requête fragSolo introuvable — la jointure a changé de forme :\n%s", query)
	}
	branche := query[debut:fin]
	ou := strings.Index(branche, "WHERE ")
	groupe := strings.Index(branche, "GROUP BY")
	if ou < 0 || groupe < 0 || groupe <= ou {
		t.Fatalf("clause WHERE de fragSolo introuvable :\n%s", branche)
	}
	return branche[ou:groupe]
}

// verifieFragSoloPorteLeScope : la sous-requête, TELLE QUE LA PRODUCTION LA COMPOSE, filtre
// sur match_id — et le nombre de paramètres liés suit.
//
// POURQUOI CETTE VÉRIFICATION DE TEXTE EN PLUS DU PLAN (2026-09-06, constat F1). Le plan est
// le juge de ce qui compte — ce que DuckDB balaie réellement — mais il ne discrimine que la
// forme `IN (...)`. MESURÉ CE JOUR SUR LES DEUX LECTEURS : avec `e.match_id = ?` en portée
// externe, DuckDB PROPAGE l'égalité à travers les clés de jointure et filtre le balayage même
// quand la sous-requête ne demande rien (un seul balayage, `Filters="match_id='...'"`) ; avec
// `e.match_id IN (...)` il ne le fait pas (deux balayages, le second sans filtre). Le scope de
// `fragSolo` du POC n'est donc pas gratuit pour autant : il ne doit pas dépendre d'une
// propagation d'optimiseur qui peut disparaître d'une version de DuckDB à l'autre. Cette
// assertion-ci le tient, et elle est ROUGE sous la mutation que le plan laisse verte.
func verifieFragSoloPorteLeScope(t *testing.T, query string, args []any, attendus int) {
	t.Helper()
	if filtre := filtreDeFragSolo(t, query); !strings.Contains(filtre, "s.match_id") {
		t.Errorf("la clause WHERE de fragSolo ne porte AUCUN filtre sur match_id — elle juge "+
			"l'unicité du frag sur la vue entière (résidu 4.0a) : %q", filtre)
	}
	if len(args) != attendus {
		t.Errorf("%d paramètre(s) lié(s), attendu %d : les match_id du scope doivent être "+
			"dupliqués (sous-requête d'abord, portée externe ensuite)", len(args), attendus)
	}
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

	f := wrFilters()
	q, args := buildWeaponRangeQuery(positionsAtKill, weaponRangeKillerColumn, f)
	// 2 × les match_id (sous-requête puis portée externe) + le filtre de joueur par xuid.
	verifieFragSoloPorteLeScope(t, q, args, 2*len(f.MatchIDs)+len(f.XUIDs))
	verifieScopePousse(t, planDe(t, pdb, q, args))
}

// TestKillDistance_ScopeBorneLeBalayageDuKillFeed — le POC, forme `= ?`. Il partage l'helper
// depuis le lot 3 : son plan gagne le même filtre, et ses neuf tests de résultat d'origine
// font foi sur le fait que ses chiffres, eux, n'ont pas bougé.
//
// IL PASSE PAR `killDistanceQueryFor`, LE SITE D'APPEL DE PRODUCTION, et surtout PAS par une
// recomposition locale à partir des constantes : celle-ci jugeait des constantes, pas le
// lecteur (constat F1, revue adversariale du lot 4, 2026-09-06).
//
// CE QUE LE PLAN NE PEUT PAS DIRE ICI, MESURÉ LE 2026-09-06. Sur la forme `= ?`, la mutation
// « scope de fragSolo ramené à TRUE » laisse le plan INCHANGÉ : un seul balayage, toujours
// `Filters="match_id='...'"` — DuckDB propage l'égalité de la portée externe à travers les
// clés de jointure. (Sur la forme `IN (...)` du test frère, il ne le fait pas : la même
// mutation y rend deux balayages dont un nu, et ce test-là vire ROUGE.) La garde du POC repose
// donc sur `verifieFragSoloPorteLeScope`, qui juge la requête COMPOSÉE PAR LA PRODUCTION et
// vire rouge sous la mutation ; le plan reste vérifié par-dessus, pour ce qui compte vraiment.
func TestKillDistance_ScopeBorneLeBalayageDuKillFeed(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	q, args := killDistanceQueryFor(kscMatchID)
	// Le match_id est lié DEUX FOIS : sous-requête d'abord, portée externe ensuite.
	verifieFragSoloPorteLeScope(t, q, args, 2)
	verifieScopePousse(t, planDe(t, pdb, q, args))
}
