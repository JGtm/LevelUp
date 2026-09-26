package migration

// compaction_registry.go — LES TABLES QUE LA COMPACTION A LE DROIT DE RÉDUIRE, et la règle qui dit
// quelles lignes elle garde.
//
// Une table n'entre ici que si (inventaire C.1 du plan
// `.ai/PLAN_PERF_COMPACTION_ET_PERIMETRE_JOUEUR_2026-09-26.md`, DC.1) :
//   (a) sa vue `_latest` retient « toutes les lignes de la dernière passe par match » ou « la
//       dernière ligne par clé », sans fusion de colonnes ;
//   (b) aucun lecteur ne lit ses anciennes passes — elles sont des passes de DÉCODAGE supersédées,
//       dont l'utilisateur a confirmé qu'elles n'ont aucun usage (2026-09-26).
//
// La règle de chaque table est écrite DEUX fois, et c'est voulu : `keep` est la clause que la
// compaction applique, `signature` est le fragment que le SQL de la vue en base DOIT contenir.
// Si une migration change un jour la règle d'une vue sans toucher ce registre, la signature ne se
// retrouve plus et la compaction REFUSE la table au lieu de garder les mauvaises lignes.
//
// Title-agnostic : la base partagée des matchs a le même schéma pour chaque titre (vues
// identiques, mesuré sur Halo Infinite et Halo 5). Une table absente d'un titre est sautée.

import (
	"fmt"
	"regexp"
	"strings"
)

// compactable : une table du registre et sa règle.
type compactable struct {
	// Table : la table append-only compactée.
	Table string
	// View : la vue `_latest` qui la lit — l'unique chemin de lecture, dont la sortie doit être
	// identique avant et après.
	View string
	// keep : clause appliquée à `SELECT * FROM <Table>` qui rend exactement les lignes que la vue
	// retient (la passe entière, y compris ce qu'un filtre supplémentaire de la vue écarte).
	keep string
	// signature : fragment NORMALISÉ (cf. normaliserVue) que le SQL de la vue doit contenir.
	signature string
}

// ordreDeVersion : l'ordre qui désigne la ligne la plus récente, commun à toutes les vues du
// registre.
const ordreDeVersion = "ORDER BY written_at DESC, id DESC"

// dernierePasse : règle « dernière passe entière par match », colonne de passe `col`.
func dernierePasse(table, col string) compactable {
	regle := fmt.Sprintf("%s = first_value(%s) OVER (PARTITION BY match_id %s)", col, col, ordreDeVersion)
	return compactable{
		Table:     table,
		View:      table + "_latest",
		keep:      "QUALIFY " + regle,
		signature: normaliserVue("FROM " + table + " QUALIFY " + regle),
	}
}

// derniereLigneParCle : règle « dernière ligne par clé » (clé `cle`, liste SQL).
func derniereLigneParCle(table, cle string) compactable {
	regle := fmt.Sprintf("row_number() OVER (PARTITION BY %s %s) = 1", cle, ordreDeVersion)
	return compactable{
		Table:     table,
		View:      table + "_latest",
		keep:      "QUALIFY " + regle,
		signature: normaliserVue("FROM " + table + " QUALIFY " + regle),
	}
}

// passeDeLaVueParente : règle « la passe que retient la vue d'une autre table » (les joueurs
// d'un résumé d'usage suivent la passe de leur film).
func passeDeLaVueParente(table, vueParente, col string) compactable {
	return compactable{
		Table: table,
		View:  table + "_latest",
		keep: fmt.Sprintf("WHERE (match_id, %s) IN (SELECT match_id, %s FROM %s)",
			col, col, vueParente),
		signature: normaliserVue(fmt.Sprintf(
			"FROM %s INNER JOIN %s ON match_id = match_id AND %s = %s", table, vueParente, col, col)),
	}
}

// tablesCompactables : le registre. `match_usage_films` précède `match_usage_players` par
// lisibilité seulement — la règle des joueurs lit la VUE des films, que la compaction des films
// laisse identique.
var tablesCompactables = []compactable{
	dernierePasse("match_kill_events", "decode_pass"),
	dernierePasse("match_lives", "decode_pass"),
	dernierePasse("match_death_context", "decode_pass"),
	dernierePasse("kill_openings", "decode_pass"),
	dernierePasse("kill_positions", "decode_pass"),
	dernierePasse("match_weapon_shots", "decode_pass"),
	dernierePasse("match_weapon_hit_distance", "decode_pass"),
	dernierePasse("match_player_positions", "positions_pass"),
	dernierePasse("match_usage_films", "summary_pass"),
	passeDeLaVueParente("match_usage_players", "match_usage_films_latest", "summary_pass"),
	dernierePasse("match_pad_pickups_by_tier", "decode_pass"),
	dernierePasse("match_flag_grabs_net", "decode_pass"),
	derniereLigneParCle("match_bomb_stats", "match_id, xuid"),
}

// CompactableTables rend les noms des tables du registre, dans l'ordre de traitement.
func CompactableTables() []string {
	out := make([]string, 0, len(tablesCompactables))
	for _, c := range tablesCompactables {
		out = append(out, c.Table)
	}
	return out
}

var (
	reQualifiant    = regexp.MustCompile(`\b[a-z_][a-z0-9_]*\.`)
	reAliasDeFrom   = regexp.MustCompile(`\bas [a-z_][a-z0-9_]* `)
	reNonSignifiant = regexp.MustCompile(`[\s()]+`)
)

// normaliserVue ramène un SQL de vue à une forme comparable : minuscules, qualifiants d'alias
// retirés (`e.decode_pass` -> `decode_pass`), alias de FROM retirés (`FROM t AS e` -> `FROM t`),
// espaces et parenthèses retirés (DuckDB réécrit le SQL d'une vue avec ses propres parenthèses).
func normaliserVue(sqlVue string) string {
	s := strings.ToLower(sqlVue)
	s = reQualifiant.ReplaceAllString(s, "")
	s = reAliasDeFrom.ReplaceAllString(s, " ")
	return reNonSignifiant.ReplaceAllString(s, "")
}

// regleDeLaVueTenue dit si le SQL de la vue en base porte encore la règle du registre.
func (c compactable) regleDeLaVueTenue(sqlVue string) bool {
	return strings.Contains(normaliserVue(sqlVue), c.signature)
}
