# LOT VA, ÉTAPE V2 — La fin de la vue A fixe le début de la vue B (2026-10-05)

> Campagne de grammaire, lot VA. Décisions de l'utilisateur du 2026-10-04 : « Lire la vue A jusqu'au
> bout », puis « Oui, lance » sur les règles (1) à (3) ci-dessous. Plan de conception :
> `scratchpad/va-conception/PLAN_LOT_VUE_A.md` (non versionné) et sa vérification adverse ; étape V1 :
> `LOT_VA_V1.md` (`e6ec7abd4`, `3bacfadeb`).
>
> Worktree `LevelUp-wt-cg3-vuea`, branche `feat/cg3-vue-a`, tête de départ `3bacfadeb` (V1), base du
> lot `87cdfa761` (= `origin/feat/v75`, contrôlé par `git fetch` au début de l'étape et avant les
> gates : pas d'avance). Référence de mesure : binaires de `87cdfa761` (ceux de V1, dont la sortie est
> identique à l'octet à `87cdfa761`).
>
> Conventions : **lu** = lu dans le jeu (Ghidra, `HaloInfinite.exe` HI_1_13_0, lecture seule) ;
> **mesuré** = compté par un outil sur les films ; **estimé** = dérivé d'une mesure par une hypothèse
> écrite. « Sain » = paquet fermé que le juge de L0 ne contredit pas (colonne `ferme` de la carte v2).

## 0. Statut

**[x] retenue, sous une réserve à lever par le pilote : le gate de corpus rend rc = 1, instruit.**
La fin de la vue A lue (E) fixe désormais le début de la vue B aux trois sites du localisateur
(cuisson, marche des morts d'objet, marche de `killsource`), selon les décisions (1) à (3) ; la loi du
bit nul devant la tête de la vue B (lue : `FUN_142f2c3b0`) s'applique à tous les candidats de tête.

- Carte v2, 20 films, contre `87cdfa761` : **313 542 → 354 796 paquets sains (+41 254)**, records
  utiles sains **2 923 597 → 3 473 088 (+549 491)** ; D1 corpus 37,7 % → 44,4 %, HI_1_13_0 76,7 % →
  91,2 %. 752 sains perdus en brut, **tous instruits** : 354 par la fin de la vue A (la marche depuis
  E bute sur un composant non porté ou en vue C — liste par composant au §5.1), 392 par la loi du bit
  nul (débuts d'avant précédés d'un 1 : fermetures factices), 6 en cascade.
- Gate 2 : 17 films en hausse ou stables ; **3 films illisibles en baisse** (`60ae07c4` −11,
  `a349fea8` −2, `a521164d` −1 sains), uniquement par 14 fermetures factices retirées par la loi du bit
  nul, instruites paquet par paquet : **exception D2**. La fin de la vue A seule ne fait baisser aucun
  film (§4.1).
- `killsource` : **aucune mort, valeur ni voie ne change** (20 films) : `killsource.Rev` reste
  `killsource-2026-09-27`, aucun backlog du parc à demander.
- Gate de corpus : **rc = 1** — 3 `MANQUE` (les 14 fermetures factices, D2), 3 `FAUX` de V-3
  (instruits : tous les tirs « hors vie » ajoutés sont des tirs de véhicule pendant un trajet
  nouvellement lu ; le banc ne compte pas les trajets comme des vies), `[FILET]` des postures (fausse
  continuité retirée, oracle physique) ; trous et rafales instruits par famille seulement. Son
  admission (rc 1) est un geste du pilote ; la revue adversariale de fin de lot reste à faire.
- Gates de code : gofmt vide, vet (et `-tags=research`) rc=0, archlint ok, G-film rc=0 (21 paquets),
  golangci-lint 0 issue ; **13 / 13 mutations ROUGES** ; vérifications (a) à (g) de la représentation
  intermédiaire tenues.
- Révisions : `grammar-2026-10-06.2` ; `killsource`, `objectives`, `profile`, `source`,
  `SchemaVersion` constants.


## 1. Ce qui est lu dans le jeu

| Fait | Lu où | Porté dans |
|---|---|---|
| Le tick s'écrit `cfg:1 · vue A · 0 · vue B · vue C`, bout à bout : l'écrivain 0 écrit la vue A (`142f2c578 CALL FUN_142f2c050`), puis UN bit de valeur 0 (`142f2c57d XOR R8D,R8D`, `142f2c583 CALL FUN_1406d49c4` : `*(w+0x30) = *(w+0x30)*2 \| param_3`) ; la vue B est écrite dans un tampon à part (`142f2c56e CALL FUN_142f2cc78`) ; `FUN_14299d2c8` écrit le bit de configuration (`14299d333`, `14299d33e`) puis recopie les tampons au bit près (`FUN_1406d5d14`, boucle `14299d34c..14299d395`) | `va_ghidra/ASM_142f2c3b0.txt`, `ASM_14299d2c8.txt`, `FUN_1406d49c4.c` (versionnés à cette étape ; relus de la recherche LS) | en-tête de `localisateur.go` |
| Le lecteur (`FUN_142987460`) lit le bit de configuration, puis la vue A (`FUN_14076a1c4` : `{R(1) ; 0 → fin ; FUN_14080a9d4}`), puis la vue B au bit qui suit. Aucune recherche, aucune signature | `FUN_142987460`, `FUN_14076a1c4` (V1, `va_ghidra/FUN_14076a1c4.c`) | `debutParLaVueA` |
| Conséquence : le début de la vue B est le bit qui suit le terminateur de la vue A, et le bit qui précède la TÊTE de la vue B vaut 0. Devant tout autre record, le bit précédent est le dernier bit du record d'avant (records contigus, `FUN_14076b9c8`) | les deux lignes ci-dessus | `debutParLaVueA`, `precedeDuTerminateur` |
| Version de chaque genre de message : celle du film en rejeu (`film + 0xCB208`), native `DAT_14474cd90` ; règle des deux classes (ÉGALE / PRÉFIXE), décision du 2026-10-04 | `FUN_141102ed0`, `FUN_1428e1c64` (V1) | `vue_a_versions.go` (V1, inchangé) |

**Non lu, et non utilisé** : rien de neuf. Le slot 123 et le composant unique de la signature restent
des MESURES (convention de production, inchangée là où la fin de la vue A ne décide pas).

## 2. La règle de production

Pour un paquet delta dont la tête annonce une liste d'événements, la vue A déjà lue par la lecture
unique (`rangerLaTete`, V1) décide (`debutParLaVueA`, `localisateur.go`) :

| Classe de la table du film | Vue A lue jusqu'au terminateur, ≥ 1 message | Sinon |
|---|---|---|
| ÉGALE (HI_1_12_0, HI_1_13_0) | **E**, toujours (`lecture.DebutParVueA`) — même quand la signature du slot 123 trouverait une autre position, même quand la marche depuis E bute ensuite sur un composant non porté : la loi du jeu prime, revenir à la signature serait une convention (décision (1)) | chemin d'avant, à l'identique |
| PRÉFIXE (HI_1_9_0 à HI_1_11_0) | **E** seulement si la marche de la vue B depuis E ferme le paquet sans règle de l'écrivain contredite (`lectureDEssai(…).Fermee`, juge de L0, monde restauré) ; sinon chemin d'avant (décision (2)) | chemin d'avant |
| ILLISIBLE | — | chemin d'avant |

Une vue A lue en partie (genre non porté, charge refusée, bit de configuration à 0, payload épuisé)
n'est JAMAIS utilisée (décision (3)) : `debutParLaVueA` exige `Porte`.

**Les trois sites du localisateur** suivent la même règle :

- cuisson : `debutDeLaVueBDeCuisson` (`marche_trames.go`) reçoit la vue A rangée — pas de seconde
  lecture — puis `localiserLaListe` à défaut ;
- marche des morts d'objet (`marchDebut`, `object_deaths_march.go`) et marche de `killsource`
  (`walk.go`) : `DebutDeLaVueB` (`localisateur.go`) lit la vue A par la lecture unique (troisième
  appelant permis de `lireLaVueA`, garde-rail archlint mis à jour) sous la grammaire de vue A du film
  (`VueADuFilm` : `fc.grammaireDeLaVueA()` pour la marche des morts, `VueADuFilmSousCarte(f.src,
  carte)` pour `killsource`, portée par sa calibration), puis `LocaliserBoucleDeRecords` dans l'ordre
  `SignaturePuisLargeurLibre` à défaut.
- la calibration du cadre de la marche des morts (`trialFrameConfig` → `marchStartOf`) juge chaque
  largeur d'identifiant par le taux de paquets que la SIGNATURE localise : la signature dépend de
  cette largeur, la fin de la vue A n'en dépend pas et localiserait les mêmes paquets sous toutes les
  largeurs. Elle garde le localisateur seul (`VueADuFilm{}`), écrit dans la doc de `marchStartOf`.

**La loi du bit nul devant la tête (D-LSR-3, lue).** Les candidats NEW de tête de la cuisson (chaîne
`debutParChaine` et fermeture `debutParFermetureRangee`) doivent suivre un bit nul
(`precedeDuTerminateur(pay, p-extra)`) : le jeu n'écrit un 0 que devant le premier record de la vue
B. Le test du bit précédent n'a plus qu'une implantation (`precedeDuTerminateur`, `localisateur.go`),
que le localisateur strict, le repli à largeur libre et les candidats appellent (règle des deux
copies : c'était la troisième) ; garde-rail `archlint/film_localisateur_unique_test.go` étendu (le
test brut `BitAt(_, x-1)` est interdit dans `grammar` hors de lui ; l'essai de position reconnaît
l'appel). Sans effet sur le localisateur strict et le repli : ils exigeaient déjà ce bit, et leurs
positions commencent à 2.

## 3. Ce qui change dans le code

Production (`film/internal/`) :

- `grammar/localisateur.go` : en-tête réécrit (la fin de la vue A d'abord, la table des trois sites,
  la règle des classes) ; `debutParLaVueA` (la décision) ; `VueADuFilm` et `VueADuFilmSousCarte` (la
  grammaire de vue A du film pour les marches qui n'ouvrent pas de contexte) ; `DebutDeLaVueB` (les deux
  marches qui lisent les morts) ; `precedeDuTerminateur` (seule implantation du test du bit qui précède
  une position ; `marchLocateStrict` et `marchLocateFallback` l'appellent, sortie identique).
- `grammar/marche_trames.go` : `marcherLePaquet` prend la grammaire de vue A du contexte et passe par
  `debutDeLaVueBDeCuisson` (la fin de la vue A rangée, sinon `localiserLaListe`) — la fonction vit
  ici parce que le pilotage de la marche des trames ne vit que dans ce fichier (garde-rail
  `marche_trames_unique_test.go`) ; en-tête mis à jour.
- `grammar/debut_de_liste.go` : les candidats de tête de `debutParChaine` et `debutParFermetureRangee`
  suivent `precedeDuTerminateur` ; docs (le candidat, `localiserLaListe`).
- `grammar/object_deaths_march.go`, `grammar/object_deaths.go` : `marchDebut` reçoit la grammaire de
  vue A (`DebutDeLaVueB`) ; `ScanMarchFacts` passe celle du film ; `marchStartOf` (calibration)
  garde le localisateur seul, la raison écrite ; en-tête (mécanisme 2).
- `facts/killsource/calibrate.go`, `walk.go`, `decode.go` : la calibration porte `VueA`
  (`VueADuFilmSousCarte(f.src, carte)`), `runWalk` reçoit la calibration (5 paramètres) et part de
  `DebutDeLaVueB` ; en-tête de `walk.go`. Chronique : complément à révision constante.
- `grammar/frame_closure_detail.go` : `PaquetDeCarte.ListeLue` ; `ListeLocalisee` exclut
  `DebutParVueA` (vérification (c)).
- `grammar/lecture/paquet.go` : `DebutParVueA` EN QUEUE de `DebutDeVueB` (valeurs stables), doc : un
  début LU n'est pas une récupération.
- `grammar/vue_a_lecture.go`, `vue_a_versions.go`, `marche_trames_rangs.go` : docs (le troisième
  appelant de la lecture unique ; la décision par classe ; la marche par rangs part de la fin de la vue
  A lue ou d'un début localisé).
- `grammar/rev.go`, `rev_chronique.go`, `testdata/grammar_rev.golden` : `grammar-2026-10-06.2`.

Instruments et garde-rails :

- `archlint/film_localisateur_unique_test.go` : l'essai de position reconnaît `precedeDuTerminateur(_, s)` ;
  le test brut `BitAt(_, x-1)` est interdit dans `grammar` hors de `precedeDuTerminateur` ; vecteurs
  `TestGardeRailBitDeTeteVecteurs`.
- `archlint/film_vue_a_lecteur_unique_test.go` : `DebutDeLaVueB` est le troisième appelant permis de
  `lireLaVueA`.
- `research/cmd_fermeture` (carte v2) : la colonne `liste` vaut `lue` pour `DebutParVueA` ; compte
  « listes d evenements / lues (fin de la vue A) » dans `fermeture_ecrivain.tsv`.
- Recopies du pilotage (gate 5) : `cmMarcher` et `rnMarcher` partent par défaut de
  `debutDeLaListeSous` (= `debutDeLaVueBDeCuisson` sous la grammaire du film) ; la branche « sans
  localisateur fourni » de `cmPaquetDe`, devenue inatteignable, est retirée.
- Sondes `research` : `grammar/va_v2_research_test.go` (`TestVAV2Pertes`, `TestVAV2Vehicules`),
  `replaybuild/va_v2_etapes_research_test.go` (`TestVAV2Etapes`).

Tests (vecteurs d'après l'écrivain) :

- `grammar/debut_par_vue_a_test.go` (neuf) : E > S, signature fictive DANS la vue A (un Script dont la
  charge porte un bit nul puis une signature stricte) → E, film ÉGALE, cuisson et marches ; E < S, la
  marche depuis E bute sur un masque hors archétype avant une signature → E pour ÉGALE, le localisateur
  à l'identique pour PRÉFIXE ; PRÉFIXE prouvé par la fermeture → E ; vue A lue en partie (bit de
  configuration à 0, genre 85 non porté, film sans table) → le localisateur à l'identique ; vrai
  paquet `bcb6d393` 1:204 (quarante Script) → 5605 par la vue A ; bit nul devant la tête (chaîne et
  fermeture) ; marche des morts sur la bobine réelle : localise exactement ce que `DebutDeLaVueB`
  localise, et plus que le localisateur seul.
- Vecteurs corrigés selon la loi : `debut_de_liste_test.go`, `debut_de_liste_masque_test.go`
  (`0x1f` → `0x1e` : la fin d'un message PUIS son terminateur 0), `debut_par_fermeture_test.go`
  (`deuxDebuts` : un terminateur devant le premier candidat). Ces vecteurs plaçaient un bit à 1 devant
  la tête de la vue B, ce que l'écrivain n'écrit pas.
- `frame_closure_detail_test.go` : par bobine, ≥ 1 liste lue en plus de ≥ 1 localisée et ≥ 1 non
  localisée (`ks_000d5950` : 119 / 1 / 353 ; `ks_e5adf7b2` : 40 / 1 / 13).
- `marche_trames_test.go` : docs de `rangerUnAvec` et de `TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide`.
- `localisateur_test.go` : `marchDebut` à quatre paramètres.
- `killsource/testdata/minibobine.golden` : 460 / 473 → 473 / 473 paquets à événements localisés.


## 4. Table par film (carte v2, gate 2)

Carte v2 (`research/cmd_fermeture -mode v2 -denominateur-fixe`, table ECS figée de la campagne,
`-plafond-gib 4`), base `87cdfa761` (`scratchpad/cg3-V1/carte_base`, identique à l'octet à V1) contre
la tête du lot. Gate 2 par `integ2/gate2.awk`. Classes des pertes (§5) : **E** = la fin de la vue A
décide, **L** = loi du bit nul, **C** = cascade.

| Film | Build | Classe | Sains base | Sains V2 | Net | Perdus bruts | dont E / L / C | dont contredits | Gagnés | Gains au bit (factices) | Pertes au bit | Utiles sains base | Utiles sains V2 | Net utiles | Utiles sains perdus bruts |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | HI_1_13_0 | egale | 19156 | 20771 | +1615 | 4 | 4 / 0 / 0 | 0 | 1619 | 1609 (1) | 11 | 159929 | 176825 | +16896 | 29 |
| `396cfc92` | HI_1_13_0 | egale | 23131 | 25954 | +2823 | 94 | 94 / 0 / 0 | 0 | 2917 | 2911 (0) | 97 | 169502 | 196024 | +26522 | 782 |
| `4f77afc1` | HI_1_13_0 | egale | 24106 | 31004 | +6898 | 113 | 103 / 10 / 0 | 3 | 7011 | 6305 (7) | 323 | 607967 | 822338 | +214371 | 1486 |
| `51ebbc0f` | HI_1_13_0 | egale | 20444 | 26876 | +6432 | 9 | 8 / 1 / 0 | 0 | 6441 | 6364 (1) | 17 | 140632 | 197931 | +57299 | 53 |
| `bcb6d393` | HI_1_12_0 | egale | 5880 | 7119 | +1239 | 4 | 4 / 0 / 0 | 0 | 1243 | 1241 (1) | 6 | 35502 | 46309 | +10807 | 10 |
| `bf15f7ab` | HI_1_13_0 | egale | 28603 | 29913 | +1310 | 3 | 2 / 1 / 0 | 0 | 1313 | 1305 (0) | 5 | 216104 | 228132 | +12028 | 14 |
| `bfecd02b` | HI_1_13_0 | egale | 27668 | 31067 | +3399 | 7 | 7 / 0 / 0 | 0 | 3406 | 3386 (0) | 9 | 239064 | 276442 | +37378 | 44 |
| `c75f33b8` | HI_1_13_0 | egale | 23872 | 27272 | +3400 | 3 | 3 / 0 / 0 | 0 | 3403 | 3316 (0) | 7 | 153997 | 180599 | +26602 | 26 |
| `d9781168` | HI_1_13_0 | egale | 26317 | 35771 | +9454 | 66 | 66 / 0 / 0 | 0 | 9520 | 9302 (0) | 102 | 180854 | 263856 | +83002 | 524 |
| `f75e7053` | HI_1_13_0 | egale | 23673 | 24517 | +844 | 59 | 59 / 0 / 0 | 0 | 903 | 898 (0) | 62 | 162137 | 168248 | +6111 | 436 |
| `fb1a1a72` | HI_1_13_0 | egale | 43457 | 46308 | +2851 | 4 | 4 / 0 / 0 | 0 | 2855 | 2821 (0) | 9 | 325815 | 351322 | +25507 | 15 |
| `50247b26` | version-31 | illisible | 139 | 139 | +0 | 0 | 0 / 0 / 0 | 0 | 0 | 0 (0) | 4 | 274 | 274 | +0 | 0 |
| `60ae07c4` | HI_1_8_0 | illisible | 13948 | 13937 | -11 | 11 | 0 / 11 / 0 | 0 | 0 | 0 (0) | 59 | 83685 | 83673 | -12 | 12 |
| `a349fea8` | version-33 | illisible | 424 | 422 | -2 | 2 | 0 / 2 / 0 | 0 | 0 | 0 (0) | 18 | 3654 | 3654 | +0 | 0 |
| `a521164d` | HI_1_4_1 | illisible | 692 | 691 | -1 | 1 | 0 / 1 / 0 | 0 | 0 | 0 (0) | 5 | 78 | 78 | +0 | 0 |
| `084a804d` | HI_1_10_0 | prefixe | 4837 | 4959 | +122 | 33 | 0 / 33 / 0 | 2 | 155 | 117 (2) | 199 | 89755 | 93723 | +3968 | 57 |
| `111fa685` | HI_1_10_0 | prefixe | 4026 | 4077 | +51 | 13 | 0 / 13 / 0 | 1 | 64 | 59 (0) | 62 | 42896 | 44261 | +1365 | 37 |
| `11de8353` | HI_1_9_0 | prefixe | 5631 | 5760 | +129 | 8 | 0 / 8 / 0 | 0 | 137 | 121 (0) | 48 | 65658 | 68561 | +2903 | 1 |
| `1c4c63c2` | HI_1_10_0 | prefixe | 13389 | 13940 | +551 | 307 | 0 / 307 / 0 | 75 | 858 | 529 (17) | 1774 | 165987 | 187027 | +21040 | 369 |
| `e5adf7b2` | HI_1_11_0 | prefixe | 4149 | 4299 | +150 | 11 | 0 / 5 / 6 | 0 | 161 | 144 (0) | 55 | 80107 | 83811 | +3704 | 205 |
| **corpus** | | | 313542 | 354796 | +41254 | 752 | 354 / 392 / 6 | 81 | 42006 | 40428 (29) | 2872 | 2923597 | 3473088 | +549491 | 4100 |


Lecture du gate 2 (« aucun film en baisse, NET par film, paquets sains ET records utiles sains ») :

- **17 films en hausse ou stables** sur les deux comptes.
- **3 films en baisse nette, tous illisibles** (la fin de la vue A n'y joue pas) : `60ae07c4`
  (HI_1_8_0) −11 sains, −12 utiles sains ; `a349fea8` (version-33) −2 sains, 0 utile ; `a521164d`
  (HI_1_4_1) −1 sain, 0 utile. **Les 14 pertes sont toutes de classe L** : la cuisson d'avant ouvrait
  la liste à un candidat NEW de tête (rang « fermeture ») précédé d'un bit à 1 — une position que
  l'écrivain ne peut pas avoir écrite comme tête de la vue B (§1). Ce sont des fermetures FACTICES que
  le juge de L0 ne voyait pas (il ne connaît pas la loi de la tête) : **exception D2 (fermeture
  factice retirée, instruite)**, paquet par paquet ci-dessous. Sans autre candidat écrivable, ces
  listes ne sont plus localisées : rien n'est deviné.

| Film | Paquet | Début d'avant (S0, rang) | Bit avant S0 | V2 |
|---|---|---|---|---|
| `60ae07c4` | 23:2160 | 579 (fermeture) | 1 | non localisee |
| `60ae07c4` | 30:1354 | 1227 (fermeture) | 1 | non localisee |
| `60ae07c4` | 30:1894 | 1400 (fermeture) | 1 | non localisee |
| `60ae07c4` | 31:720 | 452 (fermeture) | 1 | non localisee |
| `60ae07c4` | 32:700 | 3401 (fermeture) | 1 | non localisee |
| `60ae07c4` | 33:2236 | 771 (fermeture) | 1 | non localisee |
| `60ae07c4` | 37:2208 | 1470 (fermeture) | 1 | non localisee |
| `60ae07c4` | 37:34 | 1433 (fermeture) | 1 | non localisee |
| `60ae07c4` | 38:1090 | 1501 (fermeture) | 1 | non localisee |
| `60ae07c4` | 41:538 | 852 (fermeture) | 1 | non localisee |
| `60ae07c4` | 9:2256 | 2690 (fermeture) | 1 | non localisee |
| `a349fea8` | 10:932 | 11054 (fermeture) | 1 | non localisee |
| `a349fea8` | 31:22 | 11390 (fermeture) | 1 | non localisee |
| `a521164d` | 1:726 | 9082 (fermeture) | 1 | non localisee |


- Part factice des gains au bit : 29 / 40 428 (0,1 %).
- Listes : sur les films ÉGALE, les listes non localisées passent de 15 949 à 1 137 et les listes
  lues jusqu'à la fin de la vue A sont 61 650 (dont 52 142 saines) ; sur les films PRÉFIXE, 5 652
  listes sont prouvées par la fermeture depuis E (toutes saines, par construction).

D1 (part fixe, dénominateur fixe recalculé : le fixe monte quand une marche lit plus loin) : corpus
37,7 % → **44,4 %** (fixe 7 758 290 → 7 822 733) ; HI_1_13_0 76,7 % → **91,2 %** ; HI_1_12_0 24,0 % →
31,3 % ; HI_1_11_0 20,9 % → 21,9 % ; HI_1_10_0 12,3 % → 13,4 % ; HI_1_9_0 20,2 % → 21,1 % ; HI_1_8_0,
HI_1_4_1, version-31, version-33 inchangés à la décimale.

### 4.1 Décomposition par règle

Deux cartes de plus, chacune avec UNE des deux règles neuves retirée par la surcouche de sa mutation
(`decomp.sh` : `carte_loi` = surcouche m01, la fin de la vue A ignorée ; `carte_E` = surcouche m04, le
bit nul retiré des candidats), même base, même gate 2 :

| Variante | Sains | Net | Perdus bruts | Gagnés | Factices / gains au bit | Utiles sains (net) | Films en baisse |
|---|---|---|---|---|---|---|---|
| fin de la vue A seule | 355 198 | +41 656 | 354 (tous classe E) | 42 010 | 25 / 40 428 | 3 474 013 (+550 416) | **0** |
| loi du bit nul seule | 312 835 | −707 | 844 | 137 | 17 / 154 | 2 916 306 (−7 291) | 18 |
| **V2 (les deux)** | **354 796** | **+41 254** | **752** | **42 006** | **29 / 40 428** | **3 473 088 (+549 491)** | **3** (D2) |

Lecture : tout le gain vient de la fin de la vue A, qui ne fait baisser aucun film. La loi du bit nul,
lue dans le jeu et donc appliquée (consigne), ne fait que RETIRER des débuts que l'écrivain ne peut
pas avoir écrits ; seule, elle retire 844 sains et en gagne 137 ; avec la fin de la vue A, ses
pertes propres sont les 392 de la classe L, sur les films PRÉFIXE non prouvés, quelques listes des
films ÉGALE dont la vue A ne se lit pas jusqu'au bout, et les films illisibles. Elle coûte 402 sains au corpus contre la fin de la vue A seule, et
fait baisser trois films illisibles (14 sains, §4). Ce ne sont pas des pertes de lecture mais des
fermetures factices retirées (le juge de L0 ne connaît pas la loi de la tête) : exception D2.


## 5. Les 752 sains perdus, instruits paquet par paquet

Sonde `va_v2_research_test.go` (`TestVAV2Pertes`, tag `research`) : la marche de la carte v2
(`rnMarcher`, qui suit la cuisson du lot) ; pour chaque perdu, sur le monde d'avant lui, la vue A lue,
le début V2 et comment, le début que la règle d'AVANT rendrait sur ce même monde (S0, sans fin de vue
A ni bit nul des candidats) et son bit précédent, les verdicts des marches d'essai, le point d'arrêt
de la marche depuis le début V2 ; puis le verdict que la marche rend au paquet. **Contrôle croisé** :
752 / 752 perdus instruits ; le verdict et la cause rendus par la sonde sont ceux de la carte sur les
752 (0 désaccord). S0 est le début de la base (sonde V1, monde de la base) pour 747 / 752 ; les 5
autres sont des mondes différents (cascade : 4 sur `e5adf7b2`, 1 sur `4f77afc1`, où V2 part de E
quoi qu'il en soit).

| Classe | Perdus | Utiles sains perdus | Instruction |
|---|---|---|---|
| **E** — la fin de la vue A décide (films ÉGALE) | 354 | 3 328 | la marche depuis E ne ferme pas ; depuis S0 elle fermait (monde de la tête). E < S0 dans les 354. La marche depuis E s'arrête AVANT S0 dans 350 cas (a4 : la loi dit E, elle ne dit pas si S0 est une frontière de record) et TRAVERSE S0 dans 4 (a3 : S0 tombe dans un record lu depuis E, contredit). S0 était lui-même précédé d'un bit à 1 (début contredit par la loi) dans 24 cas. **Elles ne se réparent pas en revenant à la signature** (décision (1)) : la cure est la grammaire des records de tête de la vue B (VB-têtes) |
| **L** — loi du bit nul | 392 | 572 | le début d'avant S0 était, dans les 392 cas, un candidat NEW pris au rang « fermeture » et précédé d'un bit à 1 : fermeture factice retirée. V2 : liste non localisée (311) ou prise au repli « fermé au bit » (81) |
| **C** — cascade | 6 | 200 | `e5adf7b2` 10:1010, 1016, 1018, 1020, 1022, 1024 : même début en base et en V2 ; sortie de la vue B par rejet du slot 1096 (`ti=41`), que la base liait par le NEW lu à 3 383 dans 10:848 — début au repli « fermé au bit », contredit, précédé d'un bit à 1 (sonde, cible ajoutée). V2 retire ce début (loi L) : le slot n'est plus lié. Même mécanisme que R1 §6 (D-R1-5) : des sains de la base reposaient sur une liaison posée à un début que la loi de l'écrivain contredit |

### 5.1 Les pertes E, par composant (pour les lots suivants)

Cause du premier arrêt de la marche depuis E (colonne `cause` de la carte) :

| Composant / arrêt | Perdus | Films |
|---|---|---|
| `ti=43 i19 device-position-animation-name-component` | 139 | `f75e7053` 43, `4f77afc1` 11, `396cfc92` 85 |
| `ti=12 i16 managed-navpoint-override-flags` | 61 | `f75e7053` 12, `bfecd02b` 4, `4f77afc1` 39, `396cfc92` 4, `0797ce72` 1, `fb1a1a72` 1 |
| `vue C : terminateur hors cadre` | 44 | `bcb6d393` 1, `4f77afc1` 4, `396cfc92` 1, `d9781168` 37, `fb1a1a72` 1 |
| `vue B : sortie par rejet` | 37 | `4f77afc1` 31, `51ebbc0f` 1, `d9781168` 5 |
| `ti=10 i2 managed-object-navpoint-component` | 19 | `bcb6d393` 1, `4f77afc1` 3, `51ebbc0f` 5, `d9781168` 10 |
| `ti=45 i0 matchflow-sequence-data-component` | 19 | `bcb6d393` 1, `f75e7053` 1, `bfecd02b` 3, `c75f33b8` 3, `bf15f7ab` 2, `51ebbc0f` 1, `396cfc92` 4, `0797ce72` 1, `d9781168` 2, `fb1a1a72` 1 |
| `vue C : bloc 0xbc (desalignement)` | 12 | `d9781168` 12 |
| `ti=12 i18 managed-navpoint-position-offset` | 7 | `f75e7053` 3, `4f77afc1` 1, `51ebbc0f` 1, `0797ce72` 1, `fb1a1a72` 1 |
| `ti=43 i21 device-position-group-component` | 6 | `4f77afc1` 6 |
| `ti=10 i22 managed-object-interaction-filter-component` | 3 | `4f77afc1` 3 |
| `ti=35 i59 biped-spartan-ability-non-predicted-state` | 2 | `4f77afc1` 2 |
| `ti=10 i18 managed-object-networked-property-component` | 1 | `4f77afc1` 1 |
| `ti=12 i22 managed-navpoint-visual-state-groups-component-2` | 1 | `4f77afc1` 1 |
| `ti=43 i35 device-animation-layer-state-component` | 1 | `bcb6d393` 1 |
| `ti=43 i39 device-machine-flags-component` | 1 | `4f77afc1` 1 |
| `ti=45 i1 matchflow-focus-data-component` | 1 | `0797ce72` 1 |


Lecture : `ti=43 i19 device-position-animation-name` (139, dont 85 sur `396cfc92` et 43 sur
`f75e7053`) et `ti=12 i16 managed-navpoint-override-flags` (61, dont 39 sur `4f77afc1`) font 56 % des
pertes E, comme la vérification adverse du plan l'annonçait ; les arrêts en vue C (56) et les sorties
de vue B par rejet (37, slot inconnu du monde) suivent. `ti=10 i2`, `ti=45 i0`, `ti=12 i18`,
`ti=43 i21` complètent. Ce sont des composants non portés ou des records de tête non traversés, pas
des fautes de la vue A.

## 6. `killsource` (gate 3) et la marche des morts

**`cmd/killsource json`**, 20 films (19 témoins + `1c4c63c2`, carte Refuge), binaire de `87cdfa761`
contre binaire du lot : **19 JSON identiques à l'octet** ; sur `c75f33b8`, une seule ligne diffère,
`concordance.enregistrements_lus_par_les_deux_voies` 5 → 6 (`Stats.Redundant`, diagnostic non
persisté). **Aucune mort, aucune valeur, aucune voie ne change** (les 61 lignes `voie` du JSON de
`c75f33b8` sont identiques). `replay-equiv` le confirme sur ses 20 films : l'étape `killRefs` (les
références de kill tirées du décodage) est identique partout ; l'étape `killsource` diverge sur 16
films parce qu'elle hache tout le `Result` : sur la valeur de l'étape (sonde des étapes, 7 films),
seuls `Stats.PacketsLocated` et les comptes de replis (`Replis.LocalisationsALargeurLibre`,
`Replis.IndicesHorsRoster`) changent ; aucun n'est persisté dans `match_kill_events`.

**`killsource.Rev` reste `killsource-2026-09-27`** (D23 : aucune sortie persistée ne change) :
complément de chronique, golden régénéré à révision constante. **Aucun backlog du parc n'est ouvert.**
La mini-bobine de `killsource` (`testdata/minibobine.golden`) passe de 460 / 473 à 473 / 473 paquets à
événements localisés (ligne de diagnostic, régénérée).

**Marche des morts d'objet** (vérification (d)) : la marche à huit vues (`ScanMarchFacts`, `object_deaths_march.go`) part de la fin de la vue A
quand elle décide (V2 explicite). `replay-equiv` : étape `vehicles` divergente sur 6 films
(`084a804d`, `111fa685`, `11de8353`, `1c4c63c2`, `53ce4390`, `e5adf7b2`), identique sur les 14 autres.
Instruction par la valeur de l'étape (sonde `replaybuild/va_v2_etapes_research_test.go`, cuisson du
harnais sous la même racine factice, binaires de `87cdfa761` et du lot ; contrôle : `d9781168`
identique) — seuls trois champs changent, `Deaths`, `Occupancy` et `DeathStats`, et **uniquement par
ajout** (aucune lecture de la base ne disparaît) :

| Film | Morts de véhicule (base → V2) | Lectures d'occupation (base → V2) | Retirées |
|---|---|---|---|
| `084a804d` | 35 → 35 | 113 → 127 (+14) | 0 |
| `111fa685` | 4 → 4 | 8 → 9 (+1) | 0 |
| `11de8353` | 15 → **16** (+1 : slot 783 `ti=40`, `Mort`, `t = 2354459518` µs) | 26 → 34 (+8) | 0 |
| `1c4c63c2` | 4 → 4 | 24 → 47 (+23) | 0 |
| `53ce4390` | 3 → 3 | 29 → 60 (+31) | 0 |
| `e5adf7b2` | 17 → 17 | 63 → 75 (+12) | 0 |

Ce sont des records lus dans des paquets que la marche des morts n'atteignait pas (paquets à
événements localisés désormais par la fin de leur vue A). Les morts BIPÈDES et la sortie
`killsource` ne changent pas (§ ci-dessus) ; l'occupation nourrit les trajets du rejeu (gate de
corpus, §7) : `repli_identite_premier_occupant_du_siege` 642 → 670 sur les 6 films.


**Registre des replis** (journal « repli déclenché » de `replay-equiv`, 20 films) :

| Repli | Films changés | Base → V2 |
|---|---|---|
| `repli_localisation_largeur_libre` | 16 | 25 686 → **19 972** sur les 20 films (−5 714) ; sur les films ÉGALE il tombe à 2–21 par film (`d9781168` 838 → 21, `bcb6d393` 189 → 3) ; inchangé sur les 4 films illisibles |
| `repli_debut_de_liste_ferme_au_bit` | 20 | 8 047 → 5 240 |
| `repli_liaison_par_anticipation` | 18 | 6 592 → 6 185 |
| `repli_identite_premier_occupant_du_siege` | 6 | 642 → 670 |
| `repli_episode_occupation_par_trou_de_position` | 2 | 66 → 36 |
| `repli_episode_borne_par_la_vie_suivante` | 2 | 47 → 27 |
| `repli_deadstate_hors_bande_bipede` | 4 | 25 → 32 (dead-states lus en plus par la marche de killsource et écartés par le filtre de crédibilité ; aucune mort publiée ne change) |
| `repli_deadstate_indice_hors_roster` | 2 | 60 → 62 (idem) |

## 7. Gates (sorties exactes)

Environnement : `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg3-vuea`, CGO msys64 ucrt64, une
commande go à la fois ; films lus en place (`data/cache/film_chunks`, lecture seule) ; racine factice
et copie du parc au scratchpad (`scratchpad/cg3-V1/repo`, `scratchpad/cg3-V1/parc`). `git fetch` avant
les gates : `origin/feat/v75` = `87cdfa761`, pas d'avance, aucune fusion.

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0. `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc=0
   (`go vet -tags=research ./internal/replaybuild/` : rc=0).
3. `go test ./internal/archlint/ -count=1` : `ok levelup/go-api/internal/archlint 45.248s`. Un premier
   passage était ROUGE : `killsource/rev_chronique.go` à 503 lignes (seuil 500) ; rotation ordinaire du
   rang `killsource-2026-09-18` vers `rev_chronique_archive.go`, tel quel (477 lignes), puis VERT.
4. G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` :
   rc=0, 21 paquets `ok` (`scratchpad/cg3-V2/gfilm.log`). Garde-fou des recopies du pilotage :
   `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` vert, ≥ 1 liste lue par bobine.
5. `golangci-lint run` (`GOLANGCI_LINT_CACHE` au scratchpad) sur `film/internal/grammar/...`,
   `film/internal/facts/...`, `film/research/cmd_fermeture/...`, `internal/archlint/...`,
   `internal/replaybuild/...` (toutes les issues, tests compris) : `0 issues.` Sous
   `--build-tags=research` (mêmes paquets hors archlint) : 147 issues, toutes antérieures
   (`--new-from-rev=3bacfadeb` : `0 issues.` ; errcheck 108 de `cmd_fermeture`, goconst 30, …).
6. Carte v2, 20 films, `-plafond-gib 4` : rc=0, 20 films, 0 échec. Gate 2 : §4.
7. `cmd/killsource json`, 20 films : rc=0 partout ; 19 JSON identiques à l'octet, `c75f33b8` : une
   ligne (§6).
8. `replay-equiv` (20 films, racine factice) : « BILAN : 0 identique(s), 20 different(s) » des deux
   côtés contre les références non re-figées. Base contre tête, étape par étape : divergent
   `artifact` (20 films), `movementStates.stats` (20), `continuousFire.stats` (20), `movementStates`
   (17), `killsource` (16, compteurs non persistés, §6), `continuousFire` (13), `vehicles` (6, §6) ;
   **identiques** : `objectives`, `killRefs`, `deaths`, et toutes les autres étapes. Fixtures de contrat
   web régénérées : les 8 JSON identiques à `3bacfadeb` une fois les chaînes `grammar-…` neutralisées.
9. `replay-corpus-gate --reference=base --base=87cdfa761 --parc-root <copie> --work-root <scratchpad>` :
   **rc=1**. Tableau :

```
temoin       famille          base(87cdfa761)   HEAD    gains   pertes   chang.      duree base   cache  statut
bcb6d393     ctf_mono_manche      78     78       23       18        3      5.12s cache  A+F    PERTE
fb1a1a72     ctf_multi_manche     78     78       48       25        3     12.05s cache  A+F    PERTE
d9781168     oddball              78     78       66       75        4     10.64s cache  A+F    PERTE
c75f33b8     assaut_bombe         78     78       44       44        4      7.72s cache  A+F    PERTE
bf15f7ab     slayer               78     78       27       21        3      6.74s cache  A+F    PERTE
51ebbc0f     deux_manches         78     78       49       35        3      6.59s cache  A+F    PERTE
084a804d     vehicules            78     78      118        4       40    1m21.2s cache  A+F    FAUX
0797ce72     region_index_2_bits     78     78       30       13        3      8.14s cache  A+F    PERTE
111fa685     version_39           78     78       98       19        6     26.34s cache  A+F    FAUX
e5adf7b2     version_40_build_1_11     78     78       81       40        7     28.57s cache  A+F    PERTE
60ae07c4     version_37           78     78        9        9        0      18.3s cache  A+F    MANQUE
a349fea8     version_33_sans_identification     78     78        5        5        0   1m26.15s cache  A+F    MANQUE
a521164d     version_33_build_1_4_1     78     78        3        6        0      20.4s cache  A+F    MANQUE
11de8353     version_38_build_1_9_0     78     78       76        9       30     20.15s cache  A+F    PERTE
50247b26     version_31_sans_identification     78     78        2        3        0     37.66s cache  A+F    PERTE
bfecd02b     vehicules_v41_utilisateur     78     78       67       25        6      9.08s cache  A+F    PERTE
4f77afc1     equipement_origine_utilisateur     78     78      276      112       17     45.29s cache  A+F    FAUX
396cfc92     strongholds_zones     78     78       33       17        3     10.03s cache  A+F    PERTE
f75e7053     koth_collines        78     78       35       14        2      5.38s cache  A+F    PERTE
```

   Banc de vérité : **13 / 19 « ok »**, 3 « MANQUE », 3 « FAUX ».
   - `MANQUE` (`60ae07c4`, `a349fea8`, `a521164d`) : « P-1 paquets fermés au bit près » 13 948 → 13 937,
     424 → 422, 692 → 691 : les 14 pertes L du §4, fermetures factices retirées (D2, instruites paquet
     par paquet).
   - `FAUX` (`4f77afc1` V-3 112 → 263, `084a804d` 92 → 119, `111fa685` 5 → 15) : V-3 « action hors
     vie ». Instruit sur les artefacts du gate (base : cache du gate sous la copie du parc ; tête :
     `gate_work/data/cache/replays`), script `scratchpad/cg3-V2/v3.jq` : les instances « tir » de V-3
     passent de 101 à 252, 78 à 105, 0 à 10 ; **toutes, en base comme en V2, sont des tirs de véhicule
     (champ `v`) couverts par un trajet du même slot dans ce même véhicule**. Exemple : `4f77afc1`,
     slot 521 (xuid 2535469763661810) — piste bipède [1, 266], puis trajet NEUF conducteur du Wraith
     784 de 267 à 1053 (aim continu), le Wraith détruit à 1053, vie suivante du même xuid au slot 555
     à 1154 ; ses 15 tirs (348 … 1021, `v = 784`) sont dans ce trajet. Le banc prend les vies dans les
     seules pistes bipèdes (`replayverite.indexerVies`), qui s'arrêtent à la montée (D-VAV2-3). Trajets
     neufs (`trajets.jq`) : `4f77afc1` 123 (dont 119 qui commencent dans la vie bipède du slot ou à
     ≤ 5 images de sa fin, 4 après un écart de plus de 5 images : 3 de 6 à 30, 1 au-delà), `084a804d` 11, `111fa685` 1 (tous
     contigus) ; trajets de la base disparus ou redécoupés : 58, 8, 0.
   - `[FILET]` : 491 lignes, surtout des durées de postures en baisse. Oracle physique de R3
     (`va-r3/oracle_corpus.js`, adapté aux artefacts de cette étape : `scratchpad/cg3-V2/oracle_corpus.js`,
     `postures_corpus.tsv`) : sprint 479 469 → 453 477 images (−25 992 ; 37 087 retirées, 11 073
     ajoutées), escalade 39 191 → 29 854 (−9 337), saut dérivé 79 412 → 80 531 ; les images de sprint
     retirées ont la vitesse horizontale médiane des images HORS posture (≈ 1,9 à 2,3 contre ≈ 2,8 en
     sprint commun, 15 films), les images d'escalade retirées une vitesse verticale médiane nulle, et
     l'escalade la plus longue tombe de 1 960 images à 8 (`c75f33b8`) : **fausse continuité retirée**,
     comme R3 l'a établi pour LN. Trous du tir continu (`holesOpenViewB`, `holesUnlocated`, …) et
     rafales (`burstsRead` `084a804d` 126 → 124, `111fa685` 32 → 29, deux films PRÉFIXE où seule la
     loi L retire des débuts) : reclassements et pertes instruits par famille, **pas paquet par
     paquet**.
   - Télémétrie : `grammarRev` `grammar-2026-10-03.5 -> grammar-2026-10-06.2`, `profileRev`
     `profile-2026-09-17.3 -> profile-2026-10-06` (19 témoins).
10. Mutations : §8.


## 8. Mutations (`-overlay`, suite entière du paquet ; ROUGE attendu)

Script `scratchpad/cg3-V2/mutations.sh` (surcouches `mut/*/overlay.json`, générées par `mkmut.sh`), résultat `mutations.txt` : **13 / 13 ROUGES**, chacune sur le test attendu.

| Mutation | Verdict | Tests en échec |
|---|---|---|
| m01_E_ignore | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestLaMarcheDesMortsPartDeLaFinDeLaVueA, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m02_signature_reprise_apres_arret | ROUGE | TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestLaMarcheDesMortsPartDeLaFinDeLaVueA, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m03_preuve_retiree_films_anciens | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestFrameClosureRatchet, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite |
| m04_bit_nul_retire | ROUGE | TestLaTeteDeLaVueBSuitUnBitNul |
| m05_E_sur_lecture_arretee | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestFrameClosureRatchet, TestLaFermetureDeLaStructureNeDescendPas, TestScanObjectDeathsSurBobineReelle, TestUneVueALueEnPartieNeDecideRien |
| m06_E_malgre_configuration_a_0 | ROUGE | TestLeBitDeConfigurationAZeroArreteLaVueA, TestUneVueALueEnPartieNeDecideRien |
| m07_signature_avant_E | ROUGE | TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet |
| m08_killsource_sans_E | ROUGE | TestGoldenMiniBobine |
| m09_marche_des_morts_sans_E | ROUGE | TestLaMarcheDesMortsPartDeLaFinDeLaVueA |
| m10_liste_lue_comptee_localisee | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure |
| m11_cuisson_sans_E | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m12_test_brut_du_bit_precedent (sur disque) | ROUGE | TestLocalisateurDeBoucleUnique |
| m13_vue_A_lue_hors_des_appelants_permis (sur disque) | ROUGE | TestLectureDeLaVueAUnique |

Règles visées : m01 E ignoré ; m02 signature reprise quand la marche depuis E bute (film ÉGALE) ;
m03 preuve de fermeture retirée pour les films PRÉFIXE (la première surcouche ne compilait pas,
cas de `switch` dupliqué : régénérée et rejouée seule, ROUGE) ; m04 bit nul retiré des candidats de
tête ; m05 E pris sur une lecture arrêtée ; m06 E pris malgré le bit de configuration à 0 ; m07
signature essayée avant E ; m08 killsource sans E ; m09 marche des morts sans E ; m10 liste lue
comptée comme localisée ; m11 cuisson sans E ; m12 test brut du bit précédent hors de
`precedeDuTerminateur` (garde-rail sur disque, fichier temporaire retiré) ; m13 la vue A lue hors des
appelants permis (idem).


## 9. Vérifications de la représentation intermédiaire (session levelup-57)

| | Vérification | Statut | Preuve |
|---|---|---|---|
| (a) | Tête donnée aux canaux (`VueA.Etat`, genres, continuation) IDENTIQUE paquet par paquet | [x] | `rangerLaTete`, `teteDe`, `listeAnnoncee` et la passe des têtes ne changent pas. Sonde `TestVAV1Tete` rejouée sur le code de V2 (20 films, 629 142 trames delta de la marche de cuisson, plus la passe des têtes sur un contexte neuf) : 0 écart de tête, de route, d'étendue rangée, de passe ; `va_v1_tete.tsv` identique à celui de V1 (`diff` vide) |
| (b) | Vue A lue UNE fois ; étendues de `lecture.Paquet` inchangées hors genres portés ; préambule hors rangs sans changement | [x] | En cuisson, `debutDeLaVueBDeCuisson` reçoit la vue A déjà lue par `rangerLaTete` ; la preuve PRÉFIXE (`lectureDEssai`) part de E, au rang 1, sans relire la vue A. `p.VueA` rangée inchangée (sonde (a)). Les deux marches qui lisent les morts n'ont pas de structure : elles lisent la vue A une fois par paquet, par la lecture unique (`DebutDeLaVueB`, troisième appelant permis, garde-rail archlint mis à jour, mutation m13 ROUGE). Préambule `DecodeFrameViewsCurseur` et `candidatsDeTete` (balayage depuis p = 0) : non touchés ; seuls leurs CONSOMMATEURS de candidats appliquent la loi du bit nul |
| (c) | `DebutParVueA` dans `lecture.DebutDeVueB`, ADR 0037 IR-6 amendé ; la fin de la vue A est une LECTURE, `frame_closure_detail` ne la compte pas comme localisée | [x] | `DebutParVueA` en queue de l'énumération (valeurs existantes stables), doc « lecture, pas récupération » ; `PaquetDeCarte.ListeLue`, `ListeLocalisee` l'exclut (mutation m10 ROUGE) ; ADR 0037 : IR-3 (dernière phrase) et IR-6 (paragraphe neuf, EN) |
| (d) | Marche à huit vues de production : morts, véhicules, killsource inchangés sauf ce que V2 change et instruit ; `repli_localisation_largeur_libre` avant / après | [x] | Il n'y a pas de canal des morts (2.7.a non fusionné). `killsource` : aucune ligne persistée ne change (§6). Morts bipèdes (`deaths`) identiques ; véhicules : +1 mort et +89 lectures d'occupation, toutes ajoutées, aucune retirée (§6). `repli_localisation_largeur_libre` : **25 686 → 19 972** sur 20 films (il baisse là où E localise ; inchangé sur les 4 films illisibles) |
| (e) | `film_context.go` à 500 lignes | [x] | Non touché (500 lignes) ; la grammaire de vue A du contexte est lue par `fc.grammaireDeLaVueA()` (V1) |
| (f) | `consumeVueA` supprimé ; préambule gardé et écrit | [x] | Supprimé en V1, ratchet intact ; préambule inchangé (V1 §5) |
| (g) | `TestLaMarcheRangeUneVueANonPortee` et `distribuer_tetes_test.go` : genre 5 porté, raison écrite | [x] | Fait en V1 ; en V2 la doc du test renommé (`TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide`) et de `rangerUnAvec` dit que la marche partie de la tête ne traverse pas une vue A non vide et que la vue B d'un paquet à événements part de `debutDeLaVueBDeCuisson` ; `distribuer_tetes_test.go` non touché |

Fichiers de la RI touchés à cette étape (à soumettre à levelup-57 avant fusion) :
`lecture/paquet.go` (`DebutParVueA` + docs), `marche_trames.go` (`debutDeLaVueBDeCuisson`, appel, en-tête),
`marche_trames_rangs.go` (doc), `marche_trames_test.go` (docs), `frame_closure_detail.go`
(`ListeLue`), `frame_closure_detail_test.go`, `docs/adr/0037-film-intermediate-representation.md`
(IR-3, IR-6). NON touchés : `movement_states.go` (d'où l'écart du compteur, §11), le cœur et la doc
de `distribuer.go`, `distribuer_tetes.go`, `film_context.go`.


## 10. Révisions

- `grammar.Rev` : `grammar-2026-10-06` (V1) → **`grammar-2026-10-06.2`** (sortie changée ; rang suivant
  sans trou, aucune autre branche locale ne porte la valeur : `feat/ri-etape2` `grammar-2026-10-05`,
  `feat/cg2-ln` `grammar-2026-10-04`, `feat/cg2-lt` `.03.4`, `feat/cg2-ls` `.03.3`). Chronique écrite.
- `killsource.Rev` : **constante** `killsource-2026-09-27` (§6, D23).
- `objectives.Rev` : **constante** `objectives-2026-09-27` (étape `objectives` de `replay-equiv` identique sur les 20 films) ; complément dans `objectives/rev.go`, golden régénéré.
- `profile.Rev` (`profile-2026-10-06`), `source.Rev`, `replay.SchemaVersion` (78) : inchangés. Le
  contenu cuit change (états de mouvement, tir continu, véhicules) ; précédent de la campagne (vague 2,
  L3a, L4a) : `SchemaVersion` reste, la recuisson suit la montée de `grammar.Rev` (geste de
  l'utilisateur, D7).
- Régénérés par les commandes du dépôt : `grammar_rev.golden`, `killsource_rev.golden`,
  `objectives_rev.golden`, `types/testdata/shapes.golden` (ligne des révisions), fixtures web
  `replay_schema_78_*` et `manifest.json` (chaînes de révision seules).

## 11. Écarts

- **`movement_states.go` non touché (consigne), d'où un compteur dont le sens s'élargit.**
  `movementStateScanner.Trame` compte `EventPacketsNewRecordStart` pour tout début autre que
  `DebutParSignature` : les listes parties de la fin de la vue A y entrent désormais, alors que sa doc
  (`types.MovementStateStats`) dit « liste qui commence à un record NEW de tête ». Le compteur est
  transporté par les faits (`replay/filmfacts_*`), il n'est pas publié dans le document. Correction
  d'une ligne (exclure `DebutParVueA`, comme LN le faisait) proposée à la représentation
  intermédiaire (levelup-57), propriétaire du fichier ; non faite ici.
- **La carte v2 (instrument du gate 2) ne pose pas la carte du match** (`grammar.ContexteDeFilm`) :
  les positions à index des genres 5 et 6 y arrêtent la vue A (rien n'est deviné), là où la cuisson de
  production, qui pose la carte, la lit jusqu'au bout et part de E. Le gate 2 sous-estime donc V2 sur
  ces paquets ; `killsource json`, `replay-equiv` et le gate de corpus tournent avec la carte. Non
  mesuré séparément.
- **`killsource.Rev` ne monte pas**, contre l'attente de la consigne (« monte si une mort, une valeur
  ou une voie change ») : aucune ne change (§6). Aucun backlog du parc n'est à demander.
- **Gain et pertes plus grands que l'estimation du plan** (+41 254 sains contre +27 000 composés ;
  354 pertes E contre ~291) : le plan composait le crochet de LN (E seulement là où la signature
  échouait) ; V2 fait primer E partout (décision (1)), prouve E sur les films PRÉFIXE (décision (2))
  et applique la loi du bit nul aux candidats de tête.
- **Le gate de corpus rend rc = 1** (§7) : 3 `MANQUE` (les 14 pertes L des films illisibles, exception
  D2 instruite paquet par paquet), 3 `FAUX` de la règle V-3 (instruits : tous les tirs « hors vie »
  ajoutés sont des tirs de véhicule couverts par un trajet du même slot dans ce véhicule, trajets lus
  dans les paquets que la fin de la vue A ouvre ; le banc prend les vies dans les pistes bipèdes, qui
  s'arrêtent à la montée), et des pertes `[FILET]` (postures : fausse continuité retirée, oracle
  physique de R3 ; trous et rafales reclassés, instruits par famille, pas paquet par paquet).
  L'admission d'un rc 1 est un geste du pilote (plan §10 ; précédent : vague 2, `87cdfa761`).
- **Sondes de mondes différents** : les classes a3 / a4 et S0 sont prises sur le monde de la tête ;
  S0 est le début de la base pour 747 des 752 pertes (sonde V1, monde de la base).
- **Revue adversariale de fin de lot** (plan §8 : lot de localisation) : non faite dans cette étape ;
  à lancer par le pilote avant fusion.


## 12. Découvertes

- **D-VAV2-1** — `EventPacketsNewRecordStart` compte les listes lues jusqu'à la fin de la vue A
  (§11) ; à corriger dans `movement_states.go` par la RI.
- **D-VAV2-2** — La carte v2 ne pose pas la carte du match : un instrument qui mesure la cuisson
  devrait la poser (`-catalogue`/cartes), base et tête ensemble.
- **D-VAV2-3** — Le banc de vérité prend les vies dans les seules pistes bipèdes (`indexerVies`) :
  un joueur en véhicule n'y a pas de vie, et chaque tir de véhicule lu pendant un trajet est compté
  « hors vie » (V-3 : 101 instances déjà en base sur `4f77afc1`, 78 sur `084a804d`, toutes couvertes
  par un trajet). Plus la lecture des trajets progresse, plus V-3 monte à tort. Le banc devrait
  couvrir les vies par les trajets.
- **D-VAV2-4** — Les pertes E (354) désignent les records de tête de la vue B que le décodeur ne
  traverse pas encore : `ti=43 i19 device-position-animation-name` (139), `ti=12 i16
  managed-navpoint-override-flags` (61), arrêts en vue C (56), rejets de slot inconnu (37),
  `ti=10 i2` (19), `ti=45 i0` (19), `ti=12 i18` (7), `ti=43 i21` (6) — liste pour le lot VB-têtes.
- **D-VAV2-5** — Sur les films ÉGALE, les paquets SANS liste d'événements gagnent aussi : sains
  227 749 → 250 498 (+22 749), par les records NEW désormais lus en tête de liste (liaisons du monde).
- **D-VAV2-6** — L'étape `killsource` de `replay-equiv` hache tout le `Result`, compteurs de
  diagnostic compris (`Stats.PacketsLocated`, `Stats.Redundant`, `Replis`) : elle diverge sans
  qu'aucune ligne persistée ne change. Une étape « lignes persistées » dirait D23 directement.
- **D-VAV2-7** — 24 pertes E partaient en base d'un début S0 lui-même précédé d'un bit à 1 (loi de la
  tête contredite) : ces 24 « sains » de la base étaient déjà des fermetures factices.
- **D-VAV2-8** — Cascade de liaison (`e5adf7b2`, slot 1096) : six sains de la base reposaient sur un
  NEW lu à un début contredit (R1 D-R1-5 confirmé).


## 13. Pièces

Scratchpad `scratchpad/cg3-V2/` (non versionné) : `carte_tete/`, `gate2.tsv`, `gate2_perdus.txt`,
`perdus.txt`, `sonde_pertes/` et `sonde_pertes2/` (`va_v2_pertes.tsv`), `classes_par_film.tsv`,
`table_films.md`, `ks_tete/`, `re_tete_tsv/`, `re_etapes_divergentes.tsv`, `replis_tete.tsv`,
`replis_changes.tsv`, `etapes_base/`, `etapes_tete/`, `carte_loi/`, `carte_E/`, `mut/`,
`mutations.txt`, `v3.jq`, `trajets.jq`, `trajets_*.tsv`, `oracle_corpus.js`, `postures_corpus.tsv`, `base_docs/`, `gate2_loi.tsv`, `gate2_E.tsv`, `sonde_tete/`, `gfilm.log`, `lint.txt`, `lint_research.txt`, `gate.log`, `va_v2_corpus_gate.json`, et les journaux des gates. Scripts : `env.sh`, `construire.sh`, `carte.sh`, `ks.sh`,
`re.sh`, `gate.sh`, `replis.sh`, `sonde.sh`, `mkmut.sh`, `mutations.sh`, `decomp.sh`, `etapes.sh`.
Base : `scratchpad/cg3-V1/` (`carte_base/`, `ks_base/`, `re_base_tsv/`, `replis_base2.tsv`,
`sonde2/va_v1_paquets.tsv`, binaires `bin/base/`). Versionnés : `va_ghidra/ASM_142f2c3b0.txt`,
`ASM_14299d2c8.txt`, `FUN_1406d49c4.c`.
