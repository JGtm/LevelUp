package mapvar

import (
	"os"
	"path/filepath"
	"testing"
)

// LES CARTES FORGE ONT DES CALLOUTS, ET LE FICHIER NE MANQUAIT PAS.
//
// Ce temoin fige la mesure du 2026-08-27 qui a renverse une affirmation ecrite partout dans nos
// en-tetes (« une carte Forge n'a pas de zones nommees »). Isolation en porte 18, portees par
// les objets de sa propre variante, celle que la chaine de cuisson lit deja.
//
// Il est ancre sur des STRINGID, pas sur un compte : un compte seul laisserait passer une
// regression qui perdrait « cave » en gagnant deux zones ailleurs.
func TestZonesNommeesDIsolation(t *testing.T) {
	v := varianteDeTest(t, "isolation_map.mvar")

	const typeZoneNommee = -696190206
	sids := map[uint32]int{}
	for _, o := range v.Objects {
		if o.LocationID == 0 {
			continue
		}
		if o.TypeID != typeZoneNommee {
			t.Fatalf("objet #%d porte un StringId de lieu sous le type %d : le type n'est plus le seul porteur",
				o.Index, o.TypeID)
		}
		sids[o.LocationID]++
	}
	if len(sids) == 0 {
		t.Fatal("aucune zone nommee lue : la lecture du chemin #8/4[]/0/0 est cassee")
	}
	// Les six lieux dont le nom nous est connu par le catalogue natif — ils tiennent le
	// temoin a l'identite, pas au volume.
	attendus := map[uint32]string{
		0x52AA091A: "bottom mid",
		0x4F9BDABE: "cave",
		0x555CA63B: "top mid",
		0xEFF359A0: "north base",
		0xF6EB6E82: "south base",
		0x362481A3: "pipes",
	}
	for sid, nom := range attendus {
		if sids[sid] == 0 {
			t.Errorf("lieu %q (%#08x) absent des zones lues", nom, sid)
		}
	}
	total := 0
	for _, n := range sids {
		total += n
	}
	if total != 18 {
		t.Errorf("zones posees : %d, attendu 18 (mesure du 2026-08-27)", total)
	}
}

// TestZonesNommeesPortentUneForme — une zone sans forme ne borne rien. Les 18 zones
// d'Isolation portent toutes la leur ; c'est ce qui rend le rognage possible.
func TestZonesNommeesPortentUneForme(t *testing.T) {
	v := varianteDeTest(t, "isolation_map.mvar")
	n, sansForme := 0, 0
	for _, o := range v.Objects {
		if o.LocationID == 0 {
			continue
		}
		n++
		if o.Shape() == nil {
			sansForme++
		}
	}
	if n == 0 {
		t.Skip("aucune zone lue")
	}
	if sansForme != 0 {
		t.Errorf("%d zones sur %d sans forme exploitable", sansForme, n)
	}
}

func varianteDeTest(t *testing.T, nom string) *Variant {
	t.Helper()
	chemin := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp",
		".ai", "re_dump", "mapvar", nom)
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Skipf("variante absente : %v", err)
	}
	v, err := Parse(brut)
	if err != nil {
		t.Fatalf("variante illisible : %v", err)
	}
	return v
}
