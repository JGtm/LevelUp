package livesync

// history_store.go — l'état du rattrapage de l'historique Halo 5 dans la base joueur (historystate).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// HistoryStore lit et range l'état du rattrapage de l'historique d'UN joueur : la fin vue, le
// marque-page et le compte d'échecs (paquet historystate). Les écritures se font sous le bail
// d'écriture du joueur, que l'implémentation prend et relâche à chaque appel.
type HistoryStore interface {
	// Complete dit si la fin de l'historique a été vue. L'erreur signale une base joueur illisible.
	Complete(ctx context.Context) (bool, error)
	// MarkComplete range la fin (et efface le marque-page et le compte d'échecs). Vrai quand elle est rangée.
	MarkComplete(ctx context.Context) bool
	LoadCursor(ctx context.Context) (historystate.Cursor, bool)
	SaveCursor(ctx context.Context, c historystate.Cursor) bool
	LoadStreak(ctx context.Context, subject string) historystate.Streak
	SaveStreak(ctx context.Context, s historystate.Streak) bool
	ClearStreak(ctx context.Context) bool
}

// playerHistory est le HistoryStore de production : la base joueur Halo 5 (handle RW ref-compté de
// sync.OpenPlayerDB, comme les autres hooks du runner).
type playerHistory struct {
	playerDBPath string
	gamertag     string
}

func newPlayerHistory(playerDBPath, gamertag string) playerHistory {
	return playerHistory{playerDBPath: playerDBPath, gamertag: gamertag}
}

// read ouvre la base joueur le temps de fn. Les lectures de historystate journalisent leurs échecs ;
// seule l'ouverture remonte en erreur.
func (h playerHistory) read(fn func(db *sql.DB)) error {
	db, err := syncpkg.OpenPlayerDB(h.playerDBPath)
	if err != nil {
		return fmt.Errorf("h5 history: open player DB: %w", err)
	}
	defer db.Close()
	fn(db.SQLDb())
	return nil
}

// write exécute fn sous le bail d'écriture du joueur (règle ART n°4, même acquisition que le moteur
// V1), attendu au plus dblease.PlayerLeaseTimeout : le cycle ne reste pas bloqué sur un bail tenu
// ailleurs, la passe suivante réessaie. Rend vrai quand fn a rangé ce qu'elle devait.
func (h playerHistory) write(ctx context.Context, what string, fn func(db *sql.DB) bool) bool {
	leaseCtx, cancel := context.WithTimeout(ctx, dblease.PlayerLeaseTimeout)
	defer cancel()
	lease, err := dblease.AcquireWriterCtx(leaseCtx, nil, h.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.WarnContext(ctx, "h5 sync: bail joueur indisponible — état de l'historique non rangé",
			"gamertag", h.gamertag, "what", what, "err", err)
		return false
	}
	defer lease.Release()
	db, err := syncpkg.OpenPlayerDB(h.playerDBPath)
	if err != nil {
		slog.WarnContext(ctx, "h5 sync: ouverture de la base joueur échouée — état de l'historique non rangé",
			"gamertag", h.gamertag, "what", what, "err", err)
		return false
	}
	defer db.Close()
	return fn(db.SQLDb())
}

func (h playerHistory) Complete(ctx context.Context) (bool, error) {
	complete := true
	err := h.read(func(db *sql.DB) { complete = historystate.Complete(ctx, db) })
	return complete, err
}

func (h playerHistory) MarkComplete(ctx context.Context) bool {
	return h.write(ctx, "fin", func(db *sql.DB) bool {
		if !historystate.MarkComplete(ctx, db, h.gamertag) {
			return false
		}
		historystate.ClearStreak(ctx, db)
		return true
	})
}

func (h playerHistory) LoadCursor(ctx context.Context) (c historystate.Cursor, ok bool) {
	if err := h.read(func(db *sql.DB) { c, ok = historystate.LoadCursor(ctx, db) }); err != nil {
		slog.WarnContext(ctx, "h5 sync: marque-page illisible — reprise depuis le début", "gamertag", h.gamertag, "err", err)
		return historystate.Cursor{}, false
	}
	return c, ok
}

func (h playerHistory) SaveCursor(ctx context.Context, c historystate.Cursor) bool {
	return h.write(ctx, "marque-page", func(db *sql.DB) bool { return historystate.SaveCursor(ctx, db, c) })
}

func (h playerHistory) LoadStreak(ctx context.Context, subject string) (s historystate.Streak) {
	s = historystate.Streak{Subject: subject}
	if err := h.read(func(db *sql.DB) { s = historystate.LoadStreak(ctx, db, subject) }); err != nil {
		slog.WarnContext(ctx, "h5 sync: compte d'échecs illisible — remis à zéro", "gamertag", h.gamertag, "err", err)
	}
	return s
}

func (h playerHistory) SaveStreak(ctx context.Context, s historystate.Streak) bool {
	return h.write(ctx, "streak", func(db *sql.DB) bool { return historystate.SaveStreak(ctx, db, s) })
}

func (h playerHistory) ClearStreak(ctx context.Context) bool {
	return h.write(ctx, "streak", func(db *sql.DB) bool { return historystate.ClearStreak(ctx, db) })
}
