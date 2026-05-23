// Package persist — metadata_persister.go : écriture INSERT-only atomique du
// sous-batch MetadataBatch dans metadata.duckdb.
//
// Seule table couverte pour l'instant : `mode_name_tr` (traductions des noms
// de modes/playlists EN→FR/autres). C'est la seule donnée du flux sync de
// matchs qui touche metadata.duckdb — les autres tables metadata (medal_*,
// citation_mappings, mode_pair_overrides, weapon_labels, assists_model_coefs,
// etc.) sont peuplées par des CLI dédiés (cmd/seed-*) ou des migrations
// initiales, pas par le sync.
//
// Le batch metadata est rare (seulement quand un nouveau mode_en non
// traduit apparaît dans le payload), donc le sous-batch est nullable.

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// MetadataPersister écrit le sous-batch MetadataBatch dans metadata.duckdb.
type MetadataPersister struct {
	db txBeginner
}

// NewMetadataPersister construit un persister sur la connexion metadata.duckdb
// (avec write lease actif côté caller).
func NewMetadataPersister(db txBeginner) *MetadataPersister {
	return &MetadataPersister{db: db}
}

// Persist écrit le batch.Metadata en 1 transaction.
//
// Cas particuliers :
//   - batch == nil                      → error.
//   - batch.Metadata == nil             → no-op (cas nominal).
//   - aucune translation à écrire       → no-op.
//   - PK (mode_en, lang) déjà existante → INSERT OR IGNORE skip silently
//     (préserve les traductions existantes : l'édition manuelle n'est pas
//     écrasée par le flux sync automatique).
func (p *MetadataPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: MetadataPersister.Persist: batch nil")
	}
	if batch.Metadata == nil {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx metadata: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := persistModeNameTranslations(ctx, tx, batch.Metadata.ModeNameTranslations); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit metadata: %w", err)
	}
	return nil
}

func persistModeNameTranslations(ctx context.Context, tx *sql.Tx, rows []ModeNameTranslationInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, t := range rows {
		// INSERT OR IGNORE : préserve les traductions existantes (édition
		// manuelle ou seed initial). Cf. pattern utilisé par
		// applyModeNameTranslations dans steps_metadata.go.
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO mode_name_tr (mode_en, lang, name)
			VALUES (?, ?, ?)`,
			t.ModeEN, t.Lang, t.Name,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT mode_name_tr %s/%s: %w", t.ModeEN, t.Lang, err)
		}
	}
	return nil
}
