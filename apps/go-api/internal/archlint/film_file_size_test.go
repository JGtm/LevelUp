package archlint

// film_file_size_test.go — LE RATCHET DE TAILLE DES FICHIERS DU DECODEUR ET DE SES VOISINS
// (lot 2.7, 2026-09-16).
//
// # POURQUOI CE RATCHET EXISTE
//
// CLAUDE.md regle 5 fixe le seuil a 500 lignes par fichier. Il n etait garde par rien, et la
// mesure de la revue de jalon M1 (2026-09-16, constats C1 / C2) a dit ce que ca coute : entre
// `783ae680d` et `34fa53da5`, SIX fichiers deja au-dela du seuil ont GROSSI sans que personne
// ne le voie — `document_chronicle.go` +171, `killcollector/collector.go` +138,
// `replay/document.go` +30, `replay/build.go` +16, `replay/zone_states_hill.go` +13,
// `filmdec/traverse.go` +3. Aucun de ces lots n avait tort ; simplement, rien ne sonnait.
//
// # LA REGLE, EN DEUX LIGNES
//
//	fichier hors table   plafond 500 lignes
//	fichier de la table  plafond = sa taille du jour de sa mise en table, JAMAIS accru
//
// Un fichier qui descend sous son plafond n est pas une erreur — c est le sens de la marche.
// Quand il repasse sous 500, son entree se RETIRE de la table (une entree perimee finit par
// autoriser n importe quoi) ; le test le dit lui-meme.
//
// # CE QUE LE RATCHET NE FAIT PAS, ET C EST VOULU
//
// Il ne demande a personne de scinder les fichiers de la table. Il interdit de les AGRANDIR.
// La scission se decide lot par lot, avec la preuve d equivalence qui va avec (le lot 2.7 l a
// faite pour `grammar` et `killcollector` ; le volet publication viendra apres la fusion des
// lots 1.9.9 / 1.9.11 / 1.9.14, qui tiennent ces fichiers).
//
// Il ne distingue pas non plus un fichier de test d un fichier de production. Une table de
// fixtures de 1 200 lignes est exactement le genre de fichier qui grossit sans qu on s en
// apercoive, et la regle 5 ne l exempte pas.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// seuilLignesParFichier : le seuil du depot (CLAUDE.md regle 5).
const seuilLignesParFichier = 500

// racinesSurveilleesTaille : les arborescences que ce ratchet couvre, relatives a `apps/go-api`.
// Ce sont celles du chantier du decodeur de film — le perimetre du lot 2.7.
//
// LA QUATRIEME RACINE A DISPARU LE 2026-09-16 (lot 2.5.d.2) : `internal/analysis/objectiveevents`
// est descendu sous `internal/games/halo_infinite/film/internal/facts/objectives`, donc SOUS la premiere
// racine. La garder aurait fait compter ses fichiers DEUX FOIS dans le plancher — un plancher
// qu on gonfle est un plancher qui ne mesure plus rien.
var racinesSurveilleesTaille = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/sync/killcollector",
}

// plancherFichiersBalayesTaille : LE PLANCHER CONTRE UN BALAYAGE MUET. 1 298 fichiers `.go`
// mesures le 2026-09-16 dans les racines ci-dessus ; un balayage qui en rend nettement
// moins n a pas trouve l arborescence (racine renommee, chemin relatif casse) et ne garde plus
// rien. Il doit ECHOUER bruyamment, pas rendre vert sur du vide.
const plancherFichiersBalayesTaille = 1200

// plafondsParFichier — LA TABLE DATEE DES FICHIERS DEJA AU-DELA DU SEUIL, avec leur taille du
// 2026-09-16 (lot 2.7, apres la scission de `grammar` et de `killcollector`). Chemin relatif a
// `apps/go-api`, en slash.
//
// CHAQUE VALEUR NE PEUT QUE DESCENDRE. Faire monter une entree, c est autoriser exactement la
// derive que la mesure C1 / C2 a constatee ; la reponse a « mon lot ajoute vingt lignes ici »
// est de sortir vingt lignes ailleurs dans le fichier, ou de le scinder.
//
// UNE SEULE EXCEPTION, ECRITE, ET ELLE COUVRE DEUX FICHIERS : `replay/document_chronicle.go` et
// `replay/structure_test.go`. Le premier est une CHRONIQUE — une entree par version de schema,
// ajoutee en meme temps que la montee de `SchemaVersion`, et le depot a deja tranche qu elle ne
// se scinde pas (item 2.7.1 du PLAN_DECODEUR_FILM : « c est une chronique : exemption ecrite en
// tete, pas de scission »). Le second porte la JUSTIFICATION que
// `TestStructureIsOptionalInDocument` exige avant d accepter une montee : le test refuse
// `SchemaVersion + 1` tant que la raison n est pas ecrite au-dessus de lui, donc la montee de
// schema ajoute mecaniquement un paragraphe ici aussi. Leur plafond monte du volume de l entree
// ajoutee, DANS LE COMMIT QUI MONTE `SchemaVersion`, et JAMAIS autrement.
//
// `structure_test.go` EST ENTRE DANS L EXCEPTION LE 2026-09-17 (lot 2.6.3, schema 60 -> 61,
// 1 151 -> 1 168) : il etait en table depuis la fusion du lot 2.7g sans que l exception le nomme,
// et la premiere montee de schema qui a suivi l a fait rougir pour la raison meme qui la rendait
// obligatoire. Toute autre montee de ces deux lignes est le meme abus que pour les autres.
//
// RE-MESURE A L ENTREE DANS L INTEGRATION (2026-09-17, fusion du lot 2.7g) : la table a ete
// figee sur la base du lot (f950b7179) ; entre cette base et la fusion, l integration a recu la
// vague 2 de la famille 1.9, la revue de jalon M1 et la montee de schema 59 -> 60. Quatre
// fichiers avaient donc grossi AVANT que le ratchet n existe sur l integration :
// `document_chronicle.go` 1476 -> 1541 (l entree de chronique v60, l exception ecrite),
// `statborg.go` 652 -> 687, `golden_assembly_test.go` 1180 -> 1202, `structure_test.go`
// 1135 -> 1151 (goldens et structure du schema 60). Leurs plafonds sont ceux de la fusion : ce
// n est pas une montee, c est la date reelle d entree en table. A partir de ce commit, la regle
// s applique sans exception nouvelle.
var plafondsParFichier = map[string]int{
	// --- publication du rejeu : SCINDES par le volet 2.7p (fusion 2026-09-17 : document 585 -> 444,
	// lives 536 -> 184, build 523 -> 165, zone_states_hill 526 -> 363, score_timeline 524 -> 329,
	// document_vehicles 505 -> 371 — tous sous le seuil, sortis de la table). La chronique porte
	// desormais son exemption ECRITE EN TETE (item 2.7.1, 28 lignes de commentaire) : 1541 -> 1569,
	// la seule montee admise par cette exemption hors montee de SchemaVersion.
	// SCHEMA 61 -> 62 (2026-09-17, lot 4.2.1-b) : 1626 -> 1687, l entree de chronique v62 (`layers`,
	// `coverage.deathsPaths`, le reclassement de `vehicleLabels`, et ce qui n y entre PAS).
	// SCHEMA 62 -> 63 (2026-09-19, post-chantier lot 5.1) : 1687 -> 1758, l entree de chronique v63
	// (`flagCarries[].spans[].returnProgress`, `vehicleCycles`, leurs neuf compteurs de couverture,
	// pourquoi la note 3.7 concluait « non mesurable » et ce qui a change depuis, et ce qui n y
	// entre PAS — la surface web du cycle de vehicule, transmise au lot de rendu).
	// SCHEMA 63 -> 64 (2026-09-20, post-chantier lot 5.2-A) : 1762 -> 1825, l entree de chronique
	// v64 (`zoneStates[].gaugeRamps` et son `capturingTeam`, le seuil d aboutissement mesure, la
	// forme ECARTEE — un champ sur `GaugePoint`, type partage avec la jauge de retour du drapeau —
	// et l archetype `zones` ti=23 qui reste non cable).
	// SCHEMA 64 -> 65 (2026-09-21, post-chantier lot 5.3.6) : 1825 -> 1863, l entree de chronique
	// v65 (`stances[]`, ses trois genres, la marche qui les lit, et les DEUX negatifs mesures qui
	// expliquent pourquoi le sprint et le saut n y sont pas). Exception ecrite, dans le commit qui
	// monte `SchemaVersion`.
	// SCHEMA 65 -> 66 (2026-09-21, post-chantier lots 5.9.4 et 5.9.5) : 1863 -> 1933, l entree de
	// chronique v66. Elle porte DEUX genres neufs de `stances[].kind` et les separe : `sprint`
	// est LU (`i57` porte l index de la fente de capacite active, et l image nomme les trois
	// fentes), `jumpDerived` est CALCULE (l integrale de la vitesse verticale, reconnue a sa
	// hauteur). Elle porte aussi le controle du GRAPPIN qui valide la lecture de l index, et la
	// raison mesuree pour laquelle la vitesse du sprint ne peut pas trancher. UNE SEULE MONTEE
	// DE SCHEMA POUR LE LOT, donc une seule entree, dans le commit qui monte `SchemaVersion`.
	// SCHEMA 66 -> 67 (2026-09-21, post-chantier lot 5.10) : 1933 -> 1974, l entree de chronique
	// v67. Elle porte le changement de SENS de `rides[].src` (deux valeurs, `film` et
	// `proximity`, la ou trois disaient la precision des bornes), la regle de PRIMAUTE de la
	// lecture sur le repli et son prix, les deux compteurs de couverture qui remplacent les
	// trois anciens, et le verdict Theater qui a nomme le defaut (le Razorback `776/1`). UNE
	// SEULE MONTEE DE SCHEMA POUR LE LOT, donc une seule entree, dans le commit qui monte
	// `SchemaVersion`.
	// SCHEMA 67 -> 68 (2026-09-22, post-chantier lot 5.22.4) : 1974 -> 2018, l entree de
	// chronique v68. Elle porte le RENOMMAGE de `stances[].kind` `mobility` en `clamber`, la
	// raison pour laquelle un renommage d enum publie est une montee de FORME (un genre inconnu
	// est IGNORE cote web), l oracle qui a tranche la ou le binaire ne le pouvait pas (neuf
	// verdicts Theater sur neuf, lot 5.13.2 reouvert), et le refus de publier un genre `jump`
	// LU faute de preuve. UNE SEULE MONTEE DE SCHEMA POUR LE LOT, donc une seule entree, dans
	// le commit qui monte `SchemaVersion`.
	// SCHEMA 68 -> 69 (2026-09-24, integration de la vague C des retours du rejeu) : 2018 -> 2142,
	// L ENTREE UNIQUE de chronique v69 qui reunit les trois lots de la vague sous une seule
	// montee (un en-tete commun de 5 lignes, puis une partie par lot). Partie M1 (2026-09-23,
	// +40 dont +4 a la reprise apres revue adverse du 2026-09-24) : la grammaire de la vie et les
	// deux replis nommes de la publication des positions, le champ `vehicles[].samples[].g`, les
	// compteurs de couverture (`slotsArmes`, `slotsDesarmes`, `silencesNonTranches` compris), et
	// pourquoi « si le film le dit, on publie » ne couvrait pas des points que le BALAYAGE lit.
	// Partie M5 (2026-09-23, +47) : la valeur `a0` de `coverage.score.teamIdentity`, la premisse
	// « absent vaut zero » et ses deux garde-fous testes, l ordre des preuves, puis (M5.2, meme
	// montee) le champ `coverage.bridge.deathsFeed` et sa limite ecrite. Partie M4a (2026-09-23,
	// +31) : les PIECES MONTEES (`part`, `carrier`, `rides[].turret`), la variante nommee par la
	// piece ou par l arme, les compteurs de couverture et la racine `vehicleWeapons` resolue a la
	// requete. Exception ecrite, dans les commits de fusion qui reunissent la montee.
	// SCHEMA 69, PARTIE M6 (2026-09-24, lot M6 des retours du rejeu, rabattu de 70 sur 69 par sa
	// revue adverse — une seule montee par vague) : 2142 -> 2174, l en-tete v69 a quatre lots et la
	// partie M6 — la remise des mains nues sort des ramassages et des changements d arme (deux
	// compteurs `unarmedGrants`), les consequences declarees hors du document (son, paliers de
	// socle) et les familles nommees au catalogue. Exception ecrite, meme commit que la partie.
	// SCHEMA 69, PARTIE M7 (2026-09-24, integration de M6 et M7 des retours du rejeu) : 2174 ->
	// 2186, une ligne d en-tete v69 et la partie M7 — la racine `vehicleScenery` resolue a la
	// requete (verdict de decor du service, deux replis nommes), sans effet sur l empreinte cuite.
	// Exception ecrite, dans le commit de fusion qui reunit la partie a la montee.
	// SCHEMA 69 -> 70 (2026-09-24, integration de la vague D des retours du rejeu) : +134 (2142 ->
	// 2276 sur la vague C sans M6 ni M7 ; 2186 -> 2320 a la fusion de la campagne a jour, lot
	// D-fix), L ENTREE UNIQUE de chronique v70 qui reunit les lots de la vague sous une seule
	// montee (un en-tete commun, puis une partie par lot), APRES les parties M6 et M7 du 69.
	// Partie M2 (2026-09-23, +65 dont +11 a la revue adverse du 2026-09-24) : la regle des places
	// de l utilisateur, `roster[].presence` qui nait, la PLACE lue (table, tirs), chainee ou
	// ouverte, l equipe de l entite ti=9, les compteurs de `coverage.seats`, et les montees qui
	// l accompagnent (`grammar.Rev`, `SchemaDesFaits`). Partie M3 (2026-09-23) et en-tete commun
	// complete a sa fusion (`facts.Rev`), +59 : la dotation de naissance, `loadouts[].src` et
	// `.k`, `weaponChanges[].k`, le changement de SENS des premieres emissions,
	// `coverage.keyframes` et `.birthLoadouts`.
	// Partie D-fix (2026-09-24, lot correctif de la pre-integration de la vague D), +92 : la marche
	// qui ne perd plus ce qu un record prouve lui interdit de perdre, les images-cles douteuses qui
	// ne concluent rien, trois compteurs neufs et le compteur des mains nues de la naissance, les
	// deux regressions residuelles de M3 corrigees dans la marche des etats de mouvement, la mesure
	// au parc et le verdict ecrit des six ecarts de la pre-integration.
	// Reprise de la partie D-fix apres sa revue adverse (2026-09-24, meme montee 70, pre-integration
	// non fusionnee), +57 : la regle des en-tetes exacts, les compteurs neufs (preuves
	// contradictoires, verdict des NEW refuses), le repli nomme de la borne differee, les
	// changements des entrees figees declares et expliques, le choix du codec des faits (purge) et
	// les verdicts sur documents des vehicules fusionnes.
	// Exception ecrite, dans les commits de fusion qui reunissent la montee.
	"internal/games/halo_infinite/film/replay/document_chronicle.go": 2469,
	// --- production, hors perimetre du lot 2.7 (aucune preuve d equivalence ne couvrait
	// leur scission : elle se decidera au lot qui les rouvrira).
	// `assist.go` EST SORTI DE CETTE TABLE LE 2026-09-16 (lot 2.6.2) : le type `Assist` a descendu
	// dans `film/types`, le fichier est passe de 531 a 492 lignes — sous le seuil de 500, donc la
	// table n a plus a le connaitre (c est ce que `TestPlafondsDeTailleNeSontPasPerimes` exige).
	// --- entres en table A LA FUSION DU LOT 2.7g DANS L INTEGRATION (2026-09-17) : ces sept
	// fichiers etaient sous le seuil (ou n existaient pas) a la base du lot et l ont depasse par la
	// vague 2 de la famille 1.9 et le schema 60, avant que le ratchet n existe ici. Meme regle :
	// chaque valeur ne peut que descendre. `document_vehicles.go` et `score_timeline.go` sont dans
	// le perimetre du volet 2.7p (scission en cours) et sortiront de la table a sa fusion.
	"internal/games/halo_infinite/film/internal/facts/fallback/registre_killsource.go":                         555,
	"internal/games/halo_infinite/film/replay/document_shape_test.go":                                          511,
	"internal/games/halo_infinite/film/internal/facts/killsource/e197_identite_paquet_mesure_research_test.go": 654,
	"internal/games/halo_infinite/film/internal/facts/objectives/e1911_manches_mesure_research_test.go":        524,
	"internal/games/halo_infinite/film/internal/facts/objectives/statborg.go":                                  687,
	"internal/replaybuild/replaybuild.go":                                                                      577,
	// `equipment_creation.go` EST SORTI DE CETTE TABLE LE 2026-09-17 (lot 2.6.2, volet grammaire /
	// rejeu), pour la meme raison qu `assist.go` au volet facts : `EquipmentCreation` et
	// `EquipmentCreationStats` ont descendu dans `film/types`, le fichier est passe de 508 a 434
	// lignes — sous le seuil de 500, donc la table n a plus a le connaitre.
	//
	// TROIS PLAFONDS DE TEST MONTENT D UNE LIGNE le meme jour (523 -> 524, 655 -> 656,
	// 570 -> 571), et c est le meme mouvement vu de l autre cote : ces fichiers gagnent LA LIGNE
	// D IMPORT de `film/types`. C est la seule montee admise par ce lot, et elle est mecanique.
	// --- tests et instruments de mesure : tables de fixtures et balayages de recherche.
	"internal/games/halo_infinite/film/replay/golden_assembly_test.go":      1202,
	"internal/games/halo_infinite/film/internal/grammar/i59_anchor_test.go": 1152,
	// SCHEMA 61 -> 62 (2026-09-17, lot 4.2.1-b) : 1168 -> 1183, la raison ecrite de v62 que
	// `TestStructureIsOptionalInDocument` exige au-dessus de son epinglage dur.
	// SCHEMA 62 -> 63 (2026-09-19, post-chantier lot 5.1) : 1183 -> 1200, la raison ECRITE de la
	// montee v63 (les deux reapparitions publiees, les neuf compteurs, et ce qui ne monte PAS —
	// aucune des quatre revisions de decodage). C est l exception nommee en tete de ce fichier :
	// `structure_test.go` porte une entree par version de schema, et elle entre dans le commit
	// qui monte `SchemaVersion`.
	// SCHEMA 63 -> 64 (2026-09-20, lot 5.2-A) : 1200 -> 1215, la justification que
	// `TestStructureIsOptionalInDocument` exige avant d accepter la montee.
	// SCHEMA 64 -> 65 (2026-09-21, post-chantier lot 5.3.6) : 1215 -> 1221, la justification que
	// `TestStructureIsOptionalInDocument` exige avant d accepter la montee. Exception ecrite,
	// meme commit que `SchemaVersion`.
	// SCHEMA 65 -> 66 (2026-09-21, post-chantier lots 5.9.4 et 5.9.5) : 1221 -> 1236, la
	// justification que `TestStructureIsOptionalInDocument` exige avant d accepter la montee
	// (les deux genres neufs, LU et DERIVE, et ce qui valide la lecture de l index de fente) :
	// 1221 -> 1237.
	// Exception ecrite, meme commit que `SchemaVersion`.
	// SCHEMA 66 -> 67 (2026-09-21, post-chantier lot 5.10) : 1237 -> 1250, la justification que
	// `TestStructureIsOptionalInDocument` exige avant d accepter la montee — la lecture devient
	// la source primaire de l occupation, la proximite un repli qui lui cede, et `despawn` est
	// REFUSEE apres mesure de ses trois canaux. Exception ecrite, meme commit que
	// `SchemaVersion`.
	// SCHEMA 67 -> 68 (2026-09-22, post-chantier lot 5.22.4) : 1250 -> 1270, la justification
	// que `TestStructureIsOptionalInDocument` exige — le renommage de `stances[].kind`
	// `mobility` en `clamber`, pourquoi un renommage d enum publie est une montee de FORME, et
	// le refus de publier un genre `jump` LU faute de preuve. Exception ecrite, meme commit que
	// `SchemaVersion`.
	// SCHEMA 68 -> 69 (2026-09-24, integration de la vague C des retours du rejeu) : 1270 -> 1293,
	// la justification UNIQUE que `TestStructureIsOptionalInDocument` exige pour la montee commune
	// des lots M1 (grammaire de la vie, replis nommes, `samples[].g`), M5 (valeur d enum `a0`
	// du camp d un match a sens unique, champ `coverage.bridge.deathsFeed`) et M4a (pieces
	// montees posees sur leur porteur, sens neuf de `v` d un tir d artilleur et du `seat`
	// reporte). Exception ecrite, dans les commits de fusion qui reunissent la montee.
	// SCHEMA 69, PARTIE M6 (2026-09-24, lot M6 des retours du rejeu) : 1293 -> 1296, la ligne du
	// lot dans la justification de la montee 69 (remise des mains nues hors des ramassages et des
	// changements d arme). Exception ecrite, meme commit que la partie de chronique.
	// SCHEMA 69 -> 70 (2026-09-24, integration de la vague D des retours du rejeu) : +32 (1293 ->
	// 1325 sur la vague C sans M6 ; 1296 -> 1328 a la fusion de la campagne a jour, lot D-fix),
	// la justification UNIQUE que `TestStructureIsOptionalInDocument` exige pour la montee commune
	// de la vague : lot M2.3 (la presence et la place lues dans le film, le sens change de `seat`,
	// `seatSource` et `team`, revue adverse comprise) et lot M3 (quatre ajouts de forme et un
	// changement de sens : la premiere emission de chaque vie jugee contre sa naissance, jamais
	// contre un releve a venir). Exception ecrite, dans les commits de fusion qui reunissent la
	// montee.
	// Lot D-fix (meme montee v70, 2026-09-24) : 1328 -> 1333, sa ligne dans la justification v70.
	"internal/games/halo_infinite/film/replay/structure_test.go":                                  1333,
	"internal/games/halo_infinite/film/replay/t0_mouvement_research_test.go":                      872,
	"internal/games/halo_infinite/film/replay/inventory_position_i22_test.go":                     833,
	"internal/games/halo_infinite/film/replay/ground_link_research_test.go":                       814,
	"internal/games/halo_infinite/film/replay/ctf_retour_zone_research_test.go":                   812,
	"internal/games/halo_infinite/film/replay/assaut_manches_research_test.go":                    656,
	"internal/games/halo_infinite/film/replay/mapvar/noms_lieux_hunt_test.go":                     627,
	"internal/games/halo_infinite/film/killicon/killicon_test.go":                                 610,
	"internal/games/halo_infinite/film/internal/grammar/components_hooks_test.go":                 600,
	"internal/games/halo_infinite/film/replay/closures_test.go":                                   598,
	"internal/games/halo_infinite/film/replay/vehicules_v1a_test.go":                              587,
	"internal/games/halo_infinite/film/internal/grammar/vehicules_v13_deadstate_test.go":          583,
	"internal/games/halo_infinite/film/replay/minifilm_test.go":                                   581,
	"internal/games/halo_infinite/film/internal/grammar/ground_weapon_lifecycle_research_test.go": 574,
	"internal/games/halo_infinite/film/replay/equipment_uses_join_test.go":                        571,
	"internal/games/halo_infinite/film/internal/facts/objectives/assaut_pied_ancre_test.go":       558,
	"internal/sync/killcollector/positions_test.go":                                               557,
	"internal/games/halo_infinite/film/internal/grammar/golden_minibobine_test.go":                553,
	"internal/games/halo_infinite/film/internal/grammar/vehicules_v2_items_test.go":               533,
	"internal/games/halo_infinite/film/replay/attachement_phase0_bord_test.go":                    529,
	"internal/sync/killcollector/collector_test.go":                                               527,
}

// TestTailleDesFichiersDuFilmNeCroitPas : aucun fichier des racines surveillees ne depasse son
// plafond — 500 lignes par defaut, sa taille figee s il est dans la table.
func TestTailleDesFichiersDuFilmNeCroitPas(t *testing.T) {
	tailles := balayerTaillesFilm(t)
	if len(tailles) < plancherFichiersBalayesTaille {
		t.Fatalf("balayage muet : %d fichiers .go vus dans %v, plancher %d. "+
			"L arborescence a bouge ou le chemin relatif est casse — ce ratchet ne garde "+
			"plus rien et doit echouer bruyamment.",
			len(tailles), racinesSurveilleesTaille, plancherFichiersBalayesTaille)
	}
	chemins := make([]string, 0, len(tailles))
	for rel := range tailles {
		chemins = append(chemins, rel)
	}
	sort.Strings(chemins)
	for _, rel := range chemins {
		n := tailles[rel]
		plafond, fige := plafondsParFichier[rel]
		if !fige {
			plafond = seuilLignesParFichier
		}
		if n <= plafond {
			continue
		}
		if fige {
			t.Errorf("%s : %d lignes, plafond fige a %d (2026-09-16, lot 2.7).\n"+
				"UN PLAFOND NE MONTE PAS. Sortir autant de lignes ailleurs dans le fichier, "+
				"ou le scinder par deplacement pur avec sa preuve d equivalence.\n"+
				"Seule exception ecrite, et elle couvre DEUX fichiers (cf. l en-tete de "+
				"plafondsParFichier) : document_chronicle.go et structure_test.go, dont le "+
				"plafond monte du volume de l entree ajoutee, dans le commit qui monte "+
				"SchemaVersion.", rel, n, plafond)
			continue
		}
		t.Errorf("%s : %d lignes, seuil %d (CLAUDE.md regle 5).\n"+
			"Extraire un *_helpers.go / *_types.go, ou un sous-paquet. Mettre le fichier "+
			"dans plafondsParFichier N EST PAS une reponse : cette table est datee et fermee "+
			"au 2026-09-16, elle recense la dette constatee ce jour-la, pas la dette a venir.",
			rel, n, seuilLignesParFichier)
	}
}

// TestPlafondsDeTailleNeSontPasPerimes : une entree de la table qui designe un fichier disparu,
// ou qui est repassee sous le seuil, se RETIRE. Une allowlist perimee finit par autoriser
// n importe quoi (meme regle que les autres ratchets de ce paquet).
func TestPlafondsDeTailleNeSontPasPerimes(t *testing.T) {
	tailles := balayerTaillesFilm(t)
	entrees := make([]string, 0, len(plafondsParFichier))
	for rel := range plafondsParFichier {
		entrees = append(entrees, rel)
	}
	sort.Strings(entrees)
	for _, rel := range entrees {
		n, vu := tailles[rel]
		if !vu {
			t.Errorf("%s est au plafond mais n existe plus (renomme, deplace ou supprime) : "+
				"retirer l entree, ou la reecrire au nouveau chemin.", rel)
			continue
		}
		if n <= seuilLignesParFichier {
			t.Errorf("%s : %d lignes, soit sous le seuil de %d — retirer son entree de "+
				"plafondsParFichier. Le fichier est rentre dans la regle, la table n a plus "+
				"a le connaitre.", rel, n, seuilLignesParFichier)
		}
	}
}

// balayerTaillesFilm rend, par chemin relatif a `apps/go-api` (en slash), le nombre de lignes de
// chaque `.go` des racines surveillees. Le compte est celui de `wc -l` : le nombre de fins de
// ligne, la derniere ligne comprise si le fichier finit par un saut.
func balayerTaillesFilm(t *testing.T) map[string]int {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	out := map[string]int{}
	for _, racine := range racinesSurveilleesTaille {
		base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				nom := d.Name()
				if chemin != base && (strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") {
				return nil
			}
			blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du paquet
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(goAPIRoot, chemin)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = bytes.Count(blob, []byte("\n"))
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", base, err)
		}
	}
	if len(out) == 0 {
		t.Fatal(fmt.Sprint("aucun fichier .go balaye : ", racinesSurveilleesTaille))
	}
	return out
}
