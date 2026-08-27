package replay

// assaut_a3_ti13_test.go — LOT A, PHASE A3.1 : DIAGNOSTIC DU CANAL `ti=13` SUR ASSAUT.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`registre_film/A_PROTOCOLE.md`, §4).
// Le gate A3.2 ne peut pas etre TENU sur ce corpus (temoin spatial inmesurable : aucune
// forme de site `assault_bomb` au catalogue — §1 du protocole) : ce fichier est le
// DIAGNOSTIC que le plan ordonne quand meme, et il n'elit rien.
//
// TROIS LECTURES, TOUTES HERITEES TELLES QUELLES :
//
//	le BALAYAGE     `p2aScanFilm` (lot C-bis phase 2a) — grammaire recopiee CONTROLEE par
//	                ses deux gardes (registre + chainage), emissions datees sur l'horloge
//	                du MANIFESTE, la meme que les enregistrements d'entite ;
//	les RAMPES      `findZoneRamps` (production, zone_states.go) — >= 3 echantillons
//	                croissants, amplitude >= 4 096 quanta : la definition de la jauge de
//	                Bastion, INTACTE (le protocole §4 l'exige) ;
//	les EXPLOSIONS  `a1ClassesTemporelles` (phase A1) — chaque montee du score de mode
//	                d'un slot d'equipe, corroborees par le score API 9/9 (releve A0.3).
//
// LA QUESTION POSEE : le canal `ti=13` porte-t-il, sur Assaut, une structure de rampe
// (l'armement d'un site, patron de la capture de Bastion) dont la FIN precede l'explosion
// d'un delai stable (la meche) ? Le diagnostic publie les denominateurs d'ancrage, un
// inventaire des tags par slot, les rampes, et les deltas fin-de-rampe -> explosion.
//
// REGIME : gardes `ATT_FILM` + `ASSAUT_FILM`, UN FILM PAR PROCESSUS, lecture seule,
// AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ASSAUT_FILM="35b75a31"
//	go test ./internal/analysis/replay/ -run AssautA3Ti13 -v

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// TestAssautA3Ti13 — le diagnostic A3.1 sur UN film.
func TestAssautA3Ti13(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(a0FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Assaut)", a0FilmEnv)
	}
	g := filmproc.Arm("a3-assaut", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — diagnostic interrompu, ce film sort "+
			"avec sa raison", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache (%s=%q)", id, attFilmEnv, root)
	}
	dir := objChunkDir(root, id)
	p2aCheckRegistre(t, dir)
	sc := p2aScanFilm(t, dir, p2aStartMS(src))
	t.Logf("%s : bande ti=13 — %d slot(s) · %d records ancres, %d rejoues, %d CHAINES (%.1f %%) · "+
		"scalaires %d/%d = %.1f %% · par joueur %d/%d = %.1f %% · temoin decale de 3 bits "+
		"%d/%d = %.1f %% · film %d ms",
		id, sc.bandeSlots, sc.records, sc.walked, sc.chained, 100*p2aRate(sc.chained, sc.walked),
		sc.chainedScal, sc.walkedScal, 100*p2aRate(sc.chainedScal, sc.walkedScal),
		sc.chainedJoue, sc.walkedJoue, 100*p2aRate(sc.chainedJoue, sc.walkedJoue),
		sc.decale, sc.walked, 100*p2aRate(sc.decale, sc.walked), sc.t1MS-sc.t0MS)

	gauge := a3Inventaire(t, id, sc)
	rampes := a3Rampes(t, id, gauge)
	debuts, explosions := a1ClassesTemporelles(t, id, src)
	t.Logf("%s : %d debut(s) de manche brute, %d explosion(s) datee(s)", id, len(debuts),
		len(explosions))
	a3Deltas(t, id, rampes, explosions)
}

// a3Inventaire publie les emissions par (slot, tag) du canal scalaire i1 et rend les series
// de jauge (tag 3, quant 24 bits) par slot, triees par temps.
func a3Inventaire(t *testing.T, id string, sc *p2aScan) map[uint32][]zoneSample {
	t.Helper()
	parSlot := map[uint32]map[int]int{}
	gauge := map[uint32][]zoneSample{}
	for _, e := range sc.scal {
		if parSlot[e.slot] == nil {
			parSlot[e.slot] = map[int]int{}
		}
		parSlot[e.slot][e.tag]++
		if e.tag == p2aTagQuant && e.hasPay {
			gauge[e.slot] = append(gauge[e.slot], zoneSample{t: e.tMS, v: e.pay})
		}
	}
	slots := make([]uint32, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	publies := 0
	for _, s := range slots {
		total := 0
		for _, n := range parSlot[s] {
			total += n
		}
		if total < p2aMinParSlot {
			continue
		}
		publies++
		tags := make([]int, 0, len(parSlot[s]))
		for tag := range parSlot[s] {
			tags = append(tags, tag)
		}
		sort.Ints(tags)
		ligne := ""
		for _, tag := range tags {
			ligne += fmtTag(tag, parSlot[s][tag])
		}
		t.Logf("%s : slot %d — %d emission(s) i1 :%s", id, s, total, ligne)
	}
	t.Logf("%s : inventaire i1 — %d slot(s) au-dessus de %d emissions (sur %d emetteurs), "+
		"%d slot(s) a jauge (tag %d)", id, publies, p2aMinParSlot, len(parSlot), len(gauge),
		p2aTagQuant)
	for s := range gauge {
		sort.SliceStable(gauge[s], func(i, j int) bool { return gauge[s][i].t < gauge[s][j].t })
	}
	return gauge
}

// fmtTag rend " tag3=12" — compact, pour une ligne d'inventaire.
func fmtTag(tag, n int) string { return fmt.Sprintf(" tag%d=%d", tag, n) }

// a3Rampes publie les rampes de chaque slot a jauge (definition de production, intacte).
func a3Rampes(t *testing.T, id string, gauge map[uint32][]zoneSample) []zoneRamp {
	t.Helper()
	slots := make([]uint32, 0, len(gauge))
	for s := range gauge {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []zoneRamp
	for _, s := range slots {
		ramps := findZoneRamps(s, gauge[s])
		for _, r := range ramps {
			t.Logf("%s : RAMPE slot %d — [%d, %d] ms, %d -> %d quanta", id, r.slot, r.t0,
				r.tPeak, r.start, r.top)
		}
		out = append(out, ramps...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tPeak < out[j].tPeak })
	t.Logf("%s : %d rampe(s) de jauge au total", id, len(out))
	return out
}

// a3Deltas confronte chaque explosion datee aux fins de rampe : le delta a la fin de rampe
// la plus proche AVANT elle (armement ~ explosion - meche), et le delta absolu le plus
// proche toutes directions — publies tels quels, c'est un diagnostic.
func a3Deltas(t *testing.T, id string, rampes []zoneRamp, explosions []int64) {
	t.Helper()
	if len(explosions) == 0 {
		t.Logf("%s : aucune explosion datee — pas de confrontation possible sur ce film", id)
		return
	}
	if len(rampes) == 0 {
		t.Logf("%s : AUCUNE rampe de jauge — le canal ti=13 ne porte pas de structure "+
			"d'armement lisible sur ce film (0 confrontation)", id)
		return
	}
	for _, x := range explosions {
		avant := int64(math.MaxInt64)
		absolu := int64(math.MaxInt64)
		var slotAvant, slotAbs uint32
		for _, r := range rampes {
			d := x - int64(r.tPeak)
			if d >= 0 && d < avant {
				avant, slotAvant = d, r.slot
			}
			if a := attAbs(d); a < absolu {
				absolu, slotAbs = a, r.slot
			}
		}
		av := "aucune rampe avant"
		if avant < math.MaxInt64 {
			av = fmt.Sprintf("%d ms (slot %d)", avant, slotAvant)
		}
		t.Logf("%s : EXPLOSION t=%d ms — fin de rampe precedente a %s ; plus proche toutes "+
			"directions %d ms (slot %d)", id, x, av, absolu, slotAbs)
	}
}
