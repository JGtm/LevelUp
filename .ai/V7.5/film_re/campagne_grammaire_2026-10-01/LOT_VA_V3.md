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
  **non fusionné** (§9, écart E-1). **Levé par la revue (§15)** : fusionné à `fed1efed2`, puis à
  `f8a14b3b9`.
- **Revue du lot (2026-10-06, §15)** : quatorze constats, tous fondés sur pièces ; dix corrigés dans le
  code, les tests ou la doc, quatre soumis (constats 1, 4, 10, 13). Gate 2 tenu (+53 644 sains contre
  `fed1efed2`, aucun film en baisse) ; gate de corpus rc=1 avec un `FAUX` V-6 sur `084a804d`, non
  tranché, à admettre par le pilote ; mutations 6 / 6 ROUGES.
- **Décisions du pilote (2026-10-06, §15.6, §16)** : `feat/v75` fusionné à `b033d30f0` ; la classe ÉGALE
  exige la version majeure 0x29 que le jeu joue (`bcb6d393` HI_1_12_0 devient PRÉFIXE) ; D23 tenue
  (killsource constant, mesuré sur 20 films, estimé sur le parc) ; `SchemaDesFaits` 4 → 5 ; la suite des
  genres présumée est marquée dans `lecture.VueA` ; le `FAUX` V-6 de `084a804d` est instruit : trajet
  vrai (pilotage puis abordage d'une Banshee), fausse alarme du banc. Gate 2 : +53 004 sains contre
  `b033d30f0`, aucun film en baisse ; mutations 4 / 4 ROUGES.
- **Corrections du contrôle indépendant (2026-10-06, §14)** : trois retenues sur pièces, aucune rejetée.
  Entrée de `.ai/thought_log.md` ajoutée ; trois formes Bond des écrivains (en-tête à id sur deux octets,
  fin de structure de base, borne de cinq octets de l'entier variable) désormais écrites par les
  vecteurs (mutations m12 à m14 ROUGES) ; portée de la lecture du schéma Bond écrite (§1.3). Code de
  production inchangé.

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

**Portée de la lecture (correction 3 du contrôle).** Le schéma Bond de ce chemin (variante.0 .0 pour
`m_gameEngineType`, variante.1.0.5 pour `PlaybackSettings`, ses champs .0 et .2) est **lu** dans
l'exécutable HI_1_13_0 seulement. Il est appliqué aux films des builds antérieurs, dont les films
PRÉFIXE HI_1_9_0 à HI_1_12_0, où la lecture du 116 dépend de `m_gameEngineType`. Sur ces builds, sa
cohérence est **mesurée** (marche fermée sur la table des joueurs pour 1 656 films sur 1 657, valeurs
du tableau ci-dessus), **non lue** dans leurs exécutables.

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

`scratchpad/cg3-V3/mutations.sh`, `mutations.txt` : **11 / 11 ROUGES** ; après les corrections du
contrôle (§14), `scratchpad/cg3-V3-corr/mut_rejeu.sh`, `mutations_rejeu.txt` : **14 / 14 ROUGES** (les
onze, rejouées sur le test corrigé, mêmes tests rouges, et m12 à m14).

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
| m12 id d'en-tête 0xE0 lu poids fort d'abord (m6 du contrôle) | `TestLeCorpsSeLit…` |
| m13 fin de structure de base retirée du cas accepté (m7 du contrôle) | `TestLeCorpsSeLit…` |
| m14 entier variable lu sur dix octets (m8 du contrôle) | `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu` |

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

## 14. Corrections du contrôle indépendant (2026-10-06)

Contrôle de `a91476e3b` (pièces `scratchpad/cg3-V3-ctl/`) : fond vérifié, une règle du dépôt non tenue et
trois mutations de la lecture Bond restées VERTES. Pièces de la correction : `scratchpad/cg3-V3-corr/`.

| # | Constat | Verdict sur pièces | Action |
|---|---|---|---|
| 1 | Aucune entrée de `.ai/thought_log.md` dans `a91476e3b` | Fondé : `git show --stat a91476e3b` ne liste pas le journal, alors que les commits de V1 et V2 en portent une | Entrée « lot VA, étape V3 » ajoutée après celle de V2 |
| 2 | m6, m7, m8 VERTES : `FUN_140ac75e8` (forme 0xE0) et `FUN_1411b3740` (fin de base, octet 1) jamais écrits par un vecteur ; borne de cinq octets de l'entier variable non tenue par un test | Fondé. Les branches ne sont pas retirées : ce sont des formes que les écrivains du jeu produisent (`FUN_140ac75e8` : `type \| 0xE0` puis l'id sur deux octets, poids faible d'abord, pour un id ≥ 0x100 ; `FUN_140ac755c` n'écrit pas de longueur pour une base, `FUN_1411b3740` écrit 1 à sa fin ; `FUN_140ac7668` écrit un `uint` de 32 bits, donc cinq octets au plus). Les retirer ferait mal lire une structure que le jeu écrit : ce n'est pas du code mort | `bondDeLaVariante` écrit, dans `PlaybackSettings`, une base vide (`finDeBase`) et un booléen d'id 0x200 ; vecteur « longueur de la variante sur six octets » (`entierDeTeteSurSixOctets`). m12–m14 (m6–m8 du contrôle) ROUGES (§10) |
| 3 | Schéma Bond lu dans HI_1_13_0 seulement, appliqué aux builds antérieurs | Fondé | Phrase « Portée de la lecture » au §1.3 |
| 4 | Décisions du pilote : rc=1 du gate de corpus, revue adversariale, écart E-1 | Hors périmètre de l'exécutant | Aucune action ; restent au §0, §5, §11 |

Choix des vecteurs : l'id 0x200 s'écrit `0xE2 0x00 0x02` ; lu poids fort d'abord, il vaut 2, l'id de
`playOfTheGameEnabled`, et pose le drapeau sur les vecteurs où il est faux. La base vide précède
`killcamEnabled` et `playOfTheGameEnabled` : refusée, la marche saute à la fin de la structure sans les
lire. L'entier de six octets garde sa valeur : lu sur dix, la marche irait jusqu'à la table des joueurs.

Gates rejoués (le code de production ne change pas : seuls `film_variante_de_partie_test.go` et les
documents changent ; carte v2, `killsource`, `replay-equiv` et gate de corpus ne sont donc pas
rejoués, leurs entrées étant identiques à `a91476e3b`) :

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./internal/games/halo_infinite/film/...` : rc=0 ; sous `-tags=research` : rc=0.
3. `go test ./internal/games/halo_infinite/film/internal/grammar/ -count=1` : `ok … 108.727s`.
4. `go test ./internal/archlint/ -count=1` : `ok … 59.195s`.
5. `golangci-lint run ./internal/games/halo_infinite/film/internal/grammar/...` (toutes les issues) :
   `0 issues.` ; `--new-from-rev=a91476e3b ./...` : `0 issues.`
6. Mutations : 14 / 14 ROUGES (§10).

`git fetch` du 2026-10-06 : `origin/feat/v75` = `1518e6f10`, inchangé depuis V3 ; non fusionné (E-1,
décision du pilote).

## 15. Revue du lot : fusion de `feat/v75` (`fed1efed2`) et corrections (2026-10-06)

Revue du lot (quatorze constats, pièces `scratchpad/cg3-revue-jeu/`, `cg3-revue-depot/`). Pièces de la
correction : `scratchpad/cg3-revue-corr/`. `git fetch` du 2026-10-06 : `origin/feat/v75` = `fed1efed2`
(lot 2.7.a de la RI, lot LR de la campagne, page Tendances) ; **fusionné** (feat/v75 a raison), la
consigne du constat 6 l'annonçant comme décision du pilote. Écart E-1 levé.

### 15.1 La fusion

`git merge origin/feat/v75` : 21 chemins en conflit, les mêmes que la fusion d'essai de la revue
(`cg3-revue-depot-mt.txt`) plus aucun. Résolution :

- `object_deaths.go`, `localisateur_test.go` : côté `feat/v75` ; `object_deaths_march.go` : supprimé
  (2.7.a retire la marche à huit vues ; la modification du lot n'a plus d'objet).
- `localisateur.go` : en-tête réuni — la cuisson range la fin de la vue A, le canal des morts reçoit les
  records de la cuisson et ne localise lui-même ([`debutRecupere`]) que la liste que la cuisson n'a pas
  localisée, seule la marche de `killsource` appelle `DebutDeLaVueB` ; « trois appelants » corrigé en
  deux (cuisson, `DebutDeLaVueB`).
- Révisions (constats 8 et 14) : §15.4.
- `shapes.golden`, goldens de révision, 8 fixtures web `replay_schema_79_*` et manifeste : régénérés ;
  les fixtures sont identiques à celles de `fed1efed2` une fois les chaînes `grammar-…` / `profile-…`
  neutralisées (vérifié fixture par fixture, `cg3-revue-corr/fx/`).
- Fichier non en conflit cassé par la fusion (constat 6) : `debut_par_vue_a_test.go` citait
  `ScanMarchFacts`, `marchPacketsOf`, `calibrateFrameConfig`, `newMarchTimeline`, `marchDebut`. Le test
  de la marche des morts est porté sur le canal des morts (§15.2, constat 6) ; les sondes
  `va_v2_research_test.go` (`TestVAV2Vehicules` porté sur le canal) et `va_v2_corr_research_test.go`
  (retirée : elle comparait les débuts E et S de la marche retirée, paquet par paquet ; ses mesures
  restent dans LOT_VA_V2) ; doc de `killsource/va_v2_profil_research_test.go` mise à jour.
- `grammar/rev_chronique.go` passait 500 lignes : rangs `grammar-2026-09-27` à `.3` versés, mot pour
  mot, dans `rev_chronique_archive_7.go` (geste ordinaire de rotation).

### 15.2 Constats : verdict sur pièces, action

| # | Gravité | Constat | Verdict sur pièces | Action |
|---|---|---|---|---|
| 1 | majeur | La classe ÉGALE ignore la garde de version majeure 0x29 du jeu | **Fondé.** `ghidra_1428e219c.c` : `if (*param_2 == 0x29) { … *(char *)(param_1 + 0x1ae) = (char)param_2[0x32d17]; … }` sinon `FUN_142988e98` ; `sonde_options.txt` : `147 maj=40 build=HI_1_12_0 classe=2 n=123` (et `57 maj=40 build=HI_1_11_0 classe=1`, PRÉFIXE) | Garde écrite dans l'en-tête de `vue_a_versions.go` (« la garde de version majeure ») et dans les classes de `localisateur.go`. **Règle non changée** : la décision (1) du 2026-10-04 définit la classe par la table ; options (a) et (b) de la revue **soumises à l'utilisateur** (§15.6) |
| 2 | mineur | `FUN_1406aed00()` non instruit dans la garde du 85 | **Fondé.** `ghidra_1406aed00.c`, `ghidra_1406aed60.c` relus : état de fil `TLS + 0x238`, puis `FUN_1406aed60(options) == 2`, qui rend `options[0]` (game_mode) si `options + 0xE2EE1` vaut 0, sinon 1. Le relevé de la revue omet l'état de fil, écrit aussi | En-tête §85 de `vue_a_charges_execution.go` : ce que vaut le terme, ce que le film porte (game_mode, 2 sur 1 657 films), ce qui reste à lire (octet `+ 0xE2EE1`). Refus inchangé |
| 3 | mineur | Doc du Script : « le lecteur d'un film rejoué lit le préfixe que l'enregistreur a écrit » | **Fondé.** `va_ghidra/FUN_1404f25f4.c` rend `uVar2 == 2`, `FUN_1404f293c.c` rend `uVar2 != 2`, même champ `options + 4` | Doc reformulée : en rejeu le lecteur lit toujours R(15) ; un enregistreur à simulation 2 n'est pas lisible par le jeu ; la règle portée (écrivain) le contredit alors ; aucun film du cache (sim = 3 sur 1 657) |
| 4 | mineur | Numérotation des genres présumée pour les films PRÉFIXE | **Fondé.** Recalculé sur la constante : 19 positions ≠ 1 (18, 30, 35, 36, 40, 46, 48, 56, 60, 61, 69, 81, 83, 89, 90, 91, 93, 97, 107), 104 genres à 1 | Présomption écrite dans l'en-tête de `vue_a_versions.go` et rappelée dans `localisateur.go`. « Ne pas publier comme lus les genres au-delà de la tête » : **lu** qu'aucun canal de production ne lit un genre au-delà de la tête (`teteDe` prend `Genres[0]`, `listeAnnoncee` le compte, seuls consommateurs de `p.VueA.Genres` hors rangement) ; marquer la suite « présumée » dans `lecture.VueA` toucherait la structure de la RI : **soumis au pilote** (§15.6) |
| 5 | mineur | `MapQuantEntry.Region` devient une condition de lecture de la vue A, en partie choisie à la mesure | **Fondé.** `cmd/mapquant-build` : `Region` n'est posée que par `entreeRegionExterne` (`regionExterneDeclarations` = {Live Fire : 1}) ; les 78 autres entrées sont la région 0 du bloc structure-BSP du tag de niveau (`himap.BSPQuantification`) | Provenance écrite à `tablesDeLaRegionJouee` ; **règle générale** : une région jouée autre que la 0 lue dans le tag rend tout index illisible (lecture arrêtée sur la porte, table DÉFAUT lisible). Tests : `carteDeTest` passe à la région 0, cas « index de la région déclarée » ; mutation RG ROUGE. Effet : `0797ce72`, `60ae07c4` (Live Fire) — §15.3 |
| 6 | majeur | La fusion casse les tests par un fichier non en conflit ; le test (d) de la marche des morts disparaît | **Fondé** (reproduit : `go vet` rouge sur `debut_par_vue_a_test.go:258`) | Fusion faite (§15.1) ; `TestLeCanalDesMortsRecoitLesRecordsPartisDeLaFinDeLaVueA` (`canal_des_morts_test.go`) : sur la bobine 000d5950, 353 paquets partis de E dont 138 où le localisateur rendrait un autre début ; la récolte du canal (morts, occupation, records par archétype, paquets localisés) est celle des records de la cuisson. Mutation MG (le canal relocalise un paquet parti de E) ROUGE. Sondes portées ou retirées (§15.1). (d) refait sur le canal : §15.3 |
| 7 | majeur | ADR 0037 IR-6 contredit 2.7.a après fusion | **Fondé** (texte fusionné relu, l. 230-232 contre l. 110 et 279-280) | Paragraphe d'IR-6 réécrit (EN) : le canal des morts reçoit les records de la cuisson, ne demande au localisateur que les listes non localisées ; seul `killsource` appelle `DebutDeLaVueB` ; phrase de la calibration supprimée ; la garde 0x29 y est dite (soumise). Ligne `marchDebut` du tableau de `localisateur.go` remplacée |
| 8 | majeur | `grammar-2026-10-06.2` portée par deux branches avec deux empreintes | **Fondé** (goldens relus ; à la fusion, `feat/v75` porte en plus `.3`, lot LR) | Rangs `.2`, `.3`, `.4` du lot réunis en **`grammar-2026-10-06.4`**, premier rang libre après le `.3` de `feat/v75` (grep sur toutes les branches locales et distantes et les 22 worktrees : `.4` n'est porté que par ce lot). Chronique réécrite (une entrée pour le lot), COMPLEMENTs de killsource et d'objectives réunis, goldens régénérés |
| 9 | majeur | Aucun test ne garde la vue A de killsource sous la carte (X1 VERTE) | **Fondé** | `TestKillsourceLitLaVueASousLaCarteDuMatch` (`debut_par_vue_a_test.go`) : `fb1a1a72` (ÉGALE), impact (genre 6) à l'index de la région jouée ; sous la carte, `DebutDeLaVueB` rend (E, faux) ; sans carte, le localisateur. X1 ROUGE |
| 10 | majeur | Chronique de killsource : « chaque ligne déjà écrite est celle que ce code écrirait », mesuré sur 20 films seulement | **Fondé** | Option (1) : entrée reformulée (« mesuré sur 20 films, estimé sur le parc »), D23 **soumise au pilote** (§15.6). Mesure refaite après fusion : §15.3 |
| 11 | mineur | Le lecteur Bond rend une valeur par défaut comme lue | **Fondé** (vecteurs de la revue reproduits : rendus `Lue: true` avant correction) | `structure` reçoit l'ensemble des champs du chemin : un champ du chemin d'un autre type arrête la lecture ; booléen et entier 32 hors chemin sont enjambés par leur forme (lue chez leurs écrivains) ; un champ d'un type non enjambable arrête la lecture tant qu'un champ du chemin reste à lire. Les deux vecteurs ajoutés à `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu`. Mutations B1, B2 ROUGES. Parc : §15.3 |
| 12 | mineur | Le second retour de `DebutDeLaVueB` (repli à largeur libre) non gardé (X5 VERTE) | **Fondé** | `debutDuPaquet` vérifie : faux quand E décide, celui du localisateur sinon. X5 ROUGE (5 tests) |
| 13 | mineur | `FilmIdentity.Variante` en section 2 des faits sans montée de `SchemaDesFaits` | **Fondé** (`filmfacts_fichier.go` : `SchemaDesFaits = 4`, doctrine l. 130 ; aucun fait périmé servi : `GrammarRev` et `ProfileRev` montent, l'en-tête refuse tout fichier antérieur) | Instruit ici (même écart que V1 §7, correction 3) ; **décision au pilote** pour les deux étapes : monter `SchemaDesFaits`, ou amender la doctrine (§15.6) |
| 14 | mineur | `profile-2026-10-07`, date à venir | **Fondé** | À la fusion : **`profile-2026-10-06.3`**, rang suivant sans trou après le `.2` de `feat/v75` (libre sur toutes les branches) ; golden régénéré |

### 15.3 Mesures après fusion et corrections (base `fed1efed2`, tête = arbre du lot)

Binaires des deux arbres (`cg3-revue-corr/build.sh`, base depuis `git archive fed1efed2`), table ECS
identique, une commande go à la fois, films lus en place.

**Carte v2 (gate 2 officiel, `-mpp-declare`, `gate2.awk`)** : 399 135 → **452 779 sains (+53 644)**,
utiles sains 4 822 178 → **5 708 564 (+886 386)**. **Aucun film en baisse**, ni en sains ni en utiles
sains (`gate2.tsv`). 348 sains perdus en brut, tous « devenus non fermés », tous sur des films en hausse
(`4f77afc1` 99, `396cfc92` 94, `d9781168` 65, `f75e7053` 59, 2 à 7 sur six autres). Leur juge dans la
tête : composant non porté où la marche depuis E bute (`ti=43` 148, `ti=12` 68, `ti=10` 23, `ti=45` 20,
`ti=35` 2), vue C 56, sortie de la vue B 31. C'est la famille déjà instruite en V2 (LOT_VA_V2 §5.1,
décision (1)). `mpp_declare.tsv` et `fermeture_chunk3.tsv` identiques. La carte ne pose pas la carte du
match (D-VAV3-1) : les corrections 5 (région) et 11 (Bond) n'y ont pas d'effet.

**`killsource` (gate 3)** : `cmd/killsource json`, 20 films (19 témoins et `1c4c63c2`), rc=0 partout :
19 identiques à l'octet ; `c75f33b8` ne diffère que par
`concordance.enregistrements_lus_par_les_deux_voies` 5 → 6 (`Stats.Redundant`, non persisté). Aucune
mort, valeur ni voie ne change. Constat 10 : §15.6.

**`replay-equiv`** (racine factice copiée, 20 films) : `objectives` **identique sur les 20 films**.
Divergent : `artifact` (20, révisions) ; `killsource`, `movementStates`, `movementStates.stats`,
`continuousFire.stats` (16) ; `continuousFire` (13) ; **`vehicleDeaths` et `vehicleDeaths.stats` sur 6
films** (`084a804d`, `111fa685`, `11de8353`, `1c4c63c2`, `53ce4390`, `e5adf7b2`). La base de `a521164d`
a franchi le plafond mémoire au premier passage (3,81 Gio, sous la charge des mutations) ; rejouée seule,
seule `artifact` y diffère. L'étape `killsource` de `replay-equiv` porte les compteurs non persistés de
la marche (`PacketsLocated`, `LocalisationsALargeurLibre`), comme en V3.

**Vérification (d), sur le canal des morts.** Étapes `vehicleDeaths` de la cuisson (`TestVAV2Etapes`,
base contre tête) ; origine des trames par la sonde `va_revue_canal_research_test.go` (base et tête, même
profil de cuisson, `killsource/va_v2_profil_research_test.go`) :

| Film | Classe | Morts base → tête (ajoutées / retirées) | Occupations base → tête (ajoutées / retirées) | Paquets localisés |
|---|---|---|---|---|
| `084a804d` | PRÉFIXE | 48 → 48 | 186 → 278 (92 / 0) | 16 961 → 17 059 |
| `111fa685` | PRÉFIXE | 5 → 5 | 12 → 14 (3 / 1) | 7 758 → 7 866 |
| `11de8353` | PRÉFIXE | 17 → 18 (1 / 0) | 33 → 53 (22 / 2) | 7 014 → 7 232 |
| `1c4c63c2` | PRÉFIXE | 13 → 14 (1 / 0) | 57 → 148 (95 / 4) | 13 546 → 16 616 |
| `53ce4390` | ÉGALE | 3 → 3 | 30 → 60 (30 / 0) | 7 385 → 7 457 |
| `e5adf7b2` | PRÉFIXE | 17 → 19 (2 / 0) | 67 → 95 (28 / 0) | 7 578 → 7 697 |
| `d9781168` | ÉGALE | identiques | identiques | identiques |

Origine des 281 lectures changées, par la trame qui les porte (instant du paquet) :

- **97 viennent d'une trame partie de E dans la tête** (dans la base : liste récupérée par le canal 79,
  fermeture de la cuisson 18). Sur les films PRÉFIXE, E n'y est pris que prouvé par la fermeture
  (décision (2)) ; sur `53ce4390` (ÉGALE), toujours (décision (1)).
- **184 viennent d'une trame dont le début est LE MÊME dans la base et dans la tête** (liste récupérée
  par le canal 146, fermeture de la cuisson 38) : la lecture change parce que le monde change en amont,
  les trames parties de E liant et déliant d'autres entités.
- Aucune ne vient d'une trame dont seul le localisateur a changé le début.

Morts ajoutées : `11de8353` 1 (trame partie de E), `e5adf7b2` 2 (une trame partie de E, une liste
récupérée au même début), `1c4c63c2` 1 : origine NON ÉTABLIE — la sonde ne localise sa trame ni dans la
base ni dans la tête, alors que la cuisson y lit la mort ; la sonde rejoue la marche sous le profil de
la cuisson, pas la cuisson entière, et ce cas le montre (la classification ci-dessus est celle de la
sonde). C'est ce que l'étape V2 change explicitement : la cuisson part de E, et le canal des
morts reçoit ses records. Les corrections de la revue n'y sont pour rien.

Le compte `repli_localisation_largeur_libre` (banc du gate de corpus, R-1, base → tête) **baisse sur les
15 films où il est publié**. ÉGALE : `bcb6d393` 189 → 3, `fb1a1a72` 495 → 7, `d9781168` 838 → 14,
`c75f33b8` 323 → 4, `bf15f7ab` 270 → 5, `51ebbc0f` 591 → 4, `0797ce72` 494 → 148, `bfecd02b` 905 → 1,
`4f77afc1` 4 847 → 25, `396cfc92` 713 → 3, `f75e7053` 432 → 1. PRÉFIXE : `084a804d` 4 833 → 3 729,
`111fa685` 2 359 → 1 805, `e5adf7b2` 2 219 → 1 610, `11de8353` 1 866 → 1 444.

**Gate de corpus (gate 6)** : `replay-corpus-gate --reference=base --base=fed1efed2`, parc copié au
scratchpad : **rc=1**. Banc de vérité : 18 / 19 « ok », 0 `MANQUE`, **1 `FAUX`** : `084a804d` V-6
« trajet loin du véhicule » 1 → 2 (`vehicule 913 passager 571 @9187`). Le slot 571 (génération 2) gagne
dans la tête un trajet dans le véhicule 921, de 13 180,92 s à 13 208,88 s, puis une montée au siège 2 à
13 212,75 s ; la base ne lit sa montée qu'à 13 215,99 s. La lecture de 13 180,92 s vient d'une liste
récupérée par le canal au MÊME début qu'en base (monde changé en amont). Celle de 13 212,75 s vient d'une
trame partie de E (PRÉFIXE, prouvée). Vraie ou fausse lecture : **non tranché** (le banc compare des
positions que d'autres canaux publient ; aucune vérité indépendante du trajet). 701 lignes `[FILET]` :
postures 611, rafales 25, couverture 59, véhicules 6 (les trajets de `4f77afc1`). C'est la même famille
qu'en V2 contre `87cdfa761`, où le banc rendait 3 `FAUX` de V-3, disparus ici. Admission : §15.6.

**Variante de partie (correction 11), parc** : `TestVAV3Corps` sur tout le cache (1 687 répertoires, dont
1 657 lus par V3). **Les 1 657 lignes communes sont identiques à l'octet** à
`cg3-V3/v3_prod_parc_final.tsv` (30 films sont neufs depuis). Le refus élargi ne retire aucune variante
lue du parc.

**Performance (gate 4, mesurée, non attribuée)** : carte v2, somme des durées base 149 994 ms (jouée
pendant les mutations), tête 128 765 ms ; pic max 321 → 315 Mio.

### 15.4 Révisions

- `grammar.Rev` : `feat/v75` porte `grammar-2026-10-06.2` (fusion de 2.7.a) et `.3` (LR) ; le lot passe
  à **`grammar-2026-10-06.4`**, une entrée de chronique qui réunit V2, V3 et la revue. Grep sur les
  branches locales et distantes et sur les 22 worktrees : `.4` n'est porté que par ce lot.
- `profile.Rev` : **`profile-2026-10-06.3`** (au lieu de `profile-2026-10-07`).
- `killsource.Rev`, `objectives.Rev` : constantes ; les COMPLEMENTs V2 et V3 sont réunis en un seul ;
  celui de killsource dit « mesuré sur 20 films, estimé sur le parc » (constat 10).
- `source.Rev`, `replay.SchemaVersion` (79), `SchemaDesFaits` (4) : inchangés (constat 13 : §15.6).
- `grammar/rev_chronique.go` passait 500 lignes : rangs `grammar-2026-09-27` à `.3` versés, mot pour
  mot, dans `rev_chronique_archive_7.go` (rotation ordinaire).

### 15.5 Gates (sorties exactes ; pièces `cg3-revue-corr/`)

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0 ; `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/replaybuild/...` : rc=0.
3. `go test ./internal/archlint/ -count=1` : `ok … 44.579s`.
4. G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 40m`) : rc=0, 21
   paquets `ok` (`grammar` 112,957 s, `replay` 70,328 s). Tests de révision et de chronique rejoués après
   les dernières retouches : `ok`.
5. `golangci-lint run ./internal/games/halo_infinite/film/...` : `0 issues.` ; recette CI
   `--new-from-merge-base=origin/main ./...` : `0 issues.` ; `--build-tags=research` : 242 issues contre
   197 sur l'archive de base. La différence est faite de 45 `errcheck` de `research/mouvement/rapport.go`
   et `research/reapparition/rapport.go`, fichiers identiques dans les deux arbres, que la passe sur
   l'archive ne rapporte pas ; aucune issue dans un fichier du lot.
6. Carte v2, `killsource`, `replay-equiv`, gate de corpus, (d) : §15.3.
7. Mutations (`-overlay`, suite entière du paquet `grammar`, `mutations.txt`) : **6 / 6 ROUGES**.

| Mutation | Tests rouges |
|---|---|
| X1 `VueADuFilmSousCarte` sans carte | `TestKillsourceLitLaVueASousLaCarteDuMatch` |
| X5 `DebutDeLaVueB` rend `(E, vrai)` | `TestKillsourceLit…`, `TestLaFinDeLaVueADUnFilmRecent…`, `TestLaFinDeLaVueAPrime…`, `TestUnFilmAncienNePrendPas…`, `TestUnFilmAncienPrend…` |
| MG le canal relocalise une trame partie de E | `TestLeCanalDesMortsRecoitLesRecordsPartisDeLaFinDeLaVueA` |
| B1 `m_gameEngineType` d'un autre type enjambé et accepté | `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu` |
| B2 saut à la fin de structure malgré un champ du chemin à lire | `TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu` |
| RG index d'une région déclarée lu | `TestLesPositionsAIndexSeLisentSurLaRegionJouee` |

### 15.6 Décisions du pilote (2026-10-06) et leur application

Les décisions laissées ouvertes par la revue (constats 1, 4, 10, 13 et le gate de corpus) ont été
prises par le pilote le 2026-10-06. Application : §16 (mesures, gates, pièces `scratchpad/cg3-final/`).

| # | Décision du pilote | Application (fichier:ligne, tête du commit) | Preuve |
|---|---|---|---|
| 1 | Constat 1 : un film dont la version majeure de `chunk_00` n'est pas 0x29 suit la règle PRÉFIXE même à table égale (option (a)) ; seuls table ÉGALE **et** majeure 0x29 prennent E toujours | `grammar/vue_a_versions.go:41` (en-tête « la garde de version majeure », fondement `FUN_1428e219c`), `:88` `versionMajeureJouee = 0x29`, `:139` `tableDesGenresDuFilm` (seul site où la classe d'un film est décidée), `:152` `classeSousLaMajeure` ; `grammar/localisateur.go:40` ; ADR 0037 IR-6 (l. 235-239) ; chronique `grammar-2026-10-06.4` (`rev_chronique.go:349`). Relevé versé : `va_ghidra/FUN_1428e219c.c`, `va_ghidra/sonde_majeures_parc_2026-10-06.txt` | Lu dans le jeu : `if (*param_2 == 0x29) { … }` sinon `FUN_142988e98`. Tests : `vue_a_majeure_test.go:41` (vecteurs : table égale sous 0x28 → PRÉFIXE, sous 0x29 → ÉGALE, majeure non lue → PRÉFIXE ; bobines `bcb6d393` → PRÉFIXE, `fb1a1a72` → ÉGALE), `:80` (paquet où la marche depuis E bute : majeure 0x29 → E, majeure 0x28 → le localisateur à l'identique), `debut_par_vue_a_test.go:207` (vrai paquet 1:204 de `bcb6d393` : ÉGALE partirait de 5605, la classe du film exige la preuve). Mutations M1 (garde retirée) et M2 (majeure non lue acceptée) ROUGES (§16.6) |
| 2 | Constat 10, D23 : `killsource.Rev` constante, « mesuré sur 20 films (19 témoins + `1c4c63c2`), estimé sur le parc » | `facts/killsource/rev_chronique.go:465` : COMPLÉMENT réécrit — « DÉCISION D23 DU PILOTE DU 2026-10-06, MESURÉE SUR 20 FILMS ET ESTIMÉE SUR LE PARC » ; classes du parc corrigées après la décision 1 (1 401 HI_1_13_0 ÉGALE, 147 HI_1_12_0 désormais PRÉFIXE, sur 1 657) | Formulation vérifiée exacte : 19 témoins comptés dans `config/replay_corpus.toml` (`[[temoin]]`) + `1c4c63c2` ; mesure refaite contre `b033d30f0` après les décisions : 19 JSON identiques à l'octet, `c75f33b8` seul `enregistrements_lus_par_les_deux_voies` 5 → 6 (non persisté) ; les quatre goldens de `TestGoldenFilms` ne changent que par des compteurs de diagnostic (§16.3) |
| 3 | Constat 13 : la doctrine de `SchemaDesFaits` est tenue, il monte d'un cran pour la section 2 | `replay/filmfacts_fichier.go:170` (entrée SCHEMA 5 : V1 `SimulationDeLEnregistreur`, `OptionsDePartieLues`, V3 `Variante` ; justification), `:180` `SchemaDesFaits = 5` ; graines `replay/testdata/fuzz/FuzzDecodeFilmFactsFile/seed_00..03` régénérées (`-update-graines-faits`) | `TestFaitsDUnSchemaAnterieurSontRefusesSurLEnTete` et la suite `replay` vertes (G-film) ; les graines portent l'octet de schéma 5 |
| 4 | Constat 4 : marquer « présumée » la suite des genres d'un film PRÉFIXE, si c'est un petit ajout à `lecture.VueA` | Fait : un champ `PremierPresume uint16` (`lecture/paquet.go:201-206`, logé dans le rembourrage : taille de `VueA` inchangée), calculé par `premierGenrePresume` (`vue_a_versions.go:164`) à la fin de `lireLaVueA` (`vue_a_lecture.go:121`), rangé par `rangerLaVueA` (`distribuer_tetes.go:38`). Règle : sur un film PRÉFIXE, le rang du premier genre au-delà du 107 (`dernierGenreAVersionDistinctive`, dernier genre dont la version native diffère de 1) ; `len(Genres)` sinon. Aucun bit lu ne change | `vue_a_majeure_test.go:111` (zoom puis genre 110 : rang 1 sous PRÉFIXE, 2 sous ÉGALE ; la constante 107 recalculée sur la table native). Mutations M3, M4 ROUGES. Deux fichiers de la RI touchés (`lecture/paquet.go`, `distribuer_tetes.go`) : à relire par levelup-57 |
| 5 | Instruire le `FAUX` V-6 de `084a804d` | Rien corrigé (consigne) | Verdict : **les deux lectures sont vraies, le `FAUX` est une fausse alarme du banc** (§16.5) |

### 15.7 Fichiers

Production : `grammar/film_variante_de_partie.go` (constat 11), `grammar/transloc_events.go` (5),
`grammar/localisateur.go` (fusion, 1, 4, 7), `grammar/vue_a_versions.go` (1, 4),
`grammar/vue_a_charges_execution.go` (2, 3). Tests : `canal_des_morts_test.go` (6),
`debut_par_vue_a_test.go` (6, 9, 12), `film_variante_de_partie_test.go` (11), `vue_a_execution_test.go`
et `vue_a_variante_test.go` (5). Sondes : `va_v2_research_test.go` (porté), `va_v2_corr_research_test.go`
(retirée), `va_revue_canal_research_test.go` (neuve). ADR 0037 (7). Révisions et goldens (8, 10, 14).

## 16. Décisions du pilote : mesures et gates (2026-10-06, base `b033d30f0`)

`git fetch` du 2026-10-06 : `origin/feat/v75` = `b033d30f0` (lot « séries temporelles, onglet Usages =
Emprise » ; un seul fichier du film, `replay/usage_summary_families.go`), au-delà de `f8a14b3b9` :
**fusionné d'abord** (`a01e95cdc`, sans conflit). Mesures contre `b033d30f0` (`git archive`), binaires
des deux arbres, une commande go à la fois, films lus en place. Pièces : `scratchpad/cg3-final/`.

### 16.1 Révisions

- `grammar.Rev` : **`grammar-2026-10-06.4`, gardé** — rang porté par ce seul lot (grep sur toutes les
  branches locales et distantes et sur les 23 worktrees), jamais fusionné ; l'empreinte change
  (décisions 1 et 4), golden régénéré par `-update-grammar-rev`, entrée de chronique complétée.
- `profile.Rev` (`profile-2026-10-06.3`) : inchangé. La version majeure est lue par le profil existant
  (`profile.HighlightProfile`, même lecture `chunk_00 + 0`) : aucune source de `profile` ne change.
- `killsource.Rev`, `objectives.Rev` : constantes (décision 2) ; complément de killsource réécrit.
- `SchemaDesFaits` : 4 → **5** (décision 3). `replay.SchemaVersion` (79), `source.Rev` : inchangés.

### 16.2 Carte v2 (gate 2 officiel, `-mpp-declare`, `integ2/gate2.awk`) : aucun film en baisse

| Film | Sains base → tête | Net | Utiles sains base → tête | Net utiles | Sains perdus (non fermés) |
|---|---|---|---|---|---|
| `bcb6d393` | 5 880 → 6 479 | +599 | 35 502 → 40 624 | +5 122 | 0 |
| `fb1a1a72` | 43 456 → 46 308 | +2 852 | 325 815 → 351 322 | +25 507 | 4 |
| `d9781168` | 26 311 → 35 773 | +9 462 | 180 844 → 263 865 | +83 021 | 65 |
| `c75f33b8` | 23 867 → 27 272 | +3 405 | 153 973 → 180 599 | +26 626 | 3 |
| `bf15f7ab` | 28 602 → 29 913 | +1 311 | 216 098 → 228 132 | +12 034 | 2 |
| `51ebbc0f` | 20 442 → 26 877 | +6 435 | 140 632 → 197 932 | +57 300 | 7 |
| `084a804d` | 23 784 → 26 014 | +2 230 | 638 098 → 705 382 | +67 284 | 0 |
| `0797ce72` | 19 155 → 20 772 | +1 617 | 159 929 → 176 835 | +16 906 | 4 |
| `111fa685` | 11 955 → 13 263 | +1 308 | 244 820 → 278 281 | +33 461 | 0 |
| `e5adf7b2` | 12 264 → 13 153 | +889 | 302 195 → 326 339 | +24 144 | 0 |
| `60ae07c4` | 34 574 → 34 574 | 0 | 253 813 → 253 813 | 0 | 0 |
| `a349fea8` | 427 → 427 | 0 | 3 654 → 3 654 | 0 | 0 |
| `a521164d` | 693 → 693 | 0 | 78 → 78 | 0 | 0 |
| `11de8353` | 13 831 → 14 893 | +1 062 | 268 308 → 294 393 | +26 085 | 0 |
| `50247b26` | 139 → 139 | 0 | 274 → 274 | 0 | 0 |
| `bfecd02b` | 27 667 → 31 067 | +3 400 | 239 062 → 276 442 | +37 380 | 7 |
| `4f77afc1` | 24 607 → 31 355 | +6 748 | 623 437 → 832 298 | +208 861 | 99 |
| `396cfc92` | 23 130 → 25 954 | +2 824 | 169 502 → 196 024 | +26 522 | 94 |
| `f75e7053` | 23 673 → 24 517 | +844 | 162 137 → 168 248 | +6 111 | 59 |
| `1c4c63c2` | 34 678 → 42 696 | +8 018 | 704 007 → 928 344 | +224 337 | 0 |
| **corpus** | 399 135 → **452 139** | **+53 004** | 4 822 178 → **5 702 879** | **+880 701** | 344 |

Base identique à celle de `fed1efed2` (399 135 / 4 822 178) : `feat/v75` n'a rien changé au décodage.
Effet de la décision 1, mesuré : seul `bcb6d393` (HI_1_12_0) des 20 films a une table égale sous une
autre majeure ; son gain passe de +1 239 (classe ÉGALE, §15.3) à **+599** (classe PRÉFIXE, E seulement
prouvé), et ses 4 sains perdus disparaissent ; les 19 autres films sont identiques au §15.3. Les 344
sains perdus restants sont tous « devenus non fermés », sur des films en hausse (même famille qu'au
§15.3). `mpp_declare.tsv` et `fermeture_chunk3.tsv` identiques.

Performance (gate 4, mesurée, non attribuée) : somme des durées 84 239 → 114 605 ms, pic max 327 → 493
Mio (`a349fea8`, film sans section d'identification, dont la vue A ne se lit pas). Rejouée deux fois
sur trois films : 1re passe 221 → 320 Mio, 2e passe 222 → 223 Mio sur `a349fea8` — bruit d'exécution.

### 16.3 `killsource` (gate 3) et `TestGoldenFilms`

`cmd/killsource json`, 20 films, binaire de `b033d30f0` contre binaire de la tête : rc=0 partout,
**19 identiques à l'octet** (`bcb6d393` compris), `c75f33b8` ne diffère que par
`enregistrements_lus_par_les_deux_voies` 5 → 6. `TestGoldenFilms`
(`KILLSOURCE_FIXTURES=…/film_chunks`) : vert sur la base, rouge sur la tête avant régénération — les
goldens des quatre films de référence n'avaient pas été rejoués depuis V2 (le test exige les films).
Diff complet après `-update` : `000d5950` paquets à événements localisés 3 423 → 3 508 / 3 508,
`78919882` 4 816 → 5 025 / 5 027, `9b191a7f` 5 525 → 5 713 / 5 714, `fccc61cd` 4 686 → 4 839 / 4 839 et
lus par les deux voies 100 → 101 (accord 91, désaccord 0). Aucune ligne de mort, de valeur ni de voie ;
`cumul.golden` inchangé. Suite `killsource` avec les films : `ok` (103 s).

### 16.4 Gate de corpus (gate 6)

`replay-corpus-gate --reference=base --base=b033d30f0 --keep-work` : **rc=1**, banc de vérité **18 / 19
« ok », 0 `MANQUE`, 1 `FAUX`** : `084a804d` V-6 1 → 2 (`vehicule 913 passager 571 @9187`), le même qu'au
§15.3. 688 lignes `[FILET]` (postures 604, rafales 25, couverture 53, véhicules 6). `bcb6d393` : ok,
P-1 5 880 → 6 483. Télémétrie : `grammarRev` `.3` → `.4`, `profileRev` `.2` → `.3`.

### 16.5 Le `FAUX` V-6 de `084a804d` : instruction et verdict

Pièces : documents cuits par le gate (cuisson de production, `replaybuild`, profil de balayage
calibré par killsource), `gate_work/cuisson-base/…/084a804d.json` (base) et
`gate_work/data/cache/replays/halo_infinite/084a804d.json` (tête) ; extraction `jq`, distances `awk`
(`cg3-final/v6/dist.tsv`). Image = 100 ms (`frameIntervalMs`) ; l'image 8868 est l'instant
13 180,918 682 s de la lecture d'occupation.

Ce qui diffère (mesuré) : les échantillons du véhicule 913 et les pistes des slots 570 et 571 sont
**identiques** dans les deux documents ; seuls les trajets changent. Le véhicule 913 est une
**Banshee** (`family`, `chassis c6e79dcc`, apparue à (32,57 ; −11,76 ; 93,6), détruite à 9325). Les
lectures d'occupation du lot (parent 921, §15.3) tombent à l'image près sur les bornes des trajets ajoutés
(13 180,92 s → 8868, 13 208,88 s → 9148, 13 212,75 s → 9187).

| Trajet du véhicule 913 | Base | Tête |
|---|---|---|
| slot 571, siège 0, 8868 → 9148 | absent | **ajouté** (attache lue à 13 180,92 s, liste récupérée au même début qu'en base) |
| slot 570, siège 1, 9122 → 9154 ; siège 0, 9154 → 9213 | présent | présent |
| slot 571, siège 2, 9187 → 9219 | absent | **ajouté** (attache lue à 13 212,75 s, trame partie de E) |
| slot 571, siège 0, 9219 → 9270 | présent | présent |

Oracles (mesurés sur les documents) :

1. **Le trajet 8868 → 9148 est vrai.** À 8868, la Banshee est immobile sur son point d'apparition
   (déplacement < 0,03 m par image jusqu'à 8876) et le slot 571 est à **2,48 m** d'elle ; sa piste
   s'arrête après 8869 et reprend à 9147 à **1,90 m** de la Banshee, l'image de sa descente lue
   (13 208,88 s). Entre les deux, la Banshee décolle (z 93,5 → 106,8) et vole à ≈ 12,7 m/s pendant
   25 s. Dans la base, **personne** ne la pilote de 8877 à 9122 : un aéronef qui vole sans pilote. La
   piste d'un occupant assis s'interrompt toujours ainsi dans ces documents : celle du slot 570 se tait
   pendant ses trajets de 9122 à 9213 et reprend à 9212 à 2,80 m.
2. **Le trajet au siège 2, 9187 → 9219, est vrai : c'est un abordage.** La Banshee a un seul siège de
   pilote ; les sièges 1 et 2 sont des places d'abordage. Le slot 570 l'aborde par le siège 1 à 9122
   (à 2,62 m), y reste **32 images**, prend le siège 0 à 9154 ; le pilote 571 en descend à 9148
   (éjecté : sa piste reprend à 1,90 m et s'éloigne jusqu'à 10,25 m à 9164). Le slot 571 revient : 2,04 m
   à 9181, puis l'attache au siège 2 à 9187, y reste **32 images** — la même durée que l'abordage du
   570 —, et prend le siège 0 à 9219 ; le 570 en descend à 9213 (sa piste reprend à 9212 à 2,80 m puis
   s'éloigne). Sa piste s'arrête après 9187, comme celle de tout occupant assis.
3. **Pourquoi le banc le juge loin.** V-6 compare la position du passager au premier échantillon du
   trajet : à 9187, le point de la piste du 571 est à **5,50 m** de la Banshee, qui file à **11,2 m/s**
   (1,12 m entre 9186 et 9187) ; les distances avant l'attache sont 2,04 / 2,37 / 2,36 / 3,05 / 3,43 /
   4,39 / 5,50 m de 9181 à 9187 : le joueur court après l'aéronef qui s'éloigne, et l'abordage le
   rattrape. Le seuil `rayonTrajetM` (3 m, « rayon d'ancrage d'un trajet mesuré par le producteur »)
   n'a pas été mesuré sur un abordage d'aéronef en vol. Que 5,50 m soit à portée d'abordage d'une
   Banshee est **estimé** (la portée n'est pas lue dans le jeu) ; la vérité du trajet, elle, tient aux
   oracles 1 et 2, indépendants de ce seuil.

**Verdict : les deux lectures sont vraies ; le `FAUX` V-6 est une fausse alarme du banc** (seuil de
3 m appliqué au premier échantillon d'un abordage d'aéronef en mouvement). La base, elle, publiait un
vol de 25 s sans pilote et un abordage manquant. Rien n'est corrigé (consigne) ; découverte
D-VAV3-6 ci-dessous.

### 16.6 Gates (sorties exactes ; pièces `cg3-final/`)

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0 ; `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/replaybuild/...` : rc=0.
3. `go test ./internal/archlint/ -count=1` : `ok … 41.832s`.
4. G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) : rc=0,
   21 paquets `ok` (`grammar` 60,334 s, `replay` 32,772 s) ; puis `killsource` avec les films (après
   régénération des goldens de `TestGoldenFilms`) : `ok` (103 s).
5. `golangci-lint run ./internal/games/halo_infinite/film/...` : `0 issues.` ; recette CI
   `--new-from-merge-base=origin/main ./...` : `0 issues.`
6. Carte v2 : §16.2. `killsource` : §16.3. Gate de corpus : §16.4.
7. Mutations (`-overlay`, suite entière du paquet `grammar`, `mutations.txt`) : **4 / 4 ROUGES**.

| Mutation | Tests rouges |
|---|---|
| M1 garde de majeure retirée de `classeSousLaMajeure` | `TestLaClasseDUnFilmSuitLaVersionMajeureQueLeJeuJoue`, `TestUnFilmDUneAutreMajeureEstProuvePaquetParPaquet`, `TestUnVraiPaquetNePartDeLaFinDeSaVueAQueProuvee` |
| M2 majeure non lue acceptée | `TestLaClasseDUnFilmSuitLaVersionMajeureQueLeJeuJoue` |
| M3 `premierGenrePresume` ne présume rien | `TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee` |
| M4 rang non rangé dans `lecture.VueA` | `TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee` |

### 16.7 Découvertes et écarts

- **D-VAV3-6** — V-6 (`replayverite/seuils.go`, `rayonTrajetM` = 3 m) juge loin le premier échantillon
  d'un abordage d'aéronef en vol (`084a804d`, 5,50 m à 11,2 m/s). Une règle générale (portée
  d'abordage lue dans le jeu, ou seuil qui tient compte de la vitesse du véhicule à l'attache) relève
  d'une décision ; non traité.
- **E-5 — `TestGoldenFilms` de killsource non rejoué par V2 et V3** : rouge sur la tête avant ce lot
  (compteurs de diagnostic seulement, §16.3), régénéré ici. Le test exige les films (`KILLSOURCE_FIXTURES`)
  et la CI ne le joue pas.
- `TestUnVraiPaquetPartDeLaFinDeSaVueA` renommé `TestUnVraiPaquetNePartDeLaFinDeSaVueAQueProuvee` : sa
  prémisse (`bcb6d393` ÉGALE) est fausse après la décision 1.
