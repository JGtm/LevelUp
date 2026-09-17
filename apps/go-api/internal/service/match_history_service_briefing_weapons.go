// Package service — match_history_service_briefing_weapons.go : module « arme
// favorite » du bandeau de briefing de l'Explorer.
//
// C'est le SEUL module du briefing qui lit la base : tous les autres s'agrègent en
// mémoire sur les raw rows du scope. La lecture passe par port.WeaponKillsRepository,
// dont les deux implémentations (source de dégât du film / arme native du kill) sont
// choisies par le wiring — aucune comparaison de slug ici.
//
// Le filtre de lignes est celui du classement d'armes de la Synthèse
// (buildTopWeaponKills) : libellé résolu, hors sentinelles grenade/mêlée. Il sert à la
// fois le dénominateur « mesuré » et les deux entrées affichées, pour que la note de
// couverture reste cohérente avec ce qu'on montre.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// briefingWeaponsTopN est le nombre maximal d'armes affichées par le bloc. Deux, et
// jamais trois : le bloc s'empile sous « Par contexte » dans une cellule dont la
// hauteur est contrainte par le reste de la rangée.
const briefingWeaponsTopN = 2

// buildBriefingWeapons agrège les frags par arme du scope filtré et retourne l'arme
// favorite (une ou deux entrées) avec son dénominateur.
//
// Best-effort strict : repo absent, xuid vide, scope vide, capability manquante ou
// erreur de lecture → nil, jamais d'erreur propagée. Le briefing et la page ne
// dépendent pas de ce bloc.
func buildBriefingWeapons(
	ctx context.Context,
	repo port.WeaponKillsRepository,
	titleSlug, xuid string,
	filtered []domain.MatchHistoryRawRow,
) *domain.ExplorerBriefingWeapons {
	if repo == nil || xuid == "" || len(filtered) == 0 {
		return nil
	}
	matchIDs, scopeKills := briefingScopeMatchesAndKills(filtered)
	// XUIDs et jamais Gamertag : le filtre gamertag passe par une jointure
	// xuid_aliases qui rend zéro ligne EN SILENCE quand l'alias manque.
	// ResolveRoles alimente Class, dont le front tire la couleur de la barre.
	// IncludeGrenadeMelee reste faux : les lignes sentinelles grenade/mêlée ne sont
	// pas des armes et pollueraient le classement du second titre.
	wf := port.WeaponKillFilters{
		MatchIDs:            matchIDs,
		XUIDs:               []string{xuid},
		ResolveRoles:        true,
		IncludeGrenadeMelee: false,
	}
	rows, err := repo.LoadWeaponKillsAggregated(ctx, titleSlug, wf)
	if err != nil {
		// ErrCapabilityNotSupported = légitime (titre sans source d'arme) → Debug.
		// Toute autre erreur = anomalie (SQL, connexion) → Warn : même doctrine que
		// loadWeaponKillRows, pour qu'un bloc muet ne passe pas inaperçu.
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "briefing: weapon kills capability absente",
				"title", titleSlug, "match_count", len(matchIDs))
		} else {
			slog.WarnContext(ctx, "briefing: weapon kills query failed (best-effort, bloc omis)",
				"title", titleSlug, "match_count", len(matchIDs), "err", err)
		}
		return nil
	}
	// Une seule passe de filtre, partagée : le dénominateur compte ce qu'on saurait
	// nommer, le classement montre les deux premières de ce MÊME ensemble.
	retained := make([]port.WeaponKillRow, 0, len(rows))
	measured := 0
	for _, r := range rows {
		if r.Label == "" || r.IsGrenadeMelee {
			continue
		}
		retained = append(retained, r)
		measured += r.Kills
	}
	if measured == 0 {
		return nil
	}
	entries := buildTopWeaponKills(retained, briefingWeaponsTopN)
	if len(entries) == 0 {
		return nil
	}
	return &domain.ExplorerBriefingWeapons{
		Entries:       entries,
		MeasuredKills: measured,
		ScopeKills:    scopeKills,
	}
}

// briefingScopeMatchesAndKills extrait, des raw rows déjà en mémoire, les identifiants
// de match du scope et le total de frags qui sert de dénominateur. Aucune requête.
func briefingScopeMatchesAndKills(rows []domain.MatchHistoryRawRow) (matchIDs []string, kills int) {
	matchIDs = make([]string, 0, len(rows))
	for i := range rows {
		if rows[i].MatchID != "" {
			matchIDs = append(matchIDs, rows[i].MatchID)
		}
		kills += rows[i].Kills
	}
	return matchIDs, kills
}
