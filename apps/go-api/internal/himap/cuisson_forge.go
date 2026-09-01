// Package himap — cuisson_forge.go : LA CHAINE DE PRODUCTION D'UNE CARTE FORGE.
//
// LA DIFFERENCE N'EST PAS « Forge ou pas ». Cliffhanger aussi a ete concue dans Forge, mais 343
// l'a CUITE dans un module dedie : 10 223 instances de geometrie contre 443 objets de variante.
// Vagabond, elle, n'est pas cuite — 788 instances dans son canevas contre 4 709 objets dans son
// `.mvar`. Sa carte est le RACK D'OBJETS, pas le module.
//
// LA CHAINE, etablie sur pieces le 2026-08-10 (sondes `sonde_forge_gamefiles_test.go`) :
//
//	objet .mvar --type_id--> tag `food` (GlobalID, forge_objects-rtx-new.module)
//	           --refs inline--> tags `rtgo` (les MEMES maillages que la chaine sbsp)
//	           --Pos/Up/Forward--> repere monde (Left = Up x Forward, base orthonormee)
//
// CE QUI FONDE CHAQUE MAILLON, mesure et non suppose :
//   - type_id -> food : 467/468 type_id de Vagabond se resolvent ;
//   - food -> rtgo : les deps des tags food sont VIDES (457/467) et root+0x08 est
//     l'auto-reference — le lien est INLINE. 374 type_id portent au moins une ref rtgo,
//     couvrant 3 558 des 4 697 objets (75,7 %). Les 93 restants passent par `bloc`/`scen`/`mach`
//     (963/173/9 objets) : le SAUT `bloc`/`scen`/`mach` -> `hlmt` -> `rtgo` les resout (lot B,
//     2026-08-13, mesure : `sonde_forge_saut_gamefiles_test.go`) ;
//   - echelle : AUCUNE dans le `.mvar` de Vagabond. MESURE, pas suppose — le champ objet [6]
//     n'existe pas et [9] est une struct VIDE sur 4 709/4 709 objets. Le piege de l'echelle
//     d'instance sbsp, paye deux jours, a ete verifie ici sur pieces.
//
// CE QUE CETTE CHAINE NE FAIT PAS, et c'est assume : elle ne rend pas la TOILE du canevas sous
// les objets (`fo08_wetland`), et n'applique ni frontiere de mort ni eau — les cartes Forge
// declarent leurs limites dans leurs propres objets, pas dans un tag `sddt`. Au registre.
//
// Les DECLARATIONS des cartes (map_id, canevas, `.mvar`) vivent dans cartes_forge.go.
package himap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
	"levelup/go-api/internal/hinavmesh"
)

// TypesVolumesDeMort : empreinte fonctionnelle des volumes de mort, etablie sur 101 `.mvar`
// (INVESTIGATION_VOLUMES_MORT_MVAR_2026-08-10.md §2).
//
// Ils sont exclus PAR CONSTRUCTION (ils n'ont aucune ref rtgo) et COMPTES explicitement, pour
// que l'exclusion soit un fait mesure et non une consequence heureuse.
var TypesVolumesDeMort = map[int32]string{
	-588988541:  "volume de mort principal",
	176825834:   "plancher de mort",
	937132837:   "volume de mort sous-sol",
	-1751270658: "mur de limite",
}

// ErrSansObjetForge signale un `.mvar` sans objet exploitable.
var ErrSansObjetForge = errors.New("himap: aucun objet Forge a poser")

// OptionsCuissonForge decrit une carte Forge a cuire.
type OptionsCuissonForge struct {
	// RacineDeploy est la racine `deploy` de l'installation.
	RacineDeploy string
	// Objets sont les objets poses de la variante de carte (`.mvar` deja lu).
	Objets []mapvar.Object
	// Ancres sont les positions monde des objectifs de la carte.
	Ancres [][3]float64
	// CheminModuleCanevas est le `.module` du canevas (ex. fo08_wetland), ajoute a l'index
	// pour que ses `rtgo` propres soient resolus. Vide = canevas inconnu, on s'en passe.
	CheminModuleCanevas string
	// Cle est le nom sous lequel l'asset sera publie (cf. BilanCuisson.Module) : le map_id
	// de la carte (cartes_forge.go).
	Cle string
	// Echelle est le cote d'un pixel du fond, en metres. ZERO = `EchelleFondCarte`.
	// Meme reglage et memes consequences que `OptionsCuisson.Echelle` — voir cuisson.go.
	Echelle float64
	// CibleCadrePx : meme role que OptionsCuisson.CibleCadrePx.
	CibleCadrePx int
	// LES LEVIERS DE RENDU, jusqu ici reserves a la chaine native (2026-08-27). Une carte
	// Forge n en avait AUCUN : declarer un ecretage pour Isolation ne changeait pas un octet
	// de son image, et rien ne le disait. Ils ont la meme semantique que dans OptionsCuisson.
	EcreteToits            bool
	PlafondArene           float64
	SubstitutionSansPortee bool
	BoiteUtile             [4]float64
	// RogneAuxVolumesDeMort borne la matiere a l emprise des VOLUMES DE MORT de la variante.
	// C est l equivalent Forge du rognage aux zones de callout : les callouts disent ou l on
	// joue, les volumes de mort disent ou l on meurt, et les 22 cartes a callouts sont toutes
	// natives. Sans effet si la variante n en declare aucun avec une forme.
	RogneAuxVolumesDeMort bool
	// DrapeauxExclus ecarte les objets Forge dont le champ de DRAPEAUX (champ 7 du .mvar,
	// Object.Flags) vaut l une de ces valeurs. Jamais lu jusqu au 2026-08-27.
	//
	// Ce qui a mis dessus : l utilisateur suppose que les coques ne sont pas des blocs mais
	// « un genre de mur transparent » ou un effet. Mesure sur Isolation : sur 5 042 objets, le
	// drapeau vaut 21 pour 4 384 d entre eux — et 1 pour les 32 pieces du dome, qui peignent
	// 82,7 pour cent de l image. 344 objets en tout portent cette valeur.
	DrapeauxExclus map[uint8]bool
	// PlafondObjets ecarte les objets Forge POSES plus haut que ce nombre de metres au-dessus
	// du sol joue. Zero = n en ecarter aucun.
	//
	// A NE PAS CONFONDRE AVEC L ECRETAGE NI AVEC LA TRANCHE, et c est la distinction qui a
	// coute une journee le 2026-08-27. Les deux autres coupent des SURFACES par leur altitude ;
	// celle-ci ecarte un OBJET par l altitude ou il est POSE. La difference decide tout quand
	// l objet est un DOME : sa paroi descend jusqu au sol, donc toute coupe de surface en laisse
	// passer la jupe, alors que son origine, elle, est franchement au-dessus de l arene.
	//
	// Mesure sur Isolation : les 32 exemplaires du type qui peint 82,7 pour cent de l image sont
	// poses entre Z 136,0 et 160,6 quand le sol joue est a Z 117 — de 19 a 44 m au-dessus.
	PlafondObjets float64
	// NavmeshReference, s il est fourni, sert de SURFACE DE REFERENCE a la place de
	// l interpolation des ancres. Le rendu reste celui de la geometrie ordinaire — on ne
	// change que ce a quoi on compare les surfaces. C est ce qui rend a une carte a ciel
	// ferme ses structures, la ou le fond tire du seul navmesh ne montrait que le sol nu.
	NavmeshReference *hinavmesh.Maillage
	// RogneAuNavmesh efface la matiere hors du maillage de navigation, dilate de MargeNavmesh.
	RogneAuNavmesh bool
	// MargeNavmeshCarte : dilatation du masque de rognage au maillage, en metres. ZERO =
	// `MargeNavmesh` (3 m).
	//
	// La marge existe pour garder les MURS qui bordent le sol : le maillage se retire des parois,
	// et rogner au ras rendrait une carte sans ses cloisons. Mais 3 m gardent aussi ce qui borde
	// le sol sans en etre — un rocher pose contre l arene, une falaise. La resserrer est le seul
	// levier qui distingue ces masses du terrain, apres que trois autres ont ete refutes sur
	// Shogun le 2026-08-30 : la tranche de hauteur (0 ancre sur 13), l exclusion par type
	// (0 ancre sur 13 — les gros types SONT le sol) et l altitude de pose (le point de pose d une
	// piece Forge n est pas la ou sa surface se dessine).
	MargeNavmeshCarte float64
	// ToleranceNavmesh vide les cellules dont la surface retenue s ecarte de plus de N metres
	// du sol donne par le maillage. Zero = ne rien vider.
	ToleranceNavmesh float64
	// SolVuDuDessous : retenir la surface la plus BASSE au-dessus du sol joue au lieu de la plus
	// haute. Voir Rendu.ArmeSurfaceBasse — la reponse aux cartes a ciel ferme.
	SolVuDuDessous bool
	// MargeSolVuDuDessousCarte : de combien on descend sous le niveau de jeu pour accepter une
	// surface comme candidate. ZERO = 4 m, qui EXCLUENT deliberement un sous-sol. L elargir le
	// fait entrer. Voir MargeSolVuDuDessousCarte.
	MargeSolBas float64
	// RogneAuxAltitudesProches efface ce qui s ecarte trop du niveau de jeu, en gardant une marge
	// autour de ce qui reste. Voir altitude_proche.go — l idee vient de l utilisateur : garder les
	// bords clairs et couper ce qui est dehors, la clarte etant un thermometre d altitude.
	RogneAuxAltitudesProches bool
	// SeuilAltitude, MargeAltitude : en metres. Zero = les valeurs de production (6 m et 4 m).
	SeuilAltitude float64
	MargeAltitude float64
	// MinceurMin ecarte les modeles FILAIRES : ceux dont l aire du maillage, rapportee au
	// carre de leur emprise, tombe sous ce seuil. Zero = ne rien ecarter.
	//
	// C est la reponse au « gribouillis » : isoler un seul type d objet a montre le
	// 2026-08-27 que le plus nombreux d Isolation — 349 exemplaires — dessine de longues
	// BRANCHES. Multipliees par des centaines, ce sont elles qui couvrent l arene de traits.
	// Un premier essai avait echoue en classant les types par EMPRISE : ca attrape les gros
	// rochers et laisse passer les branches, longues mais minuscules en matiere. La minceur,
	// elle, les separe — une branche de 8 m d envergure porte quelques dixiemes de metre
	// carre, un rocher de meme emprise en porte des dizaines.
	MinceurMin float64
	// TypesExclus ecarte des TYPES d objet Forge du dessin. Dernier recours, quand un modele
	// balaie la carte et qu aucune coupe geometrique ne peut l atteindre — voir le diagnostic
	// « types les plus etendus » journalise a chaque cuisson.
	TypesExclus map[int32]bool
	// DessineCanevas pose AUSSI la geometrie du canevas, avant les objets de la variante.
	//
	// L etat de l art disait qu un canevas ne porte aucune instance — vrai pour fo11_blank
	// (0 instance, mesure), FAUX pour fo08_wetland qui en porte 13 281 sur son bsp lointain et
	// 814 sur son bsp d ile. Une carte Forge batie SUR le terrain du canevas etait donc rendue
	// sans son sol. Question posee par l utilisateur le 2026-08-27.
	DessineCanevas bool
	// PlancherTranche / PlafondTranche : memes roles que dans OptionsCuisson.
	PlancherTranche float64
	PlafondTranche  float64
	// SeuilArete : voir Rendu.SeuilArete.
	SeuilArete float64
	// RogneAuxZones EFFACE la matiere hors des zones de callout de la carte, dilatees de
	// `MargeZones`. Les zones sont lues dans les OBJETS de la variante (`ZonesNommeesForge`) —
	// rien a fournir, rien a telecharger. La mesure, elle, est INCONDITIONNELLE : elle seule
	// dit si le rognage est defendable sur cette carte.
	//
	// A ARBITRER CONTRE `RogneAuNavmesh`, qui rogne au maillage de navigation. Les deux
	// disent « ou l on joue » par deux sources independantes ; ils se cumulent sans se
	// contredire, mais une carte n a en general besoin que d un des deux.
	RogneAuxZones bool
	// MargeZones : dilatation du masque des zones, en metres. ZERO = `MargeMasqueZones` (4 m),
	// negatif = ne pas dilater du tout.
	MargeZones float64
	// CombleTrous peint un SOL SUPPOSE dans les trous INTERIEURS de la carte : les cellules vides
	// que l on ne peut pas atteindre depuis le bord de l image. C est la definition topologique de
	// « dedans », et c est exactement ce que l utilisateur a demande le 2026-08-30 — un fond plein
	// a l interieur de la forme, pas au-dela.
	//
	// POURQUOI CA MANQUAIT AUX CARTES FORGE. La chaine native comble depuis le 26/08, mais bornee
	// au masque des zones de callout, que les cartes Forge n avaient pas. Elles l ont depuis le
	// 27/08 ; ici on ne s en sert MEME PAS comme borne : un trou peut tomber dans la silhouette
	// sans tomber dans une zone nommee, et il faut le combler quand meme.
	//
	// SA SEULE FAIBLESSE, ET ELLE ECHOUE DU BON COTE : si le contour de la carte est ouvert, le
	// remplissage depuis le bord atteint l interieur et rien n est comble. On ne peint alors rien
	// plutot que de peindre dehors.
	CombleTrous bool
	// CombleAuMaillage peint le sol suppose partout ou le MAILLAGE dit qu on marche et ou la
	// geometrie n a rien dessine. C est l autre definition de « la forme de la carte », et la
	// seule qui rende quelque chose sur une silhouette en dentelle (voir CombleDansLeMaillage).
	CombleAuMaillage bool
	// RogneAuxComposantesAncrees efface les amas de matiere qui ne portent aucune ancre et n en
	// touchent aucun qui en porte : du decor pose hors de la silhouette jouee. Voir
	// composantes.go. Levier PAR CARTE — une carte dont les ancres ne couvrent pas tous ses
	// ilots joues y perdrait du terrain.
	RogneAuxComposantesAncrees bool
	// CadreAuxZones borne l image a l emprise des ZONES DE CALLOUT, elargie de
	// `MargeCadreZones`. A ne pas confondre avec `RogneAuxZones`, qui efface la matiere hors des
	// zones : celui-la decide ce qu on efface, celui-ci ce qu on montre. Il repond aux cartes
	// dont le cadre atteint la butee alors que le jeu tient dans un coin.
	CadreAuxZones bool
	// CadreAuxAncres borne l image a l emprise des ANCRES D OBJECTIFS, elargie de
	// `MargeCadreAncres`. Recours quand les zones de callout sont trop grossieres pour cadrer
	// (elles couvrent le canevas entier sur plusieurs cartes) et que le maillage manque ou ne se
	// lit pas. Voir BoiteDesAncres.
	CadreAuxAncres bool
	// MargeCadreAncres : marge autour des ancres, en metres. ZERO = `MargeCadreAncres` (25 m).
	//
	// REGLABLE PAR CARTE parce que 25 m ne veut pas dire la meme chose partout : sur Shogun les
	// ancres tiennent dans 31 x 30 m alors que l image en fait 93 x 78 — la marge par defaut
	// rendait donc presque toute l image, et le decor du canevas restait. Sur une carte dont les
	// objectifs sont alignes (Outlook : 31 x 3 m), une marge serree couperait au contraire
	// l arene. Le chiffre se choisit apres avoir regarde.
	MargeAncres float64
	// MaillageNiveauHaut : prendre le niveau le PLUS HAUT du maillage comme reference la ou deux
	// niveaux se superposent, pour que les etages et les passerelles survivent a la substitution.
	// Voir NiveauHautNavmesh.
	MaillageNiveauHaut bool
	// SansSubstitution laisse la surface HAUTE telle qu elle a ete dessinee, au lieu de la
	// remplacer par celle qui est la plus proche de la reference.
	//
	// A QUOI CA SERT. La substitution est ce qui fait tomber les coques et les domes sur les
	// cartes couvertes ; c est elle qui a rendu Isolation lisible. Mais elle ne distingue pas un
	// dome d une STRUCTURE EN HAUTEUR sur laquelle on ne marche pas — un mur, un auvent, un bloc
	// plein. Prendre le niveau haut du maillage sauve les etages PRATICABLES ; ce qui n est pas
	// praticable reste rabattu. Sur les cartes ou l utilisateur veut voir ces volumes-la, il faut
	// renoncer a la substitution — et accepter que les toits reviennent avec eux.
	//
	// A n armer qu apres avoir regarde : sur une carte sous voute, ca rend l image illisible.
	SansSubstitution bool
	// SeuilSubstitution : ecart minimal en metres au-dessus de la reference pour qu une surface
	// soit rabattue. Zero = rabattre des qu il y a un ecart. Voir SeuilSubstitution.
	SeuilSubstitution float64

	// SeuilCouverture : seuil de carte couverte propre a la carte, en part de matiere.
	// Zero = `SeuilCarteCouverte`. Voir SeuilCouvertureCarte pour la justification.
	SeuilCouverture float64

	// PositionsJouees : positions monde des joueurs, tirees des rejeux decodes. Vide = le levier
	// ne fait rien. Voir positions_jouees.go.
	PositionsJouees []PositionJouee
	// RayonPositions : rayon de garde autour d une position courue, en metres. ZERO =
	// `RayonPositionsJouees` (4 m).
	RayonPositions float64
	// SeuilRecollement : part de la surface d un objet que le masque doit garder pour que l objet
	// survive ENTIER. ZERO = `SeuilRecollement` (un tiers) ; NEGATIF = recollement desarme.
	SeuilRecollement float64
}

// CuitCarteForge rend le fond de carte d'une carte Forge en posant les modeles de ses objets.
func CuitCarteForge(ctx context.Context, opts OptionsCuissonForge) (*Rendu, BilanCuisson, error) {
	b := BilanCuisson{Module: opts.Cle, Ancres: len(opts.Ancres), ObjetsForge: len(opts.Objets)}
	if len(opts.Ancres) == 0 {
		return nil, b, ErrSansAncre
	}
	if len(opts.Objets) == 0 {
		return nil, b, ErrSansObjetForge
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		return nil, b, err
	}

	r := CadreSurAncresEchelle(opts.Ancres, EchellePourCadre(opts.Ancres, opts.Echelle, opts.CibleCadrePx))
	r.SeuilArete = opts.SeuilArete
	zJeu := armeTrancheDeJeuForge(r, &b, opts)
	// MEME regle que la chaine native : la voie de reference contre les toits
	// (rendu_reference.go). Une carte Forge a ciel ouvert reste sous le seuil et n'est pas
	// touchee ; la regle est universelle, pas une affaire de chaine.
	precedentSeuil := SeuilSubstitution
	SeuilSubstitution = opts.SeuilSubstitution
	defer func() { SeuilSubstitution = precedentSeuil }()
	precedentCouverture := SeuilCouvertureCarte
	SeuilCouvertureCarte = opts.SeuilCouverture
	defer func() { SeuilCouvertureCarte = precedentCouverture }()
	s := NewSurfaceReference(opts.Ancres)
	if opts.NavmeshReference != nil {
		precedent := NiveauHautNavmesh
		NiveauHautNavmesh = opts.MaillageNiveauHaut
		couvertes := r.ArmeReferenceDepuisNavmesh(opts.NavmeshReference)
		NiveauHautNavmesh = precedent
		slog.InfoContext(ctx, "mapfond: reference prise sur le maillage de navigation",
			"carte", opts.Cle, "cellules", couvertes)
	} else {
		r.ArmeReference(s)
	}

	r.ArmeTypeGagnant()
	r.ArmeObjetGagnant()
	if opts.SolVuDuDessous {
		precedentBas := MargeSolVuDuDessousCarte
		MargeSolVuDuDessousCarte = opts.MargeSolBas
		defer func() { MargeSolVuDuDessousCarte = precedentBas }()
		r.ArmeSurfaceBasse()
	}
	if opts.DessineCanevas {
		poseCanevasForge(ctx, r, &b, opts)
	}
	poseObjetsForge(ctx, r, &b, idx, forge, opts, zJeu)
	if opts.PlafondObjets > 0 {
		slog.InfoContext(ctx, "mapfond: objets poses au-dessus du plafond ecartes", "carte", opts.Cle,
			"plafond", opts.PlafondObjets, "objets", b.ObjetsAuPlafond)
	}
	if b.ObjetsDessines == 0 {
		return nil, b, fmt.Errorf("aucun des %d objets Forge n'a de modele rtgo", len(opts.Objets))
	}
	if opts.SolVuDuDessous {
		n := r.AdopteSurfaceBasse()
		slog.InfoContext(ctx, "mapfond: sol vu du dessous", "carte", opts.Cle, "pixels", n)
	}
	appliqueSubstitutionForge(ctx, r, &b, s, opts)
	rogneAuNavmeshForge(ctx, r, opts)
	mesureEtRogneZonesForge(ctx, r, &b, opts)
	combleTrousForge(ctx, r, &b, opts)
	rogneAltitudesEtPositionsForge(ctx, r, opts)
	cadreAuxAncresEtZonesForge(ctx, r, opts)
	borneALaBoite(ctx, r, &b, boiteForge(ctx, opts))
	if b.VolumesDeMort == 0 {
		b.degrade(ctx, "aucun volume de mort reconnu — l'empreinte des types a peut-etre bouge")
	}
	JugeParLesAncres(r, &b, opts.Ancres)
	journalisePixelsParType(ctx, r, b, opts.Objets)
	slog.InfoContext(ctx, "carte Forge cuite", "cle", b.Module,
		"objets", fmt.Sprintf("%d/%d", b.ObjetsDessines, b.ObjetsForge),
		"sansModele", b.ObjetsSansModele, "volumesDeMort", b.VolumesDeMort,
		"couverture", fmt.Sprintf("%.1f%%", 100*b.TauxCouverture), "couverte", b.CarteCouverte,
		"ancres", fmt.Sprintf("%d/%d", b.AncresAvecSol, b.AncresDansLeCadre))
	return r, b, nil
}

// armeTrancheDeJeuForge deduit le niveau de jeu des ancres, arme la tranche d'altitude du rendu
// et note ce niveau au bilan. Rend le niveau de jeu. Extrait de CuitCarteForge, a l'identique.
//
// LA TRANCHE EST TRANSLATEE AU SOL DES ANCRES. Le sol de Vagabond vit vers z=52 : c'est ici
// qu'on a compris qu'une tranche absolue n'avait pas de sens, et la chaine native applique
// desormais la meme regle (cf. `TrancheDeJeu`).
func armeTrancheDeJeuForge(r *Rendu, b *BilanCuisson, opts OptionsCuissonForge) float64 {
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	minT, maxT := TrancheDeJeu(zJeu)
	if opts.PlancherTranche < 0 {
		minT = zJeu + opts.PlancherTranche
	}
	if opts.PlafondTranche > 0 {
		maxT = zJeu + opts.PlafondTranche
	}
	r.Tranche(minT, maxT)
	r.NiveauDeJeu(zJeu)
	return zJeu
}

// appliqueSubstitutionForge arbitre entre les trois voies de traitement des toits et renseigne
// le taux de couverture du bilan. Extrait de CuitCarteForge, a l'identique.
func appliqueSubstitutionForge(ctx context.Context, r *Rendu, b *BilanCuisson,
	s *SurfaceReference, opts OptionsCuissonForge) {
	switch {
	case opts.SansSubstitution:
		b.TauxCouverture = r.TauxCouvertureMesure()
		b.CarteCouverte = b.TauxCouverture > SeuilCouvertureEffectif()
		slog.InfoContext(ctx, "mapfond: substitution NON appliquee sur demande", "carte", opts.Cle,
			"couverture", fmt.Sprintf("%.1f%%", 100*b.TauxCouverture))
	case opts.EcreteToits:
		b.TauxCouverture, b.CellulesSubstituees, b.CellulesEcretees = r.EcretteToits(s, opts.PlafondArene)
		b.CarteCouverte = b.TauxCouverture > SeuilCouvertureEffectif()
	default:
		b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s, opts.SubstitutionSansPortee)
	}
}

// rogneAuNavmeshForge applique les deux rognages qui prennent le maillage de navigation pour
// juge : hors maillage, puis loin du sol qu'il donne. Extrait de CuitCarteForge, a l'identique.
func rogneAuNavmeshForge(ctx context.Context, r *Rendu, opts OptionsCuissonForge) {
	if opts.RogneAuNavmesh && opts.NavmeshReference != nil {
		marge := MargeNavmesh
		if opts.MargeNavmeshCarte > 0 {
			marge = opts.MargeNavmeshCarte
		}
		n := r.EffaceHorsNavmesh(marge)
		slog.InfoContext(ctx, "mapfond: matiere effacee hors du maillage de navigation", "carte", opts.Cle, "cellules", n, "marge", marge)
	}
	if opts.ToleranceNavmesh > 0 {
		n := r.EffaceLoinDuNavmesh(opts.ToleranceNavmesh)
		slog.InfoContext(ctx, "mapfond: surfaces loin du sol vidées", "carte", opts.Cle, "tolerance", opts.ToleranceNavmesh, "cellules", n)
	}
}

// rogneAltitudesEtPositionsForge applique les deux rognages qui prennent le JEU pour juge : la
// distance au niveau de jeu, puis la distance aux positions courues. Extrait de CuitCarteForge,
// a l'identique.
func rogneAltitudesEtPositionsForge(ctx context.Context, r *Rendu, opts OptionsCuissonForge) {
	if opts.RogneAuxAltitudesProches {
		seuil, marge := SeuilAltitudeProche, MargeAltitudeProche
		if opts.SeuilAltitude > 0 {
			seuil = opts.SeuilAltitude
		}
		if opts.MargeAltitude > 0 {
			marge = opts.MargeAltitude
		}
		n := r.RogneAuxAltitudesProches(seuil, marge)
		slog.InfoContext(ctx, "mapfond: matiere loin du niveau de jeu effacee", "carte", opts.Cle,
			"seuil", seuil, "marge", marge, "cellules", n)
	}
	if len(opts.PositionsJouees) > 0 {
		rayon := RayonPositionsJouees
		if opts.RayonPositions > 0 {
			rayon = opts.RayonPositions
		}
		seuilRec := SeuilRecollement
		if opts.SeuilRecollement < 0 {
			seuilRec = 0 // negatif = recollement desarme explicitement
		} else if opts.SeuilRecollement > 0 {
			seuilRec = opts.SeuilRecollement
		}
		n := r.RogneAuxPositionsJouees(opts.PositionsJouees, rayon, seuilRec)
		slog.InfoContext(ctx, "mapfond: matiere hors des positions jouees effacee", "carte", opts.Cle,
			"positions", len(opts.PositionsJouees), "rayon", rayon, "cellules", n,
			"seuilRecollement", seuilRec, "objetsRetires", r.RecollementRetires)
	}
}

// cadreAuxAncresEtZonesForge efface les composantes sans ancre, puis borne le cadre a l'emprise
// des ancres et a celle des zones de callout. Extrait de CuitCarteForge, a l'identique : les
// bilans locaux ne servent qu'au journal, seul le bornage final alimente le bilan de cuisson.
func cadreAuxAncresEtZonesForge(ctx context.Context, r *Rendu, opts OptionsCuissonForge) {
	if opts.RogneAuxComposantesAncrees {
		n, gardees, total := r.GardeComposantesAncrees(opts.Ancres)
		slog.InfoContext(ctx, "mapfond: composantes sans ancre effacees", "carte", opts.Cle,
			"cellules", n, "composantes", fmt.Sprintf("%d/%d gardees", gardees, total))
	}
	if opts.CadreAuxAncres {
		marge := MargeCadreAncres
		if opts.MargeAncres > 0 {
			marge = opts.MargeAncres
		}
		if ba, ok := BoiteDesAncres(opts.Ancres, marge); ok {
			var bb BilanCuisson
			borneALaBoite(ctx, r, &bb, ba)
			slog.InfoContext(ctx, "mapfond: cadre borne a l emprise des ancres", "carte", opts.Cle,
				"boite", fmt.Sprintf("%.1f %.1f %.1f %.1f", ba[0], ba[1], ba[2], ba[3]),
				"cellules", bb.CellulesHorsBoite)
		}
	}
	if opts.CadreAuxZones {
		if bz, ok := BoiteDesZones(ZonesNommeesForge(opts.Objets), MargeCadreZones); ok {
			var bz2 BilanCuisson
			borneALaBoite(ctx, r, &bz2, bz)
			n := bz2.CellulesHorsBoite
			slog.InfoContext(ctx, "mapfond: cadre borne a l emprise des zones de callout", "carte", opts.Cle,
				"boite", fmt.Sprintf("%.1f %.1f %.1f %.1f", bz[0], bz[1], bz[2], bz[3]), "cellules", n)
		} else {
			slog.WarnContext(ctx, "mapfond: cadre aux zones demande mais aucune zone lue", "carte", opts.Cle)
		}
	}
}

// indexForge construit l'index des tags : le module des objets Forge d'abord (les `food` et
// leurs `rtgo`), puis le canevas et les globaux.
func indexForge(opts OptionsCuissonForge) (*ModuleIndex, *himodule.Module, error) {
	principal := filepath.Join(opts.RacineDeploy, "any", "globals", "forge", "forge_objects-rtx-new.module")
	chemins := []string{principal}
	if p := filepath.Join(opts.RacineDeploy, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existeFichier(p) {
		chemins = append(chemins, p)
	}
	if opts.CheminModuleCanevas != "" && existeFichier(opts.CheminModuleCanevas) {
		chemins = append(chemins, opts.CheminModuleCanevas)
	}
	globs, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)
	// Les definitions d'objet du SAUT (`bloc`/`scen`/`mach`, lot B) vivent pour partie dans
	// les globals de la variante `any` (globals-rtx-new, common-rtx-new) : sans eux, 17 food
	// de Vagabond ne resolvent aucune definition (sonde du 2026-08-13).
	globsAny, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "any", "globals", "*.module"))
	chemins = append(chemins, globsAny...)

	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		return nil, nil, fmt.Errorf("index des modules Forge : %w", err)
	}
	forge, err := himodule.Open(principal)
	if err != nil {
		return nil, nil, fmt.Errorf("module des objets Forge : %w", err)
	}
	return idx, forge, nil
}

// poseObjetsForge resout le modele de chaque type d'objet, puis pose ses triangles. Les cinq
// reglages qu'il consulte (objets, types exclus, minceur minimale, plafond de pose, drapeaux
// exclus) sont lus dans `opts`, comme dans poseCanevasForge ; seul `zJeu`, qui est calcule et non
// declare, reste un parametre a part.
func poseObjetsForge(ctx context.Context, r *Rendu, b *BilanCuisson,
	idx *ModuleIndex, forge *himodule.Module, opts OptionsCuissonForge, zJeu float64) {
	minceurs := map[uint32]float64{}
	modeleDuType := modeleParType(ctx, opts.Objets, idx, forge)
	assets := map[uint32]*RuntimeGeoAsset{}
	// DIAGNOSTIC DES TYPES ETENDUS. Le « gribouillis » d Isolation ne cede ni a l ecretage ni
	// au bornage : il vit A HAUTEUR DE SOL. Reste une cause possible — un TYPE d objet dont le
	// modele balaie la carte. On mesure donc, une fois par type, l emprise du premier exemplaire
	// pose ; le classement se lit dans les logs et sert a remplir le reglage typesExclus.
	etendues := map[int32]float64{}
	comptes := map[int32]int{}
	defer func() { journaliseTypesEtendus(ctx, b, etendues, comptes) }()
	for i, o := range opts.Objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; mort {
			b.VolumesDeMort++
			continue
		}
		m, ok := modeleDuType[o.TypeID]
		if !ok {
			b.ObjetsSansModele++
			continue
		}
		if modeleTropFilaire(ctx, idx, m, minceurs, opts.MinceurMin) {
			b.ObjetsFilaires++
			continue
		}
		a, deja := assets[m.id]
		if !deja {
			a = ouvreAsset(ctx, idx, m.id, m.groupe)
			assets[m.id] = a
		}
		if a == nil {
			b.ObjetsSansModele++
			continue
		}
		if opts.DrapeauxExclus[o.Flags] {
			b.ObjetsDrapeauExclu++
			continue
		}
		if opts.PlafondObjets > 0 && float64(o.Pos.Z) > zJeu+opts.PlafondObjets {
			b.ObjetsAuPlafond++
			continue
		}
		in := InstanceForge(o)
		r.TypeCourant = o.TypeID
		r.ObjetCourant = int32(i) + 1 // zero = aucune instance
		if _, vu := etendues[o.TypeID]; !vu {
			etendues[o.TypeID] = etendueMondeDe(a, in)
		}
		comptes[o.TypeID]++
		if opts.TypesExclus[o.TypeID] {
			b.ObjetsExclus++
			continue
		}
		for mi := 0; mi < a.MeshCount(); mi++ {
			if mesh := a.Mesh(mi); mesh != nil {
				r.AddMesh(mesh, in)
			}
		}
		b.ObjetsDessines++
	}
}

// modeleTropFilaire dit si le modele d'un type couvre trop peu de son emprise au sol pour etre
// pose (cf. MinceurDuModele). La mesure est mise en cache par modele dans `minceurs` — elle coute
// une ouverture d'asset. Un minceurMin nul ou negatif desarme le filtre. Extrait de
// poseObjetsForge, a l'identique.
func modeleTropFilaire(ctx context.Context, idx *ModuleIndex, m refModele,
	minceurs map[uint32]float64, minceurMin float64) bool {
	if minceurMin <= 0 {
		return false
	}
	mn, connue := minceurs[m.id]
	if !connue {
		if a := ouvreAsset(ctx, idx, m.id, m.groupe); a != nil {
			mn, _ = MinceurDuModele(a)
		}
		minceurs[m.id] = mn
	}
	return mn > 0 && mn < minceurMin
}

// refModele designe le tag de geometrie d'un type d'objet : son GlobalID et son groupe —
// `rtgo` (lecture directe) ou `mode` (render_model, lot B), qui ne s'ouvrent pas pareil.
type refModele struct {
	id     uint32
	groupe string
}

// modeleParType etablit, une fois par type d'objet, le tag de geometrie de son modele : la
// ref `rtgo` directe du `food` d'abord, le saut `bloc`/`scen`/`mach` -> `hlmt` sinon.
func modeleParType(ctx context.Context, objets []mapvar.Object,
	idx *ModuleIndex, forge *himodule.Module) map[int32]refModele {
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	out := map[int32]refModele{}
	vus := map[int32]bool{}
	for _, o := range objets {
		if vus[o.TypeID] {
			continue
		}
		vus[o.TypeID] = true
		f, ok := foodParID[uint32(o.TypeID)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			slog.DebugContext(ctx, "tag food illisible", "typeID", o.TypeID, "err", err)
			continue
		}
		if refs := refsInlineDuGroupe(tag, idx, GroupeRtgo); len(refs) > 0 {
			out[o.TypeID] = refModele{id: refs[0], groupe: GroupeRtgo}
			continue
		}
		if m, ok := modeleParSaut(ctx, idx, tag); ok {
			out[o.TypeID] = m
		}
	}
	return out
}

// GroupeHlmt / GroupeMode : le tag de modele (`model`) et le render_model, maillons du saut.
const (
	GroupeHlmt = "hlmt"
	GroupeMode = "mode"
)

// groupesSautForge : les groupes de definition d'objet Forge SANS ref rtgo directe dans leur
// `food`, dans l'ordre de frequence mesure sur Vagabond — 963 objets via `bloc`, 173 via
// `scen`, 9 via `mach` (sonde F1CouvertureRtgo, 2026-08-10).
var groupesSautForge = []string{"bloc", "scen", "mach"}

// modeleParSaut resout le modele d'un `food` sans ref rtgo directe : la definition d'objet
// passe par un tag `bloc`/`scen`/`mach`, qui reference son modele `hlmt`, lequel porte la
// geometrie — un `rtgo`, ou un `mode` (mesure 2026-08-13 : les 125 hlmt du saut de Vagabond
// ne referencent QUE des `mode`). Meme mecanique a chaque maillon — le scan des octets
// contre l'index, la methode qui a ferme F1 (`sonde_forge_saut_gamefiles_test.go`).
func modeleParSaut(ctx context.Context, idx *ModuleIndex, tagFood []byte) (refModele, bool) {
	for _, groupe := range groupesSautForge {
		for _, hObjet := range refsInlineDuGroupe(tagFood, idx, groupe) {
			objet, err := idx.Extract(hObjet)
			if err != nil {
				slog.DebugContext(ctx, "tag de saut illisible", "groupe", groupe, "id", hObjet, "err", err)
				continue
			}
			if m, ok := modeleDuHlmt(ctx, idx, objet); ok {
				return m, true
			}
		}
	}
	return refModele{}, false
}

// modeleDuHlmt rend la premiere ref de geometrie (`rtgo`, sinon `mode`) portee par les
// modeles `hlmt` d'un tag d'objet.
func modeleDuHlmt(ctx context.Context, idx *ModuleIndex, objet []byte) (refModele, bool) {
	for _, hModele := range refsInlineDuGroupe(objet, idx, GroupeHlmt) {
		hlmt, err := idx.Extract(hModele)
		if err != nil {
			slog.DebugContext(ctx, "tag hlmt illisible", "id", hModele, "err", err)
			continue
		}
		if refs := refsInlineDuGroupe(hlmt, idx, GroupeRtgo); len(refs) > 0 {
			return refModele{id: refs[0], groupe: GroupeRtgo}, true
		}
		if refs := refsInlineDuGroupe(hlmt, idx, GroupeMode); len(refs) > 0 {
			return refModele{id: refs[0], groupe: GroupeMode}, true
		}
	}
	return refModele{}, false
}

// refsInlineDuGroupe rend les GlobalID du groupe donne references dans les octets d'un tag.
func refsInlineDuGroupe(tag []byte, idx *ModuleIndex, groupe string) []uint32 {
	return RefsInline(tag, func(h uint32) bool {
		g, _, ok := idx.Lookup(h)
		return ok && g == groupe
	})
}

// RefsInline rend les GlobalID retenus par le predicat dans les octets d'un tag, par pas de
// 4, dans l'ordre du tag. Le premier est la variante principale — convention etablie au lot 2.
func RefsInline(tag []byte, retient func(uint32) bool) []uint32 {
	var out []uint32
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(u32(tag, o))
		if !vus[h] && retient(h) {
			vus[h] = true
			out = append(out, h)
		}
	}
	return out
}

// InstanceForge construit l'instance de pose d'un objet `.mvar` : base orthonormee
// Forward/Left/Up avec Left = Up x Forward, echelle unitaire (mesuree ABSENTE du format).
func InstanceForge(o mapvar.Object) Instance {
	f := normalise([3]float64{o.Forward.X, o.Forward.Y, o.Forward.Z})
	u := normalise([3]float64{o.Up.X, o.Up.Y, o.Up.Z})
	l := [3]float64{
		u[1]*f[2] - u[2]*f[1],
		u[2]*f[0] - u[0]*f[2],
		u[0]*f[1] - u[1]*f[0],
	}
	return Instance{
		Scale:    [3]float64{1, 1, 1},
		Position: [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z},
		Forward:  f,
		Left:     l,
		Up:       u,
	}
}

func existeFichier(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// BoiteDesVolumesDeMort rend l'emprise XY [minX, minY, maxX, maxY] des volumes de mort d'une
// variante, et le nombre de volumes qui l'ont formee.
//
// POURQUOI. Une carte native se rogne a ses ZONES DE CALLOUT : elles disent ou l'on joue, et
// c'est le levier qui a sauve Streets, Prism ou Forbidden. Une carte FORGE n'en a aucune — les
// 22 cartes a callouts sont toutes natives. Son equivalent est ici : les volumes de mort
// bornent le terrain par l'autre bout, en declarant ou l'on MEURT. Ils sont deja reconnus et
// comptes par la cuisson (`TypesVolumesDeMort`, empreinte etablie sur 101 `.mvar`), et
// jusqu'ici seulement pour etre ECARTES du dessin ; leur position n'avait jamais servi.
//
// La forme d'un objet donne des DEMI-EXTENTS (shape.go) : l'emprise d'un volume est donc son
// centre plus ou moins sa demi-largeur. Un volume sans forme est ignore — il ne borne rien.
func BoiteDesVolumesDeMort(objets []mapvar.Object) (boite [4]float64, n int) {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, o := range objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; !mort {
			continue
		}
		s := o.Shape()
		if s == nil {
			continue
		}
		var demi [2]float64
		switch {
		case s.Radius != nil:
			demi = [2]float64{*s.Radius, *s.Radius}
		case s.HalfX != nil && s.HalfY != nil:
			// La boite est orientee par Forward ; on prend le PIRE cas, la demi-diagonale,
			// plutot que de projeter — un bornage trop large ne retire rien a tort.
			d := math.Hypot(*s.HalfX, *s.HalfY)
			demi = [2]float64{d, d}
		default:
			continue
		}
		c := [2]float64{float64(o.Pos.X), float64(o.Pos.Y)}
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], c[k]-demi[k])
			hi[k] = math.Max(hi[k], c[k]+demi[k])
		}
		n++
	}
	if n == 0 {
		return [4]float64{}, 0
	}
	return [4]float64{lo[0], lo[1], hi[0], hi[1]}, n
}

// boiteForge rend le rectangle monde auquel borner la matiere : celui declare a la main s'il
// l'est, sinon l'emprise des volumes de mort si on la demande, sinon aucun.
func boiteForge(ctx context.Context, opts OptionsCuissonForge) [4]float64 {
	if opts.BoiteUtile[2] > opts.BoiteUtile[0] && opts.BoiteUtile[3] > opts.BoiteUtile[1] {
		return opts.BoiteUtile
	}
	if !opts.RogneAuxVolumesDeMort {
		return [4]float64{}
	}
	boite, n := BoiteDesVolumesDeMort(opts.Objets)
	slog.InfoContext(ctx, "mapfond: bornage aux volumes de mort", "carte", opts.Cle,
		"volumes", n, "boite", fmt.Sprintf("[%.1f %.1f %.1f %.1f]", boite[0], boite[1], boite[2], boite[3]))
	return boite
}

// etendueMondeDe rend le plus grand cote XY, en metres, de l'emprise MONDE du premier
// exemplaire d'un modele. Une valeur de l'ordre du metre est un objet de decor ; plusieurs
// dizaines de metres sur une arene de cent, c'est un modele qui balaie la carte.
func etendueMondeDe(a *RuntimeGeoAsset, in Instance) float64 {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for mi := 0; mi < a.MeshCount(); mi++ {
		m := a.Mesh(mi)
		if m == nil {
			continue
		}
		for _, v := range m.Vertices {
			w := in.LocalToWorld(v)
			for k := 0; k < 2; k++ {
				lo[k] = math.Min(lo[k], w[k])
				hi[k] = math.Max(hi[k], w[k])
			}
		}
	}
	if math.IsInf(lo[0], 1) {
		return 0
	}
	return math.Max(hi[0]-lo[0], hi[1]-lo[1])
}

// journaliseTypesEtendus classe les types par emprise et journalise les dix premiers. C'est la
// SEULE sortie qui permette de designer un type fautif : les objets Forge n'ont pas de nom.
func journaliseTypesEtendus(ctx context.Context, b *BilanCuisson, etendues map[int32]float64, comptes map[int32]int) {
	type ligne struct {
		typeID  int32
		etendue float64
		n       int
	}
	var l []ligne
	for t, e := range etendues {
		l = append(l, ligne{t, e, comptes[t]})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].etendue > l[j].etendue })
	if len(l) > 10 {
		l = l[:10]
	}
	detail := make([]string, 0, len(l))
	for _, x := range l {
		detail = append(detail, fmt.Sprintf("%d:%.0fm x%d", x.typeID, x.etendue, x.n))
	}
	slog.InfoContext(ctx, "mapfond: types les plus etendus", "carte", b.Module,
		"types", len(etendues), "top", strings.Join(detail, " "))
}

// poseCanevasForge dessine la geometrie du CANEVAS sous les objets de la variante. Le bsp est
// choisi par les ancres, exactement comme dans la chaine native : un canevas porte un bsp de
// decor lointain (jusqu'a 1 900 m) et un bsp d'ile jouable, et seul le second nous interesse.
//
// Best-effort declare : un canevas absent ou illisible n'empeche PAS de cuire la carte, il la
// rend seulement sans son sol — ce qui etait le comportement de tous les fonds Forge jusqu'au
// 2026-08-27.
func poseCanevasForge(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuissonForge) {
	if opts.CheminModuleCanevas == "" {
		b.degrade(ctx, "canevas demande mais introuvable : la carte sera rendue sans son sol")
		return
	}
	bsps, err := ReadModuleInstances(opts.CheminModuleCanevas)
	if err != nil {
		b.degrade(ctx, "canevas illisible (%v) : la carte sera rendue sans son sol", err)
		return
	}
	idx, err := NewModuleIndex(cheminsCanevas(opts)...)
	if err != nil {
		b.degrade(ctx, "index du canevas illisible (%v)", err)
		return
	}
	bsp := ChoisitBSP(bsps, opts.Ancres)
	dessinees, decor := PeupleRendu(ctx, r, idx, bsp)
	b.CanevasDessinees, b.CanevasEcartees = dessinees, decor
	slog.InfoContext(ctx, "mapfond: canevas dessine sous la carte", "carte", opts.Cle,
		"bsps", len(bsps), "instances", len(bsp.Instances), "dessinees", dessinees, "decor", decor)
}

// cheminsCanevas rend l'index de tags a employer pour resoudre les modeles du canevas : le
// canevas lui-meme d'abord, puis les globals.
func cheminsCanevas(opts OptionsCuissonForge) []string {
	chemins := []string{opts.CheminModuleCanevas}
	globs, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)
	globsAny, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "any", "globals", "*.module"))
	return append(chemins, globsAny...)
}

// MinceurDuModele rend la part de son EMPRISE AU SOL que le modele couvre reellement, vue de
// dessus : on projette ses triangles sur une grille de 32 x 32 et on compte les cases pleines.
//
// POURQUOI CETTE MESURE ET PAS UNE AUTRE. Le gribouillis des cartes organiques vient de
// modeles de BRANCHES poses par centaines (349 exemplaires du seul type le plus nombreux
// d Isolation, etabli le 2026-08-27 en n en dessinant qu un seul). Deux criteres ont ete
// essayes et ont ECHOUE a les distinguer :
//
//  1. l EMPRISE du modele — elle attrape les gros rochers, qui sont legitimes ;
//  2. l AIRE DU MAILLAGE rapportee a l emprise — le type des branches y sort au rang 222
//     sur 271, parmi les plus PLEINS : une branche est un tube a nombreuses facettes, sa
//     surface est grande meme si elle ne couvre rien.
//
// Ce qui separe vraiment une branche d un rocher, c est que vue de dessus elle ne remplit
// presque rien de sa boite : quelques traits dans un carre vide.
// empriseAuSolDuModele rend la boite englobante PLANAIRE du modele, en coordonnees locales, et
// son nombre total de triangles. Extrait de MinceurDuModele, a l'identique.
func empriseAuSolDuModele(a *RuntimeGeoAsset) (lo, hi [2]float64, tris int) {
	lo = [2]float64{math.Inf(1), math.Inf(1)}
	hi = [2]float64{math.Inf(-1), math.Inf(-1)}
	for mi := 0; mi < a.MeshCount(); mi++ {
		m := a.Mesh(mi)
		if m == nil {
			continue
		}
		for _, v := range m.Vertices {
			for k := 0; k < 2; k++ {
				lo[k] = math.Min(lo[k], v[k])
				hi[k] = math.Max(hi[k], v[k])
			}
		}
		tris += len(m.Triangles)
	}
	return lo, hi, tris
}

func MinceurDuModele(a *RuntimeGeoAsset) (float64, bool) {
	const cotesGrille = 32
	lo, hi, tris := empriseAuSolDuModele(a)
	l, h := hi[0]-lo[0], hi[1]-lo[1]
	if tris == 0 || l <= 0 || h <= 0 {
		return 0, false
	}
	grille := make([]bool, cotesGrille*cotesGrille)
	pleines := 0
	marque := func(x, y float64) {
		i := int((x - lo[0]) / l * cotesGrille)
		j := int((y - lo[1]) / h * cotesGrille)
		if i < 0 || j < 0 || i >= cotesGrille || j >= cotesGrille {
			return
		}
		if k := j*cotesGrille + i; !grille[k] {
			grille[k] = true
			pleines++
		}
	}
	for mi := 0; mi < a.MeshCount(); mi++ {
		m := a.Mesh(mi)
		if m == nil {
			continue
		}
		// On echantillonne CHAQUE triangle en son centre et a ses sommets : suffisant a
		// cette resolution, et sans rasterisation a ecrire.
		for _, tr := range m.Triangles {
			p, q, s := m.Vertices[tr[0]], m.Vertices[tr[1]], m.Vertices[tr[2]]
			marque(p[0], p[1])
			marque(q[0], q[1])
			marque(s[0], s[1])
			marque((p[0]+q[0]+s[0])/3, (p[1]+q[1]+s[1])/3)
		}
	}
	return float64(pleines) / float64(cotesGrille*cotesGrille), true
}

// journalisePixelsParType dit QUELS TYPES OCCUPENT L'IMAGE, en pixels et en part du total.
//
// C'est la mesure qui manquait. Le « gribouillis » des cartes organiques a resiste a trente
// rendus, a cinq coupes geometriques et a trois criteres portes par le modele. La raison est
// simple : tous ces criteres decrivent ce qu'un objet EST, aucun ne dit ce qu'il PEINT.
// Ecarter les 349 branches d'Isolation ne changeait pas un octet du fichier — elles etaient
// sous d'autres surfaces. Ici, on demande a l'image elle-meme.
func journalisePixelsParType(ctx context.Context, r *Rendu, b BilanCuisson, objets []mapvar.Object) {
	pix := r.PixelsParType()
	if len(pix) == 0 {
		return
	}
	compte := map[int32]int{}
	for _, o := range objets {
		compte[o.TypeID]++
	}
	total := 0
	for _, n := range pix {
		total += n
	}
	type ligne struct {
		typeID int32
		pixels int
	}
	l := make([]ligne, 0, len(pix))
	for t, n := range pix {
		l = append(l, ligne{t, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].pixels > l[j].pixels })
	var detail []string
	for i, x := range l {
		if i >= 8 {
			break
		}
		detail = append(detail, fmt.Sprintf("%d:%.1f%%(x%d)", x.typeID,
			100*float64(x.pixels)/float64(total), compte[x.typeID]))
	}
	slog.InfoContext(ctx, "mapfond: qui peint l image", "carte", b.Module,
		"typesVisibles", len(pix), "pixels", total, "top", strings.Join(detail, " "))
}

// mesureEtRogneZonesForge mesure la matiere hors des zones de callout de la carte, et ne
// l efface que si la carte le demande. Jumeau exact de `mesureEtRogneZones` sur la chaine
// native — meme masque, meme dilatation, meme journal — a ceci pres que les zones ne sont pas
// fournies mais LUES DANS LES OBJETS deja charges.
func mesureEtRogneZonesForge(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuissonForge) {
	zs := ZonesNommeesForge(opts.Objets)
	if len(zs) == 0 {
		return
	}
	marge := MargeMasqueZones
	if opts.MargeZones > 0 {
		marge = opts.MargeZones
	} else if opts.MargeZones < 0 {
		marge = 0
	}
	m := MasqueZones(r, ContoursDeZones(zs), marge)
	matiere, dehors := r.MesureHorsZones(m)
	b.MatiereHorsZones = dehors
	part := 0.0
	if matiere > 0 {
		part = float64(dehors) / float64(matiere)
	}
	slog.InfoContext(ctx, "mapfond: matiere hors des zones de callout (Forge)", "carte", b.Module,
		"zones", len(zs), "matiere", matiere, "dehors", dehors,
		"part", fmt.Sprintf("%.1f%%", 100*part), "rogne", opts.RogneAuxZones, "marge", marge)
	if opts.RogneAuxZones {
		b.CellulesHorsZones = r.EffaceHorsZones(m)
	}
}

// combleTrousForge peint le sol suppose dans les trous interieurs. Il passe APRES les rognages :
// combler avant reviendrait a remplir des trous qu on s apprete a effacer.
func combleTrousForge(ctx context.Context, r *Rendu, b *BilanCuisson, opts OptionsCuissonForge) {
	if opts.CombleAuMaillage {
		n := r.CombleDansLeMaillage(MargeNavmesh)
		b.CellulesSolSuppose += n
		slog.InfoContext(ctx, "mapfond: sol suppose pose dans l emprise du maillage", "carte", b.Module,
			"cellules", n)
	}
	if !opts.CombleTrous {
		return
	}
	// Masque TOUT VRAI : la borne n est pas une zone mais la silhouette elle-meme, et c est
	// `CombleTrous` qui la determine en partant du bord de l image.
	tout := make([]bool, r.NX*r.NY)
	for i := range tout {
		tout[i] = true
	}
	b.CellulesSolSuppose += r.CombleTrous(tout)
	slog.InfoContext(ctx, "mapfond: sol suppose pose dans les trous interieurs", "carte", b.Module,
		"cellules", b.CellulesSolSuppose)
}
