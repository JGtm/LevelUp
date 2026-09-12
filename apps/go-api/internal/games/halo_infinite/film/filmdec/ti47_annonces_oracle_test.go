package filmdec

// ti47_annonces_oracle_test.go — LA CONFRONTATION AUX ORACLES HORS FILM (phase 1 du plan
// `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`, items 1.2 a 1.4).
//
// L'ORACLE EST L'ARTEFACT DE REJEU, PAS UNE RECONSTRUCTION. `zoneStates` (schema 15+) porte
// l'etat de chaque zone en intervalles de propriete, sa jauge de capture en direct (schema 18),
// et — en KOTH — l'intervalle ou la zone est LA colline active. Ces trois choses sont exactement
// les evenements que le plan demande de confronter, et elles sont deja datees et deja validees
// par leurs propres lots. Les refabriquer ici serait un second oracle qui divergerait du premier.
//
// L'HORLOGE EST CELLE DU FILM DES DEUX COTES. Les emissions sont datees par le manifeste
// (`probeHorloge` : debut du chunk + ecart au premier paquet delta) ; l'artefact date ses frames
// depuis le PREMIER PAQUET DE POSITION, d'ou le champ `originMs` qu'il publie pour justement
// permettre le recalage : instantMS = originMs + frame x frameIntervalMs (meme formule que
// `scoreClock.frameOf`, en sens inverse). Sans `originMs`, le decalage va de 3,6 s a 50,8 s selon
// le match — l'artefact le dit, on ne le devine pas.
//
// CE QU'ON CONFRONTE N'EST PAS L'INSTANT D'EMISSION. Ce canal replique a 20 Hz (mesure du lot) :
// ses instants couvrent tout le match, donc leur « distance a l'evenement le plus proche » ne
// mesurerait que la densite des evenements. Ce qui fait evenement sur une piste continue, c'est
// le SAUT — une variation hors norme de la valeur. Le seuil de saut est le percentile
// [ti47SeuilSautPct] des variations DU SLOT, donc mesure sur le canal lui-meme et non ecrit
// d'avance.
//
// LE TEMOIN EST « LA MEME LOI, DECALEE ». Les sauts sont rejoues avec un decalage d'un tiers de
// match (modulo la duree) : meme nombre, meme distribution d'espacement, alignement detruit.
// C'est le temoin que le plan demande (« instants tires de la meme loi hors evenements »).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ti47FenetreMS : demi-fenetre autour d'un evenement d'oracle, ECRITE AVANT LA MESURE (meme
// valeur que la sonde F2 du lot F, pour que les deux se comparent).
const ti47FenetreMS = 2000

// ti47SeuilSautPct : percentile des variations par slot au-dela duquel une variation compte
// comme un SAUT.
const ti47SeuilSautPct = 99

// ti47Artefact est la part de l'artefact de rejeu que cet instrument lit. Volontairement
// minimale : tout champ de plus serait un couplage a un schema que ce lot ne fait pas evoluer.
type ti47Artefact struct {
	SchemaVersion   int    `json:"schemaVersion"`
	FrameCount      int    `json:"frameCount"`
	FrameIntervalMs int    `json:"frameIntervalMs"`
	OriginMs        *int64 `json:"originMs"`
	ZoneStates      []struct {
		ZoneRef int `json:"zoneRef"`
		Spans   []struct {
			T0     int  `json:"t0"`
			T1     int  `json:"t1"`
			Owner  *int `json:"owner"`
			Active bool `json:"active"`
		} `json:"spans"`
		Gauge []struct {
			T int     `json:"t"`
			V float32 `json:"v"`
		} `json:"gauge"`
	} `json:"zoneStates"`
}

// ti47Oracle est UN evenement de reference, date sur l'horloge du film.
type ti47Oracle struct {
	tMS  int
	kind string
	zone int
}

// ti47LitArtefact charge l'artefact de rejeu du film, ou dit pourquoi il ne le peut pas.
func ti47LitArtefact(t *testing.T, court string) *ti47Artefact {
	t.Helper()
	path := os.Getenv(ti47ReplayEnv)
	if path == "" {
		root := os.Getenv(ti47CacheEnv)
		if root == "" {
			t.Logf("ORACLE : ni %s ni %s — aucune confrontation possible.",
				ti47ReplayEnv, ti47CacheEnv)
			return nil
		}
		path = filepath.Join(root, "replays", "halo_infinite", court+".json")
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par la garde d'environnement
	if err != nil {
		t.Logf("ORACLE : artefact illisible (%v) — aucune confrontation.", err)
		return nil
	}
	var a ti47Artefact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Logf("ORACLE : artefact non analysable (%v) — aucune confrontation.", err)
		return nil
	}
	if a.OriginMs == nil || a.FrameIntervalMs <= 0 {
		t.Logf("ORACLE : artefact sans origine d'horloge (originMs=%v, pas=%d) — les deux"+
			" horloges ne sont pas recalables, aucune confrontation.", a.OriginMs, a.FrameIntervalMs)
		return nil
	}
	t.Logf("ORACLE : artefact schema %d · %d frames de %d ms · originMs %d · %d zones",
		a.SchemaVersion, a.FrameCount, a.FrameIntervalMs, *a.OriginMs, len(a.ZoneStates))
	return &a
}

// ti47Evenements derive les evenements de reference des etats de zone.
func ti47Evenements(a *ti47Artefact) []ti47Oracle {
	var out []ti47Oracle
	ms := func(frame int) int { return int(*a.OriginMs) + frame*a.FrameIntervalMs }
	for _, z := range a.ZoneStates {
		var prec *int
		for k, s := range z.Spans {
			switch {
			case s.Owner != nil && (k == 0 || prec == nil || *prec != *s.Owner):
				out = append(out, ti47Oracle{ms(s.T0), "prise", z.ZoneRef})
			case s.Owner == nil && prec != nil:
				out = append(out, ti47Oracle{ms(s.T0), "perte", z.ZoneRef})
			}
			if s.Active {
				out = append(out, ti47Oracle{ms(s.T0), "colline_debut", z.ZoneRef})
				out = append(out, ti47Oracle{ms(s.T1), "colline_fin", z.ZoneRef})
			}
			prec = s.Owner
		}
		gauge := make([][2]int, 0, len(z.Gauge))
		for _, g := range z.Gauge {
			gauge = append(gauge, [2]int{g.T, int(g.V * 1000)})
		}
		out = append(out, ti47RampesJauge(gauge, z.ZoneRef, ms)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// ti47RampesJauge decoupe la serie de jauge en RAMPES (montees contigues) et rend leurs bornes.
// Une rampe est la capture EN COURS : c'est le seul oracle du corpus qui approche la
// « contestation » que le plan cherche. La serie est allegee a la construction (un point par
// variation >= 0,02 ou par seconde) : une coupure se lit donc a un trou de plus de 20 frames ou
// a une redescente.
func ti47RampesJauge(gauge [][2]int, ref int, ms func(int) int) []ti47Oracle {
	var out []ti47Oracle
	if len(gauge) < 2 {
		return nil
	}
	debut := 0
	for k := 1; k < len(gauge); k++ {
		if gauge[k][0]-gauge[k-1][0] > 20 || gauge[k][1] < gauge[k-1][1] {
			out = append(out, ti47Oracle{ms(gauge[debut][0]), "jauge_debut", ref},
				ti47Oracle{ms(gauge[k-1][0]), "jauge_fin", ref})
			debut = k
		}
	}
	if debut < len(gauge)-1 {
		out = append(out, ti47Oracle{ms(gauge[debut][0]), "jauge_debut", ref},
			ti47Oracle{ms(gauge[len(gauge)-1][0]), "jauge_fin", ref})
	}
	return out
}

// ti47Saut est UNE variation hors norme : son instant et son amplitude, qui ne se separent pas.
type ti47Saut struct {
	tMS       int
	amplitude int64
}

// ti47Sauts rend les variations d'un slot au-dela de son percentile [ti47SeuilSautPct], triees
// par instant, et le seuil retenu par slot.
func ti47Sauts(m *ti47Moisson) (sauts []ti47Saut, seuils map[uint32]int64) {
	parSlot := map[uint32][]ti47Emission{}
	for _, e := range m.emissions {
		if e.tMS >= 0 {
			parSlot[e.slot] = append(parSlot[e.slot], e)
		}
	}
	seuils = map[uint32]int64{}
	for s, es := range parSlot {
		sort.SliceStable(es, func(i, j int) bool { return es[i].tMS < es[j].tMS })
		d := make([]int64, 0, len(es))
		for k := 1; k < len(es); k++ {
			d = append(d, absI64(int64(es[k].val)-int64(es[k-1].val)))
		}
		if len(d) < 20 {
			continue
		}
		tri := append([]int64(nil), d...)
		sort.Slice(tri, func(i, j int) bool { return tri[i] < tri[j] })
		seuil := tri[minI(len(tri)-1, ti47SeuilSautPct*len(tri)/100)]
		seuils[s] = seuil
		for k := 1; k < len(es); k++ {
			if a := absI64(int64(es[k].val) - int64(es[k-1].val)); a >= seuil {
				sauts = append(sauts, ti47Saut{es[k].tMS, a})
			}
		}
	}
	sort.Slice(sauts, func(i, j int) bool { return sauts[i].tMS < sauts[j].tMS })
	return sauts, seuils
}

// ti47Instants extrait les instants d'une suite de sauts (deja triee).
func ti47Instants(sauts []ti47Saut) []int {
	out := make([]int, 0, len(sauts))
	for _, s := range sauts {
		out = append(out, s.tMS)
	}
	return out
}

// ti47Distances rend la distance de chaque instant a l'oracle le plus proche, triee.
func ti47Distances(instants []int, oracles []int) []int {
	if len(oracles) == 0 {
		return nil
	}
	out := make([]int, 0, len(instants))
	for _, t := range instants {
		i := sort.SearchInts(oracles, t)
		best := 1 << 30
		if i < len(oracles) {
			best = oracles[i] - t
		}
		if i > 0 && t-oracles[i-1] < best {
			best = t - oracles[i-1]
		}
		out = append(out, best)
	}
	sort.Ints(out)
	return out
}

// ti47Decale rejoue une suite d'instants avec un decalage d'un tiers de match : meme loi,
// alignement detruit.
func ti47Decale(instants []int) []int {
	if len(instants) < 2 {
		return nil
	}
	lo, hi := instants[0], instants[len(instants)-1]
	duree := hi - lo
	if duree <= 0 {
		return nil
	}
	out := make([]int, 0, len(instants))
	for _, t := range instants {
		out = append(out, lo+(t-lo+duree/3)%duree)
	}
	sort.Ints(out)
	return out
}

// ti47Confronte publie l'alignement d'une suite d'instants aux oracles, contre son temoin.
func ti47Confronte(t *testing.T, nom string, instants, oracles []int) {
	t.Helper()
	if len(instants) == 0 || len(oracles) == 0 {
		t.Logf("  %-28s AUCUNE mesure (%d instants, %d oracles)", nom, len(instants), len(oracles))
		return
	}
	d := ti47Distances(instants, oracles)
	td := ti47Distances(ti47Decale(instants), oracles)
	dedans := 0
	for _, v := range d {
		if v <= ti47FenetreMS {
			dedans++
		}
	}
	tdedans := 0
	for _, v := range td {
		if v <= ti47FenetreMS {
			tdedans++
		}
	}
	t.Logf("  %-28s %6d instants · distance mediane %6d ms (p90 %7d) · dans +/-%d s :"+
		" %.1f %%", nom, len(instants), ti47Mediane(d), ti47Percentile(d, 90),
		ti47FenetreMS/1000, 100*float64(dedans)/float64(len(d)))
	t.Logf("  %-28s TEMOIN decale             mediane %6d ms (p90 %7d) · dans la fenetre :"+
		" %.1f %%  ->  exces %s", "", ti47Mediane(td), ti47Percentile(td, 90),
		100*float64(tdedans)/float64(maxI(1, len(td))), probeRapport(dedans, tdedans))
}

// ti47OracleRapport execute les items 1.2 a 1.4 sur un film.
func ti47OracleRapport(t *testing.T, court string, m *ti47Moisson) {
	t.Helper()
	a := ti47LitArtefact(t, court)
	if a == nil {
		return
	}
	evs := ti47Evenements(a)
	parType := map[string][]int{}
	var tous []int
	for _, e := range evs {
		parType[e.kind] = append(parType[e.kind], e.tMS)
		tous = append(tous, e.tMS)
	}
	types := make([]string, 0, len(parType))
	for k := range parType {
		types = append(types, k)
	}
	sort.Strings(types)
	var inv []string
	for _, k := range types {
		inv = append(inv, fmt.Sprintf("%s=%d", k, len(parType[k])))
	}
	t.Logf("ORACLES derives de zoneStates : %d evenements · %s", len(evs), strings.Join(inv, " · "))
	if len(evs) == 0 {
		t.Logf("  -> aucun evenement de zone sur ce film : rien a confronter.")
		return
	}

	sauts, seuils := ti47Sauts(m)
	instants := ti47Instants(sauts)
	t.Logf("SAUTS : %d variations hors norme (percentile %d par slot) sur %d emissions · %s",
		len(sauts), ti47SeuilSautPct, len(m.emissions), ti47Seuils(seuils))
	if len(sauts) > 0 {
		tri := make([]int64, 0, len(sauts))
		for _, s := range sauts {
			tri = append(tri, s.amplitude)
		}
		sort.Slice(tri, func(i, j int) bool { return tri[i] < tri[j] })
		t.Logf("   amplitude des sauts : mediane %d · max %d", tri[len(tri)/2], tri[len(tri)-1])
	}

	var toutes []int
	for _, e := range m.emissions {
		if e.tMS >= 0 {
			toutes = append(toutes, e.tMS)
		}
	}
	sort.Ints(toutes)
	t.Logf("ALIGNEMENT (item 1.2) — chaque suite contre son temoin « meme loi, decalee » :")
	ti47Confronte(t, "sauts vs TOUS oracles", instants, tous)
	ti47Confronte(t, "emissions vs TOUS oracles", toutes, tous)
	for _, k := range types {
		ti47Confronte(t, "sauts vs "+k, instants, parType[k])
	}
	ti47Matrice(t, sauts, parType, types)
}

// ti47Seuils resume les seuils de saut par slot.
func ti47Seuils(seuils map[uint32]int64) string {
	if len(seuils) == 0 {
		return "aucun slot n'a assez d'emissions pour un seuil"
	}
	v := make([]int64, 0, len(seuils))
	for _, s := range seuils {
		v = append(v, s)
	}
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	return fmt.Sprintf("seuil median %d (min %d, max %d) sur %d slots",
		v[len(v)/2], v[0], v[len(v)-1], len(v))
}

// ti47Matrice publie la matrice classe de saut x type d'oracle (items 1.3 et 1.4).
//
// LA CLASSE EST L'AMPLITUDE, faute d'enumeration : la valeur du canal est un continuum (12 760
// valeurs distinctes pour 14 451 emissions sur `7344d24f`), donc « chaque valeur corrèle-t-elle a
// un type d'evenement » n'a pas de sujet. L'amplitude par decade est la seule partition que la
// donnee autorise.
func ti47Matrice(t *testing.T, sauts []ti47Saut, parType map[string][]int, types []string) {
	t.Helper()
	classes := map[int][]int{}
	for _, s := range sauts {
		d := 0
		for v := s.amplitude; v >= 10; v /= 10 {
			d++
		}
		classes[d] = append(classes[d], s.tMS)
	}
	ds := make([]int, 0, len(classes))
	for d := range classes {
		ds = append(ds, d)
	}
	sort.Ints(ds)
	t.Logf("MATRICE classe de saut x type d'oracle (item 1.3) — part des sauts de la classe"+
		" a moins de %d s d'un evenement du type :", ti47FenetreMS/1000)
	for _, d := range ds {
		ins := classes[d]
		sort.Ints(ins)
		var parts []string
		orphelins := 0
		for _, k := range types {
			n := 0
			for _, v := range ti47Distances(ins, parType[k]) {
				if v <= ti47FenetreMS {
					n++
				}
			}
			parts = append(parts, fmt.Sprintf("%s %.0f %%", k, 100*float64(n)/float64(len(ins))))
		}
		for _, v := range ti47Distances(ins, ti47Fusionne(parType)) {
			if v > ti47FenetreMS {
				orphelins++
			}
		}
		t.Logf("   amplitude 10^%d (%5d sauts) : %s · SANS oracle : %.0f %% (item 1.4)",
			d, len(ins), strings.Join(parts, " · "),
			100*float64(orphelins)/float64(len(ins)))
	}
}

// ti47Fusionne rend tous les instants d'oracle, tries.
func ti47Fusionne(parType map[string][]int) []int {
	var out []int
	for _, v := range parType {
		out = append(out, v...)
	}
	sort.Ints(out)
	return out
}
