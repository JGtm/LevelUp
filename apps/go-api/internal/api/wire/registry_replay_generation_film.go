package wire

// registry_replay_generation_film.go — le film d'un match demandé à la génération à la demande : l'archive d'abord,
// puis le téléchargement avec le jeton DU JOUEUR DE LA DEMANDE, l'archivage, le marqueur « film absent » quand le
// film n'existe plus, et le retrait du film après la construction quand l'archive est désactivée.
//
// LE JETON. Un seul appel, `tokencache.ResolveFreshPlayerTokens(ctx, d.XUID)`, paramétré par le xuid porté par la
// demande (celui du profil de la page, admis par la garde de propriété). Aucun autre jeton ne sert : ni le pool, ni le
// premier jeton valide, ni une boucle sur les profils (garde archlint `replay_generation_token_source_test.go`). Un film
// déjà complet à l'archive ne demande aucun jeton. Les jetons ne servent qu'au client de ce film ; aucune trace n'en
// est gardée dans la file, l'état ni le journal, et aucun message ne conseille de recapturer. Un refus d'authentification
// (401, 403) pendant la résolution du film déclenche UN nouvel essai avec un jeton frais du même joueur
// (`tokencache.RetryWithFreshTokens` : cache du joueur invalidé, jeton re-dérivé).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth/tokencache"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/filmfetch"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/sync/replayartifacts"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

const (
	// replayGenerationFilmRPS : débit des requêtes du téléchargement d'un film (un seul film à la fois).
	replayGenerationFilmRPS = 3
	// replayGenerationAbsentLeaseBudget : attente maximale de l'écrivain partagé pour poser le marqueur « film
	// absent », une fois par film expiré, depuis la goroutine de la file (jamais sous une requête).
	replayGenerationAbsentLeaseBudget = 8 * time.Second
	// replayGenerationFilmPublishWindow : le délai après la fin d'un match pendant lequel un film introuvable chez
	// l'éditeur (404, 410 ou manifeste sans morceau) n'est PAS tenu pour expiré. L'éditeur publie le film après la fin du
	// match ; ce que son service répond juste avant (404) ne se distingue pas de sa réponse pour un film expiré, et seul
	// le délai les sépare. Dans ce délai la demande échoue en « film pas encore prêt » (réessayable) et le bit « film
	// absent » n'est pas posé ; passé ce délai, un film introuvable est perdu pour de bon. Un manifeste publié sans
	// morceau de temps forts n'a pas besoin du délai : finalise.ErrFilmNonFinalise le reconnaît. Seule la génération à la
	// demande applique ce délai ; l'heure de fin vient de la demande (domain.ReplayGenerationDemand.MatchEndedAt), zéro
	// = inconnue = aucun délai.
	replayGenerationFilmPublishWindow = 5 * time.Minute
)

// filmDownloader : la seule opération réseau du film, rendue par *haloadapter.Client.
type filmDownloader interface {
	GetFilmChunks(ctx context.Context, matchID string) ([]filmfetch.FilmChunk, bool, error)
}

// filmResult : ce que la résolution du film rend (chunks triés, film trouvé), en un seul résultat pour le filet
// d'authentification.
type filmResult struct {
	chunks []filmfetch.FilmChunk
	found  bool
}

// filmDemandFetcher : le téléchargement du film d'une demande, avec ses dépendances PASSÉES (jamais résolues ici)
// pour que les tests l'exercent sans réseau ni base.
type filmDemandFetcher struct {
	archiveRoot string
	// complete dit si le film est complet à l'archive (filmcache.FilmComplet).
	complete func(root, short string) (bool, error)
	// ensurePlayer rend le joueur de la demande résolu au pool du serveur (le rafraîchissement de jeton le cherche
	// par xuid). nil = rien à faire.
	ensurePlayer func(ctx context.Context, d domain.ReplayGenerationDemand) error
	// tokens rend les jetons frais DU joueur xuid : tokencache.ResolveFreshPlayerTokens en production.
	tokens func(ctx context.Context, xuid string) (*domain.HaloTokens, error)
	// newClient construit le client de téléchargement d'un film.
	newClient func(tokens *domain.HaloTokens) filmDownloader
	// withFreshTokens exécute la résolution du film avec le filet d'un refus d'authentification : un seul nouvel essai,
	// avec un jeton frais du joueur lu dans le contexte (tokencache.RetryWithFreshTokens en production). nil : un seul
	// essai, sans filet.
	withFreshTokens func(ctx context.Context, fn func(context.Context) (filmResult, error)) (filmResult, error)
	// write archive le film (filmarchive.Write).
	write func(ctx context.Context, root, short string, chunks []filmarchive.WriteChunk) error
	// markAbsent pose le bit « film absent » du match (killcollector.MarquerFilmAbsent).
	markAbsent func(ctx context.Context, matchID string) error
	// now rend l'heure courante ; nil = time.Now.
	now func() time.Time
}

// fetch télécharge et archive le film de la demande, ou constate qu'il est déjà complet à l'archive.
func (f *filmDemandFetcher) fetch(ctx context.Context, d domain.ReplayGenerationDemand) error {
	short := title.FilmShortMatchID(d.MatchID)
	complete, err := f.complete(f.archiveRoot, short)
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: état du film à l'archive illisible, film retéléchargé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "err", err)
	}
	if complete {
		return nil
	}
	if f.ensurePlayer != nil {
		if err := f.ensurePlayer(ctx, d); err != nil {
			slog.ErrorContext(ctx, "replay_generation: profil de la demande introuvable",
				"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
			return fmt.Errorf("profil de la demande: %w", err)
		}
	}
	res, err := f.resolveFilm(ctx, d)
	switch {
	case errors.Is(err, domain.ErrReplayNoToken):
		return err
	case errors.Is(err, finalise.ErrFilmNonFinalise):
		return fmt.Errorf("film non finalisé: %w", err)
	case err != nil && filmfetch.IsFilmGoneErr(err):
		return f.filmGone(ctx, d, err)
	case err != nil:
		return fmt.Errorf("téléchargement du film: %w", err)
	case !res.found || len(res.chunks) == 0:
		return f.filmGone(ctx, d, nil)
	}
	chunks := res.chunks
	wc := make([]filmarchive.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmarchive.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS, Data: c.Data,
		})
	}
	if err := f.write(ctx, f.archiveRoot, short, wc); err != nil {
		slog.ErrorContext(ctx, "replay_generation: film non archivé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
		return fmt.Errorf("archivage du film: %w", err)
	}
	return nil
}

// resolveFilm obtient les jetons du joueur de la demande, UN SEUL APPEL, puis résout le film avec un client bâti sur le
// jeton que porte le contexte : un refus d'authentification re-dérive le jeton et recommence une fois
// ([filmDemandFetcher.withFreshTokens]). Sans jeton utilisable (aucun, erreur, jeton vide) : domain.ErrReplayNoToken,
// cause journalisée, aucun conseil de recapture.
func (f *filmDemandFetcher) resolveFilm(ctx context.Context, d domain.ReplayGenerationDemand) (filmResult, error) {
	tokens, err := f.tokens(ctx, d.XUID)
	if err != nil || tokens == nil || tokens.SpartanToken == "" {
		slog.WarnContext(ctx, "replay_generation: jeton du joueur indisponible",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "xuid", d.XUID, "err", err)
		if err != nil {
			return filmResult{}, fmt.Errorf("%w: %w", domain.ErrReplayNoToken, err)
		}
		return filmResult{}, domain.ErrReplayNoToken
	}
	attempt := func(c context.Context) (filmResult, error) {
		chunks, found, err := f.newClient(ctxkeys.HaloTokens(c)).GetFilmChunks(c, d.MatchID)
		return filmResult{chunks: chunks, found: found}, err
	}
	ctx = ctxkeys.WithHaloAuth(ctx, tokens, d.XUID)
	if f.withFreshTokens == nil {
		return attempt(ctx)
	}
	return f.withFreshTokens(ctx, attempt)
}

// filmGone constate qu'un film est introuvable (404/410 ou manifeste sans morceau). Pour un match fini depuis moins de
// [replayGenerationFilmPublishWindow] : domain.ErrReplayFilmNotReady (réessayable), aucun marqueur. Sinon : pose le
// marqueur « film absent » et rend domain.ErrReplayFilmGone. Un échec du marqueur se journalise ; l'état reste « film
// indisponible » pour ce processus.
func (f *filmDemandFetcher) filmGone(ctx context.Context, d domain.ReplayGenerationDemand, cause error) error {
	if f.justEnded(d) {
		slog.InfoContext(ctx, "replay_generation: film introuvable mais match fini depuis peu, pas de verdict d'expiration",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug,
			"ended_at", d.MatchEndedAt, "window", replayGenerationFilmPublishWindow)
		if cause != nil {
			return fmt.Errorf("%w: %w", domain.ErrReplayFilmNotReady, cause)
		}
		return domain.ErrReplayFilmNotReady
	}
	if err := f.markAbsent(ctx, d.MatchID); err != nil {
		slog.WarnContext(ctx, "replay_generation: marqueur film absent non posé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
	}
	if cause != nil {
		return fmt.Errorf("%w: %w", domain.ErrReplayFilmGone, cause)
	}
	return domain.ErrReplayFilmGone
}

// justEnded dit si le match de la demande a fini il y a moins de [replayGenerationFilmPublishWindow] ; faux quand son
// heure de fin est inconnue.
func (f *filmDemandFetcher) justEnded(d domain.ReplayGenerationDemand) bool {
	if d.MatchEndedAt.IsZero() {
		return false
	}
	now := time.Now
	if f.now != nil {
		now = f.now
	}
	return now().Sub(d.MatchEndedAt) < replayGenerationFilmPublishWindow
}

// retryFilmWithFreshTokens : la résolution du film sous le filet d'un refus d'authentification (401, 403) : cache de
// jetons du joueur invalidé (le xuid et le jeton viennent du contexte), jeton re-dérivé, UN nouvel essai.
func retryFilmWithFreshTokens(ctx context.Context, fn func(context.Context) (filmResult, error)) (filmResult, error) {
	return tokencache.RetryWithFreshTokens(ctx, haloadapter.IsAuthError, fn)
}

// fetchFilmForDemand est la fonction branchée sur replayregen.Deps.FetchFilm.
func (r *ServiceRegistry) fetchFilmForDemand(ctx context.Context, d domain.ReplayGenerationDemand) error {
	paths := r.cfg.PathResolver()
	return (&filmDemandFetcher{
		archiveRoot: paths.FilmArchiveDir(),
		complete:    filmcache.FilmComplet,
		ensurePlayer: func(ctx context.Context, d domain.ReplayGenerationDemand) error {
			_, err := r.resolve(ctxkeys.WithTitleSlug(ctx, d.TitleSlug), d.PlayerSlug)
			return err
		},
		tokens:          tokencache.ResolveFreshPlayerTokens,
		withFreshTokens: retryFilmWithFreshTokens,
		newClient: func(t *domain.HaloTokens) filmDownloader {
			return syncpkg.NewHaloAPIClient(t.SpartanToken, t.ClearanceToken, replayGenerationFilmRPS)
		},
		write: filmarchive.Write,
		markAbsent: func(ctx context.Context, matchID string) error {
			acquire := syncpkg.NewSharedWriterAcquirer(r.sharedProviderForTitle(d.TitleSlug),
				paths.SharedDBPath(d.TitleSlug), "replay_generation_film_absent", replayGenerationAbsentLeaseBudget)
			return killcollector.MarquerFilmAbsent(ctx, persist.SharedWriterFn(acquire), matchID)
		},
	}).fetch(ctx, d)
}

// releaseFilmAfterBuild retire le film d'un match dont la construction vient de se clore (rejeu rangé, ou échec
// définitif) quand le réglage « archiver les films » est faux (même règle que la cuisson). Un échec de retrait se journalise : le film reste.
func (r *ServiceRegistry) releaseFilmAfterBuild(ctx context.Context, titleSlug, matchID string) {
	retirerFilmSansArchive(ctx, r.cfg.PathResolver().FilmArchiveDir(), titleSlug, matchID, r.archiveFilmsEnabled(ctx))
}

// retirerFilmSansArchive retire le film du match de l'archive quand le réglage « archiver les films » est faux.
func retirerFilmSansArchive(ctx context.Context, archiveRoot, titleSlug, matchID string, archive bool) {
	if archive {
		return
	}
	if err := filmarchive.Supprimer(archiveRoot, title.FilmShortMatchID(matchID)); err != nil {
		slog.WarnContext(ctx, "replay_generation: film généré non retiré (archive désactivée)",
			"titleSlug", titleSlug, "match_id", matchID, "err", err)
		return
	}
	observability.IncCounter(replayartifacts.CompteurFilmsRetiresSansArchive)
}

// archiveFilmsEnabled lit le réglage « archiver les films » (oui par défaut : sans réglages, le film reste).
func (r *ServiceRegistry) archiveFilmsEnabled(ctx context.Context) bool {
	if r.settingsStore == nil {
		return true
	}
	s, err := r.settingsStore.Load()
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: réglages illisibles, film conservé", "err", err)
		return true
	}
	return s == nil || s.ArchiveFilms
}
