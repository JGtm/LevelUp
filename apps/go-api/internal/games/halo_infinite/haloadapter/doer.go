package haloadapter

// Les surfaces sortantes vers Halo, une par point de sortie : chacune a son `netguard.Doer` gardé
// et sa ligne dans le journal de `netguard` (une par surface et par processus).
const (
	surfaceAPIGet           = "halo_api.get"
	surfaceAPIBlob          = "halo_api.download_blob"
	surfaceProviderGet      = "halo_provider.get"
	surfaceDiscoveryGet     = "halo_discovery.get"
	surfaceNameplateMapping = "nameplate_resolver.mapping"
	surfaceNameplateEmblem  = "nameplate_resolver.emblem"
	surfaceClassementPage   = "waypoint_leaderboard.scrape"
	surfaceClassementSaison = "waypoint_leaderboard.discover_season"
)
