package replaydoc

// reloads.go — LES RECHARGEMENTS D'ARME ET LEUR COUVERTURE, jumeaux servis de
// `film/replay/document_reloads.go`.

// Reload est UN rechargement, sur l'axe de frames du document : la frame de la trame qui écrit le message, et
// l'index de joueur de l'auteur tel que le message l'écrit (le même espace que `roster[].filmIndex`). Le film ne
// dit ni l'arme, ni la fin du rechargement : un client compte des rechargements par joueur.
type Reload struct {
	T     int `json:"t"`
	Place int `json:"place"`
}

// ReloadCoverage dit ce que le calque des rechargements a lu et ce qu'il a écarté : `read` = `copies` +
// `withoutPlayer` + `beforeOrigin` + `afterEnd` + `published`. Une trame dont la liste d'événements s'arrête sur
// un message non lu perd ceux qui la suivent (`stoppedFrames`) : le compte est alors un plancher.
type ReloadCoverage struct {
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	Read          int `json:"read"`
	Copies        int `json:"copies"`
	WithoutPlayer int `json:"withoutPlayer"`
	BeforeOrigin  int `json:"beforeOrigin"`
	AfterEnd      int `json:"afterEnd"`
	Published     int `json:"published"`
}
