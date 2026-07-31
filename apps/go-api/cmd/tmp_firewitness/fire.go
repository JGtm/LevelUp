package main

import (
	"math"

	"levelup/go-api/internal/analysis/filmdec"
)

// FireEvent — record type 105 décodé (tête seulement, cf. spec Ghidra).
type FireEvent struct {
	Chunk, Pkt int
	TS         uint64 // µs, horloge du film (même horloge que les positions)
	Variant    int    // 0 = long (0xD2), 1 = court (0xD3)
	PlayerIdx  int    // bits 36..40 >> 1
	Weapon64   uint64
	F          [5]uint32 // bits 108..112
	HasAim     bool
	AimCode    uint32
	Aim        [3]float32
}

const fireEventType = 105

// scanFireEvents décode tous les records type 105 des paquets type-0 du film.
func scanFireEvents(dir string) []FireEvent {
	var out []FireEvent
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != fireEventType {
				continue
			}
			e := FireEvent{Chunk: c, Pkt: p.Index, TS: p.TimestampUS, Variant: int(pay[0] & 1)}
			e.PlayerIdx = int(filmdec.ReadBitsAtForDiag(pay, 36, 5)) >> 1
			e.Weapon64 = uint64(filmdec.ReadBitsAtForDiag(pay, 44, 32))<<32 |
				uint64(filmdec.ReadBitsAtForDiag(pay, 76, 32))
			for i := 0; i < 5; i++ {
				e.F[i] = filmdec.ReadBitsAtForDiag(pay, 108+i, 1)
			}
			if e.Variant == 0 && e.F[2] == 1 && e.F[3] == 0 && e.F[4] == 0 && p.Size*8 >= 143 {
				e.AimCode = filmdec.ReadBitsAtForDiag(pay, 113, 30)
				v, ok := filmdec.DecodeAimVectorChecked(e.AimCode, 30)
				if ok {
					e.HasAim, e.Aim = true, v
				}
			}
			out = append(out, e)
		}
	}
	return out
}

// angleDeg renvoie l'angle en degrés entre deux vecteurs 3D.
func angleDeg(a, b [3]float64) float64 {
	na := math.Sqrt(a[0]*a[0] + a[1]*a[1] + a[2]*a[2])
	nb := math.Sqrt(b[0]*b[0] + b[1]*b[1] + b[2]*b[2])
	if na == 0 || nb == 0 {
		return math.NaN()
	}
	d := (a[0]*b[0] + a[1]*b[1] + a[2]*b[2]) / (na * nb)
	if d > 1 {
		d = 1
	}
	if d < -1 {
		d = -1
	}
	return math.Acos(d) * 180 / math.Pi
}

// median renvoie la médiane d'un échantillon (copie triée par l'appelant).
func median(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	return quantile(v, 0.5)
}

// quantile renvoie le quantile q d'un échantillon DÉJÀ TRIÉ.
func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	i := int(q * float64(len(sorted)-1))
	return sorted[i]
}
