package replay

// colline_seuil_garde_e1_test.go — ETAPE E1 : COMBIEN DE TEMPS FAUT-IL TENIR LA COLLINE POUR
// MARQUER UN POINT.
//
// EN KOTH IL N'Y A PAS DE CAPTURE : la colline se prend instantanement quand aucun adversaire
// n'y est, et c'est la GARDE qui marque. Le rejeu publie deja QUELLE colline est active et QUI
// la tient ; il lui manque le DENOMINATEUR de la progression vers le prochain point. Cet
// instrument le mesure — il ne change rien a la production.
//
// LE PROTOCOLE, SES SEUILS ET SON REPLI SONT ECRITS AVANT CETTE MESURE dans
// `.ai/V7.5/PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` (etape E1). Ce fichier l'applique, il ne le
// decide pas.
//
// CE QUI EST MESURE, EN UNE PHRASE : entre deux points, le temps pendant lequel le camp QUI
// MARQUE tient la colline selon le canal de propriete.
//
// LES PERIODES NE SONT PAS DEVINEES. Le releve du 2026-08-30 (4 artefacts) etablit que la
// colline TOURNE a chaque point : chaque periode publiee commence a l'image du point precedent
// et finit a celle du point suivant. Les bornes sont donc les INCREMENTS du score de mode, pris
// tels quels — pas une segmentation propre a cet instrument.
//
// LA BIJECTION VALEUR <-> CAMP EST GLOBALE AU FILM, ET SA MARGE EST PUBLIEE. Le canal de
// propriete ne porte pas l'index d'equipe du score de mode : les deux espaces sont distincts
// (`zoneStates.owner` suit le ROSTER, `scoreTimeline.teams[]` la convention LOCALE du film — le
// piege est consigne au plan §1.1). On choisit donc, entre les deux appariements possibles,
// celui qui explique le plus de periodes, ET on publie l'ecart entre les deux : une marge nulle
// est une bijection degeneree, et le film est ECARTE plutot que compte a son avantage.
//
// REGIME : garde `ZONE_FILM`, un film par processus, lecture seule, AUCUNE base ouverte,
// AUCUN changement de production.
//
//	$env:ZONE_FILM="<cache>/film_chunks/01e1f945"; go test ./internal/analysis/replay/ -run CollineSeuilGardeE1 -v

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

const (
	// e1MinRunSeconds : le plancher de duree d'un intervalle de propriete, repris TEL QUEL de
	// D2 — un intervalle plus court qu'une seconde ne peut pas faire monter un compteur de
	// secondes, et le compter melangerait un silence de mesure a une garde.
	e1MinRunSeconds = 1
	// e1ControleDecalageSec : le temoin (b) TEL QUE LE PLAN L'A ECRIT — les memes periodes,
	// glissees de 20 s. Il est mesure et publie, mais lu comme un CONTROLE, pas comme un temoin.
	//
	// POURQUOI IL NE PEUT PAS ECHOUER, et pourquoi ce n'est pas une decouverte : glisser une
	// fenetre de MEME LONGUEUR a l'interieur d'une periode ou la propriete est stationnaire
	// laisse la somme presque inchangee. D2-bis avait deja rencontre exactement ce defaut sur le
	// meme +20 s et l'avait consigne — « un temoin qui ne peut pas echouer n'est pas un temoin »
	// (`colline_proprietaire_d2bis_test.go`, d2bTemoinDecalageSec). Le plan E1 a herite du
	// chiffre sans heriter de la correction.
	e1ControleDecalageSec = 20
	// e1TemoinDecalageSec : LE VRAI temoin de decalage, trois fois le controle — meme facteur et
	// meme raison que D2-bis. A 60 s le decalage sort de la periode courante sur la moitie du
	// corpus (duree mediane 88,9 s) : le temoin PEUT echouer, donc il en est un. Ce n'est pas un
	// gate abaisse, c'est un gate DURCI : le +20 s reste publie a cote.
	e1TemoinDecalageSec = 60
)

// e1Cartes — LA CARTE DE CHAQUE FILM KOTH EXPLOITABLE, gelee ici.
//
// POURQUOI UN RELEVE FIGE ET PAS UNE REQUETE : le paquet `replay` n'ouvre AUCUNE DuckDB (meme
// convention que `p2aCorpus`). Releve du 2026-08-30 sur `match_registry` — les 45 films des
// variantes `KOTH:Arena` et `Ranked:King of the Hill` qui ont A LA FOIS leurs chunks en cache
// ET des bornes de quantification au catalogue.
//
// LES TROIS MATCHS CLASSES N'Y SONT PAS, et c'est une limite de la mesure, pas un oubli :
// `bf856f3a` et `dbbaaccc` n'ont pas de film en cache, `0a247154` joue sur « Solitude - Ranked »
// qui est absente de `map_quant_bounds.json`. Le seuil sera donc mesure sur le KOTH SOCIAL
// seulement.
var e1Cartes = map[string]string{
	"01e1f945": "catalyst",
	"0c009149": "snowbound",
	"1b370ce1": "curfew",
	"21ece4d8": "live fire",
	"240759c5": "absolution",
	"257d1457": "ecotone",
	"262e1c57": "salvation",
	"381230b2": "forbidden",
	"3b1de706": "curfew",
	"3b941a2a": "nemesis",
	"4a49d994": "elevation",
	"4a4a872d": "ecotone",
	"4ea58db8": "forbidden",
	"5156838d": "empyrean",
	"5d4295b8": "prism",
	"606d9844": "chasm",
	"61a0b5d2": "goliath",
	"61a47400": "empyrean",
	"63a45b8b": "dredge",
	"666c7cbc": "bazaar",
	"6bd16f21": "absolution",
	"6cdec7c3": "absolution",
	"71ad4abd": "vagabond",
	"75f1188f": "catalyst",
	"7665e832": "opulence",
	"7ae6a0da": "vagabond",
	"7c316469": "empyrean",
	"7f1bbf06": "streets",
	"8076f97f": "shogun",
	"8277ff2e": "goliath",
	"8f9b1c5a": "goliath",
	"a36c8bed": "isolation",
	"a92bab93": "illusion",
	"ac919b39": "absolution",
	"b862bfc7": "curfew",
	"b9db24b5": "fortress",
	"cae8471d": "empyrean",
	"d2b74083": "illusion",
	"d55f2475": "snowbound",
	"d62778f5": "live fire",
	"da2fd554": "shogun",
	"eeaf049b": "forbidden",
	"f6091638": "banished narrows",
	"fa5d118b": "behemoth",
	"fce2a031": "curfew",
}

// e1Periode est une periode de colline : l'intervalle de frames entre deux points, et le slot
// d'equipe qui a marque celui qui la CLOT.
type e1Periode struct {
	t0, t1 int
	scorer int
}

// TestCollineSeuilGardeE1 — LA MESURE. Un film par processus.
func TestCollineSeuilGardeE1(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	carte, ok := e1Cartes[short]
	if !ok {
		t.Skipf("film %s hors corpus E1 (pas un KOTH exploitable au releve du 2026-08-30)", short)
	}

	// Le document ne sert qu'a DEUX choses ici : l'axe de frames et l'origine d'horloge. Aucune
	// zone n'est fournie — la mesure porte sur le canal, pas sur l'appariement a une forme.
	doc, posUS := p2bBuild(t, dir, short, p2aQuant(t, carte), ZoneInput{Scanned: true}, nil)
	sc := p2bScan(t, dir)
	c := zoneCtx{origin: posUS, step: uint64(doc.FrameIntervalMS) * 1000,
		frames: doc.FrameCount, intervalMS: doc.FrameIntervalMS}
	ser := zoneSeriesOf(sc.Reads, c)

	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Logf("ECARTE   %s (%s) : aucun designateur elu — ce film retombe sur la methode par "+
			"rampes, qui ne publie aucun proprietaire", short, carte)
		return
	}
	ownerSlot := d.slot + 1

	recs := objectiveevents.StatRecords(p2aBobine(t, dir))
	score := objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)
	slots := d2ScoreSlots(score)
	if len(slots) != 2 {
		t.Logf("ECARTE   %s (%s) : %d slot(s) d'equipe au score de mode — sans les deux camps, "+
			"une periode ne peut pas nommer son marqueur", short, carte, len(slots))
		return
	}

	periodes := e1Periodes(doc, score, slots, d.first)
	if len(periodes) == 0 {
		t.Logf("ECARTE   %s (%s) : aucun increment de score datable sur l'axe des frames",
			short, carte)
		return
	}

	minFrames := e1MinRunSeconds * 1000 / max(doc.FrameIntervalMS, 1)
	runs := d2Runs(ser.owner[ownerSlot], doc.FrameCount, minFrames)
	vals := e1Valeurs(runs)
	if len(vals) != 2 {
		t.Logf("ECARTE   %s (%s) : le canal de propriete ne porte que %d valeur(s) nommee(s) — "+
			"aucune bijection a choisir", short, carte, len(vals))
		return
	}

	direct, croise := e1Accords(periodes, runs, vals, slots)
	if direct == croise {
		t.Logf("ECARTE   %s (%s) : bijection DEGENEREE (%d periodes expliquees dans les deux "+
			"sens) — le film ne dit pas quelle valeur est quel camp", short, carte, direct)
		return
	}
	// La bijection retenue : celle qui explique le plus de periodes.
	camp := map[int]uint64{slots[0]: vals[0], slots[1]: vals[1]}
	sens := "direct"
	if croise > direct {
		camp = map[int]uint64{slots[0]: vals[1], slots[1]: vals[0]}
		sens = "croise"
	}
	t.Logf("%s (%s) : designateur slot %d, propriete slot %d (%d emissions), %d periode(s), "+
		"bijection %s (%d contre %d), axe %d frames a %d ms",
		short, carte, d.slot, ownerSlot, len(ser.owner[ownerSlot]), len(periodes), sens,
		max(direct, croise), min(direct, croise), doc.FrameCount, doc.FrameIntervalMS)

	sec := float64(doc.FrameIntervalMS) / 1000
	shift := e1TemoinDecalageSec * 1000 / max(doc.FrameIntervalMS, 1)
	ctrl := e1ControleDecalageSec * 1000 / max(doc.FrameIntervalMS, 1)
	for i, p := range periodes {
		tenue := e1Tenue(runs, p.t0, p.t1)
		marqueur := camp[p.scorer]
		autre := vals[0]
		if marqueur == vals[0] {
			autre = vals[1]
		}
		// SIGNAL : le temps de garde du camp qui marque.
		e1Ligne(t, short, carte, "signal", i, float64(tenue[marqueur])*sec, float64(p.t1-p.t0+1)*sec)
		// TEMOIN (a) — LA PERMUTATION : le camp d'en face sur la meme periode. Un seuil qui
		// sortirait aussi net de ce cote-la ne mesurerait pas la garde du marqueur.
		e1Ligne(t, short, carte, "temoin_a", i, float64(tenue[autre])*sec, float64(p.t1-p.t0+1)*sec)
		// TEMOIN (b) — LE DECALAGE : la meme periode glissee de 60 s (cf. e1TemoinDecalageSec).
		if p.t1+shift < doc.FrameCount {
			dec := e1Tenue(runs, p.t0+shift, p.t1+shift)
			e1Ligne(t, short, carte, "temoin_b", i, float64(dec[marqueur])*sec, float64(p.t1-p.t0+1)*sec)
		}
		// CONTROLE — le +20 s du plan, publie pour la continuite et lu comme tel.
		if p.t1+ctrl < doc.FrameCount {
			dec := e1Tenue(runs, p.t0+ctrl, p.t1+ctrl)
			e1Ligne(t, short, carte, "controle_20s", i, float64(dec[marqueur])*sec, float64(p.t1-p.t0+1)*sec)
		}
	}
}

// e1Ligne ecrit une observation sous une forme lisible par la passe d'agregation. Le format est
// stable : c'est le contrat entre l'instrument et le rapport.
func e1Ligne(t *testing.T, short, carte, source string, i int, tenue, duree float64) {
	t.Helper()
	t.Logf("E1DATA\t%s\t%s\t%s\t%d\t%.1f\t%.1f", short, carte, source, i, tenue, duree)
}

// e1Periodes decoupe le match en periodes de colline : une par INCREMENT du score de mode.
//
// UN INCREMENT, PAS UNE EMISSION. La serie est cumulative et re-emet ses paliers ; seule une
// VALEUR PLUS HAUTE que la precedente du meme slot est un point marque. La premiere emission
// d'un slot compte comme increment si elle est deja positive — un compteur qui apparait a 1 a
// marque une fois.
//
// UNE PERIODE QUE DEUX CAMPS CLOTURENT A LA MEME FRAME EST ECARTEE : elle ne nomme pas son
// marqueur, et lui en attribuer un serait une devinette.
func e1Periodes(doc ReplayDocument, score map[int][]objectiveevents.ScorePoint, slots []int,
	debut int,
) []e1Periode {
	type inc struct {
		frame, slot int
	}
	var incs []inc
	for _, s := range slots {
		var prev int64
		for _, p := range score[s] {
			if p.Value <= prev {
				continue
			}
			prev = p.Value
			if f, ok := p2aFrameOf(doc, p.TimeMS); ok {
				incs = append(incs, inc{frame: f, slot: s})
			}
		}
	}
	sort.Slice(incs, func(i, j int) bool { return incs[i].frame < incs[j].frame })

	out := make([]e1Periode, 0, len(incs))
	prev := debut
	for i := 0; i < len(incs); i++ {
		if i+1 < len(incs) && incs[i+1].frame == incs[i].frame {
			// Deux camps marquent a la meme frame : ni l'une ni l'autre ne se lit.
			prev = incs[i].frame + 1
			i++
			continue
		}
		if incs[i].frame > prev {
			out = append(out, e1Periode{t0: prev, t1: incs[i].frame, scorer: incs[i].slot})
		}
		prev = incs[i].frame + 1
	}
	return out
}

// e1Valeurs rend les valeurs nommees du canal de propriete, triees — deterministe d'un film a
// l'autre, comme `d2MeilleureBijection`.
func e1Valeurs(runs []d2Run) []uint64 {
	seen := map[uint64]bool{}
	for _, r := range runs {
		seen[r.v] = true
	}
	out := make([]uint64, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// e1Tenue rend, par valeur du canal, le nombre de frames tenues dans [t0, t1].
func e1Tenue(runs []d2Run, t0, t1 int) map[uint64]int {
	out := map[uint64]int{}
	for _, r := range runs {
		a, b := max(r.t0, t0), min(r.t1, t1)
		if b < a {
			continue
		}
		out[r.v] += b - a + 1
	}
	return out
}

// e1Accords compte, pour chacune des deux bijections possibles, le nombre de periodes ou le camp
// qui marque est AUSSI le teneur dominant. C'est ce compte qui choisit la bijection, et l'ecart
// entre les deux qui dit si le choix a un sens.
func e1Accords(periodes []e1Periode, runs []d2Run, vals []uint64, slots []int) (int, int) {
	direct, croise := 0, 0
	for _, p := range periodes {
		tenue := e1Tenue(runs, p.t0, p.t1)
		a, b := tenue[vals[0]], tenue[vals[1]]
		if a == b {
			continue
		}
		dominant := vals[0]
		if b > a {
			dominant = vals[1]
		}
		if (p.scorer == slots[0] && dominant == vals[0]) || (p.scorer == slots[1] && dominant == vals[1]) {
			direct++
		} else {
			croise++
		}
	}
	return direct, croise
}
