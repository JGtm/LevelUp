// Package domain — admin_monitoring.go : payloads du dashboard monitoring
// admin (vue d'ensemble, convergence, data health).
//
// Tous les agrégats sont best-effort : une section indisponible est signalée
// par son champ *_error ou un pointeur nil, jamais par un 500 global (pattern
// admin_invariants). Les payloads scheduler (snapshot + historique) ne sont
// pas dupliqués ici : ils réutilisent les types JSON-contractés du package
// scheduler (déjà exposés par /_diag/auto-sync/snapshot).
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Statuts du cycle de vie d'une détection (DC-2 du plan monitoring 2026-07).
// Une occurrence après DetectionStatusResolved ré-ouvre la détection.
const (
	DetectionStatusOpen     = "open"
	DetectionStatusAcked    = "acked"
	DetectionStatusMuted    = "muted"
	DetectionStatusResolved = "resolved"
)

// IsValidDetectionStatus valide un statut de cycle de vie de détection.
func IsValidDetectionStatus(status string) bool {
	switch status {
	case DetectionStatusOpen, DetectionStatusAcked, DetectionStatusMuted, DetectionStatusResolved:
		return true
	default:
		return false
	}
}

// DetectionFingerprint calcule la clé stable d'une détection à partir de
// (title, level, module, message) — même dimension que l'ErrorCollector (DC-2).
// Hash hex tronqué : sûr comme segment d'URL (endpoint PATCH .../{fingerprint})
// là où message/module bruts casseraient le routage.
func DetectionFingerprint(title, level, module, message string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{title, level, module, message}, "\x1f")))
	return hex.EncodeToString(sum[:12]) // 24 hex chars, collision négligeable
}

// MonitoringDetection — détection persistée avec son cycle de vie (vue
// detections_latest). Remplace AdminErrorBucket (mémoire, perdu au restart)
// pour le triage : count cumulé, first/last_seen, statut actionnable.
type MonitoringDetection struct {
	Fingerprint  string `json:"fingerprint"`
	Level        string `json:"level"`
	Module       string `json:"module,omitempty"`
	Message      string `json:"message"`
	TitleSlug    string `json:"title_slug,omitempty"`
	Count        int64  `json:"count"`
	FirstSeen    string `json:"first_seen"` // RFC3339
	LastSeen     string `json:"last_seen"`  // RFC3339
	SampleDetail string `json:"sample_detail,omitempty"`
	Status       string `json:"status"`
	Note         string `json:"note,omitempty"`
	StatusAt     string `json:"status_at,omitempty"` // RFC3339, vide si jamais statué
}

// AdminDetectionsResponse — réponse de GET /admin/monitoring/detections.
type AdminDetectionsResponse struct {
	GeneratedAt string                `json:"generated_at"`
	Detections  []MonitoringDetection `json:"detections"`
	// OpenCount : nombre de détections au statut open (source des badges nav).
	OpenCount int `json:"open_count"`
}

// Statuts de fraîcheur des données d'un joueur (A4, seuils DC-3).
const (
	FreshnessStatusOK       = "ok"
	FreshnessStatusWarn     = "warn"
	FreshnessStatusCritical = "critical"
	FreshnessStatusUnknown  = "unknown"
)

// PlayerFreshness — fraîcheur des données d'un joueur suivi.
type PlayerFreshness struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
	// LastMatchAt : dernier match persisté (RFC3339, vide si aucun).
	LastMatchAt     string `json:"last_match_at,omitempty"`
	MatchAgeSeconds int64  `json:"match_age_seconds,omitempty"`
	// LastSyncOKAt : dernier cycle sync réussi (RFC3339, vide si inconnu —
	// titre hors scheduler V2 ou jamais couru depuis le boot).
	LastSyncOKAt   string `json:"last_sync_ok_at,omitempty"`
	SyncAgeSeconds int64  `json:"sync_age_seconds,omitempty"`
	// Status : ok / warn / critical / unknown (DB inaccessible).
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	CheckError string `json:"check_error,omitempty"`
}

// TitleFreshnessReport — fraîcheur par titre actif.
type TitleFreshnessReport struct {
	TitleSlug string            `json:"title_slug"`
	Players   []PlayerFreshness `json:"players"`
	// Note non vide = titre sans la capability requise / sans joueur suivi
	// (dégradation gracieuse, jamais d'erreur globale).
	Note          string `json:"note,omitempty"`
	WarnCount     int    `json:"warn_count"`
	CriticalCount int    `json:"critical_count"`
}

// FreshnessBackupInfo — âge du dernier backup réussi (A4.2 ; source :
// manifest du scheduler duckdbbackup — décision consignée au plan).
type FreshnessBackupInfo struct {
	Enabled      bool   `json:"enabled"`
	LastBackupAt string `json:"last_backup_at,omitempty"` // RFC3339, vide si jamais
	AgeSeconds   int64  `json:"age_seconds,omitempty"`
}

// AdminFreshnessResponse — réponse de GET /admin/monitoring/freshness.
type AdminFreshnessResponse struct {
	GeneratedAt string                 `json:"generated_at"`
	Titles      []TitleFreshnessReport `json:"titles"`
	// Backup : nil si scheduler backup non câblé.
	Backup *FreshnessBackupInfo `json:"backup,omitempty"`
	// CriticalTotal : joueurs critical tous titres (source du badge État).
	CriticalTotal int `json:"critical_total"`
}

// CronFailuresCriticalThreshold : échecs consécutifs à partir desquels une
// ligne cron passe critical (A6.3).
const CronFailuresCriticalThreshold = 3

// CronStatusEntry — statut d'un cron (A6, DC-5). Fusion registre mémoire
// (depuis le boot) + dernier run persisté (cron_runs — survit au restart).
type CronStatusEntry struct {
	Name          string `json:"name"`
	LastRunAt     string `json:"last_run_at,omitempty"`     // RFC3339, vide si jamais vu
	LastSuccessAt string `json:"last_success_at,omitempty"` // RFC3339
	LastError     string `json:"last_error,omitempty"`
	// ConsecutiveFailures : depuis le boot (le registre mémoire fait foi).
	ConsecutiveFailures int   `json:"consecutive_failures"`
	Runs                int64 `json:"runs"`
	LastDurationMs      int64 `json:"last_duration_ms,omitempty"`
	// Status : ok / warn (dernier run en échec) / critical (>= seuil consécutif)
	// / unknown (jamais vu, ni en mémoire ni persisté).
	Status string `json:"status"`
	// SinceBoot : false = donnée réhydratée depuis cron_runs (pas encore couru
	// depuis ce boot).
	SinceBoot bool `json:"since_boot"`
}

// FeatureHeartbeat — liveness d'une feature (A6.2, liste fermée DC-5).
type FeatureHeartbeat struct {
	Feature    string `json:"feature"`
	LastSeenAt string `json:"last_seen_at,omitempty"` // RFC3339, vide si jamais vu
	AgeSeconds int64  `json:"age_seconds,omitempty"`
	// Status : ok / never (jamais vu depuis le boot → accent destructive).
	Status string `json:"status"`
}

// AdminCronsResponse — réponse de GET /admin/monitoring/crons (A6.3).
type AdminCronsResponse struct {
	GeneratedAt string             `json:"generated_at"`
	Crons       []CronStatusEntry  `json:"crons"`
	Features    []FeatureHeartbeat `json:"features"`
}

// ResourceRuntime — état runtime Go du process (A5, DC-4).
type ResourceRuntime struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64 `json:"heap_sys_bytes"`
	// SysBytes : total demandé à l'OS par le runtime (approx. RSS).
	SysBytes uint64 `json:"sys_bytes"`
	NumGC    uint32 `json:"num_gc"`
}

// ResourceDisk — espace libre du volume data (A5.3 : seuils nommés côté ops).
type ResourceDisk struct {
	Path       string `json:"path"`
	FreeBytes  uint64 `json:"free_bytes"`
	TotalBytes uint64 `json:"total_bytes"`
	// Status : ok / warn (< seuil warn) / critical (< seuil critical) / unknown.
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ResourceDBFile — taille d'une base DuckDB + WAL éventuel.
type ResourceDBFile struct {
	// Name : libellé stable "{title}/{base}" ou "global/{base}".
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	// WalBytes : taille du .wal adjacent (0 = absent).
	WalBytes int64 `json:"wal_bytes,omitempty"`
}

// Statuts d'inventaire des bases (DBInventoryStatus).
const (
	// DBInventoryOK : racine data lisible, l'inventaire est fiable.
	DBInventoryOK = "ok"
	// DBInventoryUnavailable : racine data introuvable/illisible (RepoRoot mal
	// résolu, volume non monté, permissions) — inventaire non mesurable.
	DBInventoryUnavailable = "unavailable"
)

// AdminResourcesResponse — réponse de GET /admin/monitoring/resources (A5).
type AdminResourcesResponse struct {
	GeneratedAt string          `json:"generated_at"`
	Runtime     ResourceRuntime `json:"runtime"`
	UptimeS     int64           `json:"uptime_s"`
	// Restarts : démarrages du serveur enregistrés dans la base monitoring
	// (marqueur server_boot dans cron_runs — persiste au restart). 0 si store absent.
	Restarts int64        `json:"restarts"`
	Disk     ResourceDisk `json:"disk"`
	// Databases : shared/metadata/pve/social par titre actif + players agrégés
	// + bases globales (aliases, monitoring).
	Databases    []ResourceDBFile `json:"databases"`
	DBTotalBytes int64            `json:"db_total_bytes"`
	// DBInventoryStatus distingue à l'UI « inventaire indisponible »
	// (environnemental) d'un « aucune base » — sans lui, une racine erronée
	// produit silencieusement une table de tailles nulles trompeuse.
	DBInventoryStatus string `json:"db_inventory_status" enum:"ok,unavailable" doc:"ok = racine data lisible ; unavailable = racine data introuvable/illisible (RepoRoot mal résolu ou volume non monté)."`
	// Budgets / PoolStats : relecture des snapshots expvar existants (J1/J8).
	Budgets   map[string]interface{} `json:"budgets,omitempty"`
	PoolStats map[string]interface{} `json:"pool_stats,omitempty"`
}

// MonitoringServerInfo : identité du process serveur (overview).
type MonitoringServerInfo struct {
	UptimeS   int64  `json:"uptime_s"`
	StartedAt string `json:"started_at"` // RFC3339
	Version   string `json:"version"`
}

// AdminMonitoringOverview est la réponse de GET /admin/monitoring/overview.
// Conçue ZÉRO I/O DuckDB : tout provient d'états mémoire (snapshot scheduler,
// dernier data health check, JobStore, token store fichiers, expvar) pour
// supporter un polling 30 s sans coût.
type AdminMonitoringOverview struct {
	TitleSlug   string `json:"title_slug"`
	GeneratedAt string `json:"generated_at"` // RFC3339

	Server    MonitoringServerInfo       `json:"server"`
	Scheduler MonitoringSchedulerSummary `json:"scheduler"`
	Jobs      MonitoringJobsSummary      `json:"jobs"`

	// DataHealth : dernier audit du HealthScheduler (24h ou action manuelle).
	// Nil si aucun cycle complet depuis le boot.
	DataHealth *MonitoringDataHealth `json:"data_health,omitempty"`

	// Tokens : agrégat de la santé des tokens (détail sur /admin/token-health).
	Tokens      *MonitoringTokensSummary `json:"tokens,omitempty"`
	TokensError string                   `json:"tokens_error,omitempty"`

	// Invariants : état du DERNIER run (gauges expvar) — pas de re-exécution.
	Invariants MonitoringInvariantsSummary `json:"invariants"`

	// Snapshot : état du substrat immuable (durabilité / lecture découplée du B-swap).
	// Gauges = état courant (version, ready, backlog) ; cumuls = cuts depuis le boot.
	Snapshot MonitoringSnapshotSummary `json:"snapshot"`

	// OpenDetections : nombre de détections persistées au statut open (gauge posé
	// par le flush du store monitoring — source du badge nav « Détections »).
	// Zéro I/O DuckDB ici : simple lecture d'un compteur expvar.
	OpenDetections int64 `json:"open_detections"`

	// FreshnessCritical : joueurs en fraîcheur critical tous titres (gauge posé
	// par le calcul de GET /admin/monitoring/freshness — source du badge « État »).
	// Zéro I/O DuckDB ici. 0 tant que la fraîcheur n'a jamais été calculée.
	FreshnessCritical int64 `json:"freshness_critical"`

	// LUSRInteriorGaps : trous d'intérieur LUSR au dernier scan (gauge posée par le
	// cron data_health — source du badge nav « Données » quand > 0). Zéro I/O DuckDB.
	// 0 tant que le cron n'a pas scanné. Détail sur /admin/monitoring/lusr-gaps.
	LUSRInteriorGaps int64 `json:"lusr_interior_gaps"`

	// HTTP : compteurs de requêtes par classe de statut depuis le boot (A7,
	// DC-6 — middleware SlogLogger, titre-aware, jamais par route).
	HTTP MonitoringHTTPSummary `json:"http"`
}

// MonitoringHTTPSummary — compteurs HTTP par classe de statut depuis le boot
// (expvar http_status_*_total, posés par le middleware — zéro I/O).
type MonitoringHTTPSummary struct {
	Status2xx int64 `json:"status_2xx"`
	Status3xx int64 `json:"status_3xx"`
	Status4xx int64 `json:"status_4xx"`
	Status5xx int64 `json:"status_5xx"`
}

// MonitoringSnapshotSummary expose l'état du producteur de snapshot immuable (gauges +
// cumuls expvar titrés, zéro I/O DuckDB). Posés en fin de cycle par le SnapshotCutter.
type MonitoringSnapshotSummary struct {
	// Version : numéro du snapshot courant (0 = aucun cut encore produit). GAUGE.
	Version int64 `json:"version"`
	// ReadyMatchCount : matchs inclus dans le dernier cut produit. GAUGE.
	ReadyMatchCount int64 `json:"ready_match_count"`
	// PendingTotal : matchs enrichis pas encore snapshot-ready (backlog). GAUGE.
	PendingTotal int64 `json:"pending_total"`
	// PartialTotal : matchs ready terminalement partiels (partial_reasons non vide). GAUGE.
	PartialTotal int64 `json:"partial_total"`
	// PendingOldestAgeSeconds : âge du plus vieux pending — alerte si dépasse un
	// seuil (dérivation bloquée ; la grâce bornée finit par forcer). GAUGE.
	PendingOldestAgeSeconds int64 `json:"pending_oldest_age_seconds"`
	// CutsProduced / CutFailures / CutNoop : cumuls depuis le boot (perdus au reboot).
	CutsProduced int64 `json:"cuts_produced"`
	CutFailures  int64 `json:"cut_failures"`
	CutNoop      int64 `json:"cut_noop"`
	// ReadsServed / ReadsFallback : lectures shared MatchView servies depuis le snapshot
	// vs repli live (pilote Phase 3). Cumuls depuis le boot — mesurent l'adoption réelle.
	ReadsServed   int64 `json:"reads_served"`
	ReadsFallback int64 `json:"reads_fallback"`
}

// MonitoringSchedulerSummary résume le dernier cycle auto-sync.
type MonitoringSchedulerSummary struct {
	Available       bool   `json:"available"`               // false = scheduler non câblé
	LastCycleAt     string `json:"last_cycle_at,omitempty"` // RFC3339, vide si jamais couru
	IntervalMinutes int    `json:"interval_minutes,omitempty"`
	PoolSize        int    `json:"pool_size,omitempty"`
	LastTotal       int    `json:"last_total"`
	LastSynced      int    `json:"last_synced"`
	LastSkipped     int    `json:"last_skipped"`
	LastFailed      int    `json:"last_failed"`
	LastDurationMs  int64  `json:"last_duration_ms"`
	// ZeroInsertAlerts : joueurs dont ConsecutiveZeroInserts a atteint le seuil
	// d'alerte (sync OK mais 0 insert prolongé — API stale, gamertag changé…).
	ZeroInsertAlerts int `json:"zero_insert_alerts"`
	// InFlightClaims : syncs en vol toutes sources (watcher/HTTP/scheduler),
	// vus par le SyncGate. Couvre le « live » au sens dédup cross-source.
	InFlightClaims int `json:"in_flight_claims"`
}

// MonitoringJobsSummary résume l'activité du JobStore.
type MonitoringJobsSummary struct {
	ActiveCount int              `json:"active_count"`
	Recent      []AsyncJobStatus `json:"recent"`
}

// MonitoringDataHealth est le miroir JSON du dernier
// scheduler.DataHealthCheckResult (audit UUIDs bruts / lying bits / orphelins).
type MonitoringDataHealth struct {
	RanAt                string `json:"ran_at"` // RFC3339
	UUIDsRawCount        int    `json:"uuids_raw_count"`
	LyingBitsEvents      int    `json:"lying_bits_events"`
	LyingBitsWeaponKills int    `json:"lying_bits_weapon_kills"`
	OrphanXUIDs          int    `json:"orphan_xuids"`
	GarbageBannerURLs    int    `json:"garbage_banner_urls"`
	WarningsTotal        int    `json:"warnings_total"`
	DurationMs           int64  `json:"duration_ms"`
}

// MonitoringTokensSummary agrège les statuts token par joueur (refresh token,
// le signal le plus actionnable : expiré/absent/reauth = re-capture requise).
type MonitoringTokensSummary struct {
	Players       int `json:"players"`
	OK            int `json:"ok"`
	Expiring      int `json:"expiring"`
	Expired       int `json:"expired"`
	Absent        int `json:"absent"`
	Reauth        int `json:"reauth"`
	WithAuthError int `json:"with_auth_error"` // joueurs avec LastAuthError mémorisé
}

// MonitoringInvariantsSummary expose l'état du dernier run d'invariants
// (gauges expvar invariants_* — 0 partout si jamais couru depuis le boot).
type MonitoringInvariantsSummary struct {
	RunsTotal int64 `json:"runs_total"` // 0 = jamais couru depuis le boot
	FailLast  int64 `json:"fail_last"`
	WarnLast  int64 `json:"warn_last"`
}

// ConvergenceTotalsSinceBoot : cumuls process-wide du travail RATTRAPÉ par la
// convergence (compteurs expvar — perdus au restart, comme l'historique).
type ConvergenceTotalsSinceBoot struct {
	EventsProcessed  int64 `json:"events_processed"`
	WeaponsProcessed int64 `json:"weapons_processed"`
	PSAProcessed     int64 `json:"psa_processed"`
	AliasesUpserted  int64 `json:"aliases_upserted"`
}

// AdminConvergenceReport est la réponse de GET /admin/monitoring/convergence :
// backlog d'enrichissement restant par joueur (le sync converge dessus à
// chaque cycle, cf. internal/sync/convergence.go).
type AdminConvergenceReport struct {
	TitleSlug   string `json:"title_slug"`
	GeneratedAt string `json:"generated_at"` // RFC3339
	// Horizon : borne de sélection par cycle — missing_psa/events/weapons sont
	// PLAFONNÉS à cette valeur (afficher « N+ » quand count == horizon).
	Horizon         int                        `json:"horizon"`
	Players         []PlayerConvergenceReport  `json:"players"`
	TotalsSinceBoot ConvergenceTotalsSinceBoot `json:"totals_since_boot"`
}

// PerfCallStats : agrégat de latence d'un appel/étape/phase depuis le boot
// (miroir des agrégats expvar RecordDurationMS — count/sum/avg/max).
type PerfCallStats struct {
	Title  string `json:"title,omitempty"` // MT-05 : "" pour Halo (byte-identique)
	Name   string `json:"name"`
	Count  int64  `json:"count"`
	SumMs  int64  `json:"sum_ms"`
	AvgMs  int64  `json:"avg_ms"`
	MaxMs  int64  `json:"max_ms"`
	Errors int64  `json:"errors,omitempty"`
}

// PerfAPIBuckets : erreurs API Halo par classe (process-wide).
type PerfAPIBuckets struct {
	RateLimited429 int64 `json:"rate_limited_429"`
	Auth           int64 `json:"auth"`
	Server5xx      int64 `json:"server_5xx"`
	Network        int64 `json:"network"`
	Other          int64 `json:"other"`
}

// AdminPerfStats est la réponse de GET /admin/monitoring/perf : agrégats de
// performance depuis le boot (expvar pur, zéro I/O DuckDB) — latences API
// Halo par appel, phases d'écriture persist par DB, étapes post-sync, fenêtre
// de blocage des lectures shared.
type AdminPerfStats struct {
	GeneratedAt   string          `json:"generated_at"`
	APICalls      []PerfCallStats `json:"api_calls"`
	APIBuckets    PerfAPIBuckets  `json:"api_buckets"`
	PersistPhases []PerfCallStats `json:"persist_phases"`
	PostSyncSteps []PerfCallStats `json:"postsync_steps"`
	PostSyncTotal PerfCallStats   `json:"postsync_total"`
	// BlockedWindow : fenêtre d'indisponibilité des lectures shared par swap
	// (count = swaps complets, sum = indispo cumulée depuis le boot).
	BlockedWindow PerfCallStats `json:"blocked_window"`
	// APIByPlayer : breakdown des appels API attribuables (match_history,
	// career_rank, player_csrs, playlist_csr) par joueur — repère un joueur dont
	// les tokens/réseau échouent. Trié erreurs desc. Vide pour les appels
	// match-level (non attribuables).
	APIByPlayer []PerfPlayerCallStats `json:"api_by_player"`
}

// PerfPlayerCallStats : agrégat d'un appel API Halo attribué à un joueur
// (miroir de observability.PlayerAPIStat).
type PerfPlayerCallStats struct {
	Title  string `json:"title,omitempty"` // MT-05 : "" pour Halo (byte-identique)
	Player string `json:"player"`
	Call   string `json:"call"`
	Count  int64  `json:"count"`
	AvgMs  int64  `json:"avg_ms"`
	MaxMs  int64  `json:"max_ms"`
	Errors int64  `json:"errors"`
}

// PlayerConvergenceReport est le backlog de convergence d'un joueur.
type PlayerConvergenceReport struct {
	PlayerSlug string `json:"player_slug"`
	Gamertag   string `json:"gamertag"`
	XUID       string `json:"xuid"`
	// MissingEnrichment : matchs en shared sans row player_match_enrichment
	// (non plafonné). Les 3 suivants sont plafonnés à Horizon.
	MissingEnrichment int `json:"missing_enrichment"`
	MissingPSA        int `json:"missing_psa"`
	MissingEvents     int `json:"missing_events"`
	MissingWeapons    int `json:"missing_weapons"`
	// CheckError non vide = DBs irrésolvables pour ce joueur (compteurs à 0).
	CheckError string `json:"check_error,omitempty"`
}

// AdminErrorStats — agrégat des logs WARN/ERROR depuis le boot (collecteur
// mémoire observability, panneau « erreurs récurrentes »). Zéro I/O, perdu au
// reboot (comme l'historique des cycles).
type AdminErrorStats struct {
	GeneratedAt string             `json:"generated_at"`
	Buckets     []AdminErrorBucket `json:"buckets"`
}

// AdminErrorBucket — une erreur agrégée par (niveau, message). Count = nombre
// d'occurrences ; LastDetail = dernier échantillon de l'attribut « err ».
type AdminErrorBucket struct {
	Title      string `json:"title,omitempty"` // MT-05 : "" pour Halo (byte-identique)
	Level      string `json:"level"`
	Module     string `json:"module,omitempty"`
	Message    string `json:"message"`
	Count      int64  `json:"count"`
	FirstSeen  string `json:"first_seen"`
	LastSeen   string `json:"last_seen"`
	LastDetail string `json:"last_detail,omitempty"`
}
