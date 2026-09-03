package filmdec

// vehicules_v5b_champs_test.go — LOT V5B, ÉTAPE 3 : LE BLOC PORTE-T-IL UN CHAMP CONNU ?
//
// LE DICTIONNAIRE D'HYPOTHÈSES EST CELUI DU CHANTIER, pas une devinette : les événements
// board/exit lisent des RÉFÉRENCES GARDÉES `[garde:1][index:w][génération:2]` avec w = 8
// (domaine 2, l'occupant d'un embarquement), w = 7 (domaine 3, le siège), w = 13 (domaine 7), et
// un SIÈGE en `R(6)` juste après les trois réfs (event_list.go). Le moteur réutilise ses
// encodages : si le bloc nomme un occupant, il doit le faire sous l'une de ces formes, ou sous
// l'une des quatre formes d'entité déjà balayées par le lot V5 (`v5Extracteurs`).
//
// LA RÈGLE DE DÉCISION EST ÉCRITE AVANT LA MESURE. Un canal (forme x décalage) ne compte que
// s'il désigne le BON occupant sur TOUTES les instances attestées. Avec ~9 instances et un champ
// de 8 bits, une coïncidence sur toutes vaut (1/256)^9 : un seul canal passant serait décisif.
//
// LE TÉMOIN EST UNE PERMUTATION : les mêmes canaux, les mêmes bits, mais les occupants attendus
// décalés d'un cran. Un canal qui « marche » aussi sous permutation ne lit rien.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5BChamps -v -timeout 180m

import (
	"fmt"
	"sort"
	"testing"
)

// v5bMarge élargit le balayage de part et d'autre du bloc : la position d'insertion `p` est
// ESTIMÉE (maximum de vraisemblance), pas connue au bit près.
const v5bMarge = 48

// v5bInstance est UN bloc attesté : ses bits, et ce que l'oracle dit de son occupant.
type v5bInstance struct {
	Film        string
	Vehicule    uint32
	TS          uint64
	Pay         []byte
	Debut, D    int    // premier bit du bloc, taille du bloc
	Slot, Index uint32 // occupant attesté : slot absolu, index relatif à la base bipède
	Seat        uint32
	NOccupants  int
	BandeBase   uint32
}

// v5bForme décrit UNE façon de lire une valeur candidate à un décalage.
type v5bForme struct {
	Nom string
	// Lire rend la valeur candidate et `ok` si la forme est lisible ici (garde ouverte...).
	Lire func(pay []byte, at int, base uint32) (uint32, bool)
	// Cible dit ce que la valeur est censée valoir : "slot", "index" ou "siege".
	Cible string
}

// v5bFormes est le dictionnaire complet : les quatre formes d'entité du lot V5, les trois
// largeurs de référence gardée des événements véhicule, et le siège `R(6)`.
func v5bFormes() []v5bForme {
	var out []v5bForme
	for _, ex := range v5Extracteurs {
		e := ex
		out = append(out, v5bForme{
			Nom: "brut-" + e.Nom, Cible: "slot",
			Lire: func(pay []byte, at int, _ uint32) (uint32, bool) {
				return e.Slot(kfReadBits(pay, at, e.Largeur)), true
			},
		})
		out = append(out, v5bForme{
			Nom: "brut-idx-" + e.Nom, Cible: "index",
			Lire: func(pay []byte, at int, _ uint32) (uint32, bool) {
				return e.Slot(kfReadBits(pay, at, e.Largeur)), true
			},
		})
	}
	for _, w := range []int{dom3RefWidth, dom2RefWidth, dom7RefWidth} {
		lw := w
		out = append(out, v5bForme{
			Nom: fmt.Sprintf("refGardée(w=%d)->slot", lw), Cible: "slot",
			Lire: func(pay []byte, at int, base uint32) (uint32, bool) {
				r := readPlainRef(pay, at, lw)
				if !r.Present {
					return 0, false
				}
				return base + r.Index, true
			},
		})
		out = append(out, v5bForme{
			Nom: fmt.Sprintf("refGardée(w=%d)->index", lw), Cible: "index",
			Lire: func(pay []byte, at int, _ uint32) (uint32, bool) {
				r := readPlainRef(pay, at, lw)
				if !r.Present {
					return 0, false
				}
				return r.Index, true
			},
		})
	}
	out = append(out, v5bForme{
		Nom: "siège R(6)", Cible: "siege",
		Lire: func(pay []byte, at int, _ uint32) (uint32, bool) {
			return uint32(kfReadBits(pay, at, vehicleSeatBits)), true
		},
	})
	return out
}

// v5bAttendu rend la valeur que la cible doit prendre pour l'instance `i`.
func v5bAttendu(cible string, i v5bInstance) uint32 {
	switch cible {
	case "slot":
		return i.Slot
	case "index":
		return i.Index
	default:
		return i.Seat
	}
}

// TestV5BChamps balaie toutes les formes à tous les décalages du bloc (± marge) et publie les
// meilleurs canaux, avec leur témoin par permutation.
func TestV5BChamps(t *testing.T) {
	var inst []v5bInstance
	for _, dir := range v5Films(t) {
		inst = append(inst, v5bInstances(t, dir)...)
	}
	if len(inst) == 0 {
		t.Log("V5B CHAMPS — aucune instance attestée ; rien à balayer")
		return
	}
	t.Logf("V5B CHAMPS — %d blocs attestés", len(inst))
	for _, i := range inst {
		t.Logf("    %s véh=%-5d ts=%9.2f bloc=[%d,+%d) occupant slot=%d idx=%d siège=%d",
			i.Film, i.Vehicule, float64(i.TS)/1e6, i.Debut, i.D, i.Slot, i.Index, i.Seat)
	}
	// PERMUTATION : l'occupant attendu de l'instance k devient celui de l'instance k+1.
	perm := make([]v5bInstance, len(inst))
	copy(perm, inst)
	for k := range perm {
		src := inst[(k+1)%len(inst)]
		perm[k].Slot, perm[k].Index, perm[k].Seat = src.Slot, src.Index, src.Seat
	}
	v5bBalayeChamps(t, "RÉEL", inst)
	v5bBalayeChamps(t, "TÉMOIN (occupants permutés)", perm)
}

// v5bBalayeChamps publie, par cible, le meilleur canal (forme x décalage).
func v5bBalayeChamps(t *testing.T, quoi string, inst []v5bInstance) {
	t.Helper()
	type cle struct {
		forme int
		dec   int
	}
	formes := v5bFormes()
	best := map[string]struct {
		n int
		c cle
	}{}
	for fi, f := range formes {
		for dec := -v5bMarge; dec < inst[0].D+v5bMarge; dec++ {
			n := 0
			for _, i := range inst {
				at := i.Debut + dec
				if at < 0 || at+40 > len(i.Pay)*8 {
					continue
				}
				v, ok := f.Lire(i.Pay, at, i.BandeBase)
				if ok && v == v5bAttendu(f.Cible, i) {
					n++
				}
			}
			if b, ok := best[f.Cible]; !ok || n > b.n {
				best[f.Cible] = struct {
					n int
					c cle
				}{n, cle{fi, dec}}
			}
		}
	}
	cibles := make([]string, 0, len(best))
	for c := range best {
		cibles = append(cibles, c)
	}
	sort.Strings(cibles)
	t.Logf("  [%s] meilleur canal par cible, sur %d instances :", quoi, len(inst))
	for _, c := range cibles {
		b := best[c]
		t.Logf("      cible=%-6s  %d/%d = %s   (forme %s, décalage %+d dans le bloc)",
			c, b.n, len(inst), v5Pct(b.n, len(inst)), formes[b.c.forme].Nom, b.c.dec)
	}
}

// v5bInstances construit les blocs attestés d'un film : pour chaque record d'image-clé d'un
// véhicule apparié pendant son épisode, la position d'insertion contre le record libre le plus
// proche du même véhicule.
func v5bInstances(t *testing.T, dir string) []v5bInstance {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5B CHAMPS %s : %v", dir, err)
		return nil
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5B CHAMPS %s : %v", dir, err)
		return nil
	}
	base := ^uint32(0)
	for s := range bipedSlotBand(dir, v5TousChunks(dir)) {
		if s < base {
			base = s
		}
	}
	kfs := v5Keyframes(dir)
	var out []v5bInstance
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
			ins := v5bInsertion(o.v5KfRec, f.v5KfRec)
			if ins.D <= 0 {
				continue
			}
			for _, e := range app {
				if !e.Ok || e.Vehicule != veh || o.TS <= e.DebutUS || o.TS >= e.FinUS {
					continue
				}
				out = append(out, v5bInstance{
					Film: shortFilm(dir), Vehicule: veh, TS: o.TS, Pay: o.Payload,
					Debut: o.BitStart + ins.P, D: ins.D,
					Slot: e.Slot, Index: e.Slot - base, Seat: e.Seat,
					NOccupants: o.NOccupants, BandeBase: base,
				})
			}
		}
	}
	return out
}

// shortFilm rend les 8 derniers caractères du chemin (le short8 du film).
func shortFilm(dir string) string {
	if len(dir) <= 8 {
		return dir
	}
	return dir[len(dir)-8:]
}
