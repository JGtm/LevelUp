# LOT VA, ÉTAPE V1 — La vue A lue par une seule lecture, sortie inchangée (2026-10-05)

> Campagne de grammaire, lot VA (« Lire la vue A jusqu'au bout », décision de l'utilisateur du
> 2026-10-04 ; règle des deux classes, « Oui, lance », même jour). Plan de conception :
> `scratchpad/va-conception/PLAN_LOT_VUE_A.md` (non versionné) et sa vérification adverse. Sources :
> `LOT_LN.md` (`feat/cg2-ln`, `d9c268c9f`), recherches R1 (`va-r1/R1_PERTES_LN.md`), R2
> (`va-r2/R2_GENRES.md`, ports `va-r2/ports/`), R3 (`va-r3/R3_FILET_LN.md`).
>
> Worktree `LevelUp-wt-cg3-vuea`, branche `feat/cg3-vue-a`, base `87cdfa761` (= `origin/feat/v75`,
> contrôlé par `git fetch` au début de la reprise et avant les gates : pas d'avance). Référence de
> mesure : binaires construits depuis `git archive 87cdfa761`.
>
> Conventions : **lu** = lu dans le jeu (Ghidra, `HaloInfinite.exe` HI_1_13_0, base 0x140000000,
> lecture seule ; décompilés sous `va_ghidra/`) ; **mesuré** = compté par un outil sur les 20 films ;
> **estimé** = dérivé d'une mesure par une hypothèse écrite. « Sain » = paquet fermé que le juge de L0
> ne contredit pas (colonne `ferme` de la carte v2).

## 0. Statut

**[x] retenue.** V1 porte la lecture de la vue A de LN et les quatre ports de R2, écrit la règle des
versions en deux classes, et fait de `lireLaVueA` la SEULE lecture de la vue A de la structure. La fin
de la vue A (E) est calculée et rangée dans `lecture.Paquet.VueA` ; elle ne décide d'aucune
localisation. Contrôle d'identité tenu :

- carte v2 (`fermeture_paquets.tsv`, 20 films) **identique à l'octet** à `87cdfa761` (sha256
  `20ede385…a95b28c3` des deux côtés ; 313 542 paquets sains, 2 923 597 records utiles sains) ;
- `cmd/killsource json` **identique à l'octet** sur 20 films (19 témoins + `1c4c63c2`) ;
- `replay-equiv` (20 films) : une seule étape diverge, `artifact`, par sa chaîne de révision
  (contrôle sur les fixtures de contrat : JSON identique une fois `grammar-…`/`profile-…` neutralisés) ;
- tête donnée aux canaux, route vers la localisation et étendue rangée : identiques paquet par paquet
  sur les 20 films (629 142 trames delta), par la marche de cuisson ET par la passe des têtes ;
- `replay-corpus-gate` (base explicite `87cdfa761`) : rc=0, 19 témoins à 0 gain, 0 perte, 0 changement ; banc de vérité 19 / 19 « ok ».
- mutations : 24 / 24 ROUGES ; plus m25 (s01 du contrôle indépendant, la passe des têtes sans le
  contrôle de corruption du film), VERTE à `e6ec7abd4`, ROUGE après l'ajout de son vecteur (§11).

## 1. Ce qui est lu dans le jeu

| Fait | Lu où | Porté dans |
|---|---|---|
| Le paquet est `cfg:1 · vue A · 0 · vue B · vue C` ; la vue B commence au bit qui suit le terminateur de la vue A | écrivain `FUN_142f2c3b0` (`FUN_142f2c050`, `FUN_1406d49c4`), lecteur `FUN_142987460` | en-tête de `vue_a_lecture.go` |
| Vue A : `{ R(1) ; 0 → fin ; FUN_14080a9d4 }` ; message : genre `R(7)` < 0x7b, trois références gardées, charge par `vtable + 0x68`, contrôle de corruption `R(1) → R(32)` | `FUN_14076a1c4`, `FUN_14080a9d4`, `FUN_14076cea8` | `lireLaVueA`, `lireUnMessage` |
| Bit de configuration `DAT_144706104` : à 0, les références ne prennent pas la plage par catégorie | `FUN_142987460`, `FUN_1406d3140` | `lireLaVueA` (arrêt après la tête) |
| Table des 123 descripteurs, 13 genres vides | `FUN_140e453b4`, `FUN_1408d8220` | `vue_a_genres.go` (report LN) |
| Version de chaque genre : en rejeu celle du film (`film + 0xCB208 + genre*4`), native `DAT_14474cd90` | `FUN_141102ed0`, `FUN_1428e1c64` | `vue_a_versions.go` |
| 45 genres à charge du lot LN (44 avant la relecture RI, §12) | table LN §1.4 | `vue_a_charges*.go` (report LN) |
| 15 Script : `[W(15)]` si `game_simulation != 2` (écrivain), `W(13)`, `W(10) n`, `n` bits ; la simulation de l'enregistreur est le 2e champ `W(3)` des options de partie en tête du corps de `chunk_00` | `FUN_14080bb4c`, `FUN_142eec4d8`, `FUN_1407ec560`, `FUN_140ad4144`, `FUN_142e33478` | `vue_a_charges_execution.go`, `film_identity.go`, `profile.FilmIdentity.SimulationDeLEnregistreur` |
| 39 biped_throw_initiate : `k = R(1)` ; 0 → `R(3)` ; 1 → porte `R(32)`, `R(4)` ; puis `FUN_1407f2058`. La garde d'état (`DAT_144c1cfa8 + 4 == 2`) se ferme par le lecteur lui-même (un refus arrête la vue A) | `FUN_140c6a58c`, `FUN_14104fc8c`, `FUN_140544ec8` | `chargeLancerInitie` |
| 5 et 6 : position à index aux niveaux 0xf et 0xc ; largeurs par la loi `FUN_140be9b88` sur les bornes de la région jouée ; un autre index arrête la lecture | `FUN_140be9a14`, `FUN_140be9b88`, `FUN_14076e524` | `chargeDetonation`, `chargeImpact` sur `tablesDeLaRegionJouee` |

Bilan : 47 genres à charge portée (45 de LN, plus 15 et 39 ; 46, 44 et 59 avant la relecture RI, §12), 13 vides, soit 60 genres lisibles sur
123. Non portés : 85 et 116 (valeur d'exécution non fixée par le film, R2 §4), 12, 106 et les genres
rares (R2 §2.3 : désynchronisations).

### 1.1 La règle des versions, en deux classes (`vue_a_versions.go`)

Deux tables lues se comparent, sans seuil ni build nommé :

- **ÉGALE** : le film déclare les 123 genres, chacun à sa version native. Film récent.
- **PRÉFIXE** : il en déclare moins, chacun à sa version native. Film ancien.
- **ILLISIBLE** : section absente, table vide ou plus longue, une version différente. La vue A n'est pas
  lue au-delà de sa tête.

Les deux premières classes lisent la vue A jusqu'à son terminateur. Ce que la marche fera de E selon
la classe est l'étape V2 (décision du 2026-10-04 : E toujours pour ÉGALE ; prouvé paquet par paquet
pour PRÉFIXE). En V1, rien.

Classes mesurées sur le corpus : ÉGALE 11 films (HI_1_12_0, HI_1_13_0) ; PRÉFIXE 5 films (HI_1_9_0,
HI_1_10_0, HI_1_11_0) ; ILLISIBLE 4 films (HI_1_4_1, HI_1_8_0, version-31, version-33).

## 2. Ce qui change dans le code

- **Report LN** (par `git show d9c268c9f:<chemin>`, jamais de cherry-pick) : `vue_a_genres.go`,
  `vue_a_genres_test.go`, `vue_a_charges.go`, `vue_a_charges_armes.go`, `vue_a_charges_sacs.go`,
  `vue_a_charges_tir.go`, `vue_a_charges_test.go`, `vue_a_lecture.go`, `vue_a_lecture_test.go`,
  `vue_a_versions.go`. Écarts au report : `genresLisiblesDuFilm` devient `classeDesGenres` (deux
  classes) ; `tablesSansIndexDePlage` retiré (5 et 6 lisent la région jouée, R2) ; `parcourirLaVueA`
  et le crochet de `localiserLaListe` ne sont pas reportés.
- **Ports R2** : `vue_a_charges_execution.go` (Script, 39), `lireSimulationDeLEnregistreur`
  (`film_identity.go`), `FilmIdentity.SimulationDeLEnregistreur`/`OptionsDePartieLues`
  (`profile/identite.go`). Les positions des genres 5 et 6 ne passent PAS par le lecteur à part de R2
  (`lirePositionDeNiveau`) mais par le lecteur existant `lireE494Sur`, sur des tables restreintes à
  la région jouée (`tablesDeLaRegionJouee`, champ `regionSeule`) : une seule implantation de
  `FUN_14076e524`. `r2Diag85` n'est pas reporté (ratchet `filmdec_package_vars`).
- **La grammaire de la vue A hors du profil de balayage** : `grammaireDeLaVueA` (classe, cardinal,
  Script, tables de la région jouée) est dérivée du film (`grammaireDeLaVueASousFilm`), mémorisée dans
  `FilmContext.duFilm` avec le contrôle de corruption (`controle_corruption_du_film.go`), et ne
  voyage qu'avec le lecteur de la vue A (`Lecteur.vueA`). `GrammaireBalayage` ne change pas : aucune
  empreinte de forme (`ProfilCalibre`, `DeathStats.Config`) ne bouge (D-VA-3 du plan évité).
- **Une seule lecture** :
  - `consumeVueA` supprimé (`frame_vue_messages.go`), `FluxVueA` complété (`Debut`, `Fin`) et
    déplacé dans `vue_a_lecture.go` ;
  - `rangerLaTete(p, bal, g)` lit la vue A une fois (tête à l'identique, puis la suite si le film la
    rend lisible), la range (`rangerLaVueA`) et la rend ;
  - `marcherLePaquet` passe cette vue A lue à `lireTrameParRangs` par son départ
    (`departDeTrame{bit, vueA}`) ; la marche ne la relit pas et ne la re-range pas
    (`rangerLesVues` : `l.enTete && !l.vueARecue`) ;
  - sans vue A reçue (`decodeFrameParRangs`, essais `lectureDEssai`, cartes, sondes), la marche partie
    de la tête l'obtient de `lireLaVueA` sous `grammaireDeLaVueA{}` : la tête seule, comme
    `consumeVueA` ;
  - la marche partie de la tête ne traverse qu'une vue A VIDE (`!l.vueA.Vide` → arrêt à
    `finDeTete()`), comme avant (`Porte` valait exactement `Vide`) ;
  - `teteDe` : `Genres[0]` dès qu'un genre est rangé ; liste vide pour une vue terminée sans genre ;
    lecture tolérante sinon. `listeAnnoncee` remplace `Etat == VueArretee` ;
  - `distribuerLesTetes` passe `fc.profilDeLaVueA()` (profil sans compter le repli
    `repli_controle_corruption_section_absente` : la passe des têtes n'emploie pas la valeur de
    repli) et `fc.grammaireDeLaVueA()`.
- **Garde-rail** `archlint/film_vue_a_lecteur_unique_test.go` : `lireLaVueA` n'est appelée que par
  `rangerLaTete` et `lireTrameParRangs`, `chargeDuGenre` que par `lireUnMessage` ; toute fonction
  nommée `consumeVueA` rougit (ratchet anti-résurrection).
- **Docs** : `lecture/paquet.go` (`VueArretee`, `CauseMessageVueANonPorte`, `VueA`), en-têtes de
  `distribuer_tetes.go`, `frame_vue_messages.go`, `frame_harvest.go`, `marche_trames*.go`, doc de
  `CanalDesTetes` ; ADR 0037 (IR-3, distribution) : la vue A lue par un seul lecteur.
- **Révisions** : `grammar-2026-10-06`, `profile-2026-10-06` (§7).

## 3. Table par film

Carte v2 (`-mode v2 -denominateur-fixe`, table ECS figée de la campagne), base `87cdfa761` contre
tête du lot. Sonde `va_v1_research_test.go` (marche de cuisson, carte du match posée).

| Film | Build | Classe | Paquets | Sains (= base) | Utiles sains (= base) | Trames comparées | Écarts de tête, route, étendue, passe | Vues A lues au terminateur / paquets à événements | Taux | Accord E / S (localisés) |
|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | HI_1_13_0 | égale | 26 740 | 19 156 | 159 929 | 26 740 | 0 | 3 841 / 3 949 | 97,3 % | 99,2 % |
| `084a804d` | HI_1_10_0 | préfixe | 32 457 | 4 837 | 89 755 | 32 457 | 0 | 16 074 / 17 149 | 93,7 % | 71,7 % |
| `111fa685` | HI_1_10_0 | préfixe | 17 362 | 4 026 | 42 896 | 17 362 | 0 | 7 610 / 7 951 | 95,7 % | 80,5 % |
| `11de8353` | HI_1_9_0 | préfixe | 17 629 | 5 631 | 65 658 | 17 629 | 0 | 6 908 / 7 325 | 94,3 % | 82,9 % |
| `1c4c63c2` | HI_1_10_0 | préfixe | 79 550 | 13 389 | 165 987 | 79 550 | 0 | 23 504 / 24 997 | 94,0 % | 24,8 % |
| `396cfc92` | HI_1_13_0 | égale | 32 052 | 23 131 | 169 502 | 32 052 | 0 | 5 041 / 5 109 | 98,7 % | 97,0 % |
| `4f77afc1` | HI_1_13_0 | égale | 35 499 | 24 106 | 607 967 | 35 499 | 0 | 17 153 / 17 441 | 98,3 % | 90,1 % |
| `50247b26` | version-31 | illisible | 17 919 | 139 | 274 | 17 919 | 0 | 0 / 7 779 | 0,0 % | - |
| `51ebbc0f` | HI_1_13_0 | égale | 30 666 | 20 444 | 140 632 | 30 666 | 0 | 4 794 / 4 868 | 98,5 % | 93,5 % |
| `60ae07c4` | HI_1_8_0 | illisible | 49 696 | 13 948 | 83 685 | 49 696 | 0 | 0 / 6 056 | 0,0 % | - |
| `a349fea8` | version-33 | illisible | 28 751 | 424 | 3 654 | 28 751 | 0 | 0 / 14 492 | 0,0 % | - |
| `a521164d` | HI_1_4_1 | illisible | 11 130 | 692 | 78 | 11 130 | 0 | 0 / 4 956 | 0,0 % | - |
| `bcb6d393` | HI_1_12_0 | égale | 21 864 | 5 880 | 35 502 | 21 864 | 0 | 2 679 / 2 730 | 98,1 % | 98,5 % |
| `bf15f7ab` | HI_1_13_0 | égale | 31 053 | 28 603 | 216 104 | 31 053 | 0 | 4 260 / 4 340 | 98,2 % | 99,6 % |
| `bfecd02b` | HI_1_13_0 | égale | 31 232 | 27 668 | 239 064 | 31 232 | 0 | 5 188 / 5 274 | 98,4 % | 99,2 % |
| `c75f33b8` | HI_1_13_0 | égale | 27 800 | 23 872 | 153 997 | 27 800 | 0 | 3 003 / 3 066 | 97,9 % | 83,6 % |
| `d9781168` | HI_1_13_0 | égale | 43 645 | 26 317 | 180 854 | 43 645 | 0 | 8 523 / 8 737 | 97,6 % | 85,8 % |
| `e5adf7b2` | HI_1_11_0 | préfixe | 16 824 | 4 149 | 80 107 | 16 824 | 0 | 7 549 / 7 751 | 97,4 % | 85,9 % |
| `f75e7053` | HI_1_13_0 | égale | 28 502 | 23 673 | 162 137 | 28 502 | 0 | 4 178 / 4 263 | 98,0 % | 98,1 % |
| `fb1a1a72` | HI_1_13_0 | égale | 48 771 | 43 457 | 325 815 | 48 771 | 0 | 7 711 / 7 850 | 98,2 % | 98,6 % |
| **corpus** | | | 629 142 | **313 542** | **2 923 597** | 629 142 | **0** | | | |

Gate 2, par film (script `integ2/gate2.awk`) : net 0, sains perdus 0, gagnés 0, gains au bit 0,
pertes au bit 0, utiles sains perdus 0, sur les 20 films. Aucune perte à instruire, aucune factice
(rien ne change). « Trames comparées » compte toutes les trames delta (`FilmContext.Trames`), dont
les paquets à événements.

## 4. Mesure : vues A lues jusqu'au terminateur, et accord de E avec le début localisé actuel

Sonde `TestVAV1FinDeVueA` : marche de la carte v2 (`cmMarcher`), localisation de production
(`localiserLaListe`) sur le monde d'avant le paquet ; pour chaque paquet à événements, la vue A lue
par `lireLaVueA` (grammaire du film, tables de la région jouée de la carte du match) et sa fin E
comparée au début S. « Chaîne » = E < S et chaîne de records de la loi de E à S
(`chaineJusqua`). « E fermé » = la marche d'essai depuis E ferme le paquet (`lectureDEssai`).
MESURÉ, aucune décision n'en dépend en V1.

| Build | Classe | Paquets à événements | Lues au terminateur | Taux | Localisés | Accord | Chaîne | E < S (dont E fermé) | E > S (dont E fermé) | Accord + chaîne / localisés | Non localisés (dont E fermé) |
|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_13_0 | égale | 64 897 | 63 692 | **98,1 %** | 48 173 | 38 035 | 7 598 | 2 439 (1 303) | 101 (93) | **94,7 %** | 15 519 (10 476) |
| HI_1_12_0 | égale | 2 730 | 2 679 | **98,1 %** | 2 400 | 1 974 | 390 | 35 (1) | 1 (1) | **98,5 %** | 279 (50) |
| HI_1_11_0 | préfixe | 7 751 | 7 549 | 97,4 % | 5 842 | 3 630 | 1 390 | 815 (26) | 7 (4) | 85,9 % | 1 707 (163) |
| HI_1_10_0 | préfixe | 50 097 | 47 188 | 94,2 % | 30 571 | 12 408 | 4 457 | 13 458 (522) | 248 (23) | 55,2 % | 16 617 (725) |
| HI_1_9_0 | préfixe | 7 325 | 6 908 | 94,3 % | 5 363 | 3 267 | 1 177 | 903 (23) | 16 (0) | 82,9 % | 1 545 (138) |
| HI_1_8_0 | illisible | 6 056 | 0 | 0,0 % | - | - | - | - | - | - | - |
| HI_1_4_1 | illisible | 4 956 | 0 | 0,0 % | - | - | - | - | - | - | - |
| version-31 | illisible | 7 779 | 0 | 0,0 % | - | - | - | - | - | - | - |
| version-33 | illisible | 14 492 | 0 | 0,0 % | - | - | - | - | - | - | - |

Désaccords par mode de localisation, classe ÉGALE (11 films) : signature 462 (sur 41 759 localisés
par signature), chaîne 66, fermeture 485, fermeture au bit 1 563 sur 1 564. Classe PRÉFIXE : signature
6 299 sur 32 553, fermeture 2 198, fermeture au bit 6 862.

Arrêts de lecture (paquets à événements dont la vue A non vide ne va pas au terminateur), classe
ÉGALE : 85 non porté 1 157, 116 non porté 63, 12 non porté 14, 106 non porté 7, 5 refusé 15 (index
autre que la région jouée). Classe PRÉFIXE : genres au-delà du cardinal du film (121 à 127) et 85.

Lecture (estimé, pour V2, non décidé ici) : sur les films ÉGALE, E retombe sur le début localisé dans
94,7 à 98,5 % des localisés, et la marche depuis E ferme 10 526 des 15 798 paquets aujourd'hui non
localisés. Sur HI_1_10_0 (PRÉFIXE), l'accord tombe à 55,2 % (`1c4c63c2` 24,8 %) : c'est la famille du
tir à composantes de 3 bits de moins (R1 §3, D-VA-1 du plan), que la règle de V2 « prouvé paquet par
paquet » doit écarter. Les 462 désaccords de la signature sur les films ÉGALE sont l'ordre de grandeur
que le plan estimait (493 + 83 sur le monde de `2393d7db7`).

## 5. Vérifications de la représentation intermédiaire (session levelup-57)

| | Vérification | Statut | Preuve |
|---|---|---|---|
| (a) | Tête donnée aux canaux (`VueA.Etat`, genres, continuation) IDENTIQUE paquet par paquet | [x] | Sonde `TestVAV1Tete`, 20 films : 629 142 trames de la marche de cuisson, tête (`teteDe`) et route (`listeAnnoncee`) confrontées au témoin `teteDAvant` (recopie de `consumeVueA` + `teteDe` de `87cdfa761`) : 0 écart ; passe des têtes (`distribuerLesTetes`) sur un contexte neuf : 629 142 trames, tête ET vue A rangée identiques à celles de la marche, 0 écart. Unitaires : `TestLaTeteEstLueALIdentique` (20 000 payloads × 3 grammaires), `TestRangerLaTete`, `TestLaPasseDesTetesEstLaTeteDeLaMarche` |
| (b) | Vue A lue UNE fois ; étendues de `lecture.Paquet` inchangées hors genres désormais portés ; préambule hors rangs sans changement de sortie | [x] | Une lecture : `rangerLaTete` est le seul appel en marche de cuisson, la marche reçoit la vue A (`TestLaMarcheNeRelitPasLaVueAQuElleRecoit`, mutation m10 ROUGE) ; garde-rail archlint. Étendues : sonde, par film « identique » ou « désormais portée » (même début, même premier genre, étendue ≥), 0 « écart » (colonne `vue_a` de `va_v1_tete.tsv`) : 132 750 vues désormais portées, 496 392 identiques. Préambule : `DecodeFrameViewsCurseur` hors rangs (`br.Skip(skipLeadBits)`) GARDÉ tel quel — il ne lit pas la vue A, il saute l'amorce ; sous rangs, `decodeFrameParRangs` → `lireTrameParRangs` sans vue A reçue → `lireLaVueA` sous `grammaireDeLaVueA{}` = la tête seule, égale à `consumeVueA` (`TestLaTeteEstLueALIdentique`, cas « film sans table » : étendue, état et genres identiques) ; `lectureDEssai` passe par là ; `candidatsDeTete` (balayage depuis p = 0) inchangé, il ne lit pas la vue A. Sortie : carte v2 et killsource identiques à l'octet |
| (c) | `DebutParVueA` dans `lecture.DebutDeVueB`, ADR 0037 IR-6 amendé | [~] V2 | En V1, aucun paquet ne part de E : une valeur `DebutParVueA` que personne ne pose serait du code mort (règle 7), et amender IR-6 pour un début qui n'existe pas serait une doc inversée. Reporté à V2, par dépendance du plan (V2 = l'étape où E localise). L'ADR 0037 est amendé en V1 là où la structure change (IR-3 : la vue A lue par un seul lecteur, E rangé, aucune localisation) |
| (d) | Marche à huit vues de production (`object_deaths_march.go:133`, `object_deaths_calibrate.go`, `replay/build_vehicles.go`) : morts, véhicules, killsource inchangés ; compte de `repli_localisation_largeur_libre` | [x] | Fichiers non touchés. `replay-equiv` : étapes `killsource`, `killRefs`, `vehicles`, `objectives`, `movementStates` identiques sur 20 films. Registre des replis (journal « repli déclenché », 522 lignes film × repli) identique base / tête ; `repli_localisation_largeur_libre` : 25 686 déclenchements sur 20 films, des deux côtés (il baissera en V2 quand E localisera) |
| (e) | `film_context.go` à 500 lignes | [x] | 500 lignes avant et après : `corr, corrLue, corrLu bool` (2 lignes) remplacé par `duFilm grammaireDuFilm` (2 lignes) ; la logique vit dans `controle_corruption_du_film.go` |
| (f) | `consumeVueA` supprimé ; préambule de `DecodeFrameViewsCurseur` gardé et écrit s'il ne passe pas par le lecteur unique | [x] | Supprimé, ratchet. Le chemin hors rangs de `DecodeFrameViewsCurseur` est gardé : il ne lit pas de vue A (amorce de deux bits sautée, `DefaultPacketPreambleBits`) ; le faire passer par `lireLaVueA` changerait sa sortie sur les vues A non vides. Écrit ici |
| (g) | `TestLaMarcheRangeUneVueANonPortee` et `distribuer_tetes_test.go` : genre 5 porté, raison écrite | [x] | Renommé `TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide` : le genre 5 est désormais un vrai `projectile_detonate` écrit d'après `FUN_1408096f8` (position de niveau 0xf), lu jusqu'au terminateur sous un film récent, arrêté après son genre sous un film sans table, et la marche s'arrête au même bit dans les deux cas (raison écrite dans le godoc). `TestRangerLaTete` : cas « film récent » (vue terminée, deux genres, tête inchangée) et « film sans table ». Aucun fichier de baseline ne nommait l'ancien test |

Fichiers de la RI touchés (à soumettre à levelup-57 avant fusion) : `lecture/paquet.go` (docs
seules), `distribuer.go` (doc de `CanalDesTetes` seule ; cœur non touché), `distribuer_tetes.go`,
`distribuer_tetes_test.go`, `marche_trames.go`, `marche_trames_rangs.go`, `marche_trames_ranger.go`,
`marche_trames_test.go`, `marche_fuzz_test.go`, `frame_closure_test.go`, `frame_closure_detail_test.go`
(appel `departDeTrame`), `frame_harvest.go`, `film_context.go`, `controle_corruption_du_film.go`,
`docs/adr/0037-film-intermediate-representation.md`. NON touchés : `movement_states.go`,
`frame_closure_detail.go`, le cœur de `distribuer.go`, `object_deaths_*.go`, `replay/build_vehicles.go`.

## 6. Gates (sorties exactes)

Environnement : `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg3-vuea`, CGO msys64 ucrt64, une
commande go à la fois ; films lus en place (`data/cache/film_chunks`, lecture seule) ; racine factice
et copie du parc au scratchpad (`scratchpad/cg3-V1/`).

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0. `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc=0.
3. `go test ./internal/archlint/` : `ok levelup/go-api/internal/archlint 36.523s`.
4. G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` : rc=0, 21 paquets `ok` (sur le code final, après les dernières retouches de tests et de chroniques). Garde-fou des recopies du pilotage : `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` vert.
5. `golangci-lint run` sur `film/internal/grammar/...`, `film/internal/profile/...`,
   `film/internal/facts/...`, `internal/archlint/...` (toutes les issues, tests compris) :
   `0 issues.` Sous `--build-tags=research` (grammar, profile) : 26 issues, la même liste à la base
   `87cdfa761` (dette existante : goconst 20, goimports 2, unparam 1, unused 3), aucune neuve.
6. Carte v2 (`research/cmd_fermeture -mode v2 -denominateur-fixe`, 20 films, `-plafond-gib 4`) : rc=0,
   20 films, 0 échec ; `fermeture_paquets.tsv` identique à l'octet ; `fermeture_films.tsv` identique
   hors `pic_octets` et `duree_ms` ; tous les autres TSV identiques. Gate 2 : §3.
7. `cmd/killsource json <film> -carte … -cache data/cache -catalogue map_quant_bounds.json`, 20 films :
   rc=0 partout, 20 JSON identiques à l'octet ; stderr identiques hors horodatage.
8. `replay-equiv -repo-root <racine factice>` (20 films) : « BILAN : 0 identique(s), 20 different(s) »
   des deux côtés contre les références non re-figées depuis la vague 1 ; base contre tête, étape par
   étape (61 étapes) : seule `artifact` diverge, sur les 20 films. Fixtures de contrat web régénérées
   (`REPLAY_CONTRACT_UPDATE=1 … -run ContractFixtures -update`) : les 8 JSON sont identiques à
   `87cdfa761` une fois les chaînes `grammar-…` / `profile-…` neutralisées.
9. `replay-corpus-gate --reference=base --base=87cdfa761 --parc-root <copie> --work-root <scratchpad>` :
   rc=0. Tableau : 19 témoins, schéma 78 -> 78, gains 0, pertes 0, changements 0, statut « ok » pour chacun ; « BANC DE VERITE (19 temoin(s)) » : 19 / 19 « ok ». Télémétrie seule (jamais comptée) : `coverage.decoder.grammarRev` `grammar-2026-10-03.5 -> grammar-2026-10-06` et `coverage.decoder.profileRev` `profile-2026-09-17.3 -> profile-2026-10-06` sur 19 témoins. Aucune perte `[FILET]` à instruire.
10. Performance et mémoire de cuisson (`replay-equiv`, 3 films, deux passages alternés base / tête) :
   mesuré, secondes et pic (Gio), passage 1 / passage 2 : `1c4c63c2` base 164,9 / 165,1 s (2,07 / 2,06), tête 166,4 / 166,6 s (2,10 / 2,14) ; `000d5950` base 8,02 / 8,10 s (0,31 / 0,28), tête 8,38 / 8,34 s (0,27 / 0,28) ; `fb1a1a72` base 15,55 / 15,54 s (0,43 / 0,41), tête 16,08 / 15,98 s (0,45 / 0,42). La lecture complète de la vue A coûte de +0,9 % (`1c4c63c2`) à +4 % (`000d5950`) en temps ; le pic varie dans le bruit d'un passage à l'autre (±0,04 Gio). Les temps du `replay-equiv` sur 20 films ne servent pas ici : la tête y a tourné en même temps que la sonde.
11. Mutations (`-overlay`, suite entière du paquet ; `scratchpad/cg3-V1/mutations2.sh`,
   `mutations_final.txt`) : **24 / 24 ROUGES**.

| Mutation | Règle | Rougit |
|---|---|---|
| m01 égalité lue comme préfixe (`== GenresVueA` → `>= GenresVueA-2`) | deux classes | `TestLaTableDuFilmSeRangeEnDeuxClasses`, `TestLesBobinesDeclarentLeurTableEtLeurSimulation` |
| m02 version non comparée | versions | idem |
| m03 préfixe du Script inversé | Script | `TestLeScriptSuitLaSimulationDeLEnregistreur`, `TestUnVraiPaquetDeQuaranteScripts` |
| m04 simulation lue au premier champ (`BodyBit`) | options de partie | `TestLesOptionsDePartieSeLisentEnTeteDuCorps`, `TestLesBobinesDeclarent…`, `TestUnVraiPaquet…` |
| m05 polarité de k (39) | 39 | `TestLeLancerLitSaChargeSelonK` |
| m06 niveaux 0xf / 0xc → 0x10 | 5 et 6 | `TestLesPositionsAIndexSeLisentSurLaRegionJouee`, `TestLaMarcheDepuisLaTete…` |
| m07 index autre que la région lu | 5 et 6 | `TestLesPositionsAIndexSeLisentSurLaRegionJouee` |
| m08 tête au dernier genre | tête | `TestRangerLaTete`, `TestLaTeteEstLueALIdentique`, `TestLaPasseDesTetes…`, `TestGoldenMiniBobineFamilles` |
| m09 route = `VueArretee` seule | route | `TestLaTeteEstLueALIdentique`, `TestFrameClosureRatchet`, … |
| m10 la marche relit la vue A reçue | une lecture | `TestLaMarcheNeRelitPasLaVueAQuElleRecoit` |
| m11 terminateur non compté | terminateur | 31 tests dont `TestLaVueALueRendLeDebutDeLaVueB`, `TestUnVraiPaquetDeQuaranteScripts` |
| m12 contrôle de corruption ignoré | corruption | `TestLeControleDeCorruptionSuitLaCharge` |
| m13 bit de configuration ignoré | configuration | `TestLeBitDeConfigurationAZeroArreteLaVueA` |
| m14 cardinal du film ignoré | cardinal | `TestLaVueANeDevineRien` |
| m15 vue A rangée sans genres | rangement | `TestRangerLaTete`, … |
| m16 marche qui traverse une vue A lue non vide | marche | `TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide` |
| m17 passe des têtes qui compte le repli (paquet `killcollector`) | profil de la passe | `TestLeRapportDuContexteDuPontEstVerseALaPasse` |
| m18 `consumeVueA` restauré et appelé (SUR DISQUE, fichier temporaire retiré) | lecteur unique | `TestLectureDeLaVueAUnique` |
| m19 genre vide lu comme non vide | table des genres | `TestLaTableDesGenresEstCelleDuJeu`, `TestLaTableDesGenresCouvre…` |
| m20 version native d'un genre changée | versions natives | `TestLesBobinesDeclarent…`, `TestUnVraiPaquet…` |
| m21 polarité des dégâts | `FUN_1407f2058` | `TestLesDegatsLisentLaPorteInverseeDuJeu` |
| m22 largeur du tir court 10 → 11 | tir | `TestLeTirCourtSArreteApresSaDirection` |
| m23 polarité du rechargement | `FUN_1407f2058` | `TestLaVueALueRendLeDebutDeLaVueB` |
| m24 genre sans charge portée accepté | rien n'est deviné | `TestLaVueANeDevineRien` (après ajout du vecteur genre 85, §8) |

## 7. Révisions

- `grammar.Rev` : `grammar-2026-10-03.5` → **`grammar-2026-10-06`** (sources de la couche changées ;
  chronique : « aucune sortie ne change »). Rang : `grammar-2026-10-04` et `grammar-2026-10-05` sont
  portés par `feat/ri-etape2` (étape 2.7.a de la RI), `grammar-2026-10-03.6` par le worktree du lot
  LR ; `.7` du 2026-10-03 serait un trou. Premier rang libre sans trou : le 2026-10-06 (rang 1).
- `profile.Rev` : `profile-2026-09-17.3` → **`profile-2026-10-06`** (`identite.go` : simulation de
  l'enregistreur ; `profile-2026-10-05` est porté par `feat/ri-etape2`).
- `killsource.Rev` : **constante** `killsource-2026-09-27` (D23 : aucune sortie persistée ne change,
  20 films identiques à l'octet) ; empreinte recopiée (`-update-killsource-rev`), complément de
  chronique.
- `objectives.Rev` : **constante** `objectives-2026-09-27` (étape `objectives` de `replay-equiv`
  identique) ; empreinte recopiée, complément dans `objectives/rev.go`.
- `source.Rev` (`source-2026-09-16.2`), `replay.SchemaVersion` (78) : inchangés.
- **`SchemaDesFaits` (4) : inchangé, ÉCART À SA DOCTRINE, instruit (correction 3 du contrôle).** La
  section 2 des faits persistés (identité du film, JSON de `profile.FilmIdentity`,
  `replay/filmfacts_fichier.go`, `sectionIdentite`) gagne deux champs, `SimulationDeLEnregistreur` et
  `OptionsDePartieLues`. La doctrine de `SchemaDesFaits` (`filmfacts_fichier.go:130`) dit qu'il
  « monte quand une section naît, meurt ou change de contenu » : à la lettre, il devrait monter. Il ne
  monte pas, pour les raisons suivantes (lues dans le code) :
  - la fraîcheur des faits se juge tout ou rien sur l'en-tête : `FilmFactsEntete.Frais`
    (`filmfacts_entete.go:127-137`) compare codec, schéma, PUIS les cinq révisions de couche
    (`memesRevisionsDeCouche`, `GrammarRev` et `ProfileRev` compris). V1 fait monter `grammar.Rev`
    ET `profile.Rev`. Un fichier écrit avant V1 est donc refusé SUR SON EN-TÊTE
    (`ErrFilmFactsRevisions`), avant toute lecture de la section 2, et redécodé. C'est ce que la
    doctrine du schéma protège (« un fichier PÉRIMÉ doit se dire périmé sur son en-tête ») ;
  - la section 2 est relue par `json.Unmarshal` (`lireSectionJSON`), qui laisse à zéro un champ
    absent. Si un fichier antérieur passait l'en-tête, il serait relu avec une simulation nulle et
    `OptionsDePartieLues` faux, sans erreur. Ce cas n'est pas atteignable, puisque les révisions
    diffèrent ;
  - aucun lecteur des faits n'emploie ces deux champs. Ils ne servent qu'à la dérivation de la
    grammaire de la vue A, depuis le `chunk_00` du film (`grammaireDeLaVueASousFilm`,
    `vue_a_charges_execution.go`), au décodage, jamais depuis un fichier de faits ;
  - précédent : le lot 5.18.1 (`b867e835b`, 2026-09-22) a ajouté `ControleDeCorruption` à
    `FilmIdentity`, donc à la même section 2 (persistée depuis le lot 4.1.1, `25eb30596`,
    2026-09-17), sans monter `SchemaDesFaits` (le commit ne touche aucun `filmfacts*`).

  Ce qui reste un écart : la phrase de doctrine n'est pas tenue à la lettre. La corriger, ou monter le
  schéma à la fusion de la campagne, est une décision du pilote. Elle n'est pas prise ici (règle 5 du
  plan : pas de correction hors périmètre).
- Régénérés par les commandes du dépôt : `grammar_rev.golden`, `profile_rev.golden`,
  `killsource_rev.golden`, `objectives_rev.golden`, `types/testdata/shapes.golden` (ligne des
  révisions seule), fixtures web `replay_schema_78_*` et `manifest.json` (chaînes de révision seules).

## 8. Écarts

- **Reprise.** L'étape a été reprise d'une session interrompue (worktree et scratchpad déjà peuplés,
  aucun commit). Les binaires de la tête qui avaient servi aux mesures précédentes (10 h 03) étaient
  antérieurs aux dernières modifications du code (10 h 19 – 10 h 22) : TOUTES les mesures ont été
  refaites sur le code final (tête reconstruite à 15 h 28). La session interrompue avait écrit des
  scripts de travail en Python au scratchpad (restructuration de tests, génération de mutations) ;
  rien n'en est versionné, et les mutations de ce rapport ont été régénérées en bash/sed depuis les
  sources finales.
- **Révisions.** La session interrompue avait pris `grammar-2026-10-05` et `profile-2026-10-05`,
  valeurs déjà portées par `feat/ri-etape2` : renommées en `-10-06` (§7).
- **(c) reporté à V2** (§5).
- **Vecteur ajouté** : la mutation « genre sans charge accepté » survivait. Le vecteur de LN pour
  cette règle était le genre 15, désormais porté ; `TestLaVueANeDevineRien` lit maintenant un 85
  (`PlayerKilledEvent`, non vide, charge non portée) suivi d'un 0 que la mutation prendrait pour le
  terminateur. Puis ROUGE.
- **Vecteur corrigé** : `TestUnVraiPaquetDeQuaranteScripts` posait l'état du Script par la seule
  branche « avec préfixe » ; il applique désormais la règle de l'écrivain dans les deux branches.
- **Garde-rail archlint et `-overlay`** : le ratchet lit les sources sur le disque ; une surcouche ne
  le touche pas. La mutation m18 est prouvée SUR DISQUE (fichier temporaire
  `zz_mutation_lecture_retiree.go`, retiré aussitôt) ; le test porte aussi ses vecteurs
  (`TestGardeRailVueAVecteurs`).
- **Positions 5 et 6** : port par le lecteur existant sur tables restreintes, et non par la fonction
  à part de R2 (§2) ; même lecture (vecteurs et mutations m06, m07).
- **Killsource sur 20 films** (19 témoins + `1c4c63c2`, carte Refuge), au lieu des 19 demandés.

## 9. Découvertes

- **D-VAV1-1** — Les films PRÉFIXE sont lus jusqu'au terminateur et leur vue A est rangée
  `VueTerminee` dans `lecture.Paquet`, sous la grammaire de l'exécutable courant, qui n'est pas
  prouvée être celle de leur écrivain (tir de HI_1_9/HI_1_10, R1 §3 ; accord mesuré 55,2 % sur
  HI_1_10_0). En V1 aucun consommateur ne lit au-delà de la tête. V2 ne devra pas prendre
  `VueTerminee` d'un film PRÉFIXE pour une preuve : la classe du film doit accompagner E jusqu'à la
  décision (règle (2) de l'utilisateur, `LectureVueC.Fermee` paquet par paquet).
- **D-VAV1-2** — Sur les films ÉGALE, la marche depuis E ferme 10 526 des 15 798 paquets que la
  localisation actuelle ne localise pas (mesuré). C'est le gros du gain attendu de V2.
- **D-VAV1-3** — Le repli « fermeture au bit » de la localisation est en désaccord avec E sur
  1 563 de ses 1 564 paquets des films ÉGALE (mesuré) : ce repli ne retombe presque jamais sur la fin
  lue de la vue A.
- **D-VAV1-4** — Arrêts restants sur les films ÉGALE : 85 (1 157), 116 (63), 12 (14), 106 (7), 5
  refusé hors région jouée (15). 85 et 116 attendent V3 (variante de partie dans `chunk_00`).
- **D-VAV1-5** — Les rangs de révision du 2026-10-04 et du 2026-10-05 sont pris par la RI (2.7.a) :
  la seconde des deux branches fusionnée renumérotera (règle des rangs sans trou).

## 10. Pièces

Scratchpad `scratchpad/cg3-V1/` (non versionné) : `carte_base/`, `carte_tete/`, `gate2.tsv`,
`ks_base/`, `ks_tete/`, `re_base_tsv/`, `re_tete_tsv/`, `replis_base2.tsv`, `replis_tete2.tsv`,
`sonde2/` (`va_v1_tete.tsv`, `va_v1.tsv`, `va_v1_paquets.tsv`), `taux_par_build2.tsv`,
`taux_par_film2.tsv`, `mut2/`, `mutations_final.txt`, `gfilm_final.log`, `lint_final.txt`, `lint_final_research.txt`,
`lint_base2_research.txt`, `gate.log`, `va_v1_corpus_gate.json`, `perf2_*.log`. Scripts : `env.sh`,
`construire.sh`, `carte.sh`, `ks.sh`, `re.sh`, `replis.sh`, `sonde.sh`, `taux.awk`,
`mutations2.sh`, `gate.sh`, `perf2.sh`. Versionnés : `va_ghidra/` (décompilés de R2 pour 15, 39 et
les options de partie), `testdata/vue_a_bcb6d393_1_204.bin` et sa PROVENANCE.

## 11. Corrections du contrôle indépendant (2026-10-05)

Le contrôle de l'étape V1 (`scratchpad/cg3-V1-ctl/`) a tenu l'étape sur l'essentiel : sortie identique
à la base, vérifications (a) et (b) tenues paquet par paquet, règles relues dans Ghidra. Il a demandé
trois corrections mineures. Les trois sont appliquées ; aucune n'a été jugée fausse sur pièces.
`origin/feat/v75` vaut toujours `87cdfa761` (`git fetch` avant les gates).

| # | Correction | Verdict sur pièces | Action |
|---|---|---|---|
| 1 | Entrée de `.ai/thought_log.md` absente du commit `e6ec7abd4` | Fondée : le commit ne touche pas `.ai/thought_log.md` (`git show --stat`), alors que les commits de lot précédents en portent une (`87cdfa761`, `d9c268c9f`) | Entrée `[2026-10-05]` ajoutée |
| 2 | Mutation s01 VERTE : `profilDeLaVueA` sans `bal.Grammaire.ControleDeCorruption = c.grammaireDuFilmDerivee().controle` | Fondée : aucune bobine du dépôt ne déclare ce contrôle (`controle_corruption=false [FILM]` dans les cinq goldens de `killsource`), et le seul test qui oppose la passe des têtes à la marche roule sur deux d'entre elles | Vecteur ajouté : `TestLaPasseDesTetesLitLeControleDuFilm` (`distribuer_tetes_test.go`) |
| 3 | Section 2 des faits élargie de deux champs sans montée de `SchemaDesFaits`, écart non dit | Fondée, et sans risque de faits périmés servis : les révisions de couche, jugées sur l'en-tête, refusent tout fichier antérieur | Instruit au §7 ; le schéma n'est pas monté |

**Le vecteur (correction 2).** Un film dont le registre est celui de la bobine `000d5950` (`chunk_00`
réel) et dont le seul chunk de données porte une trame delta écrite bit à bit : configuration 1, un
zoom (genre 21, trois gardes fermées, `R(2)`), puis le contrôle `R(1) = 1, R(32)` de `FUN_14076cea8`,
puis le terminateur. Son profil est celui que la bobine rend, sauf deux champs de la section
d'identification : la table native des genres (classe ÉGALE) et le contrôle de corruption VRAI. Le
profil est posé avant la première dérivation, qui le lit comme celui du film. La passe des têtes et
la marche complète doivent ranger la même vue A, `[1, 48)`, terminée, de genre 21 seul.

- Sur le code : VERT (suite entière du paquet `grammar`, `ok … 49.898s`).
- Mutation m25 = s01 (`-overlay`, suite entière du paquet `grammar`) : ROUGE, sur ce seul test :
  `passe des tetes : vue A {Debut:1 Bits:21 Etat:2 Genres:[21 127]}, attendu {Debut:1 Bits:47 Etat:1
  Genres:[21]}`. Sans le contrôle, la passe prend le bit du contrôle pour la continuation d'un second
  message, de genre 127 (au-delà du cardinal : la vue s'arrête).

| Mutation | Règle | Rougit |
|---|---|---|
| m25 `profilDeLaVueA` sans le contrôle de corruption du film (s01 du contrôle) | profil de la passe | `TestLaPasseDesTetesLitLeControleDuFilm` |

Le contrôle a aussi trouvé VERTES s05 (garde `d.vueA.Debut == l.debutVueA`) et s06 (`<=` remplacé par
`<` dans la borne du bit de configuration). Il les juge équivalentes en pratique et ne demande aucune
correction : aucune n'est faite.

**Gates rejoués (portée de la correction : un fichier de test du paquet `grammar` et des documents).**
`gofmt -l ./internal/ ./cmd/` : vide. `go vet` du paquet `grammar`, avec et sans `-tags=research` : rc=0.
`go test ./internal/games/halo_infinite/film/internal/grammar/ -count=1` : `ok` (le test de révision du
paquet compris : empreinte `grammar` inchangée, aucun fichier de production touché).
`go test ./internal/archlint/` : `ok`. Un premier passage était ROUGE : `TestNoRawKillScopeLiteral`
refuse le littéral `"marche"` (une valeur de portée de `match_kill_events`), qui servait de clé de
table dans le test. La clé est devenue `"marche complete"`, puis le gate est passé VERT.
`golangci-lint run` sur `film/internal/grammar/...` : `0 issues.` ; sous `--build-tags=research
--new-from-rev=e6ec7abd4` : `0 issues.` Aucune sortie de production ne change. La carte v2, killsource,
`replay-equiv` et `replay-corpus-gate` ne sont pas rejoués, faute d'objet : le gate 2 reste celui du §3.
Pièces : `scratchpad/cg3-V1-corr/` (`s01.go`, `s01.json`, `s01_grammar2.log`, `grammar_tete.log`).

## 12. Corrections de la relecture RI (2026-10-05)

Deux relecteurs de la session levelup-57 ont relu l'étape sur `3bacfadeb`. Chaque constat a été
vérifié sur pièces avant d'être corrigé ; aucun n'a été jugé infondé. Aucune sortie de production ne
change (contrôle ci-dessous). La révision `grammar-2026-10-06` est gardée ; son empreinte est
régénérée à révision constante (`7aa119ae…`, `testdata/grammar_rev.golden`).

| # | Constat | Verdict sur pièces | Action (fichier:ligne) |
|---|---|---|---|
| 1 | `lectureDeTrame.finVueA` écrit, jamais lu | Fondé : `grep finVueA` ne trouve que la déclaration et les deux écritures | Champ et commentaire retirés, `debutVueA` gardé et commenté (`marche_trames_rangs.go:17-22`) |
| 2 | Branche « vue A non reçue » (`marche_trames_rangs.go:76`) sans test sur une vue A non vide | Fondé : sous la mutation m26, seul le test neuf rougit | `TestLaMarcheSansVueARecueNeTraversePasUneVueANonVide` (`marche_trames_test.go:207`), par `DecodeFrameViewsCurseur` puis `decodeFrameParRangs` |
| 3 | Règle du Script inversée et tables de la région jouée vidées : aucun test hors research ne rougit | Fondé : `TestUnVraiPaquetDeQuaranteScripts` recopiait la règle, `TestLesPositions…` posait les tables à la main, `TestLeScriptSuit…` posait l'état du Script | Les trois tests passent par la production : `profilDIdentite` (`profile.Resoudre`, `vue_a_execution_test.go:104`), `scriptDuFilm` (`:113`), `grammaireDeLaVueASousFilm(...).positions` (`:259`), `grammaireSousFilm` et `grammaireDeLaVueASousFilm` (`:304`) |
| a | `br.Skip(3 * 12)` et `br.Skip(3 * w)` recopient `FUN_140c1e9d4` | Fondé ; `consume140c1e9d4` existe (`components_biped_ability.go:318`). Même avance du curseur : `Skip(n)` et `ReadBits(n)` avancent tous deux `pos` de `n` (`source/bits.go`) | Appels (`vue_a_charges.go:153`, `vue_a_charges_tir.go:99`), liste de l'en-tête complétée (`vue_a_charges_tir.go:9-12`) ; `//nolint:unparam` de `consume140c1e9d4` retiré, sa largeur varie désormais |
| b1 | `vue_a_versions.go:33-34` : « ce que la marche fait de sa fin dépend de la classe » | Fondé : la classe n'est lue que par `lireLaVueA` (`vue_a_lecture.go:86`), la marche ne lit pas la fin d'une vue A qui porte un message | Phrase réécrite (`vue_a_versions.go:33-36`) |
| b2 | « seule implantation de `FUN_14076a1c4` » (`vue_a_lecture.go:3`, `:25-26`, archlint `:6-7`) | Fondé : `readPacketHead` (`event_list.go:87`) relit la tête, et par lui `teteDuPayload`, `lireEnteteTir36`, `scanChunkDamages` | Les trois textes disent ce qui est unique : la lecture COMPLÈTE, jusqu'au terminateur (`vue_a_lecture.go:3`, `:24-31`, `archlint/film_vue_a_lecteur_unique_test.go:6-14`) |
| b3 | Contrat de `lireE524Sur` : « après la porte » | Fondé : avec `regionSeule`, `ok=false` tombe après l'index (`lecteur_position.go:148-150`) | Contrat réécrit pour les deux cas (`lecteur_position.go:137-140`) |
| b4 | `rev_chronique.go:397-399` : « 46 genres (44 du lot LN) » | Fondé : un test d'overlay (non versionné) qui parcourt les 123 genres compte 47 genres non vides à charge portée et 13 vides ; à `d9c268c9f` (fin de LN), les `case` des trois aiguillages couvrent 45 genres | « 47 genres (45 du lot LN…) » (`rev_chronique.go:398`) ; §1 corrigé (45, 47, 60 lisibles) |
| c | `f58 == 1`, `br.ReadBits(3) == 1` | Fondé ; lus dans `ln_ghidra/g0_1407f15a4.c:354` (`*(int *)(param_3 + 0x58) == 1`) et `g40_140ff8d70.c:118-119` (`*param_3 = bVar10 ; if (bVar10 == 1)`) | `degatsF58AvecOctet` (`vue_a_charges.go:75-77`), `teteCorpsACorpsAvecSuite` (`vue_a_charges_armes.go:73-75`) |
| d | Complexité de `chargeDuGenre` et `chargeDArme` > 12 | Fondé (19 et 23 genres) ; `gocyclo` est exempté sur `film/internal/grammar/` (`.golangci.yml`), la règle 5 demande la justification | `//nolint:gocyclo // un case par genre de message porte (aiguillage)`, forme de `dispatch_player.go:336` (`vue_a_charges.go:30`, `vue_a_charges_armes.go:231`) |
| e | Troisième copie de `FUN_1407f15a4` (`killsource.evBody0`, `lot1DecodeDamageAftermath`, `chargeDegatsApres`) | Fondé ; `lot1DecodeDamageAftermath` lit « (2) +0x10 : R(1) ; si 1 : R(5) » contre « si 0 » pour les deux autres (D-LN-2). Non centralisé : la sortie de `weapon_hits` changerait et `killsource` serait touché | Exemption datée, avec son critère de retrait (`vue_a_charges.go:81-86`) ; découverte D-VAV1-6 |
| f | Troisième copie de la garde `bit < 0 OU (bit+N+7)/8 > len(d)` (`film_identity.go:212`, `:226`, `:247`) | Fondé | Helper `tientDansLeTampon` (`film_identity.go:258-262`) ; garde-rail `archlint/film_garde_de_tampon_test.go` (une seule écriture, dans l'hôte ; vecteurs) |
| g | Troisième copie du balayage de la production (`film_vue_a_lecteur_unique_test.go:91-136`, `film_localisateur_unique_test.go:133-180`, `film_tri_total_test.go:126-167`) | Fondé | `archlint/film_balayage_test.go` : `balayerLaProduction` et `balayerLaProductionHorsResearch`, appelés par les trois tests et par le garde-rail de f |

**Mutations** (`-overlay`, suite entière du paquet `grammar` hors `TestGrammarRevSuitLaGrammaire` ;
`scratchpad/mut/`) :

| Mutation | Règle | Rougit |
|---|---|---|
| m26 vue A non reçue prise pour le bit d'amorce aveugle (`FluxVueA{Debut, Vide, Porte, Fin: Debut+1}` à `marche_trames_rangs.go:76`) | branche sans vue A reçue | `TestLaMarcheSansVueARecueNeTraversePasUneVueANonVide` (seul) |
| m27 règle de l'écrivain inversée dans `scriptDuFilm` (`==` devient `!=`) | Script | `TestLeScriptSuitLaSimulationDeLEnregistreur`, `TestUnVraiPaquetDeQuaranteScripts` |
| m28 tables de la région jouée vidées à la dérivation (`grammaireDeLaVueASousFilm` sans `positions`) | 5 et 6 | `TestLesPositionsAIndexSeLisentSurLaRegionJouee` |
| m29 garde de tampon réécrite à la main dans `u32DuFlux` (SUR DISQUE, restaurée) | garde unique | `TestGardeDeTamponUnique` (archlint) |

**Sortie inchangée** (points a et f, code de production). Binaires construits depuis
`git archive 3bacfadeb` et depuis l'arbre corrigé (`scratchpad/vav1corr/`, `controle.sh`) :

- carte v2 (`cmd_fermeture -mode v2 -denominateur-fixe -paquets`) sur `bcb6d393` (HI_1_12_0, ÉGALE),
  `fb1a1a72` (HI_1_13_0, ÉGALE), `e5adf7b2` (HI_1_11_0, PRÉFIXE) : rc=0 des deux côtés,
  `fermeture_paquets.tsv` identique à l'octet (sha256 `17908ebe…6a2eb5e`, 87 460 lignes), tous les
  autres TSV identiques, `fermeture_films.tsv` identique hors `pic_octets` et `duree_ms` ;
- `cmd/killsource json` sur `bcb6d393` et `fb1a1a72` : rc=0, JSON identiques à l'octet
  (`6c98d114…`, `5c20132b…`), stderr identiques hors horodatage.

**Gates.** `gofmt -l ./internal/ ./cmd/` : vide. `go vet` du paquet `grammar` et d'`archlint` : rc=0 ;
`go vet -tags=research` du paquet `grammar` : rc=0. `go test ./internal/games/halo_infinite/film/internal/grammar/ -count=1` :
`ok … 61.745s`. `go test ./internal/archlint/ -count=1` : `ok … 43.820s`. `golangci-lint run` sur
`grammar/...` et `archlint/...` : `0 issues.` ; sous `--build-tags=research --new-from-rev=3bacfadeb`
(`grammar/...`) : `0 issues.`

**Découvertes** (notées, non traitées) :

- **D-VAV1-6** — `FUN_1407f15a4` a trois lectures de production, dont une (`lot1DecodeDamageAftermath`,
  `weapon_hits`) lit la porte de `FUN_1407f2058` à polarité inversée (D-LN-2). Un lot dédié doit les
  centraliser et corriger D-LN-2, la sortie de `weapon_hits` changeant alors.
- **D-VAV1-7** — `FUN_14080bb4c` (Script) a deux lectures : `chargeScript` et `killsource.evBody15`
  (`gate15` tranché par film côté killsource, lu dans `chunk_00` côté vue A). Deux copies : dans la
  règle 6, à surveiller.
- **D-VAV1-8** — La tête du tir (`FUN_14080c1f8` jusqu'aux deux R(1) qui suivent l'arme) se lit deux
  fois : `chargeTirArme` et `lireEnteteTir36Sous`. Deux copies : dans la règle 6, à surveiller.
