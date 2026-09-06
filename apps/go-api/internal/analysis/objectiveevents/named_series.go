package objectiveevents

// named_series.go — DES ENREGISTREMENTS AUX SERIES CUMULEES.
//
// Extrait de `named.go` le 2026-09-03 (lot 4) : le regroupement en une passe y avait porte le
// fichier au-dela du seuil de 500 lignes. La coupe suit la RESPONSABILITE, pas la ligne — ici
// vit tout ce qui va d'une liste d'enregistrements d'entite a une suite de valeurs
// exploitable (groupement par emplacement / slot / manche, rejet des ancrages parasites,
// cumul des manches, conversion d'un compteur en instants) ; `named.go` garde la table des
// emplacements, le type d'evenement et la publication.

import (
	"log/slog"
	"sort"
)

// # LES BORNES DE PRUDENCE DU DEROULAGE (lot 4b, 2026-09-03 — decision utilisateur du meme jour)
//
// POURQUOI ELLES EXISTENT. [incrementTimes] emet UN entier par UNITE gagnee par le compteur.
// Un enregistrement mal aligne — decode a une position ou il n'y en a pas, donc dont TOUS les
// canaux sont faux a la fois — porte une valeur arbitraire et POSITIVE, que ni le rejet des
// emissions negatives ni le domaine du score de mode n'attrapent. Mesure du 2026-09-03
// (item 4b.1) : `51101d1d` deroule 2 163 333 677 evenements sur le seul `comp 20 B`
// (slot 24, t = 136 636 ms) ; le `[]int` pese alors 17,31 Go et son dernier `growslice`
// (8,65 Go copies vers 17,31 Go, les deux vivants) demande 25,96 Go — les « ~26 Go, crash go
// runtime » du registre des reports (2026-08-24), retrouves a 0,2 % pres. Trois autres films
// du cache ont la meme structure (`a349fea8` 1,1 Md sur 21 B ; `1c4c63c2` 537 M sur 22 A ;
// `60ae07c4` 2,1 Md sur 21 A).
//
// CE QUE CES BORNES SONT, ET CE QU'ELLES NE SONT PAS. Un dernier rempart MEMOIRE, pas un
// filtre d'anomalie : sur un enregistrement fautif, un plafond par pas coupe les gros canaux
// et laisse passer les petits (31, 58, 2 139 mesures au meme instant sur le meme slot). Le
// filtre juste serait au niveau de l'ENREGISTREMENT — rejeter le record entier quand l'un de
// ses canaux est hors domaine, comme [modeScoreInDomain] le fait deja pour le comp 0. Il n'est
// pas dans ce lot.
const (
	// maxUnrollPerStep borne le deroulage d'UN point, PREMIER TERME COMPRIS (`prev` part de
	// zero et ne redescend jamais : la grandeur qui explose est `p.Value - prev`, pas l'ecart
	// entre deux echantillons consecutifs).
	//
	// LA VALEUR EST MESUREE, PAS DEVINEE (item 4b.1, 9 films sains + 4 bombes, sur les seuls
	// emplacements que la production deroule) : le pire deroulage d'un film SAIN vaut 17 306
	// (`d9781168`, comp 20 B, slot 12, t = 345 931 ms) — marge 5,8x ; la plus petite bombe vaut
	// 537 698 416 (`1c4c63c2`, comp 22 A) — marge 5 377x. Les deux populations sont separees
	// d'un facteur 31 000, et 100 000 tombe au milieu : aucun film sain connu n'est touche, les
	// quatre bombes connues sont neutralisees. Huit films sains sur neuf tiennent d'ailleurs
	// sous 2 — un seul porte toute l'anomalie.
	maxUnrollPerStep = 100_000
	// maxNamedEventsPerFilm borne le TOTAL emis par une passe sur un film. Le pire total sain
	// mesure vaut 21 160 (`d9781168`) — marge 47x ; les quatre bombes sont 500 a 3 900 fois
	// au-dessus. Une passe qui l'atteint s'arrete : le rejeu vaut mieux tronque qu'absent.
	maxNamedEventsPerFilm = 1_000_000
	// maxRejectLogs borne le DETAIL journalise. Un film pathologique peut porter un deroulage
	// aberrant par point (jusqu'a `statMaxRecordsPerFilm` = 33 076) : sans cette borne, le
	// dernier rempart memoire deviendrait un rempart a inonder le journal. Le COMPTE, lui,
	// n'est jamais tronque — il est publie par [eventBudget.resume].
	maxRejectLogs = 8
)

// eventBudget porte le solde d'evenements d'une passe sur un film, et la trace de ce que les
// bornes ont refuse.
//
// LE SOLDE DESCEND JUSQU'A [incrementTimes], ET CE N'EST PAS UN DETAIL D'IMPLEMENTATION.
// `statMaxRecordsPerFilm` (33 076) borne les POINTS, pas les evenements : sous la seule borne
// par pas, UNE SEULE serie peut encore emettre 33 076 x 100 000 entrees, soit 26 Gio DANS UN
// SEUL APPEL — avant que l'appelant ait la main pour verifier un total. Un plafond verifie
// seulement ENTRE les series ne protegerait donc de rien. Ce qui voyage ici est le SOLDE ; la
// VALEUR du plafond, elle, reste detenue par les trois passes qui ouvrent un budget
// ([NamedEventsFrom], [SlotIdentityFrom], [crossCheckFrom]).
type eventBudget struct {
	// origine nomme la passe dans le journal — c'est ce qui distingue « le nommage a rejete »
	// de « le pont d'identite a rejete » quand les deux lisent le meme film.
	origine string
	// reste est le nombre d'evenements encore emissibles par cette passe.
	reste int
	// rejetes compte les deroulages refuses par la borne par pas (compte EXACT, jamais tronque).
	rejetes int
	// tronque dit que le solde a ete epuise : la passe n'emet plus rien du tout ensuite.
	tronque bool
	// journalises compte les avertissements DETAILLES deja emis (cf. maxRejectLogs).
	journalises int
}

// newEventBudget ouvre le budget d'une passe sur un film.
func newEventBudget(origine string) *eventBudget {
	return &eventBudget{origine: origine, reste: maxNamedEventsPerFilm}
}

// rejeter enregistre un deroulage hors borne et le journalise (detail borne, compte exact).
func (b *eventBudget) rejeter(key statSlotKey, p ScorePoint, n int64) {
	b.rejetes++
	if b.journalises >= maxRejectLogs {
		return
	}
	b.journalises++
	slog.Warn("objectiveevents: deroulage aberrant rejete (dernier rempart memoire)",
		"passe", b.origine, "comp", key.Comp, "cote", key.Side, "slot", p.Slot,
		"time_ms", p.TimeMS, "deroulage", n, "borne", maxUnrollPerStep)
}

// epuiser marque le solde consomme. Une seule ligne de journal : les appels suivants sortent
// immediatement, il n'y a rien de nouveau a dire a chacun d'eux.
func (b *eventBudget) epuiser(key statSlotKey, p ScorePoint, n int64) {
	b.tronque = true
	slog.Warn("objectiveevents: plafond d'evenements du film atteint, deroulage interrompu",
		"passe", b.origine, "comp", key.Comp, "cote", key.Side, "slot", p.Slot,
		"time_ms", p.TimeMS, "deroulage", n, "reste", b.reste, "plafond", maxNamedEventsPerFilm)
}

// resume publie, en fin de passe, ce que les bornes ont coute. SILENCIEUX quand elles n'ont
// rien refuse — c'est le cas des neuf films sains du corpus d'equivalence, et c'est la seule
// facon pour qu'une ligne de journal signifie encore quelque chose quand elle apparait.
func (b *eventBudget) resume() {
	if b.rejetes == 0 && !b.tronque {
		return
	}
	slog.Warn("objectiveevents: bornes de deroulage appliquees sur ce film",
		"passe", b.origine, "deroulages_rejetes", b.rejetes, "tronque", b.tronque,
		"evenements_emis", maxNamedEventsPerFilm-b.reste, "plafond", maxNamedEventsPerFilm)
}

// rawSeriesByKey est [rawSeriesByRound] pour TOUS les emplacements NON REDONDANTS d'une table
// a la fois, en une seule marche de `recs` — memes filtres, meme ordre d'insertion par
// (emplacement, slot, manche), donc memes series.
//
// Les emplacements redondants sont ecartes ICI : ils n'emettent aucun evenement, et les
// grouper serait du travail jete.
func rawSeriesByKey(recs []StatRecord, table map[statSlotKey]statSlot) map[statSlotKey]map[int]map[int][]ScorePoint {
	windows := ResolveRoundWindows(recs)
	out := make(map[statSlotKey]map[int]map[int][]ScorePoint, len(table))
	for key, slot := range table {
		if !slot.Redundant {
			out[key] = map[int]map[int][]ScorePoint{}
		}
	}
	for _, r := range recs {
		// Seuls les slots de JOUEUR nomment des evenements ; et la manche declaree doit
		// s'accorder au temps, comme dans [rawSeriesByRound] (les deux marches rendent les
		// memes series, `named_onepass_test.go` en est le garde-rail).
		if IsTeamSlot(r.Slot) || windows.Excludes(r) {
			continue
		}
		for key, raw := range out {
			v, ok := r.Comps[key.Comp]
			if !ok {
				continue
			}
			val := v.A
			if key.Side == sideB {
				val = v.B
			}
			// Memes deux rejets que [rawSeriesByRound], et pour les memes raisons : une
			// emission negative est un ancrage parasite, et un score de mode hors domaine
			// denonce une emission mal alignee sur ses DEUX canaux.
			if val < 0 || (key.Comp == modeScoreComp && !modeScoreInDomain(v)) {
				continue
			}
			if raw[r.Slot] == nil {
				raw[r.Slot] = map[int][]ScorePoint{}
			}
			raw[r.Slot][r.Round] = append(raw[r.Slot][r.Round],
				ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
		}
	}
	return out
}

// seriesBySlot rend, par slot de JOUEUR, la suite chronologique des valeurs d'un
// emplacement, debarrassee des ancrages parasites.
//
// Le filtre est le meme que celui du score de mode et pour la meme raison : un compteur de
// recompense ne recule jamais, donc la plus longue sous-suite NON DECROISSANTE est la vraie
// suite. Non decroissante et non strictement croissante : un composant porte deux valeurs
// et il est reemis des que l'UNE des deux bouge, donc la meme valeur revient legitimement.
// # Les MANCHES, et pourquoi la suite est cumulee (2026-08-18)
//
// Un compteur repart de zero a chaque manche (`StatRecord.Round`). Concatener les manches sans
// rien faire donnerait une suite qui RECULE, et le filtre de plus longue sous-suite n'en
// garderait qu'une — c'est exactement ce que faisait la version d'avant, qui ne voyait de toute
// facon que la manche 1. Chaque manche est donc filtree separement, puis DECALEE du total des
// manches precedentes : la suite rendue est croissante sur tout le match et son dernier point
// est le total du match. Mesure : les frags d'un Oddball passent de 48 a 87 sur 88 attendus.
func seriesBySlot(recs []StatRecord, key statSlotKey) map[int][]ScorePoint {
	return cumulateRounds(rawSeriesByRound(recs, key, false), RealRounds(recs))
}

// rawSeriesByRound groupe les emissions par slot puis par manche, en jetant les ancrages
// parasites. teams choisit les slots d'equipe plutot que ceux de joueur.
//
// LA MANCHE DECLAREE EST CONFRONTEE AU TEMPS (2026-09-06, cf. round_windows.go) : les manches
// se jouent dans l'ordre, donc un enregistrement date hors de l'intervalle de la manche qu'il
// declare a une manche mal lue et n'alimente aucune serie. C'est le filtre qui manquait pour
// que [longestRun] ne soit pas trompe — une valeur mal lue mais PLUS GRANDE prolonge la suite
// non decroissante au lieu de la rompre.
func rawSeriesByRound(recs []StatRecord, key statSlotKey, teams bool) map[int]map[int][]ScorePoint {
	windows := ResolveRoundWindows(recs)
	raw := map[int]map[int][]ScorePoint{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) != teams || windows.Excludes(r) {
			continue
		}
		v, ok := r.Comps[key.Comp]
		if !ok {
			continue
		}
		val := v.A
		if key.Side == sideB {
			val = v.B
		}
		// Une emission NEGATIVE est un ancrage parasite : un compteur de recompense est
		// positif. Elle est jetee ICI, avant le choix de la sous-suite, et pas apres —
		// sinon elle fausse ce choix. Mesure : sur la suite (1, -115, 1), la plus longue
		// sous-suite non decroissante retenue devenait (-115, 1), ce qui datait
		// l'evenement de la DERNIERE emission au lieu de la premiere.
		if val < 0 {
			continue
		}
		// LE SCORE DE MODE EST BORNE SUR SES DEUX CANAUX, pas seulement sur celui qu'on lit.
		// Les deux valeurs d'un composant sortent de la MEME emission : un canal aberrant
		// prouve que l'emission etait mal alignee, et la valeur de l'autre ne vaut rien non
		// plus. Mesure du 2026-08-31 sur 65 films (3 986 enregistrements joueur porteurs du
		// composant 0) : le canal B vaut ZERO dans 98,3 % des cas, et l'enregistrement
		// `ce083875` slot 16 a 219075 ms porte A=66 avec B=16635 — un saut de 66 unites que
		// [incrementTimes] transformait en 66 explosions publiees au meme instant. Sa seule
		// marque distinctive est ce B hors domaine ; son A passait la borne.
		if key.Comp == modeScoreComp && !modeScoreInDomain(v) {
			continue
		}
		if raw[r.Slot] == nil {
			raw[r.Slot] = map[int][]ScorePoint{}
		}
		raw[r.Slot][r.Round] = append(raw[r.Slot][r.Round],
			ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
	}
	return raw
}

// cumulateRounds filtre chaque manche par la plus longue sous-suite non decroissante, puis
// decale les manches successives du total des precedentes. Les manches absentes de `real` sont
// IGNOREES : ce sont des ancrages fortuits, et les cumuler ferait exploser les compteurs.
//
// La suite assemblee passe par [ChronologicalTotal] : le cumul suppose que l'ordre des MANCHES
// est l'ordre du TEMPS, et cette supposition doit etre verifiee, pas presumee.
func cumulateRounds(raw map[int]map[int][]ScorePoint, real map[int]bool) map[int][]ScorePoint {
	out := make(map[int][]ScorePoint, len(raw))
	for slot, byRound := range raw {
		var offset int64
		var serie []ScorePoint
		for _, round := range sortedIntKeys(byRound) {
			if !real[round] {
				continue
			}
			pts := byRound[round]
			sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
			kept := longestRun(pts, false)
			if len(kept) == 0 {
				continue
			}
			for _, p := range kept {
				serie = append(serie, ScorePoint{
					TimeMS: p.TimeMS, Slot: slot, Value: p.Value + offset})
			}
			offset += kept[len(kept)-1].Value
		}
		if serie = ChronologicalTotal(serie); len(serie) > 0 {
			out[slot] = serie
		}
	}
	return out
}

// ChronologicalTotal ecarte d'une suite CUMULEE tout point date avant le dernier point retenu,
// et rend le nombre d'ecartes.
//
// # POURQUOI CE CONTROLE EXISTE
//
// Un total de match est construit en concatenant les manches DANS L'ORDRE DES MANCHES, chacune
// decalee du total des precedentes. Cela ne donne une suite chronologique que si l'ordre des
// manches EST celui du temps — c'est-a-dire si la decoupe par manche est juste. Elle ne l'etait
// pas : un enregistrement de la manche 1 range en manche 0 faisait rendre `{3167, 60}` puis
// `{3057, 61}` sur `51ebbc0f`, une courbe qui RECULE dans le temps (mesure du 2026-09-06). La
// cause est corrigee a la source (cf. round_windows.go) ; ce controle est le filet qui interdit
// a une courbe non chronologique d'etre publiee en silence si une autre cause apparait.
//
// Le point ecarte est le point TARDIF-DANS-LA-LISTE mais PRECOCE-DANS-LE-TEMPS : il vient
// forcement d'une manche rangee apres celle dont il porte l'instant, donc c'est lui qui est mal
// range, pas ceux qui le precedent.
//
// L'ECART EST JOURNALISE ICI, une fois, avec le slot et le premier recul : c'est un defaut, pas
// un cas nominal, et il ne doit jamais etre avale. Le journal vit dans la fonction plutot que
// chez ses appelants pour que les DEUX cumuls (par slot ici, par joueur dans `analysis/replay`)
// le rendent de la meme facon, sans dupliquer ni le message ni la decision.
func ChronologicalTotal(pts []ScorePoint) []ScorePoint {
	out := make([]ScorePoint, 0, len(pts))
	last, dropped, recul := 0, 0, 0
	for i, p := range pts {
		if i > 0 && p.TimeMS < last {
			if dropped == 0 {
				recul = p.TimeMS
			}
			dropped++
			continue
		}
		out = append(out, p)
		last = p.TimeMS
	}
	if dropped > 0 {
		slog.Warn("objectiveevents: serie cumulee NON CHRONOLOGIQUE — points ecartes",
			"slot", pts[0].Slot, "ecartes", dropped, "retenus", len(out), "premierRecul", recul)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sortedIntKeys rend les cles entieres d'un groupe de series, dans l'ordre. Elle sert aux
// MANCHES d'un slot (l'ordre du cumul en depend) et aux SLOTS d'entite d'un emplacement
// (depuis le lot 4b, l'ordre de consommation du budget d'evenements en depend : deux slots
// qui se disputent la fin du solde doivent le faire toujours dans le meme ordre, sinon la
// sortie d'un film tronque changerait a chaque execution).
//
// Anciennement `sortedRounds` : le nom disait la premiere des deux, ce qui aurait fait
// ecrire une seconde copie identique pour la seconde.
func sortedIntKeys(bySlot map[int][]ScorePoint) []int {
	out := make([]int, 0, len(bySlot))
	for r := range bySlot {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// sortedSlotKeys rend les emplacements d'une table, dans un ordre TOTAL (composant puis
// cote). Meme raison que ci-dessus : le budget d'evenements rend l'ordre de parcours
// OBSERVABLE sur un film tronque, et un parcours de map ne se rejoue pas a l'identique.
func sortedSlotKeys(table map[statSlotKey]statSlot) []statSlotKey {
	out := make([]statSlotKey, 0, len(table))
	for k := range table {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Comp != out[j].Comp {
			return out[i].Comp < out[j].Comp
		}
		return out[i].Side < out[j].Side
	})
	return out
}

// incrementTimes rend un instant par UNITE gagnee par le compteur : c'est la conversion
// d'un compteur en evenements. `key` ne sert qu'au journal des bornes ; `b` porte le solde
// d'evenements de la passe et n'est jamais nil (cf. [eventBudget]).
//
// La premiere valeur observee est comptee depuis zero — un compteur de recompense part de
// zero au coup d'envoi. Si le film ne montre le slot qu'apres coup, les unites deja
// acquises sont datees de cette premiere emission, ce qui MAJORE leur instant.
//
// # Le garde-fou, et il a ete paye
//
// `prev` ne redescend JAMAIS, sinon la meme unite se compte deux fois apres un creux. Sans
// cela, une seule emission aberrante a -115 faisait remonter le compteur de 0 a 1 en **116**
// evenements (mesure sur `1bc77d2e`, slot 24, comp 0 A). Les emissions negatives elles-memes
// sont ecartees plus tot, par [seriesBySlot].
//
// # Les deux bornes (lot 4b) — cf. l'en-tete des constantes de ce fichier
//
// Un deroulage au-dela de [maxUnrollPerStep] est REJETE : le point n'emet rien et `prev`
// avance quand meme a sa valeur — sinon le point SUIVANT rejouerait le meme ecart geant, et
// la borne n'aurait fait que deplacer l'explosion d'un cran. Le solde de la passe est
// consomme au fur et a mesure ; quand un deroulage n'y tient plus, la passe est TRONQUEE et
// n'emet plus rien, ici comme dans ses appels suivants.
func incrementTimes(pts []ScorePoint, key statSlotKey, b *eventBudget) []int {
	if b.tronque {
		return nil
	}
	var out []int
	prev := int64(0)
	for _, p := range pts {
		n := p.Value - prev // >= 0 : les valeurs negatives sont ecartees par [seriesBySlot]
		if n <= 0 {
			continue
		}
		if n > maxUnrollPerStep {
			b.rejeter(key, p, n)
			prev = p.Value
			continue
		}
		if n > int64(b.reste) {
			b.epuiser(key, p, n)
			return out
		}
		b.reste -= int(n)
		for ; prev < p.Value; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}
