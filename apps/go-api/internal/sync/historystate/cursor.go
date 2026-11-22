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

// CursorKey est la clé de `sync_meta` qui porte, par joueur, l'état d'un rattrapage d'historique en
// cours : le marque-page (la position où la dernière passe s'est arrêtée dans la liste de l'API) et
// les matchs en échec de rattrapage avec leur nombre de tentatives. Elle vit à côté de [Key] et n'a
// de sens que tant que [Key] est absente : [MarkComplete] l'efface.
//
// Valeur : un objet JSON
//
//		{"offset":N,"last_match_id":"...","saved_at":"RFC3339","failed":[{"id":"...","attempts":N}, ...]}
//
//	  - offset : nombre d'entrées de la liste de l'API (de la plus récente à la plus ancienne)
//	    déjà traitées ; la prochaine entrée à traiter est à ce décalage. >= 1 quand il y a un marque-page ;
//	    absent quand seuls des échecs sont rangés.
//	  - last_match_id : identifiant du dernier match traité, donc de l'entrée à `offset - 1`. Il sert
//	    de preuve de position : la liste de l'API gagne des matchs par le haut, et le décalage rangé
//	    ne vaut que si l'identifiant se retrouve dans la page relue.
//	  - saved_at : instant de l'écriture, pour le diagnostic seulement.
//	  - failed : les matchs que le rattrapage n'a pas réussi à ranger, du plus ancien échec au plus
//	    récent, avec le nombre de passes où ils ont échoué ; au plus [MaxFailures] entrées, les plus
//	    anciennes sortent en premier. L'appelant décide du seuil d'abandon ; ce paquet ne fait que
//	    ranger et relire.
//
// Une valeur vide (état effacé) ou illisible vaut « ni marque-page ni échec » : la passe repart
// du début, ce qui est lent mais sûr. Une valeur illisible est journalisée.
//
// L'effacement range une valeur vide au lieu de supprimer la ligne : aucun DELETE sur `sync_meta`
// (règle anti-ART, la table porte une clé primaire).
const CursorKey = "history_catchup_cursor"

// MaxFailures borne le nombre de matchs en échec rangés avec le marque-page : au-delà, les plus
// anciens échecs sortent de la liste (ils redeviennent des inconnus ordinaires).
const MaxFailures = 200

// Cursor est le marque-page d'un rattrapage d'historique, voir [CursorKey].
type Cursor struct {
	// Offset est le nombre d'entrées de la liste de l'API déjà traitées (>= 1).
	Offset int
	// LastMatchID est l'identifiant du dernier match traité, à l'entrée Offset-1 (non vide).
	LastMatchID string
}

// Valid dit si le marque-page porte une position utilisable : un décalage >= 1 et un identifiant.
func (c Cursor) Valid() bool { return c.Offset >= 1 && strings.TrimSpace(c.LastMatchID) != "" }

// Failure est un match que le rattrapage n'a pas réussi à ranger, avec le nombre de passes où il a
// échoué (>= 1).
type Failure struct {
	MatchID  string
	Attempts int
}

// CatchUp est l'état rangé d'un rattrapage en cours : marque-page (nil = aucun) et matchs en échec
// (du plus ancien au plus récent).
type CatchUp struct {
	Cursor   *Cursor
	Failures []Failure
}

// cursorWire est la forme rangée dans `sync_meta`.
type cursorWire struct {
	Offset      int           `json:"offset,omitempty"`
	LastMatchID string        `json:"last_match_id,omitempty"`
	SavedAt     string        `json:"saved_at,omitempty"`
	Failed      []failureWire `json:"failed,omitempty"`
}

type failureWire struct {
	ID       string `json:"id"`
	Attempts int    `json:"attempts"`
}

// empty dit si la forme rangée ne porte plus rien.
func (w cursorWire) empty() bool {
	return w.Offset == 0 && strings.TrimSpace(w.LastMatchID) == "" && len(w.Failed) == 0
}

// readWire lit et décode la valeur rangée. found=false : clé absente ou valeur vide. Une lecture en
// échec ou une valeur illisible rend ok=false (journalisé) : l'appelant se comporte comme sans état.
func readWire(ctx context.Context, db *sql.DB) (w cursorWire, found, ok bool) {
	if db == nil {
		return cursorWire{}, false, false
	}
	raw, present, err := syncmeta.Get(ctx, db, CursorKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du marque-page d'historique échouée — reprise depuis le début", "err", err)
		return cursorWire{}, false, false
	}
	if !present || strings.TrimSpace(raw) == "" {
		return cursorWire{}, false, true
	}
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		slog.WarnContext(ctx, "sync: marque-page d'historique illisible — reprise depuis le début", "err", err)
		return cursorWire{}, false, false
	}
	return w, true, true
}

// writeWire range la forme (ou une valeur vide quand elle ne porte plus rien).
func writeWire(ctx context.Context, db *sql.DB, w cursorWire) error {
	if w.empty() {
		return syncmeta.Set(ctx, db, CursorKey, "")
	}
	w.SavedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return syncmeta.Set(ctx, db, CursorKey, string(raw))
}

// cursorOf extrait le marque-page d'une forme rangée ; incohérent (un seul des deux champs) =
// aucun, journalisé.
func cursorOf(ctx context.Context, w cursorWire) *Cursor {
	if w.Offset == 0 && strings.TrimSpace(w.LastMatchID) == "" {
		return nil
	}
	c := Cursor{Offset: w.Offset, LastMatchID: w.LastMatchID}
	if !c.Valid() {
		slog.WarnContext(ctx, "sync: marque-page d'historique incohérent — reprise depuis le début",
			"offset", w.Offset, "has_match_id", strings.TrimSpace(w.LastMatchID) != "")
		return nil
	}
	return &c
}

// failuresOf extrait les échecs d'une forme rangée ; une entrée sans identifiant ou sans tentative
// est écartée.
func failuresOf(w cursorWire) []Failure {
	var out []Failure
	for _, f := range w.Failed {
		if strings.TrimSpace(f.ID) == "" || f.Attempts < 1 {
			continue
		}
		out = append(out, Failure{MatchID: f.ID, Attempts: f.Attempts})
	}
	return out
}

// LoadCatchUp lit l'état du rattrapage du joueur : marque-page et échecs. Rien d'exploitable (clé
// absente ou effacée, base absente, lecture en échec ou valeur illisible, ces trois derniers cas
// journalisés) : état vide, l'appelant repart du début de la liste.
func LoadCatchUp(ctx context.Context, db *sql.DB) CatchUp {
	w, found, ok := readWire(ctx, db)
	if !found || !ok {
		return CatchUp{}
	}
	return CatchUp{Cursor: cursorOf(ctx, w), Failures: failuresOf(w)}
}

// LoadCursor lit le marque-page du joueur. ok=false : pas de marque-page ; l'appelant repart du
// début de la liste. Les échecs rangés avec lui ne sont pas lus ici ([LoadCatchUp]).
func LoadCursor(ctx context.Context, db *sql.DB) (Cursor, bool) {
	if c := LoadCatchUp(ctx, db).Cursor; c != nil {
		return *c, true
	}
	return Cursor{}, false
}

// SaveCursor range le marque-page du joueur ; les échecs déjà rangés sont conservés. L'appelant
// tient le bail d'écriture du joueur (voir le doc du paquet). Rend vrai quand il est rangé ; faux
// (journalisé) pour une position invalide, une base absente ou une écriture en échec : l'ancien
// état reste alors en place, la passe suivante reprendra au plus tôt à lui.
func SaveCursor(ctx context.Context, db *sql.DB, c Cursor) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: marque-page d'historique non rangé — base joueur indisponible")
		return false
	}
	if !c.Valid() {
		slog.WarnContext(ctx, "sync: marque-page d'historique invalide — non rangé", "offset", c.Offset)
		return false
	}
	w, _, ok := readWire(ctx, db)
	if !ok {
		w = cursorWire{} // valeur illisible : remplacée, le rattrapage repart de zéro
	}
	w.Offset, w.LastMatchID = c.Offset, c.LastMatchID
	if err := writeWire(ctx, db, w); err != nil {
		slog.WarnContext(ctx, "sync: écriture du marque-page d'historique échouée", "err", err)
		return false
	}
	return true
}

// SaveFailures range la liste des matchs en échec du joueur (du plus ancien au plus récent) ; le
// marque-page déjà rangé est conservé. Au-delà de [MaxFailures] entrées, les plus anciennes sont
// écartées. Une liste vide efface les échecs. L'appelant tient le bail d'écriture du joueur. Rend
// vrai quand l'état est rangé, faux (journalisé) sinon.
func SaveFailures(ctx context.Context, db *sql.DB, failures []Failure) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: échecs de rattrapage non rangés — base joueur indisponible")
		return false
	}
	if over := len(failures) - MaxFailures; over > 0 {
		failures = failures[over:]
	}
	w, _, ok := readWire(ctx, db)
	if !ok {
		w = cursorWire{}
	}
	w.Failed = w.Failed[:0:0]
	for _, f := range failures {
		if strings.TrimSpace(f.MatchID) == "" || f.Attempts < 1 {
			continue
		}
		w.Failed = append(w.Failed, failureWire{ID: f.MatchID, Attempts: f.Attempts})
	}
	if err := writeWire(ctx, db, w); err != nil {
		slog.WarnContext(ctx, "sync: écriture des échecs de rattrapage échouée", "err", err)
		return false
	}
	return true
}

// ClearCursor efface tout l'état du rattrapage du joueur, marque-page et échecs compris (valeur
// vide, voir [CursorKey]). L'appelant tient le bail d'écriture du joueur. Sans état rangé, rien
// n'est écrit. Rend vrai quand il n'y a plus d'état, faux (journalisé) si l'effacement a échoué.
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
