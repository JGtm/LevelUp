package main

// Témoins de la passe FORGE : la jointure des libellés par string_id, la publication d'une
// zone muette, et la lecture de l'inventaire UGC.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/himap"
)

// zoneForge fabrique une zone déjà extraite : un carré de côté c centré en (x, y).
func zoneForge(index int, sid uint32, x, y, c float64) himap.ZoneNommee {
	h := c / 2
	return himap.ZoneNommee{
		Index:    index,
		StringID: sid,
		Pos:      [3]float64{x, y, 2},
		Contour:  [][2]float64{{x - h, y - h}, {x + h, y - h}, {x + h, y + h}, {x - h, y + h}},
		ZBas:     0,
		ZHaut:    5,
	}
}

// TestEntreeForgeJointLesLibellesParStringID — la SEULE clé de jointure possible pour une
// carte Forge (elle n'a ni module ni indice de volume levl).
func TestEntreeForgeJointLesLibellesParStringID(t *testing.T) {
	labels := libellesParStringID{
		0x11111111: {en: "Cave", fr: "Grotte", stringID: 0x11111111},
	}
	zs := []himap.ZoneNommee{
		zoneForge(312, 0x11111111, 0, 0, 20),
		zoneForge(101, 0x22222222, 60, 60, 12), // string_id sans texte joueur
	}
	entry, nommees := entreeDepuisZones(zs, labels, nouvellesStats())

	if entry.Provenance != replay.CalloutsProvenanceMvar || entry.Module != "" {
		t.Fatalf("entrée = provenance %q / module %q, attendu mvar / vide", entry.Provenance, entry.Module)
	}
	if nommees != 1 || len(entry.Zones) != 2 {
		t.Fatalf("zones = %d dont %d nommées, attendu 2 dont 1", len(entry.Zones), nommees)
	}
	// L'ordre est celui de l'indice d'objet : deux exécutions rendent le même fichier.
	if entry.Zones[0].VolumeIndex != 101 || entry.Zones[1].VolumeIndex != 312 {
		t.Errorf("ordre = %d, %d — attendu 101 puis 312", entry.Zones[0].VolumeIndex, entry.Zones[1].VolumeIndex)
	}
	// LA ZONE MUETTE EST PUBLIÉE, SANS NOM INVENTÉ : sa géométrie est mesurée.
	muette := entry.Zones[0]
	if muette.EN != "" || muette.FR != "" || len(muette.Polygon) != 4 {
		t.Errorf("zone muette : en=%q fr=%q polygone=%d sommets", muette.EN, muette.FR, len(muette.Polygon))
	}
	nommee := entry.Zones[1]
	if nommee.EN != "Cave" || nommee.FR != "Grotte" || nommee.Z != 2 || nommee.ZTop != 5 {
		t.Errorf("zone nommée : %+v", nommee)
	}
}

// TestEntreeForgeCompteLesStringIDPourLaMesure — la couverture des libellés se MESURE, elle
// ne s'estime pas : le compteur distingue les string_id vus des string_id résolus.
func TestEntreeForgeCompteLesStringIDPourLaMesure(t *testing.T) {
	stats := nouvellesStats()
	labels := libellesParStringID{0xAAAA: {en: "Base", fr: "Base", stringID: 0xAAAA}}
	zs := []himap.ZoneNommee{
		zoneForge(1, 0xAAAA, 0, 0, 10),
		zoneForge(2, 0xAAAA, 40, 0, 10), // même lieu, deux volumes
		zoneForge(3, 0xBBBB, 80, 0, 10),
	}
	entreeDepuisZones(zs, labels, stats)
	if len(stats.SidsDistincts) != 2 || len(stats.SidsResolus) != 1 {
		t.Errorf("string_id : %d distincts / %d résolus, attendu 2 / 1",
			len(stats.SidsDistincts), len(stats.SidsResolus))
	}
}

// TestPasDeClassementSeDesserreSurLesGrandesEmprises — le raster ne doit pas exploser sur un
// canevas Forge : le pas natif tient pour une petite zone, il se desserre au-delà.
func TestPasDeClassementSeDesserreSurLesGrandesEmprises(t *testing.T) {
	petite := []shapedPoly{{vi: 1, poly: zoneForge(1, 1, 0, 0, 10).Contour}}
	if pas := pasDeClassement(petite); pas != classifyCell {
		t.Errorf("emprise de 10 m : pas = %v, attendu le pas natif %v", pas, classifyCell)
	}
	grande := []shapedPoly{{vi: 1, poly: zoneForge(1, 1, 0, 0, 400).Contour}}
	if pas := pasDeClassement(grande); pas <= classifyCell {
		t.Errorf("emprise de 400 m : pas = %v, attendu desserré au-delà de %v", pas, classifyCell)
	} else if cellules := 400 / pas; cellules > pasClassementMax+1 {
		t.Errorf("emprise de 400 m : %v cellules, plafond %d", cellules, pasClassementMax)
	}
}

// TestChargeInventaireNeGardeQueLesCartesForgeAvecVariante — l'inventaire porte aussi les
// cartes natives et des entrées sans map.mvar : elles n'ont rien à faire dans la passe.
func TestChargeInventaireNeGardeQueLesCartesForgeAvecVariante(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inv.json")
	body := `{"schema_version":1,"cartes":[
	 {"map_id":"aaa","nom":"Native","famille":"native","mvar":["map.mvar"],"blob_prefix":"https://x/"},
	 {"map_id":"bbb","nom":"SansVariante","famille":"forge","mvar":["fo11_blank.mvar"],"blob_prefix":"https://x/"},
	 {"map_id":"ccc","nom":"Bonne","famille":"forge","mvar":["fo11_blank.mvar","map.mvar"],"blob_prefix":"https://x/"}]}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cibles, err := chargeInventaire(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cibles) != 1 || cibles[0].MapID != "ccc" {
		t.Fatalf("cibles = %+v, attendu la seule carte Forge portant map.mvar", cibles)
	}
}

// TestChargeInventaireVideEstUneErreur — un inventaire sans cible est une CONFIGURATION
// cassée (mauvais fichier, schéma changé), pas un corpus vide : il doit se voir.
func TestChargeInventaireVideEstUneErreur(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inv.json")
	if err := os.WriteFile(p, []byte(`{"schema_version":1,"cartes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeInventaire(p); err == nil {
		t.Error("inventaire sans carte Forge : attendu une erreur, obtenu nil")
	}
}

// TestConstruitPasseForgeCompteLesVariantesAbsentes — un cache vide n'est pas une panne :
// chaque carte manquante est comptée et nommée, aucune n'entre au catalogue.
func TestConstruitPasseForgeCompteLesVariantesAbsentes(t *testing.T) {
	stats := nouvellesStats()
	out := construitPasseForge([]carteUGC{
		{MapID: "aaa", Nom: "Absente1"}, {MapID: "bbb", Nom: "Absente2"},
	}, t.TempDir(), libellesParStringID{}, stats)
	if len(out) != 0 {
		t.Errorf("cartes publiées = %d, attendu 0", len(out))
	}
	if len(stats.Illisibles) != 2 || stats.Cartes != 0 {
		t.Errorf("stats = %d illisibles / %d cartes, attendu 2 / 0", len(stats.Illisibles), stats.Cartes)
	}
}

// TestZoneSansLibelleResteUneZone — la règle de publication, prise par son autre bout : une
// zone dont le string_id n'a pas de texte joueur garde sa géométrie et n'invente aucun nom.
// C'est `construitPasseForge` qui écarte la carte quand AUCUNE de ses zones n'est nommable.
func TestZoneSansLibelleResteUneZone(t *testing.T) {
	entry, nommees := entreeDepuisZones([]himap.ZoneNommee{zoneForge(1, 0xDEAD, 0, 0, 10)},
		libellesParStringID{}, nouvellesStats())
	if nommees != 0 || len(entry.Zones) != 1 {
		t.Fatalf("zones = %d dont %d nommées, attendu 1 dont 0", len(entry.Zones), nommees)
	}
	z := entry.Zones[0]
	if z.EN != "" || z.FR != "" || z.Name != "" {
		t.Errorf("libellé inventé sur une zone muette : %+v", z)
	}
	if len(z.Polygon) != 4 {
		t.Errorf("polygone = %d sommets, attendu 4 — la géométrie ne dépend pas du nom", len(z.Polygon))
	}
}
