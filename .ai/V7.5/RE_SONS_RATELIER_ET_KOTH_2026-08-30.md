# RE — RATELIER D'ARMES, LACHER, ET LE POINT MARQUE EN ROI DE LA COLLINE

> Ecrit le 2026-08-30. Suite de `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`,
> `HANDOFF_SONS_RECONSTITUTION_2026-08-27.md` et `RE_GESTES_SONORES_2026-08-27.md`.
> Demande utilisateur : les sons qui manquent au rejeu — le point marque en **Roi de la
> colline** (allie et adverse), le **ramassage d'arme sur ratelier / equipement**, et
> (ajoute en cours de passe) le **lacher**.
> Planche republiee EN PLACE : artefact `6aadf3d5-7acf-4b98-a050-6bc88db55b7d`.

## 0. Ce que cette passe etablit, en une page

| demande | verdict | ou |
|---|---|---|
| KOTH — mon equipe marque | **`play_004_mod_mp_strongholds_scoring_tick_team`** (`fddf794f`, 3,62 s) | banque des zones `1c609526` |
| KOTH — l'adversaire marque | **`..._scoring_tick_enemy`** (`9a2a8880`, 4,35 s) | idem |
| ramassage d'arme sur socle | **`play_004_mod_mp_shared_weaponpad_empty`** (`f3595a2b`) | banque NEUVE `874fc1d1` (section 8) |
| lacher d'arme | **`play_006_chm_un_spartan_weapondrop`** (`6cdd92fd`, 0,31 s) | banque NEUVE `e9a52b26` |

Les deux premieres lignes reposent sur un NEGATIF (section 2) et restent a confirmer a
l'oreille ; les deux dernieres sont des noms casses, avec leur esperance.

**AMENDEMENT DU MEME JOUR (section 8)** : l'utilisateur ECARTE `scoring_tick_team/_enemy` a
l'ecoute — « ce sont les sons de Bastion ». La ligne KOTH redevient donc OUVERTE, et la
seconde passe a ouvert trois banques neuves ou chercher. Elle a aussi trouve MIEUX pour le
ramassage : le **socle** (`weaponpad`) plutot que la **capsule** de largage (`weaponpod`).

## 1. OUTILLAGE — il avait disparu, il est reecrit

`C:\Users\Guillaume\Downloads\Halo Infinite - Sons v75\` **n'existait plus** sur ce poste
(rendus, `_outils/{rendu_geste,planche_gestes}`, `gestes/noms_evenements.json`), et
`C:\Users\Guillaume\Downloads\vgmstream\` non plus. Le dossier est RECREE avec la chaine
reecrite ; `vgmstream-cli.exe` a ete repris de
`Downloads\Halo Infinite - Sons armes\_outils\vgmstream\`, ou il vit toujours.

	_outils/cracker/    casse des noms : composition d'un vocabulaire curie (`casser`),
	                    gabarit a un jeton inconnu contre le dictionnaire du binaire
	                    (`gabarits`), moisson sur toutes les banques nommees (`moisson`),
	                    regroupement des evenements par JEU DE MEDIAS (`groupes`),
	                    noms de banques candidats (`banques`).
	_outils/rendu/      un geste -> un .wav : couches sommees a leur gain de chemin, delai
	                    d'action, nombre de lectures, mode 5 (cadence de declenchement),
	                    crete a -1,0 dBTP mesuree en DEUX PASSES.
	_outils/assembler/  met a jour la planche EN PLACE (garde ses cartes et leurs audios).

**LA PLANCHE PUBLIEE EST UNE SAUVEGARDE.** Elle a ete relue par `WebFetch` et c'est d'elle
qu'on a recupere les 430 cartes et leurs audios apres la perte du disque. A savoir la
prochaine fois qu'on croit un rendu perdu.

**CONTROLE DE REPRODUCTIBILITE, avant toute nouvelle mesure.** Le rendu reecrit rejoue six
gestes deja publies le 2026-08-27 et retrouve leur duree au centieme :

	fddf794f 3,63 s (planche 3,62) | 9a2a8880 4,35 (4,36) | 71cb04b8 5,15 (5,15)
	d8a2fcb8 2,89 s (2,89)         | 1badec8a 6,00 (6,00) | c3327c0b 1,18 (1,18)

## 2. ROI DE LA COLLINE — LE NEGATIF, ET SON DENOMINATEUR

**IL N'EXISTE AUCUN ESPACE DE NOMMAGE `koth` DANS LE JEU.** Le dictionnaire du binaire
(138 886 jetons, chaines ASCII decoupees sur les separateurs ET les frontieres camelCase)
confronte aux gabarits `play_004_mod_mp_koth_<jeton>` et
`play_004_mod_mp_kingofthehill_<jeton>` :

	cibles = les 1 275 evenements de `common`  -> esperance 0,0825 -> 0 resultat
	cibles = les 6 142 evenements de `globals` -> esperance 0,1986 -> 0 resultat

Le jeton `koth` EST dans le dictionnaire (`mp_cinematic_koth`, `page_objectives_koth`), et
`kingofthehill` aussi (`KingOfTheHill`, `MultiplayerKingOfTheHill`, `king_of_the_hill`) :
le negatif ne vient pas d'un dictionnaire trop pauvre. Aucune banque ne s'appelle non plus
`sb_004_mod_mp_koth` (39 noms candidats confrontes aux 1 496 identifiants Wwise des deux
modules, esperance 1,1e-5 ; la passe retrouve au passage `sb_004_mod_mp_ctf`,
`sb_004_mod_mp_oddball` et `sb_004_mod_mp_landgrab` — c'est sa calibration).

**CE QUE LE JEU FAIT A LA PLACE.** La banque des zones `1c609526` porte 88 evenements pour
49 jeux de medias : le meme geste y est declare une fois par mode. Deux espaces de nommage
seulement s'y cassent, et le second est neuf :

	play_004_mod_mp_strongholds_*     contested, contested_win, contested_lose,
	                                  zone_exit_team/enemy, scoring_tick_team/enemy
	play_004_mod_mp_suddendeath_*     contested, contested_win, contested_lose,
	                                  zone_spawn            <- NEUF (2026-08-30)

Gabarit `play_004_mod_mp_<jeton>_contested` et six autres suffixes connus, dictionnaire
complet, esperance 0,0199 : **seuls `strongholds` et `suddendeath` sortent**. Deux autres
espaces restent hors d'atteinte (les groupes `contested*` ont QUATRE evenements freres) ;
un jeton de mode compose de deux mots n'a pas ete tente.

**CONCLUSION, ET SA LIMITE.** Roi de la colline joue les evenements de cette banque. Le
seul couple `mon equipe` / `adversaire` qui parle de SCORE y est
`scoring_tick_team` / `_enemy`. C'est donc, a la mesure, le son du point marque en KOTH —
mais le nom dit « tic », et le mode ou le jeu le reposte n'est pas lisible dans la banque.
**La designation finale revient a l'oreille** (`RECETTE_SONS_ARMES` section 5) : la planche
porte une section `A DESIGNER` avec ces deux cartes servies ENTIERES et douze candidats de
forme (les deux autres enchainements de la banque, les stingers isoles de 3 s, la paire
miroir 20/21).

**CE QUE CA CHANGE POUR LE REJEU.** Les fichiers livres `objective_zone_tick_team/enemy`
sont une COUPE a 1,2 s attenuee a -12 dBTP — reglage voulu pour un tic PAR SECONDE en
Bastion (`RE_GESTES_SONORES` 10.3). En KOTH un point tombe toutes les 37 a 50 s
(`PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` 1.1) : il faut une SECONDE paire, entiere et au
niveau nominal, et un declencheur distinct. Les deux `.wav` sont rendus
(`rendus/zone_point_team.wav`, `zone_point_enemy.wav`).

## 3. LE RATELIER D'ARMES A SA PROPRE BANQUE — `sb_004_mod_mp_shared_weaponpod`

`8a6cb59b`, module `common`, **12 evenements, 28 medias, 11 jeux distincts**. Elle n'avait
jamais ete ouverte : le balayage de 79 noms de banque du 2026-08-27 ne la nommait pas.

	23b14d65  play_004_mod_mp_shared_weaponpod_empty              2,53 s   <- L'ARME EST PRISE
	4d97cbb6  play_004_mod_mp_shared_weaponpod_incoming           1,70 s
	d6c21ad1  play_004_mod_mp_shared_weaponpod_slam               4,17 s
	3b959c5f  play_004_mod_mp_shared_weaponpod_slam_dirt          4,17 s
	71a16091  play_004_mod_mp_shared_weaponpod_slam_water         4,17 s
	17ea3194  play_004_mod_mp_shared_weaponpod_electricity_open   1,06 s
	2f6afc6b  play_004_mod_mp_shared_weaponpod_electricity_slam   (memes medias)

Esperance : 0,0031 pour la premiere passe (8 gabarits), 0,0310 pour la seconde (80).
Quatre evenements resistent (`13939519`, `16ee5f4d`, `ba6c962b`, `c9bd1f74`) ; un cinquieme
(`e5aa4a9a`) n'est pas rendu — son media `462473905` n'est pas embarque dans la banque, il
vit dans un `.pck` du disque.

**RESERVE A DIRE.** `empty` est le son du ratelier VIDE, pas le geste du joueur. C'est le
plus proche que le jeu offre d'un « arme prise sur ratelier », et c'est une ECOUTE qui doit
le confirmer. Le ramassage d'EQUIPEMENT, lui, est deja identifie et livre :
`play_007_abl_shared_pickup` (`c73036e4`) = `objective_pad_pickup.wav` = « Equip 18 » de la
planche.

## 4. LE LACHER — `play_006_chm_un_spartan_weapondrop`

Banque `sb_006_chm_un_spartan` (`e9a52b26`, module `globals`, **230 evenements**), jamais
ouverte non plus. Le nom se casse a esperance 0,0446 sur la moisson de la famille `sb_006`.
Une couche, une variante parmi trois, gain -6 dB, **0,31 s**.

Les 229 autres evenements de cette banque resistent : vocabulaire curie de 165 mots a deux
jetons (esperance 0,0103) et douze gabarits a deux emplacements (0,0893) ne rendent rien de
plus. Le bruitage du Spartan a une grammaire de noms qui n'est pas celle des modes.

## 5. VINGT NOMS DE PLUS, en prime — moisson sur les banques nommees

Passe unique `moisson` (grammaire `play_<nom de banque sans sb_>_<jeton>[_<modulation>]`,
dictionnaire complet, esperance PAR BANQUE de 0,0004 a 0,0171, cumul 0,126) :

	play_004_mod_mp_vip_kill_team / _kill_enemy / _killed_team / _killed_enemy
	play_004_mod_mp_elimination_revive_loop / _levelup
	play_004_mod_mp_attrition_levelup       play_004_mod_mp_escalation_levelup
	play_004_mod_mp_infection_playerspawn
	play_007_abl_repairfield_active_loop    play_007_abl_shroud_appear
	play_007_abl_evade_blast_player
	play_002_ui_menu_tacmap_{open,close,focus,pan_loop,setwaypoint,intelreceived}
	play_002_ui_menu_forge_{undo,redo,error}

**Les quatre noms VIP sont directement utiles au chantier des modes porteurs** : la couronne
VIP est livree, ses sons ne l'etaient pas.

## 6. NEGATIFS DE LA PASSE, chacun avec son denominateur

1. **Pas d'espace `koth`** — section 2 (esperances 0,0825 et 0,1986).
2. **Pas de banque `sb_004_mod_mp_koth`** — 39 noms x 1 496 banques, esperance 1,1e-5.
3. **Les banques d'INTERFACE ne livrent aucun nom** : `sb_002_ui_global` (29 evenements),
   `sb_004_mod_mp_shared_ui` (20), `sb_004_mod_mp_shared_global` (9), confrontees a 51
   gabarits chacune (esperances 0,048 / 0,033 / 0,015) : **0 resultat**. Le negatif du
   2026-08-26 sur `shared_ui` est donc CONFIRME et etendu — ces banques ont une autre
   grammaire.
4. **Les banques d'animation d'arme `sb_006_chm_ge_weaanim_*` ne livrent rien** (moisson,
   esperance cumulee 0,0144).
5. **Les 79 autres evenements de la banque des zones resistent** : 30 espaces de nommage x
   dictionnaire complet en suffixe (esperance 0,0854) et 33 gabarits a deux emplacements sur
   `strongholds` (0,0939) ne rendent que les noms deja connus.

## 7. CE QUI RESTE OUVERT

1. **La designation a l'oreille** du point marque en KOTH (section `A DESIGNER` de la
   planche). Sans elle, le cablage repose sur une inference, pas sur une mesure.
2. **Le cablage lui-meme** : le rejeu ne joue pas de son sur les points KOTH. Le declencheur
   existe (bornes de periode de colline = instants de score, mesure 4 films sur 4,
   `PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` 1.1) ; `zoneSound.ts` MUET le tic en KOTH par une
   garde volontaire. Il faut une regle distincte, pas lever la garde.
3. **Le ratelier n'a aucun declencheur** dans le document de rejeu : rien ne date la prise
   d'une arme sur un socle. Meme famille de blocage que `padPickups` (intervalle, pas
   instant). Travail de DECODEUR.
4. **Le lacher n'a pas de declencheur non plus** — le film ne publie pas les changements
   d'inventaire a la seconde (voir `RE_GESTES_SONORES` 8.1 : c'est une derive de curseur, pas
   une absence de grammaire).
5. **S1 (`hsc*`) et S4 (tag de mode)** restent non lancees ; ce sont les deux dernieres voies
   de NOMMAGE pour la banque des zones.
6. **`e5aa4a9a`** du ratelier : media non embarque, a extraire du `.pck`.

---

## 8. SECONDE PASSE DU 2026-08-30 — le nom des banques ANONYMES, et « hill »

**Deux retours utilisateur, tous deux justes** : (a) « Zone 11 et Zone 12 ne collent pas a
KOTH, ce sont les sons de strongholds » ; (b) « des centaines de sons dans ce fichier, comment
je m'y retrouve ». La premiere rouvre la question, la seconde condamne la planche de 430
cartes comme instrument de travail.

### 8.1 « hill » — le negatif s'etend, et il tient

Gabarits `hill` dans TOUTES les positions plausibles (17 formes : `_hill_%s`, `%s_hill`,
`%s_hill_team/_enemy`, `strongholds_hill_%s`, `suddendeath_hill_%s`, `koth_hill_%s`, ...),
dictionnaire complet, cibles = les 88 evenements de la banque des zones, esperance 0,0484 :
**0 resultat**. Aucune banque `sb_004_mod_mp_hill_*` ni `sb_004_mod_mp_koth_*` non plus.

**Le jeton de mode des 6 freres non nommes** (les groupes `contested`, `contested_win`,
`contested_lose` ont chacun QUATRE evenements, dont deux seulement sont nommes) : 425 gabarits
`play_004_mod_mp_%s_<mot>[_modulation]` batis sur le vocabulaire des zones, dictionnaire
complet, cibles reduites aux 6 evenements, esperance 0,0825 : **0 resultat**.

**Un piege de discipline, vecu et utile a citer.** La meme passe lancee SANS reduire les
cibles (88 au lieu de 6) donne une esperance de 1,21 et sort
`play_004_mod_mp_hcp0_hills_enemy` pour `ad147b8f` — un evenement DEJA nomme
`..._strongholds_zone_exit_team`. C'est une collision fortuite, exactement ce que le seuil
existe pour ecarter. Le filtre par evenement a ete ajoute a l'outil dans la foulee.

### 8.2 LE NOM DES BANQUES ANONYMES SE CASSE AUSSI — 9 banques nommees d'un coup

Ce que la passe du 2026-08-26 n'avait pas tente : mettre le DICTIONNAIRE DU BINAIRE dans un
gabarit de **nom de banque**, et non plus un dictionnaire de mots choisis. Cibles = les 1 495
identifiants Wwise des deux modules ; une forme a la fois, esperance 0,0483 chacune.

	sb_004_mod_mp_%s          -> sb_004_mod_mp_bts               7c9b300f  18 evts  NEUVE
	                             (+ ctf, academy, assault, vip, oddball, infection,
	                              extraction, elimination, attrition, landgrab = calibration)
	sb_004_mod_mp_shared_%s   -> sb_004_mod_mp_shared_weaponrack 3ae17c71   4 evts  NEUVE
	                             sb_004_mod_mp_shared_weaponpad  874fc1d1   5 evts  NEUVE
	                             sb_004_mod_mp_shared_ping       de65048f   7 evts  NEUVE
	                             sb_004_mod_mp_shared_razorback  c0cc62ba   2 evts  NEUVE
	                             sb_004_mod_mp_shared_droppod    2b0c8838   0 evt   NEUVE
	                             sb_004_mod_mp_shared_ui         ee694c9e  20 evts  (confirmee)
	                             sb_004_mod_mp_shared_ai         586f11c8  11 evts  (confirmee)
	sb_002_ui_%s              -> sb_002_ui_s02                   f40b289f  22 wem   NEUVE

**LE RAMASSAGE D'ARME A DONC TROIS FAMILLES, ET LA BONNE EST LE SOCLE** :

	socle    sb_004_mod_mp_shared_weaponpad   f3595a2b  _empty            <- LA REPONSE
	                                          54bd9e43  _appear
	                                          5698355a  _hologram_loop
	ratelier sb_004_mod_mp_shared_weaponrack  4108ad49  _appear
	                                          f3e906da  _steam
	capsule  sb_004_mod_mp_shared_weaponpod   23b14d65  _empty  (largage BTB)

Esperance de la moisson qui les casse : 0,0182 cumulee sur 9 banques.
Marqueurs : `..._shared_ping_{danger,friendly,neutral,cancel}`.

**NEGATIF** : le nom de la banque des zones `1c609526` RESISTE toujours — 117 gabarits a deux
jetons (`sb_004_mod_mp_<mot>_%s`, `sb_004_mod_%s_<mot>`, `sb_004_mod_mp_%s_<mot>`) sur 39 mots
choisis, cible unique, esperance 0,0038 : 0 resultat. Les six plus grosses banques anonymes de
`common` (`b6397afe` 86 wem, `de5c463a`, `b50a1365`, `05112cbf`, `54c81b75`, `b46f48e9`)
resistent aussi a 504 gabarits `sb_<NNN>_<famille>_%s` chacune (esperance 0,0096 par banque).

### 8.3 LA GRAMMAIRE DES EVENEMENTS D'INTERFACE, enfin trouvee — par son temoin en clair

`sb_002_ui_global` (21 sons, 29 evenements) ne livrait AUCUN nom sous la grammaire habituelle
`play_<nom de banque sans sb_>_<jeton>`. La raison : **le nom de la banque n'est pas la base de
ses evenements**. Le gabarit `play_002_ui_menu_global_%s_open` rend
`play_002_ui_menu_global_tutorialpopup_open` (`e5c1afc7`) — et ce nom est l'un des TROIS noms
d'evenement Wwise presents EN CLAIR dans le binaire du jeu
(`RE_BANQUES_SONORES_NOMMEES` section 6, negatif n°2). La grammaire est donc confirmee par un
temoin exterieur, pas seulement par le hachage.

Les 20 autres evenements resistent (38 gabarits sur la base confirmee, esperance 0,0356). Ils
sont RENDUS et servis a l'oreille : c'est la piste la plus serieuse restante pour le point
marque, puisque c'est la que vivent les stingers d'interface.

### 8.4 CE QUI EST LIVRE

**Une planche NEUVE et COURTE**, a une adresse a elle (l'ancienne, `6aadf3d5`, reste le
catalogue complet) : `226618d1-d351-45ee-bbbb-6f64162a155c` — **68 sons**, un par question :

	Socle 5 | Ratelier 4 | Capsule 10 | Lacher 1
	Interface globale 21 | Banque `bts` 18 | Marqueurs 6      <- les 45 candidats KOTH
	Reperes 3 (les deux tics ecartes, servis ENTIERS, + l'ancre `zone_new`)

68 rendus neufs, tous produits par la chaine reecrite en section 1, controle de
reproductibilite compris.

---

## 9. TROISIEME PASSE — GHIDRA, ET LA REPONSE : ROI DE LA COLLINE N'A PAS DE SON DE SCORE

> Question utilisateur : « Ghidra peut pas te dire ou et quel est le son quand l'adversaire ou
> l'allie marque des points sur KOTH ? au moins savoir quel nom est utilise dans la banque ? »
> Reponse : Ghidra ne donne pas le nom — mais il dit OU CHERCHER, et l'endroit qu'il designe
> livre la reponse en clair.

### 9.1 Ce que Ghidra REFUTE, et son denominateur

	3 chaines "hill" dans 95 Mo de binaire   MultiplayerKingOfTheHill, KingOfTheHill,
	                                         king_of_the_hill — aucun vocabulaire audio
	2 chaines "koth"                         page_objectives_koth, mp_cinematic_koth
	identifiants d'evenement Wwise           0 occurrence en octets sur 3 temoins de plus
	                                         (fddf794f, 9a2a8880, 71cb04b8), qui s'ajoutent
	                                         aux 3 du 2026-08-26 : 6 temoins, 0 resultat

`king_of_the_hill` n'est meme pas un nom d'evenement : `FUN_140250480` en calcule le
**murmur3** (`FUN_140748a74`, constantes `0x1b873593` / `0x85ebca6b` / `0xc2b2ae35`) pour en
faire un `string_id` de tag. Rien d'audio.

### 9.2 Ce que Ghidra ETABLIT — le champ s'appelle `SoundEventHash`

`FUN_1408786f0` est le chemin de lecture d'un son. Sa telemetrie nomme elle-meme les champs de
l'objet runtime du `snd!` :

	+0x0c   string_id du tag        -> "SoundTagName"
	+0x14   identifiant Wwise       -> "SoundEventHash"
	+0x18   identifiant joueur      -> "SoundPlayerEventHash"

L'identifiant d'evenement vient donc du TAG, pas du code : le negatif du 2026-08-26 est
confirme par la structure, et pas seulement par une recherche d'octets.

### 9.3 LA DECOUVERTE : les tags `hsc*` sont du LUA COMPILE, NOMS EN CLAIR

La sonde S1, ouverte depuis le 2026-08-27 et jamais lancee, est faite. `remonter` depuis la
banque des zones rend 7 tags `hsc*` ; un vidage brut de leurs chaines montre qu'un `hsc*` n'est
pas du bytecode HaloScript anonyme mais **du Lua compile qui garde ses noms** :

	globals/scripts/global_multiplayer.lua        globals/scripts/global_multiplayer_init.lua
	globals/scripts/global_multiplayer_medals.lua globals/scripts/global_academy.lua
	scripts/parcellibrary/parcel_mp_weapon_placement.lua   (228 chemins .lua au total)

**357 scripts vides** (151 dans `any/globals/common-rtx-new`, 206 dans
`any/globals/globals-rtx-new`). C'est un ESPACE DE NOMMAGE NEUF, sans hachage, sans esperance
de collision : les noms sont ecrits.

### 9.4 LA TABLE DES REFERENCES DE TAG PAR MODE, telle que le jeu l'ecrit

Dans `global_multiplayer.lua` (`hsc* a35c6ce9`), un bloc liste les references de tag de chaque
mode. Extrait litteral, dans l'ordre du fichier :

	Bastion            CapturingLoopEnemy / CapturingLoopTeam
	                   ReverseCapturingLoopEnemy / ReverseCapturingLoopTeam
	                   ScoringLoopEnemy / ScoringLoopTeam      <- LE SON DE SCORE EST ICI
	                   PlateScoreVFX
	Strongholds
	LandGrab
	Extraction         ZoneExtractingLoop{Enemy,Team} / ZoneHackLoop{...} / ZoneConvertingLoop{...}
	Stockpile          StealLoop, StockpileCompleteFX, HackCompleteFX, TerritoryCompleteFX
	KingOfTheHill      HillContestedSound
	                   HillMovedSound                          <- ET C'EST TOUT
	TotalControl       ControlledLoopEnemy / ControlledLoopTeam

**TROIS CONCLUSIONS, chacune lisible directement :**

1. **L'utilisateur avait raison, et le jeu le dit avec ses propres mots.** Le son de score
   allie/adverse est un `ScoringLoop{Team,Enemy}` declare par **Bastion** — c'est exactement le
   couple `..._strongholds_scoring_tick_team/_enemy` ecarte a l'oreille. Il n'appartient pas a
   Roi de la colline.
2. **ROI DE LA COLLINE NE DECLARE QUE DEUX SONS**, et aucun n'est un son de score :
   `HillContestedSound` et `HillMovedSound`. La recherche d'un « son du point marque en KOTH »
   est donc **CLOSE PAR UN NEGATIF DU JEU LUI-MEME** : il n'existe pas.
3. **Ce qui marque le point, c'est le DEPLACEMENT DE LA COLLINE.** `HillMovedSound` est le son
   deja identifie a l'oreille par l'utilisateur — `71cb04b8`, « Zone 10, avant l'apparition
   d'une nouvelle zone ». En KOTH la colline tourne quand quelqu'un marque
   (`PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` 1.1, 4 films sur 4) : ce son EST le point.
   **Et il n'a pas de camp** — ni `Team` ni `Enemy` dans son nom, contrairement a ceux de
   Bastion. Le jeu ne distingue pas allie et adverse sur ce geste.

**CE QUE CA DECIDE POUR LE REJEU** : un seul son sur les points KOTH, sans camp, pose a la
bascule d'intervalle `active` — c'est-a-dire exactement la regle « nouvelle colline » deja
ecrite dans `zoneSound.ts`, sauf qu'elle EXCLUT aujourd'hui le premier intervalle. Le premier
intervalle n'est pas un point ; les suivants, si. La regle est donc DEJA la bonne.

### 9.5 Le socle et le ratelier, confirmes par le Lua

Le meme vidage donne la structure `MPItemSpawnerAudioAssets` :

	weaponHologramLoop
	__OnWeaponPadIncomingSound     __OnWeaponPadReadySound     __OnWeaponPadPickedUpSound
	__OnWeaponRackReadySound       __OnWeaponRackPickedUpSound
	callback EVENTS.onItemPickedUp

**Le ramassage sur socle EXISTE comme evenement de jeu nomme** (`onItemPickedUp`), avec un son
dedie par famille. Cote banque, `weaponpad` porte 5 evenements dont 3 nommes (`_appear`,
`_hologram_loop`, `_empty`) : la correspondance `PickedUp` <-> `_empty` reste une ECOUTE, les
deux evenements non nommes du socle etant les autres candidats.

### 9.6 Outillage — sonde jetable, supprimee du depot

`cmd/tmp_tagdump` (extraction d'un tag brut + listage de ses chaines ASCII) a servi puis a ete
SUPPRIMEE du depot. Sa source et les deux vidages complets sont archives hors depot :
`Downloads\Halo Infinite - Sons v75\_outils\tagdump\` et `_donnees\lua\`.

### 9.7 CE QUE CETTE PASSE OUVRE

Le Lua est un espace de nommage **sans hachage** que personne n'avait ouvert : 357 scripts,
228 chemins de fichiers, les noms de toutes les fonctions et de tous les champs de reference de
tag. Tout ce que les passes precedentes cherchaient par hachage (medailles, modes, objets
d'objectif, equipements) y est probablement ECRIT. C'est la piste a suivre avant toute nouvelle
passe de composition.
