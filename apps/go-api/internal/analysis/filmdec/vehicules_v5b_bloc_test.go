package filmdec

// vehicules_v5b_bloc_test.go — LOT V5B, ÉTAPE 1bis : LOCALISER LE BLOC AU BIT PRÈS, puis LE LIRE.
//
// CE QUE `TestV5BDiff` A MONTRÉ : sur les paires occupé/libre du même véhicule, l'écart de
// longueur vaut EXACTEMENT +89 bits dans 5 cas sur 9, l'accord bit à bit aligné sur le DÉBUT est
// de 100 % jusqu'au bit ~288 puis s'effondre, et l'accord aligné sur la FIN reste haut jusqu'à
// ~1 250 bits de la queue. C'est la signature d'une INSERTION : un bloc de 89 bits ajouté à une
// position fixe, tout ce qui suit étant décalé. Le témoin libre/libre reste à 96-98 % PARTOUT,
// donc la chute n'est pas de la dérive de contenu.
//
// CE QUE CE FICHIER FAIT.
//  1. `v5bInsertion` cherche la position `p` qui MAXIMISE l'accord sous le modèle
//     « O = F[0:p] + BLOC(d) + F[p:] », par sommes préfixes (coût linéaire). Elle rend aussi
//     l'accord obtenu et l'accord du modèle SANS insertion (p = fin), qui est le témoin interne :
//     si l'insertion n'explique rien, les deux scores sont égaux.
//  2. Elle DUMPE les `d` bits du bloc, avec l'occupant et le siège attestés (oracle).
//  3. `TestV5BLongueurs` publie l'histogramme des longueurs par véhicule : si le bloc est de
//     taille fixe, les longueurs d'UN véhicule se rangent en base + 89 k.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run 'TestV5BBloc|TestV5BLongueurs' -v -timeout 180m

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// v5bTailleBloc est la taille MESURÉE du bloc d'occupation, en bits (cf. l'en-tête).
const v5bTailleBloc = 89

// v5bInsert est le résultat de la recherche du point d'insertion.
type v5bInsert struct {
	// P est la position (en bits depuis le début du record) qui maximise l'accord.
	P int
	// D est la taille du bloc supposé inséré (Lo - Lf).
	D int
	// Accord / Total : bits d'accord du meilleur modèle, sur le nombre de bits comparés.
	Accord, Total int
	// AccordSansInsertion : le même score avec p = 0 (tout le décalage rejeté en tête) et
	// p = Lf (tout rejeté en queue) — les deux modèles dégénérés qui servent de témoin.
	AccordTete, AccordQueue int
}

// v5bInsertion cherche la position d'insertion la plus probable d'un bloc de `d = Lo - Lf` bits.
//
// Modèle : O[0:p] doit valoir F[0:p], et O[p+d:Lo] doit valoir F[p:Lf]. Le score d'un `p` est le
// nombre total de bits d'accord ; le nombre de bits comparés vaut Lf quel que soit `p`, donc les
// scores sont directement comparables.
func v5bInsertion(o, f v5KfRec) v5bInsert {
	lo, lf := o.LongueurEnBits, f.LongueurEnBits
	d := lo - lf
	res := v5bInsert{D: d, Total: lf}
	if d <= 0 || lf <= 0 {
		return res
	}
	// A[i] = accords des i premiers bits, alignés sur le DÉBUT.
	a := make([]int, lf+1)
	for i := 0; i < lf; i++ {
		a[i+1] = a[i]
		if keyframeBitAt(o.Payload, o.BitStart+i) == keyframeBitAt(f.Payload, f.BitStart+i) {
			a[i+1]++
		}
	}
	// B[i] = accords des i derniers bits, alignés sur la FIN.
	b := make([]int, lf+1)
	for i := 0; i < lf; i++ {
		b[i+1] = b[i]
		if keyframeBitAt(o.Payload, o.Fin-1-i) == keyframeBitAt(f.Payload, f.Fin-1-i) {
			b[i+1]++
		}
	}
	best, bestP := -1, 0
	for p := 0; p <= lf; p++ {
		if s := a[p] + b[lf-p]; s > best {
			best, bestP = s, p
		}
	}
	res.P, res.Accord = bestP, best
	res.AccordTete, res.AccordQueue = b[lf], a[lf]
	return res
}

// v5bBits rend les `n` bits à partir de `at` sous forme de chaîne binaire.
func v5bBits(pay []byte, at, n int) string {
	var s strings.Builder
	for i := 0; i < n; i++ {
		if keyframeBitAt(pay, at+i) {
			s.WriteByte('1')
		} else {
			s.WriteByte('0')
		}
	}
	return s.String()
}

// TestV5BBloc — localise le bloc au bit près sur chaque paire occupé/libre, et le dumpe.
func TestV5BBloc(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5bBlocUnFilm(t, dir)
	}
}

func v5bBlocUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B BLOC %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B BLOC %s : %v", dir, err)
		return
	}
	bande := bipedSlotBandDir(dir, v5TousChunks(dir))
	base := uint32(0)
	if slots := bande.Slots(); len(slots) > 0 {
		base = slots[0]
	}
	kfs := v5Keyframes(dir)
	t.Logf("V5B BLOC %s — base de la bande bipède = %d", dir, base)
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
			v5bPublieBloc(t, o, f, veh, base, app)
		}
	}
}

// v5bPublieBloc publie, pour une paire, la position d'insertion la plus probable, la qualité du
// modèle et les bits du bloc, en face de l'ORACLE (occupants attestés, sièges).
func v5bPublieBloc(t *testing.T, o, f v5bRec, veh uint32, base uint32, app []v5EpisodeApparie) {
	t.Helper()
	ins := v5bInsertion(o.v5KfRec, f.v5KfRec)
	if ins.D <= 0 {
		t.Logf("  véh=%-5d ts=%9.2f  Δ=%+d — pas une insertion, ignoré", veh, float64(o.TS)/1e6, ins.D)
		return
	}
	var qui []string
	for _, e := range app {
		if e.Ok && e.Vehicule == veh && o.TS > e.DebutUS && o.TS < e.FinUS {
			qui = append(qui, fmt.Sprintf("slot=%d (idx=%d) siège=%d", e.Slot, int(e.Slot)-int(base), e.Seat))
		}
	}
	t.Logf("  véh=%-5d ts=%9.2f Lo=%-5d Lf=%-5d d=%-4d  p=%-5d accord=%d/%d (%.1f %%)  "+
		"témoin tête=%d queue=%d  oracle: %v",
		veh, float64(o.TS)/1e6, o.LongueurEnBits, f.LongueurEnBits, ins.D, ins.P,
		ins.Accord, ins.Total, float64(ins.Accord)/float64(ins.Total)*100,
		ins.AccordTete, ins.AccordQueue, qui)
	t.Logf("      bloc  = %s", v5bBits(o.Payload, o.BitStart+ins.P, ins.D))
	t.Logf("      avant = %s", v5bBits(o.Payload, o.BitStart+ins.P-64, 64))
	t.Logf("      après = %s", v5bBits(o.Payload, o.BitStart+ins.P+ins.D, 64))
	t.Logf("      libre@p= %s", v5bBits(f.Payload, f.BitStart+ins.P, 64))
}

// TestV5BLongueurs — l'histogramme des longueurs de record par véhicule, sur TOUT le film (pas
// seulement les véhicules appariés). Si le bloc est de taille fixe, les longueurs d'un véhicule
// donné se rangent en `base + 89 k`. Cette mesure ne dépend d'AUCUN oracle : c'est une propriété
// du format, testable sur n'importe quel film.
func TestV5BLongueurs(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5bLongueursUnFilm(t, dir)
	}
}

func v5bLongueursUnFilm(t *testing.T, dir string) {
	t.Helper()
	kfs := v5Keyframes(dir)
	parVeh := map[int]map[int]int{}
	for _, kf := range kfs {
		for _, r := range kf {
			if r.TI != v5VehiculeTI || r.SautSlot != 1 {
				continue
			}
			if parVeh[r.Slot] == nil {
				parVeh[r.Slot] = map[int]int{}
			}
			parVeh[r.Slot][r.LongueurEnBits]++
		}
	}
	slots := make([]int, 0, len(parVeh))
	for s := range parVeh {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	// COMPTE GLOBAL : parmi les véhicules à au moins 2 longueurs distinctes, combien ont TOUTES
	// leurs longueurs congruentes modulo 89 à la plus petite ?
	multi, congru := 0, 0
	for _, s := range slots {
		h := parVeh[s]
		if len(h) < 2 {
			continue
		}
		multi++
		ls := make([]int, 0, len(h))
		for l := range h {
			ls = append(ls, l)
		}
		sort.Ints(ls)
		ok := true
		var parts []string
		for _, l := range ls {
			k := (l - ls[0]) % v5bTailleBloc
			if k != 0 {
				ok = false
			}
			parts = append(parts, fmt.Sprintf("%d(x%d,+%dk%d)", l, h[l], l-ls[0], (l-ls[0])/v5bTailleBloc))
		}
		if ok {
			congru++
		}
		t.Logf("    véh=%-5d congru89=%-5v  %v", s, ok, parts)
	}
	t.Logf("V5B LONGUEURS %s — %d véhicules à >= 2 longueurs ; toutes congruentes mod %d : %d (%s)",
		dir, multi, v5bTailleBloc, congru, v5Pct(congru, multi))
}
