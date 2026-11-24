package historystate

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/sync/syncmeta"
)

// HoleKey est la clé de `sync_meta` qui porte, par joueur, l'instant où le rattrapage Halo 5 a abandonné
// une page de la liste (trou connu de l'historique) : un horodatage RFC3339 ; absente, vide ou
// illisible = aucun trou connu. Halo 5 n'a pas de match à redemander par identifiant : une page abandonnée
// ne se retente qu'en relisant la liste. Passé [AbandonRetryAfter], [Reopen] rouvre l'historique : la
// passe suivante repart du début de la liste, saute les matchs connus et revoit la page. Le trou le plus
// ancien date l'échéance : un second trou ne la repousse pas ([NoteHole]).
//
// Comme les autres clés, l'effacement range une valeur vide au lieu de supprimer la ligne (règle anti-ART).
const HoleKey = "history_hole_since"

// holeSince lit l'instant du trou le plus ancien. ok=false : aucun trou connu (clé absente, vide ou
// illisible) ou lecture en échec (journalisée par l'appelant).
func holeSince(ctx context.Context, db *sql.DB) (since time.Time, ok bool, err error) {
	raw, found, err := syncmeta.Get(ctx, db, HoleKey)
	if err != nil || !found || strings.TrimSpace(raw) == "" {
		return time.Time{}, false, err
	}
	t, perr := time.Parse(time.RFC3339, raw)
	if perr != nil {
		slog.WarnContext(ctx, "sync: instant du trou d'historique illisible — ignoré", "err", perr)
		return time.Time{}, false, nil
	}
	return t, true, nil
}

// NoteHole range l'instant [Now] comme début d'un trou d'historique, sauf si un trou plus ancien est déjà
// noté. L'appelant tient le bail d'écriture du joueur. Rend vrai quand l'instant est rangé (ou déjà
// là), faux (journalisé) sinon.
func NoteHole(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: trou d'historique non noté — base joueur indisponible")
		return false
	}
	if _, ok, err := holeSince(ctx, db); err != nil {
		slog.WarnContext(ctx, "sync: lecture du trou d'historique échouée — non noté", "err", err)
		return false
	} else if ok {
		return true
	}
	if err := syncmeta.Set(ctx, db, HoleKey, Now().UTC().Format(time.RFC3339)); err != nil {
		slog.WarnContext(ctx, "sync: écriture du trou d'historique échouée", "err", err)
		return false
	}
	return true
}

// HoleDue dit si un trou d'historique date de [AbandonRetryAfter] ou plus. Une lecture en échec rend
// faux (journalisée) : rien ne se rouvre sur un état illisible.
func HoleDue(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}
	since, ok, err := holeSince(ctx, db)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du trou d'historique échouée", "err", err)
		return false
	}
	return ok && !Now().Before(since.Add(AbandonRetryAfter))
}

// Reopen rouvre l'historique du joueur : la fin n'est plus vue, le marque-page, le compte d'échecs et le
// trou noté sont effacés, de sorte que la passe suivante relit la liste depuis son début (les matchs
// connus sont sautés). L'appelant tient le bail d'écriture du joueur. Rend vrai quand la fin est
// retirée, faux (journalisé) sinon.
func Reopen(ctx context.Context, db *sql.DB, gamertag string) bool {
	if db == nil {
		return false
	}
	if err := syncmeta.Set(ctx, db, Key, ""); err != nil {
		slog.WarnContext(ctx, "sync: réouverture de l'historique échouée", "gamertag", gamertag, "err", err)
		return false
	}
	ClearCursor(ctx, db)
	ClearStreak(ctx, db)
	if err := syncmeta.Set(ctx, db, HoleKey, ""); err != nil {
		slog.WarnContext(ctx, "sync: effacement du trou d'historique échoué", "gamertag", gamertag, "err", err)
	}
	slog.InfoContext(ctx, "sync: historique rouvert pour retenter une page abandonnée", "gamertag", gamertag)
	return true
}
