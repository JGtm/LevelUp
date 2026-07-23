// Package domain — admin_logs.go : payloads du viewer de logs du dashboard
// monitoring admin (modules disponibles + tail filtré).
package domain

// AdminLogModule décrit un fichier logs/{module}.log.
type AdminLogModule struct {
	Module     string `json:"module"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at"` // RFC3339
}

// AdminLogModules est la réponse de GET /admin/monitoring/logs/modules.
type AdminLogModules struct {
	GeneratedAt string           `json:"generated_at"`
	Modules     []AdminLogModule `json:"modules"`
}

// AdminLogEntry est une ligne de log parsée (best-effort : une ligne non-JSON
// arrive avec raw rempli et level "unknown").
type AdminLogEntry struct {
	Time      string         `json:"time,omitempty"` // RFC3339Nano
	Level     string         `json:"level"`
	Msg       string         `json:"msg,omitempty"`
	Module    string         `json:"module,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	EventID   string         `json:"event_id,omitempty"`
	Err       string         `json:"err,omitempty"`
	Source    string         `json:"source,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
	Raw       string         `json:"raw,omitempty"`
}

// AdminLogTail est la réponse de GET /admin/monitoring/logs/tail.
type AdminLogTail struct {
	Module      string `json:"module"`
	GeneratedAt string `json:"generated_at"`
	// Entries : du plus récent au plus ancien.
	Entries      []AdminLogEntry `json:"entries"`
	ScannedBytes int64           `json:"scanned_bytes"`
	// Truncated : budget de scan épuisé — des lignes plus anciennes
	// correspondant aux filtres existent peut-être.
	Truncated bool `json:"truncated"`
	// NextOffset : curseur « charger plus » (offset octet du début de la ligne
	// la plus ancienne renvoyée) — à repasser en ?before= pour la tranche
	// suivante, plus ancienne.
	NextOffset int64 `json:"next_offset,omitempty"`
	// HasMore : des lignes plus anciennes restent à charger au-delà de
	// NextOffset (curseur arrière disponible).
	HasMore bool `json:"has_more"`
}
