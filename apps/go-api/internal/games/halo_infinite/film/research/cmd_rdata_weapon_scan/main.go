//go:build research

// rdata_weapon_scan — THROWAWAY (mission: trouver le mécanisme de SWAP d'arme).
//
// Étape 1 (ce fichier) : cartographier le registre ECS. Pour chaque typeIndex
// présent dans le World (data/cache/film_chunks/000d5950/world_dump.txt), afficher
// le nom de l'archétype (= 1er composant), le nombre de composants, et signaler ceux
// qui portent des composants liés à l'arme/loadout/variant (weapon-state-type-info,
// player-engine-loadout-index, object-multiplayer-properties = 'obje' qui porte un
// variant-name, etc.). But : identifier le typeIndex + slots des entités-armes et de
// l'archétype loadout.
//
// DEPLACE AU LOT J4.5 du PLAN_SUITE_AUDIT_DECODEUR_FILM (2026-09-26) de `cmd/rdata_weapon_scan` vers
// `film/research/cmd_rdata_weapon_scan`, sous le tag `research`, et SCINDE en quatre fichiers
// (seuil de 500 lignes du depot) par deplacement pur : c est un instrument de recherche, et il
// etait le seul consommateur hors du decodeur de `decfilm.LecteurSur` et `decfilm.Paquets` — il
// lit desormais les couches `grammar` et `source` directement, comme ses voisins de `film/research`,
// et la facade ne re-expose plus le lecteur de bits ni le marcheur de paquets.
// SEUL ECART AU DEPLACEMENT PUR : les accents des 18 litteraux imprimes (`fmt.Printf`, et une
// etiquette) sont translitteres en ASCII — `archlint/no_french_label_literal_test.go` couvre
// `internal/games` et refuse tout litteral accentue neuf ; `cmd/` n y etait pas soumis. Aucune
// logique ne change : seul le texte affiche perd ses diacritiques.
//
// Lancer : go run -tags=research ./internal/games/halo_infinite/film/research/cmd_rdata_weapon_scan
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/weapons/filmshell"
)

// Le film etudie, lu par `filmcache.LireChunk` (J2.3, 2026-09-26) : la disposition n'est plus recopiee.
const racineCache, filmEtudie = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`, "000d5950"

// chunk rend le chunk `numero` du film etudie, decompresse ; un chunk illisible arrete l'outil.
func chunk(numero int) []byte {
	raw, err := filmcache.LireChunk(racineCache, filmEtudie, numero)
	if err != nil {
		panic(err)
	}
	return source.Inflate(raw)
}

func worldDumpPath() string {
	return filepath.Join(filmcache.ChunkDir(racineCache, filmEtudie), "world_dump.txt")
}

// loadWorld parses world_dump.txt -> slot:typeIndex map and typeIndex->slots.
func loadWorld() (map[int]int, map[int][]int) {
	f, err := os.Open(worldDumpPath())
	if err != nil {
		panic(err)
	}
	defer f.Close()
	slotType := map[int]int{}
	typeSlots := map[int][]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			parts := strings.SplitN(tok, ":", 2)
			if len(parts) != 2 {
				continue
			}
			slot, e1 := strconv.Atoi(parts[0])
			ti, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil {
				continue
			}
			slotType[slot] = ti
			typeSlots[ti] = append(typeSlots[ti], slot)
		}
	}
	return slotType, typeSlots
}

// weaponish reports whether a component name is weapon/loadout/variant-bearing.
func weaponish(name string) bool {
	for _, k := range []string{"weapon", "loadout", "variant", "equip", "pickup", "item", "grenade", "ability", "multiplayer-properties"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

// ---- littéraux d'armes dans le flux ----

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) (string, bool) {
	for id, n := range filmshell.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

type packet struct {
	typ     uint16
	off     int
	size    int
	ts      uint64
	payload []byte
}

// listPackets : le decoupage de [source.Paquets], dans la forme locale de cet outil. Il
// recopiait l en-tete de seize octets — un marcheur de plus — jusqu au lot 2.4.2.
func listPackets(d []byte) []packet {
	pks := source.Paquets(d, 0)
	out := make([]packet, 0, len(pks))
	off := 0
	for i := range pks {
		taille := len(pks[i].Payload)
		out = append(out, packet{uint16(pks[i].Type), off, taille, pks[i].TS, pks[i].Payload})
		off += 16 + taille
	}
	return out
}

func main() {
	// Mode upstream : pour chaque WST gate=1 (littéral d'arme), cherche en AMONT un header
	// de record dont le slot ∈ 512-519 (biped joueur). Teste l'attribution par remontée.
	if len(os.Args) >= 2 && os.Args[1] == "upstream" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		upstreamScan(chunkIdx)
		return
	}

	// Mode litplayer : pour chaque littéral, donne le slot du 1er record du paquet (=joueur).
	if len(os.Args) >= 2 && os.Args[1] == "litplayer" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		litPlayer(chunkIdx)
		return
	}

	// Mode litpattern : agrège la structure bit autour de TOUS les littéraux d'un chunk.
	if len(os.Args) >= 2 && os.Args[1] == "litpattern" {
		chunks := []int{3, 4, 10, 15, 20, 25}
		if len(os.Args) >= 3 {
			chunks = nil
			for _, a := range os.Args[2:] {
				if v, e := strconv.Atoi(a); e == nil {
					chunks = append(chunks, v)
				}
			}
		}
		litPattern(chunks)
		return
	}

	// Mode sizes : distribution des tailles de paquets type-0 + littéraux par gros paquet.
	if len(os.Args) >= 2 && os.Args[1] == "sizes" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		sizesMode(chunkIdx)
		return
	}

	// Mode litctx : dump du contexte de bits autour des littéraux d'un paquet précis.
	// usage: litctx <chunk> <pktIndex>
	if len(os.Args) >= 2 && os.Args[1] == "litctx" {
		reg, _ := grammar.ParseRegistryChunk(chunk(0))
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		pktIdx, _ := strconv.Atoi(os.Args[3])
		litCtx(reg, chunkIdx, pktIdx)
		return
	}

	// Mode litloc : localise les littéraux dans les records décodés.
	if len(os.Args) >= 2 && os.Args[1] == "litloc" {
		reg, err := grammar.ParseRegistryChunk(chunk(0))
		if err != nil {
			panic(err)
		}
		chunkIdx, maxPkts := 3, 1199
		if len(os.Args) >= 3 {
			chunkIdx, _ = strconv.Atoi(os.Args[2])
		}
		if len(os.Args) >= 4 {
			maxPkts, _ = strconv.Atoi(os.Args[3])
		}
		litLoc(reg, worldDumpPath(), chunkIdx, maxPkts)
		return
	}

	// Mode litscan : scan des littéraux d'armes dans des chunks gameplay.
	if len(os.Args) >= 2 && os.Args[1] == "litscan" {
		chunks := []int{2, 3, 4, 5, 10, 15, 20, 25}
		if len(os.Args) >= 3 {
			chunks = nil
			for _, a := range os.Args[2:] {
				if v, e := strconv.Atoi(a); e == nil {
					chunks = append(chunks, v)
				}
			}
		}
		for _, c := range chunks {
			litScan(c)
			fmt.Println()
		}
		return
	}

	reg, err := grammar.ParseRegistryChunk(chunk(0))
	if err != nil {
		panic(err)
	}
	_, typeSlots := loadWorld()
	fmt.Printf("registre : %d archetypes ; world : %d typeIndex distincts\n\n", len(reg.Archetypes), len(typeSlots))

	// Liste triée des typeIndex présents dans le World.
	var tis []int
	for ti := range typeSlots {
		tis = append(tis, ti)
	}
	sort.Ints(tis)

	fmt.Printf("================ ARCHETYPES PRESENTS DANS LE WORLD ================\n")
	for _, ti := range tis {
		arch, ok := reg.Archetype(ti)
		name := "(hors registre)"
		ncomp := 0
		if ok {
			ncomp = len(arch.Components)
			if ncomp > 0 {
				name = arch.Components[0]
			}
		}
		slots := typeSlots[ti]
		sort.Ints(slots)
		// résumé compact des slots
		slotStr := fmt.Sprintf("%d slots", len(slots))
		if len(slots) <= 12 {
			slotStr = fmt.Sprintf("slots=%v", slots)
		} else {
			slotStr = fmt.Sprintf("%d slots [%d..%d]", len(slots), slots[0], slots[len(slots)-1])
		}
		// composants weapon-ish
		var wcomp []string
		for _, c := range arch.Components {
			if weaponish(c) {
				wcomp = append(wcomp, c)
			}
		}
		flag := ""
		if len(wcomp) > 0 {
			flag = "  <<< WEAPON/LOADOUT: " + strings.Join(dedup(wcomp), ",")
		}
		fmt.Printf("  ti=%-3d  %-44s  ncomp=%-3d  %s%s\n", ti, truncate(name, 44), ncomp, slotStr, flag)
	}

	// Dump détaillé des composants des archétypes candidats (ceux weapon-ish ou
	// passés en argument).
	fmt.Printf("\n================ COMPOSANTS DES ARCHETYPES CANDIDATS ================\n")
	candidates := map[int]bool{}
	for _, ti := range tis {
		arch, ok := reg.Archetype(ti)
		if !ok {
			continue
		}
		for _, c := range arch.Components {
			if weaponish(c) {
				candidates[ti] = true
				break
			}
		}
	}
	// args = typeIndex supplémentaires à dumper
	for _, a := range os.Args[1:] {
		if v, e := strconv.Atoi(a); e == nil {
			candidates[v] = true
		}
	}
	var cands []int
	for ti := range candidates {
		cands = append(cands, ti)
	}
	sort.Ints(cands)
	for _, ti := range cands {
		arch, _ := reg.Archetype(ti)
		fmt.Printf("\n--- ti=%d (%d composants, %d slots dans le World) ---\n", ti, len(arch.Components), len(typeSlots[ti]))
		for i, c := range arch.Components {
			mark := ""
			if weaponish(c) {
				mark = "  <<<"
			}
			fmt.Printf("  i%-2d %s%s\n", i, c, mark)
		}
	}
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
