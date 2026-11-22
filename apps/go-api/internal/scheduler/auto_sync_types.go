// Package scheduler — auto_sync_types.go : les types de résultat de l'auto-sync (cycle, joueur, instantané).
package scheduler

import (
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/sync"
)

// RunOnceResult agrège les compteurs d'un cycle de sync.
type RunOnceResult struct {
	Total    int           `json:"total"`   // joueurs dans db_profiles
	Synced   int           `json:"synced"`  // sync delta réussie
	Skipped  int           `json:"skipped"` // joueur absent du pool, watcher actif, etc.
	Failed   int           `json:"failed"`  // erreur pendant RunDelta
	Duration time.Duration `json:"duration_ns"`

	// newMatchPlayers agrège les joueurs MOTEUR (Infinite/V2) ayant inséré >= 1
	// nouveau match ce cycle — source de la notification Discord de fin de cycle
	// (toggle discord_notify_sync). Non exporté : jamais sérialisé dans le snapshot
	// diag ni l'historique. Les joueurs live-only (Halo 5) / filet de boot passent
	// par recordOutcome → Snapshot(), fusionnés à l'émission (auto_sync_notify.go).
	newMatchPlayers []notify.PlayerSyncResult
	// debt : ce que le cycle laisse à rattraper (historique incomplet, arriéré de films), pour la cadence de
	// Run ; busy : le cycle n'a pas tourné, un autre tournait déjà.
	debt cycleDebt
	busy bool
}

// PlayerOutcomeDetail capture le résultat détaillé d'une tentative de sync pour
// un joueur, pour exposition via l'endpoint admin /api/v1/_diag/auto-sync/snapshot.
type PlayerOutcomeDetail struct {
	Gamertag        string    `json:"gamertag"`
	XUID            string    `json:"xuid"`
	Outcome         string    `json:"outcome"` // "ok", "skipped", "failed"
	Reason          string    `json:"reason"`  // texte libre expliquant le résultat
	AttemptedAt     time.Time `json:"attempted_at"`
	DurationMs      int64     `json:"duration_ms"`
	MatchesInserted int       `json:"matches_inserted,omitempty"`
	MatchesSkipped  int       `json:"matches_skipped,omitempty"`
	MedalsInserted  int       `json:"medals_inserted,omitempty"`
	SyncStatus      string    `json:"sync_status,omitempty"`
	ErrorCount      int       `json:"error_count,omitempty"`
	FirstError      string    `json:"first_error,omitempty"`
	// ConsecutiveZeroInserts compte les cycles successifs où la sync delta a
	// réussi sans insérer aucun match. Reset à 0 dès qu'un cycle insère ≥1
	// match. Non incrémenté sur outcome=skipped/failed (préserve la valeur
	// précédente). Ajouté suite à l'incident 2026-05-20 : 14 jours de sync à
	// inserted=0 sans alerte, cf. fix endpoint /matches xuid(NNN).
	ConsecutiveZeroInserts int `json:"consecutive_zero_inserts,omitempty"`
	// PostSync : compteurs du pipeline post-sync du dernier RunDelta réussi
	// (dashboard monitoring admin). Limite assumée : seuls les syncs passés
	// par syncPlayer (scheduler + cycle forcé) sont capturés — les paths
	// watcher et HTTP /sync/all ne remontent pas ici (couverts par sync.log).
	PostSync *domain.PostSyncResult `json:"post_sync,omitempty"`
	// PostSyncHistoryMs : durées post-sync (ms) des derniers cycles de CE joueur
	// (ancien → récent), pour la sparkline de tendance. Rempli par Snapshot
	// depuis le ring postSyncHistory.
	PostSyncHistoryMs []int64 `json:"post_sync_history_ms,omitempty"`
}

// ConsecutiveZeroInsertWarnThreshold est le seuil au-delà duquel le scheduler
// émet un slog.WarnContext "zero-insert prolongé" pour un joueur. 6 cycles à
// 15min = 1h30 de sync delta sans aucun nouveau match — pour un joueur actif
// c'est suspect (API stale, gamertag changé, format URL incorrect, etc.).
// Pour un joueur inactif c'est normal et restera juste informatif.
const ConsecutiveZeroInsertWarnThreshold = 6

// SchedulerSnapshot est exposé par l'endpoint admin pour diagnostic.
type SchedulerSnapshot struct {
	LastCycleAt     time.Time             `json:"last_cycle_at"`
	LastCycleResult *RunOnceResult        `json:"last_cycle_result,omitempty"`
	IntervalMinutes int                   `json:"interval_minutes"`
	PoolSize        int                   `json:"pool_size"`
	Players         []PlayerOutcomeDetail `json:"players"`
	// Gate : état du gate de déduplication cross-source (claims en vol + âge +
	// compteurs). Vide si watcher désactivé (NopSyncGate).
	Gate sync.GateSnapshotData `json:"gate"`
	// SinceBoot : true si LastCycleAt reflète un cycle EFFECTIF depuis ce boot ;
	// false si le snapshot est purement RÉHYDRATÉ du disque (dernier cycle d'un
	// boot précédent, cf. C1). Combiné à LastCycleAt (zéro = aucune donnée
	// connue), il permet au front de distinguer « aucun cycle depuis le boot mais
	// snapshot connu (daté) » de « aucune donnée » et « cycle en direct ».
	SinceBoot bool `json:"since_boot"`
}
