package replay

// oddball_th10_d10_test.go — D10 / O3 : QUE DATENT LES EVENEMENTS `th=10` DU CRANE ?
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER
// (`.ai/V7.5/replay2d/registre_film/D10_PROTOCOLE.md`, §6). Ce qui suit l'applique.
//
// # POURQUOI CETTE QUESTION EST OUVERTE
//
// D6 a ECARTE les `th=10` parce qu'ils n'etaient pas compris, pas parce qu'ils etaient
// refutes (handoff §4-P3). Leur compte suit les tics API dans un rapport 3,1-3,7 ; s'ils
// datent les ramassages ou les lachers, ils sont un ancrage temporel EXACT. Ce fichier
// confronte chaque `th=10` de crane aux transitions de la chaine : les NAISSANCES de vies
// libres (l'objet reapparait — un lacher ou une re-creation) et les SILENCES (l'objet se
// tait — un ramassage). Debuts et fins de trous sont LES MEMES instants, par construction.
//
// # LE TEMOIN EST DANS LA METRIQUE
//
// L'accord se mesure contre DEUX classes concurrentes. Si les deux rendent le meme accord,
// rien n'est date — verdict « NON ETABLI » (protocole §8).
//
// # L'HORLOGE EST CELLE DE D4
//
// matchMS = (US - ScanFilmClockOrigin) / 1000 — l'expression qui a etabli la coincidence a
// 3-6 ms entre les creations du mot elu et les `th=10` (seuil (1), D4). La tolerance est
// `d4EcartEvenementMS` (1000 ms), DEJA COMMITEE en D4 — pas un reglage neuf.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM` + `ODDBALL_ORACLE_TICS` (oracle fige
// `D10_oracle_api_tics.json`), UN FILM PAR PROCESSUS, lecture seule, AUCUNE base.

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// d10TicsEnv designe le fichier d'oracle FIGE des tics de crane : xuid -> tics, releve
	// de `match_objective_stats_latest` (skull_scoring_ticks). Meme contrat que d6Oracle.
	d10TicsEnv = "ODDBALL_ORACLE_TICS"
	// d10AccordSeuil : le seuil du verdict, recopie du plan §O3.2 (accord >= 80 %).
	d10AccordSeuil = 0.80
)

// d10Th10 est UN evenement `th=10` de crane : son instant (horloge du MATCH) et son acteur.
type d10Th10 struct {
	ms   int64
	xuid string
}

// TestOddballTh10D10 — LA MESURE O3. Un film par processus.
func TestOddballTh10D10(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	tics, ok := d10Tics(t, id)
	if !ok {
		return
	}
	g := filmproc.Arm("d10-th10", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	e, ok := d8Charge(t, root, id)
	if !ok {
		return
	}
	evs := d10Evenements(t, root, id)
	if len(evs) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun evenement `th=10` de crane. NI POUR NI CONTRE.", id)
		return
	}
	naissances, silences := d10Transitions(t, root, id, e)
	t.Logf("%s : %d evenement(s) `th=10`, %d naissance(s), %d silence(s) confrontables",
		id, len(evs), len(naissances), len(silences))

	d10Confronte(t, id, evs, naissances, silences)
	d10RapportTics(t, id, evs, tics)
}

// d10Tics lit le fichier d'oracle fige des tics : xuid decimal -> skull_scoring_ticks.
func d10Tics(t *testing.T, id string) (map[string]float64, bool) {
	t.Helper()
	path := os.Getenv(d10TicsEnv)
	if path == "" {
		t.Skipf("mesure non demandee : %s vide (oracle tics fige)", d10TicsEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur de la mesure
	if err != nil {
		t.Fatalf("oracle tics illisible (%s) : %v", path, err)
	}
	var all map[string]map[string]float64
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatalf("oracle tics invalide (%s) : %v", path, err)
	}
	o, ok := all[id]
	if !ok || len(o) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun tic API. NI POUR NI CONTRE.", id)
		return nil, false
	}
	return o, true
}

// d10Evenements rend les `th=10` de crane, dates (horloge du MATCH) et nommes.
func d10Evenements(t *testing.T, root, id string) []d10Th10 {
	t.Helper()
	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache", id)
	}
	var out []d10Th10
	for _, ev := range objectiveevents.Extract(id, d4VariantOddball, src, objectiveevents.MapRoster{}) {
		if ev.EventType != objectiveevents.EventTypeSkullCarry || ev.TimeMS == nil {
			continue
		}
		e := d10Th10{ms: int64(*ev.TimeMS)}
		if len(ev.Players) > 0 {
			e.xuid = ev.Players[0].XUID
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ms < out[j].ms })
	return out
}

// d10Transitions rend les instants (horloge du MATCH, ms) des naissances et des silences
// des vies libres — la conversion est celle de D4 (`ScanFilmClockOrigin`).
func d10Transitions(t *testing.T, root, id string, e d8Etat) (naissances, silences []int64) {
	t.Helper()
	clockUS, err := ScanFilmClockOrigin(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", id, err)
	}
	conv := func(us uint64) (int64, bool) {
		if us < clockUS {
			return 0, false
		}
		return int64(us-clockUS) / 1000, true
	}
	for _, v := range e.vies {
		if ms, ok := conv(v.T0US); ok {
			naissances = append(naissances, ms)
		}
		if ms, ok := conv(v.T1US); ok {
			silences = append(silences, ms)
		}
	}
	return naissances, silences
}

// d10Confronte publie l'histogramme des ecarts et les accords par classe, puis le verdict.
func d10Confronte(t *testing.T, id string, evs []d10Th10, naissances, silences []int64) {
	t.Helper()
	bornes := []int64{100, 250, 500, 1000, 2000, 5000}
	histo := make([]int, len(bornes)+1)
	accN, accS, accU := 0, 0, 0
	for _, ev := range evs {
		gN, gS := d10EcartMin(ev.ms, naissances), d10EcartMin(ev.ms, silences)
		g := gN
		if gS < g {
			g = gS
		}
		i := 0
		for i < len(bornes) && g > bornes[i] {
			i++
		}
		histo[i]++
		if gN <= d4EcartEvenementMS {
			accN++
		}
		if gS <= d4EcartEvenementMS {
			accS++
		}
		if gN <= d4EcartEvenementMS || gS <= d4EcartEvenementMS {
			accU++
		}
	}
	t.Logf("HISTOGRAMME %s (ecart au plus proche des deux classes) : <=100 ms %d · "+
		"<=250 %d · <=500 %d · <=1000 %d · <=2000 %d · <=5000 %d · au-dela %d",
		id, histo[0], histo[1], histo[2], histo[3], histo[4], histo[5], histo[6])
	n := len(evs)
	t.Logf("ACCORD %s (tolerance %d ms) : NAISSANCES %s · SILENCES %s · UNION %s",
		id, d4EcartEvenementMS, d6Part(accN, n), d6Part(accS, n), d6Part(accU, n))
	d10VerdictTh10(t, id, accN, accS, accU, n)
}

// d10VerdictTh10 applique le seuil du plan (80 %) et le temoin des classes concurrentes.
func d10VerdictTh10(t *testing.T, id string, accN, accS, accU, n int) {
	t.Helper()
	seuil := int(float64(n) * d10AccordSeuil)
	okN, okS, okU := accN >= seuil, accS >= seuil, accU >= seuil
	switch {
	case okN && okS:
		t.Logf("VERDICT %s : les DEUX classes depassent %.0f %% — les th=10 ne se laissent "+
			"pas attribuer a l'une plutot qu'a l'autre : NON ETABLI (temoin du protocole §8).",
			id, 100*d10AccordSeuil)
	case okN:
		t.Logf("VERDICT %s : les th=10 datent les NAISSANCES de vies libres (lachers / "+
			"re-creations) — accord %.1f %% (seuil %.0f %%).", id, 100*float64(accN)/float64(n),
			100*d10AccordSeuil)
	case okS:
		t.Logf("VERDICT %s : les th=10 datent les SILENCES (ramassages) — accord %.1f %% "+
			"(seuil %.0f %%).", id, 100*float64(accS)/float64(n), 100*d10AccordSeuil)
	case okU:
		t.Logf("VERDICT %s : l'UNION depasse %.0f %% mais aucune classe seule — les th=10 "+
			"datent des TRANSITIONS sans distinguer lacher et ramassage.", id, 100*d10AccordSeuil)
	default:
		t.Logf("VERDICT %s : NON ETABLI — aucun accord ne depasse %.0f %%.", id, 100*d10AccordSeuil)
	}
}

// d10EcartMin rend l'ecart absolu minimal (ms) d'un instant a une liste d'instants.
func d10EcartMin(ms int64, liste []int64) int64 {
	best := int64(1 << 62)
	for _, v := range liste {
		if d := attAbs(ms - v); d < best {
			best = d
		}
	}
	return best
}

// d10RapportTics publie, par joueur, le compte de `th=10` contre les tics API.
func d10RapportTics(t *testing.T, id string, evs []d10Th10, tics map[string]float64) {
	t.Helper()
	parJoueur := map[string]int{}
	for _, ev := range evs {
		if ev.xuid != "" {
			parJoueur[ev.xuid]++
		}
	}
	xuids := make([]string, 0, len(tics))
	for x := range tics {
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	for _, x := range xuids {
		c := parJoueur[x]
		ratio := "-"
		if c > 0 {
			ratio = strconvRatio(tics[x], c)
		}
		t.Logf("TICS %s : joueur %s — %d th=10, %.0f tic(s) API, rapport tics/th10 %s",
			id, x[len(x)-4:], c, tics[x], ratio)
	}
	for x, c := range parJoueur {
		if _, vu := tics[x]; !vu {
			t.Logf("TICS %s : joueur %s — %d th=10 mais AUCUN tic API (hors oracle)",
				id, x[len(x)-4:], c)
		}
	}
}

// strconvRatio rend un rapport lisible a deux decimales.
func strconvRatio(num float64, den int) string {
	r := num / float64(den)
	return strconvF(r)
}

// strconvF formate un flottant court.
func strconvF(v float64) string {
	const p = 100
	r := float64(int(v*p+0.5)) / p
	b, _ := json.Marshal(r)
	return string(b)
}
