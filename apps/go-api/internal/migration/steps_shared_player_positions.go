package migration

// steps_shared_player_positions.go — CREATION de la table des POSITIONS joueurs (v3 film,
// §N de .ai/RESEARCH_THEATER_RE.md). C'est la forme D'ORIGINE ; sa forme COURANTE est celle que
// `steps_shared_player_positions_appendonly.go` lui donne — lire les deux ensemble.
//
// # CE QUE CE FICHIER POSE, ET CE QUI A CHANGE DEPUIS (decision 1 du plan v2, 2026-09-06)
//
// Il cree les colonnes et rien d'autre. Trois affirmations de son en-tete d'origine sont
// devenues FAUSSES le jour ou la table est devenue une PROJECTION DE L'ARTEFACT DE REJEU, et
// les laisser au present aurait fait croire au lecteur que le regime d'ecriture est encore
// celui d'un outil hors ligne :
//
//	LE PRODUCTEUR      les positions venaient de `analysis/positions.DecodeKeyframePositions`,
//	                   appele par `cmd/diag_weapons_v3 -positions -write`. Elles viennent
//	                   maintenant des trajectoires de l'artefact, projetees par
//	                   `sync/replayartifacts/positions.go` a la granularite de ~20 s que la
//	                   table declare (GrainPositionsMS), comme les trois autres derivations
//	                   post-rangement.
//	LE REGIME          l'ecriture etait un DELETE-then-INSERT par match. Elle est desormais
//	                   APPEND-ONLY : `persist.PlayerPositionsPersister` fait des INSERT purs,
//	                   toutes les lignes d'une projection partagent `positions_pass`, et la
//	                   LECTURE passe par `match_player_positions_latest` (regle ART n°2).
//	LE CHEMIN          « ecriture HORS chemin live, pas de pression concurrente ART » : c'est
//	                   l'inverse. L'ecriture vit dans le CYCLE DE SYNC, sous le lease shared —
//	                   c'est-a-dire exactement le regime pour lequel la conversion append-only
//	                   existe.
//
// # CE QUI N'A PAS CHANGE
//
// PAS DE PK CONTRAIGNANTE, et la raison tient toujours : un meme triplet (time_ms, x, y, z)
// peut legitimement se repeter (spawns, points fixes), donc une PK naturelle rejetterait des
// positions valides. C'est pourquoi la vue `_latest` retient LA DERNIERE PASSE par match, et
// non la derniere ligne par cle.
//
// LA TABLE EST MATCH-LEVEL : aucune colonne xuid. Le document, lui, nomme le porteur de chaque
// trajectoire — il sert a poser l'EQUIPE (lue en base par ce xuid, cf. positions.go) puis il est
// jete. La publier changerait la forme d'une table deja lue par la carte de chaleur : hors
// decision 1, consigne en decouverte.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_player_positions_v1",
		TargetDB:    TargetShared,
		Description: "Table match_player_positions (positions keyframe match-level, v3 film)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_player_positions (
					match_id   VARCHAR NOT NULL,
					time_ms    INTEGER,
					x          REAL,
					y          REAL,
					z          REAL,
					team       INTEGER,
					written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
				);
			`)
		},
	})
}
