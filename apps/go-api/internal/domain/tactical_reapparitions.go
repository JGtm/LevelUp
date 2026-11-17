package domain

import "errors"

// tactical_reapparitions.go — LA LECTURE « REAPPARITIONS ADVERSES » de l'onglet Tactique : pour une
// zone nommee choisie, ou les ADVERSAIRES reapparaissent quand l'equipe du joueur de la page TIENT cette
// zone, a cote de la repartition sans condition.
//
// # CE QUE LA LECTURE COMPTE
//
// Une reapparition adverse est la reapparition apres mort (definition du sidecar : lecture du film ; un
// match dont le document ne la porte pas n'en a aucune) d'un joueur dont l'equipe n'est pas celle
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
	// TacticalRaisonAucunMatchMesure : aucun match de l'univers n'a de sidecar exploitable.
	TacticalRaisonAucunMatchMesure = "aucun_match_mesure"
	// TacticalRaisonEquipeInconnue : les matchs mesures ne nomment pas l'equipe du joueur
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
	// Etat vaut TacticalEtatServie, ou TacticalEtatVide quand la repartition « zone tenue » n'est pas
	// lisible ; Raison dit alors pourquoi (TacticalRaison*). Une lecture vide reste une reponse 200.
	Etat   string `json:"etat"`
	Raison string `json:"raison,omitempty"`

	// Zone : la zone choisie, dans la langue de la requete.
	Zone TacticalZoneChoisie `json:"zone"`

	// Plancher : les seuils appliques a chaque repartition.
	Plancher TacticalPlancherReapparitions `json:"plancher"`

	// Reference : toutes les reapparitions adverses, sans condition. Tenue : celles qui tombent quand
	// l'equipe du joueur tient la zone choisie.
	Reference TacticalRepartitionZones `json:"reference"`
	Tenue     TacticalRepartitionZones `json:"tenue"`
}

// TacticalZoneChoisie est la zone demandee.
type TacticalZoneChoisie struct {
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
	// Suffisante : les deux planchers sont atteints. Faux : Zones est vide (RaisonSousPlancher dit lequel
	// manque).
	Suffisante bool `json:"suffisante"`
	// Zones : triees par nombre decroissant puis par indice de volume. Les reapparitions qu'aucune
	// zone nommee ne nomme n'y figurent pas (HorsZone).
	Zones []TacticalPartDeZone `json:"zones"`
}

// RaisonSousPlancher rend le code du plancher qui manque a la repartition, ou "" quand les deux sont
// atteints. Les matchs passent avant les reapparitions : c'est leur manque qui rend une repartition
// trompeuse. Les effectifs servis suffisent a le dire, la repartition ne porte pas de raison.
func (r TacticalRepartitionZones) RaisonSousPlancher() string {
	switch {
	case r.Matchs < ReapparitionsAdversesPlancherMatchs:
		return TacticalRaisonSousPlancherMatchs
	case r.Reapparitions < ReapparitionsAdversesPlancherReapparitions:
		return TacticalRaisonSousPlancherReapparitions
	}
	return ""
}

// HorsZone rend le nombre de reapparitions qu'aucune zone nommee ne nomme : Reapparitions moins la
// somme des zones servies. Zero quand la repartition n'est pas suffisante (ses zones ne sont pas servies).
func (r TacticalRepartitionZones) HorsZone() int {
	if !r.Suffisante {
		return 0
	}
	n := r.Reapparitions
	for _, z := range r.Zones {
		n -= z.Reapparitions
	}
	return n
}

// TacticalPartDeZone est la part d'une zone nommee dans une repartition.
type TacticalPartDeZone struct {
	VolumeIndex   int    `json:"volume_index"`
	Nom           string `json:"nom"`
	Reapparitions int    `json:"reapparitions"`
	// Part : la part de Reapparitions de la repartition, de 0 a 1.
	Part float64 `json:"part"`
}
