package replaybuild

// grenade_join_corpus_test.go — LE TYPE D'UNE GRENADE QUI TUE, PAR LE LANCER PLUTOT QUE PAR
// LE TAG : les phases 0 et 1 de la mesure. Instrument, aucune ligne de production n'en depend.
// La plomberie (raccord des deux lectures, oracle, gardes) vit dans `grenade_corpus_test.go`.
//
// # LA QUESTION
//
// `damagetag/data/labels.tsv` porte 17 lignes de classe GRENADE : 15 VALIDE et 2 AMBIGU
// (`31e8d17e`, entrees `gggl` 0+1 ; `88f1034c`, entrees 0+1+3). Une etiquette AMBIGU n'obtient
// jamais de vignette (killicon.go, resolve), donc pas de son depuis que le son se joint a la
// vignette. Le type est peut-etre recuperable AILLEURS : `doc.grenades` publie les LANCERS avec
// leur rang (0 Frag, 1 Plasma, 2 Dynamo, 3 Spike) et leur auteur.
//
// # POURQUOI CETTE INFERENCE EST TESTABLE
//
// Les 15 tags VALIDE forment un ORACLE : leur `detail` cite l'entree `gggl` du jeu, et cette
// entree est le MEME rang que celui des lancers. On mesure donc la jointure sur eux AVANT de
// decider si elle vaut pour les deux ambigus. Le seuil (95 % d'accord, deux temoins effondres)
// est fixe PAR LE PLAN, avant execution : un seuil choisi apres coup n'est pas un seuil.

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// fenetresMS : les fenetres publiees. C'est la COURBE qui dit s'il existe une fenetre
// naturelle ; une valeur unique ne le dirait pas. Repere connu : pour une frag, la
// replication cesse ~1,4 s avant la meche.
var fenetresMS = []int{500, 1000, 2000, 3000, 5000, 10000}

// decalageTemoinMS : le decalage applique au temoin (b). Meme ordre de grandeur que la plus
// large fenetre mesuree, donc hors de portee de toute causalite lancer -> mort.
const decalageTemoinMS = 10000

// graineTemoin : le temoin (a) tire un AUTRE joueur au hasard. La graine est fixe pour que
// l'instrument soit rejouable a l'identique.
const graineTemoin = 20260815

// fenetreInfinie : borne employee pour repondre a « ce joueur a-t-il lance QUOI QUE CE SOIT
// avant cette mort ». Un film dure quelques minutes ; cette valeur les couvre toutes, et elle
// reste finie pour que la comparaison n'ait pas a traiter un cas particulier.
const fenetreInfinie = 1 << 30

// TestPhase0_PoidsDesTagsAmbigusDeGrenade — LE DENOMINATEUR, QUI N'AVAIT JAMAIS ETE COMPTE.
//
// Trois comptes : les morts par grenade et leur repartition par tag, la part que pesent les
// deux tags AMBIGU, et les morts par grenade sans tag resolu. C'est le troisieme qui dit si le
// chantier vaut d'etre mene : sous 1 %, la ligne du registre suffit.
func TestPhase0_PoidsDesTagsAmbigusDeGrenade(t *testing.T) {
	corpus := corpusRejeu(t)
	cache := cacheDesFilms(t)
	b := nouveauBalayage()
	for _, path := range artefactsDuCorpus(t, corpus) {
		f, ok := chargerFilm(t, path, cache)
		if !ok {
			b.echecs++
			continue
		}
		b.films++
		b.compterResultat(f.res, title.FilmShortMatchID(f.doc.MatchID))
	}
	if b.films == 0 {
		t.Fatal("aucun film exploitable dans le corpus")
	}
	t.Logf("PHASE 0 — %d film(s), %d mort(s) decodee(s) toutes sources confondues", b.films, b.morts)
	t.Logf("0.1  la repartition des morts par grenade, par tag :")
	b.publierVentilation(t)
	t.Logf("0.2  tags AMBIGU : %d / %d = %.2f %% des morts par grenade (%.3f %% de toutes les "+
		"morts) ; tags VALIDE : %d / %d = %.2f %%",
		b.ambigues, b.grenades, part(b.ambigues, b.grenades), part(b.ambigues, b.morts),
		b.valides, b.grenades, part(b.valides, b.grenades))
	b.publierFilmsAmbigus(t)
	t.Logf("0.3  morts de classe GRENADE sans tag resolu (ni VALIDE ni AMBIGU) : %d", b.autresStatuts)
	t.Logf("GATE 0 : les AMBIGU pesent %.2f %% des morts par grenade, borne haute a 95 %% "+
		"(Wilson) %.2f %% — seuil du plan : 1 %%",
		part(b.ambigues, b.grenades), borneHauteWilson(b.ambigues, b.grenades))
	t.Logf("     ce corpus seul ne tranche PAS le seuil quand sa borne le depasse : c'est a quoi " +
		"sert TestPhase0Bis, qui elargit le denominateur au cache de films.")
}

// accord : le resultat d'une jointure sur une population, pour une fenetre.
type accord struct {
	population int // morts de l'oracle, pontees
	trouves    int // au moins un lancer du meme joueur dans la fenetre
	justes     int // le rang du DERNIER lancer egale le rang attendu
	confondus  int // au moins deux rangs DIFFERENTS dans la fenetre
	// justesNonConfondus / nonConfondus : la meme mesure, cas confondants retires. Publiee A
	// COTE, jamais a la place : les retirer du denominateur gonflerait le taux.
	justesNonConfondus, nonConfondus int
}

// joindre cherche le dernier lancer de `idx` dans [tMS-fenetre, tMS] et rend son rang, le
// nombre de rangs distincts vus dans la fenetre, et si un lancer a ete trouve.
func joindre(lancers []lancerGrenade, idx, tMS, fenetre int) (rang, distincts int, ok bool) {
	rangs := map[int]bool{}
	rang, ok = -1, false
	for _, l := range lancers {
		if l.idx != idx || l.tMS > tMS || tMS-l.tMS > fenetre {
			continue
		}
		rangs[l.rang] = true
		rang, ok = l.rang, true // les lancers sont tries par instant : le dernier gagne
	}
	return rang, len(rangs), ok
}

// passeDeJointure : les entrees d'une passe de mesure. LES TROIS POPULATIONS (la vraie et les
// deux temoins) passent par la MEME fonction, et ne different que par ces champs — c'est ce qui
// interdit qu'un temoin soit joue avec une regle plus favorable que la mesure.
type passeDeJointure struct {
	morts   []mortGrenade
	lancers []lancerGrenade
	fenetre int
	// idxDe choisit l'index de joueur dont on cherche les lancers. Le temoin (a) y met un
	// AUTRE joueur tire au hasard.
	idxDe func(mortGrenade) int
	// decalage deplace l'instant de la mort. Le temoin (b) y met +10 s.
	decalage int
}

// mesurer joue la jointure sur toutes les morts d'un film pour une passe donnee.
func mesurer(acc *accord, p passeDeJointure) {
	for _, m := range p.morts {
		if m.rangAttendu < 0 || m.lanceurIdx < 0 {
			continue
		}
		idx := p.idxDe(m)
		if idx < 0 {
			continue
		}
		acc.population++
		rang, distincts, ok := joindre(p.lancers, idx, m.tMS+p.decalage, p.fenetre)
		if !ok {
			continue
		}
		acc.trouves++
		confondu := distincts > 1
		if confondu {
			acc.confondus++
		} else {
			acc.nonConfondus++
		}
		if rang == m.rangAttendu {
			acc.justes++
			if !confondu {
				acc.justesNonConfondus++
			}
		}
	}
}

// TestPhase1_JointureDuLancerSurLOracle — LA MESURE QUI DECIDE.
//
// Pour chaque mort dont le tag est VALIDE (donc de type connu), on cherche le DERNIER lancer du
// proprietaire de la source dans une fenetre anterieure, et on compare le rang du lancer au rang
// de l'oracle. Trois populations : la vraie, et les deux temoins de la decision n°2 du plan
// (un AUTRE joueur tire au hasard, la meme mort decalee de +10 s).
func TestPhase1_JointureDuLancerSurLOracle(t *testing.T) {
	corpus := corpusRejeu(t)
	cache := cacheDesFilms(t)
	rng := rand.New(rand.NewSource(graineTemoin)) //nolint:gosec // temoin deterministe
	reels := make([]accord, len(fenetresMS))
	temoinsA := make([]accord, len(fenetresMS))
	temoinsB := make([]accord, len(fenetresMS))
	var pop population
	pop.parRang = map[int]int{}
	var desaccords []string
	films, lancersPublies, lancersDisponibles := 0, 0, 0
	for _, path := range artefactsDuCorpus(t, corpus) {
		f, ok := chargerFilm(t, path, cache)
		if !ok {
			continue
		}
		films++
		lancersPublies += len(f.doc.Grenades)
		if f.doc.Coverage != nil {
			lancersDisponibles += f.doc.Coverage.Grenades.Available
		}
		morts := f.mortsGrenade(title.FilmShortMatchID(f.doc.MatchID))
		pop.compter(morts, f.lancers)
		desaccords = append(desaccords, desaccordsDe(morts, f.lancers)...)
		autre := tirerAutreJoueur(rng, f)
		vrai := func(m mortGrenade) int { return m.lanceurIdx }
		for i, fen := range fenetresMS {
			base := passeDeJointure{morts: morts, lancers: f.lancers, fenetre: fen, idxDe: vrai}
			mesurer(&reels[i], base)
			temoinA := base
			temoinA.idxDe = autre
			mesurer(&temoinsA[i], temoinA)
			temoinB := base
			temoinB.decalage = decalageTemoinMS
			mesurer(&temoinsB[i], temoinB)
		}
	}
	if films == 0 {
		t.Fatal("aucun film exploitable dans le corpus")
	}
	t.Logf("PHASE 1 — %d film(s) ; %d lancer(s) publie(s) sur %d disponible(s) dans les films",
		films, lancersPublies, lancersDisponibles)
	t.Logf("population : %d mort(s) de l'oracle (tag VALIDE), %d mort(s) AMBIGU (la cible), "+
		"%d sans pont vers un index de film, %d sans AUCUN lancer anterieur du proprietaire",
		pop.oracle, pop.ambigues, pop.sansPont, pop.sansLancerDuTout)
	t.Logf("             dont %d non revendiquee(s) par le kill-feed et %d divergente(s) — dans "+
		"les deux cas la jointure cherche le lancer de la VICTIME, pas du tueur credite",
		pop.nonRevendiquees, pop.divergentes)
	pop.publierPlancher(t)
	publierCourbe(t, reels, temoinsA, temoinsB)
	publierDesaccords(t, desaccords)
}

// fenetreDesDesaccords : la fenetre a laquelle les desaccords sont NOMMES. Le plateau de
// couverture (5 s) : plus tot, un desaccord peut n'etre qu'un lancer hors fenetre.
const fenetreDesDesaccords = 5000

// desaccordsDe nomme les morts ou la jointure repond FAUX a la fenetre du plateau. C'est la
// liste qu'un controle en Theater aurait a regarder — jamais la table entiere.
func desaccordsDe(morts []mortGrenade, lancers []lancerGrenade) []string {
	var out []string
	for _, m := range morts {
		if m.rangAttendu < 0 || m.lanceurIdx < 0 {
			continue
		}
		rang, distincts, ok := joindre(lancers, m.lanceurIdx, m.tMS, fenetreDesDesaccords)
		if !ok || rang == m.rangAttendu {
			continue
		}
		out = append(out, fmt.Sprintf("film %s, t=%d ms, tag %08x : oracle rang %d, lancer rang %d"+
			" (%d rang(s) distinct(s) dans la fenetre)", m.film, m.tMS, m.tag, m.rangAttendu, rang, distincts))
	}
	return out
}

// publierDesaccords nomme les desaccords, pour qu'ils soient VERIFIABLES et non seulement
// comptes. Un taux sans ses exceptions nommees n'est pas controlable.
func publierDesaccords(t *testing.T, lignes []string) {
	t.Helper()
	sort.Strings(lignes)
	t.Logf("DESACCORDS a la fenetre de %.1f s — %d ligne(s) :", float64(fenetreDesDesaccords)/1000, len(lignes))
	for _, l := range lignes {
		t.Logf("  %s", l)
	}
}

// population : la ventilation des morts par grenade AVANT toute jointure, et la distribution
// des rangs attendus — c'est elle qui donne le PLANCHER de base (cf. publierPlancher).
type population struct {
	oracle, ambigues, sansPont, sansLancerDuTout int
	// nonRevendiquees : morts que le kill-feed ne rattache a aucun tueur (le proprietaire de la
	// source est alors la victime). divergentes : morts revendiquees dont la source appartient
	// tout de meme a la victime. Les deux comptes disent sur QUI la jointure a cherche.
	nonRevendiquees, divergentes int
	parRang                      map[int]int
}

// compter ventile les morts d'un film.
func (p *population) compter(morts []mortGrenade, lancers []lancerGrenade) {
	for _, m := range morts {
		if m.rangAttendu < 0 {
			p.ambigues++
			continue
		}
		p.oracle++
		p.parRang[m.rangAttendu]++
		if !m.revendiquee {
			p.nonRevendiquees++
		}
		if m.diverge {
			p.divergentes++
		}
		if m.lanceurIdx < 0 {
			p.sansPont++
			continue
		}
		if _, _, ok := joindre(lancers, m.lanceurIdx, m.tMS, fenetreInfinie); !ok {
			p.sansLancerDuTout++
		}
	}
}

// publierPlancher rend le score d'un predicteur CONSTANT qui repondrait toujours le rang le
// plus frequent. Ce n'est pas un temoin : c'est le PLANCHER au-dessus duquel un accord doit se
// lire. Une population dominee par un seul type rend n'importe quelle methode flatteuse.
func (p *population) publierPlancher(t *testing.T) {
	t.Helper()
	rangs := make([]int, 0, len(p.parRang))
	for r := range p.parRang {
		rangs = append(rangs, r)
	}
	sort.Slice(rangs, func(i, j int) bool { return p.parRang[rangs[i]] > p.parRang[rangs[j]] })
	var b strings.Builder
	for _, r := range rangs {
		fmt.Fprintf(&b, " rang %d : %d (%.1f %%) ;", r, p.parRang[r], part(p.parRang[r], p.oracle))
	}
	t.Logf("distribution des rangs attendus —%s", strings.TrimSuffix(b.String(), " ;"))
	if len(rangs) > 0 {
		t.Logf("PLANCHER : un predicteur constant « toujours le rang %d » scorerait %.2f %% "+
			"sur cette population, sans rien lire", rangs[0], part(p.parRang[rangs[0]], p.oracle))
	}
}

// tirerAutreJoueur rend le selecteur du temoin (a) : un AUTRE joueur du roster, tire au hasard
// une fois par mort. -1 quand le film n'a qu'un joueur au roster.
func tirerAutreJoueur(rng *rand.Rand, f *filmDeLArtefact) func(mortGrenade) int {
	idx := make([]int, 0, len(f.doc.Roster))
	for _, r := range f.doc.Roster {
		idx = append(idx, r.FilmIndex)
	}
	sort.Ints(idx)
	return func(m mortGrenade) int {
		var cand []int
		for _, i := range idx {
			if i != m.lanceurIdx {
				cand = append(cand, i)
			}
		}
		if len(cand) == 0 {
			return -1
		}
		return cand[rng.Intn(len(cand))]
	}
}

// publierCourbe rend l'accord en fonction de la fenetre, la couverture, et les deux temoins.
func publierCourbe(t *testing.T, reels, temoinsA, temoinsB []accord) {
	t.Helper()
	t.Logf("%-9s %10s %10s %9s %9s %11s %9s %9s", "fenetre", "population", "couverture",
		"accord", "confondus", "accord(sain)", "temoin(a)", "temoin(b)")
	for i, fen := range fenetresMS {
		r, a, b := reels[i], temoinsA[i], temoinsB[i]
		t.Logf("%6.1f s %10d %6d %5.1f%% %8.2f%% %9d %10.2f%% %8.2f%% %8.2f%%",
			float64(fen)/1000, r.population, r.trouves, part(r.trouves, r.population),
			part(r.justes, r.trouves), r.confondus, part(r.justesNonConfondus, r.nonConfondus),
			part(a.justes, a.trouves), part(b.justes, b.trouves))
	}
	t.Logf("colonnes : « couverture » = morts ayant AU MOINS un lancer du proprietaire dans la "+
		"fenetre ; « accord » = rang du DERNIER lancer egal au rang de l'oracle, cas confondants "+
		"COMPRIS ; « confondus » = deux rangs differents dans la fenetre ; « accord(sain) » = les "+
		"memes retires du numerateur ET du denominateur ; temoins (a) autre joueur, (b) mort "+
		"decalee de +%d s.", decalageTemoinMS/1000)
	t.Logf("LES DEUX TEMOINS, AVEC LEUR PROPRE COUVERTURE — un temoin qui trouve peu de lancers " +
		"ne s'effondre pas par vertu de la jointure, mais par rarete de son entree :")
	t.Logf("%-9s %14s %12s %14s %12s", "fenetre", "temoin(a) trv", "accord(a)", "temoin(b) trv", "accord(b)")
	for i, fen := range fenetresMS {
		a, b := temoinsA[i], temoinsB[i]
		t.Logf("%6.1f s %8d/%-5d %10.2f%% %8d/%-5d %10.2f%%",
			float64(fen)/1000, a.trouves, a.population, part(a.justes, a.trouves),
			b.trouves, b.population, part(b.justes, b.trouves))
	}
	for i, fen := range fenetresMS {
		r, a, b := reels[i], temoinsA[i], temoinsB[i]
		t.Logf("  %5.1f s : reel %d/%d justes, temoin(a) %d/%d, temoin(b) %d/%d",
			float64(fen)/1000, r.justes, r.trouves, a.justes, a.trouves, b.justes, b.trouves)
	}
}
