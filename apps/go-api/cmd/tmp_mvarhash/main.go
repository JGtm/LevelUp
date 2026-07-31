// tmp_mvarhash — outil jetable : cherche la fonction de hachage qui relie les noms
// lisibles de root[10] aux hashs portes par le sac #8 des objets.
package main

import (
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"strings"
)

func murmur3(key []byte, seed uint32) uint32 {
	const c1, c2 = 0xcc9e2d51, 0x1b873593
	h := seed
	n := len(key) / 4
	for i := 0; i < n; i++ {
		k := uint32(key[i*4]) | uint32(key[i*4+1])<<8 | uint32(key[i*4+2])<<16 | uint32(key[i*4+3])<<24
		k *= c1
		k = k<<15 | k>>17
		k *= c2
		h ^= k
		h = h<<13 | h>>19
		h = h*5 + 0xe6546b64
	}
	var k uint32
	tail := key[n*4:]
	switch len(tail) {
	case 3:
		k ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k ^= uint32(tail[0])
		k *= c1
		k = k<<15 | k>>17
		k *= c2
		h ^= k
	}
	h ^= uint32(len(key))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func fnv1a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func fnv1(s string) uint32 {
	h := fnv.New32()
	h.Write([]byte(s))
	return h.Sum32()
}

func djb2(s string) uint32 {
	var h uint32 = 5381
	for _, c := range []byte(s) {
		h = h*33 + uint32(c)
	}
	return h
}

func sdbm(s string) uint32 {
	var h uint32
	for _, c := range []byte(s) {
		h = uint32(c) + h<<6 + h<<16 - h
	}
	return h
}

// cibles observees dans breaker_ctf_breaker.mvar
var targets = map[string]int32{
	"prefab_sockets(#8[1].0[0].0[0])": -74596443,
	"type_sockets(#8[24].0[1][0])":    -1940433248,
	"type_flagstand?(95146865)":       -1270363794,
	"label_A":                         1755892232,
	"label_B":                         2110778921,
	"label_flagstand":                 1838764749,
}

var candidates = []string{
	"forerunner_capacitor", "stockpile_blue_socket_01", "stockpile_red_socket_01",
	"Forerunner Capacitor", "stockpile_socket", "Stockpile Socket",
	"flag_stand", "Flag Stand", "Blue Flag Stand", "Red Flag Stand", "Neutral Flag Stand",
	"flag", "Flag", "ctf_flag", "stockpile", "Stockpile", "capacitor",
	"objects\\gameplay\\stockpile\\forerunner_capacitor",
}

func main() {
	fmt.Println("cibles (int32 -> uint32):")
	for name, v := range targets {
		fmt.Printf("  %-34s %12d  u=%10d  0x%08X\n", name, v, uint32(v), uint32(v))
	}
	fmt.Println("\ncandidats:")
	for _, c := range candidates {
		variants := map[string]string{
			"brut":  c,
			"lower": strings.ToLower(c),
		}
		for vn, s := range variants {
			fmt.Printf("  %-50q [%s] mm3=%10d(%11d) fnv1a=%10d(%11d) fnv1=%10d djb2=%10d sdbm=%10d crc32=%10d crcC=%10d\n",
				s, vn,
				murmur3([]byte(s), 0), int32(murmur3([]byte(s), 0)),
				fnv1a(s), int32(fnv1a(s)),
				fnv1(s), djb2(s), sdbm(s),
				crc32.ChecksumIEEE([]byte(s)),
				crc32.Checksum([]byte(s), crc32.MakeTable(crc32.Castagnoli)))
		}
	}
}
