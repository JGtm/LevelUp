package port

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

// PlayerMatchFilters parametre la lecture unifiee des matchs joueur. Un service
// remplit la struct avec les seuls champs qui l'interessent ; les zero values
// signifient "pas de filtre" sauf documentation contraire.
//
// Les filtres dont la presence ou l'absence sont semantiquement distinctes
// (HadBotTeammate, IsFirefight, IsRanked, MinTimePlayedSeconds) sont des
// pointeurs : nil = pas de filtre, &true = filtre "doit etre vrai",
// &false = filtre "doit etre faux".
type PlayerMatchFilters struct {
	// Period filtre par fenetre temporelle. nil = aucun filtre temporel.
	Period *temporal.Period

	// OutcomeIn ne garde que les rows dont Self.Outcome appartient a la liste.
	// nil ou vide = pas de filtre.
	OutcomeIn []canonical.Outcome

	// HadBotTeammate : nil = pas de filtre. &false = exclusion des matchs
	// avec coequipier bot (cas Career top matches qui les rejette).
	HadBotTeammate *bool

	// IsFirefight : nil = pas de filtre. &false = exclusion firefight
	// (Career top matches), &true = uniquement firefight (page PvE).
	IsFirefight *bool

	// IsRanked : nil = pas de filtre. &true = ranked uniquement.
	IsRanked *bool

	// MinTimePlayedSeconds rejette les rows dont Self.TimePlayed < seuil.
	// Career top matches utilise 180 (3 min). nil = pas de filtre.
	MinTimePlayedSeconds *int

	// ExcludeFriendsXUIDs retire les matchs ou un de ces xuid est present
	// (utile pour Career encounters "exclure les amis"). nil ou vide = pas
	// d'exclusion.
	ExcludeFriendsXUIDs []string

	// BTBExcluded rejette les matchs dont mode_category == "BTB" (toggle UI
	// Career). false par defaut = pas de filtre.
	BTBExcluded bool

	// PlaylistKind est un alias court resolu cote handler en regex via
	// playlist_regex_whitelist. Jamais de regex libre injectee depuis ce
	// champ. nil = pas de filtre playlist.
	PlaylistKind *string

	// MapIDs ne garde que les rows dont Summary.Map.ID appartient a la liste.
	// nil ou vide = pas de filtre.
	MapIDs []string

	// Limit borne le nombre de rows retournees. 0 = pas de limite.
	// Negatif = invalide (cf. Validate).
	Limit int

	// OrderBy specifie l'ordre SQL. Vide = ordre par defaut du repo
	// (start_time DESC). Doit etre une expression SQL safe (whitelist
	// d'identifiants cote repo).
	OrderBy string
}

// ErrPlayerMatchFiltersInvalid est l'erreur retournee par Validate() quand
// la combinaison des filtres est rejetee.
var ErrPlayerMatchFiltersInvalid = errors.New("port: invalid PlayerMatchFilters")

// Validate verifie que la combinaison des filtres est coherente. Doit etre
// appelee par le service avant de passer la struct au repo, mais le repo doit
// aussi valider en defense en profondeur (input untrusted).
func (f PlayerMatchFilters) Validate() error {
	if f.Limit < 0 {
		return fmt.Errorf("%w: Limit must be >= 0, got %d",
			ErrPlayerMatchFiltersInvalid, f.Limit)
	}
	if f.MinTimePlayedSeconds != nil && *f.MinTimePlayedSeconds < 0 {
		return fmt.Errorf("%w: MinTimePlayedSeconds must be >= 0, got %d",
			ErrPlayerMatchFiltersInvalid, *f.MinTimePlayedSeconds)
	}
	if f.Period != nil && !f.Period.IsValid() {
		return fmt.Errorf("%w: Period %q is not a known value",
			ErrPlayerMatchFiltersInvalid, *f.Period)
	}
	for _, o := range f.OutcomeIn {
		if !canonical.IsKnownOutcome(o) {
			return fmt.Errorf("%w: OutcomeIn contains unknown outcome %q",
				ErrPlayerMatchFiltersInvalid, o)
		}
	}
	return nil
}

// PlayerMatchesRepository expose le loader unifie des matchs joueur. Toutes
// les pages qui agregent des matchs (Squad, MatchView, Career, Synthesis,
// Citations, Timeseries) consomment cette interface plutot que d'ecrire des
// requetes SQL ad hoc.
//
// Implemente par internal/platform/duckdb.PlayerMatchesRepo (chunk 6).
//
// Capability gating : retourne games.ErrCapabilityNotSupported si le titre
// indique par slug n'a pas la capability "match.history".
type PlayerMatchesRepository interface {
	// LoadPlayerMatches charge les matchs du joueur (gamertag) pour le titre
	// (slug), filtres par PlayerMatchFilters. Le service appelant doit appeler
	// filters.Validate() avant.
	LoadPlayerMatches(
		ctx context.Context,
		slug string,
		gamertag string,
		filters PlayerMatchFilters,
	) ([]canonical.PlayerMatchRow, error)

	// InvalidatePlayer purge les entrees cache LRU pour ce joueur. Appele par
	// le sync apres une mise a jour des matchs.
	InvalidatePlayer(slug, gamertag string)
}
