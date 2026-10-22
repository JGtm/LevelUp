// Package analysis — home_tiles_window.go : ordre et fenêtrage des tuiles de match de
// l'Accueil (D-8). La tête de la page et les pages suivantes lisent la MÊME liste ordonnée par
// la MÊME fonction : une tuile ne peut ni manquer ni se répéter d'une page à l'autre.
package analysis

import (
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// HomeTileCursor est une position dans la liste des tuiles : l'instant canonique du match
// (Summary.StartedAtUTC, lu par le fragment SQL partagé de la règle 8) puis son match_id. Il
// désigne la DERNIÈRE tuile déjà servie ; la page suivante commence strictement après lui. Un
// curseur plutôt qu'un décalage : une synchronisation entre deux pages ajoute des matchs en tête,
// elle ne déplace aucune position relative.
type HomeTileCursor struct {
	At      time.Time
	MatchID string
}

// homeTileCursorSep sépare l'instant du match_id dans la forme texte ; l'instant (RFC 3339) n'en
// contient jamais, le match_id peut en contenir (on coupe au premier).
const homeTileCursorSep = "_"

// Encode rend la forme texte opaque du curseur (renvoyée au client, rendue telle quelle).
func (c HomeTileCursor) Encode() string {
	return c.At.UTC().Format(time.RFC3339Nano) + homeTileCursorSep + c.MatchID
}

// ParseHomeTileCursor lit un curseur. Une chaîne vide est le début de la liste (ok, curseur
// nul) ; une chaîne mal formée est refusée (ok = false), jamais interprétée en silence.
func ParseHomeTileCursor(s string) (HomeTileCursor, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return HomeTileCursor{}, true
	}
	instant, id, found := strings.Cut(s, homeTileCursorSep)
	if !found || id == "" {
		return HomeTileCursor{}, false
	}
	at, err := time.Parse(time.RFC3339Nano, instant)
	if err != nil {
		return HomeTileCursor{}, false
	}
	return HomeTileCursor{At: at, MatchID: id}, true
}

// IsZero : curseur du début de la liste.
func (c HomeTileCursor) IsZero() bool { return c.MatchID == "" }

// precedes dit si la ligne r vient APRÈS le curseur dans l'ordre d'affichage (plus ancienne, ou
// au même instant avec un match_id plus petit).
func (c HomeTileCursor) precedes(r *canonical.PlayerMatchRow) bool {
	if r.Summary.StartedAtUTC.Before(c.At) {
		return true
	}
	return r.Summary.StartedAtUTC.Equal(c.At) && r.Summary.MatchID < c.MatchID
}

// OrderHomeTileRows rend les lignes qui font une tuile (match_id non vide) dans l'ordre
// d'affichage : instant canonique décroissant, puis match_id décroissant. Rend la tranche reçue
// telle quelle quand elle est déjà dans cet ordre (cas courant : le chargeur trie sur le même
// instant), sinon une copie triée — la tranche de l'appelant n'est jamais réordonnée, les totaux
// de la page (héros, faits marquants, sessions) continuent de lire l'ordre d'origine.
func OrderHomeTileRows(rows []canonical.PlayerMatchRow) []canonical.PlayerMatchRow {
	kept := rows
	for i := range rows {
		if rows[i].Summary.MatchID == "" {
			kept = make([]canonical.PlayerMatchRow, 0, len(rows))
			for j := range rows {
				if rows[j].Summary.MatchID != "" {
					kept = append(kept, rows[j])
				}
			}
			break
		}
	}
	if sort.SliceIsSorted(kept, func(i, j int) bool { return homeTileBefore(&kept[i], &kept[j]) }) {
		return kept
	}
	out := make([]canonical.PlayerMatchRow, len(kept))
	copy(out, kept)
	sort.SliceStable(out, func(i, j int) bool { return homeTileBefore(&out[i], &out[j]) })
	return out
}

// homeTileBefore : a s'affiche avant b.
func homeTileBefore(a, b *canonical.PlayerMatchRow) bool {
	if !a.Summary.StartedAtUTC.Equal(b.Summary.StartedAtUTC) {
		return a.Summary.StartedAtUTC.After(b.Summary.StartedAtUTC)
	}
	return a.Summary.MatchID > b.Summary.MatchID
}

// HomeTileWindow est une tranche de la liste ordonnée.
type HomeTileWindow struct {
	Rows []canonical.PlayerMatchRow
	// HasMore : des lignes plus anciennes suivent la dernière de Rows.
	HasMore bool
	// NextBefore : curseur de la dernière ligne de Rows ; vide si Rows l'est.
	NextBefore string
}

// WindowHomeTileRows rend au plus `limit` lignes de `ordered` (sortie d'OrderHomeTileRows)
// strictement après le curseur. La tranche rendue partage ses lignes avec `ordered`.
func WindowHomeTileRows(ordered []canonical.PlayerMatchRow, after HomeTileCursor, limit int) HomeTileWindow {
	start := 0
	if !after.IsZero() {
		start = sort.Search(len(ordered), func(i int) bool { return after.precedes(&ordered[i]) })
	}
	end := start + limit
	if limit < 0 || end > len(ordered) {
		end = len(ordered)
	}
	w := HomeTileWindow{Rows: ordered[start:end], HasMore: end < len(ordered)}
	if n := len(w.Rows); n > 0 {
		last := &w.Rows[n-1]
		w.NextBefore = HomeTileCursor{At: last.Summary.StartedAtUTC, MatchID: last.Summary.MatchID}.Encode()
	}
	return w
}

// FavoriteHomeTileRows rend, dans l'ordre d'affichage, les lignes de `ordered` dont le match est
// un favori du joueur.
func FavoriteHomeTileRows(ordered []canonical.PlayerMatchRow, favoriteIDs map[string]bool) []canonical.PlayerMatchRow {
	if len(favoriteIDs) == 0 {
		return nil
	}
	var out []canonical.PlayerMatchRow
	for i := range ordered {
		if favoriteIDs[ordered[i].Summary.MatchID] {
			out = append(out, ordered[i])
		}
	}
	return out
}
