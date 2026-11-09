package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

func buildMediaItems(rows []domain.MediaFileRow, currentPlayerSlug string) []domain.MediaItem {
	items := make([]domain.MediaItem, 0, len(rows))
	for _, r := range rows {
		basename := r.FileName
		if basename == "" {
			basename = filepath.Base(r.FilePath)
		}
		// Section = "mine" si le média appartient au joueur courant, sinon "teammate".
		// Si player_slug absent (legacy), fallback "mine" (ancienne sémantique).
		section := "mine"
		if r.PlayerSlug != nil && currentPlayerSlug != "" && *r.PlayerSlug != currentPlayerSlug {
			section = "teammate"
		}
		items = append(items, domain.MediaItem{
			Basename:       basename,
			FilePath:       r.FilePath,
			Kind:           r.Kind,
			ThumbnailPath:  r.ThumbnailPath,
			CaptureEndUTC:  r.CaptureEndUTC,
			MatchID:        r.MatchID,
			MatchStartTime: r.MatchStartTime,
			MapName:        r.MapName,
			ModeName:       r.ModeName,
			OwnerGamertag:  r.PlayerSlug,
			Section:        section,
			Liked:          r.Liked,
			LikeCount:      boolToLikeCount(r.Liked),
		})
	}
	return items
}

func (s *MediaService) resolveAvailableFilters(
	ctx context.Context,
	filters domain.MediaFilters,
	files []domain.MediaFileRow,
) domain.MediaFilterOptions {
	fallback := buildMediaFilterOptionsFromRows(files)

	options, err := s.repo.LoadMediaFilterOptions(ctx, filters)
	if err != nil {
		slog.WarnContext(ctx, "media: options de filtres indisponibles", "err", err)
		return fallback
	}

	if len(options.Maps) == 0 {
		options.Maps = fallback.Maps
	}
	if len(options.Modes) == 0 {
		options.Modes = fallback.Modes
	}
	// Init [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe
	// le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if options.Playlists == nil {
		options.Playlists = []domain.LabelValue{}
	}
	if options.Maps == nil {
		options.Maps = []domain.LabelValue{}
	}
	if options.Modes == nil {
		options.Modes = []domain.LabelValue{}
	}
	return options
}

// buildMediaFilterOptionsFromRows reconstruit les options de carte et de mode depuis les médias chargés,
// quand le dépôt ne sait pas les calculer. Le libellé est dans la langue de la requête (celui des lignes) ;
// la valeur est la clé indépendante de la langue que compare le filtre du dépôt, la même que celle des
// options nominales (LoadMediaFilterOptions) : l'identifiant de la carte, la catégorie ou le libellé du mode
// en langue par défaut. Sans clé (média sans correspondance), la valeur est le libellé.
func buildMediaFilterOptionsFromRows(rows []domain.MediaFileRow) domain.MediaFilterOptions {
	return domain.MediaFilterOptions{
		Maps: collectMediaLabelValues(rows,
			func(row domain.MediaFileRow) *string { return row.MapName },
			func(row domain.MediaFileRow) *string { return row.MapID }),
		Modes: collectMediaLabelValues(rows,
			func(row domain.MediaFileRow) *string { return row.ModeName },
			func(row domain.MediaFileRow) *string { return row.ModeFilterKey }),
	}
}

// collectMediaLabelValues dédoublonne par valeur (la clé quand elle existe, sinon le libellé) et trie par
// libellé affiché.
func collectMediaLabelValues(
	rows []domain.MediaFileRow,
	label func(domain.MediaFileRow) *string,
	key func(domain.MediaFileRow) *string,
) []domain.LabelValue {
	seen := make(map[string]struct{})
	options := make([]domain.LabelValue, 0)
	for _, row := range rows {
		labelPtr := label(row)
		if labelPtr == nil {
			continue
		}
		text := strings.TrimSpace(*labelPtr)
		if text == "" {
			continue
		}
		value := text
		if keyPtr := key(row); keyPtr != nil {
			if k := strings.TrimSpace(*keyPtr); k != "" {
				value = k
			}
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		options = append(options, domain.LabelValue{Label: text, Value: value})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

func boolToLikeCount(liked bool) int {
	if liked {
		return 1
	}
	return 0
}

// safeDestPath retourne un chemin de destination sûr pour un upload.
// Si un fichier du même nom existe déjà, ajoute un suffixe timestamp
// pour éviter les collisions silencieuses lors d'uploads simultanés.
func safeDestPath(dir, originalName string) string {
	base := filepath.Base(originalName)
	dest := filepath.Join(dir, base)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	ts := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(dir, stem+"_"+ts+ext)
}
