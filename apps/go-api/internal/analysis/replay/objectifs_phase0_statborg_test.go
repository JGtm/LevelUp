package replay

// objectifs_phase0_statborg_test.go — LE PONT STATBORG -> JOUEUR, PAR LE FILM SEUL.
//
// POURQUOI CE SECOND PONT EXISTE, ET CE QU'IL CORRIGE. `objectiveevents.SlotIdentity`
// apparie un slot statborg a une ligne de match par le triplet (frags, morts,
// assistances). C'est exact quand les compteurs du film atteignent leurs valeurs finales —
// et c'est mesure ici que ce n'est PAS toujours le cas : sur `64e8adfa` et `24dbb67d`,
// l'appariement rend 0 slot sur 8, alors qu'il en rend 8 sur 8 sur `530820e5` et
// `53ce4390`. Un film tronque (le Theater ne rend pas toujours la fin) suffit a le mettre
// en defaut : les compteurs s'arretent avant le total de l'API.
//
// LA VOIE DE REMPLACEMENT N'EMPRUNTE RIEN A L'API : les MORTS. Le statborg replique le
// compteur de morts de chaque joueur (`comp 2 B`, confirme 8/8 contre `match_participants`,
// cf. slotidentity.go) et le film porte par ailleurs son FIL DES MORTS, qui date chaque
// mort et NOMME sa victime par son xuid. Les deux sont sur l'horloge du MATCH. Apparier les
// INSTANTS plutot que les TOTAUX identifie donc le slot sans jamais consulter la base, et
// tient sur un film tronque : il suffit qu'assez de morts soient communes aux deux
// lectures.
//
// LA REGLE DE PRUDENCE EST CELLE DU DEPOT : un slot dont le meilleur candidat n'est pas
// NETTEMENT devant le suivant n'est pas apparie, et un xuid que deux slots se disputent
// n'est attribue a aucun des deux. Se taire vaut mieux qu'attribuer le drapeau au mauvais
// joueur — sur une carte, l'erreur serait invisible et credible.

import (
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// Emplacements de base du statborg. Memes valeurs que `objectiveevents` (coreKillsComp,
// coreAssistsComp) ; elles ne sont pas exportees, mais elles sont ETABLIES et documentees
// (frags en A et morts en B de comp 2, assistances en A de comp 3).
const (
	objCompCoreKills   = 2
	objCompCoreAssists = 3
)

// objAppariementMS : tolerance d'appariement entre l'instant d'une mort du fil et celui de
// la progression du compteur de morts du statborg. Meme ordre que `deathMatchWindowMS`
// (150 ms) qui borne deja le bruit d'horloge du pont bipede.
const objAppariementMS = 150

// objMargeAppariement : le meilleur candidat doit devancer le suivant d'au moins ce facteur.
// Sans marge, un slot se ferait attribuer un joueur sur une poignee de coincidences.
const objMargeAppariement = 2

// objMinAppariements : nombre minimal de morts communes pour qu'un appariement compte.
const objMinAppariements = 3

// objSerie rend, pour un emplacement donne, la suite chronologique croissante des valeurs
// d'un slot, debarrassee des emissions negatives (ancrages parasites) et des reculs.
//
// C'est la lecture minimale qui suffit ici : on ne cherche ni a compter des evenements ni a
// les nommer, seulement a savoir QUAND le compteur progresse et OU il finit.
func objSerie(recs []objectiveevents.StatRecord, comp int, cote byte) map[int][]objPoint {
	brut := map[int][]objPoint{}
	for _, r := range recs {
		if objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[comp]
		if !ok {
			continue
		}
		val := v.A
		if cote == 'B' {
			val = v.B
		}
		if val < 0 {
			continue
		}
		brut[r.Slot] = append(brut[r.Slot], objPoint{TimeMS: int64(r.TimeMS), Val: val})
	}
	out := make(map[int][]objPoint, len(brut))
	for slot, pts := range brut {
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
		var mont []objPoint
		prev := int64(-1)
		for _, p := range pts {
			if p.Val < prev {
				continue
			}
			prev = p.Val
			mont = append(mont, p)
		}
		out[slot] = mont
	}
	return out
}

// objPoint est une emission datee d'un emplacement de statistique.
type objPoint struct {
	TimeMS int64
	Val    int64
}

// objInstantsDeProgression rend un instant par UNITE gagnee par le compteur.
func objInstantsDeProgression(pts []objPoint) []int64 {
	var out []int64
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.Val; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}

// objFinal rend la derniere valeur de la serie (0 si vide).
func objFinal(pts []objPoint) int64 {
	if len(pts) == 0 {
		return 0
	}
	return pts[len(pts)-1].Val
}

// objTriplets rend, par slot statborg, le triplet final (frags, morts, assistances).
// Diagnostic : c'est lui qui montre POURQUOI l'appariement par totaux echoue sur un film.
func objTriplets(recs []objectiveevents.StatRecord) map[int][3]int64 {
	frags := objSerie(recs, objCompCoreKills, 'A')
	morts := objSerie(recs, objCompCoreKills, 'B')
	assist := objSerie(recs, objCompCoreAssists, 'A')
	out := map[int][3]int64{}
	for slot := range frags {
		out[slot] = [3]int64{objFinal(frags[slot]), objFinal(morts[slot]), objFinal(assist[slot])}
	}
	return out
}

// objStatPont apparie chaque slot statborg a un xuid par les INSTANTS DE MORT.
//
// Rend la table, le nombre de slots vus et le nombre d'appariements refuses pour marge
// insuffisante ou pour conflit — les trois se publient ensemble.
func objStatPont(recs []objectiveevents.StatRecord, deaths []Death) (map[int]uint64, int, int) {
	mortsFil := map[uint64][]int64{}
	for _, d := range deaths {
		mortsFil[d.XUID] = append(mortsFil[d.XUID], d.TimeMS)
	}
	for x := range mortsFil {
		sort.Slice(mortsFil[x], func(i, j int) bool { return mortsFil[x][i] < mortsFil[x][j] })
	}
	series := objSerie(recs, objCompCoreKills, 'B')
	revendique := map[int]uint64{}
	vus, refuses := 0, 0
	for slot, pts := range series {
		vus++
		instants := objInstantsDeProgression(pts)
		meilleur, second, gagnant := 0, 0, uint64(0)
		for x, fil := range mortsFil {
			n := objCoincidences(instants, fil)
			if n > meilleur {
				meilleur, second, gagnant = n, meilleur, x
			} else if n > second {
				second = n
			}
		}
		if meilleur < objMinAppariements || meilleur < objMargeAppariement*second {
			refuses++
			continue
		}
		revendique[slot] = gagnant
	}
	return objSansConflit(revendique, &refuses), vus, refuses
}

// objSansConflit ecarte les xuid revendiques par plusieurs slots.
func objSansConflit(revendique map[int]uint64, refuses *int) map[int]uint64 {
	parXUID := map[uint64][]int{}
	for slot, x := range revendique {
		parXUID[x] = append(parXUID[x], slot)
	}
	out := make(map[int]uint64, len(revendique))
	for x, slots := range parXUID {
		if len(slots) != 1 {
			*refuses += len(slots)
			continue
		}
		out[slots[0]] = x
	}
	return out
}

// objCoincidences compte les instants appariables entre deux series, chaque instant du fil
// ne servant qu'une fois (appariement glouton par ordre croissant).
func objCoincidences(a, b []int64) int {
	utilise := make([]bool, len(b))
	n := 0
	for _, t := range a {
		best, bd := -1, int64(objAppariementMS+1)
		for i, u := range b {
			if utilise[i] {
				continue
			}
			d := u - t
			if d < 0 {
				d = -d
			}
			if d < bd {
				bd, best = d, i
			}
		}
		if best >= 0 {
			utilise[best] = true
			n++
		}
	}
	return n
}

// objIdentityStrings convertit la table slot -> xuid dans la forme qu'attend
// `IdentifyNamedEvents`.
func objIdentityStrings(m map[int]uint64) map[int]string {
	out := make(map[int]string, len(m))
	for slot, x := range m {
		out[slot] = strconv.FormatUint(x, 10)
	}
	return out
}

// TestObjectifsPhase0PontStatborg — le CONTROLE du pont de remplacement, et sa confrontation
// au pont par triplets la ou celui-ci fonctionne.
//
// Deux ponts totalement disjoints (des TOTAUX venus de l'API contre des INSTANTS venus du
// film) qui rendent la meme table sur les films ou les deux repondent : c'est ce qui donne
// le droit d'employer le second la ou le premier se tait.
func TestObjectifsPhase0PontStatborg(t *testing.T) {
	root := objRequireRoot(t)
	joues := 0
	for _, id := range append(append([]string{}, objCTFFilms...), objBallFilm) {
		f := objCorpus[id]
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		objCheckTripletsUniques(t, f)
		recs := objectiveevents.StatRecords(src)
		b := objBridgeOf(t, root, id)
		parInstants, vus, refuses := objStatPont(recs, b.Deaths)
		parTriplets := objSlotIdentityDe(src, f)
		accords, desaccords := objCompareTables(parInstants, parTriplets)
		t.Logf("%s : statborg %d slots vus ; pont par INSTANTS %d apparies (%d refuses) ; "+
			"pont par TRIPLETS %d apparies ; recoupement %d accords / %d desaccords",
			id, vus, len(parInstants), refuses, len(parTriplets), accords, desaccords)
		for slot, tri := range objTriplets(recs) {
			t.Logf("%s : slot %d — film (frags %d, morts %d, assist %d)", id, slot, tri[0], tri[1], tri[2])
		}
		if desaccords > 0 {
			t.Errorf("%s : %d slots ou les deux ponts se contredisent — aucun des deux n'est employable",
				id, desaccords)
		}
	}
	if joues == 0 {
		t.Skipf("aucun film du corpus dans le cache (%s=%q)", objFilmEnv, root)
	}
}

// objSlotIdentityDe rend le pont par triplets (celui d'`objectiveevents`).
func objSlotIdentityDe(src objectiveevents.FilmSource, f objFilm) map[int]string {
	lines := make([]objectiveevents.PlayerLine, 0, len(f.Players))
	for _, p := range f.Players {
		lines = append(lines, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	return objectiveevents.SlotIdentity(src, lines)
}

// objCompareTables compte les slots ou les deux ponts disent la meme chose, et ceux ou ils
// se contredisent (un slot absent d'une table n'est ni l'un ni l'autre).
func objCompareTables(instants map[int]uint64, triplets map[int]string) (accords, desaccords int) {
	for slot, x := range instants {
		other, ok := triplets[slot]
		if !ok {
			continue
		}
		if other == strconv.FormatUint(x, 10) {
			accords++
		} else {
			desaccords++
		}
	}
	return accords, desaccords
}
