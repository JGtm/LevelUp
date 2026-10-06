package trends

import "math"

// meanOf retourne la moyenne de vals ; 0 si vals est vide.
func meanOf(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var s float64
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

// stdPop retourne l'écart-type de population de vals autour de mean ; 0 si vals
// est vide.
func stdPop(vals []float64, mean float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var s float64
	for _, v := range vals {
		d := v - mean
		s += d * d
	}
	return math.Sqrt(s / float64(len(vals)))
}

// sdOrOne remplace un écart-type nul par 1 : une dispersion nulle ne doit ni
// diviser par zéro ni amplifier un écart.
func sdOrOne(sd float64) float64 {
	if sd == 0 || math.IsNaN(sd) || math.IsInf(sd, 0) {
		return 1
	}
	return sd
}

// pearson retourne la corrélation de Pearson entre xs et ys (même longueur) ;
// false si elle n'est pas définie (moins de 2 points ou une série constante).
func pearson(xs, ys []float64) (float64, bool) {
	n := len(xs)
	if n < 2 || n != len(ys) {
		return 0, false
	}
	mx, my := meanOf(xs), meanOf(ys)
	var sxy, sxx, syy float64
	for i := range xs {
		dx, dy := xs[i]-mx, ys[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return 0, false
	}
	return sxy / math.Sqrt(sxx*syy), true
}
