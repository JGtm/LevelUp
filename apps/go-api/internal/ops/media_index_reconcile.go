package ops

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
)

// indexScan : ce que la passe d'indexation retient du dossier avant d'insérer.
type indexScan struct {
	// files : fichiers du titre, ceux que réclame un autre titre en moins.
	files          []string
	skippedForeign int
	rec            ReconcileResult
}

// filterForeignMedia écarte les fichiers que réclame un AUTRE titre (préfixe déclaré
// dans son title.toml, routage DEC-8) : sans cela chaque titre indexerait tous les
// clips du dossier partagé, et les clips étrangers resteraient « Sans match » à
// perpétuité.
func filterForeignMedia(files []string, opts MediaIndexOptions) (kept []string, skipped int) {
	prefixes := opts.ForeignFilenamePrefixes
	if prefixes == nil && opts.TitleSlug != "" {
		prefixes = title.DefaultRegistry().ForeignMediaFilenamePrefixes(opts.TitleSlug)
	}
	kept = make([]string, 0, len(files))
	for _, path := range files {
		if matchesForeignPrefix(filepath.Base(path), prefixes) {
			skipped++
			continue
		}
		kept = append(kept, path)
	}
	return kept, skipped
}

// reconcileIndexedFiles écarte les fichiers des autres titres puis aligne les fiches
// du joueur sur le disque (ReconcileMediaWithDisk) : renommages, déplacements,
// conversions, disparitions, selon opts.MissingPolicy. Les empreintes lues rendues
// par la passe servent à l'insertion qui suit, aucun fichier n'est haché deux fois.
//
// L'état du dossier (racine indisponible, disparition massive en attente) est
// enregistré pour le bandeau quand le titre et le joueur sont connus. Une erreur de
// la passe rend l'indexation vaine : le marquage et les empreintes en dépendent.
func reconcileIndexedFiles(ctx context.Context, db *sql.DB, opts MediaIndexOptions, store MediaPathStore, walked []string) (indexScan, error) {
	var sc indexScan
	sc.files, sc.skippedForeign = filterForeignMedia(walked, opts)

	track := opts.TitleSlug != "" && opts.Gamertag != ""
	allowBulk := false
	switch {
	case track && opts.MissingPolicy == MissingMark:
		allowBulk = ConsumeBulkConfirmation(opts.TitleSlug, opts.Gamertag)
	case track && opts.MissingPolicy == MissingRootUnavailable:
		RecordRootState(opts.TitleSlug, opts.Gamertag, RootState{Unavailable: true})
	}

	rec, err := ReconcileMediaWithDisk(ctx, db, ReconcileInput{
		PlayerSlug: opts.Gamertag, Files: sc.files, Store: store, Policy: opts.MissingPolicy,
		SkipFresh: opts.SkipFreshFiles, AllowBulk: allowBulk,
	})
	if err != nil {
		if allowBulk {
			ConfirmBulkMissing(opts.TitleSlug, opts.Gamertag) // la confirmation vaut pour une passe réussie
		}
		slog.ErrorContext(ctx, "IndexMedia: réconciliation avec le disque échouée",
			"player", opts.Gamertag, "err", err)
		return sc, err
	}
	sc.rec = rec
	if track && opts.MissingPolicy == MissingMark {
		RecordRootState(opts.TitleSlug, opts.Gamertag, RootState{BulkPending: rec.BulkPending})
	}
	logReconcile(ctx, opts.Gamertag, rec)
	return sc, nil
}

// logReconcile rend compte de ce que la passe a changé, quand elle a changé quelque chose.
func logReconcile(ctx context.Context, player string, rec ReconcileResult) {
	if rec.Relinked+rec.Converted+rec.MarkedMissing+rec.Restored+rec.Reverted+rec.SourcesCleared == 0 {
		return
	}
	slog.InfoContext(ctx, "IndexMedia: médias alignés sur le disque",
		"player", player, "relinked", rec.Relinked, "converted", rec.Converted,
		"marked_missing", rec.MarkedMissing, "restored", rec.Restored,
		"reverted", rec.Reverted, "sources_cleared", rec.SourcesCleared)
}
