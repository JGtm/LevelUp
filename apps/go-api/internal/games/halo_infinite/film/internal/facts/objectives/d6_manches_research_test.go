package objectives

// d6_manches_research_test.go — INSTRUMENT D6 : D OU VIENNENT LES MANCHES D UN FILM, SLOT PAR SLOT ?
//
// # LA QUESTION
//
// Le croisement au corpus gate du lot 0.A a releve `coverage.score.rounds` 3 -> 1 sur
// `fb1a1a72`, un CTF que le registre des reports tient pour MULTI-MANCHE depuis le 2026-09-06.
// Deux lectures opposees, et rien ne les separe sans mesure : correction de manches FANTOMES,
// ou perte d information sur un film reellement multi-manche.
//
// Le registre (ligne « Les CTF MULTI-MANCHE ne nomment presque aucun porteur », mesure du
// 2026-09-08) dit de CE film : « les enregistrements de slot JOUEUR declarent TOUS la manche 0
// [...] tandis que les manches viennent des slots d EQUIPE ». [materialRounds] EXCLUT les slots
// d equipe ; [RealRounds] les compte en revanche dans sa suite coherente. L instrument releve
// donc les DEUX cotes, slot par slot, et rejoue la decision de [contiguousRounds] manche par
// manche pour nommer le critere qui tranche.
//
// # POURQUOI ICI, ET PAS DANS `grammar` SOUS `CHUNK00_FILMS`
//
// Les enregistrements statborg ne sont lus que dans les chunks DECRITS PAR LE MANIFESTE
// ([manifestChunks], `film.go`) : un film charge par `source.LoadDir(dir, nil)` — la forme
// que prend `CHUNK00_FILMS` — n a aucun type de chunk, donc ZERO enregistrement, et la mesure
// serait vide sans le dire. L instrument passe donc par `filmcache.LoadFilm` (manifeste +
// chunks), exactement comme la production, via le `newDiskFilm` deja present dans ce paquet.
//
// Aucun code de production n est touche : l instrument appelle les memes fonctions que
// [RealRounds].
//
// USAGE (garde par environnement, saute sans elle) :
//
//	FILM_CACHE_ROOT=C:/.../LevelUp-go-migration/data/cache D6_FILMS=fb1a1a72,64e8adfa \
//	  go test ./internal/games/halo_infinite/film/internal/facts/objectives/ -run D6MatiereDesManches -v

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// d6FilmsEnv nomme les films a mesurer (short8, separes par des virgules).
const d6FilmsEnv = "D6_FILMS"

// d6Cle identifie une serie : un slot dans une manche.
type d6Cle struct{ slot, round int }

// d6Serie porte ce qu on veut savoir d un couple (slot, manche).
type d6Serie struct {
	records   int // enregistrements portant ce couple
	avecComp0 int // ... dont ceux qui transportent le composant de score de mode
	horsDom   int // ... dont ceux que la garde de domaine rejette
	run       int // plus longue suite STRICTEMENT croissante du score de mode
	premier   int64
	dernier   int64
	t0, t1    int
}

// TestD6MatiereDesManches imprime, film par film, la matiere de chaque manche et la decision.
func TestD6MatiereDesManches(t *testing.T) {
	if cacheRoot() == "" {
		t.Skipf("%s absent : instrument saute", filmCacheEnv)
	}
	for _, id := range d6Films(t) {
		film, ok := newDiskFilm(t, id)
		if !ok {
			t.Logf("%s : ECARTE (absent du cache)", id)
			continue
		}
		recs := StatRecords(film)
		t.Logf("=== %s : %d enregistrement(s) statborg ===", id, len(recs))
		d6ImprimeSeries(t, recs)
		d6ImprimeDecision(t, recs)
	}
}

// d6Films lit la liste des films demandes.
func d6Films(t *testing.T) []string {
	t.Helper()
	v := os.Getenv(d6FilmsEnv)
	if v == "" {
		t.Skipf("%s absent : instrument saute", d6FilmsEnv)
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// d6Series construit la table (slot, manche) -> mesure.
func d6Series(recs []StatRecord) map[d6Cle]*d6Serie {
	pts := map[d6Cle][]ScorePoint{}
	out := map[d6Cle]*d6Serie{}
	for _, r := range recs {
		k := d6Cle{r.Slot, r.Round}
		s := out[k]
		if s == nil {
			s = &d6Serie{t0: r.TimeMS, t1: r.TimeMS}
			out[k] = s
		}
		d6Note(s, r)
		if v, ok := r.Comps[modeScoreComp]; ok && modeScoreInDomain(v) {
			pts[k] = append(pts[k], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
		}
	}
	for k, p := range pts {
		sort.SliceStable(p, func(i, j int) bool { return p[i].TimeMS < p[j].TimeMS })
		run := longestRun(p, true)
		s := out[k]
		s.run = len(run)
		if len(run) > 0 {
			s.premier, s.dernier = run[0].Value, run[len(run)-1].Value
		}
	}
	return out
}

// d6Note range un enregistrement dans la mesure de son couple.
func d6Note(s *d6Serie, r StatRecord) {
	s.records++
	if r.TimeMS < s.t0 {
		s.t0 = r.TimeMS
	}
	if r.TimeMS > s.t1 {
		s.t1 = r.TimeMS
	}
	v, ok := r.Comps[modeScoreComp]
	if !ok {
		return
	}
	s.avecComp0++
	if !modeScoreInDomain(v) {
		s.horsDom++
	}
}

// d6ImprimeSeries imprime une ligne par couple (slot, manche), tries par manche puis slot.
func d6ImprimeSeries(t *testing.T, recs []StatRecord) {
	t.Helper()
	series := d6Series(recs)
	cles := make([]d6Cle, 0, len(series))
	for k := range series {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].round != cles[j].round {
			return cles[i].round < cles[j].round
		}
		return cles[i].slot < cles[j].slot
	})
	t.Logf("  %-6s %-5s %-7s %8s %8s %8s %6s %15s %s",
		"manche", "slot", "nature", "records", "avecC0", "horsDom", "suite", "score", "fenetre ms")
	for _, k := range cles {
		s := series[k]
		nature := "joueur"
		if IsTeamSlot(k.slot) {
			nature = "EQUIPE"
		}
		t.Logf("  %-6d %-5d %-7s %8d %8d %8d %6d %7d -> %-5d [%d, %d]",
			k.round, k.slot, nature, s.records, s.avecComp0, s.horsDom, s.run,
			s.premier, s.dernier, s.t0, s.t1)
	}
}

// d6ImprimeDecision rejoue, manche par manche, ce que [RealRounds] decide et POURQUOI.
func d6ImprimeDecision(t *testing.T, recs []StatRecord) {
	t.Helper()
	runs := d6RunsParManche(recs)
	material, present := materialRounds(recs), presentRounds(recs)
	real := RealRounds(recs)
	t.Logf("  --- decision (statMinRoundRun=%d, statMinRoundRecords=%d, part=%d%%, trou tolere=%d) ---",
		statMinRoundRun, statMinRoundRecords, statMinRoundRecordShare, statMaxEmptyRoundRun)
	t.Logf("  %-6s %8s %11s %9s %8s %s", "manche", "suite", "materielle", "presente", "RETENUE", "critere")
	for round := 0; round <= d6MaxManche(recs); round++ {
		crit := "aucun"
		switch {
		case runs[round] >= statMinRoundRun:
			crit = "suite coherente"
		case material[round]:
			crit = "matiere"
		}
		t.Logf("  %-6d %8d %11v %9v %8v %s", round, runs[round], material[round], present[round],
			real[round], crit)
	}
	t.Logf("  RealRounds = %v (%d manche(s))", d6Triees(real), len(real))
	t.Logf("  enregistrements de slot JOUEUR par manche : %v", d6JoueursParManche(recs))
	sansGarde, _ := contiguousRounds(runs, material, d6ToutPresent(recs))
	t.Logf("  SANS la garde de manche fantome (`vue && !present`, commit bb06cce5a) : %v",
		d6Triees(sansGarde))
	d6ImprimeRecouvrement(t, recs)
	d6ImprimeRepartition(t, recs)
	d6ImprimeComposants(t, recs)
	d6ImprimeDensite(t, recs)
}

// d6ToutPresent rend une table qui declare TOUTES les manches presentes.
//
// La passer a [contiguousRounds] NEUTRALISE exactement la garde de manche fantome
// (`vue && !present[round]`, lot 6.7-B1 item 4) sans toucher une ligne de production : la
// clause devient toujours fausse, et la chaine retrouve le comportement d avant le commit
// `bb06cce5a`. C est la mutation qui nomme le critere responsable d un ecart.
func d6ToutPresent(recs []StatRecord) map[int]bool {
	out := map[int]bool{}
	for round := 0; round <= statMaxRound; round++ {
		out[round] = true
	}
	_ = recs
	return out
}

// d6ImprimeDensite imprime, par manche, la DENSITE d emission (enregistrements par seconde de
// sa propre fenetre), rapportee a celle de la manche 0.
//
// C est le second discriminant entre une manche JOUEE et un ANCRAGE : une manche jouee fait
// emettre tous les slots pendant toute sa duree, donc sa densite vaut celle de la manche 0 ou
// davantage (les manches suivantes sont plus courtes) ; un ancrage est une goutte reguliere
// saupoudree sur tout le match.
func d6ImprimeDensite(t *testing.T, recs []StatRecord) {
	t.Helper()
	ref := d6Densite(recs, 0)
	t.Logf("  --- densite d emission (enregistrements par seconde de la fenetre de la manche) ---")
	for _, round := range d6ManchesVues(recs) {
		d := d6Densite(recs, round)
		part := 0.0
		if ref > 0 {
			part = d / ref * 100
		}
		t.Logf("  manche %d : %.2f enr./s (%.0f %% de la manche 0)", round, d, part)
	}
}

// d6Densite rend les enregistrements par seconde d une manche, sur sa propre fenetre.
func d6Densite(recs []StatRecord, round int) float64 {
	lo, hi, ok := d6Fenetre(recs, round)
	if !ok || hi <= lo {
		return 0
	}
	n := 0
	for _, r := range recs {
		if r.Round == round {
			n++
		}
	}
	return float64(n) / (float64(hi-lo) / 1000)
}

// d6ImprimeRecouvrement dit, pour chaque manche > 0, quelle part de ses enregistrements tombe
// DANS la fenetre temporelle de la manche 0.
//
// C est le discriminant entre une manche JOUEE et un ANCRAGE : des manches se jouent l une
// APRES l autre, donc leurs fenetres sont disjointes ; un ancrage fortuit est saupoudre sur
// toute la duree du match et se recouvre donc avec la manche 0. C est exactement le motif que
// l en-tete de [contiguousRounds] decrit pour `e60aaf06` (« un ancrage [...] dont l intervalle
// tombe ENTIEREMENT dans celui de la manche 0 »).
func d6ImprimeRecouvrement(t *testing.T, recs []StatRecord) {
	t.Helper()
	lo, hi, ok := d6Fenetre(recs, 0)
	if !ok {
		return
	}
	t.Logf("  --- recouvrement avec la fenetre de la manche 0 ([%d, %d] ms) ---", lo, hi)
	for _, round := range d6ManchesVues(recs) {
		if round == 0 {
			continue
		}
		dedans, total := 0, 0
		for _, r := range recs {
			if r.Round != round {
				continue
			}
			total++
			if r.TimeMS >= lo && r.TimeMS <= hi {
				dedans++
			}
		}
		rlo, rhi, _ := d6Fenetre(recs, round)
		t.Logf("  manche %d : %d/%d enregistrement(s) DANS la fenetre de la manche 0, fenetre propre [%d, %d]",
			round, dedans, total, rlo, rhi)
	}
}

// d6Fenetre rend les instants extremes des enregistrements d une manche.
func d6Fenetre(recs []StatRecord, round int) (lo, hi int, ok bool) {
	for _, r := range recs {
		if r.Round != round {
			continue
		}
		if !ok || r.TimeMS < lo {
			lo = r.TimeMS
		}
		if !ok || r.TimeMS > hi {
			hi = r.TimeMS
		}
		ok = true
	}
	return lo, hi, ok
}

// d6ManchesVues rend, triees, les manches qui portent au moins un enregistrement.
func d6ManchesVues(recs []StatRecord) []int {
	seen := map[int]bool{}
	for _, r := range recs {
		seen[r.Round] = true
	}
	return d6Triees(seen)
}

// d6RunsParManche rend, par manche, la plus longue suite coherente tous slots confondus —
// exactement ce que [RealRounds] calcule avant d appeler [contiguousRounds].
func d6RunsParManche(recs []StatRecord) map[int]int {
	out := map[int]int{}
	for k, s := range d6Series(recs) {
		if s.run > out[k.round] {
			out[k.round] = s.run
		}
	}
	return out
}

// d6JoueursParManche compte les enregistrements de slot JOUEUR par manche (le denominateur de
// [materialRounds]).
func d6JoueursParManche(recs []StatRecord) map[int]int {
	out := map[int]int{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			continue
		}
		out[r.Round]++
	}
	return out
}

// d6MaxManche rend le plus grand numero de manche vu.
func d6MaxManche(recs []StatRecord) int {
	high := 0
	for _, r := range recs {
		if r.Round > high {
			high = r.Round
		}
	}
	return high
}

// d6Triees rend les manches d un ensemble, triees, pour un affichage stable.
func d6Triees(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for r, ok := range m {
		if ok {
			out = append(out, r)
		}
	}
	sort.Ints(out)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// LOT 0.D.1 bis — CE QUE LE CHAMP « MANCHE » PORTE VRAIMENT
//
// Le registre du film NOMME les composants de l archetype 6 (cf.
// `.ai/V7.5/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §17.1) :
//
//	0..27   statborg-current-round-value-stat-component     (la manche EN COURS)
//	28..55  statborg-finalized-rounds-values-stat-component  (les manches TERMINEES)
//	56      statborg-round-outcomes-component                (l issue des manches)
//	57      statborg-entry-index-and-type-component
//
// Deux questions se tranchent donc SUR LE FILM, sans Ghidra :
//
//	(1) les enregistrements d une manche > 0 sont-ils GROUPES A LA FIN du match (prolongation
//	    ou manche jouee apres les autres) ou SAUPOUDRES sur toute sa duree (ancrage) ? La
//	    repartition par tranches de 60 s le dit sans appel au jugement ;
//	(2) le film porte-t-il les composants 28..55 et 56 — ceux qui n existent QUE s il y a
//	    plusieurs manches ? Leur presence est une ECRITURE du jeu, pas une inference.

// d6ImprimeRepartition imprime, par manche, le nombre d enregistrements par tranche de 60 s.
func d6ImprimeRepartition(t *testing.T, recs []StatRecord) {
	t.Helper()
	const seau = 60_000
	fin := 0
	for _, r := range recs {
		if r.TimeMS > fin {
			fin = r.TimeMS
		}
	}
	t.Logf("  --- repartition par tranche de 60 s (colonne = tranche, valeur = enregistrements) ---")
	for _, round := range d6ManchesVues(recs) {
		par := make([]int, fin/seau+2)
		for _, r := range recs {
			if r.Round == round {
				par[r.TimeMS/seau]++
			}
		}
		t.Logf("  manche %d : %v", round, par)
	}
}

// d6ImprimeComposants imprime, par manche, les index de composant vus et leur compte.
func d6ImprimeComposants(t *testing.T, recs []StatRecord) {
	t.Helper()
	t.Logf("  --- composants vus par manche (0-27 manche en cours, 28-55 manches finalisees, 56 issues) ---")
	for _, round := range d6ManchesVues(recs) {
		vus := map[int]int{}
		for _, r := range recs {
			if r.Round != round {
				continue
			}
			for i := range r.Comps {
				vus[i]++
			}
		}
		idx := make([]int, 0, len(vus))
		for i := range vus {
			idx = append(idx, i)
		}
		sort.Ints(idx)
		var courant, finalise, issues, autres int
		for _, i := range idx {
			switch {
			case i <= 27:
				courant += vus[i]
			case i <= 55:
				finalise += vus[i]
			case i == 56:
				issues += vus[i]
			default:
				autres += vus[i]
			}
		}
		t.Logf("  manche %d : index %v", round, idx)
		t.Logf("           lectures — manche en cours %d · manches FINALISEES %d · issues de manche %d · entry-index %d",
			courant, finalise, issues, autres)
	}
}
