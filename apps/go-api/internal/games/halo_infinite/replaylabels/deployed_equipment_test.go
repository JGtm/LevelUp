package replaylabels

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// TestLoad_FormesDeployeesDuManifesteReel : la table des formes d'equipement deployees que le catalogue passe au
// decodeur est EXACTEMENT l'ensemble des `[[equipment_objects]]` du VRAI manifeste `replay_labels.toml` marques
// `kind = "deployed"`, et elle n'est pas vide. Sans elle le decodeur prendrait deux murs poses au meme endroit pour
// un faux socle. La lecture du fichier est independante du chargeur (autre decodage du meme TOML).
func TestLoad_FormesDeployeesDuManifesteReel(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "replay_labels.toml"))
	if err != nil {
		t.Fatalf("manifeste du titre : %v", err)
	}
	var doc struct {
		Objets []struct {
			ID   string `toml:"id"`
			Kind string `toml:"kind"`
		} `toml:"equipment_objects"`
	}
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("lecture du manifeste : %v", err)
	}
	attendu := map[uint32]bool{}
	for _, o := range doc.Objets {
		if o.Kind != "deployed" {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(o.ID), "0x"), 16, 32)
		if err != nil {
			t.Fatalf("identifiant %q : %v", o.ID, err)
		}
		attendu[uint32(n)] = true
	}
	if len(attendu) == 0 {
		t.Fatal("le manifeste ne marque aucune forme deployee : le test ne prouverait rien")
	}

	cat, err := Load(root, "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	if len(cat.DeployedEquipment) == 0 {
		t.Fatal("DeployedEquipment est vide : la table n'arrive plus au decodeur")
	}
	if got, want := cles(cat.DeployedEquipment), cles(attendu); got != want {
		t.Errorf("formes deployees passees au decodeur = %s, attendu exactement %s", got, want)
	}
}

func cles(m map[uint32]bool) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, fmt.Sprintf("0x%08x", k))
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
