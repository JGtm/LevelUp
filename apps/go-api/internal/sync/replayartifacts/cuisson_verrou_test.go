package replayartifacts

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// TestBuildAll_VerrouTenuParLaFile_ReporteSansEchec : la file de regeneration des rejeux (ou l'action admin) decode et
// tient le verrou solo de la machine. La cuisson apres synchronisation ne decode PAS en parallele : elle reporte le match
// au cycle suivant — ni echec compte, ni avertissement par film — et le cycle suivant, verrou rendu, le cuit.
//
// La phase 1 passe par la strategie de PRODUCTION (`SpawnBuildOne`) : le refus du verrou revient avant tout lancement
// d'enfant, et son enveloppe doit rester reconnaissable (`errors.Is(ErrDecodeBusy)`).
func TestBuildAll_VerrouTenuParLaFile_ReporteSansEchec(t *testing.T) {
	repoRoot := t.TempDir()
	racineVerrou := titlePkg.NewPathResolver(repoRoot).CacheRootDir()
	lock, err := filmproc.AcquireSolo(racineVerrou, "file de regeneration", "00000001")
	if err != nil {
		t.Fatal(err)
	}
	work := []buildWork{{matchID: "0000d001"}, {matchID: "0000d002"}}
	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug, FilmRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute, BuildOne: SpawnBuildOne,
	}
	b := buildAll(context.Background(), d, work)
	if b.reportesVerrou != 2 || b.echecs != 0 || b.construits != 0 {
		t.Fatalf("reportes verrou = %d, echecs = %d, construits = %d ; attendu 2, 0, 0", b.reportesVerrou, b.echecs, b.construits)
	}
	if b.traites() != 2 {
		t.Fatalf("traites = %d : un report se compte avec les matchs decides", b.traites())
	}

	// Cycle suivant : la file a rendu le verrou. Un decodeur qui le prend comme le fait Spawn ne rencontre plus de
	// refus, et ne decode jamais deux films a la fois.
	lock.Release()
	var appels, enVol, maxEnVol int
	d.BuildOne = func(_ context.Context, req BuildOneRequest) (BuildOneResult, error) {
		l, aerr := filmproc.AcquireSolo(racineVerrou, "post-sync", req.MatchID)
		if aerr != nil {
			return BuildOneResult{}, fmt.Errorf("cuisson non lancee : %w", aerr)
		}
		defer l.Release()
		appels++
		enVol++
		maxEnVol = max(maxEnVol, enVol)
		enVol--
		return BuildOneResult{}, errors.New("echec ordinaire apres le decodage")
	}
	b = buildAll(context.Background(), d, work)
	if appels != 2 || maxEnVol != 1 || b.reportesVerrou != 0 {
		t.Fatalf("appels = %d, decodages paralleles = %d, reportes verrou = %d ; attendu 2, 1, 0", appels, maxEnVol, b.reportesVerrou)
	}
}
