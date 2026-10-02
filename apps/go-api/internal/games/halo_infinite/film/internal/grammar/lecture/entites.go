package lecture

// VueInconnue est l'[Entite.Vue] d'une liaison dont la vue n'a pas pu être attribuée.
const VueInconnue int8 = -1

// Entite est la liaison d'UN slot dans la table d'entités de la marche : l'eid, l'archétype, la
// vue qui possède l'entité et la provenance de la liaison.
type Entite struct {
	// EID est l'eid complet posé par la liaison, génération dans les bits 30-31 ; quand la
	// génération est inconnue ([Entite.GenerationConnue] faux), le slot seul.
	EID uint32
	// TI est l'archétype.
	TI uint16
	// Vue est le rang de la vue qui possède l'entité, dans la numérotation du film ;
	// [VueInconnue] quand la liaison ne la dit pas.
	Vue int8
	// Liaison est la provenance de la liaison.
	Liaison Liaison
	// GenerationConnue dit que la génération de l'eid a été lue avec la liaison.
	GenerationConnue bool
}

// Entites est la table d'entités de la marche, en LECTURE SEULE (ADR 0037 IR-5) : UNE table,
// indexée par le slot, la vue en attribut — la table de datums du décodeur partagé, dont
// l'image-clé est le vidage. Elle n'est valide que pendant le tour d'itération qui la rend : la
// marche la fait évoluer d'un paquet à l'autre.
type Entites interface {
	// Entite rend la liaison du slot ; faux quand le slot n'est pas lié.
	Entite(slot uint32) (Entite, bool)
	// Liees rend le nombre de slots liés.
	Liees() int
}
