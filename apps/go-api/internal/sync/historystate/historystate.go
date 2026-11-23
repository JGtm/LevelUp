// Package historystate porte la règle du rattrapage de l'historique d'un joueur : une seule clé,
// une seule lecture, une seule écriture, un seul budget, pour les deux chemins qui rattrapent (cycle V2, runner Halo 5).
// Le moteur V1 est un delta pur : il ne lit ni n'écrit cet état (il n'emprunte que [MaxDepth], pour le
// parcours de RunFull).
//
// Règle : un delta s'arrête après domain.DeltaKnownRun matchs connus d'affilée, donc une première
// synchronisation interrompue (plafond, fermeture du poste, page perdue) laisserait un trou définitif
// derrière les matchs déjà rangés. Tant que la fin de l'historique n'a pas été vue (clé [Key] absente de `sync_meta`), la
// passe saute les matchs connus au lieu de s'arrêter dessus, et range les inconnus plus anciens
// par tranches de [PassBudget] matchs. La clé se range seulement quand la fin a été vue sans perte.
//
// État illisible : une lecture de la clé en échec n'est jamais prise pour « complet » ([Complete]
// rend l'erreur) : la passe fait un delta ordinaire, n'écrit rien de l'état et reste « en attente ».
//
// Marque-page : une passe de rattrapage qui s'arrête avant la fin range sa position ([SaveCursor],
// clé [CursorKey], voir [Cursor]) pour que la suivante reprenne là au lieu de reparcourir la liste
// depuis le début ; [MarkComplete] l'efface.
//
// Abandon : un match (Infinite) ou une page (Halo 5) qui échoue plusieurs passes de suite est abandonné pour ne
// pas garder le joueur au rattrapage à vie, mais pas pour toujours : [AbandonRetryAfter] plus tard, il est
// retenté ([SkipSet.Due] pour un match, [HoleDue] puis [Reopen] pour une page). Un 404/410 reste exclu.
//
// Bail d'écriture : [MarkComplete], [SaveCursor] et [ClearCursor] écrivent dans la base joueur ; l'APPELANT tient le bail d'écriture
// de ce joueur pendant l'appel (règle ART n°4, `dblease.AcquireWriterCtx` / `AcquirePlayerWriterTimeout`),
// ce paquet n'en prend pas. Le cycle V2 et le runner Halo 5 l'appellent sous le bail joueur de leur passe.
//
// Seul ce paquet nomme ses clés : le garde-rail `archlint.TestHistoryKeyLiteralOnlyInHistorystate`
// refuse leurs littéraux ailleurs.
package historystate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/syncmeta"
)

const (
	// Key est la clé de `sync_meta` qui porte, par joueur, l'instant où une passe a vu la fin de
	// l'historique du titre. Absente ou vide : l'historique n'a jamais été parcouru jusqu'au bout (ou a
	// été rouvert par [Reopen]). Le nom
	// est inchangé depuis son introduction : un joueur déjà marqué le reste.
	Key = "history_complete_at"

	// MaxDepth borne la profondeur d'un parcours de l'historique : au-delà, la passe s'arrête avec
	// un avertissement, sans ranger la fin d'historique. Très au-dessus du plus grand historique
	// connu (7 000 matchs), le plus profond parcours réel reste donc complet.
	MaxDepth = 50_000
)

// PassBudget est le nombre de matchs inconnus qu'une passe de rattrapage range au plus : le
// plafond par défaut d'une passe de synchronisation, sans valeur propre.
func PassBudget() int {
	return domain.DefaultSyncOptions().MaxMatches
}

// ErrNoPlayerDB : Complete a reçu une base joueur absente (handle nil).
var ErrNoPlayerDB = errors.New("historystate: base joueur absente")

// Complete dit si la fin de l'historique a déjà été vue pour ce joueur : (vrai, nil) clé rangée,
// (faux, nil) clé absente ou vide ([Reopen] la vide). Une lecture en échec (ou une base absente, ErrNoPlayerDB) rend
// (faux, erreur) : l'état est INCONNU, jamais « complet ». L'appelant ne rattrape pas et n'écrit
// rien de l'état à cette passe (delta ordinaire), garde le joueur « en attente » pour que la
// cadence courte réessaie, et journalise l'erreur.
func Complete(ctx context.Context, db *sql.DB) (bool, error) {
	if db == nil {
		return false, ErrNoPlayerDB
	}
	value, ok, err := syncmeta.Get(ctx, db, Key)
	if err != nil {
		return false, fmt.Errorf("historystate: lecture de %s: %w", Key, err)
	}
	return ok && strings.TrimSpace(value) != "", nil
}

// MarkComplete range l'instant de fin d'historique dans la base joueur. L'appelant tient le bail
// d'écriture du joueur (voir le doc du paquet). Un échec est journalisé : la passe suivante
// reparcourra l'historique (coût borné, les matchs connus sont sautés). Rend vrai quand la clé est
// rangée, faux sinon : l'appelant s'en sert pour dire si la passe laisse l'historique en attente.
func MarkComplete(ctx context.Context, db *sql.DB, gamertag string) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: fin d'historique vue mais base joueur indisponible — non rangée", "gamertag", gamertag)
		return false
	}
	if err := syncmeta.Set(ctx, db, Key, time.Now().UTC().Format(time.RFC3339)); err != nil {
		slog.WarnContext(ctx, "sync: écriture de la fin d'historique échouée", "gamertag", gamertag, "err", err)
		return false
	}
	// La fin rangée prime sur le marque-page : un effacement en échec (journalisé par ClearCursor)
	// laisse une valeur que plus aucune passe ne lit.
	ClearCursor(ctx, db)
	slog.InfoContext(ctx, "sync: fin de l'historique atteinte — les passes suivantes seront des deltas", "gamertag", gamertag)
	return true
}
