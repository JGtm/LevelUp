package main

// garde_perte_test.go — LE GARDE MORD-IL SUR LE SINISTRE REEL ?
//
// Le temoin n est pas un cas de laboratoire : c est la forme EXACTE de la perte du 2026-09-03 —
// memes comptes de cartes, memes comptes de zones, seuls les SOMMETS baissent. Si le garde ne
// mord pas la-dessus, il ne sert a rien, parce que c est precisement ce que les invariants
// existants (22 cartes / 816 zones / 816 libelles) laissaient passer.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// carteA fabrique une entree a `sommets` sommets repartis sur UNE zone.
func carteA(module string, sommets int) replay.MapCalloutsEntry {
	poly := make([][2]float64, sommets)
	for i := range poly {
		poly[i] = [2]float64{float64(i), float64(i)}
	}
	return replay.MapCalloutsEntry{
		Module:     module,
		Provenance: replay.CalloutsProvenanceBrut,
		Zones:      []replay.CalloutZone{{VolumeIndex: 1, EN: "Zone", FR: "Zone", Polygon: poly}},
	}
}

func catalogue(natives map[string]int, forge map[string]int) replay.MapCalloutsCatalog {
	c := replay.MapCalloutsCatalog{
		SchemaVersion: replay.MapCalloutsSchemaVersion,
		TitleSlug:     "halo_infinite",
		Maps:          map[string]replay.MapCalloutsEntry{},
		MapsByID:      map[string]replay.MapCalloutsEntry{},
	}
	for m, n := range natives {
		c.Maps[m] = carteA(m, n)
	}
	for id, n := range forge {
		c.MapsByID[id] = carteA("", n)
	}
	return c
}

// ecritSur pose un catalogue sur disque, comme le ferait une passe precedente.
func ecritSur(t *testing.T, cat replay.MapCalloutsCatalog) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "map_callouts.json")
	blob, err := json.MarshalIndent(cat, "", " ")
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if err := os.WriteFile(p, blob, 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	return p
}

// TestGardePerteMordSurLeSinistreReel — MEMES CARTES, MEMES ZONES, MOINS DE SOMMETS.
//
// C est la signature du 2026-09-03 : les trois invariants deja poses restent verts et le
// catalogue a pourtant perdu environ 16 400 sommets. Le garde doit refuser.
func TestGardePerteMordSurLeSinistreReel(t *testing.T) {
	avant := catalogue(map[string]int{"btb_fragmentation": 400, "catalyst": 120}, nil)
	apres := catalogue(map[string]int{"btb_fragmentation": 40, "catalyst": 120}, nil)

	if len(avant.Maps) != len(apres.Maps) {
		t.Fatalf("temoin invalide : le nombre de cartes doit etre IDENTIQUE")
	}
	if len(avant.Maps["btb_fragmentation"].Zones) != len(apres.Maps["btb_fragmentation"].Zones) {
		t.Fatalf("temoin invalide : le nombre de zones doit etre IDENTIQUE")
	}

	p := ecritSur(t, avant)
	err := verifiePasDePerte(p, apres, false)
	if err == nil {
		t.Fatal("perte de 360 sommets acceptee : le garde ne mord pas sur le sinistre du 2026-09-03")
	}
	if perdantes := pertes(avant, apres); len(perdantes) != 1 ||
		perdantes[0].Cle != "btb_fragmentation" || perdantes[0].Avant-perdantes[0].Apres != 360 {
		t.Fatalf("pertes mal chiffrees : %+v", perdantes)
	}
}

// TestGardePerteLaissePasserUnGain : une passe qui AJOUTE de la geometrie doit s ecrire. C est le
// cas nominal du decoupage — 42 zones de sgh_interlock ont gagne des sommets le 2026-09-03.
func TestGardePerteLaissePasserUnGain(t *testing.T) {
	avant := catalogue(map[string]int{"sgh_interlock": 100}, nil)
	apres := catalogue(map[string]int{"sgh_interlock": 140}, nil)
	if err := verifiePasDePerte(ecritSur(t, avant), apres, false); err != nil {
		t.Fatalf("un gain de sommets a ete refuse : %v", err)
	}
}

// TestGardePerteCompteLesPartsEtLesTrous : le decoupage range une partie de la forme dans `Parts`
// et `Holes`. Ne compter que `Polygon` ferait passer un decoupage REUSSI pour une perte, et le
// garde bloquerait la chaine nominale.
func TestGardePerteCompteLesPartsEtLesTrous(t *testing.T) {
	avant := catalogue(map[string]int{"ridgeline": 100}, nil)

	apres := catalogue(map[string]int{"ridgeline": 30}, nil)
	e := apres.Maps["ridgeline"]
	z := e.Zones[0]
	z.Parts = [][][2]float64{make([][2]float64, 50)}
	z.Holes = [][][2]float64{make([][2]float64, 25)}
	e.Zones[0] = z
	apres.Maps["ridgeline"] = e

	if n := sommetsParCarte(apres)["ridgeline"]; n != 105 {
		t.Fatalf("sommets comptes = %d, attendu 105 (30 contour + 50 parties + 25 trous)", n)
	}
	if err := verifiePasDePerte(ecritSur(t, avant), apres, false); err != nil {
		t.Fatalf("decoupage nominal refuse : %v", err)
	}
}

// TestGardePerteVoitDisparaitreUneCarte : une carte ABSENTE de la sortie est la pire des pertes.
// Le garde la compte comme une perte totale plutot que de l ignorer.
func TestGardePerteVoitDisparaitreUneCarte(t *testing.T) {
	avant := catalogue(map[string]int{"chasm": 80}, map[string]int{"abc-123": 60})
	apres := catalogue(map[string]int{"chasm": 80}, nil)
	if err := verifiePasDePerte(ecritSur(t, avant), apres, false); err == nil {
		t.Fatal("la disparition d une carte Forge a ete acceptee")
	}
}

// TestGardePerteEchappatoireExplicite : `--accepte-perte` leve le refus. L echappatoire existe,
// elle est nommee, et elle ne doit pas se declencher toute seule — le test verifie les DEUX sens.
func TestGardePerteEchappatoireExplicite(t *testing.T) {
	avant := catalogue(map[string]int{"streets": 200}, nil)
	apres := catalogue(map[string]int{"streets": 50}, nil)
	p := ecritSur(t, avant)
	if err := verifiePasDePerte(p, apres, false); err == nil {
		t.Fatal("sans le drapeau, la perte doit etre refusee")
	}
	if err := verifiePasDePerte(p, apres, true); err != nil {
		t.Fatalf("avec --accepte-perte, la perte doit passer : %v", err)
	}
}

// TestGardePerteSansCatalogueExistant : le premier ecrit d un titre n a rien a comparer, et ce
// n est pas une faute. Un garde qui echouerait la rendrait la chaine impossible a amorcer.
func TestGardePerteSansCatalogueExistant(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "pas_de_catalogue.json")
	if err := verifiePasDePerte(absent, catalogue(map[string]int{"forest": 10}, nil), false); err != nil {
		t.Fatalf("premier ecrit refuse : %v", err)
	}
}
