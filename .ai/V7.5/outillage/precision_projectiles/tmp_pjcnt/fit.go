package main

// fit.go — DECONVOLUTION DU TAUX DE TOUCHE PAR ARME.
//
// L IDEE, ET POURQUOI ELLE EST NEUVE. Le film donne, par match et par arme, un compte de
// TIRS (les records type 105 — mesure de cadence a l appui : un record par cycle d arme,
// projectiles compris). Il ne donne le compte de TOUCHES que pour les armes a trace
// instantanee (le drapeau porteur vaut ~0 sur les projectiles). L API, elle, donne le total
// de touches du MATCH, toutes armes confondues. On a donc, par match m :
//
//	touches_API(m)  =  somme sur les armes W de  p_W * tirs_W(m)
//
// avec `tirs_W(m)` connu et `p_W` inconnu. Des centaines de matchs a melanges d armes
// differents donnent un systeme surdetermine : les p_W se resolvent aux moindres carres.
//
// LE CONTROLE POSITIF EST INTEGRE, et c est ce qui rend la mesure interpretable : les armes
// a trace instantanee ont deja un taux de touche mesurable ligne a ligne (porteurs/records).
// Si la deconvolution les retrouve, ses estimations pour les armes a projectile deviennent
// lisibles ; si elle les rate, elle ne mesure rien et le dit.
//
// CE QUE CE FICHIER N EST PAS : une productionisation. C est un instrument de verdict.

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// designRow est une ligne du systeme : un match.
type designRow struct {
	pfx      string
	famille  string
	apiHits  float64
	apiFired float64
	// tirs par arme (records type 105 du film)
	shots map[uint64]float64
	// porteurs par arme (touches lisibles, ~0 sur les armes a projectile)
	carriers map[uint64]float64
}

// fitResult porte l estimation d une arme.
type fitResult struct {
	id        uint64
	name      string
	shots     float64
	carriers  float64
	tauxPort  float64 // porteurs / records — mesurable seulement sur trace instantanee
	coefA     float64 // coefficient estime sur la moitie A
	coefB     float64 // coefficient estime sur la moitie B
	coefTotal float64 // coefficient estime sur tout
}

// solveHitRates ajuste les p_W par moindres carres (equations normales + ridge) sur les
// armes portant assez de tirs, et rend l estimation globale plus les deux moities.
func solveHitRates(rows []designRow, names map[uint64]string, minShots float64, ridge float64) ([]fitResult, float64, float64) {
	ids := retainedWeapons(rows, minShots)
	if len(ids) == 0 {
		return nil, 0, 0
	}
	all := fitOn(rows, ids, ridge)
	var a, b []designRow
	for i, r := range rows {
		if i%2 == 0 {
			a = append(a, r)
		} else {
			b = append(b, r)
		}
	}
	ca := fitOn(a, ids, ridge)
	cb := fitOn(b, ids, ridge)

	totShots := map[uint64]float64{}
	totCarr := map[uint64]float64{}
	for _, r := range rows {
		for id, v := range r.shots {
			totShots[id] += v
		}
		for id, v := range r.carriers {
			totCarr[id] += v
		}
	}
	out := make([]fitResult, 0, len(ids))
	for i, id := range ids {
		name := names[id]
		if name == "" {
			name = fmt.Sprintf("0x%016x", id)
		}
		tp := 0.0
		if totShots[id] > 0 {
			tp = totCarr[id] / totShots[id]
		}
		out = append(out, fitResult{
			id: id, name: name, shots: totShots[id], carriers: totCarr[id], tauxPort: tp,
			coefA: ca[i], coefB: cb[i], coefTotal: all[i],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].shots > out[j].shots })
	// R2 hors echantillon : coefficients de A evalues sur B, et l inverse.
	return out, r2(b, ids, ca), r2(a, ids, cb)
}

// retainedWeapons rend les armes portant au moins minShots tirs sur tout le corpus,
// triees par identifiant (ordre stable du systeme).
func retainedWeapons(rows []designRow, minShots float64) []uint64 {
	tot := map[uint64]float64{}
	for _, r := range rows {
		for id, v := range r.shots {
			tot[id] += v
		}
	}
	var ids []uint64
	for id, v := range tot {
		if v >= minShots {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// fitOn resout (X'X + ridge*I) p = X'y par elimination de Gauss avec pivot partiel.
func fitOn(rows []designRow, ids []uint64, ridge float64) []float64 {
	n := len(ids)
	xtx := make([][]float64, n)
	for i := range xtx {
		xtx[i] = make([]float64, n+1)
	}
	for _, r := range rows {
		for i, idi := range ids {
			xi := r.shots[idi]
			if xi == 0 {
				continue
			}
			for j, idj := range ids {
				xtx[i][j] += xi * r.shots[idj]
			}
			xtx[i][n] += xi * r.apiHits
		}
	}
	for i := 0; i < n; i++ {
		xtx[i][i] += ridge
	}
	return gauss(xtx, n)
}

// gauss resout le systeme augmente [A|b] de taille n par elimination avec pivot partiel.
func gauss(m [][]float64, n int) []float64 {
	for col := 0; col < n; col++ {
		piv := col
		for r := col + 1; r < n; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[piv][col]) {
				piv = r
			}
		}
		m[col], m[piv] = m[piv], m[col]
		if math.Abs(m[col][col]) < 1e-12 {
			continue
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := m[r][col] / m[col][col]
			for c := col; c <= n; c++ {
				m[r][c] -= f * m[col][c]
			}
		}
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		if math.Abs(m[i][i]) > 1e-12 {
			out[i] = m[i][n] / m[i][i]
		}
	}
	return out
}

// r2 rend le coefficient de determination des coefficients p sur les lignes fournies.
func r2(rows []designRow, ids []uint64, p []float64) float64 {
	if len(rows) == 0 {
		return 0
	}
	var mean float64
	for _, r := range rows {
		mean += r.apiHits
	}
	mean /= float64(len(rows))
	var ssRes, ssTot float64
	for _, r := range rows {
		var pred float64
		for i, id := range ids {
			pred += p[i] * r.shots[id]
		}
		ssRes += (r.apiHits - pred) * (r.apiHits - pred)
		ssTot += (r.apiHits - mean) * (r.apiHits - mean)
	}
	if ssTot == 0 {
		return 0
	}
	return 1 - ssRes/ssTot
}

// permuteHits rend une copie des lignes dont les totaux de touches API sont REDISTRIBUES
// entre matchs de la meme famille. C est la nulle : si la deconvolution retrouve les memes
// coefficients sur des touches permutees, elle ne lit pas les armes, elle lit une echelle.
func permuteHits(rows []designRow, seed int) []designRow {
	byFam := map[string][]int{}
	for i, r := range rows {
		byFam[r.famille] = append(byFam[r.famille], i)
	}
	out := append([]designRow(nil), rows...)
	for _, idx := range byFam {
		// permutation deterministe : rotation d un decalage derive de la graine
		k := len(idx)
		if k < 2 {
			continue
		}
		shift := 1 + seed%(k-1)
		vals := make([]float64, k)
		for i, j := range idx {
			vals[i] = rows[j].apiHits
		}
		for i, j := range idx {
			out[j].apiHits = vals[(i+shift)%k]
		}
	}
	return out
}

// runFit construit la matrice (match x arme) des tirs decodes et ajuste les taux de touche
// contre le total de touches de l API du match. Aucun appariement indice -> xuid n est
// necessaire : la mesure vit au grain du MATCH des deux cotes.
func runFit(root string, pfxs []string, refs map[string]*matchRef, names map[uint64]string, minShots float64, outCSV string, normalize bool) {
	rows := make([]designRow, 0, len(pfxs))
	for _, pfx := range pfxs {
		m := refs[pfx]
		recs := scanRecords(filepath.Join(root, pfx), bitCounter)
		row := designRow{pfx: pfx, famille: familyOf(m.pairName),
			shots: map[uint64]float64{}, carriers: map[uint64]float64{}}
		for _, r := range recs {
			if !isFire(r) {
				continue
			}
			row.shots[r.weapon]++
			if r.porteur {
				row.carriers[r.weapon]++
			}
		}
		for _, p := range m.players {
			row.apiHits += float64(p.shotsHit)
			row.apiFired += float64(p.shotsFired)
		}
		if row.apiHits <= 0 || len(row.shots) == 0 {
			continue
		}
		// NORMALISATION DE VISIBILITE. Le film ne montre pas la meme FRACTION des tirs
		// d un match a l autre : 0.92 en Tactical, 0.31 en Fiesta (mesure de cette
		// session). Sans correction, cette fraction entre dans les coefficients et les
		// rend inintelligibles. On la retire par le seul chiffre disponible — le total de
		// tirs de l API du match — ce qui fait de la somme des tirs decodes une estimation
		// du total reel, arme par arme.
		if normalize {
			var tot float64
			for _, v := range row.shots {
				tot += v
			}
			if tot <= 0 || row.apiFired <= 0 {
				continue
			}
			k := row.apiFired / tot
			for id := range row.shots {
				row.shots[id] *= k
			}
		}
		rows = append(rows, row)
	}
	fmt.Fprintf(os.Stderr, "matchs retenus dans le systeme: %d\n", len(rows))

	res, r2ab, r2ba := solveHitRates(rows, names, minShots, 1e-6)
	fmt.Printf("R2 hors echantillon : A->B %.4f   B->A %.4f   (%d matchs, %d armes)\n\n",
		r2ab, r2ba, len(rows), len(res))
	fmt.Printf("%-22s %9s %9s %10s %9s %9s %9s\n",
		"arme", "tirs", "porteurs", "tx_porteur", "coef_tot", "coef_A", "coef_B")
	for _, r := range res {
		fmt.Printf("%-22s %9.0f %9.0f %10.4f %9.4f %9.4f %9.4f\n",
			r.name, r.shots, r.carriers, r.tauxPort, r.coefTotal, r.coefA, r.coefB)
	}

	// LA NULLE : touches API redistribuees entre matchs de la meme famille. Une methode
	// qui rend les memes coefficients sur la nulle ne lit pas les armes.
	var nullSpread float64
	for s := 1; s <= 5; s++ {
		nr, _, _ := solveHitRates(permuteHits(rows, s*7), names, minShots, 1e-6)
		var d float64
		for i := range nr {
			d += math.Abs(nr[i].coefTotal - res[i].coefTotal)
		}
		nullSpread += d / float64(len(nr))
	}
	fmt.Printf("\nnulle (touches permutees intra-famille, 5 tirages) : ecart moyen des coefficients %.4f\n",
		nullSpread/5)

	w, closeW := openCSV(outCSV, []string{"weapon_id", "arme", "tirs", "porteurs", "taux_porteur", "coef_total", "coef_A", "coef_B"})
	defer closeW()
	for _, r := range res {
		writeRow(w, strconv.FormatUint(r.id, 10), r.name,
			fmt.Sprintf("%.0f", r.shots), fmt.Sprintf("%.0f", r.carriers),
			fmt.Sprintf("%.4f", r.tauxPort), fmt.Sprintf("%.4f", r.coefTotal),
			fmt.Sprintf("%.4f", r.coefA), fmt.Sprintf("%.4f", r.coefB))
	}
}

// loadWeaponNames lit le referentiel weapon_id -> nom exporte de metadata.duckdb.
func loadWeaponNames(path string) map[uint64]string {
	out := map[uint64]string{}
	if path == "" {
		return out
	}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil || len(rows) < 2 {
		return out
	}
	for _, row := range rows[1:] {
		if len(row) < 2 {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(row[0]), 10, 64)
		if err != nil {
			continue
		}
		out[id] = row[1]
	}
	return out
}

// boundedFit resout min ||Xp - y||^2 sous contrainte p dans [0,1], par descente de gradient
// PROJETEE sur les equations normales. Les taux de touche negatifs du grain match n etaient
// pas du bruit : c etait de l information jetee par une resolution non contrainte. Un taux de
// touche vit dans [0,1] par definition — le dire au solveur, c est retirer un degre de liberte
// qui ne correspond a rien de physique.
func boundedFit(rows []playerRow, ids []uint64, iters int) []float64 {
	n := len(ids)
	if n == 0 {
		return nil
	}
	// equations normales : G = X'X, c = X'y
	g := make([][]float64, n)
	for i := range g {
		g[i] = make([]float64, n)
	}
	c := make([]float64, n)
	for _, r := range rows {
		for i, idi := range ids {
			xi := r.shots[idi]
			if xi == 0 {
				continue
			}
			for j, idj := range ids {
				if xj := r.shots[idj]; xj != 0 {
					g[i][j] += xi * xj
				}
			}
			c[i] += xi * r.apiHits
		}
	}
	// pas = 1 / L, L majore par la trace (borne de Gershgorin suffisante ici)
	var l float64
	for i := 0; i < n; i++ {
		l += g[i][i]
	}
	if l <= 0 {
		return make([]float64, n)
	}
	step := 1 / l
	p := make([]float64, n)
	for i := range p {
		p[i] = 0.3 // depart neutre : l ordre de grandeur d une precision
	}
	grad := make([]float64, n)
	for it := 0; it < iters; it++ {
		for i := 0; i < n; i++ {
			s := -c[i]
			for j := 0; j < n; j++ {
				s += g[i][j] * p[j]
			}
			grad[i] = s
		}
		for i := 0; i < n; i++ {
			v := p[i] - step*grad[i]
			switch {
			case v < 0:
				v = 0
			case v > 1:
				v = 1
			}
			p[i] = v
		}
	}
	return p
}
