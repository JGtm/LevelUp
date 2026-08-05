// Package mapvar — hash.go : fonction de hachage des labels Forge.
//
// Identifiée par correspondance directe et vérifiable :
//
//	LabelHash("stockpile_socket") == 2110778921
//
// valeur lue telle quelle dans ctf_breaker.mvar sur les 10 objets que la table de
// noms du même fichier appelle "stockpile_blue_socket_01".."stockpile_red_socket_05".
// Il s'agit de murmur3 x86_32 avec seed 0 sur les octets ASCII du nom.
package mapvar

import "encoding/binary"

const (
	murmurC1 = 0xcc9e2d51
	murmurC2 = 0x1b873593
)

// LabelHash calcule le hash d'un nom de label (murmur3 x86_32, seed 0).
// Exporté pour que les tests puissent vérifier la table de résolution, et pour
// permettre de résoudre de nouveaux labels sans réimplémenter l'algorithme.
func LabelHash(name string) int32 {
	key := []byte(name)
	var h uint32
	nblocks := len(key) / 4
	for i := 0; i < nblocks; i++ {
		k := binary.LittleEndian.Uint32(key[i*4:])
		k *= murmurC1
		k = k<<15 | k>>17
		k *= murmurC2
		h ^= k
		h = h<<13 | h>>19
		h = h*5 + 0xe6546b64
	}
	var k uint32
	tail := key[nblocks*4:]
	switch len(tail) {
	case 3:
		k ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k ^= uint32(tail[0])
		k *= murmurC1
		k = k<<15 | k>>17
		k *= murmurC2
		h ^= k
	}
	h ^= uint32(len(key))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return int32(h)
}
