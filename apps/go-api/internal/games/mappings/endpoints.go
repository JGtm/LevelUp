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
	byKey         map[EndpointKey]string
}

// NewEndpointSet construit un EndpointSet (utilisé par le loader et les tests).
func NewEndpointSet(titleSlug string, schemaVersion int, byKey map[EndpointKey]string) *EndpointSet {
	if byKey == nil {
		byKey = make(map[EndpointKey]string)
	}
	return &EndpointSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byKey:         byKey,
	}
}

// TitleSlug retourne le slug du titre porteur.
func (s *EndpointSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *EndpointSet) SchemaVersion() int { return s.schemaVersion }

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
