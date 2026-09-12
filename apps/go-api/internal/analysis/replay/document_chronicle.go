package replay

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
//
// CE QUE LA VERSION 37 PORTE. LE COUP D'ENVOI, DATE PAR LE FILM (`t0FilmMs`) : l'instant ou la
// grille se leve, lu dans le PREMIER MOUVEMENT des pistes au lieu d'etre estime des
// `first_joined_time` de l'API. L'estimation d'API degenere a ~0 ms sur 10-15 % des matchs
// (elle date alors le coup d'envoi au chargement) ; le film, lui, le porte directement, et la
// mesure du 2026-09-02 le montre PLUS STABLE que l'etalon — ecart-type 9 752 ms contre
// 12 764 ms sur les 49 matchs au T0-API sain, avec une marge interne au film de CV 0,013.
// Champ optionnel, mais la version monte pour la raison exacte des montees v4 (l'origine) et
// v22 : la reprise du backfill se fait par SchemaVersion, et un artefact 36 doit se lire
// « a re-cuire », pas « a jour » — sans quoi aucun rejeu deja cuit ne demarrerait sur le coup
// d'envoi. `coverage.t0Film` publie le verdict, refus compris. CE QUE LA VERSION REFUSE : un
// zero ambigu. Pas de mouvement detectable, une rafale de depart a moins de deux partants, ou
// un premier mouvement a plus de 120 s de la frame 0 -> champ ABSENT, refus journalise, raison
// publiee. Chronique, seuils et mesures : t0_film.go. (Ce lot avait pris le 36 sur
// `wt/t0-film` pendant que l'identite des vies prenait le 36 sur `feat/v75` : renumerote 37
// au merge du 2026-09-02 — l'arbitrage par renumerotation ecrit aux schemas 30, 31, 33 et 35.)
//
// CE QUE LA VERSION 38 PORTE : LA LECTURE FIABLE DES USAGES D'EQUIPEMENT (lots P1, P1bis et
// P3 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03, decisions user D1-D4 — « pas
// d'heuristique : les ENREGISTREMENTS du film, ou rien »). Quatre choses, un seul chantier :
//
// (1) `translocations[]` — LES TELEPORTATIONS DU TRANSLOCATEUR, datees ET SITUEES par
// l'EVENEMENT du film (type 117 `EquipmentTranslocatorTeleportEffects`, nom source de
// l'exe) : precision 18/18 et rappel 8/8 sur 5 films (rapport R1). Jusqu'ici le client
// DEVINAIT la teleportation d'un seuil spatial (> 4 m — aveugle a un saut de 3,24 m mesure)
// ou la datait du `spent`, qui peut suivre l'usage de 16,5 s. Chaque saut porte AUSSI son
// VA-ET-VIENT (`fx/fy/fz` -> `tx/ty/tz`) : la charge de l'evenement transporte deux
// positions quantifiees — A = depart, B = arrivee —, layout lu dans l'executable et valide
// 18/18 a 0,00-0,26 m des discontinuites de piste (rapport R6 §1). Le client n'a donc plus a
// deriver le lien d'une discontinuite de piste, et la faille se dessine au point de DEPART
// (le translocateur est un ECHANGE de positions). PAS DE BORNES DE CARTE -> PAS DE
// POSITIONS : les six champs sont solidaires et absents ensemble ;
// `coverage.translocations` compte les ecartees ET les positionnees.
//
// (2) `equipmentChanges[].recovered` et `[].gap` — LE CANAL i48 SE REPARE ET PUBLIE SON
// RESIDU. La recuperation GATEE PAR LE TEMOIN DE COMPTEUR (R2 : les octets des emissions
// manquees existent dans ~3 cas sur 4, sous deux formes que le balayage strict rejette par
// construction — record sans i0, masque dense R(64) a l'ordre de bits fige par P1.0 sur 44
// temoins) re-balaye les SEULES fenetres de saut et n'accepte un candidat QUE s'il comble
// exactement le saut ; le relachement inconditionnel est REFUTE (+800 fausses acceptations
// sur 10 films, R2 §5). `gap` publie le saut RESIDUEL apres recuperation : un `from` sous
// gap n'est plus une identite fiable, et le document le dit au lieu de le laisser croire.
// Cas fondateur (Dynasty 1b2d9e08) : la prise du translocateur de JGtm etait invisible et
// son `spent` portait `from=4` (le grappin) — l'emission recuperee (c6, rang 11) le
// corrige, et les stats comptent les recuperees A PART (`coverage.equipmentChanges.
// recovered` ; les temoins missedEstimate/counterJumps/livesFirstOffSpec decrivent la
// chaine FINALE, ce qui manque ENCORE).
//
// (3) LE FILTRE DE VITESSE S'EXEMPTE SUR PIECE (aucun champ neuf, mais le CONTENU des
// pistes bouge) : `DropTeleports` rejetait 1 a 3 echantillons REELS par teleportation
// (51/51 rejets a tort mesures, R3), il est leve a ±200 ms d'un evenement 117 du MEME
// slot — jamais ailleurs : invariance bit a bit prouvee CONTRE UNE IMPLEMENTATION DE
// REFERENCE FIGEE (la semantique d'avant l'exemption, copiee verbatim dans le test), sur
// donnees synthetiques (TestDropTeleportsInvarianceSansEvenement) et sur film sans tete
// 117 (TestP1InvarianceSansTete117, 188 979 echantillons identiques a la reference).
//
// (4) `abilityImpulses[]` — L'USAGE MESURE DU PROPULSEUR (lot P3, rapports R8 et R9). Le tag
// externe des composants bipede `biped-spartan-ability` (i57) et `-non-predicted-state`
// (i59) est un R(2) : la production n'en exploitait qu'UNE valeur, le corps `tag == 3` qui
// porte le GRAPPIN depuis le 2026-08-16. Le corps `tag == 1` du MEME composant date une
// IMPULSION, et son identite vient du canal i48 — rang lu dans la MEME VIE et
// ANTERIEUREMENT (le slot migre aux reapparitions, et un joueur peut changer d'equipement
// dans sa vie ; le `sub` du composant a ete essaye comme discriminant puis REFUTE par le
// corpus, R8 par. 8.5). Preuve : 0,361 impulsion par vie de propulseur contre 0,011 par vie
// de repulseur — PLUS porte que lui — et 0,000 sur 132 vies de grappin, qui a son propre tag
// (quatre films, R8 par. 8.8) ; oracle physique independant a 6,2-8,8 m/s de pic contre
// 2,9-3,6 au hasard ; et VERITE TERRAIN Theater du 2026-09-03 sur `1cd3848a` — 5 usages
// releves, 5 impulsions rendues, ecart <= 1 s. LE CALQUE NE COUVRE PAS TOUS LES EQUIPEMENTS,
// et il le dit : seules les familles que le TITRE declare mesurees y entrent (le propulseur,
// et lui seul — le REPULSEUR n'est PAS dans ce canal, negatif MESURE dont le lot R9 a ferme
// les trois dernieres portes), les autres sont ecartees ET COMPTEES
// (`coverage.abilityImpulses.otherFamily`) — et quand la chaine d'attribution elle-meme n'a
// pas pu tourner (palette du match non classee, titre sans famille declaree, aucune vie), les
// gestes tombent dans un compteur A PART (`noResolver`) plutot que de se deguiser en « un
// autre equipement ». Aucune grammaire nouvelle n'a ete portee : les
// deux deserialiseurs publiaient deja le tag ; ce qui manquait etait le croisement avec
// l'identite.
//
// (5) `abilityCharges[]` — LES CHARGES RESTANTES (lot P5 du 2026-09-04, rapport R11). Le
// composant i56 `biped-spartan-ability-energy` transmet AU CHANGEMENT un compteur de
// charges entieres par emplacement arme (quartet HAUT de la valeur 7 bits — la lecture
// discrete du consommateur de l'exe, R11 §1.1). Serie temoin 4, 3, 2, 1, 0 sur `1cd3848a`,
// exactement aux cinq usages du releve Theater ; 36/36 accroches de grappin appariees a une
// baisse (temoin decale : 2/36). L'identite vient d'i48 (meme vie, anterieurement — la MEME
// jointure que les impulsions), la famille de la palette du titre, et seules les familles
// declarees mesurees (`[ability_charges]` du manifeste : grapple, thruster) sont publiees —
// le REPULSEUR n'arme jamais i56 (218 vies, 0 baisse, R11 §4-5). Ce sont les LECTURES,
// jamais un compte d'usages derive (une baisse peut valoir plusieurs usages), et rien n'est
// affirme avant la premiere lecture (le film ne transmet rien au ramassage — masque a 0 =
// le moteur pose plein).
//
// POLITIQUE DE SCHEMA DU LOT P5, VERIFIEE SUR PIECES LE 2026-09-04 : contrairement aux lots
// P1bis et P3 (« aucun artefact 38 cuit »), DES ARTEFACTS 38 EXISTENT desormais hors
// repertoires de test — `1b2d9e08` et `1cd3848a`, cuits pour le gate visuel du chantier. Le
// schema RESTE a 38 malgre cela, et la justification n'est pas la commodite : les deux
// ajouts (`abilityCharges` racine, `coverage.abilityCharges`) sont PUREMENT ADDITIFS et
// omitempty — un lecteur 38 existant qui les ignore reste correct sur toute ligne qu'il
// sait lire, et les deux artefacts cuits sont des temoins de gate destines a etre re-cuits
// avec le lot (la re-cuisson du parc, elle, n'a pas commence : la reprise par SchemaVersion
// ne perd rien). Une montee a 39 aurait marque « a re-cuire » un parc entier qui est DEJA
// tout entier anterieur ou egal a 38 — elle n'aurait protege aucun lecteur de plus.
//
// Champs optionnels, mais la version monte pour la raison exacte des montees v14/v22/v25 :
// la reprise du backfill se fait par SchemaVersion, et un artefact 37 doit se lire « a
// re-cuire », pas « a jour » — sans quoi aucun rejeu deja cuit ne porterait ni les
// teleportations, ni les emissions recuperees, ni les impulsions. CE QUE LA VERSION NE PORTE
// PAS : la POSITION de la faille AVANT le premier echange (aucune entite repliquee lisible —
// negatif mesure R1 §1-3 ; la charge du 117 ne porte que {effet, depart, arrivee}, R6 §1.4),
// rien pour les trois `spent` sans evenement 117 du corpus (hors TETE de liste ou expiration
// sans usage, non departage R1 §4.3 — la LISTE COMPLETE d'evenements est un chantier
// distinct), et AUCUN usage du REPULSEUR : huit canaux ont ete juges, le film ne
// l'enregistre pas (R8 par. 10.2, R9). Le RAPPEL des impulsions n'est etabli que sur UN film
// (5/5 au Theater) : ailleurs, c'est un plancher — le detecteur de records bipede ignore les
// masques denses, limite commune a tout le depot.
// Chronique :
// document_translocations.go, document_equipment_changes.go, document_ability_impulses.go,
// document_ability_charges.go, filmdec/equipment_recovery.go, filmdec/transloc_events.go,
// filmdec/ability_impulses.go, filmdec/ability_charges.go, filmdec/offline_filters.go.
//
// CE QUE LA VERSION 39 PORTE, ET CE QU'ELLE REFUSE — CINQ APPORTS EN UNE SEULE MONTÉE. Deux
// chantiers montaient le schéma en même temps sans qu'aucun de leurs numéros n'ait jamais cuit un
// artefact : les VÉHICULES (branche `feat/v75-vehicules-sons`, lots V0 à V12, qui avait numéroté
// ses trois apports 29, 30 et 31 sur une base antérieure) et l'ARMEMENT DE LA BOMBE EN ONE BOMB
// (branche `wt/assaut-stats`, qui avait numéroté le sien 39 sur le 38). Les deux arrivent ici
// POSÉS SUR LE 38 et fondus en UNE seule montée (décisions D3 et D13 du plan d'intégration,
// 2026-09-05). La reprise du backfill se faisant par SchemaVersion, un artefact 38 doit se lire
// « à re-cuire » : il ne porte AUCUN véhicule, aucun tir au volant, aucune visée d'occupant, aucune
// statistique d'objectif d'Assaut, et aucun compte à rebours de bombe sur les matchs One Bomb.
//
// (1) LES VÉHICULES (`vehicles`) : la vie de chaque véhicule `ti=40` du match — naissance
// (position exacte du record de création), identité de châssis (`MPPWord32`, stable inter-build)
// résolue en famille de sprite, trajectoire échantillonnée sur la grille du document avec son
// CAP, et les ÉPISODES D'OCCUPATION (qui est à bord, de quand à quand, sur quel siège).
// NIVEAU DE PREUVE, INÉGAL ET IL DOIT SE LIRE ICI. Ce qui est SÛR : les positions (la grammaire
// bipède rend 99,4-100 % de pas sous 35 m/s sur la bande `ti=40`, contre 21-42 % pour celle des
// objets du monde) ; le CAP par la vélocité `i1` (écart médian 1,7-2,1 deg au déplacement sur
// 4 films, R = 0,992-0,997, témoin par mélange 51-88 deg) ; l'identité `MPPWord32` (constance
// 100 % par vie, et 5 valeurs sur 7 survivent au changement de build ET de carte). Ce qui est
// PARTIEL : l'occupation, dont la primitive n'attribue que 15,6-21,1 % des vies (mais à x20-x30
// le hasard, témoin fantôme NUL) ; et la table de familles, dont la couverture est publiée
// châssis par châssis (`coverage.vehicles.unknownChassis`).
// CE QUE LA VERSION REFUSE, mesure à l'appui (V3_DESTRUCTION_DATEE_2026-09-02, 460 vies /
// 12 films / 8 gates) : la DESTRUCTION datée. Zéro occupant à bord à la fin serrée du flux, mort
// à bord ANTI-corrélée (3,8 % contre 21,3 % au témoin), véhicule qui réplique encore 13 à 36 s
// après avoir été quitté. `VehicleTrack.End` vaut `unknown`, et la disparition du sprite ne dit
// PAS que le véhicule a explosé. Détail : internal/analysis/replay/document_vehicles.go.
//
// (2) LES TIRS DES JOUEURS EMBARQUÉS. Un occupant attaché cesse de répliquer la position de son
// bipède (primitive V1a.4) : la porte des tirs, qui pose chaque tir sur la position du BIPÈDE à
// cet instant, écartait donc TOUS les tirs tirés depuis un véhicule sous la cause « sans slot » —
// mesuré sur `0d76e8f1` : 1 166 tirs publiés, 12 épisodes d'occupation, et ZÉRO tir pendant un
// épisode. Une seconde porte (`vehicle_shots.go`) reprend ces orphelins et leur donne la position
// INTERPOLÉE du véhicule occupé ; les tirs ainsi posés portent le marqueur `shots[].v` (le slot
// du véhicule), sans quoi le client chercherait un pion qui n'existe pas à cet instant.
// CE QUE LE CRITÈRE EST, ET CE QU'IL N'EST PAS. Il est l'IDENTITÉ, pas la géométrie : le record
// de tir porte son tireur (`FilmIndex`, écrit dans le film), l'épisode d'occupation porte son
// occupant, et l'instant les recoupe. Aucun critère de distance n'était possible — le record de
// tir ne porte AUCUNE position monde, il n'y a rien à comparer au véhicule. Le témoin est donc
// temporel : les mêmes tirs contre les mêmes épisodes décalés de 60 s.
// CE QUE LA VERSION REFUSE : les tirs AMBIGUS (deux slots du même joueur répliquant tous deux une
// position). Leur signature est celle d'un joueur qui n'est PAS embarqué ; les reprendre
// mélangerait un défaut du pont slot -> joueur avec un embarquement.
//
// (3) LA VISÉE DE CHAQUE OCCUPANT DE VÉHICULE (`vehicles[].rides[].aim`). Un occupant attaché
// cesse de répliquer sa POSITION — c'est la primitive du « trou » —, et le dépôt en concluait
// qu'il ne répliquait plus RIEN : le cône du conducteur était donc dessiné au CAP DU CHÂSSIS, et
// l'artilleur comme le passager n'avaient aucun cône. Le lot V11 (2026-09-03) a montré que la
// faute était dans le DÉTECTEUR : `ScanBipedRecords` exige un `i0` absolu et un masque commençant
// par 0, quand la forme la plus fréquente de la bande bipède est `i21,i25` — un record de VISÉE
// SANS POSITION.
// NIVEAU DE PREUVE. PRÉSENCE : 4 832 à 24 050 lectures par film (5 films), 46,5 à 231,2 par slot
// bipède, contre 0,2 à 0,9 par slot sur une bande FANTÔME de même cardinalité — x155 à x925.
// JUSTESSE : appariée à la lecture `i21` AVEC position du même slot à moins de 200 ms, l'écart
// médian de cap vaut 0,2 à 0,5 deg (R 0,979-0,989), témoin par mélange déterministe 75,7 à
// 93,7 deg (R 0,011-0,134) ; la référence est le champ déjà publié `Point.H`, lui-même validé par
// l'oracle du kill. COUVERTURE : sur les 35 épisodes d'occupation ATTESTÉS par la sortie,
// 35 / 35 (100 %) portent au moins une visée à bord, à 5 à 46 lectures par seconde, quand le même
// épisode porte 0 ou 1 lecture `i21` avec position. UTILITÉ : la visée N'EST PAS le cap du
// châssis — écart médian 15,7 à 21,8 deg, q3 39,6 à 52,9 deg, un tiers des instants au-delà de
// 30 deg.
// CE QUE LA VERSION REFUSE, et c'est une réfutation mesurée avec témoin (V11 § 3) : l'ORIENTATION
// DE LA TOURELLE en tant qu'objet. L'entité tourelle ne réplique RIEN dans le flux delta —
// 139,6 et 85,5 en-têtes par slot contre 86,3 et 194,2 sur une bande FANTÔME, histogramme de
// formes de masque PLAT, et les slots MUETS non-tourelle rendent le même chiffre. Les composants
// `i31` (auto-turret-aiming-vector), `i41` et `i42` (seats-override pitch/yaw) de `ti=40` ne sont
// pas seulement non portés : ils ne sont JAMAIS émis. Le cône de l'artilleur ne vient donc pas de
// la tourelle, il vient de L'HOMME qui la tient — et c'est la même mesure que celle du
// conducteur. Détail : internal/analysis/replay/vehicle_rides_aim.go.
//
// (4) L'ARMEMENT DE LA BOMBE EN ONE BOMB. Aucun champ neuf : ce qui change est le CONTENU d'un
// calque existant (`bombArmings`), et cela suffit à imposer la montée pour la raison exacte des
// montées v14/v22/v25/v37 — sans elle, aucun rejeu One Bomb déjà cuit ne porterait jamais son
// compte à rebours.
// CE QUI A CHANGÉ. Le calque `bombArmings` était gouverné par une garde de mode DOUBLE, dont la
// première écartait One Bomb PAR SON NOM : sous la lecture SIMPLE (montée contiguë, mèche fixe de
// 4,93 s) le protocole du 2026-09-01 y avait RÉFUTÉ le signal (CV 0,725). La lecture « mèche
// pausable » du même jour l'explique (9/9 explosions portées, médiane 16,18 s, CV 0,017, 0/1000
// tirages nuls) et elle est désormais EN PRODUCTION : segments contigus, armement = segment qui
// finit à son sommet plein, tenue de désarmement qui SUSPEND la mèche, et MÈCHE MESURÉE SUR LE
// FILM au lieu d'une constante unique. Les témoins ne bougent pas (Neutral Bomb 13/13, Husky Raid
// 4/4, au chiffre près) ; ce qui bouge est qu'une variante entière d'Assaut publie enfin son
// compte à rebours. La garde qui reste est la confrontation locale TOUT-OU-RIEN, en deux branches
// (couverture des explosions, puis dispersion des délais corrigés) : deux films One Bomb du
// corpus sont RETENUS par elle, et c'est voulu.
// Chronique : filmdec/navpoint_radial_segments.go, replay/bomb_armings.go, replaybuild/zones.go.
//
// (5) LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT (`bombStats`) ET SES FAITS DATÉS
// (`bombEvents`). Deux champs neufs, posés SUR LA MÊME MONTÉE 39 et non sur une 40 : le 39
// n'a jamais cuit un artefact hors répertoires de test (contrairement au 38, cf. la politique
// du lot P5 ci-dessus), et cette intégration est le commit qui le met au monde — deux montées
// pour un numéro jamais servi marqueraient « à re-cuire » un parc qui l'est déjà.
// CE QUE C'EST : `bomb_detonations`, `bomb_arms`, `bomb_grabs`,
// `time_as_bomb_carrier_seconds` et `bomb_carriers_killed`, par joueur — les statistiques que
// l'API 343 NE PUBLIE PAS pour ce mode (la famille `BombStats` du moteur est de la TÉLÉMÉTRIE
// Bond, jamais répliquée dans le film : mesure Ghidra du 2026-09-04, cause UNIQUE du silence
// des deux côtés). Elles sont reconstruites de sources déjà décodées, et chaque champ est un
// POINTEUR : `null` dit « source non mesurée », `0` dit « mesuré à zéro ».
// LES CINQ SONT MESURÉES, `bomb_carriers_killed` COMPRIS (lot G.6, 2026-09-05). Cette entrée a
// porté l'inverse — « aujourd'hui `null` PARTOUT, la paire tueur/victime n'existe pas dans la
// chaîne de cuisson » — et le motif était faux sur son second membre : l'horloge ne manquait pas
// (`killsource.Kill.TimeMS` EST celle du fil des morts), seule la VICTIME n'était pas résolue,
// et `replaybuild.killRefs` la résout désormais dans la MÊME passe. Le champ voyage par
// `Options.MatchKills` ; `Read=false` (source non lue) le laisse absent chez TOUS les joueurs —
// « on n'a pas regardé », jamais un zéro qui se lirait comme une mesure.
// Détail : internal/analysis/replay/bomb_stats.go et bomb_stats_document.go.
//
// v40 (2026-09-06) : LE PONT D'IDENTITÉ EST COMPLÉTÉ PAR LE TRIPLET, et la version monte pour
// la SEULE raison qui fait monter une version dans ce fichier — la reprise du backfill se lit
// par `SchemaVersion`. AUCUN CHAMP N'EST AJOUTÉ : c'est le CONTENU de `objectives` qui change.
//
// Ce qui a été réparé : depuis `d173b1a8c` (2026-08-28), le calque des actions d'objectif
// résolvait l'identité slot -> joueur par les seuls INSTANTS DE MORT, qui en exigent trois
// (`objectiveevents.deathInstantMin`). Un joueur qui meurt moins de trois fois — c'est-à-dire
// le meilleur du match, celui qui porte le drapeau — n'était plus nommé, et ses actions
// disparaissaient du document. Mesuré sur `c0a82e88` : 17 actions avant la bascule, 12 après,
// les DEUX seules actions de famille `flag` du match perdues avec leur auteur (7 frags,
// 2 morts). `RoundIdentity.CompletedByLines` complète l'identité par le pont par TRIPLET sur
// les seuls slots restés anonymes, et seulement sur un film mono-manche : 23 actions, 7 joueurs
// pontés sur 7, chacun publiant exactement sa ligne de la feuille de match.
//
// POURQUOI LA VERSION MONTE ALORS QUE LA FORME NE BOUGE PAS. Un artefact au schéma 39 cuit
// AVANT ce correctif peut manquer des actions d'objectif sur un match mono-manche, et rien dans
// sa forme ne le dit. Sans montée, `backfill-replay` le SAUTERAIT (« un artefact présent qui
// porte la version de schéma courante est sauté ») et il garderait définitivement son calque
// appauvri. La règle du dépôt, écrite aux montées v3, v4, v5, v14, v22, v25 et 39, est
// qu'« un artefact vN doit se voir comme À RE-CUIRE, pas comme à jour » — c'est elle qui
// s'applique, et non l'exception du lot P5 (schéma 38 maintenu), qui ne valait que parce
// qu'AUCUN artefact 38 n'existait alors hors témoins de gate. Ici la reprise doit re-cuire tout
// artefact < 40.
// Détail : internal/analysis/objectiveevents/slotidentity_rounds.go (CompletedByLines) et
// .ai/V7.5/v2/INSTRUCTION_CTF_DRAPEAUX.md section 9.
//
// v41 (2026-09-06) : TROIS CALQUES RATTRAPENT « UNE TRACK = UNE VIE ». Aucun champ n'est
// ajouté, aucune forme ne change : trois calques publient de nouveau ce qu'ils publiaient
// avant le schéma 36, et la version monte pour la raison des montées v39/v40 — la reprise du
// backfill se lit par `SchemaVersion`, et un artefact 36 à 40 est APPAUVRI sans que rien dans
// sa forme ne le dise.
//
// LE DÉFAUT, UN SEUL, EN TROIS ENDROITS. `48cf4905d` (2026-09-02) a découpé les pistes à
// `lifeGapUS` : un slot recyclé publie désormais PLUSIEURS pistes. Trois consommateurs
// supposaient encore « un slot = une piste » et ne retenaient que la DERNIÈRE, jetant en
// silence tout ce qui appartenait aux vies antérieures :
//
//	pistes      `nameClosedLives` cherchait « l'unique vie anonyme du slot » pour y poser
//	            l'identité d'une fermeture, et s'abstenait dès qu'il y en avait deux. Le
//	            document publiait alors les TIRS d'un slot dont la piste restait sans nom.
//	            `145908d1` : 53 slots au pont, 51 pistes nommées, 29 tirs orphelins — 24
//	            identités distinctes retombées à 23. Les fermetures DÉSIGNENT une vie
//	            (`closureReport.closedLife`), c'est elle qui est nommée.
//	grappin     `buildGrappleLines` bornait chaque traction à la dernière vie du slot, ce qui
//	            la rendait vide (`t1 <= t0`) puis la supprimait. `879a4dba` : 23 accroches LUES
//	            dans le film, 23 tractions publiées au schéma 34, 15 dès le 36 — `heavyReads`
//	            inchangé à 23, la lecture n'avait donc rien perdu. `084a804d` : 71 -> 61 -> 71.
//	            `coverage.grapple.pullLives` compte désormais des VIES, non des slots.
//	épisodes    `trackFrameWindows` n'indexait qu'une fenêtre par slot : un épisode de camo ou
//	            de surbouclier d'une vie antérieure tombait hors fenêtre et disparaissait.
//	            `82f29378` retrouve son unique épisode de surbouclier, `084a804d` ses 15
//	            épisodes de camouflage sur 9 vies (13 sur 8 aux schémas 36 à 40).
//
// CE QUE LA VERSION NE PRÉTEND PAS RÉPARER : un épisode ancré sur un point de trajectoire
// ABERRANT reste écarté, et c'est voulu — `13d92593` perdait un épisode de surbouclier de
// durée nulle (t0 = t1 = 3603) posé sur le seul point qui plaçait le joueur à 267 u de sa
// position précédente, celui-là même qui donnait au document des bornes de scène fausses
// (minX -227,27 -> -18,57). L'assainissement des trajectoires l'a supprimé : c'est un gain,
// il n'est pas revenu, il ne doit pas revenir.
// Détail : internal/analysis/replay/{closures.go, closures_respawn.go, owners.go,
// grapple_lines.go, equipment_episodes.go} et .ai/V7.5/v2/INSTRUCTION_REGRESSIONS_2_4.md.
//
// v42 (2026-09-06) : LE DRAPEAU A ENFIN DES PORTEURS. Aucun champ n'est ajouté, aucune clé ne
// bouge — c'est le CONTENU de `flagCarries` qui change, exactement comme aux montées v14 et v15
// du même calque, et la version monte pour la raison de toujours : la reprise du backfill se lit
// par `SchemaVersion`, et un artefact 41 est AMPUTÉ de portages sans que rien dans sa forme ne le
// dise.
//
// LE PLAFOND, ET IL FRAPPAIT LES MEILLEURS JOUEURS. Le calque nommait son porteur par le pont
// d'identité PAR MORTS (`objectiveevents.ResolveRoundIdentity`), qui exige `deathInstantMin` = 3
// instants de mort coïncidents pour attribuer un slot d'entité. Un joueur qui MEURT MOINS DE
// TROIS FOIS lui échappe PAR CONSTRUCTION — et ce sont, par définition, ceux qui portent le
// drapeau. Leurs prises étaient comptées `coverage.flagCarries.noBridge` et AUCUN intervalle
// n'était publié pour elles : le drapeau restait dessiné à sa base pendant qu'un joueur le
// portait. Le pont par TRIPLET, lui, les nomme — il apparie les totaux (frags, morts,
// assistances) du statborg aux lignes de match — mais il exige la base, que `analysis/replay`
// n'ouvre pas.
//
// LA CORRECTION EST UN CÂBLAGE, PAS UNE RÈGLE NEUVE. Le pont COMPLÉTÉ
// (`RoundIdentity.CompletedByLines`, mono-manche, compléter sans jamais contredire, aucun xuid
// deux fois) existait déjà depuis le schéma 40 : il servait les ACTIONS d'objectif, résolu dans
// `replaybuild` — la couche qui reçoit les faits du match. Il descend désormais jusqu'au calque
// du drapeau par `replay.FlagInput.Identity`, et les deux calques partagent LE MÊME pont, résolu
// une seule fois par cuisson. `analysis/replay` ne voit toujours AUCUN fait de match : il reçoit
// une table slot -> xuid. Sans lignes de match (CLI hors ligne, ouvrier distant), le champ reste
// à zéro, le paquet résout comme avant, et l'artefact est identique — la propriété « publiable
// hors ligne » est intacte.
//
// MESURE, TROIS FILMS MONO-MANCHE ET UN MULTI-MANCHE (2026-09-06) :
//
//	c0a82e88  3 prises, 0 portage -> 1 portage. Le porteur publié est 2535463878425995
//	          (7 frags, 2 morts) — le MÊME xuid, aux MÊMES instants, que les deux actions
//	          `flag_steals` / `flag_captures` du calque `objectives`, par une chaîne
//	          indépendante. Le marqueur de portage des images-clés le confirme (1/1). Les
//	          2 prises restantes sont sur un slot 12 AGRÉGÉ, que le triplet refuse de nommer.
//	e94163af  33 prises, 16 portages -> 33, `noBridge` 17 -> 0, marqueur confirmé 6/7.
//	51101d1d  12 prises, 10 portages -> 11, `noBridge` 1 -> 0, marqueur confirmé 4/4.
//	fb1a1a72  3 manches : artefact IDENTIQUE OCTET POUR OCTET. La garde mono-manche du
//	          triplet s'abstient, et c'est voulu (un slot y est réattribué d'une manche à
//	          l'autre).
//
// CE QUE LA VERSION EMPORTE AVEC ELLE, ET CE N'EST PAS UNE PERTE. `coverage.flagCarries.
// homeByObject` baisse (4 -> 0 sur `e94163af`) et un état `home` disparaît : la RENTRÉE PAR
// L'OBJET n'agit que sur un drapeau AU SOL (`applyFlagHomecoming`), et ces drapeaux-là étaient
// « au sol » uniquement parce que le portage qui les tenait n'était pas publié. La règle de
// retour du mode (`flagReturnZone`), elle, APPARAÎT sur les matchs qui ne publiaient aucun
// portage : elle n'est servie que s'il y a un drapeau à entourer.
// Détail : internal/analysis/replay/build_objectives_live.go (`FlagInput.Identity`,
// `flagIdentityOf`), internal/replaybuild/matchfacts.go (`pontParManche`),
// internal/analysis/objectiveevents/slotidentity_rounds.go (`CompletedByLines`) et
// .ai/V7.5/v2/FLAGCARRIES_COMPLEMENT_2026-09-06.md.
//
// v43 (2026-09-06) : UNE VIE ANONYME N'EST PAS UNE ABSENCE. Aucun champ n'est ajouté ; c'est le
// CONTENU de `skullCarries` — et, latent, de `bombCarries` — qui change. Le gate de présence
// (`af89b091b`, 2026-08-30) écartait un portage dont le porteur n'avait aucune vie NOMMÉE sur
// l'intervalle, et rognait celui qui en débordait. Or le pont d'identité laisse des vies
// ANONYMES (18 slots sur 160 sur `d9781168` — 142 portent au moins une vie nommée, ces 18-là
// aucune) : « aucune vie nommée » n'y veut pas dire « absent », mais « on ne sait pas ». Le gate
// ne s'applique donc plus quand une vie anonyme recouvre le portage — il ne rejette que ce que
// les pistes publiées DÉMENTENT.
//
//	mesure   En Oddball le score EST le temps de portage : chaîne de contrôle INDÉPENDANTE du
//	         film. `d9781168`, feuille de match 191 s / 196 s par équipe ; artefact du parc au
//	         schéma 23 : 172,5 s / 158,8 s ; artefact au schéma 41 : 60,1 s / 147,4 s. Le gate
//	         écartait 6 portages sur 36 (32,6 s) et en rognait 4 (91,2 s) — il éloignait
//	         l'artefact de la vérité au lieu de l'en rapprocher. Aussi `51ebbc0f` (7 sur 14) et
//	         `24dbb67d` (3 sur 20).
//	rendu    Un portage écarté ne rendait pas le crâne visible : il faisait retomber
//	         `skullPresenceAt` (web) sur la règle du repos, qui pose le crâne à sa dernière
//	         position connue PENDANT qu'un joueur court avec — le fantôme même que l'invariant
//	         d'`objectiveObjectsLayer` interdit. Le calque du porteur, lui, ne dessine déjà rien
//	         sans position : le symptôme d'origine (« icône absente ») était honnête.
//
// La version monte pour la raison des montées v39/v40/v41/v42 : un artefact 23 à 42 est appauvri
// sans que sa forme le dise, et `backfill-replay` saute un artefact à la version courante.
// Détail : internal/analysis/replay/skull_carries.go (carrierPresence.gate) et
// .ai/V7.5/v2/INSTRUCTION_RESIDUS_2026-09-06.md.
//
// v44 (2026-09-06) : LA MANCHE DÉCLARÉE D'UN ENREGISTREMENT EST CONFRONTÉE AU TEMPS. Aucun champ
// n'est ajouté ; c'est le CONTENU de `scoreTimeline` (les quatre compteurs par joueur et les
// courbes d'équipe) et des actions d'objectif qui change, sur les films À PLUSIEURS MANCHES
// SEULEMENT. La manche d'un enregistrement est lue dans deux en-têtes de 5 bits ; l'assertion
// d'en-tête étant relâchée, un résidu de faux positifs porte une manche quelconque et des
// valeurs arbitraires. La découpe par manche prenait ce numéro pour argent comptant, et la plus
// longue sous-suite non décroissante ne pouvait pas l'écarter — une valeur mal lue mais PLUS
// GRANDE prolonge la suite au lieu de la rompre. Les manches se jouant dans l'ordre, un
// enregistrement daté hors de l'intervalle de la manche qu'il déclare est désormais écarté.
//
//	mesure   `51ebbc0f` (Oddball, 2 manches) : un enregistrement daté 316 777 ms — 57 s APRÈS le
//	         début de la manche 1 — déclarait la manche 0 avec 60 assistances ; ce 60 devenait le
//	         décalage de la manche 1 et le document publiait 63 assistances pour un joueur qui en
//	         a 5 à la feuille (il coûtait aussi son frag de manche 0 : 0 -> 1). MÊME
//	         enregistrement sur `d9781168` (slot 12, `comp 3 A = 60`) : assistances 69 -> 11,
//	         soit EXACTEMENT la feuille. Le même bruit nommait 58 vols de drapeau sur `51ebbc0f`
//	         et 994 sur `24dbb67d` — deux films d'Oddball, donc sans drapeau : tombent à 0.
//	portée   Quinze témoins re-cuits : les onze autres sont IDENTIQUES À L'OCTET, dont les trois
//	         films dont l'étiquetage de manche ne suit pas l'horloge (`fb1a1a72`, `72b0a25e`,
//	         `a4083bd2`) où aucune borne n'est posable, et tous les films mono-manche — qui n'ont
//	         par construction aucune borne.
//
// La version monte pour la raison des montées v39 à v43 : un artefact 1 à 43 d'un film
// multi-manche porte des compteurs gonflés sans que sa forme le dise, et `backfill-replay` saute
// un artefact à la version courante. Détail :
// internal/analysis/objectiveevents/round_bounds.go et .ai/V7.5/v2/MANCHES_COMPTEURS_2026-09-06.md.
//
// v45 (2026-09-06) : UN TROU DE RÉPLICATION N'AMPUTE PLUS UNE DURÉE MESURÉE. Aucun champ n'est
// ajouté ; c'est le CONTENU d'`equipmentEpisodes` et de `flagCarries` qui change. Même cause
// qu'aux v41 et v43 — le découpage « une track = une vie » du v36 —, mais sur deux consommateurs
// de plus, et sur une grandeur qu'aucun COMPTAGE ne voit : la DURÉE. Les deux faits sortent du
// nouvel axe « somme des durées » du comparateur d'artefacts.
//
//	épisodes  L'état actif se lit PAR SLOT ; ses deux bornes sont des transitions LUES. Le
//	          bornage à la vie de recouvrement MAXIMAL jetait la part couverte par une autre vie
//	          du même slot, dont l'instant d'ACTIVATION — et pour le camouflage, l'état actif est
//	          ce qui PROVOQUE le trou de réplication (porteur invisible et immobile). `084a804d`
//	          slot 620 : camo lu [3105..3672] (568 frames), publié [3173..3672] (500).
//	          `spanFor` borne désormais à l'UNION des vies recouvertes.
//	drapeaux  `tracksByXUID` n'indexait que les pistes NOMMÉES : une prise que seule la vie
//	          ANONYME du porteur recouvre sortait `NoTrack`. `bcb6d393` : 9 prises sur 16
//	          perdues, `carries` 16 -> 7. L'identité vient du PONT canonique (`ResolveSlotXUID`),
//	          jamais d'une déduction locale.
//
// La version monte pour la raison des montées v39 à v44 : un artefact 36 à 44 est appauvri sans
// que sa forme le dise. Détail : internal/analysis/replay/{equipment_episodes.go, flag_carries.go}
// et .ai/V7.5/v2/INSTRUCTION_DUREES_2026-09-06.md.
//
// v46 (2026-09-06) : UN JOUEUR NE PORTE PLUS SON PROPRE DRAPEAU. Aucun champ n'est ajouté ;
// c'est le CONTENU de `flagCarries` qui change — l'attribution d'un portage à l'un des deux
// drapeaux, et donc la CAPTURE qui le ferme.
//
//	l'ordre   `assignFlags` tenait « où git chaque drapeau » en parcourant les PRISES : la
//	          position de LÂCHER d'un portage y était inscrite dès son attribution, donc avant
//	          d'avoir eu lieu. Deux portages qui se recouvrent suffisent. `bcb6d393` : le portage
//	          ouvert à 171 941 ms ne se ferme qu'à 325 913 ms, et sa position de lâcher chassait
//	          du sol celle que la prise suivante venait chercher à 0 m. Le parcours se fait
//	          désormais par ÉVÉNEMENTS DATÉS (une fin pose, une prise enlève).
//	la règle   Le repli sur le socle le plus proche est juste pour un VOL (il se fait à un socle)
//	          et faux pour une PRISE, qui se fait là où l'objet est tombé — souvent près du socle
//	          ADVERSE, c'est-à-dire de celui du porteur. Une prise que rien ne rattache au sol va
//	          désormais au drapeau DÉJÀ EN JEU quand il est le seul ; à deux, la règle se tait.
//
// Mesure `bcb6d393` (CTF:Arena 3-0, quatre porteurs de l'équipe 0) : 15 portages sur le drapeau
// de l'équipe 1 et 1 sur celui de l'équipe 0 -> 16 sur 0, et les TROIS captures reviennent sur
// le drapeau adverse (une y était publiée sur le drapeau du camp qui marquait). La version monte
// pour la raison des montées v39 à v45 : un artefact 45 attribue un portage — et sa capture — au
// mauvais drapeau sans que sa forme le dise, et `backfill-replay` saute un artefact à la version
// courante. Détail : internal/analysis/replay/flag_assign.go et
// .ai/V7.5/v2/INSTRUCTION_DRAPEAUX_2026-09-06.md.
//
// v47 (2026-09-07) : AUCUNE VIE PUBLIEE NE RESTE SANS NOM, ET LES LECTEURS LISENT LA PISTE SOUS
// L'IDENTITE RESOLUE DE SON SLOT. NEUF champs de couverture s'ajoutent — cinq au pont
// (`bridge.namedByPreviousLife/namedByNextLife/namedBySlotBridge/unnamedLives/
// unnamedLivesContested`), trois aux zones (`zones.noPosition/outside/ambiguousZone`) et un au
// drapeau (`flagCarries.ambiguousSlot`, apporte par le lot des DUREES et integre ici) ; le reste
// est un changement de CONTENU.
//
// DECISION PRODUIT (utilisateur, 2026-09-07) : « les vies anonymes n'existent pas ; une vie est
// un humain ou un bot, point ». Une piste publiee sans identite n'est PAS une categorie de
// donnee, c'est un DEFAUT DE NOMMAGE du pont — a reparer a la source, jamais a afficher.
//
//	drapeau       Le repli d'une vie SANS NOM sur le joueur du pont est REFUSE quand les vies
//	              publiees du slot le contredisent, et le refus se COMPTE
//	              (`flagCarries.ambiguousSlot`) au lieu de disparaitre en silence. Deux gardes
//	              cohabitent, sur deux matieres distinctes — le pont EPURE (vies decoupees) et
//	              `slotAmbigu` (vies publiees) — et le compteur porte les deux populations.
//	              flag_carrier_tracks.go.
//	frontieres    Un slot que DEUX joueurs nommes se partagent est desormais MARQUE
//	              (`OwnerReport.SlotAmbiguous`) et non plus seulement compte : le pont y garde le
//	              PREMIER occupant, un choix par l'ORDRE DES VIES. Le repli par le pont s'abstient
//	              sur ces slots, et une vie qui tombe ENTRE deux occupants differents est REFUSEE
//	              et comptee (`bridge.unnamedLivesContested`) plutot que tranchee au hasard.
//	              Temoin : `084a804d` slot 734. owners.go, unnamed_lives.go, published_tracks.go.
//	nommage       Apres les quatre passes existantes (fil des morts, fermetures, sieges de bot,
//	              relais), une passe finale nomme ce qui reste par l'OCCUPATION DU SLOT DANS LE
//	              TEMPS : vie nommee du meme slot qui PRECEDE, sinon celle qui SUIT, sinon le
//	              pont canonique. Le residu se compte (`unnamedLives`) et s'alarme (slog.Error) ;
//	              il ne se devine jamais. unnamed_lives.go.
//	objectifs     Le denominateur comptait les seuls RESCAPES du pont d'identite : `noSlot`
//	              valait 0 sur les 111 artefacts du parc, sans exception. Et le filtre « piste
//	              publiee » cadencait sur le seul nom LU — un joueur dont aucune vie n'est nommee
//	              alors que le pont nomme son slot perdait TOUTES ses actions. Defaut DEMONTRE
//	              (mutation) mais NON CHIFFRE sur le parc : les 35 actions de `3372e7eb` que la
//	              premiere redaction citait viennent de deux joueurs SANS AUCUNE piste, que le
//	              pont ne peut pas atteindre (revue VIES-R1, C3). objectives.go,
//	              published_tracks.go, coverage.go (`warnIfLossy` voit `Unpublished`).
//	zones         `samplesByXUID` n'indexait que les pistes NOMMEES : une capture couverte par
//	              une vie non resolue sortait `NoPosition`, ne votait plus, et quand plus aucune
//	              n'etait attribuee le calque `zoneStates` ENTIER disparaissait. `696a9d7c` 11
//	              captures perdues sur 77, `7344d24f` 12 sur 71, `af13e2b2` 5 sur 19. La cause
//	              d'une capture perdue est desormais PUBLIEE. zone_attribution.go, zone_states.go.
//	fermetures    Les deux garde-fous jugeaient sur le nuage du slot ENTIER la ou le code venait
//	              de DESIGNER une vie : sur un slot a deux vies eloignees, l intervalle teste
//	              couvrait le trou qui les separe. closures.go, closures_respawn.go.
//	vehicules     L occupant d une ride venait de `SlotXUID`, identite UNIQUE par slot pour tout
//	              le match : sur un slot recycle, l episode sortait avec le PREMIER occupant, et
//	              c'est lui qui donne sa COULEUR au vehicule. `OwnerReport.xuidAt(slot, instant)`.
//	bots          Deux bots d un meme siege s ecrasaient dans une `map[siege]nom` : le dernier
//	              balaye gagnait, sans rapport avec la chronologie du remplacement. identity.go.
//	portages      Le rognage de `carrierPresence.gate` tronquait a la vie de recouvrement
//	              MAXIMAL un portage qu un trou de replication coupe en deux — meme cause que
//	              celle qui a produit `spanFor` au v45. `unionOverlap`. skull_carries.go.
//
// La version monte pour la raison des montees v39 a v45 : un artefact 36 a 46 est appauvri sans
// que sa forme le dise, et `backfill-replay` saute un artefact a la version courante. 44, 45 et
// 46 sont deja integres (manches, durees, DRAPEAUX) : la chronique ci-dessus les porte dans
// l'ordre, et aucun numero n'est plus reserve.
// Detail : .ai/V7.5/v2/VIES_ANONYMES_2026-09-06.md.
//
// v48 (2026-09-07) : LE PONT D'IDENTITE CESSE D'ETRE MUET SUR LES FILMS DONT LA PREMIERE FIN DE
// VIE TOMBE APRES LA PREMIERE MINUTE DU MATCH. Deux champs de couverture s'ajoutent
// (`bridge.deathOffsetMatched/deathOffsetRunnerUp`) ; le reste est un changement de CONTENU.
//
//	calage      `bestDeathOffset` balayait le decalage fil des morts <-> film depuis
//	            `min(fins de vie) - 60 000`. La grandeur que cette borne suppose petite est donc
//	            l'instant de match auquel correspond LA PLUS PRECOCE DES FINS DE VIE DU FILM —
//	            pas la premiere mort, qui lui est seulement correlee : `d9781168` a sa premiere
//	            fin de vie a 18,4 s et sa premiere mort a 53,6 s, et `43716616` reste sain avec
//	            une premiere mort a 60,4 s. Une vie se termine aussi SANS mort (fin de film, fin
//	            de manche, trou de replication). Le fil est date depuis le debut du match, or la
//	            partie ne commence pas a t = 0 : au-dela de 60 s, le vrai calage tombait SOUS la
//	            borne et l'optimiseur se rabattait sur un pic de bruit. La plage est desormais
//	            celle des DONNEES (`[min(fins) - max(morts), max(fins) - min(morts)]`), localisee
//	            par un VOTE puis affinee au pas de 10 ms sur la grille d'avant. lives.go.
//	            Mesure : CINQ films du parc sur 106, et ce sont EXACTEMENT les cinq dont
//	            l'origine du fil n'etait pas publiee (`resolveOriginMs` prend ce calage pour
//	            temoin). Vies nommees : `51ebbc0f` 9 -> 71 / 87, `fb1a1a72` 17 -> 140 / 147,
//	            `4f77afc1` 44 -> 192 / 375, `11de8353` 31 -> 150 / 246, `06dfe6d9` 37 -> 225 /
//	            291. Les films deja bien cales retiennent le MEME entier de calage.
//	marge       Le vote est une HEURISTIQUE de localisation : il pourrait designer un panier de
//	            bruit et rendre un calage faux EN SILENCE. Trois candidats sont donc affines et
//	            mesures, et la paire (retenu, meilleur des autres) est PUBLIEE — un calage vrai
//	            ecrase ses concurrents (71 contre 8 sur `51ebbc0f`, 157 contre 15 sur
//	            `d9781168`). Sous `deathOffsetMargeMin`, un `slog.Warn` le dit. coverage_bridge.go.
//	roster      Le roster de lecture de l'index de joueur venait du SEUL fil des morts : un
//	            joueur qui ne meurt jamais n'y figure pas, donc n'a pas d'index, donc aucune de
//	            ses pistes n'est rattachable et il manque au roster publie. La feuille de match
//	            le COMPLETE quand l'appelant la fournit (`Options.RosterXUIDs`, vide = comportement
//	            d'avant). Mesure : `3372e7eb` publiait 6 joueurs pour 8, les deux manquants a
//	            0 mort. player_index.go, replaybuild/matchfacts.go.
//
// La version monte pour la meme raison qu'aux montees v39 a v47 : un artefact < 48 porte des
// pistes non nommees et, sur cinq films, un calque d'objectifs et une courbe de score non
// recales faute d'origine. Detail : .ai/V7.5/v2/PONT_MUET_2026-09-07.md.
//
// v49 (2026-09-08, lot M1b — décision utilisateur ferme, « corriger le décalage ») : le
// document publie `coverage.bridge.deathOffsetMs` — le calage `DeathOffsetMS` du pont
// d'identité lui-même (`horlogeFilm = horlogeMatch + deathOffsetMs`), déjà CALCULÉ à la
// cuisson (v48 publiait sa MARGE, `deathOffsetMatched`/`deathOffsetRunnerUp`, jamais la
// valeur). Champ additif, pointeur `*int64` : absent (nil) quand le pont n'a pas été
// construit ou n'a apparié aucune mort (`OwnerReport.DeathOffsetMatches == 0`) — jamais un
// zéro qui se lirait comme un calage exact (cf. BridgeHealth.DeathOffsetMs, coverage_bridge.go).
//
//	pourquoi     le lien « voir dans le rejeu » posé depuis une cellule Tactique (lot M1,
//	             `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md`) portait un instant EXACT pour
//	             deux questions sur six (`temps`/`routes`, déjà sur l'horloge du film) et une
//	             APPROXIMATION pour les quatre autres (`morts`/`kills`/`gagne`/`isole`, sur
//	             l'horloge du MATCH) — le décalage par match n'était publié nulle part. Le
//	             web ne peut PAS le mesurer lui-même (il n'a que l'artefact, jamais le film).
//	le contrat   `TacticalContribution` publie désormais `clock` (`"match"` ou `"film"`) à
//	             côté de `instant_ms` ; le web attend le document du rejeu (qui porte
//	             l'offset) avant de convertir, et n'ouvre plus jamais un instant approché en
//	             le présentant comme exact — cf. `tactical_service_cellule.go`,
//	             `lib/replay/replayLogic.resolveTacticalReplayInstant` côté web.
//
// EXCEPTION AU CRITÈRE HABITUEL DE BUMP (assumée, cf. le commentaire de `SchemaVersion`) :
// le champ est optionnel et le web sait déjà lire son absence sans regarder la version —
// le bump sert seulement à faire recuire tout le parc < 49 en une passe
// (`backfill-replay`), pour qu'un artefact ancien cesse de répondre « calage inconnu » faute
// de recuisson déclenchée. Détail : `.ai/V7.5/v2/CHRONIQUE_49_2026-09-08.md`.

// v50 (2026-09-08, lot P2 — REGISTRE D'IDENTITÉ DES JOUEURS) : le document publie POUR LA
// PREMIÈRE FOIS sur quoi repose chaque nom qu'il sert, l'identifiant stable des bots, et il
// nomme les joueurs que le pont par morts ne pouvait pas atteindre.
//
//	ce qui change   1. `identity` — la section du registre : `players` (index de joueur ↔ xuid,
//	                   ou `bid(N.0)` pour un bot), `bipedSlots` (l'occupation d'un slot BORNÉE
//	                   en frames, une ligne PAR VIE), `statborgSlots` (le slot d'entité par
//	                   MANCHE), et `coverage` — le décompte des liens PAR PROVENANCE.
//	                2. `roster[].bid` — l'identifiant stable d'un bot, forme `bid(N.0)`.
//	                3. le NOMMAGE des vies change : un slot dont aucune vie n'est nommée, quand
//	                   il ne reste qu'un seul joueur du roster sans aucune vie, est nommé par
//	                   ELIMINATION. Des pistes jusqu'ici anonymes portent donc un xuid.
//	                4. les actions d'objectif ne sont plus JETÉES quand leur auteur n'a pas de
//	                   trajectoire publiée (`coverage.objectives.unpublished` tombe à zéro,
//	                   `attached` monte d'autant).
//
//	pourquoi        « L'INDEX EST L'INDEX » (décision utilisateur du 2026-09-07, plan v2 §0.7).
//	                Le film porte des liens DIRECTS — l'index de joueur des chunks de
//	                réplication, le `BotID` de BOT_METADATA — et le document les remplaçait par
//	                des déductions sans jamais le dire. Quinze calques et deux lecteurs hors
//	                rejeu reconstruisaient chacun leur pont, avec leurs propres gardes. Trois
//	                faits mesurés en découlaient : `3372e7eb` jetait 35 actions d'objectif sur
//	                76 ; `d9781168` publiait 19 vies sans nom, toutes sur le slot d'un joueur
//	                qui ne meurt jamais ; `51ebbc0f` perdait la manche 0 d'un joueur (écart
//	                cumulé K/D/A de 9 contre la feuille de match).
//
//	le contrat      la section est OPTIONNELLE et additive (`identity,omitempty`) ; un artefact
//	                antérieur au schéma 50 n'en porte pas, et le client doit lire son absence.
//	                `roster[].bid` est optionnel de la même façon — vide pour un humain, et vide
//	                pour un bot dont la déclaration ne portait pas d'identifiant (un `bid(0.0)`
//	                inventé joindrait deux bots distincts).
//
//	ce que le lot   les liens que le film ne porte PAS restent déduits, et ils le DISENT :
//	n'a pas fait    slot de bipède ↔ index de joueur (aucune identité dans `BipedPosition`),
//	                slot de statborg ↔ joueur, équipe d'un joueur (elle vit dans la base, pas
//	                dans le film). Leur branchement appartient au plan DÉCODEUR d'après v7.5.0
//	                (inventaire P1, colonne « exige travail décodeur »).

// v50 AMENDÉ (2026-09-08, lot P-décodeur E2 — LE LIEN DIRECT CORPS ↔ JOUEUR). Le numéro ne
// bouge pas : le lot P n'est pas fusionné, le contenu cuit change encore sous ce schéma. Ce que
// la chronique ci-dessus rangeait dans « ce que le lot n'a pas fait » — « slot de bipède ↔ index
// de joueur : aucune identité dans `BipedPosition` » — EST FAIT, et par une LECTURE.
//
//	ce qui change   1. le record de CRÉATION d'un bipède (`ti=35`) porte l'index de participant
//	                   de son propriétaire, à `+67` bits de l'en-tête NEW (`filmdec.ScanBipedCreations`).
//	                   `identity.bipedSlots[].link.source` passe de `deduit` à **`direct`**, voie
//	                   `creation_bipede` — ou `creation_bipede_propagee` pour les autres séjours
//	                   du MÊME corps, qu'une découpe à `lifeGapUS` a séparés.
//	                2. `identity.coverage.bipedSlot` gagne `direct_propage` (sous-compte de
//	                   `direct`) et `non_resolu_par_cause` : la somme des causes ÉGALE
//	                   `non_resolu`, « non résolu » ne se publie plus sans son motif.
//	                3. `coverage.bridge` gagne `concordant` / `discordant` / `bridgeNamedLives`
//	                   / `directByCreation` / `directByCreationPropagated` / `bodiesWithCreation`.
//	                   LE PONT PAR MORTS NE NOMME PLUS : il pose la cause de fin et VÉRIFIE.
//	                4. des NOMS CHANGENT, et c'est le but. L'appariement glouton du pont
//	                   départageait par l'ordre des slots quand deux vies finissent au même
//	                   instant : 13 vies sur `d9781168`, 4 sur `64e8adfa`, 10 sur `c75f33b8`
//	                   portaient le joueur que le film écrit sur une AUTRE vie.
//
//	mesuré          `identity.coverage.bipedSlot.direct` : 0 % → **100 %** sur `d9781168`,
//	                `bf15f7ab`, `64e8adfa`, `3372e7eb` ; **97,7 %** sur `c75f33b8`, dont les
//	                deux vies restantes lisent un index que `identity.players` ne publie pas
//	                (`non_resolu_par_cause.index_hors_table = 2`, verdict I0 du lot).
//	                `coverage.bridge.unnamedLives` : 15/0/7/8/14 → **0/0/0/0/1**.
//	                `coverage.shots.noSlot` : 513/12/213/289/450 → **64/12/13/10/119**.
//	                `slotCollisions` de `d9781168` : 1 → **0** — le film ne porte qu'UN corps par
//	                slot, la collision était une erreur du pont.
//	                `scoreTimeline` et `identity.statborgSlots` : **identiques octet pour octet**
//	                sur les cinq films — E2 ne touche que le pont des BIPÈDES.
//
//	le contrat      strictement ADDITIF : aucun champ retiré, aucune clé renommée. Un client qui
//	                lisait `bipedSlot.direct` lit maintenant un compte non nul ; un client qui
//	                lisait `link.method` voit deux voies de plus, à ajouter à son énumération.
//
//	ce que le lot   il ne touche NI le pont statborg (`64e8adfa` garde ses 5 couples perdus, écart
//	n'a pas fait    K/D/A 16 inchangé) NI l'équipe d'un joueur (elle vit dans la base). Détail,
//	                mesures et instruction du résidu : `.ai/V7.5/v2/RESTES_E2_2026-09-08.md`.

// v52 (2026-09-11, lot B — LES CORRECTIFS DU DÉCODEUR REPRIS DU FORK). Trois changements de
// CONTENU CUIT, indépendants l'un de l'autre, et un compteur qui s'ajoute.
//
//	ce qui change   1. UN LANCER DE GRENADE REVIENT À SON LANCEUR. `locateThrow` choisissait la
//	                   naissance de projectile la plus proche DANS LE TEMPS (±200 ms) et
//	                   départageait les simultanées par le tri, donc par X : deux joueurs qui
//	                   lancent dans la même fenêtre — un lancer sur cinq — et l'un recevait la
//	                   position du projectile de l'autre. L'auteur est résolu d'abord ; parmi
//	                   TOUTES les naissances de la fenêtre on garde celle qui est à portée de sa
//	                   main, et aucune au-delà de `grenadeAuthorRadiusM = 4` m. Sans auteur
//	                   ponté, une candidate unique reste une lecture, plusieurs sont un refus.
//	                2. `grenades[].slot` EST PUBLIÉ SUR LES DEUX BRANCHES. La branche projectile
//	                   sortait à zéro — et zéro RESSEMBLE à un slot, si bien qu'un lecteur qui
//	                   colore un lancer par son lanceur ne pouvait ni nommer personne, ni voir
//	                   l'ambiguïté.
//	                3. UN VOL DE PROJECTILE S'ARRÊTE AU PREMIER PAS IMPOSSIBLE (> 10 m en
//	                   100 ms, `projectileMaxStepM`), et n'est pas recousu. `rest` tombe à false
//	                   sur un vol coupé : il CERTIFIE une fin de vol, et un vol coupé n'a pas la
//	                   sienne. Le compte des coupures est publié (`coverage.projectiles`).
//	                4. `geometry` devient les props de LA CARTE du match. Un répertoire unique
//	                   les servait à tous les matchs : les artefacts d'une carte non extraite
//	                   sortent désormais SANS props, ce qui est la vérité.
//
//	mesuré          GRENADES, distance du lancer publié au biped de son auteur au même instant
//	                (banc `grenade_ecart_research_test.go`), avant -> après :
//	                `000d5950` Cliffhanger médiane 0,44 -> 0,44 m, PIRE CAS 14,46 -> 0,56 m,
//	                lancers au-delà de 4 m 2 -> 0 ; `0797ce72` Live Fire médiane 25,42 -> 0,00 m ;
//	                `21ece4d8` Live Fire médiane 26,69 -> 0,00 m. Sur les deux films Live Fire,
//	                la branche projectile s'effondre (84 -> 1 et 137 -> 0 lancers) parce que les
//	                naissances y sont victimes du repli de quantum ci-dessous ; le nombre de
//	                lancers publiés y est INCHANGÉ (103 et 156), ils se lisent sur le biped.
//	                Golden `000d5950` : 70 -> 69 lancers posés (dénominateur inchangé),
//	                répartition par source 65/5 -> 63/6.
//	                PROJECTILES : 947 trajectoires sur 15 735 du parc (6,0 %) portaient au moins
//	                un pas impossible, soit 4 901 pas. Golden `000d5950` : 439 -> 436
//	                trajectoires publiées, 2 732 -> 2 725 points, 3 coupures.
//	                PROPS : 382 props identiques sur les 76 artefacts, cartes confondues.
//
//	la cause du 3   N'EST PAS CORRIGÉE, ELLE EST CARACTÉRISÉE. Le saut vaut l'étendue de la carte
//	                sur un axe DIVISÉE PAR UNE PUISSANCE DE DEUX — le poids d'UN bit du champ
//	                quantifié, et non un repli de toute la plage. Sur les quatre films Live Fire
//	                (`sgh_interlock`, Y sur 12 bits) il vaut exactement la MOITIÉ de l'étendue Y
//	                (médiane 31,89 m pour 63,775 m) avec |Δx| médian 0,20 m : le bit de poids
//	                fort de Y bascule, 3 907 pas sur 4 901. Sur les cartes Forge l'axe touché est
//	                plutôt X et le bit plus bas (étendue / 2^7 majoritaire). Le chantier appartient
//	                à `filmdec` : `.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`.
//
//	le contrat      `coverage.projectiles` est ADDITIF et optionnel. `grenades[].slot` et
//	                `projectiles[].p` existaient déjà : ce sont leurs VALEURS qui changent, et
//	                c'est ce qui exige le bump — un client v51 dessine aujourd'hui des lancers
//	                posés sur le mauvais joueur et des vols en travers de la carte.
//
//	ce que le lot   il ne touche NI la déquantification (la cause du point 3), NI l'attribution
//	n'a pas fait    des props des 78 autres cartes du catalogue de bornes — une seule extraction
//	                existe, attribuée à `ridgeline` par son emprise (cf. le README du répertoire).

// v53 (2026-09-12, lot B-bis — LE BIT DE TROP PEU DE LA PORTE D'i0). UNE seule CAUSE, sur une
// seule carte du catalogue — mais plusieurs calques en dépendent, et elle y rendait les
// projectiles inexploitables.
//
//	ce qui change   La PORTE d'`object-position-component` (i0 des objets du monde : projectiles
//	                ti=41, équipement ti=37, armes au sol ti=42) était écrite EN DUR à 3 bits
//	                dans `decodeWorldObjectPos` et `projPosBits` : 1 precHigh + 1 index-sel +
//	                UN bit d'index de région. La largeur de cet index est une CONSTANTE PAR
//	                CARTE (`regionIndexBits` du catalogue de bornes, `ceilLog2(nb de régions)`).
//	                Elle vaut 1 sur 78 cartes du catalogue et DEUX sur la 79e, Live Fire
//	                (`sgh_interlock`, quatre régions déclarées, arène en région 1). Le décodeur
//	                y consommait donc un bit de trop peu, et lisait les TROIS axes un bit trop
//	                tôt. La porte suit désormais la largeur d'index du descripteur de précision
//	                world-object (`IndexW`, installée depuis le catalogue), et l'index lu
//	                est COMPARÉ à la région jouée de la carte au lieu d'être exigé nul.
//
//	la mécanique    Décalés d'un bit, les champs se chevauchent : le bit de poids faible de X
//	                devient le bit de poids FORT de Y, celui de Y le bit de poids fort de Z. Un
//	                bit de poids faible bascule d'une image à l'autre — d'où un saut de la
//	                MOITIÉ de l'étendue de l'axe à chaque bascule (31,89 m pour 63,775 m
//	                d'étendue Y sur Live Fire ; 11,45 m sur Z). C'est la forme exacte que le lot
//	                B avait mesurée sans l'expliquer (« le saut vaut étendue / 2^k »).
//	                Le bit d'index restant, toujours à 1, devenait le bit de poids fort de X :
//	                le nuage de projectiles y était comprimé de moitié ET décalé d'une
//	                demi-étendue — X [-0,7 ; 34,7] au lieu de [-16,7 ; 24,4] sur `0797ce72`.
//
//	mesuré          Records i0 acceptés porteurs d'un pas impossible (instrument
//	                `bit_projectile_research_test.go`, avant -> après) : `0797ce72` 8 091 / 15 971
//	                (50,7 %) -> 7 / 15 930 (0,04 %) ; `21ece4d8` 6 958 / 13 173 (52,8 %) ->
//	                3 / 13 109 ; `c88ec007` 2 758 / 5 693 (48,5 %) -> 1 / 5 691.
//	                Cuisson complète (vols tronqués et points publiés, avant -> après) :
//	                `0797ce72` 239 -> 4 et 471 -> 2 949 ; `21ece4d8` 144 -> 1 et 209 -> 2 367 ;
//	                `30724141` 162 -> 0 et 291 -> 2 012 ; `c88ec007` 69 -> 0 et 96 -> 1 050.
//	                Pas médian après correctif sur `0797ce72` : 0,85 m, p95 2,48 m, max 7,70 m —
//	                une balistique, plus une droite.
//
//	les autres      `decodeWorldObjectPos` sert AUSSI l'équipement (`ti=37`) et les armes au sol
//	calques         (`ti=42`) : les POSES D'ÉQUIPEMENT passent de 49 à 227 sur `0797ce72` et de
//	                27 à 114 sur `21ece4d8`, et les lancers de grenade retrouvent leur lien vers
//	                le projectile né d'eux (`grenades[].proj` : 2 -> 85 et 0 -> 137). Les armes
//	                au sol, les tirs et les ramassages ne bougent pas (217, 717, 108 des deux
//	                côtés) : ils ne tiennent pas leur position de ce chemin. Un lancer se perd
//	                sur `0797ce72` (103 -> 102) — le garde d'auteur de v52 refuse une fenêtre
//	                ambiguë, comme il le faisait déjà sur Cliffhanger.
//
//	contrôle croisé INDÉPENDANT du garde-fou : la branche PROJECTILE des lancers de grenade, que
//	                le lot B voyait s'effondrer à 1 et 0 lancers sur les deux films Live Fire
//	                (toutes les naissances refusées à 25-27 m de leur lanceur), retrouve 83 et
//	                137 lancers à une médiane de 0,44 m et un pire cas de 0,50 et 0,66 m — le
//	                régime exact de Cliffhanger. Rien dans cette mesure ne passe par le seuil de
//	                10 m : elle juge la position, pas la continuité.
//
//	témoins         Cliffhanger (`000d5950`), Banished Narrows (`fb1a1a72`, `51ebbc0f`,
//	                `e60aaf06`), The Pit (`a4083bd2`), Isolation (`daaa17d6`) : pistes, points et
//	                coupures IDENTIQUES à l'unité. Le correctif ne déplace rien là où l'index de
//	                région tient sur un bit — c'est-à-dire partout ailleurs.
//
//	POURQUOI LA     Un artefact 52 d'un match Live Fire ne porte que le premier tiers de seconde
//	VERSION MONTE   de chaque vol de projectile : le garde-fou de v52 coupait au deuxième ou
//	                troisième point. La reprise du backfill se faisant par SchemaVersion, sans
//	                montée aucune recuisson ne le rattraperait.
//
//	ce que le lot   la QUEUE de pas impossibles des cartes Forge (612 pas sur cinq films, |Δx|
//	n'a pas fait    médian 21,69 m). Elle ne vient PAS de la porte : ces cartes ont un index de
//	                région d'un bit, et leurs artefacts sont identiques avant et après. Instruite
//	                au lot B-bis, elle porte la signature d'un FAUX POSITIF du balayage par
//	                position de bit — Y figé au quantum près (479, 735) pendant que X saute d'une
//	                puissance de deux exacte, sur plusieurs slots et générations à la fois. Le
//	                garde-fou de v52 la couvre et devient rare : c'est son rôle.
//	                Détail : `.ai/RAPPORT_LOT_BBIS_BIT_PROJECTILE_2026-09-12.md`.
