package replay

// visee_sens114_research_test.go — LOT A3 : LE SENS ENTREE / SORTIE DE LUNETTE.
//
// POURQUOI LA QUESTION CHANGE DE FORME. A4 a ferme l'enveloppe du type 114 : sa charge utile
// est un R(6) au bit 24, qui prend CINQ valeurs sur 00162144 avec une multiplicite tres
// desequilibree (83 paquets sur 125 portent la meme). Or un champ « sens » se partitionne en
// deux classes de tailles comparables : tout embarquement finit par un debarquement, aux
// seules morts-en-lunette pres. AUCUNE partition des multiplicites observees n'approche
// l'equilibre. Le sens n'est donc pas un bit du payload — ou l'evenement 114 ne porte qu'UN
// des deux sens, l'autre vivant dans un type d'evenement DISTINCT.
//
// LA LENTILLE NEUVE. La phase 6 a balaye tous les types d'evenement contre les 12 transitions
// CONFONDUES. Les separer est une mesure differente, et c'est celle qui repond a A3 : si le
// type 114 est l'embarquement, il couvrira les six ENTREES et pas les six SORTIES, et un autre
// type fera l'inverse.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	population : seuls les types portant >= 20 paquets sur le film sont testes (en deca, la
//	    couverture n'a pas de sens statistique) ;
//	statistique : nombre d'entrees couvertes (sur 6) et de sorties couvertes (sur 6), fenetre
//	    +/- 1,2 s ;
//	controle : la chronologie ENTIERE est translatee de delta (+/- 200 s, pas de 200 ms, la
//	    zone +/- 5 s exclue), ce qui preserve les ecarts entre transitions et la structure
//	    reelle des paquets. La p-value est la part des decalages atteignant la couverture
//	    observee ;
//	verdict : un type est CANDIDAT pour un sens si p < 1 % pour ce sens ET p >= 5 % pour
//	    l'autre — c'est-a-dire s'il separe. Un type significatif des DEUX cotes ne separe pas :
//	    il est publie comme « les deux sens » et ne repond pas a A3.
//
// SOUS GARDE (SENS114_FILM, qui doit pointer 00162144).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 SENS114_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeSens114 -v -timeout 30m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	sens114FilmEnv  = "SENS114_FILM"
	sens114MinPk    = 20
	sens114DeltaMax = 200_000
	sens114DeltaPas = 200
	sens114Garde    = 5000
	sens114PSepare  = 0.01
	sens114PMuet    = 0.05
)

// sens114Couvre compte, parmi les instants tries d'un type, combien de cibles sont couvertes.
func sens114Couvre(instants []int64, cibles []int64, delta int64) int {
	n := 0
	for _, c := range cibles {
		cible := c + delta
		i := sort.Search(len(instants), func(k int) bool {
			return instants[k] >= cible-sig114FenetreMS
		})
		if i < len(instants) && instants[i] <= cible+sig114FenetreMS {
			n++
		}
	}
	return n
}

// sens114PValue rend la part des decalages de controle atteignant la couverture observee.
func sens114PValue(instants []int64, cibles []int64, obs int) float64 {
	var total, auMoins int
	for delta := int64(-sens114DeltaMax); delta <= sens114DeltaMax; delta += sens114DeltaPas {
		if delta > -sens114Garde && delta < sens114Garde {
			continue
		}
		total++
		if sens114Couvre(instants, cibles, delta) >= obs {
			auMoins++
		}
	}
	if total == 0 {
		return 1
	}
	return float64(auMoins) / float64(total)
}

// sens114Separe rend le verdict de separation d'un type.
func sens114Separe(pIn, pOut float64) string {
	switch {
	case pIn < sens114PSepare && pOut >= sens114PMuet:
		return "CANDIDAT ENTREE"
	case pOut < sens114PSepare && pIn >= sens114PMuet:
		return "CANDIDAT SORTIE"
	case pIn < sens114PSepare && pOut < sens114PSepare:
		return "les deux sens (ne separe pas)"
	default:
		return "-"
	}
}

// sens114Cibles rend les instants film des entrees et des sorties, separement.
func sens114Cibles() (entrees, sorties []int64) {
	for _, e := range chronoVersFilm(chronoEpisodes, sig114OffsetMS) {
		entrees = append(entrees, e[0])
		sorties = append(sorties, e[1])
	}
	return entrees, sorties
}

// sens114ParType regroupe les instants des paquets par type d'evenement.
func sens114ParType(types [][2]int64) map[int64][]int64 {
	out := map[int64][]int64{}
	for _, tp := range types {
		out[tp[0]] = append(out[tp[0]], tp[1])
	}
	for ty := range out {
		sort.Slice(out[ty], func(i, j int) bool { return out[ty][i] < out[ty][j] })
	}
	return out
}

// TestViseeSens114 mesure, type par type, la couverture des entrees et celle des sorties.
func TestViseeSens114(t *testing.T) {
	dir := os.Getenv(sens114FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", sens114FilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	types, _ := canalLitTypes(dir)
	if len(types) == 0 {
		t.Fatalf("aucun paquet delta lu dans %s", dir)
	}
	entrees, sorties := sens114Cibles()
	parType := sens114ParType(types)
	var tys []int64
	for ty := range parType {
		if len(parType[ty]) >= sens114MinPk {
			tys = append(tys, ty)
		}
	}
	sort.Slice(tys, func(i, j int) bool { return tys[i] < tys[j] })
	t.Logf("A3. SENS — %d types testes (>= %d paquets) ; %d entrees et %d sorties etiquetees",
		len(tys), sens114MinPk, len(entrees), len(sorties))
	var candidats int
	for _, ty := range tys {
		inst := parType[ty]
		cIn := sens114Couvre(inst, entrees, 0)
		cOut := sens114Couvre(inst, sorties, 0)
		pIn := sens114PValue(inst, entrees, cIn)
		pOut := sens114PValue(inst, sorties, cOut)
		verdict := sens114Separe(pIn, pOut)
		if verdict != "-" {
			candidats++
		}
		t.Logf("    type %3d (%6d paquets) : entrees %d/6 (p=%.3f) · sorties %d/6 (p=%.3f) -> %s",
			ty, len(inst), cIn, pIn, cOut, pOut, verdict)
	}
	if candidats == 0 {
		t.Log("A3. VERDICT — aucun type ne separe les entrees des sorties aux seuils declares :" +
			" le sens de la mise a la lunette n'est porte ni par un champ de l'enveloppe 114," +
			" ni par un type d'evenement distinct reperable sur ce film.")
	}
}
