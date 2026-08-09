package himap

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/himodule"
)

// instanceStride est la taille d'un enregistrement `instanced geometry instances`.
// Elle est RECALCULÉE depuis le plugin au chargement et confrontée à cette valeur :
// une divergence signale un plugin désaligné avec le build du jeu.
const instanceStride = 0x140

// Offsets INTERNES à l'enregistrement (dérivés du plugin, cf. PluginInstanceStride).
const (
	insOffScale     = 0x00
	insOffForward   = 0x0C
	insOffLeft      = 0x18
	insOffUp        = 0x24
	insOffPosition  = 0x30
	insOffMeshRef   = 0x3C
	insSizeMeshRef  = 0x1C
	insOffMeshIndex = 0x74
	insOffUniqueIO  = 0x76
	insOffFlags     = 0x78
	insOffAABB      = 0x7C // x0,x1,y0,y1,z0,z1 (6 floats)
	insOffSphere    = 0x94
	insOffRadius    = 0xA0
	insOffFlags2    = 0xB2
	// insOffMeshFlags : `mesh flags override`, dans la struct `material override data`.
	// Reclaimer le lit (`BspGeometryInstanceBlock.FlagsOverride`, offset 272 = 0x110) et
	// ECARTE l'instance quand le bit `mesh is custom shadow caster` est pose. Le plugin
	// confirme l'offset au champ pres : 0xF0 `per Instance Material Block` (20 o) + 3 x _6.
	insOffMeshFlags = 0x110
)

// FlagMeshCustomShadowCaster : bit `mesh is custom shadow caster` de `mesh flags override`.
// Une instance qui le porte n'est PAS de la geometrie visible — c'est un volume dedie a la
// projection d'ombre. Reclaimer la saute.
const FlagMeshCustomShadowCaster = 1 << 3

// FlagQuickDeleted : bit « is quick deleted » de flags2 — instance supprimée en
// édition, jamais rendue par le moteur.
const FlagQuickDeleted = 1 << 2

// Instance est une instance de géométrie instanciée d'un BSP.
type Instance struct {
	Index int `json:"index"`
	// Scale : champ @0x00. Réputé vestigial ; conservé pour pouvoir le CONSTATER.
	Scale     [3]float64 `json:"scale"`
	Position  [3]float64 `json:"position"`
	Forward   [3]float64 `json:"forward"`
	Left      [3]float64 `json:"left"`
	Up        [3]float64 `json:"up"`
	AABBMin   [3]float64 `json:"aabb_min"`
	AABBMax   [3]float64 `json:"aabb_max"`
	Sphere    [3]float64 `json:"sphere"`
	Radius    float64    `json:"radius"`
	MeshIndex int        `json:"mesh_index"`
	UniqueIO  int        `json:"unique_io"`
	Flags     uint16     `json:"flags"`
	Flags2    uint16     `json:"flags2"`
	// MeshFlags est `mesh flags override` (@0x110), le champ sur lequel Reclaimer ecarte
	// les projecteurs d'ombre.
	MeshFlags uint16 `json:"mesh_flags"`
	// MeshRef est le champ `Runtime geo mesh reference` brut (0x1C octets). Son layout
	// interne est INCONNU : il n'est utilisé que comme clé opaque d'identité de mesh.
	MeshRef [insSizeMeshRef]byte `json:"-"`
}

// MeshKey identifie le mesh source d'une instance (référence de tag opaque + index).
func (in Instance) MeshKey() string {
	return fmt.Sprintf("%x/%d", in.MeshRef, in.MeshIndex)
}

// QuickDeleted signale une instance marquée supprimée en édition.
func (in Instance) QuickDeleted() bool { return in.Flags2&FlagQuickDeleted != 0 }

// ProjecteurOmbre signale une instance dont le maillage n'est qu'un volume d'ombre.
func (in Instance) ProjecteurOmbre() bool { return in.MeshFlags&FlagMeshCustomShadowCaster != 0 }

// LocalToWorld applique la transformation de placement — la lecture CANONIQUE.
//
// Convention VECTEUR-LIGNE, l'echelle appliquee AVANT la base :
//
//	v = position + (lx*sx)*forward + (ly*sy)*left + (lz*sz)*up
//
// L'ECHELLE N'EST PAS VESTIGIALE, et la croire telle a fausse tous les rendus jusqu'au
// 2026-08-09. Deux faits independants l'etablissent :
//   - la base est ORTHONORMEE (temoin TestRidgelineInstancesGeometry, |v|=1 a 1e-3 sur les
//     10 357 instances) — l'echelle ne peut donc pas y etre bakee ;
//   - Reclaimer l'applique separement : `SetTransform(TransformScale, translation, rotation)`
//     avec un quaternion construit depuis la 3x3, donc renormalise.
//
// MESURE QUI TRANCHE (ridgeline, 5 817 instances, TestReclaimerEchelleInstance) : ecart a la
// boite monde declaree de l'instance — une source INDEPENDANTE du maillage — en fraction de
// sa diagonale, median 0,2238 sans l'echelle contre 0,0665 avec. 12 009 des 14 328 instances
// portent une echelle differente de 1, de -38,9 a +116,3.
func (in Instance) LocalToWorld(l [3]float64) [3]float64 {
	return in.localToWorld([3]float64{l[0] * in.Scale[0], l[1] * in.Scale[1], l[2] * in.Scale[2]})
}

// LocalToWorldSansEchelle est la lecture CONCURRENTE — celle qui ignore le champ `scale`.
// Elle n'existe que pour etre departagee par un temoin ; ne jamais s'en servir pour produire
// une carte (meme role que DequantSigne pour la dequantification).
func (in Instance) LocalToWorldSansEchelle(l [3]float64) [3]float64 { return in.localToWorld(l) }

func (in Instance) localToWorld(l [3]float64) [3]float64 {
	var w [3]float64
	for a := 0; a < 3; a++ {
		w[a] = in.Position[a] + l[0]*in.Forward[a] + l[1]*in.Left[a] + l[2]*in.Up[a]
	}
	return w
}

// BSPInstances regroupe les instances d'un tag sbsp.
type BSPInstances struct {
	FileIndex  int        `json:"file_index"`
	UncompSize int        `json:"uncomp_size"`
	Bounds     Bounds     `json:"bounds"`
	Instances  []Instance `json:"instances"`
}

// ReadModuleInstances ouvre un .module et renvoie, pour chaque tag sbsp, ses bornes
// monde et son bloc `instanced geometry instances` décodé (tags sbsp triés par taille
// décroissante). Les tags dont le bloc est vide sont conservés avec 0 instance.
func ReadModuleInstances(modulePath string) ([]BSPInstances, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, err
	}
	files := m.Files("sbsp")
	sort.Slice(files, func(i, j int) bool { return files[i].UncompSize > files[j].UncompSize })
	out := make([]BSPInstances, 0, len(files))
	var firstErr error
	for _, f := range files {
		tag, err := m.Extract(f)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sbsp#%d: %w", f.Index, err)
			}
			continue
		}
		bi, err := instancesFromTag(tag)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sbsp#%d: %w", f.Index, err)
			}
			continue
		}
		bi.FileIndex, bi.UncompSize = f.Index, f.UncompSize
		out = append(out, bi)
	}
	if len(out) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("himap: aucun tag sbsp dans %s", modulePath)
	}
	return out, nil
}

func instancesFromTag(tag []byte) (BSPInstances, error) {
	pluginOnce.Do(loadPlugin)
	if pluginErr != nil {
		return BSPInstances{}, pluginErr
	}
	cands := tagCandidates(tag)
	if len(cands) == 0 {
		return BSPInstances{}, fmt.Errorf("en-tête de tag non reconnu")
	}
	var last error
	for _, ti := range cands {
		bi, err := instancesFromTagInfo(ti)
		if err == nil {
			return bi, nil
		}
		last = err
	}
	return BSPInstances{}, last
}

func instancesFromTagInfo(ti tagInfo) (BSPInstances, error) {
	b, _, err := boundsFromTagInfo(ti)
	if err != nil {
		return BSPInstances{}, err
	}
	target, err := ti.blockByPluginName(instancesBlockName)
	if err != nil {
		return BSPInstances{}, err
	}
	res := BSPInstances{Bounds: b}
	if target < 0 { // bloc vide : légitime (certains BSP n'ont aucune instance)
		return res, nil
	}
	abs, size := ti.blockAbs(target)
	if abs < 0 || size < 0 || abs+size > len(ti.tag) {
		return BSPInstances{}, fmt.Errorf("bloc instances hors borne")
	}
	if size%instanceStride != 0 {
		return BSPInstances{}, fmt.Errorf("taille de bloc %d non multiple de %#x", size, instanceStride)
	}
	n := size / instanceStride
	res.Instances = make([]Instance, 0, n)
	for i := 0; i < n; i++ {
		res.Instances = append(res.Instances, decodeInstance(ti.tag, abs+i*instanceStride, i))
	}
	return res, nil
}

const instancesBlockName = "instanced geometry instances"

// PluginInstanceStride recalcule la taille d'un enregistrement d'instance en sommant
// les champs du plugin. Sert de contrôle : elle doit valoir instanceStride.
func PluginInstanceStride() (int, error) {
	var root xnode
	if err := xml.Unmarshal(sbspPlugin, &root); err != nil {
		return 0, err
	}
	n := findPluginNode(root, instancesBlockName)
	if n == nil {
		return 0, fmt.Errorf("himap: %q absent du plugin", instancesBlockName)
	}
	var tmp []pluginField
	return walkPlugin(*n, 0, &tmp), nil
}

func findPluginNode(n xnode, name string) *xnode {
	for i := range n.Nodes {
		c := n.Nodes[i]
		if c.XMLName.Local == "_40" && c.V == name {
			return &n.Nodes[i]
		}
		if found := findPluginNode(c, name); found != nil {
			return found
		}
	}
	return nil
}

func decodeInstance(tag []byte, o, idx int) Instance {
	in := Instance{
		Index:     idx,
		MeshIndex: int(u16(tag, o+insOffMeshIndex)),
		UniqueIO:  int(u16(tag, o+insOffUniqueIO)),
		Flags:     u16(tag, o+insOffFlags),
		Flags2:    u16(tag, o+insOffFlags2),
		MeshFlags: u16(tag, o+insOffMeshFlags),
		Radius:    f32(tag, o+insOffRadius),
	}
	copy(in.MeshRef[:], tag[o+insOffMeshRef:o+insOffMeshRef+insSizeMeshRef])
	for a := 0; a < 3; a++ {
		in.Scale[a] = f32(tag, o+insOffScale+4*a)
		in.Forward[a] = f32(tag, o+insOffForward+4*a)
		in.Left[a] = f32(tag, o+insOffLeft+4*a)
		in.Up[a] = f32(tag, o+insOffUp+4*a)
		in.Position[a] = f32(tag, o+insOffPosition+4*a)
		in.Sphere[a] = f32(tag, o+insOffSphere+4*a)
		in.AABBMin[a] = f32(tag, o+insOffAABB+8*a)
		in.AABBMax[a] = f32(tag, o+insOffAABB+8*a+4)
	}
	return in
}

// blockByPluginName résout, PAR RANG (jamais par offset câblé), l'index du data-block
// du champ tag-block `name` du plugin. Renvoie -1 si le bloc est vide.
func (ti tagInfo) blockByPluginName(name string) (int, error) {
	rank := -1
	for i, pb := range pluginBlocks {
		if pb.name == name {
			rank = i
			break
		}
	}
	if rank < 0 {
		return 0, fmt.Errorf("champ %q absent du plugin", name)
	}
	offs, targets, err := ti.rootBlockRefs()
	if err != nil {
		return 0, err
	}
	if rank >= len(offs) {
		return 0, fmt.Errorf("rang %d hors des %d refs du tag", rank, len(offs))
	}
	// Les écarts entre tag-blocks doivent être conservés jusqu'au rang visé, sinon
	// le préfixe du root block a changé de forme et l'appariement serait faux.
	for i := 0; i < rank; i++ {
		if pluginBlocks[i+1].off-pluginBlocks[i].off != offs[i+1]-offs[i] {
			return 0, fmt.Errorf("écart de tag-blocks non conservé au rang %d", i)
		}
	}
	return targets[rank], nil
}

// rootBlockRefs renvoie les TagBlocks (type 1) référencés depuis le root block,
// triés par field_offset, avec leur data-block cible.
func (ti tagInfo) rootBlockRefs() ([]int, []int, error) {
	tag := ti.tag
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(tag, b+0x10) == 0 {
			rootBlock = i32(tag, b+0x14)
			break
		}
	}
	if rootBlock < 0 {
		return nil, nil, fmt.Errorf("root block introuvable")
	}
	type ref struct{ off, target int }
	var refs []ref
	seen := map[int]bool{}
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(tag, b+0x10) != 1 || i32(tag, b+0x18) != rootBlock {
			continue
		}
		fo := u32(tag, b+0x1C)
		if fo < 0 || fo >= 0x10000 || seen[fo] {
			continue
		}
		seen[fo] = true
		refs = append(refs, ref{fo, i32(tag, b+0x14)})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].off < refs[j].off })
	offs := make([]int, len(refs))
	targets := make([]int, len(refs))
	for i, r := range refs {
		offs[i], targets[i] = r.off, r.target
	}
	return offs, targets, nil
}

// RootBlockInfo décrit un champ tag-block du root block d'un sbsp : nom apparié par
// rang au plugin, offset réel, data-block cible, taille du bloc et compteur lu dans
// la donnée (TagBlock = 16 o de pointeur + u32 count à +0x10).
type RootBlockInfo struct {
	Rank        int
	PluginName  string
	FieldOffset int
	Target      int
	BlockSize   int
	Count       int
}

// DescribeRootBlocks renvoie l'inventaire des tag-blocks racine du plus gros sbsp d'un
// module. Outil de diagnostic : sert à constater ce qu'un build contient réellement.
func DescribeRootBlocks(modulePath string) ([]RootBlockInfo, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, err
	}
	files := m.Files("sbsp")
	if len(files) == 0 {
		return nil, fmt.Errorf("himap: aucun tag sbsp dans %s", modulePath)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].UncompSize > files[j].UncompSize })
	tag, err := m.Extract(files[0])
	if err != nil {
		return nil, err
	}
	pluginOnce.Do(loadPlugin)
	if pluginErr != nil {
		return nil, pluginErr
	}
	for _, ti := range tagCandidates(tag) {
		if _, _, err := boundsFromTagInfo(ti); err != nil {
			continue
		}
		return ti.describeRootBlocks()
	}
	return nil, fmt.Errorf("himap: en-tête de tag non reconnu")
}

func (ti tagInfo) describeRootBlocks() ([]RootBlockInfo, error) {
	offs, targets, err := ti.rootBlockRefs()
	if err != nil {
		return nil, err
	}
	rootBlock, err := ti.rootBlockIndex()
	if err != nil {
		return nil, err
	}
	rootAbs, rootSize := ti.blockAbs(rootBlock)
	out := make([]RootBlockInfo, 0, len(offs))
	for i, fo := range offs {
		info := RootBlockInfo{Rank: i, FieldOffset: fo, Target: targets[i], BlockSize: -1, Count: -1}
		if i < len(pluginBlocks) {
			info.PluginName = pluginBlocks[i].name
		}
		if targets[i] >= 0 {
			_, info.BlockSize = ti.blockAbs(targets[i])
		}
		if c := rootAbs + fo + 0x10; fo+0x14 <= rootSize && c+4 <= len(ti.tag) {
			info.Count = u32(ti.tag, c)
		}
		out = append(out, info)
	}
	return out, nil
}

func (ti tagInfo) rootBlockIndex() (int, error) {
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(ti.tag, b+0x10) == 0 {
			return i32(ti.tag, b+0x14), nil
		}
	}
	return 0, fmt.Errorf("root block introuvable")
}

// Footprint2D est l'emprise au sol d'une instance, projetée sur le plan (x, y).
type Footprint2D struct {
	MinX, MinY, MaxX, MaxY float64
	TopZ                   float64 // altitude de la face supérieure (z max)
	BottomZ                float64
}

// Footprint projette l'AABB monde de l'instance sur le plan horizontal.
func (in Instance) Footprint() Footprint2D {
	return Footprint2D{
		MinX: in.AABBMin[0], MinY: in.AABBMin[1],
		MaxX: in.AABBMax[0], MaxY: in.AABBMax[1],
		TopZ: in.AABBMax[2], BottomZ: in.AABBMin[2],
	}
}

// Area renvoie l'aire au sol de l'emprise.
func (f Footprint2D) Area() float64 {
	w, h := f.MaxX-f.MinX, f.MaxY-f.MinY
	if w <= 0 || h <= 0 || math.IsNaN(w) || math.IsNaN(h) {
		return 0
	}
	return w * h
}
