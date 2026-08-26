package replay

// colline_proprietaire_d2bis_test.go — PHASE D2-bis : LE MEME CANAL, UN ORACLE DE BASCULES.
//
// D2 a rendu un negatif dont la cause est nommee : le score de MODE en KOTH compte des collines
// GAGNEES (3-2, 4-2 : les scores de l'API), pas des secondes de garde, et deux films sur quatre
// n'en portent qu'un camp. Les evenements `th=10` de PRISE, eux, existent independamment de ce
// compteur — ils sont lus au footer, un par prise, avec l'acteur en xuid. Les quatre films
// redeviennent donc exploitables.
//
// LE CANAL TESTE NE CHANGE PAS : toujours le tag 4 du slot voisin du designateur. Seul l'oracle
// change, et c'est tout l'interet — un canal qui aurait « marche » avec un oracle et pas l'autre
// serait suspect ; ici c'est l'oracle qui etait en defaut, pas la mesure.
//
// CE QUI EST TESTE, EN UNE PHRASE : a chaque prise de colline, le canal de propriete BASCULE vers
// le camp du preneur.
//
// LE PROTOCOLE — tolerance, temoins, seuils, regle d'escalade — EST ECRIT ET COMMITE AVANT CE
// FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md`, section D2-bis, commit dedie). Ce
// qui suit l'applique, il ne le decide pas.
//
// REGIME : garde `ZONE_FILM`, un film par processus, lecture seule, AUCUNE base ouverte, AUCUN
// changement de production.
//
//	$env:ZONE_FILM="<cache>/film_chunks/01e1f945"; go test ./internal/analysis/replay/ -run CollineProprietaireD2Bis -v

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

const (
	// d2bToleranceSec : la fenetre d'appariement prise <-> bascule. VINGT SECONDES, et cette
	// valeur vient de la DONNEE : le decodeur qualifie lui-meme ces evenements d'`approx
	// (~5-20s)` (`extractFromTh10`) — l'instant lu est celui du bloc de temps fort, pas celui de
	// l'action. C'est la borne haute de l'imprecision documentee, et trancher plus finement que
	// l'oracle ne le permet serait inventer de la precision.
	d2bToleranceSec = 20
	// d2bTemoinDecalageSec : le VRAI temoin de decalage. Trois fois la tolerance.
	//
	// POURQUOI PAS 20 s, QUI ETAIT LA CONSIGNE : un decalage de 20 s tombe EXACTEMENT DANS une
	// fenetre de +/- 20 s. Il ne deplacerait pas l'appariement et rendrait donc le meme taux que
	// le signal — un temoin qui ne peut pas echouer n'est pas un temoin.
	d2bTemoinDecalageSec = 60
	// d2bControleDecalageSec : le +20 s de la consigne, mesure et publie QUAND MEME pour la
	// continuite avec D2, mais lu comme un controle de stabilite INTERNE a la fenetre — jamais
	// comme un temoin negatif.
	d2bControleDecalageSec = 20
	// d2bMinPrises : sous ce nombre de prises confrontables, un film ne compte NI POUR NI CONTRE.
	// Regle d'escalade ecrite d'avance : un denominateur insuffisant ne se compense pas.
	d2bMinPrises = 6
)

// d2bPrise est une prise de colline posee sur l'axe de frames, avec le camp de son auteur.
type d2bPrise struct {
	frame int
	team  int
}

// d2bBascule est un changement du canal de propriete vers une valeur NOMMEE.
type d2bBascule struct {
	frame int
	v     uint64
}

// d2bVerdict porte le resultat d'une confrontation (signal ou temoin).
type d2bVerdict struct {
	prises        int // prises posees sur l'axe, avec camp connu
	confrontables int // celles qui trouvent une bascule dans la tolerance
	sansBascule   int
	accord        int
	bijection     string
	// ecarts : |prise - bascule| en frames, pour la mediane (diagnostic, pas un seuil).
	ecarts []int
}

func (v d2bVerdict) taux() float64 {
	if v.confrontables == 0 {
		return 0
	}
	return 100 * float64(v.accord) / float64(v.confrontables)
}

func (v d2bVerdict) String() string {
	med := "-"
	if n := len(v.ecarts); n > 0 {
		s := append([]int(nil), v.ecarts...)
		sort.Ints(s)
		med = fmt.Sprintf("%d", s[n/2])
	}
	return fmt.Sprintf("%d/%d = %.1f %% (prises %d, sans bascule %d, ecart median %s frames, %s)",
		v.accord, v.confrontables, v.taux(), v.prises, v.sansBascule, med, v.bijection)
}

// TestCollineProprietaireD2Bis — LA MESURE. Un film par processus.
func TestCollineProprietaireD2Bis(t *testing.T) {
	e := ctCharge(t)
	c := zoneCtx{origin: e.posUS, step: uint64(e.doc.FrameIntervalMS) * 1000,
		frames: e.doc.FrameCount, intervalMS: e.doc.FrameIntervalMS}
	ser := zoneSeriesOf(e.sc.Reads, c)

	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Fatalf("%s : AUCUN designateur elu — pas de slot voisin a mesurer", e.short)
	}
	ownerSlot := d.slot + 1
	t.Logf("%s : designateur slot %d (%d bascule(s)) ; canal de propriete slot %d (%d emission(s)) ; "+
		"axe %d frames a %d ms", e.short, d.slot, len(d.changes), ownerSlot,
		len(ser.owner[ownerSlot]), e.doc.FrameCount, e.doc.FrameIntervalMS)

	prises := d2bPrises(t, e)
	tol := d2bFrames(d2bToleranceSec, e)
	t.Logf("%s : %d prise(s) de colline posee(s) sur l'axe, camp connu ; tolerance %d s = %d frames",
		e.short, len(prises), d2bToleranceSec, tol)

	sig := d2bConfronte(prises, d2bBascules(ser.owner[ownerSlot], e.doc.FrameCount), tol)
	t.Logf("SIGNAL    %s : %s", e.short, sig)

	// EXPLOITABILITE — la regle d'escalade, appliquee AVANT de lire le taux.
	if sig.confrontables < d2bMinPrises {
		t.Logf("NON EXPLOITABLE %s : %d prise(s) confrontable(s) < %d — ce film ne compte NI POUR "+
			"NI CONTRE (denominateur insuffisant)", e.short, sig.confrontables, d2bMinPrises)
	} else {
		t.Logf("EXPLOITABLE     %s : %d prise(s) confrontable(s) >= %d",
			e.short, sig.confrontables, d2bMinPrises)
	}

	// TEMOIN (a) — PERMUTATION : le meme test sur les AUTRES canaux de propriete du film. On
	// publie le MEILLEUR d'entre eux, c'est-a-dire le temoin le plus DUR.
	meilleur, mSlot, autres := d2bVerdict{}, uint32(0), 0
	for _, s := range sortedZoneSlots(ser.owner) {
		if s == ownerSlot {
			continue
		}
		v := d2bConfronte(prises, d2bBascules(ser.owner[s], e.doc.FrameCount), tol)
		if v.confrontables == 0 {
			continue
		}
		autres++
		if v.taux() > meilleur.taux() {
			meilleur, mSlot = v, s
		}
	}
	if autres == 0 {
		t.Logf("TEMOIN a  %s : aucun autre canal confrontable — temoin sans objet", e.short)
	} else {
		t.Logf("TEMOIN a  %s : permutation, pire des %d autres canaux (slot %d) : %s",
			e.short, autres, mSlot, meilleur)
	}

	// TEMOIN (b) — LE DECALAGE +60 s, hors de la fenetre de tolerance.
	bas := d2bBascules(ser.owner[ownerSlot], e.doc.FrameCount)
	dec := d2bConfronte(d2bDecale(prises, d2bFrames(d2bTemoinDecalageSec, e), e.doc.FrameCount), bas, tol)
	t.Logf("TEMOIN b  %s : decalage +%d s : %s", e.short, d2bTemoinDecalageSec, dec)

	// CONTROLE — le +20 s de la consigne d'origine. DANS la fenetre : il mesure la stabilite,
	// pas l'absence de signal. Publie pour la continuite, jamais lu comme un temoin.
	ctl := d2bConfronte(d2bDecale(prises, d2bFrames(d2bControleDecalageSec, e), e.doc.FrameCount), bas, tol)
	t.Logf("CONTROLE  %s : decalage +%d s (DANS la fenetre, pas un temoin) : %s",
		e.short, d2bControleDecalageSec, ctl)
}

// d2bPrises rend les prises de colline du film, posees sur l'axe de frames, avec le camp de leur
// auteur resolu par le ROSTER FIGE du corpus (aucune base ouverte).
func d2bPrises(t *testing.T, e ctEntree) []d2bPrise {
	t.Helper()
	evs := objectiveevents.Extract(e.short, "KOTH:Arena", p2aSource(t, e.dir),
		objectiveevents.MapRoster(e.film.p2aTeams()))
	var out []d2bPrise
	horsAxe, sansCamp := 0, 0
	for _, ev := range evs {
		if ev.EventType != objectiveevents.EventTypeHillCapture || ev.TimeMS == nil {
			continue
		}
		if ev.TeamID == nil {
			sansCamp++
			continue
		}
		f, ok := p2aFrameOf(e.doc, *ev.TimeMS)
		if !ok {
			horsAxe++
			continue
		}
		out = append(out, d2bPrise{frame: f, team: *ev.TeamID})
	}
	if horsAxe > 0 || sansCamp > 0 {
		t.Logf("%s : prises ecartees — %d hors de l'axe du rejeu, %d sans camp au roster",
			e.short, horsAxe, sansCamp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].frame < out[j].frame })
	return out
}

// d2bBascules rend les changements du canal vers une valeur NOMMEE (le neutre n'est pas une
// prise). La segmentation est celle de la production (`mergeZoneRuns`), jamais une seconde
// ecriture. La PREMIERE emission nommee compte comme une bascule : le canal passe d'inconnu a un
// camp, ce qui est bien une prise de possession.
func d2bBascules(ss []zoneSample, frames int) []d2bBascule {
	out := make([]d2bBascule, 0, len(ss))
	for _, g := range mergeZoneRuns(ss) {
		if g.v == zoneNeutralOwner || g.t >= frames {
			continue
		}
		out = append(out, d2bBascule{frame: g.t, v: g.v})
	}
	return out
}

// d2bConfronte apparie chaque prise a la bascule la PLUS PROCHE dans la tolerance, et rend
// l'accord sous la meilleure bijection valeur <-> camp.
func d2bConfronte(prises []d2bPrise, bascules []d2bBascule, tol int) d2bVerdict {
	v := d2bVerdict{prises: len(prises)}
	paires := map[uint64]map[int]int{}
	for _, p := range prises {
		b, ecart, ok := d2bPlusProche(bascules, p.frame, tol)
		if !ok {
			v.sansBascule++
			continue
		}
		v.confrontables++
		v.ecarts = append(v.ecarts, ecart)
		if paires[b.v] == nil {
			paires[b.v] = map[int]int{}
		}
		paires[b.v][p.team]++
	}
	v.accord, v.bijection = d2MeilleureBijection(paires, []int{0, 1})
	return v
}

// d2bPlusProche rend la bascule la plus proche d'une frame, dans la tolerance.
func d2bPlusProche(bascules []d2bBascule, frame, tol int) (d2bBascule, int, bool) {
	best, bestEcart, found := d2bBascule{}, 0, false
	for _, b := range bascules {
		ecart := b.frame - frame
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > tol {
			continue
		}
		if !found || ecart < bestEcart {
			best, bestEcart, found = b, ecart, true
		}
	}
	return best, bestEcart, found
}

// d2bDecale glisse les prises et rejette celles qui sortent de l'axe.
func d2bDecale(prises []d2bPrise, shift, frames int) []d2bPrise {
	out := make([]d2bPrise, 0, len(prises))
	for _, p := range prises {
		if p.frame+shift >= frames {
			continue
		}
		out = append(out, d2bPrise{frame: p.frame + shift, team: p.team})
	}
	return out
}

// d2bFrames convertit des secondes en frames a la cadence du document.
func d2bFrames(sec int, e ctEntree) int {
	return sec * 1000 / max(e.doc.FrameIntervalMS, 1)
}
