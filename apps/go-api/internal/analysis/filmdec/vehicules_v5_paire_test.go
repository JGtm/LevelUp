package filmdec

// vehicules_v5_paire_test.go — LOT V5 : LE TEST APPARIÉ, véhicule contre LUI-MÊME.
//
// POURQUOI APPARIER. `TestV5Forme` mesure que le record d'image-clé d'un véhicule OCCUPÉ est
// plus long que celui d'un véhicule « vide » (médiane 2 060 contre 1 747 bits sur `0d76e8f1`).
// Ce témoin est FAUX en soi, et il faut le dire : le corpus n'atteste que les trajets qui
// finissent par une SORTIE décodée (ratio board:exit = 1:15, cf. V3_EMBARQUEMENT § 2.4), donc
// la classe « vide » contient une majorité de véhicules RÉELLEMENT occupés dont le trajet
// n'est pas attesté. Un écart mesuré contre un témoin contaminé est une BORNE BASSE.
//
// LE TEST APPARIÉ ÉLIMINE L'AUTRE CONFUSION : celle du TYPE de véhicule (un Wraith et une
// Mongoose n'ont pas le même nombre de composants, donc pas la même longueur de record). On
// compare chaque véhicule À LUI-MÊME : sa longueur de record aux images-clés PENDANT son
// épisode attesté, contre sa longueur aux images-clés où aucun épisode ne le concerne.
//
// TÉMOIN PAR DÉCALAGE : le même appariement, épisodes décalés de 37 s.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5Paire -v -timeout 180m

import (
	"sort"
	"testing"
)

// v5PaireObs est une observation appariée : un véhicule, sa longueur médiane de record
// pendant l'épisode, et hors épisode.
type v5PaireObs struct {
	Film            string
	Vehicule        uint32
	Occupe, Libre   int
	NOccupe, NLibre int
	// SautOccupe / SautLibre : médiane de l'écart de slot au record suivant TROUVÉ, dans
	// chaque classe. C'est le contrôle de confusion (cf. v5KfRec.SautSlot).
	SautOccupe, SautLibre int
}

// TestV5Paire mesure, véhicule par véhicule, l'écart de longueur de record d'image-clé entre
// « pendant son épisode attesté » et « hors épisode ».
func TestV5PaireVehicule(t *testing.T) {
	var obs, obsTemoin, obsPur, obsPurTemoin []v5PaireObs
	for _, dir := range v5Films(t) {
		o, ot, p, pt := v5PaireUnFilm(t, dir)
		obs = append(obs, o...)
		obsTemoin = append(obsTemoin, ot...)
		obsPur = append(obsPur, p...)
		obsPurTemoin = append(obsPurTemoin, pt...)
	}
	t.Logf("")
	t.Logf("V5 PAIRE — CUMUL")
	v5PaireVerdict(t, "réel", obs)
	v5PaireVerdict(t, "témoin décalé 37 s", obsTemoin)
	v5PaireVerdict(t, "réel, records à voisin immédiat", obsPur)
	v5PaireVerdict(t, "témoin décalé, records à voisin immédiat", obsPurTemoin)
}

// TestV5PaireBipede est le MÊME test apparié, côté BIPÈDE : le record d'image-clé d'un joueur
// change-t-il de longueur pendant qu'il est à bord, par rapport à ses propres images-clés à
// pied ? C'est la question du GATE (a) : peut-on dire, à une image-clé quelconque, que CE
// joueur est dans un véhicule ?
func TestV5PaireBipede(t *testing.T) {
	var obs, obsTemoin []v5PaireObs
	for _, dir := range v5Films(t) {
		eps, _, err := v5Episodes(dir)
		if err != nil {
			t.Logf("V5 PAIRE BIPÈDE %s : %v", dir, err)
			continue
		}
		kfs := v5Keyframes(dir)
		r := v5PaireBipedeCollecte(dir, eps, kfs, 0)
		w := v5PaireBipedeCollecte(dir, eps, kfs, v5DecalTemoinUS)
		t.Logf("V5 PAIRE BIPÈDE %s — %d joueurs appariés (réel), %d (témoin)", dir, len(r), len(w))
		for _, o := range r {
			t.Logf("    bipède=%-5d à bord méd=%-6d (n=%d)   à pied méd=%-6d (n=%d)   écart=%+d bits",
				o.Vehicule, o.Occupe, o.NOccupe, o.Libre, o.NLibre, o.Occupe-o.Libre)
		}
		obs = append(obs, r...)
		obsTemoin = append(obsTemoin, w...)
	}
	t.Logf("")
	t.Logf("V5 PAIRE BIPÈDE — CUMUL")
	v5PaireVerdict(t, "réel", obs)
	v5PaireVerdict(t, "témoin décalé 37 s", obsTemoin)
}

// v5PaireBipedeCollecte est v5PaireCollecte pour les records bipède : la classe « occupé »
// est « ce slot est à bord à cette image-clé ».
func v5PaireBipedeCollecte(dir string, eps []v5Episode, kfs [][]v5KfRec, decal uint64) []v5PaireObs {
	aBord, aPied := map[uint32][]int{}, map[uint32][]int{}
	cibles := map[uint32]bool{}
	for _, e := range eps {
		cibles[e.Slot] = true
	}
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		enCours := map[uint32]bool{}
		for _, e := range eps {
			if ts > e.DebutUS+decal && ts < e.FinUS+decal {
				enCours[e.Slot] = true
			}
		}
		for _, r := range kf {
			if r.TI != v5BipedeTI || !cibles[uint32(r.Slot)] {
				continue
			}
			if enCours[uint32(r.Slot)] {
				aBord[uint32(r.Slot)] = append(aBord[uint32(r.Slot)], r.LongueurEnBits)
			} else {
				aPied[uint32(r.Slot)] = append(aPied[uint32(r.Slot)], r.LongueurEnBits)
			}
		}
	}
	var out []v5PaireObs
	for s := range cibles {
		o, l := aBord[s], aPied[s]
		if len(o) == 0 || len(l) == 0 {
			continue
		}
		out = append(out, v5PaireObs{Film: dir, Vehicule: s,
			Occupe: v5Med(o), Libre: v5Med(l), NOccupe: len(o), NLibre: len(l)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Vehicule < out[j].Vehicule })
	return out
}

func v5PaireUnFilm(t *testing.T, dir string) (reel, temoin, pur, purTemoin []v5PaireObs) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5 PAIRE %s : %v", dir, err)
		return nil, nil, nil, nil
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5 PAIRE %s : %v", dir, err)
		return nil, nil, nil, nil
	}
	kfs := v5Keyframes(dir)
	reel = v5PaireCollecte(dir, app, kfs, 0, false)
	temoin = v5PaireCollecte(dir, app, kfs, v5DecalTemoinUS, false)
	t.Logf("V5 PAIRE %s — %d véhicules appariés observés (réel), %d (témoin)",
		dir, len(reel), len(temoin))
	for _, o := range reel {
		t.Logf("    véhicule=%-5d occupé méd=%-6d (n=%d)   libre méd=%-6d (n=%d)   écart=%+d bits"+
			"   saut de slot occupé=%d libre=%d",
			o.Vehicule, o.Occupe, o.NOccupe, o.Libre, o.NLibre, o.Occupe-o.Libre,
			o.SautOccupe, o.SautLibre)
	}
	// CONTRÔLE DE CONFUSION : la même mesure, restreinte aux records dont le VOISIN SUIVANT
	// n'a PAS été sauté par le balayeur (saut de slot == 1). Si l'écart tient encore, il
	// porte bien sur la longueur d'UN record.
	pur = v5PaireCollecte(dir, app, kfs, 0, true)
	purTemoin = v5PaireCollecte(dir, app, kfs, v5DecalTemoinUS, true)
	v5PaireVerdict(t, "réel, records à voisin immédiat", pur)
	return reel, temoin, pur, purTemoin
}

// v5PaireCollecte construit une observation par véhicule apparié, avec un décalage temporel
// appliqué aux épisodes (0 = réel).
func v5PaireCollecte(
	dir string, app []v5EpisodeApparie, kfs [][]v5KfRec, decal uint64, voisinImmediat bool,
) []v5PaireObs {
	occupe := map[uint32][]int{}
	libre := map[uint32][]int{}
	sautO := map[uint32][]int{}
	sautL := map[uint32][]int{}
	cibles := map[uint32]bool{}
	for _, e := range app {
		if e.Ok {
			cibles[e.Vehicule] = true
		}
	}
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		enCours := map[uint32]bool{}
		for _, e := range app {
			if e.Ok && ts > e.DebutUS+decal && ts < e.FinUS+decal {
				enCours[e.Vehicule] = true
			}
		}
		for _, r := range kf {
			if r.TI != v5VehiculeTI || !cibles[uint32(r.Slot)] {
				continue
			}
			if voisinImmediat && r.SautSlot != 1 {
				continue
			}
			s := uint32(r.Slot)
			if enCours[s] {
				occupe[s] = append(occupe[s], r.LongueurEnBits)
				sautO[s] = append(sautO[s], r.SautSlot)
			} else {
				libre[s] = append(libre[s], r.LongueurEnBits)
				sautL[s] = append(sautL[s], r.SautSlot)
			}
		}
	}
	var out []v5PaireObs
	for v := range cibles {
		o, l := occupe[v], libre[v]
		if len(o) == 0 || len(l) == 0 {
			continue // une paire incomplète ne dit rien
		}
		out = append(out, v5PaireObs{
			Film: dir, Vehicule: v,
			Occupe: v5Med(o), Libre: v5Med(l), NOccupe: len(o), NLibre: len(l),
			SautOccupe: v5Med(sautO[v]), SautLibre: v5Med(sautL[v]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Vehicule < out[j].Vehicule })
	return out
}

func v5Med(v []int) int {
	s := append([]int(nil), v...)
	sort.Ints(s)
	return s[len(s)/2]
}

// v5PaireVerdict publie le TEST DES SIGNES : sur les paires observées, combien ont un record
// plus long pendant l'occupation ? Un signal réel donne une nette majorité ; le hasard donne
// la moitié.
func v5PaireVerdict(t *testing.T, quoi string, obs []v5PaireObs) {
	t.Helper()
	plus, moins, egal := 0, 0, 0
	somme := 0
	for _, o := range obs {
		d := o.Occupe - o.Libre
		somme += d
		switch {
		case d > 0:
			plus++
		case d < 0:
			moins++
		default:
			egal++
		}
	}
	n := len(obs)
	if n == 0 {
		t.Logf("  [%s] aucune paire complète", quoi)
		return
	}
	t.Logf("  [%s] n=%d paires — plus long occupé : %d (%.1f %%), plus court : %d, égal : %d ; "+
		"écart moyen %+.0f bits", quoi, n, plus, float64(plus)/float64(n)*100, moins, egal,
		float64(somme)/float64(n))
}
