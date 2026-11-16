package main

// reglages.go — LES RÉGLAGES DE CUISSON PAR CARTE.
//
// LA RÈGLE, ET ELLE EST LA RAISON D'ÊTRE DE CE FICHIER. `libs/himap/himap` ne contient AUCUNE
// branche par carte : c'est ce qui rend la chaîne transférable, et une seule carte possède
// l'oracle fort des positions de joueur, donc régler dans le code serait invérifiable.
// Mais le gate utilisateur du 2026-08-26 a établi, images à l'appui, que le meilleur rendu
// n'est pas le même partout — `encre` sur Cliffhanger, la cible du témoin sur Catalyst.
//
// Le choix vit donc en DONNÉE, ici, avec sa raison écrite et la date de son gate. La chaîne
// reçoit une ENTRÉE ; elle ne sait toujours pas quelle carte elle cuit.
//
// UNE ENTRÉE SANS RAISON NI DATE EST UN RÉGLAGE ORPHELIN : dans six mois personne ne saura
// s'il tient encore. `TestReglagesFondJustifies` les refuse.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

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

// sansLevier dit si l'entrée ne déclare AUCUN réglage : elle ne porte alors que son identité
// (carte, raison, date de gate) et n'a strictement aucun effet sur la cuisson.
//
// LE CONTRÔLE EST GÉNÉRIQUE, ET C'EST LE POINT. `TestReglagesFondJustifies` énumérait trois
// champs à la main — habillage, échelle, écrêtage — et cette liste avait cessé de suivre la
// structure : une entrée qui ne déclare que `moduleGeometrie` était refusée comme « sans
// effet » alors qu'elle change le module d'où la géométrie est lue. C'est exactement le cas de
// Live Fire. Une liste écrite à la main re-divergera ; la comparaison à la valeur nulle, non.
func (c reglageCarte) sansLevier() bool {
	c.Carte, c.Raison, c.GateLe = "", "", ""
	return reflect.DeepEqual(c, reglageCarte{})
}

type reglagesFond struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Note          string                  `json:"note,omitempty"`
	Cartes        map[string]reglageCarte `json:"cartes"`
}

// chargeReglages lit les réglages par carte. ABSENT n'est pas une erreur : c'est le cas
// nominal d'un titre qui n'a encore rien fait juger.
func chargeReglages(chemin string) (*reglagesFond, error) {
	blob, err := os.ReadFile(chemin)
	if errors.Is(err, os.ErrNotExist) {
		return &reglagesFond{Cartes: map[string]reglageCarte{}}, nil
	}
	if err != nil {
		return nil, err
	}
	// Un champ inconnu est REFUSÉ : un levier retiré qu'une entrée déclarerait encore serait
	// sinon ignoré en silence, et la carte cuite sans lui.
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.DisallowUnknownFields()
	var r reglagesFond
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("réglages de fond illisibles : %w", err)
	}
	if r.Cartes == nil {
		r.Cartes = map[string]reglageCarte{}
	}
	// Un habillage inconnu est REFUSÉ ici et pas silencieusement ignoré : une carte cuite
	// dans l'habillage par défaut alors que la donnée en demandait un autre passerait le
	// gate sous une fausse identité.
	for cle, c := range r.Cartes {
		if c.Style != "" && !himap.StyleFondValide(himap.StyleFond(c.Style)) {
			return nil, fmt.Errorf("réglage %q : habillage inconnu %q", cle, c.Style)
		}
		if !modeNavmeshValide(c.NavmeshMode) {
			return nil, fmt.Errorf("réglage %q : navmeshMode inconnu %q (faces ou silhouette)", cle, c.NavmeshMode)
		}
	}
	return &r, nil
}

// levier décrit un levier par carte : comment il se lit dans une entrée et comment il se journalise.
// Tous les leviers qui se lisent par une valeur passent par `armeDe` : une carte sans entrée, ou dont
// l'entrée ne l'arme pas, reçoit la valeur nulle, c'est-à-dire la valeur de production.
type levier[T any] struct {
	// libelle : message journalisé quand le levier s'applique à la carte. Supprimer de la matière ou
	// peindre un aplat ne doit jamais passer inaperçu.
	libelle string
	// attribut : nom de la valeur journalisée avec le message ; vide pour un levier booléen.
	attribut string
	// lit rend la valeur du levier dans l'entrée et dit s'il y est armé.
	lit func(reglageCarte) (T, bool)
}

// levierBool : armé quand le champ est vrai.
func levierBool(libelle string, champ func(reglageCarte) bool) levier[bool] {
	return levier[bool]{libelle: libelle, lit: func(c reglageCarte) (bool, bool) { v := champ(c); return v, v }}
}

// levierMesure : armé quand `arme` accepte la valeur du champ (strictement positive, non nulle ou
// strictement négative selon le levier) ; la valeur journalisée porte le nom `attribut`.
func levierMesure(libelle, attribut string, champ func(reglageCarte) float64, arme func(float64) bool) levier[float64] {
	return levier[float64]{libelle: libelle, attribut: attribut,
		lit: func(c reglageCarte) (float64, bool) { v := champ(c); return v, arme(v) }}
}

func positif(v float64) bool       { return v > 0 }
func nonNul(v float64) bool        { return v != 0 }
func strictNegatif(v float64) bool { return v < 0 }

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

// entreeDe rend l'entrée de réglage de la carte et dit si elle existe.
func (e *environnement) entreeDe(cle string) (reglageCarte, bool) {
	if e.reglages == nil {
		return reglageCarte{}, false
	}
	c, ok := e.reglages.Cartes[cle]
	return c, ok
}

// armeDe est l'accesseur de tous les leviers par valeur : il rend la valeur du levier pour la carte
// et dit s'il y est armé, et le journalise avec la date de son gate. Sans entrée ou levier non armé,
// il rend la valeur nulle.
func armeDe[T any](e *environnement, cle string, l levier[T]) (T, bool) {
	var nul T
	c, ok := e.entreeDe(cle)
	if !ok {
		return nul, false
	}
	v, arme := l.lit(c)
	if !arme {
		return nul, false
	}
	args := []any{"carte", cle}
	if l.attribut != "" {
		args = append(args, l.attribut, v)
	}
	slog.Info("mapfond: "+l.libelle, append(args, "gateLe", c.GateLe)...)
	return v, true
}

// valeurDe rend la valeur du levier pour la carte, nulle quand il n'y est pas armé.
func valeurDe[T any](e *environnement, cle string, l levier[T]) T {
	v, _ := armeDe(e, cle, l)
	return v
}

// styleDe rend l'habillage de cette carte : le sien s'il est déclaré, sinon celui de la
// cuisson. Le choix retenu est JOURNALISÉ — un fond publié dans un habillage qu'on n'a pas vu
// passer est un fond qu'on ne saura pas expliquer.
func (e *environnement) styleDe(cle string) himap.StyleFond {
	c, ok := e.entreeDe(cle)
	if !ok || c.Style == "" {
		return e.style
	}
	s := himap.StyleFond(c.Style)
	if s != e.style {
		slog.Info("mapfond: habillage propre a la carte", "carte", cle, "style", string(s),
			"gateLe", c.GateLe)
	}
	return s
}

// echelleDe rend l'échelle de cette carte : la sienne si elle est déclarée, sinon celle de la
// cuisson (elle-même à la valeur de production si aucune n'est demandée).
func (e *environnement) echelleDe(cle string) float64 {
	if v, ok := armeDe(e, cle, levierEchelle); ok {
		return v
	}
	return e.echelle
}

// zonesNommeesDe rend les polygones des callouts d'une carte : le contour principal de chaque
// zone, plus ses PARTIES (provenance « decoupe »).
//
// LES TROUS NE SONT PAS SOUSTRAITS, et c'est délibéré : le masque sert à décider ce qu'on
// GARDE. Un trou rempli garde un peu plus de matière — l'erreur va dans le sens prudent. Les
// soustraire demanderait une règle pair-impair globale et ferait disparaître, au moindre défaut
// de découpe, du terrain que personne n'a jugé.
func (e *environnement) zonesNommeesDe(cle string) [][][2]float64 {
	if e.callouts == nil {
		return nil
	}
	entree, ok := e.callouts.Maps[cle]
	if !ok {
		return nil
	}
	var out [][][2]float64
	for _, z := range entree.Zones {
		if len(z.Polygon) >= 3 {
			out = append(out, z.Polygon)
		}
		for _, p := range z.Parts {
			if len(p) >= 3 {
				out = append(out, p)
			}
		}
	}
	return out
}

// boiteUtileDe rend le rectangle monde declaré pour cette carte, ou zéro s'il n'y en a pas.
func (e *environnement) boiteUtileDe(cle string) [4]float64 {
	var out [4]float64
	c, ok := e.entreeDe(cle)
	if !ok || len(c.BoiteUtile) != 4 {
		return out
	}
	copy(out[:], c.BoiteUtile)
	return out
}

// moduleGeometrieDe rend le chemin ABSOLU du module qui porte la geometrie de cette carte,
// ou la chaine vide quand c'est le sien — le cas de toutes les cartes sauf une.
func (e *environnement) moduleGeometrieDe(cle string) string {
	rel, ok := armeDe(e, cle, levierModuleGeometrie)
	if !ok {
		return ""
	}
	return filepath.Join(e.racineJeu, filepath.FromSlash(rel))
}

// rogneAuNavmeshDe dit si cette carte se rogne a son maillage de navigation.
func (e *environnement) rogneAuNavmeshDe(cle string) bool {
	return valeurDe(e, cle, levierRogneAuNavmesh)
}

// rogneAuNavmeshNatifDe dit si une carte native se rogne a son maillage : oui, sauf si son
// reglage dit `rogneAuNavmesh: false`.
func (e *environnement) rogneAuNavmeshNatifDe(cle string) bool {
	return !valeurDe(e, cle, levierSansRogneAuNavmesh)
}

// rogneAuxAltitudesProchesDe dit si cette carte n affiche que ce qui est proche du niveau de
// jeu. Journalise : c est une voie qui SUPPRIME de la matiere.
func (e *environnement) rogneAuxAltitudesProchesDe(cle string) (bool, float64, float64) {
	c, ok := e.entreeDe(cle)
	if !ok || !c.RogneAuxAltitudesProches {
		return false, 0, 0
	}
	slog.Info("mapfond: rognage a l altitude du niveau de jeu arme pour cette carte", "carte", cle,
		"seuil", c.SeuilAltitude, "marge", c.MargeAltitude, "gateLe", c.GateLe)
	return true, c.SeuilAltitude, c.MargeAltitude
}

// positionsJoueesDe rend les positions réellement jouées de la carte, ou nil si le levier n'est
// pas armé pour elle. Journalisé : ce rognage s'appuie sur un corpus observé, et le nombre de
// positions retenues explique à lui seul l'agressivité du masque.
func (e *environnement) positionsJoueesDe(cle string) ([]himap.PositionJouee, float64) {
	c, ok := e.entreeDe(cle)
	if !ok || !c.RogneAuxPositionsJouees {
		return nil, 0
	}
	pos := e.positions[cle]
	if len(pos) == 0 {
		slog.Warn("mapfond: rognage aux positions demande mais catalogue vide pour cette carte",
			"carte", cle, "gateLe", c.GateLe)
		return nil, 0
	}
	slog.Info("mapfond: rognage aux positions jouees arme pour cette carte", "carte", cle,
		"positions", len(pos), "rayon", c.RayonPositions, "gateLe", c.GateLe)
	return pos, c.RayonPositions
}

// chargePositionsJouees lit le catalogue des positions jouées. Son absence n'est pas une erreur :
// aucune carte n'a alors ce levier disponible, et `positionsJoueesDe` le dira carte par carte.
func chargePositionsJouees(chemin string) map[string][]himap.PositionJouee {
	brut, err := os.ReadFile(chemin)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("mapfond: catalogue des positions jouees illisible", "err", err, "path", chemin)
		}
		return nil
	}
	var cat struct {
		Maps map[string]struct {
			Positions [][2]float64 `json:"positions"`
		} `json:"maps"`
	}
	if err := json.Unmarshal(brut, &cat); err != nil {
		slog.Warn("mapfond: catalogue des positions jouees invalide", "err", err, "path", chemin)
		return nil
	}
	out := make(map[string][]himap.PositionJouee, len(cat.Maps))
	for cle, m := range cat.Maps {
		pts := make([]himap.PositionJouee, 0, len(m.Positions))
		for _, p := range m.Positions {
			pts = append(pts, himap.PositionJouee{X: p[0], Y: p[1]})
		}
		out[cle] = pts
	}
	slog.Info("mapfond: catalogue des positions jouees charge", "cartes", len(out), "path", chemin)
	return out
}
