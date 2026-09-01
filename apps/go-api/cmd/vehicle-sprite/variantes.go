package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
)

// vocabPermutations : noms candidats de permutation de vehicule. Un StringId de nom est le
// murmur3 (mapvar.LabelHash) de la chaine ; on hache ce vocabulaire et on apparie pour nommer
// les variantes. Sur le Warthog, seuls `default` et `unarmed` (= Razorback) resolvent ; les
// trois variantes d'arme gardent leur StringId brut (mappees a la forme, cf. rapport).
var vocabPermutations = []string{
	"default", "base", "standard", "unarmed", "no_turret", "none", "empty",
	"rocket", "rockethog", "rocket_turret", "gauss", "gauss_turret", "chaingun", "gun_turret",
	"razorback", "cargo", "troop", "gungoose", "twin_gun", "mongoose", "warthog",
}

// cmdVariantes marche les regions/permutations d'un `mode` (render_model) et rend UNE image par
// VARIANTE (nom de permutation, partage entre regions). Pour une variante, chaque region
// contribue les sections de sa permutation portant ce nom, sinon (absente ou heritante) celles
// de la permutation de BASE. Toutes les variantes sont rendues au MEME cadre (comparables),
// traits noirs inclus. C'est le levier qui distingue Rockethog/Gauss/chaingun/Razorback.
//
// `-id` accepte PLUSIEURS modes (virgule) rendus dans une seule ouverture de modules (le module
// pc/globals pese ~8,5 Go : on evite de le recharger). `-diag` dumpe les regions/permutations et
// le choix de sections par variante. `-map` (modehex:stringidhex:nom,...) nomme les sorties.
func cmdVariantes(args []string) error {
	fs := flag.NewFlagSet("variantes", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir")
	variant := fs.String("variant", "any", "variante deploy")
	idHex := fs.String("id", "", "GlobalID(s) du mode (hex, virgule)")
	out := fs.String("out", ".", "dossier de sortie")
	cellmm := fs.Int("cellmm", 10, "mm/pixel")
	baseHex := fs.String("base", "0x42c9679f", "StringId de la permutation de BASE")
	diag := fs.Bool("diag", false, "dumper regions/permutations/sections par variante")
	mapSpec := fs.String("map", "", "noms de sortie: modehex:stringidhex:nom,... (sinon var_<lbl>_<id>)")
	armement := fs.Bool("armement", false, "ne rendre QUE les sections propres a la variante (tourelle seule, diagnostic)")
	full := fs.Bool("full", false, "rendre le modele ENTIER (toutes sections) a grande echelle -> full_<id>.png")
	secs := fs.String("secs", "", "groupes de sections a rendre isoles (virgule dans un groupe, ; entre groupes) -> secs_<liste>.png")
	cote := fs.Int("cote", 640, "cote px pour -full/-secs (rendu diagnostic ajuste au cadre, echelle libre)")
	axe := fs.Int("axe", 2, "axe haut du rendu diagnostic: 0=X 1=Y(profil) 2=Z(dessus)")
	_ = fs.Parse(args)

	ids, err := parseModeIDs(*idHex)
	if err != nil {
		return err
	}
	chemins, err := cheminsModules(*variant, listeModules(*mods))
	if err != nil {
		return err
	}
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		return err
	}
	baseU, _ := strconv.ParseUint(*baseHex, 0, 32)
	noms := parseNameMap(*mapSpec)
	for _, id := range ids {
		cfg := renduCfg{base: uint32(baseU), cell: float64(*cellmm) / 1000.0, out: *out, modeID: id, noms: noms[id], diag: *diag, armement: *armement, full: *full, secs: parseSecs(*secs), cote: *cote, axe: himap.AxeHaut(*axe)}
		if err := traiteMode(idx, cfg); err != nil {
			fmt.Printf("mode %#08x: %v\n", id, err)
		}
	}
	return nil
}

// renduCfg regroupe les reglages de rendu d'un mode (evite une liste de parametres trop longue).
type renduCfg struct {
	base     uint32            // StringId de la permutation de base (chassis commun)
	cell     float64           // metres par pixel (echelle fixe, cadre commun)
	out      string            // dossier de sortie
	modeID   uint32            // GlobalID du mode traite
	noms     map[uint32]string // StringId de variante -> nom de fichier (nil = var_<lbl>_<id>)
	diag     bool              // dumper regions/permutations/empreintes
	armement bool              // ne rendre que les sections propres a la variante (tourelle)
	full     bool              // rendre le modele entier a grande echelle (diagnostic)
	secs     [][]int           // groupes de sections a rendre isoles (diagnostic)
	cote     int               // cote px des rendus diagnostic
	axe      himap.AxeHaut     // axe haut des rendus diagnostic (0=X 1=Y profil 2=Z dessus)
}

// parseSecs decoupe `80,81;82,83` en groupes d'indices de section (diagnostic).
func parseSecs(spec string) [][]int {
	var out [][]int
	for _, grp := range strings.Split(spec, ";") {
		var g []int
		for _, s := range strings.Split(grp, ",") {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			if v, err := strconv.Atoi(s); err == nil {
				g = append(g, v)
			}
		}
		if len(g) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// traiteMode extrait un `mode`, marche ses regions, dumpe (option) puis rend ses variantes.
func traiteMode(idx *himap.ModuleIndex, cfg renduCfg) error {
	tag, blob, err := idx.ExtractWithResources(cfg.modeID)
	if err != nil {
		return err
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		return err
	}
	regions, err := himap.ModeRegions(tag)
	if err != nil {
		return err
	}
	if cfg.diag {
		dumpRegions(cfg.modeID, asset, regions, cfg.base)
	}
	if cfg.full || len(cfg.secs) > 0 {
		return renduDiag(asset, cfg)
	}
	return rendVariantes(asset, regions, cfg)
}

// renduDiag rend, a leur PROPRE cadre serre et a grande echelle (-cote), le modele entier
// (-full) et/ou un sous-ensemble arbitraire de sections (-secs). C'est le TEST DECISIF : voir
// si une geometrie d'arme existe dans le render_model, et a quoi ressemblent les sections de
// permutation isolees (arme vraie vs simple embase).
func renduDiag(asset *himap.RuntimeGeoAsset, cfg renduCfg) error {
	cote := cfg.cote
	if cote <= 0 {
		cote = 640
	}
	if cfg.full {
		o := himap.OptionsSprite{AxeHaut: cfg.axe, CotePx: cote}
		r, err := himap.RenduObjetIsole(asset, o)
		if err != nil {
			return err
		}
		p := filepath.Join(cfg.out, fmt.Sprintf("full_%08x.png", cfg.modeID))
		ecrirePNG(p, himap.SpriteObjetPNG(r, o))
		fmt.Printf("full -> %s  (%dx%d, %d sections)\n", p, r.NX, r.NY, asset.MeshCount())
	}
	for _, grp := range cfg.secs {
		set := map[int]bool{}
		lbls := make([]string, 0, len(grp))
		for _, s := range grp {
			set[s] = true
			lbls = append(lbls, strconv.Itoa(s))
		}
		o := himap.OptionsSprite{AxeHaut: cfg.axe, CotePx: cote, SectionsChoisies: set}
		r, err := himap.RenduObjetIsole(asset, o)
		if err != nil {
			fmt.Printf("secs[%s] KO: %v\n", strings.Join(lbls, "-"), err)
			continue
		}
		p := filepath.Join(cfg.out, fmt.Sprintf("secs_%s.png", strings.Join(lbls, "-")))
		ecrirePNG(p, himap.SpriteObjetPNG(r, o))
		fmt.Printf("secs[%s] -> %s  (%dx%d)\n", strings.Join(lbls, "-"), p, r.NX, r.NY)
	}
	return nil
}

// rendVariantes rend chaque variante d'un modele au meme cadre. Si cfg.noms est fourni, seules les
// variantes qui y figurent sont rendues, au nom de fichier demande. Si cfg.armement, seules les
// sections PROPRES a la variante (la tourelle) sont rendues — diagnostic pour la voir isolee.
func rendVariantes(asset *himap.RuntimeGeoAsset, regions []himap.Region, cfg renduCfg) error {
	dico := construitDico()
	vnoms := nomsVariantes(regions)
	fmt.Printf("mode %#08x : %d sections, %d regions, %d variantes\n",
		cfg.modeID, asset.MeshCount(), len(regions), len(vnoms))

	full, err := himap.RenduObjetIsole(asset, himap.OptionsSprite{AxeHaut: himap.HautZ, CellMetres: cfg.cell})
	if err != nil {
		return err
	}
	cmin := full.Min
	cmax := [2]float64{full.Min[0] + float64(full.NX)*full.Cell, full.Min[1] + float64(full.NY)*full.Cell}

	for _, nom := range vnoms {
		fichier := nomFichierVariante(nom, dico, cfg.noms)
		if fichier == "" {
			continue // une map est fournie et cette variante n'y est pas demandee
		}
		set := sectionsVariante(regions, nom, cfg.base)
		if cfg.armement {
			set = sectionsArmement(regions, nom, cfg.base)
		}
		fmt.Printf("variante %#08x -> %s : %d sections\n", nom, fichier, len(set))
		o := himap.OptionsSprite{AxeHaut: himap.HautZ, CellMetres: cfg.cell, SectionsChoisies: set, CadreMin: &cmin, CadreMax: &cmax}
		r, err := himap.RenduObjetIsole(asset, o)
		if err != nil {
			fmt.Printf("  rendu KO: %v\n", err)
			continue
		}
		ecrirePNG(filepath.Join(cfg.out, fichier+".png"), himap.SpriteObjetPNG(r, o))
	}
	return nil
}

// nomFichierVariante rend le nom de fichier (sans extension) d'une variante : celui demande par
// la map si elle existe (chaine vide = non demandee, a sauter), sinon `var_<lbl>_<stringid>`.
func nomFichierVariante(nom uint32, dico, noms map[uint32]string) string {
	if noms != nil {
		return noms[nom]
	}
	lbl := dico[nom]
	if lbl == "" {
		lbl = fmt.Sprintf("%08x", nom)
	}
	return fmt.Sprintf("var_%s_%08x", lbl, nom)
}

// dumpRegions imprime la table regions -> permutations (index/count de sections) puis, par
// variante, la source retenue region par region. C'est le levier de DIAGNOSTIC : il montre
// quelle region contribue quelle plage, et laquelle manque a une variante trouee.
func dumpRegions(modeID uint32, asset *himap.RuntimeGeoAsset, regions []himap.Region, base uint32) {
	dico := construitDico()
	fmt.Printf("=== DIAG mode %#08x : %d sections, %d regions ===\n", modeID, asset.MeshCount(), len(regions))
	for ri, r := range regions {
		fmt.Printf("region[%02d] name=%#08x %-9s perms=%d\n", ri, r.Name, dico[r.Name], len(r.Permutations))
		for _, p := range r.Permutations {
			fmt.Printf("    perm %#08x %-9s idx=%d count=%d\n", p.Name, dico[p.Name], p.SectionIndex, p.SectionCount)
		}
	}
	dumpSections(asset)
	for _, nom := range nomsVariantes(regions) {
		fmt.Printf("--- variante %#08x %s ---\n", nom, dico[nom])
		set := sectionsVariante(regions, nom, base)
		for ri, r := range regions {
			src, idx, cnt := choixRegion(r, nom, base)
			if cnt <= 0 {
				continue
			}
			fmt.Printf("    region[%02d] %#08x <- %-8s idx=%d count=%d\n", ri, r.Name, src, idx, cnt)
		}
		fmt.Printf("    => %d sections retenues\n", len(set))
	}
}

// dumpSections imprime l'empreinte REELLE de chaque section (min/max des sommets dequantifies,
// pas la boite globale renvoyee par Bounds) et le centroide (cX>0 = avant, cX<0 = arriere).
// Preuve de terrain : montre qu'une section de permutation d'armement (ex. section 5/6 =
// region00 de rocket/gauss) est un petit maillage, pas le corps entier — donc le corps DOIT
// venir de la base — et OU se trouve la tourelle (arriere attendu).
func dumpSections(asset *himap.RuntimeGeoAsset) {
	fmt.Println("--- empreintes reelles de sections (m, X=avant Y=gauche) ---")
	for i := 0; i < asset.MeshCount(); i++ {
		m := asset.Mesh(i)
		if m == nil || len(m.Vertices) == 0 {
			fmt.Printf("  sec[%02d] vide\n", i)
			continue
		}
		mn := [2]float64{m.Vertices[0][0], m.Vertices[0][1]}
		mx := mn
		var cx, cy float64
		for _, v := range m.Vertices {
			for a := 0; a < 2; a++ {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
			}
			cx += v[0]
			cy += v[1]
		}
		n := float64(len(m.Vertices))
		fmt.Printf("  sec[%02d] nv=%4d dx=%.2f dy=%.2f  cX=%+.2f cY=%+.2f\n",
			i, len(m.Vertices), mx[0]-mn[0], mx[1]-mn[1], cx/n, cy/n)
	}
}

// choixRegion reproduit, pour le diagnostic, la decision de sectionsVariante sur une region :
// permutation de la variante si presente et non heritante, sinon celle de base.
func choixRegion(r himap.Region, nom, base uint32) (src string, idx, cnt int) {
	p := permParNom(r, nom)
	src = "variante"
	if p == nil || p.SectionIndex < 0 {
		p = permParNom(r, base)
		src = "base"
	}
	if p == nil || p.SectionIndex < 0 {
		return "aucune", -1, 0
	}
	return src, p.SectionIndex, p.SectionCount
}

// parseModeIDs decoupe `-id=0xA,0xB` en liste de GlobalID de modes.
func parseModeIDs(spec string) ([]uint32, error) {
	var out []uint32
	for _, s := range strings.Split(spec, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 0, 32)
		if err != nil {
			return nil, fmt.Errorf("id %q illisible: %w", s, err)
		}
		out = append(out, uint32(v))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun -id de mode fourni")
	}
	return out, nil
}

// parseNameMap decoupe `-map=modehex:stringidhex:nom,...` en mode -> (stringid -> nom de fichier).
func parseNameMap(spec string) map[uint32]map[uint32]string {
	out := map[uint32]map[uint32]string{}
	for _, e := range strings.Split(spec, ",") {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		parts := strings.Split(e, ":")
		if len(parts) != 3 {
			continue
		}
		mode, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 0, 32)
		sid, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 0, 32)
		if err1 != nil || err2 != nil {
			continue
		}
		m := out[uint32(mode)]
		if m == nil {
			m = map[uint32]string{}
			out[uint32(mode)] = m
		}
		m[uint32(sid)] = strings.TrimSpace(parts[2])
	}
	return out
}

// construitDico hache le vocabulaire et rend un index StringId -> nom lisible.
func construitDico() map[uint32]string {
	d := map[uint32]string{}
	for _, s := range vocabPermutations {
		d[uint32(mapvar.LabelHash(s))] = s
	}
	return d
}

// nomsVariantes rend les noms de permutation distincts, dans l'ordre de premiere apparition.
func nomsVariantes(regions []himap.Region) []uint32 {
	vus := map[uint32]bool{}
	var out []uint32
	for _, r := range regions {
		for _, p := range r.Permutations {
			if !vus[p.Name] {
				vus[p.Name] = true
				out = append(out, p.Name)
			}
		}
	}
	return out
}

// sectionsVariante assemble les sections d'une variante : le CHASSIS COMMUN COMPLET (toutes les
// sections de la permutation de BASE, region par region) auquel on AJOUTE les sections propres a
// la variante. L'armement s'ajoute au chassis, il ne le remplace jamais — sans quoi une variante
// dont la region du corps porte une permutation propre (Rockethog, Gauss) perdait le plancher
// central qui relie cabine et arriere (trou au milieu). Une permutation dont SectionIndex < 0
// HERITE de la base (convention render_model) et ne contribue donc que via la base.
func sectionsVariante(regions []himap.Region, nom, base uint32) map[int]bool {
	set := map[int]bool{}
	ajoute := func(p *himap.Permutation) {
		if p == nil || p.SectionIndex < 0 {
			return
		}
		for s := p.SectionIndex; s < p.SectionIndex+p.SectionCount; s++ {
			set[s] = true
		}
	}
	for _, r := range regions {
		ajoute(permParNom(r, base)) // chassis commun, toujours complet
		if nom != base {
			ajoute(permParNom(r, nom)) // armement de la variante, en plus
		}
	}
	return set
}

// sectionsArmement rend UNIQUEMENT les sections propres a la variante (sa permutation par
// region, hors base) : la tourelle/armement isole du chassis. Diagnostic pour verifier a l'oeil
// que rocket/gauss/chaingun portent bien une geometrie distincte.
func sectionsArmement(regions []himap.Region, nom, base uint32) map[int]bool {
	set := map[int]bool{}
	if nom == base {
		return set
	}
	for _, r := range regions {
		p := permParNom(r, nom)
		if p == nil || p.SectionIndex < 0 {
			continue
		}
		for s := p.SectionIndex; s < p.SectionIndex+p.SectionCount; s++ {
			set[s] = true
		}
	}
	return set
}

func permParNom(r himap.Region, nom uint32) *himap.Permutation {
	for i := range r.Permutations {
		if r.Permutations[i].Name == nom {
			return &r.Permutations[i]
		}
	}
	return nil
}

func ecrirePNG(chemin string, img image.Image) {
	f, err := os.Create(chemin)
	if err != nil {
		fmt.Printf("  ecriture KO: %v\n", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Printf("  encode KO: %v\n", err)
	}
}
