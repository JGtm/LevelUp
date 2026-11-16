package domain

import "errors"

// tactical_reapparitions.go — LA LECTURE « REAPPARITIONS ADVERSES » de l'onglet Tactique : pour une
// zone nommee choisie, ou les ADVERSAIRES reapparaissent quand l'equipe du joueur de la page TIENT cette
// zone, a cote de la repartition sans condition.
//
// # CE QUE LA LECTURE COMPTE
//
// Une reapparition adverse est la reapparition apres mort (definition du sidecar : lecture du film quand
// le document la porte, repli sur l'ecart de 4 a 15 s sinon) d'un joueur dont l'equipe n'est pas celle
// du joueur de la page, dans un match de l'univers. La repartition « de reference » les compte toutes,
// sans condition ; la repartition « zone tenue » ne garde que celles qui tombent a un instant ou au moins
// ReapparitionsAdversesJoueursPourTenir joueurs vivants de l'equipe du joueur de la page se trouvent dans
// la zone choisie. Chaque reapparition est rangee dans la zone nommee qui nomme son point
// (analysis/tactical.NommerZone), jamais dans un nom de repli.
//
// # SEUILS
//
// Une repartition n'est lue qu'a partir de ReapparitionsAdversesPlancherMatchs matchs distincts ET
// ReapparitionsAdversesPlancherReapparitions reapparitions : dessous, ses zones ne sont pas servies et
// la reponse dit pourquoi (codes TacticalRaison*). Ses effectifs, eux, sont toujours servis.

// Seuils de la lecture « reapparitions adverses ».
const (
	// ReapparitionsAdversesJoueursPourTenir : le nombre de joueurs vivants de l'equipe du joueur de la
	// page qui doivent se trouver dans la zone, a l'instant de la reapparition adverse, pour qu'elle
	// soit TENUE.
	ReapparitionsAdversesJoueursPourTenir = 2
	// ReapparitionsAdversesPlancherMatchs et ReapparitionsAdversesPlancherReapparitions : les effectifs
	// minimaux d'une repartition pour qu'elle soit servie.
	ReapparitionsAdversesPlancherMatchs        = 3
	ReapparitionsAdversesPlancherReapparitions = 20
)

// Etats de la lecture : servie, ou vide avec sa raison.
const (
	TacticalEtatServie = "servie"
	TacticalEtatVide   = "vide"
)

// Raisons d'un etat vide, en codes machine STABLES : le web ecrit lui-meme le texte (FR et EN).
const (
	// TacticalRaisonModeNonCouvert : aucun match mesure de l'univers n'est d'une famille de modes
	// couverte par la prevision (ou le titre n'en declare aucune).
	TacticalRaisonModeNonCouvert = "mode_non_couvert"
	// TacticalRaisonAucunMatchMesure : aucun match de l'univers n'a de sidecar exploitable.
	TacticalRaisonAucunMatchMesure = "aucun_match_mesure"
	// TacticalRaisonEquipeInconnue : les matchs mesures et couverts ne nomment pas l'equipe du joueur
	// de la page (il n'y a pas de piste a son nom).
	TacticalRaisonEquipeInconnue = "equipe_inconnue"
	// TacticalRaisonZoneSansForme : la zone choisie n'a aucune forme au sol : on ne peut pas dire qui
	// s'y trouve.
	TacticalRaisonZoneSansForme = "zone_sans_forme"
	// TacticalRaisonSousPlancherMatchs et TacticalRaisonSousPlancherReapparitions : la repartition
	// compte moins de matchs distincts, ou moins de reapparitions, que le plancher.
	TacticalRaisonSousPlancherMatchs        = "sous_plancher_matchs"
	TacticalRaisonSousPlancherReapparitions = "sous_plancher_reapparitions"
)

// ErrTacticalZoneInconnue : le volume demande n'est pas une zone nommee de la carte. Refus de
// l'appelant, traduit en 404 par le handler.
var ErrTacticalZoneInconnue = errors.New("tactique: zone inconnue pour cette carte")

// TacticalReapparitionsAdversesRequest est la demande de la lecture.
type TacticalReapparitionsAdversesRequest struct {
	MapID string
	// Scope : la liste blanche de matchs et la composition. Le filtre de grappe et `vies` ne
	// s'appliquent pas : la lecture porte sur toutes les reapparitions adverses de l'univers.
	Scope TacticalScope
	// VolumeIndex est l'indice de la zone choisie dans le catalogue de la carte
	// (`CalloutZone.volume_index` de la lecture de placement).
	VolumeIndex int
}

// TacticalReapparitionsAdverses est la reponse : la repartition par zone des reapparitions adverses
// quand l'equipe tient la zone choisie, et la repartition sans condition qui sert de reference.
type TacticalReapparitionsAdverses struct {
	MapID string `json:"map_id"`
	// Etat vaut TacticalEtatServie, ou TacticalEtatVide quand la repartition « zone tenue » n'est pas
	// lisible ; Raison dit alors pourquoi (TacticalRaison*). Une lecture vide reste une reponse 200.
	Etat   string `json:"etat"`
	Raison string `json:"raison,omitempty"`

	// Zone : la zone choisie, dans la langue de la requete.
	Zone TacticalZoneChoisie `json:"zone"`

	// MatchsFiltres : l'univers. MatchsRetenus : ceux qui nourrissent la lecture (sidecar exploitable,
	// famille de modes couverte, equipe du joueur nommee). MatchsHorsMode : mesures mais d'un mode non
	// couvert. MatchsSansEquipe : mesures et couverts, mais sans equipe nommee pour le joueur.
	MatchsFiltres    int `json:"matchs_filtres"`
	MatchsRetenus    int `json:"matchs_retenus"`
	MatchsHorsMode   int `json:"matchs_hors_mode"`
	MatchsSansEquipe int `json:"matchs_sans_equipe"`

	// Plancher : les seuils appliques a chaque repartition.
	Plancher TacticalPlancherReapparitions `json:"plancher"`

	// Reference : toutes les reapparitions adverses, sans condition. Tenue : celles qui tombent quand
	// l'equipe du joueur tient la zone choisie.
	Reference TacticalRepartitionZones `json:"reference"`
	Tenue     TacticalRepartitionZones `json:"tenue"`
}

// TacticalZoneChoisie est la zone demandee.
type TacticalZoneChoisie struct {
	VolumeIndex int `json:"volume_index"`
	// Nom : le libelle du lieu dans la langue de la requete (repli en) ; vide sans nom au catalogue.
	Nom string `json:"nom"`
}

// TacticalPlancherReapparitions : les seuils d'une repartition.
type TacticalPlancherReapparitions struct {
	Matchs        int `json:"matchs"`
	Reapparitions int `json:"reapparitions"`
	// JoueursPourTenir : le nombre de joueurs de l'equipe qui font « tenir » la zone.
	JoueursPourTenir int `json:"joueurs_pour_tenir"`
}

// TacticalRepartitionZones est la repartition des reapparitions adverses par zone nommee.
type TacticalRepartitionZones struct {
	// Matchs : les matchs distincts qui y contribuent. Reapparitions : leur nombre.
	Matchs        int `json:"matchs"`
	Reapparitions int `json:"reapparitions"`
	// Suffisante : les deux planchers sont atteints. Faux : Zones est vide et Raison dit lequel manque.
	Suffisante bool   `json:"suffisante"`
	Raison     string `json:"raison,omitempty"`
	// Zones : triees par nombre decroissant puis par indice de volume. SansZone compte les
	// reapparitions qu'aucune zone nommee ne nomme. La somme des deux vaut Reapparitions quand la
	// repartition est suffisante.
	Zones    []TacticalPartDeZone `json:"zones"`
	SansZone int                  `json:"sans_zone"`
}

// TacticalPartDeZone est la part d'une zone nommee dans une repartition.
type TacticalPartDeZone struct {
	VolumeIndex   int    `json:"volume_index"`
	Nom           string `json:"nom"`
	Reapparitions int    `json:"reapparitions"`
	// Part : la part de Reapparitions de la repartition, de 0 a 1.
	Part float64 `json:"part"`
}
