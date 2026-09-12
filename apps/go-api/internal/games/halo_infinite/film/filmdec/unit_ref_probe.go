package filmdec

// unit_ref_probe.go — SONDE DES CHAMPS DE RÉFÉRENCE D'ENTITÉ lus par les composants `unit-*`.
//
// POURQUOI. Les événements `biped_board_vehicle` / `unit_exit_vehicle` (event_list.go) sont des
// TRANSITIONS : ils disent qui monte et qui descend, jamais qui EST à bord à un instant
// quelconque. L'état d'occupation courant, lui, doit être sérialisé quelque part — le mode
// Théâtre du jeu l'affiche à n'importe quel instant, y compris après un saut dans le temps.
//
// LES CANDIDATS SONT DES CHAMPS DE RÉFÉRENCE, et le déserialiseur en lit de deux formes :
//
//	FUN_1408f0ac4  R(1) porte ; si ouverte -> FUN_1406d3140 (entier à largeur variable :
//	               [sonde R(1) si param_3==1] + R(bitLen(range)) + R(2) de queue). C'est la
//	               MÊME primitive que celle des références d'événements (cf. V3_EMBARQUEMENT),
//	               dont la queue de 2 bits est la GÉNÉRATION du handle.
//	FUN_14080d69c / FUN_141d0f344  R(1) porte + R(32) / R(32) inconditionnel : la forme
//	               « handle complet » (gen<<30 | slot).
//
// Ces lectures traversent déjà, leur valeur était JETÉE. La sonde ne change AUCUN bit lu :
// elle publie ce que le déserialiseur vient de lire, après coup. Elle sert à la mesure de
// corrélation « qui occupe ce véhicule ? » (rapport V5_ETAT_OCCUPATION) sans avoir à deviner
// à l'avance QUEL composant porte l'information : l'appelant rattache chaque lecture au
// composant qui la contient par sa position en bits (CompResult.StartBit).
//
// MÊME CONTRAT QUE LES AUTRES SONDES (SetObjectParentStateHook, SetUnitEquipmentHook) : global
// de paquet, donc UN SEUL décodage filmdec à la fois par process ; l'appelant détient
// `LockProcessDecode` et restaure la sonde précédente.

// UnitRefKind distingue les deux formes de champ de référence.
type UnitRefKind int

const (
	// UnitRefVarWidth est la forme FUN_1408f0ac4 -> FUN_1406d3140 (porte + entier à largeur
	// variable + 2 bits de queue).
	UnitRefVarWidth UnitRefKind = iota
	// UnitRefWord32 est la forme R(1) porte + R(32) (FUN_14080d69c).
	UnitRefWord32
	// UnitRefWord32Plain est la forme R(32) inconditionnelle (FUN_141d0f344).
	UnitRefWord32Plain
)

// String rend une étiquette lisible de la forme.
func (k UnitRefKind) String() string {
	switch k {
	case UnitRefVarWidth:
		return "varw"
	case UnitRefWord32:
		return "w32g"
	case UnitRefWord32Plain:
		return "w32"
	}
	return "?"
}

// UnitRefRead est UNE lecture de champ de référence, telle que le déserialiseur l'a faite.
// Les champs sont BRUTS et NON INTERPRÉTÉS : `Val` n'est PAS déclaré « slot » ici — la base
// per-domaine de FUN_1406d3140 est une donnée d'exécution, et c'est la mesure qui dit si
// `Val` tombe dans une bande de slots connue.
type UnitRefRead struct {
	// Kind est la forme lue.
	Kind UnitRefKind
	// StartBit / EndBit localisent la lecture dans le payload : c'est par StartBit que
	// l'appelant la rattache au composant qui la contient.
	StartBit, EndBit int
	// Present dit que la porte était ouverte (toujours vrai pour UnitRefWord32Plain).
	Present bool
	// Val est la valeur lue (index à largeur variable, ou le mot de 32 bits entier) ; Tail
	// les 2 bits de queue de la forme à largeur variable (0 pour les formes 32 bits).
	Val, Tail uint32
	// Probe dit que la sonde R(1) de FUN_1406d3140 a été lue (param_3 == 1).
	Probe bool
}

// unitRefHook, si non nil, reçoit CHAQUE lecture de champ de référence.
var unitRefHook func(UnitRefRead)

// SetUnitRefHook installe (ou retire, avec nil) la sonde. L'appelant détient
// `LockProcessDecode` et restaure la valeur précédente.
func SetUnitRefHook(h func(UnitRefRead)) { unitRefHook = h }

// publishUnitRef transmet la lecture à la sonde, si elle est posée.
func publishUnitRef(r UnitRefRead) {
	if unitRefHook == nil {
		return
	}
	unitRefHook(r)
}
