package replaybuild

// schema_census.go — le décompte des rejeux rangés d'un titre par version de schéma : une lecture d'en-tête par rejeu
// (le petit fichier voisin, l'artefact entier une seule fois quand il manque), jamais le document.

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replayartifact"
)

// ArtifactDirs rend le dossier des rejeux rangés d'un titre (satisfait par `*title.PathResolver`).
type ArtifactDirs interface {
	ReplayArtifactsDir(titleSlug string) string
}

// SchemaCensus compte les rejeux rangés par version de schéma ; il satisfait `port.ReplaySchemaCensus`.
type SchemaCensus struct {
	dirs ArtifactDirs
}

// NewSchemaCensus construit le décompte sur les dossiers de rejeux des titres.
func NewSchemaCensus(dirs ArtifactDirs) *SchemaCensus { return &SchemaCensus{dirs: dirs} }

// CurrentSchema rend le schéma que produit le décodeur monté dans ce serveur.
func (*SchemaCensus) CurrentSchema() int { return replay.SchemaVersion }

// CountReplaySchemas compte les rejeux du titre. Un dossier absent rend des comptes à zéro ; l'annulation du contexte
// interrompt le décompte et rend son erreur.
func (c *SchemaCensus) CountReplaySchemas(ctx context.Context, titleSlug string) (domain.AdminReplaySchemaTitle, error) {
	out := domain.AdminReplaySchemaTitle{TitleSlug: titleSlug}
	dir := c.dirs.ReplayArtifactsDir(titleSlug)
	entries, err := replayartifact.List(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil
		}
		return out, err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		out.Total++
		h, ok := ArtifactHeader(ctx, filepath.Join(dir, e.Name))
		switch {
		case !ok:
			out.Unreadable++
		case h.SchemaVersion < replay.SchemaVersion:
			out.Behind++
		default:
			out.Current++
		}
	}
	return out, nil
}
