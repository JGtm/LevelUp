package kickoff

// backfill.go — LE rattrapage : coup d'envoi, source et temps de jeu de tous les matchs d'une base,
// par lots, chaque lot dans UN burst court du writer partagé. Il sert au démarrage du serveur
// ([ApplyRevision], quand la révision rangée diffère de timeline.KickoffRev) et à la commande
// `levelup backfill-kickoff` (avec une simulation qui n'écrit rien).
//
// REPRISE : rien n'est mémorisé entre deux passages, et il n'en est pas besoin. Un lot écrit ne
// change plus rien au passage suivant (la résolution rend ce que la base porte déjà), donc un
// passage interrompu se reprend en relançant le même rattrapage ; la révision n'est rangée
// qu'après un passage complet sans échec.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/replaybuild"
)

// DefaultLotSize : matchs par burst d'écriture. Un lot de cette taille s'écrit en une à deux
// secondes : le writer partagé n'est jamais tenu longtemps.
const DefaultLotSize = 100

// ErrPartial : l'écriture d'au moins un match a échoué ; la révision n'est pas rangée. Un rejeu illisible n'en est pas
// une cause : il se traite comme un rejeu absent.
var ErrPartial = errors.New("kickoff: rattrapage partiel")

// Access ouvre les segments de la base partagée : Read pour lire (simulation, révision rangée),
// Write pour un burst d'écriture court. Chaque fonction rend le handle et sa libération.
type Access struct {
	Read  func(ctx context.Context) (*sql.DB, func(), error)
	Write func(ctx context.Context) (*sql.DB, func(), error)
}

// Options règle un rattrapage.
type Options struct {
	TitleSlug string
	// ArtifactPath rend le chemin de l'artefact de rejeu d'un match ; nil pour un titre sans
	// rejeu (aucun coup d'envoi de source `film`).
	ArtifactPath func(matchID string) string
	// DryRun : tout est lu et résolu, rien n'est écrit (segments de lecture seulement).
	DryRun bool
	// LotSize : matchs par burst ; <= 0 vaut DefaultLotSize.
	LotSize int
}

// Backfill recalcule coup d'envoi, source et temps de jeu de tous les matchs de la base et écrit
// ce qui change (sauf en simulation). Rend le bilan ; ErrPartial quand un match a échoué, une autre
// erreur quand un segment n'a pas pu s'ouvrir (le bilan dit alors ce qui a été fait avant).
func Backfill(ctx context.Context, acc Access, o Options) (Report, error) {
	rep := newReport()
	ids, err := listMatches(ctx, acc)
	if err != nil {
		return rep, err
	}
	size := o.LotSize
	if size <= 0 {
		size = DefaultLotSize
	}
	for start := 0; start < len(ids); start += size {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		end := min(start+size, len(ids))
		if err := runLot(ctx, acc, o, ids[start:end], &rep); err != nil {
			return rep, err
		}
	}
	logUnreadable(ctx, o.TitleSlug, rep)
	if rep.Failures > 0 {
		return rep, fmt.Errorf("%w : %d match(s) en échec d'écriture, par exemple %s", ErrPartial, rep.Failures, rep.FailedExample)
	}
	return rep, nil
}

// logUnreadable journalise, une fois par passage, les rejeux présents mais illisibles : combien, et ce que la résolution
// en a fait (coup d'envoi `film` rangé gardé tel quel, ou résolution sur la source suivante comme si le rejeu était absent).
// Rien n'est journalisé quand aucun rejeu n'est illisible.
func logUnreadable(ctx context.Context, titleSlug string, rep Report) {
	if rep.UnreadableArtifacts == 0 {
		return
	}
	slog.WarnContext(ctx, "coup d'envoi : rejeux illisibles, traités comme absents (la révision n'est pas retenue)",
		"titleSlug", titleSlug, "illisibles", rep.UnreadableArtifacts,
		"film_garde", rep.UnreadableKeptFilm, "resolus_sur_la_source_suivante", rep.UnreadableFellBack)
}

// listMatches lit la liste des matchs par un segment de lecture.
func listMatches(ctx context.Context, acc Access) ([]string, error) {
	db, release, err := acc.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("kickoff: segment de lecture: %w", err)
	}
	defer release()
	return matchIDs(ctx, db)
}

// runLot traite un lot : les rejeux se lisent AVANT tout segment (lecture de fichiers), puis un seul
// segment — lecture en simulation, writer sinon — lit l'état, résout et écrit. L'état rangé se lit
// SOUS ce segment : une publication qui a écrit `film` entre la lecture des fichiers et le burst est
// vue, et un rejeu lu absent ne la défait pas (cf. [Resolve]).
func runLot(ctx context.Context, acc Access, o Options, ids []string, rep *Report) error {
	films := readFilms(ctx, o, ids, rep)
	open := acc.Write
	if o.DryRun {
		open = acc.Read
	}
	db, release, err := open(ctx)
	if err != nil {
		return fmt.Errorf("kickoff: segment du lot: %w", err)
	}
	defer release()
	plans, missing, err := prepareIDs(ctx, db, o.TitleSlug, ids, films)
	if err != nil {
		return err
	}
	// Un match disparu du registre entre la liste et le lot n'a rien à rattraper : ce n'est pas un échec.
	rep.Missing += len(missing)
	observability.AddIntT(o.TitleSlug, CompteurAbsentsDuRegistre, int64(len(missing)))
	for _, p := range plans {
		rep.add(p)
		if films[p.MatchID].State == FilmUnreadable {
			rep.addUnreadable(p)
		}
	}
	if o.DryRun {
		return nil
	}
	written, failed := Persist(ctx, db, o.TitleSlug, plans)
	rep.Written += written
	for _, id := range failed {
		rep.fail(id)
	}
	return nil
}

// readFilms lit ce que le rejeu de chaque match du lot dit de son coup d'envoi (cf. [Film]). Un rejeu
// illisible est compté (compteur et bilan) ; la synthèse du passage le journalise UNE fois ([logUnreadable]).
// Un rejeu illisible ne retient jamais la révision : [Resolve] le traite comme un rejeu absent.
func readFilms(ctx context.Context, o Options, ids []string, rep *Report) map[string]Film {
	films := make(map[string]Film, len(ids))
	for _, id := range ids {
		films[id] = Film{State: FilmAbsent}
		if o.ArtifactPath == nil {
			continue
		}
		path := o.ArtifactPath(id)
		ms, found, err := replaybuild.ArtifactKickoff(ctx, path, !o.DryRun)
		switch {
		case err != nil:
			slog.DebugContext(ctx, "coup d'envoi : rejeu illisible, rien de neuf n'en est tiré",
				"titleSlug", o.TitleSlug, "match_id", id, "err", err)
			observability.IncCounterT(o.TitleSlug, CompteurArtefactsIllisibles)
			rep.UnreadableArtifacts++
			films[id] = Film{State: FilmUnreadable}
		case found:
			rep.Artifacts++
			films[id] = FilmPublished(ms)
		}
	}
	return films
}

// ApplyRevision rattrape la base quand la révision rangée diffère de timeline.KickoffRev, puis range
// la nouvelle révision. Rend applied=false sans rien écrire quand elle est à jour. Échec partiel : la
// révision n'est pas rangée, le rattrapage se rejouera au prochain démarrage.
func ApplyRevision(ctx context.Context, acc Access, o Options) (bool, Report, error) {
	stored, err := readStoredRevision(ctx, acc)
	if err != nil {
		return false, Report{}, err
	}
	if stored == timeline.KickoffRev {
		return false, Report{}, nil
	}
	o.DryRun = false
	rep, err := Backfill(ctx, acc, o)
	if err != nil {
		return false, rep, err
	}
	if err := StoreRevision(ctx, acc); err != nil {
		return false, rep, err
	}
	slog.InfoContext(ctx, "coup d'envoi : rattrapage de révision terminé",
		"titleSlug", o.TitleSlug, "avant", stored, "revision", timeline.KickoffRev,
		"matchs", rep.Matches, "ecrits", rep.Written, "joueurs_recalcules", rep.PlayersChanged)
	return true, rep, nil
}

// StoreRevision range timeline.KickoffRev dans `sync_meta`, dans un burst d'écriture ; une base qui la
// porte déjà n'est pas réécrite.
func StoreRevision(ctx context.Context, acc Access) error {
	db, release, err := acc.Write(ctx)
	if err != nil {
		return fmt.Errorf("kickoff: segment d'écriture de la révision: %w", err)
	}
	defer release()
	stored, err := storedRevision(ctx, db)
	if err != nil {
		return err
	}
	if stored == timeline.KickoffRev {
		return nil
	}
	return persist.NewKickoffPersister(db).StoreRevision(ctx, SyncMetaKey, timeline.KickoffRev)
}

func readStoredRevision(ctx context.Context, acc Access) (string, error) {
	db, release, err := acc.Read(ctx)
	if err != nil {
		return "", fmt.Errorf("kickoff: segment de lecture: %w", err)
	}
	defer release()
	return storedRevision(ctx, db)
}
