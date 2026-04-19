// Package analysis — media.go : helpers purs pour les médias.
package analysis

import (
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
)

const (
	mediaKindClip       = "clip"
	mediaKindVideo      = "video"
	mediaKindScreenshot = "screenshot"
	mediaKindImage      = "image"

	mediaSectionMine       = "mine"
	mediaSectionUnassigned = "unassigned"
)

// BuildMediaItems transforme les lignes DuckDB en items prêts pour l'UI.
func BuildMediaItems(rows []domain.MediaFileRow, ownerGamertag string) []domain.MediaItem {
	if len(rows) == 0 {
		return nil
	}

	owner := strings.TrimSpace(ownerGamertag)
	var ownerPtr *string
	if owner != "" {
		ownerPtr = &owner
	}

	items := make([]domain.MediaItem, 0, len(rows))
	for _, row := range rows {
		basename := strings.TrimSpace(row.FileName)
		if basename == "" {
			basename = filepath.Base(row.FilePath)
		}
		if basename == "" || basename == "." {
			continue
		}

		items = append(items, domain.MediaItem{
			Basename:       basename,
			FilePath:       row.FilePath,
			Kind:           normalizeMediaKind(row.Kind),
			ThumbnailPath:  row.ThumbnailPath,
			CaptureEndUTC:  row.CaptureEndUTC,
			MatchID:        row.MatchID,
			MatchStartTime: row.MatchStartTime,
			Section:        classifyMediaSection(row.MatchID),
			OwnerGamertag:  ownerPtr,
			MapName:        row.MapName,
			ModeName:       row.ModeName,
			Liked:          row.Liked,
			LikeCount:      mediaLikeCount(row.Liked),
		})
	}

	if len(items) == 0 {
		return nil
	}
	return items
}

// SummarizeMediaSections calcule les totaux par section à partir des lignes brutes.
func SummarizeMediaSections(rows []domain.MediaFileRow) domain.MediaSectionTotals {
	totals := domain.MediaSectionTotals{}
	for _, row := range rows {
		if classifyMediaSection(row.MatchID) == mediaSectionUnassigned {
			totals.Unassigned++
			continue
		}
		totals.Mine++
	}
	return totals
}

func classifyMediaSection(matchID *string) string {
	if matchID == nil || strings.TrimSpace(*matchID) == "" {
		return mediaSectionUnassigned
	}
	return mediaSectionMine
}

func mediaLikeCount(liked bool) int {
	if liked {
		return 1
	}
	return 0
}

func normalizeMediaKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case mediaKindVideo:
		return mediaKindClip
	case mediaKindImage:
		return mediaKindScreenshot
	case mediaKindClip, mediaKindScreenshot, "thumbnail":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return mediaKindScreenshot
	}
}
