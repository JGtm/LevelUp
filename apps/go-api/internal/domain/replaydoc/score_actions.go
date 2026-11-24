package replaydoc

// score_actions.go — LE SCORE PERSONNEL PAR ACTION, DATE SUR L'AXE DU REJEU, ET LA COUVERTURE DU CALQUE.

// ScoreAction est une reponse de score personnel que le jeu envoie a un joueur quand il agit :
// la frame, le joueur qui la recoit (xuid decimal), le nom de la reponse dans le script du mode,
// ses points (negatifs pour une penalite) et, quand son libelle nomme un joueur, le xuid de celui-ci.
type ScoreAction struct {
	T      int    `json:"t"`
	XUID   string `json:"xuid,omitempty" required:"true"`
	Name   string `json:"name"`
	Points int    `json:"points"`
	Target string `json:"target,omitempty"`
}

// ScoreActionCoverage dit ce que le calque a lu, publie et ecarte. `complete` est le verdict : la
// somme des points d'un joueur ne vaut son score personnel que s'il est vrai : le décodeur le rend faux, entre autres,
// quand le tableau des scores du film montre des points que le calque ne publie pas. Les compteurs de cette
// comparaison (joueurs comparés, en dessous, au-dessus du tableau, décalage d'arbre) n'ont aucun lecteur ici : le
// verdict suffit (`testutil.ReplayChampsNonServis`).
type ScoreActionCoverage struct {
	Complete           bool   `json:"complete"`
	Group              string `json:"group,omitempty"`
	GroupMissing       bool   `json:"groupMissing"`
	Frames             int    `json:"frames"`
	ClosedFrames       int    `json:"closedFrames"`
	StoppedFrames      int    `json:"stoppedFrames"`
	ViewAUnreadable    bool   `json:"viewAUnreadable"`
	Messages           int    `json:"messages"`
	Copies             int    `json:"copies"`
	ScoreMessages      int    `json:"scoreMessages"`
	Unrelated          int    `json:"unrelated"`
	NamedWithoutPoints int    `json:"namedWithoutPoints"`
	OutsideTree        int    `json:"outsideTree"`
	UnconfirmedRank    int    `json:"unconfirmedRank"`
	Scored             int    `json:"scored"`
	Published          int    `json:"published"`
	NoRecipient        int    `json:"noRecipient"`
	NotPresent         int    `json:"notPresent"`
	NoOrigin           int    `json:"noOrigin"`
	BeforeOrigin       int    `json:"beforeOrigin"`
	AfterEnd           int    `json:"afterEnd"`
	NoTarget           int    `json:"noTarget"`
}
