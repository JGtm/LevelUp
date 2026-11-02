package killcollector

import (
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// TestRacineDuCache_SuitLArchiveChangeeAChaud : une archive vide change d'emplacement sans
// redémarrage (title.SetFilmArchiveRuntime) ; la passe suivante archive dans le nouveau dossier au
// lieu de rester sur la racine mémorisée à la première passe.
func TestRacineDuCache_SuitLArchiveChangeeAChaud(t *testing.T) {
	t.Cleanup(titlePkg.ResetFilmArchiveRuntime)
	titlePkg.ResetFilmArchiveRuntime()
	h := NewPostSyncHook(t.TempDir(), 0)
	premiere, _ := h.racineDuCache(t.Context(), nil)

	nouvelle := filepath.Join(t.TempDir(), "films")
	titlePkg.SetFilmArchiveRuntime(nouvelle)
	seconde, cache := h.racineDuCache(t.Context(), nil)
	if seconde != nouvelle || cache == nil {
		t.Fatalf("racine = %q (cache nil=%v), attendu %q", seconde, cache == nil, nouvelle)
	}
	if premiere == seconde {
		t.Fatal("la racine mémorisée n'a pas été renouvelée")
	}
	if again, _ := h.racineDuCache(t.Context(), nil); again != nouvelle {
		t.Fatalf("racine stable perdue : %q", again)
	}
}
