package main

// reglages.go — LES RÉGLAGES DE CUISSON PAR CARTE.
//
// LA RÈGLE, ET ELLE EST LA RAISON D'ÊTRE DE CE FICHIER. `internal/himap` ne contient AUCUNE
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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"levelup/go-api/internal/himap"
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
	// CadreAuxZones : borner l image a l emprise des zones de callout
	CadreAuxZones bool `json:"cadreAuxZones,omitempty"`
	// CadreAuxAncres : borner l image a l emprise des ancres d objectifs
	CadreAuxAncres bool `json:"cadreAuxAncres,omitempty"`
	// MargeAncres : marge autour des ancres pour le cadrage, en metres (0 = 25 m)
	MargeAncres float64 `json:"margeAncres,omitempty"`
	// MaillageNiveauHaut : reference prise sur le niveau le plus haut du maillage
	MaillageNiveauHaut bool `json:"maillageNiveauHaut,omitempty"`
	// SansSubstitution : garder la surface haute telle que dessinee
	SansSubstitution bool `json:"sansSubstitution,omitempty"`
	// SeuilSubstitution : ecart minimal en metres pour rabattre une surface sur la reference
	SeuilSubstitution float64 `json:"seuilSubstitution,omitempty"`

	// SeuilCouverture : seuil de carte couverte propre a la carte, en part de matiere (0,25 =
	// 25 pour cent). Zero = le seuil universel `himap.SeuilCarteCouverte`. A n armer que sur une
	// carte dont l ecart median ancre -> surface dessinee montre des toits ; la substitution est
	// invariante en silhouette, elle ne peut donc pas couter d ancre.
	SeuilCouverture float64 `json:"seuilCouverture,omitempty"`
	// MargeNavmesh : dilatation du masque de rognage au maillage, en metres (0 = 3 m)
	MargeNavmesh float64 `json:"margeNavmesh,omitempty"`
	// MargeSolBas : profondeur acceptee sous le niveau de jeu pour le sol vu du dessous
	MargeSolBas float64 `json:"margeSolBas,omitempty"`
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
	// SeuilArete : denivele en metres entre deux pixels voisins au-dela duquel on souligne un
	// bord. Zero = 0,5 m. A RELEVER sur les cartes en pieces organiques, ou le seuil par
	// defaut couvre l image d un gribouillis de traits.
	SeuilArete float64 `json:"seuilArete,omitempty"`
	// DessineCanevas : poser AUSSI la geometrie du canevas sous les objets de la variante
	// Forge. Voir OptionsCuissonForge.DessineCanevas.
	DessineCanevas bool `json:"dessineCanevas,omitempty"`
	// PlafondTranche : hauteur en metres AU-DESSUS du niveau de jeu au-dela de laquelle la
	// matiere n est meme pas projetee. Zero = tranche par defaut. Voir OptionsCuisson.
	PlafondTranche float64 `json:"plafondTranche,omitempty"`
	// RogneAuxZones : effacer la matiere hors des zones nommees dilatees
	// (himap/masque_zones.go). A ne poser qu apres avoir regarde le taux mesure.
	RogneAuxZones bool `json:"rogneAuxZones,omitempty"`
	// MargeZones : dilatation du masque en metres. Zero = 4 m. Une valeur NEGATIVE demande
	// explicitement aucune dilatation.
	MargeZones float64 `json:"margeZones,omitempty"`
	// ZonesContourSeul : ne garder que le CONTOUR principal de chaque zone, sans ses `parts`.
	// Les parties d une zone en provenance « decoupe » suivent le masque praticable et peuvent
	// s etendre loin — sur Catalyst elles longent les bras de la station. Les exclure serre le
	// masque au coeur des zones, au risque d amputer des zones reelles.
	ZonesContourSeul bool `json:"zonesContourSeul,omitempty"`
	// RogneAuxVolumesDeMort : borner la matiere a l emprise des volumes de mort de la variante
	// Forge. Equivalent Forge du rognage aux zones de callout, qui n existent que sur les
	// cartes natives (22 cartes, toutes natives).
	RogneAuxVolumesDeMort bool `json:"rogneAuxVolumesDeMort,omitempty"`
	// DrapeauxExclus : valeurs du champ de drapeaux des objets Forge a ne pas dessiner.
	DrapeauxExclus []int `json:"drapeauxExclus,omitempty"`
	// PlafondObjets : ecarter les objets Forge POSES plus haut que N metres au-dessus du sol
	// joue. Voir OptionsCuissonForge.PlafondObjets — ce n est ni l ecretage ni la tranche.
	PlafondObjets float64 `json:"plafondObjets,omitempty"`
	// NavmeshReference prend le maillage de navigation pour SURFACE DE REFERENCE, en gardant la
	// geometrie ordinaire comme source du dessin.
	NavmeshReference bool `json:"navmeshReference,omitempty"`
	// RogneAuNavmesh efface la matiere hors du maillage de navigation : le pendant Forge du
	// masque des callouts, sans rien a dessiner a la main.
	RogneAuNavmesh bool `json:"rogneAuNavmesh,omitempty"`
	// ToleranceNavmesh vide les surfaces qui s ecartent de plus de N metres du sol donne par le
	// maillage de navigation. C est ce qui fait un PLAN D ETAGE plutot qu une vue de dessus.
	ToleranceNavmesh float64 `json:"toleranceNavmesh,omitempty"`
	// SourceNavmesh cuit la carte depuis son MAILLAGE DE NAVIGATION au lieu de sa geometrie de
	// rendu. Reservee aux cartes Forge : elles seules publient un navmesh.blob.
	SourceNavmesh bool `json:"sourceNavmesh,omitempty"`
	// SolVuDuDessous : retenir la surface la plus BASSE au-dessus du sol joue. Pour les cartes
	// a ciel ferme, ou la voie haute ne montre que le plafond.
	SolVuDuDessous bool `json:"solVuDuDessous,omitempty"`
	// MinceurMin : seuil d aire rapportee au carre de l emprise sous lequel un modele Forge
	// n est PAS dessine. Zero = tout dessiner. Voir OptionsCuissonForge.MinceurMin.
	MinceurMin float64 `json:"minceurMin,omitempty"`
	// TypesExclus : identifiants de TYPE d objet Forge a ne pas dessiner. Dernier recours,
	// quand un modele balaie la carte et qu aucune coupe geometrique ne peut l atteindre. Les
	// candidats se lisent dans le log « types les plus etendus » de chaque cuisson Forge.
	TypesExclus []int32 `json:"typesExclus,omitempty"`
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
	var r reglagesFond
	if err := json.Unmarshal(blob, &r); err != nil {
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
	}
	return &r, nil
}

// styleDe rend l'habillage de cette carte : le sien s'il est déclaré, sinon celui de la
// cuisson. Le choix retenu est JOURNALISÉ — un fond publié dans un habillage qu'on n'a pas vu
// passer est un fond qu'on ne saura pas expliquer.
func (e *environnement) styleDe(cle string) himap.StyleFond {
	if e.reglages == nil {
		return e.style
	}
	c, ok := e.reglages.Cartes[cle]
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
	if e.reglages == nil {
		return e.echelle
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.Echelle <= 0 {
		return e.echelle
	}
	slog.Info("mapfond: echelle propre a la carte", "carte", cle, "mpp", c.Echelle,
		"gateLe", c.GateLe)
	return c.Echelle
}

// ecreteToitsDe dit si cette carte demande l'écrêtage des toits. Le choix est JOURNALISÉ :
// c'est la seule voie qui SUPPRIME de la matière, elle ne doit jamais passer inaperçue.
func (e *environnement) ecreteToitsDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.EcreteToits {
		return false
	}
	slog.Info("mapfond: ecretage des toits arme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
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
	contourSeul := false
	if e.reglages != nil {
		if c, ok := e.reglages.Cartes[cle]; ok {
			contourSeul = c.ZonesContourSeul
		}
	}
	for _, z := range entree.Zones {
		if len(z.Polygon) >= 3 {
			out = append(out, z.Polygon)
		}
		if contourSeul {
			continue
		}
		for _, p := range z.Parts {
			if len(p) >= 3 {
				out = append(out, p)
			}
		}
	}
	if contourSeul {
		slog.Info("mapfond: masque limite au contour des zones, parties exclues", "carte", cle, "polygones", len(out))
	}
	return out
}

// rogneAuxZonesDe dit si cette carte demande le rognage sur ses zones nommées. Journalisé :
// c'est, avec l'écrêtage, l'une des deux voies qui SUPPRIMENT de la matière.
func (e *environnement) rogneAuxZonesDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuxZones {
		return false
	}
	slog.Info("mapfond: rognage aux zones nommees arme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// plafondAreneDe rend le plafond d'arène propre à une carte, ou zéro pour celui de production.
func (e *environnement) plafondAreneDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.PlafondArene <= 0 {
		return 0
	}
	slog.Info("mapfond: plafond d arene propre a la carte", "carte", cle, "plafond", c.PlafondArene,
		"gateLe", c.GateLe)
	return c.PlafondArene
}

// sansEauDe dit si cette carte écarte l'habillage d'eau. Journalisé : retirer un calque
// entier de l'asset ne doit jamais passer inaperçu.
func (e *environnement) sansEauDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SansEau {
		return false
	}
	slog.Info("mapfond: habillage d eau ecarte pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// substitutionSansPorteeDe dit si cette carte etend la substitution a tout le cadre.
func (e *environnement) substitutionSansPorteeDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SubstitutionSansPortee {
		return false
	}
	slog.Info("mapfond: substitution etendue a tout le cadre", "carte", cle, "gateLe", c.GateLe)
	return true
}

// combleTrousDe dit si cette carte comble ses trous par un aplat. Journalisé : c'est du
// dessin, pas du relevé.
func (e *environnement) combleTrousDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.CombleTrous {
		return false
	}
	slog.Info("mapfond: comblement des trous arme (aplat, pas un releve)", "carte", cle, "gateLe", c.GateLe)
	return true
}

// plancherTrancheDe rend le plancher de tranche propre à une carte (négatif), ou zéro pour
// celui de production. Journalisé : il retire de la matière du bas de la carte.
func (e *environnement) plancherTrancheDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.PlancherTranche >= 0 {
		return 0
	}
	slog.Info("mapfond: plancher de tranche propre a la carte", "carte", cle,
		"plancher", c.PlancherTranche, "gateLe", c.GateLe)
	return c.PlancherTranche
}

// margeZonesDe rend la dilatation du masque propre à une carte, ou zéro pour celle de production.
func (e *environnement) margeZonesDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.MargeZones == 0 {
		return 0
	}
	slog.Info("mapfond: marge du masque propre a la carte", "carte", cle, "marge", c.MargeZones,
		"gateLe", c.GateLe)
	return c.MargeZones
}

// boiteUtileDe rend le rectangle monde declaré pour cette carte, ou zéro s'il n'y en a pas.
func (e *environnement) boiteUtileDe(cle string) [4]float64 {
	var out [4]float64
	if e.reglages == nil {
		return out
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || len(c.BoiteUtile) != 4 {
		return out
	}
	copy(out[:], c.BoiteUtile)
	return out
}

// moduleGeometrieDe rend le chemin ABSOLU du module qui porte la geometrie de cette carte,
// ou la chaine vide quand c'est le sien — le cas de toutes les cartes sauf une.
func (e *environnement) moduleGeometrieDe(cle string) string {
	if e.reglages == nil {
		return ""
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.ModuleGeometrie == "" {
		return ""
	}
	chemin := filepath.Join(e.racineJeu, filepath.FromSlash(c.ModuleGeometrie))
	slog.Info("mapfond: geometrie prise dans un AUTRE module", "carte", cle,
		"module", c.ModuleGeometrie, "gateLe", c.GateLe)
	return chemin
}

// rogneAuxVolumesDeMortDe dit si cette carte Forge demande le bornage aux volumes de mort.
// Journalise comme toute voie qui SUPPRIME de la matiere.
func (e *environnement) rogneAuxVolumesDeMortDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuxVolumesDeMort {
		return false
	}
	slog.Info("mapfond: bornage aux volumes de mort arme pour cette carte", "carte", cle,
		"gateLe", c.GateLe)
	return true
}

// typesExclusDe rend les types d objet Forge ecartes du dessin pour cette carte.
func (e *environnement) typesExclusDe(cle string) map[int32]bool {
	if e.reglages == nil {
		return nil
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || len(c.TypesExclus) == 0 {
		return nil
	}
	m := make(map[int32]bool, len(c.TypesExclus))
	for _, t := range c.TypesExclus {
		m[t] = true
	}
	slog.Info("mapfond: types d objet ecartes pour cette carte", "carte", cle,
		"types", len(m), "gateLe", c.GateLe)
	return m
}

// plafondTrancheDe rend la hauteur de coupe HAUTE de cette carte, en metres au-dessus du
// niveau de jeu. Zero = la tranche par defaut.
func (e *environnement) plafondTrancheDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.PlafondTranche <= 0 {
		return 0
	}
	slog.Info("mapfond: tranche plafonnee pour cette carte", "carte", cle,
		"plafond", c.PlafondTranche, "gateLe", c.GateLe)
	return c.PlafondTranche
}

// dessineCanevasDe dit si cette carte Forge demande que son canevas soit dessine sous elle.
func (e *environnement) dessineCanevasDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.DessineCanevas {
		return false
	}
	slog.Info("mapfond: canevas dessine pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// seuilAreteDe rend le seuil d arete propre a cette carte, ou zero pour le defaut.
func (e *environnement) seuilAreteDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.SeuilArete <= 0 {
		return 0
	}
	slog.Info("mapfond: seuil d arete propre a la carte", "carte", cle, "seuil", c.SeuilArete,
		"gateLe", c.GateLe)
	return c.SeuilArete
}

// minceurMinDe rend le seuil de minceur sous lequel les modeles filaires sont ecartes.
func (e *environnement) minceurMinDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.MinceurMin <= 0 {
		return 0
	}
	slog.Info("mapfond: modeles filaires ecartes sous ce seuil", "carte", cle,
		"minceurMin", c.MinceurMin, "gateLe", c.GateLe)
	return c.MinceurMin
}

// solVuDuDessousDe dit si cette carte se rend par la surface basse.
func (e *environnement) solVuDuDessousDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SolVuDuDessous {
		return false
	}
	slog.Info("mapfond: sol vu du dessous arme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// plafondObjetsDe rend la hauteur au-dessus du sol joue au-dela de laquelle un objet Forge
// n est pas pose. Zero = les poser tous.
func (e *environnement) plafondObjetsDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.PlafondObjets <= 0 {
		return 0
	}
	slog.Info("mapfond: plafond des objets propre a la carte", "carte", cle,
		"plafond", c.PlafondObjets, "gateLe", c.GateLe)
	return c.PlafondObjets
}

// drapeauxExclusDe rend les valeurs de drapeau d objet Forge a ecarter du dessin.
func (e *environnement) drapeauxExclusDe(cle string) map[uint8]bool {
	if e.reglages == nil {
		return nil
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || len(c.DrapeauxExclus) == 0 {
		return nil
	}
	m := make(map[uint8]bool, len(c.DrapeauxExclus))
	for _, d := range c.DrapeauxExclus {
		m[uint8(d)] = true
	}
	slog.Info("mapfond: drapeaux d objet ecartes pour cette carte", "carte", cle,
		"drapeaux", c.DrapeauxExclus, "gateLe", c.GateLe)
	return m
}

// sourceNavmeshDe dit si cette carte se cuit depuis son maillage de navigation.
func (e *environnement) sourceNavmeshDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SourceNavmesh {
		return false
	}
	slog.Info("mapfond: cuisson depuis le maillage de navigation", "carte", cle, "gateLe", c.GateLe)
	return true
}

// navmeshReferenceDe dit si cette carte prend sa surface de reference sur son maillage de
// navigation. A distinguer de sourceNavmeshDe, qui dessine le maillage lui-meme : ici on garde
// la geometrie ordinaire comme source du dessin, donc les structures, et on ne change que ce a
// quoi les surfaces sont comparees.
func (e *environnement) navmeshReferenceDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.NavmeshReference {
		return false
	}
	slog.Info("mapfond: reference prise sur le maillage de navigation", "carte", cle, "gateLe", c.GateLe)
	return true
}

// rogneAuNavmeshDe dit si cette carte se rogne a son maillage de navigation.
func (e *environnement) rogneAuNavmeshDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuNavmesh {
		return false
	}
	slog.Info("mapfond: rognage au maillage de navigation", "carte", cle, "gateLe", c.GateLe)
	return true
}

// toleranceNavmeshDe rend l ecart au sol au-dela duquel une surface est videe.
func (e *environnement) toleranceNavmeshDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.ToleranceNavmesh <= 0 {
		return 0
	}
	slog.Info("mapfond: tolerance au sol du maillage", "carte", cle, "tolerance", c.ToleranceNavmesh, "gateLe", c.GateLe)
	return c.ToleranceNavmesh
}

// combleAuMaillageDe dit si cette carte comble le dessin dans l emprise de son maillage de
// navigation. Journalise : peindre un aplat la ou rien n a ete mesure ne doit jamais passer
// inapercu.
func (e *environnement) combleAuMaillageDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.CombleAuMaillage {
		return false
	}
	slog.Info("mapfond: comblement dans l emprise du maillage arme pour cette carte", "carte", cle,
		"gateLe", c.GateLe)
	return true
}

// rogneAuxComposantesAncreesDe dit si cette carte efface les amas qui ne portent pas d ancre.
// Journalise : c est une voie qui SUPPRIME de la matiere, comme l ecretage et les autres
// rognages.
func (e *environnement) rogneAuxComposantesAncreesDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuxComposantesAncrees {
		return false
	}
	slog.Info("mapfond: rognage aux composantes ancrees arme pour cette carte", "carte", cle,
		"gateLe", c.GateLe)
	return true
}

// cadreAuxZonesDe dit si cette carte borne son image a l emprise de ses zones de callout.
// Journalise : borner une image est une voie qui SUPPRIME de la matiere.
func (e *environnement) cadreAuxZonesDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.CadreAuxZones {
		return false
	}
	slog.Info("mapfond: cadrage a l emprise des zones arme pour cette carte", "carte", cle,
		"gateLe", c.GateLe)
	return true
}

// cadreAuxAncresDe dit si cette carte borne son image a l emprise de ses ancres d objectifs.
func (e *environnement) cadreAuxAncresDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.CadreAuxAncres {
		return false
	}
	slog.Info("mapfond: cadrage a l emprise des ancres arme pour cette carte", "carte", cle,
		"gateLe", c.GateLe)
	return true
}

// margeAncresDe rend la marge de cadrage aux ancres propre a une carte, ou zero pour celle de
// production.
func (e *environnement) margeAncresDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.MargeAncres <= 0 {
		return 0
	}
	slog.Info("mapfond: marge de cadrage aux ancres propre a la carte", "carte", cle,
		"marge", c.MargeAncres, "gateLe", c.GateLe)
	return c.MargeAncres
}

// maillageNiveauHautDe dit si cette carte prend la reference sur le niveau le plus haut du
// maillage. Journalise : c est ce qui decide si ses etages survivent a la substitution.
func (e *environnement) maillageNiveauHautDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.MaillageNiveauHaut {
		return false
	}
	slog.Info("mapfond: reference prise sur le niveau HAUT du maillage", "carte", cle, "gateLe", c.GateLe)
	return true
}

// sansSubstitutionDe dit si cette carte renonce a la substitution par surface de reference.
// Journalise : c est le levier qui decide si les coques tombent ou si les structures restent.
func (e *environnement) sansSubstitutionDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SansSubstitution {
		return false
	}
	slog.Info("mapfond: substitution desarmee pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// seuilSubstitutionDe rend l ecart minimal de substitution propre a une carte, ou zero.
func (e *environnement) seuilSubstitutionDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.SeuilSubstitution <= 0 {
		return 0
	}
	slog.Info("mapfond: seuil de substitution propre a la carte", "carte", cle,
		"seuil", c.SeuilSubstitution, "gateLe", c.GateLe)
	return c.SeuilSubstitution
}

// margeNavmeshDe rend la marge de rognage au maillage propre a une carte, ou zero.
func (e *environnement) margeNavmeshDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.MargeNavmesh <= 0 {
		return 0
	}
	slog.Info("mapfond: marge de rognage au maillage propre a la carte", "carte", cle,
		"marge", c.MargeNavmesh, "gateLe", c.GateLe)
	return c.MargeNavmesh
}

// margeSolBasDe rend la profondeur acceptee sous le niveau de jeu, ou zero.
func (e *environnement) margeSolBasDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.MargeSolBas <= 0 {
		return 0
	}
	slog.Info("mapfond: profondeur du sol vu du dessous propre a la carte", "carte", cle,
		"marge", c.MargeSolBas, "gateLe", c.GateLe)
	return c.MargeSolBas
}

// rogneAuxAltitudesProchesDe dit si cette carte n affiche que ce qui est proche du niveau de
// jeu. Journalise : c est une voie qui SUPPRIME de la matiere.
func (e *environnement) rogneAuxAltitudesProchesDe(cle string) (bool, float64, float64) {
	if e.reglages == nil {
		return false, 0, 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.RogneAuxAltitudesProches {
		return false, 0, 0
	}
	slog.Info("mapfond: rognage a l altitude du niveau de jeu arme pour cette carte", "carte", cle,
		"seuil", c.SeuilAltitude, "marge", c.MargeAltitude, "gateLe", c.GateLe)
	return true, c.SeuilAltitude, c.MargeAltitude
}

// seuilCouvertureDe rend le seuil de carte couverte propre à une carte, ou zéro pour celui de
// production. Journalisé : abaisser ce seuil change la voie de rendu prise par la carte.
func (e *environnement) seuilCouvertureDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.SeuilCouverture <= 0 {
		return 0
	}
	slog.Info("mapfond: seuil de couverture propre a la carte", "carte", cle,
		"seuil", c.SeuilCouverture, "gateLe", c.GateLe)
	return c.SeuilCouverture
}

// positionsJoueesDe rend les positions réellement jouées de la carte, ou nil si le levier n'est
// pas armé pour elle. Journalisé : ce rognage s'appuie sur un corpus observé, et le nombre de
// positions retenues explique à lui seul l'agressivité du masque.
func (e *environnement) positionsJoueesDe(cle string) ([]himap.PositionJouee, float64) {
	if e.reglages == nil {
		return nil, 0
	}
	c, ok := e.reglages.Cartes[cle]
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

// seuilRecollementDe rend le seuil de recollement aux objets propre a une carte.
func (e *environnement) seuilRecollementDe(cle string) float64 {
	if e.reglages == nil {
		return 0
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.SeuilRecollement == 0 {
		return 0
	}
	slog.Info("mapfond: seuil de recollement propre a la carte", "carte", cle,
		"seuil", c.SeuilRecollement, "gateLe", c.GateLe)
	return c.SeuilRecollement
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

// combleZonesEntieresDe dit si cette carte comble AUSSI ses vides ouverts. Journalisé : c'est le
// levier qui avait noyé Illusion, il ne doit jamais s'armer en silence.
func (e *environnement) combleZonesEntieresDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.CombleZonesEntieres {
		return false
	}
	slog.Info("mapfond: comblement des vides OUVERTS arme pour cette carte (refute sur Illusion)",
		"carte", cle, "gateLe", c.GateLe)
	return true
}
