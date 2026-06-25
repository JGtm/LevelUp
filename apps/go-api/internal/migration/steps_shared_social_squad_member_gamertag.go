package migration

// steps_shared_social_squad_member_gamertag.go — ajoute la colonne gamertag à
// squad_member_history.
//
// Le roster d'escouade est clé xuid (cf. SquadMember), mais l'UI a besoin du
// gamertag pour afficher les membres ET recharger une composition (la page
// Escouade travaille en gamertags). On persiste donc un SNAPSHOT du gamertag au
// moment de l'ajout — c'est le libellé que l'utilisateur a choisi pour sa
// composition. APPEND-ONLY préservé : on ALTER la table _history puis on
// reconstruit la vue _latest (aucun DELETE / UPDATE indexé).
//
// Idempotent + tolérant à l'ordre : si _history n'existe pas encore (migration
// append-only pas encore jouée), on ne fait rien — cette colonne n'est
// pertinente que sur la table déjà créée (ordre canonique garanti par order.go).

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_social_squad_member_gamertag_v1",
		TargetDB:    TargetSharedSocial,
		Description: "squad_member_history : colonne gamertag (snapshot d'affichage du roster) + reconstruction vue _latest",
		ApplySchema: applySquadMemberGamertag,
	})
}

func applySquadMemberGamertag(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "squad_member_history")
	if err != nil {
		return fmt.Errorf("squad_member gamertag: check history: %w", err)
	}
	if !hasHistory {
		return nil // créée par la migration append-only (ordre canonique)
	}
	hasCol, err := columnExists(db, "squad_member_history", "gamertag")
	if err != nil {
		return fmt.Errorf("squad_member gamertag: check column: %w", err)
	}
	if hasCol {
		return nil
	}

	stmts := []string{
		`ALTER TABLE squad_member_history ADD COLUMN gamertag VARCHAR`,
		`CREATE OR REPLACE VIEW squad_member_latest AS
			SELECT id, squad_id, xuid, user_id, gamertag, is_member, joined_at, written_at
			FROM squad_member_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY squad_id, xuid
				ORDER BY written_at DESC, id DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("squad_member gamertag: step (%s): %w", firstWords(s, 3), err)
		}
	}
	return nil
}
