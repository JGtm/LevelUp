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
// LA RECETTE EST UNIVERSELLE, SES REGLAGES NE LE SONT PLUS TOUS. Jusqu'au 2026-08-26 la chaine
// ne portait aucun reglage par carte, et c'est ce qui la rendait transferable : une seule carte
// (Cliffhanger) possede l'oracle fort des positions de joueur, donc regler carte par carte etait
// impossible en principe. Le gate utilisateur du 26/08 a tranche l'inverse, et il
// avait des images a l'appui : l'habillage (`encre` sur Cliffhanger, `jeu` sur Catalyst) et
// l'ECHELLE (une petite arene rend une petite image, donc pixelisee a l'ecran) ; l'ECRETAGE des toits s'y est ajoute le meme jour (ecretage_toits.go).
//
// CE QUI RESTE INTERDIT, et c'est la vraie regle : une BRANCHE par carte dans ce paquet. Les
// axes ci-dessus sont des ENTREES (`OptionsCuisson`), choisies en DONNEE par l'appelant,
// avec leur raison ecrite et la date de leur gate. La chaine, elle, ne sait pas quelle carte
// elle cuit.
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
	// EcreteToits arme l ECRETAGE (ecretage_toits.go) : un pixel dont aucune surface n est a
	// hauteur de jeu est VIDE au lieu d etre rendu. Reglage PAR CARTE, jamais un defaut — sur
	// une carte dont les rochers hauts font l identite, il les efface.
	EcreteToits bool
	// PlafondArene : hauteur, en metres, au-dela de la reference locale, a partir de laquelle
	// une surface cesse d etre tenue pour un etage joue. ZERO = `himap.PlafondArene` (6 m).
	PlafondArene float64
	// SansEau ecarte l habillage d eau de cette carte. Reglage PAR CARTE, et il en faut un :
	// l eau est peinte par la BOITE ENGLOBANTE de son volume (sddt.go), donc une carte dont un
	// volume a une grande boite recoit un grand rectangle bleu. Recharge en porte un depuis le
	// 2026-08-10 en production. Tant que le defaut de fond n est pas instruit, l ecarter carte
	// par carte vaut mieux que publier un aplat faux.
	SansEau bool
	// SubstitutionSansPortee retire la limite de PorteeAncre (25 m) de la voie de reference.
	//
	// LE DEFAUT QU ELLE CORRIGE, ecrit des le 2026-08-13 et non rapproche du symptome jusqu au
	// 26/08 : la substitution ne touche que les cellules a moins de 25 m d une ancre, alors que
	// le cadre va bien plus loin. Sur une carte dont les ancres sont groupees, les toits
	// restent intacts des qu on s en eloigne. Elle ne VIDE jamais — contrairement a
	// l ecretage — donc elle ne peut pas percer le sol d une carte a plusieurs niveaux.
	SubstitutionSansPortee bool
	// CombleTrous pose un APLAT de sol suppose sur les cellules sans matiere qui tombent dans
	// les zones nommees. Ce n est PAS une mesure : c est un aplat assume, peint autrement et
	// compte au sidecar. Exige `ZonesNommees`.
	CombleTrous bool
	// PlancherTranche : profondeur, en metres SOUS le niveau de jeu, en deca de laquelle la
	// matiere n appartient plus a la carte. ZERO = `TrancheDeJeuMin` (-12 m).
	//
	// POURQUOI UN REGLAGE PAR CARTE (2026-08-26, Chasm). -12 m couvre les sous-sols d une
	// arene ordinaire, mais sur une carte a gouffre il laisse entrer LE FOND DU GOUFFRE : une
	// forme qui traverse la carte de part en part, qu aucun joueur n atteint sans mourir.
	// Remonter le plancher la fait sortir de la tranche, sans toucher a la geometrie jouee.
	PlancherTranche float64
	// PlafondTranche : hauteur, en metres AU-DESSUS du niveau de jeu, au-dela de laquelle la
	// matiere n est meme pas PROJETEE. Zero = la tranche par defaut (+28 m).
	//
	// A ne pas confondre avec l ecretage : celui-ci choisit, pixel par pixel, parmi les
	// surfaces DEJA dessinees ; la tranche, elle, ecarte la geometrie avant le rendu. Sur une
	// carte entierement couverte — Isolation, 93,9 pour cent, une arene sous voute — l ecretage
	// ne peut rien quand la voute descend jusqu au sol joue, alors qu une tranche basse la
	// supprime par construction.
	PlafondTranche float64
	// ZonesNommees : polygones des callouts de la carte (contours + parties). Fournis, ils
	// sont toujours MESURES (`BilanCuisson.MatiereHorsZones`) ; ils ne rognent que si
	// `RogneAuxZones`. Vides sur une carte sans callouts — toutes les cartes Forge.
	ZonesNommees [][][2]float64
	// RogneAuxZones EFFACE la matiere hors des zones nommees dilatees. Reglage PAR CARTE, a
	// ne poser qu apres avoir regarde le taux mesure.
	RogneAuxZones bool
	// MargeZones : dilatation du masque des zones, en metres. ZERO = `MargeMasqueZones` (4 m).
	// Une marge large garde le mur qui borde une zone ; une marge nulle serre au ras du sol
	// praticable. Reglage par carte : sur une carte dont des structures LONGUES frolent les
	// zones (garde-corps, rebord de gouffre), 4 m suffit a les faire entrer.
	MargeZones float64
	// BoiteUtile borne la matiere a un rectangle MONDE (minX, minY, maxX, maxY). Zero = pas de
	// bornage.
	//
	// C EST LE LEVIER MANUEL, et il est assume comme tel : quand aucun critere derive des
	// fichiers ne separe ce qu on veut garder de ce qu on veut retirer, on trace la boite a la
	// main. Il ne se justifie que par sa raison ecrite et il ne doit jamais devenir la voie
	// normale — un rectangle ne connait pas la carte.
	BoiteUtile [4]float64
	// Echelle est le cote d'un pixel du fond, en metres. ZERO = `EchelleFondCarte`.
	//
	// POURQUOI CE N'EST PLUS UNE CONSTANTE (2026-08-26). Le cadre est propre a chaque carte,
	// donc a echelle FIXE une petite arene rend une petite image : mesure du jour, la matiere
	// d'Aquarius n'occupe que 506 x 336 px. Agrandie a l'ecran, elle pixelise. L'echelle est
	// donc un reglage PAR CARTE — comme le style — et la regle qui la choisit se derive d'une
	// taille utile minimale, elle ne se decrete pas carte par carte.
	//
	// Toute valeur autre que `EchelleFondCarte` change le calage publie : le sidecar porte
	// `metersPerPixel`, les lecteurs s'y fient, et le banc de non-regression compare a
	// 0,0920 m/px (`TestEchelleDeProductionEgaleCelleDuBanc`).
	Echelle float64
	// CibleCadrePx demande une echelle AUTOMATIQUE quand Echelle est nulle : voir
	// EchellePourCadre (echelle_cible.go). Zero = echelle de production.
	CibleCadrePx int
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
	// CellulesEcretees : pixels VIDES par l ecretage des toits (voie par carte). Zero partout
	// ailleurs — la voie de reference ne supprime jamais de matiere.
	CellulesEcretees int
	// MatiereHorsZones : cellules de matiere qui tombent HORS des zones nommees dilatees.
	// MESURE, publiee meme quand on ne rogne pas — c est le chiffre qui autorise ou interdit
	// le rognage. CellulesHorsZones est ce qui a ete reellement efface.
	MatiereHorsZones  int
	CellulesHorsZones int
	// CellulesHorsBoite : matiere effacee par le bornage MANUEL (BoiteUtile).
	CellulesHorsBoite int
	// CellulesSolSuppose : cellules comblees par un APLAT, pas relevees. Publie au sidecar.
	CellulesSolSuppose int
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
	// ObjetsExclus : objets Forge ecartes par TypesExclus.
	ObjetsExclus int
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
	r := CadreSurAncresEchelle(opts.Ancres, EchellePourCadre(opts.Ancres, opts.Echelle, opts.CibleCadrePx))
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	// La tranche est TRANSLATEE AU SOL JOUE — meme regle que la chaine Forge, et pour la meme
	// raison. Justification chiffree : cf. `TrancheDeJeu` (rendu.go).
	min, max := TrancheDeJeu(zJeu)
	if opts.PlancherTranche < 0 {
		min = zJeu + opts.PlancherTranche
	}
	if opts.PlafondTranche > 0 {
		max = zJeu + opts.PlafondTranche
	}
	r.Tranche(min, max)
	r.NiveauDeJeu(zJeu)
	// La voie de reference contre les TOITS : armee avant de projeter, decidee juste apres —
	// et AVANT la frontiere, pour que celle-ci efface sur l'image finale.
	s := NewSurfaceReference(opts.Ancres)
	r.ArmeReference(s)

	if err := peupleDepuisModule(ctx, r, &b, opts); err != nil {
		return nil, b, err
	}
	if opts.EcreteToits {
		// Voie PAR CARTE, declaree en donnee : elle SUPPRIME de la matiere, ce que la voie de
		// reference ne fait jamais. Les deux liberent les memes buffers, elles ne se cumulent pas.
		b.TauxCouverture, b.CellulesSubstituees, b.CellulesEcretees = r.EcretteToits(s, opts.PlafondArene)
		b.CarteCouverte = b.TauxCouverture > SeuilCarteCouverte
	} else {
		b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s, opts.SubstitutionSansPortee)
	}
	appliqueFrontiere(ctx, r, &b, opts, zJeu)
	mesureEtRogneZones(ctx, r, &b, opts)
	borneALaBoite(ctx, r, &b, opts.BoiteUtile)
	JugeParLesAncres(r, &b, opts.Ancres)
	if !opts.SansEau {
		PoseEauDepuisModule(ctx, r, &b, opts.CheminModule)
	}
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
	return CadreSurAncresEchelle(ancres, EchelleFondCarte)
}

// CadreSurAncresEchelle est `CadreSurAncres` a une echelle choisie (cote du pixel, en metres).
//
// Le CADRE MONDE est le meme — seule la finesse de la grille change, donc la taille en pixels
// de l'image et son `metersPerPixel` publie. Deux fonds de la meme carte a deux echelles se
// superposent donc exactement une fois remis a l'echelle : c'est ce qui rend le reglage
// comparable au gate.
func CadreSurAncresEchelle(ancres [][3]float64, cell float64) *Rendu {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, a := range ancres {
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], a[k]-MargeCadre)
			hi[k] = math.Max(hi[k], a[k]+MargeCadre)
		}
	}
	return NewRendu(lo, hi, cell)
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
			a = ouvreAsset(ctx, idx, id, GroupeRtgo)
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

// ouvreAsset extrait et decode un tag de geometrie (`rtgo`, ou `mode` — lot B Forge). Un tag
// illisible est LOGGE puis saute — une carte ne doit pas mourir sur un maillage, mais une
// extraction muette masquerait un trou.
func ouvreAsset(ctx context.Context, idx *ModuleIndex, id uint32, groupe string) *RuntimeGeoAsset {
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		slog.DebugContext(ctx, "tag de geometrie illisible", "groupe", groupe, "id", fmt.Sprintf("%08x", id), "err", err)
		return nil
	}
	var a *RuntimeGeoAsset
	if groupe == GroupeMode {
		a, err = NewRenderModelAsset(tag, blob)
	} else {
		a, err = NewRuntimeGeoAsset(tag, blob)
	}
	if err != nil {
		slog.DebugContext(ctx, "tag de geometrie indecodable", "groupe", groupe, "id", fmt.Sprintf("%08x", id), "err", err)
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

// mesureEtRogneZones mesure la matiere hors des zones nommees, et ne l'efface que si la carte
// le demande. La MESURE est inconditionnelle : c'est elle qui dit si le rognage est defendable
// sur cette carte, et un chiffre qu'on ne publie pas est un chiffre que personne ne regarde.
func mesureEtRogneZones(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuisson) {
	if len(opts.ZonesNommees) == 0 {
		return
	}
	marge := MargeMasqueZones
	if opts.MargeZones > 0 {
		marge = opts.MargeZones
	} else if opts.MargeZones < 0 {
		marge = 0 // negatif = demande explicite de NE PAS dilater
	}
	m := MasqueZones(r, opts.ZonesNommees, marge)
	matiere, dehors := r.MesureHorsZones(m)
	b.MatiereHorsZones = dehors
	part := 0.0
	if matiere > 0 {
		part = float64(dehors) / float64(matiere)
	}
	slog.InfoContext(ctx, "mapfond: matiere hors des zones nommees", "carte", b.Module,
		"zones", len(opts.ZonesNommees), "matiere", matiere, "dehors", dehors,
		"part", fmt.Sprintf("%.1f%%", 100*part), "rogne", opts.RogneAuxZones)
	if opts.RogneAuxZones {
		b.CellulesHorsZones = r.EffaceHorsZones(m)
	}
	if opts.CombleTrous {
		b.CellulesSolSuppose = r.CombleTrous(m)
		slog.InfoContext(ctx, "mapfond: sol suppose pose sur les trous des zones", "carte", b.Module,
			"cellules", b.CellulesSolSuppose)
	}
}

// borneALaBoite efface la matiere hors du rectangle monde declare, s'il l'est. LEVIER MANUEL :
// il est journalise avec ses bornes, pour qu'un fond borne a la main se reconnaisse au premier
// coup d'oeil dans les logs.
func borneALaBoite(ctx context.Context, r *Rendu, b *BilanCuisson, bo [4]float64) {
	if bo[2] <= bo[0] || bo[3] <= bo[1] {
		return
	}
	efface := 0
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := 0; i < r.NX; i++ {
			k := j*r.NX + i
			vide := math.IsInf(r.z[k], -1)
			if vide && (r.solSuppose == nil || !r.solSuppose[k]) {
				continue
			}
			x := r.Min[0] + (float64(i)+0.5)*r.Cell
			if x >= bo[0] && x <= bo[2] && y >= bo[1] && y <= bo[3] {
				continue
			}
			r.z[k] = math.Inf(-1)
			// Le SOL SUPPOSE se peint sans matiere : sans cette ligne, un aplat pose hors de la
			// boite y resterait dessine et le bornage n aurait aucun effet visible (mesure du
			// 2026-08-26 sur Catalyst : 44 331 cellules effacees, image quasi inchangee).
			if r.solSuppose != nil {
				r.solSuppose[k] = false
			}
			efface++
		}
	}
	b.CellulesHorsBoite = efface
	slog.InfoContext(ctx, "mapfond: matiere bornee a la boite manuelle", "carte", b.Module,
		"boite", fmt.Sprintf("[%.1f %.1f %.1f %.1f]", bo[0], bo[1], bo[2], bo[3]), "effacees", efface)
}
