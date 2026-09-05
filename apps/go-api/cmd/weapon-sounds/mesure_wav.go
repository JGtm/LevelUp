package main

// mesure_wav.go — mode `mesurer-wav` : la fiche spectrale d'un lot de `.wav`.
//
// POURQUOI DANS L'OUTIL ET PAS EN FFMPEG. La methode V3C choisit l'etat « en conduite » d'un
// moteur par une ECHELLE SPECTRALE MONOTONE : plus le regime monte, plus le niveau monte et
// plus le facteur de crete baisse. La mesurer demande, pour CHAQUE media de CHAQUE etat de
// CHAQUE vehicule, quatre grandeurs — soit plusieurs centaines de fichiers. En ffmpeg cela
// fait quatre processus par fichier ; ici c'est une lecture et une FFT.
//
// Grandeurs rendues, toutes en dBFS sauf indication :
//
//	duree, canaux, crete, RMS, crest (crete moins RMS, en dB)
//	bas    : RMS de la bande < 200 Hz   (le corps grave : diesel, souffle, sub)
//	haut   : RMS de la bande 3-8 kHz    (le grain mecanique : chenilles, debris)
//	centroide : barycentre du spectre, en Hz (le « brillant » du timbre)
//
// Usage : ws -etroit -mode mesurer-wav -wav <dossier> [-wem <ids decimaux>] [-out t.json]

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// ficheWav : la mesure d'un fichier.
type ficheWav struct {
	Fichier    string  `json:"fichier"`
	DureeS     float64 `json:"duree_s"`
	Canaux     int     `json:"canaux"`
	CreteDB    float64 `json:"crete_dbfs"`
	RMSDB      float64 `json:"rms_dbfs"`
	CrestDB    float64 `json:"crest_db"`
	BasDB      float64 `json:"bas_200hz_dbfs"`
	HautDB     float64 `json:"haut_3_8khz_dbfs"`
	CentroHz   float64 `json:"centroide_hz"`
	Impossible string  `json:"erreur,omitempty"`
}

// mesurerWavs est le mode `mesurer-wav`.
func mesurerWavs(dossier string, ids []uint32, tousLesFichiers bool, sortie string) error {
	if dossier == "" {
		return fmt.Errorf("le mode mesurer-wav exige -wav (dossier des .wav)")
	}
	var noms []string
	if tousLesFichiers {
		entrees, err := os.ReadDir(dossier)
		if err != nil {
			return err
		}
		for _, e := range entrees {
			if filepath.Ext(e.Name()) == ".wav" {
				noms = append(noms, e.Name())
			}
		}
		sort.Strings(noms)
	} else {
		for _, id := range ids {
			noms = append(noms, fmt.Sprintf("%d.wav", id))
		}
	}
	fmt.Printf("%-26s %7s %3s %8s %8s %7s %8s %8s %9s\n",
		"fichier", "duree", "ch", "crete", "RMS", "crest", "bas200", "h3_8k", "centroide")
	var out []ficheWav
	for _, n := range noms {
		f := mesurerUnWav(filepath.Join(dossier, n))
		out = append(out, f)
		if f.Impossible != "" {
			fmt.Printf("%-26s  %s\n", n, f.Impossible)
			continue
		}
		fmt.Printf("%-26s %7.3f %3d %+8.2f %+8.2f %7.2f %+8.2f %+8.2f %9.0f\n",
			n, f.DureeS, f.Canaux, f.CreteDB, f.RMSDB, f.CrestDB, f.BasDB, f.HautDB, f.CentroHz)
	}
	if sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, j, 0o644)
}

// mesurerUnWav rend la fiche d'un fichier.
func mesurerUnWav(chemin string) ficheWav {
	f := ficheWav{Fichier: filepath.Base(chemin)}
	a, err := lireWAV(chemin)
	if err != nil {
		f.Impossible = err.Error()
		return f
	}
	f.Canaux, f.DureeS = a.Canaux, arrondi3(float64(a.nEch())/float64(a.Taux))
	f.CreteDB, f.RMSDB = dbFini(crete(a)), dbFini(rms(a))
	f.CrestDB = arrondi3(f.CreteDB - f.RMSDB)
	mono := versMono(a)
	bas, haut, centre := spectreBandes(mono, a.Taux)
	f.BasDB, f.HautDB, f.CentroHz = dbFini(bas), dbFini(haut), math.Round(centre)
	return f
}

// dbFini ramene un niveau a une valeur FINIE : `-Inf` (silence, ou fenetre trop courte pour
// la FFT) devient -200 dB. Sans cela le rapport JSON n'est pas serialisable, et un plancher
// muet est plus honnete qu'un champ absent.
func dbFini(v float64) float64 {
	if math.IsInf(v, -1) || math.IsNaN(v) {
		return -200
	}
	return arrondi3(v)
}

// versMono replie les canaux en un seul signal (moyenne) : les mesures de bande portent sur
// le contenu, pas sur l'image stereo.
func versMono(a *audio) []float64 {
	n := a.nEch()
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		var s float64
		for c := range a.Ech {
			s += a.Ech[c][i]
		}
		out[i] = s / float64(a.Canaux)
	}
	return out
}

// spectreBandes rend, par FFT a fenetres de Hann, le RMS de la bande < 200 Hz, celui de la
// bande 3-8 kHz, et le barycentre spectral en Hz.
//
// Fenetre de 4096 points, saut de 2048. Le facteur de correction 8/3 compense la fenetre de
// Hann pour que les RMS de bande soient comparables a un RMS temporel.
func spectreBandes(x []float64, taux int) (bas, haut, centroide float64) {
	// La fenetre s'adapte au fichier : les grains du Ghost font 0,12 s (5 900 points) et
	// une fenetre fixe de 4 096 points en jetterait la moitie — ou tout, pour le prefetch
	// de 0,031 s. On prend la plus grande puissance de deux qui tient, plafonnee a 4 096.
	n := 4096
	for n > len(x) {
		n /= 2
	}
	if n < 256 || taux <= 0 {
		return math.Inf(-1), math.Inf(-1), 0
	}
	fen := make([]float64, n)
	for i := range fen {
		fen[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n))
	}
	var eBas, eHaut, eTot, mTot, mFreq float64
	trames := 0
	for d := 0; d+n <= len(x); d += n / 2 {
		re := make([]float64, n)
		im := make([]float64, n)
		for i := 0; i < n; i++ {
			re[i] = x[d+i] * fen[i]
		}
		fft(re, im)
		for k := 1; k < n/2; k++ {
			p := re[k]*re[k] + im[k]*im[k]
			f := float64(k) * float64(taux) / float64(n)
			eTot += p
			// Centroide pondere par l'AMPLITUDE (racine de la puissance) : c'est la
			// definition de `aspectralstats` de ffmpeg, celle des chiffres cites par V3B
			// (jeep 1472 Hz, chenilles 4687 Hz). Ponderer par la puissance donnerait un
			// nombre correct mais incomparable a l'etat de l'art du chantier.
			m := math.Sqrt(p)
			mTot += m
			mFreq += m * f
			if f < 200 {
				eBas += p
			}
			if f >= 3000 && f <= 8000 {
				eHaut += p
			}
		}
		trames++
	}
	if trames == 0 || eTot == 0 || mTot == 0 {
		return math.Inf(-1), math.Inf(-1), 0
	}
	// Normalisation : energie par trame, par point, corrigee de la fenetre de Hann.
	norm := float64(trames) * float64(n) * float64(n) / 8 * 3
	return 10 * math.Log10(eBas/norm*2), 10 * math.Log10(eHaut/norm*2), mFreq / mTot
}

// fft applique une transformee de Fourier rapide en place (radix 2, taille puissance de 2).
func fft(re, im []float64) {
	n := len(re)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for pas := 2; pas <= n; pas <<= 1 {
		ang := -2 * math.Pi / float64(pas)
		wr, wi := math.Cos(ang), math.Sin(ang)
		for d := 0; d < n; d += pas {
			cr, ci := 1.0, 0.0
			for k := 0; k < pas/2; k++ {
				ur, ui := re[d+k], im[d+k]
				vr := re[d+k+pas/2]*cr - im[d+k+pas/2]*ci
				vi := re[d+k+pas/2]*ci + im[d+k+pas/2]*cr
				re[d+k], im[d+k] = ur+vr, ui+vi
				re[d+k+pas/2], im[d+k+pas/2] = ur-vr, ui-vi
				cr, ci = cr*wr-ci*wi, cr*wi+ci*wr
			}
		}
	}
}
