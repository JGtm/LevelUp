package objectiveevents

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// score_measure_rounds_test.go — la grammaire d'ancrage ETENDUE de l'item A.0b.1 : lire les
// enregistrements statborg que la grammaire de production REJETTE, et reconstruire le score
// total d'un match a plusieurs manches.
//
// # La cause, mesuree et non supposee
//
// `matchRecordHeader` (statborg.go) n'accepte qu'UNE forme de liste de composants : le bit qui
// suit les 2 bits de generation doit valoir 0, puis vient un compte sur 3 bits et autant
// d'index de 6 bits. Or le moteur en a DEUX (`filmdec.consumeMask`, FUN_1406d7610) :
//
//	gate = 0 : R(3) compte + compte x R(6) index   — la liste CREUSE, seule connue de l'ancrage
//	gate = 1 : R(64) masque DENSE                  — REJETEE par l'ancrage
//
// Un enregistrement qui change plus de 7 composants d'un coup passe forcement en forme dense :
// c'est exactement ce qui arrive a la FIN D'UNE MANCHE, ou les 28 compteurs de la manche se
// figent (i28..i55) tandis que les 28 compteurs courants (i0..i27) repartent de zero. La
// grammaire de production perd donc l'entite d'equipe a la premiere fin de manche — mesure du
// 2026-08-17 sur `24dbb67d` (Oddball) : plus une seule emission de score apres 290 683 ms sur un
// match de 519 s, alors que la chaine y voit 1 627 records statborg propres contre 1 104 avant.
//
// # Les deux familles de composants, et pourquoi il faut les deux
//
//	i0..i27  `statborg-current-round-value-stat-component`     la manche EN COURS
//	i28..i55 `statborg-finalized-rounds-values-stat-component` les manches FIGEES, par manche
//	i56      `statborg-round-outcomes-component`               32 x R(2)
//	i57      `statborg-entry-index-and-type-component`         R(32) + R(8)
//
// Le score d'un match = somme des manches figees + manche en cours. Le getter natif le dit dans
// la table ECS : `value = *(int32*)(world + slot*0x88 + equipe*0x1DF0 + 0x38 + manche*4)` — la
// valeur est indexee PAR MANCHE.
//
// Ceci est un INSTRUMENT (fichier de test) : aucune ligne de production n'est modifiee. Si la
// mesure tient, le portage de cette grammaire dans `statborg.go` est une decision de phase 1.

// Familles de composants de l'archetype statborg, bornes lues dans `ecs_table.tsv`.
const (
	// statCurRoundMax est le premier index qui n'est plus un compteur de manche en cours.
	statCurRoundMax = 28
	// statFinalizedMax est le premier index qui n'est plus une valeur de manche figee.
	statFinalizedMax = 56
	// statOutcomesComp / statEntryIndexComp sont les deux composants de queue.
	statOutcomesComp   = 56
	statEntryIndexComp = 57
	// statRoundMaskBits est la largeur du masque de manches d'un composant figee.
	statRoundMaskBits = 32
)

// finalizedValue est la valeur figee d'un compteur pour une manche donnee.
type finalizedValue struct {
	TimeMS, Slot, Comp, Round int
	Value                     int64
}

// curValue est la valeur d'un compteur de manche en cours, AVEC ses deux en-tetes de 5 bits.
//
// # L'hypothese « en-tete = numero de manche »
//
// La production les EXIGE nuls (`decodeComponents`, statborg.go) et les jette. Or le getter natif
// du composant est indexe par MANCHE :
//
//	value = *(int32*)(world + slot*0x88 + equipe*0x1DF0 + 0x38 + manche*4)
//
// Si l'un des deux en-tetes porte ce numero de manche, l'assertion « == 0 » de la production
// REJETTE mecaniquement les enregistrements des manches suivantes — ce qui expliquerait d'un seul
// coup les trois observations de la phase 0-bis sur Oddball : les entites d'equipe emettent bien
// apres la fin de la manche 1, aucun segment de score n'y est trouve, et les frags des joueurs
// valent la moitie de l'oracle.
//
// L'instrument relache donc l'assertion a [statHdrMaxRelaxed] et PUBLIE les en-tetes : c'est la
// mesure qui doit dire s'ils sont un index de manche.
type curValue struct {
	A, B   int64
	H1, H2 int
}

// statHdrMaxRelaxed borne les en-tetes acceptes par l'instrument. Huit valeurs suffisent (un match
// ne se joue pas en plus de huit manches) et la borne garde 2 bits de contrainte sur chacun des
// deux en-tetes, soit 4 des 10 bits de filtre anti-faux-positifs d'origine.
const statHdrMaxRelaxed = 7

// statRecordExt est un enregistrement lu par la grammaire etendue.
type statRecordExt struct {
	TimeMS int
	Slot   int
	// Form porte la forme d en-tete lue : generation, selecteur de base, forme de la liste.
	Form headerForm
	// Cur porte les compteurs de la manche en cours (i0..i27), en-tetes compris.
	Cur map[int]curValue
	// Fin porte les valeurs figees rencontrees dans cet enregistrement.
	Fin []finalizedValue
}

// statRecordsExt decode un film avec la grammaire etendue. Meme balayage et meme horloge que
// [StatRecords] : seules la forme de liste et les familles de composants changent.
func statRecordsExt(film *filmsource.Film) []statRecordExt {
	var out []statRecordExt
	for _, c := range manifestChunks(film) {
		frames := framesOf(film, c.pos)
		if len(frames) == 0 {
			continue
		}
		base := frames[0].TS
		for _, f := range frames {
			tMS := c.meta.StartMS + int((f.TS-base)/1000)
			out = append(out, scanFrameExt(f.Payload, tMS)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// scanFrameExt balaie un paquet FRAME avec la grammaire etendue.
func scanFrameExt(pay []byte, tMS int) []statRecordExt {
	var out []statRecordExt
	lim := len(pay)*8 - statTailBits
	for b := 0; b < lim; b++ {
		slot, idx, at, form, ok := matchHeaderExt(pay, b)
		if !ok {
			continue
		}
		cur, fin, ok := decodeCompsExt(pay, at, idx, tMS, slot)
		if !ok || (len(cur) == 0 && len(fin) == 0) {
			continue
		}
		out = append(out, statRecordExt{TimeMS: tMS, Slot: slot, Form: form, Cur: cur, Fin: fin})
	}
	return out
}

// Grammaire EXACTE de l'en-tete d'un record DELTA, telle que la chaine la porte :
//
//	readRecordType   R(1) ; a 1 = DELTA (sinon R(2) : NEW / DEL / fin)
//	readRecordID     R(13) identifiant bas + R(2) tag = GENERATION
//	decodeDelta      R(1) selecteur de base ; a 1 -> R(7) (largeur non consommee ici : un record
//	                 porteur d un etat de base est rejete, cf. matchHeaderExt)
//	consumeMask      R(1) gate ; 0 -> R(3) compte + compte x R(6) ; 1 -> R(64) dense
//
// Ce que l'ancrage de production en avait fait, et c'est la cle de tout : ses « 14 bits de
// slot » sont les 13 bits d'identifiant PLUS le premier bit du tag, et ses « 2 bits constants
// a 0b10 » sont le second bit du tag PLUS le selecteur de base. Sa contrainte encode donc, sans
// le dire, deux hypotheses : GENERATION = 1 et SELECTEUR DE BASE = 0. D'ou son slot toujours
// pair (2 x identifiant) et son rejet silencieux de tout record de generation differente ou
// porteur d'un etat de base explicite.
const (
	// statIDLowBits est la largeur de l'identifiant bas d'un record.
	statIDLowBits = 13
	// statTagBits est la largeur du tag de generation.
	statTagBits = 2
	// statLowMin / statLowMax bornent l'identifiant bas des entites statborg : les slots
	// d'ancrage 6..24 pairs valent 2 x identifiant, donc l'identifiant va de 3 a 12.
	statLowMin = 3
	statLowMax = 12
	// statCalibratedGen est la generation que la production presuppose (son « 0b10 » vaut
	// generation 1 + selecteur de base a 0).
	statCalibratedGen = 1
)

// extLowMax borne l'identifiant bas accepte par l'instrument. Il vaut [statLowMax] par defaut ;
// `SCORE_LOW_MAX` l'eleve pour chercher les entites statborg RECREEES apres une fin de manche,
// que le diagnostic de la chaine voit sur des identifiants bien plus hauts (576, 589, 606 ... sur
// `24dbb67d`, avec 306 et 389 records propres — l'ordre de grandeur des 268 du slot 3).
var extLowMax = statLowMax

// headerForm decrit la forme d'en-tete d'un enregistrement lu — c'est la grandeur MESUREE de
// l'item A.0b.1 : elle dit lesquelles des hypotheses de production sont fausses, et ou.
type headerForm struct {
	// Gen est le tag de generation lu.
	Gen int
	// Baseline dit que le selecteur d'etat de base valait 1 (donc R(7) lu ensuite).
	Baseline bool
	// Dense dit que la liste de composants etait un masque de 64 bits.
	Dense bool
}

// matchHeaderExt teste l'en-tete d'un record DELTA sur la grammaire exacte, sans presumer ni la
// generation, ni le selecteur de base, ni la forme de la liste.
func matchHeaderExt(pay []byte, b int) (slot int, idx []int, compAt int, form headerForm, ok bool) {
	if readBitsBE(pay, b, 1) != 1 {
		return 0, nil, 0, form, false
	}
	low := int(readBitsBE(pay, b+1, statIDLowBits))
	if low < statLowMin || low > extLowMax {
		return 0, nil, 0, form, false
	}
	q := b + 1 + statIDLowBits
	form.Gen = int(readBitsBE(pay, q, statTagBits))
	q += statTagBits
	form.Baseline = readBitsBE(pay, q, 1) == 1
	q++
	// Generation et selecteur de base sont RE-CONTRAINTS a la valeur calibree : les relacher
	// ouvre 2 bits de faux positifs, mesure sur `24dbb67d` (total de frags a 12 677 437 729 et
	// des manches figees d'index 14, 24, 31 inexistantes). Le diagnostic de la chaine montre
	// d'ailleurs que TOUTES les entites statborg vues apres la fin de la manche 1 sont de
	// generation 1 : le relachement n'etait pas justifie par les faits.
	if form.Gen != statCalibratedGen || form.Baseline {
		return 0, nil, 0, form, false
	}
	if readBitsBE(pay, q, 1) == 0 {
		idx, compAt, ok = sparseList(pay, q+1)
	} else {
		form.Dense = true
		idx, compAt, ok = denseList(pay, q+1)
	}
	return low * 2, idx, compAt, form, ok
}

// sparseList lit la forme gate=0 : R(3) compte + compte x R(6) index, strictement croissants.
func sparseList(pay []byte, p int) ([]int, int, bool) {
	n := int(readBitsBE(pay, p, 3))
	if n < 1 || n > statMaxCompPerRecord {
		return nil, 0, false
	}
	idx := make([]int, n)
	prev := -1
	for i := 0; i < n; i++ {
		idx[i] = int(readBitsBE(pay, p+3+statCompIndexBits*i, statCompIndexBits))
		if idx[i] >= statMaxComp || idx[i] <= prev {
			return nil, 0, false
		}
		prev = idx[i]
	}
	return idx, p + 3 + statCompIndexBits*n, true
}

// denseList lit la forme gate=1 : R(64) masque dense. Les bits au-dela de l'archetype doivent
// etre nuls — c'est la contrainte qui remplace celle des en-tetes de composant.
func denseList(pay []byte, p int) ([]int, int, bool) {
	if p+statDenseMaskBits > len(pay)*8 {
		return nil, 0, false
	}
	mask := readBitsBE(pay, p, statDenseMaskBits)
	if mask == 0 || mask>>statMaxComp != 0 {
		return nil, 0, false
	}
	var idx []int
	for i := 0; i < statMaxComp; i++ {
		if mask>>uint(i)&1 == 1 {
			idx = append(idx, i)
		}
	}
	return idx, p + statDenseMaskBits, true
}

// decodeCompsExt lit les composants annonces, chacun avec la grammaire de SA famille. Une
// famille inconnue ou une lecture hors bornes arrete la chaine : la largeur d'un composant
// commande la position du suivant, donc on ne devine jamais.
func decodeCompsExt(pay []byte, at int, idx []int, tMS, slot int) (map[int]curValue, []finalizedValue, bool) {
	cur := map[int]curValue{}
	var fin []finalizedValue
	q := at
	for i, id := range idx {
		switch {
		case id < statCurRoundMax:
			// L'assertion de production (les deux en-tetes valent 0) est RELACHEE au premier
			// composant : si l'en-tete est un numero de manche, exiger 0 rejette toutes les
			// manches suivantes. Les en-tetes sont conserves et publies.
			h1 := int(readBitsBE(pay, q, statHdrBits))
			h2 := int(readBitsBE(pay, q+statHdrBits, statHdrBits))
			if i == 0 && (h1 > statHdrMaxRelaxed || h2 > statHdrMaxRelaxed) {
				return nil, nil, false
			}
			v, w, ok := decodeStatComponent(pay, q)
			if !ok {
				return cur, fin, len(cur) > 0 || len(fin) > 0
			}
			cur[id] = curValue{A: v.A, B: v.B, H1: h1, H2: h2}
			q += w
		case id < statFinalizedMax:
			vals, w, ok := decodeFinalizedComponent(pay, q)
			if !ok {
				return cur, fin, len(cur) > 0 || len(fin) > 0
			}
			for round, v := range vals {
				fin = append(fin, finalizedValue{TimeMS: tMS, Slot: slot,
					Comp: id - statCurRoundMax, Round: round, Value: v})
			}
			q += w
		case id == statOutcomesComp:
			q += 2 * statRoundMaskBits // 32 issues de manche sur 2 bits
		case id == statEntryIndexComp:
			q += 40 // R(32) + R(8)
		default:
			return nil, nil, false
		}
		if q > len(pay)*8 {
			return nil, nil, false
		}
	}
	return cur, fin, true
}

// decodeFinalizedComponent lit un `statborg-finalized-rounds-values-stat-component` :
// R(32) masque de manches, puis par bit a 1 deux valeurs conditionnelles
// `R(1)[si 0 : varW]`. Rend la valeur A par manche (la valeur B suit la meme forme).
//
// Grammaire reprise de `filmdec.consumeStatborgFinalized` (`components_batch3.go`), qui la
// tient de FUN_142ed3c50 — elle n'est pas devinee ici.
func decodeFinalizedComponent(pay []byte, p int) (map[int]int64, int, bool) {
	if p+statRoundMaskBits > len(pay)*8 {
		return nil, 0, false
	}
	mask := readBitsBE(pay, p, statRoundMaskBits)
	q := p + statRoundMaskBits
	out := map[int]int64{}
	for i := 0; i < statRoundMaskBits; i++ {
		if mask>>uint(i)&1 == 0 {
			continue
		}
		for side := 0; side < 2; side++ {
			if q+1 > len(pay)*8 {
				return nil, 0, false
			}
			present := readBitsBE(pay, q, 1) == 0
			q++
			if !present {
				continue
			}
			v, w, ok := readStatVarWidth(pay, q)
			if !ok {
				return nil, 0, false
			}
			if side == 0 {
				out[i] = v
			}
			q += w
		}
	}
	return out, q - p, true
}

// roundsTotal reconstruit, par slot, le score total d'un compteur : somme des manches FIGEES
// plus la manche EN COURS.
//
// Deux precautions mesurees : (1) une valeur figee ne bouge plus, donc pour une manche donnee on
// retient la valeur la plus frequente — un faux positif isole ne peut pas l'emporter sur les
// emissions repetees ; (2) la manche en cours est prise APRES le dernier instant de finalisation,
// et filtree par la meme plus longue suite croissante que la production.
func roundsTotal(recs []statRecordExt, comp int) map[int]int64 {
	votes := map[int]map[int]map[int64]int{} // slot -> manche -> valeur -> voix
	lastFin := map[int]int{}                 // slot -> instant de la derniere finalisation
	for _, r := range recs {
		for _, f := range r.Fin {
			if f.Comp != comp {
				continue
			}
			if votes[f.Slot] == nil {
				votes[f.Slot] = map[int]map[int64]int{}
			}
			if votes[f.Slot][f.Round] == nil {
				votes[f.Slot][f.Round] = map[int64]int{}
			}
			votes[f.Slot][f.Round][f.Value]++
			if f.TimeMS > lastFin[f.Slot] {
				lastFin[f.Slot] = f.TimeMS
			}
		}
	}
	out := map[int]int64{}
	for slot, byRound := range votes {
		for _, byVal := range byRound {
			out[slot] += majorityValue(byVal)
		}
	}
	// Manche en cours : les emissions posterieures a la derniere finalisation.
	cur := map[int][]ScorePoint{}
	for _, r := range recs {
		v, ok := r.Cur[comp]
		if !ok || v.A < 0 || r.TimeMS < lastFin[r.Slot] {
			continue
		}
		cur[r.Slot] = append(cur[r.Slot], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	for slot, pts := range cur {
		kept := longestRun(pts, true)
		if len(kept) > 0 {
			out[slot] += kept[len(kept)-1].Value
		}
	}
	return out
}

// majorityValue rend la valeur la plus souvent vue (la plus grande en cas d'egalite, pour etre
// deterministe).
func majorityValue(byVal map[int64]int) int64 {
	best, bestN := int64(0), 0
	for v, n := range byVal {
		if n > bestN || (n == bestN && v > best) {
			best, bestN = v, n
		}
	}
	return best
}

// roundsSeen rend, par slot, les index de manche figee vus pour un compteur.
func roundsSeen(recs []statRecordExt, comp int) map[int][]int {
	seen := map[int]map[int]bool{}
	for _, r := range recs {
		for _, f := range r.Fin {
			if f.Comp != comp {
				continue
			}
			if seen[f.Slot] == nil {
				seen[f.Slot] = map[int]bool{}
			}
			seen[f.Slot][f.Round] = true
		}
	}
	out := map[int][]int{}
	for slot, rs := range seen {
		for r := range rs {
			out[slot] = append(out[slot], r)
		}
		sort.Ints(out[slot])
	}
	return out
}
