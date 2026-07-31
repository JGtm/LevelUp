package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// bestAssignment renvoie l'affectation index -> joueur de COÛT TOTAL MINIMAL (énumération
// exhaustive des permutations : 8! = 40320, exact, sans dépendance d'ordre — le glouton,
// lui, dépend de l'ordre d'itération quand il y a des ex aequo).
// Renvoie aussi le coût du meilleur et celui du DEUXIÈME meilleur (marge = puissance du test).
func bestAssignment(idxs []int, players []string, cost map[int]map[string]float64) (map[int]string, float64, float64) {
	n := len(players)
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	best, second := math.Inf(1), math.Inf(1)
	var bestPerm []int
	var rec func(k int)
	rec = func(k int) {
		if k == n {
			c := 0.0
			for i := 0; i < n && i < len(idxs); i++ {
				c += cost[idxs[i]][players[perm[i]]]
			}
			if c < best {
				second = best
				best = c
				bestPerm = append([]int(nil), perm...)
			} else if c < second {
				second = c
			}
			return
		}
		for i := k; i < n; i++ {
			perm[k], perm[i] = perm[i], perm[k]
			rec(k + 1)
			perm[k], perm[i] = perm[i], perm[k]
		}
	}
	rec(0)
	out := map[int]string{}
	for i := 0; i < len(idxs) && i < n; i++ {
		out[idxs[i]] = players[bestPerm[i]]
	}
	return out, best, second
}

// geometryMatrix — matrice « si l'index i était le joueur p, la visée décodée pointerait-elle
// vers quelqu'un ? » : médiane, sur les events de i porteurs d'une visée, de l'angle au joueur
// le plus proche angulairement (hors p). Un record type 105 est une APPLICATION DE DÉGÂT :
// avec la bonne origine, le rayon doit passer près d'un autre joueur.
func geometryMatrix(lives []Life, nameToLives map[string][]int, longs []FireEvent,
	idxs []int, players []string) map[int]map[string]float64 {
	out := map[int]map[string]float64{}
	for _, i := range idxs {
		out[i] = map[string]float64{}
		for _, p := range players {
			var angles []float64
			for _, e := range longs {
				if !e.HasAim || e.PlayerIdx != i {
					continue
				}
				sp, ok := posOfPlayerAt(lives, nameToLives, p, int64(e.TS))
				if !ok {
					continue
				}
				aim := [3]float64{float64(e.Aim[0]), float64(e.Aim[1]), float64(e.Aim[2])}
				bestA := math.Inf(1)
				for other := range nameToLives {
					if other == p {
						continue
					}
					q, ok := posOfPlayerAt(lives, nameToLives, other, int64(e.TS))
					if !ok {
						continue
					}
					bestA = math.Min(bestA, angleDeg(aim,
						[3]float64{float64(q.X - sp.X), float64(q.Y - sp.Y), float64(q.Z-sp.Z) - eyeHeight}))
				}
				if !math.IsInf(bestA, 1) {
					angles = append(angles, bestA)
				}
			}
			sort.Float64s(angles)
			if len(angles) == 0 {
				out[i][p] = 90 // aucune donnée : coût neutre élevé
				continue
			}
			out[i][p] = median(angles)
		}
	}
	return out
}

// printMatrix affiche une matrice index x joueur.
func printMatrix(title string, idxs []int, players []string, m map[int]map[string]float64, format string) {
	fmt.Println(title)
	fmt.Printf("%-6s", "index")
	for _, p := range players {
		fmt.Printf(" %-9.9s", p)
	}
	fmt.Println()
	for _, i := range idxs {
		fmt.Printf("%-6d", i)
		for _, p := range players {
			fmt.Printf(" "+format, m[i][p])
		}
		fmt.Println()
	}
}

// randomAimControl — TÉMOIN NUL du témoin 1 bis : mêmes tireurs, mêmes instants, mais des
// visées ISOTROPES tirées au hasard. Donne la distribution attendue « sans information ».
func randomAimControl(lives []Life, nameToLives map[string][]int, longs []FireEvent,
	idxToName map[int]string, teams map[string]int) []float64 {
	r := rand.New(rand.NewSource(42))
	var out []float64
	for _, e := range longs {
		if !e.HasAim {
			continue
		}
		shooter := idxToName[e.PlayerIdx]
		sp, ok := posOfPlayerAt(lives, nameToLives, shooter, int64(e.TS))
		if !ok {
			continue
		}
		aim := randomUnit(r)
		best := math.Inf(1)
		for name := range nameToLives {
			if name == shooter || teams[name] == teams[shooter] {
				continue
			}
			q, ok := posOfPlayerAt(lives, nameToLives, name, int64(e.TS))
			if !ok {
				continue
			}
			best = math.Min(best, angleDeg(aim,
				[3]float64{float64(q.X - sp.X), float64(q.Y - sp.Y), float64(q.Z - sp.Z)}))
		}
		if !math.IsInf(best, 1) {
			out = append(out, best)
		}
	}
	sort.Float64s(out)
	return out
}

// randomUnit tire un vecteur unitaire isotrope.
func randomUnit(r *rand.Rand) [3]float64 {
	z := 2*r.Float64() - 1
	t := 2 * math.Pi * r.Float64()
	s := math.Sqrt(1 - z*z)
	return [3]float64{s * math.Cos(t), s * math.Sin(t), z}
}

// medianEnemyAt calcule la médiane de l'angle au plus proche ennemi pour une hauteur d'oeil
// donnée : sert à mesurer le biais vertical (position d'entité vs origine du tir).
func medianEnemyAt(lives []Life, nameToLives map[string][]int, longs []FireEvent,
	idxToName map[int]string, teams map[string]int, h float64) float64 {
	var out []float64
	for _, e := range longs {
		if !e.HasAim {
			continue
		}
		shooter := idxToName[e.PlayerIdx]
		sp, ok := posOfPlayerAt(lives, nameToLives, shooter, int64(e.TS))
		if !ok {
			continue
		}
		aim := [3]float64{float64(e.Aim[0]), float64(e.Aim[1]), float64(e.Aim[2])}
		best := math.Inf(1)
		for name := range nameToLives {
			if name == shooter || teams[name] == teams[shooter] {
				continue
			}
			q, ok := posOfPlayerAt(lives, nameToLives, name, int64(e.TS))
			if !ok {
				continue
			}
			best = math.Min(best, angleDeg(aim,
				[3]float64{float64(q.X - sp.X), float64(q.Y - sp.Y), float64(q.Z-sp.Z) - h}))
		}
		if !math.IsInf(best, 1) {
			out = append(out, best)
		}
	}
	sort.Float64s(out)
	return median(out)
}

// lifePurity — TEST INTERNE AU FILM (aucune base) : chaque VIE (slot biped) ne doit contenir
// que les events de tir d'UN SEUL index de joueur. Si c'est vrai, l'attribution
// vie -> index de tir se lit dans le film seul, et le rejeu peut placer les tirs sans DB.
func lifePurity(lives []Life, longs []FireEvent, idxToName map[int]string) {
	fmt.Printf("\n=== TEST FILM-SEUL : une vie ne contient les tirs que d'un seul index ===\n")
	cnt := make([]map[int]int, len(lives))
	for i := range cnt {
		cnt[i] = map[int]int{}
	}
	placed, orphan := 0, 0
	for _, e := range longs {
		hit := -1
		for i, l := range lives {
			if int64(e.TS) >= l.StartUS && int64(e.TS) <= l.EndUS {
				if hit >= 0 {
					hit = -2 // plusieurs vies couvrent l'instant : ambigu
					break
				}
				hit = i
			}
		}
		if hit >= 0 {
			cnt[hit][e.PlayerIdx]++
			placed++
		} else {
			orphan++
		}
	}
	pure, impure, tot, dom := 0, 0, 0, 0
	agreeDB, checkedDB := 0, 0
	for i, m := range cnt {
		if len(m) == 0 {
			continue
		}
		tot++
		best, bestN, sum := -1, 0, 0
		for k, v := range m {
			sum += v
			if v > bestN {
				bestN, best = v, k
			}
		}
		dom += bestN
		if len(m) == 1 {
			pure++
		} else {
			impure++
		}
		if lives[i].Player != "" {
			checkedDB++
			if idxToName[best] == lives[i].Player {
				agreeDB++
			}
		}
	}
	fmt.Printf("events placés dans une vie : %d ; orphelins (aucune vie ne couvre l'instant) : %d\n", placed, orphan)
	fmt.Printf("vies porteuses d'au moins un tir : %d — PURES (un seul index) : %d ; mixtes : %d\n", tot, pure, impure)
	fmt.Printf("part des events dans l'index dominant de leur vie : %.1f%%\n", 100*float64(dom)/float64(maxInt(placed, 1)))
	fmt.Printf("CONTROLE croisé avec la base : l'index dominant de la vie == le joueur nommé par les morts sur %d/%d vies\n",
		agreeDB, checkedDB)
}

// selfAimLink — LIEN FILM-SEUL entre l'event de tir et le BIPED tireur : la visée du record
// (cubemap 30 bits) doit coïncider avec le vecteur de visée du biped (composant i21, cap
// quantifié 12 bits) au MÊME instant. Deux champs différents, deux largeurs différentes,
// deux composants différents : la coïncidence n'a aucune raison d'exister si le décodage ou
// l'attribution sont faux.
//
// Renvoie le taux d'accord avec l'attribution issue de la base (contrôle) et la distribution
// des écarts de cap.
func selfAimLink(lives []Life, longs []FireEvent, idxToName map[int]string) {
	fmt.Printf("\n=== LIEN FILM-SEUL : visée du tir vs cap de visée du biped (i21) ===\n")
	var best, second []float64
	agree, checked, resolved, unambig := 0, 0, 0, 0
	for _, e := range longs {
		if !e.HasAim {
			continue
		}
		fireHead := math.Atan2(float64(e.Aim[1]), float64(e.Aim[0])) * 180 / math.Pi
		type cand struct {
			li   int
			diff float64
		}
		var cs []cand
		for li, l := range lives {
			if int64(e.TS) < l.StartUS || int64(e.TS) > l.EndUS {
				continue
			}
			p, d := l.PosAt(int64(e.TS))
			if d > 120_000 {
				continue
			}
			h, ok := p.AimHeadingDeg()
			if !ok {
				continue
			}
			cs = append(cs, cand{li, angDiff(fireHead, float64(h))})
		}
		if len(cs) < 2 {
			continue
		}
		sort.Slice(cs, func(a, b int) bool { return cs[a].diff < cs[b].diff })
		best = append(best, cs[0].diff)
		second = append(second, cs[1].diff)
		resolved++
		if cs[0].diff > 2 || cs[1].diff < 20 {
			continue // désignation ambiguë : on n'attribue pas
		}
		unambig++
		if lives[cs[0].li].Player != "" {
			checked++
			if lives[cs[0].li].Player == idxToName[e.PlayerIdx] {
				agree++
			}
		}
	}
	sort.Float64s(best)
	sort.Float64s(second)
	fmt.Printf("events résolus : %d ; écart de cap au MEILLEUR biped : médiane %.1f deg (p75 %.1f)\n",
		resolved, median(best), quantile(best, 0.75))
	fmt.Printf("TEMOIN : écart au DEUXIÈME meilleur biped : médiane %.1f deg\n", median(second))
	fmt.Printf("le biped ainsi désigné est bien celui de l'index de tir (attribution base) : %d/%d = %.0f%%\n",
		agree, checked, 100*float64(agree)/float64(maxInt(checked, 1)))
}

// angDiff renvoie l'écart angulaire absolu entre deux caps en degrés, dans [0,180].
func angDiff(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// voteLifeIndex — ATTRIBUTION FILM-SEULE des vies aux index de tir : chaque désignation non
// ambiguë (visée du tir == cap i21 d'un seul biped) est un VOTE (vie -> index). Une vie
// prend l'index majoritaire. Ensuite, TOUS les events d'un index tombant dans l'intervalle
// d'une de ses vies héritent de cette origine — d'où une couverture bien supérieure au seul
// sous-ensemble non ambigu.
func voteLifeIndex(lives []Life, longs []FireEvent, idxToName map[int]string) {
	fmt.Printf("\n=== ATTRIBUTION FILM-SEULE vie -> index de tir (vote par la visée i21) ===\n")
	votes := make([]map[int]int, len(lives))
	for i := range votes {
		votes[i] = map[int]int{}
	}
	for _, e := range longs {
		if li, ok := designateShooterLife(lives, e); ok {
			votes[li][e.PlayerIdx]++
		}
	}
	lifeIdx := make([]int, len(lives))
	mixed, voted := 0, 0
	for i, m := range votes {
		lifeIdx[i] = -1
		if len(m) == 0 {
			continue
		}
		voted++
		if len(m) > 1 {
			mixed++
		}
		best, bn := -1, 0
		for k, v := range m {
			if v > bn {
				bn, best = v, k
			}
		}
		lifeIdx[i] = best
	}
	fmt.Printf("vies ayant reçu au moins un vote : %d/%d ; vies à votes MIXTES (2 index ou plus) : %d\n",
		voted, len(lives), mixed)
	// couverture : combien d'events peuvent être placés
	placed, conflict := 0, 0
	agreeDB, checkDB := 0, 0
	for _, e := range longs {
		var cands []int
		for li, l := range lives {
			if lifeIdx[li] == e.PlayerIdx && int64(e.TS) >= l.StartUS && int64(e.TS) <= l.EndUS {
				cands = append(cands, li)
			}
		}
		if len(cands) == 1 {
			placed++
			if lives[cands[0]].Player != "" {
				checkDB++
				if lives[cands[0]].Player == idxToName[e.PlayerIdx] {
					agreeDB++
				}
			}
		} else if len(cands) > 1 {
			conflict++
		}
	}
	fmt.Printf("events plaçables (une seule vie de leur index couvre l'instant) : %d/%d = %.0f%% ; conflits : %d\n",
		placed, len(longs), 100*float64(placed)/float64(maxInt(len(longs), 1)), conflict)
	fmt.Printf("CONTROLE : la vie retenue porte le joueur attendu par la base sur %d/%d = %.0f%%\n",
		agreeDB, checkDB, 100*float64(agreeDB)/float64(maxInt(checkDB, 1)))
	// accord par index avec l'attribution base
	perIdx := map[int]map[string]int{}
	for li, l := range lives {
		if lifeIdx[li] < 0 || l.Player == "" {
			continue
		}
		if perIdx[lifeIdx[li]] == nil {
			perIdx[lifeIdx[li]] = map[string]int{}
		}
		perIdx[lifeIdx[li]][l.Player]++
	}
	var ks []int
	for k := range perIdx {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	fmt.Println("index -> joueurs des vies votées (base) :")
	for _, k := range ks {
		fmt.Printf("  index %d (base dit %-16s) : %v\n", k, idxToName[k], perIdx[k])
	}
}

// designateShooterLife renvoie la vie du tireur quand la visée du record désigne UN SEUL
// biped sans ambiguïté (meilleur écart de cap < 2 deg, deuxième > 20 deg).
func designateShooterLife(lives []Life, e FireEvent) (int, bool) {
	if !e.HasAim {
		return 0, false
	}
	fireHead := math.Atan2(float64(e.Aim[1]), float64(e.Aim[0])) * 180 / math.Pi
	bestLi, bestD, secondD := -1, 1e9, 1e9
	for li, l := range lives {
		if int64(e.TS) < l.StartUS || int64(e.TS) > l.EndUS {
			continue
		}
		p, d := l.PosAt(int64(e.TS))
		if d > 120_000 {
			continue
		}
		h, ok := p.AimHeadingDeg()
		if !ok {
			continue
		}
		diff := angDiff(fireHead, float64(h))
		if diff < bestD {
			secondD, bestD, bestLi = bestD, diff, li
		} else if diff < secondD {
			secondD = diff
		}
	}
	if bestLi < 0 || bestD > 2 || secondD < 20 {
		return 0, false
	}
	return bestLi, true
}
