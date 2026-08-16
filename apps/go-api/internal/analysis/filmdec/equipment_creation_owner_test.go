package filmdec

// equipment_creation_owner_test.go — QUI a posé l'objet, et QUOI : la DEUXIÈME CHAÎNE.
//
// CE QUE LA PREMIÈRE CHAÎNE A DONNÉ (2026-08-17, phase 0) : le record de création de ti=37
// porte, dans son bloc `object-multiplayer-properties`, un mot de 32 bits qui prend une
// POIGNÉE de valeurs par film — et ces valeurs sont des GlobalID de tags `eqip` du jeu
// (11 sur 11 résolues contre 105 tags `eqip`, cf. himap/sonde_ti37_gamefiles_test.go).
// L'objet est donc IDENTIFIÉ par sa définition. Il n'est pas encore NOMMÉ.
//
// CE QUE CETTE CHAÎNE-CI APPORTE, et pourquoi elle est indépendante : le film dit AUSSI, par
// un canal sans rapport (`i48`, cf. ability_rank.go), quelle capacité d'armure chaque joueur
// PORTE, sous forme d'un rang dans la palette du match — dont la RECETTE_LOADOUT §13 nomme
// une partie (rang 1 détecteur de menaces, 2 mur de protection, 12 traqueur…). Si les objets
// créés à côté d'un porteur de rang R portent SYSTÉMATIQUEMENT le même `eqip`, alors ce
// `eqip` EST la capacité de rang R — et il prend son nom.
//
// LE CRÉATEUR NE SE LIT PAS, IL SE MESURE. Le champ `entity-ref-index5` du default-state est
// une porte FERMÉE sur 274 records de création sur 274 (`000d5950`) : le film ne transmet
// aucune référence de créateur à la naissance. Le porteur candidat est donc le bipède le plus
// proche à l'instant de la pose, et le TÉMOIN — un autre bipède vivant au même instant — dit
// si cette proximité veut dire quelque chose.
//
// LECTURE SEULE, gardé par EQUIP_CREATION_FILM. UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_CREATION_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentCreationOwner$' -timeout 60m -v

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
)

// equipOwnerWindowUS est la fenêtre temporelle dans laquelle un échantillon de position de
// bipède est jugé contemporain de la pose. 250 ms : la même borne que celle qui sépare deux
// vies de projectile (projectileGapUS), et à peu près quinze images de réplication.
const equipOwnerWindowUS = 250_000

func TestEquipmentCreationOwner(t *testing.T) {
	dir := os.Getenv(equipCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)

	wr, unite := equipOwnerRange(t)
	band := worldObjectSlotBand(dir, CountFilmChunks(dir), EquipmentTypeIndex)
	cre, _, err := ScanFilmEquipmentCreationsForBand(dir, &wr, band)
	if err != nil {
		t.Fatalf("balayage des créations impossible : %v", err)
	}
	cre = equipCreationFilter(cre, func(c EquipmentCreation) bool { return c.MaskHasI0 })
	if len(cre) == 0 {
		t.Skip("aucun record de création exploitable sur ce film")
	}
	t.Logf("unité de distance : %s", unite)
	pos := equipOwnerBipeds(t, dir, lay, wr)
	ranks := equipOwnerRanks(t, dir)
	t.Logf("FILM %s · largeurs %v · %d créations (i0 au masque) · %d positions de bipède"+
		" · %d slots à rang connu", dir, lay.AxisW, len(cre), len(pos), len(ranks))
	equipOwnerCross(t, cre, pos, ranks)
}

// equipOwnerSample est une position de bipède, normalisée par l'AABB comme les objets.
type equipOwnerSample struct {
	slot uint32
	ts   uint64
	p    [3]float32
}

// equipOwnerRange rend les bornes de déquantification. Sans carte nommée
// (EQUIP_MAP + EQUIP_BOUNDS), la mesure reste en coordonnées NORMALISÉES de l'AABB : les deux
// nuages sont dans le même repère, ce qui suffit à comparer une distance à son témoin — mais
// pas à l'énoncer en mètres, et le seuil du plan est en mètres. On le dit plutôt que de
// convertir avec une échelle devinée.
func equipOwnerRange(t *testing.T) (Vec3Range, string) {
	t.Helper()
	nom, chemin := os.Getenv("EQUIP_MAP"), os.Getenv("EQUIP_BOUNDS")
	if nom == "" || chemin == "" {
		return equipCreationUnitRange, "fraction de l'AABB (EQUIP_MAP/EQUIP_BOUNDS absents)"
	}
	cat, err := LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible (%s) : %v", chemin, err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", nom, err)
	}
	r := e.Range()
	return r, fmt.Sprintf("MÈTRES (carte %q, module %s, emprise %.1f x %.1f x %.1f m)",
		nom, e.Module, e.Max[0]-e.Min[0], e.Max[1]-e.Min[1], e.Max[2]-e.Min[2])
}

// equipOwnerBipeds lit le nuage des bipèdes en QUANTA et le déquantifie avec les MÊMES bornes
// que les objets du monde. Trié par instant pour la recherche par fenêtre.
func equipOwnerBipeds(t *testing.T, dir string, lay I0Layout, wr Vec3Range) []equipOwnerSample {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	raw, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("nuage des bipèdes indisponible : %v", err)
	}
	out := make([]equipOwnerSample, 0, len(raw))
	for _, p := range raw {
		var v [3]float32
		for a := 0; a < 3; a++ {
			f := (float32(p.Q[a]) + 0.5) / float32(uint64(1)<<lay.AxisW[a])
			v[a] = wr[a].Min + f*(wr[a].Max-wr[a].Min)
		}
		out = append(out, equipOwnerSample{slot: p.Slot, ts: p.TimestampUS, p: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// equipOwnerRanks rend le rang de capacité par slot de bipède. Un slot dont deux lectures
// divergent est ÉCARTÉ : une vie qui porterait deux capacités n'est pas une vie, c'est une
// lecture fausse — la même règle que PlayerIndexTable.
func equipOwnerRanks(t *testing.T, dir string) map[uint32]int {
	t.Helper()
	reads, st, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacité indisponibles : %v", err)
		return nil
	}
	bySlot := map[uint32]map[int]int{}
	for _, r := range reads {
		if bySlot[r.Slot] == nil {
			bySlot[r.Slot] = map[int]int{}
		}
		bySlot[r.Slot][r.Rank]++
	}
	out := map[uint32]int{}
	divergents := 0
	for s, m := range bySlot {
		if len(m) > 1 {
			divergents++
			continue
		}
		for r := range m {
			out[s] = r
		}
	}
	t.Logf("i48 — %d lectures (%d records, %d au masque, %d portes fermées) · %d slots"+
		" · %d écartés pour divergence", len(reads), st.Records, st.WithI48, st.Gated,
		len(out), divergents)
	return out
}

// equipOwnerCross croise l'identifiant `eqip` de chaque création avec le rang du bipède le
// plus proche à l'instant de la pose, et publie le TÉMOIN en regard.
func equipOwnerCross(
	t *testing.T, cre []EquipmentCreation, pos []equipOwnerSample, ranks map[uint32]int,
) {
	table := map[uint32]map[int]int{}
	temoin := map[uint32]map[int]int{}
	var dReal, dTemoin []float64
	sans := 0
	for i, c := range cre {
		near, other, ok := equipOwnerNearest(pos, c, i)
		if !ok {
			sans++
			continue
		}
		id := uint32(c.MPPVal[MPPWord32])
		dReal = append(dReal, equipOwnerDist(c, near))
		dTemoin = append(dTemoin, equipOwnerDist(c, other))
		equipOwnerPut(table, id, ranks, near.slot)
		equipOwnerPut(temoin, id, ranks, other.slot)
	}
	t.Logf("== CROISEMENT `eqip` x RANG i48 — %d créations · %d sans bipède contemporain ==",
		len(cre), sans)
	equipOwnerLogDist(t, "porteur le plus proche", dReal)
	equipOwnerLogDist(t, "TÉMOIN (autre bipède vivant)", dTemoin)
	equipOwnerLogTable(t, "MESURE", table)
	equipOwnerLogTable(t, "TÉMOIN", temoin)
}

func equipOwnerPut(dst map[uint32]map[int]int, id uint32, ranks map[uint32]int, slot uint32) {
	r, ok := ranks[slot]
	if !ok {
		r = -1 // rang inconnu : compté à part, jamais confondu avec un rang mesuré
	}
	if dst[id] == nil {
		dst[id] = map[int]int{}
	}
	dst[id][r]++
}

// equipOwnerNearest rend le bipède le plus proche dans la fenêtre temporelle, et un TÉMOIN :
// un AUTRE bipède vivant au même instant, choisi de façon déterministe (le i-ème de la liste
// des vivants) pour que la mesure soit rejouable à l'identique.
func equipOwnerNearest(
	pos []equipOwnerSample, c EquipmentCreation, i int,
) (near, other equipOwnerSample, ok bool) {
	lo := sort.Search(len(pos), func(k int) bool {
		return pos[k].ts+equipOwnerWindowUS >= c.TimestampUS
	})
	best := map[uint32]equipOwnerSample{} // un échantillon par slot : le plus proche en temps
	for k := lo; k < len(pos) && pos[k].ts <= c.TimestampUS+equipOwnerWindowUS; k++ {
		s := pos[k]
		if b, seen := best[s.slot]; !seen || equipOwnerGap(s.ts, c.TimestampUS) < equipOwnerGap(b.ts, c.TimestampUS) {
			best[s.slot] = s
		}
	}
	if len(best) < 2 {
		return near, other, false
	}
	slots := make([]uint32, 0, len(best))
	for s := range best {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(a, b int) bool { return slots[a] < slots[b] })
	first := true
	for _, s := range slots {
		if v := best[s]; first || equipOwnerDist(c, v) < equipOwnerDist(c, near) {
			near, first = v, false
		}
	}
	autres := make([]equipOwnerSample, 0, len(slots)-1)
	for _, s := range slots {
		if s != near.slot {
			autres = append(autres, best[s])
		}
	}
	return near, autres[i%len(autres)], true
}

func equipOwnerGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func equipOwnerDist(c EquipmentCreation, s equipOwnerSample) float64 {
	dx, dy, dz := float64(c.X-s.p[0]), float64(c.Y-s.p[1]), float64(c.Z-s.p[2])
	return dx*dx + dy*dy + dz*dz
}

func equipOwnerLogDist(t *testing.T, label string, d []float64) {
	t.Helper()
	if len(d) == 0 {
		t.Logf("   %-30s aucune mesure", label)
		return
	}
	sort.Float64s(d)
	t.Logf("   %-30s distance : médiane %.3f · p90 %.3f · max %.3f (n=%d)",
		label, math.Sqrt(d[len(d)/2]), math.Sqrt(d[(len(d)*9)/10]), math.Sqrt(d[len(d)-1]), len(d))
}

// equipOwnerLogTable publie la table identifiant x rang. La DIAGONALE se lit ici : un
// identifiant dont un seul rang concentre les observations est une capacité nommée.
func equipOwnerLogTable(t *testing.T, label string, table map[uint32]map[int]int) {
	t.Helper()
	ids := make([]uint32, 0, len(table))
	for id := range table {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	t.Logf("   -- %s : %d identifiants --", label, len(ids))
	for _, id := range ids {
		m := table[id]
		total, top, topRank := 0, 0, -2
		for r, n := range m {
			total += n
			if n > top || (n == top && r < topRank) {
				top, topRank = n, r
			}
		}
		t.Logf("      %#010x  n=%-5d rang dominant %3d (%d, %.0f %%) · %s",
			id, total, topRank, top, 100*float64(top)/float64(total), equipOwnerRanksLine(m))
	}
}

func equipOwnerRanksLine(m map[int]int) string {
	rs := make([]int, 0, len(m))
	for r := range m {
		rs = append(rs, r)
	}
	sort.Ints(rs)
	out := ""
	for _, r := range rs {
		out += fmt.Sprintf(" %d:%d", r, m[r])
	}
	return out
}
