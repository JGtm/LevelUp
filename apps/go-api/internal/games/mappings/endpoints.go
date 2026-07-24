package mappings

import (
	"sort"
	"time"
)

// EndpointKey identifie un host d'ingestion API d'un titre (axe MT-01).
//
// Chaque clé correspond à un host distinct consommé par la couche d'ingestion
// (platform/halo + internal/sync). Deux clés peuvent pointer le même host
// physique (ex. Nameplate et GameCMS partagent gamecms-hacs) ; elles restent
// distinctes pour que le Contract migre chaque call-site byte-pour-byte sans
// présumer d'une fusion.
type EndpointKey string

const (
	// EndpointStats — service stats/historique de matchs (halostats, port :443).
	EndpointStats EndpointKey = "stats"
	// EndpointGameCMS — référentiels CMS (gamecms-hacs).
	EndpointGameCMS EndpointKey = "gamecms"
	// EndpointEconomy — battlepass/économie (economy.svc).
	EndpointEconomy EndpointKey = "economy"
	// EndpointSkill — MMR/CSR (skill.svc, port :443).
	EndpointSkill EndpointKey = "skill"
	// EndpointUGCFilm — films / UGC discovery côté ingestion film.
	EndpointUGCFilm EndpointKey = "ugc_film"
	// EndpointDiscoveryUGC — discovery UGC (maps/playlists/variants).
	EndpointDiscoveryUGC EndpointKey = "discovery_ugc"
	// EndpointChallenges — défis (halostats, sans port explicite).
	EndpointChallenges EndpointKey = "challenges"
	// EndpointNameplate — résolution nameplate (gamecms-hacs).
	EndpointNameplate EndpointKey = "nameplate"
)

// EndpointSet est l'ensemble des hosts d'ingestion d'un titre, chargé depuis la
// section [endpoints] de config/titles/{slug}/mappings/constants.toml. Exposé en
// lecture seule ; une clé absente signifie « le titre ne supporte pas cet axe
// d'ingestion » → le caller dégrade (skip + warn), il ne fallback jamais
// silencieusement vers l'host d'un autre titre.
type EndpointSet struct {
	titleSlug     string
	schemaVersion int
	gamePrefix    string
	byKey         map[EndpointKey]string
	damageModel   DamageModelConstants
	engagement    EngagementConstants
	careerXPEras  []CareerXPEra
}

// CareerXPEra décrit un intervalle temporel sur lequel un multiplicateur uniforme
// s'applique à l'XP de carrière gagnée par match (XP = multiplicateur × personal_score).
// Chargé depuis constants.toml [[career_xp_eras]] (dates parsées + validées au boot).
//
// Bornes : From INCLUSIVE, To EXCLUSIVE. From.IsZero() = borne de début ouverte
// (-inf) ; To.IsZero() = borne de fin ouverte (+inf). Multiplier > 0. Halo Infinite :
// ×1 avant le 2025-11-18 (minuit UTC), ×2 depuis (doublement permanent Operation:
// Infinite). Consommé via games.CareerXPErasFor → analysis.EstimateCareerXP.
type CareerXPEra struct {
	From       time.Time
	To         time.Time
	Multiplier float64
}

// EngagementConstants porte les poids d'events du score d'engagement d'un titre
// (constants.toml [engagement], chantier F7). Ce sont les coefficients DÉPENDANTS
// DU GAMEPLAY (importance relative objectif/assist/mort/défaut dans le rythme de la
// courbe). Zéro-value (Default == 0) = section absente → le caller applique le défaut
// byte-identique (temporal.DefaultEventWeights). Déclarés = surcharge par titre.
type EngagementConstants struct {
	Objective float64
	Assist    float64
	Death     float64
	Default   float64
}

// DamageModelConstants porte les constantes de modèle de dégâts d'un titre
// (constants.toml [damage_model]). EffectiveHpToKill = PV effectifs pour tuer un
// joueur (bouclier + armure, échelle de dégâts de l'API), baseline des KPI
// rendement/résistance. Zéro-value (0) = non déclaré → le caller applique son défaut.
type DamageModelConstants struct {
	EffectiveHpToKill float64
	// NoNativeKDA = true → le titre ne fournit pas de KDA per-match via son API
	// (Halo 5 ; forme native = FDA NET). Défaut false (KDA natif, Infinite).
	NoNativeKDA bool
	// NoDamageTaken = true → le titre ne fournit PAS damage_taken (Halo 5). La
	// résistance défensive (DR = dégâts_subis/(hp×morts)) et tout ce qui en dépend
	// (profil de combat axe défensif, coaching « fragile », milestones endurance/
	// excellence) sont neutralisés plutôt qu'affichés faux. Défaut false (Infinite).
	NoDamageTaken bool
	// OffensiveConversionP80 = frontière élite (80e percentile) du rendement OC du
	// titre, repère de normalisation des barres/radars. 0 = non déclaré → défaut
	// 0.90 (Infinite). h5 = 1.264 (calibré sur sa propre distribution, hp=115).
	OffensiveConversionP80 float64
	// NoTeamMMR = true → le titre ne fournit PAS de MMR d'équipe/adverse par match
	// (Halo 5). La colonne MMR du tableau Escouade/Explorer est masquée (valeur nil)
	// plutôt qu'affichée à 0 (trompeur). Défaut false (MMR fourni, Infinite).
	NoTeamMMR bool
	// NB — le support du max killing spree N'EST PAS un champ du modèle de dégâts : il
	// est DÉRIVÉ de la capability events-timeline du titre (cf. games.ProvidesMaxKillingSpree).
	// Un titre qui sert des kills horodatés peut calculer la spree (analysis.ComputeMaxKillingSpree)
	// même quand la valeur native est absente (Halo 5 : match_participants.max_killing_spree NULL).
}

// NewEndpointSet construit un EndpointSet (utilisé par le loader et les tests).
// gamePrefix est le segment d'URL de jeu ("hi"/"h5") ; vide = non déclaré (le
// consommateur retombe sur le défaut).
func NewEndpointSet(titleSlug string, schemaVersion int, gamePrefix string, byKey map[EndpointKey]string) *EndpointSet {
	if byKey == nil {
		byKey = make(map[EndpointKey]string)
	}
	return &EndpointSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		gamePrefix:    gamePrefix,
		byKey:         byKey,
	}
}

// TitleSlug retourne le slug du titre porteur.
func (s *EndpointSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *EndpointSet) SchemaVersion() int { return s.schemaVersion }

// GamePrefix retourne le segment d'URL de jeu du titre ("hi"/"h5") et true s'il
// est déclaré. (_, false) si absent → le caller applique son défaut byte-identique.
func (s *EndpointSet) GamePrefix() (string, bool) {
	if s == nil || s.gamePrefix == "" {
		return "", false
	}
	return s.gamePrefix, true
}

// DamageModel retourne les constantes de modèle de dégâts du titre et true si
// effective_hp_to_kill est déclaré (> 0). (_, false) si absent → le caller
// applique son défaut byte-identique (cf. games.DefaultEffectiveHpToKill).
func (s *EndpointSet) DamageModel() (DamageModelConstants, bool) {
	if s == nil || s.damageModel.EffectiveHpToKill <= 0 {
		return DamageModelConstants{}, false
	}
	return s.damageModel, true
}

// withDamageModel attache les constantes de modèle de dégâts (appelé par le
// loader après construction ; même package). Chaînable.
func (s *EndpointSet) withDamageModel(dm DamageModelConstants) *EndpointSet {
	if s != nil {
		s.damageModel = dm
	}
	return s
}

// Engagement retourne les poids d'events du titre et true si la section est
// déclarée (Default > 0). (_, false) si absente → le caller applique le défaut
// byte-identique (temporal.DefaultEventWeights). Un poids « default » à 0 n'a pas
// de sens (tout event non-spécial pèserait 0) : il sert de sentinelle de présence.
func (s *EndpointSet) Engagement() (EngagementConstants, bool) {
	if s == nil || s.engagement.Default <= 0 {
		return EngagementConstants{}, false
	}
	return s.engagement, true
}

// withEngagement attache les poids d'engagement (appelé par le loader ; même
// package). Chaînable.
func (s *EndpointSet) withEngagement(e EngagementConstants) *EndpointSet {
	if s != nil {
		s.engagement = e
	}
	return s
}

// CareerXPEras retourne les éras de multiplicateur d'XP de carrière du titre et
// true si au moins une éra est déclarée (constants.toml [[career_xp_eras]]).
// (_, false) si absentes → le caller applique games.DefaultCareerXPEras
// (byte-identique Infinite : ×1 avant 2025-11-18, ×2 depuis).
func (s *EndpointSet) CareerXPEras() ([]CareerXPEra, bool) {
	if s == nil || len(s.careerXPEras) == 0 {
		return nil, false
	}
	return s.careerXPEras, true
}

// withCareerXPEras attache les éras d'XP de carrière (appelé par le loader ; même
// package). Chaînable.
func (s *EndpointSet) withCareerXPEras(eras []CareerXPEra) *EndpointSet {
	if s != nil {
		s.careerXPEras = eras
	}
	return s
}

// Host retourne l'host pour une clé d'endpoint, ou (_, false) si absente.
func (s *EndpointSet) Host(key EndpointKey) (string, bool) {
	if s == nil {
		return "", false
	}
	h, ok := s.byKey[key]
	return h, ok
}

// Keys retourne les clés d'endpoint présentes, triées.
func (s *EndpointSet) Keys() []EndpointKey {
	if s == nil {
		return nil
	}
	out := make([]EndpointKey, 0, len(s.byKey))
	for k := range s.byKey {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
