package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
)

// sameNameQuery décrit un fichier à indexer dont on cherche la fiche de même nom.
type sameNameQuery struct {
	player string
	stem   string // nom du fichier sans extension
	abs    string // chemin absolu du fichier
	stored string // chemin tel que la base le stocke
	store  MediaPathStore
}

// sameNameRow : la fiche retrouvée.
type sameNameRow struct {
	id       int64
	filePath string
}

// findSameNameRow cherche la fiche du joueur qui représente le même média sous un
// autre format : mêmes nom sans extension et même dossier (celui de l'original,
// sinon celui du fichier de la fiche). Une vidéo convertie dont l'original a disparu
// reste retenue quel que soit le dossier, sa version convertie vivant ailleurs.
//
// Les fiches supprimées sont vues exprès : un fichier re-déposé annule la
// suppression au lieu de créer une seconde fiche. Une fiche qui désigne déjà ce
// fichier (même chemin, comme fichier ou comme original) n'est pas un changement
// de format : la déduplication par chemin de l'appelant la traite.
//
// Une fiche introuvable n'est jamais retenue : la passe de réconciliation a déjà
// repris tout fichier qui la continuait (même empreinte, ou changement de format à
// candidat unique), donc celle qui reste est ambiguë et le fichier devient une
// nouvelle fiche. L'utilisateur la fond ensuite à la main.
func findSameNameRow(ctx context.Context, db *sql.DB, q sameNameQuery) (sameNameRow, bool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, file_path, file_name, source_path, hls_path, status
		FROM media_files
		WHERE player_slug = ? AND file_stem = ?
		ORDER BY id`, q.player, q.stem)
	if err != nil {
		return sameNameRow{}, false, fmt.Errorf("recherche d'une fiche de même nom: %w", err)
	}
	defer rows.Close()
	wantDir := pathKey(filepath.Dir(q.abs))
	for rows.Next() {
		var id int64
		var fp, fn, sp, hp, st sql.NullString
		if err := rows.Scan(&id, &fp, &fn, &sp, &hp, &st); err != nil {
			return sameNameRow{}, false, fmt.Errorf("lecture d'une fiche de même nom: %w", err)
		}
		if st.String == domain.MediaStatusMissing {
			continue
		}
		if stemOf(fn.String) != q.stem || fp.String == q.stored || sp.String == q.stored {
			continue
		}
		ref := sp.String
		if ref == "" {
			ref = fp.String
		}
		if sp.String == "" && hp.String != "" {
			return sameNameRow{id: id, filePath: fp.String}, true, nil
		}
		if pathKey(filepath.Dir(q.store.ToAbs(ref))) == wantDir {
			return sameNameRow{id: id, filePath: fp.String}, true, nil
		}
	}
	return sameNameRow{}, false, rows.Err()
}

// sourceStat rend la taille et la date de modification (en nanosecondes) d'un
// fichier, NULL si le fichier ne se laisse pas lire : la passe de réconciliation
// les complétera au passage suivant.
func sourceStat(ctx context.Context, path string) (size, mtimeNS sql.NullInt64) {
	st, err := os.Stat(path)
	if err != nil {
		slog.WarnContext(ctx, "insertMediaFile: taille et date de l'original illisibles",
			"path", path, "err", err)
		return sql.NullInt64{}, sql.NullInt64{}
	}
	return nullInt(st.Size()), nullInt(st.ModTime().UnixNano())
}
