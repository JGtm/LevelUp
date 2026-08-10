package himap

// SONDE D'INVESTIGATION (2026-08-09, non destinee au commit) — le bloc
// `instanced physics instances` du sbsp delimite-t-il la zone jouable ?
//
// Structure lue dans le plugin sbsp.xml (ligne 913), Reclaimer ne lit PAS ce bloc :
// la verification vient d'invariants JOUES ici (orthonormalite de la base, bornes,
// drapeaux declares) et d'une mutation de decalage d'offset qui doit les casser.
//
//	_41 m_collisionTagReference   @0    (28 o, GlobalID presume a +8 comme MeshRef)
//	_40 per-instance material     @28   (20 o)
//	_6  Flags/Vector1/Vector2     @48
//	_19 Scale                     @60
//	_19 Forward                   @72
//	_19 Left                      @84
//	_19 Up                        @96
//	_17 Position                  @108
//	_D  m_typeMask                @120  (Bullet, Play, InvisibleWall, Render)
//	_6  m_guid                    @124
//	_39 havok body ID array       @128  (32 o)
//	_F  flags + pad(7)            @160
//	_7  m_scene                   @168
//	                       stride = 176

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himodule"
)

const physiqueBlockName = "instanced physics instances"

// MESURE 2026-08-09 : la somme des champs du plugin donne 176, mais les tailles de bloc
// REELLES (754 176 pour cliffhanger, 1 421 184 pour catalyst) divisees par les comptes du
// root block donnent TOUTES DEUX 192 = 0xC0. Le stride vrai est donc 192 ; les 16 octets
// excedentaires sont a localiser par le scan d'orthonormalite ci-dessous.
const (
	phyStride      = 192
	phyOffTagRef   = 0
	phyOffScale    = 60
	phyOffForward  = 72
	phyOffLeft     = 84
	phyOffUp       = 96
	phyOffPosition = 108
	phyOffTypeMask = 120
	phyOffGuid     = 124
)

// Drapeaux de m_typeMask, dans l'ordre du plugin.
const (
	phyMaskBullet        = 1 << 0
	phyMaskPlay          = 1 << 1
	phyMaskInvisibleWall = 1 << 2
	phyMaskRender        = 1 << 3
)

type physiqueInstance struct {
	Index    int
	TagRef   [28]byte
	Scale    [3]float64
	Forward  [3]float64
	Left     [3]float64
	Up       [3]float64
	Position [3]float64
	TypeMask uint32
	Guid     uint32
}

func (p physiqueInstance) CollisionID() uint32 {
	return binary.LittleEndian.Uint32(p.TagRef[refOffGlobalID:])
}

// pluginStrideDe calcule le stride d'un tag-block quelconque du plugin (meme mecanisme
// que PluginInstanceStride, parametre par le nom).
func pluginStrideDe(name string) (int, error) {
	var root xnode
	if err := xml.Unmarshal(sbspPlugin, &root); err != nil {
		return 0, err
	}
	n := findPluginNode(root, name)
	if n == nil {
		return 0, fmt.Errorf("%q absent du plugin", name)
	}
	var tmp []pluginField
	return walkPlugin(*n, 0, &tmp), nil
}

func decodePhysique(tag []byte, o, idx, decalage int) physiqueInstance {
	p := physiqueInstance{
		Index:    idx,
		TypeMask: uint32(u32(tag, o+phyOffTypeMask+decalage)),
		Guid:     uint32(u32(tag, o+phyOffGuid+decalage)),
	}
	copy(p.TagRef[:], tag[o+phyOffTagRef:o+phyOffTagRef+28])
	for a := 0; a < 3; a++ {
		p.Scale[a] = f32(tag, o+phyOffScale+4*a+decalage)
		p.Forward[a] = f32(tag, o+phyOffForward+4*a+decalage)
		p.Left[a] = f32(tag, o+phyOffLeft+4*a+decalage)
		p.Up[a] = f32(tag, o+phyOffUp+4*a+decalage)
		p.Position[a] = f32(tag, o+phyOffPosition+4*a+decalage)
	}
	return p
}

// physiqueEtGuidsDuTag decode, dans le MEME tagInfo valide, le bloc physique et les
// external_guid (@0x12C) des instances de geometrie.
func physiqueEtGuidsDuTag(tag []byte, decalage int) ([]physiqueInstance, map[uint32]int, Bounds, error) {
	pluginOnce.Do(loadPlugin)
	if pluginErr != nil {
		return nil, nil, Bounds{}, pluginErr
	}
	var last error = fmt.Errorf("en-tete de tag non reconnu")
	for _, ti := range tagCandidates(tag) {
		b, _, err := boundsFromTagInfo(ti)
		if err != nil {
			last = err
			continue
		}
		target, err := ti.blockByPluginName(physiqueBlockName)
		if err != nil {
			last = err
			continue
		}
		var phys []physiqueInstance
		if target >= 0 {
			abs, size := ti.blockAbs(target)
			if abs < 0 || size < 0 || abs+size > len(ti.tag) {
				return nil, nil, Bounds{}, fmt.Errorf("bloc physique hors borne")
			}
			if size%phyStride != 0 {
				return nil, nil, Bounds{}, fmt.Errorf("taille de bloc physique %d non multiple de %d", size, phyStride)
			}
			n := size / phyStride
			phys = make([]physiqueInstance, 0, n)
			for i := 0; i < n; i++ {
				phys = append(phys, decodePhysique(ti.tag, abs+i*phyStride, i, decalage))
			}
		}
		guids := map[uint32]int{}
		if gt, err := ti.blockByPluginName(instancesBlockName); err == nil && gt >= 0 {
			abs, size := ti.blockAbs(gt)
			for i := 0; i*instanceStride+0x130 <= size; i++ {
				guids[uint32(u32(ti.tag, abs+i*instanceStride+0x12C))] = i
			}
		}
		return phys, guids, b, nil
	}
	return nil, nil, Bounds{}, last
}

// bspPrincipal extrait le tag du bsp designe par ReadModuleInstances+choisitBSP (le plus
// peuple, regle mesuree du balayage).
func bspPrincipal(t *testing.T, chemin string) ([]byte, BSPInstances) {
	t.Helper()
	bsps, err := ReadModuleInstances(chemin)
	if err != nil {
		t.Fatal(err)
	}
	bsp := ChoisitBSP(bsps, nil)
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Files("sbsp") {
		if f.Index == bsp.FileIndex {
			tag, err := m.Extract(f)
			if err != nil {
				t.Fatal(err)
			}
			return tag, bsp
		}
	}
	t.Fatalf("bsp FileIndex=%d introuvable dans %s", bsp.FileIndex, chemin)
	return nil, BSPInstances{}
}

func fractionOrthonormee(phys []physiqueInstance) float64 {
	if len(phys) == 0 {
		return 0
	}
	ok := 0
	for _, p := range phys {
		bon := true
		for _, v := range [][3]float64{p.Forward, p.Left, p.Up} {
			n := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
			if math.Abs(n-1) > 1e-3 {
				bon = false
				break
			}
		}
		if bon {
			ok++
		}
	}
	return float64(ok) / float64(len(phys))
}

// scanBaseOrthonormee cherche, pour chaque offset o de l'enregistrement, la fraction
// d'instances dont les 3 vec3 a o, o+12, o+24 sont unitaires (1e-3). Publie les offsets
// au-dessus de 90 %.
func scanBaseOrthonormee(t *testing.T, tag []byte) {
	t.Helper()
	for _, ti := range tagCandidates(tag) {
		if _, _, err := boundsFromTagInfo(ti); err != nil {
			continue
		}
		target, err := ti.blockByPluginName(physiqueBlockName)
		if err != nil || target < 0 {
			return
		}
		abs, size := ti.blockAbs(target)
		n := size / phyStride
		for o := 0; o+36 <= phyStride; o += 4 {
			ok := 0
			for i := 0; i < n; i++ {
				base := abs + i*phyStride + o
				bon := true
				for k := 0; k < 3; k++ {
					x, y, z := f32(ti.tag, base+12*k), f32(ti.tag, base+12*k+4), f32(ti.tag, base+12*k+8)
					nv := math.Sqrt(x*x + y*y + z*z)
					if math.Abs(nv-1) > 1e-3 {
						bon = false
						break
					}
				}
				if bon {
					ok++
				}
			}
			if f := float64(ok) / float64(n); f > 0.9 {
				t.Logf("scan : triple unitaire a l'offset %d (0x%x) sur %.1f %% des instances", o, o, 100*f)
			}
		}
		return
	}
}

// guidsRendus rend l'external_guid de CHAQUE instance de rendu (avec doublons — c'est la
// couverture par instance qui se mesure, pas par guid distinct).
func guidsRendus(tag []byte) []uint32 {
	for _, ti := range tagCandidates(tag) {
		if _, _, err := boundsFromTagInfo(ti); err != nil {
			continue
		}
		gt, err := ti.blockByPluginName(instancesBlockName)
		if err != nil || gt < 0 {
			return nil
		}
		abs, size := ti.blockAbs(gt)
		n := size / instanceStride
		out := make([]uint32, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, uint32(u32(ti.tag, abs+i*instanceStride+0x12C)))
		}
		return out
	}
	return nil
}

// dumpEnregistrements publie les 48 floats de quelques enregistrements physiques complets,
// pour identifier les 16 octets que le plugin ne declare pas.
func dumpEnregistrements(t *testing.T, tag []byte, nb int) {
	t.Helper()
	for _, ti := range tagCandidates(tag) {
		if _, _, err := boundsFromTagInfo(ti); err != nil {
			continue
		}
		target, err := ti.blockByPluginName(physiqueBlockName)
		if err != nil || target < 0 {
			return
		}
		abs, size := ti.blockAbs(target)
		n := size / phyStride
		if nb > n {
			nb = n
		}
		for i := 0; i < nb; i++ {
			base := abs + i*phyStride
			ligne := fmt.Sprintf("rec[%d] floats @0x00:", i)
			for o := 0; o < phyStride; o += 4 {
				if o%48 == 0 && o > 0 {
					t.Log(ligne)
					ligne = fmt.Sprintf("rec[%d] floats @0x%02x:", i, o)
				}
				ligne += fmt.Sprintf(" %9.3f", f32(ti.tag, base+o))
			}
			t.Log(ligne)
			ligne = fmt.Sprintf("rec[%d] u32 @0x70:", i)
			for o := 0x70; o < phyStride; o += 4 {
				ligne += fmt.Sprintf(" %08x", uint32(u32(ti.tag, base+o)))
			}
			t.Log(ligne)
		}
		return
	}
}

func TestSondePhysique(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}

	s, err := pluginStrideDe(physiqueBlockName)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stride du plugin pour %q : %d · stride reel constate %d (ecart %d o)",
		physiqueBlockName, s, phyStride, phyStride-s)

	cartes := []struct{ nom, chemin string }{}
	if p := moduleDuJeu(t, "pc", "ridgeline"); p != "" {
		cartes = append(cartes, struct{ nom, chemin string }{"cliffhanger", p})
	}
	if p, ok := ChercheModuleInstalle("catalyst_map"); ok {
		cartes = append(cartes, struct{ nom, chemin string }{"catalyst", p})
	}

	for _, c := range cartes {
		t.Run(c.nom, func(t *testing.T) {
			tag, bsp := bspPrincipal(t, c.chemin)
			phys, guids, _, err := physiqueEtGuidsDuTag(tag, 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s : %d instances physiques · %d instances de rendu", c.nom, len(phys), len(bsp.Instances))

			// Inventaire racine : compte declare par le root block.
			for _, ti := range tagCandidates(tag) {
				if _, _, err := boundsFromTagInfo(ti); err != nil {
					continue
				}
				infos, err := ti.describeRootBlocks()
				if err != nil {
					break
				}
				for _, info := range infos {
					if info.PluginName == physiqueBlockName {
						t.Logf("root block : count=%d · taille=%d · taille/%d=%d",
							info.Count, info.BlockSize, phyStride, info.BlockSize/phyStride)
					}
				}
				break
			}

			// SCAN : ou vit la base orthonormee (3 vec3 unitaires consecutifs) dans les
			// 192 octets ? Le vrai offset de Forward doit etre le SEUL a saturer.
			scanBaseOrthonormee(t, tag)

			// INVARIANT 1 : base orthonormee. MUTATION JOUEE : +4 doit tout casser.
			fo := fractionOrthonormee(phys)
			physDecale, _, _, _ := physiqueEtGuidsDuTag(tag, 4)
			fd := fractionOrthonormee(physDecale)
			t.Logf("orthonormalite : offset juste %.4f · offset+4 %.4f", fo, fd)
			if fo < 0.99 {
				t.Errorf("base non orthonormee a l'offset nominal (%.4f)", fo)
			}
			if fd > 0.5*fo {
				t.Errorf("la mutation +4 ne separe pas (%.4f vs %.4f) — le temoin ne teste rien", fd, fo)
			}

			// INVARIANT 2 : typeMask dans les 4 drapeaux declares.
			hist := map[uint32]int{}
			horsDrapeaux := 0
			for _, p := range phys {
				hist[p.TypeMask]++
				if p.TypeMask > 0xF {
					horsDrapeaux++
				}
			}
			t.Logf("typeMask : %d valeurs distinctes · %d hors des 4 drapeaux", len(hist), horsDrapeaux)
			for v, n := range hist {
				t.Logf("   typeMask=%04b (Bullet=%t Play=%t InvWall=%t Render=%t) : %d",
					v, v&phyMaskBullet != 0, v&phyMaskPlay != 0, v&phyMaskInvisibleWall != 0, v&phyMaskRender != 0, n)
			}

			// INVARIANT 3 : positions dans la boite du bsp (marge 10 %).
			marge := [3]float64{}
			for a := 0; a < 3; a++ {
				marge[a] = 0.1 * bsp.Bounds.Extent(a)
			}
			dedans := 0
			for _, p := range phys {
				ok := true
				for a := 0; a < 3; a++ {
					if p.Position[a] < bsp.Bounds.Min[a]-marge[a] || p.Position[a] > bsp.Bounds.Max[a]+marge[a] {
						ok = false
					}
				}
				if ok {
					dedans++
				}
			}
			t.Logf("positions dans la boite du bsp (marge 10 %%) : %d/%d", dedans, len(phys))

			// APPARIEMENT : m_guid contre external_guid des instances de rendu.
			matches := 0
			for _, p := range phys {
				if _, ok := guids[p.Guid]; ok {
					matches++
				}
			}
			t.Logf("guid : %d/%d physiques retrouves parmi les %d guids de rendu", matches, len(phys), len(guids))

			// REFERENCE DE COLLISION : groupe de tag du GlobalID a +8, resolu contre les modules.
			chemins, err := GeometrySearchPath(racine, c.chemin)
			if err != nil {
				t.Fatal(err)
			}
			idx, err := NewModuleIndex(chemins...)
			if err != nil {
				t.Fatal(err)
			}
			groupes := map[string]int{}
			nonResolus := 0
			for _, p := range phys {
				if g, _, ok := idx.Lookup(p.CollisionID()); ok {
					groupes[g]++
				} else {
					nonResolus++
				}
			}
			t.Logf("collision tag ref : groupes %v · non resolus %d", groupes, nonResolus)

			// REFERENCES DE COLLISION DISTINCTES : une poignee = primitives partagees.
			colls := map[uint32]int{}
			for _, p := range phys {
				colls[p.CollisionID()]++
			}
			t.Logf("collision : %d references distinctes", len(colls))
			if len(colls) <= 12 {
				for id, n := range colls {
					t.Logf("   coll=%08x : %d instances", id, n)
				}
			}

			// COUVERTURE COTE RENDU : combien d'instances de rendu ont un homologue
			// physique (et un homologue portant Play) ?
			physParGuid := map[uint32]uint32{} // guid -> masque cumule
			for _, p := range phys {
				physParGuid[p.Guid] |= p.TypeMask
			}
			guidRendu := guidsRendus(tag)
			avecPhys, avecPlay, guidZero := 0, 0, 0
			for _, g := range guidRendu {
				if g == 0 {
					guidZero++
				}
				if m, ok := physParGuid[g]; ok {
					avecPhys++
					if m&phyMaskPlay != 0 {
						avecPlay++
					}
				}
			}
			t.Logf("rendu : %d instances · %d avec physique · %d avec physique Play · %d guid nuls",
				len(guidRendu), avecPhys, avecPlay, guidZero)

			if os.Getenv("SONDE_DUMP") != "" {
				for i := 0; i < 4 && i < len(phys); i++ {
					p := phys[i]
					t.Logf("phys[%d] : pos=(%.2f %.2f %.2f) scale=(%.2f %.2f %.2f) mask=%04b guid=%08x coll=%08x",
						i, p.Position[0], p.Position[1], p.Position[2],
						p.Scale[0], p.Scale[1], p.Scale[2], p.TypeMask, p.Guid, p.CollisionID())
				}
				dumpEnregistrements(t, tag, 4)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// LE JUGE — le filtre par la physique confronte a la reference validee ET a
// l'oracle des 29 221 positions de joueur, sur le meme cadre que
// TestRenduCliffhanger. Cinq configurations, memes maillages, memes reglages
// valides (tous modules, tranche [-12;+28], echelle, ombres ecartees).
// ---------------------------------------------------------------------------

type itemRendu struct {
	m       *Mesh
	in      Instance
	aire    float64
	physAny bool // un homologue physique existe (guid)
	physJeu bool // ... et il porte le drapeau Play
}

func TestSondePhysiqueRendu(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref, err := cheminDepuisDepot(".ai/V7.5/dumps/carte_validee_v1.png")
	if err != nil {
		t.Skip(err)
	}
	f, err := os.Open(ref)
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = f.Close() }()
	validee, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bo := validee.Bounds()
	larg, haut := bo.Dx(), bo.Dy()
	lo := [2]float64{gateX0, gateY1 - float64(haut)*gateEchelle}
	hi := [2]float64{gateX0 + float64(larg)*gateEchelle, gateY1}

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := ChoisitBSP(bsps, ancresCliffhanger(t))

	// Le bloc physique du MEME bsp que le rendu.
	tag, bspP := bspPrincipal(t, modCarte)
	if bspP.FileIndex != bsp.FileIndex {
		t.Fatalf("bsp du rendu (%d) != bsp de la sonde (%d)", bsp.FileIndex, bspP.FileIndex)
	}
	phys, _, _, err := physiqueEtGuidsDuTag(tag, 0)
	if err != nil {
		t.Fatal(err)
	}
	physMasque := map[uint32]uint32{}
	for _, p := range phys {
		physMasque[p.Guid] |= p.TypeMask
	}
	guidRendu := guidsRendus(tag)
	if len(guidRendu) != len(bsp.Instances) {
		t.Fatalf("%d guids de rendu pour %d instances", len(guidRendu), len(bsp.Instances))
	}

	// Une seule passe d'extraction : chaque instance dessinable, avec son maillage,
	// son aire mediane et son etat physique.
	assets := map[uint32]*RuntimeGeoAsset{}
	var items []itemRendu
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tg, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tg, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			continue
		}
		masque, ok := physMasque[guidRendu[in.Index]]
		items = append(items, itemRendu{
			m: m, in: in,
			aire:    AireMedianeProjetee(m, in),
			physAny: ok,
			physJeu: ok && masque&phyMaskPlay != 0,
		})
	}
	t.Logf("%d instances dessinables (maillage resolu, hors ombres/supprimees)", len(items))

	pos := chargePositions(t)

	// `marche` > 0 applique, APRES rendu, le masque d'accessibilite depuis les ancres. Les
	// tris par instance et le masque de connexite ne sont pas de meme nature : l'un choisit
	// ce qu'on dessine, l'autre ce qui reste atteignable une fois dessine.
	configs := []struct {
		nom    string
		garde  func(itemRendu) bool
		marche float64
	}{
		{"aucun-tri", func(it itemRendu) bool { return true }, 0},
		{"grain-0.005", func(it itemRendu) bool { return it.aire <= AireMaxTriangleJouable }, 0},
		{"physique-any", func(it itemRendu) bool { return it.physAny }, 0},
		{"acces-0.85", func(it itemRendu) bool { return true }, MarcheMaxMetres},
		{"acces-0.50", func(it itemRendu) bool { return true }, 0.50},
		{"acces-1.20", func(it itemRendu) bool { return true }, 1.20},
		{"acces-et-grain", func(it itemRendu) bool { return it.aire <= AireMaxTriangleJouable }, MarcheMaxMetres},
	}
	for _, cfg := range configs {
		t.Run(cfg.nom, func(t *testing.T) {
			rendu := NewRendu(lo, hi, gateEchelle)
			rendu.Tranche(TrancheDeJeu(MedianeZ(ancresCliffhanger(t)) - AncrageDecalageSol))
			n := 0
			for _, it := range items {
				if !cfg.garde(it) {
					continue
				}
				n++
				rendu.AddMeshBorne(it.m, it.in, 0.5)
			}
			t.Logf("%s : %d/%d instances gardees", cfg.nom, n, len(items))
			if cfg.marche > 0 {
				acc := rendu.ComposanteAccessible(ancresCliffhanger(t), cfg.marche)
				garde := 0
				for _, ok := range acc {
					if ok {
						garde++
					}
				}
				rendu.Restreindre(acc)
				t.Logf("%s : composante accessible (marche %.2f m) = %d cellules", cfg.nom, cfg.marche, garde)
			}

			out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
			for py := 0; py < haut; py++ {
				for px := 0; px < larg; px++ {
					out.Set(px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
				}
				j := haut - 1 - py
				for px := 0; px < larg; px++ {
					c := color.RGBA{0, 0, 0, 255}
					if e, ok := rendu.Eclairement(px, j); ok {
						g := uint8(255 * e)
						c = color.RGBA{g, g, g, 255}
					}
					out.Set(larg+8+px, py, c)
				}
			}
			comparePixels(t, validee, bo, rendu, out, larg, haut)

			// L'ORACLE : chaque position jouee doit trouver de la matiere dans sa
			// colonne (stricte, puis avec une tolerance d'un pixel).
			dansStrict, dansVoisin, horsCadre := 0, 0, 0
			for _, p := range pos {
				i := int((p.X - rendu.Min[0]) / rendu.Cell)
				j := int((p.Y - rendu.Min[1]) / rendu.Cell)
				if i < 0 || i >= rendu.NX || j < 0 || j >= rendu.NY {
					horsCadre++
					continue
				}
				if _, ok := rendu.Eclairement(i, j); ok {
					dansStrict++
					dansVoisin++
					continue
				}
				trouve := false
				for dj := -1; dj <= 1 && !trouve; dj++ {
					for di := -1; di <= 1 && !trouve; di++ {
						ii, jj := i+di, j+dj
						if ii < 0 || ii >= rendu.NX || jj < 0 || jj >= rendu.NY {
							continue
						}
						if _, ok := rendu.Eclairement(ii, jj); ok {
							trouve = true
						}
					}
				}
				if trouve {
					dansVoisin++
				}
			}
			nb := len(pos) - horsCadre
			t.Logf("ORACLE : %d positions dans le cadre (%d hors) · dans la zone %d (%.2f %%) · a un pixel pres %d (%.2f %%)",
				nb, horsCadre, dansStrict, 100*float64(dansStrict)/float64(nb),
				dansVoisin, 100*float64(dansVoisin)/float64(nb))

			if dir := os.Getenv("SONDE_PNG_DIR"); dir != "" {
				ecritPNG(t, filepath.Join(dir, "sonde_physique_"+cfg.nom+".png"), out)
			}
		})
	}
}

// TestSondePhysiqueResolution cherche les references de collision dans TOUTE l'installation
// (132 modules), pas seulement carte + globals — ferme le point « resolvable ou non ».
func TestSondePhysiqueResolution(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	var tous []string
	_ = filepath.WalkDir(racine, func(chemin string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(chemin) == ".module" {
			tous = append(tous, chemin)
		}
		return nil
	})
	t.Logf("%d modules dans l'installation", len(tous))

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	tag, _ := bspPrincipal(t, modCarte)
	phys, _, _, err := physiqueEtGuidsDuTag(tag, 0)
	if err != nil {
		t.Fatal(err)
	}
	cherche := map[uint32]bool{}
	for _, p := range phys {
		cherche[p.CollisionID()] = true
	}
	t.Logf("%d references de collision distinctes a resoudre (cliffhanger)", len(cherche))

	groupes := map[string]int{}
	resolus := map[uint32]bool{}
	for _, chemin := range tous {
		m, err := himodule.Open(chemin)
		if err != nil {
			continue
		}
		for _, f := range m.Files("") {
			if cherche[f.GlobalID] && !resolus[f.GlobalID] {
				resolus[f.GlobalID] = true
				groupes[f.Group]++
			}
		}
	}
	t.Logf("resolus : %d/%d · groupes %v", len(resolus), len(cherche), groupes)
}
