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

	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// finObjet est ce que `json.MarshalIndent` écrit après le dernier membre d'un objet racine ;
// le plan s'insère juste avant.
const finObjet = "\n}"

// Sidecar est le fichier `<cle>.json` d'un fond : le calage du décodeur, aplati, suivi du plan.
// Sans plan, la sérialisation est celle de `replay.MapBackground` seule.
type Sidecar struct {
	replay.MapBackground
	Plan *replaydoc.MapBackgroundPlan `json:"plan,omitempty"`
}

// Load lit un sidecar complet. Les contrôles du calage (version de schéma) sont ceux de
// `replay.LoadMapBackground`.
func Load(chemin string) (*Sidecar, error) {
	bg, err := replay.LoadMapBackground(chemin)
	if err != nil {
		return nil, err
	}
	plan, err := LoadPlan(chemin)
	if err != nil {
		return nil, err
	}
	return &Sidecar{MapBackground: *bg, Plan: plan}, nil
}

// LoadPlan ne lit que le plan d'un sidecar : nil quand il n'en porte pas.
func LoadPlan(chemin string) (*replaydoc.MapBackgroundPlan, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // asset versionné, chemin résolu par PathResolver
	if err != nil {
		return nil, fmt.Errorf("fond de carte illisible (%s) : %w", chemin, err)
	}
	var seul struct {
		Plan *replaydoc.MapBackgroundPlan `json:"plan"`
	}
	if err := json.Unmarshal(blob, &seul); err != nil {
		return nil, fmt.Errorf("fond de carte invalide (%s) : %w", chemin, err)
	}
	return seul.Plan, nil
}

// Write écrit le sidecar avec l'indentation de la cuisson, sauf les arêtes du plan : une par
// ligne, `[x1,y1,x2,y2]` en clair. L'indentation complète en poserait un nombre par ligne
// (plus d'un mégaoctet sur une carte de grand gabarit) et rendrait le diff illisible.
func Write(chemin string, s Sidecar) error {
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
