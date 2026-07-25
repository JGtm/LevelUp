// Package halo_5 — medal_ghost_ids.go : allowlist EXPLICITE des medal_name_id à
// masquer côté page Médailles (décision produit V72-33, 2026-07-25).
//
// Constat : shared_matches_v2.duckdb (medals_earned) contient 3 medal_name_id gagnés
// en match réel (505244449 : 82 occurrences/4 joueurs ; 883611709 : 11/3 ;
// 3566983914 : 19/4) qui n'ont AUCUNE ligne dans metadata.duckdb (medal_definitions) —
// vérifié par diff exhaustif : les 215 lignes du catalogue seedé par l'API Metadata
// officielle (cmd/h5-metadata-fetch, endpoint /medals) ne les contiennent pas, et ils
// n'apparaissent pas non plus dans medalCategoryTable (qui couvre les 215/215).
// Croisement avec https://www.halopedia.org/List_of_Halo_5:_Guardians_medals (~220
// médailles retail documentées par catégorie) : aucun des trois ID n'a de nom
// résolvable pour y être recherché, mais leur absence totale du catalogue officiel les
// place hors de l'ensemble que le wiki documente comme actif — cohérent avec la
// section séparée du wiki listant des médailles coupées/beta ("unused/beta-only") sans
// identifiant stable public. Sans nom ni icône, analysis.MergeMedalCatalog les fait
// remonter via son fallback générique "#<id>" dans la catégorie "Autres".
//
// Masquage EXPLICITE (liste d'ID) plutôt que filtre générique "nom/icône vide" : ce
// fallback existe précisément pour ne JAMAIS perdre une médaille gagnée dont le
// catalogue n'a pas encore été (re)synchronisé (cf. doc analysis.MergeMedalCatalog) —
// un filtre générique masquerait aussi ce cas légitime. Si l'API officielle venait à
// documenter un jour l'un de ces ID, il suffit de le retirer de cette liste.
package halo_5

// GhostMedalIDs — medal_name_id sans définition catalogue (nom/icône), à exclure de la
// page Médailles. Enregistré au boot via service.RegisterGhostMedalIDs.
var GhostMedalIDs = map[int64]bool{
	505244449:  true,
	883611709:  true,
	3566983914: true,
}
