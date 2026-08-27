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

	// LES NEUF FILMS ASSAUT DU LOT A, releves le 2026-08-27 sur `match_registry` (lecture
	// seule via `cmd/diag_q`, depot principal) — meme convention que les sept films Oddball
	// ci-dessus : TOUS les films du corpus entrent ici, y compris ceux que les catalogues ne
	// couvrent pas, pour que l'exclusion soit MESUREE par l'instrument et jamais presumee.
	"35b75a31": {"Origin", "b302eb62-da9a-480b-a409-3c89df8c1a04"},
	"ce083875": {"Origin", "b302eb62-da9a-480b-a409-3c89df8c1a04"},
	"69b16f5d": {"Origin", "b302eb62-da9a-480b-a409-3c89df8c1a04"},
	"3d58eb37": {"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352"},
	"34bb3bc8": {"Rat's Nest", "133c0185-24ed-4bc2-b834-62db5c936257"},
	"df8fcbef": {"Curfew", "63d634be-0319-489d-8c21-9c4e012f664f"},
	"c75f33b8": {"Curfew", "63d634be-0319-489d-8c21-9c4e012f664f"},
	"9f57c612": {"Curfew", "63d634be-0319-489d-8c21-9c4e012f664f"},
	"1c01e34f": {"Urban Raid", "be848f91-3d87-4b80-8eb9-df3b52cb8d10"},
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
func attBornes(t *testing.T, root, id string) (filmdec.Vec3Range, filmdec.I0Layout, bool) {
	t.Helper()
	c, ok := attCartes[id]
	if !ok {
		t.Logf("%s : carte inconnue du fixture — bornes indisponibles", id)
		return filmdec.Vec3Range{}, filmdec.I0Layout{}, false
	}
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(attRefDir(root), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(c.Nom)
	if err != nil {
		t.Logf("%s : carte %q absente du catalogue de bornes (%v)", id, c.Nom, err)
		return filmdec.Vec3Range{}, filmdec.I0Layout{}, false
	}
	// e.Layout() porte largeurs d'axe, largeur d'index de région et région attendue —
	// trois constantes par carte du MÊME catalogue (Live Fire : région 1 sur 2 bits, lot C
	// catalogues 2026-08-27). Il est rendu à l'appelant pour le chemin BIPÈDE, et installé
	// ici pour le chemin WORLD-OBJECT, comme avant.
	filmdec.SetWorldObjectPrecisionFromLayout(e.Layout())
	return e.Range(), e.Layout(), true
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
