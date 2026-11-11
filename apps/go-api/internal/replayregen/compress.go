package replayregen

// compress.go — LA COMPRESSION DES REJEUX BRUTS, en fin de passe.
//
// Un rejeu refait par la file s'écrit compressé (`{short8}.json.gz`). Reste ceux qu'elle ne peut PAS refaire (film
// absent de l'archive), et ceux déjà à jour : ils restent des `{short8}.json` bruts d'avant la compression des rejeux.
// Une application de bureau installée n'a pas de ligne de commande ; la file les compresse donc elle-même, par la
// conversion de `replaybuild` (la commande `levelup replay compress` en est l'autre appelant).
//
// QUAND. Seulement à la FIN d'une passe lancée par l'utilisateur (démarrage demandé, ou reprise d'une passe que ce
// geste avait lancée) : la compression découle du geste, elle ne le remplace pas. Jamais au démarrage du serveur, jamais
// par le décompte.
//
// COMMENT. Un fichier à la fois, jamais pendant un décodage de la file (elle est séquentielle) : la porte [Service.waitTurn]
// est franchie avant CHAQUE fichier — pause de l'utilisateur, partie en cours (mode jeu), cycle de synchronisation (qui
// cuit) — donc la conversion ne passe jamais en parallèle d'une cuisson de la file ou de la synchronisation. La
// conversion est idempotente et ne supprime un `.json` qu'après avoir relu son `.json.gz` identique : un arrêt, une
// pause ou une panne n'en perd aucun.

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
)

// compressRaw compresse les rejeux bruts de tous les titres. Rend faux si l'arrêt du serveur l'a coupée (le drapeau de
// reprise reste posé, comme pour un film) ; vrai si elle est allée au bout, échecs comptés compris.
func (s *Service) compressRaw(ctx context.Context) bool {
	s.mu.Lock()
	d := s.deps
	s.compressing = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.compressing = false
		s.mu.Unlock()
	}()

	var total replaybuild.CompressReport
	opts := replaybuild.CompressOptions{
		Gate: func(ctx context.Context) error {
			if !s.waitTurn(ctx) {
				return ctx.Err()
			}
			return nil
		},
		Progress: func(r replaybuild.CompressReport) {
			total.Add(r)
			s.countCompressed(r)
		},
	}
	for _, slug := range d.Titles() {
		_, err := replaybuild.CompressDirWith(ctx, d.Paths.ReplayArtifactsDir(slug), opts)
		switch {
		case err == nil, errors.Is(err, fs.ErrNotExist): // pas de dossier de rejeux : rien à compresser
		case ctx.Err() != nil:
			slog.InfoContext(ctx, "replay_regen: compression interrompue par l'arrêt, reprise à la prochaine ouverture",
				"compresses", total.Converted+total.Resolved)
			return false
		default:
			slog.WarnContext(ctx, "replay_regen: compression des rejeux bruts abandonnée pour ce titre",
				"titleSlug", slug, "err", err)
		}
	}
	if total.Converted+total.Resolved+total.Failed+total.Different+total.ToRetry > 0 {
		slog.InfoContext(ctx, "replay_regen: rejeux bruts compressés", "converted", total.Converted,
			"resolved", total.Resolved, "failed", total.Failed, "different", total.Different, "to_retry", total.ToRetry,
			"bytes_before", total.BytesBefore, "bytes_after", total.BytesAfter)
	}
	return true
}

// countCompressed ajoute le compte rendu d'un fichier aux compteurs de la file (état et expvar). Un doublon résorbé
// finit compressé ; un doublon de contenu différent et un brut à relancer ne le sont pas : ils comptent en échec.
func (s *Service) countCompressed(r replaybuild.CompressReport) {
	ok, ko := r.Converted+r.Resolved, r.Failed+r.Different+r.ToRetry
	if ok == 0 && ko == 0 {
		return
	}
	s.mu.Lock()
	s.compressed += ok
	s.compressFailed += ko
	s.mu.Unlock()
	for i := 0; i < ok; i++ {
		observability.IncCounter(MetricCompressed)
	}
	for i := 0; i < ko; i++ {
		observability.IncCounter(MetricCompressFailed)
	}
}
