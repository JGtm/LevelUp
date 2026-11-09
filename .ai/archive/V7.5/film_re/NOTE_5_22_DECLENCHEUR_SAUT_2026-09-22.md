# NOTE 5.22 — LE DECLENCHEUR DU SAUT, PAR LE TEMOIN Madina97294 (2026-09-22)

> Film `bfecd02b`, carte `snowbound`, build `HI_1_13_0`. Temoin : **Madina97294**, xuid
> 2533274858283686, **slot 523**, vie repliquee en continu de 80,032 s a 159,996 s d horloge
> film. Verdict Theater de l utilisateur du 2026-09-22 : sept sauts entre 1:19 et 1:41 de barre.
>
> Instruments : `internal/grammar/mouvement_5_22_{ancres,differentielle,controle,generalisation}_research_test.go`
> (`//go:build research`). Variables : `MOUV511_FILM`, `MOUV511_CARTE`, `MOUV511_BORNES`,
> `MOUV511_T0` / `T1`, `MOUV522_SLOT`, `MOUV522_BITS`, `MOUV522_ORIGINE_MS`.

## 0. Ce que ce lot cherchait, et ce qu il a trouve

Le lot 5.11 avait conclu « il n y a pas de declencheur du saut dans le film », sur un temoin
(`dad793c7`) dont le bipede etait MUET avant le saut : sa fenetre ne separait pas « le saut
commence » de « la replication commence » (D6 du lot 5.13, qui demandait explicitement un film ou
le bipede soit replique AVANT et APRES le saut). Ce lot a ce temoin.

**La reponse est la meme, et elle est desormais opposable** : sur une vie repliquee A CHAQUE TICK,
l ensemble des champs qui basculent au decollage et nulle part ailleurs est VIDE. Ce qui est neuf :

1. le denominateur — les **64** composants du bipede sont presents sur `bfecd02b`, `i0` a `i63`
   sans trou, et aucun n est exclusif aux fenetres de decollage ;
2. l ecrivain dit ou vit le saut, positivement : **action d unite `0x16`**, un bit d un mot de
   64 bits par unite dans une table THREAD-LOCALE — pas un champ d objet ;
3. **un canal reste hors de portee et il est nomme** : l entree (vue de controle) n est pas
   lisible dans la fenetre du temoin, parce qu aucun paquet n y ferme (trou du rang 1, lot 5.21).

## 1. La conversion de temps, tranchee avant toute fenetre

Le brief posait `barre = originMs + frame x 100 ms` avec `originMs = 12 547 ms`. **La mesure dit
autre chose** : la barre est l horloge RELATIVE AU PREMIER PAQUET DELTA, au pas de frame pres.

| evenement | temps de barre cite par l utilisateur | instant relatif mesure |
|---|---|---|
| saut 1 | 1:24,0 | 84,050 s |
| saut 2 | 1:25,1 | 85,234 s |
| saut 3 | 1:26,5 | 86,602 s |
| saut 4 | 1:29,2 | 89,322 s |
| saut 5 | 1:33,0 | 93,126 s |
| saut 6 | 1:36,9 | 96,962 s |
| sprints lus | 1:28,0 · 1:30,3 · 1:35,7 | 88,121 · 90,356 · 95,844 s |
| queue de `mobility` | 1:37,0 - 1:37,6 | 97,096 - 97,63 s |
| vie du slot 523 | 1:19 - 2:39,9 | 80,032 - 159,996 s |

`originMs` recale les evenements de MATCH ; il n est l origine d aucune barre. Consigne §4, D2.

## 2. Les sept sauts, et celui qui manque

`TestMouvement522Ancres` publie TOUS les episodes de montee fermes de la vie (86), retenus ou non
par la fenetre de hauteur du Spartan (`types.SpartanJumpHeightM` 0,85 m, tolerance 10 %) :

| verdict | t0 | duree | hauteur integree | Theater |
|---|---:|---:|---:|---|
| RETENU | 84,050 | 0,466 s | 0,8495 m | saut a 1:24 |
| RETENU | 85,234 | 0,450 s | 0,8413 m | saut a 1:25 |
| RETENU | 86,602 | 0,483 s | 0,8466 m | saut a 1:26,5 |
| RETENU | 89,322 | 0,468 s | 0,8430 m | saut « ~1:28 » |
| **REJETE** | **91,691** | **0,667 s** | **0,9737 m** | **le saut « ~1:31 » qui manque** |
| RETENU | 93,126 | 0,466 s | 0,8476 m | saut « ~1:32 » |
| RETENU | 96,962 | 0,334 s | 0,8104 m | le saut qui finit en escalade |

**Le septieme saut n est pas absent du film** : il est dans le canal de vitesse, et c est la
fenetre de hauteur qui l ecarte (+14,6 %). Sa duree — 0,667 s contre 0,467 s — dit une montee
PROLONGEE, pas un saut plat. Rien n est corrige ici : elargir la fenetre deplacerait le compte de
`jumpDerived` sur tous les films (§4, D1, arbitrage utilisateur).

## 3. La differentielle : au decollage, trois composants

`TestMouvement522Bascule`, slot 523, 4 557 records, fenetre `[t0 - 250 ms ; t0 + 100 ms]` (l entree
precede la physique) — 277 records dedans, 4 280 hors :

| composant | dedans | part | hors | part | facteur |
|---|---:|---:|---:|---:|---:|
| `i0` position | 277 | 100,00 % | 4 106 | 95,93 % | 1,04 |
| `i1` vitesse | 263 | 94,95 % | 3 937 | 91,99 % | 1,03 |
| `i25` command-tick | 277 | 100,00 % | 4 148 | 96,92 % | 1,03 |
| `i21` desired-aiming | 187 | 67,51 % | 2 801 | 65,44 % | 1,03 |
| `i28` active-camo | 9 | 3,25 % | 20 | 0,47 % | 6,95 |
| `i57` / `i59` ability | 7 | 2,53 % | 19 | 0,44 % | 5,69 |
| `i54` mobility | 11 | 3,97 % | 103 | 2,41 % | 1,65 |

**Champs dont TOUS les instants tombent dans une fenetre de decollage : 0.** Et le zoom au record
dit la meme chose au tick pres (`TestMouvement522Fenetre`) :

```
+ 84.033 s · 2 comps : i0 i25
* 84.050 s · 3 comps : i0 i1 i25          <- LE DECOLLAGE
+ 84.066 s · 5 comps : i0 i1 i25 i57 i59  <- +16 ms : LE SPRINT S ETEINT
+ 84.083 s · 4 comps : i0 i1 i25 i28
```

**`i57`/`i59` est un CONSEQUENT, pas un declencheur** : le sprint se coupe 16-17 ms APRES quatre
des six decollages (84,066 · 85,251 · 89,338 · 96,979 s) et ne dit rien des deux autres (86,602 et
93,126 n ont aucune transition de sprint). Un declencheur precede ; celui-la suit, et il manque un
tiers des sauts.

## 4. Le canal d entree : illisible, pas refute

D2 (5.14) placait l espoir dans les bits d action de la vue de controle. `TestMouvement522Controle`
les DATE : **170 ouvertures** de la garde sur tout `bfecd02b`, la plus proche de la fenetre du
temoin a 74,040 s puis 160,309 s — **zero entre 80 et 100 s**.

**Ce n est pas une conclusion negative**, et `TestMouvement522Fermeture` dit pourquoi :

| fenetre [80 ; 100] s | compte |
|---|---:|
| paquets delta | 1 199 |
| **fermes (reste dans [0 ; 7] a bits NULS)** | **0** |
| terminateur de vue C atteint, reste > 7 bits | 1 115 |
| rang 1 non ferme / non localise | 84 |

Les 12 entrees de controle lues dans cette fenetre le sont donc a un OFFSET FAUX. Le maillon est
nomme et il est deja au registre : **le trou du rang 1 sur film dense** (D1 (5.14), nomme au 5.15,
en cours au lot 5.21). Tant qu il tient, `bfecd02b` ne peut ni prouver ni refuter un declencheur
dans le canal d entree.

## 5. L ecrivain : le saut est une action d unite, bit 0x16

Methode du piege 6 de la passation 5.11 — **par le CONSOMMATEUR, jamais par l offset**.
`FUN_140a3f6a4` enregistre les fonctions de script ; `unit_action_test_jump` (`143c2e6a8`) y est
branche sur `FUN_142b79bc4` :

```
FUN_142b79bc4()  ->  FUN_142b7dff4(FUN_140acc920(), 0x16)

FUN_142b7dff4(unite, action)
    si unite == 0xffffffff : rien
    mot = *(u64*)( *(TLS + 0x468) + (unite >> 1 & 0x7fff) * 8 )
    return (mot >> action) & 1
```

**L ENUM DES ACTIONS D UNITE**, releve sur les 25 enregistrements qui passent une constante en
clair (les 31 `player_action_test_*` passent par un autre pont) :

| action | bit | action | bit | action | bit |
|---|---:|---|---:|---|---:|
| `action` | 0 | `accept` | 2 | `cancel` | 3 |
| `primary_trigger` | 4 | `secondary_trigger` | 5 | `grenade_trigger` | 6 |
| `melee` | 7 | `rotate_weapons` | 8 | **`jump`** | **0x16** |
| `equipment` | 0x17 | `context_primary` | 0x18 | `vehicle_ability_primary` | 0x19 |
| `vehicle_ability_secondary` | 0x1a | `vehicle_ability_tertiary` | 0x1b | `look_relative_up` | 0x1c |
| `look_relative_down` | 0x1d | `look_relative_left` | 0x1e | `look_relative_right` | 0x1f |
| `move_relative_fwd` | 0x20 | `move_relative_back` | 0x21 | `move_relative_right` | 0x22 |
| `move_relative_left` | 0x23 | `start` | 0x24 | `back` | 0x25 |
| `vision_trigger` | 0x26 | `dpad_up` | 0x29 | `dpad_down` | 0x2a |
| `dpad_left` | 0x2b | `dpad_right` | 0x2c | | |

**Ce mot n est pas un champ d objet** : il vit dans une table thread-locale indexee par unite,
donc aucun deserialiseur de `ti=35` ne peut l ecrire (la liste mesuree au 5.7.1 : `0x4dc`,
`0x544`/`0x548`/`0x726`, `0x7e8`/`0x7ec`, `0xaa8`, `0x11f8`/`0x1295`/`0x1296`, `0x129c`, `0x12b4`,
`0x12e4`, `0x1324`). C est la meme forme de preuve qu au 5.11.3 pour l etat aerien, en plus forte.
Son ECRIVAIN n est pas trouve dans le budget du lot (§4, D6).

## 6. Le « bloc d action non porte » etait deja au depot

`FUN_1406d025c`, que D2 (5.14) donnait comme non porte, est **le meme deserialiseur** que celui
qu `i19 unit-actor-control` appelle depuis `FUN_1408f0778`. Le depot le porte EN ENTIER depuis le
lot 2.7 sous le nom `consume1406d025c` :

```
gate R(1) ; si 0 -> rien
  R(1) ; si 1 : R(3) + R(3)          FUN_1431ab1ec(p, mot, bit) -> u16 p[mot], bits 0..2
  R(1) ; si 1 : R(2) + R(2)          FUN_1431ab1cc(p, oct, bit) -> u8  p[4 + oct], bits 0..1
  R(1) ; si 1 : R(1) + R(1)
               FUN_1431a0bbc  R(1)[+R(8)]
               FUN_1431a0abc  R(1)[+R(10)]
               FUN_1431a0cbc  (bloc de quaternion)
  R(3)                               FUN_1406d0f20 -> p[6]
  si un drapeau de p[0] : FUN_1406d00ec   R(1)[+R(2)]
  si un drapeau de p[4] : FUN_1406d00ec   R(1)[+R(2)]
  FUN_142f26740                      queue
```

Ce n etait donc pas un trou de grammaire mais un trou de **cablage**. Cable au lot 5.22.2, aucune
largeur neuve, borne posee a la SORTIE (`br.BitPos() <= frameLen`) :

| film | paquets fermes AVANT | APRES | dont bits TOUS NULS | portant un 1 |
|---|---:|---:|---:|---:|
| `bfecd02b` | 2 884 / 30 387 | **2 900** | **2 900 / 2 900** | **0** |
| `dad793c7` | 5 354 / 5 365 | 5 354 | 5 354 / 5 354 | 0 |

**La SEMANTIQUE de ce bloc, elle, n est etablie ni d un cote ni de l autre** : le depot le nomme
« orientation / matrice d inertie » dans `unit_control.go` et « bits d action » dans
`frame_vue_controle.go`, et les setters sont generiques (ecrire un bit dans un mot). §4, D5.

## 7. La generalisation : 306 vies, aucun candidat

`TestMouvement522Generalisation`, chaque vie contre SES fenetres de decollage :

| film | records `ti=35` | desyncs | vies | episodes fermes | retenus | vies qui sautent | dedans / hors |
|---|---:|---:|---:|---:|---:|---:|---|
| `bfecd02b` | 129 572 | 4 | 68 | 1 513 | 252 | 54 | 4 954 / 124 618 |
| `4f77afc1` | 400 697 | 22 | 238 | 6 497 | 1 209 | 193 | 10 524 / 390 173 |

`COMPOSANTS EXCLUSIFS AUX FENETRES DE DECOLLAGE : []` sur les deux. Les facteurs les plus hauts
sont les confondants nommes : `i28` 5,01 / 6,79 · `i57`/`i59` 4,24 / 6,04 · `i32` surchauffe 2,45.
**`i16 object-physics-flags-component`** — le meilleur candidat de NOM du registre — est declare
14 fois sur 129 572 records et 20 fois sur 400 697, **jamais** en fenetre de decollage : il ne peut
pas porter un saut par vie et par minute.

## 8. Ce qui est publie, et ce qui ne l est pas

- **PAS de genre `stances[].kind` `jump` LU.** Ce qui n est pas prouve n est pas publie.
  `jumpDerived` reste le seul genre du saut, et son nom dit qu il est calcule.
- **`mobility` devient `clamber`** — « Escalade » / « Clamber » —, schema **67 -> 68**. L oracle
  n est pas une chaine du binaire (le lot 5.13.2 avait mesure qu il n y en a pas) : c est l ecran.
  L utilisateur a confronte dans Theater neuf intervalles de ce genre, pris sur `bfecd02b` —
  neuf escalades de rebord, **9/9**, aucun contre-exemple. Le vocabulaire du jeu corrobore :
  `_action_hoist`, `_action_vault`, `_action_climb_attach`, `_action_climb_detach` (`143ca0100`),
  `CharacterPhysicsModeClambering` (`143df73d0`, `FUN_1406b8244(idx) == 2`).

Revisions : `grammar.Rev` `grammar-2026-09-22.8 -> .9` (cablage du bloc d action) puis `.9 -> .10`
(l etiquette de genre) ; `facts.Rev` INCHANGEE a `killsource-2026-09-22.2` (`killsource` ne lit
aucun etat de mouvement) ; `replay.SchemaVersion` **68**.
