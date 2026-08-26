package replay

// colline_proprietaire_d2_test.go — PHASE D2 : QUI TIENT LA COLLINE ?
//
// Le plan `.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md` (§2.2) part d'un manque ECRIT DANS LE
// CODE : `hillStatesOf` (zone_states_hill.go) construit ses intervalles avec `Active: true` et
// `Owner` JAMAIS renseigne — le rejeu sait quelle colline est active, jamais qui la tient. Le
// volet 1 du lot C-ter le disait en toutes lettres : « on ne publie pas ce qu'on n'a pas mesure ».
// Ce fichier est cette mesure.
//
// # LE CANAL, ET IL EST DEJA LU
//
// L'objet de mode KOTH tient sur quatre slots `ti=13` consecutifs — [tag 5 designateur]
// [tag 4 proprietaire][tag 4 capteur][tag 3 jauge]. `hillDesignatorOf` ELIT deja le designateur,
// et sa condition d'election EXIGE que le slot suivant porte un canal de propriete qui parle
// (`hillDesignatorMinOwnerSamples`). Le canal candidat est donc `ser.owner[d.slot+1]`, et il est
// deja entre les mains de la production : rien n'est decode de neuf ici.
//
// # L'ORACLE : LE SCORE DE MODE, ET POURQUOI C'EN EST UN
//
// En KOTH le score de mode EST le temps de colline (« l'API compte des secondes de colline »,
// chronique 33->34 du contrat). Le camp dont le score de MODE monte pendant un intervalle est
// donc celui qui tient la colline pendant cet intervalle. C'est un oracle a la milliseconde,
// DEJA EN PRODUCTION depuis le schema 12 (`objectiveevents.SeriesTotal` + `ModeScoreComponent`),
// et il ne coute aucune lecture de film supplementaire.
//
// # CE QUE LA MESURE NE DEMANDE PAS, ET C'EST DELIBERE
//
// ELLE NE DEMANDE NI ROSTER NI SCORES D'API. On ne teste pas « la valeur 0 est le camp 0 » —
// on teste que la valeur du canal DESIGNE le camp qui marque, ce qui est la seule chose dont le
// rendu ait besoin. L'accord se mesure donc sous la MEILLEURE BIJECTION valeur <-> slot d'equipe,
// et c'est le TEMOIN qui rend la mesure honnete : un canal sans information garde la meme liberte
// de bijection et ne peut pas depasser le niveau du hasard. Exiger l'identite absolue aurait fait
// dependre la mesure de `team_0_score` / `team_1_score`, qui ne comptent PAS la meme grandeur
// d'un film KOTH a l'autre (releve du corpus : 3-2 et 4-2 contre 105-8 et 78-105).
//
// # REGIME
//
// SOUS GARDE `ZONE_FILM` (le harnais `ctCharge` du lot C-ter, reutilise plutot que recopie), UN
// film par processus, lecture seule, AUCUNE base ouverte, AUCUN changement de production.
//
//	$env:ZONE_FILM="<cache>/film_chunks/01e1f945"; go test ./internal/analysis/replay/ -run CollineProprietaireD2 -v

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// d2MinRunFrames : duree MINIMALE d'un intervalle de propriete pour etre confronte.
//
// POURQUOI UN PLANCHER. Le score de mode s'incremente d'un point par SECONDE de garde : un
// intervalle plus court que la seconde ne peut pas faire monter le compteur, et le compter comme
// « aucun camp ne marque » melangerait un silence de mesure a un desaccord. Dix frames valent une
// seconde a la cadence publiee (100 ms), et le plancher se derive de la cadence du document —
// jamais d'un nombre de frames ecrit en dur.
const d2MinRunSeconds = 1

// d2Run est un intervalle de propriete constante, en frames, bornes INCLUSES.
type d2Run struct {
	v      uint64
	t0, t1 int
}

// d2Verdict porte le resultat d'une confrontation (signal ou temoin).
type d2Verdict struct {
	// runs : intervalles nommes (valeur non neutre) assez longs pour etre confrontes.
	runs int
	// confrontables : ceux ou EXACTEMENT UN camp marque — les seuls ou l'oracle tranche.
	confrontables int
	// deuxMarquent / aucunNeMarque : les deux formes d'abstention, comptees a part.
	deuxMarquent, aucunNeMarque int
	// accord : confrontations concordantes sous la MEILLEURE bijection.
	accord int
	// bijection : la carte retenue, pour le journal.
	bijection string
}

func (v d2Verdict) taux() float64 {
	if v.confrontables == 0 {
		return 0
	}
	return 100 * float64(v.accord) / float64(v.confrontables)
}

func (v d2Verdict) String() string {
	return fmt.Sprintf("%d/%d = %.1f %% (intervalles %d, deux marquent %d, aucun %d, %s)",
		v.accord, v.confrontables, v.taux(), v.runs, v.deuxMarquent, v.aucunNeMarque, v.bijection)
}

// TestCollineProprietaireD2 — LA MESURE. Un film par processus.
func TestCollineProprietaireD2(t *testing.T) {
	e := ctCharge(t)
	c := zoneCtx{origin: e.posUS, step: uint64(e.doc.FrameIntervalMS) * 1000,
		frames: e.doc.FrameCount, intervalMS: e.doc.FrameIntervalMS}
	ser := zoneSeriesOf(e.sc.Reads, c)

	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Fatalf("%s : AUCUN designateur elu — le canal de propriete n'a pas de voisin a mesurer "+
			"(ce film retombe sur la methode par rampes, cf. buildHillStates)", e.short)
	}
	ownerSlot := d.slot + 1
	samples := ser.owner[ownerSlot]
	t.Logf("%s : designateur slot %d, %d bascule(s), premier contact frame %d ; "+
		"canal de propriete slot %d, %d emission(s) ; axe %d frames a %d ms",
		e.short, d.slot, len(d.changes), d.first, ownerSlot, len(samples),
		e.doc.FrameCount, e.doc.FrameIntervalMS)

	recs := objectiveevents.StatRecords(p2aSource(t, e.dir))
	score := objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)
	slots := d2ScoreSlots(score)
	// DIAGNOSTIC DE L'ORACLE, pose AVANT de s'en servir. Un slot d'equipe manquant peut venir
	// de DEUX causes tres differentes : le film ne replique pas la serie de ce camp, ou notre
	// filtre de stricte croissance (`longestRun`) l'a jetee. La lecture NON STRICTE du meme
	// emplacement les separe, et sans elle le negatif resterait vague.
	brut := objectiveevents.SeriesTotal(recs,
		objectiveevents.StatComponent{Comp: 0, SideB: false, Strict: false}, true)
	t.Logf("%s : oracle — slots STRICTS %v, slots BRUTS %v", e.short, slots, d2ScoreSlots(brut))
	if len(slots) != 2 {
		t.Logf("NEGATIF  %s : %d slot(s) d'equipe seulement au score de mode — l'oracle ne peut "+
			"rien trancher sur ce film, la mesure s'arrete ici (ce n'est PAS un verdict sur le canal)",
			e.short, len(slots))
		return
	}
	t.Logf("%s : score de mode — slot %d : %d point(s), final %d ; slot %d : %d point(s), final %d",
		e.short, slots[0], len(score[slots[0]]), d2Final(score[slots[0]]),
		slots[1], len(score[slots[1]]), d2Final(score[slots[1]]))

	minFrames := d2MinRunSeconds * 1000 / max(e.doc.FrameIntervalMS, 1)
	runs := d2Runs(samples, e.doc.FrameCount, minFrames)
	sig := d2Confront(runs, score, slots, e)
	t.Logf("SIGNAL   %s : %s", e.short, sig)

	// TEMOIN (a) — LA PERMUTATION : le meme test sur les AUTRES canaux de propriete du film.
	// Un canal qui ne dit rien de la colline doit s'effondrer ; on publie le MEILLEUR d'entre
	// eux, c'est-a-dire le temoin le plus DUR.
	pire := d2Verdict{}
	pireSlot := uint32(0)
	autres := 0
	for _, s := range sortedZoneSlots(ser.owner) {
		if s == ownerSlot {
			continue
		}
		v := d2Confront(d2Runs(ser.owner[s], e.doc.FrameCount, minFrames), score, slots, e)
		if v.confrontables == 0 {
			continue
		}
		autres++
		if v.taux() > pire.taux() {
			pire, pireSlot = v, s
		}
	}
	if autres == 0 {
		t.Logf("TEMOIN a %s : AUCUN autre canal de propriete confrontable — temoin sans objet", e.short)
	} else {
		t.Logf("TEMOIN a %s : permutation, pire des %d autres canaux (slot %d) : %s",
			e.short, autres, pireSlot, pire)
	}

	// TEMOIN (b) — LE DECALAGE : les memes intervalles, glisses de +20 s. Si l'accord tient
	// encore, il ne doit rien a la coincidence temporelle et tout a un desequilibre du film.
	shift := 20 * 1000 / max(e.doc.FrameIntervalMS, 1)
	dec := d2Confront(d2Decale(runs, shift, e.doc.FrameCount), score, slots, e)
	t.Logf("TEMOIN b %s : decalage +20 s : %s", e.short, dec)

	if sig.confrontables == 0 {
		t.Fatalf("%s : AUCUNE confrontation possible — la mesure n'a pas de denominateur", e.short)
	}
}

// d2Runs rend les intervalles de propriete NOMMEE (valeur non neutre) d'une serie, bornes
// incluses, en ecartant ceux trop courts pour que le compteur de secondes puisse bouger.
//
// LA SEGMENTATION EST CELLE DE LA PRODUCTION : `mergeZoneRuns` fond les re-emissions de meme
// valeur, et chaque groupe court jusqu'a la veille du suivant — exactement `ownerSpansOf`. Une
// seconde ecriture de ce decoupage divergerait au premier correctif.
func d2Runs(ss []zoneSample, frames, minFrames int) []d2Run {
	groups := mergeZoneRuns(ss)
	out := make([]d2Run, 0, len(groups))
	for i, g := range groups {
		t1 := frames - 1
		if i+1 < len(groups) {
			t1 = groups[i+1].t - 1
		}
		if t1 < g.t || g.v == zoneNeutralOwner {
			continue
		}
		if t1-g.t+1 < minFrames {
			continue
		}
		out = append(out, d2Run{v: g.v, t0: g.t, t1: t1})
	}
	return out
}

// d2Decale glisse les intervalles dans le temps et rejette ceux qui sortent de l'axe.
func d2Decale(runs []d2Run, shift, frames int) []d2Run {
	out := make([]d2Run, 0, len(runs))
	for _, r := range runs {
		if r.t1+shift >= frames {
			continue
		}
		out = append(out, d2Run{v: r.v, t0: r.t0 + shift, t1: r.t1 + shift})
	}
	return out
}

// d2Confront confronte chaque intervalle au camp qui MARQUE pendant lui, et rend l'accord sous
// la meilleure bijection valeur <-> slot d'equipe.
func d2Confront(runs []d2Run, score map[int][]objectiveevents.ScorePoint, slots []int,
	e ctEntree,
) d2Verdict {
	v := d2Verdict{runs: len(runs)}
	// paires[valeur][slot] : combien d'intervalles de cette valeur ont vu ce slot marquer seul.
	paires := map[uint64]map[int]int{}
	for _, r := range runs {
		a := d2Delta(score[slots[0]], e, r.t0, r.t1)
		b := d2Delta(score[slots[1]], e, r.t0, r.t1)
		switch {
		case a > 0 && b > 0:
			v.deuxMarquent++
			continue
		case a == 0 && b == 0:
			v.aucunNeMarque++
			continue
		}
		marqueur := slots[0]
		if b > 0 {
			marqueur = slots[1]
		}
		v.confrontables++
		if paires[r.v] == nil {
			paires[r.v] = map[int]int{}
		}
		paires[r.v][marqueur]++
	}
	v.accord, v.bijection = d2MeilleureBijection(paires, slots)
	return v
}

// d2MeilleureBijection choisit, entre les deux appariements possibles, celui qui explique le plus
// d'intervalles — et rend son compte.
//
// DEUX BIJECTIONS SEULEMENT, parce que le canal ne prend que deux valeurs nommees. Les valeurs
// observees sont triees pour que le choix soit deterministe d'un film a l'autre.
func d2MeilleureBijection(paires map[uint64]map[int]int, slots []int) (int, string) {
	vals := make([]uint64, 0, len(paires))
	for k := range paires {
		vals = append(vals, k)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	if len(vals) == 0 {
		return 0, "aucune valeur"
	}
	if len(vals) == 1 {
		// UNE SEULE VALEUR OBSERVEE : il n'y a pas de bijection a choisir, et l'accord ne
		// prouve rien — un canal constant « explique » tout ce qu'un seul camp fait. On rend
		// le compte, et le libelle DIT que la mesure est degeneree.
		n := 0
		for _, c := range paires[vals[0]] {
			if c > n {
				n = c
			}
		}
		return n, fmt.Sprintf("valeur unique %d — bijection DEGENEREE", vals[0])
	}
	direct := paires[vals[0]][slots[0]] + paires[vals[1]][slots[1]]
	croise := paires[vals[0]][slots[1]] + paires[vals[1]][slots[0]]
	if direct >= croise {
		return direct, fmt.Sprintf("%d->slot%d, %d->slot%d", vals[0], slots[0], vals[1], slots[1])
	}
	return croise, fmt.Sprintf("%d->slot%d, %d->slot%d", vals[0], slots[1], vals[1], slots[0])
}

// d2Delta rend la progression du score sur la fenetre de frames [t0, t1].
func d2Delta(pts []objectiveevents.ScorePoint, e ctEntree, t0, t1 int) int64 {
	return d2ValeurA(pts, e, t1) - d2ValeurA(pts, e, t0)
}

// d2ValeurA rend la valeur CUMULEE en vigueur a une frame : la derniere emission dont l'instant
// y tombe ou la precede. Zero avant la premiere.
func d2ValeurA(pts []objectiveevents.ScorePoint, e ctEntree, frame int) int64 {
	var out int64
	for _, p := range pts {
		f, ok := p2aFrameOf(e.doc, p.TimeMS)
		if !ok || f > frame {
			continue
		}
		out = p.Value
	}
	return out
}

// d2ScoreSlots rend les slots d'equipe qui portent une serie, tries.
func d2ScoreSlots(score map[int][]objectiveevents.ScorePoint) []int {
	out := make([]int, 0, len(score))
	for s, pts := range score {
		if len(pts) > 0 {
			out = append(out, s)
		}
	}
	sort.Ints(out)
	return out
}

// d2Final rend le dernier point d'une serie cumulee, 0 si vide.
func d2Final(pts []objectiveevents.ScorePoint) int64 {
	if len(pts) == 0 {
		return 0
	}
	return pts[len(pts)-1].Value
}
