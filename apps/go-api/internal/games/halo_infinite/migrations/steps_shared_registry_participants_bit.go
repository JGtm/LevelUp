package migrations

// steps_shared_registry_participants_bit.go — shared_registry_participants_bit_v1 : la
// reconstruction de match_registry qui retire les colonnes d'intégrité mortes ET pose le bit
// participants (matchflags.BackfillParticipants) sur chaque match qui a son roster. Mécanique et
// gardes : migration/steps_shared_registry_integrity_columns.go.
//
// POURQUOI CE STEP EST BORNÉ À CE TITRE (`OnlyTitles`). Le bit est écrit à l'insertion par le
// moteur de synchronisation de ce titre (sync.applyCompletionBitsToBatch) et lu par la sélection
// de son backfill ; un titre qui ingère par un autre chemin (Halo 5 : games/halo_5/ingest) ne
// l'écrit ni ne le lit. Sans la borne, le step tournerait aussi sur la base partagée de Halo 5
// (fallback title-owned, cf. title_set.go) et y poserait un bit que rien n'y maintient. Le step
// global shared_registry_drop_integrity_columns_v1, qui le SUIT, retire les colonnes partout
// ailleurs, sans le bit.

import "levelup/go-api/internal/migration"

func stepRegistryParticipantsBit() migration.Migration {
	return migration.Migration{
		Name:     "shared_registry_participants_bit_v1",
		TargetDB: migration.TargetShared,
		Description: "Halo Infinite : reconstruction de match_registry (swap sans UPDATE) qui retire " +
			"participants_loaded, player_count et idx_registry_player_count et pose le bit participants (512) " +
			"sur les matchs qui ont des participants",
		OnlyTitles:  []string{migration.DefaultSlug},
		ApplySchema: migration.ApplyRegistryParticipantsBitRebuild,
	}
}
