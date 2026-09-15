//go:build research

package objectiveevents

// e1911_manches_mesure_research_test.go — INSTRUMENT 1.9.11, LA MESURE : QUE VAUT LE
// DESIGNATEUR DE MANCHE QUAND UN MATCH DEPASSE SON TEMPS REGLEMENTAIRE ?
//
// (Le rapport — tout ce qui imprime — vit dans `e1911_manches_rapport_research_test.go`.)
//
// # LA QUESTION (plan, famille 1.9, item 1.9.11)
//
// Sur `fb1a1a72` (CTF:Arena, 802 s au registre pour 720 s reglementaires) le film ecrit un
// designateur de manche `2` sur 148 enregistrements, et la garde `contiguousRounds` le JETTE
// parce que la manche 1 est absente. Hypothese de l'utilisateur du 2026-09-14 : « ca ressemble
// a une prolongation ; si le temps reglementaire se finit sur une egalite ca peut arriver, mais
// je ne sais pas si c'est le seul critere ».
//
// L'instrument mesure les DEUX SENS de cette hypothese sur TOUT le cache, pas sur un temoin :
//
//	sens 1  tout match dont un designateur est JETE par la garde a-t-il depasse son temps
//	        reglementaire, et son score etait-il a EGALITE a la fin de ce temps ?
//	sens 2  tout match a egalite a la fin du temps reglementaire porte-t-il un tel designateur ?
//
// # LE TEMPS REGLEMENTAIRE VIENT DU REGISTRE, PAS DE L'HORLOGE DU FILM (piege de 0.D.1 bis)
//
// Les deux horloges ne coincident pas : `fb1a1a72` vaut 757 s au `frameCount` du document,
// 814 s a l'horloge du film et 802 s a `match_registry.duration_seconds`. C'est la duree du
// MATCH qui decide d'une prolongation (`analysis.ComputeOvertime`), donc l'entree de
// l'instrument est un TSV produit depuis la base d'oracle en lecture seule :
//
//	short8 <TAB> variante <TAB> duree_s <TAB> reglementaire_s <TAB> score_camp_0 <TAB> score_camp_1
//
// L'INSTANT DE FIN DU TEMPS REGLEMENTAIRE SE POSE PAR L'ANCRE DE FIN, et c'est un choix
// mesure : l'origine du film (premier paquet) ne coincide avec le debut du match sur aucun
// film (`film_match_start_ms` est NULL sur `fb1a1a72`), alors que la FIN des deux horloges est
// le meme evenement — le match s'arrete. Le depassement `duree - reglementaire` se retranche
// donc du DERNIER enregistrement du film. L'ancre de debut (premier enregistrement +
// reglementaire) est imprimee A COTE, pour que l'ecart entre les deux se lise.
//
// # UN CAMP RESTE A ZERO N'EMET RIEN, ET LE CONFONDRE AVEC UN CAMP ABSENT FAUSSE L'EGALITE
//
// Un composant n'est reemis QUE lorsqu'il change (en-tete de `statborg.go`) : un camp qui ne
// marque jamais ne porte AUCUNE emission de score de mode. Les slots d'equipe sont donc
// recenses sur TOUS les enregistrements, et un slot sans serie vaut ZERO — sans quoi
// `fb1a1a72` (final 0-1) n'aurait qu'un seul camp mesure et ne pourrait jamais sortir « a
// egalite ».
//
// USAGE (gardes par environnement, saute sans elles) :
//
//	FILM_CACHE_ROOT=C:/.../LevelUp-go-migration/data/cache \
//	E1911_CORPUS=C:/.../corpus_manches.tsv \
//	  go test -tags research ./internal/analysis/objectiveevents/ -run E1911 -v -timeout 60m
//
// Aucun code de production n'est touche : l'instrument appelle les memes fonctions que
// [RealRounds] et [contiguousRounds].

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// e1911CorpusEnv nomme le TSV d'entree (cf. l'en-tete pour ses colonnes).
const e1911CorpusEnv = "E1911_CORPUS"

// e1911TrancheMS est le pas de l'histogramme temporel des designateurs.
const e1911TrancheMS = 60_000

// e1911TrouMS separe deux AMAS d'un meme designateur (cf. [e1911Bloc]). Trente secondes est un
// ordre de grandeur, pas un reglage : les deux amas de `fb1a1a72` sont separes de 540 s et les
// emissions d'un amas se suivent a moins d'une seconde — tout seuil entre 2 s et 400 s rend le
// meme decoupage sur ce corpus. L'instrument imprime les blocs pour que ce soit verifiable.
const e1911TrouMS = 30_000

// e1911Match est une ligne du TSV d'entree.
type e1911Match struct {
	ID       string
	Variante string
	Duree    int // secondes, `match_registry.duration_seconds`
	Regl     int // secondes, `regulation.toml [regulation_seconds]`
	S0, S1   int // scores finaux des deux camps au registre
}

// Depassement rend les secondes au-dela du temps reglementaire (negatif = fini dans le temps).
func (m e1911Match) Depassement() int { return m.Duree - m.Regl }

// e1911Designateur porte ce qu'on veut savoir d'une valeur du champ de 5 bits.
type e1911Designateur struct {
	Valeur   int
	Records  int
	Joueur   int // enregistrements de slot JOUEUR
	Equipe   int // enregistrements de slot d'EQUIPE
	Premier  int // instant du premier enregistrement portant cette valeur, ms film
	Dernier  int
	Courants int   // premier composant dans 0-27  (`current-round-value`)
	Finalise int   // premier composant dans 28-55 (`finalized-rounds-values`)
	Issues   int   // premier composant 56 (`round-outcomes`)
	Entree   int   // premier composant 57 (`entry-index-and-type`)
	Apres    int   // enregistrements APRES la fin du temps reglementaire (ancre de fin)
	Tranches []int // histogramme par tranche de [e1911TrancheMS]
	Blocs    []e1911Bloc
	// Declarants est le nombre de SLOTS DISTINCTS qui declarent ce designateur, et `Part` sa
	// part du nombre maximal de declarants d'une manche du meme film, en pour cent.
	//
	// C'EST LE CONSENSUS, et il est deja au depot : [chainedRounds] (`round_bounds.go`) retire
	// de la chaine des bornes toute manche dont MOINS DE LA MOITIE des slots parlent, parce
	// qu'« un debut de manche est un CONSENSUS, et un slot n'en est pas un ». La mesure du
	// 2026-09-06 y separe franchement les deux populations : une manche est declaree par les
	// DIX slots, ou par un seul. L'instrument le mesure ICI pour le fait « quelles manches sont
	// REELLES », que la garde de contiguite decide encore par decret d'ordre.
	Declarants int
	Part       int
	instants   []int        // instants bruts, pour le decoupage en blocs
	slots      map[int]bool // slots distincts declarants
}

// e1911Bloc est un AMAS d'enregistrements d'un meme designateur : une suite d'instants dont deux
// voisins ne sont jamais separes de plus de [e1911TrouMS].
//
// POURQUOI DECOUPER. Le premier enregistrement d'un designateur ne date PAS le debut de sa
// manche : `fb1a1a72` porte son designateur 2 en DEUX amas (94 enregistrements entre 60 et
// 180 s, 54 entre 720 et 840 s), et `eba1e63f` son designateur 1 des 191 s. Dater la manche au
// premier enregistrement lit alors le score d'un autre moment du match. Le DERNIER amas est
// celui qui porte la phase de fin ; c'est lui qui date la manche.
type e1911Bloc struct {
	T0, T1  int
	Records int
}

// e1911Bilan est la ligne de synthese d'un film.
type e1911Bilan struct {
	M         e1911Match
	Records   int
	Desig     []e1911Designateur
	Vues      []int // designateurs vus, tries
	Retenues  []int // ce que [RealRounds] retient AUJOURD'HUI (garde de contiguite comprise)
	Jetees    []int // designateurs vus que [RealRounds] jette
	SansGarde []int // ce que les DEUX criteres d'admission retiennent, garde RETIREE
	Ecart     []int // `SansGarde` moins `Retenues` : ce que le retrait de la garde ajouterait
	FilmT0    int
	FilmT1    int
	ReglFin   int                          // instant de fin du temps reglementaire, ancre de FIN, ms film
	ReglDebut int                          // idem, ancre de DEBUT
	Slots     []int                        // slots d'EQUIPE vus, tries
	Pistes    map[int]map[int][]ScorePoint // score de mode par slot d equipe PUIS par manche
	ScoreFin  []int                        // score de mode par slot d'equipe a `ReglFin`
	ScoreDeb  []int                        // idem a `ReglDebut`
	// Manche2 est le plus petit designateur MATERIEL strictement positif du film, -1 s'il n'y
	// en a pas ; `Manche2T` est l'instant de son PREMIER enregistrement et `Manche2Score` le
	// score de chaque camp JUSTE AVANT cet instant.
	//
	// C'est la mesure qui ne doit RIEN a l'alignement des horloges : le film date lui-meme le
	// debut de la seconde manche, et le score a cet instant se lit sur la meme horloge.
	Manche2      int
	Manche2T     int
	Manche2Score []int
}

// EgaliteFin dit si les deux camps sont a egalite a la fin du temps reglementaire (ancre de fin).
func (b e1911Bilan) EgaliteFin() bool { return e1911Egalite(b.ScoreFin) }

// EgaliteDebut dit la meme chose a l'ancre de debut.
func (b e1911Bilan) EgaliteDebut() bool { return e1911Egalite(b.ScoreDeb) }

// e1911Egalite dit si tous les camps mesures portent la meme valeur (et qu'il y en a au moins 2).
func e1911Egalite(s []int) bool {
	if len(s) < 2 {
		return false
	}
	for _, v := range s[1:] {
		if v != s[0] {
			return false
		}
	}
	return true
}

// e1911Corpus lit le TSV d'entree.
func e1911Corpus(t *testing.T) []e1911Match {
	t.Helper()
	path := os.Getenv(e1911CorpusEnv)
	if path == "" {
		t.Skipf("%s absent : instrument saute", e1911CorpusEnv)
	}
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'operateur de l'instrument
	if err != nil {
		t.Fatalf("corpus %s : %v", path, err)
	}
	defer func() { _ = f.Close() }()
	var out []e1911Match
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m, ok := e1911Ligne(line)
		if !ok {
			t.Fatalf("corpus %s : ligne illisible %q", path, line)
		}
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("corpus %s : %v", path, err)
	}
	return out
}

// e1911Ligne decoupe une ligne du TSV.
func e1911Ligne(line string) (e1911Match, bool) {
	c := strings.Split(line, "\t")
	if len(c) < 6 {
		return e1911Match{}, false
	}
	n := make([]int, 4)
	for i, s := range c[2:6] {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return e1911Match{}, false
		}
		n[i] = v
	}
	return e1911Match{ID: c[0], Variante: c[1], Duree: n[0], Regl: n[1], S0: n[2], S1: n[3]}, true
}

// e1911Mesure assemble le bilan d'un film.
func e1911Mesure(m e1911Match, recs []StatRecord) e1911Bilan {
	b := e1911Bilan{M: m, Records: len(recs)}
	if len(recs) == 0 {
		return b
	}
	b.FilmT0, b.FilmT1 = recs[0].TimeMS, recs[0].TimeMS
	for _, r := range recs {
		if r.TimeMS < b.FilmT0 {
			b.FilmT0 = r.TimeMS
		}
		if r.TimeMS > b.FilmT1 {
			b.FilmT1 = r.TimeMS
		}
	}
	b.Vues = e1911Triees(presentRounds(recs))
	b.Retenues = e1911Triees(RealRounds(recs))
	b.Jetees = e1911Difference(b.Vues, b.Retenues)
	b.SansGarde = e1911SansGarde(recs)
	b.Ecart = e1911Difference(b.SansGarde, b.Retenues)
	b.ReglFin = b.FilmT1 - m.Depassement()*1000
	b.ReglDebut = b.FilmT0 + m.Regl*1000
	b.Slots, b.Pistes = e1911PistesEquipe(recs)
	b.ScoreFin = e1911ScoreALInstant(b, b.ReglFin)
	b.ScoreDeb = e1911ScoreALInstant(b, b.ReglDebut)
	b.Desig = e1911Designateurs(recs, b.ReglFin, b.FilmT1)
	b.Manche2, b.Manche2T = e1911PremiereSeconde(b.Desig)
	if b.Manche2 >= 0 {
		b.Manche2Score = e1911ScoreALInstant(b, b.Manche2T-1)
	}
	return b
}

// e1911PremiereSeconde rend le plus petit designateur MATERIEL strictement positif et l'instant
// ou commence son DERNIER amas (-1, 0 quand le film n'en ecrit aucun) — cf. [e1911Bloc] pour
// laquelle le premier enregistrement ne date pas la manche.
func e1911PremiereSeconde(desig []e1911Designateur) (int, int) {
	for _, d := range desig {
		if d.Valeur > 0 && d.Records >= statMinRoundRecords && len(d.Blocs) > 0 {
			return d.Valeur, d.Blocs[len(d.Blocs)-1].T0
		}
	}
	return -1, 0
}

// e1911SansGarde rend ce que retiendraient les DEUX CRITERES D'ADMISSION SEULS — suite
// coherente du score de mode ([statMinRoundRun]) OU matiere ([materialRounds]) — c'est-a-dire
// exactement ce que rendrait [RealRounds] si la garde de contiguite etait RETIREE.
//
// C'est la grandeur que le lot doit mesurer AVANT de retirer la garde : l'ecart avec
// `Retenues` est, film par film, ce que le retrait changerait a la production.
func e1911SansGarde(recs []StatRecord) []int {
	runs := e1911RunsParManche(recs)
	material := materialRounds(recs)
	out := map[int]bool{}
	for round := 0; round <= statMaxRound; round++ {
		if runs[round] >= statMinRoundRun || material[round] {
			out[round] = true
		}
	}
	if len(out) == 0 {
		out[0] = true
	}
	return e1911Triees(out)
}

// e1911RunsParManche rend, par manche, la plus longue suite coherente du score de mode tous
// slots confondus — exactement ce que [RealRounds] calcule avant d'appeler [contiguousRounds].
func e1911RunsParManche(recs []StatRecord) map[int]int {
	type cle struct{ slot, round int }
	series := map[cle][]ScorePoint{}
	for _, r := range recs {
		v, ok := r.Comps[modeScoreComp]
		if !ok || !modeScoreInDomain(v) {
			continue
		}
		k := cle{r.Slot, r.Round}
		series[k] = append(series[k], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	out := map[int]int{}
	for k, pts := range series {
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
		if n := len(longestRun(pts, true)); n > out[k.round] {
			out[k.round] = n
		}
	}
	return out
}

// e1911PistesEquipe rend les slots d'EQUIPE vus (tous enregistrements confondus) et, pour
// chacun, sa piste de score de mode MANCHE PAR MANCHE, filtree comme la production.
//
// LA PISTE EST PAR MANCHE, ET C'EST INDISPENSABLE : le score de mode d'un slot REPART DE ZERO a
// chaque manche (en-tete de `statborg.go`). Mettre toutes les manches dans une seule suite puis
// la filtrer par la plus longue sous-suite croissante EFFACE une manche entiere — mesure du
// 2026-09-16 sur `521c5c38` : la piste mise a plat ne gardait qu'UN point par camp alors que le
// match finit 2-1. La lecture passe donc par [rawSeriesByRound], la MEME porte que
// [SeriesByRound] et [SeriesTotal], mais SANS le filtre [RealRounds] : c'est precisement la
// garde qu'on mesure, elle ne peut pas entrer dans sa propre mesure.
//
// Un slot d'equipe SANS aucune emission de score reste dans `slots` avec une piste vide : il
// vaut zero, il n'est pas absent (cf. l'en-tete du fichier).
func e1911PistesEquipe(recs []StatRecord) ([]int, map[int]map[int][]ScorePoint) {
	vus := map[int]bool{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			vus[r.Slot] = true
		}
	}
	slots := make([]int, 0, len(vus))
	for s := range vus {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	raw := rawSeriesByRound(recs, ModeScoreComponent.key(), true)
	out := make(map[int]map[int][]ScorePoint, len(slots))
	for _, s := range slots {
		parManche := map[int][]ScorePoint{}
		for round, pts := range raw[s] {
			sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
			if kept := longestRun(pts, true); len(kept) > 0 {
				parManche[round] = kept
			}
		}
		out[s] = parManche
	}
	return slots, out
}

// e1911ScoreALInstant rend, par slot d'EQUIPE et dans l'ordre des slots, le score de mode CUMULE
// sur les manches a cet instant ou avant (zero quand le camp est muet).
//
// Le cumul est celui de la production ([cumulateRounds]) : chaque manche apporte la derniere
// valeur qu'elle a atteinte AVANT `tMS`, et une manche qui commence apres n'apporte rien.
func e1911ScoreALInstant(b e1911Bilan, tMS int) []int {
	out := make([]int, 0, len(b.Slots))
	for _, s := range b.Slots {
		total := 0
		for _, pts := range b.Pistes[s] {
			total += e1911DerniereValeur(pts, tMS)
		}
		out = append(out, total)
	}
	return out
}

// e1911DerniereValeur rend la valeur du dernier point a `tMS` ou avant (0 si aucun).
func e1911DerniereValeur(pts []ScorePoint, tMS int) int {
	v := 0
	for _, p := range pts {
		if p.TimeMS > tMS {
			break
		}
		v = int(p.Value)
	}
	return v
}

// e1911Designateurs recense, par valeur du champ de manche, ce que les enregistrements portent :
// leur compte par nature de slot, leur fenetre, leur famille de composant, la part POSTERIEURE a
// la fin du temps reglementaire, et leur histogramme temporel.
func e1911Designateurs(recs []StatRecord, reglFin, filmT1 int) []e1911Designateur {
	tranches := filmT1/e1911TrancheMS + 1
	par := map[int]*e1911Designateur{}
	for _, r := range recs {
		d := par[r.Round]
		if d == nil {
			d = &e1911Designateur{
				Valeur: r.Round, Premier: r.TimeMS, Dernier: r.TimeMS,
				Tranches: make([]int, tranches), slots: map[int]bool{},
			}
			par[r.Round] = d
		}
		e1911Note(d, r, reglFin)
	}
	ref := 0
	for _, d := range par {
		if len(d.slots) > ref {
			ref = len(d.slots)
		}
	}
	out := make([]e1911Designateur, 0, len(par))
	for _, d := range par {
		d.Blocs = e1911Blocs(d.instants)
		d.Declarants = len(d.slots)
		if ref > 0 {
			d.Part = d.Declarants * 100 / ref
		}
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Valeur < out[j].Valeur })
	return out
}

// e1911Blocs decoupe une liste d'instants en amas separes de plus de [e1911TrouMS].
func e1911Blocs(instants []int) []e1911Bloc {
	if len(instants) == 0 {
		return nil
	}
	sort.Ints(instants)
	out := []e1911Bloc{{T0: instants[0], T1: instants[0], Records: 1}}
	for _, t := range instants[1:] {
		cur := &out[len(out)-1]
		if t-cur.T1 > e1911TrouMS {
			out = append(out, e1911Bloc{T0: t, T1: t, Records: 1})
			continue
		}
		cur.T1 = t
		cur.Records++
	}
	return out
}

// e1911Note range un enregistrement dans le recensement de son designateur.
func e1911Note(d *e1911Designateur, r StatRecord, reglFin int) {
	d.Records++
	d.instants = append(d.instants, r.TimeMS)
	d.slots[r.Slot] = true
	if IsTeamSlot(r.Slot) {
		d.Equipe++
	} else {
		d.Joueur++
	}
	if r.TimeMS < d.Premier {
		d.Premier = r.TimeMS
	}
	if r.TimeMS > d.Dernier {
		d.Dernier = r.TimeMS
	}
	if r.TimeMS > reglFin {
		d.Apres++
	}
	if k := r.TimeMS / e1911TrancheMS; k >= 0 && k < len(d.Tranches) {
		d.Tranches[k]++
	}
	switch e1911Famille(r) {
	case 0:
		d.Courants++
	case 1:
		d.Finalise++
	case 2:
		d.Issues++
	default:
		d.Entree++
	}
}

// e1911Famille classe un enregistrement par le PLUS PETIT index de composant qu'il porte, selon
// le registre ECS de l'archetype 6 (0-27 `current-round-value`, 28-55 `finalized-rounds-values`,
// 56 `round-outcomes`, 57 `entry-index-and-type`).
func e1911Famille(r StatRecord) int {
	low := statMaxComp
	for i := range r.Comps {
		if i < low {
			low = i
		}
	}
	switch {
	case low < 28:
		return 0
	case low < 56:
		return 1
	case low == 56:
		return 2
	default:
		return 3
	}
}

// e1911Triees rend les cles vraies d'un ensemble de manches, triees.
func e1911Triees(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for r, ok := range m {
		if ok {
			out = append(out, r)
		}
	}
	sort.Ints(out)
	return out
}

// e1911Difference rend les elements de `a` absents de `b`.
func e1911Difference(a, b []int) []int {
	in := make(map[int]bool, len(b))
	for _, v := range b {
		in[v] = true
	}
	out := []int{}
	for _, v := range a {
		if !in[v] {
			out = append(out, v)
		}
	}
	return out
}
