package main

// L'avis du user sur le croisement du fond et du maillage de navigation, carte par carte, et la
// décision de cuisson qui en découle. Le fichier d'avis est une DONNÉE (`-avis`) : la chaîne de
// `himap` ne connaît ni cartes ni avis, seul cet outil traduit l'un dans les options de l'autre.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// États d'avis lus dans le fichier (`exceptions[cle].etat`) et états dérivés écrits au rapport.
const (
	etatPublie       = "publie"
	etatFaces        = "faces"
	etatSilhouette   = "silhouette"
	etatAucune       = "aucune"
	etatSansAvis     = "sans_avis"
	etatSansMaillage = "sans_maillage"
	etatDefaut       = "defaut"
	etatReglage      = "reglage"

	// États d'un fichier d'avis complémentaire : ignorer laisse la carte entièrement telle qu'elle
	// est (ni fond ni plan recalculés), reprendre la recuit comme les autres et la signale à part,
	// nouveau valide le résultat (rien à faire).
	etatIgnorer   = "ignorer"
	etatReprendre = "reprendre"
	etatNouveau   = "nouveau"
)

// avisFond est le contenu du fichier d'avis. Les champs non lus (`definitions`, `note`, `date`)
// restent dans le fichier pour le lecteur humain.
type avisFond struct {
	Defaut       string               `json:"defaut"`
	SansAvis     []string             `json:"sans_avis"`
	SansMaillage []string             `json:"sans_maillage"`
	Exceptions   map[string]avisCarte `json:"exceptions"`
	index        map[string]string    // cle -> etat dérivé des deux listes
	complement   map[string]string    // cle -> ignorer ou reprendre, des fichiers d avis complementaires
}

// avisCarte est l'avis d'une carte qui s'écarte du défaut ou porte un commentaire.
type avisCarte struct {
	Carte       string `json:"carte"`
	Etat        string `json:"etat"`
	Commentaire string `json:"commentaire"`
}

// decisionFond dit ce que la cuisson fait d'une carte au vu de l'avis et du réglage.
type decisionFond struct {
	// Etat : l'état d'avis, tel qu'il se lit au rapport.
	Etat string
	// Saute : la carte n'est pas touchée (fond publié gardé, ou carte sans maillage).
	Saute bool
	// Rogne : le rognage au maillage est armé par l'avis (le maillage sert alors de référence).
	Rogne bool
	// Mode : forme du masque de rognage.
	Mode himap.ModeNavmesh
	// Ignore : la carte n'est touchée en rien, pas même son plan (avis `ignorer`).
	Ignore bool
	// Reprise : l'avis complémentaire demande de la reprendre ; le rapport la liste à part.
	Reprise bool
	// ARevoir : la carte est cuite faute d'avis exploitable ; le rapport et la planche la marquent.
	ARevoir bool
}

// chargeAvis lit le fichier d'avis et refuse un défaut ou un état qu'il ne sait pas traduire :
// une carte cuite dans un mode que le fichier ne demandait pas passerait le gate sous une fausse
// identité. Le premier fichier donne la règle ; chacun des suivants (avis complémentaires)
// n'apporte que des états `ignorer`, `reprendre` ou `nouveau` par carte.
func chargeAvis(chemins ...string) (*avisFond, error) {
	if len(chemins) == 0 {
		return nil, errors.New("aucun fichier d'avis")
	}
	a, err := lisAvis(chemins[0])
	if err != nil {
		return nil, err
	}
	if !etatTraduisible(a.Defaut) || a.Defaut == "" || a.Defaut == etatAucune {
		return nil, fmt.Errorf("fichier d'avis : défaut %q inconnu (silhouette, faces ou publie)", a.Defaut)
	}
	for cle, c := range a.Exceptions {
		if !etatTraduisible(c.Etat) {
			return nil, fmt.Errorf("fichier d'avis : état %q inconnu pour %q", c.Etat, cle)
		}
	}
	a.index = map[string]string{}
	for _, cle := range a.SansAvis {
		a.index[cle] = etatSansAvis
	}
	for _, cle := range a.SansMaillage {
		a.index[cle] = etatSansMaillage
	}
	a.complement = map[string]string{}
	for _, chemin := range chemins[1:] {
		c, err := lisAvis(chemin)
		if err != nil {
			return nil, err
		}
		for cle, exc := range c.Exceptions {
			switch exc.Etat {
			case etatIgnorer, etatReprendre:
				a.complement[cle] = exc.Etat
			case etatNouveau:
			default:
				return nil, fmt.Errorf("avis complémentaire %s : état %q inconnu pour %q (ignorer, reprendre ou nouveau)", chemin, exc.Etat, cle)
			}
		}
	}
	return a, nil
}

// lisAvis lit un fichier d'avis sans le valider.
func lisAvis(chemin string) (*avisFond, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // donnée versionnée passée en argument
	if err != nil {
		return nil, fmt.Errorf("fichier d'avis illisible : %w", err)
	}
	var a avisFond
	if err := json.Unmarshal(blob, &a); err != nil {
		return nil, fmt.Errorf("fichier d'avis invalide (%s) : %w", chemin, err)
	}
	return &a, nil
}

// etatTraduisible dit si la valeur est un état que decide() sait traduire. Vide = « pas de choix,
// commentaire seul » : la carte est cuite en silhouette et marquée à revoir.
func etatTraduisible(etat string) bool {
	switch etat {
	case etatPublie, etatFaces, etatSilhouette, etatAucune, "":
		return true
	}
	return false
}

// decide rend la décision de l'avis pour une carte, sans tenir compte du réglage.
//
// Règle validée par le user (fichier d'avis) : défaut `silhouette` ; `publie` = ne rien recuire ;
// `aucune` ou vide = le défaut venait de la sélection de composante, corrigée dans la brique :
// recuite en silhouette et marquée à revoir ; `sans_avis` = silhouette, à revoir ;
// `sans_maillage` = rien à faire.
func (a *avisFond) decide(cle string) decisionFond {
	switch a.complement[cle] {
	case etatIgnorer:
		return decisionFond{Etat: etatIgnorer, Saute: true, Ignore: true}
	case etatReprendre:
		return decisionFond{Etat: etatReprendre, Rogne: true, Mode: himap.ModeSilhouette, ARevoir: true, Reprise: true}
	}
	if etat, ok := a.index[cle]; ok {
		if etat == etatSansMaillage {
			return decisionFond{Etat: etat, Saute: true}
		}
		return decisionFond{Etat: etat, Rogne: true, Mode: himap.ModeSilhouette, ARevoir: true}
	}
	exc, ok := a.Exceptions[cle]
	if !ok {
		return a.deEtat(etatDefaut, a.Defaut)
	}
	d := a.deEtat(exc.Etat, exc.Etat)
	if exc.Etat == "" {
		d.Etat = etatAucune + " (commentaire seul)"
	}
	return d
}

// deEtat traduit un état d'avis en décision ; `libelle` est l'état affiché au rapport.
func (a *avisFond) deEtat(libelle, etat string) decisionFond {
	switch etat {
	case etatPublie:
		return decisionFond{Etat: libelle, Saute: true}
	case etatFaces:
		return decisionFond{Etat: libelle, Rogne: true, Mode: himap.ModeFaces}
	case etatSilhouette:
		return decisionFond{Etat: libelle, Rogne: true, Mode: himap.ModeSilhouette}
	}
	// `aucune` ou vide.
	return decisionFond{Etat: libelle, Rogne: true, Mode: himap.ModeSilhouette, ARevoir: true}
}

// decisionFond rend la décision de cuisson d'une carte. Sans `-avis`, rien n'est décidé ici : la
// carte garde son comportement d'avant (rognage natif seul, Forge selon son réglage). Le réglage
// `navmeshMode` de la carte prime sur l'avis, y compris sur « publie ».
func (e *environnement) decisionFond(cle string) decisionFond {
	reglage := e.navmeshModeDe(cle)
	if e.avis == nil {
		return decisionFond{Etat: etatReglage, Mode: reglage}
	}
	d := e.avis.decide(cle)
	if reglage != "" {
		d = decisionFond{Etat: etatReglage, Rogne: true, Mode: reglage}
	}
	return d
}

// navmeshModeDe rend le mode de rognage propre à la carte (`navmeshMode`), vide si absent.
// Journalisé : il prime sur l'avis.
func (e *environnement) navmeshModeDe(cle string) himap.ModeNavmesh {
	if e.reglages == nil {
		return ""
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || c.NavmeshMode == "" {
		return ""
	}
	slog.Info("mapfond: mode de rognage au maillage propre a la carte", "carte", cle,
		"mode", c.NavmeshMode, "gateLe", c.GateLe)
	return himap.ModeNavmesh(c.NavmeshMode)
}

// modeNavmeshValide dit si le mode d'un réglage est l'un de ceux de la brique.
func modeNavmeshValide(mode string) bool {
	return mode == "" || mode == string(himap.ModeFaces) || mode == string(himap.ModeSilhouette)
}
