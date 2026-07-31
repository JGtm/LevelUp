// tmp_kf_offline — EPROUVE NOTRE METHODE offline sur un film local complet.
//
// Deux questions, zero Cheat Engine :
//  1. KILL FEED killer->victime reconstituable offline depuis chunk_27 (events) ?
//     -> parse analysis.ParseHighlightEvents, jointure temporelle KILL@t / DEATH@t.
//  2. ARME presente dans le flux type-2 du film comme id GLOBAL (high-32 de
//     analysis.WeaponIDToName) ou seulement un handle per-match ?
//     -> scan bit-a-bit des paquets replication pour tout u32 == high-32 d'arme.
//     32 bits = cible large => ~0.3 faux positif attendu sur tout le flux : un hit = signal.
//
// Verite-terrain : api_match_stats.json (agregats par joueur : MeleeKills,
// GrenadeKills, HeadshotKills, PowerWeaponKills) pour valider les FAMILLES trouvees.
//
// Usage : tmp_kf_offline <dir_chunks> <api_match_stats.json>
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type pkt struct {
	typ  uint16
	ts   uint64
	data []byte
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

// readU32MSB lit 32 bits MSB-first a la position bit bp.
func readU32MSB(d []byte, bp int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		p := bp + i
		v = (v << 1) | uint32((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// ---- vérité terrain API ----

type apiStats struct {
	Players []struct {
		PlayerID string `json:"PlayerId"`
		Result   int    `json:"Outcome"`
		Stats    struct {
			Core struct {
				Kills            int `json:"Kills"`
				Deaths           int `json:"Deaths"`
				HeadshotKills    int `json:"HeadshotKills"`
				MeleeKills       int `json:"MeleeKills"`
				GrenadeKills     int `json:"GrenadeKills"`
				PowerWeaponKills int `json:"PowerWeaponKills"`
			} `json:"CoreStats"`
		} `json:"PlayerTeamStats"`
	} `json:"Players"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_kf_offline <dir_chunks> [api_match_stats.json]")
		return
	}
	dir := os.Args[1]

	// table high-32 famille -> nom (collapse variantes)
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	fmt.Printf("catalogue: %d familles d'armes (high-32)\n", len(h2n))

	// charge tous les chunks
	files, _ := filepath.Glob(filepath.Join(dir, "filmChunk*"))
	sort.Slice(files, func(i, j int) bool { return natless(files[i], files[j]) })
	fmt.Printf("chunks: %d\n", len(files))

	// ---- 1. KILL FEED offline depuis chunk events (type-3 = dernier, type-9 packet) ----
	var events []analysis.HighlightEvent
	for _, f := range files {
		d := inflate(f)
		// le chunk events est le seul qui parse beaucoup d'events ; on les prend tous
		ev, _ := analysis.ParseHighlightEvents(d, 41)
		if len(ev) > len(events) {
			events = ev
		}
	}
	fmt.Printf("\n=== events offline: %d ===\n", len(events))
	printKillFeed(events)

	// ---- 2. ARME : scan global du flux replication ----
	fmt.Println("\n=== scan armes (id GLOBAL high-32) dans le flux replication ===")
	type famHit struct {
		name  string
		typ   uint16
		count int
	}
	tally := map[string]*famHit{}
	for _, f := range files {
		d := inflate(f)
		for _, p := range listPackets(d) {
			tot := len(p.data)*8 - 32
			for bp := 0; bp <= tot; bp++ {
				if name, ok := h2n[readU32MSB(p.data, bp)]; ok {
					key := fmt.Sprintf("%s|%d", name, p.typ)
					if tally[key] == nil {
						tally[key] = &famHit{name, p.typ, 0}
					}
					tally[key].count++
				}
			}
		}
	}
	if len(tally) == 0 {
		fmt.Println("  AUCUN id d'arme global trouve -> l'arme du film est un HANDLE per-match (pas le tag global)")
	} else {
		var hs []*famHit
		for _, v := range tally {
			hs = append(hs, v)
		}
		sort.Slice(hs, func(i, j int) bool { return hs[i].count > hs[j].count })
		for _, h := range hs {
			fmt.Printf("  %-28s  type-%-2d  x%d\n", h.name, h.typ, h.count)
		}
	}

	// ---- 3. verite-terrain API (familles attendues) ----
	if len(os.Args) >= 3 {
		printAPIGroundTruth(os.Args[2])
	}
}

func printKillFeed(events []analysis.HighlightEvent) {
	type ev struct {
		gt   string
		t    int
		kind string
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.Gamertag, e.TimeMS, "kill"})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.Gamertag, e.TimeMS, "death"})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	fmt.Printf("  kills=%d deaths=%d\n", len(kills), len(deaths))
	fmt.Println("  -- jointure temporelle KILL@t / DEATH@t (tol 250ms) --")
	used := make([]bool, len(deaths))
	paired := 0
	for _, k := range kills {
		best, bd := -1, 251
		for i, d := range deaths {
			if used[i] || d.gt == k.gt {
				continue
			}
			diff := k.t - d.t
			if diff < 0 {
				diff = -diff
			}
			if diff < bd {
				bd, best = diff, i
			}
		}
		if best >= 0 {
			used[best] = true
			paired++
			if paired <= 30 {
				fmt.Printf("  t=%6dms  %-16s -> %-16s\n", k.t, k.gt, deaths[best].gt)
			}
		}
	}
	fmt.Printf("  >>> %d/%d kills apparies a une victime offline\n", paired, len(kills))
}

func printAPIGroundTruth(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("\n(api stats indispo:", err, ")")
		return
	}
	var s apiStats
	if err := json.Unmarshal(raw, &s); err != nil {
		fmt.Println("\n(api stats parse:", err, ")")
		return
	}
	var k, hs, ml, gr, pw int
	for _, p := range s.Players {
		c := p.Stats.Core
		k += c.Kills
		hs += c.HeadshotKills
		ml += c.MeleeKills
		gr += c.GrenadeKills
		pw += c.PowerWeaponKills
	}
	fmt.Println("\n=== verite-terrain API (somme joueurs) ===")
	fmt.Printf("  Kills=%d  Headshot=%d  Melee=%d  Grenade=%d  PowerWeapon=%d\n", k, hs, ml, gr, pw)
	fmt.Println("  (familles attendues : Melee->Sword/Hammer, PowerWeapon->Sniper/Rocket/etc.)")
}

// natless : tri naturel filmChunk2 < filmChunk10.
func natless(a, b string) bool {
	na, nb := chunkNum(a), chunkNum(b)
	return na < nb
}
func chunkNum(p string) int {
	base := filepath.Base(p)
	n := 0
	started := false
	for _, c := range base {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
			started = true
		} else if started {
			break
		}
	}
	return n
}
