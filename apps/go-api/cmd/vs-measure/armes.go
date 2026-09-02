package main

// armes.go — sous-commande `armes` du driver jetable vs-measure (planche-contact des armes
// arriere de la famille Warthog, 2026-09-01). Trois sources, sans en sauter aucune :
//
//	-weapscan=mots   : enumere TOUS les tags `weap`, filtre par chaines ASCII (mots-cles), resout
//	                   la chaine hlmt->mode avec le resolveur CORRIGE (RefModeleVehicule est
//	                   agnostique du groupe : simple balayage de refs), et compte les refs par
//	                   groupe pour dire honnetement ce que porte le weap.
//	-modescan=mots   : enumere TOUS les `mode` dont une chaine ASCII matche (4e source, exhaustive).
//	-chassis=0xHEX   : rend le chassis (perm de base), la reference Razorback (unarmed) et CHAQUE
//	                   permutation non-base, region par region, isolee sur le canevas fixe.
//	-pieces=0xHEX,.. : rend chaque `mode` isole sur le canevas fixe (absent de l'index = saute,
//	                   il sera rendu par l'autre passe de modules).
//
// Tous les rendus partagent -cadre/-cellmm : meme grille de pixels, composables en 2D.
import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/himap"
)

const (
	permDefault = 0x42c9679f // StringId `default`
	permUnarmed = 0x4e154ee8 // StringId `unarmed` (= Razorback, VALIDE par l'utilisateur)
)

type armesCfg struct {
	out   string
	cadre float64
	cell  float64
}

func armesMain(args []string) {
	fs := flag.NewFlagSet("armes", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir (basenames ou variant:basename, virgule)")
	variant := fs.String("variant", "any", "variante deploy: any|pc|ds")
	weapScan := fs.String("weapscan", "", "mots-cles (virgule) : enumere les weap et resout hlmt->mode")
	weapMesure := fs.Bool("weapmesure", false, "mesure (bbox/centroide) le mode resolu de chaque weap present dans l'index")
	vehiScan := fs.String("vehiscan", "", "mots-cles (virgule) : enumere les vehi dont une chaine ASCII matche")
	ids := fs.String("ids", "", "tags (hex, virgule) a dumper : chaines ASCII, refs par groupe, resolution")
	modeScan := fs.String("modescan", "", "mots-cles (virgule) : enumere les mode dont une chaine ASCII matche")
	chassis := fs.String("chassis", "", "mode chassis (hex) : base + unarmed + chaque permutation isolee")
	pieces := fs.String("pieces", "", "modes (hex, virgule) a rendre isoles sur le canevas fixe")
	sons := fs.String("sons", "", "weap (hex, virgule) dont suivre la chaine sonore -> banques nommees (FNV-1)")
	nodes := fs.String("nodes", "", "modes (hex, virgule) dont dumper le squelette")
	hashes := fs.String("hashes", "", "StringId (hex, virgule) a nommer par brute-force murmur3 (sans modules)")
	out := fs.String("out", ".", "dossier de sortie des PNG")
	cadre := fs.Float64("cadre", 5, "demi-emprise du canevas fixe (m)")
	cellmm := fs.Int("cellmm", 8, "mm/pixel")
	_ = fs.Parse(args)

	if *hashes != "" {
		for h, nom := range dicoBanques() {
			fmt.Printf("FNV1 %#08x = %s\n", h, nom)
		}
		bruteForceNoeuds(splitHex(*hashes))
		return
	}
	chemins, err := cheminsModules(*variant, listeModules(*mods))
	must(err)
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	must(err)
	must(os.MkdirAll(*out, 0o755))
	cfg := armesCfg{out: *out, cadre: *cadre, cell: float64(*cellmm) / 1000.0}

	if *weapScan != "" {
		scanGroupe(idx, "weap", motsCles(*weapScan), true, *weapMesure, cfg)
	}
	if *vehiScan != "" {
		scanGroupe(idx, himap.GroupeVehi, motsCles(*vehiScan), true, false, cfg)
	}
	for _, id := range splitHex(*ids) {
		dumpTag(idx, id)
	}
	for _, id := range splitHex(*sons) {
		chaineSonore(idx, id)
	}
	for _, id := range splitHex(*nodes) {
		dumpNoeuds(idx, id)
	}
	if *modeScan != "" {
		scanGroupe(idx, himap.GroupeMode, motsCles(*modeScan), false, false, cfg)
	}
	if *chassis != "" {
		for _, id := range splitHex(*chassis) {
			rendChassisEtPermutations(idx, id, cfg)
		}
	}
	for _, id := range splitHex(*pieces) {
		rendPiece(idx, id, cfg)
	}
}

func motsCles(spec string) []string {
	var out []string
	for _, s := range strings.Split(spec, ",") {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// scanGroupe enumere tous les tags d'un groupe, garde ceux dont une chaine ASCII contient un
// mot-cle (ou TOUS si `tous`), et imprime : id, taille, chaines matchees, refs par groupe, et
// (si resoudre) le modele resolu par RefModeleVehicule.
func scanGroupe(idx *himap.ModuleIndex, groupe string, mots []string, resoudre, mesurer bool, cfg armesCfg) {
	ids := idx.EntreesDuGroupe(groupe)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	fmt.Printf("=== SCAN %s : %d tags indexes, mots-cles=%v ===\n", groupe, len(ids), mots)
	nMatch, nRes := 0, 0
	for _, id := range ids {
		tag, err := idx.Extract(id)
		if err != nil {
			fmt.Printf("%s %#08x extraction KO: %v\n", groupe, id, err)
			continue
		}
		noms := chainesASCII(tag, 4)
		matches := chainesMatchant(noms, mots)
		mid, mg, ok := uint32(0), "", false
		if resoudre {
			mid, mg, ok = himap.RefModeleVehicule(context.Background(), idx, tag)
		}
		// On imprime les weap MATCHES, et aussi tout weap qui resout un modele (nom strippe ?).
		if len(matches) == 0 && !ok {
			continue
		}
		nMatch++
		etat := "-"
		if ok {
			_, module, _ := idx.Lookup(mid)
			etat = fmt.Sprintf("%s %#08x@%s", mg, mid, moduleCourt(module))
			nRes++
		}
		fmt.Printf("%s %#08x taille=%-7d modele=%-16s refs{%s} noms=%s\n",
			groupe, id, len(tag), etat, refsParGroupe(idx, tag, id), strings.Join(apercuNoms(matches, noms, 8), " | "))
		if mesurer && ok && mg == himap.GroupeMode {
			mesureModeSiPresent(idx, fmt.Sprintf("weapmode_%08x_%08x", id, mid), mid, cfg)
		}
	}
	fmt.Printf("=== %s : %d retenus, %d avec modele resolu ===\n", groupe, nMatch, nRes)
}

// moduleCourt : `pc/globals-rtx-new.module` -> `pc:globals`.
func moduleCourt(chemin string) string {
	if chemin == "" {
		return "?"
	}
	base := strings.TrimSuffix(filepath.Base(chemin), "-rtx-new.module")
	v := filepath.Base(filepath.Dir(filepath.Dir(chemin)))
	return v + ":" + base
}

// largeurMinTourelle : une arme TENUE fait <= 0,15 m de large (echelle compacte) ; une arme de
// tourelle (monture, pods, canon Gauss) est plus large. Seuil de rendu des modes de weap.
const largeurMinTourelle = 0.25

// mesureModeSiPresent mesure un mode s'il est dans l'index courant (sinon le dit) ; s'il a la
// largeur d'une arme de tourelle, il est aussi rendu sur le canevas fixe (`weap_<mode>.png`).
func mesureModeSiPresent(idx *himap.ModuleIndex, nom string, mid uint32, cfg armesCfg) {
	tag, blob, err := idx.ExtractWithResources(mid)
	if err != nil {
		fmt.Printf("MESURE %s : extraction KO (%v)\n", nom, err)
		return
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		fmt.Printf("MESURE %s : render_model KO (%v)\n", nom, err)
		return
	}
	_, dY := imprimeMesure(nom, asset, nil)
	if dY >= largeurMinTourelle {
		rendSet(asset, nil, filepath.Join(cfg.out, fmt.Sprintf("weap_%08x.png", mid)), cfg)
		fmt.Printf("  -> rendu weap_%08x.png (largeur %.2f m >= %.2f)\n", mid, dY, largeurMinTourelle)
	}
}

// dumpTag imprime tout ce qu'on sait d'un tag : groupe, taille, chaines ASCII (>= 3), refs par
// groupe (avec la liste des hlmt/mode/rtgo/weap/vehi) et la resolution de modele.
func dumpTag(idx *himap.ModuleIndex, id uint32) {
	g, module, ok := idx.Lookup(id)
	if !ok {
		fmt.Printf("=== TAG %#08x : ABSENT de cet index ===\n", id)
		return
	}
	tag, err := idx.Extract(id)
	if err != nil {
		fmt.Printf("=== TAG %#08x extraction KO: %v ===\n", id, err)
		return
	}
	fmt.Printf("=== TAG %#08x groupe=%s module=%s taille=%d ===\n", id, g, moduleCourt(module), len(tag))
	if mid, mg, ok := himap.RefModeleVehicule(context.Background(), idx, tag); ok {
		_, mm, _ := idx.Lookup(mid)
		fmt.Printf("  RefModeleVehicule -> %s %#08x@%s\n", mg, mid, moduleCourt(mm))
	} else {
		fmt.Printf("  RefModeleVehicule -> AUCUN\n")
	}
	fmt.Printf("  refs{%s}\n", refsParGroupe(idx, tag, id))
	fmt.Printf("  ascii>=3: %s\n", strings.Join(chainesASCII(tag, 3), " | "))
}

func chainesMatchant(noms, mots []string) []string {
	var out []string
	for _, n := range noms {
		b := strings.ToLower(n)
		for _, m := range mots {
			if strings.Contains(b, m) {
				out = append(out, n)
				break
			}
		}
	}
	return out
}

// apercuNoms : les chaines matchees d'abord, puis les autres noms courts en lettres, borne a n.
func apercuNoms(matches, noms []string, n int) []string {
	vus := map[string]bool{}
	var out []string
	for _, m := range matches {
		if !vus[m] {
			vus[m] = true
			out = append(out, m)
		}
	}
	for _, s := range noms {
		if len(out) >= n {
			break
		}
		if vus[s] || len(s) > 40 || strings.ContainsAny(s, "/\\.:{}") {
			continue
		}
		vus[s] = true
		out = append(out, s)
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// refsParGroupe compte, par balayage d'octets, les refs vers des tags connus par groupe
// (tout ID, sans plancher ni parasite : c'est le compte BRUT), et liste hlmt/mode/rtgo.
func refsParGroupe(idx *himap.ModuleIndex, tag []byte, self uint32) string {
	compte := map[string]int{}
	var modeles []string
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(tag[o]) | uint32(tag[o+1])<<8 | uint32(tag[o+2])<<16 | uint32(tag[o+3])<<24
		if h == 0 || h == 0xffffffff || h == self || vus[h] {
			continue
		}
		g, _, ok := idx.Lookup(h)
		if !ok {
			continue
		}
		vus[h] = true
		compte[g]++
		if g == himap.GroupeHlmt || g == himap.GroupeMode || g == himap.GroupeRtgo || g == "weap" || g == himap.GroupeVehi {
			modeles = append(modeles, fmt.Sprintf("%s:%#08x", g, h))
		}
	}
	var gs []string
	for g := range compte {
		gs = append(gs, g)
	}
	sort.Strings(gs)
	var parts []string
	for _, g := range gs {
		parts = append(parts, fmt.Sprintf("%s=%d", g, compte[g]))
	}
	s := strings.Join(parts, " ")
	if len(modeles) > 0 {
		s += " [" + strings.Join(modeles, ",") + "]"
	}
	return s
}

// chainesASCII : suites imprimables >= min octets, dedupliquees (copie locale de vehicle-sprite).
func chainesASCII(b []byte, min int) []string {
	var out []string
	vus := map[string]bool{}
	cur := make([]byte, 0, 64)
	flush := func() {
		if len(cur) >= min {
			s := string(cur)
			if !vus[s] {
				vus[s] = true
				out = append(out, s)
			}
		}
		cur = cur[:0]
	}
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur = append(cur, c)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// rendChassisEtPermutations rend, au canevas fixe : la base (perm default de chaque region),
// la reference Razorback (base + unarmed), puis CHAQUE permutation non-base de chaque region,
// isolee. Imprime la table regions/permutations et la mesure de chaque permutation.
func rendChassisEtPermutations(idx *himap.ModuleIndex, id uint32, cfg armesCfg) {
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		fmt.Printf("chassis %#08x extraction KO: %v\n", id, err)
		return
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		fmt.Printf("chassis %#08x render_model KO: %v\n", id, err)
		return
	}
	regions, err := himap.ModeRegions(tag)
	if err != nil {
		fmt.Printf("chassis %#08x regions KO: %v\n", id, err)
		return
	}
	fmt.Printf("=== CHASSIS %#08x : %d sections, %d regions ===\n", id, asset.MeshCount(), len(regions))
	base := sectionsDeVariante(regions, permDefault)
	rendSet(asset, base, filepath.Join(cfg.out, "chassis_base.png"), cfg)
	imprimeMesure("chassis_base", asset, base)
	raz := sectionsDeVariante(regions, permUnarmed)
	rendSet(asset, raz, filepath.Join(cfg.out, "ref_razorback.png"), cfg)
	imprimeMesure("ref_razorback", asset, raz)
	for ri, r := range regions {
		for _, p := range r.Permutations {
			fmt.Printf("region[%02d] name=%#08x perm=%#08x idx=%d count=%d\n", ri, r.Name, p.Name, p.SectionIndex, p.SectionCount)
		}
	}
	for ri, r := range regions {
		for _, p := range r.Permutations {
			if p.Name == permDefault || p.SectionIndex < 0 || p.SectionCount <= 0 {
				continue
			}
			set := map[int]bool{}
			for s := p.SectionIndex; s < p.SectionIndex+p.SectionCount; s++ {
				set[s] = true
			}
			nom := fmt.Sprintf("perm_r%02d_%08x", ri, p.Name)
			rendSet(asset, set, filepath.Join(cfg.out, nom+".png"), cfg)
			imprimeMesure(nom, asset, set)
		}
	}
}

// sectionsDeVariante : chassis commun (perm default de chaque region) + les sections propres a
// la variante `nom` (meme regle que vehicle-sprite/variantes.go, SectionIndex<0 = herite).
func sectionsDeVariante(regions []himap.Region, nom uint32) map[int]bool {
	set := map[int]bool{}
	for _, r := range regions {
		for _, p := range r.Permutations {
			if (p.Name == permDefault || p.Name == nom) && p.SectionIndex >= 0 {
				for s := p.SectionIndex; s < p.SectionIndex+p.SectionCount; s++ {
					set[s] = true
				}
			}
		}
	}
	return set
}

// rendPiece rend un `mode` entier isole sur le canevas fixe (saute s'il est absent de l'index).
func rendPiece(idx *himap.ModuleIndex, id uint32, cfg armesCfg) {
	if _, _, ok := idx.Lookup(id); !ok {
		fmt.Printf("piece %#08x : absente de cet index (autre passe)\n", id)
		return
	}
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		fmt.Printf("piece %#08x extraction KO: %v\n", id, err)
		return
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		fmt.Printf("piece %#08x render_model KO: %v\n", id, err)
		return
	}
	nom := fmt.Sprintf("piece_%08x", id)
	rendSet(asset, nil, filepath.Join(cfg.out, nom+".png"), cfg)
	imprimeMesure(nom, asset, nil)
}

// rendSet rasterise un sous-ensemble de sections (nil = toutes) au canevas fixe et ecrit le PNG.
func rendSet(asset *himap.RuntimeGeoAsset, set map[int]bool, chemin string, cfg armesCfg) {
	mn := [2]float64{-cfg.cadre, -cfg.cadre}
	mx := [2]float64{cfg.cadre, cfg.cadre}
	o := himap.OptionsSprite{AxeHaut: himap.HautZ, CellMetres: cfg.cell, CadreMin: &mn, CadreMax: &mx}
	r, err := himap.RenduAssemblage([]himap.PartAssemblage{{Asset: asset, SectionsChoisies: set}}, o)
	if err != nil {
		fmt.Printf("  rendu %s KO: %v\n", filepath.Base(chemin), err)
		return
	}
	f, err := os.Create(chemin)
	if err != nil {
		fmt.Printf("  ecriture %s KO: %v\n", chemin, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, himap.SpriteObjetPNG(r, o)); err != nil {
		fmt.Printf("  encode %s KO: %v\n", chemin, err)
	}
}

// imprimeMesure : boite englobante + centroide (repere modele : +X arriere, Y lateral, Z haut)
// d'un sous-ensemble de sections (nil = toutes). Une ligne MESURE parsable. Rend (dX, dY).
func imprimeMesure(nom string, asset *himap.RuntimeGeoAsset, set map[int]bool) (float64, float64) {
	mn := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	mx := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	var sum [3]float64
	nv, nsec := 0, 0
	for i := 0; i < asset.MeshCount(); i++ {
		if set != nil && !set[i] {
			continue
		}
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
	}
	if nv == 0 {
		fmt.Printf("MESURE %s sec=0 nv=0 (vide)\n", nom)
		return 0, 0
	}
	n := float64(nv)
	fmt.Printf("MESURE %s sec=%d nv=%d X[%+.3f..%+.3f] Y[%+.3f..%+.3f] Z[%+.3f..%+.3f] dX=%.3f dY=%.3f dZ=%.3f cX=%+.3f cY=%+.3f cZ=%+.3f\n",
		nom, nsec, nv, mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], mx[0]-mn[0], mx[1]-mn[1], mx[2]-mn[2], sum[0]/n, sum[1]/n, sum[2]/n)
	return mx[0] - mn[0], mx[1] - mn[1]
}
