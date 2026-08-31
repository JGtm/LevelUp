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

### 9.8 CORRECTION, LE MEME JOUR — la section 9.4 concluait trop

**L'utilisateur refute** : « les sons de score KOTH sont pas les memes que Strongholds/Bastion ».
Il a raison, et l'erreur est identifiable.

**CE QUI EST FAUX DANS 9.4** : j'ai lu le **pool de constantes** d'un Lua compile comme s'il
etait la declaration exhaustive de l'audio d'un mode. Il ne l'est pas, pour deux raisons :

1. **Un pool de constantes DEDUPLIQUE.** Si la table de Roi de la colline porte une cle deja
   internee par celle de Bastion (`ScoringLoopTeam`, par exemple), elle n'apparait qu'UNE fois,
   sous Bastion. L'absence d'une cle sous `KingOfTheHill` ne prouve donc rien.
2. **Le script ne nomme que ce dont LE SCRIPT a besoin.** Le moteur poste des evenements que le
   Lua n'ecrit jamais — c'est meme le cas general.

**CE QUI RESTE VRAI, et c'est mesure** : `ScoringLoop{Team,Enemy}` est declare sous `Bastion` ;
`KingOfTheHillInitArgs` ne porte que `instanceName`, `kingOfTheHillObjectNames`
(`threehold_{blue,neutral,red}`), `hillContestedAnnouncerVOTag`, `hillMovedAnnouncerVOTag` ; le
binaire ne porte ni vocabulaire audio de colline ni identifiant Wwise. Rien de tout cela ne
constitue une preuve d'absence.

**RESERVE SUPPLEMENTAIRE, qu'il faut dire** : j'avais aussi assimile `ScoringLoop{Team,Enemy}`
du Lua aux evenements `..._strongholds_scoring_tick_{team,enemy}` de la banque. Les deux se
ressemblent par le sens mais **pas par la forme** : le Lua dit BOUCLE, et ces deux evenements
n'en sont pas (2 couches simultanees, 3,6 et 4,4 s). Le rattachement n'est pas etabli.

**CE QUE LA MESURE APPORTE A LA PLACE — cinq PAIRES MIROIR non attribuees.** Dans la banque des
zones, cinq couples de groupes de medias ont la MEME FORME EXACTE et des medias disjoints —
signature d'un couple `_team` / `_enemy` (les couples nommes du drapeau se comportent ainsi) :

	P1  8061054a / 6d4b6ad4   1 son EN BOUCLE, 1,00 s de silence entre deux lectures
	P2  1c21bc2d / 222abfa1   2 couches simultanees (une couche commune, une distincte)
	P3  259a15f2 / 59d1f744   3 couches, deux en boucle
	P4  93f632c0 / dcf980a5   boucle a +1,50 s, 0,50 s de silence entre deux lectures
	P5  1badec8a / af31554f   boucle, une variante parmi cinq

**P1 est le candidat de forme le plus fort** : une boucle avec une seconde de silence entre
chaque lecture est exactement ce qu'est un tic de score, et c'est ce que le mot `Loop` du Lua
decrit. Les dix sons sont rendus et servis cote a cote en tete de la planche courte
(`226618d1`), section « Roi de la colline — les paires a designer ».

**LECON DE METHODE, a garder** : un espace de nommage neuf (le Lua) donne du VOCABULAIRE, pas
des NEGATIFS. Une absence dans un pool deduplique n'est pas un denominateur.

### 9.9 LES SONS SERVIS SONT RECONSTITUES — et la paire 1 est MUETTE dans le jeu

Question utilisateur : « des sons isoles ou reconstitues a partir du package et du format
Wwise ? » **Reconstitues.** Le format donne, par geste, quatre grandeurs que le rendu applique :

	COUCHES simultanees      sommees, chacune a son GAIN DE CHEMIN (evenement -> Sound)
	DELAI de l'action        AkPropID 15, en millisecondes entieres
	NOMBRE DE LECTURES       `sLoopCount`, 0 = boucle infinie
	RYTHME des lectures      `eTransitionMode` (bout a bout, silence, cadence de declenchement)

Exemple, `259a15f2` (paire 3) : trois couches simultanees a -6 dB chacune, dont DEUX en boucle
infinie bout a bout. Un `.wem` isole n'est pas ce son.

**Choix d'ecoute, qui ne viennent pas du jeu** : une variante tiree par couche, toujours la meme
(le plus petit identifiant) ; une boucle infinie servie sur 6 s avec un fondu de sortie de 0,5 s ;
crete ramenee a -1,0 dBTP. **Non reproduit** : bus, effets et reverb Wwise, fondus pilotes par un
parametre de jeu (evalues a leur point de repos), distance. La fourchette RANGED, elle, est nulle
sur ces familles (0 sur 275 couches, mesure du 2026-08-26).

**LE DEFAUT QUE LA QUESTION A FAIT SORTIR.** Le releve des gains des dix candidats donne :

	paire 1  8061054a / 6d4b6ad4   -96 dB          <- CONVENTION WWISE DU MUET
	paire 2  222abfa1 / 1c21bc2d   +3/+2 et -3 dB
	paire 3  259a15f2 / 59d1f744   -6 dB
	paire 4  dcf980a5 / 93f632c0   +5 dB et -1 dB
	paire 5  1badec8a / af31554f   -5 dB et -12 dB

**Les deux sons de la paire 1 sont a -96 dB : le jeu ne les joue pas sur ce chemin.** Ce qu'on
entend n'existe que parce que la normalisation a -1 dBTP les ressuscite — le meme piege que
celui documente le 2026-08-27 (11 rendus sur 135 rendus silencieux par une couche a -96 dB, en
`pcm_s16le`). La paire 1 etait donnee en 9.8 comme le candidat de forme le plus fort ; elle
passe DERRIERE les paires 2 et 4, dont les gains sont ceux d'un son reellement audible. Elle
reste servie, marquee comme muette, parce que sa forme reste celle d'un tic de score.

**REGLE A GARDER** : relever le GAIN DE CHEMIN avant de proposer un candidat a l'ecoute. Une
normalisation de crete rend audible ce que le jeu tait.

---

## 10. DESIGNATIONS DE L'UTILISATEUR — 2026-08-30, a l'oreille

L'oreille de l'utilisateur vaut une mesure (`RECETTE_SONS_ARMES` section 5). Ce qui suit est
DESIGNE, donc acquis :

	SECURISATION DE LA COLLINE, ALLIEE     93f632c0   (paire 4 B, gain -1 dB)
	SECURISATION DE LA COLLINE, ADVERSE    dcf980a5   (paire 4 A, gain +5 dB)
	SPAWN D'ARME SUR SOCLE                 54bd9e43   play_004_mod_mp_shared_weaponpad_appear
	LACHER D'ARME                          6cdd92fd   play_006_chm_un_spartan_weapondrop

**REGLE DE CABLAGE, donnee avec la designation et a ne pas perdre** : le son de securisation
« doit etre PROLONGE le temps que la securisation est en cours ; une boucle relancerait le debut
du son, qui met du temps a se lancer — c'est comme une sirene ». Le rejeu doit donc l'ETIRER sur
la duree de la garde, pas le redeclencher. C'est une contrainte de rendu cote app, pas une
propriete de la banque : la banque, elle, declare bien une boucle (`sLoopCount` = 0,
`eTransitionMode` = 3, 0,5 s de silence entre deux lectures, entree a +1,50 s).

**LA PORTEE DU SPAWN D'ARME EST CONFIRMEE PAR LE LUA.** L'utilisateur precise « donc que les
armes speciales, pas les armes sur rateliers » ; la table `MPItemSpawnerAudioAssets` de
`global_multiplayer.lua` est effectivement gardee par `MP_WEAPON_TIER.Power`. La designation et
la structure disent la meme chose.

## 11. CE QUI RESTE, ET OU CHERCHER

### 11.1 Le RAMASSAGE D'ARME est introuvable — et ce n'est pas qu'il n'existe pas

L'utilisateur : « je n'ai rien vu qui correspond ». Le Lua prouve pourtant que le geste EXISTE
comme evenement de jeu nomme : `__OnWeaponPadPickedUpSound`, `__OnWeaponRackPickedUpSound`,
callback `EVENTS.onItemPickedUp`. Aucun des 5 sons du socle ni des 4 du ratelier n'y correspond
a l'ecoute.

**Piste suivante, et elle est designee par la symetrie** : le lacher vit dans
`sb_006_chm_un_spartan` (bruitage du Spartan, **230 evenements, UN SEUL nomme**). Un ramassage
est le geste symetrique d'un lacher ; il est vraisemblablement dans la meme banque. Le cassage
par gabarit y est epuise sur un jeton (dictionnaire complet, esperance 0,0074 par forme) et sur
deux (165 mots curie, 0,0103) : la voie restante est le RENDU des 230 evenements et l'oreille,
ou le Lua du bruitage s'il existe un script dedie.

### 11.2 Le SPAWN D'EQUIPEMENT SUR SOCLE — demande neuve, sons servis

Le Lua dit ou chercher : `MPEquipmentPlacement` (`hsc* b1cdc4ba`) utilise la MEME structure audio
que les socles d'armes — `MPItemSpawnerAudioAssets`, `GetHologramLoopingSound`,
`GetIncomingEffect`, `GetSpawnedEffect`, `MessagingIncoming` / `Ready` / `PickedUp`. La banque
commune des equipements est `sb_007_abl_shared` (`15c5b355`, **15 evenements, 20 medias**), dont
seuls deux sont nommes (`_pickup`, `_device_explode`).

**Les 15 sont rendus et servis** sous la question du spawn (section « Socle d'equipement » de la
planche `226618d1`). Les gains de chemin sont indiques carte par carte, parce que plusieurs sont
tres attenues (-22 a -26 dB) et que la normalisation le masque — lecon de 9.9.

Les 13 evenements non nommes resistent a 53 gabarits sur le dictionnaire complet (esperance
0,0257).

---

## 12. ETAT AU SOIR DU 2026-08-30 — ce qui est trouve, ce qui manque

### 12.1 Les cinq gestes DESIGNES

	securisation de la colline, ALLIEE     93f632c0   banque des zones 1c609526
	securisation de la colline, ADVERSE    dcf980a5   idem
	spawn d'arme sur socle                 54bd9e43   ..._weaponpad_appear (armes speciales)
	spawn d'equipement sur socle           4093f3c4   sb_007_abl_shared, wem 905776253, -5 dB
	lacher d'arme                          6cdd92fd   play_006_chm_un_spartan_weapondrop

Plus deux acquis anterieurs : le ramassage d'EQUIPEMENT (`c73036e4`, livre depuis le
2026-08-27 en `objective_pad_pickup.wav`) et le deplacement de colline (`71cb04b8`).

### 12.2 IDENTIFICATION — il ne manque qu'UN son

**Le RAMASSAGE D'ARME.** C'est le seul de la demande initiale qui resiste. Le Lua prouve le
geste (`__OnWeaponPadPickedUpSound`, `__OnWeaponRackPickedUpSound`, `EVENTS.onItemPickedUp`) ;
aucun son du socle, du ratelier ni de la capsule n'y correspond a l'oreille.

Piste lancee : `sb_006_chm_un_spartan`, la banque qui porte le lacher — **230 evenements,
107 jeux de medias, 105 rendus** (2 echecs de decodage), servis tries par duree croissante en
section « Bruitage du Spartan » de la planche. Le hachage y est epuise (un jeton sur
dictionnaire complet ; deux jetons sur 165 mots curie).

### 12.3 LIVRAISON — rien n'est fait, et les declencheurs sont inegaux

Aucun des cinq sons n'est dans `static/sounds/halo_infinite/` ni cable. Etat des declencheurs
dans le document de rejeu, verifie sur pieces :

	securisation alliee / adverse   DISPONIBLE  `zoneStates[].gauge` (rampes) + `owner`
	                                            REGLE : etirer sur la duree de garde, pas
	                                            boucler (note utilisateur, sirene)
	spawn d'arme sur socle          DISPONIBLE  `weaponPads[].spawns` publie les INSTANTS
	                                            d'apparition (`document_ground_weapons.go`)
	ramassage d'arme                ABSENT      `padPickups` donne un INTERVALLE [tLow, tHigh]
	                                            et `xuid` est TOUJOURS nul (oracle a 88,1 %,
	                                            sous le seuil de 90 % du plan)
	spawn d'equipement sur socle    ABSENT      le document publie `equipmentPlacements`
	                                            (la POSE par un joueur, T0), pas les socles
	                                            d'equipement
	lacher d'arme                   ABSENT      aucun canal date de changement d'inventaire

**Deux des cinq sont cablables tout de suite** ; les trois autres demandent du DECODEUR, pas du
son.

### 12.4 ONZE CANDIDATS pour le ramassage — le filtre est la SIGNATURE du lacher

Servir 105 sons a l'oreille n'est pas une proposition, c'est un renvoi. Le filtre honnete
existe : le lacher a une signature precise dans le format — **une couche, une variante parmi
trois, gain -6 dB, 0,31 s**. Les gestes de la banque qui portent la MEME signature (une couche,
1 parmi 3 ou 4, gain de -7 a -5 dB, duree de 0,18 a 0,50 s) sont **onze** :

	07944717 0,21 s -6    2ac4aa71 0,22 -6    2e04e44f 0,22 -6    26fa42d3 0,23 -6
	448be652 0,23 s -7    7730a533 0,28 -6    7b4299cf 0,29 -6    0bb6e5c1 0,30 -5
	04b6dac8 0,31 s -5    168832f6 0,34 -6    53b4f802 0,45 -6

Ils sont servis en section propre, AVANT le reste de la banque, avec le lacher en tete pour
comparaison. Les 94 autres restent servis derriere, tries par duree, si aucun ne colle.

### 12.5 LE RAMASSAGE D'ARME EST DESIGNE — mais PAR DEFAUT, et il faut l'ecrire

`168832f6` (« Candidat 10 »), banque `sb_006_chm_un_spartan` : une couche, une variante parmi
trois, gain -6 dB, **0,34 s**.

**LE MOT DE L'UTILISATEUR EST « faute de mieux ».** Cette designation est donc plus FAIBLE que
les cinq autres : aucune ne s'est imposee a l'ecoute, celle-ci a ete prise par elimination. La
consigner comme un acquis de meme rang que les autres serait mentir sur la mesure.

**CE QUI LA SOUTIENT TOUT DE MEME** : le retenu porte EXACTEMENT la signature du lacher, son
geste symetrique, mesuree dans le format — une couche, une variante parmi trois, gain -6 dB — et
il dure 0,34 s contre 0,31 s. Sur les 107 jeux de medias de la banque, onze seulement portent
cette signature, et c'est parmi eux qu'il a ete pris.

**CONDITION DE REPRISE** : si le rendu final sonne faux en situation, les dix autres candidats
de meme signature restent servis dans la section « Ramassage — les autres candidats » de la
planche, et le reste de la banque (94 rendus) derriere.

### 12.6 LES SIX GESTES, ETAT FINAL DE L'IDENTIFICATION

	securisation de la colline, ALLIEE     93f632c0   designe
	securisation de la colline, ADVERSE    dcf980a5   designe
	spawn d'arme sur socle                 54bd9e43   designe + nom casse (_weaponpad_appear)
	spawn d'equipement sur socle           4093f3c4   designe
	lacher d'arme                          6cdd92fd   designe + nom casse (_spartan_weapondrop)
	ramassage d'arme                       168832f6   designe PAR DEFAUT (12.5)

**L'IDENTIFICATION EST CLOSE.** Ce qui reste n'est plus du son : c'est la livraison des six
`.wav` dans `static/sounds/halo_infinite/`, le cablage, et pour trois d'entre eux un
DECLENCHEUR qui n'existe pas encore dans le document de rejeu (12.3).

---

## 13. CABLAGE — les deux gestes qui avaient deja leur declencheur

Livre le 2026-08-30, apres les designations : ce qui pouvait sonner tout de suite sonne.

### 13.1 La SECURISATION de la colline — `zoneSound.ts`

	declencheur   `ZoneSpan.active` + `owner` (le seul canal qui parle en KOTH)
	fichiers      objective_zone_securing_team / _enemy   5,50 s chacun
	plancher      ZONE_SECURING_MIN_MS = 3000 (un intervalle plus court est un TRANSFERT)

**POURQUOI PAS LA JAUGE, ET POURQUOI LES DEUX REGLES NE SE MARCHENT PAS DESSUS.** `capturing`
nait d'une rampe de jauge ; or `ZoneState.Gauge` est **TOUJOURS ABSENTE sur une colline**
(`document_zones.go` : le canal y est un compteur de transfert d'environ une seconde,
`coverage.zones.gaugePoints` vaut 0). La disjonction est donc tenue par la SOURCE, pas par une
garde a maintenir.

**DEUX ECARTS ASSUMES sur le rendu**, tous deux ecrits dans le code :

1. **Le delai d'action de 1,5 s est retire.** Il est l'entree en boucle du jeu ; servi en tete
   d'un one-shot il n'ajouterait que du silence.
2. **5,5 s ne couvrent pas une garde de 40 s.** C'est le plafond des sons d'EVENEMENT
   (`LONG_MAX_S` = 6 s dans `replaySoundAssets.guard.test.ts`), et le tenir vaut mieux que de
   laisser un son de match grimper vers le plafond des fanfares (12 s). Le premier rendu faisait
   11,5 s et **le garde-rail l'a refuse — c'est son role**. Prolonger davantage est une decision
   PRODUIT (relever le plafond des evenements), pas une decision de livraison.

### 13.2 L'APPARITION SUR SOCLE — `padSpawnSound.ts` (fichier neuf)

	declencheur   `weaponPads[].spawns` — les INSTANTS, publies par le calque
	fichier       objective_pad_spawn   1,409 s
	plafond       PAD_SPAWN_MAX_PAR_MATCH = 300

Fichier separe de `padSound.ts` pour la meme raison qui a separe `zoneSound.ts` de
`objectiveSound.ts` : la source et la doctrine different. `padSound` DEDUIT le ramassage d'un
premier tir ; ici le calque DATE l'apparition, il n'y a rien a deduire, et il n'y a pas de camp.

**LA PORTEE « ARMES SPECIALES » EST TENUE PAR LA SOURCE**, pas par un filtre : `doc.weaponPads`
ne liste que des SOCLES, les rateliers n'y sont pas. Le Lua dit la meme chose de son cote
(`MPItemSpawnerAudioAssets` gardee par `MP_WEAPON_TIER.Power`).

### 13.3 Gates

	vitest src/features/match-replay   121 fichiers, 1 883 tests   VERT
	npm run typecheck (cache purge)                                VERT
	eslint sur les 5 fichiers touches                              VERT

12 tests neufs, dont **9 qui epinglent des SILENCES** (colline neutre, camp non resolu,
transfert trop court, intervalle possede hors colline, socle sans apparition).

### 13.4 CE QUI N'EST PAS CABLE, ET POURQUOI — dont le ramassage d'EQUIPEMENT

**Le ramassage d'EQUIPEMENT n'a pas de son dans le rejeu, et l'utilisateur a raison de le
relever.** Le son EXISTE et est livre (`objective_pad_pickup.wav` = `c73036e4` =
`play_007_abl_shared_pickup`) — mais il est **consomme par la regle des ARMES** : `padSound.ts`
le joue au premier tir d'une famille d'arme de socle. Deux choses manquent, et elles sont
distinctes :

1. **Le stem est pris.** Le jour ou le son de ramassage d'ARME (`168832f6`, designe par defaut)
   sera livre, `padSound.ts` doit basculer dessus — ce qui rend `objective_pad_pickup` a son
   sens propre. Ce lot est confie a un autre agent (indication utilisateur du 2026-08-30) ;
   ce fichier ne l'a pas touche.
2. **Le declencheur n'existe pas.** Le document ne publie aucun ramassage d'equipement pose :
   `equipmentPlacements` date la POSE par un joueur (`T0`), `equipmentEpisodes` date
   l'ACTIVATION de camouflage / surbouclier. Aucun des deux n'est « untel a pris l'equipement du
   socle ». Travail de DECODEUR.

Restent egalement sans declencheur : le **spawn d'equipement sur socle** (`4093f3c4` designe, le
document ne publie pas les socles d'equipement) et le **lacher d'arme** (`6cdd92fd`, aucun canal
date de changement d'inventaire).

### 13.5 LE SPAWN D'EQUIPEMENT SUR SOCLE — ce que la recherche ajoute, et ce qu'elle contredit

Demande de verification de l'utilisateur (2026-08-30, apres sa propre designation `4093f3c4`).
Trois mesures, dans l'ordre ou elles reduisent le champ :

1. **IL N'EXISTE AUCUNE BANQUE DE SOCLE D'EQUIPEMENT.** Gabarit `sb_004_mod_mp_shared_%s` sur
   les 138 886 jetons du binaire, cibles = les 1 495 identifiants Wwise (esperance 0,048) :
   `weaponpad`, `weaponrack`, `weaponpod`, `ping`, `razorback`, `droppod`, `ui`, `ai`, `global`.
   Rien pour l'equipement.
2. **LE LUA DIT POURQUOI.** `MPEquipmentPlacement` (`hsc* b1cdc4ba`) ne porte QUE deux noms
   audio, et ce sont ceux du socle d'ARME : `MPItemSpawnerAudioAssets` et
   `GetHologramLoopingSound`. Les kits Forge du placement d'equipement et de bonus
   (`0d380ade`, `e31760a1`) n'en portent AUCUN — leurs reglages exposes sont visuels
   (« Incoming Visual FX », « Spawned Visual FX »). Le socle d'equipement reutilise donc la
   table audio du socle d'arme.
3. **ET LA REMONTEE DE TAGS VA DANS LE MEME SENS.** Depuis `sb_007_abl_shared` (`15c5b355`) :
   13 `snd!` au niveau 1 ; au niveau 2, **21 `eqip`**, 31 `sofa`, 6 `effe` et 2 `luas`. Les sons
   de cette banque pendent donc a des OBJETS D'EQUIPEMENT, pas a un spawner. (Les deux `luas`
   sont des tables de sons indexees par script — meme forme que le precedent `gggl` ; en tirer
   un sens demanderait de decompiler le bytecode, non fait.)

**CE QUE CA VEUT DIRE, ET LA RESERVE.** La structure pousse vers « le spawn d'equipement sonne
comme le spawn d'arme » (`play_004_mod_mp_shared_weaponpad_appear`, deja designe pour les armes)
— pas vers un son propre. L'utilisateur a pourtant designe `4093f3c4` a l'oreille, et une
designation vaut une mesure. **Les deux ne se contredisent pas forcement** : le socle peut jouer
le son du spawner ET l'objet pose emettre le sien. Ce qui reste vrai dans tous les cas : cette
recherche n'a produit AUCUN candidat neuf. Les seuls candidats sont les 15 sons de
`sb_007_abl_shared` (tous rendus, tous sur la planche) et les deux sons de socle d'arme encore
non identifies (`84377e39`, `8d891c5f`, eux aussi sur la planche).

---

## 14. LE RAMASSAGE SUR UN EVENEMENT DE TIR — l'aberration, dite comme telle

L'utilisateur, 2026-08-30 : « mettre un son de ramassage sur un event de tir ça ne te choque pas
comme aberration ? ». **Si, et il faut l'ecrire ici plutot que de le laisser dans un en-tete de
fichier ou personne ne le relit.**

**CE QUE FAIT `padSound.ts` AUJOURD'HUI** : il joue `objective_pad_pickup` au PREMIER TIR d'une
famille d'arme qui appartient a un socle du match, une fois par couple (joueur, famille).

**SES DEUX DENOMINATEURS, et ils sont vrais** : `padPickups` a une mediane de 20,00 s entre
`tLow` et `tHigh` (3,2 % sous 2 s) ; les changements de loadout vivent sur la meme grille
d'images-cles (0 sur 597 dates a moins de 5 s). Aucun canal ne DATE le ramassage.

**MAIS LA CONCLUSION QU'ON EN A TIREE EST FAUSSE, et elle contredit la doctrine du chantier.**
La regle ecrite partout ailleurs est : *le rejeu se TAIT plutot que de deviner*. Ici on n'a pas
choisi le silence, on a deplace le son sur un AUTRE evenement. Trois consequences audibles :

1. le son de ramassage part EN MEME TEMPS qu'un tir — deux gestes differents, un seul instant ;
2. il part pour une arme prise au sol ou sur un mort, pas seulement sur un socle ;
3. il ne part JAMAIS pour une arme ramassee et non tiree.

**LES TROIS SORTIES POSSIBLES, a trancher par l'utilisateur** :

	A. SE TAIRE          retirer la regle jusqu'a ce qu'un canal date le ramassage.
	                     Coherent avec la doctrine, coute un son de moins.
	B. BORNER AU SOCLE   poser le son a `padPickups[].tLow` — l'instant le plus tot ou le socle
	                     a PU se vider. Ce n'est plus un tir, c'est le socle qui parle ; le prix
	                     est une imprecision pouvant aller a 20 s.
	C. CROISER LES DEUX  garder le premier tir, mais SEULEMENT s'il tombe dans l'intervalle
	                     `[tLow, tHigh]` du socle de cette famille. On perd les ramassages hors
	                     socle (consequence 2) et on garde la precision ; restent les tirs
	                     tardifs, qui ne sonneront pas.

**CE FICHIER N'A PAS ETE TOUCHE** : le lot du ramassage d'arme est confie a un autre agent
(indication utilisateur du 2026-08-30). La decision A/B/C lui revient, ou a l'utilisateur.

## 15. PLANCHE DES SOCLES ET DES EQUIPEMENTS

Artefact `3c84fab7-5e36-4777-a2d9-bd1c90b08f65` — **135 sons, treize banques, trois familles**,
en depliants (`<details>`), demande de l'utilisateur pour naviguer.

	Distribution des armes    socle (5) | ratelier (4) | capsule de largage (10)
	Equipement, commun        sb_007_abl_shared (15)
	Equipements un par un     champ de reparation (9) | ecran occultant (11) | grappin (24)
	                          repulseur (11) | camouflage (5) | surbouclier (4) | propulseur (2)
	                          translocateur (23) | capteur et traqueur (12)

Chaque ligne porte son gain de chemin, sa forme, le nombre d'evenements qui jouent ce meme
materiau, et son nom Wwise QUAND il est casse — 33 le sont sur 135. Ce qui est deja livre dans
le rejeu ou designe par l'utilisateur est marque.

**QUATRE SONS SUR 139 NE SONT PAS RENDUS** (trois du champ de reparation, un de la capsule) :
leur media n'est pas embarque dans la banque, il vit dans un `.pck` du disque. Ils sont ABSENTS
de la page plutot que servis silencieux — et le nombre est ecrit en pied de page.

### 14.1 DECISION : OPTION A — le rejeu se tait

Tranchee par l'utilisateur le 2026-08-30. `padSound.ts` et son test sont **SUPPRIMES**, l'appel
retire de `replaySound.ts`, le stem retire du garde-rail, et **l'asset
`objective_pad_pickup.wav` retire du depot** — sans quoi le garde-rail « 0 asset mort » serait
rouge, et un fichier que rien ne joue est exactement ce qu'il traque.

**CE QUI N'EST PAS PERDU, et ou le retrouver** : le son reste IDENTIFIE
(`play_007_abl_shared_pickup`, evenement `c73036e4`, banque `sb_007_abl_shared`), audible sur la
planche `3c84fab7`, et re-livrable en une commande depuis les `.wem` archives. Git garde le
fichier. **Condition de reprise** : un canal qui DATE le ramassage — c'est-a-dire soit un
`PadPickup.XUID` renseigne (l'oracle plafonne a 88,1 %, sous le seuil de 90 % du lot), soit un
resserrement de `[tLow, tHigh]` (mediane 20,00 s aujourd'hui).

**UN COMMENTAIRE DE HUIT LIGNES REMPLACE LA REGLE** dans `replaySound.ts`, a l'endroit exact ou
elle vivait : ce qu'elle faisait, les deux mesures qui la justifiaient, pourquoi la conclusion
etait fausse, et ou retrouver le son. Un retrait qui ne laisse pas de trace se refait.

Gates apres retrait : `vitest src/features/match-replay` **120 fichiers / 1 880 tests VERT**,
`typecheck` cache purge VERT, `eslint` VERT.

### 14.2 « Pourquoi ces sons sont sur la planche si on ne les a pas ? »

Question de l'utilisateur, et elle pointe une IMPRECISION DE MA PART qu'il faut corriger ici.
J'ai ecrit « on n'a pas le ramassage d'equipement » et « on n'a pas le spawn d'equipement » ; ce
que je voulais dire etait « ils ne sonnent pas dans le rejeu ». **Ce sont deux choses
differentes, et la planche montre la premiere** :

	LE SON            existe, est reconstitue, est sur la planche, est parfois nomme
	L'INSTANT         n'existe pas dans le document de rejeu — rien ne date le geste

Pour ces deux-la, le SON est acquis (`c73036e4` pour le ramassage, `4093f3c4` pour le spawn,
designe a l'oreille) et c'est l'INSTANT qui manque. Dire « on ne l'a pas » melangeait les deux.
La regle d'ecriture, desormais : nommer lequel des deux manque.

---

## 16. LE MERGE DU CHANTIER RAMASSAGE CHANGE LA DONNE — les quatre sons muets sont cables

Le merge `dcbc6e458` (schemas 25-28) apporte exactement les declencheurs qui manquaient en 12.3 :

	weaponChanges      prises / lachers / echanges d'arme, dates a la frame, par vie
	                   (les re-annonces de spawn ECARTEES cote Go — pas des ramassages)
	equipmentChanges   ramassages (`taken`) et consommations (`spent`) d'equipement, dates
	groundWeapons      armes au sol individuelles, fins OBSERVEES

Le lot web du chantier avait branche l'AFFICHAGE, pas le son. Cable ce jour :

	weaponChangeSound.ts      taken -> weapon_pickup (168832f6) | dropped -> weapon_drop
	(neuf)                    (6cdd92fd). Un SWAPPED sonne le ramassage SEUL — un echange est
	                          un lacher et une prise au meme instant, superposer les deux
	                          fichiers ferait un artefact de mixage, pas deux gestes.
	equipmentChangeSound.ts   taken -> objective_pad_pickup (c73036e4, RE-LIVRE — retire le
	(neuf)                    matin meme faute de declencheur, il en a un le soir). `spent`
	                          MUET : consommer = utiliser, et l'usage sonne deja par sa
	                          famille (episodes camo/surbouclier, poses mur/capteur, traction
	                          de grappin) — un jingle generique DOUBLERAIT ces sons.
	padSpawnSound.ts          scinde par famille : socle d'EQUIPEMENT (powerup_camo /
	(scinde)                  powerup_overshield, la jointure de l'affichage
	                          `padEquipmentFamilyOf`) -> equipment_pad_spawn (4093f3c4,
	                          designe « Equip 7 ») ; socle d'ARME -> objective_pad_spawn.

**LE SPAWN D'EQUIPEMENT N'A PAS DEMANDE D'EVENEMENT NEUF**, et l'utilisateur avait raison de le
dire « deja gere, en tous cas partiellement » : `powerup_pads.go` publie les socles de power-up
DANS le canal des socles, avec leurs apparitions datees (`spawns`). La limite du « partiellement »
est ecrite : seuls camouflage et surbouclier sont des familles de socle publiees — un equipement
pose hors socle n'a pas d'apparition datee.

**Fichiers livres** (rendus du jour, dans `static/sounds/halo_infinite/`) :

	weapon_drop.wav          0,312 s   weapon_pickup.wav        0,340 s
	equipment_pad_spawn.wav  0,562 s   objective_pad_pickup.wav 0,804 s

**Gates** : vitest match-replay **125 fichiers / 1 934 tests VERT** ; typecheck cache purge
VERT ; eslint VERT. Tests neufs : le choix du `swapped`, le silence du `spent`, la scission
arme/equipement des socles.

**BILAN DU CHANTIER SON AU SOIR DU 2026-08-30 — les six gestes designes SONNENT** :
securisation de colline (x2), spawn d'arme sur socle, spawn d'equipement sur socle, lacher,
ramassage d'arme, plus le ramassage d'equipement re-livre. Il ne reste AUCUN son designe sans
declencheur.

**CI DE CLOTURE (2026-08-30 soir)** : run du head `656b6d9ce` VERT AU NIVEAU JOB — Go Build+Test
ubuntu et windows, Coverage+Baseline, Frontend, Contract, lints, Lease, Deploy Pre-Check. Le rouge
herite du merge (contrat non regenere) est solde par `656b6d9ce` ; le lot son `10593604a` est CLOS.
