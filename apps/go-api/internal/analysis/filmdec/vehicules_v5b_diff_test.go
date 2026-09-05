package filmdec

// vehicules_v5b_diff_test.go — LOT V5B, ÉTAPE 1 : OÙ le bloc d'occupation s'insère-t-il ?
//
// CE QUE LE LOT V5 A ÉTABLI (rapport V5_ETAT_OCCUPATION_2026-09-02.md § 6) : le record
// d'image-clé d'un véhicule `ti=40` est PLUS LONG quand le véhicule est occupé — 16 paires sur
// 17 appariées véhicule-contre-lui-même, écart moyen +151 bits, écart RÉCURRENT de +89 bits sur
// trois films indépendants. Un bloc de taille fixe s'ajoute. On ne sait NI où, NI ce qu'il dit.
//
// LA MÉTHODE ICI : DIFFÉRENCE BIT À BIT, même véhicule, images-clés voisines. Deux profils
// d'accord agrégés sur toutes les paires :
//   - AVANT : accord de O[i] et F[i], alignés sur le DÉBUT du record ;
//   - ARRIÈRE : accord de O[Lo-1-i] et F[Lf-1-i], alignés sur la FIN.
//
// Si un bloc de `d` bits s'insère à la position `p`, l'accord AVANT est élevé pour i < p et
// tombe au hasard (50 %) au-delà ; l'accord ARRIÈRE est élevé pour i < Lo-p-d et tombe ensuite.
// Le croisement des deux localise `p` SANS supposer aucune grammaire.
//
// LE TÉMOIN EST INDISPENSABLE ET IL EST INTÉGRÉ : les mêmes profils entre DEUX records LIBRES
// du même véhicule. Le contenu d'un record dérive (position, orientation, minuteries) : sans ce
// témoin, une chute d'accord ne dit pas si elle vient du bloc ou de la dérive.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5BDiff -v -timeout 180m

import (
	"fmt"
	"sort"
	"testing"
)

// v5bProfilMax borne le profil en bits. Les records `ti=40` observés tiennent sous ~2 200 bits
// (V5 § 6.2 : médiane 2 060 occupé, 1 747 libre) ; 4 096 laisse de la marge.
const v5bProfilMax = 4096

// v5bFenetre est la largeur de lissage du profil publié (bits). 32 bits : assez fin pour
// localiser un bloc de 89 bits, assez large pour que le taux ait un sens.
const v5bFenetre = 32

// v5bProfil accumule, par décalage en bits, le nombre de comparaisons et le nombre d'accords.
type v5bProfil struct{ acc, n []int }

func v5bNouveauProfil() *v5bProfil {
	return &v5bProfil{acc: make([]int, v5bProfilMax), n: make([]int, v5bProfilMax)}
}

func (p *v5bProfil) ajoute(i int, ok bool) {
	if i < 0 || i >= v5bProfilMax {
		return
	}
	p.n[i]++
	if ok {
		p.acc[i]++
	}
}

// v5bDiffAvant remplit le profil aligné sur le DÉBUT des deux records.
func v5bDiffAvant(p *v5bProfil, a, b v5KfRec) {
	n := a.LongueurEnBits
	if b.LongueurEnBits < n {
		n = b.LongueurEnBits
	}
	for i := 0; i < n; i++ {
		p.ajoute(i, keyframeBitAt(a.Payload, a.BitStart+i) == keyframeBitAt(b.Payload, b.BitStart+i))
	}
}

// v5bDiffArriere remplit le profil aligné sur la FIN des deux records.
func v5bDiffArriere(p *v5bProfil, a, b v5KfRec) {
	n := a.LongueurEnBits
	if b.LongueurEnBits < n {
		n = b.LongueurEnBits
	}
	for i := 0; i < n; i++ {
		p.ajoute(i, keyframeBitAt(a.Payload, a.Fin-1-i) == keyframeBitAt(b.Payload, b.Fin-1-i))
	}
}

// v5bLCP rend la longueur du plus long préfixe commun en bits.
func v5bLCP(a, b v5KfRec) int {
	n := a.LongueurEnBits
	if b.LongueurEnBits < n {
		n = b.LongueurEnBits
	}
	for i := 0; i < n; i++ {
		if keyframeBitAt(a.Payload, a.BitStart+i) != keyframeBitAt(b.Payload, b.BitStart+i) {
			return i
		}
	}
	return n
}

// v5bLCS rend la longueur du plus long suffixe commun en bits.
func v5bLCS(a, b v5KfRec) int {
	n := a.LongueurEnBits
	if b.LongueurEnBits < n {
		n = b.LongueurEnBits
	}
	for i := 0; i < n; i++ {
		if keyframeBitAt(a.Payload, a.Fin-1-i) != keyframeBitAt(b.Payload, b.Fin-1-i) {
			return i
		}
	}
	return n
}

// v5bClasse trie les records d'image-clé d'UN véhicule apparié en OCCUPÉS et LIBRES, avec le
// nombre d'occupants attestés simultanés. `voisinImmediat` impose SlotGap == 1 (garde-fou V5).
type v5bRec struct {
	v5KfRec
	NOccupants int
}

// v5bCollecteVehicule rend, pour le véhicule `veh`, ses records d'image-clé annotés du nombre
// d'occupants attestés à cet instant.
func v5bCollecteVehicule(
	app []v5EpisodeApparie, kfs [][]v5KfRec, veh uint32, decal uint64, voisinImmediat bool,
) []v5bRec {
	var out []v5bRec
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		n := 0
		for _, e := range app {
			if e.Ok && e.Vehicule == veh && ts > e.DebutUS+decal && ts < e.FinUS+decal {
				n++
			}
		}
		for _, r := range kf {
			if r.TI != v5VehiculeTI || uint32(r.Slot) != veh {
				continue
			}
			if voisinImmediat && r.SautSlot != 1 {
				continue
			}
			out = append(out, v5bRec{v5KfRec: r, NOccupants: n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out
}

// v5bVehiculesApparies rend les slots véhicule appariés à au moins un épisode.
func v5bVehiculesApparies(app []v5EpisodeApparie) []uint32 {
	set := map[uint32]bool{}
	for _, e := range app {
		if e.Ok {
			set[e.Vehicule] = true
		}
	}
	out := make([]uint32, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// v5bPlusProche rend l'élément de `cand` dont l'horodatage est le plus proche de `ts`.
func v5bPlusProche(cand []v5bRec, ts uint64) (v5bRec, bool) {
	best, ok := v5bRec{}, false
	for _, c := range cand {
		if !ok || v5AbsDiff(c.TS, ts) < v5AbsDiff(best.TS, ts) {
			best, ok = c, true
		}
	}
	return best, ok
}

// TestV5BDiff — ÉTAPE 1 : localiser le bloc. Publie, par film et en cumul, les profils d'accord
// AVANT et ARRIÈRE (occupé contre libre) et leur TÉMOIN (libre contre libre).
func TestV5BDiff(t *testing.T) {
	av, ar := v5bNouveauProfil(), v5bNouveauProfil()
	avT, arT := v5bNouveauProfil(), v5bNouveauProfil()
	for _, dir := range v5Films(t) {
		v5bDiffUnFilm(t, dir, av, ar, avT, arT)
	}
	t.Logf("")
	t.Logf("V5B DIFF — CUMUL TOUS FILMS")
	v5bPublieProfil(t, "AVANT  occupé/libre", av)
	v5bPublieProfil(t, "AVANT  témoin libre/libre", avT)
	v5bPublieProfil(t, "ARRIÈRE occupé/libre", ar)
	v5bPublieProfil(t, "ARRIÈRE témoin libre/libre", arT)
}

func v5bDiffUnFilm(t *testing.T, dir string, av, ar, avT, arT *v5bProfil) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B DIFF %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B DIFF %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	nPaires, nTemoin := 0, 0
	for _, veh := range v5bVehiculesApparies(app) {
		recs := v5bCollecteVehicule(app, kfs, veh, 0, true)
		var occ, lib []v5bRec
		for _, r := range recs {
			if r.NOccupants > 0 {
				occ = append(occ, r)
			} else {
				lib = append(lib, r)
			}
		}
		if len(occ) == 0 || len(lib) == 0 {
			continue
		}
		for _, o := range occ {
			f, ok := v5bPlusProche(lib, o.TS)
			if !ok {
				continue
			}
			nPaires++
			v5bDiffAvant(av, o.v5KfRec, f.v5KfRec)
			v5bDiffArriere(ar, o.v5KfRec, f.v5KfRec)
			t.Logf("    véh=%-5d occ ts=%9.2f L=%-5d (n=%d)  libre ts=%9.2f L=%-5d  Δ=%+5d  "+
				"LCP=%-5d LCS=%-5d", veh, float64(o.TS)/1e6, o.LongueurEnBits, o.NOccupants,
				float64(f.TS)/1e6, f.LongueurEnBits, o.LongueurEnBits-f.LongueurEnBits,
				v5bLCP(o.v5KfRec, f.v5KfRec), v5bLCS(o.v5KfRec, f.v5KfRec))
		}
		// TÉMOIN : deux records LIBRES du même véhicule, aussi proches dans le temps que la
		// paire réelle. C'est la DÉRIVE de contenu, à laquelle tout le reste se compare.
		for i := 1; i < len(lib); i++ {
			nTemoin++
			v5bDiffAvant(avT, lib[i].v5KfRec, lib[i-1].v5KfRec)
			v5bDiffArriere(arT, lib[i].v5KfRec, lib[i-1].v5KfRec)
		}
	}
	t.Logf("V5B DIFF %s — %d paires occupé/libre, %d paires témoin libre/libre",
		dir, nPaires, nTemoin)
}

// v5bPublieProfil publie le taux d'accord par fenêtre de v5bFenetre bits, tant qu'il reste au
// moins un quart des comparaisons de la première fenêtre (au-delà, les records sont épuisés et
// le taux n'a plus de support).
func v5bPublieProfil(t *testing.T, quoi string, p *v5bProfil) {
	t.Helper()
	if p.n[0] == 0 {
		t.Logf("  [%s] aucune comparaison", quoi)
		return
	}
	seuil := p.n[0] / 4
	var b []string
	for d := 0; d+v5bFenetre <= v5bProfilMax; d += v5bFenetre {
		acc, n := 0, 0
		for i := d; i < d+v5bFenetre; i++ {
			acc += p.acc[i]
			n += p.n[i]
		}
		if n <= seuil*v5bFenetre/4 {
			break
		}
		b = append(b, fmt.Sprintf("%d:%.0f%%", d, float64(acc)/float64(n)*100))
	}
	t.Logf("  [%s] n0=%d comparaisons ; accord par fenêtre de %d bits :", quoi, p.n[0], v5bFenetre)
	for i := 0; i < len(b); i += 12 {
		j := i + 12
		if j > len(b) {
			j = len(b)
		}
		t.Logf("      %v", b[i:j])
	}
}

// TestV5BMulti — ÉTAPE 2 : « un bloc par occupant ». Sur le Warthog MULTI-OCCUPANTS de
// `0d76e8f1` (épisodes 6/7/8 qui se recouvrent), la longueur du record doit CROÎTRE avec le
// nombre d'occupants attestés simultanés.
func TestV5BMulti(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5bMultiUnFilm(t, dir)
	}
}

func v5bMultiUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B MULTI %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B MULTI %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	t.Logf("V5B MULTI %s — épisodes appariés par véhicule :", dir)
	parVeh := map[uint32][]v5EpisodeApparie{}
	for _, e := range app {
		if e.Ok {
			parVeh[e.Vehicule] = append(parVeh[e.Vehicule], e)
		}
	}
	for _, veh := range v5bVehiculesApparies(app) {
		for _, e := range parVeh[veh] {
			t.Logf("    véh=%-5d occupant=%-5d [%8.2f -> %8.2f] siège=%d(%v)",
				veh, e.Slot, float64(e.DebutUS)/1e6, float64(e.FinUS)/1e6, e.Seat, e.SeatValid)
		}
	}
	for _, veh := range v5bVehiculesApparies(app) {
		recs := v5bCollecteVehicule(app, kfs, veh, 0, true)
		parN := map[int][]int{}
		for _, r := range recs {
			parN[r.NOccupants] = append(parN[r.NOccupants], r.LongueurEnBits)
		}
		if len(parN) < 2 {
			continue
		}
		ns := make([]int, 0, len(parN))
		for n := range parN {
			ns = append(ns, n)
		}
		sort.Ints(ns)
		var s []string
		for _, n := range ns {
			s = append(s, fmt.Sprintf("N=%d: méd=%d (k=%d)", n, v5Med(parN[n]), len(parN[n])))
		}
		t.Logf("    véh=%-5d longueur par nombre d'occupants — %v", veh, s)
	}
}
