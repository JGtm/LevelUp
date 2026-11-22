// Package historystate porte la règle du rattrapage de l'historique d'un joueur : une seule clé,
// une seule lecture, une seule écriture, un seul budget, pour tous les chemins de synchronisation
// (moteur V1, cycle V2, runner Halo 5).
//
// Règle : un delta s'arrête au premier match connu, donc une première synchronisation interrompue
// (plafond, fermeture du poste, page perdue) laisserait un trou définitif derrière les matchs déjà
// rangés. Tant que la fin de l'historique n'a pas été vue (clé [Key] absente de `sync_meta`), la
// passe saute les matchs connus au lieu de s'arrêter dessus, et range les inconnus plus anciens
// par tranches de [PassBudget] matchs. La clé se range seulement quand la fin a été vue sans perte.
//
// Marque-page : une passe de rattrapage qui s'arrête avant la fin range sa position ([SaveCursor],
// clé [CursorKey], voir [Cursor]) pour que la suivante reprenne là au lieu de reparcourir la liste
// depuis le début ; [MarkComplete] l'efface.
//
// Bail d'écriture : [MarkComplete], [SaveCursor] et [ClearCursor] écrivent dans la base joueur ; l'APPELANT tient le bail d'écriture
// de ce joueur pendant l'appel (règle ART n°4, `dblease.AcquireWriterCtx` / `AcquirePlayerWriterTimeout`),
// ce paquet n'en prend pas. Le moteur V1 appelle [MarkComplete] sous le bail joueur pris au début de
// sa passe, ou, en mode file asynchrone, sous celui qu'il reprend après le drain.
//
// Seul ce paquet nomme ses clés : le garde-rail `archlint.TestHistoryKeyLiteralOnlyInHistorystate`
// refuse leurs littéraux ailleurs.
package historystate

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/syncmeta"
)

const (
	// Key est la clé de `sync_meta` qui porte, par joueur, l'instant où une passe a vu la fin de
	// l'historique du titre. Absente : l'historique n'a jamais été parcouru jusqu'au bout. Le nom
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

// Complete dit si la fin de l'historique a déjà été vue pour ce joueur. Une base absente ou une
// lecture en échec compte comme « complet » (journalisé) : mieux vaut un delta ordinaire qu'un
// parcours intégral à chaque passe sur une base qui ne répond pas.
func Complete(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return true
	}
	_, ok, err := syncmeta.Get(ctx, db, Key)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture de l'état de l'historique échouée — delta ordinaire", "err", err)
		return true
	}
	return ok
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
