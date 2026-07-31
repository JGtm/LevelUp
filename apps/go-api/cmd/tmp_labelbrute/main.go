// tmp_labelbrute — outil jetable : force brute combinatoire sur les labels Forge.
// Genere des identifiants snake_case a partir d'un lexique et cherche ceux dont le
// murmur3_x86_32(seed=0) tombe sur un hash releve dans un .mvar.
//
// Garde-fou : avec N candidats et C cibles, on attend N*C/2^32 collisions fortuites.
// Toute correspondance doit donc etre jugee sur son SENS, pas sur le seul match.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func murmur3(key []byte) uint32 {
	const c1, c2 = 0xcc9e2d51, 0x1b873593
	var h uint32
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

var words = []string{
	"ctf", "flag", "stand", "return", "spawn", "include", "exclude", "zone", "res",
	"oddball", "ball", "koth", "hill", "strongholds", "stronghold", "stockpile",
	"socket", "seed", "assault", "bomb", "extraction", "vip", "slayer", "infection",
	"haven", "escalation", "land", "grab", "total", "control", "tc", "capture",
	"point", "delivery", "deliver", "objective", "obj", "mp", "forge", "breakout",
	"attackers", "defenders", "neutral", "blue", "red", "team", "respawn", "initial",
	"weapon", "vehicle", "powerup", "equipment", "skull", "base", "goal", "area",
	"navpoint", "nav", "location", "target", "core", "generator", "terminal",
	"gametype", "game", "start", "end", "safe", "volume", "trigger", "marker",
	"multi", "single", "attack", "defend", "one", "two", "three", "site", "post",
	"deposit", "collect", "score", "drop", "off", "pickup", "carrier", "home",
	"elimination", "attrition", "fiesta", "king", "of", "the", "juggernaut",
	"headhunter", "vip_zone", "hardpoint", "hill_move", "move",
}

func main() {
	targets := map[uint32]bool{}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	for _, line := range strings.Fields(string(raw)) {
		var v int64
		if _, err := fmt.Sscanf(line, "%d", &v); err == nil {
			targets[uint32(int32(v))] = true
		}
	}
	fmt.Printf("cibles=%d lexique=%d\n", len(targets), len(words))

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	count := 0
	emit := func(s string) {
		count++
		if targets[murmur3([]byte(s))] {
			fmt.Fprintf(out, "MATCH %12d  %q\n", int32(murmur3([]byte(s))), s)
		}
	}
	for _, a := range words {
		emit(a)
		for _, b := range words {
			ab := a + "_" + b
			emit(ab)
			for _, c := range words {
				abc := ab + "_" + c
				emit(abc)
				for _, d := range words {
					emit(abc + "_" + d)
				}
			}
		}
	}
	fmt.Fprintf(out, "candidats testes=%d, collisions fortuites attendues=%.2f\n",
		count, float64(count)*float64(len(targets))/4294967296.0)
}
