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

// ReloadCoverage dit ce que le calque des rechargements a écarté. Une trame dont la liste d'événements s'arrête sur
// un message non lu perd ceux qui la suivent (`stoppedFrames`) et un rechargement sans joueur n'est pas publié
// (`withoutPlayer`) : dans les deux cas le compte d'un joueur est un plancher. Les autres compteurs du décodeur n'ont
// aucun lecteur ici (`testutil.ReplayChampsNonServis`).
type ReloadCoverage struct {
	StoppedFrames int `json:"stoppedFrames"`
	WithoutPlayer int `json:"withoutPlayer"`
}
