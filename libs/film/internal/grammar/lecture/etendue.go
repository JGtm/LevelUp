package lecture

// Etendue situe une lecture dans le film : le chunk, le paquet, le premier bit et la longueur.
// C'est l'unité de provenance (ADR 0037 IR-1) : un fait qui cite une étendue se rattache aux bits
// qui le fondent. Dans la structure, un élément ne porte que son début et sa longueur (le paquet
// porte le reste) ; l'étendue se construit quand une lecture sort de la structure.
//
// Seize octets : sa taille est gelée par `tailles_test.go`.
type Etendue struct {
	// Chunk est le numéro du chunk, celui que `FilmContext.ChunkNumbers` énumère.
	Chunk int32
	// Paquet est le rang du paquet dans son chunk.
	Paquet int32
	// Debut est le premier bit, compté depuis le début du payload du paquet. Un payload
	// d'image-clé dépasse le million de bits : les positions et les longueurs tiennent sur 32 bits,
	// jamais sur 16.
	Debut uint32
	// Bits est la longueur en bits.
	Bits uint32
}
