package replay

// colline_propriete_oracle_test.go — LOT C-ter VOLET 1, CT.1.2 : L'ORACLE, LES SEUILS, LA MESURE.
//
// LA QUESTION : un tag de `ti=13` est-il « le drapeau d'activation » de la colline de KOTH ?
//
// L'ORACLE PRINCIPAL est ecrit dans le plan : les INCREMENTS du score AFFICHE (score de mode,
// composant 0 valeur A du statborg, les deux equipes) — en KOTH un point = une colline capturee,
// donc un changement de colline. Le second oracle est la borne des periodes de garde que la
// production publie aujourd'hui (`zone_states_hill.go`, methode « positions »).
//
// LES SEUILS SONT CEUX DU PLAN, ecrits ici avant la mesure et jamais ajustes apres :
//
//	(i)   EXCLUSIF  : apres la premiere activation, exactement UN slot porte la valeur active sur
//	                  >= 95 % des frames ;
//	(ii)  SYNCHRONE : une bascule tombe a +/- 1 s de >= 90 % des increments de score ;
//	(iii) TEMOINS   : slots permutes et bascules decalees de +20 s < moitie du taux reel, ET niveau
//	                  du hasard publie (part du match a +/- 1 s d'un increment, et taux attendu de
//	                  N bascules tirees au hasard).
//
// DEUX LECTURES D'UN TAG, ECRITES AVANT DE REGARDER LES CHIFFRES, parce que le plan a ete redige
// pour un DRAPEAU par slot (une colline = un slot, actif/inactif) et que la donnee peut porter
// l'activation autrement :
//
//	F — DRAPEAU par slot   : la valeur d'un slot est active ou non (booleen : 1 ; enumere : tout
//	                          sauf « absent »). Mode B : un slot est actif si AU MOINS UN joueur
//	                          l'est. Bascule = tout changement de l'etat actif d'un slot.
//	D — DESIGNATEUR        : la valeur d'un slot DESIGNE la colline courante (identifiant, index).
//	                          Bascule = tout changement de valeur (la premiere emission comprise :
//	                          l'etat initial vit dans l'image-cle, que le delta ne re-emet pas).
//	                          Mode B : par joueur, puis les instants sont fusionnes a la frame.
//	                          Exclusivite = un seul slot porteur d'une designation courante.
//
// Les tags 3 (jauge) et 4 (proprietaire) sont des TEMOINS DE COMPARAISON, pas des candidats. Le
// tag 5 n'est pas dans la liste du plan (il etait tenu pour une cle de nommage constante) : il est
// mesure au meme oracle et etiquete comme tel — c'est le journal qui tranche, pas l'instrument.
//
// L'INCREMENT TERMINAL. La capture qui atteint la limite de score CLOT le match : aucune colline
// ne lui succede. Le taux (ii) est publie sur TOUS les increments (la lettre du plan) ET hors
// increment terminal (« oracle des CHANGEMENTS de colline », les mots du plan) — les deux
// chiffres, jamais un seul, et le verdict de la lettre en premier.
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan — D17) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/01e1f945"
//	go test -count=1 -run TestCollineProprieteOracle -v -timeout 30m ./internal/analysis/replay/

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// SEUILS DE CT.1.2 — ecrits avant la mesure (plan, lot C-ter volet 1).
const (
	ctSeuilExclusivite = 0.95
	ctSeuilSynchronie  = 0.90
	// ctFenetreMS : demi-fenetre autour d'un increment.
	ctFenetreMS = 1000
	// ctDecalageMS : le temoin temporel — les memes bascules, 20 s plus tard.
	ctDecalageMS = 20000
	// ctFacteurTemoin : un temoin doit rester sous la MOITIE du taux reel.
	ctFacteurTemoin = 0.5
	// ctTirages : tirages du niveau du hasard (N bascules uniformes sur le match).
	ctTirages = 200
	// ctFusionMS : deux bascules a moins d'une frame l'une de l'autre sont UN instant.
	ctFusionMS = 100
)

// ctIncrement est un increment du score affiche : l'oracle principal.
type ctIncrement struct {
	tMS      int64
	slot     int
	value    int64
	terminal bool
}

// TestCollineProprieteOracle mesure chaque tag de ti=13 contre l'oracle des increments (CT.1.2).
func TestCollineProprieteOracle(t *testing.T) {
	e := ctCharge(t)
	out := ctOutDir(t)
	ctLogEntete(t, e)
	incs := ctIncrements(t, e)
	var sb strings.Builder
	fmt.Fprintf(&sb, "# lot C-ter volet 1 — CT.1.2 — film %s (%s / %s)\n", e.short, e.film.Mode, e.film.Carte)
	sb.WriteString("# increment\tfilm\tequipe\tvaleur\tt_ms\tframe\tterminal\n")
	sb.WriteString("# hasard_1\tfilm\tpart\tt0_ms\tt1_ms\n")
	sb.WriteString("# periode_actuelle\tfilm\tzone\tframe_t0\tframe_t1 · rampe\tfilm\tslot\tframe_t0\tframe_pic\n")
	sb.WriteString("# lecture\tfilm\tmode\ttag\tlecture\trole\tslot_candidat\tbascules\texclusivite\t" +
		"ok_tous\ttot_tous\ttaux_tous\tok_chg\ttot_chg\ttaux_chg\tdecale\tpermute\thasard_1\thasard_n\t" +
		"delai_film_ms\tdelai_rejeu_ms\tkf_premiere_presente_ms\tkf_derniere_absente_ms\tpremier_contact_ms\tverdict\n")
	ctLogOracle(t, &sb, e, incs)
	ctLogOracleSecondaire(t, &sb, e)
	for _, r := range ctLectures2(e) {
		ctMesureLecture(t, &sb, e, incs, r)
	}
	p2aWrite(t, out, e.short+"_ct12_mesure.tsv", sb.String())
}

// ctIncrements rend les increments du score AFFICHE des deux equipes, tries, le dernier marque
// TERMINAL. Le premier point d'une equipe compte s'il est deja > 0 (le score part de zero).
func ctIncrements(t *testing.T, e ctEntree) []ctIncrement {
	t.Helper()
	src := p2aSource(t, e.dir)
	recs := objectiveevents.StatRecords(src)
	series := objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)
	var out []ctIncrement
	for slot, pts := range series {
		if !objectiveevents.IsTeamSlot(slot) {
			continue
		}
		var prev int64
		for _, p := range pts {
			if p.Value > prev {
				out = append(out, ctIncrement{tMS: int64(p.TimeMS), slot: slot, value: p.Value})
			}
			prev = p.Value
		}
		t.Logf("  SCORE equipe (slot %d) : %d points, final %d", slot, len(pts), prev)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	if n := len(out); n > 0 {
		out[n-1].terminal = true
	}
	return out
}

// ctLogOracle publie l'oracle : chaque increment, son equipe, sa valeur, son instant sur les
// deux axes, et le niveau du hasard d'UNE bascule.
func ctLogOracle(t *testing.T, sb *strings.Builder, e ctEntree, incs []ctIncrement) {
	t.Helper()
	t.Logf("")
	t.Logf("=== ORACLE PRINCIPAL — %d increments du score affiche (le dernier est TERMINAL)", len(incs))
	for _, in := range incs {
		f, _ := p2aFrameOf(e.doc, int(in.tMS))
		t.Logf("  increment equipe %d -> %d a %7d ms (frame %5d)%s", in.slot, in.value, in.tMS, f,
			ctTerminal(in.terminal))
		fmt.Fprintf(sb, "increment\t%s\t%d\t%d\t%d\t%d\t%v\n", e.short, in.slot, in.value, in.tMS, f,
			in.terminal)
	}
	t0, t1 := ctBornes(e)
	part := ctPartFenetres(incs, t0, t1)
	t.Logf("  NIVEAU DU HASARD d'une bascule : %.1f %% du match [%d ; %d] est a +/- %d ms d'un"+
		" increment", 100*part, t0, t1, ctFenetreMS)
	fmt.Fprintf(sb, "hasard_1\t%s\t%.4f\t%d\t%d\n", e.short, part, t0, t1)
}

// ctTerminal etiquette l'increment terminal.
func ctTerminal(b bool) string {
	if b {
		return "  <- TERMINAL (la capture qui clot le match)"
	}
	return ""
}

// ctBornes rend les bornes du match sur l'horloge du film : origine du rejeu, derniere frame.
func ctBornes(e ctEntree) (int64, int64) {
	origin := int64(0)
	if e.doc.OriginMs != nil {
		origin = *e.doc.OriginMs
	}
	return origin, origin + int64(e.doc.FrameCount)*int64(e.doc.FrameIntervalMS)
}

// ctPartFenetres rend la part de [t0, t1] couverte par l'union des fenetres +/- ctFenetreMS
// autour des increments — la probabilite qu'une bascule tiree au hasard soit « synchrone ».
func ctPartFenetres(incs []ctIncrement, t0, t1 int64) float64 {
	if t1 <= t0 {
		return 0
	}
	type iv struct{ a, b int64 }
	ivs := make([]iv, 0, len(incs))
	for _, in := range incs {
		ivs = append(ivs, iv{max(in.tMS-ctFenetreMS, t0), min(in.tMS+ctFenetreMS, t1)})
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].a < ivs[j].a })
	var couvert int64
	curA, curB := int64(0), int64(-1)
	for _, v := range ivs {
		if v.b <= v.a {
			continue
		}
		if curB < 0 || v.a > curB {
			couvert += max(curB-curA, 0)
			curA, curB = v.a, v.b
			continue
		}
		curB = max(curB, v.b)
	}
	couvert += max(curB-curA, 0)
	return float64(couvert) / float64(t1-t0)
}

// ctLogOracleSecondaire publie les periodes de garde que la production publie AUJOURD'HUI
// (methode positions) et les rampes brutes de la jauge : le second oracle du plan.
func ctLogOracleSecondaire(t *testing.T, sb *strings.Builder, e ctEntree) {
	t.Helper()
	t.Logf("")
	n := 0
	for _, st := range e.doc.ZoneStates {
		for _, sp := range st.Spans {
			if !sp.Active {
				continue
			}
			n++
			t.Logf("  periode actuelle zone %d : frames [%5d ; %5d] = ms [%7d ; %7d]", st.ZoneRef,
				sp.T0, sp.T1, ctMSDeFrame(e, sp.T0), ctMSDeFrame(e, sp.T1))
			fmt.Fprintf(sb, "periode_actuelle\t%s\t%d\t%d\t%d\n", e.short, st.ZoneRef, sp.T0, sp.T1)
		}
	}
	c := zoneCtx{origin: e.posUS, step: uint64(e.doc.FrameIntervalMS) * 1000, frames: e.doc.FrameCount}
	ramps := zoneRampsOf(zoneSeriesOf(e.sc.Reads, c))
	t.Logf("=== ORACLE SECONDAIRE — %d periodes actives publiees aujourd'hui (positions), %d rampes"+
		" brutes de la jauge", n, len(ramps))
	for _, r := range ramps {
		fmt.Fprintf(sb, "rampe\t%s\t%d\t%d\t%d\n", e.short, r.slot, r.t0, r.tPeak)
	}
}

// ctMSDeFrame rend l'instant (horloge du film) d'une frame.
func ctMSDeFrame(e ctEntree, f int) int64 {
	origin, _ := ctBornes(e)
	return origin + int64(f)*int64(e.doc.FrameIntervalMS)
}

// ctLecture2 nomme une lecture d'un tag : (mode, tag, lecture F ou D).
type ctLecture2 struct {
	modeA bool
	tag   int
	// flag : lecture F (drapeau) ; sinon lecture D (designateur).
	flag bool
	// role : « candidat » (liste du plan), « hors liste » (tag 5), « temoin » (tags 3, 4).
	role string
}

// ctLectures2 rend les lectures a mesurer, dans l'ordre du journal.
func ctLectures2(e ctEntree) []ctLecture2 {
	out := []ctLecture2{
		{modeA: true, tag: filmdec.ManagedPropertyTagBool, flag: true, role: "candidat"},
		{modeA: true, tag: filmdec.ManagedPropertyTagEnum, flag: true, role: "candidat"},
		{modeA: true, tag: filmdec.ManagedPropertyTagEnum, flag: false, role: "candidat"},
		{modeA: true, tag: filmdec.ManagedPropertyTagU32Bis, flag: false, role: "candidat"},
		{modeA: true, tag: filmdec.ManagedPropertyTagStringID, flag: false, role: "hors liste"},
		{modeA: true, tag: filmdec.ManagedPropertyTagU32, flag: false, role: "temoin"},
		{modeA: true, tag: filmdec.ManagedPropertyTagQuant, flag: false, role: "temoin"},
	}
	for tag := filmdec.ManagedPropertyTagQuantJ; tag <= 15; tag++ {
		if tag == filmdec.ManagedPropertyTagBoolJ || tag >= filmdec.ManagedPropertyTagEnumJ {
			out = append(out, ctLecture2{modeA: false, tag: tag, flag: true, role: "candidat"})
		}
		out = append(out, ctLecture2{modeA: false, tag: tag, flag: false, role: "candidat"})
	}
	return out
}

// ctSerieSlot est la serie chainee d'un slot pour un tag, par joueur (mode A : la cle -1).
type ctSerieSlot map[int][]ctLecture

// ctSeriesDuTag rend, par slot, les lectures CHAINEES et VALUEES du tag dans le mode demande.
func ctSeriesDuTag(e ctEntree, modeA bool, tag int) map[uint32]ctSerieSlot {
	out := map[uint32]ctSerieSlot{}
	for _, l := range e.lectures {
		if l.modeA != modeA || l.tag != tag || !l.chained || !l.has {
			continue
		}
		if out[l.slot] == nil {
			out[l.slot] = ctSerieSlot{}
		}
		out[l.slot][l.film] = append(out[l.slot][l.film], l)
	}
	return out
}

// ctActif dit si une valeur est « active » au sens de la lecture F.
func ctActif(tag int, v uint64) bool {
	switch tag {
	case filmdec.ManagedPropertyTagBool, filmdec.ManagedPropertyTagBoolJ:
		return v == 1
	}
	return v != 0 // enumere : 0 = « absent » (-1)
}

// ctBasculesSlot rend les instants de bascule d'un slot, fusionnes a la frame.
//
//	lecture F : changements de l'etat actif du slot (mode B : au moins un joueur actif) ;
//	lecture D : changements de valeur, PAR joueur, premiere emission comprise.
func ctBasculesSlot(ser ctSerieSlot, r ctLecture2) []int64 {
	var raw []int64
	if r.flag {
		raw = ctBasculesDrapeau(ser, r.tag)
	} else {
		for _, ls := range ser {
			for i, l := range ls {
				if i == 0 || l.value != ls[i-1].value {
					raw = append(raw, l.tMS)
				}
			}
		}
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
	var out []int64
	for _, t := range raw {
		if n := len(out); n == 0 || t-out[n-1] > ctFusionMS {
			out = append(out, t)
		}
	}
	return out
}

// ctBasculesDrapeau rejoue l'etat actif d'un slot (au moins un joueur) et rend ses changements.
func ctBasculesDrapeau(ser ctSerieSlot, tag int) []int64 {
	var all []ctLecture
	for _, ls := range ser {
		all = append(all, ls...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].tMS < all[j].tMS })
	etat := map[int]bool{}
	actifs := 0
	prev := false
	var out []int64
	for _, l := range all {
		a := ctActif(tag, l.value)
		if a != etat[l.film] {
			if a {
				actifs++
			} else {
				actifs--
			}
			etat[l.film] = a
		}
		cur := actifs > 0
		if cur != prev {
			out = append(out, l.tMS)
			prev = cur
		}
	}
	return out
}

// ctSynchronie rend le taux d'increments a +/- ctFenetreMS d'une bascule (decalee de `shift`),
// et les ecarts (bascule la plus proche - increment) de ceux qui le sont.
func ctSynchronie(incs []ctIncrement, bascules []int64, shift int64, horsTerminal bool) (float64, int, int, []int64) {
	ok, total := 0, 0
	var ecarts []int64
	for _, in := range incs {
		if horsTerminal && in.terminal {
			continue
		}
		total++
		best, found := int64(0), false
		for _, b := range bascules {
			d := b + shift - in.tMS
			if d < 0 {
				d = -d
			}
			if d <= ctFenetreMS && (!found || d < ctAbs(best)) {
				best, found = b+shift-in.tMS, true
			}
		}
		if found {
			ok++
			ecarts = append(ecarts, best)
		}
	}
	return p2aRate(ok, total), ok, total, ecarts
}

func ctAbs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ctHasardN rend le taux de synchronie attendu de n bascules tirees uniformement sur le match
// (moyenne de ctTirages tirages, graine fixe).
func ctHasardN(incs []ctIncrement, n int, t0, t1 int64, horsTerminal bool) float64 {
	if n == 0 || t1 <= t0 {
		return 0
	}
	rng := rand.New(rand.NewSource(20260819)) //nolint:gosec // temoin de mesure, pas de securite
	sum := 0.0
	for k := 0; k < ctTirages; k++ {
		b := make([]int64, n)
		for i := range b {
			b[i] = t0 + rng.Int63n(t1-t0)
		}
		r, _, _, _ := ctSynchronie(incs, b, 0, horsTerminal)
		sum += r
	}
	return sum / ctTirages
}
