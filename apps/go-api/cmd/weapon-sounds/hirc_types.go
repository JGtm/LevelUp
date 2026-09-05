package main

// hirc_types.go — les TYPES publies par le mode `hirc-event` et l'etat de chemin qu'il
// accumule. Separes de `hirc_event.go` pour tenir le seuil de 500 lignes par fichier.
//
// Ces structures sont le CONTRAT entre le dump HIRC et le rendu (`rendu_event.go`) : le
// premier ecrit un plan JSON, le second le relit et n'invente rien. C'est ce qui garantit
// qu'un fichier livre porte les gains, offsets et hauteurs REELLEMENT lus dans la banque.

// v3eVariante : un `.wem` jouable par une couche, avec son chemin complet cuit en trois
// nombres — c'est exactement ce que le rendu applique.
type v3eVariante struct {
	Wem      uint32  `json:"wem"`
	Noeud    string  `json:"noeud_sound"`
	GainDB   float64 `json:"gain_db"`
	DelaiS   float64 `json:"delai_s"`
	PitchCts float64 `json:"pitch_cents"`
}

// v3eCouche : un point de choix de l'evenement. Le moteur joue EXACTEMENT UNE de ses
// variantes, et toutes les couches d'un evenement sonnent ENSEMBLE (aux offsets pres).
type v3eCouche struct {
	Cible     string `json:"cible"`
	TypeNoeud string `json:"type_noeud"`
	Chemin    string `json:"chemin"`
	// GainAmont : parents actor-mixer + bus, c'est-a-dire tout ce qui est AU-DESSUS du
	// noeud vise par l'action. C'est la part que les lots precedents ignoraient.
	GainAmont   float64       `json:"gain_amont_db"`
	BusEffectif string        `json:"bus_effectif,omitempty"`
	BusResolu   bool          `json:"bus_resolu"`
	Variantes   []v3eVariante `json:"variantes"`
	// RangedVolume / RangedPitch : la fourchette que le moteur tire A CHAQUE lecture,
	// accumulee le long du chemin. Jamais cuite dans les fichiers : le rendu prend la
	// valeur CENTRALE et la fourchette est publiee ici (regle du chantier).
	RangedVolume *[2]float32 `json:"ranged_volume_db,omitempty"`
	RangedPitch  *[2]float32 `json:"ranged_pitch_cents,omitempty"`
	Repetitions  *int        `json:"repetitions,omitempty"`
	// Sequence / Continu : le mode de lecture du conteneur (`eMode`, `bIsContinuous` —
	// `conteneurs_mode.go`). `Continu` avec `Repetitions = 0` veut dire que le moteur
	// ENCHAINE des tirages successifs sans fin : rendre une seule variante en boucle est
	// alors FAUX, et c'est exactement le piege des grains courts du Ghost (0,13 s).
	Sequence bool `json:"sequence,omitempty"`
	Continu  bool `json:"continu,omitempty"`
	// ModeTransition / TransitionS : `AkTransitionMode` entre deux lectures successives.
	ModeTransition int     `json:"mode_transition,omitempty"`
	TransitionS    float32 `json:"transition_s,omitempty"`
}

// v3eAction : une Event Action, avec son type brut et son delai propre.
type v3eAction struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	Brut   uint16  `json:"type_brut"`
	Cible  string  `json:"cible"`
	DelaiS float32 `json:"delai_s,omitempty"`
}

// v3eNoeud : un noeud de la hierarchie tel qu'il est ECRIT dans la banque.
type v3eNoeud struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Wem        uint32   `json:"wem,omitempty"`
	GainPropre float64  `json:"gain_propre_db"`
	Base       nodeBase `json:"base"`
	Enfants    []string `json:"enfants,omitempty"`
	// SwitchGroupe / SwitchEtats : la table complete d'un conteneur `Switch`, chaque etat
	// avec son enfant ET les `.wem` qu'il atteint. Sans les `.wem`, un etat n'est qu'un
	// hachage : c'est en MESURANT ses medias qu'on sait lequel est le regime de conduite
	// (methode V3C, echelle spectrale monotone).
	SwitchGroupe uint32       `json:"switch_groupe,omitempty"`
	SwitchDefaut uint32       `json:"switch_defaut,omitempty"`
	SwitchEtats  []etatSwitch `json:"switch_etats,omitempty"`
}

// etatSwitch : un etat d'un conteneur `Switch`, avec ce qu'il joue.
type etatSwitch struct {
	Etat    uint32   `json:"etat"`
	Enfants []string `json:"enfants"`
	Wems    []uint32 `json:"wems"`
}

// v3eEvent : le dump complet d'un evenement.
type v3eEvent struct {
	Bank  string `json:"bank"`
	Event string `json:"event"`
	// Etat : l'etat de conteneur `Switch` FORCE pour ce dump (0 = etat par defaut de la
	// banque). Un evenement a switch est publie une fois PAR etat : c'est la seule facon
	// de comparer les regimes sans recharger le module huit fois.
	Etat    uint32      `json:"etat,omitempty"`
	Actions []v3eAction `json:"actions"`
	Couches []v3eCouche `json:"couches"`
	Noeuds  []v3eNoeud  `json:"noeuds"`
}

// v3eRapport : la sortie du mode.
type v3eRapport struct {
	Module string     `json:"module"`
	Events []v3eEvent `json:"events"`
	// ProfilProps : releve des identifiants `AkPropID` reellement rencontres, avec leur
	// nombre d'occurrences. C'est ce releve qui tranche le desaccord de table documente
	// en tete de `hirc_noeuds.go`, sur donnees et non sur memoire.
	ProfilProps map[string]int `json:"profil_props"`
	// Bus : les objets `Bus`/`AuxBus` presents dans les banques ouvertes. Vide = la
	// hierarchie de bus vit dans une banque d'initialisation absente de ces tags.
	Bus map[string]float64 `json:"bus_du_module"`
	// Inconnues : toute propriete dont l'identifiant n'est pas dans `nomsAkProp`, avec ses
	// OCTETS BRUTS et le noeud qui la porte. Regle du chantier : un reglage non decode se
	// PUBLIE brut, il ne se tait pas.
	Inconnues []string `json:"proprietes_non_decodees,omitempty"`
}

// cheminV3E : l'etat accumule en descendant l'arbre.
type cheminV3E struct {
	Gain     float64
	Delai    float64
	Pitch    float64
	VolMin   float32
	VolMax   float32
	PitchMin float32
	PitchMax float32
	Texte    string
}

// avecNoeud ajoute au chemin ce qu'un noeud impose : gain propre, offset, hauteur,
// fourchettes RANGED. Les fourchettes s'ADDITIONNENT le long du chemin (deux noeuds qui
// tirent chacun +-3 dB donnent +-6 dB au total), conformement au moteur.
func (c cheminV3E) avecNoeud(nb nodeBase, etiquette string) cheminV3E {
	c.Gain += nb.gainPropre()
	if v, ok := nb.prop(propInitialDelay); ok {
		c.Delai += float64(v)
	}
	if v, ok := nb.prop(propPitch); ok {
		c.Pitch += float64(v)
	}
	if lo, hi, ok := nb.fourchetteDe(propVolume); ok {
		c.VolMin += lo
		c.VolMax += hi
	}
	if lo, hi, ok := nb.fourchetteDe(propPitch); ok {
		c.PitchMin += lo
		c.PitchMax += hi
	}
	if c.Texte != "" {
		c.Texte += " -> "
	}
	c.Texte += etiquette
	return c
}
