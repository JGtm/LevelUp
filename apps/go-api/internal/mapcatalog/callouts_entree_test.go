package mapcatalog

// Témoins de l'entrée Forge : la jointure des libellés par string_id, la publication d'une
// zone muette, la mesure de couverture et la règle de publication.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
	"levelup/go-api/internal/testutil"
)

// zoneForge fabrique une zone déjà extraite : un carré de côté c centré en (x, y).
func zoneForge(index int, sid uint32, x, y, c float64) mapvar.ZoneNommee {
	h := c / 2
	return mapvar.ZoneNommee{
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
	lex := Lexique{0x11111111: {EN: "Cave", FR: "Grotte"}}
	zs := []mapvar.ZoneNommee{
		zoneForge(312, 0x11111111, 0, 0, 20),
		zoneForge(101, 0x22222222, 60, 60, 12), // string_id sans texte joueur
	}
	entry, couv := EntreeCalloutsDepuisZones(zs, lex)

	if entry.Provenance != replay.CalloutsProvenanceMvar || entry.Module != "" {
		t.Fatalf("entrée = provenance %q / module %q, attendu mvar / vide", entry.Provenance, entry.Module)
	}
	if couv.Nommees != 1 || couv.Zones != 2 || len(entry.Zones) != 2 {
		t.Fatalf("zones = %d dont %d nommées, attendu 2 dont 1", len(entry.Zones), couv.Nommees)
	}
	// L'ordre est celui de l'indice d'objet : deux exécutions rendent le même fichier.
	if entry.Zones[0].VolumeIndex != 101 || entry.Zones[1].VolumeIndex != 312 {
		t.Errorf("ordre = %d, %d — attendu 101 puis 312", entry.Zones[0].VolumeIndex, entry.Zones[1].VolumeIndex)
	}
	// LA ZONE MUETTE EST PUBLIÉE, SANS NOM INVENTÉ : sa géométrie est mesurée.
	muette := entry.Zones[0]
	if muette.EN != "" || muette.FR != "" || muette.Name != "" || len(muette.Polygon) != 4 {
		t.Errorf("zone muette : en=%q fr=%q nom=%q polygone=%d sommets", muette.EN, muette.FR, muette.Name, len(muette.Polygon))
	}
	nommee := entry.Zones[1]
	if nommee.EN != "Cave" || nommee.FR != "Grotte" || nommee.Z != 2 || nommee.ZTop != 5 {
		t.Errorf("zone nommée : %+v", nommee)
	}
	if couv.Verdict() != VerdictPubliable || couv.SansLibelle() != 1 {
		t.Errorf("verdict %q, sans libellé %d ; attendu publiable et 1", couv.Verdict(), couv.SansLibelle())
	}
}

// TestEntreeForgeCompteLesStringIDPourLaMesure — la couverture des libellés se MESURE : deux
// volumes d'un même lieu ne comptent qu'un string_id.
func TestEntreeForgeCompteLesStringIDPourLaMesure(t *testing.T) {
	lex := Lexique{0xAAAA: {EN: "Base", FR: "Base"}}
	zs := []mapvar.ZoneNommee{
		zoneForge(1, 0xAAAA, 0, 0, 10),
		zoneForge(2, 0xAAAA, 40, 0, 10), // même lieu, deux volumes
		zoneForge(3, 0xBBBB, 80, 0, 10),
	}
	_, couv := EntreeCalloutsDepuisZones(zs, lex)
	if len(couv.StringIDs) != 2 || couv.SansLibelle() != 1 || couv.Nommees != 2 {
		t.Errorf("string_id : %d distincts / %d sans libellé / %d zones nommées, attendu 2 / 1 / 2",
			len(couv.StringIDs), couv.SansLibelle(), couv.Nommees)
	}
}

// TestVerdictAppliqueLaRegleDePublication — les trois issues de la règle unique.
func TestVerdictAppliqueLaRegleDePublication(t *testing.T) {
	cas := []struct {
		nom  string
		couv CouvertureLibelles
		veut VerdictZones
	}{
		{"aucune zone", CouvertureLibelles{}, VerdictSansZone},
		{"zones toutes muettes", CouvertureLibelles{Zones: 3}, VerdictSansLibelle},
		{"une zone nommée suffit", CouvertureLibelles{Zones: 3, Nommees: 1}, VerdictPubliable},
	}
	for _, c := range cas {
		if got := c.couv.Verdict(); got != c.veut {
			t.Errorf("%s : verdict %q, attendu %q", c.nom, got, c.veut)
		}
	}
}

// TestZoneSansLibelleResteUneZone — une zone dont le string_id n'a pas de texte garde sa
// géométrie et n'invente aucun nom ; la carte entière est alors « sans libellé ».
func TestZoneSansLibelleResteUneZone(t *testing.T) {
	entry, couv := EntreeCalloutsDepuisZones([]mapvar.ZoneNommee{zoneForge(1, 0xDEAD, 0, 0, 10)}, Lexique{})
	if couv.Nommees != 0 || len(entry.Zones) != 1 || couv.Verdict() != VerdictSansLibelle {
		t.Fatalf("zones = %d dont %d nommées, verdict %q ; attendu 1 dont 0, sans libellé",
			len(entry.Zones), couv.Nommees, couv.Verdict())
	}
	if z := entry.Zones[0]; z.EN != "" || z.FR != "" || z.Name != "" || len(z.Polygon) != 4 {
		t.Errorf("libellé inventé ou géométrie perdue sur une zone muette : %+v", z)
	}
}

// TestPasDeClassementSeDesserreSurLesGrandesEmprises — le raster ne doit pas exploser sur un
// canevas Forge : le pas natif tient pour une petite zone, il se desserre au-delà.
func TestPasDeClassementSeDesserreSurLesGrandesEmprises(t *testing.T) {
	petite := []FormeDeZone{{Index: 1, Contour: zoneForge(1, 1, 0, 0, 10).Contour}}
	if pas := pasDeClassementForge(petite); pas != PasDeClassementNatif {
		t.Errorf("emprise de 10 m : pas = %v, attendu le pas natif %v", pas, PasDeClassementNatif)
	}
	grande := []FormeDeZone{{Index: 1, Contour: zoneForge(1, 1, 0, 0, 400).Contour}}
	if pas := pasDeClassementForge(grande); pas <= PasDeClassementNatif {
		t.Errorf("emprise de 400 m : pas = %v, attendu desserré au-delà de %v", pas, PasDeClassementNatif)
	} else if cellules := 400 / pas; cellules > cellulesDeClassementMax+1 {
		t.Errorf("emprise de 400 m : %v cellules, plafond %d", cellules, cellulesDeClassementMax)
	}
}

// TestEntreeForgeSurUneVarianteSansZone — la chaîne complète sur des octets RÉELS du dépôt,
// sans réseau ni jeu installé. `vagabond_fo08_wetland.mvar` est le fichier-lien de Vagabond
// vers son CANEVAS : il décode, et ne pose aucune zone nommée — le verdict doit le dire, sans
// erreur. Des octets qui ne sont pas une variante, eux, échouent.
func TestEntreeForgeSurUneVarianteSansZone(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(root, ".ai", "V7.5", "dumps", "mapvar", "vagabond_fo08_wetland.mvar"))
	if err != nil {
		t.Fatalf("variante versionnée illisible : %v", err)
	}
	entry, couv, err := EntreeCalloutsForge(blob, Lexique{})
	if err != nil {
		t.Fatalf("EntreeCalloutsForge : %v", err)
	}
	if couv.Verdict() != VerdictSansZone || len(entry.Zones) != 0 {
		t.Fatalf("fichier-lien de canevas : verdict %q, %d zones ; attendu sans zone", couv.Verdict(), len(entry.Zones))
	}
	if _, _, err := EntreeCalloutsForge([]byte("pas une variante"), Lexique{}); err == nil {
		t.Error("octets illisibles : erreur attendue")
	}
}
