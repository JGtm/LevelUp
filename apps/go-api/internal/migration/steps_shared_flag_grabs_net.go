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
// ─── LES CINQ COLONNES DE MESURE, ET POURQUOI CHACUNE EXISTE ──────────────────────────────
//
//	flag_grabs_raw     les prises BRUTES lues sur le calque de drapeau de l'artefact
//	                   (`flagCarries[].spans`, etats portes). C'est ce que LE FILM lit — a ne
//	                   pas confondre avec `match_objective_stats.flag_grabs`, le compteur de
//	                   l'API, qui est une autre chaine et n'entre jamais ici.
//	flag_grabs_net     les memes prises, JONGLAGE REPLIE (`objectiveevents.NetFlagGrabs`).
//	openings           le nombre d'OUVERTURES DE PORTAGE que l'oracle du film a comptees sur
//	                   CE match (`coverage.flagCarries.openings`, `flag_grabs` + `flag_steals`
//	                   fusionnes). C'est le DENOMINATEUR de `flag_grabs_raw` : les pistes ne
//	                   portent que les prises que le pont a su nommer ET situer, et sans ce
//	                   compte « 25 prises brutes » se lirait comme une exhaustivite. Valeur de
//	                   MATCH, donc identique sur toutes les lignes d'une passe — repetee plutot
//	                   que rangee dans une seconde table, parce qu'une table a une colonne pour
//	                   un entier par match couterait une jointure a chaque lecture.
//	juggle_window_ms   LA FENETRE SOUS LAQUELLE CETTE LIGNE A ETE CALCULEE. Sans elle,
//	                   « 4 prises nettes » ne veut rien dire, et deux passes cuites sous deux
//	                   fenetres differentes seraient indistinguables dans la meme colonne.
//	decode_pass        l'identifiant de la PASSE (cf. ci-dessous).
//
// ─── L'UNITE DE GENERATION EST LA PASSE, PAS LA LIGNE — ET C'EST VITAL ICI ─────────────────
//
// La vue retient LA DERNIERE PASSE ENTIERE PAR MATCH (`decode_pass`), sur le modele exact de
// `kill_openings_latest` et de `match_kill_events_latest`. Elle a d'abord arbitre par CLE —
// derniere ligne par `(match_id, xuid)` — et c'etait un DEFAUT, releve en revue :
//
//	LA FENETRE AURAIT FUITE D'UNE PASSE A L'AUTRE. Une re-projection `--force` sous une
//	  NOUVELLE fenetre qui ne retrouve plus un joueur (pont qui ne le nomme plus, decodeur
//	  ameliore qui ecarte ses portages) laissait sa ligne de la passe PRECEDENTE dans la vue,
//	  avec l'ANCIENNE fenetre. Le scope portait alors DEUX fenetres, et les lecteurs qui
//	  exigent une fenetre unique (`sessionusage.ComputeFlagGrabsNet`) cessaient d'en annoncer
//	  AUCUNE — un symptome a l'ecran, dont la cause etait invisible en base.
//	UN JOUEUR RETIRE NE SE RETRACTAIT PAS. Il restait servi a jamais, melange aux lignes de
//	  la passe courante.
//
// La cle fonctionnelle `(match_id, xuid)` reste celle de la JOINTURE — c'est par elle que le
// lecteur rejoint `match_participants` — mais elle n'arbitre plus rien.
//
// ─── LA RETRACTATION EXIGE QUE LA PASSE SUIVANTE ECRIVE AU MOINS UNE LIGNE (assume) ───────
//
// Ce que la vue rend, c'est la derniere passe QUI EXISTE. Une passe qui ne lit aucun portage
// nomme n'ecrit rien du tout — ni lignes ni `decode_pass` neuf — et la vue continue de servir
// la passe precedente entiere. Meme arbitrage que `kill_openings`, et pour la meme raison :
// ecrire une passe vide obligerait a distinguer en base un match sans drapeau lisible d'un
// match jamais lu, au prix d'une ligne sentinelle dont aucun lecteur n'a l'usage.
//
// ─── ABSENT N'EST PAS ZERO, ET UN ZERO MESURE S'ECRIT 0 ───────────────────────────────────
//
// Un match sans film lu, un film qui n'est pas du CTF, un titre qui ne declare pas la
// fenetre : AUCUNE LIGNE. La grandeur se lit alors « non mesuree ».
//
// Mais sur un match QUI EST lu, un joueur du roster qui n'a jamais touche le drapeau a une
// ligne A ZERO — c'est une mesure, et la taire le rendrait indistinguable d'un joueur d'un
// match non mesure. Les cinq colonnes sont donc NOT NULL : une ligne qui existe AFFIRME.
//
// ─── FORME APPEND-ONLY (ADR 0026), CREEE DIRECTEMENT ──────────────────────────────────────
//
// Table CREEE DIRECTEMENT append-only — PAS via ApplyAppendOnlyRebuild, qui est la recette de
// CONVERSION d'une table mutable existante. ART-safe par construction (#23645) : ecriture =
// INSERT pur (`persist.FlagGrabsNetPersister`), LECTURE VIA `match_flag_grabs_net_latest`
// UNIQUEMENT. Un seul index, `match_id` : le seul acces ponctuel reel est « les prises de CE
// match », et chaque index en plus elargit la surface ART. Enrolee dans les DEUX listes
// anti-ART (`sync/append_only_state_guard_test.go`, `duckdb/no_raw_rating_reads_test.go`) —
// recette ADR 0026 etape 5.
//
// ─── LA MIGRATION A ETE MODIFIEE EN PLACE LE 2026-09-13, ET C'EST LICITE ──────────────────
//
// Un step de migration est name-keyed : le modifier apres coup ne rejoue RIEN sur une base qui
// l'a deja applique, ce qui produit d'ordinaire deux schemas divergents. Ici la table N'EXISTE
// NULLE PART — creee le 2026-09-13 sur cette meme branche, jamais deployee, jamais fusionnee.
// Le seul schema existant est celui des bases de test, recreees a chaque run. Meme licence,
// meme justification, que la modification en place de `shared_create_kill_openings`.
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
			"par joueur/match, lues du film, avec les ouvertures de l oracle et la fenetre de " +
			"jonglage appliquee) + decode_pass + index match_id + vue match_flag_grabs_net_latest " +
			"(derniere passe ENTIERE par match)",
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
//
// AUCUN commentaire SQL dans le script : le splitter d `ExecScript` coupe naivement sur `;`,
// et un `;` glisse dans un commentaire `--` casserait la migration (piege ADR 0026).
// L explication vit dans l en-tete de ce fichier.
const ddlMatchFlagGrabsNet = `
	CREATE SEQUENCE IF NOT EXISTS match_flag_grabs_net_id_seq START 1;
	CREATE TABLE IF NOT EXISTS %s (
		id               BIGINT    PRIMARY KEY DEFAULT nextval('match_flag_grabs_net_id_seq'),
		match_id         VARCHAR   NOT NULL,
		decode_pass      VARCHAR   NOT NULL,
		xuid             VARCHAR   NOT NULL,
		flag_grabs_raw   INTEGER   NOT NULL,
		flag_grabs_net   INTEGER   NOT NULL,
		openings         INTEGER   NOT NULL,
		juggle_window_ms INTEGER   NOT NULL,
		written_at       TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE INDEX IF NOT EXISTS idx_match_flag_grabs_net_match
		ON %s(match_id);
`

// ddlMatchFlagGrabsNetLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026).
//
// Patron EXACT de `kill_openings_latest` : la DERNIERE PASSE ENTIERE par match, jamais la
// derniere ligne par cle (cf. l en-tete — la fenetre fuirait d une passe a l autre).
const ddlMatchFlagGrabsNetLatest = `
	CREATE OR REPLACE VIEW match_flag_grabs_net_latest AS
	SELECT * FROM %s AS f
	QUALIFY f.decode_pass = FIRST_VALUE(f.decode_pass) OVER (
		PARTITION BY f.match_id ORDER BY f.written_at DESC, f.id DESC
	);
`
