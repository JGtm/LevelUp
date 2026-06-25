// Package domain — admin_monitoring.go : payloads du dashboard monitoring
// admin (vue d'ensemble, convergence, data health).
//
// Tous les agrégats sont best-effort : une section indisponible est signalée
// par son champ *_error ou un pointeur nil, jamais par un 500 global (pattern
// admin_invariants). Les payloads scheduler (snapshot + historique) ne sont
// pas dupliqués ici : ils réutilisent les types JSON-contractés du package
// scheduler (déjà exposés par /_diag/auto-sync/snapshot).
package domain

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
