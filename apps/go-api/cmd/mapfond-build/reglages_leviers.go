package main

// reglages_leviers.go — LA DÉCLARATION ET LA TABLE DES LEVIERS PAR CARTE.
//
// LA RÈGLE, ET ELLE EST LA RAISON D'ÊTRE DE CES FICHIERS. La brique `himap` ne contient AUCUNE
// branche par carte : c'est ce qui rend la chaîne transférable, et une seule carte possède
// l'oracle fort des positions de joueur, donc régler dans le code serait invérifiable.
// Mais le gate utilisateur du 2026-08-26 a établi, images à l'appui, que le meilleur rendu
// n'est pas le même partout — `encre` sur Cliffhanger, la cible du témoin sur Catalyst.
//
// Le choix vit donc en DONNÉE (`map_fond_reglages.json`), avec sa raison écrite et la date de son
// gate. La chaîne reçoit une ENTRÉE ; elle ne sait toujours pas quelle carte elle cuit.
//
// Trois fichiers, une responsabilité chacun :
//   - reglages_leviers.go : ce fichier, la structure d'une entrée et la table des leviers lus par valeur ;
//   - reglages_acces.go : l'accesseur générique `armeDe` et les leviers à règle propre ;
//   - reglages_validation.go : le chargement du fichier et ce qu'il refuse.
//
// UNE ENTRÉE SANS RAISON NI DATE EST UN RÉGLAGE ORPHELIN : dans six mois personne ne saura
// s'il tient encore. `TestReglagesFondJustifies` les refuse. Un levier qu'aucune carte ne pose est
// du code mort : `TestChaqueLevierADesDonneesOuUneExemptionDatee` le refuse.

import "github.com/JGtm/halo-infinite-map-images/himap"

// reglageCarte : ce qui peut varier d'une carte à l'autre. Un champ vide ou nul vaut « la
// valeur de production » — un réglage ne se déclare que là où il a été jugé.
type reglageCarte struct {
	Carte   string  `json:"carte,omitempty"`
	Style   string  `json:"style,omitempty"`
	Echelle float64 `json:"echelle,omitempty"`
	// ModuleGeometrie : chemin RELATIF A LA RACINE DU JEU d'un module qui porte la geometrie
	// de cette carte, quand ce n'est pas le sien.
	//
	// Un seul cas connu, et il a coute une conclusion fausse : `sgh_interlock` (Live Fire) ne
	// contient QUE six fichiers, dont un `levl` de 2,3 Mo — aucun sbsp, aucune bitmap. J'en
	// avais deduit « geometrie non installee » ; l'utilisateur, qui joue la carte
	// regulierement, a corrige : « ce doit etre une variante d'une autre map et le poids
	// leger doit etre la diff ». Mesure : `common-rtx-new.module` porte QUATRE sbsp qu'aucune
	// carte ne reclame, et le premier — 12 556 instances, X [-16,7 ; +46,5], Y [-10,1 ;
	// +53,7] — CONTIENT les 24 ancres d'objectif de Live Fire. La geometrie est donc bien
	// installee, ailleurs. Preuve rejouee par `TestGeometrieLiveFireDansCommon`.
	ModuleGeometrie string `json:"moduleGeometrie,omitempty"`
	// EcreteToits : vider les pixels dont aucune surface n est a hauteur de jeu
	// (himap/ecretage_toits.go). Jamais un defaut — il efface les rochers hauts d une carte
	// qui en fait son identite.
	EcreteToits bool `json:"ecreteToits,omitempty"`
	// PlafondArene : hauteur en metres au-dela de la reference locale a partir de laquelle une
	// surface cesse d etre un etage joue. Zero = 6 m. Le cran prevu : 4 m si « encore trop de
	// toits », 8 m si « trop vide ». Un seul cran a la fois, et re-gate.
	PlafondArene float64 `json:"plafondArene,omitempty"`
	// SansEau : ecarter l habillage d eau. Voir OptionsCuisson.SansEau — l eau est peinte par
	// la boite englobante de son volume, ce qui donne un rectangle bleu sur certaines cartes.
	SansEau bool `json:"sansEau,omitempty"`
	// SubstitutionSansPortee : etendre la substitution a tout le cadre au lieu du disque de
	// 25 m autour des ancres. Voir OptionsCuisson.SubstitutionSansPortee.
	SubstitutionSansPortee bool `json:"substitutionSansPortee,omitempty"`
	// CombleTrous : poser un aplat de sol suppose dans les trous des zones nommees.
	CombleTrous bool `json:"combleTrous,omitempty"`
	// CombleZonesEntieres : combler AUSSI les vides OUVERTS des zones nommees, pas seulement les
	// trous fermes. Version refutee sur Illusion le 2026-08-26 (611 959 cellules d aplat), rearmee
	// par carte apres verdict visuel. Voir himap.CombleZonesEntieres.
	CombleZonesEntieres bool `json:"combleZonesEntieres,omitempty"`
	// CombleAuMaillage : peindre le sol suppose dans toute l emprise du maillage de navigation
	CombleAuMaillage bool `json:"combleAuMaillage,omitempty"`
	// RogneAuxComposantesAncrees : effacer les amas de matiere sans ancre, hors silhouette jouee
	RogneAuxComposantesAncrees bool `json:"rogneAuxComposantesAncrees,omitempty"`
	// CadreAuxAncres : borner l image a l emprise des ancres d objectifs
	CadreAuxAncres bool `json:"cadreAuxAncres,omitempty"`
	// MargeAncres : marge autour des ancres pour le cadrage, en metres (0 = 25 m)
	MargeAncres float64 `json:"margeAncres,omitempty"`
	// MaillageNiveauHaut : reference prise sur le niveau le plus haut du maillage
	MaillageNiveauHaut bool `json:"maillageNiveauHaut,omitempty"`

	// SeuilCouverture : seuil de carte couverte propre a la carte, en part de matiere (0,25 =
	// 25 pour cent). Zero = le seuil universel `himap.SeuilCarteCouverte`. A n armer que sur une
	// carte dont l ecart median ancre -> surface dessinee montre des toits ; la substitution est
	// invariante en silhouette, elle ne peut donc pas couter d ancre.
	SeuilCouverture float64 `json:"seuilCouverture,omitempty"`
	// MargeNavmesh : dilatation du masque de rognage au maillage, en metres (0 = himap.MargeNavmesh,
	// 0,5 m)
	MargeNavmesh float64 `json:"margeNavmesh,omitempty"`
	// NavmeshMode : forme du masque de rognage au maillage (`faces` ou `silhouette`). Prime sur
	// l avis du user (`-avis`), y compris sur un `publie` : la carte est alors cuite.
	NavmeshMode string `json:"navmeshMode,omitempty"`
	// RogneAuxAltitudesProches : ne garder que ce qui est proche du niveau de jeu, plus une marge
	RogneAuxAltitudesProches bool `json:"rogneAuxAltitudesProches,omitempty"`

	// RogneAuxPositionsJouees efface la matiere loin de toute position reellement courue, lue
	// dans map_positions_jouees.json (cmd/mappos-build). Sans catalogue pour la carte, le levier
	// ne fait rien plutot que de vider l image.
	RogneAuxPositionsJouees bool `json:"rogneAuxPositionsJouees,omitempty"`
	// RayonPositions : rayon de garde autour d une position courue, en metres. Zero = 4 m.
	RayonPositions float64 `json:"rayonPositions,omitempty"`
	// SeuilRecollement : part de la surface d un objet que le masque doit garder pour que l objet
	// survive ENTIER (recollement_objets.go). Zero = un tiers ; negatif = recollement desarme.
	SeuilRecollement float64 `json:"seuilRecollement,omitempty"`
	SeuilAltitude    float64 `json:"seuilAltitude,omitempty"`
	MargeAltitude    float64 `json:"margeAltitude,omitempty"`
	// PlancherTranche : profondeur en metres SOUS le niveau de jeu (valeur NEGATIVE) en deca
	// de laquelle la matiere sort de la carte. Zero = -12 m. Voir OptionsCuisson.
	PlancherTranche float64 `json:"plancherTranche,omitempty"`
	// RogneAuxZones : effacer la matiere hors des zones nommees dilatees
	// (himap/masque_zones.go). A ne poser qu apres avoir regarde le taux mesure.
	RogneAuxZones bool `json:"rogneAuxZones,omitempty"`
	// MargeZones : dilatation du masque en metres. Zero = 4 m. Une valeur NEGATIVE demande
	// explicitement aucune dilatation.
	MargeZones float64 `json:"margeZones,omitempty"`
	// RogneAuxVolumesDeMort : borner la matiere a l emprise des volumes de mort de la variante
	// Forge. Equivalent Forge du rognage aux zones de callout, qui n existent que sur les
	// cartes natives (22 cartes, toutes natives).
	RogneAuxVolumesDeMort bool `json:"rogneAuxVolumesDeMort,omitempty"`
	// NavmeshReference prend le maillage de navigation pour SURFACE DE REFERENCE, en gardant la
	// geometrie ordinaire comme source du dessin.
	NavmeshReference bool `json:"navmeshReference,omitempty"`
	// RogneAuNavmesh efface la matiere hors du maillage de navigation : le pendant Forge du
	// masque des callouts, sans rien a dessiner a la main. Absent = defaut de la chaine (faux
	// sur une carte Forge, vrai sur une carte native) ; `false` explicite desarme le rognage
	// d'une carte native en lui laissant le maillage comme surface de reference.
	RogneAuNavmesh *bool `json:"rogneAuNavmesh,omitempty"`
	// ToleranceNavmesh vide les surfaces qui s ecartent de plus de N metres du sol donne par le
	// maillage de navigation. C est ce qui fait un PLAN D ETAGE plutot qu une vue de dessus.
	ToleranceNavmesh float64 `json:"toleranceNavmesh,omitempty"`
	// SansNavmesh : une carte NATIVE cuit sans s appuyer sur son maillage de navigation (ni
	// reference, ni rognage) mais publie quand meme son plan des murs au sidecar. Par defaut une
	// carte native est rognee a son maillage.
	SansNavmesh bool `json:"sansNavmesh,omitempty"`
	// BoiteUtile : rectangle monde [minX, minY, maxX, maxY] hors duquel la matiere est effacee.
	// LEVIER MANUEL — voir OptionsCuisson.BoiteUtile.
	BoiteUtile []float64 `json:"boiteUtile,omitempty"`
	Raison     string    `json:"raison"`
	GateLe     string    `json:"gateLe"`
}

// Les leviers lus par valeur. Les autres (habillage, boîte utile, positions jouées, altitudes
// proches, zones nommées) ont une règle propre, écrite plus bas.
var (
	levierEchelle = levierMesure("echelle propre a la carte", "mpp",
		func(c reglageCarte) float64 { return c.Echelle }, positif)
	levierEcreteToits = levierBool("ecretage des toits arme pour cette carte",
		func(c reglageCarte) bool { return c.EcreteToits })
	levierPlafondArene = levierMesure("plafond d arene propre a la carte", "plafond",
		func(c reglageCarte) float64 { return c.PlafondArene }, positif)
	levierSansEau = levierBool("habillage d eau ecarte pour cette carte",
		func(c reglageCarte) bool { return c.SansEau })
	levierSubstitutionSansPortee = levierBool("substitution etendue a tout le cadre",
		func(c reglageCarte) bool { return c.SubstitutionSansPortee })
	levierCombleTrous = levierBool("comblement des trous arme (aplat, pas un releve)",
		func(c reglageCarte) bool { return c.CombleTrous })
	levierCombleZonesEntieres = levierBool("comblement des vides OUVERTS arme pour cette carte (refute sur Illusion)",
		func(c reglageCarte) bool { return c.CombleZonesEntieres })
	levierCombleAuMaillage = levierBool("comblement dans l emprise du maillage arme pour cette carte",
		func(c reglageCarte) bool { return c.CombleAuMaillage })
	levierRogneAuxComposantesAncrees = levierBool("rognage aux composantes ancrees arme pour cette carte",
		func(c reglageCarte) bool { return c.RogneAuxComposantesAncrees })
	levierCadreAuxAncres = levierBool("cadrage a l emprise des ancres arme pour cette carte",
		func(c reglageCarte) bool { return c.CadreAuxAncres })
	levierMargeAncres = levierMesure("marge de cadrage aux ancres propre a la carte", "marge",
		func(c reglageCarte) float64 { return c.MargeAncres }, positif)
	levierSeuilCouverture = levierMesure("seuil de couverture propre a la carte", "seuil",
		func(c reglageCarte) float64 { return c.SeuilCouverture }, positif)
	levierPlancherTranche = levierMesure("plancher de tranche propre a la carte", "plancher",
		func(c reglageCarte) float64 { return c.PlancherTranche }, strictNegatif)
	levierRogneAuxZones = levierBool("rognage aux zones nommees arme pour cette carte",
		func(c reglageCarte) bool { return c.RogneAuxZones })
	levierMargeZones = levierMesure("marge du masque propre a la carte", "marge",
		func(c reglageCarte) float64 { return c.MargeZones }, nonNul)
	levierRogneAuxVolumesDeMort = levierBool("bornage aux volumes de mort arme pour cette carte",
		func(c reglageCarte) bool { return c.RogneAuxVolumesDeMort })
	levierSeuilRecollement = levierMesure("seuil de recollement propre a la carte", "seuil",
		func(c reglageCarte) float64 { return c.SeuilRecollement }, nonNul)
	levierModuleGeometrie = levier[string]{libelle: "geometrie prise dans un AUTRE module", attribut: "module",
		lit: func(c reglageCarte) (string, bool) { return c.ModuleGeometrie, c.ModuleGeometrie != "" }}
	levierNavmeshReference = levierBool("reference prise sur le maillage de navigation",
		func(c reglageCarte) bool { return c.NavmeshReference })
	levierMaillageNiveauHaut = levierBool("reference prise sur le niveau HAUT du maillage",
		func(c reglageCarte) bool { return c.MaillageNiveauHaut })
	levierMargeNavmesh = levierMesure("marge de rognage au maillage propre a la carte", "marge",
		func(c reglageCarte) float64 { return c.MargeNavmesh }, positif)
	levierToleranceNavmesh = levierMesure("tolerance au sol du maillage", "tolerance",
		func(c reglageCarte) float64 { return c.ToleranceNavmesh }, positif)
	levierSansNavmesh = levierBool("maillage de navigation desarme pour cette carte",
		func(c reglageCarte) bool { return c.SansNavmesh })
	levierNavmeshMode = levier[himap.ModeNavmesh]{libelle: "mode de rognage au maillage propre a la carte", attribut: "mode",
		lit: func(c reglageCarte) (himap.ModeNavmesh, bool) {
			return himap.ModeNavmesh(c.NavmeshMode), c.NavmeshMode != ""
		}}
	// Rognage au maillage : `true` explicite l'arme sur une carte Forge ; `false` explicite le
	// désarme sur une carte native. L'absence de valeur laisse le défaut de la chaîne.
	levierRogneAuNavmesh = levierBool("rognage au maillage de navigation",
		func(c reglageCarte) bool { return c.RogneAuNavmesh != nil && *c.RogneAuNavmesh })
	levierSansRogneAuNavmesh = levierBool("rognage au maillage desarme pour cette carte native",
		func(c reglageCarte) bool { return c.RogneAuNavmesh != nil && !*c.RogneAuNavmesh })
)
