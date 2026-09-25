package replay

// document_chronicle.go — LA CHRONIQUE DU SCHEMA : une entree par version, ce qu elle change
// et pourquoi elle monte.
//
// EXEMPTION ECRITE AU SEUIL DES 500 LIGNES (CLAUDE.md regle 5, posee le 2026-09-14, lot 1.0
// revue R1 constat R1-4). Ce fichier est APPEND-ONLY PAR CONSTRUCTION : une entree de chronique
// decrit une montee de schema DEJA CUITE dans le parc, elle ne se reecrit pas et ne se supprime
// pas — la relire est precisement ce qui permet de savoir ce qu un artefact ancien porte. Le
// scinder par tranches de versions rendrait la lecture chronologique impossible et casserait
// l extracteur unique (`testutil.ReplayChronicleVersions`, qui lit CE fichier et dont un
// resultat vide rend inerte le garde-rail `document_shape_test.go`). Sa croissance est donc
// attendue, bornee par le rythme des montees de schema, et n est pas de la dette.
//
// RETRAIT DE L EXEMPTION : le jour ou la chronique deviendrait une donnee (fichier versionne
// hors source Go) lue par le meme extracteur — pas avant.
//
// # RE-JUSTIFICATION A CE VOLUME (2026-09-16, lot 2.7 volet publication, constat C1 de la revue
// # de jalon M1)
//
// L exemption ci-dessus fut posee a 1 230 lignes ; le fichier en porte 1 541. La revue de jalon
// a demande de la RE-JUSTIFIER a ce volume plutot que de la reconduire en silence. Mesure du
// jour, collee :
//
//	lignes du fichier                            1 541
//	lignes de CODE (hors commentaire et vide)         1   (`package replay`)
//	entrees de chronique (`// v<N> (`)               45   (v2 a v60)
//	moyenne par entree                            ~34 lignes
//
// CE QUE LA MESURE ETABLIT. Le seuil des 500 lignes de CLAUDE.md borne une UNITE DE LECTURE DE
// CODE : au-dela, un fichier melange des responsabilites et sa relecture ne tient plus en tete.
// Ici il n y a qu une ligne de code et aucune responsabilite d execution : le volume est celui
// d un REGISTRE, et sa croissance est exactement le nombre de montees de schema du depot, pas
// une derive de conception. Scinder par tranches (v2-v30 / v31-v60) couperait la seule lecture
// que ce fichier existe pour rendre possible — « qu est-ce qu un artefact de version N porte » —
// et obligerait `testutil.ReplayChronicleVersions` a balayer plusieurs fichiers, c est-a-dire a
// pouvoir en OUBLIER un : un extracteur qui rend une liste incomplete laisse
// `document_shape_test.go` valider une version sans entree, le defaut precis que le garde-rail
// ferme (constat R2-1 de la revue du lot 1.0).
//
// CRITERE DE RETRAIT, INCHANGE ET MESURABLE : la chronique quitte la source Go pour une donnee
// versionnee (`config/` ou `testdata/`) lue par le MEME extracteur unique. Tant qu elle est du
// Go, l exemption tient quel que soit le volume — c est le nombre de responsabilites, pas le
// nombre de lignes, qui la fonde.

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
// PAS que le véhicule a explosé. Détail : internal/games/halo_infinite/film/replay/document_vehicles.go.
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
// conducteur. Détail : internal/games/halo_infinite/film/replay/vehicle_rides_aim.go.
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
// Détail : internal/games/halo_infinite/film/replay/bomb_stats.go et bomb_stats_document.go.
//
// v40 (2026-09-06) : LE PONT D'IDENTITÉ EST COMPLÉTÉ PAR LE TRIPLET, et la version monte pour
// la SEULE raison qui fait monter une version dans ce fichier — la reprise du backfill se lit
// par `SchemaVersion`. AUCUN CHAMP N'EST AJOUTÉ : c'est le CONTENU de `objectives` qui change.
//
// Ce qui a été réparé : depuis `d173b1a8c` (2026-08-28), le calque des actions d'objectif
// résolvait l'identité slot -> joueur par les seuls INSTANTS DE MORT, qui en exigent trois
// (`objectives.deathInstantMin`). Un joueur qui meurt moins de trois fois — c'est-à-dire
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
// Détail : internal/games/halo_infinite/film/internal/facts/objectives/slotidentity_rounds.go (CompletedByLines) et
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
// Détail : internal/games/halo_infinite/film/replay/{closures.go, closures_respawn.go, owners.go,
// grapple_lines.go, equipment_episodes.go} et .ai/V7.5/v2/INSTRUCTION_REGRESSIONS_2_4.md.
//
// v42 (2026-09-06) : LE DRAPEAU A ENFIN DES PORTEURS. Aucun champ n'est ajouté, aucune clé ne
// bouge — c'est le CONTENU de `flagCarries` qui change, exactement comme aux montées v14 et v15
// du même calque, et la version monte pour la raison de toujours : la reprise du backfill se lit
// par `SchemaVersion`, et un artefact 41 est AMPUTÉ de portages sans que rien dans sa forme ne le
// dise.
//
// LE PLAFOND, ET IL FRAPPAIT LES MEILLEURS JOUEURS. Le calque nommait son porteur par le pont
// d'identité PAR MORTS (`objectives.ResolveRoundIdentity`), qui exige `deathInstantMin` = 3
// instants de mort coïncidents pour attribuer un slot d'entité. Un joueur qui MEURT MOINS DE
// TROIS FOIS lui échappe PAR CONSTRUCTION — et ce sont, par définition, ceux qui portent le
// drapeau. Leurs prises étaient comptées `coverage.flagCarries.noBridge` et AUCUN intervalle
// n'était publié pour elles : le drapeau restait dessiné à sa base pendant qu'un joueur le
// portait. Le pont par TRIPLET, lui, les nomme — il apparie les totaux (frags, morts,
// assistances) du statborg aux lignes de match — mais il exige la base, que `games/halo_infinite/film/replay`
// n'ouvre pas.
//
// LA CORRECTION EST UN CÂBLAGE, PAS UNE RÈGLE NEUVE. Le pont COMPLÉTÉ
// (`RoundIdentity.CompletedByLines`, mono-manche, compléter sans jamais contredire, aucun xuid
// deux fois) existait déjà depuis le schéma 40 : il servait les ACTIONS d'objectif, résolu dans
// `replaybuild` — la couche qui reçoit les faits du match. Il descend désormais jusqu'au calque
// du drapeau par `replay.FlagInput.Identity`, et les deux calques partagent LE MÊME pont, résolu
// une seule fois par cuisson. `games/halo_infinite/film/replay` ne voit toujours AUCUN fait de match : il reçoit
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
// Détail : internal/games/halo_infinite/film/replay/build_objectives_live.go (`FlagInput.Identity`,
// `flagIdentityOf`), internal/replaybuild/matchfacts.go (`pontParManche`),
// internal/games/halo_infinite/film/internal/facts/objectives/slotidentity_rounds.go (`CompletedByLines`) et
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
// Détail : internal/games/halo_infinite/film/replay/skull_carries.go (carrierPresence.gate) et
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
// internal/games/halo_infinite/film/internal/facts/objectives/round_bounds.go et .ai/V7.5/v2/MANCHES_COMPTEURS_2026-09-06.md.
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
// que sa forme le dise. Détail : internal/games/halo_infinite/film/replay/{equipment_episodes.go, flag_carries.go}
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
// courante. Détail : internal/games/halo_infinite/film/replay/flag_assign.go et
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
//	                   de son propriétaire, à `+67` bits de l'en-tête NEW (`grammar.ScanBipedCreations`).
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

// v51 (2026-09-10, lot 4.3 — LE TABLEAU DE L'API NOMME LES CORPS HORS TABLE).
//
// ENTRÉE RESTAURÉE LE 2026-09-14 (lot 1.0, revue R2, constat R2-1). Elle n'avait JAMAIS été
// écrite ici : la seule description de cette montée vivait dans les notes par version de
// `document.go`, que le lot 1.0 a supprimées comme doublon — et le doublon était, pour v51, la
// SOURCE UNIQUE. La chronique affirmait par ailleurs que 51 avait été SAUTÉE à une
// renumérotation ; l'historique la contredit : `2fb53db4e` pose `SchemaVersion = 51` le
// 2026-09-10 et `b6b198baf` la remplace par 52 le lendemain. Des artefacts ont donc été cuits
// sous ce numéro, et un lecteur qui ne trouvait pas d'entrée ne pouvait pas dire ce qu'ils
// portent. (32, elle, a bien été prise sur une branche puis renumérotée 33/34 au merge du
// 2026-09-01 — cf. l'entrée v33 — donc aucun artefact intégré ne la porte.)
//
//	ce qui change   DEUX changements de CONTENU CUIT.
//	                (1) `identity.bipedSlots[].bid` NAÎT : le tableau de l'API nomme les corps
//	                dont l'index de participant est LU mais absent de `PlayerIndexTable`, et la
//	                couverture bascule de `non_resolu/index_hors_table` vers `externe`. Mesuré
//	                sur `4f77afc1` : 18 vies non résolues, dont 10 sur des index que
//	                `BOT_METADATA` déclare. Au passage `roster[].bid` cesse d'être vide sur tout
//	                le parc.
//	                (2) `abilityLabels[].family` NAÎT : la table porte la FAMILLE du manifeste,
//	                et le résumé d'usage joint dessus au lieu de reconstruire la famille par la
//	                racine du libellé — d'où `UsageSummaryRev` us4 -> us5.
//
//	POURQUOI LA     Un artefact < 51 porte des vies de bot non résolues, un `bid` vide et aucune
//	VERSION MONTE   famille d'équipement : le résumé d'usage ne peut pas être re-projeté sans
//	                recuisson. La reprise du backfill se faisant par SchemaVersion, sans montée
//	                rien ne le rattraperait.
//
//	détail          `structure_test.go` (paragraphe v51), qui porte les mêmes chiffres.
//
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
//	                à `grammar` : `.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`.
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
//
// ─────────────────────────────────────────────────────────────────────────────────────────────
// v54 (2026-09-12, lot G.2bis) — LA VERSION DU FILM EST LUE, PLUS DEVINEE
// ─────────────────────────────────────────────────────────────────────────────────────────────
//
//	la cause        Le bloc d'event de 60 octets du chunk HIGHLIGHT porte le gamertag a
//	                `b[0:32]` sur les FilmMajorVersion <= 38 et >= 41, et a `b[12:44]` sur les
//	                versions 39-40. `ScanDeaths` passait 0 en dur au parseur — c'est-a-dire le
//	                premier decoupage, pour TOUS les films. Sur un film 39-40 il lisait donc du
//	                rembourrage : la meme chaine pour tous les joueurs.
//
//	l'indicateur    Il existait et personne ne le lisait : les quatre premiers octets de
//	                `chunk_00.bin` (u32 little-endian) sont le FilmMajorVersion, la meme valeur
//	                que l'API publie dans `CustomData.FilmMajorVersion`. Helper canonique :
//	                `grammar.FilmMajorVersionFromHeader`. Parc : 1 351 films, 0 registre
//	                illisible — v31 x3, v33 x3, v37 x10, v38 x1, v39 x26, v40 x185, v41 x1123.
//
//	ce qui change   `gamertagsOf(deaths)` est la table qui nomme `roster[]` et qui rattache les
//	                identites. Mesure avant -> apres, version passee 0 puis version lue :
//	                `e5adf7b2` (v40) 17 identites nommees / 2 noms distincts -> 26 / 26 ;
//	                `111fa685` (v39) 16 / 2 -> 24 / 24. Le kill-feed de `killsource` suit la
//	                meme cause : couverture 5,1 -> 97,0 % et 6,8 -> 94,8 % sur ces deux films.
//
//	temoins         `000d5950` (v41, film de reference du golden) et `5676a9ba` (v41) : 8 / 8 et
//	                26 / 26 des DEUX cotes. Aucune carte, aucun calque, aucun autre compte ne
//	                bouge — le golden d'assemblage ne change que par sa ligne de schema, ses
//	                entrees etant figees dans `testdata/inputs_000d5950.bin.gz`.
//
//	POURQUOI LA     Un artefact 53 d'un match de mars a novembre 2025 porte un roster de deux
//	VERSION MONTE   noms pour vingt-cinq joueurs. La reprise du backfill se faisant par
//	                SchemaVersion, sans montee aucune recuisson ne le rattraperait.
//
//	le champ ajoute `coverage.filmMajorVersion` publie la version lue : elle voyage desormais AVEC
//	                l'artefact au lieu d'exiger une relecture du film. Champ OPTIONNEL, il
//	                n'aurait pas exige le bump a lui seul ; il sert la mesure par version du
//	                lot H, ou chaque calque se juge contre la grammaire qui l'a produit.
//
//	ce que le lot   le REDECODAGE lui-meme (`backfill-replay --only-existing` et le backlog
//	n'a pas fait    killsource par `KillSourceDecoderRev`) : consigne du lot, il reste a lancer.
//	                Detail : `.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md`.

// v55 (2026-09-14, lot 1.0.4 du PLAN_DECODEUR_FILM) : LE REFUS DE PUBLICATION D'UNE VIE CESSE
// D'ETRE MUET.
//
//	le defaut       `decimateTracks` ecarte toute vie dont la trajectoire decimee porte moins de
//	                `MinPoints` echantillons (defaut 2 : une vie d'un seul point n'est pas une
//	                trajectoire). Depuis l'origine du calque, ce refus ne se comptait NULLE PART
//	                — ni dans l'artefact, ni au journal. Un document publiant 90 traces la ou le
//	                film en porte 95 etait indistinguable d'un film a 90 vies, et tout lecteur
//	                qui rapporte un compte de vies au film travaillait sur un denominateur
//	                ampute sans le savoir.
//
//	le champ ajoute `coverage.tracks` : `published` / `publishedPoints` (le DENOMINATEUR, sans
//	                lequel un compte de refus ne se juge pas), `refusedMinPoints` (les VIES
//	                ecartees), `refusedPoints` (les points qu'elles portaient) et `minPoints`
//	                (le seuil APPLIQUE — un compte de refus ne se relit pas sans savoir contre
//	                quoi il a ete mesure). Les deux comptes de refus disent des choses
//	                differentes : dix vies d'un point sont un pool de slots qui s'ouvre et se
//	                referme ; une vie de dix points refusee serait un seuil mal regle.
//
//	ce qui NE       le seuil. `DefaultMinPoints` vaut 2 et le reste : ce lot PUBLIE le refus, il
//	change PAS      ne le rediscute pas — la question appartient a l'utilisateur, et ces
//	                compteurs sont exactement ce qui permet de la lui poser avec un chiffre.
//	                Aucune trace publiee ne bouge, aucun autre calque ne bouge : le regime court
//	                d'equivalence ne montre QUE ce champ.
//
//	POURQUOI LA     Le champ est optionnel, mais il decrit le document ENTIER, pas un calque
//	VERSION MONTE   secondaire : un artefact 54 ne peut pas dire ce qu'il a refuse, et rien ne
//	                permet de le deduire apres coup. La reprise du backfill se faisant par
//	                SchemaVersion, sans montee aucune recuisson ne le rattraperait.

// v56 (2026-09-14, lot 1.6 du PLAN_DECODEUR_FILM) : LE REGISTRE D'IDENTITE PREND LA TABLE DU
// FILM COMME LIEN DIRECT, ET LA VIE D'UN SEUL ECHANTILLON EST PUBLIEE.
//
//	la doctrine     « l'index c'est l'index » (decision utilisateur du 2026-09-07) : quand le film
//	                ECRIT le lien `index <-> xuid <-> gamertag`, c'est lui qu'on lit. La table des
//	                32 slots du corps de `chunk_00` (lot 1.5) devient donc la source PREMIERE du
//	                registre ; la lecture des 5 bits des chunks de replication reste, mais en
//	                COMPLEMENT — pour les seuls joueurs dont la table est muette.
//
//	pourquoi un     la table du film est ecrite au DEBUT du film : un joueur qui rejoint en cours
//	complement      de partie n'y a pas de siege. Mesure du 2026-09-14 sur les huit builds : 0 a 5
//	                par film, 13 au total. Remplacer une lecture par l'autre PERDRAIT ces joueurs ;
//	                les composer n'en perd aucun. La ou les deux parlent du meme joueur, elles
//	                disent la meme chose : 125 accords, 0 contradiction.
//
//	les champs      `identity.players[].link.method` prend la valeur `film_table` pour un lien
//	ajoutes         que la table du film pose (et garde `PlayerIndexTable` pour un lien de repli) ;
//	                `identity.coverage.filmTable` publie l'etat de la source — `lu`, `refus`
//	                (`sans_registre` / `sans_section` / `build_inconnu` / `tronque` /
//	                `table_introuvable` / `vacant_intercale`), `sieges` (la taille reelle de
//	                l'escouade, publiee comme DONNEE), `direct`, `repli`, et le controle
//	                `accord` / `contradiction` / `silence`.
//
//	ce que le       le ROSTER gagne les joueurs que la table du film assoit et que la lecture des
//	document gagne  chunks ne trouvait pas (+1 sur `a521164d` et `11de8353`), et les GAMERTAGS que
//	                le film ecrit pour des joueurs a zero mort — le fil des morts ne nomme que ceux
//	                qui meurent (`111fa685` idx=10 « FlukiestGolf », `e5adf7b2` idx=13
//	                « MarshallG6443 » etaient publies sans nom).
//
//	le seuil de     `DefaultMinPoints` passe de 2 a 1 (lot 1.6.5, decision utilisateur du
//	publication     2026-09-14 : « si le film le dit, on publie »). Une vie d'un seul echantillon
//	                — une position que le film ECRIT pour un joueur a un instant — etait refusee
//	                depuis l'origine du calque sous la phrase « ce n'est pas une trajectoire »,
//	                qui decrivait un RENDU et non une donnee. Le compteur
//	                `coverage.tracks.refusedMinPoints`, pose au schema 55 pour poser la question
//	                avec un chiffre, tombe a 0. VINGT vies entrent sur les huit builds.
//
//	l'oracle de     impose par l'utilisateur avec la decision, et MESURE
//	ces vingt vies  (`vies_un_echantillon_test.go`) : UNE porte une mort ECRITE a son instant
//	                (ecart 72 ms), CINQ sont la derniere image d'un slot que la replication n'a
//	                plus jamais repris, QUATORZE sont ORPHELINES — leur vie se ferme sur un trou
//	                de replication et la mort la plus proche du meme joueur est a 0,95 s a 300 s.
//	                Un orphelin n'est pas un cas a filtrer : c'est un DEFAUT DE LECTURE nomme, et
//	                le publier est ce qui le rend visible. Consigne au plan §4.
//
//	POURQUOI LA     la provenance d'un lien devient une donnee du document, le roster change de
//	VERSION MONTE   contenu et les traces publiees aussi : un artefact 55 ne peut dire ni d'ou
//	                vient son index de joueur, ni porter les vies d'un echantillon. La reprise du
//	                backfill se fait par SchemaVersion — sans montee aucune recuisson ne le
//	                rattraperait.

// v57 (2026-09-14, lot 1.7 du PLAN_DECODEUR_FILM) : L'EQUIPE DE CHAQUE JOUEUR EST DANS LE FILM,
// ET C'EST DE LA QU'ELLE VIENT.
//
//	ce qui etait   `Track.Team` valait -1 sur TOUTES les vies de TOUS les artefacts, et le
//	faux           document le documentait ainsi : « l'equipe n'est PAS dans le film, elle vit
//	               dans la base et le client la joint par XUID ». C'est refute :
//	               `.ai/V7.5/film_re/NOTE_EQUIPE_FILM_2026-09-12.md` etablit par DEUX chaines
//	               sans etape commune — le desassemblage du lecteur `0x140f581e8` et 160 slots
//	               sur 176 en accord exact avec `match_participants.team_id`, dont deux Grandes
//	               batailles a 24/24 — que la trame d'etat la porte, par joueur, sur 4 bits.
//
//	d'ou elle      du composant i0 de l'archetype ti=9 (`managed-player-team-designator-component`),
//	se lit         a une position DERIVEE de la grammaire (en-tete par entite 108 + mot de taille
//	               32 + etat par defaut de ti=9 + 32), jamais cablee. La valeur ecrite vaut le
//	               designateur PLUS UN : le document publie le designateur du jeu, `0..8` pour
//	               les huit camps de `mp_team_designator`, `-1` pour « aucune equipe » (FFA).
//
//	l'appariement  le premier `R(6)` de l'etat par defaut de ti=9 EST l'index de joueur de
//	entite ->      l'entite (mesure du 2026-09-14, 18 films et 7 builds). Il n'y a donc aucun
//	joueur         appariement ordinal a faire, et les joueurs ARRIVES EN COURS DE PARTIE — qui
//	               n'ont pas de siege dans la table du DEBUT du film (lot 1.6) — portent leur
//	               equipe comme les autres : 34 arrivees mesurees, toutes a designateur stable.
//
//	les champs     `tracks[].team` cesse d'etre constant a -1 ; `roster[].team` est NEUF (par
//	ajoutes        `filmIndex`, donc valable aussi pour un joueur sans vie publiee et pour un
//	               bot) ; `coverage.teams` publie ce que la lecture a couvert — `read`,
//	               `refusal`, `records`, `rejected`, `divergences`, `film`, `noTeam`, `unread` —
//	               et les trois compteurs du CONTROLE, `accord` / `contradiction` / `silence`.
//
//	la base ne     decision utilisateur du 2026-09-13 (V4) : « si le decodeur est fiable, pas
//	pose plus      besoin du repli ». `FlagInput.TeamOf` DISPARAIT : l'equipe du porteur de
//	rien           drapeau vient desormais du film, donc l'invariant « jamais son propre
//	               drapeau » tient sur une cuisson HORS LIGNE, ou il se taisait faute de lignes
//	               de match. La feuille de match arrive par `Options.ScoreboardTeams` et
//	               n'alimente QUE les trois compteurs de controle.
//
//	POURQUOI LA    le contenu cuit change sur toutes les vies et tout le roster, et un champ
//	VERSION MONTE  apparait. Un artefact 56 porte `team: -1` partout : il ne se distingue d'un
//	               artefact 57 de mode FFA que par `coverage.teams`, qui n'y est pas. La reprise
//	               du backfill se fait par SchemaVersion.

// v58 (2026-09-14, lot 1.9.0 du PLAN_DECODEUR_FILM) : L'ARTEFACT DIT QUELLE PART DE LUI VIENT
// D'UN REPLI.
//
//	ce qui etait   un REPLI — une decision de secours prise quand la lecture du film ne tranche
//	muet           pas — ne laissait aucune trace. L'audit du lot 0.E en a recense 62 dans le
//	               decodeur, dont neuf portaient un defaut DEJA MESURE (jusqu'a 95 % des poses
//	               d'un film creditees au mauvais joueur, 213 s d'attribution fausse sur une
//	               manche, 27 faux enregistrements sur Live Fire). Aucun n'etait lisible d'un
//	               artefact : deux documents, l'un lu et l'autre repli, etaient identiques.
//
//	le champ       `coverage.fallbacks` est NEUF : une liste `{name, hits}`, triee par nom, des
//	ajoute         seuls replis DECLENCHES pendant cette cuisson. Absente quand aucun ne s'est
//	               declenche. Le nom est stable et se joint au REGISTRE DES REPLIS
//	               (`film/facts/fallback`), qui porte pour chacun sa condition typee, sa date de
//	               pose, sa cible et son critere de retrait (decision D14, ADR 0034).
//
//	une liste      les replis ne se repartissent pas sur les calques existants — celui des
//	plate, et      largeurs d'axe par defaut touche TOUT le decodage, et les trois quarts des
//	pas un bloc    faits replies n'ont pas de bloc de couverture a eux. Une liste plate se lit
//	par calque     sans connaitre la carte des calques et n'oblige aucun des vingt blocs
//	               existants a changer de forme.
//
//	AUCUNE AUTRE   le lot 1.9.0 DECLARE et COMPTE ; il ne change aucune decision. Les conversions
//	DIFFERENCE     (la lecture du film qui remplace l'heuristique) sont les lots 1.9.1 a 1.9.14.
//	               L'equivalence est a zero difference hors la seule etape `artifact`, ou ce
//	               champ apparait.
//
//	POURQUOI LA    un champ apparait dans le document, donc la FORME change (garde-rail
//	VERSION MONTE  `document_shape_test.go`, qui refuse la regeneration sans montee). Un artefact
//	               57 ne peut pas dire qu'il ne doit rien a un repli : il peut seulement ne rien
//	               en dire. La reprise du backfill se fait par SchemaVersion.

// v59 (2026-09-15, lot 1.9.1 du PLAN_DECODEUR_FILM) : L'ORIGINE D'UNE POSE SE LIT DANS LE FILM,
// ET CE QUE LE FILM NE DIT PAS RESTE INCONNU.
//
//	ce qui etait   l'origine d'une pose d'equipement (`deployed` / `dropped` / `unknown`) se
//	decide par     decidait par deux regles de SECOURS : le manifeste du titre
//	une heuristique (`kind = "deployed"` -> `deployed` sans mesure, item H.2 des finitions) puis
//	               une FENETRE TEMPORELLE de 200 ms entre la creation de l'objet et la fin de la
//	               vie de son poseur. Aucune des deux ne lisait ce que le film ECRIT (D13).
//
//	les trois      (1) l'evenement de liste type 103 `EquipmentSpawnedObject` — « une PIECE a ete
//	lectures       engendree » — dont la deuxieme reference DESIGNE la vie de l'objet cree ;
//	               (2) la MORT ECRITE du poseur, que le registre d'identite apparie au siege ;
//	               (3) sa PRISE ECRITE (`equipmentChanges.taken`), qui dit qu'il a ECHANGE.
//
//	LE VOCABULAIRE `deployed` est desormais RESERVE a ce qu'un 103 designe — les panneaux de mur,
//	TRANCHE PAR    et eux seuls : c'est le seul geste de deploiement que le film ecrive.
//	L'UTILISATEUR  Un appareil PORTE qui tombe sort `dropped`, que la cause soit la mort de son
//	(2026-09-15)   porteur ou un echange : les deux sont des lachers, et l'etiquette ne les
//	               distingue pas. Une pose dont le film ne dit RIEN sort `unknown`.
//
//	le champ       `coverage.placements.byCause` est NEUF : la PROVENANCE de l'origine de chaque
//	ajoute         pose — `spawn_event`, `death_written`, `taken_written`, `both` (lectures),
//	               `manifest_piece` (le seul repli restant, registre `fallback`), `none` et
//	               `no_owner` (les deux silences). Sa somme vaut `placements`, exactement.
//	               `coverage.placements.spawnEvents` et `.spawnLists` publient les DENOMINATEURS
//	               de la premiere lecture : les evenements 103 lus, et les listes d'evenements
//	               non vides traversees. Les deux, parce qu'un `spawnEvents: 0` sur 7 850 listes
//	               dit « aucune piece engendree » quand le meme zero sur zero liste dirait « le
//	               lecteur n'a rien pu lire ».
//
//	les tolerances MESUREES, jamais choisies (13 films, 4 583 poses, 2026-09-15) : mort 200 ms
//	               (max 171,7 ms cote lachers, min 205,3 ms cote deploiements — un intervalle
//	               VIDE de 33,6 ms) ; prise 50 ms (103 des 108 prises retenues sont a moins
//	               d'UNE ms) ; designation 103 dans [0, +200] ms apres la creation (les 115 poses
//	               de panneau designees le sont entre +32,2 et +70,2 ms, toutes positives).
//
//	CE QUI BOUGE   le contenu cuit change, et c'est le but. Deux populations perdent une
//	DANS LE PARC   etiquette qu'elles n'avaient pas gagnee : les lachers a mi-vie, qui sortaient
//	               `deployed` et faisaient dessiner un geste qui n'a pas eu lieu ; et les poses
//	               dont le film ne dit rien, que la fenetre classait par correlation. Les comptes
//	               exacts sont au journal du lot (§5 du plan).
//
//	POURQUOI LA    deux champs apparaissent dans le document ET les origines publiees changent.
//	VERSION MONTE  Un artefact 58 ne peut ni dire d'ou vient l'origine de ses poses, ni distinguer
//	               un `deployed` lu d'un `deployed` devine. La reprise du backfill se fait par
//	               SchemaVersion.

// v60 (2026-09-17, vague 2 de la famille 1.9 du PLAN_DECODEUR_FILM — lots 1.9.13, 1.9.10, 1.9.9,
// 1.9.11, 1.9.7, 1.9.14 et les corrections de la revue de jalon M1, fusionnes ensemble, UNE montee
// pour la vague) : CE QUE LE FILM ECRIT DECIDE, ET
// CE QU'IL NE DIT PAS EST COMPTE.
//
//	ce qui etait   quatre faits publies se decidaient par une heuristique (D13) : la FIN D'UNE VIE
//	decide par     de joueur au trou de replication de 5 s (`lifeGapUS`) ; la FIN D'UNE VIE DE
//	une heuristique VEHICULE a une borne de recensement (« 5 s apres le dernier echantillon »,
//	               `VehicleTrack.End` toujours `unknown`) ; la FAMILLE D'UN VEHICULE au marqueur
//	               neutre des qu'un chassis manquait a la table (les occupants JETES avec) ; la
//	               MANCHE a une garde d'ordre muette (`contiguousRounds`) qui jetait un designateur
//	               ecrit sans le dire.
//
//	les lectures   (1) une vie de joueur finit a une MORT ECRITE du joueur (kill-feed / dead-state,
//	               liees par le registre d'identite), a une APPARITION ecrite (slot recycle), a une
//	               fin de manche ou a la fin du film ; un trou de replication est une LACUNE de la
//	               meme vie (lot 1.9.13 : 212 coupures par trou -> 4, 208 lacunes, 14 vies
//	               orphelines -> 0 sur les 8 builds) ; (2) une vie de vehicule finit au DEAD-STATE
//	               ecrit (`ti=40`, marche de `grammar.ScanObjectDeaths`), a la fin du film sinon
//	               (lot 1.9.10 : sur `084a804d`, 19 fins lues sur 97, dead-states a 87 ms et 406 ms
//	               des medailles datees) ; (3) un chassis est NOMME par sa piece ecrite (manifeste
//	               de la chaine de destruction : un meme vehicule porte un identifiant PAR MODULE du
//	               jeu — `0xae845375` = wraith, `0xf6f54e56` = scorpion, confirme par la killsource :
//	               37 kills de Wraith sur 40 dans les vies du chassis ; `0x038df01a` = tourelle
//	               automatique bannie, element de carte, non jouable) (lot 1.9.9) ; (4) le
//	               designateur de manche est publie tel qu'ecrit, et la garde d'ordre devient une
//	               CONTRADICTION publiee (lot 1.9.11 : 14 vraies prolongations en designateur 1
//	               contigu, 13/14 a egalite ; 24 films a designateur 2 sans manche 1 = artefact de
//	               lecture, bit 23 du paquet, non tranche).
//
//	les champs     `coverage.tracks.gaps`, `coverage.tracks.gapMs` (lacunes) ; `Point.G` = duree en
//	ajoutes        ms de la lacune qui PRECEDE le point, portee par le point qui rouvre la piste
//	               (le web ne trace pas de segment a travers) ;
//	               `vehicles[].end` ∈ {`destroyed`, `film_end`, `unknown`} (etait `unknown` seul),
//	               `vehicles[].tEnd` (optionnel, `destroyed` seulement) ;
//	               `coverage.vehicles.{deathsRead, deathsMatched, deathsUnmatched, deathsTailDesync,
//	               endDestroyed, endFilmEnd, endUnknown, samplesAfterEnd}` ;
//	               `vehicles[].family` prend les valeurs `wraith`, `scorpion`, `tourelle_auto_bannie`
//	               (et les chassis secondaires de ghost, banshee, warthog resolvent) ;
//	               `VehicleLabel.{kind, en, fr}` (servi a la requete : `kind = "map_element"` pour
//	               la tourelle) ;
//	               `coverage.score.{roundsWritten, roundsContradicted, roundsContradictedRecords,
//	               roundsDecreed}` (lot 1.9.11) ;
//	               `roster[].seat` (toujours emis : l INDEX DE FILM est le siege), `roster[].seatSource`
//	               (`lu` / `apparie`, optionnel), `coverage.seats.*` (dix compteurs) (lot 1.9.14) ;
//	               `coverage.teams.tracksSlotAmbiguous`, `coverage.identity.filmTable.collisionsIndex`
//	               (revue de jalon M1, lentille L4 : une vie sur un siege recycle ne prend plus l equipe
//	               du premier occupant, un index porte par deux xuids n est pose pour personne) ;
//	               `coverage.fallbacks[]` gagne `repli_chassis_vehicule_marqueur_neutre`,
//	               `repli_cadre_de_marche_par_defaut_conserve`, `repli_manche_zero_decretee`,
//	               `repli_appariement_par_fenetre_temporelle` (killsource : compte en expvar, pas
//	               encore dans le document — D7 (1.9.7)), `repli_siege_du_remplacant_par_appariement_ordinal`
//	               (lot 1.9.14) ; `repli_fin_de_vie_vehicule_par_recensement`
//	               SORT du registre.
//
//	CE QUI BOUGE   le contenu cuit change, et c'est le but : moins de vies de joueur, plus longues
//	DANS LE PARC   (335 vies fusionnees sur les 14 temoins, aucune perte de point, de borne, de tir,
//	               de kill ni d'objectif) ; les fins de vehicule lues ; les occupants des Wraith
//	               publies (ils etaient jetes) et leurs tirs rattaches ; le ratchet des replis
//	               `devant_la_lecture` passe de 6 a 3.
//
//	POURQUOI LA    la FORME change (champs neufs, `end` a trois valeurs) et le CONTENU change sur
//	VERSION MONTE  tout le parc. Un artefact 59 ne peut ni porter une lacune, ni une fin de vehicule
//	               lue, ni nommer un Wraith. La reprise du backfill se fait par SchemaVersion.

// v61 (2026-09-17, lot 2.6 du PLAN_DECODEUR_FILM) : L'ARTEFACT DIT SOUS QUELLES REVISIONS IL A
// ETE CUIT.
//
//	ce qui etait   la revision du decodeur ne vivait que dans le depot : les goldens de
//	hors du        `source.Rev`, `profile.Rev`, `grammar.Rev` et `facts.Rev` disent ce que la TETE
//	document       decode, jamais ce qu'un artefact DEJA CUIT porte. Pour savoir sous quelle
//	               grammaire un document avait ete produit, il fallait dater sa cuisson et
//	               remonter au commit — et la version de schema ne repond pas a cette question :
//	               plusieurs revisions de grammaire tiennent sous une meme version de schema.
//
//	le bloc        `coverage.decoder` est NEUF, et il porte LES QUATRE REVISIONS DE CALQUE
//	ajoute        (decision V15 (11)) : `sourceRev` (la porte aux octets), `profileRev` (la table
//	               du decodeur), `grammarRev` (la grammaire de lecture), `factsRev` (la couche des
//	               faits, celle qui commande le backlog killsource) — dans l'ordre du sens unique.
//	               UNE SEULE valeur ne dirait pas OU le changement a eu lieu, or c'est ce que la
//	               recuisson selective par calque (4.4) doit decider. Plus `build`, la cle du
//	               profil lue en clair dans `chunk_00` section 2 (D-3 de l'ADR 0034). Un pointeur
//	               en `omitempty`, pour la meme raison que `FilmMajorVersion` : l'ABSENCE du bloc
//	               dit « artefact anterieur a ce lot », et c'est une reponse, pas un trou.
//
//	build inconnu  `build` vaut la CHAINE VIDE et le bloc reste PRESENT dans les deux cas ou la
//	               cle n'a pas servi : un film sans section d'identification (5 films du cache,
//	               majeures 31 et 33) et un build que la table de profil ne connait pas
//	               (`ErrUnknownBuild`). Sans cette regle, l'absence du bloc porterait deux sens —
//	               « cuit avant le lot » et « build inconnu » — et c'est exactement l'ambiguite
//	               « entre deux versions de schema » que D-7 interdit. Tout ce qui a ete LU reste
//	               publie, le sous-bloc `registry` en particulier.
//
//	`coverage.     LA CLASSIFICATION DE L'EMPREINTE DU REGISTRE ECS (item 3.2.1, volet code
//	decoder.       partiel) : `fingerprint` (`0x` + 16 hex), `status` (`connue` / `inconnue`),
//	registry`      `blocks`, `namedSlots`. L'empreinte etait CALCULEE PUIS JETEE par
//	               `ReadFilmIdentity` (D4 (3.2)) et ne survivait que dans un avertissement de
//	               journal dedupliqué par processus : un film cuit sous une grammaire de
//	               composants jamais vue etait indistinguable d'un film nominal. Le troisieme
//	               statut `presumee` attend la recopie des empreintes du catalogue dans la table
//	               du PROFIL (volet 3.1.1) — le decodeur ne peut pas lire `filmprofile`, le sens
//	               unique l'interdit. Absent quand le registre n'a pas ete lu.
//
//	`coverage.     LES DENOMINATEURS DU BALAYAGE des impulsions de capacite (D3 (validation),
//	abilityImpul-  point 2) : `records`, `withI57`, `withI59`, `read`, `unread`, `tag1`. `reads`
//	ses.scan`      comptait les lectures brutes sans dire si la marche avait atteint sa cible :
//	               la validation de la recuisson M1 a perdu huit lectures sur six films entre les
//	               schemas 54 et 60, et `reads=0` etait indistinguable d'une marche cassee — ce
//	               que la doctrine de `coverage.go` interdit. Absent quand le balayage n'a PAS
//	               TOURNE : un bloc de zeros affirmerait qu'il a tourne sans rien rencontrer.
//
//	AUCUNE AUTRE   TELEMETRIE PURE : aucun rendu n'en depend, aucune decision de decodage ne
//	DIFFERENCE     change, et AUCUNE des quatre revisions ne monte dans ce lot — donc aucun match
//	               ne devient candidat au backlog killsource (D6). L'equivalence est a zero
//	               difference hors les champs `coverage.decoder` et `coverage.abilityImpulses.scan`.
//
//	POURQUOI LA    des blocs apparaissent dans le document, donc la FORME change (garde-rail
//	VERSION MONTE  `document_shape_test.go`, qui refuse la regeneration sans montee). Un artefact
//	               60 ne peut pas dire sous quelle grammaire il a ete cuit : il peut seulement ne
//	               rien en dire. Les artefacts deja cuits restent servis tels quels, leur bloc
//	               `decoder` absent jusqu'a leur prochaine cuisson — aucune recuisson requise.

// v62 (2026-09-17, lot 4.2.1 du PLAN_DECODEUR_FILM, jalon M4) : L'ARTEFACT DIT SOUS QUELLE
// REVISION CHAQUE CALQUE A ETE PRODUIT, ET PAR QUELLE VOIE SES MORTS ONT ETE LUES.
//
//	ce qui etait   `coverage.decoder` (v61) dit les revisions de la CUISSON ENTIERE, pas celles
//	hors du        d'un calque : rien ne relie un champ du document a la couche qui l'a produit.
//	document       La recuisson selective de M4 (4.4) doit decider PAR COUCHE — « seule la
//	               publication a bouge : republier depuis les faits » contre « la grammaire a
//	               bouge : redecoder » — et cette decision n'avait aucune donnee a lire. Les
//	               comptes PAR VOIE de lecture des morts, eux, etaient mesures par le decodage
//	               (`killsource.Result.Stats`) et s'arretaient a la frontiere de l'artefact.
//
//	`layers`       NEUF, A LA RACINE : `nom de calque -> revision de la couche qui l'a produit`.
//	               La cle est la balise JSON du calque ; la valeur l'une des CINQ revisions
//	               connues (`source-...`, `profile-...`, `grammar-...`, `killsource-...`,
//	               `publication-<schemaVersion>`). 47 champs racine attribues, la regle
//	               d'attribution et ses deux limites ecrites dans `layers.go`. Objet ABSENT =
//	               artefact anterieur a 62 ; entree ABSENTE dans un objet PRESENT = ce calque
//	               n'a pas ete produit, et c'est une REPONSE (une garde de mode fermee, un
//	               balayage qui n'a pas abouti) ; entree presente = produit sous la revision
//	               nommee. C'est le meme regime que `coverage`, et D-7 de l'ADR 0034 le pose :
//	               « la presence d'un calque se lit dans sa revision, jamais dans l'absence
//	               d'un champ ».
//
//	`coverage.     NEUF : les trois denominateurs (`population`, `matched`, `published`) de
//	deathsPaths`   CHACUNE des deux voies de lecture des morts — la MARCHE (`walk`) et le SCAN
//	               DIRECT (`scan`). Les deux lisent le MEME champ et repondent au meme bit quand
//	               les deux repondent (desaccord zero), mais leurs precisions different (98,2 %
//	               contre 78,4 % au gate d'appariement) : sans le compte par voie, une ligne de
//	               mort venue du rattrapage s'affichait comme une ligne nominale. Absent quand
//	               les morts n'ont pas ete lues sur ce match (`KillsInput.Read` faux) — jamais
//	               six zeros, qui se liraient comme « deux voies ont tourne, rien trouve ».
//
//	`vehicleLabels` RECLASSE, ET C'EST UNE CORRECTION MESUREE : ce champ est resolu A LA REQUETE
//	passe a la     par `service/replay_vehicle_labels.go`, et AUCUN chemin de `build*.go` ne le
//	requete        pose. Il manquait pourtant a `calquesALaRequete` (`document_shape_test.go`)
//	               depuis son ajout, ce qui le comptait dans l'empreinte de forme CUITE. Le
//	               reclassement change cette empreinte : il ne pouvait donc entrer que dans un
//	               commit qui monte `SchemaVersion`. AUCUN OCTET D'ARTEFACT NE CHANGE — la
//	               cuisson ne l'ecrivait deja pas, et le service continue de le poser a la requete.
//
//	CE QUI N'Y     les CINQ compteurs du balayage des lancers de grenade (D3 (3.3.1)) restent
//	ENTRE PAS      journalises : `grammar` ne les exporte pas, et les faire sortir changerait des
//	               octets de la couche, donc `grammar.Rev`, donc `facts.Rev` — c'est-a-dire le
//	               backlog killsource que V24 a repousse sur signal explicite. Les compteurs de
//	               3.4.1 (positions par index de plage) n'existent PAS en production : aucune
//	               `Observation` n'est attachee par `NewFilmContextForMap`. Les deux sont statues
//	               `[!]` au §4 du plan, avec leur mesure.
//
//	AUCUNE AUTRE   TELEMETRIE PURE des deux cotes : aucun rendu ne change, aucune decision de
//	DIFFERENCE     decodage ne change, et AUCUNE des quatre revisions ne monte dans ce lot — donc
//	               aucun match ne devient candidat au backlog killsource (D6). L'equivalence est
//	               a zero difference hors les champs `layers` et `coverage.deathsPaths`.
//
//	POURQUOI LA    un champ apparait a la racine et un bloc dans `coverage`, donc la FORME change
//	VERSION MONTE  (garde-rail `document_shape_test.go`, qui refuse la regeneration sans montee),
//	               et le reclassement de `vehicleLabels` change l'empreinte de forme cuite. Un
//	               artefact 61 ne peut pas dire sous quelle revision chacun de ses calques a ete
//	               produit : il peut seulement ne rien en dire. Les artefacts deja cuits restent
//	               servis tels quels, `layers` et `deathsPaths` absents jusqu'a leur prochaine
//	               cuisson.

// v63 (2026-09-19, post-chantier lot 5.1, montee UNIQUE de la branche `feat/decfilm-51`) : DEUX
// REAPPARITIONS QUI MANQUAIENT — celle du DRAPEAU et celle des VEHICULES.
//
//	`flagCarries[] LA JAUGE DE RETOUR d'un drapeau reste au sol, en serie datee sur l'echelle du
//	.spans[]       jeu (0 = vide, 1 = pleine) et sur les SEULS intervalles `dropped`. Le type
//	.returnProgress` publie est `GaugePoint`, le MEME que celui des zones : meme escalier, meme
//	               allegement, meme lecture cote client. Cle ABSENTE quand le film n'emet rien
//	               sur l'intervalle.
//
//	               CE QUE LA VERSION REPARE, ET C'EST UN MODELE ET NON UN CANAL. Ce fichier
//	               portait, depuis la v14, « LE RETOUR AUTOMATIQUE d'un drapeau reste au sol.
//	               [...] Aucune minuterie ne se deduit de cette dispersion ». C'etait vrai et ca
//	               le reste : le retour N'EST PAS un minuteur. Le jeu remplit une JAUGE au taux
//	               `1/reset + H(n)/solo` (`CalculateReturnRateHarmonic`), et cette jauge est
//	               ECRITE dans le film — `ti=13 i1` tag 3, voie delta, le MEME canal que la
//	               jauge de capture des zones. Deux voies de minuteur avaient ete cherchees et
//	               REFUTEES par la mesure (le bassin du moteur, 446/446 a « aucun minuteur » ;
//	               le minuteur manuel du navpoint, 588/588 a zero) : la lecon n'est pas « le film
//	               ne l'ecrit pas », c'est que la QUESTION etait mal posee.
//
//	               L'APPARIEMENT EST UNE CORRELATION TOTALE, pas un nom : 884 echantillons sur
//	               trois jauges et deux films tombent A 100,0 % dans un lacher de LEUR drapeau,
//	               contre 15,8 % pour le meilleur slot voisin. L'oracle qui la valide est binaire
//	               et sans exception — retour automatique 13/13 au plein, repris avant la fin
//	               0/14. Preuve, seuils et pieges : flag_return_gauge.go.
//
//	`coverage.     CINQ denominateurs NEUFS (`gaugeScanned`, `gaugeSlots`, `gaugeReads`,
//	flagCarries`   `gaugePaired`, `gaugeSpans`, `gaugePoints`) : ils separent les quatre silences
//	               qu'un `returnProgress` absent ne distingue pas — canal non lu, lu sans slot de
//	               jauge, slots sans correlation, correlation sans emission sur l'intervalle.
//
//	`vehicleCycles` LE CYCLE DE REAPPARITION par EMPLACEMENT de naissance de vehicule : mediane,
//	               deciles, ecarts mesures, manques comptes — la FORME de `PadCycle`, et le MEME
//	               juge (`gwPadsCycleFromGaps`). Le film n'ecrit aucun minuteur de reapparition
//	               de vehicule (negatif mesure sur les 294 noms de composant) : le cycle se
//	               DEDUIT de la mort datee a la naissance suivante au meme endroit, les
//	               naissances etant agglomerees a 2 m. Seuls les emplacements ETABLIS y figurent.
//
//	               POURQUOI MAINTENANT, ALORS QUE LA NOTE 3.7 CONCLUAIT « NON MESURABLE ». Elle
//	               avait raison AU 2026-09-17 : avec 1 fin datee sur 109 vies, le compte des
//	               ecarts etait ZERO sur 31 emplacements. Ce n'etait pas l'appariement mais la
//	               GRAMMAIRE — la marche de `ti=40` perdait 90 a 100 % des dead-states que le
//	               film ANNONCE, faute de lire l'etat par defaut de l'archetype. Le lot 5.1.7-b
//	               l'a pose : `4f77afc1` passe de 3 a 11 fins datees et de 97 a 149 vies
//	               publiees, `a349fea8` a 14, et `finDatee == mortsAppariees` sur les deux.
//
//	`coverage.     QUATRE denominateurs NEUFS (`cycleLocations`, `cycles`, `cycleGaps`,
//	vehicles`      `cycleMissing`) : une liste `vehicleCycles` vide ne dit pas si le film n'a
//	               aucun emplacement, si aucun n'a rendu d'ecart, ou si les ecarts etaient trop
//	               disperses pour etablir quoi que ce soit.
//
//	CE QUI N'Y     LA SURFACE WEB DU CYCLE DE VEHICULE. Le champ est publie, le client ne le
//	ENTRE PAS      dessine pas encore : le rejeu n'a AUCUNE infobulle ni carte de vehicule (les
//	               quatre infobulles existantes sont celles des poses, des socles, du drapeau et
//	               des armes au sol), et en creer une est un lot de RENDU — celui qui reprend
//	               deja l'orientation, la taille et les tirs des vehicules. Decision de pilote du
//	               2026-09-19, consignee au §4 du plan comme item TRANSMIS, pas comme dette : le
//	               contrat est pose, l'affichage suit.
//
//	AUCUNE AUTRE   les quatre revisions de DECODAGE ne bougent pas. `ti=13` etait deja lu en
//	DIFFERENCE     production (jauge des zones) et `ti=40` l'a ete au lot 5.1.7-b : cette montee
//	               n'ajoute AUCUN octet de grammaire, elle PUBLIE ce qui etait deja decode. Le
//	               fichier de FAITS, lui, s'etend — la jauge y transite, comme toute entree dont
//	               le document depend (propriete du lot 4.1 : « document depuis les faits ≡
//	               document depuis le film », a l'octet). La MAGIE DU BLOB d'entrees ne monte
//	               PAS, et c'est verifie sur pieces : la jauge voyage par `encodeGardesDeMode`,
//	               dans la section 1 A LA SUITE du blob — exactement la ou `ZoneReads` vit deja —,
//	               donc le format du blob est inchange a l'octet et les huit fixtures d'entrees
//	               restent valides. C'est `SchemaDesFaits` qui porte le changement (2 -> 3).
//
//	POURQUOI LA    deux champs apparaissent (un a la racine, un dans `flagCarries[].spans[]`) et
//	VERSION MONTE  neuf compteurs entrent dans `coverage` : la FORME change, et le garde-rail de
//	               forme refuse la regeneration sans montee. Les artefacts deja cuits restent
//	               servis tels quels, les deux champs absents jusqu'a leur prochaine cuisson.
// v66 (2026-09-21, post-chantier lots 5.9.4 et 5.9.5, decision utilisateur) : LE SPRINT, LU —
// ET LE SAUT, PUBLIE ET DIT DERIVE.
//
//	`stances[]`    DEUX GENRES DE PLUS, et ils ne sont PAS de la meme nature. `sprint` est LU :
//	               `ti=35 i57 biped-spartan-ability-component` porte l INDEX DE LA FENTE DE
//	               CAPACITE ACTIVE, et la fente 1 est le sprint. `jumpDerived` est CALCULE :
//	               c est l integrale de la vitesse verticale d `i1`, reconnue a sa HAUTEUR, et
//	               son nom porte le mot pour qu on ne puisse pas confondre les deux. Deux
//	               compteurs neufs dans `coverage.stances` : `jumpEpisodes` (montees fermees
//	               examinees) et `jumpsDerived` (celles retenues).
//
// LE SPRINT : LES TROIS FENTES SONT NOMMEES PAR L IMAGE, PAS PAR UN SCORE. `FUN_1407e9ce4`
// aiguille sur le GROUPE DE TAG de la definition de capacite et appelle, pour chacun, un
// desenregistreur qui teste l index actif contre SA fente : `'saev'` esquive -> `FUN_14319d0ac`,
// fente 0 ; `'sasp'` SPRINT -> `FUN_14319d1ec`, fente 1 ; `'sagh'` grappin -> `FUN_14319d14c`,
// fente 2. Le flux ecrit `bloc+3 = R(2) - 1`, donc le brut `2` designe la fente 1. La chaine
// complete va de la condition d animation `is_sprinting_tlg` au bit 45 des drapeaux d unite,
// pose par `Sprint::Update` (`FUN_1431a2474`) depuis une fraction rampee LOCALEMENT — ce qui
// vient du film est l ACTIVATION, pas la fraction, et c est elle qu on publie.
//
// LE CONTROLE QUI VALIDE LA LECTURE DE L INDEX EST CELUI DU GRAPPIN, ET IL EST FRANC : sur
// `4f77afc1`, la vitesse au sol pendant les intervalles de la fente 2 monte a **5,84 m/s** au
// p90, contre 2,88 hors intervalle — la TRACTION du grappin, deux fois le plateau de course.
// Si la fente 2 est bien le grappin, la lecture de l index est juste, donc la fente 1 est bien
// `'sasp'`. C est la preuve croisee la plus forte disponible, et elle porte sur le MEME champ,
// le MEME pliage, le MEME instrument.
//
// CE QUE LA VITESSE DU SPRINT, ELLE, NE PEUT PAS PROUVER — ET C EST MESURE. Le score des
// intervalles de la fente 1 contre un plateau haut de vitesse rend 37,5 % / 60,8 %
// (`bfecd02b`) et 42,0 % / 40,5 % (`4f77afc1`). Ce n est pas la fente qui est mal nommee :
// c est l etiquette physique qui est faible, et le depot l avait deja mesure (lot 5.3.5 : la
// distribution de vitesse au sol n a QU UN SEUL mode). Vm 2,55 contre Vs 2,84 sur le second
// film — 0,29 m/s d ecart, couvert par la dispersion. Ce qui converge quand meme : la vitesse
// MAXIMALE atteinte pendant un intervalle a un p10 de 2,73 m/s contre 1,91 pour un temoin
// apparie (meme vie, meme duree, cinq secondes plus tot), et sa mediane vaut 2,92 m/s.
//
// POURQUOI LE NOM PORTE LE MOT « DERIVE ». Les trois autres genres sont des bits que le
// deserialiseur publie ; celui-ci est un CALCUL. Un client qui affiche `jumpDerived` doit
// pouvoir le distinguer d une lecture sans consulter de documentation, d ou le genre distinct
// plutot qu un drapeau a cote — et d ou le libelle « Saut (derive) » / « Jump (derived) ».
//
// LA HAUTEUR EST UN FAIT DE JEU, PAS UN SEUIL D INSTRUMENT. `types.SpartanJumpHeightM` vaut
// 0,85 m : la distribution des hauteurs d episode aerien, integrees depuis la vitesse verticale
// TENUE, porte un pic etroit a cette valeur sur DEUX films — `bfecd02b` (snowbound) pic x 10,7
// au-dessus de ses voisins, montee 0,467 s ; `4f77afc1` (flood gulch) pic x 3,9, montee 0,466 s
// (lot 5.7.5, 2026-09-21). Tous les Spartans sautent la meme hauteur : c est ce qui autorise la
// derivation, et la fenetre est de +/- 10 %.
//
// CE QUI EST REFUSE, ET C EST DELIBERE. Un episode encore OUVERT a la fin de la marche n a pas
// d instant de fin mesure et sa hauteur est tronquee par le silence qui la termine : il n est pas
// publie. Et un silence de replication de plus de 250 ms n est PAS une vitesse tenue — l integrer
// fabriquerait des hauteurs.
//
// CE QUE CETTE VERSION NE FAIT PAS : le SPRINT. Sa chaine de donnees est pourtant complete
// (lot 5.9.1) — l etiquette d `i57 biped-spartan-ability` est l INDEX DE LA FENTE DE CAPACITE
// ACTIVE (`-1` = aucune, `0..2` = la fente), applique par `FUN_1406c9b1c` puis `FUN_14319db80`,
// et le bit 45 des drapeaux d unite que lit `SpartanAbilityIsSprinting` est pose par
// `Sprint::Update` (`FUN_1431a2474`) depuis une fraction rampee LOCALEMENT. Ce qui manque est le
// nom de la fente qui porte `'sasp'`. L utilisateur n a autorise aucune derive pour le sprint :
// le plateau de vitesse au sol n a qu un seul mode.
//
// QUAND LE CHAMP REPLIQUE DU SAUT SERA NOMME, un genre `jump` LU remplacera `jumpDerived`, avec
// sa propre montee. La chaine du declencheur est remontee jusqu au compteur de ticks sans contact
// `u+0x89b` et NON TROUVEE a `FUN_1408b2f90` (lot 5.9.2) : c est un maillon manquant, pas un
// refus.
//
// POURQUOI LA VERSION MONTE : un genre neuf apparait dans `stances[].kind` et deux compteurs
// entrent dans `coverage.stances` — la FORME change, et le garde-rail de forme refuse la
// regeneration sans montee.
//
// v65 (2026-09-21, post-chantier lot 5.3.6, decision utilisateur) : LES ETATS DE MOUVEMENT DU
// SPARTAN, EN INTERVALLES PAR VIE — ET TROIS SEULEMENT, PARCE QUE TROIS SEULEMENT SONT LUS.
//
//	`stances[]`    UN INTERVALLE PAR (VIE, GENRE) : `{slot, kind, t0, t1}` sur l axe de
//	               `Point.T`. Trois genres : `crouch` (accroupi, `ti=35 i29`), `slide`
//	               (glissade, `i62`), `mobility` (action de mobilite, `i54`). Le type publie
//	               est neuf, `Stance` ; la couverture aussi, `coverage.stances`.
//
// D OU ILS VIENNENT. Le film ECRIT ces trois etats A L INSTANT, pas aux images-cles : un delta
// ne porte le composant que quand l etat CHANGE. `grammar.ScanMovementStates` rend les
// TRANSITIONS et `document_stances.go` les replie en intervalles, avec le MEME plieur que les
// episodes d equipement (`episodeAccum`) — memes fenetres de vie, meme cloture a la mort.
//
// CE QUI A RENDU LE CALQUE POSSIBLE, ET C EST TOUT LE LOT 5.3. La marche de production des
// autres canaux de capacite est un CHERCHEUR D ANCRES : sur `bfecd02b` elle annonce `i29` ZERO
// fois sur 162 444 records, parce qu elle ne retient que la population pauvre `{i0,i1,i21,i25}`.
// Le calque emploie donc la marche du FRAME-PROCESSEUR (`DecodeFrameViews`, trois vues — ce que
// `FUN_142987460` deroule), avec les paquets a liste d evenements localises par la signature du
// depot. Deux pre-requis, tous deux mesures : la bascule `SimStateComplet` liee a la carte
// (lot 5.3.3-a, sans quoi `i60` ferme la traversee avant `i61-63`) et les LARGEURS D AXE DE LA
// CARTE installees sur le contexte (lot 5.3.5, sans quoi `i0` lit aux largeurs de
// `cliffhanger` : records `ti=35` 31 530 contre 97 447, desyncs 38 contre 3).
//
// LE SPRINT ET LE SAUT N Y SONT PAS, ET C EST UNE MESURE. Le sprint est REFUTE comme observable
// par la vitesse : la loi de dequantification d `i1` est exacte (ecrivain relu, constantes
// relues), un oracle independant la valide sur deux films (dispersion du rapport
// deplacement/vitesse 1,7 et 2,3 ; meme facteur d unite 0,240 et 0,236), et la distribution au
// sol n a QU UN SEUL mode, a 2-3 m/s. Le saut est LU mais PAS PROUVE : sa segmentation repose
// sur deux seuils d instrument et la signature ne tient pas d un film a l autre (0,632 s contre
// 1,567 s de duree mediane). Publier l un des deux publierait un SEUIL comme une DONNEE.
//
// MESURE DU CALQUE (`bfecd02b`, Snowbound) : 97 447 records `ti=35` dont 3 desynchronises,
// 7 941 lectures retenues, 101 slots distincts — crouch 2 490 lectures dont 495 posees, slide
// 2 487 dont 386, mobility 2 964 dont 758.
//
// LE CODEC DES FAITS MONTE AVEC (`REPLAYINPUTS24`) : les lectures voyagent par les faits, donc
// un artefact re-cuit depuis un fixture porte les memes intervalles que la cuisson complete.
//
// v64 (2026-09-20, post-chantier lot 5.2-A, demandes utilisateur du 2026-09-19) : LA COULEUR DE
// LA CAPTURE, C'EST-A-DIRE LE CAMP QUI POUSSE LA JAUGE.
//
//	`zoneStates[]  UNE ENTREE PAR RAMPE de la jauge de capture — les memes rampes dont `gauge`
//	.gaugeRamps`   est tiree (`findZoneRamps`) —, avec ses bornes et, quand elle ABOUTIT, le
//	               camp qui l'a poussee. Le type publie est neuf, `ZoneGaugeRamp`.
//
//	               CE QUE LA VERSION REPARE. La serie de jauge est ANONYME par construction :
//	               le slot de rampe ne porte aucun proprietaire (mesure du lot C-bis, deja
//	               ecrite dans `zoneStatesLayer.ts`). Le client en etait donc reduit a DEDUIRE
//	               le capteur — « le camp d'en face du proprietaire courant » —, une deduction
//	               qui ne vaut qu'a deux camps ET seulement sur une zone TENUE. Sur une base
//	               NEUTRE elle n'existe pas : le remplissage s'y peignait au neutre alors
//	               qu'une equipe poussait, et c'est exactement le constat de l'utilisateur du
//	               2026-09-19. La deduction est remplacee par une MESURE.
//
//	               LE CAMP EST CELUI DE L'ISSUE, ET IL N'EST PUBLIE QUE QUAND LA RAMPE ABOUTIT.
//	               Le canal de PROPRIETE de la zone — celui-la meme qui produit les intervalles
//	               — est relu a la frame du sommet ou juste apres, dans la fenetre
//	               d'appariement du volet (`zoneValueAfter`). Une rampe qui avorte n'apprend
//	               rien sur le pousseur : le canal y nomme encore le DEFENSEUR, et le publier
//	               ferait peindre la capture a la couleur de celui qui la subit. La cle est
//	               alors ABSENTE, et le client repeint au neutre.
//
//	               LE SEUIL D'ABOUTISSEMENT EST MESURE, PAS REGLE (8 documents a zones du
//	               cache, 241 rampes, 2026-09-20). Separees par ce que le canal de propriete
//	               fait apres le sommet : 160 rampes sont suivies d'une bascule de camp et
//	               leurs sommets vont de 0,976 a 0,999 ; les 81 autres n'en produisent AUCUNE
//	               et plafonnent a 0,986 — dont DEUX seulement au-dessus de 0,95 (0,983 et
//	               0,986), qui sont des RE-SECURISATIONS par le camp deja en place : le canal
//	               n'y change pas de valeur, donc `mergeZoneRuns` n'ouvre pas d'intervalle,
//	               mais la valeur qu'il porte EST celle du pousseur. Hors ces deux cas, le plus
//	               haut sommet sans bascule vaut 0,938 : `zoneGaugeRampComplete` (0,95) tombe
//	               dans une marge mesuree de 0,038.
//
//	POURQUOI UN    `GaugePoint` est un type PARTAGE depuis la v63 — la jauge de retour du
//	SPAN ET PAS    drapeau l'emploie (`flagCarries[].spans[].returnProgress`) — et un camp de
//	UN CHAMP SUR   capture de zone n'a aucun sens sur un retour de drapeau : y poser le champ
//	`GaugePoint`   polluerait le second calque d'une cle qu'il ne remplira jamais. Le repeter
//	               sur chaque point couterait en outre UNE CLE PAR POINT (36 pour la seule
//	               rampe temoin de `396cfc92`) pour une valeur constante sur toute la rampe ;
//	               le span en porte UNE par rampe, soit 241 entrees pour les 8 documents.
//
//	AUCUNE AUTRE   les quatre revisions de DECODAGE ne bougent pas, et AUCUN octet de
//	DIFFERENCE     `film/internal/` n'est touche. Le canal de propriete des zones etait deja lu
//	               en production depuis la v16 : cette montee PUBLIE une lecture existante, elle
//	               n'en ouvre aucune. `layers` est inchange — les zones gardent leur couche —,
//	               le format du blob d'entrees est inchange a l'octet, et aucune recuisson
//	               n'est requise pour les autres calques.
//
//	CE QUI RESTE   la grammaire porte un archetype `zones` dedie (`ti=23`,
//	OUVERT          `selectable-zone-data-component`, 32 instances) dont le deserialiseur est
//	               ECRIT mais NON CABLE (`status=deser_non_cable`, `doc_field` vide,
//	               `product_use=aucun`) et dont la table dit qu'il porte « l'identifiant, la
//	               POSITION et l'ETAT » d'une zone de mode. Lire l'etat a la source plutot que
//	               de le reconstituer par vote de canal est donc possible — c'est un lot de
//	               GRAMMAIRE, hors du perimetre de celui-ci (§4 du plan).
//
//	POURQUOI LA    un champ naît dans `zoneStates[]` et un TYPE naît avec lui : la FORME change,
//	VERSION MONTE  et le ratchet de forme refuse de se regenerer sans montee. Ce n'est pas de la
//	               telemetrie — le client peint avec. Les artefacts deja cuits restent servis
//	               tels quels, la cle absente jusqu'a leur prochaine cuisson, et le repli neutre
//	               du client est exactement celui d'aujourd'hui.

// v67 (2026-09-21, post-chantier lot 5.10, arbitrage utilisateur) : L OCCUPATION D UN VEHICULE
// EST LUE — LA PROXIMITE N EST PLUS QU UN REPLI, ET ELLE LE DIT.
//
//	`rides[].src`  DEUX VALEURS A LA PLACE DE TROIS, et elles ne disent plus la meme chose.
//	               `film` = le film ECRIT cette montee a bord (`object-parent-state`, `i10`,
//	               ecrivain `FUN_140c1e4d0`) ; `proximity` = REPLI, l episode vient du trou de
//	               position de l occupant. Les anciennes valeurs `event` / `mixed` / `gap`
//	               ventilaient la PRECISION DES BORNES d un episode heuristique ; cette
//	               ventilation descend au journal de cuisson (`vehicleRideStats`), parce que la
//	               question du lecteur a change : ce n est plus « a quelle milliseconde pres ? »
//	               mais « est-ce lu, ou deduit ? ».
//
//	`rides[].seat` MEME FORME, AUTRE SOURCE (deja en place au lot 5.10.3) : le siege vient du
//	               champ de six bits de la queue d `i10` (+0x3a0), plus du champ `R(6)` de
//	               l evenement, que la mesure D1 (5.5) avait refute — 153 occurrences de
//	               `seat = 0` pour 100 tirs de tourelle, jamais de siege 1 ni 2.
//
//	`coverage.`    DEUX COMPTEURS A LA PLACE DE TROIS : `ridesRead` et `ridesProximity`
//	`vehicles`     remplacent `ridesFromEvent` / `ridesMixed` / `ridesFromGap`. Leur somme vaut
//	               toujours `rides`, et ils disent au lecteur du document ce qui est LU et ce
//	               qui est DEDUIT — la seule ventilation qui porte une decision.
//
// LA PRIMAUTE DE LA LECTURE, ET SON PRIX. Un episode de proximite n est publie que s il ne
// CONTREDIT aucune lecture de la MEME vie de vehicule : ni chevauchement de fenetre, ni occupant
// absent des occupants que le film a nommes pour cette vie. Une vie dont le film n a RIEN lu
// n est jamais contredite — l absence de lecture n est pas une absence d occupant. Le prix est
// ecrit et compte au journal : une vie dont le film n a lu QU UN siege perd les episodes
// heuristiques de ses autres sieges.
//
// CE QUE CETTE REGLE CORRIGE, ET C EST UN VERDICT DE L UTILISATEUR (Theater, 2026-09-19) : le
// Razorback `776/1` de `4f77afc1` publiait deux episodes — `Dafar8423` et `Yessireezy` — quand
// le film n y ecrit qu UNE montee a bord, celle du slot `524` (siege 1, a 1:54.5 temps film).
// Yessireezy ne monte jamais dans ce vehicule ; il est tue A COTE a 3:01 par un tir de mortier.
// Les deux episodes sont desormais ECARTES par la lecture.
//
//	AUCUNE AUTRE   les revisions de DECODAGE ne bougent pas dans cette montee (`grammar` et
//	DIFFERENCE     `facts` ont monte au commit precedent du lot, avec la lecture d `i10`).
//	               `layers` est inchange — l occupation reste dans le calque des vehicules. La
//	               fin de vie `despawn` N EST PAS publiee : la mesure du lot 5.10.4 a refute ses
//	               trois canaux, et `end` garde ses trois valeurs.

// v68 (2026-09-22, post-chantier lot 5.22.4, verdict Theater de l utilisateur) : L ACTION DE
// MOBILITE EST NOMMEE — `stances[].kind` `mobility` DEVIENT `clamber`, L ESCALADE.
//
//	`stances[]`    UNE VALEUR RENOMMEE, PAS UNE VALEUR NEUVE : le genre `mobility` devient
//	`.kind`        `clamber`. Le bit publie est le MEME (le drapeau d amorce d
//	               `i54 biped-mobility-action-component`), la marche est la meme, le nombre
//	               d intervalles est le meme. Ce qui change est que le document DIT ce que le
//	               geste est, au lieu de dire quel composant le porte.
//
// POURQUOI LA VERSION MONTE ALORS QUE LE BIT NE CHANGE PAS : un client qui connait `mobility` ne
// reconnaitra plus ce genre, et `stanceAt` (cote web) IGNORE un genre inconnu — un artefact cuit
// avant cette montee afficherait donc des escalades muettes chez un client neuf, et l inverse.
// La valeur d un enum publie est de la FORME ; un renommage est une montee, jamais un detail.
//
// CE QUI TRANCHE, ET CE N EST PAS UNE CHAINE DU BINAIRE. Le lot 5.13.2 s etait arrete
// explicitement : la queue d `i54` est la charge utile du message reseau `initiate_mobility_action`
// (`143c97470`), les dix chaines de `mobility` du binaire sont des noms de BOUTON, d ENTREE
// d armure ou d IMAGE, et aucune n etiquette les quatre valeurs de `bloc + 0x9c`. La decision
// d alors — « les quatre valeurs restent non nommees, `mobility` NE DEVIENT PAS `clamber` » —
// tenait faute d oracle. L oracle est arrive : l utilisateur a ouvert Theater sur `bfecd02b` et
// confronte neuf intervalles de ce genre, pris sur ce film. **Les neuf sont des escalades
// de rebord, 9 verdicts sur 9, aucun contre-exemple** ; le plus net est celui du temoin du
// lot 5.22 (Madina97294, slot 523), un saut date a 96,962 s qui se termine en prise sur un
// element du decor, ou `i54` s allume de 97,096 a 97,63 s et d ou le Spartan redescend ensuite.
//
// LE VOCABULAIRE DU JEU CORROBORE SANS PROUVER : `_action_hoist`, `_action_vault`,
// `_action_climb_attach`, `_action_climb_detach` (`143ca0100` et suivants) et le mode de physique
// `CharacterPhysicsModeClambering` (`143df73d0`, `FUN_1406b8244(idx) == 2`). C est le mot du jeu
// pour ce geste ; le verdict vient de l ecran.
//
//	PAS DE GENRE   le lot 5.22 a cherche un `jump` LU et ne l a pas trouve : sur une vie
//	`jump`         repliquee A CHAQUE TICK (le temoin, 4 557 records sur 80 s), l ensemble des
//	               champs qui basculent au decollage et nulle part ailleurs est VIDE, et il l est
//	               sur les 68 vies de `bfecd02b` comme sur les 238 de `4f77afc1`, les 64
//	               composants du bipede au denominateur. `jumpDerived` reste donc le seul genre
//	               du saut, et son nom continue de dire qu il est calcule. Ce qui n est pas
//	               prouve n est pas publie.
//
//	AUCUNE AUTRE   `layers` est inchange (l escalade reste dans le calque des etats de
//	DIFFERENCE     mouvement) ; les compteurs de `coverage.stances` gardent leurs noms et leurs
//	               valeurs ; aucune revision de DECODAGE ne monte pour ce renommage seul
//	               (`grammar` a monte au commit precedent du lot, pour le cablage du bloc
//	               d action de la vue de controle).

// v69 (2026-09-24, vague C des retours du rejeu, plan `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md`
// §4.4) : UNE SEULE MONTEE POUR QUATRE LOTS — M1 (positions), M5 (score a sens unique et fil des
// morts), M4a (vehicules : pieces montees, registre), M6 (registre et catalogue : la remise des
// mains nues). Chaque lot avait pose 69 sur sa branche (M6 avait d abord pose 70, rabattu sur 69
// par sa revue adverse, constat R4 : le plan ne fait qu UNE montee par vague tant que la
// republication 69 n est pas faite, et la vague D pose deja SON 70) ; l integration les reunit
// sous ce seul numero (un artefact au schema 69 porte les quatre). Aucune revision de DECODAGE ne
// monte : republication DEPUIS LES FAITS. Les quatre parties suivent, puis celle de M7 (decor de
// carte), qui ne touche que la forme SERVIE : aucun artefact cuit n en porte la trace.
//
// v69, PARTIE M1 (2026-09-23, decision utilisateur Q15) : LA PUBLICATION DES
// POSITIONS APPLIQUE LA GRAMMAIRE DE LA VIE, DEUX REPLIS NOMMES, ET DIT SES SILENCES AUX VEHICULES.
// Republication DEPUIS LES FAITS : aucune revision de decodage ne monte (`grammar.Rev`,
// `facts.Rev`, `SchemaDesFaits` inchanges), la montee perime les seuls calques de publication.
//
//	`tracks`       ne portent plus les positions ANTERIEURES a la creation de leur corps (R-B1 :
//	               une vie ouverte par son record commence au record ; R-B2 : aucune vie avant
//	               le premier record d un slot), ni les positions hors de l EMPRISE JOUEE du
//	               film ET ISOLEES (repli `repli_position_hors_emprise_ecartee`, la garde de
//	               `boundsOf` desormais ecrite une fois : emprise_jouee.go ; une chute reelle,
//	               continue, reste publiee). La regle de creation se DESARME sur un slot dont
//	               le premier record lu n est pas `gen=1`.
//	`vehicles`     meme emprise pour les echantillons et les naissances (une fausse naissance
//	               anterieure ne l emporte plus sur la vraie ; un record posterieur a la fin de la
//	               vie n est pas sa naissance) ; de deux sejours qui se contredisent au travers
//	               d un silence de plus de `lifeGapUS`, celui que son autre voisin contredit (la
//	               naissance pour le premier), a soutien egal le plus court, est ecarte s il fait
//	               au plus 3 echantillons (repli `repli_echantillon_vehicule_au_travers_d_un_
//	               silence_ecarte`) — sinon rien, et le refus se compte ; une vie sans position
//	               restante sort en `noPosition`.
//	`vehicles[]    CHAMP NEUF : `g`, la lacune de replication qui precede l echantillon, en ms
//	.samples[].g`  — la semantique de `Point.g`. Le client TIENT la derniere position au travers.
//	`coverage      `avantCreation`, `viesAvantPremiereCreation`, `horsEmprise` (positions
//	.tracks`       BRUTES du film, avant decimation), `slotsArmes` / `slotsDesarmes` (derive de
//	               la generation du premier corps ; parc du 2026-09-24 : 11 407 et 1)
//	`coverage      `echantillonsHorsEmprise`, `spawnsHorsEmprise`, `echantillonsAuTraversDUnSilence`
//	.vehicles`     (echantillons BRUTS, fenetres de vie comprises ou non), `silencesNonTranches`
//
// POURQUOI « si le film le dit, on publie » (2026-09-14) NE COUVRAIT PAS CES POINTS : le film ne
// les dit pas, c est le BALAYAGE ANCRE qui les lit. La meme suite de bits revient au meme decalage
// sur des cartes aux quantifications differentes (annexe `RAPPORT_positions_limbe.md` §1.2 :
// 62 paires de 24 bits communs sur 1 926 chez les aberrants, 1 sur 49 600 chez les normaux), et
// le moteur ne replique aucune position d un corps avant de le creer. L utilisateur a tranche le
// 2026-09-23 (Q15) : ces vies ne sont plus publiees, elles sont COMPTEES.
//
// CE QUI NE BOUGE PAS : l origine (`originMs`, `frameCount`) reste lue sur TOUS les paquets de
// position — ecarter un point ne decale aucun calque ; `layers` ; les revisions de decodage. Les
// deux replis tombent avec la porte grammaticale au decodage (option 2 du rapport), dont ils sont
// le critere de retrait. Temoins : 81c02726 (Mongoose 770, Madina97294), ab526724, 879a4dba.
//
// v69, PARTIE M5 (2026-09-23) :
// LE MATCH A SENS UNIQUE A UN CAMP — `coverage.score.teamIdentity` GAGNE LA VALEUR `a0`.
//
//	`coverage.`    UNE VALEUR D ENUM NEUVE, `a0` : la preuve (a) du score final, appliquee au
//	`score.`       match ou UN SEUL slot d equipe porte une serie de score. Le camp muet n a
//	`teamIdentity` jamais quitte zero et le statborg n emet un composant qu a son CHANGEMENT ;
//	               quand le registre dit X-0 (X > 0) et que la serie finit EXACTEMENT a X, elle
//	               est le camp X et le slot muet l autre. Un slot seul vu (le camp muet n a rien
//	               emis, ni score ni frags) suit la meme regle.
//	`scoreTimeline` CONSEQUENCE, PAS FORME : la serie d un match a sens unique porte desormais
//	`.teams[]`     son `teamId`. Avant, elle sortait sans camp (`unresolved`) des que le perdant
//	`.teamId`      n avait jamais marque — la preuve (a) exigeait un score final sur les DEUX
//	               slots. Mesure du 2026-09-23 : six documents a une seule serie sans camp au
//	               parc, dont cinq sains (le sixieme, `ab526724`, etait un film tronque, repare
//	               depuis par O1). Le client affichait 0 — 0 tout le match sur ces documents.
//
// POURQUOI UNE VALEUR A PART ET PAS `a`. La preuve ajoute une premisse — « absent vaut zero » —
// que (a) n a pas. La couverture doit dire laquelle a tranche : un camp resolu par (a0) repose
// sur la grammaire d emission du statborg, pas sur deux lectures.
//
// LE GARDE-FOU EST L EGALITE EXACTE, et il est teste : une serie a 2 contre un registre a 3 (film
// tronque avant sa derniere capture) reste `unresolved` ; un registre 1-3 avec une seule serie a
// 3 aussi — le camp muet du FILM aurait marque au REGISTRE, son absence est un trou de lecture,
// pas un zero. Aucun identifiant de match dans la regle.
//
// L ORDRE DES PREUVES : (a), puis (a0), puis (b). (a0) passe avant la somme des frags parce
// qu elle n emprunte rien au pont d identite des joueurs. Sur les documents a une seule serie que
// (b) resolvait deja, la valeur publiee passe de `b` a `a0` et le CAMP ne change pas — c est le
// controle croise de la montee (le desaccord d un seul camp aurait ete un defaut).
//
//	`coverage.`    UN CHAMP NEUF (lot M5.2, meme montee) : le VERDICT de la lecture du fil des
//	`bridge.`      morts — `read`, `empty` (le morceau des temps forts est lu et ne porte aucune
//	`deathsFeed`   mort : une MESURE) ou `unreadable` (pas de morceau des temps forts, morceau
//	               absent, evenements illisibles : une PANNE). Absent = non mesure (assemblage
//	               sans balayage). Avant, un fil illisible ne se lisait que dans les journaux :
//	               le diagnostic d `ab526724` a exige les journaux ET le code.
//
// LA LIMITE ECRITE DU CHAMP : les faits persistes ne portent PAS le verdict. Rejoue depuis ses
// faits, un fil VIDE OU ILLISIBLE publie la cle ABSENTE (le champ se tait plutot que de choisir) :
// film et faits DIVERGENT sur ce champ — ecart NEUF, a declarer a replay-equiv et au gate de parc
// (§8, decouverte 4 : il se ferme avec la revision de faits qui persistera le verdict). Depuis le
// lot L3, un film sans morceau des temps forts n est plus cuit du tout.
//
// POURQUOI LA VERSION MONTE : une valeur d enum publie est de la FORME, et la regle de
// publication du calque de score change ; `deathsFeed` est un champ neuf de la couverture. Republication DEPUIS LES FAITS (aucune revision de
// DECODAGE ne monte : `grammar`, `facts`, `layers` et le blob d entrees sont inchanges).
//
// v69, PARTIE M4a (2026-09-23, reprise du 2026-09-24) : LES VEHICULES — PIECES MONTEES POSEES SUR LEUR
// PORTEUR, VARIANTE NOMMEE, REGISTRE DES ARMES DE VEHICULE PUBLIE.
//
//	`vehicles[]`   `part = "turret"` : PIECE MONTEE (LAAG, lance-roquettes, tourelles du Falcon et
//	               du Wraith, canon du Scorpion), jamais dessinee seule. `carrier {slot, gen}` : le
//	               chassis qui la porte, par le REPLI NOMME `repli_tourelle_porteur_voisin_de_slot`
//	               (slot +1 / +2, famille attendue, fenetres qui se recouvrent, NES ENSEMBLE a 1
//	               frame et 1 m pres ; le lien parent n est pas lu). `variant` : `rockethog` /
//	               `warthog_gauss` par la piece, `gungoose` par l arme — le sprite ; la famille ne
//	               change pas (moteur, explosion, classe d arme).
//	`rides[]`      `turret {slot, gen}` : l episode d artilleur REPORTE sur le porteur, SANS `seat`
//	               (le siege lu etait celui de la tourelle). Trois refus le gardent sur la piece :
//	               porteur non pilotable (Falcon), hors de la fenetre du porteur, occupant deja a
//	               bord (episodes qui se RECOUVRENT ; un changement de siege jointif est reporte).
//	`shots[]`      un tir d artilleur sort du PORTEUR (`v` = le chassis, `x`/`y` sa position), plus
//	               de la naissance de la tourelle (mediane 44,7 m au parc avant ce lot) ; hors de la
//	               fenetre du porteur, il n est pas pose (`shotsUnplaced`).
//	`coverage.`    DIX COMPTEURS : `turrets`, `turretsOnCarrier`, `turretCarrierBirthMismatch`,
//	`vehicles`     `turretRides`, `turretRidesDropped` = `turretRidesNotRideable` +
//	               `turretRidesOutOfWindow` + `turretRidesAlreadyAboard`, `shotsOnCarrier`,
//	               `variants`. Une piece ne compte plus dans `familyUnknown` / `unknownChassis` /
//	               `repli_chassis_vehicule_marqueur_neutre`, un artilleur reporte pas dans `ambiguous`.
//	`vehicle`      RACINE NEUVE, RESOLUE A LA REQUETE (`calquesALaRequete`, jamais cuite) : le registre
//	`Weapons`      `config/titles/{slug}/mappings/vehicle_weapons.toml` (forme, teinte, son ou silence
//	               decide, montage, libelle FR/EN) des armes que les tirs emploient, keye par
//	               `Shot.w` ; il remplace les trois tables CLIENT clees par des tags jamais filmes.
//
//	AUCUNE REVISION DE DECODAGE ne monte (vies deja assemblees : la republication DEPUIS LES FAITS
//	suffit). `facts.Rev` INCHANGEE (le registre des replis recoit une entree de DONNEES ; empreinte
//	seule recopiee). `layers` inchange : les pieces restent dans le calque des vehicules.
//
// v69, PARTIE M6 (2026-09-24, lot M6 des retours du rejeu, decisions de l utilisateur du
// 2026-09-24) : LA REMISE DES MAINS NUES N EST PAS UNE PRISE. `00007CA9` est l objet « mains nues »
// du jeu (sonde CA9 : `WeaponTags.unarmed` du Lua global) ; le jeu le REMET a chaque bipede au
// debut de chaque vie, et le film l ecrit comme une prise sur DEUX canaux (parc de la tete de la
// vague C, 107 documents : 562 ramassages natifs de classe ARME dans 77 documents, et 1 `taken`
// du canal des changements d arme — aucun au milieu d une vie).
//
//	`pickups[]`    SENS : la remise n y est plus publiee (regle nommee `filmshell.IsUnarmedFamily`,
//	               une seule ecriture du litteral, garde-rail archlint).
//	`coverage.`    UN CHAMP NEUF, `unarmedGrants` : le compte des remises. `unknownFamilies` cesse
//	`pickups.`     de les compter ; `decoded` = `published` + `beforeOrigin` + `unarmedGrants`.
//	`weaponChanges[]` SENS : une PRISE (`taken`) de l objet n est plus publiee ; un ECHANGE vers
//	               lui (le joueur a tout jete) le reste, nomme.
//	`coverage.`    UN CHAMP NEUF, `unarmedGrants` ; `decoded` = `published` + `restated` +
//	`weaponChanges.` `beforeOrigin` + `unarmedGrants`.
//	`loadouts[]`   la meme regle ecarte l objet de toute dotation publiee (passage unique
//	               `dotationWeaponName`, garde-rail archlint).
//
// CONSEQUENCES DECLAREES, HORS DU DOCUMENT : le rejeu ne joue plus le son de ramassage d une
// remise (elle sonnait `weapon_pickup` a chaque debut de vie), et la mesure des armes de base des
// paliers de socle (`prisesPour` / `prisesEnVie`, qui lisent ces deux canaux) cesse de tenir une
// remise pour une prise.
//
//	AUCUNE REVISION DE DECODAGE ne monte : republication DEPUIS LES FAITS. Meme lot, SANS effet de
//	forme : le catalogue d armes du titre nomme trois familles jusqu ici publiees en hexadecimal
//	(`hinf_unarmed`, et la bobine a fusion UNSC `hinf_coil_kinetic` pour `e9e7ff79` / `1d63a8cd`),
//	et `weaponLabels` les porte ; `killEffects` gagne l explosion a la mort de `hinf_scorpion` et
//	`hinf_rockethog`.
//
// v69, PARTIE M7 (2026-09-24, lot M7 des retours du rejeu, decision de l utilisateur du
// 2026-09-24) : LE DECOR DE CARTE EST POSE ET HORS DE LA ZONE JOUABLE, DECIDE PAR LE SERVICE.
//
//	`vehicle`      RACINE NEUVE, RESOLUE A LA REQUETE (`calquesALaRequete`, jamais cuite) : les
//	`Scenery`      cinq conditions de pose de L1.3 ET hors de la zone jouable (matiere praticable
//	               du fond de carte publie en plan, sol foule du match en hauteur). `zone`, `floor`,
//	               `candidates`, `inPlayArea`, `zoneUnknown`, `hidden[] {slot, gen, reason}` ; deux
//	               replis nommes au registre (`repli_decor_sous_le_sol_foule_du_match`,
//	               `repli_decor_carte_sans_zone_affiche`). Le client ne decide plus : L1.3 lit ce
//	               verdict. Empreinte CUITE inchangee, `facts.Rev` inchangee : rien a re-cuire.
//
// v70 (2026-09-24, vague D des retours du rejeu, plan `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md`
// §4.5) : UNE SEULE MONTEE POUR LES LOTS DE LA VAGUE — M2 (equipes, presence et place lues dans
// le film) et M3 (marche d image-cle, armes de naissance). Chaque lot, parti de
// `fe7079f41` AVANT la vague C, avait pose 69 sur sa branche ; l integration les reunit sous ce
// seul numero, apres le 69 de la vague C (dont l entree ci-dessus n est pas touchee). Les
// revisions de DECODAGE montent UNE fois pour la vague, a des valeurs datees de l integration :
// `grammar.Rev` -> `grammar-2026-09-24`, `SchemaDesFaits` 3 -> 4, `facts.Rev` ->
// `killsource-2026-09-24` (par M3 seul : la marche d image-cle que `killsource` traverse ;
// backlog killsource ouvert, geste de production sur signal). Republication apres
// RE-DECODAGE des films (verdict « redecoder »). Les parties suivent.
//
// v70, PARTIE M2 (2026-09-23, lot M2.3 de la campagne « retours rejeu », decision Q24 : option
// A) : LA PLACE ET LA PRESENCE DE CHAQUE OCCUPANT SONT LUES DANS LE FILM — LA REGLE DES PLACES.
//
// LA REGLE, ET ELLE EST DE L UTILISATEUR (2026-09-23) : « quand un joueur part, il libere la place
// de sa fiche de joueur pour son remplacant [...] le nombre de joueur dans un match est fini, il y
// a un maximum. » Une equipe a un nombre FINI de places ; une fiche = une place ; le remplacant
// (bot ou humain) prend la place du partant ; jamais plus de fiches que de places ; un parti
// n est jamais affiche.
//
// CE QUI ETAIT FAUX, MESURE (rapport `RAPPORT_equipes_b1ad85eb.md`) : sur `b1ad85eb`, Eagle a 3
// joueurs au coup d envoi puis 5, Cobra 5 a 6:24. Le siege publie etait l INDEX de participant (un
// remplacant n en herite presque jamais), l equipe etait agregee PAR INDEX (les trois bots d index
// 8, de deux equipes, n en avaient aucune), et la presence n etait publiee nulle part : le client
// gardait un parti affiche faute de successeur sur SON siege. Au parc : 14 documents sur 111
// depassent la taille d equipe, 22 affichent des joueurs partis.
//
//	`roster[]`     `presence` NAIT : une liste d intervalles `{from, to, toMax?}` en frames. `to`
//	.presence      est la derniere frame CERTAINE (vie, image-cle, paquet BOT_METADATA ; `to <
//	               from` : aucune, vu avant l origine), `toMax` la derniere ou il PEUT etre la —
//	               l entite ti=9 ne se lit qu aux images-cles (~20 s), et un depart pendant la
//	               mort sort a la premiere image-cle qui ne la porte plus (Q22), ou a l arrivee
//	               du remplacant sur la meme place. Un bot sort a la frame EXACTE de son retrait
//	               (BOT_METADATA). Un occupant present sans corps y est « pas encore apparu »
//	               (Q21) ; entre un partant et son remplacant la place est VIDE (Q20).
//	`roster[]`     devient la PLACE : un siege de la table du debut (`chunk_00`) — l index pour
//	.seat          les occupants du depart, la place LUE dans les tirs pour un remplacant qui
//	               tire (l index de tireur EST la place, 3 remplacements sur 3), sinon le CHAINAGE
//	               par equipe, ou une place OUVERTE sous la capacite estimee de l equipe (deux
//	               replis nommes, comptes ; `e5adf7b2` : 23 sieges pour 12 contre 12). `seatSource`
//	               gagne `tirs`, `ouverte` et `index` (aucune place : compte `sansPlace`) ;
//	               l appariement ordinal du lot 1.9.14 est RETIRE.
//	`roster[]`     l equipe de l ENTITE ti=9 de l entree (grammaire M2.1), plus celle de son
//	.team          index : un index repris par deux equipes ne rend plus muet aucun occupant.
//	`coverage`     onze compteurs : `placesTirs`, `placesOuvertes`, `sansPlace`, `depassements`
//	.seats         (0 attendu : les couples frame x equipe ou une equipe affiche plus d occupants
//	               que de places), `relaisBornes`, `chevauchements`, `tirsContestes`,
//	               `tirsIndexTronque` (index de tireur persiste sur 4 bits : la lecture par les
//	               tirs s abstient au-dela de 15 places), `presences` (`film` ou `vies` — le repli
//	               sans entite), et `entitesNonLiees` / `entitesContestees` / `trousDEntite`.
//
// LE CORPS D UN INDEX PARTAGE EST NOMME par l entite qui vit a sa creation (lecture, voie
// `biped_creation`) : les neuf corps de bots de `b1ad85eb` sortent de `index_hors_table`, et leurs
// pistes prennent le nom du bot. Les relais par la base (`successions.go`) deviennent un repli
// compte (`repli_vie_de_bot_par_relais_de_la_base`).
//
//	AUTRES         `grammar.Rev` -> `grammar-2026-09-24` (les entites ; valeur de la vague, cf.
//	MONTEES        l en-tete) ; `SchemaDesFaits` 3 -> 4 (les faits portent entites et instants
//	               BOT_METADATA : verdict « redecoder ») ;
//	               `facts.Rev` NE MONTE PAS (aucune ligne de kill ne change). `layers` : `roster`
//	               reste attribue a `facts`, la plus haute des deux couches qui le produisent.
//
// POURQUOI LA VERSION MONTE : un champ naît, trois changent de sens, et c est la CLE DE REPRISE du
// backfill — un v69 affiche encore un joueur parti et doit se lire « a re-cuire ». Le client lit
// un artefact anterieur sans `presence` par l enveloppe des vies (repli transitoire date, cote web).
//
// REVUE ADVERSE DU LOT (2026-09-24), MEME SCHEMA : le BOT QUI SUCCEDE A UN HUMAIN sur son index
// entre au roster quand ses entites et celles de l humain sont DISJOINTES (`c75f33b8`, `4f77afc1` :
// ses vies etaient nommees sans entree, et le web lui dessinait une place de plus, vide tout le
// match) ; `depassements` se mesure contre la CAPACITE estimee et non plus contre les places posees
// (mesure circulaire) ; une entree d un film balaye qu aucune entite ne porte tient sa place par ses
// vies jusqu a l image-cle porteuse suivante (repli par entree, nomme et compte) ; l occupant qu un
// siege de la table nomme par son xuid est la des la frame 0 meme sans entite. `coverage.seats`
// gagne six compteurs : `capacite`, `placesEnTrop` et `sansEquipe` (ce que la colonne rend au-dela
// de la capacite, ou hors de toute), `identitesHorsRoster` (0 attendu), `botsSuccesseurs`,
// `presencesParLesVies`.
//
// v70, PARTIE M3 (2026-09-23, campagne « retours rejeu », lot M3) : LES ARMES À L INSTANT — la dotation de
// naissance, l emplacement d arme, et la santé de la marche d image-clé.
//
//	`loadouts[]`   `src` NEUF (`birth` = la DOTATION DE NAISSANCE, lue dans le record NEW de
//	.src / .k      création du corps ; absent = relevé d image-clé) et `k` NEUF (l EMPLACEMENT
//	               de chaque arme de `w`, porté par les seules dotations de naissance). La
//	               dotation se pose sur la vie que sa création OUVRE (appariement des vies du
//	               slot à la création la plus proche de leur début) ; un emplacement vide ou une
//	               famille hors du catalogue d armes n est pas publié (`nonWeapon`) — et l objet
//	               « mains nues » `00007CA9` du coup d envoi est écarté par la règle NOMMÉE de M6.3
//	               (`dotationWeaponName`), compté `unarmedGrants` (cf. la partie D-fix).
//	`weaponChanges` `k` NEUF, TOUJOURS PRÉSENT : le rang de l emplacement `weapon-state-type-info`
//	.k             touché (0 = la première arme), lu dans le registre du film — jamais un index
//	               de composant en dur. C est la clé qu une dotation de naissance partage avec le
//	               flux : elle situe une prise sur un emplacement VIDE, que `from` ne nomme pas.
//	SENS           la PREMIÈRE émission d un emplacement se juge, POUR CHAQUE VIE, contre la
//	               dotation de naissance de cet emplacement (même arme = ré-annonce, écartée ;
//	               sinon une prise, un échange ou un lâcher dont `from` NOMME l arme de
//	               naissance), puis contre le dernier relevé d image-clé PASSÉ de la vie —
//	               JAMAIS un relevé À VENIR. Jusqu au schéma 68, un slot sans relevé passé rendait
//	               son premier relevé, fût-il vingt secondes plus tard : une arme ramassée
//	               entre-temps y figurait déjà et sa prise, classée ré-annonce, disparaissait. Et
//	               la chaîne des émissions d un slot ne se coupait pas à la réapparition : la
//	               première émission d une vie se lisait contre la dernière arme de la vie
//	               PRÉCÉDENTE du même slot.
//	`coverage`     `keyframes` NEUF (lot M3.1) : comment la marche d image-clé a atteint chaque
//	               record — voisin, saut de largeur, recalage sur l en-tête exact d un bipède,
//	               élection (le repli nommé `repli_ancre_d_image_cle_par_election`), fenêtres
//	               traversées sans candidat — et `framedAbsentBipeds`, les bipèdes manqués entre
//	               deux images-clés qui les portaient. `birthLoadouts` NEUF : les créations lues,
//	               celles dont le record ne se FERME pas (désynchronisé, débordant, non confirmé
//	               par le record qui suit — AUCUNE lecture de repli ne les remplace), et ce que la
//	               publication a posé, ramené dans la fenêtre de sa vie, ou écarté. `closed`
//	               compte les records FERMÉS, `read` ceux d entre eux qui portent au moins une
//	               arme du catalogue — une fermeture sans arme n est pas une dotation lue
//	               (`noDisplayable` ; `closed` = `read` + `noDisplayable`).
//
// LES DEUX RÉPARATIONS DE GRAMMAIRE QUI OUVRENT LE CANAL (sonde P3) : l état par défaut du
// bipède lit enfin le R(32) de sa dernière feuille (`default_state.go` — deux oracles, la famille
// d i43 au catalogue du match pour 293 naissances sur 293 et `n2` des images-clés, constant et
// plausible sur trois films) ; et le record NEW de naissance est pris comme ANCRE par la signature
// que `ScanBipedCreations` reconnaît déjà, faute de pouvoir démarrer la marche à la fin d une
// liste d événements sans la grammaire de chacun de ses types.
//
//	CE QUI MONTE    `grammar.Rev` et `facts.Rev` (la marche d image-clé et l état par défaut du
//	AVEC ELLE       bipède, que `killsource` traverse aussi) et le codec des faits (v26 : la santé
//	                de la marche, les dotations de naissance et leurs refus, l emplacement de
//	                chaque changement d arme). `layers` ne gagne aucun calque : `loadouts` et
//	                `weaponChanges` restent attribués à `grammar`.
//	LA MESURE       vingt-deux témoins cuits deux fois (base / lot) : vies sans aucun relevé
//	                d armes 18,7 % -> 10,2 % (1,3 % sur les builds HI_1_12_0 et HI_1_13_0, où les
//	                naissances se lisent ; celles des builds antérieurs ne se ferment pas et rien
//	                n est publié), images-clés trouées 8,6 % -> 1,0 %, prises et échanges publiés
//	                1 075 -> 1 257. Prix nommé : la marche des états de mouvement, qui traverse
//	                enfin le record NEW d un bipède, va plus loin dans certains paquets et y lie
//	                des records NEW mal lus — 327 paquets à liste de plus non localisés au corpus.
//
// v70, PARTIE D-fix (2026-09-24, lot correctif de la pré-intégration de la vague D, repris après
// sa revue adverse) : UNE ABSENCE QUE LE PAYLOAD NE PROUVE PAS NE CONCLUT RIEN, LA MARCHE NE PERD
// PLUS CE QU UN RECORD PROUVÉ LUI INTERDIT DE PERDRE, ET LE MONDE DES ÉTATS DE MOUVEMENT NE GARDE
// PLUS UNE LIAISON FAUSSE.
//
// CE QUI ÉTAIT FAUX, MESURÉ : la marche glissante de M3 atteint désormais l image-clé d AVANT-MATCH
// (après un record de 125 270 bits) ; son élection y retenait une fausse ancre (slot 192, `ti 1`,
// dans le corps du record 1298) et effaçait les dix-neuf vrais records 1280..1298 — dont le joueur
// géré de l index 0 (slot 1297). Le lot M2 lisait alors cet occupant ARRIVÉ à l image-clé suivante
// (`000d5950` : absent 16,3 s au départ ; `bcb6d393`, `fb1a1a72` : 7 entités sur 8 au départ).
//
//	CAUSE          la table est à slots CROISSANTS : un candidat que la grammaire du film PROUVE
//	               (sa marche d état complet, contenu compris, ferme sur l en-tête valide suivant)
//	               interdit tout élu qui contredit cet ordre avec lui ; l élu contredit est refusé
//	               et l élection reprend (`grammar/keyframe_world_preuve.go`). Aucun seuil. Toute
//	               marche de CUISSON est celle du film (`FilmContext.MarcheDImageCle`, garde-rail
//	               `archlint/keyframe_walk_proof_test.go`). UNE EXCEPTION en production, hors de la
//	               cuisson : la précision par arme (`ScanFilmWeaponDamages`, appelée par
//	               `sync/killcollector/hits.go`, révision propre `WeaponHitDistanceDecoderRev`)
//	               marche encore SANS preuve — deux marches d un même payload y divergent sur les
//	               fausses ancres réfutées ; y brancher la preuve change ses lignes en base, donc
//	               une décision de backfill soumise à l utilisateur (entrée datée de l allowlist).
//	PRINCIPE       une absence n est PROUVÉE que si l en-tête EXACT du record `ti=9` de l entité
//	               (`[gen|slot][ti]`, 64 bits, le filtre fort de toute ancre) n apparaît à AUCUNE
//	               position de bit du payload de l image-clé (`grammar/player_entities_entetes.go`).
//	               Recherche exhaustive, indépendante des chemins de la marche : l élection de repli,
//	               mais aussi le saut de largeur, le faux voisin et le record au-delà de la fenêtre
//	               perdent des records sans les « écarter » (constat DFIX-R2 de la revue). Un en-tête
//	               trouvé et non lu fait un DOUTE (`Doutes` des entités, persisté) : ni arrivée
//	               tardive ni départ, et `roster[].presence` ne borne que sur une absence prouvée —
//	               sinon l affichage court jusqu à l arrivée d un successeur sur la place (règle des
//	               places), au plus jusqu au bord du film. REPLI NOMMÉ ET COMPTÉ :
//	               `repli_borne_de_presence_differee_sur_doute` (registre `replay/places`, compteur
//	               `bornesDifferees`, retrait avec la marche déterministe).
//	`coverage.`    `refutations` NEUF : les élus refusés ; `contradictoryProofs` NEUF : les
//	keyframes      élections dont TOUS les candidats étaient contredits (deux preuves se contredisent,
//	               l élu d avant la preuve est gardé : un défaut de preuve compté, plus une retombée
//	               muette). L élection reste le repli nommé `repli_ancre_d_image_cle_par_election`.
//	`coverage.`    `imagesClesDouteuses` et `bornesDifferees` NEUFS (0 attendus).
//	seats
//	`coverage.`    `unarmedGrants` NEUF : la dotation de naissance écarte l objet « mains nues »
//	birthLoadouts  par la règle nommée de M6.3 (`dotationWeaponName`), compté à part de `nonWeapon`.
//	`stances[]`    DEUX RÉGRESSIONS RÉSIDUELLES DE M3, instruites par bisection, CORRIGÉES dans la
//	               marche des états de mouvement : (1) l image-clé dit aussi qui n est plus là — une
//	               liaison qu aucune de ses lectures ne porte est oubliée (un DEL non lu laissait la
//	               liaison du mort : `a0c36016`, 5 vies, M3.1 ; `grammar/keyframe_liaison.go`) ;
//	               (2) un NEW ne remplace pas une entité vivante — un NEW propre qui contredit une
//	               liaison en dur d un autre archétype n est pas lié (`0797ce72`, 11 vies ;
//	               `396cfc92`, 7 ; M3.2 ; `grammar/frame_infer.go`).
//	`coverage.`    `forgottenBindings`, `refusedNews` et son VERDICT NEUFS (constat DFIX-R6) : la
//	stances        croyance du monde peut être périmée (un DEL non lu), et un vrai NEW d un autre
//	               archétype est alors refusé à tort. L image-clé suivante tranche : elle redonne au
//	               slot l archétype du vivant (`refusedNewFalseReads`), celui du NEW
//	               (`refusedNewLostCreations` : le prix de la règle, rendu visible), ou aucun des
//	               deux (`refusedNewUndecided`).
//
// LA MESURE SUR LES SEPT BOBINES PAR BUILD : quatre images-clés changent (`bcb6d393` morceau 1
// paquets 0 et 3, `fb1a1a72` morceau 1 paquets 0 et 1), une réfutation chacune ; les records
// regagnés sont exactement ceux que la fausse ancre effaçait (1280..1298 trois fois ; 1537..1601,
// vingt-neuf équipements, armes au sol et objets, derrière la fausse ancre 1536 `ti 0`) ; 8 entités
// sur 8 lues au départ ; 0 doute. La mini-bobine `000d5950` n a pas de registre lisible (pas de
// `chunk_00`) : sa marche reste SANS preuve, et son paquet c1 p1 n est corrigé que sur le film
// ENTIER. L exigence de CONTENU de la preuve, re-mesurée (candidats d en-tête valide strictement
// entre deux records de slots consécutifs, sept bobines) : 39 471 candidats sûrement faux, 55
// fausses preuves sans l exigence, 0 avec. La recherche d en-têtes exacts : 0 doute avec la
// preuve, les seuls doutes sans elle sont le slot 1297 de ces deux films ; 3 ms par image-clé.
//
// CE QUE LE LOT CHANGE AUX ENTRÉES FIGÉES, DÉCLARÉ ET EXPLIQUÉ (constat DFIX-R3 ; diff des faits
// figés, aucun décodage de BTB) — une réfutation lit des records de plus, et les règles de BANDE
// existantes (combler la plage d un archétype, puis retirer tout slot OBSERVÉ sous un autre)
// s appliquent à ces observations :
//
//	`11de8353`     1 réfutation (records 14 902 -> 14 924). La fausse ancre réfutée était la seule
//	(BTB)          observation du slot 2048 sous un autre archétype : il rentre dans la bande des
//	               armes au sol (1 525 -> 1 526 slots). Les en-têtes NEW de ce slot sont un motif
//	               très fréquent des trames : ancres 14 697 -> 21 634, acceptées 515 -> 556, dont
//	               40 écartées par l IDENTITÉ et 1 retenue (une arme lâchée à une mort). `ti=9`
//	               753 -> 754 ; douze vies de power-up vues à une image-clé de plus.
//	`bcb6d393`     2 réfutations (+46 records). L image-clé regagnée (c1 p3, 8 662,1 s) porte les
//	               armes au sol 1590 et 1591 et les objets `ti=37` 1593, 1599 et 1600 : ils sortent
//	               de la bande de l AUTRE archétype (poses 593 -> 591 slots, vies d objet de la
//	               bande 184 -> 182, poses publiées inchangées ; armes au sol 582 -> 579). Les présences
//	               de power-up des socles 3, 7 et 10 sont prouvées une image-clé de plus (t0 223 :
//	               jusqu à 470 au lieu de 270), et une occupation de plus est datée (15 -> 16).
//	`b1f01a33`     (témoin) 2 réfutations (+30 records), même mécanisme : vies d objet 395 -> 390,
//	               bande des armes au sol 914 -> 907, power-ups acceptés 365 -> 363.
//	`000d5950`,    +1 et +2 records `ti=9` : les joueurs gérés regagnés.
//	`fb1a1a72`
//
// LE CODEC DES FAITS A CHANGÉ DANS LA VAGUE SANS SECONDE MONTÉE (constat DFIX-R5) : la santé des
// images-clés (`Doutes`), les réfutations, les preuves contradictoires et les cinq compteurs des
// états de mouvement s y ajoutent, sous les MÊMES noms de révision que la pré-intégration
// (`a85eaaf46`). Choix : la vague garde sa montée UNIQUE, et les faits de la pré-intégration sont
// PURGÉS — aucun n a été écrit dans le cache vivant (contrôlé en lecture : 0 fichier à
// `grammar-2026-09-24`), et les 67 fichiers de faits des racines temporaires qui les portaient
// (`rr-tmp-dfix`, racine d intégration du superviseur) ont été supprimés le 2026-09-24 avant toute
// reconstruction. Un fait de la vague ne peut donc se relire qu au codec de cette tête.
//
// LA MESURE AU PARC (racines temporaires, un film à la fois, aucun BTB). Cinq témoins cuits à la
// base (`ba475d2e4`), avant le lot (`ed0806a20`) et à la tête : le lot ne change que
// `coverage.keyframes` (records +18 / +18 / +38 / +9 / +30, réfutations 1 / 1 / 3 / 1 / 2), la
// présence du joueur géré de l index 0 de `000d5950` (dès la frame 0 : 4 contre 4 à chaque frame),
// les bandes d objets de `b1f01a33` ci-dessus et les états de mouvement. Règle des places tenue sur
// les cinq : au plus 4 fiches par équipe à chaque frame, 4 places par équipe, aucune place à deux
// fiches, 0 image-clé douteuse, 0 borne différée ; `b1ad85eb` 4 + 4 aux trois instants signalés
// (Eagle à 3 pendant 332 frames : la place vide d un relais, Q20). GATE DE CORPUS (onze témoins non
// BTB, base = tête de campagne) : règle des places tenue, 0 image-clé douteuse, 0 preuve
// contradictoire ; NEW refusés 1 248 = 331 lectures fausses confirmées + 1 création perdue
// (`fb1a1a72`) + 916 indécis (le slot absent de l image-clé suivante). ÉTATS DE MOUVEMENT,
// base -> tête, sur les seize films : aucune vie ne perd un intervalle, hors un accroupi d une
// frame de `c75f33b8` (0,1 s à 0,4 m/s) que l alignement du record NEW de M3.2 efface.
//
// LES SIX ÉCARTS DE LA PRÉ-INTÉGRATION, INSTRUITS SUR PIÈCES :
//
//	identitesHors- `c75f33b8` 0 -> 1 : le bot `343 Robot Hoida` (index 8, non épinglé par
//	Roster,        `killsource` avec dix humains) a une vie nommée sans entrée ; la base lui
//	tirsContestes  dessinait une place de plus, la tête aucune fiche et le compte le montre — sa
//	               correction est l épinglage, hors de la vague. `tirsContestes` 0 -> 1 : un
//	               arrivant dont les tirs désignent une place prise, posé par le chaînage nommé ;
//	               règle des places tenue (au plus 4 fiches par équipe, aucune place à deux).
//	entitesNonLiees `a349fea8` 25, `50247b26` 30 : films SANS section d identification (v33, v31),
//	               AUCUN roster (0 entrée à la base comme à la tête) — chaque entité lue reste sans
//	               entrée. Mesure d une impossibilité du film, pas une perte.
//	inventory.     +4 / +1 sur ces deux films : des lectures rattachées à un slot SANS trajectoire
//	unpublished    publiée (par définition, pas un échec de rattachement), qui suivent la hausse des
//	               lectures de M3 (969 / 984 et 650 / 668 publiées).
//	véhicules      PROUVÉ SUR DOCUMENTS (tête de la vague C décodée -> pré-intégration republiée de
//	fusionnés      ses faits, aucun décodage). `084a804d` : cinq vies fondues — 904, 905, 906 dans
//	               899, 900, 901 (même châssis, même point au centimètre, l hôte SANS AUCUN
//	               échantillon : un véhicule posé au pad, relayé 302 frames plus tard) ; 909 et 910
//	               dans 907 et 908, deux vies désormais lues ~300 frames plus tôt — 117 -> 114.
//	               `50247b26` : 829, 830, 831, 835, 836 dans 819, 820, 821, 825, 826 (hôtes sans
//	               échantillon, même châssis, 0,00 m, relais à la frame 4 052) — 64 -> 60. La marche
//	               de M3 lit l hôte à une image-clé de plus : sa borne d affichage couvre la naissance
//	               du relais, et la règle existante (`vehicle_relays.go`) s applique.
//	jauge du       `1c4c63c2` (BTB) : aucun document de base n existe sans décoder ce BTB (ses faits
//	drapeau,       de base ont été réécrits par l incident du 24/09) ; verdict par analogues — aucun
//	poses          des cinq CTF non BTB ne diverge sur ces étapes, et au parc les lectures montent
//	               sans perte (origines inconnues des poses 18 -> 11 sur `81c02726`, marques de
//	               porteur confirmées 5 -> 14 sur `a0c36016`). Contrôle au re-décodage de clôture.
//	weaponChanges  les prises « perdues » sont RECLASSÉES (même instant, même slot, même arme :
//	.taken         `swapped`, `from` = l arme de naissance) ou reconnues ré-annonces de la dotation,
//	               et un lâcher sans arme de `bfecd02b` que le relevé suivant dément disparaît :
//	               0 prise perdue sans contrepartie sur les seize films. C est le SENS de M3.
//	états de       gain net (`a0c36016` 1 432 -> 1 634 intervalles ; glissade de 41 s, escalade de
//	mouvement      30 s et accroupi de 43 s impossibles retirés) ; les pertes résiduelles de M3 sont
//	               corrigées ci-dessus.
//
//	CE QUI MONTE    rien de plus que la vague : `grammar.Rev`, `facts.Rev` et `SchemaDesFaits`
//	AVEC ELLE       gardent leur UNIQUE montée de la vague D, empreintes recopiées.
//
// v71 (2026-09-24, lot M4b de la campagne « retours rejeu », plan
// `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §4.5 « M4b — Tir des vehicules ») : LE TIR CONTINU,
// LU DANS LA VUE DE CONTROLE ET RENDU COMME THEATER. Une montee de plus que la vague D (70) : la
// FORME du document change (un calque racine et quatre blocs de couverture), et le garde-rail de
// forme refuse une forme nouvelle sous le numero de la vague.
//
//	`bursts`       NOUVEAU calque racine (document_fire_bursts.go, fire_bursts.go) : une rafale par
//	               gachette principale TENUE, lue dans l entree de controle du joueur (sonde P1-S3 :
//	               le jeu n ecrit le tir d une arme a type de prediction 1 et debit nul que par ce
//	               bit). Bornes au tick, projetees sur les frames ; l arme vient de la MONTURE (le
//	               chassis ou la piece montee de l episode, `tir_continu_armes.go`) ou de
//	               l EMPLACEMENT degaine a pied (Rayon de Sentinelle) ; la cadence est celle du TAG
//	               (Ghost 7,5/s, canons de la Banshee 8/s, Chopper 4/s, LMG du Wasp 10/s, LAAG
//	               5 -> 18/s, LMG du Falcon et mitrailleuse du Scorpion 13/s, tourelle du Wraith
//	               6 -> 12/s, Rayon 60/s). Les TROUS de lecture portes par une rafale sont publies,
//	               muets ; aucun balayage de repli (decision utilisateur du 2026-09-24).
//	`coverage.     NOUVEAU bloc : paquets lus et trous PAR CAUSE, rafales lues et le sort de
//	continuous-    chacune (publiee sur un vehicule ou a pied, ecartee par bit, joueur, monture sans
//	Fire`          arme, piste, arme inconnue ou arme a charge), coups poses (`shots`, le controle).
//	`coverage.     `shotsByUnit` / `shotsByUnitNoRide` : un tir dont la REFERENCE 0 (l unite
//	vehicles`      tireuse du record 36, lue par la grammaire) est un vehicule se pose sur LUI ;
//	               l episode ne sert plus qu a nommer l occupant (vehicle_shots_unit.go).
//	`coverage.     `tirsParPlace` : un tir est rendu a l OCCUPANT de sa place (le remplacant), plus
//	seats`         au partant dont la place porte l index (tirs_par_place.go) ; l index de tireur est
//	               lu sur CINQ bits (les places 16 a 31 d un BTB ne se confondent plus).
//
//	CE QUI MONTE    `grammar.Rev` garde son nom de vague (`grammar-2026-09-24`), empreinte
//	AVEC ELLE       recopiee (la marche lit l entree de controle complete, le bloc d action corrige,
//	                le record 36 par sa grammaire) ; `SchemaDesFaits` reste 4, le codec des faits
//	                monte (`REPLAYINPUTS27` : le canal du tir continu et les champs du record 36) ;
//	                `facts.Rev` inchangee. Republication apres RE-DECODAGE (verdict « redecoder »).
//
//	L INDEX DE      `coverage.seats.tirsIndexNonPlace` : l index de tireur n est lu comme place que
//	TIREUR JUGE     s il s accorde avec l unite tireuse (88,4 a 100 % sur sept builds, 0 % sur la
//	                build HI_1_4_1 de `a521164d`, ou le champ est constant) — sinon seule la
//	                reference 0 pose un tir (tirs_index_fiable.go, repli nomme).
//
// LA MESURE (reprise du 2026-09-25 apres la revue du lot ; racines temporaires, un film a la fois,
// aucun BTB decode en gate). LA LECTURE DE LA VUE B EST REPRISE (chronique de `grammar`, reprise
// M4b) : records NEW de tete des paquets a evenements, etat par defaut du projectile, queue du corps
// de mort, positions du corps d `i54`, six composants — vue C fermee `81c02726` 19,5 % -> 76,2 %,
// `8a485699` 9,0 % -> 63,2 %, `b1ad85eb` 26,5 % -> 89,4 %. Faits d equivalence, base `c9ef97ec6` ->
// tete, sur `000d5950`, `60ae07c4`, `fb1a1a72`, `d9781168` : bougent `fire` (memes comptes, champs
// neufs), `continuousFire` (neuve), `movementStates` (+1 a +11 % de lectures), `killsource` et
// `killRefs` (la marche des morts lit le corps de mort juste), `vehicles` et `birthLoadouts.stats`
// (`60ae07c4`) et `artifact` ; `positions` ne bouge pas. Gate de corpus (huit temoins non BTB) : les
// SEULES pertes sont des intervalles d etat et leur couverture — le sprint compte plus d intervalles
// pour moins de duree, l escalade finit plus tot — et l ORACLE PHYSIQUE du sprint (vitesse
// constante) s ameliore sur les huit (echantillons de sprint plus lents que la mediane hors sprint /
// echantillons hors sprint aussi rapides que la mediane de sprint, seuils de la base : `f75e7053`
// 18,5/16,6 % -> 11,5/16,2 %, `0797ce72` 14,0/14,2 % -> 6,6/12,5 %, `60ae07c4` 16,8/16,0 % ->
// 14,3/15,3 %, les cinq autres egaux ou meilleurs). Les deux accroupis de `f75e7053` (slot 585,
// 3655-3727 ; slot 530, 1761-1811) disparaissent : chacun chevauchait des sprints du meme slot —
// contradiction physique — et commencait pendant une escalade, dont le corps `i54` etait lu aux
// largeurs du delta. Les autres "pertes" du gate sont des compteurs de defaut qui baissent :
// `coverage.stances.refusedNews` sur les huit (`f75e7053` 317 -> 0), `eventPacketsUnlocated` sur les
// huit, `forgottenBindings` (`bf15f7ab` 411 -> 146), et `coverage.deathsPaths.directScan`, que la
// marche des morts releve. Tirs rattaches des fixtures de build : `111fa685` 88,7 -> 97,6 %,
// `11de8353` 86,7 -> 98,9 %, `e5adf7b2` 84,3 -> 96,8 %, `bcb6d393` 85,7 -> 97,3 %, `a521164d`
// 82,6 -> 92,3 % (par l unite ; l ancien rattachement y rendait les tirs au joueur 2, lu dans quatre
// bits d un champ constant) ; `b1ad85eb` 992 -> 1 061 (les remplacants). Tir continu, documents
// publies (gates G1 et G2 par famille, frags de vehicule du kill-feed) : `81c02726` une rafale dans
// les 2 s de chacun des 6 frags de G MONEY (0 au temoin -60 s, 0 hors monture) ; LAAG `1cd3848a` 3/3 ;
// canons de la Banshee `7b0d89c4` 13/14 et `8a485699` (index 7 : 4,2 s de rafale contre 37 tirs
// numerotes et dix touches) ; Chopper, LMG du Wasp et Rayon de Sentinelle publies sur `8a485699`,
// `1cd3848a` et `7b0d89c4` ; aucun document non BTB du parc ne porte de Falcon. Les coups publies
// restent sous les sauts du numero de tir. OUVERT : la 3e monture de G MONEY (`81c02726`, 2984-3021
// et 3104-3119) n est pas publiee — ses rafales sont lues, comptees `noTrack` (2) : l embarquement
// suit la naissance d un dispositif (`ti=43`, 2941) dont le record NEW desynchronise sur un composant
// non porte (`i21`), et chaque paquet suivant se clot sur son rejet.
