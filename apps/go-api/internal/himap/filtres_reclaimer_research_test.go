//go:build gamefiles && research

package himap

// filtres_reclaimer_research_test.go — RELEVES de filtres qui lisent le depot hote (D1 lot 2c,
// 2026-10-28, decision utilisateur : conserves comme instruments de recherche).
//
// Le releve des cartes natives joint les ancres du catalogue d'objectifs, et le releve
// d'Isolation lit la variante `.mvar` du dump de retro-ingenierie (`.ai/re_dump/mapvar`) : deux
// donnees que la brique ne possede pas. A la demande :
// `go test -tags=gamefiles,research ./internal/himap/ -run 'TestFiltresReclaimer'`.
// Le balayage des drapeaux d'instance (`TestBalayageDrapeauxInstances`) et le temoin d'offsets
// restent sous `gamefiles` : le jeu seul suffit.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

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

func objetsDeLaVariante(t *testing.T, carte CarteForge) []mapvar.Object {
	t.Helper()
	brut := lireVarianteDuDepot(t, carte.FichierMvar)
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Skipf("variante illisible : %v", err)
	}
	return v.Objects
}

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

func carteForgeParNom(nom string) (CarteForge, bool) {
	for _, c := range CartesForge {
		if c.Nom == nom {
			return c, true
		}
	}
	return CarteForge{}, false
}
