package replaybuild

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/LevelUp/libs/film/equipenergy"
)

// energieCatalog charge le catalogue de l energie des equipements au plus une fois par Builder —
// meme motif que `padsCatalog`. Sans lui, aucun usage restant n est compte (`replay`
// `coverage.carriedEquipment.noCatalog`), et le dire est le travail de ce journal.
func (b *Builder) energieCatalog(ctx context.Context) *equipenergy.Catalog {
	if b.energieTried {
		return b.energie
	}
	b.energieTried = true
	path := title.NewPathResolver(b.repoRoot).EquipmentEnergyPath(b.titleSlug)
	cat, err := equipenergy.Load(path)
	if err != nil {
		slog.WarnContext(ctx, "replaybuild: catalogue de l energie des equipements illisible — aucun usage restant compte",
			"err", err, "path", path, "titleSlug", b.titleSlug)
		return nil
	}
	b.energie = cat
	return cat
}
