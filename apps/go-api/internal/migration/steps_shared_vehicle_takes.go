package migration

// steps_shared_vehicle_takes.go — table `match_vehicle_takes` : LA RESSOURCE VEHICULES DE
// L'EMPRISE (prises, temps a bord, frags apparies), par camp, joueur et famille de vehicule, LUE
// DE L'ARTEFACT DE REJEU (plan `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.2).
//
// # CE QUE LA TABLE PORTE
//
// Deux natures de ligne (`row_kind`) dans UNE passe :
//
//	take    une ligne par (camp, xuid, family) : takes (D2), aboard_ms (D4), episodes, dont
//	        proximity_episodes, et frags = frags de classe engin tombes PENDANT un episode de
//	        ce joueur sur cette famille (D9, le numerateur du rendement) ;
//	match   UNE ligne par passe (camp -1, xuid et family vides), qui existe meme quand la passe
//	        n'a aucune prise : c'est elle qui fait retenir la passe entiere par la vue `_latest`
//	        et qui porte « zero mesure » contre « non mesure ».
//
// Les colonnes de COUVERTURE sont recopiees sur chaque ligne de la passe (modele de
// `pads_confirmed` dans `match_pad_pickups_by_tier`) :
//
//	measured / unmeasured_reason / doc_schema   D8 : artefact sans occupation lue (schema < 67,
//	                                            calque non balaye, pas d'image absent)
//	episodes_read / _unnamed / _no_camp         D10 : episodes lus, et ceux sans xuid (bots) ou
//	                                            sans camp, qui ne font ni prise ni temps
//	frags_read / frags_reason                   les frags de classe engin ont-ils pu etre
//	                                            apparies (origine du document, evenements de mort)
//	frags_total / frags_unmatched               D9 : frags de classe engin du match, et ceux
//	                                            qu'aucun episode de leur tueur ne couvre
//
// # ANTI-ART (ADR 0019/0026/0030)
//
// INSERT purs (`persist.VehicleTakesPersister`), LECTURE VIA `match_vehicle_takes_latest`
// UNIQUEMENT. La table entre dans les garde-rails : tables protegees de
// `sync/no_art_patterns_test.go`, `sync/append_only_state_guard_test.go`, la regexp de
// `duckdb/no_raw_rating_reads_test.go` et le registre de compaction.
//
// # L'UNITE D'ECRITURE EST LA PASSE
//
// Toutes les lignes d'une passe portent le MEME `decode_pass` ; la vue rend la DERNIERE PASSE
// ENTIERE par match, jamais la derniere ligne par cle : une famille ou un joueur que la nouvelle
// passe ne produit plus est RETRACTE.
//
// # MULTI-TITRE
//
// La table existe pour tous les titres ; elle reste VIDE sur un titre qui ne declare pas
// `film.vehicle_usage`. Le branchement se fait sur capability, jamais sur le slug.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "shared_create_vehicle_takes",
		TargetDB: TargetShared,
		Description: "Table append-only match_vehicle_takes (prises, temps a bord et frags apparies " +
			"par camp/joueur/famille de vehicule, plus une ligne match de couverture, lues de " +
			"l artefact range) + decode_pass + index match_id + vue match_vehicle_takes_latest " +
			"(derniere passe ENTIERE par match)",
		ApplySchema: applyMatchVehicleTakes,
	})
}

// applyMatchVehicleTakes cree la sequence, la table, son index et sa vue. Idempotente.
//
// ⚠ TOUT ELARGISSEMENT FUTUR passe par un step au NOM NEUF qui RECREE la vue dans la meme
// transaction de schema : DuckDB FIGE la liste de colonnes d'un `SELECT *` a la creation.
func applyMatchVehicleTakes(db *sql.DB) error {
	return execScript(db, MatchVehicleTakesTableSQL("match_vehicle_takes")+
		MatchVehicleTakesLatestViewSQL("match_vehicle_takes"))
}

// MatchVehicleTakesTableSQL rend le DDL de la table et de son index pour une REFERENCE DE TABLE
// donnee. EXPORTE POUR QUE PERSONNE NE LE RECOPIE.
func MatchVehicleTakesTableSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchVehicleTakes, tableRef, tableRef)
}

// MatchVehicleTakesLatestViewSQL rend le DDL de la vue `_latest`.
func MatchVehicleTakesLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchVehicleTakesLatest, tableRef)
}

// AUCUN commentaire SQL dans le script : le splitter d'`ExecScript` coupe naivement sur `;`
// (piege ADR 0026). Le vocabulaire des colonnes est decrit sur `persist.VehicleTakeRow` et
// `persist.VehicleTakesBatch`.
const ddlMatchVehicleTakes = `
	CREATE SEQUENCE IF NOT EXISTS match_vehicle_takes_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		id                 BIGINT    PRIMARY KEY DEFAULT nextval('match_vehicle_takes_id_seq'),
		match_id           VARCHAR   NOT NULL,
		decode_pass        VARCHAR   NOT NULL,
		written_at         TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		row_kind           VARCHAR   NOT NULL,
		camp               INTEGER   NOT NULL,
		xuid               VARCHAR   NOT NULL,
		family             VARCHAR   NOT NULL,
		takes              INTEGER   NOT NULL,
		aboard_ms          BIGINT    NOT NULL,
		episodes           INTEGER   NOT NULL,
		proximity_episodes INTEGER   NOT NULL,
		frags              INTEGER   NOT NULL,
		measured           BOOLEAN   NOT NULL,
		unmeasured_reason  VARCHAR   NOT NULL,
		doc_schema         INTEGER   NOT NULL,
		episodes_read      INTEGER   NOT NULL,
		episodes_unnamed   INTEGER   NOT NULL,
		episodes_no_camp   INTEGER   NOT NULL,
		frags_read         BOOLEAN   NOT NULL,
		frags_reason       VARCHAR   NOT NULL,
		frags_total        INTEGER   NOT NULL,
		frags_unmatched    INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_vehicle_takes_match
		ON %s(match_id);
`

// ddlMatchVehicleTakesLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026). Patron EXACT de
// `match_pad_pickups_by_tier_latest` : la DERNIERE PASSE ENTIERE par match.
const ddlMatchVehicleTakesLatest = `
	CREATE OR REPLACE VIEW match_vehicle_takes_latest AS
	SELECT * FROM %s AS t
	QUALIFY t.decode_pass = FIRST_VALUE(t.decode_pass) OVER (
		PARTITION BY t.match_id ORDER BY t.written_at DESC, t.id DESC
	);
`
