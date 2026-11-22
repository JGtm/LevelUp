package domain

// SyncNowResponse est la réponse de « Synchroniser maintenant » : le cycle est lancé en arrière-plan.
type SyncNowResponse struct {
	Status string `json:"status" enum:"started" doc:"Le cycle de synchronisation tourne en arrière-plan."`
}

// SyncCycleState dit si un cycle de synchronisation tourne en ce moment.
type SyncCycleState struct {
	Running bool `json:"running" doc:"Un cycle de synchronisation tourne (périodique ou demandé)."`
}
