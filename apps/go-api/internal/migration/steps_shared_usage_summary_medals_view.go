package migration

// steps_shared_usage_summary_medals_view.go — la vue `v_medals_earned` : les médailles d'un joueur
// sur un match, table `medals_earned` ET médailles que LevelUp dérive du film.
//
// Les médailles « Vengeur » (9000000001) et « Cible difficile » (9000000002) ne s'écrivent pas dans
// `medals_earned` : une ligne de cette table n'a pas de réécriture possible (le seul moyen serait un
// UPDATE, interdit sur ces tables), et leur valeur change quand la règle change. Elles se lisent dans
// les colonnes du résumé d'usage (`revenge_kills`, `shield_recharges`), dernière passe de chaque match, écrites en
// INSERT pur à chaque passe du résumé d'usage.
//
// Colonnes de la vue : match_id, xuid, medal_name_id, count (la date d'écriture de `medals_earned`
// n'a aucun lecteur d'affichage et n'a pas d'équivalent pour une valeur dérivée).
//
// Règles de fusion :
//   - Vengeur : dès qu'un match porte une valeur dérivée (au moins une ligne de joueur avec
//     `revenge_kills` non NULL), elle REMPLACE les lignes héritées 9000000001 du match (un ancien calcul,
//     que plus aucun code n'écrit, y comptait une même mort plusieurs fois) ; sans valeur dérivée, les
//     lignes héritées restent. Une valeur dérivée à zéro n'est pas une ligne.
//   - Cible difficile : une ligne par joueur dont `shield_recharges` est positif.
//   - Toute autre médaille : la ligne de `medals_earned`, telle quelle.
//
// Forme de la vue (ADR 0036 I2). DuckDB ne pousse sous une fenêtre `_latest` qu'un filtre constant sur
// la clé de partition, et ne le propage à travers une jointure que s'il est une égalité ou s'il porte
// la colonne de la fenêtre elle-même. La vue est donc écrite pour qu'une lecture bornée à des matchs
// (égalité ou liste) arrive jusque sous les fenêtres :
//   - les branches dérivées partent de `match_usage_films_latest` et sortent SA colonne `match_id`
//     (la dernière passe d'un match se décide sur le film, comme `match_usage_players_latest`, dont la
//     jointure est reprise) : un filtre sur la liste descend dans la fenêtre du film ; lire
//     `match_usage_players_latest` laissait cette fenêtre sur toute la table dès qu'une liste était liée ;
//   - « un Vengeur dérivé existe pour ce match » se décide par une fenêtre `PARTITION BY match_id` sur
//     l'union, jamais par un `NOT EXISTS` corrélé (le filtre du match ne le traverse pas).
// Une lecture de tout l'historique (totaux de carrière) paie la fenêtre entière, ce que l'ADR admet.
// Test : platform/duckdb/medals_earned_fenetres_test.go.
//
// Sur une base sans colonnes dérivées (base partagée d'un titre dont les tables du résumé d'usage sont
// vides ou absentes), la vue rend `medals_earned` seul. Les lecteurs d'affichage lisent la vue ; la
// table brute reste réservée à une liste fermée (archlint, medals_earned_lecture_brute_test.go).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_medals_earned_view_v1",
		TargetDB:    TargetShared,
		Description: "Vue v_medals_earned : medals_earned fusionnée avec les médailles Vengeur et Cible difficile dérivées du film (résumé d'usage)",
		ApplySchema: ApplyMedalsEarnedView,
	})
}

const ddlMedalsEarnedViewPlain = `
	CREATE OR REPLACE VIEW v_medals_earned AS
	SELECT m.match_id, m.xuid, m.medal_name_id, m.count
	FROM medals_earned AS m;
`

const ddlMedalsEarnedViewMerged = `
	CREATE OR REPLACE VIEW v_medals_earned AS
	SELECT match_id, xuid, medal_name_id, count
	FROM (
	    SELECT m.match_id, m.xuid, m.medal_name_id, m.count,
	           CASE WHEN m.medal_name_id = 9000000001 THEN 'legacy' ELSE 'raw' END AS kind
	    FROM medals_earned AS m
	    UNION ALL
	    SELECT f.match_id, u.xuid, CAST(9000000001 AS BIGINT), u.revenge_kills, 'derived'
	    FROM match_usage_films_latest AS f
	    JOIN match_usage_players AS u ON u.match_id = f.match_id AND u.summary_pass = f.summary_pass
	    WHERE u.revenge_kills IS NOT NULL
	    UNION ALL
	    SELECT f.match_id, u.xuid, CAST(9000000002 AS BIGINT), u.shield_recharges, 'raw'
	    FROM match_usage_films_latest AS f
	    JOIN match_usage_players AS u ON u.match_id = f.match_id AND u.summary_pass = f.summary_pass
	    WHERE u.shield_recharges > 0
	) AS t
	QUALIFY (t.kind = 'raw')
	     OR (t.kind = 'derived' AND t.count > 0)
	     OR (t.kind = 'legacy' AND NOT bool_or(t.kind = 'derived') OVER (PARTITION BY t.match_id));
`

// ApplyMedalsEarnedView pose la vue (exportée : les doublures de test des lecteurs la posent aussi).
// Sans table `medals_earned` rien n'est posé (aucun lecteur n'a de médailles à lire) ; sans les
// colonnes dérivées, la vue rend la table seule.
func ApplyMedalsEarnedView(db *sql.DB) error {
	ok, err := tableExists(db, "medals_earned")
	if err != nil || !ok {
		return err
	}
	derived := true
	for _, col := range []string{"revenge_kills", "shield_recharges"} {
		has, err := columnExists(db, "match_usage_players", col)
		if err != nil {
			return err
		}
		derived = derived && has
	}
	if !derived {
		return execScript(db, ddlMedalsEarnedViewPlain)
	}
	return execScript(db, ddlMedalsEarnedViewMerged)
}
