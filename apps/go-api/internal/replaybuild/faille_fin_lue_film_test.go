package replaybuild

// faille_fin_lue_film_test.go — LA FAILLE FINIT A LA SUPPRESSION LUE DE SON OBJET, sur un film reel.
//
// La faille du translocateur vit de sa pose jusqu a la suppression de l objet ti=38 que la pose cree,
// ecrite par le film (`tracked-end` de la pile i28 de l objet porte), et la pose la publie
// (`riftPoses[].t1`). Le temoin est `1b2d9e08` (Dynasty) : ses quatre poses ont leur fin lue. La
// cuisson est celle de production (`BuildBytes`, carte et decoupage poses, faits non ecrits). Gate
// LOCAL, la CI n a pas de films :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache go test ./internal/replaybuild/ -run '^TestLaFailleFinitASaSuppressionLue$'
//
// MUTATION VUE ROUGE : `finLue` (`film/replay/document_faille_pose.go`) rendant toujours nil — 0 fin
// lue sur 4.

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

func TestLaFailleFinitASaSuppressionLue(t *testing.T) {
	cache := os.Getenv("REPLAY_FILM_CACHE")
	if cache == "" {
		t.Skip("REPLAY_FILM_CACHE absent : gate local de la fin de la faille saute")
	}
	const film, carte, posesAttendues = "1b2d9e08", "Dynasty", 4
	ctx := context.Background()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	b, err := NewBuilder(ctx, racine, title.DefaultSlug)
	if err != nil {
		t.Fatalf("constructeur : %v", err)
	}
	built, err := b.SansEcritureDesFaits().BuildBytes(ctx, film, []string{carte},
		filmcache.ChunkDir(cache, film), port.MatchFacts{})
	if err != nil {
		t.Fatalf("%s : cuisson : %v", film, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(built.Blob, &doc); err != nil {
		t.Fatalf("%s : document illisible : %v", film, err)
	}
	if len(doc.RiftPoses) != posesAttendues {
		t.Fatalf("%s : %d pose(s) de faille, attendu %d", film, len(doc.RiftPoses), posesAttendues)
	}
	lues := 0
	for _, p := range doc.RiftPoses {
		switch {
		case p.T1 == nil:
			t.Errorf("%s : pose t=%d slot=%d sans fin lue", film, p.T, p.Slot)
		case *p.T1 <= p.T:
			t.Errorf("%s : pose t=%d slot=%d finit a %d, pas apres sa pose", film, p.T, p.Slot, *p.T1)
		default:
			lues++
		}
	}
	cov := doc.Coverage.CarriedEquipment
	if cov == nil || cov.RiftEndsRead != lues || cov.RiftEndsUnread != len(doc.RiftPoses)-lues {
		t.Errorf("%s : couverture %+v, attendu %d fin(s) lue(s) et %d par repli", film, cov, lues,
			len(doc.RiftPoses)-lues)
	}
}
