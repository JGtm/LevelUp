package objectiveevents

// assaut_statborg_armement_test.go — LE STATBORG CONTRE L'INSTANT D'ARMEMENT, avec plancher.
//
// # POURQUOI ROUVRIR UN BALAYAGE DEJA FAIT — et ce qui est VRAIMENT neuf
//
// La phase A6 (`replay/assaut_a6_armement_test.go`, commit `15499a133` du 2026-08-31) a balaye
// **112 canaux = 28 composants x 4 canaux**, C et D COMPRIS : ils ont ete decodes dans CE MEME
// commit, et l'en-tete du test le dit mot pour mot. Rouvrir « les canaux jamais lus » n'a donc
// pas lieu d'etre — ils ont ete lus.
//
// Ce qui n'a PAS ete fait, et que cet instrument fait :
//
//	1. L'ORACLE. A6 jugeait contre les EXPLOSIONS, en exigeant une progression quelque part dans
//	   les 120 s precedentes ET une dispersion des delais <= 20 % de la mediane. Les delais
//	   etaient MIS EN COMMUN sur les neuf films — donc sur DEUX meches differentes (4,93 s a
//	   meche fixe, 16,2 s PAUSABLE en One Bomb). Une dispersion calculee sur ce melange explose
//	   PAR CONSTRUCTION : le critere ne pouvait pas etre tenu, meme par un vrai compteur
//	   d'armement. Ici on vise l'INSTANT d'armement lui-meme, film a mèche fixe seulement.
//	2. LES COMPOSANTS 28 A 57. L'archetype en compte 58 ; A6 s'arretait a 27. Trente composants
//	   x 4 canaux = 120 emplacements jamais regardes, nommement listes comme « reste a instruire »
//	   dans `.ai/V7.5/ETAT_ASSAUT_2026-08-31.md` §3.3.
//	3. LES SLOTS D'EQUIPE. A6 ne balayait que les slots de JOUEUR (« un armement est un geste de
//	   joueur »). Un canal qui distingue l'EQUIPE sans nommer le joueur reste un resultat utile,
//	   et il est ici mesure a part.
//	4. LA TRANSITION, pas seulement l'increment. A6 ne retenait que `val > precedent`. Un etat
//	   d'amorcage peut basculer 0 -> 1 puis revenir : c'est un CHANGEMENT, pas une progression.
//	5. LE PLANCHER. A6 n'en avait pas sur la branche compteur (seule la branche minuterie avait
//	   son temoin decale de 180 s). Sans plancher, un taux de couverture ne veut rien dire —
//	   c'est le piege n. 1 du chantier, et la sonde du pied vient de le montrer a nouveau
//	   (24 % de couverture pour 22 % de plancher).
//
// # LE PROTOCOLE, FIGE AVANT LA PREMIERE MESURE
//
// ORACLE. Instant d'armement = `explosion - 4930 ms`, la meche mesuree par l'anneau `ti=12 i14`
// et gardee en production comme meche de REFERENCE (`replay.BombFuseMS`), a +-600 ms. Il ne
// s'applique qu'aux variantes A MECHE FIXE : Neutral Bomb (5 films) et Husky Raid (1 film), soit
// **17 explosions**. Les trois films One Bomb (meche pausable) sont EXCLUS du critere ; ils
// alimentent le plancher, jamais le verdict.
//
// UNITE BALAYEE : (composant 0..57) x (canal A|B|C|D) x (famille de slot : JOUEUR ou EQUIPE) x
// (genre d'evenement : PROGRESSION `val > precedent`, ou TRANSITION `val != precedent`), l'etat
// etant tenu par couple (slot, manche) — un compteur repart de zero a chaque manche.
//
// CRITERE DE RETENUE, les DEUX exiges :
//
//	COUVERTURE    au moins un evenement de ce canal dans la fenetre de CHACUN des 17 armements ;
//	SELECTIVITE   taux en fenetre >= 3 fois le PLANCHER du meme canal, mesure sur 500 instants
//	              ALEATOIRES par film (graine fixe : la mesure se rejoue a l'identique).
//
// CE QUE L'INSTRUMENT NE PEUT PAS FAIRE, dit avant de lire son verdict : il ne voit que ce que
// le statborg REEMET. Un etat porte par un canal qui n'est jamais reemis pendant l'armement est
// invisible ici, et le negatif se lira « le statborg ne reemet rien a cet instant », jamais
// « l'armement n'existe pas ».
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, un seul
// film decode a la fois (les enregistrements d'un film sortent de portee avant le suivant).
// Pic mesure : sous le plafond de 2 Gio, 6,5 s pour les neuf films.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/objectiveevents/ -run AssautStatborgArmement -v -timeout 60m

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// asMaxComp borne le balayage a l'archetype entier (58 composants, 0..57).
	asMaxComp = 58
	// asValeurMax borne le domaine d'un compteur de geste, recopie de `a6ValeurMax` : un canal
	// qui porte des dizaines de milliers n'est pas un compteur de geste.
	asValeurMax = 1000
	// asTirages / asGraine : le plancher, et sa reproductibilite.
	asTirages = 500
	asGraine  = 20260904
)

// asCanal designe une unite balayee.
type asCanal struct {
	comp   int
	canal  byte // 'A' 'B' 'C' 'D'
	equipe bool
	trans  bool // true : TRANSITION (val != precedent) ; false : PROGRESSION (val > precedent)
}

func (k asCanal) String() string {
	fam, genre := "joueur", "prog"
	if k.equipe {
		fam = "equipe"
	}
	if k.trans {
		genre = "trans"
	}
	return fmt.Sprintf("comp %2d %c %s %s", k.comp, k.canal, fam, genre)
}

// asLire rend la valeur d'un canal et si elle est PRESENTE (C et D sont conditionnels).
func asLire(v StatValue, canal byte) (int64, bool) {
	switch canal {
	case 'A':
		return v.A, true
	case 'B':
		return v.B, true
	case 'C':
		return v.C, v.HasC
	default:
		return v.D, v.HasD
	}
}

// asInstants rend, pour chaque unite balayee, les instants ou elle emet un evenement. Un seul
// parcours des enregistrements : l'etat est tenu par (canal, slot, manche).
func asInstants(recs []StatRecord) map[asCanal][]int {
	type etat struct {
		comp        int
		canal       byte
		slot, round int
	}
	dernier := map[etat]int64{}
	vu := map[etat]bool{}
	out := map[asCanal][]int{}
	for _, r := range recs {
		equipe := IsTeamSlot(r.Slot)
		for comp, v := range r.Comps {
			if comp < 0 || comp >= asMaxComp {
				continue
			}
			for _, canal := range []byte{'A', 'B', 'C', 'D'} {
				val, present := asLire(v, canal)
				if !present || val < 0 || val > asValeurMax {
					continue
				}
				e := etat{comp, canal, r.Slot, r.Round}
				if vu[e] {
					if val > dernier[e] {
						k := asCanal{comp, canal, equipe, false}
						out[k] = append(out[k], r.TimeMS)
					}
					if val != dernier[e] {
						k := asCanal{comp, canal, equipe, true}
						out[k] = append(out[k], r.TimeMS)
					}
				}
				dernier[e], vu[e] = val, true
			}
		}
	}
	for k := range out {
		sort.Ints(out[k])
	}
	return out
}

// asTouche dit si l'unite a un evenement a +-[apToleranceMS] d'un instant.
func asTouche(instants []int, instant int) bool {
	i := sort.SearchInts(instants, instant-apToleranceMS)
	return i < len(instants) && instants[i] <= instant+apToleranceMS
}

// TestAssautStatborgArmementFenetre applique le protocole de l'en-tete.
func TestAssautStatborgArmementFenetre(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautStatborgArmementFenetre", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — balayage interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	couverts := map[asCanal]int{}
	plancherTouches := map[asCanal]int{}
	presents := map[asCanal]bool{}
	armements, tirages := 0, 0
	rng := rand.New(rand.NewSource(asGraine)) //nolint:gosec // tirage de mesure, pas de crypto

	for _, id := range apFilms() {
		v := apVariante[id]
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			t.Errorf("%s : film absent du cache (%s)", id, cache)
			continue
		}
		recs, tronque := StatRecordsCtx(context.Background(), src, id)
		instants := asInstants(recs)
		tmin, tmax := 0, 0
		for _, r := range recs {
			if tmin == 0 || r.TimeMS < tmin {
				tmin = r.TimeMS
			}
			if r.TimeMS > tmax {
				tmax = r.TimeMS
			}
		}
		emissions := 0
		for k, xs := range instants {
			presents[k] = true
			emissions += len(xs)
		}
		t.Logf("%-9s %-13s %6d enregistrement(s) (tronque=%v), %4d unite(s) porteuse(s), "+
			"%6d emission(s), t %d..%d ms", id, v.nom, len(recs), tronque, len(instants), emissions, tmin, tmax)

		// PLANCHER — sur TOUS les films : c'est une propriete de densite du flux.
		if tmax > tmin {
			for i := 0; i < asTirages; i++ {
				instant := tmin + rng.Intn(tmax-tmin)
				tirages++
				for k, xs := range instants {
					if asTouche(xs, instant) {
						plancherTouches[k]++
					}
				}
			}
		}
		// FENETRE — sur les seuls films a meche fixe.
		if v.mecheFixe {
			for _, det := range afExplosions[id] {
				arm := det - apMecheMS
				armements++
				for k, xs := range instants {
					if asTouche(xs, arm) {
						couverts[k]++
					}
				}
			}
		}
	}

	t.Logf("########## VERDICT — %d armement(s) juges, plancher sur %d instant(s) aleatoire(s), "+
		"%d unite(s) balayees (58 comps x 4 canaux x 2 familles x 2 genres = 928 possibles)",
		armements, tirages, len(presents))
	asRecensement(t, presents)
	if armements == 0 || tirages == 0 {
		t.Fatalf("mesure vide (armements=%d, tirages=%d)", armements, tirages)
	}
	asVerdict(t, presents, couverts, plancherTouches, armements, tirages)
}

// asVerdict trie, juge et imprime — les retenus d'abord, puis les meilleures selectivites pour
// dire QUELLE distance il reste quand rien ne tient.
func asVerdict(t *testing.T, presents map[asCanal]bool, couverts, plancher map[asCanal]int,
	armements, tirages int) {
	t.Helper()
	type ligne struct {
		k                 asCanal
		couvert           int
		tauxF, tauxP, sel float64
	}
	ls := make([]ligne, 0, len(presents))
	for k := range presents {
		tf := float64(couverts[k]) / float64(armements)
		tp := float64(plancher[k]) / float64(tirages)
		sel := 0.0
		if tp > 0 {
			sel = tf / tp
		} else if tf > 0 {
			sel = 1e9 // plancher nul : selectivite infinie, la couverture decide seule
		}
		ls = append(ls, ligne{k, couverts[k], tf, tp, sel})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].couvert != ls[j].couvert {
			return ls[i].couvert > ls[j].couvert
		}
		return ls[i].sel > ls[j].sel
	})

	retenus := 0
	for _, l := range ls {
		if l.couvert < armements || l.sel < apFacteurSelectivite {
			continue
		}
		retenus++
		t.Logf("*** CANDIDAT %s : %d/%d armements couverts (%.0f %%), plancher %.1f %%, "+
			"selectivite %.2fx ***", l.k, l.couvert, armements, 100*l.tauxF, 100*l.tauxP, l.sel)
	}
	if retenus > 0 {
		return
	}
	t.Logf("AUCUNE UNITE NE TIENT LES DEUX CRITERES. Les douze meilleures couvertures, pour dire " +
		"QUELLE distance il reste :")
	for i, l := range ls {
		if i >= 12 {
			break
		}
		t.Logf("  %s : %2d/%d couverts (%.0f %%), plancher %.1f %%, selectivite %.2fx",
			l.k, l.couvert, armements, 100*l.tauxF, 100*l.tauxP, l.sel)
	}
	// La MEILLEURE SELECTIVITE, meme a couverture partielle : c'est elle qui dirait qu'un canal
	// « sait quelque chose » sans tout couvrir.
	sort.Slice(ls, func(i, j int) bool { return ls[i].sel > ls[j].sel })
	t.Logf("Les six meilleures SELECTIVITES, couverture partielle admise :")
	for i, l := range ls {
		if i >= 6 {
			break
		}
		t.Logf("  %s : %2d/%d couverts, plancher %.1f %%, selectivite %.2fx",
			l.k, l.couvert, armements, 100*l.tauxP, l.sel)
	}
}

// asRecensement dit CE QUI EMET, avant tout verdict : quels composants, quels canaux. C'est lui
// qui repond aux deux questions laissees ouvertes par A6 — « les composants au-dela de 27 » et
// « les canaux conditionnels C et D emettent-ils quelque chose en Assaut ».
func asRecensement(t *testing.T, presents map[asCanal]bool) {
	t.Helper()
	parCanal := map[byte]int{}
	comps := map[int]bool{}
	compsHauts := map[int]bool{}
	for k := range presents {
		parCanal[k.canal]++
		comps[k.comp] = true
		if k.comp > 27 {
			compsHauts[k.comp] = true
		}
	}
	ks := make([]int, 0, len(comps))
	for c := range comps {
		ks = append(ks, c)
	}
	sort.Ints(ks)
	t.Logf("  COMPOSANTS porteurs (domaine 0..%d) : %v", asValeurMax, ks)
	t.Logf("  dont AU-DELA de 27 (jamais balayes par A6) : %d composant(s)", len(compsHauts))
	t.Logf("  unites par CANAL : A=%d, B=%d, C=%d, D=%d (C et D = les conditionnels)",
		parCanal['A'], parCanal['B'], parCanal['C'], parCanal['D'])
}
