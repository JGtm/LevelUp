package domain

// AdminActionJournalEntry — dernière exécution connue d'une action globale du
// dashboard admin (C2). Persistée hors DuckDB (JSON), survit au reboot.
type AdminActionJournalEntry struct {
	// Action : nom canonique (registry_names_backfill, catalog_refresh,
	// lying_bits_reset, catalog_ugc_drain, data_health, sync_cycle).
	Action string `json:"action"`
	// LastRunAt : RFC3339, vide si l'action n'a jamais tourné.
	LastRunAt string `json:"last_run_at"`
	// Outcome : "ok" | "error".
	Outcome string `json:"outcome"`
	// Trigger : "manual" (bouton admin) | "tick" (cycle périodique du scheduler).
	Trigger string `json:"trigger"`
}

// AdminActionJournalResponse — réponse de GET /admin/actions/journal : dernière
// exécution de chaque action globale (pour l'affichage « Dernière exécution : … »
// sous chaque bouton).
type AdminActionJournalResponse struct {
	GeneratedAt string                    `json:"generated_at"`
	Actions     []AdminActionJournalEntry `json:"actions"`
}
