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
	// PlancherTranche : profondeur en metres SOUS le niveau de jeu (valeur NEGATIVE) en deca
	// de laquelle la matiere sort de la carte. Zero = -12 m. Voir OptionsCuisson.
	PlancherTranche float64 `json:"plancherTranche,omitempty"`
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
