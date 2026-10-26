// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// HomeService orchestre les données de la page d'accueil.
type HomeService struct {
	repo       port.HomeRepository
	cacheRepo  port.BattlePassCacheRepository
	provider   *halo.HaloProvider
	sink       port.HomePersistSink // nil → pas de persistance (tests, joueurs sans auth)
	socialRepo port.SocialRepository
	playerSlug string
	semantic   games.TitleSemanticAdapter // nil → libellés rangs construits via fallbacks (RankName)
	xuid       string                     // xuid du joueur de la PAGE (identité Spartan, coéquipiers de session)
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadPlayerStats via la couche canonique. À ce jour, le service
	// utilise le repo direct car canonical.PlayerStats ne couvre pas encore
	// la totalité du payload home KPIs (favorite_playlist, avg_kda, etc.).
	// Le hook est en place pour permettre une bascule incrémentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, fetchMatchesAndSessions charge
	// canonical et convertit via homeMatchRowFromCanonical / homeSessionsFromCanonical.
	// SkillTierLabel et SkillRankImageURL sont laissés vides côté converter
	// (TODO P4.3 : enrichir via TitleSemanticAdapter.Ranks() et
	// TitleAssetURLAdapter.CSRRankImageURL une fois les adapters CSR câblés
	// dans le service).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// careerLive (optionnel) : service live carrière découplé du post-sync
	// matchs. Quand câblé, remplace l'appel DB-only LoadSpartanIdentity par
	// un flow live throttle 5 min / 6 h + fallback DB per-field garanti.
	// Si nil, le service retombe sur l'ancien chemin (repo.LoadSpartanIdentity)
	// — utilisé par les tests et les bootstraps minimaux sans auth.
	careerLive homeSpartanIdentityProvider
	// mapGroups (optionnel) : groupes de cartes (la carte favorite compte une variante sous sa base). nil : cartes seules.
	mapGroups port.MapGroupReader
	// demoMode : en démo, certaines sections sans source réelle (défis : pas d'API
	// live + cache TTL 24h) sont servies depuis des fixtures embarquées au lieu de
	// renvoyer vide. Cf. home_service_demo.go.
	demoMode bool
	// skillBadgeResolver (optionnel, multi-titres) : résout l'URL du badge CSR
	// title-aware (csr_designations h5 sinon static HINF). Injecté au boot avec le
	// slug du titre du joueur. Nil → SkillRankImageURL laissé vide côté tuiles de
	// match (dégradation gracieuse, le label suffit). Signature (tierEN capitalisé,
	// subTier 0..6 ; 0 = Onyx) → URL. Garde le package analysis title-agnostic.
	skillBadgeResolver func(tierEN string, subTier int) string
	// roundsDecide : game_variant_name -> le RÉSULTAT du match se lit en MANCHES
	// (regulation.toml [rounds_decide], ADR 0032). Nil/absente -> les tuiles affichent le
	// score de l'API, comme avant. Injectée par WithRoundsDecide.
	roundsDecide map[string]bool
	// sessionTeammatesLoader (optionnel) : charge les coéquipiers (même équipe que
	// le joueur principal) sur une liste de matchs, pour renseigner
	// SessionSummaryItem.Teammates des sessions escouade (deep-link card → /squad).
	// Implémenté par *duckdb.SquadRepo. nil → Teammates restent vides (dégradation).
	// Cf. home_squad_session_teammates.go.
	sessionTeammatesLoader mainTeamParticipantsLoader
	// sessionFriendsResolver (optionnel) : restreint les coéquipiers de session aux
	// amis configurés du joueur consulté. nil → tous les coéquipiers alliés.
	sessionFriendsResolver teammates.FriendGamertagsResolver
	// replaySvc (optionnel) : service de rejeu 2D — MÊME service que l'endpoint /replay
	// et l'Explorer. Seul AvailableSet est appelé : un listing de dossier par requête,
	// jamais un accès disque par tuile. Nil → HasReplay reste faux partout (titre sans
	// rejeu construit — dégradation gracieuse, aucun lien mort sur les tuiles).
	replaySvc port.ReplayService
	// analysisPending (optionnel) : résolveur du signal `analysis_pending` des tuiles. Nil (titre
	// sans la capability du film, tests) → le champ reste faux partout.
	analysisPending *AnalysisPendingResolver
}

// NewHomeService crée un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// WithDemoMode active les fixtures de contenu démo (défis…) servies au niveau
// lecture quand aucune source réelle n'existe en démo. Retourne le service (chaînage).
func (s *HomeService) WithDemoMode(demo bool) *HomeService {
	s.demoMode = demo
	return s
}

// WithSkillBadgeResolver injecte le résolveur d'URL de badge CSR title-aware
// (csr_designations pour les titres additionnels, sinon static HINF). Câblé au
// boot avec le slug du titre du joueur. Dégradation gracieuse si nil (badge vide
// sur les tuiles de match — le label de palier reste affiché). Retourne le service
// pour permettre le chaînage.
func (s *HomeService) WithSkillBadgeResolver(f func(tierEN string, subTier int) string) *HomeService {
	s.skillBadgeResolver = f
	return s
}

// WithRoundsDecide injecte la table `game_variant_name -> le résultat se lit en MANCHES`
// (regulation.toml [rounds_decide], ADR 0032). Sans injection, les tuiles d'accueil
// affichent le score de l'API — jamais une régression, mais une incohérence avec la vue
// match si le titre en déclare : c'est pourquoi les TROIS factories l'injectent.
func (s *HomeService) WithRoundsDecide(roundsDecide map[string]bool) *HomeService {
	s.roundsDecide = roundsDecide
	return s
}

// WithHaloProvider remplace le provider Halo utilisé par le service.
// Utile pour injecter un provider configuré par joueur (cache local, tests).
func (s *HomeService) WithHaloProvider(provider *halo.HaloProvider) *HomeService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

// WithPersistSink configure le sink de persistance fire-and-forget.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithPersistSink(sink port.HomePersistSink) *HomeService {
	s.sink = sink
	return s
}

// WithCacheRepo configure le repository de cache BP/Challenges.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithCacheRepo(r port.BattlePassCacheRepository) *HomeService {
	s.cacheRepo = r
	return s
}

// WithSocial configure le repository social (favoris) et le slug joueur.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithSocial(repo port.SocialRepository, playerSlug string) *HomeService {
	s.socialRepo = repo
	s.playerSlug = playerSlug
	return s
}

// WithSemanticAdapter injecte le SemanticAdapter du titre courant pour résoudre
// les libellés des rangs de carrière (Ranks() expose un *mappings.RankCatalog).
// Si nil, les libellés tombent sur le fallback RankName de la player DB.
func (s *HomeService) WithSemanticAdapter(semantic games.TitleSemanticAdapter) *HomeService {
	s.semantic = semantic
	return s
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadPlayerStats. Dégradation gracieuse si nil.
func (s *HomeService) WithDataAdapter(a games.TitleDataAdapter) *HomeService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware
// + titleSlug + gamertag. Quand les 3 sont fournis, fetchMatchesAndSessions
// charge canonical et convertit. Sinon fallback repo legacy.
func (s *HomeService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *HomeService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithXUID pose le xuid du joueur de la PAGE : sujet explicite de l'identité Spartan live et de la
// résolution des coéquipiers de session (jamais lu du contexte ambiant). Retourne le service.
func (s *HomeService) WithXUID(xuid string) *HomeService {
	s.xuid = xuid
	return s
}

// WithCareerLive câble le service live carrière (XP/rang + Spartan ID
// découplés du post-sync matchs). Quand fourni, GetHomePage remplace
// l'appel repo.LoadSpartanIdentity (DB-only) par un flow live throttle
// 5 min / 6 h + fallback DB per-field garanti.
// homeSpartanIdentityProvider : contrat minimal consommé par HomeService pour
// l'identité Spartan live (implémenté par *CareerLiveService). Interface
// consumer-side (K1i, ARCHI 8) — le champ n'est plus typé sur le concret, mockable
// en test (même package).
type homeSpartanIdentityProvider interface {
	// GetSpartanIdentityFor résout l'identité Spartan du SUJET passé explicitement
	// (finding ID4) : la home fournit le xuid de la PAGE (s.xuid = pdb.XUID), jamais
	// une valeur lue du contexte ambiant.
	GetSpartanIdentityFor(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error)
}

func (s *HomeService) WithCareerLive(svc *CareerLiveService) *HomeService {
	if svc != nil { // garde concret fiable — évite le piège interface typed-nil
		s.careerLive = svc
	}
	return s
}

// SetSessionActive implémente port.SessionNotifier.
// Conservé pour compatibilité avec le watcher — aucun effet sur le handler HTTP
// qui appelle toujours le live directement.
func (s *HomeService) SetSessionActive(_ bool) {
}
