package temporal

import "math"

// LowessSmooth applique un lissage local pondere sur une serie 1D.
//
// Implementation legere de LOWESS (Locally Weighted Scatterplot Smoothing) :
// pour chaque point i, on prend les k plus proches voisins (window =
// floor(alpha * n), borne >= 3), on applique des poids tricube en fonction de
// la distance au point central, et on calcule la valeur lissee comme une
// regression lineaire ponderee de la fenetre.
//
// alpha : fraction du dataset utilisee comme fenetre, dans [0, 1]. Default
// 0.3 (lissage moyen, similaire au defaut Python statsmodels).
//
// Les valeurs NaN dans points sont preservees telles quelles dans la sortie
// (ne polluent pas les voisins).
//
// Pour les points avec moins de minPoints (= max(3, window/3)) voisins
// valides, la valeur lissee est NaN.
//
// Source : pas de dependance lib externe (gonum trop lourd pour ce besoin
// unique). Aligne fonctionnellement avec le LOWESS Python standard pour les
// usages de Form Score et perf timeline lisses du PLAN_SQUAD_GO_PORTAGE.
func LowessSmooth(points []float64, alpha float64) []float64 {
	n := len(points)
	if n == 0 {
		return nil
	}
	if alpha <= 0 {
		alpha = 0.3
	}
	if alpha > 1 {
		alpha = 1
	}

	window := int(math.Floor(alpha * float64(n)))
	if window < 3 {
		window = 3
	}
	if window > n {
		window = n
	}
	minValid := 3
	if window/3 > minValid {
		minValid = window / 3
	}

	out := make([]float64, n)
	for i := range out {
		if math.IsNaN(points[i]) {
			out[i] = math.NaN()
			continue
		}
		out[i] = lowessAt(points, i, window, minValid)
	}
	return out
}

// lowessAt calcule la valeur lissee au point i en utilisant une regression
// lineaire ponderee tricube sur la fenetre des voisins.
func lowessAt(points []float64, i, window, minValid int) float64 {
	n := len(points)
	// Les voisins : on prend les `window` indices les plus proches de i.
	// Comme la serie est ordonnee, ce sont les indices [low, high].
	half := window / 2
	low := i - half
	if low < 0 {
		low = 0
	}
	high := low + window - 1
	if high >= n {
		high = n - 1
		low = high - window + 1
		if low < 0 {
			low = 0
		}
	}

	// Distance max parmi les voisins valides (pour normaliser le poids tricube).
	maxDist := 0.0
	for j := low; j <= high; j++ {
		if math.IsNaN(points[j]) {
			continue
		}
		d := math.Abs(float64(j - i))
		if d > maxDist {
			maxDist = d
		}
	}
	if maxDist == 0 {
		// Tous les voisins sont au meme indice (i) ou un seul valide.
		// Compter le nombre de valides.
		var sum float64
		var count int
		for j := low; j <= high; j++ {
			if !math.IsNaN(points[j]) {
				sum += points[j]
				count++
			}
		}
		if count < minValid {
			return math.NaN()
		}
		return sum / float64(count)
	}

	// Regression lineaire ponderee : y = a + b*x.
	var sumW, sumWX, sumWY, sumWXX, sumWXY float64
	var validCount int
	for j := low; j <= high; j++ {
		if math.IsNaN(points[j]) {
			continue
		}
		validCount++
		dist := math.Abs(float64(j-i)) / maxDist
		w := tricube(dist)
		x := float64(j)
		y := points[j]
		sumW += w
		sumWX += w * x
		sumWY += w * y
		sumWXX += w * x * x
		sumWXY += w * x * y
	}
	if validCount < minValid {
		return math.NaN()
	}

	denom := sumW*sumWXX - sumWX*sumWX
	if denom == 0 {
		// Tous les x identiques (pas possible vu que j varie), fallback moyenne.
		return sumWY / sumW
	}
	b := (sumW*sumWXY - sumWX*sumWY) / denom
	a := (sumWY - b*sumWX) / sumW
	return a + b*float64(i)
}

// tricube est la fonction de poids LOWESS classique : w(d) = (1 - d^3)^3
// pour d in [0, 1], 0 ailleurs.
func tricube(d float64) float64 {
	if d >= 1 {
		return 0
	}
	t := 1 - d*d*d
	return t * t * t
}
