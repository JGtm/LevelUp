// Package ops — seed_demo_replays.go : les rejeux FIGÉS de la démo (décisions D-1 à D-3 du
// plan des recommandations du 2026-10-09).
//
// CE QUE FAIT CETTE PHASE, pour chaque rejeu du manifeste (`replay_matches`) :
//  1. elle range le film source dans le MAGASIN PERSISTANT des films de la démo
//     (`PathResolver.DemoFilmsCacheRoot`, hors du dossier de la démo, régénéré à chaque seed)
//     quand le cache du dépôt source le porte ; sinon elle garde celui du magasin. Le magasin
//     se provisionne une fois là où le cache de films n'existe pas (le VPS web) ;
//  2. elle installe un artefact AU DERNIER SCHÉMA (`replaybuild.ArtifactUpToDate`) : celui du
//     dépôt source s'il est à jour, sinon celui de la démo publiée s'il l'est, sinon elle
//     RECUIT le film du magasin — un film par processus, plafond mémoire explicite
//     (demoReplaySoftLimit), en attendant le verrou de décodage de la machine, sans rien
//     écrire hors de la démo. Une montée de `replay.SchemaVersion` est ainsi rattrapée à la
//     régénération suivante de la démo, sans rien demander ;
//  3. elle écrit l'index (`domain.DemoReplayIndex`) : les matchs servis, et la correspondance
//     des identités réelles vers le roster démo, que le serveur applique au document SERVI
//     (l'artefact n'est jamais modifié, décision D-1).
//
// UN REJEU QUI ÉCHOUE N'EMPORTE PAS LE SEED : il est journalisé en ERREUR et compté dans le
// bilan (`DemoReplaysReport`). Un artefact qui n'a pas pu être remis au dernier schéma reste
// servi, marqué « périmé » au bilan ; un rejeu sans aucun artefact n'est pas servi.
package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/atomicfile"
	"levelup/go-api/internal/replaybuild"
)

// Issues d'un rejeu figé au bilan du seed.
const (
	DemoReplayCopied  = "copie"    // artefact à jour copié depuis le dépôt source
	DemoReplayKept    = "conserve" // artefact embarqué déjà à jour
	DemoReplayCooked  = "recuit"   // recuit depuis le film embarqué
	DemoReplayStale   = "perime"   // servi, mais pas au dernier schéma
	DemoReplayMissing = "absent"   // aucun artefact : non servi
)

// DemoReplaysReport : le bilan de la phase, par match (issue) et le nombre de films embarqués.
type DemoReplaysReport struct {
	Outcomes map[string]string
	Films    int
}

// cacheRepoRoot rend la racine du dépôt dont viennent les films et les artefacts sources.
func (o SeedDemoOptions) cacheRepoRoot() string {
	if o.CacheRepoRoot != "" {
		return o.CacheRepoRoot
	}
	return o.RepoRoot
}

// seedDemoReplays installe les rejeux figés de la démo et écrit leur index. Phase 6c.
func seedDemoReplays(ctx context.Context, opts SeedDemoOptions, layout titlePkg.DemoLayout,
	picks []DemoReplayPick, roster []demoRosterEntry) DemoReplaysReport {
	rep := DemoReplaysReport{Outcomes: map[string]string{}}
	slug := opts.TitleSlug
	if len(picks) == 0 && !fileExists(layout.ReplaysDir(slug)) {
		return rep // titre sans rejeu figé, et rien d'embarqué à retirer
	}
	keep := map[string]bool{}
	var index domain.DemoReplayIndex
	films := titlePkg.NewPathResolver(opts.RepoRoot).DemoFilmsCacheRoot(slug)
	for _, p := range picks {
		short := titlePkg.FilmShortMatchID(p.MatchID)
		keep[short] = true
		filmOK := embedDemoFilm(ctx, titlePkg.NewPathResolver(opts.cacheRepoRoot()).CacheRootDir(), films, short)
		if filmOK {
			rep.Films++
		}
		outcome := installDemoArtifact(ctx, opts, layout, p.MatchID, filmOK)
		rep.Outcomes[p.MatchID] = outcome
		if outcome == DemoReplayMissing {
			continue
		}
		d, _ := replaybuild.ArtifactDigest(layout.ReplayArtifactPath(slug, p.MatchID))
		index.Matches = append(index.Matches, domain.DemoReplayIndexEntry{
			MatchID: p.MatchID, Mode: p.Mode, Map: p.Map, SchemaVersion: d.SchemaVersion,
		})
	}
	pruneDemoReplays(ctx, layout, films, slug, keep)
	for _, e := range roster {
		index.Identities = append(index.Identities, domain.DemoReplayIdentity{
			XUID: e.SourceXUID, DemoXUID: e.DemoXUID, DemoGamertag: e.DemoGamertag,
		})
	}
	if err := writeDemoReplayIndex(layout.ReplayIndexPath(slug), index); err != nil {
		slog.ErrorContext(ctx, "seed-demo: index des rejeux non écrit — aucun rejeu servi", "err", err)
	}
	slog.InfoContext(ctx, "seed-demo: rejeux figés", "title", slug, "picks", len(picks),
		"servis", len(index.Matches), "films", rep.Films, "bilan", rep.Outcomes)
	return rep
}

// previousArtifactPath : l'artefact de ce match dans la démo PUBLIÉE (la génération en cours
// s'écrit à part, cf. seed_demo_publish.go) ; la démo en cours d'écriture elle-même quand le
// seed écrit en place.
func previousArtifactPath(opts SeedDemoOptions, layout titlePkg.DemoLayout, matchID string) string {
	if opts.PreviousOutDir != "" {
		return titlePkg.NewDemoLayout(opts.PreviousOutDir).ReplayArtifactPath(opts.TitleSlug, matchID)
	}
	return layout.ReplayArtifactPath(opts.TitleSlug, matchID)
}

// cookDemoReplayFunc : la recuisson d'un rejeu figé. Une variable pour que les tests
// remplacent l'enfant de décodage ; la production n'en a qu'une valeur.
var cookDemoReplayFunc = cookDemoReplay

// installDemoArtifact pose l'artefact démo d'un match au dernier schéma et rend son issue.
func installDemoArtifact(ctx context.Context, opts SeedDemoOptions, layout titlePkg.DemoLayout,
	matchID string, filmOK bool) string {
	slug := opts.TitleSlug
	dst := layout.ReplayArtifactPath(slug, matchID)
	prev := previousArtifactPath(opts, layout, matchID)
	src := titlePkg.NewPathResolver(opts.cacheRepoRoot()).ReplayArtifactPath(slug, matchID)
	for _, c := range []struct{ from, outcome string }{{src, DemoReplayCopied}, {prev, DemoReplayKept}} {
		if !replaybuild.ArtifactUpToDate(c.from) {
			continue
		}
		if c.from == dst {
			return c.outcome
		}
		if err := copyArtifact(c.from, dst); err != nil {
			slog.ErrorContext(ctx, "seed-demo: copie d'artefact de rejeu échouée", "err", err, "match_id", matchID)
			continue
		}
		return c.outcome
	}
	if filmOK {
		err := cookDemoReplayFunc(ctx, opts, layout, matchID)
		if err == nil {
			return DemoReplayCooked
		}
		slog.ErrorContext(ctx, "seed-demo: recuisson du rejeu démo échouée", "err", err, "match_id", matchID)
	}
	return keepStaleArtifact(ctx, []string{prev, src}, dst, matchID)
}

// keepStaleArtifact : aucun artefact à jour n'a pu être posé. Le plus récent disponible (démo
// publiée, puis dépôt) reste servi (« périmé ») plutôt que rien ; sans aucun artefact, le
// rejeu n'est pas servi.
func keepStaleArtifact(ctx context.Context, candidates []string, dst, matchID string) string {
	for _, c := range candidates {
		if !fileExists(c) {
			continue
		}
		if c != dst {
			if err := copyArtifact(c, dst); err != nil {
				slog.ErrorContext(ctx, "seed-demo: copie d'artefact périmé échouée", "err", err, "match_id", matchID)
				continue
			}
		}
		slog.ErrorContext(ctx, "seed-demo: rejeu démo servi au schéma précédent", "match_id", matchID)
		return DemoReplayStale
	}
	slog.ErrorContext(ctx, "seed-demo: aucun artefact pour ce rejeu démo — non servi", "match_id", matchID)
	return DemoReplayMissing
}

// copyArtifact copie un artefact de rejeu, atomiquement.
func copyArtifact(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("lecture %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return atomicfile.WriteFile(dst, raw, 0o644)
}

// embedDemoFilm copie le film `short` du cache source vers le magasin persistant des films de
// la démo (les chunks et le manifeste), sans recopier un fichier déjà présent à la même
// taille. Rend true quand le magasin porte le film après l'appel (copié, ou déjà là).
func embedDemoFilm(ctx context.Context, srcCache, demoCache, short string) bool {
	srcDir := filmcache.ChunkDir(srcCache, short)
	if _, err := os.Stat(srcDir); err == nil {
		if err := copyFilmFiles(srcCache, demoCache, short); err != nil {
			slog.ErrorContext(ctx, "seed-demo: film du rejeu non embarqué", "err", err, "film", short)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		slog.ErrorContext(ctx, "seed-demo: cache de films source illisible", "err", err, "film", short)
	}
	_, found, err := filmcache.Open(demoCache, short)
	if err != nil {
		slog.ErrorContext(ctx, "seed-demo: film embarqué illisible", "err", err, "film", short)
		return false
	}
	return found
}

// copyFilmFiles copie les chunks et le manifeste d'un film entre deux racines de cache.
func copyFilmFiles(srcCache, demoCache, short string) error {
	srcDir, dstDir := filmcache.ChunkDir(srcCache, short), filmcache.ChunkDir(demoCache, short)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	pairs := [][2]string{{filmcache.ManifestPath(srcCache, short), filmcache.ManifestPath(demoCache, short)}}
	for _, e := range entries {
		if !e.IsDir() {
			pairs = append(pairs, [2]string{filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name())})
		}
	}
	for _, p := range pairs {
		if sameSize(p[0], p[1]) {
			continue
		}
		if err := copyFile(p[0], p[1]); err != nil {
			return fmt.Errorf("copie %s: %w", filepath.Base(p[0]), err)
		}
	}
	return nil
}

func sameSize(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && ia.Size() == ib.Size()
}

// pruneDemoReplays retire les artefacts de la démo et les films du magasin qui ne sont plus
// au manifeste : la démo et son magasin reflètent la liste figée, et rien d'autre (un
// artefact orphelin ferait apparaître une icône de rejeu menant à un refus).
func pruneDemoReplays(ctx context.Context, layout titlePkg.DemoLayout, films, slug string, keep map[string]bool) {
	entries, err := os.ReadDir(layout.ReplayArtifactsDir(slug))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.ErrorContext(ctx, "seed-demo: dossier des artefacts démo illisible", "err", err)
	}
	for _, e := range entries {
		short := strings.TrimSuffix(e.Name(), ".json")
		if e.IsDir() || keep[short] {
			continue
		}
		removeLogged(ctx, filepath.Join(layout.ReplayArtifactsDir(slug), e.Name()))
	}
	ids, err := filmcache.ListShortIDs(films) // magasin absent : liste vide, pas d'erreur
	if err != nil {
		slog.ErrorContext(ctx, "seed-demo: magasin des films démo illisible", "err", err)
		return
	}
	sort.Strings(ids)
	for _, short := range ids {
		if keep[short] {
			continue
		}
		removeLogged(ctx, filmcache.ChunkDir(films, short))
		removeLogged(ctx, filmcache.ManifestPath(films, short))
	}
}

func removeLogged(ctx context.Context, path string) {
	if err := os.RemoveAll(path); err != nil {
		slog.ErrorContext(ctx, "seed-demo: retrait d'un rejeu hors manifeste échoué", "err", err, "path", path)
	}
}

// writeDemoReplayIndex écrit l'index des rejeux démo.
func writeDemoReplayIndex(path string, index domain.DemoReplayIndex) error {
	if index.Matches == nil {
		index.Matches = []domain.DemoReplayIndexEntry{}
	}
	if index.Identities == nil {
		index.Identities = []domain.DemoReplayIdentity{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeJSONFile(path, index)
}
