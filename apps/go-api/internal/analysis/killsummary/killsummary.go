// Package killsummary — les résumés par match du journal des morts (ADR 0039).
//
// Une passe du journal (`match_kill_events` : toutes les morts d'un match sous un même
// `decode_pass`) se résume en trois familles, calculées UNE fois, quand la passe est écrite :
//
//	S0 tête    (match)                                         publiable + quatre comptes
//	S1 paires  (tueur xuid, victime xuid, source)              frags
//	S2 appuis  (assistant, tueur, victime si les deux sans xuid)  appuis, parts, tranches
//
// La fonction est TOTALE : aucune erreur, aucune entrée refusée. Elle lit un type de ligne
// neutre ([Row]) que deux adaptateurs remplissent (les morts en mémoire d'une passe qu'on
// écrit, les lignes d'une passe relue en base) et applique les normalisations de l'écriture du
// journal (`persist.killEventArgs`) : un texte vide se lit NULL, `SourceTag == 0` se lit
// « source non mesurée » (NULL). Les NULL restent des groupes : tueur absent, victime bot,
// source inconnue.
//
// Toute modification du grain, d'une normalisation, d'un compte ou des bornes de tranche
// (`domain.AssistTierLowMaxPct` / `AssistTierMidMaxPct`) exige une montée de [Rev] : le test
// d'empreinte du paquet (rev_test.go) hache ses sources et ces bornes, et échoue sinon.
package killsummary

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/domain"
)

// Rev : la révision de la règle de résumé, écrite dans `summary_rev` de chaque ligne. Les
// lecteurs ne filtrent jamais dessus (les vues servent la dernière génération) ; elle dit
// quelles têtes sont périmées et doivent être réécrites.
const Rev = "ks1"

// Row : une mort de la passe. Texte vide = NULL ; SourceTag 0 = source non mesurée ; une part
// nil = non mesurée (jamais zéro).
type Row struct {
	VictimXUID      string
	KillerXUID      string
	KillerGamertag  string
	AssistXUID      string
	AssistGamertag  string
	AssistKnown     bool
	SourceTag       uint32
	KillerDamagePct *uint8
	AssistDamagePct *uint8
}

// Head : la tête (S0), une par passe résumée. Les comptes portent sur toutes les morts de la
// passe ; `Publishable` est un attribut de la passe, constant sur ses lignes.
type Head struct {
	Publishable bool
	// Rows : morts de la passe.
	Rows int
	// AssistKnown : morts auxquelles un kill-event a été attaché (assistant lu, ou absence
	// d'assistant mesurée).
	AssistKnown int
	// KillerXUID : morts dont le tueur crédité porte un xuid.
	KillerXUID int
	// SourceAndKiller : morts dont la source est mesurée ET le tueur porte un xuid.
	SourceAndKiller int
}

// PairKey : le grain de S1. Champ vide / 0 = NULL, gardé comme groupe.
type PairKey struct {
	KillerXUID string
	VictimXUID string
	SourceTag  uint32
}

// Pair : une ligne de S1.
type Pair struct {
	PairKey
	Kills int
}

// AssistKey : le grain de S2. `VictimXUID` n'est renseigné que quand l'assistant ET le tueur
// sont sans xuid (bots) ; sinon il est vide et les victimes se cumulent.
type AssistKey struct {
	AssistXUID     string
	AssistGamertag string
	KillerXUID     string
	KillerGamertag string
	VictimXUID     string
}

// AssistPair : une ligne de S2. Une moyenne se lit `PctSum / PctCount`, indéfinie quand
// `PctCount` vaut 0 (jamais 0 %).
type AssistPair struct {
	AssistKey
	// N : morts du groupe, assistant nommé ou absence d'assistant mesurée.
	N int
	// Stolen : morts dont la part de l'assistant dépasse celle du tueur (deux parts mesurées).
	Stolen int
	// PctSum / PctCount : somme et nombre des parts d'assistant mesurées.
	PctSum   int
	PctCount int
	// Low / Mid / High : parts mesurées par tranche (bornes du paquet domain).
	Low  int
	Mid  int
	High int
}

// Summary : les trois familles d'une passe, dans un ordre déterministe (clé croissante).
type Summary struct {
	Rev     string
	Head    Head
	Pairs   []Pair
	Assists []AssistPair
}

// Summarize résume une passe. S2 ne retient que les morts d'une passe publiable dont
// l'assistant est connu (nommé ou absent mesuré).
func Summarize(publishable bool, rows []Row) Summary {
	s := Summary{Rev: Rev, Head: Head{Publishable: publishable, Rows: len(rows)}}
	pairs := map[PairKey]int{}
	assists := map[AssistKey]*AssistPair{}
	for i := range rows {
		r := &rows[i]
		countHead(&s.Head, r)
		pairs[PairKey{KillerXUID: r.KillerXUID, VictimXUID: r.VictimXUID, SourceTag: r.SourceTag}]++
		if publishable && r.AssistKnown {
			addAssist(assists, r)
		}
	}
	s.Pairs = sortedPairs(pairs)
	s.Assists = sortedAssists(assists)
	return s
}

// countHead ajoute une mort aux comptes de la tête.
func countHead(h *Head, r *Row) {
	if r.AssistKnown {
		h.AssistKnown++
	}
	if r.KillerXUID != "" {
		h.KillerXUID++
		if r.SourceTag != 0 {
			h.SourceAndKiller++
		}
	}
}

// addAssist ajoute une mort au groupe d'appui qui la porte.
func addAssist(groups map[AssistKey]*AssistPair, r *Row) {
	k := AssistKey{
		AssistXUID: r.AssistXUID, AssistGamertag: r.AssistGamertag,
		KillerXUID: r.KillerXUID, KillerGamertag: r.KillerGamertag,
	}
	if r.AssistXUID == "" && r.KillerXUID == "" {
		k.VictimXUID = r.VictimXUID
	}
	g := groups[k]
	if g == nil {
		g = &AssistPair{AssistKey: k}
		groups[k] = g
	}
	g.N++
	if r.AssistDamagePct == nil {
		return
	}
	pct := int(*r.AssistDamagePct)
	if r.KillerDamagePct != nil && pct > int(*r.KillerDamagePct) {
		g.Stolen++
	}
	g.PctSum += pct
	g.PctCount++
	switch {
	case pct < domain.AssistTierLowMaxPct:
		g.Low++
	case pct <= domain.AssistTierMidMaxPct:
		g.Mid++
	default:
		g.High++
	}
}

func sortedPairs(m map[PairKey]int) []Pair {
	out := make([]Pair, 0, len(m))
	for k, n := range m {
		out = append(out, Pair{PairKey: k, Kills: n})
	}
	slices.SortFunc(out, func(a, b Pair) int {
		return cmp.Or(cmp.Compare(a.KillerXUID, b.KillerXUID),
			cmp.Compare(a.VictimXUID, b.VictimXUID), cmp.Compare(a.SourceTag, b.SourceTag))
	})
	return out
}

func sortedAssists(m map[AssistKey]*AssistPair) []AssistPair {
	out := make([]AssistPair, 0, len(m))
	for _, g := range m {
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b AssistPair) int {
		return cmp.Or(cmp.Compare(a.AssistXUID, b.AssistXUID),
			cmp.Compare(a.AssistGamertag, b.AssistGamertag),
			cmp.Compare(a.KillerXUID, b.KillerXUID),
			cmp.Compare(a.KillerGamertag, b.KillerGamertag),
			cmp.Compare(a.VictimXUID, b.VictimXUID))
	})
	return out
}
