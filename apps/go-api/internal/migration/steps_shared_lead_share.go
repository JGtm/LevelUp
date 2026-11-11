package migration

// steps_shared_lead_share.go — table `match_lead_share` : LA MAINMISE, la part de la durée jouée
// pendant laquelle chaque camp est seul en tête du score, par match et par camp, LUE DE LA COURBE
// DE SCORE DE L'ARTEFACT DE REJEU.
//
// # CE QUE LA TABLE PORTE
//
// Deux natures de ligne (`row_kind`) dans UNE passe :
//
//	camp    une ligne par camp publié par le film : lead_ms = temps passé SEUL en tête ;
//	match   UNE ligne par passe (camp -1), écrite même quand la mesure est refusée : elle fait
//	        retenir la passe entière par la vue `_latest` et porte la raison du refus.
//
// Les colonnes du MATCH sont recopiées sur chaque ligne de la passe, pour que toute ligne se lise
// seule : measured / unmeasured_reason / doc_schema (mesuré ou non, et pourquoi), measured_ms
// (durée de la partie jouée), tie_ms (temps sans meneur), lead_changes (retournements),
// silent_camps (camps du roster sans série de score, qui concourent à zéro et ne mènent jamais).
//
// Somme invariante d'une passe mesurée : tie_ms + somme des lead_ms = measured_ms. Les parts se
// lisent en FRACTION 0..1 (ADR 0006) : lead_ms / measured_ms, calculée par le lecteur.
//
// # ANTI-ART (ADR 0019/0026/0030)
//
// INSERT purs (`persist.LeadSharePersister`), LECTURE VIA `match_lead_share_latest` UNIQUEMENT. La
// table entre dans les garde-rails : tables protégées de `sync/no_art_patterns_test.go`,
// `sync/append_only_state_guard_test.go`, la regexp de `duckdb/no_raw_rating_reads_test.go` et le
// registre de compaction. L'unité d'écriture est la PASSE : la vue rend la dernière passe ENTIÈRE
// par match, jamais la dernière ligne par clé.
//
// # MULTI-TITRE
//
// La table existe pour tous les titres ; elle reste VIDE sur un titre qui ne déclare pas
// `film.lead_share`. Le branchement se fait sur capability, jamais sur le slug.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "shared_match_lead_share_v1",
		TargetDB: TargetShared,
		Description: "Table append-only match_lead_share (temps en tete du score par camp, plus une " +
			"ligne match de couverture, lues de la courbe de score de l artefact range) + decode_pass " +
			"+ index match_id + vue match_lead_share_latest (derniere passe ENTIERE par match)",
		ApplySchema: applyMatchLeadShare,
	})
}

// applyMatchLeadShare crée la séquence, la table, son index et sa vue. Idempotente.
//
// ⚠ TOUT ÉLARGISSEMENT FUTUR passe par un step au NOM NEUF qui RECRÉE la vue dans la même
// transaction de schéma : DuckDB FIGE la liste de colonnes d'un `SELECT *` à la création.
func applyMatchLeadShare(db *sql.DB) error {
	return execScript(db, MatchLeadShareTableSQL("match_lead_share")+
		MatchLeadShareLatestViewSQL("match_lead_share"))
}

// MatchLeadShareTableSQL rend le DDL de la table et de son index pour une RÉFÉRENCE DE TABLE
// donnée. EXPORTÉ POUR QUE PERSONNE NE LE RECOPIE.
func MatchLeadShareTableSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchLeadShare, tableRef, tableRef)
}

// MatchLeadShareLatestViewSQL rend le DDL de la vue `_latest`.
func MatchLeadShareLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchLeadShareLatest, tableRef)
}

// AUCUN commentaire SQL dans le script : le splitter d'`ExecScript` coupe naïvement sur `;`
// (piège ADR 0026). Le vocabulaire des colonnes est décrit ci-dessus et sur
// `persist.LeadShareBatch`.
const ddlMatchLeadShare = `
	CREATE SEQUENCE IF NOT EXISTS match_lead_share_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		id                BIGINT    PRIMARY KEY DEFAULT nextval('match_lead_share_id_seq'),
		match_id          VARCHAR   NOT NULL,
		decode_pass       VARCHAR   NOT NULL,
		written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		row_kind          VARCHAR   NOT NULL,
		camp              INTEGER   NOT NULL,
		lead_ms           BIGINT    NOT NULL,
		measured          BOOLEAN   NOT NULL,
		unmeasured_reason VARCHAR   NOT NULL,
		doc_schema        INTEGER   NOT NULL,
		measured_ms       BIGINT    NOT NULL,
		tie_ms            BIGINT    NOT NULL,
		lead_changes      INTEGER   NOT NULL,
		silent_camps      INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_lead_share_match
		ON %s(match_id);
`

// ddlMatchLeadShareLatest : LE SEUL CHEMIN DE LECTURE AUTORISÉ (ADR 0026). Patron EXACT de
// `match_vehicle_takes_latest` : la DERNIÈRE PASSE ENTIÈRE par match.
const ddlMatchLeadShareLatest = `
	CREATE OR REPLACE VIEW match_lead_share_latest AS
	SELECT * FROM %s AS t
	QUALIFY t.decode_pass = FIRST_VALUE(t.decode_pass) OVER (
		PARTITION BY t.match_id ORDER BY t.written_at DESC, t.id DESC
	);
`
