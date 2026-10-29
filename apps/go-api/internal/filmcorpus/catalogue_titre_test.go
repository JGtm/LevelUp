package filmcorpus

// catalogue_titre_test.go — LE CATALOGUE DE LIBELLES DU TITRE, LU DEPUIS SES VRAIS FICHIERS, ET SON
// INSTANTANE DANS LA BRIQUE (plan D2, item 2.11).
//
// POURQUOI LE VRAI ET PAS UNE FIXTURE. Depuis le lot 3.2, les noms d'armes, de grenades et de
// capacites ne sont plus dans le code : ils vivent dans `config/titles/halo_infinite/mappings/`.
// Un golden bati sur des noms inventes ne verrait PAS une regression de configuration (un libelle
// perdu, une langue vide, un effet de tir renomme). Les goldens d'assemblage du rejeu sont batis
// sur le vrai catalogue ; ce fichier est le garde-rail de ces trois tables.
//
// POURQUOI UN INSTANTANE. La brique n'importe plus le chargeur des libelles de LevelUp : ses tests
// lisent le catalogue du titre dans `film/replay/testdata/golden_catalogue_titre.json`, produit
// ICI par le chargeur et la jointure de la production. Le test d'instantane exige que le fichier
// soit celui que les fichiers du titre donnent AUJOURD'HUI ; un libelle qui change rougit ce test
// d'abord, l'instantane se regenere, puis les goldens d'assemblage rougissent et se regenerent a
// leur tour — la chaine est la meme qu'avant, avec un maillon visible de plus.
//
// REGENERATION (jamais d'edition a la main) — DEUX verrous, comme les autres portes d'ecriture :
//
//	LEVELUP_UPDATE_CATALOGUE_TITRE=1 go test ./internal/filmcorpus/ -run InstantaneDuCatalogue \
//	  -update-catalogue-titre

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

var majInstantaneCatalogue = flag.Bool("update-catalogue-titre", false,
	"reecrit l'instantane du catalogue de libelles du titre (avec LEVELUP_UPDATE_CATALOGUE_TITRE=1)")

const majInstantaneCatalogueEnv = "LEVELUP_UPDATE_CATALOGUE_TITRE"

// cheminInstantaneCatalogue : l'instantane, depuis la racine du depot.
var cheminInstantaneCatalogue = filepath.Join("libs", "film", "replay", "testdata", "golden_catalogue_titre.json")

// texteEnFr fabrique le libelle d'un test dans les deux langues du parc, chaque langue presente,
// vide comprise : le fil que le producteur pose.
func texteEnFr(en, fr string) replay.Text {
	return replay.Text{"en": en, "fr": fr}
}

// catalogueDuTitre charge le catalogue du titre de reference depuis le depot.
//
// La jointure elle-meme (famille -> weapon_key -> nom/effet) N'EST PAS recopiee : on appelle
// `replay.NewLabelCatalog`, exactement comme la production. Une jointure recopiee dans un test est
// une jointure qui derive.
func catalogueDuTitre(t *testing.T) replay.LabelCatalog {
	t.Helper()
	dir := filepath.Join(racineDuDepot(t), "config", "titles", "halo_infinite", "mappings")

	names, err := mappings.LoadWeaponNamesFromFile(filepath.Join(dir, "weapon_names.toml"))
	if err != nil {
		t.Fatalf("weapon_names.toml : %v", err)
	}
	labels := manifesteDuTitre(t)

	byKey := map[string]replay.Label{}
	for k, v := range names.Names() {
		byKey[k] = replay.Label{Text: texteEnFr(v.Pick("en"), v.Pick("fr"))}
	}
	var abilities []replay.AbilityPalette
	for _, p := range labels.AbilityPalettes() {
		ranks := map[int]replay.Label{}
		for rank, v := range p.Ranks {
			ranks[rank] = replay.Label{Text: texteEnFr(v.Text.Pick("en"), v.Text.Pick("fr"))}
		}
		abilities = append(abilities, replay.AbilityPalette{
			ID: p.ID, Markers: p.Markers, Ranks: ranks, Families: p.Families})
	}
	grenades := make([]replay.Label, 0, len(labels.GrenadeRanks()))
	for _, v := range labels.GrenadeRanks() {
		grenades = append(grenades, replay.Label{Text: texteEnFr(v.Text.Pick("en"), v.Text.Pick("fr"))})
	}
	cat := replay.NewLabelCatalog(
		weapons.FilmshellWeaponKeysByFamily(), byKey, labels.ShotEffects(), grenades, abilities)
	// Les familles d'objets d'equipement se posent APRES construction, comme en production
	// (cf. replaylabels.Load) : elles n'entrent dans aucune jointure du catalogue. Les oublier
	// ici ferait publier au golden un document dont TOUTES les poses sont `other`, alors que
	// le manifeste en nomme — c'est-a-dire un document que la production ne sert pas.
	cat.EquipmentFamilies = labels.EquipmentObjects()
	// LES FAMILLES MESUREES PAR LE CANAL D'IMPULSION, par le MEME chemin qu'en production
	// (cf. replaylabels.Load). Les oublier ici ferait publier au golden un document SANS
	// impulsion sur un film qui en porte — c'est-a-dire l'inverse de ce que la production sert.
	cat.AbilityImpulseFamilies = labels.AbilityImpulseFamilies()
	// LES FAMILLES MESUREES PAR LE CANAL DES CHARGES (i56), meme chemin, meme raison : les
	// oublier ici ferait publier au golden un document SANS releve de charge sur un film qui
	// en porte — l'inverse de ce que la production sert.
	cat.AbilityChargeFamilies = labels.AbilityChargeFamilies()
	// LE DRAPEAU, pour la MEME raison et par le MEME chemin que les familles d'equipement
	// (cf. replaylabels.Load) : sans cette table, la chaine des socles ne reconnaitrait pas
	// l'objet d'objectif du titre, et le golden figerait un document que la production ne sert
	// pas. Le filtrage par famille est celui de la couche titre — un seul proprietaire du
	// litteral, `mappings.ObjectiveFamilyFlag`.
	for id, o := range labels.ObjectiveObjects() {
		if o.Family != mappings.ObjectiveFamilyFlag {
			continue
		}
		if cat.ObjectiveObjects == nil {
			cat.ObjectiveObjects = map[uint32]replay.Label{}
		}
		cat.ObjectiveObjects[id] = replay.Label{Text: texteEnFr(o.Label.Text.Pick("en"), o.Label.Text.Pick("fr"))}
	}
	return cat
}

// manifesteDuTitre lit le manifeste de rejeu du titre de reference. Extrait de `catalogueDuTitre` :
// le garde-rail du drapeau interroge la table des objets d'objectif SANS avoir besoin de tout le
// catalogue.
func manifesteDuTitre(t *testing.T) *mappings.ReplayLabelSet {
	t.Helper()
	path := filepath.Join(racineDuDepot(t), "config", "titles", "halo_infinite", "mappings",
		"replay_labels.toml")
	labels, err := mappings.LoadReplayLabelsFromFile(path)
	if err != nil {
		t.Fatalf("replay_labels.toml : %v", err)
	}
	return labels
}

// racineDuDepot : la racine du depot, deduite de l'emplacement de `testutil` (versionne), jamais du
// cwd. Un fichier versionne absent est une installation cassee : echec, pas Skip.
func racineDuDepot(t *testing.T) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	return root
}

// TestCatalogueDuTitreNommeLesArmesDuFilmDeReference — le golden compte 22 familles nommees sur le
// film de reference ; ce test dit POURQUOI elles le sont, et il echoue si la chaine registre ->
// weapon_names.toml se rompt.
func TestCatalogueDuTitreNommeLesArmesDuFilmDeReference(t *testing.T) {
	cat := catalogueDuTitre(t)
	if len(cat.Weapons) == 0 {
		t.Fatal("aucune arme nommee : la jointure famille -> weapon_key -> nom est rompue")
	}
	for family, lbl := range cat.Weapons {
		if lbl.Text["en"] == "" || lbl.Text["fr"] == "" {
			t.Errorf("famille %08X : libelle incomplet (en=%q fr=%q) — les deux langues sont obligatoires",
				family, lbl.Text["en"], lbl.Text["fr"])
		}
	}
	if len(cat.Grenades) != 4 {
		t.Errorf("%d rang(s) de grenade, attendu 4 (le film n'en porte pas d'autres)", len(cat.Grenades))
	}
	if len(cat.Abilities) == 0 {
		t.Error("aucune capacite nommee : la table du titre est vide")
	}
}

// TestInstantaneDuCatalogueDuTitreEstFrais — l'instantane que lisent les tests de la brique est
// celui que les fichiers du titre donnent aujourd'hui.
func TestInstantaneDuCatalogueDuTitreEstFrais(t *testing.T) {
	obtenu, err := json.MarshalIndent(catalogueDuTitre(t), "", "  ")
	if err != nil {
		t.Fatalf("serialisation du catalogue : %v", err)
	}
	obtenu = append(obtenu, '\n')
	chemin := filepath.Join(racineDuDepot(t), cheminInstantaneCatalogue)
	if *majInstantaneCatalogue {
		if os.Getenv(majInstantaneCatalogueEnv) == "" {
			t.Skip("regeneration de l'instantane : " + majInstantaneCatalogueEnv + " non defini")
		}
		if err := os.WriteFile(chemin, obtenu, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", chemin, err)
		}
		t.Fatalf("instantane reecrit : %s ; relancer sans -update-catalogue-titre pour verifier", chemin)
	}
	attendu, err := os.ReadFile(chemin) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("instantane absent (%s) : %v — le regenerer (cf. en-tete)", chemin, err)
	}
	if !bytes.Equal(attendu, obtenu) {
		t.Errorf("l'instantane du catalogue du titre n'est plus celui des fichiers du titre (%d octets lus, "+
			"%d attendus) — regenerer (cf. en-tete) puis les goldens d'assemblage de la brique ; premier "+
			"ecart : %s", len(attendu), len(obtenu), premierEcartTexte(string(attendu), string(obtenu)))
	}
	// LA RELECTURE EST FIDELE : les tests de la brique desserialisent l'instantane tel quel. Un
	// catalogue qui ne se relit pas a l'identique (champ perdu par `omitempty`, nul devenu vide)
	// ferait deriver leurs goldens sans que l'octet de l'instantane bouge.
	var relu replay.LabelCatalog
	if err := json.Unmarshal(attendu, &relu); err != nil {
		t.Fatalf("l'instantane ne se relit pas : %v", err)
	}
	if !reflect.DeepEqual(relu, catalogueDuTitre(t)) {
		t.Error("l'instantane relu differe du catalogue construit (nul/vide ou champ perdu a la serialisation)")
	}
}

// premierEcartTexte nomme la premiere ligne qui differe.
func premierEcartTexte(a, b string) string {
	la, lb := bytes.Split([]byte(a), []byte("\n")), bytes.Split([]byte(b), []byte("\n"))
	for i := range max(len(la), len(lb)) {
		var x, y string
		if i < len(la) {
			x = string(la[i])
		}
		if i < len(lb) {
			y = string(lb[i])
		}
		if x != y {
			return fmt.Sprintf("ligne %d : lu %q, attendu %q", i+1, x, y)
		}
	}
	return "aucun"
}
