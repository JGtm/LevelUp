package killcollector

// film_absent_reconcile.go — LE MARQUEUR « FILM ABSENT » NE SURVIT PAS À L'ARRIVÉE DU FILM.
//
// `MBitFilmAbsent` est terminal : il sort le match de la file de décodage et de celle des rejeux, et
// ses lecteurs le disent « analyse terminée, film perdu ». Mesuré le 2026-11-24 : 36 matchs portaient
// la marque alors que leur film était à l'archive (34 avec des morceaux), dont 33 avec un rejeu
// généré. Le film est arrivé après la marque (autre voie d'archivage, restauration, copie), et rien
// ne la retirait.
//
// DEUX GESTES, UN INVARIANT : « un match dont le film est complet à l'archive n'est pas marqué
// absent ».
//   - [ReconcileFilmAbsent] retire la marque des matchs marqués dont le film est complet
//     (filmcache.FilmComplet : manifeste fini ET morceaux présents) ; le match revient dans la file
//     par les voies existantes (le décodage de la passe film, les rattrapages de rejeu).
//   - [KillSourceCollector.AvecFilmComplet] : la passe ne pose plus la marque sur un match dont le film
//     est complet à l'archive (un « absent » constaté dans une lecture partielle ne dit rien du film).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/matchflags"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// CompteurFilmAbsentRetires : marques « film absent » retirées parce que le film est complet à l'archive.
const CompteurFilmAbsentRetires = "killsource_film_absent_retires"

// FilmCompletFn dit si le film du match d'identifiant court `short` est complet sous `root`.
type FilmCompletFn func(root, short string) (bool, error)

// ReconcileFilmAbsentDeps : ce dont la réconciliation a besoin, passé en paramètres.
type ReconcileFilmAbsentDeps struct {
	TitleSlug string
	// ArchiveRoot : la racine de l'archive des films. Vide : rien à réconcilier.
	ArchiveRoot string
	// WithRead ouvre un segment de LECTURE shared court (jamais le writer).
	WithRead func(ctx context.Context, step string, fn func(sharedDB *sql.DB))
	// AcquireWriter : le segment d'écriture, pris seulement s'il y a des marques à retirer.
	AcquireWriter persist.SharedWriterFn
	// Complete : nil = filmcache.FilmComplet.
	Complete FilmCompletFn
}

// ReconcileFilmAbsent retire `MBitFilmAbsent` des matchs marqués dont le film est complet à l'archive.
// Rend le nombre de marques retirées. Journalise ce qu'elle retire ; idempotente.
func ReconcileFilmAbsent(ctx context.Context, d ReconcileFilmAbsentDeps) (int, error) {
	if d.ArchiveRoot == "" || d.WithRead == nil || d.AcquireWriter == nil {
		return 0, nil
	}
	var marques []string
	var lecture error
	d.WithRead(ctx, "film_absent_reconcile", func(sharedDB *sql.DB) {
		marques, lecture = matchsMarquesAbsents(ctx, sharedDB)
	})
	if lecture != nil {
		return 0, fmt.Errorf("réconciliation film absent: lecture des marques: %w", lecture)
	}
	return retirerMarquesAbsentes(ctx, d, marques)
}

// retirerMarquesAbsentes retire la marque des matchs de `marques` dont le film est complet à l'archive.
func retirerMarquesAbsentes(ctx context.Context, d ReconcileFilmAbsentDeps, marques []string) (int, error) {
	complete := d.Complete
	if complete == nil {
		complete = filmcache.FilmComplet
	}
	var aRetirer []string
	for _, id := range marques {
		ok, err := complete(d.ArchiveRoot, titlePkg.FilmShortMatchID(id))
		if err != nil {
			slog.WarnContext(ctx, "killsource: état du film à l'archive illisible, marque gardée",
				"titleSlug", d.TitleSlug, "match_id", id, "err", err)
			continue
		}
		if ok {
			aRetirer = append(aRetirer, id)
		}
	}
	if len(aRetirer) == 0 {
		return 0, nil
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("réconciliation film absent: écrivain indisponible: %w", err)
	}
	defer release()
	n, err := persist.NewFilmAbsentPersister(db).ClearBit(ctx, aRetirer, int64(matchflags.MBitFilmAbsent))
	if err != nil {
		return 0, fmt.Errorf("réconciliation film absent: %w", err)
	}
	observability.AddIntT(d.TitleSlug, CompteurFilmAbsentRetires, int64(n))
	slog.InfoContext(ctx, "killsource: marques « film absent » retirées, le film est complet à l'archive",
		"titleSlug", d.TitleSlug, "marques", len(marques), "retirees", n)
	return n, nil
}

// matchsMarquesAbsents liste les matchs qui portent MBitFilmAbsent.
func matchsMarquesAbsents(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT match_id FROM match_registry
		WHERE (COALESCE(backfill_completed, 0) & CAST(? AS BIGINT)) <> 0 ORDER BY match_id`,
		int64(matchflags.MBitFilmAbsent))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AvecFilmComplet installe le contrôle « le film est-il complet à l'archive ? » que la passe
// consulte avant de poser MBitFilmAbsent : un match dont le film est complet n'est pas marqué.
// Sans lui (backfill hors ligne), la pose est inchangée.
func (c *KillSourceCollector) AvecFilmComplet(fn func(matchID string) bool) *KillSourceCollector {
	c.filmComplet = fn
	return c
}

// reconciliationsFaites : les titres dont la réconciliation a réussi dans ce processus.
var reconciliationsFaites sync.Map

// reconciliationAFaire dit si la réconciliation de ce titre reste à jouer dans ce processus.
func reconciliationAFaire(titleSlug string) bool {
	_, faite := reconciliationsFaites.Load(titleSlug)
	return !faite
}

// reconcilierFilmsAbsents retire les marques de `marques` (lues par la passe) dont le film est
// complet à l'archive, UNE fois par (processus, titre) : un film arrive à l'archive entre deux
// démarrages, jamais pendant un cycle. Une lecture sans marque (nil) ne dit rien et ne clôt pas la
// réconciliation ; un échec n'est pas retenu (repris au cycle suivant) et ne casse pas la passe.
func reconcilierFilmsAbsents(ctx context.Context, d PostSyncDeps, racine string, marques []string) {
	if racine == "" || !reconciliationAFaire(d.TitleSlug) {
		return
	}
	if _, err := retirerMarquesAbsentes(ctx, ReconcileFilmAbsentDeps{
		TitleSlug: d.TitleSlug, ArchiveRoot: racine, WithRead: d.WithRead, AcquireWriter: d.AcquireWriter,
	}, marques); err != nil {
		slog.WarnContext(ctx, "post-sync: réconciliation des marques film absent échouée, reprise au cycle suivant",
			"titleSlug", d.TitleSlug, "err", err)
		return
	}
	if marques != nil {
		reconciliationsFaites.Store(d.TitleSlug, struct{}{})
	}
}

// lireArriere lit le backlog de l etape par un segment de LECTURE court ; quand la reconciliation des
// marques « film absent » n est pas encore faite pour ce titre, il rend aussi les marques, lues dans le
// meme segment.
func (h *PostSyncHook) lireArriere(ctx context.Context, d PostSyncDeps) (backlog []string, total int, marques []string) {
	reconcilier := reconciliationAFaire(d.TitleSlug) && h.racineVoulue(d.LocalCache) != ""
	d.WithRead(ctx, "killsource_select", func(sharedDB *sql.DB) {
		backlog, total = backlogAJour(ctx, sharedDB, h.horizon, 0)
		if reconcilier {
			var err error
			if marques, err = matchsMarquesAbsents(ctx, sharedDB); err != nil {
				slog.WarnContext(ctx, "post-sync: marques film absent illisibles, reconciliation reprise au cycle suivant",
					"titleSlug", d.TitleSlug, "err", err)
				marques = nil
			}
		}
	})
	return backlog, total, marques
}
