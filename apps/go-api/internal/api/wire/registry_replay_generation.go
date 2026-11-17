package wire

// registry_replay_generation.go — la fabrique de la génération à la demande du rejeu d'un match (placement local) :
// le service lié au joueur de la page, la sonde du match (registre, bit « film absent », artefact, film archivé) et la
// présence du jeton au magasin. Aucun SQL ici : les lectures passent par les dépôts de platform/duckdb.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replayartifact"
	"levelup/go-api/internal/replayregen"
	"levelup/go-api/internal/service"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// ReplayGeneration rend le service de génération lié au joueur de la page : son profil est résolu depuis le slug du
// CHEMIN (jamais du corps), celui que la garde de propriété a déjà admis. La demande porte SON xuid.
func (r *ServiceRegistry) ReplayGeneration(ctx context.Context, slug string) (port.ReplayGeneration, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return r.replayGenerationFor(pdb, slug, replayregen.Default(), &replayMatchProbe{reg: r, pdb: pdb}), nil
}

// replayGenerationFor assemble le service pour le profil pdb (le joueur de la page) ; la file et la sonde sont passées.
func (r *ServiceRegistry) replayGenerationFor(pdb *duckdb.PlayerDB, slug string, queue port.ReplayGenerationQueue,
	probe port.ReplayMatchProbe) port.ReplayGeneration {
	return service.NewReplayGenerationService(r.capabilitiesForPDB(pdb), queue, probe, r.replayTokenPresence,
		service.ReplayGenerationPlayer{Slug: slug, XUID: pdb.XUID, Gamertag: pdb.Gamertag})
}

// replayTokenPresence dit si le joueur a un jeton de rafraîchissement au magasin : une lecture du fichier du joueur,
// aucun échange réseau. Une erreur de lecture se journalise et se lit « pas de jeton ».
func (r *ServiceRegistry) replayTokenPresence(xuid string) bool {
	if r.authStore == nil || xuid == "" {
		return false
	}
	user, err := r.authStore.Load(xuid)
	if err != nil {
		if !errors.Is(err, auth.ErrUserTokensNotFound) {
			slog.Warn("replay_generation: magasin de jetons illisible", "xuid", xuid, "err", err)
		}
		return false
	}
	return user != nil && strings.TrimSpace(user.OAuthRefreshToken) != ""
}

// replayMatchProbe lit ce que la base et le disque disent d'un match pour la génération.
type replayMatchProbe struct {
	reg *ServiceRegistry
	pdb *duckdb.PlayerDB
}

// ProbeReplayMatch : le registre (identité complète, mode géré), le bit « film absent », le rejeu rangé et le film
// archivé. Un match absent du registre ou dont le préfixe est ambigu rend Known=false.
func (p *replayMatchProbe) ProbeReplayMatch(ctx context.Context, titleSlug, matchID string) (port.ReplayMatchProbeResult, error) {
	hit, err := p.registryMatch(ctx, titleSlug, matchID)
	if err != nil || hit == nil {
		return port.ReplayMatchProbeResult{}, err
	}
	paths := p.reg.cfg.PathResolver()
	res := port.ReplayMatchProbeResult{
		FullID: hit.ID, Known: true, UnmanagedMode: hit.UnmanagedMode,
		ArtifactPresent: replayartifact.Exists(paths.ReplayArtifactPath(titleSlug, hit.ID)),
	}
	probes, err := duckdb.NewAnalysisProbeRepo(p.pdb).LoadAnalysisProbes(ctx, []string{hit.ID})
	if err != nil {
		return port.ReplayMatchProbeResult{}, fmt.Errorf("bit film absent du match: %w", err)
	}
	res.FilmAbsent = probes[hit.ID].FilmAbsent
	complete, err := filmcache.FilmComplet(paths.FilmArchiveDir(), title.FilmShortMatchID(hit.ID))
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: état du film à l'archive illisible, film lu non archivé",
			"titleSlug", titleSlug, "match_id", hit.ID, "err", err)
	}
	res.FilmArchived = complete
	return res, nil
}

// registryMatch lit le match au registre de la base partagée du titre ; nil quand il est inconnu ou ambigu. La
// lecture est rendue avant les autres (une lecture partagée à la fois).
func (p *replayMatchProbe) registryMatch(ctx context.Context, titleSlug, matchID string) (*port.ReplayRegistryMatch, error) {
	shared, release, err := p.reg.titleBases().ReadShared(ctx, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer release()
	hits, err := duckdb.NewReplayRegistryRepo(shared, nil).MatchesByIDOrPrefix(ctx, strings.TrimSpace(matchID))
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	switch {
	case len(hits) == 0:
		return nil, nil
	case len(hits) > 1:
		slog.WarnContext(ctx, "replay_generation: préfixe de match ambigu, match lu inconnu",
			"titleSlug", titleSlug, "match_id", matchID, "matches", len(hits))
		return nil, nil
	}
	return &hits[0], nil
}
