package types

// source.go — LES TYPES DE CONTRAT DE LA COUCHE SOURCE (ADR 0034 D-1).
//
// Ils ont ete deplaces de `film/source/film.go` au lot 2.6.2 (2026-09-16), SANS reecriture.
// `Film`, `Source` et `MemoryChunks` sont restes : ce sont des objets de service, pas des
// donnees.

// ChunkMeta : metadonnees d'un chunk, telles que le manifeste du film les porte. Ce paquet ne les
// LIT pas (il ne connait ni le cache ni le manifeste) : l'appelant les FOURNIT a [Load].
type ChunkMeta struct {
	Index     int
	ChunkType int
	StartMS   int
}

// Packet : un paquet de replication dans un chunk decompresse.
//
// Payload est une SOUS-TRANCHE du buffer du chunk, JAMAIS une copie. Deux consequences a
// connaitre : ecrire dans le payload modifie le chunk (et reciproquement), et garder un seul
// Packet retient tout le chunk decompresse. C'est le choix qui rend le decodage unique abordable
// en memoire — copier les payloads doublerait le pic.
type Packet struct {
	// Chunk : indice du chunk dans la source ; Index : rang du paquet dans ce chunk (0-based).
	Chunk, Index int
	// Type : le type du paquet (0 = delta/replication, 2 = image-cle, 7 = CHUNK_END...).
	Type int
	// TS : horodatage moteur du paquet, en microsecondes (horloge du film).
	TS uint64
	// Payload : les octets du paquet, sous-tranche du chunk decompresse.
	Payload []byte
}
