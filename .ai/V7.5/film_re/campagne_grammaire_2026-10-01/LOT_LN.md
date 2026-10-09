# LOT LN — Les naissances lues par la grammaire de la vue A (vague 2, 2026-10-04)

> Campagne de grammaire, vague 2 (GO de l'utilisateur du 2026-10-04), décision de l'utilisateur du
> 2026-10-02 (liste « retenues, vague 2 ») : « les naissances uniquement par la grammaire (lecture
> des messages de la vue A), sans condition par film ». Sources : PLAN §6.2 L1 (L1b), `R_NAIS.md`,
> `R_LOC.md` §4 (R-L1 (c)), D-44, D-46.
>
> Worktree `LevelUp-wt-cg2-ln`, branche `feat/cg2-ln`, base `2393d7db7` (= `feat/v75` =
> `feat/campagne-grammaire`, vague 1 fusionnée). Référence de mesure : `2393d7db7`.
>
> Conventions de la campagne : **lu** = lu dans Ghidra (`HaloInfinite.exe` HI_1_13_0, base
> 0x140000000, serveur HTTP du plugin, lecture seule : `decompile_function`, `read_memory`,
> `search_instructions`, `get_xrefs_to`) ; **mesuré** = compté sur les 20 films ; **estimé** = dérivé
> d'une mesure par une hypothèse écrite. « Sain » = paquet fermé que le juge de L0 ne contredit pas
> (colonne `ferme` de la carte v2) ; « factice » = fermé au bit près et contredit.

## 0. Statut

**[!] non retenu en l'état (contrôle du 2026-10-04, correction C1)** : le gate 6
(`replay-corpus-gate` sans perte) n'est pas tenu. Le gate sort en rc 1 et ses pertes FILET ne sont
pas instruites (§8 bis, D-LN-15). Le gate 2 (carte v2, par film, au net) est tenu ; les 1 589
sains perdus bruts ne sont PAS établis comme des fermetures factices (§5), l'exception D2 n'est donc
pas invoquée. Deux voies de levée, aucune n'est franchie à ce jour :

- (a) instruire paquet par paquet les pertes FILET : la durée des postures (`c75f33b8` 11 986 ->
  8 253, −31 % ; corpus 381 752 -> 356 922), la rafale perdue de `111fa685` (6 -> 5) et ses 6
  disparitions, `holesOpenViewB` (15 300 -> 16 831), en distinguant une posture COUPÉE à tort d'une
  posture ÉTENDUE à tort sur un trou de lecture ;
- (b) une décision datée du pilote ou de l'utilisateur qui admet ce rc 1, consignée au §3 du plan
  comme D23 l'a été pour la vague 1. Aucune n'est consignée au 2026-10-04 ; le message de
  l'utilisateur du 2026-10-04 relayé à l'exécution des corrections (« J4ai regardé les films c'est
  nickel !!! ») ne porte pas sur ce gate et n'en tient pas lieu.

Option de conception soumise au pilote (non appliquée, non mesurée en carte) : ne prendre le début E
de la vue A que si la marche partie de E ferme, sinon garder la recherche par fermeture. Mesure du
témoin (§4.1) : sur 27 276 listes non localisées lues jusqu'au terminateur, 10 800 ferment depuis E.

Ce qui suit décrit le lot tel qu'il est, corrections du contrôle comprises (§11).

L'étape 1 établit une grammaire LUE : la table des 123 genres de message est
enregistrée à adresses constantes dans l'exécutable, chaque genre porte ses domaines de référence et
son lecteur de charge, et 54 lecteurs de charge sont portés depuis leur fonction du jeu (tous les
genres dont la lecture ne dépend que du flux, de l'exécutable et du film, sauf les plus rares). La
vue A d'un paquet à événements se lit jusqu'à son terminateur quand chacun de ses messages est de
ces genres ; le bit qui suit est le premier bit de la vue B. L'étape 2 branche cette lecture dans
`localiserLaListe`, là où le localisateur strict ne trouve rien.

Carte v2, 20 films : **paquets sains 313 495 → 341 105 (+27 610), records utiles sains
2 922 510 → 3 446 684 (+524 174), AUCUN film en baisse** (ni en paquets sains, ni en utiles sains) ;
0,4 % de gains fermés au bit contredits ; 1 589 sains perdus bruts, tous « devenus non fermés »,
NON expliqués (§5) et compensés au net dans leur film. Listes non localisées 47 854 → 20 101. Killsource identique à l'octet sur les 19
témoins. Indicateur D1 (dénominateur fixe consolidé 7 758 290) : corpus 37,7 % → 44,4 %, HI_1_13_0
76,6 % → 85,0 %.

## 1. Ce qui est lu dans le jeu

### 1.1 La table des genres (nouveau ; réfute R_LOC §4.1)

`R_LOC.md` §4.1 disait « la table des gestionnaires est construite à l'exécution
(`DAT_144e61d88 + 0x210`) : le lien genre -> désérialiseur ne se lit pas dans l'image statique ».
**C'est faux** :

- `FUN_14054d014` pose `DAT_144e61d88 = DAT_144db4358 + 0x8aed0`, puis appelle `FUN_140e453b4` ;
- `FUN_140e453b4` (`ln_ghidra/FUN_140e453b4.c`) écrit les 123 descripteurs à
  `param_1 + 0x210 + genre * 8` par des affectations d'ADRESSES CONSTANTES (`&PTR_PTR_1447xxxxx`),
  et rend le compte `0x7b` (rang `+0x208`). Le lecteur `FUN_14080a9d4` lit le même tableau :
  `*(obj[0x18] + 0x210 + genre * 8)`.
- Chaque descripteur est un objet de 8 octets dont le seul champ est sa vtable (`read_memory`) :
  `+0x08` nom (stubs `48 8D 05 rel32 C3` décodés à la main), `+0x10` taille de la structure du
  message, `+0x58` domaine de la référence i, `+0x60` écrivain, `+0x68` lecteur.

La table complète (genre, nom, domaines, descripteur, lecteur, vide) est `ln_ghidra/table_genres.tsv`
(vtables : `vtables.tsv`). Contrôles : `biped_board_vehicle` = 8, vtable `0x143d0d330`, `+0x58` =
`0x142f1556c` (déjà lu par `event_list.go`) ; `unit_exit_vehicle` = 22, `+0x58` = `0x14080a018` ;
`action_weapon_fire` = 36 ; `unit_zoom` = 21 ; `EquipmentSpawnedObject` = 103. Les noms sont ceux de
la numérotation « trame » du dépôt.

**Domaines** : les 30 fonctions `+0x58` sont des cascades `85 D2 74 .. 83 EA 01 ..` (dix avec un
bloc froid lointain `0F 85 rel32` -> `83 EA 01 74/0F 84 ..`), décodées octet par octet
(`ln_ghidra/domfn.txt`). La table obtenue est IDENTIQUE, genre par genre, à celle du lot R7
(`r7_grammaire_research_test.go`), dérivée par un autre interprète.

**Genres vides** : `+0x10` rend 0 (`33 C0 C3`) pour exactement 13 genres (3, 4, 23, 24, 25, 26, 33,
49, 54, 57, 59, 92, 103), dont le lecteur est `FUN_1408d8220` (`return 1`). Dans `FUN_14080a9d4`, une
taille nulle ne lit AUCUNE charge ni le contrôle de corruption.

### 1.2 Le message (`FUN_14080a9d4`) et la vue A (`FUN_14076a1c4`)

```
vue A :   { R(1) ; 0 -> fin ; message }
message : genre = R(7) ; genre >= 0x7b -> refus
          pour i = 0, 1, 2 : R(1) garde ; si 1 : FUN_1406d3140(domaine vtable+0x58(i))
          si taille (vtable+0x10) > 0 :
              vtable+0x68(..., param_5 = 1) ; faux -> refus
              si FUN_14076cea8() : R(1) ; si 1 : R(32)
```

- `FUN_1406d3140` est porté par `readVarWidthInt` (`varwidth.go`, `FUN_140d10bb0` : domaines 0, 1,
  7, 8 sur 13 bits, 2, 3, 5 sur 8, 4 et 6 sur 9 ; la sonde du domaine 1 bascule sur l'entrée 4).
- **Le bit de configuration (correction C2).** `FUN_142987460` lit `DAT_144706104 =
  FUN_1406cf008(reader)` avant les trois vues (relu le 2026-10-04). `FUN_1406d3140` (relu) ne prend
  la plage de la catégorie dans la table de `FUN_140d10bb0` que si ce bit vaut 1 ; à 0, la plage est
  `DAT_144706100` pour toutes les catégories, et la sonde de la catégorie 1 se lit sans basculer
  (`param_3 == 1 && R(1) && DAT_144706104`). La première version du lot SAUTAIT ce bit
  (`br.Skip(bitsDeConfiguration)`). `parcourirLaVueA` le LIT désormais ; à 0, la lecture refuse
  (`arretSansTableDesCategories`) : le lecteur et les charges qui lisent une référence ne portent
  que la table par catégorie. Choix du refus plutôt que de la plage par défaut : porter la plage par
  défaut demanderait de passer le bit à chaque charge qui lit une référence (genres 0, 36, 119,
  120), pour un cas que la carte ne rencontre pas (mesure au §11).
- `FUN_14076cea8` rend le bit `chunk_00 + 0xCB45C` du film (`ControleDeCorruption`).
- `param_5 = 1` : toute branche `param_5 == 0` d'un lecteur de charge est morte dans le film.
- L'écrivain (`FUN_142f2c3b0` -> `FUN_142f2c050`, `FUN_140bbd474`) recopie les messages sans préfixe
  de longueur puis pose le terminateur `0` ; la vue B commence au bit suivant (R_LOC §4.1, relu).

### 1.3 La version de chaque genre : celle du jeu, celle du film

- `FUN_141102ed0(i)` : en rejeu de film (`FUN_1404f2b4c`, `+0xea71c == 2`), `FUN_1428e1c64` rend
  `film + 0xCB208 + i * 4` — la table par type de `chunk_00` (`profile.FilmIdentity.TypeVersions`) ;
  hors film, `DAT_14474cd90 + i * 8`.
- `DAT_14474cd90` est une table de 123 paires {version, taille de la structure} (`read_memory`,
  984 octets, `ln_ghidra/versions_natives.txt`) dont la taille est nulle EXACTEMENT pour les 13 genres
  vides : **l'index de la table par type du film est le GENRE de message**. Dix lecteurs consultent
  leur propre version (`get_xrefs_to(0x141102ed0)` : genres 35, 36, 40, 48, 89, 90, 91, 93, 97, 114).
- Mesuré (`ln_tsv/versions_des_genres_par_film.tsv`) : HI_1_12_0 et HI_1_13_0 déclarent 123 genres
  aux versions natives ; HI_1_9_0 à HI_1_11_0 en déclarent 121 ou 122, versions natives (préfixe) ;
  HI_1_8_0 déclare les genres 35 et 36 en versions 3 et 2 (natives 4 et 3) ; HI_1_4_1 a une table de
  116 décalée (genres insérés) ; version-31 et version-33 n'ont pas de section d'identification.

**Règle (`vue_a_versions.go`)** : la vue A d'un film se lit quand le film déclare au plus 123 genres,
chacun à sa version native (la table du film est un préfixe de celle du jeu) ; un genre au-delà du
cardinal du film arrête la lecture. Pourquoi l'égalité et pas « la version du film passée au
lecteur » : le lecteur du jeu ne porte que les branches de SA version (`FUN_14080c1f8` sépare
`version < 2` de `>= 2`) ; mesuré, le genre 36 de HI_1_8_0 (version 2) lu par le lecteur de la
version 3 ne retombe JAMAIS sur le début localisé (0 / 228 listes à un seul message). La règle ne
nomme aucun build ni aucun film : elle compare deux tables, l'une lue dans l'exécutable, l'autre
dans le film. Effet : HI_1_8_0, HI_1_4_1, version-31, version-33 (4 films sur 20 : `60ae07c4`, `a521164d`, `50247b26`, `a349fea8`) ne lisent pas leur
vue A ; leurs sorties ne changent pas.

### 1.4 Les charges portées (54 lecteurs)

Primitives (vérifiées, compteur `+0x2c` du flux) : `FUN_1406cf008` R(1) ; `FUN_14080d69c`
[R(1) ; si 1 : R(32)] ; `FUN_1407f2058` [R(1) ; si 0 : R(5)] ; `FUN_1406d00ec` [R(1) ; si 0 : R(2)] ;
`FUN_14076dc04` R(R9D) ; `FUN_1406d84b4` R(cinquième argument) ; `FUN_14080dec4` R(32) ;
`FUN_14076d528` [R(1) ; si 0 : R(7e arg) + R(6e arg)] ; `FUN_1407cbc24` octets R(8) jusqu'au nul,
`n` au plus (au-delà, erreur du flux : la lecture est refusée). Les immédiats viennent du
désassemblage (`search_instructions` par fonction ; `disassemble_function` épuise le tas Java du
serveur sur les grandes fonctions).

| Genre | Nom | Lecteur | Grammaire portée (largeurs lues) |
|---|---|---|---|
| 0 | damage_aftermath | `FUN_1407f15a4` | d69c ; **f2058 (si 0 : R(5))** ; R(19) ; R(1)->[R(19)+R(12)] ; R(5) R(5) R(6) ; R(1)->R(5) ; 15 R(1), le dernier garde R(32) ; R(1) R(3) R(5) R(5) R(1) R(4) ; [R(1) ; si 0 : R(10)] ; R(4)=f ; R(1)->R(32) ; si f=1 : R(8) ; R(bitLen(10)) ; R(1)->réf. domaine 0 |
| 1 | damage_section_response | `FUN_140968368` | R(5) ; [R(1) ; si 0 : R(4)] ; R(3) ; R(1)->R(19) |
| 2 | restore_damage_section | `FUN_142ef90a4` | R(1) ; si 1 : R(bitLen(32)) sinon R(bitLen(3)) |
| 5 | projectile_detonate | `FUN_1408096f8` | R(6) ; [R(1) ; si 0 : d69c + R(32)] ; d69c ; **position niveau 0xf** ; **R(0x13)** (R14D, @140809796) ; R(5) R(1) R(9) ; R(1)->R(10)+R(8) ; R(1) g ; **R(g ? 0x13 : 8)** (@14080986a) ; R(2) |
| 6 | projectile_impact_effect | `FUN_1410f03b4` | [R(1) ; si 0 : d69c + R(32)] ; R(7) R(7) R(19) ; position niveau 0xc ; R(19) R(9) R(1) |
| 7 | projectile_object_impact_effect | thunk `0x142f17474` -> `FUN_142f1c6cc` | [R(1) ; si 0 : d69c + R(32)] ; R(7) R(7) R(19) R(2) ; 3 x R(12) (`FUN_140c1e9d4`) ; R(19) R(9) R(16) R(1) R(1) |
| 8, 53 | biped_board_vehicle, unit_enter_vehicle | `FUN_142f168c0` | R(6) |
| 9 | biped_pickup | `FUN_141037828` | R(3) ; d69c |
| 11 | weapon_empty_click | `FUN_142f17ffc` | d00ec ; d69c + R(32) ; R(1) ; f2058 |
| 16 | ShowDebugText | `FUN_142eebf1c` | R(32) ; R(0x280) ; chaîne de 0x60 octets au plus |
| 21 | unit_zoom | `FUN_141168b28` | R(2) |
| 22 | unit_exit_vehicle | `FUN_142f17b94` | R(6) R(1) R(3) |
| 31 | equipment_teleport_request | `FUN_142ef8ec8` | R(4) ; f2058 |
| 36 | action_weapon_fire | `FUN_14080c1f8` | §1.5 |
| 37 | weapon_overheat | `FUN_142ef94f4` | d69c + R(32) ; d00ec |
| 38 | weapon_reload | `FUN_1407f0ff8` | 4 x R(1) ; f2058 |
| 40 | biped_melee_initiate | `FUN_140ff8d70` | version native 2 : R(3)=k ; si k=1 : R(2) ; R(2) ; d69c + R(32) ; f2058 |
| 41 | vehicle_trick | `FUN_142f17c84` | R(3) ; f2058 |
| 42 | biped_dodge | `FUN_142f169d0` | R(32) R(19) R(8) ; f2058 |
| 44 | weapon_pickup | `FUN_142f18158` | R(3) ; d00ec ; d00ec ; R(3) |
| 45 | weapon_put_away | `FUN_142f18284` | R(1) ; d00ec ; d69c + R(32) |
| 46 | weapon_drop | `FUN_142f17d74` | R(1) ; d00ec ; d69c + R(32) ; R(1) |
| 47 | weapon_throw | `FUN_142f18490` | R(1) ; d00ec ; d69c + R(32) ; f2058 |
| 48 | weapon_tether_request | `FUN_142f183f0` | d00ec ; d69c + R(32) ; version native 2 : R(1) |
| 58, 105, 118 | supercombine, ObjectKnockedBack, repair_complete | `FUN_142f17480`, `FUN_141118a00`, `FUN_142ef9074` | d69c |
| 63 | biped_laser_designation | `FUN_142f16c90` | R(1) |
| 75 | AIDialog | `FUN_140f2e634` | R(5) ; d69c ; [R(1) ; si 0 : R(12)] ; f2058 ; n = R(1)+1 ; n x R(1) |
| 76 | Dialogue2D | `FUN_140f2e87c` | d69c ; d69c ; f2058 ; 32 x R(1) |
| 80 | networked_ai_effect | `FUN_142ef8f74` | d69c ; R(2) mode ; 0 : une position 0x10 ; 1 : trois (`FUN_142eefa5c`) |
| 81, 83 | PlayerGameEvent, TeamGameEvent | `FUN_142f164f4`, `FUN_142f1686c` | `FUN_142efb480` : R(32) R(8), sac à compte R(4) (`FUN_142efb634`), sous-sac ; puis 32 x R(1) (81) ou R(9) (83) |
| 82, 84 | PlayerGameEventSmall, TeamGameEventSmall | `FUN_14080add8`, `FUN_142f16818` | `FUN_14080ae70` : R(32) R(8), sac à compte R(3) (`FUN_14080b1b8`), sous-sac `FUN_14080b034` (`consumeSacTexte`) ; puis 32 x R(1) (82) ou R(9) (84) |
| 86 | EngineClientEvent | `FUN_142f1615c` | R(32) |
| 88 | RevertMap | `FUN_142ef8b5c` | R(1) |
| 98 | Equipment | `FUN_142eebd68` | R(8) R(1) |
| 100 | PowerUpApplied | `FUN_142ef8a64` | d69c + R(32) ; d69c |
| 104 | EquipmentKnockbackPlayer | `FUN_14116c344` | d528(10, 0x13) |
| 108 | NavpointRequest | `FUN_142ef8a40` -> `FUN_142ef4a98` | R(32) R(32) |
| 109 | PersonalAILifceycleEffect | `FUN_142c61310` | R(2) ; d69c ; d69c ; R(32) |
| 119 | EquipmentKnockbackRequest | `FUN_142eebcec` | R(19) ; n = R(4)+1 ; n x {réf. domaine 0 ; R(19)} |
| 120 | PlayerCalloutRequest | `FUN_142f163e0` | R(5) ; position 0x10 ; R(4) réf. domaine 1 ; R(4) réf. domaine 0 |
| 13 genres vides | — | `FUN_1408d8220` | aucune charge |

Valeur de propriété d'un sac (`FUN_14080eff0`) : étiquette 0 rien ; 1, 2, 3, 6 R(32) ; 4 R(1) ; 5
chaîne de 16 octets au plus ; 7 position de niveau 0x10 (`FUN_140f04f18`).

Positions (`FUN_14076e524`, portage unique `lecteur_position.go`) : au niveau 0x10, le portage du
dépôt (index de plage lu sur `DAT_144632be0`, largeurs de la carte) ; aux niveaux 0xf et 0xc
(genres 5, 6), un index de plage n'est PAS lisible (`tablesSansIndexDePlage`) : les largeurs se
calculent sur les bornes de la plage indexée (`DAT_1445ccbe0 + (index * 0x20 + NIVEAU) * 0xc`), que
le film ne porte pas ; seule la porte posée (bornes du build) se lit. Mesuré : la position du genre 5
ouvre un index dans 100 % des listes examinées (sonde `lnDiagGenre5`) ; les genres 5 et 6 arrêtent
donc presque toujours la lecture.

### 1.5 Le tir (`FUN_14080c1f8`, genre 36)

Tête (comme `lireEnteteTir36`) : court R(1), bloc R(1), `FUN_141fcf670` R(7)+R(1), f2058, d00ec, d69c,
R(32), R(1), R(1) ; si bloc : R(1), horodatage = R(1) ; si horodatage : `FUN_1431a0abc` [R(1)->R(10)].
Court : R(0xa) (@14080c465) puis fin. Sinon : comptes `FUN_14080cc68` (premier = composantes, second
= cibles ; z R(1), u R(1)->R(4)...) ; boucle des cibles (R(2) genre, R(1), réf. domaine 1) ; boucle
des composantes (R(4) ; p R(1) ; si p : R(bitLen(6)), index R(1) si cibles < 3 sinon R(4), R(16),
3 x R(w) avec w = 12 si une seule composante sinon 4, plafonné à 6 si la cible désignée est de genre
1 — `FUN_14102bd24`). Si la dernière composante présente n'a pas un R(3) nul : composite
`FUN_140c9e4d8(..., 0, &garde)` (`consume142f26740`), `FUN_1408eff64` (`consume1408eff64`), et la
visée R(0x1e) si la garde est fermée. Queue : si pas d'horodatage : [R(1)->R(6)], R(6) ; si bloc :
`FUN_1431a0cbc` (`lireVecteur1431a0cbc`) puis `FUN_141102ed0(0x24) > 1` -> R(4) (version native 3) ;
sinon `FUN_14080cb98` R(2), `FUN_14080cb50` [R(1) a2 ; R(1)->R(4)], si a2 : `FUN_14320c36c`,
`FUN_142a40f18` [R(1)->2 x R(6)] ; R(6) ; [R(1)->R(7)] ; R(1)->position 0x10.

Grammaire concordante avec le lot R7 (`r7_charges_lot6_research_test.go`), relue pièce par pièce.

### 1.6 Ce qui n'est PAS lu, et pourquoi

| Genre | Ce qui manque | Fonction |
|---|---|---|
| 15 Script | le préfixe R(15) dépend d'un état d'exécution : le lecteur le lit si `FUN_1404f25f4()` (`état + 4 == 2`), l'écrivain l'écrit si `FUN_1404f293c()` (`!= 2`) ; le film n'est enregistré que par un processus de type 4 (`FUN_1405f6254`, appelants `FUN_142d5f008`, `FUN_140514828`) dont la valeur de `+4` n'est pas lue | `FUN_14080bb4c`, `FUN_142eec4d8` |
| 39 biped_throw_initiate | toute la charge est gardée par `*(DAT_144c1cfa8 + 4) == 2` (structure d'options de partie, `FUN_1410ffa34`), symétriquement chez l'écrivain ; valeur d'exécution | `FUN_140c6a58c`, `FUN_14104fc8c` |
| 85 PlayerKilledEvent | la queue dépend de `DAT_1451789b8`, `DAT_145178a48`, `FUN_1406aed00` | `FUN_14104bd08` |
| 5, 6 (position à index) | bornes de la plage indexée hors du film (§1.4) | `FUN_14076e524` |
| autres (≈ 50, rares) | non portés dans ce lot (temps) ; la lecture s'y arrête | — |

R7 (2026-09-03) mesurait le préfixe du Script « présent » (facteur 2,48 sur l'oracle de trame) :
c'est une mesure, pas une lecture ; le Script reste hors de ce lot (D-LN-7).

## 2. Ce qui change

Fichiers neufs (tous sous `apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) :
`vue_a_genres.go` (la table), `vue_a_lecture.go` (`lireLaVueA`, `parcourirLaVueA`,
`lireUnMessage`), `vue_a_versions.go` (règle du §1.3), `vue_a_charges.go`, `vue_a_charges_armes.go`,
`vue_a_charges_sacs.go`, `vue_a_charges_tir.go` (les charges), `vue_a_lecture_test.go` (vecteurs),
`vue_a_charges_test.go` (vecteurs des charges aux immédiats du jeu, correction C3), sonde
`ln_r7_research_test.go` (tag `research`), `vue_a_genres_test.go` (la table relevée, 123 lignes
nom / descripteur / lecteur / domaines des trois références, en littéraux). Les tables lues dans l'exécutable (genres, versions natives)
sont des CONSTANTES chaîne lues par fonction (`descripteurDuGenre`, `versionNative`) et les charges
sont rendues par `switch` (`chargeDuGenre`, `chargeDArme`, `chargeDEvenementDeJeu`) : aucune
variable de paquet neuve (ratchet `filmdec_package_vars`).

Crochets :
- `debut_de_liste.go` (`localiserLaListe`) : quand `marchLocateStrict` rend -1, `lireLaVueA` donne le
  début ([`lecture.DebutParVueA`]) ; la recherche par fermeture (`debutParFermetureRangee`) ne sert
  plus qu'à défaut. Les paquets localisés par la signature ne changent pas.
- `profil_balayage.go` : `GrammaireBalayage.GenresVueA`, posé par `grammaireSousFilm` (la règle des
  champs que le film déclare, à côté de `ControleDeCorruption`).
- `controle_corruption_du_film.go`, `film_context.go`, `keyframe_world_preuve.go` : la grammaire que
  le film déclare (contrôle de corruption et genres) est dérivée une fois et posée à chaque rendu du
  profil (`grammaireDeclareeParLeFilm`) — même discipline que le contrôle de corruption (un profil
  posé par `PoserProfilDeBalayage` ne l'efface pas). Anchors du registre des replis inchangées.
- `lecteur_position_ratchet_test.go` : six sites de position neufs dans la table, avec leur CALL.

Fichiers de la représentation intermédiaire touchés (au plus petit) :
- `grammar/lecture/paquet.go` : la valeur `DebutParVueA` ajoutée EN QUEUE de `DebutDeVueB`.
- `movement_states.go` : une ligne — `EventPacketsNewRecordStart` ne compte pas un début par la vue A
  (il compte les listes ouvertes par un record NEW de tête).

Aucun autre fichier de la liste de la RI (`event_list.go`, `fire_events.go`, `marche_trames*.go`,
`lecteur.go`...) n'est modifié ; les canaux de la tête de vue A ne sont pas touchés.

## 3. Révisions

- `grammar.Rev` : `grammar-2026-10-03.2` -> **`grammar-2026-10-04`** (entrée de `rev_chronique.go`,
  empreinte régénérée). Le format du dépôt impose le rang 1 d'un jour neuf (`Rang.Suit`) : la valeur
  n'est portée par aucune autre branche à l'heure du commit ; LS (en cours, même base) pourrait la
  prendre aussi — l'intégrateur renumérote (D-LN-13). Corrections du contrôle : la lecture du bit de
  configuration (C2) change les sources de la couche ; la révision du lot, jamais intégrée, reste
  `grammar-2026-10-04`, empreinte régénérée (`0392368d…`), carte v2 des 20 films identique (§11).
- `killsource.Rev` **inchangée** (`killsource-2026-09-27`), empreinte seule recopiée (D23) :
  `killsource` n'appelle pas `localiserLaListe` ; `cmd/killsource json` identique à l'octet sur les
  19 témoins. Ligne de commentaire du golden et complément de sa chronique.
- `source.Rev`, `objectives.Rev` : inchangées (leur fermeture ne rencontre pas `grammar`) ;
  `replay.SchemaVersion` inchangé (77) ; golden des formes (`types/testdata/shapes.golden`) et
  fixtures de contrat web (`replay_schema_77_*`) régénérés : ils ne diffèrent que par la chaîne de
  révision (taille −62 à −64 octets, premier écart = `grammarRev`, le reste des lignes identique).

## 4. Mesures

### 4.1 Le témoin (sonde `ln_r7_research_test.go`)

Sur chaque paquet à événements, avant sa marche (monde d'avant le paquet), la vue A lue donne E ;
la marche de BASE donne S (`lnDebutSansVueA`, le localisateur d'avant le lot). Accord : E = S ;
« accord par chaîne » : E < S et les records de E à S, lus comme la boucle de records les lit,
tombent au bit près sur S (le localisateur sautait des records de tête).

| Lecteur | Accord | Par chaîne | Désaccord E < S | E > S | Non localisés lus jusqu'au terminateur (fermés depuis E) |
|---|---|---|---|---|---|
| R7 (`r7Marche`, 2026-09-03) | 32 329 | 11 022 | 22 296 | 3 291 | 26 582 (4 676) |
| LN (ce lot) | 45 007 | 12 441 | 13 655 | 414 | 27 276 (10 800) |

Listes à UN message, HI_1_13_0 (`ln_tsv/temoin_messages_seuls_par_build_et_genre.tsv`) : genre 0
95,8 % (R7, grammaire de production : 36,4 %), 21 98,0 %, 36 96,2 %, 38 97,9 %, 82 98,0 %. Les
désaccords E < S de grande amplitude (centaines de bits) sont ceux du zoom (grammaire de deux bits)
comme des autres : le localisateur de base démarre plus loin que la vue B. Sur HI_1_10_0, même le
zoom ne s'accorde qu'à 75 % (D-LN-10).

### 4.2 Carte de fermeture v2, 20 films (`ln_tsv/gate2_par_film.tsv`)

Binaire `cmd_fermeture` de `2393d7db7` contre celui du lot, même table ECS, `-mode v2
-denominateur-fixe`, `-paquets`.

| Film | Build | Sains avant | après | net | perdus bruts | Utiles sains avant | après | net | Gains au bit (dont contredits) |
|---|---|---|---|---|---|---|---|---|---|
| bcb6d393 | HI_1_12_0 | 5 879 | 5 924 | +45 | 1 | 35 492 | 35 828 | +336 | 44 (0) |
| fb1a1a72 | HI_1_13_0 | 43 450 | 44 159 | +709 | 0 | 325 749 | 331 462 | +5 713 | 679 (0) |
| d9781168 | HI_1_13_0 | 26 317 | 31 540 | +5 223 | 7 | 180 854 | 226 685 | +45 831 | 5 066 (2) |
| c75f33b8 | HI_1_13_0 | 23 872 | 26 679 | +2 807 | 0 | 153 997 | 176 437 | +22 440 | 2 725 (0) |
| bf15f7ab | HI_1_13_0 | 28 603 | 28 852 | +249 | 0 | 216 104 | 218 295 | +2 191 | 241 (0) |
| 51ebbc0f | HI_1_13_0 | 20 443 | 23 542 | +3 099 | 2 | 140 617 | 167 919 | +27 302 | 3 035 (3) |
| 084a804d | HI_1_10_0 | 4 832 | 5 816 | +984 | 38 | 89 616 | 117 210 | +27 594 | 992 (4) |
| 0797ce72 | HI_1_13_0 | 19 156 | 19 385 | +229 | 0 | 159 929 | 162 161 | +2 232 | 219 (0) |
| 111fa685 | HI_1_10_0 | 4 026 | 5 139 | +1 113 | 26 | 42 896 | 70 083 | +27 187 | 1 133 (3) |
| e5adf7b2 | HI_1_11_0 | 4 147 | 5 497 | +1 350 | 22 | 80 067 | 116 119 | +36 052 | 1 354 (3) |
| 60ae07c4 | HI_1_8_0 | 13 946 | 13 946 | 0 | 0 | 83 669 | 83 669 | 0 | 0 |
| a349fea8 | version-33 | 424 | 424 | 0 | 0 | 3 654 | 3 654 | 0 | 0 |
| a521164d | HI_1_4_1 | 692 | 692 | 0 | 0 | 78 | 78 | 0 | 0 |
| 11de8353 | HI_1_9_0 | 5 629 | 7 242 | +1 613 | 11 | 65 621 | 101 868 | +36 247 | 1 619 (12) |
| 50247b26 | version-31 | 139 | 139 | 0 | 0 | 274 | 274 | 0 | 0 |
| bfecd02b | HI_1_13_0 | 27 666 | 28 426 | +760 | 1 | 239 045 | 246 745 | +7 700 | 743 (0) |
| 4f77afc1 | HI_1_13_0 | 24 082 | 28 582 | +4 500 | 57 | 607 244 | 745 749 | +138 505 | 4 006 (19) |
| 396cfc92 | HI_1_13_0 | 23 131 | 23 580 | +449 | 1 | 169 502 | 173 523 | +4 021 | 445 (0) |
| f75e7053 | HI_1_13_0 | 23 673 | 24 008 | +335 | 1 | 162 137 | 164 668 | +2 531 | 331 (0) |
| 1c4c63c2 | HI_1_10_0 | 13 388 | 17 533 | +4 145 | 1 422 | 165 965 | 304 257 | +138 292 | 4 902 (77) |
| **corpus** | | **313 495** | **341 105** | **+27 610** | **1 589** | **2 922 510** | **3 446 684** | **+524 174** | **27 534 (123) = 0,4 %** |

Gate 2 : **aucun film en baisse**, ni en paquets sains ni en records utiles sains. Les quatre films
dont la table des genres n'est pas un préfixe de celle du jeu (dont HI_1_8_0) sont identiques.

- Listes non localisées : 47 854 -> 20 101 (`ln_tsv/fermeture_films_*.tsv`, colonne 6).
- Naissances : rejets d'en-tête dont l'eid a une naissance non lue (tableau « Rejets » de la carte) :
  corpus 188 130 -> 180 900 ; HI_1_13_0 44 814 -> 37 823.
- Records utiles lus : 6 296 003 -> 6 858 065.
- Indicateur D1 (records utiles sains sur le dénominateur fixe consolidé de la vague 1,
  `integ2/denominateurs.tsv`, 7 758 290) : corpus 37,7 % -> **44,4 %** ; HI_1_13_0 76,6 % ->
  **85,0 %** ; variable 46,4 % -> 50,3 % (HI_1_13_0 83,8 % -> 87,2 %). Le lot lit plus que le fixe sur
  `c75f33b8` (180 143 > 179 123) et `4f77afc1` (834 448 > 825 839) : fixe recalculé 7 767 919 (corpus
  44,4 %), HI_1_13_0 3 082 896 (84,8 %).

### 4.3 Compteur `repli_debut_de_liste_ferme_au_bit` (journal `replay-equiv`, `ln_tsv/replis_changes.tsv`)

En baisse sur les 16 films qui l'émettaient : total 7 880 -> 2 038 (`1c4c63c2` 6 428 -> 1 593,
`084a804d` 472 -> 179, `d9781168` 261 -> 55, `111fa685` 141 -> 57, `11de8353` 128 -> 56, `e5adf7b2`
159 -> 46, `9f57c612` 112 -> 20, `64e8adfa` 60 -> 9, `fb1a1a72` 39 -> 8...). La lecture de la vue A
remplace le second rang de la recherche par fermeture. `repli_liaison_par_anticipation` bouge sur 14
films (baisse sur 10, hausse sur `084a804d` +36, `111fa685` +37, `e5adf7b2` +67, `11de8353` +5) :
moins de naissances ratées, plus de paquets marchés.

## 5. Pertes : non expliquées, compensées au net (réécrit, correction C4)

**Statut : 1 589 paquets sains en référence perdus bruts, NON expliqués.** Ils ne sont PAS établis
comme des fermetures factices ; l'exception D2 n'est pas invoquée. Le gate 2 est NET par film et il
tient : chaque perte est compensée dans son film (§4.2).

Faits (mesurés) :

- Tous « devenus non fermés », aucun « devenu contredit » (`ln_tsv/gate2_sains_perdus.txt`). Par
  film : `1c4c63c2` 1 422, `4f77afc1` 57, `084a804d` 38, `111fa685` 26, `e5adf7b2` 22, `11de8353`
  11, `d9781168` 7, `51ebbc0f` 2, quatre films à 1. HI_1_10_0 en porte 1 486 (`1c4c63c2`,
  `084a804d`, `111fa685`). Causes après sur `1c4c63c2` : sortie de vue B par rejet hors datum
  1 127, terminateur de vue C hors cadre 183, `ti=12 i16` 50, vue C kind non porté 15, autres 47.
- Paquet par paquet (sonde `ln_r7_research_test.go`, ligne par paquet dont le début change,
  `ln_tsv/sains_perdus_temoin.tsv`) : 1 580 des 1 589 sont des paquets localisés AVANT le lot par la
  recherche par fermeture en D (= S, le localisateur strict ne les trouvait pas), où la vue A lue
  se termine en E ≠ D : 1 579 « E < S, depuis E non fermé, depuis S fermé », 1 « E > S ». Les 9
  autres (`e5adf7b2` 10:1010 à 10:1024, six paquets ; `111fa685` 13:1160 ; `1c4c63c2` 32:2320 et
  40:1404) ne ressortent pas de la sonde avec un début changé : non instruits.
- L'écart S − E dépasse 512 bits dans 1 572 cas sur 1 580. Listes à un message : 1 043 ; dernier
  message lu de la vue A : genre 36 (tir) 695, 82 270, 0 246, 21 75, 38 64, 9 63. Sur HI_1_10_0, le
  dernier message est un tir dans 669 cas.
- Le crochet change le début de 8 855 paquets localisés par fermeture sur 10 294 (désaccord avec E :
  2 220 « fermeture », 6 634 « fermeture au bit » ; 1 accord par chaîne) ; 6 505 paquets fermés au bit avant le lot
  ne le sont plus (5 590 sur `1c4c63c2`), contre 27 534 gains fermés au bit.

Pourquoi ce n'est pas une preuve de fermeture factice. L'argument de la première version (« la vue
B commence après le terminateur de la vue A ; une fermeture partie de D > E saute les records E..D,
donc l'écrivain la contredit ») suppose E juste. Or :

- le témoin s'écarte de E en 14 069 cas sur les 71 517 paquets localisés (13 655 E < S sans chaîne,
  414 E > S ; toutes méthodes de localisation, §9) ;
- sur HI_1_10_0, où tombent 1 486 pertes, l'accord des listes à UN message est de 44,1 % pour le
  genre 36, 61,2 % pour le genre 0 et 75,1 % pour le genre 21, contre 96,2 / 95,8 / 98,0 % sur
  HI_1_13_0 (`ln_tsv/temoin_messages_seuls_par_build_et_genre.tsv`, lecteur `ln5`) ; le tir, qui
  clôt 669 des listes perdues de ce build, est le genre où la grammaire y est la plus démentie ;
- ces paquets fermaient au bit depuis D sous le juge de L0 (sains en référence).

Un désaccord entre E et S ne dit pas lequel des deux est faux : sur HI_1_10_0, le localisateur de
base démarre lui aussi souvent après la vue B (D-LN-10). Le trancher paquet par paquet demande de
situer le message où la chaîne E..D casse (genre de la vue A, largeur lue contre largeur écrite),
ce qui suppose l'écrivain de HI_1_10_0 (§9) ; non fait.

## 6. Killsource, `replay-equiv`, gate de corpus

- **Killsource** (`cmd/killsource json`, 19 témoins de `config/replay_corpus.toml`, binaire de
  `2393d7db7` contre binaire du lot) : **IDENTIQUE À L'OCTET sur les 19**. Aucune mort, valeur ni
  voie (`read_path`) ne change ; `killsource.Rev` reste.
- **`replay-equiv`** (recette de L0 : racine factice `scratchpad/v2-LN/repo`, films et
  `data/titles` copiés du parc (identiques à l'octet), `config/` et références d'équivalence de
  `2393d7db7`) : base 20 / 20 « différents » des références du dépôt (non re-figées après la vague 1,
  attendu) ; base contre lot, digests par étape (`ln_tsv/re_etapes_divergentes.tsv`) : **7 étapes sur
  61** — `artifact`, `vehicles`, `killsource` sur les 20 ; `movementStates` et `.stats` sur 16,
  `continuousFire.stats` 16, `continuousFire` 13. Les quatre films sans vue A lisible (`a521164d`,
  `a349fea8`, `60ae07c4`, `50247b26`) ne changent que sur `artifact`, `vehicles`, `killsource` :
  `digest.Of` hache les valeurs par réflexion, et le profil porté par `Vehicles.DeathStats.Config` et
  `Kills.ProfilCalibre` a un champ de plus (`GrammaireBalayage.GenresVueA`) ; une variante où le
  champ n'est pas sérialisé en JSON ne change rien à ces deux digests (contrôle sur trois films) —
  c'est la FORME. `movementStates` grandit (ex. `d9781168` 2 721 -> 3 039, `fb1a1a72` 2 624 ->
  2 665) : les lectures des paquets nouvellement localisés. Positions, morts, identités, équipes,
  objectifs, équipement : identiques.
- **`replay-corpus-gate`** : §8 bis.

## 7. Gates

Gates rejoués après les corrections du contrôle : §11.

Depuis `apps/go-api`, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg2-ln`, une commande `go`
à la fois, sur le code final (tables en constantes) :

- `gofmt -l` (film, archlint) : vide.
- `go vet ./...` : rc 0 ; `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc 0.
- `go test ./internal/archlint/... -count=1` : `ok levelup/go-api/internal/archlint 34.082s`.
- G-film (`go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/...
  ./internal/sync/killcollector/... -count=1 -timeout 30m`) : rc 0, 20 paquets `ok`, aucun échec.
- `golangci-lint run --new-from-rev=2393d7db7 ./internal/games/halo_infinite/film/...` (cache isolé
  au scratchpad) : `0 issues.`
- Vecteurs du lot (`vue_a_lecture_test.go`, 9 tests ; `vue_a_genres_test.go`, 1 test) : verts.
- Carte v2 et killsource rejoués sur le binaire du code final : carte identique à celle du §4.2
  (paquets et films), killsource identique à l'octet sur les 19 témoins.
- **Mutations** (`ln_tsv/mutations.sh`, `-overlay`, une par règle neuve, `ln_tsv/mutations.txt`) :
  **12 / 12 ROUGES** — genre vide, version native d'un genre, terminateur, contrôle de corruption,
  comparaison des versions, polarité des dégâts, largeur du tir court, crochet de
  `localiserLaListe`, cardinal des genres, polarité du rechargement, index de plage hors niveau
  d'objet, genre sans charge. Deux mutations ont d'abord SURVÉCU : le tir court (le test écrivait la
  largeur par la constante mutée ; test corrigé, immédiat `0xa` du jeu) et la version native (le test
  construisait la table native par `versionNative` ; versions de `DAT_14474cd90` épinglées dans le
  test). Puis rouges.

## 8. Sorties exactes des gates finaux

```
gofmt: []
vet rc=0
vet research rc=0
ok  	levelup/go-api/internal/archlint	34.082s
G-film rc=0   (20 lignes ok)
golangci-lint : lint rc=0 / 0 issues.
m1_genre_vide : ROUGE [TestLaTableDesGenresEstCelleDuJeu]
m1b_version_native : ROUGE [TestLaTableDuFilmEstUnPrefixeDeLaTableNative]
m2_terminateur : ROUGE [TestLaVueALueRendLeDebutDeLaVueB]
m3_corruption : ROUGE [TestLeControleDeCorruptionSuitLaCharge]
m4_versions : ROUGE [TestLaTableDuFilmEstUnPrefixeDeLaTableNative]
m5_polarite_degats : ROUGE [TestLesDegatsLisentLaPorteInverseeDuJeu]
m6_tir_court : ROUGE [TestLeTirCourtSArreteApresSaDirection]
m7_crochet : ROUGE [TestLaVueADonneLeDebutQueLaSignatureNeTrouvePas]
m8_cardinal : ROUGE [TestLaVueANeDevineRien]
m9_polarite_rechargement : ROUGE [TestLaVueALueRendLeDebutDeLaVueB]
m10_index_de_plage : ROUGE [TestUnIndexDePlageNeSeLitQuAuNiveauDObjet]
m11_genre_non_porte : ROUGE [TestLaVueANeDevineRien]
killsource json, 19 témoins, base contre final : 19 IDENT
```

`replay-corpus-gate` (`--reference=base --base=2393d7db7`, `--parc-root` sur une copie du parc au
scratchpad) : il lit le code au HEAD de `--source-root`, donc après le commit du lot ; sa sortie est
consignée au §8 bis par un commit de note.

## 8 bis. `replay-corpus-gate` (sur le commit du lot `4b4ed264b`)

Commande (`ln_tsv/gate_corpus.sh`) : `gate.exe --reference=base --base=2393d7db7 --parc-root <copie du
parc au scratchpad> --source-root <worktree> --work-root <scratchpad> --json <scratchpad>`. Base cuite
pour les 19 témoins (absente du cache de la copie). Résumé : `ln_tsv/gate_corpus_resume.txt`.

- **BANC DE VÉRITÉ (le verdict du gate) : 19 / 19 `ok`**. Seuls constats non informatifs : `P-1
  paquets fermés au bit près` en GAIN sur les 15 témoins à vue A lisible (somme +23 465 ; ex. `d9781168`
  26 317 -> 31 540 ; `1c4c63c2` n'est pas un témoin du manifeste) ; les compteurs de repli en `info`. Les quatre témoins sans vue A
  lisible (`60ae07c4`, `a349fea8`, `a521164d`, `50247b26`) : `ok`, 0 gain, 0 perte, 0 changement.
- **Code de sortie 1** : le diff de référence (axes FILET, « aucune mesure du banc ne couvre ce bloc »)
  porte 623 pertes et 6 disparitions sur 15 témoins ; 896 gains, 65 changements. Pertes par métrique :
  `stances/duree-totale` 15 témoins (somme 381 752 -> 356 922, −6,5 % ; de −1,1 % `0797ce72` à −31 %
  `c75f33b8` 11 986 -> 8 253) et 495 lignes par slot ; `coverage.continuousFire.holesOpenViewB` 15
  témoins (somme 15 300 -> 16 831) ; trous de rafale (`holesKind`, `holesBlockBC`, `innerHoles`,
  `holeRuns`, `burstsWithHole`) sur les films HI_1_9_0 à HI_1_11_0 ; `stances.forgottenBindings`,
  `dropped`, `refusedNews` sur 4 à 5 témoins ; une rafale publiée de moins sur `111fa685` (6 -> 5) et
  les 6 disparitions (trous de rafale de `111fa685`, un slot de `084a804d`).
- Lecture : ces pertes portent sur des blocs que le banc ne mesure pas ; elles suivent les paquets
  nouvellement localisés qui sont LUS sans fermer (témoin §4.1 : 27 276 listes lues, 10 800 fermées) :
  un paquet lu et non fermé ouvre un trou de vue B dans le tir continu (`holesOpenViewB` +) et coupe
  les postures qui le traversent. Non instruit paquet par paquet (D-LN-15). Précédent : le gate de la
  vague 1 montrait les mêmes axes FILET en perte (`stances/duree-totale` `fb1a1a72` 19 031 -> 16 482).

## 9. Écarts au critère

- Aucune condition par film, par carte ni par build : la règle du §1.3 compare la table du film à
  celle de l'exécutable ; les genres exclus le sont par leur lecteur (§1.6), sur tous les films.
- La règle « table du film = préfixe de la table native » exclut des films ENTIERS (4 sur 20) dès
  qu'un seul genre diffère, même absent de leurs listes : choix de prudence (une version différente
  dit que l'écrivain a changé ; la numérotation de HI_1_4_1 est décalée). Une règle par genre
  (lire un message si SA version est native) est possible et non mesurée.
- **La règle « table du film = préfixe de la table native » est NÉCESSAIRE mais NON SUFFISANTE**
  pour que la grammaire de l'exécutable soit celle du film (correction C5). Le témoin la réfute sur
  HI_1_10_0, dont les films déclarent pourtant leurs genres aux versions natives : l'accord des
  listes à un seul message y est de 44,1 % (genre 36), 61,2 % (genre 0) et 75,1 % (genre 21),
  contre 96,2 / 95,8 / 98,0 % sur HI_1_13_0 (`ln_tsv/temoin_messages_seuls_par_build_et_genre.tsv`,
  lecteur `ln5`). Une version déclarée ne dit donc pas tout ce que l'écrivain de ce build a écrit.
  Cela reste à instruire DANS LE JEU (l'exécutable de HI_1_10_0 n'est pas lu) avant de lire la vue A
  des films de ce build ; le lot la lit aujourd'hui sur eux, et c'est là que tombent 1 486 des
  1 589 pertes (§5).
- Le crochet est limité aux paquets que la signature ne localise pas (le plus petit crochet ; le
  périmètre de L1b). Sur les paquets localisés par la signature, la vue A s'écarte du début dans
  5 215 cas sur 61 223 (4 911 E < S sans chaîne, 304 E > S) : la grammaire n'y prime pas encore
  (D-LN-8). Sur l'ensemble des 71 517 paquets localisés (signature, fermeture, fermeture au bit),
  14 069 désaccords (13 655 E < S, 414 E > S) ; la première version de cette note attribuait les
  13 655 à la seule signature (`ln_tsv/temoin_lecteur_ln.tsv`, clé `temoin_par_debut`).
- Positions de niveau 0x10 : le lot suit la convention du portage unique (index de plage lu aux
  largeurs de la carte quel que soit l'index), comme toute la marche.

## 10. Découvertes (consignées, non traitées)

- **D-LN-1** Le lot R7 (2026-09-03, `RAPPORT_R7_TRAME_COMPLETE_2026-09-03.md`, sondes `r7_*`)
  marchait déjà la liste complète (96,5 % des listes de 12 films) ; R_LOC et R_NAIS (2026-10-02) ne
  l'ont pas repris. `R_LOC.md` §4.1 (« la table est construite à l'exécution ») est réfuté par
  `FUN_140e453b4`.
- **D-LN-2 (lourde)** La grammaire de PRODUCTION `lot1DecodeDamageAftermath`
  (`weapon_hits_decode.go`) lit la porte `FUN_1407f2058` à POLARITÉ INVERSÉE (« R(1) ; si 1 : R(5) »
  au lieu de « si 0 »). Le reste de la transcription est juste. Mesuré : listes à un seul
  `damage_aftermath` d'accord avec le début localisé 36,4 % sous la grammaire de production contre
  95,8 % sous la lecture du jeu (HI_1_13_0). La décodeuse sert `ScanFilmWeaponDamages` / le
  `killcollector` (touches par arme, ~872 k événements selon R7). Non corrigée (hors périmètre) :
  un lot à part, killsource et backfill à instruire.
- **D-LN-3** `event_list.go` lit le domaine 3 sur 7 bits (« mesuré ») pour `biped_board_vehicle` ;
  la table du jeu (`FUN_140d10bb0`, plage 0x100) dit 8. La vue A de ce lot lit 8 (lu).
- **D-LN-4** Le type 5 « non lisible au site d'appel » de R7 l'est : R(0x13) (R14D @140809796) et
  R(g ? 0x13 : 8) (@14080986a..140809877).
- **D-LN-5** La table par type de `chunk_00` est indexée par le genre de message (§1.3) ; le
  commentaire de `profile/identite.go` (« aucun des neuf index ne sépare... ») reste vrai, la
  sémantique est désormais établie.
- **D-LN-6** HI_1_8_0 déclare des versions anciennes des genres 35 et 36 ; HI_1_4_1 a une table de
  genres décalée : les décodeurs de production qui lisent l'événement de tête par son genre
  (`fire_events.go`, `zoom_events.go`, ...) le font sur ces films sans garde de version.
- **D-LN-7** Le Script (genre 15) bloque 4 372 listes non localisées et 11 150 localisées : établir le
  rôle (`état + 4`) du processus qui enregistre le film (type 4, `FUN_1405f6254`) lèverait le verrou
  par lecture. Idem 39 (`DAT_144c1cfa8 + 4`).
- **D-LN-8** Sur les paquets localisés par la signature, E < S sans chaîne dans 4 911 cas sur 61 223
  (3 814 où ni E ni S ne ferment, 1 085 où S seul ferme) ; la première version écrivait 13 655 (dont
  4 819 où S ne ferme qu'au bit), chiffres de TOUTES les méthodes de localisation (4 817 des 4 819
  sont des paquets localisés par « fermeture au bit »). Lecture candidate : le localisateur démarre
  au milieu de la vue B. La règle « la vue B commence au bit qui suit le terminateur de la vue A »
  est un invariant de l'écrivain candidat pour le juge (L0) ; non mesuré.
- **D-LN-9** Dans le contexte d'instrument de la carte, la position de niveau 0xf du genre 5 a les
  mêmes largeurs ([12 12 13]) sur trois cartes différentes : les largeurs des niveaux autres que 0x10
  y sont calculées sur une plage qui n'est pas celle de la carte du film. À instruire pour les autres
  sites de position à niveau différent de 0x10.
- **D-LN-10** HI_1_10_0 : même le zoom (grammaire de deux bits, références absentes) ne retombe sur
  le début localisé que dans 75 % des listes à un message (98 % sur HI_1_13_0) : le localisateur y
  démarre souvent après la vue B ; la vue B y ferme peu (17 %).
- **D-LN-11** 414 conflits durs E > S (122 sur HI_1_13_0) : la signature trouvée dans le corps d'un
  message de la vue A ? Non instruit.
- **D-LN-12** LS (en cours, même base) modifie `debut_de_liste.go` : conflit attendu avec ce crochet
  à l'intégration ; la lecture de la vue A se place après le localisateur unifié.
- **D-LN-13** La règle des rangs (`Rang.Suit` : un jour neuf commence au rang 1) ne laisse qu'une
  valeur possible par jour au premier lot qui monte : « une valeur qu'aucune autre branche ne porte »
  et « rangs sans trou » sont incompatibles entre lots parallèles d'un même jour.
- **D-LN-14** `disassemble_function` du serveur Ghidra échoue (« Java heap space ») sur les grandes
  fonctions ; `search_instructions` par mnémonique et par fonction rend le listage (`asm.sh`).
- **D-LN-15** `replay-corpus-gate` (§8 bis) : banc 19 / 19 `ok`, mais les axes FILET reculent sur les
  15 témoins à vue A lisible — durée totale des postures −6,5 % (jusqu'à −31 % sur `c75f33b8`), trous
  de vue B du tir continu +10 %, une rafale de moins sur `111fa685`. Mécanisme candidat : paquets
  nouvellement localisés lus sans fermer. À instruire avant intégration : distinguer une posture
  COUPÉE à tort d'une posture qui s'étendait à tort sur un trou de lecture (le banc ne tranche pas).

## 11. Corrections du contrôle indépendant (2026-10-04)

Contrôle sur `3031a2b25` : non conforme, corrigeable (six corrections). Toutes sont appliquées ;
aucune n'a été jugée fausse sur pièces. Un chiffre du contrôle est précisé : les 14 069 désaccords
portent sur les 71 517 paquets localisés par TOUTES les méthodes, pas par la seule signature (5 215
sur 61 223 pour la signature, §9) ; la même confusion figurait au §9 et en D-LN-8, corrigés.

| | Correction | Statut | Où |
|---|---|---|---|
| C1 | statut « [x] retenu » -> « [!] » tant que le gate 6 n'est pas tenu | [x] (le lot reste [!]) | §0 |
| C2 | bit de configuration LU (`FUN_142987460`), refus à 0, vecteur et mutation | [x] | `vue_a_lecture.go`, §1.2 |
| C3 | vecteurs aux immédiats du jeu, domaines 123 x 3 épinglés, charges couvertes, `mut.sh` tout ROUGE | [x] | `vue_a_charges_test.go`, `vue_a_genres_test.go`, `vue_a_lecture_test.go` |
| C4 | §5 réécrit : pertes brutes NON expliquées, compensées au net | [x] | §5 |
| C5 | règle des versions nécessaire, non suffisante (HI_1_10_0) | [x] | §9 |
| C6 | date retirée de l'en-tête de `vue_a_genres_test.go` ; `_ = source.BitAt`, `_ = p` retirés | [x] | sonde, test |

Détail :

- **C2.** `parcourirLaVueA` lit le bit (`if !br.ReadBit()`) et rend `arretSansTableDesCategories`
  à 0. Mesuré (sonde `TestLNR7`, `LN_LECTEUR=ln`, 20 films) : **0** paquet à événements dont la vue A
  est lue s'arrête sur un bit de configuration à 0 ; les tables du témoin restreintes aux clés de
  `ln_tsv/temoin_lecteur_ln.tsv` sont IDENTIQUES à celles de l'exécutant. Carte v2 des 20 films
  (`cmd_fermeture` du code corrigé, même table ECS, mêmes options) : `fermeture_paquets.tsv`
  IDENTIQUE à l'octet à celle du lot, tous les autres fichiers identiques hors pic mémoire et durée
  (`fermeture_films.tsv` colonnes 1 à 16 identiques). Gate 2 inchangé : 313 495 -> 341 105, aucun
  film en baisse. `killsource json` sur les 19 témoins : identique à l'octet au lot ET à la base.
- **C3.** Les écritures des tests portent les immédiats du jeu en littéraux (`R(7)` du genre,
  `R9D = 0x13`, quinze drapeaux, `FUN_1406d310c(10) = 4`, `R(7)+R(1)` du numéro de tir) ;
  `genresReleves` porte les domaines des trois références de chaque genre, en littéraux, contrôlés
  contre deux relevés indépendants (`ln_ghidra/table_genres.tsv` de l'exécutant, émulation du
  contrôle : 123 / 123 identiques). `vue_a_charges_test.go` couvre `FUN_142f1c6cc` (7),
  `FUN_142ef8f74` (80, modes 2 et 3), `FUN_142c61310` (109), `FUN_142f1686c` / `FUN_142f16818` (83,
  84), `FUN_14116c344` (104), `FUN_141037828` (9), et `FUN_1407cbc24` (chaîne sans nul).
  `FUN_140f58324` (R(9)), `FUN_141037828` (R(3)) et `FUN_142c61310` relus ce jour. Mutations du
  contrôle (`ln_tsv/mutations_controle.sh`, `-overlay`, suite entière du paquet après une base
  verte ; c6 réécrite pour le code corrigé : le bit lu puis ignoré) : **10 / 10 ROUGES**
  (`ln_tsv/mutations_controle.txt`). Mutations de l'exécutant rejouées : **12 / 12 ROUGES**
  (`ln_tsv/mutations_apres_controle.txt`).
- **C4.** Sonde : `lnR7.paquet` rend désormais une ligne par paquet dont le début change (la ligne
  était calculée et jamais rendue) ; jointure avec les pertes : `ln_tsv/sains_perdus_temoin.tsv`.
- **Révision.** Sources de la couche changées par C2 : `grammar-2026-10-04` (révision du lot, jamais
  intégrée) garde sa valeur, empreinte régénérée ; une phrase ajoutée à son entrée de chronique.
  `killsource.Rev`, `objectives.Rev`, `source.Rev`, `replay.SchemaVersion` inchangées (killsource
  identique à l'octet ; aucune sortie persistée ne change).
- **`replay-corpus-gate` non rejoué** : la carte et le témoin sont identiques au lot paquet par
  paquet ; ses pertes FILET (§8 bis) restent celles du lot, et le gate 6 reste non tenu (C1).

Sorties des gates rejoués (code corrigé, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg2-ln`,
une commande `go` à la fois) :

```
gofmt: []
vet rc=0
vet research rc=0
ok  	levelup/go-api/internal/archlint	31.055s
G-film rc=0 (20 ok, 0 FAIL)
lint rc=0 / 0 issues.            (--new-from-rev=2393d7db7, film/...)
lint research rc=0 / 0 issues.   (--build-tags=research, grammar)
carte v2 20 films : fermeture_paquets.tsv identique au lot ; gate 2 inchangé
killsource json, 19 témoins : 19 rc=0, 19 identiques au lot, 19 identiques à la base
mutations du contrôle : 10 / 10 ROUGES ; mutations de l'exécutant : 12 / 12 ROUGES
```
