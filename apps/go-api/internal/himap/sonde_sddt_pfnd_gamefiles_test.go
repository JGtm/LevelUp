package himap

// SONDE D'INVESTIGATION (2026-08-10, non destinee au commit) — la zone jouable
// vit-elle dans `sddt` (structure design : kill volumes / soft ceilings) ou dans
// `pfnd` (pathfinding : navmesh) ? Les deux tags vivent dans la variante any/ du
// module de carte.
//
// Aucun plugin n'existe pour ces tags et Reclaimer ne les implemente pas (verifie
// par grep sur les 7 refs) : toute la lecture passe par la struct-table generique
// (tagCandidates + blockAbs), et chaque interpretation doit etre etablie par un
// invariant JOUE (mutation qui casse), jamais supposee.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// moduleAny compose le chemin du module any/ d'une carte (memes noms de dossier
// que pc/ : ridgeline, catalyst).
func moduleAny(t *testing.T, carte string) string {
	t.Helper()
	dir, err := LevelsDir("any")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	p := filepath.Join(dir, carte, carte+"-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("module absent (%s)", p)
	}
	return p
}

// extraitTagsGroupe extrait tous les tags d'un groupe d'un module.
func extraitTagsGroupe(t *testing.T, chemin, groupe string) map[int][]byte {
	t.Helper()
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int][]byte{}
	for _, f := range m.Files(groupe) {
		tag, err := m.Extract(f)
		if err != nil {
			t.Logf("%s#%d : extraction impossible : %v", groupe, f.Index, err)
			continue
		}
		out[f.Index] = tag
	}
	return out
}

// meilleurTagInfo choisit le candidat d'en-tete dont la struct-table se valide
// (root block trouvable, tous les data-blocks dans les bornes du tag).
func meilleurTagInfo(tag []byte) (tagInfo, error) {
	var last error = fmt.Errorf("aucun candidat d'en-tete")
	for _, ti := range tagCandidates(tag) {
		if _, err := ti.rootBlockIndex(); err != nil {
			last = err
			continue
		}
		ok := true
		for i := 0; i < ti.dataBlocks; i++ {
			abs, size := ti.blockAbs(i)
			if abs < 0 || size < 0 || abs+size > len(ti.tag) {
				ok = false
				break
			}
		}
		if !ok {
			last = fmt.Errorf("data-block hors bornes")
			continue
		}
		return ti, nil
	}
	return tagInfo{}, last
}

// lienBloc est une reference TagBlock (type 1) de la struct-table : le bloc
// `owner` porte a `fieldOff` un champ pointant le bloc `target`.
type lienBloc struct {
	structIdx int
	owner     int
	fieldOff  int
	target    int
}

func liensBlocs(ti tagInfo) []lienBloc {
	var out []lienBloc
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(ti.tag, b+0x10) != 1 {
			continue
		}
		out = append(out, lienBloc{
			structIdx: i,
			owner:     i32(ti.tag, b+0x18),
			fieldOff:  u32(ti.tag, b+0x1C),
			target:    i32(ti.tag, b+0x14),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].owner != out[j].owner {
			return out[i].owner < out[j].owner
		}
		return out[i].fieldOff < out[j].fieldOff
	})
	return out
}

// compteChamp lit le count du TagBlock a owner+fieldOff (+0x10 comme dans
// describeRootBlocks).
func compteChamp(ti tagInfo, l lienBloc) int {
	abs, size := ti.blockAbs(l.owner)
	if l.fieldOff+0x14 > size || abs+l.fieldOff+0x14 > len(ti.tag) {
		return -1
	}
	return u32(ti.tag, abs+l.fieldOff+0x10)
}

// TestSondeSddtPfndStructure dresse l'arbre des blocs des deux tags sur les deux
// cartes : qui pointe qui, tailles, comptes, strides.
func TestSondeSddtPfndStructure(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		chemin := moduleAny(t, carte)
		for _, groupe := range []string{"sddt", "pfnd"} {
			t.Run(carte+"/"+groupe, func(t *testing.T) {
				tags := extraitTagsGroupe(t, chemin, groupe)
				if len(tags) == 0 {
					t.Fatalf("aucun tag %s dans %s", groupe, chemin)
				}
				for idx, tag := range tags {
					ti, err := meilleurTagInfo(tag)
					if err != nil {
						t.Errorf("%s#%d (%d o) : %v", groupe, idx, len(tag), err)
						continue
					}
					t.Logf("%s#%d : %d o · header=%d deps=%d dataBlocks=%d structs=%d",
						groupe, idx, len(tag), ti.headerSize, ti.deps, ti.dataBlocks, ti.structs)
					liens := liensBlocs(ti)
					root, _ := ti.rootBlockIndex()
					rootAbs, rootSize := ti.blockAbs(root)
					t.Logf("  root block = data-block %d @%d, %d o", root, rootAbs, rootSize)
					// L'arbre complet est verbeux (475 blocs sur ridgeline) : on
					// agrege par (owner, fieldOff) les liens de meme forme.
					type forme struct{ owner, fieldOff, stride int }
					agg := map[forme][]lienBloc{}
					for _, l := range liens {
						_, size := ti.blockAbs(l.target)
						c := compteChamp(ti, l)
						stride := -1
						if c > 0 && size%c == 0 {
							stride = size / c
						}
						f := forme{l.owner, l.fieldOff, stride}
						agg[f] = append(agg[f], l)
					}
					var formes []forme
					for f := range agg {
						formes = append(formes, f)
					}
					sort.Slice(formes, func(i, j int) bool {
						if formes[i].owner != formes[j].owner {
							return formes[i].owner < formes[j].owner
						}
						return formes[i].fieldOff < formes[j].fieldOff
					})
					for _, f := range formes {
						ls := agg[f]
						total, cmax := 0, 0
						for _, l := range ls {
							_, size := ti.blockAbs(l.target)
							total += size
							if c := compteChamp(ti, l); c > cmax {
								cmax = c
							}
						}
						t.Logf("  owner=%4d off=0x%04x -> %3d bloc(s), stride=%3d, taille cumulee=%7d, count max=%d",
							f.owner, f.fieldOff, len(ls), f.stride, total, cmax)
					}
					// Les liens d'un MEME owner a des fieldOff differents forment le
					// motif « petit bloc + grand bloc » : detailler les 12 premiers.
					n := 0
					for _, l := range liens {
						if l.owner == root {
							continue
						}
						abs, size := ti.blockAbs(l.target)
						c := compteChamp(ti, l)
						t.Logf("  lien: owner=%d off=0x%x -> bloc %d @%d, %d o, count=%d",
							l.owner, l.fieldOff, l.target, abs, size, c)
						n++
						if n >= 12 {
							break
						}
					}
				}
			})
		}
	}
}

// fractionNormalesUnitaires mesure, sur un bloc de records de 16 o, la part dont
// les 3 premiers floats (a l'offset donne) forment un vecteur unitaire — le
// temoin de l'hypothese « plans (a,b,c,d) ». NaN compte comme NON unitaire (piege
// paye par la sonde physique).
func fractionNormalesUnitaires(tag []byte, abs, size, stride, off int) (float64, int) {
	if stride <= 0 || size < stride {
		return 0, 0
	}
	n := size / stride
	ok := 0
	for i := 0; i < n; i++ {
		b := abs + i*stride + off
		if b+12 > len(tag) {
			return 0, 0
		}
		x, y, z := f32(tag, b), f32(tag, b+4), f32(tag, b+8)
		nv := math.Sqrt(x*x + y*y + z*z)
		if !math.IsNaN(nv) && math.Abs(nv-1) <= 1e-3 {
			ok++
		}
	}
	return float64(ok) / float64(n), n
}

// TestSondeSddtPlans joue l'hypothese « les blocs 16 o de sddt sont des plans » :
// normales unitaires a l'offset 0, mutation +4 qui doit tout casser, et dump des
// premiers enregistrements pour lecture humaine.
func TestSondeSddtPlans(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		chemin := moduleAny(t, carte)
		t.Run(carte, func(t *testing.T) {
			tags := extraitTagsGroupe(t, chemin, "sddt")
			for idx, tag := range tags {
				ti, err := meilleurTagInfo(tag)
				if err != nil {
					continue
				}
				// Tous les blocs de stride 16 (via leurs liens, pour connaitre le count).
				tot16, totRec := 0, 0
				histOK := 0.0
				histKO := 0.0
				var exemples []lienBloc
				for _, l := range liensBlocs(ti) {
					abs, size := ti.blockAbs(l.target)
					c := compteChamp(ti, l)
					if c <= 0 || size != c*16 {
						continue
					}
					tot16++
					totRec += c
					f0, _ := fractionNormalesUnitaires(ti.tag, abs, size, 16, 0)
					f4, _ := fractionNormalesUnitaires(ti.tag, abs, size, 16, 4)
					histOK += f0 * float64(c)
					histKO += f4 * float64(c)
					if len(exemples) < 3 {
						exemples = append(exemples, l)
					}
				}
				if totRec == 0 {
					t.Logf("%s sddt#%d : aucun bloc de stride 16", carte, idx)
					continue
				}
				t.Logf("%s sddt#%d : %d blocs de stride 16, %d records · normales unitaires @0 : %.1f %% · @+4 (mutation) : %.1f %%",
					carte, idx, tot16, totRec, 100*histOK/float64(totRec), 100*histKO/float64(totRec))
				for _, l := range exemples {
					abs, size := ti.blockAbs(l.target)
					c := size / 16
					if c > 6 {
						c = 6
					}
					for i := 0; i < c; i++ {
						b := abs + i*16
						t.Logf("  bloc %d rec[%d] : %9.4f %9.4f %9.4f %9.4f", l.target, i,
							f32(ti.tag, b), f32(ti.tag, b+4), f32(ti.tag, b+8), f32(ti.tag, b+12))
					}
				}
			}
		})
	}
}

// blocParTaille retrouve le premier lien dont le bloc cible a la taille et le
// stride donnes (outil de navigation quand il n'y a pas de plugin).
func liensVers(ti tagInfo, stride int) []lienBloc {
	var out []lienBloc
	for _, l := range liensBlocs(ti) {
		_, size := ti.blockAbs(l.target)
		c := compteChamp(ti, l)
		if c > 0 && size == c*stride {
			out = append(out, l)
		}
	}
	return out
}

func dumpRecord(t *testing.T, tag []byte, base, taille int, prefixe string) {
	t.Helper()
	ligne := prefixe + " f32:"
	for o := 0; o < taille; o += 4 {
		ligne += fmt.Sprintf(" %10.3f", f32(tag, base+o))
	}
	t.Log(ligne)
	ligne = prefixe + " u32:"
	for o := 0; o < taille; o += 4 {
		ligne += fmt.Sprintf(" %08x", uint32(u32(tag, base+o)))
	}
	t.Log(ligne)
}

// TestSondeSddtDump lit les enregistrements porteurs : records de 100 o
// (volumes), de 68 o (groupes ?), chaines, et triangles — puis verifie que les
// sommets des triangles satisfont les equations de plans du meme volume.
func TestSondeSddtDump(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		chemin := moduleAny(t, carte)
		t.Run(carte, func(t *testing.T) {
			tags := extraitTagsGroupe(t, chemin, "sddt")
			for idx, tag := range tags {
				ti, err := meilleurTagInfo(tag)
				if err != nil {
					continue
				}
				_ = idx
				// Chaines : tous les blocs de stride 1.
				for _, l := range liensVers(ti, 1) {
					abs, size := ti.blockAbs(l.target)
					if size > 400 {
						size = 400
					}
					brut := make([]byte, size)
					copy(brut, ti.tag[abs:abs+size])
					for i, c := range brut {
						if c == 0 {
							brut[i] = '|'
						} else if c < 0x20 || c > 0x7e {
							brut[i] = '.'
						}
					}
					t.Logf("chaines bloc %d (owner=%d off=0x%x, %d o) : %s", l.target, l.owner, l.fieldOff, size, string(brut))
				}
				// Records de 68 o.
				for _, l := range liensVers(ti, 68) {
					abs, size := ti.blockAbs(l.target)
					n := size / 68
					t.Logf("bloc 68 o : %d records (owner=%d off=0x%x)", n, l.owner, l.fieldOff)
					for i := 0; i < n && i < 4; i++ {
						dumpRecord(t, ti.tag, abs+i*68, 68, fmt.Sprintf("  rec68[%d]", i))
					}
				}
				// Records de 100 o : le corps hors TagBlocks (0x00-0x23 et 0x4c-0x63).
				for _, l := range liensVers(ti, 100) {
					abs, size := ti.blockAbs(l.target)
					n := size / 100
					t.Logf("bloc 100 o : %d records (owner=%d off=0x%x)", n, l.owner, l.fieldOff)
					for i := 0; i < n && i < 4; i++ {
						dumpRecord(t, ti.tag, abs+i*100, 100, fmt.Sprintf("  rec100[%d]", i))
					}
				}
			}
		})
	}
}

// TestSondeSddtTriangles verifie la coherence interne plans <-> triangles de
// chaque volume, et publie l'emprise des sommets contre la boite monde du bsp.
func TestSondeSddtTriangles(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	_ = racine
	chemin := moduleAny(t, "ridgeline")
	tags := extraitTagsGroupe(t, chemin, "sddt")
	// Boite monde du bsp de rendu (variante pc).
	bsps, err := ReadModuleInstances(moduleDuJeu(t, "pc", "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	bsp := choisitBSP(bsps, nil)
	t.Logf("boite bsp : min=(%.1f %.1f %.1f) max=(%.1f %.1f %.1f)",
		bsp.Bounds.Min[0], bsp.Bounds.Min[1], bsp.Bounds.Min[2],
		bsp.Bounds.Max[0], bsp.Bounds.Max[1], bsp.Bounds.Max[2])
	for idx, tag := range tags {
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			t.Fatalf("sddt#%d : %v", idx, err)
		}
		// Apparier chaque volume (record 100 o) a ses deux enfants par fieldOff.
		type volume struct {
			plans [][4]float64
			tris  [][9]float64
		}
		var volumes []volume
		l100 := liensVers(ti, 100)
		if len(l100) != 1 {
			t.Logf("sddt#%d : %d blocs de 100 o", idx, len(l100))
			continue
		}
		liens := liensBlocs(ti)
		parOwnerOff := map[[2]int]lienBloc{}
		for _, l := range liens {
			parOwnerOff[[2]int{l.owner, l.fieldOff}] = l
		}
		volBloc := l100[0].target
		_, sizeV := ti.blockAbs(volBloc)
		nV := sizeV / 100
		dansBoite, horsBoite := 0, 0
		maxResidu := 0.0
		for i := 0; i < nV; i++ {
			v := volume{}
			if lp, ok := parOwnerOff[[2]int{volBloc, i*100 + 0x24}]; ok {
				abs, size := ti.blockAbs(lp.target)
				for k := 0; k+16 <= size; k += 16 {
					v.plans = append(v.plans, [4]float64{
						f32(ti.tag, abs+k), f32(ti.tag, abs+k+4), f32(ti.tag, abs+k+8), f32(ti.tag, abs+k+12)})
				}
			}
			if lt, ok := parOwnerOff[[2]int{volBloc, i*100 + 0x38}]; ok {
				abs, size := ti.blockAbs(lt.target)
				for k := 0; k+36 <= size; k += 36 {
					var tri [9]float64
					for a := 0; a < 9; a++ {
						tri[a] = f32(ti.tag, abs+k+4*a)
					}
					v.tris = append(v.tris, tri)
				}
			}
			// Sommets dans la boite ? Et residu n.p-d des sommets contre le plan
			// le plus proche (les sommets d'un prisme vivent sur ses plans).
			for _, tri := range v.tris {
				for s := 0; s < 3; s++ {
					p := [3]float64{tri[3*s], tri[3*s+1], tri[3*s+2]}
					ok := true
					for a := 0; a < 3; a++ {
						if p[a] < bsp.Bounds.Min[a]-10 || p[a] > bsp.Bounds.Max[a]+10 {
							ok = false
						}
					}
					if ok {
						dansBoite++
					} else {
						horsBoite++
					}
					best := math.Inf(1)
					for _, pl := range v.plans {
						r := math.Abs(pl[0]*p[0] + pl[1]*p[1] + pl[2]*p[2] - pl[3])
						if r < best {
							best = r
						}
					}
					if best > maxResidu && !math.IsInf(best, 1) {
						maxResidu = best
					}
				}
			}
			volumes = append(volumes, v)
		}
		nbP, nbT := 0, 0
		for _, v := range volumes {
			nbP += len(v.plans)
			nbT += len(v.tris)
		}
		t.Logf("sddt#%d : %d volumes · %d plans · %d triangles · sommets dans la boite %d, hors %d · residu max sommet->plan le plus proche %.4f",
			idx, len(volumes), nbP, nbT, dansBoite, horsBoite, maxResidu)
	}
}

// TestSondePfndDump lit les 96 o de pfnd et les champs racine, sur les deux cartes.
func TestSondePfndDump(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		chemin := moduleAny(t, carte)
		t.Run(carte, func(t *testing.T) {
			tags := extraitTagsGroupe(t, chemin, "pfnd")
			for idx, tag := range tags {
				ti, err := meilleurTagInfo(tag)
				if err != nil {
					continue
				}
				for _, stride := range []int{24, 36, 60, 96} {
					for _, l := range liensVers(ti, stride) {
						abs, size := ti.blockAbs(l.target)
						n := size / stride
						t.Logf("pfnd#%d bloc %d o : %d records (owner=%d off=0x%x)", idx, stride, n, l.owner, l.fieldOff)
						for i := 0; i < n && i < 5; i++ {
							dumpRecord(t, ti.tag, abs+i*stride, stride, fmt.Sprintf("  rec%d[%d]", stride, i))
						}
					}
				}
			}
		})
	}
}

// sddtVolume est un volume du bloc 100 o : type presume, AABB propre, plans,
// triangles.
type sddtVolume struct {
	TypeU16 uint16
	F04     float64
	AABBMin [3]float64
	AABBMax [3]float64
	Plans   [][4]float64
	Tris    [][3][3]float64
}

// sddtCeiling est un triangle-frontiere du bloc 68 o.
type sddtCeiling struct {
	Normal   [3]float64
	D        float64
	Centroid [3]float64
	Radius   float64
	V        [3][3]float64
}

func litSddt(t *testing.T, chemin string) ([]sddtVolume, []sddtCeiling) {
	t.Helper()
	tags := extraitTagsGroupe(t, chemin, "sddt")
	for _, tag := range tags {
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		liens := liensBlocs(ti)
		parOwnerOff := map[[2]int]lienBloc{}
		for _, l := range liens {
			parOwnerOff[[2]int{l.owner, l.fieldOff}] = l
		}
		var vols []sddtVolume
		for _, l := range liensVers(ti, 100) {
			abs, size := ti.blockAbs(l.target)
			n := size / 100
			for i := 0; i < n; i++ {
				base := abs + i*100
				v := sddtVolume{
					TypeU16: u16(ti.tag, base),
					F04:     f32(ti.tag, base+4),
				}
				for a := 0; a < 3; a++ {
					v.AABBMin[a] = f32(ti.tag, base+0x4c+8*a)
					v.AABBMax[a] = f32(ti.tag, base+0x4c+8*a+4)
				}
				if lp, ok := parOwnerOff[[2]int{l.target, i*100 + 0x24}]; ok {
					pabs, psize := ti.blockAbs(lp.target)
					for k := 0; k+16 <= psize; k += 16 {
						v.Plans = append(v.Plans, [4]float64{
							f32(ti.tag, pabs+k), f32(ti.tag, pabs+k+4), f32(ti.tag, pabs+k+8), f32(ti.tag, pabs+k+12)})
					}
				}
				if lt, ok := parOwnerOff[[2]int{l.target, i*100 + 0x38}]; ok {
					tabs, tsize := ti.blockAbs(lt.target)
					for k := 0; k+36 <= tsize; k += 36 {
						var tri [3][3]float64
						for s := 0; s < 3; s++ {
							for a := 0; a < 3; a++ {
								tri[s][a] = f32(ti.tag, tabs+k+12*s+4*a)
							}
						}
						v.Tris = append(v.Tris, tri)
					}
				}
				vols = append(vols, v)
			}
		}
		var ceils []sddtCeiling
		for _, l := range liensVers(ti, 68) {
			abs, size := ti.blockAbs(l.target)
			n := size / 68
			for i := 0; i < n; i++ {
				base := abs + i*68
				c := sddtCeiling{D: f32(ti.tag, base+12), Radius: f32(ti.tag, base+28)}
				for a := 0; a < 3; a++ {
					c.Normal[a] = f32(ti.tag, base+4*a)
					c.Centroid[a] = f32(ti.tag, base+16+4*a)
				}
				for s := 0; s < 3; s++ {
					for a := 0; a < 3; a++ {
						c.V[s][a] = f32(ti.tag, base+32+12*s+4*a)
					}
				}
				ceils = append(ceils, c)
			}
		}
		if len(vols) > 0 || len(ceils) > 0 {
			return vols, ceils
		}
	}
	return nil, nil
}

// TestSondeSddtDistribution : histogramme des types, emprise des AABB, verif du
// temoin centroide=moyenne des sommets sur les triangles-frontieres.
func TestSondeSddtDistribution(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		chemin := moduleAny(t, carte)
		t.Run(carte, func(t *testing.T) {
			vols, ceils := litSddt(t, chemin)
			t.Logf("%s : %d volumes, %d triangles-frontieres", carte, len(vols), len(ceils))
			types := map[uint16]int{}
			for _, v := range vols {
				types[v.TypeU16]++
			}
			t.Logf("types u16@0 : %v", types)
			if len(vols) > 0 {
				min := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
				max := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
				var dims [3][]float64
				for _, v := range vols {
					for a := 0; a < 3; a++ {
						if v.AABBMin[a] < min[a] {
							min[a] = v.AABBMin[a]
						}
						if v.AABBMax[a] > max[a] {
							max[a] = v.AABBMax[a]
						}
						dims[a] = append(dims[a], v.AABBMax[a]-v.AABBMin[a])
					}
				}
				for a := 0; a < 3; a++ {
					sort.Float64s(dims[a])
				}
				t.Logf("emprise des %d AABB : min=(%.1f %.1f %.1f) max=(%.1f %.1f %.1f)",
					len(vols), min[0], min[1], min[2], max[0], max[1], max[2])
				t.Logf("dimension mediane des volumes : (%.2f %.2f %.2f) · p90 (%.2f %.2f %.2f) · max (%.2f %.2f %.2f)",
					dims[0][len(vols)/2], dims[1][len(vols)/2], dims[2][len(vols)/2],
					dims[0][len(vols)*9/10], dims[1][len(vols)*9/10], dims[2][len(vols)*9/10],
					dims[0][len(vols)-1], dims[1][len(vols)-1], dims[2][len(vols)-1])
			}
			// Temoin arithmetique du decodage 68 o : centroide == moyenne des sommets.
			maxEcart := 0.0
			for _, c := range ceils {
				for a := 0; a < 3; a++ {
					m := (c.V[0][a] + c.V[1][a] + c.V[2][a]) / 3
					if e := math.Abs(m - c.Centroid[a]); e > maxEcart {
						maxEcart = e
					}
				}
			}
			t.Logf("triangles-frontieres : ecart max centroide vs moyenne des sommets = %.4f", maxEcart)
			for i, c := range ceils {
				t.Logf("  frontiere[%d] : n=(%.2f %.2f %.2f) d=%.2f sommets %v", i,
					c.Normal[0], c.Normal[1], c.Normal[2], c.D, c.V)
			}
		})
	}
}
