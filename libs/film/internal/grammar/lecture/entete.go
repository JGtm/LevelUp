package lecture

// Provenance dit d'où vient la valeur d'un paramètre hors flux de la marche (ADR 0037 IR-7) :
// une valeur que la marche ne lit pas dans le paquet, mais sous laquelle elle le lit.
type Provenance uint8

// Les provenances d'un paramètre hors flux.
const (
	// ProvenanceNonRenseignee : le paramètre n'est pas résolu avant la marche — la grammaire le
	// détermine à la demande, ou il n'a pas de valeur pour ce film.
	ProvenanceNonRenseignee Provenance = iota
	// ProvenanceRelue : lue dans le film (registre `chunk_00`, manifeste).
	ProvenanceRelue
	// ProvenanceMesuree : mesurée sur le binaire du jeu et constante pour tous les films.
	ProvenanceMesuree
	// ProvenancePresumee : une valeur de référence — la table du profil, le catalogue des cartes,
	// l'image statique du binaire —, pas lue dans ce film.
	ProvenancePresumee
	// ProvenanceCalibree : calibrée sur le film, par un balayage d'essai.
	ProvenanceCalibree
	// ProvenanceImposee : imposée à la construction du contexte par l'appelant.
	ProvenanceImposee
)

// Parametre est la valeur d'un paramètre hors flux et sa provenance. Une provenance
// [ProvenanceNonRenseignee] dit que la valeur n'a pas de sens.
type Parametre[T any] struct {
	// Valeur est la valeur sous laquelle la marche lit.
	Valeur T
	// Provenance dit d'où elle vient.
	Provenance Provenance
}
