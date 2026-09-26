// Package duckdb — squad_v2_adapter.go : adapteur production qui implémente
// service.SquadV2Loader en résolvant (titleSlug, gamertag) -> *PlayerDB via
// le pool global, puis en déléguant la lecture à PlayerMatchesRepo.
//
// L'adapteur s'appuie sur un TitlePlayerResolver injecté par le wiring (cf.
// internal/api/registry.go) pour éviter d'importer le package config — ce qui
// créerait une dépendance circulaire (config importe duckdb).
//
// Capability gating : si le PlayerDB est introuvable (joueur absent de
// db_profiles, fichier stats.duckdb manquant, slug titre inconnu), l'erreur est
// traduite en games.ErrCapabilityNotSupported pour que SquadServiceV2 puisse
// dégrader gracieusement (capability gap dans la réponse au lieu d'une 5xx).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// TitlePlayerResolver traduit une paire (titleSlug, gamertag) en *PlayerDB
// prêt à l'emploi (pool-cached). L'implémentation production est fournie par
// le wiring HTTP (registry.go) à partir de config.AppConfig.LoadPlayers +
// GetOrOpen.
//
// Doit retourner une erreur (idéalement matchant ErrPlayerNotFound) si la
// paire ne correspond à aucun profil ou si le fichier stats.duckdb est absent.
type TitlePlayerResolver func(ctx context.Context, titleSlug, gamertag string) (*PlayerDB, error)

// SquadV2LoaderAdapter implémente service.SquadV2Loader en s'appuyant sur le
// pool de PlayerDB et PlayerMatchesRepo.
//
// `defaultGamertag` est utilise pour resoudre les DBs `shared` (events /
// weapon_kills / medals_earned) qui sont partagees entre tous les profils
// du titre — n'importe quel PlayerDB ouvert suffit. Le wiring HTTP appelle
// SetDefaultGamertag avec le main_gt de la session courante.
type SquadV2LoaderAdapter struct {
	resolve         TitlePlayerResolver
	defaultGamertag string
	// weaponKillsRepoFactory choisit LE lecteur de l'arme d'un kill pour ce titre.
	// Injecté par le wiring, qui est le SEUL à connaître les capabilities du titre ;
	// nil = repli historique (cf. weaponKillsRepo).
	weaponKillsRepoFactory WeaponKillsRepoFactory
}

// WeaponKillsRepoFactory rend le lecteur d'arme adossé aux capabilities du titre.
//
// POURQUOI UNE FABRIQUE INJECTÉE, ET PAS UN `NewWeaponKillsRepo` EN DUR. Le choix entre
// le lecteur historique (`weapon_kills`) et celui de la SOURCE DE DÉGÂT
// (`match_kill_events_latest.source_tag`) est une décision de CAPABILITY, et le wiring est
// le seul à la porter (`ServiceRegistry.weaponKillsRepoFor`). Cet adapteur vit dans
// `duckdb` et ne peut pas importer le wiring ; il reçoit donc la fabrique.
//
// CE QUE CE PARAMÈTRE RÉPARE (2026-09-01) : l'Escouade construisait `NewWeaponKillsRepo`
// SANS condition, seul appelant de production resté hors du gate. Sur un titre à décodeur
// de film, `weapon_kills` a été SUPPRIMÉE (`shared_drop_weapon_kills_v1`) — la page servait
// donc des séries vides pendant que toutes les autres surfaces lisaient la source de dégât.
type WeaponKillsRepoFactory func(*PlayerDB) port.WeaponKillsRepository

// NewSquadV2LoaderAdapter construit un adapteur production. Le resolver est
// injecté par le wiring (registry.go) — ne pas l'appeler avec resolver=nil.
func NewSquadV2LoaderAdapter(resolver TitlePlayerResolver) *SquadV2LoaderAdapter {
	return &SquadV2LoaderAdapter{resolve: resolver}
}

// SetWeaponKillsRepoFactory injecte le sélecteur de lecteur d'arme du wiring.
func (a *SquadV2LoaderAdapter) SetWeaponKillsRepoFactory(f WeaponKillsRepoFactory) {
	a.weaponKillsRepoFactory = f
}

// weaponKillsRepo rend le lecteur d'arme à utiliser pour ce PlayerDB.
//
// Repli sur le lecteur historique quand aucune fabrique n'est injectée : les tests et les
// appelants hors HTTP gardent le comportement d'avant le câblage, et un titre sans
// décodeur de film lit de toute façon `weapon_kills`.
func (a *SquadV2LoaderAdapter) weaponKillsRepo(pdb *PlayerDB) port.WeaponKillsRepository {
	if a.weaponKillsRepoFactory != nil {
		if repo := a.weaponKillsRepoFactory(pdb); repo != nil {
			return repo
		}
	}
	return NewWeaponKillsRepo(pdb)
}

// LoadFor charge les matchs du joueur (titleSlug, gamertag) en passant par le
// pool DuckDB et PlayerMatchesRepo.
//
// Si la résolution échoue parce que le profil ou la base est absent, l'erreur
// est traduite en games.ErrCapabilityNotSupported. Les autres erreurs (DB
// corrompue, requête en échec) remontent telles quelles.
func (a *SquadV2LoaderAdapter) LoadFor(
	ctx context.Context,
	titleSlug string,
	gamertag string,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if a.resolve == nil {
		return nil, errors.New("SquadV2LoaderAdapter: resolver non câblé")
	}

	pdb, err := a.resolve(ctx, titleSlug, gamertag)
	if err != nil {
		if isPlayerCapabilityError(err) {
			return nil, fmt.Errorf("%w: title=%q gamertag=%q (%v)",
				games.ErrCapabilityNotSupported, titleSlug, gamertag, err)
		}
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadFor: resolve %s/%s: %w",
			titleSlug, gamertag, err)
	}

	repo := NewPlayerMatchesRepo(pdb)
	rows, err := repo.Load(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadFor: load %s: %w", gamertag, err)
	}
	return rows, nil
}

// LoadHighlightEvents charge les events filmes (chunks S5+S6).
//
// Resolution PlayerDB : on prend n'importe quel joueur du squad — la table
// shared.highlight_events est partagee entre tous les profils du titre.
// Ici on n'a pas le squad disponible — on demande la resolution via un
// gamertag par defaut (le main player du squad). Le caller peut le passer
// explicitement via filters... mais filters n'a pas de Gamertag.
//
// Pragmatique : pour Phase 1 pilote, on suppose qu'au moins un joueur existe
// et on resout via le main_gt qui est passe en parametre du service. Comme
// on n'a pas accès à ce contexte ici, on délègue : si le resolve échoue,
// retourne ErrCapabilityNotSupported.
//
// TODO Phase 2 : refactor le resolver pour pouvoir charger une DB shared
// independamment d'un gamertag specifique.
func (a *SquadV2LoaderAdapter) LoadHighlightEvents(
	ctx context.Context,
	titleSlug string,
	filters port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	pdb, err := a.resolveAnyPlayerDB(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	repo := NewHighlightEventsRepo(pdb)
	events, err := repo.Load(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadHighlightEvents: %w", err)
	}
	return events, nil
}

// LoadWeaponKills charge les kills aggreges par arme (chunk S9).
func (a *SquadV2LoaderAdapter) LoadWeaponKills(
	ctx context.Context,
	titleSlug string,
	filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	pdb, err := a.resolveAnyPlayerDB(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	repo := a.weaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(ctx, titleSlug, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadWeaponKills: %w", err)
	}
	return rows, nil
}

// LoadKillMechanics agrège les mécaniques de kill NATIVES Halo 5 par xuid sur les
// matchs partagés (assassinats + compétences spartiate). Mirror de LoadWeaponKills :
// résout n'importe quelle player DB du titre (shared attaché in-process), puis
// GROUP BY xuid sur match_participants.
func (a *SquadV2LoaderAdapter) LoadKillMechanics(
	ctx context.Context,
	titleSlug string,
	filters port.WeaponKillFilters,
) ([]port.KillMechanicsRow, error) {
	pdb, err := a.resolveAnyPlayerDB(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	// PAS le lecteur gaté : les mécaniques de kill sont NATIVES et se lisent sur
	// `match_participants`, table que la bascule vers la source de dégât ne touche pas.
	// `LoadKillMechanicsAggregated` ne fait d'ailleurs pas partie de
	// `port.WeaponKillsRepository` — c'est une interface OPTIONNELLE que le lecteur adossé
	// à la source de dégât n'implémente pas, et n'a aucune raison d'implémenter.
	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadKillMechanicsAggregated(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadKillMechanics: %w", err)
	}
	return rows, nil
}

// LoadMapStatsForSquad delegue a SquadRepo.LoadMapStatsForSquad pour produire
// la reference "winrate du main avec l'escouade strict par carte" — sans
// aucun filtre temporel (cf. fix synergies session-vs-historique).
//
// Resolution : on resout le PlayerDB du main (via mainGT) car SquadRepo a
// besoin de la player DB attachee a shared.match_participants. mainXUID est
// alors lu depuis pdb.XUID. Si le main est absent du pool, ErrCapabilityNotSupported.
func (a *SquadV2LoaderAdapter) LoadMapStatsForSquad(
	ctx context.Context,
	titleSlug, mainGT string,
	squadXUIDs []string,
) (map[string]domain.MapSquadStats, error) {
	if a.resolve == nil {
		return nil, errors.New("SquadV2LoaderAdapter: resolver non câblé")
	}
	if len(squadXUIDs) == 0 {
		return nil, nil
	}
	pdb, err := a.resolve(ctx, titleSlug, mainGT)
	if err != nil {
		if isPlayerCapabilityError(err) {
			return nil, fmt.Errorf("%w: title=%q gamertag=%q (%v)",
				games.ErrCapabilityNotSupported, titleSlug, mainGT, err)
		}
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadMapStatsForSquad: resolve %s/%s: %w",
			titleSlug, mainGT, err)
	}
	if pdb == nil || pdb.XUID == "" {
		return nil, fmt.Errorf("%w: SquadV2LoaderAdapter.LoadMapStatsForSquad: pdb sans XUID pour %s",
			games.ErrCapabilityNotSupported, mainGT)
	}
	repo := NewSquadRepo(pdb)
	// excludeXUIDs nil : le chemin V2 (SquadV2Loader) ne porte pas de pool
	// d'exclusion — la composition exacte est appliquée côté TeammatesService.
	stats, err := repo.LoadMapStatsForSquad(ctx, pdb.XUID, squadXUIDs, nil)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadMapStatsForSquad: %w", err)
	}
	return stats, nil
}

// LoadPlayerAssistsModel retourne le modèle personnel OLS d'assists attendus d'un
// MEMBRE (sa player DB) pour un mode. nil (best-effort, jamais fatal) si le membre
// n'a pas de DB résoluble → l'appelant bascule sur le fallback populationnel.
func (a *SquadV2LoaderAdapter) LoadPlayerAssistsModel(
	ctx context.Context,
	titleSlug, gamertag, gameVariantName string,
) (*domain.PlayerAssistsModel, error) {
	if a.resolve == nil {
		return nil, errors.New("SquadV2LoaderAdapter: resolver non câblé")
	}
	pdb, err := a.resolve(ctx, titleSlug, gamertag)
	if err != nil {
		if isPlayerCapabilityError(err) {
			return nil, nil // membre sans player DB → pas de modèle (fallback populationnel)
		}
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadPlayerAssistsModel: resolve %s/%s: %w",
			titleSlug, gamertag, err)
	}
	if pdb == nil {
		return nil, nil
	}
	return NewMatchViewRepo(pdb, pdb.XUID).GetPlayerAssistsModel(ctx, gameVariantName)
}

// LoadPopulationalAssistsCoef retourne le fallback populationnel (slope, intercept)
// d'un mode depuis la metadata title-level (partagée). ok=false si aucune player DB
// n'est résoluble ou si la metadata est absente ; err non nil pour un vrai échec de
// requête (l'appelant loggue). Résolu une fois par mode côté appelant.
func (a *SquadV2LoaderAdapter) LoadPopulationalAssistsCoef(
	ctx context.Context,
	titleSlug, gameVariantName string,
) (slope, intercept float64, ok bool, err error) {
	pdb, rerr := a.resolveAnyPlayerDB(ctx, titleSlug)
	if rerr != nil || pdb == nil || pdb.Metadata == nil {
		return 0, 0, false, nil // pas de metadata résoluble → pas de fallback (best-effort)
	}
	s, i, cerr := NewMetadataRepo(pdb).GetAssistsCoef(ctx, gameVariantName)
	if cerr != nil {
		return 0, 0, false, cerr
	}
	return s, i, true, nil
}

// LoadMedals charge les medailles par (xuid, match) (chunk S9).
func (a *SquadV2LoaderAdapter) LoadMedals(
	ctx context.Context,
	titleSlug string,
	filters port.MedalsByXUIDFilters,
) ([]port.MedalRow, error) {
	pdb, err := a.resolveAnyPlayerDB(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(ctx, titleSlug, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadMedals: %w", err)
	}
	return rows, nil
}

// resolveAnyPlayerDB resout n'importe quel PlayerDB du titre (utilise pour
// les sources shared : highlight_events / weapon_kills / medals_earned —
// independamment du joueur tant qu'on a une DB du titre ouverte).
//
// Strategie : on essaie le resolver avec le gamertag stocke dans
// a.defaultGamertag (a peupler par le wiring). Si nil, retourne
// ErrCapabilityNotSupported.
func (a *SquadV2LoaderAdapter) resolveAnyPlayerDB(ctx context.Context, titleSlug string) (*PlayerDB, error) {
	if a.resolve == nil || a.defaultGamertag == "" {
		return nil, fmt.Errorf("%w: SquadV2LoaderAdapter no default gamertag wired",
			games.ErrCapabilityNotSupported)
	}
	pdb, err := a.resolve(ctx, titleSlug, a.defaultGamertag)
	if err != nil {
		if isPlayerCapabilityError(err) {
			return nil, fmt.Errorf("%w: title=%q (%v)",
				games.ErrCapabilityNotSupported, titleSlug, err)
		}
		return nil, fmt.Errorf("resolveAnyPlayerDB %s: %w", titleSlug, err)
	}
	return pdb, nil
}

// LoadEmblemURLs retourne l'URL de l'emblème Spartan de chaque gamertag en
// chargeant career_progression.emblem_image_url depuis le PlayerDB de chaque
// joueur. Dégradation silencieuse : un joueur absent ou sans données reçoit
// une entrée vide dans la map. Les chargements sont parallélisés.
func (a *SquadV2LoaderAdapter) LoadEmblemURLs(
	ctx context.Context,
	titleSlug string,
	gamertags []string,
) map[string]string {
	result := make(map[string]string, len(gamertags))
	if a.resolve == nil || len(gamertags) == 0 {
		return result
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, gt := range gamertags {
		wg.Add(1)
		go func(gamertag string) {
			defer wg.Done()
			pdb, err := a.resolve(ctx, titleSlug, gamertag)
			if err != nil || pdb == nil || pdb.Player == nil {
				return
			}
			var rawURL sql.NullString
			rows, err := pdb.Player.QueryRowRecovered(ctx,
				`SELECT ARG_MAX(emblem_image_url, recorded_at)
				 FILTER (WHERE NULLIF(TRIM(emblem_image_url), '') IS NOT NULL)
				 FROM career_progression`,
			)
			if err != nil {
				return
			}
			defer rows.Close()
			if err := rows.Scan(&rawURL); err != nil || !rawURL.Valid || rawURL.String == "" {
				return
			}
			built := buildHomeIdentityAssetURL("emblem", titleSlug, rawURL.String)
			if built == nil {
				return
			}
			mu.Lock()
			result[gamertag] = *built
			mu.Unlock()
		}(gt)
	}
	wg.Wait()
	return result
}

// SetDefaultGamertag configure le gamertag par defaut pour resoudre les DBs
// shared (events, weapons, medals). Appele par le wiring HTTP avec le
// gamertag du main player de la session courante.
func (a *SquadV2LoaderAdapter) SetDefaultGamertag(gt string) {
	a.defaultGamertag = gt
}

// isPlayerCapabilityError détecte les motifs d'erreur signifiant "ce joueur
// n'a pas la capability match.history" : profil introuvable dans
// db_profiles.json ou fichier stats.duckdb manquant. Tout autre échec
// (timeout, DB corrompue) reste une vraie erreur.
func isPlayerCapabilityError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// Le message canonique côté config.ResolvePlayer est "joueur introuvable" ;
	// on reste défensif via une string-match prudente (le package duckdb ne
	// peut pas importer config sans créer un cycle).
	msg := err.Error()
	for _, marker := range []string{
		"joueur introuvable",
		"player_not_found",
		"no such file",
		"does not exist",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
