package replaydoc

// gestures.go — LES GESTES DU JOUEUR ET LEUR COUVERTURE, jumeaux servis de `film/replay/document_gestes.go` : le coup
// de mêlée et le lancer amorcé, un par ligne.

// Les genres de geste que `gestures[].kind` écrit.
const (
	// GestureKindMelee : le coup de mêlée part.
	GestureKindMelee = "melee"
	// GestureKindThrowStart : le lancer est amorcé ; le projectile part plus tard (`grenades[]`).
	GestureKindThrowStart = "throwStart"
)

// Gesture est UN geste, sur l'axe de frames du document : la frame de la trame qui écrit le message, l'index de
// joueur de l'auteur tel que le message l'écrit (le même espace que `roster[].filmIndex` et `reloads[].place`, un bot
// compris) et le genre. Le film ne dit ni le résultat ni la durée : un client compte des gestes par joueur.
type Gesture struct {
	T     int    `json:"t"`
	Place int    `json:"place"`
	Kind  string `json:"kind"`
}

// GestureCoverage dit ce que le calque des gestes a retenu. Les gestes ne se publient que sur un film dont les
// commandes sont enregistrées à pleine cadence : sur un autre, tous ceux que le film écrit se comptent en `gated` et
// aucun n'est publié, ce qui dit « ce film ne porte pas les gestes » et non « personne n'a rien fait ». Une trame dont
// la liste d'événements s'arrête sur un message non lu perd ceux qui la suivent (`stoppedFrames`) et un geste sans
// joueur n'est pas publié (`withoutPlayer`) : dans les deux cas le compte d'un joueur est un plancher. Les autres
// compteurs du décodeur n'ont aucun lecteur ici (`testutil.ReplayChampsNonServis`).
type GestureCoverage struct {
	Gated         int `json:"gated"`
	StoppedFrames int `json:"stoppedFrames"`
	WithoutPlayer int `json:"withoutPlayer"`
}
