package replay

// golden_catalog_test.go — LE CATALOGUE DE LIBELLÉS DU TITRE, LU DEPUIS SES VRAIS FICHIERS.
//
// POURQUOI LE VRAI ET PAS UNE FIXTURE. Depuis le lot 3.2, les noms d'armes, de grenades et
// de capacités ne sont plus dans le code : ils vivent dans
// `config/titles/halo_infinite/mappings/`. Un golden bâti sur une fixture figerait des
// noms inventés pour le test — donc ne verrait PAS une régression de configuration (un
// libellé perdu, une langue vide, un effet de tir renommé). En lisant les vrais fichiers,
// le golden d'assemblage devient aussi le garde-rail de ces trois tables.
//
// AUCUN OCTET DE FILM N'EST LU ICI : ce sont des fichiers de config versionnés.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/games/weapons"
)

// goldenCatalog charge le catalogue du titre de référence depuis le dépôt.
//
// La jointure elle-même (famille -> weapon_key -> nom/effet) N'EST PAS recopiée : on
// appelle `NewLabelCatalog`, exactement comme la production. Une jointure recopiée dans
// un test est une jointure qui dérive.
func goldenCatalog(t *testing.T) LabelCatalog {
	t.Helper()
	dir := filepath.Join(repoRootForTest(t), "config", "titles", "halo_infinite", "mappings")

	names, err := mappings.LoadWeaponNamesFromFile(filepath.Join(dir, "weapon_names.toml"))
	if err != nil {
		t.Fatalf("weapon_names.toml : %v", err)
	}
	labels := goldenReplayLabels(t)

	byKey := map[string]Label{}
	for k, v := range names.Names() {
		byKey[k] = Label{En: v.En, Fr: v.Fr}
	}
	var abilities []AbilityPalette
	for _, p := range labels.AbilityPalettes() {
		ranks := map[int]Label{}
		for rank, v := range p.Ranks {
			ranks[rank] = Label{En: v.En, Fr: v.Fr}
		}
		abilities = append(abilities, AbilityPalette{ID: p.ID, Markers: p.Markers, Ranks: ranks})
	}
	grenades := make([]Label, 0, len(labels.GrenadeRanks()))
	for _, v := range labels.GrenadeRanks() {
		grenades = append(grenades, Label{En: v.En, Fr: v.Fr})
	}
	cat := NewLabelCatalog(
		weapons.FilmshellWeaponKeysByFamily(), byKey, labels.ShotEffects(), grenades, abilities)
	// Les familles d'objets d'équipement se posent APRÈS construction, comme en production
	// (cf. replaylabels.Load) : elles n'entrent dans aucune jointure du catalogue. Les oublier
	// ici ferait publier au golden un document dont TOUTES les poses sont `other`, alors que
	// le manifeste en nomme — c'est-à-dire un document que la production ne sert pas.
	cat.EquipmentFamilies = labels.EquipmentObjects()
	// LE DRAPEAU, pour la MÊME raison et par le MÊME chemin que les familles d'équipement
	// (cf. replaylabels.Load) : sans cette table, la chaîne des socles ne reconnaîtrait pas
	// l'objet d'objectif du titre, et le golden figerait un document que la production ne sert
	// pas. Le filtrage par famille est celui de la couche titre — un seul propriétaire du
	// littéral, `mappings.ObjectiveFamilyFlag`.
	for id, o := range labels.ObjectiveObjects() {
		if o.Family != mappings.ObjectiveFamilyFlag {
			continue
		}
		if cat.ObjectiveObjects == nil {
			cat.ObjectiveObjects = map[uint32]Label{}
		}
		cat.ObjectiveObjects[id] = Label{En: o.Label.En, Fr: o.Label.Fr}
	}
	return cat
}

// goldenReplayLabels lit le manifeste de rejeu du titre de référence. Extrait de
// `goldenCatalog` le 2026-08-18 : le garde-rail du drapeau interroge la table des objets
// d'objectif SANS avoir besoin de tout le catalogue.
func goldenReplayLabels(t *testing.T) *mappings.ReplayLabelSet {
	t.Helper()
	path := filepath.Join(repoRootForTest(t), "config", "titles", "halo_infinite", "mappings",
		"replay_labels.toml")
	labels, err := mappings.LoadReplayLabelsFromFile(path)
	if err != nil {
		t.Fatalf("replay_labels.toml : %v", err)
	}
	return labels
}

// repoRootForTest remonte jusqu'au répertoire qui porte `config/titles` — le test tourne
// depuis le paquet, pas depuis la racine.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine du dépôt introuvable (aucun config/titles au-dessus du paquet)")
	return ""
}

// TestCatalogueDuTitreNommeLesArmesDuFilmDeReference — le golden compte 22 familles
// nommées sur le film de référence ; ce test dit POURQUOI elles le sont, et il échoue si
// la chaîne registre -> weapon_names.toml se rompt.
func TestCatalogueDuTitreNommeLesArmesDuFilmDeReference(t *testing.T) {
	cat := goldenCatalog(t)
	if len(cat.Weapons) == 0 {
		t.Fatal("aucune arme nommée : la jointure famille -> weapon_key -> nom est rompue")
	}
	for family, lbl := range cat.Weapons {
		if lbl.En == "" || lbl.Fr == "" {
			t.Errorf("famille %08X : libellé incomplet (en=%q fr=%q) — les deux langues sont obligatoires",
				family, lbl.En, lbl.Fr)
		}
	}
	if len(cat.Grenades) != 4 {
		t.Errorf("%d rang(s) de grenade, attendu 4 (le film n'en porte pas d'autres)", len(cat.Grenades))
	}
	if len(cat.Abilities) == 0 {
		t.Error("aucune capacité nommée : la table du titre est vide")
	}
}
