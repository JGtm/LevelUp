package migration

// steps_shared_social_squad_xuid.go — re-key de squad_member par xuid.
//
// Contexte (PLAN_COACH_V3_GENERATION § Identité d'escouade, Phase C) :
// l'escouade Prestige doit pouvoir inclure des amis QUI NE SONT PAS utilisateurs
// de l'app (présents uniquement comme xuid dans shared.match_participants) et la
// progression d'un défi d'escouade se mesure sur match_participants (clé xuid).
// Or squad_member était clé (squad_id, user_id) où user_id = player_slug — donc
// inexploitable pour ces deux besoins.
//
// Cette migration re-clé squad_member sur xuid (membre universel) + conserve
// user_id (player_slug) OPTIONNEL, renseigné quand le membre est utilisateur de
// l'app → lui donne accès aux objectifs/arcs (règle « membre-user, sans
// consentement »).
//
// DROP sûr : aucun endpoint HTTP de création d'escouade n'a jamais été livré
// (cf. server.go : seules les routes squad_challenge existent), donc squad_member
// est prouvablement vide en prod comme en dev. On recrée à neuf plutôt que
// d'ALTER (DuckDB ne permet pas de redéfinir une PRIMARY KEY in place).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "rekey_squad_member_xuid",
		TargetDB:    TargetSharedSocial,
		Description: "Re-key squad_member par xuid (membre universel) + user_id slug optionnel (accès app)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP TABLE IF EXISTS squad_member;
				CREATE TABLE IF NOT EXISTS squad_member (
					squad_id   VARCHAR NOT NULL,
					xuid       VARCHAR NOT NULL,
					user_id    VARCHAR,
					joined_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (squad_id, xuid)
				);
				CREATE INDEX IF NOT EXISTS idx_sm_xuid ON squad_member(xuid);
				CREATE INDEX IF NOT EXISTS idx_sm_user ON squad_member(user_id);
			`)
		},
	})
}
