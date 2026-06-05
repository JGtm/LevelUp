// Command rdata_weapon_scan performs a READ-ONLY static scan of HaloInfinite.exe
// looking for the 8-byte weapon IDs (wids) extracted from match films.
//
// Purpose: the film-weapon-extraction research (inv #1-125) closed concluding the
// unknown skin wids are "impossible from binary" — but that verdict only tested the
// FILM binary + game module files (mohd), never the EXE's static data sections.
// This tool tests that last unexplored surface: are the weapon wids present as static
// constants in the executable, and if so, in which PE section and with what structure?
//
// It NEVER launches the game, NEVER writes to the exe, NEVER injects anything.
// It opens the file read-only, reads section bytes, and pattern-matches known wids.
//
// Usage:
//
//	go run ./cmd/rdata_weapon_scan [path-to-HaloInfinite.exe]
package main

import (
	"bytes"
	"debug/pe"
	"encoding/hex"
	"fmt"
	"os"
)

type wid struct {
	name string
	hex  string // 16 hex chars (8 bytes) for full wids, 8 hex chars (4 bytes) for Group-B prefixes
}

// Group A — 33 confirmed weapon wids (ground truth). FINDINGS_weapon_extraction_EN.md §4.
var groupA = []wid{
	{"MA40 AR", "48c19d2d42c9679f"}, {"BR75", "2b1824d542c9679f"},
	{"VK78 Commando", "fd98554c42c9679f"}, {"Bandit Evo", "6acdc44d42c9679f"},
	{"M392 Bandit", "2fb21c8742c9679f"}, {"CQS48 Bulldog", "b619d84a42c9679f"},
	{"Mk51 Sidekick", "f408190f42c9679f"}, {"Plasma Pistol", "c354294642c9679f"},
	{"Pulse Carbine", "30484ea642c9679f"}, {"Mangler", "80977ba542c9679f"},
	{"Shock Rifle", "9387a8b942c9679f"}, {"Disruptor", "84bd29ed42c9679f"},
	{"Ravager", "c30d87c742c9679f"}, {"Needler", "b533957e42c9679f"},
	{"Heatwave", "2ac9c2ff42c9679f"}, {"Stalker Rifle", "daf193c742c9679f"},
	{"S7 Sniper", "0a1992bc42c9679f"}, {"Cindershot", "230447b142c9679f"},
	{"Skewer", "0d20c46942c9679f"}, {"M41 SPNKr", "71ab0a2c42c9679f"},
	{"Fuel Rod SPNKr", "9d6aaed242c9679f"}, {"MLRS-2 Hydra", "767db96d42c9679f"},
	{"Sentinel Beam", "c24e549e42c9679f"}, {"Gravity Hammer", "8afc085542c9679f"},
	{"Energy Sword", "1488d0bb42c9679f"}, {"Mutilator", "d791556542c9679f"},
	{"Vestige Carbine", "3e07021742c9679f"}, {"MA5K Avenger", "f5c335dfe7232c0b"},
	{"M9 Frag Grenade", "b6dbead842c9679f"}, {"Plasma Grenade", "c1e1bab042c9679f"},
	{"Spike Grenade", "6683257c42c9679f"}, {"Dynamo Grenade (hand)", "3ad55da442c9679f"},
	{"Dynamo Grenade (proj)", "18e1fea042c9679f"},
}

// Formula-A-exclusive wids — virtual slot identifiers (never in kill events). §4.
var formulaA = []wid{
	{"slot Primary baseline", "6d32c7dc42c9679f"}, {"slot Primary variant", "f55c4bd242c9679f"},
	{"slot Secondary", "0131ea1042c9679f"}, {"slot Grenade secondary", "d48d9b8442c9679f"},
	{"slot Spike skin", "67fed82c42c9679f"}, {"slot Pair wid", "0af3952e42c9679f"},
}

// Group B — UNKNOWN skin variants, only 4-byte prefixes documented. §4.
// These are the targets: if the binary resolves them, the research re-opens.
var groupB = []wid{
	{"SPIKE 91eb16de", "91eb16de"}, {"SPIKE 60f1d512", "60f1d512"}, {"SPIKE 87fab1d4", "87fab1d4"},
	{"DYNAMO 6a672afc", "6a672afc"}, {"DYNAMO b5e3278e", "b5e3278e"},
	{"GREN16 92f99df4", "92f99df4"}, {"GREN15 f9514800", "f9514800"}, {"GREN15 edff0e96", "edff0e96"},
	{"PRIMARY 82a3f54a", "82a3f54a"},
	{"SECONDARY d0b802c4", "d0b802c4"}, {"SECONDARY 5ded6cf2", "5ded6cf2"},
	{"SECONDARY 510f248a", "510f248a"}, {"SECONDARY 6c587a12", "6c587a12"},
}

type section struct {
	name     string
	fileOff  uint32
	fileSize uint32
	va       uint32
}

func sectionOf(secs []section, off int) string {
	for _, s := range secs {
		if off >= int(s.fileOff) && off < int(s.fileOff+s.fileSize) {
			return s.name
		}
	}
	return "?"
}

func reversed(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[len(b)-1-i] = b[i]
	}
	return out
}

func findAll(data, pat []byte, max int) []int {
	var out []int
	start := 0
	for {
		i := bytes.Index(data[start:], pat)
		if i < 0 {
			break
		}
		out = append(out, start+i)
		if len(out) >= max {
			break
		}
		start += i + 1
	}
	return out
}

// hexCtx returns a hex window of `n` bytes starting `before` bytes ahead of off.
func hexCtx(data []byte, off, before, n int) string {
	s := off - before
	if s < 0 {
		s = 0
	}
	e := s + n
	if e > len(data) {
		e = len(data)
	}
	return hex.EncodeToString(data[s:e])
}

// scanGroup tries each wid as-is and byte-reversed; for full 8B wids it also falls
// back to the 4-byte unique prefix. Returns count found.
func scanGroup(data []byte, secs []section, items []wid, withCtx bool) int {
	found := 0
	for _, w := range items {
		raw, err := hex.DecodeString(w.hex)
		if err != nil {
			fmt.Printf("  [bad hex] %s\n", w.name)
			continue
		}
		type try struct {
			label string
			pat   []byte
		}
		tries := []try{{"as-is", raw}, {"reversed(u64 LE)", reversed(raw)}}
		if len(raw) == 8 {
			// halfswap = two independent little-endian uint32s: rev(b[0:4])+rev(b[4:8]).
			// This is the most likely storage form given the mohd per-4-byte byte-swap.
			halfswap := append(reversed(raw[:4]), reversed(raw[4:])...)
			tries = append(tries,
				try{"halfswap(2x u32 LE)", halfswap},
				try{"prefix4 as-is", raw[:4]},
				try{"prefix4 rev", reversed(raw[:4])})
		}
		hit := false
		for _, t := range tries {
			offs := findAll(data, t.pat, 6)
			if len(offs) == 0 {
				continue
			}
			hit = true
			sec := sectionOf(secs, offs[0])
			fmt.Printf("  FOUND  %-24s [%s] %dx  sect=%-8s off=0x%X\n",
				w.name, t.label, len(offs), sec, offs[0])
			if withCtx {
				fmt.Printf("         ctx: %s\n", hexCtx(data, offs[0], 16, 48))
			}
			break // first matching representation wins; avoid duplicate noise
		}
		if hit {
			found++
		} else {
			fmt.Printf("  ----   %-24s not found\n", w.name)
		}
	}
	return found
}

func main() {
	path := `D:\SteamLibrary\steamapps\common\Halo Infinite\HaloInfinite.exe`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read exe: %v\n", err)
		os.Exit(1)
	}

	pf, err := pe.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse PE: %v\n", err)
		os.Exit(1)
	}
	defer pf.Close()

	var secs []section
	for _, s := range pf.Sections {
		secs = append(secs, section{s.Name, s.Offset, s.Size, s.VirtualAddress})
	}

	fmt.Printf("=== %s (%d bytes) ===\n\n", path, len(data))
	fmt.Println("PE sections:")
	for _, s := range secs {
		fmt.Printf("  %-10s fileOff=0x%-8X fileSize=0x%-8X va=0x%X\n",
			s.name, s.fileOff, s.fileSize, s.va)
	}

	// Positive control: prove the byte-scanner finds patterns guaranteed present in any
	// Windows PE. If these miss, the scanner is broken and the wid "not found" is invalid.
	fmt.Println("\n=== Positive control (must all be FOUND) ===")
	for _, ctrl := range []string{"GetProcAddress", "kernel32", "USER32", "RSDS"} {
		offs := findAll(data, []byte(ctrl), 1)
		if len(offs) > 0 {
			fmt.Printf("  ok   %-16q sect=%-8s off=0x%X\n", ctrl, sectionOf(secs, offs[0]), offs[0])
		} else {
			fmt.Printf("  MISS %-16q  <-- scanner suspect!\n", ctrl)
		}
	}

	fmt.Println("\n=== Group A (33 confirmed wids — ground truth) ===")
	fa := scanGroup(data, secs, groupA, true)

	fmt.Println("\n=== Formula-A-exclusive (slot identifiers) ===")
	ff := scanGroup(data, secs, formulaA, true)

	fmt.Println("\n=== Group B (UNKNOWN skin prefixes — the targets) ===")
	fb := scanGroup(data, secs, groupB, true)

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Group A found:   %d / %d\n", fa, len(groupA))
	fmt.Printf("Formula-A found: %d / %d\n", ff, len(formulaA))
	fmt.Printf("Group B found:   %d / %d (4-byte prefixes — collisions possible)\n", fb, len(groupB))
	if fa == 0 {
		fmt.Println("\nVERDICT: no known weapon wid present in the binary as a static constant.")
		fmt.Println("=> weapon IDs are not statically tabled here; static .rdata avenue is empty.")
	}
}
