package mappings

import "sort"

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
	// NoMMR = true → le titre ne fournit PAS de MMR (Halo 5 : PreMatch/PostMatchRatings
	// servis null partout). Les surfaces MMR (team/enemy/delta_mmr, highlight underdog)
	// sont omises plutôt qu'affichées vides. Défaut false (Infinite). Via games.ProvidesMMR.
	NoMMR bool
	// OffensiveConversionP80 = frontière élite (80e percentile) du rendement OC du
	// titre, repère de normalisation des barres/radars. 0 = non déclaré → défaut
	// 0.90 (Infinite). h5 = 1.264 (calibré sur sa propre distribution, hp=115).
	OffensiveConversionP80 float64
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
