package grammar

// lecteur_position_regions_test.go — LA TABLE PAR INDEX DE REGION DE `FUN_14076e524`, AU BIT PRES.
//
// Un flux de position nomme sa region (la porte a 0, puis l index sur `DAT_144632be0` bits) et les
// trois axes qui suivent sont aux largeurs de LA LIGNE DE CET INDEX. Chaque cas fabrique le flux que
// l ecrivain pose et exige que le lecteur en consomme exactement la longueur, aux largeurs de la
// region designee : l index de la region jouee, un autre index aux largeurs differentes, un index
// que la table ne declare pas (zero bit par axe, comme le jeu).
//
// LES LARGEURS SONT ECRITES EN CLAIR (cf. `lecteur_position_sites_test.go`), tirees de l etendue
// des bornes par la loi `min(26, ceilLog2(ceil(etendue * 60)))` au niveau 0x10, `ceil(etendue * 30)`
// au niveau 0xf :
//
//	region 0   100 x 100 x 200        6000 / 6000 / 12000   -> 13/13/14   (niveau 0xf : 12/12/13)
//	region 1    50 x  50 x  40        3000 / 3000 /  2400   -> 12/12/12   (niveau 0xf : 11/11/11)
//	region 2   150 x 150 x 150        9000 / 9000 /  9000   -> 14/14/14
//	region 3  4000 x 4000 x 1000   240000 / 240000 / 60000  -> 18/18/16   (niveau 0xf : 17/17/15)
//
// La region jouee est la 1 sur deux bits d index, comme Live Fire.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// regionsDeTest rend une table de quatre regions, l index etant le rang.
func regionsDeTest() []profile.RegionQuant {
	cote := func(i uint32, e [3]float32, w [3]uint) profile.RegionQuant {
		return profile.RegionQuant{Index: i, Min: [3]float32{0, 0, 0}, Max: e, AxisWidths: w}
	}
	return []profile.RegionQuant{
		cote(0, [3]float32{100, 100, 200}, [3]uint{13, 13, 14}),
		cote(1, [3]float32{50, 50, 40}, [3]uint{12, 12, 12}),
		cote(2, [3]float32{150, 150, 150}, [3]uint{14, 14, 14}),
		cote(3, [3]float32{4000, 4000, 1000}, [3]uint{18, 18, 16}),
	}
}

// entreeAQuatreRegions est l entree de catalogue de la carte d essai : la region 1 est la jouee.
func entreeAQuatreRegions(avecTable bool) profile.MapQuantEntry {
	e := profile.MapQuantEntry{Module: "essai", Region: 1, RegionIndexBits: 2,
		Min: [3]float32{0, 0, 0}, Max: [3]float32{50, 50, 40}, AxisWidths: [3]uint{12, 12, 12}}
	if avecTable {
		e.Regions = regionsDeTest()
	}
	return e
}

// lecteurSousLaCarte rend un lecteur pose sur `buf` sous l entree de la carte d essai, la table par
// index posee par le geste de production ([ProfilDeBalayage.PoserLesRegionsDeLaCarte]) ou non.
func lecteurSousLaCarte(buf []byte, e profile.MapQuantEntry) *Lecteur {
	br := LecteurSur(buf)
	p := br.Profil()
	p.PoserLargeursObjetDuMondeDepuisDecoupage(e.Layout())
	p.PoserLesRegionsDeLaCarte(e)
	br.PoserProfil(p)
	return br
}

func TestLaPositionLitLesLargeursDeLaRegionQueLeFluxDesigne(t *testing.T) {
	cas := []struct {
		nom    string
		idx    int
		niveau int
		axes   [3]uint
	}{
		{"region jouee (1)", 1, niveauPosition, [3]uint{12, 12, 12}},
		{"region 0, autres largeurs que la jouee", 0, niveauPosition, [3]uint{13, 13, 14}},
		{"region 2", 2, niveauPosition, [3]uint{14, 14, 14}},
		{"region 3, 19 bits ecart sur la jouee a la maniere d Interlock", 3, niveauPosition, [3]uint{18, 18, 16}},
		{"region 3 au niveau 0xf", 3, niveauDetonation, [3]uint{17, 17, 15}},
		{"region 1 au niveau 0xf", 1, niveauDetonation, [3]uint{11, 11, 11}},
		{"porte posee : la table DEFAUT, jamais une region", -1, niveauPosition, axesDefautNiveau16},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(e524(c.idx, 2, c.axes))
			br := lecteurSousLaCarte(buf, entreeAQuatreRegions(true))
			pos := lireE524(br, c.niveau)
			if got := br.BitPos(); got != total {
				t.Fatalf("%d bits lus, l ecrivain en pose %d", got, total)
			}
			if pos.idx != c.idx || pos.w != c.axes {
				t.Errorf("index %d largeurs %v, attendu %d %v", pos.idx, pos.w, c.idx, c.axes)
			}
			for axe, w := range c.axes {
				if pos.q[axe] != motif(w) {
					t.Errorf("axe %d : quantum %#x, attendu %#x", axe, pos.q[axe], motif(w))
				}
			}
		})
	}
}

// TestUnIndexSansLigneNeLitAucunAxe : la table de la carte n a pas de ligne pour l index lu — il
// depasse le nombre de regions (la region 3 n est pas declaree), ou la region est invalide. Le jeu
// lit la porte et l index, puis ZERO bit par axe, sans retomber sur la region jouee ; le lecteur en
// fait autant et compte l occurrence.
func TestUnIndexSansLigneNeLitAucunAxe(t *testing.T) {
	absente := entreeAQuatreRegions(true)
	absente.Regions = absente.Regions[:3]
	invalide := entreeAQuatreRegions(true)
	invalide.Regions[3].Invalid = true
	for nom, e := range map[string]profile.MapQuantEntry{"index au-dela du nombre de regions": absente, "region invalide": invalide} {
		t.Run(nom, func(t *testing.T) {
			buf, _ := ecrireFlux(e524(3, 2, [3]uint{18, 18, 16}))
			br := lecteurSousLaCarte(buf, e)
			br.PoserContexte(ContexteDeLecture{Profil: br.Profil(), Obs: &Observation{}})
			pos, ok := lireE524Sur(br, niveauPosition, br.tablesDuProfil())
			if !ok || !pos.sansLigne {
				t.Fatalf("ok=%v sansLigne=%v : la lecture rend une position sans ligne", ok, pos.sansLigne)
			}
			if got := br.BitPos(); got != 3 { // porte + index sur 2 bits, aucun axe
				t.Errorf("%d bits lus, attendu 3 (porte + index)", got)
			}
			if pos.idx != 3 || pos.w != ([3]uint{}) {
				t.Errorf("index %d largeurs %v, attendu 3 et aucune largeur", pos.idx, pos.w)
			}
			if got := br.obs.IndexSansLigne[3]; got != 1 {
				t.Errorf("IndexSansLigne[3] = %d, attendu 1", got)
			}
		})
	}
}

// TestSansTableLaLigneDeLaRegionJoueeVautPourToutIndex : une entree de catalogue anterieure a la
// table lit, comme avant, la ligne de la region jouee pour tout index du profil.
func TestSansTableLaLigneDeLaRegionJoueeVautPourToutIndex(t *testing.T) {
	buf, total := ecrireFlux(e524(3, 2, [3]uint{12, 12, 12}))
	br := lecteurSousLaCarte(buf, entreeAQuatreRegions(false))
	if pos := lireE524(br, niveauPosition); pos.idx != 3 || pos.w != ([3]uint{12, 12, 12}) {
		t.Errorf("index %d largeurs %v", pos.idx, pos.w)
	}
	if got := br.BitPos(); got != total {
		t.Fatalf("%d bits lus, attendu %d", got, total)
	}
}

// TestLaVueALitUneAutreRegionQuandLaCarteLaDeclare : les positions des messages de la vue A
// (`tablesDeLaRegionJouee`) lisent tout index declare ; sans table, un autre index que la region
// jouee arrete la lecture, comme avant.
func TestLaVueALitUneAutreRegionQuandLaCarteLaDeclare(t *testing.T) {
	buf, total := ecrireFlux(e524(3, 2, [3]uint{18, 18, 16}))
	br := LecteurSur(buf)
	pos, ok := lireE494Sur(br, niveauPosition, tablesDeLaRegionJouee(entreeAQuatreRegions(true)))
	if !ok || pos.idx != 3 || pos.w != ([3]uint{18, 18, 16}) {
		t.Fatalf("avec table : ok=%v index %d largeurs %v", ok, pos.idx, pos.w)
	}
	if got := br.BitPos(); got != total {
		t.Fatalf("%d bits lus, l ecrivain en pose %d", got, total)
	}
	buf, _ = ecrireFlux(e524(3, 2, [3]uint{12, 12, 12}))
	br = LecteurSur(buf)
	if _, ok := lireE494Sur(br, niveauPosition, tablesDeLaRegionJouee(entreeAQuatreRegions(false))); ok {
		t.Error("sans table, un autre index que la region jouee arrete la lecture")
	}
}

// TestLaDetonationDequantifieDansLaRegionLue : la position d une detonation lue a l index 3 se
// dequantifie dans les bornes de la region 3, pas dans celles de la region jouee.
func TestLaDetonationDequantifieDansLaRegionLue(t *testing.T) {
	br := LecteurSur([]byte{0})
	br.vueA.positions = tablesDeLaRegionJouee(entreeAQuatreRegions(true))
	pos := positionQuantifiee{idx: 3, w: [3]uint{18, 18, 16}}
	pos.q = [3]uint64{1 << 17, 1 << 17, 1 << 15} // le milieu de chaque axe
	dequantDetonationPosition(br, pos)
	if x := br.detonationLue.X; x < 1990 || x > 2010 { // milieu de [0, 4000]
		t.Errorf("X = %v, attendu le milieu de la region 3 (2000)", x)
	}
	if z := br.detonationLue.Z; z < 490 || z > 510 { // milieu de [0, 1000]
		t.Errorf("Z = %v, attendu le milieu de la region 3 (500)", z)
	}
}
