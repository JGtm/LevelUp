package replaydoc

// coverage_stances.go — la couverture des ETATS DE MOUVEMENT, jumeau servi de
// `film/replay/document_stances.go`. Sortie de `coverage.go` au lot D-fix (2026-09-24) par la
// TAILLE : ce fichier-la depassait deja les 500 lignes, et le lot y ajoutait cinq compteurs.

// StanceCoverage dit ce que la marche des ETATS DE MOUVEMENT a lu et ce qu'elle a jeté — les
// dénominateurs sans lesquels « N intervalles » ne se juge pas. Une couverture partielle est un
// RESULTAT : la plupart des vies ne s'accroupissent ni ne glissent.
//
// `JumpEpisodes` / `JumpsDerived` (schéma 66) sont le dénominateur et le numérateur du SAUT,
// qui est DÉRIVÉ et non lu : montées fermées examinées, puis celles dont la hauteur intégrée
// tombe dans la fenêtre du saut du Spartan. Le rapport des deux est la sélectivité.
type StanceCoverage struct {
	Scanned               bool           `json:"scanned"`
	Absent                bool           `json:"absent,omitempty"`
	Records               int            `json:"records"`
	Desyncs               int            `json:"desyncs"`
	Reads                 int            `json:"reads"`
	Intervals             int            `json:"intervals"`
	JumpEpisodes          int            `json:"jumpEpisodes,omitempty"`
	JumpsDerived          int            `json:"jumpsDerived,omitempty"`
	ByKind                map[string]int `json:"byKind,omitempty"`
	Lives                 int            `json:"lives"`
	TracksTotal           int            `json:"tracksTotal"`
	Dropped               int            `json:"dropped,omitempty"`
	EventPacketsUnlocated int            `json:"eventPacketsUnlocated,omitempty"`
	ForgottenBindings     int            `json:"forgottenBindings,omitempty"`
	RefusedNews           int            `json:"refusedNews,omitempty"`
	RefusedNewFalseReads  int            `json:"refusedNewFalseReads,omitempty"`
	RefusedNewLostCreates int            `json:"refusedNewLostCreations,omitempty"`
	RefusedNewUndecided   int            `json:"refusedNewUndecided,omitempty"`
	MapWidths             [3]uint        `json:"mapWidths,omitempty"`
}
