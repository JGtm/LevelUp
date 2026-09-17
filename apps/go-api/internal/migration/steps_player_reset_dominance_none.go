package migration

// steps_player_reset_dominance_none.go — recalcul des badges « aucun » après
// l'ajout de SABORDAGE (6) / ABNÉGATION (7) — 2026-09-16.
//
// Le post-sync ne (re)calcule le dominance_flag que des matchs où il est NULL
// (selectMatchesMissingDominanceFlags) : 0 est terminal. Les matchs à objectifs
// déjà classés 0 ne recevraient donc jamais les deux nouveaux badges. Décision
// utilisateur : tout l'historique est recalculé.
//
// Stratégie append-only (ADR 0026, pas d'UPDATE) : une row stage='dominance' à
// dominance_flag NULL est INSÉRÉE pour chaque match dont le flag courant vaut 0.
// La vue _latest sert la row la plus récente du stage → NULL → le post-sync
// suivant recalcule ces matchs et insère la valeur définitive.
//
// PORTÉE RÉELLE DU RECALCUL (précisé le 2026-09-17) : les matchs remis à NULL rejouent
// TOUTE la chaîne `computeMatchDominanceFlag`, pas seulement les deux nouveaux badges. Un
// match aujourd'hui à 0 peut donc ressortir à 3, 4 ou 5 si ses données ont évolué depuis
// son premier calcul (courbe d'objectif écrite par la page film, kill events décodés plus
// tard, participants complétés) — c'est le comportement voulu : le flag suit la donnée
// disponible. Seuls les flags DÉJÀ non nuls ne sont pas touchés, faute d'être sélectionnés.
// Un match Slayer sans médaille ni courbe redonne 0 (le mode vit dans la shared DB, hors de
// portée d'une migration player : la remise à blanc ne peut pas le filtrer).
//
// À LIVRER DANS LA MÊME VERSION que les correctifs de `fragContrastDominanceFlag` (durée de
// match absente, comptage des équipes) : la migration se joue au premier démarrage avec le
// code déployé, donc c'est ce code-là qui écrit les valeurs définitives de tout l'historique.
//
// Gardé par la présence des colonnes stage/dominance_flag (base antérieure à la
// conversion append-only ou sans la colonne : rien à recalculer).

import "database/sql"

func init() {
	Register(Migration{
		Name:     "player_dominance_flag_reset_none_v1",
		TargetDB: TargetPlayer,
		Description: "Remet à NULL (row append-only stage='dominance') les dominance_flag à 0 " +
			"pour recalcul post-sync avec SABORDAGE/ABNÉGATION",
		ApplySchema: applyResetDominanceNone,
	})
}

func applyResetDominanceNone(db *sql.DB) error {
	for _, col := range []string{"stage", "dominance_flag"} {
		ok, err := columnExists(db, "player_match_enrichment", col)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	return execScript(db, `
		INSERT INTO player_match_enrichment (match_id, dominance_flag, stage)
		SELECT match_id, NULL, 'dominance'
		FROM player_match_enrichment_latest
		WHERE dominance_flag = 0;
	`)
}
