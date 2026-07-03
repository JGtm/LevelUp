package observability

// Sources legacy d'authentification tracees pour la telemetrie de depreciation
// (ADR 0023 Phase 5 ; prerequis du lot D2 « suppression des fallbacks legacy »,
// audits 2026-07 D1a). Ensemble BORNE — ne jamais deriver un nom de compteur d'un
// gamertag/xuid (cardinalite expvar). Les libelles reprennent le vocabulaire des
// sources du scan pool (scan_snapshot.go recordScanSource) pour rester coherent
// avec le dashboard admin « Sante des tokens ».
const (
	LegacySourceDuckDBMSAL  = "duckdb_msal"
	LegacySourceDuckDBOAuth = "duckdb_oauth"
	LegacySourceEnvOAuth    = "env_oauth"
	LegacySourceMonoUser    = "watcher_legacy"
)

// RecordLegacySourceUsed incremente le compteur expvar legacy_source_used_<source>
// (expose sous /debug/vars, cle "levelup"). A appeler quand une source d'auth
// legacy est reellement ATTEINTE/utilisee pour un refresh — jamais a la simple
// lecture bas-niveau (double comptage). `source` DOIT etre une des constantes
// LegacySource* ci-dessus. Ce compteur est le signal machine consomme par le gate
// D2 : tant qu'il est > 0 en prod, des installs dependent encore du fallback legacy.
func RecordLegacySourceUsed(source string) {
	IncCounter("legacy_source_used_" + source)
}
