package migration

// steps_shared_pad_tiers.go — table `match_pad_pickups_by_tier` : LES PRISES DE SOCLE
// VENTILEES PAR NIVEAU D ARME, lues du film et persistees au fil de l eau.
//
// # CE QUE LA TABLE PORTE, ET POURQUOI ELLE EXISTE
//
// Le bloc « controle des armes » de la vue match sait deja ranger ses prises en armes de BASE,
// de TERRAIN et de PUISSANCE : il resout le niveau A LA REQUETE, parce qu il a l artefact
// entier sous la main. Les pages d AGREGAT (Sessions, Escouade, Timeseries) ne l ont pas :
// elles lisent `match_usage_players_latest`, ou les prises sont deja comptees PAR ARME et ou
// l identite du socle a disparu. Or le niveau se lit sur LE SOCLE — sa position, croisee a la
// reference des emplacements de la carte — jamais sur le nom de l arme (mesure du 2026-09-14 :
// 70 socles sur 669 portent une arme de role « lourd » sur un ratelier, toutes nominales).
// Cette table est donc ce qui manque : la ventilation par niveau, au grain (match, joueur).
//
// # ELLE NE DEMANDE AUCUNE RECUISSON
//
// Meme doctrine que `match_flag_grabs_net` : la mesure se fait A LA LECTURE d un artefact DEJA
// RANGE (positions de `weaponPads`, canal `loadouts`), croise a la reference versionnee des
// cartes et au registre (map_id, pair_name). Aucun artefact n est re-cuit, et aucun ne doit
// l etre — la cuisson en lot est INTERDITE (bombe RAM, verrou `filmproc.AcquireSolo`).
//
// # ANTI-ART (ADR 0019/0026/0030)
//
// INSERT purs (`persist.PadTiersPersister`), LECTURE VIA `match_pad_pickups_by_tier_latest`
// UNIQUEMENT. La table entre dans les TROIS garde-rails : tables protegees de
// `sync/no_art_patterns_test.go`, `sync/append_only_state_guard_test.go`, et la regexp de
// `duckdb/no_raw_rating_reads_test.go`.
//
// # L UNITE D ECRITURE EST LA PASSE
//
// Toutes les lignes d une passe portent le MEME `decode_pass`, et la vue rend la DERNIERE
// PASSE ENTIERE par match — jamais la derniere ligne par cle. C est ce qui fait qu une
// reference de carte completee, ou une regle de mode corrigee, gagne une generation ENTIERE :
// un niveau que la nouvelle passe ne produit plus est RETRACTE au lieu de survivre a cote des
// nouveaux.
//
// # MULTI-TITRE
//
// La table existe pour tous les titres ; elle reste VIDE sur un titre qui ne declare pas
// `film.weapon_tiers`. Le branchement se fait sur capability, jamais sur le slug.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "shared_create_pad_pickups_by_tier",
		TargetDB: TargetShared,
		Description: "Table append-only match_pad_pickups_by_tier (prises de socle par niveau " +
			"d arme — base/terrain/puissance/non_classe/bonus — et par famille d arme, au grain " +
			"match/joueur, lues de l artefact range) + decode_pass + index match_id + vue " +
			"match_pad_pickups_by_tier_latest (derniere passe ENTIERE par match)",
		ApplySchema: applyMatchPadPickupsByTier,
	})
}

// applyMatchPadPickupsByTier cree la sequence, la table, son index et sa vue. Idempotente.
//
// ⚠ TOUT ELARGISSEMENT FUTUR passe par un step au NOM NEUF qui RECREE la vue dans la meme
// transaction de schema : les migrations sont name-keyed, et DuckDB FIGE la liste de colonnes
// d un `SELECT *` a la creation de la vue.
func applyMatchPadPickupsByTier(db *sql.DB) error {
	return execScript(db, MatchPadTiersTableSQL("match_pad_pickups_by_tier")+
		MatchPadTiersLatestViewSQL("match_pad_pickups_by_tier"))
}

// MatchPadTiersTableSQL rend le DDL de la table et de son index pour une REFERENCE DE TABLE
// donnee. EXPORTE POUR QUE PERSONNE NE LE RECOPIE — une DDL recopiee dans une fixture derive
// de la production sans que rien ne rougisse.
func MatchPadTiersTableSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchPadTiers, tableRef, tableRef)
}

// MatchPadTiersLatestViewSQL rend le DDL de la vue `_latest`. Meme raison d etre exportee.
func MatchPadTiersLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchPadTiersLatest, tableRef)
}

// ddlMatchPadTiers : la table et son unique index. %s (verbatim) = la reference de table.
//
// AUCUN commentaire SQL dans le script : le splitter d `ExecScript` coupe naivement sur `;`,
// et un `;` glisse dans un commentaire `--` casserait la migration (piege ADR 0026).
// L explication vit dans l en-tete de ce fichier ; le vocabulaire des colonnes est decrit sur
// `persist.PadTierRow`.
const ddlMatchPadTiers = `
	CREATE SEQUENCE IF NOT EXISTS match_pad_pickups_by_tier_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		id             BIGINT    PRIMARY KEY DEFAULT nextval('match_pad_pickups_by_tier_id_seq'),
		match_id       VARCHAR   NOT NULL,
		decode_pass    VARCHAR   NOT NULL,
		xuid           VARCHAR   NOT NULL,
		tier           VARCHAR   NOT NULL,
		weapon_family  VARCHAR   NOT NULL,
		pickups        INTEGER   NOT NULL,
		pads_confirmed INTEGER   NOT NULL,
		pads_total     INTEGER   NOT NULL,
		random_starts  BOOLEAN   NOT NULL,
		written_at     TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE INDEX IF NOT EXISTS idx_match_pad_pickups_by_tier_match
		ON %s(match_id);
`

// ddlMatchPadTiersLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026).
//
// Patron EXACT de `match_flag_grabs_net_latest` : la DERNIERE PASSE ENTIERE par match, jamais
// la derniere ligne par cle — sans quoi un niveau calcule sous une reference de carte perimee
// survivrait a cote des niveaux recalcules.
const ddlMatchPadTiersLatest = `
	CREATE OR REPLACE VIEW match_pad_pickups_by_tier_latest AS
	SELECT * FROM %s AS t
	QUALIFY t.decode_pass = FIRST_VALUE(t.decode_pass) OVER (
		PARTITION BY t.match_id ORDER BY t.written_at DESC, t.id DESC
	);
`
