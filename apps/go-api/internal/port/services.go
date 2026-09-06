// Package port — services.go : interfaces des services métier.
//
// Sprint 37 : injection de dépendances — les handlers dépendent de ces
// interfaces, pas des implémentations concrètes de internal/service/.
package port

import (
	"context"
	"errors"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/canonical"
)

// ---------------------------------------------------------------------------
// Services joueur (per-player, créés par PlayerServiceFactory)
// ---------------------------------------------------------------------------

// CareerService construit les réponses de la page Carrière.
type CareerService interface {
	GetCareerPage(ctx context.Context) (domain.CareerPageResponse, error)
	GetTopMatches(ctx context.Context) (domain.CareerTopMatchesResponse, error)
	GetEncounters(ctx context.Context) (domain.CareerEncountersResponse, error)
	// GetHighlightMatchIDs : 15 best + 15 worst match_ids (split par section)
	// + cascade counts (available_experience, available_seasons) pour les
	// dropdowns. L'enrichissement Explorer-format des rows est fait par le
	// handler via MatchHistoryService.
	GetHighlightMatchIDs(ctx context.Context, input domain.HighlightFilterInput) (domain.HighlightMatchesData, error)
	// GetTopEncounters : 10 joueurs les plus croisés au niveau carrière,
	// hors amis configurés (FriendGamertags), avec badges narratifs.
	GetTopEncounters(ctx context.Context) (domain.CareerTopEncountersResponse, error)
	// GetRivals : top 10 némésis + top 10 souffre-douleur via killer_victim_pairs.
	GetRivals(ctx context.Context) (domain.CareerRivalsResponse, error)
	// GetCareerCSRs : classements CSR par playlist depuis player_csr_snapshots
	// pour la saison demandée (vide → courante) + saisons proposables.
	GetCareerCSRs(ctx context.Context, seasonID string) (domain.CareerCSRResponse, error)
}

// RelationsService construit la réponse du hub Communauté > Relations
// (page transverse non gatée).
//
// Phase 2 : la segmentation serveur passe le FilterContextInput (expérience,
// saison/période, playlist/mode, vue solo/escouade) reçu en body POST. Un input
// zéro-valeur (corps absent) ⇒ comportement Phase 1 (tous les matchs).
type RelationsService interface {
	GetRelationsPage(ctx context.Context, input domain.FilterContextInput) (domain.RelationsPageResponse, error)
	// GetRelationsMoments construit la section « Moments & Rivalités » (Phase 3a) :
	// heatmap agrégé relation × tranche horaire (top-N) + cartes revanche (top-3
	// rivaux par matchs en ennemi). Hérite de la même segmentation serveur.
	GetRelationsMoments(ctx context.Context, input domain.FilterContextInput) (domain.RelationsMomentsResponse, error)
}

// CrossGameHit décrit la co-occurrence du joueur courant avec une relation sur
// un AUTRE titre géré par l'app : nombre de matchs communs sur ce titre +
// nom d'affichage résolu du titre (pour le libellé du badge cross-jeu).
type CrossGameHit struct {
	TitleDisplayName string // ex. "Halo 5" (résolu via TitleRegistry, jamais littéral)
	MatchesTogether  int
}

// CrossGameCooccurrence calcule, pour un ensemble de xuid de relations, les
// co-occurrences du joueur courant sur les AUTRES titres gérés par l'app.
//
// BEST-EFFORT / LECTURE SEULE / DÉFENSIF : l'implémentation énumère les titres
// via le TitleRegistry (hors titre courant), lit le shared de chaque autre titre
// (xuid global, ADR 0008), et compte les matchs communs par xuid. Toute erreur
// d'accès (DB absente, lock, capability) est avalée (skip + log) — la map
// retournée ne contient QUE les hits >= seuil. Une seule requête batch par
// autre titre (jamais une DB par relation).
type CrossGameCooccurrence interface {
	// CooccurrencesByXUID retourne, pour chaque oppXUID croisé >= seuil sur un
	// autre titre, le hit cross-jeu (titre le plus pertinent). map vide si aucun
	// autre titre disponible / aucune co-occurrence — jamais d'erreur propagée.
	CooccurrencesByXUID(ctx context.Context, oppXUIDs []string) map[string]CrossGameHit
}

// CitationsService construit les réponses Citations et Commendations.
type CitationsService interface {
	GetCitationsPage(ctx context.Context) (*domain.CitationsPageResponse, error)
	GetCommendationsPage(ctx context.Context, playerXUID string) (*domain.CommendationsPageResponse, error)
}

// CommendationTotalsService construit la page Totaux des commendations NATIVES
// (Halo 5, AXE B). Réponse vide pour les titres sans commendations natives.
type CommendationTotalsService interface {
	GetTotals(ctx context.Context) (*domain.NativeCommendationsTotalsResponse, error)
}

// ExplorerService orchestre les requêtes de l'Explorer (matchs communs).
//
// otherXUID est optionnel : s'il est fourni, le service l'utilise directement et
// saute la résolution gamertag→xuid locale (cas du Classement, où le xuid est
// déjà connu et où le joueur peut être absent des données locales).
type ExplorerService interface {
	GetCommonMatches(ctx context.Context, otherGamertag, otherXUID string, page int) (domain.ExplorerPlayerQueryResponse, error)
}

// FiltersService résout le contexte de filtres d'un joueur.
type FiltersService interface {
	Resolve(ctx context.Context, input domain.FilterContextInput) (domain.FilterContextResolved, error)
	// ResolveMatchIDs retourne la liste ordonnée (start_time DESC) des match_id
	// de la sélection — alimente le bouton "Voir les matchs" (parcours prev/next).
	ResolveMatchIDs(ctx context.Context, input domain.FilterContextInput) ([]string, error)
}

// HomeService construit les réponses de la page d'accueil Mission Control.
type HomeService interface {
	GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error)
	GetBattlePass(ctx context.Context) domain.BattlePassResponse
	GetChallenges(ctx context.Context) domain.ChallengesResponse
	// RefreshTrack hydrate (synchroneement) la définition d'un reward track
	// et toutes ses définitions d'items dans la DB metadata (battlepass_item_definitions).
	// Best-effort : silencieux en cas d'erreur (resolver miss, GameCMS down…).
	RefreshTrack(ctx context.Context, trackPath string)
}

// SeasonPassService construit la réponse de la page Season Pass (palmares).
type SeasonPassService interface {
	GetSeasonPassPage(ctx context.Context) (domain.SeasonPassPageResponse, error)
}

// SessionNotifier est notifié des changements de présence active d'un joueur.
// Implémenté par *service.HomeService pour ajuster le TTL cache BP/Challenges
// en fonction de l'état de session (actif → 5 min, inactif → 1 h).
type SessionNotifier interface {
	SetSessionActive(active bool)
}

// MatchHistoryService construit les réponses d'historique de matchs.
type MatchHistoryService interface {
	GetPage(ctx context.Context, req domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error)
	ExportCSV(ctx context.Context, req domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error)
}

// MatchViewService construit la vue détaillée d'un match.
type MatchViewService interface {
	GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error)
	GetMatchNeighbors(ctx context.Context, matchID string) (domain.MatchNeighbors, error)
	// GetMatchNeighborsFiltered : variante paramétrable Phase 2b. spec=nil/vide
	// → comportement chronologie globale identique à GetMatchNeighbors.
	GetMatchNeighborsFiltered(ctx context.Context, matchID string, spec *domain.MatchFilterSpec) (domain.MatchNeighbors, error)
	// GetObjectiveEvents retourne les events objectif v3 (timeline mode-agnostique)
	// d'un match. Retourne games.ErrCapabilityNotSupported si le titre n'a pas la
	// capability (repo non câblé ou tables absentes).
	GetObjectiveEvents(ctx context.Context, matchID string) ([]domain.ObjectiveEvent, error)
	// GetMatchPositions retourne les positions joueurs keyframe v3 (match-level,
	// §N) d'un match. Retourne games.ErrCapabilityNotSupported si le titre n'a pas
	// la capability (repo non câblé ou table absente).
	GetMatchPositions(ctx context.Context, matchID string) ([]positions.PlayerPosition, error)
}

// ErrReplayNotAvailable est renvoyé quand aucun artefact de rejeu 2D n'existe pour le
// match : la feature s'allume par PRÉSENCE d'artefact (pas de flag global OFF « pour plus
// tard » — règle 11 projet). Le handler le mappe en 404 propre.
var ErrReplayNotAvailable = errors.New("replay: aucun artefact disponible pour ce match")

// ReplayService sert l'artefact de rejeu 2D pré-construit d'un match (trajectoires
// joueurs vue du dessus, produit hors ligne par cmd/replay-build).
type ReplayService interface {
	GetReplay(ctx context.Context, matchID string) (replaydoc.ReplayDocument, error)
	// IsAvailable dit si l'artefact du match existe, SANS le lire. La Match View
	// s'en sert pour ne poser un lien « Rejeu 2D » que là où il mène quelque part :
	// un lien vers une page vide serait pire que pas de lien.
	// Contrat : aucune erreur remontée — un artefact illisible, un titre sans rejeu
	// ou un chemin absent valent tous « pas de rejeu » (dégradation gracieuse).
	IsAvailable(ctx context.Context, matchID string) bool
	// AvailableSet liste EN UN SEUL PASSAGE tous les matchs du titre qui ont un
	// artefact. C'est la forme à utiliser dès qu'on interroge la présence de rejeu
	// pour une LISTE de matchs (tableaux Explorer/escouade, des centaines de lignes) :
	// un IsAvailable par ligne serait un os.Stat par ligne.
	// Erreur : l'appelant dégrade sur un ensemble vide (aucune colonne rejeu), jamais
	// un 500 — l'absence de rejeu n'est pas une panne de page.
	AvailableSet(ctx context.Context) (ReplayAvailability, error)
	// MapBackground retourne le CALAGE du fond de carte du match : où l'image se pose
	// dans le repère monde, celui-là même où vivent les trajectoires. Retourne
	// ErrMapBackgroundNotAvailable quand la carte du match n'a pas d'image figée —
	// 21 cartes en ont, pas toutes : le rejeu retombe alors sur son sol structurel.
	MapBackground(ctx context.Context, matchID string) (*replaydoc.MapBackground, error)
	// MapBackgroundImage retourne les octets PNG du fond, même sentinelle d'absence.
	MapBackgroundImage(ctx context.Context, matchID string) ([]byte, error)
	// MapBackgroundForMap et MapBackgroundImageForMap servent le MÊME fond, keyé par
	// CARTE (map_id) au lieu du match. La grille de l'onglet Tactique n'a pas de match
	// sous la main : elle liste des cartes. La résolution est la même — et elle n'est
	// écrite qu'une fois (resolveBackgroundKeyDepuis dans le service) : seule change la
	// façon d'obtenir les identités de carte (ReplayMapNameRepo.MapKeysForMap).
	// Mêmes sentinelles d'absence que leurs jumelles par match.
	MapBackgroundForMap(ctx context.Context, mapID string) (*replaydoc.MapBackground, error)
	MapBackgroundImageForMap(ctx context.Context, mapID string) ([]byte, error)
	// MapCallouts retourne les ZONES NOMMÉES officielles de la carte du match
	// (polygones monde + libellés FR/EN, catalogue de référence versionné). Retourne
	// ErrMapCalloutsNotAvailable quand la carte n'en a pas — cas nominal des cartes
	// Forge : leur canevas ne porte aucune zone nommée, par construction.
	MapCallouts(ctx context.Context, matchID string) (*replaydoc.MapCalloutsEntry, error)
}

// ErrMapBackgroundNotAvailable est renvoyé quand aucun fond de carte figé n'existe pour la
// carte du match. C'est une ABSENCE NORMALE, pas une panne : toutes les cartes n'ont pas
// d'image (production hors ligne, jeu installé requis), et le client dégrade sur le sol
// reconstruit. Distinct de ErrReplayNotAvailable : un rejeu peut exister sans fond.
var ErrMapBackgroundNotAvailable = errors.New("replay: aucun fond de carte pour ce match")

// ErrMapCalloutsNotAvailable est renvoyé quand la carte du match n'a pas de zones
// nommées au catalogue. ABSENCE NORMALE : les 22 cartes intégrées en ont, les cartes
// Forge n'en auront jamais (leur canevas n'en porte aucune — mesuré). Le client dégrade
// en n'affichant pas le calque zones.
var ErrMapCalloutsNotAvailable = errors.New("replay: aucune zone nommée pour ce match")

// MatchMapKeys sont les identités de carte d'un match. Le map_id (asset UGC) est la clé du
// fond d'une carte FORGE — un canevas partagé par des dizaines de cartes ne peut pas keyer
// un fond ; les noms affichés mènent au module installé des cartes NATIVES (catalogue de
// bornes).
type MatchMapKeys struct {
	// MapID est l'asset UGC du match (match_registry.map_id) ; vide quand la base ne le
	// porte pas.
	MapID string
	// Names sont les noms de carte candidats, du plus fiable au moins fiable.
	Names []string
	// PairName est le pair_name BRUT du match (match_registry.pair_name) : la clé du
	// choix des rôles d'objectif servis avec le rejeu (lot 4 — le service le normalise
	// via analysis.NormalizeModeLabel). Vide quand la base ne le porte pas ; il peut
	// aussi être un UUID brut — l'appelant dégrade alors par absence, il ne devine pas.
	PairName string
}

// ReplayMapNameRepo résout la carte d'un match. Le document de rejeu ne porte aucune
// identité de carte (il est décodé des seuls chunks du film) : c'est la base qui la nomme.
type ReplayMapNameRepo interface {
	// MapKeysForMatch retourne les identités de carte du match (map_id + noms candidats).
	// Erreur = carte inconnue, l'appelant dégrade sans fond.
	MapKeysForMatch(ctx context.Context, matchID string) (MatchMapKeys, error)
	// MapKeysForMap retourne les mêmes identités à partir du SEUL map_id, pour les
	// surfaces qui raisonnent par CARTE et non par match (grille de l'onglet Tactique).
	// Le map_id est alors déjà connu ; ne manquent que les NOMS candidats, qui restent
	// nécessaires — le fond d'une carte native est keyé par son module installé, pas par
	// son asset. `PairName` est toujours vide : il n'existe qu'au niveau d'un match.
	// Erreur = carte inconnue, même dégradation que ci-dessus.
	MapKeysForMap(ctx context.Context, mapID string) (MatchMapKeys, error)
}

// MatchPlayerFact et MatchFacts sont des ALIAS de leurs types de domaine
// (`internal/domain/match_facts.go`), où ils ont été déplacés le 2026-08-24.
//
// POURQUOI ILS ONT DÉMÉNAGÉ, ET POURQUOI L'ALIAS RESTE. Ces faits traversent désormais une
// SECONDE porte que `port` ne peut pas servir : `domain.BuildQueuePayload` les transporte
// jusqu'à l'ouvrier distant, qui n'a aucune base pour les résoudre. Or `domain` n'importe
// jamais `port` — les laisser ici rendrait la file de construction impossible à typer sans
// cycle d'import.
//
// L'alias (et non une redéclaration) est ce qui rend le déplacement INVISIBLE : `port.MatchFacts`
// et `domain.MatchFacts` sont le MÊME type pour le compilateur, si bien qu'aucun des appelants
// existants n'a eu à changer. Ne pas le transformer en type distinct : ce serait deux types à
// tenir synchronisés, et le défaut « copy-paste config » du dépôt.
type MatchPlayerFact = domain.MatchPlayerFact

// MatchFacts est CE QUE LA BASE SAIT DU MATCH ET QUE LE FILM NE DIT PAS. Défini et documenté
// dans `internal/domain/match_facts.go` (dont la mesure de ce que coûte son absence).
type MatchFacts = domain.MatchFacts

// ReplayFactsRepo lit les faits d'un match pour le constructeur d'artefact de rejeu.
type ReplayFactsRepo interface {
	// FactsForMatch retourne les faits du match. Un match absent du registre rend des faits
	// VIDES sans erreur : l'appelant dégrade (artefact sans compteurs de joueur), il n'échoue pas.
	FactsForMatch(ctx context.Context, matchID string) (MatchFacts, error)
}

// ReplayLinkTarget est la cible de lien d'un match. Défini et documenté dans
// `internal/domain/replay_link.go`.
type ReplayLinkTarget = domain.ReplayLinkTarget

// ReplayLinkRepo résout, pour un LOT de matchs, de quoi construire un lien vers leur page
// de rejeu : un joueur connu qui y a participé, et le nom de carte.
//
// Port SÉPARÉ de ReplayFactsRepo bien que le même type concret l'implémente : les faits
// servent à CONSTRUIRE un artefact, ces cibles à ANNONCER un artefact déjà construit. Deux
// besoins, deux contrats — un appelant qui n'a besoin que du second ne doit pas dépendre
// du premier.
type ReplayLinkRepo interface {
	// LinkTargetsForMatches rend une entrée PAR MATCH TROUVÉ au registre, indexée par
	// match_id. knownXUIDs borne la recherche de participant aux joueurs de l'instance ;
	// vide = aucun lien résolu (les entrées sortent avec le nom de carte seul).
	LinkTargetsForMatches(ctx context.Context, matchIDs, knownXUIDs []string) (map[string]ReplayLinkTarget, error)
}

// MatchEventsService construit la timeline canonique d'events d'un match
// (kill-feed / timeline, chargée on-demand), avec résolution des gamertags via
// le chokepoint canonique. Capability-gated : retourne games.ErrCapabilityNotSupported
// si le titre n'expose pas de timeline d'events (le handler dégrade en 503).
type MatchEventsService interface {
	GetMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error)
}

// GamertagResolver résout un set BORNÉ de xuid → gamertag via le chokepoint
// canonique (v_gamertag_lookup). Les xuid non résolus (orphelins hors sources)
// sont ABSENTS de la map — le caller laisse l'identité sans gamertag et le rendu
// applique le masquage (front displayPlayerName). Implémenté par duckdb.GamertagRepo.
type GamertagResolver interface {
	ResolveGamertags(ctx context.Context, xuids []string) (map[string]string, error)
}

// MatchExclusionService gère le marquage et la liste des matchs non pertinents.
type MatchExclusionService interface {
	// SetExclusion marque ou démarque un match comme non pertinent.
	// Retourne ErrRankedMatchNotExcludable si excluded=true sur un match classé.
	// Retourne ErrMatchNotFound si le match_id n'existe pas dans shared.match_registry.
	SetExclusion(ctx context.Context, matchID string, excluded bool) error

	// ListExcluded retourne les matchs exclus du joueur.
	ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error)
}

// MatchRecomputer relance les batches perf_score + LUSR pour propager l'effet
// d'une (dé)exclusion manuelle. Implémenté par sync.MatchRecomputer.
type MatchRecomputer interface {
	// RecomputeAfterExclusion recalcule en force=true le perf_score et le LUSR
	// du joueur. matchID est conservé pour les logs uniquement (le filtrage se
	// fait via le flag is_excluded déjà persisté en DB).
	RecomputeAfterExclusion(ctx context.Context, matchID string) error
}

// MediaService construit la page de la galerie médias et gère l'upload.
type MediaService interface {
	GetMediaPage(ctx context.Context, req domain.MediaPageRequest) (*domain.MediaPageResponse, error)
	SetMediaLike(ctx context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error)
	UploadMedia(ctx context.Context, req domain.UploadRequest) (*domain.UploadResult, error)
	GetMatchCandidates(ctx context.Context, filePath string, windowMinutes int) (*domain.MediaMatchCandidatesResponse, error)
	AssociateMediaToMatch(ctx context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error)
	// ListMediaAuthors retourne les auteurs sélectionnables dans le filtre Auteurs
	// (player_slug + compte + is_self), depuis la DB — gamertag enrichi par le handler.
	ListMediaAuthors(ctx context.Context) ([]domain.MediaAuthor, error)
	// DeleteMedia supprime définitivement un média (fichiers disque + visibilité).
	// L'autorisation propriétaire/admin est appliquée par le service à partir de
	// l'identité résolue par le handler (cf. domain.CanDeleteMedia).
	DeleteMedia(ctx context.Context, req domain.MediaDeleteRequest) (*domain.MediaDeleteResponse, error)
}

// SocialService gère les interactions sociales (favoris de matchs).
type SocialService interface {
	ToggleMatchFavorite(ctx context.Context, req domain.MatchFavoriteRequest) error
}

// SessionsService construit la page des sessions.
type SessionsService interface {
	GetSessions(ctx context.Context, opts domain.SessionComputeOptions) (domain.SessionsResponse, error)
}

// SessionPageService construit la page détail de session.
type SessionPageService interface {
	GetPage(ctx context.Context, req domain.SessionPageRequest) (domain.SessionPageResponse, error)
}

// SquadService orchestre les pages Escouade et Synthèse.
type SquadService interface {
	GetSquadPage(ctx context.Context, playerXUID, gamertag, teammateXUID string) (*domain.SquadPageResponse, error)
	GetSynthesisPage(ctx context.Context, playerXUID string) (*domain.SynthesisPageResponse, error)
}

// SquadV2Service orchestre la page Squad V2 (multi-coéquipiers, fondations Phase 0).
//
// Vit en parallèle de SquadService (legacy mono-coéquipier) jusqu'à migration
// complète des consommateurs frontend (cf. PLAN_SQUAD_GO_PORTAGE).
type SquadV2Service interface {
	GetSquadPage(
		ctx context.Context,
		titleSlug string,
		mainGamertag string,
		teammateGamertags []string,
		period temporal.Period,
		experienceTypes []string,
		playlists []string,
		maps []string,
		modes []string,
	) (*domain.SquadPageV2Response, error)
}

// SynthesisService orchestre la page Synthèse (Sprint 55 D1 — autonome).
type SynthesisService interface {
	GetSynthesisPage(ctx context.Context, playerXUID string, req domain.SynthesisRequest) (*domain.SynthesisPageV2Response, error)
}

// StatsService construit la page Stats/Séries.
type StatsService interface {
	GetPage(ctx context.Context, req domain.StatsQueryRequest) (domain.StatsPageResponse, error)
}

// TeammatesService construit la page Coéquipiers.
type TeammatesService interface {
	GetPage(ctx context.Context, playerXUID string, req domain.TeammatesQueryRequest) (domain.TeammatesPageResponse, error)
}

// TimeseriesService construit la page Séries temporelles.
type TimeseriesService interface {
	GetPage(ctx context.Context, req domain.TimeseriesQueryRequest) (domain.TimeseriesPageResponse, error)
}

// ---------------------------------------------------------------------------
// Services globaux (singletons, résolution non liée à un joueur)
// ---------------------------------------------------------------------------

// BootstrapService construit le bootstrap et la liste des joueurs.
type BootstrapService interface {
	Build(ctx context.Context, sess *domain.SessionData) (*domain.BootstrapResponse, error)
	BuildPlayersList(ctx context.Context, sess *domain.SessionData) (*domain.PlayersListResponse, error)
}

// GamertagSearchService cherche des gamertags dans la base partagée.
type GamertagSearchService interface {
	Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error)
}

// ProfileService gère la création de profils joueur (extrait de setup.go).
type ProfileService interface {
	CreatePlayer(req domain.CreatePlayerProfileRequest) (playerKey string, warnings []string, err error)
}

// FriendsOrchestrator déclenche le recompute is_with_friends sur toutes les
// player DBs configurées (multi-titres). §4 du plan Squad/Sessions overhaul.
//
// Implémenté par *service.FriendsOrchestratorService. Le résultat n'est pas
// typé ici pour éviter une fuite de service vers port — les handlers le
// consomment via le type concret quand ils ont besoin du détail.
type FriendsOrchestrator interface {
	OnFriendsChanged(ctx context.Context) error
}

// ─── Asset Drawer ────────────────────────────────────────────────────────────

// AssetService construit les réponses de l'Asset Drawer (maps, armes).
type AssetService interface {
	// ListMaps retourne les maps d'un titre avec image_url, filtrées par search.
	ListMaps(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)

	// ListWeapons retourne les armes d'un titre avec image_url, filtrées par search.
	ListWeapons(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)

	// ListMedals retourne les médailles d'un titre (icône sprite), filtrées par search.
	ListMedals(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
}

// ─── Sprint 54 : Compare, Leaderboard ────────────────────────────────────────

// CompareService construit la page Compare joueur vs joueur.
type CompareService interface {
	GetPage(ctx context.Context, req domain.CompareRequest) (domain.CompareResponse, error)
}

// LeaderboardService construit la page Classement CSR.
type LeaderboardService interface {
	GetPage(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error)
	// GetCatalog liste saisons + playlists ayant des snapshots (sélecteurs).
	GetCatalog(ctx context.Context) (domain.LeaderboardCatalog, error)
}

// PlayerStatsProvider fournit les stats d'un joueur distant via Waypoint.
// Implémenté par platform/halo.CompareProvider.
type PlayerStatsProvider interface {
	FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error)
}

// ServiceRecordProvider fournit le service record complet d'un joueur distant
// (stats + temps de jeu + médailles lifetime) en un seul appel Waypoint.
// Implémenté par platform/halo.HaloProvider et décoré par service.CachedStatsProvider.
type ServiceRecordProvider interface {
	FetchServiceRecord(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error)
}

// SeasonStatsProvider fournit le nombre de matchs matchmade complétés par un
// joueur sur UNE saison donnée (service record Waypoint filtré par seasonId,
// optionnellement par isRanked). isRanked=nil → total de la saison. Implémenté
// par platform/halo.HaloProvider et décoré (cache) par service.CachedStatsProvider.
type SeasonStatsProvider interface {
	FetchSeasonServiceRecord(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error)
}

// PrivacyProvider interroge la privacy d'un compte Halo via Waypoint.
// Implémenté par platform/halo.HaloProvider.
type PrivacyProvider interface {
	GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error)
}

// RecentMatchesProvider fournit les `limit` derniers matchs PvP d'un joueur
// (xuid) fetchés en LIVE depuis l'API Halo, SANS aucune persistance (lecture
// seule, mis en cache mémoire à TTL court). Sert à alimenter les graphes "profil
// de combat" d'une cible NON suivie (dont les matchs ne sont pas en base) et
// l'échantillon récent du Face à face. Les tokens sont lus dans le contexte.
// Implémenté par sync.RecentMatchesFetcher, décoré par
// service.CachedRecentMatchesProvider (cache TTL). (nil, nil) sans auth.
type RecentMatchesProvider interface {
	FetchRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error)
}
