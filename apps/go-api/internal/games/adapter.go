// Package games — interfaces title-aware pour les services produit.
//
// Phase B du plan multi-titres : sépare en 2 interfaces (SRP) la lecture
// canonique services et l'exposition des libellés sémantiques.
//
//   - TitleDataAdapter : transforme le stockage propre du titre (DuckDB,
//     fetcher live, etc.) vers le canonique services consommé par les
//     services produit (career, home, match view…).
//
//   - TitleSemanticAdapter : expose les FieldMappingSet/AssetMappingSet/
//     OutcomeMappingSet chargés depuis les TOML versionnés Git.
//
// Les deux interfaces sont injectées séparément dans les services produit.
// Un service ne dépend que de ce dont il a besoin (data, semantic, ou les deux).
package games

import (
	"context"
	"errors"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// ErrCapabilityNotSupported est retournée par les méthodes Load* d'un
// TitleDataAdapter quand la capability requise est absente du titre.
//
// Comportement attendu côté caller : ne pas considérer comme une erreur
// inattendue, plutôt comme un signal de dégradation gracieuse (cf. plan §5.7).
var ErrCapabilityNotSupported = errors.New("capability not supported by title")

// ErrTitleNotResolved est retournée par le Resolver quand un slug est inconnu.
var ErrTitleNotResolved = errors.New("title slug not resolved")

// CapabilityKey identifie une capability produit (au sens HALO_INFINITE_CAPABILITY_MAP.md).
type CapabilityKey string

const (
	CapMatchHistory       CapabilityKey = "match.history"
	CapMatchDetailCore    CapabilityKey = "match.detail.core"
	CapMatchSkillSnapshot CapabilityKey = "match.skill.snapshot"
	CapCareerProgression  CapabilityKey = "career.progression"
	CapCareerRankCatalog  CapabilityKey = "career.rank_catalog" // rangs de carrière = catalogue table-backed avec icône par palier (Infinite ; absent pour h5, SR numérique)
	CapPveFirefight       CapabilityKey = "pve.firefight_stats"
	CapTimeseries         CapabilityKey = "analytics.timeseries"
	// CapAnalyticsCareerXPEstimate — série « XP de carrière (estimée) » par match sur
	// la page Timeseries (XP = multiplicateur d'éra × personal_score). Halo Infinite
	// uniquement (progression Career Rank fondée sur le Personal Score) ; ABSENTE pour
	// un titre à système d'XP distinct (Halo 5 = Spartan Rank). Lue via CapabilityMap
	// au service Timeseries (jamais de slug ==).
	CapAnalyticsCareerXPEstimate CapabilityKey = "analytics.career_xp_estimate"
	CapScoreboardExtra           CapabilityKey = "match.scoreboard.extra" // champs étendus du scoreboard
	CapCitationsEngine           CapabilityKey = "citations.engine"       // moteur de citations
	CapEngagement                CapabilityKey = "engagement.score"       // score + courbe + coefficients d'engagement
	CapBattlePass                CapabilityKey = "battlepass.progression" // progression battle pass / season pass
	CapChallenges                CapabilityKey = "challenges.surface"     // surface défis (hebdo/quotidiens)
	// Timeline d'événements de match (canonical MatchEvents). Halo 5 natif ;
	// Infinite reconstitué depuis highlight_events (cf. PLAN_CANONICAL_MATCH_EVENTS).
	CapMatchEventsTimeline  CapabilityKey = "match.events.timeline"   // timeline horodatée (kills/médailles/armes/spawns)
	CapMatchKillfeedPerKill CapabilityKey = "match.killfeed.per_kill" // arme PAR kill (Halo 5 natif ; Infinite degraded RE film)
	CapMatchEventsSpatial   CapabilityKey = "match.events.spatial"    // positions monde x,y,z (Halo 5 ; Infinite not_exposed)
	// Commendations NATIVES par match (Halo 5 natif : carnage
	// ProgressiveCommendationDeltas → match_commendations, AXE B prod-gate).
	// DISTINCTE de citations.engine (moteur de citations DÉRIVÉ d'Infinite) :
	// affichage NATIF tel quel, pas de reconstruction par tier/composite.
	CapCommendationsNative CapabilityKey = "commendations.native"

	// CapWeaponAccuracy — précision PAR ARME : tirs au but / tirs tirés par arme.
	// Dérivée des compteurs ShotsFired/ShotsLanded (Halo 5 natif, events
	// weapon_drop → table weapon_accuracy, somme par (xuid, weapon_id)) OU d'une
	// précision directe par arme si un titre la fournit telle quelle.
	//
	// Infinite : not_exposed, et la raison a CHANGÉ le 2026-08-02 (J4-4). Ce n'est
	// plus « la table n'est pas peuplée » — la passe de film écrit désormais les
	// tirs par arme (cf. CapFilmWeaponShots). C'est que LE TAUX N'EST PAS
	// PUBLIABLE : calculé sur le corpus entier il INVERSE l'ordre MA40/Sidekick par
	// rapport à la référence de l'API (MA40 8 points devant en brut, Sidekick
	// 3 points devant à la référence). Ce n'est pas une imprécision, c'est une
	// réponse fausse. Quatre armes seulement tiennent à ±0,03, et seulement sur les
	// joueurs dont l'arme domine ≥ 80 % de leurs tirs décodés ; les armes à
	// projectile sont fausses d'un facteur 30 à 60.
	// Critère de bascule vers degraded : une population de publication déclarée
	// dans le code (pas dans un commentaire) ET une mesure d'écart à l'API par
	// arme publiée à côté du taux. Cf. PLAN_BRANCHEMENT_KILLSOURCE §4.2.
	CapWeaponAccuracy CapabilityKey = "match.weapon.accuracy"

	// CapPlaylistCategoryStrip — le libellé de playlist du titre porte un préfixe
	// de CATÉGORIE matchmaking à retirer pour l'affichage (ex. Halo Infinite :
	// "Arène delta : Héritage" → "Delta : Héritage"). Déclarée pour les titres dont
	// les noms de playlist officiels sont ainsi préfixés ; ABSENTE pour un titre dont
	// le nom officiel n'a pas de préfixe (ex. Halo 5 : "Super Fiesta Fête" doit rester
	// entier — le strip Infinite le tronquerait en "Fête"). Lue via CapabilityMap au
	// site de résolution du libellé (jamais de slug ==).
	CapPlaylistCategoryStrip CapabilityKey = "playlist.label.strip_category"

	// CapMatchObjectiveStats — stats objectifs PAR JOUEUR PAR MATCH des modes à
	// objectif (CTF/Zones (Strongholds+KOTH)/Oddball), extraites nativement du
	// payload GetMatchStats (Players[].PlayerTeamStats[0].Stats.<BlocMode>) et
	// stockées dans shared.match_objective_stats. Halo Infinite : supported.
	// Halo 5 : not_exposed (le carnage n'agrège pas ces objectifs ; promotion
	// degraded par agrégation d'impulses = chantier ultérieur distinct). Gouverne
	// le chemin de DONNÉES serveur (JOIN _latest au scoreboard/synthesis/squad) ;
	// la porte d'AFFICHAGE UI passe par la capability title-level `objective_stats`
	// (registry.go, useCapability). Cf. PLAN_V72_OBJECTIVE_STATS.md.
	CapMatchObjectiveStats CapabilityKey = "match.objective.stats"

	// CapFilmKillSource — le titre expose un FILM Theater dont le décodeur sait
	// extraire, PAR MORT, les deux vérités : le crédit du kill-feed et la SOURCE
	// DU DÉGÂT FATAL (lue dans le dead-state de la victime). Gouverne le
	// collecteur `internal/sync` qui remplit `shared.match_kill_events`.
	//
	// Halo Infinite : supported (décodeur `games/halo_infinite/film/killsource`).
	// Halo 5 : ABSENTE — son format de film est différent ET ses mécaniques de
	// kill sont natives dans le carnage (CapNativeKillMechanics), donc il n'a
	// aucun besoin d'un décodeur de film pour la même information.
	//
	// ⚠ Clé FINE et pas fourre-tout : les tirs par arme (`match_weapon_shots`) et
	// la précision par arme sont des familles de données DISTINCTES, chacune avec
	// ses réserves de publication. Un titre peut avoir l'une sans les autres —
	// c'est l'arbitrage §4.3 du plan de branchement. Ne JAMAIS élargir cette clé
	// pour couvrir « tout ce qui vient du film ».
	//
	// Nommage : le plan proposait `film_kill_source` ; la convention du
	// vocabulaire est le point (`match.objective.stats`, `playlist.label.…`),
	// d'où `film.kill_source`. Même clé, écriture du dépôt.
	CapFilmKillSource CapabilityKey = "film.kill_source"

	// CapFilmWeaponShots — la VENTILATION DES TIRS PAR ARME décodée du film
	// (`shared.match_weapon_shots`, grain joueur × arme × match). Deuxième famille
	// de données du film, distincte du kill enrichi : elle sort de la MÊME passe de
	// décodage mais répond à une autre question, et elle a ses propres réserves.
	//
	// Halo Infinite : supported (producteur `sync/killcollector/shots.go`, k = 1,
	// 84 % des joueurs en mode standard ; Fiesta et BTB NON livrables).
	// Halo 5 : ABSENTE — pas de décodeur de film, et ses compteurs de tirs sont
	// natifs (cf. CapWeaponAccuracy, events weapon_drop).
	//
	// ⚠ CETTE CLÉ GOUVERNE LE STOCKAGE, PAS UNE PUBLICATION. La table STOCKE des
	// COMPTES ; elle ne publie aucun TAUX. Le taux par arme est la famille
	// `match.weapon.accuracy`, et pour Infinite il n'est pas publiable (voir sa
	// documentation : il inverse l'ordre MA40/Sidekick). Un titre peut donc avoir
	// celle-ci sans celle-là — c'est exactement pourquoi ce sont deux clés.
	CapFilmWeaponShots CapabilityKey = "film.weapon_shots"
)

// CapabilityMap décrit l'état des capabilities produit d'un adapter à un instant T.
type CapabilityMap map[CapabilityKey]CapabilityStatus

// CapabilityStatus reflète la sémantique de HALO_INFINITE_CAPABILITY_MAP.md.
type CapabilityStatus string

const (
	CapSupported  CapabilityStatus = "supported"
	CapDegraded   CapabilityStatus = "degraded"
	CapNotExposed CapabilityStatus = "not_exposed"
)

// Has retourne vrai si la capability est supportée ou dégradée (mais utilisable).
func (cm CapabilityMap) Has(k CapabilityKey) bool {
	switch cm[k] {
	case CapSupported, CapDegraded:
		return true
	}
	return false
}

// TitleDataAdapter expose la lecture canonique services pour un titre donné.
//
// Toutes les méthodes Load* :
//   - retournent un schéma canonique stable (cf. internal/games/canonical/) ;
//   - retournent ErrCapabilityNotSupported si la capability requise est absente ;
//   - n'ont aucune dépendance à un endpoint provider natif, à du SQL externe
//     ou à un nom de colonne titre-spécifique.
type TitleDataAdapter interface {
	TitleSlug() string
	Capabilities() CapabilityMap

	LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error)
	LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error)
	LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error)
	LoadEncounters(ctx context.Context, xuid string) ([]canonical.EncounterRow, error)
	// LoadLUSRHistory : historique des checkpoints de rating LUSR/CSR (Phase 2
	// HIGH-C). ErrCapabilityNotSupported si le titre ne porte pas de rating LUSR.
	LoadLUSRHistory(ctx context.Context, xuid string) ([]canonical.LUSRCheckpoint, error)
	// LoadTopMatches : meilleurs/pires matchs carrière (Phase 2 HIGH-C).
	LoadTopMatches(ctx context.Context, xuid string) ([]canonical.CareerTopMatch, error)
	// LoadTargetRecentMatches : profil de combat récent d'un joueur cible (Explorer,
	// Phase 2 HIGH-B). ErrCapabilityNotSupported si pas de substrat match local.
	LoadTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]canonical.RecentMatchRow, error)
	// LoadParticipantStats : agrégat des stats d'un joueur sur un set de matchs
	// (Explorer sample, Phase 2 HIGH-B). nil si set vide ; ErrCapabilityNotSupported
	// si pas de substrat participant.
	LoadParticipantStats(ctx context.Context, xuid string, matchIDs []string) (*canonical.PlayerMatchSetStats, error)
	// LoadPlayerIntersection : matchs communs + kills croisés entre 2 joueurs
	// (Explorer, Phase 2 HIGH-B). Échec sur les matchs communs = fatal ; échec sur
	// les kills croisés = dégradation gracieuse (CrossKills vide). ErrCapabilityNotSupported
	// si le titre n'a pas de substrat match partagé.
	LoadPlayerIntersection(ctx context.Context, selfXUID, otherXUID string) (*canonical.PlayerIntersection, error)
	LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error)

	// Phase B+ : scoreboard étendu + événements + amis (CapScoreboardExtra).
	// Retournent ErrCapabilityNotSupported si la capability est absente.
	LoadMatchScoreboard(ctx context.Context, matchID string) ([]canonical.MatchParticipant, error)
	LoadHighlightEvents(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error)
	LoadFriendsXUIDs(ctx context.Context, xuid string) ([]string, error)

	// LoadMatchEvents : timeline d'événements BRUTE et complète d'un match
	// (CapMatchEventsTimeline). Surface SÉPARÉE chargée on-demand.
	// ErrCapabilityNotSupported si le titre ne sert pas d'events.
	// Cf. PLAN_CANONICAL_MATCH_EVENTS.
	LoadMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error)
}

// TitleSemanticAdapter expose les libellés et la présentation pour un titre.
//
// Ranks() retourne le catalog des rangs de carrière localisés. Peut retourner
// un catalog vide (Len() == 0) si la table career_rank_translations n'a pas
// encore été peuplée — les services consommateurs doivent alors dégrader
// proprement (ex: afficher uniquement le rank_id).
//
// Assets() / Outcomes() peuvent retourner nil si les TOML correspondants ne
// sont pas définis pour le titre. Les callers doivent dégrader gracieusement.
type TitleSemanticAdapter interface {
	TitleSlug() string
	SchemaVersion() int
	Fields() *mappings.FieldMappingSet
	Ranks() *mappings.RankCatalog
	Assets() *mappings.AssetMappingSet
	Outcomes() *mappings.OutcomeMappingSet
}

// TitleAssetURLAdapter expose la composition d'URLs d'assets statiques
// title-spécifiques.
//
// Phase 6 du plan de finition multi-titres : couche 3 (cf. SRP 4-couches du
// BACKLOG static FS migration). Convertit un identifiant métier (mapName,
// medalID, tier) en URL relative, en encapsulant les conventions de naming
// propres au titre (encodage espaces map names, format CSR rank, etc.).
//
// Délègue la composition path à `internal/assets/static/` (couche 2 pure).
// Les callers (home, home_repo, citation_snippets, frontend) reçoivent cet
// adapter via le Resolver et n'ont aucune connaissance du format path/encoding.
//
// Les méthodes retournent "" quand l'identifiant est invalide (UUID, vide,
// non reconnu) — les callers doivent dégrader gracieusement (cache-aside,
// fallback nil, etc.).
type TitleAssetURLAdapter interface {
	TitleSlug() string

	// MapImageURL retourne l'URL de l'image d'une map à partir de son nom métier.
	// Retourne "" si mapName est vide, est un UUID ou n'est pas reconnu.
	MapImageURL(mapName string) string

	// MedalImageURL retourne l'URL de l'icône d'une médaille à partir de son ID numérique.
	MedalImageURL(medalID uint64) string

	// CSRRankImageURL retourne l'URL du badge d'un rang CSR (tier + sub-tier).
	// Cas spécial : pour Onyx, utiliser CSRRankImageURLOnyx (pas de sub-tier).
	CSRRankImageURL(tier string, subTier int) string

	// CSRRankImageURLOnyx retourne l'URL du badge Onyx (sans sub-tier).
	CSRRankImageURLOnyx() string

	// WeaponImageURL retourne l'URL de l'image d'une arme à partir de son
	// nom EN officiel (ex. "BR75", "Energy Sword"). Retourne "" si non reconnu.
	WeaponImageURL(nameEN string) string

	// MatchWebURL retourne l'URL de la page publique d'un match sur le portail
	// officiel du titre (ex: Waypoint pour Halo Infinite). "" si le titre n'a pas
	// de page de détail publique (dégradation : pas de lien externe).
	MatchWebURL(matchID string) string

	// PlayerMatchWebURL retourne l'URL de la page publique d'un match POUR un
	// joueur donné sur le portail officiel. "" si le titre n'a pas de page publique
	// ou si le gamertag est vide.
	PlayerMatchWebURL(gamertag, matchID string) string
}

// Resolver injecte les adapters d'un titre courant aux services produit.
//
// Il est construit au boot du serveur et exposé via la DI. Un service produit
// reçoit Resolver et appelle Data(slug), Semantic(slug), AssetURL(slug) ou
// Catalog(slug) selon son besoin.
//
// Il n'a aucune connaissance des slugs supportés en dur : c'est le constructeur
// (api/server.go) qui peuple le resolver via Register*().
type Resolver interface {
	Data(titleSlug string) (TitleDataAdapter, error)
	Semantic(titleSlug string) (TitleSemanticAdapter, error)
	AssetURL(titleSlug string) (TitleAssetURLAdapter, error)
	Catalog(titleSlug string) (TitleCatalogAdapter, error)
	DefaultSlug() string
}
