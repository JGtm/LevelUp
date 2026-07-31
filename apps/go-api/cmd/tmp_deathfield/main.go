// tmp_deathfield — SONDE (mission weapon-attribution-v3) : capture les champs de
// l'événement de MORT (object-dead-state heavy form, FUN_140c1dd44) sur la VICTIME
// au tick de mort, pour les 93 morts connues du match 000d5950.
//
// MÉCANISME (établi par RE du kill feed) : le kill feed lit l'arme/source-de-dégât
// dans le composant object-dead-state de la victime, écrit au moment du kill. La
// forme lourde biped (typeIndex==0x23) est désérialisée par FUN_140c1dd44 et porte :
//   - +0x04 / +0x08 : 2 enums R(5) résolus par table de tags (type-de-dégât/méthode).
//   - +0x10         : une RÉFÉRENCE global-id R(32) résolue par GetLocalHandleFromGlobalId
//     (table DAT_144b404f0) = MÊME mécanisme que l'arme du WeaponState
//     -> candidat fort pour l'ARME / source de dégât.
//   - +0x0c (R4), +0x0e (R3), +0x14 (R3), +0x18 (R6) : valeurs annexes.
//
// La sonde rejoue les FRAME deltas avec le World capturé (slots biped 512-519),
// capture filmdec.DeadState sur chaque biped dont le composant dead-state est présent,
// repère les ticks où Mort passe à true (la mort), puis croise par TEMPS avec les 93
// morts du chunk_27 (highlight events) + les médailles d'arme (medals_earned).
//
// Usage : tmp_deathfield [maxChunks]   (défaut : tous les chunks gameplay 02..26)
package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"

	_ "github.com/duckdb/duckdb-go/v2"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

// t0Us est l'origine temporelle du film (µs) fournie par la mission. time_ms = (ts-t0)/1000.
const t0Us = uint64(4537898226)

// combo calibré (cf. tmp_worldreplay) : extra=false, idLowBits=11.
var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type packet struct {
	typ     uint16
	ts      uint64
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{typ, ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

// knownHigh32 : une arme du catalogue dont high-32 == v (le discriminant famille).
func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

// knownLow32 : une arme du catalogue dont low-32 == v (suffixe filmshell partagé).
func knownLow32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id) == v {
			return n, true
		}
	}
	return "", false
}

func parseWorld(reg *filmdec.Registry, path string) (*filmdec.World, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w, nil
}

func freshWorld(reg *filmdec.Registry, path string) *filmdec.World {
	w, _ := parseWorld(reg, path)
	return w
}

// deathObs = une mort observée dans le film (dead-state Mort==true sur un biped).
type deathObs struct {
	ts        uint64
	timeMs    int
	slot      uint32
	ds        filmdec.DeadState
	preMask   uint64 // bits i0..i10 du mask (composants AVANT le dead-state i11)
	deadStart int    // StartBit du composant dead-state dans le payload
}

// preMaskBits = masque des composants i0..i10 (ceux qui précèdent le dead-state i11).
const preMaskBits uint64 = (uint64(1) << 11) - 1

// known death (ground truth) du chunk_27.
type knownDeath struct {
	victimX uint64
	timeMs  int
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	filmdec.SetRecordStateParam(2)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	worldPath := cache + "/world_dump.txt"

	// --- 1) GROUND TRUTH : 93 morts (chunk_27 highlight events) ---
	heRaw, _ := os.ReadFile(cache + "/chunk_27.bin")
	events, err := analysis.ParseHighlightEvents(heRaw, 0)
	if err != nil {
		fmt.Println("ParseHighlightEvents:", err)
	}
	var deaths []knownDeath
	for _, e := range events {
		if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, knownDeath{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].timeMs < deaths[j].timeMs })
	fmt.Printf("=== GROUND TRUTH : %d morts (type-20) extraites du chunk_27 ===\n", len(deaths))
	for i, d := range deaths {
		if i < 6 || i >= len(deaths)-2 {
			fmt.Printf("  death #%d victim=%d t=%dms\n", i, d.victimX, d.timeMs)
		} else if i == 6 {
			fmt.Println("  ...")
		}
	}

	// --- 2) REPLAY : capture dead-states sur les FRAME deltas biped ---
	// IMPORTANT : on ne capture que sur des records CLEAN (DesyncAt==-1) où la traversée
	// a atteint object-dead-state-component sans désaligner — sinon le bit "Mort" et le
	// global-id sont du bruit issu d'un flux décalé.
	var obs []deathObs
	totalDeltaBiped, cleanRec := 0, 0
	deadCleanPresent, deadDesyncPresent := 0, 0
	mortTrueClean, mortTrueDesync := 0, 0
	for idx := 2; idx <= maxChunk; idx++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))
		frames := listFrames(data)
		for _, fr := range frames {
			w := freshWorld(reg, worldPath)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				totalDeltaBiped++
				clean := r.DesyncAt == -1
				if clean {
					cleanRec++
				}
				if r.Trace.Dead == nil {
					continue
				}
				ds := *r.Trace.Dead
				if clean {
					deadCleanPresent++
					if ds.Mort {
						mortTrueClean++
						deadStart := -1
						for _, c := range r.Trace.Comps {
							if c.Name == "object-dead-state-component" {
								deadStart = c.StartBit
							}
						}
						obs = append(obs, deathObs{fr.ts, int((fr.ts - t0Us) / 1000), r.Slot, ds, r.Trace.Mask & preMaskBits, deadStart})
					}
				} else {
					deadDesyncPresent++
					if ds.Mort {
						mortTrueDesync++
					}
				}
			}
		}
	}
	fmt.Printf("\n=== REPLAY chunks 02..%d : deltas biped=%d (clean=%d) ===\n", maxChunk, totalDeltaBiped, cleanRec)
	fmt.Printf("    dead-state sur record CLEAN : présent=%d, Mort==true=%d  <-- capturés\n", deadCleanPresent, mortTrueClean)
	fmt.Printf("    dead-state sur record DÉSYNC : présent=%d, Mort==true=%d  <-- IGNORÉS (bruit)\n", deadDesyncPresent, mortTrueDesync)

	// répartition par slot des dead-states CLEAN Mort==true capturés
	perSlot := map[uint32]int{}
	for _, o := range obs {
		perSlot[o.slot]++
	}
	var sl []int
	for s := range perSlot {
		sl = append(sl, int(s))
	}
	sort.Ints(sl)
	fmt.Printf("    répartition par slot des captures (clean, Mort==true) : ")
	for _, s := range sl {
		fmt.Printf("slot%d=%d ", s, perSlot[uint32(s)])
	}
	fmt.Println()

	// --- 2b) STABILITÉ : les champs sont-ils CONSTANTS sur les frames consécutives d'une
	// même mort (=bit-exact) ou varient-ils frame-à-frame (=bruit) ? On regroupe par slot
	// et fenêtre 1.5s, et on affiche le nombre de valeurs GID/EnumA distinctes par groupe.
	analyzeStability(obs)

	// --- 2c) FIABILITÉ par préfixe : si i0..i10 sont ABSENTS du mask, le dead-state i11
	// commence juste après l'en-tête (mask) -> lecture la plus fiable (pas de dérive de
	// largeur amont). On compare la qualité des champs selon le nb de composants amont.
	analyzePrefixReliability(obs)

	// --- 2d) VOCABULAIRE slot 519 (le biped le plus décodé) : un champ "source de dégât"
	// réel a un PETIT vocabulaire répété ; du bruit a ~autant de valeurs que d'obs.
	analyzeSlotVocabulary(obs, 519)
	analyzeSlotVocabulary(obs, 515)

	// --- 2e) IDENTITÉ slot : quel xuid victime correspond le mieux aux ticks de mort
	// (transitions Mort false->true) d'un slot donné ? (les corpses durent ; on prend
	// les transitions = débuts de mort distincts à >2s d'écart).
	identifySlot(obs, deaths, 519)
	identifySlot(obs, deaths, 515)

	// --- 3) Caractérise les dead-states capturés ---
	if len(obs) == 0 {
		fmt.Println("\n!!! AUCUN dead-state Mort==true capturé.")
		fmt.Println("    => soit la traversée delta désync AVANT object-dead-state-component sur les bipeds,")
		fmt.Println("       soit le dead-state ne se réplique pas via le chemin delta testé (combo idLowBits=11).")
		fmt.Println("    Diagnostic ci-dessous : combien de bipeds atteignent object-dead-state-component ?")
		diagReachDeadState(reg, worldPath, maxChunk)
		return
	}

	// dé-duplique : un même tick de mort se re-réplique sur plusieurs frames ; on garde
	// la 1ère occurrence par (slot, fenêtre ~1s) avec un GlobalID stable.
	sort.Slice(obs, func(i, j int) bool {
		if obs[i].slot != obs[j].slot {
			return obs[i].slot < obs[j].slot
		}
		return obs[i].ts < obs[j].ts
	})
	fmt.Printf("\n=== %d observations Mort==true (slot, t, champs capturés) — premières 40 ===\n", len(obs))
	gidDistinct := map[uint32]int{}
	enumADist := map[int32]int{}
	enumBDist := map[int32]int{}
	gidHigh32Hit, gidLow32Hit := 0, 0
	for i, o := range obs {
		ds := o.ds
		gidDistinct[ds.GlobalID]++
		enumADist[ds.EnumA]++
		enumBDist[ds.EnumB]++
		tag := ""
		if ds.GlobalID != 0xFFFFFFFF {
			if n, ok := knownHigh32(ds.GlobalID); ok {
				tag += " GID.high32=ARME:" + n
				gidHigh32Hit++
			}
			if n, ok := knownLow32(ds.GlobalID); ok {
				tag += " GID.low32=ARME:" + n
				gidLow32Hit++
			}
		}
		if i < 40 {
			fmt.Printf("  slot=%d t=%-7dms Mort=%v EnumA=%-3d EnumB=%-3d v0c=%-2d v0e=%-2d hasRef=%v gidPres=%v GID=0x%08x v14=%d v18=%d%s\n",
				o.slot, o.timeMs, ds.Mort, ds.EnumA, ds.EnumB, ds.Val0c, ds.Val0e,
				ds.HasRef, ds.GIDPresent, ds.GlobalID, ds.Val14, ds.Val18, tag)
		}
	}

	// --- 4) Distributions des champs candidats ---
	fmt.Printf("\n=== DISTRIBUTION GlobalID (+0x10, R32) sur %d obs ===\n", len(obs))
	dumpDistU32(gidDistinct)
	fmt.Printf("  -> GID dont high32 ∈ catalogue armes : %d ; dont low32 ∈ catalogue : %d\n", gidHigh32Hit, gidLow32Hit)

	fmt.Printf("\n=== DISTRIBUTION EnumA (+0x04, R5) — candidat type-de-dégât/méthode ===\n")
	dumpDistI32(enumADist)
	fmt.Printf("\n=== DISTRIBUTION EnumB (+0x08, R5) — candidat type-de-dégât/méthode ===\n")
	dumpDistI32(enumBDist)

	// --- 5) Croisement TEMPS avec les 93 morts (±200ms) ---
	fmt.Printf("\n=== CROISEMENT TEMPS : dead-state Mort==true vs 93 morts connues (±200ms) ===\n")
	matched := 0
	dedup := map[string]bool{}
	for _, o := range obs {
		key := fmt.Sprintf("%d:%d", o.slot, o.timeMs/1000)
		if dedup[key] {
			continue
		}
		dedup[key] = true
		best := -1
		bestDt := 1 << 30
		for di, d := range deaths {
			dt := o.timeMs - d.timeMs
			if dt < 0 {
				dt = -dt
			}
			if dt < bestDt {
				bestDt = dt
				best = di
			}
		}
		if best >= 0 && bestDt <= 200 {
			matched++
			if matched <= 30 {
				wn := "(GID absent/inconnu)"
				if n, ok := knownHigh32(o.ds.GlobalID); ok {
					wn = "high32=" + n
				} else if n, ok := knownLow32(o.ds.GlobalID); ok {
					wn = "low32=" + n
				} else if o.ds.GlobalID != 0xFFFFFFFF {
					wn = fmt.Sprintf("GID=0x%08x (hors catalogue)", o.ds.GlobalID)
				}
				fmt.Printf("  death t=%-7dms victim=%d <-> dead-state slot=%d t=%-7dms (dt=%dms) EnumA=%d EnumB=%d %s\n",
					deaths[best].timeMs, deaths[best].victimX, o.slot, o.timeMs, bestDt, o.ds.EnumA, o.ds.EnumB, wn)
			}
		}
	}
	fmt.Printf("  >>> %d dead-states (dédupliqués slot+s) appariés à une mort connue (±200ms)\n", matched)

	// --- 6) Vérité-terrain partielle : médailles d'arme (medals_earned) ---
	crossWeaponMedals(deaths)
}

// analyzeStability regroupe les obs par (slot, fenêtre 1.5s) et compte, par groupe,
// les valeurs distinctes de GlobalID et EnumA. Si une mort est bit-exacte, ses frames
// consécutives doivent porter la MÊME valeur (1 distincte) ; plusieurs valeurs = bruit.
func analyzeStability(obs []deathObs) {
	type grp struct {
		slot   uint32
		t0, t1 int
		gids   map[uint32]bool
		enumA  map[int32]bool
		frames int
	}
	var groups []*grp
	sort.Slice(obs, func(i, j int) bool {
		if obs[i].slot != obs[j].slot {
			return obs[i].slot < obs[j].slot
		}
		return obs[i].timeMs < obs[j].timeMs
	})
	for _, o := range obs {
		var g *grp
		if len(groups) > 0 {
			last := groups[len(groups)-1]
			if last.slot == o.slot && o.timeMs-last.t1 <= 1500 {
				g = last
			}
		}
		if g == nil {
			g = &grp{slot: o.slot, t0: o.timeMs, gids: map[uint32]bool{}, enumA: map[int32]bool{}}
			groups = append(groups, g)
		}
		g.t1 = o.timeMs
		g.gids[o.ds.GlobalID] = true
		g.enumA[o.ds.EnumA] = true
		g.frames++
	}
	stableGID, stableEnumA := 0, 0
	fmt.Printf("\n=== STABILITÉ par mort regroupée (slot+fenêtre1.5s) : %d groupes ===\n", len(groups))
	for i, g := range groups {
		if len(g.gids) == 1 {
			stableGID++
		}
		if len(g.enumA) == 1 {
			stableEnumA++
		}
		if i < 20 {
			fmt.Printf("  slot=%d t=[%d..%d]ms frames=%-3d GIDdistinct=%d EnumAdistinct=%d\n",
				g.slot, g.t0, g.t1, g.frames, len(g.gids), len(g.enumA))
		}
	}
	fmt.Printf("  >>> groupes à GID constant=%d/%d ; à EnumA constant=%d/%d (constant => bit-exact ; variable => bruit)\n",
		stableGID, len(groups), stableEnumA, len(groups))
}

// analyzePrefixReliability sépare les obs selon le nombre de composants i0..i10 présents
// dans le mask. preMask==0 (aucun composant amont) => le dead-state i11 est lu juste après
// le mask, sans dérive amont possible : c'est la lecture la plus fiable.
func analyzePrefixReliability(obs []deathObs) {
	fmt.Printf("\n=== FIABILITÉ par préfixe (composants i0..i10 présents avant le dead-state) ===\n")
	byPop := map[int]int{}     // popcount(preMask) -> nb obs
	zeroPrefix := []deathObs{} // preMask==0 : dead-state = 1er composant
	for _, o := range obs {
		byPop[popcount(o.preMask)]++
		if o.preMask == 0 {
			zeroPrefix = append(zeroPrefix, o)
		}
	}
	var pops []int
	for p := range byPop {
		pops = append(pops, p)
	}
	sort.Ints(pops)
	for _, p := range pops {
		fmt.Printf("  composants amont présents=%d : %d obs\n", p, byPop[p])
	}
	fmt.Printf("\n  -- obs à préfixe VIDE (preMask==0, dead-state lu juste après le mask) : %d --\n", len(zeroPrefix))
	gidHit, low32Hit := 0, 0
	enumA := map[int32]int{}
	for i, o := range zeroPrefix {
		enumA[o.ds.EnumA]++
		tag := ""
		if n, ok := knownHigh32(o.ds.GlobalID); ok {
			tag = " GID.high32=ARME:" + n
			gidHit++
		} else if n, ok := knownLow32(o.ds.GlobalID); ok {
			tag = " GID.low32=ARME:" + n
			low32Hit++
		}
		if i < 25 {
			fmt.Printf("    slot=%d t=%-7dms deadStart=%d EnumA=%-3d EnumB=%-3d v0c=%-2d v0e=%-2d GID=0x%08x v14=%d v18=%d%s\n",
				o.slot, o.timeMs, o.deadStart, o.ds.EnumA, o.ds.EnumB, o.ds.Val0c, o.ds.Val0e, o.ds.GlobalID, o.ds.Val14, o.ds.Val18, tag)
		}
	}
	fmt.Printf("  >>> préfixe-vide : GID.high32∈catalogue=%d, GID.low32∈catalogue=%d sur %d obs\n", gidHit, low32Hit, len(zeroPrefix))
	fmt.Printf("  distribution EnumA (préfixe-vide) : ")
	type kv struct {
		k int32
		v int
	}
	var arr []kv
	for k, v := range enumA {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for _, e := range arr {
		fmt.Printf("%d:%d ", e.k, e.v)
	}
	fmt.Println()
}

// analyzeSlotVocabulary mesure, pour un slot donné, la taille du vocabulaire de chaque
// champ (GID, EnumA, EnumB, Val0c). Petit vocabulaire répété = champ sémantique réel ;
// vocabulaire ≈ nb d'obs = bruit (lecture désalignée).
func analyzeSlotVocabulary(obs []deathObs, slot uint32) {
	var sub []deathObs
	for _, o := range obs {
		if o.slot == slot {
			sub = append(sub, o)
		}
	}
	if len(sub) == 0 {
		fmt.Printf("\n=== VOCABULAIRE slot %d : aucune obs ===\n", slot)
		return
	}
	gid := map[uint32]int{}
	ea := map[int32]int{}
	eb := map[int32]int{}
	gidPresent := 0
	for _, o := range sub {
		ea[o.ds.EnumA]++
		eb[o.ds.EnumB]++
		if o.ds.GIDPresent && o.ds.GlobalID != 0xFFFFFFFF {
			gid[o.ds.GlobalID]++
			gidPresent++
		}
	}
	fmt.Printf("\n=== VOCABULAIRE slot %d : %d obs ===\n", slot, len(sub))
	fmt.Printf("  GID présent (gidPres && !=FFFFFFFF) : %d obs ; valeurs GID distinctes : %d\n", gidPresent, len(gid))
	fmt.Printf("  EnumA : %d valeurs distinctes ; EnumB : %d valeurs distinctes (sur %d obs)\n", len(ea), len(eb), len(sub))
	fmt.Printf("  -> ratio distinct/obs : GID=%.2f EnumA=%.2f EnumB=%.2f (proche de 1 = bruit ; petit = sémantique)\n",
		ratio(len(gid), gidPresent), ratio(len(ea), len(sub)), ratio(len(eb), len(sub)))
	// GID répétés (vocabulaire potentiel de sources de dégât)
	type kv struct {
		k uint32
		v int
	}
	var arr []kv
	for k, v := range gid {
		if v >= 2 {
			arr = append(arr, kv{k, v})
		}
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	fmt.Printf("  GID répétés (count>=2) : %d valeurs\n", len(arr))
	for i := 0; i < len(arr) && i < 15; i++ {
		tag := ""
		if n, ok := knownHigh32(arr[i].k); ok {
			tag = " (=ARME high32 " + n + ")"
		} else if n, ok := knownLow32(arr[i].k); ok {
			tag = " (=ARME low32 " + n + ")"
		}
		fmt.Printf("    GID=0x%08x : %d fois%s\n", arr[i].k, arr[i].v, tag)
	}
}

// identifySlot extrait les "débuts de mort" d'un slot (transitions : obs séparées de >2s
// = morts distinctes) et, pour chacun, le victim-xuid de la mort connue la plus proche en
// temps. Le xuid majoritaire = l'identité du slot (slot->xuid).
func identifySlot(obs []deathObs, deaths []knownDeath, slot uint32) {
	var sub []deathObs
	for _, o := range obs {
		if o.slot == slot {
			sub = append(sub, o)
		}
	}
	sort.Slice(sub, func(i, j int) bool { return sub[i].timeMs < sub[j].timeMs })
	// débuts de mort : 1er d'un cluster (>2s depuis le précédent capturé)
	var starts []deathObs
	prev := -100000
	for _, o := range sub {
		if o.timeMs-prev > 2000 {
			starts = append(starts, o)
		}
		prev = o.timeMs
	}
	victimVotes := map[uint64]int{}
	matched := 0
	for _, s := range starts {
		best, bestDt := uint64(0), 1<<30
		for _, d := range deaths {
			dt := s.timeMs - d.timeMs
			if dt < 0 {
				dt = -dt
			}
			if dt < bestDt {
				bestDt = dt
				best = d.victimX
			}
		}
		if bestDt <= 300 {
			victimVotes[best]++
			matched++
		}
	}
	fmt.Printf("\n=== IDENTITÉ slot %d : %d débuts-de-mort distincts, %d appariés (±300ms) ===\n", slot, len(starts), matched)
	type kv struct {
		k uint64
		v int
	}
	var arr []kv
	for k, v := range victimVotes {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for i := 0; i < len(arr) && i < 5; i++ {
		fmt.Printf("    victim xuid=%-18d : %d votes\n", arr[i].k, arr[i].v)
	}
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		n += int(x & 1)
		x >>= 1
	}
	return n
}

func diagReachDeadState(reg *filmdec.Registry, worldPath string, maxChunk int) {
	reachHist := map[string]int{} // 1er composant qui désync sur un biped, par nom
	deadReached := 0
	bipedTotal := 0
	for idx := 2; idx <= maxChunk; idx++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))
		for _, fr := range listFrames(data) {
			w := freshWorld(reg, worldPath)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				bipedTotal++
				sawDead := false
				for _, c := range r.Trace.Comps {
					if c.Name == "object-dead-state-component" {
						sawDead = true
					}
				}
				if sawDead {
					deadReached++
				}
				if r.DesyncAt != -1 {
					if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
						reachHist[arch.Components[r.DesyncAt]]++
					}
				}
			}
		}
	}
	fmt.Printf("  bipeds traversés=%d ; ayant atteint object-dead-state-component (présent dans le mask + porté)=%d\n",
		bipedTotal, deadReached)
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range reachHist {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	fmt.Println("  1er composant qui désync sur un biped (top 15) :")
	for i := 0; i < len(arr) && i < 15; i++ {
		fmt.Printf("    %-45s : %d\n", arr[i].k, arr[i].v)
	}
}

func dumpDistU32(m map[uint32]int) {
	type kv struct {
		k uint32
		v int
	}
	var arr []kv
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for i := 0; i < len(arr) && i < 20; i++ {
		fmt.Printf("    0x%08x : %d\n", arr[i].k, arr[i].v)
	}
	fmt.Printf("    (valeurs distinctes : %d)\n", len(arr))
}

func dumpDistI32(m map[int32]int) {
	type kv struct {
		k int32
		v int
	}
	var arr []kv
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for i := 0; i < len(arr) && i < 32; i++ {
		fmt.Printf("    %-4d : %d\n", arr[i].k, arr[i].v)
	}
	fmt.Printf("    (valeurs distinctes : %d)\n", len(arr))
}

// crossWeaponMedals charge les médailles d'arme du match (medals_earned) — vérité-terrain
// partielle : une médaille Sniper/Sword/etc. à un time_ms donné ancre l'arme d'un kill.
func crossWeaponMedals(deaths []knownDeath) {
	fmt.Printf("\n=== VÉRITÉ-TERRAIN PARTIELLE : medals_earned (armes) du match 000d5950 ===\n")
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("  (DB indisponible :", err, ")")
		return
	}
	defer db.Close()
	var fullID string
	if err := db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&fullID); err != nil {
		fmt.Println("  match lookup:", err)
		return
	}
	rows, err := db.Query(`SELECT xuid, medal_name_id, COUNT(*) FROM medals_earned WHERE match_id = ? GROUP BY xuid, medal_name_id ORDER BY COUNT(*) DESC`, fullID)
	if err != nil {
		fmt.Println("  medals_earned query:", err)
		return
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var xuid sql.NullString
		var nameID sql.NullInt64
		var cnt int
		rows.Scan(&xuid, &nameID, &cnt)
		if n < 25 {
			fmt.Printf("  xuid=%-18s name_id=%-12d count=%d\n", xuid.String, nameID.Int64, cnt)
		}
		n++
	}
	fmt.Printf("  (%d lignes médaille/joueur ; les médailles arme-spécifiques ancrent l'arme par joueur)\n", n)
}
