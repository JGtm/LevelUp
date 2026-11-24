package replaydoc

// eliminations.go — LES ÉLIMINATIONS DE CHAQUE JOUEUR, AVEC LEUR MARQUE DE TIR À LA TÊTE ET DE FRAG PARFAIT, et la
// couverture du calque. Jumeaux servis de `film/replay/document_eliminations.go`.

// Elimination est UNE élimination d'un joueur : la frame où son compteur d'éliminations monte, le tueur (xuid
// décimal), la victime quand le film la donne, et deux marques que le jeu écrit dans ses compteurs. Une marque est
// vraie, fausse ou ABSENTE (inconnue) : un inconnu ne se lit jamais comme un faux. `perfect` ne vaut jamais faux.
type Elimination struct {
	T        int    `json:"t"`
	XUID     string `json:"xuid,omitempty" required:"true"`
	Victim   string `json:"victim,omitempty"`
	Headshot *bool  `json:"headshot,omitempty"`
	Perfect  *bool  `json:"perfect,omitempty"`
}

// EliminationCoverage est la couverture du calque des éliminations : sa PRÉSENCE dit que le tableau des scores du film
// a été lu, et c'est la seule chose qu'un lecteur en tient (sans elle, aucune ligne du fil n'est marquée). Les compteurs
// du décodeur n'ont aucun lecteur ici (`testutil.ReplayChampsNonServis`) : le type n'en publie aucun.
type EliminationCoverage struct{}
