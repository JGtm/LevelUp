package replay

// colline_proprietaire_d2ter_test.go — PHASE D2-ter : LE MEME CANAL, L'ORACLE CONTINU.
//
// TROISIEME ET DERNIER ORACLE. D2 a echoue parce que le score de MODE en KOTH compte des
// collines gagnees ; D2-bis a mesure 87-89 % avec une tolerance de +/- 20 s qui porte
// l'imprecision de l'ORACLE — les `th=10` sont dates au bloc de temps fort, pas a l'action. Les
// 11 a 13 % d'ecart pouvaient donc etre le plancher de bruit de l'oracle, pas l'erreur du canal.
// Le score PERSONNEL est continu : il ne demande aucune tolerance d'appariement, et mesure donc
// le canal lui-meme.
//
// CE QUI EST TESTE : pendant qu'un camp TIENT la colline selon le canal, ce sont les joueurs DE
// CE CAMP dont le score personnel monte.
//
// LE PROTOCOLE — dominance exigee, temoins, seuils, escalade — EST ECRIT ET COMMITE AVANT CE
// FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md`, section D2-ter). Ce qui suit
// l'applique, il ne le decide pas.
//
// REGIME : garde `ZONE_FILM`, un film par processus, lecture seule, AUCUNE base, AUCUN
// changement de production.
//
//	$env:ZONE_FILM="<cache>/film_chunks/01e1f945"; go test ./internal/analysis/replay/ -run CollineProprietaireD2Ter -v

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

const (
	// d2tMinRunSec : duree minimale d'un intervalle. Le score personnel tique a la seconde ;
	// plus court, il n'accumule rien de lisible.
	d2tMinRunSec = 5
	// d2tDominanceFacteur / d2tDominanceMin : la DOMINANCE exigee pour qu'un intervalle tranche.
	//
	// POURQUOI UNE DOMINANCE ET PAS « UN SEUL CAMP MARQUE ». En KOTH le score personnel monte
	// AUSSI par les frags, des deux cotes : l'exclusivite ne se produirait presque jamais. Le
	// facteur 2 ecarte le coude a coude, ou le bruit des frags deciderait ; le plancher de 5
	// points ecarte les intervalles ou personne n'a rien accumule. Les deux sont poses AVANT la
	// mesure.
	d2tDominanceFacteur = 2
	d2tDominanceMin     = 5
	// d2tTemoinDecalageSec : le temoin de decalage. Plus FAIBLE ici qu'en D2-bis — le score
	// s'accumule continument, un glissement ne casse pas tout. C'est le temoin de PERMUTATION
	// qui porte la charge de la preuve, et le dire d'avance evite de surinterpreter celui-ci.
	d2tTemoinDecalageSec = 60
	// d2tObjetSlots : l'objet de mode occupe QUATRE slots consecutifs a partir du designateur
	// ([tag 5][tag 4 proprietaire][tag 4 capteur][tag 3 jauge], cf. zone_states_hill.go).
	//
	// C'EST L'EXCLUSION STRUCTURELLE DU TEMOIN DE PERMUTATION, et la lecon de `606d9844` : en
	// D2-bis le « pire autre canal » elu etait `d.slot+2`, le capteur du MEME objet — un frere,
	// pas une permutation. Une borne structurelle, jamais une liste de slots a maintenir.
	d2tObjetSlots = 4
	// d2tMinIntervalles : sous ce nombre d'intervalles confrontables, le film ne compte NI POUR
	// NI CONTRE.
	d2tMinIntervalles = 6
)

// d2tVerdict porte le resultat d'une confrontation (signal ou temoin).
type d2tVerdict struct {
	runs          int // intervalles nommes assez longs
	confrontables int // ceux ou un camp DOMINE
	sansDominance int
	accord        int
	bijection     string
	// medDom / medPerdu : deltas MEDIANS du camp dominant et du camp domine, sur les
	// intervalles confrontables. DIAGNOSTIC d'ampleur, jamais un critere du gate.
	medDom, medPerdu int64
	doms, perdus     []int64
}

func (v d2tVerdict) taux() float64 {
	if v.confrontables == 0 {
		return 0
	}
	return 100 * float64(v.accord) / float64(v.confrontables)
}

func (v d2tVerdict) String() string {
	return fmt.Sprintf("%d/%d = %.1f %% (intervalles %d, sans dominance %d, %s)",
		v.accord, v.confrontables, v.taux(), v.runs, v.sansDominance, v.bijection)
}

// TestCollineProprietaireD2Ter — LA MESURE. Un film par processus.
func TestCollineProprietaireD2Ter(t *testing.T) {
	e := ctCharge(t)
	c := zoneCtx{origin: e.posUS, step: uint64(e.doc.FrameIntervalMS) * 1000,
		frames: e.doc.FrameCount, intervalMS: e.doc.FrameIntervalMS}
	ser := zoneSeriesOf(e.sc.Reads, c)

	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Fatalf("%s : AUCUN designateur elu — pas de slot voisin a mesurer", e.short)
	}
	ownerSlot := d.slot + 1

	// LE PONT, ET SES DENOMINATEURS. Sans eux, un accord ne se juge pas : un pont qui ne nomme
	// que la moitie des slots mesurerait la moitie du match sans le dire.
	recs := objectiveevents.StatRecords(p2aBobine(t, e.dir))
	deaths, err := ScanFilmDeaths(e.dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", e.short, err)
	}
	identity := objectiveevents.SlotIdentityByDeaths(recs, deathInstantsOf(deaths))
	teams := e.film.p2aTeams()
	slotTeam := map[int]int{}
	for slot, xuid := range identity {
		if tm, ok := teams[xuid]; ok {
			slotTeam[slot] = tm
		}
	}
	t.Logf("%s : designateur slot %d (%d bascule(s)) ; canal slot %d (%d emission(s)) ; "+
		"pont %d slot(s) nomme(s), %d rattache(s) a un camp ; axe %d frames a %d ms",
		e.short, d.slot, len(d.changes), ownerSlot, len(ser.owner[ownerSlot]),
		len(identity), len(slotTeam), e.doc.FrameCount, e.doc.FrameIntervalMS)
	if len(slotTeam) < 2 {
		t.Fatalf("%s : le pont rattache %d slot(s) a un camp — sans les deux camps l'oracle ne "+
			"peut rien trancher", e.short, len(slotTeam))
	}

	perso := objectiveevents.SeriesTotal(recs, objectiveevents.PersonalScoreComponent, false)
	minFrames := d2tMinRunSec * 1000 / max(e.doc.FrameIntervalMS, 1)
	runs := d2Runs(ser.owner[ownerSlot], e.doc.FrameCount, minFrames)
	sig := d2tConfronte(runs, perso, slotTeam, e)
	t.Logf("SIGNAL    %s : %s", e.short, sig)
	// DIAGNOSTIC DE L'ORACLE — pose APRES le verdict, il ne le change pas : il le NOMME.
	// L'AMPLEUR des deltas dit de quoi le score personnel est fait pendant un intervalle. Un
	// tic de colline vaut quelques points ; un frag en vaut environ cent. Si le delta dominant
	// median se compte en centaines, l'oracle ne mesure pas la garde mais les frags.
	t.Logf("DIAGNOSTIC %s : delta personnel par intervalle confrontable — dominant median %d, "+
		"domine median %d (un frag vaut ~100, un tic de colline quelques points)",
		e.short, sig.medDom, sig.medPerdu)

	if sig.confrontables < d2tMinIntervalles {
		t.Logf("NON EXPLOITABLE %s : %d intervalle(s) confrontable(s) < %d — ce film ne compte NI "+
			"POUR NI CONTRE", e.short, sig.confrontables, d2tMinIntervalles)
	} else {
		t.Logf("EXPLOITABLE     %s : %d intervalle(s) confrontable(s) >= %d",
			e.short, sig.confrontables, d2tMinIntervalles)
	}

	// TEMOIN (a) — PERMUTATION, slots FRERES EXCLUS STRUCTURELLEMENT.
	meilleur, mSlot, autres := d2tVerdict{}, uint32(0), 0
	for _, s := range sortedZoneSlots(ser.owner) {
		if s >= d.slot && s < d.slot+d2tObjetSlots {
			continue // frere du meme objet de mode : ce n'est pas une permutation
		}
		v := d2tConfronte(d2Runs(ser.owner[s], e.doc.FrameCount, minFrames), perso, slotTeam, e)
		if v.confrontables == 0 {
			continue
		}
		autres++
		if v.taux() > meilleur.taux() {
			meilleur, mSlot = v, s
		}
	}
	if autres == 0 {
		t.Logf("TEMOIN a  %s : aucun autre canal confrontable hors de l'objet de mode — "+
			"temoin sans objet", e.short)
	} else {
		t.Logf("TEMOIN a  %s : permutation (freres %d..%d exclus), pire des %d autres canaux "+
			"(slot %d) : %s", e.short, d.slot, d.slot+d2tObjetSlots-1, autres, mSlot, meilleur)
	}

	// TEMOIN (b) — DECALAGE +60 s. Plus faible ici, et annonce comme tel.
	shift := d2tTemoinDecalageSec * 1000 / max(e.doc.FrameIntervalMS, 1)
	dec := d2tConfronte(d2Decale(runs, shift, e.doc.FrameCount), perso, slotTeam, e)
	t.Logf("TEMOIN b  %s : decalage +%d s (temoin FAIBLE par nature) : %s",
		e.short, d2tTemoinDecalageSec, dec)
}

// d2tConfronte confronte chaque intervalle de propriete au camp dont le score personnel DOMINE,
// et rend l'accord sous la meilleure bijection valeur <-> camp.
func d2tConfronte(runs []d2Run, perso map[int][]objectiveevents.ScorePoint, slotTeam map[int]int,
	e ctEntree,
) d2tVerdict {
	v := d2tVerdict{runs: len(runs)}
	paires := map[uint64]map[int]int{}
	for _, r := range runs {
		parCamp := map[int]int64{}
		for slot, tm := range slotTeam {
			parCamp[tm] += d2Delta(perso[slot], e, r.t0, r.t1)
		}
		dom, ok := d2tDominant(parCamp)
		if !ok {
			v.sansDominance++
			continue
		}
		v.confrontables++
		v.doms = append(v.doms, parCamp[dom])
		for tm, dd := range parCamp {
			if tm != dom {
				v.perdus = append(v.perdus, dd)
			}
		}
		if paires[r.v] == nil {
			paires[r.v] = map[int]int{}
		}
		paires[r.v][dom]++
	}
	v.accord, v.bijection = d2MeilleureBijection(paires, []int{0, 1})
	v.medDom, v.medPerdu = d2tMediane(v.doms), d2tMediane(v.perdus)
	return v
}

// d2tMediane rend la mediane d'une serie, 0 si vide.
func d2tMediane(vs []int64) int64 {
	if len(vs) == 0 {
		return 0
	}
	s := append([]int64(nil), vs...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

// d2tDominant rend le camp qui DOMINE l'intervalle, ou (0, false) quand aucun ne domine assez.
// La regle est celle du protocole : max >= 2 x min ET max >= 5 points.
func d2tDominant(parCamp map[int]int64) (int, bool) {
	if len(parCamp) < 2 {
		return 0, false
	}
	camps := make([]int, 0, len(parCamp))
	for tm := range parCamp {
		camps = append(camps, tm)
	}
	sort.Ints(camps)
	best, second, bestTeam := int64(-1), int64(-1), 0
	for _, tm := range camps {
		if d := parCamp[tm]; d > best {
			best, second, bestTeam = d, best, tm
		} else if d > second {
			second = d
		}
	}
	if best < d2tDominanceMin || best < int64(d2tDominanceFacteur)*second {
		return 0, false
	}
	return bestTeam, true
}
