package migration

// steps_shared_flag_grabs_net.go — table `match_flag_grabs_net` : LES PRISES DE DRAPEAU
// BRUTES ET NETTES, par joueur et par match, LUES DU FILM.
//
// ─── POURQUOI UNE TABLE, ET PAS DEUX COLONNES DE PLUS SUR match_objective_stats ───────────
//
// La meme raison que `match_bomb_stats`, et elle n'a pas bouge : `match_objective_stats` est
// alimentee par le SYNC API et sa vue `_latest` ne garde QU'UNE ligne par `(match_id, xuid)`.
// Deux producteurs sur la meme table (l'API et le film) et le dernier ecrit MASQUE les
// colonnes de l'autre — un re-sync API effacerait de la vue les prises nettes. La separation
// n'est pas un rangement, c'est ce qui empeche une perte silencieuse.
//
// ─── LES TROIS COLONNES, ET POURQUOI LA TROISIEME EXISTE ──────────────────────────────────
//
//	flag_grabs_raw     les prises BRUTES lues sur le calque de drapeau de l'artefact
//	                   (`flagCarries[].spans`, etats portes). C'est le compteur officiel tel
//	                   que le FILM le rend — il se compare a `match_objective_stats.flag_grabs`
//	                   sans s'y substituer.
//	flag_grabs_net     les memes prises, JONGLAGE REPLIE (`objectiveevents.NetFlagGrabs`).
//	juggle_window_ms   LA FENETRE SOUS LAQUELLE CETTE LIGNE A ETE CALCULEE. Sans elle,
//	                   « 4 prises nettes » ne veut rien dire, et deux passes cuites sous deux
//	                   fenetres differentes seraient indistinguables dans la meme colonne. Elle
//	                   voyage donc AVEC la mesure, comme la `source` d'un fait date.
//
// ─── ABSENT N'EST PAS ZERO ────────────────────────────────────────────────────────────────
//
// Un match sans film decode, un film qui n'est pas du CTF, un titre qui ne declare pas la
// fenetre : AUCUNE LIGNE. La grandeur se lit alors « non mesuree », jamais « zero prise ».
// Les trois colonnes sont NOT NULL pour cette raison exacte — une ligne qui existe AFFIRME,
// et ce qu'on n'a pas mesure n'a pas de ligne.
//
// ─── FORME APPEND-ONLY (ADR 0026), COMME SES DEUX SOEURS ──────────────────────────────────
//
// Table CREEE DIRECTEMENT append-only (id PK sequence + `written_at` + vue `_latest`) — PAS
// via ApplyAppendOnlyRebuild, qui est la recette de CONVERSION d'une table mutable existante.
// ART-safe par construction (#23645) : ecriture = INSERT pur
// (`persist.FlagGrabsNetPersister`), LECTURE VIA `match_flag_grabs_net_latest` UNIQUEMENT —
// une lecture brute servirait les lignes d'une passe de decodage precedente, voire d'une
// fenetre precedente. Un seul index, `match_id` : le seul acces ponctuel reel est « les prises
// de CE match », et chaque index en plus elargit la surface ART.
//
// ─── CONSEQUENCE ASSUMEE — la table est creee dans le shared de TOUS les titres ────────────
//
// Ce step vit dans le socle, comme ses soeurs issues du film (`match_bomb_stats`,
// `match_kill_events`, `match_weapon_shots`). Sur un titre dont le format de film differe, ou
// qui ne declare pas la fenetre de jonglage, la table existe et reste VIDE. Le branchement
// produit se fait sur capability (`film.flag_grabs_net`), jamais sur le slug.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "shared_create_flag_grabs_net",
		TargetDB: TargetShared,
		Description: "Table append-only match_flag_grabs_net (prises de drapeau brutes et nettes " +
			"par joueur/match, lues du film, avec la fenetre de jonglage appliquee) + index " +
			"match_id + vue match_flag_grabs_net_latest",
		ApplySchema: applyMatchFlagGrabsNet,
	})
}

// applyMatchFlagGrabsNet cree la sequence, la table, son index et sa vue. Idempotente.
//
// ⚠ TOUT ELARGISSEMENT FUTUR passe par un step au NOM NEUF qui RECREE la vue dans la meme
// transaction de schema : les migrations sont name-keyed, et DuckDB FIGE la liste de colonnes
// d'un `SELECT *` a la creation de la vue. Modele :
// `shared_objective_stats_add_stockpile_extraction`.
func applyMatchFlagGrabsNet(db *sql.DB) error {
	return execScript(db, MatchFlagGrabsNetTableSQL("match_flag_grabs_net")+
		MatchFlagGrabsNetLatestViewSQL("match_flag_grabs_net"))
}

// MatchFlagGrabsNetTableSQL rend le DDL de la table et de son index pour une REFERENCE DE
// TABLE donnee.
//
// EXPORTE POUR QUE PERSONNE NE LE RECOPIE — meme raison que `MatchBombStatsTableSQL` : une DDL
// recopiee dans une fixture de test derive de la production sans que rien ne rougisse.
func MatchFlagGrabsNetTableSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchFlagGrabsNet, tableRef, tableRef)
}

// MatchFlagGrabsNetLatestViewSQL rend le DDL de la vue `_latest` pour une REFERENCE DE TABLE
// donnee. Meme raison d etre exportee, meme patron que ses soeurs.
func MatchFlagGrabsNetLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(ddlMatchFlagGrabsNetLatest, tableRef)
}

// ddlMatchFlagGrabsNet : la table et son unique index. %s (verbatim) = la reference de table.
const ddlMatchFlagGrabsNet = `
	CREATE SEQUENCE IF NOT EXISTS match_flag_grabs_net_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		-- identite technique (append-only : PK non naturelle, ADR 0026)
		id               BIGINT  PRIMARY KEY DEFAULT nextval('match_flag_grabs_net_id_seq'),
		match_id         VARCHAR NOT NULL,
		-- xuid en decimal, la meme clef que match_participants.xuid. Un bot n'a pas de XUID :
		-- il n'a donc pas de ligne ici, et son absence n'est pas un zero.
		xuid             VARCHAR NOT NULL,

		-- ── LES DEUX COMPTES ET LEUR REGLE — NOT NULL : une ligne qui existe AFFIRME ────
		flag_grabs_raw   INTEGER NOT NULL,
		flag_grabs_net   INTEGER NOT NULL,
		-- juggle_window_ms : la fenetre sous laquelle flag_grabs_net a ete calcule. Elle
		-- voyage avec la mesure — « 4 prises nettes » ne se lit pas sans elle.
		juggle_window_ms INTEGER NOT NULL,

		-- written_at : l'arbitre de la vue _latest. Toutes les lignes d'une meme passe le
		-- partagent (pose par le persister), pour que la vue retienne une generation entiere.
		written_at       TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE INDEX IF NOT EXISTS idx_match_flag_grabs_net_match
		ON %s(match_id);
`

// ddlMatchFlagGrabsNetLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026).
//
// Patron EXACT de `MatchBombStatsLatestViewSQL` : derniere ligne par `(match_id, xuid)`,
// `written_at DESC` puis `id DESC` pour departager deux ecritures tombees dans la meme
// milliseconde.
const ddlMatchFlagGrabsNetLatest = `
	CREATE OR REPLACE VIEW match_flag_grabs_net_latest AS
	SELECT *
	FROM %s
	QUALIFY ROW_NUMBER() OVER (
		PARTITION BY match_id, xuid
		ORDER BY written_at DESC, id DESC
	) = 1;
`
