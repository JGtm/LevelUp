package migration

// steps_shared_usage_summary.go — tables `match_usage_players` et `match_usage_films` : LE
// RÉSUMÉ D'USAGE D'UN MATCH, dérivé de l'artefact de rejeu déjà cuit (chantier
// session-usage, HANDOFF_SESSION_USAGE_BDD_2026-09-04).
//
// ─── POURQUOI DEUX TABLES, ET POURQUOI EN BASE ─────────────────────────────────────────────
//
// La page Sessions agrège par session ce que chaque joueur a fait de son équipement et des
// socles. Ces grandeurs n'existent que dans les artefacts de rejeu (1,8 Mo pièce, ~16 Mo de
// JSON pour une session de neuf matchs) : le résumé est persisté PAR MATCH au sync, quelques
// centaines d'octets par joueur, et la page lit la base.
//
//	match_usage_players  1 ligne par (match, joueur) : tractions de grappin, épisodes de
//	                     camouflage/surbouclier (compte, durée, frags), déploiements par
//	                     famille (JSON — ensemble ouvert), objets lâchés HORS grenades,
//	                     lancers de grenade (produit, jamais affiché sur Sessions),
//	                     prises de socle d'ARME (total + ventilation JSON par famille).
//	match_usage_films    1 ligne par match : l'échelle de temps du film (frame_interval_ms,
//	                     frame_count, duration_ms), les comptes de prises (nommées,
//	                     anonymes, occupations totales), les socles d'arme présents avec
//	                     leurs occupations (JSON) et les occupations de socle de BONUS par
//	                     famille (JSON).
//
// L'EFFECTIF DES CAMPS N'Y EST PAS, À DESSEIN : il vit dans `match_participants` (les parts
// d'équipe et les parités s'y joignent à la lecture) — le dupliquer ferait deux vérités.
//
// ─── LA FRONTIÈRE SOCLE D'ARME / SOCLE DE BONUS ────────────────────────────────────────────
//
// `weaponPads[].weapon` mélange familles d'arme (huit hexa) et noms canoniques de bonus
// (`powerup_camo`, ...). La frontière est `replay.PadWeaponFamilyKey` et elle seule : les
// bonus vont dans `powerup_pickups_json` (anonymes par construction), JAMAIS dans
// `pad_pickups`. Cf. internal/analysis/replay/usage_summary.go.
//
// ─── APPEND-ONLY, MÊME UNITÉ DE GÉNÉRATION QUE match_weapon_hit_distance ───────────────────
//
// PK technique `id`, `written_at`, INSERT purs (ADR 0026/0030 : aucun UPDATE/DELETE/ON
// CONFLICT). L'unité de production est L'ARTEFACT ENTIER : les vues `_latest` retiennent LA
// DERNIÈRE PASSE PAR MATCH (`summary_pass`), jamais la dernière ligne par clé — un `_latest`
// ligne par ligne laisserait survivre les joueurs d'une passe précédente qu'une
// re-projection ne produit plus. Le persister (INSERT-only) vit dans
// internal/persist/usage_summary_persister.go.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_usage_summary_v1",
		TargetDB:    TargetShared,
		Description: "Tables append-only match_usage_players + match_usage_films (résumé d'usage équipement/socles dérivé de l'artefact de rejeu) + vues _latest par passe",
		ApplySchema: applyMatchUsageSummary,
	})
}

// applyMatchUsageSummary crée les deux tables, leurs index et leurs vues. Idempotente. Les
// tables étant NET-NEUVES, le piège « CREATE TABLE IF NOT EXISTS n'ajoute jamais une PK à
// une table existante » ne s'applique pas.
func applyMatchUsageSummary(db *sql.DB) error {
	for _, script := range []string{
		ddlMatchUsagePlayers, ddlMatchUsageFilms,
		// films_latest AVANT players_latest : la seconde se définit sur la première.
		ddlMatchUsageFilmsLatest, ddlMatchUsagePlayersLatest,
	} {
		if err := execScript(db, script); err != nil {
			return err
		}
	}
	return nil
}

// ddlMatchUsagePlayers : le grain (match, joueur). UN SEUL INDEX, même argument que
// match_weapon_shots : DuckDB est colonnaire, le seul accès ponctuel réel est « les lignes
// de CE match » ; chaque index en plus coûte à l'INSERT et élargit la surface ART.
const ddlMatchUsagePlayers = `
	CREATE SEQUENCE IF NOT EXISTS match_usage_players_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_usage_players (
		-- identité technique (append-only : PK non naturelle, ADR 0026)
		id              BIGINT    PRIMARY KEY DEFAULT nextval('match_usage_players_id_seq'),
		match_id        VARCHAR   NOT NULL,
		-- summary_pass : identifiant d'UNE projection d'UN artefact. Unité de génération.
		summary_pass    VARCHAR   NOT NULL,
		-- summary_rev : révision de la projection (replay.UsageSummaryRev). Dit QUELS matchs
		-- re-résumer après un changement de règle, au lieu de tout re-projeter.
		summary_rev     VARCHAR   NOT NULL,
		written_at      TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		xuid            VARCHAR   NOT NULL,

		-- ── ÉQUIPEMENT ─────────────────────────────────────────────────────────────────
		grapple_pulls        INTEGER NOT NULL,
		camo_episodes        INTEGER NOT NULL,
		camo_ms              BIGINT  NOT NULL,
		camo_kills           INTEGER NOT NULL,
		overshield_episodes  INTEGER NOT NULL,
		overshield_ms        BIGINT  NOT NULL,
		overshield_kills     INTEGER NOT NULL,
		-- deployed_json : déploiements par famille du document ({"wall":2,...}). JSON parce
		-- que l'ensemble des familles est ouvert (manifeste du titre) — une colonne par
		-- famille figerait le schéma sur l'état d'un corpus.
		deployed_json        VARCHAR NOT NULL,
		-- dropped_objects : objets lâchés à la mort, HORS familles de grenade (décision
		-- utilisateur 2026-09-04).
		dropped_objects      INTEGER NOT NULL,
		-- grenades_thrown : produit mais JAMAIS affiché sur la page Sessions (même décision).
		grenades_thrown      INTEGER NOT NULL,

		-- ── SOCLES D'ARME ──────────────────────────────────────────────────────────────
		-- pad_pickups : prises NOMMÉES à ce joueur, socles d'ARME SEULEMENT (les bonus
		-- vivent au grain match, powerup_pickups_json — frontière PadWeaponFamilyKey).
		pad_pickups          INTEGER NOT NULL,
		-- pad_pickups_json : la même grandeur ventilée par clé de famille NORMALISÉE
		-- ({"<clé>":n,...}, replay.PadWeaponFamilyKey : huit hexa minuscules sans 0x —
		-- jamais la forme verbatim du document : deux artefacts peuvent écrire la même
		-- famille sous deux conventions, et des clés verbatim la couperaient en deux).
		pad_pickups_json     VARCHAR NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_usage_players_match
		ON match_usage_players(match_id, written_at);
`

// ddlMatchUsageFilms : le grain match — l'échelle de temps du film et les comptes de socle.
const ddlMatchUsageFilms = `
	CREATE SEQUENCE IF NOT EXISTS match_usage_films_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_usage_films (
		id              BIGINT    PRIMARY KEY DEFAULT nextval('match_usage_films_id_seq'),
		match_id        VARCHAR   NOT NULL,
		summary_pass    VARCHAR   NOT NULL,
		summary_rev     VARCHAR   NOT NULL,
		-- artifact_schema : replay.SchemaVersion de l'artefact projeté. Avec summary_rev,
		-- c'est ce qui décide d'un re-résumé après une re-cuisson.
		artifact_schema INTEGER   NOT NULL,
		written_at      TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- ── L'ÉCHELLE DE TEMPS DU FILM ─────────────────────────────────────────────────
		-- frame_interval_ms = 0 : artefact sans échelle (ancien schéma) — les durées (camo_ms,
		-- overshield_ms) valent alors 0 et le lecteur DOIT le voir ici plutôt que les croire.
		frame_interval_ms INTEGER NOT NULL,
		frame_count       INTEGER NOT NULL,
		duration_ms       INTEGER NOT NULL,

		-- ── LES SOCLES ─────────────────────────────────────────────────────────────────
		-- pad_occupancies : TOUTES les occupations achevées (arme + bonus + socle
		-- introuvable). pad_named/pad_unnamed : prises de socle d'ARME nommées/anonymes.
		pad_occupancies   INTEGER NOT NULL,
		pad_named         INTEGER NOT NULL,
		pad_unnamed       INTEGER NOT NULL,
		-- weapon_pads_json : socles d'ARME présents, UN élément par socle du document, dans
		-- son ordre ([{"weapon":"<clé normalisée>","occupations":4,"named":3},...] — même clé
		-- que pad_pickups_json) — deux socles de la même arme restent deux entrées,
		-- l'agrégation appartient au lecteur.
		weapon_pads_json  VARCHAR NOT NULL,
		-- powerup_pickups_json : occupations de socle de BONUS par nom canonique
		-- ({"powerup_camo":10,...}). ANONYMES PAR CONSTRUCTION : un bonus s'identifie par un
		-- nom, pas par une famille d'arme — rien n'en descend jamais sur une ligne de joueur.
		powerup_pickups_json VARCHAR NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_usage_films_match
		ON match_usage_films(match_id, written_at);
`

// ddlMatchUsagePlayersLatest / ddlMatchUsageFilmsLatest : LE SEUL CHEMIN DE LECTURE AUTORISÉ
// (ADR 0026 — une lecture brute sert les lignes d'une passe précédente). Dernière PASSE par
// match, jamais dernière ligne par clé : l'unité de production est l'artefact entier.
//
// LA PASSE COURANTE SE DÉCIDE SUR match_usage_films, PAS SUR LA TABLE JOUEURS ELLE-MÊME
// (revue adversariale 2026-09-04, deux relecteurs indépendants) : le persister écrit UNE
// ligne film par passe, TOUJOURS — y compris pour une passe sans ligne joueur (artefact
// re-cuit appauvri, roster perdu). Une vue joueurs qui partitionnerait sur sa propre table
// continuerait de servir les joueurs de la passe précédente dans ce cas-là, pendant que
// films_latest servirait la nouvelle passe — le mélange exact que ces vues interdisent.
const ddlMatchUsagePlayersLatest = `
	CREATE OR REPLACE VIEW match_usage_players_latest AS
	SELECT p.*
	FROM match_usage_players AS p
	JOIN match_usage_films_latest AS f
	  ON f.match_id = p.match_id AND f.summary_pass = p.summary_pass;
`

const ddlMatchUsageFilmsLatest = `
	CREATE OR REPLACE VIEW match_usage_films_latest AS
	SELECT f.*
	FROM match_usage_films AS f
	QUALIFY f.summary_pass = FIRST_VALUE(f.summary_pass) OVER (
		PARTITION BY f.match_id ORDER BY f.written_at DESC, f.id DESC
	);
`
