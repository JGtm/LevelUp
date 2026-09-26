// Package types PORTE LES TYPES DE DONNEES QUI TRAVERSENT UNE FRONTIERE DE COUCHE DU DECODEUR
// DE FILM (item 2.6.2 du PLAN_DECODEUR_FILM_2026-09-13, decision V15 (13)).
//
// # POURQUOI CE PAQUET EXISTE
//
// Mesure du 2026-09-17 (note de preparation §3.2) : 81 types exportes par les quatre paquets du
// decodeur traversent une frontiere de couche, soit 40 % de ce qu ils exportent. Tant qu un type
// de DONNEES est declare dans le paquet qui le produit, le consommateur doit importer le
// producteur pour le nommer — et un import « pour un type » est exactement ce qui fait remonter
// des dependances dans le sens interdit (ADR 0034 D-1 : source -> profile -> grammar -> facts ->
// replay). Les types de contrat vivent donc ICI, ou toutes les couches peuvent les nommer sans
// se nommer entre elles.
//
// # C EST UNE FEUILLE, ET C EST LA PROPRIETE QUI COMPTE
//
// AUCUN import du depot, jamais — meme doctrine que `games/canonical`, `domain/replaydoc` et
// `film/source`. Garde-rail : `archlint/film_types_leaf_test.go`. C est cette propriete qui
// autorise `film/source`, elle-meme feuille pour cause de cycle (`filmsource_leaf_test.go`), a
// l importer : une feuille importee par une feuille ne ferme aucun cycle.
//
// # CE QUI Y ENTRE, ET CE QUI N Y ENTRE PAS
//
//	Y ENTRE      un type de DONNEES PUR : des champs, pas de methode, aucune dependance vers un
//	             autre paquet du depot.
//	N Y ENTRE PAS un objet de SERVICE (`Film`, `Source`, `MemoryChunks`, `FilmContext`,
//	             `Profile`, `ScanFilmOptions`), un type qui porte une REGLE (une methode : la
//	             regle appartient a la couche qui la mesure), un type dont un champ nomme un
//	             autre paquet (`killsource.Result` porte `grammar.ProfilDeBalayage`), et un type
//	             dont un champ NON EXPORTE est pose par son producteur (`killsource.Kill` porte
//	             son identite de paquet : la deplacer la rendrait impossible a ecrire).
//
// # LA MESURE DU VOLET GRAMMAIRE (2026-09-17, lot 2.6.2), ET CE QU ELLE A ECARTE
//
// Le plan annoncait 49 types pour `grammar -> replay`, chiffre du 2026-09-13. La RE-MESURE, par
// intersection « types exportes par la racine de `grammar` x identifiants employes par `replay`,
// production seule », en rend **48** : six des 53 comptes en 2026-09 ont descendu dans la couche
// `profile` au lot 2.5.b (`I0Layout`, `MPPWidths`, `MapQuantCatalog`, `MapQuantEntry`, `Profile`,
// `Vec3Range`), et `ProfilDeBalayage` y est apparu.
//
// **TRENTE-ET-UN des 48 y sont entres**, et DIX-SEPT sont ecartes — la mesure dit pourquoi :
//
//	OBJET DE SERVICE, 2   `FilmContext` (porte un memo et un profil), `ScanFilmOptions` (option
//	                      d appel) — tous deux appartiennent a `grammar`.
//	PORTE UNE REGLE, 12   `BipedAim`, `BipedCreation`, `BipedPosition`, `CarrierMarkScan`,
//	                      `FireEvent`, `GrenadeThrow`, `NavpointRadialScan`, `NavpointSegment`,
//	                      `ProfilDeBalayage`, `TeamScanReport`, `WorldObjectKeyframes`,
//	                      `ZoomEvent` — une methode qui decode un champ brut, qui nomme un
//	                      symbole de `grammar` ou de `profile`, ou qui tranche une question de
//	                      grammaire.
//	CHAMP D UNE AUTRE     `EquipmentPlacementStats` (porte `grammar.MPPCalibration`, qui a une
//	COUCHE, 3             methode et des champs `profile.*`), `ManagedPropertyRead` (porte
//	                      `grammar.ManagedPropertyField`, qui a une methode),
//	                      `ObjectDeathStats` (porte `grammar.FrameConfig`, un objet de service
//	                      avec un observateur).
//
// QUATRE TYPES SONT VENUS PAR CLOTURE — 31 + 4 = **35** —, sans etre consommes directement par
// `replay` : ils sont le TYPE D UN CHAMP d un type entrant, et ils sont eux-memes purs —
// `EquipmentChangeKind` (et ses trois constantes), `DeadState`, `PlayerSlotShorts`,
// `InventoryDeltaAmmo`. Les laisser derriere aurait interdit l entree de `EquipmentChange`,
// `ObjectDeath`, `PlayerSlot` et `InventoryDelta` : la cloture des champs fait partie de la
// mesure, pas du confort.
//
// UNE CONSTANTE EST VENUE PAR LA MEME CLOTURE : `MPPFieldCount`, la LONGUEUR des deux tableaux
// de `EquipmentCreation`. Une dimension de tableau fait partie de la forme du type, donc du
// contrat ; l enumeration `grammar.MPPField` — qui porte une methode — la REPREND
// (`MPPFieldCount = types.MPPFieldCount`), de sorte qu il n y a qu une source et pas deux
// litteraux.
//
// # LES RENVOIS DE GODOC POINTENT VERS LA COUCHE QUI PRODUIT
//
// Les commentaires de ce paquet sont ceux des declarations d origine, deplaces SANS reecriture :
// un `[decodeStatComponent]` ou un `[Load]` y designe un symbole de la couche productrice. Les
// reecrire aurait fait passer un deplacement pur pour un changement.
//
// # LA FORME EST FIGEE, ET UNE MUTATION SANS MONTEE DE REVISION ROUGIT
//
// `testdata/shapes.golden` porte, pour chaque type, ses champs, leurs types Go et leurs tags
// JSON, a cote des revisions des couches qui les produisent (`source.Rev`, `killsource.Rev`,
// `objectives.Rev`). Voir
// `shapes_test.go` : un golden par type ferait autant de fichiers que de questions ; le document
// de rejeu, lui, en a UN seul pour tout un arbre.
package types
