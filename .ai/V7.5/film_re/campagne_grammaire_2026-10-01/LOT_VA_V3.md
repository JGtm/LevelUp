# LOT VA, ÉTAPE V3 — La variante de partie du film décide les genres 85 et 116 (2026-10-06)

Branche `feat/cg3-vue-a`, base `8c83e2d3a` (tête de V2 ; code identique à `30d94a60b`). Worktree
`LevelUp-wt-cg3-vuea`. Pièces : `scratchpad/cg3-V3/`. Ghidra `HaloInfinite.exe` HI_1_13_0, base
0x140000000, HTTP 127.0.0.1:8089, lecture seule ; relevés versés sous `va_ghidra/` (57 fichiers neufs).

Conventions : **lu** = lu dans Ghidra ou dans le film ; **mesuré** = compté ; **estimé** = dérivé d'une
mesure par une hypothèse écrite.

## 0. Statut

- **116 teleport_effects : `[x]`.** Ce qui décide sa branche est le type de moteur de la variante de
  partie, que le film porte (lu dans le jeu, lu dans le film : `m_gameEngineType` = 2 sur les 1 657
  films du cache à section d'identification, donc `DAT_145121140` = 3 ≠ 1). Porté.
- **85 PlayerKilledEvent : `[!]` sur tous les films dont la vue A se lit.** La variante est lue :
  `killcamEnabled` = faux, mais **`playOfTheGameEnabled` = vrai** sur les 1 648 films HI_1_8_0 (6 sur
  13) à HI_1_13_0 du cache. La garde de la queue vaut alors `play_of_the_game_enabled &&
  FUN_1406aed00()`, un réglage nommé du processus posé à l'exécution (R2 §4, lu) que **le film ne porte
  pas** : la valeur ne se devine pas, le 85 ne se lit pas. La règle générale est portée (partie fixe
  seule quand les drapeaux du film rendent la garde fausse quels que soient les réglages) ; elle ne
  décide sur le cache que 9 films (`playOfTheGameEnabled` faux : HI_1_4_1, HI_1_5_1, 7 HI_1_8_0), dont
  la vue A ne se lit pas (versions différentes). Instruction complète §1.4.
- Étape **retenue** et committée : gain mesuré en cuisson et à la sonde, aucune mort, valeur ou voie de
  `killsource` changée, carte v2 identique à l'octet (gate 2 tenu sans exception), gate de corpus rc=1
  par deux lignes `[FILET]` de `d9781168` instruites (§5) — **admission à décider par le pilote**.
- `origin/feat/v75` a avancé (`1518e6f10`) mais porte toujours `8dfadd07e` (lot 2.7.a de la RI) :
  **non fusionné** (§9, écart E-1).

## 1. Ce qui est lu dans le jeu

### 1.1 Le corps de `chunk_00`, jusqu'à la table des joueurs

Lecteur `FUN_1407ee138` (appelé par `FUN_14299ac50` sur `film + 0xCE690`, avec la version du film),
écrivain `FUN_1407ec560` (adresses des appels : `ASM_1407ec560_sites.txt`). Le corps est la
structure des options de partie (pas `0x1134F0`, celle que `FUN_1404f1614` désigne pour la partie
courante) :

| Champ | Lecteur | Écrivain |
|---|---|---|
| R(3) R(3) R(2) R(7) R(64) R(32) R(3) R(32) R(32) R(32) | en ligne, `FUN_1406d676c(.., 0x40)` | id., `FUN_1406d60f4(.., 0x40)` |
| message Bond (`options + 0xE951C`) | `FUN_140ee5f40` | `FUN_140b857d8` |
| chaîne ≤ 0x80 octets (`+ 0xEA694`) | `FUN_1407cbc24` (sans nul en 0x80 octets : erreur du flux) | `FUN_1407ebe7c`, R9D = 0x80 |
| R(32) R(32) (`+ 0xEA714`, `+ 0xEA718`) | en ligne | en ligne |
| R(1) ; si 1 : R(0x700) (`+ 0xEA72C`) | `FUN_140ee5ef8` (`MOV R9D,0x700` 142442cf2) | `FUN_1410bc140` (`MOV R9D,0x700` 1424a9822) |
| R(1) R(1) R(2) R(1) R(32) R(32) R(1) (`+ 0xEA719` … `+ 0xEA80C`) | en ligne | en ligne |
| R(1) présence ; si 1 : message Bond de la variante (`options + 0x28`) | `FUN_140b3a118` | `FUN_14051a4b8(type) != 0` puis `FUN_140b85504` |
| deux chaînes ≤ 0x100, R(32), R(0x6C0) | `FUN_1407cbc24` x2, `FUN_1406d676c` x2 | `FUN_1407ebe7c` x2, `FUN_1406d60f4(.., 0x20)`, `(.., 0x6C0)` |
| 32 enregistrements de joueur | boucle `FUN_1407ee98c` | boucle `FUN_1407ecb08` |

### 1.2 Les messages Bond : CompactBinary v2

Les deux messages passent par un adaptateur de flux (`PTR_FUN_143d45e08` = {`FUN_1406d5f18`, …},
`MEM_143d45e08.json`) qui écrit huit bits par octet dans le flux. Le protocole est Microsoft Bond
CompactBinary, version 2 (`local_13c8 = 2` / `MOV word ptr [RBP-0x70],2` ; lecteurs `local_60 = 2`) :

| Élément | Écrivain lu |
|---|---|
| début de structure : longueur en octets (v2, hors structure de base) | `FUN_140ac755c` |
| entier variable : 7 bits par octet, poids faible d'abord, 0x80 = suite | `FUN_140ac7668` |
| en-tête de champ : `type \| id<<5` (id < 6), `type\|0xC0` + 1 octet, `type\|0xE0` + 2 octets | `FUN_140ac75e8` |
| liste : `type \| (n+1)<<5` si n < 7 (v2), sinon type puis n | `FUN_140ac7430` |
| fin de structure : un octet 0 (1 pour une structure de base) | `FUN_1411b3740` |
| booléen : en-tête type 2 + un octet | `FUN_1424d6668` |
| int32 : en-tête type 0x10 + zigzag | `FUN_140d1a268` |
| champ optionnel écrit seulement s'il diffère de son défaut (modificateur `+0x50`, défaut `+0x58/+0x60`) | `FUN_140b23f64`, `FUN_140d1a268`, `FUN_1410301c4` |

### 1.3 Le chemin des champs de la variante

`FUN_140b85504` copie la variante depuis `variante + 4` (`LEA RDX,[RAX+0x4]` 140b85566,
`FUN_140958628` → `FUN_140958670`, 0xE3A68 octets) dans l'un de quatre pointeurs, puis écrit une
structure à quatre champs listes (`FUN_1410bcd3c`, `FUN_1410bcee0` : en-têtes 0x0b, 0x2b, 0x4b, 0x6b,
type 0xA). Le lecteur (`FUN_140958494`) recopie le pointeur non nul dans la variante.

| Chemin | Fonction | Champ | Métadonnée lue |
|---|---|---|---|
| variante.0 (`V + 0`) | `FUN_141099830` → `FUN_1410301c4` | .0 int32, `V[0]` = `variante + 4` | `"m_gameEngineType"`, défaut 0 (`FUN_14114076c`, `FUN_14014dce0`) |
| variante.1 (`V + 0x1F0`) | `FUN_141098e84` → `FUN_141057d14` | .0 (`FUN_141099c88` → `FUN_141025dd0`) | — |
| variante.1.0.5 (`V + 0x1F0 + 0x44` = `variante + 0x238`) | `FUN_141097cbc` → `FUN_141097cec` | structure | `"i343.NetProtocol.GameOptions.PlaybackSettings"` (`FUN_141154c7c`) |
| .0 booléen (`variante + 0x238`) | `FUN_140b23f64` | | `"killcamEnabled"`, défaut faux (`FUN_14014cc60`) |
| .2 booléen (`variante + 0x240`) | `FUN_140b23f64` | | `"playOfTheGameEnabled"`, défaut faux (`FUN_140150510`) |

`FUN_14051a4b8` (1→1, 2→3, 3→2, autre→0) puis `FUN_140a93ec8` posent `DAT_145121140`
(`FUN_140a938b4`, variante de `FUN_1404f2650` = `options + 0x28`) : il vaut 1 si et seulement si
`m_gameEngineType` vaut 1 (`FUN_142b5c658` pose 1).

**Fermeture (mesurée, sonde `va_v3_research_test.go`, 1 664 répertoires du cache)** : la marche du
corps tombe sur le premier enregistrement que `ReadPlayerTable` trouve par son propre balayage sur
**1 656 films sur 1 657** à section d'identification (le 1 657e, HI_1_5_1, n'a pas de table des joueurs
lisible ; sa variante se lit) ; chaque longueur Bond tombe sur son octet de fin. Valeurs lues :

| Build | Films | m_gameEngineType | killcamEnabled | playOfTheGameEnabled |
|---|---|---|---|---|
| HI_1_13_0 | 1 401 | 2 | faux (omis) | vrai |
| HI_1_12_0 | 147 | 2 | faux | vrai |
| HI_1_11_0 | 57 | 2 | faux | vrai |
| HI_1_10_0 | 34 | 2 | faux | vrai |
| HI_1_9_0 | 3 | 2 | faux | vrai |
| HI_1_8_0 | 13 | 2 | faux | vrai 6, faux 7 |
| HI_1_5_1, HI_1_4_1 | 2 | 2 | faux | faux |

### 1.4 85 : instruction du `[!]`

Lecteur `FUN_14104bd08`, écrivain `FUN_142f18fd0` (même garde, `FUN_14076d018 || FUN_14076cffc`) :

- `FUN_14076d018` = `DAT_1451789b8 && FUN_1406aed00() && DAT_145121140 != 1 && variante[+0x238]` ;
- `FUN_14076cffc` = `DAT_145178a48 && FUN_1406aed00() && variante[+0x240]`.

Lu : `variante[+0x238]` = killcamEnabled, `[+0x240]` = playOfTheGameEnabled (§1.3) ; `DAT_145121140 != 1`
(le film porte 2, donc 3). `DAT_1451789b8` / `DAT_145178a48` = réglages nommés `kill_playback_enabled`
/ `play_of_the_game_enabled` (`FUN_140373a60`, `FUN_140373b40`, enregistrés à faux par
`FUN_140ad2d3c`, R2 §4) ; `FUN_1406aed00` lit un état de fil et le mode de la partie courante.
Avec killcamEnabled faux, la première branche est fausse ; avec playOfTheGameEnabled vrai, la seconde
vaut celle du réglage d'exécution, que le film ne porte pas. **La queue (R(32), R(32), R(4)) ne se
décide pas dans le film** : 85 reste illisible (refus, comme avant). Le diagnostic de R2 (partie fixe
seule s'accorde au début localisé dans 97 % des cas) suggère que l'enregistreur avait le réglage à
faux ; c'est une mesure, pas une lecture : non retenue (critère de l'utilisateur).

### 1.5 116

Lecteur `FUN_142ef93e0` : R(1) ; si 1 : `FUN_140c5f938(.., mode 0)` ; R(1) ; `FUN_14080d69c` (rend
son R(1) ; si 1 : R(32)) ; si 1 : deux positions `FUN_1424e0e38` (CALLs 142ef944a, 142ef945d) =
`FUN_14076e494(.., 0x10, 0, 0, p6 = 0)` (`ASM_1424e0e38.txt`). `FUN_140c5f938` en mode 0 :
`DAT_145121140 != 1` → `FUN_140c5fa84` (porté : `consumeObjectForwardAndUp`), sinon `FUN_142e29bac`
(non porté). Écrivain `FUN_142efa2a8` → `FUN_141f86118` : même branche sur `DAT_145121140`.

## 2. Ce qui change dans le code

- `profile/identite.go` : `FilmIdentity.Variante` (`VarianteDePartie` : `Lue`, `Presente`,
  `TypeDeMoteur`, `KillcamEnabled`, `PlayOfTheGameEnabled`).
- `grammar/film_variante_de_partie.go` (neuf, 290 lignes) : la marche du corps (`marcherLeCorps`) et
  la lecture Bond restreinte au chemin lu ; un champ d'un autre type à une place du chemin, une
  longueur qui ne tombe pas sur son octet de fin, une chaîne sans nul, un débordement du tampon
  (`Deborde`) : rien n'est rendu. Réutilise `lireChaine` (pas de seconde copie).
- `grammar/film_identity.go` : une ligne, `id.Variante = lireLaVarianteDePartie(...)`.
- `grammar/vue_a_charges_execution.go` : `varianteDuFilm`, `chargeJoueurTue` (85),
  `chargeEffetsDeTeleportation` (116) ; `vue_a_charges.go` : deux `case` ; `vue_a_versions.go` :
  champ `variante` de `grammaireDeLaVueA`, posé par `grammaireDeLaVueASousFilm`.
- Tests : `film_variante_de_partie_test.go` (vecteurs d'après les écrivains : un message Bond écrit
  champ par champ, défauts omis, id échappés, listes vides ; trois corps illisibles ; les sept bobines
  par build ferment sur la table des joueurs), `vue_a_variante_test.go` (85 sous sept variantes, 116
  sous six ; les deux bobines récentes), `lecteur_position_ratchet_test.go` (le site 116, deux CALLs
  relevés), `vue_a_lecture_test.go` (libellé du 85).
- Sonde `va_v3_research_test.go` (tag `research`).
- Révisions et goldens (§8).

Aucun fichier de la RI touché : `movement_states.go`, `distribuer.go`, `lecture/`, `canal_*`,
`film_context.go`, `marche_trames*.go`, `localisateur.go` inchangés.

## 3. Taux de vues A lues jusqu'au terminateur (sonde `TestVAV1FinDeVueA`, 20 films, carte du match posée)

Base = `git archive 8c83e2d3a`, tête = V3 (`sonde_base/`, `sonde_tete/`).

| Build | Paquets à vue A | Base | V3 |
|---|---|---|---|
| HI_1_13_0 | 64 897 | 63 692 (98,1 %) | **63 755 (98,2 %)** |
| HI_1_12_0 | 2 730 | 2 679 | 2 679 |
| HI_1_11_0 | 7 751 | 7 549 | 7 549 |
| HI_1_10_0 | 50 097 | 47 188 | **47 210** |
| HI_1_9_0 | 7 325 | 6 908 (94,3 %) | **6 964 (95,1 %)** |
| HI_1_8_0, HI_1_4_1, version-31, version-33 | 33 283 | 0 | 0 |

Par film :

| Film | Build | Paquets à vue A | Lus base | Lus V3 | Δ |
|---|---|---|---|---|---|
| `d9781168` | HI_1_13_0 | 8 737 | 8 523 | 8 586 | +63 |
| `11de8353` | HI_1_9_0 | 7 325 | 6 908 | 6 964 | +56 |
| `084a804d` | HI_1_10_0 | 17 149 | 16 074 | 16 085 | +11 |
| `1c4c63c2` | HI_1_10_0 | 24 997 | 23 504 | 23 513 | +9 |
| `111fa685` | HI_1_10_0 | 7 951 | 7 610 | 7 612 | +2 |
| 15 autres | | | | | 0 |

2 223 paquets changent d'état d'arrêt : 2 076 `non porté 85` → `refusé 85` (même arrêt, autre
libellé : la charge existe et refuse) ; **141 lus jusqu'au terminateur** ; 4 `non porté 116` → `refusé
116` (HI_1_10_0 : position à index d'une autre région, ou film sans carte) ; 2 arrêtés plus loin sur un
autre genre non porté (96, 90).

Les 141, contre la localisation de base (verdict de la sonde, monde de la carte v2) :

| Film | Avant | Verdict E | n |
|---|---|---|---|
| `d9781168` | non localisé | depuis E fermé | 28 |
| `d9781168` | non localisé | depuis E non fermé | 10 |
| `d9781168` | signature | accord / par chaîne | 21 / 2 |
| `d9781168` | signature | désaccord E < S, ni E ni S fermés | 2 |
| `11de8353` | signature | accord / par chaîne | 30 / 14 |
| `11de8353` | non localisé | depuis E non fermé | 9 |
| `11de8353` | signature, fermeture au bit | désaccord | 3 |
| `084a804d`, `111fa685`, `1c4c63c2` | signature, fermeture | désaccord (film PRÉFIXE : E non prouvé, chemin actuel) | 17 |
| `084a804d`, `1c4c63c2` | non localisé | depuis E non fermé | 5 |

Sans carte du match (`sonde_tete_sans_carte/`), les 116 de ces films sont tous refusés (62 HI_1_13_0,
46 HI_1_9_0, 12 HI_1_10_0) : leurs positions portent un index de plage (§4).

## 4. Table par film (carte v2, gate 2)

`research/cmd_fermeture -mode v2 -denominateur-fixe -paquets -plafond-gib 4`, table ECS figée, base =
`cg3-V2c/fusion2/carte_tete` (code identique à `8c83e2d3a`), tête = `cg3-V3/m1/carte`.
**`fermeture_paquets.tsv` et 12 autres tables identiques à l'octet** ; `fermeture_films.tsv` et le
résumé ne diffèrent que par `pic_octets`, `duree_ms` et la date. Gate 2 (`integ2/gate2.awk`) :

| Film | Sains base | Sains V3 | Utiles sains base | Utiles sains V3 | Juge (gagnés / perdus) | Factices | Pertes au bit |
|---|---|---|---|---|---|---|---|
| `0797ce72` | 20 772 | 20 772 | 176 835 | 176 835 | 0 / 0 | 0 | 0 |
| `084a804d` | 4 992 | 4 992 | 93 780 | 93 780 | 0 / 0 | 0 | 0 |
| `111fa685` | 4 090 | 4 090 | 44 298 | 44 298 | 0 / 0 | 0 | 0 |
| `11de8353` | 5 768 | 5 768 | 68 563 | 68 563 | 0 / 0 | 0 | 0 |
| `1c4c63c2` | 14 245 | 14 245 | 187 431 | 187 431 | 0 / 0 | 0 | 0 |
| `396cfc92` | 25 954 | 25 954 | 196 024 | 196 024 | 0 / 0 | 0 | 0 |
| `4f77afc1` | 31 015 | 31 015 | 822 451 | 822 451 | 0 / 0 | 0 | 0 |
| `50247b26` | 139 | 139 | 274 | 274 | 0 / 0 | 0 | 0 |
| `51ebbc0f` | 26 877 | 26 877 | 197 932 | 197 932 | 0 / 0 | 0 | 0 |
| `60ae07c4` | 13 948 | 13 948 | 83 685 | 83 685 | 0 / 0 | 0 | 0 |
| `a349fea8` | 424 | 424 | 3 654 | 3 654 | 0 / 0 | 0 | 0 |
| `a521164d` | 692 | 692 | 78 | 78 | 0 / 0 | 0 | 0 |
| `bcb6d393` | 7 119 | 7 119 | 46 309 | 46 309 | 0 / 0 | 0 | 0 |
| `bf15f7ab` | 29 914 | 29 914 | 228 138 | 228 138 | 0 / 0 | 0 | 0 |
| `bfecd02b` | 31 067 | 31 067 | 276 442 | 276 442 | 0 / 0 | 0 | 0 |
| `c75f33b8` | 27 272 | 27 272 | 180 599 | 180 599 | 0 / 0 | 0 | 0 |
| `d9781168` | 35 774 | 35 774 | 263 867 | 263 867 | 0 / 0 | 0 | 0 |
| `e5adf7b2` | 4 311 | 4 311 | 84 083 | 84 083 | 0 / 0 | 0 | 0 |
| `f75e7053` | 24 517 | 24 517 | 168 248 | 168 248 | 0 / 0 | 0 | 0 |
| `fb1a1a72` | 46 308 | 46 308 | 351 322 | 351 322 | 0 / 0 | 0 | 0 |
| **corpus** | 355 198 | 355 198 | 3 474 013 | 3 474 013 | 0 / 0 | 0 | 0 |

Gate 2 tenu (aucun film en baisse, sains ET utiles sains). D1 inchangé (44,4 % corpus, 91,2 %
HI_1_13_0). **Pourquoi la carte ne voit pas le gain (lu dans le code, mesuré)** : l'instrument ouvre
ses films par `grammar.ContexteDeFilm` (`NewFilmContext`, sans carte du match) ; `tablesDeLEntree`
d'une entrée vide rend `indexLisible = faux`, et toute position à index (genres 5, 6, 116) est refusée.
Les 116 lus en cuisson portent un index (§3, sonde sans carte). Découverte D-VAV3-1.

## 5. Cuisson : `replay-equiv` et gate de corpus (gate 6)

`replay-equiv` (20 films, racine factice `cg3-V1/repo`) contre les TSV de la base
(`cg3-V2c/fusion2/re_tete_tsv`) : divergent `artifact` (20, chaînes de révision), et seulement :

- `d9781168` : `continuousFire.stats`, `killsource`, `movementStates`, `movementStates.stats` ;
- `11de8353` : `vehicles`, `movementStates.stats`.

`objectives`, `killRefs`, `deaths` identiques partout. Instruction par les valeurs des étapes
(`TestVAV2Etapes`, base = `git archive 8c83e2d3a`, `etapes_base/`, `etapes_tete/`) :

- `d9781168 killsource` : seuls les compteurs non persistés `PacketsLocated` 8 611 → 8 642 et
  `LocalisationsALargeurLibre` 21 → 14 ; `cmd/killsource json` identique à l'octet (§6).
- `d9781168 continuousFire.stats` : `Reached` 43 431 → 43 468, `Closed` 36 082 → 36 116 (+34),
  `Unlocated` 74 → 36 (−38), `NotClosing` 7 326 → 7 329 (+3), `OpenViewB` 140 → 141 (+1), `Holes`
  7 563 → 7 529 (−34), `Entries` 208 542 → 208 749. Les 38 paquets non localisés que la vue A lue
  jusqu'au bout localise désormais : 34 ferment, 4 restent ouverts (non fermés ou ouverts en vue B).
  Aucun paquet fermé n'est perdu : la sonde ne change le début que de ces 38 et de 2 paquets en
  désaccord dont ni E ni S ne fermaient ; les 23 autres sont en accord (même début).
- `d9781168 movementStates` : 418 octets de plus (états lus dans ces paquets).
- `11de8353 vehicles` : `Deaths`, `Occupancy`, `Positions`, `Events`, `Creations`, `Aims`,
  `Keyframes`, `Scanned`, `Stats` **identiques** ; seul `DeathStats.CleanRecords` change (ti=2 611 →
  614, ti=5 2 187 → 2 192, et le même +3/+5 dans le second histogramme) : la marche à huit vues lit
  davantage d'enregistrements propres. `movementStates.stats` : `EventPacketsNewRecordStart` 1 273 →
  1 287. Film PRÉFIXE : E n'y sert que prouvé par la fermeture (décision (2)).

`replay-corpus-gate --reference=base --base=8c83e2d3a` (base cuite par le gate) : **rc=1**.

```
temoin       famille          base(8c83e2d3a)   HEAD    gains   pertes   chang.      duree base   cache  statut
bcb6d393     ctf_mono_manche      79     79        0        0        0      4.69s cuite  A+F    ok
fb1a1a72     ctf_multi_manche     79     79        0        0        0     11.52s cuite  A+F    ok
d9781168     oddball              79     79       20        2        2      8.42s cuite  A+F    PERTE
c75f33b8     assaut_bombe         79     79        0        0        0      5.54s cuite  A+F    ok
bf15f7ab     slayer               79     79        0        0        0      8.23s cuite  A+F    ok
51ebbc0f     deux_manches         79     79        0        0        0       6.3s cuite  A+F    ok
084a804d     vehicules            79     79        0        0        0    1m6.22s cuite  A+F    ok
0797ce72     region_index_2_bits     79     79        0        0        0      6.65s cuite  A+F    ok
111fa685     version_39           79     79        0        0        0     20.66s cuite  A+F    ok
e5adf7b2     version_40_build_1_11     79     79        0        0        0     22.12s cuite  A+F    ok
60ae07c4     version_37           79     79        0        0        0     14.59s cuite  A+F    ok
a349fea8     version_33_sans_identification     79     79        0        0        0   1m12.42s cuite  A+F    ok
a521164d     version_33_build_1_4_1     79     79        0        0        0     19.52s cuite  A+F    ok
11de8353     version_38_build_1_9_0     79     79        0        0        0      20.1s cuite  A+F    ok
50247b26     version_31_sans_identification     79     79        0        0        0     37.55s cuite  A+F    ok
bfecd02b     vehicules_v41_utilisateur     79     79        0        0        0      9.05s cuite  A+F    ok
4f77afc1     equipement_origine_utilisateur     79     79        0        0        0     48.79s cuite  A+F    ok
396cfc92     strongholds_zones     79     79        0        0        0     11.85s cuite  A+F    ok
f75e7053     koth_collines        79     79        0        0        0      6.42s cuite  A+F    ok
```

Banc de vérité : **19 / 19 « ok »**, 0 `MANQUE`, 0 `FAUX`. `d9781168` : `[gain] P-1 paquets fermés au
bit près 36 082 → 36 116`, `[info] R-1 repli repli_localisation_largeur_libre 21 → 14`. Les deux
lignes `[FILET]` :

- `coverage.continuousFire.holesNotClosing 7 326 → 7 329` et `holesOpenViewB 140 → 141` : instruites
  ci-dessus — 4 des 38 trous « non localisés » devenus localisés sans fermer. **Reclassement, pas
  perte** : le total des trous baisse de 34, les fermés montent de 34. Changements : `stances` du slot
  565 (durée 134 → 122) et `sprint` 741 → 743 : postures lues dans les paquets nouvellement localisés.

L'admission du rc=1 relève d'une décision datée (pilote).

## 6. `killsource` (gate 3) et vérification (d)

`cmd/killsource json`, 20 films (19 témoins et `1c4c63c2`), binaire de `8c83e2d3a`
(`cg3-V2c/fusion2/ks_tete`) contre binaire du lot (`cg3-V3/m1/ks`) : **20 JSON identiques à l'octet**,
rc=0 partout. Aucune mort, valeur ni voie ne change : `killsource.Rev` reste `killsource-2026-09-27`
(D23), golden régénéré à révision constante.

Vérification (d), sur la marche à huit vues de production (le canal des morts n'est pas dans cette
base) : morts et occupation des véhicules identiques (§5, `11de8353`, `d9781168`) ; le compte
`repli_localisation_largeur_libre` **baisse quand E localise** : 21 → 14 sur `d9781168` (film ÉGALE),
1 030 = 1 030 sur `11de8353` (film PRÉFIXE, E non prouvé là où la marche de `killsource` le
demanderait).

## 7. Gates (sorties exactes)

Environnement : `GOCACHE=go-build-cg3-vuea`, CGO msys64 ucrt64, une commande go à la fois, films lus
en place. Pièces `scratchpad/cg3-V3/g1/`.

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0. `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc=0.
3. `go test ./internal/archlint/ -count=1` : `ok levelup/go-api/internal/archlint 54.123s` (rejoué
   après l'ajout de la sonde).
4. G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) : rc=0,
   21 paquets `ok`.
5. `golangci-lint` (`GOLANGCI_LINT_CACHE` au scratchpad) : paquets touchés (`grammar/...`,
   `profile/...`, `facts/...`, `types/...`), toutes les issues : `0 issues.` ; recette CI
   `--new-from-merge-base=origin/main ./...` : `0 issues.` ; sous `--build-tags=research`, toutes les
   issues : 26 issues (goconst 20, goimports 2, unparam 1, unused 3), **le même ensemble
   qu'à la base** (`git archive 8c83e2d3a`, même commande : 26, mêmes fichiers et lignes ; seul le nom de
   constante suggéré par une issue goconst varie) ; aucune dans un fichier de ce lot ;
   `--new-from-rev=HEAD` : `0 issues.`
6. Carte v2 : §4. `killsource` : §6. `replay-equiv` : « BILAN : 0 identique(s), 20 different(s) »
   contre les références non re-figées (rc=1, attendu) ; étapes : §5. Gate de corpus : §5.
7. Mutations : §10.
8. Performance (gate 4, mesurée, non attribuée) : carte v2, somme des durées 84 320 → 96 046 ms, pic
   max 308 → 318 Mio, sur deux exécutions isolées (bruit d'exécution du même ordre entre les runs de
   V2). La marche du corps enjambe chaque structure Bond par sa longueur, une fois par film
   (`ReadFilmIdentity`).

## 8. Révisions

- `profile.Rev` : `profile-2026-10-06` → **`profile-2026-10-07`**. `profile-2026-10-06.2`, seul rang sans
  trou du 2026-10-06 (le garde-rail refuse `.1` et exige les rangs sans trou), est pris par `feat/v75`
  et le lot LR ; même geste que V1 (`profile-2026-10-06` pris le 2026-10-05). Golden régénéré.
- `grammar.Rev` : `grammar-2026-10-06.3` → **`grammar-2026-10-06.4`** (aucune autre branche ne le
  porte), entrée de chronique. Golden régénéré.
- `killsource.Rev` constante (D23, §6) ; complément de chronique ; golden régénéré.
- `objectives` : constante ; étape `objectives` de `replay-equiv` identique sur les 20 films ; complément
  de chronique ; golden régénéré.
- `source.Rev`, `replay.SchemaVersion` (79) : inchangés. `types/testdata/shapes.golden` et les 8
  fixtures web `replay_schema_79_*` + manifeste régénérés (`REPLAY_CONTRACT_UPDATE=1 … -run
  ContractFixtures -update`) : contenu identique une fois les chaînes `grammar-…` / `profile-…`
  neutralisées (vérifié fixture par fixture).

## 9. Vérifications de la RI (a)–(g)

- (a) Tête aux canaux identique paquet par paquet : V3 ne touche ni `rangerLaTete` ni `teteDe` ;
  `TestLaTeteEstLueALIdentique` (20 000 payloads, 3 grammaires) vert.
- (b) Vue A lue une fois ; étendues de `lecture.Paquet` inchangées hors genres désormais portés (85
  refusé sans variante décidée, 116 porté) ; préambule de `DecodeFrameViewsCurseur` non touché.
- (c) `DebutParVueA`, ADR 0037 IR-6 : non touchés par V3 (V2).
- (d) §6 : `repli_localisation_largeur_libre` 21 → 14 (`d9781168`), 1 030 = 1 030 (`11de8353`) ; morts,
  occupation et `killsource` inchangés.
- (e) `film_context.go` non touché (toujours 500 lignes) ; l'ajout vit dans un fichier neuf.
- (f) `consumeVueA` : supprimé en V1, non réintroduit (ratchet `film_vue_a_lecteur_unique_test.go`
  vert).
- (g) `marche_trames_test.go`, `distribuer_tetes_test.go` : non touchés ; seul le libellé du 85 de
  `vue_a_lecture_test.go` change (raison : la charge existe, elle refuse sans variante décidée).
- Fichiers de la RI touchés : **aucun**.

## 10. Mutations (`-overlay`, suite entière du paquet `grammar` ; ROUGE attendu)

`scratchpad/cg3-V3/mutations.sh`, `mutations.txt` : **11 / 11 ROUGES**.

| Mutation | Tests rouges |
|---|---|
| m01 killcamEnabled lu au champ 1 | `TestLeCorpsSeLitJusquALaTableDesJoueurs` |
| m02 playOfTheGameEnabled lu au champ 3 | `TestLaVarianteDesBobines`, `TestLeCorpsSeLit…`, `TestLesBobinesRecentesDecidentLeurs85Et116` |
| m03 drapeau inversé | les trois mêmes |
| m04 longueur de structure ignorée | `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu` |
| m05 m_gameEngineType sans zigzag | `TestLaVarianteDesBobines`, `TestLeCorpsSeLit…` |
| m06 chaîne sans nul acceptée | `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu` |
| m07 garde du 85 sans playOfTheGame | `TestLeJoueurTueSeLitQuandLeFilmDecideSaQueue`, `TestLesBobinesRecentes…` |
| m08 condition de moteur du 85 inversée | `TestLeJoueurTue…` |
| m09 116 : branche du moteur 1 lue comme l'autre | `TestLesEffetsDeTeleportationSuiventLeMoteurDuFilm` |
| m10 116 : positions au niveau 0xf | `TestLesEffetsDeTeleportation…` |
| m11 variante absente du film lue | `TestLeJoueurTue…` |

## 11. Écarts

- **E-1 — `feat/v75` non fusionné.** `git fetch` du 2026-10-06 : `origin/feat/v75` = `1518e6f10`
  (lot « tendances », aucun fichier du décodeur), au-dessus de `8dfadd07e` (lot 2.7.a de la RI, canal
  des morts) dont la consigne dit qu'il n'est pas fusionné avant ce lot, et dont la fusion d'essai de
  V2 rendait 19 conflits (LOT_VA_V2 §7.3). Fusion non faite ; gates joués contre `8c83e2d3a`.
- **E-2 — révision de profil au 2026-10-07** (§8) : la date est celle du premier rang libre et sans
  trou, pas celle du lot.
- **E-3 — gate 2 sans effet mesurable** : la carte v2 ne pose pas la carte du match (D-VAV3-1) ; le
  gain est mesuré par la sonde (avec carte) et par la cuisson (gate de corpus).
- **E-4 — 85 porté comme règle générale** bien qu'il ne décide sur le cache aucun film dont la vue A se
  lit (9 films anciens) : la règle est celle de l'écrivain, testée par vecteurs et mutations (m07, m08,
  m11) ; ce n'est pas un réglage choisi à la mesure.
- Pas de revue adversariale de fin de lot dans cette étape (prévue par le plan pour V3) : à lancer par
  le pilote.

## 12. Découvertes

- **D-VAV3-1** — La carte v2 (`cmd_fermeture`) ouvre ses films sans carte du match : toute position à
  index des genres 5, 6 et 116 y est refusée. Le gate 2 ne voit donc ni ces genres ni ce qu'ils
  débloquent. Lot d'instrument à part (poser la carte du manifeste du corpus, comme R2 le faisait par
  surcouche), sans quoi V1/V2/V3 sont mesurés en cuisson seulement pour ces genres.
- **D-VAV3-2** — Le corps de `chunk_00` se lit maintenant jusqu'à la table des joueurs par la grammaire
  du jeu (1 656 / 1 657 films). `ReadPlayerTable` trouve encore la table par balayage ; la marche du
  corps en donne le premier bit (lot dédié possible, hors V3).
- **D-VAV3-3** — `m_gameEngineType` = 2 (`DAT_145121140` = 3) sur tout le cache : la branche
  `FUN_140c5fa84` que la production prend pour le composant i2 (`consumeObjectForwardAndUp`, R2 §4)
  est désormais fondée par le film, pour le mode 0 de `FUN_140c5f938` ; non branché sur ce composant.
- **D-VAV3-4** — `playOfTheGameEnabled` vaut vrai sur tous les films HI_1_9_0 à HI_1_13_0 : la queue du
  85 n'est décidable que par le réglage d'exécution `play_of_the_game_enabled`. Seule une lecture
  (observation du processus, Cheat Engine, ou un film enregistré avec le réglage connu) la fixerait.
- **D-VAV3-5** — `FUN_14080d69c` rend le bit qu'il lit (le 116 s'en sert comme garde) ; le portage
  `consumeGateR` ne le rend pas, d'où une lecture en ligne dans `chargeEffetsDeTeleportation`.

## 13. Pièces

`scratchpad/cg3-V3/` : `gh/` (décompilations), `sondes/`, `v3_parc.tsv` et `v3_prod_parc_final.tsv`
(marche du corps, 1 664 films), `m1/` (binaires, carte, killsource, `re_tsv`, `gate.log`,
`corpus_gate.json`), `gate2.tsv`, `sonde_base/`, `sonde_tete/`, `sonde_tete_sans_carte/`,
`sonde_changes.tsv`, `etapes_base/`, `etapes_tete/`, `g1/` (gates 1), `mut/`, `mutations.txt`,
`src_base/` (`git archive 8c83e2d3a`).
