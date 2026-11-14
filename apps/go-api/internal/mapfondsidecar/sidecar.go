// Package mapfondsidecar lit et écrit le sidecar d'un fond de carte dans sa forme complète :
// le calage du décodeur de film (`replay.MapBackground`) plus le plan des murs.
//
// Le décodeur ne connaît pas le plan : `replay.LoadMapBackground` l'ignore à la lecture et une
// réécriture par `replay.MapBackground` le perdrait. Tout outil qui réécrit un sidecar passe
// donc par ce paquet, sans quoi le plan disparaît sans erreur.
package mapfondsidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// finObjet est ce que `json.MarshalIndent` écrit après le dernier membre d'un objet racine ;
// le plan s'insère juste avant.
const finObjet = "\n}"

// suffixeImagePlan sépare la clé de l'extension dans le nom de l'image plan d'un fond
// (`<cle>.plan.webp`) : l'image plan se pose sur le fond avec le même calage et ne s'en distingue
// que par ce suffixe.
const suffixeImagePlan = ".plan"

// NomImagePlan rend le nom du fichier image plan d'une clé de fond pour l'extension `ext`
// (point compris : `.png`, `.webp`).
func NomImagePlan(cle, ext string) string {
	return cle + suffixeImagePlan + ext
}

// CleDeImagePlan rend la clé de fond dont `nomFichier` (sans dossier) est l'image plan, et dit si
// c'en est une. Seul point qui sait lire le nom d'une image plan : l'outil de conversion en WebP
// s'en sert pour retrouver le sidecar d'une image qu'il convertit.
func CleDeImagePlan(nomFichier string) (cle string, ok bool) {
	base := strings.TrimSuffix(nomFichier, filepath.Ext(nomFichier))
	cle, ok = strings.CutSuffix(base, suffixeImagePlan)
	return cle, ok && cle != ""
}

// MapBackground est le fichier `<cle>.json` d'un fond : le calage du décodeur, aplati, suivi du plan.
// Sans plan, la sérialisation est celle de `replay.MapBackground` seule. C'est le document
// STOCKÉ de la racine « fond de carte » : il porte le nom de son jumeau servi
// (`replaydoc.MapBackground`) et la projection `replayview.MapBackgroundOf` en part tout entière.
type MapBackground struct {
	replay.MapBackground
	Plan *replaydoc.MapBackgroundPlan `json:"plan,omitempty"`
	// PlanImage : nom du fichier image plan du dossier des fonds, mêmes dimensions et même calage
	// que l'image du fond ; vide sans plan. `Write` l'écrit avant le plan, que le plan à une arête
	// par ligne suit en fin d'objet.
	PlanImage string `json:"planImage,omitempty"`
}

// Load lit un sidecar complet. Les contrôles du calage (version de schéma) sont ceux de
// `replay.LoadMapBackground`.
func Load(chemin string) (*MapBackground, error) {
	bg, err := replay.LoadMapBackground(chemin)
	if err != nil {
		return nil, err
	}
	ext, err := lireExtensions(chemin)
	if err != nil {
		return nil, err
	}
	return &MapBackground{MapBackground: *bg, PlanImage: ext.PlanImage, Plan: ext.Plan}, nil
}

// extensionsSidecar : ce que le sidecar porte en plus du calage du décodeur.
type extensionsSidecar struct {
	PlanImage string                       `json:"planImage"`
	Plan      *replaydoc.MapBackgroundPlan `json:"plan"`
}

// lireExtensions ne lit que le plan et l'image plan d'un sidecar : vides quand il n'en porte pas.
func lireExtensions(chemin string) (extensionsSidecar, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // asset versionné, chemin résolu par PathResolver
	if err != nil {
		return extensionsSidecar{}, fmt.Errorf("fond de carte illisible (%s) : %w", chemin, err)
	}
	var ext extensionsSidecar
	if err := json.Unmarshal(blob, &ext); err != nil {
		return extensionsSidecar{}, fmt.Errorf("fond de carte invalide (%s) : %w", chemin, err)
	}
	return ext, nil
}

// Write écrit le sidecar avec l'indentation de la cuisson, sauf les arêtes du plan : une par
// ligne, `[x1,y1,x2,y2]` en clair. L'indentation complète en poserait un nombre par ligne
// (plus d'un mégaoctet sur une carte de grand gabarit) et rendrait le diff illisible.
func Write(chemin string, s MapBackground) error {
	plan := s.Plan
	s.Plan = nil
	blob, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation du sidecar %s : %w", chemin, err)
	}
	if plan != nil {
		bloc, err := membrePlan(plan)
		if err != nil {
			return fmt.Errorf("sérialisation du plan de %s : %w", chemin, err)
		}
		blob = append(blob[:len(blob)-len(finObjet)], bloc...)
	}
	return os.WriteFile(chemin, append(blob, '\n'), 0o644) //nolint:gosec // asset versionné
}

// membrePlan rend le membre `"plan"` à poser en fin d'objet, virgule d'ouverture et accolade
// finale de l'objet comprises : les arêtes une par ligne.
func membrePlan(p *replaydoc.MapBackgroundPlan) ([]byte, error) {
	source, err := json.Marshal(p.Source)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, ",\n  \"plan\": {\n    \"source\": %s,\n    \"vertices\": %d,\n    \"faces\": %d,\n    \"edges\": [",
		source, p.Vertices, p.Faces)
	for i, e := range p.Edges {
		ligne, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n      ")
		b.Write(ligne)
	}
	if len(p.Edges) > 0 {
		b.WriteString("\n    ")
	}
	b.WriteString("]\n  }")
	b.WriteString(finObjet)
	return b.Bytes(), nil
}
