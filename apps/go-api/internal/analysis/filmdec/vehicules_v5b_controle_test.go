package filmdec

// vehicules_v5b_controle_test.go — LOT V5B : LE CONTRÔLE QUI PEUT TUER LE SIGNAL, et l'ancrage
// du bloc dans la grammaire.
//
// LE CONFONDANT, ÉNONCÉ AVANT DE LE MESURER. Un véhicule OCCUPÉ est un véhicule CONDUIT, donc un
// véhicule qui BOUGE. Un composant de mouvement à précision dynamique émet plus de bits quand
// l'objet est dynamique. Le « bloc de 89 bits » du lot V5 peut donc n'être PAS un bloc
// d'occupation mais un bloc de MOUVEMENT — et le contenu mesuré va dans ce sens : les 89 bits
// sont de haute entropie et NE CONTIENNENT PAS le slot ni l'index de l'occupant attesté
// (balayage de toutes les largeurs 6..16 à tous les décalages : 0 touche sur 5 blocs).
//
// LE TEST QUI SÉPARE LES DEUX : la table 2x2 « occupé x en mouvement », véhicule par véhicule.
//   - Si le bloc est de l'OCCUPATION : les records occupés-À-L'ARRÊT le portent quand même.
//   - Si le bloc est du MOUVEMENT : les records libres-EN-MOUVEMENT le portent aussi, et les
//     records occupés-à-l'arrêt ne le portent pas.
//
// L'ANCRAGE : `TestV5BAncrage` rejoue la grammaire de production (`WalkKeyframeBody`, état
// complet) sur les records occupés et publie les frontières de composants autour de la position
// d'insertion `p`. Si `p` tombe sur une frontière, le bloc EST un composant (ou son extension).
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run 'TestV5BMouvement|TestV5BAncrage' -v -timeout 180m

import (
	"fmt"
	"sort"
	"testing"
)

// v5bFenetreVitesseUS est la demi-fenêtre autour d'une image-clé dans laquelle on cherche les
// échantillons de position du véhicule pour estimer son déplacement (1,5 s).
const v5bFenetreVitesseUS = 1_500_000

// v5bSeuilBouge est le déplacement (en quanta) au-delà duquel le véhicule est déclaré EN
// MOUVEMENT sur la fenêtre. Les quanta sont l'unité native du flux ; le seuil est volontairement
// bas — c'est la SÉPARATION arrêt/mouvement qui compte, pas sa calibration physique.
const v5bSeuilBouge = 30.0

// v5bDeplacement rend l'amplitude du déplacement du véhicule `veh` autour de `ts`, en quanta, et
// le nombre d'échantillons qui l'ont servie.
func v5bDeplacement(pos []v5EchQ, veh uint32, ts uint64) (float64, int) {
	var fen []v5EchQ
	for _, p := range pos {
		if p.Slot == veh && v5AbsDiff(p.TS, ts) <= v5bFenetreVitesseUS {
			fen = append(fen, p)
		}
	}
	if len(fen) < 2 {
		return 0, len(fen)
	}
	best := 0.0
	for i := range fen {
		for j := i + 1; j < len(fen); j++ {
			if d := v5DistQ(fen[i].Q, fen[j].Q); d > best {
				best = d
			}
		}
	}
	return best, len(fen)
}

// v5bCellule agrège les longueurs de record d'une case de la table 2x2.
type v5bCellule struct{ l []int }

func (c *v5bCellule) add(n int) { c.l = append(c.l, n) }
func (c *v5bCellule) str() string {
	if len(c.l) == 0 {
		return "—"
	}
	return fmt.Sprintf("méd=%d (k=%d)", v5Med(c.l), len(c.l))
}

// TestV5BMouvement publie, véhicule par véhicule, la table 2x2 « occupé x en mouvement ».
func TestV5BMouvement(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5bMouvementUnFilm(t, dir)
	}
}

func v5bMouvementUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B MOUVEMENT %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B MOUVEMENT %s : %v", dir, err)
		return
	}
	bandeV := worldObjectSlotBandDir(dir, CountFilmChunks(dir), v5VehiculeTI)
	for _, s := range bipedSlotBandDir(dir, v5TousChunks(dir)).Slots() {
		delete(bandeV, s)
	}
	posV, err := v5PositionsBande(dir, NewSlotBand(bandeV))
	if err != nil {
		t.Logf("V5B MOUVEMENT %s : positions véhicule : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	t.Logf("V5B MOUVEMENT %s — table 2x2 par véhicule apparié (seuil %g quanta sur ±%.1f s)",
		dir, v5bSeuilBouge, float64(v5bFenetreVitesseUS)/1e6)
	// Cumul global des quatre cases, pour un verdict indépendant du véhicule.
	glob := map[string]*v5bCellule{}
	for _, k := range []string{"occ+bouge", "occ+arret", "libre+bouge", "libre+arret"} {
		glob[k] = &v5bCellule{}
	}
	for _, veh := range v5bVehiculesApparies(app) {
		cells := map[string]*v5bCellule{}
		for _, k := range []string{"occ+bouge", "occ+arret", "libre+bouge", "libre+arret"} {
			cells[k] = &v5bCellule{}
		}
		for _, r := range v5bCollecteVehicule(app, kfs, veh, 0, true) {
			d, n := v5bDeplacement(posV, veh, r.TS)
			if n < 2 {
				continue // pas d'estimation : la case n'est pas décidable, on n'invente pas
			}
			cle := "libre"
			if r.NOccupants > 0 {
				cle = "occ"
			}
			if d > v5bSeuilBouge {
				cle += "+bouge"
			} else {
				cle += "+arret"
			}
			cells[cle].add(r.LongueurEnBits)
			glob[cle].add(r.LongueurEnBits)
		}
		t.Logf("    véh=%-5d  occupé+bouge %-18s occupé+arrêt %-18s libre+bouge %-18s libre+arrêt %s",
			veh, cells["occ+bouge"].str(), cells["occ+arret"].str(),
			cells["libre+bouge"].str(), cells["libre+arret"].str())
	}
	t.Logf("  CUMUL %s : occupé+bouge %s | occupé+arrêt %s | libre+bouge %s | libre+arrêt %s",
		dir, glob["occ+bouge"].str(), glob["occ+arret"].str(),
		glob["libre+bouge"].str(), glob["libre+arret"].str())
}

// TestV5BVitesse — LE TEST SANS ORACLE, sur TOUS les véhicules du film. Pour chaque record
// d'image-clé `ti=40` à voisin immédiat, on calcule l'EXCÈS de longueur du record par rapport à
// la plus petite longueur jamais observée POUR CE VÉHICULE, et on le croise avec le DÉPLACEMENT
// du véhicule autour de cette image-clé.
//
// C'EST LE TEST QUI TRANCHE le confondant. La classe « libre » de l'oracle est contaminée (ratio
// board:exit = 1:15), donc « libre + en mouvement » peut être un véhicule secrètement occupé.
// Mais si l'excès de longueur suit le DÉPLACEMENT sur des records dont l'immense majorité n'a
// aucun épisode attesté, alors la variable expliquée est la CINÉMATIQUE, et l'occupation n'en est
// qu'une cause indirecte.
//
// LE SECOND VOLET est l'inverse et il est décisif pour le lot : à DÉPLACEMENT CONTRÔLÉ (records
// en mouvement seulement), l'occupation attestée ajoute-t-elle encore quelque chose ? Si non, le
// « signal d'occupation » du lot V5 est un signal de mouvement.
func TestV5BVitesse(t *testing.T) {
	cum := &v5bCumulVitesse{bins: make([]v5bBinVitesse, len(v5bBinsDeplacement)+1)}
	for _, dir := range v5Films(t) {
		v5bVitesseUnFilm(t, dir, cum)
	}
	t.Logf("")
	t.Logf("V5B VITESSE — CUMUL TOUS FILMS (records ti=40 à voisin immédiat)")
	t.Logf("  records examinés=%d, sans estimation de déplacement (< 2 échantillons)=%d, retenus=%d",
		cum.vus, cum.sansPos, cum.retenus)
	for i, b := range cum.bins {
		if b.n == 0 {
			continue
		}
		t.Logf("    déplacement %-11s n=%-5d  excès médian=%-6d  excès >= %d bits : %d (%s)",
			v5bBorne(i), b.n, v5Med(b.exces), v5bTailleBloc, b.avecBloc, v5Pct(b.avecBloc, b.n))
	}
	bas, haut := cum.regroupe(v5bSeuilBouge)
	t.Logf("  VERDICT MOUVEMENT : à l'arrêt (déplacement <= %g q) %d/%d = %s portent >= %d bits "+
		"d'excès ; en mouvement %d/%d = %s",
		v5bSeuilBouge, bas.avecBloc, bas.n, v5Pct(bas.avecBloc, bas.n), v5bTailleBloc,
		haut.avecBloc, haut.n, v5Pct(haut.avecBloc, haut.n))
	t.Logf("  À MOUVEMENT CONTRÔLÉ (records en mouvement seulement) : occupation ATTESTÉE "+
		"%d/%d = %s ; pas d'épisode attesté %d/%d = %s",
		cum.occBloc, cum.occN, v5Pct(cum.occBloc, cum.occN),
		cum.libBloc, cum.libN, v5Pct(cum.libBloc, cum.libN))
}

// v5bBinsDeplacement borne les classes de déplacement publiées (en quanta sur la fenêtre).
var v5bBinsDeplacement = []float64{0, 1, 10, 30, 100, 300, 1000}

func v5bBorne(i int) string {
	if i < len(v5bBinsDeplacement) {
		return fmt.Sprintf("< %g", v5bBinsDeplacement[i])
	}
	return "≥ dernier"
}

// v5bBinVitesse est une classe de déplacement.
type v5bBinVitesse struct {
	n, avecBloc int
	exces       []int
}

// v5bCumulVitesse agrège le test sur tous les films.
type v5bCumulVitesse struct {
	bins          []v5bBinVitesse
	vus, sansPos  int
	retenus       int
	occN, occBloc int // records EN MOUVEMENT dont l'occupation est attestée
	libN, libBloc int // records EN MOUVEMENT sans épisode attesté
}

// regroupe rend les deux classes « à l'arrêt » et « en mouvement » autour de `seuil`.
func (c *v5bCumulVitesse) regroupe(seuil float64) (bas, haut v5bBinVitesse) {
	for i, b := range c.bins {
		borne := 1e18
		if i < len(v5bBinsDeplacement) {
			borne = v5bBinsDeplacement[i]
		}
		if borne <= seuil {
			bas.n, bas.avecBloc = bas.n+b.n, bas.avecBloc+b.avecBloc
		} else {
			haut.n, haut.avecBloc = haut.n+b.n, haut.avecBloc+b.avecBloc
		}
	}
	return bas, haut
}

func v5bVitesseUnFilm(t *testing.T, dir string, cum *v5bCumulVitesse) {
	t.Helper()
	bandeV := worldObjectSlotBandDir(dir, CountFilmChunks(dir), v5VehiculeTI)
	for _, s := range bipedSlotBandDir(dir, v5TousChunks(dir)).Slots() {
		delete(bandeV, s)
	}
	posV, err := v5PositionsBande(dir, NewSlotBand(bandeV))
	if err != nil {
		t.Logf("V5B VITESSE %s : %v", dir, err)
		return
	}
	parSlot := map[uint32][]v5EchQ{}
	for _, p := range posV {
		parSlot[p.Slot] = append(parSlot[p.Slot], p)
	}
	// Occupation attestée, quand elle existe (le test tient sans : les compteurs occ/lib
	// restent alors vides et le premier volet suffit).
	occupe := map[uint32][][2]uint64{}
	if eps, _, err := v5Episodes(dir); err == nil {
		if app, err := v5Apparier(dir, eps); err == nil {
			for _, e := range app {
				if e.Ok {
					occupe[e.Vehicule] = append(occupe[e.Vehicule], [2]uint64{e.DebutUS, e.FinUS})
				}
			}
		}
	}
	kfs := v5Keyframes(dir)
	base := map[int]int{}
	for _, kf := range kfs {
		for _, r := range kf {
			if r.TI != v5VehiculeTI || r.SautSlot != 1 {
				continue
			}
			if b, ok := base[r.Slot]; !ok || r.LongueurEnBits < b {
				base[r.Slot] = r.LongueurEnBits
			}
		}
	}
	bins := make([]v5bBinVitesse, len(v5bBinsDeplacement)+1)
	for _, kf := range kfs {
		for _, r := range kf {
			if r.TI != v5VehiculeTI || r.SautSlot != 1 {
				continue
			}
			cum.vus++
			d, ns := v5bDeplacement(parSlot[uint32(r.Slot)], uint32(r.Slot), r.TS)
			if ns < 2 {
				cum.sansPos++
				continue
			}
			cum.retenus++
			i := sort.SearchFloat64s(v5bBinsDeplacement, d)
			ex := r.LongueurEnBits - base[r.Slot]
			avec := ex >= v5bTailleBloc
			for _, b := range []*[]v5bBinVitesse{&bins, &cum.bins} {
				(*b)[i].n++
				(*b)[i].exces = append((*b)[i].exces, ex)
				if avec {
					(*b)[i].avecBloc++
				}
			}
			if d <= v5bSeuilBouge {
				continue // le volet « à mouvement contrôlé » ne parle que des records qui bougent
			}
			att := false
			for _, w := range occupe[uint32(r.Slot)] {
				if r.TS > w[0] && r.TS < w[1] {
					att = true
				}
			}
			switch {
			case att:
				cum.occN++
				if avec {
					cum.occBloc++
				}
			default:
				cum.libN++
				if avec {
					cum.libBloc++
				}
			}
		}
	}
	t.Logf("V5B VITESSE %s — excès de longueur par classe de déplacement (base = plus court "+
		"record du véhicule) :", dir)
	for i, b := range bins {
		if b.n == 0 {
			continue
		}
		t.Logf("    déplacement %-11s n=%-5d  excès médian=%-6d  excès >= %d bits : %d (%s)",
			v5bBorne(i), b.n, v5Med(b.exces), v5bTailleBloc, b.avecBloc, v5Pct(b.avecBloc, b.n))
	}
}

// TestV5BAncrage rejoue la grammaire de production sur les records véhicule occupés et publie
// les frontières de composants, pour dire si la position d'insertion `p` tombe sur l'une d'elles.
func TestV5BAncrage(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5bAncrageUnFilm(t, dir)
	}
}

func v5bAncrageUnFilm(t *testing.T, dir string) {
	t.Helper()
	brut, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Logf("V5B ANCRAGE %s : chunk_00 : %v", dir, err)
		return
	}
	reg, err := ParseRegistryChunk(brut)
	if err != nil {
		t.Logf("V5B ANCRAGE %s : registre : %v", dir, err)
		return
	}
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B ANCRAGE %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B ANCRAGE %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	release := LockProcessDecode()
	defer release()
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
			if !ok || o.LongueurEnBits <= f.LongueurEnBits {
				continue
			}
			ins := v5bInsertion(o.v5KfRec, f.v5KfRec)
			v5bPublieAncrage(t, veh, o, f, ins, reg)
		}
	}
}

// v5bPublieAncrage publie, pour une paire, les frontières de composants encadrant `p` dans le
// record OCCUPÉ et dans le record LIBRE, sous la variante « état complet » (la lecture désignée
// par le lot R3 pour la table d'image-clé).
func v5bPublieAncrage(t *testing.T, veh uint32, o, f v5bRec, ins v5bInsert, reg *Registry) {
	t.Helper()
	v := KeyframeBodyVariant{DefaultState: true, Gate: false, Mask: false}
	to := WalkKeyframeBody(o.Payload, o.BitStart, reg, v)
	tf := WalkKeyframeBody(f.Payload, f.BitStart, reg, v)
	t.Logf("  véh=%-5d ts=%9.2f  p=%-5d d=%-4d | occupé : desync=%d fin=%+d bits du record ; "+
		"libre : desync=%d fin=%+d",
		veh, float64(o.TS)/1e6, ins.P, ins.D,
		to.DesyncAt, to.EndBit-o.Fin, tf.DesyncAt, tf.EndBit-f.Fin)
	t.Logf("      composants occupé : %s", v5bBornes(to, o.BitStart, ins.P))
	t.Logf("      composants libre  : %s", v5bBornes(tf, f.BitStart, ins.P))
}

// v5bBornes rend les composants traversés avec leur décalage RELATIF au début du record, en
// marquant celui qui contient `p`.
func v5bBornes(tr EntityTrace, bitStart, p int) string {
	type b struct {
		nom string
		off int
		ok  bool
	}
	var l []b
	for _, c := range tr.Comps {
		l = append(l, b{fmt.Sprintf("i%d %s", c.Index, c.Name), c.StartBit - bitStart, c.Ported})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].off < l[j].off })
	var s []string
	for i, c := range l {
		marque := ""
		if c.off <= p && (i+1 == len(l) || l[i+1].off > p) {
			marque = " <<< p"
		}
		porte := ""
		if !c.ok {
			porte = "!"
		}
		s = append(s, fmt.Sprintf("%d=%s%s%s", c.off, c.nom, porte, marque))
	}
	return fmt.Sprint(s)
}
