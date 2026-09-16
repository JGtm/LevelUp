package replaylabels

// vehicle_families_test.go — LE GARDE-RAIL DES FAMILLES DE CHASSIS QUALIFIEES (lot 1.9.9,
// 2026-09-16).
//
// DEUX FICHIERS DECLARENT LA MEME CHOSE, ET ILS DOIVENT S ACCORDER :
//
//	config/titles/{slug}/mappings/replay_labels.toml   [[vehicle_families]].sprite
//	static/vehicles-assets/{slug}/replay/index.json    la ligne `file` de la famille
//
// `sprite = true` dit au SERVICE de composer une URL ; `file` dit au CLIENT de quel PNG il
// s agit et a quelle echelle le dessiner. Les deux qui divergent produisent l une des deux
// pannes que ce lot existe pour supprimer : une URL morte (404 par match) ou un asset servi que
// personne ne demande. La regle du depot (<= 2 copies d un meme fait, et un garde-rail a la
// troisieme) s applique telle quelle.
//
// CE TEST LIT LES FICHIERS REELS DU DEPOT, pas une fixture : une DDL de test recopiee derive sans
// que rien ne le voie.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// spriteIndexEntry — le sous-ensemble de `index.json` que ce garde-rail lit.
type spriteIndexEntry struct {
	File    string `json:"file"`
	Famille string `json:"famille"`
	Statut  string `json:"statut"`
}

// TestVehicleFamiliesQualifieesAccordeesALIndexDesSprites : `sprite` du manifeste du titre et la
// presence d un `file` dans l index des assets disent-ils la meme chose ?
func TestVehicleFamiliesQualifieesAccordeesALIndexDesSprites(t *testing.T) {
	root := repoRoot(t)
	cat, err := Load(root, "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	if len(cat.VehicleFamilies) == 0 {
		t.Skip("le titre ne qualifie aucune famille de chassis : rien a accorder")
	}
	index := lireIndexDesSprites(t, root, "halo_infinite")
	for fam, info := range cat.VehicleFamilies {
		e, present := index[fam]
		if !present {
			t.Errorf("famille %q qualifiee par replay_labels.toml mais ABSENTE de l index des "+
				"sprites : le garde-rail du paquet `replay` la refusera, et le client n aurait "+
				"aucune echelle pour elle", fam)
			continue
		}
		if info.Sprite && e.File == "" {
			t.Errorf("famille %q : `sprite = true` dans replay_labels.toml, mais l index des "+
				"sprites ne porte AUCUN `file` — le service composerait une URL morte", fam)
		}
		if !info.Sprite && e.File != "" {
			t.Errorf("famille %q : `sprite = false` dans replay_labels.toml alors que l index "+
				"sert %q — l asset existe et personne ne le demande ; passer `sprite = true`",
				fam, e.File)
		}
	}
}

// TestTourelleAutoBannieQualifieeParLeTitre : la famille du lot 1.9.9 est declaree, bilingue,
// de nature `map_element`, et SANS asset a ce jour.
//
// POURQUOI UN TEST NOMME SUR UNE SEULE FAMILLE. C est la seule que le titre qualifie, et sa
// qualification est ce qui fait la difference entre « dessinee comme un element de carte » et
// « marqueur neutre » — la decision utilisateur du 2026-09-14 tient a ces quatre champs.
func TestTourelleAutoBannieQualifieeParLeTitre(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	info, ok := cat.VehicleFamilies["tourelle_auto_bannie"]
	if !ok {
		t.Fatal("famille `tourelle_auto_bannie` non qualifiee par replay_labels.toml : sans elle " +
			"le document ne publie ni nature ni libelle, et le client retombe sur le marqueur neutre")
	}
	if info.Kind != mappings.VehicleFamilyKindMapElement {
		t.Errorf("nature = %q, attendu %q", info.Kind, mappings.VehicleFamilyKindMapElement)
	}
	if info.En == "" || info.Fr == "" {
		t.Errorf("libelle incomplet (en=%q fr=%q) : les deux langues sont obligatoires",
			info.En, info.Fr)
	}
	if info.Sprite {
		t.Error("`sprite = true` : aucun asset n existe pour cette famille a ce jour — le jour ou " +
			"l utilisateur en fournit un, c est CETTE ligne et l index des sprites qui changent")
	}
}

// lireIndexDesSprites rend l index des assets de vehicule du titre, keye par famille.
func lireIndexDesSprites(t *testing.T, root, slug string) map[string]spriteIndexEntry {
	t.Helper()
	path := filepath.Join(root, "static", "vehicles-assets", slug, "replay", "index.json")
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fige dans le test
	if err != nil {
		t.Skipf("index des sprites absent (%v) — garde-rail saute", err)
	}
	var entries []spriteIndexEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("index des sprites illisible (%s) : %v", path, err)
	}
	out := make(map[string]spriteIndexEntry, len(entries))
	for _, e := range entries {
		out[e.Famille] = e
	}
	return out
}
