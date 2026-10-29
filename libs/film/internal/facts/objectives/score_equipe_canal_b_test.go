package objectives

// score_equipe_canal_b_test.go — E8.2 : LE CANAL B D'UN SLOT D'EQUIPE N'EST PAS UN SCORE.
//
// Deux films classes de Lattice (KOTH) publiaient une courbe de score fausse : `26602661` rendait
// un score nul pour le camp qui avait marque un point (« descente » 1 -> 0), `7de0b91d` rendait un
// score nul pour le camp qui en avait marque un (le point disparaissait, le seuil de garde en
// etait fausse). Les emissions BRUTES, releves au fichier de faits de chaque film (2026-10-17) :
//
//	26602661  slot 8 :  206864 ms A=1 B=0 ; 221567 ms A=-69 B=120 ; 237600 ms A=0 B=5
//	7de0b91d  slot 8 :  283850 ms A=0 B=5 ; 434821 ms A=1 B=0 ; 551853 ms A=0 B=5
//
// Registre : 26602661 finit 3-1 (le camp du slot 8 a UN point), 7de0b91d finit 1-2 (le camp du slot 8
// a UN point). La cause n'est ni la manche ni le camp : l'emission `A=0 B=5` PASSE le filtre de
// domaine (0 <= 5 <= 250) et, plus tardive que le vrai point, GAGNE le departage de la plus longue
// sous-suite (a longueur egale, `longestRun` garde la derniere emission). Or sur les 21 042
// emissions d'equipe du composant 0 des 237 films du cache de faits, 20 997 portent B = 0 ; les 45
// autres (35 films) sont des paires illisibles — `(0,5)` seize fois, le reste aberrant — et
// aucune n'est un score.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/types"
)

// emissionEquipe pose une emission du composant 0 d'un slot d'equipe, manche 0.
func emissionEquipe(slot, tMS int, a, b int64) types.StatRecord {
	return types.StatRecord{
		TimeMS: tMS, Slot: slot, Round: 0,
		Comps: map[int]types.StatValue{modeScoreComp: {A: a, B: b}},
	}
}

// valeursDe rend les valeurs d'une suite, dans l'ordre.
func valeursDe(pts []types.ScorePoint) []int64 {
	out := make([]int64, 0, len(pts))
	for _, p := range pts {
		out = append(out, p.Value)
	}
	return out
}

func memesValeurs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestScoreEquipe_EmissionACanalBNonNulNeGagneJamais — les emissions brutes de `26602661` et de
// `7de0b91d` : le point reel survit, l'emission a canal B non nul est jetee.
func TestScoreEquipe_EmissionACanalBNonNulNeGagneJamais(t *testing.T) {
	cas := []struct {
		nom        string
		recs       []types.StatRecord
		total, par []int64
	}{
		{
			nom: "26602661 : le point du camp 1 n'est pas remplace par un zero",
			recs: []types.StatRecord{
				emissionEquipe(8, 206864, 1, 0),
				emissionEquipe(8, 221567, -69, 120),
				emissionEquipe(8, 237600, 0, 5),
			},
			total: []int64{1}, par: []int64{1},
		},
		{
			nom: "7de0b91d : le point du camp 0 n'est pas masque par un zero tardif",
			recs: []types.StatRecord{
				emissionEquipe(8, 283850, 0, 5),
				emissionEquipe(8, 434821, 1, 0),
				emissionEquipe(8, 551853, 0, 5),
			},
			total: []int64{1}, par: []int64{1},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			total := SeriesTotal(c.recs, ModeScoreComponent(), true, nil)[8]
			if got := valeursDe(total); !memesValeurs(got, c.total) {
				t.Errorf("SeriesTotal = %v, attendu %v", got, c.total)
			}
			par := SeriesByRound(c.recs, ModeScoreComponent(), true, nil)[8][0]
			if got := valeursDe(par); !memesValeurs(got, c.par) {
				t.Errorf("SeriesByRound = %v, attendu %v", got, c.par)
			}
		})
	}
}

// TestScoreEquipe_CanalBNulInchange — une suite d'equipe dont tous les canaux B sont nuls (le cas
// de 20 997 emissions sur 21 042) rend ce qu'elle rendait.
func TestScoreEquipe_CanalBNulInchange(t *testing.T) {
	recs := []types.StatRecord{
		emissionEquipe(6, 343374, 1, 0),
		emissionEquipe(6, 428527, 2, 0),
		emissionEquipe(6, 638129, 3, 0),
	}
	total := SeriesTotal(recs, ModeScoreComponent(), true, nil)[6]
	if got := valeursDe(total); !memesValeurs(got, []int64{1, 2, 3}) {
		t.Errorf("SeriesTotal = %v, attendu [1 2 3]", got)
	}
}

// TestScoreEquipe_LeCanalBDUnJoueurResteLu — la regle est celle du slot d'EQUIPE : un joueur
// d'Oddball re-emet le composant 0 a valeur A egale quand sa valeur B bouge
// ([SkullTicksComponent]), et ces emissions-la sont des lectures.
func TestScoreEquipe_LeCanalBDUnJoueurResteLu(t *testing.T) {
	recs := []types.StatRecord{
		emissionEquipe(10, 1000, 3, 0),
		emissionEquipe(10, 2000, 3, 7),
		emissionEquipe(10, 3000, 4, 7),
	}
	par := SeriesByRound(recs, SkullTicksComponent(), false, nil)[10][0]
	if got := valeursDe(par); !memesValeurs(got, []int64{3, 3, 4}) {
		t.Errorf("SeriesByRound(joueur) = %v, attendu [3 3 4] : l'emission a canal B non nul d'un JOUEUR est une lecture", got)
	}
}
