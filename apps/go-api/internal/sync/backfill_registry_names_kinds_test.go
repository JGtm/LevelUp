package sync

import (
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
)

// TestRegistryNameKinds_AlignesSurLesGenresDAsset : le planificateur passe au persister les genres
// games.AssetKind* ; le persister les reconnaît par ses propres constantes. Une divergence ferait
// refuser chaque écriture (genre inconnu).
func TestRegistryNameKinds_AlignesSurLesGenresDAsset(t *testing.T) {
	paires := map[string]string{
		games.AssetKindPlaylist:    persist.RegistryNamePlaylist,
		games.AssetKindMap:         persist.RegistryNameMap,
		games.AssetKindPair:        persist.RegistryNamePair,
		games.AssetKindGameVariant: persist.RegistryNameGameVariant,
	}
	for genre, persiste := range paires {
		if genre != persiste {
			t.Errorf("genre %q != constante du persister %q", genre, persiste)
		}
	}
}
