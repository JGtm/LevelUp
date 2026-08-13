// Package himap — cartes_forge.go : LE CATALOGUE DES CARTES FORGE DECLAREES.
//
// Ce fichier est de la DONNEE : une declaration par carte Forge dont le fond est produit.
// La chaine de cuisson vit dans cuisson_forge.go ; les garde-rails de la declaration dans
// cle_forge_test.go (cle map_id unique, `.mvar` de carte et non de rack, fond publie).
package himap

import "path/filepath"

// CarteForge declare une carte Forge a cuire. Le catalogue d'objectifs ne suffit pas a la
// designer : l'entree de Vagabond y porte le module GENERIQUE `map` (comme Highpower), donc la
// selection par nom de module est ambigue. On la designe par son `map_id` (asset UGC) et on
// VERIFIE la jointure par le compte d'objets du catalogue.
type CarteForge struct {
	// MapID est l'asset UGC : cle du catalogue d'objectifs ET cle de publication du fond
	// (`{map_id}.png/json`). Un canevas est partage par des dizaines de cartes Forge : une
	// cle par module y ferait collision (documente ici meme jusqu'au 2026-08-13) — le
	// map_id, present sur chaque match de `match_registry`, est unique par construction.
	MapID string
	// Nom est le nom affiche de la carte (match_registry.map_name) — lisibilite des
	// rapports et des logs, JAMAIS une cle.
	Nom string
	// FichierMvar est le nom du `.mvar` dans le depot de variantes de carte.
	FichierMvar string
	// ModuleCanevas est le dossier du module sur lequel la carte est batie.
	ModuleCanevas string
}

// Les canevas Forge installes : les 8 dossiers de module sur lesquels les cartes
// communautaires sont baties. Constantes nommees — le meme litteral en 20+ declarations
// re-divergerait (regle des <= 2 copies).
const (
	CanevasSpace   = "fo03_space"
	CanevasDesert  = "fo05_desert"
	CanevasDeepsea = "fo06_deepsea"
	CanevasWetland = "fo08_wetland"
	CanevasAcademy = "fo09_academy"
	CanevasBlank   = "fo11_blank"
	CanevasFrost   = "fo13_frost"
)

// CartesForge : les cartes Forge dont l'asset est produit.
//
// COMMENT ON SAIT QU'UNE CARTE EST FORGE, et ce n'est pas un prefixe de nom : son module de
// canevas ne porte AUCUNE instance de geometrie. La chaine native le dit d'elle-meme —
// « aucune instance dessinee sur 0 du bsp retenu » — c'est ainsi que Corpo a ete identifiee, en
// echouant. Un canevas vierge (`fo11_blank`) est vide par construction : la carte EST son rack
// d'objets.
var CartesForge = []CarteForge{
	{
		MapID:         "105f5d84-8de1-4908-af3a-1c4f3bf9d642",
		Nom:           "Vagabond",
		FichierMvar:   "vagabond_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "8be179f7-8940-4868-b881-44cad1ca8711",
		Nom:           "Corpo",
		FichierMvar:   "corpo_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	// Pilotes du lot fonds par map_id (2026-08-13) : seules cartes jouees SEULES sur leur
	// canevas. Preuve level_id : Starboard -747133697 (0xD377A4FF) -> fo03_space, Dredge
	// 2123870979 (0x7E97B303) -> fo06_deepsea, unicite 1/1 (`TestPreuveLevelIDCartes`).
	{
		MapID:         "7a9265af-a880-487b-8829-68d88fcfb145",
		Nom:           "Starboard",
		FichierMvar:   "starboard_map.mvar",
		ModuleCanevas: CanevasSpace,
	},
	{
		MapID:         "e4bb06db-065f-4902-b93b-d8dac315eac4",
		Nom:           "Dredge",
		FichierMvar:   "dredge_map.mvar",
		ModuleCanevas: CanevasDeepsea,
	},
	// Masse du lot fonds par map_id (2026-08-13), ordre du plan (nombre de matchs).
	// Preuve carte -> canevas : level_id, unicite 1/1 (`TestPreuveLevelIDCartes`, un
	// level_id par canevas — table commentee dans cmd/mapquant-build). Le map_id est
	// l'asset majoritaire de `match_registry` pour ce nom de carte.
	{
		MapID:         "648ae7aa-c5d0-4f80-861a-79eb30440fcb",
		Nom:           "The Pit",
		FichierMvar:   "the_pit_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "410f1c01-aca6-4567-9df5-9b16bd550cb2",
		Nom:           "Snowbound",
		FichierMvar:   "snowbound_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "d035fc3e-f298-4c14-9487-465be2e1dc1f",
		Nom:           "Empyrean",
		FichierMvar:   "empyrean_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "b302eb62-da9a-480b-a409-3c89df8c1a04",
		Nom:           "Origin",
		FichierMvar:   "origin_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "78da545f-a168-4a5e-9c8d-dd379067c352",
		Nom:           "Absolution",
		FichierMvar:   "absolution_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "63d634be-0319-489d-8c21-9c4e012f664f",
		Nom:           "Curfew",
		FichierMvar:   "curfew_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "cfd90b63-62fd-441a-8015-8d7804b9c3c3",
		Nom:           "Dynasty",
		FichierMvar:   "dynasty_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "2890782c-0a33-4f2c-a468-e3a7d6cd6db4",
		Nom:           "Shiro",
		FichierMvar:   "shiro_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "4bffd021-92c0-422b-8b6e-8f595511458c",
		Nom:           "Cliffside",
		FichierMvar:   "cliffside_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "2be34415-bc96-4d02-875c-c4f2aa135f89",
		Nom:           "Nemesis",
		FichierMvar:   "nemesis_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "921aebb1-783d-45e4-bacd-7ad869fa8dae",
		Nom:           "Domicile",
		FichierMvar:   "domicile_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "0d1c9255-d912-416c-befc-5f3e5e176df2",
		Nom:           "Fortress",
		FichierMvar:   "fortress_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "504ebf22-12b6-46c3-a9c1-ea20ca5bf03c",
		Nom:           "Goliath",
		FichierMvar:   "goliath_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "01af558d-53ab-4f05-ba68-92d805fc6260",
		Nom:           "Isolation",
		FichierMvar:   "isolation_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42",
		Nom:           "Solitude",
		FichierMvar:   "solitude_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "cf034ec8-ee47-43c2-b2e8-4751c22b3d4d",
		Nom:           "Houseki",
		FichierMvar:   "houseki_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "bb7b78ae-3468-46ce-b5ba-cca61c3a338a",
		Nom:           "High Ground",
		FichierMvar:   "high_ground_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "cd08bc7a-7ba5-4502-be87-c58b641fc94d",
		Nom:           "Salvation",
		FichierMvar:   "salvation_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "edcd4467-6846-455f-ac44-f1034476f774",
		Nom:           "Takamanohara",
		FichierMvar:   "takamanohara_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "76043dc6-2724-45e2-9b5a-6fe2e75da588",
		Nom:           "Elevation",
		FichierMvar:   "elevation_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "df7dbf08-b8de-4ade-9d7f-1947128c9ae4",
		Nom:           "Kiken'na",
		FichierMvar:   "kiken'na_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "9ad226d8-8947-4c5b-95bc-d220187698c1",
		Nom:           "Banished Narrows",
		FichierMvar:   "banished_narrows_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "98a83f87-2420-48c6-93f8-cc6b62d73235",
		Nom:           "Kaiketsu",
		FichierMvar:   "kaiketsu_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "a289bafe-102e-4363-98f7-80b596007338",
		Nom:           "Obituary",
		FichierMvar:   "obituary_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "255bbe78-b191-476e-b0ae-0763c3bc2f44",
		Nom:           "Opulence",
		FichierMvar:   "opulence_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "2c9f3490-6be2-4d90-9015-02095651e91e",
		Nom:           "Command",
		FichierMvar:   "command_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "1ede38fa-4d30-4dfa-a8b7-5d08bf4e46e3",
		Nom:           "Fortitude",
		FichierMvar:   "fortitude_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "41217472-3020-4bd8-bce9-b2a2b0d50896",
		Nom:           "Refuge",
		FichierMvar:   "refuge_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "bae4df14-4f4a-424c-aac1-2f795c807146",
		Nom:           "Critical Dewpoint",
		FichierMvar:   "critical_dewpoint_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "c5ac9f12-660e-4f1a-83e7-2e7536bbcb04",
		Nom:           "Perilous",
		FichierMvar:   "perilous_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "33075df7-01c8-40e1-8b3e-1baee0054c76",
		Nom:           "Shogun",
		FichierMvar:   "shogun_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "95b69e4b-485f-4c6c-9b00-4bd68c94c1e9",
		Nom:           "Sylvanus",
		FichierMvar:   "sylvanus_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "98783453-ce40-4020-9e87-62099a290b62",
		Nom:           "Smallhalla",
		FichierMvar:   "smallhalla_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
}

// EstCanevasForge dit si un dossier installe est le CANEVAS d'une carte Forge declaree — donc
// si la chaine NATIVE doit le laisser tranquille : il ne porte aucune geometrie de carte, la
// carte Forge est le rack d'objets de son `.mvar` et son fond se publie sous son map_id.
func EstCanevasForge(module string) bool {
	for _, c := range CartesForge {
		if c.ModuleCanevas == module {
			return true
		}
	}
	return false
}

// DepotVariantesCarte : ou sont les `.mvar`, RELATIVEMENT A LA RACINE DU DEPOT.
//
// Ce dossier n'est pas versionne (`.gitignore`), au meme titre que l'installation du jeu :
// c'est une entree d'outillage hors ligne, pas une donnee de reference. Sa constitution est
// decrite dans `.ai/V7.5/` — un `.mvar` absent fait echouer la cuisson de sa carte, jamais des
// autres.
const DepotVariantesCarte = ".ai/re_dump/mapvar"

// CheminCanevasForge rend le chemin du `.module` du canevas d'une carte Forge, ou la chaine
// vide s'il n'est pas installe (la cuisson s'en passe alors, cf. OptionsCuissonForge).
func CheminCanevasForge(c CarteForge) string {
	dir, err := LevelsDir("pc")
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, c.ModuleCanevas, c.ModuleCanevas+"-rtx-new.module")
	if !existeFichier(p) {
		return ""
	}
	return p
}
