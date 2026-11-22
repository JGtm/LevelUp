package replaydoc

// eliminations.go — LES ÉLIMINATIONS DE CHAQUE JOUEUR, AVEC LEUR MARQUE DE TIR À LA TÊTE ET DE FRAG PARFAIT, et la
// couverture du calque. Jumeaux servis de `film/replay/document_eliminations.go`.

// Elimination est UNE élimination d'un joueur : la frame où son compteur d'éliminations monte, le tueur (xuid
// décimal), la victime quand le film la donne, et deux marques que le jeu écrit dans ses compteurs. Une marque est
// vraie, fausse ou ABSENTE (inconnue) : un inconnu ne se lit jamais comme un faux. `perfect` ne vaut jamais faux.
type Elimination struct {
	T        int    `json:"t"`
	XUID     string `json:"xuid"`
	Victim   string `json:"victim,omitempty"`
	Headshot *bool  `json:"headshot,omitempty"`
	Perfect  *bool  `json:"perfect,omitempty"`
}

// EliminationCoverage dit ce que le calque des éliminations a posé et ce qu'il n'a pas su marquer. Invariants :
// `eliminations` = `headYes` + `headNo` + `headUnknown` = `perfectYes` + `perfectUnknown` ; `headUnknown` =
// `headNoCounter` + `headAmbiguous` + `headGuarded`.
type EliminationCoverage struct {
	Eliminations     int `json:"eliminations"`
	WithVictim       int `json:"withVictim"`
	HeadYes          int `json:"headYes"`
	HeadNo           int `json:"headNo"`
	HeadUnknown      int `json:"headUnknown"`
	HeadNoCounter    int `json:"headNoCounter"`
	HeadAmbiguous    int `json:"headAmbiguous"`
	HeadGuarded      int `json:"headGuarded"`
	HeadOrphanRises  int `json:"headOrphanRises"`
	PerfectYes       int `json:"perfectYes"`
	PerfectUnknown   int `json:"perfectUnknown"`
	PerfectAmbiguous int `json:"perfectAmbiguous"`
	PerfectOrphans   int `json:"perfectOrphans"`
}
