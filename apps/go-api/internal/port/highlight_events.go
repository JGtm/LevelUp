package port

import (
	"context"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// HighlightEventFilters parametre la lecture unifiee des events filmes.
//
// Garde-fou critique : sans MatchIDs ni (PlayerXUID + Since), la requete SQL
// genere un scan complet de shared.highlight_events. Validate() rejette ces
// combinaisons. (cf. PLAN_META_FOUNDATIONS_GO § 8 risques.)
type HighlightEventFilters struct {
	// MatchIDs ne garde que les events de ces matchs. Recommande pour
	// les usages fermes (kill feed MatchView, impact roles Squad).
	MatchIDs []string

	// PlayerXUID filtre les events dont KillerXUID, VictimXUID ou PlayerXUID
	// vaut cette valeur. nil = pas de filtre. Les three roles sont matches
	// en OR par le repo.
	PlayerXUID *string

	// EventTypes ne garde que les events dont EventType appartient a la liste.
	// nil ou vide = tous les types.
	EventTypes []canonical.HighlightEventType

	// Since ne garde que les events dont le match a demarre apres ce timestamp
	// (lookup via shared.match_registry.start_time). nil = pas de filtre.
	Since *time.Time

	// Limit borne le nombre d'events retournes. 0 = pas de limite.
	// Negatif = invalide.
	Limit int

	// OrderBy specifie l'ordre SQL. Vide = ordre par defaut du repo
	// (match_id ASC, time_ms ASC).
	OrderBy string
}

// ErrHighlightEventFiltersInvalid est l'erreur retournee par Validate() quand
// la combinaison des filtres est rejetee.
var ErrHighlightEventFiltersInvalid = errors.New("port: invalid HighlightEventFilters")

// ErrHighlightEventFiltersTooBroad est retournee specifiquement quand la
// combinaison ferait un scan complet (ni MatchIDs, ni PlayerXUID+Since).
var ErrHighlightEventFiltersTooBroad = errors.New("port: HighlightEventFilters too broad (provide MatchIDs or PlayerXUID+Since)")

// Validate verifie que la combinaison des filtres est coherente et bornee.
//
// Cas rejetes :
//   - Limit < 0
//   - EventType inconnu dans EventTypes
//   - Filtre trop large : MatchIDs vide ET (PlayerXUID nil OU Since nil)
//
// Cette validation doit etre appelee cote service avant de passer la struct
// au repo. Le repo doit re-valider en defense en profondeur.
func (f HighlightEventFilters) Validate() error {
	if f.Limit < 0 {
		return fmt.Errorf("%w: Limit must be >= 0, got %d",
			ErrHighlightEventFiltersInvalid, f.Limit)
	}
	for _, t := range f.EventTypes {
		if !canonical.IsKnownHighlightEventType(t) {
			return fmt.Errorf("%w: EventTypes contains unknown type %q",
				ErrHighlightEventFiltersInvalid, t)
		}
	}
	if len(f.MatchIDs) == 0 {
		if f.PlayerXUID == nil || f.Since == nil {
			return ErrHighlightEventFiltersTooBroad
		}
	}
	return nil
}

// HighlightEventsRepository expose le loader unifie des events filmes.
// Consomme par Squad (impact 8 roles), MatchView (kill feed, cadence,
// dominance), Timeseries (first events rolling, intensity, cadence), Career
// (first_blood badges optionnels).
//
// Implemente par internal/platform/duckdb.HighlightEventsRepo (chunk 6).
//
// Capability gating : retourne games.ErrCapabilityNotSupported si le titre
// indique par slug n'a pas la capability "match.detail.events".
type HighlightEventsRepository interface {
	// LoadHighlightEvents charge les events du titre filtres par
	// HighlightEventFilters. Le service appelant doit appeler filters.Validate()
	// avant.
	LoadHighlightEvents(
		ctx context.Context,
		slug string,
		filters HighlightEventFilters,
	) ([]canonical.HighlightEvent, error)

	// InvalidateMatch purge les entrees cache LRU pour ce match. Appele par
	// le sync apres une mise a jour des events.
	InvalidateMatch(slug, matchID string)
}
