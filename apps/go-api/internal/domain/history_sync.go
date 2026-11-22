package domain

import "time"

// SyncNowResponse est la réponse de « Synchroniser maintenant » : le cycle est lancé en arrière-plan.
type SyncNowResponse struct {
	Status string `json:"status" enum:"started" doc:"Le cycle de synchronisation tourne en arrière-plan."`
}

// SyncCycleState dit si un cycle de synchronisation tourne en ce moment.
type SyncCycleState struct {
	Running bool `json:"running" doc:"Un cycle de synchronisation tourne (périodique ou demandé)."`
}

// Valeurs de TitleHistoryStatus.State.
const (
	HistoryStateComplete   = "complete"
	HistoryStateCatchingUp = "catching_up"
	HistoryStatePaused     = "paused"
	HistoryStateUnknown    = "unknown"
)

// HistorySyncStatus est l'état de l'historique d'un joueur, par titre suivi, et le calendrier de la
// synchronisation : de quoi dire à l'écran où en est le rattrapage et quand la prochaine passe a lieu.
type HistorySyncStatus struct {
	AutoSyncEnabled bool                 `json:"auto_sync_enabled" doc:"La synchronisation automatique est activée dans les réglages."`
	NextCycleAt     *time.Time           `json:"next_cycle_at,omitempty" doc:"Instant du prochain cycle périodique ; absent tant que la boucle n'a pas démarré."`
	Titles          []TitleHistoryStatus `json:"titles" doc:"Un élément par titre suivi pour ce joueur."`
}

// TitleHistoryStatus est l'état de l'historique d'un joueur sur un titre.
type TitleHistoryStatus struct {
	TitleSlug     string       `json:"title_slug"`
	TitleName     string       `json:"title_name" doc:"Nom du titre, tel que le registre le déclare."`
	State         string       `json:"state" enum:"complete,catching_up,paused,unknown" doc:"complete : fin de l'historique vue ; catching_up : rattrapage en cours ; paused : titre en pause ; unknown : lecture en échec."`
	StateLabel    string       `json:"state_label" doc:"Libellé de l'état, dans la langue de la requête."`
	MatchCount    int          `json:"match_count" doc:"Matchs du joueur enregistrés dans la base partagée du titre."`
	OldestMatchAt *time.Time   `json:"oldest_match_at,omitempty" doc:"Début du plus ancien match enregistré."`
	LastPass      *HistoryPass `json:"last_pass,omitempty" doc:"Dernière passe de synchronisation du joueur sur ce titre depuis le démarrage du serveur."`
}

// Valeurs de HistoryPass.Status.
const (
	HistoryPassOK      = "ok"
	HistoryPassPartial = "partial"
	HistoryPassFailed  = "failed"
	HistoryPassSkipped = "skipped"
)

// HistoryPass est la dernière passe de synchronisation d'un joueur sur un titre.
type HistoryPass struct {
	At           time.Time `json:"at" doc:"Fin de la passe."`
	MatchesAdded int       `json:"matches_added" doc:"Matchs ajoutés par la passe."`
	Status       string    `json:"status" enum:"ok,partial,failed,skipped" doc:"Issue de la passe."`
	StatusLabel  string    `json:"status_label" doc:"Libellé de l'issue, dans la langue de la requête."`
}
