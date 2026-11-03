//go:build integration

package killcollector

import (
	"context"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// TestRunPostSync_VerrouTenu_ReporteSansDecoder : la file de regeneration des rejeux decode dans son enfant et tient le
// verrou solo. L'etape 1.57, qui decode dans le processus du serveur, ne demande AUCUN film et ne decode rien : la passe
// est reportee au cycle suivant. Verrou rendu, le meme backlog est repris.
func TestRunPostSync_VerrouTenu_ReporteSansDecoder(t *testing.T) {
	db := baseBacklog(t)
	inscrireMatch(t, db, "m1", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 0)
	depot := racineDepot(t)
	h := NewPostSyncHook(depot, 3)

	tenu, err := filmproc.AcquireSolo(titlePkg.NewPathResolver(depot).CacheRootDir(), "file de regeneration", "00000001")
	if err != nil {
		t.Fatal(err)
	}
	films := &filmsTraces{}
	RunPostSync(context.Background(), h, depsDeTest(db, films, nil), []string{"m1"})
	if len(films.demandes) != 0 {
		t.Fatalf("films demandes = %v : aucun film ne se telecharge ni ne se decode pendant qu'un autre decodage tient la machine", films.demandes)
	}

	tenu.Release()
	RunPostSync(context.Background(), h, depsDeTest(db, films, nil), []string{"m1"})
	if len(films.demandes) != 1 || films.demandes[0] != "m1" {
		t.Fatalf("films demandes = %v : le cycle suivant reprend le match", films.demandes)
	}
}
