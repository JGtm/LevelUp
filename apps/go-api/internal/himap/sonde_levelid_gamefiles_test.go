package himap

// LA PREUVE level_id DES APPARIEMENTS carte -> module, MATERIALISEE EN TEST DURABLE.
//
// POURQUOI CE TEST EXISTE. `cmd/mapquant-build` ne catalogue une carte que sur PREUVE
// EXTERNE a toute mesure : identite de nom, ou methode level_id (plan maitre §J0.2,
// Vagabond). La preuve Vagabond avait ete jouee par un outil jetable (`cmd/tmp_forgedim`,
// supprime) ; ce test la rejoue a chaque passage sur les modules INSTALLES, pour chaque
// entree de `mapModule` etablie par level_id — les deux temoins historiques (Catalyst,
// Vagabond) plus les cartes debloquees le 2026-08-13.
//
// LA METHODE, ET SES DEUX VERROUS :
//
//  1. Le level_id est lu dans `<carte>_map.mvar` — la variante PAR DEFAUT, dont le nom ne
//     designe AUCUN module. La preuve ne presuppose donc pas la regle de nommage des
//     fichiers-liens (`resolution.go`), elle la corrobore : le fichier-lien
//     `<carte>_<module>.mvar` doit porter le MEME level_id.
//  2. Le level_id est balaye sur TOUS les modules de `deploy/any/levels` + `deploy/ds/levels`
//     (champ GlobalID a +0x28 de l'entree, groupe `levl`) et doit designer EXACTEMENT UN
//     dossier installe — celui attendu. Deux occurrences = la preuve ne departage pas,
//     zero = elle ne prouve rien : dans les deux cas le test est ROUGE.
//
// Le depot de variantes (`.ai/re_dump/mapvar`) et l'installation du jeu ne sont pas
// versionnes : leur absence fait SKIP avec sa raison, jamais un vert silencieux.

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

// preuvesLevelID : les appariements affiches -> installes tenus par la methode level_id.
// MEME TABLE que `mapModule` (cmd/mapquant-build/main.go) pour les entrees non etablies
// par identite de nom — si une ligne bouge la-bas, elle bouge ici.
var preuvesLevelID = []struct {
	Carte  string // nom affiche (match_registry.map_name)
	Module string // dossier installe attendu
}{
	// Temoins historiques de la methode (etablis en 2026-07-31, plan maitre §J0.2).
	{"Catalyst", "catalyst"},
	{"Vagabond", "fo08_wetland"},
	// Cartes debloquees le 2026-08-13 (lot bornes de dequantification).
	{"Corpo", "fo11_blank"},
	{"Deadlock", "btb_drydock"},
	{"Live Fire", "sgh_interlock"},
	{"Oasis", "btb_exiled"},
	{"Prism", "sgh_crystalcaves"},
	{"Recharge", "sgh_blueprint"},
	{"Scarr", "btb_engine"},
	// Pilotes du lot fonds par map_id (2026-08-13) : cartes Forge seules sur leur canevas.
	{"Starboard", "fo03_space"},
	{"Dredge", "fo06_deepsea"},
	// Masse du lot fonds par map_id (2026-08-13), ordre du plan (nombre de matchs). Le
	// module attendu vient du fichier-lien du depot (`<carte>_<module>.mvar`) ; la sonde
	// le corrobore par le level_id, en exigeant l'unicite sur l'installation.
	{"The Pit", "fo09_academy"},
	{"Snowbound", "fo13_frost"},
	{"Empyrean", "fo11_blank"},
	{"Origin", "fo08_wetland"},
	{"Absolution", "fo09_academy"},
	{"Curfew", "fo11_blank"},
	{"Dynasty", "fo08_wetland"},
	{"Shiro", "fo05_desert"},
	{"Cliffside", "fo05_desert"},
	{"Nemesis", "fo08_wetland"},
	{"Domicile", "fo05_desert"},
	{"Fortress", "fo09_academy"},
	{"Goliath", "fo11_blank"},
	{"Isolation", "fo08_wetland"},
	{"Solitude", "fo11_blank"},
	// Declaree le 2026-08-27 : la variante CLASSEE de Solitude est un autre asset UGC, jouee
	// 5 fois, et elle n avait pas de fond. Seule des 29 cartes du reliquat a etre cuisinable —
	// les 28 autres n ont aucune ancre dans le catalogue d objectifs.
	{"Solitude - Ranked", "fo11_blank"},
	{"Houseki", "fo09_academy"},
	{"High Ground", "fo08_wetland"},
	{"Salvation", "fo11_blank"},
	{"Takamanohara", "fo11_blank"},
	{"Elevation", "fo11_blank"},
	{"Kiken'na", "fo08_wetland"},
	{"Banished Narrows", "fo05_desert"},
	{"Kaiketsu", "fo05_desert"},
	{"Obituary", "fo09_academy"},
	{"Opulence", "fo11_blank"},
	{"Command", "fo09_academy"},
	{"Fortitude", "fo05_desert"},
	{"Refuge", "fo08_wetland"},
	{"Critical Dewpoint", "fo11_blank"},
	{"Perilous", "fo08_wetland"},
	{"Shogun", "fo11_blank"},
	{"Sylvanus", "fo05_desert"},
	{"Smallhalla", "fo08_wetland"},
	// Reliquat du registre, instruit le 2026-08-16 (PLAN_ALERTES_REPLAY_PARTOUT phase 3) :
	// les cartes Forge jouees 1 a 8 fois, restees hors catalogue au lot du 2026-08-13 faute
	// d'avoir ete demandees. MEME methode, MEME sonde, MEME exigence d'unicite.
	//
	// ATTENTION — QUATORZE DE CES VINGT-DEUX NE SONT PAS AU CATALOGUE, et c'est voulu : leur
	// module est PROUVE (c'est ce que cette table atteste) mais ses BORNES sont fausses. Le
	// controle `DetectI0Layout` du meme lot rend un desaccord SYSTEMATIQUE sur les canevas
	// `fo05_desert`, `fo11_blank` et `fo13_frost` — bornes -> [18 18 18], films -> [15 15 17],
	// 12 films sur 12. Le meme cas de figure que `Live Fire` ci-dessus : module etabli,
	// bornes indisponibles. Voir le commentaire de `mapModule` (cmd/mapquant-build) et le
	// registre des reports.
	{"Ecotone", "fo11_blank"},
	{"Threshold", "fo11_blank"},
	{"Pharaoh", "fo11_blank"},
	{"Credence", "fo11_blank"},
	{"Disciple", "fo11_blank"},
	{"Nadair", "fo11_blank"},
	{"Warehouse", "fo11_blank"},
	{"Insolence", "fo09_academy"},
	{"Merchant's Square", "fo09_academy"},
	{"Urban Raid", "fo09_academy"},
	{"Cole Protocol", "fo09_academy"},
	{"Solution", "fo05_desert"},
	{"Flood Gulch", "fo05_desert"},
	{"Dawnbreaker", "fo05_desert"},
	{"Vallaheim Firefight", "fo05_desert"},
	{"Thunderhead", "fo08_wetland"},
	{"Ronin", "fo08_wetland"},
	{"Rat's Nest", "fo08_wetland"},
	{"Scarlett's Landing", "fo08_wetland"},
	{"Outlook", "fo13_frost"},
	// `Lattice - Ranked` garde son suffixe ICI : le depot ne porte aucun `lattice_map.mvar`,
	// et la preuve doit nommer un fichier qui existe. Au catalogue la cle se normalise en
	// `lattice` (`NormalizeMapName` retire ` - ranked`), ce que le registre attend.
	{"Lattice - Ranked", "fo13_frost"},
	// Carte dont l'API n'a JAMAIS resolu le nom d'affichage : `map_name` vaut l'identifiant
	// d'asset. Elle se prouve comme les autres — son `.mvar` est au depot sous ce nom.
	{"944396dd-5661-4a16-b1d8-a6053f762c55", "fo13_frost"},
}

// TestPreuveLevelIDCartes rejoue la preuve level_id de chaque appariement : lecture du
// level_id dans le .mvar par defaut, corroboration par le fichier-lien, unicite du dossier
// installe qui porte ce level_id en tag `levl`.
func TestPreuveLevelIDCartes(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	index, nModules := indexLevlInstalle(t, root)
	t.Logf("index levl : %d level_id distincts sur %d modules (any/levels + ds/levels)", len(index), nModules)

	for _, c := range preuvesLevelID {
		t.Run(c.Carte, func(t *testing.T) {
			id := levelIDDuDepot(t, mvarParDefaut(c.Carte))
			if lien := levelIDDuDepot(t, mvarLien(c.Carte, c.Module)); lien != id {
				t.Fatalf("fichier-lien en desaccord : %s_map.mvar porte %d, le lien %s porte %d",
					mvarBase(c.Carte), id, c.Module, lien)
			}
			dossiers := dossiersDistincts(index[uint32(id)])
			t.Logf("%s : level_id %d (0x%08X) -> %v", c.Carte, id, uint32(id), dossiers)
			if len(dossiers) != 1 {
				t.Fatalf("le level_id %d designe %d dossiers (%v) — la preuve exige l'unicite",
					id, len(dossiers), dossiers)
			}
			if dossiers[0] != c.Module {
				t.Fatalf("le level_id %d designe %q, attendu %q", id, dossiers[0], c.Module)
			}
		})
	}
}

// mvarBase : nom de fichier du depot pour une carte affichee (minuscules, espaces -> `_`).
func mvarBase(carte string) string {
	return strings.ReplaceAll(strings.ToLower(carte), " ", "_")
}

func mvarParDefaut(carte string) string    { return mvarBase(carte) + "_map.mvar" }
func mvarLien(carte, module string) string { return mvarBase(carte) + "_" + module + ".mvar" }

// levelIDDuDepot lit le level_id (root[1][0][0]) d'un .mvar du depot de variantes.
func levelIDDuDepot(t *testing.T, fichier string) int32 {
	t.Helper()
	chemin, err := cheminDepuisDepot(filepath.Join(DepotVariantesCarte, fichier))
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatalf("%s : %v", fichier, err)
	}
	if v.LevelID == 0 {
		t.Fatalf("%s : level_id nul — la preuve ne prouverait rien", fichier)
	}
	return v.LevelID
}

// indexLevlInstalle balaye tous les .module de `any/levels` + `ds/levels` et rend
// GlobalID des tags `levl` -> chemins des modules porteurs, plus le compte de modules.
//
// Un module illisible est une ERREUR, jamais un saut silencieux : un balayage partiel
// affaiblirait la pretention d'unicite sans que rien ne le dise.
func indexLevlInstalle(t *testing.T, deployRoot string) (map[uint32][]string, int) {
	t.Helper()
	index := map[uint32][]string{}
	n := 0
	for _, variante := range []string{"any", "ds"} {
		racine := filepath.Join(deployRoot, variante, "levels")
		err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".module") {
				return err
			}
			m, openErr := himodule.Open(p)
			if openErr != nil {
				t.Fatalf("module illisible, balayage invalide : %s : %v", p, openErr)
			}
			n++
			for _, f := range m.Files("levl") {
				index[f.GlobalID] = append(index[f.GlobalID], p)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("balayage %s : %v", racine, err)
		}
	}
	if n == 0 {
		t.Fatal("aucun module balaye — l'unicite ne se mesurerait sur rien")
	}
	return index, n
}

// dossiersDistincts reduit des chemins de modules a leurs dossiers de carte, dedupliques
// (le meme dossier existe sous any/ et ds/ : c'est la MEME carte, pas deux occurrences).
func dossiersDistincts(chemins []string) []string {
	var out []string
	vus := map[string]bool{}
	for _, p := range chemins {
		d := filepath.Base(filepath.Dir(p))
		if !vus[d] {
			vus[d] = true
			out = append(out, d)
		}
	}
	return out
}
