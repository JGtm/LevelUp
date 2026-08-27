package himap

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// LE RELEVE — combien chacun des trois filtres de visibilite retirerait-il ?
//
// Trois declarations du format designent de la geometrie que le jeu ne veut pas voir sur une
// vue de dessus (cf. filtres_reclaimer.go). Aucune n'est branchee sur la cuisson. La question
// n'est PAS « faut-il les brancher » mais « combien cela retire-t-il » : un filtre qui touche
// 0,1 % des triangles ne vaut pas un lot, un filtre qui en touche 40 % doit etre regarde a
// l'image avant d'etre pose. Ces tests ne corrigent rien et ne cuisent rien — ils comptent.
//
// AUCUN MAILLAGE N'EST DECODE. Les tags sont ouverts avec un blob de ressources NIL et les
// triangles se lisent aux descripteurs (`TrianglesDuMaillage`) : c'est ce qui rend le releve
// tenable en memoire la ou une cuisson demande des giga-octets.

// cartesNativesMesurees : les cartes NATIVES du releve, avec leur cle de catalogue
// d'objectifs. Le dossier installe et le nom du catalogue ne coincident pas (cf.
// ChercheModuleInstalle) — les deux sont donc portes ici.
//
// LES DEUX DERNIERES NE SONT PAS LA PAR SYMETRIE. Recharge et Launch Site sont les deux
// cartes ou `TestDrapeauCarteParBSP` trouve le drapeau `exclude from intel map` le plus pose
// — 25,2 % et 21,0 % des instances retenues, contre 0 % sur quatre des cinq premieres. Un
// releve qui ne mesurerait que des cartes a zero conclurait « filtre sans effet » en
// n'ayant simplement pas regarde la ou il agit.
var cartesNativesMesurees = []struct{ dossier, moduleCatalogue string }{
	{"ctf_forbidden", "ctf_forbidden"},
	{"chasm", "chasm_map"},
	{"catalyst", "catalyst"},
	{"sgh_streets", "streets_sgh_streets"},
	{"btb_exiled", "btb_exiled"},
	{"sgh_blueprint", "recharge_sgh_blueprint"},
	{"va_launchsite", "launch_site_va_launchsite"},
}

// ---------------------------------------------------------------------------
// 3. LE RELEVE
// ---------------------------------------------------------------------------

// releveFiltres est le compte d'une carte. Chaque filtre est compte DEUX FOIS : en instances
// (ce qu'on ecarterait) et en triangles (ce que l'image y perdrait). Les deux divergent —
// une instance ecartee qui ne porte que 12 triangles ne change rien a l'image.
type releveFiltres struct {
	// unite nomme ce que compte `instances` : un bsp compte des instances de geometrie, une
	// variante Forge compte des objets poses. Sans ce nom, la meme ligne de journal ferait
	// lire « instances du bsp » sur une carte qui n'a pas de bsp.
	unite                            string
	instances, retenues, dessinables int
	sansRtgo, sansLOD                int
	triangles                        int
	intelToutes, intelDess, intelTri int
	ombreSect, ombreTri              int
	proxies, proxiesTri              int
	horsLOD, horsLODTri              int
	sansLOD0, sansLOD0Tri            int
}

func (r releveFiltres) journalise(t *testing.T, titre string) {
	t.Helper()
	pc := func(n, d int) string {
		if d == 0 {
			return "  n/a"
		}
		return fmt.Sprintf("%5.2f%%", 100*float64(n)/float64(d))
	}
	t.Logf("%s", titre)
	t.Logf("   %d %s · %d retenues par les filtres ACTUELS · %d dessinables "+
		"(%d sans rtgo, %d sans LOD) · %d triangles",
		r.instances, r.unite, r.retenues, r.dessinables, r.sansRtgo, r.sansLOD, r.triangles)
	t.Logf("   exclude from intel map : %d/%d %s (%s) · %d/%d dessinables (%s) · %s des triangles",
		r.intelToutes, r.instances, r.unite, pc(r.intelToutes, r.instances),
		r.intelDess, r.dessinables, pc(r.intelDess, r.dessinables), pc(r.intelTri, r.triangles))
	t.Logf("   section custom shadow caster : %d/%d dessinables (%s) · %s des triangles",
		r.ombreSect, r.dessinables, pc(r.ombreSect, r.dessinables), pc(r.ombreTri, r.triangles))
	t.Logf("   LOD has shadow proxies : %d/%d dessinables (%s) · %s des triangles",
		r.proxies, r.dessinables, pc(r.proxies, r.dessinables), pc(r.proxiesTri, r.triangles))
	t.Logf("   aucun enregistrement au LOD 0 (le filtre de Reclaimer) : %d/%d dessinables (%s) · %s des triangles",
		r.sansLOD0, r.dessinables, pc(r.sansLOD0, r.dessinables), pc(r.sansLOD0Tri, r.triangles))
	t.Logf("   pour information — LOD RETENU hors du LOD 0 : %d/%d dessinables (%s) · %s des triangles",
		r.horsLOD, r.dessinables, pc(r.horsLOD, r.dessinables), pc(r.horsLODTri, r.triangles))
}

// poseInstance est ce qu'on retient d'une instance pour la mesure : de quel maillage elle
// tire sa geometrie, et si elle porte le drapeau de carte.
type poseInstance struct {
	mesh  int
	intel bool
}

// nomsBitsFlagsInstance : les 16 drapeaux du champ `flags` d'une instance, dans l'ordre du
// plugin. Ils ne sont pas relus du XML ici — `TestPluginDrapeauxDeFiltre` verrouille deja
// l'accord — mais ecrits pour que le balayage se lise sans le plugin sous les yeux.
var nomsBitsFlagsInstance = [16]string{
	"render only", "does not block aoe damage", "remove from dynamic shadow geometry",
	"cinema only", "exclude from cinema", "disable play collision", "disable bullet collision",
	"ignore cubemap volume", "always generate floating shadow", "PVS always visible",
	"PVS always use LOD 0", "PVS don't use as an occluder", "exclude from intel map",
	"Exclude from broadphase calculation", "can generate decorators", "isHLOD",
}

// TestBalayageDrapeauxInstances balaye TOUS les modules installes et compte, BIT A BIT, le
// champ `flags` des instances de geometrie.
//
// POURQUOI L'HISTOGRAMME COMPLET ET PAS LE SEUL BIT 12. Un compte de zero ne dit pas la meme
// chose selon ce que valent les autres bits : si le champ entier etait toujours nul, ce
// serait l'OFFSET qui serait faux, pas le drapeau qui serait inutilise. Le releve des 16 bits
// separe les deux — et c'est la seule facon de conclure « inutilise » sans se tromper.
//
// Le balayage porte aussi sur les modules GLOBAUX : la geometrie de Live Fire vit dans
// `pc/globals/common-rtx-new.module`, pas dans le module de sa carte.
func TestBalayageDrapeauxInstances(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	dir, err := LevelsDir("pc")
	if err != nil {
		t.Skip(err)
	}
	dossiers, err := os.ReadDir(dir)
	if err != nil {
		t.Skip(err)
	}
	var chemins []string
	for _, d := range dossiers {
		if !d.IsDir() {
			continue
		}
		p := filepath.Join(dir, d.Name(), d.Name()+"-rtx-new.module")
		if _, err := os.Stat(p); err == nil {
			chemins = append(chemins, p)
		}
	}
	globs, _ := filepath.Glob(filepath.Join(racine, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)

	var total int
	var bits [16]int
	for _, p := range chemins {
		bsps, err := ReadModuleInstances(p)
		if err != nil {
			// ErrAucunTagSbsp est un fait connu (sgh_interlock), pas une panne : on le dit
			// et on continue, sans le confondre avec une chaine cassee.
			t.Logf("%-28s : %v", filepath.Base(p), err)
			continue
		}
		n, intel := 0, 0
		for _, b := range bsps {
			for _, in := range b.Instances {
				n++
				for k := 0; k < 16; k++ {
					if in.Flags&(1<<uint(k)) != 0 {
						bits[k]++
					}
				}
				if in.ExclueDeCarteIntel() {
					intel++
				}
			}
		}
		total += n
		t.Logf("%-28s : %6d instances · %d `exclude from intel map`", filepath.Base(p), n, intel)
	}
	t.Logf("TOTAL %d instances sur %d modules", total, len(chemins))
	for k, nom := range nomsBitsFlagsInstance {
		part := 0.0
		if total > 0 {
			part = 100 * float64(bits[k]) / float64(total)
		}
		t.Logf("   bit %2d %-38s %8d  (%5.2f%%)", k, nom, bits[k], part)
	}
	if total == 0 {
		t.Skip("aucune instance lue")
	}
	nonNuls := 0
	for _, n := range bits {
		if n > 0 {
			nonNuls++
		}
	}
	// Le champ doit VIVRE. S'il ne portait aucun bit sur des dizaines de milliers
	// d'instances, la conclusion a tirer serait « offset faux », pas « drapeau inutilise ».
	if nonNuls == 0 {
		t.Fatalf("aucun des 16 bits de `flags` n'est jamais pose sur %d instances : "+
			"l'offset %#x est suspect", total, insOffFlags)
	}
}

// TestFiltresReclaimerNatives compte, sur cinq cartes NATIVES installees, ce que chacun des
// trois filtres retirerait. Ne cuit rien : aucun maillage n'est decode, les triangles se
// lisent aux descripteurs du tag (cf. TrianglesDuMaillage).
//
// UNE CARTE PAR SOUS-TEST, ET CE N'EST PAS COSMETIQUE. Resoudre les tags de geometrie exige
// l'index des modules de production (`GeometrySearchPath`), et `himodule.Open` charge chaque
// `.module` ENTIER en memoire — les globaux pesent une douzaine de gigaoctets a eux seuls.
// Enchainer les cinq cartes dans un seul processus laisse le ramasse-miettes decider quand
// rendre ces gigaoctets ; les separer permet de lancer
// `-run 'TestFiltresReclaimerNatives/<carte>'` carte par carte, chaque processus rendant tout
// a sa sortie. Sur un poste qui fait tourner une campagne de cuisson en parallele, c'est la
// difference entre une mesure et une machine qui pagine.
func TestFiltresReclaimerNatives(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	for _, c := range cartesNativesMesurees {
		t.Run(c.dossier, func(t *testing.T) {
			modCarte := moduleDuJeu(t, "pc", c.dossier)
			bsps, err := ReadModuleInstances(modCarte)
			if err != nil {
				t.Fatalf("%s : %v", c.dossier, err)
			}
			bsp := ChoisitBSP(bsps, ancresFiltres(t, c.moduleCatalogue))
			idx := indexDeCarte(t, racine, modCarte)
			releveDuBSP(t, idx, bsp).journalise(t,
				fmt.Sprintf("=== %s (%d bsp, bsp retenu #%d) ===", c.dossier, len(bsps), bsp.FileIndex))
		})
	}
}

// releveDuBSP parcourt les instances d'un bsp en n'ouvrant chaque tag rtgo qu'UNE fois.
//
// Le regroupement par tag n'est pas une optimisation de confort : garder les assets ouverts
// au fil des instances retiendrait tous les tags du module en memoire, et ce paquet a deja
// paye ce piege ailleurs.
func releveDuBSP(t *testing.T, idx *ModuleIndex, bsp BSPInstances) releveFiltres {
	t.Helper()
	var r releveFiltres
	r.unite, r.instances = "instances du bsp", len(bsp.Instances)
	parTag := map[uint32][]poseInstance{}
	for _, in := range bsp.Instances {
		if in.ExclueDeCarteIntel() {
			r.intelToutes++
		}
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		r.retenues++
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			r.sansRtgo++
			continue
		}
		parTag[id] = append(parTag[id], poseInstance{mesh: in.MeshIndex, intel: in.ExclueDeCarteIntel()})
	}
	for id, poses := range parTag {
		tag, err := idx.Extract(id)
		if err != nil {
			r.sansRtgo += len(poses)
			continue
		}
		// Blob NIL : on ne decode aucun sommet. Les descripteurs, les sections et les
		// drapeaux de LOD vivent tous dans le tag.
		a, err := NewRuntimeGeoAsset(tag, nil)
		if err != nil {
			r.sansRtgo += len(poses)
			continue
		}
		for _, p := range poses {
			cumuleFiltres(&r, a, p)
		}
	}
	return r
}

func cumuleFiltres(r *releveFiltres, a *RuntimeGeoAsset, p poseInstance) {
	f, ok := a.FiltreDuMaillage(p.mesh)
	if !ok {
		r.sansLOD++
		return
	}
	tris, okT := a.TrianglesDuMaillage(p.mesh)
	if !okT {
		r.sansLOD++
		return
	}
	r.dessinables++
	r.triangles += tris
	if p.intel {
		r.intelDess++
		r.intelTri += tris
	}
	if f.ProjecteurOmbre() {
		r.ombreSect++
		r.ombreTri += tris
	}
	if f.ProxiesOmbre() {
		r.proxies++
		r.proxiesTri += tris
	}
	if f.HorsLOD(0) {
		r.horsLOD++
		r.horsLODTri += tris
	}
	if !a.PorteLeLOD(p.mesh, 0) {
		r.sansLOD0++
		r.sansLOD0Tri += tris
	}
}

// ---------------------------------------------------------------------------
// 4. ISOLATION — la carte Forge du releve
// ---------------------------------------------------------------------------

// TestFiltresReclaimerIsolation mesure les DEUX moities d'une carte Forge : le rack d'objets
// (des modeles `rtgo`/`mode`, donc les filtres de SECTION) et le canevas sur lequel elle est
// posee (des instances de bsp, donc le drapeau de carte).
//
// Le drapeau `exclude from intel map` n'existe pas sur un objet `.mvar` : il est porte par
// l'instance de geometrie d'un bsp. Sur une carte Forge il ne peut donc mordre que si la
// cuisson dessine le canevas (`DessineCanevas`).
func TestFiltresReclaimerIsolation(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	carte, ok := carteForgeParNom("Isolation")
	if !ok {
		t.Skip("Isolation n'est pas declaree dans CartesForge")
	}
	objets := objetsDeLaVariante(t, carte)
	opts := OptionsCuissonForge{
		RacineDeploy: racine, Objets: objets,
		CheminModuleCanevas: CheminCanevasForge(carte), Cle: carte.MapID,
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	modeles := modeleParType(t.Context(), objets, idx, forge)

	comptes := map[int32]int{}
	for _, o := range objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; mort {
			continue
		}
		comptes[o.TypeID]++
	}

	var r releveFiltres
	sansModele := 0
	for typeID, n := range comptes {
		m, ok := modeles[typeID]
		if !ok {
			sansModele += n
			continue
		}
		a := ouvreAsset(t.Context(), idx, m.id, m.groupe)
		if a == nil {
			sansModele += n
			continue
		}
		// Un objet Forge pose TOUS les maillages de son modele (cf. poseObjetsForge) : le
		// releve est donc par (objet x maillage), pas par objet.
		for mi := 0; mi < a.MeshCount(); mi++ {
			for k := 0; k < n; k++ {
				cumuleFiltres(&r, a, poseInstance{mesh: mi})
			}
		}
	}
	r.unite = "objets de la variante"
	r.instances, r.retenues = len(objets), len(objets)-sansModele
	r.journalise(t, fmt.Sprintf("=== Isolation (Forge, %d objets, %d types, %d objets sans modele) ===",
		len(objets), len(comptes), sansModele))

	releveCanevasIsolation(t, carte, idx)
}

// releveCanevasIsolation compte le drapeau de carte sur les instances du CANEVAS — la seule
// geometrie d'une carte Forge qui puisse le porter.
func releveCanevasIsolation(t *testing.T, carte CarteForge, idx *ModuleIndex) {
	t.Helper()
	chemin := CheminCanevasForge(carte)
	if chemin == "" {
		t.Log("canevas non resolu : pas de releve d'instances")
		return
	}
	bsps, err := ReadModuleInstances(chemin)
	if err != nil {
		t.Logf("canevas %s : %v", filepath.Base(chemin), err)
		return
	}
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}
	releveDuBSP(t, idx, bsp).journalise(t, fmt.Sprintf(
		"=== canevas d'Isolation (%s, %d bsp, bsp le plus peuple #%d) ===",
		filepath.Base(chemin), len(bsps), bsp.FileIndex))
}

// ---------------------------------------------------------------------------
// outils communs
// ---------------------------------------------------------------------------

func indexDeCarte(t *testing.T, racine, modCarte string) *ModuleIndex {
	t.Helper()
	chemins, err := GeometrySearchPath(racine, modCarte)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// ancresFiltres rend les ancres d'objectifs d'un module du catalogue. Nom propre a ce
// fichier : d'autres sondes en portent une variante, et deux symboles homonymes dans le
// meme paquet de test ne compilent pas.
func ancresFiltres(t *testing.T, module string) [][3]float64 {
	t.Helper()
	var pts [][3]float64
	for _, e := range ancresDuModule(t, module) {
		for _, o := range e.Objectives {
			pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	if len(pts) == 0 {
		t.Logf("aucune ancre pour %q : le bsp sera choisi au nombre d'instances", module)
	}
	return pts
}

// assetsNatifs ouvre au plus `max` tags rtgo distincts references par une carte installee.
func assetsNatifs(t *testing.T, dossier string, max int) []*RuntimeGeoAsset {
	t.Helper()
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	modCarte := moduleDuJeu(t, "pc", dossier)
	idx := indexDeCarte(t, racine, modCarte)
	bsps, err := ReadModuleInstances(modCarte)
	if err != nil {
		t.Fatal(err)
	}
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}
	vus := map[uint32]bool{}
	var out []*RuntimeGeoAsset
	for _, in := range bsp.Instances {
		id := in.RuntimeGeoID()
		if vus[id] || len(out) >= max {
			continue
		}
		vus[id] = true
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			continue
		}
		tag, err := idx.Extract(id)
		if err != nil {
			continue
		}
		if a, err := NewRuntimeGeoAsset(tag, nil); err == nil {
			out = append(out, a)
		}
	}
	return out
}

func carteForgeParNom(nom string) (CarteForge, bool) {
	for _, c := range CartesForge {
		if c.Nom == nom {
			return c, true
		}
	}
	return CarteForge{}, false
}

func objetsDeLaVariante(t *testing.T, carte CarteForge) []mapvar.Object {
	t.Helper()
	depot, err := cheminDepuisDepot(DepotVariantesCarte)
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(filepath.Join(depot, carte.FichierMvar))
	if err != nil {
		t.Skipf("variante absente : %v", err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Skipf("variante illisible : %v", err)
	}
	return v.Objects
}
