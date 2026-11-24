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
//   - Vengeur : dès qu'un match porte une valeur dérivée, elle REMPLACE les lignes héritées 9000000001 du
//     match (un ancien calcul, que plus aucun code n'écrit, y comptait une même mort plusieurs fois) ;
//     sans valeur dérivée, les lignes héritées restent. Une valeur dérivée à zéro n'est pas une ligne.
//     « Le match porte une valeur dérivée » se décide sur l'EXISTENCE d'une passe de LevelUp mesurée pour
//     le match (`match_usage_films_latest.levelup_rev` rangée, ce que la passe ne pose qu'une fois le fil
//     des morts et les équipes lus), pas sur la présence d'une ligne de joueur : une passe mesurée qui n'a
//     produit aucune ligne de joueur dit « zéro Vengeur », et l'ancien décompte ne revient pas (étape
//     `shared_medals_earned_view_v2`). Avant elle (v1), la décision se prenait sur une ligne de joueur.
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
		ApplySchema: applyMedalsEarnedViewV1,
	})
	// Étape NEUVE : l'étape v1 est déjà jouée en production, elle ne se modifie pas. Enregistrée juste après
	// elle, dans le même fichier : l'ordre du registre suit canonicalOrder.
	Register(Migration{
		Name:        "shared_medals_earned_view_v2",
		TargetDB:    TargetShared,
		Description: "Vue v_medals_earned : « le match porte une valeur dérivée » se décide sur la passe de LevelUp mesurée du match (levelup_rev), plus sur la présence d'une ligne de joueur",
		ApplySchema: ApplyMedalsEarnedView,
	})
}

const ddlMedalsEarnedViewPlain = `
	CREATE OR REPLACE VIEW v_medals_earned AS
	SELECT m.match_id, m.xuid, m.medal_name_id, m.count
	FROM medals_earned AS m;
`

// ddlMedalsEarnedViewMergedV1 : la vue telle que l'étape v1 la pose (décision « dérivé » sur une ligne de
// joueur). Gardée pour que l'étape v1 reste ce qu'elle était ; la vue servie est la v2.
const ddlMedalsEarnedViewMergedV1 = `
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

// ddlMedalsEarnedViewMerged : la vue servie. Comme la v1, plus une ligne « marqueur » par match dont la
// dernière passe de LevelUp est mesurée (`levelup_rev` non vide) : elle ne sort jamais (écartée par le
// QUALIFY), elle dit seulement que le match porte une valeur dérivée, ce qui retire le Vengeur hérité même
// quand la passe n'a produit aucune ligne de joueur. Le marqueur part de `match_usage_films_latest` et sort
// sa colonne `match_id` : un filtre sur des matchs descend dans la fenêtre du film (ADR 0036 I2).
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
	    UNION ALL
	    SELECT f.match_id, '', CAST(0 AS BIGINT), CAST(0 AS SMALLINT), 'marker'
	    FROM match_usage_films_latest AS f
	    WHERE COALESCE(f.levelup_rev, '') <> ''
	) AS t
	QUALIFY (t.kind = 'raw')
	     OR (t.kind = 'derived' AND t.count > 0)
	     OR (t.kind = 'legacy' AND NOT bool_or(t.kind IN ('derived', 'marker')) OVER (PARTITION BY t.match_id));
`

// applyMedalsEarnedViewV1 : ce que l'étape v1 posait (vue de décision sur une ligne de joueur).
func applyMedalsEarnedViewV1(db *sql.DB) error {
	return applyMedalsEarnedViewDDL(db, ddlMedalsEarnedViewMergedV1)
}

// ApplyMedalsEarnedView pose la vue (exportée : les doublures de test des lecteurs la posent aussi).
// Sans table `medals_earned` rien n'est posé (aucun lecteur n'a de médailles à lire) ; sans les
// colonnes dérivées, la vue rend la table seule.
func ApplyMedalsEarnedView(db *sql.DB) error {
	return applyMedalsEarnedViewDDL(db, ddlMedalsEarnedViewMerged)
}

// applyMedalsEarnedViewDDL pose la vue fusionnée `merged` (ou la vue simple, voir ApplyMedalsEarnedView).
func applyMedalsEarnedViewDDL(db *sql.DB, merged string) error {
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
	return execScript(db, merged)
}
