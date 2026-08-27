package replay

// attachement_phase0_cartes_test.go — LES RÉFÉRENCES DE CARTE DE LA PHASE 0.
//
// POURQUOI CE FICHIER EXISTE À PART. Les deux items de mesure ont besoin de la même chose et
// pour la même raison : sans les BORNES de quantification de la carte, un objet du monde ne
// rend que des quanta, et « à moins de 1,5 m » n'a aucun sens. Ces bornes sont des DONNÉES DE
// RÉFÉRENCE VERSIONNÉES (`map_quant_bounds.json`), pas une lecture de base.
//
// LE NOM ET L'IDENTIFIANT DE LA CARTE, EUX, NE SONT PAS DANS LE FILM — c'est établi et
// documenté (`map_objectives.go` : « l'artefact est décodé des seuls chunks du film, qui ne
// nomment ni la carte ni le mode »). Ils sont donc GELÉS ici comme fixture, relevés une fois
// pour toutes dans l'instantané parquet `match_registry_20260711_090652.parquet` (lecture
// seule, aucune base ouverte) — même convention que les lignes de match d'`objCorpus`.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// attRefEnv permet de désigner le répertoire des données de référence du titre. Sans elle, il
// est déduit de la racine du cache film : `<dépôt>/data/cache` -> `<dépôt>/data/titles/...`.
const attRefEnv = "ATT_REF"

// attCartes — nom de carte et identifiant d'asset, par film. Relevé le 2026-08-18 sur
// `match_registry_20260711_090652.parquet`.
var attCartes = map[string]struct{ Nom, MapID string }{
	"64e8adfa": {"Catalyst", "f7e8cde9-0c0a-487c-94a3-61bfa0f20465"},
	"530820e5": {"Catalyst", "f7e8cde9-0c0a-487c-94a3-61bfa0f20465"},
	"53ce4390": {"Behemoth", "e9a5a982-6c4e-4db6-9383-7b03671460eb"},
	"084a804d": {"Fortitude Heavies", "305b1bdd-9a7b-4975-bacf-8bd63c8c13d2"},
	"a349fea8": {"Fragmentation Heavies", "0d849a52-fedb-4aea-b5a3-caee268f1f49"},

	// LES SEPT FILMS ODDBALL DU RECENSEMENT D1, relevés le 2026-08-27 par
	// `cmd/zone-attribution -census` (lecture seule de `match_registry`, sortie figée dans
	// `registre_film/D1_recensement_modes.log`) — et non plus sur l'instantané parquet, qui
	// date de juillet et ne connaît pas ces matchs.
	//
	// LES SEPT SONT LÀ, Y COMPRIS LES TROIS QUE LES CATALOGUES NE COUVRENT PAS. Live Fire n'a
	// pas de bornes de quantification, Lattice n'est pas au catalogue d'objectifs : ces trois
	// films sortiront par le chemin NON EXPLOITABLE de l'instrument, qui NOMME la cause. Les
	// omettre ici ferait disparaître l'exclusion au lieu de la mesurer.
	"24dbb67d": {"Recharge - Ranked", "336b5174-3579-4fd8-b2f0-922e4a5f7628"},
	"43716616": {"Smallhalla", "98783453-ce40-4020-9e87-62099a290b62"},
	"51ebbc0f": {"Banished Narrows", "9ad226d8-8947-4c5b-95bc-d220187698c1"},
	"60ae07c4": {"Live Fire - Ranked", "309253f8-7a75-48ff-83e1-e7fb3db2ac47"},
	"92f18088": {"Lattice - Ranked", "1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48"},
	"c88ec007": {"Live Fire", "6c01f693-c968-4a71-b157-efc35ffcf71f"},
	"d9781168": {"Dredge", "e4bb06db-065f-4902-b93b-d8dac315eac4"},
}

// attRefDir rend le répertoire des données de référence du titre.
func attRefDir(root string) string {
	if v := os.Getenv(attRefEnv); v != "" {
		return v
	}
	return filepath.Join(root, "..", "titles", "halo_infinite", "reference")
}

// attBornes rend les bornes monde de la carte d'un film ET installe les largeurs d'axe de
// cette carte pour le chemin objet du monde.
//
// L'APPELANT DOIT DÉTENIR `LockProcessDecode` ET RESTAURER `WorldObjectPrecision` : c'est un
// global de paquet, et le correctif du 2026-08-15 a mesuré ce que coûte de l'oublier (tous
// les objets déquantifiés aux largeurs de la carte précédente).
func attBornes(t *testing.T, root, id string) (filmdec.Vec3Range, bool) {
	t.Helper()
	c, ok := attCartes[id]
	if !ok {
		t.Logf("%s : carte inconnue du fixture — bornes indisponibles", id)
		return filmdec.Vec3Range{}, false
	}
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(attRefDir(root), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(c.Nom)
	if err != nil {
		t.Logf("%s : carte %q absente du catalogue de bornes (%v)", id, c.Nom, err)
		return filmdec.Vec3Range{}, false
	}
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: [3]uint{
		uint(e.AxisWidths[0]), uint(e.AxisWidths[1]), uint(e.AxisWidths[2])}})
	return e.Range(), true
}

// attMarqueurs rend les objectifs PONCTUELS d'un rôle donné sur la carte d'un film. Rend une
// liste vide (et le dit) quand la carte n'est pas au catalogue : c'est le cas NOMINAL, le
// catalogue ne couvre pas toutes les cartes jouées.
func attMarqueurs(t *testing.T, root, id, role string) []PointObjective {
	t.Helper()
	c, ok := attCartes[id]
	if !ok {
		return nil
	}
	cat, err := LoadMapObjectives(filepath.Join(attRefDir(root), "map_objectives.json"))
	if err != nil {
		t.Fatalf("catalogue d'objectifs : %v", err)
	}
	e, err := cat.Lookup(c.MapID)
	if err != nil {
		t.Logf("%s : carte %s absente du catalogue d'objectifs (%v)", id, c.MapID, err)
		return nil
	}
	return e.PointsOfRole(mapvar.Role(role))
}
