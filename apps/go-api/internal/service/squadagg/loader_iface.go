package squadagg

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// SquadV2Loader charge les données brutes nécessaires aux calculs d'escouade V2
// (matchs, events filmés, kills par arme, médailles, emblèmes, stats par carte,
// scores objectif). Extrait ici (K3b) car partagé par le service-root escouade ET
// le sous-package teammates ; le loger dans la feuille squadagg évite le cycle
// service↔teammates. Implémentation prod : duckdb.SquadV2LoaderAdapter & co.
type SquadV2Loader interface {
	// LoadFor charge les matchs d'un joueur (chunk S1).
	LoadFor(
		ctx context.Context,
		slug string,
		gamertag string,
		filters port.PlayerMatchFilters,
	) ([]canonical.PlayerMatchRow, error)

	// LoadHighlightEvents charge les events filmes pour les matchs partages
	// (chunks S5+S6). Implementation prod : duckdb.HighlightEventsRepo.
	LoadHighlightEvents(
		ctx context.Context,
		slug string,
		filters port.HighlightEventFilters,
	) ([]canonical.HighlightEvent, error)

	// LoadWeaponKills charge les kills aggreges par arme (chunk S9).
	// Implementation prod : duckdb.WeaponKillsRepo.
	LoadWeaponKills(
		ctx context.Context,
		slug string,
		filters port.WeaponKillFilters,
	) ([]port.WeaponKillRow, error)

	// LoadKillMechanics charge les mécaniques de kill NATIVES Halo 5 agrégées par
	// xuid (assassinats + compétences spartiate) sur les matchs partagés.
	// Implementation prod : duckdb.SquadV2LoaderAdapter (query match_participants).
	// Réutilise WeaponKillFilters (MatchIDs + XUIDs). Dégrade en lignes à 0 pour
	// les titres sans ces colonnes.
	LoadKillMechanics(
		ctx context.Context,
		slug string,
		filters port.WeaponKillFilters,
	) ([]port.KillMechanicsRow, error)

	// LoadMedals charge les medailles par (xuid, match) (chunk S9).
	// Implementation prod : duckdb.MedalsByXUIDRepo.
	LoadMedals(
		ctx context.Context,
		slug string,
		filters port.MedalsByXUIDFilters,
	) ([]port.MedalRow, error)

	// LoadEmblemURLs retourne l'URL de l'emblème Spartan de chaque gamertag
	// (depuis career_progression.emblem_image_url). Dégradation silencieuse :
	// les joueurs sans DB ou sans données retournent une entrée vide.
	LoadEmblemURLs(
		ctx context.Context,
		slug string,
		gamertags []string,
	) map[string]string

	// LoadMapStatsForSquad retourne les stats par carte (winrate, perf moyenne)
	// du joueur principal sur les matchs où TOUS les xuids du squad sont
	// participants. Aucun filtre temporel — référence "avec cette escouade
	// exacte" pour les charts BulletWinrate / PerfVsHistorical (S3).
	// Capability absente -> nil sans erreur.
	LoadMapStatsForSquad(
		ctx context.Context,
		slug, mainGT string,
		squadXUIDs []string,
	) (map[string]domain.MapSquadStats, error)

	// LoadObjectiveScores retourne le total des award_score de catégorie
	// "objective" par match_id pour le joueur (slug, gamertag). Utilisé pour
	// l'axe objectif du radar synergie (teammates.06). Dégradation silencieuse
	// : retourne map vide + nil si personal_score_awards est absent ou si le
	// joueur est introuvable.
	LoadObjectiveScores(
		ctx context.Context,
		slug string,
		gamertag string,
		matchIDs []string,
	) (map[string]int, error)

	// LoadPlayerAssistsModel retourne le modèle personnel OLS d'assists attendus
	// d'un MEMBRE (sa player DB) pour un mode (game_variant_name). nil si le membre
	// n'a pas de modèle (< seuil d'échantillons / player DB absente) → l'appelant
	// bascule sur le fallback populationnel. Sert l'écart au FDA attendu par joueur
	// (teammates.16, chaîne D8).
	LoadPlayerAssistsModel(
		ctx context.Context,
		slug, gamertag, gameVariantName string,
	) (*domain.PlayerAssistsModel, error)

	// LoadPopulationalAssistsCoef retourne le fallback populationnel (slope, intercept)
	// du modèle d'assists attendus d'un mode (metadata title-level, partagée). ok=false
	// si les coefs sont indisponibles (table absente / titre sans modèle). Résolu
	// une fois par mode côté appelant.
	LoadPopulationalAssistsCoef(
		ctx context.Context,
		slug, gameVariantName string,
	) (slope, intercept float64, ok bool, err error)
}
