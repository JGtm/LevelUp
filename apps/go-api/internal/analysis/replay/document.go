// Package replay assemble l'artefact de rejeu 2D (vue du dessus) d'un match à partir
// des données décodées du film : trajectoires des joueurs (Étape A) et géométrie de
// carte (Étape B) ; le kill feed (Étape C) reste à faire. Assemblage pur — aucun accès
// DB ni HTTP ; le décodage lourd est délégué à internal/analysis/filmdec.
//
// Le document (ReplayDocument) est produit HORS LIGNE par cmd/replay-build à partir des
// SEULS chunks du film (zéro capture Cheat Engine) et servi tel quel par l'API. C'est
// délibérément un DTO d'artefact bespoke (pas un type canonical) : c'est une charge utile
// de rendu, versionnée par SchemaVersion pour la compat client.
//
// Repère : les positions sont en MÈTRES MONDE. Ce paragraphe disait le contraire
// (« l'échelle/offset absolus ne sont PAS garantis », handoff ALL_PLAYERS_TRAJECTORIES) et
// c'était vrai AVANT filmdec/map_bounds.go : le film ne porte que des indices de quantum, et
// tant que les bornes du BSP manquaient, la déquantification employait celles de Cliffhanger
// pour toutes les cartes — d'où un facteur d'échelle arbitraire. Depuis, cmd/replay-build
// EXIGE la carte (`-map`) et refuse de produire un artefact sans ses bornes.
//
// CE QUE ÇA AUTORISE, et pourquoi la correction n'est pas cosmétique : le fond de carte
// figé (`MapBackground`) est calé dans ce même repère, donc il se superpose au rejeu.
// Contrôle sur 000d5950 : les bornes du rejeu tombent dans le cadre de `ridgeline.png`
// (cf. TestMapBackground_DonneesReelles). Le client garde son auto-cadrage via Bounds.
package replay

// SchemaVersion est incrémenté quand la forme de ReplayDocument change d'une façon que le
// client web doit gérer. L'ajout de champs OPTIONNELS (omitempty) ne casse pas le client
// et n'incrémente pas la version ; seul un changement cassant le fait.
//
// v2 (2026-08-02, lot 3.1/3.2) : les trois tables de libellés deviennent BILINGUES
// (`{en, fr}` au lieu d'une chaîne) et le type d'un lancer de grenade devient son RANG
// (`rank`) au lieu d'un nom. Motif : les catalogues étaient codés en Go, dont deux en
// français — ce qui interdisait l'anglais autant qu'un second titre — et les grenades
// étaient nommées deux fois, différemment, sur la même fiche.
//
// v3 (2026-08-13, plan parité lot 2) : le lancer de grenade publie son LIEN vers le
// projectile né de lui (`Grenade.proj`), et le document publie la table
// `killEffects` (weapon_key -> famille de rendu) qui donne leur famille aux effets de
// mort du kill feed. Les deux champs sont sérialisés en omitempty, mais la version
// monte quand même : les effets de repos de grenade côté client N'EXISTENT que si
// l'artefact porte le lien, et la reprise du backfill (lot 6) se fait par
// SchemaVersion — un artefact v2 doit se voir comme « à re-cuire », pas comme à jour.
//
// v4 (2026-08-14, plan parité lot 7.2) : le document publie son ORIGINE (`originMs`) —
// l'instant de sa frame 0 sur l'horloge du fil des éliminations (cf. origin.go). Le champ
// est optionnel, mais la version monte : sans lui le client ne peut PAS caler le fil
// autrement que par appariement statistique, et un artefact v3 doit donc se voir comme
// « à re-cuire », pas comme à jour.
// v5 (2026-08-14, plan parité lot 7.1) : le document publie `neutralDeaths` — le TYPE des
// morts que personne ne revendique (chute / hors-limites, ou sa propre source de dégât), lu
// dans le dead-state du film par le décodeur de source de dégât. Le champ est optionnel, mais
// la version monte : sans lui le fil ne peut poser sur ces lignes qu'un repère générique, et
// un artefact v4 doit se voir comme « à re-cuire », pas comme à jour.
//
// v6 (2026-08-14, plan PLAN_RANG_CAPACITE_I48 étape 1.2) : la capacité d'armure CHANGE DE
// GRANDEUR, et c'est pourquoi elle change aussi de nom. `Inventory.a` portait un INDEX
// TRONQUÉ — le motif d'ancrage du canal d'image-clé se termine par `010`, les bits de POIDS
// FORT du rang, si bien que ce canal ne voit que les rangs 16 à 23 et rendait `rang − 16`.
// Le document publie désormais `abilities` : le RANG de palette complet, sur une seule
// grandeur, alimenté par DEUX canaux (i48 pour toute la palette, l'image-clé pour sa fenêtre
// 16-23). `Inventory.a` est RETIRÉ plutôt que réinterprété : republier une autre grandeur
// sous la même clé aurait laissé tout client non mis à jour lire un nombre qui ne veut plus
// dire la même chose — c'est le défaut qui a coûté ce chantier. `abilityLabels` est donc
// keyé par RANG, et un artefact v5 doit se voir comme « à re-cuire », pas comme à jour.
//
// v7 (2026-08-16, plan PLAN_EQUIPEMENT_TI37 phase 1) : le document publie
// `equipmentEpisodes` — l'état ACTIF du camouflage et du surbouclier, en épisodes datés
// par vie (cf. equipment_episodes.go). Le champ est optionnel, mais la version monte :
// l'effet plein-fiche et les sons d'équipement côté client N'EXISTENT que si l'artefact
// porte les épisodes, et la reprise du backfill se fait par SchemaVersion — un artefact
// v6 doit se voir comme « à re-cuire », pas comme à jour (le re-build de masse pendant au
// registre portera ce champ avec le correctif de précision des objets du monde).
//
// v8 (2026-08-16, plan PLAN_GRAPPIN_LIGNE phase 1) : le document publie `grappleLines` —
// les TRACTIONS de grappin datées par vie avec leur point d'accroche en coordonnées monde
// (cf. grapple_lines.go ; source : le corps tag==3 d'i59, porté et prouvé au gate 0 du
// même plan). Le champ est optionnel, mais la version monte : la ligne joueur -> ancre
// côté client N'EXISTE que si l'artefact la porte, et la reprise du backfill se fait par
// SchemaVersion — un artefact v7 doit se voir comme « à re-cuire », pas comme à jour.
//
// v9 (2026-08-18, plan PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) : le document publie
// `equipmentPlacements` — les POSES d'objets d'équipement (mur de protection, capteur de
// menaces, et les objets du monde qui partagent l'archétype), avec leur position monde, leur
// fenêtre [t0, t1], l'identifiant `eqip` que le jeu leur donne, le poseur mesuré et son cap de
// visée (cf. equipment_placements.go ; identité prouvée au gate 1 de PLAN_IDENTITE_TI37, largeur
// du bloc MPP calibrée par oracle de position au gate 0 du présent plan). Le champ est
// optionnel, mais la version monte : les marqueurs d'équipement posé côté client N'EXISTENT que
// si l'artefact les porte, et la reprise du backfill se fait par SchemaVersion — un artefact v8
// doit se voir comme « à re-cuire », pas comme à jour.
//
// v10 (2026-08-18, plan PLAN_ORIGINE_POSES_ET_FAMILLES phase G) : chaque pose porte son
// ORIGINE MESURÉE (`origin` : `deployed` / `dropped` / `unknown`), et `coverage.placements`
// la croise avec la famille. La version monte alors qu'un champ s'ajoute à un sous-objet, et
// c'est justifié par ce que la mesure a trouvé : **`equipmentPlacements` n'était pas ce que
// son nom dit**. Sur les 11 films calibrés, 3 242 des 3 661 poses à poseur mesuré (88,6 %)
// naissent dans les 2 frames et les 1,5 m du DERNIER POINT de leur poseur — ce sont les objets
// qu'il PORTAIT, relâchés quand sa vie s'achève, pas des poses sur la carte. Un client v9
// dessine donc, aujourd'hui, un mur là où personne n'en a déployé ; sans montée de version il
// continuerait, puisque la reprise du backfill se fait par SchemaVersion.
//
// L'HYPOTHÈSE DU PLAN ÉTAIT L'AUTRE BOUT DE LA VIE, ET ELLE EST RÉFUTÉE : le critère écrit
// avant mesure cherchait une DOTATION AU SPAWN (création dans les 2 frames du DÉBUT de la vie
// du poseur). Elle compte 4 poses sur 3 661 (0,1 %), et les 4 sont des vies de 0,13 à 1,49 s
// où début et fin se confondent — le mode n'existe pas. Le témoin interne est dans la même
// mesure : distance médiane de la pose à la position de DÉBUT de vie 27,03 m, à celle de FIN
// 0,57 m, soit un facteur 47,5 sur les mêmes poses.
//
// v11 (2026-08-17, plan PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : `weaponPads` et `padPickups` —
// les SOCLES D'ARME du match. Chronique complète, et surtout ce que la mesure a REFUSÉ de
// publier (le ramasseur, les armes lâchées, le catalogue de carte) : document_ground_weapons.go.
//
// v12 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot A phase 1) : `scoreTimeline` — LE
// SCORE DANS LE TEMPS, et deux correctifs du calque `objectives` (vide en production, décalé de
// `originMs`). Chronique complète, oracle et limites : document_score.go.
//
// v13 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1) : `Point.p` — LE
// DEUXIÈME AXE DE LA VISÉE (élévation en degrés, positif = vers le haut). Un champ optionnel
// sur un sous-objet, et pourtant la version monte : jusqu'ici le cône de visée était dessiné à
// sa longueur maximale sur chaque point porteur de cap, ce qui AFFIRMAIT une visée horizontale
// que le film contredit. Chronique complète, convention mesurée et réserve : document_aim.go.
//
// v14 (2026-08-18, plan PLAN_OBJECTIFS_VIVANTS_2E_LECTURE phase 1 item 1.3) : `flagCarries` — LA VIE DE
// CHAQUE DRAPEAU de CTF, et `coverage.flagCarries`. Chronique, sources et refus : document_objectives_live.go.
//
// v15 (2026-08-18, plan PLAN_DRAPEAU_OBJET phase 2) : AUCUN CHAMP NEUF — c'est le CONTENU de
// `flagCarries` qui change. L'OBJET drapeau, lu dans le même archétype que les armes au sol,
// DATE le lâcher volontaire (un `carried_open` devient `carried`) et remet l'état `dropped` là
// où l'objet repose. Un artefact v14 se lit donc « à re-cuire », pas « à jour ». La piste de
// l'objet, elle, n'est PAS publiée : son contrôle de provenance l'a refusée. Chronique complète,
// mesures et refus : document_objectives_live.go.
//
// v16 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot C-bis phase 2b) : `zoneStates` — L'ETAT
// DE CHAQUE ZONE (qui la tient, depuis quand, jusqu'a quel niveau de jauge), et `coverage.zones`.
// Chronique, sources et refus : document_zones.go.
//
// v17 (2026-08-19, plan PLAN_POWERUP_SOCLE_CATALYST phase 8) : AUCUN CHAMP NEUF À LA RACINE —
// c'est le CONTENU de `weaponPads` qui change. Les SOCLES DE POWER-UP y entrent (voie `ti=37`,
// famille du manifeste pour identifiant), et `coverage.groundWeapons` gagne les quatre
// compteurs de cette voie. Un artefact v16 se lit donc « à re-cuire », pas « à jour » : il
// n'a jamais pu porter ces socles. Chronique, mesure et garde-fous : powerup_pads.go.
//
// v18 (2026-08-19, plan PLAN_EXPLOITATION_REGISTRE_FILM lot C-ter volet 3) : `zoneStates[].gauge`
// — LA JAUGE DE CAPTURE EN DIRECT (serie datee, allegee, pendant les rampes seulement) et
// `coverage.zones.gaugePoints`. Un sous-champ optionnel, et pourtant la version monte : le sommet
// statique de v16 (`progress`, CONSERVE dans le contrat) se lisait comme une jauge alors qu'il
// n'en etait que le maximum — le client cesse de le dessiner, et ne dessine l'arc que d'un
// artefact qui porte la serie. LE NUMERO EST 18, PAS 17 : le 17 est parti aux socles de power-up,
// fusionnes avant nous (regle du depot : un numero par montee, dans l'ordre de fusion). Un v16
// COMME un v17 se lit donc « a re-cuire » — ni l'un ni l'autre ne porte la serie. Chronique :
// document_zones.go ; regle et seuils : zone_states_gauge.go.
// v19 (2026-08-25, lot 1 « lecture vide ») : `inventory[].empty` — POURQUOI cette lecture ne
// rend rien (`dead` quand le fil des morts corrobore, `unknown` sinon). Un champ optionnel sur un
// sous-objet, et pourtant la version monte, pour la raison exacte des montées v13 et v18 : sans
// lui un artefact AFFIRME, par une lecture nue `{"t":N,"slot":S}`, que le joueur n'a plus rien —
// et le client, qui retient la lecture la plus recente <= T, EFFACE la fiche pendant ~20 s alors
// qu'une lecture pleine existait juste avant. 17,4 % des lectures publiees sont dans ce cas
// (mesure du 2026-08-24). Un artefact v18 doit donc se lire « a re-cuire », pas « a jour » : il
// ne peut porter aucun marqueur, et la reprise du backfill se fait par SchemaVersion. Chronique,
// mesure et temoin : inventory.go + inventory_dead_readings.go.
// v20 (2026-08-25, lot 4.4 du suivi delta de l'inventaire) : `grenadeReads` — LES GRENADES
// PORTEES SUR LEUR PROPRE AXE, alimentees par les images-cles ET par les paquets delta, chaque
// lecture publiant sa SOURCE (`kf` / `delta`). Le champ est optionnel, mais la version monte
// pour la raison exacte des montees v6 et v13 : le client CONSOMME cet axe pour la boite de
// grenades — il y lit desormais une lecture d'age median 8,09 s la ou `inventory` seul en
// donnait une de 10,00 s (mesure sur 70 films, 28 confrontables) — et la reprise du backfill se
// fait par SchemaVersion. Un artefact v19 doit donc se lire « a re-cuire », pas « a jour » : il
// ne peut porter aucune lecture delta.
//
// CE QUE LA VERSION 20 NE PORTE PAS, ET POURQUOI C'EST ECRIT ICI. Les MUNITIONS delta ont ete
// implementees et mesurees, puis REFUSEES par leur propre mesure : leur concordance avec les
// images-cles plafonne a 92,80 % et DESCEND quand on rapproche les deux lectures (88,06 % a
// 0,10 s contre 93,19 % a 2 s), ce qu'une consommation reelle entre les deux mesures ferait a
// l'envers. `Inventory.Am` reste donc alimente par les seules images-cles. Chronique complete,
// chiffres et porte par film : .ai/V7.5/replay2d/LOT4_SUIVI_DELTA_2026-08-25.md.
// CE QUE LA VERSION 21 PORTE, ET POURQUOI ELLE MONTE ALORS QU'AUCUNE CLE NE BOUGE. Le
// PROPRIETAIRE DE LA COLLINE est desormais publie sur la voie du designateur (KOTH) : une
// periode de colline se SUBDIVISE aux changements de main et chaque morceau porte son camp dans
// `ZoneSpan.Owner`, la ou un artefact 20 le laisse toujours nul. Le champ existait deja, seul son
// CONTENU change — la forme du document est identique, et le compte de champs ne bouge pas. La
// version monte pour la raison exacte des montees v6, v13 et v19 : la reprise du backfill se fait
// par SchemaVersion, et un artefact v20 doit se lire « a re-cuire », pas « a jour ». Sans ce
// bump, aucun rejeu deja cuit ne montrerait jamais la possession de la colline.
//
// NIVEAU DE PREUVE, ecrit ici parce que c'est ce qu'un lecteur doit trouver sur place : 88-89 %
// d'accord contre un temoin a 56 %, canal jamais refute, elu 4 films sur 4, erreur concentree aux
// BASCULES. Accepte par decision utilisateur du 2026-08-26 (precedent : la garde de l'ouvrier a
// 88 %). Les trois campagnes d'oracle et leurs negatifs sont en tete de `hillStatesOf`.
//
// CE QUE LA VERSION 21 NE PORTE PAS. Le portage du CRANE d'Oddball : son identite est etablie
// (`0x0017592C`, 4 films sur 4) et elle entre au manifeste, mais l'oracle du portage a ete
// REFUTE par sa propre mesure (40,6 a 66,7 % de trous a porteur unique contre un seuil de 90 %,
// temoin hors trou a 66,7 et 71,4 %). Aucun calque de portage de crane n'existe. Et les zones de
// TOTAL CONTROL : le designateur y rend jusqu'a 77 designations simultanees sur un mode a trois
// zones, la mesure est close `[!]` pour v7.5. Chronique complete : .ai/V7.5/PLAN_OBJECTIFS_
// ETAT_VIVANT_2026-08.md.
// CE QUE LA VERSION 22 PORTE, ET POURQUOI ELLE MONTE. La COURONNE VIP est desormais publiee
// (`vipCrown`) : chaque SELECTION `vip_selected` (`comp 22 A` = `TimesSelectedAsVip`, resolu au
// gate corrige — 100 % par joueur x3 films, temoin decale 0) ouvre une periode de port, fermee
// par la mort du VIP (kill feed) ou la selection suivante. La reconstruction a ete MESUREE : les
// periodes somment, par joueur, a `TimeAsVip` de l'API au SUB-SECONDE (recouv 100 % 3/3, 24/24
// joueurs a +0,2-0,3 s), contre un temoin d'attribution aleatoire effondre (exactitude 8/8
// contre 0-1/8). Le champ est optionnel, mais la version monte pour la raison exacte des montees
// v14 (drapeau), v16 (zones) et v21 (proprietaire de colline) : la reprise du backfill se fait
// par SchemaVersion, et un artefact 21 doit se lire « a re-cuire », pas « a jour » — sans quoi
// aucun rejeu VIP deja cuit ne montrerait jamais la couronne. GARDE DE MODE : `comp 22 A` vaut
// `flag_grabs` en CTF, donc la couronne n'est lue que sur les films que l'APPELANT reconnait VIP
// par `game_variant_name` (comme la colline de KOTH) — jamais devinee dans le film.
// Chronique complete : .ai/V7.5/replay2d/registre_film/VIP_COURONNE_PROTOCOLE.md.
//
// CE QUE LA VERSION 23 PORTE, ET POURQUOI ELLE MONTE. Le PORTEUR DU CRANE d'Oddball est desormais
// publie (`skullCarries`) : le porteur est le joueur dont les TICS DE SCORE DE MODE montent
// (`comp 0 A` = `skull_scoring_ticks`), un TRAIN de tics d'un meme joueur ETANT une periode de
// portage. Le porteur est nomme par le pont d'INSTANTS DE MORT PAR MANCHE (le slot est reattribue
// d'une manche a l'autre). Ce que le crane LIBRE (`objectiveObjects`, v21) refusait de dire — PAR
// QUI le crane est porte — est enfin dit : la couche LIBRE reste la POSITION du crane pose, ce
// calque-ci est le PORTEUR par-dessus. Le portage avait resiste a CINQ campagnes (proximite,
// traversee, score personnel : negatifs) ; le canal des TICS de score de mode, lui, tient — gate
// oracle porteur PRINCIPAL correct 7/7 films, gate terrain manche 1 de d9781168 prises 9/9 et
// porteurs d'intervalle 8/9 (seuil 8/9), emplacement identifie par l'oracle films confondus.
// Champ optionnel, mais la version monte pour la raison exacte des montees v14 (drapeau), v16
// (zones), v21 (colline) et v22 (couronne) : la reprise du backfill se fait par SchemaVersion, et
// un artefact 22 doit se lire « a re-cuire », pas « a jour ». GARDE DE MODE : `comp 0 A` est le
// score de mode de tout mode, donc le porteur n'est lu que sur un film que l'APPELANT reconnait
// Oddball par `game_variant_name` — jamais devine dans le film. Chronique complete :
// .ai/V7.5/replay2d/registre_film/ODDBALL_PORTEUR_PROTOCOLE.md.
//
// CE QUE LA VERSION 24 PORTE, ET POURQUOI ELLE MONTE. `equipmentEpisodes[].k`/`.a` — LES
// FRAGS ET ASSISTANCES DU PORTEUR pendant l'episode (camo, surbouclier), et
// `coverage.equipment.killsRead` qui dit si la mesure a ete TENTEE pour ce match (faux =
// non mesure, jamais confondre avec un compte a zero). Champs optionnels sur un sous-objet
// deja publie, et pourtant la version monte, pour la raison exacte des montees v13/v18/v19 :
// un artefact 23 n'a jamais pu porter ces compteurs, et la reprise du backfill se fait par
// SchemaVersion. PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1, decision utilisateur 8a/8b
// (DEC-7 revisee) : GO a petite population (camo 35,2 % = 25/71, surbouclier 55,6 % = 10/18,
// global 39,3 % = 35/89 en lecture STRICTE `LineByLinePublishable` — la population qui
// affiche reellement des chiffres) ; re-mesure obligatoire apres la cuisson de masse.
// CE QUE LA VERSION 25 PORTE. Les PRISES ET LES LÂCHERS D'ARME (`weaponChanges`), datés à la
// milliseconde du paquet et nommés. Jusqu'ici le document ne portait, sur ce sujet, que
// `padPickups` : « ce socle s'est vidé quelque part dans cet intervalle », sans le joueur. Le
// négatif du 2026-08-12 (« le film ne porte aucun événement de ramassage ») visait l'archétype
// ARME AU SOL ; le signal est sur le PORTEUR, et il y est daté.
//
// NIVEAU DE PREUVE, écrit ici parce que c'est ce qu'un lecteur doit trouver sur place. Le canal
// est JUSTE : sur 5 627 tirs de trois films, il ne retire jamais une arme encore utilisée. Sa
// COMPLÉTUDE n'est PAS établie — les oracles hors ligne sont soit trop grossiers (images-clés,
// 20 s), soit saturés (l'union des inventaires plafonne à 98-100 % avant lui). Ce qui a été
// mesuré à la place est la PLAUSIBILITÉ : hors drapeaux, 22 et 21 ramassages par match sur deux
// CTF Arena, composés d'armes de socle et de râtelier (Gravity Hammer, S7 Sniper, M41 SPNKr,
// Pulse Carbine, BR75) et jamais d'armes de départ, pour 10 et 13 socles sur ces cartes.
//
// CE QUE LA VERSION 25 NE PORTE PAS. Le SOCLE d'origine d'une prise : trois hypothèses de lien
// vers l'objet du monde ont été mesurées et réfutées (suppression 1/71, attachement 1/21,
// appariement par les armes 5-12 % contre 70 % exigés). Et la FIN DE VIE réelle d'une arme
// lâchée : `WeaponChange.Until` applique la durée publiée par le jeu comme CONVENTION
// d'affichage, parce que le jeu n'a pas de minuterie inconditionnelle — seules 5 à 14 % des
// armes au sol reçoivent un événement de disparition dans le film, et c'est son comportement,
// pas un défaut de lecture.
//
// CE QUE LA VERSION 26 PORTE. Les RAMASSAGES ET LES CONSOMMATIONS D'ÉQUIPEMENT
// (`equipmentChanges`) : la capacité d'armure suit la même règle que l'arme en main — son
// composant (i48) n'entre au masque du flux delta que lorsqu'elle CHANGE, donc chaque émission
// est un événement daté. Le document portait déjà `abilities[]`, qui dit ce qu'un joueur PORTE ;
// ce calque dit ce qui lui ARRIVE. Champ optionnel, mais la version monte pour la raison exacte
// des montées v14, v16, v21, v22 et v25 : la reprise du backfill se fait par SchemaVersion, et un
// artefact 25 doit se lire « à re-cuire », pas « à jour ».
//
// NIVEAU DE PREUVE — ET IL EST MEILLEUR QUE CELUI DES ARMES, pour une raison de format. Le
// compteur de rotation d'i48 avance de 1 à chaque émission (aucune répétition sur 50 transitions,
// 3 films) et repart à 5 à la première émission de chaque vie (264 cas sur 269). Ce calque porte
// donc son PROPRE TÉMOIN DE COMPLÉTUDE : un pas de compteur supérieur à 1 dénonce les émissions
// manquées et les compte — environ 16 pour 319 vues sur le corpus, soit de l'ordre de 95 % de
// couverture, LUE et non supposée. La couverture publie ce témoin (`missedEstimate`,
// `counterJumps`, `livesFirstOffSpec`). Deux autres propriétés sont mesurées : la porte ouverte
// est la CONSOMMATION et jamais la mort (17 cas sur 3 films, zéro dans la dernière seconde de la
// vie, la plus tardive laissant 8,8 s à vivre) ; et la première émission d'une vie n'a PAS un
// sens unique — contemporaine de la naissance du bipède c'est une réapparition équipée (83 % des
// vies d'un film à 0 ms), tardive c'est un ramassage (médiane 16 à 18 s sur deux films d'arène,
// 0 % sous la seconde). Les réapparitions sont donc ÉCARTÉES de la publication : les compter
// pour des ramassages fausserait le décompte du simple au double.
//
// CE QUE LA VERSION 26 NE PORTE PAS. Le SOCLE d'où vient l'équipement ramassé — même impasse
// que pour les armes, et pour les mêmes hypothèses réfutées. Ni ce que portait le joueur avant
// la première émission d'une vie : `EquipmentChange.From` vaut alors `NoAbilityRank`, et le
// film ne dit rien de plus.

// CE QUE LA VERSION 27 PORTE, ET CE QU'ELLE RETIRE. Les ARMES AU SOL individuelles
// (`groundWeapons`) : chaque objet arme qui a bougé — l'arme d'un mort, l'arme de départ
// abandonnée — avec sa position de repos, son origine mesurée, et une fin OBSERVÉE : `pickup`
// (une prise du flux delta tombe dans sa fenêtre de vie à moins de 1,5 m — mesure fondatrice
// du 2026-08-30 : l'objet le plus proche d'une prise est à 0,61-0,75 m en médiane contre 4-7 m
// pour un témoin), `seen` (dernière image-clé qui le recense — la disparition est dans les
// ~20 s suivantes), ou `open` (rien ne prouve sa disparition). EN CONTREPARTIE, LE CHAMP
// `until` de `weaponChanges` (v25) EST RETIRÉ : c'était une durée de table (10/20/30 s), une
// convention refusée par l'utilisateur — « je veux juste voir quand elle est au sol et quand
// elle disparaît ». Un artefact 26 doit se lire « à re-cuire » : il porte encore la convention
// et aucune arme au sol observée.
// CE QUE LA VERSION 28 PORTE, ET CE QU'ELLE REFUSE. Les POSES D'ÉQUIPEMENT gagnent leur FIN
// D'AFFICHAGE OBSERVÉE (`until` / `untilMax` / `end` sur `equipmentPlacements`) : la même
// mécanique que les armes au sol de v27 — dernière image-clé qui recense l'objet, première qui
// ne le recense plus — appliquée au recensement `ti=37` que la chaîne des socles lisait déjà.
// Jusqu'ici `t1` (fin du MOUVEMENT) était la seule borne et son contrat interdisait de s'en
// servir comme disparition ; un artefact 27 doit se lire « à re-cuire ». CE QUE LA VERSION
// REFUSE, mesure à l'appui (2026-08-30, mesure D) : la fin `pickup` pour l'équipement. Le lien
// spatial prise i48 -> pose est RÉFUTÉ — l'équipement tombe à la mort AVEC les grenades du
// mort, plusieurs objets naissent au mètre carré, et la matrice GlobalID x rang des liens
// n'est pas diagonale (un même objet lié à trois rangs ; à candidat unique, 0 à 2 paires par
// film, incohérentes). Le ramassage d'équipement reste dans `equipmentChanges` (QUI et QUAND),
// sans lien vers l'objet du sol.
// SCHEMA 29 (2026-08-31) — LA LUNETTE. `Point.S` publie le palier de visee a la lunette, lu
// dans les evenements `unit_zoom` du film (type 21) et non dans le record de position : c'est un
// etat A BASCULE, d'une autre source que le cap et l'elevation. La version monte pour la raison
// exacte du schema 13 (l'elevation) — ce n'est pas un champ de plus, c'est le SENS DU CONE DE
// VISEE qui change une seconde fois : jusqu'ici le client dessinait la meme ouverture pour un
// joueur a la hanche et un joueur a la lunette, et affirmait donc, sans le dire, que la visee
// etait toujours aussi large. La reprise du backfill se faisant par SchemaVersion, un artefact
// 23 doit se lire « a re-cuire », pas « a jour ».
//
// CE QUI A RENDU CE CHAMP POSSIBLE, ET IL FAUT LE DIRE : sept campagnes de mesure ont conclu
// « aucun evenement de zoom dans la bobine ». Elles lisaient le type d'evenement decale d'UN bit
// (le bit de configuration en tete de paquet etait ignore), et leurs chaines pretendument
// independantes partageaient cette erreur. Le negatif est REFUTE ; ~400 000 evenements de
// lunette dorment dans le corpus. Validation, pont vers le joueur et reserves : sur `Point.S`.
//
// CE QUE LA VERSION 30 PORTE. Le RAMASSAGE NATIF (`pickups`) : l'événement `biped_pickup` de
// la bobine — le type 9 de la liste d'événements d'un paquet delta, décodé pour la première
// fois (grammaire lue dans l'exe, cadrage jugé par l'oracle de trame sur deux films). Il DATE
// à la milliseconde, ATTRIBUE (sa référence vaut `512 + index` = le slot du ramasseur, exact
// sur 32/32 paires de vérité terrain) et NOMME l'objet par son identifiant de catalogue —
// le même espace de valeurs que `Loadout.W` pour les armes (100 % des familles vues par
// i43..i46 y figurent). Son R(3) de tête sépare les armes du reste : classes 0/1 → 63-72 %
// d'armes connues, classes 2/3 → 0,0 % sur 118 événements.
//
// CE QUE LA VERSION 30 LÈVE. `padPickups` cesse d'être un intervalle anonyme de vingt
// secondes : quand un ramassage natif de la MÊME famille tombe dans la fenêtre, l'occupation
// porte son instant EXACT (`t`) et son ramasseur (`xuid`). C'est la condition de levée que le
// contrat de `PadPickup.XUID` avait écrite — « un oracle plus RAPPROCHÉ que 20 s » — et elle
// n'est plus une inférence : l'événement porte le ramasseur. RIEN N'EST EFFACÉ : une
// occupation que le canal natif ne couvre pas garde son intervalle intact, et `xuid` y reste
// `null`. Un artefact ANTERIEUR se lit donc sans changement ; il ne porte simplement pas ces
// datations. (La version 29 est celle de la LUNETTE, arrivee en parallele sur feat/v75 : les
// deux chantiers ont pris le 29 le meme jour, celui-ci a ete renumerote 30 au merge.)
//
// CE QUE LA VERSION 30 REFUSE. De remplacer `weaponChanges` : les deux canaux coexistent parce
// qu'ils ne disent pas la même chose (l'un qualifie prise/lâcher/échange et connaît
// l'emplacement d'arme, l'autre voit des prises que le premier rate et nomme le ramasseur).
// Et de prétendre à la complétude : le balayage ne décode que l'événement EN TÊTE de sa liste,
// donc un ramassage en deuxième position lui échappe — `coverage.pickups.multiEvent` publie
// cette borne. Les classes non-arme SONT publiées, sur mesure et non par principe : 80,5 % et
// 72,2 % d'entre elles n'ont AUCUNE émission i48 du même slot à moins de 500 ms (témoin décalé
// à 0,0 %) — elles comblent un trou, elles ne doublonnent pas `equipmentChanges`.
//
// CE QUE LA VERSION 31 PORTE : LE NOM DE L'OBJET RAMASSÉ (`pickups[].family`), et une nature
// de ramassage à trois valeurs au lieu de deux.
//
// D'OÙ VIENT LE NOM, ET POURQUOI CE N'EST PAS UNE STATISTIQUE. Le `R(32)` des classes non-arme
// est un GlobalID de tag `eqip` : le manifeste `[[equipment_objects]]` du titre le nomme, et ce
// manifeste a été bâti en remontant la chaîne `sofd -> sofa -> {string_id, eqip}` dans les
// FICHIERS DU JEU (2026-08-18). Mesure du 2026-09-01 sur les deux films de référence :
// 82/82 et 36/36 des ramassages non-arme résolus, 8/8 identifiants distincts, ZÉRO
// chevauchement avec le catalogue d'armes dans les deux sens, et concordance 2/2 avec les deux
// étiquettes que la corrélation avait acquises de son côté. Les armes se résolvent par le même
// geste dans `LabelCatalog.Keys` (famille -> weapon_key).
//
// CE QUE LA VERSION 31 TRANCHE. `kind` distinguait l'arme du reste ; il distingue désormais
// `weapon` / `grenade` (classe 2) / `equipment` (classe 3). La séparation ne vient pas d'une
// corrélation mais du NOM : une fois les identifiants résolus, la classe 2 est grenade dans
// 100,0 % de ses événements et la classe 3 dans 0,0 %, deux films, aucun identifiant réparti
// sur les deux classes. `item` N'EST PAS RENOMMÉ : il reste le repli des classes non-arme dont
// la nature n'est pas établie.
//
// CE QUE LA VERSION 31 REFUSE. De descendre un libellé : `family` est un SLUG, la traduction
// reste au client (règle multi-titre). De remplir un nom qu'elle n'a pas : un identifiant
// qu'aucun catalogue ne connaît sort SANS `family`, et `coverage.pickups.unknownFamilies` le
// compte — le manifeste ne déclare que 21 objets, les trous doivent se voir. Et de publier
// l'ORIGINE d'une prise (socle de la carte contre objet tombé au sol) : mesurée le 2026-09-01,
// non concluante (25,6 % d'injectivité contre 50 % exigés), et le dépôt ne déclare aucun point
// d'apparition d'équipement. La réfutation reste en place.
//
// RISQUE DE COLLISION DE NUMÉRO, consigné comme au schéma 29->30 : ce lot prend le 31 sur la
// branche `wt/pickup-nommage` alors que le 30 vient d'arriver sur `feat/v75`. Un autre chantier
// peut prendre le 31 le même jour ; l'arbitrage se fait au merge, par renumérotation, comme la
// dernière fois.
//
// CE QUE LA VERSION 33 PORTE, ET CE QU'ELLE REFUSE. L'ARMEMENT DE LA BOMBE d'Assaut
// (`bombArmings`) : le début du hold (`bomb_arming_start`), l'instant armé (`bomb_armed`) et la
// mèche (4,93 s), lus dans l'anneau du marqueur `ti=12 i14` — protocole du 2026-09-01 avec
// tirage nul (13/13 Neutral Bomb CV 0,016, 4/4 Husky Raid, 0/1000 tirages nuls aussi bien). Le
// compte à rebours côté client N'EXISTE que si l'artefact porte le calque, et la reprise du
// backfill se fait par SchemaVersion : un artefact 32 doit se lire « à re-cuire », pas « à
// jour ». CE QUE LA VERSION REFUSE : ONE BOMB, où le signal ne tient pas (CV 0,725, 87/1000) —
// deux gardes indépendantes (nom de variante chez l'appelant, confrontation locale aux
// explosions du même film) retiennent le calque à la source ; et QUI ARME — le navpoint est un
// marqueur d'écran, pas un acteur, aucun xuid n'est publié. Chronique, sources et refus :
// document_bomb_armings.go. (Ce lot avait pris 29 et 30 sur `wt/bombe-visuel` pendant que
// la lunette, les ramassages et leur nommage prenaient 29-32 sur `feat/v75` : renumerote
// 33/34 au merge du 2026-09-01, l'arbitrage ecrit aux schemas 30 et 31.)
//
// CE QUE LA VERSION 34 PORTE. Le PORTEUR DE LA BOMBE d'Assaut (`bombCarries`) : les périodes
// de portage en intervalles de frames nommés par le xuid — le patron exact de `skullCarries`
// (v23), sur un AUTRE canal : la bombe est un OBJET TENU, répliquée dans le composant
// weapon-state-type-info du bipède comme une arme (famille `0x3fee4fcf`, B1 2026-09-01 :
// unique candidate des 9 films d'Assaut ; l'atlas HUD la nomme « ball | bomb »). PRISE =
// transition VERS la famille, LÂCHER = transition DEPUIS, et la MORT du porteur ferme SANS
// émission (piège mesuré — fermeture par le fil des morts, `BuildHeldObjectCarry`). Mesures :
// témoin Oddball 46/46 (100 %), porteur à la pose = détonateur statborg 13/17 (3 des 4
// désaccords penchent CANAL par la position, B3), mèche libre 27/28 (96,4 %) ; le délai
// médian lâcher -> explosion (4 804 ms) établit que le LÂCHER du canal EST le geste de pose.
// GARDE DE MODE : toutes les variantes de la famille bomb, ONE BOMB COMPRISE — le négatif de
// v33 vise l'anneau d'armement, pas ce canal. Champ optionnel, mais la version monte pour la
// raison exacte des montées v22, v23 et v33 : la reprise du backfill se fait par
// SchemaVersion, et un artefact 33 doit se lire « à re-cuire », pas « à jour » — sans quoi
// aucun rejeu d'Assaut déjà cuit ne montrerait jamais la bombe portée. CE QUE LA VERSION NE
// PORTE PAS : la bombe AU SOL — l'objet n'a pas de canal mesuré ; entre un lâcher et la
// prise suivante, le client la dérive des périodes et des pistes déjà publiées (dernier
// point du lâcheur), sans qu'aucune position inventée n'entre dans l'artefact. Chronique,
// sources et refus : document_bomb_carries.go.
//
// CE QUE LA VERSION 35 PORTE. LE RETOUR DU DRAPEAU DE CTF, dans ses deux moitiés. (1) Le retour
// AUTOMATIQUE est enfin DATÉ : un drapeau resté au sol rentre chez lui quand l'OBJET renaît à son
// socle (`coverage.flagCarries.homeByObject`). Jusqu'ici aucune chaîne ne le datait — le statborg
// ne crédite personne — et les états `dropped` couraient jusqu'à la reprise ou la fin de l'axe,
// des lâchers de plus de deux minutes qui n'ont jamais existé à l'écran. Contrôle : sur les
// retours que le statborg CRÉDITE, les deux chaînes tombent à la même frame dans 15 cas sur 15
// (100 %, écart médian 1 frame ; compté par ÉVÉNEMENT crédité DISTINCT, `flag_returns` ne nommant
// pas son drapeau). (2) `flagReturnZone` publie la RÈGLE du mode — rayon de la zone de
// retour, minuterie à vide, durée à un défenseur — que le titre déclare dans son manifeste. LA
// CONTESTATION N'EN FAIT PAS PARTIE : le jeu la décrit, mais l'utilisateur ne l'a jamais observée,
// ses constantes sont illisibles, et la mesure explique le silence (sur 72 lâchers où un ennemi
// entre dans la zone, 56 finissent par une REPRISE — à 1,3 m, un ennemi ne conteste pas, il
// RAMASSE). (3) LA VARIANTE « DRAPEAU NEUTRE » est reconnue : elle ne publie
// plus DEUX drapeaux qui n'existent pas mais UN SEUL, d'équipe -1, au socle du centre. Le mode
// n'est pas dans le film — c'est l'OBJET qui tranche, par le socle où il renaît, et la couverture
// publie le verdict avec les deux comptes qui le fondent (`neutralFlag`, `neutralBirths`,
// `teamBirths`). Un artefact 34 doit se lire « à re-cuire » : ses drapeaux au sol n'ont ni retour
// automatique ni zone, et ses parties à drapeau neutre portent un drapeau de trop. (Ce lot avait
// pris le 29 sur `wt/ctf-zone-retour` pendant que la lunette, les ramassages et la bombe
// prenaient 29-34 sur `feat/v75` : renumerote 35 au merge du 2026-09-02, l'arbitrage ecrit aux
// schemas 30, 31 et 33.)
//
// CE QUE LA VERSION 36 PORTE : L'IDENTITÉ DES VIES (lot du 2026-09-02, retour user « joueurs
// en attente de respawn éternels / quit-rejoin-bots »). (1) UNE TRACK = UNE VIE, réellement :
// la découpe applique `lifeGapUS` (la règle de `buildLifeSpans`) là où une track par SLOT
// fusionnait les vies d'un slot recyclé — le premier porteur nommé « vivait » à la place de
// son remplaçant, dont la fiche restait « Éliminé » à jamais. (2) LE NOMMAGE SE FAIT PAR VIE
// (`nameTracksByLives`, les fermetures nomment aussi la vie qu'elles closent) : un slot
// recyclé porte une identité PAR OCCUPANT. (3) LES BOTS EXISTENT : `roster[].bot` (entrée
// sans xuid, nom suffixé « [bot] », déclarée par BOT_METADATA) et `tracks[].bot` (le nom du
// bot sur les vies que le pont attribue à son index — fermeture A, jamais une devinette).
// La version monte pour la raison des montées v22/v23/v33 : la reprise du backfill se fait
// par SchemaVersion, et un artefact 35 doit se lire « à re-cuire » — sans quoi aucun rejeu
// déjà cuit ne montrerait ni les occupants d'un slot recyclé ni les bots.
// CE QUE LA VERSION REFUSE : nommer une vie que rien ne fonde (les segments anonymes d'un
// slot nommé RESTENT anonymes — l'héritage de slot était exactement le bug corrigé) ; et le
// COMPTEUR DE RESPAWN RÉEL (`player-respawn-timer`, ti=5 i1) — décodé mais aux entiers BRUTS
// dont l'unité n'a jamais été calibrée (protocole cmd/tmp_vitals non joué) : publier une
// grandeur à unité devinée est interdit ici, la condition de reprise est au registre.
const SchemaVersion = 36

// ReplayDocument est le rejeu 2D sérialisé d'un match.
type ReplayDocument struct {
	SchemaVersion int    `json:"schemaVersion"`
	MatchID       string `json:"matchId"`
	TitleSlug     string `json:"titleSlug"`
	// FrameCount est le nombre de pas de temps discrets. Les points de trajectoire
	// référencent cet axe via Point.T dans [0, FrameCount).
	FrameCount int     `json:"frameCount"`
	Bounds     Bounds  `json:"bounds"`
	Tracks     []Track `json:"tracks"`

	// --- Champs OPTIONNELS (n'incrémentent pas SchemaVersion) ---

	// FrameIntervalMS est la durée réelle d'un pas de temps, en millisecondes. Absent =
	// axe de temps sans échelle (anciens artefacts) : le client choisit sa cadence.
	// Présent, la vitesse « 1x » vaut 1000/FrameIntervalMS frames par seconde réelle.
	FrameIntervalMS int `json:"frameIntervalMs,omitempty"`
	// DurationMS est la durée réelle couverte par le rejeu, en millisecondes.
	DurationMS int `json:"durationMs,omitempty"`
	// OriginMs est l'instant de la FRAME 0 sur l'horloge du fil des éliminations, en
	// millisecondes — c'est-à-dire sur l'horloge que le client reconstruit par
	// `event_time_ms + t0_ms` (le T0 réel du match, déjà servi par la Match View).
	//
	// CE QU'IL FERME. La frame 0 est calée sur le premier paquet de POSITION du film, un
	// instant qui varie de 3,6 s à 39,8 s après le début du film selon le match (chargement
	// et mise en place). Sans ce champ, poser le fil sur l'axe du rejeu exigeait de mesurer
	// ce décalage par appariement statistique côté navigateur — ce qui suppose des victimes
	// nommées, et échoue quand `killer_victim_pairs` ne couvre pas le match. Avec lui, le
	// recalage est une soustraction : `replayMs = event_time_ms + t0_ms − originMs`.
	//
	// D'OÙ IL VIENT : la différence de deux en-têtes de paquet du MÊME film (premier paquet
	// de position − premier paquet du chunk 1, qui est le zéro du film). Aucune base, aucune
	// horloge murale, aucune estimation. Détail, mesures et témoin indépendant : origin.go.
	//
	// POINTEUR, PAS int : c'est le PIÈGE omitempty. Une origine de zéro (film dont le
	// premier paquet porte déjà une position) serait omise et relue comme « pas d'origine »,
	// donc traitée en repli alors qu'elle est mesurée. ABSENT veut dire, et seulement :
	// l'origine n'est pas établie — le client retombe alors sur son appariement.
	OriginMs *int64 `json:"originMs,omitempty"`
	// Geometry est le fond de carte : props Forge orientés (repères contextuels, pas les
	// sols). Absent si la géométrie n'a pas été fournie au build.
	Geometry []MapObject `json:"geometry,omitempty"`
	// GeometryBounds est l'étendue XY de Geometry, distincte de Bounds (les props
	// débordent de la zone parcourue). Le client peut cadrer sur l'union des deux.
	GeometryBounds *Bounds `json:"geometryBounds,omitempty"`
	// Structure est la géométrie STRUCTURELLE de la carte : l'emprise au sol de chaque
	// instance de géométrie instanciée du BSP (sols, plateformes, rampes, murs), avec
	// l'altitude de sa face supérieure. C'est le VRAI fond de carte, à distinguer de
	// Geometry (props Forge, 0,25 m² de médiane, 3,4 % de la carte couverts).
	// Absente si la carte n'a pas de fichier de structure figé (cf. cmd/mapstruct-build).
	Structure []Surface `json:"structure,omitempty"`
	// StructureBounds est l'étendue XY de Structure (elle déborde largement de Bounds :
	// la structure couvre toute la carte, les joueurs n'en parcourent qu'une partie).
	StructureBounds *Bounds `json:"structureBounds,omitempty"`
	// Shots est la liste des tirs décodés du film et RATTACHÉS à un slot (cf. shots.go).
	// Absent si le décodage n'a rien pu rattacher. Ce n'est PAS la liste exhaustive des
	// tirs du match : voir Shot pour ce que le champ garantit et ce qu'il ne garantit pas.
	Shots []Shot `json:"shots,omitempty"`
	// Loadouts est l'inventaire d'armes de chaque slot aux instants de keyframe (cf.
	// loadouts.go). Absent si le film n'a livré aucun loadout.
	Loadouts []Loadout `json:"loadouts,omitempty"`
	// Inventory est l'inventaire complet lu aux images-clés : grenades portées avec leur type,
	// capacité d'armure, munitions et emplacement dégainé (cf. inventory.go). Absent si le film
	// n'a livré aucun état.
	Inventory []Inventory `json:"inventory,omitempty"`
	// GrenadeLabels nomme les RANGS de type de grenade, dans l'ordre des compteurs
	// d'Inventory.G — et c'est LA SEULE table qui les nomme, y compris pour le type d'un
	// lancer (Grenade.Rank y est un index). Deux chaînes indépendantes établissent
	// l'ordre (35 lancers appariés aux décréments, et la table du binaire) : la question
	// est close. Source : replay_labels.toml du titre.
	GrenadeLabels []Label `json:"grenadeLabels,omitempty"`
	// Abilities est le RANG DE PALETTE de la capacité d'armure portée, lu au fil du film
	// (cf. abilities.go). Absent si aucun canal n'a rien rendu.
	Abilities []AbilityRead `json:"abilities,omitempty"`
	// GrenadeReads est l'axe des GRENADES PORTEES, alimente par DEUX canaux : le record de
	// biped des images-cles et les composants i22/i47 des paquets delta, chaque lecture disant
	// d'ou elle vient (cf. grenade_reads.go). Absent si aucun canal n'a rien rendu.
	//
	// IL NE REMPLACE PAS `Inventory` : celui-ci reste la source des munitions, de l'emplacement
	// degaine et du marqueur de lecture vide. Les deux axes coexistent parce qu'ils n'ont pas la
	// meme cadence, et les melanger ferait masquer une lecture pleine par une lecture partielle.
	GrenadeReads []GrenadeRead `json:"grenadeReads,omitempty"`
	// AbilityLabels nomme les RANGS de capacité que le document emploie.
	//
	// LA TABLE EST PARTIELLE, et un rang absent GARDE SON NUMÉRO à l'écran, marqué non
	// interprétable. Combler par le nom d'une capacité voisine se lirait comme une certitude.
	// La table est de surcroît PROPRE À LA PALETTE du match (cf. abilities.go) : deux films
	// peuvent donner deux noms différents au même rang, et un film dont la palette n'est pas
	// classée ne reçoit AUCUN nom.
	AbilityLabels map[string]Label `json:"abilityLabels,omitempty"`
	// EquipmentEpisodes est l'état ACTIF d'un équipement, en épisodes datés par vie
	// (cf. equipment_episodes.go) : camouflage (i28 queue[1], interrupteur mesuré) et
	// surbouclier (i5 non clampé, règle q > 64). Deux familles SEULEMENT, parce que deux
	// seulement sont mesurées — les autres équipements restent sans état plutôt que
	// devinés. Absent si aucune vie publiée ne porte d'épisode.
	EquipmentEpisodes []EquipmentEpisode `json:"equipmentEpisodes,omitempty"`
	// GrappleLines est la liste des TRACTIONS de grappin (cf. grapple_lines.go) : la
	// fenêtre datée [t0, t1] — du tir à l'ARRIVÉE mesurée sur la trajectoire — et le point
	// d'accroche en coordonnées monde. La position du joueur pendant la fenêtre est celle
	// de sa Track : la ligne se trace de la position courante vers l'ancre. Absent si
	// aucune traction n'a été lue (film sans grappin, ou vies non publiées).
	GrappleLines []GrappleLine `json:"grappleLines,omitempty"`
	// EquipmentPlacements est la liste des POSES d'objets d'équipement sur la carte
	// (cf. equipment_placements.go) : position monde, fenêtre [t0, t1], famille
	// (`wall` / `sensor` / `other`), identifiant `eqip` du jeu, poseur mesuré (-1 quand aucun)
	// et cap de visée du poseur quand il a été lu.
	//
	// `t1` EST UNE MISE AU REPOS, PAS UNE DISPARITION : le film ne date la disparition d'aucun
	// objet d'équipement (mesure du 2026-08-18). La fenêtre publiée est donc une BORNE
	// INFÉRIEURE de la présence de l'objet ; effacer la pose à `t1` affirmerait une disparition
	// que rien ne mesure.
	//
	// UNE POSE N'EST PAS RATTACHÉE À UNE Track : c'est un objet du monde, pas un joueur.
	// `owner` désigne la VIE qui l'a posé quand la proximité l'atteste, et rien d'autre.
	// Absent si le film n'a pas tranché la largeur de son bloc de réplication (la couverture
	// le dit alors : `calibrated: false`) ou s'il ne porte aucune pose.
	EquipmentPlacements []EquipmentPlacement `json:"equipmentPlacements,omitempty"`
	// WeaponChanges est la liste des PRISES ET DES LÂCHERS d'arme (cf.
	// document_weapon_changes.go) : qui, quand, quelle arme, et — sur un lâcher — jusqu'à
	// quelle frame le client peut montrer l'arme au sol. Les ré-annonces d'une arme déjà
	// portée au spawn en sont ÉCARTÉES : ce ne sont pas des ramassages. Absent si le film
	// n'en porte aucun.
	WeaponChanges []WeaponChange `json:"weaponChanges,omitempty"`
	// Pickups est la liste des RAMASSAGES NATIFS (cf. document_pickups.go) : l'événement
	// `biped_pickup` de la bobine, daté à la milliseconde, ATTRIBUÉ à son ramasseur et portant
	// l'identifiant de catalogue de l'objet.
	//
	// IL NE REMPLACE PAS `weaponChanges`, IL LE COMPLÈTE — et les deux sont publiés parce
	// qu'ils ne disent pas la même chose. `weaponChanges` sait qualifier (prise, lâcher,
	// échange) et connaît l'emplacement d'arme ; le canal natif ne qualifie rien mais il voit
	// des prises que l'autre rate et il nomme le ramasseur. Là où les deux voient la même
	// prise, ils s'accordent (21/21 et 11/12, arme nommée, à moins de 500 ms).
	Pickups []Pickup `json:"pickups,omitempty"`
	// EquipmentChanges est la liste des RAMASSAGES ET DES CONSOMMATIONS d'équipement (cf.
	// document_equipment_changes.go) : qui, quand, quelle capacité — et, sur une
	// consommation, laquelle vient d'être usée. Les annonces de RÉAPPARITION en sont
	// ÉCARTÉES : ce ne sont pas des ramassages, et ce que le joueur porte à sa naissance est
	// déjà dans `abilities`. Absent si le film n'en porte aucun.
	EquipmentChanges []EquipmentChange `json:"equipmentChanges,omitempty"`
	// GroundWeapons est la liste des ARMES AU SOL individuelles (cf.
	// document_ground_weapon_items.go) : où chacune gît, de quand à quand l'afficher, qui l'a
	// lâchée et qui l'a prise quand le flux delta le dit. Les fins sont OBSERVÉES (ramassage
	// daté, ou recensement des images-clés) — jamais une durée de table. Les armes de socle
	// restent au calque `weaponPads`. Absent si le film n'en porte aucune.
	GroundWeapons []GroundWeapon `json:"groundWeapons,omitempty"`
	// WeaponPads (les SOCLES D'ARME du match) et PadPickups (leurs occupations ACHEVÉES) : une
	// donnée de MATCH et non de carte, publiée seulement là où la récurrence est mesurée.
	// Forme, chronique et refus de publication : document_ground_weapons.go.
	WeaponPads []WeaponPad `json:"weaponPads,omitempty"`
	PadPickups []PadPickup `json:"padPickups,omitempty"`
	// Grenades est la liste des LANCERS de grenade rattachés à un slot (cf. grenades.go).
	// Contrairement aux tirs, chaque lancer porte son auteur DANS le film — il n'est pas
	// deviné. Ce n'est pas l'inventaire de grenades (c'est i22, non résolu) : c'est
	// l'événement « ce joueur a lancé cette grenade à cet instant ».
	Grenades []Grenade `json:"grenades,omitempty"`
	// Projectiles est la liste des TRAJECTOIRES de projectile (cf. projectiles.go). Le dernier
	// point est la derniere position REPLIQUEE, pas un impact : le film ne porte aucun
	// evenement de detonation.
	Projectiles []Projectile `json:"projectiles,omitempty"`
	// WeaponLabels nomme les identifiants d'arme employés par le document : famille (8 chiffres
	// hexadécimaux, cf. Loadout.W) ou identifiant global (16 chiffres, cf. Shot.Weapon) -> nom
	// canonique.
	//
	// POURQUOI UNE TABLE ET PAS UN NOM SUR CHAQUE ÉVÉNEMENT : 475 tirs pour 22 armes distinctes.
	// Répéter le libellé alourdirait le document sans rien apprendre.
	//
	// LE TAG BRUT RESTE À CÔTÉ DU LIBELLÉ, jamais à sa place — règle du dépôt : on ne stocke
	// jamais une résolution qui peut s'améliorer. Un identifiant absent de cette table garde
	// donc son hexadécimal à l'écran, et n'emprunte pas le nom d'une arme voisine.
	//
	// Source : `weapon_names.toml` du titre (nom, bilingue) + `replay_labels.toml`
	// (effet de rendu), joints par le weapon_key du registre d'armes.
	WeaponLabels map[string]WeaponLabel `json:"weaponLabels,omitempty"`
	// KillEffects associe un weapon_key du titre à la famille de RENDU de ses effets
	// (mêmes valeurs que WeaponLabel.Fx). C'est la table qui donne leur famille aux
	// EFFETS DE MORT : les kills du feed portent un weapon_key (résolu côté base),
	// jamais un identifiant d'arme film — sans cette table, le client ne peut joindre
	// aucun effet à un kill. Une clé absente = effet neutre, jamais celui d'une voisine.
	// Source : replay_labels.toml du titre ([shot_effects]).
	KillEffects map[string]string `json:"killEffects,omitempty"`
	// NeutralDeaths dit, pour les morts que PERSONNE ne revendique, DE QUOI le joueur est
	// mort. Le fil du rejeu en fait des lignes grises sans tueur ni arme ; sans cette table
	// il n'a qu'un repère générique à y poser. Absente = aucune mort de ce type établie sur
	// ce film — le fil garde son repère neutre, jamais l'icône d'une autre mort.
	NeutralDeaths []NeutralDeath `json:"neutralDeaths,omitempty"`
	// Roster est la liste des joueurs du film : leur identité et leur index de film.
	//
	// CE QU'IL SERT : le client y trouve l'ensemble des joueurs du match, y compris ceux dont
	// aucune vie n'a pu être nommée. Il lui permet aussi de traduire l'index de film porté par
	// les événements (un lancer de grenade écrit son auteur par index) en identité.
	// Absent quand le film n'a livré ni fil des morts ni table d'index.
	Roster []RosterEntry `json:"roster,omitempty"`
	// MapObjectives est le calque STATIQUE des objectifs du MODE JOUÉ (zones de
	// Bastion/Extraction, apparitions et livraisons de drapeau, socles), joint par
	// map_id au catalogue versionné — REMPLI À LA REQUÊTE par le service, jamais écrit
	// dans l'artefact : l'artefact ne connaît ni sa carte ni son mode (cf.
	// map_objectives.go). Absent quand le mode n'a pas d'objectifs statiques (Slayer),
	// quand map_id est vide ou la carte hors catalogue — jamais une erreur.
	MapObjectives *MapObjectives `json:"mapObjectives,omitempty"`
	// MapWeaponPads est le calque des EMPLACEMENTS DE SOCLE de la carte, CROISÉ avec les
	// socles du match — REMPLI À LA REQUÊTE par le service, jamais écrit dans l'artefact
	// (même règle et même raison que MapObjectives, cf. map_weapon_pads.go).
	//
	// IL NE PORTE QUE LES EMPLACEMENTS ALLUMÉS, et c'est une décision produit du
	// 2026-08-19 : le fichier de carte POSE les socles, le mode les ALLUME. Un emplacement
	// que `weaponPads` ne confirme pas à moins d'un mètre ne part PAS au client —
	// Cliffhanger en porte dix-sept au fichier, dix en CTF et zéro en Super Fiesta.
	//
	// CE QU'IL CHANGE POUR LE CLIENT : la position dessinée devient celle du SPAWNER, connue
	// dès la première image et au centimètre, au lieu du centroïde des apparitions vues. La
	// PRÉSENCE ne change pas : elle reste celle du match, socle par socle. Absent quand
	// map_id est vide, la carte hors catalogue, ou qu'aucun emplacement n'est confirmé — le
	// client retombe alors sur les socles du film seuls.
	MapWeaponPads *MapWeaponPads `json:"mapWeaponPads,omitempty"`
	// Objectives est la liste des ACTIONS D'OBJECTIF nommées : ce que chaque joueur a
	// accompli (capture de drapeau, retour, prise de zone, porteur stoppé), daté à la
	// milliseconde et attribué à un xuid (cf. objectives.go).
	//
	// CE QU'ELLE APPORTE que les autres calques n'ont pas : les tirs et les positions
	// disent où les joueurs étaient ; celle-ci dit ce qu'ils ont FAIT. Absente quand le
	// mode n'est pas un mode à objectifs, ou quand l'appelant n'a pas fourni les lignes de
	// match nécessaires au pont d'identité.
	Objectives []ObjectiveAction `json:"objectives,omitempty"`
	// ScoreTimeline est LE SCORE DANS LE TEMPS des deux camps et de chaque joueur (forme, oracle
	// et limites : document_score.go). Absente quand l'appelant n'a rien fourni à lire.
	ScoreTimeline *ScoreTimeline `json:"scoreTimeline,omitempty"`
	// FlagCarries est LA VIE DE CHAQUE DRAPEAU de CTF, en intervalles d'état (forme, sources et refus :
	// document_objectives_live.go). Absente hors CTF — `coverage.flagCarries` dit lequel des deux silences.
	FlagCarries []FlagCarry `json:"flagCarries,omitempty"`
	// FlagReturnZone est LA RÈGLE DE RETOUR du mode, telle que le manifeste du titre la donne
	// (schéma 35) : le rayon de la zone autour d'un drapeau tombé, la minuterie qui le ramène
	// tout seul, et la durée quand UN défenseur s'y tient. Le client en tire le cercle et la
	// jauge ; l'occupation, elle, se compte chez lui — l'équipe d'un joueur n'est PAS dans le
	// film (cf. Track.Team), elle vit dans la base et le client la joint déjà.
	//
	// ABSENTE quand le titre ne la déclare pas, ou quand le film n'est pas une partie de CTF :
	// rien à dessiner, et surtout pas un cercle sur un mode qui n'en a pas.
	FlagReturnZone *FlagReturnZone `json:"flagReturnZone,omitempty"`
	// ObjectiveObjects est OÙ SE TROUVE L'OBJET D'OBJECTIF QUAND PERSONNE NE LE PORTE — les vies
	// LIBRES du crâne d'Oddball (forme, canal et refus : document_objective_objects.go). Un trou
	// entre deux vies est un portage, mais le document ne dit PAS par qui : l'oracle du porteur a
	// été mesuré et réfuté (phase D4). Absente quand le titre ne déclare aucun objet publiable ou
	// quand le film n'en porte pas — `coverage.objectiveObjects` dit lequel des silences.
	ObjectiveObjects []ObjectiveObjectLife `json:"objectiveObjects,omitempty"`
	// ZoneStates est L'ÉTAT DE CHAQUE ZONE du mode, en intervalles de propriété (forme, sources et
	// refus : document_zones.go). Chaque entrée pointe une zone de `mapObjectives.zones` par son
	// index. Absente hors des modes à zones, et quand l'appelant n'a fourni aucun catalogue de
	// carte — `coverage.zones` distingue les deux silences.
	ZoneStates []ZoneState `json:"zoneStates,omitempty"`
	// VipCrown est LES PERIODES DE PORT DE LA COURONNE VIP, en intervalles de frames nommes par le
	// xuid du VIP (forme, sources et garde de mode : document_vip_crown.go). La couronne est a la
	// position de son porteur — le client la pose sur sa piste. Absente hors VIP —
	// `coverage.vipCrown` dit lequel des silences (film non-VIP contre film VIP sans periode).
	VipCrown []VipPeriod `json:"vipCrown,omitempty"`
	// BombArmings est L'ARMEMENT DE LA BOMBE d'Assaut : le début du hold, l'instant armé et la
	// mèche — le compte à rebours [t, t+fuseMs] se dessine sans autre donnée (forme, provenance
	// et refus : document_bomb_armings.go). Absente hors des variantes d'Assaut couvertes
	// (jamais One Bomb) — `coverage.bombArmings` dit lequel des silences.
	BombArmings []BombArming `json:"bombArmings,omitempty"`
	// SkullCarries est LES PERIODES DE PORTAGE DU CRANE d'Oddball, en intervalles de frames nommes
	// par le xuid du porteur (forme, sources et garde de mode : document_skull_carries.go). Le
	// crane porte est a la position de son porteur — le client le pose sur sa piste, comme la
	// couronne VIP. Absente hors Oddball — `coverage.skullCarries` dit lequel des silences (film
	// non-Oddball contre film Oddball sans portage). Le crane LIBRE (`objectiveObjects`) reste la
	// couche POSITION ; celle-ci est la couche PORTEUR.
	SkullCarries []SkullCarry `json:"skullCarries,omitempty"`
	// BombCarries est LES PERIODES DE PORTAGE DE LA BOMBE d'Assaut, en intervalles de frames
	// nommes par le xuid du porteur (forme, sources et garde de mode : document_bomb_carries.go)
	// — le patron de `skullCarries`, sur le canal des armes tenues. La bombe portee est a la
	// position de son porteur ; entre un lacher et la prise suivante, le client la derive des
	// periodes et des pistes (dernier point du lacheur — la bombe au sol n'a pas de canal
	// mesure). Absente hors de la famille bomb — `coverage.bombCarries` dit lequel des silences.
	BombCarries []BombCarry `json:"bombCarries,omitempty"`
	// Coverage dit, pour chaque calque, COMBIEN il a rattaché SUR COMBIEN existaient, et
	// pourquoi il a écarté le reste (cf. coverage.go).
	//
	// POURQUOI C'EST DANS LE DOCUMENT ET PAS SEULEMENT DANS LES JOURNAUX : publier 147 tirs
	// sans dire que 519 existent laisse croire à l'exhaustivité. L'écart doit être lisible
	// là où le résultat l'est. Absent des artefacts construits avant cette version.
	Coverage *Coverage `json:"coverage,omitempty"`
}

// RosterEntry est un joueur du film : son identité, et l'index sous lequel le film le désigne.
//
// LES DEUX NE SONT PAS INTERCHANGEABLES. Le xuid IDENTIFIE ; l'index ORDONNE, et il n'a de
// sens qu'à l'intérieur de ce film. Les garder côte à côte est ce qui permet de traduire un
// événement (qui porte l'index) sans jamais confondre l'un avec l'autre.
type RosterEntry struct {
	// XUID en décimal, même forme que Track.XUID et que la base.
	XUID string `json:"xuid"`
	// FilmIndex est l'index du joueur DANS CE FILM, lu dans les cinq bits qui précèdent son
	// xuid (cf. player_index.go, 26 chunks concordants sur le film de référence).
	FilmIndex int `json:"filmIndex"`
	// Name est le gamertag TEL QUE LE FILM L'ÉCRIT, dans le même enregistrement que le xuid.
	//
	// CE N'EST PAS UNE RÉSOLUTION : rien n'est allé le chercher ailleurs, donc rien ne peut
	// l'avoir mal apparié. Il rend le rejeu lisible sans base de données. Ce qu'il ne donne
	// PAS, et que seule la base porte : l'équipe, et les compteurs du match. Vide si
	// l'enregistrement ne le portait pas.
	Name string `json:"name,omitempty"`
	// Bot est vrai pour une entrée déclarée par BOT_METADATA (schéma 36) : son XUID est VIDE
	// — un bot n'en a pas, et le normaliser en pseudo-identifiant fusionnerait des bots — et
	// son Name porte le suffixe « [bot] », comme la base l'écrit. FilmIndex est le slot du
	// roster de réplication que le paquet type 12 déclare.
	Bot bool `json:"bot,omitempty"`
}

// Loadout est l'ensemble des armes PORTÉES par un slot à un instant de référence.
//
// CE QUE LE CHAMP GARANTIT : à l'instant T, ce slot AVAIT ces armes dans son inventaire.
// Témoin croisé sur une source indépendante (l'arme des events de tir) : 98,3 % d'accord
// contre 7,2 % pour le témoin qui ne casse QUE la jointure record->slot. Détail et limites
// en tête de loadouts.go.
//
// CE QU'IL NE GARANTIT PAS, et il faut le dire à l'écran :
//   - QUELLE arme est dégainée. Le loadout est l'inventaire, pas la main. Croiser avec le
//     dernier Shot du même slot désigne l'arme en main ; sans tir récent, on ne sait pas.
//   - la CONTINUITÉ. Un keyframe toutes les ~18-20 s : entre deux instants publiés, un
//     ramassage d'arme est invisible. Un client qui maintient la dernière valeur connue
//     affiche un état de référence, pas une mesure de l'instant.
//   - les GRENADES, la capacité d'armure, les munitions : NON décodées.
type Loadout struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped porteur : il désigne la Track concernée.
	Slot uint32 `json:"slot"`
	// W liste les identifiants de FAMILLE d'arme (high-32 du weapon-id 64 bits) en
	// hexadécimal 8 chiffres, dans l'ordre de lecture du record. La famille est l'identité
	// de l'arme, le suffixe bas ne porte que la variante cosmétique (cf. weaponv3.CanonWeaponID) ;
	// les alias d'un même canon sont repliés — un canon = une entrée.
	W []string `json:"w"`
}

// Shot est un tir décodé, placé à la position de son tireur.
//
// CE QUE LE CHAMP GARANTIT :
//   - le tir a INFLIGÉ DES DÉGÂTS. Le record du film (event type 105) n'existe que quand un
//     dégât est appliqué : il n'y a pas de record de tir manqué, donc pas de notion
//     « touché / raté » à afficher — tous les tirs publiés ici ont touché quelqu'un.
//   - l'origine (X, Y) est la position du biped tireur à l'instant du tir.
//
// CE QU'IL NE GARANTIT PAS :
//   - l'exhaustivité : seuls les tirs dont le tireur a pu être rattaché SANS AMBIGUÏTÉ sont
//     publiés (30 à 57 % des events selon le film) ;
//   - la VICTIME : elle n'est pas décodée du film (champ de largeur runtime). Le trait part
//     du tireur dans la direction visée, il ne relie pas deux joueurs.
type Shot struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped tireur : il désigne la Track d'où part le tir.
	Slot uint32 `json:"slot"`
	// X, Y sont l'origine du tir (position du tireur).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// H (optionnel) est le CAP DE VISÉE du tir en degrés, même convention que Point.H.
	// Absent quand la visée n'était pas lisible hors ligne (le champ vit après des boucles
	// de longueur variable dans ~80 % des records) : le client dessine alors un simple
	// marqueur, sans direction. Même PIÈGE omitempty que Point.H, même parade (0 -> 360).
	H float32 `json:"h,omitempty"`
	// Weapon est l'identifiant global 64 bits de l'arme, en hexadécimal (un entier 64 bits
	// ne survit pas au `number` JavaScript). Clé de metadata.weapon_labels.weapon_id.
	Weapon string `json:"w,omitempty"`
}

// Bounds est l'étendue alignée sur les axes de tous les points de trajectoire, dans le
// repère monde partagé. Permet au client d'ajuster la scène au viewport (le range monde
// absolu est inutile au rendu — seule la disposition relative importe).
type Bounds struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
	// MinZ / MaxZ (optionnels) donnent l'amplitude verticale, pour colorer les étages.
	// PIÈGE omitempty : une borne exactement nulle est omise — les valeurs sont issues
	// d'une déquantification à mi-bucket (min + step*(q+0.5)), un zéro exact est donc
	// hors d'atteinte en pratique ; le client lit une borne absente comme 0.
	MinZ float32 `json:"minZ,omitempty"`
	MaxZ float32 `json:"maxZ,omitempty"`
}

// Track est la trajectoire d'une entité (slot biped) sur la timeline du rejeu.
//
// ATTENTION : un slot est réattribué aux respawns — une Track = UNE VIE, pas un joueur.
// Le regroupement des vies par joueur se fait par XUID.
type Track struct {
	Slot uint32 `json:"slot"`
	// Team vaut -1 : L'ÉQUIPE N'EST PAS DANS LE FILM. Elle vit dans la base, avec le gamertag,
	// et le client la joint par XUID (cf. XUID ci-dessous). Le champ est conservé pour les
	// artefacts d'un titre qui la porterait ; le laisser à -1 n'est pas un oubli.
	Team int `json:"team"`
	// Name est TOUJOURS VIDE, et c'est délibéré : le film ne porte aucun gamertag. Le remplir
	// exigerait de lire la base depuis un outil hors ligne dont toute la valeur est de n'en
	// dépendre pas. Le client joint le nom par XUID.
	Name string `json:"name,omitempty"`
	// XUID est l'IDENTITÉ du porteur de cette vie, en décimal (un entier 64 bits ne survit pas
	// au `number` JavaScript ; le décimal est aussi la forme employée par la base).
	//
	// POURQUOI LE XUID ET PAS UN INDEX. Un index est un ORDRE, jamais une identité — la leçon
	// a coûté une fausse découverte à ce chantier (un tri alphabétique publié comme une
	// permutation du format). Le xuid est stable, global, et indépendant de tout tri : c'est
	// la seule clé sur laquelle un client peut joindre sans rien supposer.
	//
	// D'OÙ IL VIENT : le fil des morts du film nomme chaque vie par le xuid de sa victime
	// (cf. lives.go). Vide quand la vie n'a pas été nommée — 15 vies sur 105 sur le film de
	// référence, dont 4 antérieures au début réel du match et 6 survivants de fin de partie,
	// que le film ne clôt par aucun événement.
	XUID string `json:"xuid,omitempty"`
	// Bot est le NOM du bot qui porte cette vie (schéma 36), suffixe « [bot] » compris — un
	// bot n'a pas de xuid, et c'est le seul cas où une vie est nommée sans en avoir un.
	//
	// D'OÙ IL VIENT : BOT_METADATA (paquet type 12) déclare slot de roster + nom, et le pont
	// des fermetures attribue un slot de biped à cet index quand l'unicité le permet
	// (cf. nameBotTracks). Une vie de bot que le pont ne peut pas attribuer reste anonyme —
	// même règle que les humains : rien plutôt que faux.
	Bot    string  `json:"bot,omitempty"`
	Points []Point `json:"points"`
	// StartFrame / EndFrame (optionnels) bornent la vie de la track sur l'axe de temps :
	// le client peut masquer l'entité hors de cette fenêtre au lieu de la figer.
	StartFrame int `json:"startFrame,omitempty"`
	EndFrame   int `json:"endFrame,omitempty"`
}
