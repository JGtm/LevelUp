// Package ops — seed_demo_media_h5.go : extraction des clips Halo 5 pour la démo.
//
// Différences vs extractDemoMedia (Infinite) :
//   - les captures Halo 5 (« Halo_5_Guardians-YYYY-MM-DD_HHhMM.mp4 ») sont du mp4
//     H.264/AAC mono-piste → WEB-NATIF, servi en DIRECT (pas de flux HLS) ;
//   - l'attribution n'est PAS « même carte » mais l'ASSOCIATION RÉELLE indexée
//     (index-media --title halo_5 a associé chaque clip à son match par timestamp) ;
//   - les matchs ainsi associés sont AJOUTÉS au corpus (selectMediaAnchoredMatchIDs)
//     car ils sont anciens (2018) — hors de la fenêtre « récents » du corpus.
//
// Layout démo : les FICHIERS vont dans le dir média PLAT (data/demo/players/DEMO/
// media — base dir unique servie par ServeMediaFile), tandis que le shared_social
// est title-scopé (lu par resolveDemoPlayer pour le titre h5).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// numericMediaID dérive un ID NUMÉRIQUE stable (string d'un int64 positif) du nom de
// fichier — requis car media_match_associations.media_file_id est BIGINT (insertDemoMediaRow
// CAST l'ID). FNV-1a 64 bits masqué positif : déterministe (reseed idempotent), collision
// improbable sur la poignée de clips démo.
func numericMediaID(name string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return strconv.FormatUint(h.Sum64()&0x7fffffffffffffff, 10)
}

// halo5CapturePrefix : préfixe des captures Halo 5 (Windows Game Bar).
const halo5CapturePrefix = "Halo_5_Guardians"

// h5CaptureTimezone : fuseau d'interprétation des horodatages de noms de captures h5
// (heure locale Windows Game Bar). Aligné sur config.defaultUserTimezone.
const h5CaptureTimezone = "Europe/Paris"

// parseH5CaptureStart extrait l'instant de capture du nom de fichier h5
// (« Halo_5_Guardians-YYYY-MM-DD_HHhMM ») via le parseur partagé. NullTime invalide
// si non parsable.
func parseH5CaptureStart(name string) sql.NullTime {
	loc, err := time.LoadLocation(h5CaptureTimezone)
	if err != nil {
		loc = time.UTC
	}
	if t := parseCaptureTimeFromFilename(name, loc); t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{}
}

// h5MediaAssocRow : un clip h5 indexé + son match associé (prod shared_social h5).
type h5MediaAssocRow struct {
	FileName  string // ex: "Halo_5_Guardians-2018-08-08_22h33.mp4"
	FilePath  string // chemin source ABSOLU du mp4 (media_files.file_path)
	ThumbPath string // chemin source ABSOLU de la miniature (.webp) ou ""
	MatchID   string // match associé (média_match_associations)
}

// selectMediaAnchoredMatchIDs lit, dans le shared_social SOURCE (prod) du titre, les
// match_ids associés aux clips Halo 5 — pour les AJOUTER au corpus démo (sinon ces
// matchs anciens seraient hors fenêtre et les clips n'auraient aucun match à pointer).
// Retourne nil (sans erreur) si pas de shared_social source / pas de clips.
func selectMediaAnchoredMatchIDs(ctx context.Context, srcSocialDB string, maxMedia int) ([]string, error) {
	if !fileExists(srcSocialDB) {
		return nil, nil
	}
	rows, err := loadH5MediaAssociations(ctx, srcSocialDB, maxMedia)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for _, r := range rows {
		if _, ok := seen[r.MatchID]; ok {
			continue
		}
		seen[r.MatchID] = struct{}{}
		out = append(out, r.MatchID)
	}
	return out, nil
}

// loadH5MediaAssociations lit les clips h5 indexés (préfixe Halo_5_Guardians) + leur
// match associé. Limité à maxMedia (les plus récents par capture_start). Best-effort :
// table absente / schéma legacy → liste vide.
func loadH5MediaAssociations(ctx context.Context, srcSocialDB string, maxMedia int) ([]h5MediaAssocRow, error) {
	db, err := sql.Open("duckdb", srcSocialDB+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared_social source: %w", err)
	}
	defer db.Close()

	// media_match_associations_latest = dernières associations ACTIVES (la vue filtre
	// déjà is_active) → pas de colonne is_active à filtrer ici.
	q := fmt.Sprintf(`
		SELECT mf.file_name, mf.file_path, COALESCE(mf.thumbnail_path, ''), mma.match_id
		FROM media_files mf
		JOIN media_match_associations_latest mma ON mma.media_file_id = mf.id
		WHERE mf.file_name LIKE '%s%%'
		ORDER BY mf.capture_start_utc DESC NULLS LAST, mf.file_name
		LIMIT %d`, halo5CapturePrefix, maxMedia)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		// shared_social legacy (pas de _latest) ou pas de média → best-effort vide.
		slog.WarnContext(ctx, "seed-demo h5 media: lecture associations partielle", "err", err)
		return nil, nil
	}
	defer rows.Close()
	var out []h5MediaAssocRow
	for rows.Next() {
		var r h5MediaAssocRow
		if err := rows.Scan(&r.FileName, &r.FilePath, &r.ThumbPath, &r.MatchID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// extractDemoMediaH5 copie les clips Halo 5 (mp4 servi direct + miniature) vers le dir
// média PLAT démo et insère les media_files (schéma canonique shared_social) + leur
// association au match ancré. Seuls les clips dont le match associé est dans le corpus
// (matchIDs) sont retenus. Retourne le nombre de clips importés.
func extractDemoMediaH5(
	ctx context.Context,
	srcSocialDB, outSocialDB, outMediaDir string,
	matchIDs []string,
	playerSlug string,
	maxMedia int,
) (int, error) {
	if !fileExists(srcSocialDB) {
		slog.InfoContext(ctx, "seed-demo h5 media: shared_social source absent — pas de clip", "src", srcSocialDB)
		return 0, nil
	}
	if err := os.MkdirAll(outMediaDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir media: %w", err)
	}
	// shared_social démo fresh (idempotence reseed). Migrations PAR TITRE (RunForTitleDB) :
	// la racine create_base_shared_social_schema (table media_files) est title-enregistrée,
	// pas globale → RunForDB ne la crée pas. Schéma title-agnostique → DefaultSlug.
	_ = os.Remove(outSocialDB)
	_ = os.Remove(outSocialDB + ".wal")
	if err := applyTitleMigrationsOnPath(outSocialDB, titlePkg.DefaultSlug, migration.TargetSharedSocial); err != nil {
		return 0, fmt.Errorf("migrations shared_social démo h5: %w", err)
	}

	assocs, err := loadH5MediaAssociations(ctx, srcSocialDB, maxMedia*3) // marge : on filtre ensuite par corpus
	if err != nil {
		return 0, fmt.Errorf("load h5 media associations: %w", err)
	}
	corpus := map[string]struct{}{}
	for _, id := range matchIDs {
		corpus[id] = struct{}{}
	}

	db, err := sql.Open("duckdb", outSocialDB)
	if err != nil {
		return 0, fmt.Errorf("open out shared_social: %w", err)
	}
	defer db.Close()

	imported := 0
	for _, a := range assocs {
		if imported >= maxMedia {
			break
		}
		if _, ok := corpus[a.MatchID]; !ok {
			continue // match non seedé → clip sans cible dans la démo
		}
		stem := strings.TrimSuffix(a.FileName, filepath.Ext(a.FileName))
		ext := filepath.Ext(a.FileName)
		if ext == "" {
			ext = extMP4
		}
		// Copier le mp4 (servi DIRECT) vers le dir plat sous son nom de fichier.
		if !fileExists(a.FilePath) {
			slog.WarnContext(ctx, "seed-demo h5 media: source mp4 absente, skip", "path", a.FilePath)
			continue
		}
		if err := copyFile(a.FilePath, filepath.Join(outMediaDir, a.FileName)); err != nil {
			slog.WarnContext(ctx, "seed-demo h5 media: copie mp4 échouée", "name", a.FileName, "err", err)
			continue
		}
		thumbnail := ""
		if a.ThumbPath != "" && fileExists(a.ThumbPath) {
			tn := stem + ".webp"
			if err := copyFile(a.ThumbPath, filepath.Join(outMediaDir, tn)); err == nil {
				thumbnail = tn
			}
		}
		captureStart := parseH5CaptureStart(a.FileName)
		row := demoMediaRow{
			// ID NUMÉRIQUE stable : media_match_associations(_history).media_file_id est
			// BIGINT (insertDemoMediaRow CAST l'ID en BIGINT). Un ID = nom de fichier
			// échouerait le CAST. Le nom lisible reste dans file_stem/file_name.
			ID:            numericMediaID(a.FileName),
			PlayerSlug:    playerSlug,
			FilePath:      a.FileName, // relatif = nom de fichier (servi direct contre le base dir)
			FileName:      stem,
			Kind:          mediaKindVideo,
			FileStem:      stem,
			FileExt:       ext,
			ThumbnailPath: thumbnail,
			MTime:         captureOrNow(captureStart),
			CaptureStart:  captureStart,
			MatchID:       a.MatchID,
		}
		if err := insertDemoMediaRow(ctx, db, row); err != nil {
			slog.WarnContext(ctx, "seed-demo h5 media: insert échouée", "name", a.FileName, "err", err)
			continue
		}
		imported++
		slog.InfoContext(ctx, "seed-demo h5 media: clip copié", "name", a.FileName, "match", a.MatchID)
	}
	if imported == 0 {
		slog.WarnContext(ctx, "seed-demo h5 media: aucun clip dont le match associé est dans le corpus")
	}
	return imported, nil
}
