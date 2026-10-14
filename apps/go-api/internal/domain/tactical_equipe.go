package domain

// tactical_equipe.go — CE QUE LE DETAIL D'UNE ZONE DIT DES COEQUIPIERS D'UNE MORT : qui etait la,
// ou, a quelle distance du mort et du tueur ; si un coequipier a abattu le tueur dans la foulee ; et
// le bilan de la zone entiere. Plus la SEQUENCE d'une mort (vue D), lue sur les trajectoires du rejeu.
//
// AUCUN CHAMP « NON MESURE ». Une donnee que la source ne porte pas est ABSENTE (pointeur nil,
// champ omis), jamais un texte ni un zero de repli. Les comptes sont des « N sur M », jamais un
// taux nu : le denominateur voyage avec le numerateur.
//
// LES REGLES DE PORTEE SONT ICI, CALCULEES COTE GO (`analysis/coordination.APortee`) : le web lit
// `statut` et `dans_le_rayon`, il ne compare jamais une distance a un rayon.

// Etats d'un coequipier a l'instant d'une mort (`match_death_mates.state`).
const (
	TacticalMateVisible   = "visible"
	TacticalMateEnAttente = "waiting"
	TacticalMateHorsDeVue = "out_of_sight"
)

// Statuts d'une MORT, par le coequipier visible le plus proche et le rayon du match.
const (
	// TacticalStatutAccompagnee : au moins un coequipier visible a la portee du radar.
	TacticalStatutAccompagnee = "accompagnee"
	// TacticalStatutVisibleHorsRayon : un coequipier visible, mais au-dela de la portee.
	TacticalStatutVisibleHorsRayon = "visible_hors_rayon"
	// TacticalStatutAucunVisible : aucun coequipier visible, au moins un en mesure d'accompagner
	// (vivant hors de vue).
	TacticalStatutAucunVisible = "aucun_visible"
	// TacticalStatutEquipeATerre : personne ne pouvait accompagner (aucun visible ni hors de vue).
	TacticalStatutEquipeATerre = "equipe_a_terre"
)

// Types d'une riposte.
const (
	// TacticalRiposteTueurAbattu : un coequipier de la victime a abattu son tueur.
	TacticalRiposteTueurAbattu = "tueur_abattu"
	// TacticalRiposteAbattuEnRetour : l'auteur d'un frag a ete abattu dans la foulee.
	TacticalRiposteAbattuEnRetour = "abattu_en_retour"
)

// TacticalCoequipier est un coequipier de la victime a l'instant de sa mort.
type TacticalCoequipier struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag,omitempty"`
	// Statut : TacticalMate* (visible | waiting | out_of_sight).
	Statut string `json:"statut"`
	// DansLeRayon : visible ET a la portee du radar du match (borne inclusive). Faux pour tout autre
	// statut, et quand le match n'a pas de rayon.
	DansLeRayon bool `json:"dans_le_rayon"`
	// Point : sa position, pour le seul statut visible.
	Point *TacticalPoint `json:"point,omitempty"`
	// DistanceMortM : distance horizontale a la victime ; DistanceTueurM : au tueur, quand la
	// position du tueur est mesuree. Deux decimales, absentes hors du statut visible.
	DistanceMortM  *float64 `json:"distance_mort_m,omitempty"`
	DistanceTueurM *float64 `json:"distance_tueur_m,omitempty"`
}

// TacticalEquipeALaMort est l'equipe de la victime a l'instant de sa mort.
type TacticalEquipeALaMort struct {
	// Statut : TacticalStatut*. Absent quand le match n'a pas de portee de radar connue.
	Statut string `json:"statut,omitempty"`
	// RayonM : la portee du radar du match, en metres. Absente quand elle n'est pas connue.
	RayonM *float64 `json:"rayon_m,omitempty"`
	// Coequipiers : tous les coequipiers de la victime, tries par xuid.
	Coequipiers []TacticalCoequipier `json:"coequipiers"`
}

// TacticalRiposte est la riposte qui a suivi un evenement, dans la fenetre de
// `coordination.FenetreRiposteMs`.
type TacticalRiposte struct {
	// Type : TacticalRiposte*.
	Type string `json:"type"`
	// DelaiMs : ecart avec l'evenement, dans [0, fenetre].
	DelaiMs int64 `json:"delai_ms"`
	// ParGamertag : qui a tire (le coequipier vengeur, ou le tueur du joueur abattu en retour) ;
	// ParXUID est vide pour un bot.
	ParGamertag string `json:"par_gamertag,omitempty"`
	ParXUID     string `json:"par_xuid,omitempty"`
	// Point : la position de cet auteur au coup, quand la source la mesure.
	Point *TacticalPoint `json:"point,omitempty"`
}

// TacticalNSurM est un compte accompagne de son denominateur.
type TacticalNSurM struct {
	N int `json:"n"`
	M int `json:"m"`
}

// TacticalBilanMorts est le bilan des MORTS d'une zone.
type TacticalBilanMorts struct {
	// Total : les morts de la zone ; Examinees : celles dont l'equipe a ete lue (somme des quatre
	// statuts). L'ecart est le compte des morts d'un match non encore relu, pas une categorie.
	Total     int `json:"total"`
	Examinees int `json:"examinees"`
	// Les quatre statuts (TacticalStatut*).
	Accompagnees      int `json:"accompagnees"`
	VisiblesHorsRayon int `json:"visibles_hors_rayon"`
	AucunVisible      int `json:"aucun_visible"`
	EquipeATerre      int `json:"equipe_a_terre"`
	// Rose : le tueur vu depuis la victime, huit secteurs de 45 degres, le premier centre sur le
	// haut du plan (+Y), dans le sens horaire (`coordination.Rose8`).
	Rose []int `json:"rose"`
	// MedianeTueurM : distance mediane victime -> tueur ; MedianePlusProcheM : distance mediane au
	// coequipier visible le plus proche. Absentes sans mesure.
	MedianeTueurM      *float64 `json:"mediane_tueur_m,omitempty"`
	MedianePlusProcheM *float64 `json:"mediane_plus_proche_m,omitempty"`
	// TueurAbattu : N morts dont le tueur a ete abattu par un coequipier, sur M morts au tueur
	// identifie au journal.
	TueurAbattu TacticalNSurM `json:"tueur_abattu"`
}

// TacticalBilanFrags est le bilan des FRAGS d'une zone.
type TacticalBilanFrags struct {
	// AbattuEnRetour : N frags dont l'auteur a ete abattu dans la foulee, sur M frags lus au journal.
	AbattuEnRetour TacticalNSurM `json:"abattu_en_retour"`
}

// TacticalBilanZone est le bilan de la zone selectionnee : `Morts` pour les lectures qui comptent
// des morts (morts, isole, solde), `Frags` pour celles qui comptent des frags (kills, solde).
type TacticalBilanZone struct {
	Morts *TacticalBilanMorts `json:"morts,omitempty"`
	Frags *TacticalBilanFrags `json:"frags,omitempty"`
}

// MateDeMort est une ligne de `match_death_mates_latest` : un coequipier de la victime a l'instant
// d'une mort du journal.
type MateDeMort struct {
	MatchID    string
	VictimXUID string
	TimeMs     int64
	MateXUID   string
	// Gamertag : le nom du coequipier au registre du match ; vide quand il n'y figure pas.
	Gamertag  string
	Etat      string
	X, Y      *float64
	DistanceM *float64
}

// CleDeMort designe une mort du journal.
type CleDeMort struct {
	MatchID    string
	VictimXUID string
	TimeMs     int64
}

// KillDuJournal est un kill du journal publiable d'un match (`match_kill_events_latest`), avec la
// position du tueur au coup quand `kill_positions_latest` la porte. Les identites vides sont des
// bots (un bot n'a pas de xuid) : le gamertag reste renseigne.
type KillDuJournal struct {
	MatchID        string
	TimeMs         int64
	VictimXUID     string
	VictimGamertag string
	KillerXUID     string
	KillerGamertag string
	// KillerX, KillerY : nil quand aucune position n'est mesuree pour ce coup.
	KillerX, KillerY *float64
}

// ─── La sequence d'une mort (vue D) ──────────────────────────────────────────────────────────

// Roles des acteurs d'une sequence.
const (
	TacticalRoleVictime    = "victime"
	TacticalRoleTueur      = "tueur"
	TacticalRoleCoequipier = "coequipier"
)

// TacticalSequenceRequest demande la sequence d'UNE mort.
type TacticalSequenceRequest struct {
	MapID      string
	MatchID    string
	VictimXUID string
	// TimeMs : l'instant de la mort sur l'horloge du MATCH (comme TacticalContribution.InstantMs).
	TimeMs int64
}

// TacticalSequencePoint est une position echantillonnee, a `OffsetMs` de la mort. Coupure : le film
// n'a rien replique juste avant ce point — la trace ne se relie pas au point precedent.
type TacticalSequencePoint struct {
	OffsetMs int     `json:"offset_ms"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Coupure  bool    `json:"coupure,omitempty"`
}

// TacticalSequenceActeur est un acteur de la sequence : sa trace au pas d'echantillonnage de la
// reponse. L'identite vient de la BASE (journal, coequipiers de la mort), jamais de l'equipe du film.
type TacticalSequenceActeur struct {
	Role     string                  `json:"role"`
	XUID     string                  `json:"xuid,omitempty"`
	Gamertag string                  `json:"gamertag,omitempty"`
	Points   []TacticalSequencePoint `json:"points"`
}

// TacticalSequenceEtape est un instant de la sequence : le cercle de portee et ce qu'il contient.
type TacticalSequenceEtape struct {
	// OffsetMs : de -4000 a +3000 ms autour de la mort, au pas d'une seconde.
	OffsetMs int `json:"offset_ms"`
	// InstantAfficheMs : l'horloge AFFICHEE du rejeu a cet instant (ms depuis le coup d'envoi,
	// plancher 0). Absent quand le coup d'envoi du match est inconnu.
	InstantAfficheMs *int64 `json:"instant_affiche_ms,omitempty"`
	// Centre du cercle de portee : la victime avant sa mort, le point de mort ensuite.
	Centre TacticalPoint `json:"centre"`
	// DansLeRayon : coequipiers positionnes a la portee du centre ; Positionnes : coequipiers dont
	// la position est connue a cet instant (le denominateur).
	DansLeRayon int `json:"dans_le_rayon"`
	Positionnes int `json:"positionnes"`
}

// TacticalSequence est la sequence d'une mort.
type TacticalSequence struct {
	RayonM float64 `json:"rayon_m"`
	// TraineMs : la duree de la traine tracee derriere chaque acteur (4 s).
	TraineMs int `json:"traine_ms"`
	// PasMs : le pas d'echantillonnage des traces.
	PasMs   int                      `json:"pas_ms"`
	Mort    TacticalPoint            `json:"mort"`
	Acteurs []TacticalSequenceActeur `json:"acteurs"`
	Etapes  []TacticalSequenceEtape  `json:"etapes"`
}

// ─── Les pistes lues dans l'artefact de rejeu (entree de la sequence) ─────────────────────────

// TacticalPistePoint est un point d'une piste de l'artefact : image, position monde, lacune qui le
// precede en ms (`g`, absent = continu). Les etiquettes JSON sont celles de l'artefact : ce type est
// decode DIRECTEMENT depuis le fichier, il n'est jamais servi.
type TacticalPistePoint struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	G int     `json:"g,omitempty"`
}

// TacticalPiste est une vie de l'artefact : son porteur nomme (xuid, ou nom de bot) et ses points.
// L'equipe du film n'est PAS lue : l'identite des acteurs vient de la base.
type TacticalPiste struct {
	XUID   string               `json:"xuid,omitempty"`
	Bot    string               `json:"bot,omitempty"`
	Points []TacticalPistePoint `json:"points"`
}

// TacticalPistes est la part d'un artefact que lit la sequence : l'echelle de temps, l'origine du
// film sur l'axe du match et les pistes. OriginMs absent = l'horloge n'est pas etablie.
type TacticalPistes struct {
	FrameIntervalMS int             `json:"frameIntervalMs"`
	OriginMs        *int64          `json:"originMs"`
	T0FilmMs        *int64          `json:"t0FilmMs"`
	Tracks          []TacticalPiste `json:"tracks"`
}
