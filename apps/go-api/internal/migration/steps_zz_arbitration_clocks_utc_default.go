package migration

// steps_zz_arbitration_clocks_utc_default.go — REPARE le DEFAULT de TOUTE colonne
// d'horodatage naive sur les bases EXISTANTES (lot S6, cloture de la campagne ouverte
// par R1/S2 sur `written_at`).
//
// PREFIXE `zz_` A DESSEIN, ne pas renommer sans lire ceci. Go execute les `init()` d'un
// package dans l'ordre ALPHABETIQUE des fichiers, et `TestSortByCanonicalIsNoOpOnCurrentRegistry`
// exige que l'ordre d'enregistrement reproduise exactement `canonicalOrder`. Ce step doit
// etre le DERNIER de l'ordre — comme son jumeau de S2 — parce qu'il repare des tables
// EXISTANTES : s'enregistrer avant un createur de table le ferait passer sur une base ou
// cette table n'existe pas encore, et le DEFAULT fautif du createur survivrait au boot.
// Sous son nom naturel (`steps_arbitration_...`) il s'enregistrait en 1er et le test
// echouait.
//
// Le defaut est celui de S2, sur les AUTRES colonnes : `<col> TIMESTAMP DEFAULT now()`
// (ou CURRENT_TIMESTAMP) rend un TIMESTAMPTZ que DuckDB coerce vers une colonne TIMESTAMP
// NAIVE par le fuseau de SESSION. Sur un poste a UTC+2, toute ligne ecrite sans valeur
// explicite se date deux heures dans le FUTUR, alors que les ecrivains applicatifs posent
// `time.Now().UTC()`.
//
// S2 avait borne la reparation a `written_at` parce que c'etait la colonne de tri des vues
// `<table>_latest` (ADR 0026). Le relevé S6 a montré que la borne etait trop etroite :
// `lusr_component_history` arbitre `lusr_component_history_latest` sur `computed_at`, dont
// le DEFAULT etait reste sensible au fuseau — et ses deux ecrivains (persist et
// sync/skill) la dataient donc sur deux horloges. Le mecanisme de perte silencieuse
// demontre par R1 s'appliquait a l'identique, sous un autre nom de colonne.
//
// POURQUOI UN STEP JUMEAU, ET NON L'EXTENSION DU PREDICAT DE S2. Le runner saute tout step
// deja inscrit au ledger (`runSteps` : `state, exists := applied[name]` → pas de
// `ApplySchema`). `written_at_default_utc_*` a deja tourne en prod : elargir SON prédicat
// n'aurait JAMAIS rejoué sur les bases qui portent le defaut, c'est-a-dire precisement
// celles a reparer. Un nom neuf est la seule forme de reparation qui atteigne le parc
// existant. Le step de S2 reste enregistre tel quel — son entree au ledger fait foi pour
// les bases qui l'ont joue, et il est desormais subsume par celui-ci (meme predicat, sans
// le filtre de nom de colonne) : les deux partagent la meme implementation.
//
// Data-driven a dessein, SANS liste de colonnes ni de tables a maintenir : une colonne
// TIMESTAMP naive dont le DEFAULT appelle une horloge sensible au fuseau est fautive quel
// que soit son nom. Enregistre sur les cinq targets ; no-op complet quand il n'y a rien a
// reparer.
//
// NON DESTRUCTIF, donc rejouable sans risque : `ALTER ... SET DEFAULT` ne touche AUCUNE
// ligne existante — il ne change que la valeur servie aux INSERT futurs. Aucun UPDATE de
// reparation n'est fait ici, et aucun ne doit l'etre : un UPDATE de masse sur ces tables
// serait le vecteur ART que la campagne append-only a eteint (ADR 0019/0026). L'historique
// deja ecrit reste biaise, et c'est assume — les lignes fautives sortent du corpus par
// vieillissement.
//
// Idempotent : apres l'ALTER, le DEFAULT normalise par DuckDB
// (`CAST(main.timezone('UTC', now()) AS TIMESTAMP)`) ne matche plus le predicat.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// timestampDefectiveDefaultsSQL liste les colonnes d'horodatage a reparer dans la base
// COURANTE. Le `%s` recoit le filtre de nom de colonne (vide = toutes). Filtres, dans
// l'ordre d'importance :
//   - `database_name = current_database()` : duckdb_columns() voit AUSSI les bases
//     ATTACHees (metadata l'est sur shared) — sans ce filtre on tenterait un ALTER sur une
//     base tenue en lecture seule par un autre chemin.
//   - `data_type = 'TIMESTAMP'` : une colonne TIMESTAMPTZ n'a pas le defaut (elle garde
//     l'instant absolu) ; la reparer serait un changement de semantique gratuit.
//   - DEFAULT sensible au fuseau ET pas deja converti (idempotence).
const timestampDefectiveDefaultsSQL = `
	SELECT c.table_name, c.column_name
	FROM duckdb_columns() c
	JOIN duckdb_tables() t
	  ON t.database_name = c.database_name
	 AND t.schema_name   = c.schema_name
	 AND t.table_name    = c.table_name
	WHERE c.database_name = current_database()
	  AND c.schema_name   = 'main'
	  AND c.data_type     = 'TIMESTAMP'
	  AND c.column_default IS NOT NULL
	  AND (lower(c.column_default) LIKE '%%now()%%'
	    OR lower(c.column_default) LIKE '%%current_timestamp%%')
	  AND lower(c.column_default) NOT LIKE '%%timezone(''utc''%%'
	  %s
	ORDER BY c.table_name, c.column_name`

// defectiveColumn — une colonne a reparer.
type defectiveColumn struct {
	Table  string
	Column string
}

// EnsureTimestampDefaultsUTC bascule vers TimestampDefaultUTC le DEFAULT de TOUTE colonne
// d'horodatage naive encore datee sur le fuseau de session, quel que soit son nom.
// Retourne le nombre de colonnes reparees. Exporte : rejoue aussi hors runner (outillage
// de reparation ponctuelle).
func EnsureTimestampDefaultsUTC(db *sql.DB) (int, error) {
	return repairDefectiveDefaults(db, "")
}

// repairDefectiveDefaults est l'implementation commune aux deux steps de la campagne.
// `colonneFiltre` est un fragment SQL additionnel (vide = toutes les colonnes).
func repairDefectiveDefaults(db *sql.DB, colonneFiltre string) (int, error) {
	ctx := context.Background()
	colonnes, err := defectiveDefaultColumns(ctx, db, colonneFiltre)
	if err != nil {
		return 0, err
	}
	for _, c := range colonnes {
		// Identifiants issus du catalogue DuckDB lui-meme (pas d'entree utilisateur),
		// quotes pour supporter un nom sensible a la casse.
		stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`,
			c.Table, c.Column, TimestampDefaultUTC)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return 0, fmt.Errorf("migration: DEFAULT UTC sur %s.%s: %w", c.Table, c.Column, err)
		}
		slog.InfoContext(ctx, "migration: DEFAULT d'horodatage bascule en UTC explicite",
			"table", c.Table, "column", c.Column)
	}
	return len(colonnes), nil
}

// defectiveDefaultColumns lit le catalogue et retourne les colonnes a reparer.
func defectiveDefaultColumns(ctx context.Context, db *sql.DB, colonneFiltre string) ([]defectiveColumn, error) {
	q := fmt.Sprintf(timestampDefectiveDefaultsSQL, colonneFiltre)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migration: detection des DEFAULT d'horodatage: %w", err)
	}
	defer rows.Close()
	var out []defectiveColumn
	for rows.Next() {
		var c defectiveColumn
		if err := rows.Scan(&c.Table, &c.Column); err != nil {
			return nil, fmt.Errorf("migration: scan des DEFAULT d'horodatage: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// arbitrationClocksStepName construit le nom du step par target (un nom = une ligne
// schema_migrations, et chaque base a son propre ledger).
func arbitrationClocksStepName(target TargetDB) string {
	return "arbitration_clocks_default_utc_" + string(target)
}

func init() {
	for _, target := range []TargetDB{
		TargetShared, TargetPlayer, TargetSharedPvE, TargetSharedSocial, TargetMetadata,
	} {
		Register(Migration{
			Name:     arbitrationClocksStepName(target),
			TargetDB: target,
			Description: "horodatages : DEFAULT en UTC explicite sur TOUTE colonne naive " +
				"(cloture de la campagne written_at — computed_at arbitre aussi une vue _latest)",
			ApplySchema: func(db *sql.DB) error {
				_, err := EnsureTimestampDefaultsUTC(db)
				return err
			},
		})
	}
}
