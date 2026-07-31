// tmp_labelhash — outil jetable : extrait les chaines ASCII des tags d'un .module,
// calcule leur murmur3_x86_32(seed=0) et les confronte a une liste de hashs cibles
// releves dans un .mvar. Objectif : resoudre les labels/types d'objets Forge SANS
// deviner un libelle.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_labelhash <module|dossier> [fichier_hashs_cibles]")
		os.Exit(2)
	}
	targets := map[uint32]bool{}
	if len(os.Args) > 2 {
		raw, err := os.ReadFile(os.Args[2])
		if err == nil {
			for _, line := range strings.Fields(string(raw)) {
				var v int64
				if _, err := fmt.Sscanf(line, "%d", &v); err == nil {
					targets[uint32(int32(v))] = true
				}
			}
		}
	}
	fmt.Printf("cibles=%d\n", len(targets))

	mods := collectModules(os.Args[1])
	fmt.Printf("modules=%d\n", len(mods))

	found := map[uint32]string{}
	seen := map[string]bool{}
	total := 0
	for _, mp := range mods {
		m, err := himodule.Open(mp)
		if err != nil {
			fmt.Printf("  [skip] %s: %v\n", filepath.Base(mp), err)
			continue
		}
		files := m.Files("")
		for _, f := range files {
			buf, err := m.Extract(f)
			if err != nil || len(buf) == 0 {
				continue
			}
			for _, s := range asciiStrings(buf) {
				if seen[s] {
					continue
				}
				seen[s] = true
				total++
				h := murmur3([]byte(s))
				if len(targets) == 0 || targets[h] {
					if _, dup := found[h]; !dup {
						found[h] = s
					}
				}
			}
		}
		fmt.Printf("  %s: %d fichiers, cumul %d chaines uniques, %d cibles resolues\n",
			filepath.Base(mp), len(files), total, len(found))
	}

	keys := make([]int, 0, len(found))
	for h := range found {
		keys = append(keys, int(int32(h)))
	}
	sort.Ints(keys)
	fmt.Println("--- resolutions ---")
	for _, k := range keys {
		fmt.Printf("%12d  %q\n", k, found[uint32(int32(k))])
	}
	fmt.Printf("total chaines uniques testees: %d\n", total)
}

func collectModules(root string) []string {
	st, err := os.Stat(root)
	if err != nil {
		return nil
	}
	if !st.IsDir() {
		return []string{root}
	}
	var out []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".module") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// asciiStrings extrait les identifiants imprimables de 3 a 96 caracteres.
func asciiStrings(buf []byte) []string {
	var out []string
	start := -1
	for i := 0; i <= len(buf); i++ {
		var c byte
		if i < len(buf) {
			c = buf[i]
		}
		ok := i < len(buf) && (c >= 0x20 && c < 0x7F)
		if ok {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			if n := i - start; n >= 3 && n <= 96 {
				run := string(buf[start:i])
				out = append(out, run)
				// Un run imprimable peut coller plusieurs identifiants (pas de
				// separateur nul) : on re-decoupe en tokens [A-Za-z0-9_].
				out = append(out, tokens(run)...)
			}
			start = -1
		}
	}
	return out
}

// tokens decoupe un run imprimable en identifiants [A-Za-z0-9_].
func tokens(s string) []string {
	var out []string
	cur := strings.Builder{}
	flush := func() {
		if cur.Len() >= 3 {
			out = append(out, cur.String())
		}
		cur.Reset()
	}
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			cur.WriteByte(c)
			continue
		}
		flush()
	}
	flush()
	return out
}
