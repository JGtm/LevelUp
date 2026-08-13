// Package himap — cuisson.go : LA CHAINE DE PRODUCTION DU FOND DE CARTE.
//
// POURQUOI CE FICHIER EXISTE. Jusqu'au 2026-08-10, la chaine validee sur 25 cartes natives et
// une carte Forge n'existait QUE dans des tests : `TestRenduCarte` cadrait, peuplait, bornait,
// coloriait, et ecrivait un PNG au chemin d'une variable d'environnement. Une chaine qui ne vit
// que dans un test ne PRODUIT rien — elle constate. Aucun asset n'etait livre.
//
// CE FICHIER NE CHANGE PAS LA RECETTE, il la sort des tests. Chaque etape ci-dessous est le
// portage litteral d'un helper de test, avec ses chiffres et ses justifications d'origine. Les
// tests l'appellent desormais ; ils ne la portent plus.
//
// LA CHAINE, dans l'ordre, et d'ou vient chaque regle :
//
//	cadre     voisinage des ancres d'objectifs (MargeCadre)      map_objectives.json
//	tranche   [TrancheDeJeuMin ; TrancheDeJeuMax] = [-12 ; +28]  prototype s31_raster.py
//	decor     grain du maillage, AireMaxTriangleJouable          mesure, valide utilisateur
//	frontiere maillage sddt par parite de rayon                  tag de la carte
//	eau       volumes sddt (PoseEau)                             tag de la carte
//	niveau    mediane des ancres moins AncrageDecalageSol        ancres
//
// AUCUN REGLAGE PAR CARTE. C'est la propriete qui rend la chaine transferable : une seule carte
// (Cliffhanger) possede l'oracle fort des positions de joueur, regler carte par carte serait
// donc impossible en principe, pas seulement penible.
//
// DEGRADATIONS : une carte sans tag sddt, sans volume d'eau, ou dont la frontiere exclut des
// ancres, est SIGNALEE (journal + `BilanCuisson.Degradations`) et cuite quand meme. Jamais
// avalee en silence — un oracle absent qui passe au vert est le piege le plus cher de ce
// chantier.
package himap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"sort"
)

// MargeCadre : marge autour des ancres, en metres, qui borne l'image.
//
// Meme portee que `PorteeAncre`, doublee pour que le cadre montre les abords de l'aire de jeu
// et pas seulement elle. Ce n'est PAS le tri d'instances refute au §10.8 du handoff (celui-la
// ecartait de la geometrie sans rien changer) : c'est le CADRE de l'image. Sans lui, l'emprise
// du sbsp reclame des grilles de plusieurs Go sur les grandes cartes.
const MargeCadre = 2 * PorteeAncre

// MargeBornageInstance : marge, en metres, autour de la boite monde declaree d'une instance,
// au-dela de laquelle son maillage n'ecrit plus.
//
// La boite (sbsp @0x7C) et le maillage (tag rtgo) viennent de deux sources independantes, et
// quelques instances des modules globaux debordent d'un facteur 42,8 de leur diagonale au
// 99e centile. Sans bornage elles deversent du decor sur toute la carte.
const MargeBornageInstance = 0.5

// ErrSansAncre : la carte n'a aucune ancre d'objectif au catalogue. Sans ancre il n'y a ni
// cadre, ni niveau de jeu, ni oracle — la cuisson s'arrete au lieu de deviner.
var ErrSansAncre = errors.New("himap: aucune ancre d'objectif — pas d'oracle, pas de cadre")

// OptionsCuisson decrit une carte a cuire.
type OptionsCuisson struct {
	// RacineDeploy est la racine `deploy` de l'installation (cf. DeployRoot).
	RacineDeploy string
	// CheminModule est le .module de la carte, variante `pc` (la seule qui porte la
	// geometrie de rendu).
	CheminModule string
	// Ancres sont les positions monde des objectifs de la carte.
	Ancres [][3]float64
	// SansFrontiere neutralise la coquille de mort. TEMOIN de comparaison uniquement :
	// jamais une option de production.
	SansFrontiere bool
}

// BilanCuisson chiffre ce que la cuisson a fait. Il est publie avec l'asset : un fond de carte
// dont on ne sait pas combien d'ancres ont trouve leur sol n'est pas verifiable.
type BilanCuisson struct {
	Module string
	// BSPs est le nombre de tags sbsp du module ; BSPInstances celui du bsp retenu par les
	// ancres (une carte declare aussi un horizon lointain, cf. ChoisitBSP).
	BSPs         int
	BSPInstances int
	// Dessinees / EcarteesDecor : instances rendues, et instances tenues pour du decor par
	// le grain de leur maillage.
	Dessinees     int
	EcarteesDecor int
	// NiveauDeJeu est l'altitude du sol joue, deduite des ancres.
	NiveauDeJeu float64
	// Ancres / AncresDansLeCadre / AncresAvecSol : l'oracle FAIBLE, disponible sur toutes les
	// cartes. Une ancre sans sol dessine est un trou de reconstruction.
	Ancres            int
	AncresDansLeCadre int
	AncresAvecSol     int
	// EcartMedianAncre est l'ecart median, en metres, entre l'ancre ramenee au sol et le sol
	// dessine. NaN si aucune ancre n'a trouve de sol.
	EcartMedianAncre float64
	// TauxCouverture / CarteCouverte / CellulesSubstituees : la voie de reference contre les
	// TOITS (rendu_reference.go). TauxCouverture est la part de matiere qui cache un sol
	// praticable ; au-dela de SeuilCarteCouverte la carte est couverte et ses pixels, dans la
	// portee des ancres, montrent la surface la plus proche de la reference. Une carte non
	// couverte n'est pas touchee — son PNG reste identique au bit.
	TauxCouverture      float64
	CarteCouverte       bool
	CellulesSubstituees int
	// PlansFrontiere / FrontiereAppliquee / CellulesEffacees : la coquille de mort declaree
	// par la carte.
	PlansFrontiere     int
	FrontiereAppliquee bool
	CellulesEffacees   int
	// VolumesEau / CellulesEau : l'habillage d'eau, pose apres le terrain et sans le toucher.
	VolumesEau  int
	CellulesEau int
	// ObjetsForge / ObjetsDessines / ObjetsSansModele / VolumesDeMort : cuisson d'une carte
	// FORGE (cf. cuisson_forge.go). Tous nuls sur une carte native.
	ObjetsForge      int
	ObjetsDessines   int
	ObjetsSansModele int
	VolumesDeMort    int
	// Degradations liste, en clair, ce qui a manque. Vide = chaine complete.
	Degradations []string
}

func (b *BilanCuisson) degrade(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	b.Degradations = append(b.Degradations, msg)
	slog.WarnContext(ctx, "cuisson degradee", "module", b.Module, "raison", msg)
}

// CuitCarteNative rend le fond de carte d'une carte CUITE DANS UN MODULE (les 25 cartes
// natives), et le bilan chiffre de sa cuisson.
//
// Le rendu retourne porte le z-buffer, les normales et l'eau ; l'encoder en image est le role
// de `FondPNG` (fond_png.go), qui publie aussi le calage.
func CuitCarteNative(ctx context.Context, opts OptionsCuisson) (*Rendu, BilanCuisson, error) {
	b := BilanCuisson{Module: filepath.Base(filepath.Dir(opts.CheminModule)), Ancres: len(opts.Ancres)}
	if len(opts.Ancres) == 0 {
		return nil, b, ErrSansAncre
	}
	r := CadreSurAncres(opts.Ancres)
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	// La tranche est TRANSLATEE AU SOL JOUE — meme regle que la chaine Forge, et pour la meme
	// raison. Justification chiffree : cf. `TrancheDeJeu` (rendu.go).
	r.Tranche(TrancheDeJeu(zJeu))
	r.NiveauDeJeu(zJeu)
	// La voie de reference contre les TOITS : armee avant de projeter, decidee juste apres —
	// et AVANT la frontiere, pour que celle-ci efface sur l'image finale.
	s := NewSurfaceReference(opts.Ancres)
	r.ArmeReference(s)

	if err := peupleDepuisModule(ctx, r, &b, opts); err != nil {
		return nil, b, err
	}
	b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s)
	appliqueFrontiere(ctx, r, &b, opts, zJeu)
	JugeParLesAncres(r, &b, opts.Ancres)
	PoseEauDepuisModule(ctx, r, &b, opts.CheminModule)
	slog.InfoContext(ctx, "carte cuite", "module", b.Module,
		"instances", b.Dessinees, "decor", b.EcarteesDecor,
		"ancres", fmt.Sprintf("%d/%d", b.AncresAvecSol, b.AncresDansLeCadre),
		"couverture", fmt.Sprintf("%.1f%%", 100*b.TauxCouverture), "couverte", b.CarteCouverte,
		"substituees", b.CellulesSubstituees,
		"px", r.NX*r.NY, "degradations", len(b.Degradations))
	return r, b, nil
}

// CadreSurAncres prepare un rendu borne au voisinage des ancres, a l'echelle de production.
func CadreSurAncres(ancres [][3]float64) *Rendu {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, a := range ancres {
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], a[k]-MargeCadre)
			hi[k] = math.Max(hi[k], a[k]+MargeCadre)
		}
	}
	return NewRendu(lo, hi, EchelleFondCarte)
}

// peupleDepuisModule resout l'index de modules, choisit le bsp par les ancres, et peuple.
func peupleDepuisModule(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuisson) error {
	chemins, err := GeometrySearchPath(opts.RacineDeploy, opts.CheminModule)
	if err != nil {
		return fmt.Errorf("chemins de geometrie : %w", err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		return fmt.Errorf("index des modules : %w", err)
	}
	bsps, err := ReadModuleInstances(opts.CheminModule)
	if err != nil {
		return fmt.Errorf("instances du module : %w", err)
	}
	bsp := ChoisitBSP(bsps, opts.Ancres)
	b.BSPs, b.BSPInstances = len(bsps), len(bsp.Instances)
	b.Dessinees, b.EcarteesDecor = PeupleRendu(ctx, r, idx, bsp)
	if b.Dessinees == 0 {
		return fmt.Errorf("aucune instance dessinee sur %d du bsp retenu", len(bsp.Instances))
	}
	return nil
}

// PeupleRendu projette les instances d'un bsp dans un rendu deja cadre.
//
// C'est LE coeur de la chaine, et il est partage avec le banc de non-regression
// (`TestBancCliffhanger`) : deux boucles jumelles auraient diverge, et le banc aurait alors
// cesse de garder la production.
//
// Rend le nombre d'instances dessinees et le nombre ecartees comme decor.
func PeupleRendu(ctx context.Context, r *Rendu, idx *ModuleIndex, bsp BSPInstances) (int, int) {
	assets := map[uint32]*RuntimeGeoAsset{}
	dessinees, decor := 0, 0
	for _, in := range bsp.Instances {
		// Reclaimer ECARTE les projecteurs d'ombre (`MeshIsCustomShadowCaster`) : leur
		// maillage est un volume de projection, pas de la geometrie visible.
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			continue
		}
		a, deja := assets[id]
		if !deja {
			a = ouvreAsset(ctx, idx, id)
			assets[id] = a
		}
		if a == nil {
			continue
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			continue
		}
		if EstDecorGrossier(m, in, AireMaxTriangleJouable) {
			decor++
			continue
		}
		dessinees++
		r.AddMeshBorne(m, in, MargeBornageInstance)
	}
	return dessinees, decor
}

// ouvreAsset extrait et decode un tag rtgo. Un tag illisible est LOGGE puis saute — une carte
// ne doit pas mourir sur un maillage, mais une extraction muette masquerait un trou.
func ouvreAsset(ctx context.Context, idx *ModuleIndex, id uint32) *RuntimeGeoAsset {
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		slog.DebugContext(ctx, "rtgo illisible", "id", fmt.Sprintf("%08x", id), "err", err)
		return nil
	}
	a, err := NewRuntimeGeoAsset(tag, blob)
	if err != nil {
		slog.DebugContext(ctx, "rtgo indecodable", "id", fmt.Sprintf("%08x", id), "err", err)
		return nil
	}
	return a
}

// appliqueFrontiere borne le rendu a la frontiere de mort declaree par la carte.
//
// ELLE NE S'APPLIQUE QUE SI ELLE GARDE TOUTES LES ANCRES. Quatre cartes sur 25 y perdent de la
// zone jouable — sur celles-la on s'en passe plutot que d'amputer la carte.
func appliqueFrontiere(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuisson, zJeu float64) {
	if opts.SansFrontiere {
		b.degrade(ctx, "frontiere de mort NON appliquee (temoin)")
		return
	}
	s, ok := frontiereDuModule(ctx, b, opts.CheminModule)
	if !ok {
		return
	}
	b.PlansFrontiere = len(s.Frontieres)
	if !FrontiereGardeLesAncres(s, opts.Ancres, zJeu) {
		b.degrade(ctx, "frontiere de %d plans EXCLUT des ancres — non appliquee", len(s.Frontieres))
		return
	}
	b.CellulesEffacees = r.RestreintALaFrontiere(s, zJeu)
	b.FrontiereAppliquee = true
}

// frontiereDuModule lit le tag sddt de la variante `any` du module.
func frontiereDuModule(ctx context.Context, b *BilanCuisson, cheminModule string) (Sddt, bool) {
	chemin, err := ModuleVariante(cheminModule, "any")
	if err != nil {
		b.degrade(ctx, "sddt : %v", err)
		return Sddt{}, false
	}
	s, err := LitSddt(chemin)
	if err != nil {
		b.degrade(ctx, "sddt illisible (%s) : %v", filepath.Base(chemin), err)
		return Sddt{}, false
	}
	if len(s.Frontieres) == 0 {
		b.degrade(ctx, "aucune frontiere dans le sddt de %s", filepath.Base(chemin))
		return s, false
	}
	return s, true
}

// PoseEauDepuisModule pose les volumes d'eau du tag sddt sur un rendu deja peuple.
//
// L'eau est un HABILLAGE : elle est posee apres le terrain et ne le touche pas (le banc le
// prouve — z-buffer et eclairement identiques a l'octet avant/apres).
func PoseEauDepuisModule(ctx context.Context, r *Rendu, b *BilanCuisson, cheminModule string) {
	chemin, err := ModuleVariante(cheminModule, "any")
	if err != nil {
		b.degrade(ctx, "eau : %v", err)
		return
	}
	s, err := LitSddt(chemin)
	if err != nil {
		b.degrade(ctx, "eau : sddt illisible (%s) : %v", filepath.Base(chemin), err)
		return
	}
	if len(s.VolumesEau) == 0 {
		b.degrade(ctx, "eau : aucun volume d'eau dans le sddt de %s", filepath.Base(chemin))
		return
	}
	r.PoseEau(s.VolumesEau)
	b.VolumesEau = len(s.VolumesEau)
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			if r.Eau(i, j) {
				b.CellulesEau++
			}
		}
	}
}

// JugeParLesAncres renseigne l'oracle FAIBLE du bilan : chaque ancre, ramenee au sol,
// trouve-t-elle de la matiere sous elle ?
//
// Il ne dit pas que la carte est belle, il dit qu'elle n'est pas trouee la ou on joue. Le seul
// juge du rendu reste l'utilisateur.
func JugeParLesAncres(r *Rendu, b *BilanCuisson, ancres [][3]float64) {
	var ecarts []float64
	b.AncresDansLeCadre, b.AncresAvecSol = 0, 0
	for _, a := range ancres {
		i := int((a[0] - r.Min[0]) / r.Cell)
		j := int((a[1] - r.Min[1]) / r.Cell)
		if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
			continue
		}
		b.AncresDansLeCadre++
		if z, ok := r.Altitude(i, j); ok {
			b.AncresAvecSol++
			ecarts = append(ecarts, a[2]-AncrageDecalageSol-z)
		}
	}
	b.EcartMedianAncre = Centile(ecarts, 0.5)
}

// ChoisitBSP retient le bsp qui contient le PLUS D'ANCRES, et a defaut celui qui porte le plus
// d'instances.
//
// MESURE DU 2026-08-09 : sur les 27 cartes du balayage, le bsp le plus peuple est TOUJOURS
// celui qui contient les ancres. Cette regle ne change donc AUCUN chiffre — elle supprime une
// dependance au hasard, elle ne corrige rien. Le taux de 306/474 est identique avec et sans.
//
// Pourquoi le compte d'instances ne suffit pas EN PRINCIPE : une carte declare plusieurs bsp,
// dont un decor lointain. Cliffhanger en a deux — l'arene,
// 113 x 114 m et 10 357 instances, et un horizon de 6 619 x 10 471 m avec 3 971 instances. Ici
// le plus peuple est le bon PAR CHANCE ; rien ne le garantit ailleurs, et retenir l'horizon
// donne une carte vide de tout ce qui interesse. Les ancres, elles, sont dans l'aire de jeu par
// construction : elles designent le bon bsp sans qu'on ait a le deviner. Le repli sur le plus
// peuple ne sert que si aucune ancre ne tombe dans aucune boite.
func ChoisitBSP(bsps []BSPInstances, ancres [][3]float64) BSPInstances {
	var meilleur BSPInstances
	if len(ancres) > 0 {
		mieux := 0
		for _, b := range bsps {
			n := 0
			for _, a := range ancres {
				if a[0] >= b.Bounds.Min[0] && a[0] <= b.Bounds.Max[0] &&
					a[1] >= b.Bounds.Min[1] && a[1] <= b.Bounds.Max[1] &&
					a[2] >= b.Bounds.Min[2] && a[2] <= b.Bounds.Max[2] {
					n++
				}
			}
			if n > mieux {
				mieux, meilleur = n, b
			}
		}
		if mieux > 0 {
			return meilleur
		}
	}
	for _, b := range bsps {
		if len(b.Instances) > len(meilleur.Instances) {
			meilleur = b
		}
	}
	return meilleur
}

// MedianeZ rend l'altitude mediane d'un jeu de points.
func MedianeZ(pts [][3]float64) float64 {
	zs := make([]float64, 0, len(pts))
	for _, p := range pts {
		zs = append(zs, p[2])
	}
	if len(zs) == 0 {
		return 0
	}
	return Centile(zs, 0.5)
}

// Centile rend le centile p d'un echantillon, sans le modifier. Un echantillon vide rend NaN
// plutot que zero : un zero silencieux se confond avec une mesure.
func Centile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[int(p*float64(len(c)-1))]
}
