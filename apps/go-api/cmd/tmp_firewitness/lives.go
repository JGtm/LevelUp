package main

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// Life est une vie de biped : une suite de positions du même slot sans trou majeur.
type Life struct {
	Index      int
	Slot       uint32
	Pts        []filmdec.BipedPosition
	StartUS    int64
	EndUS      int64
	Player     string // nom attribué (corrélation de la FIN de vie, puis chaînage des respawns)
	DeathNamed bool   // nommée par la corrélation des morts (source primaire)
	chained    bool   // nommée par chaînage de respawn
	Team       int
}

// lifeGapUS : au-delà de ce trou dans un même slot, on ouvre une nouvelle vie.
const lifeGapUS = 5_000_000

// buildLives regroupe les positions par slot puis par continuité temporelle.
func buildLives(pos []filmdec.BipedPosition) []Life {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var slots []uint32
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []Life
	for _, s := range slots {
		ps := bySlot[s]
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		cur := []filmdec.BipedPosition{ps[0]}
		for _, p := range ps[1:] {
			if int64(p.TimestampUS)-int64(cur[len(cur)-1].TimestampUS) > lifeGapUS {
				out = append(out, Life{Slot: s, Pts: cur})
				cur = nil
			}
			cur = append(cur, p)
		}
		out = append(out, Life{Slot: s, Pts: cur})
	}
	for i := range out {
		out[i].StartUS = int64(out[i].Pts[0].TimestampUS)
		out[i].EndUS = int64(out[i].Pts[len(out[i].Pts)-1].TimestampUS)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartUS < out[j].StartUS })
	for i := range out {
		out[i].Index = i
	}
	return out
}

// chainRespawns complète l'attribution des vies restées anonymes (leur fin n'a pas été
// appariée à une mort) par une règle CONTRAINTE, pas par la simple proximité :
// la vie L appartient à P si (a) P a une mort dans [L.début-15 s, L.début-1 s] — c'est un
// respawn — et (b) AUCUNE vie déjà attribuée à P ne chevauche L (un joueur n'a qu'un corps).
// On n'attribue que si un SEUL joueur satisfait les deux conditions.
//
// CONTRÔLE INTERNE : la même règle est appliquée aux vies DÉJÀ nommées par la corrélation
// des morts ; le taux d'accord mesure la fiabilité de la règle.
func chainRespawns(lives []Life, deaths []Death, t0, off int64) (added, agree, checked, ambiguous int) {
	const minDelayUS = 1_000_000
	const maxDelayUS = 15_000_000
	deathUS := map[string][]int64{}
	for _, d := range deaths {
		if d.LifeIndex < 0 {
			continue
		}
		deathUS[d.Victim] = append(deathUS[d.Victim], t0+(d.TimeMS+off)*1000)
	}
	candidates := func(li int) []string {
		var out []string
		for p, ts := range deathUS {
			respawn := false
			for _, t := range ts {
				if lives[li].StartUS >= t+minDelayUS && lives[li].StartUS <= t+maxDelayUS {
					respawn = true
					break
				}
			}
			if !respawn {
				continue
			}
			overlap := false
			for j := range lives {
				if j == li || lives[j].Player != p {
					continue
				}
				if lives[j].StartUS <= lives[li].EndUS && lives[li].StartUS <= lives[j].EndUS {
					overlap = true
					break
				}
			}
			if !overlap {
				out = append(out, p)
			}
		}
		sort.Strings(out)
		return out
	}
	for li := range lives {
		if lives[li].Player == "" {
			continue
		}
		c := candidates(li)
		if len(c) == 1 {
			checked++
			if c[0] == lives[li].Player {
				agree++
			}
		}
	}
	for li := range lives {
		if lives[li].Player != "" {
			continue
		}
		c := candidates(li)
		if len(c) == 1 {
			lives[li].Player = c[0]
			lives[li].chained = true
			added++
		} else if len(c) > 1 {
			ambiguous++
		}
	}
	return added, agree, checked, ambiguous
}

// PosAt renvoie la position de la vie la plus proche de tUS, et l'écart en µs.
func (l Life) PosAt(tUS int64) (filmdec.BipedPosition, int64) {
	best := l.Pts[0]
	bestD := int64(1) << 62
	for _, p := range l.Pts {
		d := int64(p.TimestampUS) - tUS
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, best = d, p
		}
	}
	return best, bestD
}

// Death est une mort lue dans killer_victim_pairs (horloge du MATCH, ms).
type Death struct {
	Killer, Victim string
	TimeMS         int64
	LifeIndex      int // vie appariée (-1 si aucune)
}

// loadDeaths lit le CSV exporté de killer_victim_pairs.
func loadDeaths(path string, names map[string]string) ([]Death, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	var out []Death
	seen := map[string]bool{} // killer_victim_pairs peut contenir des doublons (table append-only)
	for _, r := range rows[1:] {
		t, _ := strconv.ParseInt(r[col["time_ms"]], 10, 64)
		key := r[col["killer_xuid"]] + "|" + r[col["victim_xuid"]] + "|" + r[col["time_ms"]]
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Death{
			Killer: r[col["killer_gamertag"]], Victim: r[col["victim_gamertag"]],
			TimeMS: t, LifeIndex: -1,
		})
		names[r[col["killer_xuid"]]] = r[col["killer_gamertag"]]
		names[r[col["victim_xuid"]]] = r[col["victim_gamertag"]]
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out, nil
}

// Participant est une ligne de match_participants (stats de référence).
type Participant struct {
	XUID, Gamertag             string
	Team, Kills, Deaths        int
	ShotsFired, ShotsHit       int
	MeleeKills, GrenadeKills   int
	DamageDealt, PersonalScore float64
}

// loadParticipants lit le CSV exporté de match_participants, nommé via les alias trouvés
// dans killer_victim_pairs (la colonne gamertag y est vide).
func loadParticipants(path string, names map[string]string) ([]Participant, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	atoi := func(r []string, k string) int {
		v, _ := strconv.Atoi(r[col[k]])
		return v
	}
	var out []Participant
	for _, r := range rows[1:] {
		x := r[col["xuid"]]
		out = append(out, Participant{
			XUID: x, Gamertag: names[x], Team: atoi(r, "team_id"),
			Kills: atoi(r, "kills"), Deaths: atoi(r, "deaths"),
			ShotsFired: atoi(r, "shots_fired"), ShotsHit: atoi(r, "shots_hit"),
			MeleeKills: atoi(r, "melee_kills"), GrenadeKills: atoi(r, "grenade_kills"),
		})
	}
	return out, nil
}
