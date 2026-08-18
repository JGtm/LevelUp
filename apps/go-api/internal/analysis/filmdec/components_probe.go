package filmdec

// components_probe.go — LE HOOK DES SONDES, ET LE PETIT OUTILLAGE PARTAGE DES HOOKS.
//
// A QUOI SERT `SetProbeHook`. Quatre composants sont deja PORTES — leurs bits sont lus, leurs
// valeurs jetees — et personne ne sait a quoi ils servent : `managed-object-networked-splash-
// message-static` et `-dynamic` (ti=47), `high-frequency` (ti=4, 34 110 records par film
// Strongholds, role inconnu), `managed-object-property-name` (ti=13). Les lots de SONDES
// (F2/F3/F5 du plan d'exploitation) doivent pouvoir les REGARDER sans toucher a `traverse.go`,
// et sans qu'on ait a inventer un type publie pour une valeur dont on ignore le sens. Un hook
// generique est la reponse honnete : il rend le `ti` du registre, le champ, et les bits.
//
// LE `ti` VIENT DU REGISTRE, PAS D'UNE CONSTANTE. `consumeByName` recoit `typeIndex` de la
// traversee ; le hook le repasse tel quel. Aucun numero d'archetype n'est cable ici — et c'est
// necessaire, puisque le lot 0 a mesure DEUX decoupages de registre differents sur le corpus.
//
// AUCUNE VALEUR N'EST INTERPRETEE. Une sonde qui nommerait ses champs prejugerait de ce
// qu'elle cherche.

import "fmt"

// Etiquettes de registre des composants sondes.
const (
	compSplashMessageStatic   = "managed-object-networked-splash-message-static-component"
	compSplashMessageDynamic  = "managed-object-networked-splash-message-dynamic-component"
	compHighFrequency         = "high-frequency"
	compManagedObjectPropName = "managed-object-property-name-component"
)

// ProbeComponent designe le composant sonde. Enumeration STABLE, pas un index de registre
// (meme raison que `GameEngineField`).
type ProbeComponent int

// Les quatre composants sondes, et leur compte.
const (
	ProbeSplashStatic              ProbeComponent = iota // ti=47 i0 : R(24) toujours lu, plus un corps garde
	ProbeSplashDynamic                                   // ti=47 i1 : R(24)
	ProbeHighFrequency                                   // ti=4  i0 : R(8) en variante FRAME
	ProbeManagedObjectPropertyName                       // ti=13 i0 : R(32)
	ProbeComponentCount            = 4
)

// String rend l'etiquette de registre du composant sonde.
func (p ProbeComponent) String() string {
	switch p {
	case ProbeSplashStatic:
		return compSplashMessageStatic
	case ProbeSplashDynamic:
		return compSplashMessageDynamic
	case ProbeHighFrequency:
		return compHighFrequency
	case ProbeManagedObjectPropertyName:
		return compManagedObjectPropName
	}
	return fmt.Sprintf("sonde inconnue (%d)", int(p))
}

// probeHook, si non nil, recoit les valeurs des composants sondes. Global de paquet :
// l'appelant detient `LockProcessDecode`.
//
// PAS DE `present` ICI, a la difference des trois autres hooks : aucun des quatre composants
// n'a de porte de tete. Ajouter un booleen toujours vrai serait un champ qui mentirait le jour
// ou l'un d'eux en gagnerait une.
var probeHook func(ti uint32, comp ProbeComponent, values []uint64)

// SetProbeHook installe (ou retire, avec nil) la sonde generique.
func SetProbeHook(h func(ti uint32, comp ProbeComponent, values []uint64)) { probeHook = h }

func publishProbe(ti uint32, comp ProbeComponent, values ...uint64) {
	if probeHook != nil {
		probeHook(ti, comp, values)
	}
}

// bit2u convertit un bit lu en valeur publiable.
//
// Le paquet porte deja un `b2u` identique, mais il vit dans `capture_test.go` : un fichier de
// test n'est pas compile dans le binaire de production, donc l'y appeler ne compilerait pas.
// Deux copies, pas trois — et la seconde a une raison structurelle, pas un oubli.
func bit2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
