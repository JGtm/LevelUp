package filmdec

// transloc_positions_film_test.go — VALIDATION SUR PIÈCES du va-et-vient publié par le
// chemin de PRODUCTION (P1bis) : sur le cas index (Dynasty `1b2d9e08`), les positions
// décodées de la charge de l'événement 117 doivent tomber sur les discontinuités de piste
// MESURÉES du plan (ancres A1/A2, PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03) — le même
// contrôle que le rapport R6 (18/18, écarts 0,00-0,26 m), joué cette fois par
// `ScanFilmTranslocatorTeleports` et non par l'instrument de recherche.
//
// AUCUN VERROU DE PAQUET : ce scanner n'appelle aucun déserialiseur de trame (cf. l'en-tête
// de transloc_events.go). Gardé par P1_FILM + P1_MAP (les films ne sont pas versionnés), skip
// par défaut ; le catalogue de bornes, lui, EST versionné et se lit à son chemin de dépôt.
//
//	CGO_ENABLED=0 P1_FILM=<depot>/data/cache/film_chunks/1b2d9e08 \
//	  P1_MAP=944396dd-5661-4a16-b1d8-a6053f762c55 \
//	  go test ./internal/analysis/filmdec/ -run '^TestP1bisPositionsDynasty$' -v -timeout 30m

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const p1MapEnv = "P1_MAP"

// p1bisTolM : tolérance 2D entre position décodée et discontinuité de piste. La piste
// publiée est arrondie au centimètre et échantillonnée à 100 ms : R6 a mesuré 0,00-0,26 m de
// résidu sur les 18 événements, la borne laisse la marge de la grille sans laisser passer
// une erreur de bornes (qui se compte en dizaines de mètres).
const p1bisTolM = 0.35

// p1bisAncre est une vérité terrain du plan : le saut attendu d'un événement, dans l'ordre
// des instants. `hasFrom` distingue les ancres dont seul le point d'ARRIVÉE est mesuré.
type p1bisAncre struct {
	slot         uint32
	fromX, fromY float64
	toX, toY     float64
	hasFrom      bool
}

// p1bisDynasty : les trois événements du cas index, dans l'ordre des instants
// ({535@1761, 560@3261, 560@3419} — constat (b) du plan). Départs et arrivées viennent des
// ancres A1/A2 et du constat P1bis, mesurés sur les PISTES PUBLIÉES de l'artefact.
var p1bisDynasty = []p1bisAncre{
	{slot: 535, fromX: 2.79, fromY: 152.17, toX: 17.34, toY: 135.50, hasFrom: true},
	{slot: 560, toX: 17.35, toY: 123.20},
	{slot: 560, fromX: 11.32, fromY: 116.69, toX: 18.34, toY: 120.19, hasFrom: true},
}

// p1bisCatalogue charge le catalogue de bornes versionné du titre et rend l'entrée demandée.
func p1bisCatalogue(t *testing.T, key string) MapQuantEntry {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", path, err)
	}
	entry, err := cat.Lookup(key)
	if err != nil {
		t.Fatalf("entrée %q du catalogue : %v", key, err)
	}
	return entry
}

// TestP1bisPositionsDynasty rejoue le va-et-vient du cas index par le chemin de production.
func TestP1bisPositionsDynasty(t *testing.T) {
	dir, key := os.Getenv(p1FilmEnv), os.Getenv(p1MapEnv)
	if dir == "" || key == "" {
		t.Skipf("%s ou %s absent : validation sur pièces sautée", p1FilmEnv, p1MapEnv)
	}
	entry := p1bisCatalogue(t, key)
	t.Logf("CARTE %q (module %s) : bornes min=%v max=%v largeurs=%v région=%d (%d bit(s))",
		key, entry.Module, entry.Min, entry.Max, entry.AxisWidths, entry.Region,
		entry.EffectiveRegionIndexBits())
	evts := ScanFilmTranslocatorTeleports(dir, &entry)
	positionnees := 0
	for _, e := range evts {
		if e.HasPositions {
			positionnees++
			t.Logf("EVENEMENT 117 slot %d @%dus : (%.2f,%.2f,%.2f) -> (%.2f,%.2f,%.2f)",
				e.Slot, e.TimestampUS, e.From[0], e.From[1], e.From[2], e.To[0], e.To[1], e.To[2])
			continue
		}
		t.Logf("EVENEMENT 117 slot %d @%dus : SANS POSITION (charge non lue)", e.Slot, e.TimestampUS)
	}
	t.Logf("BILAN : %d événement(s), %d positionné(s)", len(evts), positionnees)
	if !strings.Contains(dir, "1b2d9e08") {
		t.Log("film différent du cas index : constats chiffrés non contrôlés")
		return
	}
	if len(evts) != len(p1bisDynasty) {
		t.Fatalf("%d tête(s) 117, attendu %d sur le cas index (R1 §4.1)", len(evts), len(p1bisDynasty))
	}
	for i, want := range p1bisDynasty {
		p1bisCompare(t, i, want, evts[i])
	}
}

// p1bisCompare confronte UN événement décodé à son ancre et chiffre les écarts.
func p1bisCompare(t *testing.T, i int, want p1bisAncre, got TranslocatorTeleport) {
	t.Helper()
	if got.Slot != want.slot {
		t.Errorf("événement %d : slot %d, attendu %d", i, got.Slot, want.slot)
		return
	}
	if !got.HasPositions {
		t.Errorf("événement %d (slot %d) : aucune position décodée — la charge du 117 devait"+
			" être lue avec les bornes de la carte", i, got.Slot)
		return
	}
	dTo := math.Hypot(float64(got.To[0])-want.toX, float64(got.To[1])-want.toY)
	if dTo > p1bisTolM {
		t.Errorf("événement %d (slot %d) : arrivée (%.2f,%.2f), attendue (%.2f,%.2f) — écart %.2f m",
			i, got.Slot, got.To[0], got.To[1], want.toX, want.toY, dTo)
	}
	if !want.hasFrom {
		t.Logf("ECART événement %d (slot %d) : arrivée %.2f m (départ non ancré)", i, got.Slot, dTo)
		return
	}
	dFrom := math.Hypot(float64(got.From[0])-want.fromX, float64(got.From[1])-want.fromY)
	if dFrom > p1bisTolM {
		t.Errorf("événement %d (slot %d) : départ (%.2f,%.2f), attendu (%.2f,%.2f) — écart %.2f m",
			i, got.Slot, got.From[0], got.From[1], want.fromX, want.fromY, dFrom)
	}
	t.Logf("ECART événement %d (slot %d) : départ %.2f m · arrivée %.2f m", i, got.Slot, dFrom, dTo)
}
