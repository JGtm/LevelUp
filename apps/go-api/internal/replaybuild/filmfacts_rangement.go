package replaybuild

// filmfacts_rangement.go — RANGER LES FAITS D UNE CUISSON QUI A DECODE.
//
// SORTI DE `filmfacts_cuisson.go` le 2026-09-26 (lot J3.4 du PLAN_SUITE_AUDIT_DECODEUR_FILM) :
// le rangement y gagnait sa condition — le verdict du puits — et le fichier passait les 500
// lignes. DEPLACEMENT PUR de `ecrireLesFaits` ; `rangerLesFaits` est neuf.

import (
	"context"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/platform/atomicfile"
)

// rangerLesFaits range les faits d une cuisson qui a DECODE — SEULEMENT si le puits d artefact
// rangerait son artefact (lot J3.4 du PLAN_SUITE_AUDIT_DECODEUR_FILM, constat RA1-1).
//
// LE SCENARIO QU IL FERME : une cuisson sans faits de base produit un artefact APPAUVRI, que le
// puits refuse parce qu il retrograderait celui en place. Ses faits, cuits sous des gardes pauvres,
// s ecrivaient pourtant, et la reparation suivante les rejouait. La decision est CELLE DU PUITS
// ([refusParLePuits]), prise ici sur le meme disque : `BuildBytes` tourne dans l enfant de cuisson
// avec la racine du parent, qui range juste apres. Seul un ecrivain CONCURRENT du meme artefact
// entre les deux gestes (le depot d un ouvrier distant) ferait diverger la prediction ; il ne
// rend jamais des faits faux, au pire des faits non ecrits ou ecrits pour un artefact conserve.
func (b *Builder) rangerLesFaits(ctx context.Context, matchID string, blob []byte,
	f *replay.FilmFactsFile,
) {
	outPath := title.NewPathResolver(b.repoRoot).ReplayArtifactPath(b.titleSlug, matchID)
	if err := refusParLePuits(outPath, b.titleSlug, matchID, blob); err != nil {
		slog.WarnContext(ctx, "cuisson: faits de film NON ecrits — le puits refuserait l artefact "+
			"de cette cuisson", "match_id", matchID, "raison", err)
		return
	}
	b.ecrireLesFaits(ctx, matchID, f)
}

// ecrireLesFaits range les faits d une cuisson qui a DECODE. Non fatal.
func (b *Builder) ecrireLesFaits(ctx context.Context, matchID string, f *replay.FilmFactsFile) {
	if f == nil {
		slog.WarnContext(ctx, "cuisson: aucun fait a persister (decoupage d i0 illisible) — la "+
			"prochaine cuisson de ce match redecodera", "match_id", matchID)
		return
	}
	blob, err := replay.EncodeFilmFactsFile(f)
	if err != nil {
		slog.ErrorContext(ctx, "cuisson: faits de film non serialisables", "match_id", matchID,
			"err", err)
		return
	}
	chemin := b.filmFactsPath(matchID)
	if err := os.MkdirAll(title.NewPathResolver(b.repoRoot).FilmFactsDir(b.titleSlug), 0o750); err != nil {
		slog.ErrorContext(ctx, "cuisson: dossier des faits de film non cree", "match_id", matchID,
			"path", chemin, "err", err)
		return
	}
	if err := atomicfile.WriteFile(chemin, blob, 0o600); err != nil {
		slog.ErrorContext(ctx, "cuisson: faits de film non ecrits", "match_id", matchID,
			"path", chemin, "err", err)
		return
	}
	slog.InfoContext(ctx, "cuisson: faits de film ecrits", "match_id", matchID, "path", chemin,
		"bytes", len(blob))
}
