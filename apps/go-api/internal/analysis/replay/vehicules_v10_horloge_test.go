package replay

// vehicules_v10_horloge_test.go — CONVERSION HORLOGE DU FILM -> TEMPS DE MATCH pour les
// candidates du lot V10. LECTURE SEULE, garde par V4_ROOT / V4_FILMS.
//
// LE DEFAUT QU IL FERME. Le § 4.3 du rapport V10 datait ses candidates en HORODATAGE DE PAQUET
// (`BipedPosition.TimestampUS`). Ce n est PAS un temps de match : `origin.go` l etablit, c est une
// horloge MOTEUR — un temps depuis le demarrage du jeu, qui vaut 2 259 a 8 583 s sur les films
// temoins. Sous cette forme une candidate a « 2 519,9 s » est inverifiable dans le Theatre d un
// match qui dure 654 s.
//
// LA CONVERSION, ET ELLE NE S INVENTE PAS — c est celle de `origin.go` :
//
//	le PREMIER PAQUET DU CHUNK 1 est le debut du film, auquel le manifeste donne `start_ms = 0` :
//	c est donc le ZERO de l horloge de match (celle des `event_time_ms`). D ou
//
//	    tempsMatchMS = (horodatagePaquetUS - ScanFilmClockOrigin(dir)) / 1000
//
//	et, accessoirement, `originMs` du document publie n est rien d autre que cette meme
//	soustraction appliquee au PREMIER PAQUET DE POSITION.
//
// LE REPERE DE VALIDATION, INDEPENDANT ET VERIFIABLE. Sous `V10_ORIGIN=1` l instrument recalcule
// `originMs = (premier paquet de position - premier paquet du chunk 1) / 1000` par le MEME chemin
// que la production, et l affiche a cote de la duree du rejeu. Il se confronte a la valeur que
// porte l artefact deja construit (`0d76e8f1` : `originMs = 10667`, `durationMs = 654200`) : si la
// conversion est juste, les deux coincident, et TOUTES les candidates du film tombent dans
// [0, originMs + durationMs]. Le second controle est porte par l instrument lui-meme (colonne
// `dansMatch`), film par film, sans rien supposer.
//
//	CGO_ENABLED=0 V4_ROOT=<depot>/data/cache V10_ORIGIN=1 \
//	  V4_FILMS="0d76e8f1:Behemoth,...,fccc61cd:Launch Site" \
//	  go test ./internal/analysis/replay/ -run '^TestV10Horloge$' -v -timeout 120m

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v10Cand est une candidate du § 4.3, datee sur l horloge du FILM (secondes).
type v10Cand struct {
	film    string
	slot    uint32
	famille string
	chuteS  float64 // debut du palier bas = l instant publiable
	finS    float64 // derniere lecture d i4 de la vie
	stricte bool    // appartient a la regle stricte (palier i4 <= 0,10, >= 2 lectures)
}

// v10Candidates : les 14 vies mesurees par `TestV10VitaliteTerminale` (regle large `i4 <= 0,25`),
// dont les 7 de la regle stricte. Les valeurs sont RECOPIEES de la mesure, pas recalculees ici :
// ce fichier ne fait que la CONVERSION D HORLOGE, et il doit rester rejouable en 30 s.
var v10Candidates = []v10Cand{
	{"0d76e8f1", 792, "wasp", 2519.9, 2523.1, true},
	{"21468645", 783, "falcon", 1734.8, 1738.0, true},
	{"4898d586", 804, "chassis inconnu", 4057.3, 4060.2, true},
	{"51d3ab9f", 778, "warthog", 2822.7, 2823.7, true},
	{"51d3ab9f", 784, "chopper", 2806.4, 2807.0, true},
	{"b232e02d", 776, "banshee", 1123.2, 1123.8, true},
	{"e1bdb97f", 776, "warthog", 3046.8, 3047.4, true},
	{"4898d586", 771, "warthog", 3847.0, 3850.5, false},
	{"4898d586", 772, "ghost", 3599.9, 3606.1, false},
	{"51d3ab9f", 772, "mongoose", 2577.2, 2577.3, false},
	{"51d3ab9f", 794, "wasp", 2881.7, 2882.1, false},
	{"8a049c50", 769, "warthog", 1418.9, 1419.8, false},
	{"e1bdb97f", 774, "ghost", 2947.6, 2952.7, false},
	{"e232ffce", 772, "warthog", 4734.0, 4737.0, false},
}

func TestV10Horloge(t *testing.T) {
	root := v4Root(t)
	zero := map[string]float64{}   // film -> zero de l horloge de match, en s (horloge film)
	fin := map[string]float64{}    // film -> dernier paquet, en s de match
	origin := map[string]float64{} // film -> originMs recalcule, en s (si V10_ORIGIN)
	frames := map[string]int{}
	for _, f := range v4Corpus(t) {
		dir := objChunkDir(root, f.ID)
		if filmdec.CountFilmChunks(dir) == 0 {
			t.Logf("V10 horloge %s : film absent du cache — saute", f.ID)
			continue
		}
		clk, err := ScanFilmClockOrigin(dir)
		if err != nil {
			t.Logf("V10 horloge %s : origine d horloge illisible (%v) — conversion IMPOSSIBLE", f.ID, err)
			continue
		}
		zero[f.ID] = float64(clk) / 1e6
		fin[f.ID] = (float64(v10DernierPaquetUS(dir)) - float64(clk)) / 1e6
		t.Logf("V10 horloge %s — zero de match (1er paquet du chunk 1) = %.3f s d horloge film · dernier paquet a %.1f s de match",
			f.ID, zero[f.ID], fin[f.ID])
		if os.Getenv("V10_ORIGIN") != "" {
			if o, n, ok := v10OriginMs(t, root, f); ok {
				origin[f.ID], frames[f.ID] = o, n
				t.Logf("    [CONTROLE] originMs recalcule = %.0f ms · frameCount = %d · durationMs = %d"+
					"  (repere connu `0d76e8f1` : originMs 10667, durationMs 654200)",
					o*1000, n, n*DefaultFrameIntervalMS)
			}
		}
	}
	v10TableConvertie(t, zero, fin, origin, frames)
}

// v10DernierPaquetUS rend l horodatage du DERNIER paquet du film — la borne haute qui permet de
// dire si une candidate tombe bien dans le match.
func v10DernierPaquetUS(dir string) uint64 {
	var last uint64
	for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.TimestampUS > last {
				last = pk.TimestampUS
			}
		}
	}
	return last
}

// v10OriginMs recalcule `originMs` PAR LE MEME CHEMIN QUE LA PRODUCTION : premier paquet de
// position moins premier paquet du chunk 1. C est le repere de validation.
func v10OriginMs(t *testing.T, root string, f v0Film) (float64, int, bool) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	dir := objChunkDir(root, f.ID)
	entry, ok := v4Carte(t, root, f.Carte)
	if !ok {
		return 0, 0, false
	}
	filmdec.SetWorldObjectPrecisionFromLayout(entry.Layout())
	wr := entry.Range()
	bip, ok := v4Bipedes(t, dir, entry, &wr)
	if !ok || len(bip) == 0 {
		return 0, 0, false
	}
	clk, err := ScanFilmClockOrigin(dir)
	if err != nil {
		return 0, 0, false
	}
	step := uint64(DefaultFrameIntervalMS) * 1000
	return (float64(bip[0].TimestampUS) - float64(clk)) / 1e6,
		frameSpan(bip, bip[0].TimestampUS, step), true
}

// v10TableConvertie rend le tableau final, pret a transmettre.
func v10TableConvertie(t *testing.T, zero, fin, origin map[string]float64, frames map[string]int) {
	cands := append([]v10Cand(nil), v10Candidates...)
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].stricte != cands[j].stricte {
			return cands[i].stricte
		}
		if cands[i].film != cands[j].film {
			return cands[i].film < cands[j].film
		}
		return cands[i].slot < cands[j].slot
	})
	t.Logf("\n########## CANDIDATES V10 EN TEMPS DE MATCH ##########")
	t.Logf("%-9s %-5s %-16s %10s %10s %8s %10s %s",
		"film", "slot", "famille", "chuteFilm", "chuteMatch", "mm:ss", "finMatch", "dansMatch")
	for _, c := range cands {
		z, ok := zero[c.film]
		if !ok {
			t.Logf("%-9s %-5d %-16s %10.1f %10s %8s %10s %s",
				c.film, c.slot, c.famille, c.chuteS, "?", "?", "?", "ORIGINE INCONNUE")
			continue
		}
		ch, fn := c.chuteS-z, c.finS-z
		dans := "OUI"
		if ch < 0 || ch > fin[c.film] {
			dans = fmt.Sprintf("NON (film 0..%.1f s)", fin[c.film])
		}
		t.Logf("%-9s %-5d %-16s %10.1f %10.1f %8s %10.1f %s",
			c.film, c.slot, c.famille, c.chuteS, ch, v10MMSS(ch), fn, dans)
	}
	for f, o := range origin {
		t.Logf("  [CONTROLE %s] originMs = %.0f ms · durationMs = %d ms · fenetre de rejeu [%.1f .. %.1f] s de match",
			f, o*1000, frames[f]*DefaultFrameIntervalMS, o,
			o+float64(frames[f]*DefaultFrameIntervalMS)/1000)
	}
}

// v10MMSS formate un temps de match en mm:ss — la forme avec laquelle on navigue dans le Theatre.
func v10MMSS(s float64) string {
	if s < 0 {
		return "avant 0"
	}
	total := int(s + 0.5)
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}
