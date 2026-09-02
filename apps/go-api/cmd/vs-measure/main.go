// Command vs-measure — OUTIL JETABLE de diagnostic (chantier vehicules v7.5, rework
// Warthog/Gungoose 2026-09-01). Mesure la boite englobante (metres) et le centroide (x,y,z)
// d'un render_model (`mode`), en repere LOCAL du modele (X=longitudinal ; Y=lateral ; Z=haut —
// convention rectifiee objet_isole.go). Le sens de X depend du modele : Scorpion +X=ARRIERE,
// famille Warthog (chassis 0x561f2ca7, Razorback) +X=AVANT — corrige par l'utilisateur le
// 2026-09-02 (cf. plateau.go). Ne modifie AUCUN code partage :
// il ne fait qu'importer internal/himap. A SUPPRIMER apres usage (pas de code mort).
//
// Usage :
//
//	vs-measure -modules="pc:globals-rtx-new.module,globals-rtx-new.module" -modes=0x561f2ca7[,...]
//	vs-measure -modules=... -vehis=0x00002705[,...]   (resout vehi->hlmt->mode puis mesure)
//	-secs : ajoute le detail par section (nv, dx,dy,dz, cX,cY,cZ)
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
)

// vocabNoeuds : noms candidats de noeuds de squelette a resoudre par murmur3 (mapvar.LabelHash),
// pour identifier un noeud d'attache d'arme sans son nom (StringId brut sinon).
var vocabNoeuds = []string{
	"root", "b_root", "pelvis", "body", "hull", "chassis", "frame", "base",
	"gunner", "warthog_gunner", "warthog_gunner_gauss", "warthog_gunner_rocket",
	"driver", "passenger", "seat", "seat_gunner", "turret", "turret_g", "turret_base",
	"turret_mount", "weapon", "weapon_mount", "primary", "gun", "gun_mount", "mount",
	"chaingun", "laag", "pedestal", "hardpoint", "attach", "attach_weapon", "w_gun",
	"warthog_g", "warthog_b_g", "mongoose_g", "gungoose", "wargoose", "spine", "hips",
	"turret_pitch", "turret_yaw", "pitch", "yaw", "barrel", "muzzle", "socket",
}

func main() {
	// Sous-commande `armes` (planche-contact des armes arriere Warthog) : cf. armes.go.
	if len(os.Args) > 1 && os.Args[1] == "armes" {
		armesMain(os.Args[2:])
		return
	}
	// Sous-commande `plateau` (arme posee au centre du plateau arriere) : cf. plateau.go.
	if len(os.Args) > 1 && os.Args[1] == "plateau" {
		plateauMain(os.Args[2:])
		return
	}
	fs := flag.NewFlagSet("vs-measure", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir (basenames ou variant:basename, virgule)")
	variant := fs.String("variant", "any", "variante deploy: any|pc|ds")
	modes := fs.String("modes", "", "GlobalID de modes (hex, virgule) a mesurer directement")
	vehis := fs.String("vehis", "", "GlobalID de vehi (hex, virgule) a resoudre (vehi->mode) puis mesurer")
	secs := fs.Bool("secs", false, "detail par section")
	nodes := fs.String("nodes", "", "GlobalID de mode(s) (hex, virgule) dont dumper le squelette (noeuds + transformee model-space)")
	_ = fs.Parse(os.Args[1:])

	chemins, err := cheminsModules(*variant, listeModules(*mods))
	must(err)
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	must(err)

	var modeIDs []uint32
	for _, s := range splitHex(*modes) {
		modeIDs = append(modeIDs, s)
	}
	for _, vid := range splitHex(*vehis) {
		tag, err := idx.Extract(vid)
		if err != nil {
			fmt.Printf("vehi %#08x extraction KO: %v\n", vid, err)
			continue
		}
		mid, grp, ok := himap.RefModeleVehicule(context.Background(), idx, tag)
		if !ok {
			fmt.Printf("vehi %#08x : SANS MODELE\n", vid)
			continue
		}
		fmt.Printf("vehi %#08x -> %s %#08x\n", vid, grp, mid)
		modeIDs = append(modeIDs, mid)
	}
	for _, id := range modeIDs {
		mesureMode(idx, id, *secs)
	}
	for _, id := range splitHex(*nodes) {
		dumpNoeuds(idx, id)
	}
}

// dumpNoeuds charge un `mode`, parse son squelette et imprime chaque noeud (nom resolu, parent,
// echelle locale, transformee model-space) ; signale les echelles != 1 et les noms d'arme.
func dumpNoeuds(idx *himap.ModuleIndex, id uint32) {
	tag, _, err := idx.ExtractWithResources(id)
	if err != nil {
		fmt.Printf("mode %#08x extraction KO: %v\n", id, err)
		return
	}
	nds, err := himap.ModeNodes(tag)
	if err != nil {
		fmt.Printf("mode %#08x noeuds KO: %v\n", id, err)
		return
	}
	dico := dicoNoeuds()
	fmt.Printf("=== mode %#08x : %d noeuds ===\n", id, len(nds))
	for i, n := range nds {
		m := himap.NodeModelTransform(nds, i)
		nom := dico[n.Name]
		flag := ""
		if !himap.EchelleEstUnitaire(n.Scale) {
			flag += " <<ECHELLE_LOCALE!=1"
		}
		if !himap.EchelleEstUnitaire(m.Scale) {
			flag += " <<ECHELLE_MODEL!=1"
		}
		if estNomArme(nom) {
			flag += " <<ARME?"
		}
		fmt.Printf("  n[%03d] %#08x %-20s parent=%-4d sLoc=%.3f | model pos=(%+.3f,%+.3f,%+.3f) sMod=%.3f%s\n",
			i, n.Name, nom, n.Parent, n.Scale, m.Trans[0], m.Trans[1], m.Trans[2], m.Scale, flag)
	}
}

func dicoNoeuds() map[uint32]string {
	d := map[uint32]string{}
	for _, s := range vocabNoeuds {
		d[uint32(mapvar.LabelHash(s))] = s
	}
	return d
}

func estNomArme(nom string) bool {
	for _, k := range []string{"gun", "turret", "weapon", "gunner", "mount", "laag", "chaingun", "barrel", "muzzle", "pedestal", "hardpoint"} {
		if strings.Contains(nom, k) {
			return true
		}
	}
	return false
}

// mesureMode charge un `mode`, agrege la boite/centroide de TOUS ses sommets et l'imprime.
func mesureMode(idx *himap.ModuleIndex, id uint32, detail bool) {
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		fmt.Printf("mode %#08x extraction KO: %v\n", id, err)
		return
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		fmt.Printf("mode %#08x render_model KO: %v\n", id, err)
		return
	}
	mn := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	mx := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	var sum [3]float64
	nv, nsec := 0, 0
	for i := 0; i < asset.MeshCount(); i++ {
		m := asset.Mesh(i)
		if m == nil || len(m.Vertices) == 0 {
			continue
		}
		nsec++
		for _, v := range m.Vertices {
			for a := 0; a < 3; a++ {
				mn[a] = math.Min(mn[a], v[a])
				mx[a] = math.Max(mx[a], v[a])
				sum[a] += v[a]
			}
			nv++
		}
		if detail {
			imprimeSection(asset, i, m)
		}
	}
	if nv == 0 {
		fmt.Printf("mode %#08x : AUCUN sommet exploitable (%d sections)\n", id, asset.MeshCount())
		return
	}
	c := [3]float64{sum[0] / float64(nv), sum[1] / float64(nv), sum[2] / float64(nv)}
	bmn, bmx, okb := asset.Bounds(0)
	fmt.Printf("=== mode %#08x : %d sections (%d non vides), %d sommets ===\n", id, asset.MeshCount(), nsec, nv)
	fmt.Printf("  BBOX geom (m)  X[%+.3f..%+.3f] Y[%+.3f..%+.3f] Z[%+.3f..%+.3f]\n", mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
	fmt.Printf("  ETENDUE (m)    dX=%.3f (long, +X=arr)  dY=%.3f (lat)  dZ=%.3f (haut)\n", mx[0]-mn[0], mx[1]-mn[1], mx[2]-mn[2])
	fmt.Printf("  CENTROIDE (m)  cX=%+.3f  cY=%+.3f  cZ=%+.3f\n", c[0], c[1], c[2])
	if okb {
		fmt.Printf("  BOUNDS quant   X[%+.3f..%+.3f] Y[%+.3f..%+.3f] Z[%+.3f..%+.3f]\n", bmn[0], bmx[0], bmn[1], bmx[1], bmn[2], bmx[2])
	}
}

// imprimeSection imprime l'empreinte 3D d'une section (dx,dy,dz + centroide) — detail optionnel.
func imprimeSection(asset *himap.RuntimeGeoAsset, i int, m *himap.Mesh) {
	mn := m.Vertices[0]
	mx := m.Vertices[0]
	var s [3]float64
	for _, v := range m.Vertices {
		for a := 0; a < 3; a++ {
			mn[a] = math.Min(mn[a], v[a])
			mx[a] = math.Max(mx[a], v[a])
			s[a] += v[a]
		}
	}
	n := float64(len(m.Vertices))
	fmt.Printf("    sec[%02d] nv=%5d dX=%.2f dY=%.2f dZ=%.2f  cX=%+.2f cY=%+.2f cZ=%+.2f\n",
		i, len(m.Vertices), mx[0]-mn[0], mx[1]-mn[1], mx[2]-mn[2], s[0]/n, s[1]/n, s[2]/n)
}

func splitHex(spec string) []uint32 {
	var out []uint32
	for _, s := range strings.Split(spec, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 0, 32)
		if err != nil {
			fmt.Printf("id %q illisible: %v\n", s, err)
			continue
		}
		out = append(out, uint32(v))
	}
	return out
}

func listeModules(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func cheminsModules(variant string, specs []string) ([]string, error) {
	root, err := himap.DeployRoot()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, s := range specs {
		v, base := variant, s
		if i := strings.IndexByte(s, ':'); i > 0 {
			v, base = s[:i], s[i+1:]
		}
		p := filepath.Join(root, v, "globals", base)
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("module %s introuvable (%s)", s, p)
		}
		out = append(out, p)
	}
	return out, nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur:", err)
		os.Exit(1)
	}
}
