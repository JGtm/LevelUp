package main

import (
	"bufio"
	"encoding/csv"
	"math"
	"os"
	"sort"
	"strconv"
)

// maxVZ : seuil de vitesse verticale sous lequel on considère le joueur POSÉ.
const maxVZ = 0.15

// tsPerSecond : l'horodatage du film est en microsecondes (écart typique entre deux
// paquets consécutifs ~1,6e4, soit 16 ms = une trame à 60 Hz).
const tsPerSecond = 1e6

type sample struct {
	slot    int
	ts      float64
	x, y, z float64
}

// groundedOnly renvoie les positions dont la vitesse verticale, estimée par différence
// centrée sur la trajectoire du MÊME joueur, est quasi nulle. Le critère ne regarde
// jamais la géométrie : il ne peut donc pas favoriser la vraie carte face aux témoins.
func groundedOnly(path string) [][3]float64 {
	rows := readSamples(path)
	bySlot := map[int][]sample{}
	for _, s := range rows {
		bySlot[s.slot] = append(bySlot[s.slot], s)
	}
	var out [][3]float64
	for _, ss := range bySlot {
		sort.Slice(ss, func(i, j int) bool { return ss[i].ts < ss[j].ts })
		for i := 1; i+1 < len(ss); i++ {
			dt := (ss[i+1].ts - ss[i-1].ts) / tsPerSecond
			if dt <= 0 || dt > 0.2 {
				continue
			}
			if math.Abs((ss[i+1].z-ss[i-1].z)/dt) <= maxVZ {
				out = append(out, [3]float64{ss[i].x, ss[i].y, ss[i].z})
			}
		}
	}
	return out
}

func readSamples(path string) []sample {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return nil
	}
	var out []sample
	for i, rec := range recs {
		if i == 0 || len(rec) < 7 {
			continue
		}
		var s sample
		var e error
		if s.slot, e = strconv.Atoi(rec[0]); e != nil {
			continue
		}
		if s.ts, e = strconv.ParseFloat(rec[3], 64); e != nil {
			continue
		}
		v := [3]*float64{&s.x, &s.y, &s.z}
		bad := false
		for a := 0; a < 3; a++ {
			if *v[a], e = strconv.ParseFloat(rec[4+a], 64); e != nil {
				bad = true
				break
			}
		}
		if !bad {
			out = append(out, s)
		}
	}
	return out
}
