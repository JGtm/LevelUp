package main

// runVerifKills — LISTE À VÉRIFIER EN THEATER. Pour un film, imprime les DÉTECTIONS du décodeur
// (pas une vérité-terrain : la seule capture live 9b191a7f est FIREARM-ONLY) :
//   (a) GRENADE : kills chunk_27 dont l'arme attribuée est une grenade (nom contient "Grenade" —
//       couvre Frag/Plasma/Dynamo/Spike/Shock/Stick, exclut "Plasma Pistol") OU catégorie GRENADE
//       (event de lancer 0x4c0c00 thrower==tueur same-clock via classifyAug).
//   (b) MÊLÉE : kills dont l'arme = mêlée (Sword/Hammer/Blade/Mutilator/Diminisher/Melee) OU
//       catégorie MÊLÉE (event 0xD3 famille mêlée same-clock, |Δ|<=400ms).
//   (c) ABSENT : kills chunk_27 (non-suicide) mappables au roster dont la paire (tueur,victime) n'a
//       AUCUN kill-event décodable dans le film brut (= les ABSENT "vraiment introuvables" de runGap).
//
// Temps affiché = horloge JEU du kill chunk_27 (KVPair.TimeMS, ms relatifs au match) en mm:ss —
// l'horloge qu'affiche Theater. La catégorie/arme vient du flux frame décodé de la MÊME paire (les
// deux horloges diffèrent ; l'appariement paire<->cluster est bijectif par ordre, pas par temps).
//
// HONNÊTETÉ : aucune accuracy ici. Ces listes sont des sorties du décodeur à confirmer visuellement.

import (
	"fmt"
	"sort"
	"strconv"
)

// vkKill : un kill du kill-feed autoritatif chunk_27 (horloge jeu).
type vkKill struct {
	kx, vx uint64
	tms    int64
}

// vkGrenSubs : sous-chaînes de nom d'ARME grenade. Tous les items grenade du catalogue contiennent
// "Grenade" (Frag/Plasma/Dynamo/Spike/Shock Grenade + Grenade Stick) ; "Plasma Pistol" ne matche pas.
var vkGrenSubs = []string{"Grenade"}

// vkMeleeSubs : sous-chaînes de nom d'ARME mêlée pure (aucune arme à feu ne les contient).
var vkMeleeSubs = []string{"Sword", "Hammer", "Blade", "Mutilator", "Diminisher", "Melee"}

func vkIsGren(w string) bool  { return containsAnyKF(w, vkGrenSubs) }
func vkIsMelee(w string) bool { return containsAnyKF(w, vkMeleeSubs) }

// vkClock : ms d'horloge jeu -> "mm:ss". Négatif/aberrant -> champ brut annoté.
func vkClock(tms int64) string {
	if tms < 0 {
		return fmt.Sprintf("ts=%dms", tms)
	}
	s := tms / 1000
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func runVerifKills(m string, h32 map[uint32]string) {
	dedup, _, _ := decodePipeline(m, h32)
	kvRaw, nKills := chunk27KV(m)
	gt := chunk27Gamertags(m)
	var c27 [][2]uint64
	var kills []vkKill
	for _, p := range kvRaw {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
		kills = append(kills, vkKill{kx, vx, p.TimeMS})
	}
	rmap, overlap, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	grens, melees := scanEvents(m, h32)
	kfFullScan = false // kill-events réels (locator), pas le full-scan à FP
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		pr := [2]int{e.killer, e.victim}
		rawByPair[pr] = append(rawByPair[pr], e)
	}
	// clusters par paire, meilleure arme d'abord (consommation bijective par kill chunk_27).
	clusters := map[[2]int][]pipeKill{}
	for pr, evs := range rawByPair {
		cc := clusterBestWeapon(evs)
		sort.SliceStable(cc, func(i, j int) bool {
			if weaponScore(cc[i]) != weaponScore(cc[j]) {
				return weaponScore(cc[i]) > weaponScore(cc[j])
			}
			return cc[i].ts < cc[j].ts
		})
		clusters[pr] = cc
	}
	name := func(x uint64) string {
		if g, ok := gt[x]; ok && g != "" {
			return g
		}
		return fmt.Sprintf("xuid:%d", x)
	}
	sort.SliceStable(kills, func(i, j int) bool { return kills[i].tms < kills[j].tms })

	used := map[[2]int]int{}
	var gren, mel, absent []string
	nSelf, nUnmapped := 0, 0
	for _, k := range kills {
		if k.kx == k.vx {
			nSelf++
			continue
		}
		line := fmt.Sprintf("%s -> %s @ %s", name(k.kx), name(k.vx), vkClock(k.tms))
		ki, kok := inv[k.kx]
		vi, vok := inv[k.vx]
		if !kok || !vok {
			nUnmapped++
			continue
		}
		pr := [2]int{ki, vi}
		if len(rawByPair[pr]) == 0 {
			absent = append(absent, line)
			continue
		}
		cc := clusters[pr]
		idx := used[pr]
		used[pr]++
		if idx >= len(cc) {
			continue // paire décodable mais moins de kill-events que de kills : gap de couverture, PAS "absent"
		}
		w, cat, _ := classifyAug(cc[idx], k.kx, melees, grens, rmap)
		switch {
		case cat == "GRENADE" || vkIsGren(w):
			gren = append(gren, line+"  ["+w+"]")
		case cat == "MÊLÉE" || vkIsMelee(w):
			mel = append(mel, line+"  ["+w+"]")
		}
	}

	fmt.Printf("=== VERIFKILLS %s : chunk_27=%d roster_overlap=%d/%d self=%d unmapped=%d ===\n",
		m, nKills, overlap, nKills, nSelf, nUnmapped)
	printBlock("GRENADE", gren)
	printBlock("MÊLÉE", mel)
	printBlock("ABSENT (paire sans kill-event décodable)", absent)
	fmt.Printf("counts g/m/absent/total = %d/%d/%d/%d\n", len(gren), len(mel), len(absent), nKills)
}

func printBlock(title string, lines []string) {
	fmt.Printf("--- %s (%d) ---\n", title, len(lines))
	for _, l := range lines {
		fmt.Println("  " + l)
	}
}
