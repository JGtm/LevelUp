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

// SkipSetKey est la clé de `sync_meta` qui porte, par joueur, les matchs que la découverte ne
// demande plus : un match que l'API rend introuvable (404/410) et un match que le rattrapage a
// abandonné après plusieurs passes en échec. Sans cet ensemble, un tel match n'est jamais rangé donc
// jamais « connu » : chaque delta le relit et le redemande tant qu'il reste proche du haut de la
// liste, et il compte à chaque fois comme un match ajouté.
//
// Valeur : un objet JSON
//
//		{"skipped":[{"id":"...","why":"not_found"|"abandoned","at":"RFC3339"}, ...],"saved_at":"RFC3339"}
//
//	  - skipped : du plus ancien au plus récent ajout, au plus [MaxSkipSet] entrées ; au-delà, les plus
//	    anciennes sortent (le match redevient un inconnu ordinaire, redemandé une fois).
//	  - why : la raison. `not_found` est définitif ; `abandoned` ne l'est que [AbandonRetryAfter].
//	  - at : instant de l'entrée, ou de son dernier abandon. Facultatif : une entrée sans date (valeur
//	    rangée avant ce champ) vaut la date de la dernière écriture de l'ensemble (`saved_at`), au plus tard
//	    celle de son abandon ; sans `saved_at` non plus, elle vaut « maintenant » et n'échoit pas.
//	  - saved_at : instant de l'écriture.
//
// Un match de l'ensemble est NEUTRE pour la découverte : ni connu ni inconnu, il ne compte pas dans
// la suite de matchs connus d'affilée qui arrête un delta et ne la remet pas à zéro, n'entre pas dans
// le budget d'une passe et n'est pas demandé. Contrairement au marque-page ([CursorKey]), la fin
// d'historique ([MarkComplete]) ne l'efface pas : un match introuvable le reste après la fin. Un match
// abandonné, lui, ÉCHOIT [AbandonRetryAfter] après son abandon : il sort de l'ensemble actif ([SkipSet.Due])
// et la découverte le retente avec ses tentatives remises à zéro.
//
// Une valeur vide ou illisible vaut « ensemble vide » (la seconde est journalisée, et remplacée à la
// prochaine écriture). Une valeur rangée se remplace, jamais ne se supprime : aucun DELETE sur
// `sync_meta` (règle anti-ART, la table porte une clé primaire).
const SkipSetKey = "history_skip_set"

// MaxSkipSet borne le nombre de matchs de l'ensemble des matchs à ne plus demander.
const MaxSkipSet = 2000

// AbandonRetryAfter est le délai après lequel un match abandonné (Infinite) ou une page abandonnée
// (Halo 5) est retenté : une panne de l'API ou un match refusé un jour ne doit pas laisser un trou
// définitif dans l'historique. Un match introuvable (404/410) n'échoit jamais.
const AbandonRetryAfter = 24 * time.Hour

// Now est l'horloge du paquet (les dates d'abandon et leur échéance) ; les tests la remplacent.
var Now = time.Now

// SkipReason dit pourquoi un match n'est plus demandé.
type SkipReason string

const (
	// SkipNotFound : l'API rend le match introuvable (404/410). Définitif.
	SkipNotFound SkipReason = "not_found"
	// SkipAbandoned : le rattrapage a abandonné le match après plusieurs passes en échec. Retenté
	// après [AbandonRetryAfter].
	SkipAbandoned SkipReason = "abandoned"
)

// Skip est un match à ne plus demander, avec sa raison.
type Skip struct {
	MatchID string
	Reason  SkipReason
}

// SkipSet est l'ensemble des matchs à ne plus demander, lu à un instant.
type SkipSet struct {
	// Active : les matchs que la découverte ne demande pas (introuvables, abandonnés non échus).
	Active map[string]bool
	// Due : les matchs abandonnés depuis [AbandonRetryAfter] ou plus, du plus ancien rangé au plus
	// récent : la découverte les retente, tentatives remises à zéro.
	Due []string
}

// skipSetWire est la forme rangée dans `sync_meta`.
type skipSetWire struct {
	Skipped []skipWire `json:"skipped"`
	SavedAt string     `json:"saved_at,omitempty"`
}

type skipWire struct {
	ID  string `json:"id"`
	Why string `json:"why,omitempty"`
	At  string `json:"at,omitempty"`
}

// readSkipWire lit la valeur rangée. ok=false : la lecture a échoué (rien ne peut être conclu, une
// écriture ne doit pas partir d'un ensemble supposé vide). Une valeur absente, vide ou illisible rend
// un ensemble vide avec ok=true (l'illisible est journalisé).
func readSkipWire(ctx context.Context, db *sql.DB) (w skipSetWire, ok bool) {
	raw, found, err := syncmeta.Get(ctx, db, SkipSetKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture de l'ensemble des matchs à ne plus demander échouée", "err", err)
		return skipSetWire{}, false
	}
	if !found || strings.TrimSpace(raw) == "" {
		return skipSetWire{}, true
	}
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		slog.WarnContext(ctx, "sync: ensemble des matchs à ne plus demander illisible — considéré vide", "err", err)
		return skipSetWire{}, true
	}
	return w, true
}

// at rend la date de l'entrée : la sienne, sinon `saved_at` de l'ensemble ; zéro quand aucune n'est lisible.
func (s skipWire) at(setSavedAt string) time.Time {
	for _, raw := range []string{s.At, setSavedAt} {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

// due dit si l'entrée est un abandon échu à `now`. Une entrée sans date lisible n'échoit pas.
func (s skipWire) due(setSavedAt string, now time.Time) bool {
	if SkipReason(s.Why) != SkipAbandoned {
		return false
	}
	at := s.at(setSavedAt)
	return !at.IsZero() && !now.Before(at.Add(AbandonRetryAfter))
}

// LoadSkipSet lit l'ensemble des matchs que la découverte ne demande plus pour ce joueur, à l'instant
// [Now]. Une base absente, une lecture en échec (journalisée) ou un ensemble vide rendent un ensemble
// vide : la découverte se comporte alors sans lui, au pire elle redemande un match introuvable.
func LoadSkipSet(ctx context.Context, db *sql.DB) SkipSet {
	if db == nil {
		return SkipSet{}
	}
	w, ok := readSkipWire(ctx, db)
	if !ok || len(w.Skipped) == 0 {
		return SkipSet{}
	}
	now := Now()
	out := SkipSet{Active: make(map[string]bool, len(w.Skipped))}
	for _, s := range w.Skipped {
		id := strings.TrimSpace(s.ID)
		switch {
		case id == "":
		case s.due(w.SavedAt, now):
			out.Due = append(out.Due, id)
		default:
			out.Active[id] = true
		}
	}
	return out
}

// AddSkips range des matchs à ne plus demander, à la suite de ceux déjà rangés, datés [Now]. Un match
// déjà présent garde sa place ; l'abandon d'un match déjà abandonné (retenté puis de nouveau abandonné)
// renouvelle sa date ; un match introuvable devient définitif même s'il était abandonné. Au-delà de
// [MaxSkipSet] entrées, les plus anciennes sortent. Les entrées sans date reçoivent celle de la dernière
// écriture, pour ne pas échoir plus tard que prévu. L'appelant tient le bail d'écriture du joueur (voir
// le doc du paquet). Rend vrai quand tous les matchs sont rangés (aussi quand il n'y avait rien de neuf
// à écrire), faux (journalisé) sinon : l'ancien ensemble reste alors en place.
func AddSkips(ctx context.Context, db *sql.DB, adds []Skip) bool {
	if len(adds) == 0 {
		return true
	}
	if db == nil {
		slog.WarnContext(ctx, "sync: matchs à ne plus demander non rangés — base joueur indisponible")
		return false
	}
	w, ok := readSkipWire(ctx, db)
	if !ok {
		return false
	}
	now := Now().UTC().Format(time.RFC3339)
	changed := mergeSkips(&w, adds, now)
	if changed == 0 {
		return true
	}
	if over := len(w.Skipped) - MaxSkipSet; over > 0 {
		w.Skipped = w.Skipped[over:]
	}
	w.SavedAt = now
	raw, err := json.Marshal(w)
	if err != nil {
		slog.WarnContext(ctx, "sync: ensemble des matchs à ne plus demander non sérialisable", "err", err)
		return false
	}
	if err := syncmeta.Set(ctx, db, SkipSetKey, string(raw)); err != nil {
		slog.WarnContext(ctx, "sync: écriture de l'ensemble des matchs à ne plus demander échouée", "err", err)
		return false
	}
	return true
}

// mergeSkips applique `adds` à l'ensemble rangé et rend le nombre d'entrées ajoutées ou modifiées.
func mergeSkips(w *skipSetWire, adds []Skip, now string) int {
	for i := range w.Skipped {
		if w.Skipped[i].At == "" {
			w.Skipped[i].At = w.SavedAt // date de la dernière écriture : au plus tard celle de l'abandon
		}
	}
	index := make(map[string]int, len(w.Skipped)+len(adds))
	for i, s := range w.Skipped {
		index[s.ID] = i
	}
	changed := 0
	for _, s := range adds {
		id := strings.TrimSpace(s.MatchID)
		if id == "" {
			continue
		}
		i, present := index[id]
		if !present {
			index[id] = len(w.Skipped)
			w.Skipped = append(w.Skipped, skipWire{ID: id, Why: string(s.Reason), At: now})
			changed++
			continue
		}
		cur := &w.Skipped[i]
		renewed := s.Reason == SkipAbandoned && SkipReason(cur.Why) == SkipAbandoned
		madeFinal := s.Reason == SkipNotFound && SkipReason(cur.Why) != SkipNotFound
		if renewed || madeFinal {
			cur.Why, cur.At = string(s.Reason), now
			changed++
		}
	}
	return changed
}
