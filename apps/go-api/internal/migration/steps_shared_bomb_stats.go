package migration

// steps_shared_bomb_stats.go — table `match_bomb_stats` : LES CINQ STATISTIQUES D'OBJECTIF DU
// MODE ASSAUT, par joueur et par match, RECONSTRUITES DU FILM.
//
// ─── POURQUOI UNE TABLE, ET PAS CINQ COLONNES DE PLUS SUR match_objective_stats ───────────
//
// `match_objective_stats` est alimentee par le SYNC API et sa vue `_latest` ne garde QU'UNE
// ligne par `(match_id, xuid)` — la plus recente. Deux producteurs sur la meme table (l'API et
// le film) et le dernier ecrit MASQUE les colonnes de l'autre : un re-sync API effacerait de la
// vue les statistiques venues du film. La separation n'est donc pas une preference de rangement,
// c'est ce qui empeche une perte silencieuse.
//
// ─── D'OU VIENT CHAQUE COLONNE (mesure, jamais deduction) ─────────────────────────────────
//
//	bomb_detonations              statborg `comp 0` canal A des slots de JOUEUR. En Assaut un
//	                              point de mode EST une explosion — gate 4/4 films sur moities
//	                              disjointes. ⚠ CE N'EST PAS `BombStats_BombDetonations` du
//	                              binaire : cette famille-la est de la TELEMETRIE Bond, que le
//	                              film ne replique pas (mesure Ghidra 2026-09-04). Le compteur
//	                              lu ici est le compteur GENERIQUE de points de mode, qui vaut
//	                              numeriquement les explosions parce qu'en Assaut rien d'autre
//	                              ne fait bouger le score.
//	bomb_arms                     JOINTURE (E2) : armement date par l'anneau `ti=12 i14`,
//	                              acteur nomme par le canal des armes tenues (le LACHER est le
//	                              geste de pose). Couverture mesuree 9/13 armements sur 5 films.
//	bomb_grabs                    periodes de portage du canal des armes tenues, famille bombe
//	                              `0x3fee4fcf`.
//	time_as_bomb_carrier_seconds  somme des periodes FERMEES (lacher ou mort) du meme canal.
//	bomb_carriers_killed          kill dont la VICTIME est en periode de portage a l'instant du
//	                              kill.
//
// ─── ABSENT N'EST PAS ZERO — c'est la raison d'etre des colonnes NULLABLE ─────────────────
//
// Le noyau d'extraction (`analysis/replay/bomb_stats.go`) rend des POINTEURS pilotes par des
// temoins de LECTURE : un canal non balaye laisse le champ a `nil`, il ne le met pas a zero. Ce
// schema TIENT cette distinction : les cinq colonnes sont nullable, une valeur non mesuree
// s'ecrit NULL. Ecrire 0 dirait « mesure : rien ne s'est passe » la ou la verite est « on n'a
// pas regarde » — et un agregat sommerait ces faux zeros sans rien signaler.
//
// ─── FORME APPEND-ONLY (ADR 0026), COMME match_objective_stats ────────────────────────────
//
// Table CREEE DIRECTEMENT append-only (id PK sequence + `written_at` + vue `_latest`) — PAS via
// ApplyAppendOnlyRebuild, qui est la recette de CONVERSION d'une table mutable existante.
// ART-safe par construction (#23046) : ecriture = INSERT pur
// (`persist.BombStatsPersister`), LECTURE VIA `match_bomb_stats_latest` UNIQUEMENT — une
// lecture brute servirait les lignes d'une passe de decodage precedente. Un seul index,
// `match_id` : DuckDB est colonnaire, le seul acces ponctuel reel est « les stats de CE match »,
// et chaque index en plus elargit la surface ART.
//
// ─── CONSEQUENCE ASSUMEE — la table est creee dans le shared de TOUS les titres ────────────
//
// Ce step vit dans le socle et non dans `games/halo_infinite/migrations/`, comme ses soeurs
// issues du film (`match_kill_events`, `match_weapon_shots`, `match_weapon_hit_distance`). Sur
// un titre dont le format de film differe (Halo 5) la table existe et reste VIDE : c'est la
// meme consequence, deja assumee et documentee, que pour `match_kill_events`. Le branchement
// produit se fait sur capability (`film.bomb_stats`), jamais sur le slug.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "shared_create_bomb_stats",
		TargetDB: TargetShared,
		Description: "Table append-only match_bomb_stats (5 statistiques d Assaut par joueur/match, " +
			"reconstruites du film) + index match_id + vue match_bomb_stats_latest",
		ApplySchema: applyMatchBombStats,
	})
}

// applyMatchBombStats cree la sequence, la table, son index et sa vue. Idempotente
// (`IF NOT EXISTS` / `OR REPLACE`). La table etant NET-NEUVE, le piege « CREATE TABLE IF NOT
// EXISTS n'ajoute jamais une PK a une table existante » ne s'applique pas ici.
//
// ⚠ TOUT ELARGISSEMENT FUTUR (une 6e statistique, le desamorcage...) passe par un step au NOM
// NEUF, et ce step-la RECREE la vue dans la MEME transaction de schema : les migrations sont
// name-keyed (une base deja migree ne rejoue jamais `shared_create_bomb_stats`), et DuckDB FIGE
// la liste de colonnes d'un `SELECT *` a la creation de la vue. Modele a copier :
// `shared_objective_stats_add_stockpile_extraction`.
func applyMatchBombStats(db *sql.DB) error {
	return execScript(db, MatchBombStatsTableSQL("match_bomb_stats")+
		MatchBombStatsLatestViewSQL("match_bomb_stats"))
}

// MatchBombStatsTableSQL rend le DDL de la table d Assaut et de son index pour une REFERENCE DE
// TABLE donnee.
//
// EXPORTE POUR QUE PERSONNE NE LE RECOPIE. Une DDL recopiee dans une fixture de test derive de
// la production sans que rien ne rougisse — c est le piege le plus cher du depot (une colonne
// ajoutee ici, un test qui continue de passer sur l ancienne forme). Les fixtures d integration
// (`platform/duckdb`) appellent cette fonction avec `shared.match_bomb_stats` ; la migration
// l appelle avec `match_bomb_stats`. UNE seule definition, deux references.
func MatchBombStatsTableSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchBombStats, tableRef, tableRef)
}

// MatchBombStatsLatestViewSQL rend le DDL de la vue `_latest` pour une REFERENCE DE TABLE
// donnee. Meme raison d etre exportee, meme patron que MatchObjectiveStatsLatestViewSQL.
func MatchBombStatsLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchBombStatsLatest, tableRef)
}

// ddlMatchBombStats : la table et son unique index. %s (verbatim) = la reference de table.
const ddlMatchBombStats = `
	CREATE SEQUENCE IF NOT EXISTS match_bomb_stats_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		-- identite technique (append-only : PK non naturelle, ADR 0026)
		id                           BIGINT    PRIMARY KEY DEFAULT nextval('match_bomb_stats_id_seq'),
		match_id                     VARCHAR   NOT NULL,
		-- xuid en decimal, la meme clef que match_participants.xuid. Un bot n'a pas de XUID :
		-- il n'a donc pas de ligne ici, et son absence n'est pas un zero.
		xuid                         VARCHAR   NOT NULL,

		-- ── LES CINQ MESURES — NULL = NON MESURE, jamais « zero mesure » ────────────────
		bomb_detonations             INTEGER,
		bomb_arms                    INTEGER,
		bomb_grabs                   INTEGER,
		time_as_bomb_carrier_seconds DOUBLE,
		bomb_carriers_killed         INTEGER,

		-- written_at : l'arbitre de la vue _latest. Toutes les lignes d'une meme passe le
		-- partagent (pose par le persister), pour que la vue retienne une generation entiere.
		written_at                   TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE INDEX IF NOT EXISTS idx_match_bomb_stats_match
		ON %s(match_id);
`

// ddlMatchBombStatsLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026 — une lecture brute
// sert les lignes d'une passe de decodage precedente).
//
// Patron EXACT de `MatchObjectiveStatsLatestViewSQL` (objective_stats_view.go) : derniere ligne
// par `(match_id, xuid)`, `written_at DESC` puis `id DESC` pour departager deux ecritures tombees
// dans la meme milliseconde. Le DDL n'est PAS partage avec celui de la vue objectifs : ce sont
// deux relations differentes, et le garde-rail archlint qui interdit d'inliner la vue
// `match_objective_stats_latest` ne porte pas sur celle-ci.
const ddlMatchBombStatsLatest = `
	CREATE OR REPLACE VIEW match_bomb_stats_latest AS
	SELECT *
	FROM %s
	QUALIFY ROW_NUMBER() OVER (
		PARTITION BY match_id, xuid
		ORDER BY written_at DESC, id DESC
	) = 1;
`
