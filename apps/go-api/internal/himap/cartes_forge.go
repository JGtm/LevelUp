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
	CanevasSpace    = "fo03_space"
	CanevasDesert   = "fo05_desert"
	CanevasDeepsea  = "fo06_deepsea"
	CanevasWetland  = "fo08_wetland"
	CanevasAcademy  = "fo09_academy"
	CanevasBlank    = "fo11_blank"
	CanevasFrost    = "fo13_frost"
	CanevasDeadland = "fo10_deadland"
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
	// LES 27 CARTES DU RELIQUAT, DECLAREES LE 2026-08-27 apres leur entree au CATALOGUE
	// D OBJECTIFS. Elles etaient jouees par les joueurs suivis sans avoir de fond : il leur
	// manquait la troisieme condition de cuisson — le .mvar et le canevas ne suffisent pas,
	// le cadre se construit sur les ancres d objectif. cmd/mapobj-build les a ingerees hors
	// ligne depuis leur .mvar (aucun appel reseau). Cole Protocol reste dehors : son .mvar
	// ne porte AUCUN objectif, il n y a donc pas de cadre a construire.
	{
		MapID:         "d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b",
		Nom:           "Insolence",
		FichierMvar:   "insolence_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "7097bc4f-efcf-4c5a-a96e-4ddb03e84d2a",
		Nom:           "Flood Gulch",
		FichierMvar:   "flood_gulch_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "944396dd-5661-4a16-b1d8-a6053f762c55",
		Nom:           "944396dd-5661-4a16-b1d8-a6053f762c55",
		FichierMvar:   "944396dd-5661-4a16-b1d8-a6053f762c55_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "8816f240-9038-404b-bbd5-ef4f2b00f482",
		Nom:           "Ecotone",
		FichierMvar:   "ecotone_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "ee43d273-8677-45c2-a8cd-aedd2c463dc9",
		Nom:           "Solution",
		FichierMvar:   "solution_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "ddbb3a00-b109-4703-af07-00433512af38",
		Nom:           "Threshold",
		FichierMvar:   "threshold_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "305b1bdd-9a7b-4975-bacf-8bd63c8c13d2",
		Nom:           "Fortitude Heavies",
		FichierMvar:   "fortitude_heavies_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "37bc3df6-93e8-4d74-b16e-5ceaa30ebc23",
		Nom:           "Thunderhead Heavies",
		FichierMvar:   "thunderhead_heavies_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "28a3ac28-f69d-4fa9-9ebf-a0449c89c8da",
		Nom:           "Thunderhead",
		FichierMvar:   "thunderhead_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "88d45250-97dd-4a28-8fb2-b52baaeebb39",
		Nom:           "Pharaoh",
		FichierMvar:   "pharaoh_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "e3681516-2930-491c-b94f-7dbfa161e000",
		Nom:           "Obituary Heavies",
		FichierMvar:   "obituary_heavies_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "7dfec55d-d968-4a38-aa68-bc31e8874cfd",
		Nom:           "Merchant's Square",
		FichierMvar:   "merchant's_square_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "0cc728d2-9b4d-4b80-95c9-18c77c095575",
		Nom:           "Credence",
		FichierMvar:   "credence_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "e8268e75-6583-42ad-9e0c-2d2f043f5f0f",
		Nom:           "Vallaheim Firefight",
		FichierMvar:   "vallaheim_firefight_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "be848f91-3d87-4b80-8eb9-df3b52cb8d10",
		Nom:           "Urban Raid",
		FichierMvar:   "urban_raid_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "525451ca-0bfa-4b5c-8a0f-29524e0f2834",
		Nom:           "Disciple",
		FichierMvar:   "disciple_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "f459867d-7457-4397-a332-dbbb6812792a",
		Nom:           "Ronin",
		FichierMvar:   "ronin_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "46a8319c-2c63-46ee-9382-788906dcb049",
		Nom:           "Origin - Ranked",
		FichierMvar:   "origin_-_ranked_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "6dbd1c0d-a6c2-4697-8453-f0799d941741",
		Nom:           "Nadair",
		FichierMvar:   "nadair_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "ea7b30e6-d1b9-4c2c-a9e0-de955c48525d",
		Nom:           "Outlook",
		FichierMvar:   "outlook_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "c10c7e79-ec91-4f07-92e8-e570ed9ce79c",
		Nom:           "Refuge Heavies",
		FichierMvar:   "refuge_heavies_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48",
		Nom:           "Lattice - Ranked",
		FichierMvar:   "lattice_-_ranked_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "133c0185-24ed-4bc2-b834-62db5c936257",
		Nom:           "Rat's Nest",
		FichierMvar:   "rat's_nest_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "2a339c65-5128-4457-88d4-0906e265034e",
		Nom:           "Insolence Heavies",
		FichierMvar:   "insolence_heavies_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "79042fc0-ba3d-4046-aa97-5a6902462531",
		Nom:           "Scarlett's Landing",
		FichierMvar:   "scarlett's_landing_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "5b12d6d9-1b5c-4eac-b2a9-de4c36b8a6d3",
		Nom:           "Warehouse",
		FichierMvar:   "warehouse_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "89dd4003-455c-4a1c-bcea-43acd514b20d",
		Nom:           "Dawnbreaker",
		FichierMvar:   "dawnbreaker_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	// Solitude - Ranked, declaree le 2026-08-26 : seule carte du reliquat qui avait deja ses
	// ancres au catalogue.
	{
		MapID:         "4a5e5612-2b2e-4375-a0b3-9335a68815f3",
		Nom:           "Solitude - Ranked",
		FichierMvar:   "solitude_-_ranked_map.mvar",
		ModuleCanevas: CanevasBlank,
	},

	// TROIS CARTES DU LOT DU 28/08 NE SONT PAS DECLAREES ICI, ET C EST VOULU : Munera Platform W4,
	// Munera Platform H6 et Out With A Bang ne portent AUCUN objectif dans leur variante. Or le
	// cadre d un fond se batit sur les ancres d objectifs — sans elles il n y a rien a cadrer. Les
	// declarer ici reviendrait a promettre un asset qu on ne sait pas produire, ce que le garde-rail
	// `TestFondForgeJamaisSousCleModule` refuse a juste titre. Elles sont consignees BLOQUEES au
	// registre des cartes, avec ce motif.
	//
	// LOT DU 2026-08-28 — LES VINGT-CINQ CARTES LES PLUS JOUEES QUI N ETAIENT PAS AU CATALOGUE.
	//
	// Elles ne viennent pas du corpus de l utilisateur mais du CLASSEMENT PUBLIC du jeu : le
	// navigateur de Halo Waypoint se laisse interroger sans jeton, trie par parties recentes et
	// pagine. Sur les 200 cartes les plus jouees, 109 manquaient — Vacancy, que l utilisateur
	// reclamait, n en etait que la partie visible.
	//
	// CE QUI EST ECARTE, ET POURQUOI : les variantes « - Ranked » (meme dessin que leur base
	// — sauf Vacancy et Showdown Arena, dont la BASE manquait aussi, d ou leur presence ici),
	// les variantes Firefight (meme geometrie, autre mode), les republications UGC de cartes
	// natives que nous avons deja (Bazaar, Fragmentation, Oasis, Breaker — 0 a 1 000 objets),
	// et les cartes d entrainement communautaires (verdict utilisateur du 2026-08-28).
	//
	// Le canevas de chacune est lu dans la LISTE DE FICHIERS de son asset : une carte Forge
	// publie son canevas a cote de sa variante (`fo09_academy.mvar`). Pas de deduction par le
	// nom, pas de table a maintenir.
	{
		MapID:         "4fb5b69f-5104-450b-9ed0-a232f997e8f9",
		Nom:           "Vacancy",
		FichierMvar:   "4fb5b69f-5104-450b-9ed0-a232f997e8f9_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "1042b738-30d6-453b-a057-cec8bda009b3",
		Nom:           "Showdown Arena",
		FichierMvar:   "1042b738-30d6-453b-a057-cec8bda009b3_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "654dff62-d618-496a-8914-06ab73d991e3",
		Nom:           "Interference",
		FichierMvar:   "654dff62-d618-496a-8914-06ab73d991e3_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "8cf45707-ced1-4722-b736-c139cbf0d1e2",
		Nom:           "Ardent Prayer",
		FichierMvar:   "8cf45707-ced1-4722-b736-c139cbf0d1e2_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "841242db-013f-4b10-871e-c432379e852c",
		Nom:           "Courtyard",
		FichierMvar:   "841242db-013f-4b10-871e-c432379e852c_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "50a1a3b5-c54d-4e40-8744-b92051d48897",
		Nom:           "Diminished",
		FichierMvar:   "50a1a3b5-c54d-4e40-8744-b92051d48897_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "0c299a3a-260e-4f74-8111-d2c048d0b033",
		Nom:           "Megapolis",
		FichierMvar:   "0c299a3a-260e-4f74-8111-d2c048d0b033_map.mvar",
		ModuleCanevas: CanevasSpace,
	},
	{
		MapID:         "37a9b5f0-6be7-4a46-8010-1fe6f7ea5611",
		Nom:           "Ruujaya",
		FichierMvar:   "37a9b5f0-6be7-4a46-8010-1fe6f7ea5611_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "b6c7bdfa-0021-4ef7-b99f-332084f1060c",
		Nom:           "Yuletide",
		FichierMvar:   "b6c7bdfa-0021-4ef7-b99f-332084f1060c_map.mvar",
		ModuleCanevas: CanevasFrost,
	},
	{
		MapID:         "66f4fe86-ba15-43d2-9270-bbd67dd22d8f",
		Nom:           "Foundry",
		FichierMvar:   "66f4fe86-ba15-43d2-9270-bbd67dd22d8f_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "1441775d-a1f1-4571-aa7f-1d1ebf6ae15c",
		Nom:           "Guardian",
		FichierMvar:   "1441775d-a1f1-4571-aa7f-1d1ebf6ae15c_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "b4d13418-c0b5-47dc-9515-931dfda77d9f",
		Nom:           "Serenity",
		FichierMvar:   "b4d13418-c0b5-47dc-9515-931dfda77d9f_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "e6d73380-fbe2-467d-a63f-427939411d13",
		Nom:           "Powerhouse",
		FichierMvar:   "e6d73380-fbe2-467d-a63f-427939411d13_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "beedcb81-a43a-4810-b43c-a6e4949b64a6",
		Nom:           "Canopy",
		FichierMvar:   "beedcb81-a43a-4810-b43c-a6e4949b64a6_map.mvar",
		ModuleCanevas: CanevasDesert,
	},
	{
		MapID:         "2143a29c-19ec-4671-8992-f15cbfeeab21",
		Nom:           "Ivory Tower",
		FichierMvar:   "2143a29c-19ec-4671-8992-f15cbfeeab21_map.mvar",
		ModuleCanevas: CanevasDeadland,
	},
	{
		MapID:         "fdde5715-cd2b-4f78-ad7d-9a83a4f968c6",
		Nom:           "Lone Wolf",
		FichierMvar:   "fdde5715-cd2b-4f78-ad7d-9a83a4f968c6_map.mvar",
		ModuleCanevas: CanevasAcademy,
	},
	{
		MapID:         "252e2a45-93fa-4661-a5cf-4a35803f9149",
		Nom:           "Boulevard",
		FichierMvar:   "252e2a45-93fa-4661-a5cf-4a35803f9149_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "6c433ed1-d56b-4ea1-9520-423ddf8e0f11",
		Nom:           "Alpha Site",
		FichierMvar:   "6c433ed1-d56b-4ea1-9520-423ddf8e0f11_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "3922c263-396e-4686-8097-a6b43c3e11b7",
		Nom:           "Security Zone",
		FichierMvar:   "3922c263-396e-4686-8097-a6b43c3e11b7_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "71c3a721-a4db-4866-a831-19991594f384",
		Nom:           "Ghost Town",
		FichierMvar:   "71c3a721-a4db-4866-a831-19991594f384_map.mvar",
		ModuleCanevas: CanevasWetland,
	},
	{
		MapID:         "78cebfc7-980a-4e8f-86a7-cf22f55f670e",
		Nom:           "Cold Storage",
		FichierMvar:   "78cebfc7-980a-4e8f-86a7-cf22f55f670e_map.mvar",
		ModuleCanevas: CanevasBlank,
	},
	{
		MapID:         "47823612-9de0-4ca9-8a95-b3a6ebd7ca91",
		Nom:           "Immolate",
		FichierMvar:   "47823612-9de0-4ca9-8a95-b3a6ebd7ca91_map.mvar",
		ModuleCanevas: CanevasBlank,
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

// DepotNavmesh : ou sont les navmesh.blob publies avec les cartes Forge, RELATIVEMENT A LA
// RACINE DU DEPOT. Meme statut que le depot de variantes : entree d outillage hors ligne, non
// versionnee. Un blob absent fait retomber la carte sur la cuisson ordinaire, jamais echouer.
const DepotNavmesh = ".ai/re_dump/navmesh"

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
