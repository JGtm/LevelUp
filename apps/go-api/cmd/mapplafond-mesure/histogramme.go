package main

// histogramme.go — L'HISTOGRAMME D'ALTITUDE, AU CENTIMETRE.
//
// POURQUOI PAS UN TABLEAU DE VALEURS. Le corpus de rejeux porte des centaines de milliers de
// positions et une carte cuite jusqu'a quatre millions de pixels : tout garder en memoire pour
// en tirer trois centiles coute cher sans rien apprendre de plus (lecon « balayage corpus =
// bombe RAM », `.ai/` registre). Un histogramme au CENTIMETRE rend les memes centiles a la
// precision de la source — les artefacts de rejeu publient deux decimales (cf.
// `replay.Point`) — pour quelques milliers d'entrees quelle que soit la taille du corpus.

import (
	"math"
	"sort"
)

// histogramme compte des altitudes par bin d'un centimetre.
type histogramme struct {
	bins map[int]int
	n    int
	// cles est le cache des bins TRIES. Invalide (nil) des qu'une valeur entre.
	cles []int
}

func nouvelHistogramme() *histogramme {
	return &histogramme{bins: map[int]int{}}
}

// ajoute compte une altitude. Une valeur non finie est REFUSEE plutot que rangee dans un bin
// aberrant : une position NaN dans le flux ne doit pas deplacer un centile.
func (h *histogramme) ajoute(z float64) {
	if math.IsNaN(z) || math.IsInf(z, 0) {
		return
	}
	h.bins[int(math.Round(z*100))]++
	h.n++
	h.cles = nil
}

func (h *histogramme) taille() int { return h.n }

// triees rend les bins occupes, en ordre croissant.
func (h *histogramme) triees() []int {
	if h.cles != nil {
		return h.cles
	}
	c := make([]int, 0, len(h.bins))
	for k := range h.bins {
		c = append(c, k)
	}
	sort.Ints(c)
	h.cles = c
	return c
}

// centile rend le centile p (0..1) de l'echantillon. NaN sur un echantillon vide : un zero
// silencieux se confondrait avec une mesure (meme regle que `himap.Centile`).
func (h *histogramme) centile(p float64) float64 {
	if h.n == 0 {
		return math.NaN()
	}
	rang := int(p * float64(h.n-1))
	cumul := 0
	for _, k := range h.triees() {
		cumul += h.bins[k]
		if cumul > rang {
			return float64(k) / 100
		}
	}
	return float64(h.triees()[len(h.triees())-1]) / 100
}

// maximum rend la plus haute altitude comptee, NaN si vide.
func (h *histogramme) maximum() float64 {
	if h.n == 0 {
		return math.NaN()
	}
	c := h.triees()
	return float64(c[len(c)-1]) / 100
}

// minimum rend la plus basse altitude comptee, NaN si vide.
func (h *histogramme) minimum() float64 {
	if h.n == 0 {
		return math.NaN()
	}
	return float64(h.triees()[0]) / 100
}

// nombreAuDessus compte les valeurs STRICTEMENT au-dessus d'un seuil.
func (h *histogramme) nombreAuDessus(seuil float64) int {
	n := 0
	for _, k := range h.triees() {
		if float64(k)/100 > seuil {
			n += h.bins[k]
		}
	}
	return n
}

// partAuDessus rend la part (0..1) des valeurs strictement au-dessus d'un seuil. NaN si vide,
// pour la meme raison que `centile`.
func (h *histogramme) partAuDessus(seuil float64) float64 {
	if h.n == 0 {
		return math.NaN()
	}
	return float64(h.nombreAuDessus(seuil)) / float64(h.n)
}
