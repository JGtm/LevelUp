package historystate

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/sync/syncmeta"
)

// CursorKey est la clé de `sync_meta` qui porte, par joueur, le marque-page d'un rattrapage
// d'historique en cours : la position où la dernière passe s'est arrêtée dans la liste de l'API.
// Elle vit à côté de [Key] et n'a de sens que tant que [Key] est absente : [MarkComplete] l'efface.
//
// Valeur : un objet JSON `{"offset":N,"last_match_id":"...","saved_at":"RFC3339"}`.
//   - offset : nombre d'entrées de la liste de l'API (de la plus récente à la plus ancienne)
//     déjà traitées ; la prochaine entrée à traiter est à ce décalage. Toujours >= 1.
//   - last_match_id : identifiant du dernier match traité, donc de l'entrée à `offset - 1`. Il sert
//     de preuve de position : la liste de l'API gagne des matchs par le haut, et le décalage rangé
//     ne vaut que si l'identifiant se retrouve dans la page relue.
//   - saved_at : instant de l'écriture, pour le diagnostic seulement.
//
// Une valeur vide (marque-page effacé) ou illisible vaut « pas de marque-page » : la passe repart
// du début, ce qui est lent mais sûr. Une valeur illisible est journalisée.
//
// L'effacement range une valeur vide au lieu de supprimer la ligne : aucun DELETE sur `sync_meta`
// (règle anti-ART, la table porte une clé primaire).
const CursorKey = "history_catchup_cursor"

// Cursor est le marque-page d'un rattrapage d'historique, voir [CursorKey].
type Cursor struct {
	// Offset est le nombre d'entrées de la liste de l'API déjà traitées (>= 1).
	Offset int
	// LastMatchID est l'identifiant du dernier match traité, à l'entrée Offset-1 (non vide).
	LastMatchID string
}

// Valid dit si le marque-page porte une position utilisable : un décalage >= 1 et un identifiant.
func (c Cursor) Valid() bool { return c.Offset >= 1 && strings.TrimSpace(c.LastMatchID) != "" }

// cursorWire est la forme rangée dans `sync_meta`.
type cursorWire struct {
	Offset      int    `json:"offset"`
	LastMatchID string `json:"last_match_id"`
	SavedAt     string `json:"saved_at,omitempty"`
}

// LoadCursor lit le marque-page du joueur. ok=false : pas de marque-page (clé absente ou effacée, base
// absente, lecture en échec ou valeur illisible, ces trois derniers cas journalisés) ; l'appelant
// repart du début de la liste.
func LoadCursor(ctx context.Context, db *sql.DB) (Cursor, bool) {
	if db == nil {
		return Cursor{}, false
	}
	raw, found, err := syncmeta.Get(ctx, db, CursorKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du marque-page d'historique échouée — reprise depuis le début", "err", err)
		return Cursor{}, false
	}
	if !found || strings.TrimSpace(raw) == "" {
		return Cursor{}, false
	}
	var w cursorWire
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		slog.WarnContext(ctx, "sync: marque-page d'historique illisible — reprise depuis le début", "err", err)
		return Cursor{}, false
	}
	c := Cursor{Offset: w.Offset, LastMatchID: w.LastMatchID}
	if !c.Valid() {
		slog.WarnContext(ctx, "sync: marque-page d'historique incohérent — reprise depuis le début",
			"offset", w.Offset, "has_match_id", strings.TrimSpace(w.LastMatchID) != "")
		return Cursor{}, false
	}
	return c, true
}

// SaveCursor range le marque-page du joueur. L'appelant tient le bail d'écriture du joueur (voir le
// doc du paquet). Rend vrai quand il est rangé ; faux (journalisé) pour une position invalide, une
// base absente ou une écriture en échec : l'ancien marque-page reste alors en place, la passe
// suivante reprendra au plus tôt à lui.
func SaveCursor(ctx context.Context, db *sql.DB, c Cursor) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: marque-page d'historique non rangé — base joueur indisponible")
		return false
	}
	if !c.Valid() {
		slog.WarnContext(ctx, "sync: marque-page d'historique invalide — non rangé", "offset", c.Offset)
		return false
	}
	raw, err := json.Marshal(cursorWire{
		Offset:      c.Offset,
		LastMatchID: c.LastMatchID,
		SavedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.WarnContext(ctx, "sync: marque-page d'historique non sérialisable", "err", err)
		return false
	}
	if err := syncmeta.Set(ctx, db, CursorKey, string(raw)); err != nil {
		slog.WarnContext(ctx, "sync: écriture du marque-page d'historique échouée", "err", err)
		return false
	}
	return true
}

// ClearCursor efface le marque-page du joueur (valeur vide, voir [CursorKey]). L'appelant tient le
// bail d'écriture du joueur. Sans marque-page rangé, rien n'est écrit. Rend vrai quand il n'y a plus
// de marque-page, faux (journalisé) si l'effacement a échoué.
func ClearCursor(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}
	raw, found, err := syncmeta.Get(ctx, db, CursorKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du marque-page d'historique échouée avant effacement", "err", err)
		return false
	}
	if !found || strings.TrimSpace(raw) == "" {
		return true
	}
	if err := syncmeta.Set(ctx, db, CursorKey, ""); err != nil {
		slog.WarnContext(ctx, "sync: effacement du marque-page d'historique échoué", "err", err)
		return false
	}
	return true
}
