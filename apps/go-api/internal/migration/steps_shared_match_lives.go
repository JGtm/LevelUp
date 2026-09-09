// Package migration — steps_shared_match_lives.go : les DEUX tables des faits d'isolement,
// ecrites AU SYNC par le collecteur de kills (plan Tactique, phase 7C, 2026-09-07).
//
// ─── POURQUOI AU SYNC, ET PAS A LA CUISSON ───────────────────────────────────────────────
//
// Decision utilisateur du 2026-09-07 : « les donnees d'un match en base sont completes au sync ;
// seul le rejeu peut attendre la cuisson ». Une version precedente faisait dire au FILM, a la
// lecture, qui etait mort a l'instant d'une mort — elle le deduisait de la chronologie d'un
// artefact de rejeu. Deux defauts : le film ne sait pas dire qui est mort (une vie nommee par
// FERMETURE DE SLOT n'est pas une vie close par une mort, cf. `replay/owners.go`), et un fait de
// base se trouvait dependre du calendrier de cuisson des artefacts.
//
// Le collecteur, lui, scanne DEJA tout ce qu'il faut dans la meme passe
// (`killcollector/positions.go:buildPositionRows`) : positions bipeds, fil des morts, index de
// joueur, pont slot->xuid. Ces deux tables sont une SECONDE PROJECTION de ce travail — aucun
// decodage nouveau.
//
// ─── APPEND-ONLY, ET LA PASSE COMME UNITE DE GENERATION (ADR 0026/0030) ──────────────────
//
// PK technique `id`, `written_at`, INSERT purs. Les deux vues `_latest` retiennent LA DERNIERE
// PASSE PAR MATCH (`decode_pass`), jamais la derniere ligne par cle — meme doctrine que
// `match_kill_events_latest`, et pour la meme raison : l'unite de production est le MATCH ENTIER.
// Une passe B qui rendrait 95 vies la ou A en rendait 99 laisserait sinon survivre 4 vies de A,
// et le contexte d'une mort melangerait deux decodages.
package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_lives_v1",
		TargetDB:    TargetShared,
		Description: "Tables append-only match_lives (vies nommees du film) et match_death_context (voisinage a l instant d une mort) + vues _latest par passe de decodage — faits d isolement au sync, plan Tactique 7C",
		ApplySchema: applyMatchLives,
	})
}

// applyMatchLives cree les deux tables, leurs index et leurs vues. Idempotente
// (`IF NOT EXISTS` / `OR REPLACE`) : les tables etant NET-NEUVES, le piege « CREATE TABLE IF NOT
// EXISTS n'ajoute jamais une PK a une table existante » ne s'applique pas ici.
//
// Relocation : ces steps appartiennent fonctionnellement a Halo Infinite (les films Halo 5 ont un
// autre format) et rejoindront `internal/games/halo_infinite/migrations/` avec le lot de la voie
// B (ADR 0025), exactement comme `shared_match_kill_events_v1`. Consequence assumee : les tables
// sont creees VIDES dans le shared des autres titres.
func applyMatchLives(db *sql.DB) error {
	for _, ddl := range []string{
		ddlMatchLives, ddlMatchLivesLatest,
		ddlMatchDeathContext, ddlMatchDeathContextLatest,
	} {
		if err := execScript(db, ddl); err != nil {
			return err
		}
	}
	return nil
}

// ddlMatchLives : une ligne par VIE NOMMEE du film.
//
// ─── DEUX COLONNES POUR DEUX QUESTIONS ORTHOGONALES ──────────────────────────────────────
//
// `end_cause` dit COMMENT la vie s'est terminee ; `named_by` dit COMMENT ON SAIT A QUI elle
// appartient. Les fondre en une seule colonne — ce que le plan ecrivait d'abord — obligeait a
// choisir entre nommer un survivant et dire qu'il a survecu, et le choix fait le comptait MORT.
// C'est exactement le defaut qui a coute la premiere lecture d'isolement (P0 de la revue ronde 2,
// 2026-09-07). Verifie sur pieces : seuls deux sites du paquet `replay` nomment une vie
// (`lives.go`, le fil des morts ; `owners.go`, la fermeture de slot).
//
//	end_cause   'death'     le fil des morts apparie la fin de la vie. LA SEULE QUI DISE QUE
//	                        LE JOUEUR EST MORT.
//	            'film_end'  la replication du slot s'arrete et ne reprend jamais : le SURVIVANT.
//	            'cut'       un trou de plus de 5 s qu'aucune mort n'explique : typiquement un
//	                        embarquement en vehicule, ou le biped cesse d'etre replique alors
//	                        que le joueur est bien vivant.
//	named_by    'death'     nommee par LECTURE du fil des morts.
//	            'closure'   nommee par DEDUCTION (fermeture de slot) — jamais une mort.
//	            'biped_creation'          nommee par LECTURE du record de creation du bipede
//	                                      (lien direct corps -> joueur, registre d'identite E2).
//	            'biped_creation_propagee' le meme record, propage a un autre sejour du meme corps.
//	            'elimination'             DEDUCTION : elimination sur le roster (registre P2).
//	            'exclusion_temporelle'    DEDUCTION : exclusion sur l'intervalle de vie (P2-bis).
//
// UN SEUL INDEX, celui qui sert la vue `_latest` et la lecture par match. DuckDB est colonnaire :
// un index ART ne sert que les acces ponctuels, et chaque index elargit la surface ART #23046 le
// jour ou quelqu'un ecrirait un DELETE.
const ddlMatchLives = `
	CREATE SEQUENCE IF NOT EXISTS match_lives_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_lives (
		id           BIGINT PRIMARY KEY DEFAULT nextval('match_lives_id_seq'),
		match_id     VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe de decodage d UN film. Unite de generation :
		-- la vue _latest retient une passe ENTIERE, jamais un melange.
		decode_pass  VARCHAR   NOT NULL,
		-- decoder_rev : version du decodeur. Sert a savoir QUELS matchs redecoder apres un
		-- changement, au lieu de tout redecoder.
		decoder_rev  VARCHAR   NOT NULL,
		written_at   TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- xuid : jamais NULL. Une vie anonyme n est PAS ecrite — la publier sous un xuid nul
		-- fabriquerait un joueur fantome auquel chaque film ajouterait des vies.
		xuid         VARCHAR   NOT NULL,
		-- start_ms / end_ms : horloge du MATCH, la meme que match_kill_events.time_ms. C est
		-- ce qui rend les deux tables joignables ; publier l horloge du FILM produirait un
		-- decalage CONSTANT, donc invisible a l oeil et fatal a toute jointure.
		start_ms     BIGINT    NOT NULL,
		end_ms       BIGINT    NOT NULL,
		end_cause    VARCHAR   NOT NULL,
		named_by     VARCHAR   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_lives_lookup
		ON match_lives(match_id, xuid, start_ms, written_at);
`

const ddlMatchLivesLatest = `
	CREATE OR REPLACE VIEW match_lives_latest AS
	SELECT l.*
	FROM match_lives AS l
	QUALIFY l.decode_pass = FIRST_VALUE(l.decode_pass) OVER (
		PARTITION BY l.match_id ORDER BY l.written_at DESC, l.id DESC
	);
`

// ddlMatchDeathContext : une ligne par MORT DU JOURNAL, disant qui l entourait.
//
// ─── LA CLE DE JOINTURE EST CELLE DU JOURNAL ─────────────────────────────────────────────
//
// (`match_id`, `victim_xuid`, `time_ms`) est la cle de `match_kill_events`. La lecture
// « ou je meurs isole » joint les deux : le journal dit OU et QUAND on meurt, cette table dit
// QUI etait la. Une cle differente aurait oblige a rapprocher deux horloges — et un rapprochement
// tolerant transforme un defaut de calage en resultat plausible.
//
// ─── LES QUATRE ETATS D UN COEQUIPIER, ET POURQUOI ILS SONT COMPTES A PART ────────────────
//
//	visible          une position repliquee dans la DERNIERE SECONDE avant la mort. C est le
//	                 seul etat qui autorise une DISTANCE : les autres n ont pas de position
//	                 credible a cet instant.
//	waiting          sa derniere mort au journal precede l instant, et aucune position depuis :
//	                 il attend sa reapparition. Il ne peut pas accompagner.
//	out_of_sight     ni en attente, ni parti, mais sans position recente — typiquement un
//	                 vehicule non replique. IL EST VIVANT, et c est pour cela qu il ne se
//	                 confond pas avec `waiting` : le compter mort ferait sortir « equipe a
//	                 terre » une mort survenue a trois metres d un coequipier en Warthog.
//	left             son depart figure en base avant l instant. Il n est plus dans la partie.
//
// `nearest_teammate_m` est NULL quand AUCUN coequipier n est visible : c est une absence de
// mesure, jamais une distance infinie ni un zero. Distance 2D HORIZONTALE (le Z est ignore, comme
// dans toutes les lectures de l onglet, qui sont des vues du dessus) en METRES MONDE.
const ddlMatchDeathContext = `
	CREATE SEQUENCE IF NOT EXISTS match_death_context_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_death_context (
		id                     BIGINT PRIMARY KEY DEFAULT nextval('match_death_context_id_seq'),
		match_id               VARCHAR   NOT NULL,
		decode_pass            VARCHAR   NOT NULL,
		decoder_rev            VARCHAR   NOT NULL,
		written_at             TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- (match_id, victim_xuid, time_ms) : la cle de match_kill_events.
		victim_xuid            VARCHAR   NOT NULL,
		time_ms                BIGINT    NOT NULL,

		-- nearest_teammate_m : NULL = aucun coequipier VISIBLE. Une absence de mesure, jamais
		-- une distance infinie.
		nearest_teammate_m     DOUBLE,
		teammates_visible      INTEGER   NOT NULL,
		teammates_waiting      INTEGER   NOT NULL,
		teammates_out_of_sight INTEGER   NOT NULL,
		teammates_left         INTEGER   NOT NULL,
		-- teammates_total : le denominateur, pour que la somme des quatre etats se verifie.
		teammates_total        INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_death_context_lookup
		ON match_death_context(match_id, victim_xuid, time_ms, written_at);
`

const ddlMatchDeathContextLatest = `
	CREATE OR REPLACE VIEW match_death_context_latest AS
	SELECT c.*
	FROM match_death_context AS c
	QUALIFY c.decode_pass = FIRST_VALUE(c.decode_pass) OVER (
		PARTITION BY c.match_id ORDER BY c.written_at DESC, c.id DESC
	);
`
