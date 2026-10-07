// Package persist — registry_names_persister.go : réinscription des NOMS d'assets d'un match dans
// match_registry (map_name, pair_name, playlist_name, game_variant_name).
//
// # POURQUOI CETTE ÉCRITURE VIT ICI
//
// `match_registry` est une table « match-of-record » : `internal/sync/shared_write_guard_test.go`
// n'en autorise l'écriture que depuis ce package (ou une courte allowlist legacy). Même forme que
// `t0_film_persister.go` et `killsource_sans_killfeed_persister.go`.
//
// # CE QUE LE PERSISTER GARANTIT
//
// Il ne remplace JAMAIS un vrai nom : chaque UPDATE porte la garde
// `WHERE match_id = ? AND (<col> IS NULL OR <col> = <col_id>)`. Un nom déjà juste reste en
// place ; réécrire un match déjà convergé n'affecte aucune ligne (idempotent). Il refuse un nom
// vide. Il ne touche ni `mode_category` (indexée) ni aucune autre colonne.
//
// # ART-SAFETY
//
// UN `UPDATE ... WHERE match_id = ?` par colonne et par match, tous ceux d'un match dans UNE
// transaction, sous le writer exclusif de l'appelant — la forme autorisée par
// `no_art_patterns_test.go`. Les quatre colonnes de nom ne sont pas indexées. Les quatre
// statements sont des littéraux fermés (aucun nom de colonne interpolé).
//
// # CE QUE CE PERSISTER NE DÉCIDE PAS
//
// Ni quel nom écrire, ni pour quels matchs : l'appelant (sync.BackfillRegistryNames) a lu les
// traductions et planifié les écritures. Ce fichier ÉCRIT.
package persist

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Les colonnes de nom réinscriptibles, désignées par le genre d'asset (mêmes valeurs que
// games.AssetKind*).
const (
	RegistryNamePlaylist    = "playlist"
	RegistryNameMap         = "map"
	RegistryNamePair        = "pair"
	RegistryNameGameVariant = "game_variant"
)

// registryNameUpdates : un statement fermé par colonne, gardé sur « nom absent ou égal à
// l'identifiant ».
var registryNameUpdates = map[string]string{
	RegistryNamePlaylist: `UPDATE match_registry SET playlist_name = ?
		WHERE match_id = ? AND (playlist_name IS NULL OR playlist_name = playlist_id)`,
	RegistryNameMap: `UPDATE match_registry SET map_name = ?
		WHERE match_id = ? AND (map_name IS NULL OR map_name = map_id)`,
	RegistryNamePair: `UPDATE match_registry SET pair_name = ?
		WHERE match_id = ? AND (pair_name IS NULL OR pair_name = pair_id)`,
	RegistryNameGameVariant: `UPDATE match_registry SET game_variant_name = ?
		WHERE match_id = ? AND (game_variant_name IS NULL OR game_variant_name = game_variant_id)`,
}

// RegistryNameWrite : le nom à inscrire dans une colonne (Kind = RegistryName*) d'un match.
type RegistryNameWrite struct {
	Kind string
	Name string
}

// RegistryNamesPersister réinscrit les noms d'assets d'un match. `db` doit porter un write lease
// actif sur shared_matches_v2.duckdb.
type RegistryNamesPersister struct {
	db txBeginner
}

// NewRegistryNamesPersister construit le persister.
func NewRegistryNamesPersister(db txBeginner) *RegistryNamesPersister {
	return &RegistryNamesPersister{db: db}
}

// WriteMatchNames inscrit, pour UN match, chaque nom de `writes` dans sa colonne, si et
// seulement si la colonne est NULL ou égale à l'identifiant. Rend les genres effectivement
// écrits (une ligne affectée). Un genre inconnu, un nom vide ou un matchID vide est refusé
// AVANT toute écriture ; une erreur SQL annule la transaction du match.
func (p *RegistryNamesPersister) WriteMatchNames(ctx context.Context, matchID string,
	writes []RegistryNameWrite) ([]string, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, errors.New("persist: WriteMatchNames: matchID vide")
	}
	for _, w := range writes {
		if _, ok := registryNameUpdates[w.Kind]; !ok {
			return nil, fmt.Errorf("persist: WriteMatchNames %s: genre inconnu %q", matchID, w.Kind)
		}
		if strings.TrimSpace(w.Name) == "" {
			return nil, fmt.Errorf("persist: WriteMatchNames %s: nom vide pour %s", matchID, w.Kind)
		}
	}
	if len(writes) == 0 {
		return nil, nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("persist: WriteMatchNames BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi

	var ecrits []string
	for _, w := range writes {
		res, err := tx.ExecContext(ctx, registryNameUpdates[w.Kind], w.Name, matchID)
		if err != nil {
			return nil, fmt.Errorf("persist: WriteMatchNames update %s %s: %w", w.Kind, matchID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("persist: WriteMatchNames RowsAffected %s %s: %w", w.Kind, matchID, err)
		}
		if n > 0 {
			ecrits = append(ecrits, w.Kind)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("persist: WriteMatchNames Commit %s: %w", matchID, err)
	}
	return ecrits, nil
}
