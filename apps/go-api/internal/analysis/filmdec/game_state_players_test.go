package filmdec

// game_state_players_test.go — LA LECTURE DE ti=5 : la ventilation par slot, la distribution
// des valeurs de ses onze champs, et la chasse au slot du moteur de partie quand les images-cles
// ne le portent pas. `game_state_measure_test.go` porte l en-tete, le contrat et les seuils ;
// ce fichier en est scinde pour tenir le seuil de 500 lignes.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// gamePlayerSlotProfile rend la repartition des lectures ti=5 PAR SLOT.
//
// POURQUOI CETTE VENTILATION DECIDE DU DENOMINATEUR. La bande observee de ti=5 compte 32
// slots, alors qu'une partie d'arene n'a que huit joueurs : le moteur declare des entites
// joueur pour la capacite maximale du serveur, et vingt-quatre d'entre elles ne parleront
// jamais. Comparer le debit de la bande entiere au temoin fantome divise donc le signal par
// quatre AVANT toute mesure. La ventilation par slot dit combien de slots portent
// reellement du trafic, et c'est ce nombre-la qui est le denominateur honnete.
func gamePlayerSlotProfile(t *testing.T, sc GameEntityScan) {
	t.Helper()
	if len(sc.Player) == 0 {
		return
	}
	per := map[uint32]int{}
	for _, r := range sc.Player {
		per[r.Slot]++
	}
	slots := make([]uint32, 0, len(per))
	for s := range per {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return per[slots[i]] > per[slots[j]] })
	var parts []string
	for i, s := range slots {
		if i >= 16 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", s, per[s]))
	}
	t.Logf("ti=5 VENTILATION PAR SLOT (%d slots ont rendu une lecture) : %s",
		len(per), strings.Join(parts, " "))
	nb, vd := sc.Stats[GameEntityClassNeighbour], sc.Stats[GameEntityClassVoid]
	top := 0
	for i, s := range slots {
		if i >= 8 {
			break
		}
		top += per[s]
	}
	n := len(slots)
	if n > 8 {
		n = 8
	}
	if n > 0 {
		t.Logf("ti=5 DEBIT DES 8 SLOTS LES PLUS BAVARDS : %.1f lectures/slot contre %.1f "+
			"(voisinage) et %.1f (vide) -> x%.2f et x%.2f", float64(top)/float64(n),
			gameWantedPerSlot(nb), gameWantedPerSlot(vd),
			gameRatio(float64(top)/float64(n), gameWantedPerSlot(nb)),
			gameRatio(float64(top)/float64(n), gameWantedPerSlot(vd)))
	}
}

// gamePlayerStateValues rend, pour chaque champ de ti=5, la distribution de ses valeurs.
// C'est la matiere de P.0.5 : un champ dont toutes les lectures rendent la meme valeur ne
// porte aucun etat, et un champ dont les valeurs sont uniformement etalees sur son domaine
// est du bruit — les deux se voient ici et nulle part ailleurs.
func gamePlayerStateValues(t *testing.T, sc GameEntityScan) {
	t.Helper()
	if len(sc.Player) == 0 {
		return
	}
	t.Logf("ti=5 DISTRIBUTION DES VALEURS (P.0.5)")
	for f := 0; f < PlayerStateFieldCount; f++ {
		fl := PlayerStateField(f)
		hist, n, gated := map[string]int{}, 0, 0
		for _, r := range sc.Player {
			if !r.PlayerSeen[fl] {
				continue
			}
			n++
			if !r.PlayerPresent[fl] {
				gated++
				continue
			}
			hist[gamePlayerVals(r, fl)]++
		}
		if n == 0 {
			t.Logf("    %-48s AUCUNE lecture", fl)
			continue
		}
		t.Logf("    %-48s lectures %5d · porte fermee %5d · valeurs distinctes %4d · %s",
			fl, n, gated, len(hist), gameTopValues(hist, 6))
	}
}

// gameTopValues rend les k valeurs les plus frequentes d'un histogramme.
func gameTopValues(hist map[string]int, k int) string {
	type kv struct {
		v string
		n int
	}
	rows := make([]kv, 0, len(hist))
	for v, n := range hist {
		rows = append(rows, kv{v, n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].v < rows[j].v
	})
	var parts []string
	for i, r := range rows {
		if i >= k {
			break
		}
		v := r.v
		if len(v) > 40 {
			v = v[:40] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s(%d)", v, r.n))
	}
	return strings.Join(parts, " ")
}

// gameHuntEngineSlot cherche le slot du moteur de partie quand les images-cles ne le portent
// pas. LE CRITERE EST LA GRAMMAIRE, PAS LES VALEURS : on compte par slot les en-tetes dont le
// masque tient dans les 27 composants de ti=0 et annonce le round-timer. Chercher un slot par
// ce que ses valeurs devraient valoir construirait le resultat.
func gameHuntEngineSlot(t *testing.T, dir string, sc GameEntityScan) {
	t.Helper()
	if len(sc.Bands[GameEngineTypeIndex]) > 0 {
		return
	}
	t.Logf("CHASSE ti=0 : la bande est VIDE (aucun slot d'archetype 0 dans les images-cles) "+
		"— recherche du slot par la grammaire de l'archetype et l'annonce de %s",
		compGameEngineRoundTimer)
	rows, err := HuntArchetypeSlots(dir, GameEngineTypeIndex, compGameEngineRoundTimer)
	if err != nil {
		t.Logf("CHASSE ti=0 impossible : %v", err)
		return
	}
	tot, mid := 0, 0.0
	vals := make([]int, 0, len(rows))
	for _, r := range rows {
		tot += r.WithMust
		vals = append(vals, r.WithMust)
	}
	sort.Ints(vals)
	if len(vals) > 0 {
		mid = float64(vals[len(vals)/2])
	}
	t.Logf("CHASSE ti=0 : %d slots ont porte au moins un en-tete · total annonces i5 %d "+
		"· mediane par slot %.1f", len(rows), tot, mid)
	for i, r := range rows {
		if i >= 10 {
			break
		}
		t.Logf("    slot %5d · en-tetes %6d · dans la grammaire %6d · annoncent i5 %6d",
			r.Slot, r.Records, r.InGrammar, r.WithMust)
	}
	if os.Getenv(gameClockHuntEnv) == "" {
		t.Logf("CHASSE HORLOGE non demandee (%s absent) : la passe qui marche tous les slots "+
			"coute ~50 s par film et n'a rien rendu sur les films deja mesures", gameClockHuntEnv)
		return
	}
	gameHuntClock(t, dir)
}

// gameHuntClock cherche le slot de l'horloge par sa SIGNATURE : une valeur qui decroit de
// facon monotone sur toute la partie. Le critere d'identification ne fixe ni la pente, ni la
// valeur de depart, ni l'instant de depart — les trois grandeurs que B.0.2 mesure ensuite
// contre des oracles exterieurs.
func gameHuntClock(t *testing.T, dir string) {
	t.Helper()
	cands, err := HuntGameEngineClock(dir)
	if err != nil {
		t.Logf("CHASSE HORLOGE impossible : %v", err)
		return
	}
	t.Logf("CHASSE HORLOGE : %d slots ont rendu au moins une lecture d'horloge", len(cands))
	shown := 0
	for _, c := range cands {
		if shown >= 15 {
			break
		}
		shown++
		mono := 0.0
		if c.Down+c.Up > 0 {
			mono = float64(c.Down) / float64(c.Down+c.Up)
		}
		t.Logf("    slot %5d · lectures %6d · decroit %5d / croit %5d (monotonie %.2f) "+
			"· A [%.1f , %.1f] s · pente %.3f s/s · fenetre %.1f s",
			c.Slot, c.Samples, c.Down, c.Up, mono, c.MinA, c.MaxA, c.Slope,
			float64(c.LastUS-c.FirstUS)/1e6)
	}
}
