package domain

// Types de coordination — mesures d'equipe PARTAGEES par l'onglet Tactique et la page
// Escouade (l'echange, la couverture). Structs purs : aucune I/O, aucun SQL.

// Couverture est LE type de retour d'un taux dans ce domaine. Il n'y a pas de variante qui
// rendrait un float64 nu, et c'est une regle, pas une preference.
//
// POURQUOI. Un taux seul ment de deux facons. Il ment sur la TAILLE : « 100 % de morts
// vengees » sur huit morts n'est pas une performance, c'est un echantillon. Il ment sur le
// VOLUME : deux joueurs a 40 % ne se comparent pas si l'un joue deux fois plus. Le type
// force donc les trois grandeurs a voyager ensemble — le taux, le compte brut, et la
// quantite par match — et porte le drapeau d'echantillon faible avec elles (doctrine
// SquadAssistPairsTable, plan tactique 2026-09-05).
//
// Les tags JSON sont poses le 2026-09-06 (correction R4) : ce type traverse le contrat
// HTTP de l'onglet Tactique, et un `Brut` / `EchantillonFaible` en PascalCase a cote des
// `map_id` / `matchs_retenus` de ses voisins ferait un contrat qui parle deux langues.
type Couverture struct {
	// Taux est en unite 0..1 (ADR 0006), jamais en pourcentage : la mise en forme est un
	// choix d'affichage, pas une propriete de la mesure.
	Taux float64 `json:"taux"`

	// Brut est le numerateur : le nombre d'evenements comptes (morts vengees, ...).
	Brut int `json:"brut"`

	// ParMatch est la quantite brute ramenee au match. C'est ce qui rend deux joueurs
	// comparables quand ils n'ont pas joue le meme nombre de matchs.
	ParMatch float64 `json:"par_match"`

	// N est le denominateur : la taille de l'echantillon (morts examinees, ...).
	N int `json:"n"`

	// EchantillonFaible dit que N est sous le plancher (cf.
	// coordination.SeuilEchantillonFaible). L'affichage doit alors poser la reserve
	// `explorer.briefing.low_sample` — et ne classer personne.
	EchantillonFaible bool `json:"echantillon_faible"`
}

// KillEvent est l'entree minimale de la mesure d'echange : une mort, son tueur quand il est
// connu, et son instant. Projection de `match_kill_events_latest` (`victim_xuid`,
// `feed_killer_xuid`, `time_ms`, `match_id`) faite par l'appelant — le paquet
// `analysis/coordination` ne lit aucune base.
type KillEvent struct {
	MatchID string

	// KillerXUID est VIDE quand personne ne revendique la mort : chute, hors-limites,
	// grenade perdue, degat de l'environnement. Un tel evenement ne venge rien (decision
	// produit 2026-09-05) — seul un kill de coequipier compte.
	KillerXUID string

	VictimXUID string

	// TimeMs est l'instant de la mort sur l'horloge du match, en millisecondes.
	TimeMs int64
}

// EquipesParMatch donne le numero d'equipe de chaque joueur, PAR MATCH : matchID -> xuid ->
// equipe. La composition change d'un match a l'autre, et une table globale melangerait deux
// compositions au premier joueur ayant change de camp. Un xuid absent a une equipe INCONNUE :
// il ne peut alors etre ni coequipier ni adversaire, et aucun echange ne se conclut sur lui.
type EquipesParMatch map[string]map[string]int

// MortSuivie est le suivi d'UNE mort : a-t-elle ete vengee, par qui, en combien de temps.
type MortSuivie struct {
	MatchID     string
	VictimeXUID string

	// TueurXUID est vide quand la mort n'est revendiquee par personne.
	TueurXUID string
	TimeMs    int64

	// Vengeable dit qu'un echange etait POSSIBLE : un tueur identifie, d'une equipe connue
	// et adverse. Une mort non vengeable n'entre pas au denominateur du taux d'echange —
	// compter comme un echec une mort que personne ne pouvait venger fausserait la mesure.
	Vengeable bool

	Vengee      bool
	VengeurXUID string

	// DelaiMs est le temps ecoule entre la mort et l'echange. Nul si la mort n'est pas vengee.
	DelaiMs int64
}

// PaireEchange agrege les echanges d'un vengeur pour un venge — la matrice « qui echange
// pour qui » de la page Escouade.
type PaireEchange struct {
	// VengeurXUID a tue le tueur ; VengeXUID est le coequipier tombe.
	VengeurXUID string
	VengeXUID   string

	Nombre       int
	DelaiMoyenMs float64
}

// BilanEchanges est le resultat complet de la mesure : chaque mort suivie, les paires
// agregees, et les deux comptes qui alimentent une `Couverture` (NbVengees sur NbVengeables).
type BilanEchanges struct {
	Morts  []MortSuivie
	Paires []PaireEchange

	NbVengeables int
	NbVengees    int
}
