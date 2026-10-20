package main

// wav_io.go — lecture / ecriture WAV et operations de melange, en flottant.
//
// POURQUOI EN GO ET PAS EN FFMPEG (lot V3E). Les rendus precedents chainaient `volume`,
// `aformat` et `amix` : trois etapes qui reformatent, et une somme dont il faut se fier au
// drapeau `normalize=0`. Le premier essai du lot V3D a d'ailleurs ECRETE parce que la somme
// se faisait en entier. Ici la somme est faite une seule fois, en `float64`, avec les gains
// de chemin EXACTS du dump HIRC, et le seul traitement applique ensuite est UNE
// multiplication scalaire (la normalisation a -1 dBFS). Aucune egalisation, aucune
// compression, aucun limiteur.
//
// Formats lus : PCM entier 8/16/24/32 bits et flottant 32/64 bits, `WAVE_FORMAT_EXTENSIBLE`
// compris. Format ecrit : PCM 24 bits (les couches basses d'un melange descendent a -20 dB,
// le 16 bits y quantifierait la queue).

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// audio : un signal decode, un tableau par canal, echantillons dans [-1, 1].
type audio struct {
	Taux   int
	Canaux int
	Ech    [][]float64
}

// nEch rend le nombre d'echantillons par canal.
func (a *audio) nEch() int {
	if len(a.Ech) == 0 {
		return 0
	}
	return len(a.Ech[0])
}

// lireWAV decode un fichier WAV.
func lireWAV(chemin string) (*audio, error) {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return nil, err
	}
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, fmt.Errorf("%s : pas un RIFF/WAVE", chemin)
	}
	var format, canaux, bits int
	var taux int
	var data []byte
	for off := 12; off+8 <= len(b); {
		id := string(b[off : off+4])
		taille := int(binary.LittleEndian.Uint32(b[off+4:]))
		corps := off + 8
		if taille < 0 || corps+taille > len(b) {
			taille = len(b) - corps
		}
		switch id {
		case "fmt ":
			if taille < 16 {
				return nil, fmt.Errorf("%s : chunk fmt trop court", chemin)
			}
			format = int(binary.LittleEndian.Uint16(b[corps:]))
			canaux = int(binary.LittleEndian.Uint16(b[corps+2:]))
			taux = int(binary.LittleEndian.Uint32(b[corps+4:]))
			bits = int(binary.LittleEndian.Uint16(b[corps+14:]))
			if format == 0xFFFE && taille >= 40 {
				format = int(binary.LittleEndian.Uint16(b[corps+24:]))
			}
		case "data":
			data = b[corps : corps+taille]
		}
		off = corps + taille
		if taille%2 == 1 {
			off++
		}
	}
	if canaux <= 0 || bits <= 0 || data == nil {
		return nil, fmt.Errorf("%s : en-tete WAV incomplet", chemin)
	}
	lire, octets, err := decodeurEchantillon(format, bits)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", chemin, err)
	}
	n := len(data) / (octets * canaux)
	a := &audio{Taux: taux, Canaux: canaux, Ech: make([][]float64, canaux)}
	for c := range a.Ech {
		a.Ech[c] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for c := 0; c < canaux; c++ {
			a.Ech[c][i] = lire(data[(i*canaux+c)*octets:])
		}
	}
	return a, nil
}

// decodeurEchantillon rend le lecteur d'un echantillon et sa largeur en octets.
func decodeurEchantillon(format, bits int) (func([]byte) float64, int, error) {
	switch {
	case format == 1 && bits == 8:
		return func(p []byte) float64 { return (float64(p[0]) - 128) / 128 }, 1, nil
	case format == 1 && bits == 16:
		return func(p []byte) float64 { return float64(int16(binary.LittleEndian.Uint16(p))) / 32768 }, 2, nil
	case format == 1 && bits == 24:
		return func(p []byte) float64 {
			v := int32(p[0]) | int32(p[1])<<8 | int32(int8(p[2]))<<16
			return float64(v) / 8388608
		}, 3, nil
	case format == 1 && bits == 32:
		return func(p []byte) float64 { return float64(int32(binary.LittleEndian.Uint32(p))) / 2147483648 }, 4, nil
	case format == 3 && bits == 32:
		return func(p []byte) float64 { return float64(math.Float32frombits(binary.LittleEndian.Uint32(p))) }, 4, nil
	case format == 3 && bits == 64:
		return func(p []byte) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(p)) }, 8, nil
	}
	return nil, 0, fmt.Errorf("format WAV %d / %d bits non gere", format, bits)
}

// ecrireWAV24 ecrit un WAV PCM 24 bits. Les valeurs hors [-1, 1] sont ECRETEES, et
// l'appelant est cense avoir normalise avant : un ecretage ici est un defaut de rendu.
func ecrireWAV24(chemin string, a *audio) error {
	n := a.nEch()
	corps := make([]byte, 0, n*a.Canaux*3)
	for i := 0; i < n; i++ {
		for c := 0; c < a.Canaux; c++ {
			v := a.Ech[c][i]
			if v > 1 {
				v = 1
			} else if v < -1 {
				v = -1
			}
			q := int32(math.Round(v * 8388607))
			corps = append(corps, byte(q), byte(q>>8), byte(q>>16))
		}
	}
	blocAlign := a.Canaux * 3
	entete := make([]byte, 44)
	copy(entete[0:], "RIFF")
	binary.LittleEndian.PutUint32(entete[4:], uint32(36+len(corps)))
	copy(entete[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(entete[16:], 16)
	binary.LittleEndian.PutUint16(entete[20:], 1)
	binary.LittleEndian.PutUint16(entete[22:], uint16(a.Canaux))
	binary.LittleEndian.PutUint32(entete[24:], uint32(a.Taux))
	binary.LittleEndian.PutUint32(entete[28:], uint32(a.Taux*blocAlign))
	binary.LittleEndian.PutUint16(entete[32:], uint16(blocAlign))
	binary.LittleEndian.PutUint16(entete[34:], 24)
	copy(entete[36:], "data")
	binary.LittleEndian.PutUint32(entete[40:], uint32(len(corps)))
	return os.WriteFile(chemin, append(entete, corps...), 0o644)
}

// versStereo monte un signal mono sur deux canaux, a GAIN UNITAIRE sur chacun.
//
// CHOIX DECLARE : Wwise placerait la source en 3D et la panoramiserait ; hors du jeu, il n'y
// a ni position ni auditeur. Le gain unitaire est le meme choix que le rendu rev10 (chaine
// `aformat=channel_layouts=stereo`), ce qui rend les deux rendus comparables mesure a mesure.
// Les deux canaux sont des COPIES, jamais deux references au meme tableau : une aliasing
// ici ferait appliquer deux fois tout traitement ulterieur (mesure du 2026-09-02 : le gain
// de chemin arrivait au carre sur les couches mono, +7 dB devenaient +14 dB).
func versStereo(a *audio) *audio {
	if a.Canaux >= 2 {
		return a
	}
	return &audio{Taux: a.Taux, Canaux: 2, Ech: [][]float64{
		append([]float64(nil), a.Ech[0]...),
		append([]float64(nil), a.Ech[0]...),
	}}
}

// reechantillonner change la vitesse de lecture d'un facteur `rapport` (>1 = plus aigu et
// plus court), par interpolation LINEAIRE.
//
// C'est ce que fait un `Pitch` Wwise : il ne transpose pas a duree constante, il change la
// vitesse. L'interpolation lineaire est declaree telle quelle : sur les +-400 cents observes
// elle suffit, mais ce n'est pas un reechantillonneur a bande limitee.
func reechantillonner(a *audio, rapport float64) *audio {
	if rapport == 1 || rapport <= 0 {
		return a
	}
	n := a.nEch()
	m := int(float64(n) / rapport)
	out := &audio{Taux: a.Taux, Canaux: a.Canaux, Ech: make([][]float64, a.Canaux)}
	for c := 0; c < a.Canaux; c++ {
		out.Ech[c] = make([]float64, m)
		for i := 0; i < m; i++ {
			x := float64(i) * rapport
			j := int(x)
			f := x - float64(j)
			v0 := a.Ech[c][j]
			v1 := v0
			if j+1 < n {
				v1 = a.Ech[c][j+1]
			}
			out.Ech[c][i] = v0*(1-f) + v1*f
		}
	}
	return out
}

// crete rend le niveau de crete en dBFS.
func crete(a *audio) float64 {
	m := 0.0
	for _, canal := range a.Ech {
		for _, v := range canal {
			if av := math.Abs(v); av > m {
				m = av
			}
		}
	}
	if m == 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(m)
}

// rms rend le niveau efficace en dBFS, tous canaux confondus.
func rms(a *audio) float64 {
	var somme float64
	var n int
	for _, canal := range a.Ech {
		for _, v := range canal {
			somme += v * v
			n++
		}
	}
	if n == 0 || somme == 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(math.Sqrt(somme/float64(n)))
}

// appliquerGain multiplie le signal par un gain en dB, en place.
func appliquerGain(a *audio, db float64) {
	g := math.Pow(10, db/20)
	for c := range a.Ech {
		for i := range a.Ech[c] {
			a.Ech[c][i] *= g
		}
	}
}

// fondreBords applique un fondu d'entree et de sortie de `ms` millisecondes.
// Sert UNIQUEMENT aux boucles d'ecoute : un corps de boucle Wwise coupe net clique.
func fondreBords(a *audio, ms float64) {
	n := a.nEch()
	f := int(float64(a.Taux) * ms / 1000)
	if f <= 0 || 2*f >= n {
		return
	}
	for c := range a.Ech {
		for i := 0; i < f; i++ {
			g := float64(i) / float64(f)
			a.Ech[c][i] *= g
			a.Ech[c][n-1-i] *= g
		}
	}
}
